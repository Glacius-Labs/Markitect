// Package claude is the Claude Code adapter of the agent runtime: the launch
// contract of the pinned CLI on the user's token, and the reader of its
// stream-json output with its failure classes. Its launch
// lessons come from the first native qualification of V1's Claude adapter (K1b).
package claude

import (
	"bytes"
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

// Version is the pinned CLI the image installs and every launch must report.
const Version = "2.1.308"

// Fixed container paths. agentd's private initialization writes the login
// and the two system prompt files into the private home.
const (
	executable       = "/usr/local/bin/claude"
	home             = "/home/agent"
	configDir        = home + "/.claude"
	systemPrompt     = home + "/.markitect/system-prompt.md"
	appendPrompt     = home + "/.markitect/append-system-prompt.md"
	builtinToolNames = "Bash,Read,Edit,Write,Glob,Grep"
)

// launchSettings turns off the twelve plugins the pinned CLI registers; an
// offline launch otherwise loaded two of them (K1b).
const launchSettings = `{"enabledPlugins":{"cc-plugin-agents-md@builtin":false,"cc-plugin-claude-test@builtin":false,"cc-plugin-diff@builtin":false,"cc-plugin-easel@builtin":false,"cc-plugin-mermaid@builtin":false,"cc-plugin-mods-guide@builtin":false,"cc-plugin-plugin-authoring@builtin":false,"cc-plugin-responsive-mode@builtin":false,"cc-plugin-sec-default@builtin":false,"cc-plugin-telemetry@builtin":false,"cc-plugin-tips@builtin":false,"cc-plugin-you-should-know@builtin":false}}`

// mcpConfig names the session helper as a chat's only MCP server.
const mcpConfig = `{"mcpServers":{"markitect":{"type":"stdio","command":"/usr/local/bin/markitect-agentd","args":["mcp"]}}}`

const (
	// tokenVariable is where the CLI reads a `claude setup-token` token from.
	tokenVariable = "CLAUDE_CODE_OAUTH_TOKEN"
	// tokenAuthMethod is how `claude auth status` names a login by that token.
	tokenAuthMethod = "oauth_token"
)

// Provider is the Claude Code adapter.
type Provider struct{}

func (Provider) Name() string { return "claude" }

// TokenVariable is the variable the CLI reads a `claude setup-token` token from.
func (Provider) TokenVariable() string { return tokenVariable }

// Home installs the harness: Base replaces Claude Code's system prompt and
// Context is appended to it.
func (Provider) Home(d harness.Document) (frame.Initialize, error) {
	home := &frame.ClaudeHome{SystemPrompt: d.Base, AppendSystemPrompt: d.Context}
	if len(d.Base)+len(d.Context) > frame.MaxPayload-(64<<10) {
		return frame.Initialize{}, fmt.Errorf("claude: the harness exceeds one private setup frame (%d bytes); the chat's context is too large", frame.MaxPayload)
	}
	return frame.Initialize{Claude: home}, nil
}

func (Provider) AuthStatus(home string) ([]string, []string) {
	return []string{executable, "auth", "status"}, statusEnvironment(home)
}

// CheckAuth requires the account login, through the token or a sign-in, and
// never an API key.
func (Provider) CheckAuth(stdout []byte) error {
	var status struct {
		LoggedIn   bool   `json:"loggedIn"`
		AuthMethod string `json:"authMethod"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout), &status); err != nil {
		return errors.New("claude auth status printed no readable status")
	}
	if !status.LoggedIn || (status.AuthMethod != "claude.ai" && status.AuthMethod != tokenAuthMethod) {
		return fmt.Errorf("claude auth status reports the method %q; only the account login is admitted", status.AuthMethod)
	}
	return nil
}

// Turn starts one non-interactive response. The first creates the
// conversation; every later one resumes it. Project settings, CLAUDE.md,
// memory, plugins, skills, slash commands, other MCP servers and native
// sub-agents stay out, and native tools run without prompts: the container
// is the sandbox.
func (Provider) Turn(t agents.Turn) ([]string, []string, []byte, error) {
	if !t.First && t.Conversation == "" {
		return nil, nil, nil, errors.New("claude: a later turn needs the conversation's session ID")
	}
	if !oneOf(t.Effort, "low", "medium", "high", "xhigh", "max") || t.Model == "" || strings.ContainsAny(t.Model, " \n") {
		return nil, nil, nil, fmt.Errorf("claude: model %q with effort %q is not a launch setting", t.Model, t.Effort)
	}
	argv := []string{executable, "--print", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose"}
	if !t.First {
		argv = append(argv, "--resume", t.Conversation)
	}
	argv = append(argv, "--model", t.Model, "--effort", t.Effort,
		"--system-prompt-file", systemPrompt, "--append-system-prompt-file", appendPrompt,
		"--permission-mode", "bypassPermissions", "--permission-prompts", "none",
		"--tools", builtinToolNames, "--disable-slash-commands", "--setting-sources", "user", "--settings", launchSettings,
		"--strict-mcp-config", "--mcp-config", mcpConfig)
	env := append(environment(), toolTimeout(t.Deadline, time.Now()))
	input, err := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": []map[string]string{{"type": "text", "text": t.Message}}}})
	if err != nil {
		return nil, nil, nil, err
	}
	return argv, env, append(input, '\n'), nil
}

// environment pins the private configuration directory and switches off
// CLAUDE.md and memory, background work, built-in sub-agents, checkpoints,
// tool deferral, updates and nonessential traffic. It carries no credential.
// Subprocess environment scrubbing stays off: the pinned CLI requires
// bubblewrap for it and refuses every command without it (K1b).
func environment() []string {
	return []string{
		"HOME=" + home,
		"NODE_OPTIONS=",
		"CLAUDE_CONFIG_DIR=" + configDir,
		"CLAUDE_CODE_DISABLE_CLAUDE_MDS=1",
		"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1",
		"CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1",
		"CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS=1",
		"CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING=1",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		"DISABLE_AUTOUPDATER=1",
		"ENABLE_TOOL_SEARCH=false",
	}
}

// statusEnvironment pins the user's own configuration folder and switches off
// CLAUDE.md and memory, checkpoints, tool deferral, updates and nonessential
// traffic. It carries no credential.
func statusEnvironment(home string) []string {
	return []string{
		"HOME=" + home,
		"NODE_OPTIONS=",
		"CLAUDE_CONFIG_DIR=" + home + "/.claude",
		"CLAUDE_CODE_DISABLE_CLAUDE_MDS=1",
		"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1",
		"CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING=1",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		"DISABLE_AUTOUPDATER=1",
		"ENABLE_TOOL_SEARCH=false",
	}
}

// toolTimeout lets a Manager tool call run until the turn's deadline: the
// host ends the turn first.
func toolTimeout(deadline, now time.Time) string {
	timeout := 10 * time.Minute
	if !deadline.IsZero() {
		timeout = max(deadline.Sub(now), 0) + time.Minute
	}
	return "MCP_TOOL_TIMEOUT=" + strconv.FormatInt(timeout.Milliseconds(), 10)
}

func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}
