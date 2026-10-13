package projectrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

func stageReviewFixture(t *testing.T) (string, ManagerStageSpec, ManagerStageReviewSpec, *stageStoreInvoker, Host) {
	t.Helper()
	root, spec, invoker, host := stageStoreFixture(t)
	reviewer := spec.Agent
	reviewer.Command = "reviewer-fixture"
	review := projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{"root": reviewer}, MaxRounds: 2, MaxManagerRounds: 3}
	spec.Review = &review
	config, err := reviewer.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	accepted := projectmodel.ManagerContext{Manager: projectmodel.Manager{ID: "accepted-ancestor"}}
	acceptedDigest, _ := digest(accepted)
	reviewSpec := ManagerStageReviewSpec{AcceptedModelDigest: acceptedDigest, Validation: spec.Validation, Agent: reviewer, Config: config, AcceptedModel: accepted, Purpose: "Assess this explicitly selected proposal and its evidence", ProposedContext: json.RawMessage(`{"classification":"proposed-by-parent","managerId":"root"}`), Evidence: []agentexec.Artifact{{Path: "README.md", Mode: "0644", Digest: rawContentDigest([]byte("selected source evidence")), Content: []byte("selected source evidence")}}, Grounding: map[string][]string{"README.md": {"evidence:selected"}}, ChildBindings: json.RawMessage(`{"children":[]}`), SourceRoot: root, TargetRoot: spec.TargetRoot, Limits: spec.Limits, RuntimeLimits: spec.RuntimeLimits, Review: review}
	invoker.reviewRaw = []byte("{ \"status\" : \"pass\", \"summary\":\"exact selected report assessed\", \"findings\" : [] }")
	return root, spec, reviewSpec, invoker, host
}

func TestManagerStageConfiguredReviewSharesAuthorityAndCompletionToken(t *testing.T) {
	root, spec, reviewSpec, invoker, host := stageReviewFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Review.Enabled || preview.Review.ConfigFingerprint == spec.Validation.ConfigFingerprint {
		t.Fatal("explicit independent reviewer binding was not disclosed")
	}
	manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, manager.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "typed"}); err == nil {
		t.Fatal("configured review was bypassed")
	}
	invoker.observe = func() {
		store, _ := newManagerStageStore(root)
		state, err := store.readLatestState(manager.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if len(state.Invocations) != 2 || len(state.RoleStartReservations) != 2 || state.Invocations[1].Attempt.Status != "started" {
			t.Fatal("review did not use the same durable attempt authority")
		}
	}
	review, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, manager.StateDigest)
	if err != nil {
		t.Fatal(err)
	}
	if review.Status != "pass" || review.StateDigest == manager.StateDigest || invoker.calls != 2 {
		t.Fatal("review receipt/state token is not actual durable evidence")
	}
	invoker.observe = nil
	store, _ := newManagerStageStore(root)
	state, err := store.readLatestState(manager.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if state.RoleStartReservations[1].Request.Role != "reviewer" || state.RoleStartReservations[1].Request.State != "completed" {
		t.Fatal("reviewer reservation role/lifecycle was mislabeled")
	}
	if len(state.Invocations) != 2 || len(state.Reviews) != 1 || state.CostMicros != 4 {
		t.Fatal("manager/reviewer were not charged once under one session budget")
	}
	raw, err := readStageEvidence(store, manager.RunID, state.Invocations[1].Attempt.ReportRef)
	if err != nil || !bytes.Equal(raw, invoker.reviewRaw) {
		t.Fatal("review raw report was rewritten")
	}
	input, err := readStageEvidence(store, manager.RunID, state.Invocations[1].Attempt.RequestRef)
	if err != nil {
		t.Fatal(err)
	}
	var request agentexec.Request
	if err := json.Unmarshal(input, &request); err != nil {
		t.Fatal(err)
	}
	var reviewContext reviewerContext
	if err := json.Unmarshal(request.Context, &reviewContext); err != nil {
		t.Fatal(err)
	}
	if reviewContext.AcceptedModel.Manager.ID != "accepted-ancestor" || reviewContext.ManagerID != "root" || reviewContext.SubjectKind != "manager-stage-report" || reviewContext.CandidateDigest != manager.Invocation.Attempt.ReportRef.Digest || len(reviewContext.SourceEvidence) != 1 || bytes.Contains(input, []byte("implementer-transcript")) {
		t.Fatal("review promoted draft intent or lost exact scoped report/evidence")
	}
	// A new caller recovers the same existing review; no reviewer dispatch repeats.
	again, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, review.StateDigest)
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 2 || again.ReviewRecord.Receipt.RunID != review.ReviewRecord.Receipt.RunID {
		t.Fatal("existing review replayed")
	}
	done, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, again.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: rawContentDigest([]byte("validated-draft"))})
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != "accepted" {
		t.Fatal("passing exact review did not permit typed completion")
	}
}

func TestManagerStageReviewReceiptBeforeRecordCrashRecoversWithoutDispatch(t *testing.T) {
	root, spec, reviewSpec, invoker, host := stageReviewFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	validations := 0
	host.ValidateManagerStage = func(ManagerStageValidationBinding) error {
		validations++
		if validations == 2 {
			return errors.New("injected post-receipt validation interruption")
		}
		return nil
	}
	if _, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, manager.StateDigest); err == nil {
		t.Fatal("injected interruption did not stop acceptance")
	}
	store, _ := newManagerStageStore(root)
	state, err := store.readLatestState(manager.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Reviews) != 0 || len(state.Invocations) != 2 || state.Invocations[1].Attempt.ReportRef == nil {
		t.Fatal("review receipt/raw evidence was not retained before acceptance")
	}
	host.ValidateManagerStage = func(ManagerStageValidationBinding) error { return nil }
	recovered, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, state.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 2 || recovered.Status != "pass" {
		t.Fatal("review receipt recovery dispatched another reviewer")
	}
}

func TestManagerStageFailedOrExhaustedReviewNeverAttaches(t *testing.T) {
	t.Run("failure", func(t *testing.T) {
		root, spec, reviewSpec, invoker, host := stageReviewFixture(t)
		invoker.reviewRaw = []byte(`{"status":"fail","summary":"Defect in report grounding","findings":[{"path":"README.md","expectation":"Ground the exact claim","grounding":"evidence:selected"}]}`)
		preview, err := PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
		if err != nil {
			t.Fatal(err)
		}
		review, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, manager.StateDigest)
		if err != nil {
			t.Fatal(err)
		}
		if review.Status != "fail" {
			t.Fatal("failed reviewer was not retained")
		}
		if _, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, review.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "typed"}); err == nil {
			t.Fatal("failed review attached stage")
		}
	})
	t.Run("budget", func(t *testing.T) {
		root, spec, reviewSpec, invoker, host := stageReviewFixture(t)
		spec.Binding.Budget.MaxStarts = 1
		spec.Limits.MaxStarts = 1
		spec.RuntimeLimits.MaxStarts = 1
		reviewSpec.Limits = spec.Limits
		reviewSpec.RuntimeLimits = spec.RuntimeLimits
		preview, err := PreviewManagerStage(invoker, root, spec)
		if err != nil {
			t.Fatal(err)
		}
		manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, manager.StateDigest); err == nil {
			t.Fatal("review bypassed shared exhausted starts")
		}
		if invoker.calls != 1 {
			t.Fatal("exhausted review dispatched")
		}
		if _, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, manager.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "typed"}); err == nil {
			t.Fatal("budget exhaustion waived configured review")
		}
	})
}

func TestManagerStageReviewVerdictTamperRejectedByEvidenceReadback(t *testing.T) {
	root, spec, reviewSpec, invoker, host := stageReviewFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	review, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, manager.StateDigest)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newManagerStageStore(root)
	state, err := store.readLatestState(manager.RunID)
	if err != nil {
		t.Fatal(err)
	}
	state.Reviews[0].Outcome = "fail"
	state.Digest = ""
	state.Digest, _ = digest(state)
	dir, _ := store.runDir(manager.RunID)
	entries, err := os.ReadDir(filepath.Join(dir, "states"))
	if err != nil {
		t.Fatal(err)
	}
	var last string
	for _, entry := range entries {
		if entry.Name() > last {
			last = entry.Name()
		}
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(dir, "states", last), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, review.StateDigest, ManagerStageValidation{Accepted: true, TypedResultDigest: "typed"}); err == nil {
		t.Fatal("rehashed forged review verdict was accepted")
	}
}

func TestManagerStageReviewRejectsChangedTrustedTargetFacts(t *testing.T) {
	for _, change := range []string{"empty-target", "wrong-target", "target-head", "context", "accepted-model", "trusted-root"} {
		t.Run(change, func(t *testing.T) {
			root, spec, reviewSpec, invoker, host := stageReviewFixture(t)
			preview, err := PreviewManagerStage(invoker, root, spec)
			if err != nil {
				t.Fatal(err)
			}
			manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "empty-target":
				reviewSpec.TargetRoot = ""
			case "wrong-target":
				reviewSpec.TargetRoot = t.TempDir()
			case "target-head":
				reviewSpec.Validation.AcceptedTargetHead = "other-head"
			case "context":
				reviewSpec.Validation.ContextDigest = "other-context"
			case "accepted-model":
				reviewSpec.AcceptedModel.Manager.ID = "unaccepted"
				reviewSpec.AcceptedModelDigest, _ = digest(reviewSpec.AcceptedModel)
			case "trusted-root":
				reviewSpec.Validation.TargetRoot = t.TempDir()
			}
			if _, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, manager.StateDigest); err == nil {
				t.Fatal("review admitted changed trusted facts")
			}
			if invoker.calls != 1 {
				t.Fatal("changed review dispatched")
			}
		})
	}
}

func TestManagerStageReviewOverrunCannotRecoverOrAttach(t *testing.T) {
	for _, kind := range []string{"cost", "duration"} {
		t.Run(kind, func(t *testing.T) {
			root, spec, reviewSpec, invoker, host := stageReviewFixture(t)
			if kind == "cost" {
				spec.Binding.Budget.MaxCostMicros = 3
				spec.Limits.MaxCostMicros = 3
				reviewSpec.Limits = spec.Limits
			}
			preview, err := PreviewManagerStage(invoker, root, spec)
			if err != nil {
				t.Fatal(err)
			}
			manager, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "duration" {
				reviewSpec.Config.Timeout = time.Millisecond
				reviewSpec.Limits.MaxDuration = 5 * time.Millisecond
				invoker.observe = func() { time.Sleep(10 * time.Millisecond) }
			}
			if _, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, manager.StateDigest); err == nil {
				t.Fatal("review overrun accepted")
			}
			store, _ := newManagerStageStore(root)
			state, err := store.readLatestState(manager.RunID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ReviewManagerStage(context.Background(), host, invoker, root, manager.RunID, manager.AttemptID, reviewSpec, state.Digest); err == nil {
				t.Fatal("review overrun recovered")
			}
			state, err = store.readLatestState(manager.RunID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := CompleteManagerStage(root, manager.RunID, manager.AttemptID, state.Digest, ManagerStageValidation{Accepted: true, TypedResultDigest: "typed"}); err == nil {
				t.Fatal("review overrun attached")
			}
			if invoker.calls > 2 {
				t.Fatal("review overrun replayed")
			}
		})
	}
}
