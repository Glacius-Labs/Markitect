// Package codex is the Codex adapter of the agent runtime: the launch
// contract of the pinned Codex CLI's exec mode in a Manager container, the
// access-only copy of the ChatGPT account login, and the reader of its JSON
// events with the failure classes the evaluation voids.
package codex

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

// Version is the pinned CLI the image installs.
const Version = "0.162.0"

const (
	executable = "/usr/local/bin/codex"
	home       = "/home/agent"
	codexHome  = home + "/.codex"
)

// switches keep everything but the Manager tools out: no apps, memories,
// multi-agent work or other native delegation, no project documents, the
// file login store, and /work untrusted. They match the baseline arm's
// switches (tools/comparison/runner.py).
var switches = []string{
	"--disable", "apps", "--disable", "memories", "--disable", "multi_agent", "--disable", "multi_agent_v2",
	"-c", "agents.enabled=false", "--disable", "image_generation", "--disable", "browser_use", "--disable", "computer_use",
	"-c", `cli_auth_credentials_store="file"`, "-c", `model_provider="openai"`,
	"-c", "project_doc_max_bytes=0", "-c", "project_doc_fallback_filenames=[]",
	"-c", `projects."/work".trust_level="untrusted"`,
	"-c", `mcp_servers.markitect.command="/usr/local/bin/markitect-agentd"`, "-c", `mcp_servers.markitect.args=["mcp"]`,
	"--ignore-user-config", "--ignore-rules", "--dangerously-bypass-approvals-and-sandbox",
}

// Provider is the Codex adapter.
type Provider struct{}

func (Provider) Name() string { return "codex" }

// Home installs only the login; Codex receives the harness with the first
// message.
func (Provider) Home(l agents.Login, _ harness.Document) (frame.Initialize, error) {
	return frame.Initialize{CodexAuth: l.Copy}, nil
}

func (Provider) AuthStatus() ([]string, []string) {
	return []string{executable, "-c", `cli_auth_credentials_store="file"`, "login", "status"}, environment()
}

// CheckAuth requires the ChatGPT account login.
func (Provider) CheckAuth(stdout []byte) error {
	status := strings.TrimSpace(string(stdout))
	if status != "Logged in using ChatGPT" {
		return fmt.Errorf("codex login status reports %q; only the ChatGPT account login is admitted", firstLine(status))
	}
	return nil
}

// Turn starts one exec turn: the first creates the thread and carries the
// harness's role, repository and model before the task; later ones resume
// the thread. The message arrives on standard input.
func (Provider) Turn(t agents.Turn) ([]string, []string, []byte, error) {
	if !t.First && t.Conversation == "" {
		return nil, nil, nil, errors.New("codex: a later turn needs the thread ID")
	}
	if !oneOf(t.Effort, "minimal", "low", "medium", "high", "xhigh") || t.Model == "" || strings.ContainsAny(t.Model, " \"\n") {
		return nil, nil, nil, fmt.Errorf("codex: model %q with effort %q is not a launch setting", t.Model, t.Effort)
	}
	argv := []string{executable, "exec"}
	if !t.First {
		argv = append(argv, "resume")
	}
	argv = append(argv, "--json")
	argv = append(argv, switches...)
	argv = append(argv, "-m", t.Model, "-c", `model_reasoning_effort="`+t.Effort+`"`,
		"-c", "mcp_servers.markitect.tool_timeout_sec="+strconv.FormatInt(int64(toolTimeout(t.Deadline, time.Now()).Seconds()), 10))
	if !t.First {
		argv = append(argv, t.Conversation)
	}
	argv = append(argv, "-")
	message := t.Message
	if t.First {
		message = t.Document.Base + "\n\n" + t.Document.Context + "\n\n" + t.Message
	}
	return argv, environment(), []byte(message), nil
}

func environment() []string {
	return []string{"HOME=" + home, "CODEX_HOME=" + codexHome, "NODE_OPTIONS="}
}

func toolTimeout(deadline, now time.Time) time.Duration {
	if deadline.IsZero() {
		return 10 * time.Minute
	}
	return max(deadline.Sub(now), 0) + time.Minute
}

// Login admits a Codex auth.json with a ChatGPT account login. The
// Manager's copy keeps the ID token, access token and account, and an empty
// refresh token: the pinned CLI requires the field but the Manager can then
// neither refresh nor rotate the login. An API key is refused, and so is an
// access token past its expiry.
func (Provider) Login(source []byte, now time.Time) (agents.Login, error) {
	var file struct {
		APIKey *string `json:"OPENAI_API_KEY"`
		Tokens *struct {
			ID      string `json:"id_token"`
			Access  string `json:"access_token"`
			Refresh string `json:"refresh_token"`
			Account string `json:"account_id"`
		} `json:"tokens"`
		LastRefresh *string `json:"last_refresh"`
	}
	if len(source) == 0 || len(source) > frame.MaxPrivateAuthBytes || json.Unmarshal(source, &file) != nil {
		return agents.Login{}, errors.New("codex: the login is not a Codex auth.json")
	}
	if file.APIKey != nil && *file.APIKey != "" {
		return agents.Login{}, errors.New("codex: the login holds an OpenAI API key; only a ChatGPT account login is admitted")
	}
	t := file.Tokens
	if t == nil || t.Access == "" || t.ID == "" {
		return agents.Login{}, errors.New("codex: the login holds no ChatGPT account tokens")
	}
	expires, err := expiry(t.Access)
	if err != nil {
		return agents.Login{}, err
	}
	if !now.Before(expires) {
		return agents.Login{}, errors.New("codex: the login's access token has expired; run codex on the host to refresh it")
	}
	login := agents.Login{Expires: expires, Secrets: [][]byte{[]byte(t.Access), []byte(t.ID)}}
	if t.Refresh != "" {
		login.Secrets = append(login.Secrets, []byte(t.Refresh))
	}
	copied, err := json.Marshal(map[string]any{
		"OPENAI_API_KEY": nil,
		"tokens":         map[string]string{"id_token": t.ID, "access_token": t.Access, "refresh_token": "", "account_id": t.Account},
		"last_refresh":   file.LastRefresh,
	})
	if err != nil {
		return agents.Login{}, err
	}
	login.Copy = copied
	return login, nil
}

// expiry reads the exp claim of a JWT access token.
func expiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, errors.New("codex: the access token is no JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return time.Time{}, errors.New("codex: the access token is no JWT")
	}
	var claims struct {
		Exp json.Number `json:"exp"`
	}
	d := json.NewDecoder(bytes.NewReader(payload))
	d.UseNumber()
	if d.Decode(&claims) != nil {
		return time.Time{}, errors.New("codex: the access token has no readable claims")
	}
	exp, err := claims.Exp.Int64()
	if err != nil || exp <= 0 {
		return time.Time{}, errors.New("codex: the access token has no expiry")
	}
	return time.Unix(exp, 0), nil
}

func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
