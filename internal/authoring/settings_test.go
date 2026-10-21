package authoring

import (
	"strings"
	"testing"
)

const executionDoc = `apiVersion: markitect.execution/v1
strategy: direct
# The budget is an unvalidated starting value.
budget:
  executionTime: 25m # whole run
  turnTimeout: 10m
  agentTurns: 120
  passes: 12
profiles:
  execute: project-agent
  verify: project-agent
packages:
  lending:
    profiles: {execute: fast-agent}
    instructions: {execute: [docs/lending.md]}
ownership:
  - {path: ./, owner: .}
checks:
  - id: go-tests
    profile: go
    directory: .
    args: [test, ./...]
`

func TestEditSettingsChangesOnlyTheEditedFields(t *testing.T) {
	settings, err := readSettings([]byte(executionDoc))
	if err != nil {
		t.Fatal(err)
	}
	if settings.Budget.AgentTurns != 120 || settings.Profiles.Execute != "project-agent" || len(settings.Overrides) != 1 || settings.Overrides[0].Execute != "fast-agent" || settings.Team != nil {
		t.Fatalf("read settings = %+v", settings)
	}
	settings.Budget.AgentTurns = 60
	settings.Profiles.Verify = "review-agent"
	settings.Overrides = []PackageProfiles{{Package: "catalog", Verify: "review-agent"}}
	settings.Team = &Team{MaxHelpers: 2, Helpers: []Helper{{Role: "worker", Writes: true}, {Role: "architect", Profile: "review-agent", Instructions: []string{"docs/architect.md"}}}}
	out, err := editSettings([]byte(executionDoc), settings)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		"# The budget is an unvalidated starting value.", "executionTime: 25m # whole run", "agentTurns: 60", "verify: review-agent",
		"- {path: ./, owner: .}", "args: [test, ./...]", "instructions: {execute: [docs/lending.md]}",
		"catalog:", "profiles: {verify: review-agent}", "maxHelpers: 2", "- {role: worker, writes: true}",
		"- {role: architect, profile: review-agent, writes: false, instructions: [docs/architect.md]}",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("edited document lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "fast-agent") {
		t.Errorf("lending's profile override was not removed:\n%s", text)
	}
	again, err := readSettings(out)
	if err != nil || again.Budget.AgentTurns != 60 || again.Team == nil || len(again.Team.Helpers) != 2 || len(again.Overrides) != 1 || again.Overrides[0].Package != "catalog" {
		t.Fatalf("the edited document reads back as %+v, %v", again, err)
	}
	settings.Team = nil
	out, err = editSettings(out, settings)
	if err != nil || strings.Contains(string(out), "team:") {
		t.Fatalf("removing the team leaves %s, %v", out, err)
	}
}

func TestEditSettingsRemovesAnEmptyPackageEntry(t *testing.T) {
	doc := strings.Replace(executionDoc, "    instructions: {execute: [docs/lending.md]}\n", "", 1)
	settings, err := readSettings([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	settings.Overrides = nil
	out, err := editSettings([]byte(doc), settings)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "lending") || !strings.Contains(string(out), "packages: {}") {
		t.Fatalf("an emptied Package entry stays:\n%s", out)
	}
}

func TestDiffLinesCutsUnchangedRuns(t *testing.T) {
	var a, b []string
	for i := range 20 {
		a = append(a, "line "+string(rune('a'+i)))
	}
	b = append(b, a...)
	b[10] = "changed"
	lines := diffLines(a, b)
	var ops, gaps []string
	for _, l := range lines {
		if l.Gap > 0 {
			gaps = append(gaps, l.Op)
			continue
		}
		ops = append(ops, l.Op)
	}
	if strings.Join(ops, "") != "===-+===" || len(gaps) != 2 {
		t.Fatalf("diff = %+v", lines)
	}
	if lines[len(lines)-1].Gap != 6 || lines[0].Gap != 7 {
		t.Fatalf("gaps = %d and %d, want 7 and 6", lines[0].Gap, lines[len(lines)-1].Gap)
	}
}

// A field the owner did not change keeps its text: comments and block style of
// the team and of a Package's profiles survive an edit elsewhere.
func TestEditSettingsKeepsUnchangedFieldsVerbatim(t *testing.T) {
	doc := `apiVersion: markitect.execution/v1
strategy: direct
budget:
  executionTime: 25m
  turnTimeout: 10m
  agentTurns: 120
  passes: 12
profiles:
  execute: project-agent
  verify: project-agent
packages:
  lending:
    profiles:
      execute: fast-agent # the quick one
team:
  # one writer at a time
  maxHelpers: 1
  helpers:
    - role: worker
      writes: true
      instructions: []
`
	settings, err := readSettings([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	same, err := editSettings([]byte(doc), settings)
	if err != nil || string(same) != doc {
		t.Fatalf("an unchanged edit rewrites the document:\n%s %v", same, err)
	}
	settings.Budget.Passes = 9
	out, err := editSettings([]byte(doc), settings)
	if err != nil || string(out) != strings.Replace(doc, "passes: 12", "passes: 9", 1) {
		t.Fatalf("a budget edit changes more than the budget:\n%s %v", out, err)
	}
}
