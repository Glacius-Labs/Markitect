package authoring

import (
	"fmt"
)

func (g *Graph) resolveBindings() {
	bindings := map[string]*Resource{}
	// Every resource that declares implements makes a compatibility claim, even
	// when this project selects a different implementation for the contract.
	for _, impl := range g.Resources {
		if impl.Kind == "Project" || impl.Kind == "Package" || (impl.Kind != "Agent" && impl.Kind != "Skill" && impl.Kind != "Workflow") {
			continue
		}
		for _, ref := range impl.Spec.Implements {
			if contract := g.lookupRef(impl, ref, "Contract"); contract != nil {
				g.validateImplementation(impl, contract)
			}
		}
	}
	// Package-local bindings are fixed by the package and cannot be replaced by
	// consumer Project bindings.
	for _, name := range sortedResources(g.Packages) {
		manifest := g.Packages[name]
		for _, binding := range manifest.Spec.Bindings {
			g.addBinding(bindings, manifest, binding, name, true)
		}
	}
	for _, binding := range g.Project.Spec.Bindings {
		contract := g.resolveExactRef(g.Project, binding.Contract, "Contract", "binding.contract")
		if contract != nil && contract.Package != "" && g.packageHasBinding(contract.GraphKey()) {
			g.diag(g.Project, "binding.package-override", fmt.Sprintf("consumer Project binding cannot override package-local binding for %s", contract.GraphKey()))
			continue
		}
		g.addBinding(bindings, g.Project, binding, "", false)
	}
	for _, r := range g.Resources {
		if r.Kind == "Project" || r.Kind == "Package" || (r.Kind != "Agent" && r.Kind != "Skill" && r.Kind != "Workflow") {
			continue
		}
		for _, need := range r.Spec.Needs {
			contract := g.lookupRef(r, need, "Contract")
			if contract == nil {
				continue
			}
			implementation, ok := bindings[contract.GraphKey()]
			if !ok {
				g.diag(r, "binding.missing", fmt.Sprintf("needed contract %s has no binding", contract.GraphKey()))
				continue
			}
			if r.Package != "" && (!g.packageHasBinding(contract.GraphKey()) || implementation.Package != r.Package) {
				g.diag(r, "binding.package-scope", "package requirements must be bound by their own Package manifest to a package-local implementation")
				continue
			}
			g.addRelationship(Relationship{
				From: r.GraphKey(), To: implementation.GraphKey(), Relation: "selected-implementation",
				Path: g.scopeOwner(r.Package).Path, Line: g.scopeOwner(r.Package).Line,
				Reference: Ref{Kind: implementation.Kind, Namespace: implementation.Metadata.Namespace, Name: implementation.Metadata.Name, Package: implementation.Package}, Selected: true,
				Context: true, Invalidate: true,
			})
			g.addEdge(r.GraphKey(), contract.GraphKey())
		}
	}
}

func (g *Graph) addBinding(bindings map[string]*Resource, owner *Resource, binding Binding, packageName string, packageLocal bool) {
	if binding.Contract.Kind != "Contract" || binding.Contract.Namespace == "" || binding.Contract.Name == "" || binding.Implementation.Kind == "" || binding.Implementation.Namespace == "" || binding.Implementation.Name == "" {
		g.diag(owner, "binding.qualified", "binding contract and implementation must be fully qualified")
		return
	}
	if packageLocal && (binding.Contract.Package != "" || binding.Implementation.Package != "") {
		g.diag(owner, "binding.package-scope", "package-local bindings must reference resources in the same package")
		return
	}
	contract := g.resolveExactRef(owner, binding.Contract, "Contract", "binding.contract")
	impl := g.resolveExactRef(owner, binding.Implementation, "", "binding.implementation")
	if contract == nil || impl == nil {
		g.diag(owner, "binding.target", fmt.Sprintf("binding %s -> %s has a missing or private target", binding.Contract.GraphKey(packageName, "", ""), binding.Implementation.GraphKey(packageName, "", "")))
		return
	}
	if contract.Kind != "Contract" {
		g.diag(owner, "binding.contract-kind", "binding contract target must have kind Contract")
		return
	}
	if impl.Kind != "Agent" && impl.Kind != "Skill" && impl.Kind != "Workflow" {
		g.diag(owner, "binding.implementation-kind", fmt.Sprintf("%s cannot implement a contract", impl.Kind))
		return
	}
	if packageLocal && (contract.Package != packageName || impl.Package != packageName) {
		g.diag(owner, "binding.package-scope", "package-local binding targets must belong to the owning package")
		return
	}
	key := contract.GraphKey()
	if previous, ok := bindings[key]; ok && previous.GraphKey() != impl.GraphKey() {
		g.diag(owner, "binding.ambiguous", fmt.Sprintf("contract %s has more than one implementation", key))
		return
	}
	bindings[key] = impl
	g.validateImplementation(impl, contract)
	declared := false
	for _, ref := range impl.Spec.Implements {
		if target := g.lookupRef(impl, ref, "Contract"); target == contract {
			declared = true
		}
	}
	if !declared {
		g.diag(impl, "contract.undeclared", fmt.Sprintf("implementation does not declare implements reference to %s", contract.GraphKey()))
	}
	ownerPath := owner.Path
	g.addRelationship(Relationship{
		From: impl.GraphKey(), To: contract.GraphKey(), Relation: "binding",
		Path: ownerPath, Line: owner.Line, Reference: binding.Contract, Selected: true,
		Context: true, Invalidate: true,
	})
}

func (g *Graph) validateImplementation(implementation, contract *Resource) {
	if contract.Spec.Kind != implementation.Kind {
		g.diag(implementation, "contract.kind", fmt.Sprintf("contract expects %s but implementation is %s", contract.Spec.Kind, implementation.Kind))
	}
	if !sameStrings(contract.Spec.Input, implementation.Spec.Input) || !sameStrings(contract.Spec.Output, implementation.Spec.Output) {
		g.diag(implementation, "contract.signature", fmt.Sprintf("implementation signature does not exactly match contract %s", contract.GraphKey()))
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
