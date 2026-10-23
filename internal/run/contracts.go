// Package run executes one complete modeled hierarchy and returns only after
// candidate retention, resource closure and terminal record readback.
package run

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

type Mandate string

const (
	Assessment Mandate = "assessment"
	Implement  Mandate = "implement"
	Verify     Mandate = "verify"
)

type Technical string

const (
	Completed  Technical = "completed"
	Failed     Technical = "failed"
	Incomplete Technical = "incomplete"
)

type Decision string

const (
	Approve      Decision = "approve"
	Reject       Decision = "reject"
	ModelBlocker Decision = "model_blocker"
	Submitted    Decision = "submitted"
)

type Finding struct {
	Manager string `json:"manager"`
	Text    string `json:"text"`
}
type Note struct {
	ID      string `json:"id"`
	Project string `json:"project"`
	Run     string `json:"run"`
	Manager string `json:"manager"`
	Context string `json:"context"`
	Text    string `json:"text"`
}
type Commit struct {
	ID   string `json:"id"`
	Tree string `json:"tree"`
}

// Handoff preserves actual task base/result and accepted child commits. Review
// sees the complete diff; no accepted child paths are subtracted from it.
type ParentReview struct {
	Decision Decision  `json:"decision"`
	Findings []Finding `json:"findings"`
}
type Handoff struct {
	Diff     string        `json:"diff"`
	Review   *ParentReview `json:"review,omitempty"`
	Manager  string        `json:"manager"`
	Base     Commit        `json:"base"`
	Result   Commit        `json:"result"`
	Children []Integration `json:"children"`
	Findings []Finding     `json:"findings"`
	Decision Decision      `json:"decision"`
}
type Integration struct {
	Manager      string `json:"manager"`
	Base         Commit `json:"base"`
	Child        Commit `json:"child"`
	ParentBefore Commit `json:"parentBefore"`
	ParentAfter  Commit `json:"parentAfter"`
}
type Result struct {
	ID        string                `json:"id"`
	Project   string                `json:"project"`
	Binding   string                `json:"binding"`
	Mandate   Mandate               `json:"mandate"`
	Technical Technical             `json:"technical"`
	Decision  Decision              `json:"decision,omitempty"`
	Candidate *project.CandidateRef `json:"candidate,omitempty"`
	Findings  []Finding             `json:"findings"`
	Handoffs  []Handoff             `json:"handoffs"`
	Checks    []CheckObservation    `json:"checks"`
	Cleanup   []Closure             `json:"cleanup"`
	Errors    []string              `json:"errors"`
	Messages  int                   `json:"messages"`
}
type Closure struct {
	Resource string `json:"resource"`
	Closed   bool   `json:"closed"`
	Error    string `json:"error,omitempty"`
}
type CheckObservation struct {
	Commit string `json:"commit"`
	ID     string `json:"id"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

// Allowance is shared by every Run of a command, including later corrections.
// Cleanup uses a separate finite bound and never grants further session work.
type Allowance struct {
	mu              sync.Mutex
	Deadline        time.Time
	remaining, used int
}

func NewAllowance(deadline time.Time, messages int) *Allowance {
	return &Allowance{Deadline: deadline, remaining: messages}
}
func (a *Allowance) take() error {
	if a == nil {
		return errors.New("missing finite allowance")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !time.Now().Before(a.Deadline) || a.remaining <= 0 {
		return errors.New("Run allowance exhausted")
	}
	a.remaining--
	a.used++
	return nil
}
func (a *Allowance) Used() int {
	if a == nil {
		return 0
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.used
}
func (a *Allowance) valid() bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.remaining > 0 && time.Now().Before(a.Deadline)
}

type Request struct {
	ID, Project        string
	Prepared           project.Prepared
	Input              project.CandidateRef
	Mandate            Mandate
	Findings           []Finding
	Allowance          *Allowance
	CleanupTimeout     time.Duration
	PersistenceTimeout time.Duration
}

type Sessions interface {
	Open(context.Context, SessionSpec) (Session, error)
}

// Open must return any partially created handle with its error so Execute can
// close it. Provider-specific support is a separately evidenced adapter fact.
type SessionSpec struct {
	RunID     string
	Manager   organization.Manager
	Mandate   Mandate
	Prepared  project.Prepared
	Workspace Copy
	Findings  []Finding
	Tools     Tools
}
type Session interface {
	Exchange(context.Context, Message) (Response, error)
	Close(context.Context) error
}
type Message struct {
	Task     string
	Feedback []Finding
}
type Response struct {
	Decision Decision
	Findings []Finding
}

// Inbox events are private to one Run. Ask returns immediately; Await delivers
// actual incoming questions, replies, parent tasks or fixed child handoffs.
type Question struct{ ID, From, To, Text string }
type Answer struct{ ID, From, Text string }
type Assignment struct {
	Task     string
	Feedback []Finding
	Base     Commit
}
type Event struct {
	Question   *Question
	Answer     *Answer
	Assignment *Assignment
	Handoff    *Handoff
}
type Tools struct {
	Delegate   func(context.Context, string, string, []Finding) error
	Await      func(context.Context) (Event, error)
	Review     func(context.Context, string, string, Decision, []Finding) error
	Ask        func(context.Context, string, string) (string, error)
	Reply      func(context.Context, string, string) error
	ReadNotes  func(context.Context) ([]Note, error)
	AppendNote func(context.Context, string, string) (Note, error)
}

// WorkspaceService owns exactly one Run's private Git service and copies.
// Retain verifies a self-contained candidate in an empty independent repository.
type Workspaces interface {
	Open(context.Context, Request) (WorkspaceService, error)
}
type WorkspaceService interface {
	Copy(context.Context, string, Commit) (Copy, error)
	Retain(context.Context, Copy, Commit) (project.CandidateRef, error)
	Close(context.Context) error
}
type Copy interface {
	Location() string
	Update(context.Context, Copy, Commit) error
	Snapshot(context.Context) (Commit, error)
	Inspect(context.Context, Commit) (project.PreviewPreparationAttempt, error)
	Integrate(context.Context, Copy, Commit) (Commit, error)
	Review(context.Context, Copy, Commit, Commit) (string, error)
	Close(context.Context) error
}
type Records interface {
	Save(context.Context, Result) error
	Read(context.Context, string, string) (Result, error)
	Notes(context.Context, string) ([]Note, error)
	Append(context.Context, Note) (Note, error)
}
type CandidateChecks interface {
	Check(context.Context, project.Prepared, Copy, Commit) ([]CheckObservation, error)
}

type Executor struct {
	Sessions   Sessions
	Workspaces Workspaces
	Records    Records
	Checks     CandidateChecks
}

var ErrUnavailable = errors.New("native sessions unavailable: adapter capability is not ready")

type UnavailableSessions struct{}

func (UnavailableSessions) Open(context.Context, SessionSpec) (Session, error) {
	return nil, ErrUnavailable
}

// Technical check request validation shared with the concrete check adapter.
const maxRecordBytes = 16 << 20

func identifier(s string) bool {
	return strings.TrimSpace(s) != "" && len(s) <= 4096 && !strings.ContainsRune(s, 0)
}
func text(s string) bool { return !strings.ContainsRune(s, 0) }
func imageIdentity(s string) bool {
	if !strings.HasPrefix(s, "sha256:") || len(s) != 71 {
		return false
	}
	for _, r := range s[7:] {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}
func validID(s string) bool {
	if s == "" || len(s) > 100 {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.", r) {
			return false
		}
	}
	return s != "." && s != ".."
}
func validateRequest(r Request) error {
	if !validID(r.ID) || !validID(r.Project) || !r.Prepared.Valid() || r.Input.Project != r.Project || r.Input.Binding != r.Prepared.Binding() || r.Input.Commit == "" || r.Input.Tree == "" || !r.Allowance.valid() || r.CleanupTimeout <= 0 || r.CleanupTimeout > time.Minute || r.PersistenceTimeout <= 0 || r.PersistenceTimeout > time.Minute {
		return fmt.Errorf("invalid Run request, checked binding or finite limits")
	}
	if r.Mandate != Assessment && r.Mandate != Implement && r.Mandate != Verify {
		return fmt.Errorf("invalid mandate")
	}
	return nil
}
