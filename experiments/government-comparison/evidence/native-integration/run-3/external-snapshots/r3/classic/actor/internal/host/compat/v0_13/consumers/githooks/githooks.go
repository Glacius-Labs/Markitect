package githooks

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
	"strings"

	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

const (
	ConfigVersion            = "markitect.example.org/git-hooks-check/v1alpha1"
	ReportVersion            = "markitect.example.org/git-hooks-report/v1alpha1"
	StatusPassed             = "passed"
	StatusFindings           = "findings"
	StatusNotConfigured      = "not-configured"
	maxConfigBytes           = 1 << 20
	maxHookBytes             = 1 << 20
	maxConfiguredEntrypoints = 128
)

type Config struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Hooks      []Hook `yaml:"hooks" json:"hooks"`
}

type Hook struct {
	Name   string `yaml:"name" json:"name"`
	Stage  string `yaml:"stage" json:"stage"`
	Path   string `yaml:"path" json:"path"`
	Digest string `yaml:"digest,omitempty" json:"digest,omitempty"`
	Owner  string `yaml:"owner" json:"owner"`
}

// Input contains only normalized resource identities, configured hook bytes,
// and Host-supplied ownership facts.
type Input struct {
	Model     core.SemanticModel
	Config    Config
	Artifacts map[string][]byte
	Ownership map[string][]string
}

type Finding struct {
	Code     string `yaml:"code"`
	Severity string `yaml:"severity"`
	Name     string `yaml:"name,omitempty"`
	Stage    string `yaml:"stage,omitempty"`
	Path     string `yaml:"path,omitempty"`
	Owner    string `yaml:"owner,omitempty"`
	Message  string `yaml:"message"`
}

type Report struct {
	APIVersion     string    `yaml:"apiVersion"`
	Status         string    `yaml:"status"`
	SnapshotID     string    `yaml:"snapshotID,omitempty"`
	SnapshotDigest string    `yaml:"snapshotDigest"`
	ModelDigest    string    `yaml:"modelDigest"`
	ConfigDigest   string    `yaml:"configDigest"`
	Findings       []Finding `yaml:"findings"`
}

// DecodeConfig decodes the module-owned, closed configuration format.
func DecodeConfig(data []byte) (Config, error) {
	if len(data) == 0 || len(data) > maxConfigBytes {
		return Config{}, fmt.Errorf("git hooks config must contain 1..%d bytes", maxConfigBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode git hooks config: %w", err)
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Config{}, errors.New("git hooks config must contain exactly one YAML document")
		}
		return Config{}, fmt.Errorf("decode trailing git hooks config: %w", err)
	}
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

// Check checks configured exact hook entrypoints only. It never reads a
// filesystem, executes a hook, or interprets hook contents.
func Check(input Input) (Report, error) {
	if err := validateModel(input.Model); err != nil {
		return Report{}, err
	}
	if err := validateConfig(input.Config); err != nil {
		return Report{}, err
	}
	report := Report{
		APIVersion: ReportVersion, Status: StatusPassed,
		SnapshotID: input.Model.Snapshot.ID, SnapshotDigest: input.Model.Snapshot.Digest,
		ModelDigest: input.Model.ModelDigest, ConfigDigest: configDigest(input.Config), Findings: []Finding{},
	}
	if len(input.Config.Hooks) == 0 {
		report.Status = StatusNotConfigured
		report.Findings = append(report.Findings, Finding{
			Code: "hooks.not-configured", Severity: "info",
			Message: "No hook entrypoints are configured; no hook presence or linkage was verified.",
		})
		return report, nil
	}
	owners := modelOwners(input.Model)
	for _, hook := range input.Config.Hooks {
		data, exists := input.Artifacts[hook.Path]
		if !exists {
			report.Findings = append(report.Findings, hookFinding(hook, "hooks.artifact.missing", "configured hook entrypoint bytes were not supplied"))
		} else {
			if len(data) > maxHookBytes {
				return Report{}, fmt.Errorf("configured hook %q exceeds the %d-byte input limit", hook.Name, maxHookBytes)
			}
			if digest(data) != hook.Digest {
				report.Findings = append(report.Findings, hookFinding(hook, "hooks.digest.mismatch", "hook bytes do not match the configured SHA-256 digest"))
			}
		}
		if !owners[hook.Owner] {
			report.Findings = append(report.Findings, hookFinding(hook, "hooks.owner.resource-missing", "configured owner is not a resource identity in the supplied semantic model"))
		}
		if !contains(input.Ownership[hook.Path], hook.Owner) {
			report.Findings = append(report.Findings, hookFinding(hook, "hooks.owner.link-missing", "supplied ownership facts do not link this path to the configured owner"))
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
		return fmt.Errorf("unsupported git hooks config apiVersion %q", config.APIVersion)
	}
	if len(config.Hooks) > maxConfiguredEntrypoints {
		return fmt.Errorf("git hooks config exceeds %d configured entrypoints", maxConfiguredEntrypoints)
	}
	seenNames := map[string]bool{}
	seenPaths := make([]string, 0, len(config.Hooks))
	for _, hook := range config.Hooks {
		if !validName(hook.Name) || seenNames[hook.Name] {
			return fmt.Errorf("git hook names must be non-empty and unique: %q", hook.Name)
		}
		seenNames[hook.Name] = true
		switch hook.Stage {
		case "pre-commit", "pre-push", "commit-msg", "custom":
		default:
			return fmt.Errorf("git hook %q has unsupported stage %q", hook.Name, hook.Stage)
		}
		if !validExactPath(hook.Path) {
			return fmt.Errorf("git hook path must be safe and exact: %q", hook.Path)
		}
		for _, previous := range seenPaths {
			if strings.EqualFold(previous, hook.Path) {
				return fmt.Errorf("git hook paths must not collide under portable case folding: %q", hook.Path)
			}
		}
		seenPaths = append(seenPaths, hook.Path)
		if !validDigest(hook.Digest) {
			return fmt.Errorf("git hook %q digest must be 64 lowercase hexadecimal SHA-256 characters", hook.Name)
		}
		if !validName(hook.Owner) {
			return fmt.Errorf("git hook %q must name an expected managed owner identity", hook.Name)
		}
	}
	return nil
}

func configDigest(config Config) string {
	normalized := config
	normalized.Hooks = append([]Hook(nil), config.Hooks...)
	sort.Slice(normalized.Hooks, func(i, j int) bool {
		if normalized.Hooks[i].Path != normalized.Hooks[j].Path {
			return normalized.Hooks[i].Path < normalized.Hooks[j].Path
		}
		return normalized.Hooks[i].Name < normalized.Hooks[j].Name
	})
	canonical, _ := json.Marshal(normalized)
	return digest(canonical)
}

func validateModel(model core.SemanticModel) error {
	if model.APIVersion != core.SemanticModelVersion || model.StructuralStatus != "passed" || strings.TrimSpace(model.ModelDigest) == "" || strings.TrimSpace(model.Snapshot.Digest) == "" {
		return errors.New("git hooks check requires a structurally valid semantic model with fixed snapshot and model digests")
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

func hookFinding(hook Hook, code, message string) Finding {
	return Finding{Code: code, Severity: "error", Name: hook.Name, Stage: hook.Stage, Path: hook.Path, Owner: hook.Owner, Message: message}
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

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		left, right := findings[i], findings[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Stage != right.Stage {
			return left.Stage < right.Stage
		}
		return left.Owner < right.Owner
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
