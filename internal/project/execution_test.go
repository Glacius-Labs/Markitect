package project_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func executionInputs(t *testing.T, content string) (project.InputSelection, project.RawInputs) {
	t.Helper()
	selection, raw := libraryInputs(t)
	censusFile(t, &raw, ".markitect/execution.yaml", snapshot.Executable, []byte(content))
	location := project.SourceLocation{Boundary: "explicit-execution-source", Path: path(t, "acquired/project-settings.data")}
	file, err := snapshot.NewEntry(location.Path, snapshot.Executable, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	selection.Execution = []project.SourceLocation{location}
	raw.Execution = []project.SourceFile{{Location: location, File: file, RepositoryPath: path(t, ".markitect/execution.yaml")}}
	return selection, raw
}

func executionDiagnostic(diagnostics []project.ExecutionDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

const libraryOwnershipYAML = `ownership:
  - path: realization/
    owner: .
  - path: realization/lending/
    owner: lending
  - path: realization/lending/reminders/
    owner: lending/reminders
  - path: realization/library/
    owner: library
  - path: realization/cmd/library/
    owner: library
    reason: The callable interface fixes the executable location.
`

func withOwnership(t *testing.T, ownership string) string {
	t.Helper()
	return strings.Replace(documentedExecution(t), "checks:\n", ownership+"checks:\n", 1)
}

func TestExecutionOwnershipDecodesWithSourcesAndIsValidatedAgainstTheModel(t *testing.T) {
	result := executionResult(t, withOwnership(t, libraryOwnershipYAML))
	document, ok := result.Document()
	if !ok || document.Ownership == nil {
		t.Fatalf("ownership failed: %s %+v", result.Status(), result.Diagnostics())
	}
	ownership := document.Ownership
	if ownership.Source.Field != "/ownership" || len(ownership.Value) != 5 {
		t.Fatalf("ownership changed: %+v", ownership)
	}
	for i, want := range []struct{ path, owner string }{
		{"realization/", "."}, {"realization/lending/", "lending"}, {"realization/lending/reminders/", "lending/reminders"}, {"realization/library/", "library"}, {"realization/cmd/library/", "library"},
	} {
		area := ownership.Value[i]
		field := "/ownership/" + strconv.Itoa(i)
		if area.Path.Value != want.path || area.Path.Source.Field != field+"/path" || area.Owner.Value != want.owner || area.Owner.Source.Field != field+"/owner" {
			t.Errorf("area %d or its sources changed: %+v", i, area)
		}
	}
	last := ownership.Value[4]
	if last.Reason.Value != "The callable interface fixes the executable location." || last.Reason.Source.Field != "/ownership/4/reason" || ownership.Value[0].Reason.Source.File != "" {
		t.Errorf("reason or its absence changed: %+v %+v", last.Reason, ownership.Value[0].Reason)
	}
	// An exact file, a directory and the whole repository are the three selectors.
	selectors := executionResult(t, withOwnership(t, "ownership:\n  - {path: './', owner: .}\n  - {path: realization/lending/, owner: lending}\n  - {path: realization/Makefile, owner: .}\n  - {path: realization/lending/Makefile, owner: lending, reason: Lending builds itself.}\n"))
	if document, ok := selectors.Document(); !ok || len(document.Ownership.Value) != 4 {
		t.Fatalf("selectors refused: %s %+v", selectors.Status(), selectors.Diagnostics())
	}
	empty := executionResult(t, withOwnership(t, "ownership: []\n"))
	if document, ok := empty.Document(); !ok || document.Ownership == nil || len(document.Ownership.Value) != 0 {
		t.Fatalf("explicit empty ownership lost: %+v", empty.Diagnostics())
	}
	absent, _ := executionResult(t, documentedExecution(t)).Document()
	if absent.Ownership != nil {
		t.Fatalf("absent ownership was invented: %+v", absent.Ownership)
	}
}

func TestExecutionOwnershipRefusals(t *testing.T) {
	for _, tc := range []struct{ name, ownership, code string }{
		{"not a list", "ownership: realization/\n", "execution.list"},
		{"entry not a mapping", "ownership: [realization/]\n", "execution.mapping"},
		{"unknown key", "ownership:\n  - {path: realization/, owner: ., kind: file}\n", "execution.unknown_key"},
		{"missing path", "ownership:\n  - {owner: .}\n", "execution.field_missing"},
		{"missing owner", "ownership:\n  - {path: realization/}\n", "execution.field_missing"},
		{"empty reason", "ownership:\n  - {path: realization/, owner: ., reason: ''}\n", "execution.string"},
		{"escaping path", "ownership:\n  - {path: ../a/, owner: .}\n", "execution.ownership_path"},
		{"absolute path", "ownership:\n  - {path: /tmp/a/, owner: .}\n", "execution.ownership_path"},
		{"unclean path", "ownership:\n  - {path: realization//lending/, owner: .}\n", "execution.ownership_path"},
		{"glob-free but empty", "ownership:\n  - {path: '/', owner: .}\n", "execution.ownership_path"},
		{"unknown owner", "ownership:\n  - {path: realization/, owner: missing}\n", "execution.package"},
		{"imported owner", "ownership:\n  - {path: realization/, owner: software}\n", "execution.package"},
		{"equal specificity", "ownership:\n  - {path: realization/, owner: .}\n  - {path: realization/, owner: library}\n", "execution.ownership"},
		{"same exact file twice", "ownership:\n  - {path: realization/Makefile, owner: .}\n  - {path: realization/Makefile, owner: library}\n", "execution.ownership"},
		{"whole repository twice", "ownership:\n  - {path: ./, owner: .}\n  - {path: ./, owner: library}\n", "execution.ownership"},
		{"file and directory of one path", "ownership:\n  - {path: realization, owner: .}\n  - {path: realization/, owner: library}\n", "execution.ownership"},
		{"below an exact file", "ownership:\n  - {path: realization/a, owner: .}\n  - {path: realization/a/b, owner: library}\n", "execution.ownership"},
		{"markitect directory", "ownership:\n  - {path: .markitect/, owner: .}\n", "execution.ownership"},
		{"below markitect", "ownership:\n  - {path: .markitect/registry/lending/, owner: lending}\n", "execution.ownership"},
		{"exact file in markitect", "ownership:\n  - {path: .markitect/execution.yaml, owner: .}\n", "execution.ownership"},
		{"git directory", "ownership:\n  - {path: .git/, owner: .}\n", "execution.ownership"},
		{"areas differ by case", "ownership:\n  - {path: realization/, owner: .}\n  - {path: Realization/, owner: library}\n", "execution.ownership"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := executionResult(t, withOwnership(t, tc.ownership))
			if _, ok := result.Document(); ok || result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), tc.code) {
				t.Fatalf("invalid ownership accepted or wrong diagnostic: %s %+v", result.Status(), result.Diagnostics())
			}
		})
	}
	// The removed layout section is no longer part of the contract.
	removed := executionResult(t, withOwnership(t, "layout:\n  roots: [realization]\n"))
	if _, ok := removed.Document(); ok || !executionDiagnostic(removed.Diagnostics(), "execution.unknown_key") {
		t.Fatalf("the removed layout section was accepted: %s %+v", removed.Status(), removed.Diagnostics())
	}
}

func TestExecutionOwnershipDiagnosticsNameWhatIsWrong(t *testing.T) {
	for _, tc := range []struct{ ownership, want string }{
		{"ownership:\n  - {path: realization/, owner: .}\n  - {path: realization/, owner: library}\n", "equal specificity"},
		{"ownership:\n  - {path: .markitect/x/, owner: .}\n", ".markitect"},
		{"ownership:\n  - {path: realization/, owner: missing}\n", "missing"},
	} {
		result := executionResult(t, withOwnership(t, tc.ownership))
		found := false
		for _, d := range result.Diagnostics() {
			found = found || strings.Contains(d.Message, tc.want)
		}
		if !found {
			t.Errorf("no diagnostic mentions %q: %+v", tc.want, result.Diagnostics())
		}
	}
}

const libraryTeamYAML = `team:
  maxHelpers: 2
  helpers:
    - role: architect
      profile: library-agent
      writes: false
      instructions: [ops/architect.md]
    - role: worker
      writes: true
`
