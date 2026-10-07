package assurance

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

func runFixture(t *testing.T, nodeID string, scopes []string, checks []records.CheckIdentity, outcome string) (NodeRunOutput, records.Freshness) {
	t.Helper()
	evidence := testEvidence(t, nodeID, scopes, checks, outcome)
	return NodeRunOutput{
		NodeID: nodeID, Disposition: RunCompleted, Record: evidence.Record, Result: evidence.Result,
	}, evidence.Current
}

func runGraph(rootIDs []string, nodes ...Node) Input {
	return Input{RootIDs: rootIDs, Nodes: nodes}
}

func TestExecuteRunsBottomUpAndComposesOwnFailureAfterChildPass(t *testing.T) {
	parentChecks, childChecks := testChecks("compose"), testChecks("unit")
	parentOutput, parentCurrent := runFixture(t, "parent", []string{"app"}, parentChecks, OutcomeFailed)
	childOutput, childCurrent := runFixture(t, "child", []string{"app"}, childChecks, OutcomePassed)
	input := RunInput{
		Graph: runGraph([]string{"parent"},
			Node{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: parentChecks},
			Node{ID: "child", ScopeIDs: []string{"app"}, RequiredChecks: childChecks},
		),
		Current: map[string]records.Freshness{"parent": parentCurrent, "child": childCurrent},
	}
	outputs := map[string]NodeRunOutput{"parent": parentOutput, "child": childOutput}
	var order []string
	report, err := Execute(context.Background(), input, func(_ context.Context, in NodeRunInput) (NodeRunOutput, error) {
		order = append(order, in.Node.ID)
		if in.Node.ID == "parent" {
			if len(in.Children) != 1 || in.Children[0].Result.Outcome != OutcomePassed || in.Children[0].Evidence == nil {
				t.Fatalf("parent child input = %#v", in.Children)
			}
			if in.Children[0].Evidence.Result.Reason != "" {
				t.Fatal("parent received child verification reasoning")
			}
		}
		return outputs[in.Node.ID], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"child", "parent"}) {
		t.Fatalf("execution order = %v", order)
	}
	if report.Evaluation.Roots[0].Outcome != OutcomeFailed {
		t.Fatalf("root outcome = %q, want failed from the parent's own check", report.Evaluation.Roots[0].Outcome)
	}
	if got := report.Invocations[1].Status; got != InvocationCheckerFailed {
		t.Fatalf("parent status = %q", got)
	}
}

func TestExecuteRunsSharedDiamondNodeOnce(t *testing.T) {
	checks := testChecks("scope")
	outputs := make(map[string]NodeRunOutput)
	current := make(map[string]records.Freshness)
	for _, id := range []string{"root", "left", "right", "leaf"} {
		outputs[id], current[id] = runFixture(t, id, []string{"app"}, checks, OutcomePassed)
	}
	graph := runGraph([]string{"root"},
		Node{ID: "root", ScopeIDs: []string{"app"}, Children: []string{"right", "left"}, RequiredChecks: checks},
		Node{ID: "left", ScopeIDs: []string{"app"}, Children: []string{"leaf"}, RequiredChecks: checks},
		Node{ID: "right", ScopeIDs: []string{"app"}, Children: []string{"leaf"}, RequiredChecks: checks},
		Node{ID: "leaf", ScopeIDs: []string{"app"}, RequiredChecks: checks},
	)
	var calls []string
	report, err := Execute(context.Background(), RunInput{Graph: graph, Current: current}, func(_ context.Context, in NodeRunInput) (NodeRunOutput, error) {
		calls = append(calls, in.Node.ID)
		return outputs[in.Node.ID], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"leaf", "left", "right", "root"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	if len(report.Invocations) != len(want) || report.Evaluation.Roots[0].Outcome != OutcomePassed {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestExecuteRejectsMalformedGraphBeforeRunner(t *testing.T) {
	checks := testChecks("check")
	input := RunInput{Graph: runGraph([]string{"one"},
		Node{ID: "one", ScopeIDs: []string{"app"}, Children: []string{"two"}, RequiredChecks: checks},
		Node{ID: "two", ScopeIDs: []string{"app"}, Children: []string{"one"}, RequiredChecks: checks},
	)}
	calls := 0
	_, err := Execute(context.Background(), input, func(context.Context, NodeRunInput) (NodeRunOutput, error) {
		calls++
		return NodeRunOutput{}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("error = %v, want cycle validation", err)
	}
	if calls != 0 {
		t.Fatalf("runner called %d times for malformed graph", calls)
	}
}

func TestExecutePropagatesStaleChildAsIncompleteEvenWhenParentPasses(t *testing.T) {
	parentChecks, childChecks := testChecks("compose"), testChecks("unit")
	parentOutput, parentCurrent := runFixture(t, "parent", []string{"app"}, parentChecks, OutcomePassed)
	childOutput, childCurrent := runFixture(t, "child", []string{"app"}, childChecks, OutcomePassed)
	childCurrent.Revision = testRevisionB
	input := RunInput{
		Graph: runGraph([]string{"parent"},
			Node{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: parentChecks},
			Node{ID: "child", ScopeIDs: []string{"app"}, RequiredChecks: childChecks},
		),
		Current: map[string]records.Freshness{"parent": parentCurrent, "child": childCurrent},
	}
	outputs := map[string]NodeRunOutput{"parent": parentOutput, "child": childOutput}
	report, err := Execute(context.Background(), input, func(_ context.Context, in NodeRunInput) (NodeRunOutput, error) {
		return outputs[in.Node.ID], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := findNode(t, report.Evaluation, "child").Outcome; got != OutcomeIncomplete {
		t.Fatalf("stale child outcome = %q", got)
	}
	if got := findNode(t, report.Evaluation, "parent").Outcome; got != OutcomeIncomplete {
		t.Fatalf("parent outcome = %q, want incomplete", got)
	}
}

func TestExecuteParentCannotPassWhenChildCheckerFails(t *testing.T) {
	parentChecks, childChecks := testChecks("compose"), testChecks("unit")
	parentOutput, parentCurrent := runFixture(t, "parent", []string{"app"}, parentChecks, OutcomePassed)
	childOutput, childCurrent := runFixture(t, "child", []string{"app"}, childChecks, OutcomeFailed)
	input := RunInput{
		Graph: runGraph([]string{"parent"},
			Node{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: parentChecks},
			Node{ID: "child", ScopeIDs: []string{"app"}, RequiredChecks: childChecks},
		),
		Current: map[string]records.Freshness{"parent": parentCurrent, "child": childCurrent},
	}
	outputs := map[string]NodeRunOutput{"parent": parentOutput, "child": childOutput}
	var parentSawFailure bool
	report, err := Execute(context.Background(), input, func(_ context.Context, in NodeRunInput) (NodeRunOutput, error) {
		if in.Node.ID == "parent" {
			parentSawFailure = len(in.Children) == 1 && in.Children[0].Result.Outcome == OutcomeFailed
		}
		return outputs[in.Node.ID], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !parentSawFailure {
		t.Fatal("parent callback did not receive the failed direct child outcome")
	}
	if got := findNode(t, report.Evaluation, "parent").Outcome; got != OutcomeFailed {
		t.Fatalf("parent outcome = %q, want failed", got)
	}
}

func TestExecuteDistinguishesCheckerFailureFromSkippedWork(t *testing.T) {
	checks := testChecks("unit")
	failedOutput, failedCurrent := runFixture(t, "failed", []string{"app"}, checks, OutcomeFailed)
	_, skippedCurrent := runFixture(t, "skipped", []string{"app"}, checks, OutcomePassed)
	input := RunInput{Graph: runGraph([]string{"failed", "skipped"},
		Node{ID: "failed", ScopeIDs: []string{"app"}, RequiredChecks: checks},
		Node{ID: "skipped", ScopeIDs: []string{"app"}, RequiredChecks: checks},
	), Current: map[string]records.Freshness{"failed": failedCurrent, "skipped": skippedCurrent}}
	report, err := Execute(context.Background(), input, func(_ context.Context, in NodeRunInput) (NodeRunOutput, error) {
		if in.Node.ID == "skipped" {
			return NodeRunOutput{NodeID: "skipped", Disposition: RunSkipped, Reason: "prerequisite unavailable"}, nil
		}
		return failedOutput, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Invocations[0].Status != InvocationCheckerFailed || report.Invocations[0].Outcome != OutcomeFailed {
		t.Fatalf("checker failure invocation = %#v", report.Invocations[0])
	}
	if report.Invocations[1].Status != InvocationSkipped || report.Invocations[1].Outcome != OutcomeIncomplete {
		t.Fatalf("skipped invocation = %#v", report.Invocations[1])
	}
}

func TestExecuteCancellationStopsBeforeParent(t *testing.T) {
	parentChecks, childChecks := testChecks("compose"), testChecks("unit")
	_, parentCurrent := runFixture(t, "parent", []string{"app"}, parentChecks, OutcomePassed)
	childOutput, childCurrent := runFixture(t, "child", []string{"app"}, childChecks, OutcomePassed)
	input := RunInput{Graph: runGraph([]string{"parent"},
		Node{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: parentChecks},
		Node{ID: "child", ScopeIDs: []string{"app"}, RequiredChecks: childChecks},
	), Current: map[string]records.Freshness{"parent": parentCurrent, "child": childCurrent}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	report, err := Execute(ctx, input, func(_ context.Context, in NodeRunInput) (NodeRunOutput, error) {
		calls++
		cancel()
		return childOutput, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
	if calls != 1 || len(report.Invocations) != 2 || report.Invocations[1].Status != InvocationCancelled {
		t.Fatalf("calls/report = %d, %#v", calls, report.Invocations)
	}
	if report.Evaluation.Roots[0].Outcome != OutcomeIncomplete {
		t.Fatalf("cancelled root outcome = %q", report.Evaluation.Roots[0].Outcome)
	}
}

func TestExecuteRejectsRunnerNodeScopeAndCheckMismatches(t *testing.T) {
	checks := testChecks("expected")
	output, current := runFixture(t, "node", []string{"app"}, checks, OutcomePassed)
	otherChecks := testChecks("other")
	otherScopeOutput, _ := runFixture(t, "node", []string{"elsewhere"}, checks, OutcomePassed)
	otherCheckOutput, _ := runFixture(t, "node", []string{"app"}, otherChecks, OutcomePassed)
	for _, test := range []struct {
		name   string
		output NodeRunOutput
		want   string
	}{
		{name: "node", output: func() NodeRunOutput { value := output; value.NodeID = "elsewhere"; return value }(), want: "scheduled node"},
		{name: "scope", output: otherScopeOutput, want: "scope"},
		{name: "checks", output: otherCheckOutput, want: "checks"},
	} {
		t.Run(test.name, func(t *testing.T) {
			report, err := Execute(context.Background(), RunInput{
				Graph:   runGraph([]string{"node"}, Node{ID: "node", ScopeIDs: []string{"app"}, RequiredChecks: checks}),
				Current: map[string]records.Freshness{"node": current},
			}, func(context.Context, NodeRunInput) (NodeRunOutput, error) { return test.output, nil })
			if err != nil {
				t.Fatal(err)
			}
			if report.Invocations[0].Status != InvocationInvalidOutput || report.Evaluation.Roots[0].Outcome != OutcomeIncomplete {
				t.Fatalf("invocation/evaluation = %#v / %#v", report.Invocations[0], report.Evaluation.Roots[0])
			}
			matched := false
			for _, cause := range report.Invocations[0].Causes {
				matched = matched || strings.Contains(cause.Message, test.want)
			}
			if !matched {
				t.Fatalf("causes = %#v, want message containing %q", report.Invocations[0].Causes, test.want)
			}
		})
	}
}

func TestExecuteRetriesAreBoundedAndInputDigestIsCanonical(t *testing.T) {
	checks := testChecks("check")
	unusedChecks := testChecks("unused")
	output, current := runFixture(t, "node", []string{"app"}, checks, OutcomePassed)
	_, unusedCurrent := runFixture(t, "unused", []string{"app"}, unusedChecks, OutcomePassed)
	node := Node{ID: "node", ScopeIDs: []string{"app"}, RequiredChecks: checks}
	unused := Node{ID: "unused", ScopeIDs: []string{"app"}, RequiredChecks: unusedChecks}
	input := RunInput{
		Graph:   runGraph([]string{"node"}, node, unused),
		Current: map[string]records.Freshness{"node": current, "unused": unusedCurrent}, Retries: MaxRetries,
	}
	attempts := 0
	runner := func(_ context.Context, _ NodeRunInput) (NodeRunOutput, error) {
		attempts++
		if attempts <= MaxRetries {
			return NodeRunOutput{}, errors.New("temporary runner failure")
		}
		return output, nil
	}
	first, err := Execute(context.Background(), input, runner)
	if err != nil {
		t.Fatal(err)
	}
	if attempts != MaxRetries+1 || first.Invocations[0].Attempts != MaxRetries+1 || first.Invocations[0].Status != InvocationCompleted {
		t.Fatalf("attempts/invocation = %d / %#v", attempts, first.Invocations[0])
	}
	if first.InputDigest == "" {
		t.Fatal("input digest is empty")
	}
	input.Graph.Nodes[0], input.Graph.Nodes[1] = input.Graph.Nodes[1], input.Graph.Nodes[0]
	second, err := Execute(context.Background(), input, func(_ context.Context, _ NodeRunInput) (NodeRunOutput, error) {
		return output, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.InputDigest != second.InputDigest {
		t.Fatalf("input digest changed for identical fixed inputs: %q != %q", first.InputDigest, second.InputDigest)
	}
	input.Retries = MaxRetries + 1
	if _, err := Execute(context.Background(), input, func(context.Context, NodeRunInput) (NodeRunOutput, error) {
		t.Fatal("runner called with retries over the bound")
		return NodeRunOutput{}, nil
	}); err == nil {
		t.Fatal("expected excessive retry validation error")
	}
}

func TestExecutionOrderUsesChildFirstSharedScopesWithoutEvidence(t *testing.T) {
	identity := records.CheckIdentity{ID: "own", Version: "1", Digest: "sha256:" + strings.Repeat("a", 64)}
	graph := Input{RootIDs: []string{"parent"}, Nodes: []Node{
		{ID: "parent", ScopeIDs: []string{"a", "b"}, Children: []string{"right", "left"}, RequiredChecks: []records.CheckIdentity{identity}},
		{ID: "left", ScopeIDs: []string{"a"}, Children: []string{"leaf"}, RequiredChecks: []records.CheckIdentity{identity}},
		{ID: "right", ScopeIDs: []string{"a", "b"}, Children: []string{"leaf"}, RequiredChecks: []records.CheckIdentity{identity}},
		{ID: "leaf", ScopeIDs: []string{"a"}, RequiredChecks: []records.CheckIdentity{identity}},
	}}
	order, err := ExecutionOrder(graph)
	if err != nil || strings.Join(order, ",") != "leaf,left,right,parent" {
		t.Fatalf("child-first order: %v %v", order, err)
	}
	graph.Nodes[3].Children = []string{"parent"}
	if _, err := ExecutionOrder(graph); err == nil {
		t.Fatal("cyclic execution order accepted")
	}
}
