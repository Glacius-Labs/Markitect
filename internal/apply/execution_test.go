package apply_test

import (
	"context"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func executionInputs(t *testing.T, content string) (apply.InputSelection, apply.RawInputs) {
	t.Helper()
	selection, raw := libraryInputs(t)
	censusFile(t, &raw, ".markitect/execution.yaml", snapshot.Executable, []byte(content))
	location := apply.SourceLocation{Boundary: "explicit-execution-source", Path: path(t, "acquired/project-settings.data")}
	file, err := snapshot.NewEntry(location.Path, snapshot.Executable, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	selection.Execution = []apply.SourceLocation{location}
	raw.Execution = []apply.SourceFile{{Location: location, File: file, RepositoryPath: path(t, ".markitect/execution.yaml")}}
	return selection, raw
}

func executionDiagnostic(diagnostics []apply.ExecutionDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestDocumentedExecutionDecodesWithoutInventingResolvedSettings(t *testing.T) {
	content := documentedExecution(t)
	selection, raw := executionInputs(t, content)
	stage := registryModel(t, selection, raw)
	result := stage.DecodeExecution()
	document, ok := result.Document()
	if result.Status() != model.Passed || !ok || len(result.Diagnostics()) != 0 {
		t.Fatalf("documented execution failed: %s %+v", result.Status(), result.Diagnostics())
	}
	budget := document.Budget
	if document.APIVersion.Value != "markitect.execution/v1" || document.Strategy.Value != "recursive-passes-v1" || budget.ExecutionTime.Value != 25*time.Minute || budget.TurnTimeout.Value != 10*time.Minute || budget.AgentTurns.Value != 120 || budget.Passes.Value != 12 {
		t.Fatal("declared budget changed")
	}
	if document.Defaults.Execute.Profile == nil || document.Defaults.Execute.Profile.Value != "library-agent" || document.Defaults.Execute.Profile.Source.Field != "/profiles/execute" || document.Defaults.Verify.Profile.Source.Field != "/profiles/verify" || document.Defaults.Execute.Instructions != nil || document.Defaults.Verify.Instructions != nil {
		t.Fatal("profile selection or instruction absence was invented")
	}
	if document.Packages == nil || len(document.Packages.Value) != 1 || document.Packages.Value[0].Package != "lending" || document.Packages.Value[0].Verify.Profile.Value != "library-agent" || document.Packages.Value[0].Execute.Profile != nil || document.Ownership != nil || document.Team != nil || document.Scope != nil || document.Checks == nil || len(document.Checks.Value) != 1 || document.Checks.Value[0].RequiredBy.Value != (model.Identity{Package: "library", ID: "callable-interface"}) {
		t.Fatal("authored selections changed")
	}
	if budget.AgentTurns.Source.ContentID != raw.Execution[0].File.ContentID() || budget.AgentTurns.Source.File != ".markitect/execution.yaml" || budget.AgentTurns.Source.Field != "/budget/agentTurns" || !reflect.DeepEqual(result.Model().Inputs(), stage.Inputs()) || string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content {
		t.Fatal("original source/binding changed")
	}
	source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
		return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: fixtureHead}, nil
	}}
	preview, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	prepared := preview.DecodeExecution()
	if prepared.Status() != model.Passed || prepared.Model().Inputs().Revision != fixtureHead {
		t.Fatalf("fixed preview decoding failed: %+v", prepared.Diagnostics())
	}
}

func TestExecutionFieldTypeFailureRetainsKnownAuthoredValues(t *testing.T) {
	content := strings.Replace(documentedExecution(t), "agentTurns: 120", "agentTurns: many", 1)
	selection, raw := executionInputs(t, content)
	result := registryModel(t, selection, raw).DecodeExecution()
	if _, ok := result.Document(); ok || result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), "execution.integer") {
		t.Fatalf("bad budget passed: %+v", result.Diagnostics())
	}
	partial := result.Partial()
	if partial.Budget.AgentTurns.Source.File != "" || partial.Budget.Passes.Value != 12 || partial.Budget.TurnTimeout.Value != 10*time.Minute || string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content {
		t.Fatal("known values or malformed original bytes discarded")
	}
	for _, diagnostic := range result.Diagnostics() {
		if diagnostic.Code == "execution.integer" && (diagnostic.Source.Field != "/budget/agentTurns" || diagnostic.Source.ContentID != raw.Execution[0].File.ContentID() || diagnostic.Position == nil) {
			t.Fatal("type error lost exact source coordinates")
		}
	}
}

// libraryOwnershipYAML is the Library's declared ownership: Root owns the
// artifact tree, each Package its own directory below it, and the callable
// interface's executable location belongs to the library Package.
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

func withTeam(t *testing.T, team string) string {
	t.Helper()
	return strings.Replace(documentedExecution(t), "checks:\n", team+"checks:\n", 1)
}

func TestExecutionTeamDecodesWithSources(t *testing.T) {
	result := executionResult(t, withTeam(t, libraryTeamYAML))
	document, ok := result.Document()
	if !ok || document.Team == nil {
		t.Fatalf("team failed: %s %+v", result.Status(), result.Diagnostics())
	}
	team := document.Team
	if team.MaxHelpers.Value != 2 || team.MaxHelpers.Source.Field != "/team/maxHelpers" || len(team.Helpers) != 2 {
		t.Fatalf("team changed: %+v", team)
	}
	architect, worker := team.Helpers[0], team.Helpers[1]
	if architect.Role.Value != "architect" || architect.Role.Source.Field != "/team/helpers/0/role" ||
		architect.Profile == nil || architect.Profile.Value != "library-agent" || architect.Profile.Source.Field != "/team/helpers/0/profile" ||
		architect.Writes.Value || architect.Writes.Source.Field != "/team/helpers/0/writes" ||
		architect.Instructions == nil || !reflect.DeepEqual(architect.Instructions.Value, []string{"ops/architect.md"}) || architect.Instructions.Source.Field != "/team/helpers/0/instructions" {
		t.Errorf("architect or its sources changed: %+v", architect)
	}
	if worker.Role.Value != "worker" || !worker.Writes.Value || worker.Writes.Source.Field != "/team/helpers/1/writes" || worker.Profile != nil || worker.Instructions != nil {
		t.Errorf("an absent profile or instruction list was invented: %+v", worker)
	}
	absent, _ := executionResult(t, documentedExecution(t)).Document()
	if absent.Team != nil {
		t.Fatalf("an absent team was invented: %+v", absent.Team)
	}
	none := executionResult(t, withTeam(t, "team:\n  maxHelpers: 1\n  helpers: []\n"))
	if document, ok := none.Document(); !ok || document.Team == nil || len(document.Team.Helpers) != 0 {
		t.Fatalf("an explicitly empty helper list was refused: %s %+v", none.Status(), none.Diagnostics())
	}
}

func TestExecutionTeamRefusals(t *testing.T) {
	for _, tc := range []struct{ name, team, code string }{
		{"team not a mapping", "team: [worker]\n", "execution.mapping"},
		{"unknown key", "team:\n  maxHelpers: 1\n  helpers: []\n  size: 3\n", "execution.unknown_key"},
		{"missing maxHelpers", "team:\n  helpers: []\n", "execution.field_missing"},
		{"missing helpers", "team:\n  maxHelpers: 1\n", "execution.field_missing"},
		{"helpers not a list", "team:\n  maxHelpers: 1\n  helpers: worker\n", "execution.list"},
		{"maxHelpers not positive", "team:\n  maxHelpers: 0\n  helpers: []\n", "execution.positive"},
		{"missing role", "team:\n  maxHelpers: 1\n  helpers:\n    - {writes: true}\n", "execution.field_missing"},
		{"missing writes", "team:\n  maxHelpers: 1\n  helpers:\n    - {role: worker}\n", "execution.field_missing"},
		{"writes is not a boolean", "team:\n  maxHelpers: 1\n  helpers:\n    - {role: worker, writes: 'yes'}\n", "execution.boolean"},
		{"writes is a number", "team:\n  maxHelpers: 1\n  helpers:\n    - {role: worker, writes: 1}\n", "execution.boolean"},
		{"role manager", "team:\n  maxHelpers: 1\n  helpers:\n    - {role: manager, writes: true}\n", "execution.helper_role"},
		{"role reviewer", "team:\n  maxHelpers: 1\n  helpers:\n    - {role: reviewer, writes: false}\n", "execution.helper_role"},
		{"role not a name", "team:\n  maxHelpers: 1\n  helpers:\n    - {role: Worker, writes: true}\n", "execution.helper_role"},
		{"role with a path", "team:\n  maxHelpers: 1\n  helpers:\n    - {role: ../worker, writes: true}\n", "execution.helper_role"},
		{"duplicate role", "team:\n  maxHelpers: 2\n  helpers:\n    - {role: worker, writes: true}\n    - {role: worker, writes: false}\n", "execution.helper_role"},
		{"unknown helper key", "team:\n  maxHelpers: 1\n  helpers:\n    - {role: worker, writes: true, budget: 5}\n", "execution.unknown_key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := executionResult(t, withTeam(t, tc.team))
			if _, ok := result.Document(); ok || result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), tc.code) {
				t.Fatalf("invalid team accepted or wrong diagnostic: %s %+v", result.Status(), result.Diagnostics())
			}
		})
	}
}

// teamSettings resolves the effective team of every Manager of the Library
// fixture with the given execution document.
func teamSettings(t *testing.T, content string) organization.Settings {
	t.Helper()
	selection, raw := fileCheckInputs(t)
	rewriteFileCheckSource(t, &raw, &raw.Execution[0], content)
	addRuntimeDocument(t, &selection, &raw, testRuntimeDocument)
	m := registryModel(t, selection, raw)
	agent, check := runtimePorts(t)
	result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
	settings, ok := result.Settings()
	if err != nil || !ok || result.Status() != model.Passed {
		t.Fatalf("settings unavailable: %v %s %+v", err, result.Status(), result.Diagnostics())
	}
	return settings
}

func TestTeamDefaultsApplyWhenTheProjectDeclaresNoTeam(t *testing.T) {
	settings := teamSettings(t, documentedExecution(t))
	for _, name := range []string{".", "lending", "lending/reminders", "library"} {
		manager, found := settings.Manager(name)
		if !found {
			t.Fatalf("no settings for %s", name)
		}
		if manager.MaxHelpers != 1 || len(manager.Helpers) != 1 {
			t.Fatalf("%s: the product default is one worker at a time: %+v", name, manager)
		}
		worker := manager.Helpers[0]
		if worker.Role != "worker" || !worker.Writes || worker.Profile.Value.Name != manager.Execute.Profile.Value.Name {
			t.Errorf("%s: the default worker writes and runs on the Manager's Execute profile: %+v", name, worker)
		}
	}
}

func TestDeclaredTeamReplacesTheProductDefault(t *testing.T) {
	content := withTeam(t, "team:\n  maxHelpers: 3\n  helpers:\n    - {role: architect, profile: library-agent, writes: false}\n    - {role: tester, writes: true}\n")
	// The Package override gives lending another Execute profile; a helper
	// without a profile follows its own Manager's.
	settings := teamSettings(t, content)
	for _, name := range []string{".", "lending", "library"} {
		manager, _ := settings.Manager(name)
		if manager.MaxHelpers != 3 || len(manager.Helpers) != 2 || manager.Helpers[0].Role != "architect" || manager.Helpers[0].Writes || manager.Helpers[1].Role != "tester" || !manager.Helpers[1].Writes {
			t.Fatalf("%s: declared team not resolved: %+v", name, manager)
		}
		if manager.Helpers[0].Profile.Value.Name != "library-agent" || manager.Helpers[1].Profile.Value.Name != manager.Execute.Profile.Value.Name {
			t.Errorf("%s: helper profiles: %+v", name, manager.Helpers)
		}
	}
	// A declared empty team removes the default worker.
	settings = teamSettings(t, withTeam(t, "team:\n  maxHelpers: 1\n  helpers: []\n"))
	if manager, _ := settings.Manager("lending"); len(manager.Helpers) != 0 {
		t.Fatalf("a declared empty team kept the product worker: %+v", manager.Helpers)
	}
}
