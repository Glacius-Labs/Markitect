package projectrun

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func schedulerTask(id, parent string, depth int, state string, owns ...string) ManagerTask {
	return ManagerTask{ManagerID: id, ParentTask: parent, Depth: depth, State: state, Owns: owns}
}

func TestManagerDependenciesUseExplicitReferencesInConsumerDirection(t *testing.T) {
	report := projectmodel.Report{
		Statements: []projectmodel.Statement{
			{ID: "statement:provider", Owner: "provider"},
			{ID: "statement:consumer", Owner: "consumer", Requires: []string{"statement:provider"}},
		},
	}
	tasks := []ManagerTask{
		schedulerTask("root", "", 0, "worked", "."),
		schedulerTask("consumer", "root", 1, "queued", "src/consumer/"),
		schedulerTask("provider", "root", 1, "queued", "src/provider/"),
	}
	got, err := managerDependencies(report, tasks)
	if err != nil {
		t.Fatal(err)
	}
	if len(got["consumer"]) != 1 || got["consumer"][0] != "provider" {
		t.Fatalf("consumer prerequisites = %v, want [provider]", got["consumer"])
	}
	if len(got["provider"]) != 0 {
		t.Fatalf("provider acquired reverse dependency: %v", got["provider"])
	}
	actions, err := readyManagerActions(tasks, got, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0] != (managerAction{managerID: "provider", phase: "work"}) {
		t.Fatalf("ready actions = %+v, want provider work only", actions)
	}
	tasks[2].State = "worked"
	actions, err = readyManagerActions(tasks, got, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0] != (managerAction{managerID: "consumer", phase: "work"}) {
		t.Fatalf("after provider completion, ready actions = %+v, want consumer work", actions)
	}
}

func TestPlanManagersIncludesTypedDependencyOwnerAndAncestors(t *testing.T) {
	const (
		rootID       = "root"
		consumerID   = "consumer"
		providerTree = "provider-tree"
		providerID   = "provider"
		unrelatedID  = "unrelated"
	)
	report := projectmodel.Report{
		Managers: []projectmodel.Manager{
			{ID: rootID},
			{ID: consumerID, Parent: rootID, Owns: []string{"src/consumer/"}},
			{ID: providerTree, Parent: rootID, Owns: []string{"src/provider/"}},
			{ID: providerID, Parent: providerTree, Owns: []string{"src/provider/lib/"}},
			{ID: unrelatedID, Parent: rootID, Owns: []string{"src/unrelated/"}},
		},
		Statements: []projectmodel.Statement{
			{ID: "contract:consumer", Owner: consumerID, Description: "Prose mentions unrelated; references are typed below.", Requires: []string{"contract:provider"}},
			{ID: "contract:provider", Owner: providerID},
			{ID: "contract:unrelated", Owner: unrelatedID, Description: "Uses provider as prose, but has no typed dependency."},
		},
	}
	tasks, selected, _, err := planManagers(report, map[string][]byte{}, PlanRequest{
		Operation: OperationApply,
		Goal:      "Update the consumer.",
		Managers:  []string{consumerID},
	}, nil, nil, Limits{MaxDepth: 4})
	if err != nil {
		t.Fatalf("planManagers: %v", err)
	}
	for _, id := range []string{rootID, consumerID, providerTree, providerID} {
		if !selected[id] {
			t.Errorf("typed dependency closure omitted Manager %s: selected=%v", id, selected)
		}
	}
	if selected[unrelatedID] {
		t.Fatalf("prose caused unrelated Manager selection: %v", selected)
	}
	if len(tasks) != 4 {
		t.Fatalf("selected tasks = %+v, want root, consumer, provider-tree, provider", tasks)
	}
}

func TestPlanManagersRejectsCyclesInSelectedTypedDependencies(t *testing.T) {
	const rootID, alphaID, bravoID = "root", "alpha", "bravo"
	report := projectmodel.Report{
		Managers: []projectmodel.Manager{
			{ID: rootID},
			{ID: alphaID, Parent: rootID},
			{ID: bravoID, Parent: rootID},
		},
		Statements: []projectmodel.Statement{
			{ID: "statement:alpha", Owner: alphaID, Requires: []string{"statement:bravo"}},
			{ID: "statement:bravo", Owner: bravoID, Uses: []string{"statement:alpha"}},
		},
	}
	_, _, _, err := planManagers(report, nil, PlanRequest{Operation: OperationApply, Managers: []string{alphaID}}, nil, nil, Limits{MaxDepth: 4})
	if err == nil || !strings.Contains(err.Error(), "manager dependency cycle") {
		t.Fatalf("plan cycle result = %v, want selected dependency cycle error", err)
	}
}

func TestManagerDependenciesExcludeAncestorAndDescendantEdges(t *testing.T) {
	report := projectmodel.Report{Statements: []projectmodel.Statement{
		{ID: "statement:parent", Owner: "parent"},
		{ID: "statement:child", Owner: "child", Uses: []string{"statement:parent"}},
	}}
	tasks := []ManagerTask{
		schedulerTask("root", "", 0, "worked", "."),
		schedulerTask("parent", "root", 1, "worked", "src/"),
		schedulerTask("child", "parent", 2, "queued", "src/child/"),
	}
	got, err := managerDependencies(report, tasks)
	if err != nil {
		t.Fatal(err)
	}
	if len(got["child"]) != 0 {
		t.Fatalf("ancestor edge should be provided by tree order, got %v", got["child"])
	}
	actions, err := readyManagerActions(tasks, got, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].managerID != "child" {
		t.Fatalf("child should become ready after its parent work: %+v", actions)
	}
}

func TestReadyManagerActionsWaitForCrossBranchDependencyIntegration(t *testing.T) {
	report := projectmodel.Report{Statements: []projectmodel.Statement{
		{ID: "statement:provider", Owner: "provider"},
		{ID: "statement:consumer", Owner: "consumer", Uses: []string{"statement:provider"}},
	}}
	tasks := []ManagerTask{
		schedulerTask("root", "", 0, "worked", "."),
		schedulerTask("consumer", "root", 1, "queued", "src/consumer/"),
		schedulerTask("provider", "root", 1, "worked", "src/provider/"),
		schedulerTask("provider-child", "provider", 2, "worked", "src/provider/child/"),
	}
	deps, err := managerDependencies(report, tasks)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readyManagerActions(tasks, deps, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0] != (managerAction{managerID: "provider", phase: "integrate"}) {
		t.Fatalf("consumer ran before dependency tree integration: %+v", actions)
	}
	tasks[2].State = "integrated"
	actions, err = readyManagerActions(tasks, deps, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0] != (managerAction{managerID: "consumer", phase: "work"}) {
		t.Fatalf("consumer was not released after dependency integration: %+v", actions)
	}
}

func TestReadyManagerActionsBatchIndependentWorkAndSerializeOverlappingScopes(t *testing.T) {
	tasks := []ManagerTask{
		schedulerTask("root", "", 0, "worked", "."),
		schedulerTask("alpha", "root", 1, "queued", "src/shared/"),
		schedulerTask("bravo", "root", 1, "queued", "src/shared/bravo/"),
		schedulerTask("charlie", "root", 1, "queued", "tests/"),
	}
	actions, err := readyManagerActions(tasks, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []managerAction{{managerID: "alpha", phase: "work"}, {managerID: "charlie", phase: "work"}}
	if len(actions) != len(want) {
		t.Fatalf("ready batch = %+v, want %+v", actions, want)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("ready batch = %+v, want %+v", actions, want)
		}
	}
}

func TestReadyManagerActionsScheduleParentBeforeChildAndIntegrationBottomUp(t *testing.T) {
	tasks := []ManagerTask{
		schedulerTask("root", "", 0, "queued", "."),
		schedulerTask("middle", "root", 1, "queued", "src/"),
		schedulerTask("leaf", "middle", 2, "queued", "src/leaf/"),
	}
	actions, err := readyManagerActions(tasks, nil, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0] != (managerAction{managerID: "root", phase: "work"}) {
		t.Fatalf("initial batch = %+v, want root work", actions)
	}

	tasks[0].State = "worked"
	actions, err = readyManagerActions(tasks, nil, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0] != (managerAction{managerID: "middle", phase: "work"}) {
		t.Fatalf("second batch = %+v, want middle work", actions)
	}

	tasks[1].State = "worked"
	tasks[2].State = "worked"
	actions, err = readyManagerActions(tasks, nil, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0] != (managerAction{managerID: "middle", phase: "integrate"}) {
		t.Fatalf("bottom-up batch = %+v, want middle integration", actions)
	}

	tasks[1].State = "integrated"
	actions, err = readyManagerActions(tasks, nil, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0] != (managerAction{managerID: "root", phase: "integrate"}) {
		t.Fatalf("root integration before child integration = %+v", actions)
	}
}

func TestManagerDependenciesDetectCyclesDeterministically(t *testing.T) {
	report := projectmodel.Report{Statements: []projectmodel.Statement{
		{ID: "statement:a", Owner: "alpha", Requires: []string{"statement:b"}},
		{ID: "statement:b", Owner: "bravo", Uses: []string{"statement:a"}},
	}}
	tasks := []ManagerTask{
		schedulerTask("root", "", 0, "worked", "."),
		schedulerTask("alpha", "root", 1, "queued", "src/a/"),
		schedulerTask("bravo", "root", 1, "queued", "src/b/"),
	}
	var first string
	for i := 0; i < 8; i++ {
		_, err := managerDependencies(report, tasks)
		if err == nil || !strings.Contains(err.Error(), "manager dependency cycle") {
			t.Fatalf("cycle result = %v, want deterministic cycle error", err)
		}
		if i == 0 {
			first = err.Error()
		} else if err.Error() != first {
			t.Fatalf("cycle diagnostic varied: %q != %q", err, first)
		}
	}
}
