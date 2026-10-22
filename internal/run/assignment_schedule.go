package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const (
	DelegateChildTool = "delegate_child"
	AcceptChildTool   = "accept_child"
	ReturnChildTool   = "return_child"
	AskChildTool      = "ask_child"
)

var assignmentDecisionTools = []ToolSpec{
	{Name: DelegateChildTool, Description: "Give a required direct child its task and a copy of your current Git stand.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"task":{"type":"string"}},"required":["child","task"],"additionalProperties":false}`)},
	{Name: AcceptChildTool, Description: "Accept the reviewed fixed child commit and integrate it into your Git copy.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"commit":{"type":"string"}},"required":["child","commit"],"additionalProperties":false}`)},
	{Name: ReturnChildTool, Description: "Return a reviewed result to its child with a concrete correction.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"correction":{"type":"string"}},"required":["child","correction"],"additionalProperties":false}`)},
	{Name: AskChildTool, Description: "Ask a different direct child for information needed to answer or integrate work.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"question":{"type":"string"}},"required":["child","question"],"additionalProperties":false}`)},
}

// AssignmentSchedule starts a model-derived Implement tree. MaxMessages bounds
// all Host-to-Agent messages, including direct parent answers. The caller also
// supplies a finite context for Git operations and an absolute log directory.
type AssignmentSchedule struct {
	IDPrefix      string
	Tree          snapshot.Tree
	Task          string
	MaxMessages   int
	Timeout       time.Duration
	TranscriptDir string
	PreparedRoot  AssignmentWorkspace
	// CreateHome exclusively creates the new home before an Agent opens. The
	// legacy RunAssignmentTree wrapper permits nil; OpenAssignmentTree requires it.
	CreateHome func(context.Context, string) error
}

type assignmentAction struct {
	name       string
	child      string
	task       string
	commit     string
	correction string
}

type assignmentMailbox struct {
	mu      sync.Mutex
	actions []assignmentAction
	consult func(context.Context, string, string) (string, error)
}

func (m *assignmentMailbox) handle(ctx context.Context, name string, args json.RawMessage) ToolAnswer {
	var a struct {
		Child      string `json:"child"`
		Task       string `json:"task"`
		Commit     string `json:"commit"`
		Correction string `json:"correction"`
		Question   string `json:"question"`
	}
	if err := json.Unmarshal(args, &a); err != nil || strings.TrimSpace(a.Child) == "" {
		return ToolAnswer{Text: "a direct child is required", Refused: true}
	}
	action := assignmentAction{name: name, child: a.Child, task: strings.TrimSpace(a.Task), commit: a.Commit, correction: strings.TrimSpace(a.Correction)}
	switch name {
	case AskChildTool:
		if strings.TrimSpace(a.Question) == "" || a.Task != "" || a.Commit != "" || a.Correction != "" || m.consult == nil {
			return ToolAnswer{Text: "consultation needs a direct child and question", Refused: true}
		}
		answer, err := m.consult(ctx, a.Child, strings.TrimSpace(a.Question))
		if err != nil {
			return ToolAnswer{Text: err.Error(), Refused: true}
		}
		return ToolAnswer{Text: answer}
	case DelegateChildTool:
		if action.task == "" || a.Commit != "" || a.Correction != "" || a.Question != "" {
			return ToolAnswer{Text: "delegation needs a task", Refused: true}
		}
	case AcceptChildTool:
		if action.commit == "" || a.Task != "" || a.Correction != "" || a.Question != "" {
			return ToolAnswer{Text: "acceptance needs the inspected commit", Refused: true}
		}
	case ReturnChildTool:
		if action.correction == "" || a.Task != "" || a.Commit != "" || a.Question != "" {
			return ToolAnswer{Text: "return needs a concrete correction", Refused: true}
		}
	default:
		return ToolAnswer{Text: "unknown assignment tool", Refused: true}
	}
	m.mu.Lock()
	m.actions = append(m.actions, action)
	m.mu.Unlock()
	return ToolAnswer{Text: "Decision received; the Host will apply it after this response."}
}

func (m *assignmentMailbox) take() []assignmentAction {
	m.mu.Lock()
	defer m.mu.Unlock()
	actions := m.actions
	m.actions = nil
	return actions
}

type assignmentScheduler struct {
	flow      *AssignmentHandoffs
	team      map[string]Manager
	mailboxes map[string]*assignmentMailbox
	ids       map[string]WorkspaceID
	options   AssignmentSchedule
	mu        sync.Mutex
	used      int
	homes     []string
}

// AssignmentTree keeps the Implement assignments alive for targeted correction
// after independent Verify. The caller serializes work/correction operations,
// closes the tree once finished, and owns the retained workspaces and homes.
type AssignmentTree struct {
	Flow      *AssignmentHandoffs
	scheduler *assignmentScheduler
	team      []Manager
	ready     bool
	begun     bool
	closed    bool
	closeErr  error
}

func OpenAssignmentTree(ctx context.Context, team []Manager, workspaces AssignmentWorkspaces, agents AssignmentAgents, integration AssignmentIntegration, options AssignmentSchedule) (*AssignmentTree, error) {
	if options.CreateHome == nil {
		return nil, errors.New("assignment tree needs exclusive fresh home creation")
	}
	return openAssignmentTree(ctx, team, workspaces, agents, integration, options)
}

// RunAssignmentTree lets each Manager delegate, answer questions, inspect and
// decide on its direct children. It returns the fixed Root result and retained
// workspaces; A3 owns invoking it around assessment and final Verify.
func RunAssignmentTree(ctx context.Context, team []Manager, workspaces AssignmentWorkspaces, agents AssignmentAgents, integration AssignmentIntegration, options AssignmentSchedule) (result AssignmentHandoff, flow *AssignmentHandoffs, err error) {
	tree, err := openAssignmentTree(ctx, team, workspaces, agents, integration, options)
	if tree == nil {
		return AssignmentHandoff{}, nil, err
	}
	defer func() { err = errors.Join(err, tree.Close()) }()
	if err != nil {
		return AssignmentHandoff{}, tree.Flow, err
	}
	result, err = tree.Implement(ctx)
	return result, tree.Flow, err
}

func openAssignmentTree(ctx context.Context, team []Manager, workspaces AssignmentWorkspaces, agents AssignmentAgents, integration AssignmentIntegration, options AssignmentSchedule) (*AssignmentTree, error) {
	if _, bounded := ctx.Deadline(); !bounded || !assignmentName.MatchString(options.IDPrefix) || len(options.IDPrefix) > 75 || options.MaxMessages < 1 || options.Timeout <= 0 || strings.TrimSpace(options.Task) == "" || !filepath.IsAbs(options.TranscriptDir) {
		return nil, errors.New("assignment tree needs an identity, task, absolute transcript directory and finite message and time bounds")
	}
	flow, err := NewAssignmentHandoffs(team, workspaces, agents, integration)
	if err != nil {
		return nil, err
	}
	s := &assignmentScheduler{flow: flow, team: make(map[string]Manager), mailboxes: make(map[string]*assignmentMailbox), ids: make(map[string]WorkspaceID), options: options}
	tree := &AssignmentTree{Flow: flow, scheduler: s, team: append([]Manager(nil), team...)}
	for i, manager := range team {
		if manager.ExecuteProfile == "" {
			return tree, fmt.Errorf("Manager %q has no Execute profile", manager.Path)
		}
		s.team[manager.Path] = manager
		s.mailboxes[manager.Path] = &assignmentMailbox{}
		path := manager.Path
		s.mailboxes[path].consult = func(ctx context.Context, child, question string) (string, error) {
			return s.consultChild(ctx, path, child, question)
		}
		s.ids[manager.Path] = WorkspaceID(options.IDPrefix + "-" + strconv.Itoa(i))
	}
	flow.parentAnswer = s.parentAnswer
	root, err := s.request(ctx, ".")
	if err != nil {
		return tree, err
	}
	root.Tree = options.Tree
	root.Prepared = options.PreparedRoot
	if err = flow.StartRoot(ctx, root); err != nil {
		return tree, err
	}
	return tree, nil
}

func (t *AssignmentTree) Implement(ctx context.Context) (AssignmentHandoff, error) {
	if t == nil || t.closed || t.begun {
		return AssignmentHandoff{}, errors.New("assignment tree cannot start this work")
	}
	t.begun = true
	result, err := t.scheduler.work(ctx, ".", t.scheduler.options.Task)
	t.ready = err == nil
	return result, err
}

// Correct asks Root to route concrete corrections through its direct owners.
// Unaffected children keep their assignments, commits and accepted integration.
func (t *AssignmentTree) Correct(ctx context.Context, correction string) (AssignmentHandoff, error) {
	if t == nil || t.closed || !t.ready || strings.TrimSpace(correction) == "" {
		return AssignmentHandoff{}, errors.New("correction needs a completed open assignment tree and concrete task")
	}
	t.ready = false
	s := t.scheduler
	if err := s.message(ctx, ".", "Independent Verify returned required corrections. Correct your own work and use return_child for work owned below you; do not take over another owner's responsibility. Preserve unaffected results.\n"+correction); err != nil {
		return AssignmentHandoff{}, err
	}
	result, err := s.finishCorrection(ctx, ".")
	t.ready = err == nil
	return result, err
}

func (t *AssignmentTree) Homes() []string {
	if t == nil {
		return nil
	}
	return append([]string(nil), t.scheduler.homes...)
}

func (t *AssignmentTree) Close() error {
	if t == nil {
		return nil
	}
	if !t.closed {
		t.closed = true
		t.closeErr = t.scheduler.closeAgents(t.team)
	}
	return t.closeErr
}

func (s *assignmentScheduler) request(ctx context.Context, path string) (AssignmentRequest, error) {
	id := s.ids[path]
	home := string(id) + "-home"
	s.homes = append(s.homes, home)
	if s.options.CreateHome != nil {
		if err := s.options.CreateHome(ctx, home); err != nil {
			return AssignmentRequest{}, err
		}
	}
	return AssignmentRequest{ID: id, Profile: s.team[path].ExecuteProfile, Home: home, Transcript: filepath.Join(s.options.TranscriptDir, string(id)+".jsonl"), Tools: assignmentDecisionTools, Handler: s.mailboxes[path].handle}, nil
}

func (s *assignmentScheduler) reserveMessage() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.used >= s.options.MaxMessages {
		return errors.New("assignment message allowance exhausted")
	}
	s.used++
	return nil
}

func (s *assignmentScheduler) messageResult(ctx context.Context, path, prompt string) (TurnResult, error) {
	stack, _ := ctx.Value(assignmentMessageStackKey{}).([]string)
	for _, active := range stack {
		if active == path {
			return TurnResult{}, fmt.Errorf("Manager %q is already answering in this question chain", path)
		}
	}
	if err := s.reserveMessage(); err != nil {
		return TurnResult{}, err
	}
	ctx = context.WithValue(ctx, assignmentMessageStackKey{}, append(append([]string(nil), stack...), path))
	answer, err := s.flow.Message(ctx, path, prompt, s.options.Timeout)
	if err != nil {
		return answer, err
	}
	if !answer.Completed {
		return answer, fmt.Errorf("Manager %q did not complete its response", path)
	}
	return answer, nil
}

type assignmentMessageStackKey struct{}

func (s *assignmentScheduler) message(ctx context.Context, path, prompt string) error {
	_, err := s.messageResult(ctx, path, prompt)
	return err
}

func (s *assignmentScheduler) parentAnswer(ctx context.Context, parent, child, question string) (string, error) {
	answer, err := s.messageResult(ctx, parent, fmt.Sprintf("Your direct child %s asks: %s", child, question))
	return answer.Final, err
}

func (s *assignmentScheduler) consultChild(ctx context.Context, parent, child, question string) (string, error) {
	if s.team[child].Parent != parent || child == parent {
		return "", fmt.Errorf("Manager %q can ask only a direct child, not %q", parent, child)
	}
	answer, err := s.messageResult(ctx, child, fmt.Sprintf("Your parent %s asks: %s", parent, question))
	return answer.Final, err
}

type assignmentClose struct {
	path string
	done chan struct {
		result TurnResult
		err    error
	}
}

func (s *assignmentScheduler) closeAgents(team []Manager) error {
	limit := min(s.options.Timeout, time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	var attempts []assignmentClose
	for i := len(team) - 1; i >= 0; i-- {
		path := team[i].Path
		if s.flow.team[path].agent == nil {
			continue
		}
		attempt := assignmentClose{path: path, done: make(chan struct {
			result TurnResult
			err    error
		}, 1)}
		attempts = append(attempts, attempt)
		go func(a assignmentClose) {
			result, err := s.flow.Close(ctx, a.path)
			a.done <- struct {
				result TurnResult
				err    error
			}{result, err}
		}(attempt)
	}
	var all error
	for _, attempt := range attempts {
		select {
		case close := <-attempt.done:
			if !close.result.Stopped {
				all = errors.Join(all, fmt.Errorf("Agent %q stop was not observed", attempt.path))
			}
			all = errors.Join(all, close.err)
		case <-ctx.Done():
			all = errors.Join(all, fmt.Errorf("Agent %q close outcome uncertain: %w", attempt.path, ctx.Err()))
		}
	}
	return all
}

func (s *assignmentScheduler) work(ctx context.Context, path, task string) (AssignmentHandoff, error) {
	manager := s.team[path]
	prompt := fmt.Sprintf("Your responsibility is %s. %s\nTask from your parent: %s\nRequired direct children: %s. Give each child a concrete task using delegate_child. Work in your own Git copy; the Host will later show each fixed child result for your review.", manager.Role, manager.Brief, task, strings.Join(manager.Children, ", "))
	if err := s.message(ctx, path, prompt); err != nil {
		return AssignmentHandoff{}, err
	}
	delegations := s.mailboxes[path].take()
	seen := make(map[string]bool)
	for _, action := range delegations {
		if action.name != DelegateChildTool || s.team[action.child].Parent != path || seen[action.child] {
			return AssignmentHandoff{}, fmt.Errorf("Manager %q made an invalid direct-child delegation to %q", path, action.child)
		}
		seen[action.child] = true
	}
	if len(seen) != len(manager.Children) {
		return AssignmentHandoff{}, fmt.Errorf("Manager %q did not delegate every required child", path)
	}
	for _, action := range delegations {
		request, err := s.request(ctx, action.child)
		if err != nil {
			return AssignmentHandoff{}, err
		}
		if err := s.flow.DelegateCurrent(ctx, path, action.child, request); err != nil {
			return AssignmentHandoff{}, err
		}
	}
	for _, action := range delegations {
		if _, err := s.work(ctx, action.child, action.task); err != nil {
			return AssignmentHandoff{}, err
		}
		if err := s.review(ctx, path, action.child); err != nil {
			return AssignmentHandoff{}, err
		}
	}
	if len(manager.Children) != 0 {
		if err := s.message(ctx, path, "All direct-child results are integrated. Check the composition and relevant tests in your Git copy. Summarize your result, observations and open matters for your parent."); err != nil {
			return AssignmentHandoff{}, err
		}
		if actions := s.mailboxes[path].take(); len(actions) != 0 {
			return AssignmentHandoff{}, fmt.Errorf("Manager %q made an unexpected decision after integration", path)
		}
	}
	return s.flow.Submit(ctx, path)
}

func (s *assignmentScheduler) review(ctx context.Context, parent, child string) error {
	for {
		changes, handoff, err := s.flow.Changes(ctx, parent, child)
		if err != nil {
			return err
		}
		prompt := fmt.Sprintf("Review your direct child %s's fixed result %s from %s. Child summary: %s\nObserved commands from its latest response:\n%s\nActual Git changes:\n%s\nInspect conformance and tests. Use accept_child with that commit or return_child with a concrete correction. Your decision must be yours.", child, handoff.ResultCommit, handoff.StartingCommit, handoff.Summary, s.commandObservations(child), changes)
		if err := s.message(ctx, parent, prompt); err != nil {
			return err
		}
		decision := s.mailboxes[parent].take()
		if len(decision) != 1 || decision[0].child != child {
			return fmt.Errorf("Manager %q did not decide on its direct child %q", parent, child)
		}
		a := decision[0]
		switch a.name {
		case ReturnChildTool:
			if err := s.correct(ctx, parent, child, a.correction); err != nil {
				return err
			}
		case AcceptChildTool:
			if a.commit != handoff.ResultCommit {
				return fmt.Errorf("Manager %q accepted a different commit for %q", parent, child)
			}
			_, err := s.flow.Accept(ctx, parent, child, a.commit)
			if errors.Is(err, ErrIntegrationConflict) {
				if err := s.message(ctx, parent, fmt.Sprintf("Git found a real conflict integrating %s at %s: %v. Give the owning child a concrete correction using return_child.", child, a.commit, err)); err != nil {
					return err
				}
				correction := s.mailboxes[parent].take()
				if len(correction) != 1 || correction[0].name != ReturnChildTool || correction[0].child != child {
					return fmt.Errorf("Manager %q did not route the Git conflict to %q", parent, child)
				}
				if err := s.correct(ctx, parent, child, correction[0].correction); err != nil {
					return err
				}
				continue
			}
			return err
		default:
			return fmt.Errorf("Manager %q made no acceptance or return decision for %q", parent, child)
		}
	}
}

func (s *assignmentScheduler) correct(ctx context.Context, parent, child, correction string) error {
	if err := s.reserveMessage(); err != nil {
		return err
	}
	answer, err := s.flow.Return(ctx, parent, child, correction, s.options.Timeout)
	if err != nil {
		return err
	}
	if !answer.Completed {
		return fmt.Errorf("Manager %q did not complete its correction", child)
	}
	_, err = s.finishCorrection(ctx, child)
	return err
}

func (s *assignmentScheduler) finishCorrection(ctx context.Context, path string) (AssignmentHandoff, error) {
	actions := s.mailboxes[path].take()
	seen := make(map[string]bool)
	for _, a := range actions {
		if a.name != ReturnChildTool || s.team[a.child].Parent != path || seen[a.child] {
			return AssignmentHandoff{}, fmt.Errorf("Manager %q must route each correction to a distinct direct child", path)
		}
		seen[a.child] = true
		if err := s.correct(ctx, path, a.child, a.correction); err != nil {
			return AssignmentHandoff{}, err
		}
		if err := s.review(ctx, path, a.child); err != nil {
			return AssignmentHandoff{}, err
		}
	}
	if len(actions) > 0 {
		if err := s.message(ctx, path, "Corrected child results are integrated. Check the changed composition and relevant tests; preserve unaffected work. Summarize the corrected Git result and any open matters."); err != nil {
			return AssignmentHandoff{}, err
		}
		if len(s.mailboxes[path].take()) != 0 {
			return AssignmentHandoff{}, fmt.Errorf("Manager %q left unresolved correction decisions after integration", path)
		}
	}
	return s.flow.Submit(ctx, path)
}

func (s *assignmentScheduler) commandObservations(path string) string {
	observed := s.flow.Observations(path)
	if len(observed) == 0 || len(observed[len(observed)-1].Commands) == 0 {
		return "No command observations reported."
	}
	var b strings.Builder
	for _, command := range observed[len(observed)-1].Commands {
		name, output := command.Command, command.Output
		if len(name) > 2048 {
			name = name[:2048] + " [truncated]"
		}
		if len(output) > 4096 {
			output = output[:4096] + " [truncated]"
		}
		fmt.Fprintf(&b, "- %s: %s", name, command.Status)
		if command.ExitCode != nil {
			fmt.Fprintf(&b, " (exit %d)", *command.ExitCode)
		}
		if output != "" {
			fmt.Fprintf(&b, ": %s", output)
		}
		b.WriteByte('\n')
		if b.Len() > 16<<10 {
			return b.String()[:16<<10] + "\n[command observations truncated]"
		}
	}
	return b.String()
}
