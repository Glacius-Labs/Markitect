package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
)

var assignmentDecisionTools = []ToolSpec{
	{Name: DelegateChildTool, Description: "Give a required direct child its task and a copy of your current Git stand.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"task":{"type":"string"}},"required":["child","task"],"additionalProperties":false}`)},
	{Name: AcceptChildTool, Description: "Accept the reviewed fixed child commit and integrate it into your Git copy.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"commit":{"type":"string"}},"required":["child","commit"],"additionalProperties":false}`)},
	{Name: ReturnChildTool, Description: "Return a reviewed result to its child with a concrete correction.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"correction":{"type":"string"}},"required":["child","correction"],"additionalProperties":false}`)},
}

// AssignmentSchedule starts a model-derived Implement tree. MaxMessages bounds
// the scheduler's Agent messages; the caller also supplies a finite context for
// the whole assignment, including direct parent answers and Git operations.
type AssignmentSchedule struct {
	IDPrefix    string
	Tree        snapshot.Tree
	Task        string
	MaxMessages int
	Timeout     time.Duration
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
}

func (m *assignmentMailbox) handle(_ context.Context, name string, args json.RawMessage) ToolAnswer {
	var a struct {
		Child      string `json:"child"`
		Task       string `json:"task"`
		Commit     string `json:"commit"`
		Correction string `json:"correction"`
	}
	if err := json.Unmarshal(args, &a); err != nil || strings.TrimSpace(a.Child) == "" {
		return ToolAnswer{Text: "a direct child is required", Refused: true}
	}
	action := assignmentAction{name: name, child: a.Child, task: strings.TrimSpace(a.Task), commit: a.Commit, correction: strings.TrimSpace(a.Correction)}
	switch name {
	case DelegateChildTool:
		if action.task == "" || a.Commit != "" || a.Correction != "" {
			return ToolAnswer{Text: "delegation needs a task", Refused: true}
		}
	case AcceptChildTool:
		if action.commit == "" || a.Task != "" || a.Correction != "" {
			return ToolAnswer{Text: "acceptance needs the inspected commit", Refused: true}
		}
	case ReturnChildTool:
		if action.correction == "" || a.Task != "" || a.Commit != "" {
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
	used      int
}

// RunAssignmentTree lets each Manager delegate, answer questions, inspect and
// decide on its direct children. It returns the fixed Root result and retained
// workspaces; A3 owns invoking it around assessment and final Verify.
func RunAssignmentTree(ctx context.Context, team []Manager, workspaces AssignmentWorkspaces, agents AssignmentAgents, integration AssignmentIntegration, options AssignmentSchedule) (result AssignmentHandoff, flow *AssignmentHandoffs, err error) {
	if _, bounded := ctx.Deadline(); !bounded || !assignmentName.MatchString(options.IDPrefix) || len(options.IDPrefix) > 75 || options.MaxMessages < 1 || options.Timeout <= 0 || strings.TrimSpace(options.Task) == "" {
		return AssignmentHandoff{}, nil, errors.New("assignment tree needs an identity, task and finite message and time bounds")
	}
	flow, err = NewAssignmentHandoffs(team, workspaces, agents, integration)
	if err != nil {
		return AssignmentHandoff{}, nil, err
	}
	s := &assignmentScheduler{flow: flow, team: make(map[string]Manager), mailboxes: make(map[string]*assignmentMailbox), ids: make(map[string]WorkspaceID), options: options}
	for i, manager := range team {
		if manager.ExecuteProfile == "" {
			return AssignmentHandoff{}, flow, fmt.Errorf("Manager %q has no Execute profile", manager.Path)
		}
		s.team[manager.Path] = manager
		s.mailboxes[manager.Path] = &assignmentMailbox{}
		s.ids[manager.Path] = WorkspaceID(options.IDPrefix + "-" + strconv.Itoa(i))
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		for i := len(team) - 1; i >= 0; i-- {
			if flow.team[team[i].Path].agent != nil {
				_, closeErr := flow.Close(closeCtx, team[i].Path)
				err = errors.Join(err, closeErr)
			}
		}
	}()
	root := s.request(".")
	root.Tree = options.Tree
	if err = flow.StartRoot(ctx, root); err != nil {
		return AssignmentHandoff{}, flow, err
	}
	result, err = s.work(ctx, ".", options.Task)
	return result, flow, err
}

func (s *assignmentScheduler) request(path string) AssignmentRequest {
	id := s.ids[path]
	return AssignmentRequest{ID: id, Profile: s.team[path].ExecuteProfile, Home: string(id) + "-home", Tools: assignmentDecisionTools, Handler: s.mailboxes[path].handle}
}

func (s *assignmentScheduler) message(ctx context.Context, path, prompt string) error {
	if s.used >= s.options.MaxMessages {
		return errors.New("assignment message allowance exhausted")
	}
	s.used++
	answer, err := s.flow.Message(ctx, path, prompt, s.options.Timeout)
	if err != nil {
		return err
	}
	if !answer.Completed {
		return fmt.Errorf("Manager %q did not complete its response", path)
	}
	return nil
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
		if err := s.flow.DelegateCurrent(ctx, path, action.child, s.request(action.child)); err != nil {
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
		prompt := fmt.Sprintf("Review your direct child %s's fixed result %s from %s. Child summary: %s\nActual Git changes:\n%s\nInspect conformance and tests. Use accept_child with that commit or return_child with a concrete correction. Your decision must be yours.", child, handoff.ResultCommit, handoff.StartingCommit, handoff.Summary, changes)
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
	if s.used >= s.options.MaxMessages {
		return errors.New("assignment message allowance exhausted")
	}
	s.used++
	answer, err := s.flow.Return(ctx, parent, child, correction, s.options.Timeout)
	if err != nil {
		return err
	}
	if !answer.Completed {
		return fmt.Errorf("Manager %q did not complete its correction", child)
	}
	if actions := s.mailboxes[child].take(); len(actions) != 0 {
		return fmt.Errorf("Manager %q made an unexpected new delegation during correction", child)
	}
	_, err = s.flow.Submit(ctx, child)
	return err
}
