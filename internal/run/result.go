package run

import (
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
)

// ResultFormat is the version of the Result's JSON form.
const ResultFormat = 2

// Outcomes of a Command: only a landed one changed the target branch.
const (
	Landed    = "landed"
	NotLanded = "not-landed"
)

// States say how a Command ended, more precisely than its outcome.
const (
	// StateLanded: the result is on the target branch.
	StateLanded = "landed"
	// StateNothing: nothing is affected; no agent started.
	StateNothing = "nothing-to-do"
	// StateVerified: verify found every item conforming, and the required
	// checks pass.
	StateVerified = "verified"
	// StateFindings: verify found items that do not conform, decisions for the
	// owner, or required checks that do not pass.
	StateFindings = "findings"
	// StateCheckFailed: the spec has a real error; no agent started.
	StateCheckFailed = "check-failed"
	// StateSpecProblem: a session stopped because the spec contradicts itself
	// or leaves open something that changes behaviour at the project's
	// boundary. Nothing landed.
	StateSpecProblem = "spec-problem"
	// StateRefused: the repair rounds ran out with the gate or the reviewer
	// still refusing. Nothing landed.
	StateRefused = "refused"
	// StateKept: the work passed but cannot land; it stays on its own branch.
	StateKept = "kept"
	// StateIncomplete: a deadline, a provider's failure, a required check that
	// could not run, a review the Command could not accept or a technical
	// failure. Nothing it did is reported as success.
	StateIncomplete = "incomplete"
)

// Result is what a Command returns and what its Result page shows (concept 3.0
// §10). The Command writes it from what happened, never an agent. Its JSON form
// is the evaluation suite's Result contract.
type Result struct {
	Format  int    `json:"format"`
	Run     string `json:"run"`
	Command string `json:"command"`
	Outcome string `json:"outcome"`
	State   string `json:"state"`
	// Why says in one line why the Command ended as it did.
	Why string `json:"why"`
	// Base is the commit impact was computed from, Start the commit the
	// Command started from, Work the commit its session work ended in and
	// Landed the commit that holds the result on the target branch.
	Base   string `json:"base,omitempty"`
	Start  string `json:"start,omitempty"`
	Work   string `json:"work,omitempty"`
	Landed string `json:"landed,omitempty"`
	// Target is the branch it lands on; Branch the branch a Result that does
	// not land stays on.
	Target string `json:"target"`
	Branch string `json:"branch,omitempty"`
	Full   bool   `json:"full"`

	// Items holds each item's latest answer.
	Items []ResultItem `json:"items"`
	// Depth says how much checking really happened.
	Depth Depth `json:"depth"`
	// NotTested holds what the reviewer's closing words say it could not test.
	NotTested      string          `json:"notTested,omitempty"`
	RequiredChecks RequiredChecks  `json:"requiredChecks"`
	OwnerDecisions []OwnerDecision `json:"ownerDecisions"`
	Findings       []ResultFinding `json:"findings"`
	// MissingLinks lists locations a reviewer named that the registry does not
	// list for that item.
	MissingLinks []string `json:"missingLinks,omitempty"`
	ImpactNotes  []string `json:"impactNotes,omitempty"`
	// Repairs counts the repair rounds the work took, and SentBack says what each
	// round went back for.
	Repairs  int             `json:"repairs"`
	SentBack []string        `json:"sentBack,omitempty"`
	Sessions []SessionRecord `json:"sessions"`
	Usage    agentport.Usage `json:"usage"`
	// Replaced lists the kept Run branches a landing covers; they are deleted
	// with it.
	Replaced []string `json:"replaced,omitempty"`
	// FullCheck says how many changes have landed on the target since the last
	// full check.
	FullCheck FullCheck `json:"fullCheck"`
	// Cleanup lists what may remain after the Command; empty means everything
	// it created is gone. It is what the stored Result page says.
	Cleanup []string `json:"cleanup"`
	// AfterLanding lists what a landed Command's cleanup after landing left. The
	// landed page cannot say it, since it is part of the landing commit; empty
	// when nothing remains, absent when it did not land.
	AfterLanding []string  `json:"afterLanding,omitzero"`
	Began        time.Time `json:"began"`
	Ended        time.Time `json:"ended"`

	// impact is what the Command's impact covers, for the trailers of its
	// Result commit.
	impact []string
}

// FullCheckEvery is how many changes may land after a full check before the
// next is due: a starting value, to adjust with evidence.
const FullCheckEvery = 10

// FullCheck is how many changes have landed on the target since the last full
// check: a `verify --full` that ended verified or with findings, or a landed
// `apply --full`. The Command starts nothing by itself.
type FullCheck struct {
	// Landed counts the landings after the commit the newest full check
	// examined, or every landing when there was none.
	Landed int `json:"landed"`
	// Since is the commit the newest full check examined; empty when no full
	// check is known.
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

// Depth is how much checking really happened, as the Command measured it from
// the reviewers' transcripts (concept 3.0 §10). A pointer shows that someone
// looked, not that they judged correctly.
type Depth struct {
	// Measured: the provider's transcript shows reads at all.
	Measured bool `json:"measured"`
	// FilesOpened counts the files the transcripts show being read.
	FilesOpened int `json:"filesOpened"`
	// Commands counts the shell commands, Subagents the subagents.
	Commands  int `json:"commands"`
	Subagents int `json:"subagents"`
	// Pointers counts the answers' locations, of which the transcript shows
	// PointersOpened read and cannot tell for PointersNotMeasured.
	Pointers            int `json:"pointers"`
	PointersOpened      int `json:"pointersOpened"`
	PointersNotMeasured int `json:"pointersNotMeasured"`
	// RepeatedAnswers counts the answers that repeat another's words and
	// pointers.
	RepeatedAnswers int `json:"repeatedAnswers"`
}

// RequiredChecks says what the required checks ran on and whether they passed.
type RequiredChecks struct {
	Ran bool `json:"ran"`
	// Commit is the commit they ran on last.
	Commit string        `json:"commit,omitempty"`
	Checks []CheckResult `json:"checks"`
	Note   string        `json:"note,omitempty"`
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

// SessionRecord is one session and how it ended.
type SessionRecord struct {
	Role   agentport.Role   `json:"role"`
	Round  int              `json:"round"`
	Reason agentport.Reason `json:"reason"`
	Detail string           `json:"detail,omitempty"`
	// Class is the provider's failure class of a failed session.
	Class   agentport.FailureClass `json:"class,omitempty"`
	Agent   string                 `json:"agent,omitempty"`
	Models  []string               `json:"models,omitempty"`
	Usage   agentport.Usage        `json:"usage"`
	Seconds float64                `json:"seconds"`
}
