package run

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// TaskState is one state of a Manager task (core protocol, task machine).
type TaskState string

const (
	TaskQueued     TaskState = "queued"
	TaskExploring  TaskState = "exploring"
	TaskExplored   TaskState = "explored"
	TaskReady      TaskState = "ready"
	TaskRunning    TaskState = "running"
	TaskBlocked    TaskState = "blocked"
	TaskReviewing  TaskState = "reviewing"
	TaskDone       TaskState = "done"
	TaskIncomplete TaskState = "incomplete"
)

// Wait names what a blocked task waits for (core rule T5).
type Wait struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref,omitempty"`
}

const (
	waitReply    = "reply"
	waitDelivery = "delivery"
	waitChildren = "children"
	waitOwner    = "owner"
)

// Submission is the frozen candidate of a reviewing task (core rule T7).
// Candidate is the Host-computed S4 record of the owner's unaccepted changes
// and Tree its observed tree; both are absent for a no-change submission.
// Aggregate is the aggregate version the submitting step saw (core rule D3),
// and Uses names the producers the candidate used (core rule D4).
type Submission struct {
	Purpose, Scope string
	Proposal       BlobRef
	Candidate      BlobRef
	Tree           snapshot.TreeID
	Aggregate      uint64
	AggregateTree  snapshot.TreeID
	Uses           []string
}

type taskValue struct {
	Slot         organization.Manager
	ID           string
	State        TaskState
	Since        uint64
	Explored     bool
	ExploredAt   uint64
	Report       BlobRef
	Phase        Phase
	Session      SessionID
	Step         StepID
	Conversation ProviderSessionID
	Waits        []Wait
	Used         []string
	Own          BlobRef
	OwnTree      snapshot.TreeID
	Submission   Submission
	Review       reviewValue
	Reviews      int
	LocalDone    bool
	Deliveries   []Delivery
	Rejections   map[string]int
	Reason       string
	Failure      string
	HostFault    bool
	Withheld     string
	Kept         reviewValue
	Worked       uint64
	Claims       []Claim
}

// TaskView is the read-only projection of one Manager task.
type TaskView struct {
	Manager, Parent, ID string
	Children            []string
	State               TaskState
	Explored            bool
	Report              BlobRef
	Phase               Phase
	Session             SessionID
	Step                StepID
	Waits               []Wait
	Own                 BlobRef
	OwnTree             snapshot.TreeID
	Submission          Submission
	Expectations        BlobRef
	LocalDone           bool
	Deliveries          []Delivery
	Rejections          map[string]int
	Reason              string
	Failure             string
	Withheld            string
}

func (t taskValue) view() TaskView {
	s := t.Submission
	s.Uses = slices.Clone(s.Uses)
	return TaskView{Manager: t.Slot.Package, Parent: t.Slot.Parent, ID: t.ID, Children: slices.Clone(t.Slot.Children), State: t.State, Explored: t.Explored, Report: t.Report,
		Phase: t.Phase, Session: t.Session, Step: t.Step, Waits: slices.Clone(t.Waits), Own: t.Own, OwnTree: t.OwnTree, Submission: s, Expectations: t.Review.Expectations,
		LocalDone: t.LocalDone, Deliveries: slices.Clone(t.Deliveries), Rejections: maps.Clone(t.Rejections), Reason: t.Reason, Failure: t.Failure, Withheld: t.Withheld}
}

// Tasks returns every task of the current run, root first, then by path.
func (c *Coordinator) Tasks() []TaskView {
	var out []TaskView
	for _, slot := range c.state.managerBasis.Managers {
		if t, ok := c.state.tasks[slot.Package]; ok {
			out = append(out, t.view())
		}
	}
	return out
}

// Task returns one task by its Manager path.
func (c *Coordinator) Task(manager string) (TaskView, bool) {
	t, ok := c.state.tasks[manager]
	return t.view(), ok
}

// AggregateVersion is the version of the run's accepted aggregate (core rule
// D2); the captured source is version zero.
func (c *Coordinator) AggregateVersion() uint64 { return c.state.aggregateVersion }

// prepareTasks creates one task per Manager for a new run. Open issues do not
// carry into a continuation run yet (core rule I5).
func (s *stateValue) prepareTasks(b Binding, runID string) error {
	s.tasks = nil
	s.issues = nil
	s.issueOrder = nil
	s.messages = nil
	s.aggregateVersion, s.aggregateTree, s.sourceTree = 0, snapshot.TreeID{}, snapshot.TreeID{}
	s.checks, s.checkOrder, s.checkInvocations = nil, nil, nil
	s.verification, s.publication, s.publicationChanges = VerificationRecord{}, PublicationRecord{}, 0
	s.managerBasis = bindingWire{}
	if b.version != 3 {
		return nil
	}
	w, _, err := decodeBindingRecord(b.payload)
	if err != nil {
		return err
	}
	s.managerBasis = w
	tree, err := censusTree(w)
	if err != nil {
		return err
	}
	s.aggregateTree, s.sourceTree = tree, tree
	s.tasks = map[string]taskValue{}
	s.issues = map[string]Issue{}
	s.checks, s.checkInvocations = map[CheckID]checkValue{}, map[CheckID]string{}
	for _, m := range w.Managers {
		id := fmt.Sprintf("task-%x", sha256.Sum256([]byte(runID+":"+b.Ref().ContentID.String()+":"+m.Package)))
		s.tasks[m.Package] = taskValue{Slot: m, ID: id, State: TaskQueued, Since: s.revision}
	}
	return nil
}

// censusTree is the identity of the captured source, aggregate version zero.
func censusTree(w bindingWire) (snapshot.TreeID, error) {
	var entries []snapshot.Entry
	for _, f := range w.Census {
		path, err := snapshot.NewPath(f.Path)
		if err != nil {
			return snapshot.TreeID{}, ErrInvalid
		}
		entry, err := snapshot.NewEntry(path, f.Mode, f.Bytes)
		if err != nil {
			return snapshot.TreeID{}, ErrInvalid
		}
		entries = append(entries, entry)
	}
	tree, err := snapshot.NewTree(entries)
	if err != nil {
		return snapshot.TreeID{}, ErrInvalid
	}
	return tree.Digest(), nil
}

// Admission is the next admissible step of the run (core rule T4). Home is the
// provider home the step must use and Resume the Manager conversation it
// continues; an empty Resume starts a fresh conversation, as every review step
// does.
type Admission struct {
	Manager, Task string
	Phase         Phase
	Home          string
	Resume        ProviderSessionID
	// Writes is whether the step may change files: every work and integrate
	// step, and a reply step once the write barrier is open (core rule T11).
	Writes bool
}

func (p Phase) reviewer() bool { return p == PhaseReviewerExpectations || p == PhaseReviewerCandidate }
func (p Phase) writing() bool  { return p == PhaseWork || p == PhaseIntegrate }

// writes reports whether a task's step in the phase may change files: a reply
// step of a done or blocked task may correct the Manager's own files once
// every Manager has explored (core rules T2, T10, T11). A task blocked on the
// owner's decision about its frozen submission keeps that candidate unchanged
// until the fresh review (core rule T8).
func (s stateValue) writes(t taskValue, phase Phase) bool {
	if phase != PhaseReply {
		return phase.writing()
	}
	frozen := t.State == TaskBlocked && t.Submission.Proposal != (BlobRef{})
	return (t.State == TaskDone || t.State == TaskBlocked) && !frozen && s.barrierOpen()
}

func (s stateValue) explorable(t taskValue) bool {
	if t.State != TaskQueued {
		return false
	}
	for _, child := range t.Slot.Children {
		if !s.tasks[child].Explored {
			return false
		}
	}
	return true
}

// exploreSince is the revision at which a queued task's Explore became
// admissible: the start for a leaf, otherwise its last child's Explore result.
func (s stateValue) exploreSince(t taskValue) uint64 {
	since := t.Since
	for _, child := range t.Slot.Children {
		since = max(since, s.tasks[child].ExploredAt)
	}
	return since
}

func (s stateValue) barrierOpen() bool {
	for _, t := range s.tasks {
		if !t.Explored {
			return false
		}
	}
	return len(s.tasks) > 0
}

// barrierAt is the revision at which the write barrier opened: the last
// Explore result.
func (s stateValue) barrierAt() uint64 {
	var at uint64
	for _, t := range s.tasks {
		at = max(at, t.ExploredAt)
	}
	return at
}

func (s stateValue) anyIncomplete() bool {
	for _, t := range s.tasks {
		if t.State == TaskIncomplete {
			return true
		}
	}
	return false
}

func managerHome(t taskValue) string { return t.ID + "-manager" }

// admission returns the item a task offers to the queue, if any.
func (s stateValue) admission(t taskValue, barrier bool) (Admission, int, uint64, bool) {
	a := Admission{Manager: t.Slot.Package, Task: t.ID}
	switch {
	case t.Session != "":
		return Admission{}, 0, 0, false
	case (t.State == TaskDone || t.State == TaskBlocked || t.State == TaskExplored) && len(s.unanswered(t)) > 0:
		// A Manager that cannot otherwise act answers in a reply step and keeps
		// its state unless the step corrects its files (core rules T10, T11).
		a.Phase, a.Home, a.Resume = PhaseReply, managerHome(t), t.Conversation
		a.Writes = s.writes(t, PhaseReply)
		return a, 2, s.unanswered(t)[0].At, true
	case s.explorable(t):
		a.Phase, a.Home, a.Resume = PhaseExplore, managerHome(t), t.Conversation
		return a, 0, s.exploreSince(t), true
	case t.State == TaskReviewing && t.Review.Expectations == (BlobRef{}):
		// A review never writes, so it may run before the barrier (core rule
		// T3). Each review step starts a fresh conversation in a fresh home.
		a.Phase, a.Home = PhaseReviewerExpectations, fmt.Sprintf("%s-review-%d-expectations", t.ID, t.Reviews+1)
		return a, 1, t.Since, true
	case t.State == TaskReviewing:
		a.Phase, a.Home = PhaseReviewerCandidate, fmt.Sprintf("%s-review-%d-candidate", t.ID, t.Reviews)
		return a, 1, t.Since, true
	case barrier && t.State == TaskReady:
		a.Phase, a.Home, a.Resume, a.Writes = PhaseWork, managerHome(t), t.Conversation, true
		if t.LocalDone {
			a.Phase = PhaseIntegrate // core rule T9
		}
		return a, 2, max(t.Since, s.barrierAt()), true
	}
	return Admission{}, 0, 0, false
}

// next selects the admissible item that arose first; ties break by Manager
// path, then by kind: Explore, review, work (core rule T4). An incomplete task
// stops all admission (core rule T14).
func (s stateValue) next() (Admission, bool) {
	if s.current.State != Running || s.current.StopRequested || len(s.tasks) == 0 || s.anyIncomplete() {
		return Admission{}, false
	}
	type item struct {
		since uint64
		kind  int
		a     Admission
	}
	var items []item
	barrier := s.barrierOpen()
	for _, t := range s.tasks {
		if a, kind, since, ok := s.admission(t, barrier); ok {
			items = append(items, item{since, kind, a})
		}
	}
	if len(items) == 0 {
		return Admission{}, false
	}
	slices.SortFunc(items, func(a, b item) int {
		if a.since != b.since {
			return compareUint(a.since, b.since)
		}
		if order := strings.Compare(a.a.Manager, b.a.Manager); order != 0 {
			return order
		}
		return a.kind - b.kind
	})
	return items[0].a, true
}

func compareUint(a, b uint64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// Next returns the run's next admissible step, if any.
func (c *Coordinator) Next() (Admission, bool) {
	if c.pending != nil || c.recovering {
		return Admission{}, false
	}
	return c.state.next()
}

func (s stateValue) taskForSession(id SessionID) (taskValue, bool) {
	for _, t := range s.tasks {
		if t.Session == id && id != "" {
			return t, true
		}
	}
	return taskValue{}, false
}

func recordForTask(t taskValue, phase Phase) ManagerRecord {
	m := ManagerRecord{Manager: t.Slot.Package, Task: t.ID, Phase: phase}
	if t.Slot.Parent != "" {
		p := t.Slot.Parent
		m.Parent = &p
	}
	return m
}

// stepRole returns the session role and reservation purpose of a phase.
func stepRole(p Phase) (string, Purpose) {
	if p.reviewer() {
		return "reviewer", Assessment
	}
	return "manager", Writing
}

var emptyTree = func() snapshot.TreeID {
	tree, _ := snapshot.NewTree(nil)
	return tree.Digest()
}()

// beforeTaskSession binds session and step preparation to the admitted task
// and keeps one step at a time (core rules T4, S5).
func (s *stateValue) beforeTaskSession(c Command) error {
	if s.basisVersion != 3 {
		if c.Manager != (ManagerRecord{}) {
			return ErrInvalid
		}
		return nil
	}
	r := c.Session
	if c.Kind == PrepareSession {
		next, ok := s.next()
		t, found := s.tasks[c.Manager.Manager]
		if !ok || !found || next.Manager != c.Manager.Manager || next.Phase != c.Manager.Phase || !sameManager(recordForTask(t, next.Phase), c.Manager) {
			return ErrState
		}
		if s.hasLiveSessions() || s.hasOpen() {
			return ErrState
		}
		catalog, err := ToolCatalog(next.Phase)
		role, purpose := stepRole(next.Phase)
		if err != nil || r.Plan.Tools != catalog.Ref() || r.Plan.Role != role || r.Plan.ReadOnly == next.Writes || c.Reservation.Purpose != purpose || r.Plan.ProviderHome != next.Home || r.Plan.Resume != next.Resume {
			return ErrState
		}
		// A Manager without unaccepted changes starts from the aggregate (core
		// rule D3). The expectations step sees no candidate; the candidate step
		// sees the frozen candidate (core rule T7).
		want := emptyTree
		switch {
		case next.Phase == PhaseReviewerCandidate && t.Submission.Candidate != (BlobRef{}):
			want = t.Submission.Tree
		case next.Phase == PhaseReviewerCandidate:
			want = t.Submission.AggregateTree
		case next.Phase != PhaseReviewerExpectations && t.Own == (BlobRef{}):
			want = s.aggregateTree
		case next.Phase != PhaseReviewerExpectations:
			want = r.Plan.Candidate // own changes replayed onto the aggregate
		}
		if r.Plan.Candidate != want {
			return ErrState
		}
		t.Session = r.Session
		t.Phase = next.Phase
		switch next.Phase {
		case PhaseExplore:
			t.State = TaskExploring
		case PhaseWork, PhaseIntegrate:
			t.State = TaskRunning
		case PhaseReviewerExpectations:
			t.Reviews++
		}
		s.tasks[t.Slot.Package] = t
		return nil
	}
	t, ok := s.taskForSession(r.Session)
	if !ok {
		return ErrState
	}
	switch c.Kind {
	case PrepareStep:
		_, purpose := stepRole(t.Phase)
		if !sameManager(recordForTask(t, t.Phase), c.Manager) || t.Step != "" || c.Reservation.Purpose != purpose {
			return ErrState
		}
		input, ok := payloadFor(c, r.StepPlan.Input)
		if !ok {
			return ErrInvalid
		}
		if _, err := validateStepPacket(input, c, t, *s); err != nil {
			return err
		}
		contract, ok := payloadFor(c, r.StepPlan.OutputContract)
		if !ok {
			return ErrInvalid
		}
		if sc, err := DecodeStepContract(contract); err != nil || sc.Phase != t.Phase {
			return ErrInvalid
		}
		t.Step = r.Step
		s.tasks[t.Slot.Package] = t
	case PrepareTransport:
		if r.Control {
			if s.current.State != Reconciling {
				return ErrState
			}
			return nil
		}
		session := s.sessions[r.Session]
		if r.Step == "" {
			// The adapter's protocol handshake, one message at a time, until
			// the session is ready or bound; a Codex App Server sends three.
			if session.Ready || session.Bound {
				return ErrState
			}
			return nil
		}
		if r.Step != t.Step {
			return ErrState
		}
		for key, m := range s.transports {
			if key.Session == r.Session && m.Record.Step == r.Step && !m.Record.Control && r.Call == "" {
				// After the turn input, a send may only answer a recorded tool call.
				return ErrState
			}
		}
	case ObserveStep:
		// Outcomes arrive only through answered Host tool calls.
		if r.Step != t.Step || r.Proposed != (BlobRef{}) {
			return ErrState
		}
		if !r.Uncertain {
			sent := false
			for key, m := range s.transports {
				if key.Session == r.Session && m.Record.Step == r.Step && !m.Record.Control && m.Outcome == Sent {
					sent = true
				}
			}
			if !sent {
				return ErrState
			}
		}
	}
	return nil
}

// afterTaskSession records the Manager's bound provider conversation, which
// its next step resumes (core rule S3). Reviews never resume.
func (s *stateValue) afterTaskSession(c Command) error {
	if s.basisVersion != 3 || c.Kind != BindSession || c.Session.Uncertain || c.Session.ProviderSession == "" {
		return nil
	}
	t, ok := s.taskForSession(c.Session.Session)
	if !ok || t.Phase.reviewer() {
		return nil
	}
	t.Conversation = c.Session.ProviderSession
	s.tasks[t.Slot.Package] = t
	return nil
}

// TaskConclude records the end of one task step after its container stopped
// (core rules T3, T5, T6, T8, S4). The outcome is the step's answered outcome
// tool call, unless the Host's own check failed the step.
const TaskConclude = "task.step-concluded"

// ConclusionRecord is the Host's account of one ended step. Input is the
// outcome tool call's recorded input; Candidate and Tree are the S4 result of
// a writing step that changed files. Aggregate is the aggregate version after
// the step: the version the step saw, or the next one when the step's review
// accepted a changed candidate (core rule D2).
type ConclusionRecord struct {
	Manager   string
	Task      string
	Step      StepID
	Outcome   string
	Input     BlobRef
	Candidate BlobRef
	Tree      snapshot.TreeID
	Aggregate uint64
	// AggregateTree is the composed aggregate's tree identity when the step's
	// review accepted a changed candidate.
	AggregateTree snapshot.TreeID
	Reason        string
	// HostFault marks a failed step that the Host, not the Agent, could not
	// finish.
	HostFault bool
	// Changed marks a step that changed its workspace, whether or not the Host
	// accepted the change (core rule T11).
	Changed bool
}

const outcomeFailed = "failed"

func (r ConclusionRecord) payloadRefs() []BlobRef {
	var refs []BlobRef
	for _, ref := range []BlobRef{r.Input, r.Candidate} {
		if ref != (BlobRef{}) {
			refs = append(refs, ref)
		}
	}
	return refs
}

func (r ConclusionRecord) validate() error {
	// A session the Host abandoned before its step concludes without a step.
	if !text(r.Manager) || !identifier(r.Task) || !identifier(string(r.Step)) && (r.Step != "" || r.Outcome != outcomeFailed) || !text(r.Outcome) || r.Reason != "" && !boundedInput(r.Reason) {
		return ErrInvalid
	}
	if (r.Outcome == outcomeFailed) != (r.Input == (BlobRef{})) || r.Outcome == outcomeFailed && (r.Reason == "" || r.Candidate != (BlobRef{})) || r.HostFault && r.Outcome != outcomeFailed {
		return ErrInvalid
	}
	if (r.Candidate == (BlobRef{})) != (r.Tree == (snapshot.TreeID{})) {
		return ErrInvalid
	}
	for _, ref := range r.payloadRefs() {
		if !ref.Valid() {
			return ErrInvalid
		}
	}
	return nil
}

// stepOutcomeCall returns the step's one answered outcome tool call.
func (s stateValue) stepOutcomeCall(t taskValue) (ToolView, int) {
	var found ToolView
	count := 0
	for key, tool := range s.tools {
		if key.Session == t.Session && tool.Record.Step == t.Step && tool.Replied.BatchID != "" && tool.OK && OutcomeTool(t.Phase, tool.Record.Tool) {
			found = tool
			count++
		}
	}
	return found, count
}

type submitInput struct {
	Purpose      string            `json:"purpose"`
	Scope        string            `json:"scope"`
	Summary      string            `json:"summary"`
	Assessments  []json.RawMessage `json:"assessments"`
	Dependencies []string          `json:"dependencies"`
}

type reportInput struct {
	Summary         string   `json:"summary"`
	ProposedChanges []string `json:"proposedChanges"`
	Prerequisites   []string `json:"prerequisites"`
	Uncertainties   []string `json:"uncertainties"`
	Questions       []string `json:"questions"`
}

type waitInput struct {
	On     []Wait `json:"on"`
	Reason string `json:"reason"`
}

type expectationsInput struct {
	Cases []struct {
		Name          string `json:"name"`
		Expectation   string `json:"expectation"`
		Discriminates string `json:"discriminates"`
	} `json:"cases"`
}

// CheckOutcome checks an outcome tool call of the manager's current step
// against its phase and the run state. The Host answers the call only when it
// passes, and the step's conclusion applies the same check, so a refused call
// leaves the Agent its step to correct it.
func (c *Coordinator) CheckOutcome(manager, tool string, args []byte) error {
	t, ok := c.state.tasks[manager]
	if !ok || t.Session == "" {
		return errors.New("this task has no running step")
	}
	return c.state.checkOutcome(t, tool, args)
}

func (s stateValue) checkOutcome(t taskValue, tool string, args []byte) error {
	if !OutcomeTool(t.Phase, tool) {
		return fmt.Errorf("%s does not end a %s step", tool, t.Phase)
	}
	switch tool {
	case ToolReport:
		var in reportInput
		if err := decodeArgs(args, &in); err != nil {
			return err
		}
		if !text(in.Summary) || in.ProposedChanges == nil || in.Prerequisites == nil || in.Uncertainties == nil || in.Questions == nil {
			return errors.New("report needs a summary and all four lists")
		}
	case ToolSubmit:
		var in submitInput
		if err := decodeArgs(args, &in); err != nil {
			return err
		}
		if in.Purpose != "local" && in.Purpose != "integration" || in.Scope != "limited" && in.Scope != "complete" || !text(in.Summary) || in.Assessments == nil || in.Dependencies == nil {
			return errors.New("submit needs purpose local or integration, scope limited or complete, a summary, assessments and dependencies")
		}
		if t.Phase == PhaseExplore && (in.Purpose != "local" || in.Scope != "complete") {
			return errors.New("during Explore only a complete local no-change result can be submitted")
		}
		if t.Phase == PhaseWork && in.Purpose != "local" || t.Phase == PhaseIntegrate && in.Purpose != "integration" {
			return errors.New("the submission's purpose does not match this step: work submits local, integration submits integration")
		}
		if in.Purpose == "integration" {
			// An integration composes done children only (core rule T9).
			for _, child := range t.Slot.Children {
				if c := s.tasks[child]; c.State != TaskDone {
					return fmt.Errorf("child %s is %s, not done: wait for your children before you integrate", child, c.State)
				}
			}
		}
		for _, d := range in.Dependencies {
			if _, ok := s.tasks[d]; !ok || d == t.Slot.Package {
				return fmt.Errorf("dependency %q is not another Manager of this run", d)
			}
		}
	case ToolWait:
		var in waitInput
		if err := decodeArgs(args, &in); err != nil {
			return err
		}
		if len(in.On) == 0 || !text(in.Reason) {
			return errors.New("wait needs at least one wait and a reason")
		}
		// A wait ends on any arrival, so it is refused only when none of its
		// items can ever arrive (core rule T10).
		var stuck []string
		for _, w := range in.On {
			if err := s.checkWait(t, w); err != nil {
				return err
			}
			if reason := s.stuckWait(t, w); reason != "" {
				stuck = append(stuck, reason)
			}
		}
		if len(stuck) == len(in.On) {
			return fmt.Errorf("this wait would never end: %s; deliver what they wait for first, such as your own local work, and let the integrating parent compose the parts", strings.Join(stuck, "; "))
		}
	case ToolRecordExpectations:
		var in expectationsInput
		if err := decodeArgs(args, &in); err != nil {
			return err
		}
		if len(in.Cases) == 0 {
			return errors.New("record_expectations needs at least one case")
		}
		names := map[string]bool{}
		for _, c := range in.Cases {
			if !text(c.Name) || !text(c.Expectation) || !text(c.Discriminates) {
				return errors.New("each case needs a name, an expectation and what it discriminates")
			}
			if names[c.Name] {
				return fmt.Errorf("case name %q is used twice; the candidate review assesses each case by its name", c.Name)
			}
			names[c.Name] = true
		}
	case ToolSubmitReview:
		var in reviewInput
		if err := decodeArgs(args, &in); err != nil {
			return err
		}
		if err := s.checkReview(t, in); err != nil {
			return err
		}
	case ToolSend:
		// A reply step ends with its reply, which CheckSend checked.
		var in sendInput
		if err := decodeArgs(args, &in); err != nil || in.Kind != messageReply {
			return errors.New("a reply step ends with a reply")
		}
	}
	return nil
}

// checkWait accepts only waits that something in this version can satisfy
// (core rule T10). Replies to Manager questions arrive with messages.
func (s stateValue) checkWait(t taskValue, w Wait) error {
	switch w.Kind {
	case waitDelivery:
		producer, ok := s.tasks[w.Ref]
		if !ok || w.Ref == t.Slot.Package {
			return fmt.Errorf("delivery wait %q does not name another Manager of this run", w.Ref)
		}
		if producer.State == TaskDone {
			return fmt.Errorf("%s is done; its accepted result is already in your workspace", w.Ref)
		}
	case waitChildren:
		if len(t.Slot.Children) == 0 || s.childrenDone(t) || w.Ref != "" {
			return errors.New("a children wait needs children that are not all done, and no ref")
		}
	case waitReply, waitOwner:
		to, asked := s.pendingQuestions(t)[w.Ref]
		if q, ok := s.issues[w.Ref]; !asked && ok && q.Kind == issueQuestion && q.Task == t.Slot.Package && q.State == issueOpen {
			to, asked = q.To, true
		}
		if !asked {
			return fmt.Errorf("%q is not an open question you asked", w.Ref)
		}
		if w.Kind == waitOwner && to != addressedToOwner {
			return fmt.Errorf("%q is not a question to the owner", w.Ref)
		}
	default:
		return fmt.Errorf("unknown wait kind %q", w.Kind)
	}
	return nil
}

func (s *stateValue) concludeStep(c Command) error {
	r := c.Conclusion
	t, ok := s.tasks[r.Manager]
	if !ok || t.ID != r.Task || t.Step != r.Step || t.Session == "" {
		return ErrState
	}
	// The container stopped and every effect of the session ended: the step
	// ran to its end, or was never dispatched and released (core rule S2).
	session, live := s.sessions[t.Session]
	if !live || !session.Terminated || session.Uncertain || s.toolOpen(t.Session) || s.transportOpen(t.Session) || open(s.invocations[session.Invocation]) {
		return ErrState
	}
	if step := s.steps[t.Step]; t.Step != "" && (!step.Terminal || step.Uncertain || open(s.invocations[step.Invocation])) {
		return ErrState
	}
	refs := r.payloadRefs()
	if len(c.sessionPayloads) != len(refs) {
		return ErrInvalid
	}
	for _, ref := range refs {
		if _, ok := payloadFor(c, ref); !ok {
			return ErrInvalid
		}
	}
	call, count := s.stepOutcomeCall(t)
	if count > 1 {
		return ErrState
	}
	// The step's messages take effect with its conclusion, before its outcome
	// that may wait on them; a failed step's messages are dropped with it
	// (core rule T6), except a reply step's (core rule T11). A reply step the
	// Host could not finish is retried.
	if t.Phase == PhaseReply && !r.HostFault {
		s.replyStepTaken(t, r.Step)
	}
	if r.Outcome != outcomeFailed || t.Phase == PhaseReply && !r.HostFault {
		if err := s.recordSends(t, s.revision); err != nil {
			return err
		}
	}
	writes := s.writes(t, t.Phase)
	if r.Outcome != outcomeFailed && writes {
		t.Claims = addClaims(t.Claims, s.stepClaims(t)) // core rule S4
	}
	// A failed conclusion may override an answered outcome when the Host's own
	// check of the step failed (core rules S4, T6).
	var args []byte
	if r.Outcome != outcomeFailed {
		input, _ := payloadFor(c, r.Input)
		var err error
		args, err = CallArguments(r.Outcome, input.Bytes())
		if count != 1 || r.Outcome != call.Record.Tool || r.Input != call.Record.Wire.Journal || err != nil || s.checkOutcome(t, r.Outcome, args) != nil {
			return ErrState
		}
	}
	if !writes && r.Candidate != (BlobRef{}) || t.Phase == PhaseReply && r.Candidate != (BlobRef{}) && !r.Changed {
		return ErrState
	}
	revision := s.revision
	t.Session, t.Step = "", ""
	var err error
	if t.Phase.reviewer() {
		err = s.concludeReview(r, &t, revision, args)
	} else {
		err = s.concludeWork(r, &t, revision, args)
	}
	if err != nil {
		return err
	}
	s.tasks[t.Slot.Package] = t
	s.advanceTasks(revision)
	return nil
}

// concludeWork applies an Explore, work or integration step's outcome (core
// rules T2, T3, T5, T6).
func (s *stateValue) concludeWork(r ConclusionRecord, t *taskValue, revision uint64, args []byte) error {
	if r.Aggregate != s.aggregateVersion || r.AggregateTree != (snapshot.TreeID{}) {
		return ErrState
	}
	if r.Candidate != (BlobRef{}) {
		t.Own, t.OwnTree = r.Candidate, r.Tree
	}
	if t.Phase == PhaseReply {
		if r.Changed && !r.HostFault && s.writes(*t, PhaseReply) {
			// A reply step that changed the Manager's files returns its task to
			// work, so that the change goes through review, and a done task's
			// ancestors wait for it again (core rule T11). A change the Host
			// refused reaches the work step as its failure.
			if t.State == TaskDone {
				t.LocalDone = false
				s.blockAncestors(t.Slot.Package, "", revision)
			}
			t.State, t.Since, t.Waits = TaskReady, revision, nil
			if r.Outcome == outcomeFailed {
				t.Failure, t.HostFault = r.Reason, false
			}
		}
		return nil
	}
	t.Failure, t.HostFault, t.Worked = "", false, revision
	if r.Outcome != outcomeFailed {
		// A failed step leaves no new submission, so a withheld pass's note
		// stays for the next step.
		t.Withheld = ""
	}
	switch r.Outcome {
	case outcomeFailed:
		t.State, t.Since, t.Failure, t.HostFault = TaskReady, revision, r.Reason, r.HostFault
		if t.Phase == PhaseExplore {
			t.State = TaskQueued
		}
	case ToolReport:
		t.State, t.Explored, t.ExploredAt, t.Report = TaskExplored, true, revision, r.Input
	case ToolSubmit:
		var in submitInput
		if json.Unmarshal(args, &in) != nil {
			return ErrInvalid
		}
		if t.Phase == PhaseExplore {
			// The parent's Explore reads it as the child's report (core rules
			// T1, T2), whatever the review decides.
			t.Explored, t.ExploredAt, t.Report = true, revision, r.Input
		}
		uses := slices.Clone(t.Used)
		for _, d := range in.Dependencies {
			if !slices.Contains(uses, d) {
				uses = append(uses, d)
			}
		}
		slices.Sort(uses)
		t.State, t.Since, t.Review = TaskReviewing, revision, reviewValue{}
		t.Submission = Submission{Purpose: in.Purpose, Scope: in.Scope, Proposal: r.Input, Candidate: t.Own, Tree: t.OwnTree, Aggregate: s.aggregateVersion, AggregateTree: s.aggregateTree, Uses: uses}
		if k := t.Kept; k.Inputs != "" && k.Inputs == s.expectationInputs(*t, k.Assessed) {
			// A withheld or rejected candidate leaves its recorded expectations
			// standing while their inputs are unchanged, apart from findings
			// the reviews against them raised or resolved; the expectations
			// step sees neither the candidate nor the aggregate. The next review
			// assesses the same cases in a fresh candidate home (core rule T7,
			// DEC-046, DEC-047).
			t.Review = k
			t.Reviews++
		}
		t.Kept = reviewValue{}
	case ToolWait:
		var in waitInput
		if json.Unmarshal(args, &in) != nil {
			return ErrInvalid
		}
		t.State, t.Since, t.Waits = TaskBlocked, revision, in.On
		for _, w := range in.On {
			if w.Kind == waitDelivery && !slices.Contains(t.Used, w.Ref) {
				t.Used = append(slices.Clone(t.Used), w.Ref)
			}
		}
	default:
		return ErrState
	}
	return nil
}

// advanceTasks applies the rules that follow any conclusion: the write barrier
// opens once every task is explored (core rule T2), blocked tasks wake when
// what they named arrived (core rule T10), and the run moves to Verifying when
// the root is done (core rule T13).
func (s *stateValue) advanceTasks(revision uint64) {
	if s.barrierOpen() {
		for name, t := range s.tasks {
			if t.State == TaskExplored {
				t.State, t.Since = TaskReady, revision
				s.tasks[name] = t
			}
		}
	}
	for name, t := range s.tasks {
		if t.State == TaskBlocked && (slices.ContainsFunc(t.Waits, func(w Wait) bool { return s.waitArrived(t, w) }) || s.messaged(t)) {
			t.State, t.Since, t.Waits = TaskReady, revision, nil
			s.tasks[name] = t
		}
	}
	if root, ok := s.tasks[organization.Root]; ok && root.State == TaskDone && s.current.State == Running {
		s.current.State = Verifying
		s.runs[s.current.ID] = s.current
	}
}

// messaged reports whether a blocked task has a question that none of its
// Explore, work or integrate steps has seen. The message makes the task ready,
// so that its next step answers it and can act on it (core rule T10). A task
// awaiting the owner's decision on its frozen submission stays blocked and
// answers in a read-only reply step (core rule T8), and so does a task whose
// last step saw the question but left it unanswered.
func (s stateValue) messaged(t taskValue) bool {
	if t.Submission.Proposal != (BlobRef{}) {
		return false
	}
	return slices.ContainsFunc(s.unanswered(t), func(q Issue) bool { return q.At > t.Worked })
}

// stuckWait says why one wait item can never arrive: the Manager it needs waits
// for the asker.
func (s stateValue) stuckWait(t taskValue, w Wait) string {
	switch w.Kind {
	case waitDelivery:
		if s.waitsFor(w.Ref, t.Slot.Package, map[string]int{}) {
			return fmt.Sprintf("%s waits for you, so its delivery cannot come first", w.Ref)
		}
		// An ancestor whose local work is accepted delivers next its integration,
		// which waits for every descendant (core rule T9).
		if s.ancestor(w.Ref, t.Slot.Package) && s.tasks[w.Ref].LocalDone {
			return fmt.Sprintf("%s has delivered its local work; its next delivery is its integration, which waits for you", w.Ref)
		}
	case waitChildren:
		for _, child := range t.Slot.Children {
			if s.tasks[child].State != TaskDone && s.waitsFor(child, t.Slot.Package, map[string]int{}) {
				return fmt.Sprintf("your child %s waits for you, so your children cannot all finish first", child)
			}
		}
	}
	return ""
}

// ancestor reports whether manager is an ancestor of other.
func (s stateValue) ancestor(manager, other string) bool {
	for p := s.tasks[other].Slot.Parent; p != ""; p = s.tasks[p].Slot.Parent {
		if p == manager {
			return true
		}
	}
	return false
}

// waitsFor reports whether a blocked Manager can act again only after another
// Manager acts: each of its waits needs that Manager, directly or through other
// blocked Managers. memo holds each Manager's result, or pending while it is
// being decided.
func (s stateValue) waitsFor(manager, other string, memo map[string]int) bool {
	const pending, yes, no = 1, 2, 3
	switch memo[manager] {
	case pending, no:
		return false
	case yes:
		return true
	}
	t, ok := s.tasks[manager]
	if !ok || t.State != TaskBlocked || len(t.Waits) == 0 {
		memo[manager] = no
		return false
	}
	memo[manager] = pending
	for _, w := range t.Waits {
		stuck := false
		switch w.Kind {
		case waitDelivery:
			stuck = w.Ref == other || s.waitsFor(w.Ref, other, memo)
		case waitChildren:
			for _, child := range t.Slot.Children {
				if s.tasks[child].State != TaskDone && (child == other || s.waitsFor(child, other, memo)) {
					stuck = true
				}
			}
		}
		if !stuck {
			memo[manager] = no
			return false
		}
	}
	memo[manager] = yes
	return true
}

func (s stateValue) waitArrived(t taskValue, w Wait) bool {
	switch w.Kind {
	case waitDelivery:
		return slices.ContainsFunc(s.tasks[w.Ref].Deliveries, func(d Delivery) bool { return d.At > t.Since })
	case waitChildren:
		return s.childrenDone(t)
	case waitReply, waitOwner:
		return s.issues[w.Ref].State != issueOpen
	}
	return false
}

func (s stateValue) childrenDone(t taskValue) bool {
	for _, child := range t.Slot.Children {
		if s.tasks[child].State != TaskDone {
			return false
		}
	}
	return true
}

// OutcomeInput returns the recorded input of the current step's answered
// outcome tool call.
func (c *Coordinator) OutcomeInput(manager string) (BlobRef, bool) {
	t, ok := c.state.tasks[manager]
	if !ok || t.Step == "" {
		return BlobRef{}, false
	}
	call, count := c.state.stepOutcomeCall(t)
	return call.Record.Wire.Journal, count == 1
}

// freshTaskAdmission requires commands that admit or conclude task steps to be
// built by the Coordinator's own methods. Historical replay sets the proof.
func (c *Coordinator) freshTaskAdmission(cmd Command, digest snapshot.ContentID) error {
	if cmd.Kind == PrepareRun {
		if cmd.binding.version != 3 {
			return ErrState
		}
		return nil
	}
	switch cmd.Kind {
	case ReserveInvocation:
		return ErrState // No unscoped executable consumer exists.
	case PrepareSession, PrepareStep, TaskConclude, PrepareCheck:
		if c.state.basisVersion != 3 || (cmd.Kind == PrepareSession || cmd.Kind == PrepareStep) && cmd.Manager == (ManagerRecord{}) || cmd.managerProof != digest {
			return ErrState
		}
	case PrepareTransport:
		if c.state.basisVersion != 3 && (!cmd.Session.Control || c.Current().State != Reconciling) {
			return ErrState
		}
	}
	return nil
}

// CallArguments returns the arguments of a recorded tool call input. The input
// is the call as the adapter observed it, a JSON object that names the tool
// and carries its arguments, as the agentd frame does.
func CallArguments(tool string, input []byte) ([]byte, error) {
	var frame struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(input, &frame); err != nil || frame.Name != tool || len(frame.Arguments) == 0 {
		return nil, errors.New("the call names another tool or carries no arguments")
	}
	return frame.Arguments, nil
}

// decodeArgs decodes tool arguments strictly: unknown fields and trailing
// content are refused.
func decodeArgs(data []byte, out any) error {
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return fmt.Errorf("invalid arguments: %v", err)
	}
	if d.More() {
		return errors.New("invalid arguments: trailing content")
	}
	return nil
}
