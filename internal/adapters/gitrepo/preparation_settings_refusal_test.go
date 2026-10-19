package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

func TestM1GitSettingsPreparationIgnoredCensusAndRealExternalFixedRefusal(t *testing.T) {
	repo, full, historical, host := gitSettingsFixture(t)
	originalExecution, err := os.ReadFile(filepath.Join(repo.Dir, ".markitect", "execution.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	const currentProjectInstruction = "Today's repository manager instructions.\r\n"
	repo.Write("ops/manager.md", currentProjectInstruction)
	repo.Write(".markitect/execution.yaml", string(originalExecution)+"\n# Today's execution marker.\n")
	head := repo.Commit("Advance actual project instruction and execution bytes")
	const today = "scratch/settings-session.ignored"
	const untracked = "\x00\xffActual untracked settings-session bytes.\r\n"
	repo.Write(today, untracked)
	if head == historical || repo.Git("check-ignore", "--", today) != today || repo.Git("ls-files", "--", today) != "" || repo.Git("status", "--porcelain", "--untracked-files=all") != "" {
		t.Fatal("expected clean HEAD plus a real ordinary Git-ignored untracked file")
	}
	project := apply.SourceLocation{Boundary: repo.Dir, Path: fixturePath(t, "ops/manager.md")}
	for _, mode := range []string{"capture", "working", "fixed"} {
		t.Run(mode, func(t *testing.T) {
			port := &recordingPreparationGit{repository: adapter(t)}
			metadata := &gitSettingsMetadataPorts{t: t, policy: []apply.SourceLocation{project, host, host, project}}
			service := apply.NewPreparationService(apply.NewModelService(port, port))
			selection := full
			if mode == "fixed" {
				selection.Instructions = append(append([]apply.SourceLocation(nil), full.Instructions...), host)
				request := apply.PreviewRequest{Selection: selection, Revision: historical}
				result, err := service.PreviewSettings(context.Background(), request, model.Limits{}, metadata, metadata)
				if !errors.Is(err, ErrUnsafe) || result.Status() != model.NotCompleted || len(result.Attempts()) != 1 || len(port.previews) != 1 || len(port.captures) != 0 || !reflect.DeepEqual(result.Request(), request) {
					t.Fatalf("real external fixed Settings preview accepted or lost request: %s %v", result.Status(), err)
				}
				attempt := result.Attempts()[0]
				inputs := attempt.Structural().Model().Inputs()
				if _, ok := result.Prepared(); ok {
					t.Fatal("unsafe fixed preview exposed complete Settings preparation")
				}
				if !reflect.DeepEqual(attempt.Structural().Model().Request(), port.previewRequests[0]) || !reflect.DeepEqual(inputs, port.previews[0]) || inputs.Head != head || inputs.Revision != historical || attempt.Error() != err || attempt.Structural().Error() != err || attempt.Structural().Model().Compilation().Status != model.NotCompleted {
					t.Fatal("real Git refusal discarded its partial observation or fixed binding")
				}
				if len(inputs.Raw.Local.Files.Entries()) == 0 || len(inputs.Raw.Dependencies) != 2 || len(inputs.Raw.Execution) != 1 || !bytes.Equal(inputs.Raw.Execution[0].File.Bytes(), originalExecution) || len(inputs.Raw.HostProfiles) != 1 || inputs.Raw.HostProfiles[0].Location != full.HostProfiles[0] || len(inputs.Raw.HostProfiles[0].File.Bytes()) != 0 || len(inputs.Raw.Instructions) != 0 {
					t.Fatal("refused historical preview imported today's execution, external Host or instruction bytes")
				}
				if _, found := inputs.Raw.Census.Lookup(fixturePath(t, today)); found {
					t.Fatal("refused historical observation gained today's ignored file")
				}
				if len(metadata.agents) != 0 || len(metadata.checks) != 0 || len(attempt.Settings().ProfileObservations()) != 0 || attempt.Settings().Status() != model.NotCompleted {
					t.Fatal("metadata ran or invented profile reports after source refusal")
				}
				return
			}
			selection.Execution, selection.Registry, selection.Ignore, selection.Instructions = nil, nil, nil, nil
			var first, last apply.RawInputs
			var accounting realization.Accounting
			var status model.Status
			if mode == "capture" {
				request := apply.CaptureRequest{Selection: selection}
				result, err := service.CaptureSettings(context.Background(), request, model.Limits{}, metadata, metadata)
				if err != nil || len(result.Attempts()) != 2 || len(port.captures) != 2 || !reflect.DeepEqual(result.Request(), request) {
					t.Fatalf("actual current Settings capture: %s %v", result.Status(), err)
				}
				for i, attempt := range result.Attempts() {
					assertGitCapturedSettingsAttempt(t, attempt, port.captureRequests[i], port.captures[i])
				}
				if _, ok := result.Prepared(); ok {
					t.Fatal("ordinary ignored file became complete Settings preparation")
				}
				attempts := result.Attempts()
				first, last = attempts[0].Structural().Model().Inputs().Raw, attempts[1].Structural().Model().Inputs().Raw
				accounting, _ = attempts[1].Structural().Files().Accounting()
				status = result.Status()
				if attempts[1].Structural().Model().Inputs().Head != head || attempts[1].Structural().Model().Inputs().Guard == "" {
					t.Fatal("current capture lost actual HEAD/opaque guard")
				}
				if _, ok := attempts[1].Settings().Settings(); ok {
					t.Fatal("failed accounting exposed complete static settings")
				}
			} else {
				request := apply.PreviewRequest{Selection: selection}
				result, err := service.PreviewSettings(context.Background(), request, model.Limits{}, metadata, metadata)
				if err != nil || len(result.Attempts()) != 2 || len(port.previews) != 2 || !reflect.DeepEqual(result.Request(), request) {
					t.Fatalf("actual current working Settings preview: %s %v", result.Status(), err)
				}
				for i, attempt := range result.Attempts() {
					if !reflect.DeepEqual(attempt.Structural().Model().Request(), port.previewRequests[i]) || !reflect.DeepEqual(attempt.Structural().Model().Inputs(), port.previews[i]) || !reflect.DeepEqual(attempt.Settings().FileCheck(), attempt.Structural().Files()) || !reflect.DeepEqual(attempt.Settings().Defaults(), attempt.Defaults()) || port.previews[i].Revision != "" || port.previews[i].Head != head {
						t.Fatal("current working preview changed original parents/revision")
					}
				}
				if _, ok := result.Prepared(); ok {
					t.Fatal("ordinary ignored file became a complete working preview")
				}
				attempts := result.Attempts()
				first, last = attempts[0].Structural().Model().Inputs().Raw, attempts[1].Structural().Model().Inputs().Raw
				accounting, _ = attempts[1].Structural().Files().Accounting()
				status = result.Status()
				if _, ok := attempts[1].Settings().Settings(); ok {
					t.Fatal("failed working accounting exposed complete static settings")
				}
			}
			assertOriginalPreparationGap(t, first, last)
			for _, raw := range []apply.RawInputs{first, last} {
				entry, present := raw.Census.Lookup(fixturePath(t, today))
				if !present || !bytes.Equal(entry.Bytes(), []byte(untracked)) {
					t.Fatal("ordinary Gitignore concealed actual Census bytes")
				}
			}
			// Capture names exactly the Git-ignored untracked files; previews name none.
			var untracked []string
			for _, p := range last.Untracked {
				untracked = append(untracked, p.String())
			}
			var wantUntracked []string
			if mode == "capture" {
				wantUntracked = []string{today}
			}
			if !reflect.DeepEqual(untracked, wantUntracked) {
				t.Fatalf("untracked %q, want %q", untracked, wantUntracked)
			}
			findings := accounting.Findings()
			if status != model.Failed || accounting.Status() != realization.AccountingInvalid || len(findings) != 1 || findings[0].Code != "file.unexplained" || findings[0].Path.String() != today || fileSeamAccounted(t, accounting, today).Class() != realization.UnexplainedFile || len(metadata.agents) != 2 || len(metadata.checks) != 2 {
				t.Fatalf("actual ordinary ignored file did not prevent Passed: %s %+v", status, findings)
			}
		})
	}
}
