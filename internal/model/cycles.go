package model

import (
	"fmt"
	"sort"
	"strings"
)

// Successors returns the Packages p depends on: its explicit edges and its
// parent, the implicit edge that scope by location creates.
func (m *Model) Successors(p string) []string {
	var out []string
	for to := range m.Edges[p] {
		out = append(out, to)
	}
	if p != RootPackage {
		if parent := ParentOf(p); m.Edges[p][parent] == nil {
			out = append(out, parent)
		}
	}
	sort.Strings(out)
	return out
}

// checkCycles reports every strongly connected set of Packages as one cycle,
// with one concrete path through it.
func (c *compiler) checkCycles() {
	for _, scc := range c.m.Cycles() {
		in := map[string]bool{}
		for _, p := range scc {
			in[p] = true
		}
		steps, file, line := c.m.cyclePath(scc[0], in)
		var names []string
		for _, p := range scc {
			names = append(names, PackageName(p))
		}
		c.diags.Add("package.cycle", file, line, "the Packages %s depend on each other in a cycle: %s. Move the shared concept into a common parent, or invert one dependency", strings.Join(names, ", "), strings.Join(steps, "; "))
	}
}

// Cycles returns each set of Packages that depend on each other in a cycle,
// sorted, in a stable order: the sets package.cycle reports. A cycle may run
// through the implicit edge from a Package to its parent.
func (m *Model) Cycles() [][]string {
	var out [][]string
	for _, scc := range m.components() {
		if len(scc) > 1 {
			out = append(out, scc)
		}
	}
	return out
}

// components returns the strongly connected components of the Package graph
// (Tarjan), each sorted, in a stable order.
func (m *Model) components() [][]string {
	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var out [][]string
	next := 0
	var visit func(string)
	visit = func(v string) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range m.Successors(v) {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sort.Strings(scc)
			out = append(out, scc)
		}
	}
	for _, p := range m.SortedPackages() {
		if _, seen := index[p]; !seen {
			visit(p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

// cyclePath finds the shortest path from start back to start inside one
// component and describes each step; it also returns where the first explicit
// link of the path is written.
func (m *Model) cyclePath(start string, in map[string]bool) ([]string, string, int) {
	prev := map[string]string{}
	queue := []string{start}
	seen := map[string]bool{}
	var end string
	for len(queue) > 0 && end == "" {
		v := queue[0]
		queue = queue[1:]
		for _, w := range m.Successors(v) {
			if !in[w] {
				continue
			}
			if w == start {
				end = v
				break
			}
			if !seen[w] {
				seen[w] = true
				prev[w] = v
				queue = append(queue, w)
			}
		}
	}
	path := []string{start}
	for v := end; v != start; v = prev[v] {
		path = append([]string{v}, path...)
	}
	path = append([]string{start}, path...)
	var steps []string
	file, line := "", 0
	for i := 0; i+1 < len(path); i++ {
		from, to := path[i], path[i+1]
		uses := m.Edges[from][to]
		if len(uses) == 0 {
			steps = append(steps, fmt.Sprintf("%s lies inside %s", PackageName(from), PackageName(to)))
			continue
		}
		u := uses[0]
		steps = append(steps, fmt.Sprintf("%s links [[%s]]", u.From, u.Target))
		if file == "" {
			file, line = u.File, u.Line
		}
	}
	return steps, file, line
}
