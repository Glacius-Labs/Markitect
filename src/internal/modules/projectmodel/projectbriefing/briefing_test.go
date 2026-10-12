package projectbriefing

import (
	"errors"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestBuildDerivesBriefingsFromExplicitRevisionValues(t *testing.T) {
	before, after := revisionFixtures()
	bundle, err := Build(before, after, Provenance{DecisionReference: "ADR-17", Actor: "owner", Authority: "accepted decision"})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Events) != 1 || bundle.Events[0].Before == nil || bundle.Events[0].After == nil {
		t.Fatalf("events = %#v", bundle.Events)
	}
	if bundle.Events[0].Before.Spec["description"] != "before" || bundle.Events[0].After.Spec["description"] != "after" {
		t.Fatalf("before/after = %#v / %#v", bundle.Events[0].Before, bundle.Events[0].After)
	}
	if bundle.Revision != after.Revision || bundle.SinceRevision != before.Revision || bundle.Digest != FullBundleDigest(bundle) {
		t.Fatalf("revision or digest binding lost: %#v", bundle)
	}
}

func TestBuildUnchangedValuesHaveStableBriefingIdentity(t *testing.T) {
	before, after := revisionFixtures()
	unchanged := after
	unchanged.Revision = strings.Repeat("c", 40)
	unchanged.Report.Digest = "report-c"
	first, err := Build(after, unchanged, Provenance{DecisionReference: "ADR-17", Actor: "owner", Authority: "accepted decision"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(after, unchanged, Provenance{DecisionReference: "ADR-17", Actor: "owner", Authority: "accepted decision"})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Events) != 0 || first.Global.ID != second.Global.ID {
		t.Fatalf("unchanged build changed: first=%#v second=%#v", first, second)
	}
	_ = before
}

func TestBuildRejectsProvisionalRevisionValue(t *testing.T) {
	before, after := revisionFixtures()
	after.Provisional = true
	_, err := Build(before, after, Provenance{DecisionReference: "ADR-17", Actor: "owner", Authority: "accepted decision"})
	if !errors.Is(err, ErrUncommittedModel) {
		t.Fatalf("error = %v, want ErrUncommittedModel", err)
	}
}

func TestResolutionViewCopiesHintsAndAcceptedEvidence(t *testing.T) {
	accepted := Resolution{EventID: "event", ModelRevision: strings.Repeat("a", 40), ModelDigest: "model", Evidence: VerifiedResolutionEvidence{EventIDs: []string{"event"}, CoveredManagerIDs: []string{"manager"}, EvidenceRefs: []string{"ref"}, Delivered: []DeliveredFile{{Path: "file", Mode: "100644", Object: strings.Repeat("b", 40)}}}}
	hint := accepted
	hint.ModelRevision = strings.Repeat("c", 40)
	hint.Evidence.EventIDs = []string{"event", "other"}
	laterAccepted := accepted
	laterAccepted.ModelRevision = strings.Repeat("d", 40)
	store := Store{Resolutions: []Resolution{accepted, laterAccepted}}
	before := StoreDigest(store)
	view := NewResolutionView(store, []Resolution{hint})
	if after := StoreDigest(store); after != before {
		t.Fatalf("constructing a status view changed durable store digest: before=%s after=%s", before, after)
	}
	store.Resolutions[0].Evidence.EventIDs[0] = "mutated"
	hint.Evidence.Delivered[0].Path = "mutated"
	status := EventResolutionStatus(view, "event")
	if status.Status != StatusResolved || status.Resolution == nil || status.Resolution.ModelRevision != accepted.ModelRevision || status.Resolution.Evidence.EventIDs[0] != "event" {
		t.Fatalf("accepted resolution changed or hint took priority: %#v", status)
	}
	status.Resolution.Evidence.Delivered[0].Path = "returned mutation"
	again := EventResolutionStatus(view, "event")
	if again.Resolution.Evidence.Delivered[0].Path != "file" {
		t.Fatalf("returned evidence aliases view: %#v", again.Resolution)
	}
	hintView := NewResolutionView(Store{}, []Resolution{hint})
	hintStatus := EventResolutionStatus(hintView, "event")
	if hintStatus.Status != StatusDeliveredUncommitted || hintStatus.Resolution == nil {
		t.Fatalf("hint status = %#v", hintStatus)
	}
}

func revisionFixtures() (RevisionModel, RevisionModel) {
	manager := projectmodel.Manager{ID: core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Namespace: "demo", Name: "root"}.Key(), Name: "root", Namespace: "demo", Owns: []string{"."}}
	statement := projectmodel.Statement{ID: core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "demo", Name: "rule"}.Key(), Name: "rule", Namespace: "demo", Owner: manager.ID, Category: "rule", Description: "before", Public: true, Uses: []string{}, Requires: []string{}}
	identity := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "demo", Name: "rule"}
	beforeDef := core.Definition{APIVersion: projectmodel.APIVersion, Kind: identity.Kind, Metadata: core.Metadata{Namespace: identity.Namespace, Name: identity.Name}, Purpose: "Rule", Spec: map[string]any{"category": "rule", "description": "before", "public": true, "uses": []any{}, "requires": []any{}}, Source: core.Source{Path: ".markitect/model/demo.yaml", Digest: "before"}}
	afterDef := beforeDef
	afterDef.Spec = map[string]any{"category": "rule", "description": "after", "public": true, "uses": []any{}, "requires": []any{}}
	afterDef.Source = core.Source{Path: ".markitect/model/demo.yaml", Digest: "after"}
	before := RevisionModel{Revision: strings.Repeat("a", 40), Model: core.Model{Digest: "model-a", Definitions: []core.Definition{beforeDef}}, Report: projectmodel.Report{Digest: "report-a", Managers: []projectmodel.Manager{manager}, Statements: []projectmodel.Statement{statement}}}
	statement.Description = "after"
	after := RevisionModel{Revision: strings.Repeat("b", 40), Model: core.Model{Digest: "model-b", Definitions: []core.Definition{afterDef}}, Report: projectmodel.Report{Digest: "report-b", Managers: []projectmodel.Manager{manager}, Statements: []projectmodel.Statement{statement}}}
	return before, after
}
