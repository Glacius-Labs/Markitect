package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
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

func TestApplyCompositionPreparesFreshTranscriptDirectoryBeforeRunStartup(t *testing.T) {
	enginePath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// Binding this executable performs no engine command. The concrete command
	// is inspected at the startup boundary, before any Run/container/provider.
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	checker, err := checks.NewCandidateRunner(nil, engine, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, blocked := range []bool{false, true} {
		t.Run(fmt.Sprintf("blocked=%v", blocked), func(t *testing.T) {
			cfg := runConfig{StateDirectory: filepath.Join(testkit.TempDir(t), "fresh", "state"), RepositoryIdentity: "fixture"}
			const id = "first-apply"
			parent := filepath.Join(cfg.StateDirectory, cfg.RepositoryIdentity, "transcripts")
			if blocked {
				if err := os.MkdirAll(filepath.Dir(parent), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(parent, []byte("preserve existing file"), 0600); err != nil {
					t.Fatal(err)
				}
			} else if _, err := os.Stat(cfg.StateDirectory); !os.IsNotExist(err) {
				t.Fatal("fixture must begin without state", err)
			}
			sessions := &codexappserver.Agents{}
			command, err := composeApplyCommand(cfg, id, repository, engine, sessions, checker, codexappserver.Profile{})
			if blocked {
				if err == nil || command.Executor != nil {
					t.Fatal("unusable transcript parent admitted a Run", err)
				}
				if _, err := os.Stat(filepath.Join(cfg.StateDirectory, cfg.RepositoryIdentity, "temporary")); !os.IsNotExist(err) {
					t.Fatal("workspace preparation began after evidence failure", err)
				}
				if data, err := os.ReadFile(parent); err != nil || string(data) != "preserve existing file" {
					t.Fatal("blocking input was modified", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			executor := command.Executor.(workWindow).executor
			actual, ok := executor.Sessions.(*codexappserver.Agents)
			want := filepath.Join(parent, id)
			if !ok || actual != sessions || actual.TranscriptRoot != want {
				t.Fatal("native Sessions lost the invocation transcript destination")
			}
			info, err := os.Stat(actual.TranscriptRoot)
			if err != nil || !info.IsDir() {
				t.Fatal("native Session startup has no transcript directory", err)
			}
			if runtime.GOOS != "windows" && info.Mode().Perm() != 0700 {
				t.Fatalf("transcript directory mode=%o, want 700", info.Mode().Perm())
			}
			// The adapter's exclusive create must work without test-side directory
			// setup, and must still refuse a duplicate transcript identity.
			path := filepath.Join(actual.TranscriptRoot, "manager.jsonl")
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal("first transcript open failed", err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if duplicate, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600); !os.IsExist(err) {
				if duplicate != nil {
					_ = duplicate.Close()
				}
				t.Fatal("exclusive transcript creation changed", err)
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
