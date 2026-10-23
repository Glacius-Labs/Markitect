package realization_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

func layoutFacts() model.Result {
	return model.Result{Status: model.Passed, Models: []model.ModelResult{{
		Local: true, ID: "library-model", Status: model.Passed,
		Packages: []model.Package{{Name: "library"}, {Name: "lending"}, {Name: "lending/reminders"}},
		Definitions: []model.Definition{
			{Identity: borrow, ModelID: "library-model"}, {Identity: callable, ModelID: "library-model"},
		},
	}}}
}

func dir(name string, who realization.Owner) realization.Area {
	if name == "./" {
		return realization.Area{Directory: true, Owner: who}
	}
	return realization.Area{Path: strings.TrimSuffix(name, "/"), Directory: true, Owner: who}
}

func exactFile(name string, who realization.Owner) realization.Area {
	return realization.Area{Path: name, Owner: who}
}

// libraryLayout: Root owns the realization directory, each Package its own
// directory, library additionally the exact build file in Root's directory
// and Root an exact file inside lending's directory.
func libraryLayout(t *testing.T) realization.Layout {
	t.Helper()
	layout, err := realization.NewLayout(layoutFacts(), []realization.Area{
		dir("realization/", realization.RootOwner()),
		dir("realization/lending/", owner(t, "lending")),
		dir("realization/lending/reminders/", owner(t, "lending/reminders")),
		dir("realization/cmd/library/", owner(t, "library")),
		exactFile("realization/lending/Makefile", realization.RootOwner()),
		exactFile("realization/go.mod", owner(t, "library")),
	})
	if err != nil {
		t.Fatal(err)
	}
	return layout
}

func ownerName(o realization.Owner) string {
	if o.IsRoot() {
		return "."
	}
	name, _ := o.Package()
	return name
}

// The most specific area owns a path: exact files over directories, deeper
// directories over shallower ones, for every Manager; uncovered paths have no
// owner.
func TestOwnershipTheMostSpecificAreaWins(t *testing.T) {
	layout := libraryLayout(t)
	cases := map[string]string{
		"realization/README.md":                  ".",
		"realization/go.mod":                     "library",
		"realization/lending/loan.go":            "lending",
		"realization/lending/Makefile":           ".",
		"realization/lending/reminders/x.go":     "lending/reminders",
		"realization/lending/reminders/a/b.go":   "lending/reminders",
		"realization/cmd/library/main.go":        "library",
		"realization/cmd/other/main.go":          ".",
		"realization/lending/remindersx/file.go": "lending",
	}
	for name, want := range cases {
		got, ok := layout.OwnerOf(path(t, name))
		if !ok || ownerName(got) != want {
			t.Errorf("%s: owner %q (%v), want %q", name, ownerName(got), ok, want)
		}
	}
	if _, ok := layout.OwnerOf(path(t, "README.md")); ok {
		t.Errorf("a path outside every area has an owner")
	}
	if got, ok := layout.DirectoryOwner("realization/lending/reminders/new"); !ok || ownerName(got) != "lending/reminders" {
		t.Errorf("directory owner %q", ownerName(got))
	}
	if got, _ := layout.DirectoryOwner("realization"); ownerName(got) != "." {
		t.Errorf("directory owner of the root directory %q", ownerName(got))
	}
	whole, err := realization.NewLayout(layoutFacts(), []realization.Area{dir("./", realization.RootOwner()), dir("src/lending/", owner(t, "lending"))})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := whole.OwnerOf(path(t, "README.md")); !ok || !got.IsRoot() {
		t.Errorf("an explicit broad Root area owns every other path")
	}
	if got, _ := whole.OwnerOf(path(t, "src/lending/x.go")); ownerName(got) != "lending" {
		t.Errorf("a broad Root area absorbed another Manager's domain")
	}
}

func TestOwnershipRefusals(t *testing.T) {
	cases := map[string][]realization.Area{
		"under .markitect":        {dir(".markitect/registry/", realization.RootOwner())},
		"under .git":              {exactFile(".git/config", realization.RootOwner())},
		"unknown owner":           {dir("src/", realization.Owner{})},
		"not a Manager":           {dir("src/", owner(t, "catalog"))},
		"equal specificity":       {dir("src/", realization.RootOwner()), dir("src/", owner(t, "lending"))},
		"file and directory":      {dir("src/a/", realization.RootOwner()), exactFile("src/a", owner(t, "lending"))},
		"below an exact file":     {exactFile("src/a", realization.RootOwner()), exactFile("src/a/b", owner(t, "lending"))},
		"letter case":             {dir("src/Lending/", realization.RootOwner()), dir("src/lending/", owner(t, "lending"))},
		"exact file without path": {{Owner: realization.RootOwner()}},
	}
	for name, areas := range cases {
		if _, err := realization.NewLayout(layoutFacts(), areas); !errors.Is(err, realization.ErrInvalidLayout) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestOwnershipDescribesAreasWithForeignExclusions(t *testing.T) {
	layout := libraryLayout(t)
	root := layout.Describe(realization.RootOwner())
	want := []string{"realization/ except realization/cmd/library/ (Manager \"library\"), realization/go.mod (Manager \"library\"), realization/lending/ (Manager \"lending\")", "realization/lending/Makefile"}
	if !slices.Equal(root, want) {
		t.Errorf("Root areas\n%q\nwant\n%q", root, want)
	}
	lending := layout.Describe(owner(t, "lending"))
	if len(lending) != 1 || !strings.Contains(lending[0], "realization/lending/reminders/ (Manager \"lending/reminders\")") || !strings.Contains(lending[0], "realization/lending/Makefile (Root)") {
		t.Errorf("lending areas %q", lending)
	}
}

// Accounting under declared ownership: a directory entry registers its
// registrant's files below it, an exact entry is more specific, other owners'
// files are excluded, and wrong owners and uncovered files block.
func TestAccountingExpandsDirectoryEntriesWithinOwnership(t *testing.T) {
	layout := libraryLayout(t)
	tree := fileTree(t, "realization/README.md", "realization/go.mod", "realization/lending/loan.go", "realization/lending/loan_test.go",
		"realization/lending/Makefile", "realization/lending/reminders/notice.go", "realization/cmd/library/main.go")
	lending := owner(t, "lending")
	entries := []realization.Registration{
		registration(t, realization.RootOwner(), artifact(t, "realization/README.md", callable)),
		registration(t, realization.RootOwner(), artifact(t, "realization/lending/Makefile", borrow)),
		registration(t, owner(t, "library"), artifact(t, "realization/go.mod", callable)),
		registration(t, owner(t, "library"), directoryArtifact(t, "realization/cmd/library", callable)),
		registration(t, lending, directoryArtifact(t, "realization/lending", borrow)),
		registration(t, lending, artifactForm(t, "realization/lending/loan_test.go", realization.Test, borrow)),
		registration(t, owner(t, "lending/reminders"), directoryArtifact(t, "realization/lending/reminders", borrow)),
	}
	registry, err := realization.NewRegistry(entries)
	if err != nil {
		t.Fatal(err)
	}
	report := realization.AccountFiles(layoutFacts(), tree, registry, nil, layout)
	if report.Status() != realization.Accounted {
		t.Fatalf("findings %+v", report.Findings())
	}
	forms := map[string]string{}
	for _, f := range report.Files() {
		r, ok := f.Registration()
		if !ok {
			t.Fatalf("%s unregistered", f.Entry().Path())
		}
		forms[f.Entry().Path().String()] = ownerName(r.Owner()) + " " + string(r.Artifact().Form()) + " " + r.Artifact().Selector()
	}
	if forms["realization/lending/loan_test.go"] != "lending test realization/lending/loan_test.go" || forms["realization/lending/loan.go"] != "lending code realization/lending/" ||
		forms["realization/lending/Makefile"] != ". code realization/lending/Makefile" || forms["realization/lending/reminders/notice.go"] != "lending/reminders code realization/lending/reminders/" {
		t.Errorf("effective registrations %v", forms)
	}
	if got := report.Artifacts(borrow); !slices.Contains(got, path(t, "realization/lending/loan.go")) || !slices.Contains(got, path(t, "realization/lending/reminders/notice.go")) {
		t.Errorf("borrow artifacts %v", got)
	}
}

func TestAccountingBlocksWrongOwnersEmptyDirectoriesAndUncoveredFiles(t *testing.T) {
	layout := libraryLayout(t)
	tree := fileTree(t, "realization/lending/loan.go", "realization/lending/reminders/notice.go", "docs/notes.md", "realization/cmd/library/main.go")
	lending := owner(t, "lending")
	registry, err := realization.NewRegistry([]realization.Registration{
		registration(t, lending, directoryArtifact(t, "realization/lending", borrow)),
		// lending's broad entry does not take its child's file, and its exact entry
		// of the child's file is a wrong-owner registration.
		registration(t, lending, artifact(t, "realization/lending/reminders/notice.go", borrow)),
		// A directory entry at a directory another Manager owns.
		registration(t, lending, directoryArtifact(t, "realization/cmd", borrow)),
		// A directory entry that registers nothing.
		registration(t, owner(t, "library"), directoryArtifact(t, "realization/cmd/library/empty", callable)),
	})
	if err != nil {
		t.Fatal(err)
	}
	report := realization.AccountFiles(layoutFacts(), tree, registry, nil, layout)
	codes := findingCodes(report)
	want := []string{"file.unexplained", "file.unexplained", "layout.no_owner", "registry.missing_file", "registry.outside_area", "registry.outside_area"}
	if !slices.Equal(codes, want) {
		t.Fatalf("codes %v, want %v: %+v", codes, want, report.Findings())
	}
}

// Without declared ownership the most specific entry decides, and a protected
// input below a directory entry stays an input.
func TestAccountingWithoutOwnershipUsesTheMostSpecificEntry(t *testing.T) {
	tree := fileTree(t, "src/a.go", "src/b.go", "src/cfg.yaml")
	registry, err := realization.NewRegistry([]realization.Registration{
		registration(t, owner(t, "lending"), directoryArtifact(t, "src", borrow)),
		registration(t, owner(t, "library"), artifact(t, "src/b.go", callable)),
	})
	if err != nil {
		t.Fatal(err)
	}
	report := realization.AccountFiles(checkedFacts(), tree, registry, []realization.Classification{classified(t, "src/cfg.yaml", realization.MarkitectInput)}, realization.Layout{})
	if report.Status() != realization.Accounted {
		t.Fatalf("findings %+v", report.Findings())
	}
	for _, f := range report.Files() {
		r, _ := f.Registration()
		switch f.Entry().Path().String() {
		case "src/b.go":
			if ownerName(r.Owner()) != "library" {
				t.Errorf("b.go registered by %s", ownerName(r.Owner()))
			}
		case "src/cfg.yaml":
			if f.Class() != realization.MarkitectInput {
				t.Errorf("a directory entry took over a protected input")
			}
		}
	}
}
