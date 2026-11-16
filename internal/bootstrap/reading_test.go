package bootstrap

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// readingSkills are the skills that explain the spec to a person and change
// nothing in the project (concept, §8).
var readingSkills = []string{"tour", "catchup", "explain"}

var (
	// skillCommand finds a markitect command a skill names in backticks.
	skillCommand = regexp.MustCompile("`markitect ([^`]+)`")
	// catchupItem is the line of markitect status --all that the catch-up
	// skill takes as one item: DEFINITION@FINGERPRINT, from the first column.
	catchupItem = regexp.MustCompile(`(?m)^[^ \n].*@[0-9a-f]{6}$`)
)

// Every markitect command the reading skills name runs on a project that
// compiles, the worked example of the architect skill, and is one that only
// reads: a flag they rely on that the tool drops fails here. The catch-up skill
// takes the items from the text of status --all by their lines
// DEFINITION@FINGERPRINT, each with its state, the first line under it but the
// one about its parts; those lines must be exactly the items status --all
// --json lists, whatever their state.
func TestTheReadingSkillsCommandsRunAndOnlyRead(t *testing.T) {
	emptyHome(t)
	data, err := os.ReadFile(examplePage)
	if err != nil {
		t.Fatal(err)
	}
	r := newFolder(t)
	files := exampleFiles(t, strings.ReplaceAll(string(data), "\r\n", "\n"))
	for path, text := range files {
		r.Write(".markitect/"+path, text)
		if strings.HasPrefix(path, "registry") {
			for _, row := range registryRow.FindAllStringSubmatch(text, -1) {
				r.Write(row[1], "x\n")
			}
		}
	}
	for _, line := range strings.Split(files["ignore"], "\n") {
		if line != "" && !strings.HasPrefix(line, "#") {
			r.Write(line, "x\n")
		}
	}
	mustRun(t, r.Dir, "tidy")

	// A Package of the example and one of its Definitions stand in for the
	// skills' FOLDER, TARGET and DEFINITION.
	var list struct {
		Packages []struct {
			Path        string
			Definitions []struct{ ID string }
		}
	}
	if err := json.Unmarshal([]byte(mustRun(t, r.Dir, "list", "--json")), &list); err != nil {
		t.Fatal(err)
	}
	folder, definition := "", ""
	for _, p := range list.Packages {
		if p.Path != "" && p.Path != "Root" && len(p.Definitions) > 1 {
			folder, definition = p.Path, p.Definitions[1].ID
			break
		}
	}
	if folder == "" {
		t.Fatal("the example has no Package with Definitions")
	}

	reads := map[string]bool{"status": true, "doc": true, "list": true, "vet": true, "inbox": true}
	for _, skill := range readingSkills {
		text, err := os.ReadFile("../skills/claude/skills/markitect-" + skill + "/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		commands := skillCommand.FindAllStringSubmatch(string(text), -1)
		if len(commands) == 0 {
			t.Fatalf("the %s skill names no markitect command", skill)
		}
		for _, m := range commands {
			line := strings.NewReplacer("FOLDER", folder, "TARGET", definition, "DEFINITION", definition).Replace(m[1])
			args := strings.Fields(line)
			if !reads[args[0]] || args[0] == "inbox" && len(args) > 1 && (args[1] == "add" || args[1] == "close") {
				t.Errorf("the %s skill names markitect %s, which does not only read", skill, args[0])
				continue
			}
			code, out, errOut := run(t, r.Dir, args...)
			if code != 0 && code != 1 {
				t.Errorf("the %s skill's markitect %s: exit %d\n%s%s", skill, line, code, out, errOut)
			}
			if strings.Contains(line, "--json") && !json.Valid([]byte(out)) {
				t.Errorf("the %s skill's markitect %s prints no JSON:\n%s", skill, line, out)
			}
		}
	}

	// The record holds an ok, a fail and a changed Definition, so that the
	// lines of status --all show more than one state.
	verifyAll(t, r.Dir)
	var st struct {
		Items []struct{ Definition, Fingerprint, State string }
	}
	items := func() {
		t.Helper()
		_, out, _ := run(t, r.Dir, "status", "--all", "--json")
		if err := json.Unmarshal([]byte(out), &st); err != nil || len(st.Items) < 3 {
			t.Fatalf("status --all --json: %v\n%s", err, out)
		}
	}
	items()
	failed := st.Items[0].Definition + "@" + st.Items[0].Fingerprint
	mustRun(t, r.Dir, "mark", "-m", "a reason", failed, "fail")
	changed := ".markitect/spec/" + definition + ".md"
	r.Write(changed, read(t, r.Dir, changed)+"\nOne sentence more.\n")
	items()
	want, states := map[string]bool{}, map[string]bool{}
	for _, it := range st.Items {
		want[it.Definition+"@"+it.Fingerprint] = true
		states[it.State] = true
	}
	if !states["ok"] || !states["fail"] || !states["changed"] {
		t.Fatalf("the fixture shows the states %v, want ok, fail and changed", states)
	}
	skill, err := os.ReadFile("../skills/claude/skills/markitect-catchup/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	// The six classes are written out: mawk 1.3.4, still shipped by some Linux
	// distributions, knows no {6}.
	for _, pattern := range []string{"/^[^ ].*@" + strings.Repeat("[0-9a-f]", 6) + "$/", "/^  with its /"} {
		if !strings.Contains(string(skill), pattern) {
			t.Fatalf("the catch-up skill no longer takes its items by %s, which this test follows", pattern)
		}
	}
	// As the skill's awk does: each item line with the first line under it
	// that is not the one about its parts, its state.
	_, text, _ := run(t, r.Dir, "status", "--all")
	keys, key := map[string]string{}, ""
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		switch {
		case catchupItem.MatchString(line):
			key = line
			keys[key] = ""
		case key != "" && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "  with its ") && keys[key] == "":
			keys[key] = strings.TrimPrefix(line, "  ")
		}
	}
	if len(keys) != len(want) {
		t.Fatalf("the catch-up skill would find %d items in status --all, and status --all --json lists %d:\n%s", len(keys), len(want), text)
	}
	for item, state := range keys {
		if !want[item] {
			t.Errorf("the catch-up skill would take %q for an item", item)
		}
		if state == "" {
			t.Errorf("the catch-up skill would find no state for %s", item)
		}
	}
	if !strings.HasPrefix(keys[failed], "fail: a reason") {
		t.Errorf("the catch-up skill would take %q for the state of %s", keys[failed], failed)
	}
}
