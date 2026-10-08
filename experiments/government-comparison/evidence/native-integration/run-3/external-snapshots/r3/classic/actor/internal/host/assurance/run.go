package assurance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

const MaxRetries = 2

const (
	RunCompleted = "completed"
	RunSkipped   = "skipped"
)

const (
	InvocationCompleted     = "completed"
	InvocationCheckerFailed = "checker-failed"
	InvocationEscalated     = "escalated"
	InvocationIncomplete    = "incomplete"
	InvocationSkipped       = "skipped"
	InvocationRunnerError   = "runner-error"
	InvocationInvalidOutput = "invalid-output"
	InvocationCancelled     = "cancelled"
)

// RunInput supplies the fixed graph and immutable current bindings captured by
// the Host. Retries is the number of additional attempts after a runner error.
type RunInput struct {
	Graph   Input
	Current map[string]records.Freshness
	Retries int
}

// NodeRunInput gives a runner only the node it owns, its Host-supplied current
// binding, and direct child evidence. Child verification reasons and cause
// messages are omitted; no transitive child payload is included.
type NodeRunInput struct {
	Node     Node
	Current  records.Freshness
	Children []ChildRunInput
}

// ChildRunInput contains a direct child's composed result and, when available,
// its validated direct evidence. Evidence is copied and verification Reason is
// cleared before it reaches the parent runner.
type ChildRunInput struct {
	Result   NodeResult
	Evidence *Evidence
}

// NodeRunOutput is the evidence produced by one runner call. Current freshness
// is intentionally not returned by the runner; Execute binds the Host's copy.
type NodeRunOutput struct {
	NodeID      string
	Disposition string
	Reason      string
	Record      records.ProjectionRecord
	Result      records.VerificationResult
}

// Runner performs the scoped work for exactly one declared node. It may inspect
// direct child outcomes and evidence, then execute that node's own checks. An
// explicit skipped disposition records incomplete work without inventing a
// checker result.
type Runner func(context.Context, NodeRunInput) (NodeRunOutput, error)

// Invocation is one scheduled node. Attempts includes retries and is zero for
// nodes left uncalled after cancellation. Outcome is the composed node result;
// Status describes this node's own runner/check evidence.
type Invocation struct {
	NodeID   string
	Attempts int
	Status   string
	Outcome  string
	Causes   []Cause
}

// RunReport separates deterministic inputs, ordering, causes, and composed
// outcomes from elapsed wall time.
type RunReport struct {
	InputDigest string
	Invocations []Invocation
	Evaluation  Report
	WallTime    time.Duration
}

// Execute invokes each reachable node once in deterministic postorder. A DAG
// node shared by multiple parents is run once. Structural graph validation and
// Host binding validation finish before the runner is called.
func Execute(ctx context.Context, input RunInput, runner Runner) (RunReport, error) {
	if ctx == nil {
		return RunReport{}, errors.New("context is required")
	}
	if runner == nil {
		return RunReport{}, errors.New("runner is required")
	}
	if input.Retries < 0 || input.Retries > MaxRetries {
		return RunReport{}, fmt.Errorf("retries must be between 0 and %d", MaxRetries)
	}
	if len(input.Graph.Evidence) != 0 {
		return RunReport{}, errors.New("execution input cannot contain pre-supplied evidence")
	}

	graph := canonicalGraph(input.Graph)
	if _, err := Evaluate(graph); err != nil {
		return RunReport{}, err
	}
	nodes := make(map[string]Node, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	order := postorder(graph)
	current, err := canonicalCurrent(input.Current)
	if err != nil {
		return RunReport{}, err
	}
	for _, nodeID := range order {
		binding, ok := current[nodeID]
		if !ok {
			return RunReport{}, fmt.Errorf("Host current binding is missing for reachable node %q", nodeID)
		}
		if !sameChecks(nodes[nodeID].RequiredChecks, binding.Checks) {
			return RunReport{}, fmt.Errorf("Host current checks do not exactly match node %q", nodeID)
		}
		if strings.TrimSpace(binding.RecordID) == "" || strings.TrimSpace(binding.Revision) == "" ||
			strings.TrimSpace(binding.ModelDigest) == "" || strings.TrimSpace(binding.TargetSnapshotDigest) == "" ||
			strings.TrimSpace(binding.Verifier.ID) == "" || strings.TrimSpace(binding.Verifier.Version) == "" ||
			strings.TrimSpace(binding.Verifier.Digest) == "" {
			return RunReport{}, fmt.Errorf("Host current binding for node %q is incomplete", nodeID)
		}
	}

	digest, err := runInputDigest(graph, current, input.Retries)
	if err != nil {
		return RunReport{}, err
	}
	start := time.Now()
	report := RunReport{InputDigest: digest, Invocations: make([]Invocation, 0, len(order))}
	evidenceByNode := make(map[string]Evidence, len(order))
	runCauses := make(map[string][]Cause, len(order))
	cancelled := false

	for index, nodeID := range order {
		if ctx.Err() != nil {
			cancelled = true
			for _, remaining := range order[index:] {
				report.Invocations = append(report.Invocations, Invocation{
					NodeID: remaining, Status: InvocationCancelled, Outcome: OutcomeIncomplete,
					Causes: []Cause{{Code: "execution-cancelled", NodeID: remaining, Message: "node was not called because execution was cancelled"}},
				})
			}
			break
		}

		runInput, err := makeNodeRunInput(graph, nodes[nodeID], current[nodeID], evidenceByNode)
		if err != nil {
			return report, err
		}
		invocation := Invocation{NodeID: nodeID}
		var output NodeRunOutput
		var runnerErr error
		for attempt := 0; attempt <= input.Retries; attempt++ {
			if ctx.Err() != nil {
				break
			}
			invocation.Attempts++
			output, runnerErr = runner(ctx, cloneNodeRunInput(runInput))
			if runnerErr == nil {
				break
			}
			if ctx.Err() != nil || attempt == input.Retries {
				break
			}
		}
		if ctx.Err() != nil {
			cancelled = true
			cause := Cause{Code: "execution-cancelled", NodeID: nodeID, Message: "execution was cancelled while running this node"}
			runCauses[nodeID] = append(runCauses[nodeID], cause)
			invocation.Status = InvocationCancelled
		} else if runnerErr != nil {
			cause := Cause{Code: "runner-error", NodeID: nodeID, Message: runnerErr.Error()}
			runCauses[nodeID] = append(runCauses[nodeID], cause)
			invocation.Status = InvocationRunnerError
		} else if output.Disposition == RunSkipped && output.NodeID != nodeID {
			cause := Cause{Code: "invalid-run-output", NodeID: nodeID, Message: fmt.Sprintf("runner returned node %q for scheduled node %q", output.NodeID, nodeID)}
			runCauses[nodeID] = append(runCauses[nodeID], cause)
			invocation.Status = InvocationInvalidOutput
		} else if output.Disposition == RunSkipped {
			cause := Cause{Code: "runner-skipped", NodeID: nodeID, Message: output.Reason}
			runCauses[nodeID] = append(runCauses[nodeID], cause)
			invocation.Status = InvocationSkipped
		} else if output.Disposition != RunCompleted {
			cause := Cause{Code: "invalid-run-disposition", NodeID: nodeID, Message: fmt.Sprintf("unsupported disposition %q", output.Disposition)}
			runCauses[nodeID] = append(runCauses[nodeID], cause)
			invocation.Status = InvocationInvalidOutput
		} else if evidence, validationErr := validateRunnerOutput(nodes[nodeID], current[nodeID], output); validationErr != nil {
			cause := Cause{Code: "invalid-run-output", NodeID: nodeID, Message: validationErr.Error()}
			runCauses[nodeID] = append(runCauses[nodeID], cause)
			invocation.Status = InvocationInvalidOutput
		} else {
			evidenceByNode[nodeID] = evidence
			ownOutcome, evalErr := evaluateOwnOutcome(nodes[nodeID], evidence)
			if evalErr != nil {
				return report, evalErr
			}
			switch ownOutcome {
			case OutcomeFailed:
				invocation.Status = InvocationCheckerFailed
			case OutcomeEscalated:
				invocation.Status = InvocationEscalated
			case OutcomeIncomplete:
				invocation.Status = InvocationIncomplete
			default:
				invocation.Status = InvocationCompleted
			}
		}
		report.Invocations = append(report.Invocations, invocation)
		if cancelled {
			for _, remaining := range order[index+1:] {
				report.Invocations = append(report.Invocations, Invocation{
					NodeID: remaining, Status: InvocationCancelled, Outcome: OutcomeIncomplete,
					Causes: []Cause{{Code: "execution-cancelled", NodeID: remaining, Message: "node was not called because execution was cancelled"}},
				})
			}
			break
		}
	}

	evaluation, evalErr := Evaluate(Input{RootIDs: graph.RootIDs, Nodes: graph.Nodes, Evidence: evidenceSlice(order, evidenceByNode)})
	if evalErr != nil {
		return report, evalErr
	}
	report.Evaluation = evaluation
	for i := range report.Invocations {
		invocation := &report.Invocations[i]
		result := resultFor(evaluation, invocation.NodeID)
		invocation.Outcome = result.Outcome
		invocation.Causes = append([]Cause(nil), result.Causes...)
		invocation.Causes = append(invocation.Causes, runCauses[invocation.NodeID]...)
		sort.Slice(invocation.Causes, func(i, j int) bool { return causeLess(invocation.Causes[i], invocation.Causes[j]) })
	}
	report.WallTime = time.Since(start)
	if cancelled {
		return report, ctx.Err()
	}
	return report, nil
}

// ExecutionOrder returns the validated deterministic child-first order without
// inventing materialization or verification evidence.
func ExecutionOrder(input Input) ([]string, error) {
	graph := canonicalGraph(input)
	report, err := Evaluate(graph)
	if err != nil {
		return nil, err
	}
	if len(report.Nodes) != len(graph.Nodes) {
		return nil, errors.New("all execution scopes must be reachable from declared roots")
	}
	return postorder(graph), nil
}

func canonicalGraph(input Input) Input {
	graph := Input{RootIDs: append([]string(nil), input.RootIDs...), Nodes: make([]Node, len(input.Nodes))}
	sort.Strings(graph.RootIDs)
	for i, node := range input.Nodes {
		graph.Nodes[i] = cloneNode(node)
		sort.Strings(graph.Nodes[i].ScopeIDs)
		sort.Strings(graph.Nodes[i].Children)
		sort.Slice(graph.Nodes[i].RequiredChecks, func(a, b int) bool {
			return graph.Nodes[i].RequiredChecks[a].ID < graph.Nodes[i].RequiredChecks[b].ID
		})
	}
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].ID < graph.Nodes[j].ID })
	return graph
}

func canonicalCurrent(current map[string]records.Freshness) (map[string]records.Freshness, error) {
	copy := make(map[string]records.Freshness, len(current))
	for nodeID, binding := range current {
		binding.Checks = append([]records.CheckIdentity(nil), binding.Checks...)
		sort.Slice(binding.Checks, func(i, j int) bool { return binding.Checks[i].ID < binding.Checks[j].ID })
		if err := validateRequiredChecks(binding.Checks); err != nil {
			return nil, fmt.Errorf("Host current binding for node %q: %w", nodeID, err)
		}
		copy[nodeID] = binding
	}
	return copy, nil
}

func postorder(graph Input) []string {
	nodes := make(map[string]Node, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	seen := make(map[string]bool, len(graph.Nodes))
	order := make([]string, 0, len(graph.Nodes))
	var visit func(string)
	visit = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		for _, childID := range nodes[id].Children {
			visit(childID)
		}
		order = append(order, id)
	}
	for _, rootID := range graph.RootIDs {
		visit(rootID)
	}
	return order
}

func makeNodeRunInput(graph Input, node Node, current records.Freshness, evidence map[string]Evidence) (NodeRunInput, error) {
	input := NodeRunInput{Node: cloneNode(node), Current: cloneFreshness(current)}
	for _, childID := range node.Children {
		childReport, err := Evaluate(Input{RootIDs: []string{childID}, Nodes: graph.Nodes, Evidence: evidenceSlice(postorder(graph), evidence)})
		if err != nil {
			return NodeRunInput{}, err
		}
		child := resultFor(childReport, childID)
		child.Causes = append([]Cause(nil), child.Causes...)
		for i := range child.Causes {
			child.Causes[i].Message = ""
		}
		childInput := ChildRunInput{Result: child}
		if childEvidence, ok := evidence[childID]; ok {
			sanitized := cloneEvidence(childEvidence)
			sanitized.Result.Reason = ""
			childInput.Evidence = &sanitized
		}
		input.Children = append(input.Children, childInput)
	}
	return input, nil
}

func validateRunnerOutput(node Node, current records.Freshness, output NodeRunOutput) (Evidence, error) {
	if output.NodeID != node.ID {
		return Evidence{}, fmt.Errorf("runner returned node %q for scheduled node %q", output.NodeID, node.ID)
	}
	if err := records.ValidateProjectionRecord(output.Record); err != nil {
		return Evidence{}, fmt.Errorf("runner returned invalid projection record: %w", err)
	}
	if !sameStrings(node.ScopeIDs, output.Record.ScopeIDs) {
		return Evidence{}, errors.New("runner projection record scope does not exactly match the scheduled node")
	}
	if err := records.ValidateVerificationResult(output.Result); err != nil {
		return Evidence{}, fmt.Errorf("runner returned invalid verification result: %w", err)
	}
	if output.Result.RecordID != output.Record.ID || output.Result.Revision != output.Record.Revision ||
		output.Result.ModelDigest != output.Record.ModelDigest || output.Result.TargetSnapshotDigest != output.Record.TargetSnapshotDigest {
		return Evidence{}, errors.New("runner verification result does not bind to its projection record")
	}
	checks := make([]records.CheckIdentity, len(output.Result.Checks))
	for i, check := range output.Result.Checks {
		checks[i] = records.CheckIdentity{ID: check.ID, Version: check.Version, Digest: check.Digest}
	}
	if !sameChecks(node.RequiredChecks, checks) {
		return Evidence{}, errors.New("runner verification checks do not exactly match the scheduled node")
	}
	return Evidence{
		NodeID: node.ID, Record: cloneRecord(output.Record), Result: cloneVerificationResult(output.Result),
		Current: cloneFreshness(current),
	}, nil
}

func evaluateOwnOutcome(node Node, evidence Evidence) (string, error) {
	node.Children = nil
	report, err := Evaluate(Input{RootIDs: []string{node.ID}, Nodes: []Node{node}, Evidence: []Evidence{evidence}})
	if err != nil {
		return "", err
	}
	return report.Roots[0].Outcome, nil
}

func runInputDigest(graph Input, current map[string]records.Freshness, retries int) (string, error) {
	type binding struct {
		NodeID  string
		Current records.Freshness
	}
	bindings := make([]binding, 0, len(current))
	ids := make([]string, 0, len(current))
	for id := range current {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		bindings = append(bindings, binding{NodeID: id, Current: current[id]})
	}
	data, err := json.Marshal(struct {
		Graph   Input
		Current []binding
		Retries int
	}{Graph: graph, Current: bindings, Retries: retries})
	if err != nil {
		return "", fmt.Errorf("digest execution input: %w", err)
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func evidenceSlice(order []string, evidence map[string]Evidence) []Evidence {
	items := make([]Evidence, 0, len(evidence))
	for _, id := range order {
		if item, ok := evidence[id]; ok {
			items = append(items, cloneEvidence(item))
		}
	}
	return items
}

func resultFor(report Report, nodeID string) NodeResult {
	for _, node := range report.Nodes {
		if node.NodeID == nodeID {
			return node
		}
	}
	return NodeResult{NodeID: nodeID, Outcome: OutcomeIncomplete}
}

func cloneNode(node Node) Node {
	node.ScopeIDs = append([]string(nil), node.ScopeIDs...)
	node.Children = append([]string(nil), node.Children...)
	node.RequiredChecks = append([]records.CheckIdentity(nil), node.RequiredChecks...)
	return node
}

func cloneNodeRunInput(input NodeRunInput) NodeRunInput {
	input.Node = cloneNode(input.Node)
	input.Current = cloneFreshness(input.Current)
	input.Children = append([]ChildRunInput(nil), input.Children...)
	for i := range input.Children {
		input.Children[i].Result.ScopeIDs = append([]string(nil), input.Children[i].Result.ScopeIDs...)
		input.Children[i].Result.Children = append([]string(nil), input.Children[i].Result.Children...)
		input.Children[i].Result.Causes = append([]Cause(nil), input.Children[i].Result.Causes...)
		if input.Children[i].Evidence != nil {
			copy := cloneEvidence(*input.Children[i].Evidence)
			input.Children[i].Evidence = &copy
		}
	}
	return input
}

func cloneFreshness(current records.Freshness) records.Freshness {
	current.Checks = append([]records.CheckIdentity(nil), current.Checks...)
	return current
}

func cloneEvidence(evidence Evidence) Evidence {
	evidence.Record = cloneRecord(evidence.Record)
	evidence.Result = cloneVerificationResult(evidence.Result)
	evidence.Current = cloneFreshness(evidence.Current)
	return evidence
}

func cloneRecord(record records.ProjectionRecord) records.ProjectionRecord {
	record.ScopeIDs = append([]string(nil), record.ScopeIDs...)
	record.PolicyIDs = append([]string(nil), record.PolicyIDs...)
	record.Artifacts = append([]records.Artifact(nil), record.Artifacts...)
	return record
}

func cloneVerificationResult(result records.VerificationResult) records.VerificationResult {
	result.Checks = append([]records.CheckResult(nil), result.Checks...)
	return result
}
