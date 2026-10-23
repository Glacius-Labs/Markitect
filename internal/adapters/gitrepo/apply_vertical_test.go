package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

type applyExecutor struct {
	t                      *testing.T
	git                    *LocalGit
	store                  *runstore.Store
	requests               []run.Request
	managers               []*controlledManagers
	noChange, rejectVerify bool
}

func (e *applyExecutor) Execute(ctx context.Context, request run.Request) (run.Result, error) {
	e.t.Helper()
	for _, previous := range e.managers {
		for _, session := range previous.sessions {
			if !session.closed {
				e.t.Fatal("new Run opened before old Session closure")
			}
			if _, err := os.Stat(session.spec.Workspace.Location().Directory); !os.IsNotExist(err) {
				e.t.Fatal("old live copy survived")
			}
		}
	}
	if len(e.requests) > 0 && e.requests[0].Allowance != request.Allowance {
		e.t.Fatal("correction reset allowance")
	}
	managers := &controlledManagers{crossed: make(chan struct{})}
	if request.Mandate == run.Verify && e.rejectVerify {
		managers.forceDecision = run.Reject
	}
	if request.Mandate == run.Verify {
		e.rejectVerify = false
	}
	e.requests = append(e.requests, request)
	e.managers = append(e.managers, managers)
	return (&run.Executor{Sessions: managers, Workspaces: e.git, Records: e.store}).Execute(ctx, request)
}

func TestApplyCompleteControlledHierarchyPublishesOnlyStoredClosedVerify(t *testing.T) {
	r, prepared, git, _ := runFixture(t)
	r.Git("switch", "-c", "codex/controlled-apply")
	head := r.Git("rev-parse", "HEAD")
	store, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	executor := &applyExecutor{t: t, git: git, store: store}
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	request := run.Request{ID: "apply", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(2*time.Minute), 200), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
	report, err := command.Execute(context.Background(), request)
	if err != nil || report.Status != model.Passed {
		t.Fatalf("status=%s error=%v final=%s/%s", report.Status, err, report.Workflow.Final.Technical, report.Workflow.Final.Decision)
	}
	if report.Publication == nil || !report.Publication.Readback || report.Publication.NoChange || len(report.Publication.CompletedPaths) != 2 {
		t.Fatalf("publication=%+v", report.Publication)
	}
	var mandates []run.Mandate
	for _, request := range executor.requests {
		mandates = append(mandates, request.Mandate)
	}
	if !reflect.DeepEqual(mandates, []run.Mandate{run.Assessment, run.Implement, run.Verify}) {
		t.Fatal(mandates)
	}
	if r.Git("rev-parse", "HEAD") != head {
		t.Fatal("Apply committed owner checkout")
	}
	for path, expected := range map[string]string{"out/a.txt": "parent result\n", "out/deep.txt": "repaired after integration\n", "out/accounted.ignored": "original ignored input\n"} {
		data, err := os.ReadFile(filepath.Join(r.Dir, filepath.FromSlash(path)))
		if err != nil || string(data) != expected {
			t.Fatal(path, string(data), err)
		}
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "review-scratch.txt")); !os.IsNotExist(err) {
		t.Fatal("review scratch published")
	}
	final := report.Workflow.Final
	fresh, _ := runstore.New(git.StateRoot)
	stored, err := fresh.Read(context.Background(), "fixture", final.ID)
	if err != nil || !reflect.DeepEqual(stored, final) {
		t.Fatal("closed Verify not durable", err)
	}
	if final.Input.Commit != final.Candidate.Commit || final.Input.Tree != final.Candidate.Tree || final.Input.Key == final.Candidate.Key {
		t.Fatal("Verify did not keep exact input objects under independently retained output key")
	}
}
