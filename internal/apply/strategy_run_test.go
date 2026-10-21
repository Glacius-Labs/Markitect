package apply_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// preservingAgents follows the Orchestrator's protocol for an Apply in which
// nothing needs to change: every Manager decides its children's proposals,
// submits its basis (with a plan in a planning Pass) and its complete result
// preserving each own obligation, and every reviewer cites an inspected source
// for each own obligation. unstopped names a Manager whose first work turn
// reports no observed stop of its writers. The reviewer of finds raises one
// genuine finding on its first obligation in its first Verify, naming the
// earliest correction correction (implement when empty), and disproves it in
// the next; a refused correction is kept in refused and the finding raised for
// implement instead. lookups names the Definitions a Manager looks up, with its
// Brief, in its first turn; their answers are kept, after edits changed the
// stand's files. compacted, forgets and fails are keyed by "<turn kind>
// <Manager>": the first such turn reports a compaction, the first continued one
// comes back in another conversation, or the first one fails before any tool
// call; marks keeps which turn did what. A conversation otherwise continues.
// The Model assessment that precedes the work accepts the model: no scripted
// event happens in its turns, unless gap makes the reviewer of finds raise its
// finding as a model_gap in the first assessment instead of the first Verify,
// which nobody disputes. abort names a Manager whose first work turn reports
// a model problem with abort.
// Any other refused tool call fails the test, so the script cannot drift from
// the engine's protocol unnoticed.
type preservingAgents struct {
	t           *testing.T
	obligations map[string][]string
	unstopped   string
	finds       string
	correction  string
	refused     []string
	lookups     map[string][]string
	edits       map[string]string
	ws          *treeWorkspaces
	compacted   map[string]bool
	forgets     map[string]bool
	fails       map[string]bool
	mu          sync.Mutex
	turns       []run.Turn
	marks       map[string]string
	answers     map[string]string
	raised      bool
	open        bool
	ids         []string
	assessing   map[string]bool // conversation -> it is a Model assessment
	gap         bool
	abort       string
}

var ownFindingRE = regexp.MustCompile(`Finding (F\d+) \[`)

// once reports and consumes a scripted event of the turn.
func (a *preservingAgents) once(events map[string]bool, turn run.Turn, mark string) bool {
	key := string(turn.Kind) + " " + turn.Manager
	if !events[key] || mark == "lost" && turn.Resume == "" {
		return false
	}
	delete(events, key)
	if a.marks == nil {
		a.marks = map[string]string{}
	}
	a.marks[turn.ID] = mark
	return true
}

func (a *preservingAgents) Run(ctx context.Context, turn run.Turn, tools run.ToolHandler) (run.TurnResult, error) {
	a.mu.Lock()
	a.turns = append(a.turns, turn)
	if a.assessing == nil {
		a.assessing = map[string]bool{}
	}
	assessing := a.assessing[turn.Resume] || turn.Resume == "" && strings.Contains(turn.Prompt, "You assess the bound model")
	failed := !assessing && a.once(a.fails, turn, "failed")
	a.mu.Unlock()
	if failed {
		return run.TurnResult{Conversation: "conv-" + turn.Home, Stopped: true}, errors.New("the scripted turn failed")
	}
	if turn.Kind == run.VerifyTurn && (!turn.ReadOnly || slices.ContainsFunc(turn.Mounts, func(m run.PathMount) bool { return m.Writable })) {
		a.t.Errorf("verify turn %s can write", turn.ID)
	}
	call := func(name string, args map[string]any) string {
		data, _ := json.Marshal(args)
		answer := tools(ctx, name, data)
		if answer.Refused {
			a.t.Errorf("turn %s: %s %v refused: %s", turn.ID, name, args, answer.Text)
		}
		return answer.Text
	}
	a.mu.Lock()
	var lookups []string
	compacted := false
	if !assessing {
		lookups = a.lookups[turn.Manager]
		delete(a.lookups, turn.Manager)
		compacted = a.once(a.compacted, turn, "compacted")
	}
	a.mu.Unlock()
	if len(lookups) > 0 && a.ws != nil {
		a.ws.edit(turn.Workspace, a.edits)
	}
	if len(lookups) > 0 {
		brief := call("look_up", map[string]any{"topic": "brief"})
		a.mu.Lock()
		if a.answers == nil {
			a.answers = map[string]string{}
		}
		a.answers[turn.Manager+" brief"] = brief
		a.mu.Unlock()
		for _, id := range lookups {
			text := call("look_up", map[string]any{"topic": "definition", "id": id})
			a.mu.Lock()
			a.answers[turn.Manager+" "+id] = text
			a.mu.Unlock()
		}
	}
	for _, m := range decideRE.FindAllStringSubmatch(turn.Prompt, -1) {
		call("report", map[string]any{"kind": "accept", "proposal": m[1]})
	}
	a.mu.Lock()
	aborts := turn.Kind == run.ExecuteTurn && turn.Manager == a.abort
	if aborts {
		a.abort = ""
	}
	a.mu.Unlock()
	if aborts {
		call("report", map[string]any{"kind": "problem", "category": "model", "abort": true, "text": "lending/loan-policy and lending/borrow state different loan limits."})
		return run.TurnResult{Conversation: "conv-" + turn.Home, Usage: run.Usage{Model: "m", InputTokens: 10, OutputTokens: 5, ProviderTurns: 1}, Stopped: true}, nil
	}
	plans := slices.ContainsFunc(turn.Tools, func(s run.ToolSpec) bool {
		return s.Name == "report" && strings.Contains(string(s.InputSchema), `"plan":`)
	})
	submission := func(kind string) map[string]any {
		report := map[string]any{"kind": kind, "summary": kind + " of " + turn.Manager + "."}
		raise := false
		if turn.Kind == run.VerifyTurn && turn.Manager == a.finds && assessing == a.gap {
			a.mu.Lock()
			// The assignment of a later Verify names the findings it raised.
			if _, earlier, found := strings.Cut(turn.Prompt, "Open findings you raised in earlier Passes"); found {
				for _, m := range ownFindingRE.FindAllStringSubmatch(earlier, -1) {
					if !slices.Contains(a.ids, m[1]) {
						a.ids = append(a.ids, m[1])
					}
				}
			}
			switch {
			case kind != "result":
			case !a.raised:
				raise, a.raised, a.open = true, true, true
			case len(a.ids) > 0:
				a.open = false
				var closed []map[string]any
				for _, id := range a.ids {
					closed = append(closed, map[string]any{"id": id, "as": "disproved", "reason": "The scripted reviewer judged it wrongly.", "evidence": "The unchanged candidate meets the obligation."})
				}
				report["closed"], a.ids = closed, nil
			}
			a.mu.Unlock()
		}
		var covered []map[string]any
		for i, id := range a.obligations[turn.Manager] {
			entry := map[string]any{"definition": id, "status": "preserved", "note": "The existing artifacts meet it.", "conditions": "scripted static fixture contract; no public behavior asserted"}
			switch {
			case raise && i == 0:
				entry["status"], entry["note"] = "open", "The scripted finding holds it open."
				correction := a.correction
				switch {
				case a.gap:
					correction = "model_gap"
				case correction == "":
					correction = "implement"
				}
				report["findings"] = []map[string]any{{"owner": turn.Manager, "obligation": id, "expected": "the stand keeps " + id, "observed": "A scripted counterexample.", "blocking": true, "correction": correction}}
			case !plans:
				entry["evidence"] = []map[string]any{{"kind": "static", "claim": "the stand keeps " + id, "source": ".markitect/execution.yaml", "location": "the whole file", "reasoning": "This scripted reviewer inspected the stand."}}
			}
			covered = append(covered, entry)
		}
		if !assessing && (turn.Manager == "." || turn.Manager == "lending") {
			report["integration"] = map[string]any{"status": "done", "conditions": "scripted static composition", "note": "protocol evidence only", "evidence": []map[string]any{{"kind": "static", "authority": "pass_mandate", "claim": "fixture layout composes", "source": ".markitect/execution.yaml", "location": "ownership", "reasoning": "static metadata, no executable claim"}}}
		}
		if covered != nil {
			report["obligations"] = covered
		}
		if plans {
			report["plan"] = map[string]any{"approach": "Preserve the conforming stand.", "artifacts": []string{}, "inputs": []any{}, "outputs": []any{}, "dependencies": []any{}, "order": "No order is needed.", "blockers": []string{}, "assumptions": []string{}}
		}
		if kind == "result" {
			report["verdict"] = "satisfied"
			// A genuine finding within its responsibility keeps a reviewer's
			// result from being satisfied.
			a.mu.Lock()
			within := a.open && (turn.Manager == "." || strings.HasPrefix(a.finds, turn.Manager+"/"))
			a.mu.Unlock()
			if raise || turn.Kind == run.VerifyTurn && within {
				report["verdict"] = "needs_correction"
			}
		}
		if kind == "basis" && strings.Contains(turn.Prompt, "MODEL-FIRST BASIS:") {
			for _, entry := range covered {
				entry["status"], entry["note"] = "open", "model-derived static conditions; candidate assessment pending"
				delete(entry, "evidence")
			}
			if entry, ok := report["integration"].(map[string]any); ok {
				entry["status"], entry["note"] = "open", "model-derived static composition; candidate assessment pending"
				delete(entry, "evidence")
			}
		}
		return report
	}
	_, rest, _ := strings.Cut(turn.Prompt, "\nNow: ")
	now, _, _ := strings.Cut(rest, "\n")
	switch {
	case strings.HasPrefix(now, "work on your own contribution"):
		call("report", submission("basis"))
	case strings.HasPrefix(now, "submit your complete result"), strings.HasPrefix(now, "integrate your children"):
		report := submission("result")
		findings, _ := report["findings"].([]map[string]any)
		if len(findings) == 1 && findings[0]["correction"] != "implement" {
			data, _ := json.Marshal(report)
			answer := tools(ctx, "report", data)
			if !answer.Refused {
				break
			}
			a.mu.Lock()
			a.refused = append(a.refused, answer.Text)
			a.mu.Unlock()
			findings[0]["correction"] = "implement"
		}
		call("report", report)
	}
	stopped := true
	if turn.Manager == a.unstopped && turn.Kind == run.ExecuteTurn {
		a.mu.Lock()
		stopped, a.unstopped = false, ""
		a.mu.Unlock()
	}
	conversation := turn.Resume
	if conversation == "" {
		conversation = "conv-" + turn.Home
	}
	a.mu.Lock()
	if !assessing && a.once(a.forgets, turn, "lost") {
		// The provider did not continue the conversation it was asked to.
		conversation = "lost-" + turn.Home
	}
	a.assessing[conversation] = assessing
	a.mu.Unlock()
	return run.TurnResult{Conversation: conversation, Usage: run.Usage{Model: "m", InputTokens: 10, OutputTokens: 5, ProviderTurns: 1}, Stopped: stopped, Compacted: compacted}, nil
}

// passingRunner runs every check of a candidate successfully.
type passingRunner struct {
	mu   sync.Mutex
	runs []run.CheckRequest
}

func (r *passingRunner) Run(_ context.Context, req run.CheckRequest) (run.CheckReceipt, error) {
	r.mu.Lock()
	r.runs = append(r.runs, req)
	r.mu.Unlock()
	return run.CheckReceipt{ID: req.ID, ImageID: req.ImageID, Tree: req.Tree.Digest(), Started: true, Exited: true}, nil
}

// treeWorkspaces keeps each volume's exact tree, modes included; the scripted
// Agents change nothing but scripted edits.
type treeWorkspaces struct {
	mu     sync.Mutex
	trees  map[run.WorkspaceID]snapshot.Tree
	closed []run.WorkspaceID
}

// edit replaces files of the volume a turn mounts, keeping their modes.
func (w *treeWorkspaces) edit(volume string, files map[string]string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for id, tree := range w.trees {
		if "vol-"+string(id) != volume {
			continue
		}
		var entries []snapshot.Entry
		for _, e := range tree.Entries() {
			if content, ok := files[e.Path().String()]; ok {
				e, _ = snapshot.NewEntry(e.Path(), e.Mode(), []byte(content))
			}
			entries = append(entries, e)
		}
		w.trees[id], _ = snapshot.NewTree(entries)
	}
}

func (w *treeWorkspaces) Materialize(_ context.Context, spec run.WorkspaceSpec) (run.WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.trees == nil {
		w.trees = map[run.WorkspaceID]snapshot.Tree{}
	}
	w.trees[spec.ID] = spec.Tree
	return run.WorkspaceObservation{ID: spec.ID, Volume: "vol-" + string(spec.ID), Tree: spec.Tree.Digest()}, nil
}

func (w *treeWorkspaces) Harvest(_ context.Context, r run.HarvestRequest) (run.WorkspaceHarvest, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	tree, ok := w.trees[r.ID]
	if !ok {
		return run.WorkspaceHarvest{}, errors.New("no such volume")
	}
	return run.WorkspaceHarvest{ID: r.ID, Tree: tree}, nil
}

func (w *treeWorkspaces) Peek(ctx context.Context, r run.HarvestRequest) (run.WorkspaceHarvest, error) {
	return w.Harvest(ctx, r)
}

func (w *treeWorkspaces) Close(_ context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = append(w.closed, id)
	delete(w.trees, id)
	return run.WorkspaceObservation{ID: id, Closed: true}, nil
}

// strategyRequest prepares the library fixture with the given strategy through
// the product preparation, as every Apply does.
func strategyRequest(t *testing.T, strategy string) apply.ApplyRequest {
	t.Helper()
	return strategyRequestWithTurns(t, strategy, 120)
}

// strategyRequestWithTurns also sets the budget's agent turns.
func strategyRequestWithTurns(t *testing.T, strategy string, turns int) apply.ApplyRequest {
	t.Helper()
	execution := strings.Replace(withOwnership(t, libraryOwnershipYAML), "strategy: direct", "strategy: "+strategy, 1)
	execution = strings.Replace(execution, "agentTurns: 120", fmt.Sprintf("agentTurns: %d", turns), 1)
	selection, raw := executionInputs(t, execution)
	libraryLayoutFiles(t, &selection, &raw)
	return apply.ApplyRequest{RunID: "run-1", Prepared: preparedAttempt(t, selection, raw), Selection: apply.InputSelection{Repository: "/repo"}, Build: "test-build",
		MaxEntries: 1000, MaxBytes: 1 << 20, CheckRequest: checkRequest, RequireVolumeSubpaths: func(context.Context) error { return nil }}
}

// Both strategies run their own work sequence on the same base process and the
// same shared guards: the Model check and File check at every Pass boundary, the
// required check on the immutable candidate, a fresh read-only Verify, and the
// guarded publication of the verified candidate. The binding of the strategy,
// the effective settings and the build is recorded before any Agent work.
func TestBothStrategiesShareTheGatesVerifyAndPublication(t *testing.T) {
	compositions := map[string]string{}
	t.Cleanup(func() {
		if d, s := compositions["direct"], compositions["staged"]; !strings.HasPrefix(d, "sha256:") || !strings.HasPrefix(s, "sha256:") || d == s {
			t.Errorf("composition digests direct %q, staged %q", d, s)
		}
	})
	for _, tc := range []struct {
		strategy string
		work     []run.Purpose
	}{
		{"direct", []run.Purpose{run.Implement}},
		{"staged", []run.Purpose{run.Plan, run.Prepare, run.Implement}},
	} {
		t.Run(tc.strategy, func(t *testing.T) {
			req := strategyRequest(t, tc.strategy)
			req.Deadline = time.Now().Add(20 * time.Minute)
			req.WorkDeadline = req.Deadline.Add(-12 * time.Minute)
			cfg, err := apply.BuildApplyConfig(req, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			agents := &preservingAgents{t: t, obligations: map[string][]string{}}
			for _, m := range cfg.Team {
				agents.obligations[m.Path] = m.Obligations
			}
			workspaces, runner, evidence := &treeWorkspaces{}, &passingRunner{}, &memoryEvidence{}
			publisher := &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationNoChange}}
			req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = agents, workspaces, runner, evidence, publisher
			result, err := apply.Apply(context.Background(), req)
			if err != nil || result.Outcome.Status != run.Verified {
				t.Fatalf("outcome %s %s: %v", result.Outcome.Status, result.Outcome.Reason, err)
			}
			var purposes []run.Purpose
			for _, p := range result.Outcome.Passes {
				purposes = append(purposes, p.Purpose)
				if !p.Sealed || p.Verdict != run.Satisfied {
					t.Errorf("pass %s %s: sealed %v verdict %s", p.ID, p.Purpose, p.Sealed, p.Verdict)
				}
			}
			if want := append(append([]run.Purpose{run.Assess}, tc.work...), run.Verify); !slices.Equal(purposes, want) {
				t.Fatalf("passes %v, want %v", purposes, want)
			}
			gates := map[string]int{}
			for _, e := range evidence.events {
				if e.Kind == "gate" {
					gates[e.Pass]++
				}
			}
			// The Model assessment judges the model of the checked capture; every
			// other Pass runs the gates.
			for _, p := range result.Outcome.Passes {
				if (gates[p.ID] == 0) != (p.Purpose == run.Assess) {
					t.Errorf("pass %s (%s) ran %d gates", p.ID, p.Purpose, gates[p.ID])
				}
			}
			if len(runner.runs) != 1 || runner.runs[0].Tree.Digest() != result.Outcome.Candidate.Digest() {
				t.Fatalf("the required check ran %d times, not once on the candidate", len(runner.runs))
			}
			if len(publisher.requests) != 1 || len(publisher.requests[0].Changes) != 0 || result.Receipt == nil || result.Receipt.Outcome != apply.PublicationNoChange || !slices.Contains(workspaces.closed, "run-1-work") {
				t.Fatalf("publication %+v receipt %+v closed %v", publisher.requests, result.Receipt, workspaces.closed)
			}
			b := result.Binding
			if !cfg.Budget.WorkDeadline.Equal(req.WorkDeadline) || !b.Budget.WorkDeadline.Equal(req.WorkDeadline) || !b.Budget.Deadline.Equal(req.Deadline) {
				t.Fatalf("effective work/review cutoffs lost: %+v", b.Budget)
			}
			compositions[tc.strategy] = b.Composition
			if b.Entry != "apply" || string(b.Strategy) != tc.strategy || !slices.Equal(b.Work, append([]run.Purpose{run.Assess}, tc.work...)) || b.Build != "test-build" || !strings.HasPrefix(b.Execution, ".markitect/execution.yaml@") || !strings.HasPrefix(b.Defaults, "product:apply/defaults.yaml@") {
				t.Fatalf("binding %+v", b)
			}
			if len(b.Managers) != len(cfg.Team) || b.Managers[0].Execute.Profile != "library-agent" || b.Managers[0].Execute.Identity == "" || len(b.Checks) != 1 || b.Checks[0].Name != "project-tests" || b.Budget.Passes != 12 {
				t.Fatalf("effective settings %+v", b)
			}
			if data, _ := evidence.events[0].Data.(map[string]any); evidence.events[0].Kind != "run.start" || !reflect.DeepEqual(data["binding"], b) {
				t.Fatalf("run.start does not bind %+v: %+v", b, evidence.events[0])
			}
		})
	}
}

// Writer cessation is a shared guard: in either strategy a work turn without an
// observed stop of its writers ends the run Incomplete, and nothing is
// accepted from it or published.
func TestBothStrategiesStopWithoutObservedWriterCessation(t *testing.T) {
	for _, strategy := range []string{"direct", "staged"} {
		t.Run(strategy, func(t *testing.T) {
			req := strategyRequest(t, strategy)
			cfg, err := apply.BuildApplyConfig(req, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			agents := &preservingAgents{t: t, obligations: map[string][]string{}, unstopped: "."}
			for _, m := range cfg.Team {
				agents.obligations[m.Path] = m.Obligations
			}
			publisher := &fakePublisher{}
			req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = agents, &treeWorkspaces{}, &passingRunner{}, &memoryEvidence{}, publisher
			result, _ := apply.Apply(context.Background(), req)
			if result.Outcome.Status != run.Incomplete || !strings.Contains(result.Outcome.Reason, "quiescence is uncertain") || len(publisher.requests) != 0 {
				t.Fatalf("outcome %s %q, publications %d", result.Outcome.Status, result.Outcome.Reason, len(publisher.requests))
			}
			for _, p := range result.Outcome.Passes {
				for _, m := range p.Managers {
					if p.Purpose != run.Assess && m.Manager == "." && m.Basis != "" {
						t.Fatalf("Root's basis was accepted from an unstopped turn: %+v", m)
					}
				}
			}
		})
	}
}

// A genuine Verify finding takes the correction path of the strategy, a new
// Implement in both here because the finding names implement, and a complete
// fresh Verify closes it before anything is published.
func TestBothStrategiesCorrectAFindingThroughAFreshVerify(t *testing.T) {
	for _, tc := range []struct {
		strategy, correction string
		passes               []run.Purpose
		refused              bool
	}{
		// direct knows only implement: a finding that names plan is refused.
		{"direct", "plan", []run.Purpose{run.Assess, run.Implement, run.Verify, run.Implement, run.Verify}, true},
		// staged restarts at the earliest purpose the finding names.
		{"staged", "prepare", []run.Purpose{run.Assess, run.Plan, run.Prepare, run.Implement, run.Verify, run.Prepare, run.Implement, run.Verify}, false},
	} {
		t.Run(tc.strategy, func(t *testing.T) {
			// Seven Passes of six Managers need more turns than the fixture's 120.
			req := strategyRequestWithTurns(t, tc.strategy, 400)
			cfg, err := apply.BuildApplyConfig(req, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			agents := &preservingAgents{t: t, obligations: map[string][]string{}, finds: "lending/reminders", correction: tc.correction}
			for _, m := range cfg.Team {
				agents.obligations[m.Path] = m.Obligations
			}
			publisher := &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationNoChange}}
			req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = agents, &treeWorkspaces{}, &passingRunner{}, &memoryEvidence{}, publisher
			result, err := apply.Apply(context.Background(), req)
			if err != nil || result.Outcome.Status != run.Verified {
				t.Fatalf("outcome %s %s: %v", result.Outcome.Status, result.Outcome.Reason, err)
			}
			var purposes []run.Purpose
			var verdicts []run.Verdict
			for _, p := range result.Outcome.Passes {
				purposes, verdicts = append(purposes, p.Purpose), append(verdicts, p.Verdict)
			}
			verify := slices.Index(purposes, run.Verify)
			if !slices.Equal(purposes, tc.passes) || verify < 0 || verdicts[verify] != run.NeedsCorrection || verdicts[len(tc.passes)-1] != run.Satisfied {
				t.Fatalf("passes %v verdicts %v", purposes, verdicts)
			}
			if refused := len(agents.refused) == 1 && strings.Contains(agents.refused[0], "correction"); refused != tc.refused {
				t.Errorf("correction %s refused %v: %v", tc.correction, refused, agents.refused)
			}
			if len(result.Outcome.Findings) != 1 || !result.Outcome.Findings[0].Closed || result.Outcome.Findings[0].ClosedAs != "disproved" || len(publisher.requests) != 1 {
				t.Fatalf("findings %+v, publications %d", result.Outcome.Findings, len(publisher.requests))
			}
		})
	}
}

// An unknown strategy refuses in the preparation, before any Run binds it, and
// a request without the build identity is refused before any effect.
func TestApplyRefusesAnUnknownStrategyAndAMissingBuild(t *testing.T) {
	selection, raw := executionInputs(t, strings.Replace(withOwnership(t, libraryOwnershipYAML), "strategy: direct", "strategy: single-agent", 1))
	result := registryModel(t, selection, raw).DecodeExecution()
	if result.Status() == "passed" || !slices.ContainsFunc(result.Diagnostics(), func(d apply.ExecutionDiagnostic) bool { return d.Code == "execution.strategy" }) {
		t.Fatalf("an unknown strategy decoded: %s %+v", result.Status(), result.Diagnostics())
	}
	selection, raw = executionInputs(t, strings.Replace(withOwnership(t, libraryOwnershipYAML), "strategy: direct\n", "", 1))
	result = registryModel(t, selection, raw).DecodeExecution()
	if result.Status() == "passed" || !slices.ContainsFunc(result.Diagnostics(), func(d apply.ExecutionDiagnostic) bool {
		return d.Code == "execution.field_missing" && d.Source.Field == "/strategy"
	}) {
		t.Fatalf("a missing strategy decoded: %s %+v", result.Status(), result.Diagnostics())
	}
	req := strategyRequest(t, "direct")
	req.Build = " "
	req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = failingAgents{}, &treeWorkspaces{}, &passingRunner{}, &memoryEvidence{}, &fakePublisher{}
	if result, err := apply.Apply(context.Background(), req); err == nil || !strings.Contains(err.Error(), "implementation identity") || result.Outcome.Status != "" {
		t.Fatalf("a run without a build identity: %+v %v", result, err)
	}
}
