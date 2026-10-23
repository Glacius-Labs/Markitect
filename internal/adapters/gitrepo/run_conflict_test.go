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
	owner *conflictManagers
	spec  run.SessionSpec
	calls int
}

func (*conflictManager) Close(context.Context) error { return nil }
func (m *conflictManager) Exchange(ctx context.Context, message run.Message) (run.Response, error) {
	m.calls++
	tools := m.spec.Tools
	if m.spec.Manager.Package != "." && m.calls == 1 {
		event, err := tools.Await(ctx)
		if err != nil {
			return run.Response{}, err
		}
		if event.Assignment == nil {
			return run.Response{}, fmt.Errorf("expected assignment")
		}
		message.Feedback = event.Assignment.Feedback
	}
	if m.calls == 1 {
		for _, child := range m.spec.Manager.Children {
			if err := tools.Delegate(ctx, child, "Implement your responsibility.", nil); err != nil {
				return run.Response{}, err
			}
		}
	}
	remaining := 0
	if m.calls == 1 {
		remaining = len(m.spec.Manager.Children)
	}
	for remaining > 0 {
		event, err := tools.Await(ctx)
		if err != nil {
			return run.Response{}, err
		}
		if event.Handoff == nil {
			return run.Response{}, fmt.Errorf("expected fixed child handoff")
		}
		h := event.Handoff
		m.owner.mu.Lock()
		inject := h.Manager == "a/deep" && m.owner.conflicts == 0
		m.owner.mu.Unlock()
		if inject {
			path := filepath.Join(m.spec.Workspace.Location().Directory, "out/deep.txt")
			if err = os.WriteFile(path, []byte("parent conflict\n"), 0600); err != nil {
				return run.Response{}, err
			}
			if err = tools.Review(ctx, h.Manager, h.Result.ID, run.Approve, nil); err == nil || !errors.Is(err, run.ErrMergeConflict) {
				return run.Response{}, fmt.Errorf("actual conflict missing: %v", err)
			}
			// The native caller must learn that the failed approval already
			// recorded Reject: no repair or replacement handoff starts until
			// this parent delegates correction with concrete Findings.
			for _, detail := range []string{"already recorded as rejected", "no child correction was started", "delegate", "concrete Findings"} {
				if !strings.Contains(err.Error(), detail) {
					return run.Response{}, fmt.Errorf("conflict feedback hides recorded review or next action (%s): %w", detail, err)
				}
			}
			if repeated := tools.Review(ctx, h.Manager, h.Result.ID, run.Reject, []run.Finding{{Text: "Request conflict correction."}}); repeated == nil {
				return run.Response{}, fmt.Errorf("already reviewed fixed child accepted another Review")
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil || string(data) != "parent conflict\n" {
				return run.Response{}, fmt.Errorf("failed merge changed parent: %q %v", data, readErr)
			}
			m.owner.mu.Lock()
			m.owner.conflicts++
			m.owner.rejectedCommit = h.Result.ID
			m.owner.mu.Unlock()
			if err = tools.Delegate(ctx, h.Manager, "Repair the child change on the current parent baseline.", []run.Finding{{Text: "Git conflict in out/deep.txt: preserve the parent baseline and return a concrete child repair."}}); err != nil {
				return run.Response{}, err
			}
			continue
		}
		if err = tools.Review(ctx, h.Manager, h.Result.ID, run.Approve, nil); err != nil {
			return run.Response{}, err
		}
		remaining--
	}
	if m.spec.Manager.Package == "a/deep" {
		value := "child original\n"
		if m.calls > 1 {
			if m.calls == 2 && (len(message.Feedback) != 2 || !strings.Contains(message.Feedback[0].Text, "Git conflict")) || m.calls == 3 && (len(message.Feedback) != 1 || !strings.Contains(message.Feedback[0].Text, "unresolved Git index")) {
				return run.Response{}, fmt.Errorf("owning child lost concrete conflict feedback")
			}
			data, err := os.ReadFile(filepath.Join(m.spec.Workspace.Location().Directory, "out/deep.txt"))
			if err != nil || (!strings.Contains(string(data), "parent conflict\n") || !strings.Contains(string(data), "child original\n") || !strings.Contains(string(data), "<<<<<<<")) {
				return run.Response{}, fmt.Errorf("repair lacks actual parent baseline: %q %v", data, err)
			}
			if _, err = m.spec.Workspace.Snapshot(ctx); err == nil || !strings.Contains(err.Error(), "unresolved Git index") {
				return run.Response{}, fmt.Errorf("unresolved merge snapshot was not refused: %v", err)
			}
			if m.calls == 2 {
				// Deliberately submit while unresolved: Run must return ordinary feedback
				// to this same Manager, without accepting or terminating technically.
				m.owner.mu.Lock()
				m.owner.refusals++
				m.owner.mu.Unlock()
				return run.Response{Decision: run.Submitted}, nil
			}
			value = "child repaired parent conflict\n"
			m.owner.mu.Lock()
			m.owner.repairs++
			m.owner.mu.Unlock()
		}
		if err := os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, "out/deep.txt"), []byte(value), 0600); err != nil {
			return run.Response{}, err
		}
		if m.calls > 1 {
			// This is the owning child's ordinary workspace action, after resolving
			// the file. The executor never stages an unresolved conflict for it.
			if _, err := localCommand(ctx, m.spec.Workspace.Location().Directory, nil, "add", "--", "out/deep.txt"); err != nil {
				return run.Response{}, err
			}
		}
	}
	return run.Response{Decision: run.Submitted}, nil
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
	if _, err := snapshotGit(ctx, r.Dir, localCommand, nil); err == nil {
		t.Fatal("unresolved index accepted")
	}
	r.Write("same.txt", "child\n")
	r.Git("add", "--", "same.txt")
	fixed, err := snapshotGit(ctx, r.Dir, localCommand, nil)
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
			_, err = integrateGit(ctx, r.Dir, r.Dir, child, command, func(context.Context) ([]string, error) { return nil, nil })
			if !errors.Is(err, injected) || errors.Is(err, run.ErrMergeConflict) {
				t.Fatal("technical failure became repairable", err)
			}
		})
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
func (m *ignoringManager) Exchange(ctx context.Context, _ run.Message) (run.Response, error) {
	tools := m.spec.Tools
	if m.spec.Manager.Package != "." {
		if _, err := tools.Await(ctx); err != nil {
			return run.Response{Decision: run.Submitted}, nil
		}
	}
	for _, child := range m.spec.Manager.Children {
		if err := tools.Delegate(ctx, child, "Complete modeled work.", nil); err != nil {
			return run.Response{Decision: run.Submitted}, nil
		}
	}
	for range m.spec.Manager.Children {
		event, err := tools.Await(ctx)
		if err != nil {
			return run.Response{Decision: run.Submitted}, nil
		}
		if event.Handoff == nil {
			return run.Response{}, fmt.Errorf("missing handoff")
		}
		if err = tools.Review(ctx, event.Handoff.Manager, event.Handoff.Result.ID, run.Approve, nil); err != nil {
			m.owner.mu.Lock()
			m.owner.ignored = true
			m.owner.mu.Unlock()
		}
	}
	return run.Response{Decision: run.Submitted}, nil
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
