package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// The Sessions are controlled; capture, snapshots, conflicting merges, repaired
// baselines, candidate retention and result readback use the actual adapters.
func TestGitConflictReturnsToOwningChildBeforeIntegration(t *testing.T) {
	_, p, g, ref := runFixture(t)
	store, err := runstore.New(g.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	sessions := &conflictManagers{}
	executor := run.Executor{Workspaces: g, Sessions: sessions, Records: store}
	result, err := executor.Execute(context.Background(), run.Request{MaxManagers: 5, ID: "conflict-repair", Project: "fixture", Prepared: p, Input: ref, Mandate: run.Implement, Allowance: run.NewAllowance(time.Now().Add(90*time.Second), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil || result.Technical != run.Completed || result.Decision != run.Submitted || result.Candidate == nil {
		t.Fatal(result, err)
	}
	if sessions.conflicts != 1 || sessions.repairs != 1 || sessions.refusals != 1 {
		t.Fatalf("conflicts=%d repairs=%d", sessions.conflicts, sessions.repairs)
	}
	for _, h := range result.Handoffs {
		if h.Manager == "a" && (len(h.Children) != 1 || h.Children[0].Child.ID == sessions.rejectedCommit) {
			t.Fatal("conflicted handoff accepted", h)
		}
		if h.Manager == "a/deep" && (h.Review == nil || h.Review.Decision != run.Approve || h.Result.ID == sessions.rejectedCommit) {
			t.Fatal("repair did not receive exact renewed review", h)
		}
	}
	destination := t.TempDir()
	if err = g.Reopen(context.Background(), *result.Candidate, destination); err != nil {
		t.Fatal(err)
	}
	if _, err = localCommand(context.Background(), destination, nil, "merge-base", "--is-ancestor", sessions.rejectedCommit, result.Candidate.Commit); err != nil {
		t.Fatal("rejected child history lost", err)
	}
	for _, h := range result.Handoffs {
		if h.Manager == "a/deep" {
			if _, err = localCommand(context.Background(), destination, nil, "merge-base", "--is-ancestor", h.Base.ID, h.Result.ID); err != nil {
				t.Fatal("actual parent baseline ancestry lost", err)
			}
		}
	}
	data, err := os.ReadFile(filepath.Join(destination, "out/deep.txt"))
	if err != nil || string(data) != "child repaired parent conflict\n" {
		t.Fatal(string(data), err)
	}
}

type conflictManagers struct {
	mu                           sync.Mutex
	conflicts, repairs, refusals int
	rejectedCommit               string
}

func (m *conflictManagers) Open(_ context.Context, s run.SessionSpec) (run.Session, error) {
	return &conflictManager{owner: m, spec: s}, nil
}

type conflictManager struct {
	owner        *conflictManagers
	spec         run.SessionSpec
	calls, tasks int
}

func (*conflictManager) Close(context.Context) error { return nil }
func (m *conflictManager) Exchange(ctx context.Context, message run.Message) error {
	m.calls++
	tools := m.spec.Tools
	var task *run.Assignment
	for _, e := range message.Events {
		if e.Assignment != nil {
			task = e.Assignment
			m.tasks++
		}
	}
	if task == nil {
		return fmt.Errorf("expected an assignment, got %+v", message)
	}
	if m.tasks == 1 {
		for _, child := range m.spec.Manager.Children {
			if err := tools.Delegate(ctx, child, "Implement your responsibility.", nil); err != nil {
				return err
			}
		}
	}
	remaining := 0
	if m.tasks == 1 {
		remaining = len(m.spec.Manager.Children)
	}
	for remaining > 0 {
		event, err := tools.Await(ctx)
		if err != nil {
			return err
		}
		if event.Handoff == nil {
			return fmt.Errorf("expected fixed child handoff")
		}
		h := event.Handoff
		m.owner.mu.Lock()
		inject := h.Manager == "a/deep" && m.owner.conflicts == 0
		m.owner.mu.Unlock()
		if inject {
			path := filepath.Join(m.spec.Workspace.Location().Directory, "out/deep.txt")
			if err = os.WriteFile(path, []byte("parent conflict\n"), 0600); err != nil {
				return err
			}
			if err = tools.Review(ctx, h.Manager, h.Result.ID, run.Approve, nil); err == nil || !errors.Is(err, run.ErrMergeConflict) {
				return fmt.Errorf("actual conflict missing: %v", err)
			}
			// The caller learns that the failed approval already recorded Reject
			// and that only its correction with Findings starts the repair.
			for _, detail := range []string{"recorded as reject", "delegate a correction with Findings", "child's copy"} {
				if !strings.Contains(err.Error(), detail) {
					return fmt.Errorf("conflict feedback hides recorded review or next action (%s): %w", detail, err)
				}
			}
			if repeated := tools.Review(ctx, h.Manager, h.Result.ID, run.Reject, []run.Finding{{Text: "Request conflict correction."}}); repeated == nil {
				return fmt.Errorf("already reviewed fixed child accepted another Review")
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil || string(data) != "parent conflict\n" {
				return fmt.Errorf("failed merge changed parent: %q %v", data, readErr)
			}
			m.owner.mu.Lock()
			m.owner.conflicts++
			m.owner.rejectedCommit = h.Result.ID
			m.owner.mu.Unlock()
			if err = tools.Delegate(ctx, h.Manager, "Repair the child change on the current parent baseline.", []run.Finding{{Text: "Git conflict in out/deep.txt: preserve the parent baseline and return a concrete child repair."}}); err != nil {
				return err
			}
			continue
		}
		if err = tools.Review(ctx, h.Manager, h.Result.ID, run.Approve, nil); err != nil {
			return err
		}
		remaining--
	}
	if m.spec.Manager.Package == "a/deep" {
		value := "child original\n"
		if m.tasks > 1 {
			if len(task.Findings) != 2 || !strings.Contains(task.Findings[0].Text, "Git conflict") || !strings.Contains(task.Findings[1].Text, "preserved Git merge conflict") {
				return fmt.Errorf("owning child lost concrete conflict feedback: %+v", task.Findings)
			}
			data, err := os.ReadFile(filepath.Join(m.spec.Workspace.Location().Directory, "out/deep.txt"))
			if err != nil || (!strings.Contains(string(data), "parent conflict\n") || !strings.Contains(string(data), "child original\n") || !strings.Contains(string(data), "<<<<<<<")) {
				return fmt.Errorf("repair lacks actual parent baseline: %q %v", data, err)
			}
			if _, err = m.spec.Workspace.Snapshot(ctx); err == nil || !strings.Contains(err.Error(), "unresolved Git index") {
				return fmt.Errorf("unresolved merge snapshot was not refused: %v", err)
			}
			// Deliberately judge while unresolved: Run refuses the judgment with
			// the concrete reason, without accepting or terminating technically.
			if err = tools.Judge(ctx, run.Submitted, nil); err == nil || !strings.Contains(err.Error(), "unresolved Git index") {
				return fmt.Errorf("unresolved judgment was not refused: %v", err)
			}
			m.owner.mu.Lock()
			m.owner.refusals++
			m.owner.mu.Unlock()
			value = "child repaired parent conflict\n"
			m.owner.mu.Lock()
			m.owner.repairs++
			m.owner.mu.Unlock()
		}
		if err := os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, "out/deep.txt"), []byte(value), 0600); err != nil {
			return err
		}
		if m.tasks > 1 {
			// This is the owning child's ordinary workspace action, after resolving
			// the file. The executor never stages an unresolved conflict for it.
			if _, err := localCommand(ctx, m.spec.Workspace.Location().Directory, nil, "add", "--", "out/deep.txt"); err != nil {
				return err
			}
		}
	}
	return tools.Judge(ctx, run.Submitted, nil)
}

func TestResolvedMergeKeepsBothParentsWhenTreeMatchesChildHead(t *testing.T) {
	r := testkit.NewRepo(t)
	r.Write("same.txt", "base\n")
	r.Commit("base")
	r.Git("checkout", "-b", "child")
	r.Write("same.txt", "child\n")
	r.Commit("child")
	child := r.Git("rev-parse", "HEAD")
	r.Git("checkout", "-b", "parent", "HEAD~1")
	r.Write("same.txt", "parent\n")
	r.Commit("parent")
	parent := r.Git("rev-parse", "HEAD")
	r.Git("checkout", "child")
	ctx := context.Background()
	if _, err := localCommand(ctx, r.Dir, nil, "merge", "--no-edit", parent); err == nil {
		t.Fatal("expected actual conflict")
	}
	if _, err := snapshotGit(ctx, r.Dir, localCommand, nil, localExecutables(r.Dir)); err == nil {
		t.Fatal("unresolved index accepted")
	}
	r.Write("same.txt", "child\n")
	r.Git("add", "--", "same.txt")
	fixed, err := snapshotGit(ctx, r.Dir, localCommand, nil, localExecutables(r.Dir))
	if err != nil {
		t.Fatal(err)
	}
	if fixed.ID == child {
		t.Fatal("equal tree skipped required merge ancestry")
	}
	for _, ancestor := range []string{child, parent} {
		if _, err = localCommand(ctx, r.Dir, nil, "merge-base", "--is-ancestor", ancestor, fixed.ID); err != nil {
			t.Fatal("merge ancestry lost", ancestor, err)
		}
	}
	pending, err := localCommand(ctx, r.Dir, nil, "rev-parse", "--revs-only", "--end-of-options", "MERGE_HEAD")
	if err != nil || pending != "" {
		t.Fatal("merge not closed", pending, err)
	}
}

func TestIntegrationTechnicalFailureNeverBecomesRepairableConflict(t *testing.T) {
	for _, stage := range []string{"fetch", "abort", "readback"} {
		t.Run(stage, func(t *testing.T) {
			r := testkit.NewRepo(t)
			r.Write("same.txt", "base\n")
			r.Commit("base")
			r.Git("checkout", "-b", "child")
			r.Write("same.txt", "child\n")
			r.Commit("child")
			ctx := context.Background()
			child, err := object(ctx, r.Dir, "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			r.Git("checkout", "-b", "parent", "HEAD~1")
			r.Write("same.txt", "parent\n")
			r.Commit("parent")
			injected := errors.New("injected technical " + stage)
			command := func(ctx context.Context, dir string, input []byte, args ...string) (string, error) {
				if stage == "fetch" && args[0] == "fetch" || stage == "abort" && args[0] == "merge" && len(args) > 1 && args[1] == "--abort" || stage == "readback" && args[0] == "diff" && len(args) > 1 && args[1] == "--quiet" {
					return "", injected
				}
				return localCommand(ctx, dir, input, args...)
			}
			_, err = integrateGit(ctx, r.Dir, r.Dir, child, command, func(context.Context) ([]string, error) { return nil, nil }, localExecutables(r.Dir))
			if !errors.Is(err, injected) || errors.Is(err, run.ErrMergeConflict) {
				t.Fatal("technical failure became repairable", err)
			}
		})
	}
}

// A parent's own build output stays untracked across a refused integration:
// the conflict remains repairable and restoration keeps exactly that output.
func TestIntegrationConflictKeepsParentBuildOutput(t *testing.T) {
	r := testkit.NewRepo(t)
	r.Write("same.txt", "base\n")
	r.Commit("base")
	r.Git("checkout", "-b", "child")
	r.Write("same.txt", "child\n")
	r.Commit("child")
	ctx := context.Background()
	child, err := object(ctx, r.Dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	r.Git("checkout", "-b", "parent", "HEAD~1")
	r.Write("same.txt", "parent\n")
	r.Commit("parent")
	r.Write("parent-tool", "\x7fELF\x02\x01\x01\x00parent build output")
	_, err = integrateGit(ctx, r.Dir, r.Dir, child, localCommand, func(context.Context) ([]string, error) { return nil, nil }, localExecutables(r.Dir))
	if !errors.Is(err, run.ErrMergeConflict) {
		t.Fatal("parent build output turned a repairable conflict into a technical failure", err)
	}
	if status := r.Git("status", "--porcelain", "--untracked-files=all"); status != "?? parent-tool" {
		t.Fatalf("restored parent lost or captured its build output: %q", status)
	}
}

type technicalWorkspaces struct {
	run.Workspaces
	failure error
}

func (w technicalWorkspaces) Open(ctx context.Context, r run.Request) (run.WorkspaceService, error) {
	s, err := w.Workspaces.Open(ctx, r)
	return technicalService{WorkspaceService: s, failure: w.failure}, err
}

type technicalService struct {
	run.WorkspaceService
	failure error
}

func (s technicalService) Copy(ctx context.Context, name string, base run.Commit) (run.Copy, error) {
	c, err := s.WorkspaceService.Copy(ctx, name, base)
	return technicalCopy{Copy: c, name: name, failure: s.failure}, err
}
func unwrapTechnical(c run.Copy) run.Copy {
	if wrapped, ok := c.(technicalCopy); ok {
		return wrapped.Copy
	}
	return c
}
func (s technicalService) Retain(ctx context.Context, c run.Copy, fixed run.Commit) (project.CandidateRef, error) {
	return s.WorkspaceService.Retain(ctx, unwrapTechnical(c), fixed)
}

type technicalCopy struct {
	run.Copy
	name    string
	failure error
}

func (c technicalCopy) Update(ctx context.Context, source run.Copy, base run.Commit) error {
	return c.Copy.Update(ctx, unwrapTechnical(source), base)
}
func (c technicalCopy) Review(ctx context.Context, source run.Copy, base, result run.Commit) (string, error) {
	return c.Copy.Review(ctx, unwrapTechnical(source), base, result)
}
func (c technicalCopy) Integrate(ctx context.Context, source run.Copy, result run.Commit) (run.Commit, error) {
	if c.name == "." {
		return run.Commit{}, c.failure
	}
	return c.Copy.Integrate(ctx, unwrapTechnical(source), result)
}

type ignoringManagers struct {
	mu      sync.Mutex
	ignored bool
}

func (m *ignoringManagers) Open(_ context.Context, s run.SessionSpec) (run.Session, error) {
	return &ignoringManager{owner: m, spec: s}, nil
}

type ignoringManager struct {
	owner *ignoringManagers
	spec  run.SessionSpec
}

func (*ignoringManager) Close(context.Context) error { return nil }
func (m *ignoringManager) Exchange(ctx context.Context, _ run.Message) error {
	tools := m.spec.Tools
	for _, child := range m.spec.Manager.Children {
		if err := tools.Delegate(ctx, child, "Complete modeled work.", nil); err != nil {
			return tools.Judge(ctx, run.Submitted, nil)
		}
	}
	for range m.spec.Manager.Children {
		event, err := tools.Await(ctx)
		if err != nil {
			return tools.Judge(ctx, run.Submitted, nil)
		}
		if event.Handoff == nil {
			return fmt.Errorf("missing handoff")
		}
		if err = tools.Review(ctx, event.Handoff.Manager, event.Handoff.Result.ID, run.Approve, nil); err != nil {
			m.owner.mu.Lock()
			m.owner.ignored = true
			m.owner.mu.Unlock()
		}
	}
	// Ignore every returned error, including the judgment's own refusal.
	_ = tools.Judge(ctx, run.Submitted, nil)
	return nil
}

func TestSessionCannotIgnoreTechnicalIntegrationFailure(t *testing.T) {
	_, p, g, ref := runFixture(t)
	store, err := runstore.New(g.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range []string{"fetch", "abort", "readback"} {
		t.Run(stage, func(t *testing.T) {
			injected := errors.New("injected technical " + stage)
			sessions := &ignoringManagers{}
			executor := run.Executor{Workspaces: technicalWorkspaces{Workspaces: g, failure: injected}, Sessions: sessions, Records: store}
			result, err := executor.Execute(context.Background(), run.Request{MaxManagers: 5, ID: "ignored-" + stage, Project: "fixture", Prepared: p, Input: ref, Mandate: run.Implement, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
			if !errors.Is(err, injected) || result.Technical == run.Completed || result.Candidate != nil || !sessions.ignored {
				t.Fatal("ignored tool failure became success", result, err)
			}
			persisted, readErr := store.Read(context.Background(), "fixture", "ignored-"+stage)
			if readErr != nil || persisted.Technical == run.Completed || persisted.Candidate != nil {
				t.Fatal("terminal record erased technical failure", persisted, readErr)
			}
		})
	}
}
