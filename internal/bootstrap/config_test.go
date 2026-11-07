package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// markitect config sets a value at the project's level or the user's, and the
// project's beats the user's, which beats the built-in one, key by key (DEC-102).
// A change rewrites the agents' files of the project, and --list says where each
// value comes from.
func TestConfigSetsValuesAtTwoLevels(t *testing.T) {
	emptyHome(t)
	home := os.Getenv("HOME")
	r := newFolder(t)
	mustRun(t, r.Dir, "init")

	out := mustRun(t, r.Dir, "config", "--user", "agents.builder.model", "opus")
	wantAll(t, out, "set agents.builder.model to opus in "+filepath.Join(home, ".markitect", "harness.yaml"), "updated .claude/agents/markitect-builder.md")
	user := read(t, home, ".markitect/harness.yaml")
	wantAll(t, user, "# Markitect's harness configuration for all of your projects", "agents:\n  builder:\n    model: opus\n")
	wantAll(t, read(t, r.Dir, ".claude/agents/markitect-builder.md"), "\nmodel: opus\neffort: high\n")

	out = mustRun(t, r.Dir, "config", "agents.builder.effort", "max")
	wantAll(t, out, "set agents.builder.effort to max in .markitect/harness.yaml", "commit the change")
	wantAll(t, read(t, r.Dir, ".markitect/harness.yaml"), "# The harness configuration of this project", "effort: max\n")
	mustRun(t, r.Dir, "config", "agents.builder.model", "haiku")
	wantAll(t, read(t, r.Dir, ".claude/agents/markitect-builder.md"), "\nmodel: haiku\neffort: max\n")
	if got := strings.TrimSpace(mustRun(t, r.Dir, "config", "agents.builder.model")); got != "haiku" {
		t.Fatalf("the model in force: %q", got)
	}
	out = mustRun(t, r.Dir, "config", "--user", "agents.builder.model", "sonnet")
	wantAll(t, out, "note: this project's .markitect/harness.yaml sets agents.builder.model to haiku, which beats yours here")

	mustRun(t, r.Dir, "config", "strategy.default", "parallel")
	if got := strings.TrimSpace(mustRun(t, r.Dir, "config", "strategy.review")); got != "parallel" {
		t.Fatalf("a skill's strategy without its own: %q", got)
	}
	out = mustRun(t, r.Dir, "config", "--list")
	for _, line := range []string{
		"agents.builder.model     haiku     project (.markitect/harness.yaml)",
		"agents.builder.effort    max       project (.markitect/harness.yaml)",
		"agents.reviewer.model    sonnet    built-in",
		"strategy.default         parallel  project (.markitect/harness.yaml)",
		"strategy.review          parallel  project (.markitect/harness.yaml), via strategy.default",
	} {
		if !strings.Contains(out, line) {
			t.Errorf("lacks %q:\n%s", line, out)
		}
	}
	var doc struct {
		FormatVersion int    `json:"format_version"`
		Kind          string `json:"kind"`
		Values        []struct {
			Key, Value, Origin, File, Via string
		}
		Problems []string
	}
	if err := json.Unmarshal([]byte(mustRun(t, r.Dir, "config", "--list", "--json")), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Kind != "config" || doc.FormatVersion != 1 || len(doc.Problems) != 0 || doc.Values[0].Key != "agents.builder.model" || doc.Values[0].Origin != "project" {
		t.Fatalf("the JSON document: %+v", doc)
	}

	// A strategy names only a workflow some level defines.
	code, _, errOut := run(t, r.Dir, "config", "strategy.build", "quick")
	if code != 2 || !strings.Contains(errOut, "no harness.yaml defines") {
		t.Fatalf("named an undefined workflow: %d %s", code, errOut)
	}
	r.Write(".claude/workflows/quick.js", "export const meta = { name: \"quick\", description: \"mine\" }\n")
	out = mustRun(t, r.Dir, "config", "--user", "workflows.quick", ".claude/workflows/quick.js")
	if strings.Contains(out, "note:") {
		t.Fatalf("a note on a workflow that is in place:\n%s", out)
	}
	mustRun(t, r.Dir, "config", "strategy.build", "quick")
	out = mustRun(t, r.Dir, "config", "--user", "workflows.other", "flows/other.js")
	wantAll(t, out, "note: flows/other.js does not exist yet", "note: Claude Code starts a saved workflow by name only from .claude/workflows/")

	// Removing the workflow leaves a problem that --list reports.
	mustRun(t, r.Dir, "config", "--user", "--unset", "workflows.quick")
	code, out, _ = run(t, r.Dir, "config", "--list")
	if code != 1 || !strings.Contains(out, "problem: strategy.build names the workflow quick") {
		t.Fatalf("--list with a problem: %d\n%s", code, out)
	}
	mustRun(t, r.Dir, "config", "--unset", "strategy.build")
	if code, out, _ := run(t, r.Dir, "config", "--unset", "strategy.build"); code != 1 || !strings.Contains(out, "sets no strategy.build") {
		t.Fatalf("a second unset: %d %s", code, out)
	}

	// Removing the project's values gives the user's back to the agents' files.
	mustRun(t, r.Dir, "config", "--unset", "agents.builder.model")
	mustRun(t, r.Dir, "config", "--unset", "agents.builder.effort")
	mustRun(t, r.Dir, "config", "--user", "agents.builder.model", "opus")
	wantAll(t, read(t, r.Dir, ".claude/agents/markitect-builder.md"), "\nmodel: opus\neffort: high\n")
	if out := mustRun(t, r.Dir, "update", "--check", "--files"); !strings.Contains(out, "project: up to date") {
		t.Fatalf("the agents' files do not match the configuration:\n%s", out)
	}
	if _, out, _ := run(t, r.Dir, "check"); strings.Contains(out, "harness") {
		t.Fatalf("check speaks of the configuration:\n%s", out)
	}
}

// Bad keys and values are usage errors, and nothing is written; outside a project
// only the user's level can change.
func TestConfigRefusesWhatItDoesNotTake(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	before := snapshot(t, r.Dir)
	for _, args := range [][]string{
		{"config", "agents.builder.model", "opus; rm -rf /"},
		{"config", "agents.builder.effort", "huge"},
		{"config", "agents.tester.model", "opus"},
		{"config", "colour", "blue"},
		{"config", "strategy.build", "Not A Name"},
		{"config", "workflows.parallel", "x.js"},
		{"config", "--list", "agents.builder.model"},
		{"config", "--json", "agents.builder.model"},
		{"config", "a", "b", "c"},
		{"config"},
		{"config", "colour"},
	} {
		if code, _, _ := run(t, r.Dir, args...); code != 2 {
			t.Errorf("markitect %s: exit %d, want 2", strings.Join(args, " "), code)
		}
	}
	if diff := changed(before, snapshot(t, r.Dir)); len(diff) > 0 {
		t.Fatalf("a refused change wrote %v", diff)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".markitect")); !os.IsNotExist(err) {
		t.Fatalf("a refused change made the user's folder: %v", err)
	}

	empty := newFolder(t)
	if code, _, errOut := run(t, empty.Dir, "config", "agents.builder.model", "opus"); code != 2 || !strings.Contains(errOut, "--user") {
		t.Fatalf("set a project value outside a project: %d %s", code, errOut)
	}
	mustRun(t, empty.Dir, "config", "--user", "agents.builder.model", "opus")
	if out := mustRun(t, empty.Dir, "config", "--list"); !strings.Contains(out, "opus") || strings.Contains(out, "updated") {
		t.Fatalf("--list outside a project:\n%s", out)
	}

	// A harness.yaml the tool cannot read stops update and check, not status.
	r.Write(".markitect/harness.yaml", "agents: [builder]\n")
	if code, _, errOut := run(t, r.Dir, "update", "--files"); code != 2 || !strings.Contains(errOut, ".markitect/harness.yaml") {
		t.Fatalf("update with a broken configuration: %d %s", code, errOut)
	}
	if code, out, _ := run(t, r.Dir, "check"); code != 1 || !strings.Contains(out, "harness.syntax") {
		t.Fatalf("check with a broken configuration: %d %s", code, out)
	}
}

// init installs the parallel workflow and the config skill with the harness, and
// update restores the workflow and removes one the binary no longer ships.
func TestInitInstallsTheParallelWorkflow(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	script := read(t, r.Dir, ".claude/workflows/markitect-parallel.js")
	if !strings.HasPrefix(script, "export const meta = {\n  name: \"markitect-parallel\",") {
		t.Fatalf("the workflow's meta:\n%.200s", script)
	}
	for _, want := range []string{`agentType: "markitect-builder", isolation: "worktree"`, `agentType: "markitect-reviewer"`, `agentType: "markitect-auditor"`, "const STALE_ROUNDS = 3", "markitect status"} {
		if !strings.Contains(script, want) {
			t.Errorf("the workflow lacks %q", want)
		}
	}
	for _, banned := range []string{"Date.now", "Math.random", "new Date()", "import("} {
		if strings.Contains(script, banned) {
			t.Errorf("the workflow uses %s, which the runtime refuses", banned)
		}
	}
	skill := read(t, r.Dir, ".claude/skills/markitect-config/SKILL.md")
	if !strings.Contains(skill, "name: markitect-config\n") || strings.Contains(skill, "disable-model-invocation") {
		t.Fatalf("the config skill:\n%s", skill)
	}
	for _, s := range []string{"build", "review", "cleanup", "bughunt", "audit"} {
		text := read(t, r.Dir, ".claude/skills/markitect-"+s+"/SKILL.md")
		if !strings.Contains(text, "markitect config strategy."+s) || !strings.Contains(text, "markitect-parallel") || !strings.Contains(text, `"mode": "`+s+`"`) {
			t.Errorf("the %s skill does not choose its strategy:\n%s", s, text)
		}
	}
	r.Write(".claude/workflows/markitect-parallel.js", "changed\n")
	r.Write(".claude/workflows/markitect-retired.js", "no longer shipped\n")
	r.Write(".claude/workflows/mine.js", "the owner's\n")
	out := mustRun(t, r.Dir, "update", "--files")
	wantAll(t, out, ".claude/workflows/markitect-parallel.js", ".claude/workflows/markitect-retired.js")
	if read(t, r.Dir, ".claude/workflows/markitect-parallel.js") != script || read(t, r.Dir, ".claude/workflows/mine.js") != "the owner's\n" {
		t.Fatal("update did not restore the workflow, or touched the owner's")
	}
	if _, err := os.Stat(filepath.Join(r.Dir, ".claude", "workflows", "markitect-retired.js")); !os.IsNotExist(err) {
		t.Fatalf("the retired workflow stayed: %v", err)
	}
	if _, out, _ := run(t, r.Dir, "check"); strings.Contains(out, "markitect-parallel") || !strings.Contains(out, "mine.js") {
		t.Fatalf("the workflow needs no registry entry, the owner's does:\n%s", out)
	}
}
