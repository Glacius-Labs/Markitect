package host

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/githooks"
	"github.com/Glacius-Labs/Markitect/internal/modules/pipelines"
)

const (
	ModuleChecksReportVersion = "markitect.example.org/module-checks-report/v1alpha1"
	ModuleChecksPassed        = "passed"
	ModuleChecksFindings      = "findings"
	ModuleChecksNotConfigured = "not-configured"
)

// ModuleChecksOptions selects one fixed source snapshot and the exact module
// configuration paths to compose. Empty module paths are reported as
// not-configured.
type ModuleChecksOptions struct {
	Root            string
	Revision        string
	HooksConfigPath string
	PipelinesPath   string
}

// ModuleChecksReport preserves each selected module's report and provenance.
// Modules are always ordered git-hooks then pipelines.
type ModuleChecksReport struct {
	APIVersion     string              `yaml:"apiVersion"`
	Status         string              `yaml:"status"`
	SnapshotID     string              `yaml:"snapshotID,omitempty"`
	SnapshotDigest string              `yaml:"snapshotDigest"`
	ModelDigest    string              `yaml:"modelDigest"`
	Modules        []ModuleCheckResult `yaml:"modules"`
}

// ModuleCheckResult records the exact configuration source and its native
// module report without flattening away module-specific findings or identity.
type ModuleCheckResult struct {
	Name         string            `yaml:"name"`
	ConfigPath   string            `yaml:"configPath,omitempty"`
	ConfigDigest string            `yaml:"configDigest,omitempty"`
	Status       string            `yaml:"status"`
	GitHooks     *githooks.Report  `yaml:"gitHooks,omitempty"`
	Pipelines    *pipelines.Report `yaml:"pipelines,omitempty"`
}

// CheckModules loads one source snapshot, compiles its strict semantic model,
// and invokes only the explicitly selected read-only modules.
func CheckModules(options ModuleChecksOptions) (ModuleChecksReport, error) {
	if options.HooksConfigPath == "" && options.PipelinesPath == "" {
		return ModuleChecksReport{}, errors.New("at least one module configuration path is required")
	}
	snap, err := source.Load(options.Root, options.Revision)
	if err != nil {
		return ModuleChecksReport{}, fmt.Errorf("load source snapshot: %w", err)
	}
	project, err := Parse(snap)
	if err != nil {
		return ModuleChecksReport{}, fmt.Errorf("parse project: %w", err)
	}
	model, err := CompileModel(project)
	if err != nil {
		return ModuleChecksReport{}, fmt.Errorf("compile semantic model: %w", err)
	}
	if len(project.Diagnostics) != 0 || model.ValidationStatus != ModuleChecksPassed || model.StructuralStatus != ModuleChecksPassed {
		return ModuleChecksReport{}, strictModuleCheckError(project.Diagnostics)
	}

	ownership := moduleInputOwners(project)
	report := ModuleChecksReport{
		APIVersion: ModuleChecksReportVersion, Status: ModuleChecksPassed,
		SnapshotID: snap.ID, SnapshotDigest: snap.Digest(), ModelDigest: model.ModelDigest,
		Modules: make([]ModuleCheckResult, 0, 2),
	}
	if options.HooksConfigPath != "" {
		result, err := runGitHooksModule(project, model, options.HooksConfigPath, ownership)
		if err != nil {
			return ModuleChecksReport{}, err
		}
		report.Modules = append(report.Modules, result)
		report.Status = aggregateModuleStatus(report.Status, result.Status)
	} else {
		report.Modules = append(report.Modules, ModuleCheckResult{Name: "git-hooks", Status: githooks.StatusNotConfigured})
	}
	if options.PipelinesPath != "" {
		result, err := runPipelinesModule(project, model, options.PipelinesPath, ownership)
		if err != nil {
			return ModuleChecksReport{}, err
		}
		report.Modules = append(report.Modules, result)
		report.Status = aggregateModuleStatus(report.Status, result.Status)
	} else {
		report.Modules = append(report.Modules, ModuleCheckResult{Name: "pipelines", Status: pipelines.StatusNotConfigured})
	}
	return report, nil
}

func runGitHooksModule(project *Project, model core.SemanticModel, configPath string, ownership map[string][]string) (ModuleCheckResult, error) {
	configBytes, err := configuredModuleFile(project, configPath, ownership)
	if err != nil {
		return ModuleCheckResult{}, fmt.Errorf("git hooks config: %w", err)
	}
	config, err := githooks.DecodeConfig(configBytes)
	if err != nil {
		return ModuleCheckResult{}, err
	}
	artifacts, err := configuredArtifacts(project, config.Hooks, func(h githooks.Hook) (string, string) { return h.Path, h.Name }, ownership)
	if err != nil {
		return ModuleCheckResult{}, fmt.Errorf("git hooks artifacts: %w", err)
	}
	native, err := githooks.Check(githooks.Input{Model: model, Config: config, Artifacts: artifacts, Ownership: ownership})
	if err != nil {
		return ModuleCheckResult{}, fmt.Errorf("git hooks module: %w", err)
	}
	return ModuleCheckResult{Name: "git-hooks", ConfigPath: configPath, ConfigDigest: native.ConfigDigest, Status: native.Status, GitHooks: &native}, nil
}

func runPipelinesModule(project *Project, model core.SemanticModel, configPath string, ownership map[string][]string) (ModuleCheckResult, error) {
	configBytes, err := configuredModuleFile(project, configPath, ownership)
	if err != nil {
		return ModuleCheckResult{}, fmt.Errorf("pipelines config: %w", err)
	}
	config, err := pipelines.DecodeConfig(configBytes)
	if err != nil {
		return ModuleCheckResult{}, err
	}
	artifacts, err := configuredArtifacts(project, config.Pipelines, func(p pipelines.Pipeline) (string, string) { return p.Path, p.Name }, ownership)
	if err != nil {
		return ModuleCheckResult{}, fmt.Errorf("pipeline artifacts: %w", err)
	}
	checks := make([]pipelines.CheckFact, 0, len(project.Graph.Project.Spec.Checks))
	for _, check := range project.Graph.Project.Spec.Checks {
		// This is a literal argv rendering with one space between arguments. It
		// supports exact YAML scalar comparison; it is not shell parsing or proof
		// that a pipeline executes the check.
		checks = append(checks, pipelines.CheckFact{Name: check.Name, Reference: strings.Join(check.Run, " ")})
	}
	native, err := pipelines.Check(pipelines.Input{Model: model, Config: config, Artifacts: artifacts, Ownership: ownership, CheckFacts: checks})
	if err != nil {
		return ModuleCheckResult{}, fmt.Errorf("pipelines module: %w", err)
	}
	return ModuleCheckResult{Name: "pipelines", ConfigPath: configPath, ConfigDigest: native.ConfigDigest, Status: native.Status, Pipelines: &native}, nil
}

func configuredModuleFile(project *Project, file string, ownership map[string][]string) ([]byte, error) {
	if err := validateModuleSourcePath(file); err != nil {
		return nil, err
	}
	data, ok := project.Snapshot.Files[file]
	if !ok {
		return nil, fmt.Errorf("exact config file %q is absent from the selected snapshot", file)
	}
	if len(ownership[file]) == 0 {
		return nil, fmt.Errorf("exact config file %q is not an explicitly declared input of a canonical resource", file)
	}
	return data, nil
}

func configuredArtifacts[T any](project *Project, entries []T, pathAndName func(T) (string, string), ownership map[string][]string) (map[string][]byte, error) {
	artifacts := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		file, name := pathAndName(entry)
		if len(ownership[file]) == 0 {
			return nil, fmt.Errorf("configured artifact %q (%s) is not an explicitly declared input of a canonical resource", file, name)
		}
		data, ok := project.Snapshot.Files[file]
		if !ok {
			return nil, fmt.Errorf("configured artifact %q (%s) is absent from the selected snapshot", file, name)
		}
		artifacts[file] = append([]byte(nil), data...)
	}
	return artifacts, nil
}

func moduleInputOwners(project *Project) map[string][]string {
	result := make(map[string][]string)
	for owner, files := range project.InputFiles {
		for _, file := range files {
			result[file] = append(result[file], owner)
		}
	}
	for file := range result {
		sort.Strings(result[file])
	}
	return result
}

func validateModuleSourcePath(file string) error {
	if file == "" || strings.TrimSpace(file) != file || strings.ContainsAny(file, "\\:\x00") || path.IsAbs(file) || path.Clean(file) != file || file == "." || strings.HasPrefix(file, "../") || strings.HasSuffix(file, "/") {
		return fmt.Errorf("path %q must be an exact repository-relative file path", file)
	}
	for _, segment := range strings.Split(file, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("path %q must be an exact repository-relative file path", file)
		}
	}
	return nil
}

func strictModuleCheckError(diagnostics []core.Diagnostic) error {
	if len(diagnostics) == 0 {
		return errors.New("strict project check did not pass")
	}
	ordered := append([]core.Diagnostic(nil), diagnostics...)
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Message < b.Message
	})
	parts := make([]string, 0, len(ordered))
	for _, diagnostic := range ordered {
		location := diagnostic.Path
		if diagnostic.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, diagnostic.Line)
		}
		parts = append(parts, fmt.Sprintf("%s %s: %s", location, diagnostic.Code, diagnostic.Message))
	}
	return fmt.Errorf("strict project check failed: %s", strings.Join(parts, "; "))
}

func aggregateModuleStatus(current, module string) string {
	if current == ModuleChecksFindings || module == githooks.StatusFindings || module == pipelines.StatusFindings {
		return ModuleChecksFindings
	}
	if current == ModuleChecksNotConfigured || module == githooks.StatusNotConfigured || module == pipelines.StatusNotConfigured {
		return ModuleChecksNotConfigured
	}
	return ModuleChecksPassed
}
