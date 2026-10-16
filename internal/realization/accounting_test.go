package realization_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var (
	borrow   = model.Identity{Package: "lending", ID: "borrow"}
	policy   = model.Identity{Package: "lending", ID: "loan-policy"}
	callable = model.Identity{Package: "library", ID: "callable-interface"}
)

// These are explicit compiler-result facts for testing the consumer, not a
// replacement compiler or evidence of language conformance. Only fields relevant
// to registry accounting are populated.
func compiledFacts() model.Result {
	return model.Result{Status: model.Passed, Models: []model.ModelResult{
		{Local: true, ID: "library-model", Status: model.Passed,
			Packages:    []model.Package{{Name: "library"}, {Name: "lending"}, {Name: "empty"}, {Name: "empty/leaf"}},
			Definitions: []model.Definition{{Identity: borrow, ModelID: "library-model"}, {Identity: policy, ModelID: "library-model"}, {Identity: callable, ModelID: "library-model"}}},
		{ID: "vocabulary", Status: model.Passed, Packages: []model.Package{{Name: "foreign"}, {Name: "foreign/empty"}}, Definitions: []model.Definition{{Identity: model.Identity{Package: "foreign", ID: "Vocabulary"}, ModelID: "vocabulary"}}},
	}}
}

func fileTree(t *testing.T, names ...string) snapshot.Tree {
	t.Helper()
	var entries []snapshot.Entry
	for _, name := range names {
		e, err := snapshot.NewEntry(path(t, name), snapshot.Regular, []byte(name))
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, e)
	}
	tree, err := snapshot.NewTree(entries)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func classified(t *testing.T, name string, class realization.FileClass) realization.Classification {
	t.Helper()
	c, err := realization.NewClassification(path(t, name), class, "A fixed supplied classification.")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func findingCodes(report realization.Accounting) []string {
	var codes []string
	for _, finding := range report.Findings() {
		codes = append(codes, finding.Code)
	}
	slices.Sort(codes)
	return codes
}

func TestAccountsLibraryAndGenericFilesWithoutInferringOwnership(t *testing.T) {
	compiled := compiledFacts()
	files := fileTree(t, "realization/internal/lending/borrow.go", "go.mod", "README.md", "model/library.yaml", ".markitect/registry/registry.yaml", ".markitect/state/control")
	registrations := registry(t,
		registration(t, owner(t, "lending"), artifact(t, "realization/internal/lending/borrow.go", borrow, policy)),
		registration(t, owner(t, "library"), artifact(t, "go.mod", callable)),
	)
	classes := []realization.Classification{
		classified(t, "README.md", realization.IgnoredFile),
		classified(t, "model/library.yaml", realization.CanonicalInput),
		classified(t, ".markitect/registry/registry.yaml", realization.MarkitectInput),
	}
	report := realization.AccountFiles(compiled, files, registrations, classes)
	if report.Status() != realization.Accounted || len(report.Findings()) != 0 || report.TreeID() != files.Digest() {
		t.Fatalf("accounting: %+v", report.Findings())
	}
	if !reflect.DeepEqual(report.Artifacts(borrow), []snapshot.Path{path(t, "realization/internal/lending/borrow.go")}) || !reflect.DeepEqual(report.Artifacts(policy), report.Artifacts(borrow)) {
		t.Fatal("many-to-many model links lost")
	}
	wantClasses := map[string]realization.FileClass{
		".markitect/registry/registry.yaml": realization.MarkitectInput, ".markitect/state/control": realization.OperationalState,
		"README.md": realization.IgnoredFile, "go.mod": realization.RegisteredArtifact, "model/library.yaml": realization.CanonicalInput,
		"realization/internal/lending/borrow.go": realization.RegisteredArtifact,
	}
	for _, file := range report.Files() {
		if file.Class() != wantClasses[file.Entry().Path().String()] || file.Why() == "" {
			t.Fatalf("file: %+v", file)
		}
		reg, present := file.Registration()
		if present != (file.Class() == realization.RegisteredArtifact) {
			t.Fatal("registration presence is wrong")
		}
		if present && reg.Artifact().Path() != file.Entry().Path() {
			t.Fatal("registration differs from observed file")
		}
	}
	// A fixed result cannot be mutated through compiler inputs or report copies.
	compiled.Models[0].Packages[0].Name = "changed"
	compiled.Models[0].Definitions[0].Identity = model.Identity{}
	classes[0] = realization.Classification{}
	report.Files()[0] = realization.File{}
	report.Artifacts(borrow)[0] = snapshot.Path{}
	if report.Files()[0].Entry().Path().String() != ".markitect/registry/registry.yaml" || report.Artifacts(borrow)[0].String() != "realization/internal/lending/borrow.go" {
		t.Fatal("fixed result exposes aliases")
	}
}

func TestEmptyLocalPackagesAndExplicitRootAreOwnersButImportsAreNot(t *testing.T) {
	for _, o := range []realization.Owner{owner(t, "empty"), owner(t, "empty/leaf"), realization.RootOwner()} {
		report := realization.AccountFiles(compiledFacts(), fileTree(t, "shared/config.txt"), registry(t, registration(t, o, artifact(t, "shared/config.txt", callable, borrow))), nil)
		if report.Status() != realization.Accounted {
			t.Fatalf("supplied owner rejected: %+v", report.Findings())
		}
	}
	for _, name := range []string{"foreign", "foreign/empty", "missing"} {
		report := realization.AccountFiles(compiledFacts(), fileTree(t, "a"), registry(t, registration(t, owner(t, name), artifact(t, "a", borrow))), nil)
		if report.Status() != realization.AccountingInvalid || !reflect.DeepEqual(findingCodes(report), []string{"registry.unknown_owner"}) || len(report.Artifacts(borrow)) != 0 {
			t.Fatalf("owner %q: %+v", name, report.Findings())
		}
	}
	// An imported Definition can be cited; it does not make its Package an owner.
	id := model.Identity{Package: "foreign", ID: "Vocabulary"}
	report := realization.AccountFiles(compiledFacts(), fileTree(t, "a"), registry(t, registration(t, owner(t, "library"), artifact(t, "a", id))), nil)
	if report.Status() != realization.Accounted || len(report.Artifacts(id)) != 1 {
		t.Fatal(report.Findings())
	}
}

func TestMissingDanglingConflictingAndUnexplainedFilesBlockAccounting(t *testing.T) {
	registrations := []realization.Registration{
		registration(t, owner(t, "lending"), artifact(t, "missing.go", borrow)),
		registration(t, realization.RootOwner(), artifact(t, "present.go", model.Identity{Package: "gone", ID: "removed"})),
		registration(t, owner(t, "lending"), artifact(t, "protected.yaml", borrow)),
		registration(t, owner(t, "lending"), artifact(t, "ignored.go", borrow)),
	}
	classes := []realization.Classification{
		classified(t, "protected.yaml", realization.CanonicalInput),
		classified(t, "ignored.go", realization.IgnoredFile),
		classified(t, "not-present", realization.IgnoredFile),
		classified(t, "duplicate", realization.MarkitectInput),
		classified(t, "duplicate", realization.IgnoredFile),
	}
	files := fileTree(t, "present.go", "protected.yaml", "ignored.go", "duplicate", "unexplained.txt")
	first := realization.AccountFiles(compiledFacts(), files, registry(t, registrations...), classes)
	slices.Reverse(registrations)
	slices.Reverse(classes)
	second := realization.AccountFiles(compiledFacts(), files, registry(t, registrations...), classes)
	want := []string{"classification.duplicate", "classification.missing_file", "file.conflicting_classification", "file.conflicting_classification", "file.unexplained", "registry.dangling_basis", "registry.missing_file"}
	if first.Status() != realization.AccountingInvalid || !reflect.DeepEqual(findingCodes(first), want) || !reflect.DeepEqual(first.Findings(), second.Findings()) {
		t.Fatalf("findings: %+v / %+v", first.Findings(), second.Findings())
	}
	if len(first.Artifacts(borrow)) != 0 {
		t.Fatal("invalid registrations were counted as realization")
	}
	copy := first.Findings()
	copy[0] = realization.Finding{}
	if first.Findings()[0].Code == "" {
		t.Fatal("findings exposed an alias")
	}
}

func TestIncompleteModelDoesNotMeanAuthoritativelyEmptyInventory(t *testing.T) {
	for _, status := range []model.Status{"", model.NotCompleted, model.Failed} {
		compiled := compiledFacts()
		compiled.Status = status
		compiled.Models[0].Packages = nil
		compiled.Models[0].Definitions = nil
		files := fileTree(t, "a")
		reg := registry(t, registration(t, owner(t, "lending"), artifact(t, "a", borrow)))
		report := realization.AccountFiles(compiled, files, reg, nil)
		wantStatus := realization.AccountingIncomplete
		if status == model.Failed {
			wantStatus = realization.AccountingInvalid
		}
		if report.Status() != wantStatus || !reflect.DeepEqual(findingCodes(report), []string{"model.unavailable"}) || len(report.Artifacts(borrow)) != 0 {
			t.Fatalf("model %q: %+v", status, report.Findings())
		}
		// Known independent file errors survive unavailable model resolution.
		report = realization.AccountFiles(compiled, snapshot.Tree{}, reg, nil)
		if report.Status() != realization.AccountingInvalid || !reflect.DeepEqual(findingCodes(report), []string{"model.unavailable", "registry.missing_file"}) {
			t.Fatal(report.Findings())
		}
	}
	compiled := compiledFacts()
	compiled.Models[0].Status = model.NotCompleted
	compiled.Models[0].Packages = nil
	report := realization.AccountFiles(compiled, fileTree(t, "a"), registry(t, registration(t, owner(t, "lending"), artifact(t, "a", borrow))), nil)
	if report.Status() == realization.Accounted || slices.Contains(findingCodes(report), "registry.unknown_owner") {
		t.Fatal("unavailable own model was interpreted as empty")
	}
}

func TestIgnoringFilesCannotSupplyDefinitionRealization(t *testing.T) {
	compiled := compiledFacts()
	report := realization.AccountFiles(compiled, fileTree(t, "ignored.go", ".markitect/state/control"), realization.Registry{}, []realization.Classification{classified(t, "ignored.go", realization.IgnoredFile)})
	// All files have structural accounting; no Definition realization is claimed.
	if report.Status() != realization.Accounted || len(report.Artifacts(borrow)) != 0 || len(report.Artifacts(callable)) != 0 {
		t.Fatal("ignore fabricated realization")
	}
	if (realization.Accounting{}).Status() != realization.AccountingIncomplete {
		t.Fatal("zero report claimed success")
	}
	for _, class := range []realization.FileClass{realization.RegisteredArtifact, realization.OperationalState, realization.UnexplainedFile, "invalid"} {
		if _, err := realization.NewClassification(path(t, "a"), class, "why"); !errors.Is(err, realization.ErrInvalidClassification) {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		p   snapshot.Path
		why string
	}{{snapshot.Path{}, "why"}, {path(t, "a"), " \n "}, {path(t, ".markitect/state/control"), "why"}} {
		if _, err := realization.NewClassification(tc.p, realization.IgnoredFile, tc.why); !errors.Is(err, realization.ErrInvalidClassification) {
			t.Fatal(err)
		}
	}
	class := classified(t, "a", realization.MarkitectInput)
	if class.Path().String() != "a" || class.Class() != realization.MarkitectInput || class.Why() == "" {
		t.Fatal("classification changed")
	}
}

func TestContradictoryPublicModelFactsCannotPass(t *testing.T) {
	for _, change := range []func(*model.Result){
		func(r *model.Result) { r.Models = nil },
		func(r *model.Result) { r.Models[0].Local = false },
		func(r *model.Result) { r.Models[1].Local = true },
		func(r *model.Result) { r.Models[1].ID = r.Models[0].ID },
		func(r *model.Result) { r.Models[0].ID = "" },
		func(r *model.Result) { r.Models[0].Packages = nil },
		func(r *model.Result) { r.Models[0].Packages = append(r.Models[0].Packages, r.Models[0].Packages[0]) },
		func(r *model.Result) { r.Models[0].Definitions[0].ModelID = "other" },
		func(r *model.Result) { r.Models[0].Definitions[0].Identity = model.Identity{} },
		func(r *model.Result) {
			r.Models[0].Definitions = append(r.Models[0].Definitions, r.Models[0].Definitions[0])
		},
	} {
		compiled := compiledFacts()
		change(&compiled)
		if report := realization.AccountFiles(compiled, snapshot.Tree{}, realization.Registry{}, nil); report.Status() == realization.Accounted {
			t.Fatal("contradictory facts claimed accounting success")
		}
	}
	if report := realization.AccountFiles(compiledFacts(), snapshot.Tree{}, realization.Registry{}, []realization.Classification{{}}); report.Status() != realization.AccountingInvalid {
		t.Fatal("zero classification accepted")
	}
}

func TestGenericAccountingAndAuthoritativeEmptyInventory(t *testing.T) {
	id := model.Identity{Package: "engine/api", ID: "Endpoint"}
	compiled := model.Result{Status: model.Passed, Models: []model.ModelResult{{
		ID: "generic-project", Local: true, Status: model.Passed,
		Packages:    []model.Package{{Name: "engine"}, {Name: "engine/api"}, {Name: "empty"}},
		Definitions: []model.Definition{{Identity: id, ModelID: "generic-project"}},
	}}}
	reg := registry(t, registration(t, owner(t, "engine"), artifact(t, "source/Endpoint.go", id)))
	files := fileTree(t, "source/Endpoint.go")
	first := realization.AccountFiles(compiled, files, reg, nil)
	slices.Reverse(compiled.Models[0].Packages)
	second := realization.AccountFiles(compiled, files, reg, nil)
	if first.Status() != realization.Accounted || !reflect.DeepEqual(first.Files(), second.Files()) || len(first.Artifacts(id)) != 1 {
		t.Fatal("generic project accounting depends on library names or inventory order")
	}
	compiled.Models[0].Packages = nil
	compiled.Models[0].Definitions = nil
	if report := realization.AccountFiles(compiled, snapshot.Tree{}, realization.Registry{}, nil); report.Status() != realization.Accounted {
		t.Fatal("a Passed empty inventory was not authoritative")
	}
	compiled.Models[0].Status = model.NotCompleted
	if report := realization.AccountFiles(compiled, snapshot.Tree{}, realization.Registry{}, nil); report.Status() != realization.AccountingIncomplete {
		t.Fatal("an unavailable inventory was treated as an authoritative empty set")
	}
}

func TestOperationalStateExclusionHasAnExactBoundary(t *testing.T) {
	report := realization.AccountFiles(compiledFacts(), fileTree(t, ".markitect/statex/control"), realization.Registry{}, nil)
	if report.Status() != realization.AccountingInvalid || !reflect.DeepEqual(findingCodes(report), []string{"file.unexplained"}) {
		t.Fatal("a neighboring ordinary path was silently excluded as operational state")
	}
}
