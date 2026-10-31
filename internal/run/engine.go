// Package run is the Run engine of concept 2.0 (§6, §7): one Run per change.
// Impact decides who works; every working Manager is a maker and a fresh
// checker that hand off through commits on the Run's own Git remote; parents
// merge their children's approved work, and Root's checker is the final
// check. The host writes the Result from commits, verdicts and checks and
// lands it as in a merge queue. Agents start through the agent port, the
// remote through Remotes, and the host's own Git work goes through Git.
//
// The engine is described in internal/run/README.md.
package run

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// Request is one apply or verify.
type Request struct {
	Mode agentport.Mode
	// Run names the Run; empty makes one from the time.
	Run string
	// Target is the branch of the target repository the Run starts from
	// and lands on.
	Target string
	// Base is the commit impact is computed from; empty means the last
	// commit Markitect landed on the target.
	Base     string
	Full     bool
	Findings []analysis.Finding
	Limits   Limits
}

// Limits bound a Run (concept 1.0 §20): one budget that corrections share,
// and deadlines that reach every blocking operation.
type Limits struct {
	// Deadline bounds the whole Run, landing and cleanup's start included.
	Deadline time.Time
	// Starts is the number of agent starts the Run may make.
	Starts int
	// Rounds is the number of attempts one Manager's maker may make at its
	// own work: its first start and each start after its work was refused,
	// by its checker's rejection or by the host because what it pushed was
	// no submission or lacked work its task named. Starts that other
	// Managers or the host cause (a message in its inbox, a reminder of
	// unanswered children, its children's changed approved work) are no
	// attempts; Starts, the idle rule and the deadlines bound them. Checker
	// starts are not counted: each reviews one submission, and the idle rule
	// stops a checker that leaves no review twice in a row.
	Rounds int
	// Start bounds one agent start.
	Start time.Duration
}

// DefaultLimits are the limits of a Run that names none.
func DefaultLimits(now time.Time) Limits {
	return Limits{Deadline: now.Add(4 * time.Hour), Starts: 200, Rounds: 6, Start: time.Hour}
}

// Engine runs Runs.
type Engine struct {
	Git     Git
	Remotes Remotes
	Agents  agentport.Agents
	// Now and Token default to the clock and a random token.
	Now   func() time.Time
	Token func() (string, error)
	// Cleanup bounds the removal of the Run's remote after the Run ends.
	Cleanup time.Duration
	// LockPoll is how often a Run that waits in the target branch's lock
	// reads it again; zero means 15 seconds.
	LockPoll time.Duration
}

// ErrUsage marks a request the engine cannot run, such as an unknown base.
var ErrUsage = errors.New("cannot run")

// runName is what a Run may be called: it names a branch, containers and a
// network.
var runName = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,62}$`)

// RunBranchPrefix is where a Result that does not land stays.
const RunBranchPrefix = "markitect/runs/"

// Run runs one Run and returns its Result. An error means the Run could not
// start; once it started, every way it ends is a Result, and only a landed
// Result changed the target branch.
func (e *Engine) Run(ctx context.Context, req Request) (*Result, error) {
	now := e.now()
	if req.Limits == (Limits{}) {
		req.Limits = DefaultLimits(now)
	}
	if req.Run == "" {
		// The name names the Run's branch, Result page, containers and
		// lock entry, and Runs may start side by side in the same second,
		// on one host or on several, so the time alone is not enough.
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
	if lim := req.Limits; lim.Starts < 1 || lim.Rounds < 1 || lim.Start < 0 || !lim.Deadline.After(now) {
		return nil, fmt.Errorf("%w: the limits need at least one start and one round per Manager, and a deadline in the future", ErrUsage)
	}
	ctx, cancel := context.WithDeadline(ctx, req.Limits.Deadline)
	defer cancel()
	r := &Result{Format: ResultFormat, Run: req.Run, Command: string(req.Mode), Outcome: NotLanded, Target: req.Target, Full: req.Full, Began: now,
		Items: []ResultItem{}, NotParticipating: []NotParticipant{}, OwnerDecisions: []OwnerDecision{}, Findings: []ResultFinding{},
		Notes: []agentport.Note{}, Starts: []AgentStart{}, Messages: []agentport.Message{}, Cleanup: []string{},
		RequiredChecks: RequiredChecks{Checks: []runremote.CheckResult{}}}

	// In apply the Run takes its place in the target branch's lock: Runs
	// whose impact overlaps take turns, others work side by side (concept
	// 2.0 §6.5). When it had to wait and the target moved meanwhile, it
	// starts again from the new tip. verify changes nothing and takes no
	// place.
	var (
		start string
		cur   *project.Project
		imp   analysis.Impact
	)
	nonce, err := e.token()
	if err != nil {
		return nil, err
	}
	until := req.Limits.Deadline.Add(2 * e.cleanup())
	left := func() {
		if err := e.leave(ctx, req.Target, nonce); err != nil {
			r.Cleanup = append(r.Cleanup, fmt.Sprintf("the Run's entry in the lock of %s stays until %s: %v", req.Target, until.UTC().Format(time.RFC3339), err))
		}
	}
	waitedOut := func() (*Result, error) {
		r.State, r.Why = StateIncomplete, "the Run reached its deadline in the target branch's lock, before any agent started"
		return e.finish(r), nil
	}
	for {
		var err error
		start, cur, imp, err = e.prepare(ctx, req, r)
		switch {
		case err != nil && ctx.Err() != nil && len(r.Queue) > 0:
			return waitedOut()
		case err != nil || r.State != "":
			return e.done(r, err)
		}
		if req.Mode == agentport.Verify {
			break
		}
		waited, err := e.enter(ctx, req.Target, lockEntry{Run: req.Run, Until: until, Nonce: nonce, Impact: r.impact})
		r.Queue = append(r.Queue, waited...)
		if err != nil {
			// A write that reported a failure may have taken effect.
			left()
			switch {
			case errors.Is(err, errRunNameTaken):
				return nil, fmt.Errorf("%w: %v", ErrUsage, err)
			case ctx.Err() != nil:
				return waitedOut()
			}
			return nil, err
		}
		tip, err := e.Git.Fetch(ctx, req.Target)
		if err == nil && tip == start {
			break
		}
		// Its entry holds the impact of the old tip; it must be gone before
		// the Run enters again with the new one.
		if lerr := e.leave(ctx, req.Target, nonce); lerr != nil {
			return nil, fmt.Errorf("leaving the lock of %s to start again from its new tip: %w; the Run's entry stays until %s", req.Target, lerr, until.UTC().Format(time.RFC3339))
		}
		switch {
		case err != nil && ctx.Err() != nil:
			return waitedOut()
		case err != nil:
			return nil, err
		}
		r.Queue = append(r.Queue, fmt.Sprintf("%s moved from %s to %s while the Run waited, so it started again from there", req.Target, short(start), short(tip)))
	}
	// The Run holds its lock entry and, once started, its remote until
	// release, which a Run that keeps its work calls before it writes its
	// page, so the page says what both left, and a landed Run after landing.
	locked := req.Mode == agentport.Apply
	var remote Remote
	release := func(into *[]string) {
		if remote != nil {
			cctx, ccancel := context.WithTimeout(context.WithoutCancel(ctx), e.cleanup())
			if err := remote.Close(cctx); err != nil {
				*into = append(*into, fmt.Sprintf("the Run's remote: %v", err))
			}
			ccancel()
			remote = nil
		}
		if locked {
			locked = false
			if err := e.leave(ctx, req.Target, nonce); err != nil {
				*into = append(*into, fmt.Sprintf("the Run's entry in the lock of %s stays until %s: %v", req.Target, until.UTC().Format(time.RFC3339), err))
			}
		}
	}
	defer release(&r.Cleanup)

	mode := runremote.Apply
	if req.Mode == agentport.Verify {
		mode = runremote.Verify
	}
	plan, err := runremote.PlanFor(req.Run, mode, start, cur, imp)
	if err != nil {
		return nil, err
	}
	creds, err := e.credentials(plan, req.Mode)
	if err != nil {
		return nil, err
	}
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("the Run's plan: %w", err)
	}

	var bundle bytes.Buffer
	if err := e.Git.Bundle(ctx, start, &bundle); err != nil {
		return nil, err
	}
	started, err := e.Remotes.Start(ctx, plan, &bundle)
	if err != nil {
		r.State = StateIncomplete
		r.Why = fmt.Sprintf("the Run's remote did not start: %v", err)
		r.Cleanup = append(r.Cleanup, "the Run's remote may remain if its start failed half-way; Sweep removes it")
		return e.finish(r), nil
	}
	remote = started

	s := newSchedule(e, req, plan, cur, imp, started, creds)
	if req.Mode == agentport.Verify && len(plan.Checks) > 0 {
		// The checkers judge the start with the required checks' results
		// as evidence, so the host runs them before the checkers start.
		s.startChecks = e.checkStart(ctx, started, start)
	}
	s.run(ctx)
	s.report(r)
	if r.State != "" {
		// A stopped Run still reports honestly, bounded on its own, with
		// what its agents pushed last.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), e.cleanup())
		defer cancel()
		if tips, err := s.sync(ctx); err == nil {
			s.tips = tips
		} else {
			r.Cleanup = append(r.Cleanup, fmt.Sprintf("the Run's remote could not be read after the stop, so the Run branch may miss its last pushes: %v", err))
		}
	}
	e.collectChecks(ctx, started, s, r)

	switch {
	case r.State != "":
		// The schedule stopped the Run: a model defect, a budget, a
		// deadline or no progress. Its work stays on the Run branch.
		release(&r.Cleanup)
		e.keep(ctx, r, s)
	case req.Mode == agentport.Verify:
		var found []string
		if n := len(r.Findings); n > 0 {
			found = append(found, fmt.Sprintf("%d Findings", n))
		}
		if n := len(r.OwnerDecisions); n > 0 {
			found = append(found, fmt.Sprintf("%d decisions for the owner", n))
		}
		switch rc := r.RequiredChecks; {
		case len(s.plan.Checks) == 0 || rc.Ran:
		case len(rc.Checks) > 0:
			found = append(found, "required checks that did not pass on the whole repository")
		default:
			found = append(found, "required checks whose results could not be confirmed")
		}
		if len(found) > 0 {
			r.State = StateFindings
			r.Why = "verify found " + strings.Join(found, " and ")
		} else {
			r.State = StateVerified
			r.Why = "every checklist item conforms and the required checks pass on the whole repository"
			if len(s.plan.Checks) == 0 {
				r.Why = "every checklist item conforms"
			}
		}
		if isFullCheck(r) {
			// This Run is the full check: it examined its start.
			r.FullCheck = FullCheck{Since: r.Start}
		}
		release(&r.Cleanup)
		e.keep(ctx, r, s)
	default:
		e.land(ctx, r, s, started, release)
	}
	return e.finish(r), nil
}

// prepare reads the tip of the target, the base and the impact into the
// Result. A start that does not compile or a change that reaches no Manager
// sets the Result's state, and no agent starts.
func (e *Engine) prepare(ctx context.Context, req Request, r *Result) (string, *project.Project, analysis.Impact, error) {
	var imp analysis.Impact
	start, err := e.Git.Fetch(ctx, req.Target)
	if err != nil {
		return "", nil, imp, fmt.Errorf("%w: the target branch %q: %v", ErrUsage, req.Target, err)
	}
	base := req.Base
	if base == "" {
		landed, ok, err := e.lastBase(ctx, start)
		switch {
		case err != nil:
			return "", nil, imp, err
		case ok:
			base = landed
		case req.Full:
			base = start
		default:
			return "", nil, imp, fmt.Errorf("%w: no Run has landed on %s yet, so there is no default base; name one with --base, or cover everything with --full", ErrUsage, req.Target)
		}
	}
	bc, err := e.Git.Commit(ctx, base)
	if err != nil {
		return "", nil, imp, fmt.Errorf("%w: base %q is no commit: %v", ErrUsage, base, err)
	}
	base = bc.ID
	r.Base, r.Start = base, start
	r.FullCheck = e.fullCheck(ctx, req.Target, start)

	cur, err := project.Load(e.Git.State(ctx, start))
	if err != nil {
		return "", nil, imp, err
	}
	if len(cur.Errors) > 0 {
		r.State = StateCheckFailed
		r.Why = fmt.Sprintf("check fails at %s with %d compile errors, so no agent starts; the first: %s", short(start), len(cur.Errors), cur.Errors[0].String())
		return start, cur, imp, nil
	}
	before, err := project.Load(e.Git.State(ctx, base))
	if err != nil {
		return "", nil, imp, err
	}
	all, err := e.Git.Changed(ctx, base, start)
	if err != nil {
		return "", nil, imp, err
	}
	var changed []analysis.Change
	for _, c := range all {
		if !strings.HasPrefix(c.Path, project.Dir+"/") {
			changed = append(changed, c)
		}
	}
	imp = analysis.ComputeImpact(analysis.ImpactInput{Base: before, Current: cur, Changed: changed, Findings: req.Findings, Full: req.Full})
	r.ImpactNotes, r.impact = imp.Notes, impactKeys(imp)
	working := map[string]bool{}
	for _, pkg := range imp.Packages {
		working[pkg] = true
	}
	r.NotParticipating = []NotParticipant{}
	for _, pkg := range cur.Model.SortedPackages() {
		if !working[pkg] {
			r.NotParticipating = append(r.NotParticipating, NotParticipant{Package: runremote.Name(pkg), Reason: "the change reaches nothing it defines or owns"})
		}
	}
	if len(imp.Packages) == 0 {
		r.State = StateNothing
		r.Why = fmt.Sprintf("nothing changed between %s and %s that reaches a Manager; no agent starts", short(base), short(start))
	}
	return start, cur, imp, nil
}

// done ends a Run that did not start its remote: with an error when it
// could not run, else with its Result.
func (e *Engine) done(r *Result, err error) (*Result, error) {
	if err != nil {
		return nil, err
	}
	return e.finish(r), nil
}

// finish completes the Result once the Run returns. A Run that wrote its
// page keeps the end time stamped there, so the page and the Result agree.
func (e *Engine) finish(r *Result) *Result {
	if r.Ended.IsZero() {
		return e.stamp(r)
	}
	return r
}

// stamp sets the Result's end time and usage, as the page it is about to
// write states them.
func (e *Engine) stamp(r *Result) *Result {
	r.Ended = e.now()
	r.Usage = agentport.Usage{}
	for _, st := range r.Starts {
		r.Usage.InputTokens += st.Usage.InputTokens
		r.Usage.OutputTokens += st.Usage.OutputTokens
		r.Usage.Partial = r.Usage.Partial || st.Usage.Partial
	}
	return r
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

func (e *Engine) token() (string, error) {
	if e.Token != nil {
		return e.Token()
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// credential is one agent's user and token.
type credential struct{ user, token string }

// credentials gives every working Manager a maker (in apply) and a checker,
// each with its own token; the plan keeps only the tokens' hashes.
func (e *Engine) credentials(plan *runremote.Plan, mode agentport.Mode) (map[string]map[agentport.Role]credential, error) {
	creds := map[string]map[agentport.Role]credential{}
	roles := []agentport.Role{agentport.Maker, agentport.Checker}
	if mode == agentport.Verify {
		roles = roles[1:]
	}
	for _, m := range plan.Managers {
		if !m.Working {
			continue
		}
		creds[m.ID] = map[agentport.Role]credential{}
		for _, role := range roles {
			token, err := e.token()
			if err != nil {
				return nil, err
			}
			user := strings.TrimPrefix(strings.ReplaceAll(runremote.Branch(m.ID), "/", "."), "managers.") + "-" + string(role)
			sum := sha256.Sum256([]byte(token))
			plan.Agents = append(plan.Agents, runremote.Agent{User: user, Manager: m.ID, Role: runremote.Role(role), TokenSHA256: hex.EncodeToString(sum[:])})
			creds[m.ID][role] = credential{user, token}
		}
	}
	return creds, nil
}

// checkStart runs the required checks on the Run's start, a commit the
// remote holds, as the host's check. Checks that could not run get one more
// try, since a Run whose items all conform would otherwise end with
// unconfirmed checks.
func (e *Engine) checkStart(ctx context.Context, remote Remote, start string) *startChecks {
	var sc *startChecks
	for try := 0; try < 2 && (sc == nil || sc.err != nil) && ctx.Err() == nil; try++ {
		var b bytes.Buffer
		if err := e.Git.Bundle(ctx, start, &b); err != nil {
			sc = &startChecks{err: err}
			continue
		}
		results, err := remote.Check(ctx, &b, start)
		sc = &startChecks{results: results, err: err}
	}
	if sc == nil {
		sc = &startChecks{err: context.Cause(ctx)}
	}
	return sc
}

// collectChecks records where the required checks ran on the whole
// repository: in apply on the push of Root's approved submission, in verify
// on the Run's start, where the host ran them before the checkers started.
// The hooks ran them on that commit's full repository state but refused only
// a failing merged state, so landing refuses failing results on an unmoved
// target and runs them again on the merged state of a moved one.
func (e *Engine) collectChecks(ctx context.Context, remote Remote, s *schedule, r *Result) {
	rc := &r.RequiredChecks
	if len(s.plan.Checks) == 0 {
		rc.Note = "execution.yaml names no required checks"
		return
	}
	if s.req.Mode == agentport.Verify {
		sc := s.startChecks
		*rc = RequiredChecks{Commit: s.plan.Base, Checks: sc.results, WholeRepository: true,
			Ran: sc.err == nil && len(sc.results) == len(s.plan.Checks) && allPassed(sc.results)}
		if rc.Checks == nil {
			rc.Checks = []runremote.CheckResult{}
		}
		if sc.err != nil {
			rc.Note = fmt.Sprintf("the required checks could not run on the start: %v", sc.err)
		}
		return
	}
	root := s.nodes[model.RootPackage]
	if root == nil || root.approvedSubmission == "" {
		rc.Note = "Root's work was not approved, so the required checks ran on no state that would land"
		return
	}
	pushes, err := remote.Log(ctx)
	if err != nil {
		rc.Note = fmt.Sprintf("the remote's decision log could not be read: %v", err)
		return
	}
	for _, p := range pushes {
		for _, u := range p.Updates {
			if p.Decision.Accepted && u.New == root.approvedSubmission && len(p.Decision.Checks) > 0 {
				rc.Commit, rc.Checks = u.New, p.Decision.Checks
				rc.Ran, rc.WholeRepository = allPassed(p.Decision.Checks) && len(p.Decision.Checks) == len(s.plan.Checks), true
			}
		}
	}
	if rc.Commit == "" {
		rc.Note = fmt.Sprintf("the decision log holds no checked push of Root's approved submission %s", short(root.approvedSubmission))
	}
}

func allPassed(checks []runremote.CheckResult) bool {
	for _, c := range checks {
		if !c.Passed {
			return false
		}
	}
	return true
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
