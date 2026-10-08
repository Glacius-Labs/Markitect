// Package azurepipelines implements the Markitect Azure Pipelines projection
// Module. It consumes only explicitly selected Core values and Host-supplied
// check argv; it does not access the repository or Azure services.
package azurepipelines

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

const (
	ModuleName       = "markitect-azure-pipelines"
	ModuleVersion    = "1.0.0"
	ProjectorID      = "azure-pipeline"
	TargetTechnology = "azurepipelines"
	OutputName       = "azure-pipelines.yml"
	OutputMode       = "100644"
	RequiredCheck    = "canonical-workflow-check"
	maxPathBytes     = 4096
	maxPathParts     = 64
	maxPathPartBytes = 255
	maxScopeRecords  = 1024
	maxOutputBytes   = 8 << 20
)

// NamedCheck is an exact Project check resolved by Host. The Module never
// discovers commands from prose or runs them itself.
type NamedCheck struct {
	Name string
	Argv []string
}

// ArtifactObservation is a caller-supplied file within the selected target
// prefix. This Module performs no filesystem observation.
type ArtifactObservation struct {
	Path  string
	Bytes []byte
	Mode  string
}

// ArtifactBinding records an exact path, digest, and mode from an active
// ProjectionRecord or successful VerificationResult translated by Host.
type ArtifactBinding struct {
	Path   string
	Digest string
	Mode   string
}

type PriorProjection struct {
	RequestDigest    string
	Complete         bool
	Artifacts        []ArtifactBinding
	RetiredArtifacts []string
}

type VerificationBinding struct {
	Passed        bool
	RequestDigest string
	Artifacts     []ArtifactBinding
}

// Input is the exact selected scope and target boundary supplied by Host.
// Edges remain available for future policies that explicitly need them; this
// Module currently does not infer pipeline behavior from relationships.
type Input struct {
	Definitions       []core.Definition
	Schemas           []core.Schema
	Edges             []core.Edge
	Policies          []core.Definition
	TargetPrefix      string
	AllowedRoots      []string
	Checks            []NamedCheck
	RequestDigest     string
	CanonicalAffected bool
	InventoryComplete bool
	Previous          *PriorProjection
	ObservedArtifacts []ArtifactObservation
	Verification      *VerificationBinding
}

type Decision string

const (
	DecisionWork     Decision = "work"
	DecisionNoop     Decision = "no-op"
	DecisionEscalate Decision = "escalate"
)

type Escalation struct {
	Code    string
	Message string
}

// Result contains deterministic candidate bytes. It does not write a file or
// claim Azure execution, remote acceptance, or human approval.
type Result struct {
	Files       map[string][]byte
	Mode        string
	Diagnostics []core.Diagnostic
}

// Proposal is this Module's work/no-op/escalate decision for one target.
// Files is populated only for work and names only the Module-owned output.
type Proposal struct {
	Decision                Decision
	Reasons                 []string
	EvidenceRefreshRequired bool
	Files                   map[string][]byte
	Mode                    string
	Escalations             []Escalation
}

// Render creates the Azure YAML from explicit policies and exact Host-resolved
// checks. The caller remains responsible for applying and verifying bytes.
func Render(input Input) Result {
	result := Result{Files: map[string][]byte{}, Mode: OutputMode}
	root, err := normalizePortablePath(input.TargetPrefix)
	if err != nil {
		return diagnostic("azurepipelines.target-prefix.invalid", err.Error())
	}
	if !underAllowedRoot(root, input.AllowedRoots) {
		return diagnostic("azurepipelines.target-root.disallowed", "Projection target is outside the capability's registered roots")
	}
	policies, err := validateScope(input)
	if err != nil {
		return diagnostic("azurepipelines.scope.invalid", err.Error())
	}
	checks, err := validateChecks(input.Checks)
	if err != nil {
		return diagnostic("azurepipelines.check.invalid", err.Error())
	}
	target := path.Join(root, OutputName)
	data, err := renderYAML(target, input.Definitions, policies, checks)
	if err != nil {
		return diagnostic("azurepipelines.render.failed", err.Error())
	}
	result.Files[target] = data
	return result
}

// Propose owns the disposition for the exact Azure Pipelines file. Unknown,
// incomplete, stale, or ambiguous state fails closed. A no-op needs an exact
// prior binding, complete inventory, byte/mode equality, and current evidence.
func Propose(input Input) Proposal {
	escalate := func(code, message string) Proposal {
		return Proposal{Decision: DecisionEscalate, Reasons: []string{code}, Files: map[string][]byte{}, Escalations: []Escalation{{Code: code, Message: message}}}
	}
	root, err := normalizePortablePath(input.TargetPrefix)
	if err != nil {
		return escalate("target.prefix.invalid", err.Error())
	}
	if !underAllowedRoot(root, input.AllowedRoots) {
		return escalate("target.root.disallowed", "Projection target is outside the capability's registered roots")
	}
	if !validDigest(input.RequestDigest) {
		return escalate("request.digest.invalid", "a canonical request digest is required")
	}
	if input.Previous != nil && !validDigest(input.Previous.RequestDigest) {
		return escalate("record.request.digest.invalid", "active record contains an invalid request digest")
	}
	if !input.InventoryComplete {
		return escalate("target.inventory.incomplete", "the target prefix was not completely observed")
	}
	rendered := Render(input)
	if len(rendered.Diagnostics) != 0 {
		return escalate("projection.input.invalid", rendered.Diagnostics[0].Message)
	}
	target := path.Join(root, OutputName)
	if len(input.ObservedArtifacts) > core.MaxDefinitions {
		return escalate("target.inventory.limit", "target inventory exceeds the bounded artifact count")
	}
	observed := make(map[string]ArtifactObservation, len(input.ObservedArtifacts))
	for _, item := range input.ObservedArtifacts {
		clean, pathErr := normalizePortablePath(item.Path)
		if pathErr != nil || clean != item.Path || !within(root, clean) || item.Mode == "" {
			return escalate("target.inventory.invalid", "target inventory contains an invalid path or missing mode")
		}
		if _, exists := observed[clean]; exists {
			return escalate("target.inventory.duplicate", "target inventory contains a duplicate path")
		}
		observed[clean] = item
	}
	for name := range observed {
		if strings.EqualFold(name, target) && name != target {
			return escalate("artifact.path.alias", "target inventory contains a case-fold alias of the Module-owned Azure pipeline")
		}
	}
	prior := make(map[string]ArtifactBinding)
	if input.Previous != nil {
		if len(input.Previous.RetiredArtifacts) != 0 {
			return escalate("artifact.retired", "active ownership includes retired artifacts; review is required and this Module never deletes files")
		}
		if len(input.Previous.Artifacts) > core.MaxDefinitions {
			return escalate("record.artifact.limit", "active record exceeds the bounded artifact count")
		}
		for _, item := range input.Previous.Artifacts {
			clean, pathErr := normalizePortablePath(item.Path)
			if pathErr != nil || clean != item.Path || !within(root, clean) || !validDigest(item.Digest) || item.Mode != OutputMode {
				return escalate("record.artifact.invalid", "active record contains an invalid artifact binding")
			}
			if clean != target {
				return escalate("artifact.retired", "active ownership contains a path this Module no longer proposes")
			}
			if _, exists := prior[clean]; exists {
				return escalate("record.artifact.duplicate", "active record contains a duplicate artifact path")
			}
			prior[clean] = item
		}
	}
	for name := range observed {
		if _, owned := prior[name]; !owned {
			return escalate("artifact.unknown", "target contains an unowned artifact that must be classified before projection")
		}
	}
	if input.Previous == nil {
		return work(rendered, "not-materialized")
	}
	binding, owned := prior[target]
	current, exists := observed[target]
	if !owned {
		if exists {
			return escalate("artifact.unowned", "the proposed Azure pipeline exists without active ownership")
		}
		return work(rendered, "incomplete-materialization")
	}
	if !exists || digest(current.Bytes) != binding.Digest || current.Mode != binding.Mode {
		return work(rendered, "projection-drift")
	}
	if !input.Previous.Complete {
		return work(rendered, "incomplete-materialization")
	}
	if !bytes.Equal(current.Bytes, rendered.Files[target]) {
		if input.CanonicalAffected {
			return work(rendered, "canonical-or-binding-change")
		}
		return work(rendered, "projection-output-changed")
	}
	proposal := Proposal{Decision: DecisionNoop, Files: map[string][]byte{}, Mode: OutputMode}
	proposal.EvidenceRefreshRequired = !verificationCurrent(input.RequestDigest, input.Verification, prior, observed)
	if proposal.EvidenceRefreshRequired {
		proposal.Reasons = []string{"evidence-refresh-required"}
	} else {
		proposal.Reasons = []string{"representation-current"}
	}
	return proposal
}

func work(rendered Result, reason string) Proposal {
	return Proposal{Decision: DecisionWork, Reasons: []string{reason}, Files: rendered.Files, Mode: OutputMode}
}

func diagnostic(code, message string) Result {
	return Result{Files: map[string][]byte{}, Mode: OutputMode, Diagnostics: []core.Diagnostic{{Code: code, Message: message}}}
}

type policyBinding struct {
	definition core.Definition
}

func validateScope(input Input) (map[string]policyBinding, error) {
	if len(input.Definitions) == 0 {
		return nil, fmt.Errorf("selected Core scope contains no Definitions")
	}
	if len(input.Definitions) > maxScopeRecords || len(input.Schemas) > core.MaxSchemas || len(input.Policies) > maxScopeRecords {
		return nil, fmt.Errorf("selected Core scope exceeds the bounded Module input limits")
	}
	kinds := make(map[string]bool)
	for _, definition := range input.Definitions {
		if !schemaContains(input.Schemas, core.KindIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind}) {
			return nil, fmt.Errorf("Definition %s has no supplied Schema contract", definition.Identity().Key())
		}
		kinds[(core.KindIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind}).Key()] = true
	}
	matched := make(map[string]policyBinding, len(kinds))
	for _, policy := range input.Policies {
		identity := policy.Identity().Key()
		if policy.Kind != "ProjectionPolicy" {
			return nil, fmt.Errorf("selected resource %s is not a ProjectionPolicy", identity)
		}
		source, ok := policy.Spec["sourceKind"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("ProjectionPolicy %s has no sourceKind object", identity)
		}
		api, apiOK := source["apiVersion"].(string)
		kind, kindOK := source["kind"].(string)
		target, targetOK := policy.Spec["targetTechnology"].(string)
		guidance, guidanceOK := policy.Spec["guidance"].(string)
		if !apiOK || !kindOK || !targetOK || !guidanceOK || api == "" || kind == "" || strings.TrimSpace(target) != target || target == "" {
			return nil, fmt.Errorf("ProjectionPolicy %s has malformed sourceKind, targetTechnology, or guidance", identity)
		}
		if target != TargetTechnology {
			return nil, fmt.Errorf("ProjectionPolicy %s targets %q rather than %s", identity, target, TargetTechnology)
		}
		key := (core.KindIdentity{APIVersion: api, Kind: kind}).Key()
		if !kinds[key] {
			return nil, fmt.Errorf("ProjectionPolicy %s references a Kind outside the selected scope", identity)
		}
		if strings.TrimSpace(guidance) == "" {
			return nil, fmt.Errorf("ProjectionPolicy %s has empty guidance", identity)
		}
		if _, duplicate := matched[key]; duplicate {
			return nil, fmt.Errorf("Kind %s has multiple selected Azure Pipelines ProjectionPolicies", key)
		}
		matched[key] = policyBinding{definition: policy}
	}
	for key := range kinds {
		if _, exists := matched[key]; !exists {
			return nil, fmt.Errorf("Kind %s has no selected Azure Pipelines ProjectionPolicy", key)
		}
	}
	return matched, nil
}

func validateChecks(checks []NamedCheck) ([]NamedCheck, error) {
	if len(checks) == 0 || len(checks) > maxScopeRecords {
		return nil, fmt.Errorf("at least one bounded required Project check must be supplied")
	}
	byName := make(map[string]bool, len(checks))
	result := append([]NamedCheck(nil), checks...)
	foundRequired := false
	for _, check := range result {
		if check.Name == "" || len(check.Name) > 256 || !utf8.ValidString(check.Name) || strings.TrimSpace(check.Name) != check.Name || strings.ContainsAny(check.Name, "\r\n") {
			return nil, fmt.Errorf("required Project check has an invalid name")
		}
		if byName[check.Name] {
			return nil, fmt.Errorf("required Project check %q is duplicated", check.Name)
		}
		byName[check.Name] = true
		if check.Name == RequiredCheck {
			foundRequired = true
		}
		if len(check.Argv) == 0 || len(check.Argv) > 256 {
			return nil, fmt.Errorf("required Project check %q has no bounded argv", check.Name)
		}
		for _, arg := range check.Argv {
			if arg == "" || len(arg) > maxPathBytes || !utf8.ValidString(arg) || strings.TrimSpace(arg) != arg || strings.ContainsAny(arg, "\r\n") || !portableShellToken(arg) {
				return nil, fmt.Errorf("required Project check %q contains an argv token that cannot be projected without shell reinterpretation", check.Name)
			}
		}
	}
	if !foundRequired {
		return nil, fmt.Errorf("required Project check %q was not supplied by Host", RequiredCheck)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// portableShellToken permits only characters that remain one literal argv
// token in both common Azure script shells when joined by spaces.
func portableShellToken(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._/+:-=", r) {
			continue
		}
		return false
	}
	return true
}

func renderYAML(target string, definitions []core.Definition, policies map[string]policyBinding, checks []NamedCheck) ([]byte, error) {
	definitions = append([]core.Definition(nil), definitions...)
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Identity().Key() < definitions[j].Identity().Key() })
	policyDefinitions := make([]core.Definition, 0, len(policies))
	for _, value := range policies {
		policyDefinitions = append(policyDefinitions, value.definition)
	}
	sort.Slice(policyDefinitions, func(i, j int) bool {
		return policyDefinitions[i].Identity().Key() < policyDefinitions[j].Identity().Key()
	})
	var out bytes.Buffer
	fmt.Fprintf(&out, "# Markitect Module: %s@%s\n", ModuleName, ModuleVersion)
	fmt.Fprintf(&out, "# Projector: %s\n# Target: %s\n# Artifact: %s\n", ProjectorID, TargetTechnology, target)
	out.WriteString("# Source Definitions:\n")
	for _, definition := range definitions {
		fmt.Fprintf(&out, "#   %s\n", definition.Identity().Key())
		fmt.Fprintf(&out, "#     source: %s:%d %s\n", strconv.Quote(definition.Source.Path), definition.Source.Line, strconv.Quote(definition.Source.Digest))
		if err := writeJSONComment(&out, "Markitect Definition JSON", definition); err != nil {
			return nil, err
		}
	}
	out.WriteString("# ProjectionPolicies:\n")
	for _, policy := range policyDefinitions {
		fmt.Fprintf(&out, "#   %s\n", policy.Identity().Key())
		if err := writeJSONComment(&out, "Markitect ProjectionPolicy JSON", policy); err != nil {
			return nil, err
		}
	}
	out.WriteString("steps:\n")
	for _, check := range checks {
		out.WriteString("  - script: ")
		out.WriteString(yamlSingleQuote(strings.Join(check.Argv, " ")))
		out.WriteString("\n    displayName: ")
		out.WriteString(yamlSingleQuote(check.Name))
		out.WriteByte('\n')
		if out.Len() > maxOutputBytes {
			return nil, fmt.Errorf("rendered Azure pipeline exceeds the Module output size limit")
		}
	}
	if out.Len() > maxOutputBytes {
		return nil, fmt.Errorf("rendered Azure pipeline exceeds the Module output size limit")
	}
	return out.Bytes(), nil
}

func writeJSONComment(out *bytes.Buffer, label string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode inert %s provenance: %w", label, err)
	}
	fmt.Fprintf(out, "# %s: %s\n", label, encoded)
	if out.Len() > maxOutputBytes {
		return fmt.Errorf("rendered Azure pipeline exceeds the Module output size limit")
	}
	return nil
}

func yamlSingleQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }

func schemaContains(schemas []core.Schema, identity core.KindIdentity) bool {
	for _, schema := range schemas {
		if schema.APIVersion == identity.APIVersion {
			if _, ok := schema.Kinds[identity.Kind]; ok {
				return true
			}
		}
	}
	return false
}

func normalizePrefix(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.Contains(value, "\\") || path.IsAbs(value) {
		return "", fmt.Errorf("target prefix must be a clean relative slash path")
	}
	trimmed := strings.TrimSuffix(value, "/")
	clean := path.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != trimmed {
		return "", fmt.Errorf("target prefix must be a clean relative slash path")
	}
	return clean, nil
}

// normalizePortablePath rejects paths whose meaning can alias on supported
// Windows and case-insensitive filesystems. Target paths are repository
// relative slash paths, even when the eventual agent runs on another OS.
func normalizePortablePath(value string) (string, error) {
	if !utf8.ValidString(value) || len(value) > maxPathBytes {
		return "", fmt.Errorf("target path must be valid UTF-8 within the path size limit")
	}
	clean, err := normalizePrefix(value)
	if err != nil {
		return "", err
	}
	components := strings.Split(clean, "/")
	if len(components) > maxPathParts {
		return "", fmt.Errorf("target path contains too many components")
	}
	for _, component := range components {
		if len(component) > maxPathPartBytes {
			return "", fmt.Errorf("target path component exceeds the size limit")
		}
		if component == "" || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
			return "", fmt.Errorf("target path contains an empty or non-portable component")
		}
		for _, r := range component {
			if r < 0x20 || r == 0x7f || strings.ContainsRune(`<>:"|?*`, r) {
				return "", fmt.Errorf("target path contains a non-portable component")
			}
		}
		if strings.EqualFold(component, ".git") {
			return "", fmt.Errorf("target path may not enter Git metadata")
		}
		base := strings.ToUpper(strings.SplitN(component, ".", 2)[0])
		if isReservedWindowsName(base) {
			return "", fmt.Errorf("target path contains a reserved Windows component")
		}
	}
	return clean, nil
}

func isReservedWindowsName(base string) bool {
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CLOCK$":
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return true
	}
	if len([]rune(base)) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) {
		last := []rune(base)[3]
		if last == '¹' || last == '²' || last == '³' {
			return true
		}
	}
	return false
}

func underAllowedRoot(target string, roots []string) bool {
	if len(roots) == 0 || len(roots) > maxScopeRecords {
		return false
	}
	for _, root := range roots {
		clean, err := normalizePortablePath(root)
		if err == nil && within(clean, target) {
			return true
		}
	}
	return false
}

func within(root, name string) bool { return name == root || strings.HasPrefix(name, root+"/") }

func validDigest(value string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+64 {
		return false
	}
	value = strings.TrimPrefix(value, prefix)
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && strings.ToLower(value) == value
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func verificationCurrent(request string, verification *VerificationBinding, prior map[string]ArtifactBinding, observed map[string]ArtifactObservation) bool {
	if verification == nil || !verification.Passed || verification.RequestDigest != request || len(verification.Artifacts) != len(prior) {
		return false
	}
	verified := make(map[string]ArtifactBinding, len(verification.Artifacts))
	for _, item := range verification.Artifacts {
		if _, exists := verified[item.Path]; exists {
			return false
		}
		verified[item.Path] = item
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
