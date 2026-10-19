package run

import (
	"context"
	"errors"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ErrInvalid reports a Config the engine cannot run.
var ErrInvalid = errors.New("invalid reconciliation configuration")

// Manager is one permanent responsibility of the run's team. Path is the
// organization Package path; Root is "." and has no Parent. Children are the
// direct child paths. Role is the short description a parent or child sees for
// this contact. Brief is the Manager's standing context, rendered by apply:
// its applicable model obligations with provenance, its owned artifact areas,
// its registry shard and the required checks. The engine adds the mandate,
// contacts, owner decisions, findings, messages and check results itself.
// Write is the Manager's write plan for Execute turns. Every Manager takes part
// in every Apply in the same order: Root starts its first Execute turn at once,
// every other Manager once its parent's first Execute turn has ended (or the
// parent is disabled), so a parent supplies the foundation its children need
// first.
type Manager struct {
	Path           string
	Parent         string
	Children       []string
	Role           string
	Brief          string
	ExecuteProfile string
	VerifyProfile  string
	Write          WritePlan
}

// WritePlan is where an Execute turn of one Manager may write in the shared
// working tree. Without Enforced the whole tree is writable and ownership is
// trusted (no declared layout). With Enforced, Base says whether the whole tree
// is writable before the Mounts, which then overlay it in order, parents
// before their descendants. Every mount path is an existing directory of the
// shared tree; the engine creates them before the first Execute turn.
type WritePlan struct {
	Enforced bool
	Base     bool
	Mounts   []PathMount
}

// PathMount overlays one directory of the shared tree, relative to /work.
type PathMount struct {
	Path     string `json:"path"`
	Writable bool   `json:"writable"`
}

// Check is one required check of the project. Request is complete except for
// its ID and Tree, which the engine sets for each candidate. A failed check is
// a blocking finding addressed to Owner.
type Check struct {
	Name       string
	RequiredBy string
	Owner      string
	Request    CheckRequest
}

// Budget bounds one run. Deadline is absolute: at that time the engine stops
// every running turn and check and finishes Incomplete. AgentTurns counts
// Execute and Verify turns; Rounds counts Verify rounds, each preceded by one
// Execute phase.
type Budget struct {
	Deadline    time.Time
	TurnTimeout time.Duration
	AgentTurns  int
	Rounds      int
}

// Finding is one concrete deviation. RaisedBy is the Manager whose Verify
// raised it, or "markitect" for a failed check or a structural accounting
// finding. Owner is the Manager that must act. Only a later fresh Verify of the
// raising Manager closes a Verify finding, with a reason; a structural finding
// holds for exactly the candidate it was observed on.
type Finding struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	RaisedBy     string `json:"raisedBy"`
	Obligation   string `json:"obligation"`
	Observed     string `json:"observed"`
	Blocking     bool   `json:"blocking"`
	Round        int    `json:"round"`
	Closed       bool   `json:"closed"`
	ClosedRound  int    `json:"closedRound,omitempty"`
	ClosedReason string `json:"closedReason,omitempty"`
}

// MarkitectFinding is the RaisedBy value of findings the engine observes itself.
const MarkitectFinding = "markitect"

// Question is a question Root asked the human.
type Question struct {
	ID       string    `json:"id"`
	Text     string    `json:"text"`
	Asked    time.Time `json:"asked"`
	Answer   string    `json:"answer,omitempty"`
	Answered bool      `json:"answered"`
}

// Owner is the human's channel. Ask publishes Root's question; Answer reports
// the human's answer once one exists. Work that does not depend on the answer
// continues meanwhile.
type Owner interface {
	Ask(context.Context, Question) error
	Answer(context.Context, string) (string, bool, error)
}

// Event is one entry of the run's append-only evidence.
type Event struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Manager string    `json:"manager,omitempty"`
	Turn    string    `json:"turn,omitempty"`
	Round   int       `json:"round,omitempty"`
	Data    any       `json:"data,omitempty"`
}

// Evidence retains the run's evidence. A Record error ends the run Incomplete,
// because the run could no longer show what happened. Transcript returns the
// Host path for a turn's provider stream, or empty for none.
type Evidence interface {
	Record(Event) error
	Transcript(turn string) string
}

// Config fixes one reconciliation run. Apply prepares every value from one
// complete capture; the engine discovers nothing. Team lists Root first and
// every parent before its children. Start is the captured starting stand with
// its bytes. Account returns the blocking structural findings of a candidate,
// each with its Owner: unexplained or duplicate files, invalid registry
// shards, registrations outside the registering Manager's area and changed
// protected inputs. OwnerOf returns the Manager that owns a path under the
// project's persistent layout. Owner may be nil when no human can answer.
type Config struct {
	RunID          string
	Team           []Manager
	Start          snapshot.Tree
	OwnerRequest   string
	OwnerDecisions []string
	Checks         []Check
	Account        func(snapshot.Tree) []Finding
	OwnerOf        func(snapshot.Path) string
	Budget         Budget
	MaxEntries     int
	MaxBytes       int64
	Agents         Agents
	Workspaces     Workspaces
	Runner         Checks
	Owner          Owner
	Evidence       Evidence
	Clock          func() time.Time
}

// Status is the result of a run.
type Status string

const (
	// Verified: every Manager assessed the same immutable candidate in a fresh
	// Verify, no blocking finding is open and every required check passed on it.
	Verified Status = "verified"
	// Incomplete: the budget ended, an owner question stayed open, or
	// quiescence, capture or evidence was uncertain. Nothing may be published.
	Incomplete Status = "incomplete"
	// Failed: a definite error of the run's inputs or ports.
	Failed Status = "failed"
)

// Outcome is the run's result. Candidate is the verified stand and is set only
// when Status is Verified; it equals Start when Execute changed nothing. The
// engine closes every candidate volume itself. Once the shared work volume
// "<RunID>-work" holds the start stand, the engine never closes it: after
// Reconcile returns, in every outcome, the caller owns it and decides to close it
// or keep it as retained work. A work volume whose materialization failed is
// closed by the engine.
type Outcome struct {
	Status    Status
	Reason    string
	Candidate snapshot.Tree
	Rounds    []Round
	Findings  []Finding
	Questions []Question
	Turns     []TurnRecord
	Usage     Usage
	Homes     []string
}

// Round is one Verify round over one immutable candidate. The Execute phase
// before it carries the same number.
type Round struct {
	Number      int           `json:"number"`
	Tree        string        `json:"tree"`
	Checks      []CheckResult `json:"checks"`
	Assessments []Assessment  `json:"assessments"`
	Green       bool          `json:"green"`
}

// CheckResult is one required check on a round's candidate.
type CheckResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// Assessment is one Manager's fresh Verify result on a round's candidate.
type Assessment struct {
	Manager   string `json:"manager"`
	Turn      string `json:"turn"`
	Satisfied bool   `json:"satisfied"`
	Summary   string `json:"summary"`
}

// TurnRecord is one Agent turn as the engine observed it. Status is the
// Execute turn's reported state (done or waiting), "assessed" for a Verify
// turn that reported, or empty when the turn reported nothing.
type TurnRecord struct {
	ID      string    `json:"id"`
	Manager string    `json:"manager"`
	Kind    TurnKind  `json:"kind"`
	Round   int       `json:"round"`
	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended"`
	Usage   Usage     `json:"usage"`
	Stopped bool      `json:"stopped"`
	Status  string    `json:"status,omitempty"`
	Reason  string    `json:"reason,omitempty"`
}

// Reconcile runs the cycle of the current Apply concept (DEC-058): Execute in
// one shared working tree starting at Root, where every other Manager starts
// when its parent's first turn has ended and results are reported upward;
// observed quiescence and one immutable candidate; fresh recursive Verify with
// the required checks on that candidate; and repair through the owners with a
// new candidate and a new whole Verify round until a round is green or the
// budget ends. There is no Verify before the first Execute. The shared work
// volume "<RunID>-work" belongs to the caller once Reconcile returns, in every
// outcome, including Verified.
func Reconcile(ctx context.Context, cfg Config) (Outcome, error) {
	return reconcile(ctx, cfg)
}
