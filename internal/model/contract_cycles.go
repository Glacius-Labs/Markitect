package model

// Only mandatory object-field edges participate. Neither ordinary references
// nor optional edges require recursive value expansion.
func (c *contractCompiler) checkObjectCycles() {
	var nodes []Identity
	indices := map[Identity]int{}
	var addresses []definitionAddress
	for model, scope := range c.scopes {
		for index, definition := range scope.definitions {
			contract := c.result.definitions[definition.identity]
			if contract != nil && contract.property != nil && contract.property.valueType == "object" {
				indices[definition.identity] = len(nodes)
				nodes = append(nodes, definition.identity)
				addresses = append(addresses, definitionAddress{model, index})
			}
		}
	}
	edges := make([][]int, len(nodes))
	for index, id := range nodes {
		for _, field := range c.result.definitions[id].property.fields {
			if target, found := indices[field]; found {
				property := c.result.definitions[field].property
				if property.min != nil && property.min.Sign() > 0 {
					edges[index] = append(edges[index], target)
				}
			}
		}
	}
	for _, component := range cyclicComponents(edges) {
		for _, index := range component {
			c.finding(addresses[index], "contract.object_cycle", "This object contract participates in a cycle made entirely of mandatory object fields.", Value{})
		}
	}
}

// Iterative Kosaraju avoids consuming the Go call stack on a long contract
// graph. Return only actual cycle members, including singleton self-edges.
func cyclicComponents(edges [][]int) [][]int {
	reverse := make([][]int, len(edges))
	for from, targets := range edges {
		for _, to := range targets {
			reverse[to] = append(reverse[to], from)
		}
	}
	type frame struct{ node, next int }
	visited := make([]bool, len(edges))
	var order []int
	for root := range edges {
		if visited[root] {
			continue
		}
		visited[root] = true
		stack := []frame{{node: root}}
		for len(stack) > 0 {
			last := len(stack) - 1
			f := &stack[last]
			if f.next < len(edges[f.node]) {
				next := edges[f.node][f.next]
				f.next++
				if !visited[next] {
					visited[next] = true
					stack = append(stack, frame{node: next})
				}
			} else {
				order = append(order, f.node)
				stack = stack[:last]
			}
		}
	}
	clear(visited)
	var cycles [][]int
	for oi := len(order) - 1; oi >= 0; oi-- {
		root := order[oi]
		if visited[root] {
			continue
		}
		visited[root] = true
		component := []int{root}
		for cursor := 0; cursor < len(component); cursor++ {
			for _, next := range reverse[component[cursor]] {
				if !visited[next] {
					visited[next] = true
					component = append(component, next)
				}
			}
		}
		cyclic := len(component) > 1
		for _, next := range edges[root] {
			cyclic = cyclic || next == root
		}
		if cyclic {
			cycles = append(cycles, component)
		}
	}
	return cycles
}
