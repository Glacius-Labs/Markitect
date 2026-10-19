package gitrepo

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

func fileSeamCaptured(t *testing.T, selection apply.InputSelection) apply.CapturedRegistryPreparation {
	t.Helper()
	r := adapter(t)
	stage, err := apply.NewModelService(r, r).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || stage.Compilation().Status != model.Passed {
		t.Fatalf("real Git acquisition/model failed: %v %+v", err, stage.Compilation().Diagnostics)
	}
	prepared := stage.PrepareRegistry()
	if prepared.Status() != model.Passed {
		t.Fatalf("real acquired registry failed: %+v", prepared.Diagnostics())
	}
	return prepared
}

func TestM1GitFileCheckRetainsAllAuthoredInstructionOccurrences(t *testing.T) {
	_, selection, _ := fileCheckSeamFixture(t)
	parent := fileSeamCaptured(t, selection)
	checked := parent.CheckFiles()
	inputs := checked.RegistryPreparation().Model().Inputs()
	document, available := checked.RegistryPreparation().Model().DecodeExecution().Document()
	if checked.Status() != model.Passed || !available || len(inputs.Raw.Instructions) != 3 || document.Packages == nil || len(document.Packages.Value) != 1 || document.Packages.Value[0].Package != "lending" {
		t.Fatalf("complete real instruction acquisition failed: %+v", checked.Diagnostics())
	}
	for _, list := range []struct {
		actual *organization.Setting[[]string]
		field  string
		want   []string
	}{
		{document.Defaults.Execute.Instructions, "/instructions/execute", []string{"ops/manager.md", "ops/reviewer.md", "ops/manager.md"}},
		{document.Defaults.Verify.Instructions, "/instructions/verify", []string{"ops/reviewer.md"}},
		{document.Packages.Value[0].Execute.Instructions, "/packages/lending/instructions/execute", []string{"ops/lending.md"}},
		{document.Packages.Value[0].Verify.Instructions, "/packages/lending/instructions/verify", []string{"ops/manager.md", "ops/reviewer.md"}},
	} {
		if list.actual == nil || !reflect.DeepEqual(list.actual.Value, list.want) || list.actual.Source.Field != list.field || list.actual.Source.File != ".markitect/execution.yaml" || list.actual.Source.ContentID != inputs.Raw.Execution[0].File.ContentID() {
			t.Fatalf("authored instruction order, repetition or source changed at %s: %+v", list.field, list.actual)
		}
	}
	for i, name := range []string{"ops/manager.md", "ops/reviewer.md", "ops/lending.md"} {
		source := inputs.Raw.Instructions[i]
		census, present := inputs.Raw.Census.Lookup(fixturePath(t, name))
		if !present || source.Location != selection.Instructions[i] || source.RepositoryPath != census.Path() || source.File.Mode() != census.Mode() || !bytes.Equal(source.File.Bytes(), []byte(fileSeamInstruction(name))) || !bytes.Equal(source.File.Bytes(), census.Bytes()) {
			t.Fatalf("actual instruction coordinates/mode/original bytes changed: %s", name)
		}
	}
	if !reflect.DeepEqual(inputs, parent.Model().Inputs()) {
		t.Fatal("instruction checking changed the captured parent or guard")
	}
}

func TestM1GitFileCheckObservedButUncapturedInputsRemainIncomplete(t *testing.T) {
	for _, missing := range []string{"ignore", "instruction"} {
		t.Run(missing, func(t *testing.T) {
			_, selection, head := fileCheckSeamFixture(t)
			if missing == "ignore" {
				selection.Ignore = nil
			} else {
				selection.Instructions = selection.Instructions[1:]
			}
			parent := fileSeamCaptured(t, selection)
			checked := parent.CheckFiles()
			inputs := checked.RegistryPreparation().Model().Inputs()
			accounting, observed := checked.Accounting()
			if checked.Status() != model.NotCompleted || !observed || inputs.Head != head || inputs.Guard == "" || !reflect.DeepEqual(inputs, parent.Model().Inputs()) {
				t.Fatalf("uncaptured input became complete or lost its actual parent: %s %+v", checked.Status(), checked.Diagnostics())
			}
			if classes, ok := checked.Classifications(); ok || classes != nil {
				t.Fatal("incomplete classifications became usable")
			}
			if protections, ok := checked.Protections(); ok || protections != nil {
				t.Fatal("incomplete protections became usable")
			}
			diagnostics := checked.Diagnostics()
			if missing == "ignore" {
				policy, found := inputs.Raw.Census.Lookup(fixturePath(t, ".markitect/ignore.yaml"))
				if !found || !bytes.Equal(policy.Bytes(), []byte(fileSeamIgnore)) || len(inputs.Raw.Ignore) != 0 || len(diagnostics) != 1 || diagnostics[0].Code != "files.ignore_uncaptured" || diagnostics[0].Source.File != ".markitect/ignore.yaml" || diagnostics[0].Source.ContentID != policy.ContentID() || accounting.Status() != realization.AccountingInvalid {
					t.Fatalf("actual uncaptured ignore observation lost: %+v %+v", diagnostics, accounting.Findings())
				}
				if len(accounting.Findings()) != 1 || accounting.Findings()[0].Code != "file.unexplained" || accounting.Findings()[0].Path.String() != ".gitignore" {
					t.Fatalf("missing policy concealed its actual accounting gap: %+v", accounting.Findings())
				}
			} else {
				instruction, found := inputs.Raw.Census.Lookup(fixturePath(t, "ops/manager.md"))
				if !found || !bytes.Equal(instruction.Bytes(), []byte(fileSeamInstruction("ops/manager.md"))) || len(inputs.Raw.Instructions) != 2 || accounting.Status() != realization.Accounted || len(diagnostics) != 3 {
					t.Fatalf("actual uncaptured instruction or repeated references lost: %+v", diagnostics)
				}
				for i, field := range []string{"/instructions/execute/0", "/instructions/execute/2", "/packages/lending/instructions/verify/0"} {
					if diagnostics[i].Code != "files.instruction_uncaptured" || diagnostics[i].Source.File != ".markitect/execution.yaml" || diagnostics[i].Source.Field != field || diagnostics[i].Source.ContentID != inputs.Raw.Execution[0].File.ContentID() {
						t.Fatalf("uncaptured instruction occurrence lost its authored source: %+v", diagnostics[i])
					}
				}
			}
		})
	}
}

func TestM1GitFileCheckIgnoredUntrackedFileIsVisibleAndHistoricalPreviewIsFixed(t *testing.T) {
	repo, selection, historical := fileCheckSeamFixture(t)
	repo.Write("ops/manager.md", "Today's committed instructions.\r\n")
	head := repo.Commit("Advance current HEAD with changed instructions")
	const today = "editor/session.ignored"
	const original = "\x00\xffToday-only untracked editor bytes.\r\n"
	repo.Write(today, original)
	if repo.Git("check-ignore", "--", today) != today || repo.Git("ls-files", "--", today) != "" {
		t.Fatal("fixture is not an actual Git-ignored untracked file")
	}
	parent := fileSeamCaptured(t, selection)
	checked := parent.CheckFiles()
	accounting, observed := checked.Accounting()
	inputs := checked.RegistryPreparation().Model().Inputs()
	file := fileSeamAccounted(t, accounting, today)
	if checked.Status() != model.Failed || !observed || accounting.Status() != realization.AccountingInvalid || len(accounting.Findings()) != 1 || accounting.Findings()[0].Code != "file.unexplained" || accounting.Findings()[0].Path.String() != today || file.Class() != realization.UnexplainedFile || !bytes.Equal(file.Entry().Bytes(), []byte(original)) || inputs.Head != head || inputs.Guard == "" {
		t.Fatalf("Gitignore hid ordinary acquisition/accounting evidence: %s %+v %+v", checked.Status(), checked.Diagnostics(), accounting.Findings())
	}
	if !bytes.Equal(inputs.Raw.Instructions[0].File.Bytes(), []byte("Today's committed instructions.\r\n")) {
		t.Fatal("current capture did not use current committed instructions")
	}
	r := adapter(t)
	preview, err := apply.NewModelService(r, r).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: historical}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	fixed := preview.PrepareRegistry().CheckFiles()
	fixedAccounting, available := fixed.Accounting()
	fixedInputs := fixed.RegistryPreparation().Model().Inputs()
	_, hasToday := fixedInputs.Raw.Census.Lookup(fixturePath(t, today))
	if fixed.Status() != model.Passed || !available || fixedAccounting.Status() != realization.Accounted || hasToday || historical == head || fixedInputs.Head != head || fixedInputs.Revision != historical || fixed.RegistryPreparation().Model().Request().Revision != historical || !reflect.DeepEqual(fixedInputs, preview.Inputs()) || !bytes.Equal(fixedInputs.Raw.Instructions[0].File.Bytes(), []byte(fileSeamInstruction("ops/manager.md"))) {
		t.Fatalf("historical preview imported today's files/instructions or lost revision: %s %+v", fixed.Status(), fixed.Diagnostics())
	}
}
