// Package modulechecks composes explicit pure module checks over one source
// snapshot and the current canonical project model.
package modulechecks

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	
	"github.com/Glacius-Labs/Markitect/internal/tooling/source"
	"github.com/Glacius-Labs/Markitect/internal/tooling/modules/githooks"
	"github.com/Glacius-Labs/Markitect/internal/tooling/modules/pipelines"
)

const (
	ReportVersion  = "markitect.example.org/module-checks-report/v1alpha1"
	StatusPassed   = "passed"
	StatusFindings = "findings"
)

// Options selects one source snapshot and the exact module configuration paths
// to compose. Empty module paths are reported as not-configured.
type Options struct {
	Root            string
	Revision        string
	HooksConfigPath string
	PipelinesPath   string
}

// Report preserves each selected module's report and source provenance.
// Modules are always ordered git-hooks then pipelines.
type Report struct {
	APIVersion     string   `yaml:"apiVersion"`
	Status         string   `yaml:"status"`
	SnapshotID     string   `yaml:"snapshotID,omitempty"`
	SnapshotDigest string   `yaml:"snapshotDigest"`
	ModelDigest    string   `yaml:"modelDigest"`
	Modules        []Result `yaml:"modules"`
}

// Result records the exact configuration source and native module report.
type Result struct {
	Name         string            `yaml:"name"`
	ConfigPath   string            `yaml:"configPath,omitempty"`
	ConfigDigest string            `yaml:"configDigest,omitempty"`
	Status       string            `yaml:"status"`
	GitHooks     *githooks.Report  `yaml:"gitHooks,omitempty"`
	Pipelines    *pipelines.Report `yaml:"pipelines,omitempty"`
}

// Check loads one immutable or provisional source snapshot, compiles the
// selected canonical project model, and invokes only the explicitly selected
// read-only checks.
func Check(options Options) (Report, error) {
	if options.HooksConfigPath == "" && options.PipelinesPath == "" {
		return Report{}, errors.New("at least one module configuration path is required")
	}
	snap, err := source.Load(options.Root, options.Revision)
	if err != nil {
		return Report{}, fmt.Errorf("load source snapshot: %w", err)
	}
	project, err := projectwork.FromSnapshot(options.Root, snap)
	if err != nil {
		return Report{}, fmt.Errorf("compile current project model: %w", err)
	}
	inventory, err := snapshotInventory(snap)
	if err != nil {
		return Report{}, err
	}
	modelReport := projectmodel.Analyze(project.Model, inventory)
	if err := strictModelError(modelReport); err != nil {
		return Report{}, err
	}

	hookModel := hookModelFacts(project, snap, modelReport)
	pipelineModel := pipelineModelFacts(project, snap, modelReport)
	ownership := declaredArtifactOwnership(modelReport)
	report := Report{
		APIVersion: ReportVersion, Status: StatusPassed,
		SnapshotID: snap.ID, SnapshotDigest: snap.Digest(), ModelDigest: project.Model.Digest,
		Modules: make([]Result, 0, 2),
	}
	if options.HooksConfigPath != "" {
		result, err := runHooksModule(snap, hookModel, ownership, options.HooksConfigPath)
		if err != nil {
			return Report{}, err
		}
		report.Modules = append(report.Modules, result)
		report.Status = aggregateStatus(report.Status, result.Status)
	} else {
		report.Modules = append(report.Modules, Result{Name: "git-hooks", Status: githooks.StatusNotConfigured})
	}
	if options.PipelinesPath != "" {
		result, err := runPipelinesModule(snap, pipelineModel, ownership, modelReport.Checks, options.PipelinesPath)
		if err != nil {
			return Report{}, err
		}
		report.Modules = append(report.Modules, result)
		report.Status = aggregateStatus(report.Status, result.Status)
	} else {
		report.Modules = append(report.Modules, Result{Name: "pipelines", Status: pipelines.StatusNotConfigured})
	}
	return report, nil
}

func runHooksModule(snap *source.Input, model githooks.ModelFacts, ownership []githooks.OwnershipFact, configPath string) (Result, error) {
	configBytes, err := configuredModuleFile(snap, configPath, ownership)
	if err != nil {
		return Result{}, fmt.Errorf("git hooks config: %w", err)
	}
	config, err := githooks.DecodeConfig(configBytes)
	if err != nil {
		return Result{}, err
	}
	artifacts, err := configuredArtifacts(snap, config.Hooks, func(h githooks.Hook) (string, string) { return h.Path, h.Name }, ownership)
	if err != nil {
		return Result{}, fmt.Errorf("git hooks artifacts: %w", err)
	}
	native, err := githooks.Check(githooks.Input{Model: model, Config: config, Artifacts: artifacts, Ownership: ownership})
	if err != nil {
		return Result{}, fmt.Errorf("git hooks module: %w", err)
	}
	return Result{Name: "git-hooks", ConfigPath: configPath, ConfigDigest: native.ConfigDigest, Status: native.Status, GitHooks: &native}, nil
}

func runPipelinesModule(snap *source.Input, model pipelines.ModelFacts, ownership []githooks.OwnershipFact, checks []projectmodel.Check, configPath string) (Result, error) {
	configBytes, err := configuredModuleFile(snap, configPath, ownership)
	if err != nil {
		return Result{}, fmt.Errorf("pipelines config: %w", err)
	}
	config, err := pipelines.DecodeConfig(configBytes)
	if err != nil {
		return Result{}, err
	}
	artifacts, err := configuredArtifacts(snap, config.Pipelines, func(p pipelines.Pipeline) (string, string) { return p.Path, p.Name }, ownership)
	if err != nil {
		return Result{}, fmt.Errorf("pipeline artifacts: %w", err)
	}
	checkFacts := make([]pipelines.CheckFact, 0, len(checks))
	for _, check := range checks {
		// This renders argv literally with one space between arguments. It is not
		// shell parsing or proof that a provider executes the command.
		checkFacts = append(checkFacts, pipelines.CheckFact{Name: check.Name, Reference: strings.Join(check.Command, " ")})
	}
	native, err := pipelines.Check(pipelines.Input{Model: model, Config: config, Artifacts: artifacts, Ownership: pipelineOwnership(ownership), CheckFacts: checkFacts})
	if err != nil {
		return Result{}, fmt.Errorf("pipelines module: %w", err)
	}
	return Result{Name: "pipelines", ConfigPath: configPath, ConfigDigest: native.ConfigDigest, Status: native.Status, Pipelines: &native}, nil
}

func hookModelFacts(project *projectvalue.Project, snap *source.Input, report projectmodel.Report) githooks.ModelFacts {
	facts := githooks.ModelFacts{
		StructuralStatus: "passed", SnapshotID: snap.ID, SnapshotDigest: snap.Digest(),
		ModelDigest: project.Model.Digest, OwnerIDs: make([]string, 0, len(report.Artifacts)),
	}
	for _, artifact := range report.Artifacts {
		facts.OwnerIDs = append(facts.OwnerIDs, artifact.ID)
	}
	sort.Strings(facts.OwnerIDs)
	return facts
}

func pipelineModelFacts(project *projectvalue.Project, snap *source.Input, report projectmodel.Report) pipelines.ModelFacts {
	facts := pipelines.ModelFacts{
		StructuralStatus: "passed", SnapshotID: snap.ID, SnapshotDigest: snap.Digest(),
		ModelDigest: project.Model.Digest, OwnerIDs: make([]string, 0, len(report.Artifacts)),
	}
	for _, artifact := range report.Artifacts {
		facts.OwnerIDs = append(facts.OwnerIDs, artifact.ID)
	}
	sort.Strings(facts.OwnerIDs)
	return facts
}

func declaredArtifactOwnership(report projectmodel.Report) []githooks.OwnershipFact {
	var facts []githooks.OwnershipFact
	for _, artifact := range report.Artifacts {
		for _, path := range artifact.Paths {
			// Module checks use exact links only. Prefix artifacts remain valid for
			// project coverage but do not authorize a configured module input.
			if validateModuleSourcePath(path) == nil {
				facts = append(facts, githooks.OwnershipFact{Path: path, OwnerID: artifact.ID})
			}
		}
	}
	sort.Slice(facts, func(i, j int) bool {
		if facts[i].Path != facts[j].Path {
			return facts[i].Path < facts[j].Path
		}
		return facts[i].OwnerID < facts[j].OwnerID
	})
	return facts
}

func pipelineOwnership(facts []githooks.OwnershipFact) []pipelines.OwnershipFact {
	result := make([]pipelines.OwnershipFact, 0, len(facts))
	for _, fact := range facts {
		result = append(result, pipelines.OwnershipFact{Path: fact.Path, OwnerID: fact.OwnerID})
	}
	return result
}

func configuredModuleFile(snap *source.Input, file string, ownership []githooks.OwnershipFact) ([]byte, error) {
	if err := validateModuleSourcePath(file); err != nil {
		return nil, err
	}
	data, ok := snap.Files[file]
	if !ok {
		return nil, fmt.Errorf("exact config file %q is absent from the selected snapshot", file)
	}
	if !hasAnyOwner(ownership, file) {
		return nil, fmt.Errorf("exact config file %q is not an explicitly declared input of a canonical Artifact", file)
	}
	return append([]byte(nil), data...), nil
}

func configuredArtifacts[T any](snap *source.Input, entries []T, pathAndName func(T) (string, string), ownership []githooks.OwnershipFact) (map[string][]byte, error) {
	artifacts := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		file, name := pathAndName(entry)
		if !hasAnyOwner(ownership, file) {
			return nil, fmt.Errorf("configured artifact %q (%s) is not an explicitly declared input of a canonical Artifact", file, name)
		}
		data, ok := snap.Files[file]
		if !ok {
			return nil, fmt.Errorf("configured artifact %q (%s) is absent from the selected snapshot", file, name)
		}
		artifacts[file] = append([]byte(nil), data...)
	}
	return artifacts, nil
}

func hasAnyOwner(facts []githooks.OwnershipFact, file string) bool {
	for _, fact := range facts {
		if fact.Path == file {
			return true
		}
	}
	return false
}

func strictModelError(report projectmodel.Report) error {
	if report.Status == "succeeded" && len(report.Findings) == 0 && len(report.Unknown) == 0 {
		return nil
	}
	parts := make([]string, 0, len(report.Findings)+len(report.Unknown))
	for _, finding := range report.Findings {
		parts = append(parts, fmt.Sprintf("%s %s: %s", finding.Code, finding.Subject, finding.Message))
	}
	for _, unknown := range report.Unknown {
		parts = append(parts, "unknown repository path "+unknown)
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		return fmt.Errorf("strict project model analysis did not pass: status %q", report.Status)
	}
	return fmt.Errorf("strict project model analysis did not pass: %s", strings.Join(parts, "; "))
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

func aggregateStatus(current, module string) string {
	if current == StatusFindings || module == githooks.StatusFindings || module == pipelines.StatusFindings {
		return StatusFindings
	}
	if current == githooks.StatusNotConfigured || current == pipelines.StatusNotConfigured || module == githooks.StatusNotConfigured || module == pipelines.StatusNotConfigured {
		return githooks.StatusNotConfigured
	}
	return StatusPassed
}

func snapshotInventory(snap *source.Input) ([]projectmodel.File, error) {
	files := make([]projectmodel.File, 0, len(snap.Files))
	for path, data := range snap.Files {
		mode := snap.Modes[path]
		if mode == "" {
			return nil, fmt.Errorf("source snapshot file %q has no mode", path)
		}
		digest := sha256.Sum256(data)
		files = append(files, projectmodel.File{Path: path, Mode: mode, Digest: hex.EncodeToString(digest[:])})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
