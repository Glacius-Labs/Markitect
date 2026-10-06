package dotnet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

// ArtifactObservation is a Host-supplied artifact within the selected target
// prefix. This package performs no filesystem observation.
type ArtifactObservation struct {
	Path  string
	Bytes []byte
	Mode  string
}

// ArtifactBinding is the path, content digest, and mode from an active record
// or verification result.
type ArtifactBinding struct {
	Path   string
	Digest string
	Mode   string
}

// PriorProjection is the translated active record for this Projection.
type PriorProjection struct {
	RequestDigest    string
	Complete         bool
	Artifacts        []ArtifactBinding
	RetiredArtifacts []string
}

// VerificationBinding is a Host-translated successful verification for one
// exact request and artifact set.
type VerificationBinding struct {
	Passed        bool
	RequestDigest string
	Artifacts     []ArtifactBinding
}

// RepairFinding is a Host-translated semantic failure to address within the
// already selected .NET projection scope.
type RepairFinding struct {
	Subject string
	Detail  string
}

// RepairEvidence is supplied only after the Host validates a current,
// completed semantic failure. Its IDs remain opaque to this Module.
type RepairEvidence struct {
	RecordID string
	ResultID string
	Findings []RepairFinding
}

const (
	maxRepairFindings       = 128
	maxRepairIdentifierSize = 512
	maxRepairSubjectSize    = 512
	maxRepairDetailSize     = 2 * 1024
)

// Decision is the Module's disposition for its bounded target scope.
type Decision string

const (
	DecisionWork     Decision = "work"
	DecisionNoop     Decision = "no-op"
	DecisionEscalate Decision = "escalate"
)

// ProposalInput contains only the selected Core scope, selected Projection
// policies, registered target roots, and Host-translated prior/observed state.
type ProposalInput = Input

// KindContext retains source ontology meaning and its matching target policy
// for an Executor. It is built generically from the supplied Core values.
type KindContext struct {
	Identity      core.KindIdentity
	SchemaPurpose string
	Kind          core.Kind
	Definitions   []core.Definition
	Policies      []core.Definition
}

// ExecutorTask describes a bounded .NET work request. The Executor chooses
// candidate filenames; this Module does not prescribe source paths.
type ExecutorTask struct {
	RequestDigest          string
	Objective              string
	TargetPrefix           string
	AllowedRoots           []string
	AllowedExtensions      []string
	Kinds                  []KindContext
	ExistingOwnedArtifacts []ArtifactObservation
	Constraints            []string
	Repair                 *RepairEvidence
}

// Proposal reports work, no materialization work, or a fail-closed escalation.
// EvidenceRefreshRequired is separate from artifact work.
type Proposal struct {
	Decision                Decision
	Reasons                 []string
	EvidenceRefreshRequired bool
	Task                    *ExecutorTask
	Escalations             []Escalation
}

// Escalation identifies a missing or ambiguous fact that needs an owner decision.
// Propose selects bounded .NET work from source semantics, target policies,
// and inspected ownership state. It never creates candidate files or claims
// semantic verification.
func Propose(input Input) Proposal {
	prefix, roots, err := validateTarget(input.TargetPrefix, input.AllowedRoots)
	if err != nil {
		return escalate("target.scope.invalid", err.Error())
	}
	if input.Previous != nil && !validDigest(input.Previous.RequestDigest) {
		return escalate("record.request.digest.invalid", "active record contains an invalid request digest")
	}
	if !validDigest(input.RequestDigest) {
		return escalate("request.digest.invalid", "a canonical request digest is required")
	}
	if len(input.Schemas) > core.MaxSchemas || len(input.Definitions) == 0 || len(input.Definitions) > core.MaxDefinitions || len(input.Policies) > core.MaxDefinitions {
		return escalate("scope.limit", "selected Core scope exceeds the bounded Module input limits")
	}
	kinds, escalations := buildKindContexts(input)
	if len(escalations) > 0 {
		return Proposal{Decision: DecisionEscalate, Reasons: escalationCodes(escalations), Escalations: escalations}
	}
	observed, escalations := validateInventory(prefix, input)
	if len(escalations) > 0 {
		return Proposal{Decision: DecisionEscalate, Reasons: escalationCodes(escalations), Escalations: escalations}
	}
	prior, escalations := validatePrior(prefix, input.Previous)
	if len(escalations) > 0 {
		return Proposal{Decision: DecisionEscalate, Reasons: escalationCodes(escalations), Escalations: escalations}
	}
	if !input.InventoryComplete {
		return escalate("target.inventory.incomplete", "the target prefix was not completely observed")
	}
	for name := range observed {
		if _, owned := prior[name]; !owned {
			return escalate("artifact.unknown", "target contains an unowned artifact that must be classified before projection")
		}
	}
	if input.Previous == nil {
		return work(input, prefix, roots, kinds, nil, []string{"not-materialized"})
	}
	if len(prior) == 0 {
		return work(input, prefix, roots, kinds, nil, []string{"incomplete-materialization"})
	}
	drift := false
	for name, binding := range prior {
		current, exists := observed[name]
		if !exists || digest(current.Bytes) != binding.Digest || current.Mode != binding.Mode {
			drift = true
		}
	}
	if drift {
		return work(input, prefix, roots, kinds, observedOwned(prior, observed), []string{"projection-drift"})
	}
	if !input.Previous.Complete {
		return work(input, prefix, roots, kinds, observedOwned(prior, observed), []string{"incomplete-materialization"})
	}
	if input.CanonicalAffected {
		return work(input, prefix, roots, kinds, observedOwned(prior, observed), []string{"canonical-or-binding-change"})
	}
	if input.Repair != nil {
		if err := validateRepairEvidence(input.Repair); err != nil {
			return escalate("repair.evidence.invalid", err.Error())
		}
		proposal := work(input, prefix, roots, kinds, observedOwned(prior, observed), []string{"semantic-verification-failed"})
		proposal.Task.Objective = "Repair the selected .NET representation to address the supplied semantic verification findings without changing canonical intent."
		proposal.Task.Constraints = append(proposal.Task.Constraints, "Address only the supplied semantic findings within the existing selected scope and target bounds.")
		repair := *input.Repair
		repair.Findings = append([]RepairFinding(nil), input.Repair.Findings...)
		proposal.Task.Repair = &repair
		return proposal
	}
	refresh := !verificationCurrent(input.RequestDigest, input.Verification, prior, observed)
	proposal := Proposal{Decision: DecisionNoop, EvidenceRefreshRequired: refresh}
	if refresh {
		proposal.Reasons = []string{"evidence-refresh-required"}
	} else {
		proposal.Reasons = []string{"representation-current"}
	}
	return proposal
}

func validateRepairEvidence(evidence *RepairEvidence) error {
	if evidence == nil {
		return fmt.Errorf("semantic repair evidence is missing")
	}
	if !validRepairIdentifier(evidence.RecordID) || !validRepairIdentifier(evidence.ResultID) {
		return fmt.Errorf("semantic repair evidence requires bounded nonempty record and result IDs")
	}
	if len(evidence.Findings) == 0 || len(evidence.Findings) > maxRepairFindings {
		return fmt.Errorf("semantic repair evidence must contain between 1 and %d findings", maxRepairFindings)
	}
	for _, finding := range evidence.Findings {
		if !validRepairText(finding.Subject, maxRepairSubjectSize) {
			return fmt.Errorf("semantic repair finding has an empty, invalid, or oversized subject")
		}
		if !validRepairText(finding.Detail, maxRepairDetailSize) {
			return fmt.Errorf("semantic repair finding has an empty, invalid, or oversized detail")
		}
	}
	return nil
}

func validRepairIdentifier(value string) bool {
	return len(value) > 0 && len(value) <= maxRepairIdentifierSize && utf8.ValidString(value) && strings.TrimSpace(value) != ""
}

func validRepairText(value string, maxBytes int) bool {
	return len(value) > 0 && len(value) <= maxBytes && utf8.ValidString(value) && strings.TrimSpace(value) != ""
}

func buildKindContexts(input ProposalInput) ([]KindContext, []Escalation) {
	schemaKinds := map[string]struct {
		purpose string
		kind    core.Kind
	}{}
	for _, schema := range input.Schemas {
		for name, kind := range schema.Kinds {
			key := (core.KindIdentity{APIVersion: schema.APIVersion, Kind: name}).Key()
			schemaKinds[key] = struct {
				purpose string
				kind    core.Kind
			}{schema.Purpose, kind}
		}
	}
	definitions := append([]core.Definition(nil), input.Definitions...)
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Identity().Key() < definitions[j].Identity().Key() })
	selected := map[string][]core.Definition{}
	for _, definition := range definitions {
		key := (core.KindIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind}).Key()
		selected[key] = append(selected[key], definition)
	}
	policies := map[string][]core.Definition{}
	var escalations []Escalation
	for _, policy := range input.Policies {
		kind, target, guidance, err := policyFields(policy)
		if err != nil {
			escalations = append(escalations, Escalation{Code: "policy.invalid", Message: err.Error()})
			continue
		}
		key := kind.Key()
		if target != "dotnet" {
			escalations = append(escalations, Escalation{Code: "policy.target-mismatch", Message: fmt.Sprintf("policy %s targets %q rather than dotnet", policy.Identity().Key(), target)})
			continue
		}
		if _, ok := schemaKinds[key]; !ok || len(selected[key]) == 0 {
			escalations = append(escalations, Escalation{Code: "policy.kind-outside-scope", Message: fmt.Sprintf("policy %s references a Kind outside the selected Schema scope", policy.Identity().Key())})
			continue
		}
		if strings.TrimSpace(guidance) == "" {
			escalations = append(escalations, Escalation{Code: "policy.guidance-empty", Message: fmt.Sprintf("policy %s has no usable guidance", policy.Identity().Key())})
			continue
		}
		policies[key] = append(policies[key], policy)
	}
	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	contexts := make([]KindContext, 0, len(keys))
	for _, key := range keys {
		contract, exists := schemaKinds[key]
		identity := core.KindIdentity{APIVersion: selected[key][0].APIVersion, Kind: selected[key][0].Kind}
		if !exists {
			escalations = append(escalations, Escalation{Code: "kind.schema-missing", Message: fmt.Sprintf("selected Definition Kind %s has no supplied Schema contract", key)})
			continue
		}
		if len(policies[key]) == 0 {
			escalations = append(escalations, Escalation{Code: "policy.guidance-missing", Message: fmt.Sprintf("Kind %s has no selected dotnet ProjectionPolicy", key)})
			continue
		}
		if len(policies[key]) > 1 {
			escalations = append(escalations, Escalation{Code: "policy.ambiguous", Message: fmt.Sprintf("Kind %s has multiple selected dotnet ProjectionPolicies", key)})
			continue
		}
		contexts = append(contexts, KindContext{Identity: identity, SchemaPurpose: contract.purpose, Kind: contract.kind, Definitions: selected[key], Policies: policies[key]})
	}
	sort.Slice(escalations, func(i, j int) bool {
		if escalations[i].Code != escalations[j].Code {
			return escalations[i].Code < escalations[j].Code
		}
		return escalations[i].Message < escalations[j].Message
	})
	return contexts, escalations
}

func policyFields(policy core.Definition) (core.KindIdentity, string, string, error) {
	if policy.Kind != "ProjectionPolicy" {
		return core.KindIdentity{}, "", "", fmt.Errorf("selected resource %s is not a ProjectionPolicy", policy.Identity().Key())
	}
	source, ok := policy.Spec["sourceKind"].(map[string]any)
	if !ok {
		return core.KindIdentity{}, "", "", fmt.Errorf("ProjectionPolicy %s has no sourceKind object", policy.Identity().Key())
	}
	api, apiOK := source["apiVersion"].(string)
	kind, kindOK := source["kind"].(string)
	target, targetOK := policy.Spec["targetTechnology"].(string)
	guidance, guidanceOK := policy.Spec["guidance"].(string)
	if !apiOK || !kindOK || !targetOK || !guidanceOK || api == "" || kind == "" || strings.TrimSpace(target) != target || target == "" {
		return core.KindIdentity{}, "", "", fmt.Errorf("ProjectionPolicy %s has malformed sourceKind, targetTechnology, or guidance", policy.Identity().Key())
	}
	return core.KindIdentity{APIVersion: api, Kind: kind}, target, guidance, nil
}

func validateTarget(prefix string, roots []string) (string, []string, error) {
	clean, err := cleanRelative(prefix)
	if err != nil {
		return "", nil, err
	}
	if len(roots) == 0 {
		return "", nil, fmt.Errorf("no registered target roots were supplied")
	}
	allowed := make([]string, 0, len(roots))
	for _, root := range roots {
		value, rootErr := cleanRelative(root)
		if rootErr != nil {
			return "", nil, fmt.Errorf("invalid registered target root")
		}
		if within(value, clean) {
			allowed = append(allowed, value)
		}
	}
	if len(allowed) == 0 {
		return "", nil, fmt.Errorf("Projection target is outside the capability's registered roots")
	}
	sort.Strings(allowed)
	return clean, allowed, nil
}

func cleanRelative(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.Contains(value, "\\") || path.IsAbs(value) {
		return "", fmt.Errorf("target path must be a clean relative slash path")
	}
	trimmed := strings.TrimSuffix(value, "/")
	clean := path.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != trimmed {
		return "", fmt.Errorf("target path must be a clean relative slash path")
	}
	return clean, nil
}
func within(root, name string) bool { return name == root || strings.HasPrefix(name, root+"/") }
func validDigest(value string) bool {
	value = strings.TrimPrefix(value, "sha256:")
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && strings.ToLower(value) == value
}
func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validateInventory(prefix string, input ProposalInput) (map[string]ArtifactObservation, []Escalation) {
	result := map[string]ArtifactObservation{}
	if len(input.ObservedArtifacts) > core.MaxDefinitions {
		return nil, []Escalation{{Code: "target.inventory.limit", Message: "target inventory exceeds the bounded artifact count"}}
	}
	var escalations []Escalation
	for _, artifact := range input.ObservedArtifacts {
		clean, err := cleanRelative(artifact.Path)
		if err != nil || clean != artifact.Path || !within(prefix, clean) || artifact.Mode == "" {
			escalations = append(escalations, Escalation{Code: "target.inventory.invalid", Message: "target inventory contains an invalid path or missing mode"})
			continue
		}
		if _, exists := result[clean]; exists {
			escalations = append(escalations, Escalation{Code: "target.inventory.duplicate", Message: "target inventory contains a duplicate path"})
			continue
		}
		result[clean] = artifact
	}
	return result, sortEscalations(escalations)
}

func validatePrior(prefix string, prior *PriorProjection) (map[string]ArtifactBinding, []Escalation) {
	result := map[string]ArtifactBinding{}
	if prior == nil {
		return result, nil
	}
	if len(prior.RetiredArtifacts) > 0 {
		return nil, []Escalation{{Code: "artifact.retired", Message: "active ownership includes retired artifacts; review is required and this Module never deletes files"}}
	}
	if len(prior.Artifacts) > core.MaxDefinitions {
		return nil, []Escalation{{Code: "record.artifact.limit", Message: "active record exceeds the bounded artifact count"}}
	}
	var escalations []Escalation
	for _, artifact := range prior.Artifacts {
		clean, err := cleanRelative(artifact.Path)
		if err != nil || clean != artifact.Path || !within(prefix, clean) || !validDigest(artifact.Digest) || artifact.Mode == "" {
			escalations = append(escalations, Escalation{Code: "record.artifact.invalid", Message: "active record contains an invalid artifact binding"})
			continue
		}
		if _, exists := result[clean]; exists {
			escalations = append(escalations, Escalation{Code: "record.artifact.duplicate", Message: "active record contains a duplicate artifact path"})
			continue
		}
		result[clean] = artifact
	}
	return result, sortEscalations(escalations)
}

func observedOwned(prior map[string]ArtifactBinding, observed map[string]ArtifactObservation) []ArtifactObservation {
	names := make([]string, 0, len(prior))
	for name := range prior {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]ArtifactObservation, 0, len(names))
	for _, name := range names {
		if value, ok := observed[name]; ok {
			result = append(result, value)
		}
	}
	return result
}

func verificationCurrent(request string, verification *VerificationBinding, prior map[string]ArtifactBinding, observed map[string]ArtifactObservation) bool {
	if verification == nil || !verification.Passed || verification.RequestDigest != request || len(verification.Artifacts) != len(prior) {
		return false
	}
	verified := map[string]ArtifactBinding{}
	for _, artifact := range verification.Artifacts {
		if _, exists := verified[artifact.Path]; exists {
			return false
		}
		verified[artifact.Path] = artifact
	}
	if len(verified) != len(prior) {
		return false
	}
	for name, binding := range prior {
		current, exists := observed[name]
		proof, proved := verified[name]
		if !exists || !proved || digest(current.Bytes) != binding.Digest || current.Mode != binding.Mode || proof.Digest != binding.Digest || proof.Mode != binding.Mode {
			return false
		}
	}
	return true
}

func work(input ProposalInput, prefix string, roots []string, kinds []KindContext, owned []ArtifactObservation, reasons []string) Proposal {
	sort.Strings(reasons)
	return Proposal{Decision: DecisionWork, Reasons: reasons, Task: &ExecutorTask{
		RequestDigest:          input.RequestDigest,
		Objective:              "Project the selected canonical scope into the declared .NET target using its supplied semantics and matching ProjectionPolicy guidance.",
		TargetPrefix:           prefix,
		AllowedRoots:           roots,
		AllowedExtensions:      []string{".cs", ".csproj"},
		Kinds:                  kinds,
		ExistingOwnedArtifacts: owned,
		Constraints: []string{
			"Choose candidate filenames from the supplied semantics and project conventions.",
			"Return candidate files only under TargetPrefix and registered AllowedRoots.",
			"Return the complete owned representation: include a candidate file for every existing owned artifact path listed in the task, even when its contents are unchanged. Omission is not deletion or retirement.",
			"Do not delete or claim verification of any artifact.",
		},
	}}
}
func escalate(code, message string) Proposal {
	return Proposal{Decision: DecisionEscalate, Reasons: []string{code}, Escalations: []Escalation{{Code: code, Message: message}}}
}
func escalationCodes(values []Escalation) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Code)
	}
	sort.Strings(result)
	return uniqueStrings(result)
}
func sortEscalations(values []Escalation) []Escalation {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		return values[i].Message < values[j].Message
	})
	return values
}
func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
