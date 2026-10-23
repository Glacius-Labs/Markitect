package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestApplyClosesRuntimeInstructionsThroughFreshCompleteCapture(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(fmt.Sprintf("local=%v", local), func(t *testing.T) {
			r := productFixture(t)
			r.Write(".markitect/execution.yaml", "apiVersion: markitect.execution/v2\nchecks: []\n")
			inputRoot := testkit.TempDir(t)
			instructionRoot := inputRoot
			if local {
				instructionRoot = r.Dir
				r.Write("runtime-rules.txt", "Exact runtime instruction bytes.\n")
			} else if err := os.WriteFile(filepath.Join(inputRoot, "runtime-rules.txt"), []byte("Exact runtime instruction bytes.\n"), 0600); err != nil {
				t.Fatal(err)
			}
			r.Commit("Capture model before runtime instruction closure")
			engine, _ := os.Executable()
			text := fmt.Sprintf("apiVersion: markitect.runtime/v1\nagentProfiles:\n  account:\n    codexAppServer:\n      profileVersion: 2\n      engineExecutable: %q\n      imageID: %s\n      agentdVersion: development\n      recordedCodexSchemaDigest: %s\n      instructions:\n        - {boundary: %q, path: runtime-rules.txt}\ncheckProfiles: {}\n", engine, bootstrapImageID, bootstrapSchema, instructionRoot)
			if err := os.WriteFile(filepath.Join(inputRoot, "runtime.yaml"), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			manifest, _ := snapshot.NewPath(".markitect/model.yaml")
			runtimePath, _ := snapshot.NewPath("runtime.yaml")
			selection := project.InputSelection{Repository: r.Dir, Local: project.ModelSelection{Boundary: ".", ManifestPath: manifest}, RuntimeProfiles: []project.SourceLocation{{Boundary: inputRoot, Path: runtimePath}}}
			repository, err := gitrepo.New(gitrepo.Limits{})
			if err != nil {
				t.Fatal(err)
			}
			preparation := project.NewPreparationService(project.NewModelService(repository, repository))
			reader := &bootstrapImageReader{}
			agents := codexappserver.NewMetadataPreparer(func(codexappserver.EngineConfig) (codexappserver.ImageReader, error) { return reader, nil })
			prepared, metadata, status, err := captureApplyInputs(context.Background(), preparation, repository, selection, agents, nil)
			if err != nil || status != model.Passed || metadata.Status != model.Passed || !prepared.Valid() || len(metadata.RequiredInstructions) != 0 {
				t.Fatal(status, metadata.Status, err)
			}
			if len(reader.calls) != 6 {
				t.Fatal("metadata did not observe both fresh captures", reader.calls)
			}
			found := false
			for _, file := range prepared.Attempt().Model().Inputs().Raw.Instructions {
				if file.Location.Boundary == instructionRoot && file.Location.Path.String() == "runtime-rules.txt" && string(file.File.Bytes()) == "Exact runtime instruction bytes.\n" {
					found = true
				}
			}
			if !found {
				t.Fatal("new complete capture omitted exact instruction bytes")
			}
		})
	}
}

func TestStateDestinationRejectsProductContainmentAndAcceptsRealOutsideAnchor(t *testing.T) {
	repository := testkit.TempDir(t)
	if err := safeStateDestination(repository, filepath.Join(repository, "new", "state")); err == nil {
		t.Fatal("state inside candidate accepted")
	}
	outside := filepath.Join(testkit.TempDir(t), "new", "state")
	if err := safeStateDestination(repository, outside); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatal("preflight created state", err)
	}
}
