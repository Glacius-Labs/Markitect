package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Report kinds a Manager submits through the tool report.
const (
	kindProgress = "progress"
	kindWait     = "wait"
	kindBasis    = "basis"
	kindResult   = "result"
	kindAccept   = "accept"
	kindReturn   = "return"
	kindReopen   = "reopen"
	kindRevise   = "revise"
	kindProblem  = "problem"
)

// Dependency stages of a plan.
const (
	stageNameBasis  = "basis"
	stageNameResult = "result"
)

// Coverage statuses of an own obligation in a basis or result.
const (
	coverDone          = "done"
	coverPreserved     = "preserved"
	coverNotApplicable = "not_applicable"
	coverOpen          = "open"
)

// Problem categories.
var problemCategories = []string{"model", "ownership", "dependency", "technical", "other"}

type proposalStatus string

const (
	proposalSubmitted   proposalStatus = "submitted"
	proposalChecked     proposalStatus = "checked"
	proposalAccepted    proposalStatus = "accepted"
	proposalReturned    proposalStatus = "returned"
	proposalFailed      proposalStatus = "failed_check"
	proposalReopened    proposalStatus = "reopened"
	proposalInvalidated proposalStatus = "invalidated"
	proposalWithdrawn   proposalStatus = "withdrawn"
)

// proposal is one submitted basis or complete result: the shared result
// envelope of every purpose.
type proposal struct {
	id     string
	kind   string
	member *member
	env    envelope
	status proposalStatus
	// composite is the accepted stand composed with the submitter's checked
	// area (work Passes with writes).
	composite snapshot.Tree
	decidedBy string
	reason    string
	submitted time.Time
	// raised, closing and disputes are the submission's finding effects; they
	// take effect only when the proposal is accepted.
	raised   []Finding
	closing  map[string]closure
	disputes map[string]Dispute
}

// envelope is the shared result envelope. Plan is the purpose payload of a
// planning Pass; Artifacts and Evidence are the payload of the other purposes.
// Findings and Closed are the finding references the submission recorded.
type envelope struct {
	Summary     string     `json:"summary"`
	Obligations []coverage `json:"obligations"`
	Plan        *plan      `json:"plan,omitempty"`
	Artifacts   []string   `json:"artifacts,omitempty"`
	Evidence    []string   `json:"evidence,omitempty"`
	Verdict     Verdict    `json:"verdict,omitempty"`
	Findings    []string   `json:"findings,omitempty"`
	Closed      []string   `json:"closed,omitempty"`
	Disputed    []string   `json:"disputed,omitempty"`
	Children    []string   `json:"children,omitempty"`
}

// coverage is the status of one own obligation.
type coverage struct {
	Definition string `json:"definition"`
	Status     string `json:"status"`
	Note       string `json:"note,omitempty"`
}

// plan is the payload of a planning Pass: the Manager's own concrete approach.
type plan struct {
	Approach     string       `json:"approach"`
	Artifacts    []string     `json:"artifacts"`
	Inputs       []planInput  `json:"inputs"`
	Outputs      []planOutput `json:"outputs"`
	Dependencies []dependency `json:"dependencies"`
	Order        string       `json:"order"`
	Blockers     []string     `json:"blockers"`
	Assumptions  []string     `json:"assumptions"`
}

type planInput struct {
	Need  string `json:"need"`
	Owner string `json:"owner"`
}

type planOutput struct {
	Provides  string   `json:"provides"`
	Consumers []string `json:"consumers"`
}

// dependency is a declared start condition: in the Pass of purpose Pass, the
// Manager starts only after Owner's basis or result is accepted.
type dependency struct {
	Owner string  `json:"owner"`
	Need  string  `json:"need"`
	Pass  Purpose `json:"pass"`
	Stage string  `json:"stage"`
}

type findingIn struct {
	Owner      string `json:"owner"`
	Obligation string `json:"obligation"`
	Observed   string `json:"observed"`
	Blocking   *bool  `json:"blocking"`
	Correction string `json:"correction"`
	Subject    string `json:"subject"`
	Definition string `json:"definition"`
}

type closeIn struct {
	ID       string `json:"id"`
	Reason   string `json:"reason"`
	As       string `json:"as"`
	Evidence string `json:"evidence"`
}

type disputeIn struct {
	ID       string `json:"id"`
	Reason   string `json:"reason"`
	Evidence string `json:"evidence"`
}

// closure is one validated closure of a submission.
type closure struct {
	reason, as, evidence string
}

type reportIn struct {
	Kind        string      `json:"kind"`
	Summary     string      `json:"summary"`
	For         string      `json:"for"`
	Proposal    string      `json:"proposal"`
	Child       string      `json:"child"`
	Reason      string      `json:"reason"`
	Obligations []coverage  `json:"obligations"`
	Plan        *plan       `json:"plan"`
	Artifacts   []string    `json:"artifacts"`
	Evidence    []string    `json:"evidence"`
	Verdict     string      `json:"verdict"`
	Findings    []findingIn `json:"findings"`
	Closed      []closeIn   `json:"closed"`
	Disputed    []disputeIn `json:"disputed"`
	Category    string      `json:"category"`
	Text        string      `json:"text"`
	Abort       bool        `json:"abort"`
}

// fields lists the fields a report kind may carry besides kind.
var reportFields = map[string][]string{
	kindProgress: {"summary"},
	kindWait:     {"for", "reason"},
	kindBasis:    {"summary", "obligations", "plan", "artifacts", "evidence", "findings", "closed", "disputed"},
	kindResult:   {"summary", "obligations", "plan", "artifacts", "evidence", "verdict", "findings", "closed", "disputed"},
	kindAccept:   {"proposal"},
	kindReturn:   {"proposal", "reason"},
	kindReopen:   {"child", "reason"},
	kindRevise:   {"reason"},
	kindProblem:  {"category", "text", "abort"},
}

// present lists the fields of a report that carry a value.
func (in reportIn) present() []string {
	var out []string
	add := func(name string, set bool) {
		if set {
			out = append(out, name)
		}
	}
	add("summary", in.Summary != "")
	add("for", in.For != "")
	add("proposal", in.Proposal != "")
	add("child", in.Child != "")
	add("reason", in.Reason != "")
	add("obligations", in.Obligations != nil)
	add("plan", in.Plan != nil)
	add("artifacts", in.Artifacts != nil)
	add("evidence", in.Evidence != nil)
	add("verdict", in.Verdict != "")
	add("findings", in.Findings != nil)
	add("closed", in.Closed != nil)
	add("disputed", in.Disputed != nil)
	add("category", in.Category != "")
	add("text", in.Text != "")
	add("abort", in.Abort)
	return out
}

// report serves the tool report of a Manager's turn.
func (e *engine) report(m *member, ts *turnState, args json.RawMessage) ToolAnswer {
	var in reportIn
	if err := decodeStrict(args, &in); err != nil {
		return refuse("%v", err)
	}
	allowed, ok := reportFields[in.Kind]
	if !ok {
		return refuse("kind must be one of progress, wait, basis, result, accept, return, reopen, revise or problem.")
	}
	for _, field := range in.present() {
		if !slices.Contains(allowed, field) {
			return refuse("%s is not part of a report of kind %s.", field, in.Kind)
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if m.p != e.cur || m.p.sealed {
		return refuse("This turn belongs to %s, which no longer accepts reports.", m.p.id)
	}
	if m.stage == stageStart {
		return refuse("Your work in this Pass was reset; wait until it starts again.")
	}
	if m.stage == stageBlocked && in.Kind != kindWait && in.Kind != kindProblem {
		return refuse("Your own work waits for %s. Until then you may answer, ask, wait or report a problem; you cannot submit, report progress or decide.", m.waiting)
	}
	var answer ToolAnswer
	switch in.Kind {
	case kindProgress:
		answer = e.reportProgress(m, ts, in)
	case kindWait:
		answer = e.reportWait(m, ts, in)
	case kindBasis, kindResult:
		answer = e.submit(m, ts, in)
	case kindAccept, kindReturn:
		answer = e.decide(m, ts, in)
	case kindReopen:
		answer = e.reopen(m, ts, in)
	case kindRevise:
		answer = e.revise(m, ts, in)
	case kindProblem:
		answer = e.problem(m, ts, in)
	}
	if !answer.Refused {
		ts.reports = append(ts.reports, in.Kind)
	}
	return answer
}

func (e *engine) reportProgress(m *member, ts *turnState, in reportIn) ToolAnswer {
	if m.stage != stageOwn && m.stage != stageIntegrate {
		return refuse("progress is reported while you work on your basis or your complete result.")
	}
	summary, err := toolText("summary", in.Summary)
	if err != nil {
		return refuse("%v", err)
	}
	ts.progress = summary
	_ = e.record(Event{Kind: "report.progress", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"summary": summary}})
	return ToolAnswer{Text: "Progress recorded. You continue in your next turn."}
}

func (e *engine) reportWait(m *member, ts *turnState, in reportIn) ToolAnswer {
	what := strings.TrimSpace(in.For)
	if q := e.questionLocked(what); q != nil {
		if q.From != m.Path || q.Answered || q.Lapsed {
			return refuse("%s is not an open question of yours.", what)
		}
	} else if c, ok := m.p.byPath[what]; !ok || !slices.Contains(m.Children, what) || c.stage == stageDone {
		return refuse("for names an open question you asked (Q...) or a child that is not Done.")
	}
	ts.wait = what
	_ = e.record(Event{Kind: "report.wait", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"for": what, "reason": strings.TrimSpace(in.Reason)}})
	return ToolAnswer{Text: "Recorded: you wait for " + what + ". You are woken when work for you arrives."}
}

// submit validates and records a basis or a complete result. It is checked
// and decided after the turn ended.
func (e *engine) submit(m *member, ts *turnState, in reportIn) ToolAnswer {
	p := m.p
	switch {
	case ts.proposal != nil:
		return refuse("You already submitted %s in this turn.", ts.proposal.id)
	case len(ts.delegations) > 0 || len(m.helpers) > 0:
		return refuse("Your helpers are still working; submit after their results reached you.")
	case ts.blocked:
		return refuse("This turn started while your own work was blocked, so it only answers what is asked of you; your work starts in your next turn.")
	case in.Kind == kindBasis && m.stage != stageOwn:
		return refuse("A basis is submitted while you work on your own contribution; you are at %s.", m.stage)
	case in.Kind == kindResult && m.stage != stageIntegrate:
		if m.stage == stageOwn {
			return refuse("Submit your basis first: your children start from it. Your complete result follows after their results are accepted.")
		}
		return refuse("A complete result is submitted after all your children's results are accepted; you are at %s.", m.stage)
	}
	summary, err := toolText("summary", in.Summary)
	if err != nil {
		return refuse("%v", err)
	}
	env := envelope{Summary: summary}
	if env.Obligations, err = e.coverageOf(m, in); err != nil {
		return refuse("%v", err)
	}
	if p.spec.Plans {
		if in.Plan == nil {
			return refuse("plan is required in this Pass.")
		}
		if err := e.checkPlan(m, in.Plan); err != nil {
			return refuse("plan: %v", err)
		}
		env.Plan = in.Plan
	} else if in.Plan != nil {
		return refuse("plan is only part of a planning Pass.")
	}
	if env.Artifacts, err = e.artifactsOf(m, in.Artifacts); err != nil {
		return refuse("%v", err)
	}
	for i, ev := range in.Evidence {
		text, err := toolText(fmt.Sprintf("evidence[%d]", i), ev)
		if err != nil {
			return refuse("%v", err)
		}
		env.Evidence = append(env.Evidence, text)
	}
	raised, err := e.findingsOf(m, in.Findings)
	if err != nil {
		return refuse("%v", err)
	}
	closing, err := e.closuresOf(m, in.Closed)
	if err != nil {
		return refuse("%v", err)
	}
	disputes, err := e.disputesOf(m, in.Disputed)
	if err != nil {
		return refuse("%v", err)
	}
	if in.Kind == kindResult {
		switch Verdict(in.Verdict) {
		case Satisfied, NeedsCorrection:
			env.Verdict = Verdict(in.Verdict)
		default:
			return refuse("verdict must be %q or %q.", Satisfied, NeedsCorrection)
		}
		for _, q := range e.questions {
			if q.From == m.Path && !q.Answered && !q.Lapsed {
				return refuse("Your question %s is unanswered; a complete result cannot hide it. Wait for the answer first.", q.ID)
			}
		}
		if err := e.verdictConsistent(m, env, raised, closing); err != nil {
			return refuse("%v", err)
		}
	} else if in.Verdict != "" {
		return refuse("verdict belongs to a complete result.")
	}
	if in.Kind == kindResult {
		for _, c := range m.Children {
			if r := p.byPath[c].result; r != nil {
				env.Children = append(env.Children, r.id)
			}
		}
	}
	e.nextProposal++
	pr := &proposal{id: fmt.Sprintf("P%d", e.nextProposal), kind: in.Kind, member: m, env: env, status: proposalSubmitted, submitted: e.now(),
		raised: raised, closing: closing, disputes: disputes}
	p.proposals = append(p.proposals, pr)
	ts.proposal = pr
	_ = e.record(Event{Kind: "proposal.submitted", Pass: p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"proposal": pr.id, "kind": pr.kind, "purpose": p.spec.Purpose, "envelope": env,
		"findings": raised, "closed": slices.Sorted(mapKeys(closing)), "disputed": slices.Sorted(mapKeys(disputes))}})
	text := fmt.Sprintf("Submitted %s %s. It is checked when this turn ends", in.Kind, pr.id)
	if m.Parent != "" {
		text += " and then decided by your parent"
	}
	return ToolAnswer{Text: text + "."}
}

// coverageOf validates the obligation coverage of a submission: every entry is
// an own obligation once, with a known status and a note where one is needed.
// A complete result covers every own obligation; a planning Pass needs the
// method and intended evidence of each covered obligation.
func (e *engine) coverageOf(m *member, in reportIn) ([]coverage, error) {
	seen := map[string]bool{}
	var out []coverage
	for i, c := range in.Obligations {
		if !slices.Contains(m.Obligations, c.Definition) {
			return nil, fmt.Errorf("obligations[%d]: %q is not one of your own obligations", i, c.Definition)
		}
		if seen[c.Definition] {
			return nil, fmt.Errorf("obligations[%d]: %q is listed twice", i, c.Definition)
		}
		seen[c.Definition] = true
		switch c.Status {
		case coverDone, coverPreserved, coverNotApplicable, coverOpen:
		default:
			return nil, fmt.Errorf("obligations[%d]: status must be done, preserved, not_applicable or open", i)
		}
		note := strings.TrimSpace(c.Note)
		needsNote := c.Status == coverNotApplicable || c.Status == coverOpen || m.p.spec.Plans
		if needsNote && note == "" {
			return nil, fmt.Errorf("obligations[%d]: a note is required for %s", i, c.Definition)
		}
		if note != "" {
			if _, err := toolText(fmt.Sprintf("obligations[%d].note", i), note); err != nil {
				return nil, err
			}
		}
		out = append(out, coverage{Definition: c.Definition, Status: c.Status, Note: note})
	}
	if in.Kind == kindResult {
		var missing []string
		for _, id := range m.Obligations {
			if !seen[id] {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("a complete result covers every own obligation; missing: %s", strings.Join(missing, ", "))
		}
	}
	return out, nil
}

// checkPlan validates a plan payload: a concrete approach and dependencies,
// inputs and outputs that name actual Managers, without a dependency cycle.
func (e *engine) checkPlan(m *member, pl *plan) error {
	if _, err := toolText("approach", pl.Approach); err != nil {
		return err
	}
	known := func(owner string) bool { return e.byPath[owner] != nil }
	for i, a := range pl.Artifacts {
		if _, err := snapshot.NewPath(strings.TrimSuffix(a, "/")); err != nil {
			return fmt.Errorf("artifacts[%d] %q is not a repository-relative path", i, a)
		}
	}
	for i, in := range pl.Inputs {
		if _, err := toolText(fmt.Sprintf("inputs[%d].need", i), in.Need); err != nil {
			return err
		}
		if !known(in.Owner) {
			return fmt.Errorf("inputs[%d]: owner %q is no Manager of the run", i, in.Owner)
		}
	}
	for i, out := range pl.Outputs {
		if _, err := toolText(fmt.Sprintf("outputs[%d].provides", i), out.Provides); err != nil {
			return err
		}
		for _, c := range out.Consumers {
			if !known(c) {
				return fmt.Errorf("outputs[%d]: consumer %q is no Manager of the run", i, c)
			}
		}
	}
	work := slices.DeleteFunc(slices.Clone(m.p.spec.Corrections), func(p Purpose) bool { return p == m.p.spec.Purpose })
	for i, d := range pl.Dependencies {
		switch {
		case !known(d.Owner) || d.Owner == m.Path:
			return fmt.Errorf("dependencies[%d]: owner %q must be another Manager of the run", i, d.Owner)
		case !slices.Contains(work, d.Pass):
			return fmt.Errorf("dependencies[%d]: pass must be one of %v", i, work)
		case d.Stage != stageNameBasis && d.Stage != stageNameResult:
			return fmt.Errorf("dependencies[%d]: stage must be basis or result", i)
		}
		if _, err := toolText(fmt.Sprintf("dependencies[%d].need", i), d.Need); err != nil {
			return err
		}
	}
	if cycle := e.planCycleLocked(m, pl); cycle != "" {
		return fmt.Errorf("the dependencies form a cycle with the accepted plans and the parent-first order: %s", cycle)
	}
	return nil
}

// planCycleLocked looks for a cycle among the start, basis and result events
// of every Manager in each later work purpose: a basis follows the start, a
// result follows the basis and every child's result, a child starts after its
// parent's basis, and every declared dependency precedes its Manager's start.
// The candidate plan replaces the Manager's earlier plan of this Pass.
func (e *engine) planCycleLocked(m *member, candidate *plan) string {
	plans := map[string]*plan{m.Path: candidate}
	for _, x := range m.p.members {
		if x == m {
			continue
		}
		for _, pr := range []*proposal{x.result, x.pending, x.basis} {
			if pr != nil && pr.env.Plan != nil && pr.status != proposalInvalidated && pr.status != proposalReturned && pr.status != proposalFailed {
				plans[x.Path] = pr.env.Plan
				break
			}
		}
	}
	for _, purpose := range m.p.spec.Corrections {
		type node struct {
			manager string
			event   int // 0 start, 1 basis, 2 result
		}
		edges := map[node][]node{}
		for _, x := range e.team {
			start, basis, result := node{x.Path, 0}, node{x.Path, 1}, node{x.Path, 2}
			edges[basis] = append(edges[basis], start)
			edges[result] = append(edges[result], basis)
			for _, c := range x.Children {
				edges[result] = append(edges[result], node{c, 2})
			}
			if x.Parent != "" {
				edges[start] = append(edges[start], node{x.Parent, 1})
			}
			if pl := plans[x.Path]; pl != nil {
				for _, d := range pl.Dependencies {
					if d.Pass == purpose {
						ev := 1
						if d.Stage == stageNameResult {
							ev = 2
						}
						edges[start] = append(edges[start], node{d.Owner, ev})
					}
				}
			}
		}
		const (
			white = iota
			grey
			black
		)
		color := map[node]int{}
		var path []node
		var found []node
		var visit func(n node) bool
		visit = func(n node) bool {
			color[n] = grey
			path = append(path, n)
			for _, next := range edges[n] {
				switch color[next] {
				case grey:
					i := slices.Index(path, next)
					found = slices.Clone(path[i:])
					return true
				case white:
					if visit(next) {
						return true
					}
				}
			}
			path = path[:len(path)-1]
			color[n] = black
			return false
		}
		for _, x := range e.team {
			for ev := 0; ev < 3; ev++ {
				if n := (node{x.Path, ev}); color[n] == white && visit(n) {
					names := []string{"start", "basis", "result"}
					var parts []string
					for _, f := range found {
						parts = append(parts, f.manager+" "+names[f.event])
					}
					return fmt.Sprintf("in %s: %s", purpose, strings.Join(parts, " needs "))
				}
			}
		}
	}
	return ""
}

// artifactsOf validates the artifact references of a submission: in a work
// Pass they name the submitter's own artifacts.
func (e *engine) artifactsOf(m *member, artifacts []string) ([]string, error) {
	var out []string
	for i, a := range artifacts {
		path, err := snapshot.NewPath(strings.TrimSuffix(a, "/"))
		if err != nil {
			return nil, fmt.Errorf("artifacts[%d] %q is not a repository-relative path", i, a)
		}
		if !m.p.spec.Review && !m.p.spec.Plans {
			if owner := e.cfg.OwnerOf(path); owner != m.Path {
				return nil, fmt.Errorf("artifacts[%d] %q is owned by %s, not by you; reference only your own artifacts", i, a, owner)
			}
		}
		out = append(out, a)
	}
	return out, nil
}

// findingsOf validates the findings of a submission: each names the
// submitter or a descendant as owner, the obligation, the observed evidence,
// whether it is genuine and the correction purpose it needs.
func (e *engine) findingsOf(m *member, in []findingIn) ([]Finding, error) {
	if len(in) > maxToolItems {
		return nil, fmt.Errorf("at most %d findings per report", maxToolItems)
	}
	spec := m.p.spec
	var out []Finding
	for i, f := range in {
		if f.Blocking == nil {
			return nil, fmt.Errorf("findings[%d]: blocking is required", i)
		}
		if !e.within(f.Owner, m.Path) {
			return nil, fmt.Errorf("findings[%d]: owner %q must be you (%q) or one of your descendants", i, f.Owner, m.Path)
		}
		obligation, err := toolText(fmt.Sprintf("findings[%d].obligation", i), f.Obligation)
		if err != nil {
			return nil, err
		}
		observed, err := toolText(fmt.Sprintf("findings[%d].observed", i), f.Observed)
		if err != nil {
			return nil, err
		}
		add := Finding{Owner: f.Owner, RaisedBy: m.Path, Obligation: obligation, Observed: observed, Blocking: *f.Blocking, Correction: spec.defaultCorrection()}
		if f.Correction != "" {
			if !slices.Contains(spec.Corrections, Purpose(f.Correction)) {
				return nil, fmt.Errorf("findings[%d]: correction must be one of %v", i, spec.Corrections)
			}
			add.Correction = Purpose(f.Correction)
		}
		switch {
		case spec.ArtifactFindingsAdvisory && f.Subject == subjectModel:
			add.Subject = subjectModel
		case spec.ArtifactFindingsAdvisory && f.Subject == subjectArtifact:
			if strings.TrimSpace(f.Definition) == "" {
				return nil, fmt.Errorf("findings[%d]: definition is required when subject is %q", i, subjectArtifact)
			}
			if e.cfg.ResolveDefinition == nil || !e.cfg.ResolveDefinition(m.p.tree, f.Definition) {
				return nil, fmt.Errorf("findings[%d]: %q is not a Definition of the candidate model", i, f.Definition)
			}
			// An artifact that deviates from the model is reported, never repaired
			// by the model's Managers, so it cannot block.
			add.Subject, add.Definition, add.Blocking = subjectArtifact, f.Definition, false
		case spec.ArtifactFindingsAdvisory:
			return nil, fmt.Errorf("findings[%d]: subject must be %q or %q", i, subjectArtifact, subjectModel)
		case f.Subject != "" || f.Definition != "":
			return nil, fmt.Errorf("findings[%d]: subject and definition are not part of this Pass's findings", i)
		}
		out = append(out, add)
	}
	return out, nil
}

// closuresOf validates closures: a finding closes only through a submission of
// its raiser's responsibility in a later Pass of the same kind, with a reason.
// A review finding is resolved on a changed candidate; on the unchanged
// candidate it can only be disproved, with evidence, as erroneous.
func (e *engine) closuresOf(m *member, in []closeIn) (map[string]closure, error) {
	if len(in) > maxToolItems {
		return nil, fmt.Errorf("at most %d closures per report", maxToolItems)
	}
	closing := map[string]closure{}
	for i, c := range in {
		f := e.findingLocked(c.ID)
		switch {
		case f == nil || f.RaisedBy != m.Path:
			return nil, fmt.Errorf("closed[%d]: %q is not a finding you raised", i, c.ID)
		case f.Closed || closing[c.ID].reason != "":
			return nil, fmt.Errorf("closed[%d]: %q is already closed", i, c.ID)
		case f.Pass == m.p.id:
			return nil, fmt.Errorf("closed[%d]: %q was raised in this Pass; a later Pass assesses it", i, c.ID)
		case f.Review != m.p.spec.Review:
			return nil, fmt.Errorf("closed[%d]: %q was raised in another kind of Pass; only that kind closes it", i, c.ID)
		}
		reason, err := toolText(fmt.Sprintf("closed[%d].reason", i), c.Reason)
		if err != nil {
			return nil, err
		}
		as := c.As
		if as == "" {
			as = ClosedResolved
		}
		unchanged := f.Review && f.Candidate == m.p.tree.Digest().String()
		var evidence string
		switch {
		case as != ClosedResolved && as != ClosedDisproved:
			return nil, fmt.Errorf("closed[%d].as must be %q or %q", i, ClosedResolved, ClosedDisproved)
		case as == ClosedResolved && unchanged:
			return nil, fmt.Errorf("closed[%d]: %q was raised on this unchanged candidate, so it cannot be resolved here; close it as disproved with evidence only if it is erroneous", i, c.ID)
		}
		if evidence, err = toolText(fmt.Sprintf("closed[%d].evidence", i), c.Evidence); err != nil {
			return nil, fmt.Errorf("%v: a closure states its evidence", err)
		}
		closing[c.ID] = closure{reason: reason, as: as, evidence: evidence}
	}
	return closing, nil
}

// disputesOf validates disputes: in a review, a reviewer other than the raiser
// whose responsibility holds the raiser or the finding's owner may object
// explicitly, with reason and evidence, to an open genuine review finding, once
// per candidate. A failed required check cannot be disputed.
func (e *engine) disputesOf(m *member, in []disputeIn) (map[string]Dispute, error) {
	if len(in) > 0 && !m.p.spec.Review {
		return nil, errors.New("disputed belongs to an independent review")
	}
	if len(in) > maxToolItems {
		return nil, fmt.Errorf("at most %d disputes per report", maxToolItems)
	}
	candidate := m.p.tree.Digest().String()
	disputes := map[string]Dispute{}
	for i, d := range in {
		f := e.findingLocked(d.ID)
		switch {
		case f == nil || f.Closed || !f.Blocking || !f.Review:
			return nil, fmt.Errorf("disputed[%d]: %q is no open genuine review finding", i, d.ID)
		case f.RaisedBy == MarkitectFinding:
			return nil, fmt.Errorf("disputed[%d]: %q is a failed required check; only a passing check on a candidate closes it", i, d.ID)
		case f.RaisedBy == m.Path:
			return nil, fmt.Errorf("disputed[%d]: you raised %q; close it as disproved with evidence instead", i, d.ID)
		case !e.within(f.RaisedBy, m.Path) && !e.within(f.Owner, m.Path):
			return nil, fmt.Errorf("disputed[%d]: neither the raiser nor the owner of %q is within your responsibility", i, d.ID)
		case slices.ContainsFunc(f.Disputes, func(x Dispute) bool { return x.Candidate == candidate }) || disputes[d.ID].By != "":
			return nil, fmt.Errorf("disputed[%d]: %q is already disputed on this candidate", i, d.ID)
		}
		reason, err := toolText(fmt.Sprintf("disputed[%d].reason", i), d.Reason)
		if err != nil {
			return nil, err
		}
		evidence, err := toolText(fmt.Sprintf("disputed[%d].evidence", i), d.Evidence)
		if err != nil {
			return nil, err
		}
		disputes[d.ID] = Dispute{By: m.Path, Pass: m.p.id, Candidate: candidate, Reason: reason, Evidence: evidence}
	}
	return disputes, nil
}

// reassess reports whether only fresh disputes keep the last review from being
// clean.
func (e *engine) reassess() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.passes) == 0 {
		return false
	}
	p := e.passes[len(e.passes)-1]
	if !p.spec.Review || !p.sealed || p.verdict != NeedsCorrection || len(p.checks) != len(e.cfg.Checks) {
		return false
	}
	for _, c := range p.checks {
		if !c.Passed {
			return false
		}
	}
	open := 0
	for _, f := range e.findings {
		if !f.Blocking || f.Closed {
			continue
		}
		open++
		if !slices.ContainsFunc(f.Disputes, func(d Dispute) bool { return d.Pass == p.id }) {
			return false
		}
	}
	return open > 0
}

// verdictConsistent refuses a result verdict that contradicts the open genuine
// findings within the submitter's responsibility after this submission: a
// review counts every genuine finding, a work Pass the findings of work Passes.
func (e *engine) verdictConsistent(m *member, env envelope, raised []Finding, closing map[string]closure) error {
	review := m.p.spec.Review
	open := 0
	counts := func(f *Finding) bool {
		return f.Blocking && e.within(f.Owner, m.Path) && (review || !f.Review && f.RaisedBy != MarkitectFinding)
	}
	for _, f := range e.findings {
		if !f.Closed && closing[f.ID].reason == "" && counts(f) {
			open++
		}
	}
	for i := range raised {
		if counts(&raised[i]) {
			open++
		}
	}
	hasOpenCoverage := slices.ContainsFunc(env.Obligations, func(c coverage) bool { return c.Status == coverOpen })
	switch {
	case env.Verdict == Satisfied && open > 0:
		return fmt.Errorf("verdict satisfied, but %d genuine finding(s) within your responsibility stay open; close those you raised with a reason or report needs_correction", open)
	case env.Verdict == Satisfied && hasOpenCoverage:
		return fmt.Errorf("verdict satisfied, but an own obligation is open; complete it or report needs_correction with a finding")
	case env.Verdict == NeedsCorrection && open == 0:
		return fmt.Errorf("verdict needs_correction, but no genuine finding within your responsibility is open; raise the finding that names the required correction")
	}
	return nil
}

// decide applies a parent's semantic decision on a checked proposal of one of
// its direct children.
func (e *engine) decide(m *member, ts *turnState, in reportIn) ToolAnswer {
	var pr *proposal
	for _, x := range m.p.proposals {
		if x.id == in.Proposal {
			pr = x
		}
	}
	switch {
	case pr == nil:
		return refuse("%q is no proposal of this Pass.", in.Proposal)
	case pr.member.Parent != m.Path:
		return refuse("%s is not a proposal of one of your children.", pr.id)
	case pr.status != proposalChecked || pr.member.pending != pr:
		return refuse("%s is %s and awaits no decision.", pr.id, pr.status)
	}
	ts.decided = true
	if in.Kind == kindReturn {
		reason, err := toolText("reason", in.Reason)
		if err != nil {
			return refuse("%v", err)
		}
		e.returnLocked(pr, m.Path, reason)
		return ToolAnswer{Text: fmt.Sprintf("Returned %s to %s.", pr.id, pr.member.Path)}
	}
	e.acceptLocked(pr, m.Path)
	if pr.status != proposalAccepted {
		return ToolAnswer{Text: fmt.Sprintf("%s could not be accepted: it no longer passes the File check against the accepted stand and went back to %s.", pr.id, pr.member.Path)}
	}
	return ToolAnswer{Text: fmt.Sprintf("Accepted %s of %s.", pr.id, pr.member.Path)}
}

func (e *engine) reopen(m *member, ts *turnState, in reportIn) ToolAnswer {
	if m.stage != stageIntegrate {
		return refuse("You reopen a child's accepted result while you integrate, before you submit your own result.")
	}
	if !slices.Contains(m.Children, in.Child) {
		return refuse("%q is not one of your children.", in.Child)
	}
	child := m.p.byPath[in.Child]
	if child.stage != stageDone || child.result == nil {
		return refuse("%s has no accepted result in this Pass.", in.Child)
	}
	reason, err := toolText("reason", in.Reason)
	if err != nil {
		return refuse("%v", err)
	}
	ts.decided = true
	e.reopenLocked(m, child, reason)
	return ToolAnswer{Text: fmt.Sprintf("Reopened the result of %s; you wait for its new result.", in.Child)}
}

func (e *engine) revise(m *member, ts *turnState, in reportIn) ToolAnswer {
	if (m.stage != stageDelegated && m.stage != stageIntegrate) || m.basis == nil {
		return refuse("You revise an accepted basis while your children work on it or before you submit your result.")
	}
	reason, err := toolText("reason", in.Reason)
	if err != nil {
		return refuse("%v", err)
	}
	ts.revised = true
	e.reviseLocked(m, reason)
	return ToolAnswer{Text: "Your basis is revised: its consumers wait, and your area is writable again from your next turn."}
}

func (e *engine) problem(m *member, ts *turnState, in reportIn) ToolAnswer {
	if !slices.Contains(problemCategories, in.Category) {
		return refuse("category must be one of %s.", strings.Join(problemCategories, ", "))
	}
	text, err := toolText("text", in.Text)
	if err != nil {
		return refuse("%v", err)
	}
	if in.Abort && (m.Parent != "" || in.Category != "model") {
		return refuse("Only Root ends a run, and only for a model that is erroneous or contradictory (category model).")
	}
	e.addProblemLocked(Problem{Manager: m.Path, Category: in.Category, Text: text, Abort: in.Abort})
	if in.Abort {
		e.reject(ModelErrorReason + text)
		return ToolAnswer{Text: "Recorded. The run ends now without publishing anything."}
	}
	return ToolAnswer{Text: "Problem recorded. Escalate it with the tool escalate when your parent must decide."}
}

// takeEffectsLocked records the findings, closures and disputes of an accepted
// proposal. A proposal that is returned, fails its check, is withdrawn or
// invalidated before its acceptance changes no finding.
func (e *engine) takeEffectsLocked(pr *proposal) {
	m, p := pr.member, pr.member.p
	for _, f := range pr.raised {
		pr.env.Findings = append(pr.env.Findings, e.addFindingLocked(f).ID)
	}
	for _, id := range slices.Sorted(mapKeys(pr.closing)) {
		f, c := e.findingLocked(id), pr.closing[id]
		if f == nil || f.Closed {
			continue
		}
		f.Closed, f.ClosedPass, f.ClosedReason, f.ClosedAs, f.ClosedEvidence = true, p.id, c.reason, c.as, c.evidence
		pr.env.Closed = append(pr.env.Closed, id)
		_ = e.record(Event{Kind: "finding.closed", Pass: p.id, Manager: m.Path, Data: *f})
	}
	for _, id := range slices.Sorted(mapKeys(pr.disputes)) {
		f := e.findingLocked(id)
		if f == nil || f.Closed {
			continue
		}
		f.Disputes = append(f.Disputes, pr.disputes[id])
		pr.env.Disputed = append(pr.env.Disputed, id)
		_ = e.record(Event{Kind: "finding.disputed", Pass: p.id, Manager: m.Path, Data: *f})
	}
	pr.raised, pr.closing, pr.disputes = nil, nil, nil
}

// checkSubmission is the technical gate of a submission in a work Pass with
// writes: after the submitter's writers stopped, its area is observed, composed
// with the accepted stand and File checked. It runs outside the lock.
func (e *engine) checkSubmission(pr *proposal) {
	defer e.wg.Done()
	m := pr.member
	e.mu.Lock()
	accepted := m.p.accepted
	e.mu.Unlock()
	var findings []Finding
	var composite snapshot.Tree
	h, err := e.peek()
	if err != nil {
		e.halt(fmt.Sprintf("the area of %s could not be observed for its submission %s: %v", m.Path, pr.id, err))
	} else {
		var unsupported []string
		for _, name := range h.Unsupported {
			if p, perr := snapshot.NewPath(name); perr != nil || e.cfg.OwnerOf(p) == m.Path {
				unsupported = append(unsupported, name)
			}
		}
		if len(unsupported) > 0 {
			findings = append(findings, Finding{Owner: m.Path, Obligation: "every entry of your area is a regular file or a directory", Observed: fmt.Sprintf("%d unsupported entries (symbolic links, devices or names that are not valid paths): %s", len(unsupported), listNames(unsupported))})
		}
		if composite, err = overlay(accepted, h.Tree, m.Path, e.cfg.OwnerOf); err != nil {
			findings = append(findings, Finding{Owner: m.Path, Obligation: "your area composes with the accepted stand", Observed: err.Error()})
		} else if len(findings) == 0 {
			findings = ownFindings(e.cfg.FileCheck(composite), m.Path)
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	m.checking = false
	m.p.checking--
	if e.ctx.Err() != nil || pr.status != proposalSubmitted || m.pending != pr {
		e.recordStateLocked(m, "the check ended")
		e.cond.Broadcast()
		return
	}
	if len(findings) > 0 {
		e.technicalReturnLocked(pr, findings)
		return
	}
	pr.composite = composite
	e.checkedLocked(pr)
}

// ownFindings keeps the findings a submitter must resolve: those addressed to
// it. Every other owner's state in the composition is its accepted version or
// work that owner still has to submit; the whole stand is checked at the Pass's
// exit.
func ownFindings(findings []Finding, owner string) []Finding {
	return slices.DeleteFunc(findings, func(f Finding) bool { return f.Owner != owner })
}

// peek observes the shared work volume, retrying a briefly uncertain
// observation.
func (e *engine) peek() (WorkspaceHarvest, error) {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		h, err := e.cfg.Workspaces.Peek(e.ctx, HarvestRequest{ID: e.workID, MaxEntries: e.cfg.MaxEntries, MaxBytes: e.cfg.MaxBytes})
		if err == nil && !h.Uncertain {
			return h, nil
		}
		last = err
		if err == nil {
			last = fmt.Errorf("the observation is uncertain: %s", h.Reason)
		}
		select {
		case <-e.ctx.Done():
			return WorkspaceHarvest{}, e.ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 200 * time.Millisecond):
		}
	}
	return WorkspaceHarvest{}, last
}

// checkedLocked records a passed technical check and brings the submission to
// its decision: Root's to the Orchestrator, which validated it against the
// declared Pass requirements, every other one to its direct parent.
func (e *engine) checkedLocked(pr *proposal) {
	m := pr.member
	pr.status = proposalChecked
	_ = e.record(Event{Kind: "proposal.checked", Pass: m.p.id, Manager: m.Path, Data: map[string]any{"proposal": pr.id, "passed": true}})
	if m.Parent == "" {
		e.acceptLocked(pr, "the Orchestrator")
		return
	}
	parent := m.p.byPath[m.Parent]
	e.deliverLocked(parent, item{text: e.renderProposalLocked(pr)})
	e.recordStateLocked(m, "submitted for its parent's decision")
	e.scheduleLocked()
}
