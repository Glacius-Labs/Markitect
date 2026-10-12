//go:build !runtime_unit

package projectapp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projecthistory"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

// These are application boundary integration tests: source acquisition,
// process execution, history and exploration persistence all use the real
// Operations composition.
func TestApplyRejectsReadinessEditAtGuardedWriteBoundary(t *testing.T) {
	t.Parallel()
	setDeliveryProcessFixtureEnv(t, deliveryCandidateCheckEnv, "1")
	root := deliveryFixture(t)
	operations := deliveryOperations()
	deliveryCreateAcknowledgedScope(t, operations, root)
	request := projectrun.PlanRequest{ExplorationID: "delivery", ScopeID: "orders-change", Operation: "apply", Goal: "Update the orders implementation", Managers: deliveryManagers(), ExecuteAuthorized: true}
	plan, err := operations.Plan(PlanOperation{Selection: Selection{Root: root}, Request: request})
	if err != nil {
		t.Fatal(err)
	}
	run, err := operations.Run(context.Background(), RunOperation{Root: root, RunID: plan.ID})
	if err != nil || run.Status != projectrun.StatusIntegrated {
		t.Fatalf("Run = %s, %v", run.Status, err)
	}
	verified, err := operations.Verify(context.Background(), RunOperation{Root: root, RunID: plan.ID})
	if err != nil || verified.Status != projectrun.StatusVerified {
		t.Fatalf("Verify = %s, %v", verified.Status, err)
	}
	preflight, err := operations.PreflightApply(PreflightOperation{Root: root, RunID: plan.ID, CandidateID: run.Candidate.ID})
	if err != nil {
		t.Fatal(err)
	}
	base := operations
	guarded := operations
	mutated := false
	guarded.Host.Load = func(path, revision string) (*projectvalue.Project, error) {
		if revision == "" && !mutated {
			mutated = true
			deliveryAppendBlockingDecision(t, base, root)
		}
		return base.Host.Load(path, revision)
	}
	apply, err := guarded.Apply(ApplyOperation{Root: root, Request: projectrun.ApplyRequest{RunID: preflight.RunID, PlanID: preflight.PlanID, CandidateID: preflight.CandidateID, ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch, ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree}})
	if err == nil || !mutated {
		t.Fatalf("Apply did not reject concurrent readiness edit: %+v, %v", apply, err)
	}
	if apply.Status == projectrun.StatusApplied || len(apply.Written) != 0 {
		t.Fatalf("stale readiness edit wrote candidate source: %+v", apply)
	}
	if data, readErr := os.ReadFile(filepath.Join(root, "src", "orders", "implementation.txt")); readErr != nil || string(data) != "orders implementation v1\n" {
		t.Fatalf("guarded Apply changed source: %q, %v", data, readErr)
	}
	status, err := base.Status(RunOperation{Root: root, RunID: plan.ID})
	if err != nil || status.Run.Status != projectrun.StatusVerified || status.Run.Candidate.ID != run.Candidate.ID {
		t.Fatalf("zero-write rejection changed verified run state: %+v, %v", status.Run, err)
	}
	receipts, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(projectrun.RunsPath), plan.ID, "apply", "*.json"))
	if err != nil || len(receipts) != 0 {
		t.Fatalf("zero-write rejection persisted Apply receipts %v, err=%v", receipts, err)
	}
	exploration, err := base.Explore(ExploreOperation{Selection: Selection{Root: root}, ExplorationID: "delivery"})
	if err != nil || exploration.Record == nil || len(exploration.Record.Completions) != 0 {
		t.Fatalf("zero-write rejection persisted exploration completion: %+v, %v", exploration.Record, err)
	}
}

func TestGuidedPreviewsMatchExecuteWithoutWritingHistory(t *testing.T) {
	t.Parallel()
	for _, later := range []bool{false, true} {
		name := "first model change"
		if later {
			name = "later model change"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := deliveryFixture(t)
			operations := deliveryOperations()
			if later {
				revision := deliveryCommitStatementChange(t, root, "Implement the %s artifact.", "Implement the %s artifact with explicitly documented line-item behavior.")
				if _, err := projecthistory.EnsureAcceptedHistory(root, revision); err != nil {
					t.Fatal(err)
				}
				deliveryCommitStatementChange(t, root, "Implement the %s artifact with explicitly documented line-item behavior.", "Implement the %s artifact with explicitly documented line-item and tax behavior.")
			} else {
				deliveryCommitStatementChange(t, root, "Implement the %s artifact.", "Implement the %s artifact with explicitly documented line-item behavior.")
			}
			head := deliveryGit(t, root, "rev-parse", "HEAD")
			historyPath := filepath.Join(root, filepath.FromSlash(".markitect/state/briefings/history.json"))
			beforeRead, err := projecthistory.Read(root)
			if err != nil {
				t.Fatal(err)
			}
			before := beforeRead.PersistedDigest
			deliveryCreateAcknowledgedScope(t, operations, root)
			request := projectrun.PlanRequest{ExplorationID: "delivery", ScopeID: "orders-change", Operation: "apply", Goal: "Update the orders implementation", Managers: deliveryManagers()}
			preview, err := operations.Plan(PlanOperation{Selection: Selection{Root: root}, Request: request})
			if err != nil {
				t.Fatalf("plan preview: %v", err)
			}
			if !later {
				setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
				full, _ := operations.FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
				if !deliveryMapsEqual(full.BriefingDigests, preview.BriefingDigests) {
					t.Fatalf("FullVerify briefing map %v differs from Plan preview %v", full.BriefingDigests, preview.BriefingDigests)
				}
				if _, statErr := os.Stat(historyPath); !os.IsNotExist(statErr) {
					t.Fatalf("previews wrote accepted history on a fresh guided repository: %v", statErr)
				}
			}
			afterRead, readErr := projecthistory.Read(root)
			if readErr != nil || afterRead.PersistedDigest != before {
				t.Fatalf("previews changed briefing store: before=%s after=%s err=%v", before, afterRead.PersistedDigest, readErr)
			}
			request.ExecuteAuthorized = true
			executed, err := operations.Plan(PlanOperation{Selection: Selection{Root: root}, Request: request})
			if err != nil {
				t.Fatalf("execute rejected preview acknowledgement: %v", err)
			}
			briefed := false
			for _, digest := range executed.BriefingDigests {
				briefed = briefed || digest != ""
			}
			if !briefed || !deliveryMapsEqual(executed.BriefingDigests, preview.BriefingDigests) {
				t.Fatalf("execute briefings %v differ from preview %v", executed.BriefingDigests, preview.BriefingDigests)
			}
			if _, statErr := os.Stat(historyPath); statErr != nil {
				t.Fatalf("execute did not persist accepted history: %v", statErr)
			}
		})
	}
}
