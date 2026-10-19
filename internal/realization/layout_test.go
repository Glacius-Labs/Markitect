package realization_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
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

func directoryOf(t *testing.T, name string) snapshot.DirectoryBoundary {
	t.Helper()
	if name == "." {
		return snapshot.RootDirectory()
	}
	boundary, err := snapshot.DirectoryAt(path(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return boundary
}

func roots(t *testing.T, names ...string) []snapshot.DirectoryBoundary {
	t.Helper()
	var result []snapshot.DirectoryBoundary
	for _, name := range names {
		result = append(result, directoryOf(t, name))
	}
	return result
}

func exception(t *testing.T, name string, who realization.Owner) realization.LayoutException {
	t.Helper()
	return realization.LayoutException{Path: path(t, name), Owner: who, Reason: "The callable interface fixes the location."}
}

func libraryLayout(t *testing.T) realization.Layout {
	t.Helper()
	layout, err := realization.NewLayout(layoutFacts(), roots(t, "realization"), []realization.LayoutException{exception(t, "realization/cmd/library", owner(t, "library"))})
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

func TestLayoutOwnershipDeepestMappingWins(t *testing.T) {
	layout := libraryLayout(t)
	for path_, want := range map[string]string{
		"realization/lending/borrow.go":             "lending",
		"realization/lending/internal/deep/x.go":    "lending",
		"realization/lending/reminders/x.go":        "lending/reminders",
		"realization/lending/reminders/a/b/test.go": "lending/reminders",
		"realization/library/library.go":            "library",
		"realization/cmd/library/main.go":           "library",
		"realization/cmd/library":                   "library",
		"realization/cmd/other/main.go":             ".",
		"realization/go.mod":                        ".",
		"realization/lending":                       ".",
		"realization/unknown/x.go":                  ".",
		"realizationx/lending/x.go":                 ".",
		"docs/guide.md":                             ".",
		"lending/x.go":                              ".",
	} {
		if got := ownerName(layout.OwnerOf(path(t, path_))); got != want {
			t.Errorf("owner of %s = %q, want %q", path_, got, want)
		}
	}
	if !layout.Declared() || (realization.Layout{}).Declared() {
		t.Fatal("declared state wrong")
	}
	if got := ownerName((realization.Layout{}).OwnerOf(path(t, "realization/lending/x.go"))); got != "." {
		t.Fatalf("undeclared layout must fall back to Root, got %q", got)
	}
}

func TestLayoutExceptionAgainstMirroredDepth(t *testing.T) {
	layout, err := realization.NewLayout(layoutFacts(), roots(t, "realization", "docs"), []realization.LayoutException{
		exception(t, "realization/lending/reminders/generated", owner(t, "lending")),
		exception(t, "realization/lending", realization.RootOwner()),
		exception(t, "docs/lending/api.md", owner(t, "library")),
	})
	if err != nil {
		t.Fatal(err)
	}
	for path_, want := range map[string]string{
		"realization/lending/reminders/generated/g.go": "lending", // deeper exception beats the mirrored match
		"realization/lending/reminders/x.go":           "lending/reminders",
		"realization/lending/borrow.go":                ".", // an exception on the mirrored directory overrides it
		"docs/lending/api.md":                          "library",
		"docs/lending/other.md":                        "lending",
		"docs/library/readme.md":                       "library",
	} {
		if got := ownerName(layout.OwnerOf(path(t, path_))); got != want {
			t.Errorf("owner of %s = %q, want %q", path_, got, want)
		}
	}
}

func TestLayoutRefusals(t *testing.T) {
	root := realization.RootOwner()
	for name, test := range map[string]struct {
		roots      []string
		exceptions []realization.LayoutException
	}{
		"repository root as root":     {roots: []string{"."}},
		"repeated root":               {roots: []string{"realization", "realization"}},
		"nested roots":                {roots: []string{"realization", "realization/lending"}},
		"root in markitect":           {roots: []string{".markitect/area"}},
		"root markitect itself":       {roots: []string{".markitect"}},
		"exception in markitect":      {exceptions: []realization.LayoutException{exception(t, ".markitect/model/x", root)}},
		"exception markitect itself":  {exceptions: []realization.LayoutException{exception(t, ".markitect", root)}},
		"repeated exception":          {exceptions: []realization.LayoutException{exception(t, "a/b", root), exception(t, "a/b", owner(t, "library"))}},
		"unknown owner":               {exceptions: []realization.LayoutException{exception(t, "a/b", owner(t, "missing"))}},
		"zero owner":                  {exceptions: []realization.LayoutException{exception(t, "a/b", realization.Owner{})}},
		"missing reason":              {exceptions: []realization.LayoutException{{Path: path(t, "a/b"), Owner: root}}},
		"zero exception path":         {exceptions: []realization.LayoutException{{Owner: root, Reason: "Why."}}},
		"roots differing by case":     {roots: []string{"realization", "Realization"}},
		"exception differs by case":   {roots: []string{"realization"}, exceptions: []realization.LayoutException{exception(t, "Realization/cmd", root)}},
		"exceptions differ by case":   {exceptions: []realization.LayoutException{exception(t, "a/Cmd", root), exception(t, "a/cmd", root)}},
		"exception vs mirrored dir":   {roots: []string{"realization"}, exceptions: []realization.LayoutException{exception(t, "realization/Lending/x", root)}},
		"exceptions fold to one key":  {exceptions: []realization.LayoutException{exception(t, "a/Straße", root), exception(t, "a/STRASSE", root)}},
		"exception differs in prefix": {roots: []string{"realization"}, exceptions: []realization.LayoutException{exception(t, "realization/LENDING", owner(t, "library"))}},
	} {
		t.Run(name, func(t *testing.T) {
			layout, err := realization.NewLayout(layoutFacts(), roots(t, test.roots...), test.exceptions)
			if !errors.Is(err, realization.ErrInvalidLayout) {
				t.Fatalf("layout accepted or wrong error: %v", err)
			}
			if layout.Declared() {
				t.Fatal("refusal returned a usable layout")
			}
		})
	}
	if _, err := realization.NewLayout(model.Result{}, roots(t, "realization"), nil); !errors.Is(err, realization.ErrInvalidLayout) {
		t.Fatalf("layout over an unavailable model accepted: %v", err)
	}
	// The same spelling twice as an exception on a mirrored directory is not a
	// case collision; neither are roots without exceptions.
	if _, err := realization.NewLayout(layoutFacts(), roots(t, "realization"), []realization.LayoutException{exception(t, "realization/lending", realization.RootOwner())}); err != nil {
		t.Fatalf("exception on the exact mirrored directory refused: %v", err)
	}
	if _, err := realization.NewLayout(layoutFacts(), nil, nil); err != nil {
		t.Fatalf("empty declared layout refused: %v", err)
	}
}

func TestAccountingRequiresRegistrationsInTheirAreaShard(t *testing.T) {
	compiled := layoutFacts()
	files := fileTree(t, "realization/lending/reminders/x.go", "realization/go.mod", "realization/cmd/library/main.go")
	layout := libraryLayout(t)
	right := registry(t,
		registration(t, owner(t, "lending/reminders"), artifact(t, "realization/lending/reminders/x.go", borrow)),
		registration(t, realization.RootOwner(), artifact(t, "realization/go.mod", callable)),
		registration(t, owner(t, "library"), artifact(t, "realization/cmd/library/main.go", callable)),
	)
	if report := realization.AccountFiles(compiled, files, right, nil, layout); report.Status() != realization.Accounted || len(report.Findings()) != 0 {
		t.Fatalf("correct shards refused: %+v", report.Findings())
	}
	wrong := registry(t,
		registration(t, owner(t, "lending"), artifact(t, "realization/lending/reminders/x.go", borrow)),
		registration(t, realization.RootOwner(), artifact(t, "realization/go.mod", callable)),
		registration(t, owner(t, "lending"), artifact(t, "realization/cmd/library/main.go", callable)),
	)
	report := realization.AccountFiles(compiled, files, wrong, nil, layout)
	if report.Status() != realization.AccountingInvalid {
		t.Fatal("registrations outside the owner's area accepted")
	}
	var outside []string
	for _, finding := range report.Findings() {
		if finding.Code != "registry.outside_area" {
			t.Errorf("unexpected finding %+v", finding)
		}
		if got, _ := finding.Owner.Package(); got != "lending" || finding.Message == "" {
			t.Errorf("finding must name the registering Manager and the actual owner: %+v", finding)
		}
		outside = append(outside, finding.Path.String())
	}
	if want := []string{"realization/cmd/library/main.go", "realization/lending/reminders/x.go"}; !slices.Equal(outside, want) {
		t.Fatalf("findings at %q, want %q", outside, want)
	}
	if got := report.Artifacts(borrow); len(got) != 0 {
		t.Fatalf("an out-of-area registration still counted as a link: %v", got)
	}
	// Without a declared layout the shard alone decides, as before.
	if report := realization.AccountFiles(compiled, files, wrong, nil, realization.Layout{}); report.Status() != realization.Accounted {
		t.Fatalf("undeclared layout rejected a registration: %+v", report.Findings())
	}
}

func TestLayoutDescribesAreasForPrompts(t *testing.T) {
	layout := libraryLayout(t)
	tree := fileTree(t, "realization/go.mod", "realization/cmd/library/main.go", "realization/lending/x.go", "realization/lending/reminders/y.go")
	for who, want := range map[string][]string{
		"lending":           {"realization/lending/** except realization/lending/reminders/**"},
		"lending/reminders": {"realization/lending/reminders/**"},
		"library":           {"realization/library/**", "realization/cmd/library/**"},
		".":                 {"files directly in realization/ such as realization/go.mod", "every file outside the artifact roots and exceptions"},
	} {
		o := realization.RootOwner()
		if who != "." {
			o = owner(t, who)
		}
		if got := layout.Describe(o, tree); !reflect.DeepEqual(got, want) {
			t.Errorf("areas of %s = %q, want %q", who, got, want)
		}
	}
	carved, err := realization.NewLayout(layoutFacts(), roots(t, "realization"), []realization.LayoutException{
		exception(t, "realization/lending/vendored", realization.RootOwner()),
		exception(t, "realization/lending/Makefile", owner(t, "library")),
	})
	if err != nil {
		t.Fatal(err)
	}
	files := fileTree(t, "realization/lending/Makefile")
	want := []string{"realization/lending/** except realization/lending/Makefile, realization/lending/reminders/**, realization/lending/vendored/**"}
	if got := carved.Describe(owner(t, "lending"), files); !reflect.DeepEqual(got, want) {
		t.Errorf("carved areas = %q, want %q", got, want)
	}
	if got := carved.Describe(owner(t, "library"), files); !reflect.DeepEqual(got, []string{"realization/library/**", "realization/lending/Makefile"}) {
		t.Errorf("exception file areas = %q", got)
	}
}

func TestLayoutDescribesExceptionOnTheMirroredDirectoryForItsOwner(t *testing.T) {
	layout, err := realization.NewLayout(layoutFacts(), roots(t, "realization"), []realization.LayoutException{exception(t, "realization/lending", owner(t, "library"))})
	if err != nil {
		t.Fatal(err)
	}
	tree := fileTree(t, "realization/lending/x.go")
	if got := ownerName(layout.OwnerOf(path(t, "realization/lending/x.go"))); got != "library" {
		t.Fatalf("owner of the mirrored directory's file = %q, want library", got)
	}
	for who, want := range map[string][]string{
		"lending": nil,
		"library": {"realization/library/**", "realization/lending/** except realization/lending/reminders/**"},
	} {
		if got := layout.Describe(owner(t, who), tree); !reflect.DeepEqual(got, want) {
			t.Errorf("areas of %s = %q, want %q", who, got, want)
		}
	}
	// An exception on its own mirrored directory is described once.
	same, err := realization.NewLayout(layoutFacts(), roots(t, "realization"), []realization.LayoutException{exception(t, "realization/lending", owner(t, "lending"))})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := same.Describe(owner(t, "lending"), tree), []string{"realization/lending/** except realization/lending/reminders/**"}; !reflect.DeepEqual(got, want) {
		t.Errorf("areas = %q, want %q", got, want)
	}
}

func TestLayoutFilesInUnmappedRootSubdirectoriesHaveNoOwner(t *testing.T) {
	layout := libraryLayout(t)
	for path_, want := range map[string]bool{
		"realization/go.mod":              true, // directly in the root: Root
		"realization/lending/borrow.go":   true,
		"realization/cmd/library/main.go": true, // exception
		"realization/cmd/other/main.go":   false,
		"realization/cmd/main.go":         false,
		"realization/unknown/x.go":        false,
		"realization/lending":             true, // a file named like a Package, directly in the root
		"docs/guide.md":                   true, // outside the roots
		"realizationx/a/b.go":             true,
	} {
		if got := layout.HasOwner(path(t, path_)); got != want {
			t.Errorf("HasOwner(%s) = %v, want %v", path_, got, want)
		}
		if got := ownerName(layout.OwnerOf(path(t, path_))); !want && got != "." {
			t.Errorf("owner of the unowned %s = %q, Root must resolve it", path_, got)
		}
	}
	if !(realization.Layout{}).HasOwner(path(t, "realization/unknown/x.go")) {
		t.Fatal("an undeclared layout reports an unowned file")
	}
}

func TestAccountingReportsFilesWithoutAnOwnerToRoot(t *testing.T) {
	files := fileTree(t, "realization/go.mod", "realization/cmd/other/main.go", "realization/stray/x.go", "realization/stray/ignored.txt")
	root := realization.RootOwner()
	classifications := []realization.Classification{classified(t, "realization/stray/ignored.txt", realization.IgnoredFile)}
	registered := registry(t,
		registration(t, root, artifact(t, "realization/go.mod", callable)),
		registration(t, root, artifact(t, "realization/cmd/other/main.go", callable)))
	report := realization.AccountFiles(layoutFacts(), files, registered, classifications, libraryLayout(t))
	var got []string
	for _, finding := range report.Findings() {
		if !finding.Owner.IsRoot() || finding.Message == "" {
			t.Errorf("finding must be addressed to Root with a message: %+v", finding)
		}
		got = append(got, finding.Code+" "+finding.Path.String())
	}
	// A registered or unregistered file is reported once as unowned, never as unexplained.
	if want := []string{"layout.no_owner realization/cmd/other/main.go", "layout.no_owner realization/stray/x.go"}; !slices.Equal(got, want) || report.Status() != realization.AccountingInvalid {
		t.Fatalf("findings %q status %v, want %q", got, report.Status(), want)
	}
	for _, finding := range report.Findings() {
		if !strings.Contains(finding.Message, "the layout needs an exception") || !strings.Contains(finding.Message, "the file must move") {
			t.Errorf("a start stand with unowned files must say how to resolve them: %q", finding.Message)
		}
	}
	if links := report.Artifacts(callable); !slices.Equal(links, []snapshot.Path{path(t, "realization/go.mod")}) {
		t.Fatalf("an unowned registration counted as a link: %v", links)
	}
	// Without a declared layout nothing changes.
	if report := realization.AccountFiles(layoutFacts(), files, registered, classifications, realization.Layout{}); len(report.Findings()) != 1 || report.Findings()[0].Code != "file.unexplained" {
		t.Fatalf("undeclared layout: %+v", report.Findings())
	}
}

func TestLayoutAreasAreDirectoriesParentsFirst(t *testing.T) {
	layout := libraryLayout(t)
	var got []string
	for _, area := range layout.Areas(fileTree(t, "realization/go.mod", "realization/cmd/library/main.go")) {
		got = append(got, area.Path.String()+" "+ownerName(area.Owner))
	}
	want := []string{"realization/cmd/library library", "realization/lending lending", "realization/lending/reminders lending/reminders", "realization/library library"}
	if !slices.Equal(got, want) {
		t.Fatalf("areas %q, want %q", got, want)
	}
	// An exception on a file is no area; one on a mirrored directory replaces it.
	carved, err := realization.NewLayout(layoutFacts(), roots(t, "realization"), []realization.LayoutException{
		exception(t, "realization/lending/Makefile", owner(t, "library")),
		exception(t, "realization/lending", owner(t, "library")),
	})
	if err != nil {
		t.Fatal(err)
	}
	got = nil
	for _, area := range carved.Areas(fileTree(t, "realization/lending/Makefile")) {
		got = append(got, area.Path.String()+" "+ownerName(area.Owner))
	}
	if want := []string{"realization/lending library", "realization/lending/reminders lending/reminders", "realization/library library"}; !slices.Equal(got, want) {
		t.Fatalf("carved areas %q, want %q", got, want)
	}
}

func TestLayoutExceptionKindDecidesFileOrDirectory(t *testing.T) {
	file := realization.LayoutException{Path: path(t, "realization/lending/Makefile"), File: true, Owner: owner(t, "library"), Reason: "The Makefile is shared."}
	layout, err := realization.NewLayout(layoutFacts(), roots(t, "realization"), []realization.LayoutException{file})
	if err != nil {
		t.Fatal(err)
	}
	// A declared file is no area although the start tree does not hold it; the
	// same exception without a kind is a directory and an area.
	var got []string
	for _, area := range layout.Areas(fileTree(t, "realization/go.mod")) {
		got = append(got, area.Path.String())
	}
	if want := []string{"realization/lending", "realization/lending/reminders", "realization/library"}; !slices.Equal(got, want) {
		t.Fatalf("areas with a declared file exception %q, want %q", got, want)
	}
	file.File = false
	directory, err := realization.NewLayout(layoutFacts(), roots(t, "realization"), []realization.LayoutException{file})
	if err != nil {
		t.Fatal(err)
	}
	got = nil
	for _, area := range directory.Areas(fileTree(t, "realization/go.mod")) {
		got = append(got, area.Path.String())
	}
	if want := []string{"realization/lending", "realization/lending/Makefile", "realization/lending/reminders", "realization/library"}; !slices.Equal(got, want) {
		t.Fatalf("areas with a directory exception %q, want %q", got, want)
	}
	// The declared file still owns its path and is described as a file.
	if owner := layout.OwnerOf(path(t, "realization/lending/Makefile")); owner != file.Owner {
		t.Fatalf("owner %v", owner)
	}
	want := []string{"realization/library/**", "realization/lending/Makefile"}
	if got := layout.Describe(owner(t, "library"), fileTree(t, "realization/go.mod")); !slices.Equal(got, want) {
		t.Fatalf("description %q, want %q", got, want)
	}
}

func TestLayoutAreasSkipAMirroredDirectoryThatIsAFile(t *testing.T) {
	var got []string
	for _, area := range libraryLayout(t).Areas(fileTree(t, "realization/library", "realization/cmd/library/main.go")) {
		got = append(got, area.Path.String())
	}
	if want := []string{"realization/cmd/library", "realization/lending", "realization/lending/reminders"}; !slices.Equal(got, want) {
		t.Fatalf("areas %q, want %q", got, want)
	}
}
