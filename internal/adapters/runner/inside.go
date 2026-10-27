package runner

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// applyStopDelay bounds how long Apply may take to stop after the runner was
// asked to stop: Apply closes its Sessions and containers and writes its
// evidence within its own cleanup and persistence bounds (at most 60 s each).
const applyStopDelay = 3 * time.Minute

// maxApplyReport bounds the Apply report the runner keeps and returns.
const maxApplyReport = 64 << 20

// stateAdministration is Apply's cooperating writer lock inside a repository.
// It is operational state, never part of a candidate or the hand-back.
const stateAdministration = ".markitect/state/"

// manifestList collects repeated --dependency values.
type manifestList []string

func (m *manifestList) String() string     { return strings.Join(*m, ",") }
func (m *manifestList) Set(v string) error { *m = append(*m, v); return nil }

// Main is the runner image's entrypoint:
//
//	markitect-runner --base COMMIT [--model PATH] [--dependency PATH]...
//
// It returns the process exit status: 0 when Apply passed and its result was
// handed back, 1 when Apply failed, 2 for invalid usage and 3 otherwise.
func Main(ctx context.Context, args []string, stderr io.Writer) int {
	return entry(ctx, args, stderr, ContainerLayout())
}

func entry(ctx context.Context, args []string, stderr io.Writer, layout Layout) int {
	flags := flag.NewFlagSet("markitect-runner", flag.ContinueOnError)
	flags.SetOutput(stderr)
	base := flags.String("base", "", "full commit ID of the checkout to clone")
	model := flags.String("model", "", "repository-relative model.yaml")
	var dependencies manifestList
	flags.Var(&dependencies, "dependency", "repository-relative dependency model.yaml; repeat")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: markitect-runner --base COMMIT [--model PATH] [--dependency PATH]...")
		return 2
	}
	result, code := Inside(ctx, InsideRequest{Layout: layout, Base: *base, Model: *model, Dependencies: dependencies, Stderr: stderr})
	switch {
	case result.Error != "":
		fmt.Fprintf(stderr, "markitect-runner: Apply %s; no result handed back: %s\n", result.ApplyStatus, result.Error)
	case result.Commit != "":
		fmt.Fprintf(stderr, "markitect-runner: Apply passed; commit %s handed back\n", result.Commit)
	default:
		fmt.Fprintf(stderr, "markitect-runner: Apply %s; nothing to hand back\n", result.ApplyStatus)
	}
	return code
}

// InsideRequest is the entrypoint's work inside the runner image.
type InsideRequest struct {
	Layout Layout
	// Base is the checkout commit the launcher observed; the clone must hold it.
	Base string
	// Model and Dependencies are repository-relative manifest paths.
	Model        string
	Dependencies []string
	// Stderr receives Apply's own diagnostics.
	Stderr io.Writer
}

// Inside clones the read-only checkout at Base onto WorkBranch, runs Apply
// against the clone and writes the hand-back: ReportFile, ResultFile and, for
// a verified change, BundleFile. Only when Apply passed with a read-back
// publication does it commit exactly the published paths on top of Base. It
// returns the result it wrote and the process exit status.
func Inside(ctx context.Context, r InsideRequest) (Result, int) {
	result := Result{FormatVersion: resultFormat, ApplyStatus: StatusNotCompleted, ApplyExit: -1, Base: r.Base}
	if r.Stderr == nil {
		r.Stderr = io.Discard
	}
	if err := inside(ctx, r, &result); err != nil {
		result.Error = err.Error()
		result.Commit, result.Ref, result.Paths = "", "", nil
	}
	if err := writeResult(r.Layout.Output, result); err != nil {
		fmt.Fprintln(r.Stderr, "markitect-runner: write result:", err)
		return result, 3
	}
	switch {
	case result.Error != "":
		return result, 3
	case result.ApplyStatus == StatusPassed:
		return result, 0
	case result.ApplyStatus == StatusFailed:
		return result, 1
	}
	return result, 3
}

func inside(ctx context.Context, r InsideRequest, result *Result) error {
	if !validObjectID(r.Base) {
		return fmt.Errorf("base %q is not a full commit ID", r.Base)
	}
	if r.Model != "" && !RepositoryRelative(r.Model, false) {
		return fmt.Errorf("model %q is not repository-relative", r.Model)
	}
	for _, dependency := range r.Dependencies {
		if !RepositoryRelative(dependency, false) {
			return fmt.Errorf("dependency %q is not repository-relative", dependency)
		}
	}
	entries, err := os.ReadDir(r.Layout.Output)
	if err != nil {
		return fmt.Errorf("hand-back area: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("hand-back area is not empty")
	}
	if err := cloneCheckout(ctx, r.Layout, r.Base); err != nil {
		return err
	}
	report, exit, runErr := runApply(ctx, r)
	result.ApplyExit = exit
	if err := os.WriteFile(filepath.Join(r.Layout.Output, ReportFile), report, 0o644); err != nil {
		return errors.Join(runErr, fmt.Errorf("keep Apply report: %w", err))
	}
	if runErr != nil {
		return runErr
	}
	applied, err := readApplyReport(report)
	if err != nil {
		return err
	}
	result.ApplyStatus, result.ApplyID = applied.Status, applied.ID
	if applied.Status != StatusPassed {
		return nil
	}
	return handBack(ctx, r.Layout, r.Base, applied, result)
}

// cloneCheckout clones the read-only checkout through Git's own transport,
// never with hardlinks, and checks out Base on WorkBranch without a remote.
func cloneCheckout(ctx context.Context, layout Layout, base string) error {
	if _, err := os.Lstat(layout.Clone); !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("clone path already exists or cannot be observed")
	}
	if err := os.MkdirAll(filepath.Dir(layout.Clone), 0o700); err != nil {
		return err
	}
	source := gitCall{safe: []string{layout.Checkout, filepath.Join(layout.Checkout, ".git")}}
	if _, err := source.run(ctx, "clone", "--quiet", "--no-checkout", "--no-hardlinks", "--", fileURL(layout.Checkout), layout.Clone); err != nil {
		return fmt.Errorf("clone checkout: %w", err)
	}
	clone := gitCall{dir: layout.Clone}
	if _, err := clone.run(ctx, "checkout", "--quiet", "--no-track", "-b", WorkBranch, base); err != nil {
		return fmt.Errorf("checkout base %s (did the checkout move?): %w", base, err)
	}
	if _, err := clone.run(ctx, "remote", "remove", "origin"); err != nil {
		return err
	}
	head, err := clone.text(ctx, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || head != base {
		return fmt.Errorf("clone HEAD is not base %s", base)
	}
	return nil
}

func fileURL(p string) string {
	slashed := filepath.ToSlash(p)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return "file://" + slashed
}

// runApply runs markitect apply against the clone. A stop request reaches
// Apply as SIGTERM so that it can close its containers and keep its evidence.
func runApply(ctx context.Context, r InsideRequest) ([]byte, int, error) {
	args := []string{"apply", "--repo", r.Layout.Clone,
		"--runtime-profile", filepath.Join(r.Layout.Invocation, RuntimeProfileFile),
		"--run-config", filepath.Join(r.Layout.Invocation, RunConfigFile)}
	if r.Model != "" {
		args = append(args, "--model", r.Model)
	}
	for _, dependency := range r.Dependencies {
		args = append(args, "--dependency", dependency)
	}
	cmd := exec.CommandContext(ctx, r.Layout.Markitect, args...)
	out := &boundedBuffer{limit: maxApplyReport}
	cmd.Stdout, cmd.Stderr = out, r.Stderr
	cmd.Cancel = func() error {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	cmd.WaitDelay = applyStopDelay
	err := cmd.Run()
	if out.exceeded {
		return nil, -1, fmt.Errorf("Apply report exceeds %d bytes", maxApplyReport)
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out.Bytes(), exit.ExitCode(), nil
	}
	if err != nil {
		return out.Bytes(), -1, fmt.Errorf("run Apply: %w", err)
	}
	return out.Bytes(), 0, nil
}

// applyReport is the part of Apply's JSON report the hand-back depends on.
type applyReport struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Apply  *struct {
		Publication *struct {
			NoChange       bool     `json:"noChange"`
			CompletedPaths []string `json:"completedPaths"`
			Readback       bool     `json:"readback"`
		} `json:"publication"`
	} `json:"apply"`
}

func readApplyReport(data []byte) (applyReport, error) {
	var report applyReport
	if err := json.Unmarshal(data, &report); err != nil {
		return report, fmt.Errorf("Apply report is not readable JSON")
	}
	switch report.Status {
	case StatusPassed, StatusFailed, StatusNotCompleted:
	default:
		return report, fmt.Errorf("Apply report has an unknown status")
	}
	if report.ID != "" && !applyIDPattern.MatchString(report.ID) {
		return report, fmt.Errorf("Apply report has an invalid ID")
	}
	return report, nil
}

// handBack commits exactly the published paths on top of base and bundles the
// commit. A no-change publication hands back no bundle.
func handBack(ctx context.Context, layout Layout, base string, applied applyReport, result *Result) error {
	if applied.ID == "" || applied.Apply == nil || applied.Apply.Publication == nil || !applied.Apply.Publication.Readback {
		return fmt.Errorf("passed Apply report has no read-back publication")
	}
	publication := applied.Apply.Publication
	paths := publication.CompletedPaths
	seen := map[string]bool{}
	for _, p := range paths {
		if !RepositoryRelative(p, false) || seen[p] || strings.HasPrefix(p+"/", stateAdministration) {
			return fmt.Errorf("Apply report names an invalid or repeated published path")
		}
		seen[p] = true
	}
	clone := gitCall{dir: layout.Clone}
	if publication.NoChange {
		if len(paths) != 0 {
			return fmt.Errorf("no-change publication names completed paths")
		}
		if stray, err := strayChanges(ctx, clone, seen); err != nil || len(stray) != 0 {
			return errors.Join(err, fmt.Errorf("clone changed although Apply published no change"))
		}
		result.NoChange = true
		return nil
	}
	if len(paths) == 0 {
		return fmt.Errorf("changed publication names no completed paths")
	}
	if _, err := (gitCall{dir: layout.Clone, stdin: []byte(strings.Join(paths, "\x00") + "\x00")}).run(ctx, "add", "--all", "--force", "--pathspec-from-file=-", "--pathspec-file-nul"); err != nil {
		return err
	}
	staged, err := clone.run(ctx, "diff", "--cached", "--name-only", "-z", "--no-renames", "--no-ext-diff")
	if err != nil {
		return err
	}
	if !samePaths(nulPaths(staged), paths) {
		return fmt.Errorf("staged changes differ from the published paths")
	}
	stray, err := strayChanges(ctx, clone, seen)
	if err != nil {
		return err
	}
	if len(stray) != 0 {
		return fmt.Errorf("clone holds %d unpublished change(s), for example %q", len(stray), stray[0])
	}
	message := "Markitect Apply " + applied.ID + "\n\nVerified result published by Apply " + applied.ID + " in the Linux runner on base " + base + ".\n"
	if _, err := clone.run(ctx, "-c", "user.name=Markitect Runner", "-c", "user.email=runner@markitect.invalid", "commit", "--quiet", "--no-verify", "-m", message); err != nil {
		return err
	}
	commit, err := clone.text(ctx, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return err
	}
	if err := verifyResultCommit(ctx, clone, commit, base, paths); err != nil {
		return err
	}
	bundle := filepath.Join(layout.Output, BundleFile)
	if _, err := clone.run(ctx, "bundle", "create", "--quiet", bundle, ResultRef, "^"+base); err != nil {
		return err
	}
	if _, err := clone.run(ctx, "bundle", "verify", "--quiet", bundle); err != nil {
		return err
	}
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	result.Commit, result.Ref, result.Paths = commit, ResultRef, sorted
	return nil
}

// strayChanges lists working-tree changes outside the published paths and
// Apply's state administration.
func strayChanges(ctx context.Context, clone gitCall, published map[string]bool) ([]string, error) {
	out, err := clone.run(ctx, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames")
	if err != nil {
		return nil, err
	}
	var stray []string
	for _, entry := range nulPaths(out) {
		if len(entry) < 4 {
			return nil, fmt.Errorf("unexpected Git status entry")
		}
		p := entry[3:]
		if !published[p] && !strings.HasPrefix(p, stateAdministration) {
			stray = append(stray, p)
		}
	}
	return stray, nil
}

// verifyResultCommit requires a single-parent commit on base whose changes are
// exactly the published paths. The launcher repeats it after the import.
func verifyResultCommit(ctx context.Context, repository gitCall, commit, base string, paths []string) error {
	parents, err := repository.text(ctx, "rev-list", "--parents", "-n", "1", commit)
	if err != nil {
		return err
	}
	if fields := strings.Fields(parents); len(fields) != 2 || fields[0] != commit || fields[1] != base {
		return fmt.Errorf("result commit %s is not a single-parent commit on base %s", commit, base)
	}
	changed, err := repository.run(ctx, "diff", "--name-only", "-z", "--no-renames", "--no-ext-diff", base, commit)
	if err != nil {
		return err
	}
	if !samePaths(nulPaths(changed), paths) {
		return fmt.Errorf("result commit %s changes other paths than the published ones", commit)
	}
	return nil
}
