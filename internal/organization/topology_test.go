package organization_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func requireTopology(t *testing.T, result model.Result) organization.Topology {
	t.Helper()
	topology, err := organization.BuildTopology(result)
	if err != nil {
		t.Fatal(err)
	}
	return topology
}

func refs(pkg string, ids ...string) []model.Identity {
	result := make([]model.Identity, len(ids))
	for i, id := range ids {
		result[i] = model.Identity{Package: pkg, ID: id}
	}
	return result
}

func TestLibraryShapedCompiledTopology(t *testing.T) {
	input := localFixture(t, map[string][]string{
		"catalog":           {"copy", "availability"},
		"lending":           {"return", "loan", "borrow"},
		"lending/reminders": {"assess-overdue"},
		"library":           {"realization", "library", "callable-interface"},
		"members":           {"member", "eligibility"},
	}, "empty")
	compiled := compileFixture(t, input)
	topology := requireTopology(t, compiled)
	want := []organization.Manager{
		{Package: ".", Children: []string{"catalog", "empty", "lending", "library", "members"}},
		{Package: "catalog", Parent: ".", Definitions: refs("catalog", "availability", "copy")},
		{Package: "empty", Parent: "."},
		{Package: "lending", Parent: ".", Children: []string{"lending/reminders"}, Definitions: refs("lending", "borrow", "loan", "return")},
		{Package: "lending/reminders", Parent: "lending", Definitions: refs("lending/reminders", "assess-overdue")},
		{Package: "library", Parent: ".", Definitions: refs("library", "callable-interface", "library", "realization")},
		{Package: "members", Parent: ".", Definitions: refs("members", "eligibility", "member")},
	}
	if got := topology.Managers(); !reflect.DeepEqual(got, want) {
		t.Fatalf("local topology:\n got %+v\nwant %+v", got, want)
	}
	if topology.ModelID() != compiled.Models[0].ID || topology.ModelDigest() != compiled.Models[0].Digest {
		t.Fatal("topology lost its local model binding")
	}
	for _, foreign := range []string{"markitect", "markitect/core", "software"} {
		if _, found := topology.Manager(foreign); found {
			t.Errorf("imported package %q created a Manager", foreign)
		}
	}
	for _, manager := range topology.Managers() {
		for _, identity := range manager.Definitions {
			if identity.Package != manager.Package {
				t.Errorf("%s inherited Definition %+v", manager.Package, identity)
			}
		}
	}
	leaf, _ := topology.Manager("empty")
	if leaf.HasLocalWork() || leaf.PassThrough() || len(leaf.Children) != 0 {
		t.Fatalf("empty leaf has work or loses its leaf semantics: %+v", leaf)
	}
}

func TestIntermediatePackagesAndExactPassThrough(t *testing.T) {
	input := localFixture(t, map[string][]string{
		"a/b/c":       {"item"},
		"local":       {"item"},
		"local/child": {"item"},
	}, "a/b/empty", "a/b/empty/deep")
	topology := requireTopology(t, compileFixture(t, input))
	for _, test := range []struct {
		path, parent           string
		children               []string
		passThrough, localWork bool
	}{
		{".", "", []string{"a", "local"}, false, false},
		{"a", ".", []string{"a/b"}, true, false},
		{"a/b", "a", []string{"a/b/c", "a/b/empty"}, false, false},
		{"a/b/c", "a/b", nil, false, true},
		{"a/b/empty", "a/b", []string{"a/b/empty/deep"}, true, false},
		{"a/b/empty/deep", "a/b/empty", nil, false, false},
		{"local", ".", []string{"local/child"}, false, true},
		{"local/child", "local", nil, false, true},
	} {
		manager, found := topology.Manager(test.path)
		if !found || manager.Parent != test.parent || !reflect.DeepEqual(manager.Children, test.children) || manager.PassThrough() != test.passThrough || manager.HasLocalWork() != test.localWork {
			t.Errorf("%s: got %+v passThrough=%v localWork=%v", test.path, manager, manager.PassThrough(), manager.HasLocalWork())
		}
	}
}

func TestRootIsOrdinaryButNeverAPassThroughPackage(t *testing.T) {
	for _, directories := range [][]string{nil, {"one"}, {"one/two/three"}} {
		compiled := compileFixture(t, localFixture(t, nil, directories...))
		topology := requireTopology(t, compiled)
		root, found := topology.Manager(organization.Root)
		if !found || root.Parent != "" || root.HasLocalWork() || root.PassThrough() || len(root.Definitions) != 0 {
			t.Fatalf("invalid ordinary root: %+v", root)
		}
		if len(directories) == 0 && len(topology.Managers()) != 1 {
			t.Fatal("a passed empty model must have exactly its ordinary root")
		}
	}
}

func TestTopologyDoesNotDependOnEnumerationOrPhysicalSources(t *testing.T) {
	input := localFixture(t, map[string][]string{"z": {"second", "first"}, "a/child": {"item"}}, "empty")
	compiled := compileFixture(t, input)
	want := requireTopology(t, compiled)
	slices.Reverse(input.Root.Entries)
	slices.Reverse(input.Dependencies)
	if got := requireTopology(t, compileFixture(t, input)); !reflect.DeepEqual(got.Managers(), want.Managers()) {
		t.Fatal("source enumeration changed topology")
	}
	shuffled := copyResult(compiled)
	slices.Reverse(shuffled.Models)
	for i := range shuffled.Models {
		slices.Reverse(shuffled.Models[i].Packages)
		slices.Reverse(shuffled.Models[i].Definitions)
		for j := range shuffled.Models[i].Packages {
			shuffled.Models[i].Packages[j].Source = model.Source{Origin: "opaque relocation", Path: "never/read/this"}
		}
		for j := range shuffled.Models[i].Definitions {
			shuffled.Models[i].Definitions[j].Source = model.Source{Path: "unrelated/physical.yaml"}
		}
	}
	before := copyResult(shuffled)
	got := requireTopology(t, shuffled)
	if !reflect.DeepEqual(got.Managers(), want.Managers()) || got.ModelDigest() != want.ModelDigest() || !reflect.DeepEqual(shuffled, before) {
		t.Fatal("topology depends on order/provenance or mutates compiler output")
	}
}

func TestTopologyOwnsItsCollections(t *testing.T) {
	compiled := compileFixture(t, localFixture(t, map[string][]string{"feature": {"first", "second"}, "feature/child": {"item"}}))
	topology := requireTopology(t, compiled)
	want := topology.Managers()
	all := topology.Managers()
	all[0].Children[0] = "changed"
	all[1].Definitions[0].ID = "changed"
	manager, _ := topology.Manager("feature")
	manager.Children[0] = "changed"
	manager.Definitions[0].ID = "changed"
	compiled.Models[0].Packages[0].Name = "changed"
	compiled.Models[0].Definitions[0].Identity.ID = "changed"
	if !reflect.DeepEqual(topology.Managers(), want) {
		t.Fatal("caller collection mutations changed the topology")
	}
	if _, found := topology.Manager("missing"); found {
		t.Fatal("unknown Manager lookup succeeded")
	}
}

func TestEmptyPackageChangesTopologyWithoutChangingCanonicalDigest(t *testing.T) {
	input := localFixture(t, map[string][]string{"feature": {"item"}})
	before := requireTopology(t, compileFixture(t, input))
	input.Root.Entries = append(slices.Clone(input.Root.Entries), model.Entry{Path: "model/empty", Kind: model.Directory})
	after := requireTopology(t, compileFixture(t, input))
	if before.ModelDigest() != after.ModelDigest() {
		t.Fatal("an empty Package changed the canonical model digest")
	}
	if reflect.DeepEqual(before.Managers(), after.Managers()) {
		t.Fatal("the authoritative empty Package was lost from topology")
	}
	if _, found := before.Manager("empty"); found {
		t.Fatal("the original topology unexpectedly contained the empty Package")
	}
	manager, found := after.Manager("empty")
	if !found || manager.Parent != organization.Root || manager.HasLocalWork() || manager.PassThrough() {
		t.Fatalf("the added empty Package is not an empty leaf: %+v", manager)
	}
}

func TestExactUnicodePathsAndLocallyEmptyImportedSpelling(t *testing.T) {
	compiled := compileFixture(t, localFixture(t, map[string][]string{
		"é/子": {"ö"}, "z": {"item"},
	}, "software"))
	topology := requireTopology(t, compiled)
	var paths []string
	for _, manager := range topology.Managers() {
		paths = append(paths, manager.Package)
	}
	if want := []string{".", "software", "z", "é", "é/子"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("exact Package spellings/order: got %q want %q", paths, want)
	}
	manager, found := topology.Manager("é/子")
	if !found || manager.Parent != "é" || !reflect.DeepEqual(manager.Definitions, refs("é/子", "ö")) {
		t.Fatalf("Unicode Package identity changed: %+v", manager)
	}
	// Local empty Packages do not globally bind names. A local empty spelling
	// remains a slot even if an imported model owns populated vocabulary there.
	empty, found := topology.Manager("software")
	if !found || empty.HasLocalWork() || len(empty.Definitions) != 0 {
		t.Fatalf("imported vocabulary replaced a local empty Package: %+v", empty)
	}
	if _, found := topology.Manager("e\u0301/子"); found {
		t.Fatal("lookup normalized a nonexact Package spelling")
	}
}

func TestUnsuccessfulCompilerResultsNeverBecomeEmptyTopologies(t *testing.T) {
	passed := compileFixture(t, localFixture(t, nil, "empty"))
	for _, status := range []model.Status{"", model.Failed, model.NotCompleted} {
		for _, localOnly := range []bool{false, true} {
			result := copyResult(passed)
			if localOnly {
				result.Models[0].Status = status
			} else {
				result.Status = status
			}
			requireRejected(t, result)
		}
	}
	requireRejected(t, model.Result{})
	passed.Coordinates = model.NotCompleted
	if _, err := organization.BuildTopology(passed); err != nil {
		t.Fatalf("coordinate coverage became an unintended content gate: %v", err)
	}
}

func TestInconsistentPassedInputsAreRefusedRatherThanCompleted(t *testing.T) {
	baseline := compileFixture(t, localFixture(t, map[string][]string{"a/b": {"item"}}))
	for _, mutate := range []func(*model.Result){
		func(r *model.Result) { r.Models = nil },
		func(r *model.Result) { r.Models[0].Local = false },
		func(r *model.Result) { r.Models[1].Local = true },
		func(r *model.Result) { r.Models[0].ID = "" },
		func(r *model.Result) { r.Models[0].Digest = "" },
		func(r *model.Result) { r.Models[0].Packages = nil },
		func(r *model.Result) { r.Models[0].Packages = r.Models[0].Packages[1:] },
		func(r *model.Result) { r.Models[0].Packages = append(r.Models[0].Packages, r.Models[0].Packages[0]) },
		func(r *model.Result) {
			r.Models[0].Definitions = append(r.Models[0].Definitions, r.Models[0].Definitions[0])
		},
		func(r *model.Result) { r.Models[0].Definitions[0].Identity.ID = "" },
		func(r *model.Result) { r.Models[0].Definitions[0].ModelID = "foreign" },
	} {
		result := copyResult(baseline)
		mutate(&result)
		requireRejected(t, result)
	}
	for _, badPath := range []string{"", ".", "../a", "/a", "a/", "a//b", "a/./b", "a\\b", "C:a", "a\x00b"} {
		result := copyResult(baseline)
		result.Models[0].Packages[0].Name = badPath
		requireRejected(t, result)
	}
}

func requireRejected(t *testing.T, result model.Result) {
	t.Helper()
	topology, err := organization.BuildTopology(result)
	if err == nil || len(topology.Managers()) != 0 || topology.ModelID() != "" || topology.ModelDigest() != "" {
		t.Fatalf("invalid input produced usable topology: %+v error=%v", topology.Managers(), err)
	}
	if _, found := topology.Manager(organization.Root); found {
		t.Fatal("rejected input still exposed a root Manager")
	}
}
