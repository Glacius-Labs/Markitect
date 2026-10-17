// Package organization derives local Manager organization from compiled model
// values and resolves explicitly supplied execution settings. It performs no
// acquisition or execution.
package organization

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

// Root identifies the ordinary Manager of the selected local model. It is
// execution metadata, not a canonical v1 Package or Definition.
const Root = "."

// Manager is one local responsibility. Children are direct Package paths;
// Definitions contains only this Package's own canonical Definition identities.
// Root has no parent or local Definitions. All lists use exact UTF-8 ordering.
type Manager struct {
	Package     string
	Parent      string
	Children    []string
	Definitions []model.Identity
}

// HasLocalWork reports local Definition responsibility, not child integration
// work, participation or any completed realization assessment. An empty leaf
// still participates and requires assessment and review.
func (m Manager) HasLocalWork() bool { return len(m.Definitions) != 0 }

// PassThrough identifies an intermediate Package with no local Definitions
// and exactly one child. It describes structure only.
// Root is not an intermediate canonical Package.
// Deprecated: this legacy predicate grants no startup or participation
// exemption. Every local Manager must participate, including this structure.
func (m Manager) PassThrough() bool {
	return m.Package != "" && m.Package != Root && !m.HasLocalWork() && len(m.Children) == 1
}

// Topology binds the complete local Manager tree to the compiler's model ID
// and canonical content digest. Empty Packages affect this tree but remain
// outside that digest. A zero Topology has no Manager slots.
type Topology struct {
	modelID     string
	modelDigest string
	managers    []Manager
	index       map[string]int
}

func (t Topology) ModelID() string     { return t.modelID }
func (t Topology) ModelDigest() string { return t.modelDigest }

// Managers returns owned copies, ordered Root first, then by Package path.
func (t Topology) Managers() []Manager {
	result := make([]Manager, len(t.managers))
	for i, manager := range t.managers {
		result[i] = cloneManager(manager)
	}
	return result
}

// Manager looks up an exact local Package path or Root and returns an owned
// copy. Imported Packages do not gain visibility through this lookup.
func (t Topology) Manager(packagePath string) (Manager, bool) {
	i, found := t.index[packagePath]
	if !found {
		return Manager{}, false
	}
	return cloneManager(t.managers[i]), true
}

// BuildTopology consumes a successful compiler result. The caller owns
// acceptance of that result; this function does not authenticate approval.
// The selected local ModelResult supplies the complete own Package inventory,
// including empty leaves and intermediate Packages. Missing inventory is an
// error, never reconstructed from Definitions or physical source paths.
// Coordinate coverage is independent of content validity and is not a gate.
// On error the returned Topology is zero and exposes no partial Manager tree.
func BuildTopology(compiled model.Result) (Topology, error) {
	if compiled.Status != model.Passed {
		return Topology{}, fmt.Errorf("topology requires a passed compiler result, got %q", compiled.Status)
	}
	var local *model.ModelResult
	for i := range compiled.Models {
		candidate := &compiled.Models[i]
		if candidate.Local {
			if local != nil {
				return Topology{}, fmt.Errorf("topology requires exactly one local model")
			}
			local = candidate
		}
	}
	if local == nil || local.Status != model.Passed {
		return Topology{}, fmt.Errorf("topology requires exactly one passed local model")
	}
	if local.ID == "" || local.Digest == "" {
		return Topology{}, fmt.Errorf("passed local model must have an ID and canonical digest")
	}
	packages := append([]model.Package(nil), local.Packages...)
	sort.Slice(packages, func(i, j int) bool { return packages[i].Name < packages[j].Name })
	topology := Topology{
		modelID: local.ID, modelDigest: local.Digest,
		managers: make([]Manager, len(packages)+1),
		index:    map[string]int{Root: 0},
	}
	topology.managers[0] = Manager{Package: Root}
	for i, pkg := range packages {
		if !packagePath(pkg.Name) {
			return Topology{}, fmt.Errorf("invalid local Package path %q", pkg.Name)
		}
		if _, found := topology.index[pkg.Name]; found {
			return Topology{}, fmt.Errorf("duplicate local Package %q", pkg.Name)
		}
		topology.index[pkg.Name] = i + 1
		topology.managers[i+1] = Manager{Package: pkg.Name, Parent: parentPackage(pkg.Name)}
	}
	for i := 1; i < len(topology.managers); i++ {
		manager := &topology.managers[i]
		parent, found := topology.index[manager.Parent]
		if !found {
			return Topology{}, fmt.Errorf("local Package %q is missing parent %q in compiler inventory", manager.Package, manager.Parent)
		}
		topology.managers[parent].Children = append(topology.managers[parent].Children, manager.Package)
	}
	seen := make(map[model.Identity]bool, len(local.Definitions))
	for _, definition := range local.Definitions {
		identity := definition.Identity
		i, found := topology.index[identity.Package]
		if !found || identity.Package == Root || identity.ID == "" || definition.ModelID != local.ID {
			return Topology{}, fmt.Errorf("Definition %q/%q does not belong to the local Package inventory", identity.Package, identity.ID)
		}
		if seen[identity] {
			return Topology{}, fmt.Errorf("duplicate local Definition %q/%q", identity.Package, identity.ID)
		}
		seen[identity] = true
		topology.managers[i].Definitions = append(topology.managers[i].Definitions, identity)
	}
	for i := range topology.managers {
		definitions := topology.managers[i].Definitions
		sort.Slice(definitions, func(i, j int) bool { return definitions[i].ID < definitions[j].ID })
	}
	return topology, nil
}

func cloneManager(manager Manager) Manager {
	manager.Children = append([]string(nil), manager.Children...)
	manager.Definitions = append([]model.Identity(nil), manager.Definitions...)
	return manager
}

// The compiler owns language name validity. These checks only reject shapes
// that cannot form exact Package ancestry; they never normalize a spelling.
func packagePath(name string) bool {
	if name == "" || strings.ContainsAny(name, "\\:\x00") {
		return false
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func parentPackage(name string) string {
	if slash := strings.LastIndexByte(name, '/'); slash >= 0 {
		return name[:slash]
	}
	return Root
}
