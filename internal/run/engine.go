// Package run is the Command engine of concept 3.0 (§6 to §10): `apply` and
// `verify` over one container. Impact gives the checklist; an implementer
// changes a folder; the Command commits it and runs a gate no agent can talk
// past; a fresh reviewer answers every item in a clean copy; a refusal goes back
// for at most two repair rounds; and the Command writes the Result and lands it.
// Sessions start through the agent port, the container through Sandboxes, and
// the Command's own Git work goes through Git.
//
// The engine is described in internal/run/README.md.
package run

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

const (
	// repairRounds bounds how often a refusal goes back to the implementer.
	repairRounds = 2
	// maxRefusals bounds how many compile errors one refusal lists.
	maxRefusals = 30
)

// Request is one apply or verify.
type Request struct {
	Mode agentport.Mode
	// Run names the Run; empty makes one from the time.
	Run string
	// Target is the branch of the target repository the Command starts from
	// and lands on.
	Target string
	// Base is the commit impact is computed from; empty means the last work
	// Markitect landed on the target.
	Base     string
	Full     bool
	Findings []analysis.Finding
	Limits   Limits
}

// Limits bound a Command: finite budgets, and deadlines that reach every
// blocking operation.
type Limits struct {
	// Deadline bounds the whole Command, landing and cleanup's start included.
	Deadline time.Time
	// Session bounds one session.
	Session time.Duration
}

// DefaultLimits are the limits of a Command that names none.
func DefaultLimits(now time.Time) Limits {
	return Limits{Deadline: now.Add(6 * time.Hour), Session: 3 * time.Hour}
}

// Engine runs Commands.
type Engine struct {
	Git       Git
	Sandboxes Sandboxes
	Agents    agentport.Agents
	// Secrets are values no landing may carry: the user's token.
	Secrets [][]byte
	// Now defaults to the clock.
	Now func() time.Time
	// Cleanup bounds the removal of the container after the Command ends.
	Cleanup time.Duration
}

// ErrUsage marks a request the engine cannot run, such as an unknown base.
var ErrUsage = errors.New("cannot run")

// runName is what a Run may be called: it names a branch, a container and its
// volumes.
var runName = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,62}$`)

// RunBranchPrefix is where a Result that does not land stays.
const RunBranchPrefix = "markitect/runs/"

// command is one running Command.
type command struct {
	e   *Engine
	req Request
	r   *Result
	cur *project.Project
	imp analysis.Impact
	box Sandbox
	// before is the project at the base.
	before *project.Project
	// items is the list as it stands; answers holds each item's latest answer.
	items   []Item
	answers map[string]Answer
	// work is the commit the work stands at.
	work string

	mu     sync.Mutex
	rounds map[agentport.Role]int
	seen   []agentport.Seen
}

// Run runs one Command and returns its Result. An error means it could not
// start; once it started, every way it ends is a Result, and only a landed
// Result changed the target branch.
func (e *Engine) Run(ctx context.Context, req Request) (*Result, error) {
	now := e.now()
	if req.Limits == (Limits{}) {
		req.Limits = DefaultLimits(now)
	}
	if req.Run == "" {
		// The name names the Run's branch, Result page and container, and
		// Commands may start side by side in the same second, so the time alone
		// is not enough.
		b := make([]byte, 3)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		req.Run = "r" + now.UTC().Format("20060102-150405") + "-" + hex.EncodeToString(b)
	}
	if req.Mode != agentport.Apply && req.Mode != agentport.Verify {
		return nil, fmt.Errorf("%w: mode %q", ErrUsage, req.Mode)
	}
	if !runName.MatchString(req.Run) {
		return nil, fmt.Errorf("%w: run %q: use lower-case letters, digits, '.', '_' or '-', starting with a letter or digit", ErrUsage, req.Run)
	}
	if lim := req.Limits; lim.Session <= 0 || !lim.Deadline.After(now) {
		return nil, fmt.Errorf("%w: the limits need a session limit and a deadline in the future", ErrUsage)
	}
	ctx, cancel := context.WithDeadline(ctx, req.Limits.Deadline)
	defer cancel()
	r := &Result{Format: ResultFormat, Run: req.Run, Command: string(req.Mode), Outcome: NotLanded, Target: req.Target, Full: req.Full, Began: now,
		Items: []ResultItem{}, OwnerDecisions: []OwnerDecision{}, Findings: []ResultFinding{}, Sessions: []SessionRecord{}, Cleanup: []string{},
		RequiredChecks: RequiredChecks{Checks: []CheckResult{}}}
	c := &command{e: e, req: req, r: r, answers: map[string]Answer{}, rounds: map[agentport.Role]int{}}
	if err := c.prepare(ctx); err != nil || r.State != "" {
		return e.done(r, err)
	}
	read, write := io.Pipe()
	go func() { write.CloseWithError(e.Git.Bundle(ctx, r.Start, write)) }()
	box, err := e.Sandboxes.Start(ctx, req.Run, r.Start, read)
	read.Close()
	if err != nil {
		r.State = StateIncomplete
		r.Why = fmt.Sprintf("the Command's container did not start: %v", err)
		r.Cleanup = append(r.Cleanup, "the Command's container may remain if its start failed half-way")
		return e.finish(r), nil
	}
	c.box = box
	defer c.release(&r.Cleanup)
	c.work = r.Start
	if req.Mode == agentport.Verify {
		c.verify(ctx)
	} else {
		c.apply(ctx)
	}
	return e.finish(r), nil
}

// prepare reads the tip of the target, the base and the checklist into the
// Result (concept 3.0 §6, steps 1 and 2). A real spec error, or nothing
// affected, sets the Result's state, and no agent starts. A file a hand edit
// left unlisted does not stop the Command: it becomes an item.
func (c *command) prepare(ctx context.Context) error {
	e, r, req := c.e, c.r, c.req
	start, err := e.Git.Fetch(ctx, req.Target)
	if err != nil {
		return fmt.Errorf("%w: the target branch %q: %v", ErrUsage, req.Target, err)
	}
	base := req.Base
	if base == "" {
		landed, ok, err := e.lastBase(ctx, start)
		switch {
		case err != nil:
			return err
		case ok:
			base = landed
		case req.Full || len(req.Findings) > 0:
			// Their items do not depend on what changed since a landing.
			base = start
		default:
			return fmt.Errorf("%w: no Run has landed on %s yet, so there is no default base; name one with --base, or cover everything with --full", ErrUsage, req.Target)
		}
	}
	bc, err := e.Git.Commit(ctx, base)
	if err != nil {
		return fmt.Errorf("%w: base %q is no commit: %v", ErrUsage, base, err)
	}
	base = bc.ID
	branches, err := e.Git.Branches(ctx, RunBranchPrefix)
	if err != nil {
		return err
	}
	if _, taken := branches[RunBranchPrefix+req.Run]; taken {
		return fmt.Errorf("%w: run %q already has its branch %s%s; name another or delete it", ErrUsage, req.Run, RunBranchPrefix, req.Run)
	}
	r.Base, r.Start = base, start
	r.FullCheck = e.fullCheck(ctx, req.Target, start)

	if c.cur, err = project.Load(e.Git.State(ctx, start)); err != nil {
		return err
	}
	if real := realErrors(c.cur.Errors); len(real) > 0 {
		r.State = StateCheckFailed
		r.Why = fmt.Sprintf("check fails at %s with %d compile errors, so no agent starts; the first: %s", short(start), len(real), real[0].String())
		return nil
	}
	if c.before, err = project.Load(e.Git.State(ctx, base)); err != nil {
		return err
	}
	if c.imp, err = c.impact(ctx, c.cur, start); err != nil {
		return err
	}
	c.items = checklist(c.imp, c.before, c.cur, req.Findings)
	r.ImpactNotes, r.impact = c.imp.Notes, impactKeys(c.imp, c.items)
	if len(c.items) == 0 {
		r.State = StateNothing
		r.Why = fmt.Sprintf("nothing changed between %s and %s that reaches the spec; no agent starts", short(base), short(start))
	}
	return nil
}

// impact is what the change from the base to a commit reaches, with that
// commit's registry.
func (c *command) impact(ctx context.Context, cur *project.Project, commit string) (analysis.Impact, error) {
	all, err := c.e.Git.Changed(ctx, c.r.Base, commit)
	if err != nil {
		return analysis.Impact{}, err
	}
	var changed []analysis.Change
	for _, f := range all {
		if !strings.HasPrefix(f.Path, project.Dir+"/") {
			changed = append(changed, f)
		}
	}
	return analysis.ComputeImpact(analysis.ImpactInput{Base: c.before, Current: cur, Changed: changed, Findings: c.req.Findings, Full: c.req.Full}), nil
}

// realErrors leaves out what a hand edit can leave behind without the spec
// being wrong: the items a Command settles.
func realErrors(ds model.Diagnostics) model.Diagnostics {
	var out model.Diagnostics
	for _, d := range ds {
		if d.Code != codeUnregistered {
			out = append(out, d)
		}
	}
	return out
}

// apply is concept 3.0 §6, steps 4 to 9.
func (c *command) apply(ctx context.Context) {
	r := c.r
	repo, err := c.box.Deliver(ctx, agentport.ImplementerUser, r.Start)
	if err != nil {
		c.stop(ctx, StateIncomplete, fmt.Sprintf("the implementer's folder could not be made: %v", err))
		return
	}
	note := c.box.NotePath(agentport.ImplementerUser, stopNote)
	prompt := implementerPrompt(repo, r.Base, note, c.items, c.cur.Checks)
	conversation, reviewed := "", r.Start
	for {
		end := c.session(ctx, agentport.Implementer, agentport.ImplementerUser, repo, prompt, conversation)
		if end.Reason != agentport.Finished {
			c.stop(ctx, StateIncomplete, sessionFailure(agentport.Implementer, end))
			return
		}
		conversation = end.Conversation
		if err := c.collect(ctx, r.Repairs); err != nil {
			c.stop(ctx, StateIncomplete, fmt.Sprintf("the work could not be committed: %v", err))
			return
		}
		r.Work = c.work
		if text, ok, err := c.box.Note(ctx, agentport.ImplementerUser, stopNote); err != nil {
			c.stop(ctx, StateIncomplete, fmt.Sprintf("the implementer's note could not be read: %v", err))
			return
		} else if ok {
			r.OwnerDecisions = append(r.OwnerDecisions, OwnerDecision{Matter: string(text), Recommendation: "Settle it in the spec and run the Command again."})
			c.stop(ctx, StateSpecProblem, "the implementer stopped: the spec needs the owner")
			return
		}
		refusals, err := c.gate(ctx, r.Start, c.work, false)
		if err != nil {
			// An environment failure uses no repair round.
			c.stop(ctx, StateIncomplete, fmt.Sprintf("the gate could not run: %v", err))
			return
		}
		if len(refusals) == 0 {
			var done bool
			if refusals, done = c.review(ctx, reviewed); done {
				return
			}
			reviewed = c.work
		}
		if len(refusals) == 0 {
			c.fillItems()
			c.land(ctx)
			return
		}
		if r.Repairs == repairRounds {
			c.stop(ctx, StateRefused, fmt.Sprintf("the repair rounds ran out, and these stay open: %s", strings.Join(refusals, "; ")))
			return
		}
		r.Repairs++
		r.SentBack = append(r.SentBack, strings.Join(refusals, "; "))
		prompt = repairPrompt(note, refusals)
	}
}

// collect commits the implementer's folder and brings the commit to the host.
func (c *command) collect(ctx context.Context, round int) error {
	message := fmt.Sprintf("Markitect Run %s: the implementer's work, round %d\n", c.r.Run, round+1)
	commit, bundle, err := c.box.Collect(ctx, c.work, message)
	if err != nil {
		return err
	}
	if bundle != nil {
		defer bundle.Close()
		if err := c.e.Git.Import(ctx, bundle); err != nil {
			return err
		}
	}
	c.work = commit
	return nil
}

// gate is concept 3.0 §6, step 6: fixed checks with no agent. The comparisons
// and `check` run outside the container on the commit itself, so nothing left
// inside can affect them. The required checks run inside, in a clean copy, as a
// user that holds nothing of the sessions'. from is the state the commit's own
// changes are measured from. It returns why the work is refused; an error means
// the gate could not run, which is the environment's failure and no refusal.
func (c *command) gate(ctx context.Context, from, commit string, bring bool) ([]string, error) {
	e := c.e
	changed, err := e.Git.Changed(ctx, from, commit)
	if err != nil {
		return nil, err
	}
	links, err := e.Git.Links(ctx, commit)
	if err != nil {
		return nil, err
	}
	var refusals []string
	for _, f := range changed {
		// Without regard to case: a variant of the folder's name is the same
		// folder in a Windows or macOS checkout.
		if strings.HasPrefix(strings.ToLower(f.Path), project.Dir+"/") && !registryPage(f.Path) {
			refusals = append(refusals, fmt.Sprintf("%s changed: everything under %s/ except the registry pages is the owner's", f.Path, project.Dir))
		}
		if to, ok := links[f.Path]; ok && leavesTree(f.Path, to) {
			refusals = append(refusals, fmt.Sprintf("%s is a link that leaves the repository, to %s", f.Path, to))
		}
	}
	p, err := project.Load(e.Git.State(ctx, commit))
	if err != nil {
		return nil, err
	}
	for i, d := range p.Errors {
		if i == maxRefusals {
			refusals = append(refusals, fmt.Sprintf("and %d more compile errors", len(p.Errors)-maxRefusals))
			break
		}
		refusals = append(refusals, "markitect check: "+d.String())
	}
	for _, s := range p.Registry.Stale {
		refusals = append(refusals, fmt.Sprintf("a stale registry entry on %s: %s %s%s", s.PageFile, s.Code, s.File, s.Target))
	}
	if len(refusals) > 0 {
		return refusals, nil
	}
	return c.requiredChecks(ctx, commit, p.Checks, bring)
}

// requiredChecks runs the required checks on a commit and makes their results
// the Result's. A check that could not start, for lack of a program or service,
// is an error; one that failed is a refusal.
func (c *command) requiredChecks(ctx context.Context, commit string, checks []project.Check, bring bool) ([]string, error) {
	r := c.r
	if len(checks) == 0 {
		return nil, nil
	}
	var bundle io.Reader
	if bring {
		read, write := io.Pipe()
		go func() { write.CloseWithError(c.e.Git.Bundle(ctx, commit, write)) }()
		defer read.Close()
		bundle = read
	}
	results, err := c.box.Check(ctx, bundle, commit, checks)
	r.RequiredChecks = RequiredChecks{Commit: commit, Checks: results}
	if results == nil {
		r.RequiredChecks.Checks = []CheckResult{}
	}
	if err != nil {
		r.RequiredChecks.Note = fmt.Sprintf("the checks could not run: %v", err)
		return nil, err
	}
	var refusals []string
	for _, res := range results {
		if res.NotRun {
			r.RequiredChecks.Note = fmt.Sprintf("the check %s could not start", res.Name)
			return nil, fmt.Errorf("the required check %s could not start (exit %d): %s", res.Name, res.Exit, res.Output)
		}
		if !res.Passed {
			refusals = append(refusals, fmt.Sprintf("the required check %s failed:\n%s", res.Name, res.Output))
		}
	}
	r.RequiredChecks.Ran = len(refusals) == 0 && len(results) == len(checks)
	return refusals, nil
}

// leavesTree says whether a symbolic link at file points outside the repository.
func leavesTree(file, to string) bool {
	if path.IsAbs(to) || strings.Contains(to, `\`) {
		return true
	}
	// A ".." after another part can pass through a link, where the text says
	// nothing about the way.
	inner := false
	for _, part := range strings.Split(to, "/") {
		switch {
		case part == ".." && inner:
			return true
		case part != ".." && part != "" && part != ".":
			inner = true
		}
	}
	target := path.Join(path.Dir(file), to)
	return target == ".." || strings.HasPrefix(target, "../")
}

func registryPage(file string) bool {
	return file == project.RootPage || strings.HasPrefix(file, project.RegistryDir+"/")
}

// review rebuilds the list from the work and has fresh reviewers answer it
// (concept 3.0 §6, steps 5, 7 and 8). A later round asks only about what the new
// diff touched and what is still open; other answers stand. It returns why the
// work goes back, and true when the Command has ended.
func (c *command) review(ctx context.Context, reviewed string) ([]string, bool) {
	r := c.r
	state := c.e.Git.State(ctx, c.work)
	after, err := project.Load(state)
	if err != nil {
		c.stop(ctx, StateIncomplete, fmt.Sprintf("the work could not be read: %v", err))
		return nil, true
	}
	imp, err := c.impact(ctx, after, c.work)
	if err != nil {
		c.stop(ctx, StateIncomplete, fmt.Sprintf("the work's impact could not be read: %v", err))
		return nil, true
	}
	c.items = mergeItems(c.items, checklist(imp, c.before, after, c.req.Findings))
	r.impact = dedupe(append(r.impact, impactKeys(imp, c.items)...))
	diff, err := c.e.Git.Diff(ctx, reviewed, c.work)
	if err != nil {
		c.stop(ctx, StateIncomplete, fmt.Sprintf("the work's diff could not be read: %v", err))
		return nil, true
	}
	items, earlier, err := c.toReview(ctx, reviewed)
	if err != nil {
		c.stop(ctx, StateIncomplete, fmt.Sprintf("the work's changes could not be read: %v", err))
		return nil, true
	}
	if len(items) == 0 {
		return nil, false
	}
	if failure := c.answer(ctx, state, items, func(repo, note string, group []Item) string {
		return reviewerPrompt(reviewInput{repo: repo, note: note, base: reviewed, commit: c.work, items: group, diff: diff, checks: r.RequiredChecks.Checks, earlier: earlier})
	}); failure != "" {
		c.stop(ctx, StateIncomplete, failure)
		return nil, true
	}
	r.Depth = measure(c.answers, c.seen)
	var refusals []string
	for _, it := range c.items {
		switch a := c.answers[it.Key]; a.Word {
		case AnswerSpecProblem:
			r.OwnerDecisions = append(r.OwnerDecisions, OwnerDecision{Matter: fmt.Sprintf("%s: %s", it.Key, a.Explanation), Recommendation: "Settle it in the spec and run the Command again."})
		case AnswerWrong:
			refusals = append(refusals, fmt.Sprintf("%s: %s %s", it.Key, a.Explanation, pointers(a.Pointers)))
		}
	}
	if len(r.OwnerDecisions) > 0 {
		c.stop(ctx, StateSpecProblem, "a reviewer found a spec problem: the spec needs the owner")
		return nil, true
	}
	return refusals, false
}

// toReview is what a review of the work must answer: everything on the first
// review; later, what has no answer or is open, and whatever the new diff
// touched. It also returns what the earlier reviews sent back.
func (c *command) toReview(ctx context.Context, reviewed string) ([]Item, []string, error) {
	if len(c.answers) == 0 {
		return c.items, nil, nil
	}
	changed, err := c.e.Git.Changed(ctx, reviewed, c.work)
	if err != nil {
		return nil, nil, err
	}
	touched := map[string]bool{}
	for _, f := range changed {
		touched[f.Path] = true
	}
	var items []Item
	var earlier []string
	for _, it := range c.items {
		a, answered := c.answers[it.Key]
		again := !answered || a.Word == AnswerWrong
		for _, f := range it.Look {
			again = again || touched[f]
		}
		for _, p := range a.Pointers {
			again = again || touched[p.File]
		}
		if a.Word == AnswerWrong {
			earlier = append(earlier, fmt.Sprintf("%s: %s", it.Key, a.Explanation))
		}
		if again {
			items = append(items, it)
		}
	}
	return items, earlier, nil
}

// answer runs the reviewers over items and takes their answers: one reviewer
// over a change's list; with --full, the starting point of one reviewer per
// Package, side by side, each in a clean copy of its own with only its own
// items (concept 3.0 §8). It returns why the answers cannot stand, or "".
func (c *command) answer(ctx context.Context, state project.Repository, items []Item, prompt func(repo, note string, group []Item) string) string {
	groups := [][]Item{items}
	if c.req.Full {
		groups = c.byPackage(items)
	}
	failures := make([]string, len(groups))
	var wg sync.WaitGroup
	users := make(chan string, agentport.ReviewerUsers)
	for n := 1; n <= agentport.ReviewerUsers; n++ {
		users <- agentport.ReviewerUser(n)
	}
	for i, group := range groups {
		wg.Add(1)
		go func() {
			defer wg.Done()
			user := <-users
			defer func() { users <- user }()
			failures[i] = c.answerGroup(ctx, state, user, group, prompt)
		}()
	}
	wg.Wait()
	for _, f := range failures {
		if f != "" {
			return f
		}
	}
	return ""
}

// byPackage splits a list by the Package of each item's Definition; an item
// with none goes with Root's.
func (c *command) byPackage(items []Item) [][]Item {
	by := map[string][]Item{}
	for _, it := range items {
		pkg := model.RootPackage
		if d := c.cur.Model.Definitions[it.Definition]; d != nil {
			pkg = d.Package
		}
		by[pkg] = append(by[pkg], it)
	}
	var groups [][]Item
	for _, pkg := range sortedKeys(by) {
		groups = append(groups, by[pkg])
	}
	return groups
}

// answerGroup is one fresh reviewer in a clean copy of its own. What the Command
// cannot accept of its answers goes back to it once.
func (c *command) answerGroup(ctx context.Context, state project.Repository, user string, items []Item, prompt func(repo, note string, group []Item) string) string {
	repo, err := c.box.Deliver(ctx, user, c.work)
	if err != nil {
		return fmt.Sprintf("the reviewer's copy could not be made: %v", err)
	}
	note := c.box.NotePath(user, answersNote)
	// The reviewer starts outside the copy, so nothing the work committed there
	// is discovered from where it starts.
	dir := path.Dir(repo)
	end := c.session(ctx, agentport.Reviewer, user, dir, prompt(repo, note, items), "")
	if end.Reason != agentport.Finished {
		return sessionFailure(agentport.Reviewer, end)
	}
	seen := end.Seen
	answers, problems, links, err := c.readAnswers(ctx, state, repo, user, items, seen)
	if err != nil {
		return fmt.Sprintf("the reviewer's answers could not be read: %v", err)
	}
	if len(problems) > 0 {
		end = c.session(ctx, agentport.Reviewer, user, dir, correctionPrompt(note, problems), end.Conversation)
		if end.Reason != agentport.Finished {
			return sessionFailure(agentport.Reviewer, end)
		}
		// The second look at the files counts with the first.
		seen.Opened = append(seen.Opened, end.Seen.Opened...)
		seen.Commands = append(seen.Commands, end.Seen.Commands...)
		seen.Subagents += end.Seen.Subagents
		answers, problems, links, err = c.readAnswers(ctx, state, repo, user, items, seen)
		if err != nil {
			return fmt.Sprintf("the reviewer's answers could not be read: %v", err)
		}
		if len(problems) > 0 {
			return "the review could not be accepted after one request to correct it: " + strings.Join(problems, "; ")
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, a := range answers {
		c.answers[k] = a
	}
	c.seen = append(c.seen, seen)
	c.r.MissingLinks = dedupe(append(c.r.MissingLinks, links...))
	if text := strings.TrimSpace(end.Text); text != "" {
		if c.r.NotTested != "" {
			text = c.r.NotTested + " " + text
		}
		c.r.NotTested = text
	}
	return ""
}

func (c *command) readAnswers(ctx context.Context, state project.Repository, repo, user string, items []Item, seen agentport.Seen) (map[string]Answer, []string, []string, error) {
	raw, ok, err := c.box.Note(ctx, user, answersNote)
	if err != nil {
		return nil, nil, nil, err
	}
	if !ok {
		return nil, []string{"you wrote no answers to " + c.box.NotePath(user, answersNote)}, nil, nil
	}
	answers, problems := parseAnswers(raw, items)
	checked, links := checkAnswers(state, repo, items, answers, seen)
	return answers, append(problems, checked...), links, nil
}

// verify is the reviewer alone on a clean checkout of the commit (concept 3.0
// §12): it changes nothing and ends with Findings.
func (c *command) verify(ctx context.Context) {
	r := c.r
	if _, err := c.requiredChecks(ctx, r.Start, c.cur.Checks, false); err != nil {
		c.stop(ctx, StateIncomplete, fmt.Sprintf("the required checks could not run: %v", err))
		return
	}
	state := c.e.Git.State(ctx, r.Start)
	if failure := c.answer(ctx, state, c.items, func(repo, note string, group []Item) string {
		return reviewerPrompt(reviewInput{repo: repo, note: note, base: r.Base, commit: r.Start, items: group, checks: r.RequiredChecks.Checks, verify: true})
	}); failure != "" {
		c.stop(ctx, StateIncomplete, failure)
		return
	}
	r.Depth = measure(c.answers, c.seen)
	unsettled := 0
	for _, it := range c.items {
		a := c.answers[it.Key]
		// A file the registry does not account for fails `check` whatever a
		// reviewer says of it.
		if it.Kind == ItemUnlisted || it.Kind == ItemStale {
			unsettled++
		}
		switch a.Word {
		case AnswerWrong:
			if it.Definition != "" {
				r.Findings = append(r.Findings, ResultFinding{Definition: it.Definition, Observed: strings.TrimSpace(a.Explanation + " " + pointers(a.Pointers)), Expected: "what the Definition says"})
			} else if it.Kind != ItemUnlisted && it.Kind != ItemStale {
				unsettled++
			}
		case AnswerSpecProblem:
			r.OwnerDecisions = append(r.OwnerDecisions, OwnerDecision{Matter: fmt.Sprintf("%s: %s", it.Key, a.Explanation), Recommendation: "Settle it in the spec."})
		}
	}
	var found []string
	if n := len(r.Findings); n > 0 {
		found = append(found, fmt.Sprintf("%d Findings", n))
	}
	if n := len(r.OwnerDecisions); n > 0 {
		found = append(found, fmt.Sprintf("%d decisions for the owner", n))
	}
	if unsettled > 0 {
		found = append(found, fmt.Sprintf("%d items the registry or a reviewer does not settle", unsettled))
	}
	if len(c.cur.Checks) > 0 && !r.RequiredChecks.Ran {
		found = append(found, "required checks that did not pass")
	}
	if len(found) > 0 {
		r.State, r.Why = StateFindings, "verify found "+strings.Join(found, " and ")
	} else {
		r.State, r.Why = StateVerified, "every item answered without a Finding"
	}
	if isFullCheck(r) {
		// This Run is the full check: it examined its start.
		r.FullCheck = FullCheck{Since: r.Start}
	}
	c.fillItems()
	c.release(&r.Cleanup)
	c.e.keepOn(ctx, r, r.Start, "", "", "", nil)
}

// session runs one session and records how it ended.
func (c *command) session(ctx context.Context, role agentport.Role, user, dir, prompt, conversation string) agentport.End {
	e, lim := c.e, c.req.Limits
	began := e.now()
	deadline := began.Add(lim.Session)
	if lim.Deadline.Before(deadline) {
		deadline = lim.Deadline
	}
	c.mu.Lock()
	c.rounds[role]++
	round := c.rounds[role]
	c.mu.Unlock()
	end := e.Agents.Run(ctx, c.box, agentport.Session{Run: c.req.Run, Role: role, User: user, Round: round, Dir: dir, Prompt: prompt, Conversation: conversation, Deadline: deadline})
	c.mu.Lock()
	defer c.mu.Unlock()
	c.r.Sessions = append(c.r.Sessions, SessionRecord{Role: role, Round: round, Reason: end.Reason, Detail: end.Detail, Class: end.Class, Agent: end.Agent, Models: end.Models, Usage: end.Usage, Seconds: e.now().Sub(began).Seconds()})
	c.r.Usage = c.r.Usage.Add(end.Usage)
	return end
}

// stop ends a Command that does not land: its work and Result stay on the Run's
// own branch, and the Result says what is open.
func (c *command) stop(ctx context.Context, state, why string) {
	r := c.r
	c.fillItems()
	r.Depth = measure(c.answers, c.seen)
	c.release(&r.Cleanup)
	c.e.keepOn(ctx, r, c.work, "", state, why, nil)
}

// fillItems writes each item's latest answer into the Result.
func (c *command) fillItems() {
	items := make([]ResultItem, 0, len(c.items))
	for _, it := range c.items {
		a := c.answers[it.Key]
		items = append(items, ResultItem{Item: it.Key, Kind: it.Kind, Definition: it.Definition, Look: it.Look, Answer: a.Word, Pointers: a.Pointers, Explanation: a.Explanation})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Item < items[j].Item })
	c.r.Items = items
}

// release ends the container once; into collects what that left.
func (c *command) release(into *[]string) {
	if c.box == nil {
		return
	}
	box := c.box
	c.box = nil
	cctx, cancel := context.WithTimeout(context.Background(), c.e.cleanup())
	defer cancel()
	if err := box.Close(cctx); err != nil {
		*into = append(*into, fmt.Sprintf("the Command's container: %v", err))
	}
}

// sessionFailure says how a session ended in a Result's words.
func sessionFailure(role agentport.Role, end agentport.End) string {
	why := fmt.Sprintf("the %s's session ended as %s", role, end.Reason)
	if end.Class != "" {
		why += " (" + string(end.Class) + ")"
	}
	if end.Detail != "" {
		why += ": " + end.Detail
	}
	return why
}

// done ends a Command that started no container: with an error when it could
// not run, else with its Result.
func (e *Engine) done(r *Result, err error) (*Result, error) {
	if err != nil {
		return nil, err
	}
	return e.finish(r), nil
}

// finish completes the Result once the Command returns. One that wrote its page
// keeps the end time stamped there, so the page and the Result agree.
func (e *Engine) finish(r *Result) *Result {
	if r.Ended.IsZero() {
		return e.stamp(r)
	}
	return r
}

// stamp sets the Result's end time, as the page it is about to write states it,
// and takes the secrets out of everything a session said in it.
func (e *Engine) stamp(r *Result) *Result {
	r.Ended = e.now()
	e.scrub(r)
	return r
}

// scrub replaces every secret in the Result's words, in one place: stop notes,
// answers, closing words and check output all come from code under review.
func (e *Engine) scrub(r *Result) {
	data, err := json.Marshal(r)
	if err != nil {
		return
	}
	found := false
	for _, s := range e.Secrets {
		if len(s) > 0 && bytes.Contains(data, s) {
			data, found = bytes.ReplaceAll(data, s, []byte("[redacted]")), true
		}
	}
	if found {
		// An error here leaves the Result as it was, and the page says less.
		if err := json.Unmarshal(data, r); err != nil {
			r.OwnerDecisions, r.Findings, r.NotTested = nil, nil, ""
			r.Why = "the Result held the account token and could not be cleaned"
		}
	}
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Engine) cleanup() time.Duration {
	if e.Cleanup > 0 {
		return e.Cleanup
	}
	return 2 * time.Minute
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
