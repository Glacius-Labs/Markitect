package organization

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ResolveSettings resolves supplied project defaults and only the exact local
// Manager override. There is no ancestor inheritance, deep merge, interpolation
// or implicit profile/instruction acquisition. It renders and executes nothing,
// and does not assess instruction prose against model obligations.
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
	// Defaults must be valid even when every Manager overrides a role.
	if _, err := validation.role(defaults.Execute, RoleOverride{}, profiles, instructions); err != nil {
		return Settings{}, fmt.Errorf("default execute: %w", err)
	}
	if _, err := validation.role(defaults.Verify, RoleOverride{}, profiles, instructions); err != nil {
		return Settings{}, fmt.Errorf("default verify: %w", err)
	}
	overrides, err := validation.exactOverrides(topology, facts.Overrides)
	if err != nil {
		return Settings{}, err
	}
	result := Settings{modelID: topology.modelID, modelDigest: topology.modelDigest, index: make(map[string]int)}
	for _, slot := range topology.managers {
		override := overrides[slot.Package]
		manager := ManagerSettings{Package: slot.Package}
		manager.Execute, err = validation.role(defaults.Execute, override.Execute, profiles, instructions)
		if err != nil {
			return Settings{}, fmt.Errorf("Manager %q execute role: %w", slot.Package, err)
		}
		manager.Verify, err = validation.role(defaults.Verify, override.Verify, profiles, instructions)
		if err != nil {
			return Settings{}, fmt.Errorf("Manager %q verify role: %w", slot.Package, err)
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

func validSettingsTopology(topology Topology) error {
	if topology.modelID == "" || topology.modelDigest == "" || len(topology.managers) == 0 {
		return fmt.Errorf("settings require a successful local topology")
	}
	return nil
}

func (v settingsValidation) exactOverrides(topology Topology, overrides []PackageOverride) (map[string]PackageOverride, error) {
	byPackage := make(map[string]PackageOverride, len(overrides))
	for _, override := range overrides {
		if _, found := topology.index[override.Package]; !found {
			return nil, fmt.Errorf("override is not an exact local Manager: %q", override.Package)
		}
		if _, found := byPackage[override.Package]; found {
			return nil, fmt.Errorf("duplicate Package override %q", override.Package)
		}
		byPackage[override.Package] = override
	}
	return byPackage, nil
}
