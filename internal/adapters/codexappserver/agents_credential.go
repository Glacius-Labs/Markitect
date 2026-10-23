package codexappserver

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

type runtimeLogin struct {
	Mode   string  `json:"auth_mode"`
	APIKey *string `json:"OPENAI_API_KEY"`
	Tokens struct {
		ID      string `json:"id_token"`
		Access  string `json:"access_token"`
		Refresh string `json:"refresh_token"`
		Account string `json:"account_id"`
	} `json:"tokens"`
	LastRefresh *string `json:"last_refresh"`
}

// credential validates the resolved credential, remembers its secrets for
// redaction and returns the provider environment. No private file is written
// until the Manager container has started and announced agentd.
func (t *turnRun) credential(ctx context.Context, c RuntimeCredential) ([]string, error) {
	if c.Mode == "api-key" {
		if c.APIKey == "" || c.RuntimeLogin != nil || strings.ContainsAny(c.APIKey, "\r\n\x00") {
			return nil, ErrSession
		}
		t.secrets = [][]byte{[]byte(c.APIKey)}
		return []string{"CODEX_HOME=/home/agent/.codex", "OPENAI_API_KEY=" + c.APIKey}, nil
	}
	if c.Mode != "dedicated-runtime-login" || c.APIKey != "" || len(c.RuntimeLogin) > frame.MaxPrivateAuthBytes {
		return nil, ErrSession
	}
	var login runtimeLogin
	if strictValue(c.RuntimeLogin, &login) != nil || login.Mode != "chatgpt" || login.APIKey != nil || login.Tokens.Access == "" || login.Tokens.ID == "" || login.Tokens.Account == "" {
		return nil, ErrSession
	}
	for _, secret := range []string{login.Tokens.Access, login.Tokens.ID, login.Tokens.Refresh} {
		if secret != "" {
			t.secrets = append(t.secrets, []byte(secret))
		}
	}
	return []string{"CODEX_HOME=/home/agent/.codex"}, nil
}

func (t *turnRun) installCredential(ctx context.Context, c RuntimeCredential) error {
	if c.Mode != "dedicated-runtime-login" {
		return nil
	}
	var login runtimeLogin
	if strictValue(c.RuntimeLogin, &login) != nil {
		return ErrSession
	}
	// This slice preserves the existing access-only copy. Safe account refresh
	// and fan-out remain subject to the separately authorized native probe.
	login.Tokens.Refresh = ""
	data, err := json.Marshal(login)
	if err != nil || len(data) > frame.MaxPrivateAuthBytes {
		return ErrSession
	}
	// A lost setup reply can still have installed a file; always attempt cleanup.
	t.loginInstalled = true
	if err := t.peer.Initialize(ctx, frame.Initialize{CodexAuth: data}); err != nil {
		return t.safe(err)
	}
	return nil
}

func (t *turnRun) removeCredential(ctx context.Context) error {
	if !t.loginInstalled {
		return nil
	}
	if t.peer == nil {
		return ErrSession
	}
	if err := t.peer.ClearPrivateHome(ctx); err != nil {
		return t.safe(err)
	}
	t.loginInstalled = false
	return nil
}
