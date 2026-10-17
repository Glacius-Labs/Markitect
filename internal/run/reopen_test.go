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
