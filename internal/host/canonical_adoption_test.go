package host

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/host/recordstore"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func adoptionFixture(t *testing.T) (*CanonicalSource, *snapshot.Snapshot, core.DefinitionIdentity, CanonicalAdoptionSelection) {
	t.Helper()
	fixed, target := reconciliationFixture(t)
	target.Provisional = false
	target.ID = strings.Repeat("b", 40)
	fixed.Config.Checks = []authoring.Check{{Name: "canonical-projection-fixture", Run: []string{"go", "version"}}}
	target.Files["docs/represented/legacy.md"] = []byte("Human-reviewed narrative; preserve this layout exactly.\n")
	target.Modes["docs/represented/legacy.md"] = snapshot.RegularMode
	identity := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "commerce", Name: "application-markdown"}
	return fixed, target, identity, CanonicalAdoptionSelection{ActiveRecords: []records.ProjectionRecord{}, Artifacts: []string{"docs/represented/legacy.md"}, ReviewReference: "owner-review/example: retain existing representation"}
}
func TestCanonicalAdoptionRetainsExistingBytesWithoutRendererOrWrites(t *testing.T) {
	fixed, target, identity, selection := adoptionFixture(t)
	before := target.Digest()
	plan, err := PrepareCanonicalAdoption(fixed, target, identity, selection)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := PrepareCanonicalAdoption(fixed, target, identity, selection)
	a, _ := json.Marshal(plan)
	b, _ := json.Marshal(repeated)
	if err != nil || string(a) != string(b) {
		t.Fatal("adoption plan is not deterministic")
	}
	if plan.Record.Origin != records.OriginAdopted || plan.Record.AdoptionRevision != target.ID || plan.Record.Artifacts[0].Change != records.ChangeRetained {
		t.Fatalf("lost adoption provenance: %#v", plan)
	}
	// This path/content is deliberately not the Markdown renderer's index.md.
	// Adoption verifies the reviewed representation rather than regenerating it.
	adopted, err := AdoptCanonicalProjection(fixed, target, identity, selection, plan.PlanDigest, records.VerifierIdentity{ID: "fixed-test-checks", Version: "1", Digest: sha256Prefix(sha256Hex([]byte("verifier")))})
	if err != nil || adopted.Record == nil || adopted.Verification.Result.Outcome != records.OutcomePassed {
		t.Fatalf("adoption=%#v err=%v", adopted, err)
	}
	if target.Digest() != before {
		t.Fatal("adoption mutated captured representation")
	}
	if _, exists := target.Files["docs/represented/index.md"]; exists {
		t.Fatal("adoption regenerated renderer-preferred artifact")
	}
	facts := []records.ArtifactFact{{Path: selection.Artifacts[0], Digest: plan.Record.Artifacts[0].Digest, Mode: snapshot.RegularMode, Role: records.RoleProjectionTarget}, {Path: "docs/represented/unknown.md", Digest: sha256Prefix(sha256Hex([]byte("unknown"))), Mode: snapshot.RegularMode, Role: records.RoleUnknown}}
	index, err := records.BuildOwnershipIndex([]records.ProjectionRecord{*adopted.Record}, facts)
	if err != nil {
		t.Fatal(err)
	}
	if index.Artifacts[selection.Artifacts[0]].Status != records.OwnershipManaged || index.Artifacts["docs/represented/unknown.md"].Status != records.OwnershipUnknown {
		t.Fatal("adoption broadened managed scope")
	}
	target.Provisional = true
	reconcile, err := PlanCanonicalReconciliation(fixed, fixed, target, []records.ProjectionRecord{*adopted.Record})
	if err != nil {
		t.Fatal(err)
	}
	for _, work := range reconcile.Work {
		if work.ProjectionID == identity.Key() {
			t.Fatal("adopted unchanged representation required regeneration")
		}
	}
}
func TestCanonicalAdoptionApprovalBindsEverySelectedInput(t *testing.T) {
	for _, change := range []string{"bytes", "mode", "revision", "review", "selection", "checker"} {
		t.Run(change, func(t *testing.T) {
			fixed, target, identity, selection := adoptionFixture(t)
			plan, err := PrepareCanonicalAdoption(fixed, target, identity, selection)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "bytes":
				target.Files[selection.Artifacts[0]] = []byte("changed")
			case "mode":
				target.Modes[selection.Artifacts[0]] = "100755"
			case "revision":
				target.ID = strings.Repeat("c", 40)
			case "review":
				selection.ReviewReference = "different owner review"
			case "selection":
				target.Files["docs/represented/another.md"] = []byte("more")
				target.Modes["docs/represented/another.md"] = snapshot.RegularMode
				selection.Artifacts = append(selection.Artifacts, "docs/represented/another.md")
			case "checker":
				fixed.Config.Checks[0].Run = []string{"go", "env"}
			}
			fresh, err := PrepareCanonicalAdoption(fixed, target, identity, selection)
			if err != nil {
				t.Fatal(err)
			}
			if fresh.PlanDigest == plan.PlanDigest {
				t.Fatal("changed input did not stale approval")
			}
			result, err := AdoptCanonicalProjection(fixed, target, identity, selection, plan.PlanDigest, records.VerifierIdentity{})
			if err == nil || result.Record != nil || !strings.Contains(err.Error(), "approval") {
				t.Fatal("stale adoption was accepted")
			}
		})
	}
}

func TestCanonicalDurableAdoptionBindsAbsentVersusPresentLedger(t *testing.T) {
	fixed, target, identity, selection := adoptionFixture(t)
	configDigest := sha256Prefix(sha256Hex([]byte("runtime-config")))
	absent := CanonicalAdoptionLedgerBinding{Present: false, ActiveRecordIDs: []string{}, ConfigDigest: configDigest}
	absentPlan, err := PrepareCanonicalDurableAdoption(fixed, target, identity, selection, absent, []records.ProjectionRecord{})
	if err != nil {
		t.Fatal(err)
	}
	present := CanonicalAdoptionLedgerBinding{Present: true, StoreID: "store-id", Head: sha256Prefix(sha256Hex([]byte("head"))), ActiveRecordIDs: []string{}, ConfigDigest: configDigest}
	presentPlan, err := PrepareCanonicalDurableAdoption(fixed, target, identity, selection, present, []records.ProjectionRecord{})
	if err != nil {
		t.Fatal(err)
	}
	if absentPlan.PlanDigest == presentPlan.PlanDigest {
		t.Fatal("approval did not bind absent versus present ledger state")
	}
	present.Head = sha256Prefix(sha256Hex([]byte("new head")))
	changedHead, err := PrepareCanonicalDurableAdoption(fixed, target, identity, selection, present, []records.ProjectionRecord{})
	if err != nil {
		t.Fatal(err)
	}
	if changedHead.PlanDigest == presentPlan.PlanDigest {
		t.Fatal("approval did not bind external ledger head")
	}
	selection.ActiveRecords = []records.ProjectionRecord{{}}
	if _, err := PrepareCanonicalDurableAdoption(fixed, target, identity, selection, absent, []records.ProjectionRecord{}); err == nil {
		t.Fatal("durable adoption trusted caller-supplied active ownership")
	}
}

func canonicalDurableAdoptionSelectionFailureFixture(t *testing.T) (string, string, *CanonicalSource, *snapshot.Snapshot, core.DefinitionIdentity, CanonicalAdoptionSelection, CanonicalControllerConfig, string, string, string) {
	t.Helper()
	root, baseRevision, cfg := canonicalControllerFixture(t)
	fixed, err := LoadCanonicalSource(root, baseRevision, "examples/canonical-projection/canonical.yaml", true)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"src/Commerce/CreateOrderHandler.cs": "namespace Commerce; public class CreateOrderHandler { }\n",
		"src/Commerce/EffectAxis.cs":         "namespace Commerce; public record EffectAxis { public string Boundary { get; init; } = \"application\"; }\n",
		"src/Commerce/Commerce.csproj":       "<Project><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>\n",
	} {
		scopedTestWrite(t, root, name, content)
	}
	scopedTestGit(t, root, "add", "src/Commerce")
	scopedTestGit(t, root, "commit", "-m", "add selected adoption artifacts")
	targetRevision := scopedTestGit(t, root, "rev-parse", "HEAD")
	target, err := source.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	identity := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "commerce", Name: "application-dotnet"}
	selection := CanonicalAdoptionSelection{
		ActiveRecords:   []records.ProjectionRecord{},
		Artifacts:       []string{"src/Commerce/CreateOrderHandler.cs", "src/Commerce/EffectAxis.cs", "src/Commerce/Commerce.csproj"},
		ReviewReference: "owner-review/partial-failure-test",
	}
	headBefore := scopedTestGit(t, root, "rev-parse", "HEAD")
	indexBefore := scopedTestGit(t, root, "diff", "--cached", "--binary")
	statusBefore := scopedTestGit(t, root, "status", "--porcelain")
	plan, err := PrepareCanonicalAdoptionForRuntime(root, fixed, target, identity, selection, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return root, plan.PlanDigest, fixed, target, identity, selection, cfg, headBefore, indexBefore, statusBefore
}

func assertCanonicalDurableAdoptionAttemptUnverified(t *testing.T, root string, cfg CanonicalControllerConfig, applied CanonicalAdoptionApply, activeIDs []string) {
	t.Helper()
	store, err := recordstore.Open(cfg.RecordStore, canonicalControllerForbidden(t, root))
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Records) != 1 || state.Records[0].ID != applied.Record.ID || !reflect.DeepEqual(state.ActiveSelection.RecordIDs, activeIDs) || len(state.Verifications) != 0 {
		t.Fatalf("partial adoption did not retain the expected unverified attempt state: %#v", state)
	}
	if applied.Record.Origin != records.OriginAdopted || applied.Record.State != records.StateMaterializedUnverified {
		t.Fatalf("adoption attempt claimed unsupported origin or verification: %#v", applied.Record)
	}
}

func TestCanonicalDurableAdoptionSelectionFailureReportsObservedInactiveAttempt(t *testing.T) {
	root, digest, fixed, target, identity, selection, cfg, headBefore, indexBefore, statusBefore := canonicalDurableAdoptionSelectionFailureFixture(t)
	failure := errors.New("injected stale-head CAS rejection")
	applied, err := applyCanonicalAdoptionToLedger(root, fixed, target, identity, selection, cfg, digest, true,
		func(_ *recordstore.Store, _ string, _ []string) (recordstore.State, error) {
			return recordstore.State{}, failure
		})
	if err == nil || !errors.Is(err, failure) || !strings.Contains(err.Error(), failure.Error()) || applied.Status != records.StatePartialFailure || applied.Record == nil || applied.ActiveSelectionStatus != "observed-not-selected" || applied.LedgerHead == "" || len(applied.ActiveRecordIDs) != 0 {
		t.Fatalf("selection failure did not report the observed inactive attempt and original cause: result=%#v err=%v", applied, err)
	}
	assertCanonicalDurableAdoptionAttemptUnverified(t, root, cfg, applied, []string{})
	if scopedTestGit(t, root, "rev-parse", "HEAD") != headBefore || scopedTestGit(t, root, "diff", "--cached", "--binary") != indexBefore || scopedTestGit(t, root, "status", "--porcelain") != statusBefore {
		t.Fatal("partial durable adoption changed source HEAD, index or worktree")
	}
}

func TestCanonicalDurableAdoptionSelectionErrorReportsCommittedSelection(t *testing.T) {
	root, digest, fixed, target, identity, selection, cfg, _, _, _ := canonicalDurableAdoptionSelectionFailureFixture(t)
	failure := errors.New("selection event committed; response readback failed")
	applied, err := applyCanonicalAdoptionToLedger(root, fixed, target, identity, selection, cfg, digest, true,
		func(store *recordstore.Store, head string, ids []string) (recordstore.State, error) {
			if _, selectErr := store.SelectActive(head, ids); selectErr != nil {
				t.Fatalf("commit active selection: %v", selectErr)
			}
			return recordstore.State{}, failure
		})
	if err == nil || !errors.Is(err, failure) || applied.ActiveSelectionStatus != "observed-selected" || applied.Record == nil || applied.LedgerHead == "" || len(applied.ActiveRecordIDs) != 1 || applied.ActiveRecordIDs[0] != applied.Record.ID {
		t.Fatalf("committed selection was not reported from recovery state: result=%#v err=%v", applied, err)
	}
	assertCanonicalDurableAdoptionAttemptUnverified(t, root, cfg, applied, []string{applied.Record.ID})
}

func TestCanonicalDurableAdoptionSelectionErrorWithUnreadableRecoveryReportsUnknown(t *testing.T) {
	root, digest, fixed, target, identity, selection, cfg, _, _, _ := canonicalDurableAdoptionSelectionFailureFixture(t)
	failure := errors.New("injected selection failure")
	recoveryFailure := errors.New("injected recovery read failure")
	applied, err := applyCanonicalAdoptionToLedgerWithRecoveryRead(root, fixed, target, identity, selection, cfg, digest, true,
		func(_ *recordstore.Store, _ string, _ []string) (recordstore.State, error) {
			return recordstore.State{}, failure
		},
		func(_ *recordstore.Store) (recordstore.State, error) { return recordstore.State{}, recoveryFailure })
	if err == nil || !errors.Is(err, failure) || !strings.Contains(err.Error(), recoveryFailure.Error()) || applied.ActiveSelectionStatus != "unknown" || applied.LedgerHead != "" || applied.ActiveRecordIDs != nil || applied.Record == nil {
		t.Fatalf("unreadable recovery did not preserve unknown state and both causes: result=%#v err=%v", applied, err)
	}
	assertCanonicalDurableAdoptionAttemptUnverified(t, root, cfg, applied, []string{})
}
func TestCanonicalAdoptionRefusesUnsafeOrIncompleteSelection(t *testing.T) {
	for _, invalid := range []string{"missing", "outside", "duplicate", "symlink", "unsafe", "alias", "no-review", "no-checks", "provisional", "source-changed", "structural"} {
		t.Run(invalid, func(t *testing.T) {
			fixed, target, identity, selection := adoptionFixture(t)
			switch invalid {
			case "missing":
				selection.Artifacts = []string{"docs/represented/missing.md"}
			case "outside":
				selection.Artifacts = []string{"scratch/file.md"}
			case "duplicate":
				selection.Artifacts = append(selection.Artifacts, selection.Artifacts[0])
			case "symlink":
				target.Modes[selection.Artifacts[0]] = "120000"
			case "unsafe":
				selection.Artifacts = []string{"docs/represented/../legacy.md"}
			case "alias":
				target.Files["docs/represented/LEGACY.md"] = []byte("case alias")
			case "no-review":
				selection.ReviewReference = " "
			case "no-checks":
				fixed.Config.Checks = nil
			case "provisional":
				target.Provisional = true
			case "source-changed":
				target.Files[fixed.ConfigPath] = []byte("different canonical source")
			case "structural":
				fixed.Diagnostics = []core.Diagnostic{{Code: "broken"}}
			}
			before := target.Digest()
			if _, err := PrepareCanonicalAdoption(fixed, target, identity, selection); err == nil {
				t.Fatal("invalid adoption selection accepted")
			}
			if target.Digest() != before {
				t.Fatal("refused plan changed target")
			}
		})
	}
}
func TestCanonicalAdoptionDoesNotReturnRecordWhenCheckFails(t *testing.T) {
	fixed, target, identity, selection := adoptionFixture(t)
	fixed.Config.Checks[0].Run = []string{"go", "run", "missing-check.go"}
	plan, err := PrepareCanonicalAdoption(fixed, target, identity, selection)
	if err != nil {
		t.Fatal(err)
	}
	before := target.Digest()
	result, err := AdoptCanonicalProjection(fixed, target, identity, selection, plan.PlanDigest, records.VerifierIdentity{ID: "fixed-test-checks", Version: "1", Digest: sha256Prefix(sha256Hex([]byte("verifier")))})
	if err == nil || result.Record != nil || result.Verification.Result.Outcome != records.OutcomeFailed {
		t.Fatalf("failed checks became adopted ownership: %#v %v", result, err)
	}
	if !reflect.DeepEqual(before, target.Digest()) {
		t.Fatal("failed adoption mutated target")
	}
}

func TestCanonicalAdoptionExposesUnknownsAndRejectsActiveOwnerCollisions(t *testing.T) {
	fixed, target, identity, selection := adoptionFixture(t)
	target.Files["docs/represented/unmatched.md"] = []byte("Unresolved existing owner")
	plan, err := PrepareCanonicalAdoption(fixed, target, identity, selection)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.UnmatchedArtifacts, []string{"docs/represented/unmatched.md"}) {
		t.Fatalf("unmatched bytes disappeared: %#v", plan.UnmatchedArtifacts)
	}
	conflicting := plan.Record
	conflicting.Origin = ""
	conflicting.ReviewReference = ""
	conflicting.AdoptionRevision = ""
	conflicting.ProjectionID = (core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "commerce", Name: "other"}).Key()
	conflicting, err = records.NewProjectionRecord(conflicting)
	if err != nil {
		t.Fatal(err)
	}
	selection.ActiveRecords = []records.ProjectionRecord{conflicting}
	if _, err := PrepareCanonicalAdoption(fixed, target, identity, selection); err == nil {
		t.Fatal("adoption claimed an actively owned artifact")
	}
	alias := conflicting
	alias.Artifacts = append([]records.Artifact(nil), conflicting.Artifacts...)
	alias.Artifacts[0].Path = strings.ToUpper(alias.Artifacts[0].Path)
	alias, err = records.NewProjectionRecord(alias)
	if err != nil {
		t.Fatal(err)
	}
	selection.ActiveRecords = []records.ProjectionRecord{alias}
	if _, err := PrepareCanonicalAdoption(fixed, target, identity, selection); err == nil {
		t.Fatal("case-alias active owner evaded adoption conflict check")
	}
	selection.ActiveRecords = []records.ProjectionRecord{plan.Record}
	if _, err := PrepareCanonicalAdoption(fixed, target, identity, selection); err == nil {
		t.Fatal("adoption silently replaced active representation owner")
	}
	selection.ActiveRecords = nil
	if _, err := PrepareCanonicalAdoption(fixed, target, identity, selection); err == nil {
		t.Fatal("implicit unknown active ownership accepted")
	}
}
