package markdown

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

// ArtifactObservation is a caller-supplied observation. A complete inventory
// is required before this package can claim no materialization work.
type ArtifactObservation struct {
	Path  string
	Bytes []byte
	Mode  string
}

// ArtifactBinding records the bytes and mode claimed by an active record or a
// passing verification result.
type ArtifactBinding struct {
	Path   string
	Digest string
	Mode   string
}

// PriorProjection is the Host-translated active record for this Projection.
type PriorProjection struct {
	RequestDigest    string
	Complete         bool
	Artifacts        []ArtifactBinding
	RetiredArtifacts []string
}

// VerificationBinding is a Host-translated result for this exact request and
// the artifact set it checked.
type VerificationBinding struct {
	Passed        bool
	RequestDigest string
	Artifacts     []ArtifactBinding
}

// Decision is the Module's disposition for its own selected representation.
type Decision string

const (
	DecisionWork     Decision = "work"
	DecisionNoop     Decision = "no-op"
	DecisionEscalate Decision = "escalate"
)

// ProposalInput extends the existing Render input with only the bounded state
// needed for a Module-owned decision. TargetPrefix is a directory prefix.
// RequestDigest binds the current request. CanonicalAffected separates local
// semantic work from conservative global evidence refresh.
type ProposalInput = Input

// Proposal reports materialization work only. Files contains the desired
// deterministic page for WORK; it is empty for NOOP and ESCALATE. Evidence
// refresh remains an independent fact and never implies artifact edits.
type Proposal struct {
	Decision                Decision
	Reasons                 []string
	EvidenceRefreshRequired bool
	Files                   map[string][]byte
	Escalations             []Escalation
}

// Escalation identifies a missing or ambiguous fact the Module cannot decide.
type Escalation struct {
	Code    string
	Message string
}

const pageName = "index.md"

// RenderProjection renders the canonical scope to the Markdown capability's
// module-owned page beneath TargetPrefix. Render's exact TargetPath API remains
// available to existing callers.
func RenderProjection(input Input) Result {
	root, err := normalizeTargetPrefix(input.TargetPrefix)
	if err != nil {
		return Result{Files: map[string][]byte{}, Diagnostics: []core.Diagnostic{{Code: "markdown.target-prefix.invalid", Message: err.Error()}}}
	}
	input.TargetPath = path.Join(root, pageName)
	return Render(input)
}

// Propose decides whether the Markdown capability needs to materialize its
// deterministic page. The page name and rendering convention belong here,
// not in Host. This function never writes files or deletes retired artifacts.
func Propose(input Input) Proposal {
	out := Proposal{Decision: DecisionEscalate, Files: map[string][]byte{}}
	root, err := normalizeTargetPrefix(input.TargetPrefix)
	if err != nil {
		return escalate("target.prefix.invalid", err.Error())
	}
	if !underAllowedRoot(root, input.AllowedRoots) {
		return escalate("target.root.disallowed", "Projection target is outside the capability's registered roots")
	}
	if input.Previous != nil && !validDigest(input.Previous.RequestDigest) {
		return escalate("record.request.digest.invalid", "active record contains an invalid request digest")
	}
	if !validDigest(input.RequestDigest) {
		return escalate("request.digest.missing", "current request digest is required")
	}
	if !input.InventoryComplete {
		return escalate("target.inventory.incomplete", "the target prefix was not completely observed")
	}
	if err := validatePolicies(input.Policies, input.Schemas, input.Definitions); err != nil {
		return escalate("policy.invalid", err.Error())
	}
	pathName := path.Join(root, pageName)
	rendered := Render(Input{Definitions: input.Definitions, Schemas: input.Schemas, Policies: input.Policies, TargetPath: pathName})
	if len(rendered.Diagnostics) > 0 || len(rendered.Files[pathName]) == 0 {
		return escalate("scope.invalid", "selected Core scope cannot be rendered into the declared target")
	}
	if len(input.ObservedArtifacts) > core.MaxDefinitions {
		return escalate("target.inventory.limit", "target inventory exceeds the bounded artifact count")
	}
	observed := map[string]ArtifactObservation{}
	for _, artifact := range input.ObservedArtifacts {
		clean, pathErr := normalizeArtifactPath(artifact.Path)
		if pathErr != nil || clean != artifact.Path || !within(root, clean) || artifact.Mode == "" {
			return escalate("target.inventory.invalid", "target inventory contains an invalid path or missing mode")
		}
		if _, exists := observed[clean]; exists {
			return escalate("target.inventory.duplicate", "target inventory contains a duplicate path")
		}
		observed[clean] = artifact
	}
	prior := map[string]ArtifactBinding{}
	if input.Previous != nil {
		if len(input.Previous.RetiredArtifacts) > 0 {
			return escalate("artifact.retired", "active ownership includes artifacts outside this Module's current output set; review is required")
		}
		if len(input.Previous.Artifacts) > core.MaxDefinitions {
			return escalate("record.artifact.limit", "active record exceeds the bounded artifact count")
		}
		for _, artifact := range input.Previous.Artifacts {
			clean, pathErr := normalizeArtifactPath(artifact.Path)
			if pathErr != nil || clean != artifact.Path || !within(root, clean) || !validDigest(artifact.Digest) || artifact.Mode == "" {
				return escalate("record.artifact.invalid", "active record contains an invalid artifact binding")
			}
			if clean != pathName {
				return escalate("artifact.retired", "active ownership contains a path this Markdown capability no longer proposes")
			}
			if _, exists := prior[clean]; exists {
				return escalate("record.artifact.duplicate", "active record contains a duplicate artifact path")
			}
			prior[clean] = artifact
		}
	}
	for name := range observed {
		if _, owned := prior[name]; !owned {
			return escalate("artifact.unknown", "target contains an unowned artifact that must be classified before projection")
		}
	}
	if input.Previous == nil {
		out.Decision = DecisionWork
		out.Reasons = []string{"not-materialized"}
		out.Files = rendered.Files
		return out
	}
	if _, ok := prior[pathName]; !ok {
		if _, exists := observed[pathName]; exists {
			return escalate("artifact.unowned", "the proposed Markdown page exists without active ownership")
		}
		out.Decision = DecisionWork
		out.Reasons = []string{"incomplete-materialization"}
		out.Files = rendered.Files
		return out
	}
	binding := prior[pathName]
	current, exists := observed[pathName]
	if !exists {
		out.Decision = DecisionWork
		out.Reasons = []string{"projection-drift"}
		out.Files = rendered.Files
		return out
	}
	currentDigest := digest(current.Bytes)
	if currentDigest != binding.Digest || current.Mode != binding.Mode {
		out.Decision = DecisionWork
		out.Reasons = []string{"projection-drift"}
		out.Files = rendered.Files
		return out
	}
	if !input.Previous.Complete {
		out.Decision = DecisionWork
		out.Reasons = []string{"incomplete-materialization"}
		out.Files = rendered.Files
		return out
	}
	if !bytesEqual(current.Bytes, rendered.Files[pathName]) {
		out.Decision = DecisionWork
		if input.CanonicalAffected {
			out.Reasons = []string{"canonical-or-binding-change"}
		} else {
			out.Reasons = []string{"projection-output-changed"}
		}
		out.Files = rendered.Files
		return out
	}
	out.Decision = DecisionNoop
	out.EvidenceRefreshRequired = !verificationCurrent(input.RequestDigest, input.Verification, prior, observed)
	if out.EvidenceRefreshRequired {
		out.Reasons = []string{"evidence-refresh-required"}
	} else {
		out.Reasons = []string{"representation-current"}
	}
	return out
}

func validatePolicies(policies []core.Definition, schemas []core.Schema, definitions []core.Definition) error {
	selected := map[string]bool{}
	available := map[string]bool{}
	for _, schema := range schemas {
		for kind := range schema.Kinds {
			available[(core.KindIdentity{APIVersion: schema.APIVersion, Kind: kind}).Key()] = true
		}
	}
	for _, definition := range definitions {
		selected[(core.KindIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind}).Key()] = true
	}
	seen := map[string]bool{}
	for _, policy := range policies {
		kind, target, guidance, err := policyFields(policy)
		if err != nil {
			return err
		}
		key := kind.Key()
		if target != "markdown" {
			return fmt.Errorf("policy %s targets %q rather than markdown", policy.Identity().Key(), target)
		}
		if !available[key] || !selected[key] {
			return fmt.Errorf("policy %s references a Kind outside the selected Schema scope", policy.Identity().Key())
		}
		if strings.TrimSpace(guidance) == "" {
			return fmt.Errorf("policy %s has empty guidance", policy.Identity().Key())
		}
		if seen[key] {
			return fmt.Errorf("multiple Markdown policies select Kind %s", key)
		}
		seen[key] = true
	}
	selectedKeys := make([]string, 0, len(selected))
	for key := range selected {
		selectedKeys = append(selectedKeys, key)
	}
	sort.Strings(selectedKeys)
	for _, key := range selectedKeys {
		if !seen[key] {
			return fmt.Errorf("Kind %s has no selected Markdown ProjectionPolicy", key)
		}
	}
	return nil
}

func policyFields(policy core.Definition) (core.KindIdentity, string, string, error) {
	if policy.Kind != "ProjectionPolicy" {
		return core.KindIdentity{}, "", "", fmt.Errorf("selected policy %s is not a ProjectionPolicy", policy.Identity().Key())
	}
	source, ok := policy.Spec["sourceKind"].(map[string]any)
	if !ok {
		return core.KindIdentity{}, "", "", fmt.Errorf("policy %s has no sourceKind object", policy.Identity().Key())
	}
	api, apiOK := source["apiVersion"].(string)
	kind, kindOK := source["kind"].(string)
	target, targetOK := policy.Spec["targetTechnology"].(string)
	guidance, guidanceOK := policy.Spec["guidance"].(string)
	if !apiOK || !kindOK || !targetOK || !guidanceOK || api == "" || kind == "" || target == "" {
		return core.KindIdentity{}, "", "", fmt.Errorf("policy %s has malformed sourceKind, targetTechnology, or guidance", policy.Identity().Key())
	}
	return core.KindIdentity{APIVersion: api, Kind: kind}, target, guidance, nil
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
		actual, exists := observed[name]
		proof, proved := verified[name]
		if !exists || !proved || digest(actual.Bytes) != binding.Digest || actual.Mode != binding.Mode || proof.Digest != binding.Digest || proof.Mode != binding.Mode {
			return false
		}
	}
	return true
}

func underAllowedRoot(target string, roots []string) bool {
	if len(roots) == 0 {
		return false
	}
	for _, root := range roots {
		clean, err := normalizeTargetPrefix(root)
		if err == nil && within(clean, target) {
			return true
		}
	}
	return false
}
func within(root, name string) bool                      { return name == root || strings.HasPrefix(name, root+"/") }
func normalizeArtifactPath(value string) (string, error) { return normalizeTargetPrefix(value) }
func normalizeTargetPrefix(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.Contains(value, "\\") || path.IsAbs(value) {
		return "", fmt.Errorf("target prefix must be a clean relative slash path")
	}
	clean := path.Clean(strings.TrimSuffix(value, "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != strings.TrimSuffix(value, "/") {
		return "", fmt.Errorf("target prefix must be a clean relative slash path")
	}
	return clean, nil
}
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
func bytesEqual(a, b []byte) bool {
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
func escalate(code, message string) Proposal {
	return Proposal{Decision: DecisionEscalate, Reasons: []string{code}, Escalations: []Escalation{{Code: code, Message: message}}, Files: map[string][]byte{}}
}
