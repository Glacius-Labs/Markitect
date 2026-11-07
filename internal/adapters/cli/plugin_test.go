package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

var update = flag.Bool("update", false, "rewrite the plugin's skill from the binary's personal skill")

// The Claude Code plugin lives in plugin/. The marketplace that lists it, and pins
// it to a release tag, is the repository Glacius-Labs/Claude-Plugins.
const (
	pluginDir      = "../../../plugin"
	pluginManifest = pluginDir + "/.claude-plugin/plugin.json"
	pluginSkill    = pluginDir + "/skills/" + pluginSkillName + "/SKILL.md"
)

// pluginSkillName is the name of the plugin's skill, and of its folder.
const pluginSkillName = "getting-started"

func readJSON(t *testing.T, file string, into any) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
}

// The plugin brings the same skill as markitect install writes, so that a session
// with either, or both, is told the same thing. Only the name differs: Claude Code
// names a plugin's skill plugin:skill, and markitect:markitect would repeat itself,
// so the plugin's skill is markitect:getting-started. Run this test with -update to
// rewrite the plugin's copy.
func TestPluginSkillIsThePersonalSkill(t *testing.T) {
	want := bytes.Replace(skills.Personal(), []byte("name: markitect\n"), []byte("name: "+pluginSkillName+"\n"), 1)
	if bytes.Equal(want, skills.Personal()) {
		t.Fatal("the personal skill has no name line to replace")
	}
	if *update {
		if err := os.WriteFile(pluginSkill, want, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(pluginSkill)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), want) {
		t.Fatalf("%s differs from the personal skill of this binary; run: go test ./internal/adapters/cli -run TestPluginSkill -update", filepath.ToSlash(pluginSkill))
	}
}

// The plugin's version is the version of the source, always: the release tag of
// the catalog pins the plugin to the commit of a release, where the launcher reads
// the version it fetches. Between releases both say -dev, and the launcher then
// refuses to fetch anything.
func TestPluginHasTheVersionOfTheSource(t *testing.T) {
	var manifest struct {
		Name, Version string
	}
	readJSON(t, pluginManifest, &manifest)
	if manifest.Name != "markitect" {
		t.Errorf("the plugin is named %q, but the skill and the install command say markitect", manifest.Name)
	}
	if source := buildinfo.Version(); source != manifest.Version {
		t.Fatalf("the source says %s but the plugin names %s; a release commit and the commit after it set both", source, manifest.Version)
	}
}

// The hooks file is plain JSON of a fixed shape, and its hooks name the launcher:
// one at the start of a session, one before shell commands and one before every
// tool that writes a file, as init writes them into a project's settings. The
// plugin's hooks filter in their own shell, without regard to case, rather than
// with Claude Code's rule, so that markitect.exe mark reaches the guard too.
func TestPluginHooksRunTheLauncher(t *testing.T) {
	var h struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []struct{ If, Command string }
		}
	}
	readJSON(t, pluginDir+"/hooks/hooks.json", &h)
	type group struct{ matcher, rule string }
	want := map[string][]group{
		"SessionStart": {{"startup|resume", ""}},
		"PreToolUse":   {{"Bash", ""}, {run.EditTools, ""}},
	}
	for event, groups := range want {
		got := h.Hooks[event]
		if len(got) != len(groups) {
			t.Fatalf("%s: %+v, want %d groups", event, got, len(groups))
		}
		for i, g := range groups {
			if got[i].Matcher != g.matcher || len(got[i].Hooks) != 1 {
				t.Fatalf("%s group %d: %+v, want one hook before %s", event, i, got[i], g.matcher)
			}
			hook := got[i].Hooks[0]
			if hook.If != g.rule {
				t.Errorf("%s group %d runs before %q, want %q", event, i, hook.If, g.rule)
			}
			if !bytes.Contains([]byte(hook.Command), []byte("${CLAUDE_PLUGIN_ROOT}/bin/markitect")) {
				t.Errorf("%s runs %q, not the launcher", event, hook.Command)
			}
		}
	}
}
