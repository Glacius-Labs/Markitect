package projectexplore

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type structureShape struct {
	ScopeID               string   `json:"scopeId"`
	ScopeName             string   `json:"scopeName"`
	Goal                  string   `json:"goal"`
	Operation             string   `json:"operation"`
	ManagerIDs            []string `json:"managerIds"`
	ResponsibleManagerIDs []string `json:"responsibleManagerIds"`
	RequiredArtifacts     []string `json:"requiredArtifacts"`
	Files                 []string `json:"files"`
	Checks                []string `json:"checks"`
}

// BindingDigest returns a stable digest of the complete current basis and
// named scope. No random run or plan ID participates in this identity.
func BindingDigest(binding Binding) (string, error) {
	binding, err := canonicalBinding(binding)
	if err != nil {
		return "", err
	}
	return digest(binding)
}

// StructureDigest binds the exact named-scope file structure and its declared
// Managers, artifacts, operation, and checks, independent of run identifiers.
func StructureDigest(binding Binding) (string, error) {
	binding, err := canonicalBinding(binding)
	if err != nil {
		return "", err
	}
	return digest(structureShape{
		ScopeID: binding.ScopeID, ScopeName: binding.ScopeName, Goal: binding.Goal, Operation: binding.Operation,
		ManagerIDs: binding.ManagerIDs, ResponsibleManagerIDs: binding.ResponsibleManagerIDs, RequiredArtifacts: binding.RequiredArtifacts, Files: binding.FileStructure, Checks: binding.Checks,
	})
}

// EvaluateReadiness computes readiness against the exact caller-supplied fixed
// binding. A stale or changed binding does not reuse a previous acknowledgement.
func EvaluateReadiness(record Record, scopeID string, binding Binding) (ReadinessReport, error) {
	var report ReadinessReport
	if err := validateRecord(record); err != nil {
		return report, err
	}
	binding, err := canonicalBinding(binding)
	if err != nil {
		return report, err
	}
	if binding.ScopeID != scopeID {
		return report, errors.New("readiness scope does not match its fixed binding")
	}
	modelAccepted := binding.ModelAccepted && binding.ModelRevision != "" && binding.ModelRevision == binding.Head && fullObjectID(binding.Head) && strings.TrimSpace(binding.AcceptancePolicy) != ""
	scope, ok := findScope(record, scopeID)
	if !ok {
		return report, fmt.Errorf("exploration %s has no scope %q", record.ID, scopeID)
	}
	if scope.Name != binding.ScopeName || scope.Goal != binding.Goal || scope.Operation != binding.Operation || !equalStrings(scope.ManagerIDs, binding.ManagerIDs) {
		return report, errors.New("readiness binding differs from the recorded named scope")
	}
	bindingDigest, err := BindingDigest(binding)
	if err != nil {
		return report, err
	}
	structureDigest, err := StructureDigest(binding)
	if err != nil {
		return report, err
	}
	report = ReadinessReport{
		APIVersion: APIVersion, ExplorationID: record.ID, ScopeID: scopeID,
		Blockers: []string{}, OpenDecisions: []string{}, BindingDigest: bindingDigest, StructureDigest: structureDigest,
	}
	if record.Status != StatusActive {
		report.Blockers = append(report.Blockers, "exploration is already completed")
	}
	if !modelAccepted {
		report.Blockers = append(report.Blockers, "readiness requires the accepted canonical model at a fixed committed revision under the declared acceptance policy")
	}
	if modelAccepted && len(binding.ResponsibleManagerIDs) == 0 {
		report.Blockers = append(report.Blockers, "readiness requires the computed responsible Manager set for this plan")
	}
	completion, complete := completionFor(record, scopeID)
	if complete {
		if completion.BindingDigest == bindingDigest && completion.StructureDigest == structureDigest {
			report.Blockers = append(report.Blockers, "scope has already been successfully applied")
		} else {
			report.Blockers = append(report.Blockers, "scope was completed against a different binding")
		}
	}
	for _, decision := range record.Decisions {
		if !contains(decision.ScopeIDs, scopeID) {
			continue
		}
		if decision.Status != "answered" {
			report.OpenDecisions = append(report.OpenDecisions, decision.ID)
		}
		if decision.Blocking && decision.Status != "answered" {
			report.Blockers = append(report.Blockers, fmt.Sprintf("blocking decision %q is %s", decision.ID, decision.Status))
		}
	}
	acknowledged := false
	for _, ack := range record.Acknowledgements {
		if ack.ScopeID == scopeID && ack.BindingDigest == bindingDigest && ack.StructureDigest == structureDigest {
			acknowledged = true
			break
		}
	}
	if !acknowledged {
		report.Blockers = append(report.Blockers, "the exact proposed structure has not been acknowledged for this binding")
	}
	sort.Strings(report.OpenDecisions)
	sort.Strings(report.Blockers)
	report.Ready = len(report.Blockers) == 0
	report.Digest, err = readinessDigest(report)
	if err != nil {
		return ReadinessReport{}, err
	}
	return report, nil
}

// AcknowledgeStructure appends a caller-declared review assertion only when it
// binds the exact current repository and structure. It does not authenticate
// the actor or alter canonical model files.
func AcknowledgeStructure(record *Record, scopeID string, binding Binding, ack StructureAcknowledgement) error {
	if record == nil {
		return errors.New("exploration record is required")
	}
	bindingDigest, err := BindingDigest(binding)
	if err != nil {
		return err
	}
	structureDigest, err := StructureDigest(binding)
	if err != nil {
		return err
	}
	if ack.ScopeID != scopeID || ack.BindingDigest != bindingDigest || ack.StructureDigest != structureDigest || strings.TrimSpace(ack.Actor) == "" || strings.TrimSpace(ack.Authority) == "" || strings.TrimSpace(ack.Provenance) == "" || ack.RecordedAt.IsZero() {
		return errors.New("structure acknowledgement must name actor, authority, provenance, and exact current binding and structure digests")
	}
	if _, ok := findScope(*record, scopeID); !ok {
		return fmt.Errorf("exploration %s has no scope %q", record.ID, scopeID)
	}
	for _, prior := range record.Acknowledgements {
		if prior.ScopeID == scopeID && prior.BindingDigest == bindingDigest && prior.StructureDigest == structureDigest {
			if prior.Actor == ack.Actor && prior.Authority == ack.Authority && prior.Provenance == ack.Provenance {
				return nil
			}
			return errors.New("a conflicting acknowledgement already exists for this exact binding")
		}
	}
	record.Acknowledgements = append(record.Acknowledgements, ack)
	sort.Slice(record.Acknowledgements, func(i, j int) bool {
		left, right := record.Acknowledgements[i], record.Acknowledgements[j]
		if left.ScopeID != right.ScopeID {
			return left.ScopeID < right.ScopeID
		}
		if left.BindingDigest != right.BindingDigest {
			return left.BindingDigest < right.BindingDigest
		}
		return left.StructureDigest < right.StructureDigest
	})
	return reseal(record)
}

// completeScope records only a successful verified Apply receipt for a ready
// scope. Repeating the same receipt is idempotent; a different receipt cannot
// rewrite completion history. The package-private entry point prevents generic
// callers from creating a completion-capable WritePlan.
func completeScope(record *Record, scopeID string, binding Binding, receipt ApplyReceipt) error {
	if record == nil {
		return errors.New("exploration record is required")
	}
	if err := validateApplyReceipt(receipt); err != nil {
		return err
	}
	if prior, ok := completionFor(*record, scopeID); ok {
		if prior == receipt {
			return nil
		}
		return errors.New("scope already has a different successful Apply receipt")
	}
	if _, err := canonicalBinding(binding); err != nil {
		return err
	}
	if receipt.ScopeID != scopeID || binding.ScopeID != scopeID {
		return errors.New("Apply receipt does not name the current scope")
	}
	if _, ok := findScope(*record, scopeID); !ok {
		return fmt.Errorf("exploration %s has no scope %q", record.ID, scopeID)
	}
	readiness, err := EvaluateReadiness(*record, scopeID, binding)
	if err != nil {
		return err
	}
	if !readiness.Ready || receipt.BindingDigest != readiness.BindingDigest || receipt.StructureDigest != readiness.StructureDigest {
		return errors.New("Apply receipt does not bind a currently ready scope and its exact acknowledged structure")
	}
	acknowledged := false
	for _, ack := range record.Acknowledgements {
		if ack.ScopeID == scopeID && ack.BindingDigest == receipt.BindingDigest && ack.StructureDigest == receipt.StructureDigest {
			acknowledged = true
			break
		}
	}
	if !acknowledged {
		return errors.New("Apply receipt does not match a recorded exact-binding structure acknowledgement")
	}
	record.Completions = append(record.Completions, receipt)
	sort.Slice(record.Completions, func(i, j int) bool { return record.Completions[i].ScopeID < record.Completions[j].ScopeID })
	if len(record.Completions) == len(record.Scopes) {
		record.Status = StatusCompleted
	}
	return reseal(record)
}

func readinessDigest(report ReadinessReport) (string, error) {
	report.Digest = ""
	return digest(report)
}

func reseal(record *Record) error {
	if err := validateRecord(*record); err != nil {
		return err
	}
	digest, err := recordDigest(*record)
	if err != nil {
		return err
	}
	record.Digest = digest
	return nil
}

func findScope(record Record, id string) (Scope, bool) {
	for _, scope := range record.Scopes {
		if scope.ID == id {
			return scope, true
		}
	}
	return Scope{}, false
}

func completionFor(record Record, id string) (ApplyReceipt, bool) {
	for _, completion := range record.Completions {
		if completion.ScopeID == id {
			return completion, true
		}
	}
	return ApplyReceipt{}, false
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
