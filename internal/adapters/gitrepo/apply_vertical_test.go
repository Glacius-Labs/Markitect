package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

type applyExecutor struct {
	t                      *testing.T
	git                    *LocalGit
	store                  *runstore.Store
	requests               []run.Request
	managers               []*controlledManagers
	noChange, rejectVerify bool
	assessmentDecision     run.Decision
	assessmentFindings     []run.Finding
	// verifyFindings are Verify Root's approval Findings; refusedApprovals are
	// the approvals it tries first, each of which must be refused.
	verifyFindings   []run.Finding
	refusedApprovals [][]run.Finding
	realization      func(context.Context, *controlledManager) error // replaces Implement's writes
	failClose        bool
	records          run.Records
	// wrap, when set, supplies the Sessions of each Run around its managers.
	wrap func(run.Mandate, *controlledManagers) run.Sessions
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
		managers.forceDecision, managers.rootFindings = e.assessmentDecision, e.assessmentFindings
	}
	managers.failClose = e.failClose
	if e.noChange {
		managers.noChange = true
		managers.acceptedRepair = true
	}
	if request.Mandate == run.Implement {
		managers.realization = e.realization
	}
	if request.Mandate == run.Verify {
		managers.rootFindings, managers.refusedApprovals = e.verifyFindings, e.refusedApprovals
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
	var sessions run.Sessions = managers
	if e.wrap != nil {
		sessions = e.wrap(request.Mandate, managers)
	}
	return (&run.Executor{Sessions: sessions, Workspaces: e.git, Records: records}).Execute(ctx, request)
}

// deepRegistry is a/deep's registry shard in the run fixture.
const deepRegistry = "apiVersion: markitect.registry/v1\nartifacts:\n  - path: out/deep.txt\n    form: code\n    realizes: [a/deep/item]\n    why: Tiny controlled realization.\n"

// deletedDefinition leaves a/deep an artifact of a Definition the owner has
// deleted from the model: its registry link is stale.
func deletedDefinition(r *testkit.Repo) {
	r.Write("out/gone.txt", "realizes a deleted Definition\n")
	r.Write(".markitect/registry/a/deep/registry.yaml", deepRegistry+"  - path: out/gone.txt\n    form: code\n    realizes: [a/deep/gone]\n    why: Realized a Definition the owner deleted.\n")
}

// staleLinkSessions records every Manager's check of the state it starts
// from. In Implement, a/deep removes the artifact of its deleted Definition
// and the link once its parent's correction names the stale link.
type staleLinkSessions struct {
	controlled *controlledManagers
	mandate    run.Mandate
	mu         sync.Mutex
	checks     map[string]run.Check
}

type staleLinkSession struct {
	*controlledManager
	owner   *staleLinkSessions
	checked bool
}

func (s *staleLinkSessions) Open(ctx context.Context, spec run.SessionSpec) (run.Session, error) {
	session, err := s.controlled.Open(ctx, spec)
	if session == nil {
		return nil, err
	}
	return &staleLinkSession{controlledManager: session.(*controlledManager), owner: s}, err
}

func (s *staleLinkSession) Exchange(ctx context.Context, message run.Message) error {
	if !s.checked {
		s.checked = true
		check, err := s.spec.Tools.Check(ctx)
		if err != nil {
			return err
		}
		s.owner.mu.Lock()
		s.owner.checks[s.spec.Manager.Package] = check
		s.owner.mu.Unlock()
	}
	if s.owner.mandate == run.Implement && s.spec.Manager.Package == "a/deep" {
		for _, e := range message.Events {
			if e.Assignment == nil || len(e.Assignment.Findings) == 0 {
				continue
			}
			if text := fmt.Sprint(e.Assignment.Findings); !strings.Contains(text, "registry.stale_link") || !strings.Contains(text, "out/gone.txt") {
				return fmt.Errorf("correction lost the stale link diagnostic: %s", text)
			}
			root := s.spec.Workspace.Location().Directory
			if err := os.Remove(filepath.Join(root, "out/gone.txt")); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(root, ".markitect/registry/a/deep/registry.yaml"), []byte(deepRegistry), 0600); err != nil {
				return err
			}
		}
	}
	return s.controlledManager.Exchange(ctx, message)
}

const (
	paymentsShard = ".markitect/registry/payments/registry.yaml"
	bShard        = "apiVersion: markitect.registry/v1\nartifacts:\n  - path: out/merged/pay.txt\n    form: code\n    realizes: [b/Pay]\n    why: Payment realization taken over after the merge.\n"
)

// mergedPackage is Phase C's C7 in miniature: the owner merged a Package
// payments into b, moving its Definition to b/Pay and its area out/merged/ to
// b, while payments' shard still registers the artifact.
func mergedPackage(r *testkit.Repo) {
	r.Write(".markitect/model/b/pay.yaml", "imports: [{package: markitect/core, as: Core}]\nid: Pay\nkind: Core.Kind\nbody: A payment obligation merged into b.\nspec: {}\n")
	r.Write(".markitect/execution.yaml", "apiVersion: markitect.execution/v2\nownership:\n  - {path: './', owner: .}\n  - {path: out/a.txt, owner: a}\n  - {path: out/accounted.ignored, owner: a}\n  - {path: out/deep.txt, owner: a/deep}\n  - {path: out/merged/, owner: b}\nchecks: []\n")
	r.Write("out/merged/pay.txt", "payment realization\n")
	r.Write(paymentsShard, "apiVersion: markitect.registry/v1\nartifacts:\n  - path: out/merged/pay.txt\n    form: code\n    realizes: [payments/pay]\n    why: Realized payments before the merge.\n")
}

// §6.1, §6.3: a merged Package leaves an orphaned shard. The input starts
// Apply; the Manager that now owns the artifact's area takes the registration
// over in its own shard in Implement, Root deletes the orphaned shard, and
// publication passes.
func TestApplyRealizesAPackageMergeThroughTheAreaOwnerAndRoot(t *testing.T) {
	r, prepared, git, _ := runFixture(t, mergedPackage)
	store, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	var runs []*staleLinkSessions
	executor := &applyExecutor{t: t, git: git, store: store, noChange: true, wrap: func(mandate run.Mandate, managers *controlledManagers) run.Sessions {
		managers.realization = func(ctx context.Context, m *controlledManager) error {
			root := m.spec.Workspace.Location().Directory
			check, err := m.spec.Tools.Check(ctx)
			if err != nil {
				return err
			}
			switch m.spec.Manager.Package {
			case "b":
				if check.Valid || len(check.Stale) != 1 || !strings.Contains(check.Stale[0], "out/merged/pay.txt") {
					return fmt.Errorf("b was not handed the orphaned registration: %+v", check)
				}
				// b writes only its own shard; its entry supersedes the orphaned one.
				if err := os.MkdirAll(filepath.Join(root, ".markitect/registry/b"), 0700); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(root, ".markitect/registry/b/registry.yaml"), []byte(bShard), 0600)
			case ".":
				if check.Valid || len(check.Stale) != 1 || !strings.Contains(check.Stale[0], paymentsShard) {
					return fmt.Errorf("Root was not handed only the orphaned shard: %+v", check)
				}
				return os.Remove(filepath.Join(root, filepath.FromSlash(paymentsShard)))
			}
			return nil
		}
		s := &staleLinkSessions{controlled: managers, mandate: mandate, checks: map[string]run.Check{}}
		runs = append(runs, s)
		return s
	}}
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	request := run.Request{ID: "merged-package", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 200), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
	report, err := command.Execute(context.Background(), request)
	if err != nil || report.Status != model.Passed || len(runs) != 3 {
		t.Fatalf("status=%s error=%v final=%s/%s runs=%d", report.Status, err, report.Workflow.Final.Technical, report.Workflow.Final.Decision, len(runs))
	}
	// Each Manager's first check: b owes the registration, Root it and the shard.
	owes := map[string]int{".": 2, "a": 0, "a/deep": 0, "b": 1, "b/empty": 0}
	for i, mandate := range []run.Mandate{run.Assessment, run.Implement, run.Verify} {
		for manager, n := range owes {
			check := runs[i].checks[manager]
			switch {
			case mandate == run.Assessment && (!check.Valid || len(check.Stale) != n):
				t.Fatalf("Assessment check of %s does not report the merge as work: %+v", manager, check)
			case mandate == run.Implement && (check.Valid != (n == 0) || len(check.Stale) != n):
				t.Fatalf("Implement check of %s is not held to its own responsibility: %+v", manager, check)
			case mandate == run.Verify && (!check.Valid || len(check.Stale) != 0):
				t.Fatalf("Verify check of %s still sees orphaned registry state: %+v", manager, check)
			}
		}
	}
	if report.Publication == nil || !report.Publication.Readback {
		t.Fatalf("publication=%+v", report.Publication)
	}
	if paths := slices.Sorted(slices.Values(report.Publication.CompletedPaths)); !reflect.DeepEqual(paths, []string{".markitect/registry/b/registry.yaml", paymentsShard}) {
		t.Fatal("publication did not move exactly the registration", paths)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, filepath.FromSlash(paymentsShard))); !os.IsNotExist(err) {
		t.Fatal("orphaned shard survived publication", err)
	}
	if data, err := os.ReadFile(filepath.Join(r.Dir, ".markitect/registry/b/registry.yaml")); err != nil || string(data) != bShard {
		t.Fatal("the area owner's registration was not published", string(data), err)
	}
	if data, err := os.ReadFile(filepath.Join(r.Dir, "out/merged/pay.txt")); err != nil || string(data) != "payment realization\n" {
		t.Fatal("the merged artifact changed", string(data), err)
	}
}

// §6.3: deleting a Definition leaves a stale registry link, which is
// realization, not a model error. The input starts Apply; Assessment sees the
// link and approves; in Implement only the shard owner's subtree and Root are
// held to it, the owner removes the artifact and the link after its parent's
// correction, and publication passes.
func TestApplyRealizesADeletedDefinitionThroughTheShardOwner(t *testing.T) {
	r, prepared, git, _ := runFixture(t, deletedDefinition)
	store, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	var runs []*staleLinkSessions
	executor := &applyExecutor{t: t, git: git, store: store, noChange: true, wrap: func(mandate run.Mandate, managers *controlledManagers) run.Sessions {
		s := &staleLinkSessions{controlled: managers, mandate: mandate, checks: map[string]run.Check{}}
		runs = append(runs, s)
		return s
	}}
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	request := run.Request{ID: "deleted-definition", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 200), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
	report, err := command.Execute(context.Background(), request)
	if err != nil || report.Status != model.Passed || len(runs) != 3 {
		t.Fatalf("status=%s error=%v final=%s/%s runs=%d", report.Status, err, report.Workflow.Final.Technical, report.Workflow.Final.Decision, len(runs))
	}
	responsible := map[string]bool{".": true, "a": true, "a/deep": true, "b": false, "b/empty": false}
	for i, mandate := range []run.Mandate{run.Assessment, run.Implement, run.Verify} {
		for manager, owes := range responsible {
			check := runs[i].checks[manager]
			stale := len(check.Stale) == 1 && strings.Contains(check.Stale[0], "out/gone.txt") && strings.Contains(check.Stale[0], "a/deep/gone")
			switch {
			case mandate == run.Assessment && (!check.Valid || stale != owes || !owes && len(check.Stale) != 0):
				t.Fatalf("Assessment check of %s does not report the stale link as work: %+v", manager, check)
			case mandate == run.Implement && (check.Valid == owes || stale != owes || owes && !strings.Contains(check.Diagnostic, "registry.stale_link")):
				t.Fatalf("Implement check of %s is not held to its own responsibility: %+v", manager, check)
			case mandate == run.Verify && (!check.Valid || len(check.Stale) != 0):
				t.Fatalf("Verify check of %s still sees a stale link: %+v", manager, check)
			}
		}
	}
	if executor.managers[1].corrections != 1 || len(executor.managers[1].refusals) != 1 || !strings.Contains(executor.managers[1].refusals[0], "registry.stale_link") {
		t.Fatal("the stale link did not reach the shard owner through its parent's correction", executor.managers[1].corrections, executor.managers[1].refusals)
	}
	if report.Publication == nil || !report.Publication.Readback {
		t.Fatalf("publication=%+v", report.Publication)
	}
	if paths := slices.Sorted(slices.Values(report.Publication.CompletedPaths)); !reflect.DeepEqual(paths, []string{".markitect/registry/a/deep/registry.yaml", "out/gone.txt"}) {
		t.Fatal("publication did not remove exactly the artifact and its link", paths)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "out/gone.txt")); !os.IsNotExist(err) {
		t.Fatal("artifact of the deleted Definition survived", err)
	}
	if data, err := os.ReadFile(filepath.Join(r.Dir, ".markitect/registry/a/deep/registry.yaml")); err != nil || string(data) != deepRegistry {
		t.Fatal("stale link survived publication", string(data), err)
	}
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
			report, err := command.Execute(context.Background(), run.Request{ID: kind, Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
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
	report, err := command.Execute(context.Background(), run.Request{ID: "stored", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 150), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
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
	request := run.Request{ID: "apply", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 200), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
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
	report, err := command.Execute(context.Background(), run.Request{ID: "nochange", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 150), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil || report.Status != model.Passed || report.Publication == nil || !report.Publication.NoChange || !report.Publication.Readback || len(report.Publication.CompletedPaths) != 0 || len(executor.requests) != 3 {
		t.Fatal(report.Status, report.Publication, err)
	}
	if input.Commit == prepared.Attempt().Model().Inputs().Head {
		t.Fatal("fixture did not distinguish captured ignored input from HEAD")
	}
	if changes := rootChanges(executor.managers[2]); changes == nil || *changes != "" {
		t.Fatal("Verify of an unchanged candidate saw changes", changes)
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
	allowance := run.NewAllowance(runDeadline(t), 150)
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	report, err := command.Execute(context.Background(), run.Request{ID: "correction", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: allowance, CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil || report.Status != model.Passed || len(executor.requests) != 5 {
		t.Fatal(report.Status, len(executor.requests), err, report.Workflow.Final.Errors)
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

// budgetDrain spends the shared allowance in Verify down to keep messages
// before Root's judgment takes effect: Root's judgment is held, and each
// continued response reads Notes, which costs its message, until keep remain.
type budgetDrain struct {
	controlled *controlledManagers
	allowance  *run.Allowance
	keep       int
}

func (d budgetDrain) Open(ctx context.Context, spec run.SessionSpec) (run.Session, error) {
	if spec.Manager.Package != "." {
		return d.controlled.Open(ctx, spec)
	}
	root := &drainingRoot{allowance: d.allowance, keep: d.keep, tools: spec.Tools}
	spec.Tools.Judge = func(_ context.Context, decision run.Decision, findings []run.Finding) error {
		root.held, root.decision, root.findings = true, decision, findings
		return nil
	}
	session, err := d.controlled.Open(ctx, spec)
	if err != nil {
		return session, err
	}
	root.Session = session
	return root, nil
}

type drainingRoot struct {
	run.Session
	allowance *run.Allowance
	keep      int
	tools     run.Tools
	held      bool
	decision  run.Decision
	findings  []run.Finding
}

func (r *drainingRoot) Exchange(ctx context.Context, m run.Message) error {
	if !r.held {
		return r.Session.Exchange(ctx, m)
	}
	if r.allowance.Remaining() > r.keep {
		_, err := r.tools.ReadNotes(ctx, "")
		return err
	}
	r.held = false
	return r.tools.Judge(ctx, r.decision, r.findings)
}

// §20.1, §20.2 (phase C M-c3-C1: a correction pair started with 39 messages
// left and needed at least 42): Apply starts no correction pair the remaining
// shared allowance cannot start, as Implement and Verify each need every
// Manager once. It ends incomplete with that cause, not with a refused Run
// request. Only an impossible start is refused: with exactly enough messages
// the pair starts, without a promise that it fits.
func TestApplyStartsNoCorrectionPairTheRemainingBudgetCannotCover(t *testing.T) {
	for _, short := range []bool{true, false} {
		t.Run(fmt.Sprintf("short=%v", short), func(t *testing.T) {
			_, prepared, git, _ := runFixture(t)
			store, _ := runstore.New(git.StateRoot)
			keep := 2 * len(prepared.Topology().Managers())
			if short {
				keep--
			}
			allowance := run.NewAllowance(runDeadline(t), 100)
			drained := false
			executor := &applyExecutor{t: t, git: git, store: store, rejectVerify: true, wrap: func(mandate run.Mandate, managers *controlledManagers) run.Sessions {
				if mandate != run.Verify || drained {
					return managers
				}
				drained = true
				return budgetDrain{controlled: managers, allowance: allowance, keep: keep}
			}}
			command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
			report, err := command.Execute(context.Background(), run.Request{ID: "correction-budget", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: allowance, CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
			stopped := err != nil && strings.Contains(err.Error(), "budget does not cover a correction pair")
			if !short {
				if stopped || len(executor.requests) < 4 {
					t.Fatal("a correction pair the allowance can start was refused", len(executor.requests), err)
				}
				return
			}
			if !stopped || !errors.Is(err, run.ErrBudget) || strings.Contains(err.Error(), "invalid Run request") {
				t.Fatal("the impossible correction pair did not end as a budget stop", err)
			}
			if report.Status != model.NotCompleted || report.Publication != nil || len(executor.requests) != 3 || report.Workflow.Final.Decision != run.Reject || allowance.Remaining() != keep {
				t.Fatal("the budget stop started a Run or was not incomplete", report.Status, len(executor.requests), report.Workflow.Final.Decision, allowance.Remaining())
			}
		})
	}
}

// fixtureRequest is an Apply request over the run fixture with ample limits.
func fixtureRequest(t *testing.T, id string, prepared project.Prepared) run.Request {
	return run.Request{ID: id, Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 150), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
}

// rootChanges is the diff Verify Root's assignment carried, nil when no
// assignment of the Run carried one.
func rootChanges(c *controlledManagers) *string {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range c.messages {
		for _, e := range m.Events {
			if e.Assignment != nil && e.Assignment.Changes != nil {
				return e.Assignment.Changes
			}
		}
	}
	return nil
}

// DEC-073: an Assessment Finding that names a current Definition reaches
// Implement, as the repair of a defect under an unchanged model must. A Finding
// drawn only from earlier Notes names none: Implement never receives it, and
// it is kept as a Note, noted and not implemented.
func TestApplyHandsImplementOnlyFindingsThatNameACurrentDefinition(t *testing.T) {
	_, prepared, git, _ := runFixture(t)
	store, _ := runstore.New(git.StateRoot)
	repair := run.Finding{Manager: ".", Definition: "a/Thing", Text: "out/a.txt does not realize a/Thing; repair it."}
	stale := run.Finding{Manager: ".", Text: "Earlier Notes suggest more test coverage."}
	executor := &applyExecutor{t: t, git: git, store: store, assessmentFindings: []run.Finding{{Definition: repair.Definition, Text: repair.Text}, {Text: stale.Text}}}
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	report, err := command.Execute(context.Background(), fixtureRequest(t, "initial", prepared))
	if err != nil || report.Status != model.Passed || len(executor.requests) != 3 {
		t.Fatal(report.Status, len(executor.requests), err)
	}
	assessment := report.Workflow.Runs[0]
	if !slices.Contains(assessment.Findings, repair) || !slices.Contains(assessment.Findings, stale) {
		t.Fatal("the Assessment result lost a Finding", assessment.Findings)
	}
	if got := executor.requests[1]; got.Mandate != run.Implement || !reflect.DeepEqual(got.Findings, []run.Finding{repair}) {
		t.Fatal("Implement did not receive exactly the Definition-bound Finding", got.Findings)
	}
	for _, s := range executor.managers[1].sessions {
		if !reflect.DeepEqual(s.spec.Findings, []run.Finding{repair}) {
			t.Fatal("an Implement Manager's Initial Findings differ", s.spec.Manager.Package, s.spec.Findings)
		}
	}
	page, err := store.Notes(context.Background(), "fixture", "")
	if err != nil {
		t.Fatal(err)
	}
	kept := 0
	for _, n := range page.Notes {
		if n.Run == assessment.ID && strings.Contains(n.Context, "noted, not implemented") {
			if n.Text != stale.Text || n.Manager != "." {
				t.Fatal("a Finding with a Definition became a Note, or the Note lost its origin", n)
			}
			kept++
		}
	}
	if kept != 1 {
		t.Fatal("the unbound Assessment Finding was not kept as exactly one Note", kept, page.Notes)
	}
}

// DEC-073: every Verify judges the complete diff since the command's bound
// input. A change of the first Implement stays visible to the Verify after a
// correction, which receives the reject's Definition-bound Findings.
func TestApplyVerifySeesTheDiffSinceTheBoundInputAcrossCorrections(t *testing.T) {
	_, prepared, git, _ := runFixture(t)
	store, _ := runstore.New(git.StateRoot)
	executor := &applyExecutor{t: t, git: git, store: store, rejectVerify: true}
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	report, err := command.Execute(context.Background(), fixtureRequest(t, "diff", prepared))
	if err != nil || report.Status != model.Passed || len(executor.requests) != 5 {
		t.Fatal(report.Status, len(executor.requests), err)
	}
	origin := executor.requests[0].Origin
	if origin.Input != executor.requests[0].Input || origin.Verified != "" {
		t.Fatal("the first Run is not bound to the captured input", origin)
	}
	for i, r := range executor.requests {
		if r.Origin != origin || report.Workflow.Runs[i].Origin != origin {
			t.Fatal("the bound input changed across the command", i, r.Origin)
		}
	}
	rejected := *report.Workflow.Runs[2].Candidate
	if executor.requests[3].Input != rejected {
		t.Fatal("the correction did not start from the rejected candidate")
	}
	for i, m := range executor.managers {
		changes := rootChanges(m)
		if executor.requests[i].Mandate != run.Verify {
			if changes != nil {
				t.Fatal("a non-Verify Root received changes", i)
			}
			continue
		}
		// The first Implement changed out/a.txt; the correction left it alone.
		if changes == nil || !strings.Contains(*changes, "out/a.txt") || !strings.Contains(*changes, "-initial") || !strings.Contains(*changes, "+parent result") {
			t.Fatal("Verify did not see the change since the bound input", i, changes)
		}
	}
	if got := executor.requests[3].Findings; len(got) == 0 || len(got) != len(report.Workflow.Runs[2].Findings) || !reflect.DeepEqual(got, run.Initial(prepared, report.Workflow.Runs[2].Findings)) {
		t.Fatal("the correction lost a Definition-bound reject Finding", got)
	}
}

// DEC-073 (WP4a): the host states that the bound input was already verified
// only from a stored closed Verify approval of exactly that input under the
// same basis. A first command gets no such fact; a command over the verified
// candidate gets it in every Run and Session. The fact does not block a
// needed repair: a Definition-bound Assessment Finding reaches Implement, and
// Verify sees the repair against the bound input and approves it only with a
// Finding that names the current Definition whose defect it corrects, which
// the result records. An unchanged candidate under the fact needs none.
func TestApplyStatesTheVerifiedFactOnlyForAStoredApprovalOfTheSameInput(t *testing.T) {
	// Ordinary Git cannot commit an empty Package directory, so this owner's
	// model gives the fixture's Package b/empty an obligation.
	r, prepared, git, _ := runFixture(t, func(r *testkit.Repo) {
		r.Write(".markitect/model/b/empty/duty.yaml", "imports: [{package: markitect/core, as: Core}]\nid: Duty\nkind: Core.Kind\nbody: An obligation of the formerly empty Package.\nspec: {}\n")
	})
	if tree := r.Git("ls-tree", "-r", "-t", "HEAD", ".markitect/model/b/empty"); !strings.Contains(tree, "duty.yaml") {
		t.Fatal("the Package is not an ordinary Git tree", tree)
	}
	store, _ := runstore.New(git.StateRoot)
	ctx := context.Background()
	first := &applyExecutor{t: t, git: git, store: store}
	command := apply.Command{Executor: first, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	report, err := command.Execute(ctx, fixtureRequest(t, "first", prepared))
	if err != nil || report.Status != model.Passed || first.requests[0].Origin.Verified != "" {
		t.Fatal("a first command received a fact or failed", report.Status, err)
	}
	approval := report.Workflow.Final
	// The owner commits the published candidate with ordinary Git: another
	// commit, the same captured content.
	r.Commit("owner commits the published candidate")
	next := prepareFixture(t, r.Dir)
	if next.Basis() != prepared.Basis() {
		t.Fatal("realization changed the basis")
	}
	defect := run.Finding{Manager: ".", Definition: "a/Thing", Text: "out/a.txt still violates a/Thing: repair it."}
	repair := func(_ context.Context, m *controlledManager) error {
		if m.spec.Manager.Package != "a" {
			return nil
		}
		return os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, "out", "a.txt"), []byte("repaired\n"), 0600)
	}
	// Verify Root's approval of the repair names the defect it corrects; a
	// plain approval and one naming an unknown Definition are refused first.
	corrected := run.Finding{Manager: ".", Definition: "a/Thing", Text: "The bound input's out/a.txt violated a/Thing; the change corrects it."}
	second := &applyExecutor{t: t, git: git, store: store, noChange: true, assessmentFindings: []run.Finding{{Definition: defect.Definition, Text: defect.Text}}, realization: repair,
		verifyFindings: []run.Finding{{Definition: corrected.Definition, Text: corrected.Text}}, refusedApprovals: [][]run.Finding{nil, {{Definition: "a/Missing", Text: corrected.Text}}}}
	command.Executor = second
	report, err = command.Execute(ctx, fixtureRequest(t, "second", next))
	if err != nil || report.Status != model.Passed || len(second.requests) != 3 || report.Publication == nil || !report.Publication.Readback {
		t.Fatal(report.Status, err)
	}
	if refusals := second.managers[2].refusals; len(refusals) != 2 || !strings.Contains(refusals[0], "Verify Run "+approval.ID+" already verified") || !strings.Contains(refusals[0], "names the defect it corrects") || !strings.Contains(refusals[1], "a/Missing, which is not a Definition of the checked model") {
		t.Fatal("an approval of the change without a current corrected Definition was not refused", refusals)
	}
	if verified := report.Workflow.Runs[2]; verified.Decision != run.Approve || !slices.Contains(verified.Findings, corrected) {
		t.Fatal("the Verify result does not record the corrected Definition", verified.Decision, verified.Findings)
	}
	if got := second.requests[1].Findings; !reflect.DeepEqual(got, []run.Finding{defect}) {
		t.Fatal("the needed repair did not reach Implement under the fact", got)
	}
	for i, request := range second.requests {
		if request.Origin.Input.Commit == approval.Input.Commit || request.Origin.Input.Tree != approval.Input.Tree || request.Origin.Verified != approval.ID || report.Workflow.Runs[i].Origin != request.Origin {
			t.Fatal("the fact did not reach every Run", i, request.Origin)
		}
		for _, s := range second.managers[i].sessions {
			if s.spec.Origin != request.Origin {
				t.Fatal("a Session lacks the fact", s.spec.Manager.Package)
			}
		}
	}
	if changes := rootChanges(second.managers[2]); changes == nil || !strings.Contains(*changes, "-parent result") || !strings.Contains(*changes, "+repaired") {
		t.Fatal("Verify did not see the repair against the bound input", changes)
	}
	// The published repair, committed by the owner and captured under another
	// commit, keeps the fact of its approval; one changed byte loses it.
	repaired := report.Workflow.Final
	r.Commit("owner commits the published repair")
	capture := func() (project.Prepared, project.CandidateRef, []run.Result) {
		p := prepareFixture(t, r.Dir)
		input, err := git.Capture(ctx, "fixture", p)
		if err != nil {
			t.Fatal(err)
		}
		found, err := store.Approvals(ctx, "fixture", run.ApprovalKey(p.Basis(), input))
		if err != nil {
			t.Fatal(err)
		}
		return p, input, found
	}
	same, input, found := capture()
	if input.Commit == repaired.Input.Commit || input.Tree != repaired.Input.Tree || run.Verified(same, input, found) != repaired.ID {
		t.Fatal("the same state under another commit lost the fact", input, repaired.Input)
	}
	// Under the fact, an unchanged candidate needs no named defect: Verify
	// sees an empty diff and approves plainly.
	third := &applyExecutor{t: t, git: git, store: store, noChange: true}
	command.Executor = third
	report, err = command.Execute(ctx, fixtureRequest(t, "third", same))
	if err != nil || report.Status != model.Passed || len(third.requests) != 3 || third.requests[2].Origin.Verified != repaired.ID {
		t.Fatal("the unchanged verified input was not approved plainly", report.Status, err)
	}
	if changes := rootChanges(third.managers[2]); changes == nil || *changes != "" || len(third.managers[2].refusals) != 0 {
		t.Fatal("Verify of the unchanged input saw changes or a refusal", changes, third.managers[2].refusals)
	}
	r.Write("out/a.txt", "repairee\n")
	r.Commit("owner changes one byte")
	edited, input, found := capture()
	if len(found) != 0 || run.Verified(edited, input, []run.Result{repaired, approval}) != "" {
		t.Fatal("a one-byte change kept the fact")
	}
	// No fact without the same basis, input and closed approval.
	_, changed, _, _ := runFixture(t, func(r *testkit.Repo) {
		r.Write(".markitect/model/b/empty/duty.yaml", "imports: [{package: markitect/core, as: Core}]\nid: Duty\nkind: Core.Kind\nbody: An obligation of the formerly empty Package.\nspec: {}\n")
		r.Write(".markitect/execution.yaml", "# Changed execution configuration.\napiVersion: markitect.execution/v2\nchecks: []\n")
	})
	other := approval.Input
	other.Tree = "other"
	rejected := approval
	rejected.Decision = run.Reject
	unclosed := approval
	unclosed.Cleanup = nil
	for name, tc := range map[string]struct {
		prepared project.Prepared
		input    project.CandidateRef
		stored   run.Result
	}{"basis": {changed, approval.Input, approval}, "input": {prepared, other, approval}, "decision": {prepared, approval.Input, rejected}, "closure": {prepared, approval.Input, unclosed}} {
		if got := run.Verified(tc.prepared, tc.input, []run.Result{tc.stored}); got != "" {
			t.Fatal("fact without the same", name)
		}
	}
	if run.Verified(prepared, approval.Input, []run.Result{approval}) != approval.ID {
		t.Fatal("the stored approval itself gives no fact")
	}
}
