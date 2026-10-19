package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var errTest = errors.New("test failure")

func init() { ownerPollInterval = 5 * time.Millisecond }

func mustPath(t testing.TB, s string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func treeOf(t testing.TB, files map[string]string) snapshot.Tree {
	t.Helper()
	var entries []snapshot.Entry
	for path, content := range files {
		e, err := snapshot.NewEntry(mustPath(t, path), snapshot.Regular, []byte(content))
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, e)
	}
	tree, err := snapshot.NewTree(entries)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// fakeWS keeps volumes in memory. A volume mounted by a running fake turn
// makes Harvest uncertain, like the real port.
type fakeWS struct {
	t         testing.TB
	mu        sync.Mutex
	vols      map[WorkspaceID]*fakeVol
	byName    map[string]*fakeVol
	mounted   map[string]int
	order     []string
	closed    []WorkspaceID
	harvests  int
	uncertain bool
	extra     []string
	matErr    error
}

type fakeVol struct {
	id     WorkspaceID
	name   string
	files  map[string]string
	closed bool
}

func newFakeWS(t testing.TB) *fakeWS {
	return &fakeWS{t: t, vols: map[WorkspaceID]*fakeVol{}, byName: map[string]*fakeVol{}, mounted: map[string]int{}}
}

func (w *fakeWS) Materialize(_ context.Context, s WorkspaceSpec) (WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.matErr != nil {
		return WorkspaceObservation{}, w.matErr
	}
	if !nameRE.MatchString(string(s.ID)) {
		w.t.Errorf("workspace ID %q is not a valid name", s.ID)
	}
	if _, dup := w.vols[s.ID]; dup {
		return WorkspaceObservation{}, errors.New("exists")
	}
	v := &fakeVol{id: s.ID, name: "vol-" + string(s.ID), files: map[string]string{}}
	for _, e := range s.Tree.Entries() {
		v.files[e.Path().String()] = string(e.Bytes())
	}
	w.vols[s.ID], w.byName[v.name] = v, v
	w.order = append(w.order, string(s.ID))
	return WorkspaceObservation{ID: s.ID, Volume: v.name, Tree: s.Tree.Digest()}, nil
}

func (w *fakeWS) tree(v *fakeVol) snapshot.Tree { return treeOf(w.t, v.files) }

func (w *fakeWS) Harvest(_ context.Context, r HarvestRequest) (WorkspaceHarvest, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.harvests++
	v := w.vols[r.ID]
	if v == nil {
		return WorkspaceHarvest{}, errors.New("no such volume")
	}
	h := WorkspaceHarvest{ID: r.ID, Tree: w.tree(v), Unsupported: w.extra, Uncertain: w.uncertain || w.mounted[v.name] > 0}
	if h.Uncertain {
		h.Reason = "volume is mounted or limits were hit"
	}
	return h, nil
}

func (w *fakeWS) Peek(ctx context.Context, r HarvestRequest) (WorkspaceHarvest, error) {
	return w.Harvest(ctx, r)
}

func (w *fakeWS) Close(_ context.Context, id WorkspaceID) (WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = append(w.closed, id)
	if v := w.vols[id]; v != nil {
		v.closed = true
	}
	return WorkspaceObservation{ID: id, Closed: true}, nil
}

func (w *fakeWS) mount(name string, delta int) {
	w.mu.Lock()
	w.mounted[name] += delta
	w.mu.Unlock()
}

func (w *fakeWS) write(name, path, content string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.byName[name].files[path] = content
}

func (w *fakeWS) isClosed(id WorkspaceID) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Contains(w.closed, id)
}

// fakeAgents runs scripted turns. The script calls the tool handler like a
// provider would.
type fakeAgents struct {
	t         testing.TB
	ws        *fakeWS
	mu        sync.Mutex
	script    func(*call) error
	turns     []Turn
	active    int
	maxActive int
	prompts   map[string]string
	// noConversation makes Execute turns report no resumable conversation.
	noConversation bool
}

type call struct {
	t      testing.TB
	ctx    context.Context
	turn   Turn
	h      ToolHandler
	ag     *fakeAgents
	noStop bool
}

func (a *fakeAgents) Run(ctx context.Context, turn Turn, h ToolHandler) (TurnResult, error) {
	if !nameRE.MatchString(turn.ID) || !nameRE.MatchString(turn.Home) || turn.Profile == "" || turn.Timeout <= 0 || len(turn.Tools) == 0 || turn.Workspace == "" {
		a.t.Errorf("incomplete turn %+v", turn)
	}
	a.mu.Lock()
	a.turns = append(a.turns, turn)
	if a.prompts == nil {
		a.prompts = map[string]string{}
	}
	a.prompts[turn.ID] = turn.Prompt
	a.active++
	a.maxActive = max(a.maxActive, a.active)
	a.mu.Unlock()
	writes := turn.Kind == ExecuteTurn && !turn.ReadOnly
	if writes {
		a.ws.mount(turn.Workspace, 1)
	}
	c := &call{t: a.t, ctx: ctx, turn: turn, h: h, ag: a}
	var err error
	if a.script != nil {
		err = a.script(c)
	} else {
		err = c.defaultScript()
	}
	if writes {
		a.ws.mount(turn.Workspace, -1)
	}
	a.mu.Lock()
	a.active--
	a.mu.Unlock()
	res := TurnResult{Stopped: !c.noStop, Started: time.Now(), Ended: time.Now(), Usage: Usage{Model: "m", InputTokens: 10, OutputTokens: 5, ProviderTurns: 1}}
	if turn.Kind == ExecuteTurn && !a.noConversation {
		res.Conversation = turn.Resume
		if res.Conversation == "" {
			res.Conversation = "conv-" + turn.ID
		}
	}
	return res, err
}

func (a *fakeAgents) turnsOf(kind TurnKind) []Turn {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []Turn
	for _, t := range a.turns {
		if t.Kind == kind {
			out = append(out, t)
		}
	}
	return out
}

func (a *fakeAgents) prompt(id string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.prompts[id]
}

func (c *call) round() int {
	var n int
	rest := strings.TrimPrefix(c.turn.ID, "run1-r")
	fmt.Sscanf(rest, "%d", &n)
	return n
}

// is reports whether the turn is the given Manager's turn kind in the round.
func (c *call) is(round int, manager string, kind TurnKind) bool {
	return c.round() == round && c.turn.Manager == manager && c.turn.Kind == kind
}

func (c *call) tool(name string, args any) ToolAnswer {
	raw, err := json.Marshal(args)
	if err != nil {
		c.t.Fatal(err)
	}
	return c.h(c.ctx, name, raw)
}

func (c *call) write(path, content string) {
	if c.turn.ReadOnly {
		c.t.Errorf("turn %s writes to a read-only workspace", c.turn.ID)
	}
	c.ag.ws.write(c.turn.Workspace, path, content)
}

type fnd map[string]any

func finding(owner, obligation string, blocking bool) fnd {
	return fnd{"owner": owner, "obligation": obligation, "observed": "observed " + obligation, "blocking": blocking}
}

func (c *call) assess(satisfied bool, findings []fnd, closed ...string) ToolAnswer {
	var cl []map[string]string
	for _, id := range closed {
		cl = append(cl, map[string]string{"id": id, "reason": "resolved"})
	}
	return c.tool(toolAssess, map[string]any{"summary": "assessed " + c.turn.Manager, "satisfied": satisfied, "findings": findings, "closed": cl})
}

func (c *call) finish(status string) ToolAnswer {
	return c.tool(toolFinish, map[string]any{"status": status, "summary": "did something"})
}

func (c *call) send(to, text string) ToolAnswer {
	return c.tool(toolSend, map[string]any{"to": to, "text": text})
}

var idRE = regexp.MustCompile(`Finding (F\d+) \[`)

// myOpen lists the open findings the Verify prompt shows as raised by this Manager.
func (c *call) myOpen() []string {
	_, rest, ok := strings.Cut(c.turn.Prompt, "Open findings you raised earlier")
	if !ok {
		return nil
	}
	section, _, _ := strings.Cut(rest, "\n\n")
	var ids []string
	for _, m := range idRE.FindAllStringSubmatch(section, -1) {
		ids = append(ids, m[1])
	}
	return ids
}

// defaultScript assesses satisfied and closes every finding the Manager raised
// earlier, or finishes an Execute turn as done.
func (c *call) defaultScript() error {
	if c.turn.Kind == VerifyTurn {
		if a := c.assess(true, nil, c.myOpen()...); a.Refused {
			c.t.Errorf("assess refused: %s", a.Text)
		}
		return nil
	}
	if a := c.finish("done"); a.Refused {
		c.t.Errorf("finish refused: %s", a.Text)
	}
	return nil
}

func (c *call) wait(ch <-chan struct{}) error {
	select {
	case <-ch:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	case <-time.After(5 * time.Second):
		c.t.Errorf("turn %s timed out waiting", c.turn.ID)
		return errors.New("timeout")
	}
}

// fakeChecks passes unless fail names the check.
type fakeChecks struct {
	mu   sync.Mutex
	reqs []CheckRequest
	fail func(CheckRequest) (CheckReceipt, error, bool)
}

func (f *fakeChecks) Run(_ context.Context, r CheckRequest) (CheckReceipt, error) {
	f.mu.Lock()
	f.reqs = append(f.reqs, r)
	f.mu.Unlock()
	if f.fail != nil {
		if rec, err, ok := f.fail(r); ok {
			return rec, err
		}
	}
	return CheckReceipt{ID: r.ID, ImageID: r.ImageID, Tree: r.Tree.Digest(), Started: true, Exited: true}, nil
}

func failing(name string, rounds ...int) func(CheckRequest) (CheckReceipt, error, bool) {
	return func(r CheckRequest) (CheckReceipt, error, bool) {
		for _, n := range rounds {
			if string(r.ID) == fmt.Sprintf("run1-r%d-%s", n, name) {
				return CheckReceipt{ID: r.ID, ImageID: r.ImageID, Tree: r.Tree.Digest(), Started: true, Exited: true, ExitCode: 1, Output: []byte("FAIL: boom")}, nil, true
			}
		}
		return CheckReceipt{}, nil, false
	}
}

type fakeOwner struct {
	mu      sync.Mutex
	asked   []Question
	answers map[string]string
	askErr  error
}

func (o *fakeOwner) Ask(_ context.Context, q Question) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.askErr != nil {
		return o.askErr
	}
	o.asked = append(o.asked, q)
	return nil
}

func (o *fakeOwner) Answer(_ context.Context, id string) (string, bool, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	a, ok := o.answers[id]
	return a, ok, nil
}

func (o *fakeOwner) answer(id, text string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.answers == nil {
		o.answers = map[string]string{}
	}
	o.answers[id] = text
}

func (o *fakeOwner) questions() []Question {
	o.mu.Lock()
	defer o.mu.Unlock()
	return slices.Clone(o.asked)
}

type recEvidence struct {
	mu        sync.Mutex
	events    []Event
	failAfter int
	failKind  string
}

func (r *recEvidence) Record(e Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failKind != "" && e.Kind == r.failKind || r.failAfter > 0 && len(r.events) >= r.failAfter {
		return errors.New("disk full")
	}
	r.events = append(r.events, e)
	return nil
}

func (r *recEvidence) Transcript(turn string) string { return "" }

func (r *recEvidence) kinds(kind string) []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Event
	for _, e := range r.events {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

func (r *recEvidence) count(kind string) int { return len(r.kinds(kind)) }

type fixture struct {
	t      *testing.T
	ws     *fakeWS
	agents *fakeAgents
	checks *fakeChecks
	owner  *fakeOwner
	ev     *recEvidence
	cfg    Config
}

var teamPaths = []string{".", "lending", "lending/reminders", "library"}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ws := newFakeWS(t)
	f := &fixture{t: t, ws: ws, agents: &fakeAgents{t: t, ws: ws}, checks: &fakeChecks{}, owner: &fakeOwner{}, ev: &recEvidence{}}
	team := []Manager{
		{Path: ".", Children: []string{"lending", "library"}, Role: "whole library", Brief: "Root brief.", ExecuteProfile: "exec", VerifyProfile: "ver"},
		{Path: "lending", Parent: ".", Children: []string{"lending/reminders"}, Role: "loans", Brief: "Lending brief.", ExecuteProfile: "exec", VerifyProfile: "ver"},
		{Path: "lending/reminders", Parent: "lending", Role: "overdue notices", Brief: "Reminders brief.", ExecuteProfile: "exec", VerifyProfile: "ver"},
		{Path: "library", Parent: ".", Role: "program", Brief: "Library brief.", ExecuteProfile: "exec", VerifyProfile: "ver"},
	}
	f.cfg = Config{
		RunID: "run1",
		Team:  team,
		Start: treeOf(t, map[string]string{"go.mod": "module x\n", "library/a.go": "a", "lending/l.go": "l", "lending/reminders/r.go": "r"}),
		Account: func(snapshot.Tree) []Finding {
			return nil
		},
		OwnerOf: func(p snapshot.Path) string {
			best := "."
			for _, m := range teamPaths[1:] {
				if strings.HasPrefix(p.String(), m+"/") && len(m) > len(best) {
					best = m
				}
			}
			return best
		},
		OwnerRequest:   "Realize the model.",
		OwnerDecisions: []string{"Decision: three loans."},
		Budget:         Budget{Deadline: time.Now().Add(20 * time.Second), TurnTimeout: time.Minute, AgentTurns: 100, Rounds: 6},
		MaxEntries:     1000,
		MaxBytes:       1 << 20,
		Agents:         f.agents,
		Workspaces:     ws,
		Runner:         f.checks,
		Owner:          f.owner,
		Evidence:       f.ev,
	}
	return f
}

func (f *fixture) addCheck(name, owner string) {
	req := checkRequestFor(f.t)
	f.cfg.Checks = append(f.cfg.Checks, Check{Name: name, RequiredBy: "library/realization", Owner: owner, Request: req})
}

func checkRequestFor(t testing.TB) CheckRequest {
	dir, err := snapshot.DirectoryAt(mustPath(t, "realization"))
	if err != nil {
		t.Fatal(err)
	}
	return CheckRequest{ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Executable: "/usr/local/go/bin/go", Args: []string{"test", "./..."}, Directory: dir,
		Limits: CheckLimits{MilliCPUs: 1000, MemoryBytes: 1 << 30, Processes: 64}, Timeout: time.Minute, MaxOutputBytes: 1 << 16}
}

func (f *fixture) run() Outcome {
	f.t.Helper()
	out, err := Reconcile(context.Background(), f.cfg)
	if err != nil && out.Status != Failed {
		f.t.Fatalf("unexpected error %v with status %s", err, out.Status)
	}
	f.assertEnded(out)
	return out
}

// assertEnded checks the invariants every outcome has: all turns ended and
// were observed stopped, and no fake container still runs.
func (f *fixture) assertEnded(out Outcome) {
	f.t.Helper()
	f.agents.mu.Lock()
	active := f.agents.active
	f.agents.mu.Unlock()
	if active != 0 {
		f.t.Errorf("%d turns still run", active)
	}
	for _, tr := range out.Turns {
		if tr.Ended.IsZero() {
			f.t.Errorf("turn %s has no end", tr.ID)
		}
	}
}

func turnIDs(out Outcome) []string {
	var ids []string
	for _, t := range out.Turns {
		ids = append(ids, t.ID)
	}
	return ids
}

func countKind(out Outcome, kind TurnKind) int {
	n := 0
	for _, t := range out.Turns {
		if t.Kind == kind {
			n++
		}
	}
	return n
}
