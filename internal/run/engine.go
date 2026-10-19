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

// ownerPollInterval is how often the engine asks the owner channel for answers.
var ownerPollInterval = 3 * time.Second

const (
	maxExecuteFailures = 2
	maxVerifyAttempts  = 2
	maxToolText        = 16 << 10
	maxToolFindings    = 64
	outputTail         = 8 << 10
	maxListedPaths     = 40
	maxListedNames     = 20
)

// candidate is one immutable stand with its read-only volume.
type candidate struct {
	index  int
	tree   snapshot.Tree
	id     WorkspaceID
	volume string
	// unsupported lists the entries of the harvested shared tree that a tree
	// cannot hold; the candidate cannot be green while they stand.
	unsupported []string
}

// item is one undelivered piece of work for a Manager. finding is the ID of
// the finding an item restates; such an item lapses once the finding is closed.
// report marks a child's done report, which alone does not start the parent's
// turn while the parent's subtree still works.
type item struct {
	text    string
	finding string
	report  bool
}

// mgr is the engine's state of one Manager. Every field is guarded by engine.mu.
type mgr struct {
	Manager
	slug         string
	execHome     string
	verifyHome   string
	conversation string
	items        []item
	consumed     []item
	running      bool
	disabled     bool
	// released is set when the Manager's start was released: Root's at the start
	// of the run, every other Manager's when its parent's first Execute turn
	// ended or the parent was disabled. A Manager that is not released runs no
	// turn; its items wait.
	released   bool
	failures   int
	execCount  int
	execUsed   bool
	verifyUsed bool
	ran        bool
	changed    []string
	// refused is set when the budget refused the Manager's Execute turn in the
	// current phase: its pending items no longer keep the phase open.
	refused bool

	// executed is set by an Execute turn that finished without error; until
	// then the Manager's turn carries the full mandate, even when a failed turn
	// left a conversation behind.
	executed bool
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
	execRound   int
	work        string
	executeOpen bool
	closing     bool
	running     int
	polling     bool
	nextFinding int
	nextQuest   int

	// budgetLimited is set when the budget refused an Execute turn in the
	// current phase; phaseTurns counts the Execute turns admitted in it.
	budgetLimited bool
	phaseTurns    int
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
		for _, name := range []string{cfg.RunID + "-r99999-" + slug + "-x99999", cfg.RunID + "-r99999-" + slug + "-v99999", cfg.RunID + "-" + slug + "-x", cfg.RunID + "-" + slug + "-v"} {
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
			if c == m.Path {
				return bad("manager %q is its own child", m.Path)
			}
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
	case cfg.ArtifactFindingsAdvisory && cfg.ResolveDefinition == nil:
		return bad("ArtifactFindingsAdvisory needs ResolveDefinition")
	}
	last := 0
	for _, q := range cfg.Questions {
		n := questionNumber(q.ID)
		if n <= last || !q.Answered {
			return bad("earlier question %q must be answered and numbered after Q%d", q.ID, last)
		}
		last = n
	}
	return nil
}

// questionNumber returns n of a question identity "Qn" (n >= 1), or 0. A refused
// question keeps its number, so earlier questions may have gaps.
func questionNumber(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "Q"))
	if err != nil || n < 1 || id != fmt.Sprintf("Q%d", n) {
		return 0
	}
	return n
}

// validatePlan checks the mounts of an enforced write plan: clean relative
// paths of the shared tree without duplicates, none of them "." or an existing
// file of the start stand, and every mount before the mounts of its
// descendants, so that a later mount never shadows an earlier one. The Base and
// Mounts of an unenforced plan are not used.
func validatePlan(m Manager, files map[string]bool) error {
	if !m.Write.Enforced {
		return nil
	}
	seen := map[string]bool{}
	for _, mount := range m.Write.Mounts {
		if _, err := snapshot.NewPath(mount.Path); err != nil {
			return fmt.Errorf("manager %q has a mount %q that is not a clean relative path below /work", m.Path, mount.Path)
		}
		if seen[mount.Path] {
			return fmt.Errorf("manager %q mounts %q twice", m.Path, mount.Path)
		}
		seen[mount.Path] = true
		for end := 0; end <= len(mount.Path); end++ {
			if (end == len(mount.Path) || mount.Path[end] == '/') && files[mount.Path[:end]] {
				return fmt.Errorf("manager %q mounts %q, but %q is a file of the start stand", m.Path, mount.Path, mount.Path[:end])
			}
		}
	}
	for i, mount := range m.Write.Mounts {
		for _, later := range m.Write.Mounts[i+1:] {
			if strings.HasPrefix(mount.Path, later.Path+"/") {
				return fmt.Errorf("manager %q lists mount %q after its descendant %q; parents come first", m.Path, later.Path, mount.Path)
			}
		}
	}
	return nil
}

// mountDirectories returns the sorted union of the mount paths of all enforced
// write plans: the directories the shared work volume needs before the first
// Execute turn.
func (e *engine) mountDirectories() []string {
	set := map[string]bool{}
	for _, m := range e.team {
		if !m.Write.Enforced {
			continue
		}
		for _, mount := range m.Write.Mounts {
			set[mount.Path] = true
		}
	}
	dirs := make([]string, 0, len(set))
	for dir := range set {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	return dirs
}

func newEngine(parent context.Context, cfg Config) *engine {
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	ctx, cancel := context.WithDeadline(parent, cfg.Budget.Deadline)
	e := &engine{cfg: cfg, ctx: ctx, cancel: cancel, base: context.WithoutCancel(parent), byPath: map[string]*mgr{}, delivered: map[string]int{}}
	e.cond = sync.NewCond(&e.mu)
	for _, q := range cfg.Questions {
		e.questions = append(e.questions, &q)
	}
	if len(cfg.Questions) > 0 {
		e.nextQuest = questionNumber(cfg.Questions[len(cfg.Questions)-1].ID)
	}
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

func (e *engine) materialize(id WorkspaceID, tree snapshot.Tree, directories []string) (candidate, bool) {
	obs, err := e.cfg.Workspaces.Materialize(e.ctx, WorkspaceSpec{ID: id, Tree: tree, Directories: directories})
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

// run drives the rounds: Execute phase, capture, Verify round. Every exit
// leaves no turn or check running except Execute turns that reconcile waits
// for after cancelling the context.
func (e *engine) run() (Status, string, snapshot.Tree) {
	cfg := e.cfg
	managers := make([]string, 0, len(e.team))
	for _, m := range e.team {
		managers = append(managers, m.Path)
	}
	if e.record(Event{Kind: "run.start", Data: map[string]any{"run": cfg.RunID, "managers": managers, "start": cfg.Start.Digest().String(), "agentTurns": cfg.Budget.AgentTurns, "rounds": cfg.Budget.Rounds, "deadline": cfg.Budget.Deadline}}) != nil {
		return e.incomplete("")
	}
	if cfg.Budget.AgentTurns < 2*len(e.team) {
		return e.incomplete(fmt.Sprintf("the agent turn budget of %d does not cover one Execute turn and one Verify turn of each of the %d managers", cfg.Budget.AgentTurns, len(e.team)))
	}
	work := e.volumeID("work")
	shared, ok := e.materialize(work, cfg.Start, e.mountDirectories())
	if !ok {
		e.closeVolume(work)
		return e.incomplete("")
	}
	e.work = shared.volume
	e.mu.Lock()
	e.execRound = 1
	// The start stand's drift goes to its owners behind their initial item.
	for _, f := range groupedStart(cfg.StartFindings) {
		raised := e.structuralLocked(f, 0)
		o := e.byPath[raised.Owner]
		o.items = append(o.items, item{text: renderFinding(raised), finding: raised.ID})
	}
	e.releaseLocked(e.team[0], "the run started")
	e.mu.Unlock()
	previous := cfg.Start
	for round := 1; ; round++ {
		stalled := e.executePhase(round)
		if e.ctx.Err() != nil {
			return e.incomplete("")
		}
		if stalled {
			e.halt("the agent turn budget ended without a green round: no further Execute turn fits beside a whole Verify round")
			return e.incomplete("")
		}
		if reason := e.disabledOwnerOpen(); reason != "" {
			e.halt(reason)
			return e.incomplete("")
		}
		cand, ok := e.capture(round, work, previous)
		if !ok {
			return e.incomplete("")
		}
		previous = cand.tree
		green := e.verifyRound(round, cand)
		e.closeVolume(cand.id)
		if e.ctx.Err() != nil {
			return e.incomplete("")
		}
		if green {
			return Verified, "", cand.tree
		}
		if reason := e.disabledOwnerOpen(); reason != "" {
			e.halt(reason)
			return e.incomplete("")
		}
		if round >= cfg.Budget.Rounds {
			return e.incomplete(fmt.Sprintf("the round budget of %d ended without a green round", cfg.Budget.Rounds))
		}
	}
}

// groupedStart keeps the start findings, except that more than maxListedNames
// findings of one owner and obligation become one finding that counts them and
// shows the first maxListedNames, so a large drift cannot flood a prompt.
func groupedStart(findings []Finding) []Finding {
	type key struct{ owner, obligation string }
	var order []key
	groups := map[key][]Finding{}
	for _, f := range findings {
		k := key{f.Owner, f.Obligation}
		if groups[k] == nil {
			order = append(order, k)
		}
		groups[k] = append(groups[k], f)
	}
	var out []Finding
	for _, k := range order {
		group := groups[k]
		if len(group) <= maxListedNames {
			out = append(out, group...)
			continue
		}
		shown := make([]string, maxListedNames)
		for i, f := range group[:maxListedNames] {
			shown[i] = f.Observed
		}
		merged := group[0]
		merged.Observed = fmt.Sprintf("%d findings of this kind; the first %d:\n- %s", len(group), maxListedNames, strings.Join(shown, "\n- "))
		out = append(out, merged)
	}
	return out
}

// capture harvests the quiescent shared tree into the candidate of the round.
// previous is the stand of the previous candidate, or the start.
func (e *engine) capture(round int, work WorkspaceID, previous snapshot.Tree) (candidate, bool) {
	h, err := e.cfg.Workspaces.Harvest(e.ctx, HarvestRequest{ID: work, MaxEntries: e.cfg.MaxEntries, MaxBytes: e.cfg.MaxBytes})
	if err == nil && h.Uncertain {
		err = fmt.Errorf("harvest is uncertain: %s", h.Reason)
	}
	if err != nil {
		e.halt("capture of the shared tree is uncertain: " + err.Error())
		return candidate{}, false
	}
	next, ok := e.materialize(e.volumeID(fmt.Sprintf("c%d", round)), h.Tree, nil)
	if !ok {
		e.closeVolume(e.volumeID(fmt.Sprintf("c%d", round)))
		return candidate{}, false
	}
	next.index, next.unsupported = round, h.Unsupported
	e.observeWrites(round, previous, h.Tree)
	_ = e.record(Event{Kind: "candidate", Round: round, Data: map[string]any{"tree": h.Tree.Digest().String(), "workspace": next.id, "volume": next.volume, "entries": h.Tree.EntryCount(), "harvest": map[string]any{"uncertain": h.Uncertain, "unsupported": h.Unsupported, "reason": h.Reason}}})
	if e.ctx.Err() != nil {
		e.closeVolume(next.id)
		return candidate{}, false
	}
	return next, true
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
