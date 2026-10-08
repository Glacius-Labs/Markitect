package projectrun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
)

func TestResumeReconcilesDurableRunningStateAfterCrash(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		inFlight      bool
		retryReady    bool
		wantInvokes   int
		wantStatus    string
		wantWorkCount int
	}{
		{name: "queued work continues", wantInvokes: 4, wantStatus: StatusIntegrated, wantWorkCount: 1},
		{name: "known retry ready continues without resetting ledger", retryReady: true, wantInvokes: 4, wantStatus: StatusIntegrated, wantWorkCount: 2},
		{name: "in flight is blocked without replay", inFlight: true, wantInvokes: 0, wantStatus: StatusBlocked, wantWorkCount: 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root := makeProjectRunFixture(t)
			setupE2EProcess(t, "")
			plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{
				Goal:              "Implement both owned artifacts and integrate them.",
				Managers:          []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")},
				ExecuteAuthorized: true,
			})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			report := persistRunningCrashState(t, root, plan, scenario.inFlight, scenario.retryReady)
			invoker := &resumeCountingInvoker{inner: ProcessInvoker{}}
			resumed, resumeErr := Resume(context.Background(), projectworkHost(), invoker, root, plan.ID)
			if scenario.inFlight {
				if resumeErr == nil || !strings.Contains(resumeErr.Error(), "uncertain in-flight") {
					t.Fatalf("in-flight process was not reported uncertain: %v", resumeErr)
				}
				if resumed.Status != scenario.wantStatus || invoker.calls != scenario.wantInvokes {
					t.Fatalf("in-flight process was replayed or not blocked: status=%s invokes=%d", resumed.Status, invoker.calls)
				}
				uncertainTask := findTask(resumed.Tasks, e2eManagerID("", "project-owner"))
				if len(resumed.Invocations) != 0 || uncertainTask == nil || uncertainTask.State != "uncertain" {
					t.Fatalf("uncertain task state was not retained: %+v", uncertainTask)
				}
				store, err := newRunStore(root)
				if err != nil {
					t.Fatal(err)
				}
				persisted, err := store.readLatestState(plan.ID)
				if err != nil {
					t.Fatal(err)
				}
				persistedTask := findTask(persisted.Tasks, uncertainTask.ManagerID)
				if persisted.Status != StatusBlocked || persistedTask == nil || persistedTask.State != "uncertain" {
					t.Fatalf("uncertainty was not durably blocked: status=%s task=%+v", persisted.Status, persistedTask)
				}
				return
			}
			if resumeErr != nil {
				t.Fatalf("Resume running crash state: status=%s err=%v", resumed.Status, resumeErr)
			}
			if resumed.Status != scenario.wantStatus || invoker.calls != scenario.wantInvokes {
				t.Fatalf("queued work did not continue: status=%s invokes=%d", resumed.Status, invoker.calls)
			}
			rootTask := findTask(resumed.Tasks, e2eManagerID("", "project-owner"))
			if rootTask == nil || rootTask.WorkAttempts != scenario.wantWorkCount || rootTask.Attempts != scenario.wantWorkCount+1 {
				t.Fatalf("attempt ledger was reset or duplicated: %+v", rootTask)
			}
			if scenario.retryReady {
				if len(resumed.Invocations) != len(report.Invocations)+scenario.wantInvokes || resumed.Invocations[0].CostMicros != 17 {
					t.Fatalf("durable prior invocation/cost was not retained: %+v", resumed.Invocations)
				}
			}
		})
	}
}

func persistRunningCrashState(t *testing.T, root string, plan PlanRecord, inFlight, retryReady bool) RunReport {
	t.Helper()
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := store.readCandidate(dir, plan.InitialCandidateID)
	if err != nil {
		t.Fatal(err)
	}
	state := RunReport{
		APIVersion: APIVersion, ID: plan.ID, PlanID: plan.ID, Status: StatusRunning,
		Mode: ModeControlledLocal, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot, ModelDigest: plan.ModelDigest,
		RuntimeDigest: plan.RuntimeDigest, Tasks: cloneTasks(plan.Managers),
		Candidate: CandidateRef{ID: plan.InitialCandidateID, Snapshot: candidate.Digest, Files: map[string]string{}, Integrated: false},
		Revision:  1,
	}
	rootTask := findTask(state.Tasks, e2eManagerID("", "project-owner"))
	if rootTask == nil {
		t.Fatal("plan has no project owner task")
	}
	if inFlight {
		rootTask.State = "invoking"
		rootTask.WorkAttempts = 1
		rootTask.Attempts = 1
	} else if retryReady {
		rootTask.State = "work-retry-ready"
		rootTask.WorkAttempts = 1
		rootTask.Attempts = 1
		rootTask.RepairPhase = "work"
		rootTask.RepairDiagnostic = "candidate path is outside this manager's ownership"
		state.Invocations = append(state.Invocations, InvocationLog{TaskID: rootTask.ID, Role: agentexec.RoleExecutor,
			Phase: "work", Outcome: agentexec.OutcomeProposed, CostMicros: 17})
	}
	if err := store.appendState(state); err != nil {
		t.Fatal(err)
	}
	return state
}

type resumeCountingInvoker struct {
	inner Invoker
	calls int
}

func (i *resumeCountingInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	return i.inner.Run(ctx, config, request, options)
}

func (i *resumeCountingInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return i.inner.Fingerprint(config)
}

var _ = os.ErrNotExist
var _ = filepath.Separator
