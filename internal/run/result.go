package run

import (
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// ResultFormat is the version of the Result's JSON form.
const ResultFormat = 1

// Outcomes of a Run: only a landed Run changed the target branch.
const (
	Landed    = "landed"
	NotLanded = "not-landed"
)

// States say how a Run ended, more precisely than its outcome.
const (
	// StateLanded: the result is on the target branch.
	StateLanded = "landed"
	// StateNothing: the change reaches no Manager; no agent started.
	StateNothing = "nothing-to-do"
	// StateVerified: verify found every item conforming.
	StateVerified = "verified"
	// StateFindings: verify found items that do not conform.
	StateFindings = "findings"
	// StateCheckFailed: the initial gate failed; no agent started.
	StateCheckFailed = "check-failed"
	// StateModelDefect: a real model defect ended the Run at Root.
	StateModelDefect = "model-defect"
	// StateKept: the Run's work was approved but cannot land; it stays on
	// its Run branch.
	StateKept = "kept"
	// StateIncomplete: the Run stopped before its work was approved: a
	// budget, a deadline, an agent that made no progress, or a technical
	// failure. Nothing it did is reported as success.
	StateIncomplete = "incomplete"
)

// Result is what a Run returns and what its Result page shows (concept 2.0
// §6.4). The host writes it from commits, verdicts and checks, never an
// agent. Its JSON form is the evaluation suite's Result contract.
type Result struct {
	Format  int    `json:"format"`
	Run     string `json:"run"`
	Command string `json:"command"`
	Outcome string `json:"outcome"`
	State   string `json:"state"`
	// Why says in one line why the Run ended as it did.
	Why string `json:"why"`
	// Base is the commit impact was computed from, Start the commit the Run
	// started from and Landed the commit that holds the result on the
	// target branch.
	Base   string `json:"base,omitempty"`
	Start  string `json:"start,omitempty"`
	Landed string `json:"landed,omitempty"`
	// Target is the branch the Run lands on; Branch the Run branch a
	// result that does not land stays on.
	Target string `json:"target"`
	Branch string `json:"branch,omitempty"`
	Full   bool   `json:"full"`

	Items            []ResultItem        `json:"items"`
	NotParticipating []NotParticipant    `json:"notParticipating"`
	RequiredChecks   RequiredChecks      `json:"requiredChecks"`
	OwnerDecisions   []OwnerDecision     `json:"ownerDecisions"`
	Findings         []ResultFinding     `json:"findings"`
	Notes            []agentport.Note    `json:"notes"`
	ImpactNotes      []string            `json:"impactNotes,omitempty"`
	Starts           []AgentStart        `json:"starts"`
	Messages         []agentport.Message `json:"messages"`
	Usage            agentport.Usage     `json:"usage"`
	// Unkept lists approved work of a Run that does not land which its Run
	// branch does not hold.
	Unkept []string `json:"unkept,omitempty"`
	// Cleanup lists what may remain after the Run; empty means everything
	// it created is gone.
	Cleanup []string  `json:"cleanup"`
	Began   time.Time `json:"began"`
	Ended   time.Time `json:"ended"`
}

// ResultItem is one checklist item with its verdict, evidence and the
// checker that approved it.
type ResultItem struct {
	Manager    string `json:"manager"`
	Item       string `json:"item"`
	Part       string `json:"part"`
	Definition string `json:"definition,omitempty"`
	Verdict    string `json:"verdict,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
	// Submission is the commit that holds the verdict.
	Submission string `json:"submission,omitempty"`
	Checker    string `json:"checker,omitempty"`
	Approved   bool   `json:"approved"`
	// Check is the checker's own evidence for the item.
	Check string `json:"check,omitempty"`
}

// NotParticipant is a Package that did not work, and why.
type NotParticipant struct {
	Package string `json:"package"`
	Reason  string `json:"reason"`
}

// RequiredChecks says whether the required checks ran on the whole
// repository state that lands, or would land.
type RequiredChecks struct {
	Ran             bool                    `json:"ran"`
	WholeRepository bool                    `json:"wholeRepository"`
	Commit          string                  `json:"commit,omitempty"`
	Checks          []runremote.CheckResult `json:"checks"`
	Note            string                  `json:"note,omitempty"`
}

// OwnerDecision is a decision only the owner can make, with a
// recommendation.
type OwnerDecision struct {
	Manager        string `json:"manager"`
	Matter         string `json:"matter"`
	Recommendation string `json:"recommendation"`
}

// ResultFinding is one Finding of verify: a Definition, what was observed
// and what is expected. It is the input of apply --findings.
type ResultFinding struct {
	Definition string `json:"definition"`
	Observed   string `json:"observed"`
	Expected   string `json:"expected"`
	Manager    string `json:"manager"`
}

// AgentStart is one agent start and how it ended.
type AgentStart struct {
	Manager string           `json:"manager"`
	Role    agentport.Role   `json:"role"`
	Round   int              `json:"round"`
	Reason  agentport.Reason `json:"reason"`
	Detail  string           `json:"detail,omitempty"`
	// Class is the provider's failure class of a failed start.
	Class   agentport.FailureClass `json:"class,omitempty"`
	Agent   string                 `json:"agent,omitempty"`
	Usage   agentport.Usage        `json:"usage"`
	Seconds float64                `json:"seconds"`
}
