package agentrules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
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

// ProjectionPolicy explicitly binds one provider file to exact canonical
// Definitions and project-owned guidance. TargetPath may be omitted; when set,
// it must match the module-owned provider filename under TargetPrefix.
type ProjectionPolicy struct {
	ID            string                    `json:"id"`
	Provider      Provider                  `json:"provider"`
	TargetPath    string                    `json:"targetPath,omitempty"`
	DefinitionIDs []core.DefinitionIdentity `json:"definitionIds"`
	Guidance      string                    `json:"guidance"`
}

// TargetObservation is caller-supplied target evidence. Neither Render nor
// Propose reads or writes this path.
type TargetObservation struct {
	Path     string `json:"path"`
	Owner    string `json:"owner"`
	Exists   bool   `json:"exists"`
	Digest   string `json:"digest,omitempty"`
	Writable bool   `json:"writable"`
}

type Input struct {
	Provider           Provider
	Schemas            []core.Schema
	Definitions        []core.Definition // Definitions eligible for projection.
	RelatedDefinitions []core.Definition // Read-only context for resolving selected references.
	Policies           []ProjectionPolicy
	TargetPrefix       string // Canonical repository-relative slash path; "." is repository root.
	AllowedRoots       []string
	RequestDigest      string
	CanonicalAffected  []core.DefinitionIdentity
	Observed           []TargetObservation
	ReadOnly           bool
}

type CandidateFile struct {
	Path          string `json:"path"`
	Content       []byte `json:"content"`
	Digest        string `json:"digest"`
	RequestDigest string `json:"requestDigest"`
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

// Render builds a deterministic desired provider file from explicit canonical
// inputs. It does not require observed ownership or writability and performs no I/O.
func Render(input Input) (CandidateFile, error) {
	if strings.TrimSpace(input.RequestDigest) == "" {
		return CandidateFile{}, errors.New("request digest is required")
	}
	if input.Provider != Codex && input.Provider != Claude {
		return CandidateFile{}, errors.New("provider must be codex or claude")
	}
	model, policy, selected, err := canonicalSelection(input)
	if err != nil {
		return CandidateFile{}, err
	}
	filename := providerFilename(input.Provider)
	candidatePath, err := safeTargetPath(policy.TargetPath, input.TargetPrefix, input.AllowedRoots, filename)
	if err != nil {
		return CandidateFile{}, err
	}
	content, err := render(input.Provider, policy, model, selected)
	if err != nil {
		return CandidateFile{}, errors.New("validated canonical facts could not be rendered")
	}
	return CandidateFile{Path: candidatePath, Content: content, Digest: digestBytes(content), RequestDigest: input.RequestDigest}, nil
}

// Propose compares a rendered candidate with caller-supplied target evidence.
// It performs no filesystem or provider I/O.
func Propose(input Input) Result {
	result := Result{Decision: DecisionEscalate, RequestDigest: input.RequestDigest}
	file, err := Render(input)
	if err != nil {
		result.Reasons = []string{err.Error()}
		return result
	}
	observation, ok, reason := targetObservation(input.Observed, file.Path)
	if reason != "" {
		result.Reasons = []string{reason}
		return result
	}
	if !ok {
		result.Reasons = []string{"target ownership or current target state is unknown"}
		return result
	}
	if observation.Owner != Owner {
		result.Reasons = []string{"target is not explicitly owned by this module"}
		return result
	}
	if observation.Exists && !validDigest(observation.Digest) {
		result.Reasons = []string{"existing target has no valid observed digest"}
		return result
	}
	if !observation.Exists && observation.Digest != "" {
		result.Reasons = []string{"absent target must not have an observed digest"}
		return result
	}
	result.Files = []CandidateFile{file}
	if observation.Exists && observation.Digest == file.Digest {
		result.Decision = DecisionNoOp
		return result
	}
	if input.ReadOnly || !observation.Writable {
		result.Decision = DecisionEscalate
		result.Reasons = []string{"target needs an update but is read-only"}
		result.Files = nil
		return result
	}
	result.Decision = DecisionWork
	return result
}

func canonicalSelection(input Input) (core.Model, ProjectionPolicy, map[string]core.DefinitionIdentity, error) {
	modelDefinitions := make([]core.Definition, 0, len(input.Definitions)+len(input.RelatedDefinitions))
	modelDefinitions = append(modelDefinitions, input.Definitions...)
	modelDefinitions = append(modelDefinitions, input.RelatedDefinitions...)
	model, diagnostics := core.Compile(input.Schemas, modelDefinitions, "")
	if len(diagnostics) != 0 {
		return core.Model{}, ProjectionPolicy{}, nil, errors.New("canonical schemas or definitions do not compile")
	}
	policy, ok := providerPolicy(input.Policies, input.Provider)
	if !ok {
		return core.Model{}, ProjectionPolicy{}, nil, errors.New("provider projection guidance is missing or ambiguous")
	}
	if strings.TrimSpace(policy.ID) == "" || strings.TrimSpace(policy.Guidance) == "" || len(policy.DefinitionIDs) == 0 {
		return core.Model{}, ProjectionPolicy{}, nil, errors.New("provider projection policy requires an id, exact definitions, and explicit guidance")
	}
	projectable := make(map[string]core.Definition, len(input.Definitions))
	for _, definition := range input.Definitions {
		projectable[definition.Identity().Key()] = definition
	}
	selected := make(map[string]core.DefinitionIdentity, len(policy.DefinitionIDs))
	for _, identity := range policy.DefinitionIDs {
		key := identity.Key()
		if _, duplicate := selected[key]; duplicate {
			return core.Model{}, ProjectionPolicy{}, nil, errors.New("provider policy selects a canonical Definition more than once")
		}
		if _, exists := projectable[key]; !exists {
			return core.Model{}, ProjectionPolicy{}, nil, errors.New("provider policy selects a Definition outside the projectable input scope")
		}
		selected[key] = identity
	}
	affected := make(map[string]struct{}, len(input.CanonicalAffected))
	for _, identity := range input.CanonicalAffected {
		key := identity.Key()
		if _, duplicate := affected[key]; duplicate {
			return core.Model{}, ProjectionPolicy{}, nil, errors.New("affected canonical Definition is listed more than once")
		}
		if _, exists := projectable[key]; !exists {
			return core.Model{}, ProjectionPolicy{}, nil, errors.New("affected canonical Definition is absent from the projectable input scope")
		}
		if _, covered := selected[key]; !covered {
			return core.Model{}, ProjectionPolicy{}, nil, errors.New("affected canonical Definition has no explicit provider projection guidance")
		}
		affected[key] = struct{}{}
	}
	return model, policy, selected, nil
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

func safeTargetPath(policyPath, prefix string, roots []string, filename string) (string, error) {
	if !validRepoPath(prefix, true) || len(roots) == 0 {
		return "", errors.New("target prefix and allowed roots must be canonical repository-relative slash paths")
	}
	candidate := path.Join(prefix, filename)
	if policyPath != "" && (!validRepoPath(policyPath, false) || policyPath != candidate) {
		return "", errors.New("policy target must match the exact module-owned provider filename under TargetPrefix")
	}
	seen := make(map[string]string, len(roots))
	allowed := false
	for _, root := range roots {
		if !validRepoPath(root, true) {
			return "", errors.New("allowed roots must be canonical repository-relative slash paths")
		}
		for _, prior := range seen {
			if strings.EqualFold(prior, root) {
				if prior != root {
					return "", errors.New("allowed roots contain a case collision")
				}
				return "", errors.New("allowed roots contain a duplicate path")
			}
		}
		seen[root] = root
		allowed = allowed || within(root, candidate)
	}
	if !allowed {
		return "", errors.New("provider target is outside every allowed root")
	}
	return candidate, nil
}

func validRepoPath(value string, allowDot bool) bool {
	if value == "" || strings.ContainsAny(value, `\\:`) || strings.HasPrefix(value, "/") || path.Clean(value) != value {
		return false
	}
	if value == "." {
		return allowDot
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func within(root, candidate string) bool {
	if root == "." || root == candidate {
		return true
	}
	return strings.HasPrefix(candidate, root+"/")
}

func targetObservation(observations []TargetObservation, candidate string) (TargetObservation, bool, string) {
	var found TargetObservation
	count := 0
	candidateAlias := pathAlias(candidate)
	for _, observation := range observations {
		if !strings.EqualFold(pathAlias(observation.Path), candidateAlias) {
			continue
		}
		if !validRepoPath(observation.Path, false) {
			return TargetObservation{}, false, "observed target path is an unsafe alias of the provider file"
		}
		if observation.Path != candidate {
			return TargetObservation{}, false, "observed target path has a case or path alias collision"
		}
		found = observation
		count++
	}
	if count > 1 {
		return TargetObservation{}, false, "target inventory contains duplicate observations"
	}
	return found, count == 1, ""
}

func pathAlias(value string) string {
	portable := strings.ReplaceAll(value, "\\", "/")
	return path.Clean(portable)
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
