package projectrun

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

type failedManagerReceiptInvoker struct {
	mode  string
	calls int
}

func (i *failedManagerReceiptInvoker) Run(_ context.Context, _ agentexec.Config, _ agentexec.Request, _ agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	if i.mode == "preflight" {
		return agentexec.RunResult{}, errors.New("manager preflight failed")
	}

	inputTokens, outputTokens := int64(2_000_000), int64(3_000_000)
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: &inputTokens, OutputTokens: &outputTokens}
	nativeWork := &agentexec.NativeWork{
		WorkspaceBaseDigest:  "sha256:" + strings.Repeat("0", 64),
		WorkspaceFinalDigest: "sha256:" + strings.Repeat("1", 64),
		DeltaDigest:          "sha256:" + strings.Repeat("2", 64),
		ChangedPaths:         []string{"src/orders/implementation.txt"},
		ToolCalls:            1,
		HelperStarts:         0,
		HelperAccounting:     "disabled",
	}
	runID := "fixture-native-manager-run"
	receipt := agentexec.Receipt{
		APIVersion: agentexec.APIVersion, RunID: runID, InputDigest: "sha256:" + strings.Repeat("3", 64),
		Outcome: agentexec.OutcomeIncomplete, Usage: usage, NativeWork: nativeWork,
		PrivateLogDigest: "sha256:" + strings.Repeat("4", 64), StdoutDigest: "sha256:" + strings.Repeat("5", 64),
	}
	if i.mode == "process-error" {
		return agentexec.RunResult{Receipt: receipt}, errors.New("external runner failed")
	}
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: runID, Role: agentexec.RoleExecutor,
		InputDigest: receipt.InputDigest, Outcome: agentexec.OutcomeIncomplete,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{},
		VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{"bounded fixture failure"},
		Usage: usage, NativeWork: nativeWork,
	}
	return agentexec.RunResult{Response: response, Receipt: receipt}, nil
}

func (*failedManagerReceiptInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return agentexec.Fingerprint(config)
}

func TestManagerFailurePersistsReturnedReceiptExactlyOnce(t *testing.T) {
	for _, mode := range []string{"incomplete-native", "process-error", "preflight"} {
		t.Run(mode, func(t *testing.T) {
			root := makeProjectRunFixture(t)
			setupE2EProcess(t, "")
			invoker := &failedManagerReceiptInvoker{mode: mode}
			plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{
				Goal: "Implement the owned orders artifact.", Managers: []string{e2eManagerID("orders", "orders")}, ExecuteAuthorized: true,
			})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			run, runErr := Run(context.Background(), projectworkHost(), invoker, root, plan.ID)
			if runErr == nil || run.Status != StatusFailed {
				t.Fatalf("Manager failure was not terminal: status=%s err=%v", run.Status, runErr)
			}
			if invoker.calls != 1 {
				t.Fatalf("Manager invoked %d times, want one bounded attempt", invoker.calls)
			}
			attempts := 0
			for _, task := range run.Tasks {
				attempts += task.Attempts
			}
			if attempts != 1 {
				t.Fatalf("Manager attempt ledger counted %d starts, want exactly one", attempts)
			}

			wantReceipts := 1
			if mode == "preflight" {
				wantReceipts = 0
			}
			if len(run.Invocations) != wantReceipts {
				t.Fatalf("run retained %d invocation receipts, want %d: %+v", len(run.Invocations), wantReceipts, run.Invocations)
			}
			if wantReceipts == 1 {
				invocation := run.Invocations[0]
				if invocation.Receipt.RunID != "fixture-native-manager-run" || invocation.ReportID != invocation.Receipt.RunID || invocation.CostMicros != 5 {
					t.Fatalf("returned receipt identity or known cost was lost: %+v", invocation)
				}
				if invocation.Receipt.NativeWork == nil || invocation.Receipt.PrivateLogDigest == "" || invocation.Receipt.Usage == nil {
					t.Fatalf("native receipt, usage, or private log binding was lost: %+v", invocation.Receipt)
				}
				if mode == "incomplete-native" && invocation.Receipt.NativeWork.ChangedPaths[0] != "src/orders/implementation.txt" {
					t.Fatalf("incomplete native receipt lost its observed delta paths: %+v", invocation.Receipt.NativeWork)
				}
			}

			store, err := newRunStore(root)
			if err != nil {
				t.Fatal(err)
			}
			persisted, err := store.readLatestState(run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(persisted.Invocations) != wantReceipts {
				t.Fatalf("durable state retained %d invocation receipts, want %d", len(persisted.Invocations), wantReceipts)
			}
			if !reflect.DeepEqual(persisted.Invocations, run.Invocations) {
				t.Fatalf("durable invocation receipt differs from returned report: returned=%+v persisted=%+v", run.Invocations, persisted.Invocations)
			}
			if wantReceipts == 1 && persisted.Invocations[0].Receipt.RunID != "fixture-native-manager-run" {
				t.Fatalf("durable state lost Manager receipt: %+v", persisted.Invocations)
			}
		})
	}
}
