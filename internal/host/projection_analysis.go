package host

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/projections"
)

// ProjectionInclusion maps directly matched semantic source resources to one
// explicitly declared projection contract. It creates no graph/context edge
// and does not infer ownership from observed output paths.
type ProjectionInclusion struct {
	ConfigPath         string                   `yaml:"configPath"`
	ConfigDigest       string                   `yaml:"configDigest"`
	ContractID         string                   `yaml:"contractId"`
	Sources            []string                 `yaml:"sources"`
	MatchingSourceKeys []string                 `yaml:"matchingSourceKeys"`
	Representation     string                   `yaml:"representation"`
	Targets            []string                 `yaml:"targets"`
	Materializer       projections.Materializer `yaml:"materializer"`
	Freedom            []string                 `yaml:"freedom,omitempty"`
	VerificationChecks []string                 `yaml:"verificationChecks,omitempty"`
	DependsOn          []string                 `yaml:"dependsOn,omitempty"`
	ViaContracts       []string                 `yaml:"viaContracts,omitempty"`
	ViaInputs          []string                 `yaml:"viaInputs,omitempty"`
}

// ProjectionContext returns the contracts whose explicitly configured source
// keys occur in a compiled Context's input keys. The caller owns context
// closure selection; this helper only adds the declared target fan-out.
func ProjectionContext(p *Project, keys []string) ([]ProjectionInclusion, error) {
	return projectionInclusionsForKeys(p, keys, true, nil)
}

// ProjectionImpact returns the contracts whose explicitly configured source
// keys occur in an Impact affected-key set. The caller owns base/candidate
// comparison; this helper does not add semantic dependencies or infer outputs.
func ProjectionImpact(p *Project, affected []string, changedPaths ...[]string) ([]ProjectionInclusion, error) {
	var changed []string
	if len(changedPaths) > 0 {
		changed = changedPaths[0]
	}
	return projectionInclusionsForKeys(p, affected, false, changed)
}

func projectionInclusionsForKeys(p *Project, keys []string, includeDomains bool, changedPaths []string) ([]ProjectionInclusion, error) {
	configPath, coveragePath, active, err := RegisteredProjection(p)
	if err != nil {
		return nil, err
	}
	if !active {
		return []ProjectionInclusion{}, nil
	}
	if p == nil || p.Snapshot == nil {
		return nil, errors.New("projection analysis requires a captured Project snapshot")
	}
	if err := source.ValidateIncludedPaths([]string{configPath}); err != nil {
		return nil, fmt.Errorf("projection config path: %w", err)
	}
	data, ok := p.Snapshot.Files[configPath]
	if !ok {
		return nil, fmt.Errorf("registered projection config %q is absent from the selected snapshot", configPath)
	}
	config, err := projections.ParseConfig(data)
	if err != nil {
		return nil, err
	}
	model, err := CompileModel(p)
	if err != nil {
		return nil, fmt.Errorf("projection analysis semantic model: %w", err)
	}
	sourceIndex, err := projections.SourceIndex(model)
	if err != nil {
		return nil, fmt.Errorf("projection analysis source index: %w", err)
	}
	for _, contract := range config.Contracts {
		for _, key := range contract.Sources {
			if _, ok := sourceIndex[key]; !ok {
				return nil, fmt.Errorf("projection contract %q references unknown semantic source %q", contract.ID, key)
			}
		}
	}
	effectiveKeys := append([]string(nil), keys...)
	changed := make(map[string]bool, len(changedPaths))
	for _, name := range changedPaths {
		changed[name] = true
	}
	var viaInputs []string
	for _, name := range []string{configPath, coveragePath, p.Graph.Project.Path} {
		if changed[name] {
			viaInputs = append(viaInputs, name)
		}
	}
	sort.Strings(viaInputs)
	for key, entry := range sourceIndex {
		if entry.Kind != "Domain" || !strings.HasPrefix(key, "domain:") {
			continue
		}
		selected := includeDomains || changed[entry.Source.Path]
		if !selected && entry.Package != "" && p.Graph.Project != nil {
			for _, pin := range p.Graph.Project.Spec.Packages {
				if pin.Name == entry.Package && changed[pin.Archive] {
					selected = true
				}
			}
		}
		if selected {
			effectiveKeys = append(effectiveKeys, key)
		}
	}
	return mapProjectionInclusions(config, effectiveKeys, configPath, hashBytes(data), projectionInclusionOptions{IncludeDependents: !includeDomains, ViaInputs: viaInputs}), nil
}

type projectionInclusionOptions struct {
	IncludeDependents bool
	ViaInputs         []string
}

func mapProjectionInclusions(config projections.Config, keys []string, configPath, configDigest string, options ...projectionInclusionOptions) []ProjectionInclusion {
	selected := make(map[string]bool, len(keys))
	for _, key := range keys {
		if key != "" {
			selected[key] = true
		}
	}
	var option projectionInclusionOptions
	if len(options) > 0 {
		option = options[0]
	}
	if len(selected) == 0 && len(option.ViaInputs) == 0 {
		return []ProjectionInclusion{}
	}

	matchedContracts := map[string]bool{}
	for _, contract := range config.Contracts {
		if len(option.ViaInputs) > 0 {
			matchedContracts[contract.ID] = true
		}
		for _, key := range contract.Sources {
			if selected[key] {
				matchedContracts[contract.ID] = true
			}
		}
	}
	propagate := option.IncludeDependents
	if propagate {
		// Bounded closure over declared contract prerequisites, not semantic edges.
		for changed := true; changed; {
			changed = false
			for _, contract := range config.Contracts {
				if matchedContracts[contract.ID] {
					continue
				}
				for _, prerequisite := range contract.DependsOn {
					if matchedContracts[prerequisite] {
						matchedContracts[contract.ID] = true
						changed = true
						break
					}
				}
			}
		}
	}
	var inclusions []ProjectionInclusion
	for _, contract := range config.Contracts {
		matching := make([]string, 0, len(contract.Sources))
		for _, sourceKey := range contract.Sources {
			if selected[sourceKey] {
				matching = append(matching, sourceKey)
			}
		}
		if !matchedContracts[contract.ID] {
			continue
		}
		var via []string
		if propagate {
			for _, prerequisite := range contract.DependsOn {
				if matchedContracts[prerequisite] {
					via = append(via, prerequisite)
				}
			}
		}
		sort.Strings(via)
		prerequisites := append([]string(nil), contract.DependsOn...)
		sort.Strings(prerequisites)
		sort.Strings(matching)
		sources := append([]string(nil), contract.Sources...)
		sort.Strings(sources)
		targets := make([]string, 0, len(contract.Targets))
		for _, target := range contract.Targets {
			targets = append(targets, target.Path)
		}
		sort.Strings(targets)
		checks := append([]string(nil), contract.VerificationChecks...)
		sort.Strings(checks)
		freedom := append([]string(nil), contract.Freedom...)
		sort.Strings(freedom)
		inclusions = append(inclusions, ProjectionInclusion{
			ConfigPath:         configPath,
			ConfigDigest:       configDigest,
			ContractID:         contract.ID,
			Sources:            sources,
			MatchingSourceKeys: matching,
			Representation:     contract.Representation,
			Targets:            targets,
			Materializer:       contract.Materializer,
			Freedom:            freedom,
			VerificationChecks: checks,
			DependsOn:          prerequisites,
			ViaContracts:       via,
			ViaInputs:          append([]string(nil), option.ViaInputs...),
		})
	}
	sort.Slice(inclusions, func(i, j int) bool { return inclusions[i].ContractID < inclusions[j].ContractID })
	return inclusions
}
