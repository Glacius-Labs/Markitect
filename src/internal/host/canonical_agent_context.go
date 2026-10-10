package host

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host/canonical"
)

const maxCanonicalAgentDefinitions = 128

type CanonicalContextInclusion struct {
	Identity string     `json:"identity"`
	Reason   string     `json:"reason"`
	Via      *core.Edge `json:"via,omitempty"`
}

// CanonicalAgentContext explicitly separates selected ownership from related,
// read-only facts. It contains no hidden executor transcript or source search.
type CanonicalAgentContext struct {
	APIVersion         string                      `json:"apiVersion"`
	Revision           string                      `json:"revision"`
	ModelDigest        string                      `json:"modelDigest"`
	RequestDigest      string                      `json:"requestDigest"`
	ProjectionID       string                      `json:"projectionId"`
	Representation     string                      `json:"representation"`
	Module             canonical.Pin               `json:"module"`
	ScopeIDs           []string                    `json:"scopeIds"`
	Definitions        []core.Definition           `json:"definitions"`
	RelatedDefinitions []core.Definition           `json:"relatedDefinitions"`
	Schemas            []core.Schema               `json:"schemas"`
	Policies           []core.Definition           `json:"policies"`
	Inclusions         []CanonicalContextInclusion `json:"inclusions"`
	TargetPrefix       string                      `json:"targetPrefix"`
	Authority          string                      `json:"authority"`
}

// BuildCanonicalAgentContext follows only existing resolved outgoing edges at
// the caller's explicit bounded depth (0..2). Inclusion grants no mutation or
// ownership of either canonical Definitions or external representation scopes.
func BuildCanonicalAgentContext(model core.Model, request canonical.ProjectionRequest, referenceDepth int) (CanonicalAgentContext, error) {
	result := CanonicalAgentContext{}
	if err := canonical.ValidateProjectionRequest(model, request); err != nil {
		return result, err
	}
	if referenceDepth < 0 || referenceDepth > 2 {
		return result, errors.New("agent reference context depth must be between 0 and 2")
	}
	if request.ModelDigest != model.Digest || request.Revision != model.Revision {
		return result, errors.New("agent context request is stale relative to normalized model")
	}
	result = CanonicalAgentContext{
		APIVersion: "markitect.canonical/agent-context/v1alpha1", Revision: model.Revision, ModelDigest: model.Digest, RequestDigest: request.RequestDigest,
		ProjectionID: request.Projection.Identity().Key(), Representation: request.Projector.Target, Module: request.ModulePin,
		Definitions: append([]core.Definition(nil), request.Definitions...), Policies: append([]core.Definition(nil), request.Policies...),
		TargetPrefix: request.TargetPrefix,
		Authority:    "Canonical Definitions and policies are read-only authority. Related facts explain references only; they grant no artifact ownership. Return bounded candidate target bytes or an explicit escalation. Do not invent or adopt canonical intent.",
	}
	index := map[string]core.Definition{}
	for _, d := range model.Definitions {
		index[d.Identity().Key()] = d
	}
	seen := map[string]bool{}
	frontier := []string{}
	for _, d := range request.Definitions {
		key := d.Identity().Key()
		canonicalDef, ok := index[key]
		if !ok || !equalCanonicalValue(d, canonicalDef) {
			return CanonicalAgentContext{}, fmt.Errorf("agent context subject is not current normalized Definition: %s", key)
		}
		if seen[key] {
			return CanonicalAgentContext{}, fmt.Errorf("duplicate agent context subject: %s", key)
		}
		seen[key] = true
		frontier = append(frontier, key)
		result.ScopeIDs = append(result.ScopeIDs, key)
		result.Inclusions = append(result.Inclusions, CanonicalContextInclusion{Identity: key, Reason: "explicit-projection-scope"})
	}
	if len(seen) == 0 || len(seen) > maxCanonicalAgentDefinitions {
		return CanonicalAgentContext{}, errors.New("agent context requires 1..128 selected Definitions")
	}
	sort.Strings(frontier)
	edges := append([]core.Edge(nil), model.Edges...)
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Property != b.Property {
			return a.Property < b.Property
		}
		return a.To < b.To
	})
	for depth := 0; depth < referenceDepth; depth++ {
		selected := map[string]bool{}
		for _, id := range frontier {
			selected[id] = true
		}
		next := []string{}
		for _, edge := range edges {
			if !selected[edge.From] || seen[edge.To] {
				continue
			}
			d, ok := index[edge.To]
			if !ok {
				return CanonicalAgentContext{}, errors.New("normalized reference context contains an unresolved target")
			}
			if len(seen) >= maxCanonicalAgentDefinitions {
				return CanonicalAgentContext{}, errors.New("bounded reference context exceeds 128 Definitions; explicit smaller scope is required")
			}
			seen[edge.To] = true
			next = append(next, edge.To)
			result.RelatedDefinitions = append(result.RelatedDefinitions, d)
			via := edge
			result.Inclusions = append(result.Inclusions, CanonicalContextInclusion{Identity: edge.To, Reason: "declared-outgoing-reference-context-only", Via: &via})
		}
		sort.Strings(next)
		frontier = next
	}
	needed := map[string]map[string]bool{}
	for _, d := range append(append([]core.Definition(nil), result.Definitions...), result.RelatedDefinitions...) {
		if needed[d.APIVersion] == nil {
			needed[d.APIVersion] = map[string]bool{}
		}
		needed[d.APIVersion][d.Kind] = true
	}
	for _, schema := range model.Schemas {
		kinds := needed[schema.APIVersion]
		if len(kinds) == 0 {
			continue
		}
		scoped := schema
		scoped.Kinds = map[string]core.Kind{}
		for name := range kinds {
			kind, ok := schema.Kinds[name]
			if !ok {
				return CanonicalAgentContext{}, errors.New("agent context Kind contract missing")
			}
			scoped.Kinds[name] = kind
		}
		result.Schemas = append(result.Schemas, scoped)
	}
	sort.Strings(result.ScopeIDs)
	sort.Slice(result.Definitions, func(i, j int) bool {
		return result.Definitions[i].Identity().Key() < result.Definitions[j].Identity().Key()
	})
	sort.Slice(result.RelatedDefinitions, func(i, j int) bool {
		return result.RelatedDefinitions[i].Identity().Key() < result.RelatedDefinitions[j].Identity().Key()
	})
	sort.Slice(result.Inclusions, func(i, j int) bool { return result.Inclusions[i].Identity < result.Inclusions[j].Identity })
	sort.Slice(result.Schemas, func(i, j int) bool { return result.Schemas[i].APIVersion < result.Schemas[j].APIVersion })
	return result, nil
}
