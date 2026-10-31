package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// Settings read the agents' settings from the environment and admit each
// login as an agent would, and show no part of it but its expiry.
func TestDesktopSettingsReadTheAgentsAndTheLogins(t *testing.T) {
	home := testkit.TempDir(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, v := range []string{"MARKITECT_PROVIDER", "MARKITECT_MODEL", "MARKITECT_LOGIN", "MARKITECT_MANAGER_IMAGE", "MARKITECT_NETWORK"} {
		t.Setenv(v, "")
	}
	t.Setenv("MARKITECT_ENGINE", "/usr/bin/docker")
	expires := time.Now().Add(3 * time.Hour).Truncate(time.Millisecond)
	raw, _ := json.Marshal(map[string]any{"claudeAiOauth": map[string]any{"accessToken": "sk-ant-oat01-SECRET", "refreshToken": "sk-ant-ort01-SECRET", "expiresAt": expires.UnixMilli()}})
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", ".credentials.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	s := desktopSettings(context.Background())
	if s.Agents.Provider != "claude" || s.Agents.Model != "claude-sonnet-5-5" || s.Agents.Effort != "high" || s.Agents.Network != providerNetwork || s.Agents.Engine != "/usr/bin/docker" {
		t.Errorf("agents %+v", s.Agents)
	}
	if len(s.Agents.Problems) != 1 || !strings.Contains(s.Agents.Problems[0], "MARKITECT_MANAGER_IMAGE") {
		t.Errorf("problems %v", s.Agents.Problems)
	}
	if len(s.Logins) != 2 {
		t.Fatalf("logins %+v", s.Logins)
	}
	claude, codex := s.Logins[0], s.Logins[1]
	if claude.Provider != "claude" || claude.State != "ok" || !claude.Used || claude.Expires == nil || !claude.Expires.Equal(expires) {
		t.Errorf("claude %+v", claude)
	}
	if codex.Provider != "codex" || codex.State != "missing" || codex.Used || codex.Command != "codex login" {
		t.Errorf("codex %+v", codex)
	}
	if out, _ := json.Marshal(s); strings.Contains(string(out), "SECRET") {
		t.Fatal("a token reached the settings")
	}

	// An API key is refused, with the provider's reason.
	raw, _ = json.Marshal(map[string]any{"claudeAiOauth": map[string]any{"accessToken": "sk-ant-api03-KEY"}})
	os.WriteFile(filepath.Join(home, ".claude", ".credentials.json"), raw, 0o600)
	t.Setenv("MARKITECT_PROVIDER", "codex")
	s = desktopSettings(context.Background())
	if c := s.Logins[0]; c.State != "refused" || !strings.Contains(c.Detail, "API key") || c.Used {
		t.Errorf("an API key %+v", c)
	}
	if s.Agents.Model != "gpt-6-luna" || !s.Logins[1].Used {
		t.Errorf("codex used: %+v", s)
	}
}
