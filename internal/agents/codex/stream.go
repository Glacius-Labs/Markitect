package codex

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/agents"
)

// Reader returns the reader of one exec turn's JSON events.
func (Provider) Reader(t agents.Turn) agents.Reader {
	return &reader{thread: t.Conversation, model: t.Model}
}

type reader struct {
	thread string
	// model is the model the turn was launched with: exec's events do not
	// name it, unlike Claude Code's system/init.
	model     string
	completed bool
	// failure is the error of a failed turn; lastError the latest error
	// event, which may be a reconnect the turn recovered from.
	failure, lastError string
	usage              agents.Usage
}

type event struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Message  string `json:"message"`
	Error    *struct {
		Message string `json:"message"`
	} `json:"error"`
	Item *struct {
		Type string `json:"type"`
	} `json:"item"`
	Usage *struct {
		Input       int64 `json:"input_tokens"`
		CachedInput int64 `json:"cached_input_tokens"`
		Output      int64 `json:"output_tokens"`
	} `json:"usage"`
}

func (r *reader) Line(raw []byte) error {
	var e event
	if err := json.Unmarshal(raw, &e); err != nil || e.Type == "" {
		return errors.New("codex: an output line is no exec event")
	}
	switch e.Type {
	case "thread.started":
		if r.thread != "" && e.ThreadID != r.thread {
			return errors.New("codex: the turn started another thread than the resumed one")
		}
		r.thread = e.ThreadID
	case "item.started", "item.updated", "item.completed":
		if e.Item != nil && strings.HasPrefix(e.Item.Type, "collab") {
			return errors.New("codex: the output holds native multi-agent work; delegation runs only through the Manager tree")
		}
	case "turn.completed":
		r.completed = true
		if e.Usage != nil {
			// Input tokens include the cached ones.
			r.usage = r.usage.Add(agents.Usage{Input: e.Usage.Input - e.Usage.CachedInput, CacheRead: e.Usage.CachedInput, Output: e.Usage.Output})
		}
	case "turn.failed":
		if e.Error != nil {
			r.failure = e.Error.Message
		}
	case "error":
		r.lastError = e.Message
	}
	return nil
}

func (r *reader) Outcome() agents.Outcome {
	out := agents.Outcome{Conversation: r.thread, Model: r.model, Usage: r.usage}
	// Codex reports usage only when a turn completes.
	out.Usage.Partial = !r.completed
	// A completed turn answered, whatever error events came before it.
	switch {
	case r.completed:
		out.End = agents.Answered
	case r.failure != "":
		out.End, out.Class, out.Detail = agents.Failed, classify(r.failure), r.failure
	case r.lastError != "":
		out.End, out.Class, out.Detail = agents.Failed, classify(r.lastError), r.lastError
	}
	return out
}

var (
	capacityText  = regexp.MustCompile(`(?i)at capacity|overloaded|\b529\b|slow.?down`)
	rateLimitText = regexp.MustCompile(`(?i)rate.?limit|usage.?limit|too many requests|\b429\b`)
	loginText     = regexp.MustCompile(`(?i)\b401\b|\b403\b|unauthorized|forbidden|token.?expired|refresh.?token|log ?in again|not logged in`)
	serverText    = regexp.MustCompile(`(?i)\b50[0-4]\b|internal server error|bad gateway|service unavailable|gateway timeout|server.?error|stream (error|disconnected)|connection (reset|refused|closed)`)
)

// classify sorts a failed turn by the error text the CLI reports after its
// own retries.
func classify(message string) agents.Class {
	switch {
	case capacityText.MatchString(message):
		return agents.Capacity
	case rateLimitText.MatchString(message):
		return agents.RateLimit
	case loginText.MatchString(message):
		return agents.LoginFailed
	case serverText.MatchString(message):
		return agents.ServerError
	}
	return agents.Other
}
