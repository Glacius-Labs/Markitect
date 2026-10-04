package core

import (
	"fmt"
	"sort"
	"strings"
)

func (g *Graph) resolveDomainRelations() {
	if g == nil || g.Registry == nil {
		return
	}
	for _, key := range sortedKeys(g.Resources) {
		source := g.Resources[key]
		if source == nil {
			continue
		}
		domain, ok := g.Registry.Domain(source.APIVersion)
		if !ok {
			continue
		}
		relations := g.Registry.Relations(source.APIVersion)
		covered := map[string]bool{}
		for _, name := range sortedMapKeys(relations) {
			definition := relations[name]
			if !contains(definition.SourceKinds, source.Kind) {
				continue
			}
			covered[definition.Field] = true
			refs := relationValues(source.Data[definition.Field])
			if definition.MinTargets != nil && len(refs) < *definition.MinTargets {
				g.diag(source, "relation.min-targets", fmt.Sprintf("relation %s requires at least %d target(s), found %d", name, *definition.MinTargets, len(refs)))
			}
			if definition.MaxTargets != nil && len(refs) > *definition.MaxTargets {
				g.diag(source, "relation.max-targets", fmt.Sprintf("relation %s allows at most %d target(s), found %d", name, *definition.MaxTargets, len(refs)))
			}
			seen := map[string]bool{}
			for _, value := range refs {
				ref, err := valueRef(value, source.APIVersion, definition.TargetKinds)
				if err != nil {
					g.diag(source, "relation.reference", fmt.Sprintf("relation %s: %v", name, err))
					continue
				}
				if ref.APIVersion != source.APIVersion {
					g.diag(source, "relation.api-version", fmt.Sprintf("relation %s targets must use domain apiVersion %q", name, source.APIVersion))
					continue
				}
				allowed := map[string]bool{}
				for _, kind := range definition.TargetKinds {
					allowed[kind] = true
				}
				target := g.resolveTarget(source, ref, "", allowed, "relation "+name)
				if target == nil {
					continue
				}
				if seen[target.GraphKey()] {
					g.diag(source, "relation.duplicate-target", fmt.Sprintf("relation %s declares target %s more than once", name, target.GraphKey()))
					continue
				}
				seen[target.GraphKey()] = true
				g.addRelationship(Relationship{From: source.GraphKey(), To: target.GraphKey(), Relation: name, DomainAPIVersion: source.APIVersion, Path: source.Path, Line: source.Line, Reference: ref, Context: definition.Context, Invalidate: definition.Invalidate, Acyclic: definition.Acyclic})
			}
		}
		// Typed references always resolve, even where their field has no graph
		// projection. A RelationDefinition adds its independently declared effects.
		kind, _ := g.Registry.Lookup(source.APIVersion, source.Kind)
		for _, field := range sortedMapKeys(kind.Properties) {
			if covered[field] {
				continue
			}
			walkTypedRefs(source.Data[field], kind.Properties[field], func(value any, refKind string) {
				ref, err := valueRef(value, source.APIVersion, []string{refKind})
				if err != nil {
					g.diag(source, "reference.typed", fmt.Sprintf("field %s: %v", field, err))
					return
				}
				if ref.APIVersion != source.APIVersion {
					g.diag(source, "reference.api-version", fmt.Sprintf("typed references in domain %q cannot target apiVersion %q", source.APIVersion, ref.APIVersion))
					return
				}
				allowed := map[string]bool{refKind: true}
				target := g.resolveTarget(source, ref, refKind, allowed, field)
				if target != nil {
					g.addRelationship(Relationship{From: source.GraphKey(), To: target.GraphKey(), Relation: field, Path: source.Path, Line: source.Line, Reference: ref})
				}
			})
		}
		_ = domain
	}
}

func valueRef(value any, apiVersion string, targetKinds []string) (Ref, error) {
	m, ok := asMap(value)
	if !ok {
		return Ref{}, fmt.Errorf("target must be a reference object")
	}
	ref := Ref{APIVersion: apiVersion}
	for key, value := range m {
		v, ok := value.(string)
		if !ok {
			return Ref{}, fmt.Errorf("reference %s must be string", key)
		}
		switch key {
		case "apiVersion":
			ref.APIVersion = v
		case "kind":
			ref.Kind = v
		case "name":
			ref.Name = v
		case "namespace":
			ref.Namespace = v
		case "package":
			ref.Package = v
		default:
			return Ref{}, fmt.Errorf("unknown reference field %q", key)
		}
	}
	if ref.Name == "" {
		return Ref{}, fmt.Errorf("reference requires name")
	}
	if ref.Kind == "" {
		if len(targetKinds) == 1 && targetKinds[0] != "*" {
			ref.Kind = targetKinds[0]
		} else {
			return Ref{}, fmt.Errorf("reference requires kind")
		}
	}
	return ref, nil
}

func walkTypedRefs(value any, property PropertyDefinition, visit func(any, string)) {
	switch property.Type {
	case "ref":
		if value != nil {
			visit(value, property.RefKind)
		}
	case "array":
		if values, ok := value.([]any); ok && property.Items != nil {
			for _, item := range values {
				walkTypedRefs(item, *property.Items, visit)
			}
		}
	case "object":
		if m, ok := asMap(value); ok {
			for _, field := range sortedMapKeys(property.Properties) {
				walkTypedRefs(m[field], property.Properties[field], visit)
			}
		}
	}
}

func (g *Graph) detectTypedRelationCycles() {
	type edge struct {
		from, to string
		path     string
		line     int
	}
	byRelation := map[string][]edge{}
	for _, relation := range g.Relationships {
		if relation.Acyclic {
			byRelation[relation.DomainAPIVersion+"\x00"+relation.Relation] = append(byRelation[relation.DomainAPIVersion+"\x00"+relation.Relation], edge{relation.From, relation.To, relation.Path, relation.Line})
		}
	}
	for _, name := range sortedMapKeys(byRelation) {
		label := name
		if version, relation, ok := strings.Cut(name, "\x00"); ok {
			label = version + "/" + relation
		}
		edges := byRelation[name]
		adj := map[string][]string{}
		for _, e := range edges {
			adj[e.from] = append(adj[e.from], e.to)
		}
		for from := range adj {
			sort.Strings(adj[from])
		}
		state := map[string]int{}
		reported := map[string]bool{}
		var visit func(string)
		visit = func(node string) {
			state[node] = 1
			for _, to := range adj[node] {
				if state[to] == 1 {
					if !reported[node] {
						reported[node] = true
						g.addDiagnostic(Diagnostic{Code: "relation.cycle", Message: fmt.Sprintf("relation %s contains a cycle through %s", label, to)})
					}
					continue
				}
				if state[to] == 0 {
					visit(to)
				}
			}
			state[node] = 2
		}
		for _, node := range sortedMapKeys(adj) {
			if state[node] == 0 {
				visit(node)
			}
		}
	}
}
