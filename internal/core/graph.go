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

// Build resolves a complete resource snapshot. Diagnostics are sorted so the
// result is stable regardless of the caller's resource order.
func Build(resources []*Resource) *Graph {
	g := &Graph{Resources: map[string]*Resource{}, Edges: map[string][]string{}, ResourceAreas: map[string]Area{}}
	ordered := append([]*Resource(nil), resources...)
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a == nil || b == nil {
			return a == nil && b != nil
		}
		if a.Key() != b.Key() {
			return a.Key() < b.Key()
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
		if !contentKinds[r.Kind] && r.Kind != "Project" {
			g.diag(r, "resource.kind", fmt.Sprintf("unsupported kind %q", r.Kind))
		}
		if previous, ok := g.Resources[r.Key()]; ok {
			g.diag(r, "resource.duplicate", fmt.Sprintf("duplicate resource identity %s (also at %s)", r.Key(), previous.Path))
			continue
		}
		g.Resources[r.Key()] = r
		g.Edges[r.Key()] = nil
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
	if g.Project != nil {
		g.validateProject()
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
	if p.Spec.Profile != "generic" && p.Spec.Profile != "konfyra" && p.Spec.Profile != "cockpit" {
		g.diag(p, "project.profile", fmt.Sprintf("unsupported profile %q", p.Spec.Profile))
	}
	for _, target := range p.Spec.Targets {
		if target != "codex" && target != "claude" {
			g.diag(p, "project.target", fmt.Sprintf("unsupported target %q", target))
		}
	}
	for entrypoint, refs := range p.Spec.RuleAdapters {
		if strings.TrimSpace(entrypoint) == "" {
			g.diag(p, "rule-adapter.name", "rule adapter entrypoint name is empty")
		}
		for _, ref := range refs {
			source := g.resolveRef(p, ref, "Rule", map[string]bool{"Rule": true, "Workflow": true, "Text": true}, "ruleAdapters["+entrypoint+"]")
			if source != nil && source.Kind != "Rule" && source.Kind != "Workflow" && source.Kind != "Text" {
				g.diag(p, "rule-adapter.kind", fmt.Sprintf("rule adapter source %s must be Rule, Workflow, or Text", source.Key()))
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
	areas := g.Project.Spec.Areas
	for _, a := range areas {
		for _, ns := range a.Imports {
			if !g.hasArea(ns) {
				g.diag(g.Project, "area.import", fmt.Sprintf("area %q imports unknown namespace %q", a.Name, ns))
			}
		}
	}
	for _, r := range g.Resources {
		if r.Kind == "Project" {
			continue
		}
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
		g.ResourceAreas[r.Key()] = *best
		if r.Metadata.Namespace != best.Name {
			g.diag(r, "area.namespace", fmt.Sprintf("resource namespace %q does not match owning area %q", r.Metadata.Namespace, best.Name))
		}
	}
}

func (g *Graph) hasArea(name string) bool {
	for _, a := range g.Project.Spec.Areas {
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
	for i := range g.Project.Spec.Areas {
		a := &g.Project.Spec.Areas[i]
		root := cleanPath(a.Path)
		if within(file, root) && (len(root) > bestLen || (len(root) == bestLen && (best == nil || a.Name < best.Name))) {
			best, bestLen = a, len(root)
		}
	}
	return best
}

func (g *Graph) resolveAreaRules() {
	for _, r := range g.Resources {
		if r.Kind == "Project" {
			continue
		}
		file := cleanPath(r.Path)
		var ancestors []Area
		for _, a := range g.Project.Spec.Areas {
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
	namespace := ref.Namespace
	if namespace == "" {
		namespace = area.Name
	}
	kind := ref.Kind
	if kind == "" {
		kind = "Rule"
	}
	if kind != "Rule" {
		g.diag(resource, "reference.kind", fmt.Sprintf("area.rules reference declares kind %s; expected Rule", kind))
		return
	}
	target := g.Resources[namespace+"/Rule/"+ref.Name]
	if target == nil {
		g.diag(resource, "reference.missing", fmt.Sprintf("area.rules reference %s/Rule/%s does not resolve", namespace, ref.Name))
		return
	}
	g.addRelationship(Relationship{
		From: resource.Key(), To: target.Key(), Relation: "area.rules",
		Path: g.Project.Path, Line: g.Project.Line, Reference: ref, Area: area.Name,
	})
}

func (g *Graph) resolveResources() {
	for _, r := range g.Resources {
		if r.Kind == "Project" {
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
	if allowed == nil {
		g.diag(source, "reference.field", fmt.Sprintf("%s is not allowed on %s", field, source.Kind))
		return nil
	}
	if ref.Name == "" {
		g.diag(source, "reference.name", fmt.Sprintf("%s reference has an empty name", field))
		return nil
	}
	if source.Kind != "Project" && ref.Namespace != "" && ref.Namespace != source.Metadata.Namespace {
		area := g.areaFor(source)
		if area == nil || !contains(area.Imports, ref.Namespace) {
			g.diag(source, "reference.scope", fmt.Sprintf("%s reference to namespace %q is not imported by area %q", field, ref.Namespace, areaName(area)))
			return nil
		}
	}
	requestedKind := ref.Kind
	if requestedKind == "" {
		requestedKind = defaultKind
	}
	var matches []*Resource
	for _, k := range sortedKeys(g.Resources) {
		target := g.Resources[k]
		if target.Metadata.Name != ref.Name {
			continue
		}
		if ref.Namespace != "" && target.Metadata.Namespace != ref.Namespace {
			continue
		}
		if ref.Namespace == "" && target.Metadata.Namespace != source.Metadata.Namespace {
			continue
		}
		if requestedKind != "" && target.Kind != requestedKind {
			continue
		}
		matches = append(matches, target)
	}
	if len(matches) == 0 {
		g.diag(source, "reference.missing", fmt.Sprintf("%s reference %s does not resolve", field, ref.Key(source.Metadata.Namespace, requestedKind)))
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
	if ref.Kind != "" && ref.Kind != target.Kind {
		g.diag(source, "reference.kind", fmt.Sprintf("%s declares kind %s but resolves to %s", field, ref.Kind, target.Kind))
		return nil
	}
	if target.Kind != "Project" {
		g.addRelationship(Relationship{
			From: source.Key(), To: target.Key(), Relation: field,
			Path: source.Path, Line: source.Line, Reference: ref,
		})
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
	bindings := map[string]Ref{}
	// Every resource that declares implements makes a compatibility claim, even
	// when this project selects a different implementation for the contract.
	for _, impl := range g.Resources {
		if impl.Kind != "Agent" && impl.Kind != "Skill" && impl.Kind != "Workflow" {
			continue
		}
		for _, ref := range impl.Spec.Implements {
			if contract := g.lookupRef(impl, ref, "Contract"); contract != nil {
				g.validateImplementation(impl, contract)
			}
		}
	}
	for _, b := range g.Project.Spec.Bindings {
		if b.Contract.Kind != "Contract" || b.Contract.Namespace == "" || b.Contract.Name == "" || b.Implementation.Kind == "" || b.Implementation.Namespace == "" || b.Implementation.Name == "" {
			g.diag(g.Project, "binding.qualified", "binding contract and implementation must be fully qualified")
			continue
		}
		contract := g.lookupExact(b.Contract)
		impl := g.lookupExact(b.Implementation)
		if contract == nil || impl == nil {
			g.diag(g.Project, "binding.target", fmt.Sprintf("binding %s -> %s has a missing target", b.Contract.Key("", "Contract"), b.Implementation.Key("", "")))
			continue
		}
		if contract.Kind != "Contract" {
			g.diag(g.Project, "binding.contract-kind", "binding contract target must have kind Contract")
			continue
		}
		if impl.Kind != "Agent" && impl.Kind != "Skill" && impl.Kind != "Workflow" {
			g.diag(g.Project, "binding.implementation-kind", fmt.Sprintf("%s cannot implement a contract", impl.Kind))
			continue
		}
		if previous, ok := bindings[contract.Key()]; ok && previous.Key("", "") != b.Implementation.Key("", "") {
			g.diag(g.Project, "binding.ambiguous", fmt.Sprintf("contract %s has more than one implementation", contract.Key()))
			continue
		}
		bindings[contract.Key()] = b.Implementation
		declared := false
		for _, ref := range impl.Spec.Implements {
			ns := ref.Namespace
			if ns == "" {
				ns = impl.Metadata.Namespace
			}
			if ref.Name == contract.Metadata.Name && ns == contract.Metadata.Namespace && (ref.Kind == "" || ref.Kind == "Contract") {
				declared = true
			}
		}
		if !declared {
			g.diag(impl, "contract.undeclared", fmt.Sprintf("implementation does not declare implements reference to %s", contract.Key()))
		}
		g.addRelationship(Relationship{
			From: impl.Key(), To: contract.Key(), Relation: "binding",
			Path: g.Project.Path, Line: g.Project.Line, Reference: b.Contract, Selected: true,
		})
	}
	for _, r := range g.Resources {
		if r.Kind != "Agent" && r.Kind != "Skill" && r.Kind != "Workflow" {
			continue
		}
		for _, need := range r.Spec.Needs {
			contract := g.lookupRef(r, need, "Contract")
			if contract == nil {
				continue
			}
			implementation, ok := bindings[contract.Key()]
			if !ok {
				g.diag(r, "binding.missing", fmt.Sprintf("needed contract %s has no project binding", contract.Key()))
				continue
			}
			impl := g.lookupExact(implementation)
			if impl != nil {
				g.addRelationship(Relationship{
					From: r.Key(), To: impl.Key(), Relation: "selected-implementation",
					Path: g.Project.Path, Line: g.Project.Line, Reference: implementation, Selected: true,
				})
				// The explicit needs edge was added by resolveResources. Keep the
				// compact edge present even if the declaration was malformed there.
				g.addEdge(r.Key(), contract.Key())
			}
		}
	}
}

func (g *Graph) validateImplementation(implementation, contract *Resource) {
	if contract.Spec.Kind != implementation.Kind {
		g.diag(implementation, "contract.kind", fmt.Sprintf("contract expects %s but implementation is %s", contract.Spec.Kind, implementation.Kind))
	}
	if !sameStrings(contract.Spec.Input, implementation.Spec.Input) || !sameStrings(contract.Spec.Output, implementation.Spec.Output) {
		g.diag(implementation, "contract.signature", fmt.Sprintf("implementation signature does not exactly match contract %s", contract.Key()))
	}
}

func (g *Graph) lookupRef(source *Resource, ref Ref, kind string) *Resource {
	ns := ref.Namespace
	if ns == "" {
		ns = source.Metadata.Namespace
	}
	return g.Resources[ns+"/"+kind+"/"+ref.Name]
}

func (g *Graph) lookupExact(ref Ref) *Resource {
	if ref.Kind == "" || ref.Namespace == "" {
		return nil
	}
	return g.Resources[ref.Key("", "")]
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
		if r.Kind != "Skill" && r.Kind != "Agent" {
			continue
		}
		if names[r.Kind] == nil {
			names[r.Kind] = map[string]*Resource{}
		}
		if prev, ok := names[r.Kind][r.Metadata.Name]; ok {
			g.diag(r, "provider.name", fmt.Sprintf("provider name %q is duplicated for kind %s (also %s)", r.Metadata.Name, r.Kind, prev.Key()))
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
					reachable[target.Key()] = true
				}
			}
		}
		changed := true
		for changed {
			changed = false
			for _, source := range g.Resources {
				if source.Kind != "Workflow" || !reachable[source.Key()] {
					continue
				}
				for _, ref := range source.Spec.Uses {
					if target := g.resolveUse(source, ref); target != nil && target.Kind == "Workflow" && !reachable[target.Key()] {
						reachable[target.Key()] = true
						changed = true
					}
				}
			}
		}
		for _, target := range g.Resources {
			if target.Kind == "Workflow" && !reachable[target.Key()] {
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
		for _, ref := range source.Spec.Uses {
			target := g.resolveUse(source, ref)
			if target != nil && (target.Kind == "Workflow" || target.Kind == "Skill" || target.Kind == "Agent") {
				adj[source.Key()] = append(adj[source.Key()], target.Key())
			}
		}
		if source.Kind == "Agent" || source.Kind == "Skill" || source.Kind == "Workflow" {
			for _, ref := range source.Spec.Needs {
				contract := g.lookupRef(source, ref, "Contract")
				if contract == nil {
					continue
				}
				for _, binding := range g.Project.Spec.Bindings {
					if binding.Contract.Key("", "Contract") == contract.Key() {
						if impl := g.lookupExact(binding.Implementation); impl != nil {
							adj[source.Key()] = append(adj[source.Key()], impl.Key())
						}
					}
				}
			}
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
	ns := ref.Namespace
	if ns == "" {
		ns = source.Metadata.Namespace
	}
	if ref.Kind != "" {
		return g.Resources[ns+"/"+ref.Kind+"/"+ref.Name]
	}
	var found *Resource
	for kind := range allowed {
		if target := g.Resources[ns+"/"+kind+"/"+ref.Name]; target != nil {
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
	if relationship.From == relationship.To {
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
	g.addDiagnostic(Diagnostic{Code: code, Path: r.Path, Line: r.Line, Message: message})
}
func (g *Graph) addDiagnostic(d Diagnostic) { g.Diagnostics = append(g.Diagnostics, d) }
