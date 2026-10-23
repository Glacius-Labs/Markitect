package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

type applyExecutor struct {
	t                      *testing.T
	git                    *LocalGit
	store                  *runstore.Store
	requests               []run.Request
	managers               []*controlledManagers
	noChange, rejectVerify bool
	assessmentDecision     run.Decision
	failClose              bool
	records                run.Records
}

type deadlineCapture struct {
	calls    int
	deadline time.Time
}

func (c *deadlineCapture) Capture(ctx context.Context, _ string, _ project.Prepared) (project.CandidateRef, error) {
	c.calls++
	deadline, ok := ctx.Deadline()
	if !ok || deadline != c.deadline {
		return project.CandidateRef{}, errors.New("original command deadline missing at capture")
	}
	<-ctx.Done()
	return project.CandidateRef{}, ctx.Err()
}

func TestApplyOriginalDeadlineBoundsCaptureBeforeAnyRunOrPublication(t *testing.T) {
	_, prepared, git, _ := runFixture(t)
	store, _ := runstore.New(git.StateRoot)
	for _, expired := range []bool{true, false} {
		deadline := time.Now().Add(30 * time.Millisecond)
		if expired {
			deadline = time.Now().Add(-time.Second)
		}
		capture := &deadlineCapture{deadline: deadline}
		executor := &applyExecutor{t: t, git: git, store: store}
		command := apply.Command{Executor: executor, Candidates: capture, Records: store, Publisher: &unexpectedPublisher{}}
		start := time.Now()
		report, err := command.Execute(context.Background(), run.Request{ID: "deadline", Project: "fixture", Prepared: prepared, Allowance: run.NewAllowance(deadline, 100), MaxManagers: 5})
		if err == nil || report.Status != model.NotCompleted || len(executor.requests) != 0 || report.Publication != nil || time.Since(start) > time.Second {
			t.Fatal("command escaped original deadline", err, report.Status)
		}
		if expired && capture.calls != 0 || !expired && (capture.calls != 1 || !errors.Is(err, context.DeadlineExceeded)) {
			t.Fatal("deadline did not guard source capture", expired, capture.calls, err)
		}
	}
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
	if request.Mandate == run.Assessment {
		managers.forceDecision = e.assessmentDecision
	}
	managers.failClose = e.failClose
	if e.noChange {
		managers.noChange = true
		managers.acceptedRepair = true
	}
	if request.Mandate == run.Verify && e.rejectVerify {
		managers.forceDecision = run.Reject
	}
	if request.Mandate == run.Verify {
		e.rejectVerify = false
	}
	e.requests = append(e.requests, request)
	e.managers = append(e.managers, managers)
	var records run.Records = e.store
	if e.records != nil {
		records = e.records
	}
	return (&run.Executor{Sessions: managers, Workspaces: e.git, Records: records}).Execute(ctx, request)
}

type unexpectedPublisher struct{ calls int }

func (*unexpectedPublisher) Preflight(context.Context, project.Prepared) error { return nil }
func (p *unexpectedPublisher) Publish(context.Context, apply.PublicationRequest) (apply.Publication, error) {
	p.calls++
	return apply.Publication{Readback: true}, nil
}

func TestApplyStopsForModelBlockerCleanupUncertaintyAndMandatoryStorageFailure(t *testing.T) {
	_, prepared, git, _ := runFixture(t)
	store, _ := runstore.New(git.StateRoot)
	for _, kind := range []string{"blocker", "cleanup", "storage"} {
		t.Run(kind, func(t *testing.T) {
			executor := &applyExecutor{t: t, git: git, store: store}
			switch kind {
			case "blocker":
				executor.assessmentDecision = run.ModelBlocker
			case "cleanup":
				executor.failClose = true
			case "storage":
				executor.records = failingRecords{Records: store, failSave: true}
			}
			publisher := &unexpectedPublisher{}
			command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: publisher}
			report, err := command.Execute(context.Background(), run.Request{ID: kind, Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
			if publisher.calls != 0 || report.Publication != nil || len(executor.requests) != 1 {
				t.Fatal("failed Run reached publication/another Run", kind, publisher.calls)
			}
			if kind == "blocker" && (err != nil || report.Status != model.Failed) || kind != "blocker" && (err == nil || report.Status != model.NotCompleted) {
				t.Fatal(kind, report.Status, err)
			}
		})
	}
}

func TestApplyReopensStoredFinalVerifyAndRefusesSubstitutedAcceptance(t *testing.T) {
	_, prepared, git, _ := runFixture(t)
	store, _ := runstore.New(git.StateRoot)
	executor := &applyExecutor{t: t, git: git, store: store, noChange: true}
	publisher := &unexpectedPublisher{}
	command := apply.Command{Executor: executor, Candidates: git, Records: failingRecords{Records: store, mismatch: true}, Publisher: publisher}
	report, err := command.Execute(context.Background(), run.Request{ID: "stored", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(2*time.Minute), 150), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err == nil || report.Status != model.NotCompleted || publisher.calls != 0 || len(executor.requests) != 3 {
		t.Fatal(report.Status, publisher.calls, len(executor.requests), err)
	}
}

func TestApplyCompleteControlledHierarchyPublishesOnlyStoredClosedVerify(t *testing.T) {
	r, prepared, git, _ := runFixture(t)
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
	// Reuse actual closed acceptance to exercise independent publication races.
	// Only this disposable fixture is restored; each refusal preserves user data.
	for _, kind := range []string{"branch", "index", "partial", "readback", "new file"} {
		t.Run("publication-"+kind, func(t *testing.T) {
			for _, entry := range prepared.Attempt().Model().Inputs().Raw.Census.Entries() {
				if err := os.WriteFile(filepath.Join(r.Dir, filepath.FromSlash(entry.Path().String())), entry.Bytes(), 0644); err != nil {
					t.Fatal(err)
				}
			}
			repository := adapter(t)
			userPath := filepath.Join(r.Dir, "out/deep.txt")
			switch kind {
			case "branch":
				r.Git("switch", "-c", "codex/user-branch")
				defer r.Git("switch", "codex/run-fixture")
			case "index":
				r.Write("out/a.txt", "user staged\n")
				r.Git("add", "out/a.txt")
				r.Write("out/a.txt", "initial\n")
				defer r.Git("reset", "--mixed", head)
			case "partial", "readback", "new file":
				repository.observe = func(event, path string) {
					if kind == "partial" && event == "publication-before-target" && path == "out/deep.txt" || kind == "readback" && event == "publication-readback" {
						r.Write("out/deep.txt", "concurrent user\n")
					}
					if kind == "new file" && event == "publication-before-target" {
						userPath = filepath.Join(r.Dir, "user-new.txt")
						r.Write("user-new.txt", "concurrent user\n")
					}
				}
			}
			result, err := (&Publisher{Repository: repository, Store: git}).Publish(context.Background(), apply.PublicationRequest{Prepared: prepared, Approval: final})
			if err == nil || result.Readback {
				t.Fatal("publication accepted drift", kind, result)
			}
			if kind == "partial" && !reflect.DeepEqual(result.CompletedPaths, []string{"out/a.txt"}) {
				t.Fatal("partial receipt changed", result)
			}
			if kind == "readback" && len(result.CompletedPaths) != 2 {
				t.Fatal("readback hid actual transfers", result)
			}
			if kind == "partial" || kind == "readback" || kind == "new file" {
				data, readErr := os.ReadFile(userPath)
				if readErr != nil || string(data) != "concurrent user\n" {
					t.Fatal("user change lost", string(data), readErr)
				}
			}
			if kind == "index" && r.Git("show", ":out/a.txt") != "user staged" {
				t.Fatal("user staged change lost")
			}
			if kind == "new file" {
				if err := os.Remove(userPath); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestApplyNoChangeStillRunsEveryMandateAndPreservesCapturedIgnoredInput(t *testing.T) {
	_, prepared, git, input := runFixture(t)
	store, _ := runstore.New(git.StateRoot)
	executor := &applyExecutor{t: t, git: git, store: store, noChange: true}
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	report, err := command.Execute(context.Background(), run.Request{ID: "nochange", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(2*time.Minute), 150), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil || report.Status != model.Passed || report.Publication == nil || !report.Publication.NoChange || !report.Publication.Readback || len(report.Publication.CompletedPaths) != 0 || len(executor.requests) != 3 {
		t.Fatal(report.Status, report.Publication, err)
	}
	if input.Commit == prepared.Attempt().Model().Inputs().Head {
		t.Fatal("fixture did not distinguish captured ignored input from HEAD")
	}
	for i, mandate := range []run.Mandate{run.Assessment, run.Implement, run.Verify} {
		if executor.requests[i].Mandate != mandate {
			t.Fatal(executor.requests[i].Mandate)
		}
	}
}

func TestApplyVerifyRejectionStartsFreshCorrectionWithOriginalAllowanceAndFindings(t *testing.T) {
	_, prepared, git, _ := runFixture(t)
	store, _ := runstore.New(git.StateRoot)
	executor := &applyExecutor{t: t, git: git, store: store, rejectVerify: true}
	allowance := run.NewAllowance(time.Now().Add(3*time.Minute), 150)
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	report, err := command.Execute(context.Background(), run.Request{ID: "correction", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: allowance, CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil || report.Status != model.Passed || len(executor.requests) != 5 {
		t.Fatal(report.Status, len(executor.requests), err)
	}
	if got := executor.requests[3]; got.Mandate != run.Implement || got.Input != *report.Workflow.Runs[2].Candidate || !reflect.DeepEqual(got.Findings, report.Workflow.Runs[2].Findings) || len(got.Findings) == 0 {
		t.Fatal("fresh correction lost exact candidate or mandatory Findings", got)
	}
	used := 0
	for _, result := range report.Workflow.Runs {
		used += result.Messages
	}
	if used != allowance.Used() {
		t.Fatal("message allowance reset", used, allowance.Used())
	}
}
