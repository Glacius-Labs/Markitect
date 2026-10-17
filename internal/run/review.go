package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Issue is a finding or question in the shared issue shape (core protocol,
// issues). A finding changes state only through an independent review (core
// rule I1), a question only through its addressee's reply (core rule I2).
// Source is the recorded review input that raised it.
type Issue struct {
	ID, Kind, Severity string
	Task, To, Purpose  string
	Definitions        []string
	Grounds, Request   string
	State              string
	Raised             StepID
	Source             BlobRef
	At                 uint64
	// Answer is the reply that resolved a question; ReplyStep the reply step
	// it admitted for a done task (core rule T11).
	Answer    string
	ReplyStep StepID
}

const (
	issueFinding       = "finding"
	issueQuestion      = "question"
	issueOpen          = "open"
	issueResolved      = "resolved"
	issueNotApplicable = "not-applicable"
	severityBlocking   = "blocking"
	severityAdvisory   = "advisory"
	addressedToOwner   = "owner"
)

// Delivery is a Host-accepted candidate (core rule D1). Aggregate is the
// aggregate version after its composition; At is the revision of acceptance.
type Delivery struct {
	Purpose, Scope string
	Proposal       BlobRef
	Candidate      BlobRef
	Aggregate      uint64
	At             uint64
}

// reviewValue is the progress of the current review: the expectations its
// first step recorded, which its candidate step receives (core rule T7).
type reviewValue struct {
	Expectations BlobRef
}

type reviewInput struct {
	Verdict     string            `json:"verdict"`
	Summary     string            `json:"summary"`
	Question    string            `json:"question,omitempty"`
	Evidence    []json.RawMessage `json:"evidence"`
	Findings    []findingInput    `json:"findings"`
	Resolutions []resolutionInput `json:"resolutions"`
}
type findingInput struct {
	To                string   `json:"to,omitempty"`
	Severity          string   `json:"severity"`
	Definitions       []string `json:"definitions"`
	Grounds           string   `json:"grounds"`
	RequestedEvidence string   `json:"requestedEvidence"`
}
type resolutionInput struct {
	Finding  string            `json:"finding"`
	State    string            `json:"state"`
	Grounds  string            `json:"grounds"`
	Evidence []json.RawMessage `json:"evidence"`
}

// ReviewVerdict is the Host's reading of a submitted review (core rule T8).
type ReviewVerdict string

const (
	VerdictAccept   ReviewVerdict = "accept"
	VerdictWithhold ReviewVerdict = "withhold"
	VerdictReject   ReviewVerdict = "reject"
	VerdictDecision ReviewVerdict = "decision"
	VerdictFresh    ReviewVerdict = "fresh"
)

// Issues returns the current run's issues of one task, in recording order.
func (c *Coordinator) Issues(manager string) []Issue {
	var out []Issue
	for _, id := range c.state.issueOrder {
		if issue := c.state.issues[id]; issue.Task == manager {
			issue.Definitions = slices.Clone(issue.Definitions)
			out = append(out, issue)
		}
	}
	return out
}

func (s stateValue) openIssues(manager string) []Issue {
	var out []Issue
	for _, id := range s.issueOrder {
		if issue := s.issues[id]; issue.Task == manager && issue.State == issueOpen {
			out = append(out, issue)
		}
	}
	return out
}

// checkReview refuses a review that cannot stand as a verdict: a resolution
// of an issue that is not an open finding of the task, or a pass that leaves
// an open blocking finding of the review purpose or raises a new one (core
// rule T8).
func (s stateValue) checkReview(t taskValue, in reviewInput) error {
	if !text(in.Summary) || in.Evidence == nil || in.Findings == nil || in.Resolutions == nil {
		return errors.New("submit_review needs a summary, evidence, findings and resolutions")
	}
	resolved := map[string]bool{}
	for _, r := range in.Resolutions {
		issue, ok := s.issues[r.Finding]
		if !ok || issue.Task != t.Slot.Package || issue.Kind != issueFinding || issue.State != issueOpen || resolved[r.Finding] {
			return fmt.Errorf("%q is not an open finding of this task", r.Finding)
		}
		if r.State != issueResolved && r.State != issueNotApplicable || !text(r.Grounds) || r.Evidence == nil {
			return errors.New("each resolution needs the state resolved or not-applicable, grounds and evidence")
		}
		resolved[r.Finding] = true
	}
	for _, f := range in.Findings {
		if f.Severity != severityBlocking && f.Severity != severityAdvisory || f.Definitions == nil || !text(f.Grounds) || !text(f.RequestedEvidence) {
			return errors.New("each finding needs the severity blocking or advisory, definitions, grounds and the requested evidence")
		}
		if f.To != "" && f.To != t.Slot.Package && (t.Submission.Purpose != "integration" || !s.descends(f.To, t.Slot.Package)) {
			return fmt.Errorf("a finding goes to the reviewed Manager or, in an integration review, to one of its descendants; %q is neither", f.To)
		}
	}
	switch in.Verdict {
	case "pass":
		if slices.ContainsFunc(in.Findings, func(f findingInput) bool { return f.Severity == severityBlocking }) {
			return errors.New("a pass cannot raise a blocking finding; request changes instead")
		}
		for _, issue := range s.openIssues(t.Slot.Package) {
			if issue.Kind == issueFinding && issue.Severity == severityBlocking && issue.Purpose == t.Submission.Purpose && !resolved[issue.ID] {
				return fmt.Errorf("a pass must resolve or mark not-applicable the open blocking finding %s", issue.ID)
			}
		}
	case "changes_requested":
		if !slices.ContainsFunc(in.Findings, func(f findingInput) bool { return f.Severity == severityBlocking }) {
			return errors.New("changes_requested needs at least one blocking finding")
		}
	case "needs_decision":
		if !text(in.Question) {
			return errors.New("needs_decision must state the decision the owner must make")
		}
	case "failed_assessment":
	default:
		return fmt.Errorf("unknown verdict %q", in.Verdict)
	}
	return nil
}

// verdictOf reads a checked review against the run: a pass is withheld while
// another open blocking issue covers the subject (core rule I3), or while a
// producer the candidate used delivered again after the candidate's step saw
// the aggregate (core rule D4).
func (s stateValue) verdictOf(t taskValue, in reviewInput) ReviewVerdict {
	switch in.Verdict {
	case "changes_requested":
		return VerdictReject
	case "needs_decision":
		return VerdictDecision
	case "failed_assessment":
		return VerdictFresh
	}
	resolved := map[string]bool{}
	for _, r := range in.Resolutions {
		resolved[r.Finding] = true
	}
	// An open blocking issue blocks its subject and every aggregate that
	// contains it (core rule I3).
	for _, manager := range s.subtree(t) {
		for _, issue := range s.openIssues(manager) {
			if issue.Severity == severityBlocking && !resolved[issue.ID] {
				return VerdictWithhold
			}
		}
	}
	for _, producer := range t.Submission.Uses {
		if slices.ContainsFunc(s.tasks[producer].Deliveries, func(d Delivery) bool { return d.Aggregate > t.Submission.Aggregate && d.Candidate != (BlobRef{}) }) {
			return VerdictWithhold
		}
	}
	return VerdictAccept
}

// ReviewVerdict reports how the Host reads the current candidate review's
// answered verdict, so that the Host composes an accepted delivery before it
// concludes the step (core rule D2).
func (c *Coordinator) ReviewVerdict(manager string, input Payload) (ReviewVerdict, error) {
	t, ok := c.state.tasks[manager]
	if !ok || t.State != TaskReviewing || t.Phase != PhaseReviewerCandidate || t.Step == "" {
		return "", ErrState
	}
	args, err := CallArguments(ToolSubmitReview, input.Bytes())
	if err != nil {
		return "", ErrInvalid
	}
	var in reviewInput
	if err := decodeArgs(args, &in); err != nil {
		return "", ErrInvalid
	}
	if err := c.state.checkReview(t, in); err != nil {
		return "", ErrInvalid
	}
	return c.state.verdictOf(t, in), nil
}

// StepTimeout is the bound step timeout of a Manager's steps; reviews of its
// task use it too.
func (c *Coordinator) StepTimeout(manager string) time.Duration {
	for _, m := range c.state.managerBasis.Settings {
		if m.Package == manager {
			return time.Duration(m.StepTimeout.Value)
		}
	}
	return 0
}

func (s stateValue) rejectionLimit(manager string) int {
	for _, m := range s.managerBasis.Settings {
		if m.Package == manager {
			return m.ReviewRounds.Value
		}
	}
	return 0
}

// concludeReview applies a review step's outcome (core rules T7, T8, T13, D2).
func (s *stateValue) concludeReview(r ConclusionRecord, t *taskValue, revision uint64, args []byte) error {
	fresh := func() {
		// A fresh review of the same frozen candidate; no rejection counts.
		t.Review, t.Since = reviewValue{}, revision
	}
	if r.Outcome == outcomeFailed || t.Phase == PhaseReviewerExpectations {
		if r.Aggregate != s.aggregateVersion || r.AggregateTree != (snapshot.TreeID{}) {
			return ErrState
		}
		if r.Outcome == outcomeFailed {
			fresh()
		} else {
			t.Review.Expectations, t.Since = r.Input, revision
		}
		return nil
	}
	var in reviewInput
	if json.Unmarshal(args, &in) != nil {
		return ErrInvalid
	}
	verdict := s.verdictOf(*t, in)
	composed := verdict == VerdictAccept && t.Submission.Candidate != (BlobRef{})
	want := s.aggregateVersion
	if composed {
		want++
	}
	if r.Aggregate != want || composed == (r.AggregateTree == (snapshot.TreeID{})) {
		return ErrState
	}
	if verdict == VerdictFresh {
		fresh()
		return nil
	}
	// A withheld pass does not stand, so it resolves nothing (core rule I1).
	s.recordReview(r, *t, in, verdict != VerdictWithhold)
	purpose, scope := t.Submission.Purpose, t.Submission.Scope
	t.Review = reviewValue{}
	switch verdict {
	case VerdictWithhold:
		t.State, t.Since, t.Submission = TaskReady, revision, Submission{}
	case VerdictDecision:
		// The owner's answer starts a fresh review of the same frozen
		// candidate (core rule T8); the run waits for it (core rule W1).
		id := fmt.Sprintf("question-%s", r.Step)
		s.addIssue(Issue{ID: id, Kind: issueQuestion, Severity: severityBlocking, Task: t.Slot.Package, To: addressedToOwner, Purpose: purpose, Request: in.Question, State: issueOpen, Raised: r.Step, Source: r.Input, At: revision})
		t.State, t.Since, t.Waits = TaskBlocked, revision, []Wait{{Kind: waitOwner, Ref: id}}
	case VerdictReject:
		t.Rejections = maps.Clone(t.Rejections)
		if t.Rejections == nil {
			t.Rejections = map[string]int{}
		}
		t.Rejections[purpose]++
		t.Submission = Submission{}
		// Findings about a descendant's artifacts reopen it; the reviewed
		// Manager integrates again once its children are done (core rule T11).
		corrected := false
		for _, f := range in.Findings {
			if f.To != "" && f.To != t.Slot.Package && f.Severity == severityBlocking {
				s.reopen(f.To, t.Slot.Package, revision)
				corrected = true
			}
		}
		switch {
		case s.rejectionLimit(t.Slot.Package) <= t.Rejections[purpose]:
			t.State, t.Reason = TaskIncomplete, fmt.Sprintf("the %s review rejected %d candidates, which reaches the rejection limit", purpose, t.Rejections[purpose])
		case corrected:
			t.State, t.Since, t.Waits = TaskBlocked, revision, []Wait{{Kind: waitChildren}}
		default:
			t.State, t.Since = TaskReady, revision
		}
	case VerdictAccept:
		if composed {
			s.aggregateVersion, s.aggregateTree = want, r.AggregateTree
			t.Own, t.OwnTree = BlobRef{}, snapshot.TreeID{}
		}
		t.Used = nil
		t.Deliveries = append(slices.Clone(t.Deliveries), Delivery{Purpose: purpose, Scope: scope, Proposal: t.Submission.Proposal, Candidate: t.Submission.Candidate, Aggregate: s.aggregateVersion, At: revision})
		t.Submission = Submission{}
		switch {
		case scope == "limited":
			t.State, t.Since = TaskReady, revision
		case purpose == "integration" || len(t.Slot.Children) == 0:
			t.State = TaskDone
		default:
			t.LocalDone = true
			t.State, t.Since, t.Waits = TaskBlocked, revision, []Wait{{Kind: waitChildren}}
		}
	}
	return nil
}

// recordReview records a review's findings, addressed to the reviewed task,
// and applies its resolutions when the review stands (core rule I1).
func (s *stateValue) recordReview(r ConclusionRecord, t taskValue, in reviewInput, resolve bool) {
	for _, res := range in.Resolutions {
		if !resolve {
			break
		}
		issue := s.issues[res.Finding]
		issue.State = res.State
		s.issues[res.Finding] = issue
	}
	for i, f := range in.Findings {
		to, purpose := t.Slot.Package, t.Submission.Purpose
		if f.To != "" && f.To != to {
			// A correction of a child's artifacts belongs to the child's own
			// delivery (core rule D5).
			to, purpose = f.To, "local"
		}
		s.addIssue(Issue{ID: fmt.Sprintf("finding-%s-%d", r.Step, i+1), Kind: issueFinding, Severity: f.Severity, Task: to, To: to, Purpose: purpose,
			Definitions: slices.Clone(f.Definitions), Grounds: f.Grounds, Request: f.RequestedEvidence, State: issueOpen, Raised: r.Step, Source: r.Input})
	}
}

// descends reports whether a Manager lies strictly below another.
func (s stateValue) descends(child, ancestor string) bool {
	for t, ok := s.tasks[child]; ok && t.Slot.Parent != ""; t, ok = s.tasks[t.Slot.Parent] {
		if t.Slot.Parent == ancestor {
			return true
		}
	}
	return false
}

// subtree lists a task's Manager and every descendant.
func (s stateValue) subtree(t taskValue) []string {
	out := []string{t.Slot.Package}
	for _, child := range t.Slot.Children {
		out = append(out, s.subtree(s.tasks[child])...)
	}
	return out
}

// reopen returns a done task that an open blocking finding addresses to work,
// and every ancestor below the reviewed Manager whose accepted integration
// contains it to waiting for its children; counters carry on (core rule T11).
func (s *stateValue) reopen(manager, reviewed string, revision uint64) {
	x := s.tasks[manager]
	if x.State == TaskDone {
		x.State, x.Since, x.LocalDone, x.Waits = TaskReady, revision, false, nil
		s.tasks[manager] = x
	}
	for p := x.Slot.Parent; p != "" && p != reviewed; p = s.tasks[p].Slot.Parent {
		a := s.tasks[p]
		if a.State == TaskDone {
			a.State, a.Since, a.Waits = TaskBlocked, revision, []Wait{{Kind: waitChildren}}
			s.tasks[p] = a
		}
	}
}

func (s *stateValue) addIssue(issue Issue) {
	s.issues[issue.ID] = issue
	s.issueOrder = append(s.issueOrder, issue.ID)
}
