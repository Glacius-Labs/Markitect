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

var ErrAssignmentModelProblem = errors.New("Root confirmed a material problem of the fixed model")

var assignmentDecisionTools = []ToolSpec{
	{Name: DelegateChildTool, Description: "Give a required direct child its task and a copy of your current Git stand.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"task":{"type":"string"}},"required":["child","task"],"additionalProperties":false}`)},
	{Name: AcceptChildTool, Description: "Accept the direct child result currently presented for review. The Host binds the exact fixed handoff and integrates it into your Git copy.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"}},"required":["child"],"additionalProperties":false}`)},
	{Name: ReturnChildTool, Description: "Return a reviewed result to its child with a concrete correction.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"correction":{"type":"string"}},"required":["child","correction"],"additionalProperties":false}`)},
	{Name: AskChildTool, Description: "Queue a question to a direct child. A working child receives it after its current step; its answer arrives in a follow-up message to you. Do not block or invent an answer while waiting.", InputSchema: json.RawMessage(`{"type":"object","properties":{"child":{"type":"string"},"question":{"type":"string"}},"required":["child","question"],"additionalProperties":false}`)},
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
	// Independent branches can call it concurrently with distinct home names.
	CreateHome func(context.Context, string) error
}

type assignmentAction struct {
	name       string
	child      string
	task       string
	correction string
}

type assignmentScheduler struct {
	flow            *AssignmentHandoffs
	team            map[string]Manager
	mailboxes       map[string]*assignmentMailbox
	ids             map[string]WorkspaceID
	options         AssignmentSchedule
	mu              sync.Mutex
	used            int
	homes           []string
	problem         *ReviewIssue
	closed          bool
	questionsCtx    context.Context
	cancelQuestions context.CancelFunc
	questionWorkers sync.WaitGroup
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
	if _, ok := workspaces.(AssignmentUpdates); !ok {
		return nil, errors.New("assignment tree needs retained workspace updates")
	}
	flow, err := NewAssignmentHandoffs(team, workspaces, agents, integration)
	if err != nil {
		return nil, err
	}
	s := &assignmentScheduler{flow: flow, team: make(map[string]Manager), mailboxes: make(map[string]*assignmentMailbox), ids: make(map[string]WorkspaceID), options: options}
	s.questionsCtx, s.cancelQuestions = context.WithCancel(ctx)
	tree := &AssignmentTree{Flow: flow, scheduler: s, team: append([]Manager(nil), team...)}
	for i, manager := range team {
		if manager.ExecuteProfile == "" {
			return tree, fmt.Errorf("Manager %q has no Execute profile", manager.Path)
		}
		s.team[manager.Path] = manager
		s.mailboxes[manager.Path] = &assignmentMailbox{path: manager.Path, scheduler: s}
		s.ids[manager.Path] = WorkspaceID(options.IDPrefix + "-" + strconv.Itoa(i))
	}
	flow.parentAnswer = s.parentAnswer
	if err = s.openAll(ctx, team); err != nil {
		return tree, err
	}
	return tree, nil
}

func (t *AssignmentTree) Implement(ctx context.Context) (AssignmentHandoff, error) {
	if t == nil || t.closed || t.begun {
		return AssignmentHandoff{}, errors.New("assignment tree cannot start this work")
	}
	t.begun = true
	result, err := t.scheduler.work(ctx, ".", t.scheduler.options.Task, "")
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
	result, err := s.correctTask(ctx, ".", "Independent Verify returned required corrections. Correct your own work and use return_child for work owned below you; do not take over another owner's responsibility. Preserve unaffected results.\n"+correction)
	t.ready = err == nil
	return result, err
}

func (t *AssignmentTree) Homes() []string {
	if t == nil {
		return nil
	}
	t.scheduler.mu.Lock()
	defer t.scheduler.mu.Unlock()
	return append([]string(nil), t.scheduler.homes...)
}

func (t *AssignmentTree) ModelProblem() *ReviewIssue {
	if t == nil {
		return nil
	}
	s := t.scheduler
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.problem == nil {
		return nil
	}
	issue := *s.problem
	return &issue
}

func (t *AssignmentTree) Close() error {
	if t == nil {
		return nil
	}
	if !t.closed {
		t.closed = true
		t.scheduler.mu.Lock()
		t.scheduler.closed = true
		t.scheduler.cancelQuestions()
		t.scheduler.mu.Unlock()
		t.closeErr = errors.Join(t.scheduler.closeAgents(t.team), t.scheduler.waitQuestions())
	}
	return t.closeErr
}

func (s *assignmentScheduler) request(ctx context.Context, path string) (AssignmentRequest, error) {
	id := s.ids[path]
	home := string(id) + "-home"
	s.mu.Lock()
	s.homes = append(s.homes, home)
	s.mu.Unlock()
	if s.options.CreateHome != nil {
		if err := s.options.CreateHome(ctx, home); err != nil {
			return AssignmentRequest{}, err
		}
	}
	var tools []ToolSpec
	if len(s.team[path].Children) > 0 {
		tools = assignmentDecisionTools
	}
	handler := s.mailboxes[path].handle
	if path == "." {
		tools = append(append([]ToolSpec(nil), tools...), ToolSpec{Name: ModelProblemTool, Description: "End affected implementation for a material fixed-model problem you confirm, naming its statements and incompatible outcomes.", InputSchema: reviewIssueSchema()})
		handler = func(ctx context.Context, name string, args json.RawMessage) ToolAnswer {
			if name != ModelProblemTool {
				return s.mailboxes[path].handle(ctx, name, args)
			}
			var issue ReviewIssue
			if json.Unmarshal(args, &issue) != nil || strings.TrimSpace(issue.Owner) == "" || strings.TrimSpace(issue.Obligation) == "" || strings.TrimSpace(issue.Expected) == "" || strings.TrimSpace(issue.Observed) == "" || ctx.Err() != nil {
				return ToolAnswer{Text: "a confirmed model problem needs its owner, statements and material alternatives", Refused: true}
			}
			s.mu.Lock()
			if s.closed {
				s.mu.Unlock()
				return ToolAnswer{Text: "assignment tree is closing", Refused: true}
			}
			if s.problem == nil {
				s.problem = &issue
			}
			s.mu.Unlock()
			return ToolAnswer{Text: "Confirmed model problem retained; affected work ends without acceptance."}
		}
	}
	return AssignmentRequest{ID: id, Profile: s.team[path].ExecuteProfile, Home: home, Transcript: filepath.Join(s.options.TranscriptDir, string(id)+".jsonl"), Tools: tools, Handler: handler}, nil
}

func (s *assignmentScheduler) reserveMessage() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.used >= s.options.MaxMessages {
		return errors.New("assignment message allowance exhausted")
	}
	if s.closed {
		return errors.New("assignment tree is closing")
	}
	if s.problem != nil {
		return ErrAssignmentModelProblem
	}
	s.used++
	return nil
}

type assignmentClose struct {
	path string
	done chan struct {
		result TurnResult
		err    error
	}
}

func (s *assignmentScheduler) closeAgents(team []Manager) error {
	paths := make([]string, 0, len(team))
	for i := len(team) - 1; i >= 0; i-- {
		paths = append(paths, team[i].Path)
	}
	return s.closePaths(paths)
}

func (s *assignmentScheduler) closePaths(paths []string) error {
	limit := min(s.options.Timeout, time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	var attempts []assignmentClose
	for _, path := range paths {
		s.flow.mu.Lock()
		started := s.flow.team[path].agent != nil
		s.flow.mu.Unlock()
		if !started {
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

func (s *assignmentScheduler) commandObservations(path, commit string) string {
	var observed []TurnResult
	for _, result := range s.flow.Results() {
		if result.Manager == path && result.Handoff.ResultCommit == commit {
			observed = result.Observations
		}
	}
	if len(observed) == 0 {
		return "No command observations reported."
	}
	var b strings.Builder
	for _, response := range observed {
		for _, command := range response.Commands {
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
	}
	if b.Len() == 0 {
		return "No command observations reported."
	}
	return b.String()
}
