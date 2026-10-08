package assurance

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

const (
	MaxNodes    = 128
	MaxDepth    = 8
	MaxScopeIDs = 128
	MaxChecks   = 128
)

const (
	OutcomePassed     = records.OutcomePassed
	OutcomeFailed     = records.OutcomeFailed
	OutcomeIncomplete = records.OutcomeIncomplete
	OutcomeEscalated  = records.OutcomeEscalated
)

var digestPattern = regexp.MustCompile("^sha256:[0-9a-f]{64}$")

// Node is one explicitly declared verification scope. Children are direct
// composition scopes, and RequiredChecks are this node's own obligations.
type Node struct {
	ID             string
	ScopeIDs       []string
	Children       []string
	RequiredChecks []records.CheckIdentity
}

// Evidence contains supplied operational facts. Current is the caller's
// explicit current binding for the referenced record and verification result.
type Evidence struct {
	NodeID  string
	Record  records.ProjectionRecord
	Result  records.VerificationResult
	Current records.Freshness
}

// Input names the roots whose assurance is requested. Nodes outside the
// reachable root subgraph are validated structurally but do not affect results.
type Input struct {
	RootIDs  []string
	Nodes    []Node
	Evidence []Evidence
}

// Cause points to exact supplied evidence or a direct child outcome. Child
// causes are also available on that child's NodeResult.
type Cause struct {
	Code     string
	NodeID   string
	ChildID  string
	RecordID string
	ResultID string
	Message  string
}

// NodeResult reports one reachable node and its direct children. Outcome is
// derived from its own evidence and every direct child outcome.
type NodeResult struct {
	NodeID   string
	ScopeIDs []string
	Children []string
	Outcome  string
	RecordID string
	ResultID string
	Causes   []Cause
}

// Report includes deterministic root results and the unique set of reachable
// node results, both sorted by node ID.
type Report struct {
	Roots []NodeResult
	Nodes []NodeResult
}

// Evaluate validates the finite declared graph and composes only explicitly
// requested roots. It performs no I/O and executes no checks.
func Evaluate(input Input) (Report, error) {
	if len(input.Nodes) == 0 || len(input.Nodes) > MaxNodes {
		return Report{}, fmt.Errorf("node count must be between 1 and %d", MaxNodes)
	}
	if len(input.RootIDs) == 0 {
		return Report{}, errors.New("at least one root ID is required")
	}

	nodes := make(map[string]Node, len(input.Nodes))
	for _, node := range input.Nodes {
		if !validText(node.ID) {
			return Report{}, errors.New("node ID must be nonempty text")
		}
		if _, exists := nodes[node.ID]; exists {
			return Report{}, fmt.Errorf("duplicate node %q", node.ID)
		}
		if len(node.ScopeIDs) == 0 || len(node.ScopeIDs) > MaxScopeIDs {
			return Report{}, fmt.Errorf("node %q scope ID count must be between 1 and %d", node.ID, MaxScopeIDs)
		}
		if len(node.RequiredChecks) == 0 || len(node.RequiredChecks) > MaxChecks {
			return Report{}, fmt.Errorf("node %q required check count must be between 1 and %d", node.ID, MaxChecks)
		}
		node.ScopeIDs = append([]string(nil), node.ScopeIDs...)
		sort.Strings(node.ScopeIDs)
		if err := uniqueTextIDs("scope ID", node.ScopeIDs); err != nil {
			return Report{}, fmt.Errorf("node %q: %w", node.ID, err)
		}
		node.Children = append([]string(nil), node.Children...)
		sort.Strings(node.Children)
		if err := uniqueTextIDs("child ID", node.Children); err != nil {
			return Report{}, fmt.Errorf("node %q: %w", node.ID, err)
		}
		node.RequiredChecks = append([]records.CheckIdentity(nil), node.RequiredChecks...)
		sort.Slice(node.RequiredChecks, func(i, j int) bool {
			return node.RequiredChecks[i].ID < node.RequiredChecks[j].ID
		})
		if err := validateRequiredChecks(node.RequiredChecks); err != nil {
			return Report{}, fmt.Errorf("node %q: %w", node.ID, err)
		}
		nodes[node.ID] = node
	}

	roots := append([]string(nil), input.RootIDs...)
	sort.Strings(roots)
	if err := uniqueTextIDs("root ID", roots); err != nil {
		return Report{}, err
	}
	for _, root := range roots {
		if _, ok := nodes[root]; !ok {
			return Report{}, fmt.Errorf("dangling root %q", root)
		}
	}

	allIDs := make([]string, 0, len(nodes))
	for id := range nodes {
		allIDs = append(allIDs, id)
	}
	sort.Strings(allIDs)
	for _, id := range allIDs {
		node := nodes[id]
		parentScopes := stringSet(node.ScopeIDs)
		for _, childID := range node.Children {
			child, ok := nodes[childID]
			if !ok {
				return Report{}, fmt.Errorf("node %q has dangling child %q", id, childID)
			}
			for _, scopeID := range child.ScopeIDs {
				if !parentScopes[scopeID] {
					return Report{}, fmt.Errorf("child %q scope %q escapes parent %q", childID, scopeID, id)
				}
			}
		}
	}

	// Validate cycles and maximum DAG height over the complete explicit input,
	// including currently unreachable nodes.
	colors := make(map[string]uint8, len(nodes))
	heights := make(map[string]int, len(nodes))
	var height func(string) (int, error)
	height = func(id string) (int, error) {
		switch colors[id] {
		case 1:
			return 0, fmt.Errorf("cycle detected at node %q", id)
		case 2:
			return heights[id], nil
		}
		colors[id] = 1
		longest := 1
		for _, childID := range nodes[id].Children {
			childHeight, err := height(childID)
			if err != nil {
				return 0, err
			}
			if childHeight+1 > longest {
				longest = childHeight + 1
			}
		}
		if longest > MaxDepth {
			return 0, fmt.Errorf("node %q exceeds maximum DAG depth %d", id, MaxDepth)
		}
		colors[id], heights[id] = 2, longest
		return longest, nil
	}
	for _, id := range allIDs {
		if _, err := height(id); err != nil {
			return Report{}, err
		}
	}

	if len(input.Evidence) > len(nodes) {
		return Report{}, errors.New("evidence count exceeds node count")
	}
	evidenceByNode := make(map[string]Evidence, len(input.Evidence))
	for _, evidence := range input.Evidence {
		if _, ok := nodes[evidence.NodeID]; !ok {
			return Report{}, fmt.Errorf("evidence references unknown node %q", evidence.NodeID)
		}
		if _, exists := evidenceByNode[evidence.NodeID]; exists {
			return Report{}, fmt.Errorf("duplicate evidence for node %q", evidence.NodeID)
		}
		evidenceByNode[evidence.NodeID] = evidence
	}

	own := make(map[string]ownResult, len(nodes))
	for _, id := range allIDs {
		evidence, present := evidenceByNode[id]
		own[id] = evaluateOwn(nodes[id], evidence, present)
	}

	reachable := make(map[string]NodeResult, len(nodes))
	var evaluateNode func(string) NodeResult
	evaluateNode = func(id string) NodeResult {
		if result, ok := reachable[id]; ok {
			return result
		}
		node := nodes[id]
		self := own[id]
		result := NodeResult{
			NodeID: id, ScopeIDs: append([]string(nil), node.ScopeIDs...),
			Children: append([]string(nil), node.Children...),
			Outcome:  self.outcome, RecordID: self.recordID, ResultID: self.resultID,
			Causes: append([]Cause(nil), self.causes...),
		}
		childOutcomes := make([]string, 0, len(node.Children))
		for _, childID := range node.Children {
			child := evaluateNode(childID)
			childOutcomes = append(childOutcomes, child.Outcome)
			if child.Outcome != OutcomePassed {
				result.Causes = append(result.Causes, Cause{
					Code: "child-" + child.Outcome, NodeID: id, ChildID: child.NodeID,
					RecordID: child.RecordID, ResultID: child.ResultID,
					Message: "direct child outcome is " + child.Outcome,
				})
			}
		}
		result.Outcome = rollup(self.outcome, childOutcomes)
		sort.Slice(result.Causes, func(i, j int) bool { return causeLess(result.Causes[i], result.Causes[j]) })
		reachable[id] = result
		return result
	}

	rootResults := make([]NodeResult, 0, len(roots))
	for _, root := range roots {
		rootResults = append(rootResults, evaluateNode(root))
	}
	reports := make([]NodeResult, 0, len(reachable))
	reachableIDs := make([]string, 0, len(reachable))
	for id := range reachable {
		reachableIDs = append(reachableIDs, id)
	}
	sort.Strings(reachableIDs)
	for _, id := range reachableIDs {
		reports = append(reports, reachable[id])
	}
	return Report{Roots: rootResults, Nodes: reports}, nil
}

type ownResult struct {
	outcome  string
	recordID string
	resultID string
	causes   []Cause
}

func evaluateOwn(node Node, evidence Evidence, present bool) ownResult {
	if !present {
		return ownResult{outcome: OutcomeIncomplete, causes: []Cause{{
			Code: "missing-evidence", NodeID: node.ID, Message: "node has no supplied projection and verification evidence",
		}}}
	}
	recordID, resultID := evidence.Record.ID, evidence.Result.ID
	incomplete := func(code, message string) ownResult {
		return ownResult{outcome: OutcomeIncomplete, recordID: recordID, resultID: resultID, causes: []Cause{{
			Code: code, NodeID: node.ID, RecordID: recordID, ResultID: resultID, Message: message,
		}}}
	}
	if err := records.ValidateProjectionRecord(evidence.Record); err != nil {
		return incomplete("invalid-projection-record", err.Error())
	}
	if !sameStrings(node.ScopeIDs, evidence.Record.ScopeIDs) {
		return incomplete("record-scope-mismatch", "projection record scope does not exactly match the declared node scope")
	}

	var causes []Cause
	baseOutcome := OutcomePassed
	switch evidence.Record.State {
	case records.StatePartialFailure:
		baseOutcome = OutcomeFailed
		causes = append(causes, Cause{
			Code: "materialization-failed", NodeID: node.ID, RecordID: recordID,
			Message: "projection record reports partial failure",
		})
	case records.StateEscalated:
		baseOutcome = OutcomeEscalated
		causes = append(causes, Cause{
			Code: "materialization-escalated", NodeID: node.ID, RecordID: recordID,
			Message: "projection record reports escalation",
		})
	case records.StateMaterializedUnverified:
	default:
		return incomplete("invalid-projection-state", fmt.Sprintf("unsupported projection state %q", evidence.Record.State))
	}

	if !sameChecks(node.RequiredChecks, evidence.Current.Checks) {
		causes = append(causes, Cause{
			Code: "check-declaration-mismatch", NodeID: node.ID, RecordID: recordID, ResultID: resultID,
			Message: "current freshness checks do not exactly match the node required checks",
		})
		return ownResult{outcome: rollup(baseOutcome, []string{OutcomeIncomplete}), recordID: recordID, resultID: resultID, causes: causes}
	}
	if err := records.ValidateVerificationFreshness(evidence.Result, evidence.Record, evidence.Current); err != nil {
		causes = append(causes, Cause{
			Code: "invalid-or-stale-verification", NodeID: node.ID, RecordID: recordID, ResultID: resultID,
			Message: err.Error(),
		})
		return ownResult{outcome: rollup(baseOutcome, []string{OutcomeIncomplete}), recordID: recordID, resultID: resultID, causes: causes}
	}

	resultOutcome := evidence.Result.Outcome
	if resultOutcome != OutcomePassed {
		causes = append(causes, Cause{
			Code: "verification-" + resultOutcome, NodeID: node.ID, RecordID: recordID, ResultID: resultID,
			Message: verificationMessage(evidence.Result),
		})
	}
	return ownResult{
		outcome:  rollup(baseOutcome, []string{resultOutcome}),
		recordID: recordID, resultID: resultID, causes: causes,
	}
}

func rollup(own string, children []string) string {
	failed, escalated, incomplete := own == OutcomeFailed, own == OutcomeEscalated, own == OutcomeIncomplete
	for _, outcome := range children {
		failed = failed || outcome == OutcomeFailed
		escalated = escalated || outcome == OutcomeEscalated
		incomplete = incomplete || outcome == OutcomeIncomplete
	}
	// This explicit order retains every cause in the report while assigning a
	// single status: known failure, escalation, missing/invalid evidence, pass.
	switch {
	case failed:
		return OutcomeFailed
	case escalated:
		return OutcomeEscalated
	case incomplete:
		return OutcomeIncomplete
	default:
		return OutcomePassed
	}
}

func verificationMessage(result records.VerificationResult) string {
	if strings.TrimSpace(result.Reason) != "" {
		return result.Reason
	}
	return "verification outcome is " + result.Outcome
}

func validateRequiredChecks(checks []records.CheckIdentity) error {
	seen := map[string]bool{}
	for _, check := range checks {
		if !validText(check.ID) || !validText(check.Version) {
			return errors.New("required check ID and version must be nonempty text")
		}
		if !digestPattern.MatchString(check.Digest) {
			return fmt.Errorf("required check %q digest must be sha256", check.ID)
		}
		if seen[check.ID] {
			return fmt.Errorf("duplicate required check %q", check.ID)
		}
		seen[check.ID] = true
	}
	return nil
}

func sameChecks(left, right []records.CheckIdentity) bool {
	if len(left) != len(right) {
		return false
	}
	a, b := append([]records.CheckIdentity(nil), left...), append([]records.CheckIdentity(nil), right...)
	sort.Slice(a, func(i, j int) bool { return a[i].ID < a[j].ID })
	sort.Slice(b, func(i, j int) bool { return b[i].ID < b[j].ID })
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	a, b := append([]string(nil), left...), append([]string(nil), right...)
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func uniqueTextIDs(kind string, values []string) error {
	for i, value := range values {
		if !validText(value) {
			return fmt.Errorf("%s must be nonempty text", kind)
		}
		if i > 0 && values[i-1] == value {
			return fmt.Errorf("duplicate %s %q", kind, value)
		}
	}
	return nil
}

func validText(value string) bool {
	return strings.TrimSpace(value) != ""
}

func causeLess(a, b Cause) bool {
	if a.NodeID != b.NodeID {
		return a.NodeID < b.NodeID
	}
	if a.ChildID != b.ChildID {
		return a.ChildID < b.ChildID
	}
	if a.ResultID != b.ResultID {
		return a.ResultID < b.ResultID
	}
	if a.Code != b.Code {
		return a.Code < b.Code
	}
	return a.Message < b.Message
}
