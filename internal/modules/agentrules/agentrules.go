package agentrules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

// Owner is the sole owner value accepted for provider rule files produced here.
const Owner = "markitect-agent-rules"

type Provider string

const (
	Codex  Provider = "codex"
	Claude Provider = "claude"
)

type Decision string

const (
	DecisionWork     Decision = "work"
	DecisionNoOp     Decision = "no-op"
	DecisionEscalate Decision = "escalate"
)

// ProjectionPolicy explicitly binds one provider file to the exact canonical
// Definitions and project-owned guidance that may appear in it.
type ProjectionPolicy struct {
	ID            string                    `json:"id"`
	Provider      Provider                  `json:"provider"`
	TargetPath    string                    `json:"targetPath"`
	DefinitionIDs []core.DefinitionIdentity `json:"definitionIds"`
	Guidance      string                    `json:"guidance"`
}

// TargetObservation is caller-supplied target evidence. Propose never reads or
// writes this path; the caller supplies owner, existence, digest, and writability.
type TargetObservation struct {
	Path     string `json:"path"`
	Owner    string `json:"owner"`
	Exists   bool   `json:"exists"`
	Digest   string `json:"digest,omitempty"`
	Writable bool   `json:"writable"`
}

type Input struct {
	Provider          Provider
	Schemas           []core.Schema
	Definitions       []core.Definition
	Policies          []ProjectionPolicy
	TargetPrefix      string
	AllowedRoots      []string
	RequestDigest     string
	CanonicalAffected []core.DefinitionIdentity
	Observed          []TargetObservation
	ReadOnly          bool
}

type CandidateFile struct {
	Path    string
	Content []byte
	Digest  string
}

type Result struct {
	Decision      Decision
	RequestDigest string
	Reasons       []string
	Files         []CandidateFile
}

type referenceFact struct {
	From     core.DefinitionIdentity `json:"from"`
	Property string                  `json:"property"`
	To       core.DefinitionIdentity `json:"to"`
	Source   core.Source             `json:"source"`
}

type semanticFacts struct {
	Schemas     []core.Schema     `json:"schemas"`
	Definitions []core.Definition `json:"definitions"`
	References  []referenceFact   `json:"references"`
}

// Propose creates a deterministic provider document from explicit, validated
// Core inputs. It performs no filesystem or provider I/O.
func Propose(input Input) Result {
	result := Result{Decision: DecisionEscalate, RequestDigest: input.RequestDigest}
	fail := func(reason string) Result {
		result.Reasons = []string{reason}
		result.Files = nil
		return result
	}
	if strings.TrimSpace(input.RequestDigest) == "" {
		return fail("request digest is required")
	}
	if input.Provider != Codex && input.Provider != Claude {
		return fail("provider must be codex or claude")
	}
	model, diagnostics := core.Compile(input.Schemas, input.Definitions, "")
	if len(diagnostics) != 0 {
		return fail("canonical schemas or definitions do not compile")
	}
	policy, ok := providerPolicy(input.Policies, input.Provider)
	if !ok {
		return fail("provider projection guidance is missing or ambiguous")
	}
	if strings.TrimSpace(policy.ID) == "" || strings.TrimSpace(policy.Guidance) == "" || len(policy.DefinitionIDs) == 0 {
		return fail("provider projection policy requires an id, exact definitions, and explicit guidance")
	}
	definitions := make(map[string]core.Definition, len(model.Definitions))
	for _, definition := range model.Definitions {
		definitions[definition.Identity().Key()] = definition
	}
	selected := make(map[string]core.DefinitionIdentity, len(policy.DefinitionIDs))
	for _, identity := range policy.DefinitionIDs {
		key := identity.Key()
		if _, duplicate := selected[key]; duplicate {
			return fail("provider policy selects a canonical Definition more than once")
		}
		if _, exists := definitions[key]; !exists {
			return fail("provider policy selects a missing canonical Definition")
		}
		selected[key] = identity
	}
	for _, affected := range input.CanonicalAffected {
		if _, exists := definitions[affected.Key()]; !exists {
			return fail("affected canonical Definition is absent from the supplied model")
		}
		if _, covered := selected[affected.Key()]; !covered {
			return fail("affected canonical Definition has no explicit provider projection guidance")
		}
	}

	filename := providerFilename(input.Provider)
	candidatePath, reason := safeTargetPath(policy.TargetPath, input.TargetPrefix, input.AllowedRoots, filename)
	if reason != "" {
		return fail(reason)
	}
	observation, ok := targetObservation(input.Observed, candidatePath)
	if !ok {
		return fail("target ownership or current target state is unknown")
	}
	if observation.Owner != Owner {
		return fail("target is not explicitly owned by this module")
	}
	if observation.Exists && !validDigest(observation.Digest) {
		return fail("existing target has no valid observed digest")
	}
	if !observation.Exists && observation.Digest != "" {
		return fail("absent target must not have an observed digest")
	}

	content, err := render(input.Provider, policy, model, selected)
	if err != nil {
		return fail("validated canonical facts could not be rendered")
	}
	digest := digestBytes(content)
	result.Files = []CandidateFile{{Path: candidatePath, Content: content, Digest: digest}}
	if observation.Exists && observation.Digest == digest {
		result.Decision = DecisionNoOp
		return result
	}
	if input.ReadOnly || !observation.Writable {
		return fail("target needs an update but is read-only")
	}
	result.Decision = DecisionWork
	return result
}

func providerPolicy(policies []ProjectionPolicy, provider Provider) (ProjectionPolicy, bool) {
	var found ProjectionPolicy
	count := 0
	for _, policy := range policies {
		if policy.Provider == provider {
			found = policy
			count++
		} else if policy.Provider != Codex && policy.Provider != Claude {
			return ProjectionPolicy{}, false
		}
	}
	return found, count == 1
}

func providerFilename(provider Provider) string {
	if provider == Codex {
		return "AGENTS.md"
	}
	return "CLAUDE.md"
}

func safeTargetPath(policyPath, prefix string, roots []string, filename string) (string, string) {
	if !filepath.IsAbs(prefix) || len(roots) == 0 {
		return "", "target prefix and allowed roots must be absolute paths"
	}
	cleanPrefix := filepath.Clean(prefix)
	expected := filepath.Join(cleanPrefix, filename)
	if !filepath.IsAbs(policyPath) || !samePath(filepath.Clean(policyPath), expected) {
		return "", "policy target must be the provider file directly under the declared target prefix"
	}
	for _, root := range roots {
		if !filepath.IsAbs(root) {
			continue
		}
		if within(filepath.Clean(root), expected) {
			return expected, ""
		}
	}
	return "", "provider target is outside every allowed root"
}

func targetObservation(observations []TargetObservation, path string) (TargetObservation, bool) {
	var found TargetObservation
	count := 0
	for _, observation := range observations {
		if samePath(filepath.Clean(observation.Path), path) {
			found = observation
			count++
		}
	}
	return found, count == 1
}

func within(root, path string) bool {
	if samePath(root, path) {
		return true
	}
	prefix := strings.TrimRight(root, `\\/`) + string(filepath.Separator)
	if filepath.Separator == '\\' {
		return strings.HasPrefix(strings.ToLower(path), strings.ToLower(prefix))
	}
	return strings.HasPrefix(path, prefix)
}

func samePath(left, right string) bool {
	if filepath.Separator == '\\' {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func digestBytes(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func render(provider Provider, policy ProjectionPolicy, model core.Model, selected map[string]core.DefinitionIdentity) ([]byte, error) {
	definitions := make([]core.Definition, 0, len(selected))
	for _, definition := range model.Definitions {
		if _, ok := selected[definition.Identity().Key()]; ok {
			definitions = append(definitions, definition)
		}
	}
	selectedKinds := make(map[string]map[string]struct{})
	for _, definition := range definitions {
		if selectedKinds[definition.APIVersion] == nil {
			selectedKinds[definition.APIVersion] = make(map[string]struct{})
		}
		selectedKinds[definition.APIVersion][definition.Kind] = struct{}{}
	}
	schemas := make([]core.Schema, 0, len(selectedKinds))
	for _, candidate := range model.Schemas {
		kindNames, ok := selectedKinds[candidate.APIVersion]
		if !ok {
			continue
		}
		projected := candidate
		projected.Kinds = make(map[string]core.Kind, len(kindNames))
		for kindName := range kindNames {
			projected.Kinds[kindName] = candidate.Kinds[kindName]
		}
		schemas = append(schemas, projected)
	}
	refs := make([]referenceFact, 0)
	for _, edge := range model.Edges {
		from, fromOK := definitionForKey(model.Definitions, edge.From)
		to, toOK := definitionForKey(model.Definitions, edge.To)
		if !fromOK || !toOK {
			return nil, fmt.Errorf("compiled reference endpoint is absent")
		}
		if _, ok := selected[from.Identity().Key()]; !ok {
			continue
		}
		refs = append(refs, referenceFact{From: from.Identity(), Property: edge.Property, To: to.Identity(), Source: edge.Source})
	}
	sort.Slice(refs, func(i, j int) bool {
		left, _ := json.Marshal(refs[i])
		right, _ := json.Marshal(refs[j])
		return string(left) < string(right)
	})
	facts := semanticFacts{Schemas: schemas, Definitions: definitions, References: refs}
	factsJSON, err := json.MarshalIndent(facts, "", "  ")
	if err != nil {
		return nil, err
	}
	switch provider {
	case Codex:
		return renderCodex(policy.Guidance, string(factsJSON)), nil
	case Claude:
		return renderClaude(policy.Guidance, string(factsJSON)), nil
	default:
		return nil, fmt.Errorf("unsupported provider")
	}
}

func definitionForKey(definitions []core.Definition, key string) (core.Definition, bool) {
	for _, definition := range definitions {
		if definition.Identity().Key() == key {
			return definition, true
		}
	}
	return core.Definition{}, false
}

func writeCodeBlock(b *strings.Builder, language, content string) {
	longest := 2
	for i := 0; i < len(content); {
		if content[i] != '`' {
			i++
			continue
		}
		start := i
		for i < len(content) && content[i] == '`' {
			i++
		}
		if i-start > longest {
			longest = i - start
		}
	}
	fence := strings.Repeat("`", longest+1)
	b.WriteString(fence)
	b.WriteString(language)
	b.WriteByte('\n')
	b.WriteString(content)
	b.WriteByte('\n')
	b.WriteString(fence)
	b.WriteString("\n")
}
