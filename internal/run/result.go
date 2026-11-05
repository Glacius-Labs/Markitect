package run

import "time"

// ResultFormat is the version of the Result's JSON form.
const ResultFormat = 4

// Outcomes of a Run: only a landed one changed the owner's branch.
const (
	Landed    = "landed"
	NotLanded = "not-landed"
)

// States say how a Run ended, more precisely than its outcome.
const (
	// StateLanded: the result is on the owner's branch.
	StateLanded = "landed"
	// StateNothing: nothing is affected; no agent started.
	StateNothing = "nothing-to-do"
	// StateVerified: verify found every item conforming, and the required checks
	// pass.
	StateVerified = "verified"
	// StateFindings: verify found items that do not conform, decisions for the
	// owner, or required checks that do not pass.
	StateFindings = "findings"
	// StateCheckFailed: the spec has an error; no agent started.
	StateCheckFailed = "check-failed"
	// StateSpecProblem: the spec contradicts itself or leaves open something that
	// changes behaviour at the project's boundary. Nothing landed.
	StateSpecProblem = "spec-problem"
	// StateRefused: the repair rounds or the calls of submit ran out. Nothing
	// landed.
	StateRefused = "refused"
	// StateKept: the work passed but cannot land; it stays on its own branch.
	StateKept = "kept"
	// StateStopped: the owner ended the Run. Nothing landed.
	StateStopped = "stopped"
	// StateIncomplete: a deadline, a required check that could not start, or a
	// session that would not go on. Nothing it did is reported as success.
	StateIncomplete = "incomplete"
)

// Result is what a Run's Result page shows. The tool writes it from what
// happened, never an agent.
type Result struct {
	Format  int    `json:"format"`
	Run     string `json:"run"`
	Command string `json:"command"`
	Outcome string `json:"outcome"`
	State   string `json:"state"`
	// Why says in one line why the Run ended as it did.
	Why string `json:"why"`
	// Base is the commit impact was computed from, Start the commit the Run
	// started from and Work the commit its work ended in.
	Base  string `json:"base,omitempty"`
	Start string `json:"start,omitempty"`
	Work  string `json:"work,omitempty"`
	// Target is the owner's branch; Branch the branch a Result that does not land
	// stays on.
	Target string `json:"target"`
	Branch string `json:"branch,omitempty"`
	Full   bool   `json:"full"`

	// Items holds each item's latest answer.
	Items          []ResultItem    `json:"items"`
	OwnerDecisions []OwnerDecision `json:"ownerDecisions"`
	Findings       []ResultFinding `json:"findings"`
	// Unsettled lists what verify found that names no Definition: a required check
	// that fails, an error from check.
	Unsettled []string `json:"unsettled,omitempty"`
	// MissingLinks lists locations a reviewer named that the registry does not
	// list for that item.
	MissingLinks []string `json:"missingLinks,omitempty"`
	ImpactNotes  []string `json:"impactNotes,omitempty"`
	// Repairs counts the repair rounds the work took, and SentBack says what each
	// round went back for.
	Repairs  int      `json:"repairs"`
	SentBack []string `json:"sentBack,omitempty"`
	// Failure says why the session's last turn ended on an API error.
	Failure string `json:"failure,omitempty"`
	// Replaced lists the kept Run branches a landing covers; they are deleted with
	// it.
	Replaced []string `json:"replaced,omitempty"`
	// FullCheck says how many changes have landed on the branch since the last
	// full check.
	FullCheck FullCheck `json:"fullCheck"`
	Began     time.Time `json:"began"`
	Ended     time.Time `json:"ended"`

	// impact is what the Run's impact covers, for the trailers of its Result
	// commit.
	impact []string
}

// FullCheckEvery is how many changes may land after a full check before the next
// is due: a starting value, to adjust with evidence.
const FullCheckEvery = 10

// FullCheck is how many changes have landed on the branch since the last full
// check: a `verify --full` that ended verified or with findings, or a landed
// `apply --full`. The tool starts nothing by itself.
type FullCheck struct {
	// Landed counts the landings after the commit the newest full check examined,
	// or every landing when there was none.
	Landed int `json:"landed"`
	// Since is the commit the newest full check examined; empty when no full check
	// is known.
	Since string `json:"since,omitempty"`
	// Due is set from FullCheckEvery landings on.
	Due bool `json:"due"`
	// Note says why the count could not be read; Landed is then 0.
	Note string `json:"note,omitempty"`
}

// ResultItem is one checklist item with its latest answer.
type ResultItem struct {
	Item       string   `json:"item"`
	Kind       string   `json:"kind"`
	Definition string   `json:"definition,omitempty"`
	Look       []string `json:"look,omitempty"`
	// Answer is empty when no review answered it.
	Answer      string    `json:"answer,omitempty"`
	Pointers    []Pointer `json:"pointers,omitempty"`
	Explanation string    `json:"explanation,omitempty"`
}

// OwnerDecision is a decision only the owner can make, with a recommendation.
type OwnerDecision struct {
	Matter         string `json:"matter"`
	Recommendation string `json:"recommendation"`
}

// ResultFinding is one Finding of verify: a Definition, what was observed and
// what is expected. It is the input of apply --findings.
type ResultFinding struct {
	Definition string `json:"definition"`
	Observed   string `json:"observed"`
	Expected   string `json:"expected"`
}
