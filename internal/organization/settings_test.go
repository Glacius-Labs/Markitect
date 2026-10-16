package organization_test

import (
	"bytes"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestSettingsDefaultsCoverAllLocalManagers(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	settings := requireSettings(t, topology, facts)
	var paths []string
	for _, manager := range settings.Managers() {
		paths = append(paths, manager.Package)
		if manager.StepTimeout != facts.Defaults.StepTimeout || manager.ReviewRounds != facts.Defaults.ReviewRounds || manager.BatchNoChangeReviews != facts.Defaults.BatchNoChangeReviews || !reflect.DeepEqual(manager.CreationRoots, facts.Defaults.CreationRoots) {
			t.Errorf("%s did not receive complete explicit defaults: %+v", manager.Package, manager)
		}
		if manager.Manager.Profile.Value != facts.Profiles[0] || manager.Reviewer.Profile.Value != facts.Profiles[1] {
			t.Errorf("%s role profile identities or sources changed", manager.Package)
		}
	}
	if want := []string{".", "empty", "lending", "lending/reminders", "software"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("local Manager settings/order: %q, want %q", paths, want)
	}
	if _, found := settings.Manager("markitect/core"); found {
		t.Fatal("imported Package acquired settings")
	}
	if settings.ModelID() != topology.ModelID() || settings.ModelDigest() != topology.ModelDigest() || !reflect.DeepEqual(settings.Ceilings(), facts.Ceilings) {
		t.Fatal("model binding or explainable project ceilings lost")
	}
	// The locally empty spelling 'software' remains local despite the imported
	// populated software vocabulary. Empty leaves still receive defaults.
	if got := managerSettings(t, settings, "software"); got.Manager.Profile.Value.Name != "writer" {
		t.Fatal("local empty Package was hidden by imported vocabulary")
	}
}

func TestSettingsForRootOnlyModelAndExactUnicodeOverrides(t *testing.T) {
	rootOnly := requireTopology(t, compileFixture(t, localFixture(t, nil)))
	facts := settingsFixture(t)
	facts.Defaults.BatchNoChangeReviews.Value = true
	settings := requireSettings(t, rootOnly, facts)
	if managers := settings.Managers(); len(managers) != 1 || managers[0].Package != "." || !managers[0].BatchNoChangeReviews.Value {
		t.Fatalf("explicit defaults did not resolve for the ordinary root alone: %+v", managers)
	}
	topology := requireTopology(t, compileFixture(t, localFixture(t, map[string][]string{"é/子": {"item"}}, "software")))
	facts.Overrides = []organization.PackageOverride{
		{Package: "é/子", Manager: organization.RoleOverride{Profile: settingPointer("alternate", "/packages/é~1子/profiles/manager")}},
		{Package: "software", ReviewRounds: settingPointer(3, "/packages/software/reviewRounds")},
	}
	settings = requireSettings(t, topology, facts)
	if got := managerSettings(t, settings, "é/子"); got.Manager.Profile.Value.Name != "alternate" || !got.BatchNoChangeReviews.Value {
		t.Fatal("exact Unicode override or project-wide batch default lost")
	}
	if got := managerSettings(t, settings, "software"); got.ReviewRounds.Value != 3 {
		t.Fatal("local empty spelling could not override its settings")
	}
	facts.Overrides[0].Package = "e\u0301/子"
	refused, err := organization.ResolveSettings(topology, facts)
	if err == nil {
		t.Fatal("Package lookup silently normalized a nonexact Unicode override")
	}
	assertZeroSettings(t, refused)
}

func TestSettingsExactOverridesReplaceAndAppendWithoutInheritance(t *testing.T) {
	facts := settingsFixture(t)
	facts.Overrides = []organization.PackageOverride{
		{Package: ".", Manager: organization.RoleOverride{Profile: settingPointer("alternate", "/packages/./profiles/manager")}},
		{
			Package: "lending",
			Manager: organization.RoleOverride{
				Profile:      settingPointer("alternate", "/packages/lending/profiles/manager"),
				Instructions: settingPointer([]string{"local", "style", "empty"}, "/packages/lending/instructions/manager"),
			},
			Reviewer:      organization.RoleOverride{Instructions: settingPointer([]string{}, "/packages/lending/instructions/reviewer")},
			StepTimeout:   settingPointer(8*time.Minute, "/packages/lending/stepTimeout"),
			ReviewRounds:  settingPointer(3, "/packages/lending/reviewRounds"),
			CreationRoots: settingPointer([]snapshot.DirectoryBoundary{directory(t, "docs")}, "/packages/lending/creationRoots"),
		},
		{Package: "empty", CreationRoots: settingPointer([]snapshot.DirectoryBoundary{}, "/packages/empty/creationRoots")},
	}
	settings := requireSettings(t, settingsTopology(t), facts)
	lending := managerSettings(t, settings, "lending")
	if lending.Manager.Profile.Value.Name != "alternate" || lending.Reviewer.Profile.Value.Name != "critic" || lending.StepTimeout.Value != 8*time.Minute || lending.ReviewRounds.Value != 3 {
		t.Fatalf("exact replacements failed: %+v", lending)
	}
	if roots := lending.CreationRoots.Value; len(roots) != 1 || roots[0].String() != "docs" {
		t.Fatalf("root lists were merged rather than replaced: %+v", roots)
	}
	var ids []string
	for _, fact := range lending.Manager.Instructions {
		ids = append(ids, fact.ID)
	}
	if want := []string{"style", "local", "style", "empty"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("instruction order/repeats changed: %q, want %q", ids, want)
	}
	if want := []organization.SettingSource{facts.Defaults.Manager.Instructions.Source, facts.Overrides[1].Manager.Instructions.Source}; !reflect.DeepEqual(lending.Manager.InstructionSources, want) {
		t.Fatal("instruction selection provenance lost")
	}
	if len(lending.Reviewer.Instructions) != 1 || len(lending.Reviewer.InstructionSources) != 2 {
		t.Fatal("an empty appended instruction list cleared defaults or lost its source")
	}
	if lending.Manager.Profile.Source != facts.Overrides[1].Manager.Profile.Source || lending.Manager.Profile.Value.Source != facts.Profiles[2].Source || lending.StepTimeout.Source != facts.Overrides[1].StepTimeout.Source || lending.CreationRoots.Source != facts.Overrides[1].CreationRoots.Source {
		t.Fatal("exact selection/catalog field provenance changed")
	}
	for _, path := range []string{"lending/reminders", "software"} {
		got := managerSettings(t, settings, path)
		if got.Manager.Profile.Value.Name != "writer" || got.StepTimeout.Value != 5*time.Minute || got.ReviewRounds.Value != 2 || got.CreationRoots.Value[0].String() != "real" || len(got.Manager.Instructions) != 1 {
			t.Errorf("%s inherited root/ancestor overrides: %+v", path, got)
		}
	}
	if got := managerSettings(t, settings, "."); got.Manager.Profile.Value.Name != "alternate" || got.StepTimeout.Value != 5*time.Minute {
		t.Fatal("ordinary root did not receive only its own override")
	}
	empty := managerSettings(t, settings, "empty")
	if len(empty.CreationRoots.Value) != 0 || empty.CreationRoots.Source != facts.Overrides[2].CreationRoots.Source {
		t.Fatal("explicit empty root replacement treated as absent")
	}
}

func TestSettingsPreserveOriginalInstructionBytesAndOwnAllViews(t *testing.T) {
	facts := settingsFixture(t)
	facts.Overrides = []organization.PackageOverride{{Package: "lending", Manager: organization.RoleOverride{Instructions: settingPointer([]string{"style"}, "/packages/lending/instructions/manager")}, CreationRoots: settingPointer([]snapshot.DirectoryBoundary{directory(t, "docs")}, "/packages/lending/creationRoots")}}
	settings := requireSettings(t, settingsTopology(t), facts)
	want := settings.Managers()
	wantCeilings := settings.Ceilings()
	if got := managerSettings(t, settings, "lending").Manager.Instructions[0].Bytes; !bytes.Equal(got, []byte("\xef\xbb\xbfUse concise reports.\r\nKeep source references.")) {
		t.Fatalf("original BOM, CRLF or final bytes rewritten: %q", got)
	}
	facts.Instructions[0].Bytes[0] = '!'
	facts.Profiles[0].Identity = "changed"
	facts.Defaults.Manager.Instructions.Value[0] = "missing"
	facts.Defaults.CreationRoots.Value[0] = directory(t, "changed")
	facts.Ceilings.CreationRoots.Value[0] = directory(t, "changed")
	facts.Overrides[0].Manager.Instructions.Value[0] = "missing"
	facts.Overrides[0].CreationRoots.Value[0] = directory(t, "changed")
	facts.Overrides[0].CreationRoots.Source.Field = "changed"
	all := settings.Managers()
	all[0].Manager.Instructions[0].Bytes[0] = '?'
	all[0].Manager.InstructionSources[0].Field = "changed"
	all[0].CreationRoots.Value[0] = directory(t, "changed")
	manager := managerSettings(t, settings, "lending")
	manager.Manager.Instructions[0].Bytes[0] = '?'
	if bytes.Equal(manager.Manager.Instructions[0].Bytes, manager.Manager.Instructions[1].Bytes) {
		t.Fatal("repeated instruction output entries alias each other's bytes")
	}
	manager.Reviewer.Instructions[0].Bytes[0] = '?'
	manager.CreationRoots.Value[0] = directory(t, "changed")
	ceilings := settings.Ceilings()
	ceilings.CreationRoots.Value[0] = directory(t, "changed")
	if !reflect.DeepEqual(settings.Managers(), want) || !reflect.DeepEqual(settings.Ceilings(), wantCeilings) {
		t.Fatal("mutating input/output slices or instruction bytes changed fixed Settings")
	}
}

func TestSettingsDeterminismAcrossCatalogAndOverrideEnumeration(t *testing.T) {
	topology := settingsTopology(t)
	facts := settingsFixture(t)
	facts.Overrides = []organization.PackageOverride{{Package: ".", ReviewRounds: settingPointer(3, "/packages/./reviewRounds")}, {Package: "empty", StepTimeout: settingPointer(time.Minute, "/packages/empty/stepTimeout")}}
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

func TestSettingsCreationDirectoriesUseExactInclusiveAreas(t *testing.T) {
	topology := settingsTopology(t)
	for _, test := range []struct {
		allowed, selected string
		accept            bool
	}{
		{"real", "real", true}, {"real", "real/sub", true}, {"real", "real-other", false},
		{"real/sub", "real", false}, {"real", ".", false}, {".", ".", true}, {".", "real", true},
		{"real", "Real", false}, {"é", "é/子", true}, {"é", "e\u0301", false},
	} {
		t.Run(test.allowed+"->"+test.selected, func(t *testing.T) {
			facts := settingsFixture(t)
			facts.Ceilings.CreationRoots.Value = []snapshot.DirectoryBoundary{directory(t, test.allowed)}
			facts.Defaults.CreationRoots.Value = []snapshot.DirectoryBoundary{directory(t, test.selected)}
			settings, err := organization.ResolveSettings(topology, facts)
			if (err == nil) != test.accept {
				t.Fatalf("directory ceiling result: accept=%v err=%v", test.accept, err)
			}
			if !test.accept {
				assertZeroSettings(t, settings)
			} else if got := managerSettings(t, settings, ".").CreationRoots.Value[0]; got.String() != test.selected {
				t.Fatal("directory spelling changed")
			}
		})
	}
	// An explicit empty ceiling and selection permit no creation areas; empty
	// is not an implicit unrestricted repository-root boundary.
	facts := settingsFixture(t)
	facts.Ceilings.CreationRoots.Value = nil
	facts.Defaults.CreationRoots.Value = nil
	if got := managerSettings(t, requireSettings(t, topology, facts), "."); len(got.CreationRoots.Value) != 0 {
		t.Fatal("empty creation selection became unrestricted")
	}
}

func assertZeroSettings(t *testing.T, settings organization.Settings) {
	t.Helper()
	if settings.ModelID() != "" || settings.ModelDigest() != "" || len(settings.Managers()) != 0 || !reflect.DeepEqual(settings.Ceilings(), organization.SettingsCeilings{}) {
		t.Fatal("refusal exposed partial Manager settings, model binding or ceilings")
	}
	if _, found := settings.Manager("."); found {
		t.Fatal("refusal exposed root settings")
	}
}
