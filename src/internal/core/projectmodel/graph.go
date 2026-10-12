package projectmodel

import (
	"encoding/json"
	"errors"
	"sort"
)

// ErrNodeNotFound is returned for an ID the graph does not show, whether it
// does not exist or is not visible in the graph's scope.
var ErrNodeNotFound = errors.New("node not found")

// Trace limits: zero selects the default; larger values are capped.
const (
	defaultTraceDepth = 6
	maxTraceDepth     = 32
	defaultTraceNodes = 500
	maxTraceNodes     = 5000
)

// ProjectGraph is the whole project's knowledge graph: every definition the
// report carries, the inventory files, and every declared relation.
func ProjectGraph(r Report) KnowledgeGraph {
	g := graphBuilder{scope: "project", nodes: map[string]GraphNode{}, edges: map[GraphEdge]bool{}}
	for _, m := range r.Managers {
		g.manager(m)
	}
	for _, s := range r.Statements {
		g.statement(s, "statement")
	}
	for _, a := range r.Artifacts {
		g.artifact(a)
	}
	for _, c := range r.Checks {
		g.check(c)
	}
	for _, d := range r.Decisions {
		g.decision(d)
	}
	for _, f := range r.Files {
		g.node(GraphNode{Kind: "file", ID: f.Path, Name: f.Path, Owner: f.Owner})
		g.edge(f.Path, "owned by", f.Owner)
	}
	return g.build()
}

// ManagerGraph is one Manager's knowledge graph, built from its Context alone,
// so it shows exactly what Context shows: its own definitions, the public
// contracts they need, and its children without their instructions.
func ManagerGraph(c ManagerContext) KnowledgeGraph {
	g := graphBuilder{scope: c.Manager.ID, nodes: map[string]GraphNode{}, edges: map[GraphEdge]bool{}}
	g.manager(c.Manager)
	for _, child := range c.Children {
		g.manager(child)
	}
	for _, s := range c.Statements {
		g.statement(s, "statement")
	}
	for _, s := range c.Contracts {
		g.statement(s, "contract")
	}
	for _, a := range c.Artifacts {
		g.artifact(a)
	}
	for _, check := range append(append([]Check(nil), c.Checks...), c.ForeignChecks...) {
		g.check(check)
	}
	for _, d := range c.Decisions {
		g.decision(d)
	}
	return g.build()
}

type graphBuilder struct {
	scope string
	nodes map[string]GraphNode
	edges map[GraphEdge]bool
}

func (g *graphBuilder) node(n GraphNode) { g.nodes[n.ID] = n }

func (g *graphBuilder) edge(from, relation, to string) {
	if from != "" && to != "" {
		g.edges[GraphEdge{From: from, Relation: relation, To: to}] = true
	}
}

func (g *graphBuilder) manager(m Manager) {
	g.node(GraphNode{Kind: "manager", ID: m.ID, Name: m.Name})
	g.edge(m.ID, "parent", m.Parent)
}

func (g *graphBuilder) statement(s Statement, kind string) {
	g.node(GraphNode{Kind: kind, ID: s.ID, Name: s.Name, Owner: s.Owner})
	g.edge(s.ID, "owned by", s.Owner)
	for _, id := range s.Uses {
		g.edge(s.ID, "uses", id)
	}
	for _, id := range s.Requires {
		g.edge(s.ID, "requires", id)
	}
}

func (g *graphBuilder) artifact(a Artifact) {
	g.node(GraphNode{Kind: "artifact", ID: a.ID, Name: a.Name, Owner: a.Owner})
	g.edge(a.ID, "owned by", a.Owner)
	for _, id := range a.Realizes {
		g.edge(a.ID, "realizes", id)
	}
	for _, id := range a.Checks {
		g.edge(a.ID, "checked by", id)
	}
	for _, p := range a.Paths {
		g.edge(a.ID, "expects", p)
	}
}

func (g *graphBuilder) check(c Check) {
	g.node(GraphNode{Kind: "check", ID: c.ID, Name: c.Name, Owner: c.Owner})
	g.edge(c.ID, "owned by", c.Owner)
	for _, id := range c.Uses {
		g.edge(c.ID, "exercises", id)
	}
}

func (g *graphBuilder) decision(d Decision) {
	g.node(GraphNode{Kind: "decision", ID: d.ID, Name: d.Name, Owner: d.Owner})
	g.edge(d.ID, "owned by", d.Owner)
	g.edge(d.ID, "decides on", d.Subject)
	g.edge(d.ID, "actor", d.Actor)
}

// build adds a reference node, identity only, for every edge end that is not a
// full node, and sorts nodes and edges.
func (g *graphBuilder) build() KnowledgeGraph {
	out := KnowledgeGraph{Scope: g.scope, Nodes: []GraphNode{}, Edges: []GraphEdge{}}
	for e := range g.edges {
		for _, id := range []string{e.From, e.To} {
			if _, ok := g.nodes[id]; !ok {
				g.nodes[id] = GraphNode{Kind: referenceKind(id, e), ID: id, Reference: true}
			}
		}
		out.Edges = append(out.Edges, e)
	}
	for _, n := range g.nodes {
		out.Nodes = append(out.Nodes, n)
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].ID < out.Nodes[j].ID })
	sort.Slice(out.Edges, func(i, j int) bool {
		a, b := out.Edges[i], out.Edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		return a.To < b.To
	})
	out.Digest = digest(struct {
		Scope string
		Nodes []GraphNode
		Edges []GraphEdge
	}{out.Scope, out.Nodes, out.Edges})
	return out
}

// referenceKind names the kind of a node known only by an edge: a definition
// identity carries its kind, and an artifact's expected path is a path.
func referenceKind(id string, e GraphEdge) string {
	var identity [4]string
	if json.Unmarshal([]byte(id), &identity) == nil && identity[1] != "" {
		switch identity[1] {
		case managerKind:
			return "manager"
		case statementKind:
			return "statement"
		case artifactKind:
			return "artifact"
		case checkKind:
			return "check"
		case decisionKind:
			return "decision"
		}
	}
	if e.Relation == "expects" && e.To == id {
		return "path"
	}
	return "file"
}

// Trace walks the graph breadth first from one node and returns every node it
// reaches within the limits, each with one shortest witness path of edges.
// Edges keep their own direction; direction "in" follows them backwards and
// "both" follows either way. Result order and witnesses are deterministic.
func (g KnowledgeGraph) Trace(req TraceRequest) (TraceResult, error) {
	start := -1
	for i, n := range g.Nodes {
		if n.ID == req.From {
			start = i
		}
	}
	if start < 0 {
		return TraceResult{}, ErrNodeNotFound
	}
	depth, limit := req.MaxDepth, req.MaxNodes
	if depth <= 0 {
		depth = defaultTraceDepth
	}
	if limit <= 0 {
		limit = defaultTraceNodes
	}
	depth, limit = min(depth, maxTraceDepth), min(limit, maxTraceNodes)
	type step struct {
		next string
		edge GraphEdge
	}
	adjacent := map[string][]step{}
	for _, e := range g.Edges {
		if req.Direction != "in" {
			adjacent[e.From] = append(adjacent[e.From], step{e.To, e})
		}
		if req.Direction == "in" || req.Direction == "both" {
			adjacent[e.To] = append(adjacent[e.To], step{e.From, e})
		}
	}
	nodes := map[string]GraphNode{}
	for _, n := range g.Nodes {
		nodes[n.ID] = n
	}
	out := TraceResult{Scope: g.Scope, From: req.From, Complete: true}
	witness := map[string][]GraphEdge{req.From: {}}
	frontier := []string{req.From}
	out.Reached = append(out.Reached, TracedNode{Node: nodes[req.From], Witness: []GraphEdge{}})
	for d := 1; d <= depth && len(frontier) > 0; d++ {
		var next []string
		for _, id := range frontier {
			for _, s := range adjacent[id] {
				if _, seen := witness[s.next]; seen {
					continue
				}
				if len(out.Reached) >= limit {
					out.Complete = false
					return out, nil
				}
				witness[s.next] = append(append([]GraphEdge{}, witness[id]...), s.edge)
				out.Reached = append(out.Reached, TracedNode{Node: nodes[s.next], Depth: d, Witness: witness[s.next]})
				next = append(next, s.next)
			}
		}
		frontier = next
	}
	for _, id := range frontier {
		for _, s := range adjacent[id] {
			if _, seen := witness[s.next]; !seen {
				out.Complete = false
			}
		}
	}
	return out, nil
}
