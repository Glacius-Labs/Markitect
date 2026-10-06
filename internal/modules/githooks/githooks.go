// Package githooks implements the statically composed Git Hooks Projection Module.
package githooks

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

const (
	ProjectorID      = "git-hooks"
	TargetTechnology = "githooks"
	ArtifactMode     = "100755"
)

type Decision string

const (
	DecisionWork     Decision = "work"
	DecisionNoOp     Decision = "no-op"
	DecisionEscalate Decision = "escalate"
)

type NamedCheck struct {
	Name string
	Argv []string
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

type Input struct {
	Definitions       []core.Definition
	Schemas           []core.Schema
	Edges             []core.Edge
	Policies          []core.Definition
	TargetPrefix      string
	AllowedRoots      []string
	RequiredChecks    []NamedCheck
	RequestDigest     string
	InventoryComplete bool
	Previous          *PriorProjection
	ObservedArtifacts []ArtifactObservation
	Verification      *VerificationBinding
}

type CandidateFile struct {
	Path          string
	Content       []byte
	Digest        string
	RequestDigest string
	Mode          string
}
type Escalation struct {
	Code    string
	Message string
}
type Proposal struct {
	Decision                Decision
	Reasons                 []string
	EvidenceRefreshRequired bool
	Files                   []CandidateFile
	Escalations             []Escalation
}

// Render derives candidate hook bytes from selected canonical data and exact
// check vectors. It performs no observation or ownership decision; Host must
// separately bind target preimages and authorize materialization.
func Render(input Input) Proposal {
	if !validDigest(input.RequestDigest) {
		return escalate("request.digest.invalid", "a lowercase sha256 request digest is required")
	}
	target, _, err := safeTargetPath(input.TargetPrefix, input.AllowedRoots)
	if err != nil {
		return escalate("target.invalid", err.Error())
	}
	selected, err := normalizedScope(input)
	if err != nil {
		return escalate("scope.invalid", err.Error())
	}
	policies, err := selectedPolicies(input.Policies, selected)
	if err != nil {
		return escalate("policy.invalid", err.Error())
	}
	checks, err := validateChecks(input.RequiredChecks)
	if err != nil {
		return escalate("checks.invalid", err.Error())
	}
	candidate, err := render(target, input.RequestDigest, selected, input.Edges, policies, checks)
	if err != nil {
		return escalate("output.invalid", "validated Git Hooks facts could not be rendered")
	}
	return Proposal{Decision: DecisionWork, Files: []CandidateFile{candidate}}
}

// Propose derives work/no-op/escalation from the rendered candidate and supplied
// target evidence. It performs no repository/provider I/O or hook installation.
func Propose(input Input) Proposal {
	rendered := Render(input)
	if rendered.Decision == DecisionEscalate {
		return rendered
	}
	candidate := rendered.Files[0]
	target, prefix := candidate.Path, path.Dir(candidate.Path)
	if !input.InventoryComplete {
		return escalate("target.inventory.incomplete", "the selected target prefix was not completely observed")
	}
	if input.Previous != nil && !validDigest(input.Previous.RequestDigest) {
		return escalate("record.request.digest.invalid", "active record contains an invalid request digest")
	}
	observed, found, err := observeTarget(input.ObservedArtifacts, target, prefix)
	if err != nil {
		return escalate("target.inventory.invalid", err.Error())
	}
	previous, owned, err := priorTarget(input.Previous, target, prefix)
	if err != nil {
		return escalate("record.artifact.invalid", err.Error())
	}
	if found && !owned {
		return escalate("artifact.unowned", "the target exists without an active Module ownership binding")
	}
	if !found || !owned || input.Previous == nil || !input.Previous.Complete ||
		observed.Mode != ArtifactMode || previous.Digest != digest(observed.Bytes) ||
		!bytes.Equal(observed.Bytes, candidate.Content) {
		return Proposal{Decision: DecisionWork, Reasons: []string{"projection-required"}, Files: []CandidateFile{candidate}}
	}
	refresh := !verificationCurrent(input.RequestDigest, input.Verification, previous)
	reason := "representation-current"
	if refresh {
		reason = "evidence-refresh-required"
	}
	return Proposal{Decision: DecisionNoOp, Reasons: []string{reason}, EvidenceRefreshRequired: refresh, Files: []CandidateFile{candidate}}
}

type sourceKind struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}
type selectedPolicy struct {
	Identity core.DefinitionIdentity
	Source   sourceKind
	Guidance string
}

func normalizedScope(input Input) (map[string]core.Definition, error) {
	if len(input.Definitions) == 0 {
		return nil, errors.New("selected canonical scope contains no Definitions")
	}
	model, diagnostics := core.Compile(input.Schemas, input.Definitions, "")
	if len(diagnostics) != 0 {
		return nil, errors.New("selected Schemas and Definitions do not compile")
	}
	definitions := make(map[string]core.Definition, len(model.Definitions))
	for _, definition := range model.Definitions {
		definitions[definition.Identity().Key()] = definition
	}
	compiled := make(map[string]bool, len(model.Edges))
	for _, edge := range model.Edges {
		compiled[edgeKey(edge)] = true
	}
	seen := make(map[string]bool, len(input.Edges))
	for _, edge := range input.Edges {
		key := edgeKey(edge)
		if seen[key] {
			return nil, errors.New("selected Relations contain a duplicate edge")
		}
		if !compiled[key] {
			return nil, errors.New("selected Relation is absent from the normalized Definition scope")
		}
		seen[key] = true
	}
	return definitions, nil
}

func selectedPolicies(policies []core.Definition, selected map[string]core.Definition) ([]selectedPolicy, error) {
	if len(policies) == 0 {
		return nil, errors.New("selected scope has no ProjectionPolicy")
	}
	kinds := make(map[string]bool)
	for _, definition := range selected {
		kinds[(core.KindIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind}).Key()] = true
	}
	covered, ids := make(map[string]bool), make(map[string]bool)
	result := make([]selectedPolicy, 0, len(policies))
	for _, policy := range policies {
		identity := policy.Identity()
		if policy.Kind != "ProjectionPolicy" {
			return nil, fmt.Errorf("selected policy %s is not a ProjectionPolicy", identity.Key())
		}
		if identity.APIVersion == "" || identity.Namespace == "" || identity.Name == "" || ids[identity.Key()] {
			return nil, errors.New("selected ProjectionPolicy identity is incomplete or duplicated")
		}
		ids[identity.Key()] = true
		source, ok := policy.Spec["sourceKind"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("ProjectionPolicy %s has no sourceKind object", identity.Key())
		}
		api, apiOK := source["apiVersion"].(string)
		kind, kindOK := source["kind"].(string)
		target, targetOK := policy.Spec["targetTechnology"].(string)
		guidance, guidanceOK := policy.Spec["guidance"].(string)
		if !apiOK || !kindOK || !targetOK || !guidanceOK || strings.TrimSpace(api) == "" ||
			strings.TrimSpace(kind) == "" || target != TargetTechnology || strings.TrimSpace(guidance) == "" {
			return nil, fmt.Errorf("ProjectionPolicy %s has invalid sourceKind, targetTechnology, or guidance", identity.Key())
		}
		kindKey := (core.KindIdentity{APIVersion: api, Kind: kind}).Key()
		if !kinds[kindKey] {
			return nil, fmt.Errorf("ProjectionPolicy %s selects a Kind outside the bounded Definition scope", identity.Key())
		}
		if covered[kindKey] {
			return nil, fmt.Errorf("multiple Git Hooks policies select Kind %s", kindKey)
		}
		covered[kindKey] = true
		result = append(result, selectedPolicy{Identity: identity, Source: sourceKind{APIVersion: api, Kind: kind}, Guidance: guidance})
	}
	for kind := range kinds {
		if !covered[kind] {
			return nil, fmt.Errorf("selected Kind %s has no Git Hooks ProjectionPolicy", kind)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left := (core.KindIdentity{APIVersion: result[i].Source.APIVersion, Kind: result[i].Source.Kind}).Key()
		right := (core.KindIdentity{APIVersion: result[j].Source.APIVersion, Kind: result[j].Source.Kind}).Key()
		if left != right {
			return left < right
		}
		return result[i].Identity.Key() < result[j].Identity.Key()
	})
	return result, nil
}

func validateChecks(checks []NamedCheck) ([]NamedCheck, error) {
	if len(checks) == 0 {
		return nil, errors.New("at least one exact required Project check argv vector is required")
	}
	seen := make(map[string]bool, len(checks))
	result := make([]NamedCheck, len(checks))
	for i, check := range checks {
		if strings.TrimSpace(check.Name) == "" || seen[check.Name] {
			return nil, errors.New("required Project check names must be nonempty and unique")
		}
		if len(check.Argv) == 0 || check.Argv[0] == "" {
			return nil, fmt.Errorf("required Project check %q has no executable argv[0]", check.Name)
		}
		for _, argument := range check.Argv {
			if strings.IndexByte(argument, 0) >= 0 {
				return nil, fmt.Errorf("required Project check %q contains a NUL argument", check.Name)
			}
		}
		seen[check.Name] = true
		result[i] = NamedCheck{Name: check.Name, Argv: append([]string(nil), check.Argv...)}
	}
	return result, nil
}

func render(target, request string, definitions map[string]core.Definition, edges []core.Edge, policies []selectedPolicy, checks []NamedCheck) (CandidateFile, error) {
	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("# Generated by the Markitect Git Hooks Projection Module.\n")
	script.WriteString("# Projection guidance is descriptive; configured checks below provide enforcement.\n")
	for _, policy := range policies {
		record := struct {
			Identity         core.DefinitionIdentity `json:"identity"`
			SourceKind       sourceKind              `json:"sourceKind"`
			TargetTechnology string                  `json:"targetTechnology"`
			Guidance         string                  `json:"guidance"`
		}{policy.Identity, policy.Source, TargetTechnology, policy.Guidance}
		line, err := json.Marshal(record)
		if err != nil {
			return CandidateFile{}, err
		}
		script.WriteString("# ProjectionPolicy: ")
		script.Write(line)
		script.WriteByte('\n')
	}
	identities := make([]core.DefinitionIdentity, 0, len(definitions))
	for _, definition := range definitions {
		identities = append(identities, definition.Identity())
	}
	sort.Slice(identities, func(i, j int) bool { return identities[i].Key() < identities[j].Key() })
	for _, identity := range identities {
		line, err := json.Marshal(identity)
		if err != nil {
			return CandidateFile{}, err
		}
		script.WriteString("# selected-definition: ")
		script.Write(line)
		script.WriteByte('\n')
		// Canonical values remain inert, recoverable evidence in the target. Only
		// the explicitly configured check argv below provides execution semantics.
		definition := definitions[identity.Key()]
		definitionLine, err := json.Marshal(struct {
			Identity core.DefinitionIdentity `json:"identity"`
			Purpose  string                  `json:"purpose"`
			Spec     map[string]any          `json:"spec"`
		}{identity, definition.Purpose, definition.Spec})
		if err != nil {
			return CandidateFile{}, err
		}
		script.WriteString("# canonical-definition: ")
		script.Write(definitionLine)
		script.WriteByte('\n')
	}
	orderedEdges := append([]core.Edge(nil), edges...)
	sort.Slice(orderedEdges, func(i, j int) bool { return edgeKey(orderedEdges[i]) < edgeKey(orderedEdges[j]) })
	for _, edge := range orderedEdges {
		line, err := json.Marshal(edge)
		if err != nil {
			return CandidateFile{}, err
		}
		script.WriteString("# selected-relation: ")
		script.Write(line)
		script.WriteByte('\n')
	}
	script.WriteString("set -eu\n\n")
	for _, check := range checks {
		record, err := json.Marshal(check)
		if err != nil {
			return CandidateFile{}, err
		}
		script.WriteString("# Required Project check: ")
		script.Write(record)
		script.WriteByte('\n')
		for i, argument := range check.Argv {
			if i != 0 {
				script.WriteByte(' ')
			}
			script.WriteString(shellQuote(argument))
		}
		script.WriteByte('\n')
	}
	content := []byte(script.String())
	return CandidateFile{Path: target, Content: content, Digest: digest(content), RequestDigest: request, Mode: ArtifactMode}, nil
}

func safeTargetPath(prefix string, roots []string) (string, string, error) {
	cleanPrefix := strings.TrimSuffix(prefix, "/")
	if len(prefix) > 1024 || cleanPrefix == "" || !validRepoPath(cleanPrefix, true) || len(roots) == 0 {
		return "", "", errors.New("target prefix and allowed roots must be clean repository-relative slash paths")
	}
	target := path.Join(cleanPrefix, "pre-commit")
	allowed, seen := false, make(map[string]bool, len(roots))
	for _, root := range roots {
		if !validRepoPath(root, true) {
			return "", "", errors.New("allowed roots must be clean repository-relative slash paths")
		}
		if seen[root] {
			return "", "", errors.New("allowed roots contain a duplicate path")
		}
		seen[root] = true
		allowed = allowed || within(root, target)
	}
	if !allowed {
		return "", "", errors.New("Git Hooks target is outside every allowed root")
	}
	return target, cleanPrefix, nil
}

func validRepoPath(value string, allowDot bool) bool {
	if value == "" || len(value) > 1024 || !utf8.ValidString(value) ||
		strings.TrimSpace(value) != value || strings.Contains(value, "\\") ||
		strings.HasPrefix(value, "/") || strings.ContainsAny(value, `:*?<>|"`) || path.Clean(value) != value {
		return false
	}
	if value == "." {
		return allowDot
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." ||
			strings.TrimSpace(component) != component || strings.HasSuffix(component, ".") ||
			strings.EqualFold(component, ".git") || reservedPathComponent(component) {
			return false
		}
		for _, character := range component {
			if character < 32 {
				return false
			}
		}
	}
	return true
}

func reservedPathComponent(value string) bool {
	base := strings.ToUpper(strings.SplitN(value, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9'
}
func within(root, target string) bool {
	return root == "." || root == target || strings.HasPrefix(target, root+"/")
}

func observeTarget(observations []ArtifactObservation, target, prefix string) (ArtifactObservation, bool, error) {
	if len(observations) > core.MaxDefinitions {
		return ArtifactObservation{}, false, errors.New("target inventory exceeds the bounded artifact count")
	}
	var found ArtifactObservation
	count := 0
	for _, item := range observations {
		if !validRepoPath(item.Path, false) || item.Mode == "" {
			return ArtifactObservation{}, false, errors.New("target inventory contains an invalid path or missing mode")
		}
		if item.Path != target && pathOverlap(item.Path, target) {
			return ArtifactObservation{}, false, errors.New("target inventory contains a path alias or overlap with the Module-owned target")
		}
		if item.Path != target && prefixCaseAlias(item.Path, prefix) {
			return ArtifactObservation{}, false, errors.New("target inventory contains a case alias of the selected target prefix")
		}
		if !within(prefix, item.Path) {
			return ArtifactObservation{}, false, errors.New("target inventory contains a path outside the selected target prefix")
		}
		if item.Path == target {
			found = item
			count++
		}
	}
	if count > 1 {
		return ArtifactObservation{}, false, errors.New("target inventory contains a duplicate Module-owned path")
	}
	return found, count == 1, nil
}

func pathOverlap(left, right string) bool {
	a, b := strings.Split(left, "/"), strings.Split(right, "/")
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}

func prefixCaseAlias(candidate, prefix string) bool {
	candidateParts, prefixParts := strings.Split(candidate, "/"), strings.Split(prefix, "/")
	limit := len(candidateParts)
	if len(prefixParts) < limit {
		limit = len(prefixParts)
	}
	for i := 0; i < limit; i++ {
		if !strings.EqualFold(candidateParts[i], prefixParts[i]) {
			return false
		}
		if candidateParts[i] != prefixParts[i] {
			return true
		}
	}
	return false
}

func priorTarget(previous *PriorProjection, target, prefix string) (ArtifactBinding, bool, error) {
	if previous == nil {
		return ArtifactBinding{}, false, nil
	}
	if len(previous.RetiredArtifacts) > 0 || len(previous.Artifacts) > 1 {
		return ArtifactBinding{}, false, errors.New("active record has retired or excess Module artifacts requiring review")
	}
	if len(previous.Artifacts) == 0 {
		return ArtifactBinding{}, false, nil
	}
	binding := previous.Artifacts[0]
	if !validRepoPath(binding.Path, false) || !within(prefix, binding.Path) || binding.Path != target ||
		!validDigest(binding.Digest) || binding.Mode != ArtifactMode {
		return ArtifactBinding{}, false, errors.New("active record does not contain one valid Git Hooks artifact binding")
	}
	return binding, true, nil
}

func verificationCurrent(request string, verification *VerificationBinding, previous ArtifactBinding) bool {
	if verification == nil || !verification.Passed || verification.RequestDigest != request || len(verification.Artifacts) != 1 {
		return false
	}
	proof := verification.Artifacts[0]
	return proof.Path == previous.Path && proof.Digest == previous.Digest && proof.Mode == ArtifactMode
}
func edgeKey(edge core.Edge) string { data, _ := json.Marshal(edge); return string(data) }
func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func validDigest(value string) bool {
	prefix := "sha256:"
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, prefix))
	return err == nil && hex.EncodeToString(decoded) == strings.TrimPrefix(value, prefix)
}
func escalate(code, message string) Proposal {
	return Proposal{Decision: DecisionEscalate, Reasons: []string{code}, Escalations: []Escalation{{Code: code, Message: message}}}
}
