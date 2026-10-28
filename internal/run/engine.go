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
	// Rounds is the number of maker starts, and of checker starts, one
	// Manager may make.
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
		req.Run = "r" + now.UTC().Format("20060102-150405")
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

	start, err := e.Git.Fetch(ctx, req.Target)
	if err != nil {
		return nil, fmt.Errorf("%w: the target branch %q: %v", ErrUsage, req.Target, err)
	}
	base := req.Base
	if base == "" {
		landed, ok, err := e.Git.LastLanded(ctx, start)
		switch {
		case err != nil:
			return nil, err
		case ok:
			base = landed
		case req.Full:
			base = start
		default:
			return nil, fmt.Errorf("%w: no Run has landed on %s yet, so there is no default base; name one with --base, or cover everything with --full", ErrUsage, req.Target)
		}
	}
	bc, err := e.Git.Commit(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("%w: base %q is no commit: %v", ErrUsage, base, err)
	}
	base = bc.ID
	r.Base, r.Start = base, start

	cur, err := project.Load(e.Git.State(ctx, start))
	if err != nil {
		return nil, err
	}
	if len(cur.Errors) > 0 {
		r.State = StateCheckFailed
		r.Why = fmt.Sprintf("check fails at %s with %d compile errors, so no agent starts; the first: %s", short(start), len(cur.Errors), cur.Errors[0].String())
		return e.finish(r), nil
	}
	before, err := project.Load(e.Git.State(ctx, base))
	if err != nil {
		return nil, err
	}
	all, err := e.Git.Changed(ctx, base, start)
	if err != nil {
		return nil, err
	}
	var changed []analysis.Change
	for _, c := range all {
		if !strings.HasPrefix(c.Path, project.Dir+"/") {
			changed = append(changed, c)
		}
	}
	imp := analysis.ComputeImpact(analysis.ImpactInput{Base: before, Current: cur, Changed: changed, Findings: req.Findings, Full: req.Full})
	r.ImpactNotes = imp.Notes
	working := map[string]bool{}
	for _, pkg := range imp.Packages {
		working[pkg] = true
	}
	for _, pkg := range cur.Model.SortedPackages() {
		if !working[pkg] {
			r.NotParticipating = append(r.NotParticipating, NotParticipant{Package: runremote.Name(pkg), Reason: "the change reaches nothing it defines or owns"})
		}
	}
	if len(imp.Packages) == 0 {
		r.State = StateNothing
		r.Why = fmt.Sprintf("nothing changed between %s and %s that reaches a Manager; no agent starts", short(base), short(start))
		return e.finish(r), nil
	}

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
	remote, err := e.Remotes.Start(ctx, plan, &bundle)
	if err != nil {
		r.State = StateIncomplete
		r.Why = fmt.Sprintf("the Run's remote did not start: %v", err)
		r.Cleanup = append(r.Cleanup, "the Run's remote may remain if its start failed half-way; Sweep removes it")
		return e.finish(r), nil
	}
	defer func() {
		cctx, ccancel := context.WithTimeout(context.WithoutCancel(ctx), e.cleanup())
		defer ccancel()
		if err := remote.Close(cctx); err != nil {
			r.Cleanup = append(r.Cleanup, fmt.Sprintf("the Run's remote: %v", err))
		}
	}()

	s := newSchedule(e, req, plan, cur, imp, remote, creds)
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
	e.collectChecks(ctx, remote, s, r)

	switch {
	case r.State != "":
		// The schedule stopped the Run: a model defect, a budget, a
		// deadline or no progress. Its work stays on the Run branch.
		e.keep(ctx, r, s)
	case req.Mode == agentport.Verify:
		var found []string
		if n := len(r.Findings); n > 0 {
			found = append(found, fmt.Sprintf("%d Findings", n))
		}
		if n := len(r.OwnerDecisions); n > 0 {
			found = append(found, fmt.Sprintf("%d decisions for the owner", n))
		}
		if len(s.plan.Checks) > 0 && !r.RequiredChecks.Ran {
			found = append(found, "required checks that did not pass on the whole repository")
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
		e.keep(ctx, r, s)
	default:
		e.land(ctx, r, s, remote)
	}
	return e.finish(r), nil
}

func (e *Engine) finish(r *Result) *Result {
	r.Ended = e.now()
	r.Usage = agentport.Usage{}
	for _, st := range r.Starts {
		r.Usage.InputTokens += st.Usage.InputTokens
		r.Usage.OutputTokens += st.Usage.OutputTokens
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

// collectChecks records where the required checks ran on the whole
// repository: in apply on the push of Root's approved submission, in verify
// on the push of Root's verdicts, whose state is the Run's starting commit.
// The hooks ran them on that commit's full repository state; landing on a
// moved target runs them again on the merged state.
func (e *Engine) collectChecks(ctx context.Context, remote Remote, s *schedule, r *Result) {
	rc := &r.RequiredChecks
	if len(s.plan.Checks) == 0 {
		rc.Note = "execution.yaml names no required checks"
		return
	}
	root := s.nodes[model.RootPackage]
	if root == nil || root.approvedSubmission == "" {
		rc.Note = "Root's work was not approved, so the required checks ran on no state that would land"
		if s.req.Mode == agentport.Verify {
			rc.Note = "Root's checker pushed no verdicts, so the required checks did not run on the whole repository"
		}
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
		what := "approved submission"
		if s.req.Mode == agentport.Verify {
			what = "verdicts"
		}
		rc.Note = fmt.Sprintf("the decision log holds no checked push of Root's %s %s", what, short(root.approvedSubmission))
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
