// Package migrate converts Konfyra's canonical Markdown mechanisms into
// Markitect resources without writing to the source tree.
package migrate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

const (
	projectPath = "markitect.yaml"
)

// Konfyra plans Markitect YAML resources from a captured Konfyra source tree.
// It never reads from or writes to the filesystem, and navigation links do not
// establish normative dependencies.
func Konfyra(snapshot *source.Snapshot) (map[string][]byte, error) {
	if snapshot == nil {
		return nil, errors.New("Konfyra migration snapshot is nil")
	}
	definitions, err := collectDefinitions(snapshot)
	if err != nil {
		return nil, err
	}
	project, err := konfyraProject(definitions, snapshot)
	if err != nil {
		return nil, err
	}

	outputs := make(map[string][]byte, len(definitions)+1)
	projectBytes, err := format.Encode(project)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", projectPath, err)
	}
	outputs[projectPath] = projectBytes
	for _, definition := range definitions {
		encoded, err := format.Encode(definition.resource)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", definition.sourcePath, err)
		}
		outputs[definition.resource.Path] = encoded
	}
	return outputs, nil
}

// ValidateKonfyraParity verifies that every migrated definition retains its
// source body and supported metadata, and that the plan contains no extras.
func ValidateKonfyraParity(snapshot *source.Snapshot, outputs map[string][]byte) error {
	if snapshot == nil {
		return errors.New("Konfyra migration snapshot is nil")
	}
	definitions, err := collectDefinitions(snapshot)
	if err != nil {
		return err
	}
	if len(outputs) != len(definitions)+1 {
		return fmt.Errorf("migration plan has %d outputs; want %d definitions plus %s", len(outputs), len(definitions), projectPath)
	}
	projectBytes, ok := outputs[projectPath]
	if !ok {
		return fmt.Errorf("migration plan is missing %s", projectPath)
	}
	project, err := format.Parse(projectPath, projectBytes)
	if err != nil {
		return err
	}
	if project.Kind != "Project" || project.Metadata.Name != "konfyra" || project.Spec.Profile != "konfyra" || len(project.Spec.Targets) != 0 || len(project.Spec.RuleAdapters) != 0 {
		return errors.New("Konfyra project must retain the staged profile with provider targets and rule adapters omitted")
	}
	for _, expected := range definitions {
		data, ok := outputs[expected.resource.Path]
		if !ok {
			return fmt.Errorf("migration plan is missing %s", expected.resource.Path)
		}
		actual, err := format.Parse(expected.resource.Path, data)
		if err != nil {
			return err
		}
		if actual.Kind != expected.resource.Kind || actual.Metadata != expected.resource.Metadata || actual.Spec.Text != expected.resource.Spec.Text || actual.Spec.Description != expected.resource.Spec.Description || !sameProviders(actual.Spec.Providers, expected.resource.Spec.Providers) {
			return fmt.Errorf("migration parity failed for %s", expected.sourcePath)
		}
	}
	return nil
}

// DependencyCandidate is a navigation reference worth reviewing as a possible
// Markitect dependency. The migration never promotes candidates to Uses.
type DependencyCandidate struct {
	Source  string `yaml:"source"`
	Target  string `yaml:"target"`
	Context string `yaml:"context"`
}

// CandidateDependencies reports explicit Skill and Agent navigation to direct
// workflow sources for human review. Links and quoted paths are evidence of
// navigation, not authority to add normative or execution dependencies.
func CandidateDependencies(snapshot *source.Snapshot) ([]DependencyCandidate, error) {
	if snapshot == nil {
		return nil, errors.New("Konfyra migration snapshot is nil")
	}
	definitions, err := collectDefinitions(snapshot)
	if err != nil {
		return nil, err
	}
	workflows := map[string]bool{}
	for _, item := range definitions {
		if item.resource.Kind == "Workflow" {
			workflows[item.sourcePath] = true
		}
	}
	var candidates []DependencyCandidate
	seen := map[string]bool{}
	for _, item := range definitions {
		if item.resource.Kind != "Skill" && item.resource.Kind != "Agent" {
			continue
		}
		for _, candidate := range workflowCandidates(item.sourcePath, item.resource.Spec.Text) {
			if !workflows[candidate.Target] {
				continue
			}
			key := candidate.Source + "\x00" + candidate.Target + "\x00" + candidate.Context
			if !seen[key] {
				seen[key] = true
				candidates = append(candidates, candidate)
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Target != b.Target {
			return a.Target < b.Target
		}
		return a.Context < b.Context
	})
	return candidates, nil
}

type definition struct {
	sourcePath string
	resource   *core.Resource
}

type mechanism struct {
	directory string
	kind      string
}

var mechanisms = []mechanism{
	{directory: "rules", kind: "Rule"},
	{directory: "workflows", kind: "Workflow"},
	{directory: "skills", kind: "Skill"},
	{directory: "agents", kind: "Agent"},
}

func collectDefinitions(snapshot *source.Snapshot) ([]definition, error) {
	productRoots := discoverProducts(snapshot)
	paths := make([]string, 0)
	for file := range snapshot.Files {
		if _, ok := directMechanism(file); ok {
			paths = append(paths, file)
		}
	}
	sort.Strings(paths)

	definitions := make([]definition, 0, len(paths))
	for _, file := range paths {
		kind, _ := directMechanism(file)
		if strings.EqualFold(path.Base(file), "README.md") {
			continue
		}
		data := snapshot.Files[file]
		name := strings.TrimSuffix(path.Base(file), path.Ext(file))
		desc, body := "", normalizeLineEndings(data)
		providers := core.Providers{}
		if kind == "Skill" {
			metadata, content, err := parseSkill(file, data)
			if err != nil {
				return nil, err
			}
			if metadata.Name != name {
				return nil, fmt.Errorf("%s: metadata name %q must match filename %q", file, metadata.Name, name)
			}
			desc, body = metadata.Description, normalizeLineEndings(content)
		} else if kind == "Agent" {
			metadata, content, err := parseAgent(file, data)
			if err != nil {
				return nil, err
			}
			if metadata.Name != name {
				return nil, fmt.Errorf("%s: metadata name %q must match filename %q", file, metadata.Name, name)
			}
			desc, body = metadata.Description, normalizeLineEndings(content)
			providers = metadata.providers()
		}
		namespace, err := namespaceFor(file, productRoots)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		resourcePath := strings.TrimSuffix(file, path.Ext(file)) + ".yaml"
		resource := &core.Resource{
			APIVersion: core.APIVersion,
			Kind:       kind,
			Metadata:   core.Metadata{Name: name, Namespace: namespace},
			Path:       resourcePath,
			Spec:       core.Spec{Text: string(body), Description: desc, Providers: providers},
		}
		definitions = append(definitions, definition{sourcePath: file, resource: resource})
	}
	return definitions, nil
}

func directMechanism(file string) (string, bool) {
	parts := strings.Split(file, "/")
	if len(parts) < 4 || parts[0] != "docs" || path.Ext(file) != ".md" {
		return "", false
	}
	for _, candidate := range mechanisms {
		if parts[len(parts)-2] == candidate.directory {
			return candidate.kind, true
		}
	}
	return "", false
}

func discoverProducts(snapshot *source.Snapshot) []string {
	const prefix = "docs/products/"
	seen := map[string]bool{}
	for file := range snapshot.Files {
		if !strings.HasPrefix(file, prefix) || !strings.HasSuffix(file, "/README.md") {
			continue
		}
		rest := strings.TrimPrefix(file, prefix)
		if !strings.Contains(rest, "/") {
			continue
		}
		product := strings.SplitN(rest, "/", 2)[0]
		if product != "agents" && product != "skills" {
			seen[product] = true
		}
	}
	products := make([]string, 0, len(seen))
	for product := range seen {
		products = append(products, product)
	}
	sort.Strings(products)
	return products
}

func namespaceFor(file string, products []string) (string, error) {
	if strings.HasPrefix(file, "docs/general/") {
		return "general", nil
	}
	if strings.HasPrefix(file, "docs/core/") {
		return "core", nil
	}
	if strings.HasPrefix(file, "docs/modules/") {
		return "modules", nil
	}
	const prefix = "docs/products/"
	if strings.HasPrefix(file, prefix) {
		rest := strings.TrimPrefix(file, prefix)
		product := strings.SplitN(rest, "/", 2)[0]
		for _, root := range products {
			if strings.EqualFold(product, root) {
				return strings.ToLower(root), nil
			}
		}
		return "products", nil
	}
	return "", errors.New("mechanism is outside a supported owner area")
}

type skillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func parseSkill(file string, data []byte) (skillMetadata, []byte, error) {
	var metadata skillMetadata
	frontmatter, body, err := splitFrontmatter(file, data)
	if err != nil {
		return metadata, nil, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(frontmatter))
	decoder.KnownFields(true)
	if err := decoder.Decode(&metadata); err != nil {
		return metadata, nil, fmt.Errorf("%s: invalid skill YAML front matter: %w", file, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return metadata, nil, fmt.Errorf("%s: skill front matter contains multiple YAML documents", file)
		}
		return metadata, nil, fmt.Errorf("%s: invalid skill YAML front matter: %w", file, err)
	}
	if metadata.Name == "" || metadata.Description == "" {
		return metadata, nil, fmt.Errorf("%s: skill front matter requires nonempty name and description", file)
	}
	return metadata, body, nil
}

type agentMetadata struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Codex       *codexMetadata  `json:"codex"`
	Claude      *claudeMetadata `json:"claude"`
}

type codexMetadata struct {
	Model                string `json:"model"`
	ModelReasoningEffort string `json:"model_reasoning_effort"`
	SandboxMode          string `json:"sandbox_mode"`
}

type claudeMetadata struct {
	Model           string   `json:"model"`
	Effort          string   `json:"effort"`
	PermissionMode  string   `json:"permissionMode"`
	Tools           []string `json:"tools"`
	DisallowedTools []string `json:"disallowedTools"`
	MaxTurns        int      `json:"maxTurns"`
}

func (m agentMetadata) providers() core.Providers {
	return core.Providers{
		Codex: &core.Provider{Model: m.Codex.Model, Effort: m.Codex.ModelReasoningEffort, Sandbox: m.Codex.SandboxMode},
		Claude: &core.Provider{
			Model: m.Claude.Model, Effort: m.Claude.Effort, PermissionMode: m.Claude.PermissionMode,
			Tools: append([]string(nil), m.Claude.Tools...), DisallowedTools: append([]string(nil), m.Claude.DisallowedTools...), MaxTurns: m.Claude.MaxTurns,
		},
	}
}

func parseAgent(file string, data []byte) (agentMetadata, []byte, error) {
	var metadata agentMetadata
	frontmatter, body, err := splitFrontmatter(file, data)
	if err != nil {
		return metadata, nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(frontmatter))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return metadata, nil, fmt.Errorf("%s: invalid agent JSON front matter: %w", file, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return metadata, nil, fmt.Errorf("%s: agent front matter contains trailing JSON", file)
		}
		return metadata, nil, fmt.Errorf("%s: invalid agent JSON front matter: %w", file, err)
	}
	if metadata.Name == "" || metadata.Description == "" || metadata.Codex == nil || metadata.Claude == nil {
		return metadata, nil, fmt.Errorf("%s: agent front matter requires name, description, codex and claude settings", file)
	}
	if metadata.Codex.Model == "" || metadata.Codex.ModelReasoningEffort == "" || metadata.Codex.SandboxMode == "" {
		return metadata, nil, fmt.Errorf("%s: agent codex settings require model, model_reasoning_effort and sandbox_mode", file)
	}
	if metadata.Claude.Model == "" || metadata.Claude.Effort == "" || metadata.Claude.PermissionMode == "" {
		return metadata, nil, fmt.Errorf("%s: agent claude settings require model, effort and permissionMode", file)
	}
	return metadata, body, nil
}

func splitFrontmatter(file string, data []byte) ([]byte, []byte, error) {
	position := 0
	lineNumber := 0
	frontStart, frontEnd, bodyStart := -1, -1, -1
	for position <= len(data) {
		lineNumber++
		relEnd := bytes.IndexByte(data[position:], '\n')
		lineEnd := len(data)
		next := len(data)
		if relEnd >= 0 {
			lineEnd = position + relEnd
			next = lineEnd + 1
		}
		line := data[position:lineEnd]
		line = bytes.TrimSuffix(line, []byte{'\r'})
		if lineNumber == 1 {
			if !bytes.Equal(line, []byte("---")) {
				return nil, nil, fmt.Errorf("%s: expected YAML or JSON front matter", file)
			}
			frontStart = next
		} else if frontStart >= 0 && bytes.Equal(line, []byte("---")) {
			frontEnd = position
			bodyStart = next
			return data[frontStart:frontEnd], data[bodyStart:], nil
		}
		if relEnd < 0 {
			break
		}
		position = next
	}
	return nil, nil, fmt.Errorf("%s: front matter has no closing delimiter", file)
}

func workflowCandidates(sourcePath, body string) []DependencyCandidate {
	seen := map[string]bool{}
	var candidates []DependencyCandidate
	for _, line := range strings.Split(body, "\n") {
		var paths []string
		for _, link := range markdownTargets.FindAllStringSubmatch(line, -1) {
			paths = append(paths, link[1])
		}
		for _, match := range quotedWorkflowPaths.FindAllStringSubmatch(line, -1) {
			paths = append(paths, match[1])
		}
		for _, rawPath := range paths {
			target, ok := normalizeWorkflowTarget(sourcePath, rawPath)
			if !ok {
				continue
			}
			context := strings.TrimSpace(line)
			key := target + "\x00" + context
			if seen[key] {
				continue
			}
			seen[key] = true
			candidates = append(candidates, DependencyCandidate{Source: sourcePath, Target: target, Context: context})
		}
	}
	return candidates
}

func normalizeWorkflowTarget(sourcePath, rawPath string) (string, bool) {
	target := strings.TrimSpace(rawPath)
	if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
		return "", false
	}
	if cut, _, ok := strings.Cut(target, "#"); ok {
		target = cut
	}
	if cut, _, ok := strings.Cut(target, "?"); ok {
		target = cut
	}
	if path.Ext(target) != ".md" {
		return "", false
	}
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
	} else if !strings.HasPrefix(target, "docs/") {
		target = path.Join(path.Dir(sourcePath), target)
	}
	target = path.Clean(target)
	return target, strings.HasPrefix(target, "docs/")
}

var (
	markdownTargets     = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	quotedWorkflowPaths = regexp.MustCompile("(?:`|\\\"|')([^`\\\"'\\s]+/workflows/[^`\\\"'\\s]+\\.md)(?:`|\\\"|')")
)

func normalizeLineEndings(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}

func konfyraProject(definitions []definition, snapshot *source.Snapshot) (*core.Resource, error) {
	var generalRules []core.Ref
	for _, item := range definitions {
		if item.resource.Kind == "Rule" && strings.HasPrefix(item.sourcePath, "docs/general/rules/") {
			generalRules = append(generalRules, core.Ref{Kind: "Rule", Name: item.resource.Metadata.Name, Namespace: "general"})
		}
	}
	sort.Slice(generalRules, func(i, j int) bool { return generalRules[i].Name < generalRules[j].Name })
	areas := []core.Area{
		{Name: "docs", Path: "docs", Imports: []string{"general"}, Rules: generalRules},
		{Name: "general", Path: "docs/general"},
		{Name: "core", Path: "docs/core", Imports: []string{"general"}},
		{Name: "modules", Path: "docs/modules", Imports: []string{"general", "core"}},
		{Name: "products", Path: "docs/products", Imports: []string{"general", "core", "modules"}},
	}
	for _, product := range discoverProducts(snapshot) {
		areas = append(areas, core.Area{Name: strings.ToLower(product), Path: "docs/products/" + product, Imports: []string{"products", "general", "core", "modules"}})
	}
	return &core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: "konfyra"},
		Path:       projectPath,
		Spec:       core.Spec{Profile: "konfyra", Areas: areas},
	}, nil
}

func sameProviders(a, b core.Providers) bool {
	return sameProvider(a.Codex, b.Codex) && sameProvider(a.Claude, b.Claude)
}

func sameProvider(a, b *core.Provider) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Model == b.Model && a.Effort == b.Effort && a.Sandbox == b.Sandbox && a.PermissionMode == b.PermissionMode && a.MaxTurns == b.MaxTurns && strings.Join(a.Tools, "\x00") == strings.Join(b.Tools, "\x00") && strings.Join(a.DisallowedTools, "\x00") == strings.Join(b.DisallowedTools, "\x00")
}
