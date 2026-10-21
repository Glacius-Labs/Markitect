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

// fakeWS keeps volumes in memory. A volume mounted writable by a running fake
// turn makes Harvest uncertain, like the real port.
type fakeWS struct {
	t             testing.TB
	mu            sync.Mutex
	vols          map[WorkspaceID]*fakeVol
	byName        map[string]*fakeVol
	mounted       map[string]int
	order         []string
	closed        []WorkspaceID
	harvests      int
	peeks         int
	uncertain     bool
	extra         []string
	peekUncertain bool
	dirs          map[WorkspaceID][]string
	matErr        error
}

type fakeVol struct {
	id    WorkspaceID
	name  string
	files map[string]string
}

func newFakeWS(t testing.TB) *fakeWS {
	return &fakeWS{t: t, vols: map[WorkspaceID]*fakeVol{}, byName: map[string]*fakeVol{}, mounted: map[string]int{}, dirs: map[WorkspaceID][]string{}}
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
	w.dirs[s.ID] = s.Directories
	w.order = append(w.order, string(s.ID))
	return WorkspaceObservation{ID: s.ID, Volume: v.name, Tree: s.Tree.Digest()}, nil
}

func (w *fakeWS) treeLocked(v *fakeVol) snapshot.Tree { return treeOf(w.t, v.files) }

func (w *fakeWS) Harvest(_ context.Context, r HarvestRequest) (WorkspaceHarvest, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.harvests++
	v := w.vols[r.ID]
	if v == nil {
		return WorkspaceHarvest{}, errors.New("no such volume")
	}
	h := WorkspaceHarvest{ID: r.ID, Tree: w.treeLocked(v), Unsupported: w.extra, Uncertain: w.uncertain || w.mounted[v.name] > 0}
	if h.Uncertain {
		h.Reason = "volume is mounted or limits were hit"
	}
	return h, nil
}

func (w *fakeWS) Peek(_ context.Context, r HarvestRequest) (WorkspaceHarvest, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.peeks++
	v := w.vols[r.ID]
	if v == nil {
		return WorkspaceHarvest{}, errors.New("no such volume")
	}
	h := WorkspaceHarvest{ID: r.ID, Tree: w.treeLocked(v), Unsupported: w.extra, Uncertain: w.peekUncertain}
	if h.Uncertain {
		h.Reason = "the volume changed while it was observed"
	}
	return h, nil
}

func (w *fakeWS) Close(_ context.Context, id WorkspaceID) (WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = append(w.closed, id)
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

func (w *fakeWS) remove(name, path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.byName[name].files, path)
}

func (w *fakeWS) file(id WorkspaceID, path string) (string, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	content, ok := w.vols[id].files[path]
	return content, ok
}

func (w *fakeWS) isClosed(id WorkspaceID) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Contains(w.closed, id)
}

// fakeAgents runs scripted turns. The script calls the tool handler like a
// provider would; without a script every turn follows the protocol.
type fakeAgents struct {
	t           testing.TB
	ws          *fakeWS
	obligations map[string][]string
	mu          sync.Mutex
	script      func(*call) error
	turns       []Turn
	active      int
	maxActive   int
}

type call struct {
	t      testing.TB
	ctx    context.Context
	turn   Turn
	h      ToolHandler
	ag     *fakeAgents
	noStop bool
	// conversation overrides the resumable conversation the turn reports.
	conversation *string
	// decided lists the proposals the script already decided in this turn.
	decided map[string]bool
}

func (a *fakeAgents) Run(ctx context.Context, turn Turn, h ToolHandler) (TurnResult, error) {
	if !nameRE.MatchString(turn.ID) || !nameRE.MatchString(turn.Home) || turn.Profile == "" || turn.Timeout <= 0 || len(turn.Tools) == 0 || turn.Workspace == "" {
		a.t.Errorf("incomplete turn %+v", turn)
	}
	if turn.Kind == VerifyTurn && (!turn.ReadOnly || slices.ContainsFunc(turn.Mounts, func(m PathMount) bool { return m.Writable })) {
		a.t.Errorf("verify turn %s can write", turn.ID)
	}
	a.mu.Lock()
	a.turns = append(a.turns, turn)
	a.active++
	a.maxActive = max(a.maxActive, a.active)
	a.mu.Unlock()
	writes := !turn.ReadOnly || slices.ContainsFunc(turn.Mounts, func(m PathMount) bool { return m.Writable })
	if writes {
		a.ws.mount(turn.Workspace, 1)
	}
	c := &call{t: a.t, ctx: ctx, turn: turn, h: h, ag: a}
	var err error
	if a.script != nil {
		err = a.script(c)
	} else {
		c.protocol()
	}
	if writes {
		a.ws.mount(turn.Workspace, -1)
	}
	a.mu.Lock()
	a.active--
	a.mu.Unlock()
	res := TurnResult{Stopped: !c.noStop, Started: time.Now(), Ended: time.Now(), Usage: Usage{Model: "m", InputTokens: 10, OutputTokens: 5, ProviderTurns: 1}}
	res.Conversation = turn.Resume
	if res.Conversation == "" {
		res.Conversation = "conv-" + turn.ID
	}
	if c.conversation != nil {
		res.Conversation = *c.conversation
	}
	return res, err
}

func (a *fakeAgents) all() []Turn {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.turns)
}

func (a *fakeAgents) turnsOf(manager string) []Turn {
	var out []Turn
	for _, t := range a.all() {
		if t.Manager == manager {
			out = append(out, t)
		}
	}
	return out
}

func (c *call) tool(name string, args any) ToolAnswer {
	raw, err := json.Marshal(args)
	if err != nil {
		c.t.Fatal(err)
	}
	a := c.h(c.ctx, name, raw)
	// Remember what this turn already decided or answered, so that the
	// protocol does not act on its prompt's stale list again.
	var in struct{ Kind, Proposal, About string }
	if !a.Refused && json.Unmarshal(raw, &in) == nil {
		if c.decided == nil {
			c.decided = map[string]bool{}
		}
		switch {
		case name == toolReport && (in.Kind == kindAccept || in.Kind == kindReturn):
			c.decided[in.Proposal] = true
		case name == toolSend && in.Kind == sendAnswer:
			c.decided[in.About] = true
		}
	}
	return a
}

// must calls a tool and fails the test when it is refused.
func (c *call) must(name string, args any) ToolAnswer {
	a := c.tool(name, args)
	if a.Refused {
		c.t.Errorf("turn %s: %s %v refused: %s", c.turn.ID, name, args, a.Text)
	}
	return a
}

// pass returns the Pass identity of the turn, "p1" for run1-p1-root-m1.
func (c *call) pass() string {
	rest := strings.TrimPrefix(c.turn.ID, "run1-")
	p, _, _ := strings.Cut(rest, "-")
	return p
}

func (c *call) is(pass, manager string) bool { return c.pass() == pass && c.turn.Manager == manager }

// now is the task line of the turn's prompt.
func (c *call) now() string {
	_, rest, _ := strings.Cut(c.turn.Prompt, "\nNow: ")
	line, _, _ := strings.Cut(rest, "\n")
	return line
}

func (c *call) hasReportField(field string) bool {
	for _, tool := range c.turn.Tools {
		if tool.Name == toolReport {
			var s struct {
				Properties map[string]any `json:"properties"`
			}
			if err := json.Unmarshal(tool.InputSchema, &s); err != nil {
				c.t.Fatal(err)
			}
			_, ok := s.Properties[field]
			return ok
		}
	}
	return false
}

func (c *call) planning() bool {
	for _, tool := range c.turn.Tools {
		if tool.Name == toolReport {
			var s struct {
				Properties map[string]any `json:"properties"`
			}
			if err := json.Unmarshal(tool.InputSchema, &s); err != nil {
				c.t.Fatal(err)
			}
			_, ok := s.Properties["plan"]
			return ok
		}
	}
	return false
}

func (c *call) helper() bool {
	return !slices.ContainsFunc(c.turn.Tools, func(t ToolSpec) bool { return t.Name == toolSend })
}

var (
	decideRE = regexp.MustCompile(`- decide (P\d+),`)
	answerRE = regexp.MustCompile(`- answer (Q\d+) from`)
	raisedRE = regexp.MustCompile(`Finding (F\d+) \[`)
)

func (c *call) pendingDecisions() []string {
	var ids []string
	for _, m := range decideRE.FindAllStringSubmatch(c.turn.Prompt, -1) {
		ids = append(ids, m[1])
	}
	return ids
}

func (c *call) pendingQuestions() []string {
	var ids []string
	for _, m := range answerRE.FindAllStringSubmatch(c.turn.Prompt, -1) {
		ids = append(ids, m[1])
	}
	return ids
}

// earlierRaised lists the findings the review prompt shows as raised by this
// reviewer in earlier Passes.
func (c *call) earlierRaised() []string {
	_, rest, ok := strings.Cut(c.turn.Prompt, "Open findings you raised in earlier Passes")
	if !ok {
		return nil
	}
	section, _, _ := strings.Cut(rest, "\n\n")
	var ids []string
	for _, m := range raisedRE.FindAllStringSubmatch(section, -1) {
		ids = append(ids, m[1])
	}
	return ids
}

// sourceOf is a start-stand file each fixture Manager owns, which a review
// cites as inspected source.
var sourceOf = map[string]string{".": "go.mod", "lending": "lending/l.go", "lending/reminders": "lending/reminders/r.go", "library": "library/a.go"}

func (c *call) coverage() []map[string]any {
	var out []map[string]any
	for _, id := range c.ag.obligations[c.turn.Manager] {
		entry := map[string]any{"definition": id, "status": coverDone, "note": "covered by the work of " + c.turn.Manager, "conditions": "fixture contract; static declarations only, no public runtime claim"}
		if !c.planning() {
			entry["evidence"] = []map[string]string{{"kind": "static", "claim": "the fixture contract of " + id, "source": sourceOf[c.turn.Manager], "location": "whole file", "reasoning": "the fixture realizes it by contract"}}
		}
		out = append(out, entry)
	}
	return out
}

func samplePlan() map[string]any {
	return map[string]any{"approach": "do the own part", "artifacts": []string{}, "inputs": []any{}, "outputs": []any{}, "dependencies": []any{}, "order": "basis first", "blockers": []string{}, "assumptions": []string{}}
}

// envelope builds a basis or result report with full coverage.
func (c *call) envelope(kind string, extra map[string]any) map[string]any {
	args := map[string]any{"kind": kind, "summary": kind + " of " + c.turn.Manager, "obligations": c.coverage()}
	if (c.turn.Manager == "." || c.turn.Manager == "lending") && c.hasReportField("integration") {
		args["integration"] = map[string]any{"status": coverDone, "conditions": "model-derived static fixture composition", "note": "protocol fixture only: source declares compatible static layout", "evidence": []map[string]string{{"kind": "static", "authority": "library/interface", "claim": "static fixture layout", "source": "go.mod", "location": "module declaration", "reasoning": "static declaration; no executable behavior asserted"}}}
	}
	if c.planning() {
		args["plan"] = samplePlan()
	}
	if kind == kindResult {
		args["verdict"] = string(Satisfied)
	}
	for k, v := range extra {
		args[k] = v
	}
	if kind == kindBasis && strings.Contains(c.turn.Prompt, "MODEL-FIRST BASIS:") {
		for _, entry := range args["obligations"].([]map[string]any) {
			entry["status"], entry["note"] = coverOpen, "model-derived cases; candidate assessment pending"
			delete(entry, "evidence")
		}
		if entry, ok := args["integration"].(map[string]any); ok {
			entry["status"], entry["note"] = coverOpen, "model-derived composition; candidate assessment pending"
			delete(entry, "evidence")
		}
	}
	return args
}

func (c *call) basis(extra map[string]any) ToolAnswer {
	return c.tool(toolReport, c.envelope(kindBasis, extra))
}

func (c *call) result(extra map[string]any) ToolAnswer {
	return c.tool(toolReport, c.envelope(kindResult, extra))
}

func (c *call) accept(id string) ToolAnswer {
	return c.tool(toolReport, map[string]any{"kind": kindAccept, "proposal": id})
}

func (c *call) send(to, kind, text, about string) ToolAnswer {
	args := map[string]any{"to": to, "kind": kind, "text": text}
	if about != "" {
		args["about"] = about
	}
	return c.tool(toolSend, args)
}

// protocol follows the base process: decide the children's proposals and
// answer the questions waiting for this Manager, then submit the basis or the
// complete result its stage asks for. A reviewer closes the findings it raised
// earlier. A helper reports its task done.
func (c *call) protocol() {
	if c.helper() {
		c.must(toolReport, map[string]any{"summary": "did the task"})
		return
	}
	for _, id := range c.pendingDecisions() {
		if !c.decided[id] {
			c.must(toolReport, map[string]any{"kind": kindAccept, "proposal": id})
		}
	}
	for _, q := range c.pendingQuestions() {
		if !c.decided[q] {
			c.answerQuestion(q)
		}
	}
	now := c.now()
	switch {
	case strings.HasPrefix(now, "work on your own contribution"):
		extra := map[string]any{}
		if raised := c.earlierRaised(); len(raised) > 0 {
			var closed []map[string]string
			for _, id := range raised {
				closed = append(closed, map[string]string{"id": id, "reason": "resolved on this candidate", "evidence": "the corrected source on this candidate"})
			}
			extra["closed"] = closed
		}
		c.must(toolReport, c.envelope(kindBasis, extra))
	case strings.HasPrefix(now, "submit your complete result"), strings.HasPrefix(now, "integrate your children"):
		// A genuine finding within the responsibility keeps the verdict at
		// needs_correction.
		if a := c.result(nil); a.Refused && strings.Contains(a.Text, "stay open") {
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
		} else if a.Refused {
			c.t.Errorf("turn %s: result refused: %s", c.turn.ID, a.Text)
		}
	}
}

// answerQuestion answers a question addressed to this Manager, to its asker.
func (c *call) answerQuestion(id string) {
	re := regexp.MustCompile(`Question ` + id + ` from your (?:parent|child) ([^ ]+)`)
	from := "parent"
	if m := re.FindStringSubmatch(c.turn.Prompt); m != nil {
		from = m[1]
		if strings.Contains(c.turn.Prompt, "Question "+id+" from your parent") {
			from = "parent"
		}
	}
	c.must(toolSend, map[string]any{"to": from, "kind": sendAnswer, "text": "answer to " + id, "about": id})
}

// writable reports whether the turn's mounts allow a write to path: the base
// mount first, then each mount in order overrides it for its directory or file.
func writable(turn Turn, path string) bool {
	ok := !turn.ReadOnly
	for _, m := range turn.Mounts {
		if path == m.Path || !m.File && strings.HasPrefix(path, m.Path+"/") {
			ok = m.Writable
		}
	}
	return ok
}

func (c *call) write(path, content string) {
	if !writable(c.turn, path) {
		c.t.Errorf("turn %s writes %s, which is read-only for it", c.turn.ID, path)
		return
	}
	c.ag.ws.write(c.turn.Workspace, path, content)
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

// fakeChecks passes unless fail decides otherwise.
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

// failingIn fails the named check in the given Passes.
func failingIn(name string, passes ...string) func(CheckRequest) (CheckReceipt, error, bool) {
	return func(r CheckRequest) (CheckReceipt, error, bool) {
		for _, p := range passes {
			if string(r.ID) == fmt.Sprintf("run1-%s-%s", p, name) {
				return CheckReceipt{ID: r.ID, ImageID: r.ImageID, Tree: r.Tree.Digest(), Started: true, Exited: true, ExitCode: 1, Output: []byte("FAIL: boom")}, nil, true
			}
		}
		return CheckReceipt{}, nil, false
	}
}

type recEvidence struct {
	mu       sync.Mutex
	events   []Event
	failKind string
}

func (r *recEvidence) Record(e Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failKind != "" && e.Kind == r.failKind {
		return errors.New("disk full")
	}
	r.events = append(r.events, e)
	return nil
}

func (r *recEvidence) Transcript(string) string { return "" }

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

// index returns the position of the first event matching kind, manager and,
// when given, a data field value; -1 when none.
func (r *recEvidence) index(kind, manager string, field string, value any) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, e := range r.events {
		if e.Kind != kind || manager != "" && e.Manager != manager {
			continue
		}
		if field == "" {
			return i
		}
		if data, ok := e.Data.(map[string]any); ok && data[field] == value {
			return i
		}
	}
	return -1
}

type fixture struct {
	t      *testing.T
	ws     *fakeWS
	agents *fakeAgents
	checks *fakeChecks
	ev     *recEvidence
	cfg    Config
}

var teamPaths = []string{".", "lending", "lending/reminders", "library"}

// ownerByPrefix is the fixture's ownership: the deepest Manager whose path
// prefixes the file, and Root for every other file.
func ownerByPrefix(p snapshot.Path) string {
	best := "."
	for _, m := range teamPaths[1:] {
		if strings.HasPrefix(p.String(), m+"/") && len(m) > len(best) {
			best = m
		}
	}
	return best
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ws := newFakeWS(t)
	obligations := map[string][]string{"lending": {"lending/borrow"}, "lending/reminders": {"lending/reminders/notice"}, "library": {"library/interface"}}
	f := &fixture{t: t, ws: ws, agents: &fakeAgents{t: t, ws: ws, obligations: obligations}, checks: &fakeChecks{}, ev: &recEvidence{}}
	plan := func(dirs ...string) WritePlan {
		var mounts []PathMount
		for _, d := range dirs {
			writable := !strings.HasPrefix(d, "!")
			mounts = append(mounts, PathMount{Path: strings.TrimPrefix(d, "!"), Writable: writable})
		}
		return WritePlan{Mounts: mounts}
	}
	team := []Manager{
		{Path: ".", Children: []string{"lending", "library"}, Role: "whole library", Brief: "Root brief.", ExecuteProfile: "exec", VerifyProfile: "ver",
			Write: WritePlan{Base: true, Mounts: []PathMount{{Path: "lending"}, {Path: "library"}}}},
		{Path: "lending", Parent: ".", Children: []string{"lending/reminders"}, Role: "loans", Brief: "Lending brief.", Obligations: obligations["lending"], ExecuteProfile: "exec", VerifyProfile: "ver",
			Write: plan("lending", "!lending/reminders")},
		{Path: "lending/reminders", Parent: "lending", Role: "overdue notices", Brief: "Reminders brief.", Obligations: obligations["lending/reminders"], ExecuteProfile: "exec", VerifyProfile: "ver",
			Write: plan("lending/reminders")},
		{Path: "library", Parent: ".", Role: "program", Brief: "Library brief.", Obligations: obligations["library"], ExecuteProfile: "exec", VerifyProfile: "ver",
			Write: plan("library")},
	}
	definitions := map[string]ModelDefinition{
		"lending/borrow":           {ID: "lending/borrow", Owner: "lending", Text: "Borrow a copy."},
		"lending/reminders/notice": {ID: "lending/reminders/notice", Owner: "lending/reminders", Text: "Notice overdue loans."},
		"library/interface":        {ID: "library/interface", Owner: "library", Text: "One program."},
	}
	f.cfg = Config{
		RunID:        "run1",
		Team:         team,
		Start:        treeOf(t, map[string]string{"go.mod": "module x\n", "library/a.go": "a", "lending/l.go": "l", "lending/reminders/r.go": "r"}),
		ModelCheck:   func(snapshot.Tree) []Finding { return nil },
		FileCheck:    func(snapshot.Tree) []Finding { return nil },
		OwnerOf:      ownerByPrefix,
		Registry:     func(snapshot.Tree) RegistryView { return RegistryView{Shards: map[string]string{}} },
		Definition:   func(id string) (ModelDefinition, bool) { d, ok := definitions[id]; return d, ok },
		OwnerRequest: "Realize the model.",
		Budget:       Budget{Deadline: time.Now().Add(20 * time.Second), TurnTimeout: time.Minute, AgentTurns: 400, Passes: 12},
		MaxEntries:   1000,
		MaxBytes:     1 << 20,
		Agents:       f.agents,
		Workspaces:   ws,
		Runner:       f.checks,
		Evidence:     f.ev,
	}
	return f
}

func (f *fixture) addCheck(name, owner string) {
	f.cfg.Checks = append(f.cfg.Checks, Check{Name: name, RequiredBy: "library/interface", Owner: owner, Request: checkRequestFor(f.t)})
}

func checkRequestFor(t testing.TB) CheckRequest {
	dir, err := snapshot.DirectoryAt(mustPath(t, "realization"))
	if err != nil {
		t.Fatal(err)
	}
	return CheckRequest{ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Executable: "/usr/local/go/bin/go", Args: []string{"test", "./..."}, Directory: dir,
		Limits: CheckLimits{MilliCPUs: 1000, MemoryBytes: 1 << 30, Processes: 64}, Timeout: time.Minute, MaxOutputBytes: 1 << 16}
}

var corrections = []Purpose{Plan, Prepare, Implement}

func planSpec() PassSpec {
	return PassSpec{Purpose: Plan, Mandate: "Plan your own obligations.", Plans: true, Corrections: corrections}
}

func prepareSpec() PassSpec {
	return PassSpec{Purpose: Prepare, Mandate: "Prepare your foundation.", Writes: true, Corrections: corrections}
}

func implementSpec() PassSpec {
	return PassSpec{Purpose: Implement, Mandate: "Realize your obligations.", Writes: true, Corrections: corrections}
}

func verifySpec() PassSpec {
	return PassSpec{Purpose: Verify, Mandate: "Assess the candidate.", Review: true, Guidance: "A genuine finding is a model violation.", Corrections: corrections}
}

// open starts a run of the fixture.
func (f *fixture) open() *Run {
	f.t.Helper()
	r, err := Open(context.Background(), f.cfg)
	if err != nil {
		f.t.Fatalf("open: %v", err)
	}
	return r
}

// finish ends a run and checks the invariants of every outcome: every turn
// ended and was observed, and no fake container still runs.
func (f *fixture) finish(r *Run, reason string) Outcome {
	f.t.Helper()
	out, err := r.Finish(reason)
	if err != nil && out.Status != Failed {
		f.t.Fatalf("unexpected error %v with status %s", err, out.Status)
	}
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
	return out
}

// apply runs Plan, Prepare, Implement and Verify once.
func (f *fixture) apply() (Outcome, []PassResult) {
	f.t.Helper()
	r := f.open()
	var results []PassResult
	for _, spec := range []PassSpec{planSpec(), prepareSpec(), implementSpec(), verifySpec()} {
		res := r.Pass(spec)
		results = append(results, res)
		if res.Stopped {
			break
		}
	}
	return f.finish(r, "the forward Passes ended"), results
}

func turnIDs(out Outcome) []string {
	var ids []string
	for _, t := range out.Turns {
		ids = append(ids, t.ID)
	}
	return ids
}
