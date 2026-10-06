package records

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	ProjectionRecordAPIVersion   = "markitect.example.org/projection-record/v1alpha1"
	VerificationResultAPIVersion = "markitect.example.org/verification-result/v1alpha1"
	OwnershipIndexAPIVersion     = "markitect.example.org/artifact-ownership-index/v1alpha1"

	OriginAdopted               = "adopted"
	StateMaterializedUnverified = "materialized-unverified"
	StatePartialFailure         = "partial-failure"
	StateEscalated              = "escalated"
	ChangeCreated               = "created"
	ChangeModified              = "modified"
	ChangeRetained              = "retained"
	OutcomePassed               = "passed"
	OutcomeFailed               = "failed"
	OutcomeIncomplete           = "incomplete"
	OutcomeEscalated            = "escalated"
	CheckPassed                 = "passed"
	CheckFailed                 = "failed"
	CheckIncomplete             = "incomplete"

	RoleCanonicalSource  = "canonical-source"
	RoleExternalInput    = "external-input"
	RoleProjectionTarget = "projection-target"
	RoleToolOwned        = "tool-owned"
	RoleVendorOwned      = "vendor-owned"
	RoleIgnored          = "ignored"
	RoleExcluded         = "excluded"
	RoleUnknown          = "unknown"

	OwnershipManaged         = "managed"
	OwnershipDrift           = "drift"
	OwnershipUnobserved      = "unobserved"
	OwnershipUnknown         = "unknown"
	OwnershipExcluded        = "excluded"
	OwnershipIgnored         = "ignored"
	OwnershipCanonicalSource = "canonical-source"
	OwnershipExternalInput   = "external-input"
	OwnershipToolOwned       = "tool-owned"
	OwnershipVendorOwned     = "vendor-owned"
)

var (
	digestPattern   = regexp.MustCompile("^sha256:[0-9a-f]{64}$")
	revisionPattern = regexp.MustCompile("^(?:[0-9a-f]{40}|[0-9a-f]{64})$")
)

type ModuleIdentity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}
type ProjectorIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type Artifact struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Mode   string `json:"mode"`
	Change string `json:"change"`
}
type ProjectionRecord struct {
	Origin               string            `json:"origin,omitempty"`
	AdoptionRevision     string            `json:"adoptionRevision,omitempty"`
	ReviewReference      string            `json:"reviewReference,omitempty"`
	APIVersion           string            `json:"apiVersion"`
	ID                   string            `json:"id"`
	Revision             string            `json:"revision"`
	ModelDigest          string            `json:"modelDigest"`
	PlanDigest           string            `json:"planDigest"`
	InputSnapshotDigest  string            `json:"inputSnapshotDigest"`
	RequestDigest        string            `json:"requestDigest"`
	Module               ModuleIdentity    `json:"module"`
	ProjectionID         string            `json:"projectionId"`
	Projector            ProjectorIdentity `json:"projector"`
	ScopeIDs             []string          `json:"scopeIds"`
	PolicyIDs            []string          `json:"policyIds"`
	Artifacts            []Artifact        `json:"artifacts"`
	TargetSnapshotDigest string            `json:"targetSnapshotDigest"`
	PriorRecordID        string            `json:"priorRecordId,omitempty"`
	State                string            `json:"state"`
}
type recordEnvelope struct {
	Record        ProjectionRecord `json:"record"`
	ContentDigest string           `json:"contentDigest"`
}
type VerifierIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}
type CheckIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}
type CheckResult struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
	Outcome string `json:"outcome"`
}
type VerificationResult struct {
	APIVersion                    string           `json:"apiVersion"`
	ID                            string           `json:"id"`
	RecordID                      string           `json:"recordId"`
	Revision                      string           `json:"revision"`
	ModelDigest                   string           `json:"modelDigest"`
	TargetSnapshotDigest          string           `json:"targetSnapshotDigest"`
	EvidenceRevision              string           `json:"evidenceRevision,omitempty"`
	EvidenceSnapshotDigest        string           `json:"evidenceSnapshotDigest,omitempty"`
	ControllerConfigDigest        string           `json:"controllerConfigDigest,omitempty"`
	ControllerVerifierInputDigest string           `json:"controllerVerifierInputDigest,omitempty"`
	Verifier                      VerifierIdentity `json:"verifier"`
	Checks                        []CheckResult    `json:"checks"`
	Outcome                       string           `json:"outcome"`
	Reason                        string           `json:"reason,omitempty"`
}
type verificationEnvelope struct {
	Result        VerificationResult `json:"result"`
	ContentDigest string             `json:"contentDigest"`
}

// Freshness is supplied from the exact current model, targets, and declared checks.
type Freshness struct {
	Revision             string
	ModelDigest          string `json:"modelDigest"`
	RecordID             string
	TargetSnapshotDigest string
	Verifier             VerifierIdentity
	Checks               []CheckIdentity
}
type ProjectionFreshness struct {
	Revision            string
	ModelDigest         string
	PlanDigest          string
	InputSnapshotDigest string
	RequestDigest       string
	Targets             []ArtifactFact
}
type ArtifactFact struct {
	Path   string `json:"path"`
	Role   string `json:"role"`
	Digest string `json:"digest"`
	Mode   string `json:"mode"`
	Reason string `json:"reason,omitempty"`
}
type OwnershipEntry struct {
	Path           string   `json:"path"`
	Role           string   `json:"role"`
	Status         string   `json:"status"`
	Reason         string   `json:"reason,omitempty"`
	ObservedDigest string   `json:"observedDigest,omitempty"`
	ObservedMode   string   `json:"observedMode,omitempty"`
	ExpectedDigest string   `json:"expectedDigest,omitempty"`
	ExpectedMode   string   `json:"expectedMode,omitempty"`
	OwnerRecordIDs []string `json:"ownerRecordIds"`
	ScopeIDs       []string `json:"scopeIds"`
}
type OwnershipIndex struct {
	APIVersion string                    `json:"apiVersion"`
	ByScope    map[string][]string       `json:"byScope"`
	Artifacts  map[string]OwnershipEntry `json:"artifacts"`
}

// NewProjectionRecord validates and canonicalizes owner-supplied facts, then
// computes a content ID. Duplicate scope, policy, or artifact identities fail.
func NewProjectionRecord(input ProjectionRecord) (ProjectionRecord, error) {
	input.APIVersion, input.ID = ProjectionRecordAPIVersion, ""
	if err := validateRecordFields(input); err != nil {
		return ProjectionRecord{}, err
	}
	input.ScopeIDs, _ = sortedUnique(input.ScopeIDs)
	input.PolicyIDs, _ = sortedUnique(input.PolicyIDs)
	input.Artifacts = append([]Artifact(nil), input.Artifacts...)
	sort.Slice(input.Artifacts, func(i, j int) bool { return input.Artifacts[i].Path < input.Artifacts[j].Path })
	targetDigest, err := targetSnapshotDigestFromArtifacts(input.Artifacts)
	if err != nil {
		return ProjectionRecord{}, err
	}
	input.TargetSnapshotDigest = targetDigest
	input.ID, err = projectionRecordDigest(input)
	if err != nil {
		return ProjectionRecord{}, err
	}
	return input, nil
}

// ValidateProjectionRecord checks schema, canonical ordering, facts, and content ID.
func ValidateProjectionRecord(record ProjectionRecord) error {
	if err := validateRecordFields(record); err != nil {
		return err
	}
	if !isDigest(record.ID) {
		return errors.New("projection record ID must be a sha256 content digest")
	}
	if !isDigest(record.TargetSnapshotDigest) {
		return errors.New("target snapshot digest must be sha256")
	}
	if !isSortedUnique(record.ScopeIDs) || !isSortedUnique(record.PolicyIDs) {
		return errors.New("scope and policy IDs must be sorted and unique")
	}
	if !sort.SliceIsSorted(record.Artifacts, func(i, j int) bool { return record.Artifacts[i].Path < record.Artifacts[j].Path }) {
		return errors.New("artifacts must be sorted by path")
	}
	targetDigest, err := targetSnapshotDigestFromArtifacts(record.Artifacts)
	if err != nil {
		return err
	}
	if record.TargetSnapshotDigest != targetDigest {
		return errors.New("target snapshot digest does not match artifact facts")
	}
	expected, err := projectionRecordDigest(record)
	if err != nil {
		return err
	}
	if record.ID != expected {
		return errors.New("projection record content ID mismatch")
	}
	return nil
}

// AppendPayload returns deterministic JSON for caller-owned append persistence.
func AppendPayload(record ProjectionRecord) ([]byte, error) {
	if err := ValidateProjectionRecord(record); err != nil {
		return nil, err
	}
	return json.Marshal(recordEnvelope{Record: record, ContentDigest: record.ID})
}

// NewVerificationResult canonicalizes checks and computes a deterministic content ID.
func NewVerificationResult(input VerificationResult) (VerificationResult, error) {
	input.APIVersion, input.ID = VerificationResultAPIVersion, ""
	if err := validateVerificationFields(input); err != nil {
		return VerificationResult{}, err
	}
	input.Checks = append([]CheckResult(nil), input.Checks...)
	sort.Slice(input.Checks, func(i, j int) bool { return input.Checks[i].ID < input.Checks[j].ID })
	var err error
	input.ID, err = verificationDigest(input)
	return input, err
}
func ValidateVerificationResult(result VerificationResult) error {
	if err := validateVerificationFields(result); err != nil {
		return err
	}
	if !isDigest(result.ID) {
		return errors.New("verification result ID must be a sha256 content digest")
	}
	if !sort.SliceIsSorted(result.Checks, func(i, j int) bool { return result.Checks[i].ID < result.Checks[j].ID }) {
		return errors.New("checks must be sorted by ID")
	}
	for i := 1; i < len(result.Checks); i++ {
		if result.Checks[i-1].ID == result.Checks[i].ID {
			return fmt.Errorf("duplicate check %q", result.Checks[i].ID)
		}
	}
	expected, err := verificationDigest(result)
	if err != nil {
		return err
	}
	if result.ID != expected {
		return errors.New("verification result content ID mismatch")
	}
	return nil
}
func VerificationAppendPayload(result VerificationResult) ([]byte, error) {
	if err := ValidateVerificationResult(result); err != nil {
		return nil, err
	}
	return json.Marshal(verificationEnvelope{Result: result, ContentDigest: result.ID})
}

// ValidateProjectionFreshness compares a record with the exact revision,
// canonical model, reviewed plan, captured input, request, and target facts.
func ValidateProjectionFreshness(record ProjectionRecord, current ProjectionFreshness) error {
	if err := ValidateProjectionRecord(record); err != nil {
		return err
	}
	if current.Revision != record.Revision {
		return errors.New("projection record is stale for the current revision")
	}
	if current.ModelDigest != record.ModelDigest {
		return errors.New("projection record is stale for the current model")
	}
	if current.PlanDigest != record.PlanDigest {
		return errors.New("projection record is bound to a different reviewed plan")
	}
	if current.InputSnapshotDigest != record.InputSnapshotDigest {
		return errors.New("projection record is bound to a different captured input snapshot")
	}
	if current.RequestDigest != record.RequestDigest {
		return errors.New("projection record is bound to a different projection request")
	}
	targetDigest, err := TargetSnapshotDigest(current.Targets)
	if err != nil {
		return err
	}
	if targetDigest != record.TargetSnapshotDigest {
		return errors.New("projection record is stale for the current target snapshot")
	}
	return nil
}

// ValidateVerificationFreshness binds results to a record and exactly the declared verifier/check set.
func ValidateVerificationFreshness(result VerificationResult, record ProjectionRecord, current Freshness) error {
	if err := ValidateProjectionRecord(record); err != nil {
		return fmt.Errorf("invalid projection record: %w", err)
	}
	if err := ValidateVerificationResult(result); err != nil {
		return err
	}
	if result.RecordID != record.ID || current.RecordID != record.ID {
		return errors.New("verification references a different projection record")
	}
	if result.Revision != record.Revision || current.Revision != record.Revision {
		return errors.New("verification is stale for the current revision")
	}
	if result.ModelDigest != record.ModelDigest || current.ModelDigest != record.ModelDigest {
		return errors.New("verification is stale for the current model")
	}
	if result.TargetSnapshotDigest != record.TargetSnapshotDigest || current.TargetSnapshotDigest != record.TargetSnapshotDigest {
		return errors.New("verification is stale for the current target snapshot")
	}
	if !sameVerifier(result.Verifier, current.Verifier) {
		return errors.New("verification verifier differs from its declaration")
	}
	expected := append([]CheckIdentity(nil), current.Checks...)
	sort.Slice(expected, func(i, j int) bool { return expected[i].ID < expected[j].ID })
	if err := validateCheckIdentities(expected); err != nil {
		return err
	}
	if len(result.Checks) != len(expected) {
		return errors.New("verification must contain exactly the declared checks")
	}
	for i, got := range result.Checks {
		want := expected[i]
		if got.ID != want.ID || got.Version != want.Version || got.Digest != want.Digest {
			return fmt.Errorf("check %q differs from its declared identity", got.ID)
		}
	}
	if result.Outcome == OutcomePassed {
		if result.Verifier.Digest == "" || len(result.Checks) == 0 {
			return errors.New("passed verification requires a verifier digest and checks")
		}
		for _, check := range result.Checks {
			if check.Digest == "" || check.Outcome != CheckPassed {
				return errors.New("passed verification cannot contain failed or incomplete checks")
			}
		}
	}
	return nil
}

// TargetSnapshotDigest hashes exact selected paths, byte digests, and modes.
func TargetSnapshotDigest(targets []ArtifactFact) (string, error) {
	items := make([]Artifact, 0, len(targets))
	seen := map[string]bool{}
	for _, fact := range targets {
		if err := validatePath(fact.Path); err != nil {
			return "", err
		}
		if seen[fact.Path] {
			return "", fmt.Errorf("duplicate target path %q", fact.Path)
		}
		seen[fact.Path] = true
		if !isDigest(fact.Digest) {
			return "", fmt.Errorf("target %q digest must be sha256", fact.Path)
		}
		if err := validateMode(fact.Mode); err != nil {
			return "", fmt.Errorf("target %q: %w", fact.Path, err)
		}
		items = append(items, Artifact{Path: fact.Path, Digest: fact.Digest, Mode: fact.Mode})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return digestJSON(items)
}

// ValidateRecordReferences rejects duplicate ledger IDs and dangling prior-record links.
// The caller supplies the append ledger; ordering and persistence remain caller-owned.
func ValidateRecordReferences(records []ProjectionRecord) error {
	byID := make(map[string]ProjectionRecord, len(records))
	for _, record := range records {
		if err := ValidateProjectionRecord(record); err != nil {
			return fmt.Errorf("invalid projection record: %w", err)
		}
		if _, exists := byID[record.ID]; exists {
			return fmt.Errorf("duplicate projection record %q", record.ID)
		}
		byID[record.ID] = record
	}
	for _, record := range records {
		if record.PriorRecordID == "" {
			continue
		}
		if record.PriorRecordID == record.ID {
			return fmt.Errorf("projection record %q refers to itself", record.ID)
		}
		prior, exists := byID[record.PriorRecordID]
		if !exists {
			return fmt.Errorf("projection record %q has dangling prior record %q", record.ID, record.PriorRecordID)
		}
		if prior.ProjectionID != record.ProjectionID {
			return fmt.Errorf("projection record %q refers to prior record for a different Projection", record.ID)
		}
	}
	return nil
}

// BuildOwnershipIndex indexes caller-selected active records against explicit
// inventory facts; it makes no claim that the inventory is complete or ordered.
func BuildOwnershipIndex(activeRecords []ProjectionRecord, facts []ArtifactFact) (OwnershipIndex, error) {
	index := OwnershipIndex{APIVersion: OwnershipIndexAPIVersion, ByScope: map[string][]string{}, Artifacts: map[string]OwnershipEntry{}}
	owners := map[string]map[string]Artifact{}
	recordByID := map[string]ProjectionRecord{}
	for _, record := range activeRecords {
		if err := ValidateProjectionRecord(record); err != nil {
			return OwnershipIndex{}, fmt.Errorf("invalid active record: %w", err)
		}
		if _, ok := recordByID[record.ID]; ok {
			return OwnershipIndex{}, fmt.Errorf("duplicate active record %q", record.ID)
		}
		recordByID[record.ID] = record
		for _, scope := range record.ScopeIDs {
			index.ByScope[scope] = append(index.ByScope[scope], artifactPaths(record.Artifacts)...)
		}
		for _, artifact := range record.Artifacts {
			if owners[artifact.Path] == nil {
				owners[artifact.Path] = map[string]Artifact{}
			}
			owners[artifact.Path][record.ID] = artifact
			if len(owners[artifact.Path]) > 1 {
				return OwnershipIndex{}, fmt.Errorf("artifact %q has multiple active owners", artifact.Path)
			}
		}
	}
	for scope, names := range index.ByScope {
		sort.Strings(names)
		index.ByScope[scope] = names
	}
	observed := map[string]ArtifactFact{}
	for _, fact := range facts {
		if err := validateArtifactFact(fact); err != nil {
			return OwnershipIndex{}, err
		}
		if _, ok := observed[fact.Path]; ok {
			return OwnershipIndex{}, fmt.Errorf("duplicate inventory fact %q", fact.Path)
		}
		observed[fact.Path] = fact
	}
	all := map[string]bool{}
	for name := range owners {
		all[name] = true
	}
	for name := range observed {
		all[name] = true
	}
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	aliases := map[string]string{}
	for _, name := range names {
		key := strings.ToLower(name)
		if previous, exists := aliases[key]; exists && previous != name {
			return OwnershipIndex{}, fmt.Errorf("artifact paths %q and %q collide under portable case matching", previous, name)
		}
		aliases[key] = name
	}
	for _, name := range names {
		fact, hasFact := observed[name]
		entry := OwnershipEntry{Path: name, Role: RoleUnknown, Status: OwnershipUnobserved, OwnerRecordIDs: []string{}, ScopeIDs: []string{}}
		if hasFact {
			entry.Role, entry.Reason = fact.Role, fact.Reason
			entry.ObservedDigest, entry.ObservedMode = fact.Digest, fact.Mode
			entry.Status = statusForUnownedRole(fact.Role)
		}
		if byRecord := owners[name]; len(byRecord) != 0 {
			if hasFact && fact.Role != RoleProjectionTarget {
				return OwnershipIndex{}, fmt.Errorf("projection-owned artifact %q has conflicting role %q", name, fact.Role)
			}
			for id, artifact := range byRecord {
				entry.OwnerRecordIDs = append(entry.OwnerRecordIDs, id)
				entry.ExpectedDigest, entry.ExpectedMode = artifact.Digest, artifact.Mode
				entry.ScopeIDs = append(entry.ScopeIDs, recordByID[id].ScopeIDs...)
			}
			sort.Strings(entry.OwnerRecordIDs)
			entry.ScopeIDs, _ = sortedUnique(entry.ScopeIDs)
			switch {
			case !hasFact:
				entry.Status = OwnershipUnobserved
			case entry.ExpectedDigest == fact.Digest && entry.ExpectedMode == fact.Mode:
				entry.Status = OwnershipManaged
			default:
				entry.Status = OwnershipDrift
			}
		} else if hasFact && fact.Role == RoleProjectionTarget {
			entry.Status = OwnershipUnknown
		}
		index.Artifacts[name] = entry
	}
	return index, nil
}

func validateRecordFields(record ProjectionRecord) error {
	if record.Origin == OriginAdopted {
		if !revisionPattern.MatchString(record.AdoptionRevision) || strings.TrimSpace(record.ReviewReference) == "" || len(record.ReviewReference) > 4096 || !utf8.ValidString(record.ReviewReference) || strings.ContainsRune(record.ReviewReference, 0) {
			return errors.New("adopted record requires a full evidence revision and bounded owner-supplied review reference")
		}
		if record.State != StateMaterializedUnverified {
			return errors.New("adopted record must describe a complete retained representation")
		}
		for _, artifact := range record.Artifacts {
			if artifact.Change != ChangeRetained {
				return errors.New("adopted record can claim only retained artifacts")
			}
		}
	} else if record.Origin != "" || record.AdoptionRevision != "" || record.ReviewReference != "" {
		return errors.New("unsupported record origin or adoption metadata without adopted origin")
	}
	if record.APIVersion != ProjectionRecordAPIVersion {
		return fmt.Errorf("unsupported projection record apiVersion %q", record.APIVersion)
	}
	if !revisionPattern.MatchString(record.Revision) {
		return errors.New("revision must be a full lowercase 40- or 64-character immutable ID")
	}
	if !isDigest(record.ModelDigest) || !isDigest(record.PlanDigest) || !isDigest(record.InputSnapshotDigest) || !isDigest(record.RequestDigest) {
		return errors.New("model, plan, input snapshot, and request digests must be sha256")
	}
	if err := validateText("module name", record.Module.Name); err != nil {
		return err
	}
	if err := validateText("module version", record.Module.Version); err != nil {
		return err
	}
	if !isDigest(record.Module.Digest) {
		return errors.New("module digest must be sha256")
	}
	if err := validateText("projection ID", record.ProjectionID); err != nil {
		return err
	}
	if err := validateText("projector ID", record.Projector.ID); err != nil {
		return err
	}
	if err := validateText("projector version", record.Projector.Version); err != nil {
		return err
	}
	if len(record.ScopeIDs) == 0 {
		return errors.New("record requires at least one scope ID")
	}
	if err := validateUniqueText("scope ID", record.ScopeIDs); err != nil {
		return err
	}
	if err := validateUniqueText("policy ID", record.PolicyIDs); err != nil {
		return err
	}
	if len(record.Artifacts) == 0 {
		return errors.New("record requires at least one artifact")
	}
	seen := map[string]bool{}
	aliases := map[string]string{}
	for _, artifact := range record.Artifacts {
		if err := validatePath(artifact.Path); err != nil {
			return err
		}
		if seen[artifact.Path] {
			return fmt.Errorf("duplicate artifact path %q", artifact.Path)
		}
		seen[artifact.Path] = true
		alias := strings.ToLower(artifact.Path)
		if previous, exists := aliases[alias]; exists && previous != artifact.Path {
			return fmt.Errorf("record artifact paths %q and %q collide under portable case matching", previous, artifact.Path)
		}
		aliases[alias] = artifact.Path
		if !isDigest(artifact.Digest) {
			return fmt.Errorf("artifact %q digest must be sha256", artifact.Path)
		}
		if err := validateMode(artifact.Mode); err != nil {
			return fmt.Errorf("artifact %q: %w", artifact.Path, err)
		}
		if artifact.Change != ChangeCreated && artifact.Change != ChangeModified && artifact.Change != ChangeRetained {
			return fmt.Errorf("artifact %q has unsupported change %q; deletion is not supported", artifact.Path, artifact.Change)
		}
	}
	if record.PriorRecordID != "" && !isDigest(record.PriorRecordID) {
		return errors.New("prior record ID must be sha256")
	}
	switch record.State {
	case StateMaterializedUnverified, StatePartialFailure, StateEscalated:
	default:
		return fmt.Errorf("unsupported materialization state %q", record.State)
	}
	return nil
}
func validateVerificationFields(result VerificationResult) error {
	if result.APIVersion != VerificationResultAPIVersion {
		return fmt.Errorf("unsupported verification result apiVersion %q", result.APIVersion)
	}
	if !isDigest(result.RecordID) {
		return errors.New("result requires a projection record ID")
	}
	if !revisionPattern.MatchString(result.Revision) {
		return errors.New("result requires a full immutable revision")
	}
	if !isDigest(result.ModelDigest) || !isDigest(result.TargetSnapshotDigest) {
		return errors.New("model and target snapshot digests must be sha256")
	}
	if (result.EvidenceRevision == "") != (result.EvidenceSnapshotDigest == "") {
		return errors.New("verification evidence revision and snapshot digest must be supplied together")
	}
	if result.EvidenceRevision != "" {
		if !revisionPattern.MatchString(result.EvidenceRevision) {
			return errors.New("verification evidence revision must be a full immutable revision")
		}
		if !isDigest(result.EvidenceSnapshotDigest) {
			return errors.New("verification evidence snapshot digest must be sha256")
		}
	}
	if (result.ControllerConfigDigest == "") != (result.ControllerVerifierInputDigest == "") {
		return errors.New("controller config and verifier input digests must be supplied together")
	}
	if result.ControllerConfigDigest != "" {
		if result.EvidenceRevision == "" {
			return errors.New("controller verification provenance requires evidence revision and snapshot digest")
		}
		if !isDigest(result.ControllerConfigDigest) || !isDigest(result.ControllerVerifierInputDigest) {
			return errors.New("controller config and verifier input digests must be sha256")
		}
	}
	if err := validateVerifier(result.Verifier); err != nil {
		return err
	}
	if err := validateCheckResults(result.Checks); err != nil {
		return err
	}
	switch result.Outcome {
	case OutcomePassed, OutcomeFailed, OutcomeIncomplete, OutcomeEscalated:
	default:
		return fmt.Errorf("unsupported verification outcome %q", result.Outcome)
	}
	if result.Outcome == OutcomePassed {
		if result.Verifier.Digest == "" || len(result.Checks) == 0 {
			return errors.New("passed result requires a verifier digest and at least one check")
		}
		for _, check := range result.Checks {
			if check.Digest == "" || check.Outcome != CheckPassed {
				return errors.New("passed result cannot contain failed or incomplete checks")
			}
		}
	}
	if result.Outcome == OutcomeEscalated && strings.TrimSpace(result.Reason) == "" {
		return errors.New("escalated result requires a reason")
	}
	return nil
}
func validateVerifier(v VerifierIdentity) error {
	if err := validateText("verifier ID", v.ID); err != nil {
		return err
	}
	if err := validateText("verifier version", v.Version); err != nil {
		return err
	}
	if v.Digest != "" && !isDigest(v.Digest) {
		return errors.New("verifier digest must be sha256 when supplied")
	}
	return nil
}
func validateCheckResults(checks []CheckResult) error {
	seen := map[string]bool{}
	for _, check := range checks {
		if err := validateText("check ID", check.ID); err != nil {
			return err
		}
		if err := validateText("check version", check.Version); err != nil {
			return err
		}
		if check.Digest != "" && !isDigest(check.Digest) {
			return fmt.Errorf("check %q digest must be sha256", check.ID)
		}
		if seen[check.ID] {
			return fmt.Errorf("duplicate verification check %q", check.ID)
		}
		seen[check.ID] = true
		switch check.Outcome {
		case CheckPassed, CheckFailed, CheckIncomplete:
		default:
			return fmt.Errorf("unsupported check outcome %q", check.Outcome)
		}
	}
	return nil
}
func validateCheckIdentities(checks []CheckIdentity) error {
	for i, check := range checks {
		if err := validateText("check ID", check.ID); err != nil {
			return err
		}
		if err := validateText("check version", check.Version); err != nil {
			return err
		}
		if !isDigest(check.Digest) {
			return fmt.Errorf("declared check %q digest must be sha256", check.ID)
		}
		if i > 0 && checks[i-1].ID == check.ID {
			return fmt.Errorf("duplicate declared check %q", check.ID)
		}
	}
	return nil
}
func validateArtifactFact(f ArtifactFact) error {
	if err := validatePath(f.Path); err != nil {
		return err
	}
	if !isDigest(f.Digest) {
		return fmt.Errorf("inventory artifact %q digest must be sha256", f.Path)
	}
	if err := validateMode(f.Mode); err != nil {
		return fmt.Errorf("inventory artifact %q: %w", f.Path, err)
	}
	switch f.Role {
	case RoleCanonicalSource, RoleExternalInput, RoleProjectionTarget, RoleToolOwned, RoleVendorOwned, RoleIgnored, RoleExcluded, RoleUnknown:
	default:
		return fmt.Errorf("unsupported inventory role %q", f.Role)
	}
	if (f.Role == RoleExcluded || f.Role == RoleIgnored) && strings.TrimSpace(f.Reason) == "" {
		return fmt.Errorf("inventory artifact %q role %q requires a reason", f.Path, f.Role)
	}
	return nil
}
func statusForUnownedRole(role string) string {
	switch role {
	case RoleCanonicalSource:
		return OwnershipCanonicalSource
	case RoleExternalInput:
		return OwnershipExternalInput
	case RoleToolOwned:
		return OwnershipToolOwned
	case RoleVendorOwned:
		return OwnershipVendorOwned
	case RoleIgnored:
		return OwnershipIgnored
	case RoleExcluded:
		return OwnershipExcluded
	default:
		return OwnershipUnknown
	}
}
func projectionRecordDigest(r ProjectionRecord) (string, error) {
	r.ID = ""
	// Empty sets have one semantic digest even across JSON/YAML round trips.
	if len(r.ScopeIDs) == 0 {
		r.ScopeIDs = nil
	}
	if len(r.PolicyIDs) == 0 {
		r.PolicyIDs = nil
	}
	if len(r.Artifacts) == 0 {
		r.Artifacts = nil
	}
	return digestJSON(r)
}
func verificationDigest(r VerificationResult) (string, error) {
	r.ID = ""
	if len(r.Checks) == 0 {
		r.Checks = nil
	}
	return digestJSON(r)
}
func digestJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func targetSnapshotDigestFromArtifacts(artifacts []Artifact) (string, error) {
	items := make([]Artifact, len(artifacts))
	for i, a := range artifacts {
		items[i] = Artifact{Path: a.Path, Digest: a.Digest, Mode: a.Mode}
	}
	return digestJSON(items)
}
func validateText(label, value string) error {
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return fmt.Errorf("%s must be nonempty trimmed UTF-8 text without NUL", label)
	}
	return nil
}
func validateUniqueText(label string, values []string) error {
	seen := map[string]bool{}
	for _, value := range values {
		if err := validateText(label, value); err != nil {
			return err
		}
		if seen[value] {
			return fmt.Errorf("duplicate %s %q", label, value)
		}
		seen[value] = true
	}
	return nil
}
func sortedUnique(values []string) ([]string, error) {
	result := append([]string(nil), values...)
	sort.Strings(result)
	for i := 1; i < len(result); i++ {
		if result[i-1] == result[i] {
			return nil, fmt.Errorf("duplicate identity %q", result[i])
		}
	}
	return result, nil
}
func isSortedUnique(values []string) bool {
	for i, value := range values {
		if value == "" || (i > 0 && values[i-1] >= value) {
			return false
		}
	}
	return true
}
func validatePath(value string) error {
	if err := validateText("artifact path", value); err != nil {
		return err
	}
	if strings.Contains(value, "\\") || strings.Contains(value, ":") || strings.HasPrefix(value, "/") || path.Clean(value) != value {
		return fmt.Errorf("path %q must be a clean portable relative POSIX path", value)
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("path %q contains a control character", value)
		}
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.TrimRight(segment, " .") != segment {
			return fmt.Errorf("path %q has an invalid segment", value)
		}
		base := strings.ToUpper(strings.SplitN(segment, ".", 2)[0])
		switch base {
		case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
			return fmt.Errorf("path %q contains a nonportable device name", value)
		}
	}
	return nil
}
func validateMode(value string) error {
	if value != "100644" && value != "100755" {
		return fmt.Errorf("unsupported regular-file mode %q", value)
	}
	return nil
}
func isDigest(value string) bool { return digestPattern.MatchString(value) }
func artifactPaths(artifacts []Artifact) []string {
	out := make([]string, 0, len(artifacts))
	for _, a := range artifacts {
		out = append(out, a.Path)
	}
	return out
}
func sameVerifier(a, b VerifierIdentity) bool {
	return a.ID == b.ID && a.Version == b.Version && a.Digest == b.Digest
}
