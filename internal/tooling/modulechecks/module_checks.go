// Package modulechecks composes read-only module checks from actual canonical
// preparation and its bound census. It executes no hook or pipeline.
package modulechecks

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/tooling/modules/githooks"
	"github.com/Glacius-Labs/Markitect/internal/tooling/modules/pipelines"
)

const StatusPassed = "passed"

type Options struct {
	Root, Revision, HooksConfigPath, PipelinesPath string
}

type Report struct {
	Status    string           `yaml:"status"`
	Hooks     githooks.Report  `yaml:"hooks"`
	Pipelines pipelines.Report `yaml:"pipelines"`
}

func Check(options Options) (Report, error) {
	ctx := context.Background()
	root, err := filepath.Abs(options.Root)
	if err != nil {
		return Report{}, err
	}
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		return Report{}, err
	}
	revision := ""
	if options.Revision != "" {
		revision, err = repository.ResolveRevision(ctx, root, options.Revision)
		if err != nil {
			return Report{}, err
		}
	}
	manifest, _ := snapshot.NewPath(".markitect/model.yaml")
	result, err := apply.NewPreparationService(apply.NewModelService(repository, repository)).Preview(ctx, apply.PreviewRequest{
		Selection: apply.InputSelection{Repository: root, Local: apply.ModelSelection{Boundary: ".", ManifestPath: manifest}},
		Revision:  revision,
	}, model.Limits{})
	if err != nil {
		return Report{}, err
	}
	prepared, complete := result.Prepared()
	if !complete {
		return Report{}, fmt.Errorf("module checks require passed structural preparation, got %s", result.Status())
	}
	return checkPrepared(prepared, options)
}

func checkPrepared(prepared apply.PreviewPreparationAttempt, options Options) (Report, error) {
	inputs := prepared.Model().Inputs()
	checked := prepared.Model().Checked()
	registry, ok := prepared.Registry().Registry()
	if !ok {
		return Report{}, errors.New("prepared registry unavailable")
	}
	owners := []string{"."}
	modelDigest := ""
	var checkFacts []pipelines.CheckFact
	for _, m := range checked.Models {
		if !m.Local {
			continue
		}
		modelDigest = m.Digest
		for _, p := range m.Packages {
			owners = append(owners, p.Name)
		}
		for _, d := range m.Definitions {
			if d.Kind != (model.Identity{Package: "vocabulary", ID: "Check"}) {
				continue
			}
			for _, f := range d.Spec {
				if f.Name == "command" {
					command, valid := f.Value.Text()
					if !valid {
						return Report{}, fmt.Errorf("Check %s/%s command is not text", d.Identity.Package, d.Identity.ID)
					}
					checkFacts = append(checkFacts, pipelines.CheckFact{Name: d.Identity.Package + "/" + d.Identity.ID, Reference: command})
				}
			}
		}
	}
	var hooksOwnership []githooks.OwnershipFact
	var pipelineOwnership []pipelines.OwnershipFact
	for _, r := range registry.Entries() {
		owner := "."
		if p, packageOwner := r.Owner().Package(); packageOwner {
			owner = p
		}
		name := r.Artifact().Path().String()
		hooksOwnership = append(hooksOwnership, githooks.OwnershipFact{Path: name, OwnerID: owner})
		pipelineOwnership = append(pipelineOwnership, pipelines.OwnershipFact{Path: name, OwnerID: owner})
	}
	files := make(map[string][]byte)
	for _, e := range inputs.Raw.Census.Entries() {
		files[e.Path().String()] = e.Bytes()
	}
	readConfig := func(name string) ([]byte, error) {
		p, err := snapshot.NewPath(name)
		if err != nil {
			return nil, err
		}
		if _, registered := registry.Lookup(p); !registered {
			return nil, fmt.Errorf("module configuration %q is not a registered artifact", name)
		}
		data, found := files[name]
		if !found {
			return nil, fmt.Errorf("module configuration %q absent from the selected census", name)
		}
		return data, nil
	}
	report := Report{Status: StatusPassed, Hooks: githooks.Report{APIVersion: githooks.ReportVersion, Status: githooks.StatusNotConfigured}, Pipelines: pipelines.Report{APIVersion: pipelines.ReportVersion, Status: pipelines.StatusNotConfigured}}
	digest := inputs.Raw.Census.Digest().String()
	if options.HooksConfigPath != "" {
		data, err := readConfig(options.HooksConfigPath)
		if err != nil {
			return Report{}, err
		}
		config, err := githooks.DecodeConfig(data)
		if err != nil {
			return Report{}, err
		}
		report.Hooks, err = githooks.Check(githooks.Input{Model: githooks.ModelFacts{StructuralStatus: StatusPassed, SnapshotID: inputs.Revision, SnapshotDigest: digest, ModelDigest: modelDigest, OwnerIDs: owners}, Config: config, Artifacts: files, Ownership: hooksOwnership})
		if err != nil {
			return Report{}, err
		}
		if report.Hooks.Status != githooks.StatusPassed {
			report.Status = "findings"
		}
	}
	if options.PipelinesPath != "" {
		data, err := readConfig(options.PipelinesPath)
		if err != nil {
			return Report{}, err
		}
		config, err := pipelines.DecodeConfig(data)
		if err != nil {
			return Report{}, err
		}
		report.Pipelines, err = pipelines.Check(pipelines.Input{Model: pipelines.ModelFacts{StructuralStatus: StatusPassed, SnapshotID: inputs.Revision, SnapshotDigest: digest, ModelDigest: modelDigest, OwnerIDs: owners}, Config: config, Artifacts: files, Ownership: pipelineOwnership, CheckFacts: checkFacts})
		if err != nil {
			return Report{}, err
		}
		if report.Pipelines.Status != pipelines.StatusPassed {
			report.Status = "findings"
		}
	}
	return report, nil
}
