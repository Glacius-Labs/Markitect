package projectrun

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
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
			revision := identityHead(t, root)
			project := processRuntimeProject(t, root, revision, false)
			host := processRuntimeHost(root, project)
			plan, err := Plan(host, root, revision, PlanRequest{
				Goal: "Implement the owned orders artifact.", Managers: []string{e2eManagerID("orders", "orders")}, ExecuteAuthorized: true,
			})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			run, runErr := Run(context.Background(), host, invoker, root, plan.ID)
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
			receiptCount := 0
			for _, invocation := range run.Invocations {
				if invocation.Receipt.RunID != "" {
					receiptCount++
				}
			}
			if receiptCount != wantReceipts {
				t.Fatalf("run retained %d returned receipts, want %d", receiptCount, wantReceipts)
			}
			if mode == "preflight" {
				if len(run.Invocations) != 1 || run.Invocations[0].Attempt == nil || run.Invocations[0].Attempt.Status != "uncertain" || run.Invocations[0].Receipt.RunID != "" || run.Invocations[0].CostKnown {
					t.Fatal("preflight lost its charged uncertain original attempt or invented a receipt")
				}
				if len(run.RoleStartReservations) != 1 || run.RoleStartReservations[0].Request.RequestID != run.Invocations[0].Attempt.ID || run.RoleStartReservations[0].Request.State != "unknown" {
					t.Fatal("preflight lost or replaced its original charged reservation")
				}
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
			durableReceiptCount := 0
			for _, invocation := range persisted.Invocations {
				if invocation.Receipt.RunID != "" {
					durableReceiptCount++
				}
			}
			if durableReceiptCount != wantReceipts {
				t.Fatalf("durable state retained %d returned receipts, want %d", durableReceiptCount, wantReceipts)
			}
			if !reflect.DeepEqual(persisted.Invocations, run.Invocations) {
				t.Fatalf("durable invocation receipt differs from returned report: returned=%+v persisted=%+v", run.Invocations, persisted.Invocations)
			}
			if mode == "preflight" {
				if _, err := Resume(context.Background(), host, invoker, root, plan.ID); err == nil {
					t.Fatal("unpriced uncertain preflight automatically replayed")
				}
				after, err := store.readLatestState(run.ID)
				if err != nil {
					t.Fatal(err)
				}
				if invoker.calls != 1 || len(after.RoleStartReservations) != 1 || after.RoleStartReservations[0].Key != run.RoleStartReservations[0].Key || len(after.Invocations) != 1 || after.Invocations[0].Attempt.ID != run.Invocations[0].Attempt.ID {
					t.Fatal("uncertain preflight replayed, replaced or refilled its authority")
				}
			}
			if wantReceipts == 1 && persisted.Invocations[0].Receipt.RunID != "fixture-native-manager-run" {
				t.Fatalf("durable state lost Manager receipt: %+v", persisted.Invocations)
			}
		})
	}
}
