package core

import (
	"fmt"
	"sort"
)

var contentKinds = map[string]bool{
	"Text": true, "Rule": true, "Contract": true,
	"Workflow": true, "Skill": true, "Agent": true,
}

var usesKinds = map[string]map[string]bool{
	"Skill":    {"Workflow": true, "Text": true},
	"Workflow": {"Workflow": true, "Skill": true, "Agent": true, "Text": true},
	"Agent":    {"Skill": true, "Workflow": true, "Text": true},
	"Contract": {"Text": true},
}

// Build resolves a Project snapshot and the directly pinned content package
// origins. Diagnostics are sorted for stable output.
func Build(resources []*Resource) *Graph {
	return BuildWithRegistry(resources, NewRegistry())
}

// BuildWithRegistry validates registered generic kinds and builds context and
// invalidation projections from the same normalized resource values.
func BuildWithRegistry(resources []*Resource, registry *Registry) *Graph {
	if registry == nil {
		registry = NewRegistry()
	}
	g := &Graph{Resources: map[string]*Resource{}, Packages: map[string]*Resource{}, Edges: map[string][]string{}, InvalidationEdges: map[string][]string{}, ResourceAreas: map[string]Area{}, Registry: registry}
	identities := map[string]*Resource{}
	ordered := append([]*Resource(nil), resources...)
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a == nil || b == nil {
			return a == nil && b != nil
		}
		if a.GraphKey() != b.GraphKey() {
			return a.GraphKey() < b.GraphKey()
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return resourceOrderKey(a) < resourceOrderKey(b)
	})
	for _, r := range ordered {
		if r == nil {
			g.addDiagnostic(Diagnostic{Code: "resource.nil", Message: "resource is nil"})
			continue
		}
		if _, ok := registry.Lookup(r.APIVersion, r.Kind); !ok {
			g.diag(r, "resource.kind", fmt.Sprintf("unsupported kind %q for apiVersion %q", r.Kind, r.APIVersion))
		} else if r.Data != nil && r.APIVersion != APIVersion {
			if err := registry.ValidateData(r.APIVersion, r.Kind, r.Data); err != nil {
				g.diag(r, "resource.schema", err.Error())
			}
		}
		if r.Kind == "Package" {
			if r.Package != r.Metadata.Name {
				g.diag(r, "package.origin", "Package manifest must have a runtime origin matching metadata.name")
			}
			if r.Metadata.Namespace != "" {
				g.diag(r, "package.namespace", "Package metadata must not have a namespace")
			}
			if previous, ok := g.Packages[r.Metadata.Name]; ok {
				g.diag(r, "package.duplicate", fmt.Sprintf("duplicate Package manifest %q (also at %s)", r.Metadata.Name, previous.Path))
			} else {
				g.Packages[r.Metadata.Name] = r
			}
		}
		key := r.GraphKey()
		identity := r.Package + "::" + r.Metadata.Namespace + "/" + r.IdentityVersion() + "/" + r.Kind + "/" + r.Metadata.Name
		if prior := identities[identity]; prior != nil && prior.APIVersion != r.APIVersion {
			g.diag(r, "resource.version-identity", fmt.Sprintf("resource identity %s is declared with conflicting apiVersions %q and %q", identity, prior.APIVersion, r.APIVersion))
		} else if prior == nil {
			identities[identity] = r
		}
		if previous, ok := g.Resources[key]; ok {
			g.diag(r, "resource.duplicate", fmt.Sprintf("duplicate resource identity %s (also at %s)", key, previous.Path))
			continue
		}
		g.Resources[key] = r
		g.Edges[key] = nil
		if r.Kind == "Project" {
			if g.Project == nil {
				g.Project = r
			} else {
				g.diag(r, "project.count", "exactly one Project resource is required")
			}
		}
	}
	if g.Project == nil {
		g.addDiagnostic(Diagnostic{Code: "project.count", Message: "exactly one Project resource is required"})
	}
	for _, r := range g.Resources {
		if r.Kind != "Project" && len(r.Spec.Checks) != 0 {
			g.diag(r, "check.kind", "checks are only allowed on Project resources")
		}
	}
	if g.Project != nil {
		g.validatePackagePins()
		g.validateProject()
		g.validatePackageManifests()
		g.assignAreas()
		g.validateProviderNames()
		g.resolveAreaRules()
		g.resolveResources()
		g.resolveBindings()
		g.validateRuleChecks()
		g.resolveDomainRelations()
		g.EvaluateConstraints()
		g.detectRuntimeCycles()
		g.detectTypedRelationCycles()
	}
	g.sortEdges()
	g.sortRelationships()
	sort.Slice(g.Diagnostics, func(i, j int) bool {
		a, b := g.Diagnostics[i], g.Diagnostics[j]
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Message < b.Message
	})
	return g
}

func resourceOrderKey(r *Resource) string {
	return fmt.Sprintf("%d\x00%s\x00%#v", r.Line, r.APIVersion, r.Spec)
}

func (g *Graph) addEdge(from, to string) {
	if from == to {
		return
	}
	for _, existing := range g.Edges[from] {
		if existing == to {
			return
		}
	}
	g.Edges[from] = append(g.Edges[from], to)
}

func (g *Graph) addInvalidationEdge(from, to string) {
	if from == to {
		return
	}
	for _, existing := range g.InvalidationEdges[from] {
		if existing == to {
			return
		}
	}
	g.InvalidationEdges[from] = append(g.InvalidationEdges[from], to)
}

func (g *Graph) addRelationship(relationship Relationship) {
	// A self-selected implementation is an execution dependency. Preserve it
	// for runtime cycle detection even though context edges omit self-links.
	if relationship.From == relationship.To && relationship.Relation != "selected-implementation" && !relationship.Acyclic {
		return
	}
	if relationship.Context {
		g.addEdge(relationship.From, relationship.To)
	}
	if relationship.Invalidate {
		g.addInvalidationEdge(relationship.From, relationship.To)
	}
	g.Relationships = append(g.Relationships, relationship)
}

func (g *Graph) sortRelationships() {
	sort.Slice(g.Relationships, func(i, j int) bool {
		a, b := g.Relationships[i], g.Relationships[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		if a.DomainAPIVersion != b.DomainAPIVersion {
			return a.DomainAPIVersion < b.DomainAPIVersion
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Area != b.Area {
			return a.Area < b.Area
		}
		if a.Reference.Namespace != b.Reference.Namespace {
			return a.Reference.Namespace < b.Reference.Namespace
		}
		if a.Reference.Kind != b.Reference.Kind {
			return a.Reference.Kind < b.Reference.Kind
		}
		if a.Reference.Name != b.Reference.Name {
			return a.Reference.Name < b.Reference.Name
		}
		return !a.Selected && b.Selected
	})
	if len(g.Relationships) < 2 {
		return
	}
	unique := g.Relationships[:1]
	for _, relationship := range g.Relationships[1:] {
		if relationship != unique[len(unique)-1] {
			unique = append(unique, relationship)
		}
	}
	g.Relationships = unique
}

func (g *Graph) sortEdges() {
	for key := range g.Edges {
		sort.Strings(g.Edges[key])
	}
}
func sortedKeys(m map[string]*Resource) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func (g *Graph) diag(r *Resource, code, message string) {
	g.addDiagnostic(Diagnostic{Code: code, Path: r.Path, Package: r.Package, Line: r.Line, Message: message})
}
func (g *Graph) addDiagnostic(d Diagnostic) { g.Diagnostics = append(g.Diagnostics, d) }
