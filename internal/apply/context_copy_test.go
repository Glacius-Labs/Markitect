package apply_test

import (
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func TestStaticContextAccessorsKeepOwnedCopies(t *testing.T) {
	selection, raw := scopedContextInputs(t, "lending", contextScope)
	result := registryModel(t, selection, raw).DecodeExecution().SelectContexts()
	expectedDefinitions := make(map[model.Identity]model.Definition)
	for _, owner := range result.Execution().Model().Compilation().Models {
		for _, definition := range owner.Definitions {
			expectedDefinitions[definition.Identity] = definition
		}
	}
	original, ok := result.Selections()
	if !ok {
		t.Fatalf("copy fixture failed: %+v", result.Diagnostics())
	}
	changed, _ := result.Selections()
	changed[0] = organization.ContextSelection{}
	if fresh, _ := result.Selections(); !reflect.DeepEqual(fresh, original) {
		t.Fatal("selection slice changed internal complete set")
	}
	lending := selectedContext(t, original, "lending")
	global, found := lending.Definition(contextID("rules", "global"))
	if !found || global.Definition.Title == nil || len(global.Reasons) == 0 || global.Reasons[0].Policy == nil {
		t.Fatal("fixture lacks nested copy targets")
	}
	global.Definition.Body = "changed"
	*global.Definition.Title = "changed"
	global.Definition.Spec[0].Name = "changed"
	global.Reasons[0].Policy.Field = "changed"
	global.Reasons[0].ScopePackage = "changed"
	definitions := lending.Definitions()
	definitions[0].Definition.Body = "changed"
	direct, found := lending.Definition(contextID("lending", "direct"))
	if !found || len(direct.Definition.BodyLinks) != 1 || direct.Definition.BodyLinks[0].Display == nil || direct.Definition.BodyLinks[0].Coordinates == nil {
		t.Fatal("fixture lacks body-link copy targets")
	}
	*direct.Definition.BodyLinks[0].Display = "changed"
	coordinates := direct.Definition.BodyLinks[0].Coordinates
	if coordinates.Body != nil {
		coordinates.Body.StartByte = 99
	}
	if coordinates.Source != nil {
		coordinates.Source.Start.Line = 99
	}
	if fresh, _ := result.Selections(); !reflect.DeepEqual(fresh, original) {
		t.Fatal("nested context copy mutation reached source selections")
	}
	for _, definition := range lending.Definitions() {
		if !reflect.DeepEqual(definition.Definition, expectedDefinitions[definition.Definition.Identity]) {
			t.Fatal("context content changed from independent parent compilation")
		}
	}
	freshGlobal, _ := lending.Definition(contextID("rules", "global"))
	if freshGlobal.Reasons[0].Policy.Field != "/scope/0" || freshGlobal.Reasons[0].ScopePackage != "." {
		t.Fatal("scope reason mutation changed retained provenance")
	}
	parent := result.Execution().Partial()
	parent.Scope.Value[0].Source.Field = "changed"
	parent.Scope.Value[0].Value.Kind.ID = "changed"
	inputs := result.Execution().Model().Inputs()
	inputs.Raw.Execution[0].RepositoryPath = path(t, "other.yaml")
	for i := range inputs.Raw.Local.Source.Entries {
		if len(inputs.Raw.Local.Source.Entries[i].Bytes) != 0 {
			inputs.Raw.Local.Source.Entries[i].Bytes[0] = '!'
			break
		}
	}
	if fresh, _ := result.Selections(); !reflect.DeepEqual(fresh, original) || !reflect.DeepEqual(result.Execution().Model().Inputs().Raw, raw) {
		t.Fatal("parent accessor changed static results or raw source")
	}
}

func TestStaticContextRefusalDiagnosticsAreCopied(t *testing.T) {
	selection, raw := scopedContextInputs(t, "unknown", contextScope)
	result := registryModel(t, selection, raw).DecodeExecution().SelectContexts()
	original := result.Diagnostics()
	changed := result.Diagnostics()
	if len(changed) == 0 {
		t.Fatal("fixture lacks refusal diagnostics")
	}
	expected := original[0]
	changed[0].Package, changed[0].Message = "changed", "changed"
	if !reflect.DeepEqual(result.Diagnostics(), original) || result.Diagnostics()[0] != expected {
		t.Fatal("diagnostic slice changed retained refusal")
	}
}
