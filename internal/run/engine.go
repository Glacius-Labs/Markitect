package run

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var nameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

// ownerPollInterval is how often the engine asks the owner channel for answers.
var ownerPollInterval = 3 * time.Second

const (
	maxExecuteFailures = 2
	maxVerifyAttempts  = 2
	maxToolText        = 16 << 10
	maxToolFindings    = 64
	outputTail         = 8 << 10
	maxListedPaths     = 40
)

// candidate is one immutable stand with its read-only volume.
type candidate struct {
	index  int
	tree   snapshot.Tree
	id     WorkspaceID
	volume string
}

// mgr is the engine's state of one Manager. Every field is guarded by engine.mu.
type mgr struct {
	Manager
	slug         string
	execHome     string
	verifyHome   string
	conversation string
	items        []string
	consumed     []string
	running      bool
	disabled     bool
	failures     int
	execCount    int
	execUsed     bool
	verifyUsed   bool
	ran          bool
	changed      []string
}

type roundState struct {
	n           int
	tree        snapshot.Tree
	volume      string
	checks      []CheckResult
	structural  []*Finding
	assessments map[string]Assessment
	assessed    map[string]bool
}

type engine struct {
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc
	base   context.Context

	stopMu  sync.Mutex
	reason  string
	failure error

	evMu  sync.Mutex
	evErr error

	mu          sync.Mutex
	cond        *sync.Cond
	wg          sync.WaitGroup
	team        []*mgr
	byPath      map[string]*mgr
	findings    []*Finding
	delivered   map[string]int
	questions   []*Question
	turns       []TurnRecord
	usage       Usage
	turnCount   int
	rounds      []Round
	rs          *roundState
	execRound   int
	work        string
	executeOpen bool
	inPhase     bool
	closing     bool
	running     int
	polling     bool
	nextFinding int
	nextQuest   int
}

func slugOf(path string) string {
	if path == "." {
		return "root"
	}
	return strings.ReplaceAll(path, "/", "-")
}

func (e *engine) volumeID(suffix string) WorkspaceID { return WorkspaceID(e.cfg.RunID + "-" + suffix) }

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
	if err := checkName("work volume", cfg.RunID+"-work"); err != nil {
		return err
	}
	if len(cfg.Team) == 0 || cfg.Team[0].Path != "." || cfg.Team[0].Parent != "" {
		return bad("the team must start with Root \".\" without a parent")
	}
	index := map[string]int{}
	slugs := map[string]string{}
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
		for _, name := range []string{cfg.RunID + "-r99999-" + slug + "-x99999", cfg.RunID + "-r99999-" + slug + "-v99999", cfg.RunID + "-" + slug + "-x", cfg.RunID + "-" + slug + "-v"} {
			if err := checkName("generated name", name); err != nil {
				return err
			}
		}
		if m.ExecuteProfile == "" || m.VerifyProfile == "" {
			return bad("manager %q needs an Execute and a Verify profile", m.Path)
		}
		if i == 0 {
			continue
		}
		parent, ok := index[m.Parent]
		if !ok {
			return bad("manager %q has parent %q that is not listed before it", m.Path, m.Parent)
		}
		found := false
		for _, c := range cfg.Team[parent].Children {
			found = found || c == m.Path
		}
		if !found {
			return bad("parent %q does not list child %q", m.Parent, m.Path)
		}
	}
	for _, m := range cfg.Team {
		for _, c := range m.Children {
			i, ok := index[c]
			if !ok || cfg.Team[i].Parent != m.Path {
				return bad("child %q of %q is not a listed manager with that parent", c, m.Path)
			}
		}
	}
	if err := checkName("candidate volume", cfg.RunID+"-c99999"); err != nil {
		return err
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
		req.ID = CheckID(cfg.RunID + "-r99999-" + c.Name)
		if !req.Valid() {
			return bad("check %q has an incomplete request", c.Name)
		}
	}
	b := cfg.Budget
	if b.Deadline.IsZero() || b.TurnTimeout <= 0 || b.AgentTurns <= 0 || b.Rounds <= 0 {
		return bad("the budget needs a deadline, a turn timeout, agent turns and rounds")
	}
	if !(HarvestRequest{ID: WorkspaceID(cfg.RunID + "-work"), MaxEntries: cfg.MaxEntries, MaxBytes: cfg.MaxBytes}).Valid() {
		return bad("harvest bounds are outside the host ceilings")
	}
	switch {
	case cfg.Account == nil || cfg.OwnerOf == nil:
		return bad("Account and OwnerOf are required")
	case cfg.Agents == nil || cfg.Workspaces == nil || cfg.Evidence == nil:
		return bad("Agents, Workspaces and Evidence are required")
	case len(cfg.Checks) > 0 && cfg.Runner == nil:
		return bad("checks need a Runner")
	}
	return nil
}

func newEngine(parent context.Context, cfg Config) *engine {
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	ctx, cancel := context.WithDeadline(parent, cfg.Budget.Deadline)
	e := &engine{cfg: cfg, ctx: ctx, cancel: cancel, base: context.WithoutCancel(parent), byPath: map[string]*mgr{}, delivered: map[string]int{}}
	e.cond = sync.NewCond(&e.mu)
	for _, m := range cfg.Team {
		slug := slugOf(m.Path)
		x := &mgr{Manager: m, slug: slug, execHome: cfg.RunID + "-" + slug + "-x", verifyHome: cfg.RunID + "-" + slug + "-v"}
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

// halt stops the run with the first reason given and cancels all running work.
func (e *engine) halt(reason string) {
	e.stopMu.Lock()
	if e.reason == "" {
		e.reason = reason
	}
	e.stopMu.Unlock()
	e.cancel()
}

// fail ends the run with a definite error of its inputs or ports.
func (e *engine) fail(err error) {
	e.stopMu.Lock()
	if e.failure == nil {
		e.failure = err
	}
	e.stopMu.Unlock()
	e.halt(err.Error())
}

// portError reports a port failure unless the run was already stopped.
func (e *engine) portError(what string, err error) {
	if e.ctx.Err() != nil {
		return
	}
	e.fail(fmt.Errorf("%s: %w", what, err))
}

func (e *engine) stopReason(fallback string) string {
	e.stopMu.Lock()
	defer e.stopMu.Unlock()
	switch {
	case e.reason != "":
		return e.reason
	case errors.Is(e.ctx.Err(), context.DeadlineExceeded):
		return "the deadline was reached"
	case e.ctx.Err() != nil:
		return "the run was cancelled"
	}
	return fallback
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

func (e *engine) materialize(id WorkspaceID, tree snapshot.Tree) (candidate, bool) {
	obs, err := e.cfg.Workspaces.Materialize(e.ctx, WorkspaceSpec{ID: id, Tree: tree})
	if err == nil && (obs.Uncertain || obs.Volume == "" || obs.Tree != tree.Digest()) {
		err = fmt.Errorf("materialization was not confirmed: %s", obs.Reason)
	}
	if err != nil {
		e.portError("materialize "+string(id), err)
		return candidate{}, false
	}
	_ = e.record(Event{Kind: "workspace.materialize", Data: map[string]any{"workspace": id, "volume": obs.Volume, "tree": tree.Digest().String(), "entries": tree.EntryCount()}})
	return candidate{tree: tree, id: id, volume: obs.Volume}, e.ctx.Err() == nil
}

// affordRound reports whether the turn budget still covers one complete Verify round.
func (e *engine) affordRound() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.turnCount+len(e.team) <= e.cfg.Budget.AgentTurns
}

// reserveTurnLocked counts one turn when the budget leaves room for it and for
// the following reserve turns.
func (e *engine) reserveTurnLocked(reserve int) bool {
	if e.turnCount+1+reserve > e.cfg.Budget.AgentTurns {
		return false
	}
	e.turnCount++
	return true
}

func reconcile(parent context.Context, cfg Config) (Outcome, error) {
	if err := validate(cfg); err != nil {
		return Outcome{Status: Failed, Reason: err.Error()}, err
	}
	e := newEngine(parent, cfg)
	status, reason, tree := e.run()
	e.mu.Lock()
	e.closing = true
	e.executeOpen = false
	e.mu.Unlock()
	e.cancel()
	e.wg.Wait()

	e.stopMu.Lock()
	failure := e.failure
	e.stopMu.Unlock()
	switch {
	case failure != nil:
		status, reason, tree = Failed, failure.Error(), snapshot.Tree{}
	case status == Verified && e.evidenceFailed() != nil:
		status, reason, tree = Incomplete, e.evidenceFailed().Error(), snapshot.Tree{}
	}
	out := e.outcome(status, reason, tree)
	if err := e.record(Event{Kind: "outcome", Data: map[string]any{"status": status, "reason": reason}}); err != nil && status == Verified {
		out.Status, out.Reason, out.Candidate = Incomplete, err.Error(), snapshot.Tree{}
	}
	if out.Status == Failed {
		return out, failure
	}
	return out, nil
}

func (e *engine) evidenceFailed() error {
	e.evMu.Lock()
	defer e.evMu.Unlock()
	return e.evErr
}

func (e *engine) outcome(status Status, reason string, tree snapshot.Tree) Outcome {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := Outcome{Status: status, Reason: reason, Candidate: tree, Turns: append([]TurnRecord(nil), e.turns...), Rounds: append([]Round(nil), e.rounds...), Usage: e.usage}
	for _, f := range e.findings {
		out.Findings = append(out.Findings, *f)
	}
	for _, q := range e.questions {
		out.Questions = append(out.Questions, *q)
	}
	for _, m := range e.team {
		if m.execUsed {
			out.Homes = append(out.Homes, m.execHome)
		}
		if m.verifyUsed {
			out.Homes = append(out.Homes, m.verifyHome)
		}
	}
	return out
}

func (e *engine) incomplete(fallback string) (Status, string, snapshot.Tree) {
	return Incomplete, e.stopReason(fallback), snapshot.Tree{}
}

// run drives the rounds. Every exit leaves no turn or check running except
// Execute turns that reconcile waits for after cancelling the context.
func (e *engine) run() (Status, string, snapshot.Tree) {
	cfg := e.cfg
	managers := make([]string, 0, len(e.team))
	for _, m := range e.team {
		managers = append(managers, m.Path)
	}
	if e.record(Event{Kind: "run.start", Data: map[string]any{"run": cfg.RunID, "managers": managers, "start": cfg.Start.Digest().String(), "agentTurns": cfg.Budget.AgentTurns, "rounds": cfg.Budget.Rounds, "deadline": cfg.Budget.Deadline}}) != nil {
		return e.incomplete("")
	}
	if !e.affordRound() {
		return e.incomplete(fmt.Sprintf("the agent turn budget of %d does not cover one Verify round of %d managers", cfg.Budget.AgentTurns, len(e.team)))
	}
	cand, ok := e.materialize(e.volumeID("c0"), cfg.Start)
	if !ok {
		e.closeVolume(e.volumeID("c0"))
		return e.incomplete("")
	}
	work := e.volumeID("work")
	shared, ok := e.materialize(work, cfg.Start)
	if !ok {
		e.closeVolume(cand.id)
		e.closeVolume(work)
		return e.incomplete("")
	}
	e.work = shared.volume
	for round := 1; ; round++ {
		if round > 1 && !e.affordRound() {
			return e.incomplete("the agent turn budget cannot cover another Verify round")
		}
		green := e.verifyRound(round, cand)
		e.closeVolume(cand.id)
		if e.ctx.Err() != nil {
			return e.incomplete("")
		}
		if green {
			e.closeVolume(work)
			return Verified, "", cand.tree
		}
		if reason := e.disabledOwnerOpen(); reason != "" {
			e.halt(reason)
			return e.incomplete("")
		}
		if round >= cfg.Budget.Rounds {
			return e.incomplete(fmt.Sprintf("the round budget of %d ended without a green round", cfg.Budget.Rounds))
		}
		e.executePhase()
		if e.ctx.Err() != nil {
			return e.incomplete("")
		}
		if reason := e.disabledOwnerOpen(); reason != "" {
			e.halt(reason)
			return e.incomplete("")
		}
		next, ok := e.capture(round, work, cand.tree)
		if !ok {
			return e.incomplete("")
		}
		cand = next
	}
}

// capture harvests the quiescent shared tree into the next candidate.
func (e *engine) capture(round int, work WorkspaceID, previous snapshot.Tree) (candidate, bool) {
	h, err := e.cfg.Workspaces.Harvest(e.ctx, HarvestRequest{ID: work, MaxEntries: e.cfg.MaxEntries, MaxBytes: e.cfg.MaxBytes})
	if err == nil && !h.Complete() {
		err = fmt.Errorf("harvest is incomplete (uncertain %t, %d unsupported entries): %s", h.Uncertain, len(h.Unsupported), h.Reason)
	}
	if err != nil {
		e.halt("capture of the shared tree is uncertain: " + err.Error())
		return candidate{}, false
	}
	next, ok := e.materialize(e.volumeID(fmt.Sprintf("c%d", round)), h.Tree)
	if !ok {
		e.closeVolume(e.volumeID(fmt.Sprintf("c%d", round)))
		return candidate{}, false
	}
	next.index = round
	e.observeWrites(round, previous, h.Tree)
	_ = e.record(Event{Kind: "candidate", Round: round + 1, Data: map[string]any{"tree": h.Tree.Digest().String(), "workspace": next.id, "volume": next.volume, "entries": h.Tree.EntryCount(), "harvest": map[string]any{"uncertain": h.Uncertain, "unsupported": h.Unsupported, "reason": h.Reason}}})
	return next, e.ctx.Err() == nil
}

// observeWrites records which owners' files changed although the owner ran no
// Execute turn since the previous candidate. Managers are trusted; this is an
// observation, not enforcement.
func (e *engine) observeWrites(round int, before, after snapshot.Tree) {
	changed := map[string][]string{}
	for _, c := range snapshot.Compare(before, after) {
		owner := e.cfg.OwnerOf(c.Path())
		if _, ok := e.byPath[owner]; !ok {
			owner = "."
		}
		changed[owner] = append(changed[owner], c.Path().String())
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, m := range e.team {
		m.changed = nil
		if paths := changed[m.Path]; len(paths) > 0 && !m.ran {
			m.changed = paths
			_ = e.record(Event{Kind: "write.observed", Manager: m.Path, Round: round, Data: map[string]any{"paths": paths}})
		}
		m.ran = false
	}
}

// disabledOwnerOpen names a reason when a Manager stopped by repeated failures
// still owns an open blocking finding, which no later round can resolve.
func (e *engine) disabledOwnerOpen() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, f := range e.findings {
		if f.Blocking && !f.Closed && e.byPath[f.Owner].disabled {
			return fmt.Sprintf("manager %q stopped after repeated failed turns while finding %s is open", f.Owner, f.ID)
		}
	}
	return ""
}
