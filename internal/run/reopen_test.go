package run_test

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// withCreationArea lets every Manager create files under out.
func withCreationArea(t *testing.T) func(*organization.SettingsFacts) {
	return func(f *organization.SettingsFacts) {
		p, _ := snapshot.NewPath("out")
		dir, err := snapshot.DirectoryAt(p)
		if err != nil {
			t.Fatal(err)
		}
		f.Defaults.CreationRoots.Value = []snapshot.DirectoryBoundary{dir}
		f.Ceilings.CreationRoots.Value = []snapshot.DirectoryBoundary{dir}
	}
}

// A restarted Host forms the same claims from the run's record: a reservation
// and a registry proposal made before the restart still hold for the rework
// after it (core rule S4).
func TestClaimsSurviveAnEngineRestart(t *testing.T) {
	t.Parallel()
	newFile := map[string]string{"out/library.go": "package library\n\ntype Item struct{}\n", "out/item_test.go": "package library\n"}
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{
				{Tool: run.ToolReservePath, Args: `{"path":"out/item_test.go","form":"test"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/item_test.go","form":"test","realizes":["lending/Item"],"why":"Tests Item."}`, Want: run.ToolOK},
				{Write: newFile, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return []action{{Tool: run.ToolSubmitReview, Args: rejectArgs, Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate:
			return []action{{Tool: run.ToolSubmitReview, Args: passArgs(openFinding(t, s)), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			return []action{{Write: map[string]string{"out/item_test.go": "package library\n\n// Tests follow.\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarnessWith(t, stepInputsWith(t, true, withCreationArea(t)), agent)
	h.advanceUntil(func(a run.Admission) bool {
		return a.Manager == "lending" && a.Phase == run.PhaseWork && h.task("lending").Rejections["local"] == 1
	})
	if len(h.c.Claims("lending")) != 2 {
		t.Fatal("the claims were not recorded", h.c.Claims("lending"))
	}
	h.engine = h.newEngine(h.c) // the Host restarts before the rework
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("the run did not complete after the restart", h.c.Current(), h.sessions.answers)
	}
	record, ok := h.engine.Aggregate().Registry().Lookup(mustPath(t, "out/item_test.go"))
	if !ok || record.Artifact().Why() != "Tests Item." {
		t.Fatal("the registry proposal was lost across the restart")
	}
}

// An integration review addresses a finding about a child's artifacts to that
// child: the done child is reopened, the parent waits for it, and the
// child's next local review must resolve the finding (core rules D5, T11).
func TestIntegrationFindingReopensTheChild(t *testing.T) {
	t.Parallel()
	childFinding := `{"verdict":"changes_requested","summary":"Item lacks a title.","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"resolutions":[],"findings":[{"to":"lending","severity":"blocking","definitions":["lending/Item"],"grounds":"Item has no title.","requestedEvidence":"A title field."}]}`
	var finding string
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 1: // the integration review
			finding = fmt.Sprintf("finding-%s-1", s.Packet.Step)
			return []action{
				{Tool: run.ToolSubmitReview, Args: strings.Replace(childFinding, `"to":"lending"`, `"to":"nowhere"`, 1), Want: run.ToolRefused},
				// A finding to a child requests its correction.
				{Tool: run.ToolSubmitReview, Args: strings.Replace(childFinding, `"verdict":"changes_requested"`, `"verdict":"needs_decision","question":"Which title?"`, 1), Want: run.ToolRefused},
				{Tool: run.ToolSubmitReview, Args: childFinding, Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 1:
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("Item has no title.")) {
				t.Error("the reopened child does not see the finding")
			}
			// The finding changed what the child's cases rest on, so a
			// reviewer records fresh ones before the rework (DEC-048, DEC-049).
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("against these recorded review cases")) {
				t.Error("the reopened child does not see its fresh cases")
			}
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{ Title string }\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 1:
			return []action{
				{Tool: run.ToolSubmitReview, Args: passArgs(), Want: run.ToolRefused},
				{Tool: run.ToolSubmitReview, Args: passArgs(finding), Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("the run did not complete", h.c.Current(), h.sessions.answers)
	}
	order := admitted(h)
	if strings.Count(strings.Join(order, ","), "lending work") != 2 || strings.Count(strings.Join(order, ","), ". integrate") != 2 {
		t.Fatal("the child was not reopened or the parent did not integrate again", order)
	}
	// The accepted second integration restarts the count (DEC-050).
	if root := h.task("."); root.Rejections["integration"] != 0 {
		t.Fatal("the accepted integration did not restart the rejection count", root.Rejections)
	}
	// The second integration review keeps the cases of the rejected one
	// (DEC-047); the root records cases for its local and its integration
	// review only.
	if n := strings.Count(strings.Join(order, ","), ". reviewer-expectations"); n != 2 {
		t.Fatal("the integration review after the rejection recorded new cases", n, order)
	}
	// The parent's finding changes what the child's cases rest on, so the
	// child's fresh cases are recorded before its rework (DEC-048, DEC-049).
	var lending []string
	for _, step := range order {
		if strings.HasPrefix(step, "lending ") {
			lending = append(lending, strings.TrimPrefix(step, "lending "))
		}
	}
	if !slices.Equal(lending, []string{"explore", "reviewer-expectations", "work", "reviewer-candidate", "reviewer-expectations", "work", "reviewer-candidate"}) {
		t.Fatal("the reopened child's cases were not recorded fresh before its rework", lending)
	}
	issues := h.c.Issues("lending")
	if !slices.ContainsFunc(issues, func(i run.Issue) bool { return i.ID == finding && i.State == "resolved" && i.Purpose == "local" }) {
		t.Fatalf("the child's local review did not resolve the finding: %+v", issues)
	}
	if got := aggregateFile(t, h.engine, "out/library.go"); got != "package library\n\ntype Item struct{ Title string }\n" {
		t.Fatalf("the correction was not composed: %q", got)
	}
}

// A done Manager whose answer shows that its own file needs a correction makes
// it in its reply step: the task returns to work, the change goes through its
// review, and the parent integrates the corrected delivery (core rule T11).
func TestAReplyStepCorrectsADoneManagersFile(t *testing.T) {
	t.Parallel()
	var h *harness
	var asked string
	corrected := "package library\n\ntype Item struct{ Title string }\n"
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending", "Item needs a title; can you add it?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(asked), Want: run.ToolOK},
			}
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && h.task("lending").State != run.TaskDone:
			return []action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"children"}],"reason":"Lending corrects Item."}`, Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReply:
			if h.task("lending").State != run.TaskDone {
				t.Error("the reply step is not the done Manager's")
			}
			return []action{{Write: map[string]string{"out/library.go": corrected}, Tool: run.ToolSend, Args: sendArgs("reply", asked, "Added the title."), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 1:
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("Title string")) {
				t.Error("the returned task does not see its corrected file")
			}
			return []action{{Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h = newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("the run did not complete", h.c.Current(), h.sessions.answers)
	}
	order := strings.Join(admitted(h), ",")
	if !strings.Contains(order, "lending reply") || strings.Count(order, "lending work") != 2 || strings.Count(order, "lending reviewer-candidate") != 2 {
		t.Fatal("the corrected file did not return lending to work and review", order)
	}
	// A reply that changes no expectation input keeps the recorded cases
	// (DEC-048, DEC-049).
	if n := strings.Count(order, "lending reviewer-expectations"); n != 1 {
		t.Fatal("the corrected file's review recorded new cases", n, order)
	}
	if got := aggregateFile(t, h.engine, "out/library.go"); got != corrected {
		t.Fatalf("the correction was not composed: %q", got)
	}
}

// A done grandchild that corrects its file in a reply step returns its done
// parent to waiting for its children: the parent integrates the correction
// again before the root does (core rule T11).
func TestAReplyCorrectionReopensTheDoneAncestors(t *testing.T) {
	t.Parallel()
	var h *harness
	var asked string
	corrected := "package reminders\n\nconst Days = 14\n"
	var lendingAfter run.TaskState
	agent := func(s agentStep) []action {
		if order := admitted(h); lendingAfter == "" && len(order) > 1 && order[len(order)-2] == "lending/reminders reply" {
			lendingAfter = h.task("lending").State
		}
		switch {
		case s.Manager == "lending/reminders" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{
				{Tool: run.ToolReservePath, Args: `{"path":"out/reminders.go","form":"code"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/reminders.go","form":"code","realizes":["lending/reminders/Item"],"why":"Realizes reminders."}`, Want: run.ToolOK},
				{Write: map[string]string{"out/reminders.go": "package reminders\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending/reminders", "Reminders need the loan period; can you add it?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(asked), Want: run.ToolOK},
			}
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && h.task("lending").State != run.TaskDone:
			return []action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"children"}],"reason":"Reminders correct their file."}`, Want: run.ToolOK}}
		case s.Manager == "lending/reminders" && s.Phase == run.PhaseReply:
			return []action{{Write: map[string]string{"out/reminders.go": corrected}, Tool: run.ToolSend, Args: sendArgs("reply", asked, "Added the period."), Want: run.ToolOK}}
		case s.Manager == "lending/reminders" && s.Phase == run.PhaseWork && s.Attempt == 1:
			return []action{{Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt >= 1:
			return []action{{Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	inputs := stepInputsWith(t, false, withCreationArea(t))
	inputs.Binding.Policy.Assignment.Steps, inputs.Binding.Policy.Assignment.SessionStarts = 64, 64
	h = newHarnessWith(t, inputs, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("the run did not complete", h.c.Current(), admitted(h))
	}
	if lendingAfter != run.TaskBlocked {
		t.Fatal("the done parent did not wait for its corrected child", lendingAfter)
	}
	if order := strings.Join(admitted(h), ","); strings.Count(order, "lending integrate") != 2 {
		t.Fatal("the parent did not integrate the correction again", order)
	}
	if got := aggregateFile(t, h.engine, "out/reminders.go"); got != corrected {
		t.Fatalf("the correction was not composed: %q", got)
	}
}

// A parent that asks a done child and submits its integration in the same step
// does not get that integration accepted while the child corrects its file:
// the pass is withheld, and the parent integrates the corrected delivery once
// the child is done again (core rules T9, T11).
func TestAnIntegrationWaitsForAChildThatCorrectsItself(t *testing.T) {
	t.Parallel()
	var h *harness
	var asked string
	corrected := "package library\n\ntype Item struct{ Title string }\n"
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending", "Item needs a title; can you add it?"), Want: run.ToolOK},
				{Tool: run.ToolSubmit, Args: submitArgs("integration", "complete"), Want: run.ToolOK},
			}
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && h.task("lending").State != run.TaskDone:
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("other Managers do not see it: an integration needs every child done, and lending was not done; wait for your children")) {
				t.Error("the integration step after a withheld pass does not see why")
			}
			return []action{
				{Tool: run.ToolSubmit, Args: submitArgs("integration", "complete"), Want: run.ToolRefused},
				{Tool: run.ToolWait, Args: `{"on":[{"kind":"children"}],"reason":"Lending corrects Item."}`, Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseReply:
			return []action{{Write: map[string]string{"out/library.go": corrected}, Tool: run.ToolSend, Args: sendArgs("reply", asked, "Added the title."), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 1:
			return []action{{Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h = newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || h.task("lending").State != run.TaskDone {
		t.Fatal("the run did not complete with the corrected child", h.c.Current(), h.task("lending").State, h.sessions.answers)
	}
	if got := aggregateFile(t, h.engine, "out/library.go"); got != corrected {
		t.Fatalf("the integration was accepted without the correction: %q", got)
	}
	if h.task(".").Withheld != "" {
		t.Fatal("the withheld note outlived the step that saw it", h.task(".").Withheld)
	}
}

// A reply step's change the Host refuses, here a new file the reply phase
// cannot reserve, still returns the task to work, whose step sees the refusal
// and makes the change properly (core rules S4, T11).
func TestARefusedReplyCorrectionReachesTheWorkStep(t *testing.T) {
	t.Parallel()
	var h *harness
	var asked string
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending", "Add a test file?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(asked), Want: run.ToolOK},
			}
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && h.task("lending").State != run.TaskDone:
			return []action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"children"}],"reason":"Lending adds a test."}`, Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseReply:
			return []action{{Write: map[string]string{"out/item_test.go": "package library\n"}, Tool: run.ToolSend, Args: sendArgs("reply", asked, "Adding the test."), Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 1:
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("outside its owner's claims")) {
				t.Error("the work step does not see the refused correction")
			}
			return []action{
				{Tool: run.ToolReservePath, Args: `{"path":"out/item_test.go","form":"test"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/item_test.go","form":"test","realizes":["lending/Item"],"why":"Tests Item."}`, Want: run.ToolOK},
				{Write: map[string]string{"out/item_test.go": "package library\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	h = newHarnessWith(t, stepInputsWith(t, true, withCreationArea(t)), agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("the run did not complete", h.c.Current(), h.sessions.answers)
	}
	if got := aggregateFile(t, h.engine, "out/item_test.go"); got != "package library\n" {
		t.Fatalf("the corrected file was not composed: %q", got)
	}
}

// A task blocked on the owner's decision about its frozen submission answers
// read-only: a change fails the step, the task stays blocked, and the run waits
// for the owner (core rules T8, T11).
func TestAReplyOfATaskAwaitingADecisionIsReadOnly(t *testing.T) {
	t.Parallel()
	var h *harness
	var asked string
	decision := `{"verdict":"needs_decision","summary":"Unclear.","question":"Which title?","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"findings":[],"resolutions":[]}`
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return []action{{Tool: run.ToolSubmitReview, Args: decision, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseWork && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending", "Item needs a title; can you add it?"), Want: run.ToolOK},
				{Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseReply:
			return []action{{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{ Title string }\n"}, Tool: run.ToolSend, Args: sendArgs("reply", asked, "Added the title."), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h = newHarness(t, true, agent)
	runToEnd(t, h)
	if !slices.Contains(admitted(h), "lending reply") {
		t.Fatal("the blocked task did not answer", admitted(h))
	}
	if h.c.Current().State != run.Waiting || h.task("lending").State != run.TaskBlocked {
		t.Fatal("the task awaiting a decision left its frozen candidate", h.c.Current(), h.task("lending").State)
	}
}

// A question that a task's work step saw but left unanswered gets a reply step,
// and a reply that changes no file keeps the task blocked, even when it holds
// an unaccepted registry proposal from an earlier step (core rules S4, T10).
func TestAReplyWithoutChangesKeepsABlockedTaskBlocked(t *testing.T) {
	t.Parallel()
	var h *harness
	var own, other string
	var after run.TaskState
	agent := func(s agentStep) []action {
		// The step after lending's reply observes lending's state.
		if order := admitted(h); after == "" && len(order) > 1 && order[len(order)-2] == "lending reply" {
			after = h.task("lending").State
		}
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			own = questionID(s, 1)
			return []action{
				{Tool: run.ToolReservePath, Args: `{"path":"out/item_test.go","form":"test"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/item_test.go","form":"test","realizes":["lending/Item"],"why":"Tests Item."}`, Want: run.ToolOK},
				{Write: map[string]string{"out/library.go": "package library\n\ntype Item struct{}\n", "out/item_test.go": "package library\n"}, Tool: run.ToolSend, Args: sendArgs("question", "parent", "Title or name?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(own), Want: run.ToolOK},
			}
		case s.Manager == "." && s.Phase == run.PhaseWork && s.Attempt == 0:
			other = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending", "Which file holds Item?"), Want: run.ToolOK},
				{Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseReply:
			return []action{{Tool: run.ToolSend, Args: sendArgs("reply", other, "out/library.go."), Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && h.task("lending").State != run.TaskDone:
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("reply", own, "Title."), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: `{"on":[{"kind":"children"}],"reason":"Lending realizes Item."}`, Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	h = newHarnessWith(t, stepInputsWith(t, true, withCreationArea(t)), agent)
	runToEnd(t, h)
	if after != run.TaskBlocked {
		t.Fatal("a reply without changes returned the blocked task to work", after, admitted(h))
	}
}

// A question that reaches a task awaiting the owner's decision on its frozen
// submission does not wake it: the task answers read-only, keeps its candidate
// and the run waits for the owner (core rules T8, T10).
func TestAQuestionLeavesATaskAwaitingADecisionBlocked(t *testing.T) {
	t.Parallel()
	var h *harness
	var asked string
	decision := `{"verdict":"needs_decision","summary":"Unclear.","question":"Which title?","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"findings":[],"resolutions":[]}`
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return []action{{Tool: run.ToolSubmitReview, Args: decision, Want: run.ToolOK}}
		case s.Manager == "lending/reminders" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"delivery","ref":"empty"}],"reason":"Reminders build on empty."}`, Want: run.ToolOK}}
		case s.Manager == "lending/reminders" && s.Phase == run.PhaseWork && s.Attempt == 1:
			if h.task("lending").Submission.Proposal == (run.BlobRef{}) {
				t.Error("lending is not yet awaiting the owner's decision")
			}
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending", "Which period applies?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(asked), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseReply:
			return []action{{Tool: run.ToolSend, Args: sendArgs("reply", asked, "Fourteen days."), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	inputs := stepInputs(t, false)
	inputs.Binding.Policy.Assignment.Steps, inputs.Binding.Policy.Assignment.SessionStarts = 64, 64
	h = newHarnessWith(t, inputs, agent)
	runToEnd(t, h)
	lending := h.task("lending")
	if !slices.Contains(admitted(h), "lending reply") || h.c.Current().State != run.Waiting || lending.State != run.TaskBlocked || lending.Submission.Proposal == (run.BlobRef{}) {
		t.Fatal("the question woke the task awaiting a decision", admitted(h), h.c.Current(), lending.State)
	}
}

// A question that reaches a blocked task makes it ready: its next step is a
// work step that sees the question, answers it and can act on it, here by
// waiting again for the asker's delivery (core rule T10).
func TestAQuestionMakesABlockedTaskReady(t *testing.T) {
	t.Parallel()
	var h *harness
	var asked string
	var answered bool
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"delivery","ref":"lending/reminders"}],"reason":"Lending composes the reminders."}`, Want: run.ToolOK}}
		case s.Manager == "lending/reminders" && s.Phase == run.PhaseWork && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending", "Which loan period applies?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(asked), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 1:
			answered = bytes.Contains(s.Packet.Context.Bytes, []byte("Which loan period applies?"))
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("reply", asked, "Fourteen days."), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: `{"on":[{"kind":"delivery","ref":"lending/reminders"}],"reason":"Lending composes the reminders."}`, Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	inputs := stepInputsWith(t, false, withCreationArea(t))
	inputs.Binding.Policy.Assignment.Steps, inputs.Binding.Policy.Assignment.SessionStarts = 64, 64
	h = newHarnessWith(t, inputs, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("the run did not complete", h.c.Current(), admitted(h))
	}
	if !answered || slices.Contains(admitted(h), "lending reply") {
		t.Fatal("the blocked task answered in a reply step instead of its next work step", answered, admitted(h))
	}
}

// A child's open owner question does not block its parent's local delivery,
// which does not contain the child's work; the run waits for the owner
// (core rules I3, W1).
func TestAChildsOpenQuestionLeavesTheParentsLocalReview(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return []action{{Tool: run.ToolSubmitReview, Args: `{"verdict":"needs_decision","summary":"Open.","question":"Which format?","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"findings":[],"resolutions":[]}`, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return []action{{Tool: run.ToolSubmitReview, Args: rejectArgs, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseReviewerCandidate && strings.Contains(string(s.Packet.Context.Bytes), "finding-step-"):
			return []action{{Tool: run.ToolSubmitReview, Args: passArgs(openFinding(t, s)), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	order := strings.Join(admitted(h), ",")
	if h.c.Current().State != run.Waiting || strings.Count(order, ". work") != 2 {
		t.Fatal("the parent's local review was withheld for the child's question", h.c.Current(), order)
	}
}

// A finding to a grandchild reopens it, and the chain above it integrates
// again (core rule T11).
func TestAGrandchildFindingReintegratesTheChain(t *testing.T) {
	t.Parallel()
	grand := `{"verdict":"changes_requested","summary":"Reminder lacks a title.","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"resolutions":[],"findings":[{"to":"lending/reminders","severity":"blocking","definitions":["lending/reminders/Item"],"grounds":"No title.","requestedEvidence":"A title."}]}`
	var finding string
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 1:
			finding = fmt.Sprintf("finding-%s-1", s.Packet.Step)
			return []action{{Tool: run.ToolSubmitReview, Args: grand, Want: run.ToolOK}}
		case s.Manager == "lending/reminders" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 1:
			return []action{{Tool: run.ToolSubmitReview, Args: passArgs(finding), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	inputs := stepInputs(t, false)
	inputs.Binding.Policy.Assignment.Steps, inputs.Binding.Policy.Assignment.SessionStarts = 64, 64
	h := newHarnessWith(t, inputs, agent)
	runToEnd(t, h)
	order := strings.Join(admitted(h), ",")
	if h.c.Current().State != run.Verifying || strings.Count(order, "lending integrate") != 2 || strings.Count(order, ". integrate") != 2 {
		t.Fatal("the grandchild's correction did not re-integrate the chain", h.c.Current(), order)
	}
}

// Findings to a child and to its own child reopen both, whatever their
// order; the child works locally again before it integrates (core rule T11).
func TestFindingsToAChildAndItsChildReopenBoth(t *testing.T) {
	t.Parallel()
	both := `{"verdict":"changes_requested","summary":"Both lack a title.","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"resolutions":[],"findings":[` +
		`{"to":"lending/reminders","severity":"blocking","definitions":["lending/reminders/Item"],"grounds":"No title.","requestedEvidence":"A title."},` +
		`{"to":"lending","severity":"blocking","definitions":["lending/Item"],"grounds":"No title either.","requestedEvidence":"A title."}]}`
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 1:
			return []action{{Tool: run.ToolSubmitReview, Args: both, Want: run.ToolOK}}
		case (s.Manager == "lending/reminders" || s.Manager == "lending") && s.Phase == run.PhaseReviewerCandidate && strings.Contains(string(s.Packet.Context.Bytes), "finding-step-"):
			return []action{{Tool: run.ToolSubmitReview, Args: passArgs(openFinding(t, s)), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	inputs := stepInputs(t, false)
	inputs.Binding.Policy.Assignment.Steps, inputs.Binding.Policy.Assignment.SessionStarts = 64, 64
	h := newHarnessWith(t, inputs, agent)
	h.advanceUntil(func(run.Admission) bool { return h.task(".").Rejections["integration"] == 1 })
	if l := h.task("lending"); l.State != run.TaskReady || l.LocalDone {
		t.Fatalf("the child was not reopened for local work: %+v", l)
	}
	runToEnd(t, h)
	order := strings.Join(admitted(h), ",")
	if h.c.Current().State != run.Verifying || strings.Count(order, "lending work") != 2 {
		t.Fatal("the reopened child did not work again", h.c.Current(), order)
	}
}

// A reservation refused because another Manager reserved the path names that
// Manager, so the refused Manager can wait for its delivery or ask it.
func TestARefusedReservationNamesItsHolder(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{
				{Tool: run.ToolReservePath, Args: `{"path":"out/shared.go","form":"code"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/shared.go","form":"code","realizes":["lending/Item"],"why":"Shared code."}`, Want: run.ToolOK},
				{Write: map[string]string{"out/shared.go": "package library\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return append([]action{{Tool: run.ToolReservePath, Args: `{"path":"out/shared.go","form":"code"}`, Want: run.ToolRefused}}, happyAgent(s)...)
		}
		return happyAgent(s)
	}
	h := newHarnessWith(t, stepInputsWith(t, true, withCreationArea(t)), agent)
	runToEnd(t, h)
	if !slices.ContainsFunc(h.sessions.answers, func(a string) bool {
		return strings.HasPrefix(a, "lending work reserve_path refused") && strings.Contains(a, "The Manager . reserved out/shared.go") && strings.Contains(a, `"ref":"."`)
	}) {
		t.Fatal("the refusal does not name the holder", h.sessions.answers)
	}
}

// A path another Manager's accepted delivery registers is named with its
// owner and is already in the workspace; no wait is suggested.
func TestARefusedReservationOfARegisteredPathNamesItsOwner(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"delivery","ref":"."}],"reason":"Needs the shared file."}`, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{
				{Tool: run.ToolReservePath, Args: `{"path":"out/shared.go","form":"code"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/shared.go","form":"code","realizes":["lending/Item"],"why":"Shared code."}`, Want: run.ToolOK},
				{Write: map[string]string{"out/shared.go": "package library\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 1:
			return append([]action{{Tool: run.ToolReservePath, Args: `{"path":"out/shared.go","form":"code"}`, Want: run.ToolRefused}}, happyAgent(s)...)
		}
		return happyAgent(s)
	}
	h := newHarnessWith(t, stepInputsWith(t, true, withCreationArea(t)), agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || !slices.ContainsFunc(h.sessions.answers, func(a string) bool {
		return strings.HasPrefix(a, "lending work reserve_path refused") && strings.Contains(a, "out/shared.go is registered to the Manager . and already in your workspace") && !strings.Contains(a, "Wait for it")
	}) {
		t.Fatal("the refusal does not name the owner of the registered path", h.c.Current(), h.sessions.answers)
	}
}

// A wait that only the waiting Manager itself could end is refused: a child
// cannot wait for the delivery of a parent that waits for its children
// (core rule T10).
func TestAWaitThatClosesACycleIsRefused(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"children"}],"reason":"Integrates its child."}`, Want: run.ToolOK}}
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			return append([]action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"delivery","ref":"."}],"reason":"Wants the parent's code."}`, Want: run.ToolRefused}}, happyAgent(s)...)
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if !slices.ContainsFunc(h.sessions.answers, func(a string) bool {
		return strings.HasPrefix(a, "lending work wait refused") && strings.Contains(a, ". waits for you")
	}) {
		t.Fatal("the cyclic wait was not refused with its reason", h.sessions.answers)
	}
	if h.c.Current().State != run.Verifying {
		t.Fatal("the run did not complete after the refusal", h.c.Current())
	}
}

// Reserving one's own registered file is refused with the advice to change it
// directly.
func TestReservingOnesOwnFileSaysToChangeItDirectly(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0 {
			return append([]action{{Tool: run.ToolReservePath, Args: `{"path":"out/library.go","form":"code"}`, Want: run.ToolRefused}}, happyAgent(s)...)
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if !slices.ContainsFunc(h.sessions.answers, func(a string) bool {
		return strings.HasPrefix(a, "lending work reserve_path refused") && strings.Contains(a, "out/library.go is already yours: change it directly")
	}) {
		t.Fatal("the refusal does not say the file is the Manager's own", h.sessions.answers)
	}
}

// A registration must carry the form of its reservation; reserving the path
// again changes that form. In real attempt D3 a mismatch passed both calls and
// failed the whole step at capture.
func TestARegistrationCarriesItsReservationsForm(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0 {
			return []action{
				{Tool: run.ToolReservePath, Args: `{"path":"out/item_test.go","form":"code"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/item_test.go","form":"test","realizes":["lending/Item"],"why":"Tests Item."}`, Want: run.ToolRefused},
				{Tool: run.ToolReservePath, Args: `{"path":"out/item_test.go","form":"test"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/item_test.go","form":"test","realizes":["lending/Item"],"why":"Tests Item."}`, Want: run.ToolOK},
				{Write: map[string]string{"out/item_test.go": "package library\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	h := newHarnessWith(t, stepInputsWith(t, true, withCreationArea(t)), agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || !slices.ContainsFunc(h.sessions.answers, func(a string) bool {
		return strings.HasPrefix(a, "lending work register refused") && strings.Contains(a, "out/item_test.go is reserved with form code")
	}) {
		t.Fatal("the form mismatch was not refused at the call", h.c.Current(), h.sessions.answers)
	}
	record, ok := h.engine.Aggregate().Registry().Lookup(mustPath(t, "out/item_test.go"))
	if !ok || record.Artifact().Form() != "test" {
		t.Fatal("the file was not delivered with its corrected form")
	}
}

// Reserving a registered new file again with another form leaves the earlier
// record's form behind; the submission is refused until the record matches.
func TestASubmissionWaitsForMatchingForms(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0 {
			return []action{
				{Tool: run.ToolReservePath, Args: `{"path":"out/item_test.go","form":"code"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/item_test.go","form":"code","realizes":["lending/Item"],"why":"Tests Item."}`, Want: run.ToolOK},
				{Tool: run.ToolReservePath, Args: `{"path":"out/item_test.go","form":"test"}`, Want: run.ToolOK},
				{Write: map[string]string{"out/item_test.go": "package library\n"}, Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolRefused},
				{Tool: run.ToolRegister, Args: `{"path":"out/item_test.go","form":"test","realizes":["lending/Item"],"why":"Tests Item."}`, Want: run.ToolOK},
				{Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	h := newHarnessWith(t, stepInputsWith(t, true, withCreationArea(t)), agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || !slices.ContainsFunc(h.sessions.answers, func(a string) bool {
		return strings.HasPrefix(a, "lending work submit refused") && strings.Contains(a, "out/item_test.go is reserved with form test but registered with form code")
	}) {
		t.Fatal("the mismatching submission was not refused", h.c.Current(), h.sessions.answers)
	}
	if record, ok := h.engine.Aggregate().Registry().Lookup(mustPath(t, "out/item_test.go")); !ok || record.Artifact().Form() != "test" {
		t.Fatal("the file was not delivered with its corrected form")
	}
}

// A Manager may register its own existing file with another form; only new
// files follow their reservation.
func TestAnExistingOwnFileMayChangeItsForm(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0 {
			return append([]action{{Tool: run.ToolRegister, Args: `{"path":"out/library.go","form":"doc","realizes":["lending/Item"],"why":"Documents Item."}`, Want: run.ToolOK}}, happyAgent(s)...)
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if record, ok := h.engine.Aggregate().Registry().Lookup(mustPath(t, "out/library.go")); h.c.Current().State != run.Verifying || !ok || record.Artifact().Form() != "doc" {
		t.Fatal("the own file's form did not change", h.c.Current(), h.sessions.answers)
	}
}

// A wait that ends a writing step is refused on a form mismatch too: its
// capture would fail the same way.
func TestAWaitWaitsForMatchingForms(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0 {
			return []action{
				{Tool: run.ToolReservePath, Args: `{"path":"out/item_test.go","form":"code"}`, Want: run.ToolOK},
				{Tool: run.ToolRegister, Args: `{"path":"out/item_test.go","form":"code","realizes":["lending/Item"],"why":"Tests Item."}`, Want: run.ToolOK},
				{Tool: run.ToolReservePath, Args: `{"path":"out/item_test.go","form":"test"}`, Want: run.ToolOK},
				{Write: map[string]string{"out/item_test.go": "package library\n"}, Tool: run.ToolWait, Args: `{"on":[{"kind":"delivery","ref":"."}],"reason":"Needs the root."}`, Want: run.ToolRefused},
				{Tool: run.ToolRegister, Args: `{"path":"out/item_test.go","form":"test","realizes":["lending/Item"],"why":"Tests Item."}`, Want: run.ToolOK},
				{Tool: run.ToolSubmit, Args: submitArgs("local", "complete"), Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	h := newHarnessWith(t, stepInputsWith(t, true, withCreationArea(t)), agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || !slices.ContainsFunc(h.sessions.answers, func(a string) bool {
		return strings.HasPrefix(a, "lending work wait refused") && strings.Contains(a, "is reserved with form test but registered with form code")
	}) {
		t.Fatal("the mismatching wait was not refused", h.c.Current(), h.sessions.answers)
	}
}
