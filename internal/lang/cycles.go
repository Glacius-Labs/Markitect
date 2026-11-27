package lang

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// CodeModuleCycle is the code of a group of modules that depend on each other in
// a cycle.
const CodeModuleCycle = "module.cycle"

// A module depends on another when one of its documents has a type line that
// names a document of the other, an extends line that names one, a property type
// line that links one, or, in a section that is a reference property of its
// type, a link to one. A link in a text property, in the text before the first
// section, in a section the type does not declare and every link of a README is
// a mention and counts for nothing. A target that is no document counts for
// nothing; a link that is internal to its module still counts, since it points
// from one module to the other.

// dependency is the first link that makes one module depend on another.
type dependency struct {
	file   string
	line   int
	target string
	// verb says what the link does, for a message.
	verb string
}

// The verbs of the four kinds of dependency.
const (
	verbTypeLine    = "is of type"
	verbExtendsLine = "extends"
	verbPropertyRef = "declares a reference to"
	verbReference   = "refers to"
)

func (a dependency) before(b dependency) bool {
	if a.file != b.file {
		return a.file < b.file
	}
	if a.line != b.line {
		return a.line < b.line
	}
	return a.target < b.target
}

type moduleEdge struct{ from, to string }

// moduleEdges returns each dependency between two different modules with the
// first link that causes it, by file, line and target.
func (s *Model) moduleEdges() map[moduleEdge]dependency {
	edges := map[moduleEdge]dependency{}
	for _, d := range s.Documents {
		from := path.Dir(d.Path)
		add := func(target string, line int, verb string) {
			if _, ok := s.byPath[target]; !ok {
				return
			}
			key := moduleEdge{from, path.Dir(target)}
			if key.from == key.to {
				return
			}
			dep := dependency{d.Path, line, target, verb}
			if old, ok := edges[key]; !ok || dep.before(old) {
				edges[key] = dep
			}
		}
		if d.HasType() {
			add(d.TypePath, d.TypeLine, verbTypeLine)
		}
		if d.HasExtends() {
			add(d.ExtendsPath, d.ExtendsLine, verbExtendsLine)
		}
		if d.holdsProperties() {
			for _, sec := range d.Sections {
				for _, r := range sec.Property.Refs {
					if !isReadmeDocument(s, r.Target) {
						add(r.Target, sec.Property.Line, verbPropertyRef)
					}
				}
			}
		}
		if !d.HasType() {
			continue
		}
		t, ok := s.Document(d.TypePath)
		if !ok {
			continue
		}
		for _, want := range s.ChainSections(t) {
			sec, has := d.Section(want.Name)
			if !has || s.PropertyType(t, want.Name).Kind != PropertyRefs {
				continue
			}
			for _, l := range referenceLinks(d, sec) {
				add(l.Target, l.Line, verbReference)
			}
		}
	}
	return edges
}

// moduleLess orders modules by path, the model folder first.
func moduleLess(a, b string) bool {
	if a == modelFolder || b == modelFolder {
		return a == modelFolder && b != modelFolder
	}
	return a < b
}

// moduleName names a module in a message.
func moduleName(m string) string {
	if m == modelFolder {
		return "(model folder)"
	}
	return m
}

// checkModuleCycles reports each group of modules that all reach each other,
// once, at the first link of one shortest cycle through the first module of the
// group in path order.
func (s *Model) checkModuleCycles() []Diagnostic {
	edges := s.moduleEdges()
	if len(edges) == 0 {
		return nil
	}
	adj := map[string][]string{}
	nodeSet := map[string]bool{}
	for e := range edges {
		adj[e.from] = append(adj[e.from], e.to)
		nodeSet[e.from], nodeSet[e.to] = true, true
	}
	var nodes []string
	for n := range nodeSet {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool { return moduleLess(nodes[i], nodes[j]) })
	for n := range adj {
		list := adj[n]
		sort.Slice(list, func(i, j int) bool { return moduleLess(list[i], list[j]) })
	}
	var out []Diagnostic
	for _, group := range stronglyConnected(nodes, adj) {
		if len(group) < 2 {
			continue
		}
		cycle := shortestCycle(group, adj)
		var names, steps []string
		for i, m := range cycle {
			names = append(names, moduleName(m))
			if i+1 < len(cycle) {
				dep := edges[moduleEdge{m, cycle[i+1]}]
				steps = append(steps, fmt.Sprintf("%s:%d %s %s", dep.file, dep.line, dep.verb, dep.target))
			}
		}
		first := edges[moduleEdge{cycle[0], cycle[1]}]
		out = append(out, Diagnostic{Path: first.file, Line: first.line, Code: CodeModuleCycle,
			Message: fmt.Sprintf("module cycle %s: %s", strings.Join(names, " -> "), strings.Join(steps, ", "))})
	}
	return out
}

// stronglyConnected returns the groups of nodes that all reach each other, each
// in path order, Tarjan's algorithm over nodes and successors in path order.
func stronglyConnected(nodes []string, adj map[string][]string) [][]string {
	index := map[string]int{}
	low := map[string]int{}
	on := map[string]bool{}
	var stack []string
	var groups [][]string
	next := 0
	var visit func(v string)
	visit = func(v string) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		on[v] = true
		for _, w := range adj[v] {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if on[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] != index[v] {
			return
		}
		var group []string
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			on[w] = false
			group = append(group, w)
			if w == v {
				break
			}
		}
		sort.Slice(group, func(i, j int) bool { return moduleLess(group[i], group[j]) })
		groups = append(groups, group)
	}
	for _, v := range nodes {
		if _, seen := index[v]; !seen {
			visit(v)
		}
	}
	return groups
}

// shortestCycle returns one shortest cycle through the first module of the
// group, as the modules from it back to it, the first module at both ends. A
// breadth-first search over successors in path order finds the same cycle every
// time.
func shortestCycle(group []string, adj map[string][]string) []string {
	in := map[string]bool{}
	for _, m := range group {
		in[m] = true
	}
	root := group[0]
	parent := map[string]string{}
	seen := map[string]bool{root: true}
	queue := []string{root}
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		for _, v := range adj[u] {
			if !in[v] {
				continue
			}
			if v == root {
				var back []string
				for m := u; m != root; m = parent[m] {
					back = append(back, m)
				}
				cycle := []string{root}
				for i := len(back) - 1; i >= 0; i-- {
					cycle = append(cycle, back[i])
				}
				return append(cycle, root)
			}
			if !seen[v] {
				seen[v] = true
				parent[v] = u
				queue = append(queue, v)
			}
		}
	}
	return nil
}
