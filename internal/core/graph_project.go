package core

import (
	"fmt"
	"regexp"
	"strings"
)

var providerAdapterName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

func (g *Graph) validateProject() {
	p := g.Project
	if adapters := p.Spec.ProviderAdapters; adapters != nil {
		for name, sources := range adapters.RuleSources {
			if len(name) > 63 || !providerAdapterName.MatchString(name) || len(sources) == 0 {
				g.diag(p, "provider-adapter.rule", fmt.Sprintf("rule adapter %q needs a valid name and at least one source", name))
			}
		}
		for _, group := range []struct {
			kind    string
			retired []string
		}{{"Skill", adapters.RetiredSkills}, {"Agent", adapters.RetiredAgents}} {
			seenRetired := map[string]bool{}
			for _, name := range group.retired {
				if len(name) > 63 || !providerAdapterName.MatchString(name) {
					g.diag(p, "provider-adapter.retired-name", fmt.Sprintf("retired %s name %q is invalid", group.kind, name))
				}
				if seenRetired[name] {
					g.diag(p, "provider-adapter.retired-duplicate", fmt.Sprintf("retired %s name %q is duplicated", group.kind, name))
				}
				seenRetired[name] = true
				for _, resource := range g.Resources {
					if resource.Package == "" && resource.Kind == group.kind && resource.Metadata.Name == name {
						g.diag(p, "provider-adapter.retired-active", fmt.Sprintf("retired %s name %q is still an active resource", group.kind, name))
					}
				}
			}
		}
	}
	for _, target := range p.Spec.Targets {
		if target != "codex" && target != "claude" && target != "markdown" {
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
		if a.Name == "." || a.Name == ".." || strings.ContainsAny(a.Name, `/\\:`) || strings.ContainsRune(a.Name, 0) {
			g.diag(p, "area.invalid", fmt.Sprintf("area name %q must be a safe single path segment", a.Name))
		}
		if seen[a.Name] {
			g.diag(p, "area.duplicate", fmt.Sprintf("duplicate area %q", a.Name))
		}
		seen[a.Name] = true
	}
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
