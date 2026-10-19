package run

import (
	"crypto/sha256"
	"errors"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These are direct pure accounting/reducer units, not reachable V3 execution
// histories or Coordinator permits. Reviewer/check/assessment execution is not
// implemented. Their arithmetic and uncertainty invariants remain independently
// exercised rather than bypassing public admission with a testing switch.
func unitState(t *testing.T) *stateValue {
	t.Helper()
	c, err := NewCoordinator("unit")
	if err != nil {
		t.Fatal(err)
	}
	s := &c.state
	s.assignment = "a"
	s.current = RunView{ID: "r", Assignment: "a", State: Running}
	s.runs["r"] = s.current
	s.policy = BudgetPolicy{Ceilings: Capacity{Runs: 8, SessionStarts: 20, Steps: 40, ExecutionTime: 20 * time.Minute}, Assignment: Capacity{Runs: 3, SessionStarts: 8, Steps: 10, ExecutionTime: 10 * time.Minute}, Reserve: Capacity{SessionStarts: 4, Steps: 4, ExecutionTime: 2 * time.Minute}, MaxStepTimeout: 2 * time.Minute, HostOperationTimeout: time.Minute}
	s.initialPolicy = s.policy
	return s
}
func unitCommand(kind string) Command {
	return Command{ID: "unit", Kind: kind, Assignment: "a", Run: "r", Reason: "explicit unit observation", NextActor: "owner"}
}
func unitSettle(t *testing.T, s *stateValue, id string, elapsed time.Duration) {
	t.Helper()
	c := unitCommand(SettleInvocation)
	c.Invocation = id
	c.Elapsed = elapsed
	if err := s.apply(c); err != nil {
		t.Fatal(err)
	}
}
func unitReserve(t *testing.T, s *stateValue, id string, kind InvocationKind, purpose Purpose, max time.Duration) {
	t.Helper()
	if err := s.reserve(Reservation{id, kind, purpose, max}); err != nil {
		t.Fatal(err)
	}
}
func unitBudget(s *stateValue) BudgetView { return budget(s.policy, int64(len(s.runs)), s.invocations) }
func TestWritingProtectsReserveAndAssessmentCanSpendIt(t *testing.T) {
	t.Parallel()
	s := unitState(t)
	for n := 0; n < 6; n++ {
		id := string(rune('a' + n))
		unitReserve(t, s, id, AgentStep, Writing, time.Second)
		unitSettle(t, s, id, 0)
	}
	if unitBudget(s).Remaining.Steps != 4 {
		t.Fatal(unitBudget(s))
	}
	if err := s.reserve(Reservation{"writer", AgentStep, Writing, time.Second}); !errors.Is(err, ErrBudget) {
		t.Fatal(err)
	}
	// An expectations step that prepares writing protects the reserve too
	// (DEC-049).
	if err := s.reserve(Reservation{"prepare", AgentStep, Preparation, time.Second}); !errors.Is(err, ErrBudget) {
		t.Fatal(err)
	}
	unitReserve(t, s, "review", AgentStep, Assessment, time.Minute)
	if unitBudget(s).Held.Steps != 1 || unitBudget(s).Consumed.Steps != 6 {
		t.Fatal(unitBudget(s))
	}
	unitSettle(t, s, "review", time.Second)
	if unitBudget(s).Remaining.Steps != 3 || unitBudget(s).Consumed.ExecutionNanoseconds != "1000000000" {
		t.Fatal(unitBudget(s))
	}
	if err := s.reserve(Reservation{"long", AgentStep, Assessment, 3 * time.Minute}); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if err := s.reserve(Reservation{"review", AgentStep, Assessment, time.Second}); !errors.Is(err, ErrConflict) {
		t.Fatal("duplicate invocation", err)
	}
}
func TestActualOverrunIsPreservedAndBlocksAllAdmission(t *testing.T) {
	t.Parallel()
	s := unitState(t)
	unitReserve(t, s, "step", AgentStep, Assessment, time.Second)
	unitSettle(t, s, "step", 11*time.Minute)
	b := unitBudget(s)
	if !b.Overrun || b.Remaining != (Capacity{}) || b.Consumed.Steps != 1 || b.Consumed.ExecutionNanoseconds != "660000000000" {
		t.Fatal(b)
	}
	if err := s.reserve(Reservation{"new", SessionStart, Assessment, time.Second}); !errors.Is(err, ErrBudget) {
		t.Fatal(err)
	}
}
func TestUnknownEffectCannotBeFreedByFailureStopOrWaiting(t *testing.T) {
	t.Parallel()
	s := unitState(t)
	unitReserve(t, s, "step", AgentStep, Writing, time.Minute)
	u := unitCommand(ObserveUncertain)
	u.Invocation = "step"
	if err := s.apply(u); err != nil {
		t.Fatal(err)
	}
	before := unitBudget(s)
	if s.current.State != Reconciling || before.Held.Steps != 1 || before.Held.ExecutionTime != time.Minute {
		t.Fatal(before)
	}
	for _, o := range []Outcome{Failed, Stopped, Incomplete, Stale} {
		f := unitCommand(FinishRun)
		f.Outcome = o
		if err := s.apply(f); !errors.Is(err, ErrState) {
			t.Fatal(o, err)
		}
	}
	if err := s.apply(unitCommand(WaitRun)); !errors.Is(err, ErrState) {
		t.Fatal("unknown effect waited", err)
	}
	if err := s.apply(unitCommand(StopRun)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, unitBudget(s)) {
		t.Fatal("stop freed hold")
	}
	unitSettle(t, s, "step", time.Second)
	f := unitCommand(FinishRun)
	f.Outcome = Stopped
	if err := s.apply(f); err != nil {
		t.Fatal(err)
	}
	if s.current.Outcome != Stopped || s.current.State != Finished {
		t.Fatal(s.current)
	}
}
func TestProvenNonDispatchAndWaitingDoNotCharge(t *testing.T) {
	t.Parallel()
	s := unitState(t)
	before := unitBudget(s)
	unitReserve(t, s, "start", SessionStart, Writing, time.Minute)
	u := unitCommand(ObserveUncertain)
	u.Invocation = "start"
	if err := s.apply(u); err != nil {
		t.Fatal(err)
	}
	r := unitCommand(ReleaseUndispatched)
	r.Invocation = "start"
	if err := s.apply(r); err != nil {
		t.Fatal(err)
	}
	if err := s.apply(unitCommand(ResumeRun)); err != nil {
		t.Fatal(err)
	}
	if err := s.apply(unitCommand(WaitRun)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, unitBudget(s)) || s.current.State != Waiting {
		t.Fatal(unitBudget(s))
	}
	later := unitCommand(ResumeRun)
	later.ObservedAt = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := s.apply(later); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, unitBudget(s)) {
		t.Fatal("timestamps charged")
	}
}
func TestContinuationRequiresRemainingStructuralReviewFloor(t *testing.T) {
	t.Parallel()
	demand := reviewDemand{6, 12, 12}
	base := BudgetView{Remaining: Capacity{Runs: 1, SessionStarts: 12, Steps: 12, ExecutionTime: time.Second}}
	if !reviewCapacity(base, demand) {
		t.Fatal("exact demand refused")
	}
	for _, kind := range []string{"starts", "steps", "time", "runs", "overrun"} {
		b := base
		switch kind {
		case "starts":
			b.Remaining.SessionStarts--
		case "steps":
			b.Remaining.Steps--
		case "time":
			b.Remaining.ExecutionTime = 0
		case "runs":
			b.Remaining.Runs = 0
		case "overrun":
			b.Overrun = true
		}
		if reviewCapacity(b, demand) {
			t.Fatal(kind)
		}
	}
}
func TestContinuationCanUseStructuralFloorBelowConfiguredReserve(t *testing.T) {
	t.Parallel()
	s := unitState(t)
	s.policy.Reserve = Capacity{SessionStarts: 6, Steps: 6, ExecutionTime: 5 * time.Minute}
	s.policy.Assignment = Capacity{Runs: 3, SessionStarts: 8, Steps: 10, ExecutionTime: 10 * time.Minute}
	for n := 0; n < 4; n++ {
		id := string(rune('a' + n))
		unitReserve(t, s, id, SessionStart, Assessment, time.Second)
		unitSettle(t, s, id, 0)
	}
	for n := 0; n < 6; n++ {
		id := string(rune('k' + n))
		unitReserve(t, s, id, AgentStep, Assessment, time.Second)
		unitSettle(t, s, id, 0)
	}
	unitReserve(t, s, "check", ProjectCheck, Assessment, 6*time.Minute)
	unitSettle(t, s, "check", 6*time.Minute)
	view := unitBudget(s)
	if view.Remaining.SessionStarts != 4 || view.Remaining.Steps != 4 || view.Remaining.ExecutionTime != 4*time.Minute || !reviewCapacity(view, reviewDemand{4, 4, 4}) {
		t.Fatal(view)
	}
	if err := s.reserve(Reservation{"write", AgentStep, Writing, time.Second}); !errors.Is(err, ErrBudget) {
		t.Fatal(err)
	}
	unitReserve(t, s, "review", AgentStep, Assessment, time.Second)
}
func TestZeroReviewContinuationHasZeroStructuralFloor(t *testing.T) {
	t.Parallel()
	w := bindingWire{Version: 2, Managers: []organization.Manager{{Package: organization.Root}}}
	old := recordedReviewDemand(w)
	if old != (reviewDemand{}) {
		t.Fatal(old)
	}
	if !reviewCapacity(BudgetView{Remaining: Capacity{Runs: 1}}, old) {
		t.Fatal("historical zero floor changed")
	}
	w.Version = 3
	if recordedReviewDemand(w) != (reviewDemand{1, 2, 2}) {
		t.Fatal("ordinary root was skipped")
	}
}
func TestAssignmentExtensionPreservesConsumptionHoldsAndCapRevisions(t *testing.T) {
	t.Parallel()
	s := unitState(t)
	unitReserve(t, s, "spent", AgentStep, Assessment, time.Minute)
	unitSettle(t, s, "spent", 20*time.Second)
	unitReserve(t, s, "held", SessionStart, Assessment, time.Minute)
	before := unitBudget(s)
	c := unitCommand(ExtendAssignment)
	c.OwnerInput = "exact owner extension\r\n"
	c.Caps = s.policy.Assignment
	c.Caps.Runs++
	c.Caps.SessionStarts++
	c.Caps.Steps++
	c.Caps.ExecutionTime += time.Minute
	if err := s.apply(c); err != nil {
		t.Fatal(err)
	}
	after := unitBudget(s)
	if before.Consumed != after.Consumed || before.Held != after.Held || after.Remaining.ExecutionTime != before.Remaining.ExecutionTime+time.Minute || s.capRevision() != 2 || s.extensions[0].OwnerInput != c.OwnerInput {
		t.Fatal(before, after, s.extensions)
	}
	unitReserve(t, s, "new", AgentStep, Assessment, time.Minute)
	if s.invocations["new"].CapRevision != 2 || s.invocations["held"].CapRevision != 1 {
		t.Fatal(s.invocations)
	}
}
func TestExtensionDoesNotClearObservedOverrun(t *testing.T) {
	t.Parallel()
	s := unitState(t)
	unitReserve(t, s, "overrun", AgentStep, Assessment, time.Second)
	unitSettle(t, s, "overrun", 2*time.Second)
	before := unitBudget(s)
	c := unitCommand(ExtendAssignment)
	c.OwnerInput = "extend"
	c.Caps = s.policy.Ceilings
	if err := s.apply(c); err != nil {
		t.Fatal(err)
	}
	after := unitBudget(s)
	if !after.Overrun || after.Remaining != (Capacity{}) || after.Consumed != before.Consumed {
		t.Fatal(after)
	}
}
func TestReviewerSessionContractRequiresROAssessmentAndSettledReadiness(t *testing.T) {
	t.Parallel()
	s := unitState(t)
	profile, _ := NewPayload("application/json", "utf-8", []byte("profile"))
	tools, _ := NewPayload("application/json", "utf-8", []byte("tools"))
	plan := SessionPlan{ID: "reviewer", Role: "reviewer", Workspace: "unit-reviewer", ProviderHome: "provider-home", ReadOnly: true, ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", CredentialRef: "credential", MaxWireBytes: 4096, Profile: profile.Ref(), Tools: tools.Ref()}
	plan.Candidate[0] = 1
	prepare := PrepareSessionCommand(RecordMeta{}, "a", "r", plan, Reservation{"open", SessionStart, Assessment, time.Minute}, profile, tools)
	for name, edit := range map[string]func(*SessionPlan){
		"writable reviewer":          func(p *SessionPlan) { p.ReadOnly = false },
		"resumed reviewer":           func(p *SessionPlan) { p.Resume = "provider-session" },
		"missing provider home":      func(p *SessionPlan) { p.ProviderHome = "" },
		"provider home is workspace": func(p *SessionPlan) { p.ProviderHome = p.Workspace },
	} {
		bad := prepare
		edit(&bad.Session.Plan)
		if err := bad.Session.validate(PrepareSession); !errors.Is(err, ErrInvalid) {
			t.Fatal(name, err)
		}
	}
	bad := prepare
	bad.Reservation.Purpose = Writing
	if err := s.applySession(bad); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if err := s.applySession(prepare); err != nil {
		t.Fatal(err)
	}
	input, _ := NewPayload("text/plain", "utf-8", []byte("expectations before candidate"))
	contract, _ := NewPayload("text/plain", "utf-8", []byte("expectations"))
	step := PrepareStepCommand(RecordMeta{}, "a", "r", StepPlan{Session: plan.ID, ID: "expectations", SessionBinding: plan.Binding(), Input: input.Ref(), OutputContract: contract.Ref()}, Reservation{"expectations", AgentStep, Assessment, time.Minute}, input, contract)
	if err := s.applySession(step); !errors.Is(err, ErrState) {
		t.Fatal("step before ready", err)
	}
	ready := unitCommand(ReadySession)
	ready.Session = SessionRecord{Session: plan.ID, Wire: WireRecord{Identity: WireIdentity{ContentID: input.Ref().ContentID, Size: input.Ref().Size}, Journal: input.Ref()}, Reason: "unit ready"}
	ready.sessionPayloads = []Payload{input}
	if err := s.applySession(ready); err != nil {
		t.Fatal(err)
	}
	s.sessionReceipt(ready, AppendReceipt{BatchID: "unit-marker"}) // Structural unit marker, not a committed receipt.
	if err := s.applySession(step); !errors.Is(err, ErrState) {
		t.Fatal("startup not settled", err)
	}
	unitSettle(t, s, "open", time.Second)
	if err := s.applySession(step); err != nil {
		t.Fatal(err)
	}
	if s.invocations["expectations"].Reservation.Purpose != Assessment {
		t.Fatal("review cost changed")
	}
}

func TestManagerPolicyRejectsForgedOperationAndLegacyNewFields(t *testing.T) {
	t.Parallel()
	doc, _ := NewPayload("text/plain", "utf-8", []byte("original producer document"))
	source := organization.SettingSource{File: "execution", Field: "/limits/maxContextBytes", ContentID: doc.Ref().ContentID}
	limit := organization.Setting[int64]{Value: 4096, Source: source}
	required := []string{}
	w := bindingWire{Version: 3, ManagerPolicy: liveManagerPolicy(), MaxContextBytes: &limit, RequiredChecks: &required, Documents: []documentBinding{{Source: organization.SettingSource{File: "execution", ContentID: source.ContentID}, Bytes: doc.Bytes()}}}
	if err := validateManagerPolicy(w); err != nil {
		t.Fatal(err)
	}
	w.ManagerPolicy.Operation.Bytes = []byte("narrowed responsibility")
	w.ManagerPolicy.Operation.Product.ContentID = sha256.Sum256(w.ManagerPolicy.Operation.Bytes)
	if err := validateManagerPolicy(w); !errors.Is(err, ErrInvalid) {
		t.Fatal("self-consistent forged immutable operation", err)
	}
	for _, version := range []uint32{1, 2} {
		for _, field := range []string{"policy", "scope", "limit"} {
			old := bindingWire{Version: version}
			switch field {
			case "policy":
				old.ManagerPolicy = liveManagerPolicy()
			case "scope":
				scope := []organization.Setting[organization.ScopeRule]{}
				old.Scope = &scope
			case "limit":
				old.MaxContextBytes = &limit
			}
			if err := validateManagerPolicy(old); !errors.Is(err, ErrInvalid) {
				t.Fatal("new field smuggled into legacy", version, field, err)
			}
		}
	}
}

// A send's answer says how to end the step for the phase and message kind;
// a reply step needs no note. In D2 and D4 a root ended its work step right
// after a question, without an outcome.
func TestSendNoteFitsThePhase(t *testing.T) {
	t.Parallel()
	question, hint := []byte(`{"kind":"question","to":"parent","body":"Which?"}`), []byte(`{"kind":"hint","to":"a","body":"Use it."}`)
	for _, tc := range []struct {
		phase    Phase
		args     []byte
		contains string
	}{
		{PhaseWork, question, `{"on":[{"kind":"reply","ref":"q-1"}],"reason":"Needs the answer."}`},
		{PhaseIntegrate, hint, "submit or wait"},
		{PhaseReply, []byte(`{"kind":"reply","to":"q-1","body":"Yes."}`), ""},
	} {
		note := sendNote(tc.phase, tc.args, "q-1")
		if tc.contains == "" && note != "" || tc.contains != "" && !strings.Contains(note, tc.contains) {
			t.Fatalf("%s: note %q", tc.phase, note)
		}
	}
}
