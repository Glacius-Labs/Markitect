package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/agents"
)

// Reader returns the reader of one turn's stream-json output.
func (Provider) Reader(t agents.Turn) agents.Reader {
	tools := strings.Split(builtinToolNames, ",")
	for _, name := range t.Tools {
		tools = append(tools, "mcp__markitect__"+name)
	}
	slices.Sort(tools)
	return &reader{tools: tools, resume: t.Conversation}
}

type reader struct {
	tools    []string
	resume   string
	launched bool
	session  string
	model    string
	// lastError is the error category of the latest failed request:
	// from an api_retry line or an assistant message; a later assistant
	// message without one clears it.
	lastError string
	result    *line
	usage     agents.Usage
	// messages holds each assistant message's usage, the largest count of
	// its lines: a turn without a result reports no total.
	messages map[string]agents.Usage
	// last is the latest assistant message, whose input is the
	// conversation's size.
	last string
}

// line is the projection of one stream-json line this reader consumes.
type line struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Session string `json:"session_id"`
	// system/init
	Model          string   `json:"model"`
	Version        string   `json:"claude_code_version"`
	APIKeySource   string   `json:"apiKeySource"`
	PermissionMode string   `json:"permissionMode"`
	Tools          []string `json:"tools"`
	Agents         []string `json:"agents"`
	Skills         []string `json:"skills"`
	SlashCommands  []string `json:"slash_commands"`
	Plugins        []struct {
		Name string `json:"name"`
	} `json:"plugins"`
	MCPServers []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"mcp_servers"`
	// assistant and user
	Parent  *string `json:"parent_tool_use_id"`
	Error   string  `json:"error"`
	Message struct {
		ID    string `json:"id"`
		Usage usage  `json:"usage"`
	} `json:"message"`
	// result
	IsError        bool     `json:"is_error"`
	Result         string   `json:"result"`
	Errors         []string `json:"errors"`
	APIErrorStatus int      `json:"api_error_status"`
	Usage          usage    `json:"usage"`
}

// usage is the token count of a result or of one assistant message.
type usage struct {
	Input         int64 `json:"input_tokens"`
	CacheCreation int64 `json:"cache_creation_input_tokens"`
	CacheRead     int64 `json:"cache_read_input_tokens"`
	Output        int64 `json:"output_tokens"`
}

// maxUsage keeps the larger count of each field: a message's later lines
// repeat its input and stream more output.
func maxUsage(a, b agents.Usage) agents.Usage {
	return agents.Usage{Input: max(a.Input, b.Input), CacheCreation: max(a.CacheCreation, b.CacheCreation), CacheRead: max(a.CacheRead, b.CacheRead), Output: max(a.Output, b.Output)}
}

func (u usage) count() agents.Usage {
	return agents.Usage{Input: u.Input, CacheCreation: u.CacheCreation, CacheRead: u.CacheRead, Output: u.Output}
}

func (r *reader) Line(raw []byte) error {
	var l line
	if err := json.Unmarshal(raw, &l); err != nil || l.Type == "" {
		return errors.New("claude: an output line is no stream-json message")
	}
	if l.Session != "" && r.session != "" && l.Session != r.session {
		return errors.New("claude: the output names another session")
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
	case l.Type == "assistant" || l.Type == "user":
		// tool_progress and other lines may name a running tool; only a
		// conversation message with a parent is a native sub-agent's.
		if l.Parent != nil && *l.Parent != "" {
			return errors.New("claude: the output holds native sub-agent work; delegation runs only through the Manager tree")
		}
		if l.Type == "assistant" {
			r.lastError = l.Error
			if l.Message.ID != "" {
				if r.messages == nil {
					r.messages = map[string]agents.Usage{}
				}
				r.messages[l.Message.ID] = maxUsage(r.messages[l.Message.ID], l.Message.Usage.count())
				r.last = l.Message.ID
			}
		}
	case l.Type == "result":
		r.result = &l
		r.usage = r.usage.Add(l.Usage.count())
	case l.Type == "control_request":
		return errors.New("claude: the CLI asked for a control answer, which this adapter never gives")
	}
	return nil
}

// checkInit requires the launch the adapter asked for, on every turn: the
// account login and no API key, the pinned CLI, the permission mode, exactly
// the built-in and Manager tools with the Manager tool server up, and no
// agents, skills, plugins or slash commands.
func (r *reader) checkInit(l line) error {
	refuse := func(format string, args ...any) error {
		return fmt.Errorf("claude: the launch reported %s", fmt.Sprintf(format, args...))
	}
	tools := slices.Clone(l.Tools)
	slices.Sort(tools)
	switch {
	case l.APIKeySource != "none":
		return refuse("the API key source %q; only the account login (none) is admitted", l.APIKeySource)
	case l.Version != Version:
		return refuse("version %q instead of the pinned %s", l.Version, Version)
	case l.PermissionMode != "bypassPermissions":
		return refuse("permission mode %q", l.PermissionMode)
	case len(l.Agents)+len(l.Skills)+len(l.Plugins)+len(l.SlashCommands) > 0:
		return refuse("agents, skills, plugins or slash commands; none may load")
	case !slices.Equal(tools, r.tools):
		return refuse("the tools %q instead of %q", l.Tools, r.tools)
	case r.resume != "" && l.Session != r.resume:
		return refuse("session %q instead of the resumed %q", l.Session, r.resume)
	}
	for _, s := range l.MCPServers {
		if s.Name == "markitect" && (s.Status == "connected" || s.Status == "pending") {
			return nil
		}
	}
	return refuse("no connected markitect tool server")
}

func (r *reader) Outcome() agents.Outcome {
	out := agents.Outcome{Conversation: r.session, Model: r.model, Usage: r.usage}
	if u, ok := r.messages[r.last]; ok {
		out.Context = u.Input + u.CacheCreation + u.CacheRead
	}
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
		// The host reports a turn without a result.
	case !r.result.IsError:
		out.End = agents.Answered
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
		out.Detail = detail
	}
	return out
}

// classify sorts a failed turn by the error category Claude Code names after
// its own retries, or by the HTTP status of the result.
func classify(category string, status int) agents.Class {
	switch {
	case category == "overloaded" || category == "overloaded_error" || status == 529:
		return agents.Capacity
	case category == "rate_limit" || category == "billing_error" || status == 429:
		return agents.RateLimit
	case category == "server_error" || category == "api_error" || category == "unknown" || status >= 500 && status < 600:
		return agents.ServerError
	case category == "authentication_failed" || status == 401 || status == 403:
		return agents.LoginFailed
	}
	return agents.Other
}
