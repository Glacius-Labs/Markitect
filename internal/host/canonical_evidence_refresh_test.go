package host

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/host/recordstore"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

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

func TestSameRefreshExternalDependenciesRejectsChangedRelatedDefinitionOrSchema(t *testing.T) {
	selected := core.Definition{APIVersion: "example/v1", Kind: "Parent", Metadata: core.Metadata{Name: "parent", Namespace: "billing"}}
	child := core.Definition{APIVersion: "example/v1", Kind: "Child", Metadata: core.Metadata{Name: "invoice", Namespace: "billing"}, Purpose: "stable"}
	edge := core.Edge{From: selected.Identity().Key(), To: child.Identity().Key(), Property: "related"}
	model := core.Model{Definitions: []core.Definition{selected, child}, Schemas: []core.Schema{{APIVersion: "example/v1", Purpose: "stable schema"}}}
	if !sameRefreshExternalDependencies(model, model, []core.Definition{selected}, []core.Edge{edge}, []core.Edge{edge}) {
		t.Fatal("unchanged direct external dependency should remain refreshable")
	}
	changedChild := model
	changedChild.Definitions = append([]core.Definition(nil), model.Definitions...)
	changedChild.Definitions[1].Purpose = "changed child meaning"
	if sameRefreshExternalDependencies(model, changedChild, []core.Definition{selected}, []core.Edge{edge}, []core.Edge{edge}) {
		t.Fatal("changed external child was ignored because the edge id stayed fixed")
	}
	changedSchema := model
	changedSchema.Schemas = append([]core.Schema(nil), model.Schemas...)
	changedSchema.Schemas[0].Purpose = "changed schema meaning"
	if sameRefreshExternalDependencies(model, changedSchema, []core.Definition{selected}, []core.Edge{edge}, []core.Edge{edge}) {
		t.Fatal("changed external schema was ignored")
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
	root, oldRevision, fixed := scopedCanonicalFixture(t)
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
	currentRevision := scopedTestGit(t, root, "rev-parse", "HEAD")
	cfg := CanonicalControllerConfig{
		APIVersion: CanonicalControllerAPIVersion, RecordStore: storeDir, PrivateLogs: filepath.Join(external, "logs"),
		Executor: CanonicalRunnerConfig{Command: "go", ProviderVersion: "test", TimeoutSeconds: 10},
		Verifier: CanonicalRunnerConfig{Command: "go", ProviderVersion: "test", TimeoutSeconds: 10},
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
	apply, err := ApplyCanonicalEvidenceRefresh(root, configPath, cfg, proposal, proposal.Digest, true)
	if err != nil || apply.Status != "refreshed" || len(apply.Records) != 1 {
		t.Fatalf("Apply retained evidence refresh: report=%+v err=%v", apply, err)
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
	for _, name := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil || !bytes.Equal(data, prepared.Outputs[name]) {
			t.Fatalf("refresh changed retained target %s: err=%v", name, err)
		}
	}
}
