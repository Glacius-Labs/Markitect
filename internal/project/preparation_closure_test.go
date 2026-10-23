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

func preparationInstructions(t *testing.T, raw *project.RawInputs) {
	t.Helper()
	content := strings.Replace(documentedExecution(t), "checks:", "instructions: {'.': [ops/one.md, ops/two.md, ops/one.md, ops/three.md], lending: [ops/four.md, ops/five.md, ops/one.md]}\nchecks:", 1)
	rewriteFileCheckSource(t, raw, &raw.Execution[0], content)
	for _, name := range []string{"ops/one.md", "ops/two.md", "ops/three.md", "ops/four.md", "ops/five.md"} {
		censusFile(t, raw, name, snapshot.Executable, []byte("Actual project instruction.\r\n"))
	}
}

func TestPreparationDerivesEveryFileOnceWithoutResolvingExplicitBoundaries(t *testing.T) {
	selection, actual := fileCheckInputs(t)
	preparationInstructions(t, &actual)
	fileCheckSource(t, &selection, &actual, "ignore", ".markitect/ignore.yaml", snapshot.Executable, emptyFilePolicy, false)
	// Keep an already captured opaque registry locator and all model boundaries.
	selection.Registry = selection.Registry[:1]
	// A Runtime instruction with the same spelling cannot satisfy a project ref.
	external := project.SourceLocation{Boundary: "explicit-runtime-root", Path: path(t, "ops/one.md")}
	entry, err := snapshot.NewEntry(external.Path, snapshot.Regular, []byte("Distinct Runtime bytes"))
	if err != nil {
		t.Fatal(err)
	}
	selection.RuntimeProfiles = []project.SourceLocation{external}
	selection.Instructions = []project.SourceLocation{external}
	actual.RuntimeProfiles = []project.SourceFile{{Location: external, File: entry}}
	actual.Instructions = []project.SourceFile{{Location: external, File: entry}}
	var requests []project.CaptureRequest
	source := controlledSource{capture: func(_ context.Context, request project.CaptureRequest) (project.CapturedInputs, error) {
		requests = append(requests, request)
		return captured(preparationObservation(t, request.Selection, actual)), nil
	}}
	result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
	prepared, ok := result.Prepared()
	if err != nil || !ok || len(requests) != 2 {
		t.Fatalf("closure: %s %v %+v", result.Status(), err, result.Attempts()[len(result.Attempts())-1].Files().Diagnostics())
	}
	got := requests[1].Selection
	if got.Repository != selection.Repository || got.Local != selection.Local || !reflect.DeepEqual(got.Dependencies, selection.Dependencies) || !reflect.DeepEqual(got.RuntimeProfiles, selection.RuntimeProfiles) || got.Registry[0] != selection.Registry[0] || got.Instructions[0] != external {
		t.Fatal("explicit model, dependency, Runtime or opaque repository locations changed")
	}
	if len(got.Registry) != 2 || len(got.Ignore) != 1 || len(got.Instructions) != 6 {
		t.Fatalf("closure file count: %+v", got)
	}
	for index, name := range []string{"ops/one.md", "ops/two.md", "ops/three.md", "ops/four.md", "ops/five.md"} {
		if got.Instructions[index+1] != (project.SourceLocation{Boundary: selection.Repository, Path: path(t, name)}) {
			t.Fatal("derived acquisition order/locations changed")
		}
	}
	authored, _ := prepared.Execution().Document()
	if !reflect.DeepEqual(authored.Instructions.Value[0].Files.Value, []string{"ops/one.md", "ops/two.md", "ops/one.md", "ops/three.md"}) || !reflect.DeepEqual(authored.Instructions.Value[1].Files.Value, []string{"ops/four.md", "ops/five.md", "ops/one.md"}) {
		t.Fatal("authored order/repeats were repaired")
	}
	if !reflect.DeepEqual(prepared.Model().Inputs().Raw.Execution[0].File.Bytes(), actual.Execution[0].File.Bytes()) || prepared.Model().Inputs().Raw.Instructions[1].File.Mode() != snapshot.Executable {
		t.Fatal("source bytes/modes changed")
	}
	if len(result.Attempts()[0].Model().Inputs().Raw.Instructions) != 1 || len(result.Attempts()[0].Model().Inputs().Raw.Ignore) != 0 {
		t.Fatal("old Raw was extended")
	}
}

func TestPreparationStopsWhenSecondExecutionChangesClosure(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(map[bool]string{false: "add", true: "remove"}[remove], func(t *testing.T) {
			selection, actual := fileCheckInputs(t)
			preparationInstructions(t, &actual)
			selection.Registry = nil // Force exactly one full recapture.
			calls := 0
			source := controlledSource{capture: func(_ context.Context, request project.CaptureRequest) (project.CapturedInputs, error) {
				calls++
				if calls > 2 {
					t.Fatal("unbounded acquisition loop")
				}
				if calls == 2 {
					content := strings.Replace(string(actual.Execution[0].File.Bytes()), "ops/two.md", "ops/new.md", 1)
					if remove {
						content = documentedExecution(t)
					} else {
						censusFile(t, &actual, "ops/new.md", snapshot.Regular, []byte("new closure"))
					}
					rewriteFileCheckSource(t, &actual, &actual.Execution[0], content)
				}
				return captured(preparationObservation(t, request.Selection, actual)), nil
			}}
			result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
			if !errors.Is(err, project.ErrPreparationClosure) || result.Status() != model.NotCompleted || calls != 2 || len(result.Attempts()) != 2 {
				t.Fatalf("moving closure: %s %v", result.Status(), err)
			}
			if _, ok := result.Prepared(); ok {
				t.Fatal("moving closure exposed complete preparation")
			}
			first, second := result.Attempts()[0], result.Attempts()[1]
			if first.Execution().Partial().Instructions == nil || reflect.DeepEqual(first.Model().Inputs().Raw.Execution[0].File.Bytes(), second.Model().Inputs().Raw.Execution[0].File.Bytes()) {
				t.Fatal("attempts lost distinct original observations")
			}
		})
	}
}

func TestPreparationRefusesUncapturedClosureOnSecondFullAcquisition(t *testing.T) {
	for _, missing := range []string{"registry", "ignore", "instructions"} {
		t.Run(missing, func(t *testing.T) {
			selection, actual := fileCheckInputs(t)
			preparationInstructions(t, &actual)
			fileCheckSource(t, &selection, &actual, "ignore", ".markitect/ignore.yaml", snapshot.Regular, emptyFilePolicy, false)
			selection.Registry = nil
			calls := 0
			source := controlledSource{capture: func(_ context.Context, request project.CaptureRequest) (project.CapturedInputs, error) {
				calls++
				raw := preparationObservation(t, request.Selection, actual)
				if calls == 2 {
					switch missing {
					case "registry":
						raw.Registry = nil
					case "ignore":
						raw.Ignore = nil
					case "instructions":
						raw.Instructions = nil
					}
				}
				return captured(raw), nil
			}}
			result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
			if !errors.Is(err, project.ErrIncompleteInputs) || result.Status() != model.NotCompleted || calls != 2 || result.Attempts()[1].Error() != err {
				t.Fatalf("uncaptured %s: %s %v", missing, result.Status(), err)
			}
			if _, ok := result.Prepared(); ok {
				t.Fatal("partial capture exposed prepared values")
			}
			if result.Attempts()[1].Model().Inputs().Raw.Census.Digest() != actual.Census.Digest() {
				t.Fatal("partial second observation lost")
			}
		})
	}
}

func TestPreparationRetainsKnownFailureAndActualAccounting(t *testing.T) {
	for _, invalid := range []string{"execution", "ignore", "instruction", "unexplained", "registry"} {
		t.Run(invalid, func(t *testing.T) {
			selection, raw := fileCheckInputs(t)
			switch invalid {
			case "execution":
				rewriteFileCheckSource(t, &raw, &raw.Execution[0], strings.Replace(documentedExecution(t), "direct", "unknown", 1))
			case "ignore":
				fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, "unknown: true\n", true)
			case "instruction":
				rewriteFileCheckSource(t, &raw, &raw.Execution[0], strings.Replace(documentedExecution(t), "checks:", "instructions: {'.': [ops/absent.md]}\nchecks:", 1))
			case "unexplained":
				censusFile(t, &raw, "unexplained.txt", snapshot.Regular, []byte("actual unowned file"))
			case "registry":
				rewriteFileCheckSource(t, &raw, &raw.Registry[0], "unknown: true\n")
			}
			calls := 0
			source := controlledSource{capture: func(context.Context, project.CaptureRequest) (project.CapturedInputs, error) {
				calls++
				return captured(raw), nil
			}}
			result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
			if err != nil || result.Status() != model.Failed || calls != 1 {
				t.Fatalf("known invalid %s: %s %v", invalid, result.Status(), err)
			}
			if _, ok := result.Prepared(); ok {
				t.Fatal("known failure exposed prepared values")
			}
			attempt := result.Attempts()[0]
			if len(attempt.Files().Diagnostics()) == 0 && invalid != "unexplained" {
				t.Fatal("original stage diagnostics lost")
			}
			accounting, ran := attempt.Files().Accounting()
			if invalid == "unexplained" && (!ran || accounting.Status() != realization.AccountingInvalid || len(accounting.Findings()) == 0) {
				t.Fatal("actual failed accounting report lost")
			}
			if invalid == "instruction" && (!ran || accounting.Status() != realization.Accounted) {
				t.Fatal("known missing instruction incorrectly replaced actual accounting")
			}
		})
	}
}

func TestPreparationRetainsInitialAccountingBeforeCapturingObservedPolicy(t *testing.T) {
	selection, actual := fileCheckInputs(t)
	censusFile(t, &actual, ".editor/library-session.json", snapshot.Regular, []byte("editor"))
	fileCheckSource(t, &selection, &actual, "ignore", ".markitect/ignore.yaml", snapshot.Regular, editorFilePolicy, false)
	calls := 0
	source := controlledSource{capture: func(_ context.Context, request project.CaptureRequest) (project.CapturedInputs, error) {
		calls++
		return captured(preparationObservation(t, request.Selection, actual)), nil
	}}
	result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || result.Status() != model.Passed || calls != 2 {
		t.Fatalf("policy recapture: %s %v", result.Status(), err)
	}
	first := result.Attempts()[0].Files()
	accounting, ran := first.Accounting()
	if first.Status() != model.NotCompleted || !ran || accounting.Status() != realization.AccountingInvalid || len(accounting.Findings()) != 1 || accounting.Findings()[0].Code != "file.unexplained" || !fileDiagnostic(first, "files.ignore_uncaptured") {
		t.Fatal("original incomplete accounting report or policy diagnostics replaced")
	}
	prepared, _ := result.Prepared()
	final, _ := prepared.Files().Accounting()
	if final.Status() != realization.Accounted || len(final.Findings()) != 0 {
		t.Fatal("new actual policy failed to account files")
	}
}

func TestPreparationClosesKnownSourcesBesideInvalidExecutionOrRegistry(t *testing.T) {
	for _, invalid := range []string{"execution", "registry"} {
		for _, preview := range []bool{false, true} {
			t.Run(invalid+map[bool]string{false: "/capture", true: "/preview"}[preview], func(t *testing.T) {
				selection, actual := fileCheckInputs(t)
				preparationInstructions(t, &actual)
				fileCheckSource(t, &selection, &actual, "ignore", ".markitect/ignore.yaml", snapshot.Regular, emptyFilePolicy, false)
				if invalid == "execution" {
					rewriteFileCheckSource(t, &actual, &actual.Execution[0], strings.Replace(string(actual.Execution[0].File.Bytes()), "direct", "unknown", 1))
				} else {
					rewriteFileCheckSource(t, &actual, &actual.Registry[0], "unknown: true\n")
				}
				selection.Registry = selection.Registry[:1]
				calls := 0
				observe := func(selected project.InputSelection) project.RawInputs {
					calls++
					return preparationObservation(t, selected, actual)
				}
				source := controlledSource{
					capture: func(_ context.Context, request project.CaptureRequest) (project.CapturedInputs, error) {
						return captured(observe(request.Selection)), nil
					},
					preview: func(_ context.Context, request project.PreviewRequest) (project.PreviewedInputs, error) {
						return project.PreviewedInputs{Raw: observe(request.Selection), Head: fixtureHead, Revision: request.Revision}, nil
					},
				}
				service := project.NewPreparationService(project.NewModelService(source, source))
				var observed project.RawInputs
				var status model.Status
				var err error
				var first, last []project.FileCheckDiagnostic
				if preview {
					result, failure := service.Preview(context.Background(), project.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
					err, status = failure, result.Status()
					attempts := result.Attempts()
					observed, first, last = attempts[len(attempts)-1].Model().Inputs().Raw, attempts[0].Files().Diagnostics(), attempts[len(attempts)-1].Files().Diagnostics()
					if _, ok := result.Prepared(); ok {
						t.Fatal("known invalid preview prepared")
					}
				} else {
					result, failure := service.Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
					err, status = failure, result.Status()
					attempts := result.Attempts()
					observed, first, last = attempts[len(attempts)-1].Model().Inputs().Raw, attempts[0].Files().Diagnostics(), attempts[len(attempts)-1].Files().Diagnostics()
					if _, ok := result.Prepared(); ok {
						t.Fatal("known invalid capture prepared")
					}
				}
				if err != nil || status != model.Failed || calls != 2 || len(observed.Registry) != 2 || len(observed.Ignore) != 1 || len(observed.Instructions) != 5 || len(first) == 0 || len(last) == 0 {
					t.Fatalf("known failure suppressed full acquisition: %s %v calls=%d", status, err, calls)
				}
			})
		}
	}
}
