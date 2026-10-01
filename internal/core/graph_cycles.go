package core

import (
	"strings"
)

func (g *Graph) detectRuntimeCycles() {
	// Only concrete execution edges participate. Rule and implements links are
	// context edges and must not manufacture runtime cycles.
	adj := map[string][]string{}
	for _, source := range g.Resources {
		if source.Kind == "Project" || source.Kind == "Package" {
			continue
		}
		for _, ref := range source.Spec.Uses {
			target := g.resolveUse(source, ref)
			if target != nil && (target.Kind == "Workflow" || target.Kind == "Skill" || target.Kind == "Agent") {
				adj[source.GraphKey()] = append(adj[source.GraphKey()], target.GraphKey())
			}
		}
	}
	for _, relationship := range g.Relationships {
		if relationship.Relation == "selected-implementation" {
			adj[relationship.From] = append(adj[relationship.From], relationship.To)
		}
	}
	state, stack, reported := map[string]int{}, []string{}, map[string]bool{}
	var visit func(string)
	visit = func(key string) {
		state[key] = 1
		stack = append(stack, key)
		for _, next := range adj[key] {
			if state[next] == 0 {
				visit(next)
				continue
			}
			if state[next] == 1 {
				cycle := append([]string(nil), stack...)
				start := 0
				for i, item := range cycle {
					if item == next {
						start = i
						break
					}
				}
				cycle = append(cycle[start:], next)
				label := strings.Join(cycle, " -> ")
				if !reported[label] {
					reported[label] = true
					g.diag(g.Resources[key], "graph.cycle", "runtime dependency cycle: "+label)
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[key] = 2
	}
	for _, key := range sortedKeys(g.Resources) {
		if state[key] == 0 {
			visit(key)
		}
	}
}
