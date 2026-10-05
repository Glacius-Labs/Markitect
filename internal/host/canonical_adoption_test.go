package host

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
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
