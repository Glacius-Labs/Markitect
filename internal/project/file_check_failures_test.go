package project_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestFileCheckPreservesRegistryAndExecutionRefusals(t *testing.T) {
	for _, tc := range []struct {
		name       string
		want       model.Status
		accounting bool
	}{
		{"invalid registry", model.Failed, false}, {"uncaptured registry", model.NotCompleted, false},
		{"invalid execution", model.Failed, true}, {"uncaptured execution", model.NotCompleted, true},
		{"execution tool limit", model.NotCompleted, true}, {"missing policy plus invalid execution", model.Failed, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := fileCheckInputs(t)
			switch tc.name {
			case "invalid registry":
				rewriteFileCheckSource(t, &raw, &raw.Registry[0], strings.Replace(rootRegistry, "form: doc", "form: unknown", 1))
			case "uncaptured registry":
				selection.Registry, raw.Registry = selection.Registry[:1], raw.Registry[:1]
			case "invalid execution":
				rewriteFileCheckSource(t, &raw, &raw.Execution[0], strings.Replace(documentedExecution(t), "timeout: 2m", "timeout: 0s", 1))
			case "uncaptured execution":
				selection.Execution, raw.Execution = nil, nil
			case "execution tool limit":
				rewriteFileCheckSource(t, &raw, &raw.Execution[0], strings.Replace(documentedExecution(t), "timeout: 2m", "timeout: 999999999999999999999h", 1))
			case "missing policy plus invalid execution":
				fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, emptyFilePolicy, false)
				rewriteFileCheckSource(t, &raw, &raw.Execution[0], strings.Replace(documentedExecution(t), "timeout: 2m", "timeout: false", 1))
			}
			parent := registryModel(t, selection, raw).PrepareRegistry()
			result := parent.CheckFiles()
			_, ran := result.Accounting()
			if result.Status() != tc.want || ran != tc.accounting || !reflect.DeepEqual(result.RegistryPreparation().Model().Inputs().Raw, raw) || !reflect.DeepEqual(result.RegistryPreparation().Diagnostics(), parent.Diagnostics()) {
				t.Fatalf("refusal collapsed or observations lost: %s %+v", result.Status(), result.Diagnostics())
			}
			if values, ok := result.Protections(); ok || len(values) != 0 {
				t.Fatal("partial source restrictions exposed as a complete set")
			}
		})
	}
}

func TestFileCheckPartialAcquisitionAndSourceDriftCannotPass(t *testing.T) {
	for _, name := range []string{"acquisition error", "instruction bytes", "instruction mode", "model failed", "model limit"} {
		t.Run(name, func(t *testing.T) {
			selection, raw := fileCheckInputs(t)
			limits := model.Limits{}
			var failure error
			switch name {
			case "acquisition error":
				failure = errors.New("source observation interrupted")
			case "instruction bytes", "instruction mode":
				fileCheckSource(t, &selection, &raw, "instruction", "ops/one.md", snapshot.Regular, "captured", true)
				mode, content := snapshot.Regular, "changed"
				if name == "instruction mode" {
					mode, content = snapshot.Executable, "captured"
				}
				entry, err := snapshot.NewEntry(raw.Instructions[0].Location.Path, mode, []byte(content))
				if err != nil {
					t.Fatal(err)
				}
				raw.Instructions[0].File = entry
			case "model failed":
				selection.Dependencies, raw.Dependencies = nil, nil
			case "model limit":
				limits.Entries = 1
			}
			stage, err := captureService(captured(raw), failure).CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, limits)
			if name == "acquisition error" && !errors.Is(err, failure) {
				t.Fatal("source error lost")
			}
			if strings.HasPrefix(name, "instruction") && !errors.Is(err, project.ErrIncompleteInputs) {
				t.Fatal("instruction source drift passed parent binding")
			}
			result := stage.PrepareRegistry().CheckFiles()
			want := model.NotCompleted
			if name == "model failed" {
				want = model.Failed
			}
			if result.Status() != want || !reflect.DeepEqual(result.RegistryPreparation().Model().Inputs().Raw, raw) {
				t.Fatal("nonpassed source/model promoted or discarded")
			}
			if _, ran := result.Accounting(); ran {
				t.Fatal("partial registry was used for invented accounting")
			}
		})
	}
}

func TestFileCheckWrongIgnoreSourceAndDuplicateSelectionFail(t *testing.T) {
	for _, name := range []string{"wrong path", "duplicate source"} {
		selection, raw := fileCheckInputs(t)
		path := ".markitect/ignore.yaml"
		if name == "wrong path" {
			path = ".markitect/other.yaml"
		}
		fileCheckSource(t, &selection, &raw, "ignore", path, snapshot.Regular, emptyFilePolicy, true)
		if name == "duplicate source" {
			selection.Ignore = append(selection.Ignore, selection.Ignore[0])
			raw.Ignore = append(raw.Ignore, raw.Ignore[0])
		}
		if result := checkedFiles(t, selection, raw); result.Status() != model.Failed {
			t.Fatalf("ambiguous policy source passed: %+v", result.Diagnostics())
		}
	}
}

func TestFileCheckGitignoredUntrackedFileStillNeedsAccounting(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	// Source's actual Census includes this even when Git exempts it from the
	// untracked cleanliness check. Accounting has no Gitignore input or bypass.
	censusFile(t, &raw, ".cache/editor.bin", snapshot.Regular, []byte("observed Git-ignored untracked file"))
	result := checkedFiles(t, selection, raw)
	accounting, ran := result.Accounting()
	if result.Status() != model.Failed || !ran || accounting.Status() != realization.AccountingInvalid || len(accounting.Findings()) != 1 || accounting.Findings()[0].Code != "file.unexplained" {
		t.Fatal("Gitignored census file silently skipped")
	}
}
