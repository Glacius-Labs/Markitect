// Package run executes one complete modeled hierarchy and returns only after
// candidate retention, resource closure and terminal record readback.
package run

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
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

// Shared Note limits. A Note stays far below one tool reply even when every
// byte needs escaping, and a page of Notes always fits the tool transport.
const (
	MaxNoteText    = 16 << 10
	MaxNoteContext = 1 << 10
	// NotePageBytes bounds the encoded size of the Notes in one page and
	// NotePageCount their number; a page holds at least one Note.
	NotePageBytes = 128 << 10
	NotePageCount = 100
)

// NotePage is one page of Notes, oldest first. Next is the cursor of the
// following page and empty after the last one.
type NotePage struct {
	Notes []Note `json:"notes"`
	Next  string `json:"next"`
}
type Commit struct {
	ID   string `json:"id"`
	Tree string `json:"tree"`
}

// Handoff preserves actual task base/result and integrated child commits. Review
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
	// Check is the deterministic check of Result. Every Implement submission
	// carries it; nil means no check ran.
	Check *Check `json:"check,omitempty"`
}

// Check is the deterministic check of one fixed state: structural validity,
// accounting and the declared checks placed in that Manager's copy. Checks
// lists the declared checks that ran; a check not listed did not run. Stale
// lists the registry links in the Manager's responsibility to Definitions the
// model no longer has: they make an Implement or Verify check invalid, while
// Assessment only reports them as realization still to do.
type Check struct {
	Commit     Commit   `json:"commit"`
	Valid      bool     `json:"valid"`
	Diagnostic string   `json:"diagnostic,omitempty"`
	Checks     []string `json:"checks"`
	Stale      []string `json:"stale,omitempty"`
}
type Integration struct {
	Manager      string `json:"manager"`
	Base         Commit `json:"base"`
	Child        Commit `json:"child"`
	ParentBefore Commit `json:"parentBefore"`
	ParentAfter  Commit `json:"parentAfter"`
}

// SessionObservation contains concrete native construction facts. Controlled
// sessions omit this optional observation rather than fabricate native identity.
type SessionObservation struct{ ProfileIdentity, ImageID, HarnessDigest string }
type ManagerSessionObservation struct {
	Manager     string             `json:"manager"`
	Observation SessionObservation `json:"observation"`
}
type ObservedSession interface{ Observation() SessionObservation }

// ActingSession is a Session whose adapter observes native executions outside
// Run's tools, such as shell commands, file changes and web searches.
// Acted reports whether its last Exchange executed one. Controlled sessions
// act only through Run's tools and their copy, which Run observes itself.
type ActingSession interface{ Acted() bool }

type Result struct {
	ID        string                      `json:"id"`
	Project   string                      `json:"project"`
	Binding   string                      `json:"binding"`
	Mandate   Mandate                     `json:"mandate"`
	Technical Technical                   `json:"technical"`
	Decision  Decision                    `json:"decision,omitempty"`
	Sessions  []ManagerSessionObservation `json:"sessions"`
	Input     project.CandidateRef        `json:"input"`
	Candidate *project.CandidateRef       `json:"candidate,omitempty"`
	Findings  []Finding                   `json:"findings"`
	Handoffs  []Handoff                   `json:"handoffs"`
	Checks    []CheckObservation          `json:"checks"`
	Cleanup   []Closure                   `json:"cleanup"`
	Errors    []string                    `json:"errors"`
	Messages  int                         `json:"messages"`
}
type Closure struct {
	Resource string `json:"resource"`
	Closed   bool   `json:"closed"`
	Error    string `json:"error,omitempty"`
}
type CheckObservation struct {
	Commit  string       `json:"commit"`
	ID      string       `json:"id"`
	Receipt CheckReceipt `json:"receipt"`
	Error   string       `json:"error,omitempty"`
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
		return ErrBudget
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

// Ready observes capacity without granting messages or extending the deadline.
func (a *Allowance) Ready(managers int) bool {
	return managers > 0 && a.valid() && a.minimum(managers)
}

type Request struct {
	MaxManagers        int
	ID, Project        string
	Prepared           project.Prepared
	Input              project.CandidateRef
	Mandate            Mandate
	Findings           []Finding
	Allowance          *Allowance
	CleanupTimeout     time.Duration
	PersistenceTimeout time.Duration
	// Zero retains the standalone caller's whole-Run deadline. Apply supplies
	// its explicit finite per-message limit.
	MessageTimeout time.Duration
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

// Exchange is one native response to a Message. It returns only after its
// active writes and tool handlers have drained. Its end is not a judgment: the
// Manager judges through the Judge tool. Close must interrupt an active
// Exchange and report actual process/home closure.
type Session interface {
	Exchange(context.Context, Message) error
	Close(context.Context) error
}

// Message carries the events delivered to an idle Manager and the host's
// diagnostics about its active assignment.
type Message struct {
	Events   []Event   `json:"events,omitempty"`
	Feedback []Finding `json:"feedback,omitempty"`
}

// Events are private to one Run. Ask returns immediately; Await delivers
// actual parent assignments, incoming questions, answers or fixed child handoffs.
type Question struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}
type Answer struct {
	ID   string `json:"id"`
	From string `json:"from"`
	Text string `json:"text"`
}
type Assignment struct {
	Task     string    `json:"task"`
	Findings []Finding `json:"findings"`
	Base     Commit    `json:"base"`
}
type Event struct {
	Assignment *Assignment `json:"assignment,omitempty"`
	Question   *Question   `json:"question,omitempty"`
	Answer     *Answer     `json:"answer,omitempty"`
	Handoff    *Handoff    `json:"handoff,omitempty"`
}

// Tools is the fixed Manager tool set. Each one acts for the Manager that owns
// the Session; Run sets the origin of every Finding.
type Tools struct {
	Delegate   func(context.Context, string, string, []Finding) error
	Await      func(context.Context) (Event, error)
	Review     func(context.Context, string, string, Decision, []Finding) error
	Ask        func(context.Context, string, string) (string, error)
	Reply      func(context.Context, string, string) error
	Judge      func(context.Context, Decision, []Finding) error
	Check      func(context.Context) (Check, error)
	ReadNotes  func(context.Context, string) (NotePage, error)
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

// CopyLocation names one actual storage location. Directory is for local
// controlled execution; native copies use an existing private Linux volume
// subdirectory. It never names a synchronized second writable working copy.
type CopyLocation struct{ Directory, Volume, Subpath string }

func (l CopyLocation) Valid() bool {
	if l.Directory != "" {
		return l.Volume == "" && l.Subpath == "" && filepath.IsAbs(l.Directory) && filepath.Clean(l.Directory) == l.Directory
	}
	if !validID(l.Volume) || l.Subpath == "" || path.Clean(l.Subpath) != l.Subpath || strings.HasPrefix(l.Subpath, "/") || strings.Contains(l.Subpath, "\\") {
		return false
	}
	for _, part := range strings.Split(l.Subpath, "/") {
		if !validID(part) {
			return false
		}
	}
	return true
}

// DisposableCopy owns one verified non-Git tree for one check. Its closure
// belongs to that check; the enclosing Run service also owns failure cleanup.
type DisposableCopy interface {
	Location() CopyLocation
	Tree() snapshot.TreeID
	Directories() []snapshot.Path
	Close(context.Context) error
}

type Copy interface {
	Materialize(context.Context, snapshot.Tree, []snapshot.Path) (DisposableCopy, error)
	Location() CopyLocation
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
	Notes(context.Context, string, string) (NotePage, error)
	Append(context.Context, Note) (Note, error)
}
type CandidateChecks interface {
	Check(context.Context, []project.ExecutionCheck, Copy, Commit) ([]CheckObservation, error)
}

type Executor struct {
	Sessions   Sessions
	Workspaces Workspaces
	Records    Records
	Checks     CandidateChecks
}

// ErrMergeConflict identifies a verified unresolved Git index or preserved
// child-side parent merge for ordinary Manager repair. Other Git errors are technical.
var ErrMergeConflict = errors.New("parent baseline has unresolved Git conflicts")

var ErrObservationIncomplete = errors.New("Run physical observation incomplete")

var ErrBudget = errors.New("Run allowance exhausted")

// ErrNoProgress ends a Run whose Manager repeatedly answers the continuation of
// its open assignment without doing anything: no Manager tool call, no native
// execution and no working-state change. The result is incomplete.
var ErrNoProgress = errors.New("Manager took no action on its open assignment")
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
	if r.MessageTimeout < 0 || r.MessageTimeout > time.Hour {
		return fmt.Errorf("invalid per-message timeout")
	}
	if !validID(r.ID) || !validID(r.Project) || !r.Prepared.Valid() || r.Input.Project != r.Project || r.Input.Binding != r.Prepared.Binding() || r.Input.Commit == "" || r.Input.Tree == "" || !r.Allowance.valid() || r.MaxManagers <= 0 || len(r.Prepared.Topology().Managers()) > r.MaxManagers || !r.Allowance.minimum(len(r.Prepared.Topology().Managers())) || r.CleanupTimeout <= 0 || r.CleanupTimeout > time.Minute || r.PersistenceTimeout <= 0 || r.PersistenceTimeout > time.Minute {
		return fmt.Errorf("invalid Run request, checked binding or finite limits")
	}
	if r.Mandate != Assessment && r.Mandate != Implement && r.Mandate != Verify {
		return fmt.Errorf("invalid mandate")
	}
	return nil
}

func (a *Allowance) minimum(n int) bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.remaining >= n
}
