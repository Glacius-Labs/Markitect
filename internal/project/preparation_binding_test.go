package project_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestPreparationPreviewKeepsWorkingAndFixedRevisionOnEveryAttempt(t *testing.T) {
	for _, revision := range []string{"", fixtureHead} {
		for _, complete := range []bool{true, false} {
			t.Run(fmt.Sprintf("revision=%s/complete=%v", revision, complete), func(t *testing.T) {
				selection, actual := fileCheckInputs(t)
				if !complete {
					selection.Registry = nil
					selection.Execution = nil
				}
				request := project.PreviewRequest{Selection: selection, Revision: revision}
				calls := 0
				source := controlledSource{preview: func(_ context.Context, selected project.PreviewRequest) (project.PreviewedInputs, error) {
					calls++
					if selected.Revision != revision {
						t.Fatal("fixed revision changed during recapture")
					}
					return project.PreviewedInputs{Raw: preparationObservation(t, selected.Selection, actual), Head: fixtureHead, Revision: selected.Revision}, nil
				}}
				result, err := project.NewPreparationService(project.NewModelService(nil, source)).Preview(context.Background(), request, model.Limits{})
				prepared, ok := result.Prepared()
				want := 1
				if !complete {
					want = 2
				}
				if err != nil || !ok || result.Status() != model.Passed || calls != want || !reflect.DeepEqual(result.Request(), request) {
					t.Fatalf("preview: %s %v", result.Status(), err)
				}
				if prepared.Model().Inputs().Revision != revision || !reflect.DeepEqual(prepared.Contexts().Execution().Model().Inputs(), prepared.Files().RegistryPreparation().Model().Inputs()) {
					t.Fatal("preview stage identities diverged")
				}
				for _, attempt := range result.Attempts() {
					if attempt.Model().Request().Revision != revision {
						t.Fatal("attempt lost revision")
					}
				}
			})
		}
	}
}

func TestPreparationRejectsSecondPreviewRevisionSubstitution(t *testing.T) {
	selection, actual := fileCheckInputs(t)
	selection.Registry = nil
	calls := 0
	source := controlledSource{preview: func(_ context.Context, selected project.PreviewRequest) (project.PreviewedInputs, error) {
		calls++
		revision := selected.Revision
		if calls == 2 {
			revision = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}
		return project.PreviewedInputs{Raw: preparationObservation(t, selected.Selection, actual), Head: fixtureHead, Revision: revision}, nil
	}}
	result, err := project.NewPreparationService(project.NewModelService(nil, source)).Preview(context.Background(), project.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
	if !errors.Is(err, project.ErrIncompleteInputs) || calls != 2 || result.Status() != model.NotCompleted {
		t.Fatalf("revision substitution: %s %v", result.Status(), err)
	}
	if _, ok := result.Prepared(); ok {
		t.Fatal("substituted preview exposed prepared values")
	}
	if result.Attempts()[1].Model().Inputs().Revision == fixtureHead || result.Attempts()[1].Model().Request().Revision != fixtureHead {
		t.Fatal("actual substituted observation or original request lost")
	}
}

func TestPreparationPreservesPartialAcquisitionsAndCancellation(t *testing.T) {
	for _, preview := range []bool{false, true} {
		for _, failureAt := range []int{0, 1, 2} {
			t.Run(fmt.Sprintf("preview=%v/failureAt=%d", preview, failureAt), func(t *testing.T) {
				selection, actual := fileCheckInputs(t)
				selection.Registry = nil
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				failure := errors.New("second acquisition refused after partial observations")
				if failureAt == 0 {
					cancel()
					failure = context.Canceled
				}
				calls := 0
				var observed project.RawInputs
				observe := func(selected project.InputSelection) (project.RawInputs, error) {
					calls++
					observed = preparationObservation(t, selected, actual)
					if calls == failureAt {
						observed.Registry = nil // actual partial observations are still useful
						if failureAt == 1 {
							cancel()
							return observed, context.Canceled
						}
						return observed, failure
					}
					return observed, nil
				}
				source := controlledSource{
					capture: func(_ context.Context, request project.CaptureRequest) (project.CapturedInputs, error) {
						raw, err := observe(request.Selection)
						return captured(raw), err
					},
					preview: func(_ context.Context, request project.PreviewRequest) (project.PreviewedInputs, error) {
						raw, err := observe(request.Selection)
						return project.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: request.Revision}, err
					},
				}
				service := project.NewPreparationService(project.NewModelService(source, source))
				var err, retained error
				var status model.Status
				var lastRaw project.RawInputs
				var attempts int
				var checked model.Result
				if preview {
					result, failure := service.Preview(ctx, project.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
					err, status, attempts = failure, result.Status(), len(result.Attempts())
					if _, ok := result.Prepared(); ok {
						t.Fatal("errored preview prepared")
					}
					last := result.Attempts()[attempts-1]
					lastRaw, checked, retained = last.Model().Inputs().Raw, last.Model().Checked(), last.Error()
				} else {
					result, failure := service.Capture(ctx, project.CaptureRequest{Selection: selection}, model.Limits{})
					err, status, attempts = failure, result.Status(), len(result.Attempts())
					if _, ok := result.Prepared(); ok {
						t.Fatal("errored capture prepared")
					}
					last := result.Attempts()[attempts-1]
					lastRaw, checked, retained = last.Model().Inputs().Raw, last.Model().Checked(), last.Error()
				}
				if failureAt == 1 {
					failure = context.Canceled
				}
				wantAttempts := max(1, failureAt)
				if !errors.Is(err, failure) || retained != err || status != model.NotCompleted || calls != failureAt || attempts != wantAttempts || checked.Status != model.NotCompleted || len(checked.Models) != 0 {
					t.Fatalf("partial/error binding: %s %v calls=%d attempts=%d", status, err, calls, attempts)
				}
				if calls > 0 && !reflect.DeepEqual(lastRaw, observed) {
					t.Fatal("partial observation lost or supplemented with old bytes")
				}
			})
		}
	}
}

func TestPreparationRetainsActualManifestPreflightFailure(t *testing.T) {
	for _, incomplete := range []bool{false, true} {
		selection, raw, inspection := partialManifest(t, incomplete)
		failure := fmt.Errorf("observed manifest: %w", project.NewManifestPreflightError(selection.Local, inspection))
		source := controlledSource{capture: func(context.Context, project.CaptureRequest) (project.CapturedInputs, error) {
			return captured(raw), failure
		}}
		result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
		if err != failure || result.Status() != inspection.Status || len(result.Attempts()) != 1 || !reflect.DeepEqual(result.Attempts()[0].Model().Inputs().Raw, raw) {
			t.Fatalf("preflight status/evidence: %s %v", result.Status(), err)
		}
		var observed *project.ManifestPreflightError
		if !errors.As(result.Attempts()[0].Error(), &observed) || !reflect.DeepEqual(observed.Inspection(), inspection) {
			t.Fatal("actual manifest diagnostics lost")
		}
	}
}

func TestPreparationRejectsInvalidExplicitSelectionsWithoutRepair(t *testing.T) {
	for _, invalid := range []string{"repository", "local", "dependency", "execution", "registry", "ignore", "runtime", "instruction", "path", "boundary"} {
		t.Run(invalid, func(t *testing.T) {
			selection, actual := fileCheckInputs(t)
			location := selection.Execution[0]
			switch invalid {
			case "repository":
				selection.Repository = ""
			case "local":
				selection.Local = project.ModelSelection{}
			case "dependency":
				selection.Dependencies = append(selection.Dependencies, selection.Dependencies[0])
			case "execution":
				selection.Execution = append(selection.Execution, location)
			case "registry":
				selection.Registry = append(selection.Registry, selection.Registry[0])
			case "ignore":
				selection.Ignore = []project.SourceLocation{location, location}
			case "runtime":
				selection.RuntimeProfiles = []project.SourceLocation{location, location}
			case "instruction":
				selection.Instructions = []project.SourceLocation{location, location}
			case "path":
				selection.RuntimeProfiles = []project.SourceLocation{{Boundary: "runtime"}}
			case "boundary":
				selection.Instructions = []project.SourceLocation{{Path: location.Path}}
			}
			source := controlledSource{capture: func(context.Context, project.CaptureRequest) (project.CapturedInputs, error) {
				t.Fatal("invalid request reached acquisition")
				return captured(actual), nil
			}}
			request := project.CaptureRequest{Selection: selection}
			result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), request, model.Limits{})
			if !errors.Is(err, project.ErrPreparationSelection) || result.Status() != model.Failed || len(result.Attempts()) != 0 || !reflect.DeepEqual(result.Request(), request) {
				t.Fatal("invalid original request repaired or lost")
			}
		})
	}
}

func TestPreparationResultsCopyRequestsAndKeepOnlyFinalAcquisitionBinding(t *testing.T) {
	selection, actual := fileCheckInputs(t)
	preparationInstructions(t, &actual)
	selection.Registry = nil
	calls := 0
	source := controlledSource{capture: func(_ context.Context, request project.CaptureRequest) (project.CapturedInputs, error) {
		calls++
		if calls == 2 {
			entries := actual.Census.Entries()
			for i, entry := range entries {
				if entry.Path().String() == "realization/library.md" {
					entries[i], _ = snapshot.NewEntry(entry.Path(), snapshot.Executable, []byte("New actual capture bytes"))
				}
			}
			actual.Census, _ = snapshot.NewTree(entries)
		}
		observed := captured(preparationObservation(t, request.Selection, actual))
		observed.Guard = project.PublicationGuard(fmt.Sprint("actual-guard-", calls))
		if calls == 2 {
			observed.Head = strings.Repeat("b", 40)
		}
		return observed, nil
	}}
	request := project.CaptureRequest{Selection: selection}
	result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), request, model.Limits{})
	prepared, ok := result.Prepared()
	if err != nil || !ok {
		t.Fatalf("fresh capture: %v %s", err, result.Status())
	}
	attempts := result.Attempts()
	if attempts[0].Model().Inputs().Head == prepared.Model().Inputs().Head || attempts[0].Model().Inputs().Raw.Census.Digest() == prepared.Model().Inputs().Raw.Census.Digest() || prepared.Model().Inputs().Guard != "actual-guard-2" {
		t.Fatal("new complete acquisition reused old evidence")
	}
	accounting, _ := prepared.Files().Accounting()
	if accounting.TreeID() != prepared.Model().Inputs().Raw.Census.Digest() {
		t.Fatal("accounting bound to old census")
	}
	request.Selection.Dependencies[0] = project.ModelSelection{}
	copyRequest := result.Request()
	copyRequest.Selection.Execution[0] = project.SourceLocation{}
	copyRequest.Selection.Dependencies[0] = project.ModelSelection{}
	attempts[0] = project.CapturedPreparationAttempt{}
	copyInputs := prepared.Model().Inputs()
	changedEntry := -1
	for i := range copyInputs.Raw.Local.Source.Entries {
		if len(copyInputs.Raw.Local.Source.Entries[i].Bytes) > 0 {
			copyInputs.Raw.Local.Source.Entries[i].Bytes[0] ^= 1
			changedEntry = i
			break
		}
	}
	copyInputs.Raw.Instructions[0] = project.SourceFile{}
	document := prepared.Execution().Partial()
	document.Instructions.Value[0].Files.Value[0] = "mutated"
	if result.Request().Selection.Dependencies[0].Boundary == "" || result.Request().Selection.Execution[0].Boundary == "" || result.Attempts()[0].Model().Inputs().Guard != "actual-guard-1" || !reflect.DeepEqual(result.Attempts()[1].Execution().Partial().Instructions.Value[0].Files.Value, []string{"ops/one.md", "ops/two.md", "ops/one.md", "ops/three.md"}) || changedEntry < 0 || reflect.DeepEqual(copyInputs.Raw.Local.Source.Entries[changedEntry].Bytes, prepared.Model().Inputs().Raw.Local.Source.Entries[changedEntry].Bytes) {
		t.Fatal("caller mutation escaped a copying accessor")
	}
}

func TestPreparationDoesNotReplaceAnExplicitWrongExecutionSelection(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	raw.Execution = []project.SourceFile{raw.Registry[0]}
	selection.Execution = []project.SourceLocation{raw.Execution[0].Location}
	request := project.CaptureRequest{Selection: selection}
	calls := 0
	source := controlledSource{capture: func(_ context.Context, selected project.CaptureRequest) (project.CapturedInputs, error) {
		calls++
		if !reflect.DeepEqual(selected, request) {
			t.Fatal("explicit wrong execution location silently replaced")
		}
		return captured(raw), nil
	}}
	result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), request, model.Limits{})
	if err != nil || result.Status() != model.Failed || calls != 1 || !reflect.DeepEqual(result.Request(), request) {
		t.Fatalf("wrong explicit source: %s %v", result.Status(), err)
	}
	if !fileDiagnostic(result.Attempts()[0].Files(), "execution.source_path") {
		t.Fatal("actual execution source diagnostic lost")
	}
}

func TestPreparationKeepsModelCheckFailureAndResourceLimitDistinct(t *testing.T) {
	for _, limited := range []bool{false, true} {
		selection, raw := fileCheckInputs(t)
		limits := model.Limits{Entries: 1}
		want, code := model.NotCompleted, "limit.entries"
		if !limited {
			limits = model.Limits{}
			want, code = model.Failed, "input.duplicate_path"
			raw.Local.Source.Entries = append(raw.Local.Source.Entries, raw.Local.Source.Entries[0])
		}
		source := controlledSource{capture: func(context.Context, project.CaptureRequest) (project.CapturedInputs, error) {
			return captured(raw), nil
		}}
		result, err := project.NewPreparationService(project.NewModelService(source, nil)).Capture(context.Background(), project.CaptureRequest{Selection: selection}, limits)
		if err != nil || result.Status() != want || len(result.Attempts()) != 1 || !hasDiagnostic(result.Attempts()[0].Model().Checked(), code) || result.Attempts()[0].Model().RequestedLimits() != limits {
			t.Fatalf("Model check status: %s %v", result.Status(), err)
		}
	}
}
