package projectbriefing

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestEnsureAcceptedHistoryBootstrapsAndAutomaticallyRecordsCommittedChanges(t *testing.T) {
	root, baseline, changed := committedModelFixture(t)
	receipt, err := EnsureAcceptedHistory(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.BaselineRevision != baseline || receipt.Revision != changed || len(receipt.Bundles) != 1 {
		t.Fatalf("ensure receipt = %#v", receipt)
	}
	bundle := receipt.Bundles[0]
	if bundle.SinceRevision != baseline || bundle.Revision != changed || len(bundle.Events) == 0 {
		t.Fatalf("automatic briefing did not bind adjacent accepted revisions: %#v", bundle)
	}
	if bundle.Provenance.DecisionReference != "git-commit:"+changed || !strings.HasPrefix(bundle.Provenance.Actor, "git-commit-author:") || !strings.Contains(bundle.Provenance.Authority, "Git identity is not authenticated") {
		t.Fatalf("automatic provenance overclaims or omits source: %#v", bundle.Provenance)
	}
	state, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if digest != receipt.StoreDigest || state.History == nil || state.History.BaselineRevision != baseline || state.History.Revision != changed || len(state.Briefings) != 1 {
		t.Fatalf("persisted history = %#v, digest=%s", state, digest)
	}
	again, err := EnsureAcceptedHistory(root, changed)
	if err != nil || again.StoreDigest != receipt.StoreDigest || len(again.Bundles) != 0 {
		t.Fatalf("idempotent ensure = %#v err=%v", again, err)
	}
	project, err := projectwork.Load(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	manager := bundle.Events[0].AffectedManagers[0]
	briefings, events, _, err := LoadForManager(root, project.Model.Digest, manager, changed)
	if err != nil || len(briefings) != 1 || len(events) != 1 {
		t.Fatalf("ensured manager history briefings=%d events=%d err=%v", len(briefings), len(events), err)
	}
}

func TestEnsureAcceptedHistoryAdvancesCodeOnlyAndIgnoresWorkingDraft(t *testing.T) {
	root, baseline, changed := committedModelFixture(t)
	receipt, err := EnsureAcceptedHistory(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	original, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	draft := strings.Replace(string(original), "before shipment", "before any dispatch", 1)
	if draft == string(original) {
		t.Fatal("could not create model draft")
	}
	if err := os.WriteFile(modelPath, []byte(draft), 0644); err != nil {
		t.Fatal(err)
	}
	draftReceipt, err := EnsureAcceptedHistory(root, changed)
	if err != nil || draftReceipt.StoreDigest != receipt.StoreDigest || len(draftReceipt.Bundles) != 0 {
		t.Fatalf("working draft changed accepted history receipt=%#v err=%v", draftReceipt, err)
	}
	if err := os.WriteFile(modelPath, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("Committed code-only change.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitCommitTest(t, root, "documentation only")
	codeOnly := gitOutputTest(t, root, "rev-parse", "HEAD")
	advanced, err := EnsureAcceptedHistory(root, codeOnly)
	if err != nil || len(advanced.Bundles) != 0 || advanced.BaselineRevision != baseline || advanced.Revision != codeOnly {
		t.Fatalf("code-only commit did not advance cursor without event receipt=%#v err=%v", advanced, err)
	}
	state, _, err := Read(root)
	if err != nil || state.History == nil || state.History.Revision != codeOnly || len(state.Briefings) != 1 {
		t.Fatalf("code-only cursor state=%#v err=%v", state, err)
	}
}

func TestEnsureAcceptedHistoryPreservesRevertAndRejectsManualAggregateWrite(t *testing.T) {
	root, baseline, changed := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	original, err := sourceFileAtRevision(t, root, baseline, ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, original, 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "revert accepted change")
	reverted := gitOutputTest(t, root, "rev-parse", "HEAD")
	receipt, err := EnsureAcceptedHistory(root, reverted)
	if err != nil || len(receipt.Bundles) != 1 || receipt.Bundles[0].SinceRevision != changed || receipt.Bundles[0].Revision != reverted {
		t.Fatalf("revert was not recorded as its own adjacent event: receipt=%#v err=%v", receipt, err)
	}
	state, digest, err := Read(root)
	if err != nil || len(state.Briefings) != 2 {
		t.Fatalf("history did not retain both transitions: %#v err=%v", state, err)
	}
	aggregate, err := Generate(root, baseline, reverted, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, aggregate, digest); !errors.Is(err, ErrAmbiguousHistory) && !errors.Is(err, ErrStaleModel) {
		t.Fatalf("manual aggregate write crossed accepted cursor: %v", err)
	}
}

func TestEnsureAcceptedHistoryRequiresCommittedCompleteValidActiveHistory(t *testing.T) {
	root, _, revision := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, strings.Repeat("a", 40)); !errors.Is(err, ErrUncommittedModel) {
		t.Fatalf("non-HEAD revision error = %v", err)
	}
	path := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	if err := os.WriteFile(path, []byte("not: [valid"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "invalid committed model")
	invalid := gitOutputTest(t, root, "rev-parse", "HEAD")
	if _, err := EnsureAcceptedHistory(root, invalid); err == nil || errors.Is(err, ErrNoAcceptedModel) {
		t.Fatalf("invalid accepted intermediate model was not rejected: %v", err)
	}
	historical, err := EnsureAcceptedHistory(root, revision)
	if err != nil || historical.Revision != revision {
		t.Fatalf("valid selected ancestor on the active branch was rejected: receipt=%#v err=%v", historical, err)
	}
}

func TestEnsureAcceptedHistoryRejectsBrokenStatementReferencesAtBaselineAndIntermediate(t *testing.T) {
	setMissingStatementReference := func(t *testing.T, root string) {
		t.Helper()
		path := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(content), "  requires: []", "  requires:\n    - namespace: commerce.sales.orders\n      name: missing-statement", 1)
		if updated == string(content) {
			t.Fatal("could not add missing Statement reference")
		}
		if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("baseline", func(t *testing.T) {
		root, baseline, _ := committedModelFixture(t)
		gitTest(t, root, "checkout", "--orphan", "feature-invalid-baseline")
		setMissingStatementReference(t, root)
		gitTest(t, root, "add", ".")
		gitCommitTest(t, root, "invalid first project model")
		revision := gitOutputTest(t, root, "rev-parse", "HEAD")
		if _, err := EnsureAcceptedHistory(root, revision); err == nil {
			t.Fatalf("broken baseline model was accepted (prior valid revision %s): %v", baseline, err)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
			t.Fatalf("invalid baseline created accepted history: %v", err)
		}
	})

	t.Run("intermediate", func(t *testing.T) {
		root, _, _ := committedModelFixture(t)
		setMissingStatementReference(t, root)
		gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
		gitCommitTest(t, root, "invalid committed reference")
		revision := gitOutputTest(t, root, "rev-parse", "HEAD")
		if _, err := EnsureAcceptedHistory(root, revision); err == nil {
			t.Fatalf("broken intermediate model was accepted: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
			t.Fatalf("invalid intermediate created accepted history: %v", err)
		}
	})
}

func TestEnsureAcceptedHistoryRejectsAnalyzedStructuralErrorsButAllowsIncompleteFindings(t *testing.T) {
	root, _, _ := committedModelFixture(t)
	artifactPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "artifacts.yaml")
	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	withMissingImplementation := strings.Replace(string(artifact), "src/shop/orders/", "src/shop/not-yet-implemented-orders/", 1)
	if withMissingImplementation == string(artifact) {
		t.Fatal("could not create expected incomplete implementation path")
	}
	if err := os.WriteFile(artifactPath, []byte(withMissingImplementation), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/artifacts.yaml")
	gitCommitTest(t, root, "expected implementation gap")
	incompleteRevision := gitOutputTest(t, root, "rev-parse", "HEAD")
	incompleteProject, err := projectwork.Load(root, incompleteRevision)
	if err != nil {
		t.Fatal(err)
	}
	if !hasIncompleteFinding(incompleteProject.Report.Findings) {
		t.Fatalf("fixture no longer covers implementation/coverage incompleteness: %#v", incompleteProject.Report.Findings)
	}
	if _, err := EnsureAcceptedHistory(root, incompleteRevision); err != nil {
		t.Fatalf("incomplete implementation/coverage was treated as an invalid canonical model: %v", err)
	}
	priorState, priorDigest, err := Read(root)
	if err != nil || priorState.History == nil || priorState.History.Revision != incompleteRevision {
		t.Fatalf("incomplete valid model did not establish accepted cursor: state=%#v err=%v", priorState.History, err)
	}
	statementPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "inventory", "reservations", "release-reservation.yaml")
	statement, err := os.ReadFile(statementPath)
	if err != nil {
		t.Fatal(err)
	}
	privateStatement := strings.Replace(string(statement), "  public: true", "  public: false", 1)
	if privateStatement == string(statement) {
		t.Fatal("could not make referenced Statement private")
	}
	if err := os.WriteFile(statementPath, []byte(privateStatement), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/inventory/reservations/release-reservation.yaml")
	gitCommitTest(t, root, "cross-manager private reference")
	revision := gitOutputTest(t, root, "rev-parse", "HEAD")
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatalf("Core-valid model should reach project structural analysis: %v", err)
	}
	if project.Model.Digest == "" || !hasStructuralError(project.Report.Findings) {
		t.Fatalf("fixture did not produce a compiled model with a structural report error: digest=%q findings=%#v", project.Model.Digest, project.Report.Findings)
	}
	if _, err := EnsureAcceptedHistory(root, revision); !errors.Is(err, ErrAmbiguousHistory) {
		t.Fatalf("structurally invalid committed model was accepted: %v", err)
	}
	state, digest, err := Read(root)
	if err != nil || state.History == nil || state.History.Revision != incompleteRevision || digest != priorDigest {
		t.Fatalf("structurally invalid model advanced accepted history: state=%#v digest=%s err=%v", state.History, digest, err)
	}
}

func hasStructuralError(findings []projectmodel.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == "error" {
			return true
		}
	}
	return false
}

func hasIncompleteFinding(findings []projectmodel.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == "incomplete" {
			return true
		}
	}
	return false
}

func TestEnsureAcceptedHistoryRejectsMissingProjectAndShallowAncestry(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "--initial-branch=feature-no-project")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("not a project\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitCommitTest(t, root, "ordinary repository")
	noProject := gitOutputTest(t, root, "rev-parse", "HEAD")
	if _, err := EnsureAcceptedHistory(root, noProject); !errors.Is(err, ErrNoAcceptedModel) {
		t.Fatalf("repository without project model error = %v", err)
	}
	root, _, revision := committedModelFixture(t)
	shallowFile := filepath.Join(root, ".git", "shallow")
	if err := os.WriteFile(shallowFile, []byte(revision+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureAcceptedHistory(root, revision); !errors.Is(err, ErrAmbiguousHistory) {
		t.Fatalf("shallow history error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
		t.Fatalf("incomplete history created an accepted baseline: %v", err)
	}
}

func TestEnsureAcceptedHistoryFirstModelIsBaselineWithoutInventedChange(t *testing.T) {
	root, baseline, _ := committedModelFixture(t)
	gitTest(t, root, "checkout", baseline)
	project, err := projectwork.Load(root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := EnsureAcceptedHistory(root, baseline)
	if err != nil || receipt.BaselineRevision != baseline || receipt.Revision != baseline || len(receipt.Bundles) != 0 {
		t.Fatalf("first committed project model bootstrap receipt=%#v err=%v", receipt, err)
	}
	state, _, err := Read(root)
	if err != nil || len(state.Briefings) != 0 || state.History == nil {
		t.Fatalf("initial model was invented as a change: %#v err=%v", state, err)
	}
	manager := project.Report.Managers[0].ID
	briefings, events, _, err := LoadForManager(root, project.Model.Digest, manager, baseline)
	if err != nil || len(briefings) != 0 || len(events) != 0 {
		t.Fatalf("baseline manager context briefings=%d events=%d err=%v", len(briefings), len(events), err)
	}
}

func TestEnsureAcceptedHistoryInitWaitsForFirstCommittedModel(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "--initial-branch=feature-init-history")
	if _, err := EnsureAcceptedHistory(root, strings.Repeat("a", 40)); !errors.Is(err, ErrUncommittedModel) {
		t.Fatalf("unborn repository error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".markitect"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := projectwork.Init(root, "history-bootstrap", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
		t.Fatalf("uncommitted init unexpectedly seeded accepted history: %v", err)
	}
	gitTest(t, root, "add", ".")
	gitCommitTest(t, root, "initialize project")
	revision := gitOutputTest(t, root, "rev-parse", "HEAD")
	receipt, err := EnsureAcceptedHistory(root, revision)
	if err != nil || receipt.BaselineRevision != revision || receipt.Revision != revision || len(receipt.Bundles) != 0 {
		t.Fatalf("first committed init baseline receipt=%#v err=%v", receipt, err)
	}
}

func TestVerifiedResolutionIsCandidateBoundSeparateFromDismissal(t *testing.T) {
	root, _, revision := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, revision); err != nil {
		t.Fatal(err)
	}
	state, digest, err := Read(root)
	if err != nil || len(state.Briefings) != 1 || len(state.Briefings[0].Events) == 0 {
		t.Fatalf("briefing history state=%#v err=%v", state, err)
	}
	event := state.Briefings[0].Events[0]
	manager := event.AffectedManagers[0]
	digest, err = Dismiss(root, event.ID, manager, digest)
	if err != nil {
		t.Fatal(err)
	}
	state, digest, err = Read(root)
	if err != nil || EventResolutionStatus(state, event.ID).Status != "unresolved" {
		t.Fatalf("dismissal changed resolution status: status=%#v err=%v", EventResolutionStatus(state, event.ID), err)
	}
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validResolutionEvidence(event.ID, project.Report.Managers)
	if _, err := ResolveVerified(root, revision, "stale-model-digest", evidence, digest); !errors.Is(err, ErrStaleModel) {
		t.Fatalf("stale selected model resolved event: %v", err)
	}
	incomplete := evidence
	incomplete.FullVerifyPassed = false
	if _, err := ResolveVerified(root, revision, project.Model.Digest, incomplete, digest); !errors.Is(err, ErrResolution) {
		t.Fatalf("failed full Verify resolved event: %v", err)
	}
	missingManager := evidence
	missingManager.CoveredManagerIDs = missingManager.CoveredManagerIDs[:len(missingManager.CoveredManagerIDs)-1]
	if _, err := ResolveVerified(root, revision, project.Model.Digest, missingManager, digest); !errors.Is(err, ErrResolution) {
		t.Fatalf("partial Manager coverage resolved event: %v", err)
	}
	newDigest, err := ResolveVerified(root, revision, project.Model.Digest, evidence, digest)
	if err != nil {
		t.Fatal(err)
	}
	state, current, err := Read(root)
	if err != nil || current != newDigest || len(state.Dismissals) != 1 || len(state.Resolutions) != 1 {
		t.Fatalf("resolution/dismissal store state=%#v digest=%s err=%v", state, current, err)
	}
	status := EventResolutionStatus(state, event.ID)
	if status.Status != "resolved" || status.Resolution == nil || status.Resolution.Evidence.ApplyDigest != evidence.ApplyDigest {
		t.Fatalf("event resolution status=%#v", status)
	}
	repeatedDigest, err := ResolveVerified(root, revision, project.Model.Digest, evidence, current)
	if err != nil || repeatedDigest != current {
		t.Fatalf("identical resolution was not idempotent: digest=%s err=%v", repeatedDigest, err)
	}
	conflicting := evidence
	conflicting.RunID = "another-run"
	if _, err := ResolveVerified(root, revision, project.Model.Digest, conflicting, current); !errors.Is(err, ErrAlreadyResolved) {
		t.Fatalf("different candidate evidence replaced resolution: %v", err)
	}
	path := filepath.Join(root, filepath.FromSlash(storePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	state.Resolutions[0].Evidence.ApplyDigest = "tampered-apply-digest"
	data, err = json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Read(root); !errors.Is(err, ErrResolution) {
		t.Fatalf("tampered resolution evidence passed store validation: %v", err)
	}
}

func TestVerifiedResolutionRejectsFutureEventAtOldModelRevision(t *testing.T) {
	root, _, firstRevision := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, firstRevision); err != nil {
		t.Fatal(err)
	}
	firstPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	content, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	thirdValue := strings.Replace(string(content), "before shipment", "prior to fulfillment", 1)
	if thirdValue == string(content) {
		t.Fatal("could not create future model revision")
	}
	if err := os.WriteFile(firstPath, []byte(thirdValue), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "later model change")
	futureRevision := gitOutputTest(t, root, "rev-parse", "HEAD")
	if _, err := EnsureAcceptedHistory(root, futureRevision); err != nil {
		t.Fatal(err)
	}
	state, digest, err := Read(root)
	if err != nil || len(state.Briefings) != 2 {
		t.Fatalf("future history state=%#v err=%v", state, err)
	}
	var futureEvent Event
	for _, bundle := range state.Briefings {
		if bundle.Revision == futureRevision && len(bundle.Events) > 0 {
			futureEvent = bundle.Events[0]
		}
	}
	if futureEvent.ID == "" {
		t.Fatal("future accepted event was not persisted")
	}
	oldProject, err := projectwork.Load(root, firstRevision)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validResolutionEvidence(futureEvent.ID, oldProject.Report.Managers)
	if _, err := ResolveVerified(root, firstRevision, oldProject.Model.Digest, evidence, digest); !errors.Is(err, ErrStaleModel) {
		t.Fatalf("future event was resolved against an older model: %v", err)
	}
}

func validResolutionEvidence(eventID string, managers []projectmodel.Manager) VerifiedResolutionEvidence {
	managerIDs := make([]string, 0, len(managers))
	for _, manager := range managers {
		managerIDs = append(managerIDs, manager.ID)
	}
	sort.Strings(managerIDs)
	return VerifiedResolutionEvidence{
		EventIDs: []string{eventID}, RunID: "run-verified", PlanDigest: "plan-digest",
		CandidateID: "candidate-verified", CandidateDigest: "candidate-digest",
		VerificationDigest: "full-verify-digest", ApplyDigest: "apply-digest", FullVerifyPassed: true,
		CoveredManagerIDs: managerIDs, EvidenceRefs: []string{"check:integration", "verify:all-managers"},
	}
}
