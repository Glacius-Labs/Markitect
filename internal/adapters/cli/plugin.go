package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/run"
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
			if err == nil && guardsEveryTool(hooks) {
				return true
			}
		}
	}
	return false
}

// guardsEveryTool says whether a plugin's hooks run the guard before the shell
// and before every tool that writes a file or starts an agent. An older plugin,
// whose hooks ran it only before some of them, does not count.
func guardsEveryTool(data []byte) bool {
	var h struct {
		Hooks struct {
			PreToolUse []struct {
				Matcher string `json:"matcher"`
				Hooks   []struct {
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
	}
	if json.Unmarshal(data, &h) != nil {
		return false
	}
	covered := map[string]bool{}
	for _, group := range h.Hooks.PreToolUse {
		guards := false
		for _, hook := range group.Hooks {
			guards = guards || strings.Contains(hook.Command, "hook pretool")
		}
		if !guards {
			continue
		}
		for _, tool := range strings.Split(group.Matcher, "|") {
			covered[strings.TrimSpace(tool)] = true
		}
	}
	for _, tool := range strings.Split("Bash|"+run.EditTools+"|"+run.AgentTools, "|") {
		if !covered[tool] {
			return false
		}
	}
	return true
}
