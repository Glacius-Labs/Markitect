package run_test

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func questionID(s agentStep, n int) string { return fmt.Sprintf("question-%s-%d", s.Packet.Step, n) }

func sendArgs(kind, to, body string) string {
	return fmt.Sprintf(`{"kind":%q,"to":%q,"body":%q}`, kind, to, body)
}

func waitReplyArgs(id string) string {
	return fmt.Sprintf(`{"on":[{"kind":"reply","ref":%q}],"reason":"Needs the answer."}`, id)
}

// A Manager asks its parent and waits; the question makes the parent, blocked
// on its children, ready, and its next step answers and waits again; the reply
// wakes the asker, whose next step sees it (core rules M1, M3, M5, T10).
func TestQuestionToParentIsAnsweredInItsNextStep(t *testing.T) {
	t.Parallel()
	var h *harness
	var asked string
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "parent", "Should Item carry a title?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(asked), Want: run.ToolOK},
			}
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && h.task("lending").State != run.TaskDone:
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("Should Item carry a title?")) {
				t.Error("the parent's step does not show the question")
			}
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("reply", asked, "Yes, a title."), Want: run.ToolOK},
				{Tool: run.ToolSend, Args: sendArgs("reply", asked, "Twice."), Want: run.ToolRefused},
				{Tool: run.ToolWait, Args: `{"on":[{"kind":"children"}],"reason":"Lending realizes Item."}`, Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("Yes, a title.")) {
				t.Error("the asker's next step does not show the reply")
			}
		}
		return happyAgent(s)
	}
	h = newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.c.Current(), h.sessions.answers)
	}
	if order := admitted(h); slices.Contains(order, ". reply") {
		t.Fatal("the blocked parent answered in a reply step", order)
	}
	var question run.Issue
	for _, issue := range h.c.Issues("lending") {
		if issue.ID == asked {
			question = issue
		}
	}
	if question.Kind != "question" || question.To != "." || question.State != "resolved" || question.Answer == "" {
		t.Fatalf("question not answered: %+v", question)
	}
	messages := h.c.Messages()
	if len(messages) != 2 || messages[1].Kind != "reply" || messages[1].To != "lending" || messages[1].About != asked {
		t.Fatalf("messages not recorded: %+v", messages)
	}
}

// A question to the owner moves the run to Waiting; the owner's answer wakes
// the asker, and the run continues (core rules M4, W1, W2).
func TestOwnerQuestionWaitsAndTheAnswerContinuesTheRun(t *testing.T) {
	t.Parallel()
	var asked string
	rootSaw := false
	agent := func(s agentStep) []action {
		if s.Manager == "." && s.Phase == run.PhaseIntegrate {
			rootSaw = bytes.Contains(s.Packet.Context.Bytes, []byte("No, keep the name."))
		}
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "owner", "May Item be renamed?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: fmt.Sprintf(`{"on":[{"kind":"owner","ref":%q}],"reason":"Owner decision."}`, asked), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("No, keep the name.")) {
				t.Error("the asker does not see the owner's answer")
			}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if r := h.c.Current(); r.State != run.Waiting || r.NextActor != "owner" || !strings.Contains(r.Reason, asked) {
		t.Fatal("run does not wait for the owner", r)
	}
	if qs := h.c.OwnerQuestions(); len(qs) != 1 || qs[0].ID != asked {
		t.Fatal("owner question not listed", qs)
	}
	answer, err := h.c.AnswerCommand(meta("answer"), asked, "No, keep the name.")
	if err != nil {
		t.Fatal(err)
	}
	commit(t, h.c, h.j, answer)
	resume := command("resume", run.ResumeRun)
	resume.Run = h.c.Current().ID
	commit(t, h.c, h.j, resume)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not continue after the answer", h.c.Current())
	}
	// The owner's decision is an input of lending's cases, so they are recorded
	// again before its next work step (DEC-049).
	if n := strings.Count(strings.Join(admitted(h), ","), "lending reviewer-expectations"); n != 2 {
		t.Fatal("the owner's answer did not renew lending's cases", n, admitted(h))
	}
	// The answer binds the whole run, so the root sees it too, but it renews
	// only the asking task's cases (DEC-053).
	if !rootSaw || strings.Count(strings.Join(admitted(h), ","), ". reviewer-expectations") != 2 {
		t.Fatal("the answer did not reach the root or renewed its cases", rootSaw, admitted(h))
	}
}

// The owner's decisions bound with the run reach every step from the start,
// and a restored run keeps them (DEC-053).
func TestBoundOwnerDecisionsReachEveryStep(t *testing.T) {
	t.Parallel()
	const decision = "Unknown argument fields return invalidRequest."
	missed := []string{}
	agent := func(s agentStep) []action {
		if !bytes.Contains(s.Packet.Context.Bytes, []byte("The owner decided: "+decision)) {
			missed = append(missed, s.Manager+" "+string(s.Phase))
		}
		return happyAgent(s)
	}
	inputs := stepInputs(t, true)
	inputs.Binding.OwnerDecisions = []string{decision}
	h := newHarnessWith(t, inputs, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(missed) != 0 {
		t.Fatal("a step did not see the bound owner decision", h.c.Current(), missed)
	}
}

// A Host restarted from the journal keeps the run's bound decisions: the steps
// after the restore still see them (DEC-053).
func TestBoundOwnerDecisionsSurviveARestore(t *testing.T) {
	t.Parallel()
	const decision = "Unknown argument fields return invalidRequest."
	missed := []string{}
	agent := func(s agentStep) []action {
		if !bytes.Contains(s.Packet.Context.Bytes, []byte("The owner decided: "+decision)) {
			missed = append(missed, s.Manager+" "+string(s.Phase))
		}
		return happyAgent(s)
	}
	inputs := stepInputs(t, true)
	inputs.Binding.OwnerDecisions = []string{decision}
	h := newHarnessWith(t, inputs, agent)
	h.advanceUntil(func(a run.Admission) bool { return a.Manager == "lending" && a.Phase == run.PhaseWork })
	restored, err := run.Restore(context.Background(), &readJournal{store: h.j}, "repo", run.RestoreLimits{PageSize: 16, Batches: 2000, Events: 4000, Bytes: 64 << 20})
	if err != nil {
		t.Fatal(err)
	}
	h.c = restored
	h.engine = h.newEngine(restored)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(missed) != 0 {
		t.Fatal("a step after the restore did not see the bound decision", h.c.Current(), missed)
	}
}

// An answer to one task's question reaches a sibling's later steps but does
// not renew the sibling's standing cases (DEC-053).
func TestAnAnswerKeepsASiblingsStandingCases(t *testing.T) {
	t.Parallel()
	var asked string
	rootSaw := false
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseWork && s.Attempt == 0:
			return []action{{Tool: run.ToolWait, Args: `{"on":[{"kind":"children"}],"reason":"Lending first."}`, Want: run.ToolOK}}
		case s.Manager == "." && s.Phase == run.PhaseWork:
			rootSaw = bytes.Contains(s.Packet.Context.Bytes, []byte("No, keep the name."))
		case s.Manager == "lending" && s.Phase == run.PhaseWork && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "owner", "May Item be renamed?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: fmt.Sprintf(`{"on":[{"kind":"owner","ref":%q}],"reason":"Owner decision."}`, asked), Want: run.ToolOK},
			}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	answer, err := h.c.AnswerCommand(meta("answer"), asked, "No, keep the name.")
	if err != nil {
		t.Fatal(err)
	}
	commit(t, h.c, h.j, answer)
	resume := command("resume", run.ResumeRun)
	resume.Run = h.c.Current().ID
	commit(t, h.c, h.j, resume)
	runToEnd(t, h)
	// The root's local cases were recorded before the answer and still stand
	// for its local submission after it; only its integration records new ones.
	if h.c.Current().State != run.Verifying || !rootSaw || strings.Count(strings.Join(admitted(h), ","), ". reviewer-expectations") != 2 {
		t.Fatal("the answer renewed the sibling's cases or did not reach it", h.c.Current(), rootSaw, admitted(h))
	}
}

// The owner's answer to a review's question starts a fresh review of the
// same frozen candidate, which sees the decision (core rule T8).
func TestAnsweredReviewQuestionStartsAFreshReview(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate {
			if s.Attempt == 0 {
				return []action{{Tool: run.ToolSubmitReview, Args: `{"verdict":"needs_decision","summary":"Open format.","question":"Which format?","evidence":[],"cases":[{"name":"item","assessment":"evidenced","grounds":"Checked."}],"findings":[],"resolutions":[]}`, Want: run.ToolOK}}
			}
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("Plain text.")) {
				t.Error("the fresh review does not see the owner's decision")
			}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	qs := h.c.OwnerQuestions()
	if h.c.Current().State != run.Waiting || len(qs) != 1 {
		t.Fatal("run does not wait for the review's question", h.c.Current(), qs)
	}
	answer, err := h.c.AnswerCommand(meta("answer"), qs[0].ID, "Plain text.")
	if err != nil {
		t.Fatal(err)
	}
	commit(t, h.c, h.j, answer)
	if v := h.task("lending"); v.State != run.TaskReviewing || v.Submission.Proposal == (run.BlobRef{}) {
		t.Fatalf("the answer did not start a fresh review: %+v", v)
	}
	resume := command("resume", run.ResumeRun)
	resume.Run = h.c.Current().ID
	commit(t, h.c, h.j, resume)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.c.Current())
	}
}

// A parent's hint reaches its direct child only; a message to an unknown
// addressee is refused (core rule M1).
func TestHintsReachDirectChildren(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseWork:
			return append([]action{
				{Tool: run.ToolSend, Args: sendArgs("hint", "lending", "Prefer small files."), Want: run.ToolOK},
				{Tool: run.ToolSend, Args: sendArgs("hint", "nowhere", "Lost."), Want: run.ToolRefused},
			}, happyAgent(s)...)
		case s.Manager == "lending" && s.Phase == run.PhaseWork:
			if !bytes.Contains(s.Packet.Context.Bytes, []byte("Prefer small files.")) {
				t.Error("the child does not see its parent's hint")
			}
		}
		return happyAgent(s)
	}
	h := newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not complete", h.c.Current())
	}
	restored, err := run.Restore(context.Background(), &readJournal{store: h.j}, "repo", run.RestoreLimits{PageSize: 16, Batches: 2000, Events: 4000, Bytes: 64 << 20})
	if err != nil || len(restored.Messages()) != len(h.c.Messages()) {
		t.Fatal("restore lost the messages", err)
	}
}

// A question to another branch travels along the tree; the Managers between
// sender and addressee see it without being dispatched (core rule M2).
func TestQuestionsTravelAlongTheTree(t *testing.T) {
	t.Parallel()
	agent := func(s agentStep) []action {
		if s.Manager == "lending/reminders" && s.Phase == run.PhaseWork && s.Attempt == 0 {
			return append([]action{{Tool: run.ToolSend, Args: sendArgs("question", "empty", "Do you own the shared format?"), Want: run.ToolOK}}, happyAgent(s)...)
		}
		return happyAgent(s)
	}
	h := newHarness(t, false, agent)
	runToEnd(t, h)
	messages := h.c.Messages()
	if len(messages) == 0 || strings.Join(messages[0].Path, ",") != "lending,." || messages[0].To != "empty" {
		t.Fatalf("question not routed along the tree: %+v", messages)
	}
	if !bytes.Contains(h.packet("lending", run.PhaseIntegrate).Context.Bytes, []byte("Do you own the shared format?")) {
		t.Fatal("a Manager on the path does not see the question")
	}
}

// Each open question admits its own reply step: two Managers that ask a done
// Manager both get a reply (core rule T11).
func TestEachQuestionGetsItsReplyStep(t *testing.T) {
	t.Parallel()
	var asks []string
	replied := 0
	agent := func(s agentStep) []action {
		switch {
		case (s.Manager == "lending" || s.Manager == "lending/reminders") && s.Phase == run.PhaseWork && s.Attempt == 0:
			id := questionID(s, 1)
			asks = append(asks, id)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "empty", "Which format, "+s.Manager+"?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(id), Want: run.ToolOK},
			}
		case s.Manager == "empty" && s.Phase == run.PhaseReply:
			id := asks[replied]
			replied++
			return []action{{Tool: run.ToolSend, Args: sendArgs("reply", id, "The shared one."), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	inputs := stepInputs(t, false)
	inputs.Binding.Policy.Assignment.Steps, inputs.Binding.Policy.Assignment.SessionStarts = 64, 64
	h := newHarnessWith(t, inputs, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(asks) != 2 || replied != 2 {
		t.Fatal("not every question was answered", h.c.Current(), asks, replied)
	}
	for _, m := range h.c.Messages() {
		if m.Kind == "reply" && !slices.Contains(asks, m.About) {
			t.Fatalf("a reply answers no asked question: %+v", m)
		}
	}
}

// A reply step the Host could not finish drops its reply and is retried.
func TestAReplyStepTheHostCouldNotFinishIsRetried(t *testing.T) {
	t.Parallel()
	var asked string
	var h *harness
	replies := 0
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "." && s.Phase == run.PhaseIntegrate && s.Attempt == 0:
			asked = questionID(s, 1)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "lending", "Should Item carry a title?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(asked), Want: run.ToolOK},
			}
		case s.Manager == "lending" && s.Phase == run.PhaseReply:
			replies++
			if replies == 1 {
				h.ws.failHarvest = 1
			}
			return []action{{Tool: run.ToolSend, Args: sendArgs("reply", asked, "Yes, a title."), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	h = newHarness(t, true, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || replies != 2 {
		t.Fatal("the failed reply step was not retried", h.c.Current(), replies)
	}
	var count int
	for _, m := range h.c.Messages() {
		if m.Kind == "reply" {
			count++
		}
	}
	if count != 1 {
		t.Fatal("the failed reply step's reply was kept", count)
	}
}

// A reply step takes the question it answered: two questions to the same
// done Manager, answered newest first, both get a reply (core rule T11).
func TestAReplyStepTakesTheQuestionItAnswered(t *testing.T) {
	t.Parallel()
	var asks []string
	replied := 0
	agent := func(s agentStep) []action {
		switch {
		case (s.Manager == "lending" || s.Manager == "lending/reminders") && s.Phase == run.PhaseWork && s.Attempt == 0:
			id := questionID(s, 1)
			asks = append(asks, id)
			return []action{
				{Tool: run.ToolSend, Args: sendArgs("question", "empty", "Which format, "+s.Manager+"?"), Want: run.ToolOK},
				{Tool: run.ToolWait, Args: waitReplyArgs(id), Want: run.ToolOK},
			}
		case s.Manager == "empty" && s.Phase == run.PhaseReply:
			id := asks[len(asks)-1-replied]
			replied++
			return []action{{Tool: run.ToolSend, Args: sendArgs("reply", id, "The shared one."), Want: run.ToolOK}}
		}
		return happyAgent(s)
	}
	inputs := stepInputs(t, false)
	inputs.Binding.Policy.Assignment.Steps, inputs.Binding.Policy.Assignment.SessionStarts = 64, 64
	h := newHarnessWith(t, inputs, agent)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying || len(asks) != 2 || replied != 2 {
		t.Fatal("a question was left without its reply step", h.c.Current(), asks, replied)
	}
}
