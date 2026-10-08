package core

import (
	"fmt"
	"sort"
)

// BuildNormalized indexes normalized resource values and evaluates only the
// explicitly supplied generic vocabulary, resolved relationships and waiver set.
func BuildNormalized(resources []*Resource, registry *Registry, relationships []Relationship, exceptions []PolicyException, policyDate string, digestEncodings map[string][]byte) *Graph {
	if registry == nil {
		registry = NewRegistry()
	}
	g := &Graph{Resources: map[string]*Resource{}, Edges: map[string][]string{}, InvalidationEdges: map[string][]string{}, Registry: registry, invalidPolicyPaths: map[string]bool{}, subjectDigestEncodings: cloneEncodings(digestEncodings)}
	identities := map[string]*Resource{}
	ordered := append([]*Resource(nil), resources...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i] == nil || ordered[j] == nil {
			return ordered[i] == nil && ordered[j] != nil
		}
		if ordered[i].GraphKey() != ordered[j].GraphKey() {
			return ordered[i].GraphKey() < ordered[j].GraphKey()
		}
		return resourceOrderKey(ordered[i]) < resourceOrderKey(ordered[j])
	})
	for _, r := range ordered {
		if r == nil {
			g.addDiagnostic(Diagnostic{Code: "resource.nil", Message: "resource is nil"})
			continue
		}
		if _, ok := registry.Lookup(r.APIVersion, r.Kind); !ok {
			g.diag(r, "resource.kind", fmt.Sprintf("unsupported kind %q for apiVersion %q", r.Kind, r.APIVersion))
		} else if r.Data == nil {
			g.diag(r, "resource.data", "normalized resource data must be an object")
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
	}
	for _, relationship := range relationships {
		g.addRelationship(relationship)
	}
	g.detectTypedRelationCycles()
	g.EvaluateConstraints(exceptions, policyDate)
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
	return fmt.Sprintf("%d\x00%s\x00%#v", r.Line, r.APIVersion, r.Data)
}
func (g *Graph) addRelationship(r Relationship) {
	if r.From == r.To && !r.Acyclic {
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
	for _, v := range g.Edges[from] {
		if v == to {
			return
		}
	}
	g.Edges[from] = append(g.Edges[from], to)
}
func (g *Graph) addInvalidationEdge(from, to string) {
	if from == to {
		return
	}
	for _, v := range g.InvalidationEdges[from] {
		if v == to {
			return
		}
	}
	g.InvalidationEdges[from] = append(g.InvalidationEdges[from], to)
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
func (g *Graph) sortEdges() {
	for key := range g.Edges {
		sort.Strings(g.Edges[key])
	}
	for key := range g.InvalidationEdges {
		sort.Strings(g.InvalidationEdges[key])
	}
}
func sortedKeys(m map[string]*Resource) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func (g *Graph) diag(r *Resource, code, message string) {
	g.addDiagnostic(Diagnostic{Code: code, Path: r.Path, Package: r.Package, Line: r.Line, Message: message})
}
func (g *Graph) addDiagnostic(d Diagnostic) { g.Diagnostics = append(g.Diagnostics, d) }

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func cloneEncodings(values map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(values))
	for k, v := range values {
		out[k] = append([]byte(nil), v...)
	}
	return out
}

// ResolveTypedRelationships resolves generic typed references from the supplied
// descriptors and normalized Data. Frontends can apply source-authoring access
// rules to these candidate facts before passing the accepted set to BuildNormalized.
func ResolveTypedRelationships(resources []*Resource, registry *Registry) ([]Relationship, []Diagnostic) {
	if registry == nil {
		registry = NewRegistry()
	}
	g := &Graph{Resources: map[string]*Resource{}, Edges: map[string][]string{}, InvalidationEdges: map[string][]string{}, Registry: registry}
	ordered := append([]*Resource(nil), resources...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i] == nil || ordered[j] == nil {
			return ordered[i] == nil && ordered[j] != nil
		}
		return ordered[i].GraphKey() < ordered[j].GraphKey()
	})
	for _, r := range ordered {
		if r == nil {
			g.addDiagnostic(Diagnostic{Code: "resource.nil", Message: "resource is nil"})
			continue
		}
		g.Resources[r.GraphKey()] = r
		g.Edges[r.GraphKey()] = nil
	}
	g.resolveDomainRelations()
	g.sortEdges()
	g.sortRelationships()
	return g.Relationships, g.Diagnostics
}
