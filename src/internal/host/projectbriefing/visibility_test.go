package projectbriefing

import (
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestProjectEventScopesDefinitionSidesAndRelationsToHistoricalContext(t *testing.T) {
	definition := core.Definition{
		APIVersion: projectmodel.APIVersion,
		Kind:       "Statement",
		Metadata:   core.Metadata{Name: "contract", Namespace: "demo"},
		Purpose:    "not part of the ManagerContext projection",
		Spec:       map[string]any{"uses": []any{"private-target"}, "requires": []any{"private-requirement"}},
	}
	id := definition.Identity().Key()
	managerID := `manager:demo`
	visibleBefore := managerView{
		statements: map[string]projectmodel.Statement{
			id: {ID: id, Public: true, Uses: []string{"public-target"}, Requires: []string{"public-requirement"}},
		},
		artifacts: map[string]bool{}, checks: map[string]projectmodel.Check{}, decisions: map[string]bool{},
		managers: map[string]bool{managerID: true},
	}
	hiddenAfter := emptyManagerView()
	event := Event{
		ID: "event", DefinitionID: definition.Identity(), Change: "modified",
		Before: &definition, After: &definition,
		AffectedManagers:  []string{managerID, `manager:private-child`},
		AffectedArtifacts: []string{`artifact:private-child`},
	}

	projected, visible := projectEvent(event, visibleBefore, hiddenAfter)
	if !visible {
		t.Fatal("public Before should keep the event visible to its historical ManagerContext")
	}
	if projected.Before == nil || projected.After != nil {
		t.Fatalf("public-to-private projection retained wrong sides: Before=%v After=%v", projected.Before, projected.After)
	}
	uses, ok := projected.Before.Spec["uses"].([]string)
	if !ok || !reflect.DeepEqual(uses, []string{"public-target"}) {
		t.Fatalf("visible public relation projection = %#v, want only public-target", projected.Before.Spec["uses"])
	}
	requires, ok := projected.Before.Spec["requires"].([]string)
	if !ok || !reflect.DeepEqual(requires, []string{"public-requirement"}) {
		t.Fatalf("visible public obligation projection = %#v, want only public-requirement", projected.Before.Spec["requires"])
	}
	if projected.Before.Purpose != "" {
		t.Fatalf("Statement purpose outside ManagerContext survived projection: %q", projected.Before.Purpose)
	}
	if !reflect.DeepEqual(projected.AffectedManagers, []string{managerID}) || len(projected.AffectedArtifacts) != 0 {
		t.Fatalf("private affected identifiers survived projection: managers=%v artifacts=%v", projected.AffectedManagers, projected.AffectedArtifacts)
	}

	projected, visible = projectEvent(event, hiddenAfter, visibleBefore)
	if !visible || projected.Before != nil || projected.After == nil {
		t.Fatalf("private-to-public projection retained wrong sides: visible=%v Before=%v After=%v", visible, projected.Before, projected.After)
	}
}
