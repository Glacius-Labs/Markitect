package host

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/host/recordstore"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const canonicalEvidenceRefreshVerifierEnv = "MARKITECT_EVIDENCE_REFRESH_TEST_VERIFIER"

func TestCanonicalEvidenceRefreshFakeVerifier(t *testing.T) {
	if os.Getenv(canonicalEvidenceRefreshVerifierEnv) != "1" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil || invocation.Request.Role != agentexec.RoleVerifier {
		os.Exit(41)
	}
	refs := map[string]bool{}
	for _, id := range invocation.Request.ScopeIDs {
		refs[id] = true
	}
	for _, id := range invocation.Request.PolicyIDs {
		refs[id] = true
	}
	for _, artifact := range invocation.Request.Artifacts {
		refs[artifact.Path] = true
	}
	evidenceRefs := make([]string, 0, len(refs))
	observations := make([]agentexec.Observation, 0, len(refs))
	for ref := range refs {
		evidenceRefs = append(evidenceRefs, ref)
		observations = append(observations, agentexec.Observation{Subject: ref, Outcome: agentexec.OutcomePassed, Detail: "fresh test verifier inspected the supplied evidence"})
	}
	sort.Strings(evidenceRefs)
	sort.Slice(observations, func(i, j int) bool { return observations[i].Subject < observations[j].Subject })
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomePassed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: evidenceRefs,
		VerifierObservations: observations, Uncertainty: []string{},
	}
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		os.Exit(42)
	}
	os.Exit(0)
}

func TestSameRefreshSelectedContractNormalizesOnlyGlobalFreshnessFields(t *testing.T) {
	base := canonical.ProjectionRequest{
		Revision: "old-revision", ModelDigest: "old-model", RequestDigest: "old-request",
		Projection:       core.Definition{APIVersion: "markitect.foundation/v1", Kind: "Projection", Metadata: core.Metadata{Name: "billing", Namespace: "example"}, Purpose: "same", Source: core.Source{Path: "projection.yaml", Digest: "file-digest"}},
		Binding:          canonical.ProjectionBinding{Module: "billing-module"},
		ModulePin:        canonical.Pin{Name: "billing-module", Version: "1.0.0", Digest: "module-digest"},
		Projector:        canonical.ProjectorRegistration{ID: "dotnet", Version: "v1", Target: "dotnet", AllowedRoots: []string{"src/billing"}},
		Definitions:      []core.Definition{{APIVersion: "example/v1", Kind: "UseCase", Metadata: core.Metadata{Name: "invoice", Namespace: "billing"}, Purpose: "issue invoice", Spec: map[string]any{"status": "open"}, Source: core.Source{Path: "billing.yaml", Digest: "billing-file"}}},
		Schemas:          []core.Schema{{APIVersion: "example/v1", Purpose: "billing", Source: core.Source{Path: "schema.yaml", Digest: "schema-file"}}},
		Policies:         []core.Definition{{APIVersion: "markitect.foundation/v1", Kind: "ProjectionPolicy", Metadata: core.Metadata{Name: "billing-policy", Namespace: "example"}, Purpose: "same policy", Source: core.Source{Path: "policy.yaml", Digest: "policy-file"}}},
		TargetRepository: ".", TargetPath: "src/billing", TargetPrefix: "src/billing",
		TargetFiles: map[string][]byte{"src/billing/invoice.cs": []byte("old")}, TargetDigests: map[string]string{"src/billing/invoice.cs": "old-digest"},
	}
	current := base
	current.Revision, current.ModelDigest, current.RequestDigest = "new-revision", "new-model", "new-request"
	current.TargetFiles = map[string][]byte{"src/billing/invoice.cs": []byte("new")}
	current.TargetDigests = map[string]string{"src/billing/invoice.cs": "new-digest"}
	if !sameRefreshSelectedContract(base, current) {
		t.Fatal("global freshness and target evidence changes should not alter selected canonical contract")
	}
	changed := current
	changed.Definitions = append([]core.Definition(nil), current.Definitions...)
	changed.Definitions[0].Purpose = "different meaning"
	if sameRefreshSelectedContract(base, changed) {
		t.Fatal("selected Definition change was normalized away")
	}
	changed = current
	changed.Policies = append([]core.Definition(nil), current.Policies...)
	changed.Policies[0].Purpose = "different rule"
	if sameRefreshSelectedContract(base, changed) {
		t.Fatal("selected Policy change was normalized away")
	}
	changed = current
	changed.ModulePin.Digest = "other-module-digest"
	if sameRefreshSelectedContract(base, changed) {
		t.Fatal("Module binding change was normalized away")
	}
}

func TestCanonicalEvidenceRefreshCheckInputsAreProjectionScoped(t *testing.T) {
	cfg := CanonicalControllerConfig{
		CheckInputs: []string{"checks/common.go"},
		AssuranceScopes: []CanonicalAssuranceScope{
			{ProjectionID: "billing", CheckInputs: []string{"checks/billing.go"}},
			{ProjectionID: "orders", CheckInputs: []string{"checks/orders.go"}},
		},
	}
	got := canonicalEvidenceRefreshCheckInputs(cfg, []string{"billing"})
	if !equalStringSets(got, []string{"checks/common.go", "checks/billing.go"}) {
		t.Fatalf("selected refresh check inputs = %#v", got)
	}
}

func TestSameRefreshRelatedContextRejectsChangedDepthTwoGrandchild(t *testing.T) {
	selected := core.Definition{APIVersion: "example/v1", Kind: "Parent", Metadata: core.Metadata{Name: "parent", Namespace: "billing"}}
	child := core.Definition{APIVersion: "example/v1", Kind: "Child", Metadata: core.Metadata{Name: "invoice", Namespace: "billing"}, Purpose: "stable child"}
	grandchild := core.Definition{APIVersion: "example/v1", Kind: "Rule", Metadata: core.Metadata{Name: "retention", Namespace: "billing"}, Purpose: "stable grandchild"}
	first := core.Edge{From: selected.Identity().Key(), To: child.Identity().Key(), Property: "related"}
	second := core.Edge{From: child.Identity().Key(), To: grandchild.Identity().Key(), Property: "governedBy"}
	oldContext := CanonicalAgentContext{
		Definitions:        []core.Definition{selected},
		RelatedDefinitions: []core.Definition{child, grandchild},
		Schemas:            []core.Schema{{APIVersion: "example/v1", Purpose: "stable schema"}},
		Inclusions: []CanonicalContextInclusion{
			{Identity: child.Identity().Key(), Reason: "declared-outgoing-reference-context-only", Via: &first},
			{Identity: grandchild.Identity().Key(), Reason: "declared-outgoing-reference-context-only", Via: &second},
		},
	}
	if !sameRefreshRelatedContext(oldContext, oldContext) {
		t.Fatal("identical depth-two supplied context should remain refreshable")
	}
	changedGrandchild := oldContext
	changedGrandchild.RelatedDefinitions = append([]core.Definition(nil), oldContext.RelatedDefinitions...)
	changedGrandchild.RelatedDefinitions[1].Purpose = "changed grandchild meaning"
	if sameRefreshRelatedContext(oldContext, changedGrandchild) {
		t.Fatal("changed depth-two grandchild was ignored because the intermediate reference stayed fixed")
	}
	changedSchema := oldContext
	changedSchema.Schemas = append([]core.Schema(nil), oldContext.Schemas...)
	changedSchema.Schemas[0].Purpose = "changed related Kind contract"
	if sameRefreshRelatedContext(oldContext, changedSchema) {
		t.Fatal("changed schema in the supplied bounded context was ignored")
	}
}

func TestCanonicalEvidenceRefreshRequiresCurrentSourceHead(t *testing.T) {
	root := "../.."
	head, err := source.GitOutput(root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil {
		t.Fatal(err)
	}
	if err := canonicalEvidenceRefreshHeadMatches(root, strings.TrimSpace(string(head))); err != nil {
		t.Fatalf("current source HEAD was rejected: %v", err)
	}
	if err := canonicalEvidenceRefreshHeadMatches(root, strings.Repeat("0", 40)); err == nil {
		t.Fatal("refresh accepted a reviewed source revision different from current HEAD")
	}
}

func TestCanonicalEvidenceRefreshRetainsArtifactsAndRequiresFreshReview(t *testing.T) {
	root, _, _ := scopedCanonicalFixture(t)
	projectionFile := "examples/canonical-projection/definitions/commerce.markdown-projection.yaml"
	projectionBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(projectionFile)))
	if err != nil {
		t.Fatal(err)
	}
	projectionText := string(projectionBytes)
	for _, reference := range []string{
		"      - apiVersion: commerce.example.org/v1\n        kind: Handler\n        namespace: commerce\n        name: create-order-handler\n",
		"      - apiVersion: commerce.example.org/v1\n        kind: EffectAxis\n        namespace: commerce\n        name: create-order-effects\n",
	} {
		projectionText = strings.Replace(projectionText, reference, "", 1)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(projectionFile)), []byte(projectionText), 0644); err != nil {
		t.Fatal(err)
	}
	schemaFile := "examples/canonical-projection/modules/commerce/schema.yaml"
	schemaBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(schemaFile)))
	if err != nil {
		t.Fatal(err)
	}
	schemaText := strings.Replace(string(schemaBytes), "  Handler:\n    purpose: Describes one application component responsible for handling a use case.\n    properties: {}\n", "  Handler:\n    purpose: Describes one application component responsible for handling a use case.\n    properties:\n      effectAxis:\n        purpose: Identifies the effect contract referenced by this handler.\n        type: reference\n        minCount: 1\n        maxCount: 1\n        target:\n          apiVersion: commerce.example.org/v1\n          kind: EffectAxis\n", 1)
	if schemaText == string(schemaBytes) {
		t.Fatal("depth-two fixture schema did not change")
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(schemaFile)), []byte(schemaText), 0644); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "examples/canonical-projection/modules/commerce/module.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	moduleDigest, err := canonical.DigestPackage(canonical.ModulePackage{ManifestBytes: manifest, Files: map[string][]byte{"schema.yaml": []byte(schemaText)}})
	if err != nil {
		t.Fatal(err)
	}
	configFile := "examples/canonical-projection/canonical.yaml"
	configBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(configFile)))
	if err != nil {
		t.Fatal(err)
	}
	configText := strings.Replace(string(configBytes), "sha256:586af5b08ee218c51f43bfed7e083a293d60b2674bcfad3b14354164faa0668f", moduleDigest, 1)
	configText = strings.Replace(configText, "checks:\n  - name: canonical-projection-fixture\n    run: [go, run, examples/canonical-projection/evidence/check.go]\n", "checks:\n  - name: canonical-projection-fixture\n    run: [go, version]\n", 1)
	if configText == string(configBytes) {
		t.Fatal("depth-two fixture did not update the exact commerce Module pin")
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(configFile)), []byte(configText), 0644); err != nil {
		t.Fatal(err)
	}
	handlerFile := "examples/canonical-projection/definitions/create-order.handler.yaml"
	handlerBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(handlerFile)))
	if err != nil {
		t.Fatal(err)
	}
	handlerText := strings.Replace(string(handlerBytes), "spec: {}\n", "spec:\n  effectAxis:\n    apiVersion: commerce.example.org/v1\n    kind: EffectAxis\n    namespace: commerce\n    name: create-order-effects\n", 1)
	if handlerText == string(handlerBytes) {
		t.Fatal("depth-two fixture Handler did not change")
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(handlerFile)), []byte(handlerText), 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "bound Markdown context through a declared grandchild")
	oldRevision := scopedTestGit(t, root, "rev-parse", "HEAD")
	fixed, err := LoadSelectedCanonicalSource(root, oldRevision, "examples/canonical-projection/canonical.yaml", true)
	if err != nil {
		t.Fatalf("load depth-two source fixture: %v", err)
	}
	if len(fixed.Diagnostics) != 0 {
		t.Fatalf("load depth-two source fixture diagnostics: %v", fixed.Diagnostics)
	}
	projectionID := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "commerce", Name: "application-markdown"}
	working := &snapshot.Snapshot{ID: "working", Provisional: true, Files: cloneByteMap(fixed.Snapshot.Files), Modes: cloneStringMap(fixed.Snapshot.Modes)}
	toolDigest := sha256Prefix(sha256Hex([]byte("refresh-integration-test")))
	prepared, err := PrepareCanonicalProjection(fixed, working, projectionID, "refresh-test/1", toolDigest, nil)
	if err != nil || len(prepared.Outputs) == 0 {
		t.Fatalf("prepare static Markdown target: outputs=%d err=%v", len(prepared.Outputs), err)
	}
	paths := make([]string, 0, len(prepared.Outputs))
	for name, content := range prepared.Outputs {
		paths = append(paths, name)
		scopedTestWrite(t, root, name, string(content))
	}
	sort.Strings(paths)
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "materialize retained Markdown artifacts")

	observedPaths := append(canonicalSourcePaths(fixed), paths...)
	observedPaths = sortedUniquePaths(observedPaths)
	observed, err := source.ObserveSelectedWorking(root, observedPaths)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = PrepareCanonicalProjection(fixed, observed.Snapshot, projectionID, "refresh-test/1", toolDigest, nil)
	if err != nil {
		t.Fatal(err)
	}
	prior, err := buildCanonicalProjectionRecord(prepared, observed.Snapshot, paths, records.StateMaterializedUnverified)
	if err != nil {
		t.Fatal(err)
	}
	externalParent := os.TempDir()
	if runtime.GOOS == "windows" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		externalParent = filepath.Join(home, "AppData", "Local")
	}
	externalParent, err = filepath.EvalSymlinks(externalParent)
	if err != nil {
		t.Fatal(err)
	}
	externalParent, err = realDirectory(externalParent)
	if err != nil {
		t.Fatal(err)
	}
	external, err := os.MkdirTemp(externalParent, "markitect-refresh-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(external) })
	storeDir := filepath.Join(external, "records")
	store, err := recordstore.Initialize(storeDir, []string{})
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.AppendAttempt(state.Head, prior)
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.SelectActive(state.Head, []string{prior.ID})
	if err != nil {
		t.Fatal(err)
	}
	oldPass, err := records.NewVerificationResult(records.VerificationResult{
		RecordID: prior.ID, Revision: prior.Revision, ModelDigest: prior.ModelDigest,
		TargetSnapshotDigest: prior.TargetSnapshotDigest,
		Verifier:             records.VerifierIdentity{ID: "verifier", Version: "1", Digest: toolDigest},
		Checks:               []records.CheckResult{{ID: "old-check", Version: "1", Digest: toolDigest, Outcome: records.OutcomePassed}},
		Outcome:              records.OutcomePassed,
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.AppendVerification(state.Head, oldPass)
	if err != nil {
		t.Fatal(err)
	}

	otherID := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "commerce", Name: "application-dotnet"}.Key()
	otherArtifact := records.ArtifactFact{Path: "unrelated/Billing/note.txt", Role: records.RoleProjectionTarget, Digest: sha256Prefix(sha256Hex([]byte("unique content outside source and declared targets\n"))), Mode: snapshot.RegularMode}
	otherTargetDigest, err := records.TargetSnapshotDigest([]records.ArtifactFact{otherArtifact})
	if err != nil {
		t.Fatal(err)
	}
	other, err := records.NewProjectionRecord(records.ProjectionRecord{
		Revision: oldRevision, ModelDigest: fixed.Model.Digest, PlanDigest: toolDigest,
		InputSnapshotDigest: toolDigest, RequestDigest: toolDigest, ProjectionID: otherID,
		Module:    records.ModuleIdentity{Name: "other", Version: "1", Digest: toolDigest},
		Projector: records.ProjectorIdentity{ID: "other", Version: "1"}, ScopeIDs: []string{"other"},
		Artifacts:            []records.Artifact{{Path: otherArtifact.Path, Digest: otherArtifact.Digest, Mode: otherArtifact.Mode, Change: records.ChangeRetained}},
		TargetSnapshotDigest: otherTargetDigest, State: records.StateMaterializedUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.AppendAttempt(state.Head, other)
	if err != nil {
		t.Fatal(err)
	}
	activeIDs := []string{prior.ID, other.ID}
	sort.Strings(activeIDs)
	state, err = store.SelectActive(state.Head, activeIDs)
	if err != nil {
		t.Fatal(err)
	}

	projectionPath := filepath.Join(root, "examples/canonical-projection/definitions/commerce.projection.yaml")
	projectionSource, err := os.ReadFile(projectionPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(projectionSource), "Materializes the explicitly selected commerce application scope in the .NET source tree.", "Unrelated global Projection purpose changed.", 1)
	if changed == string(projectionSource) {
		t.Fatal("unrelated Projection fixture did not change")
	}
	if err := os.WriteFile(projectionPath, []byte(changed), 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "change unrelated canonical Projection")
	effectFile := "examples/canonical-projection/definitions/create-order.effect-axis.yaml"
	effectPath := filepath.Join(root, filepath.FromSlash(effectFile))
	effectOriginal, err := os.ReadFile(effectPath)
	if err != nil {
		t.Fatal(err)
	}
	effectChanged := strings.Replace(string(effectOriginal), "Keeps externally visible order effects behind the application boundary.", "Changed depth-two grandchild semantics.", 1)
	if effectChanged == string(effectOriginal) {
		t.Fatal("depth-two grandchild fixture did not change")
	}
	if err := os.WriteFile(effectPath, []byte(effectChanged), 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "change bounded depth-two grandchild")
	depthTwoRevision := scopedTestGit(t, root, "rev-parse", "HEAD")
	blocked, err := ProposeCanonicalEvidenceRefresh(root, depthTwoRevision, depthTwoRevision, "examples/canonical-projection/canonical.yaml", CanonicalControllerConfig{
		APIVersion: CanonicalControllerAPIVersion, RecordStore: storeDir, PrivateLogs: filepath.Join(external, "logs"), ReferenceDepth: 2,
		Executor: CanonicalRunnerConfig{Command: "go", ProviderVersion: "test", TimeoutSeconds: 10},
		Verifier: CanonicalRunnerConfig{Command: "go", ProviderVersion: "test", TimeoutSeconds: 10},
	}, []string{projectionID.Key()})
	if err != nil || blocked.Status != "blocked" || len(blocked.Findings) != 1 || blocked.Findings[0].Code != "related-context.changed" {
		t.Fatalf("changed depth-two grandchild was not conservatively blocked: proposal=%+v err=%v", blocked, err)
	}
	if err := os.WriteFile(effectPath, effectOriginal, 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "restore retained depth-two context")
	currentRevision := scopedTestGit(t, root, "rev-parse", "HEAD")
	cfg := CanonicalControllerConfig{
		APIVersion: CanonicalControllerAPIVersion, RecordStore: storeDir, PrivateLogs: filepath.Join(external, "logs"), ReferenceDepth: 2,
		Executor: CanonicalRunnerConfig{Command: "go", ProviderVersion: "test", TimeoutSeconds: 10},
		Verifier: CanonicalRunnerConfig{Command: os.Args[0], Args: []string{"-test.run=^TestCanonicalEvidenceRefreshFakeVerifier$"}, Model: "refresh-test-verifier", ProviderVersion: "test", TimeoutSeconds: 10, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20},
	}
	configPath := "examples/canonical-projection/canonical.yaml"
	proposal, err := ProposeCanonicalEvidenceRefresh(root, currentRevision, currentRevision, configPath, cfg, []string{projectionID.Key()})
	if err != nil || proposal.Status != "planned" {
		t.Fatalf("preview retained evidence refresh: status=%s findings=%+v err=%v", proposal.Status, proposal.Findings, err)
	}
	if !equalStringSets(proposal.UnselectedStaleProjectionIDs, []string{otherID}) {
		t.Fatalf("unselected stale scopes were not separately reported: %#v", proposal.UnselectedStaleProjectionIDs)
	}
	if len(proposal.Items) != 1 || proposal.Items[0].Record.State != records.StateMaterializedUnverified || proposal.Items[0].Record.PriorRecordID != prior.ID {
		t.Fatalf("refresh proposal did not create a new linked unverified record: %#v", proposal.Items)
	}

	beforeHead := state.Head
	tampered := proposal
	tampered.Items = append([]CanonicalEvidenceRefreshItem(nil), proposal.Items...)
	tampered.Items[0].PriorRecordID = "different-prior"
	if _, err := ApplyCanonicalEvidenceRefresh(root, configPath, cfg, tampered, proposal.Digest, true); err == nil {
		t.Fatal("Apply accepted changed reviewed items with the original proposal digest")
	}
	if _, err := ApplyCanonicalEvidenceRefresh(root, configPath, cfg, proposal, proposal.Digest, false); err == nil {
		t.Fatal("Apply accepted without explicit write")
	}
	state, err = store.Read()
	if err != nil || state.Head != beforeHead {
		t.Fatalf("refused refresh changed ledger: state=%s err=%v", state.Head, err)
	}

	// Moving the source branch after review requires a new current-bound preview.
	scopedTestWrite(t, root, "unrelated/HEAD-advance.txt", "advance branch without changing selected scope\n")
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "advance source HEAD after review")
	if _, err := ApplyCanonicalEvidenceRefresh(root, configPath, cfg, proposal, proposal.Digest, true); err == nil {
		t.Fatal("Apply accepted a preview bound to a prior source HEAD")
	}
	currentRevision = scopedTestGit(t, root, "rev-parse", "HEAD")
	proposal, err = ProposeCanonicalEvidenceRefresh(root, currentRevision, currentRevision, configPath, cfg, []string{projectionID.Key()})
	if err != nil || proposal.Status != "planned" {
		t.Fatalf("fresh preview after HEAD advance: %s %v", proposal.Status, err)
	}
	if !equalStringSets(proposal.UnselectedStaleProjectionIDs, []string{otherID}) {
		t.Fatalf("partial refresh lost unselected stale record notice: %#v", proposal.UnselectedStaleProjectionIDs)
	}

	// A ledger append after review invalidates the preview even if selected bytes stay fixed.
	noise, err := records.NewProjectionRecord(records.ProjectionRecord{
		Revision: currentRevision, ModelDigest: fixed.Model.Digest, PlanDigest: sha256Prefix(sha256Hex([]byte("noise-plan"))),
		InputSnapshotDigest: toolDigest, RequestDigest: toolDigest, ProjectionID: "markitect.foundation/v1:Projection:noise",
		Module: records.ModuleIdentity{Name: "noise", Version: "1", Digest: toolDigest}, Projector: records.ProjectorIdentity{ID: "noise", Version: "1"},
		ScopeIDs: []string{"noise"}, Artifacts: []records.Artifact{{Path: "noise/file.txt", Digest: toolDigest, Mode: snapshot.RegularMode, Change: records.ChangeCreated}},
		State: records.StateMaterializedUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.AppendAttempt(state.Head, noise)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyCanonicalEvidenceRefresh(root, configPath, cfg, proposal, proposal.Digest, true); err == nil {
		t.Fatal("Apply accepted a stale ledger-bound preview")
	}
	proposal, err = ProposeCanonicalEvidenceRefresh(root, currentRevision, currentRevision, configPath, cfg, []string{projectionID.Key()})
	if err != nil || proposal.Status != "planned" {
		t.Fatalf("fresh preview after unrelated ledger append: %s %v", proposal.Status, err)
	}
	beforeApplyHead := scopedTestGit(t, root, "rev-parse", "HEAD")
	beforeApplyIndex := scopedTestGit(t, root, "diff", "--cached", "--binary")
	beforeApplyStatus := scopedTestGit(t, root, "status", "--porcelain")
	beforeTargets, err := source.ObserveSelectedWorking(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	apply, err := ApplyCanonicalEvidenceRefresh(root, configPath, cfg, proposal, proposal.Digest, true)
	if err != nil || apply.Status != "refreshed" || len(apply.Records) != 1 {
		t.Fatalf("Apply retained evidence refresh: report=%+v err=%v", apply, err)
	}
	afterTargets, err := source.ObserveSelectedWorking(root, paths)
	if err != nil || scopedTestGit(t, root, "rev-parse", "HEAD") != beforeApplyHead || scopedTestGit(t, root, "diff", "--cached", "--binary") != beforeApplyIndex || scopedTestGit(t, root, "status", "--porcelain") != beforeApplyStatus || afterTargets.Snapshot.Digest() != beforeTargets.Snapshot.Digest() {
		t.Fatalf("retained refresh mutated source or target evidence: err=%v", err)
	}
	state, err = store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Verifications) != 1 || state.Verifications[0].ID != oldPass.ID || state.Verifications[0].RecordID != prior.ID {
		t.Fatalf("refresh changed old verification history: %#v", state.Verifications)
	}
	byRecordID := map[string]records.ProjectionRecord{}
	for _, record := range state.Records {
		byRecordID[record.ID] = record
	}
	active := make([]records.ProjectionRecord, 0, len(state.ActiveSelection.RecordIDs))
	for _, id := range state.ActiveSelection.RecordIDs {
		active = append(active, byRecordID[id])
	}
	if len(active) != 2 {
		t.Fatalf("active records after partial refresh: %#v", active)
	}
	for _, record := range active {
		if record.ProjectionID == projectionID.Key() && (record.ID == prior.ID || record.State != records.StateMaterializedUnverified) {
			t.Fatalf("old PASS was promoted or old record left active: %#v", record)
		}
	}
	if _, err := VerifyCanonicalController(context.Background(), root, currentRevision, currentRevision, configPath, cfg, false); err == nil {
		t.Fatal("strict Verify accepted the remaining active stale scope")
	}
	newRecord := apply.Records[0]
	state, err = store.SelectActive(state.Head, []string{newRecord.ID})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(canonicalEvidenceRefreshVerifierEnv, "1")
	verification, err := VerifyCanonicalController(context.Background(), root, currentRevision, currentRevision, configPath, cfg, true)
	if err != nil || verification.Outcome != records.OutcomePassed || len(verification.Results) != 1 {
		t.Fatalf("fresh Verify after retained refresh: outcome=%s results=%#v err=%v", verification.Outcome, verification.Results, err)
	}
	if verification.Results[0].RecordID != newRecord.ID || verification.Results[0].RecordID == prior.ID {
		t.Fatalf("fresh verification did not bind the new retained record: %#v", verification.Results[0])
	}
	state, err = store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Verifications) != 2 || state.Verifications[0].RecordID != prior.ID || state.Verifications[1].RecordID != newRecord.ID {
		t.Fatalf("fresh verification changed old result or failed to append a new result: %#v", state.Verifications)
	}
	for _, name := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil || !bytes.Equal(data, prepared.Outputs[name]) {
			t.Fatalf("refresh changed retained target %s: err=%v", name, err)
		}
	}
}
