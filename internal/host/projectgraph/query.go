package projectgraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectknowledge"
)

type Action string

const (
	ActionGraph     Action = "graph"
	ActionRelations Action = "relations"
	ActionExplain   Action = "explain"
	ActionTrace     Action = "trace"
	ActionHistory   Action = "history"
	ActionCoverage  Action = "coverage"
)

type Request struct {
	Action        Action `json:"action"`
	TargetID      string `json:"targetId,omitempty"`
	Reverse       bool   `json:"reverse,omitempty"`
	Bidirectional bool   `json:"bidirectional,omitempty"`
	MaxDepth      int    `json:"maxDepth,omitempty"`
	MaxSteps      int    `json:"maxSteps,omitempty"`
	MaxResults    int    `json:"maxResults,omitempty"`
}

type Result struct {
	Binding        projectknowledge.Binding     `json:"binding"`
	QueryVersion   string                       `json:"queryVersion"`
	QueryDigest    string                       `json:"queryDigest"`
	Provisional    bool                         `json:"provisional"`
	CaptureDigest  string                       `json:"captureDigest,omitempty"`
	SourceBindings []RecordBinding              `json:"sourceBindings,omitempty"`
	Action         Action                       `json:"action"`
	Coverage       *Coverage                    `json:"coverage,omitempty"`
	Nodes          []projectknowledge.Node      `json:"nodes,omitempty"`
	Edges          []projectknowledge.Edge      `json:"edges,omitempty"`
	Trace          *projectknowledge.WalkResult `json:"trace,omitempty"`
}

// Query answers read-only questions from exactly one immutable scoped view.
// All graph and witness output carries the binding of the selected data.
func Query(view *View, request Request) (Result, error) {
	result, err := execute(view, request)
	if err != nil {
		return Result{}, err
	}
	result.QueryVersion = "projectgraph/query/v1"
	payload, _ := json.Marshal(struct {
		Version string
		Request Request
		Result  Result
	}{result.QueryVersion, request, result})
	sum := sha256.Sum256(payload)
	result.QueryDigest = "sha256:" + hex.EncodeToString(sum[:])
	return result, nil
}

func execute(view *View, request Request) (Result, error) {
	if view == nil || view.index == nil {
		return Result{}, errors.New("project graph view is required")
	}
	result := Result{Binding: view.index.Binding(), Action: request.Action, Provisional: view.provisional, CaptureDigest: view.captureDigest, SourceBindings: append([]RecordBinding(nil), view.sourceBindings...)}
	switch request.Action {
	case ActionGraph:
		result.Nodes = view.index.Nodes()
		result.Edges = view.index.Edges()
		return result, nil
	case ActionRelations, ActionExplain:
		if request.TargetID == "" {
			return Result{}, errors.New("targetId is required")
		}
		node, err := view.index.Node(request.TargetID)
		if err != nil {
			return Result{}, projectknowledge.ErrNotFound
		}
		incoming, err := view.index.Incoming(request.TargetID)
		if err != nil {
			return Result{}, projectknowledge.ErrNotFound
		}
		outgoing, err := view.index.Outgoing(request.TargetID)
		if err != nil {
			return Result{}, projectknowledge.ErrNotFound
		}
		result.Nodes = []projectknowledge.Node{node}
		result.Edges = dedupeResultEdges(append(incoming, outgoing...))
		return result, nil
	case ActionTrace:
		if request.TargetID == "" {
			return Result{}, errors.New("targetId is required")
		}
		depth, steps, count := request.MaxDepth, request.MaxSteps, request.MaxResults
		if depth == 0 {
			depth = 8
		}
		if steps == 0 {
			steps = 5000
		}
		if count == 0 {
			count = 500
		}
		if request.Reverse && request.Bidirectional {
			return Result{}, errors.New("reverse and bidirectional are mutually exclusive")
		}
		walk, err := view.index.Walk(projectknowledge.WalkRequest{Start: request.TargetID, Reverse: request.Reverse, Bidirectional: request.Bidirectional, MaxDepth: depth, MaxSteps: steps, MaxResults: count})
		if err != nil {
			if errors.Is(err, projectknowledge.ErrNotFound) {
				return Result{}, projectknowledge.ErrNotFound
			}
			return Result{}, err
		}
		result.Trace = &walk
		return result, nil
	case ActionHistory:
		if request.TargetID == "" {
			return Result{}, errors.New("targetId is required")
		}
		node, err := view.index.Node(request.TargetID)
		if err != nil {
			for _, candidate := range view.index.Nodes() {
				if candidate.Kind != "IdentityChange" {
					continue
				}
				raw, ok := candidate.Properties["previous"]
				if !ok {
					continue
				}
				var previous core.DefinitionIdentity
				if json.Unmarshal(raw, &previous) == nil && previous.Key() == request.TargetID {
					result.Nodes = append(result.Nodes, candidate)
					out, _ := view.index.Outgoing(candidate.ID)
					result.Edges = dedupeResultEdges(out)
					for _, edge := range result.Edges {
						related, e := view.index.Node(edge.To)
						if e == nil {
							result.Nodes = append(result.Nodes, related)
						}
					}
					result.Nodes = dedupeResultNodes(result.Nodes)
					return result, nil
				}
			}
			return Result{}, projectknowledge.ErrNotFound
		}
		incoming, _ := view.index.Incoming(request.TargetID)
		outgoing, _ := view.index.Outgoing(request.TargetID)
		for _, candidate := range view.index.Nodes() {
			if candidate.Kind != "IdentityChange" && candidate.Kind != "Decision" {
				continue
			}
			in, _ := view.index.Incoming(candidate.ID)
			out, _ := view.index.Outgoing(candidate.ID)
			for _, e := range append(in, out...) {
				if e.From == request.TargetID || e.To == request.TargetID {
					result.Nodes = append(result.Nodes, candidate)
					result.Edges = append(result.Edges, e)
				}
			}
		}
		// Model history is a bounded one-hop view around this visible current
		// definition: include only explicitly linked events, their resolution,
		// and briefings that explicitly contain those events. Never follow a
		// resolution's run reference into operational records.
		for _, candidate := range view.index.Nodes() {
			if candidate.Kind != "ModelHistoryEvent" {
				continue
			}
			in, _ := view.index.Incoming(candidate.ID)
			out, _ := view.index.Outgoing(candidate.ID)
			var definitionEdge *projectknowledge.Edge
			for n := range out {
				if out[n].Property == "changedDefinition" && out[n].To == request.TargetID {
					e := out[n]
					definitionEdge = &e
					break
				}
			}
			if definitionEdge == nil {
				continue
			}
			result.Nodes = append(result.Nodes, candidate)
			result.Edges = append(result.Edges, *definitionEdge)
			for _, edge := range out {
				if edge.Property != "hasResolution" {
					continue
				}
				result.Edges = append(result.Edges, edge)
				if resolution, e := view.index.Node(edge.To); e == nil {
					result.Nodes = append(result.Nodes, resolution)
				}
			}
			for _, edge := range in {
				if edge.Property != "containsEvent" {
					continue
				}
				result.Edges = append(result.Edges, edge)
				if briefing, e := view.index.Node(edge.From); e == nil {
					result.Nodes = append(result.Nodes, briefing)
				}
			}
		}
		result.Nodes = append(result.Nodes, node)
		result.Nodes = dedupeResultNodes(result.Nodes)
		result.Edges = dedupeResultEdges(append(append(result.Edges, incoming...), outgoing...))
		return result, nil
	case ActionCoverage:
		coverage := view.coverage
		result.Coverage = &coverage
		return result, nil
	default:
		return Result{}, errors.New("unsupported project graph action")
	}
}

func dedupeResultNodes(nodes []projectknowledge.Node) []projectknowledge.Node {
	seen := map[string]bool{}
	out := make([]projectknowledge.Node, 0, len(nodes))
	for _, n := range nodes {
		if !seen[n.ID] {
			seen[n.ID] = true
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func dedupeResultEdges(edges []projectknowledge.Edge) []projectknowledge.Edge {
	type key struct{ from, to, property string }
	seen := map[key]bool{}
	out := make([]projectknowledge.Edge, 0, len(edges))
	for _, e := range edges {
		k := key{e.From, e.To, e.Property}
		if !seen[k] {
			seen[k] = true
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].Property != out[j].Property {
			return out[i].Property < out[j].Property
		}
		return out[i].To < out[j].To
	})
	return out
}
