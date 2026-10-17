package organization

import (
	"fmt"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ResolveSettings resolves supplied project defaults and only the exact local
// Manager override. There is no ancestor inheritance, deep merge, interpolation
// or implicit profile/instruction acquisition. Restrictions from later tasks
// are not inputs to this configuration operation. It renders and executes
// nothing, and does not assess instruction prose against model obligations.
// On any refusal it returns zero Settings, never a partially usable result.
func ResolveSettings(topology Topology, facts SettingsFacts) (Settings, error) {
	if err := validSettingsTopology(topology); err != nil {
		return Settings{}, err
	}
	validation := settingsValidation{files: make(map[string]snapshot.ContentID), fields: make(map[SettingSource]any)}
	profiles, instructions, err := validation.catalogs(facts)
	if err != nil {
		return Settings{}, err
	}
	defaults := facts.Defaults
	if err := validation.selectionDefaults(defaults, facts.Ceilings); err != nil {
		return Settings{}, err
	}
	// Defaults must be valid even when every Manager overrides a role.
	if _, err := validation.role(defaults.Manager, RoleOverride{}, profiles, instructions); err != nil {
		return Settings{}, fmt.Errorf("default manager: %w", err)
	}
	if _, err := validation.role(defaults.Reviewer, RoleOverride{}, profiles, instructions); err != nil {
		return Settings{}, fmt.Errorf("default reviewer: %w", err)
	}
	overrides, err := validation.exactOverrides(topology, facts.Overrides)
	if err != nil {
		return Settings{}, err
	}
	result := Settings{modelID: topology.modelID, modelDigest: topology.modelDigest, index: make(map[string]int), ceilings: facts.Ceilings}
	result.ceilings.CreationRoots.Value = slices.Clone(facts.Ceilings.CreationRoots.Value)
	for _, slot := range topology.managers {
		override := overrides[slot.Package]
		if err := validation.managerSelection(slot.Package, defaults, override, facts.Ceilings); err != nil {
			return Settings{}, err
		}
		manager := ManagerSettings{
			Package: slot.Package, StepTimeout: replaceSetting(defaults.StepTimeout, override.StepTimeout),
			RejectionLimit: replaceSetting(defaults.RejectionLimit, override.RejectionLimit),
			CreationRoots:  replaceSetting(defaults.CreationRoots, override.CreationRoots), BatchNoChangeReviews: defaults.BatchNoChangeReviews,
		}
		manager.Manager, err = validation.role(defaults.Manager, override.Manager, profiles, instructions)
		if err != nil {
			return Settings{}, fmt.Errorf("Manager %q manager role: %w", slot.Package, err)
		}
		manager.Reviewer, err = validation.role(defaults.Reviewer, override.Reviewer, profiles, instructions)
		if err != nil {
			return Settings{}, fmt.Errorf("Manager %q reviewer role: %w", slot.Package, err)
		}
		result.index[slot.Package] = len(result.managers)
		result.managers = append(result.managers, cloneManagerSettings(manager))
	}
	result.configuration = validation
	return result, nil
}

func replaceSetting[T any](base Setting[T], override *Setting[T]) Setting[T] {
	if override != nil {
		return *override
	}
	return base
}
