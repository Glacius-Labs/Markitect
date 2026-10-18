package run_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Only the step the queue admits can start, with the phase's role, tools,
// workspace mode, provider home and conversation (core rules T4, S3, S5).
func TestTaskAdmissionRefusesWhatTheQueueDidNotAdmit(t *testing.T) {
	t.Parallel()
	c, j := running(t)
	item, ok := c.Next()
	if !ok || item.Manager != "empty" || item.Phase != run.PhaseExplore || item.Resume != "" {
		t.Fatal("children-first Explore is not the first admission", item)
	}
	good := taskSession(t, c, "s", stepInputs(t, false).Binding.Census.Digest())
	profile := payload(t, `{"profile":"synthetic recorded metadata"}`)
	workTools, _ := run.ToolCatalog(run.PhaseWork)
	exploreTools, _ := run.ToolCatalog(run.PhaseExplore)
	for name, edit := range map[string]func(*run.SessionPlan, *run.Reservation) run.Payload{
		"writable explore": func(p *run.SessionPlan, _ *run.Reservation) run.Payload { p.ReadOnly = false; return exploreTools },
		"reviewer role": func(p *run.SessionPlan, r *run.Reservation) run.Payload {
			p.Role, r.Purpose = "reviewer", run.Assessment
			return exploreTools
		},
		"other catalogue": func(p *run.SessionPlan, _ *run.Reservation) run.Payload { p.Tools = workTools.Ref(); return workTools },
		"other home": func(p *run.SessionPlan, _ *run.Reservation) run.Payload {
			p.ProviderHome = "other-home"
			return exploreTools
		},
		"invented resume": func(p *run.SessionPlan, _ *run.Reservation) run.Payload { p.Resume = "invented"; return exploreTools },
		"assessment spend": func(_ *run.SessionPlan, r *run.Reservation) run.Payload {
			r.Purpose = run.Assessment
			return exploreTools
		},
	} {
		plan, reservation := good.Session.Plan, good.Reservation
		tools := edit(&plan, &reservation)
		cmd, err := c.PrepareTaskSessionCommand(meta("bad-"+name), item, plan, reservation, profile, tools)
		if err == nil {
			_, err = c.Propose(cmd)
		}
		if err == nil {
			t.Fatal(name, "accepted")
		}
	}
	raw := run.PrepareSessionCommand(meta("raw"), "a", "r", good.Session.Plan, good.Reservation, profile, exploreTools)
	if _, err := c.Propose(raw); !errors.Is(err, run.ErrState) {
		t.Fatal("unproven session preparation accepted", err)
	}
	other := item
	other.Manager = "lending/reminders"
	if _, err := c.PrepareTaskSessionCommand(meta("other"), other, good.Session.Plan, good.Reservation, profile, exploreTools); !errors.Is(err, run.ErrState) {
		t.Fatal("a step the queue did not admit was prepared", err)
	}
	commit(t, c, j, good)
	if next, ok := c.Next(); !ok || next.Manager != "lending/reminders" {
		t.Fatal("queue lost the other leaf", next)
	}
	again := taskSession(t, c, "second", stepInputs(t, false).Binding.Census.Digest())
	if _, err := c.Propose(again); !errors.Is(err, run.ErrState) {
		t.Fatal("a second step started while one is live (core rule S5)", err)
	}
}

// Step preparation compiles the original model again and matches the exact
// bound settings; nothing the binding does not record can change the input.
func TestStepPreparationRecompilesOriginalsAndMatchesExactBoundSettings(t *testing.T) {
	t.Parallel()
	inputs := stepInputs(t, true)
	b, err := run.Bind(inputs.Binding)
	if err != nil {
		t.Fatal(err)
	}
	c, j := runningWith(t, b)
	cmd := taskSession(t, c, "s", inputs.Binding.Census.Digest())
	commit(t, c, j, cmd)
	io, _ := c.SessionChannel(j, "s", run.RefuseTools)
	send(t, io, "init", "")
	if _, err := io.Ready(context.Background(), readyObservation(t)); err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("open-s", time.Second))
	plan := run.StepPlan{Session: "s", ID: "first", SessionBinding: cmd.Session.Plan.Binding()}
	reservation := run.Reservation{ID: "first", Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Second}
	for _, kind := range []string{"model", "settings", "basis", "context-limit"} {
		t.Run(kind, func(t *testing.T) {
			altered := inputs
			switch kind {
			case "model":
				altered.Binding.ModelInput.Root.Entries = append([]model.Entry(nil), inputs.Binding.ModelInput.Root.Entries...)
				altered.Binding.ModelInput.Root.Entries[1].Bytes = []byte("id: changed\n")
			case "settings":
				altered.Settings.Defaults.RejectionLimit.Value = 3
			case "basis":
				altered.Binding.Source.Guard = "other"
			case "context-limit":
				altered.Binding.MaxContextBytes.Value = 1
			}
			if _, err := c.PrepareTaskStepCommand(meta("bad-"+kind), plan, reservation, altered, sourceFacts(inputs)); err == nil {
				t.Fatal("accepted unchecked changed producer input")
			}
		})
	}
	if _, err := c.PrepareTaskStepCommand(meta("good"), plan, reservation, inputs, sourceFacts(inputs)); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredReviewDemandIncludesRootEmptyAndTwoReviewSteps(t *testing.T) {
	t.Parallel()
	for _, one := range []bool{true, false} {
		inputs := stepInputs(t, one)
		want := int64(6)
		if one {
			want = 3
		}
		b, err := run.Bind(inputs.Binding)
		if err != nil || b.RequiredReviews() != want {
			t.Fatal(b.RequiredReviews(), err)
		}
		for _, kind := range []string{"starts", "steps"} {
			bad := inputs.Binding
			if kind == "starts" {
				bad.Policy.Reserve.SessionStarts = 2*want - 1
			} else {
				bad.Policy.Reserve.Steps = 2*want - 1
			}
			if _, err := run.Bind(bad); !errors.Is(err, run.ErrInvalid) {
				t.Fatal("underfunded review steps accepted", kind, err)
			}
		}
	}
}

// Each step's context carries the scoped model, the owner's request at the
// root, the children's reports for a parent and the Manager's own artifacts
// (core rules T1, D3).
func TestStepContextCarriesScopeOwnerRequestReportsAndOwnArtifacts(t *testing.T) {
	t.Parallel()
	var facts organization.SettingsFacts
	input := fixtureWith(t, func(m *model.Input) {
		for _, d := range []struct{ path, text string }{
			{"model/lending/applies.yaml", "imports: [{package: markitect/core, as: Core}]\nid: applies\nkind: Core.Property\nbody: Explicit modeled scope.\nspec: {type: string, minCount: 0, maxCount: -1}\n"},
			{"model/lending/scoped-rule.yaml", "imports: [{package: markitect/core, as: Core}]\nid: ScopedRule\nkind: Core.Kind\nbody: Scoped obligation type.\nspec: {properties: [applies]}\n"},
			{"model/lending/reminders/global.yaml", "imports: [{package: lending}]\nid: global\nkind: lending/ScopedRule\nbody: Global scoped initial assessment obligation.\nspec: {applies: ['.']}\n"},
		} {
			m.Root.Entries = append(m.Root.Entries, model.Entry{Path: d.path, Kind: model.Regular, Bytes: []byte(d.text)})
		}
	}, func(f *organization.SettingsFacts) { facts = *f })
	input.Scope = []organization.Setting[organization.ScopeRule]{{Value: organization.ScopeRule{Kind: model.Identity{Package: "lending", ID: "ScopedRule"}, Property: model.Identity{Package: "lending", ID: "applies"}}, Source: source(input.Documents[0], "/scope/0")}}
	inputs := run.StepPreparation{Binding: input, Settings: facts}
	h := newHarnessWith(t, inputs, happyAgent)
	h.advanceUntil(func(a run.Admission) bool { return a.Manager == "." && a.Phase == run.PhaseExplore })
	if lending := h.packet("lending", run.PhaseExplore); !bytes.Contains(lending.Context.Bytes, []byte("package library\n")) || !bytes.Contains(lending.Context.Bytes, []byte("Inspected the area.")) {
		t.Fatal("own artifact or child report omitted from the parent's Explore context")
	}
	// The scope is part of the bound basis; a substitute is refused.
	cmd := taskSession(t, h.c, "s", h.engine.Aggregate().Tree().Digest())
	commit(t, h.c, h.j, cmd)
	io, _ := h.c.SessionChannel(h.j, "s", run.RefuseTools)
	send(t, io, "init", "")
	if _, err := io.Ready(context.Background(), readyObservation(t)); err != nil {
		t.Fatal(err)
	}
	commit(t, h.c, h.j, settle("open-s", time.Second))
	plan := run.StepPlan{Session: "s", ID: "root-step", SessionBinding: cmd.Session.Plan.Binding()}
	reservation := run.Reservation{ID: "root-step", Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Second}
	stepFacts := run.StepFacts{Tree: h.engine.Aggregate().Tree(), Registry: h.engine.Aggregate().Registry()}
	changed := inputs
	changed.Binding.Scope = nil
	if _, err := h.c.PrepareTaskStepCommand(meta("substitute-scope"), plan, reservation, changed, stepFacts); err == nil {
		t.Fatal("scope replaced under unchanged basis")
	}
	if _, err := h.c.PrepareTaskStepCommand(meta("missing-reports"), plan, reservation, inputs, stepFacts); err == nil {
		t.Fatal("parent context rendered without its children's reports")
	}
	for _, ref := range h.c.StepQuotes(".") {
		stepFacts.Quoted = append(stepFacts.Quoted, h.j.blobs[ref])
	}
	step, err := h.c.PrepareTaskStepCommand(meta("root-step"), plan, reservation, inputs, stepFacts)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, h.c, h.j, step)
	var packet run.StepPacket
	if err := json.Unmarshal(h.j.blobs[step.Session.StepPlan.Input].Bytes(), &packet); err != nil {
		t.Fatal(err)
	}
	root := packet.Context.Bytes
	if !bytes.Contains(root, []byte("Global scoped initial assessment obligation.")) || !bytes.Contains(root, input.OwnerRequest.Bytes()) {
		t.Fatal("scoped model or actual root request omitted")
	}
}

// A step concludes only after its container stopped and with its answered
// outcome; the reducer checks that outcome again (core rules S1, T5, T6).
func TestConclusionWaitsForTheStoppedContainerAndChecksTheOutcome(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"report", "bad report", "refused"} {
		t.Run(kind, func(t *testing.T) {
			inputs := stepInputs(t, true)
			b, _ := run.Bind(inputs.Binding)
			c, j := runningWith(t, b)
			ctx := context.Background()
			cmd := taskSession(t, c, "s", inputs.Binding.Census.Digest())
			commit(t, c, j, cmd)
			answer := run.ToolOutcome{Status: run.ToolOK, Result: []byte(`{"accepted":true}`)}
			if kind == "refused" {
				answer = run.ToolOutcome{Status: run.ToolRefused, Reason: "refused by the Host"}
			}
			io, _ := c.SessionChannel(j, "s", func(context.Context, run.ToolCall, int64) (run.ToolOutcome, error) { return answer, nil })
			send(t, io, "init", "")
			if _, err := io.Ready(ctx, readyObservation(t)); err != nil {
				t.Fatal(err)
			}
			commit(t, c, j, settle("open-s", time.Second))
			commit(t, c, j, stepPreparationWith(t, c, inputs, "step", "invoke-step"))
			send(t, io, "turn", "step")
			if _, err := io.Bind(ctx, run.SessionObservation{RecordMeta: meta("bind"), ProviderSession: "provider-s", Wire: wire(t, `{"thread":"s"}`), Reason: "observed"}); err != nil {
				t.Fatal(err)
			}
			if _, err := io.ObserveStep(ctx, run.StepObservation{RecordMeta: meta("started"), Step: "step", ProviderStep: "turn", Wire: wire(t, `{"turn":"started"}`), Reason: "observed"}); err != nil {
				t.Fatal(err)
			}
			args := reportArgs
			if kind == "bad report" {
				args = `{"summary":"Missing lists."}`
			}
			if _, err := io.Call(ctx, run.ToolCall{RecordMeta: meta("call"), Step: "step", Call: "c1", Tool: run.ToolReport, Input: wire(t, `{"id":1,"name":"report","arguments":`+args+`}`)}); err != nil {
				t.Fatal(err)
			}
			conclude := func() error {
				var input run.Payload
				if ref, ok := c.OutcomeInput("lending"); ok {
					input = j.blobs[ref]
				}
				cmd, err := c.ConcludeStepCommand(meta("conclude"), "lending", run.StepResult{Input: input, Aggregate: c.AggregateVersion()})
				if err != nil {
					return err
				}
				_, err = c.Propose(cmd)
				return err
			}
			if err := conclude(); !errors.Is(err, run.ErrState) {
				t.Fatal("concluded a running step", err)
			}
			if _, err := io.ObserveStep(ctx, run.StepObservation{RecordMeta: meta("terminal"), Step: "step", ProviderStep: "turn", Wire: wire(t, `{"turn":"completed"}`), Terminal: true, Reason: "observed"}); err != nil {
				t.Fatal(err)
			}
			commit(t, c, j, settle("invoke-step", time.Second))
			if err := conclude(); !errors.Is(err, run.ErrState) {
				t.Fatal("concluded before the container stopped", err)
			}
			if _, err := io.ObserveStop(ctx, run.StopObservation{RecordMeta: meta("stopped"), Terminated: true, Reason: "observed container exit"}); err != nil {
				t.Fatal(err)
			}
			err := conclude()
			switch kind {
			case "report", "refused":
				if err != nil {
					t.Fatal(err)
				}
				ref, _ := c.OutcomeInput("lending")
				var input run.Payload
				if kind == "report" {
					input = j.blobs[ref]
				}
				cmd, _ := c.ConcludeStepCommand(meta("conclude"), "lending", run.StepResult{Input: input, Aggregate: c.AggregateVersion()})
				commit(t, c, j, cmd)
				want := run.TaskExplored
				if kind == "refused" {
					want = run.TaskQueued // core rule T6
				}
				if v, _ := c.Task("lending"); v.State != want {
					t.Fatal("conclusion applied the wrong outcome", v.State)
				}
			case "bad report":
				if !errors.Is(err, run.ErrState) {
					t.Fatal("an outcome the Host's check refuses was concluded", err)
				}
				cmd, err := c.ConcludeStepCommand(meta("conclude-failed"), "lending", run.StepResult{Aggregate: c.AggregateVersion(), Failure: "the answered report fails the Host's check"})
				if err != nil {
					t.Fatal(err)
				}
				commit(t, c, j, cmd)
				if v, _ := c.Task("lending"); v.State != run.TaskQueued {
					t.Fatal("failed Explore did not return to queued", v.State)
				}
			}
		})
	}
}

func stepPreparationWith(t *testing.T, c *run.Coordinator, inputs run.StepPreparation, id run.StepID, invocation string) run.Command {
	t.Helper()
	v, _ := c.Session("s")
	cmd, err := c.PrepareTaskStepCommand(meta("prepare-"+string(id)), run.StepPlan{Session: "s", ID: id, SessionBinding: v.Plan.Binding()}, run.Reservation{ID: invocation, Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Minute}, inputs, sourceFacts(inputs))
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

// A local review is renewed when a producer that the candidate used delivers
// again before the review passes (core rule D4); no rejection counts, and the
// next work step learns why its pass was not accepted.
func TestReviewIsRenewedWhenAUsedProducerDeliversAgain(t *testing.T) {
	t.Parallel()
	var notes []bool
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseWork:
			notes = append(notes, bytes.Contains(s.Packet.Context.Bytes, []byte("other Managers do not see it: lending, whose delivery your candidate used, delivered again after your step saw the aggregate; submit again")))
			return []action{{Tool: run.ToolSubmit, Args: `{"purpose":"local","scope":"complete","summary":"Uses lending.","assessments":[],"dependencies":["lending"]}`, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return nil // a failed review step: a fresh review follows
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{}\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "limited"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{ Title string }\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.c.Current(), h.sessions.answers)
	}
	// Each of lending's two deliveries renews the root's pending review.
	root := h.task(".")
	if len(root.Rejections) != 0 || strings.Count(strings.Join(admitted(h), ","), ". work") != 3 || root.Deliveries[0].Aggregate != 2 {
		t.Fatal("the renewed review counted a rejection or the root did not work again", root.Rejections, root.Deliveries, admitted(h))
	}
	if !slices.Equal(notes, []bool{false, true, true}) || root.Withheld != "" {
		t.Fatal("the work steps after a withheld pass did not see why", notes, root.Withheld)
	}
	if got := aggregateFile(t, h.engine, "out/library.go"); got != "package library\n\ntype Item struct{ Title string }\n" {
		t.Fatalf("limited and complete deliveries not composed in order: %q", got)
	}
}

// A pass the Host withholds leaves its recorded expectations standing while
// their inputs are unchanged: the resubmission's review skips the expectations
// step and assesses the same cases (core rules D4, T7).
func TestARenewedReviewKeepsItsRecordedExpectations(t *testing.T) {
	t.Parallel()
	var h *harness
	var seen []run.BlobRef
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseWork:
			return []action{{Tool: run.ToolSubmit, Args: `{"purpose":"local","scope":"complete","summary":"Uses lending.","assessments":[],"dependencies":["lending"]}`, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseReviewerExpectations:
			// Each recording differs, so a reused one is recognizable.
			args := fmt.Sprintf(`{"cases":[{"name":"item","expectation":"The library realizes Item (recording %d).","discriminates":"A missing Item type fails."}]}`, s.Attempt)
			return []action{{Tool: run.ToolRecordExpectations, Args: args, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseReviewerCandidate:
			seen = append(seen, h.task(".").Expectations)
			if s.Attempt == 0 {
				return nil // a failed review step: a fresh review follows
			}
			if s.Attempt == 1 {
				// An advisory finding of the withheld pass itself does not
				// change what its expectations rest on.
				return []action{{Tool: run.ToolSubmitReview, Args: `{"verdict":"pass","summary":"Fine.","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"findings":[{"severity":"advisory","definitions":["lending/Item"],"grounds":"Item could carry a title.","requestedEvidence":"A title field."}],"resolutions":[]}`, Want: run.ToolOK}}
			}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{}\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "limited"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{ Title string }\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h = newHarness(t, true, agent)
	runToEnd(t, h)
	order := strings.Join(admitted(h), ",")
	if h.c.Current().State != run.Verifying || strings.Count(order, ". work") != 3 {
		t.Fatal("run did not complete through two withheld passes", h.c.Current(), order)
	}
	// The failed review starts a fresh local review with new expectations, the
	// two withheld passes keep theirs, and the integration records its own.
	if strings.Count(order, ". reviewer-expectations") != 3 || len(seen) != 5 || seen[0] == seen[1] || seen[1] != seen[2] || seen[2] != seen[3] || seen[3] == seen[4] {
		t.Fatal("the renewed reviews did not keep the recorded expectations", order, seen)
	}
	restored, err := run.Restore(context.Background(), &readJournal{store: h.j}, "repo", run.RestoreLimits{PageSize: 16, Batches: 2000, Events: 4000, Bytes: 64 << 20})
	if err != nil || !reflect.DeepEqual(restored.Tasks(), h.c.Tasks()) {
		t.Fatal("restore does not rebuild the kept reviews", err)
	}
}

// A resubmission whose expectation inputs differ from the withheld pass's,
// here its scope, records fresh expectations (core rule T7).
func TestAChangedResubmissionRecordsFreshExpectations(t *testing.T) {
	t.Parallel()
	var h *harness
	type review struct {
		scope        string
		expectations run.BlobRef
	}
	var seen []review
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseWork:
			scope := "complete"
			if s.Attempt == 1 {
				scope = "limited"
			}
			return []action{{Tool: run.ToolSubmit, Args: `{"purpose":"local","scope":"` + scope + `","summary":"Uses lending.","assessments":[],"dependencies":["lending"]}`, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseReviewerExpectations:
			args := fmt.Sprintf(`{"cases":[{"name":"item","expectation":"The library realizes Item (recording %d).","discriminates":"A missing Item type fails."}]}`, s.Attempt)
			return []action{{Tool: run.ToolRecordExpectations, Args: args, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseReviewerCandidate:
			v := h.task(".")
			seen = append(seen, review{v.Submission.Scope, v.Expectations})
			if s.Attempt == 0 {
				return nil // a failed review step: a fresh review follows
			}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{}\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "limited"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{ Title string }\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h = newHarness(t, true, agent)
	runToEnd(t, h)
	// Three local submissions mean that two passes were withheld.
	if h.c.Current().State != run.Verifying || strings.Count(strings.Join(admitted(h), ","), ". work") < 3 {
		t.Fatal("run did not complete through withheld passes", h.c.Current(), admitted(h))
	}
	changed := 0
	for i := 1; i < len(seen); i++ {
		if seen[i].scope != seen[i-1].scope {
			changed++
			if seen[i].expectations == seen[i-1].expectations {
				t.Fatal("a resubmission with another scope reused the kept expectations", seen)
			}
		}
	}
	if changed == 0 {
		t.Fatal("no review saw a changed scope", seen)
	}
}

// A rejection keeps the recorded cases even when it resolves a finding that
// was open when they were recorded: the rework sees the cases, and the next
// review of the same scope assesses them without recording new ones (DEC-047).
func TestARejectionKeepsTheCasesItsReviewResolvedAgainst(t *testing.T) {
	t.Parallel()
	var advisory string
	var notes []bool
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{}\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "limited"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			notes = append(notes, strings.Contains(string(s.Packet.Context.Bytes), "Your last review assessed your candidate against these recorded cases"))
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{ Title string }\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			advisory = "finding-" + string(s.Packet.Step) + "-1"
			return []action{{Tool: run.ToolSubmitReview, Args: `{"verdict":"pass","summary":"Fine.","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"findings":[{"severity":"advisory","definitions":["lending/Item"],"grounds":"Item could carry a title.","requestedEvidence":"A title field."}],"resolutions":[]}`, Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 1:
			reject := `{"verdict":"changes_requested","summary":"Item is incomplete.","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"resolutions":[{"finding":"` + advisory + `","state":"resolved","grounds":"Item has a title.","evidence":[]}],"findings":[{"severity":"blocking","definitions":["lending/Item"],"grounds":"No test.","requestedEvidence":"A test."}]}`
			return []action{{Tool: run.ToolSubmitReview, Args: reject, Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 2:
			return []action{{Tool: run.ToolSubmitReview, Args: passArgs(openFinding(t, s)), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	order := strings.Join(admitted(h), ",")
	// The limited and the complete submission record their own cases; the
	// rework after the rejection keeps them.
	if h.c.Current().State != run.Verifying || strings.Count(order, "lending reviewer-expectations") != 2 || !slices.Equal(notes, []bool{false, true}) {
		t.Fatal("the rejection did not keep its recorded cases", h.c.Current(), notes, order)
	}
}

// A rejected candidate's next submission with another scope records fresh
// cases (core rule T7, DEC-047).
func TestARejectionThenAnotherScopeRecordsFreshCases(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Write: map[string]string{"out/library.go": "package library\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "limited"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return []action{{Tool: run.ToolSubmitReview, Args: rejectArgs, Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 1:
			return []action{{Tool: run.ToolSubmitReview, Args: passArgs(openFinding(t, s)), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if order := strings.Join(admitted(h), ","); h.c.Current().State != run.Verifying || strings.Count(order, "lending reviewer-expectations") != 2 {
		t.Fatal("the complete submission reused the limited submission's cases", h.c.Current(), order)
	}
}

// A failed step after a withheld pass leaves no new submission, so the step
// after it still learns why the pass was not accepted (core rules D4, T6).
func TestAWithheldNoteOutlivesAFailedStep(t *testing.T) {
	t.Parallel()
	var notes []bool
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseWork:
			notes = append(notes, bytes.Contains(s.Packet.Context.Bytes, []byte("lending, whose delivery your candidate used, delivered again")))
			if s.Attempt == 1 {
				return nil // a failed work step
			}
			return []action{{Tool: run.ToolSubmit, Args: `{"purpose":"local","scope":"complete","summary":"Uses lending.","assessments":[],"dependencies":["lending"]}`, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return nil // a failed review step: a fresh review follows
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{}\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "limited"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{ Title string }\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.c.Current(), h.sessions.answers)
	}
	if len(notes) < 3 || !notes[1] || !notes[2] {
		t.Fatal("the step after a failed step did not see the withheld note", notes)
	}
}

// When the budget no longer fits the next step, the run finishes Incomplete
// with its work retained (core rule T14).
func TestBudgetExhaustionFinishesTheRunIncomplete(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Phase == run.PhaseReviewerCandidate {
			return nil // every candidate review fails, so fresh reviews follow
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	r := h.c.Current()
	if r.State != run.Finished || r.Outcome != run.Incomplete || !strings.Contains(r.Reason, "budget") {
		t.Fatalf("run not finished Incomplete on budget: %+v", r)
	}
	if v := h.task("lending"); v.Own == (run.BlobRef{}) || v.State != run.TaskReviewing {
		t.Fatalf("work not retained: %+v", v)
	}
}
