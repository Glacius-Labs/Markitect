package run_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type fakeChecks struct {
	exit  int
	runs  []run.CheckRequest
	clock *time.Time // advanced by took on each run
	took  time.Duration
	// result, when set, decides each run's exit code and output; notStarted
	// reports runs whose process never started.
	result     func(run.CheckRequest) (int, string)
	notStarted bool
}

func (f *fakeChecks) Run(_ context.Context, r run.CheckRequest) (run.CheckReceipt, error) {
	f.runs = append(f.runs, r)
	if f.clock != nil {
		*f.clock = f.clock.Add(f.took)
	}
	exit, output := f.exit, "ok\n"
	if f.result != nil {
		exit, output = f.result(r)
	}
	return run.CheckReceipt{ID: r.ID, ImageID: r.ImageID, Tree: r.Tree.Digest(), Started: !f.notStarted, Exited: !f.notStarted, ExitCode: exit, Output: []byte(output)}, nil
}

func requiredCheck(t *testing.T) run.RequiredCheck {
	t.Helper()
	dir := snapshot.RootDirectory()
	return run.RequiredCheck{Name: "go-tests", Request: run.CheckRequest{ID: "check-go-tests", ImageID: testImage, Executable: "/usr/local/go/bin/go", Args: []string{"test", "./..."},
		Directory: dir, Limits: run.CheckLimits{MilliCPUs: 1000, MemoryBytes: 1 << 30, Processes: 256}, Timeout: time.Minute, MaxOutputBytes: 1 << 16}}
}

func verifiedHarness(t *testing.T, exit int) (*harness, *fakeChecks) {
	t.Helper()
	h := newHarness(t, true, happyAgent)
	checks := &fakeChecks{exit: exit}
	h.engine = h.newEngineWith(h.c, checks)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not reach Verifying", h.c.Current())
	}
	target := h.engine.Aggregate().Tree()
	passed, err := h.engine.Verify(context.Background(), target, []run.RequiredCheck{requiredCheck(t)})
	if err != nil {
		t.Fatal(err)
	}
	if h.c.Current().State == run.Verifying {
		commit(t, h.c, h.j, h.c.VerifyRunCommand(meta("verify"), target.Digest(), passed))
	}
	return h, checks
}

// The required checks run on the exact aggregate before publication, and the
// final evidence check moves the run to Publishing (execution contract,
// Verifying).
func TestVerificationRunsRequiredChecksOnTheAggregate(t *testing.T) {
	t.Parallel()
	h, checks := verifiedHarness(t, 0)
	if h.c.Current().State != run.Publishing || len(checks.runs) != 1 || checks.runs[0].Tree.Digest() != h.c.AggregateTree() {
		t.Fatal("verification did not check the aggregate", h.c.Current(), checks.runs)
	}
	if !h.engine.Aggregate().Tree().Equal(checks.runs[0].Tree) {
		t.Fatal("the checked tree is not the engine's aggregate")
	}
	views := h.c.Checks()
	if len(views) != 1 || !views[0].Passed || views[0].Record.Receipt == (run.BlobRef{}) {
		t.Fatal("check receipt not recorded", views)
	}
	if b := h.c.Budget(); b.Held != (run.Capacity{}) {
		t.Fatal("check reservation not settled", b.Held)
	}
}

func TestFailingRequiredCheckFinishesTheRunIncomplete(t *testing.T) {
	t.Parallel()
	h, _ := verifiedHarness(t, 1)
	if r := h.c.Current(); r.State != run.Finished || r.Outcome != run.Incomplete {
		t.Fatal("failing check did not finish the run Incomplete", r)
	}
}

// Publication records its changes before the first write; its receipt
// decides the outcome (execution contract, Publishing and outcomes).
func TestPublicationOutcomes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		outcome string
		empty   bool
		state   run.State
		result  run.Outcome
		err     error
	}{
		{"applied", false, run.Finished, run.Applied, nil},
		{"applied", true, "", "", run.ErrInvalid},
		{"no_change", true, run.Finished, run.NoChange, nil},
		{"no_change", false, "", "", run.ErrInvalid},
		{"stale", false, run.Finished, run.Stale, nil},
		{"failed", false, run.Finished, run.Failed, nil},
		{"partial", false, run.Reconciling, "", nil},
		{"unknown", false, run.Reconciling, "", nil},
	} {
		t.Run(tc.outcome+map[bool]string{true: "-empty"}[tc.empty], func(t *testing.T) {
			h, _ := verifiedHarness(t, 0)
			source := h.inputs.Binding.Census
			changes := snapshot.Compare(source, h.engine.Aggregate().Tree())
			if tc.empty {
				changes = nil
			}
			if !tc.empty && len(changes) == 0 {
				t.Fatal("the aggregate has no changes")
			}
			prepare, err := h.c.PreparePublicationCommand(meta("prepare-publication"), "publication-1", h.engine.Aggregate().Tree().Digest(), changes)
			if err != nil {
				t.Fatal(err)
			}
			commit(t, h.c, h.j, prepare)
			publish, err := h.c.PublishRunCommand(meta("published"), run.PublicationResult{ID: "publication-1", Outcome: tc.outcome})
			if err != nil {
				t.Fatal(err)
			}
			if tc.err != nil {
				if _, err := h.c.Propose(publish); !errors.Is(err, tc.err) {
					t.Fatal("publication outcome accepted against its changes", err)
				}
				return
			}
			commit(t, h.c, h.j, publish)
			if r := h.c.Current(); r.State != tc.state || r.Outcome != tc.result {
				t.Fatal("wrong run outcome", r)
			}
			restored, err := run.Restore(context.Background(), &readJournal{store: h.j}, "repo", run.RestoreLimits{PageSize: 16, Batches: 2000, Events: 4000, Bytes: 64 << 20})
			if err != nil || restored.Current() != h.c.Current() || !reflect.DeepEqual(restored.Checks(), h.c.Checks()) {
				t.Fatal("restore lost verification or publication", err)
			}
		})
	}
}

func TestVerificationRefusesUnadmittedChecks(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true, happyAgent)
	check := requiredCheck(t)
	check.Request.Tree = h.inputs.Binding.Census
	if cmd, err := h.c.PrepareCheckCommand(meta("early-check"), check.Name, check.Request, run.Reservation{ID: "early", Kind: run.ProjectCheck, Purpose: run.Assessment, Maximum: time.Minute}); err != nil {
		t.Fatal(err)
	} else if _, err := h.c.Propose(cmd); err == nil {
		t.Fatal("a check ran while the run was not Verifying")
	}
	if _, err := h.c.Propose(h.c.VerifyRunCommand(meta("early-verify"), h.c.AggregateTree(), nil)); !errors.Is(err, run.ErrState) {
		t.Fatal("a running run was verified", err)
	}
}

// A candidate review runs a configured check on the frozen candidate and cites
// its receipt; other steps and unknown checks are refused (Host tools).
func TestCandidateReviewRequestsChecksOnItsCandidate(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			return append([]action{{Tool: run.ToolRequestCheck, Args: `{"check":"go-tests"}`, Want: run.ToolRefused}}, happyAgent(s)...)
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate:
			return append([]action{
				{Tool: run.ToolRequestCheck, Args: `{"check":"lint"}`, Want: run.ToolRefused},
				{Tool: run.ToolRequestCheck, Args: `{"check":"slow"}`, Want: run.ToolRefused},
				{Tool: run.ToolRequestCheck, Args: `{"check":"go-tests"}`, Want: run.ToolOK},
			}, happyAgent(s)...)
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	checks := &fakeChecks{clock: &h.clock, took: 15 * time.Second}
	h.engine = h.newEngineWith(h.c, checks)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(checks.runs) != 1 {
		t.Fatal("review check not run once", h.c.Current(), len(checks.runs))
	}
	// The check's time is charged once, as the check's own invocation.
	for _, v := range h.c.Invocations() {
		if v.Reservation.Kind == run.AgentStep && v.Elapsed >= checks.took {
			t.Fatal("a step was charged the check's time", v.Reservation.ID, v.Elapsed)
		}
		if v.Reservation.Kind == run.ProjectCheck && v.Elapsed < checks.took {
			t.Fatal("the check was not charged its time", v.Elapsed)
		}
	}
	views := h.c.Checks()
	// lending's candidate became the aggregate, so its tree is the aggregate's.
	if len(views) != 1 || views[0].Record.Task != "lending" || !views[0].Passed || views[0].Record.Tree != h.engine.Aggregate().Tree().Digest() {
		t.Fatal("review check not recorded on its candidate", views)
	}
	if b := h.c.Budget(); b.Held != (run.Capacity{}) {
		t.Fatal("check hold survived", b.Held)
	}
}

// The final evidence check names every check the basis requires; a check
// that passed on the tree is not run again; a prepared publication cannot be
// finished over without its receipt.
func TestVerificationBindsRequiredChecksAndPublication(t *testing.T) {
	t.Parallel()
	inputs := stepInputs(t, true)
	inputs.Binding.RequiredChecks = []string{"go-tests"}
	h := newHarnessWith(t, inputs, happyAgent)
	checks := &fakeChecks{}
	h.engine = h.newEngineWith(h.c, checks)
	runToEnd(t, h)
	// Each submission already ran the required check as the Host's early gate.
	gated := len(checks.runs)
	target := h.engine.Aggregate().Tree()
	if _, err := h.c.Propose(h.c.VerifyRunCommand(meta("unchecked"), target.Digest(), nil)); !errors.Is(err, run.ErrState) {
		t.Fatal("verified without the required check", err)
	}
	first, err := h.engine.Verify(context.Background(), target, []run.RequiredCheck{requiredCheck(t)})
	if err != nil {
		t.Fatal(err)
	}
	again, err := h.engine.Verify(context.Background(), target, []run.RequiredCheck{requiredCheck(t)})
	if err != nil || len(checks.runs) != gated+1 || !reflect.DeepEqual(first, again) {
		t.Fatal("a passed check ran again", len(checks.runs), gated, err)
	}
	commit(t, h.c, h.j, h.c.VerifyRunCommand(meta("verify"), target.Digest(), first))
	prepare, err := h.c.PreparePublicationCommand(meta("prepare-publication"), "publication-1", target.Digest(), snapshot.Compare(h.inputs.Binding.Census, target))
	if err != nil {
		t.Fatal(err)
	}
	commit(t, h.c, h.j, prepare)
	finish := command("finish", run.FinishRun)
	finish.Run, finish.Outcome, finish.Reason = h.c.Current().ID, run.Failed, "give up"
	if _, err := h.c.Propose(finish); !errors.Is(err, run.ErrState) {
		t.Fatal("a prepared publication was finished over without its receipt", err)
	}
	other, _ := h.c.PreparePublicationCommand(meta("other-target"), "publication-2", h.inputs.Binding.Census.Digest(), nil)
	if _, err := h.c.Propose(other); err == nil {
		t.Fatal("a second publication was prepared")
	}
}

// A review check the budget does not fit is refused and the review goes on; a
// required check the budget does not fit finishes the run Incomplete.
func TestChecksOutsideTheBudget(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate {
			return append([]action{{Tool: run.ToolRequestCheck, Args: `{"check":"go-tests"}`, Want: run.ToolRefused}}, happyAgent(s)...)
		}
		return happyAgent(s)
	}
	inputs := stepInputs(t, true)
	// A review step holds a minute, which leaves no room for its check.
	inputs.Binding.Policy.Assignment.ExecutionTime = 66 * time.Second
	inputs.Binding.Policy.Reserve.ExecutionTime = time.Second
	h := newHarnessWith(t, inputs, agent)
	checks := &fakeChecks{}
	h.engine = h.newEngineWith(h.c, checks)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(checks.runs) != 0 {
		t.Fatal("a review check outside the budget ran or stopped the run", h.c.Current(), len(checks.runs))
	}
	required := requiredCheck(t)
	required.Request.Timeout = 2 * time.Minute
	if _, err := h.engine.Verify(context.Background(), h.engine.Aggregate().Tree(), []run.RequiredCheck{required}); err != nil {
		t.Fatal(err)
	}
	if r := h.c.Current(); r.State != run.Finished || r.Outcome != run.Incomplete || len(checks.runs) != 0 {
		t.Fatal("a required check outside the budget did not finish the run Incomplete", r)
	}
}

// brokenLibrary fails while out/library.go is marked broken.
func brokenLibrary(t *testing.T) func(run.CheckRequest) (int, string) {
	return func(r run.CheckRequest) (int, string) {
		if e, ok := r.Tree.Lookup(mustPath(t, "out/library.go")); ok && strings.Contains(string(e.Bytes()), "BROKEN") {
			return 1, "FAIL: library is broken\n"
		}
		return 0, "ok\n"
	}
}

// gatedHarness runs the one-Package run with go-tests as the basis's required
// check and as a configured check.
func gatedHarness(t *testing.T, agent agentScript, checks *fakeChecks) *harness {
	t.Helper()
	inputs := stepInputs(t, true)
	inputs.Binding.RequiredChecks = []string{"go-tests"}
	h := newHarnessWith(t, inputs, agent)
	h.engine = h.newEngineWith(h.c, checks)
	return h
}

// checksCharged counts the project checks that were charged for the purpose.
func checksCharged(h *harness, purpose run.Purpose) int {
	n := 0
	for _, v := range h.c.Invocations() {
		if v.Reservation.Kind == run.ProjectCheck && v.Reservation.Purpose == purpose {
			n++
		}
	}
	return n
}

// fixesItsFile submits a file that fails the check, is refused, and submits
// the corrected file in the same step.
func fixesItsFile(s agentStep) []action {
	if s.Manager != "lending" || s.Phase != run.PhaseWork {
		return happyAgent(s)
	}
	return []action{
		{Write: map[string]string{"out/library.go": "package library\n\n// BROKEN\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolRefused},
		{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{}\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
	}
}

// A submit whose required check fails is refused with the end of the output
// while the step still runs; the Agent fixes its file and submits in the same
// step, so the step concludes as one submission and its review passes (core
// rules S4, T8).
func TestSubmitIsRefusedWhileARequiredCheckFails(t *testing.T) {
	t.Parallel()
	checks := &fakeChecks{}
	h := gatedHarness(t, fixesItsFile, checks)
	checks.result = brokenLibrary(t)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.c.Current(), h.sessions.answers)
	}
	if n := strings.Count(strings.Join(admitted(h), ","), "lending work"); n != 1 {
		t.Fatal("the step did not conclude as one submission", admitted(h), h.sessions.answers)
	}
	if !slices.Contains(h.sessions.answers, "lending work submit refused the required check go-tests fails on these files; fix it, or wait for what it needs: FAIL: library is broken") {
		t.Fatal("the submit was not refused with the check output", h.sessions.answers)
	}
	if task := h.task("lending"); len(task.Deliveries) == 0 || len(task.Rejections) != 0 {
		t.Fatal("the submission was not accepted without a rejection", task)
	}
}

// A passing required check lets the submit through; the review still runs its
// own check, and the gate's checks are charged as writing work.
func TestAPassingRequiredCheckLetsTheSubmitThrough(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate {
			return append([]action{{Tool: run.ToolRequestCheck, Args: `{"check":"go-tests"}`, Want: run.ToolOK}}, happyAgent(s)...)
		}
		return happyAgent(s)
	}
	h := gatedHarness(t, agent, &fakeChecks{})
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.c.Current(), h.sessions.answers)
	}
	if checksCharged(h, run.Assessment) != 1 || checksCharged(h, run.Writing) == 0 || len(h.c.Checks()) != checksCharged(h, run.Assessment)+checksCharged(h, run.Writing) {
		t.Fatal("the review's check or the gate did not run", checksCharged(h, run.Assessment), checksCharged(h, run.Writing), len(h.c.Checks()))
	}
	for _, v := range h.c.Checks() {
		if !v.Passed || v.Record.Task == "" {
			t.Fatal("a check did not pass for its Manager", v)
		}
	}
}

// Without required checks the Host runs no gate check.
func TestNoRequiredChecksRunNoGateCheck(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true, happyAgent)
	checks := &fakeChecks{exit: 1}
	h.engine = h.newEngineWith(h.c, checks)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(checks.runs) != 0 || len(h.c.Checks()) != 0 {
		t.Fatal("a gate check ran without required checks", h.c.Current(), len(checks.runs))
	}
}

// The gate's checks are journaled as writing work, settle with the step and
// restore from the journal; a verification check is not admitted as writing
// work.
func TestGateChecksAreJournaledAsWritingWork(t *testing.T) {
	t.Parallel()
	checks := &fakeChecks{}
	h := gatedHarness(t, fixesItsFile, checks)
	checks.result = brokenLibrary(t)
	runToEnd(t, h)
	if checksCharged(h, run.Writing) < 2 || len(h.c.Checks()) != checksCharged(h, run.Writing) {
		t.Fatal("the gate checks were not charged as writing work", checksCharged(h, run.Writing), h.c.Checks())
	}
	for _, v := range h.c.Invocations() {
		if v.Reservation.Kind == run.ProjectCheck && v.State != run.Settled {
			t.Fatal("a gate check stayed open", v)
		}
	}
	if b := h.c.Budget(); b.Held != (run.Capacity{}) {
		t.Fatal("a gate hold survived", b.Held)
	}
	restored, err := run.Restore(context.Background(), &readJournal{store: h.j}, "repo", run.RestoreLimits{PageSize: 16, Batches: 2000, Events: 4000, Bytes: 64 << 20})
	if err != nil || !reflect.DeepEqual(restored.Checks(), h.c.Checks()) || !reflect.DeepEqual(restored.Invocations(), h.c.Invocations()) {
		t.Fatal("restore lost the gate checks", err)
	}
	check := requiredCheck(t)
	check.Request.Tree = h.engine.Aggregate().Tree()
	cmd, err := h.c.PrepareCheckCommand(meta("writing-verification"), check.Name, check.Request, run.Reservation{ID: "writing-verification", Kind: run.ProjectCheck, Purpose: run.Writing, Maximum: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.c.Propose(cmd); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("a verification check was admitted as writing work", err)
	}
}

// The gate is advisory: a check that does not fit the step's remaining time is
// skipped and the submit goes through.
func TestTheGateIsSkippedWhenTheCheckDoesNotFitTheStep(t *testing.T) {
	t.Parallel()
	var h *harness
	agent := func(s agentStep) []action {
		if s.Phase == run.PhaseWork || s.Phase == run.PhaseIntegrate {
			h.clock = h.clock.Add(45 * time.Second)
		}
		return happyAgent(s)
	}
	checks := &fakeChecks{exit: 1}
	h = gatedHarness(t, agent, checks)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(checks.runs) != 0 {
		t.Fatal("a gate check ran without time", h.c.Current(), len(checks.runs), h.sessions.answers)
	}
}

// The refusal carries check output, which may hold anything: a tail cut inside
// a character, an invalid byte and a NUL must still leave the Host a refusal
// it can record, so the Agent corrects the files in the same step.
func TestGateRefusalIsRecordableTextWhateverTheOutput(t *testing.T) {
	t.Parallel()
	// 9000 bytes of three-byte characters: the 8 KiB cut falls inside one.
	output := strings.Repeat("€", 3000) + "\x00 tail\xff\x01 end\n"
	checks := &fakeChecks{}
	h := gatedHarness(t, fixesItsFile, checks)
	checks.result = func(r run.CheckRequest) (int, string) {
		if code, _ := brokenLibrary(t)(r); code != 0 {
			return 1, output
		}
		return 0, "ok\n"
	}
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.c.Current(), h.sessions.answers)
	}
	const prefix = "lending work submit refused the required check go-tests fails on these files; fix it, or wait for what it needs: "
	var reason string
	for _, a := range h.sessions.answers {
		if strings.HasPrefix(a, prefix) {
			reason = strings.TrimPrefix(a, prefix)
		}
	}
	if reason == "" || !utf8.ValidString(reason) || strings.ContainsAny(reason, "\x00\x01") || len(reason) > 8<<10 || !strings.HasSuffix(reason, "tail end") {
		t.Fatalf("the refusal is not clean text of at most 8 KiB: %d bytes %q", len(reason), reason[max(0, len(reason)-20):])
	}
}

// A check that did not run to an observed result says nothing about the files,
// so the submit goes through.
func TestGateStaysSilentWhenTheCheckDidNotRun(t *testing.T) {
	t.Parallel()
	checks := &fakeChecks{exit: 1, notStarted: true}
	h := gatedHarness(t, happyAgent, checks)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(checks.runs) == 0 {
		t.Fatal("the gate did not try the check or refused a submit", h.c.Current(), len(checks.runs), h.sessions.answers)
	}
}

// Writing work protects the completion reserve: a gate check that would use it
// is skipped like one that does not fit the step.
func TestTheGateIsSkippedWhenTheCheckWouldNeedTheReserve(t *testing.T) {
	t.Parallel()
	inputs := stepInputs(t, true)
	inputs.Binding.RequiredChecks = []string{"go-tests"}
	// A work step fits above the reserve; the 22s check on top of it does not.
	inputs.Binding.Policy.Reserve.ExecutionTime = 8*time.Minute + 50*time.Second
	h := newHarnessWith(t, inputs, happyAgent)
	checks := &fakeChecks{exit: 1}
	h.engine = h.newEngineWith(h.c, checks)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(checks.runs) != 0 {
		t.Fatal("a gate check used the completion reserve", h.c.Current(), len(checks.runs), h.sessions.answers)
	}
}

// The reducer admits a task check only while the task holds an admitted work,
// integrate or candidate-review step, charged for that step's purpose.
func TestReducerAdmitsTaskChecksOnlyInWritingAndCandidateSteps(t *testing.T) {
	t.Parallel()
	var h *harness
	var asked string
	got := map[string]error{}
	probe := func(name, manager string, purpose run.Purpose) {
		check := requiredCheck(t)
		check.Request.Tree = h.inputs.Binding.Census
		check.Request.ID = run.CheckID("check-probe-" + name)
		cmd, err := h.c.PrepareReviewCheckCommand(meta("probe-"+name), manager, check.Name, check.Request, run.Reservation{ID: "probe-" + name, Kind: run.ProjectCheck, Purpose: purpose, Maximum: time.Minute})
		if err != nil {
			t.Fatal(err)
		}
		_, got[name] = h.c.Propose(cmd)
	}
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending", "Should Item carry a title?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(asked), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseReply:
			probe("reply", "lending", run.Writing)
			return []action{{Tool: run.ToolSend, Args: sendArgs("reply", asked, "Yes, a title."), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerExpectations && s.Attempt == 0:
			probe("expectations", "lending", run.Assessment)
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			probe("writing", "lending", run.Writing)
			probe("assessment-in-writing", "lending", run.Assessment)
			probe("no-step", ".", run.Writing)
		}
		return happyAgent(s)
	}
	h = newHarness(t, true, agent)
	h.engine = h.newEngineWith(h.c, &fakeChecks{})
	runToEnd(t, h)
	want := map[string]error{"writing": nil, "reply": run.ErrState, "expectations": run.ErrState, "no-step": run.ErrState, "assessment-in-writing": run.ErrInvalid}
	for name, err := range want {
		if !errors.Is(got[name], err) || err == nil && got[name] != nil {
			t.Errorf("%s check: %v, want %v", name, got[name], err)
		}
	}
	if len(got) != len(want) {
		t.Errorf("probes ran: %v", got)
	}
}
