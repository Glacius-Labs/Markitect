package organization_test

import (
	"crypto/sha256"
	"testing"
	"time"

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
		{"missing default manager profile", func(f *organization.SettingsFacts) { f.Defaults.Manager.Profile = organization.Setting[string]{} }},
		{"missing default reviewer profile", func(f *organization.SettingsFacts) { f.Defaults.Reviewer.Profile = organization.Setting[string]{} }},
		{"missing default manager instructions", func(f *organization.SettingsFacts) {
			f.Defaults.Manager.Instructions = organization.Setting[[]string]{}
		}},
		{"missing default reviewer instructions", func(f *organization.SettingsFacts) {
			f.Defaults.Reviewer.Instructions = organization.Setting[[]string]{}
		}},
		{"missing default timeout", func(f *organization.SettingsFacts) { f.Defaults.StepTimeout = organization.Setting[time.Duration]{} }},
		{"missing default rejection limit", func(f *organization.SettingsFacts) { f.Defaults.RejectionLimit = organization.Setting[int]{} }},
		{"missing default roots", func(f *organization.SettingsFacts) {
			f.Defaults.CreationRoots = organization.Setting[[]snapshot.DirectoryBoundary]{}
		}},
		{"missing default batch flag", func(f *organization.SettingsFacts) { f.Defaults.BatchNoChangeReviews = organization.Setting[bool]{} }},
		{"missing timeout ceiling", func(f *organization.SettingsFacts) { f.Ceilings.StepTimeout = organization.Setting[time.Duration]{} }},
		{"missing rejection-limit ceiling", func(f *organization.SettingsFacts) { f.Ceilings.RejectionLimit = organization.Setting[int]{} }},
		{"missing roots ceiling", func(f *organization.SettingsFacts) {
			f.Ceilings.CreationRoots = organization.Setting[[]snapshot.DirectoryBoundary]{}
		}},
		{"missing source file", func(f *organization.SettingsFacts) { f.Defaults.StepTimeout.Source.File = "" }},
		{"missing source content ID", func(f *organization.SettingsFacts) { f.Defaults.StepTimeout.Source.ContentID = snapshot.ContentID{} }},
		{"missing source field", func(f *organization.SettingsFacts) { f.Defaults.StepTimeout.Source.Field = "" }},
		{"conflicting file content identities", func(f *organization.SettingsFacts) { f.Defaults.StepTimeout.Source.ContentID[0] ^= 1 }},
		{"zero default timeout", func(f *organization.SettingsFacts) { f.Defaults.StepTimeout.Value = 0 }},
		{"negative default timeout", func(f *organization.SettingsFacts) { f.Defaults.StepTimeout.Value = -time.Minute }},
		{"zero default rejection limit", func(f *organization.SettingsFacts) { f.Defaults.RejectionLimit.Value = 0 }},
		{"negative default rejection limit", func(f *organization.SettingsFacts) { f.Defaults.RejectionLimit.Value = -1 }},
		{"default timeout over ceiling", func(f *organization.SettingsFacts) { f.Defaults.StepTimeout.Value = 11 * time.Minute }},
		{"default rejection limit over ceiling", func(f *organization.SettingsFacts) { f.Defaults.RejectionLimit.Value = 4 }},
		{"zero timeout ceiling", func(f *organization.SettingsFacts) { f.Ceilings.StepTimeout.Value = 0 }},
		{"negative timeout ceiling", func(f *organization.SettingsFacts) { f.Ceilings.StepTimeout.Value = -1 }},
		{"zero rejection-limit ceiling", func(f *organization.SettingsFacts) { f.Ceilings.RejectionLimit.Value = 0 }},
		{"negative rejection-limit ceiling", func(f *organization.SettingsFacts) { f.Ceilings.RejectionLimit.Value = -1 }},
		{"zero default directory", func(f *organization.SettingsFacts) { f.Defaults.CreationRoots.Value = []snapshot.DirectoryBoundary{{}} }},
		{"zero ceiling directory", func(f *organization.SettingsFacts) { f.Ceilings.CreationRoots.Value = []snapshot.DirectoryBoundary{{}} }},
		{"duplicate default directory", func(f *organization.SettingsFacts) {
			f.Defaults.CreationRoots.Value = append(f.Defaults.CreationRoots.Value, f.Defaults.CreationRoots.Value[0])
		}},
		{"duplicate ceiling directory", func(f *organization.SettingsFacts) {
			f.Ceilings.CreationRoots.Value = append(f.Ceilings.CreationRoots.Value, f.Ceilings.CreationRoots.Value[0])
		}},
		{"default outside directory ceiling", func(f *organization.SettingsFacts) {
			f.Defaults.CreationRoots.Value = []snapshot.DirectoryBoundary{directory(t, "real-other")}
		}},
		{"empty directory ceiling with nonempty default", func(f *organization.SettingsFacts) { f.Ceilings.CreationRoots.Value = nil }},
		{"missing profile catalog", func(f *organization.SettingsFacts) { f.Profiles = nil }},
		{"unknown profile", func(f *organization.SettingsFacts) { f.Defaults.Manager.Profile.Value = "missing" }},
		{"empty profile reference", func(f *organization.SettingsFacts) { f.Defaults.Manager.Profile.Value = "" }},
		{"duplicate profile name", func(f *organization.SettingsFacts) { f.Profiles = append(f.Profiles, f.Profiles[0]) }},
		{"empty unused profile name", func(f *organization.SettingsFacts) { f.Profiles[2].Name = "" }},
		{"empty unused profile identity", func(f *organization.SettingsFacts) { f.Profiles[2].Identity = "" }},
		{"unsourced unused profile", func(f *organization.SettingsFacts) { f.Profiles[2].Source = organization.SettingSource{} }},
		{"conflicting profile field", func(f *organization.SettingsFacts) { f.Profiles[2].Source = f.Profiles[0].Source }},
		{"missing instruction catalog", func(f *organization.SettingsFacts) { f.Instructions = nil }},
		{"unknown instruction", func(f *organization.SettingsFacts) { f.Defaults.Manager.Instructions.Value = []string{"missing"} }},
		{"empty instruction reference", func(f *organization.SettingsFacts) { f.Defaults.Reviewer.Instructions.Value = []string{""} }},
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
		{"zero override timeout", organization.PackageOverride{StepTimeout: settingPointer(time.Duration(0), "/packages/lending/stepTimeout")}},
		{"negative override timeout", organization.PackageOverride{StepTimeout: settingPointer(-time.Minute, "/packages/lending/stepTimeout")}},
		{"override timeout ceiling", organization.PackageOverride{StepTimeout: settingPointer(11*time.Minute, "/packages/lending/stepTimeout")}},
		{"zero override rejection limit", organization.PackageOverride{RejectionLimit: settingPointer(0, "/packages/lending/rejectionLimit")}},
		{"negative override rejection limit", organization.PackageOverride{RejectionLimit: settingPointer(-1, "/packages/lending/rejectionLimit")}},
		{"override rejection limit ceiling", organization.PackageOverride{RejectionLimit: settingPointer(4, "/packages/lending/rejectionLimit")}},
		{"override root ceiling", organization.PackageOverride{CreationRoots: settingPointer([]snapshot.DirectoryBoundary{directory(t, "real-other")}, "/packages/lending/creationRoots")}},
		{"unsourced present override", organization.PackageOverride{RejectionLimit: &organization.Setting[int]{Value: 2}}},
		{"unknown manager profile override", organization.PackageOverride{Manager: organization.RoleOverride{Profile: settingPointer("missing", "/packages/lending/profiles/manager")}}},
		{"unknown reviewer profile override", organization.PackageOverride{Reviewer: organization.RoleOverride{Profile: settingPointer("missing", "/packages/lending/profiles/reviewer")}}},
		{"unsourced appended instructions", organization.PackageOverride{Manager: organization.RoleOverride{Instructions: &organization.Setting[[]string]{}}}},
		{"unknown appended instruction", organization.PackageOverride{Reviewer: organization.RoleOverride{Instructions: settingPointer([]string{"missing"}, "/packages/lending/instructions/reviewer")}}},
		{"conflicting sourced override", organization.PackageOverride{StepTimeout: settingPointer(8*time.Minute, "/defaults/stepTimeout")}},
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
		facts.Overrides = append(facts.Overrides, organization.PackageOverride{Package: manager.Package, Manager: organization.RoleOverride{Profile: settingPointer("alternate", "/packages/"+manager.Package+"/profiles/manager")}})
	}
	facts.Defaults.Manager.Profile.Value = "missing"
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
	if string(managerSettings(t, settings, ".").Manager.Instructions[0].Value.Bytes) != string(facts.Instructions[0].Bytes) {
		t.Fatal("resolver rewrote or interpreted original instruction prose")
	}
}
