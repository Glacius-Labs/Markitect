package projectrun

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

type stageStoreInvoker struct {
	calls     int
	raw       []byte
	reviewRaw []byte
	options   agentexec.RunOptions
	observe   func()
	usage     *agentexec.Usage
}

func (i *stageStoreInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return rawContentDigest([]byte("stage-fixture-fingerprint/" + config.Command + "/" + config.Model)), nil
}
func (i *stageStoreInvoker) Run(_ context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	i.options = options
	if i.observe != nil {
		i.observe()
	}
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	fp, _ := i.Fingerprint(config)
	usage := i.usage
	if usage == nil {
		in, out := int64(1), int64(1)
		usage = &agentexec.Usage{Source: "provider-reported", InputTokens: &in, OutputTokens: &out}
	}
	raw := i.raw
	if bytes.Contains(request.Context, []byte(`"kind":"projectrun-review/v1"`)) && i.reviewRaw != nil {
		raw = i.reviewRaw
	}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest, Role: agentexec.RoleExecutor, Outcome: agentexec.OutcomeProposed, ReportJSON: append(json.RawMessage(nil), raw...), Usage: usage}
	receipt := agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest, ContextDigest: rawContentDigest(request.Context), ConfigDigest: fp, Outcome: agentexec.OutcomeProposed, Usage: usage}
	return agentexec.RunResult{Response: response, Receipt: receipt}, nil
}

func stageStoreFixture(t *testing.T) (string, ManagerStageSpec, *stageStoreInvoker, Host) {
	t.Helper()
	base := testkit.TempDir(t)
	sourceRoot, targetRoot := filepath.Join(base, "source"), filepath.Join(base, "target")
	for _, dir := range []string{sourceRoot, targetRoot} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	invoker := &stageStoreInvoker{raw: []byte("{\n  \"report\" : { \"value\" : 1 },  \"unused\": true\n}")}
	fp, _ := invoker.Fingerprint(agentexec.Config{Command: "fixture", Model: "fixture"})
	budget := ManagerStageBudget{MaxStarts: 4, MaxRetries: 2, MaxDuration: time.Minute, MaxCostMicros: 1000}
	binding := ManagerStageBinding{SessionID: "fixture-session", SourceIdentity: rawContentDigest([]byte("source")), TargetIdentity: rawContentDigest([]byte("target")), SourceRevision: strings.Repeat("a", 40), DiscoveryDigest: rawContentDigest([]byte("discovery")), InitialTargetRevision: strings.Repeat("b", 40), TargetContextDigest: rawContentDigest([]byte("accepted")), Budget: budget}
	binding.SourceRootDigest = rawContentDigest([]byte(sourceRoot))
	binding.TargetRootDigest = rawContentDigest([]byte(targetRoot))
	acceptedDigest, _ := digest(projectmodel.ManagerContext{Manager: projectmodel.Manager{ID: "accepted-ancestor"}})
	validation := ManagerStageValidationBinding{AcceptedModelDigest: acceptedDigest, SourceRoot: sourceRoot, TargetRoot: targetRoot, SessionID: binding.SessionID, SessionDigest: rawContentDigest([]byte("session")), IterationID: "iteration", ManagerID: "root", AgentManagerID: "root", Phase: "propose", AcceptedTargetHead: binding.InitialTargetRevision, ContextDigest: rawContentDigest([]byte("context")), RequestContractDigest: rawContentDigest([]byte("contract")), ConfigFingerprint: fp, RuntimeFileDigest: rawContentDigest([]byte("runtime"))}
	agent := projectconfig.Agent{Command: "fixture", Model: "fixture", ProviderVersion: "fixture/1", Timeout: projectconfig.Duration(time.Second), MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20, Pricing: projectconfig.Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}}
	config := agentexec.Config{Command: agent.Command, Model: agent.Model, ProviderVersion: agent.ProviderVersion, Timeout: time.Second, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20}
	request := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: binding.SourceRevision, ModelDigest: rawContentDigest([]byte("model")), ModulePin: "fixture-module", ProjectionID: "fixture-stage", ScopeIDs: []string{"root"}, PolicyIDs: []string{"report-only"}, Context: json.RawMessage(`{"kind":"projectadoption-manager-proposal/v1","phase":"propose"}`), Artifacts: []agentexec.Artifact{}}
	spec := ManagerStageSpec{SelectedSourceRoot: sourceRoot, TargetRoot: targetRoot, Binding: binding, Validation: validation, Agent: agent, Config: config, Request: request, Limits: budget, RuntimeLimits: projectconfig.Limits{MaxStarts: 4, MaxRetries: 2, MaxDuration: projectconfig.Duration(time.Minute), MaxCostMicros: 1000, MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 2 << 20}}
	return sourceRoot, spec, invoker, Host{ValidateManagerStage: func(ManagerStageValidationBinding) error { return nil }}
}

func TestManagerStageStoresExactReportBeforeAttachAndRecoversWithoutDispatch(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, RunsPath)); !os.IsNotExist(err) {
		t.Fatal("preview initialized Runtime state")
	}
	invoker.observe = func() {
		store, _ := newManagerStageStore(root)
		report, err := store.readLatestState(preview.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Invocations) != 1 || report.Invocations[0].Attempt.Status != "started" || len(report.RoleStartReservations) != 1 {
			t.Fatal("dispatch preceded its durable start/reservation")
		}
	}
	result, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newManagerStageStore(root)
	raw, err := readStageEvidence(store, result.RunID, result.Invocation.Attempt.ReportRef)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, invoker.raw) || !bytes.Equal(result.Result.Response.ReportJSON, invoker.raw) {
		t.Fatal("opaque report bytes were normalized")
	}
	if result.Invocation.CostMicros != 2 || !result.Invocation.CostKnown {
		t.Fatalf("Brownfield separate ceiling rounding changed: %+v", result.Invocation)
	}
	if len(invoker.options.InputRoots) != 2 || invoker.options.Workspace != nil || managerStagePathWithin(invoker.options.PrivateLogDirectory, root) || managerStagePathWithin(invoker.options.PrivateLogDirectory, spec.TargetRoot) {
		t.Fatal("Brownfield did not use its exact read-only audit/private-log policy")
	}
	invoker.observe = nil
	current, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := RunManagerStage(context.Background(), host, invoker, root, spec, current.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 1 || recovered.AttemptID != result.AttemptID || !bytes.Equal(recovered.Result.Response.ReportJSON, invoker.raw) {
		t.Fatal("receipt-before-CAS recovery dispatched or changed raw bytes")
	}
	accepted, err := CompleteManagerStage(root, result.RunID, result.AttemptID, recovered.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: rawContentDigest([]byte("typed-domain-validation"))})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != "accepted" {
		t.Fatal("validated completion not durable")
	}
	if _, err := CompleteManagerStage(root, result.RunID, result.AttemptID, recovered.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "changed"}); err == nil {
		t.Fatal("stale completion digest accepted")
	}
	runs, err := ListRuns(root)
	if err != nil || len(runs) != 0 {
		t.Fatalf("internal stage polluted ordinary runs: %+v %v", runs, err)
	}
	if _, err := Status(root, result.RunID); err == nil {
		t.Fatal("planless stage was presented as an ordinary plan")
	}
}

func TestManagerStageReservationBeforeInvocationLogCrashCannotReplay(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	store, _ := newManagerStageStore(root)
	unlock, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	report, err := initializeManagerStage(store, spec.Binding)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := newManagerStageJournal(store, &report, spec.RuntimeLimits)
	if err != nil {
		t.Fatal(err)
	}
	_, err = journal.budget.ReserveRoot(context.Background(), spec.Validation.IterationID, spec.Validation.Phase, agentexec.RoleStartRequest{RequestID: "original-early-request", ParentSessionID: report.ID, Role: agentexec.RoleExecutor})
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
		t.Fatal("reservation-only uncertainty allowed a replacement attempt")
	}
	after, err := store.readLatestState(report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 0 || len(after.RoleStartReservations) != 1 || len(after.Invocations) != 0 {
		t.Fatal("early crash refilled/redispatched its charged reservation")
	}
}

func TestManagerStageEvidenceTamperBlocksRecovery(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newManagerStageStore(root)
	file, err := stageEvidencePath(store, result.RunID, *result.Invocation.Attempt.ReportRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(`{"tampered":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewManagerStage(invoker, root, spec); err == nil {
		t.Fatal("tampered opaque evidence passed authority readback")
	}
	if invoker.calls != 1 {
		t.Fatal("tamper triggered a provider call")
	}
}

func TestManagerStageNewEvidenceNeverSerializesProviderSecrets(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	secrets := []string{"secret-command-argument", "secret-model-option", "secret-environment-value", "secret-runtime-file-content", "secret-transport-config"}
	spec.Agent.Args = []string{secrets[0]}
	spec.Config.Args = []string{secrets[0]}
	spec.Agent.ModelOptions = map[string]any{"credential": secrets[1]}
	spec.Config.ModelOptions, _ = json.Marshal(spec.Agent.ModelOptions)
	const environmentName = "ARCH15_TEST_SYNTHETIC_CREDENTIAL"
	t.Setenv(environmentName, secrets[2])
	names := []string{environmentName}
	spec.Agent.Environment = names
	spec.Config.EnvironmentAllowlist = &names
	runtimeFile := filepath.Join(testkit.TempDir(t), "runtime.fixture")
	if err := os.WriteFile(runtimeFile, []byte(secrets[3]), 0600); err != nil {
		t.Fatal(err)
	}
	spec.Config.RuntimeFiles = []agentexec.RuntimeFile{{Path: runtimeFile}}
	spec.Agent.RuntimeFiles = spec.Config.RuntimeFiles
	spec.Config.TransportConfig = json.RawMessage(`{"credential":"` + secrets[4] + `"}`)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	public, err := json.Marshal(struct {
		Preview    ManagerStagePreview
		Invocation InvocationLog
	}{preview, result.Invocation})
	if err != nil {
		t.Fatal(err)
	}
	scan := func(data []byte) {
		for _, secret := range secrets {
			if bytes.Contains(data, []byte(secret)) {
				t.Fatal("provider secret entered new evidence projection")
			}
		}
	}
	scan(public)
	if err := filepath.WalkDir(filepath.Join(root, RunsPath), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err == nil {
			scan(data)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestManagerStageBudgetCannotRefillAndUnpricedUsageCannotRetry(t *testing.T) {
	t.Run("ceiling", func(t *testing.T) {
		root, spec, invoker, host := stageStoreFixture(t)
		spec.Binding.Budget.MaxStarts = 1
		spec.Limits.MaxStarts = 1
		spec.RuntimeLimits.MaxStarts = 1
		preview, err := PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err != nil {
			t.Fatal(err)
		}
		spec.Binding.Budget.MaxStarts = 20
		spec.Limits.MaxStarts = 20
		spec.Validation.IterationID = "next-iteration"
		preview, err = PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		if preview.RemainingStarts != 0 {
			t.Fatal("higher runtime refilled first session ceiling")
		}
		if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
			t.Fatal("frozen start ceiling bypassed")
		}
		if invoker.calls != 1 {
			t.Fatal("exhaustion dispatched")
		}
	})
	t.Run("usage", func(t *testing.T) {
		root, spec, invoker, host := stageStoreFixture(t)
		in, out := int64(1), int64(1)
		invoker.usage = &agentexec.Usage{Source: "estimated", InputTokens: &in, OutputTokens: &out}
		preview, err := PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		result, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
		if err == nil {
			t.Fatal("non-provider usage became known Brownfield cost")
		}
		spec.RetryOfAttemptID = result.AttemptID
		preview, err = PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
			t.Fatal("unpriced failure allowed retry")
		}
		if invoker.calls != 1 {
			t.Fatal("unpriced failure replayed")
		}
	})
}

func TestManagerStageTerminalReservationCrashReconcilesSamePermit(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newManagerStageStore(root)
	unlock, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.readLatestState(manager.RunID)
	if err != nil {
		t.Fatal(err)
	}
	// Model the durable boundary immediately before permit.Update, keeping receipt.
	report.RoleStartReservations[0].Request.State = "requested"
	if err := persistState(store, &report); err != nil {
		t.Fatal(err)
	}
	unlock()
	preview, err = PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	after, err := store.readLatestState(manager.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 1 || recovered.AttemptID != manager.AttemptID || len(after.RoleStartReservations) != 1 || after.RoleStartReservations[0].Request.State != "completed" {
		t.Fatal("terminal recovery replaced/recharged or left its original permit pending")
	}
}

func TestManagerStageOverrunCannotRecoverOrComplete(t *testing.T) {
	for _, kind := range []string{"cost", "duration"} {
		t.Run(kind, func(t *testing.T) {
			root, spec, invoker, host := stageStoreFixture(t)
			if kind == "cost" {
				spec.Binding.Budget.MaxCostMicros, spec.Limits.MaxCostMicros = 1, 1
			} else {
				spec.Config.Timeout = time.Millisecond
				spec.Binding.Budget.MaxDuration, spec.Limits.MaxDuration = 2*time.Millisecond, 2*time.Millisecond
				invoker.observe = func() { time.Sleep(10 * time.Millisecond) }
			}
			preview, err := PreviewManagerStage(invoker, root, spec)
			if err != nil {
				t.Fatal(err)
			}
			original, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
			if err == nil || original.Invocation.Attempt == nil || original.Invocation.Attempt.ReportRef == nil {
				t.Fatal("overrun was accepted or lost durable receipt")
			}
			store, _ := newManagerStageStore(root)
			raw, err := readStageEvidence(store, original.RunID, original.Invocation.Attempt.ReportRef)
			if err != nil || !bytes.Equal(raw, invoker.raw) {
				t.Fatal("overrun original evidence was lost")
			}
			preview, err = PreviewManagerStage(invoker, root, spec)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
				t.Fatal("overrun recovered as attachable")
			}
			state, err := store.readLatestState(original.RunID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := CompleteManagerStage(root, original.RunID, original.AttemptID, state.Digest, ManagerStageValidation{Accepted: true, TypedResultDigest: "typed"}); err == nil {
				t.Fatal("overrun completed directly")
			}
			if invoker.calls != 1 {
				t.Fatal("overrun replayed provider")
			}
		})
	}
}

func TestManagerStageAcceptedCompletionIsImmutableAndIdempotent(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	spec.Binding.Budget.MaxCostMicros, spec.Limits.MaxCostMicros = 2, 2
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, manager.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "digest-A"})
	if err != nil {
		t.Fatal(err)
	}
	again, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, accepted.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "digest-A"})
	if err != nil || again.StateDigest != accepted.StateDigest {
		t.Fatal("same completion appended or failed")
	}
	if _, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, again.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "digest-B"}); err == nil {
		t.Fatal("accepted result rewritten")
	}
}

func TestManagerStageTightenedCeilingCannotRefillOrComplete(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	spec.Limits.MaxCostMicros = 1
	preview, err = PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest); err == nil {
		t.Fatal("tightened overrun recovered")
	}
	spec.Limits.MaxCostMicros = 1000
	preview, err = PreviewManagerStage(invoker, root, spec)
	if err != nil || preview.RemainingCostMicros != 0 {
		t.Fatal("tightened budget refilled")
	}
	if _, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, preview.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "typed"}); err == nil {
		t.Fatal("completion bypassed tightened budget")
	}
	if invoker.calls != 1 {
		t.Fatal("tightening replayed")
	}
}

func TestManagerStageStrictReportedUsage(t *testing.T) {
	one, zero, negative, huge := int64(1), int64(0), int64(-1), int64(1<<63-1)
	valid := func() *agentexec.Usage {
		return &agentexec.Usage{Source: "provider-reported", InputTokens: &one, OutputTokens: &one}
	}
	bounds := ManagerStageBounds{InputPriceMicrosPerMillion: 1, OutputPriceMicrosPerMillion: 1}
	cases := []struct {
		name            string
		usage           *agentexec.Usage
		known, overflow bool
	}{
		{"nil", nil, false, false}, {"valid", valid(), true, false},
		{"estimated", &agentexec.Usage{Source: "estimated", InputTokens: &one, OutputTokens: &one}, false, false},
		{"input-missing", &agentexec.Usage{Source: "provider-reported", OutputTokens: &one}, false, false},
		{"output-missing", &agentexec.Usage{Source: "provider-reported", InputTokens: &one}, false, false},
		{"negative", &agentexec.Usage{Source: "provider-reported", InputTokens: &negative, OutputTokens: &one}, false, false},
		{"tools", &agentexec.Usage{Source: "provider-reported", InputTokens: &one, OutputTokens: &one, ToolCalls: &one}, false, false},
		{"zero-tools", &agentexec.Usage{Source: "provider-reported", InputTokens: &one, OutputTokens: &one, ToolCalls: &zero}, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, known, overflow := managerStageCost(c.usage, bounds)
			if known != c.known || overflow != c.overflow {
				t.Fatal("usage trust boundary changed")
			}
		})
	}
	_, known, overflow := managerStageCost(&agentexec.Usage{Source: "provider-reported", InputTokens: &huge, OutputTokens: &huge}, ManagerStageBounds{InputPriceMicrosPerMillion: 1_000_000_000_000, OutputPriceMicrosPerMillion: 1_000_000_000_000})
	if known || !overflow {
		t.Fatal("overflow is attachable")
	}
}
