package cockpit

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Project folds the first n records of a run into the view at that boundary.
// It reads nothing after record n: a later answer, verdict or outcome cannot
// appear in an earlier view. The report is shown only at the final boundary.
func Project(id string, records []Record, n int, src Source, report *Report) View {
	if n < 0 || n > len(records) {
		n = len(records)
	}
	p := newProjector(id)
	for _, r := range records[:n] {
		p.apply(r)
	}
	v := p.view()
	v.Boundary = Boundary{Seq: n, Total: len(records), Final: n == len(records)}
	if n > 0 {
		v.Boundary.At = records[n-1].At
	}
	v.Source = src
	if v.Boundary.Final {
		v.Status.Report = report
	}
	classifyWaits(&v)
	v.Account = account(&v)
	v.Gaps = p.gaps(v, src)
	return v
}

type projector struct {
	head      RunHead
	order     []string
	managers  map[string]*ManagerView
	passes    []*PassView
	byPass    map[string]*PassView
	current   string
	turns     []*TurnView
	byTurn    map[string]*TurnView
	proposals []*ProposalView
	byProp    map[string]*ProposalView
	findings  []*FindingView
	byFinding map[string]*FindingView
	questions []*QuestionView
	byQ       map[string]*QuestionView
	problems  []ProblemView
	spans     []Span
	open      map[string]int // pass/manager -> index of its open span
	said      map[string]*Said
	status    Status
	invalid   []int
	undecoded []int
	unknown   map[string]int
	mismatch  []string
	exchanges bool
	deps      map[string][]Dependency // proposal -> its plan's declared dependencies
	plans     map[string][]Dependency // Manager -> dependencies of its accepted plan
}

func newProjector(id string) *projector {
	return &projector{
		head: RunHead{ID: id, Managers: []string{}}, managers: map[string]*ManagerView{}, byPass: map[string]*PassView{},
		byTurn: map[string]*TurnView{}, byProp: map[string]*ProposalView{}, byFinding: map[string]*FindingView{},
		byQ: map[string]*QuestionView{}, open: map[string]int{}, said: map[string]*Said{}, unknown: map[string]int{},
		deps: map[string][]Dependency{}, plans: map[string][]Dependency{},
	}
}

// decode reads a record's data as T. A record whose data does not decode is
// remembered as a limitation; its zero value is used.
func decode[T any](p *projector, r Record) T {
	var v T
	if len(r.Data) > 0 && json.Unmarshal(r.Data, &v) != nil {
		p.undecoded = append(p.undecoded, r.Seq)
	}
	return v
}

func (p *projector) apply(r Record) {
	if r.Invalid != "" {
		p.invalid = append(p.invalid, r.Seq)
		return
	}
	switch r.Kind {
	case "run.start":
		d := decode[struct {
			Run        string    `json:"run"`
			Managers   []string  `json:"managers"`
			Start      string    `json:"start"`
			AgentTurns int       `json:"agentTurns"`
			Passes     int       `json:"passes"`
			Deadline   time.Time `json:"deadline"`
		}](p, r)
		p.head.Started, p.head.Deadline, p.head.AgentTurns, p.head.Passes, p.head.Start = r.At, d.Deadline, d.AgentTurns, d.Passes, d.Start
		for _, m := range d.Managers {
			p.manager(m)
		}
	case "pass.start":
		d := decode[struct {
			Purpose string `json:"purpose"`
			Review  bool   `json:"review"`
			Writes  bool   `json:"writes"`
			Tree    string `json:"tree"`
		}](p, r)
		pv := p.pass(r.Pass)
		pv.Purpose, pv.Review, pv.Writes, pv.StartTree, pv.StartSeq, pv.Started = d.Purpose, d.Review, d.Writes, d.Tree, r.Seq, r.At
		if !d.Review {
			for _, f := range p.findings {
				if f.Blocking && !f.Closed {
					pv.Correction = true
					pv.Addresses = append(pv.Addresses, f.ID)
				}
			}
		}
		p.current = r.Pass
		for _, m := range p.managers {
			*m = ManagerView{Path: m.Path, Label: m.Label, Parent: m.Parent, Children: m.Children, Depth: m.Depth, Disabled: m.Disabled}
		}
	case "manager.state_changed":
		d := decode[struct {
			From       string `json:"from"`
			To         string `json:"to"`
			Stage      string `json:"stage"`
			Why        string `json:"why"`
			WaitingFor string `json:"waitingFor"`
		}](p, r)
		m := p.manager(r.Manager)
		if r.Pass == p.current {
			m.State, m.Stage, m.Since, m.Why, m.Wait = d.To, d.Stage, r.Seq, d.Why, nil
			if d.To == string(run.Waiting) {
				m.Wait = &Wait{Text: d.WaitingFor, Since: r.Seq, SinceAt: r.At}
			}
		}
		p.closeSpan(r.Pass, r.Manager, r)
		p.open[r.Pass+"\x00"+r.Manager] = len(p.spans)
		p.spans = append(p.spans, Span{Manager: r.Manager, Pass: r.Pass, State: d.To, Wait: d.WaitingFor, FromSeq: r.Seq, From: r.At})
	case "manager.release":
		d := decode[struct {
			Parent string `json:"parent"`
		}](p, r)
		m := p.manager(r.Manager)
		if r.Pass == p.current {
			m.Released, m.ReleasedSeq = true, r.Seq
		}
		if d.Parent != m.Parent {
			p.mismatch = append(p.mismatch, fmt.Sprintf("%s released under %q, its path names %q", r.Manager, d.Parent, m.Parent))
		}
	case "registry.view":
		d := decode[struct {
			Registrations []json.RawMessage `json:"registrations"`
		}](p, r)
		p.pass(r.Pass).Registry = len(d.Registrations)
	case "manager.unblocked", "report.progress", "check.requested", "workspace.materialize", "workspace.close", "home.remove", "owner.error", "extract.phase":
		// Listed as entries; they change nothing the view projects.
	case "report.wait":
		d := decode[struct {
			For    string `json:"for"`
			Reason string `json:"reason"`
		}](p, r)
		p.said[r.Pass+"\x00"+r.Manager] = &Said{Seq: r.Seq, For: d.For, Reason: d.Reason}
	case "turn.start":
		d := decode[struct {
			Role     string `json:"role"`
			Kind     string `json:"kind"`
			Profile  string `json:"profile"`
			ReadOnly bool   `json:"readOnly"`
			Resume   string `json:"resume"`
		}](p, r)
		t := &TurnView{ID: r.Turn, Short: shortTurn(r.Turn), Pass: r.Pass, Manager: r.Manager, Role: d.Role, Kind: d.Kind, ReadOnly: d.ReadOnly, Profile: d.Profile, Resume: d.Resume != "", StartSeq: r.Seq, Started: r.At}
		p.turns = append(p.turns, t)
		p.byTurn[r.Turn] = t
		if m := p.manager(r.Manager); r.Pass == p.current && !isHelperRole(d.Role) {
			m.Turn = t
		}
	case "turn.end":
		d := decode[struct {
			Stopped bool       `json:"stopped"`
			Reason  string     `json:"reason"`
			Reports []string   `json:"reports"`
			Usage   *run.Usage `json:"usage"`
		}](p, r)
		if t := p.byTurn[r.Turn]; t != nil {
			stopped := d.Stopped
			t.EndSeq, t.Ended, t.Stopped, t.Reason, t.Reports, t.Usage = r.Seq, r.At, &stopped, d.Reason, d.Reports, d.Usage
		}
		if m := p.managers[r.Manager]; m != nil && m.Turn != nil && m.Turn.ID == r.Turn {
			m.Turn = nil
		}
	case "helper.delegated", "helper.start", "helper.end":
		d := decode[struct {
			Helper string `json:"helper"`
			Role   string `json:"role"`
		}](p, r)
		m := p.manager(r.Manager)
		switch r.Kind {
		case "helper.start":
			m.Helpers = append(m.Helpers, Helper{ID: d.Helper, Role: d.Role, Turn: r.Turn})
		case "helper.end":
			for i, h := range m.Helpers {
				if h.ID == d.Helper {
					m.Helpers = append(m.Helpers[:i:i], m.Helpers[i+1:]...)
					break
				}
			}
		}
	case "proposal.submitted":
		d := decode[struct {
			Proposal string `json:"proposal"`
			Kind     string `json:"kind"`
			Purpose  string `json:"purpose"`
			Envelope struct {
				Summary  string   `json:"summary"`
				Verdict  string   `json:"verdict"`
				Children []string `json:"children"`
				Plan     *struct {
					Dependencies []Dependency `json:"dependencies"`
				} `json:"plan"`
			} `json:"envelope"`
			Findings []json.RawMessage `json:"findings"`
			Closed   []string          `json:"closed"`
			Disputed []string          `json:"disputed"`
		}](p, r)
		pr := &ProposalView{ID: d.Proposal, Pass: r.Pass, Manager: r.Manager, Kind: d.Kind, Purpose: d.Purpose, Turn: r.Turn, Seq: r.Seq, At: r.At,
			Summary: clip(d.Envelope.Summary, 320), Verdict: d.Envelope.Verdict, Status: "submitted", Raised: len(d.Findings), Closes: d.Closed, Disputes: d.Disputed, Children: d.Envelope.Children}
		p.proposals = append(p.proposals, pr)
		p.byProp[pr.ID] = pr
		if d.Envelope.Plan != nil {
			p.deps[pr.ID] = d.Envelope.Plan.Dependencies
		}
		if r.Pass == p.current {
			p.manager(r.Manager).Pending = pr.ID
		}
	case "proposal.checked":
		d := decode[struct {
			Proposal string `json:"proposal"`
			Passed   bool   `json:"passed"`
		}](p, r)
		if pr := p.byProp[d.Proposal]; pr != nil {
			passed := d.Passed
			pr.Checked = &passed
			if !passed {
				pr.Status = "check failed"
				p.clearPending(pr)
			}
		}
	case "basis.accepted", "result.accepted", "basis.returned", "result.returned", "result.reopened", "basis.invalidated", "proposal.invalidated", "proposal.withdrawn":
		p.decide(r)
	case "finding.recorded", "finding.closed", "finding.disputed":
		f := decode[run.Finding](p, r)
		fv := p.byFinding[f.ID]
		if fv == nil {
			fv = &FindingView{Seq: r.Seq}
			p.findings = append(p.findings, fv)
			p.byFinding[f.ID] = fv
		}
		fv.Finding = f
		if f.Closed && fv.ClosedSeq == 0 {
			fv.ClosedSeq = r.Seq
		}
	case "question.asked", "question.answered", "question.lapsed":
		q := decode[run.Question](p, r)
		qv := p.byQ[q.ID]
		if qv == nil {
			qv = &QuestionView{Seq: r.Seq, Status: "open"}
			p.questions = append(p.questions, qv)
			p.byQ[q.ID] = qv
		}
		qv.Question = q
		p.exchanges = true
		switch r.Kind {
		case "question.answered":
			qv.Status, qv.AnsweredSeq, qv.AnsweredBy = "answered", r.Seq, r.Manager
		case "question.lapsed":
			qv.Status, qv.LapsedSeq = "lapsed", r.Seq
		}
	case "message":
		p.exchanges = true
	case "problem.raised":
		p.problems = append(p.problems, ProblemView{Problem: decode[run.Problem](p, r), Seq: r.Seq})
	case "manager.disabled":
		p.manager(r.Manager).Disabled = true
	case "candidate":
		d := decode[struct {
			Tree string `json:"tree"`
		}](p, r)
		p.pass(r.Pass).Candidate = d.Tree
	case "check":
		c := decode[run.CheckResult](p, r)
		pv := p.pass(r.Pass)
		pv.Checks = append(pv.Checks, CheckView{Seq: r.Seq, Name: c.Name, Passed: c.Passed, ExitCode: c.ExitCode, Reason: c.Reason})
	case "gate":
		d := decode[struct {
			At     string `json:"at"`
			Result struct {
				ModelCheck *int   `json:"modelCheck"`
				FileCheck  *int   `json:"fileCheck"`
				Tree       string `json:"tree"`
			} `json:"result"`
		}](p, r)
		pv := p.pass(r.Pass)
		pv.Gates = append(pv.Gates, GateView{Seq: r.Seq, At: d.At, ModelCheck: d.Result.ModelCheck, FileCheck: d.Result.FileCheck, Tree: d.Result.Tree})
	case "pass.sealed":
		d := decode[struct {
			Verdict  string              `json:"verdict"`
			Tree     string              `json:"tree"`
			Managers []run.ManagerRecord `json:"managers"`
		}](p, r)
		if pv := p.byPass[r.Pass]; pv != nil {
			pv.Sealed, pv.SealedSeq, pv.SealedAt, pv.Verdict, pv.Tree, pv.Managers = true, r.Seq, r.At, d.Verdict, d.Tree, d.Managers
			for i := range pv.Managers {
				pv.Managers[i].Summary = clip(pv.Managers[i].Summary, 320)
			}
			// The Orchestrator enforces the dependencies of the plans accepted
			// when the Plan Pass sealed: each Manager's accepted result.
			if pv.Purpose == string(run.Plan) {
				p.plans = map[string][]Dependency{}
				for _, m := range pv.Managers {
					p.plans[m.Manager] = p.deps[m.Result]
				}
			}
		}
		for _, m := range p.order {
			p.closeSpan(r.Pass, m, r)
		}
	case "outcome":
		d := decode[struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		}](p, r)
		p.status.Outcome, p.status.Reason, p.status.OutcomeSeq = d.Status, d.Reason, r.Seq
		for key := range p.open {
			pass, m, _ := strings.Cut(key, "\x00")
			p.closeSpan(pass, m, r)
		}
	default:
		p.unknown[r.Kind]++
	}
}

// pass returns the Pass with this identity. A review Pass's candidate, gate
// and checks are recorded before its pass.start; they open the Pass's entry,
// which pass.start completes.
func (p *projector) pass(id string) *PassView {
	if pv := p.byPass[id]; pv != nil {
		return pv
	}
	pv := &PassView{ID: id}
	p.passes = append(p.passes, pv)
	p.byPass[id] = pv
	return pv
}

// decide applies a decision on, or an invalidation of, a submitted proposal.
func (p *projector) decide(r Record) {
	d := decode[struct {
		Proposal string `json:"proposal"`
		By       string `json:"by"`
		Reason   string `json:"reason"`
		Why      string `json:"why"`
		Stand    string `json:"stand"`
	}](p, r)
	pr := p.byProp[d.Proposal]
	if pr == nil {
		return
	}
	reason := d.Reason
	if reason == "" {
		reason = d.Why
	}
	pr.DecSeq, pr.Reason = r.Seq, clip(reason, 320)
	if d.By != "" {
		pr.By = d.By
	}
	m := p.managers[pr.Manager]
	switch r.Kind {
	case "basis.accepted", "result.accepted":
		pr.Status, pr.Stand = "accepted", d.Stand
		p.clearPending(pr)
		if m != nil && pr.Pass == p.current {
			if r.Kind == "basis.accepted" {
				m.Basis = pr.ID
			} else {
				m.Result = pr.ID
			}
		}
	case "basis.returned", "result.returned":
		pr.Status = "returned"
		p.clearPending(pr)
	case "result.reopened":
		pr.Status = "reopened"
		if m != nil && m.Result == pr.ID {
			m.Result = ""
		}
	case "basis.invalidated":
		pr.Status = "invalidated"
		if m != nil && m.Basis == pr.ID {
			m.Basis = ""
		}
	case "proposal.invalidated", "proposal.withdrawn":
		pr.Status = strings.TrimPrefix(r.Kind, "proposal.")
		p.clearPending(pr)
	}
}

func (p *projector) clearPending(pr *ProposalView) {
	if m := p.managers[pr.Manager]; m != nil && m.Pending == pr.ID {
		m.Pending = ""
	}
}

func (p *projector) closeSpan(pass, manager string, r Record) {
	key := pass + "\x00" + manager
	if i, ok := p.open[key]; ok {
		p.spans[i].ToSeq, p.spans[i].To = r.Seq, r.At
		delete(p.open, key)
	}
}

// manager returns the Manager with this path, adding it in its place of the
// organization: its parent is its path without the last element, Root's for a
// top-level Package, as the organization derives it from the model.
func (p *projector) manager(path string) *ManagerView {
	if m := p.managers[path]; m != nil {
		return m
	}
	m := &ManagerView{Path: path, Label: labelOf(path), Parent: parentOf(path), Children: []string{}}
	if m.Parent != "" {
		parent := p.manager(m.Parent)
		parent.Children = append(parent.Children, path)
		m.Depth = parent.Depth + 1
	}
	p.managers[path] = m
	p.order = append(p.order, path)
	p.head.Managers = append(p.head.Managers, path)
	return m
}

func labelOf(path string) string {
	if path == "." {
		return "Root"
	}
	return path
}

func isHelperRole(role string) bool {
	return role != "manager" && role != "reviewer"
}

// shortTurn is the last element of a turn identity, its per-Manager counter
// such as m3 or r2.
func shortTurn(id string) string {
	if i := strings.LastIndexByte(id, '-'); i >= 0 {
		return id[i+1:]
	}
	return id
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func (p *projector) view() View {
	v := View{Run: p.head, Status: p.status, Current: p.current, Managers: []ManagerView{}, Passes: []PassView{}, Turns: []TurnView{},
		Spans: p.spans, Proposals: []ProposalView{}, Findings: []FindingView{}, Questions: []QuestionView{}, Problems: p.problems}
	if v.Spans == nil {
		v.Spans = []Span{}
	}
	if v.Problems == nil {
		v.Problems = []ProblemView{}
	}
	for _, pv := range p.passes {
		v.Passes = append(v.Passes, *pv)
	}
	for _, t := range p.turns {
		v.Turns = append(v.Turns, *t)
	}
	for _, pr := range p.proposals {
		v.Proposals = append(v.Proposals, *pr)
	}
	for _, f := range p.findings {
		v.Findings = append(v.Findings, *f)
	}
	for _, q := range p.questions {
		v.Questions = append(v.Questions, *q)
	}
	open := map[string][]string{}
	for _, f := range p.findings {
		if f.Blocking && !f.Closed {
			open[f.Owner] = append(open[f.Owner], f.ID)
		}
	}
	purpose := ""
	if pv := p.byPass[p.current]; pv != nil && pv.Purpose != string(run.Plan) {
		purpose = pv.Purpose
	}
	for _, path := range p.preorder() {
		m := *p.managers[path]
		for _, d := range p.plans[path] {
			if d.Pass != purpose {
				continue
			}
			if owner := p.managers[d.Owner]; owner != nil {
				d.Accepted = d.Stage == "basis" && owner.Basis != "" || d.Stage == "result" && owner.Result != ""
			}
			m.Declared = append(m.Declared, d)
		}
		m.Checking = m.State == string(run.Working) && m.Turn == nil
		m.Findings = open[path]
		if m.Wait != nil {
			w := *m.Wait
			// The Orchestrator waits for exactly what the Manager's own
			// report.wait named; its reason belongs to that wait only.
			if s := p.said[p.current+"\x00"+path]; s != nil && s.For == w.Text && s.Seq < w.Since {
				w.Agent = s
			}
			m.Wait = &w
		}
		v.Managers = append(v.Managers, m)
	}
	return v
}

// preorder lists the Managers parents first, children in recorded order.
func (p *projector) preorder() []string {
	var out []string
	var walk func(string)
	walk = func(path string) {
		out = append(out, path)
		for _, c := range p.managers[path].Children {
			walk(c)
		}
	}
	if p.managers["."] != nil {
		walk(".")
	}
	if len(out) < len(p.order) {
		seen := map[string]bool{}
		for _, m := range out {
			seen[m] = true
		}
		for _, m := range p.order {
			if !seen[m] {
				out = append(out, m)
			}
		}
	}
	return out
}

// gaps lists what the evidence does not establish for this view.
func (p *projector) gaps(v View, src Source) []Gap {
	var out []Gap
	if src.Error != "" {
		out = append(out, Gap{Code: "source.error", Text: "The last read of the evidence failed: " + src.Error})
	}
	if src.Replaced > 0 {
		out = append(out, Gap{Code: "source.replaced", Text: fmt.Sprintf("The events file was replaced or truncated %d time(s) while observed; the history was read again from its start.", src.Replaced)})
	}
	if v.Boundary.Final && src.TailBytes > 0 {
		out = append(out, Gap{Code: "source.tail", Text: fmt.Sprintf("%d bytes after the last complete record are not a complete record yet and are not shown.", src.TailBytes)})
	}
	if len(p.invalid) > 0 {
		out = append(out, Gap{Code: "record.invalid", Seq: p.invalid[0], Text: fmt.Sprintf("%d record(s) could not be read, first at line %d; they keep their place but show nothing.", len(p.invalid), p.invalid[0])})
	}
	if len(p.undecoded) > 0 {
		out = append(out, Gap{Code: "record.data", Seq: p.undecoded[0], Text: fmt.Sprintf("The data of %d record(s) has an unexpected shape, first at line %d; their fields are shown as unknown.", len(p.undecoded), p.undecoded[0])})
	}
	if len(p.unknown) > 0 {
		kinds := make([]string, 0, len(p.unknown))
		for k := range p.unknown {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		out = append(out, Gap{Code: "record.kind", Text: "Records of kinds the cockpit does not interpret are listed but not projected: " + strings.Join(kinds, ", ") + "."})
	}
	for _, m := range p.mismatch {
		out = append(out, Gap{Code: "organization.parent", Text: "The recorded release disagrees with the organization derived from the Package paths: " + m + "."})
	}
	for _, m := range v.Managers {
		if m.Wait != nil && m.Wait.Outdated != "" {
			out = append(out, Gap{Code: "wait.unrecorded", Text: "Before a Manager's productive work starts, the Orchestrator does not record a changed wait that leaves it Waiting: after its release or after one declared dependency is met. Such waits are marked outdated; their current reason is not recorded."})
			break
		}
	}
	if p.exchanges {
		out = append(out, Gap{Code: "exchange.delivery", Text: "Sends, answers and decisions are recorded; delivery into a Manager's turn and its reading are not, so no exchange is shown as received or read."})
	}
	if v.Boundary.Final {
		switch {
		case v.Status.Outcome == "":
			out = append(out, Gap{Code: "run.no_outcome", Text: "The record ends without an outcome: the run may still be running or may have been interrupted. The record alone does not say which."})
		case !src.Report:
			out = append(out, Gap{Code: "run.no_report", Text: "The outcome is recorded but report.json is absent: the publication and the Run outcome are not recorded."})
		}
		if r := v.Status.Report; r != nil {
			if r.Derived {
				out = append(out, Gap{Code: "report.derived", Text: "This report has no Run outcome field; the outcome shown is mapped from its command status."})
			}
			if r.Publication != nil {
				out = append(out, Gap{Code: "publication.candidate", Text: "The publication receipt names no candidate digest. Apply publishes only the candidate of its clean Verify; the receipt itself records the read-back state of each path."})
			}
		}
	}
	out = append(out, Gap{Code: "candidate.files", Text: "Candidates are recorded by tree digest; their file differences are not part of the run evidence. The published paths appear in the report."})
	return out
}
