package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Subdirectories and files of one launch's host directory.
const (
	invocationSubdir = "invocation"
	resultSubdir     = "result"
	logFile          = "runner.log"
)

// importTimeout bounds the local fetch of the bundle and the branch creation.
const importTimeout = 2 * time.Minute

// LaunchRequest is one Apply started in the runner image from the owner's host.
type LaunchRequest struct {
	// Engine is the absolute path of the host's engine CLI, the run
	// configuration's engineExecutable.
	Engine string
	// Image is the runner image's engine image ID (sha256:...).
	Image string
	// Repository is the owner's checkout.
	Repository string
	// Model and Dependencies are repository-relative manifest paths.
	Model        string
	Dependencies []string
	// RunConfig and RuntimeProfile are the translated documents the runner reads.
	RunConfig, RuntimeProfile []byte
	// Credentials are the host login files, in the order of CredentialPath(1..n).
	Credentials []string
	// OutputRoot receives one new directory per launch.
	OutputRoot string
	// StateVolume is the named engine volume that keeps the runner's Apply state.
	StateVolume string
	// ResultBranch names the branch for a verified change; empty selects
	// markitect/apply-<Apply ID>.
	ResultBranch string
	// StopTimeout bounds the runner's graceful stop after cancellation.
	StopTimeout time.Duration
	// Log receives the runner's diagnostics; they are also kept in the launch directory.
	Log io.Writer
	// GOOS spells host paths for the engine CLI; empty means runtime.GOOS.
	GOOS string
}

// Launch is the outcome of one launch on the host.
type Launch struct {
	// ID names the launch, its container and its host directory.
	ID string
	// Directory is the launch's host directory: invocation, result and log.
	Directory string
	// Base is the checkout commit the runner cloned.
	Base string
	// Result is the runner's hand-back record, when it was written.
	Result Result
	// Report is the runner's Apply report, when it was written.
	Report json.RawMessage
	// Branch and Commit name the imported verified change.
	Branch, Commit string
}

// ReportPath is the host path of the runner's Apply report.
func (l Launch) ReportPath() string {
	if l.Directory == "" {
		return ""
	}
	return filepath.Join(l.Directory, resultSubdir, ReportFile)
}

// Start checks the checkout, starts the runner and waits for it, then imports
// a verified change as a new branch. The returned error names a failure of the
// launch, the runner or the import; an Apply that did not pass is reported in
// Result without an error. Start never changes the checkout's working tree,
// index or current branch; the import adds the change's objects and creates
// the new branch, nothing else.
func Start(ctx context.Context, r LaunchRequest) (out Launch, err error) {
	goos := r.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	log := r.Log
	if log == nil {
		log = io.Discard
	}
	if !filepath.IsAbs(r.Engine) || filepath.Clean(r.Engine) != r.Engine {
		return out, fmt.Errorf("runner engine must be an explicit absolute path")
	}
	if !imageIDPattern.MatchString(r.Image) {
		return out, fmt.Errorf("runner image %q is not an engine image ID (sha256:<64 hex>)", r.Image)
	}
	if r.StopTimeout <= 0 || len(r.RunConfig) == 0 || len(r.RuntimeProfile) == 0 || r.OutputRoot == "" {
		return out, fmt.Errorf("runner launch needs a stop timeout, invocation documents and an output root")
	}
	if out.Base, err = inspectCheckout(ctx, r.Repository); err != nil {
		return out, err
	}
	if r.ResultBranch != "" {
		if err := checkNewBranch(ctx, r.Repository, r.ResultBranch); err != nil {
			return out, err
		}
	}
	if err := checkLocalEngine(ctx, r.Engine); err != nil {
		return out, err
	}
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return out, err
	}
	out.ID = hex.EncodeToString(nonce[:])
	// Validate the complete invocation before anything is written.
	dir := filepath.Join(r.OutputRoot, out.ID)
	mounts, err := Mounts(goos, launchSpec(r, dir))
	if err != nil {
		return out, err
	}
	name := "markitect-runner-" + out.ID
	args, err := RunArgs(RunSpec{Name: name, Label: out.ID, Image: r.Image, Base: out.Base, Model: r.Model, Dependencies: r.Dependencies, Mounts: mounts})
	if err != nil {
		return out, err
	}
	if out.Directory, err = prepareLaunchDirectory(r.OutputRoot, out.ID, r.RunConfig, r.RuntimeProfile); err != nil {
		return out, err
	}
	runErr := runEngine(ctx, r.Engine, args, name, r.StopTimeout, out.Directory, log)
	result, resultErr := readResult(filepath.Join(out.Directory, resultSubdir, ResultFile))
	if resultErr != nil {
		return out, errors.Join(fmt.Errorf("runner hand-back unavailable: %w", resultErr), runErr)
	}
	out.Result = result
	if report, err := readBoundedFile(out.ReportPath(), maxApplyReport); err == nil && json.Valid(report) && len(report) > 0 {
		out.Report = json.RawMessage(report)
	}
	if result.Base != out.Base {
		return out, fmt.Errorf("runner cloned %s, not the observed base %s", result.Base, out.Base)
	}
	if result.Error != "" {
		return out, fmt.Errorf("runner: %s", result.Error)
	}
	if result.ApplyStatus != StatusPassed || result.NoChange {
		return out, nil
	}
	branch := r.ResultBranch
	if branch == "" {
		branch = "markitect/apply-" + result.ApplyID
	}
	// The verified change exists now; a late stop request must not lose it, so
	// the short local import gets its own bound.
	importCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), importTimeout)
	defer cancel()
	bundle := filepath.Join(out.Directory, resultSubdir, BundleFile)
	if err := importResult(importCtx, r.Repository, bundle, result, branch); err != nil {
		return out, fmt.Errorf("import the verified change from %s: %w", bundle, err)
	}
	out.Branch, out.Commit = branch, result.Commit
	return out, nil
}

// launchSpec names the host side of one launch's mounts. Only the checkout's
// .git directory is mounted, never its working tree: the runner clones
// committed objects, and untracked or ignored files such as .env stay on the host.
func launchSpec(r LaunchRequest, dir string) MountSpec {
	return MountSpec{GitDir: filepath.Join(r.Repository, ".git"), Invocation: filepath.Join(dir, invocationSubdir), Credentials: r.Credentials, Output: filepath.Join(dir, resultSubdir), StateVolume: r.StateVolume}
}

// checkLocalEngine requires the engine CLI to reach an engine on this machine.
// The runner's bind sources and its socket mount resolve on the engine's host,
// so a remote engine would see other paths, or none.
func checkLocalEngine(ctx context.Context, engine string) error {
	if os.Getenv("DOCKER_HOST") != "" {
		return fmt.Errorf("DOCKER_HOST is set; the runner needs the local engine: unset DOCKER_HOST and use the local engine's context")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, engine, "context", "inspect", "--format", "{{.Endpoints.docker.Host}}")
	out, stderr := &boundedBuffer{limit: 4 << 10}, &boundedBuffer{limit: 4 << 10}
	cmd.Stdout, cmd.Stderr = out, stderr
	if err := cmd.Run(); err != nil || out.exceeded {
		return fmt.Errorf("the engine's current context could not be read (%v: %s); the runner needs the local engine", err, strings.TrimSpace(stderr.String()))
	}
	endpoint := strings.TrimSpace(out.String())
	if !strings.HasPrefix(endpoint, "unix://") && !strings.HasPrefix(endpoint, "npipe://") {
		return fmt.Errorf("the engine's current context uses %q; the runner needs the local engine through a socket or named pipe: switch with \"docker context use default\" (or desktop-linux for Docker Desktop) and unset DOCKER_CONTEXT", endpoint)
	}
	return nil
}

// inspectCheckout returns the checkout's HEAD commit. The runner sees only the
// checkout's .git directory and clones committed history from it, so the
// launcher refuses uncommitted changes and untracked files that Git does not
// ignore instead of dropping them; Git-ignored files stay on the host. Any
// named branch, main included, is accepted: the runner writes only its clone
// and a new branch. The checkout must be a main working tree with its own
// objects: a linked worktree's .git file and object alternates name host paths
// that do not exist inside the runner.
func inspectCheckout(ctx context.Context, repository string) (string, error) {
	if !filepath.IsAbs(repository) || filepath.Clean(repository) != repository {
		return "", fmt.Errorf("checkout must be an explicit absolute path")
	}
	info, err := os.Lstat(filepath.Join(repository, ".git"))
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("the runner needs a checkout whose .git is a directory; linked worktrees and submodules are not supported")
	}
	// Borrowed objects (clone --shared or --reference) live at host paths the
	// runner cannot see.
	if _, err := os.Lstat(filepath.Join(repository, ".git", "objects", "info", "alternates")); err == nil {
		return "", fmt.Errorf("the runner needs a checkout with its own objects; this one borrows objects through objects/info/alternates")
	}
	g := gitCall{dir: repository}
	top, err := g.text(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	topInfo, topErr := os.Stat(top)
	repoInfo, repoErr := os.Stat(repository)
	if topErr != nil || repoErr != nil || !os.SameFile(topInfo, repoInfo) {
		return "", fmt.Errorf("checkout must be the repository root")
	}
	if _, err := g.text(ctx, "symbolic-ref", "--quiet", "HEAD"); err != nil {
		return "", fmt.Errorf("the runner needs the checkout on a named branch, not a detached HEAD")
	}
	base, err := g.text(ctx, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", err
	}
	if !validObjectID(base) {
		return "", fmt.Errorf("checkout HEAD %q is not a full commit ID", base)
	}
	status, err := g.run(ctx, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames", "--ignore-submodules=none")
	if err != nil {
		return "", err
	}
	if err := refuseUncommitted(nulPaths(status)); err != nil {
		return "", err
	}
	return base, nil
}

// refuseUncommitted fails closed on everything a clone would silently drop:
// uncommitted changes to tracked files and untracked files Git does not ignore.
func refuseUncommitted(entries []string) error {
	var changed, untracked int
	example := ""
	for _, entry := range entries {
		if len(entry) < 4 {
			return fmt.Errorf("unexpected Git status entry")
		}
		if entry[:2] == "??" {
			untracked++
		} else {
			changed++
		}
		if example == "" {
			example = entry[3:]
		}
	}
	if changed+untracked == 0 {
		return nil
	}
	return fmt.Errorf("the checkout has %d uncommitted change(s) and %d untracked file(s), for example %q: commit or remove them; runner mode applies committed history only", changed, untracked, example)
}

// checkNewBranch requires a valid branch name that does not exist yet.
func checkNewBranch(ctx context.Context, repository, branch string) error {
	g := gitCall{dir: repository}
	if strings.HasPrefix(branch, "-") {
		return fmt.Errorf("result branch %q is not a valid branch name", branch)
	}
	if normalized, err := g.text(ctx, "check-ref-format", "--branch", branch); err != nil || normalized != branch {
		return fmt.Errorf("result branch %q is not a valid branch name", branch)
	}
	if _, err := g.text(ctx, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		return fmt.Errorf("result branch %q already exists", branch)
	}
	return nil
}

func prepareLaunchDirectory(root, id string, runConfig, runtimeProfile []byte) (string, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	dir := filepath.Join(root, id)
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", err
	}
	for _, sub := range []string{invocationSubdir, resultSubdir} {
		if err := os.Mkdir(filepath.Join(dir, sub), 0o700); err != nil {
			return dir, err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, invocationSubdir, RunConfigFile), runConfig, 0o600); err != nil {
		return dir, err
	}
	return dir, os.WriteFile(filepath.Join(dir, invocationSubdir, RuntimeProfileFile), runtimeProfile, 0o600)
}

// runEngine runs the runner attached and keeps its diagnostics. Cancellation
// asks the engine to stop the runner, which stops Apply with SIGTERM so that
// Apply closes its containers and writes its evidence; the engine kills it
// after the stop timeout.
func runEngine(ctx context.Context, engine string, args []string, name string, stopTimeout time.Duration, dir string, log io.Writer) error {
	file, err := os.OpenFile(filepath.Join(dir, logFile), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	w := io.MultiWriter(log, file)
	cmd := exec.CommandContext(ctx, engine, args...)
	cmd.Stdout, cmd.Stderr = w, w
	cmd.Cancel = func() error {
		// Stop the container, not only its CLI: a container left running would
		// keep Apply going and hold the state lock. A stop that cannot start or
		// fails falls back to removing the container.
		remove := func() error { return exec.Command(engine, RemoveArgs(name)...).Run() }
		stop := exec.Command(engine, StopArgs(name, stopTimeout)...)
		if stop.Start() == nil {
			go func() {
				if stop.Wait() != nil {
					_ = remove()
				}
			}()
			return nil
		}
		removal := exec.Command(engine, RemoveArgs(name)...)
		if removal.Start() == nil {
			go removal.Wait()
			return nil
		}
		cmd.Process.Kill()
		return fmt.Errorf("could not ask the engine to stop runner container %s; remove it with \"docker rm --force %s\"", name, name)
	}
	cmd.WaitDelay = stopTimeout + time.Minute
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("runner container %s: %w", name, err)
	}
	return nil
}

// StopArgs returns the engine arguments that stop the runner gracefully.
func StopArgs(name string, timeout time.Duration) []string {
	return []string{"stop", "--time", strconv.Itoa(int(timeout.Seconds())), name}
}

// RemoveArgs returns the engine arguments that remove the runner at once.
func RemoveArgs(name string) []string { return []string{"rm", "--force", name} }

// importResult fetches the verified commit from the bundle and creates the
// branch only if it does not exist. The commit must be a single-parent commit
// on the observed base whose changes are exactly the published paths.
func importResult(ctx context.Context, repository, bundle string, result Result, branch string) error {
	if err := checkNewBranch(ctx, repository, branch); err != nil {
		return err
	}
	g := gitCall{dir: repository}
	if _, err := g.run(ctx, "bundle", "verify", "--quiet", bundle); err != nil {
		return fmt.Errorf("result bundle: %w", err)
	}
	heads, err := g.text(ctx, "bundle", "list-heads", bundle)
	if err != nil {
		return err
	}
	if heads != result.Commit+" "+ResultRef {
		return fmt.Errorf("result bundle does not hold exactly commit %s at %s", result.Commit, ResultRef)
	}
	// Objects only: no FETCH_HEAD, tags, submodules or automatic maintenance.
	// --no-auto-maintenance needs Git 2.29, like --no-write-fetch-head.
	if _, err := g.run(ctx, "-c", "transfer.fsckObjects=true", "fetch", "--quiet", "--no-tags", "--no-write-fetch-head", "--no-auto-maintenance", "--recurse-submodules=no", bundle, ResultRef); err != nil {
		return fmt.Errorf("fetch result bundle: %w", err)
	}
	if err := verifyResultCommit(ctx, g, result.Commit, result.Base, result.Paths); err != nil {
		return err
	}
	create := gitCall{dir: repository, stdin: []byte("create refs/heads/" + branch + " " + result.Commit + "\n")}
	if _, err := create.run(ctx, "update-ref", "--stdin"); err != nil {
		return fmt.Errorf("create result branch %q: %w", branch, err)
	}
	return nil
}
