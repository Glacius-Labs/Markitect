package run

import "time"

// States say how gate ended the work or a review of it.
const (
	// StatePassed: every item has an answer and none is wrong; gate wrote a
	// certificate.
	StatePassed = "passed"
	// StateSpecProblem: the spec contradicts itself or leaves open something that
	// changes behaviour at the project's boundary. Nothing is certified.
	StateSpecProblem = "spec-problem"
	// StateRefused: the repair rounds ran out. Nothing is certified.
	StateRefused = "refused"
	// StateVerified: a review of finished work found every item conforming.
	StateVerified = "verified"
	// StateFindings: a review of finished work found items that do not conform,
	// or decisions for the owner.
	StateFindings = "findings"
)

// Result is what gate's Result page shows. The tool writes it from what happened,
// never an agent.
type Result struct {
	// Name names the page: gate-TIME-STATE.
	Name    string
	Command string
	State   string
	// Why says in one line why the work ended as it did.
	Why string
	// Base is the certificate the change started from, "none" without one.
	Base string
	// Certificate is the certificate gate wrote beside a passing Result.
	Certificate string
	Full        bool
	// Items holds each item's latest answer.
	Items []ResultItem
	// Reports are the spec problems the builder reported, with the reviewer's
	// word on each.
	Reports        []ResultReport
	OwnerDecisions []OwnerDecision
	Findings       []ResultFinding
	// Unsettled lists what a review of finished work found that names no
	// Definition, such as an error from check.
	Unsettled []string
	// MissingLinks lists locations a reviewer named that the registry does not
	// list for that item.
	MissingLinks []string
	ImpactNotes  []string
	// Repairs counts the repair rounds the work took, and SentBack says what each
	// round went back for.
	Repairs  int
	SentBack []string
	Began    time.Time
	Ended    time.Time
}

// ResultItem is one checklist item with its latest answer, as the certificate
// holds it.
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

// ResultReport is a spec problem the builder reported, and the reviewer's answer:
// "spec problem" confirmed it, "wrong" found that the spec settles it, and none
// means no reviewer has answered yet.
type ResultReport struct {
	Reported    string
	Answer      string
	Pointers    []Pointer
	Explanation string
}

// OwnerDecision is a decision only the owner can make, with a recommendation.
type OwnerDecision struct {
	Matter         string `json:"matter"`
	Recommendation string `json:"recommendation"`
}

// ResultFinding is one Finding of a review of finished work: a Definition, what
// was observed and what is expected. It is the input of checklist --findings.
type ResultFinding struct {
	Definition string
	Observed   string
	Expected   string
}
