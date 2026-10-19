package organization_test

import (
	"bytes"
	"reflect"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func TestSettingsDefaultsCoverAllLocalManagers(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	settings := requireSettings(t, topology, facts)
	var paths []string
	for _, manager := range settings.Managers() {
		paths = append(paths, manager.Package)
		if manager.Execute.Profile.Value != facts.Profiles[0] || manager.Verify.Profile.Value != facts.Profiles[1] {
			t.Errorf("%s role profile identities or sources changed", manager.Package)
		}
	}
	if want := []string{".", "empty", "lending", "lending/reminders", "software"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("local Manager settings/order: %q, want %q", paths, want)
	}
	if _, found := settings.Manager("markitect/core"); found {
		t.Fatal("imported Package acquired settings")
	}
	if settings.ModelID() != topology.ModelID() || settings.ModelDigest() != topology.ModelDigest() {
		t.Fatal("model binding lost")
	}
	// The locally empty spelling 'software' remains local despite the imported
	// populated software vocabulary. Empty leaves still receive defaults.
	if got := managerSettings(t, settings, "software"); got.Execute.Profile.Value.Name != "writer" {
		t.Fatal("local empty Package was hidden by imported vocabulary")
	}
}

func TestSettingsForRootOnlyModelAndExactUnicodeOverrides(t *testing.T) {
	rootOnly := requireTopology(t, compileFixture(t, localFixture(t, nil)))
	facts := settingsFixture(t)
	settings := requireSettings(t, rootOnly, facts)
	if managers := settings.Managers(); len(managers) != 1 || managers[0].Package != "." {
		t.Fatalf("explicit defaults did not resolve for the ordinary root alone: %+v", managers)
	}
	topology := requireTopology(t, compileFixture(t, localFixture(t, map[string][]string{"é/子": {"item"}}, "software")))
	facts.Overrides = []organization.PackageOverride{
		{Package: "é/子", Execute: organization.RoleOverride{Profile: settingPointer("alternate", "/packages/é~1子/profiles/execute")}},
		{Package: "software", Verify: organization.RoleOverride{Profile: settingPointer("alternate", "/packages/software/profiles/verify")}},
	}
	settings = requireSettings(t, topology, facts)
	if got := managerSettings(t, settings, "é/子"); got.Execute.Profile.Value.Name != "alternate" || got.Verify.Profile.Value.Name != "critic" {
		t.Fatal("exact Unicode override lost")
	}
	if got := managerSettings(t, settings, "software"); got.Verify.Profile.Value.Name != "alternate" {
		t.Fatal("local empty spelling could not override its settings")
	}
	facts.Overrides[0].Package = "é/子"
	refused, err := organization.ResolveSettings(topology, facts)
	if err == nil {
		t.Fatal("Package lookup silently normalized a nonexact Unicode override")
	}
	assertZeroSettings(t, refused)
}

func TestSettingsExactOverridesReplaceAndAppendWithoutInheritance(t *testing.T) {
	facts := settingsFixture(t)
	facts.Overrides = []organization.PackageOverride{
		{Package: ".", Execute: organization.RoleOverride{Profile: settingPointer("alternate", "/packages/./profiles/execute")}},
		{
			Package: "lending",
			Execute: organization.RoleOverride{
				Profile:      settingPointer("alternate", "/packages/lending/profiles/execute"),
				Instructions: settingPointer([]string{"local", "style", "empty"}, "/packages/lending/instructions/execute"),
			},
			Verify: organization.RoleOverride{Instructions: settingPointer([]string{}, "/packages/lending/instructions/verify")},
		},
	}
	settings := requireSettings(t, settingsTopology(t), facts)
	lending := managerSettings(t, settings, "lending")
	if lending.Execute.Profile.Value.Name != "alternate" || lending.Verify.Profile.Value.Name != "critic" {
		t.Fatalf("exact replacements failed: %+v", lending)
	}
	var ids []string
	for _, fact := range lending.Execute.Instructions {
		ids = append(ids, fact.Value.ID)
	}
	if want := []string{"style", "local", "style", "empty"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("instruction order/repeats changed: %q, want %q", ids, want)
	}
	// The repeated style ID has the same original-file source but distinct
	// selecting fields. A list of contributing sources alone cannot establish
	// which occurrence came from which input list.
	first, repeated := lending.Execute.Instructions[0], lending.Execute.Instructions[2]
	if first.Source != facts.Defaults.Execute.Instructions.Source || repeated.Source != facts.Overrides[1].Execute.Instructions.Source || first.Value.Source != facts.Instructions[0].Source || repeated.Value.Source != first.Value.Source || !bytes.Equal(first.Value.Bytes, repeated.Value.Bytes) {
		t.Fatal("repeated instruction occurrences lost their distinct selection sources or shared original content binding")
	}
	if want := []organization.SettingSource{facts.Defaults.Execute.Instructions.Source, facts.Overrides[1].Execute.Instructions.Source}; !reflect.DeepEqual(lending.Execute.InstructionSources, want) {
		t.Fatal("instruction selection provenance lost")
	}
	if len(lending.Verify.Instructions) != 1 || len(lending.Verify.InstructionSources) != 2 {
		t.Fatal("an empty appended instruction list cleared defaults or lost its source")
	}
	if lending.Execute.Profile.Source != facts.Overrides[1].Execute.Profile.Source || lending.Execute.Profile.Value.Source != facts.Profiles[2].Source {
		t.Fatal("exact selection/catalog field provenance changed")
	}
	for _, path := range []string{"lending/reminders", "software"} {
		got := managerSettings(t, settings, path)
		if got.Execute.Profile.Value.Name != "writer" || len(got.Execute.Instructions) != 1 {
			t.Errorf("%s inherited root/ancestor overrides: %+v", path, got)
		}
	}
	if got := managerSettings(t, settings, "."); got.Execute.Profile.Value.Name != "alternate" || got.Verify.Profile.Value.Name != "critic" {
		t.Fatal("ordinary root did not receive only its own override")
	}
}

func TestSettingsPreserveOriginalInstructionBytesAndOwnAllViews(t *testing.T) {
	facts := settingsFixture(t)
	facts.Overrides = []organization.PackageOverride{{Package: "lending", Execute: organization.RoleOverride{Instructions: settingPointer([]string{"style"}, "/packages/lending/instructions/execute")}}}
	settings := requireSettings(t, settingsTopology(t), facts)
	want := settings.Managers()
	if got := managerSettings(t, settings, "lending").Execute.Instructions[0].Value.Bytes; !bytes.Equal(got, []byte("\xef\xbb\xbfUse concise reports.\r\nKeep source references.")) {
		t.Fatalf("original BOM, CRLF or final bytes rewritten: %q", got)
	}
	facts.Instructions[0].Bytes[0] = '!'
	facts.Profiles[0].Identity = "changed"
	facts.Defaults.Execute.Instructions.Value[0] = "missing"
	facts.Overrides[0].Execute.Instructions.Value[0] = "missing"
	facts.Overrides[0].Execute.Instructions.Source.Field = "changed"
	all := settings.Managers()
	all[0].Execute.Instructions[0].Value.Bytes[0] = '?'
	all[0].Execute.Instructions[0].Source.Field = "changed"
	all[0].Execute.InstructionSources[0].Field = "changed"
	manager := managerSettings(t, settings, "lending")
	manager.Execute.Instructions[0].Value.Bytes[0] = '?'
	if bytes.Equal(manager.Execute.Instructions[0].Value.Bytes, manager.Execute.Instructions[1].Value.Bytes) {
		t.Fatal("repeated instruction output entries alias each other's bytes")
	}
	manager.Verify.Instructions[0].Value.Bytes[0] = '?'
	if !reflect.DeepEqual(settings.Managers(), want) {
		t.Fatal("mutating input/output slices or instruction bytes changed fixed Settings")
	}
}

func TestSettingsDeterminismAcrossCatalogAndOverrideEnumeration(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	facts.Overrides = []organization.PackageOverride{
		{Package: ".", Execute: organization.RoleOverride{Profile: settingPointer("alternate", "/packages/./profiles/execute")}},
		{Package: "empty", Verify: organization.RoleOverride{Profile: settingPointer("alternate", "/packages/empty/profiles/verify")}},
	}
	want := requireSettings(t, topology, facts)
	slices.Reverse(facts.Overrides)
	slices.Reverse(facts.Profiles)
	slices.Reverse(facts.Instructions)
	before := facts.Overrides[0]
	got := requireSettings(t, topology, facts)
	if !reflect.DeepEqual(got.Managers(), want.Managers()) || !reflect.DeepEqual(before, facts.Overrides[0]) {
		t.Fatal("resolution depends on catalog/key order or mutates supplied overrides")
	}
}

func assertZeroSettings(t *testing.T, settings organization.Settings) {
	t.Helper()
	if settings.ModelID() != "" || settings.ModelDigest() != "" || len(settings.Managers()) != 0 {
		t.Fatal("refusal exposed partial Manager settings or model binding")
	}
	if _, found := settings.Manager("."); found {
		t.Fatal("refusal exposed root settings")
	}
}
