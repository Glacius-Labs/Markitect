package project_test

import (
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func TestStaticContextsUseOnlyExactAuthoredModeledScope(t *testing.T) {
	selection, raw := scopedContextInputs(t, "lending", contextScope)
	parent := registryModel(t, selection, raw).DecodeExecution()
	result := parent.SelectContexts()
	contexts, ok := result.Selections()
	if !ok {
		t.Fatalf("modeled scope failed: %+v", result.Diagnostics())
	}
	for _, pkg := range []string{".", "lending", "lending/reminders", "members"} {
		selected := selectedContext(t, contexts, pkg)
		global, found := selected.Definition(contextID("rules", "global"))
		if !found || !hasContextReason(global, organization.ModeledScope, global.Definition.Identity, "/spec/appliesTo/0") {
			t.Fatalf("global scope lost for %q: %+v", pkg, global.Reasons)
		}
		for _, reason := range global.Reasons {
			if reason.Kind == organization.ModeledScope && (reason.Policy == nil || *reason.Policy != parent.Partial().Scope.Value[0].Source || reason.Policy.ContentID != raw.Execution[0].File.ContentID() || reason.Policy.Field != "/scope/0" || reason.ScopePackage != ".") {
				t.Fatal("modeled scope policy was synthesized or source changed")
			}
		}
		ancestor, found := selected.Definition(contextID("rules", "ancestor"))
		if want := pkg == "lending" || pkg == "lending/reminders"; found != want {
			t.Fatalf("ancestor selection for %q = %v", pkg, found)
		}
		if found && (!hasContextReason(ancestor, organization.ModeledScope, ancestor.Definition.Identity, "/spec/appliesTo/0") || !hasContextReason(ancestor, organization.ModeledScope, ancestor.Definition.Identity, "/spec/appliesTo/1")) {
			t.Fatal("repeated authored scope occurrence reasons lost")
		}
		for _, excluded := range []model.Identity{contextID("rules", "sibling"), contextID("rules", "absent"), contextID("rules", "empty"), contextID("unrelated", "own"), contextID("extra", "far")} {
			if _, found := selected.Definition(excluded); found {
				t.Fatalf("unrelated/transitive Definition %+v leaked to %q", excluded, pkg)
			}
		}
	}
	lending := selectedContext(t, contexts, "lending")
	near, found := lending.Definition(contextID("extra", "near"))
	if !found || !hasContextReason(near, organization.OutgoingReference, contextID("lending", "direct"), "/bodyLinks/0") {
		t.Fatal("direct edge lost with scope enabled")
	}
	root := selectedContext(t, contexts, ".")
	if _, found := root.Definition(contextID("lending", "borrow")); found {
		t.Fatal("root acquired descendant own Definitions through global scope")
	}
	if !reflect.DeepEqual(result.Execution().Model().Inputs(), parent.Model().Inputs()) {
		t.Fatal("modeled fixture binding changed")
	}
}

func TestStaticContextScopeAbsenceAndExplicitEmptyStayDistinct(t *testing.T) {
	var selections [][]organization.ContextSelection
	for _, scope := range []string{"", "scope: []\n"} {
		selection, raw := scopedContextInputs(t, "lending", scope)
		parent := registryModel(t, selection, raw).DecodeExecution()
		result := parent.SelectContexts()
		contexts, ok := result.Selections()
		if !ok {
			t.Fatalf("empty/absent policy failed: %+v", result.Diagnostics())
		}
		if (result.Execution().Partial().Scope == nil) != (scope == "") {
			t.Fatal("authored policy absence lost")
		}
		if len(selectedContext(t, contexts, ".").Definitions()) != 0 {
			t.Fatal("scope inferred without a declared selector")
		}
		selections = append(selections, contexts)
	}
	if !reflect.DeepEqual(selections[0], selections[1]) {
		t.Fatal("explicit empty scope changed static selection")
	}
}
