// Package agentport is how a Command starts sessions (concept 3.0 §6, §11): one
// session with one prompt, as an implementer or a reviewer, in the Command's
// one container, and how it ended. Provider adapters implement Agents; the
// sandbox implements Shell, the only way a session reaches the container.
//
// A session's work reaches the Command only through what it leaves in its
// folder and in the notes the Command reads after it has ended. Nothing else it
// says counts.
package agentport

import (
	"context"
	"fmt"
	"io"
	"time"
)

// Mode is what a Command does.
type Mode string

const (
	// Apply: an implementer changes the artifacts, a reviewer compares them
	// with the spec, and the work lands.
	Apply Mode = "apply"
	// Verify: a reviewer only, and nothing lands.
	Verify Mode = "verify"
)

// Role is a session's part in a Command.
type Role string

const (
	Implementer Role = "implementer"
	Reviewer    Role = "reviewer"
)

// ImplementerUser is the Linux user of the implementer's session.
const ImplementerUser = "implementer"

// ReviewerUsers bounds the reviewers that work side by side, each as a Linux user
// of its own, so that nothing one does reaches another.
const ReviewerUsers = 8

// ReviewerUser is the Linux user of the nth reviewer, from 1.
func ReviewerUser(n int) string { return fmt.Sprintf("reviewer-%d", n) }

// Exec is one command in the Command's container.
type Exec struct {
	// User is the container's user the command runs as.
	User string
	// Dir is the absolute working directory in the container.
	Dir string
	// Argv is the command, run without a shell; Argv[0] is an absolute path.
	Argv []string
	// Env holds NAME=value additions.
	Env []string
	// Secrets holds NAME=value additions whose value never appears on a command
	// line, where a process listing could show it.
	Secrets        []string
	Stdin          io.Reader
	Stdout, Stderr io.Writer
}

// ExitError is a command that ran and ended with a nonzero status.
type ExitError struct{ Code int }

func (e *ExitError) Error() string { return fmt.Sprintf("exit status %d", e.Code) }

// Shell runs commands in the Command's one container. Exec returns once the
// command has ended; when ctx ends, it first ends every process of that user.
type Shell interface {
	Exec(ctx context.Context, e Exec) error
}

// Agents starts sessions. Run starts one session in the shell and returns once
// it has ended: it finished, failed, reached Deadline or ctx ended.
type Agents interface {
	Run(ctx context.Context, sh Shell, s Session) End
}

// Session is one session with one prompt.
type Session struct {
	Run  string
	Role Role
	// User is the Linux user the session runs as.
	User string
	// Round counts this role's sessions in this Command, from 1.
	Round int
	// Dir is the folder the session starts in.
	Dir string
	// Prompt is the whole of what the session is told.
	Prompt string
	// Conversation names an earlier session of this role to continue; empty
	// starts a fresh one.
	Conversation string
	Deadline     time.Time
}

// Reasons a session ended.
type Reason string

const (
	// Finished: the session ended by itself. It says nothing about its work.
	Finished Reason = "finished"
	// Failed: the provider or the launch failed.
	Failed Reason = "failed"
	// DeadlineReached: Deadline passed and the session was stopped.
	DeadlineReached Reason = "deadline"
	// Cancelled: the context ended and the session was stopped.
	Cancelled Reason = "cancelled"
)

// FailureClass is the cause of a failed session at the provider.
type FailureClass string

const (
	Capacity    FailureClass = "capacity"
	RateLimit   FailureClass = "rate-limit"
	ServerError FailureClass = "server-error"
	Login       FailureClass = "login"
	Other       FailureClass = "other"
)

// Usage counts one session's tokens, or a Command's.
type Usage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
	// Partial: a session reported no total, so the counts are a lower bound.
	Partial bool `json:"partial,omitempty"`
}

// Add sums two counts.
func (u Usage) Add(v Usage) Usage {
	return Usage{u.InputTokens + v.InputTokens, u.OutputTokens + v.OutputTokens, u.Partial || v.Partial}
}

// Seen is what a session's transcript shows it did, subagents included. A
// transcript shows some of it and never why, so it can only limit what a
// reviewer claims (concept 3.0 §8, §10).
type Seen struct {
	// Measured: the provider's transcript shows reads at all.
	Measured bool `json:"measured"`
	// Opened holds the files the transcript shows being read, as named there.
	Opened []string `json:"opened,omitempty"`
	// Commands holds the shell commands the transcript shows being run.
	Commands []string `json:"commands,omitempty"`
	// Subagents counts the subagents the session started.
	Subagents int `json:"subagents,omitempty"`
}

// End is how a session ended.
type End struct {
	Reason Reason
	// Class is the provider's failure class of a failed session.
	Class FailureClass
	// Detail explains anything but Finished.
	Detail string
	// Conversation names the session for a later one that continues it.
	Conversation string
	// Agent is provider:model as the provider reported it; Models lists every
	// model that answered, subagents included.
	Agent  string
	Models []string
	Usage  Usage
	Seen   Seen
	// Text is the session's closing words.
	Text string
}
