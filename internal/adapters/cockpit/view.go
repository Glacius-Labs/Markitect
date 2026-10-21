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
	Gaps      []Gap          `json:"gaps"`
}

// RunHead is what run.start recorded.
type RunHead struct {
	ID         string    `json:"id"`
	Started    time.Time `json:"started,omitzero"`
	Deadline   time.Time `json:"deadline,omitzero"`
	AgentTurns int       `json:"agentTurns,omitempty"`
	Passes     int       `json:"passes,omitempty"`
	Start      string    `json:"start,omitempty"`
	Managers   []string  `json:"managers"`
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
	Outcome    string  `json:"outcome,omitempty"`
	Reason     string  `json:"reason,omitempty"`
	OutcomeSeq int     `json:"outcomeSeq,omitempty"`
	Report     *Report `json:"report,omitempty"`
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

// Said is a Manager's own report.wait: the contact or question it named and
// its reason, authored by the Agent.
type Said struct {
	Seq    int    `json:"seq"`
	For    string `json:"for"`
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
}

// FindingView is one finding as recorded up to the boundary.
type FindingView struct {
	run.Finding
	Seq       int `json:"seq"`
	ClosedSeq int `json:"closedSeq,omitempty"`
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
