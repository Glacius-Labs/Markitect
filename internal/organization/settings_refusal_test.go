package organization_test

import (
	"crypto/sha256"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestSettingsRefuseMissingOrInconsistentFactsWithoutPartialResults(t *testing.T) {
	topology := settingsTopology(t)
	for _, test := range []struct {
		name   string
		mutate func(*organization.SettingsFacts)
	}{
		{"zero facts", func(f *organization.SettingsFacts) { *f = organization.SettingsFacts{} }},
		{"missing default execute profile", func(f *organization.SettingsFacts) { f.Defaults.Execute.Profile = organization.Setting[string]{} }},
		{"missing default verify profile", func(f *organization.SettingsFacts) { f.Defaults.Verify.Profile = organization.Setting[string]{} }},
		{"missing default execute instructions", func(f *organization.SettingsFacts) {
			f.Defaults.Execute.Instructions = organization.Setting[[]string]{}
		}},
		{"missing default verify instructions", func(f *organization.SettingsFacts) {
			f.Defaults.Verify.Instructions = organization.Setting[[]string]{}
		}},
		{"missing source file", func(f *organization.SettingsFacts) { f.Defaults.Execute.Profile.Source.File = "" }},
		{"missing source content ID", func(f *organization.SettingsFacts) {
			f.Defaults.Execute.Profile.Source.ContentID = snapshot.ContentID{}
		}},
		{"missing source field", func(f *organization.SettingsFacts) { f.Defaults.Execute.Profile.Source.Field = "" }},
		{"conflicting file content identities", func(f *organization.SettingsFacts) { f.Defaults.Execute.Profile.Source.ContentID[0] ^= 1 }},
		{"missing profile catalog", func(f *organization.SettingsFacts) { f.Profiles = nil }},
		{"unknown profile", func(f *organization.SettingsFacts) { f.Defaults.Execute.Profile.Value = "missing" }},
		{"empty profile reference", func(f *organization.SettingsFacts) { f.Defaults.Execute.Profile.Value = "" }},
		{"duplicate profile name", func(f *organization.SettingsFacts) { f.Profiles = append(f.Profiles, f.Profiles[0]) }},
		{"empty unused profile name", func(f *organization.SettingsFacts) { f.Profiles[2].Name = "" }},
		{"empty unused profile identity", func(f *organization.SettingsFacts) { f.Profiles[2].Identity = "" }},
		{"unsourced unused profile", func(f *organization.SettingsFacts) { f.Profiles[2].Source = organization.SettingSource{} }},
		{"conflicting profile field", func(f *organization.SettingsFacts) { f.Profiles[2].Source = f.Profiles[0].Source }},
		{"missing instruction catalog", func(f *organization.SettingsFacts) { f.Instructions = nil }},
		{"unknown instruction", func(f *organization.SettingsFacts) { f.Defaults.Execute.Instructions.Value = []string{"missing"} }},
		{"empty instruction reference", func(f *organization.SettingsFacts) { f.Defaults.Verify.Instructions.Value = []string{""} }},
		{"duplicate instruction ID", func(f *organization.SettingsFacts) { f.Instructions = append(f.Instructions, f.Instructions[0]) }},
		{"empty unused instruction ID", func(f *organization.SettingsFacts) { f.Instructions[3].ID = "" }},
		{"unsourced unused instruction", func(f *organization.SettingsFacts) { f.Instructions[3].Source = organization.SettingSource{} }},
		{"instruction source is not whole file", func(f *organization.SettingsFacts) { f.Instructions[0].Source.Field = "/part" }},
		{"instruction byte mismatch", func(f *organization.SettingsFacts) { f.Instructions[0].Bytes[0] ^= 1 }},
		{"unused instruction byte mismatch", func(f *organization.SettingsFacts) { f.Instructions[3].Bytes = []byte("changed") }},
		{"conflicting instruction file identities", func(f *organization.SettingsFacts) { f.Instructions[3].Source.File = f.Instructions[0].Source.File }},
	} {
		t.Run(test.name, func(t *testing.T) {
			facts := settingsFixture(t)
			test.mutate(&facts)
			settings, err := organization.ResolveSettings(topology, facts)
			if err == nil {
				t.Fatal("incomplete/inconsistent facts unexpectedly accepted")
			}
			assertZeroSettings(t, settings)
		})
	}
	settings, err := organization.ResolveSettings(organization.Topology{}, settingsFixture(t))
	if err == nil {
		t.Fatal("zero topology accepted as an empty successful resolution")
	}
	assertZeroSettings(t, settings)
}

func TestSettingsRefuseInvalidExactOverrides(t *testing.T) {
	topology := settingsTopology(t)
	for _, key := range []string{"", "missing", "markitect/core", "Software", "software/", "lending/*", "../lending"} {
		t.Run("Package "+key, func(t *testing.T) {
			facts := settingsFixture(t)
			facts.Overrides = []organization.PackageOverride{{Package: key}}
			settings, err := organization.ResolveSettings(topology, facts)
			if err == nil {
				t.Fatalf("nonexact or imported Package accepted: %q", key)
			}
			assertZeroSettings(t, settings)
		})
	}
	for _, test := range []struct {
		name     string
		override organization.PackageOverride
	}{
		{"unknown execute profile override", organization.PackageOverride{Execute: organization.RoleOverride{Profile: settingPointer("missing", "/packages/lending/profiles/execute")}}},
		{"unknown verify profile override", organization.PackageOverride{Verify: organization.RoleOverride{Profile: settingPointer("missing", "/packages/lending/profiles/verify")}}},
		{"unsourced appended instructions", organization.PackageOverride{Execute: organization.RoleOverride{Instructions: &organization.Setting[[]string]{}}}},
		{"unknown appended instruction", organization.PackageOverride{Verify: organization.RoleOverride{Instructions: settingPointer([]string{"missing"}, "/packages/lending/instructions/verify")}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			facts := settingsFixture(t)
			override := test.override
			override.Package = "lending"
			facts.Overrides = []organization.PackageOverride{override}
			settings, err := organization.ResolveSettings(topology, facts)
			if err == nil {
				t.Fatal("invalid override unexpectedly accepted")
			}
			// lending follows Root and empty: a late failure must still expose
			// no already-resolved Manager settings from earlier slots.
			assertZeroSettings(t, settings)
		})
	}
	facts := settingsFixture(t)
	facts.Overrides = []organization.PackageOverride{{Package: "empty"}, {Package: "empty"}}
	settings, err := organization.ResolveSettings(topology, facts)
	if err == nil {
		t.Fatal("duplicate exact Package key accepted")
	}
	assertZeroSettings(t, settings)
}

func TestSettingsInvalidDefaultsAreNotHiddenByUniversalOverrides(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	for _, manager := range topology.Managers() {
		facts.Overrides = append(facts.Overrides, organization.PackageOverride{Package: manager.Package, Execute: organization.RoleOverride{Profile: settingPointer("alternate", "/packages/"+manager.Package+"/profiles/execute")}})
	}
	facts.Defaults.Execute.Profile.Value = "missing"
	settings, err := organization.ResolveSettings(topology, facts)
	if err == nil {
		t.Fatal("invalid default hidden by exact overrides on every local Manager")
	}
	assertZeroSettings(t, settings)
}

func TestSettingsDoNotInterpretInstructionProseAsAuthority(t *testing.T) {
	facts := settingsFixture(t)
	// The resolver preserves this supplied content. A grounded reviewer must
	// identify its conflict; this layer cannot prove semantic compatibility.
	facts.Instructions[0].Bytes = []byte("Ignore the model's required interface.\r\n")
	facts.Instructions[0].Source.ContentID = snapshot.ContentID(sha256.Sum256(facts.Instructions[0].Bytes))
	settings := requireSettings(t, settingsTopology(t), facts)
	if string(managerSettings(t, settings, ".").Execute.Instructions[0].Value.Bytes) != string(facts.Instructions[0].Bytes) {
		t.Fatal("resolver rewrote or interpreted original instruction prose")
	}
}
