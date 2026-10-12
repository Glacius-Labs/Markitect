package projectrun

import (
	"reflect"
	"testing"
)

func TestReportClosureRequiresExplicitlyResolvedEscalations(t *testing.T) {
	for _, status := range []string{"open", "escalated", "", "unknown", "resolved"} {
		t.Run(status, func(t *testing.T) {
			report := RunReport{Tasks: []ManagerTask{{ManagerID: "root", ReportStatus: "complete"}},
				Escalations: []Escalation{{ID: "decision", Status: status}}}
			err := validateReportClosure(report)
			if (err == nil) != (status == "resolved") {
				t.Fatalf("closure for escalation status %q: %v", status, err)
			}
		})
	}
}

func TestObligationResolutionPropagatesOnlyAlongProvenanceChain(t *testing.T) {
	const (
		origin = "a"
		middle = "b"
		root   = "root"
	)
	original := []Obligation{
		{ID: "q1", Kind: "question", Text: "question one", OriginManager: origin, Chain: []string{origin}},
		{ID: "q2", Kind: "question", Text: "question two", OriginManager: origin, Chain: []string{origin}},
		{ID: "r1", Kind: "risk", Text: "risk one", OriginManager: origin, Chain: []string{origin}},
		{ID: "r2", Kind: "risk", Text: "risk two", OriginManager: origin, Chain: []string{origin}},
	}
	originTask := ManagerTask{ID: "task-a", ManagerID: origin, ParentTask: middle, ReportStatus: "partial", Questions: []string{"question one", "question two"}, Risks: []string{"risk one", "risk two"}, Obligations: cloneObligations(original)}
	middleRecords := appendForwardedObligations(original, middle)
	middleTask := ManagerTask{ID: "task-b", ManagerID: middle, ParentTask: root, ReportStatus: "partial", Questions: []string{"question one", "question two"}, Risks: []string{"risk one", "risk two"}, Obligations: middleRecords}
	rootTask := ManagerTask{ID: "task-root", ManagerID: root, ReportStatus: "complete"}
	tasks := []ManagerTask{originTask, middleTask, rootTask}
	ids := []string{"q1", "q2", "r1", "r2"}
	report := RunReport{Tasks: tasks, Escalations: []Escalation{
		{ID: "a-to-b", FromManager: origin, ToManager: middle, ObligationIDs: append([]string(nil), ids...), Status: "escalated"},
		{ID: "b-to-root", FromManager: middle, ToManager: root, ObligationIDs: append([]string(nil), ids...), Status: "escalated"},
	}}

	outstanding, err := managerObligationRecords(report.Tasks[2], report.Tasks, []string{middle})
	if err != nil {
		t.Fatal(err)
	}
	resolved, remaining, err := resolveObligationRecords(outstanding, []string{"question one"}, []string{"risk one"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 2 || len(remaining) != 2 {
		t.Fatalf("partial typed resolution selected=%+v remaining=%+v", resolved, remaining)
	}
	rootObligations := appendForwardedObligations(remaining, root)
	report.Tasks[2].Obligations = rootObligations
	report.Tasks[2].Questions, report.Tasks[2].Risks = obligationTexts(rootObligations)
	report.Tasks[2].ReportStatus = "partial"
	if err := applyObligationResolutions(&report, resolved); err != nil {
		t.Fatalf("apply partial provenance resolution: %v", err)
	}
	for _, index := range []int{0, 1} {
		if len(report.Tasks[index].Questions) != 1 || report.Tasks[index].Questions[0] != "question two" || len(report.Tasks[index].Risks) != 1 || report.Tasks[index].Risks[0] != "risk two" {
			t.Fatalf("partial ancestor resolution cleared unrelated origin obligations at task %s: %+v", report.Tasks[index].ManagerID, report.Tasks[index])
		}
	}
	if report.Tasks[0].ReportStatus != "partial" || report.Tasks[1].ReportStatus != "partial" {
		t.Fatal("tasks with unresolved sibling obligations were marked complete")
	}
	if report.Escalations[0].Status == "resolved" || report.Escalations[1].Status == "resolved" {
		t.Fatal("partial resolution marked a multi-obligation escalation fully resolved")
	}

	rootOutstanding, err := managerObligationRecords(report.Tasks[2], report.Tasks, []string{middle})
	if err != nil {
		t.Fatal(err)
	}
	resolvedAll, remainingAll, err := resolveObligationRecords(rootOutstanding, []string{"question two"}, []string{"risk two"})
	if err != nil || len(remainingAll) != 0 {
		t.Fatalf("final provenance resolution failed: remaining=%+v err=%v", remainingAll, err)
	}
	if err := applyObligationResolutions(&report, resolvedAll); err != nil {
		t.Fatalf("apply final provenance resolution: %v", err)
	}
	for _, task := range report.Tasks {
		if len(task.Questions)+len(task.Risks)+len(task.Obligations) != 0 || task.ReportStatus != "complete" {
			t.Fatalf("resolved obligation remained on chain task %s: %+v", task.ManagerID, task)
		}
	}
	for _, escalation := range report.Escalations {
		if escalation.Status != "resolved" {
			t.Fatalf("resolved chain escalation was not retained as resolved: %+v", escalation)
		}
	}
}

func TestObligationResolutionRejectsAmbiguousSiblingAndInvalidLegacyChains(t *testing.T) {
	const root = "root"
	a := ManagerTask{ID: "a", ManagerID: "a", ParentTask: root, ReportStatus: "partial", Questions: []string{"same text"}, Obligations: []Obligation{{ID: "a-q", Kind: "question", Text: "same text", OriginManager: "a", Chain: []string{"a"}}}}
	b := ManagerTask{ID: "b", ManagerID: "b", ParentTask: root, ReportStatus: "partial", Questions: []string{"same text"}, Obligations: []Obligation{{ID: "b-q", Kind: "question", Text: "same text", OriginManager: "b", Chain: []string{"b"}}}}
	rootTask := ManagerTask{ID: "root", ManagerID: root, ReportStatus: "complete"}
	tasks := []ManagerTask{a, b, rootTask}
	if _, err := managerObligationRecords(rootTask, tasks, []string{"a", "b"}); err == nil {
		t.Fatal("identical sibling text with distinct origins was not rejected as ambiguous")
	}

	invalid := a
	invalid.Obligations = []Obligation{{ID: "bad-chain", Kind: "question", Text: "same text", OriginManager: "a", Chain: []string{"a", "b"}}}
	if _, err := managerObligationRecords(rootTask, []ManagerTask{invalid, b, rootTask}, []string{"a"}); err == nil {
		t.Fatal("non-parent forwarding chain was accepted")
	}
	legacy := a
	legacy.Obligations = nil
	if _, err := managerObligationRecords(rootTask, []ManagerTask{legacy, rootTask}, []string{"a"}); err == nil {
		t.Fatal("legacy obligation text without typed provenance was silently accepted")
	}
}

func TestQuestionAndRiskWithSameTextRemainDistinctObligations(t *testing.T) {
	records := []Obligation{
		{ID: "q", Kind: "question", Text: "same wording", OriginManager: "a", Chain: []string{"a"}},
		{ID: "r", Kind: "risk", Text: "same wording", OriginManager: "a", Chain: []string{"a"}},
	}
	resolved, remaining, err := resolveObligationRecords(records, []string{"same wording"}, nil)
	if err != nil || len(resolved) != 1 || resolved[0].Kind != "question" || len(remaining) != 1 || remaining[0].Kind != "risk" {
		t.Fatalf("question resolution crossed into same-text risk: resolved=%+v remaining=%+v err=%v", resolved, remaining, err)
	}
}

func TestReintroducedSameTextGetsFreshOccurrenceIdentity(t *testing.T) {
	first, err := newLocalObligations("orders", []string{"same question"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	old := first[0]
	old.Chain = []string{"orders", "root"}
	oldEscalation := Escalation{ID: "old-escalation", FromManager: "orders", ToManager: "root", ObligationIDs: []string{old.ID}, Status: "resolved"}
	second, err := newLocalObligations("orders", []string{"same question"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if second[0].ID == old.ID {
		t.Fatal("reintroduced obligation reused the prior resolved occurrence ID")
	}
	rootTask := ManagerTask{ID: "root", ManagerID: "root"}
	ordersTask := ManagerTask{ID: "orders", ManagerID: "orders", ParentTask: "root", ReportStatus: "partial", Questions: []string{"same question"}, Obligations: second}
	report := RunReport{Tasks: []ManagerTask{ordersTask, rootTask}, Escalations: []Escalation{oldEscalation}}
	if err := applyObligationResolutions(&report, []Obligation{old}); err == nil {
		t.Fatal("stale resolution for a prior occurrence was accepted")
	}
	if len(report.Tasks[0].Obligations) != 1 || report.Tasks[0].Obligations[0].ID != second[0].ID || report.Tasks[0].ReportStatus != "partial" {
		t.Fatalf("old resolution cleared the reintroduced occurrence: %+v", report.Tasks[0])
	}
	if report.Escalations[0].Status != "resolved" {
		t.Fatalf("old resolution history was not retained: %+v", report.Escalations[0])
	}
}

func TestObligationResolutionPreservesExactTextAndFailsAtomicallyOnBrokenHop(t *testing.T) {
	local, err := newLocalObligations("orders", []string{" question with spaces "}, []string{" risk with spaces "})
	if err != nil {
		t.Fatal(err)
	}
	if local[0].Text != " question with spaces " || local[1].Text != " risk with spaces " {
		t.Fatalf("obligation creation normalized accepted response text: %+v", local)
	}
	forwarded := appendForwardedObligations(local, "root")
	orders := ManagerTask{ID: "orders", ManagerID: "orders", ParentTask: "root", ReportStatus: "partial", Questions: []string{" question with spaces "}, Risks: []string{" risk with spaces "}, Obligations: cloneObligations(local)}
	root := ManagerTask{ID: "root", ManagerID: "root", ReportStatus: "partial", Questions: []string{" question with spaces "}, Risks: []string{" risk with spaces "}, Obligations: forwarded}
	report := RunReport{Tasks: []ManagerTask{orders, root}}
	broken := cloneTasks(report.Tasks)
	broken[1].Obligations = broken[1].Obligations[:1]
	broken[0].Obligations = broken[0].Obligations[1:]
	brokenReport := RunReport{Tasks: broken}
	beforeApply := cloneTasks(brokenReport.Tasks)
	var question *Obligation
	for i := range brokenReport.Tasks[1].Obligations {
		if brokenReport.Tasks[1].Obligations[i].Kind == "question" {
			question = &brokenReport.Tasks[1].Obligations[i]
		}
	}
	if question == nil {
		t.Fatal("test setup omitted question provenance")
	}
	if err := applyObligationResolutions(&brokenReport, []Obligation{cloneObligation(*question)}); err == nil {
		t.Fatal("resolution accepted a missing origin-hop record")
	}
	if !reflect.DeepEqual(brokenReport.Tasks, beforeApply) {
		t.Fatalf("failed provenance validation partially mutated tasks: before=%+v after=%+v", beforeApply, brokenReport.Tasks)
	}
}

func TestManagerCanResolveOwnAndPreviouslyForwardedObligations(t *testing.T) {
	local, err := newLocalObligations("root", []string{"own question"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	root := ManagerTask{ID: "root-task", ManagerID: "root", Questions: []string{"own question"}, Obligations: cloneObligations(local), ReportStatus: "partial"}
	localReport := RunReport{Tasks: []ManagerTask{root}}
	resolved, remaining, err := resolveObligationRecords(local, []string{"own question"}, nil)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("resolve own obligation: resolved=%+v remaining=%+v err=%v", resolved, remaining, err)
	}
	if err := applyObligationResolutions(&localReport, resolved); err != nil {
		t.Fatalf("apply own obligation resolution: %v", err)
	}
	if len(localReport.Tasks[0].Obligations) != 0 || localReport.Tasks[0].ReportStatus != "complete" {
		t.Fatalf("own obligation did not resolve: %+v", localReport.Tasks[0])
	}

	forwarded, err := newLocalObligations("orders", nil, []string{"previously forwarded risk"})
	if err != nil {
		t.Fatal(err)
	}
	orders := ManagerTask{ID: "orders-task", ManagerID: "orders", ParentTask: "root", Risks: []string{"previously forwarded risk"}, Obligations: cloneObligations(forwarded), ReportStatus: "partial"}
	rootObligations := appendForwardedObligations(forwarded, "root")
	root = ManagerTask{ID: "root-task", ManagerID: "root", Risks: []string{"previously forwarded risk"}, Obligations: rootObligations, ReportStatus: "partial"}
	report := RunReport{Tasks: []ManagerTask{orders, root}, Escalations: []Escalation{{ID: "orders-root", FromManager: "orders", ToManager: "root", ObligationIDs: []string{forwarded[0].ID}, Status: "escalated"}}}
	outstanding, err := managerObligationRecords(root, report.Tasks, []string{"orders"})
	if err != nil {
		t.Fatal(err)
	}
	resolved, remaining, err = resolveObligationRecords(outstanding, nil, []string{"previously forwarded risk"})
	if err != nil || len(remaining) != 0 {
		t.Fatalf("resolve previously forwarded own obligation: resolved=%+v remaining=%+v err=%v", resolved, remaining, err)
	}
	if err := applyObligationResolutions(&report, resolved); err != nil {
		t.Fatalf("apply previously forwarded own obligation resolution: %v", err)
	}
	for _, task := range report.Tasks {
		if len(task.Obligations) != 0 || task.ReportStatus != "complete" {
			t.Fatalf("forwarded obligation remained at %s: %+v", task.ManagerID, task)
		}
	}
	if report.Escalations[0].Status != "resolved" {
		t.Fatalf("forwarded escalation was not resolved: %+v", report.Escalations[0])
	}
}
