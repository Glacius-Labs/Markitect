package apply_test

import (
	"context"
	"encoding/json"
	"errors"
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

// memoryWorkspaces keeps volumes in memory; Agents write into them by volume name.
type memoryWorkspaces struct {
	mu     sync.Mutex
	files  map[string]map[string]string // volume -> path -> content
	ids    map[run.WorkspaceID]string
	closed []run.WorkspaceID
	refuse run.WorkspaceID // Materialize fails for this volume
}

func newMemoryWorkspaces() *memoryWorkspaces {
	return &memoryWorkspaces{files: map[string]map[string]string{}, ids: map[run.WorkspaceID]string{}}
}

func (w *memoryWorkspaces) Materialize(_ context.Context, spec run.WorkspaceSpec) (run.WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if spec.ID == w.refuse {
		return run.WorkspaceObservation{}, errors.New("no space left on device")
	}
	volume := "vol-" + string(spec.ID)
	w.files[volume], w.ids[spec.ID] = map[string]string{}, volume
	for _, e := range spec.Tree.Entries() {
		w.files[volume][e.Path().String()] = string(e.Bytes())
	}
	return run.WorkspaceObservation{ID: spec.ID, Volume: volume, Tree: spec.Tree.Digest()}, nil
}

func (w *memoryWorkspaces) tree(id run.WorkspaceID) (snapshot.Tree, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	volume, ok := w.ids[id]
	if !ok {
		return snapshot.Tree{}, errors.New("no such volume")
	}
	var entries []snapshot.Entry
	for name, content := range w.files[volume] {
		p, err := snapshot.NewPath(name)
		if err != nil {
			return snapshot.Tree{}, err
		}
		e, err := snapshot.NewEntry(p, snapshot.Regular, []byte(content))
		if err != nil {
			return snapshot.Tree{}, err
		}
		entries = append(entries, e)
	}
	return snapshot.NewTree(entries)
}

func (w *memoryWorkspaces) Harvest(_ context.Context, r run.HarvestRequest) (run.WorkspaceHarvest, error) {
	tree, err := w.tree(r.ID)
	return run.WorkspaceHarvest{ID: r.ID, Tree: tree}, err
}

func (w *memoryWorkspaces) Peek(ctx context.Context, r run.HarvestRequest) (run.WorkspaceHarvest, error) {
	return w.Harvest(ctx, r)
}

func (w *memoryWorkspaces) Close(_ context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = append(w.closed, id)
	delete(w.ids, id)
	return run.WorkspaceObservation{ID: id, Closed: true}, nil
}

func (w *memoryWorkspaces) write(volume string, files map[string]string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for name, content := range files {
		w.files[volume][name] = content
	}
}

// scriptedAgents follows the Orchestrator's protocol for every Manager and
// reviewer: it decides the proposals of its children, answers the questions
// addressed to it, writes the scripted files in its first working turn, submits
// its basis and then its complete result. A Manager can ask the user before it
// writes, and a reviewer can raise findings with its result. A turn that is
// refused a tool call fails the test, so the script cannot drift from the
// engine's protocol unnoticed.
type scriptedAgents struct {
	t        *testing.T
	ws       *memoryWorkspaces
	writes   map[string]map[string]string // "<run> <manager>" -> files written in the first working turn
	findings map[string][]map[string]any  // "<run> <manager>" -> findings a reviewer raises with its result
	asks     map[string]string            // "<run> <manager>" -> a question beyond the tree it tries before it works
	mu       sync.Mutex
	turns    []run.Turn
	refusals map[string]string // key -> the answer to that question
	assessed map[string]bool   // conversation -> it is a Model assessment
}

var (
	scriptedRun = regexp.MustCompile(`^(x-1-(?:structure|model))-p\d+-`)
	decideRE    = regexp.MustCompile(`- decide (P\d+),`)
	answerRE    = regexp.MustCompile(`- answer (Q\d+) from`)
)

func (a *scriptedAgents) Run(ctx context.Context, turn run.Turn, tools run.ToolHandler) (run.TurnResult, error) {
	a.mu.Lock()
	a.turns = append(a.turns, turn)
	a.mu.Unlock()
	if turn.Kind == run.VerifyTurn && (!turn.ReadOnly || slices.ContainsFunc(turn.Mounts, func(m run.PathMount) bool { return m.Writable })) {
		a.t.Errorf("verify turn %s can write", turn.ID)
	}
	key := scriptedRun.FindStringSubmatch(turn.ID)[1] + " " + turn.Manager
	a.mu.Lock()
	assessing := a.assessed[turn.Resume] || turn.Resume == "" && strings.Contains(turn.Prompt, "You assess the bound model")
	a.mu.Unlock()
	call := func(name string, args map[string]any) run.ToolAnswer {
		data, _ := json.Marshal(args)
		answer := tools(ctx, name, data)
		if answer.Refused {
			a.t.Errorf("turn %s: %s %v refused: %s", turn.ID, name, args, answer.Text)
		}
		return answer
	}
	for _, m := range decideRE.FindAllStringSubmatch(turn.Prompt, -1) {
		call("report", map[string]any{"kind": "accept", "proposal": m[1]})
	}
	for _, m := range answerRE.FindAllStringSubmatch(turn.Prompt, -1) {
		// Nobody asks a Manager anything in these scripts but the Orchestrator's own forwards.
		a.t.Errorf("turn %s: unexpected question %s", turn.ID, m[1])
	}
	_, rest, _ := strings.Cut(turn.Prompt, "\nNow: ")
	now, _, _ := strings.Cut(rest, "\n")
	switch {
	case strings.HasPrefix(now, "work on your own contribution"):
		if question := a.asks[key]; question != "" && turn.Kind == run.ExecuteTurn && a.refusals[key] == "" {
			data, _ := json.Marshal(map[string]any{"to": "user", "kind": "question", "text": question})
			answer := tools(ctx, "send", data)
			if !answer.Refused {
				a.t.Errorf("turn %s: a question beyond the tree was not refused: %s", turn.ID, answer.Text)
			}
			a.mu.Lock()
			a.refusals[key] = answer.Text
			a.mu.Unlock()
		}
		if turn.Kind == run.ExecuteTurn {
			a.ws.write(turn.Workspace, a.writes[key])
		}
		call("report", map[string]any{"kind": "basis", "summary": "Basis of " + turn.Manager + "."})
	case strings.HasPrefix(now, "submit your complete result"), strings.HasPrefix(now, "integrate your children"):
		result := map[string]any{"kind": "result", "summary": "Described " + turn.Manager + ".", "verdict": "satisfied"}
		if turn.Kind == run.VerifyTurn {
			result["summary"] = "The model matches the artifacts."
			if findings := a.findings[key]; len(findings) > 0 && !assessing {
				result["findings"] = findings
			}
		}
		call("report", result)
	}
	conversation := turn.Resume
	if conversation == "" {
		conversation = "conv-" + turn.ID
	}
	a.mu.Lock()
	a.assessed[conversation] = assessing
	a.mu.Unlock()
	return run.TurnResult{Conversation: conversation, Usage: run.Usage{Model: "m", InputTokens: 10, OutputTokens: 5, ProviderTurns: 1}, Stopped: true}, nil
}

// bothPhases scripts an Extract whose two phases both verify: Root writes the
// structure proposal, then every Manager registers the artifacts of its area.
func bothPhases(t *testing.T) (apply.ExtractRequest, *scriptedAgents, *memoryWorkspaces, *fakePublisher, snapshot.Tree, snapshot.Tree, snapshot.Tree) {
	t.Helper()
	req := extractRequest(t)
	start := extractStart(t)
	complete := registeredModel(t, start)
	proposed := proposedTree(t, start)
	// Each Manager writes what the verified candidates contain for its area.
	diff := func(from, to snapshot.Tree, accept func(string) bool) map[string]string {
		files := map[string]string{}
		for _, change := range snapshot.Compare(from, to) {
			if after, ok := change.After(); ok && accept(change.Path().String()) {
				files[change.Path().String()] = string(after.Bytes())
			}
		}
		return files
	}
	workspaces := newMemoryWorkspaces()
	agents := &scriptedAgents{t: t, ws: workspaces, refusals: map[string]string{}, assessed: map[string]bool{}, writes: map[string]map[string]string{
		"x-1-structure .": diff(start, proposed, func(string) bool { return true }),
	}, findings: map[string][]map[string]any{
		"x-1-model lending": {{"owner": "lending", "subject": "artifact", "definition": "lending/lending", "obligation": "Loans are limited", "observed": "borrow.go admits a sixth loan.", "blocking": true}},
	}}
	for _, owner := range []string{".", "catalog", "library", "lending", "lending/reminders", "members"} {
		shard := ".markitect/registry/" + owner + "/registry.yaml"
		if owner == "." {
			shard = ".markitect/registry/registry.yaml"
		}
		agents.writes["x-1-model "+owner] = diff(proposed, complete, func(name string) bool { return name == shard })
	}
	publisher := &fakePublisher{receipt: apply.PublicationReceipt{ID: "x-1", Outcome: apply.PublicationApplied}}
	req.Agents, req.Workspaces, req.Evidence, req.Publisher = agents, workspaces, &memoryEvidence{}, publisher
	return req, agents, workspaces, publisher, start, proposed, complete
}

func TestExtractRunsBothPhasesAndPublishesTheModel(t *testing.T) {
	req, agents, workspaces, publisher, start, proposed, complete := bothPhases(t)
	evidence := &memoryEvidence{}
	var removed []string
	req.Evidence = evidence
	req.RemoveHome = func(_ context.Context, home string) error {
		removed = append(removed, home)
		return nil
	}
	result, err := apply.Extract(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Structure.Status != run.Verified || result.Model == nil || result.Model.Status != run.Verified {
		t.Fatalf("structure %s %s; model %+v", result.Structure.Status, result.Structure.Reason, result.Model)
	}
	if !result.Structure.Candidate.Equal(proposed) || !result.Model.Candidate.Equal(complete) {
		t.Fatal("the verified candidates are not the proposals the Agents wrote")
	}
	// Each phase is one work Pass of its own purpose and one independent Verify;
	// every Manager takes two turns in each: its basis and its complete result.
	if len(result.Structure.Passes) != 2 || len(result.Model.Passes) != 2 {
		t.Fatalf("passes %d/%d", len(result.Structure.Passes), len(result.Model.Passes))
	}
	for i, tc := range []struct {
		pass    run.PassRecord
		purpose run.Purpose
		review  bool
	}{
		{result.Structure.Passes[0], "structure", false}, {result.Structure.Passes[1], run.Verify, true},
		{result.Model.Passes[0], "model", false}, {result.Model.Passes[1], run.Verify, true},
	} {
		if tc.pass.Purpose != tc.purpose || tc.pass.Review != tc.review || tc.pass.Verdict != run.Satisfied || !tc.pass.Sealed {
			t.Errorf("pass %d: %+v", i, tc.pass)
		}
		for _, m := range tc.pass.Managers {
			if m.State != run.Done || m.Verdict != run.Satisfied || m.Basis == "" || m.Result == "" {
				t.Errorf("pass %d: Manager %s did not complete: %+v", i, m.Manager, m)
			}
		}
	}
	if len(result.Structure.Passes[0].Managers) != 1 || len(result.Model.Passes[0].Managers) != 6 {
		t.Errorf("the structure phase is Root alone and the model phase the proposed tree: %d/%d", len(result.Structure.Passes[0].Managers), len(result.Model.Passes[0].Managers))
	}
	// Parents take further turns to decide their children's proposals, so the
	// model phase has at least the two turns of each of its six Managers in both
	// Passes.
	if len(result.Structure.Turns) != 4 || len(result.Model.Turns) < 24 || result.Model.Usage.InputTokens != int64(10*len(result.Model.Turns)) {
		t.Fatalf("turns %d/%d usage %+v", len(result.Structure.Turns), len(result.Model.Turns), result.Model.Usage)
	}
	if len(result.ProposedChecks) != 0 {
		t.Errorf("no check was proposed: %+v", result.ProposedChecks)
	}
	// The account lists the Packages of the verified model in path order, with the
	// title of the overview Definition when it has one.
	var packages []string
	for _, p := range result.Packages {
		packages = append(packages, p.Path)
		if want := map[string]string{"library": "Small library"}[p.Path]; p.Title != want {
			t.Errorf("title of %s: %q, want %q", p.Path, p.Title, want)
		}
	}
	if want := []string{"catalog", "lending", "lending/reminders", "library", "members"}; !slices.Equal(packages, want) {
		t.Errorf("packages %v, want %v", packages, want)
	}
	// The reviewer called the artifact finding blocking; it is stored advisory and
	// the model verified anyway.
	deviations := apply.ExtractDeviations(result.Model.Findings)
	if len(deviations) != 1 || deviations[0].Definition != "lending/lending" || !slices.Equal(deviations[0].Owners, []string{"lending"}) || !slices.Equal(deviations[0].RaisedBy, []string{"lending"}) || len(deviations[0].Observed) != 1 || !strings.Contains(deviations[0].Observed[0], "sixth loan") {
		t.Fatalf("deviations %+v", deviations)
	}
	for _, f := range result.Model.Findings {
		if f.Subject == "artifact" && f.Blocking {
			t.Errorf("a blocking artifact finding: %+v", f)
		}
	}
	// Publication: the guarded Publisher gets exactly the model, registry and layout changes.
	if result.Receipt == nil || result.Receipt.Outcome != apply.PublicationApplied || len(publisher.requests) != 1 || result.RetainedWork != "" {
		t.Fatalf("receipt %+v retained %q", result.Receipt, result.RetainedWork)
	}
	published := publisher.requests[0]
	if published.Guard != req.Prepared.Structural().Model().Inputs().Guard || published.ID != "x-1" || len(published.Changes) != len(snapshot.Compare(start, complete)) {
		t.Fatalf("publication request %+v", published)
	}
	for _, c := range published.Changes {
		name := c.Path().String()
		if !strings.HasPrefix(name, ".markitect/model/") && !strings.HasPrefix(name, ".markitect/registry/") && name != ".markitect/execution.yaml" {
			t.Errorf("an artifact was published: %s", name)
		}
	}
	// Both phases' volumes closed (the first once its candidate is taken, the second after publication); every home removed.
	if !slices.Contains(workspaces.closed, "x-1-structure-work") || !slices.Contains(workspaces.closed, "x-1-model-work") {
		t.Errorf("closed %v", workspaces.closed)
	}
	if len(removed) != len(result.Structure.Homes)+len(result.Model.Homes) || len(removed) == 0 {
		t.Errorf("removed %v", removed)
	}
	// The Brief reached the Agent of each phase with its mandate; a Verify turn never writes.
	var structureExecute, modelVerify run.Turn
	for _, turn := range agents.turns {
		if structureExecute.ID == "" && turn.Manager == "." && turn.Kind == run.ExecuteTurn && strings.HasPrefix(turn.ID, "x-1-structure") {
			structureExecute = turn
		}
		if modelVerify.ID == "" && turn.Kind == run.VerifyTurn && strings.HasPrefix(turn.ID, "x-1-model") && turn.Manager == "lending" {
			modelVerify = turn
		}
	}
	if !strings.Contains(structureExecute.Prompt, "Your task is the structure of the model") || !strings.Contains(structureExecute.Prompt, "structure phase of an Extract") ||
		!slices.ContainsFunc(structureExecute.Tools, func(s run.ToolSpec) bool { return s.Name == "check" }) {
		t.Errorf("structure prompt:\n%s", structureExecute.Prompt)
	}
	if !strings.Contains(modelVerify.Prompt, "report it with subject artifact") || !strings.Contains(modelVerify.Prompt, "could important behavior of this area change") {
		t.Errorf("verify prompt lacks the Extract guidance:\n%s", modelVerify.Prompt)
	}
	phases := 0
	var turnBudgets []any
	for _, e := range evidence.events {
		if e.Kind == "extract.phase" {
			phases++
		}
		if e.Kind == "run.start" {
			turnBudgets = append(turnBudgets, e.Data.(map[string]any)["agentTurns"])
		}
	}
	if phases != 2 {
		t.Errorf("phase events: %d", phases)
	}
	// The structure phase gets half of the 120 agent turns; the model phase the rest it left.
	if len(turnBudgets) != 2 || turnBudgets[0] != 60 || turnBudgets[1] != 120-len(result.Structure.Turns) {
		t.Errorf("agent-turn budgets of the phases: %v", turnBudgets)
	}
}

func TestExtractReportsTheChecksTheStructureProposed(t *testing.T) {
	req, agents, _, publisher, _, _, _ := bothPhases(t)
	agents.writes["x-1-structure ."][".markitect/execution.yaml"] += extractProposedCheck("library-tests")
	result, err := apply.Extract(context.Background(), req)
	if err != nil || result.Model == nil || result.Model.Status != run.Verified {
		t.Fatalf("%v %+v", err, result.Model)
	}
	want := []apply.ProposedCheck{{ID: "project-vet", Profile: "library-tests", Directory: "realization", Args: []string{"vet", "./..."}, Timeout: "2m"}}
	if !reflect.DeepEqual(result.ProposedChecks, want) {
		t.Errorf("proposed checks %+v, want %+v", result.ProposedChecks, want)
	}
	// The Managers of the model phase see the proposal as a declared check, and the
	// execution document is published with it.
	modelBriefs := 0
	for _, turn := range agents.turns {
		if turn.Kind == run.ExecuteTurn && strings.HasPrefix(turn.ID, "x-1-model") && strings.Contains(turn.Prompt, "project-vet: profile library-tests: vet ./...") {
			modelBriefs++
		}
	}
	if modelBriefs == 0 {
		t.Error("no model-phase Brief lists the proposed check")
	}
	if len(publisher.requests) != 1 || !slices.ContainsFunc(publisher.requests[0].Changes, func(c snapshot.Change) bool { return c.Path().String() == ".markitect/execution.yaml" }) {
		t.Errorf("the execution document is not published: %+v", publisher.requests)
	}
}

// Nobody outside the Manager tree answers during an Extract: Root's question
// to the user is refused with the closed-run rule in both phases, and the
// phases go on without it.
func TestExtractRefusesQuestionsBeyondTheTree(t *testing.T) {
	req, agents, _, _, _, _, _ := bothPhases(t)
	agents.asks = map[string]string{"x-1-structure .": "Is the reminders area part of lending?", "x-1-model .": "Is go.mod part of the library?"}
	result, err := apply.Extract(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Model == nil || result.Model.Status != run.Verified {
		t.Fatalf("structure %s %s; model %+v", result.Structure.Status, result.Structure.Reason, result.Model)
	}
	if len(agents.refusals) != 2 {
		t.Fatalf("refusals %v", agents.refusals)
	}
	for key, text := range agents.refusals {
		if !strings.Contains(text, "Nobody outside the Manager tree answers") || strings.Contains(text, "owner decisions") {
			t.Errorf("%s was told %q", key, text)
		}
	}
	for _, turn := range agents.turns {
		if strings.Contains(turn.Prompt, "Is the reminders area part of lending?") {
			t.Errorf("turn %s shows the refused question", turn.ID)
		}
	}
}

func TestExtractStopsAfterAnUnverifiedStructurePhaseAndKeepsItsWork(t *testing.T) {
	req := extractRequest(t)
	workspaces := newMemoryWorkspaces()
	agents := &scriptedAgents{t: t, ws: workspaces, refusals: map[string]string{}, assessed: map[string]bool{}, writes: map[string]map[string]string{}}
	publisher := &fakePublisher{}
	evidence := &memoryEvidence{}
	req.Agents, req.Workspaces, req.Evidence, req.Publisher = agents, workspaces, evidence, publisher
	// Root changes nothing: the candidate has no Package, so the File check
	// returns every basis it submits.
	result, err := apply.Extract(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Structure.Status != run.Incomplete || result.Model != nil || result.Receipt != nil || len(publisher.requests) != 0 || result.RetainedWork != "x-1-structure-work" {
		t.Fatalf("%+v", result)
	}
	if result.Structure.Reason == "" || len(result.Structure.Passes) == 0 || result.Structure.Passes[0].Sealed {
		t.Errorf("the unverified phase has no reason, or sealed a Pass nothing could accept: %+v", result.Structure)
	}
	returned := false
	for _, e := range evidence.events {
		data, _ := e.Data.(map[string]any)
		if lines, ok := data["findings"].([]string); ok && e.Kind == "proposal.checked" && data["passed"] == false {
			returned = returned || slices.ContainsFunc(lines, func(line string) bool { return strings.Contains(line, "model.no_package") })
		}
	}
	if !returned {
		t.Errorf("no submission was returned for model.no_package: %d events", len(evidence.events))
	}
}

func TestExtractKeepsTheStructureWhenTheModelPhaseCannotStart(t *testing.T) {
	req, _, workspaces, publisher, _, _, _ := bothPhases(t)
	workspaces.refuse = "x-1-model-work"
	result, err := apply.Extract(context.Background(), req)
	if err == nil || result.Structure.Status != run.Verified || result.Model == nil || result.Model.Status != run.Failed {
		t.Fatalf("err %v, structure %s, model %+v", err, result.Structure.Status, result.Model)
	}
	if result.RetainedWork != "x-1-structure-work" || slices.Contains(workspaces.closed, "x-1-structure-work") || len(publisher.requests) != 0 {
		t.Fatalf("retained %q closed %v", result.RetainedWork, workspaces.closed)
	}
}

func TestExtractKeepsTheModelWorkWhenTheModelPhaseDoesNotVerify(t *testing.T) {
	req, agents, workspaces, publisher, _, _, _ := bothPhases(t)
	// Root registers nothing: its artifacts stay unexplained in every round.
	agents.writes["x-1-model ."] = nil
	result, err := apply.Extract(context.Background(), req)
	if err != nil || result.Structure.Status != run.Verified || result.Model == nil || result.Model.Status != run.Incomplete {
		t.Fatalf("err %v, structure %s, model %+v", err, result.Structure.Status, result.Model)
	}
	// The model phase's own volume is the result; the structure one is closed.
	if result.RetainedWork != "x-1-model-work" || slices.Contains(workspaces.closed, "x-1-model-work") || !slices.Contains(workspaces.closed, "x-1-structure-work") ||
		result.Receipt != nil || len(publisher.requests) != 0 {
		t.Fatalf("retained %q closed %v receipt %+v", result.RetainedWork, workspaces.closed, result.Receipt)
	}
}

func TestExtractEndsTheModelPhaseWhenAnAgentLeavesAFileOutsideAnyArea(t *testing.T) {
	req, agents, workspaces, publisher, _, _, _ := bothPhases(t)
	// The Runtime's state is no area of any Manager: a file there never enters
	// an accepted stand, so the stand at the end of the Pass differs from it and
	// the phase cannot publish.
	agents.writes["x-1-model ."][".markitect/state/runs/x.json"] = "{}" + "\n"
	result, err := apply.Extract(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Structure.Status != run.Verified || result.Model == nil || result.Model.Status == run.Verified || !strings.Contains(result.Model.Reason, ".markitect/state/runs/x.json") {
		t.Fatalf("structure %s; model %+v", result.Structure.Status, result.Model)
	}
	if result.RetainedWork != "x-1-model-work" || slices.Contains(workspaces.closed, "x-1-model-work") || result.Receipt != nil || len(publisher.requests) != 0 {
		t.Fatalf("retained %q closed %v receipt %+v", result.RetainedWork, workspaces.closed, result.Receipt)
	}
}

func TestExtractKeepsTheStructureWhenTheModelPhaseHasTooFewTurns(t *testing.T) {
	req, agents, workspaces, publisher, _, _, _ := bothPhases(t)
	// 13 turns give the structure phase 6 and the model phase the 9 it has left
	// after the structure phase's four; its six Managers need 12 for one Pass.
	req.Prepared = extractAttemptWithTurns(t, 13)
	agents.writes["x-1-structure ."][".markitect/execution.yaml"] = strings.Replace(extractExecutionProposed, "agentTurns: 120", "agentTurns: 13", 1)
	result, err := apply.Extract(context.Background(), req)
	if err != nil || result.Structure.Status != run.Verified || result.Model == nil || result.Model.Status != run.Incomplete ||
		!strings.Contains(result.Model.Reason, "agent turn budget of 9 does not cover a basis and a result of each of the 6 managers") {
		t.Fatalf("err %v, structure %s, model %+v", err, result.Structure.Status, result.Model)
	}
	if result.RetainedWork != "x-1-structure-work" || slices.Contains(workspaces.closed, "x-1-structure-work") || result.Receipt != nil || len(publisher.requests) != 0 {
		t.Fatalf("retained %q closed %v", result.RetainedWork, workspaces.closed)
	}
}

func TestExtractDeadlineIsTheEarlierOfTheOperatorAndTheExecutionTime(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name     string
		operator time.Time
		want     time.Time
	}{
		{"operator first", now.Add(5 * time.Minute), now.Add(5 * time.Minute)},
		{"execution time first", now.Add(2 * time.Hour), now.Add(25 * time.Minute)},
		{"no operator deadline", time.Time{}, now.Add(25 * time.Minute)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _, _, _, _, _, _ := bothPhases(t)
			evidence := &memoryEvidence{}
			req.Evidence, req.Clock, req.Deadline = evidence, func() time.Time { return now }, tc.operator
			if _, err := apply.Extract(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			var deadlines []time.Time
			for _, e := range evidence.events {
				if e.Kind == "run.start" {
					deadlines = append(deadlines, e.Data.(map[string]any)["deadline"].(time.Time))
				}
			}
			// Both phases share one deadline.
			if len(deadlines) != 2 || !deadlines[0].Equal(tc.want) || !deadlines[1].Equal(tc.want) {
				t.Errorf("deadlines %v, want %v", deadlines, tc.want)
			}
		})
	}
}
