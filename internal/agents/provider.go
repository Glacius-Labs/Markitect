package agents

import (
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

// Provider is one agent CLI's launch contract: which variable takes the user's
// token, how a session or turn is started, and how its output is read. This is
// the one place per provider sessions start from (DEC-092). The Claude Code and
// Codex adapters implement it; the agent runtime knows no provider.
type Provider interface {
	// Name is the provider's name as the evaluation names it: claude or codex.
	Name() string
	// TokenVariable is the variable the CLI reads the user's account token
	// from; empty when this provider has no such route yet.
	TokenVariable() string
	// Home is the private initialization agentd installs before a chat's first
	// provider start: the harness's system documents, where the CLI takes them
	// as files.
	Home(harness.Document) (frame.Initialize, error)
	// AuthStatus is the command that reports the login the CLI would use with
	// its home at home; CheckAuth reads its standard output and refuses
	// anything but the account login.
	AuthStatus(home string) (argv, env []string)
	CheckAuth(stdout []byte) error
	// Launch starts one unattended session of a Command: its arguments,
	// environment additions and standard input.
	Launch(Launch) (argv, env []string, stdin []byte, err error)
	// SessionReader reads the standard output of one session.
	SessionReader(Launch) SessionReader
	// Turn starts one turn: its arguments, environment additions and
	// standard input.
	Turn(Turn) (argv, env []string, stdin []byte, err error)
	// Reader reads the standard output of one turn.
	Reader(Turn) Reader
}

// Launch is one unattended session of a Command.
type Launch struct {
	// Home is the session user's home; the provider's own files go below it.
	Home string
	// Conversation names the session to continue; empty starts a fresh one.
	Conversation string
	Prompt       string
	Model        string
	Effort       string
	// Budget is the most the session may spend; zero sets no limit.
	Budget float64
}

// SessionReader reads one session's output and tells what it shows.
type SessionReader interface {
	Reader
	// Seen tells what the transcript shows the session did, subagents included.
	Seen() agentport.Seen
	// Models lists every model that answered, subagents included.
	Models() []string
}

// Turn is one provider turn: one message to the conversation and the
// provider's answer.
type Turn struct {
	// First is the conversation's first turn; Conversation names it for the
	// later ones (Claude: chosen by the adapter; Codex: reported by the CLI).
	First        bool
	Conversation string
	Message      string
	Document     harness.Document
	Model        string
	Effort       string
	Deadline     time.Time
}

// Reader reads one turn's output, line by line. Line returns an error for a
// line that breaks the launch contract, such as a native sub-agent, a launch
// with another login or an unchecked configuration; the turn then ends.
type Reader interface {
	Line(raw []byte) error
	// Outcome is the turn's end as the output reports it, read after the
	// process exited.
	Outcome() Outcome
}

// Progress is what one output line of a turn says while it runs: a message
// the provider wrote, and what it is doing now, in one line.
type Progress struct {
	Text, Doing string
}

// ProgressReader is a Reader that can also read a turn's progress, which a
// chat shows while the turn runs.
type ProgressReader interface {
	Reader
	Progress(raw []byte) Progress
}

// Doing words one activity for the owner: a verb and what it works on, a path
// relative to the copy, on one line. Whoever shows it shortens it, after any
// redaction: a secret cut in half is no longer recognized.
func Doing(verb, target string) string {
	target = strings.Join(strings.Fields(target), " ")
	target = strings.ReplaceAll(target, workDir+"/", "")
	return strings.TrimSpace(verb + " " + target)
}

// End says how a turn ended.
type End string

const (
	// Answered: the provider finished its turn.
	Answered End = "answered"
	// Failed: the provider or the launch contract failed; Class says which.
	Failed End = "failed"
	// Deadline: the turn reached its deadline and was stopped.
	Deadline End = "deadline"
	// Stopped: the caller cancelled the turn.
	Stopped End = "stopped"
)

// Class sorts a failed turn so that the evaluation can void provider
// failures (capacity, rate or usage limit, server error, refused login) and
// count everything else.
type Class string

const (
	Capacity    Class = "capacity"
	RateLimit   Class = "rate-limit"
	ServerError Class = "server-error"
	LoginFailed Class = "login"
	// Other is every failure that is not the provider's: the launch
	// contract, a crash, a protocol error. It is never voided.
	Other Class = "other"
)

// Outcome is one turn's end.
type Outcome struct {
	End   End
	Class Class // set when End is Failed
	// Detail is a short redacted reason.
	Detail string
	// Conversation names the provider conversation for the next turn.
	Conversation string
	// Model is the model the provider reported.
	Model string
	Usage Usage
	// Text is the provider's last answer of an answered turn.
	Text string
}

// Usage counts one turn's tokens as the provider reports them.
type Usage struct {
	Input, CacheCreation, CacheRead, Output int64
	// Partial: the provider reported no total for the turn, as for a turn
	// that was stopped, crashed or failed before its end, so the counts are
	// what its output showed until then and a lower bound.
	Partial bool
}

// Add sums two counts.
func (u Usage) Add(v Usage) Usage {
	return Usage{u.Input + v.Input, u.CacheCreation + v.CacheCreation, u.CacheRead + v.CacheRead, u.Output + v.Output, u.Partial || v.Partial}
}
