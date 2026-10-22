package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

// ErrIntegrationConflict distinguishes an ordinary Git conflict from an
// integration failure whose outcome is incomplete or uncertain.
var ErrIntegrationConflict = errors.New("child Git result conflicts with parent stand")

// AssignmentIntegration supplies the Git operations needed when a parent
// inspects and accepts a child's fixed result. It never changes the owner's
// repository; both copies are internal Agent workspaces.
type AssignmentIntegration interface {
	Changes(context.Context, AssignmentWorkspace, AssignmentHandoff) (string, error)
	Integrate(context.Context, AssignmentWorkspace, AssignmentWorkspace, AssignmentHandoff, string) (string, error)
}

type assignmentMember struct {
	manager      Manager
	agent        *Assignment
	partial      *Assignment
	transcript   string
	observed     []TurnResult
	results      []AssignmentResult
	observedFrom int
	starting     bool
	handoff      AssignmentHandoff
	inspected    string
	returned     bool
	accepted     map[string]string
	messages     chan struct{}
}

// AssignmentResult retains the native observations delivered since the prior
// handoff with the commit that received them. Commands may precede later edits;
// their relevance to this result remains a review judgment.
type AssignmentResult struct {
	Manager      string
	Handoff      AssignmentHandoff
	Observations []TurnResult
	Transcript   string
}

func (f *AssignmentHandoffs) Results() []AssignmentResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	var paths []string
	for path := range f.team {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	var results []AssignmentResult
	for _, path := range paths {
		for _, r := range f.team[path].results {
			r.Observations = append([]TurnResult(nil), r.Observations...)
			for i := range r.Observations {
				r.Observations[i] = cloneTurnResult(r.Observations[i])
			}
			results = append(results, r)
		}
	}
	return results
}

func (m *assignmentMember) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.held(ctx) {
		return nil
	}
	select {
	case m.messages <- struct{}{}:
		if err := ctx.Err(); err != nil {
			m.release(ctx)
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type assignmentLockKey struct{}

func (m *assignmentMember) held(ctx context.Context) bool {
	held, _ := ctx.Value(assignmentLockKey{}).([]*assignmentMember)
	return slices.Contains(held, m)
}

func (m *assignmentMember) release(ctx context.Context) {
	if !m.held(ctx) {
		<-m.messages
	}
}

// serial keeps one parent's conversation and Git copy together through a
// review decision and its integration. Questions wait on the same queue.
func (f *AssignmentHandoffs) serial(ctx context.Context, path string, work func(context.Context) error) error {
	m := f.team[path]
	if err := m.acquire(ctx); err != nil {
		return err
	}
	defer m.release(ctx)
	held, _ := ctx.Value(assignmentLockKey{}).([]*assignmentMember)
	return work(context.WithValue(ctx, assignmentLockKey{}, append(append([]*assignmentMember(nil), held...), m)))
}

// A parent must not wait on a working child while holding its own queue: that
// child could already be waiting for an answer from this parent.
func (f *AssignmentHandoffs) consult(ctx context.Context, path string, work func(context.Context) error) error {
	m := f.team[path]
	if m == nil {
		return fmt.Errorf("Manager %q has no assignment", path)
	}
	if m.held(ctx) {
		return work(ctx)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case m.messages <- struct{}{}:
		defer m.release(ctx)
		held, _ := ctx.Value(assignmentLockKey{}).([]*assignmentMember)
		return work(context.WithValue(ctx, assignmentLockKey{}, append(append([]*assignmentMember(nil), held...), m)))
	default:
		return fmt.Errorf("child %q is working or waiting; consult its fixed result when available instead of waiting inside a parent answer", path)
	}
}

// AssignmentHandoffs routes work through the model-derived Manager tree. The
// parent owns the substantive review decision; this type enforces only direct
// relationships, fixed Git results and participation of every required child.
// It is the Apply migration seam while Extract continues using Passes.
type AssignmentHandoffs struct {
	mu           sync.Mutex
	team         map[string]*assignmentMember
	workspaces   AssignmentWorkspaces
	agents       AssignmentAgents
	integration  AssignmentIntegration
	parentAnswer func(context.Context, string, string, string) (string, error)
}

func NewAssignmentHandoffs(team []Manager, workspaces AssignmentWorkspaces, agents AssignmentAgents, integration AssignmentIntegration) (*AssignmentHandoffs, error) {
	if len(team) == 0 || team[0].Path != "." || team[0].Parent != "" || workspaces == nil || agents == nil || integration == nil {
		return nil, errors.New("an assignment team needs Root, workspaces, Agents and Git integration")
	}
	f := &AssignmentHandoffs{team: make(map[string]*assignmentMember), workspaces: workspaces, agents: agents, integration: integration}
	for _, manager := range team {
		if manager.Path == "" || f.team[manager.Path] != nil {
			return nil, fmt.Errorf("repeated or empty Manager path %q", manager.Path)
		}
		f.team[manager.Path] = &assignmentMember{manager: manager, accepted: make(map[string]string), messages: make(chan struct{}, 1)}
	}
	for _, manager := range team {
		if manager.Path == "." {
			continue
		}
		parent := f.team[manager.Parent]
		if parent == nil || parent == f.team[manager.Path] {
			return nil, fmt.Errorf("Manager %q lacks a distinct parent", manager.Path)
		}
		found := false
		for _, child := range parent.manager.Children {
			if child == manager.Path {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Manager %q is not a declared direct child of %q", manager.Path, manager.Parent)
		}
	}
	for _, manager := range team {
		seen := make(map[string]bool)
		for _, child := range manager.Children {
			if seen[child] || f.team[child] == nil || f.team[child].manager.Parent != manager.Path {
				return nil, fmt.Errorf("Manager %q has an invalid direct child %q", manager.Path, child)
			}
			seen[child] = true
		}
		path := manager.Path
		for hops := 0; path != "."; hops++ {
			if hops >= len(team) {
				return nil, fmt.Errorf("Manager %q is not connected to Root", manager.Path)
			}
			path = f.team[path].manager.Parent
		}
	}
	return f, nil
}

// StartRoot starts the Root Agent on the caller's captured starting stand.
func (f *AssignmentHandoffs) StartRoot(ctx context.Context, request AssignmentRequest) error {
	return f.start(ctx, "", ".", request)
}

// Delegate starts a required direct child with a stand and task supplied by
// its parent. The caller obtains a fixed parent foundation before deriving the
// child's Tree; no implicit shared-live stand is consulted here.
func (f *AssignmentHandoffs) Delegate(ctx context.Context, parent, child string, request AssignmentRequest) error {
	return f.start(ctx, parent, child, request)
}

// DelegateCurrent starts a direct child from its parent's fixed current Git
// stand. The parent keeps its own copy and conversation while the child works.
func (f *AssignmentHandoffs) DelegateCurrent(ctx context.Context, parent, child string, request AssignmentRequest) error {
	prepared, err := f.prepareChild(ctx, parent, child, request)
	if err != nil {
		return err
	}
	return f.openReserved(ctx, parent, child, prepared)
}

// Prepare sibling foundations under the parent's queue before opening their
// containers in parallel. A prepared copy remains inspectable if startup aborts.
func (f *AssignmentHandoffs) prepareChild(ctx context.Context, parent, child string, request AssignmentRequest) (AssignmentRequest, error) {
	forks, ok := f.workspaces.(AssignmentForks)
	if !ok {
		return request, errors.New("assignment workspaces cannot fork a parent's Git stand")
	}
	f.mu.Lock()
	m, p := f.team[child], f.team[parent]
	if m == nil || p == nil || m.manager.Parent != parent || m.agent != nil || m.starting || p.agent == nil || p.starting {
		f.mu.Unlock()
		return request, fmt.Errorf("Manager %q cannot start under %q", child, parent)
	}
	m.starting = true
	f.mu.Unlock()
	if request.ID == "" {
		request.ID = WorkspaceID("assignment-" + strings.ReplaceAll(child, "/", "-"))
	}
	if err := p.acquire(ctx); err != nil {
		f.mu.Lock()
		m.starting = false
		f.mu.Unlock()
		return request, err
	}
	defer p.release(ctx)
	prepared, err := forks.ForkAssignment(ctx, p.agent.Workspace(), request.ID)
	if err != nil {
		f.mu.Lock()
		if prepared != nil {
			m.partial = &Assignment{work: prepared}
		}
		m.starting = false
		f.mu.Unlock()
		return request, err
	}
	request.Prepared = prepared
	f.mu.Lock()
	m.partial = &Assignment{work: prepared}
	f.mu.Unlock()
	return request, nil
}

func (f *AssignmentHandoffs) start(ctx context.Context, parent, path string, request AssignmentRequest) error {
	f.mu.Lock()
	m := f.team[path]
	if m == nil || m.manager.Parent != parent || m.agent != nil || m.starting || parent != "" && (f.team[parent] == nil || f.team[parent].agent == nil || f.team[parent].starting) {
		f.mu.Unlock()
		return fmt.Errorf("Manager %q cannot start under %q", path, parent)
	}
	// Reserve the single assignment before opening its container.
	m.starting = true
	f.mu.Unlock()
	if request.ID == "" {
		request.ID = WorkspaceID("assignment-" + strings.ReplaceAll(path, "/", "-"))
	}
	return f.openReserved(ctx, parent, path, request)
}

func (f *AssignmentHandoffs) openReserved(ctx context.Context, parent, path string, request AssignmentRequest) error {
	m := f.team[path]
	f.mu.Lock()
	m.transcript = request.Transcript
	f.mu.Unlock()
	if parent != "" {
		request.AskParent = func(ctx context.Context, question string) (string, error) {
			if f.parentAnswer != nil {
				return f.parentAnswer(ctx, parent, path, question)
			}
			f.mu.Lock()
			p := f.team[parent]
			f.mu.Unlock()
			if p == nil || p.agent == nil {
				return "", fmt.Errorf("parent %q is unavailable", parent)
			}
			if err := p.acquire(ctx); err != nil {
				return "", err
			}
			defer p.release(ctx)
			answer, err := p.agent.Message(ctx, fmt.Sprintf("Your direct child %s asks: %s", path, question), time.Minute)
			if err == nil && !answer.Completed {
				err = fmt.Errorf("parent %q did not complete its answer", parent)
			}
			return answer.Final, err
		}
	}
	a, err := StartAssignment(ctx, f.workspaces, f.agents, request)
	f.mu.Lock()
	if err == nil {
		m.agent = a
		m.partial = nil
	} else if a != nil {
		m.partial = a
	}
	m.starting = false
	f.mu.Unlock()
	return err
}

// Message sends an ordinary task, answer or correction to an existing Agent.
func (f *AssignmentHandoffs) Message(ctx context.Context, path, message string, timeout time.Duration) (TurnResult, error) {
	f.mu.Lock()
	m := f.team[path]
	var a *Assignment
	if m != nil {
		a = m.agent
	}
	f.mu.Unlock()
	if a == nil {
		return TurnResult{}, fmt.Errorf("Manager %q has no assignment", path)
	}
	if err := m.acquire(ctx); err != nil {
		return TurnResult{}, err
	}
	defer m.release(ctx)
	result, err := a.Message(ctx, message, timeout)
	f.mu.Lock()
	m.observed = append(m.observed, cloneTurnResult(result))
	f.mu.Unlock()
	return result, err
}

// Workspace exposes a started or partially established Agent copy even after
// failure or container shutdown. The caller owns its eventual release.
func (f *AssignmentHandoffs) Workspace(path string) AssignmentWorkspace {
	f.mu.Lock()
	defer f.mu.Unlock()
	if m := f.team[path]; m != nil {
		if m.agent != nil {
			return m.agent.Workspace()
		}
		if m.partial != nil {
			return m.partial.Workspace()
		}
	}
	return nil
}

// Transcript names the ordinary provider log for this assignment.
func (f *AssignmentHandoffs) Transcript(path string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if m := f.team[path]; m != nil {
		return m.transcript
	}
	return ""
}

// Observations retains native responses, including a partial failed response.
func (f *AssignmentHandoffs) Observations(path string) []TurnResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := f.team[path]
	if m == nil {
		return nil
	}
	results := make([]TurnResult, len(m.observed))
	for i, observed := range m.observed {
		results[i] = cloneTurnResult(observed)
	}
	return results
}

func cloneTurnResult(result TurnResult) TurnResult {
	result.Commands = append([]CommandObservation(nil), result.Commands...)
	for i := range result.Commands {
		if code := result.Commands[i].ExitCode; code != nil {
			value := *code
			result.Commands[i].ExitCode = &value
		}
	}
	return result
}

// RetainedWorkspace exposes a partial starting copy after setup failure. It
// cannot be used as a running parent; a later attempt needs a new assignment ID.
func (f *AssignmentHandoffs) RetainedWorkspace(path string) AssignmentWorkspace {
	f.mu.Lock()
	defer f.mu.Unlock()
	if m := f.team[path]; m != nil && m.partial != nil {
		return m.partial.Workspace()
	}
	return nil
}

// Close ends an Agent's container while preserving its fixed internal results
// and working copy for inspection or later cleanup by the caller.
func (f *AssignmentHandoffs) Close(ctx context.Context, path string) (TurnResult, error) {
	f.mu.Lock()
	m := f.team[path]
	var a *Assignment
	if m != nil {
		a = m.agent
	}
	f.mu.Unlock()
	if a == nil {
		return TurnResult{}, fmt.Errorf("Manager %q has no assignment", path)
	}
	return a.Close(ctx)
}

// Submit records the Agent's current named Git result and last summary. A
// corrected submission replaces the pending result, retaining older commits.
func (f *AssignmentHandoffs) Submit(ctx context.Context, path string) (AssignmentHandoff, error) {
	f.mu.Lock()
	m := f.team[path]
	if m == nil || m.agent == nil {
		f.mu.Unlock()
		return AssignmentHandoff{}, fmt.Errorf("Manager %q has no assignment", path)
	}
	f.mu.Unlock()
	if err := m.acquire(ctx); err != nil {
		return AssignmentHandoff{}, err
	}
	defer m.release(ctx)
	f.mu.Lock()
	for _, child := range m.manager.Children {
		if !f.childIntegrated(m, child) {
			f.mu.Unlock()
			return AssignmentHandoff{}, fmt.Errorf("Manager %q has not integrated required child %q", path, child)
		}
	}
	a := m.agent
	f.mu.Unlock()
	handoff, err := a.Submit(ctx)
	if err != nil {
		return AssignmentHandoff{}, err
	}
	f.mu.Lock()
	for _, child := range m.manager.Children {
		if !f.childIntegrated(m, child) {
			f.mu.Unlock()
			return AssignmentHandoff{}, fmt.Errorf("Manager %q changed required child %q during submission", path, child)
		}
	}
	m.handoff = handoff
	result := AssignmentResult{Manager: path, Handoff: handoff, Transcript: m.transcript}
	for _, o := range m.observed[m.observedFrom:] {
		result.Observations = append(result.Observations, cloneTurnResult(o))
	}
	m.results = append(m.results, result)
	m.observedFrom = len(m.observed)
	m.inspected = ""
	m.returned = false
	f.mu.Unlock()
	return handoff, nil
}

// Caller holds f.mu.
func (f *AssignmentHandoffs) childIntegrated(parent *assignmentMember, child string) bool {
	c := f.team[child]
	return c != nil && !c.returned && parent.accepted[child] != "" && c.handoff.ResultCommit == parent.accepted[child]
}

// Changes gives the parent the child's actual fixed Git difference before it
// decides whether to return or accept the work.
func (f *AssignmentHandoffs) Changes(ctx context.Context, parent, child string) (string, AssignmentHandoff, error) {
	f.mu.Lock()
	m, c := f.team[parent], f.team[child]
	if m == nil || c == nil || c.manager.Parent != parent || c.handoff.ResultCommit == "" {
		f.mu.Unlock()
		return "", AssignmentHandoff{}, fmt.Errorf("no submitted direct-child result %q for %q", child, parent)
	}
	work, handoff := c.agent.Workspace(), c.handoff
	f.mu.Unlock()
	changes, err := f.integration.Changes(ctx, work, handoff)
	if err == nil {
		f.mu.Lock()
		if c.handoff.ResultCommit == handoff.ResultCommit {
			c.inspected = handoff.ResultCommit
		}
		f.mu.Unlock()
	}
	return changes, handoff, err
}

// Return gives a concrete correction to the same child assignment.
func (f *AssignmentHandoffs) Return(ctx context.Context, parent, child, correction string, timeout time.Duration) (TurnResult, error) {
	if strings.TrimSpace(correction) == "" {
		return TurnResult{}, errors.New("a returned result needs a concrete correction")
	}
	f.mu.Lock()
	m, c := f.team[parent], f.team[child]
	if m == nil || c == nil || c.manager.Parent != parent || c.handoff.ResultCommit == "" {
		f.mu.Unlock()
		return TurnResult{}, fmt.Errorf("no submitted direct-child result %q for %q", child, parent)
	}
	c.returned = true
	f.mu.Unlock()
	return f.Message(ctx, child, correction, timeout)
}

// Accept integrates only the result the parent inspected. A conflict returns
// to the caller so the parent can assign its correction to the owning child.
func (f *AssignmentHandoffs) Accept(ctx context.Context, parent, child, inspectedCommit string) (string, error) {
	f.mu.Lock()
	m := f.team[parent]
	f.mu.Unlock()
	if m == nil {
		return "", fmt.Errorf("unknown parent %q", parent)
	}
	if err := m.acquire(ctx); err != nil {
		return "", err
	}
	defer m.release(ctx)
	f.mu.Lock()
	c := f.team[child]
	if m == nil || c == nil || c.manager.Parent != parent || c.handoff.ResultCommit == "" || c.returned || inspectedCommit != c.inspected || inspectedCommit != c.handoff.ResultCommit || m.accepted[child] == inspectedCommit || m.agent == nil {
		f.mu.Unlock()
		return "", fmt.Errorf("no inspected direct-child result %q for %q", child, parent)
	}
	pwork, cwork, handoff, previous := m.agent.Workspace(), c.agent.Workspace(), c.handoff, m.accepted[child]
	f.mu.Unlock()
	result, err := f.integration.Integrate(ctx, pwork, cwork, handoff, previous)
	if result == "" {
		if err == nil {
			err = errors.New("Git integration returned no parent result commit")
		}
		return "", err
	}
	f.mu.Lock()
	m.accepted[child] = handoff.ResultCommit
	changed := c.returned || c.handoff.ResultCommit != handoff.ResultCommit
	f.mu.Unlock()
	if changed {
		return result, errors.Join(err, fmt.Errorf("child %q changed while its Git result was integrated", child))
	}
	return result, err
}
