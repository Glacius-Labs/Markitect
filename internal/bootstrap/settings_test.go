package bootstrap

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// secretToken stands for the user's token from `claude setup-token`.
const secretToken = "sk-ant-oat01-SECRET-VALUE-1234"

// Settings read the agents' settings from the environment and say whether each
// provider has its account token, and show no part of the token.
func TestDesktopSettingsReadTheAgentsAndTheTokens(t *testing.T) {
	for _, v := range []string{"MARKITECT_PROVIDER", "MARKITECT_MODEL", "MARKITECT_EFFORT", "MARKITECT_MANAGER_IMAGE", "MARKITECT_NETWORK"} {
		t.Setenv(v, "")
	}
	t.Setenv("MARKITECT_ENGINE", "/usr/bin/docker")
	t.Setenv(TokenVariable, secretToken)

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
	if claude.Provider != "claude" || claude.State != "ok" || !claude.Used || claude.File != TokenVariable || claude.Command != "claude setup-token" || claude.Expires != nil {
		t.Errorf("claude %+v", claude)
	}
	// Codex has no token route, so it is refused whatever the environment holds.
	if codex.Provider != "codex" || codex.State != "refused" || codex.Used || !strings.Contains(codex.Detail, "no account token route") {
		t.Errorf("codex %+v", codex)
	}
	if out, _ := json.Marshal(s); strings.Contains(string(out), secretToken) || strings.Contains(string(out), "SECRET") {
		t.Fatalf("the token reached the settings: %s", out)
	}
}

func TestDesktopSettingsNameAMissingTokenAsAReasonNoAgentStarts(t *testing.T) {
	for _, v := range []string{"MARKITECT_PROVIDER", "MARKITECT_MODEL", "MARKITECT_EFFORT", "MARKITECT_MANAGER_IMAGE", "MARKITECT_NETWORK"} {
		t.Setenv(v, "")
	}
	t.Setenv("MARKITECT_ENGINE", "/usr/bin/docker")
	t.Setenv(TokenVariable, "")

	s := desktopSettings(context.Background())
	if c := s.Logins[0]; c.State != "missing" || !c.Used {
		t.Errorf("claude without a token %+v", c)
	}
	if p := s.Agents.Problems; len(p) != 2 || !strings.Contains(p[1], "claude setup-token") || !strings.Contains(p[1], TokenVariable) {
		t.Errorf("problems %v", p)
	}
}

func TestDesktopSettingsRefuseCodexAsTheUsedProvider(t *testing.T) {
	for _, v := range []string{"MARKITECT_MODEL", "MARKITECT_EFFORT", "MARKITECT_MANAGER_IMAGE", "MARKITECT_NETWORK"} {
		t.Setenv(v, "")
	}
	t.Setenv("MARKITECT_ENGINE", "/usr/bin/docker")
	t.Setenv(TokenVariable, secretToken)
	t.Setenv("MARKITECT_PROVIDER", "codex")

	s := desktopSettings(context.Background())
	if s.Agents.Model != "gpt-6-luna" || !s.Logins[1].Used || s.Logins[1].State != "refused" || s.Logins[0].Used {
		t.Errorf("codex used: %+v", s)
	}
	// The used provider's refusal is a reason no agent starts.
	if p := s.Agents.Problems; len(p) != 2 || !strings.Contains(p[1], "codex") || !strings.Contains(p[1], "no account token route") {
		t.Errorf("problems %v", p)
	}
	if out, _ := json.Marshal(s); strings.Contains(string(out), secretToken) {
		t.Fatalf("the token reached the settings: %s", out)
	}
}

func TestDesktopSettingsRefuseAnUnknownProvider(t *testing.T) {
	t.Setenv("MARKITECT_PROVIDER", "gemini")
	t.Setenv("MARKITECT_ENGINE", "/usr/bin/docker")
	s := desktopSettings(context.Background())
	found := false
	for _, p := range s.Agents.Problems {
		found = found || strings.Contains(p, "gemini")
	}
	if !found {
		t.Errorf("problems %v", s.Agents.Problems)
	}
}
