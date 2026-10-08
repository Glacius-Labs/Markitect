package core

import (
	"fmt"
	"sort"
	"strings"
)

func validateSameTargetPath(constraint, side string, names []string, selectedKind string, domain DomainDefinition) error {
	if len(names) < 1 || len(names) > 2 {
		return fmt.Errorf("constraint %q same-target %s path must contain one or two relations", constraint, side)
	}
	possibleSources := []string{selectedKind}
	for _, name := range names {
		relation, exists := domain.Relations[name]
		if !exists {
			return fmt.Errorf("constraint %q same-target %s path refers to undefined relation %q", constraint, side, name)
		}
		for _, source := range possibleSources {
			if !contains(relation.SourceKinds, source) {
				return fmt.Errorf("constraint %q same-target %s relation %q cannot accept possible source kind %q", constraint, side, name, source)
			}
		}
		if contains(relation.TargetKinds, "*") {
			return fmt.Errorf("constraint %q same-target %s relation %q cannot use wildcard targetKinds", constraint, side, name)
		}
		possibleSources = append([]string(nil), relation.TargetKinds...)
	}
	return nil
}

func terminalKinds(names []string, domain DomainDefinition) []string {
	if len(names) == 0 {
		return nil
	}
	kinds := append([]string(nil), domain.Relations[names[len(names)-1]].TargetKinds...)
	sort.Strings(kinds)
	return kinds
}

func (g *Graph) evaluateSameTarget(domain DomainDefinition, constraint ConstraintDefinition, resource *Resource) (PolicyResult, bool) {
	assertion := constraint.Assert
	left, leftResources, leftOK := g.walkTargetPath(domain, constraint.Name, resource, "left", assertion.Left)
	right, rightResources, rightOK := g.walkTargetPath(domain, constraint.Name, resource, "right", assertion.Right)
	if !leftOK || !rightOK {
		if g.invalidPolicyPaths == nil {
			g.invalidPolicyPaths = map[string]bool{}
		}
		g.invalidPolicyPaths[exceptionTargetKey(PolicyException{APIVersion: domain.APIVersion, Constraint: constraint.Name, Subject: resource.GraphKey()})] = true
		return PolicyResult{}, false
	}
	traceResources := append([]*Resource{resource}, leftResources...)
	traceResources = append(traceResources, rightResources...)
	subjectDigest, err := g.digestTargetComparison(resource, traceResources, left, right)
	if err != nil {
		g.diag(resource, "constraint.digest", fmt.Sprintf("constraint %q path subject digest failed: %v", constraint.Name, err))
		return PolicyResult{}, false
	}
	message := ""
	if left.Target != right.Target {
		message = fmt.Sprintf("constraint %q requires both paths to resolve to the same canonical resource identity; left %s resolves to %s, right %s resolves to %s", constraint.Name, formatTargetPath(resource.GraphKey(), left), left.Target, formatTargetPath(resource.GraphKey(), right), right.Target)
	}
	constraintDigest, err := digestSameTargetConstraint(domain, constraint)
	if err != nil {
		g.diag(resource, "constraint.digest", fmt.Sprintf("constraint %q digest failed: %v", constraint.Name, err))
		return PolicyResult{}, false
	}
	result := PolicyResult{APIVersion: domain.APIVersion, Constraint: constraint.Name, Subject: resource.GraphKey(), Status: PolicyPassed, Message: message, ConstraintDigest: constraintDigest, SubjectDigest: subjectDigest, Comparison: &TargetComparison{Left: left, Right: right}}
	if message != "" {
		result.Status = PolicyFailed
		g.addDiagnostic(Diagnostic{Code: "constraint." + constraint.Name, Path: resource.Path, Package: resource.Package, Line: resource.Line, Message: fmt.Sprintf("%s: %s", domain.APIVersion, message), PolicyResult: &PolicyResultRef{APIVersion: result.APIVersion, Constraint: result.Constraint, Subject: result.Subject}})
	}
	return result, true
}

func (g *Graph) walkTargetPath(domain DomainDefinition, constraint string, subject *Resource, side string, names []string) (TargetPath, []*Resource, bool) {
	trace := TargetPath{Relations: append([]string(nil), names...)}
	resources := []*Resource{}
	current := subject
	for _, name := range names {
		definition := domain.Relations[name]
		g.addPolicyDependency(PolicyDependency{Subject: subject.GraphKey(), Input: current.GraphKey(), APIVersion: domain.APIVersion, Constraint: constraint, Relation: name, Path: current.Path, Line: current.Line})
		if current.APIVersion != domain.APIVersion || !contains(definition.SourceKinds, current.Kind) {
			g.pathDiagnostic(subject, current, domain, constraint, side, names, name, "does not accept source kind "+current.Kind)
			return trace, resources, false
		}
		raw := relationValues(current.Data[definition.Field])
		resolved := []Relationship{}
		for _, relationship := range g.Relationships {
			if relationship.From == current.GraphKey() && relationship.Relation == name && relationship.DomainAPIVersion == domain.APIVersion {
				resolved = append(resolved, relationship)
			}
		}
		if len(raw) != 1 || len(resolved) != 1 {
			g.pathDiagnostic(subject, current, domain, constraint, side, names, name, fmt.Sprintf("requires exactly one raw and resolved target (found %d raw, %d resolved)", len(raw), len(resolved)))
			return trace, resources, false
		}
		relationship := resolved[0]
		target := g.Resources[relationship.To]
		if target == nil || target.APIVersion != domain.APIVersion || !contains(definition.TargetKinds, target.Kind) {
			g.pathDiagnostic(subject, current, domain, constraint, side, names, name, "resolved target has an incompatible kind or API version")
			return trace, resources, false
		}
		trace.Steps = append(trace.Steps, TargetStep{From: current.GraphKey(), To: target.GraphKey(), Relation: name, DomainAPIVersion: domain.APIVersion, Path: current.Path, Line: current.Line})
		g.addPolicyDependency(PolicyDependency{Subject: subject.GraphKey(), Input: target.GraphKey(), APIVersion: domain.APIVersion, Constraint: constraint, Relation: name, Path: current.Path, Line: current.Line})
		resources = append(resources, target)
		current = target
	}
	trace.Target = current.GraphKey()
	return trace, resources, true
}

func (g *Graph) pathDiagnostic(subject, current *Resource, domain DomainDefinition, constraint, side string, names []string, relation, reason string) {
	g.addDiagnostic(Diagnostic{Code: "constraint.path", Path: current.Path, Package: current.Package, Line: current.Line, Message: fmt.Sprintf("%s: constraint %q for subject %s same-target %s path [%s] failed at source %s, relation %q: %s", domain.APIVersion, constraint, subject.GraphKey(), side, strings.Join(names, " -> "), current.GraphKey(), relation, reason)})
}

func formatTargetPath(subject string, path TargetPath) string {
	parts := []string{subject}
	for _, step := range path.Steps {
		parts = append(parts, "-"+step.Relation+"->", step.To)
	}
	return strings.Join(parts, " ")
}

func (g *Graph) addPolicyDependency(dependency PolicyDependency) {
	for _, existing := range g.PolicyDependencies {
		if existing == dependency {
			return
		}
	}
	g.PolicyDependencies = append(g.PolicyDependencies, dependency)
}

func digestSameTargetConstraint(domain DomainDefinition, constraint ConstraintDefinition) (string, error) {
	type pathBinding struct {
		Relation      string                        `yaml:"relation"`
		Definition    RelationDefinition            `yaml:"definition"`
		SourceSchemas map[string]PropertyDefinition `yaml:"sourceSchemas"`
	}
	type canonical struct {
		APIVersion string               `yaml:"apiVersion"`
		Constraint ConstraintDefinition `yaml:"constraint"`
		Left       []pathBinding        `yaml:"left"`
		Right      []pathBinding        `yaml:"right"`
	}
	result := canonical{APIVersion: domain.APIVersion, Constraint: constraint}
	for _, side := range [][]string{constraint.Assert.Left, constraint.Assert.Right} {
		bindings := make([]pathBinding, 0, len(side))
		for _, name := range side {
			relation, exists := domain.Relations[name]
			if !exists {
				return "", fmt.Errorf("relation %q is undefined", name)
			}
			schemas := map[string]PropertyDefinition{}
			for _, sourceKind := range relation.SourceKinds {
				kind, exists := domain.Kinds[sourceKind]
				if !exists {
					return "", fmt.Errorf("relation %q refers to undefined source kind %q", name, sourceKind)
				}
				property, exists := kind.Properties[relation.Field]
				if !exists {
					return "", fmt.Errorf("relation %q refers to undefined field %q on source kind %q", name, relation.Field, sourceKind)
				}
				schemas[sourceKind] = property
			}
			bindings = append(bindings, pathBinding{Relation: name, Definition: relation, SourceSchemas: schemas})
		}
		if len(result.Left) == 0 {
			result.Left = bindings
		} else {
			result.Right = bindings
		}
	}
	return digestYAML(result)
}

func (g *Graph) digestTargetComparison(subject *Resource, resources []*Resource, left, right TargetPath) (string, error) {
	type member struct {
		Identity string `yaml:"identity"`
		Digest   string `yaml:"digest"`
	}
	unique := map[string]*Resource{}
	for _, resource := range resources {
		if resource != nil {
			unique[resource.GraphKey()] = resource
		}
	}
	members := make([]member, 0, len(unique))
	for _, key := range sortedKeys(unique) {
		digest, err := digestResource(unique[key], g.subjectDigestEncodings[key])
		if err != nil {
			return "", err
		}
		members = append(members, member{Identity: key, Digest: digest})
	}
	// Source paths and line numbers remain in the public trace, not in this semantic digest.
	type semanticStep struct{ From, To, Relation, DomainAPIVersion string }
	semantic := func(path TargetPath) []semanticStep {
		steps := make([]semanticStep, 0, len(path.Steps))
		for _, step := range path.Steps {
			steps = append(steps, semanticStep{step.From, step.To, step.Relation, step.DomainAPIVersion})
		}
		return steps
	}
	return digestYAML(struct {
		Subject string         `yaml:"subject"`
		Members []member       `yaml:"members"`
		Left    []semanticStep `yaml:"left"`
		Right   []semanticStep `yaml:"right"`
	}{Subject: subject.GraphKey(), Members: members, Left: semantic(left), Right: semantic(right)})
}
