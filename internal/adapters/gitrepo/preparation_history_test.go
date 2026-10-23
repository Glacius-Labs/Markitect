package gitrepo

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

func TestM1GitPreparationRetainsIgnoredUntrackedCensusAndHistoricalInstructions(t *testing.T) {
	repo, selection, historical := fileCheckSeamFixture(t)
	selection.Execution, selection.Registry, selection.Ignore, selection.Instructions = nil, nil, nil, nil
	const currentInstruction = "Today's committed manager instructions.\r\n"
	repo.Write("ops/manager.md", currentInstruction)
	head := repo.Commit("Advance HEAD with current instruction bytes")
	const today = "scratch/today.ignored"
	const untracked = "\x00\xffActual ignored untracked bytes.\r\n"
	repo.Write(today, untracked)
	if head == historical || repo.Git("check-ignore", "--", today) != today || repo.Git("ls-files", "--", today) != "" || repo.Git("status", "--porcelain", "--untracked-files=all") != "" {
		t.Fatal("expected distinct historical HEAD and a clean repository with a Git-ignored untracked file")
	}
	for _, mode := range []string{"capture", "working", "fixed"} {
		t.Run(mode, func(t *testing.T) {
			port := &recordingPreparationGit{repository: adapter(t)}
			service := project.NewPreparationService(project.NewModelService(port, port))
			var first, last project.RawInputs
			var accounting realization.Accounting
			var status model.Status
			if mode == "capture" {
				result, err := service.Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
				if err != nil || len(result.Attempts()) != 2 || len(port.captures) != 2 {
					t.Fatalf("actual ignored-file capture: %v", err)
				}
				if _, available := result.Prepared(); available {
					t.Fatal("unexplained Git-ignored file became complete preparation")
				}
				attempts := result.Attempts()
				for i, attempt := range attempts {
					if !reflect.DeepEqual(attempt.Model().Inputs(), port.captures[i]) || !reflect.DeepEqual(attempt.Model().Request(), port.captureRequests[i]) {
						t.Fatal("current capture replaced an original attempt")
					}
				}
				first, last = attempts[0].Model().Inputs().Raw, attempts[1].Model().Inputs().Raw
				accounting, _ = attempts[1].Files().Accounting()
				status = result.Status()
				if attempts[1].Model().Inputs().Head != head || attempts[1].Model().Inputs().Guard == "" {
					t.Fatal("capture lost its actual current HEAD/opaque guard")
				}
			} else {
				request := project.PreviewRequest{Selection: selection}
				if mode == "fixed" {
					request.Revision = historical
				}
				result, err := service.Preview(context.Background(), request, model.Limits{})
				if err != nil || len(result.Attempts()) != 2 || len(port.previews) != 2 {
					t.Fatalf("actual historical/working preview: %v", err)
				}
				_, available := result.Prepared()
				if available != (mode == "fixed") {
					t.Fatal("wrong preview preparation completeness")
				}
				attempts := result.Attempts()
				for i, attempt := range attempts {
					if !reflect.DeepEqual(attempt.Model().Inputs(), port.previews[i]) || !reflect.DeepEqual(attempt.Model().Request(), port.previewRequests[i]) || port.previews[i].Head != head || port.previews[i].Revision != request.Revision || port.previewRequests[i].Revision != request.Revision {
						t.Fatal("preview replaced an original acquisition or its exact historical revision")
					}
				}
				first, last = attempts[0].Model().Inputs().Raw, attempts[1].Model().Inputs().Raw
				accounting, _ = attempts[1].Files().Accounting()
				status = result.Status()
			}
			assertOriginalPreparationGap(t, first, last)
			assertPreparationFileClosure(t, last)
			var manager project.SourceFile
			for _, source := range last.Instructions {
				if source.RepositoryPath.String() == "ops/manager.md" {
					manager = source
				}
			}
			if mode == "fixed" {
				for _, raw := range []project.RawInputs{first, last} {
					if _, present := raw.Census.Lookup(fixturePath(t, today)); present {
						t.Fatal("historical recapture imported today's ignored untracked file")
					}
				}
				if status != model.Passed || accounting.Status() != realization.Accounted || len(accounting.Findings()) != 0 || !bytes.Equal(manager.File.Bytes(), []byte(fileSeamInstruction("ops/manager.md"))) {
					t.Fatal("historical preparation imported today's instruction bytes")
				}
			} else {
				for _, raw := range []project.RawInputs{first, last} {
					file, present := raw.Census.Lookup(fixturePath(t, today))
					if !present || !bytes.Equal(file.Bytes(), []byte(untracked)) {
						t.Fatal("ordinary Gitignore hid actual Census bytes")
					}
				}
				findings := accounting.Findings()
				if status != model.Failed || accounting.Status() != realization.AccountingInvalid || len(findings) != 1 || findings[0].Code != "file.unexplained" || findings[0].Path.String() != today || !bytes.Equal(manager.File.Bytes(), []byte(currentInstruction)) || fileSeamAccounted(t, accounting, today).Class() != realization.UnexplainedFile {
					t.Fatalf("current unexplained file did not block structural success: %s %+v", status, findings)
				}
			}
		})
	}
}
