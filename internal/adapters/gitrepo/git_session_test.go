package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Controlled exec peer for existing real-Git integration tests. Production
// never falls back to RunHelper; only this synthetic peer delegates to their
// existing local-Git/error-injection peers after undoing service path mapping.
type testAssignmentExec struct {
	work  *Workspaces
	execs atomic.Int64
}

func (*testAssignmentExec) ID() string                  { return strings.Repeat("c", 64) }
func (s *testAssignmentExec) GitExecCount() int         { return int(s.execs.Load()) }
func (*testAssignmentExec) Close(context.Context) error { return nil }
func (s *testAssignmentExec) Exec(ctx context.Context, argv []string, stdin io.Reader, stdout io.Writer) error {
	primary := argv[len(argv)-1]
	if index := slices.Index(argv, "-C"); index >= 0 && index+1 < len(argv) {
		primary = argv[index+1]
	}
	var volumes []string
	for volume, target := range s.work.gitPaths {
		if primary == target || strings.HasPrefix(primary, target+"/") {
			volumes = append(volumes, volume)
			break
		}
	}
	if len(volumes) != 1 {
		return fmt.Errorf("test exec has no primary workspace for %v", argv)
	}
	for _, arg := range argv {
		for volume, target := range s.work.gitPaths {
			if (arg == target || strings.HasPrefix(arg, target+"/")) && !slices.Contains(volumes, volume) {
				volumes = append(volumes, volume)
			}
		}
	}
	if len(volumes) > 2 {
		return fmt.Errorf("test exec references too many workspace roots")
	}
	translated := slices.Clone(argv)
	spec := s.work.helper
	spec.Mounts = nil
	for i, volume := range volumes {
		target := []string{"/work", "/source"}[i]
		spec.Mounts = append(spec.Mounts, container.Mount{Volume: volume, Target: target})
		for n, arg := range argv {
			if n > 0 && (argv[n-1] == "-m" || argv[n-1] == "--message") {
				continue
			}
			original := s.work.gitPaths[volume]
			if arg == original || strings.HasPrefix(arg, original+"/") {
				translated[n] = target + strings.TrimPrefix(arg, original)
			}
		}
	}
	s.execs.Add(1)
	return s.work.runtime.RunHelper(ctx, spec, translated, stdin, stdout)
}

func attachTestGit(w *Workspaces) {
	w.git = &testAssignmentExec{work: w}
	w.gitPaths = make(map[string]string)
	for id := range w.owned {
		registerTestGitVolume(w, w.volume(id))
	}
}

func registerTestGitVolume(w *Workspaces, volume string) {
	w.gitPaths[volume] = "/assignments/" + volume
}

type observedAssignmentExec struct {
	container.ExecSession
	execs, closes int
	closed        bool
	closeErr      error
}

func (s *observedAssignmentExec) Exec(ctx context.Context, argv []string, input io.Reader, output io.Writer) error {
	if s.closed {
		return fmt.Errorf("test service closed")
	}
	s.execs++
	return s.ExecSession.Exec(ctx, argv, input, output)
}

func (s *observedAssignmentExec) Close(context.Context) error {
	if !s.closed {
		s.closes++
		s.closed = true
	}
	return s.closeErr
}

func TestApplyGitReservesAllMountsAndRetainsOneContainer(t *testing.T) {
	w, runtime := identityFixture(t)
	if runtime.starts != 1 || len(runtime.startSpec.Mounts) != 6 {
		t.Fatalf("did not reserve all mounts once: starts=%d spec=%+v", runtime.starts, runtime.startSpec)
	}
	for _, mount := range runtime.startSpec.Mounts {
		if !mount.NoCopy || mount.ReadOnly || !strings.HasPrefix(mount.Target, "/assignments/") {
			t.Fatalf("wrong service mount: %+v", mount)
		}
	}
	ctx := context.Background()
	root, err := w.PrepareAssignment(ctx, run.WorkspaceSpec{ID: "root"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := w.ForkAssignment(ctx, root, "manager")
	if err != nil {
		t.Fatal(err)
	}
	commit, err := child.Commit(ctx, "/work/message must stay literal")
	if err != nil {
		t.Fatal(err)
	}
	if got := ordinaryAssignmentGit(t, runtime.git.roots[child.Volume()], "log", "-1", "--format=%s", commit); got != "/work/message must stay literal" {
		t.Fatalf("path mapping rewrote commit message: %q", got)
	}
	if _, err := w.ReadAssignmentCommit(ctx, run.AssignmentCommitRequest{Source: child, Commit: commit, MaxEntries: 100, MaxBytes: 4096}); err != nil {
		t.Fatal(err)
	}
	if runtime.starts != 1 || runtime.session.execs < 10 || runtime.session.closes != 0 {
		t.Fatalf("Git did not retain the service: %+v", runtime.session)
	}
	disposable, err := w.Materialize(ctx, run.WorkspaceSpec{ID: "check-123"})
	if err != nil {
		t.Fatal(err)
	}
	unreserved := &assignmentWorkspace{owner: w, id: disposable.ID, volume: disposable.Volume}
	if _, err := unreserved.git(ctx, "init"); err == nil {
		t.Fatal("ran Git in an undeclared disposable copy")
	}
	if closed, err := w.Close(ctx, disposable.ID); err != nil || !closed.Closed {
		t.Fatalf("disposable check copy did not close during Git session: %+v %v", closed, err)
	}
	if _, err := w.Materialize(ctx, run.WorkspaceSpec{ID: "root"}); err == nil {
		t.Fatal("reused a claimed workspace")
	}
	if _, err := w.Close(ctx, "root"); err == nil {
		t.Fatal("removed workspace before Git container closed")
	}
	runtime.workspaceFake.mounts = []string{strings.Repeat("d", 64)}
	if err := w.unmounted(ctx, root.Volume()); err == nil {
		t.Fatal("foreign mount was treated as the Git service")
	}
	runtime.workspaceFake.mounts = nil
	for i := 0; i < 2; i++ {
		if err := w.CloseGit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if runtime.session.closes != 1 || runtime.removals != 1 || !runtime.volumes[root.Volume()].Exists || !runtime.volumes[child.Volume()].Exists {
		t.Fatal("close did not retain evidence volumes or closed twice")
	}
	if w.GitContainerID() != runtime.session.ID() || w.GitExecCount() == 0 {
		t.Fatal("close lost retained container ID or exec count")
	}
	if _, err := child.Commit(ctx, "late"); err == nil {
		t.Fatal("Git command after service close")
	}
}

func TestApplyGitStartFailureRetainsVolumesAndNeverReturnsAReadyCopy(t *testing.T) {
	w, runtime := identityRuntimeFixture(t)
	runtime.failStart = true
	if err := w.StartGit(context.Background(), []run.WorkspaceID{"root", "review"}); err == nil {
		t.Fatal("accepted failed service start")
	}
	if len(runtime.volumes) != 2 || runtime.removals != 0 || runtime.starts != 1 || len(runtime.helpers) != 0 || w.GitContainerID() != "" || w.GitExecCount() != 0 {
		t.Fatal("failed start lost volumes, started preparation helpers or claimed a container")
	}
	if _, err := w.PrepareAssignment(context.Background(), run.WorkspaceSpec{ID: "root"}); err == nil {
		t.Fatal("prepared assignment after service start failed")
	}
	if err := w.StartGit(context.Background(), []run.WorkspaceID{"root", "review"}); err == nil || runtime.starts != 1 {
		t.Fatal("retried failed service start")
	}
}

func TestApplyGitRejectsInvalidReservationsBeforeAnyResource(t *testing.T) {
	for _, ids := range [][]run.WorkspaceID{nil, {"one", "one"}, {"one", "../foreign"}} {
		w, runtime := identityRuntimeFixture(t)
		if err := w.StartGit(context.Background(), ids); err == nil || len(runtime.volumes) != 0 || runtime.starts != 0 {
			t.Fatalf("invalid reservation touched runtime: %v", ids)
		}
	}
}

func TestApplyGitCloseFailureRetainsEvidenceAndPreventsVolumeRemoval(t *testing.T) {
	w, runtime := identityFixture(t)
	failure := errors.New("controlled uncertain service cleanup")
	runtime.session.closeErr = failure
	for i := 0; i < 2; i++ {
		if err := w.CloseGit(context.Background()); !errors.Is(err, failure) {
			t.Fatal("cleanup failure lost")
		}
	}
	if _, err := w.Close(context.Background(), "root"); err == nil || runtime.removals != 0 || w.GitContainerID() != runtime.session.ID() {
		t.Fatal("uncertain cleanup permitted evidence deletion or lost container identity")
	}
}
