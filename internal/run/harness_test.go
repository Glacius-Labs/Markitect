package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// stepInputs keeps the real settings inputs so live step preparation resolves
// them again rather than trusting arbitrary Host context JSON. The one-Package
// variant keeps only the root and lending.
func stepInputs(t *testing.T, onePackage bool) run.StepPreparation {
	return stepInputsWith(t, onePackage, nil)
}

// stepInputsWith also changes the settings facts before they are resolved.
func stepInputsWith(t *testing.T, onePackage bool, settings func(*organization.SettingsFacts)) run.StepPreparation {
	var facts organization.SettingsFacts
	change := func(input *model.Input) {
		if !onePackage {
			return
		}
		var entries []model.Entry
		for _, e := range input.Root.Entries {
			if e.Path != "model/empty" && e.Path != "model/lending/reminders/item.yaml" {
				entries = append(entries, e)
			}
		}
		input.Root.Entries = entries
	}
	input := fixtureWith(t, change, func(f *organization.SettingsFacts) {
		if settings != nil {
			settings(f)
		}
		facts = *f
	})
	subjects := int64(len(input.Settings.Managers()))
	topology, err := organization.BuildTopology(input.Compilation)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range topology.Managers() {
		if len(m.Children) > 0 {
			subjects++
		}
	}
	input.Policy.Reserve.SessionStarts = 2 * subjects
	input.Policy.Reserve.Steps = 2 * subjects
	return run.StepPreparation{Binding: input, Settings: facts}
}

// sourceFacts are the step facts of the captured source, version zero.
func sourceFacts(inputs run.StepPreparation) run.StepFacts {
	return run.StepFacts{Tree: inputs.Binding.Census, Registry: inputs.Binding.Registry}
}

// taskSession prepares a session for the run's next admissible step.
func taskSession(t *testing.T, c *run.Coordinator, id run.SessionID, candidate snapshot.TreeID) run.Command {
	t.Helper()
	item, ok := c.Next()
	if !ok {
		t.Fatal("no admissible step")
	}
	role, purpose := "manager", run.Writing
	if item.Phase == run.PhaseReviewerExpectations || item.Phase == run.PhaseReviewerCandidate {
		role, purpose = "reviewer", run.Assessment
	}
	profile := payload(t, `{"profile":"synthetic recorded metadata"}`)
	tools, err := run.ToolCatalog(item.Phase)
	if err != nil {
		t.Fatal(err)
	}
	plan := run.SessionPlan{ID: id, Role: role, Workspace: "linux-workspace-" + string(id), ProviderHome: item.Home, Resume: item.Resume, Candidate: candidate,
		ReadOnly: item.Phase != run.PhaseWork && item.Phase != run.PhaseIntegrate, ImageID: testImage, CredentialRef: "credential-ref", MaxWireBytes: 1 << 20, Profile: profile.Ref(), Tools: tools.Ref()}
	cmd, err := c.PrepareTaskSessionCommand(meta("prepare-"+string(id)), item, plan, run.Reservation{ID: "open-" + string(id), Kind: run.SessionStart, Purpose: purpose, Maximum: time.Minute}, profile, tools)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

// fakeWorkspaces keeps workspace trees in memory.
type fakeWorkspaces struct {
	trees   map[run.WorkspaceID]snapshot.Tree
	volumes map[string]run.WorkspaceID
	// Fault injection: the next failHarvest harvests fail; failMaterialize
	// refuses every materialization. closes counts Close calls.
	failHarvest     int
	failMaterialize bool
	closes          int
}

func newWorkspaces() *fakeWorkspaces {
	return &fakeWorkspaces{trees: map[run.WorkspaceID]snapshot.Tree{}, volumes: map[string]run.WorkspaceID{}}
}

func (w *fakeWorkspaces) Materialize(_ context.Context, s run.WorkspaceSpec) (run.WorkspaceObservation, error) {
	if w.failMaterialize {
		return run.WorkspaceObservation{ID: s.ID, Uncertain: true, Reason: "injected"}, errors.New("injected materialization failure")
	}
	if _, ok := w.trees[s.ID]; ok {
		return run.WorkspaceObservation{ID: s.ID}, errors.New("workspace exists")
	}
	w.trees[s.ID] = s.Tree
	volume := "vol-" + string(s.ID)
	w.volumes[volume] = s.ID
	return run.WorkspaceObservation{ID: s.ID, Volume: volume, Tree: s.Tree.Digest()}, nil
}

func (w *fakeWorkspaces) Harvest(_ context.Context, r run.HarvestRequest) (run.WorkspaceHarvest, error) {
	if w.failHarvest > 0 {
		w.failHarvest--
		return run.WorkspaceHarvest{ID: r.ID, Uncertain: true, Reason: "injected"}, errors.New("injected harvest failure")
	}
	tree, ok := w.trees[r.ID]
	if !ok {
		return run.WorkspaceHarvest{ID: r.ID, Uncertain: true, Reason: "absent"}, errors.New("absent")
	}
	return run.WorkspaceHarvest{ID: r.ID, Tree: tree}, nil
}

func (w *fakeWorkspaces) Close(_ context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
	w.closes++
	delete(w.trees, id)
	return run.WorkspaceObservation{ID: id, Closed: true}, nil
}

// write changes one file of a mounted workspace, as an Agent would.
func (w *fakeWorkspaces) write(t *testing.T, volume, path, content string) {
	t.Helper()
	id := w.volumes[volume]
	entries := map[string]snapshot.Entry{}
	for _, e := range w.trees[id].Entries() {
		entries[e.Path().String()] = e
	}
	entries[path] = file(t, path, snapshot.Regular, content)
	var all []snapshot.Entry
	for _, e := range entries {
		all = append(all, e)
	}
	w.trees[id] = tree(t, all)
}

// agentStep is what a scripted Agent sees of one step.
type agentStep struct {
	Manager string
	Phase   run.Phase
	Attempt int
	Packet  run.StepPacket
}

// action is one scripted Agent action: optional file writes, then one tool
// call. Want, when set, is the expected answer status.
type action struct {
	Write map[string]string
	Tool  string
	Args  string
	Want  string
}

type agentScript func(agentStep) []action

// fakeSessions is a scripted adapter. It drives each step through the session
// channel exactly as the provider adapters do: init, readiness or bind, the
// turn input, tool calls with their answers and the terminal observation.
type fakeSessions struct {
	t        *testing.T
	ws       *fakeWorkspaces
	script   agentScript
	attempts map[string]int
	answers  []string
	packets  []run.StepPacket
	n        int
	// Fault injection: the next failOpen opens fail after the init send.
	// stops counts Stop calls.
	failOpen int
	stops    int
	onOpen   func()
}

func (f *fakeSessions) meta(kind string) run.RecordMeta {
	f.n++
	return meta(fmt.Sprintf("fake-%s-%d", kind, f.n))
}

func (f *fakeSessions) send(ctx context.Context, io run.SessionIO, step run.StepID, call run.ProviderCallID, data string) error {
	f.n++
	id := run.MessageID(fmt.Sprintf("message-%d", f.n))
	if _, err := io.PrepareSend(ctx, run.TransportPreparation{RecordMeta: f.meta("send"), Message: id, Step: step, Call: call, Wire: wire(f.t, data)}); err != nil {
		return err
	}
	_, err := io.ObserveSend(ctx, run.TransportObservation{RecordMeta: f.meta("sent"), Message: id, Outcome: run.Sent, Reason: "observed write"})
	return err
}

func (f *fakeSessions) Open(ctx context.Context, a run.SessionAdmission, io run.SessionIO) (run.SessionObservation, error) {
	if err := f.send(ctx, io, "", "", `{"control":"initialize"}`); err != nil {
		return run.SessionObservation{}, err
	}
	if f.onOpen != nil {
		f.onOpen()
	}
	if f.failOpen > 0 {
		// Like the provider adapters, record the start as uncertain first.
		f.failOpen--
		o := run.SessionObservation{RecordMeta: f.meta("bind"), Uncertain: true, Reason: "injected start failure"}
		if _, err := io.Bind(ctx, o); err != nil {
			return o, err
		}
		return o, errors.New("injected start failure")
	}
	if a.Plan.Resume != "" {
		_, err := io.Bind(ctx, run.SessionObservation{RecordMeta: f.meta("bind"), ProviderSession: a.Plan.Resume, Wire: wire(f.t, `{"resumed":true}`), Reason: "resumed conversation"})
		return run.SessionObservation{ProviderSession: a.Plan.Resume}, err
	}
	_, err := io.Ready(ctx, run.SessionReadiness{RecordMeta: f.meta("ready"), Wire: wire(f.t, `{"control":"initialized"}`), Reason: "observed control acknowledgment"})
	return run.SessionObservation{Ready: true}, err
}

func (f *fakeSessions) Step(ctx context.Context, a run.StepAdmission, io run.SessionIO) (run.StepObservation, error) {
	var packet run.StepPacket
	if err := json.Unmarshal(a.Input.Bytes(), &packet); err != nil {
		return run.StepObservation{}, err
	}
	if _, err := run.DecodeStepContract(a.OutputContract); err != nil {
		return run.StepObservation{}, err
	}
	step := a.Plan.ID
	if err := f.send(ctx, io, step, "", `{"user":"compiled turn"}`); err != nil {
		return run.StepObservation{}, fmt.Errorf("turn input: %w", err)
	}
	if a.Session.Resume == "" {
		if _, err := io.Bind(ctx, run.SessionObservation{RecordMeta: f.meta("bind"), ProviderSession: run.ProviderSessionID("provider-" + string(a.Session.ID)), Wire: wire(f.t, `{"thread":"started"}`), Reason: "observed provider conversation"}); err != nil {
			return run.StepObservation{}, fmt.Errorf("bind: %w", err)
		}
	}
	if _, err := io.ObserveStep(ctx, run.StepObservation{RecordMeta: f.meta("started"), Step: step, ProviderStep: run.ProviderStepID("turn-" + string(step)), Wire: wire(f.t, `{"turn":"started"}`), Reason: "observed turn start"}); err != nil {
		return run.StepObservation{}, fmt.Errorf("start: %w", err)
	}
	f.packets = append(f.packets, packet)
	key := packet.Manager + " " + string(packet.Phase)
	attempt := f.attempts[key]
	f.attempts[key]++
	for i, act := range f.script(agentStep{Manager: packet.Manager, Phase: packet.Phase, Attempt: attempt, Packet: packet}) {
		for path, content := range act.Write {
			f.ws.write(f.t, a.Session.Workspace, path, content)
		}
		callID := run.ProviderCallID(fmt.Sprintf("call-%s-%d", step, i))
		frame := fmt.Sprintf(`{"id":%d,"name":%q,"arguments":%s}`, i+1, act.Tool, act.Args)
		result, err := io.Call(ctx, run.ToolCall{RecordMeta: f.meta("call"), Step: step, Call: callID, Tool: act.Tool, Input: wire(f.t, frame)})
		if err != nil {
			return run.StepObservation{}, fmt.Errorf("call %s: %w", act.Tool, err)
		}
		f.answers = append(f.answers, fmt.Sprintf("%s %s %s %s %s", packet.Manager, packet.Phase, act.Tool, result.Outcome.Status, result.Outcome.Reason))
		if act.Want != "" && result.Outcome.Status != act.Want {
			f.t.Errorf("%s %s %s answered %s: %s", packet.Manager, packet.Phase, act.Tool, result.Outcome.Status, result.Outcome.Reason)
		}
		if err := f.send(ctx, io, step, callID, string(result.Output.Bytes())); err != nil {
			return run.StepObservation{}, fmt.Errorf("answer %s: %w", act.Tool, err)
		}
	}
	observation := run.StepObservation{RecordMeta: f.meta("terminal"), Step: step, ProviderStep: run.ProviderStepID("turn-" + string(step)), Wire: wire(f.t, `{"turn":"completed"}`), Terminal: true, Reason: "observed turn completion"}
	if _, err := io.ObserveStep(ctx, observation); err != nil {
		return observation, fmt.Errorf("terminal: %w", err)
	}
	return observation, nil
}

func (f *fakeSessions) Inspect(context.Context, run.InspectionRequest) (run.StepObservation, error) {
	return run.StepObservation{}, errors.New("not inspected")
}

func (f *fakeSessions) Stop(ctx context.Context, r run.StopRequest, io run.SessionIO) (run.StopObservation, error) {
	f.stops++
	observation := run.StopObservation{RecordMeta: f.meta("stop"), Terminated: true, Reason: "observed container exit"}
	_, err := io.ObserveStop(ctx, observation)
	return observation, err
}

// harness runs real Coordinator and Engine code against scripted ports.
type harness struct {
	t        *testing.T
	c        *run.Coordinator
	j        *memoryJournal
	inputs   run.StepPreparation
	ws       *fakeWorkspaces
	sessions *fakeSessions
	engine   *run.Engine
	clock    time.Time
	// journal, when set, replaces j as the engine's journal.
	journal run.Journal
}

func newHarness(t *testing.T, onePackage bool, script agentScript) *harness {
	t.Helper()
	return newHarnessWith(t, stepInputs(t, onePackage), script)
}

func newHarnessWith(t *testing.T, inputs run.StepPreparation, script agentScript) *harness {
	t.Helper()
	b, err := run.Bind(inputs.Binding)
	if err != nil {
		t.Fatal(err)
	}
	c, j := runningWith(t, b)
	ws := newWorkspaces()
	h := &harness{t: t, c: c, j: j, inputs: inputs, ws: ws, sessions: &fakeSessions{t: t, ws: ws, script: script, attempts: map[string]int{}}, clock: observed}
	h.engine = h.newEngine(c)
	return h
}

func (h *harness) newEngine(c *run.Coordinator) *run.Engine {
	h.t.Helper()
	return h.newEngineWith(c, nil)
}

func (h *harness) newEngineWith(c *run.Coordinator, checks run.Checks) *run.Engine {
	h.t.Helper()
	profile := payload(h.t, `{"profile":"synthetic recorded metadata"}`)
	var journal run.Journal = h.j
	if h.journal != nil {
		journal = h.journal
	}
	var reviewChecks map[string]run.CheckRequest
	if checks != nil {
		reviewChecks = map[string]run.CheckRequest{"go-tests": requiredCheck(h.t).Request}
	}
	e, err := run.NewEngine(context.Background(), run.EngineConfig{Coordinator: c, Journal: journal, Sessions: h.sessions, Workspaces: h.ws, Checks: checks, ReviewChecks: reviewChecks, Preparation: h.inputs,
		Profile: func(run.ProfileRequest) (run.SessionProfile, error) {
			return run.SessionProfile{ImageID: testImage, CredentialRef: "credential-ref", Profile: profile, MaxWireBytes: 1 << 20}, nil
		},
		Clock: func() time.Time { h.clock = h.clock.Add(time.Millisecond); return h.clock },
		Limits: run.EngineLimits{StartTimeout: time.Second, StopTimeout: time.Second, HarvestEntries: 1000, HarvestBytes: 1 << 20,
			Record: realization.CandidateRecordLimits{EncodedBytes: 1 << 20, FileBytes: 1 << 20, ManifestEntries: 1000, MetadataRows: 10000}}})
	if err != nil {
		h.t.Fatal(err)
	}
	return e
}

// packet returns the input of the manager's first step in a phase.
func (h *harness) packet(manager string, phase run.Phase) run.StepPacket {
	h.t.Helper()
	for _, p := range h.sessions.packets {
		if p.Manager == manager && p.Phase == phase {
			return p
		}
	}
	h.t.Fatal("no step", manager, phase)
	return run.StepPacket{}
}

// advanceUntil runs admitted steps until stop reports true for the next
// admission.
func (h *harness) advanceUntil(stop func(run.Admission) bool) {
	h.t.Helper()
	for {
		item, ok := h.c.Next()
		if !ok || stop(item) {
			return
		}
		if _, err := h.engine.Advance(context.Background()); err != nil {
			h.t.Fatal(err)
		}
	}
}

func (h *harness) task(manager string) run.TaskView {
	h.t.Helper()
	v, ok := h.c.Task(manager)
	if !ok {
		h.t.Fatal("no task", manager)
	}
	return v
}

// Scripted Agent arguments.
const (
	reportArgs       = `{"summary":"Inspected the area.","proposedChanges":[],"prerequisites":[],"uncertainties":[],"questions":[]}`
	expectationsArgs = `{"cases":[{"name":"item","expectation":"The library realizes Item.","discriminates":"A missing Item type fails."}]}`
)

func submitArgs(purpose, scope string) string {
	return fmt.Sprintf(`{"purpose":%q,"scope":%q,"summary":"Realized the area.","assessments":[],"dependencies":[]}`, purpose, scope)
}

func passArgs(resolve ...string) string {
	resolutions := "["
	for i, id := range resolve {
		if i > 0 {
			resolutions += ","
		}
		resolutions += fmt.Sprintf(`{"finding":%q,"state":"resolved","grounds":"The candidate now realizes it.","evidence":[]}`, id)
	}
	return `{"verdict":"pass","summary":"The candidate meets the expectations.","evidence":[],"findings":[],"resolutions":` + resolutions + `]}`
}

const rejectArgs = `{"verdict":"changes_requested","summary":"Item is missing.","evidence":[],"resolutions":[],"findings":[{"severity":"blocking","definitions":["lending/Item"],"grounds":"No type realizes Item.","requestedEvidence":"A type named Item."}]}`

// happyAgent explores with a report, submits local and integration work
// complete, writes lending's file, and passes every review.
func happyAgent(s agentStep) []action {
	switch s.Phase {
	case run.PhaseExplore:
		return []action{{Tool: run.ToolReport, Args: reportArgs, Want: run.ToolOK}}
	case run.PhaseWork:
		var writes map[string]string
		if s.Manager == "lending" {
			writes = map[string]string{"out/library.go": "package library\n\ntype Item struct{}\n"}
		}
		return []action{{Write: writes, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
	case run.PhaseIntegrate:
		return []action{{Tool: run.ToolSubmit, Args: submitArgs("integration", "complete"), Want: run.ToolOK}}
	case run.PhaseReviewerExpectations:
		return []action{{Tool: run.ToolRecordExpectations, Args: expectationsArgs, Want: run.ToolOK}}
	case run.PhaseReviewerCandidate:
		return []action{{Tool: run.ToolSubmitReview, Args: passArgs(), Want: run.ToolOK}}
	}
	return nil
}

// exploreNext drives the run's next admissible Explore step to a report with
// fixed identities: session id, reservations open-<id> and invoke-<id>.
func exploreNext(t *testing.T, c *run.Coordinator, j *memoryJournal, inputs run.StepPreparation, id run.SessionID) {
	t.Helper()
	ctx := context.Background()
	item, ok := c.Next()
	if !ok || item.Phase != run.PhaseExplore {
		t.Fatal("no admissible Explore step", item)
	}
	cmd := taskSession(t, c, id, inputs.Binding.Census.Digest())
	commit(t, c, j, cmd)
	f := &fakeSessions{t: t, ws: newWorkspaces(), script: func(agentStep) []action { return []action{{Tool: run.ToolReport, Args: reportArgs, Want: run.ToolOK}} }, attempts: map[string]int{}}
	io, err := c.SessionChannel(j, id, func(_ context.Context, call run.ToolCall, _ int64) (run.ToolOutcome, error) {
		args, err := run.CallArguments(call.Tool, call.Input.Journal.Bytes())
		if err == nil {
			err = c.CheckOutcome(item.Manager, call.Tool, args)
		}
		if err != nil {
			return run.ToolOutcome{Status: run.ToolRefused, Reason: err.Error()}, nil
		}
		return run.ToolOutcome{Status: run.ToolOK, Result: []byte(`{"accepted":true}`)}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := c.SessionAdmission(ctx, j, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Open(ctx, admission, io); err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("open-"+string(id), time.Second))
	var quoted []run.Payload
	for _, ref := range c.StepQuotes(item.Manager) {
		quoted = append(quoted, j.blobs[ref])
	}
	facts := sourceFacts(inputs)
	facts.Quoted = quoted
	step, err := c.PrepareTaskStepCommand(meta("prepare-step-"+string(id)), run.StepPlan{Session: id, ID: run.StepID("step-" + string(id)), SessionBinding: cmd.Session.Plan.Binding()},
		run.Reservation{ID: "invoke-" + string(id), Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Minute}, inputs, facts)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, step)
	stepAdmission, err := c.StepAdmission(ctx, j, step.Session.Step)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Step(ctx, stepAdmission, io); err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("invoke-"+string(id), time.Second))
	stop, err := c.StopRequest(id, "the step ended")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Stop(ctx, stop, io); err != nil {
		t.Fatal(err)
	}
	ref, ok := c.OutcomeInput(item.Manager)
	if !ok {
		t.Fatal("no answered report")
	}
	conclude, err := c.ConcludeStepCommand(meta("conclude-"+string(id)), item.Manager, run.StepResult{Input: j.blobs[ref], Aggregate: c.AggregateVersion()})
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, conclude)
}
