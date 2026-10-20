package apply_test

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
)

func TestSessionBasisNamesRootsExecuteProfileBudgetAndLanguage(t *testing.T) {
	selection, raw := executionInputs(t, withOwnership(t, libraryOwnershipYAML))
	libraryLayoutFiles(t, &selection, &raw)
	attempt := preparedAttempt(t, selection, raw)
	basis, err := apply.SessionBasisOf(attempt)
	if err != nil {
		t.Fatal(err)
	}
	settings, _ := attempt.Settings().Settings()
	root, _ := settings.Manager(".")
	budget, _ := attempt.Settings().Budget()
	if basis.Profile == "" || basis.Profile != root.Execute.Profile.Value.Name || basis.TurnTimeout != budget.TurnTimeout.Value || basis.Turns != budget.AgentTurns.Value {
		t.Fatalf("basis = %+v", basis)
	}
	if !strings.Contains(basis.Guide, "Writing Definitions (language v1)") || !strings.Contains(basis.Guide, "Kinds of the imported vocabulary") {
		t.Fatalf("the basis lacks the language guide:\n%s", basis.Guide)
	}
	if basis.ModelDir != ".markitect/model" {
		t.Fatalf("model directory = %q", basis.ModelDir)
	}
	if basis.Attempt.Model().Inputs().Head != attempt.Structural().Model().Inputs().Head {
		t.Fatal("the basis is not the attempt's structural preparation")
	}
}
