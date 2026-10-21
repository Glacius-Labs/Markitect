package run

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var nameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

const (
	maxTurnFailures = 2
	maxEmptyTurns   = 2
	maxToolText     = 16 << 10
	maxToolItems    = 64
	outputTail      = 8 << 10
	maxListed       = 40
	maxListedNames  = 20
)

// mgr is the run-wide state of one Manager. Every field is guarded by
// engine.mu.
type mgr struct {
	Manager
	slug string
	// workHome and conversation hold the Manager's work conversation, which
	// continues across the Execute Passes of the run.
	workHome     string
	conversation string
	// introduced is set once a work turn that carried the Manager's standing
	// context ended without error. rehydrate says that its work conversation
	// may have lost that context: the provider compacted it or did not continue
	// it; the next work turn carries the standing context again.
	introduced bool
	rehydrate  bool
	failures   int
	disabled   bool
}

type engine struct {
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc
	base   context.Context

	stopMu sync.Mutex
	status Status
	reason string
	// failure is a definite error of the configuration or a port.
	failure error

	evMu  sync.Mutex
	evErr error

	mu        sync.Mutex
	cond      *sync.Cond
	wg        sync.WaitGroup
	team      []*mgr
	byPath    map[string]*mgr
	findings  []*Finding
	questions []*Question
	problems  []*Problem
	receipts  []*ProbeReceipt
	turns     []TurnRecord
	usage     Usage
	turnCount int
	passes    []*pass
	cur       *pass
	workID    WorkspaceID
	work      string
	// stand is the last sealed whole stand: the start, then each Pass's exit.
	stand snapshot.Tree
	// plans holds the accepted plan results of the latest planning Pass.
	plans map[string]*proposal
	// verified is the candidate of the last Pass when it was a clean review.
	verified *snapshot.Tree
	homes    []string
	homeSet  map[string]bool
	closing  bool

	nextFinding, nextQuestion, nextProposal, nextProblem, nextReceipt int
}

func slugOf(path string) string {
	if path == "." {
		return "root"
	}
	return strings.ReplaceAll(path, "/", "-")
}

func validate(cfg Config) error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
	}
	checkName := func(what, name string) error {
		if !nameRE.MatchString(name) {
			return bad("%s %q is not a valid name", what, name)
		}
		return nil
	}
	if err := checkName("run ID", cfg.RunID); err != nil {
		return err
	}
	for _, name := range []string{cfg.RunID + "-work", cfg.RunID + "-p9999-c"} {
		if err := checkName("generated name", name); err != nil {
			return err
		}
	}
	if len(cfg.Team) == 0 || cfg.Team[0].Path != "." || cfg.Team[0].Parent != "" {
		return bad("the team must start with Root \".\" without a parent")
	}
	index := map[string]int{}
	slugs := map[string]string{}
	files := map[string]bool{}
	for _, entry := range cfg.Start.Entries() {
		files[entry.Path().String()] = true
	}
	for i, m := range cfg.Team {
		if !identifier(m.Path) || m.Path != strings.TrimSpace(m.Path) || strings.ContainsAny(m.Path, "\\") {
			return bad("manager path %q is invalid", m.Path)
		}
		if _, dup := index[m.Path]; dup {
			return bad("manager %q is listed twice", m.Path)
		}
		index[m.Path] = i
		slug := slugOf(m.Path)
		if other, dup := slugs[slug]; dup {
			return bad("managers %q and %q share the name %q", other, m.Path, slug)
		}
		slugs[slug] = m.Path
		for _, name := range []string{cfg.RunID + "-p9999-" + slug + "-m99999", cfg.RunID + "-p9999-" + slug + "-r99999", cfg.RunID + "-p9999-" + slug + "-h99999-home", cfg.RunID + "-p9999-" + slug + "-r", cfg.RunID + "-" + slug + "-m"} {
			if err := checkName("generated name", name); err != nil {
				return err
			}
		}
		if m.ExecuteProfile == "" || m.VerifyProfile == "" {
			return bad("manager %q needs an Execute and a Verify profile", m.Path)
		}
		if err := validatePlan(m, files); err != nil {
			return bad("%v", err)
		}
		seen := map[string]bool{}
		for _, id := range m.Obligations {
			if !identifier(id) || seen[id] {
				return bad("manager %q has an empty or repeated obligation %q", m.Path, id)
			}
			seen[id] = true
		}
		roles := map[string]bool{}
		for _, h := range m.Helpers {
			if !nameRE.MatchString(h.Role) || roles[h.Role] || h.Profile == "" || h.Role == "manager" || h.Role == "reviewer" {
				return bad("manager %q has an invalid, repeated or reserved helper role %q, or one without a profile", m.Path, h.Role)
			}
			roles[h.Role] = true
		}
		if m.MaxHelpers < 0 || len(m.Helpers) > 0 && m.MaxHelpers == 0 {
			return bad("manager %q needs a positive helper bound for its helper roles", m.Path)
		}
		if i == 0 {
			continue
		}
		if m.Parent == m.Path {
			return bad("manager %q is its own parent", m.Path)
		}
		parent, ok := index[m.Parent]
		if !ok {
			return bad("manager %q has parent %q that is not listed before it", m.Path, m.Parent)
		}
		if !slices.Contains(cfg.Team[parent].Children, m.Path) {
			return bad("parent %q does not list child %q", m.Parent, m.Path)
		}
	}
	for _, m := range cfg.Team {
		for _, c := range m.Children {
			i, ok := index[c]
			if c == m.Path || !ok || cfg.Team[i].Parent != m.Path {
				return bad("child %q of %q is not a listed manager with that parent", c, m.Path)
			}
		}
	}
	names := map[string]bool{}
	for _, c := range cfg.Checks {
		if !nameRE.MatchString(c.Name) || names[c.Name] {
			return bad("check name %q is invalid or repeated", c.Name)
		}
		names[c.Name] = true
		if _, ok := index[c.Owner]; !ok {
			return bad("check %q has unknown owner %q", c.Name, c.Owner)
		}
		req := c.Request
		req.ID = CheckID(cfg.RunID + "-p9999-" + c.Name)
		if !req.Valid() {
			return bad("check %q has an incomplete request", c.Name)
		}
		if c.ProfileArgs < 0 || c.ProfileArgs > len(req.Args) {
			return bad("check %q has %d profile arguments of %d", c.Name, c.ProfileArgs, len(req.Args))
		}
	}
	b := cfg.Budget
	if b.Deadline.IsZero() || b.TurnTimeout <= 0 || b.AgentTurns <= 0 || b.Passes <= 0 {
		return bad("the budget needs a deadline, a turn timeout, agent turns and passes")
	}
	if !(HarvestRequest{ID: WorkspaceID(cfg.RunID + "-work"), MaxEntries: cfg.MaxEntries, MaxBytes: cfg.MaxBytes}).Valid() {
		return bad("harvest bounds are outside the Runtime ceilings")
	}
	switch {
	case cfg.ModelCheck == nil || cfg.FileCheck == nil || cfg.OwnerOf == nil || cfg.Registry == nil || cfg.Definition == nil:
		return bad("ModelCheck, FileCheck, OwnerOf, Registry and Definition are required")
	case cfg.Agents == nil || cfg.Workspaces == nil || cfg.Evidence == nil:
		return bad("Agents, Workspaces and Evidence are required")
	case len(cfg.Checks) > 0 && cfg.Runner == nil:
		return bad("checks need a Runner")
	}
	return nil
}

// questionNumber returns n of a question identity "Qn" (n >= 1), or 0.
func questionNumber(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "Q"))
	if err != nil || n < 1 || id != fmt.Sprintf("Q%d", n) {
		return 0
	}
	return n
}

// validatePlan checks the mounts of a write plan: clean relative paths of the
// shared tree without duplicates, none of them "."; a file mount names a file
// of the start stand and a directory mount no file and nothing below a file;
// every mount comes before the mounts of its descendants, so that a later mount
// never shadows an earlier one.
func validatePlan(m Manager, files map[string]bool) error {
	seen := map[string]bool{}
	for i, mount := range m.Write.Mounts {
		if _, err := snapshot.NewPath(mount.Path); err != nil {
			return fmt.Errorf("manager %q has a mount %q that is not a clean relative path below /work", m.Path, mount.Path)
		}
		if seen[mount.Path] {
			return fmt.Errorf("manager %q mounts %q twice", m.Path, mount.Path)
		}
		seen[mount.Path] = true
		if mount.File {
			if !files[mount.Path] {
				return fmt.Errorf("manager %q mounts the file %q, which the start stand does not hold", m.Path, mount.Path)
			}
		} else {
			for end := 0; end <= len(mount.Path); end++ {
				if (end == len(mount.Path) || mount.Path[end] == '/') && files[mount.Path[:end]] {
					return fmt.Errorf("manager %q mounts the directory %q, but %q is a file of the start stand", m.Path, mount.Path, mount.Path[:end])
				}
			}
		}
		for _, later := range m.Write.Mounts[i+1:] {
			if strings.HasPrefix(mount.Path, later.Path+"/") {
				return fmt.Errorf("manager %q lists mount %q after its descendant %q; parents come first", m.Path, later.Path, mount.Path)
			}
		}
	}
	return nil
}

func newEngine(parent context.Context, cfg Config) *engine {
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	ctx, cancel := context.WithDeadline(parent, cfg.Budget.Deadline)
	e := &engine{cfg: cfg, ctx: ctx, cancel: cancel, base: context.WithoutCancel(parent), byPath: map[string]*mgr{}, homeSet: map[string]bool{}, stand: cfg.Start}
	e.cond = sync.NewCond(&e.mu)
	for _, m := range cfg.Team {
		slug := slugOf(m.Path)
		x := &mgr{Manager: m, slug: slug, workHome: cfg.RunID + "-" + slug + "-m"}
		e.team = append(e.team, x)
		e.byPath[m.Path] = x
	}
	// Wake waiting loops when the run context ends.
	context.AfterFunc(ctx, func() {
		e.mu.Lock()
		e.cond.Broadcast()
		e.mu.Unlock()
	})
	return e
}

func (e *engine) now() time.Time { return e.cfg.Clock() }

// halt stops the run Incomplete with the first reason given and cancels all
// running work.
func (e *engine) halt(reason string) { e.stop(Incomplete, reason, nil) }

// reject stops the run Failed: a definite rejection such as a failed gate or
// Root's model-error decision.
func (e *engine) reject(reason string) { e.stop(Failed, reason, nil) }

// fail stops the run Failed with a definite error of its inputs or ports.
func (e *engine) fail(err error) { e.stop(Failed, err.Error(), err) }

func (e *engine) stop(status Status, reason string, err error) {
	e.stopMu.Lock()
	if e.status == "" {
		e.status, e.reason, e.failure = status, reason, err
	}
	e.stopMu.Unlock()
	e.cancel()
}

// portError reports a port failure unless the run was already stopped.
func (e *engine) portError(what string, err error) {
	if e.ctx.Err() != nil {
		return
	}
	e.fail(fmt.Errorf("%s: %w", what, err))
}

// stopped returns the status and reason of a stopped run, or empty ones.
func (e *engine) stopped() (Status, string, error) {
	e.stopMu.Lock()
	defer e.stopMu.Unlock()
	switch {
	case e.status != "":
		return e.status, e.reason, e.failure
	case errors.Is(e.ctx.Err(), context.DeadlineExceeded):
		return Incomplete, "the deadline was reached", nil
	case e.ctx.Err() != nil:
		return Incomplete, "the run was cancelled", nil
	}
	return "", "", nil
}

// record appends one event to the evidence. A failure stops the run, and no
// further event is attempted.
func (e *engine) record(ev Event) error {
	e.evMu.Lock()
	defer e.evMu.Unlock()
	if e.evErr != nil {
		return e.evErr
	}
	if ev.At.IsZero() {
		ev.At = e.now()
	}
	if err := e.cfg.Evidence.Record(ev); err != nil {
		e.evErr = fmt.Errorf("evidence could not be recorded: %w", err)
		e.halt(e.evErr.Error())
		return e.evErr
	}
	return nil
}

func (e *engine) evidenceFailed() error {
	e.evMu.Lock()
	defer e.evMu.Unlock()
	return e.evErr
}

func (e *engine) closeVolume(id WorkspaceID) {
	obs, err := e.cfg.Workspaces.Close(e.base, id)
	data := map[string]any{"workspace": id, "closed": obs.Closed, "uncertain": obs.Uncertain}
	if err != nil {
		data["error"] = err.Error()
	}
	if obs.Reason != "" {
		data["reason"] = obs.Reason
	}
	_ = e.record(Event{Kind: "workspace.close", Data: data})
}

// materialize writes an exact tree into a new volume and confirms it.
func (e *engine) materialize(id WorkspaceID, tree snapshot.Tree, directories []string) (string, bool) {
	obs, err := e.cfg.Workspaces.Materialize(e.ctx, WorkspaceSpec{ID: id, Tree: tree, Directories: directories})
	if err == nil && (obs.Uncertain || obs.Volume == "" || obs.Tree != tree.Digest()) {
		err = fmt.Errorf("materialization was not confirmed: %s", obs.Reason)
	}
	if err != nil {
		e.portError("materialize "+string(id), err)
		return "", false
	}
	_ = e.record(Event{Kind: "workspace.materialize", Data: map[string]any{"workspace": id, "volume": obs.Volume, "tree": tree.Digest().String(), "entries": tree.EntryCount()}})
	return obs.Volume, e.ctx.Err() == nil
}

// mountDirectories returns the sorted union of the directory mounts of all
// write plans: the directories the shared work volume needs before the first
// turn.
func (e *engine) mountDirectories() []string {
	set := map[string]bool{}
	for _, m := range e.team {
		for _, mount := range m.Write.Mounts {
			if !mount.File {
				set[mount.Path] = true
			}
		}
	}
	return slices.Sorted(func(yield func(string) bool) {
		for dir := range set {
			if !yield(dir) {
				return
			}
		}
	})
}

// open records the start and materializes the shared work volume.
func (e *engine) open() {
	cfg := e.cfg
	managers := make([]string, 0, len(e.team))
	for _, m := range e.team {
		managers = append(managers, m.Path)
	}
	if e.record(Event{Kind: "run.start", Data: map[string]any{"run": cfg.RunID, "binding": cfg.Binding, "managers": managers, "start": cfg.Start.Digest().String(), "agentTurns": cfg.Budget.AgentTurns, "passes": cfg.Budget.Passes, "deadline": cfg.Budget.Deadline}}) != nil {
		return
	}
	if cfg.Budget.AgentTurns < 2*len(e.team) {
		e.halt(fmt.Sprintf("the agent turn budget of %d does not cover a basis and a result of each of the %d managers in one Pass", cfg.Budget.AgentTurns, len(e.team)))
		return
	}
	e.workID = WorkspaceID(cfg.RunID + "-work")
	volume, ok := e.materialize(e.workID, cfg.Start, e.mountDirectories())
	if !ok {
		// The volume never held the start stand: nobody else owns it.
		e.closeVolume(e.workID)
		return
	}
	e.mu.Lock()
	e.work = volume
	e.mu.Unlock()
}

// reserveTurnLocked counts one turn when the budget leaves room for it.
func (e *engine) reserveTurnLocked() bool {
	if e.turnCount >= e.cfg.Budget.AgentTurns {
		return false
	}
	e.turnCount++
	return true
}

// useHomeLocked records a provider home a turn uses, so the caller removes it.
func (e *engine) useHomeLocked(home string) {
	if !e.homeSet[home] {
		e.homeSet[home] = true
		e.homes = append(e.homes, home)
	}
}

// finish ends the run and returns its outcome.
func (e *engine) finish(reason string) (Outcome, error) {
	e.mu.Lock()
	e.closing = true
	e.mu.Unlock()
	status, why, failure := e.stopped()
	e.cancel()
	e.wg.Wait()
	e.mu.Lock()
	var candidate snapshot.Tree
	switch {
	case status != "":
	case e.verified != nil:
		status, candidate = Verified, *e.verified
	default:
		status, why = Incomplete, reason
		if why == "" {
			why = "the run ended without a clean independent Verify"
		}
	}
	e.mu.Unlock()
	if status == Verified && e.evidenceFailed() != nil {
		status, why, candidate = Incomplete, e.evidenceFailed().Error(), snapshot.Tree{}
	}
	out := e.outcome(status, why, candidate)
	if err := e.record(Event{Kind: "outcome", Data: map[string]any{"status": status, "reason": why}}); err != nil && status == Verified {
		out.Status, out.Reason, out.Candidate = Incomplete, err.Error(), snapshot.Tree{}
	}
	if out.Status == Failed && failure != nil {
		return out, failure
	}
	return out, nil
}

func (e *engine) outcome(status Status, reason string, tree snapshot.Tree) Outcome {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := Outcome{Status: status, Reason: reason, Candidate: tree, Turns: slices.Clone(e.turns), Usage: e.usage, Homes: slices.Clone(e.homes)}
	for _, p := range e.passes {
		out.Passes = append(out.Passes, p.recordLocked())
	}
	for _, f := range e.findings {
		out.Findings = append(out.Findings, *f)
	}
	for _, q := range e.questions {
		out.Questions = append(out.Questions, *q)
	}
	for _, p := range e.problems {
		out.Problems = append(out.Problems, *p)
	}
	for _, r := range e.receipts {
		out.Probes = append(out.Probes, *r)
	}
	return out
}

// within reports whether owner is the manager or one of its descendants.
func (e *engine) within(owner, manager string) bool {
	for m := e.byPath[owner]; m != nil; m = e.byPath[m.Parent] {
		if m.Path == manager {
			return true
		}
	}
	return false
}

func (e *engine) findingLocked(id string) *Finding {
	for _, f := range e.findings {
		if f.ID == id {
			return f
		}
	}
	return nil
}

func (e *engine) questionLocked(id string) *Question {
	for _, q := range e.questions {
		if q.ID == id {
			return q
		}
	}
	return nil
}

// addFindingLocked records a new finding of the current Pass.
func (e *engine) addFindingLocked(f Finding) *Finding {
	e.nextFinding++
	f.ID = fmt.Sprintf("F%d", e.nextFinding)
	if e.cur != nil {
		f.Pass, f.Review = e.cur.id, e.cur.spec.Review
		if f.Review {
			f.Candidate = e.cur.tree.Digest().String()
		}
	}
	p := &f
	e.findings = append(e.findings, p)
	_ = e.record(Event{Kind: "finding.recorded", Pass: f.Pass, Manager: f.Owner, Data: *p})
	return p
}

// corrections returns the distinct correction purposes of the open genuine
// findings; with work, only of the findings of work Passes that work Passes
// judge and close.
func (e *engine) corrections(work bool) []Purpose {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []Purpose
	for _, f := range e.findings {
		if work && (f.Review || f.RaisedBy == MarkitectFinding) {
			continue
		}
		if f.Blocking && !f.Closed && f.Correction != "" && !slices.Contains(out, f.Correction) {
			out = append(out, f.Correction)
		}
	}
	return out
}

// addProblemLocked records a structured problem report.
func (e *engine) addProblemLocked(p Problem) *Problem {
	e.nextProblem++
	p.ID = fmt.Sprintf("X%d", e.nextProblem)
	if e.cur != nil && p.Pass == "" {
		p.Pass = e.cur.id
	}
	x := &p
	e.problems = append(e.problems, x)
	_ = e.record(Event{Kind: "problem.raised", Pass: p.Pass, Manager: p.Manager, Data: *x})
	return x
}
