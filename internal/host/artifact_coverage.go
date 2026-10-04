package host

import (
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/artifactcoverage"
)

// CheckArtifacts composes artifact ownership from one working-tree snapshot,
// the normalized Project model, resolved input facts, and renderer outputs.
// The supplied configPath is repository-relative; an empty path selects the
// default markitect-artifacts.yaml.
func CheckArtifacts(root, configPath string) (artifactcoverage.Report, error) {
	if configPath == "" {
		configPath = "markitect-artifacts.yaml"
	}
	snap, err := source.Load(root, "")
	if err != nil {
		return artifactcoverage.Report{}, err
	}
	configBytes, ok := snap.Files[configPath]
	if !ok {
		return artifactcoverage.Report{}, fmt.Errorf("artifact coverage config %q is absent from the working tree snapshot", configPath)
	}
	config, err := artifactcoverage.ParseConfig(configBytes)
	if err != nil {
		return artifactcoverage.Report{}, err
	}
	if err := validateArtifactCoverageSourcePaths(config, configPath, snap); err != nil {
		return artifactcoverage.Report{}, err
	}
	config.SourcePath = configPath

	project, err := Parse(snap)
	if err != nil {
		return artifactcoverage.Report{}, fmt.Errorf("parse Markitect project: %w", err)
	}
	model, err := CompileModel(project)
	if err != nil {
		return artifactcoverage.Report{}, fmt.Errorf("compile normalized project model: %w", err)
	}

	_, generatedOwners, renderErr := GenerateOutputsWithOwners(project)

	inventory := artifactcoverage.Inventory{
		SnapshotDigest:  snap.Digest(),
		Files:           snap.Files,
		CanonicalOwners: canonicalArtifactOwners(project, model),
		InputOwners:     resolvedArtifactInputs(project),
		GeneratedOwners: rendererArtifactOwners(generatedOwners),
	}
	report, err := artifactcoverage.Check(config, inventory)
	if err != nil {
		return artifactcoverage.Report{}, err
	}
	for _, diagnostic := range project.Diagnostics {
		report.Findings = append(report.Findings, artifactcoverage.Finding{Code: diagnostic.Code, Path: diagnostic.Path, Message: diagnostic.Message})
	}
	if renderErr != nil {
		report.Findings = append(report.Findings, artifactcoverage.Finding{Code: "render", Message: renderErr.Error()})
	}
	if err := reconcileArtifactFindings(&report, root); err != nil {
		return artifactcoverage.Report{}, err
	}
	return report, nil
}

func validateArtifactCoverageSourcePaths(config artifactcoverage.Config, configPath string, snap *snapshot.Snapshot) error {
	paths := make([]string, 0, len(snap.Files)+1+len(config.Spec.Tooling)+len(config.Spec.Exclusions))
	for name := range snap.Files {
		paths = append(paths, name)
	}
	paths = append(paths, configPath)
	for _, root := range config.Spec.Roots {
		if err := source.ValidateIncludedPaths([]string{root}); err != nil {
			return fmt.Errorf("managed root %q is not available to Markitect snapshots: %w", root, err)
		}
	}
	for _, declaration := range config.Spec.Tooling {
		paths = append(paths, declaration.Path)
	}
	for _, exclusion := range config.Spec.Exclusions {
		paths = append(paths, exclusion.Path)
	}
	if err := source.ValidateIncludedPaths(paths); err != nil {
		return fmt.Errorf("artifact coverage paths are not available to Markitect snapshots: %w", err)
	}
	return nil
}

func canonicalArtifactOwners(project *Project, model core.SemanticModel) []artifactcoverage.OwnerFact {
	owners := make([]artifactcoverage.OwnerFact, 0, len(model.Resources)+len(model.DomainInputs)+1)
	if project != nil && project.Graph != nil && project.Graph.Project != nil {
		owners = append(owners, artifactcoverage.OwnerFact{Path: project.Graph.Project.Path, Owner: project.Graph.Project.GraphKey()})
	}
	for _, resource := range model.Resources {
		if resource.Identity.Package != "" {
			continue
		}
		owners = append(owners, artifactcoverage.OwnerFact{Path: resource.Source.Path, Owner: resource.Identity.Key})
	}
	for _, domain := range model.DomainInputs {
		if domain.Package == "" {
			owners = append(owners, artifactcoverage.OwnerFact{Path: domain.Path, Owner: "domain:" + domain.APIVersion + "/" + domain.Name})
		}
	}
	return owners
}

func resolvedArtifactInputs(project *Project) []artifactcoverage.OwnerFact {
	if project == nil || project.Graph == nil {
		return nil
	}
	var owners []artifactcoverage.OwnerFact
	for resourceKey, files := range project.InputFiles {
		resource := project.Graph.Resources[resourceKey]
		if resource == nil || resource.Package != "" || resource.Kind == "Project" || resource.Kind == "Package" {
			continue
		}
		for _, name := range files {
			owners = append(owners, artifactcoverage.OwnerFact{Path: name, Owner: resourceKey})
		}
	}
	sort.Slice(owners, func(i, j int) bool {
		if owners[i].Path != owners[j].Path {
			return owners[i].Path < owners[j].Path
		}
		return owners[i].Owner < owners[j].Owner
	})
	return owners
}

func rendererArtifactOwners(byPath map[string][]string) []artifactcoverage.OwnerFact {
	var owners []artifactcoverage.OwnerFact
	for name, resourceOwners := range byPath {
		for _, owner := range resourceOwners {
			owners = append(owners, artifactcoverage.OwnerFact{Path: name, Owner: owner})
		}
	}
	sort.Slice(owners, func(i, j int) bool {
		if owners[i].Path != owners[j].Path {
			return owners[i].Path < owners[j].Path
		}
		return owners[i].Owner < owners[j].Owner
	})
	return owners
}

func reconcileArtifactFindings(report *artifactcoverage.Report, root string) error {
	for i := range report.Findings {
		switch report.Findings[i].Code {
		case "missing-generated":
			message := "renderer-owned generated file is missing; run markitect render --write"
			if start := strings.Index(report.Findings[i].Message, "(owners: "); start >= 0 {
				message += " " + report.Findings[i].Message[start:]
			}
			report.Findings[i].Message = message
		case "stale-tooling":
			report.Findings[i].Message = "tooling ownership declaration does not name a file in the working tree snapshot"
		case "stale-exclusion":
			report.Findings[i].Message = "exclusion declaration does not name a file in the working tree snapshot"
		case "stale-root":
			report.Findings[i].Message = "managed root is absent from the working tree and has no renderer-owned output"
		}
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	for i := 0; i < len(report.Findings); {
		finding := report.Findings[i]
		if finding.Code != "stale-root" {
			i++
			continue
		}
		full := filepath.Join(absoluteRoot, filepath.FromSlash(finding.Path))
		info, statErr := os.Lstat(full)
		if statErr == nil && info.IsDir() && !isReparsePoint(info) {
			report.Findings = append(report.Findings[:i], report.Findings[i+1:]...)
			continue
		}
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return fmt.Errorf("inspect managed root %q: %w", finding.Path, statErr)
		}
		i++
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Path != report.Findings[j].Path {
			return report.Findings[i].Path < report.Findings[j].Path
		}
		if report.Findings[i].Code != report.Findings[j].Code {
			return report.Findings[i].Code < report.Findings[j].Code
		}
		return report.Findings[i].Message < report.Findings[j].Message
	})
	if len(report.Findings) == 0 {
		report.Status = "passed"
	} else {
		report.Status = "failed"
	}
	return nil
}
