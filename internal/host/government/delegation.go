package government

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

const DelegationPlanVersion = "markitect.government-delegation-plan/v1alpha1"

// DelegationLimits are explicit preflight bounds. Zero is invalid: callers must
// consciously select finite limits. The hard ceilings keep a malformed caller
// configuration from turning planning into unbounded fanout.
type DelegationLimits struct {
	MaxDepth  int `json:"maxDepth" yaml:"maxDepth"`
	MaxFanout int `json:"maxFanout" yaml:"maxFanout"`
	MaxCalls  int `json:"maxCalls" yaml:"maxCalls"`
}

const (
	maxDelegationDepth  = 32
	maxDelegationFanout = 32
	maxDelegationCalls  = 4096
)

// DelegationNode represents one participating Area in the frozen responsibility
// hierarchy. Work, Actions, and Mandates are local only. Subjects and Paths
// include descendants to bind each invocation's read context; descendant paths
// do not become writer permissions for this node.
type DelegationNode struct {
	Area     core.DefinitionIdentity   `json:"area" yaml:"area"`
	Depth    int                       `json:"depth" yaml:"depth"`
	Work     Work                      `json:"work" yaml:"work"`
	Subjects []core.DefinitionIdentity `json:"subjects" yaml:"subjects"`
	Paths    []string                  `json:"paths" yaml:"paths"`
	Actions  []string                  `json:"actions" yaml:"actions"`
	Mandates []core.DefinitionIdentity `json:"mandates" yaml:"mandates"`
	Children []DelegationNode          `json:"children" yaml:"children"`
}

// DelegationPlan is a deterministic, read-only expansion of a frozen Plan.
// It contains no actor results and grants no authority beyond the Plan's
// existing writer assignments and explicit prior Mandates.
type DelegationPlan struct {
	APIVersion     string           `json:"apiVersion" yaml:"apiVersion"`
	Status         string           `json:"status" yaml:"status"`
	ModelDigest    string           `json:"modelDigest" yaml:"modelDigest"`
	PlanDigest     string           `json:"planDigest" yaml:"planDigest"`
	Limits         DelegationLimits `json:"limits" yaml:"limits"`
	EstimatedCalls int              `json:"estimatedCalls" yaml:"estimatedCalls"`
	Root           DelegationNode   `json:"root" yaml:"root"`
	Findings       []Finding        `json:"findings" yaml:"findings"`
	Digest         string           `json:"digest" yaml:"digest"`
}

// BuildDelegationPlan expands only Areas that own selected work and their
// required ancestors. Each changed path remains on its one declared writer.
// A participating Area receives one separate review call for its local and
// parent-integration obligations; only Areas with direct writer paths receive
// an executor call. The caller binds the distinct review role and its inputs.
func BuildDelegationPlan(m Model, p Plan, limits DelegationLimits) DelegationPlan {
	dp := DelegationPlan{
		APIVersion:  DelegationPlanVersion,
		Status:      "planned",
		ModelDigest: m.Digest,
		PlanDigest:  p.Digest,
		Limits:      limits,
		Findings:    append([]Finding(nil), p.Findings...),
	}
	add := func(code, subject, detail string) {
		dp.Findings = append(dp.Findings, Finding{Code: code, Subject: subject, Detail: detail})
	}
	if p.Status == "blocked" || len(p.Findings) != 0 {
		dp.Status = "blocked"
	}
	if limits.MaxDepth < 1 || limits.MaxDepth > maxDelegationDepth {
		add("delegation.limit.depth", "", fmt.Sprintf("maxDepth must be 1..%d", maxDelegationDepth))
	}
	if limits.MaxFanout < 1 || limits.MaxFanout > maxDelegationFanout {
		add("delegation.limit.fanout", "", fmt.Sprintf("maxFanout must be 1..%d", maxDelegationFanout))
	}
	if limits.MaxCalls < 1 || limits.MaxCalls > maxDelegationCalls {
		add("delegation.limit.calls", "", fmt.Sprintf("maxCalls must be 1..%d", maxDelegationCalls))
	}
	if len(dp.Findings) != 0 {
		dp.Status = "blocked"
		return finishDelegationPlan(dp)
	}

	rootDef, ok := m.byID[m.Root.Key()]
	if !ok || rootDef.Kind != "Area" {
		add("delegation.root", m.Root.Key(), "active root must resolve to an Area")
		return finishDelegationPlan(dp)
	}

	nodes := map[string]*DelegationNode{}
	workAreas := map[string]bool{}
	parents := map[string]string{}
	areas := map[string]core.DefinitionIdentity{}
	getNode := func(id core.DefinitionIdentity) *DelegationNode {
		key := id.Key()
		if nodes[key] == nil {
			nodes[key] = &DelegationNode{Area: id}
			areas[key] = id
		}
		return nodes[key]
	}
	getNode(m.Root).Work = Work{Area: m.Root}
	for _, d := range m.Canonical.Definitions {
		if d.APIVersion == APIVersion && d.Kind == "Area" {
			id := d.Identity()
			parents[id.Key()] = optionalIdentity(d.Spec["parent"])
			areas[id.Key()] = id
		}
	}

	// Resolve work Areas through the frozen parent chain, retaining structural
	// ancestors as integration nodes. Reject disconnected or cyclic chains even
	// though Compile normally catches them; callers may supply a constructed Model.
	for _, work := range p.Work {
		areaKey := work.Area.Key()
		if _, exists := areas[areaKey]; !exists {
			add("delegation.area", areaKey, "work Area does not resolve in the frozen model")
			continue
		}
		seen := map[string]bool{}
		cur := areaKey
		depth := 0
		overDepth := false
		overCalls := false
		for cur != "" {
			if seen[cur] {
				add("delegation.cycle", areaKey, "Area hierarchy contains a cycle")
				break
			}
			seen[cur] = true
			depth++
			if depth > limits.MaxDepth {
				add("delegation.depth", areaKey, fmt.Sprintf("Area chain exceeds maxDepth %d", limits.MaxDepth))
				overDepth = true
				break
			}
			id, exists := areas[cur]
			if !exists {
				add("delegation.disconnected", areaKey, "Area hierarchy contains an unresolved parent")
				break
			}
			if nodes[cur] == nil && len(nodes) >= limits.MaxCalls {
				add("delegation.calls", "", fmt.Sprintf("participating Area count exceeds maxCalls %d before execution", limits.MaxCalls))
				overCalls = true
				break
			}
			getNode(id)
			if cur == m.Root.Key() {
				break
			}
			cur = parents[cur]
		}
		if !overDepth && !overCalls && cur != m.Root.Key() {
			add("delegation.disconnected", areaKey, "work Area does not descend from the active root")
			continue
		}
		if overDepth || overCalls {
			break
		}
		node := getNode(work.Area)
		if workAreas[areaKey] {
			add("delegation.work.duplicate", areaKey, "Plan contains more than one local Work entry for an Area")
			continue
		}
		workAreas[areaKey] = true
		node.Work = cloneWork(work)
	}

	if len(dp.Findings) != 0 {
		dp.Status = "blocked"
		return finishDelegationPlan(dp)
	}

	// Apply fanout and minimum-call bounds before recursively copying a tree or
	// constructing aggregate scopes. One review per participating Area is the
	// lower bound; every Area with local writer files also requires an executor.
	childCounts := map[string]int{}
	for key := range nodes {
		if key != m.Root.Key() {
			childCounts[parents[key]]++
		}
	}
	minimumCalls := len(nodes)
	for _, node := range nodes {
		if len(node.Work.Paths) > 0 {
			minimumCalls++
		}
	}
	for parent, count := range childCounts {
		if count > limits.MaxFanout {
			add("delegation.fanout", parent, fmt.Sprintf("participating child count %d exceeds maxFanout %d", count, limits.MaxFanout))
		}
	}
	if minimumCalls > limits.MaxCalls {
		add("delegation.calls", "", fmt.Sprintf("minimum actor calls %d exceeds maxCalls %d", minimumCalls, limits.MaxCalls))
	}
	if len(dp.Findings) != 0 {
		dp.Status = "blocked"
		return finishDelegationPlan(dp)
	}

	// Build a reverse index of direct mandates from the active model. A mandate
	// is evidence only when its exact identity, Area, action and subjects match.
	mandatesByArea := map[string][]core.Definition{}
	declaredWriters := map[string]string{}
	artifactSubjects := map[string][]core.DefinitionIdentity{}
	for _, d := range m.Canonical.Definitions {
		if d.APIVersion == APIVersion && d.Kind == "Mandate" {
			area := identity(d.Spec["area"]).Key()
			mandatesByArea[area] = append(mandatesByArea[area], d)
		}
		if d.APIVersion == APIVersion && d.Kind == "Artifact" && d.Spec["writer"] != nil {
			declaredWriters[strings.ToLower(fmt.Sprint(d.Spec["path"]))] = identity(d.Spec["writer"]).Key()
		}
		if d.APIVersion == APIVersion && d.Kind == "Realization" {
			artifact := m.byID[identity(d.Spec["artifact"]).Key()]
			path := strings.ToLower(fmt.Sprint(artifact.Spec["path"]))
			artifactSubjects[path] = append(artifactSubjects[path], identity(d.Spec["subject"]))
		}
	}
	for _, node := range nodes {
		areaKey := node.Area.Key()
		node.Mandates = append([]core.DefinitionIdentity(nil), node.Work.Mandates...)
		for _, mandateID := range node.Work.Mandates {
			found := false
			for _, mandate := range mandatesByArea[areaKey] {
				if mandate.Identity().Key() != mandateID.Key() {
					continue
				}
				found = true
			}
			if !found {
				add("delegation.mandate", mandateID.Key(), "local Work references no exact prior Mandate for its Area")
			}
		}
		// Actions describe this frozen task, not every power retained in the
		// underlying prior Mandate. The mandate identity remains available for
		// authority context, while structural integration nodes receive no write
		// action. Only the exact caller-selected model path uses amend-model;
		// every other selected path still requires implement.
		for _, path := range node.Work.Paths {
			action := delegationPathAction(p, path)
			if !contains(node.Actions, action) {
				node.Actions = append(node.Actions, action)
			}
		}
		sort.Strings(node.Actions)
		sortIDs(node.Work.Subjects)
		sortIDs(node.Work.Mandates)
		sort.Strings(node.Work.Paths)
	}

	// Every actual local implementation scope must be covered by the exact
	// mandate already named by Plan. No authority is copied from descendants.
	for _, node := range nodes {
		if len(node.Work.Paths) == 0 && len(node.Work.Subjects) == 0 {
			continue
		}
		if len(node.Work.Paths) == 0 {
			action := p.Action
			if action == "" {
				action = "implement"
			}
			for _, subject := range node.Work.Subjects {
				if !localMandateAllows(mandatesByArea[node.Area.Key()], node.Work.Mandates, subject, action) {
					add("delegation.authority", subject.Key(), "accountable Area lacks an explicit prior "+action+" Mandate")
				}
			}
		}
		for _, path := range node.Work.Paths {
			modeledSubjects := artifactSubjects[strings.ToLower(path)]
			if len(modeledSubjects) == 0 {
				add("delegation.realization", path, "local writer path has no modeled subject realization")
				continue
			}
			for _, subject := range modeledSubjects {
				action := delegationPathAction(p, path)
				if !localMandateAllows(mandatesByArea[node.Area.Key()], node.Work.Mandates, subject, action) {
					add("delegation.authority", subject.Key(), "local writer lacks an explicit prior "+action+" Mandate for "+path)
				}
				if !containsIdentity(node.Work.Subjects, subject) {
					add("delegation.scope.missing", path, "local work omits modeled subject "+subject.Key())
				}
			}
		}
	}

	// Check that every participating node is still connected before producing
	// the value tree. The tree itself is assembled below in stable key order.
	for key := range nodes {
		if key == m.Root.Key() {
			continue
		}
		parentKey := parents[key]
		parent := nodes[parentKey]
		if parent == nil {
			add("delegation.disconnected", key, "participating Area has no participating parent")
		}
	}
	// Copy recursively from the indexed nodes, avoiding pointers in the public contract.
	var build func(string, int) DelegationNode
	build = func(key string, depth int) DelegationNode {
		n := *nodes[key]
		n.Depth = depth
		n.Children = nil
		childKeys := make([]string, 0)
		for childKey := range nodes {
			if parents[childKey] == key {
				childKeys = append(childKeys, childKey)
			}
		}
		sort.Strings(childKeys)
		for _, childKey := range childKeys {
			n.Children = append(n.Children, build(childKey, depth+1))
		}
		return n
	}
	dp.Root = build(m.Root.Key(), 1)

	pathOwner := map[string]string{}
	var summarize func(*DelegationNode) ([]core.DefinitionIdentity, []string, int)
	summarize = func(node *DelegationNode) ([]core.DefinitionIdentity, []string, int) {
		node.Subjects = append([]core.DefinitionIdentity(nil), node.Work.Subjects...)
		node.Paths = append([]string(nil), node.Work.Paths...)
		calls := 1 // independent review call for every participating Area
		if len(node.Work.Paths) != 0 {
			calls++ // one execution call only for an Area with direct writer files
		}
		for _, child := range node.Children {
			childSubjects, childPaths, childCalls := summarize(&child)
			calls += childCalls
			node.Subjects = append(node.Subjects, childSubjects...)
			node.Paths = append(node.Paths, childPaths...)
			// Values are copied, so store the recursively summarized child back.
			for i := range node.Children {
				if node.Children[i].Area.Key() == child.Area.Key() {
					node.Children[i] = child
					break
				}
			}
		}
		sortIDs(node.Subjects)
		node.Subjects = uniqueIDs(node.Subjects)
		sort.Strings(node.Paths)
		node.Paths = uniqueStrings(node.Paths)
		for _, path := range node.Work.Paths {
			if !SafePath(path) {
				add("delegation.path", path, "local writer path is unsafe")
				continue
			}
			if declaredWriters[strings.ToLower(path)] != node.Area.Key() {
				add("delegation.writer", path, "local path is not assigned to this Area by the frozen model")
			}
			folded := strings.ToLower(path)
			if prior, exists := pathOwner[folded]; exists && prior != node.Area.Key() {
				add("delegation.writer.collision", path, "path is assigned to both "+prior+" and "+node.Area.Key())
			} else if exists {
				add("delegation.writer.duplicate", path, "path is assigned more than once to one Area")
			} else {
				pathOwner[folded] = node.Area.Key()
			}
		}
		return node.Subjects, node.Paths, calls
	}
	_, _, dp.EstimatedCalls = summarize(&dp.Root)

	// Validate delegation between each participating parent and child using only
	// explicit active Mandates. Aggregate Subjects describe task/read scope; they
	// do not grant a structural parent writer rights.
	var validateEdges func(DelegationNode)
	validateEdges = func(parent DelegationNode) {
		if len(parent.Children) > limits.MaxFanout {
			add("delegation.fanout", parent.Area.Key(), fmt.Sprintf("participating child count %d exceeds maxFanout %d", len(parent.Children), limits.MaxFanout))
		}
		for _, child := range parent.Children {
			if !subsetIDs(child.Subjects, parent.Subjects) {
				add("delegation.scope.expansion", child.Area.Key(), "child task scope exceeds parent aggregate task scope")
			}
			var checkScope func(DelegationNode)
			checkScope = func(descendant DelegationNode) {
				for _, path := range descendant.Work.Paths {
					for _, subject := range artifactSubjects[strings.ToLower(path)] {
						action := delegationPathAction(p, path)
						if !mandateChainCovers(m, parent.Area, []core.DefinitionIdentity{subject}, action) {
							add("delegation.authority.expansion", child.Area.Key(), "parent has no explicit prior "+action+" Mandate covering "+subject.Key())
						}
					}
				}
				for _, nested := range descendant.Children {
					checkScope(nested)
				}
			}
			checkScope(child)
			validateEdges(child)
		}
	}
	validateEdges(dp.Root)
	if len(dp.Findings) != 0 {
		dp.Status = "blocked"
	}
	return finishDelegationPlan(dp)
}

func delegationPathAction(p Plan, path string) string {
	if p.Action == "amend-model" && p.ModelPath != "" && path == p.ModelPath {
		return "amend-model"
	}
	return "implement"
}

func cloneWork(w Work) Work {
	w.Subjects = append([]core.DefinitionIdentity(nil), w.Subjects...)
	w.Paths = append([]string(nil), w.Paths...)
	w.Mandates = append([]core.DefinitionIdentity(nil), w.Mandates...)
	return w
}

func localMandateAllows(definitions []core.Definition, mandateIDs []core.DefinitionIdentity, subject core.DefinitionIdentity, action string) bool {
	for _, d := range definitions {
		if !containsIdentity(mandateIDs, d.Identity()) || !contains(stringsList(d.Spec["actions"]), action) {
			continue
		}
		if subsetIDs([]core.DefinitionIdentity{subject}, identities(d.Spec["scope"])) {
			return true
		}
	}
	return false
}

func mandateChainCovers(m Model, area core.DefinitionIdentity, subjects []core.DefinitionIdentity, action string) bool {
	for _, d := range m.Canonical.Definitions {
		if d.APIVersion == APIVersion && d.Kind == "Mandate" && identity(d.Spec["area"]).Key() == area.Key() && contains(stringsList(d.Spec["actions"]), action) && subsetIDs(subjects, identities(d.Spec["scope"])) {
			return true
		}
	}
	return false
}

func containsIdentity(ids []core.DefinitionIdentity, want core.DefinitionIdentity) bool {
	for _, id := range ids {
		if id.Key() == want.Key() {
			return true
		}
	}
	return false
}

func uniqueIDs(ids []core.DefinitionIdentity) []core.DefinitionIdentity {
	if len(ids) < 2 {
		return ids
	}
	out := ids[:1]
	for _, id := range ids[1:] {
		if out[len(out)-1].Key() != id.Key() {
			out = append(out, id)
		}
	}
	return out
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}

func delegationTreeDepth(root DelegationNode) int {
	max := root.Depth
	for _, child := range root.Children {
		if childDepth := delegationTreeDepth(child); childDepth > max {
			max = childDepth
		}
	}
	return max
}

func finishDelegationPlan(p DelegationPlan) DelegationPlan {
	sortFindings(p.Findings)
	p.Digest = ""
	p.Digest = Digest(p)
	return p
}
