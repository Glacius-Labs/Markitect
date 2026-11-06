package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

var update = flag.Bool("update", false, "rewrite the plugin's skill from the binary's personal skill")

// The Claude Code plugin lives in plugin/ and is listed by the marketplace in
// .claude-plugin/ at the root of the repository.
const (
	pluginDir      = "../../../plugin"
	marketplace    = "../../../.claude-plugin/marketplace.json"
	pluginManifest = pluginDir + "/.claude-plugin/plugin.json"
	pluginSkill    = pluginDir + "/skills/markitect/SKILL.md"
)

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
// with either, or both, is told the same thing. Run this test with -update to
// rewrite the plugin's copy.
func TestPluginSkillIsThePersonalSkill(t *testing.T) {
	want := skills.Personal()
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

// The plugin's version names the release whose binary it fetches. A release is cut
// from a commit where the source version and the plugin's version agree; between
// releases the source says -dev and the plugin still names the last release.
func TestPluginNamesAReleaseThatMatchesTheSource(t *testing.T) {
	var manifest struct {
		Name, Version string
	}
	readJSON(t, pluginManifest, &manifest)
	if manifest.Name != "markitect" {
		t.Errorf("the plugin is named %q, but the skill and the install command say markitect", manifest.Name)
	}
	if buildinfo.IsDev(manifest.Version) {
		t.Fatalf("the plugin names the development version %s, which has no binary to fetch", manifest.Version)
	}
	if _, err := buildinfo.Newer(manifest.Version, "0.0.0"); err != nil {
		t.Fatalf("the plugin's version is no release version: %v", err)
	}
	if source := buildinfo.Version(); !buildinfo.IsDev(source) && source != manifest.Version {
		t.Fatalf("the source is released as %s but the plugin names %s; a release commit sets both", source, manifest.Version)
	}
}

func TestMarketplaceListsThePlugin(t *testing.T) {
	var m struct {
		Name    string
		Plugins []struct {
			Name, Source, Version string
		}
	}
	readJSON(t, marketplace, &m)
	if m.Name != "markitect" || len(m.Plugins) != 1 {
		t.Fatalf("the marketplace is %+v, want markitect with one plugin", m)
	}
	p := m.Plugins[0]
	if p.Name != "markitect" || p.Source != "./plugin" {
		t.Errorf("the entry is %+v, want the plugin markitect in ./plugin (the install command is markitect@markitect)", p)
	}
	if p.Version != "" {
		t.Errorf("the entry sets the version %s as well as plugin.json, which Claude Code reports as a mismatch", p.Version)
	}
}

// The hooks file is plain JSON of a fixed shape, and its hooks name the launcher.
func TestPluginHooksRunTheLauncher(t *testing.T) {
	var h struct {
		Hooks map[string][]struct {
			Hooks []struct{ Command string }
		}
	}
	readJSON(t, pluginDir+"/hooks/hooks.json", &h)
	for _, event := range []string{"SessionStart", "PreToolUse"} {
		groups := h.Hooks[event]
		if len(groups) != 1 || len(groups[0].Hooks) != 1 {
			t.Fatalf("%s: %+v, want one hook", event, groups)
		}
		if c := groups[0].Hooks[0].Command; !bytes.Contains([]byte(c), []byte("${CLAUDE_PLUGIN_ROOT}/bin/markitect")) {
			t.Errorf("%s runs %q, not the launcher", event, c)
		}
	}
}
