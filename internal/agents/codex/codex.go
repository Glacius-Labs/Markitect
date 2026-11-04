// Package codex is the Codex adapter of the agent runtime: the launch contract
// of the pinned Codex CLI's exec mode and the reader of its JSON events with the
// failure classes the evaluation voids. It has no login route yet, so no session
// starts on it (DEC-093).
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

// TokenVariable is empty: Markitect has no token route for Codex yet. OpenAI's
// access tokens are described for ChatGPT Enterprise, and whether a personal
// plan gets one is not stated, so a Codex session cannot start (DEC-093).
func (Provider) TokenVariable() string { return "" }

// Home holds nothing to install without a login route.
func (Provider) Home(harness.Document) (frame.Initialize, error) {
	return frame.Initialize{}, errNoRoute
}

// Launch refuses: see TokenVariable.
func (Provider) Launch(agents.Launch) ([]string, []string, []byte, error) {
	return nil, nil, nil, errNoRoute
}

// SessionReader has nothing to read: see TokenVariable.
func (Provider) SessionReader(agents.Launch) agents.SessionReader { return nil }

var errNoRoute = errors.New("codex: Markitect has no login route for Codex yet")

// AuthStatus runs `codex login status` with its standard error on standard
// output: the pinned CLI prints its report on standard error (observed
// offline in the pinned image).
func (Provider) AuthStatus(string) ([]string, []string) {
	return []string{"/bin/sh", "-c", "exec " + executable + ` -c cli_auth_credentials_store='"file"' login status 2>&1`}, environment()
}

// CheckAuth requires the ChatGPT account login: one line of the report
// says so, and none names an API key.
func (Provider) CheckAuth(out []byte) error {
	chatGPT := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "api key") {
			return fmt.Errorf("codex login status reports %q; only the ChatGPT account login is admitted", line)
		}
		chatGPT = chatGPT || line == "Logged in using ChatGPT"
	}
	if !chatGPT {
		return fmt.Errorf("codex login status reports %q; only the ChatGPT account login is admitted", firstLine(strings.TrimSpace(string(out))))
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
