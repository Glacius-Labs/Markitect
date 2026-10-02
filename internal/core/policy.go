package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

const maxPolicyExceptions = 64

// EvaluateConstraints produces stable per-subject or collection results, then
// applies only source-bound exceptions to exact failing per-resource results.
func (g *Graph) EvaluateConstraints() {
	if g == nil || g.Registry == nil {
		return
	}
	retained := g.Diagnostics[:0]
	for _, diagnostic := range g.Diagnostics {
		if strings.HasPrefix(diagnostic.Code, "constraint.") || strings.HasPrefix(diagnostic.Code, "policy.exception.") || diagnostic.Code == "policy.date" {
			continue
		}
		retained = append(retained, diagnostic)
	}
	g.Diagnostics = retained
	g.PolicyResults = nil
	for _, domain := range g.Registry.Domains() {
		for _, constraint := range domain.Constraints {
			g.evaluateConstraint(domain, constraint)
		}
	}
	g.applyPolicyExceptions()
	sort.Slice(g.PolicyResults, func(i, j int) bool {
		a, b := g.PolicyResults[i], g.PolicyResults[j]
		if a.APIVersion != b.APIVersion {
			return a.APIVersion < b.APIVersion
		}
		if a.Constraint != b.Constraint {
			return a.Constraint < b.Constraint
		}
		return a.Subject < b.Subject
	})
}

func (g *Graph) evaluateConstraint(domain DomainDefinition, constraint ConstraintDefinition) {
	constraint = normalizeConstraint(constraint)
	constraintDigest, err := digestConstraint(domain, constraint)
	if err != nil {
		g.addDiagnostic(Diagnostic{Code: "constraint.digest", Path: domain.Path, Line: domain.Line, Message: fmt.Sprintf("constraint %q digest failed: %v", constraint.Name, err)})
		return
	}
	selected := g.selectedForConstraint(domain, constraint)
	assertion := constraint.Assert
	if assertion.Op == "unique" || assertion.Op == "count" && assertion.Scope != "resource" {
		message := evaluateCollectionConstraint(domain, constraint, selected)
		result := PolicyResult{APIVersion: domain.APIVersion, Constraint: constraint.Name, Status: PolicyPassed, Message: message, ConstraintDigest: constraintDigest}
		result.SubjectDigest, err = digestSelection(selected)
		if err != nil {
			g.addDiagnostic(Diagnostic{Code: "constraint.digest", Path: domain.Path, Line: domain.Line, Message: fmt.Sprintf("constraint %q selection digest failed: %v", constraint.Name, err)})
			return
		}
		if message != "" {
			result.Status = PolicyFailed
			g.addDiagnostic(Diagnostic{Code: "constraint." + constraint.Name, Path: domain.Path, Line: domain.Line, Message: fmt.Sprintf("%s: %s", domain.APIVersion, message)})
		}
		g.PolicyResults = append(g.PolicyResults, result)
		return
	}
	for _, resource := range selected {
		message := evaluateResourceConstraint(g, domain, constraint, resource)
		subjectDigest, digestErr := digestResource(resource)
		if digestErr != nil {
			g.addDiagnostic(Diagnostic{Code: "constraint.digest", Path: resource.Path, Package: resource.Package, Line: resource.Line, Message: fmt.Sprintf("constraint %q subject digest failed: %v", constraint.Name, digestErr)})
			continue
		}
		result := PolicyResult{APIVersion: domain.APIVersion, Constraint: constraint.Name, Subject: resource.GraphKey(), Status: PolicyPassed, Message: message, ConstraintDigest: constraintDigest, SubjectDigest: subjectDigest}
		if message != "" {
			result.Status = PolicyFailed
			g.diag(resource, "constraint."+constraint.Name, fmt.Sprintf("%s: %s", domain.APIVersion, message))
		}
		g.PolicyResults = append(g.PolicyResults, result)
	}
}

func normalizeConstraint(constraint ConstraintDefinition) ConstraintDefinition {
	if constraint.Assert.Op == "count" && constraint.Assert.Scope == "" {
		constraint.Assert.Scope = "selection"
	}
	return constraint
}

// digestConstraint binds a policy exception to the semantics of the relation
// it consumes, not only to the assertion's relation name. A relation can keep
// its name while changing its field, endpoints, or graph effects.
func digestConstraint(domain DomainDefinition, constraint ConstraintDefinition) (string, error) {
	constraint = normalizeConstraint(constraint)
	type relationBinding struct {
		APIVersion string             `yaml:"apiVersion"`
		Name       string             `yaml:"name"`
		Definition RelationDefinition `yaml:"definition"`
	}
	type canonicalConstraint struct {
		APIVersion string               `yaml:"apiVersion"`
		Constraint ConstraintDefinition `yaml:"constraint"`
		Relation   *relationBinding     `yaml:"relation,omitempty"`
	}
	canonical := canonicalConstraint{APIVersion: domain.APIVersion, Constraint: constraint}
	if constraint.Assert.Relation != "" {
		relation, exists := domain.Relations[constraint.Assert.Relation]
		if !exists {
			return "", fmt.Errorf("constraint relation %q is undefined", constraint.Assert.Relation)
		}
		canonical.Relation = &relationBinding{APIVersion: domain.APIVersion, Name: constraint.Assert.Relation, Definition: relation}
	}
	return digestYAML(canonical)
}

func (g *Graph) selectedForConstraint(domain DomainDefinition, constraint ConstraintDefinition) []*Resource {
	selected := []*Resource{}
	var relationSources map[string]bool
	assertion := constraint.Assert
	if assertion.Relation != "" && (assertion.Op == "count" || assertion.Op == "allowed-targets") {
		relationSources = map[string]bool{}
		for _, kind := range domain.Relations[assertion.Relation].SourceKinds {
			relationSources[kind] = true
		}
	}
	for _, key := range sortedKeys(g.Resources) {
		resource := g.Resources[key]
		if resource.APIVersion != domain.APIVersion || constraint.Select.Kind != "" && resource.Kind != constraint.Select.Kind {
			continue
		}
		if len(relationSources) > 0 && !relationSources[resource.Kind] {
			continue
		}
		matches := true
		for label, expected := range constraint.Select.Labels {
			actual, exists := resource.Metadata.Labels[label]
			if !exists || actual != expected {
				matches = false
				break
			}
		}
		if matches {
			selected = append(selected, resource)
		}
	}
	return selected
}

func evaluateCollectionConstraint(domain DomainDefinition, constraint ConstraintDefinition, selected []*Resource) string {
	a := constraint.Assert
	if a.Op == "count" {
		count := len(selected)
		if a.Relation != "" {
			count = 0
			relation := domain.Relations[a.Relation].Field
			for _, resource := range selected {
				count += relationCount(resource.Data[relation])
			}
		}
		if outsideBounds(count, a.Min, a.Max) {
			return fmt.Sprintf("selection count is %d outside declared bounds", count)
		}
		return ""
	}
	seen := map[string]string{}
	for _, resource := range selected {
		value, exists := resource.Data[a.Field]
		if !exists {
			continue
		}
		key := fmt.Sprintf("%T:%v", value, value)
		if prior, duplicate := seen[key]; duplicate {
			return fmt.Sprintf("field %s is not unique (%s and %s)", a.Field, prior, resource.GraphKey())
		}
		seen[key] = resource.GraphKey()
	}
	return ""
}

func evaluateResourceConstraint(graph *Graph, domain DomainDefinition, constraint ConstraintDefinition, resource *Resource) string {
	a := constraint.Assert
	switch a.Op {
	case "present":
		value, exists := resource.Data[a.Field]
		if !exists || value == nil {
			return fmt.Sprintf("%s is missing %s", resource.GraphKey(), a.Field)
		}
	case "equal":
		value, exists := resource.Data[a.Field]
		if !exists || !reflect.DeepEqual(value, a.Value) {
			return fmt.Sprintf("%s field %s does not equal required value", resource.GraphKey(), a.Field)
		}
	case "allowed":
		value, exists := resource.Data[a.Field]
		if !exists {
			return fmt.Sprintf("%s is missing %s", resource.GraphKey(), a.Field)
		}
		for _, candidate := range a.Values {
			if reflect.DeepEqual(value, candidate) {
				return ""
			}
		}
		return fmt.Sprintf("%s field %s has a disallowed value", resource.GraphKey(), a.Field)
	case "allowed-targets":
		for _, relationship := range graph.Relationships {
			if relationship.From != resource.GraphKey() || relationship.Relation != a.Relation || relationship.DomainAPIVersion != domain.APIVersion {
				continue
			}
			kind := relationship.Reference.Kind
			allowed := false
			for _, candidate := range a.Values {
				if k, ok := candidate.(string); ok && kind == k {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Sprintf("%s relation %s targets disallowed kind %q", resource.GraphKey(), a.Relation, kind)
			}
		}
	case "count":
		relation := domain.Relations[a.Relation]
		count := relationCount(resource.Data[relation.Field])
		if outsideBounds(count, a.Min, a.Max) {
			return fmt.Sprintf("%s relation %s has %d targets outside declared bounds", resource.GraphKey(), a.Relation, count)
		}
	}
	return ""
}

func relationCount(value any) int {
	switch refs := value.(type) {
	case []any:
		return len(refs)
	case map[string]any:
		return 1
	default:
		return 0
	}
}

func relationValues(value any) []map[string]any {
	if value == nil {
		return nil
	}
	if refs, ok := value.([]any); ok {
		out := make([]map[string]any, 0, len(refs))
		for _, ref := range refs {
			if m, ok := asMap(ref); ok {
				out = append(out, m)
			}
		}
		return out
	}
	if ref, ok := asMap(value); ok {
		return []map[string]any{ref}
	}
	return nil
}

func outsideBounds(value int, min, max *int) bool {
	return min != nil && value < *min || max != nil && value > *max
}

func (g *Graph) applyPolicyExceptions() {
	if g.Project == nil {
		return
	}
	exceptions := g.Project.Spec.PolicyExceptions
	if len(exceptions) > maxPolicyExceptions {
		g.diag(g.Project, "policy.exception.limit", fmt.Sprintf("at most %d policy exceptions are supported", maxPolicyExceptions))
		return
	}
	policyDate := g.Project.Spec.PolicyDate
	if policyDate != "" {
		if _, err := time.Parse("2006-01-02", policyDate); err != nil {
			g.diag(g.Project, "policy.date", "policyDate must be a valid YYYY-MM-DD date")
			policyDate = ""
		}
	}
	for _, exception := range exceptions {
		if exception.ExpiresOn != "" && policyDate == "" {
			g.diag(g.Project, "policy.exception.date-required", fmt.Sprintf("policy exception %q has expiresOn but Project.spec.policyDate is missing or invalid", exception.Name))
		}
	}
	nameCounts, targetCounts := map[string]int{}, map[string]int{}
	for _, exception := range exceptions {
		nameCounts[exception.Name]++
		targetCounts[exceptionTargetKey(exception)]++
	}
	for index, exception := range exceptions {
		if nameCounts[exception.Name] > 1 || targetCounts[exceptionTargetKey(exception)] > 1 {
			g.diag(g.Project, "policy.exception.duplicate", fmt.Sprintf("policy exception %q duplicates a name or target", exception.Name))
			continue
		}
		if !validDNS(exception.Name) || !validAPIVersion(exception.APIVersion) || !validIdentifier(exception.Constraint) || exception.Subject == "" || strings.TrimSpace(exception.Rationale) == "" || strings.TrimSpace(exception.Owner) == "" || strings.TrimSpace(exception.Decision) == "" || !validPolicyDigest(exception.ConstraintDigest) || !validPolicyDigest(exception.SubjectDigest) {
			g.diag(g.Project, "policy.exception.invalid", fmt.Sprintf("policy exception at index %d has invalid or missing required values", index))
			continue
		}
		if exception.ExpiresOn != "" {
			if policyDate == "" {
				continue // The missing/invalid pinned date already has a finding.
			}
			expiresOn, err := time.Parse("2006-01-02", exception.ExpiresOn)
			date, dateErr := time.Parse("2006-01-02", policyDate)
			if err != nil {
				g.diag(g.Project, "policy.exception.expiry", fmt.Sprintf("policy exception %q expiresOn must be a valid YYYY-MM-DD date", exception.Name))
				continue
			}
			if dateErr != nil || !date.Before(expiresOn) {
				g.diag(g.Project, "policy.exception.expired", fmt.Sprintf("policy exception %q expired on or before policyDate %s", exception.Name, policyDate))
				continue
			}
		}
		resultIndex := g.policyResultIndex(exception.APIVersion, exception.Constraint, exception.Subject)
		if resultIndex < 0 {
			if collectionIndex := g.policyResultIndex(exception.APIVersion, exception.Constraint, ""); collectionIndex >= 0 {
				g.diag(g.Project, "policy.exception.not-waivable", fmt.Sprintf("policy exception %q targets a collection constraint that cannot be waived", exception.Name))
				continue
			}
			if domain, constraint, exists := g.findPolicyConstraint(exception.APIVersion, exception.Constraint); exists && g.Resources[exception.Subject] != nil {
				currentConstraintDigest, constraintErr := digestConstraint(domain, constraint)
				currentSubjectDigest, subjectErr := digestResource(g.Resources[exception.Subject])
				if constraintErr != nil || subjectErr != nil || exception.ConstraintDigest != currentConstraintDigest || exception.SubjectDigest != currentSubjectDigest {
					g.diag(g.Project, "policy.exception.stale", fmt.Sprintf("policy exception %q is stale; constraint or subject content changed", exception.Name))
					continue
				}
				g.diag(g.Project, "policy.exception.unneeded", fmt.Sprintf("policy exception %q no longer matches a selected subject", exception.Name))
				continue
			}
			g.diag(g.Project, "policy.exception.unknown", fmt.Sprintf("policy exception %q does not identify a known per-resource constraint and subject", exception.Name))
			continue
		}
		result := &g.PolicyResults[resultIndex]
		if exception.ConstraintDigest != result.ConstraintDigest || exception.SubjectDigest != result.SubjectDigest {
			g.diag(g.Project, "policy.exception.stale", fmt.Sprintf("policy exception %q is stale; constraint or subject content changed", exception.Name))
			continue
		}
		if result.Subject == "" {
			g.diag(g.Project, "policy.exception.not-waivable", fmt.Sprintf("policy exception %q targets a collection result that cannot be waived", exception.Name))
			continue
		}
		if result.Status == PolicyPassed {
			g.diag(g.Project, "policy.exception.unneeded", fmt.Sprintf("policy exception %q is no longer needed", exception.Name))
			continue
		}
		if result.Status != PolicyFailed {
			g.diag(g.Project, "policy.exception.not-waivable", fmt.Sprintf("policy exception %q does not target a failing per-resource result", exception.Name))
			continue
		}
		result.Status = PolicyWaived
		result.ExceptionName = exception.Name
		result.Rationale = exception.Rationale
		result.Owner = exception.Owner
		result.Decision = exception.Decision
		result.ExpiresOn = exception.ExpiresOn
		result.PolicyDate = policyDate
		g.removeConstraintDiagnostic(result)
	}
}

func (g *Graph) policyResultIndex(apiVersion, constraint, subject string) int {
	for i := range g.PolicyResults {
		result := &g.PolicyResults[i]
		if result.APIVersion == apiVersion && result.Constraint == constraint && result.Subject == subject {
			return i
		}
	}
	return -1
}

func (g *Graph) findPolicyConstraint(apiVersion, constraint string) (DomainDefinition, ConstraintDefinition, bool) {
	if g.Registry == nil {
		return DomainDefinition{}, ConstraintDefinition{}, false
	}
	domain, ok := g.Registry.Domain(apiVersion)
	if !ok {
		return DomainDefinition{}, ConstraintDefinition{}, false
	}
	for _, candidate := range domain.Constraints {
		if candidate.Name == constraint {
			return domain, candidate, true
		}
	}
	return domain, ConstraintDefinition{}, false
}

func (g *Graph) removeConstraintDiagnostic(result *PolicyResult) {
	resource := g.Resources[result.Subject]
	if resource == nil {
		return
	}
	message := fmt.Sprintf("%s: ", result.APIVersion)
	for i, diagnostic := range g.Diagnostics {
		if diagnostic.Code == "constraint."+result.Constraint && diagnostic.Path == resource.Path && diagnostic.Package == resource.Package && diagnostic.Line == resource.Line && diagnostic.Message == message+result.Message {
			g.Diagnostics = append(g.Diagnostics[:i], g.Diagnostics[i+1:]...)
			return
		}
	}
}

func exceptionTargetKey(exception PolicyException) string {
	return exception.APIVersion + "\x00" + exception.Constraint + "\x00" + exception.Subject
}

func validPolicyDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	encoded := strings.TrimPrefix(value, "sha256:")
	decoded, err := hex.DecodeString(encoded)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == encoded
}

func digestSelection(resources []*Resource) (string, error) {
	type member struct {
		Subject string `yaml:"subject"`
		Digest  string `yaml:"digest"`
	}
	members := make([]member, 0, len(resources))
	for _, resource := range resources {
		digest, err := digestResource(resource)
		if err != nil {
			return "", err
		}
		members = append(members, member{Subject: resource.GraphKey(), Digest: digest})
	}
	return digestYAML(members)
}

func digestResource(resource *Resource) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("resource is nil")
	}
	data, err := yaml.Marshal(resource)
	if err != nil {
		return "", err
	}
	type canonicalSubject struct {
		Identity string `yaml:"identity"`
		Resource string `yaml:"resource"`
	}
	return digestYAML(canonicalSubject{Identity: resource.GraphKey(), Resource: string(data)})
}

func digestYAML(value any) (string, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
