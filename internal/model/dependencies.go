package model

import (
	"fmt"
	"sort"
)

// dependencyGraph records the known logical graph separately from verified
// direct bindings. Logical edges support cycle proofs even when a version is
// wrong; direct edges require an exact, complete binding at both ends.
type dependencyGraph struct {
	edges    [][]int
	direct   [][]int
	status   []Status
	findings []identityFinding
}

// bindDependencies binds only explicitly supplied model metadata. It never
// selects a candidate by order, normalization, or a preferred exact spelling
// when another supplied identity collides under the contract comparison.
func bindDependencies(scopes []sourceScope) dependencyGraph {
	n := len(scopes)
	g := dependencyGraph{
		edges: make([][]int, n), direct: make([][]int, n),
		status: make([]Status, n),
	}
	for i := range scopes {
		if scopes[i].manifestReady && scopes[i].inventoryComplete {
			g.status[i] = Passed
		} else {
			g.status[i] = NotCompleted
		}
	}
	knownIdentities := true
	byKey := make(map[string][]int, n)
	for i, scope := range scopes {
		if scope.modelID == "" {
			knownIdentities = false
			continue
		}
		byKey[nameKey(scope.modelID)] = append(byKey[nameKey(scope.modelID)], i)
	}
	addFinding := func(index int, code, message string) {
		g.findings = append(g.findings, identityFinding{scope: index, diagnostic: Diagnostic{
			Code: code, Message: message, Source: source(scopes[index].origin, scopes[index].manifest),
		}})
		g.status[index] = Failed
	}
	for _, candidates := range byKey {
		if len(candidates) < 2 {
			continue
		}
		for _, index := range candidates {
			addFinding(index, "dependency.duplicate_model", fmt.Sprintf("Model ID %q collides with another supplied model identity.", scopes[index].modelID))
		}
	}
	for i, scope := range scopes {
		if i > 0 && scope.manifestReady && (!scope.versionDeclared || scope.version.kind != StringValue || scope.version.text == "") {
			addFinding(i, "dependency.provided_version", "A supplied dependency model must declare a nonempty exact version string.")
		}
		seenEdges, seenDirect := map[int]bool{}, map[int]bool{}
		for _, requirement := range scope.requirements {
			if requirement.version.kind != StringValue || requirement.version.text == "" {
				addFinding(i, "dependency.version", fmt.Sprintf("Dependency %q must request a nonempty exact version string.", requirement.id))
				continue
			}
			candidates := byKey[nameKey(requirement.id)]
			if len(candidates) > 1 {
				addFinding(i, "dependency.ambiguous", fmt.Sprintf("Dependency ID %q matches multiple supplied model identities.", requirement.id))
				continue
			}
			if len(candidates) == 0 {
				if knownIdentities {
					addFinding(i, "dependency.missing", fmt.Sprintf("Dependency model %q is not present in the supplied model set.", requirement.id))
				} else if g.status[i] != Failed {
					g.status[i] = NotCompleted
				}
				continue
			}
			j := candidates[0]
			candidate := scopes[j]
			if candidate.modelID != requirement.id {
				addFinding(i, "dependency.spelling", fmt.Sprintf("Dependency ID %q must use the exact declared spelling %q.", requirement.id, candidate.modelID))
				continue
			}
			if !seenEdges[j] {
				g.edges[i] = append(g.edges[i], j)
				seenEdges[j] = true
			}
			if !candidate.versionDeclared || candidate.version.kind != StringValue || candidate.version.text == "" {
				if candidate.manifestReady && j == 0 {
					addFinding(j, "dependency.target_version", fmt.Sprintf("Local target model %q has no valid exact version string required by its consumer.", candidate.modelID))
				} else if g.status[i] != Failed {
					g.status[i] = NotCompleted
				}
				continue
			}
			if candidate.version.text != requirement.version.text {
				addFinding(i, "dependency.version_mismatch", fmt.Sprintf("Dependency %q requires exact version %q, but the supplied model declares %q.", requirement.id, requirement.version.text, candidate.version.text))
				continue
			}
			if scope.manifestReady && candidate.manifestReady {
				if !seenDirect[j] {
					g.direct[i] = append(g.direct[i], j)
					seenDirect[j] = true
				}
			} else if g.status[i] != Failed {
				g.status[i] = NotCompleted
			}
		}
		sort.Ints(g.edges[i])
		sort.Ints(g.direct[i])
	}
	markCycles(scopes, &g, addFinding)
	propagateDependencyStatus(&g)
	sort.Slice(g.findings, func(i, j int) bool {
		a, b := g.findings[i], g.findings[j]
		if a.scope != b.scope {
			return a.scope < b.scope
		}
		if a.diagnostic.Code != b.diagnostic.Code {
			return a.diagnostic.Code < b.diagnostic.Code
		}
		return a.diagnostic.Message < b.diagnostic.Message
	})
	return g
}

// markCycles uses iterative Kosaraju passes so graph depth does not consume the
// Go call stack. Every actual member receives a finding with the sorted IDs of
// its strongly connected component.
func markCycles(scopes []sourceScope, g *dependencyGraph, addFinding func(int, string, string)) {
	n := len(g.edges)
	reverse := make([][]int, n)
	for from, targets := range g.edges {
		for _, to := range targets {
			reverse[to] = append(reverse[to], from)
		}
	}
	visited := make([]bool, n)
	type frame struct{ node, next int }
	order := make([]int, 0, n)
	for root := 0; root < n; root++ {
		if visited[root] {
			continue
		}
		visited[root] = true
		stack := []frame{{node: root}}
		for len(stack) > 0 {
			last := len(stack) - 1
			f := &stack[last]
			if f.next < len(g.edges[f.node]) {
				next := g.edges[f.node][f.next]
				f.next++
				if !visited[next] {
					visited[next] = true
					stack = append(stack, frame{node: next})
				}
				continue
			}
			order = append(order, f.node)
			stack = stack[:last]
		}
	}
	for i := range visited {
		visited[i] = false
	}
	for oi := len(order) - 1; oi >= 0; oi-- {
		root := order[oi]
		if visited[root] {
			continue
		}
		component := []int{root}
		visited[root] = true
		for cursor := 0; cursor < len(component); cursor++ {
			for _, next := range reverse[component[cursor]] {
				if !visited[next] {
					visited[next] = true
					component = append(component, next)
				}
			}
		}
		cyclic := len(component) > 1
		if !cyclic {
			for _, next := range g.edges[root] {
				if next == root {
					cyclic = true
					break
				}
			}
		}
		if !cyclic {
			continue
		}
		ids := make([]string, 0, len(component))
		for _, index := range component {
			ids = append(ids, scopes[index].modelID)
		}
		sort.Strings(ids)
		message := fmt.Sprintf("Dependency cycle includes model IDs %q.", ids)
		sort.Ints(component)
		for _, index := range component {
			addFinding(index, "dependency.cycle", message)
		}
	}
}

// A known dependent inherits failure or incompleteness from its dependency.
// Failure outranks incomplete metadata and the reverse-edge queue visits only
// nodes whose status can still change.
func propagateDependencyStatus(g *dependencyGraph) {
	reverse := make([][]int, len(g.edges))
	for from, targets := range g.edges {
		for _, to := range targets {
			reverse[to] = append(reverse[to], from)
		}
	}
	queue := make([]int, len(g.status))
	for i := range queue {
		queue[i] = i
	}
	for head := 0; head < len(queue); head++ {
		dependency := queue[head]
		for _, dependent := range reverse[dependency] {
			next := g.status[dependent]
			if g.status[dependency] == Failed {
				next = Failed
			} else if g.status[dependency] == NotCompleted && next == Passed {
				next = NotCompleted
			}
			if next != g.status[dependent] {
				g.status[dependent] = next
				queue = append(queue, dependent)
			}
		}
	}
}
