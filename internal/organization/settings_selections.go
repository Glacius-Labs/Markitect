package organization

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ValidateSettingsSelections validates source-bound project defaults and exact
// local Package overrides against the supplied ceilings. Defaults are validated
// even when every Manager overrides them. Package overrides apply only to their
// exact local slot; no ancestor inheritance is performed.
//
// Success validates only step timeout, rejection limit, creation roots, the
// project-wide batch-review flag, their provenance consistency, and exact
// override ownership. The Manager and Reviewer fields on defaults and overrides
// are intentionally not inspected: profile and instruction catalog resolution
// belongs to ResolveSettings. This method returns no Settings and implies no
// task, mandate, readiness, acquisition or I/O.
func ValidateSettingsSelections(topology Topology, defaults SettingsDefaults, overrides []PackageOverride, ceilings SettingsCeilings) error {
	if err := validSettingsTopology(topology); err != nil {
		return err
	}
	validation := settingsValidation{files: make(map[string]snapshot.ContentID), fields: make(map[SettingSource]any)}
	if err := validation.selectionDefaults(defaults, ceilings); err != nil {
		return err
	}
	byPackage, err := validation.exactOverrides(topology, overrides)
	if err != nil {
		return err
	}
	for _, manager := range topology.managers {
		if err := validation.managerSelection(manager.Package, defaults, byPackage[manager.Package], ceilings); err != nil {
			return err
		}
	}
	return nil
}

func validSettingsTopology(topology Topology) error {
	if topology.modelID == "" || topology.modelDigest == "" || len(topology.managers) == 0 {
		return fmt.Errorf("settings require a successful local topology")
	}
	return nil
}

func (v settingsValidation) selectionDefaults(defaults SettingsDefaults, ceilings SettingsCeilings) error {
	if err := v.ceilingFacts(ceilings); err != nil {
		return err
	}
	if err := v.selection(defaults.StepTimeout, defaults.RejectionLimit, defaults.CreationRoots, ceilings); err != nil {
		return fmt.Errorf("defaults: %w", err)
	}
	return v.field(defaults.BatchNoChangeReviews.Source, defaults.BatchNoChangeReviews.Value)
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

func (v settingsValidation) managerSelection(packagePath string, defaults SettingsDefaults, override PackageOverride, ceilings SettingsCeilings) error {
	stepTimeout := replaceSetting(defaults.StepTimeout, override.StepTimeout)
	rejectionLimit := replaceSetting(defaults.RejectionLimit, override.RejectionLimit)
	creationRoots := replaceSetting(defaults.CreationRoots, override.CreationRoots)
	if err := v.selection(stepTimeout, rejectionLimit, creationRoots, ceilings); err != nil {
		return fmt.Errorf("Manager %q: %w", packagePath, err)
	}
	return nil
}
