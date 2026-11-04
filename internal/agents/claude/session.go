package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agents"
)

const (
	// tokenVariable is where the CLI reads a `claude setup-token` token from.
	tokenVariable = "CLAUDE_CODE_OAUTH_TOKEN"
	// tokenAuthMethod is how `claude auth status` names a login by that token.
	tokenAuthMethod = "oauth_token"
	// sessionTools are what an unattended session may use, its own subagents
	// included (concept 3.0 §11).
	sessionTools = "Bash,Read,Edit,Write,Glob,Grep,Agent,TodoWrite"
	// noShellTools are the same without one that runs a command. A session still
	// writes its note with Write.
	noShellTools = "Read,Write,Glob,Grep,Agent,TodoWrite"
)

// sessionSettings keep the provider's own extensions out of a session: the
// built-in plugins, and every hook whatever a settings file says.
var sessionSettings = strings.Replace(launchSettings, `{"enabledPlugins"`, `{"disableAllHooks":true,"enabledPlugins"`, 1)

// builtinAgents are the subagents the pinned CLI brings with it. A session that
// reports any other has loaded one from somewhere it must not.
var builtinAgents = []string{"claude", "Explore", "general-purpose", "Plan", "statusline-setup"}

// Launch starts one unattended session. Nothing of the repository's own reaches
// it: --setting-sources user reads neither the project's settings files nor its
// .mcp.json, --strict-mcp-config connects no MCP server, and the environment
// switches off CLAUDE.md and memory. Each user's configuration folder is its own
// and the Command made it. Every subagent runs on the session's model (concept
// 3.0 §11), and the reader ends a session that another model answers.
func (Provider) Launch(l agents.Launch) ([]string, []string, []byte, error) {
	if !oneOf(l.Effort, "low", "medium", "high", "xhigh", "max") || l.Model == "" || strings.ContainsAny(l.Model, " \n") {
		return nil, nil, nil, fmt.Errorf("claude: model %q with effort %q is not a launch setting", l.Model, l.Effort)
	}
	argv := []string{executable, "--print", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose"}
	if l.Conversation != "" {
		argv = append(argv, "--resume", l.Conversation)
	}
	tools := sessionTools
	if l.NoShell {
		tools = noShellTools
	}
	argv = append(argv, "--model", l.Model, "--effort", l.Effort,
		"--permission-mode", "bypassPermissions", "--permission-prompts", "none",
		"--tools", tools, "--disable-slash-commands", "--setting-sources", "user", "--settings", sessionSettings,
		"--strict-mcp-config")
	if l.Budget > 0 {
		argv = append(argv, "--max-budget-usd", strconv.FormatFloat(l.Budget, 'f', 2, 64))
	}
	env := append(sessionEnvironment(l.Home), "CLAUDE_CODE_SUBAGENT_MODEL="+l.Model, "CLAUDE_CODE_SUBAGENT_MODEL_FORCE=1", "ANTHROPIC_DEFAULT_HAIKU_MODEL="+l.Model)
	input, err := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": []map[string]string{{"type": "text", "text": l.Prompt}}}})
	if err != nil {
		return nil, nil, nil, err
	}
	return argv, env, append(input, '\n'), nil
}

// sessionEnvironment pins the user's own configuration folder and switches off
// CLAUDE.md and memory, checkpoints, tool deferral, updates and nonessential
// traffic. It carries no credential.
func sessionEnvironment(home string) []string {
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

// SessionReader reads one session's stream-json output: what the provider
// reported, every model that answered, and what the transcript shows.
func (Provider) SessionReader(l agents.Launch) agents.SessionReader {
	return &sessionReader{want: l.Model, resume: l.Conversation, models: map[string]bool{}, opened: map[string]bool{}}
}

type sessionReader struct {
	// want is the one model that may answer.
	want     string
	resume   string
	launched bool
	session  string
	model    string
	// lastError is the error category of the latest failed request.
	lastError string
	result    *line
	total     agents.Usage
	messages  map[string]agents.Usage
	models    map[string]bool
	opened    map[string]bool
	commands  []string
	subagents int
}

// sessionLine is what a session's line adds to the chat's projection of it.
type sessionLine struct {
	Message struct {
		Model   string `json:"model"`
		Content []struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			Input struct {
				File    string `json:"file_path"`
				Command string `json:"command"`
			} `json:"input"`
		} `json:"content"`
	} `json:"message"`
	ModelUsage map[string]struct {
		Input         int64 `json:"inputTokens"`
		Output        int64 `json:"outputTokens"`
		CacheRead     int64 `json:"cacheReadInputTokens"`
		CacheCreation int64 `json:"cacheCreationInputTokens"`
	} `json:"modelUsage"`
}

func (r *sessionReader) Line(raw []byte) error {
	var l line
	if err := json.Unmarshal(raw, &l); err != nil || l.Type == "" {
		return errors.New("claude: an output line is no stream-json message")
	}
	if !r.launched && (l.Type == "assistant" || l.Type == "user" || l.Type == "result") {
		return fmt.Errorf("claude: a %s message came before the launch reported its configuration", l.Type)
	}
	switch {
	case l.Type == "system" && l.Subtype == "init":
		if err := r.checkInit(l); err != nil {
			return err
		}
		r.launched, r.session, r.model = true, l.Session, l.Model
	case l.Type == "system" && l.Subtype == "api_retry":
		if l.Error != "" {
			r.lastError = l.Error
		}
	case l.Type == "assistant":
		// Subagents' messages count too, with their tool calls; only the main
		// conversation's clear an error.
		if l.Parent == nil || *l.Parent == "" {
			r.lastError = l.Error
		}
		var s sessionLine
		if err := json.Unmarshal(raw, &s); err != nil {
			return errors.New("claude: an assistant message cannot be read")
		}
		r.see(s)
		if m := s.Message.Model; m != "" && m != "<synthetic>" && m != r.want {
			return fmt.Errorf("claude: the model %s answered in a session on %s", m, r.want)
		}
		if l.Message.ID != "" {
			if r.messages == nil {
				r.messages = map[string]agents.Usage{}
			}
			r.messages[l.Message.ID] = maxUsage(r.messages[l.Message.ID], l.Message.Usage.count())
		}
	case l.Type == "result":
		r.result = &l
		var s sessionLine
		if err := json.Unmarshal(raw, &s); err != nil {
			return errors.New("claude: a result cannot be read")
		}
		// The per-model figures hold subagents too; the result's own usage
		// does not.
		r.total = agents.Usage{}
		for name, m := range s.ModelUsage {
			// Recorded only: the CLI's own background calls count here, and the
			// session has ended. The answers above are what must be on the model.
			r.models[name] = true
			r.total = r.total.Add(agents.Usage{Input: m.Input, CacheCreation: m.CacheCreation, CacheRead: m.CacheRead, Output: m.Output})
		}
		if len(s.ModelUsage) == 0 {
			r.total = l.Usage.count()
		}
	case l.Type == "control_request":
		return errors.New("claude: the CLI asked for a control answer, which this adapter never gives")
	}
	return nil
}

func (r *sessionReader) see(s sessionLine) {
	if m := s.Message.Model; m != "" && m != "<synthetic>" {
		r.models[m] = true
	}
	for _, c := range s.Message.Content {
		if c.Type != "tool_use" {
			continue
		}
		switch c.Name {
		case "Read":
			if c.Input.File != "" {
				r.opened[c.Input.File] = true
			}
		case "Bash":
			if c.Input.Command != "" {
				r.commands = append(r.commands, c.Input.Command)
			}
		case "Agent", "Task":
			r.subagents++
		}
	}
}

// checkInit requires the launch the adapter asked for, on every session: the
// account login and no API key, the pinned CLI, the permission mode, and nothing
// loaded from anywhere but the CLI itself: no MCP server or tool, plugin or
// skill, and no subagent it does not bring with it.
func (r *sessionReader) checkInit(l line) error {
	refuse := func(format string, args ...any) error {
		return fmt.Errorf("claude: the launch reported %s", fmt.Sprintf(format, args...))
	}
	switch {
	case l.APIKeySource != "none":
		return refuse("the API key source %q; only the account login (none) is admitted", l.APIKeySource)
	case l.Version != Version:
		return refuse("version %q instead of the pinned %s", l.Version, Version)
	case l.PermissionMode != "bypassPermissions":
		return refuse("permission mode %q", l.PermissionMode)
	case l.Model != r.want:
		return refuse("the model %q instead of %q", l.Model, r.want)
	case len(l.Skills)+len(l.Plugins)+len(l.SlashCommands) > 0:
		return refuse("skills, plugins or slash commands; none may load")
	case len(l.MCPServers) > 0:
		return refuse("the MCP server %q; none may load", l.MCPServers[0].Name)
	case r.resume != "" && l.Session != r.resume:
		return refuse("session %q instead of the resumed %q", l.Session, r.resume)
	}
	for _, a := range l.Agents {
		if !slices.Contains(builtinAgents, a) {
			return refuse("the subagent %q; only the CLI's own may load", a)
		}
	}
	for _, t := range l.Tools {
		if strings.HasPrefix(t, "mcp__") {
			return refuse("the MCP tool %q; none may load", t)
		}
	}
	return nil
}

func (r *sessionReader) Outcome() agents.Outcome {
	out := agents.Outcome{Conversation: r.session, Model: r.model, Usage: r.total}
	if r.result == nil {
		// Without a result, the assistant messages tell the input exactly;
		// their output counts are those streamed at each message's start.
		out.Usage = agents.Usage{Partial: true}
		for _, u := range r.messages {
			out.Usage = out.Usage.Add(u)
		}
	}
	switch {
	case r.result == nil:
		// The caller reports a session without a result.
	case !r.result.IsError && r.result.Subtype == "success":
		out.End, out.Text = agents.Answered, r.result.Result
	default:
		out.End = agents.Failed
		out.Class = classify(r.lastError, r.result.APIErrorStatus)
		detail := r.result.Result
		if detail == "" {
			detail = strings.Join(r.result.Errors, "; ")
		}
		if r.lastError != "" {
			detail = r.lastError + ": " + detail
		}
		if detail == "" {
			// A budget or turn limit ends a session without an error's words.
			detail = r.result.Subtype
		}
		out.Detail = detail
	}
	return out
}

// Seen tells what the transcript shows: every Read and shell command, the
// subagents' included.
func (r *sessionReader) Seen() agentport.Seen {
	opened := make([]string, 0, len(r.opened))
	for f := range r.opened {
		opened = append(opened, f)
	}
	sort.Strings(opened)
	return agentport.Seen{Measured: true, Opened: opened, Commands: r.commands, Subagents: r.subagents}
}

// Models lists every model that answered, subagents included.
func (r *sessionReader) Models() []string {
	out := make([]string, 0, len(r.models))
	for m := range r.models {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}
