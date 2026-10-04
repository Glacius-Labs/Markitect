package authoring

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

func (g *Graph) areas(packageName string) []Area {
	if packageName == "" {
		if g.Project == nil {
			return nil
		}
		return g.Project.Spec.Areas
	}
	if manifest := g.Packages[packageName]; manifest != nil {
		return manifest.Spec.Areas
	}
	return nil
}

func (g *Graph) scopeOwner(packageName string) *Resource {
	if packageName == "" {
		return g.Project
	}
	return g.Packages[packageName]
}

func cleanPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(path.Clean("/"+p), "/")
	if p == "." {
		return ""
	}
	return strings.TrimSuffix(p, "/")
}

func within(candidate, root string) bool {
	if root == "" {
		return false
	}
	return candidate == root || strings.HasPrefix(candidate, root+"/")
}

func (g *Graph) assignAreas() {
	owners := []*Resource{g.Project}
	for _, name := range sortedResources(g.Packages) {
		owners = append(owners, g.Packages[name])
	}
	for _, owner := range owners {
		if owner == nil {
			continue
		}
		for _, a := range g.areas(owner.Package) {
			for _, ns := range a.Imports {
				if !g.hasArea(owner.Package, ns) {
					g.diag(owner, "area.import", fmt.Sprintf("area %q imports unknown namespace %q", a.Name, ns))
				}
			}
		}
	}
	for _, r := range g.Resources {
		if r.APIVersion == APIVersion && (r.Kind == "Project" || r.Kind == "Package") {
			continue
		}
		areas := g.areas(r.Package)
		file := cleanPath(r.Path)
		bestLen := -1
		var best *Area
		ambiguous := false
		for i := range areas {
			root := cleanPath(areas[i].Path)
			if within(file, root) {
				if len(root) > bestLen {
					bestLen, best, ambiguous = len(root), &areas[i], false
				} else if len(root) == bestLen {
					ambiguous = true
					if best == nil || areas[i].Name < best.Name {
						best = &areas[i]
					}
				}
			}
		}
		if best == nil {
			g.diag(r, "area.unowned", fmt.Sprintf("resource path %q is outside every project area", r.Path))
			continue
		}
		if ambiguous {
			g.diag(r, "area.ambiguous", fmt.Sprintf("resource path %q matches equally specific areas", r.Path))
		}
		g.ResourceAreas[r.GraphKey()] = *best
		if r.Metadata.Namespace != best.Name {
			g.diag(r, "area.namespace", fmt.Sprintf("resource namespace %q does not match owning area %q", r.Metadata.Namespace, best.Name))
		}
	}
}

func (g *Graph) hasArea(packageName, name string) bool {
	for _, a := range g.areas(packageName) {
		if a.Name == name {
			return true
		}
	}
	return false
}

func (g *Graph) areaFor(r *Resource) *Area {
	file := cleanPath(r.Path)
	bestLen := -1
	var best *Area
	areas := g.areas(r.Package)
	for i := range areas {
		a := &areas[i]
		root := cleanPath(a.Path)
		if within(file, root) && (len(root) > bestLen || (len(root) == bestLen && (best == nil || a.Name < best.Name))) {
			best, bestLen = a, len(root)
		}
	}
	return best
}

func (g *Graph) resolveAreaRules() {
	for _, r := range g.Resources {
		if r.APIVersion == APIVersion && (r.Kind == "Project" || r.Kind == "Package") {
			continue
		}
		file := cleanPath(r.Path)
		areas := g.areas(r.Package)
		var ancestors []Area
		for _, a := range areas {
			if within(file, cleanPath(a.Path)) {
				ancestors = append(ancestors, a)
			}
		}
		sort.Slice(ancestors, func(i, j int) bool {
			li, lj := len(cleanPath(ancestors[i].Path)), len(cleanPath(ancestors[j].Path))
			if li != lj {
				return li < lj
			}
			return ancestors[i].Name < ancestors[j].Name
		})
		for _, a := range ancestors {
			for _, ref := range a.Rules {
				g.resolveAreaRule(r, a, ref)
			}
		}
	}
}

func (g *Graph) resolveAreaRule(resource *Resource, area Area, ref Ref) {
	if ref.APIVersion == "" {
		ref.APIVersion = APIVersion
	}
	if ref.Namespace == "" {
		ref.Namespace = area.Name
	}
	if ref.Kind == "" {
		ref.Kind = "Rule"
	}
	if ref.Kind != "Rule" {
		g.diag(resource, "reference.kind", fmt.Sprintf("area.rules reference declares kind %s; expected Rule", ref.Kind))
		return
	}
	target := g.resolveTarget(resource, ref, "Rule", map[string]bool{"Rule": true}, "area.rules")
	if target == nil {
		return
	}
	owner := g.scopeOwner(resource.Package)
	g.addRelationship(Relationship{
		From: resource.GraphKey(), To: target.GraphKey(), Relation: "area.rules",
		Path: owner.Path, Line: owner.Line, Reference: ref,
		Context: true, Invalidate: true,
	})
}

func areaName(a *Area) string {
	if a == nil {
		return "<unowned>"
	}
	return a.Name
}
func contains(items []string, s string) bool {
	for _, item := range items {
		if item == s {
			return true
		}
	}
	return false
}
