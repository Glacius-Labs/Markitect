package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// The two subscription login modes. There is no API-key mode.
const (
	// ModeCredentials is a Claude Code credentials file of a normal account
	// login (.credentials.json with claudeAiOauth), mounted read-only.
	ModeCredentials = "subscription-credentials"
	// ModeToken is the long-lived subscription token of `claude setup-token`
	// in a file, mounted read-only.
	ModeToken = "subscription-token"
)

// RuntimeCredential contains only explicitly selected subscription login
// bytes. Refresh authority is not installed in a Manager copy.
type RuntimeCredential struct {
	Mode  string
	Login []byte
}
type CredentialResolver func(context.Context, string) (RuntimeCredential, error)

// login is one validated subscription login, prepared for one Manager.
type login struct {
	mode        string
	credentials []byte // the access-only credentials copy for the private home
	token       string // the subscription token for the provider environment
	method      string // the authMethod `claude auth status` must report
	secrets     [][]byte
}

// apiKeyVariables name the environment variables through which Claude Code
// would authenticate with an API key or route to another provider instead of
// the subscription login.
var apiKeyVariables = []string{
	"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_AWS_API_KEY", "ANTHROPIC_FOUNDRY_API_KEY",
	"ANTHROPIC_FOUNDRY_AUTH_TOKEN", "AWS_BEARER_TOKEN_BEDROCK", "CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_VERTEX", "CLAUDE_CODE_USE_FOUNDRY",
}

// RefuseAPIKeys refuses an environment that sets an API-key or provider
// variable, and names the first one with the reason. The Claude adapter uses
// subscription login only; the value is never read into the error.
func RefuseAPIKeys(environ []string) error {
	for _, entry := range environ {
		name, value, _ := strings.Cut(entry, "=")
		for _, refused := range apiKeyVariables {
			if strings.EqualFold(name, refused) && value != "" {
				return fmt.Errorf("%w: %s is set in the Apply environment; the Claude adapter authenticates only with a Claude subscription login and never with an API key or another provider, so unset it", ErrSession, refused)
			}
		}
	}
	return nil
}

// apiKeyPrefix starts every Anthropic API key. A subscription login never
// carries one.
const apiKeyPrefix = "sk-ant-api"

// parseLogin validates the resolved credential and keeps its secrets for
// redaction. No private file is written until the Manager container has
// started and announced agentd.
func parseLogin(c RuntimeCredential, now time.Time) (login, error) {
	refuse := func(reason string) (login, error) {
		return login{}, fmt.Errorf("%w: %s", ErrSession, reason)
	}
	if len(c.Login) == 0 || len(c.Login) > frame.MaxPrivateAuthBytes {
		return refuse("the subscription login is empty or exceeds its byte limit")
	}
	switch c.Mode {
	case ModeCredentials:
		return parseCredentials(c.Login, now)
	case ModeToken:
		token := strings.TrimSpace(string(c.Login))
		if len(token) < 16 || len(token) > 4096 || strings.ContainsFunc(token, func(r rune) bool { return r <= ' ' || r > '~' }) {
			return refuse("the subscription token file must hold one token of printable characters")
		}
		if strings.HasPrefix(token, apiKeyPrefix) {
			return refuse("the token file holds an Anthropic API key; only a subscription token from claude setup-token is admitted")
		}
		return login{mode: ModeToken, token: token, method: "oauth_token", secrets: [][]byte{[]byte(token)}}, nil
	}
	return refuse("only the subscription-credentials and subscription-token login modes are admitted")
}

// parseCredentials admits a Claude Code credentials file with a claude.ai
// account login. The Manager copy keeps only claudeAiOauth without its
// refresh token: this adapter neither refreshes nor rewrites the source, and
// other entries of the file, such as MCP server logins, never reach a Manager.
func parseCredentials(data []byte, now time.Time) (login, error) {
	refuse := func(reason string) (login, error) {
		return login{}, fmt.Errorf("%w: %s", ErrSession, reason)
	}
	file, err := protocolObject(data)
	if err != nil {
		return refuse("the credentials file is not one JSON object")
	}
	oauth, err := protocolObject(file["claudeAiOauth"])
	if err != nil {
		return refuse("the credentials file holds no claudeAiOauth account login")
	}
	access, err := protocolString(oauth["accessToken"])
	if err != nil || access == "" {
		return refuse("the account login has no access token")
	}
	if strings.HasPrefix(access, apiKeyPrefix) {
		return refuse("the account login holds an Anthropic API key; only a subscription login is admitted")
	}
	secrets := [][]byte{[]byte(access)}
	if value, ok := oauth["refreshToken"]; ok && string(bytes.TrimSpace(value)) != "null" {
		refresh, err := protocolString(value)
		if err != nil {
			return refuse("the account login has a malformed refresh token")
		}
		if refresh != "" {
			secrets = append(secrets, []byte(refresh))
		}
	}
	if value, ok := oauth["expiresAt"]; ok && string(bytes.TrimSpace(value)) != "null" {
		var expires json.Number
		d := json.NewDecoder(bytes.NewReader(value))
		d.UseNumber()
		if d.Decode(&expires) != nil {
			return refuse("the account login has a malformed expiry")
		}
		ms, err := expires.Int64()
		if err != nil {
			return refuse("the account login has a malformed expiry")
		}
		if !now.Before(time.UnixMilli(ms)) {
			return refuse("the account login's access token has expired; renew the login on the host (claude /login), because this adapter does not refresh it")
		}
	}
	delete(oauth, "refreshToken")
	copied, err := json.Marshal(map[string]map[string]json.RawMessage{"claudeAiOauth": oauth})
	if err != nil || len(copied) > frame.MaxPrivateAuthBytes {
		return refuse("the access-only credentials copy exceeds its byte limit")
	}
	return login{mode: ModeCredentials, credentials: copied, method: "claude.ai", secrets: secrets}, nil
}

// authMethod reads the authMethod of `claude auth status` JSON output.
func authMethod(data []byte) (string, error) {
	o, err := protocolObject(bytes.TrimSpace(data))
	if err != nil {
		return "", err
	}
	method, err := protocolString(o["authMethod"])
	if err != nil || method == "" {
		return "", ErrProtocol
	}
	return method, nil
}
