package apply_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// These ports test control flow against the actual prepared Library model.
// They do not establish that a provider recognizes a semantic defect.
type realizationCopies struct {
	mu       sync.Mutex
	t        *testing.T
	copies   map[run.WorkspaceID]*realizationCopy
	trees    map[string]snapshot.Tree
	sequence int
}
type realizationCopy struct {
	owner *realizationCopies
	id    run.WorkspaceID
	start string
	live  snapshot.Tree
}

func (w *realizationCopy) ID() run.WorkspaceID    { return w.id }
func (w *realizationCopy) Volume() string         { return string(w.id) }
func (w *realizationCopy) StartingCommit() string { return w.start }
func (w *realizationCopy) Commit(context.Context, string) (string, error) {
	w.owner.mu.Lock()
	defer w.owner.mu.Unlock()
	return w.commitLocked()
}
func (w *realizationCopy) commitLocked() (string, error) {
	w.owner.sequence++
	id := fmt.Sprintf("%040x", w.owner.sequence)
	w.owner.trees[id] = w.live
	return id, nil
}
func (w *realizationCopy) ContainsCommit(_ context.Context, id string) (bool, error) {
	w.owner.mu.Lock()
	defer w.owner.mu.Unlock()
	_, ok := w.owner.trees[id]
	return ok, nil
}
func (w *realizationCopy) Close(context.Context) error { return nil }
func (w *realizationCopies) PrepareAssignment(ctx context.Context, spec run.WorkspaceSpec) (run.AssignmentWorkspace, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.copies == nil {
		w.copies = make(map[run.WorkspaceID]*realizationCopy)
		w.trees = make(map[string]snapshot.Tree)
	}
	if w.copies[spec.ID] != nil {
		return nil, errors.New("copy already exists")
	}
	c := &realizationCopy{owner: w, id: spec.ID, live: spec.Tree}
	w.copies[spec.ID] = c
	c.start, _ = c.commitLocked()
	return c, nil
}
func (w *realizationCopies) ForkAssignment(ctx context.Context, source run.AssignmentWorkspace, id run.WorkspaceID) (run.AssignmentWorkspace, error) {
	commit, err := source.Commit(ctx, "foundation")
	if err != nil {
		return nil, err
	}
	return w.ForkAssignmentCommit(ctx, source, commit, id)
}
func (w *realizationCopies) ForkAssignmentCommit(_ context.Context, _ run.AssignmentWorkspace, commit string, id run.WorkspaceID) (run.AssignmentWorkspace, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.copies[id] != nil {
		return nil, errors.New("copy already exists")
	}
	tree, ok := w.trees[commit]
	if !ok {
		return nil, errors.New("commit unavailable")
	}
	c := &realizationCopy{owner: w, id: id, start: commit, live: tree}
	w.copies[id] = c
	return c, nil
}
func (w *realizationCopies) ReadAssignmentCommit(_ context.Context, r run.AssignmentCommitRequest) (snapshot.Tree, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	tree, ok := w.trees[r.Commit]
	if !ok {
		return tree, errors.New("commit unavailable")
	}
	return tree, nil
}
func (w *realizationCopies) UpdateAssignment(_ context.Context, target, _ run.AssignmentWorkspace, commit, previous string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	updates := map[string]*string{}
	for _, delta := range snapshot.Compare(w.trees[previous], w.trees[commit]) {
		entry, ok := delta.After()
		if ok {
			value := string(entry.Bytes())
			updates[delta.Path().String()] = &value
		} else {
			updates[delta.Path().String()] = nil
		}
	}
	c := target.(*realizationCopy)
	c.live = change(w.t, c.live, updates)
	return nil
}
func (w *realizationCopies) Changes(_ context.Context, _ run.AssignmentWorkspace, h run.AssignmentHandoff) (string, error) {
	return "actual controlled difference for result " + h.ResultCommit, nil
}
func (w *realizationCopies) Integrate(ctx context.Context, parent, child run.AssignmentWorkspace, h run.AssignmentHandoff, previous string) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	base := h.StartingCommit
	if previous != "" {
		base = previous
	}
	updates := map[string]*string{}
	for _, c := range snapshot.Compare(w.trees[base], w.trees[h.ResultCommit]) {
		entry, ok := c.After()
		if ok {
			value := string(entry.Bytes())
			updates[c.Path().String()] = &value
		} else {
			updates[c.Path().String()] = nil
		}
	}
	p := parent.(*realizationCopy)
	p.live = change(w.t, p.live, updates)
	return p.commitLocked()
}

type realizationAgents struct {
	mu                                                          sync.Mutex
	t                                                           *testing.T
	copies                                                      *realizationCopies
	opened                                                      map[string]*realizationAgent
	gap, returnOnce, closeFailure, implementationGap, verifyGap bool
	verifications                                               int
	oversizedOutput                                             bool
	assessmentPromptBytes                                       int
}
type realizationAgent struct {
	owner    *realizationAgents
	spec     run.AssignmentSpec
	handler  run.ToolHandler
	closed   bool
	mode     string
	messages int
}

func (a *realizationAgents) OpenAssignment(_ context.Context, spec run.AssignmentSpec, h run.ToolHandler) (run.AssignmentSession, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.opened == nil {
		a.opened = make(map[string]*realizationAgent)
	}
	s := &realizationAgent{owner: a, spec: spec, handler: h, mode: "implement"}
	for _, tool := range spec.Tools {
		if tool.Name == run.AcceptModelTool {
			s.mode = "assess"
		}
		if tool.Name == run.AcceptResultTool {
			s.mode = "verify"
		}
	}
	a.opened[spec.ID] = s
	return s, nil
}
func (s *realizationAgent) Message(ctx context.Context, prompt string, _ time.Duration) (run.TurnResult, error) {
	s.messages++
	call := func(name string, value any) error {
		body, _ := json.Marshal(value)
		answer := s.handler(ctx, name, body)
		if answer.Refused {
			return errors.New(answer.Text)
		}
		return nil
	}
	s.owner.copies.mu.Lock()
	copy := s.owner.copies.copies[run.WorkspaceID(s.spec.Workspace)]
	s.owner.copies.mu.Unlock()
	if s.mode == "assess" {
		s.owner.assessmentPromptBytes = len(prompt)
		if s.owner.gap {
			return run.TurnResult{Completed: true, Final: "A material public decision is unresolved."}, call(run.ModelProblemTool, run.ReviewIssue{Owner: "lending", Obligation: "lending/borrow", Expected: "The model settles or delegates the public boundary.", Observed: "Concrete incompatible outcomes are left materially open."})
		}
		return run.TurnResult{Completed: true, Final: "No model blocker identified."}, call(run.AcceptModelTool, map[string]string{"commit": copy.start, "reason": "Inspected the fixed model and delegated freedom."})
	}
	if s.mode == "verify" {
		if s.owner.oversizedOutput && (len(prompt) > s.owner.assessmentPromptBytes+(40<<10) || !strings.Contains(prompt, "[truncated; full observation retained by Host]")) {
			s.owner.t.Fatal("oversized native output was not bounded explicitly in the Verify prompt")
		}
		s.owner.verifications++
		if s.owner.verifyGap {
			return run.TurnResult{Completed: true, Final: "A material intent contradiction prevents acceptance."}, call(run.ModelProblemTool, run.ReviewIssue{Owner: ".", Obligation: "library/callable-interface", Expected: "Consistent public behavior.", Observed: "Concrete incompatible fixed-model statements."})
		}
		if s.owner.returnOnce && s.owner.verifications == 1 {
			return run.TurnResult{Completed: true, Final: "Green suite claims do not establish the borrowing boundary."}, call(run.RequestCorrectionTool, run.ReviewIssue{Owner: "lending", Obligation: "lending/borrow", Expected: "The modeled borrowing boundary holds.", Observed: "Controlled model-derived defect requiring owner correction."})
		}
		return run.TurnResult{Completed: true, Final: "Independently accepted this integrated stand."}, call(run.AcceptResultTool, map[string]string{"commit": copy.start, "reason": "Inspected assertions and the relevant integration."})
	}
	if strings.Contains(prompt, "Your responsibility") {
		if s.owner.implementationGap && strings.HasSuffix(s.spec.ID, "-work-0") {
			return run.TurnResult{Completed: true, Final: "Confirmed a material model problem."}, call(run.ModelProblemTool, run.ReviewIssue{Owner: ".", Obligation: "library/callable-interface", Expected: "Consistent public behavior.", Observed: "Concrete incompatible fixed-model statements."})
		}
		_, tail, _ := strings.Cut(prompt, "Required direct children: ")
		children, _, _ := strings.Cut(tail, ". Give each child")
		for _, child := range strings.Split(children, ", ") {
			if child != "" {
				if err := call(run.DelegateChildTool, map[string]string{"child": child, "task": "Realize your own fixed-model responsibility"}); err != nil {
					return run.TurnResult{}, err
				}
			}
		}
	} else if strings.HasPrefix(prompt, "Review your direct child") {
		fields := strings.Fields(prompt)
		child := strings.TrimSuffix(fields[4], "'s")
		commit := fields[7]
		if err := call(run.AcceptChildTool, map[string]string{"child": child, "commit": commit}); err != nil {
			return run.TurnResult{}, err
		}
	} else if strings.HasPrefix(prompt, "Independent Verify returned") && strings.Contains(prompt, "Owner lending;") {
		if err := call(run.ReturnChildTool, map[string]string{"child": "lending", "correction": "Repair the controlled borrowing boundary defect"}); err != nil {
			return run.TurnResult{}, err
		}
	} else if strings.HasPrefix(prompt, "Repair the controlled borrowing boundary defect") {
		copy.live = change(s.owner.t, copy.live, map[string]*string{"realization/lending/borrow.go": text("package lending // corrected required borrowing boundary\n")})
	}
	if strings.Contains(prompt, "everything else in /work is read-only") || strings.Contains(prompt, "get the Orchestrator's receipt") {
		s.owner.t.Fatal("legacy enforcement or receipt mandate leaked into an assignment brief")
	}
	exit := 0
	output := "controlled ordinary observation"
	if s.owner.oversizedOutput {
		output = strings.Repeat("large native test output\n", 1<<16)
	}
	return run.TurnResult{Completed: true, Final: "Completed own work and relevant integration.", Commands: []run.CommandObservation{{Command: "go test ./...", Status: "exited", ExitCode: &exit, Output: output}}}, nil
}
func (s *realizationAgent) Close(context.Context) (run.TurnResult, error) {
	s.closed = true
	if s.owner.closeFailure && strings.HasSuffix(s.spec.ID, "-work-0") {
		return run.TurnResult{}, errors.New("injected unknown stop")
	}
	return run.TurnResult{Stopped: true}, nil
}

func realizationRequest(t *testing.T) (apply.AssignmentRealizationRequest, *realizationAgents) {
	legacy, _ := layoutRequest(t)
	work := &realizationCopies{t: t}
	agents := &realizationAgents{t: t, copies: work}
	homes := make(map[string]bool)
	var homesMu sync.Mutex
	return apply.AssignmentRealizationRequest{ID: "controlled", Build: "test-build", Prepared: legacy.Prepared, OwnerRequest: "Realize the fixed model", MaxMessages: 50, MessageTimeout: time.Second, MaxEntries: 1000, MaxBytes: 1 << 20, TranscriptDir: t.TempDir(), Workspaces: work, Agents: agents, Integration: work, Checks: &passingRunner{}, CheckRequest: checkRequest, CreateHome: func(_ context.Context, home string) error {
		homesMu.Lock()
		defer homesMu.Unlock()
		if homes[home] {
			return errors.New("home exists")
		}
		homes[home] = true
		return nil
	}}, agents
}

func TestAssignmentRealizationSeparatesAssessmentWorkAndVerify(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, agents := realizationRequest(t)
	result, err := apply.RealizeAssignments(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.AcceptedCommit == "" || result.AcceptedCommit != result.Root.ResultCommit || result.TreeCommit != result.AcceptedCommit || !result.NoChange || len(result.Reviews) != 1 || len(result.Checks) != 1 {
		t.Fatalf("named no-change result: %+v", result)
	}
	if result.Assessment.Workspace.ID() == result.Work.Flow.Workspace(".").ID() || result.Reviews[0].Workspace.ID() == result.Work.Flow.Workspace(".").ID() {
		t.Fatal("independent mandates shared a working copy")
	}
	if result.Checks[0].Commit != result.AcceptedCommit || result.Checks[0].Receipt.Tree != result.Tree.Digest() {
		t.Fatal("required check was not bound to the accepted result")
	}
	for id, agent := range agents.opened {
		if !agent.closed {
			t.Fatalf("%s was not closed", id)
		}
	}
	for _, record := range result.Work.Flow.Results() {
		if record.Handoff.ResultCommit == "" || len(record.Observations) == 0 {
			t.Fatal("work handoff lost its ordinary observations")
		}
	}
	records := result.Work.Flow.Results()
	*records[0].Observations[0].Commands[0].ExitCode = 99
	if *result.Work.Flow.Results()[0].Observations[0].Commands[0].ExitCode != 0 {
		t.Fatal("caller changed a retained native result")
	}
}

func TestAssignmentRealizationBoundsVerifyProjectionAndRetainsFullNativeOutput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, agents := realizationRequest(t)
	agents.oversizedOutput = true
	result, err := apply.RealizeAssignments(ctx, req)
	if err != nil || result.AcceptedCommit == "" {
		t.Fatalf("bounded review: %v", err)
	}
	if len(result.Work.Flow.Results()[0].Observations[0].Commands[0].Output) < 1<<20 {
		t.Fatal("prompt projection clipped the retained native result")
	}
}

func TestAssignmentRealizationStopsNewModelProblemsDuringWorkOrVerify(t *testing.T) {
	for _, where := range []string{"implement", "verify"} {
		t.Run(where, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			req, agents := realizationRequest(t)
			agents.implementationGap, agents.verifyGap = where == "implement", where == "verify"
			result, err := apply.RealizeAssignments(ctx, req)
			if result.ModelProblem == nil || result.AcceptedCommit != "" || result.Work == nil {
				t.Fatalf("model problem was lost: %+v %v", result, err)
			}
			if where == "implement" && (!errors.Is(err, run.ErrAssignmentModelProblem) || len(result.Checks) != 0) {
				t.Fatal("confirmed intent problem did not stop implementation immediately")
			}
			if where == "implement" {
				topology, topologyErr := organization.BuildTopology(req.Prepared.Structural().Model().Checked())
				if topologyErr != nil || len(agents.opened) != len(topology.Managers())+1 || len(result.Reviews) != 0 {
					t.Fatalf("whole dormant tree/assessment startup or Verify stop incorrect: %d Agents, %v", len(agents.opened), topologyErr)
				}
				for id, agent := range agents.opened {
					if agent.mode == "implement" && !strings.HasSuffix(id, "-work-0") && agent.messages != 0 {
						t.Fatalf("dormant child %s began work after Root's model problem", id)
					}
					if !agent.closed || agents.copies.copies[run.WorkspaceID(agent.spec.Workspace)] == nil {
						t.Fatalf("model problem lost cleanup or retained copy for %s", id)
					}
					if (agent.mode == "assess" || strings.HasSuffix(id, "-work-0")) && agent.messages != 1 {
						t.Fatalf("assessment/Root message count changed: %s=%d", id, agent.messages)
					}
				}
				if len(result.Work.Flow.Observations(".")) != 1 || !strings.Contains(result.Work.Flow.Observations(".")[0].Final, "model problem") {
					t.Fatal("Root model-problem observation was discarded")
				}
			}
			if where == "verify" && (err != nil || len(result.Reviews) != 1) {
				t.Fatal("final Verify model problem was not retained separately")
			}
		})
	}
}

func TestAssignmentRealizationRefusesChangedRequiredCheckBeforeEffects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, agents := realizationRequest(t)
	req.CheckRequest = func(s apply.CheckSelection) (run.CheckRequest, error) {
		r, err := checkRequest(s)
		r.Args = []string{"version"}
		return r, err
	}
	result, err := apply.RealizeAssignments(ctx, req)
	if err == nil || len(agents.opened) != 0 || result.StartingWorkspace != nil {
		t.Fatal("a substituted check invocation reached Agent work")
	}
}

func TestAssignmentRealizationStopsForModelGapBeforeImplementation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, agents := realizationRequest(t)
	agents.gap = true
	result, err := apply.RealizeAssignments(ctx, req)
	if err != nil || result.ModelProblem == nil || result.Work != nil || len(agents.opened) != 1 || len(result.Checks) != 0 || result.AcceptedCommit != "" || result.StartingWorkspace == nil {
		t.Fatalf("model blocker: %+v %v", result, err)
	}
}

func TestAssignmentRealizationReturnsVerifyDefectThroughOwnerAndChecksNewCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, agents := realizationRequest(t)
	agents.returnOnce = true
	result, err := apply.RealizeAssignments(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.AcceptedCommit == "" || result.NoChange || len(result.Reviews) != 2 || len(result.Reviews[0].Issues) != 1 || result.Reviews[0].AcceptedCommit != "" || len(result.Checks) != 2 || result.Checks[0].Commit == result.Checks[1].Commit || result.Checks[1].Commit != result.AcceptedCommit {
		t.Fatalf("targeted correction: %+v", result)
	}
	if result.Reviews[0].Workspace.ID() == result.Reviews[1].Workspace.ID() {
		t.Fatal("corrected result reused the prior verifier")
	}
}

func TestAssignmentRealizationDoesNotResetAllowanceOrAcceptUncertainClosure(t *testing.T) {
	for _, scenario := range []string{"allowance", "closure"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			req, agents := realizationRequest(t)
			if scenario == "allowance" {
				req.MaxMessages = 1
			} else {
				agents.closeFailure = true
			}
			result, err := apply.RealizeAssignments(ctx, req)
			if err == nil || result.AcceptedCommit != "" || result.StartingWorkspace == nil {
				t.Fatalf("incomplete result: %+v %v", result, err)
			}
			if scenario == "allowance" && result.Messages != 1 {
				t.Fatal("a new purpose reset the invocation allowance")
			}
			if scenario == "closure" && (result.Root.ResultCommit == "" || len(result.Reviews) != 1) {
				t.Fatal("closure failure lost the inspected result")
			}
		})
	}
}
