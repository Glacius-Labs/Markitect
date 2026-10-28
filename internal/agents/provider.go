package agents

import (
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

// Provider is one agent CLI's launch contract: how its login is copied into
// the private home, how a turn is started, and how its output is read. The
// Claude Code and Codex adapters implement it; the agent runtime knows no
// provider.
type Provider interface {
	// Name is the provider's name as the evaluation names it: claude or codex.
	Name() string
	// Login admits the operator's login file and makes the Manager's copy:
	// account login only, never an API key, and without refresh authority.
	// An expired access token is refused.
	Login(source []byte, now time.Time) (Login, error)
	// Home is the private initialization agentd installs before the first
	// provider start: the login copy and, where the CLI takes them as files,
	// the harness's system documents.
	Home(Login, harness.Document) (frame.Initialize, error)
	// AuthStatus is the command that reports the login the CLI would use;
	// CheckAuth reads its standard output and refuses anything but the
	// account login.
	AuthStatus() (argv, env []string)
	CheckAuth(stdout []byte) error
	// Turn starts one turn: its arguments, environment additions and
	// standard input.
	Turn(Turn) (argv, env []string, stdin []byte, err error)
	// Reader reads the standard output of one turn.
	Reader(Turn) Reader
}

// Login is a Manager's copy of the account login. Secrets are the values
// that never appear in a transcript or an error.
type Login struct {
	Copy    []byte
	Secrets [][]byte
	Expires time.Time
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
	// Tools are the Manager tools the turn must offer, by name.
	Tools    []string
	Deadline time.Time
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
}

// Usage counts one turn's tokens as the provider reports them.
type Usage struct {
	Input, CacheCreation, CacheRead, Output int64
}

// Add sums two counts.
func (u Usage) Add(v Usage) Usage {
	return Usage{u.Input + v.Input, u.CacheCreation + v.CacheCreation, u.CacheRead + v.CacheRead, u.Output + v.Output}
}
