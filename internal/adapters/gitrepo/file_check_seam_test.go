package gitrepo

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const fileSeamIgnore = "apiVersion: markitect.ignore/v1\r\nfiles:\r\n  - path: .gitignore\r\n    reason: Git fixture exclusions, not Markitect coverage.\r\n"

func fileSeamInstruction(name string) string { return "\xef\xbb\xbfGuidance for " + name + ".\r\n" }

func fileCheckSeamFixture(t *testing.T) (*testkit.Repo, apply.InputSelection, string) {
	t.Helper()
	repo, selection, _ := registrySeamFixture(t)
	// Replace the adapter-only opaque-input fixtures with actual authored inputs.
	repo.Git("rm", "--", "config/execution.data", "config/registry.data", "config/ignore.data", "host/profile.data", "host/instruction.txt")
	selection.HostProfiles = nil
	selected := func(name string) apply.SourceLocation {
		return apply.SourceLocation{Boundary: repo.Dir, Path: fixturePath(t, name)}
	}
	selection.Execution = []apply.SourceLocation{selected(".markitect/execution.yaml")}
	selection.Ignore = []apply.SourceLocation{selected(".markitect/ignore.yaml")}
	selection.Instructions = nil
	for _, name := range []string{"ops/manager.md", "ops/reviewer.md", "ops/lending.md"} {
		repo.Write(name, fileSeamInstruction(name))
		selection.Instructions = append(selection.Instructions, selected(name))
	}
	execution := `apiVersion: markitect.execution/v1
strategy: shared-workspace-v1
budget:
  executionTime: 25m
  turnTimeout: 10m
  agentTurns: 120
  rounds: 4
profiles:
  execute: library-agent
  verify: library-agent
instructions:
  execute: [ops/manager.md, ops/reviewer.md, ops/manager.md]
  verify: [ops/reviewer.md]
packages:
  lending:
    instructions:
      execute: [ops/lending.md]
      verify: [ops/manager.md, ops/reviewer.md]
checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    requiredBy: library/callable-interface
    timeout: 2m
`
	repo.Write(".markitect/execution.yaml", execution)
	repo.Write(".markitect/ignore.yaml", fileSeamIgnore)
	repo.Write(seamRootShard, seamRootYAML+"  - path: real/library.go\r\n    form: code\r\n    realizes: [library/library]\r\n    why: Library entry fixture.\r\n")
	return repo, selection, repo.Commit("Complete authored M1 inputs for structural file checking")
}

func TestM1GitFileCheckAccountsCompleteCaptureAndPreviews(t *testing.T) {
	_, selection, head := fileCheckSeamFixture(t)
	r := adapter(t)
	service := apply.NewModelService(r, r)
	stage, err := service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || stage.Checked().Status != model.Passed || len(stage.Checked().Models) != 3 {
		t.Fatalf("actual M1/Vocabulary/Core capture failed: %v %+v", err, stage.Checked().Diagnostics)
	}
	parent := stage.PrepareRegistry()
	checked := parent.CheckFiles()
	accounting, observed := checked.Accounting()
	classes, complete := checked.Classifications()
	protections, protected := checked.Protections()
	if parent.Status() != model.Passed || checked.Status() != model.Passed || !observed || accounting.Status() != realization.Accounted || len(accounting.Findings()) != 0 || len(checked.Diagnostics()) != 0 || !complete || !protected || len(classes) == 0 || len(protections) == 0 {
		t.Fatalf("actual M1 file check failed: %s %+v %+v", checked.Status(), checked.Diagnostics(), accounting.Findings())
	}
	inputs := checked.RegistryPreparation().Model().Inputs()
	if inputs.Head != head || inputs.Guard == "" || !reflect.DeepEqual(inputs, stage.Inputs()) || !reflect.DeepEqual(checked.RegistryPreparation().Records(), parent.Records()) || accounting.TreeID() != inputs.Raw.Census.Digest() || len(accounting.Files()) != len(inputs.Raw.Census.Entries()) {
		t.Fatal("file check lost its complete Census or exact model/registry/capture parent")
	}
	for name, want := range map[string]realization.FileClass{
		"inputs/local/model.yaml":   realization.CanonicalInput,
		".markitect/execution.yaml": realization.MarkitectInput,
		".markitect/ignore.yaml":    realization.MarkitectInput,
		seamRootShard:               realization.MarkitectInput,
		seamChildShard:              realization.MarkitectInput,
		"ops/manager.md":            realization.MarkitectInput,
		"ops/reviewer.md":           realization.MarkitectInput,
		"ops/lending.md":            realization.MarkitectInput,
		".gitignore":                realization.IgnoredFile,
		"real/library.go":           realization.RegisteredArtifact,
		"realization/library.md":    realization.RegisteredArtifact,
		"realization/lending.go":    realization.RegisteredArtifact,
	} {
		file := fileSeamAccounted(t, accounting, name)
		if file.Class() != want {
			t.Fatalf("%s classified as %s, want %s", name, file.Class(), want)
		}
	}
	entry := fileSeamAccounted(t, accounting, "real/library.go").Entry()
	if entry.Mode() != snapshot.Executable || !bytes.Equal(entry.Bytes(), []byte("\xef\xbb\xbfpackage library\r\n")) {
		t.Fatal("real Git executable mode or original BOM/CRLF artifact bytes changed")
	}
	if !reflect.DeepEqual(accounting.Artifacts(model.Identity{Package: "lending", ID: "borrow"}), []snapshot.Path{fixturePath(t, "realization/lending.go")}) {
		t.Fatal("actual child registration lost its Definition link")
	}
	for _, revision := range []string{"", head} {
		preview, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision}, model.Limits{})
		if err != nil {
			t.Fatalf("actual preview %q failed: %v", revision, err)
		}
		result := preview.PrepareRegistry().CheckFiles()
		report, available := result.Accounting()
		if result.Status() != model.Passed || !available || !reflect.DeepEqual(report, accounting) || !reflect.DeepEqual(result.RegistryPreparation().Model().Inputs(), preview.Inputs()) || preview.Inputs().Revision != revision || result.RegistryPreparation().Model().Request().Revision != revision {
			t.Fatalf("actual preview lost accounting or its own revision binding: %s %+v", result.Status(), result.Diagnostics())
		}
	}
}

func fileSeamAccounted(t *testing.T, report realization.Accounting, name string) realization.File {
	t.Helper()
	for _, file := range report.Files() {
		if file.Entry().Path().String() == name {
			return file
		}
	}
	t.Fatalf("actual accounting omitted %s", name)
	return realization.File{}
}
