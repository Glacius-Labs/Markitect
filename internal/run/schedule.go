package run

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// idleLimit is how many starts in a row may leave nothing behind (no
// submission, review or message) before the Run stops as making no progress.
const idleLimit = 2

// schedule runs the agents of one Run. One goroutine decides; agents run in
// their own goroutines and report how they ended, and host tool calls only
// record messages and wake the decider. Nobody waits for a message: an
// answer starts its recipient again.
type schedule struct {
	e      *Engine
	req    Request
	plan   *runremote.Plan
	cur    *project.Project
	imp    analysis.Impact
	remote Remote
	creds  map[string]map[agentport.Role]credential

	mu      sync.Mutex
	nodes   map[string]*node
	order   []*node // children before parents
	tips    map[string]string
	starts  int
	running int
	ended   chan ended
	wake    chan struct{}
	stopped bool
	state   string
	why     string
	cancel  context.CancelFunc

	messages  []agentport.Message
	notes     []agentport.Note
	decisions []OwnerDecision
	records   []AgentStart
	leftovers []string
}

// node is one working Manager.
type node struct {
	id       string // the Package path; Root's is ""
	name     string // as trailers and messages write it; Root is "."
	parent   *node
	children []*node
	items    []agentport.Item
	branch   string

	running    bool
	rounds     map[agentport.Role]int
	idle       int
	inbox      []agentport.Message
	sent       bool // the running start sent a message
	waitFor    map[string]bool
	unanswered map[string]agentport.Message
	reminded   map[string]bool
	// told is the children's approvals the maker last got to merge.
	told string

	// released: the Manager may start; from is the commit its work starts
	// from, the base or a state of its parent's branch (concept 2.0 §6.2).
	released bool
	from     string

	tip                string // the branch tip after the last start
	submission         string // the newest submission at the tip, awaiting review
	approvedSubmission string
	approval           string // the checker's approving commit
	approved           bool
	verdicts           []runremote.Line // of the newest submission
	lastSubmission     string
	approvedVerdicts   []runremote.Line
	checks             []runremote.Line
	review             string // the checker of the approval
	done               bool   // verify: verdicts pushed
}

type ended struct {
	n     *node
	role  agentport.Role
	end   agentport.End
	began time.Time
	round int
}

func newSchedule(e *Engine, req Request, plan *runremote.Plan, cur *project.Project, imp analysis.Impact, remote Remote, creds map[string]map[agentport.Role]credential) *schedule {
	s := &schedule{e: e, req: req, plan: plan, cur: cur, imp: imp, remote: remote, creds: creds,
		nodes: map[string]*node{}, tips: map[string]string{}, ended: make(chan ended), wake: make(chan struct{}, 1)}
	lists := map[string]analysis.Checklist{}
	for _, c := range imp.Checklists {
		lists[c.Package] = c
	}
	for _, m := range plan.Managers {
		if !m.Working {
			continue
		}
		n := &node{id: m.ID, name: runremote.Name(m.ID), branch: runremote.Branch(m.ID), rounds: map[agentport.Role]int{},
			waitFor: map[string]bool{}, unanswered: map[string]agentport.Message{}, reminded: map[string]bool{}, tip: plan.Base, from: plan.Base}
		n.items = checklistItems(lists[m.ID])
		s.nodes[m.ID] = n
	}
	// Deeper Packages first, so children come before their parents.
	ids := sortedKeys(s.nodes)
	for depth := maxDepth(ids); depth >= 0; depth-- {
		for _, id := range ids {
			if packageDepth(id) == depth {
				s.order = append(s.order, s.nodes[id])
			}
		}
	}
	for _, n := range s.order {
		if n.id == model.RootPackage {
			continue
		}
		p := s.nodes[model.ParentOf(n.id)]
		n.parent = p
		p.children = append([]*node{n}, p.children...)
	}
	return s
}

func packageDepth(id string) int {
	if id == model.RootPackage {
		return 0
	}
	return strings.Count(id, "/") + 1
}

func maxDepth(ids []string) int {
	d := 0
	for _, id := range ids {
		d = max(d, packageDepth(id))
	}
	return d
}

// checklistItems pairs each item key, as the plan names it, with what the
// item is about; runremote.Items names them in this order.
func checklistItems(c analysis.Checklist) []agentport.Item {
	keys := runremote.Items(c)
	var items []agentport.Item
	next := func(part, def, why string) {
		items = append(items, agentport.Item{Key: keys[len(items)], Part: part, Definition: def, Why: why})
	}
	for _, it := range c.Own {
		next(agentport.Own, it.Definition, strings.Join(it.Reasons, "; "))
	}
	for _, it := range c.Inherited {
		next(agentport.Inherited, it.Definition, strings.Join(it.Reasons, "; "))
	}
	for _, f := range c.Findings {
		next(agentport.Finding, f.Definition, "Observed: "+f.Observed+" Expected: "+f.Expected)
	}
	for _, st := range c.Stale {
		next(agentport.Stale, st.Target, fmt.Sprintf("%s in %s: %s", st.Code, st.PageFile, st.File))
	}
	return items
}

// run starts agents until Root's work is approved (apply), every checker has
// pushed its verdicts (verify), or the Run stops.
func (s *schedule) run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	defer cancel()
	for {
		s.mu.Lock()
		if !s.stopped {
			if err := ctx.Err(); err != nil {
				s.stopLocked(StateIncomplete, fmt.Sprintf("the Run's deadline passed or it was cancelled: %v", context.Cause(ctx)))
			} else {
				s.step(ctx)
			}
		}
		if s.running == 0 && (s.stopped || s.finishedLocked()) {
			s.mu.Unlock()
			return
		}
		if s.running == 0 && !s.stopped {
			s.stopLocked(StateIncomplete, "no Manager can go on: "+s.waitingLocked())
			s.mu.Unlock()
			return
		}
		// Once stopped, only the running agents' ends are awaited.
		done := ctx.Done()
		if s.stopped {
			done = nil
		}
		s.mu.Unlock()
		select {
		case ev := <-s.ended:
			s.onEnd(ctx, ev)
		case <-s.wake:
		case <-done:
		}
	}
}

func (s *schedule) poke() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// stopLocked ends the Run: no agent starts any more and running ones are
// stopped. The first reason stays.
func (s *schedule) stopLocked(state, why string) {
	if s.stopped {
		return
	}
	s.stopped, s.state, s.why = true, state, why
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *schedule) finishedLocked() bool {
	if s.req.Mode == agentport.Verify {
		for _, n := range s.order {
			if !n.done {
				return false
			}
		}
		return true
	}
	for _, n := range s.order {
		if !n.approved || len(n.waitFor) > 0 || len(n.inbox) > 0 || len(n.unanswered) > 0 {
			return false
		}
	}
	return true
}

func (s *schedule) waitingLocked() string {
	var parts []string
	for _, n := range s.order {
		switch {
		case s.req.Mode == agentport.Verify && !n.done:
			parts = append(parts, n.name+" has no verdicts")
		case s.req.Mode == agentport.Apply && !n.released:
			parts = append(parts, n.name+" waits for its parent's approved work")
		case s.req.Mode == agentport.Apply && !n.approved:
			parts = append(parts, fmt.Sprintf("%s is not approved (waits for %d answers)", n.name, len(n.waitFor)))
		}
		if len(n.unanswered) > 0 {
			parts = append(parts, fmt.Sprintf("%s did not answer %s", n.name, strings.Join(sortedKeys(n.unanswered), ", ")))
		}
	}
	return strings.Join(parts, "; ")
}

// step starts every agent that can work now.
func (s *schedule) step(ctx context.Context) {
	for _, n := range s.order {
		if s.stopped {
			return
		}
		if n.running {
			continue
		}
		if s.req.Mode == agentport.Verify {
			if !n.done {
				s.startLocked(ctx, n, agentport.Checker, s.verifyTask(n))
			}
			continue
		}
		if !n.released && !s.releaseLocked(ctx, n) {
			continue
		}
		switch {
		case n.submission != "":
			s.startLocked(ctx, n, agentport.Checker, s.reviewTask(n))
		case len(n.inbox) > 0:
			s.startLocked(ctx, n, agentport.Maker, s.makerTask(n))
		case n.rounds[agentport.Maker] == 0 && len(n.waitFor) == 0 && (len(n.items) > 0 || s.childrenApproved(n)):
			// A parent with items of its own submits before its children
			// start; one without waits for their approved work.
			s.startLocked(ctx, n, agentport.Maker, s.makerTask(n))
		case n.rounds[agentport.Maker] > 0 && len(n.children) > 0 && s.childrenApproved(n) && s.approvals(n) != n.told:
			// The children's approved work changed since the maker last
			// saw it: it merges and submits again, and its own approval no
			// longer holds.
			n.told, n.approved = s.approvals(n), false
			s.hostMessage(n, "Your working children have approved work: "+n.told+". Merge it, review it and submit again.")
			s.startLocked(ctx, n, agentport.Maker, s.makerTask(n))
		default:
			var open []string
			for _, id := range sortedKeys(n.unanswered) {
				if !n.reminded[id] {
					n.reminded[id] = true
					open = append(open, fmt.Sprintf("%s from %s", id, n.unanswered[id].From))
				}
			}
			if len(open) > 0 {
				s.hostMessage(n, "Your children wait for your answer to "+strings.Join(open, ", ")+". Answer each one.")
				s.startLocked(ctx, n, agentport.Maker, s.makerTask(n))
			}
		}
	}
}

// releaseLocked lets a Manager start in apply (concept 2.0 §6.2): Root at
// once; any other once its parent may start and, when the parent has items
// of its own, has approved work. Its branch then starts from that approval,
// or from where its parent started, so shared files of the parent exist
// before the child's required checks run.
func (s *schedule) releaseLocked(ctx context.Context, n *node) bool {
	p := n.parent
	if p == nil {
		n.released = true
		return true
	}
	if !p.released && !s.releaseLocked(ctx, p) {
		return false
	}
	from := p.from
	if len(p.items) > 0 {
		if !p.approved {
			return false
		}
		from = p.approval
	}
	if from != s.plan.Base {
		if err := s.remote.StartFrom(ctx, n.branch, from); err != nil {
			s.stopLocked(StateIncomplete, fmt.Sprintf("Manager %s could not start from its parent's state %s: %v", n.name, short(from), err))
			return false
		}
	}
	n.released, n.from, n.tip = true, from, from
	return true
}

// childrenApproved: every working child has approved work that holds its
// whole subtree's, so a parent first approved for its own files does not
// count while its children work.
func (s *schedule) childrenApproved(n *node) bool {
	for _, c := range n.children {
		if !c.approved || c.running || len(c.waitFor) > 0 || len(c.inbox) > 0 || !s.childrenApproved(c) {
			return false
		}
	}
	return true
}

// startLocked starts one agent within the budget.
func (s *schedule) startLocked(ctx context.Context, n *node, role agentport.Role, task string) {
	lim := s.req.Limits
	if s.starts >= lim.Starts {
		s.stopLocked(StateIncomplete, fmt.Sprintf("the budget of %d agent starts is used up", lim.Starts))
		return
	}
	if n.rounds[role] >= lim.Rounds {
		s.stopLocked(StateIncomplete, fmt.Sprintf("Manager %s used its %d %s starts without approved work", n.name, lim.Rounds, role))
		return
	}
	s.starts++
	n.rounds[role]++
	if role == agentport.Maker && len(n.children) > 0 && s.childrenApproved(n) && s.approvals(n) != n.told {
		// The task names the children's approvals, so the maker has them;
		// whatever started it, its own approval no longer holds.
		n.told, n.approved = s.approvals(n), false
	}
	n.running, n.sent = true, false
	inbox := n.inbox
	if role == agentport.Maker {
		n.inbox = nil
	}
	deadline := lim.Deadline
	if d := s.e.now().Add(lim.Start); lim.Start > 0 && d.Before(deadline) {
		deadline = d
	}
	st := agentport.Start{Run: s.plan.Run, Mode: s.req.Mode, Role: role, Round: n.rounds[role], Task: task,
		Manager: agentport.Manager{ID: n.name, Parent: parentName(n)}, Checklist: n.items, Inbox: inbox, Deadline: deadline,
		Tools: tools(role, n)}
	for _, c := range n.children {
		st.Manager.Children = append(st.Manager.Children, c.name)
	}
	if role == agentport.Checker {
		st.Inbox = nil
	}
	cred := s.creds[n.id][role]
	st.Git = agentport.GitAccess{URL: s.remote.URL(), Network: s.remote.Network(), Branch: n.branch, Base: n.from, User: cred.user, Token: cred.token}
	h := &host{s: s, n: n, role: role, tools: map[agentport.Tool]bool{}}
	for _, t := range st.Tools {
		h.tools[t] = true
	}
	s.running++
	began := s.e.now()
	go func() {
		actx, cancel := context.WithDeadline(ctx, deadline)
		end := s.e.Agents.Run(actx, st, h)
		cancel()
		h.close()
		s.ended <- ended{n: n, role: role, end: end, began: began, round: st.Round}
	}()
}

func parentName(n *node) string {
	if n.parent == nil {
		return ""
	}
	return n.parent.name
}

// tools are the host tools of a start. A checker, the only role in verify,
// only leaves Notes: it is fresh and does not read them (concept 2.0 §6.2).
func tools(role agentport.Role, n *node) []agentport.Tool {
	if role == agentport.Checker {
		return []agentport.Tool{agentport.ToolNote}
	}
	ts := []agentport.Tool{agentport.ToolNote, agentport.ToolNotes, agentport.ToolAsk, agentport.ToolEscalate}
	if len(n.children) > 0 {
		ts = append(ts, agentport.ToolAnswer, agentport.ToolAssign)
	}
	return ts
}

// onEnd reads what an agent left on the Run's remote and decides what
// follows.
func (s *schedule) onEnd(ctx context.Context, ev ended) {
	tips, syncErr := s.sync(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	n := ev.n
	n.running = false
	s.running--
	s.records = append(s.records, AgentStart{Manager: n.name, Role: ev.role, Round: ev.round, Reason: ev.end.Reason, Detail: ev.end.Detail, Class: ev.end.Class,
		Agent: ev.end.Agent, Usage: ev.end.Usage, Seconds: s.e.now().Sub(ev.began).Seconds()})
	if ev.end.Cleanup != "" {
		s.leftovers = append(s.leftovers, fmt.Sprintf("%s %s round %d: %s", n.name, ev.role, ev.round, ev.end.Cleanup))
	}
	if s.stopped {
		return
	}
	if ctx.Err() != nil {
		s.stopLocked(StateIncomplete, fmt.Sprintf("the Run's deadline passed or it was cancelled: %v", context.Cause(ctx)))
		return
	}
	if syncErr != nil {
		s.stopLocked(StateIncomplete, fmt.Sprintf("the Run's remote could not be read after %s's %s ended: %v", n.name, ev.role, syncErr))
		return
	}
	s.tips = tips
	tip := tips[n.branch]
	moved := tip != "" && tip != n.tip
	if moved {
		defer func() { n.tip = tip }()
	}
	var progress bool
	var err error
	switch {
	case s.req.Mode == agentport.Verify:
		progress, err = s.afterVerifier(ctx, n, tip, moved)
	case ev.role == agentport.Maker:
		progress, err = s.afterMaker(ctx, n, tip, moved)
	default:
		progress, err = s.afterChecker(ctx, n, tip, moved)
	}
	if err != nil {
		s.stopLocked(StateIncomplete, fmt.Sprintf("reading %s's branch: %v", n.name, err))
		return
	}
	if progress || n.sent {
		n.idle = 0
		return
	}
	n.idle++
	if n.idle >= idleLimit {
		s.stopLocked(StateIncomplete, fmt.Sprintf("Manager %s's %s made no progress in %d starts in a row (last: %s %s)", n.name, ev.role, n.idle, ev.end.Reason, ev.end.Detail))
		return
	}
	if ev.role == agentport.Checker {
		// A checker starts fresh again on the same submission.
		return
	}
	s.hostMessage(n, fmt.Sprintf("Your last start ended (%s) without a submission on %s and without a message. Continue your task.", ev.end.Reason, n.branch))
}

// sync reads every branch of the Run's remote into the host's repository.
func (s *schedule) sync(ctx context.Context) (map[string]string, error) {
	var b bytes.Buffer
	if err := s.remote.Export(ctx, &b); err != nil {
		return nil, err
	}
	return s.e.Git.Import(ctx, &b)
}

// afterMaker: a new submission at the tip goes to the checker once it holds
// every approved child's work.
func (s *schedule) afterMaker(ctx context.Context, n *node, tip string, moved bool) (bool, error) {
	if !moved {
		// An approved Manager started again, for example by an answer,
		// that changed nothing stays approved.
		return n.approved, nil
	}
	n.approved = false
	c, err := s.e.Git.Commit(ctx, tip)
	if err != nil {
		return false, err
	}
	t, perr := runremote.ParseMessage(c.Message)
	if perr != nil || t.Kind != runremote.Submission {
		if n.sent {
			return true, nil
		}
		s.hostMessage(n, fmt.Sprintf("Your branch %s ends at %s, which is no submission. Finish with a submission commit that gives every checklist item a verdict.", n.branch, short(tip)))
		return true, nil
	}
	var missing []string
	// A parent's own work comes first; its children start from its
	// approval (concept 2.0 §6.2).
	for _, ch := range n.children {
		if s.ownFirst(n) {
			break
		}
		if !ch.approved {
			// The child is still working; its approval brings a host
			// message and a new start.
			return true, nil
		}
		ok, err := s.e.Git.Contains(ctx, tip, ch.approval)
		if err != nil {
			return false, err
		}
		if !ok {
			missing = append(missing, fmt.Sprintf("%s's approved work %s (branch %s)", ch.name, ch.approval, ch.branch))
		}
	}
	if len(missing) > 0 {
		s.hostMessage(n, "Your submission "+short(tip)+" does not contain "+strings.Join(missing, ", ")+". Merge it and submit again.")
		return true, nil
	}
	n.submission, n.lastSubmission, n.verdicts = tip, tip, t.Verdicts
	return true, nil
}

// afterChecker: an approval finishes the Manager unless it confirms a model
// problem, which escalates; a rejection goes back to the maker.
func (s *schedule) afterChecker(ctx context.Context, n *node, tip string, moved bool) (bool, error) {
	if !moved {
		return false, nil
	}
	c, err := s.e.Git.Commit(ctx, tip)
	if err != nil {
		return false, err
	}
	t, perr := runremote.ParseMessage(c.Message)
	if perr != nil || t.Kind != runremote.Review || len(c.Parents) != 1 || c.Parents[0] != n.submission {
		return false, nil
	}
	submission := n.submission
	n.submission = ""
	if t.Review == runremote.Rejected {
		var lines []string
		for _, l := range t.Checks {
			if l.Word == runremote.Disputed {
				lines = append(lines, fmt.Sprintf("- %s: %s", l.Item, l.Evidence))
			}
		}
		s.deliver(agentport.Message{Kind: agentport.Review, To: n.name, Text: fmt.Sprintf("Your checker rejected submission %s:\n%s", short(submission), strings.Join(lines, "\n"))})
		return true, nil
	}
	n.approved, n.approval, n.approvedSubmission, n.approvedVerdicts, n.checks = true, tip, submission, n.verdicts, t.Checks
	n.review = s.creds[n.id][agentport.Checker].user
	var problems []string
	for _, v := range n.verdicts {
		if v.Word == runremote.ModelProblem {
			problems = append(problems, fmt.Sprintf("%s: %s", v.Item, v.Evidence))
		}
	}
	if len(problems) > 0 {
		if n.parent == nil {
			for _, v := range n.verdicts {
				if v.Word == runremote.ModelProblem {
					s.decisions = append(s.decisions, OwnerDecision{Manager: n.name, Matter: v.Item + ": " + v.Evidence, Recommendation: recommendation(v.Evidence)})
				}
			}
			s.stopLocked(StateModelDefect, "Root's approved work names a model problem it cannot settle; the Run ends without landing")
			return true, nil
		}
		n.approved = false
		m := s.send(n, agentport.Escalation, n.parent.name, "", "My approved submission "+short(submission)+" names model problems I cannot settle within the model:\n"+strings.Join(problems, "\n"))
		n.waitFor[m.ID] = true
		return true, nil
	}
	return true, nil
}

// afterVerifier: one commit without file changes that holds the checker's
// verdicts.
func (s *schedule) afterVerifier(ctx context.Context, n *node, tip string, moved bool) (bool, error) {
	if !moved {
		return false, nil
	}
	c, err := s.e.Git.Commit(ctx, tip)
	if err != nil {
		return false, err
	}
	t, perr := runremote.ParseMessage(c.Message)
	if perr != nil || t.Kind != runremote.Submission {
		return false, nil
	}
	n.done, n.verdicts, n.approvedVerdicts, n.approvedSubmission, n.lastSubmission, n.approval = true, t.Verdicts, t.Verdicts, tip, tip, tip
	n.review = s.creds[n.id][agentport.Checker].user
	return true, nil
}

// recommendation reads a "Recommendation:" part of an escalation; the host
// adds none of its own beyond asking the owner to settle the model.
func recommendation(text string) string {
	if _, rec, ok := strings.Cut(text, "Recommendation:"); ok && strings.TrimSpace(rec) != "" {
		return strings.TrimSpace(rec)
	}
	return "Settle the matter in the model, then run apply again."
}

// hostMessage gives a Manager's maker a message from the host, which starts
// it again.
func (s *schedule) hostMessage(n *node, text string) {
	s.deliver(agentport.Message{Kind: agentport.Assignment, To: n.name, Text: text})
}

func (s *schedule) deliver(m agentport.Message) agentport.Message {
	m.ID = fmt.Sprintf("m%d", len(s.messages)+1)
	s.messages = append(s.messages, m)
	if n := s.byName(m.To); n != nil {
		n.inbox = append(n.inbox, m)
	}
	s.poke()
	return m
}

// send records a message from a Manager along the tree.
func (s *schedule) send(from *node, kind, to, replyTo, text string) agentport.Message {
	m := s.deliver(agentport.Message{Kind: kind, From: from.name, To: to, ReplyTo: replyTo, Text: text})
	if kind == agentport.Question || kind == agentport.Escalation {
		if p := s.byName(to); p != nil {
			p.unanswered[m.ID] = m
		}
	}
	return m
}

func (s *schedule) byName(name string) *node {
	if name == "" {
		return nil
	}
	if name == "." {
		return s.nodes[model.RootPackage]
	}
	return s.nodes[name]
}

// report fills the Result with what the schedule saw.
func (s *schedule) report(r *Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.State, r.Why = s.state, s.why
	r.Starts = append(r.Starts, s.records...)
	r.Messages = append(r.Messages, s.messages...)
	r.Notes = append(r.Notes, s.notes...)
	r.OwnerDecisions = append(r.OwnerDecisions, s.decisions...)
	r.Cleanup = append(r.Cleanup, s.leftovers...)
	for _, n := range s.order {
		approved := n.approved || (s.req.Mode == agentport.Verify && n.done)
		source, submission := n.verdicts, n.lastSubmission
		if approved {
			source, submission = n.approvedVerdicts, n.approvedSubmission
		}
		verdicts := map[string]runremote.Line{}
		for _, v := range source {
			verdicts[v.Item] = v
		}
		checks := map[string]runremote.Line{}
		for _, c := range n.checks {
			checks[c.Item] = c
		}
		for _, it := range n.items {
			ri := ResultItem{Manager: n.name, Item: it.Key, Part: it.Part, Definition: it.Definition}
			if v, ok := verdicts[it.Key]; ok {
				ri.Verdict, ri.Evidence, ri.Submission = v.Word, v.Evidence, submission
			}
			if approved {
				ri.Checker, ri.Approved = n.review, true
				ri.Check = checks[it.Key].Evidence
			}
			r.Items = append(r.Items, ri)
			if s.req.Mode == agentport.Verify && ri.Verdict == runremote.ModelProblem {
				r.OwnerDecisions = append(r.OwnerDecisions, OwnerDecision{Manager: n.name, Matter: it.Key + ": " + ri.Evidence, Recommendation: recommendation(ri.Evidence)})
			}
			if s.req.Mode == agentport.Verify && ri.Verdict == runremote.Deviates {
				observed, expected, ok := strings.Cut(ri.Evidence, "Expected:")
				if !ok {
					expected = fmt.Sprintf("the artifacts conform to [[%s]]", it.Definition)
				}
				r.Findings = append(r.Findings, ResultFinding{Definition: it.Definition, Observed: strings.TrimSpace(strings.TrimPrefix(observed, "Observed:")), Expected: strings.TrimSpace(expected), Manager: n.name})
			}
		}
	}
}
