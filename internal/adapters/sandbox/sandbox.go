// Package sandbox runs a Command's one container (concept 3.0 §9), natively on
// Windows, Linux and macOS with only the Docker CLI. Nothing of the user's is
// mounted: code enters and leaves as a Git bundle on a command's standard input
// and output. The implementer, each reviewer and the required checks are Linux
// users of their own, and the tools belong to none of them, so nothing one leaves
// reaches another. No process in the container holds a capability.
package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const (
	// RunLabel marks every container and volume of a Command with its Run.
	RunLabel = "markitect.run"
	// The users that are no role; see packaging/runtime.
	checksUser = "checks"
	toolsUser  = "tools"
	workRoot   = "/work"
	homeRoot   = "/home"
	gitPath    = "/usr/bin/git"
	shPath     = "/bin/sh"
	// repoDir is the bare repository that holds what entered the sandbox.
	repoDir = workRoot + "/" + toolsUser + "/repo.git"
	// workBranch is the branch a Collect hands out.
	workBranch = "work"
	// outputTail is how much of a check's output the Result sees.
	outputTail = 16 << 10
	// maxNote bounds a note a role left.
	maxNote = 16 << 20
	// settle bounds a command that ends what a user left.
	settle = 30 * time.Second
)

// uids are the image's users (packaging/runtime/Dockerfile).
var uids = func() map[string]string {
	m := map[string]string{agentport.ImplementerUser: "1001:1001", checksUser: "1003:1003", toolsUser: "1004:1004"}
	for n := 1; n <= agentport.ReviewerUsers; n++ {
		m[agentport.ReviewerUser(n)] = fmt.Sprintf("%d:%d", 1010+n, 1010+n)
	}
	return m
}()

// sessionUsers are the users whose processes hold code under review.
var sessionUsers = func() []string {
	users := []string{agentport.ImplementerUser}
	for n := 1; n <= agentport.ReviewerUsers; n++ {
		users = append(users, agentport.ReviewerUser(n))
	}
	return users
}()

// clearShared removes what the running user left in the two places every user can
// write, whatever it set their modes to.
const clearShared = `for d in /tmp /dev/shm; do find "$d" -mindepth 1 -user "$(id -u)" -type d -exec chmod u+rwx {} \; 2>/dev/null || true; find "$d" -mindepth 1 -user "$(id -u)" -delete 2>/dev/null || true; done; `

// neutralAttributes switches off every attribute that would let a file's
// checked-out bytes differ from its blob: Git converts nothing on the way in or
// out (concept 3.0 §9).
const neutralAttributes = "* -text -eol -crlf -ident -filter -working-tree-encoding -export-ignore -export-subst !diff"

// gitEnv keeps the user's and the image's Git configuration out of the tools'
// Git work.
var gitEnv = []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0",
	"GIT_AUTHOR_NAME=Markitect", "GIT_AUTHOR_EMAIL=markitect@markitect.invalid", "GIT_COMMITTER_NAME=Markitect", "GIT_COMMITTER_EMAIL=markitect@markitect.invalid"}

// Limits bound the Command's container, its required checks included.
type Limits struct {
	CPUs   string // docker --cpus, for example "4"
	Memory string // docker --memory, for example "8g"
	Pids   int
	Tmp    string // the size of /tmp, for example "1g"
}

// Sandboxes starts Command containers from one image on one network.
type Sandboxes struct {
	Engine *Engine
	Image  string
	// Network reaches the provider. Hosts and DNS are passed to the engine as
	// they are.
	Network    string
	Hosts, DNS []string
	Limits     Limits
}

var _ run.Sandboxes = (*Sandboxes)(nil)

var runID = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,62}$`)

// Start creates the container and its two volumes, and puts the commit in. On
// failure it removes everything it created.
func (s *Sandboxes) Start(ctx context.Context, name, commit string, bundle io.Reader) (_ run.Sandbox, err error) {
	if !runID.MatchString(name) {
		return nil, fmt.Errorf("run %q: a container name needs lower-case letters, digits, '.', '_' or '-'", name)
	}
	if left, err := s.Engine.labelled(ctx, name); err != nil {
		return nil, err
	} else if len(left) > 0 {
		return nil, fmt.Errorf("run %q already has containers or volumes %v; remove them first", name, left)
	}
	b := &Box{engine: s.Engine, run: name, container: "markitect-run-" + name}
	defer func() {
		if err != nil {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
			defer cancel()
			err = errors.Join(err, Sweep(cleanup, s.Engine, name))
		}
	}()
	label := RunLabel + "=" + name
	for _, v := range []string{b.container + "-work", b.container + "-home"} {
		if err := s.Engine.run(ctx, nil, nil, nil, "volume", "create", "--label", label, v); err != nil {
			return nil, err
		}
	}
	// The container itself belongs to the tools' user. Fresh named volumes take
	// over the image's folders with their owners, so each user can write its
	// own and no other.
	args := []string{"run", "--detach", "--init", "--name", b.container, "--label", label, "--read-only", "--cap-drop", "ALL",
		"--security-opt", "no-new-privileges", "--user", uids[toolsUser], "--workdir", workRoot,
		"--mount", "type=volume,src=" + b.container + "-work,dst=" + workRoot, "--mount", "type=volume,src=" + b.container + "-home,dst=" + homeRoot,
		"--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=" + orDefault(s.Limits.Tmp, "1g")}
	if s.Limits.CPUs != "" {
		args = append(args, "--cpus", s.Limits.CPUs)
	}
	if s.Limits.Memory != "" {
		args = append(args, "--memory", s.Limits.Memory)
	}
	if s.Limits.Pids > 0 {
		args = append(args, "--pids-limit", strconv.Itoa(s.Limits.Pids))
	}
	if s.Network != "" {
		args = append(args, "--network", s.Network)
	}
	for _, h := range s.Hosts {
		args = append(args, "--add-host", h)
	}
	for _, d := range s.DNS {
		args = append(args, "--dns", d)
	}
	args = append(args, "--entrypoint", "/bin/sleep", s.Image, "infinity")
	if err := s.Engine.run(ctx, nil, nil, nil, args...); err != nil {
		return nil, err
	}
	var env bytes.Buffer
	if err := s.Engine.run(ctx, nil, &env, nil, "exec", "--user", uids[toolsUser], b.container, "/usr/bin/env"); err != nil {
		return nil, fmt.Errorf("reading the image's environment: %w", err)
	}
	b.image = strings.Split(strings.TrimSpace(env.String()), "\n")
	// Everything Git makes in the repository is the tools' alone.
	if err := b.tools(ctx, nil, nil, "umask 077; "+gitPath+" init -q --bare "+repoDir+" && "+gitPath+" --git-dir="+repoDir+` config core.autocrlf false && printf '%s\n' "`+neutralAttributes+`" > `+repoDir+"/info/attributes"); err != nil {
		return nil, fmt.Errorf("the sandbox's repository: %w", err)
	}
	if err := b.bring(ctx, bundle); err != nil {
		return nil, fmt.Errorf("the starting commit: %w", err)
	}
	return b, nil
}

// Box is one Command's started container.
type Box struct {
	engine    *Engine
	run       string
	container string
	image     []string

	mu     sync.Mutex
	copies int
	// implementer is the implementer's folder.
	implementer string
}

var _ run.Sandbox = (*Box)(nil)

// Exec runs one command as a user of the container. When ctx ends, every process
// of that user ends with it.
func (b *Box) Exec(ctx context.Context, e agentport.Exec) error {
	uid, ok := uids[e.User]
	if !ok {
		return fmt.Errorf("the sandbox has no user %q", e.User)
	}
	if len(e.Argv) == 0 || !strings.HasPrefix(e.Argv[0], "/") {
		return fmt.Errorf("exec needs an absolute command, not %q", e.Argv)
	}
	args := []string{"exec", "--user", uid}
	if e.Stdin != nil {
		args = append(args, "--interactive")
	}
	if e.Dir != "" {
		args = append(args, "--workdir", e.Dir)
	}
	for _, kv := range append(b.userEnv(e.User), e.Env...) {
		args = append(args, "--env", kv)
	}
	// By name only: a secret stays off the command line, where a process listing
	// could show it. The engine's client reads it from its own environment.
	for _, kv := range e.Secrets {
		name, _, _ := strings.Cut(kv, "=")
		args = append(args, "--env", name)
	}
	args = append(args, b.container)
	args = append(args, e.Argv...)
	err := b.engine.command(ctx, e.Stdin, e.Stdout, e.Stderr, e.Secrets, args...)
	if ctx.Err() != nil && e.User != toolsUser {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), settle)
		defer cancel()
		return errors.Join(ctx.Err(), b.endUser(cleanup, e.User))
	}
	return err
}

// userEnv is the image's environment for one user: a HOME and TMPDIR of its own,
// and whatever else pointed into the image's home pointing into its own.
func (b *Box) userEnv(user string) []string {
	home := path.Join(homeRoot, user)
	old := ""
	for _, kv := range b.image {
		if v, ok := strings.CutPrefix(kv, "HOME="); ok && v != "/" {
			old = strings.TrimSuffix(v, "/")
		}
	}
	env := []string{"HOME=" + home, "TMPDIR=" + path.Join(home, ".tmp")}
	for _, kv := range b.image {
		k, v, _ := strings.Cut(kv, "=")
		if k == "HOME" || k == "TMPDIR" || k == "PATH" || k == "PWD" || k == "HOSTNAME" || k == "" {
			continue
		}
		if old != "" && (v == old || strings.HasPrefix(v, old+"/")) {
			env = append(env, k+"="+home+v[len(old):])
		}
	}
	return env
}

// tools runs a script as the tools' user, with Git's configuration kept out.
func (b *Box) tools(ctx context.Context, stdin io.Reader, stdout io.Writer, script string, args ...string) error {
	return b.script(ctx, toolsUser, stdin, stdout, script, args...)
}

// script runs a fixed shell script as a user; what varies goes in as arguments.
func (b *Box) script(ctx context.Context, user string, stdin io.Reader, stdout io.Writer, script string, args ...string) error {
	var stderr bytes.Buffer
	argv := append([]string{shPath, "-eu", "-c", script, "sandbox"}, args...)
	err := b.Exec(ctx, agentport.Exec{User: user, Dir: workRoot, Argv: argv, Env: gitEnv, Stdin: stdin, Stdout: stdout, Stderr: &stderr})
	if err != nil && stderr.Len() > 0 {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return err
}

// bring puts what a bundle holds into the sandbox's repository.
func (b *Box) bring(ctx context.Context, bundle io.Reader) error {
	return b.tools(ctx, bundle, nil, `tmp=$(mktemp -p "$HOME/.tmp"); cat > "$tmp"; `+gitPath+" --git-dir="+repoDir+` fetch -q "$tmp" '+refs/*:refs/in/*'; rm -f "$tmp"`)
}

// copyFor makes a new clean folder for a user, with the commit checked out and
// its own Git history, from nothing that user or any other left.
func (b *Box) copyFor(ctx context.Context, user, commit string) (string, error) {
	b.mu.Lock()
	b.copies++
	folder := path.Join(workRoot, user, strconv.Itoa(b.copies))
	b.mu.Unlock()
	repo := path.Join(folder, "repo")
	if err := b.tools(ctx, nil, nil, gitPath+" --git-dir="+repoDir+` update-ref "refs/heads/copy-$1" "$1"`, commit); err != nil {
		return "", fmt.Errorf("commit %s: %w", commit, err)
	}
	read, write := io.Pipe()
	done := make(chan error, 1)
	go func() {
		err := b.tools(ctx, nil, write, gitPath+" --git-dir="+repoDir+` bundle create -q - "refs/heads/copy-$1"`, commit)
		write.CloseWithError(err)
		done <- err
	}()
	// Written by the user itself, so it owns its copy; the commit's own
	// attributes convert nothing.
	script := `mkdir -p "$1"; tmp=$(mktemp -p "$HOME/.tmp"); cat > "$tmp"; ` + gitPath + ` init -q "$1"; cd "$1"; ` +
		gitPath + ` config core.autocrlf false; printf '%s\n' "` + neutralAttributes + `" > .git/info/attributes; ` +
		gitPath + ` fetch -q "$tmp" "refs/heads/copy-$2:refs/heads/start"; ` + gitPath + ` checkout -q --detach start; rm -f "$tmp"`
	err := b.script(ctx, user, read, nil, script, repo, commit)
	read.CloseWithError(err)
	return repo, errors.Join(err, <-done)
}

// Deliver makes a user fresh and gives it a new clean copy of a commit.
func (b *Box) Deliver(ctx context.Context, user, commit string) (string, error) {
	if err := b.clearUser(ctx, user); err != nil {
		return "", err
	}
	repo, err := b.copyFor(ctx, user, commit)
	if err == nil && user == agentport.ImplementerUser {
		b.mu.Lock()
		b.implementer = repo
		b.mu.Unlock()
	}
	return repo, err
}

// clearUser ends a user's processes and removes all it left: its folder, its home
// with whatever configuration a session wrote there, and its files in the shared
// places. Run as that user, so it holds whatever modes the user set.
func (b *Box) clearUser(ctx context.Context, user string) error {
	if err := b.endUser(ctx, user); err != nil {
		return err
	}
	return b.script(ctx, user, nil, nil, `for d in "$1" "$2"; do chmod -R u+rwX "$d" 2>/dev/null || true; find "$d" -mindepth 1 -delete 2>/dev/null || true; done; mkdir -p "$2/.tmp" "$2/notes"`,
		path.Join(workRoot, user), path.Join(homeRoot, user))
}

// Collect ends the implementer's processes and commits its folder: the folder
// comes out as a tar stream written by its own user, and the tools' user commits
// it in a place of its own with its own Git data, so that nothing the session
// left runs and whatever Git does not take is not there.
func (b *Box) Collect(ctx context.Context, parent, message string) (string, io.ReadCloser, error) {
	b.mu.Lock()
	repo := b.implementer
	b.mu.Unlock()
	if repo == "" {
		return "", nil, errors.New("the implementer has no folder")
	}
	user := agentport.ImplementerUser
	if err := b.endUser(ctx, user); err != nil {
		return "", nil, err
	}
	read, write := io.Pipe()
	done := make(chan error, 1)
	go func() {
		// A file its session made unreadable is still its own to read.
		err := b.script(ctx, user, nil, write, `cd "$1"; chmod -R u+rwX . 2>/dev/null || true; exec tar --exclude=.git -cf - .`, repo)
		write.CloseWithError(err)
		done <- err
	}()
	var out bytes.Buffer
	script := `scratch=$(mktemp -d -p "$HOME/.tmp"); tar --no-same-owner --no-same-permissions -xf - -C "$scratch"; export GIT_DIR=` + repoDir + ` GIT_WORK_TREE="$scratch" GIT_INDEX_FILE="$scratch.index"; cd "$scratch"; ` +
		gitPath + ` read-tree "$1"; ` + gitPath + ` add -A; tree=$(` + gitPath + ` write-tree); cd /; rm -rf "$scratch" "$scratch.index"; ` +
		`if [ "$tree" = "$(` + gitPath + ` rev-parse "$1^{tree}")" ]; then echo "$1"; else printf '%s\n' "$2" | ` + gitPath + ` commit-tree "$tree" -p "$1"; fi`
	err := b.tools(ctx, read, &out, script, parent, message)
	read.CloseWithError(err)
	if err := errors.Join(err, <-done); err != nil {
		return "", nil, err
	}
	commit := strings.TrimSpace(out.String())
	if commit == parent {
		return parent, nil, nil
	}
	if err := b.tools(ctx, nil, nil, gitPath+" --git-dir="+repoDir+` update-ref "refs/heads/`+workBranch+`" "$1"`, commit); err != nil {
		return "", nil, err
	}
	// On disk and not in memory: work can be as large as its build outputs.
	spool, err := os.CreateTemp("", "markitect-work-*.bundle")
	if err != nil {
		return "", nil, err
	}
	bundle := &spooled{spool}
	if err := b.tools(ctx, nil, spool, gitPath+" --git-dir="+repoDir+` bundle create -q - "refs/heads/`+workBranch+`" "^$1"`, parent); err != nil {
		bundle.Close()
		return "", nil, err
	}
	if _, err := spool.Seek(0, io.SeekStart); err != nil {
		bundle.Close()
		return "", nil, err
	}
	return commit, bundle, nil
}

// spooled is a file that goes when it is closed.
type spooled struct{ *os.File }

func (s *spooled) Close() error {
	return errors.Join(s.File.Close(), os.Remove(s.File.Name()))
}

// Check runs the required checks on a commit (concept 3.0 §6, step 6): they run
// code the agent wrote, so they stay inside the container, in a clean folder of
// their own, as a third user that is given no token, while no session runs. The
// Command ends their processes afterwards, and the folder and home are thrown
// away. A check whose command cannot start is marked NotRun.
func (b *Box) Check(ctx context.Context, bundle io.Reader, commit string, checks []project.Check) (results []run.CheckResult, err error) {
	// No session runs while they do, and nothing a session left behind either.
	for _, user := range sessionUsers {
		if err := b.endUser(ctx, user); err != nil {
			return nil, err
		}
	}
	if bundle != nil {
		if err := b.bring(ctx, bundle); err != nil {
			return nil, err
		}
	}
	repo, err := b.copyFor(ctx, checksUser, commit)
	if err != nil {
		return nil, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), settle)
		defer cancel()
		if cerr := b.clearChecks(cleanup); cerr != nil && err == nil {
			results, err = nil, cerr
		}
	}()
	for _, c := range checks {
		res := b.runCheck(ctx, c, repo)
		results = append(results, res)
		if !res.Passed {
			break
		}
	}
	if ctx.Err() != nil {
		return results, ctx.Err()
	}
	return results, nil
}

func (b *Box) runCheck(ctx context.Context, c project.Check, repo string) run.CheckResult {
	r := run.CheckResult{Name: c.Name, Exit: -1}
	dir := path.Join(repo, c.Directory)
	// A state may not hold the check's directory yet: that is the work's state.
	if err := b.script(ctx, checksUser, nil, nil, `test -d "$1"`, dir); err != nil {
		r.NotRun = !b.running(ctx)
		r.Output = fmt.Sprintf("[the check's directory %s does not exist in this state]\n", c.Directory)
		return r
	}
	checkCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	out := &tailBuffer{limit: outputTail}
	start := time.Now()
	err := b.Exec(checkCtx, agentport.Exec{User: checksUser, Dir: dir, Argv: []string{shPath, "-c", c.Command}, Stdout: out, Stderr: out})
	r.Seconds = time.Since(start).Round(10 * time.Millisecond).Seconds()
	r.Output = out.String()
	r.TimedOut = errors.Is(checkCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil
	var exit *agentport.ExitError
	switch {
	case err == nil:
		r.Exit, r.Passed = 0, true
	case !b.running(ctx):
		// The client's own exit code says nothing once the container is gone.
		r.NotRun = true
		r.Output += "\n[the container is gone]\n"
	case errors.As(err, &exit):
		r.Exit = exit.Code
		// 126 and 127 are the shell's: a program that cannot run or is not
		// there, which is the environment's failure, unless the check names a
		// program in the repository, which is the work's.
		r.NotRun = (exit.Code == 126 || exit.Code == 127) && !inRepository(c.Command)
	case r.TimedOut:
		r.Output += "\n[the check timed out]\n"
	default:
		r.NotRun = true
		r.Output += "\n[the check did not start: " + err.Error() + "]\n"
	}
	return r
}

// inRepository says whether a check's command starts with a program given by a
// relative path.
func inRepository(command string) bool {
	fields := strings.Fields(command)
	return len(fields) > 0 && strings.Contains(fields[0], "/") && !strings.HasPrefix(fields[0], "/")
}

// running says whether the Command's container still runs.
func (b *Box) running(ctx context.Context) bool {
	probe, cancel := context.WithTimeout(context.WithoutCancel(ctx), settle)
	defer cancel()
	out, err := b.engine.output(probe, "inspect", "--format", "{{.State.Running}}", b.container)
	return err == nil && out == "true"
}

// clearChecks ends what the checks left and throws their folder and home away.
func (b *Box) clearChecks(ctx context.Context) error { return b.clearUser(ctx, checksUser) }

// endUser ends every process of a user and removes what it left in the shared
// places. Each user can only signal and remove its own.
func (b *Box) endUser(ctx context.Context, user string) error {
	err := b.engine.run(ctx, nil, nil, nil, "exec", "--user", uids[user], b.container, shPath, "-c", "kill -9 -1 2>/dev/null; "+clearShared+"exit 0")
	if err != nil {
		return fmt.Errorf("ending the processes of %s: %w", user, err)
	}
	return nil
}

// NotePath is where a user writes a note of that name, outside its folder.
func (b *Box) NotePath(user, name string) string {
	return path.Join(homeRoot, user, "notes", name)
}

// Note ends the user's processes, so that nothing it left can change the note
// while it is read or after, and reads the note.
func (b *Box) Note(ctx context.Context, user, name string) ([]byte, bool, error) {
	if err := b.endUser(ctx, user); err != nil {
		return nil, false, err
	}
	var out bytes.Buffer
	err := b.script(ctx, user, nil, &limited{limit: maxNote, w: &out}, `if [ -f "$1" ] && [ ! -L "$1" ]; then cat "$1"; else exit 3; fi`, b.NotePath(user, name))
	var exit *agentport.ExitError
	if errors.As(err, &exit) && exit.Code == 3 {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return out.Bytes(), true, nil
}

// Close removes the container with its volumes.
func (b *Box) Close(ctx context.Context) error { return Sweep(ctx, b.engine, b.run) }

// Sweep removes every container and volume that carries the Run's label, also
// those of a Command that stopped without cleaning up, and verifies none is left.
func Sweep(ctx context.Context, e *Engine, name string) error {
	label := "label=" + RunLabel + "=" + name
	ids, err := e.output(ctx, "ps", "--all", "--quiet", "--filter", label)
	if err != nil {
		return err
	}
	var errs []error
	if f := strings.Fields(ids); len(f) > 0 {
		errs = append(errs, e.run(ctx, nil, nil, nil, append([]string{"rm", "--force", "--volumes"}, f...)...))
	}
	vols, err := e.output(ctx, "volume", "ls", "--quiet", "--filter", label)
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	if f := strings.Fields(vols); len(f) > 0 {
		errs = append(errs, e.run(ctx, nil, nil, nil, append([]string{"volume", "rm", "--force"}, f...)...))
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	left, err := e.labelled(ctx, name)
	if err != nil {
		return err
	}
	if len(left) > 0 {
		return fmt.Errorf("run %q still has containers or volumes %v", name, left)
	}
	return nil
}

// Engine runs the Docker CLI at an absolute path; the host never searches PATH.
type Engine struct{ path string }

// NewEngine binds the Docker CLI at path.
func NewEngine(cli string) (*Engine, error) {
	if !isAbs(cli) {
		return nil, fmt.Errorf("the container engine %q must be an absolute path", cli)
	}
	return &Engine{path: cli}, nil
}

func isAbs(p string) bool {
	return strings.HasPrefix(p, "/") || (len(p) > 2 && p[1] == ':')
}

const maxEngineError = 16 << 10

// run runs one docker command; stdout may be nil.
func (e *Engine) run(ctx context.Context, stdin io.Reader, stdout io.Writer, env []string, args ...string) error {
	return e.command(ctx, stdin, stdout, nil, env, args...)
}

// command runs one docker command with env added to its client's environment. A
// command that ran in a container and failed is an *agentport.ExitError; stdout
// and errors may be nil.
func (e *Engine) command(ctx context.Context, stdin io.Reader, stdout, stderrTo io.Writer, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, e.path, args...)
	cmd.Stdin = stdin
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	if stdout == nil {
		stdout = &out
	}
	stderr := &tailBuffer{limit: maxEngineError}
	cmd.Stdout, cmd.Stderr = stdout, io.Writer(stderr)
	if stderrTo != nil {
		cmd.Stderr = io.MultiWriter(stderr, stderrTo)
	}
	cmd.WaitDelay = 5 * time.Second
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		return ctx.Err()
	case errors.As(err, &exit) && args[0] == "exec" && exit.ExitCode() > 0 && exit.ExitCode() != 125:
		// 125 is the engine's own failure; 126 and 127 the command's.
		return fmt.Errorf("%w: %s", &agentport.ExitError{Code: exit.ExitCode()}, strings.TrimSpace(stderr.String()))
	default:
		return fmt.Errorf("docker %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
}

func (e *Engine) output(ctx context.Context, args ...string) (string, error) {
	var out bytes.Buffer
	err := e.run(ctx, nil, &out, nil, args...)
	return strings.TrimSpace(out.String()), err
}

func (e *Engine) labelled(ctx context.Context, name string) ([]string, error) {
	label := "label=" + RunLabel + "=" + name
	ids, err := e.output(ctx, "ps", "--all", "--quiet", "--filter", label)
	if err != nil {
		return nil, err
	}
	vols, err := e.output(ctx, "volume", "ls", "--quiet", "--filter", label)
	if err != nil {
		return nil, err
	}
	return append(strings.Fields(ids), strings.Fields(vols)...), nil
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// tailBuffer keeps the last limit bytes written to it.
type tailBuffer struct {
	limit int
	mu    sync.Mutex
	buf   []byte
	cut   bool
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.limit {
		t.buf = append([]byte(nil), t.buf[len(t.buf)-t.limit:]...)
		t.cut = true
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cut {
		return "[…]" + string(t.buf)
	}
	return string(t.buf)
}

// limited passes on at most limit bytes and drops the rest.
type limited struct {
	limit int
	w     io.Writer
}

func (l *limited) Write(p []byte) (int, error) {
	if l.limit > 0 {
		n := min(len(p), l.limit)
		if _, err := l.w.Write(p[:n]); err != nil {
			return 0, err
		}
		l.limit -= n
	}
	return len(p), nil
}
