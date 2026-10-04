package host

import (
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

// PolicyResultChange compares one stable Domain constraint/subject identity
// across two fixed snapshots. A missing side is explicitly not-applicable;
// it is never presented as a passing PolicyResult.
type PolicyResultChange struct {
	APIVersion string           `yaml:"apiVersion"`
	Constraint string           `yaml:"constraint"`
	Subject    string           `yaml:"subject,omitempty"`
	Base       PolicyResultSide `yaml:"base"`
	Candidate  PolicyResultSide `yaml:"candidate"`
}

// PolicyResultSide keeps the result and the exact policy definition that
// interpreted it in that snapshot. Package relocation is provenance, not a
// policy-result change by itself.
type PolicyResultSide struct {
	Status        string                     `yaml:"status"`
	AbsenceReason string                     `yaml:"absenceReason,omitempty"`
	Result        *core.PolicyResult         `yaml:"result,omitempty"`
	DomainInput   *core.ModelDomainInput     `yaml:"domainInput,omitempty"`
	PackagePin    *core.PackagePin           `yaml:"packagePin,omitempty"`
	Definition    *core.ConstraintDefinition `yaml:"definition,omitempty"`
}

// AnalyzeImpact is the explicit read-only workflow for structurally valid
// snapshots, including snapshots with ordinary failed policy results. It uses
// the same compiled models as model/context and preserves Changes' conservative
// affected-resource semantics.
func AnalyzeImpact(before, after *Project) (*Impact, error) {
	if before == nil || after == nil || before.Snapshot == nil || after.Snapshot == nil {
		return nil, fmt.Errorf("impact analysis requires two parsed projects with fixed snapshots")
	}
	if diagnostics := before.StructuralDiagnostics(); len(diagnostics) > 0 {
		return nil, fmt.Errorf("base has structural diagnostics; impact analysis is blocked")
	}
	if diagnostics := after.StructuralDiagnostics(); len(diagnostics) > 0 {
		return nil, fmt.Errorf("candidate has structural diagnostics; impact analysis is blocked")
	}
	baseModel, err := CompileModel(before)
	if err != nil {
		return nil, fmt.Errorf("compile base semantic model: %w", err)
	}
	candidateModel, err := CompileModel(after)
	if err != nil {
		return nil, fmt.Errorf("compile candidate semantic model: %w", err)
	}
	result := Changes(before, after)
	baseAnalysis := analysisSnapshot(baseModel)
	result.Analysis = &AnalysisEvidence{
		Mode: "policy-failure-analysis", Complete: true,
		Base: &baseAnalysis, Candidate: analysisSnapshot(candidateModel),
		Notice: "Read-only impact analysis is not verification or acceptance. Policy failures remain failures.",
	}
	result.PolicyChanges, result.DirectPolicySubjects, err = policyResultChanges(before, baseModel, after, candidateModel)
	if err != nil {
		return nil, err
	}
	directCount, affectedCount := len(result.DirectPolicySubjects), len(result.Affected)
	result.DirectPolicySubjectCount = &directCount
	result.AffectedCount = &affectedCount
	return result, nil
}

type policyResultIdentity struct {
	api, constraint, subject string
}

func policyResultChanges(before *Project, beforeModel core.SemanticModel, after *Project, afterModel core.SemanticModel) ([]PolicyResultChange, []string, error) {
	base := indexPolicyResults(beforeModel.PolicyResults)
	candidate := indexPolicyResults(afterModel.PolicyResults)
	identities := make(map[policyResultIdentity]bool, len(base)+len(candidate))
	for identity := range base {
		identities[identity] = true
	}
	for identity := range candidate {
		identities[identity] = true
	}
	ordered := make([]policyResultIdentity, 0, len(identities))
	for identity := range identities {
		ordered = append(ordered, identity)
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.api != b.api {
			return a.api < b.api
		}
		if a.constraint != b.constraint {
			return a.constraint < b.constraint
		}
		return a.subject < b.subject
	})

	changes := make([]PolicyResultChange, 0)
	subjects := map[string]bool{}
	for _, identity := range ordered {
		left, hasLeft := base[identity]
		right, hasRight := candidate[identity]
		if hasLeft && hasRight && policyResultSemanticallyEqual(left, right) {
			continue
		}
		leftSide, err := policyResultSide(before, beforeModel, identity, left, hasLeft)
		if err != nil {
			return nil, nil, fmt.Errorf("base policy %s/%s: %w", identity.api, identity.constraint, err)
		}
		rightSide, err := policyResultSide(after, afterModel, identity, right, hasRight)
		if err != nil {
			return nil, nil, fmt.Errorf("candidate policy %s/%s: %w", identity.api, identity.constraint, err)
		}
		changes = append(changes, PolicyResultChange{APIVersion: identity.api, Constraint: identity.constraint, Subject: identity.subject, Base: leftSide, Candidate: rightSide})
		if identity.subject != "" {
			subjects[identity.subject] = true
		}
	}
	direct := make([]string, 0, len(subjects))
	for subject := range subjects {
		direct = append(direct, subject)
	}
	sort.Strings(direct)
	return changes, direct, nil
}

func indexPolicyResults(results []core.PolicyResult) map[policyResultIdentity]core.PolicyResult {
	indexed := make(map[policyResultIdentity]core.PolicyResult, len(results))
	for _, result := range results {
		indexed[policyResultIdentity{result.APIVersion, result.Constraint, result.Subject}] = result
	}
	return indexed
}

func policyResultSide(project *Project, model core.SemanticModel, identity policyResultIdentity, result core.PolicyResult, exists bool) (PolicyResultSide, error) {
	side := PolicyResultSide{}
	if exists {
		side.Status = result.Status
		copy := clonePolicyResults([]core.PolicyResult{result})[0]
		side.Result = &copy
	} else {
		side.Status = "not-applicable"
		side.AbsenceReason = policyResultAbsence(project, model, identity)
		if side.AbsenceReason == "" {
			return PolicyResultSide{}, fmt.Errorf("normalized model has no result although the constraint selects the subject")
		}
	}
	input, hasInput, _, constraint, hasConstraint := modelPolicyDefinition(model, identity.api, identity.constraint)
	if hasInput {
		inputCopy := input
		side.DomainInput = &inputCopy
		if input.Package != "" {
			for _, pin := range project.Graph.Project.Spec.Packages {
				if pin.Name == input.Package {
					pinCopy := pin
					side.PackagePin = &pinCopy
					break
				}
			}
		}
	}
	if hasConstraint {
		definitionCopy := constraint
		side.Definition = &definitionCopy
	}
	return side, nil
}

func policyResultAbsence(project *Project, model core.SemanticModel, identity policyResultIdentity) string {
	_, _, _, constraint, defined := modelPolicyDefinition(model, identity.api, identity.constraint)
	if !defined {
		return "constraint-not-defined"
	}
	collection := constraintProducesCollectionResult(constraint)
	if (identity.subject == "") != collection {
		return "result-scope-changed"
	}
	if identity.subject == "" {
		return "result-not-present"
	}
	var subject *core.ModelResource
	for i := range model.Resources {
		if model.Resources[i].Identity.Key == identity.subject {
			subject = &model.Resources[i]
			break
		}
	}
	if subject == nil {
		return "subject-absent"
	}
	if subject.Identity.APIVersion != identity.api || !constraintSelects(constraint, *subject, model) {
		return "not-selected"
	}
	return ""
}

// constraintProducesCollectionResult mirrors the existing finite evaluator's
// result shape: unique and selection-scoped count produce one subjectless
// outcome; all other supported assertions produce per-subject outcomes.
func constraintProducesCollectionResult(constraint core.ConstraintDefinition) bool {
	assertion := constraint.Assert
	return assertion.Op == "unique" || assertion.Op == "count" && assertion.Scope != "resource"
}

func modelPolicyDefinition(model core.SemanticModel, apiVersion, name string) (core.ModelDomainInput, bool, core.ModelDomain, core.ConstraintDefinition, bool) {
	var input core.ModelDomainInput
	inputFound := false
	for _, candidate := range model.DomainInputs {
		if candidate.APIVersion == apiVersion {
			input, inputFound = candidate, true
			break
		}
	}
	for _, domain := range model.Domains {
		if domain.APIVersion != apiVersion {
			continue
		}
		for _, constraint := range domain.Constraints {
			if constraint.Name == name {
				return input, inputFound, domain, constraint, true
			}
		}
	}
	return input, inputFound, core.ModelDomain{}, core.ConstraintDefinition{}, false
}

func constraintSelects(constraint core.ConstraintDefinition, subject core.ModelResource, model core.SemanticModel) bool {
	if constraint.Select.Kind != "" && subject.Identity.Kind != constraint.Select.Kind {
		return false
	}
	for name, expected := range constraint.Select.Labels {
		if subject.Labels[name] != expected {
			return false
		}
	}
	assertion := constraint.Assert
	if assertion.Relation != "" && (assertion.Op == "count" || assertion.Op == "allowed-targets") {
		var domain *core.ModelDomain
		for i := range model.Domains {
			if model.Domains[i].APIVersion == subject.Identity.APIVersion {
				domain = &model.Domains[i]
				break
			}
		}
		if domain == nil {
			return false
		}
		relation, ok := domain.Relations[assertion.Relation]
		if !ok {
			return false
		}
		allowed := false
		for _, kind := range relation.SourceKinds {
			if kind == subject.Identity.Kind {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

func policyResultSemanticallyEqual(left, right core.PolicyResult) bool {
	if left.APIVersion != right.APIVersion || left.Constraint != right.Constraint || left.Subject != right.Subject ||
		left.Status != right.Status || left.Message != right.Message || left.ConstraintDigest != right.ConstraintDigest || left.SubjectDigest != right.SubjectDigest ||
		left.ExceptionName != right.ExceptionName || left.Rationale != right.Rationale || left.Owner != right.Owner || left.Decision != right.Decision || left.ExpiresOn != right.ExpiresOn || left.PolicyDate != right.PolicyDate {
		return false
	}
	return semanticComparisonEqual(left.Comparison, right.Comparison)
}

func semanticComparisonEqual(left, right *core.TargetComparison) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return semanticTargetPathEqual(left.Left, right.Left) && semanticTargetPathEqual(left.Right, right.Right)
}

func semanticTargetPathEqual(left, right core.TargetPath) bool {
	if left.Target != right.Target || len(left.Relations) != len(right.Relations) || len(left.Steps) != len(right.Steps) {
		return false
	}
	for i := range left.Relations {
		if left.Relations[i] != right.Relations[i] {
			return false
		}
	}
	for i := range left.Steps {
		a, b := left.Steps[i], right.Steps[i]
		if a.From != b.From || a.To != b.To || a.Relation != b.Relation || a.DomainAPIVersion != b.DomainAPIVersion {
			return false
		}
	}
	return true
}
