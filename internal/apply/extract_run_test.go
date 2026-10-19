package apply_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

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
}

func newMemoryWorkspaces() *memoryWorkspaces {
	return &memoryWorkspaces{files: map[string]map[string]string{}, ids: map[run.WorkspaceID]string{}}
}

func (w *memoryWorkspaces) Materialize(_ context.Context, spec run.WorkspaceSpec) (run.WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
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
	return run.WorkspaceObservation{ID: id, Closed: true}, nil
}

func (w *memoryWorkspaces) write(volume string, files map[string]string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for name, content := range files {
		w.files[volume][name] = content
	}
}

// scriptedAgents writes each Manager's files in its first Execute turn, then
// finishes; every Verify turn is satisfied, and a Manager can raise findings.
type scriptedAgents struct {
	ws       *memoryWorkspaces
	writes   map[string]map[string]string // turn-ID phase/manager key -> files
	findings map[string][]map[string]any
	mu       sync.Mutex
	turns    []run.Turn
	enforced map[string]bool
}

func (a *scriptedAgents) Run(ctx context.Context, turn run.Turn, tools run.ToolHandler) (run.TurnResult, error) {
	a.mu.Lock()
	a.turns = append(a.turns, turn)
	a.mu.Unlock()
	key := strings.SplitN(turn.ID, "-r", 2)[0] + " " + turn.Manager
	call := func(name string, args any) {
		data, _ := json.Marshal(args)
		tools(ctx, name, data)
	}
	if turn.Kind == run.ExecuteTurn {
		a.ws.write(turn.Workspace, a.writes[key])
		call("finish", map[string]any{"status": "done", "summary": "Described " + turn.Manager + "."})
	} else {
		call("assess", map[string]any{"summary": "The model matches the artifacts.", "satisfied": true, "findings": a.findings[key]})
	}
	return run.TurnResult{Conversation: "c-" + turn.Manager, Usage: run.Usage{InputTokens: 10, OutputTokens: 5}, Stopped: true}, nil
}

func TestExtractRunsBothPhasesAndPublishesTheModel(t *testing.T) {
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
	agents := &scriptedAgents{ws: workspaces, writes: map[string]map[string]string{
		"x-1-structure .": diff(start, proposed, func(string) bool { return true }),
	}, findings: map[string][]map[string]any{
		"x-1-model lending": {{"owner": "lending", "obligation": "Realization deviation: lending/lending", "observed": "borrow.go admits a sixth loan.", "blocking": false}},
	}}
	for _, owner := range []string{".", "catalog", "library", "lending", "lending/reminders", "members"} {
		shard := ".markitect/registry/" + owner + "/registry.yaml"
		if owner == "." {
			shard = ".markitect/registry/registry.yaml"
		}
		agents.writes["x-1-model "+owner] = diff(proposed, complete, func(name string) bool { return name == shard })
	}
	publisher := &fakePublisher{receipt: apply.PublicationReceipt{ID: "x-1", Outcome: apply.PublicationApplied}}
	evidence := &memoryEvidence{}
	var removed []string
	req.Agents, req.Workspaces, req.Evidence, req.Publisher = agents, workspaces, evidence, publisher
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
	if len(result.Structure.Turns) != 2 || len(result.Model.Turns) < 12 || len(result.Structure.Rounds) != 1 || len(result.Model.Rounds) != 1 || result.Model.Usage.InputTokens != int64(10*len(result.Model.Turns)) {
		t.Fatalf("turns %d/%d rounds %d/%d usage %+v", len(result.Structure.Turns), len(result.Model.Turns), len(result.Structure.Rounds), len(result.Model.Rounds), result.Model.Usage)
	}
	deviations := apply.ExtractDeviations(result.Model.Findings)
	if len(deviations) != 1 || deviations[0].Owner != "lending" || !strings.Contains(deviations[0].Observed, "sixth loan") {
		t.Fatalf("deviations %+v", deviations)
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
		if turn.Manager == "." && turn.Kind == run.ExecuteTurn && strings.HasPrefix(turn.ID, "x-1-structure") {
			structureExecute = turn
		}
		if turn.Kind == run.VerifyTurn && strings.HasPrefix(turn.ID, "x-1-model") && turn.Manager == "lending" {
			modelVerify = turn
		}
	}
	if !strings.Contains(structureExecute.Prompt, "This first phase proposes only the structure of the model") || !strings.Contains(structureExecute.Prompt, "structure phase of an Extract") ||
		!slices.ContainsFunc(structureExecute.Tools, func(s run.ToolSpec) bool { return s.Name == "check" }) {
		t.Errorf("structure prompt:\n%s", structureExecute.Prompt)
	}
	if !strings.Contains(modelVerify.Prompt, "Realization deviation") || !strings.Contains(modelVerify.Prompt, "could important behavior of this area change") {
		t.Errorf("verify prompt lacks the Extract guidance:\n%s", modelVerify.Prompt)
	}
	phases := 0
	for _, e := range evidence.events {
		if e.Kind == "extract.phase" {
			phases++
		}
	}
	if phases != 2 {
		t.Errorf("phase events: %d", phases)
	}
}

func TestExtractStopsAfterAnUnverifiedStructurePhaseAndKeepsItsWork(t *testing.T) {
	req := extractRequest(t)
	workspaces := newMemoryWorkspaces()
	agents := &scriptedAgents{ws: workspaces, writes: map[string]map[string]string{}}
	publisher := &fakePublisher{}
	req.Agents, req.Workspaces, req.Evidence, req.Publisher = agents, workspaces, &memoryEvidence{}, publisher
	// Root changes nothing: the candidate has no Package, accounting blocks every round.
	result, err := apply.Extract(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Structure.Status != run.Incomplete || result.Model != nil || result.Receipt != nil || len(publisher.requests) != 0 || result.RetainedWork != "x-1-structure-work" {
		t.Fatalf("%+v", result)
	}
	blocked := false
	for _, f := range result.Structure.Findings {
		blocked = blocked || f.Obligation == "model.no_package"
	}
	if !blocked {
		t.Errorf("findings %+v", result.Structure.Findings)
	}
}
