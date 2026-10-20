package apply_test

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestFileCheckBindsProtectionToActualManifestRoot(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	raw.Local.Root = area(t, "incorrect-root")
	policy := "apiVersion: markitect.ignore/v1\nfiles:\n"
	for _, file := range raw.Local.Files.Entries() {
		if strings.HasPrefix(file.Path().String(), "model/") {
			policy += "  - {path: .markitect/" + file.Path().String() + ", reason: cannot-hide-model}\n"
		}
	}
	fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, policy, true)
	result := checkedFiles(t, selection, raw)
	if result.Status() != model.Failed || !fileDiagnostic(result, "files.model_root_mismatch") {
		t.Fatalf("contradictory root hid canonical inputs: %s %+v", result.Status(), result.Diagnostics())
	}
}

func TestFileCheckFutureProtectionsDoNotCreatePhantomClasses(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	result := checkedFiles(t, selection, raw)
	protections, ok := result.Protections()
	if !ok {
		t.Fatal("fixture failed")
	}
	for _, name := range []string{".markitect/model/lending/future.yaml", ".markitect/registry/future/registry.yaml", ".markitect/ignore.yaml", ".markitect/state/control.json", ".git/future-index"} {
		covered := false
		for _, protection := range protections {
			if file, exact := protection.File(); exact && file == path(t, name) {
				covered = true
			}
			if directory, area := protection.Directory(); area && directory.Contains(path(t, name)) {
				covered = true
			}
		}
		if !covered {
			t.Fatalf("future input unprotected: %s", name)
		}
	}
	classes, _ := result.Classifications()
	for _, class := range classes {
		if _, found := raw.Census.Lookup(class.Path()); !found {
			t.Fatalf("phantom classification: %s", class.Path())
		}
	}
	for _, name := range []string{".markitect/notes.txt", "realization/new.go"} {
		for _, protection := range protections {
			if directory, area := protection.Directory(); area && directory.Contains(path(t, name)) {
				t.Fatalf("acquisition/creation boundary became blanket protection: %s", name)
			}
		}
	}
}

func TestFileCheckAccountsOnlyDeclaredRootsInsideAcquisitionAreas(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	dependency := &raw.Dependencies[0]
	dependency.RepositoryArea = area(t, "vendor/vocabulary")
	for _, file := range dependency.Files.Entries() {
		censusFile(t, &raw, "vendor/vocabulary/"+file.Path().String(), file.Mode(), file.Bytes())
	}
	fileCheckSource(t, &selection, &raw, "runtime", ".markitect/runtime.yaml", snapshot.Regular, "literal Runtime profile source, no decoder or readiness claim", true)
	fileCheckSource(t, &selection, &raw, "instruction", "ops/empty.md", snapshot.Regular, "", true)
	censusFile(t, &raw, ".markitect/notes.txt", snapshot.Regular, []byte("not a model input"))
	censusFile(t, &raw, "vendor/vocabulary/notes.txt", snapshot.Regular, []byte("not a dependency Definition"))
	result := checkedFiles(t, selection, raw)
	accounting, _ := result.Accounting()
	if result.Status() != model.Failed || len(accounting.Findings()) != 2 {
		t.Fatalf("acquisition extras silently exempted: %+v", accounting.Findings())
	}
	policy := "apiVersion: markitect.ignore/v1\nfiles: [{path: .markitect/notes.txt, reason: local-notes}, {path: vendor/vocabulary/notes.txt, reason: acquisition-notes}]\n"
	fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, policy, true)
	result = checkedFiles(t, selection, raw)
	accounting, _ = result.Accounting()
	if result.Status() != model.Passed || accounting.Status() != realization.Accounted {
		t.Fatalf("explicit mapped input classification failed: %+v %+v", result.Diagnostics(), accounting.Findings())
	}
	for _, file := range accounting.Files() {
		if strings.HasPrefix(file.Entry().Path().String(), "vendor/vocabulary/model/") && file.Class() != realization.CanonicalInput {
			t.Fatal("actual in-repository dependency was not a canonical input")
		}
	}
}

func TestFileCheckObservedModelTreeFileRequiresActualModelCapture(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	censusFile(t, &raw, ".markitect/model/lending/unobserved.yaml", snapshot.Regular, []byte("uncaptured Definition"))
	result := checkedFiles(t, selection, raw)
	if result.Status() != model.NotCompleted || !fileDiagnostic(result, "files.model_source_uncaptured") {
		t.Fatalf("unselected tree file silently exempted: %+v", result.Diagnostics())
	}
}
