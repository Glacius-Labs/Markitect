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
	childFinding := `{"verdict":"changes_requested","summary":"Item lacks a title.","evidence":[],"resolutions":[],"findings":[{"to":"lending","severity":"blocking","definitions":["lending/Item"],"grounds":"Item has no title.","requestedEvidence":"A title field."}]}`
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
	if root := h.task("."); root.Rejections["integration"] != 1 {
		t.Fatal("the integration rejection was not counted", root.Rejections)
	}
	issues := h.c.Issues("lending")
	if !slices.ContainsFunc(issues, func(i run.Issue) bool { return i.ID == finding && i.State == "resolved" && i.Purpose == "local" }) {
		t.Fatalf("the child's local review did not resolve the finding: %+v", issues)
	}
	if got := aggregateFile(t, h.engine, "out/library.go"); got != "package library\n\ntype Item struct{ Title string }\n" {
		t.Fatalf("the correction was not composed: %q", got)
	}
}

// A child's open owner question does not block its parent's local delivery,
// which does not contain the child's work; the run waits for the owner
// (core rules I3, W1).
func TestAChildsOpenQuestionLeavesTheParentsLocalReview(t *testing.T) {
	agent := func(s agentStep) []action {
		switch {
		case s.Manager == "lending" && s.Phase == run.PhaseReviewerCandidate && s.Attempt == 0:
			return []action{{Tool: run.ToolSubmitReview, Args: `{"verdict":"needs_decision","summary":"Open.","question":"Which format?","evidence":[],"findings":[],"resolutions":[]}`, Want: run.ToolOK}}
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
	grand := `{"verdict":"changes_requested","summary":"Reminder lacks a title.","evidence":[],"resolutions":[],"findings":[{"to":"lending/reminders","severity":"blocking","definitions":["lending/reminders/Item"],"grounds":"No title.","requestedEvidence":"A title."}]}`
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
	both := `{"verdict":"changes_requested","summary":"Both lack a title.","evidence":[],"resolutions":[],"findings":[` +
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
