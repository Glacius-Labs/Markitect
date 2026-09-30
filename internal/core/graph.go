package core

import (
	"fmt"
	"path"
	"sort"
	"strings"
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
	g := &Graph{Resources: map[string]*Resource{}, Packages: map[string]*Resource{}, Edges: map[string][]string{}, ResourceAreas: map[string]Area{}}
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
		if !contentKinds[r.Kind] && r.Kind != "Project" && r.Kind != "Package" {
			g.diag(r, "resource.kind", fmt.Sprintf("unsupported kind %q", r.Kind))
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
		g.detectRuntimeCycles()
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

func (g *Graph) validateProject() {
	p := g.Project
	for _, target := range p.Spec.Targets {
		if target != "codex" && target != "claude" {
			g.diag(p, "project.target", fmt.Sprintf("unsupported target %q", target))
		}
	}
	checkNames := map[string]bool{}
	for _, check := range p.Spec.Checks {
		if err := ValidateCheck(check); err != nil {
			g.diag(p, "check.invalid", fmt.Sprintf("project check %q is invalid: %v", check.Name, err))
		}
		if checkNames[check.Name] {
			g.diag(p, "check.duplicate", fmt.Sprintf("project check name %q is duplicated", check.Name))
		}
		checkNames[check.Name] = true
	}
	for entrypoint, refs := range p.Spec.RuleAdapters {
		if strings.TrimSpace(entrypoint) == "" {
			g.diag(p, "rule-adapter.name", "rule adapter entrypoint name is empty")
		}
		for _, ref := range refs {
			if ref.Package != "" {
				g.diag(p, "rule-adapter.package", "ruleAdapters cannot directly reference package resources; use an explicit local wrapper")
				continue
			}
			source := g.resolveRef(p, ref, "Rule", map[string]bool{"Rule": true, "Workflow": true, "Text": true}, "ruleAdapters["+entrypoint+"]")
			if source != nil && source.Kind != "Rule" && source.Kind != "Workflow" && source.Kind != "Text" {
				g.diag(p, "rule-adapter.kind", fmt.Sprintf("rule adapter source %s must be Rule, Workflow, or Text", source.GraphKey()))
			}
		}
	}
	seen := map[string]bool{}
	for _, a := range p.Spec.Areas {
		if a.Name == "" || a.Path == "" {
			g.diag(p, "area.invalid", "area name and path are required")
			continue
		}
		if seen[a.Name] {
			g.diag(p, "area.duplicate", fmt.Sprintf("duplicate area %q", a.Name))
		}
		seen[a.Name] = true
	}
}

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
	for _, name := range sortedKeys(g.Packages) {
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
		if r.Kind == "Project" || r.Kind == "Package" {
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
		if r.Kind == "Project" || r.Kind == "Package" {
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
		Path: owner.Path, Line: owner.Line, Reference: ref, Area: area.Name,
	})
}

func (g *Graph) resolveResources() {
	for _, r := range g.Resources {
		if r.Kind == "Project" || r.Kind == "Package" {
			continue
		}
		for _, ref := range r.Spec.Rules {
			g.resolveRef(r, ref, "Rule", map[string]bool{"Rule": true}, "rules")
		}
		for _, ref := range r.Spec.Uses {
			g.resolveRef(r, ref, "", usesKinds[r.Kind], "uses")
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
		Path: source.Path, Line: source.Line, Reference: ref,
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
	var matches []*Resource
	kinds := make([]string, 0, len(allowed))
	if requestedKind != "" {
		kinds = append(kinds, requestedKind)
	} else {
		for kind := range allowed {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
	}
	for _, kind := range kinds {
		key := (Ref{Kind: kind, Namespace: requestedNamespace, Name: ref.Name}).GraphKey(targetPackage, "", "")
		if target := g.Resources[key]; target != nil {
			matches = append(matches, target)
		}
	}
	if len(matches) == 0 {
		key := (Ref{Kind: requestedKind, Namespace: requestedNamespace, Name: ref.Name}).GraphKey(targetPackage, "", "")
		g.diag(source, "reference.missing", fmt.Sprintf("%s reference %s does not resolve", field, key))
		return nil
	}
	if len(matches) > 1 {
		g.diag(source, "reference.ambiguous", fmt.Sprintf("%s reference %q is ambiguous; specify kind and namespace", field, ref.Name))
		return nil
	}
	target := matches[0]
	if !allowed[target.Kind] {
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
	for _, name := range sortedKeys(g.Packages) {
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
	return g.Resources[(Ref{Kind: requestedKind, Namespace: namespace, Name: ref.Name}).GraphKey(packageName, "", "")]
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

func (g *Graph) validateProviderNames() {
	names := map[string]map[string]*Resource{}
	for _, r := range g.Resources {
		if r.Package != "" || (r.Kind != "Skill" && r.Kind != "Agent") {
			continue
		}
		if names[r.Kind] == nil {
			names[r.Kind] = map[string]*Resource{}
		}
		if prev, ok := names[r.Kind][r.Metadata.Name]; ok {
			g.diag(r, "provider.name", fmt.Sprintf("provider name %q is duplicated for kind %s (also %s)", r.Metadata.Name, r.Kind, prev.GraphKey()))
		} else {
			names[r.Kind][r.Metadata.Name] = r
		}
	}
}

func (g *Graph) validateRuleChecks() {
	for _, r := range g.Resources {
		if r.Kind != "Rule" || r.Spec.Check == "" {
			continue
		}
		var selectedResources map[string]bool
		if r.Package != "" {
			selectedResources = map[string]bool{}
			for _, relationship := range g.Relationships {
				if relationship.To == r.GraphKey() && (relationship.Relation == "rules" || relationship.Relation == "area.rules") {
					// Area rules already have an edge for every governed resource,
					// including resources owned by a more specific descendant area.
					selectedResources[relationship.From] = true
				}
			}
			if len(selectedResources) == 0 {
				continue // Importing a package does not activate its rule checks.
			}
		}
		if r.Spec.Check != "workflow-has-entrypoint" {
			g.diag(r, "rule.check", fmt.Sprintf("unsupported rule check %q", r.Spec.Check))
			continue
		}
		reachable := map[string]bool{}
		for _, source := range g.Resources {
			if source.Kind != "Skill" && source.Kind != "Agent" {
				continue
			}
			for _, ref := range source.Spec.Uses {
				target := g.resolveUse(source, ref)
				if target != nil && target.Kind == "Workflow" {
					reachable[target.GraphKey()] = true
				}
			}
		}
		changed := true
		for changed {
			changed = false
			for _, source := range g.Resources {
				if source.Kind != "Workflow" || !reachable[source.GraphKey()] {
					continue
				}
				for _, ref := range source.Spec.Uses {
					if target := g.resolveUse(source, ref); target != nil && target.Kind == "Workflow" && !reachable[target.GraphKey()] {
						reachable[target.GraphKey()] = true
						changed = true
					}
				}
			}
		}
		for _, target := range g.Resources {
			inScope := false
			if selectedResources == nil {
				// A local rule keeps its legacy project-wide behavior for local
				// resources, but must not impose that policy on imported content.
				inScope = target.Package == r.Package
			} else {
				inScope = selectedResources[target.GraphKey()]
			}
			if target.Kind == "Workflow" && !reachable[target.GraphKey()] && inScope {
				g.diag(target, "workflow.entrypoint", "workflow is not used by a Skill, Agent, or another reachable Workflow")
			}
		}
	}
}

func (g *Graph) detectRuntimeCycles() {
	// Only concrete execution edges participate. Rule and implements links are
	// context edges and must not manufacture runtime cycles.
	adj := map[string][]string{}
	for _, source := range g.Resources {
		if source.Kind == "Project" || source.Kind == "Package" {
			continue
		}
		for _, ref := range source.Spec.Uses {
			target := g.resolveUse(source, ref)
			if target != nil && (target.Kind == "Workflow" || target.Kind == "Skill" || target.Kind == "Agent") {
				adj[source.GraphKey()] = append(adj[source.GraphKey()], target.GraphKey())
			}
		}
	}
	for _, relationship := range g.Relationships {
		if relationship.Relation == "selected-implementation" {
			adj[relationship.From] = append(adj[relationship.From], relationship.To)
		}
	}
	state, stack, reported := map[string]int{}, []string{}, map[string]bool{}
	var visit func(string)
	visit = func(key string) {
		state[key] = 1
		stack = append(stack, key)
		for _, next := range adj[key] {
			if state[next] == 0 {
				visit(next)
				continue
			}
			if state[next] == 1 {
				cycle := append([]string(nil), stack...)
				start := 0
				for i, item := range cycle {
					if item == next {
						start = i
						break
					}
				}
				cycle = append(cycle[start:], next)
				label := strings.Join(cycle, " -> ")
				if !reported[label] {
					reported[label] = true
					g.diag(g.Resources[key], "graph.cycle", "runtime dependency cycle: "+label)
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[key] = 2
	}
	for _, key := range sortedKeys(g.Resources) {
		if state[key] == 0 {
			visit(key)
		}
	}
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

func (g *Graph) addRelationship(relationship Relationship) {
	// A self-selected implementation is an execution dependency. Preserve it
	// for runtime cycle detection even though context edges omit self-links.
	if relationship.From == relationship.To && relationship.Relation != "selected-implementation" {
		return
	}
	g.addEdge(relationship.From, relationship.To)
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
