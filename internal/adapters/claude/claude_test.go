package claude

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestBackgroundedLine(t *testing.T) {
	for _, line := range []string{"backgrounded · 2802d530\n  claude agents    list sessions\n", "backgrounded · 888951fa · owner session\n"} {
		m := backgrounded.FindStringSubmatch(line)
		if m == nil || len(m[1]) != 8 {
			t.Fatalf("%q: %v", line, m)
		}
	}
	if backgrounded.MatchString("Workspace not trusted.") {
		t.Fatal("matched an error")
	}
}

func TestLaunchFlags(t *testing.T) {
	flags := strings.Join(launchFlags(run.SessionSpec{Run: "r1"}, "s.json"), " ")
	for _, want := range []string{"--model sonnet", "--effort high", "--permission-mode dontAsk", "--settings s.json", "--setting-sources project", "--strict-mcp-config"} {
		if !strings.Contains(flags, want) {
			t.Errorf("%q not in %s", want, flags)
		}
	}
	if strings.Contains(flags, "--agent") {
		t.Errorf("a plain session names an agent: %s", flags)
	}
	flags = strings.Join(launchFlags(run.SessionSpec{Run: "r1", Agent: "markitect-reviewer", Model: "opus", Effort: "max"}, "s.json"), " ")
	for _, want := range []string{"--agent markitect-reviewer", "--model opus", "--effort max"} {
		if !strings.Contains(flags, want) {
			t.Errorf("%q not in %s", want, flags)
		}
	}
}

// The settings are what keeps a run unattended, bound and out of a remote.
func TestSettings(t *testing.T) {
	c := CLI{Executable: filepath.Join("C:", "Program Files", "markitect.exe")}
	data, err := json.Marshal(c.settings(run.SessionSpec{Run: "r1", Store: "C:/a b/markitect"}))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`"baseRef":"head"`, `"autoMemoryEnabled":false`, `"Bash"`, `"Bash(git push *)"`, `"Bash(gh *)"`,
		`"CLAUDE_CODE_SUBAGENT_MODEL":"sonnet"`, `"CLAUDE_CODE_SUBAGENT_MODEL_FORCE":"1"`, `"BASH_DEFAULT_TIMEOUT_MS"`,
		`"Stop"`, `"StopFailure"`, `"PreToolUse"`, `claudeMdExcludes`, `hook stop --store`, `hook pretool --store`, `--run r1`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("%q not in %s", want, text)
		}
	}
	// A path with a space survives the shell a hook runs in.
	if !strings.Contains(text, `\"C:/a b/markitect\"`) {
		t.Errorf("the store is not quoted: %s", text)
	}
}
