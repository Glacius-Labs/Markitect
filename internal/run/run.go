package run

import (
	"context"
	"errors"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ErrInvalid reports a Config or Pass the Orchestrator cannot run.
var ErrInvalid = errors.New("invalid run configuration")

// Purpose names what one Pass is for. Apply first assesses the bound model,
// then runs the work purposes of its strategy, Implement in direct and Plan,
// Prepare and Implement in staged, and then Verify; Extract supplies its own
// work purposes. Every purpose uses the same recursive base process.
type Purpose string

const (
	Assess    Purpose = "assess"
	Plan      Purpose = "plan"
	Prepare   Purpose = "prepare"
	Implement Purpose = "implement"
	Verify    Purpose = "verify"
)

// Manager is one required responsibility of the run's organization. Path is
// the organization Package path; Root is "." and has no Parent. Children are
// the direct child paths. Role is the short description its contacts see.
// Brief is the Manager's standing context: its own obligations with
// provenance, labeled foreign context with owners, its owned areas, local
// registry and the required checks. Obligations are the identities of its own
// Definitions: every basis and result covers each of them. Write is where its
// work turns may write in a Pass whose effects allow writes; it is always
// enforced. Helpers are the local roles it may delegate bounded tasks to,
// at most MaxHelpers at a time.
type Manager struct {
	Path           string
	Parent         string
	Children       []string
	Role           string
	Brief          string
	Obligations    []string
	ExecuteProfile string
	VerifyProfile  string
	Write          WritePlan
	Helpers        []Helper
	MaxHelpers     int
}

// Helper is one configured role a Manager may instantiate for a delegated
// task: a Worker, an Architect or another specialist. A helper runs in its own
// container with its own provider home, reads the whole stand and writes only
// the explicit subset of its Manager's area that the delegation names, and only
// when Writes allows it. It reports to its Manager only.
type Helper struct {
	Role         string
	Profile      string
	Instructions string
	Writes       bool
}

// WritePlan is where a Manager's turns may write in the shared working tree:
// Base says whether the whole tree is writable before the Mounts, which overlay
// it in order, parents before their descendants. A mount is a directory of the
// shared tree, or with File an existing file of the start stand. The
// Orchestrator creates the mount directories before the first turn.
type WritePlan struct {
	Base   bool
	Mounts []PathMount
}

// PathMount overlays one directory or file of the shared tree, relative to
// /work.
type PathMount struct {
	Path     string `json:"path"`
	Writable bool   `json:"writable"`
	File     bool   `json:"file,omitempty"`
}

// Check is one required check of the project. Request is complete except for
// its ID and Tree, which the Orchestrator sets for each candidate. A failed
// check is a genuine finding addressed to Owner. ProfileArgs is the number of
// leading Request.Args that belong to the check's execution profile: a probe
// run with this check's profile keeps them and replaces the project arguments
// with its own.
type Check struct {
	Name        string
	RequiredBy  string
	Owner       string
	Request     CheckRequest
	ProfileArgs int
}

// Budget bounds one run. Deadline is absolute: at that time the Orchestrator
// stops every running turn and check and finishes Incomplete. AgentTurns counts
// every Agent turn of Managers, reviewers and helpers; Passes counts the
// Passes of the run.
type Budget struct {
	Deadline    time.Time
	TurnTimeout time.Duration
	AgentTurns  int
	Passes      int
}

// Verdict is the substantive judgment of a complete result, a Manager's or a
// Pass's: Satisfied means its stated objective is met, NeedsCorrection that
// the completed work identified required corrective work.
type Verdict string

const (
	Satisfied       Verdict = "satisfied"
	NeedsCorrection Verdict = "needs_correction"
)

// Finding is one concrete deviation. RaisedBy is the Manager whose submission
// raised it, or "markitect" for a failed required check. Owner is the Manager
// that must act: the raiser or one of its descendants. Blocking marks a genuine
// finding (an observed model violation, a missing required realization or
// evidence item, a failed required check); a non-blocking one is a labeled
// note. Correction is the earliest work purpose its correction needs. A
// finding stays open until a later Pass of the same kind closes it through a
// submission of its raiser, with a reason; a reviewer that omits it does not
// close it. A markitect finding holds for exactly the candidate it was
// observed on.
type Finding struct {
	ID         string  `json:"id"`
	Owner      string  `json:"owner"`
	RaisedBy   string  `json:"raisedBy"`
	Obligation string  `json:"obligation"`
	Observed   string  `json:"observed"`
	Blocking   bool    `json:"blocking"`
	Correction Purpose `json:"correction,omitempty"`
	Pass       string  `json:"pass"`
	Review     bool    `json:"review"`
	// Candidate is the tree of the review Pass's candidate a review finding was
	// raised on.
	Candidate string `json:"candidate,omitempty"`
	// Disputes are explicit independent objections of ancestor reviewers; a
	// disputed finding stays open and blocking until its disposition.
	Disputes     []Dispute `json:"disputes,omitempty"`
	Closed       bool      `json:"closed"`
	ClosedPass   string    `json:"closedPass,omitempty"`
	ClosedReason string    `json:"closedReason,omitempty"`
	// ClosedAs is "resolved" for a defect corrected on a later candidate and
	// "disproved" for a finding a fresh independent assessment showed to be
	// erroneous; ClosedEvidence carries a disproof's evidence.
	ClosedAs       string `json:"closedAs,omitempty"`
	ClosedEvidence string `json:"closedEvidence,omitempty"`
	// Subject and Definition are set only on findings of a review Pass with
	// ArtifactFindingsAdvisory: Subject is "artifact" or "model", and Definition
	// names the Definition an artifact finding deviates from.
	Subject    string `json:"subject,omitempty"`
	Definition string `json:"definition,omitempty"`
	// Receipts are the probe receipts the finding cites as its evidence.
	Receipts []string `json:"receipts,omitempty"`
}

// Dispute is one explicit independent objection to a finding: the reviewer that
// raised it, the Pass and candidate it was raised on, its reason and evidence.
type Dispute struct {
	By        string `json:"by"`
	Pass      string `json:"pass"`
	Candidate string `json:"candidate"`
	Reason    string `json:"reason"`
	Evidence  string `json:"evidence"`
}

// Closure kinds of a finding.
const (
	ClosedResolved  = "resolved"
	ClosedDisproved = "disproved"
)

// MarkitectFinding is the RaisedBy value of findings the Orchestrator observes
// itself.
const MarkitectFinding = "markitect"

// Question is one question along the Manager tree. From and To are adjacent:
// parent and child. About names the question this one serves when it forwards
// an exchange along the tree. Nobody outside the tree answers during a run.
type Question struct {
	ID         string    `json:"id"`
	From       string    `json:"from"`
	To         string    `json:"to"`
	About      string    `json:"about,omitempty"`
	Pass       string    `json:"pass"`
	Text       string    `json:"text"`
	Asked      time.Time `json:"asked"`
	Answer     string    `json:"answer,omitempty"`
	Answered   bool      `json:"answered"`
	Lapsed     bool      `json:"lapsed,omitempty"`
	AnsweredAt time.Time `json:"answeredAt,omitzero"`
}

// Problem is a structured report of a Manager: its category, its text and
// whether it was escalated to the parent. Abort marks Root's decision that the
// model is erroneous or contradictory, which ends the run Failed.
type Problem struct {
	ID        string `json:"id"`
	Manager   string `json:"manager"`
	Pass      string `json:"pass"`
	Category  string `json:"category"`
	Text      string `json:"text"`
	Escalated bool   `json:"escalated,omitempty"`
	Question  string `json:"question,omitempty"`
	Abort     bool   `json:"abort,omitempty"`
}

// Event is one entry of the run's append-only evidence.
type Event struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Pass    string    `json:"pass,omitempty"`
	Manager string    `json:"manager,omitempty"`
	Turn    string    `json:"turn,omitempty"`
	Data    any       `json:"data,omitempty"`
}

// Evidence retains the run's evidence. A Record error ends the run Incomplete,
// because the run could no longer show what happened. Transcript returns the
// path for a turn's provider stream, or empty for none.
type Evidence interface {
	Record(Event) error
	Transcript(turn string) string
}

// RegistryView is the derived, read-only project-wide view of the local
// registries of one stand: the exact versions of the shards it includes and
// the effective registration of every registered artifact. Only the
// Orchestrator assembles it, from checked versions; it is no model input and no
// authority.
type RegistryView struct {
	Shards        map[string]string `json:"shards"`
	Registrations []Registration    `json:"registrations"`
}

// Registration is the effective registration of one artifact: its owner, form,
// the Definitions it realizes and the shard entry (an exact path or a
// directory ending in "/") that registers it.
type Registration struct {
	Path     string   `json:"path"`
	Owner    string   `json:"owner"`
	Form     string   `json:"form"`
	Realizes []string `json:"realizes"`
	Entry    string   `json:"entry"`
}

// ModelDefinition is the text of one Definition of the bound model with its
// owning Package, for look_up.
type ModelDefinition struct {
	ID    string
	Owner string
	Text  string
}

// Config fixes one run. Apply and Extract prepare every value from one
// complete capture; the Orchestrator discovers nothing. Team lists Root first
// and every parent before its children. Start is the captured starting stand.
//
// ModelCheck and FileCheck are the deterministic gates. FileCheck returns the
// blocking findings of a stand's file accounting, each with its Owner:
// unexplained or duplicate files, invalid local registries, registrations
// outside their owner's area, missing owners and changed protected inputs.
// ModelCheck returns the findings of the formal check of the stand's model.
// Both pass at every Pass boundary and on every candidate; FileCheck also
// gates every submitted basis and result on the composition of the accepted
// stand with the submitter's own stable area. Both are pure and must be safe
// for concurrent use. OwnerOf returns the Manager that owns a path; Registry
// assembles the derived registry view of a stand; Definition looks up a
// Definition of the bound model.
//
// OwnerRequest is the invoking AI's request; nobody answers questions during
// the run. ResolveDefinition confirms a Definition identity on a candidate for review
// Passes with ArtifactFindingsAdvisory; it must be safe for concurrent use.
// Binding is what the caller bound the run to besides these inputs, such as an
// Apply's strategy, effective settings and implementation identity; run.start
// records it as given and the Orchestrator chooses nothing by it.
type Config struct {
	RunID        string
	Binding      any
	Team         []Manager
	Start        snapshot.Tree
	OwnerRequest string
	// RootGuidance is Root's own standing duty in the work Passes of this kind
	// of run, such as ending an Apply for a defective model.
	RootGuidance string
	Checks       []Check
	ModelCheck   func(snapshot.Tree) []Finding
	FileCheck    func(snapshot.Tree) []Finding
	OwnerOf      func(snapshot.Path) string
	Registry     func(snapshot.Tree) RegistryView
	Definition   func(id string) (ModelDefinition, bool)
	Budget       Budget
	MaxEntries   int
	MaxBytes     int64
	Agents       Agents
	Workspaces   Workspaces
	Runner       Checks
	Evidence     Evidence
	Clock        func() time.Time

	ResolveDefinition func(snapshot.Tree, string) bool
}

// PassSpec declares one Pass. Mandate states the purpose's task for every
// participant. Writes allows each Manager to change its owned area and local
// registry; without it the project is read-only. Review makes the Pass an
// independent assessment of one immutable candidate: the current stand is
// materialized read-only, the gates and required checks run on it, and every
// Manager's role is a fresh reviewer without the author's conversation.
// Guidance (review) states what a genuine finding is. Plans requires the plan
// payload and keeps the accepted plans: their dependencies for a later Pass's
// purpose become start conditions of that Pass. Corrections lists the work
// purposes a finding may name, earliest first; the last is the default.
// ArtifactFindingsAdvisory (review) makes every finding state its subject:
// an artifact finding names a Definition and never blocks. BindWaits (work)
// declares dependencies through waits, for a workflow without a planning Pass:
// the accepted basis or result of a Manager outside the waiter's own line that
// a wait names, or that is already accepted when the wait is reported, is bound
// to the waiter's work; its revision or reopening resets a waiter whose basis
// built on it, for which it is a start condition of the restart, and returns a
// waiter whose result alone built on it to its integration. Assesses (work)
// makes the Pass the read-only assessment of the bound model before any
// realization: a sealed verdict other than satisfied ends the run Failed, since
// no Pass of the run can correct the model.
type PassSpec struct {
	Purpose                  Purpose
	Mandate                  string
	Writes                   bool
	Review                   bool
	Guidance                 string
	Plans                    bool
	Corrections              []Purpose
	ArtifactFindingsAdvisory bool
	BindWaits                bool
	Assesses                 bool
}

// Status is the result of a run.
type Status string

const (
	// Verified: the last Pass was a clean independent review of one immutable
	// candidate: every Manager's assessment is accepted and satisfied, no genuine
	// finding is open and every required check passed on that candidate.
	Verified Status = "verified"
	// Incomplete: required work is unfinished: the budget or deadline ended, a
	// wait could not be resolved, or a turn, capture or the evidence was
	// uncertain. Nothing may be published.
	Incomplete Status = "incomplete"
	// Failed: a definite rejection: a failed Model check or File check at a
	// boundary, Root's decision that the model is erroneous or contradictory, or
	// a definite error of the run's inputs or ports.
	Failed Status = "failed"
)

// Outcome is the run's result. Candidate is the verified stand and is set only
// when Status is Verified. The Orchestrator closes candidate volumes itself.
// Once the shared work volume "<RunID>-work" holds the start stand, it never
// closes it: after Finish, the caller owns it and closes it or keeps it as
// retained work. Probes are every probe receipt of the run, adverse ones
// included, whatever the reports cited.
type Outcome struct {
	Status    Status
	Reason    string
	Candidate snapshot.Tree
	Passes    []PassRecord
	Findings  []Finding
	Questions []Question
	Problems  []Problem
	Turns     []TurnRecord
	Probes    []ProbeReceipt
	Usage     Usage
	Homes     []string
}

// PassRecord is one sealed or ended Pass. Tree is the stand it was sealed on,
// or for a review the candidate. Verdict is empty unless the Pass completed.
type PassRecord struct {
	ID       string          `json:"id"`
	Purpose  Purpose         `json:"purpose"`
	Review   bool            `json:"review"`
	Tree     string          `json:"tree,omitempty"`
	Verdict  Verdict         `json:"verdict,omitempty"`
	Sealed   bool            `json:"sealed"`
	Checks   []CheckResult   `json:"checks,omitempty"`
	Managers []ManagerRecord `json:"managers"`
}

// ManagerRecord is one Manager's participation in a Pass: its last state, its
// accepted basis and result and the result's verdict and summary.
type ManagerRecord struct {
	Manager string  `json:"manager"`
	State   State   `json:"state"`
	Basis   string  `json:"basis,omitempty"`
	Result  string  `json:"result,omitempty"`
	Verdict Verdict `json:"verdict,omitempty"`
	Summary string  `json:"summary,omitempty"`
}

// CheckResult is one required check on a review Pass's candidate.
type CheckResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// TurnRecord is one Agent turn as the Orchestrator observed it. Role is
// "manager", "reviewer" or a helper role. Mode is what the turn was admitted
// for: "work" on the Manager's own basis or result, "communication" while its
// productive work was blocked, "coordination" for decisions and answers after
// its basis, "review" in an independent review and "helper" for a delegated
// task.
type TurnRecord struct {
	ID      string   `json:"id"`
	Manager string   `json:"manager"`
	Role    string   `json:"role"`
	Pass    string   `json:"pass"`
	Kind    TurnKind `json:"kind"`
	Mode    string   `json:"mode"`
	// Context says how the turn received its standing context: brief (a
	// fresh conversation), history (a continued one) or rehydrated (a
	// continued one that may have lost it). Compacted is the provider's report
	// that it compacted the conversation during the turn.
	Context   string    `json:"context,omitempty"`
	Compacted bool      `json:"compacted,omitempty"`
	Started   time.Time `json:"started"`
	Ended     time.Time `json:"ended"`
	Usage     Usage     `json:"usage"`
	Stopped   bool      `json:"stopped"`
	Reports   []string  `json:"reports,omitempty"`
	Reason    string    `json:"reason,omitempty"`
}

// State is a Manager's scheduling state within one Pass.
type State string

const (
	// Ready: its start conditions hold and admitted work can be scheduled.
	Ready State = "ready"
	// Working: a turn is active or a submission of it is being checked.
	Working State = "working"
	// Waiting: a named answer, basis, child result or other dependency is missing.
	Waiting State = "waiting"
	// Done: its complete Pass responsibility is accepted.
	Done State = "done"
)

// PassResult is what one Pass established. Stopped says that the run ended
// during the Pass; Finish then reports why. Verdict is set when the Pass was
// sealed.
type PassResult struct {
	Record  PassRecord
	Verdict Verdict
	Stopped bool
	// Corrections are the correction purposes of the open genuine findings
	// this kind of Pass judges: those of work Passes after a work Pass, all
	// after a review.
	Corrections []Purpose
}

// Run is one bound run of the Orchestrator. Open binds it and holds the shared
// working tree; Pass runs one recursive Pass over the whole required Manager
// tree; Finish ends it. A Run is driven by one goroutine.
type Run struct{ e *engine }

// Open validates the configuration, records the start and materializes the
// shared work volume "<RunID>-work" from the start stand. An invalid
// configuration is an error and starts nothing; a run that cannot start is
// returned so that Finish reports why.
func Open(ctx context.Context, cfg Config) (*Run, error) {
	if err := validate(cfg); err != nil {
		return nil, err
	}
	e := newEngine(ctx, cfg)
	e.open()
	return &Run{e: e}, nil
}

// Pass runs one Pass of the declared purpose with a new identity: the base
// process for every required Manager, then the whole-stand exit gates and the
// sealed result. An invalid spec ends the run Failed.
func (r *Run) Pass(spec PassSpec) PassResult { return r.e.runPass(spec) }

// Reassess reports whether the last Pass was a review that only fresh
// independent disputes keep from being clean: its required checks passed on
// the candidate and every open genuine finding was disputed in that very Pass.
// A new complete review of the same unchanged candidate, with a new Pass
// identity, then decides the disputes instead of an artificial correction.
func (r *Run) Reassess() bool { return r.e.reassess() }

// Corrections returns the correction purposes the open genuine findings name,
// so the caller chooses the earliest necessary work Pass.
func (r *Run) Corrections() []Purpose { return r.e.corrections(false) }

// Finish ends the run: it waits for every turn and check, records the outcome
// and returns it. reason explains an unverified run that did not stop by
// itself, such as an exhausted Pass budget chosen by the caller.
func (r *Run) Finish(reason string) (Outcome, error) { return r.e.finish(reason) }

// ModelErrorReason begins the Failed reason of a run that Root ended because it
// judged the model erroneous or contradictory.
const ModelErrorReason = "Root judged the model erroneous or contradictory: "

// ModelAssessmentReason begins the Failed reason of a run whose model
// assessment ended with a material model problem its Root confirmed.
const ModelAssessmentReason = "the model assessment found a material model problem: "
