package organization_test

import (
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestValidateSettingsSelectionsDoesNotNeedRoleCatalogs(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	facts.Profiles = nil
	facts.Instructions = nil
	facts.Defaults.Manager.Profile.Value = "unresolved-manager"
	facts.Defaults.Reviewer.Profile.Value = "unresolved-reviewer"
	if err := organization.ValidateSettingsSelections(topology, facts.Defaults, facts.Overrides, facts.Ceilings); err != nil {
		t.Fatalf("selection validation depended on role catalogs: %v", err)
	}
}

func TestValidateSettingsSelectionsChecksDefaultsEvenWhenEveryManagerOverrides(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	facts.Defaults.RejectionLimit.Value = 3
	facts.Ceilings.RejectionLimit.Value = 2
	for _, manager := range topology.Managers() {
		facts.Overrides = append(facts.Overrides, organization.PackageOverride{
			Package:        manager.Package,
			RejectionLimit: settingPointer(2, "/packages/"+manager.Package+"/rejectionLimit"),
		})
	}
	if err := organization.ValidateSettingsSelections(topology, facts.Defaults, facts.Overrides, facts.Ceilings); err == nil {
		t.Fatal("invalid default was accepted because every Manager overrides it")
	}
}

func TestValidateSettingsSelectionsAcceptsWithinCeilingExactOverride(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	facts.Defaults.RejectionLimit.Value = 2
	facts.Ceilings.RejectionLimit.Value = 3
	facts.Overrides = []organization.PackageOverride{{
		Package:        "lending",
		RejectionLimit: settingPointer(3, "/packages/lending/rejectionLimit"),
	}}
	if err := organization.ValidateSettingsSelections(topology, facts.Defaults, facts.Overrides, facts.Ceilings); err != nil {
		t.Fatalf("valid exact override rejected: %v", err)
	}

	// The existing resolver must continue to apply only the exact local value.
	resolvedFacts := facts
	resolvedFacts.Profiles = settingsFixture(t).Profiles
	resolvedFacts.Instructions = settingsFixture(t).Instructions
	settings := requireSettings(t, topology, resolvedFacts)
	if got := managerSettings(t, settings, "lending").RejectionLimit.Value; got != 3 {
		t.Fatalf("exact override got %d, want 3", got)
	}
	if got := managerSettings(t, settings, "lending/reminders").RejectionLimit.Value; got != 2 {
		t.Fatalf("descendant inherited parent override: got %d, want default 2", got)
	}
}

func TestValidateSettingsSelectionsRejectsNonlocalAndDuplicateOverrides(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	for name, overrides := range map[string][]organization.PackageOverride{
		"imported": {{Package: "markitect/core"}},
		"unknown":  {{Package: "catalog"}},
		"duplicate": {
			{Package: "lending"},
			{Package: "lending"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := organization.ValidateSettingsSelections(topology, facts.Defaults, overrides, facts.Ceilings); err == nil {
				t.Fatal("invalid exact override set accepted")
			}
		})
	}
}

func TestValidateSettingsSelectionsChecksSourceConsistencyAndCeilings(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	t.Run("conflicting default and override source", func(t *testing.T) {
		override := setting(7*time.Minute, facts.Defaults.StepTimeout.Source.Field)
		override.Source = facts.Defaults.StepTimeout.Source
		if err := organization.ValidateSettingsSelections(topology, facts.Defaults, []organization.PackageOverride{{Package: "lending", StepTimeout: &override}}, facts.Ceilings); err == nil {
			t.Fatal("conflicting values asserted for one source field accepted")
		}
	})
	t.Run("outside creation-root ceiling", func(t *testing.T) {
		facts.Defaults.CreationRoots = setting([]snapshot.DirectoryBoundary{directory(t, "elsewhere")}, facts.Defaults.CreationRoots.Source.Field)
		if err := organization.ValidateSettingsSelections(topology, facts.Defaults, nil, facts.Ceilings); err == nil {
			t.Fatal("creation root outside project ceiling accepted")
		}
	})
	t.Run("zero topology", func(t *testing.T) {
		if err := organization.ValidateSettingsSelections(organization.Topology{}, facts.Defaults, nil, facts.Ceilings); err == nil {
			t.Fatal("zero topology accepted")
		}
	})
}
