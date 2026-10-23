package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/authoring"
	"github.com/Glacius-Labs/Markitect/internal/model"
	runport "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// sessionWorkspace is the session's workspace: the start tree with the
// changes an Agent made, observed by Peek.
type sessionWorkspace struct {
	mu              sync.Mutex
	tree            snapshot.Tree
	spec            runport.WorkspaceSpec
	closed          bool
	removed         []string
	failMaterialize error
	materializing   chan struct{} // closed when Materialize starts, if set
	release         chan struct{} // Materialize waits for it, if set
}

func (w *sessionWorkspace) Materialize(_ context.Context, spec runport.WorkspaceSpec) (runport.WorkspaceObservation, error) {
	if w.materializing != nil {
		close(w.materializing)
		<-w.release
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.failMaterialize != nil {
		return runport.WorkspaceObservation{}, w.failMaterialize
	}
	w.spec, w.tree = spec, spec.Tree
	return runport.WorkspaceObservation{ID: spec.ID, Volume: string(spec.ID)}, nil
}

func (w *sessionWorkspace) Harvest(ctx context.Context, r runport.HarvestRequest) (runport.WorkspaceHarvest, error) {
	return w.Peek(ctx, r)
}

func (w *sessionWorkspace) Peek(_ context.Context, r runport.HarvestRequest) (runport.WorkspaceHarvest, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return runport.WorkspaceHarvest{ID: r.ID, Tree: w.tree}, nil
}

func (w *sessionWorkspace) Close(_ context.Context, id runport.WorkspaceID) (runport.WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return runport.WorkspaceObservation{ID: id, Closed: true}, nil
}

// read returns one file of the workspace.
func (w *sessionWorkspace) read(t *testing.T, name string) string {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	path, err := snapshot.NewPath(name)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := w.tree.Lookup(path)
	if !ok {
		t.Fatalf("the workspace has no %s", name)
	}
	return string(entry.Bytes())
}

// write changes one file of the workspace, as an Agent's edit would.
func (w *sessionWorkspace) write(t *testing.T, name, content string) {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	tree, err := replaceEntry(w.tree, name, content)
	if err != nil {
		t.Fatal(err)
	}
	w.tree = tree
}

func replaceEntry(tree snapshot.Tree, name, content string) (snapshot.Tree, error) {
	path, err := snapshot.NewPath(name)
	if err != nil {
		return snapshot.Tree{}, err
	}
	var entries []snapshot.Entry
	for _, e := range tree.Entries() {
		if e.Path() != path {
			entries = append(entries, e)
		}
	}
	entry, err := snapshot.NewEntry(path, snapshot.Regular, []byte(content))
	if err != nil {
		return snapshot.Tree{}, err
	}
	return snapshot.NewTree(append(entries, entry))
}

// scriptedAgents runs each turn with a script that may edit the workspace and
// call the session's tools.
type scriptedAgents struct {
	mu     sync.Mutex
	turns  []runport.Turn
	script func(ctx context.Context, turn runport.Turn, tools runport.ToolHandler) (runport.TurnResult, error)
}

func (a *scriptedAgents) Run(ctx context.Context, turn runport.Turn, tools runport.ToolHandler) (runport.TurnResult, error) {
	a.mu.Lock()
	a.turns = append(a.turns, turn)
	a.mu.Unlock()
	return a.script(ctx, turn, tools)
}

func sessionFixture(t *testing.T, agents *scriptedAgents) (*authoring.Service, *sessionWorkspace, string, string) {
	t.Helper()
	repo, _, service := authoringFixture(t)
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	selection, err := repositorySelection(repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	captured, err := apply.NewPreparationService(apply.NewModelService(repository, repository)).Capture(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || captured.Status() != model.Passed {
		t.Fatalf("capture = %s, %v", captured.Status(), err)
	}
	attempts := captured.Attempts()
	workspace := &sessionWorkspace{}
	state := t.TempDir()
	var removed []string
	service.WithSessions(func(context.Context) (authoring.SessionStart, error) {
		return authoring.SessionStart{
			Basis: apply.SessionBasis{Attempt: attempts[len(attempts)-1], Profile: "project-agent", TurnTimeout: time.Minute, Turns: 2, Guide: "\nLANGUAGE GUIDE\n", ModelDir: ".markitect/model"},
			Ports: authoring.SessionPorts{Agents: agents, Workspaces: workspace, MaxEntries: 1 << 16, MaxBytes: 1 << 30, Evidence: state,
				RemoveVolume: func(_ context.Context, name string) error {
					removed = append(removed, name)
					workspace.removed = removed
					return nil
				},
				RequireVolumeSubpaths: func(context.Context) error { return nil }},
		}, nil
	})
	t.Cleanup(func() { _ = service.CloseSessions(context.Background()) })
	return service, workspace, state, repo.Dir
}

func waitForSession(t *testing.T, service *authoring.Service, id string) authoring.Session {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if s, ok := service.Session(id); ok && s.State != authoring.Working {
			return s
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("session %s still works", id)
	return authoring.Session{}
}

const sessionModelFile = ".markitect/model/product/cockpit/accepted-proposals.yaml"

func TestCockpitModelSessionProposesOnlyModelChanges(t *testing.T) {
	ctx := context.Background()
	var workspace *sessionWorkspace
	agents := &scriptedAgents{}
	agents.script = func(ctx context.Context, turn runport.Turn, tools runport.ToolHandler) (runport.TurnResult, error) {
		workspace.write(t, sessionModelFile, strings.Replace(workspace.read(t, sessionModelFile), "spec: {}", "spec: {}\n# reviewed", 1))
		check := tools(ctx, "check", json.RawMessage(`{}`))
		submit := tools(ctx, "submit", json.RawMessage(`{"summary":"Marked the Rule as reviewed."}`))
		answer := tools(ctx, "answer", json.RawMessage(`{"text":"One point is unclear."}`))
		return runport.TurnResult{Conversation: "conversation-1", Final: check.Text + "|" + submit.Text + "|" + answer.Text, Usage: runport.Usage{ProviderTurns: 1, InputTokens: 10}, Started: time.Now(), Stopped: true}, nil
	}
	service, ws, state, _ := sessionFixture(t, agents)
	workspace = ws

	s, err := service.StartSession(ctx, "Mark the accepted-proposals Rule as reviewed.")
	if err != nil {
		t.Fatal(err)
	}
	s = waitForSession(t, service, s.ID)
	turn := agents.turns[0]
	if turn.Kind != runport.ExecuteTurn || !turn.ReadOnly || len(turn.Mounts) != 2 || !turn.Mounts[0].Writable || turn.Mounts[0].Path != ".markitect/model" || turn.Mounts[1].Path != ".markitect/registry" ||
		turn.Profile != "project-agent" || turn.Timeout != time.Minute || turn.Resume != "" || turn.Home == "" || turn.Workspace != s.ID+"-work" ||
		turn.Transcript != filepath.Join(state, "sessions", s.ID, "transcripts", turn.ID+".jsonl") || !strings.Contains(turn.Prompt, "Mark the accepted-proposals Rule as reviewed.") || !strings.Contains(turn.Prompt, "LANGUAGE GUIDE") {
		t.Fatalf("turn = %+v", turn)
	}
	if len(ws.spec.Directories) != 2 {
		t.Fatalf("workspace spec = %+v", ws.spec.Directories)
	}
	got := s.Turns[0]
	if s.State != authoring.Waiting || len(got.Tools) != 3 || len(got.Answers) != 1 || got.Answers[0] != "One point is unclear." || !got.Stopped || got.Tools[0].Refused || !strings.Contains(got.Tools[0].Answer, "Structural check: passed") ||
		!strings.Contains(got.Tools[0].Answer, "changed: "+sessionModelFile) || len(got.Proposals) != 1 || got.Usage.InputTokens != 10 || s.Usage.ProviderTurns != 1 {
		t.Fatalf("session = %+v", s)
	}
	p, ok := service.Proposal(got.Proposals[0])
	if !ok || p.Kind != "model" || p.Session != s.ID || p.Summary != "Marked the Rule as reviewed." || p.Check.Status != "passed" || len(p.Changes) != 1 || p.Changes[0].Path != sessionModelFile {
		t.Fatalf("proposal = %+v", p)
	}

	// The next prompt continues the conversation; the limit ends the session's prompts.
	agents.script = func(ctx context.Context, turn runport.Turn, tools runport.ToolHandler) (runport.TurnResult, error) {
		return runport.TurnResult{Conversation: "conversation-1", Final: "nothing to change", Started: time.Now(), Stopped: true}, nil
	}
	if _, err := service.PromptSession(s.ID, "Anything else?"); err != nil {
		t.Fatal(err)
	}
	s = waitForSession(t, service, s.ID)
	if agents.turns[1].Resume != "conversation-1" || len(s.Turns) != 2 || s.Turns[1].Final != "nothing to change" {
		t.Fatalf("second turn = %+v, session %+v", agents.turns[1], s)
	}
	if _, err := service.PromptSession(s.ID, "One more?"); !errors.Is(err, authoring.ErrLimit) {
		t.Fatalf("a prompt beyond the limit = %v", err)
	}
	if _, err := service.StartSession(ctx, "Another session"); !errors.Is(err, authoring.ErrSessionOpen) {
		t.Fatalf("a second open session = %v", err)
	}

	closed, err := service.CloseSession(ctx, s.ID)
	if err != nil || closed.State != authoring.Closed || !ws.closed || len(ws.removed) != 1 || ws.removed[0] != s.ID+"-home" {
		t.Fatalf("close = %+v, %v (workspace closed %v, removed %v)", closed, err, ws.closed, ws.removed)
	}
	events, err := os.ReadFile(filepath.Join(state, "sessions", s.ID, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"session.start", "prompt", "turn.start", "tool", "proposal", "turn.end", "session.close"} {
		if !strings.Contains(string(events), `"kind":"`+kind+`"`) {
			t.Errorf("the session record lacks %s:\n%s", kind, events)
		}
	}
}

func TestCockpitModelSessionRefusesChangesOutsideTheModel(t *testing.T) {
	ctx := context.Background()
	var workspace *sessionWorkspace
	agents := &scriptedAgents{}
	agents.script = func(ctx context.Context, turn runport.Turn, tools runport.ToolHandler) (runport.TurnResult, error) {
		workspace.write(t, "README.md", "changed outside the model\n")
		answer := tools(ctx, "submit", json.RawMessage(`{"summary":"Changed the README."}`))
		bad := tools(ctx, "submit", json.RawMessage(`{"summary":""}`))
		unknown := tools(ctx, "write_file", json.RawMessage(`{}`))
		if !answer.Refused || !bad.Refused || !unknown.Refused {
			return runport.TurnResult{}, errors.New("an out-of-scope, empty or unknown tool call was accepted")
		}
		return runport.TurnResult{Final: answer.Text, Started: time.Now(), Stopped: true}, nil
	}
	service, ws, _, _ := sessionFixture(t, agents)
	workspace = ws
	s, err := service.StartSession(ctx, "Change the README.")
	if err != nil {
		t.Fatal(err)
	}
	s = waitForSession(t, service, s.ID)
	if s.Turns[0].Reason != "" || len(s.Turns[0].Proposals) != 0 || len(service.Proposals()) != 0 || !strings.Contains(s.Turns[0].Final, "may not change README.md") {
		t.Fatalf("session = %+v", s)
	}
}

func TestCockpitModelSessionStopsAWorkingTurnWhenClosed(t *testing.T) {
	ctx := context.Background()
	started := make(chan struct{})
	agents := &scriptedAgents{script: func(ctx context.Context, turn runport.Turn, tools runport.ToolHandler) (runport.TurnResult, error) {
		close(started)
		<-ctx.Done()
		return runport.TurnResult{Stopped: true, Reason: "cancelled"}, ctx.Err()
	}}
	service, ws, _, _ := sessionFixture(t, agents)
	s, err := service.StartSession(ctx, "Work slowly.")
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if _, err := service.PromptSession(s.ID, "Faster?"); !errors.Is(err, authoring.ErrBusy) {
		t.Fatalf("a prompt while working = %v", err)
	}
	closed, err := service.CloseSession(ctx, s.ID)
	if err != nil || closed.State != authoring.Closed || !closed.Turns[0].Stopped || !ws.closed {
		t.Fatalf("close while working = %+v, %v", closed, err)
	}
	if _, err := service.PromptSession(s.ID, "Hello?"); err == nil {
		t.Fatal("a closed session took a prompt")
	}
}

// A turn whose container end was not observed halts the session: no further
// prompt may start a second container on the same workspace.
func TestCockpitModelSessionHaltsWhenTheContainerEndIsNotObserved(t *testing.T) {
	ctx := context.Background()
	agents := &scriptedAgents{script: func(context.Context, runport.Turn, runport.ToolHandler) (runport.TurnResult, error) {
		return runport.TurnResult{Started: time.Now(), Stopped: false, Reason: "the container did not stop"}, errors.New("stop not observed")
	}}
	service, ws, _, _ := sessionFixture(t, agents)
	s, err := service.StartSession(ctx, "Change something.")
	if err != nil {
		t.Fatal(err)
	}
	s = waitForSession(t, service, s.ID)
	if s.State != authoring.Halted || s.Error == "" || s.Turns[0].Stopped {
		t.Fatalf("session = %+v", s)
	}
	if _, err := service.PromptSession(s.ID, "Again?"); !errors.Is(err, authoring.ErrHalted) {
		t.Fatalf("a prompt to a halted session = %v", err)
	}
	if closed, err := service.CloseSession(ctx, s.ID); err != nil || closed.State != authoring.Closed || !ws.closed {
		t.Fatalf("close = %+v, %v", closed, err)
	}
}

// A failed materialization closes what it may have created, and the session
// ends with its reason in the record.
func TestCockpitModelSessionReleasesAFailedWorkspace(t *testing.T) {
	agents := &scriptedAgents{script: func(context.Context, runport.Turn, runport.ToolHandler) (runport.TurnResult, error) {
		t.Error("no turn may run without a workspace")
		return runport.TurnResult{}, nil
	}}
	service, ws, state, _ := sessionFixture(t, agents)
	ws.failMaterialize = errors.New("disk full")
	s, err := service.StartSession(context.Background(), "Change something.")
	if err == nil || s.State != authoring.Closed || !ws.closed {
		t.Fatalf("start on a failing workspace = %+v, %v (closed %v)", s, err, ws.closed)
	}
	events, readErr := os.ReadFile(filepath.Join(state, "sessions", s.ID, "events.jsonl"))
	if readErr != nil || !strings.Contains(string(events), `"kind":"session.fail"`) {
		t.Fatalf("the record of a failed start = %s, %v", events, readErr)
	}
	ws.failMaterialize = nil
	agents.script = func(context.Context, runport.Turn, runport.ToolHandler) (runport.TurnResult, error) {
		return runport.TurnResult{Started: time.Now(), Stopped: true}, nil
	}
	if _, err := service.StartSession(context.Background(), "Again."); err != nil {
		t.Fatalf("a failed session still blocks a new one: %v", err)
	}
}

// Closing while the session prepares ends it when the preparation ends, and
// releases the workspace the preparation made.
func TestCockpitModelSessionClosedWhilePreparing(t *testing.T) {
	agents := &scriptedAgents{script: func(context.Context, runport.Turn, runport.ToolHandler) (runport.TurnResult, error) {
		t.Error("no turn may run in a session closed while it prepared")
		return runport.TurnResult{}, nil
	}}
	service, ws, _, _ := sessionFixture(t, agents)
	ws.materializing = make(chan struct{})
	ws.release = make(chan struct{})
	started := make(chan struct{})
	var startErr error
	go func() {
		defer close(started)
		_, startErr = service.StartSession(context.Background(), "Change something.")
	}()
	<-ws.materializing
	id := service.Sessions()[0].ID
	closing := make(chan authoring.Session)
	go func() {
		s, _ := service.CloseSession(context.Background(), id)
		closing <- s
	}()
	for s, _ := service.Session(id); s.State != authoring.Closing; s, _ = service.Session(id) {
		time.Sleep(5 * time.Millisecond)
	}
	close(ws.release)
	closed := <-closing
	<-started
	if startErr == nil || closed.State != authoring.Closed || closed.Error == "" || !ws.closed {
		t.Fatalf("closed while preparing = %+v, start %v, workspace closed %v", closed, startErr, ws.closed)
	}
}

// While a close waits for the working turn to end, the session takes no
// prompt, and the ending turn does not reopen it.
func TestCockpitModelSessionTakesNoPromptWhileClosing(t *testing.T) {
	ctx := context.Background()
	started := make(chan struct{})
	ending := make(chan struct{})
	agents := &scriptedAgents{script: func(ctx context.Context, _ runport.Turn, _ runport.ToolHandler) (runport.TurnResult, error) {
		close(started)
		<-ctx.Done()
		<-ending
		return runport.TurnResult{Started: time.Now(), Stopped: true}, nil
	}}
	service, _, _, _ := sessionFixture(t, agents)
	s, err := service.StartSession(ctx, "Work.")
	if err != nil {
		t.Fatal(err)
	}
	<-started
	closing := make(chan authoring.Session)
	go func() {
		closed, _ := service.CloseSession(ctx, s.ID)
		closing <- closed
	}()
	for v, _ := service.Session(s.ID); v.State != authoring.Closing; v, _ = service.Session(s.ID) {
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := service.PromptSession(s.ID, "Sneak in."); !errors.Is(err, authoring.ErrSessionClosed) {
		t.Fatalf("a prompt while closing = %v", err)
	}
	close(ending)
	if closed := <-closing; closed.State != authoring.Closed || len(closed.Turns) != 1 {
		t.Fatalf("closed = %+v", closed)
	}
	if v, _ := service.Session(s.ID); v.State != authoring.Closed {
		t.Fatalf("the ended turn reopened the session: %s", v.State)
	}
}
