// Package markdownreference implements an independent, bounded Markdown
// reference-bundle Projection Module over normalized Core inputs.
package markdownreference

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

const (
	ModuleName    = "markitect-markdown-reference"
	ModuleVersion = "1.0.0"
	Target        = "markdown"
	Entrypoint    = "markdown-reference"
	OutputName    = "canonical-projection.md"
	RegularFile   = "100644"

	maxObservations = 10_000
	maxObservedSize = 64 << 20
	maxBundleSize   = 64 << 20
	maxRenderedSize = 128 << 20
)

type Decision string

const (
	DecisionWork     Decision = "work"
	DecisionNoop     Decision = "no-op"
	DecisionEscalate Decision = "escalate"
)

// Target is explicit Host-resolved target configuration. Prefix is a
// repository-relative directory, and AllowedRoots are registered capability
// roots supplied by the Host.
type TargetConfig struct {
	Technology   string
	Prefix       string
	AllowedRoots []string
}

// Check is the bounded result of the Host's selection and target-inventory
// checks. The Module never discovers omitted Definitions or target files.
type Check struct {
	ScopeComplete     bool
	InventoryComplete bool
	CanonicalAffected bool
	RequestDigest     string
}

type ArtifactObservation struct {
	Path  string
	Bytes []byte
	Mode  string
}

type ArtifactBinding struct {
	Path   string
	Digest string
	Mode   string
}

type PriorRecord struct {
	Complete         bool
	RequestDigest    string
	Artifacts        []ArtifactBinding
	RetiredArtifacts []string
}

type Verification struct {
	Passed        bool
	RequestDigest string
	Artifacts     []ArtifactBinding
}

// Input contains only normalized, explicitly selected Core values and
// Host-translated target/check/observation/record facts.
type Input struct {
	Schemas     []core.Schema
	Selected    []core.DefinitionIdentity
	Definitions []core.Definition
	Policies    []core.Definition
	Target      TargetConfig
	Check       Check
	Observed    []ArtifactObservation
	Prior       *PriorRecord
	Verified    *Verification
}

type Artifact struct {
	Bytes []byte
	Mode  string
}

type Escalation struct {
	Code    string
	Message string
}

type Proposal struct {
	Decision Decision
	Reasons  []string
	Files    map[string]Artifact
	Issues   []Escalation
}

type RenderResult struct {
	Files  map[string]Artifact
	Issues []Escalation
}

// Render validates and renders the exact supplied semantic scope without
// making any claim about target inventory, prior ownership, or verification.
func Render(input Input) RenderResult {
	result := RenderResult{Files: map[string]Artifact{}}
	if issue := validateTarget(input.Target); issue != nil {
		result.Issues = []Escalation{*issue}
		return result
	}
	if issue := validateSemantics(input); issue != nil {
		result.Issues = []Escalation{*issue}
		return result
	}
	data, err := render(input)
	if err != nil {
		result.Issues = []Escalation{{Code: "reference.render.invalid", Message: err.Error()}}
		return result
	}
	if len(data) > maxRenderedSize {
		result.Issues = []Escalation{{Code: "reference.render.bounds", Message: "rendered bundle exceeds the Module output limit"}}
		return result
	}
	result.Files[path.Join(strings.TrimSuffix(input.Target.Prefix, "/"), OutputName)] = Artifact{Bytes: data, Mode: RegularFile}
	return result
}

// Propose renders only the exact supplied semantic scope. It does not access
// the repository or perform writes.
func Propose(input Input) Proposal {
	rendered := Render(input)
	if len(rendered.Issues) != 0 {
		issue := rendered.Issues[0]
		return escalate(issue.Code, issue.Message)
	}
	if !input.Check.ScopeComplete {
		return escalate("reference.scope.incomplete", "selected semantic scope is incomplete")
	}
	if !validDigest(input.Check.RequestDigest) {
		return escalate("reference.check.digest-invalid", "a valid request digest is required")
	}
	if input.Prior != nil && !validDigest(input.Prior.RequestDigest) {
		return escalate("reference.prior.digest-invalid", "the prior record has no valid request digest")
	}
	outputPath := path.Join(strings.TrimSuffix(input.Target.Prefix, "/"), OutputName)
	artifact := rendered.Files[outputPath]
	if !input.Check.InventoryComplete {
		return escalate("reference.inventory.incomplete", "target inventory is incomplete")
	}
	if len(input.Observed) > maxObservations {
		return escalate("reference.inventory.limit", "target inventory exceeds the Module limit")
	}
	observed, issue := observedByPath(input.Observed, strings.TrimSuffix(input.Target.Prefix, "/"))
	if issue != nil {
		return escalate(issue.Code, issue.Message)
	}
	if input.Prior == nil {
		if len(observed) != 0 {
			return escalate("reference.owner.unknown", "the target prefix contains an artifact without a prior Module ownership record")
		}
		return work(outputPath, artifact, "initial-materialization")
	}
	if !input.Prior.Complete {
		return escalate("reference.prior.incomplete", "the prior ownership record is incomplete")
	}
	if len(input.Prior.RetiredArtifacts) != 0 {
		return escalate("reference.prior.retired", "the prior record contains retired artifacts requiring owner review")
	}
	prior, issue := priorByPath(input.Prior.Artifacts, outputPath)
	if issue != nil {
		return escalate(issue.Code, issue.Message)
	}
	for observedPath := range observed {
		if observedPath != outputPath {
			return escalate("reference.target.ambiguous", "the selected target prefix contains an artifact outside this Module's prior ownership")
		}
	}
	current, exists := observed[outputPath]
	if exists && prior == nil {
		return escalate("reference.owner.unknown", "the output exists without matching prior ownership")
	}
	if prior == nil {
		return work(outputPath, artifact, "initial-materialization")
	}
	if !exists || current.Mode != prior.Mode || digest(current.Bytes) != prior.Digest {
		return work(outputPath, artifact, "projection-drift")
	}
	if !bytes.Equal(current.Bytes, artifact.Bytes) {
		reason := "projection-output-changed"
		if input.Check.CanonicalAffected {
			reason = "canonical-or-binding-change"
		}
		return work(outputPath, artifact, reason)
	}
	if !verificationCurrent(input.Check.RequestDigest, input.Verified, prior, current) {
		return Proposal{Decision: DecisionNoop, Reasons: []string{"evidence-refresh-required"}, Files: map[string]Artifact{}}
	}
	return Proposal{Decision: DecisionNoop, Reasons: []string{"representation-current"}, Files: map[string]Artifact{}}
}

func validateTarget(target TargetConfig) *Escalation {
	if target.Technology != Target {
		return &Escalation{Code: "reference.target.unsupported", Message: "the selected target must be markdown"}
	}
	requestedPrefix := strings.TrimSuffix(target.Prefix, "/")
	prefix, ok := cleanRelativeDir(requestedPrefix)
	if !ok {
		return &Escalation{Code: "reference.target-prefix.invalid", Message: "target prefix must be a clean repository-relative directory"}
	}
	if prefix != requestedPrefix {
		return &Escalation{Code: "reference.target-prefix.invalid", Message: "target prefix must use canonical slash-separated spelling"}
	}
	allowed := false
	for _, root := range target.AllowedRoots {
		canonicalRoot := strings.TrimSuffix(root, "/")
		cleanRoot, valid := cleanRelativeDir(canonicalRoot)
		if valid && cleanRoot == canonicalRoot && (prefix == canonicalRoot || strings.HasPrefix(prefix, canonicalRoot+"/")) {
			allowed = true
			break
		}
	}
	if !allowed {
		return &Escalation{Code: "reference.target.root-disallowed", Message: "target prefix is outside the explicitly supplied allowed roots"}
	}
	return nil
}

func validateSemantics(input Input) *Escalation {
	if len(input.Schemas) == 0 || len(input.Schemas) > core.MaxSchemas || len(input.Definitions) == 0 || len(input.Definitions) > core.MaxDefinitions || len(input.Selected) != len(input.Definitions) || len(input.Policies) == 0 || len(input.Policies) > core.MaxDefinitions {
		return &Escalation{Code: "reference.input.bounds", Message: "selected Schemas, Definitions, identities, or policies are empty, inconsistent, or over limit"}
	}
	totalSize := 0
	schemas := make(map[string]core.Schema, len(input.Schemas))
	for _, schema := range input.Schemas {
		if strings.TrimSpace(schema.APIVersion) == "" || strings.TrimSpace(schema.Purpose) == "" || len(schema.Kinds) == 0 {
			return &Escalation{Code: "reference.schema.invalid", Message: "a selected Schema is missing its API version, purpose, or Kinds"}
		}
		if _, duplicate := schemas[schema.APIVersion]; duplicate {
			return &Escalation{Code: "reference.schema.duplicate", Message: "selected Schemas contain a duplicate API version"}
		}
		schemas[schema.APIVersion] = schema
		encoded, err := json.Marshal(schema)
		if err != nil || len(encoded) > core.MaxSchemaBytes {
			return &Escalation{Code: "reference.schema.bounds", Message: "a selected Schema cannot be represented within Core's size limit"}
		}
		totalSize += len(encoded)
	}
	definitions := make(map[string]core.Definition, len(input.Definitions))
	selected := make(map[string]bool, len(input.Selected))
	for _, identity := range input.Selected {
		key := identity.Key()
		if selected[key] {
			return &Escalation{Code: "reference.selection.duplicate", Message: "selected Definition identities contain a duplicate"}
		}
		selected[key] = true
	}
	for _, definition := range input.Definitions {
		identity := definition.Identity()
		key := identity.Key()
		if !selected[key] {
			return &Escalation{Code: "reference.selection.mismatch", Message: "a supplied Definition is outside the exact selected identities"}
		}
		if _, duplicate := definitions[key]; duplicate {
			return &Escalation{Code: "reference.definition.duplicate", Message: "supplied Definitions contain a duplicate identity"}
		}
		if _, ok := schemas[definition.APIVersion]; !ok || strings.TrimSpace(definition.Purpose) == "" {
			return &Escalation{Code: "reference.definition.invalid", Message: "a selected Definition has no selected Schema or purpose"}
		}
		encoded, err := json.Marshal(definition)
		if err != nil || len(encoded) > core.MaxDefinitionBytes {
			return &Escalation{Code: "reference.definition.bounds", Message: "a selected Definition cannot be represented within Core's size limit"}
		}
		totalSize += len(encoded)
		definitions[key] = definition
	}
	if len(definitions) != len(selected) {
		return &Escalation{Code: "reference.selection.mismatch", Message: "the exact selected identity list and supplied Definitions differ"}
	}
	selectedKinds := map[string]bool{}
	for _, definition := range input.Definitions {
		kind := core.KindIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind}
		schema := schemas[kind.APIVersion]
		if _, exists := schema.Kinds[kind.Kind]; !exists {
			return &Escalation{Code: "reference.kind.missing", Message: "a selected Definition Kind is absent from its selected Schema"}
		}
		selectedKinds[kind.Key()] = true
	}
	policies := map[string]bool{}
	for _, policy := range input.Policies {
		kind, target, _, ok := policyDetails(policy)
		if !ok || target != Target || !selectedKinds[kind.Key()] || policies[kind.Key()] {
			return &Escalation{Code: "reference.policy.invalid", Message: "selected ProjectionPolicies must uniquely and validly cover selected Kinds for markdown"}
		}
		encoded, err := json.Marshal(policy)
		if err != nil || len(encoded) > core.MaxDefinitionBytes {
			return &Escalation{Code: "reference.policy.bounds", Message: "a selected ProjectionPolicy cannot be represented within Core's size limit"}
		}
		totalSize += len(encoded)
		if totalSize > maxBundleSize {
			return &Escalation{Code: "reference.bundle.bounds", Message: "selected canonical inputs exceed the Module bundle-size limit"}
		}
		policies[kind.Key()] = true
	}
	if len(policies) != len(selectedKinds) {
		return &Escalation{Code: "reference.policy.incomplete", Message: "every selected Kind requires exactly one selected Markdown ProjectionPolicy"}
	}
	return nil
}

func policyDetails(policy core.Definition) (core.KindIdentity, string, string, bool) {
	if policy.Kind != "ProjectionPolicy" {
		return core.KindIdentity{}, "", "", false
	}
	source, ok := policy.Spec["sourceKind"].(map[string]any)
	if !ok {
		return core.KindIdentity{}, "", "", false
	}
	api, apiOK := source["apiVersion"].(string)
	kind, kindOK := source["kind"].(string)
	target, targetOK := policy.Spec["targetTechnology"].(string)
	guidance, guidanceOK := policy.Spec["guidance"].(string)
	if !apiOK || !kindOK || !targetOK || !guidanceOK || api == "" || kind == "" || target == "" || strings.TrimSpace(guidance) == "" || strings.TrimSpace(policy.Purpose) == "" {
		return core.KindIdentity{}, "", "", false
	}
	return core.KindIdentity{APIVersion: api, Kind: kind}, target, guidance, true
}

func render(input Input) ([]byte, error) {
	var out bytes.Buffer
	fmt.Fprintf(&out, "# Canonical Reference Bundle\n\nModule: `%s@%s`  \nTarget: `%s`\n", ModuleName, ModuleVersion, input.Target.Technology)
	schemas := append([]core.Schema(nil), input.Schemas...)
	sort.Slice(schemas, func(i, j int) bool { return schemas[i].APIVersion < schemas[j].APIVersion })
	out.WriteString("\n## Schema contracts\n")
	for _, schema := range schemas {
		encoded, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode Schema %q: %w", schema.APIVersion, err)
		}
		fmt.Fprintf(&out, "\n<!-- markitect-reference:schema:%s -->\n\n~~~json\n%s\n~~~\n", schema.APIVersion, encoded)
	}
	definitions := append([]core.Definition(nil), input.Definitions...)
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Identity().Key() < definitions[j].Identity().Key() })
	out.WriteString("\n## Selected canonical Definitions\n")
	for _, definition := range definitions {
		encoded, err := json.MarshalIndent(definition, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode Definition %q: %w", definition.Identity().Key(), err)
		}
		fmt.Fprintf(&out, "\n<!-- markitect-reference:definition:%s -->\n\n~~~json\n%s\n~~~\n", definition.Identity().Key(), encoded)
	}
	policies := append([]core.Definition(nil), input.Policies...)
	sort.Slice(policies, func(i, j int) bool { return policies[i].Identity().Key() < policies[j].Identity().Key() })
	out.WriteString("\n## Selected ProjectionPolicies\n")
	for _, policy := range policies {
		encoded, err := json.MarshalIndent(policy, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode ProjectionPolicy %q: %w", policy.Identity().Key(), err)
		}
		fmt.Fprintf(&out, "\n<!-- markitect-reference:policy:%s -->\n\n~~~json\n%s\n~~~\n", policy.Identity().Key(), encoded)
	}
	return out.Bytes(), nil
}

func cleanRelativeDir(value string) (string, bool) {
	if strings.TrimSpace(value) == "" || strings.Contains(value, `\`) || strings.Contains(value, ":") || strings.HasPrefix(value, "/") {
		return "", false
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != value {
		return "", false
	}
	return clean, true
}

func observedByPath(observations []ArtifactObservation, targetPrefix string) (map[string]ArtifactObservation, *Escalation) {
	items := make(map[string]ArtifactObservation, len(observations))
	total := 0
	for _, item := range observations {
		clean, ok := cleanRelativeDir(item.Path)
		if !ok || clean != item.Path {
			return nil, &Escalation{Code: "reference.observation.path-invalid", Message: "an observed artifact path is not canonical and repository-relative"}
		}
		if !within(targetPrefix, item.Path) {
			return nil, &Escalation{Code: "reference.observation.outside-target", Message: "an observed artifact is outside the selected target prefix"}
		}
		if _, duplicate := items[item.Path]; duplicate {
			return nil, &Escalation{Code: "reference.observation.duplicate", Message: "target observations contain a duplicate path"}
		}
		total += len(item.Bytes)
		if total > maxObservedSize {
			return nil, &Escalation{Code: "reference.observation.bounds", Message: "observed target bytes exceed the Module limit"}
		}
		items[item.Path] = item
	}
	return items, nil
}

func within(root, name string) bool { return name == root || strings.HasPrefix(name, root+"/") }

func priorByPath(bindings []ArtifactBinding, outputPath string) (*ArtifactBinding, *Escalation) {
	if len(bindings) > 1 {
		return nil, &Escalation{Code: "reference.prior.ambiguous", Message: "the prior record claims multiple Module-owned artifacts"}
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	binding := bindings[0]
	if binding.Path != outputPath || binding.Mode != RegularFile || !validDigest(binding.Digest) {
		return nil, &Escalation{Code: "reference.prior.invalid", Message: "prior artifact path, digest, or regular-file mode does not match the Module contract"}
	}
	return &binding, nil
}

func verificationCurrent(request string, verification *Verification, prior *ArtifactBinding, observed ArtifactObservation) bool {
	if verification == nil || !verification.Passed || verification.RequestDigest != request || len(verification.Artifacts) != 1 {
		return false
	}
	proof := verification.Artifacts[0]
	return proof.Path == prior.Path && proof.Digest == prior.Digest && proof.Mode == prior.Mode && digest(observed.Bytes) == prior.Digest && observed.Mode == prior.Mode
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func digest(value []byte) string {
	encoded := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(encoded[:])
}

func work(path string, artifact Artifact, reason string) Proposal {
	return Proposal{Decision: DecisionWork, Reasons: []string{reason}, Files: map[string]Artifact{path: artifact}}
}

func escalate(code, message string) Proposal {
	return Proposal{Decision: DecisionEscalate, Files: map[string]Artifact{}, Issues: []Escalation{{Code: code, Message: message}}}
}
