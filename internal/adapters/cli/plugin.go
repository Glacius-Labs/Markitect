package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// pluginName is the start of the key under which Claude Code's settings enable
// the plugin, and its record of installed plugins lists it: markitect@ and the
// marketplace.
const pluginName = "markitect@"

// hookProject is the project of a hook's call: the folder Claude Code names to
// its hooks, or else the one the call runs in.
func hookProject(input []byte) string {
	if dir := os.Getenv("CLAUDE_PROJECT_DIR"); dir != "" {
		return dir
	}
	var in struct {
		Cwd string `json:"cwd"`
	}
	if json.Unmarshal(input, &in) == nil && in.Cwd != "" {
		return in.Cwd
	}
	return "."
}

// pluginGuards says whether the Claude Code plugin guards a project's sessions
// with hooks of its own: the last of the user's, the project's and the local
// settings that names the plugin enables it, and Claude Code's record lists it
// as installed with hooks that run the guard. Whatever it cannot read counts as
// no, so that a doubt costs a second run of the guard, never the guard.
func pluginGuards(project string) bool {
	config, err := claudeConfigDir()
	if err != nil {
		return false
	}
	enabled := map[string]bool{}
	for _, file := range []string{
		filepath.Join(config, "settings.json"),
		filepath.Join(project, ".claude", "settings.json"),
		filepath.Join(project, ".claude", "settings.local.json"),
	} {
		var s struct {
			EnabledPlugins map[string]json.RawMessage `json:"enabledPlugins"`
		}
		data, err := os.ReadFile(file)
		if err != nil || json.Unmarshal(data, &s) != nil {
			continue
		}
		for key, value := range s.EnabledPlugins {
			if strings.HasPrefix(key, pluginName) {
				enabled[key] = bytes.Equal(bytes.TrimSpace(value), []byte("true"))
			}
		}
	}
	var record struct {
		Plugins map[string]json.RawMessage `json:"plugins"`
	}
	data, err := os.ReadFile(filepath.Join(config, "plugins", "installed_plugins.json"))
	if err != nil || json.Unmarshal(data, &record) != nil {
		return false
	}
	type install struct {
		InstallPath string `json:"installPath"`
	}
	for key, on := range enabled {
		if !on {
			continue
		}
		var installs []install
		if json.Unmarshal(record.Plugins[key], &installs) != nil {
			var one install
			if json.Unmarshal(record.Plugins[key], &one) != nil {
				continue
			}
			installs = []install{one}
		}
		for _, in := range installs {
			if in.InstallPath == "" {
				continue
			}
			hooks, err := os.ReadFile(filepath.Join(in.InstallPath, "hooks", "hooks.json"))
			if err == nil && bytes.Contains(hooks, []byte("hook pretool")) {
				return true
			}
		}
	}
	return false
}
