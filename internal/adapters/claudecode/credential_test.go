package claudecode

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

var credentialNow = time.UnixMilli(1_800_000_000_000)

// fixtureCredentials is a synthetic Claude Code credentials file: an account
// login with refresh authority and an unrelated MCP server login.
const fixtureCredentials = `{"claudeAiOauth":{"accessToken":"sk-ant-oat01-private-access","refreshToken":"sk-ant-ort01-private-refresh","expiresAt":1900000000000,"scopes":["user:inference","user:profile"],"subscriptionType":"max"},"mcpOAuth":{"server":{"accessToken":"private-mcp"}}}`

const fixtureToken = "sk-ant-oat01-private-long-lived-token"

func TestCredentialsCopyKeepsOnlyTheAccessLogin(t *testing.T) {
	l, err := parseLogin(RuntimeCredential{Mode: ModeCredentials, Login: []byte(fixtureCredentials)}, credentialNow)
	if err != nil {
		t.Fatal(err)
	}
	if l.mode != ModeCredentials || l.method != "claude.ai" || l.token != "" {
		t.Fatalf("login %+v", l)
	}
	var copied map[string]map[string]any
	if err := json.Unmarshal(l.credentials, &copied); err != nil || len(copied) != 1 {
		t.Fatalf("copy %s", l.credentials)
	}
	oauth := copied["claudeAiOauth"]
	if oauth["accessToken"] != "sk-ant-oat01-private-access" || oauth["subscriptionType"] != "max" || oauth["expiresAt"] != float64(1900000000000) {
		t.Fatalf("access login changed: %s", l.credentials)
	}
	if _, ok := oauth["refreshToken"]; ok || strings.Contains(string(l.credentials), "refresh") || strings.Contains(string(l.credentials), "mcp") {
		t.Fatalf("refresh authority or another login reached the Manager copy: %s", l.credentials)
	}
	if len(l.secrets) != 2 || string(l.secrets[0]) != "sk-ant-oat01-private-access" || string(l.secrets[1]) != "sk-ant-ort01-private-refresh" {
		t.Fatal("tokens are not remembered for redaction")
	}
}

func TestTokenLoginTravelsOnlyInTheProviderEnvironment(t *testing.T) {
	l, err := parseLogin(RuntimeCredential{Mode: ModeToken, Login: []byte(fixtureToken + "\n")}, credentialNow)
	if err != nil {
		t.Fatal(err)
	}
	if l.token != fixtureToken || l.credentials != nil || l.method != "oauth_token" || len(l.secrets) != 1 {
		t.Fatalf("token login %+v", l)
	}
	env := providerEnvironment(l.token)
	if env[len(env)-1] != "CLAUDE_CODE_OAUTH_TOKEN="+fixtureToken || !contains(env, "CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1") {
		t.Fatalf("environment %v", env)
	}
	for _, name := range environmentNames(env) {
		if strings.Contains(name, fixtureToken) {
			t.Fatal("environment evidence carries the token")
		}
	}
	if strings.Contains(strings.Join(providerEnvironment(""), " "), "OAUTH") {
		t.Fatal("a credentials-file login set a token variable")
	}
}

func TestLoginRefusesAPIKeysOtherModesMalformedAndExpired(t *testing.T) {
	expired := strings.Replace(fixtureCredentials, "1900000000000", "1700000000000", 1)
	for _, tc := range []struct {
		name   string
		cred   RuntimeCredential
		reason string
	}{
		{"API key mode", RuntimeCredential{Mode: "api-key", Login: []byte("sk-ant-api03-x")}, "login modes"},
		{"API key token", RuntimeCredential{Mode: ModeToken, Login: []byte("sk-ant-api03-private-key-value")}, "API key"},
		{"API key in credentials", RuntimeCredential{Mode: ModeCredentials, Login: []byte(`{"claudeAiOauth":{"accessToken":"sk-ant-api03-private"}}`)}, "API key"},
		{"console API key file", RuntimeCredential{Mode: ModeCredentials, Login: []byte(`{"primaryApiKey":"sk-ant-api03-private"}`)}, "claudeAiOauth"},
		{"no access token", RuntimeCredential{Mode: ModeCredentials, Login: []byte(`{"claudeAiOauth":{"refreshToken":"r"}}`)}, "access token"},
		{"expired", RuntimeCredential{Mode: ModeCredentials, Login: []byte(expired)}, "expired"},
		{"malformed expiry", RuntimeCredential{Mode: ModeCredentials, Login: []byte(`{"claudeAiOauth":{"accessToken":"a","expiresAt":"soon"}}`)}, "expiry"},
		{"duplicate keys", RuntimeCredential{Mode: ModeCredentials, Login: []byte(`{"claudeAiOauth":{"accessToken":"a","accessToken":"b"}}`)}, "JSON object"},
		{"token with spaces", RuntimeCredential{Mode: ModeToken, Login: []byte("two words in one token file")}, "printable"},
		{"empty", RuntimeCredential{Mode: ModeToken}, "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseLogin(tc.cred, credentialNow)
			if !errors.Is(err, ErrSession) || !strings.Contains(err.Error(), tc.reason) || strings.Contains(err.Error(), "private") {
				t.Fatalf("got %v, want a refusal naming %q without secrets", err, tc.reason)
			}
		})
	}
}

func TestAPIKeyEnvironmentIsRefusedByName(t *testing.T) {
	if err := RefuseAPIKeys([]string{"PATH=/usr/bin", "ANTHROPIC_API_KEY=", "CLAUDE_CODE_OAUTH_TOKEN=not-an-api-key"}); err != nil {
		t.Fatal("an empty variable or a subscription token is not an API key:", err)
	}
	for _, name := range apiKeyVariables {
		err := RefuseAPIKeys([]string{"HOME=/home/x", name + "=private-value"})
		if !errors.Is(err, ErrSession) || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "subscription") || strings.Contains(err.Error(), "private-value") {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestAuthStatusMethod(t *testing.T) {
	method, err := authMethod([]byte(`{"loggedIn":true,"authMethod":"claude.ai","email":"someone@example.invalid"}` + "\n"))
	if err != nil || method != "claude.ai" {
		t.Fatal(method, err)
	}
	for _, out := range []string{"", "Logged in", `{"loggedIn":false}`, `{"authMethod":""}`, `{"authMethod":1}`} {
		if _, err := authMethod([]byte(out)); err == nil {
			t.Fatalf("unreadable status admitted: %q", out)
		}
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
