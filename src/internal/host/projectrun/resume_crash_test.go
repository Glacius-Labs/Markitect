package projectrun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

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
