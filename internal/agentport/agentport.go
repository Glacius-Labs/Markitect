// Package agentport is the boundary between the Run engine and the agents it
// starts (concept 2.0 §6.2, §12): every working Manager has one maker and one
// checker for the whole Run, and the engine gives each a turn with its role,
// task, checklist, tools and deadline, and observes how the turn ends.
// Provider adapters implement Agents; the engine implements Host, the tools
// an agent may call during a turn.
//
// An agent's work reaches the engine only through Git: its submission, its
// checker's review and a parent's merge are commits on the Run's remote, which
// the engine reads after the turn ends. The tools carry messages along the
// tree and Shared Notes, never work. No call blocks on another agent.
package agentport

import (
	"context"
	"time"
)

// Mode is what the Run does.
type Mode string

const (
	// Apply: makers change the artifacts, checkers review, parents merge.
	Apply Mode = "apply"
	// Verify: checkers only; each writes its verdicts as a commit without
	// file changes.
	Verify Mode = "verify"
)

// Role is an agent's part in its Manager.
type Role string

const (
	Maker   Role = "maker"
	Checker Role = "checker"
)

// Agents runs agents. A Run's Manager and Role name one agent, which lives
// for the whole Run. Run gives that agent one turn for s: the first turn
// starts it, a later one continues its conversation, and an agent that had to
// end meanwhile starts again from the Run's remote. Run returns once the turn
// has ended: it finished, failed, reached s.Deadline or ctx ended. It never
// returns while the agent may still change anything. Run is called
// concurrently for different Managers, never for one agent twice at once.
type Agents interface {
	Run(ctx context.Context, s Start, host Host) End
	// Close ends every agent of the Run that still lives and removes what
	// they created. It returns what may remain; nothing when all is gone.
	Close(ctx context.Context, run string) []string
}

// Start is everything one turn of an agent needs.
type Start struct {
	Run  string
	Mode Mode
	Role Role
	// Manager is the Package the agent works for.
	Manager Manager
	// Round counts the turns of this role for this Manager in this Run,
	// from 1. A later turn follows a rejection, an answer, a parent's
	// assignment or a new submission, which Inbox and Task carry.
	Round int
	// Task says in the host's words what this turn must achieve.
	Task string
	// Event says in the host's words only what brought a later turn about,
	// such as a new submission with the required checks' results on it, or
	// the children's approved work; the review, the answer or the
	// settlement are messages in Inbox. A turn that continues an agent gets
	// Event and Inbox instead of Task, whose text and checklist its
	// conversation holds already; a turn that starts it gets Task. Empty
	// when Task has nothing to shorten.
	Event string
	// Checklist holds every item the Manager's verdicts must cover.
	Checklist []Item
	// Inbox holds the messages delivered with this turn, oldest first.
	Inbox []Message
	// Git is how the agent reaches the Run's remote.
	Git GitAccess
	// Tools are the host tools this turn may call; Host refuses others.
	Tools []Tool
	// Deadline bounds the turn; it must have ended by then.
	Deadline time.Time
}

// Manager is one working Manager of the Run.
type Manager struct {
	// ID is the Package path; Root's is ".".
	ID string
	// Parent is the parent's ID; empty for Root.
	Parent string
	// Children are the working children's IDs. Children that do not work in
	// this Run are not listed.
	Children []string
}

// Item parts of a checklist (concept §6.2).
const (
	Own       = "own"
	Inherited = "inherited"
	Finding   = "finding"
	Stale     = "stale"
)

// Item is one checklist item. Key is what a verdict trailer names; it holds
// no "|" and no line break.
type Item struct {
	Key  string
	Part string
	// Definition is the Definition the item is about; empty for stale
	// realization that names only a file.
	Definition string
	// Why says why the change reaches the item: the impact's reasons, a
	// Finding's observed and expected text, or the stale entry.
	Why string
}

// GitAccess is one agent's credential for the Run's remote. The token is
// secret: it reaches only the agent's own container.
type GitAccess struct {
	// URL is the repository on the Run's network.
	URL string
	// Network is the container network the remote is reachable on.
	Network string
	// Branch is the Manager's branch; every agent reads every branch.
	Branch string
	// Base is the full commit the Manager's work starts from: the Run's
	// start, or the state of its parent's branch it started from (concept
	// 2.0 §6.2). A checker's diff is from Base.
	Base  string
	User  string
	Token string
}

// Tool is a host tool an agent may call while it runs.
type Tool string

const (
	// ToolAsk asks the parent a question (Host.Ask).
	ToolAsk Tool = "ask"
	// ToolEscalate hands the parent a matter the Manager cannot settle
	// within the model; at Root it ends the Run as a model defect
	// (Host.Escalate).
	ToolEscalate Tool = "escalate"
	// ToolAnswer answers a child's question or escalation (Host.Answer).
	ToolAnswer Tool = "answer"
	// ToolAssign sends a child work, for example a correction after the
	// parent's review (Host.Assign).
	ToolAssign Tool = "assign"
	// ToolNote records a Shared Note (Host.Note).
	ToolNote Tool = "note"
	// ToolNotes reads the Run's Shared Notes (Host.Notes).
	ToolNotes Tool = "notes"
)

// Host is what an agent may call while it runs, for its own Manager. A call
// returns at once and never waits for another agent; an answer arrives with a
// later turn. An error is a refusal meant for the agent: it says why and
// what to do next.
type Host interface {
	// Ask sends the parent a question.
	Ask(ctx context.Context, text string) (Message, error)
	// Escalate sends the parent a matter the Manager cannot settle within
	// the model. Root has no parent: its escalation is a model defect and
	// ends the Run without landing.
	Escalate(ctx context.Context, text string) (Message, error)
	// Answer answers a question or escalation this Manager received.
	Answer(ctx context.Context, to string, text string) (Message, error)
	// Assign sends a working child a task.
	Assign(ctx context.Context, child string, text string) (Message, error)
	// Note records a side observation with this agent as its origin.
	Note(ctx context.Context, text string) (Note, error)
	// Notes reads a page of the Run's Notes after the cursor; an empty
	// cursor starts at the oldest.
	Notes(ctx context.Context, after string) (NotePage, error)
}

// Message kinds.
const (
	Assignment = "assignment"
	Question   = "question"
	Escalation = "escalation"
	Answer     = "answer"
	// Review is a checker's rejection, delivered to its maker.
	Review = "review"
)

// Message is one message along the tree.
type Message struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// From and To are Manager IDs; From is empty for the host.
	From string `json:"from,omitempty"`
	To   string `json:"to"`
	// ReplyTo is the question or escalation an answer answers.
	ReplyTo string `json:"replyTo,omitempty"`
	Text    string `json:"text"`
}

// Note is one Shared Note: a side observation with its origin and no
// authority (concept 1.0 §19).
type Note struct {
	ID      string    `json:"id"`
	Manager string    `json:"manager"`
	Role    Role      `json:"role"`
	Text    string    `json:"text"`
	Time    time.Time `json:"time"`
}

// NotePage is one page of Notes, oldest first. Next is the cursor of the
// following page; it is empty on the last one.
type NotePage struct {
	Notes []Note
	Next  string
}

// Reasons an agent ended.
type Reason string

const (
	// Finished: the agent ended its work by itself.
	Finished Reason = "finished"
	// Failed: the agent or its provider stopped with an error.
	Failed Reason = "failed"
	// DeadlineReached: Start.Deadline passed and the agent was stopped.
	DeadlineReached Reason = "deadline"
	// Cancelled: the context ended and the agent was stopped.
	Cancelled Reason = "cancelled"
)

// FailureClass is the cause of a failed end at the provider.
type FailureClass string

const (
	Capacity    FailureClass = "capacity"
	RateLimit   FailureClass = "rate-limit"
	ServerError FailureClass = "server-error"
	Login       FailureClass = "login"
	Other       FailureClass = "other"
)

// End is how a turn ended. Finished says nothing about the quality of the
// work; the engine reads that from the Run's remote.
type End struct {
	Reason Reason
	// Detail explains Failed, DeadlineReached and Cancelled.
	Detail string
	// Class sorts a Failed end by its cause at the provider, so that a
	// provider failure can be told apart from the agent's own; empty for
	// every other reason.
	Class FailureClass
	// Agent names the provider and model that ran, as observed.
	Agent string
	Usage Usage
	// Continued: the turn continued a living agent's conversation in its
	// container; otherwise it started the agent, in a new container.
	Continued bool
	// Cleanup is empty when everything an agent ended after this turn
	// created is gone, and otherwise says what may remain. An uncertain
	// cleanup is never empty.
	Cleanup string
}

// Usage is what the provider reported for one turn; zero when it reported
// nothing.
type Usage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
	// Partial: the provider reported no total for the turn, as for one the
	// Run ended, that reached its deadline or that broke off, so the counts
	// are a lower bound.
	Partial bool `json:"partial,omitempty"`
}
