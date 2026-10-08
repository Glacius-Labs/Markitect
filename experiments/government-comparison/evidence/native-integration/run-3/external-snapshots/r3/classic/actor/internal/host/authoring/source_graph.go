package authoring

import (
	"fmt"
	"sort"
	"strings"

	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

type Graph struct {
	Core              *core.Graph
	Resources         map[string]*Resource
	Packages          map[string]*Resource
	Edges             map[string][]string
	Relationships     []Relationship
	ResourceAreas     map[string]Area
	Project           *Resource
	Registry          *core.Registry
	Diagnostics       []Diagnostic
	InvalidationEdges map[string][]string
}

func Build(resources []*Resource) *Graph { return BuildWithRegistry(resources, NewRegistry()) }

// BuildWithRegistry validates the source-owned Project/Package graph, derives
// authorized relationships once, and passes only normalized values to Core.
func BuildWithRegistry(resources []*Resource, registry *core.Registry) *Graph {
	if registry == nil {
		registry = NewRegistry()
	}
	g := &Graph{Resources: map[string]*Resource{}, Packages: map[string]*Resource{}, Edges: map[string][]string{}, InvalidationEdges: map[string][]string{}, ResourceAreas: map[string]Area{}, Registry: registry}
	ordered := append([]*Resource(nil), resources...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i] == nil || ordered[j] == nil {
			return ordered[i] == nil && ordered[j] != nil
		}
		if ordered[i].GraphKey() != ordered[j].GraphKey() {
			return ordered[i].GraphKey() < ordered[j].GraphKey()
		}
		return sourceOrderKey(ordered[i]) < sourceOrderKey(ordered[j])
	})
	identities := map[string]*Resource{}
	for _, r := range ordered {
		if r == nil {
			g.addDiagnostic(Diagnostic{Code: "resource.nil", Message: "resource is nil"})
			continue
		}
		if r.Data == nil && r.APIVersion == core.APIVersion && r.Kind != "Project" && r.Kind != "Package" && r.Kind != "Domain" {
			data, err := normalizedSourceData(r.Spec)
			if err != nil {
				g.diag(r, "resource.data", fmt.Sprintf("cannot normalize source spec: %v", err))
			} else {
				r.Data = data
			}
		}
		if r.Kind == "Package" && r.Package == "" {
			r.Package = r.Metadata.Name
		}
		if _, ok := registry.Lookup(r.APIVersion, r.Kind); !ok {
			g.diag(r, "resource.kind", fmt.Sprintf("unsupported kind %q for apiVersion %q", r.Kind, r.APIVersion))
		}
		key := r.GraphKey()
		identity := r.Package + "::" + r.Metadata.Namespace + "/" + r.IdentityVersion() + "/" + r.Kind + "/" + r.Metadata.Name
		if prior := identities[identity]; prior != nil && prior.APIVersion != r.APIVersion {
			g.diag(r, "resource.version-identity", fmt.Sprintf("resource identity %s is declared with conflicting apiVersions %q and %q", identity, prior.APIVersion, r.APIVersion))
		} else if prior == nil {
			identities[identity] = r
		}
		if prior := g.Resources[key]; prior != nil {
			g.diag(r, "resource.duplicate", fmt.Sprintf("duplicate resource identity %s (also at %s)", key, prior.Path))
			continue
		}
		g.Resources[key] = r
		g.Edges[key] = nil
		if r.Kind == "Package" {
			if prior := g.Packages[r.Metadata.Name]; prior != nil {
				g.diag(r, "package.duplicate", fmt.Sprintf("duplicate Package manifest %q (also at %s)", r.Metadata.Name, prior.Path))
			} else {
				g.Packages[r.Metadata.Name] = r
			}
		}
		if r.Kind == "Project" {
			if g.Project == nil {
				g.Project = r
			} else {
				g.diag(r, "project.count", "exactly one Project resource is required")
			}
		}
		if r.Kind != "Project" && len(r.Spec.Checks) > 0 {
			g.diag(r, "check.kind", "checks are only allowed on Project resources")
		}
	}
	if g.Project == nil {
		g.addDiagnostic(Diagnostic{Code: "project.count", Message: "exactly one Project resource is required"})
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
	}
	// Core computes generic typed-reference candidates. Host applies source-owned
	// package/export/import access before any relationship reaches Core.
	normalized := make([]*core.Resource, 0, len(g.Resources))
	for _, key := range sortedResources(g.Resources) {
		r := g.Resources[key]
		if r.Kind != "Project" && r.Kind != "Package" && r.Kind != "Domain" {
			normalized = append(normalized, &r.Core)
		}
	}
	// Built-in AI references are resolved and access-checked by this source
	// frontend above. Core's generic Domain resolver intentionally handles only
	// resources authored in registered custom Domains.
	genericResources := make([]*core.Resource, 0, len(normalized))
	for _, resource := range normalized {
		if resource.APIVersion != core.APIVersion {
			genericResources = append(genericResources, resource)
		}
	}
	candidates, candidateDiagnostics := core.ResolveTypedRelationships(genericResources, registry)
	for _, d := range candidateDiagnostics {
		g.addDiagnostic(d)
	}
	for _, candidate := range candidates {
		if candidate.DomainAPIVersion == core.APIVersion {
			continue
		} // built-in AI links were resolved above with source access checks
		source := g.Resources[candidate.From]
		if source == nil {
			continue
		}
		allowed := map[string]bool{}
		defaultKind := candidate.Reference.Kind
		if domain, ok := registry.Domain(candidate.DomainAPIVersion); ok {
			if rel, ok := domain.Relations[candidate.Relation]; ok {
				for _, kind := range rel.TargetKinds {
					allowed[kind] = true
				}
				if defaultKind == "" && len(rel.TargetKinds) == 1 && rel.TargetKinds[0] != "*" {
					defaultKind = rel.TargetKinds[0]
				}
			}
		}
		if len(allowed) == 0 {
			allowed[defaultKind] = true
		}
		target := g.resolveTarget(source, candidate.Reference, defaultKind, allowed, candidate.Relation)
		if target != nil && target.GraphKey() == candidate.To {
			g.addRelationship(candidate)
		}
	}
	g.detectRuntimeCycles()
	g.sortEdges()
	g.sortRelationships()
	content := make([]*core.Resource, 0, len(normalized))
	for _, resource := range normalized {
		content = append(content, resource)
	}
	exceptions := make([]core.PolicyException, 0)
	policyDate := ""
	if g.Project != nil {
		policyDate = g.Project.Spec.PolicyDate
		for _, e := range g.Project.Spec.PolicyExceptions {
			exceptions = append(exceptions, e)
		}
	}
	digestResources := make([]*Resource, 0, len(content))
	for _, r := range normalized {
		digestResources = append(digestResources, g.Resources[r.GraphKey()])
	}
	encodings, err := DigestEncodings(digestResources)
	if err != nil {
		g.addDiagnostic(Diagnostic{Code: "resource.digest-encoding", Message: err.Error()})
	}
	g.Core = core.BuildNormalized(content, registry, g.Relationships, exceptions, policyDate, encodings)
	g.Relationships = g.Core.Relationships
	g.Edges = g.Core.Edges
	g.InvalidationEdges = g.Core.InvalidationEdges
	coreDiagnosticStart := len(g.Diagnostics)
	g.Diagnostics = append(g.Diagnostics, g.Core.Diagnostics...)
	if g.Project != nil {
		for i := coreDiagnosticStart; i < len(g.Diagnostics); i++ {
			diagnostic := &g.Diagnostics[i]
			if diagnostic.Code != "policy.date" && !strings.HasPrefix(diagnostic.Code, "policy.exception.") {
				continue
			}
			diagnostic.Path = g.Project.Path
			diagnostic.Line = g.Project.Line
			diagnostic.Package = g.Project.Package
		}
	}
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

func normalizedSourceData(spec Spec) (map[string]any, error) {
	encoded, err := yaml.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := yaml.Unmarshal(encoded, &data); err != nil {
		return nil, err
	}
	if data == nil {
		data = map[string]any{}
	}
	return data, nil
}

func sourceOrderKey(r *Resource) string {
	return fmt.Sprintf("%d\x00%s\x00%#v", r.Line, r.APIVersion, r.Spec)
}
func (g *Graph) addRelationship(r Relationship) {
	if r.From == r.To && r.Relation != "selected-implementation" && !r.Acyclic {
		return
	}
	if r.Context {
		g.addEdge(r.From, r.To)
	}
	if r.Invalidate {
		g.addInvalidationEdge(r.From, r.To)
	}
	g.Relationships = append(g.Relationships, r)
}
func (g *Graph) addEdge(from, to string) {
	if from == to {
		return
	}
	for _, e := range g.Edges[from] {
		if e == to {
			return
		}
	}
	g.Edges[from] = append(g.Edges[from], to)
}
func (g *Graph) addInvalidationEdge(from, to string) {
	if from == to {
		return
	}
	for _, e := range g.InvalidationEdges[from] {
		if e == to {
			return
		}
	}
	g.InvalidationEdges[from] = append(g.InvalidationEdges[from], to)
}
func (g *Graph) sortEdges() {
	for key := range g.Edges {
		sort.Strings(g.Edges[key])
	}
	for key := range g.InvalidationEdges {
		sort.Strings(g.InvalidationEdges[key])
	}
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
		return !a.Selected && b.Selected
	})
	if len(g.Relationships) < 2 {
		return
	}
	out := g.Relationships[:1]
	for _, r := range g.Relationships[1:] {
		if r != out[len(out)-1] {
			out = append(out, r)
		}
	}
	g.Relationships = out
}
func sortedResources(m map[string]*Resource) []string {
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
