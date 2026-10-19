package apply_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestPreparationStillMissingClosurePreservesOnlyCurrentFailure(t *testing.T) {
	for _, mode := range []string{"capture", "working", "fixed"} {
		for _, currentFailure := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/currentFailure=%v", mode, currentFailure), func(t *testing.T) {
				selection, actual := fileCheckInputs(t)
				preparationInstructions(t, &actual)
				validExecution := string(actual.Execution[0].File.Bytes())
				rewriteFileCheckSource(t, &actual, &actual.Execution[0], strings.Replace(validExecution, "shared-workspace-v1", "unknown", 1))
				calls := 0
				observe := func(selected apply.InputSelection) apply.RawInputs {
					calls++
					if calls > 2 {
						t.Fatal("third acquisition attempted")
					}
					if calls == 2 && !currentFailure {
						rewriteFileCheckSource(t, &actual, &actual.Execution[0], validExecution)
					}
					raw := preparationObservation(t, selected, actual)
					if calls == 2 {
						// Retain the selected file's actual bytes and location, but no
						// repository binding. Such an instruction observation cannot
						// satisfy the authored repository ref. The ModelService can
						// compile; FileCheck still reports instruction_uncaptured.
						raw.Instructions[0].RepositoryPath = snapshot.Path{}
					}
					return raw
				}
				source := controlledSource{
					capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
						raw := observe(request.Selection)
						return apply.CapturedInputs{Raw: raw, Head: fixtureHead, Guard: apply.PublicationGuard(fmt.Sprint("actual-guard-", calls))}, nil
					},
					preview: func(_ context.Context, request apply.PreviewRequest) (apply.PreviewedInputs, error) {
						return apply.PreviewedInputs{Raw: observe(request.Selection), Head: fixtureHead, Revision: request.Revision}, nil
					},
				}
				service := apply.NewPreparationService(apply.NewModelService(source, source))
				var err error
				var status model.Status
				var firstExecution, secondExecution apply.AuthoredExecution
				var firstDiagnostics, secondDiagnostics []apply.FileCheckDiagnostic
				var firstStatus, secondStatus model.Status
				if mode == "capture" {
					request := apply.CaptureRequest{Selection: selection}
					result, failure := service.Capture(context.Background(), request, model.Limits{})
					err, status = failure, result.Status()
					attempts := result.Attempts()
					if len(attempts) != 2 || !reflect.DeepEqual(result.Request(), request) {
						t.Fatal("attempts or original request lost")
					}
					first, second := attempts[0], attempts[1]
					firstExecution, secondExecution = first.Execution().Partial(), second.Execution().Partial()
					firstDiagnostics, secondDiagnostics = first.Files().Diagnostics(), second.Files().Diagnostics()
					firstStatus, secondStatus = first.Execution().Status(), second.Execution().Status()
					if second.Error() != nil || second.Model().Compilation().Status != model.Passed || second.Model().Inputs().Guard != "actual-guard-2" || first.Model().Inputs().Guard != "actual-guard-1" {
						t.Fatal("acquisition error or old guard obscured the actual second stage")
					}
					if _, ok := result.Prepared(); ok {
						t.Fatal("incomplete capture exposed prepared values")
					}
				} else {
					request := apply.PreviewRequest{Selection: selection}
					if mode == "fixed" {
						request.Revision = fixtureHead
					}
					result, failure := service.Preview(context.Background(), request, model.Limits{})
					err, status = failure, result.Status()
					attempts := result.Attempts()
					if len(attempts) != 2 || !reflect.DeepEqual(result.Request(), request) {
						t.Fatal("attempts or original request lost")
					}
					first, second := attempts[0], attempts[1]
					firstExecution, secondExecution = first.Execution().Partial(), second.Execution().Partial()
					firstDiagnostics, secondDiagnostics = first.Files().Diagnostics(), second.Files().Diagnostics()
					firstStatus, secondStatus = first.Execution().Status(), second.Execution().Status()
					if second.Error() != nil || second.Model().Compilation().Status != model.Passed || second.Model().Request().Revision != request.Revision || first.Model().Request().Revision != request.Revision {
						t.Fatal("acquisition error or changed revision obscured the actual second stage")
					}
					if _, ok := result.Prepared(); ok {
						t.Fatal("incomplete preview exposed prepared values")
					}
				}
				want, wantExecution := model.NotCompleted, model.Passed
				if currentFailure {
					want, wantExecution = model.Failed, model.Failed
				}
				if !errors.Is(err, apply.ErrPreparationClosure) || calls != 2 || status != want || firstStatus != model.Failed || secondStatus != wantExecution {
					t.Fatalf("current second failure precedence: status=%s want=%s err=%v calls=%d first=%s second=%s", status, want, err, calls, firstStatus, secondStatus)
				}
				if firstExecution.Strategy.Value != "unknown" || len(firstDiagnostics) == 0 || len(secondDiagnostics) == 0 {
					t.Fatal("original attempts or reports replaced")
				}
				if currentFailure && secondExecution.Strategy.Value != "unknown" || !currentFailure && secondExecution.Strategy.Value != "shared-workspace-v1" {
					t.Fatal("second current execution replaced by an old observation")
				}
				uncaptured := false
				for _, diagnostic := range secondDiagnostics {
					uncaptured = uncaptured || diagnostic.Code == "files.instruction_uncaptured"
				}
				if !uncaptured {
					t.Fatal("test did not retain a second-stage closure gap")
				}
			})
		}
	}
}
