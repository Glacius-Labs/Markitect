package project_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestFileCheckClosesEveryAuthoredInstructionListAndPreservesRepeats(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	content := strings.Replace(string(raw.Execution[0].File.Bytes()), "checks:", "instructions: {'.': [ops/one.md, ops/two.md, ops/one.md, ops/three.md], lending: [ops/four.md, ops/five.md, ops/one.md]}\nchecks:", 1)
	rewriteFileCheckSource(t, &raw, &raw.Execution[0], content)
	for _, name := range []string{"ops/one.md", "ops/two.md", "ops/three.md", "ops/four.md", "ops/five.md"} {
		fileCheckSource(t, &selection, &raw, "instruction", name, snapshot.Executable, "Original instructions.\r\n", true)
	}
	// Acquisition retains repetitions too; identical complete observations must
	// not turn a valid repeated authored list into a duplicate catalog failure.
	selection.Instructions = append(selection.Instructions, selection.Instructions[0])
	raw.Instructions = append(raw.Instructions, raw.Instructions[0])
	before := registryModel(t, selection, raw).DecodeExecution().Partial()
	result := checkedFiles(t, selection, raw)
	accounting, ran := result.Accounting()
	if result.Status() != model.Passed || !ran || accounting.Status() != realization.Accounted {
		t.Fatalf("valid repeated sources rejected: %+v", result.Diagnostics())
	}
	if !reflect.DeepEqual(result.RegistryPreparation().Model().DecodeExecution().Partial(), before) || !reflect.DeepEqual(result.RegistryPreparation().Model().Inputs().Raw, raw) {
		t.Fatal("instruction order/repeats/original source changed")
	}
	for _, file := range accounting.Files() {
		if strings.HasPrefix(file.Entry().Path().String(), "ops/") && file.Class() != realization.MarkitectInput {
			t.Fatal("instruction was treated as an ordinary artifact")
		}
	}
}

func TestFileCheckDoesNotResolveProjectInstructionsFromExternalLocators(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	location := project.SourceLocation{Boundary: "explicit-external", Path: path(t, "ops/one.md")}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte("external bytes"))
	if err != nil {
		t.Fatal(err)
	}
	selection.Instructions = []project.SourceLocation{location}
	raw.Instructions = []project.SourceFile{{Location: location, File: entry}}
	selection.RuntimeProfiles = []project.SourceLocation{location}
	raw.RuntimeProfiles = []project.SourceFile{{Location: location, File: entry}}
	result := checkedFiles(t, selection, raw)
	if result.Status() != model.Passed {
		t.Fatalf("external sources gained phantom Census coordinates: %+v", result.Diagnostics())
	}
	classes, _ := result.Classifications()
	for _, class := range classes {
		if class.Path() == location.Path {
			t.Fatal("external input became a repository classification")
		}
	}
	rewriteFileCheckSource(t, &raw, &raw.Execution[0], strings.Replace(documentedExecution(t), "checks:", "instructions: {'.': [ops/one.md]}\nchecks:", 1))
	censusFile(t, &raw, "ops/one.md", snapshot.Regular, []byte("actual project bytes"))
	result = checkedFiles(t, selection, raw)
	if result.Status() != model.NotCompleted || !fileDiagnostic(result, "files.instruction_uncaptured") {
		t.Fatal("opaque external locator satisfied project instruction capture")
	}
}

func TestFileCheckValidatesInstructionPathsAndUnusedOverrides(t *testing.T) {
	for _, ref := range []string{"../escape.md", "ops/", "/absolute.md", "ops/missing.md"} {
		selection, raw := fileCheckInputs(t)
		content := strings.Replace(documentedExecution(t), "checks:", "instructions: {lending: ['"+ref+"']}\nchecks:", 1)
		rewriteFileCheckSource(t, &raw, &raw.Execution[0], content)
		if result := checkedFiles(t, selection, raw); result.Status() != model.Failed {
			t.Fatalf("invalid instruction reference passed: %s", ref)
		}
	}
	selection, raw := fileCheckInputs(t)
	content := strings.Replace(documentedExecution(t), "checks:", "instructions: {unknown: [ops/unavailable.md]}\nchecks:", 1)
	rewriteFileCheckSource(t, &raw, &raw.Execution[0], content)
	result := checkedFiles(t, selection, raw)
	if result.Status() != model.Failed || !fileDiagnostic(result, "execution.package") || !fileDiagnostic(result, "files.instruction_missing") {
		t.Fatal("invalid/unused override suppressed instruction closure")
	}
}

func TestFileCheckInstructionAbsenceAndUncapturedSources(t *testing.T) {
	for _, present := range []bool{false, true} {
		selection, raw := fileCheckInputs(t)
		content := strings.Replace(string(raw.Execution[0].File.Bytes()), "checks:", "instructions: {'.': [ops/one.md]}\nchecks:", 1)
		rewriteFileCheckSource(t, &raw, &raw.Execution[0], content)
		if present {
			censusFile(t, &raw, "ops/one.md", snapshot.Regular, []byte("observed, not captured"))
		}
		result := checkedFiles(t, selection, raw)
		want, code := model.Failed, "files.instruction_missing"
		if present {
			want, code = model.NotCompleted, "files.instruction_uncaptured"
		}
		accounting, ran := result.Accounting()
		if result.Status() != want || !fileDiagnostic(result, code) || !ran || accounting.Status() != realization.Accounted {
			t.Fatalf("instruction closure lost: %s %+v", result.Status(), result.Diagnostics())
		}
		for _, diagnostic := range result.Diagnostics() {
			if diagnostic.Code == code && (diagnostic.Source.Field != "/instructions/./0" || diagnostic.Source.ContentID != raw.Execution[0].File.ContentID()) {
				t.Fatal("instruction reference source lost")
			}
		}
	}
}

func TestFileCheckInstructionCannotBecomeIgnoredOrRegistered(t *testing.T) {
	for _, disposition := range []string{"ignored", "artifact"} {
		selection, raw := fileCheckInputs(t)
		rewriteFileCheckSource(t, &raw, &raw.Execution[0], strings.Replace(string(raw.Execution[0].File.Bytes()), "checks:", "instructions: {'.': [ops/one.md]}\nchecks:", 1))
		censusFile(t, &raw, "ops/one.md", snapshot.Regular, []byte("uncaptured instructions"))
		if disposition == "ignored" {
			fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, "apiVersion: markitect.ignore/v1\nfiles: [{path: ops/one.md, reason: conceal}]\n", true)
		} else {
			rewriteFileCheckSource(t, &raw, &raw.Registry[0], rootRegistry+"  - path: ops/one.md\n    form: doc\n    realizes: [library/library]\n    why: Incorrect artifact claim.\n")
		}
		result := checkedFiles(t, selection, raw)
		accounting, ran := result.Accounting()
		if result.Status() != model.Failed || !ran || accounting.Status() != realization.AccountingInvalid || !fileDiagnostic(result, "files.instruction_uncaptured") {
			t.Fatalf("instruction conflict failed to dominate missing source: %s %+v %+v", result.Status(), result.Diagnostics(), accounting.Findings())
		}
		for _, file := range accounting.Files() {
			if file.Entry().Path().String() == "ops/one.md" && file.Class() != realization.ConflictingFile {
				t.Fatal("declared instruction accepted as ignore/artifact")
			}
		}
	}
}
