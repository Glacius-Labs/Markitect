package cockpit

import (
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Node identities of the two participants that are not Managers. A Manager's
// node identity is its Package path; "@" never starts one.
const (
	nodeOrchestrator = "@orchestrator"
	nodeUser         = "@user"
)

// recordedUser is the identity earlier runs recorded for the owner in their
// question events, when Root could still ask the user. It is read only so that
// old evidence still renders; a Run asks the owner nothing now.
const recordedUser = "user"

// View is what the inspector shows at one recorded boundary: the first Seq
// records of the run and nothing after them. Report content appears only at
// the final boundary of a run whose report exists.
type View struct {
	Run       RunHead        `json:"run"`
	Boundary  Boundary       `json:"boundary"`
	Source    Source         `json:"source"`
	Status    Status         `json:"status"`
	Account   []Sentence     `json:"account"`
	Passes    []PassView     `json:"passes"`
	Current   string         `json:"current,omitempty"`
	Managers  []ManagerView  `json:"managers"`
	Turns     []TurnView     `json:"turns"`
	Spans     []Span         `json:"spans"`
	Proposals []ProposalView `json:"proposals"`
	Findings  []FindingView  `json:"findings"`
	Questions []QuestionView `json:"questions"`
	Problems  []ProblemView  `json:"problems"`
	Probes    []ProbeView    `json:"probes"`
	Lookups   []LookupView   `json:"lookups"`
	Bindings  []BindingView  `json:"bindings"`
	// OwnerChannel says that the record holds owner-channel records (a
	// question to the owner or an owner error) of an earlier run. A Run no
	// longer has an owner channel, so the inspector draws the owner only then.
	OwnerChannel bool  `json:"ownerChannel"`
	Gaps         []Gap `json:"gaps"`
}

// RunHead is what run.start recorded, including the binding the caller gave
// the run: the strategy, the build and the budget. A run of an earlier build
// has no binding; those fields stay empty.
type RunHead struct {
	ID         string    `json:"id"`
	Started    time.Time `json:"started,omitzero"`
	Deadline   time.Time `json:"deadline,omitzero"`
	AgentTurns int       `json:"agentTurns,omitempty"`
	Passes     int       `json:"passes,omitempty"`
	Start      string    `json:"start,omitempty"`
	Managers   []string  `json:"managers"`
	// Strategy is the composition the run was bound to ("direct" or "staged"),
	// Work its work Passes in order and Corrections the work Passes a finding
	// may name.
	Strategy    string   `json:"strategy,omitempty"`
	Work        []string `json:"work,omitempty"`
	Corrections []string `json:"corrections,omitempty"`
	Build       string   `json:"build,omitempty"`
	Source      string   `json:"source,omitempty"`
	// ExecutionTime and TurnTimeout are the budget's durations as written.
	ExecutionTime string `json:"executionTime,omitempty"`
	TurnTimeout   string `json:"turnTimeout,omitempty"`
}

// Boundary is the recorded position of the view. Total is the number of
// records read; Final says that the view includes all of them.
type Boundary struct {
	Seq   int       `json:"seq"`
	At    time.Time `json:"at,omitzero"`
	Total int       `json:"total"`
	Final bool      `json:"final"`
}

// Status is the Run's recorded state at the boundary. Outcome is the
// Orchestrator's recorded outcome event; Report is only present at the final
// boundary. Without an outcome the record does not say whether the run still
// runs or was interrupted; live observation adds that separately.
type Status struct {
	Outcome    string `json:"outcome,omitempty"`
	Reason     string `json:"reason,omitempty"`
	OutcomeSeq int    `json:"outcomeSeq,omitempty"`
	// Ending names a closed ending the outcome and the record establish
	// together: "model_problem" when a Model assessment or Root confirmed a
	// material model problem. That ends the Run without publication; the
	// model is corrected outside it and a new Run starts.
	Ending string  `json:"ending,omitempty"`
	Report *Report `json:"report,omitempty"`
}

// Sentence is one factual statement of the account, split into segments of
// which some refer to an inspectable record.
type Sentence []Segment

// Segment is text, optionally referring to a detail ("manager:.", "proposal:P3").
type Segment struct {
	Text string `json:"text"`
	Ref  string `json:"ref,omitempty"`
}

// PassView is one Pass as recorded up to the boundary.
type PassView struct {
	ID         string              `json:"id"`
	Purpose    string              `json:"purpose"`
	Review     bool                `json:"review"`
	Writes     bool                `json:"writes"`
	StartTree  string              `json:"startTree,omitempty"`
	StartSeq   int                 `json:"startSeq"`
	Started    time.Time           `json:"started"`
	Correction bool                `json:"correction,omitempty"`
	Addresses  []string            `json:"addresses,omitempty"`
	Candidate  string              `json:"candidate,omitempty"`
	Checks     []CheckView         `json:"checks,omitempty"`
	Gates      []GateView          `json:"gates,omitempty"`
	Sealed     bool                `json:"sealed"`
	SealedSeq  int                 `json:"sealedSeq,omitempty"`
	SealedAt   time.Time           `json:"sealedAt,omitzero"`
	Verdict    string              `json:"verdict,omitempty"`
	Tree       string              `json:"tree,omitempty"`
	Managers   []run.ManagerRecord `json:"managers,omitempty"`
	Registry   int                 `json:"registrations,omitempty"`
	// Unchanged names the earlier Pass of the same purpose that reviewed this
	// very candidate: this review decides that Pass's disputes and suspicions
	// instead of judging a corrected candidate.
	Unchanged string `json:"unchanged,omitempty"`
	// Turns, CostUSD and OutputTokens sum the usage of the Pass's turns that
	// ended up to the boundary.
	Turns        int     `json:"turns,omitempty"`
	CostUSD      float64 `json:"costUSD,omitempty"`
	OutputTokens int64   `json:"outputTokens,omitempty"`
}

// CheckView is one required check on a review Pass's candidate.
type CheckView struct {
	Seq      int    `json:"seq"`
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	ExitCode int    `json:"exitCode"`
	Reason   string `json:"reason,omitempty"`
}

// GateView is one whole-stand gate: the counts of Model check and File check
// findings and the tree they ran on.
type GateView struct {
	Seq        int    `json:"seq"`
	At         string `json:"at"`
	ModelCheck *int   `json:"modelCheck,omitempty"`
	FileCheck  *int   `json:"fileCheck,omitempty"`
	Tree       string `json:"tree,omitempty"`
}

// ManagerView is one Manager in the current Pass.
type ManagerView struct {
	Path     string    `json:"path"`
	Label    string    `json:"label"`
	Parent   string    `json:"parent,omitempty"`
	Children []string  `json:"children"`
	Depth    int       `json:"depth"`
	State    string    `json:"state,omitempty"`
	Stage    string    `json:"stage,omitempty"`
	Since    int       `json:"since,omitempty"`
	Why      string    `json:"why,omitempty"`
	Wait     *Wait     `json:"wait,omitempty"`
	Turn     *TurnView `json:"turn,omitempty"`
	Checking bool      `json:"checking,omitempty"`
	Helpers  []Helper  `json:"helpers,omitempty"`
	Basis    string    `json:"basis,omitempty"`
	Result   string    `json:"result,omitempty"`
	Pending  string    `json:"pending,omitempty"`
	Findings []string  `json:"findings,omitempty"`
	Disabled bool      `json:"disabled,omitempty"`
	Released bool      `json:"released"`
	// ReleasedSeq is the record that released it in the current Pass.
	ReleasedSeq int `json:"releasedSeq,omitempty"`
	// Declared are the dependencies its accepted plan declares for the
	// current Pass's purpose, with whether each is accepted at the boundary.
	Declared []Dependency `json:"declared,omitempty"`
	// Context is the standing context the run bound for this Manager.
	Context *ContextCoverage `json:"context,omitempty"`
}

// ContextCoverage is what run.start bound as one Manager's model context: the
// digest of its Brief, the Definitions supplied in full and the Definitions
// supplied by reference only, which the Manager can look up.
type ContextCoverage struct {
	Brief     string   `json:"brief,omitempty"`
	Full      []string `json:"full"`
	Reference []string `json:"reference"`
}

// LookupView is one lookup of a Manager's turn: a Definition of the bound
// model, or its own Brief again.
type LookupView struct {
	Seq     int    `json:"seq"`
	Pass    string `json:"pass"`
	Manager string `json:"manager"`
	Turn    string `json:"turn,omitempty"`
	Kind    string `json:"kind"`
	ID      string `json:"id,omitempty"`
	Owner   string `json:"owner,omitempty"`
	Bytes   int    `json:"bytes"`
	Refused bool   `json:"refused,omitempty"`
	Helper  string `json:"helper,omitempty"`
}

// BindingView is one recorded dependency of a Manager's work on another
// Manager's accepted submission, or a refused binding that would close a cycle.
type BindingView struct {
	Seq      int    `json:"seq"`
	Pass     string `json:"pass"`
	Manager  string `json:"manager"`
	On       string `json:"on"`
	Stage    string `json:"stage"`
	Serves   string `json:"serves,omitempty"`
	Proposal string `json:"proposal,omitempty"`
	Rebound  bool   `json:"rebound,omitempty"`
	Cycle    string `json:"cycle,omitempty"`
}

// ProbeView is one probe: the claim and the expectation recorded before it ran,
// the receipt once it ran and the disposition of an adverse receipt. The
// probe's bytes and output stay in the record (probe:ID).
type ProbeView struct {
	ID          string                `json:"id"`
	Pass        string                `json:"pass"`
	Manager     string                `json:"manager"`
	Turn        string                `json:"turn,omitempty"`
	Obligation  string                `json:"obligation"`
	Claim       string                `json:"claim"`
	Expected    string                `json:"expected"`
	Check       string                `json:"check,omitempty"`
	StandKind   string                `json:"standKind"`
	Stand       string                `json:"stand,omitempty"`
	Path        string                `json:"path,omitempty"`
	Proposal    string                `json:"proposal,omitempty"`
	Rerun       string                `json:"rerun,omitempty"`
	RequestSeq  int                   `json:"requestSeq"`
	ReceiptSeq  int                   `json:"receiptSeq,omitempty"`
	Outcome     string                `json:"outcome,omitempty"`
	ExitCode    int                   `json:"exitCode"`
	TimedOut    bool                  `json:"timedOut,omitempty"`
	Uncertain   bool                  `json:"uncertain,omitempty"`
	Reason      string                `json:"reason,omitempty"`
	Adverse     bool                  `json:"adverse,omitempty"`
	DisposedSeq int                   `json:"disposedSeq,omitempty"`
	Disposition *run.ProbeDisposition `json:"disposition,omitempty"`
	VoidedSeq   int                   `json:"voidedSeq,omitempty"`
	VoidedWhy   string                `json:"voidedWhy,omitempty"`
}

// Dependency is one dependency an accepted plan declares: the owner's basis or
// result for a purpose, and why it is needed.
type Dependency struct {
	Owner    string `json:"owner"`
	Need     string `json:"need"`
	Pass     string `json:"pass"`
	Stage    string `json:"stage"`
	Accepted bool   `json:"accepted"`
}

// Helper is one active helper of a Manager.
type Helper struct {
	ID   string `json:"id"`
	Role string `json:"role"`
	Turn string `json:"turn,omitempty"`
}

// Wait is a recorded wait and what the record says it depends on. Text is the
// Orchestrator's own wording. Kind classifies its known formats; On names the
// participants that must act next and Refs the records it names. Cause follows
// those participants through the same view as far as they wait themselves.
type Wait struct {
	Text    string    `json:"text"`
	Kind    string    `json:"kind"`
	On      []string  `json:"on,omitempty"`
	Refs    []string  `json:"refs,omitempty"`
	Need    string    `json:"need,omitempty"`
	Since   int       `json:"since"`
	SinceAt time.Time `json:"sinceAt"`
	Agent   *Said     `json:"agent,omitempty"`
	Cause   []Step    `json:"cause,omitempty"`
	// Outdated says why the recorded wait no longer holds at the boundary:
	// the record shows the condition it names fulfilled after the wait began.
	// The Orchestrator does not record a changed wait while the state stays
	// Waiting, so the record has no newer wording; On then names the owners
	// of the still unaccepted dependencies of the accepted plan.
	Outdated string `json:"outdated,omitempty"`
}

// Said is a Manager's own report.wait: the contact, question or Manager it
// named, the stage (basis or result) it waits for when it named another
// Manager's accepted submission, and its reason, authored by the Agent.
type Said struct {
	Seq    int    `json:"seq"`
	For    string `json:"for"`
	Stage  string `json:"stage,omitempty"`
	Reason string `json:"reason"`
}

// Step is one participant in a wait's cause chain and its recorded state.
type Step struct {
	Node   string `json:"node"`
	State  string `json:"state,omitempty"`
	Doing  string `json:"doing,omitempty"`
	Wait   string `json:"wait,omitempty"`
	Depth  int    `json:"depth"`
	Repeat bool   `json:"repeat,omitempty"`
}

// TurnView is one Agent turn.
type TurnView struct {
	ID       string     `json:"id"`
	Short    string     `json:"short"`
	Pass     string     `json:"pass"`
	Manager  string     `json:"manager"`
	Role     string     `json:"role"`
	Kind     string     `json:"kind"`
	ReadOnly bool       `json:"readOnly"`
	Profile  string     `json:"profile,omitempty"`
	Resume   bool       `json:"resume,omitempty"`
	StartSeq int        `json:"startSeq"`
	Started  time.Time  `json:"started"`
	EndSeq   int        `json:"endSeq,omitempty"`
	Ended    time.Time  `json:"ended,omitzero"`
	Stopped  *bool      `json:"stopped,omitempty"`
	Reason   string     `json:"reason,omitempty"`
	Reports  []string   `json:"reports,omitempty"`
	Usage    *run.Usage `json:"usage,omitempty"`
	// Mode is what the turn was admitted for (work, communication,
	// coordination, review or helper). Context says how it received its
	// standing context: brief (a fresh conversation), history (a continued
	// one) or rehydrated (a continued one that may have lost it).
	Mode      string `json:"mode,omitempty"`
	Context   string `json:"context,omitempty"`
	Compacted bool   `json:"compacted,omitempty"`
	Probes    int    `json:"probes,omitempty"`
}

// Span is one recorded state of a Manager in a Pass, from the state change
// that began it to the one that ended it (ToSeq 0 while it lasts).
type Span struct {
	Manager string    `json:"manager"`
	Pass    string    `json:"pass"`
	State   string    `json:"state"`
	Wait    string    `json:"wait,omitempty"`
	FromSeq int       `json:"fromSeq"`
	From    time.Time `json:"from"`
	ToSeq   int       `json:"toSeq,omitempty"`
	To      time.Time `json:"to,omitzero"`
}

// ProposalView is one submitted basis or result and what became of it.
type ProposalView struct {
	ID       string    `json:"id"`
	Pass     string    `json:"pass"`
	Manager  string    `json:"manager"`
	Kind     string    `json:"kind"`
	Purpose  string    `json:"purpose,omitempty"`
	Turn     string    `json:"turn,omitempty"`
	Seq      int       `json:"seq"`
	At       time.Time `json:"at"`
	Summary  string    `json:"summary,omitempty"`
	Verdict  string    `json:"verdict,omitempty"`
	Status   string    `json:"status"`
	By       string    `json:"by,omitempty"`
	DecSeq   int       `json:"decisionSeq,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	Checked  *bool     `json:"checked,omitempty"`
	Raised   int       `json:"raised,omitempty"`
	Closes   []string  `json:"closes,omitempty"`
	Disputes []string  `json:"disputes,omitempty"`
	Children []string  `json:"children,omitempty"`
	Stand    string    `json:"stand,omitempty"`
	// Coverage is the obligation coverage of a Model assessment submission:
	// per Definition its status and the evidence behind it.
	Coverage []CoverageView `json:"coverage,omitempty"`
}

// CoverageView is the status of one own obligation in a submission with the
// evidence items that support its claims.
type CoverageView struct {
	Definition string         `json:"definition"`
	Status     string         `json:"status"`
	Note       string         `json:"note,omitempty"`
	Evidence   []EvidenceView `json:"evidence,omitempty"`
}

// EvidenceView supports one claim: a probe receipt or an inspected source
// location of the stand with the reviewer's reasoning.
type EvidenceView struct {
	Claim     string `json:"claim"`
	Receipt   string `json:"receipt,omitempty"`
	Source    string `json:"source,omitempty"`
	Location  string `json:"location,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
}

// FindingView is one finding as recorded up to the boundary.
type FindingView struct {
	run.Finding
	Seq       int `json:"seq"`
	ClosedSeq int `json:"closedSeq,omitempty"`
	// Effect says what the finding does to the Run at the boundary: "blocks"
	// (an open confirmed deviation or model problem), "investigate" (an open
	// suspicion, which names no correction and is decided by a fresh review),
	// "advisory" (a suggestion or an advisory artifact finding, which never
	// blocks) or "closed".
	Effect string `json:"effect"`
	// Class is the class of a Model assessment finding: model_error,
	// model_gap or unclassified. The engine carries it in the correction
	// field, because an assessment has no correction purpose.
	Class string `json:"class,omitempty"`
}

// QuestionView is one question and its recorded fate.
type QuestionView struct {
	run.Question
	Seq         int    `json:"seq"`
	AnsweredSeq int    `json:"answeredSeq,omitempty"`
	AnsweredBy  string `json:"answeredBy,omitempty"`
	LapsedSeq   int    `json:"lapsedSeq,omitempty"`
	Status      string `json:"status"`
}

// ProblemView is one raised problem.
type ProblemView struct {
	run.Problem
	Seq int `json:"seq"`
}

// Gap is a limitation of what the evidence establishes for this view.
type Gap struct {
	Code string `json:"code"`
	Text string `json:"text"`
	Seq  int    `json:"seq,omitempty"`
}
