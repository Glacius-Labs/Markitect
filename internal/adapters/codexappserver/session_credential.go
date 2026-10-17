package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
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

func (e *sessionEntry) credential(ctx context.Context, c RuntimeCredential, helper RuntimeHelper) ([]string, error) {
	if c.Mode == "api-key" {
		if c.APIKey == "" || c.RuntimeLogin != nil || strings.ContainsAny(c.APIKey, "\r\n\x00") {
			return nil, ErrSession
		}
		e.secrets = [][]byte{[]byte(c.APIKey)}
		return []string{"CODEX_HOME=/home/agent/.codex", "OPENAI_API_KEY=" + c.APIKey}, nil
	}
	if c.Mode != "dedicated-runtime-login" || c.APIKey != "" || len(c.RuntimeLogin) > maxProtocolMessage || helper == nil {
		return nil, ErrSession
	}
	var login runtimeLogin
	if strictValue(c.RuntimeLogin, &login) != nil || login.Mode != "chatgpt" || login.APIKey != nil || login.Tokens.Access == "" || login.Tokens.ID == "" || login.Tokens.Account == "" {
		return nil, ErrSession
	}
	for _, secret := range []string{login.Tokens.Access, login.Tokens.ID, login.Tokens.Refresh} {
		if secret != "" {
			e.secrets = append(e.secrets, []byte(secret))
		}
	}
	login.Tokens.Refresh = ""
	data, err := json.Marshal(login)
	if err != nil {
		return nil, ErrSession
	}
	// Mark a possible installation before the helper runs: a partial helper
	// failure may still leave an access token in the retained provider home.
	e.runtimeAttempted = true
	e.loginInstalled = true
	write := []string{"/bin/sh", "-c", "mkdir -p /home/agent/.codex && umask 077 && rm -f /home/agent/.codex/auth.json && cat > /home/agent/.codex/auth.json"}
	if err = helper(ctx, e.credentialSpec("issue"), write, bytes.NewReader(data), io.Discard); err != nil {
		return nil, e.safe(err)
	}
	return []string{"CODEX_HOME=/home/agent/.codex"}, nil
}

func (e *sessionEntry) credentialSpec(action string) container.ContainerSpec {
	spec := sessionSpec(e.plan, e.profile)
	spec.Name += "-auth-" + action
	spec.Workdir = "/home/agent"
	spec.Mounts = []container.Mount{{Volume: e.plan.ProviderHome, Target: "/home/agent"}}
	return spec
}
func (e *sessionEntry) removeCredential(ctx context.Context, helper RuntimeHelper) error {
	if !e.loginInstalled {
		return nil
	}
	if helper == nil {
		return ErrSession
	}
	if err := helper(ctx, e.credentialSpec("remove"), []string{"/bin/rm", "-f", "/home/agent/.codex/auth.json"}, nil, io.Discard); err != nil {
		return e.safe(err)
	}
	e.loginInstalled = false
	return nil
}
