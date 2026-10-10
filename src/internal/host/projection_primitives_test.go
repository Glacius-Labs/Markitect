package host

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func TestProjectionWriterUsesCapturedInputsWithoutLegacyProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "intent.txt"), []byte("canonical bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	captured, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	written, err := WriteProjectionContents(root, captured, map[string][]byte{"target/result.txt": []byte("representation")})
	if err != nil || len(written) != 1 || written[0] != "target/result.txt" {
		t.Fatalf("write: %v %v", written, err)
	}
	intent, err := os.ReadFile(filepath.Join(root, "intent.txt"))
	if err != nil || string(intent) != "canonical bytes" {
		t.Fatalf("intent changed: %s %v", intent, err)
	}
	if _, err := WriteProjectionContents(root, captured, map[string][]byte{"target/result.txt": []byte("stale")}); err == nil || !strings.Contains(err.Error(), "source changed") {
		t.Fatalf("stale capture accepted: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "target/result.txt"))
	if err != nil || string(data) != "representation" {
		t.Fatalf("stale write mutated target: %s %v", data, err)
	}
}

func TestProjectionPrimitivesPreserveTrustBoundaries(t *testing.T) {
	root := t.TempDir()
	for _, captured := range []*snapshot.Snapshot{nil, {Files: map[string][]byte{}}} {
		if _, err := WriteProjectionContents(root, captured, map[string][]byte{"x": []byte("y")}); err == nil {
			t.Fatal("writer accepted immutable or absent capture")
		}
	}
	captured, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteProjectionContents(root, captured, map[string][]byte{"../escape": []byte("x")}); err == nil {
		t.Fatal("unsafe path accepted")
	}
	for _, captured := range []*snapshot.Snapshot{nil, {Provisional: true}} {
		if _, err := verifySnapshotScoped(captured, []authoring.Check{{Name: "check", Run: []string{"go", "version"}}}, verifyDefaultTime, true); err == nil {
			t.Fatal("verification accepted absent or provisional snapshot")
		}
	}
	if _, err := verifySnapshotScoped(&snapshot.Snapshot{}, nil, verifyDefaultTime, true); err == nil {
		t.Fatal("no-check verification accepted")
	}
}

func TestProjectionWriterCarriesExecutableArtifactModeOnlyAfterReadback(t *testing.T) {
	root := t.TempDir()
	captured, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("#!/bin/sh\nexit 0\n")
	written, err := WriteProjectionArtifacts(root, captured, map[string][]byte{"hooks/pre-commit": content}, map[string]string{"hooks/pre-commit": snapshot.ExecutableMode})
	if runtime.GOOS == "windows" {
		if err == nil || !strings.Contains(err.Error(), "unsupported on Windows") || len(written) != 0 {
			t.Fatalf("Windows accepted unobservable executable mode: paths=%v err=%v", written, err)
		}
		if _, statErr := os.Stat(filepath.Join(root, "hooks", "pre-commit")); !os.IsNotExist(statErr) {
			t.Fatal("unsupported mode refusal left an output")
		}
		return
	}
	if err != nil || len(written) != 1 {
		t.Fatalf("write executable artifact: paths=%v err=%v", written, err)
	}
	observed, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(observed.Files["hooks/pre-commit"], content) || observed.Modes["hooks/pre-commit"] != snapshot.ExecutableMode {
		t.Fatalf("actual artifact mode was not observed: files=%q modes=%v", observed.Files["hooks/pre-commit"], observed.Modes)
	}
}
