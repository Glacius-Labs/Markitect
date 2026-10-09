// Package projectknowledge builds a bounded, immutable, read-only projection
// over an explicitly selected Core model and caller-supplied neutral facts.
// It is an index, not an authority for project policy, freshness, or impact.
package projectknowledge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func Build(model core.Model, facts ProjectFacts, scope Scope) (*Index, error) {
	if model.Digest == "" || facts.SnapshotDigest == "" || facts.Digest == "" || strings.TrimSpace(scope.ID) == "" {
		return nil, errors.New("model digest, snapshot digest, facts digest, and scope ID are required")
	}
	if len(model.Definitions) > maxNodes || len(model.Edges) > maxEdges || len(facts.Facts) > maxNodes || len(facts.Relations) > maxEdges {
		return nil, errors.New("supplied model or project facts exceed graph input limits")
	}
	if len(scope.Nodes) == 0 || len(scope.Nodes) > maxNodes || len(scope.Edges) > maxEdges {
		return nil, errors.New("scope must explicitly select a bounded non-empty node set and bounded edges")
	}
	bindingPayload := len(GraphVersion) + len(model.Digest) + len(model.Revision) + len(facts.SnapshotDigest) + len(facts.ProjectDigest) + len(facts.Digest) + len(scope.ID)
	if bindingPayload > maxPayloadBytes {
		return nil, errors.New("graph binding exceeds payload limit")
	}
	inputPayload := bindingPayload
	for _, selected := range scope.Nodes {
		if len(selected.Fields) > core.MaxPropertiesPerObject {
			return nil, errors.New("node field allowlist exceeds property count limit")
		}
		inputPayload += len(selected.ID)
		for _, field := range selected.Fields {
			inputPayload += len(field)
		}
		if inputPayload > maxPayloadBytes {
			return nil, errors.New("scope input exceeds payload limit")
		}
	}
	for _, edge := range scope.Edges {
		inputPayload += len(edge.From) + len(edge.To) + len(edge.Property)
		if inputPayload > maxPayloadBytes {
			return nil, errors.New("scope input exceeds payload limit")
		}
	}
	coreDefs := make(map[string]core.Definition, len(model.Definitions))
	for _, d := range model.Definitions {
		id := d.Identity().Key()
		if _, exists := coreDefs[id]; exists {
			return nil, fmt.Errorf("duplicate Core identity %q", id)
		}
		coreDefs[id] = d
	}
	factByID := make(map[string]Fact, len(facts.Facts))
	for _, f := range facts.Facts {
		if f.ID == "" || f.Kind == "" {
			return nil, errors.New("fact ID and kind are required")
		}
		if f.State != FactKnown && f.State != FactPartial && f.State != FactUnknown {
			return nil, fmt.Errorf("fact %q has invalid completeness state", f.ID)
		}
		if f.State == FactUnknown && len(f.Properties) != 0 {
			return nil, fmt.Errorf("unknown fact %q cannot carry known literal properties", f.ID)
		}
		if len(f.Properties) > core.MaxPropertiesPerObject {
			return nil, fmt.Errorf("fact %q exceeds property count limit", f.ID)
		}
		inputPayload += len(f.ID) + len(f.Kind)
		for key, value := range f.Properties {
			inputPayload += len(key) + len(value)
			if inputPayload > maxPayloadBytes {
				return nil, errors.New("supplied project facts exceed payload limit")
			}
		}
		inputPayload += len(f.Source.Path) + len(f.Source.Digest)
		if inputPayload > maxPayloadBytes {
			return nil, errors.New("supplied project facts exceed payload limit")
		}
		if _, exists := coreDefs[f.ID]; exists {
			return nil, fmt.Errorf("fact ID collides with Core identity %q", f.ID)
		}
		if _, exists := factByID[f.ID]; exists {
			return nil, fmt.Errorf("duplicate fact ID %q", f.ID)
		}
		factByID[f.ID] = f
	}
	selected := make(map[string]NodeSelection, len(scope.Nodes))
	for _, s := range scope.Nodes {
		if s.ID == "" {
			return nil, errors.New("selected node ID is required")
		}
		if _, exists := selected[s.ID]; exists {
			return nil, fmt.Errorf("duplicate selected node %q", s.ID)
		}
		if _, ok := coreDefs[s.ID]; !ok {
			if _, ok = factByID[s.ID]; !ok {
				return nil, fmt.Errorf("selected node %q is absent", s.ID)
			}
		}
		selected[s.ID] = s
	}
	if len(selected) > maxNodes {
		return nil, errors.New("selected node limit exceeded")
	}
	idx := &Index{nodes: make([]Node, 0, len(selected)), byID: make(map[string]int, len(selected)), out: map[string][]int{}, in: map[string][]int{}}
	ids := make([]string, 0, len(selected))
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	payload := bindingPayload
	for _, id := range ids {
		s := selected[id]
		var n Node
		if d, ok := coreDefs[id]; ok {
			n = Node{ID: id, Kind: d.Kind}
			if s.IncludePurpose {
				n.Purpose = d.Purpose
			}
			if s.IncludeSource {
				src := d.Source
				n.Source = &src
			}
			props, err := selectedCoreProperties(model, d, s.Fields)
			if err != nil {
				return nil, fmt.Errorf("node %q: %w", id, err)
			}
			n.Properties = props
		} else {
			f := factByID[id]
			n = Node{ID: id, Kind: f.Kind, FactState: f.State}
			if s.IncludeSource {
				src := f.Source
				n.Source = &src
			}
			props, err := selectedFactProperties(f, s.Fields)
			if err != nil {
				return nil, fmt.Errorf("node %q: %w", id, err)
			}
			n.Properties = props
			if s.IncludePurpose {
				return nil, fmt.Errorf("purpose is unavailable for fact node %q", id)
			}
		}
		b, _ := json.Marshal(n)
		payload += len(b)
		if payload > maxPayloadBytes {
			return nil, errors.New("graph payload limit exceeded")
		}
		idx.byID[id] = len(idx.nodes)
		idx.nodes = append(idx.nodes, n)
	}
	coreEdges := map[EdgeKey]core.Edge{}
	for _, e := range model.Edges {
		k := EdgeKey{e.From, e.To, e.Property}
		if _, exists := coreEdges[k]; exists {
			return nil, fmt.Errorf("conflicting Core edge %v", k)
		}
		coreEdges[k] = e
	}
	factEdges := map[EdgeKey]Relation{}
	for _, e := range facts.Relations {
		k := EdgeKey{e.From, e.To, e.Property}
		inputPayload += len(e.From) + len(e.To) + len(e.Property) + len(e.Source.Path) + len(e.Source.Digest) + len(e.Basis)
		if inputPayload > maxPayloadBytes {
			return nil, errors.New("supplied project relations exceed payload limit")
		}
		if _, exists := factEdges[k]; exists {
			return nil, fmt.Errorf("conflicting fact relation %v", k)
		}
		factEdges[k] = e
	}
	seenEdges := map[EdgeKey]bool{}
	for _, k := range scope.Edges {
		if k.From == "" || k.To == "" || k.Property == "" {
			return nil, errors.New("edge key fields are required")
		}
		if seenEdges[k] {
			return nil, fmt.Errorf("duplicate selected edge %v", k)
		}
		seenEdges[k] = true
		if _, ok := selected[k.From]; !ok {
			return nil, fmt.Errorf("edge source %q is outside selected nodes", k.From)
		}
		if _, ok := selected[k.To]; !ok {
			return nil, fmt.Errorf("edge target %q is outside selected nodes", k.To)
		}
		if _, coreOK := coreEdges[k]; coreOK {
			if _, factOK := factEdges[k]; factOK {
				return nil, fmt.Errorf("Core edge conflicts with supplied fact relation %v", k)
			}
		}
		var edge Edge
		if e, ok := coreEdges[k]; ok {
			edge = Edge{From: e.From, To: e.To, Property: e.Property, Source: e.Source}
		} else if e, ok := factEdges[k]; ok {
			edge = Edge{From: e.From, To: e.To, Property: e.Property, Source: e.Source, Basis: e.Basis}
		} else {
			return nil, fmt.Errorf("selected edge %v is absent", k)
		}
		b, _ := json.Marshal(edge)
		payload += len(b)
		if payload > maxPayloadBytes {
			return nil, errors.New("graph payload limit exceeded")
		}
		idx.edges = append(idx.edges, edge)
	}
	sort.Slice(idx.edges, func(i, j int) bool {
		a, b := idx.edges[i], idx.edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Property != b.Property {
			return a.Property < b.Property
		}
		return a.To < b.To
	})
	for i, e := range idx.edges {
		idx.out[e.From] = append(idx.out[e.From], i)
		idx.in[e.To] = append(idx.in[e.To], i)
	}
	idx.binding = Binding{GraphVersion: GraphVersion, ModelDigest: model.Digest, Revision: model.Revision, SnapshotDigest: facts.SnapshotDigest, ProjectDigest: facts.ProjectDigest, FactsDigest: facts.Digest, ScopeID: scope.ID}
	canonical := struct {
		Version        string `json:"version"`
		ModelDigest    string `json:"modelDigest"`
		Revision       string `json:"revision"`
		SnapshotDigest string `json:"snapshotDigest"`
		ProjectDigest  string `json:"projectDigest"`
		FactsDigest    string `json:"factsDigest"`
		ScopeID        string `json:"scopeId"`
		Nodes          []Node `json:"nodes"`
		Edges          []Edge `json:"edges"`
	}{GraphVersion, model.Digest, model.Revision, facts.SnapshotDigest, facts.ProjectDigest, facts.Digest, scope.ID, idx.nodes, idx.edges}
	b, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	if len(b) > maxPayloadBytes {
		return nil, errors.New("serialized graph payload exceeds limit")
	}
	sum := sha256.Sum256(b)
	idx.binding.GraphDigest = "sha256:" + hex.EncodeToString(sum[:])
	return idx, nil
}

func selectedCoreProperties(model core.Model, d core.Definition, fields []string) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	if len(fields) == 0 {
		return out, nil
	}
	kind, ok := model.Kind(core.KindIdentity{APIVersion: d.APIVersion, Kind: d.Kind})
	if !ok {
		return nil, errors.New("Core kind contract is absent")
	}
	seen := map[string]bool{}
	for _, name := range fields {
		if seen[name] {
			return nil, fmt.Errorf("duplicate field %q", name)
		}
		seen[name] = true
		contract, exists := kind.Properties[name]
		if !exists {
			return nil, fmt.Errorf("field %q is not declared", name)
		}
		value, present := d.Spec[name]
		if !present {
			continue
		}
		if contract.Type == core.TypeReference || contract.Type == core.TypeKindReference {
			continue
		}
		clean, err := stripReferences(value, contract)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", name, err)
		}
		b, err := json.Marshal(clean)
		if err != nil {
			return nil, err
		}
		out[name] = append(json.RawMessage(nil), b...)
	}
	return out, nil
}

func stripReferences(value any, p core.Property) (any, error) {
	if p.Type == core.TypeReference || p.Type == core.TypeKindReference {
		return nil, nil
	}
	if a, ok := value.([]any); ok {
		out := make([]any, 0, len(a))
		for _, v := range a {
			x, err := stripReferences(v, p)
			if err != nil {
				return nil, err
			}
			out = append(out, x)
		}
		return out, nil
	}
	if p.Type == core.TypeObject {
		m, ok := value.(map[string]any)
		if !ok {
			return nil, errors.New("object value has unexpected type")
		}
		out := map[string]any{}
		for k, v := range m {
			child, ok := p.Properties[k]
			if !ok {
				return nil, fmt.Errorf("undeclared nested property %q", k)
			}
			if child.Type == core.TypeReference || child.Type == core.TypeKindReference {
				continue
			}
			x, err := stripReferences(v, child)
			if err != nil {
				return nil, err
			}
			out[k] = x
		}
		return out, nil
	}
	return value, nil
}

func selectedFactProperties(f Fact, fields []string) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	seen := map[string]bool{}
	for _, name := range fields {
		if seen[name] {
			return nil, fmt.Errorf("duplicate field %q", name)
		}
		seen[name] = true
		raw, ok := f.Properties[name]
		if !ok {
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("invalid JSON field %q", name)
		}
		var extra any
		if err := dec.Decode(&extra); err != io.EOF {
			return nil, fmt.Errorf("invalid trailing JSON in field %q", name)
		}
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		out[name] = append(json.RawMessage(nil), b...)
	}
	return out, nil
}
