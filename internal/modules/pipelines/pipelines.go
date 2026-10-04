package pipelines

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

const (
	ConfigVersion          = "markitect.example.org/pipelines-check/v1alpha1"
	ReportVersion          = "markitect.example.org/pipelines-report/v1alpha1"
	StatusPassed           = "passed"
	StatusFindings         = "findings"
	StatusNotConfigured    = "not-configured"
	maxConfigBytes         = 1 << 20
	maxPipelineBytes       = 2 << 20
	maxConfiguredPipelines = 128
	maxExpectedChecks      = 128
)

type Config struct {
	APIVersion string     `yaml:"apiVersion" json:"apiVersion"`
	Pipelines  []Pipeline `yaml:"pipelines" json:"pipelines"`
}

type Pipeline struct {
	Name           string          `yaml:"name" json:"name"`
	Provider       string          `yaml:"provider" json:"provider"`
	Path           string          `yaml:"path" json:"path"`
	Digest         string          `yaml:"digest,omitempty" json:"digest,omitempty"`
	Owner          string          `yaml:"owner" json:"owner"`
	ExpectedChecks []ExpectedCheck `yaml:"expectedChecks,omitempty" json:"expectedChecks,omitempty"`
}

type ExpectedCheck struct {
	Name     string `yaml:"name" json:"name"`
	YAMLPath string `yaml:"yamlPath" json:"yamlPath"`
}

// CheckFact is an explicit Host-supplied check identity and the exact scalar
// text expected at one configured YAML location. It is not a parsed command.
type CheckFact struct {
	Name      string
	Reference string
}

// Input carries only configured pipeline bytes, explicit ownership/check facts,
// and the normalized semantic model used to resolve managed owner identities.
type Input struct {
	Model      core.SemanticModel
	Config     Config
	Artifacts  map[string][]byte
	Ownership  map[string][]string
	CheckFacts []CheckFact
}

type Finding struct {
	Code     string `yaml:"code"`
	Severity string `yaml:"severity"`
	Pipeline string `yaml:"pipeline,omitempty"`
	Provider string `yaml:"provider,omitempty"`
	Path     string `yaml:"path,omitempty"`
	Owner    string `yaml:"owner,omitempty"`
	Check    string `yaml:"check,omitempty"`
	YAMLPath string `yaml:"yamlPath,omitempty"`
	Message  string `yaml:"message"`
}

type Report struct {
	APIVersion             string    `yaml:"apiVersion"`
	Status                 string    `yaml:"status"`
	SnapshotID             string    `yaml:"snapshotID,omitempty"`
	SnapshotDigest         string    `yaml:"snapshotDigest"`
	ModelDigest            string    `yaml:"modelDigest"`
	ConfigDigest           string    `yaml:"configDigest"`
	PipelinesChecked       int       `yaml:"pipelinesChecked"`
	CheckReferencesChecked int       `yaml:"checkReferencesChecked"`
	Findings               []Finding `yaml:"findings"`
}

// DecodeConfig decodes the module-owned, closed pipeline-check configuration.
func DecodeConfig(data []byte) (Config, error) {
	if len(data) == 0 || len(data) > maxConfigBytes {
		return Config{}, fmt.Errorf("pipelines config must contain 1..%d bytes", maxConfigBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode pipelines config: %w", err)
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Config{}, errors.New("pipelines config must contain exactly one YAML document")
		}
		return Config{}, fmt.Errorf("decode trailing pipelines config: %w", err)
	}
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

// Check checks only configured pipeline paths and optional exact YAML scalar
// references. It does not interpret provider execution or shell semantics.
func Check(input Input) (Report, error) {
	if err := validateModel(input.Model); err != nil {
		return Report{}, err
	}
	if err := validateConfig(input.Config); err != nil {
		return Report{}, err
	}
	facts := make(map[string]string, len(input.CheckFacts))
	for _, fact := range input.CheckFacts {
		if !validName(fact.Name) || strings.TrimSpace(fact.Reference) == "" {
			return Report{}, fmt.Errorf("pipeline check facts require a name and non-empty literal reference")
		}
		if _, duplicate := facts[fact.Name]; duplicate {
			return Report{}, fmt.Errorf("duplicate pipeline check fact %q", fact.Name)
		}
		facts[fact.Name] = fact.Reference
	}
	report := Report{
		APIVersion: ReportVersion, Status: StatusPassed,
		SnapshotID: input.Model.Snapshot.ID, SnapshotDigest: input.Model.Snapshot.Digest,
		ModelDigest: input.Model.ModelDigest, ConfigDigest: configDigest(input.Config), Findings: []Finding{},
	}
	if len(input.Config.Pipelines) == 0 {
		report.Status = StatusNotConfigured
		report.Findings = append(report.Findings, Finding{
			Code: "pipelines.not-configured", Severity: "info",
			Message: "No pipeline paths are configured; no pipeline artifact or check linkage was verified.",
		})
		return report, nil
	}

	owners := modelOwners(input.Model)
	pipelines := append([]Pipeline(nil), input.Config.Pipelines...)
	sort.Slice(pipelines, func(i, j int) bool { return pipelines[i].Path < pipelines[j].Path })
	for _, pipeline := range pipelines {
		report.PipelinesChecked++
		data, exists := input.Artifacts[pipeline.Path]
		if !exists {
			report.Findings = append(report.Findings, pipelineFinding(pipeline, "pipelines.artifact.missing", "", "", "configured pipeline bytes were not supplied"))
		} else if len(data) > maxPipelineBytes {
			return Report{}, fmt.Errorf("configured pipeline %q exceeds the %d-byte input limit", pipeline.Name, maxPipelineBytes)
		} else if digest(data) != pipeline.Digest {
			report.Findings = append(report.Findings, pipelineFinding(pipeline, "pipelines.digest.mismatch", "", "", "pipeline bytes do not match the configured SHA-256 digest"))
		}
		if !owners[pipeline.Owner] {
			report.Findings = append(report.Findings, pipelineFinding(pipeline, "pipelines.owner.resource-missing", "", "", "configured owner is not a resource identity in the supplied semantic model"))
		}
		if !contains(input.Ownership[pipeline.Path], pipeline.Owner) {
			report.Findings = append(report.Findings, pipelineFinding(pipeline, "pipelines.owner.link-missing", "", "", "supplied ownership facts do not link this path to the configured owner"))
		}
		if !exists || len(pipeline.ExpectedChecks) == 0 {
			continue
		}
		root, err := decodeSingleYAML(data)
		if err != nil {
			report.Findings = append(report.Findings, pipelineFinding(pipeline, "pipelines.yaml.invalid", "", "", "configured pipeline is not one valid YAML document"))
			continue
		}
		for _, expected := range pipeline.ExpectedChecks {
			report.CheckReferencesChecked++
			literal, ok := facts[expected.Name]
			if !ok {
				report.Findings = append(report.Findings, pipelineFinding(pipeline, "pipelines.check-fact.missing", expected.Name, expected.YAMLPath, "no explicit Host-supplied check fact matches this configured reference"))
				continue
			}
			actual, err := scalarAtYAMLPointer(root, expected.YAMLPath)
			if err != nil {
				report.Findings = append(report.Findings, pipelineFinding(pipeline, "pipelines.check-reference.unavailable", expected.Name, expected.YAMLPath, err.Error()))
				continue
			}
			if actual != literal {
				report.Findings = append(report.Findings, pipelineFinding(pipeline, "pipelines.check-reference.mismatch", expected.Name, expected.YAMLPath, "YAML scalar differs from the exact Host-supplied literal reference"))
			}
		}
	}
	sortFindings(report.Findings)
	if hasErrors(report.Findings) {
		report.Status = StatusFindings
	}
	return report, nil
}

func validateConfig(config Config) error {
	if config.APIVersion != ConfigVersion {
		return fmt.Errorf("unsupported pipelines config apiVersion %q", config.APIVersion)
	}
	if len(config.Pipelines) > maxConfiguredPipelines {
		return fmt.Errorf("pipelines config exceeds %d configured paths", maxConfiguredPipelines)
	}
	seenNames := map[string]bool{}
	seenPaths := make([]string, 0, len(config.Pipelines))
	totalChecks := 0
	for _, pipeline := range config.Pipelines {
		if !validName(pipeline.Name) || seenNames[pipeline.Name] {
			return fmt.Errorf("pipeline names must be non-empty and unique: %q", pipeline.Name)
		}
		seenNames[pipeline.Name] = true
		switch pipeline.Provider {
		case "github-actions":
			if !strings.HasPrefix(pipeline.Path, ".github/workflows/") || !validYAMLExtension(pipeline.Path) {
				return fmt.Errorf("GitHub Actions pipeline %q must name one YAML file under .github/workflows/", pipeline.Name)
			}
		case "azure-devops":
			if !validYAMLExtension(pipeline.Path) {
				return fmt.Errorf("Azure DevOps pipeline %q must name one YAML file", pipeline.Name)
			}
		default:
			return fmt.Errorf("pipeline %q has unsupported provider %q", pipeline.Name, pipeline.Provider)
		}
		if !validExactPath(pipeline.Path) {
			return fmt.Errorf("pipeline path must be safe and exact: %q", pipeline.Path)
		}
		for _, previous := range seenPaths {
			if strings.EqualFold(previous, pipeline.Path) {
				return fmt.Errorf("pipeline paths must not collide under portable case folding: %q", pipeline.Path)
			}
		}
		seenPaths = append(seenPaths, pipeline.Path)
		if !validDigest(pipeline.Digest) {
			return fmt.Errorf("pipeline %q digest must be 64 lowercase hexadecimal SHA-256 characters", pipeline.Name)
		}
		if !validName(pipeline.Owner) {
			return fmt.Errorf("pipeline %q must name an expected managed owner identity", pipeline.Name)
		}
		seenChecks := map[string]bool{}
		for _, check := range pipeline.ExpectedChecks {
			totalChecks++
			if !validName(check.Name) || seenChecks[check.Name] {
				return fmt.Errorf("pipeline %q expected check names must be non-empty and unique: %q", pipeline.Name, check.Name)
			}
			seenChecks[check.Name] = true
			if !validYAMLPointer(check.YAMLPath) {
				return fmt.Errorf("pipeline %q check %q must use an exact YAML pointer", pipeline.Name, check.Name)
			}
		}
	}
	if totalChecks > maxExpectedChecks {
		return fmt.Errorf("pipelines config exceeds %d expected check references", maxExpectedChecks)
	}
	return nil
}

func configDigest(config Config) string {
	normalized := config
	normalized.Pipelines = append([]Pipeline(nil), config.Pipelines...)
	for i := range normalized.Pipelines {
		normalized.Pipelines[i].ExpectedChecks = append([]ExpectedCheck(nil), normalized.Pipelines[i].ExpectedChecks...)
		sort.Slice(normalized.Pipelines[i].ExpectedChecks, func(a, b int) bool {
			left, right := normalized.Pipelines[i].ExpectedChecks[a], normalized.Pipelines[i].ExpectedChecks[b]
			if left.Name != right.Name {
				return left.Name < right.Name
			}
			return left.YAMLPath < right.YAMLPath
		})
	}
	sort.Slice(normalized.Pipelines, func(i, j int) bool {
		if normalized.Pipelines[i].Path != normalized.Pipelines[j].Path {
			return normalized.Pipelines[i].Path < normalized.Pipelines[j].Path
		}
		return normalized.Pipelines[i].Name < normalized.Pipelines[j].Name
	})
	canonical, _ := json.Marshal(normalized)
	return digest(canonical)
}

func validateModel(model core.SemanticModel) error {
	if model.APIVersion != core.SemanticModelVersion || model.StructuralStatus != "passed" || strings.TrimSpace(model.ModelDigest) == "" || strings.TrimSpace(model.Snapshot.Digest) == "" {
		return errors.New("pipelines check requires a structurally valid semantic model with fixed snapshot and model digests")
	}
	seen := map[string]bool{}
	for _, resource := range model.Resources {
		key := resource.Identity.Key
		if strings.TrimSpace(key) == "" || seen[key] {
			return fmt.Errorf("semantic model has a blank or duplicate resource identity %q", key)
		}
		seen[key] = true
	}
	return nil
}

func modelOwners(model core.SemanticModel) map[string]bool {
	result := make(map[string]bool, len(model.Resources))
	for _, resource := range model.Resources {
		result[resource.Identity.Key] = true
	}
	return result
}

func validExactPath(value string) bool {
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\\\x00:*?[]") || strings.HasPrefix(value, "/") || path.Clean(value) != value {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func validYAMLExtension(value string) bool {
	return strings.HasSuffix(strings.ToLower(value), ".yaml") || strings.HasSuffix(strings.ToLower(value), ".yml")
}

func validYAMLPointer(value string) bool {
	if !strings.HasPrefix(value, "/") || value == "/" {
		return false
	}
	parts := strings.Split(value[1:], "/")
	for _, part := range parts {
		if part == "" {
			return false
		}
		for i := 0; i < len(part); i++ {
			if part[i] == '~' {
				if i+1 >= len(part) || (part[i+1] != '0' && part[i+1] != '1') {
					return false
				}
				i++
			}
		}
	}
	return true
}

func decodeSingleYAML(data []byte) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return nil, errors.New("invalid YAML document")
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("multiple YAML documents are unsupported")
		}
		return nil, err
	}
	return document.Content[0], nil
}

func scalarAtYAMLPointer(root *yaml.Node, pointer string) (string, error) {
	if !validYAMLPointer(pointer) {
		return "", errors.New("configured YAML pointer is invalid")
	}
	node := root
	for _, encoded := range strings.Split(pointer[1:], "/") {
		segment := strings.ReplaceAll(strings.ReplaceAll(encoded, "~1", "/"), "~0", "~")
		switch node.Kind {
		case yaml.MappingNode:
			var found *yaml.Node
			matches := 0
			for i := 0; i+1 < len(node.Content); i += 2 {
				key := node.Content[i]
				if key.Kind == yaml.ScalarNode && key.Tag == "!!str" && key.Value == segment {
					found = node.Content[i+1]
					matches++
				}
			}
			if matches != 1 {
				return "", fmt.Errorf("YAML pointer segment %q is missing or ambiguous", segment)
			}
			node = found
		case yaml.SequenceNode:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || strconv.Itoa(index) != segment || index >= len(node.Content) {
				return "", fmt.Errorf("YAML pointer sequence index %q is unavailable", segment)
			}
			node = node.Content[index]
		default:
			return "", fmt.Errorf("YAML pointer segment %q does not address a mapping or sequence", segment)
		}
		if node.Kind == yaml.AliasNode {
			return "", errors.New("YAML aliases are unsupported at configured check references")
		}
	}
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return "", errors.New("configured check reference must identify one YAML string scalar")
	}
	return node.Value, nil
}

func validName(value string) bool {
	return value != "" && len(value) <= 128 && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

func validDigest(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func pipelineFinding(pipeline Pipeline, code, check, yamlPath, message string) Finding {
	return Finding{Code: code, Severity: "error", Pipeline: pipeline.Name, Provider: pipeline.Provider, Path: pipeline.Path, Owner: pipeline.Owner, Check: check, YAMLPath: yamlPath, Message: message}
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		left, right := findings[i], findings[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.Check != right.Check {
			return left.Check < right.Check
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.YAMLPath != right.YAMLPath {
			return left.YAMLPath < right.YAMLPath
		}
		return left.Pipeline < right.Pipeline
	})
}

func hasErrors(findings []Finding) bool {
	for _, finding := range findings {
		if finding.Severity == "error" {
			return true
		}
	}
	return false
}
