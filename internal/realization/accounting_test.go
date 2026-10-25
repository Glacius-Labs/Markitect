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

var (
	borrow   = model.Identity{Package: "lending", ID: "borrow"}
	policy   = model.Identity{Package: "lending", ID: "loan-policy"}
	callable = model.Identity{Package: "library", ID: "callable-interface"}
)

// These are explicit Model check result facts for testing the consumer, not a
// replacement Model check or evidence of language conformance. Only fields relevant
// to registry accounting are populated.
func checkedFacts() model.Result {
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
	checked := checkedFacts()
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
	report := realization.AccountFiles(checked, files, registrations, classes, realization.Layout{})
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
	// A fixed result cannot be mutated through Model check inputs or report copies.
	checked.Models[0].Packages[0].Name = "changed"
	checked.Models[0].Definitions[0].Identity = model.Identity{}
	classes[0] = realization.Classification{}
	report.Files()[0] = realization.File{}
	report.Artifacts(borrow)[0] = snapshot.Path{}
	if report.Files()[0].Entry().Path().String() != ".markitect/registry/registry.yaml" || report.Artifacts(borrow)[0].String() != "realization/internal/lending/borrow.go" {
		t.Fatal("fixed result exposes aliases")
	}
}

func TestEmptyLocalPackagesAndExplicitRootAreOwnersButImportsAreNot(t *testing.T) {
	for _, o := range []realization.Owner{owner(t, "empty"), owner(t, "empty/leaf"), realization.RootOwner()} {
		report := realization.AccountFiles(checkedFacts(), fileTree(t, "shared/config.txt"), registry(t, registration(t, o, artifact(t, "shared/config.txt", callable, borrow))), nil, realization.Layout{})
		if report.Status() != realization.Accounted {
			t.Fatalf("supplied owner rejected: %+v", report.Findings())
		}
	}
	for _, name := range []string{"foreign", "foreign/empty", "missing"} {
		report := realization.AccountFiles(checkedFacts(), fileTree(t, "a"), registry(t, registration(t, owner(t, name), artifact(t, "a", borrow))), nil, realization.Layout{})
		if report.Status() != realization.AccountingInvalid || !reflect.DeepEqual(findingCodes(report), []string{"registry.unknown_owner"}) || len(report.Artifacts(borrow)) != 0 {
			t.Fatalf("owner %q: %+v", name, report.Findings())
		}
	}
	// An imported Definition can be cited; it does not make its Package an owner.
	id := model.Identity{Package: "foreign", ID: "Vocabulary"}
	report := realization.AccountFiles(checkedFacts(), fileTree(t, "a"), registry(t, registration(t, owner(t, "library"), artifact(t, "a", id))), nil, realization.Layout{})
	if report.Status() != realization.Accounted || len(report.Artifacts(id)) != 1 {
		t.Fatal(report.Findings())
	}
}

func TestMissingConflictingAndUnexplainedFilesBlockAccounting(t *testing.T) {
	registrations := []realization.Registration{
		registration(t, owner(t, "lending"), artifact(t, "missing.go", borrow)),
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
	files := fileTree(t, "protected.yaml", "ignored.go", "duplicate", "unexplained.txt")
	first := realization.AccountFiles(checkedFacts(), files, registry(t, registrations...), classes, realization.Layout{})
	slices.Reverse(registrations)
	slices.Reverse(classes)
	second := realization.AccountFiles(checkedFacts(), files, registry(t, registrations...), classes, realization.Layout{})
	want := []string{"classification.duplicate", "classification.missing_file", "file.conflicting_classification", "file.conflicting_classification", "file.unexplained", "registry.missing_file"}
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

// A link to a Definition the model no longer has is realization still to do:
// the artifact stays registered to its owner, its other links still count and
// accounting stays valid, while Stale names every such link.
func TestStaleLinksAreRealizationStillToDoNotInvalidAccounting(t *testing.T) {
	removed := model.Identity{Package: "gone", ID: "removed"}
	registrations := []realization.Registration{
		registration(t, owner(t, "lending"), artifact(t, "mixed.go", borrow, removed)),
		registration(t, realization.RootOwner(), artifact(t, "only.go", removed)),
	}
	first := realization.AccountFiles(checkedFacts(), fileTree(t, "mixed.go", "only.go"), registry(t, registrations...), nil, realization.Layout{})
	slices.Reverse(registrations)
	second := realization.AccountFiles(checkedFacts(), fileTree(t, "mixed.go", "only.go"), registry(t, registrations...), nil, realization.Layout{})
	if first.Status() != realization.Accounted || len(first.Findings()) != 0 || !reflect.DeepEqual(first.Stale(), second.Stale()) {
		t.Fatalf("stale links invalidated accounting: %+v %+v", first.Findings(), first.Stale())
	}
	stale := first.Stale()
	if len(stale) != 2 || stale[0].Path.String() != "mixed.go" || stale[0].Owner != owner(t, "lending") || stale[1].Path.String() != "only.go" || !stale[1].Owner.IsRoot() {
		t.Fatalf("stale links lost their artifact or owner: %+v", stale)
	}
	for _, f := range stale {
		if f.Code != "registry.stale_link" || f.Definition != removed || !strings.Contains(f.Message, "gone/removed") {
			t.Fatalf("stale link does not name its absent Definition: %+v", f)
		}
	}
	if !reflect.DeepEqual(first.Artifacts(borrow), []snapshot.Path{path(t, "mixed.go")}) || len(first.Artifacts(removed)) != 0 {
		t.Fatal("stale link counted as realization, or valid link lost", first.Artifacts(borrow))
	}
	for _, f := range first.Files() {
		if f.Class() != realization.RegisteredArtifact {
			t.Fatal("artifact with a stale link lost its registration", f.Entry().Path())
		}
	}
	stale[0] = realization.Finding{}
	if first.Stale()[0].Code == "" {
		t.Fatal("stale links exposed an alias")
	}
}

func TestIncompleteModelDoesNotMeanAuthoritativelyEmptyInventory(t *testing.T) {
	for _, status := range []model.Status{"", model.NotCompleted, model.Failed} {
		checked := checkedFacts()
		checked.Status = status
		checked.Models[0].Packages = nil
		checked.Models[0].Definitions = nil
		files := fileTree(t, "a")
		reg := registry(t, registration(t, owner(t, "lending"), artifact(t, "a", borrow)))
		report := realization.AccountFiles(checked, files, reg, nil, realization.Layout{})
		wantStatus := realization.AccountingIncomplete
		if status == model.Failed {
			wantStatus = realization.AccountingInvalid
		}
		if report.Status() != wantStatus || !reflect.DeepEqual(findingCodes(report), []string{"model.unavailable"}) || len(report.Artifacts(borrow)) != 0 {
			t.Fatalf("model %q: %+v", status, report.Findings())
		}
		// Known independent file errors survive unavailable model resolution.
		report = realization.AccountFiles(checked, snapshot.Tree{}, reg, nil, realization.Layout{})
		if report.Status() != realization.AccountingInvalid || !reflect.DeepEqual(findingCodes(report), []string{"model.unavailable", "registry.missing_file"}) {
			t.Fatal(report.Findings())
		}
	}
	checked := checkedFacts()
	checked.Models[0].Status = model.NotCompleted
	checked.Models[0].Packages = nil
	report := realization.AccountFiles(checked, fileTree(t, "a"), registry(t, registration(t, owner(t, "lending"), artifact(t, "a", borrow))), nil, realization.Layout{})
	if report.Status() == realization.Accounted || slices.Contains(findingCodes(report), "registry.unknown_owner") {
		t.Fatal("unavailable own model was interpreted as empty")
	}
}

func TestIgnoringFilesCannotSupplyDefinitionRealization(t *testing.T) {
	checked := checkedFacts()
	report := realization.AccountFiles(checked, fileTree(t, "ignored.go", ".markitect/state/control"), realization.Registry{}, []realization.Classification{classified(t, "ignored.go", realization.IgnoredFile)}, realization.Layout{})
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
		checked := checkedFacts()
		change(&checked)
		if report := realization.AccountFiles(checked, snapshot.Tree{}, realization.Registry{}, nil, realization.Layout{}); report.Status() == realization.Accounted {
			t.Fatal("contradictory facts claimed accounting success")
		}
	}
	if report := realization.AccountFiles(checkedFacts(), snapshot.Tree{}, realization.Registry{}, []realization.Classification{{}}, realization.Layout{}); report.Status() != realization.AccountingInvalid {
		t.Fatal("zero classification accepted")
	}
}

func TestGenericAccountingAndAuthoritativeEmptyInventory(t *testing.T) {
	id := model.Identity{Package: "engine/api", ID: "Endpoint"}
	checked := model.Result{Status: model.Passed, Models: []model.ModelResult{{
		ID: "generic-project", Local: true, Status: model.Passed,
		Packages:    []model.Package{{Name: "engine"}, {Name: "engine/api"}, {Name: "empty"}},
		Definitions: []model.Definition{{Identity: id, ModelID: "generic-project"}},
	}}}
	reg := registry(t, registration(t, owner(t, "engine"), artifact(t, "source/Endpoint.go", id)))
	files := fileTree(t, "source/Endpoint.go")
	first := realization.AccountFiles(checked, files, reg, nil, realization.Layout{})
	slices.Reverse(checked.Models[0].Packages)
	second := realization.AccountFiles(checked, files, reg, nil, realization.Layout{})
	if first.Status() != realization.Accounted || !reflect.DeepEqual(first.Files(), second.Files()) || len(first.Artifacts(id)) != 1 {
		t.Fatal("generic project accounting depends on library names or inventory order")
	}
	checked.Models[0].Packages = nil
	checked.Models[0].Definitions = nil
	if report := realization.AccountFiles(checked, snapshot.Tree{}, realization.Registry{}, nil, realization.Layout{}); report.Status() != realization.Accounted {
		t.Fatal("a Passed empty inventory was not authoritative")
	}
	checked.Models[0].Status = model.NotCompleted
	if report := realization.AccountFiles(checked, snapshot.Tree{}, realization.Registry{}, nil, realization.Layout{}); report.Status() != realization.AccountingIncomplete {
		t.Fatal("an unavailable inventory was treated as an authoritative empty set")
	}
}

func TestOperationalStateExclusionHasAnExactBoundary(t *testing.T) {
	report := realization.AccountFiles(checkedFacts(), fileTree(t, ".markitect/statex/control"), realization.Registry{}, nil, realization.Layout{})
	if report.Status() != realization.AccountingInvalid || !reflect.DeepEqual(findingCodes(report), []string{"file.unexplained"}) {
		t.Fatal("a neighboring ordinary path was silently excluded as operational state")
	}
}
