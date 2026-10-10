package authoring

import (
	"fmt"
	"sort"
	"strings"
)

func (g *Graph) resolveResources() {
	for _, r := range g.Resources {
		if r.Kind == "Project" || r.Kind == "Package" {
			continue
		}
		for _, ref := range r.Spec.Rules {
			g.resolveRef(r, ref, "Rule", map[string]bool{"Rule": true}, "rules")
		}
		for _, ref := range r.Spec.Uses {
			allowed := usesKinds[r.Kind]
			if ref.APIVersion != "" && ref.APIVersion != APIVersion {
				allowed = map[string]bool{"*": true}
			}
			g.resolveRef(r, ref, "", allowed, "uses")
		}
		for _, ref := range r.Spec.Needs {
			g.resolveRef(r, ref, "Contract", map[string]bool{"Contract": true}, "needs")
		}
		for _, ref := range r.Spec.Implements {
			g.resolveRef(r, ref, "Contract", map[string]bool{"Contract": true}, "implements")
		}
	}
}

func (g *Graph) resolveRef(source *Resource, ref Ref, defaultKind string, allowed map[string]bool, field string) *Resource {
	target := g.resolveTarget(source, ref, defaultKind, allowed, field)
	if target == nil || target.Kind == "Project" || source.Kind == "Project" && field == "binding" {
		return target
	}
	g.addRelationship(Relationship{
		From: source.GraphKey(), To: target.GraphKey(), Relation: field,
		Path: source.Path, Line: source.Line, Reference: ref, Context: true, Invalidate: true,
	})
	return target
}

// resolveTarget validates and locates a reference without adding graph edges.
// Bindings use it during selection; only selected consumer-to-implementation
// relationships become context dependencies.
func (g *Graph) resolveTarget(source *Resource, ref Ref, defaultKind string, allowed map[string]bool, field string) *Resource {
	if allowed == nil {
		g.diag(source, "reference.field", fmt.Sprintf("%s is not allowed on %s", field, source.Kind))
		return nil
	}
	if ref.Name == "" {
		g.diag(source, "reference.name", fmt.Sprintf("%s reference has an empty name", field))
		return nil
	}
	if source.Package != "" && ref.Package != "" {
		g.diag(source, "package.transitive-reference", fmt.Sprintf("package %q cannot reference another package", source.Package))
		return nil
	}
	targetPackage := source.Package
	if ref.Package != "" {
		targetPackage = ref.Package
		if _, declared := g.projectPackagePin(ref.Package); !declared {
			g.diag(source, "package.reference-unpinned", fmt.Sprintf("reference names package %q, which is not declared by the Project", ref.Package))
			return nil
		}
		if strings.HasPrefix(field, "ruleAdapters") {
			g.diag(source, "rule-adapter.package", "ruleAdapters cannot directly reference package resources; use an explicit local wrapper")
			return nil
		}
	}
	requestedNamespace := ref.Namespace
	if requestedNamespace == "" {
		requestedNamespace = source.Metadata.Namespace
	}
	if targetPackage == source.Package && source.Kind != "Project" && source.Kind != "Package" && requestedNamespace != source.Metadata.Namespace {
		area := g.areaFor(source)
		if area == nil || !contains(area.Imports, requestedNamespace) {
			g.diag(source, "reference.scope", fmt.Sprintf("%s reference to namespace %q is not imported by area %q", field, requestedNamespace, areaName(area)))
			return nil
		}
	}
	requestedKind := ref.Kind
	if requestedKind == "" {
		requestedKind = defaultKind
	}
	requestedAPI := ref.APIVersion
	if requestedAPI == "" {
		requestedAPI = source.APIVersion
	}
	if requestedAPI != source.APIVersion && !g.Registry.IsAPIVersionRegistered(requestedAPI) {
		g.diag(source, "reference.api-version", fmt.Sprintf("%s reference names unregistered apiVersion %q", field, requestedAPI))
		return nil
	}
	var matches []*Resource
	kinds := make([]string, 0, len(allowed))
	if requestedKind != "" {
		kinds = append(kinds, requestedKind)
	} else if allowed["*"] {
		for _, targetKey := range sortedResources(g.Resources) {
			target := g.Resources[targetKey]
			if target.Metadata.Namespace == requestedNamespace && target.Package == targetPackage && target.APIVersion == requestedAPI && target.Kind != "Project" && target.Kind != "Package" && target.Kind != "Domain" {
				kinds = append(kinds, target.Kind)
			}
		}
		sort.Strings(kinds)
	} else {
		for kind := range allowed {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
	}
	for _, kind := range kinds {
		key := (Ref{APIVersion: requestedAPI, Kind: kind, Namespace: requestedNamespace, Name: ref.Name}).GraphKey(targetPackage, "", "")
		if target := g.Resources[key]; target != nil {
			matches = append(matches, target)
		}
	}
	if len(matches) == 0 {
		key := (Ref{APIVersion: requestedAPI, Kind: requestedKind, Namespace: requestedNamespace, Name: ref.Name}).GraphKey(targetPackage, "", "")
		g.diag(source, "reference.missing", fmt.Sprintf("%s reference %s does not resolve", field, key))
		return nil
	}
	if len(matches) > 1 {
		g.diag(source, "reference.ambiguous", fmt.Sprintf("%s reference %q is ambiguous; specify kind and namespace", field, ref.Name))
		return nil
	}
	target := matches[0]
	if !allowed[target.Kind] && !allowed["*"] {
		g.diag(source, "reference.kind", fmt.Sprintf("%s cannot reference %s", field, target.Kind))
		return nil
	}
	if source.Package == "" && target.Package != "" && !g.IsExported(target) {
		g.diag(source, "package.reference-private", fmt.Sprintf("package resource %s is not exported", target.GraphKey()))
		return nil
	}
	if ref.Kind != "" && ref.Kind != target.Kind {
		g.diag(source, "reference.kind", fmt.Sprintf("%s declares kind %s but resolves to %s", field, ref.Kind, target.Kind))
		return nil
	}
	return target
}

func (g *Graph) lookupRef(source *Resource, ref Ref, kind string) *Resource {
	if source == nil || ref.Name == "" {
		return nil
	}
	namespace := ref.Namespace
	if namespace == "" {
		namespace = source.Metadata.Namespace
	}
	requestedKind := ref.Kind
	if requestedKind == "" {
		requestedKind = kind
	}
	packageName := source.Package
	if ref.Package != "" {
		if source.Package != "" {
			return nil
		}
		packageName = ref.Package
	}
	apiVersion := ref.APIVersion
	if apiVersion == "" && source != nil {
		apiVersion = source.APIVersion
	}
	return g.Resources[(Ref{APIVersion: apiVersion, Kind: requestedKind, Namespace: namespace, Name: ref.Name}).GraphKey(packageName, "", "")]
}

func (g *Graph) lookupExact(ref Ref) *Resource {
	if ref.Kind == "" || ref.Namespace == "" {
		return nil
	}
	return g.Resources[ref.GraphKey("", "", "")]
}

func (g *Graph) resolveExactRef(owner *Resource, ref Ref, kind, field string) *Resource {
	if ref.Kind == "" || ref.Namespace == "" || ref.Name == "" || (kind != "" && ref.Kind != kind) {
		g.diag(owner, "binding.qualified", fmt.Sprintf("%s reference must specify kind, namespace, and name", field))
		return nil
	}
	return g.resolveTarget(owner, ref, kind, map[string]bool{ref.Kind: true}, field)
}

func (g *Graph) resolveUse(source *Resource, ref Ref) *Resource {
	allowed := usesKinds[source.Kind]
	if allowed == nil {
		return nil
	}
	if ref.Kind != "" {
		if !allowed[ref.Kind] {
			return nil
		}
		return g.lookupRef(source, ref, ref.Kind)
	}
	var found *Resource
	for kind := range allowed {
		if target := g.lookupRef(source, ref, kind); target != nil {
			if found != nil {
				return nil
			}
			found = target
		}
	}
	return found
}
