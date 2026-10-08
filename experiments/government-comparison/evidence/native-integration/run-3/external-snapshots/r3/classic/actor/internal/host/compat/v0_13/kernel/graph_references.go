package core

import (
	"fmt"
	"sort"
)

func (g *Graph) resolveTarget(source *Resource, ref Ref, defaultKind string, allowed map[string]bool, field string) *Resource {
	if source == nil || ref.Name == "" {
		if source != nil {
			g.diag(source, "reference.name", field+" reference has an empty name")
		}
		return nil
	}
	api := ref.APIVersion
	if api == "" {
		api = source.APIVersion
	}
	kind := ref.Kind
	if kind == "" {
		kind = defaultKind
	}
	if !g.Registry.IsAPIVersionRegistered(api) {
		g.diag(source, "reference.api-version", fmt.Sprintf("%s reference names unregistered apiVersion %q", field, api))
		return nil
	}
	origin := source.Package
	if ref.Package != "" {
		origin = ref.Package
	}
	namespace := ref.Namespace
	if namespace == "" {
		namespace = source.Metadata.Namespace
	}
	var kinds []string
	if kind != "" {
		kinds = []string{kind}
	} else if allowed["*"] {
		for _, r := range g.Resources {
			if r.APIVersion == api && r.Metadata.Namespace == namespace && r.Package == origin {
				kinds = append(kinds, r.Kind)
			}
		}
		sort.Strings(kinds)
	} else {
		for k := range allowed {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
	}
	var matches []*Resource
	for _, candidate := range kinds {
		key := (Ref{APIVersion: api, Kind: candidate, Namespace: namespace, Name: ref.Name}).GraphKey(origin, "", "")
		if target := g.Resources[key]; target != nil {
			matches = append(matches, target)
		}
	}
	if len(matches) == 0 {
		g.diag(source, "reference.missing", fmt.Sprintf("%s reference %s does not resolve", field, (Ref{APIVersion: api, Kind: kind, Namespace: namespace, Name: ref.Name}).GraphKey(origin, "", "")))
		return nil
	}
	if len(matches) > 1 {
		g.diag(source, "reference.ambiguous", fmt.Sprintf("%s reference %q is ambiguous; specify kind", field, ref.Name))
		return nil
	}
	target := matches[0]
	if !allowed[target.Kind] && !allowed["*"] {
		g.diag(source, "reference.kind", fmt.Sprintf("%s cannot reference %s", field, target.Kind))
		return nil
	}
	return target
}
