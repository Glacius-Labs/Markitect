package release

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
)

func TestTransitionalFoundationsMatchLiveAndFixedSourceArchives(t *testing.T) {
	root := transitionalSourceFixture(t)
	live, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	liveArchive, err := makeArchive(live)
	if err != nil {
		t.Fatal(err)
	}

	fixedFiles, err := snapshotModuleFiles(fixedSourceSnapshot(t, root))
	if err != nil {
		t.Fatal(err)
	}
	fixed, err := collectModuleSnapshot(fixedFiles)
	if err != nil {
		t.Fatal(err)
	}
	fixedArchive, err := makeArchive(fixed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(liveArchive, fixedArchive) {
		t.Fatal("live and fixed-source packaging selected different files or bytes")
	}

	contents := archiveContents(t, liveArchive)
	for _, name := range []string{
		"src/cmd/markitect/main.go",
		"src/internal/engine/engine.go",
		"internal/model/model.go",
		"internal/snapshot/tree.go",
		"internal/organization/organization.go",
		"internal/realization/realization.go",
		"internal/model/testdata/fixture.go",
	} {
		if _, ok := contents[name]; !ok {
			t.Errorf("archive omitted %s", name)
		}
	}
	if _, ok := contents["src/internal/model/model.go"]; ok {
		t.Fatal("root foundation was rewritten under the legacy source prefix")
	}

	before := sha256.Sum256(liveArchive)
	writeLayoutFile(t, root, "internal/organization/organization.go", "package organization\n// changed\n")
	changed, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	changedArchive, err := makeArchive(changed)
	if err != nil {
		t.Fatal(err)
	}
	after := sha256.Sum256(changedArchive)
	if before == after {
		t.Fatal("root foundation source change did not change the source archive digest")
	}
}

func TestTransitionalSourceLayoutRejectsMixedIncompleteAndUnknownTrees(t *testing.T) {
	tests := []struct {
		name   string
		change func(string)
	}{
		{"mixed root command", func(root string) {
			writeLayoutFile(t, root, "cmd/markitect/main.go", "package main\n")
		}},
		{"unknown root internal", func(root string) {
			writeLayoutFile(t, root, "internal/adapters/unexpected.go", "package adapters\n")
		}},
		{"foundation-only source", func(root string) {
			if err := os.RemoveAll(filepath.Join(root, "src")); err != nil {
				t.Fatal(err)
			}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := transitionalSourceFixture(t)
			test.change(root)
			if _, err := sourceLayoutPrefix(root); err == nil {
				t.Fatal("live source layout was accepted")
			}
			fixedFiles, err := snapshotModuleFiles(fixedSourceSnapshot(t, root))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := collectModuleSnapshot(fixedFiles); err == nil {
				t.Fatal("fixed source layout was accepted")
			}
		})
	}
}

func TestTransitionalSourceLayoutRequiresBothActiveTrees(t *testing.T) {
	root := transitionalSourceFixture(t)
	if err := os.RemoveAll(filepath.Join(root, "src", "internal")); err != nil {
		t.Fatal(err)
	}
	if _, err := sourceLayoutPrefix(root); err == nil {
		t.Fatal("live source layout without src/internal was accepted")
	}
	fixedFiles, err := snapshotModuleFiles(fixedSourceSnapshot(t, root))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := collectModuleSnapshot(fixedFiles); err == nil {
		t.Fatal("fixed source layout without src/internal was accepted")
	}
}

func TestTransitionalSourceLayoutRequiresEntrypointAndInternalGoFiles(t *testing.T) {
	for _, test := range []struct {
		name   string
		remove string
	}{
		{"command entrypoint", "src/cmd/markitect/main.go"},
		{"internal Go source", "src/internal/engine/engine.go"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := transitionalSourceFixture(t)
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(test.remove))); err != nil {
				t.Fatal(err)
			}
			if _, err := collect(root); err == nil {
				t.Fatal("live source without the required file was accepted")
			}
			fixedFiles, err := snapshotModuleFiles(fixedSourceSnapshot(t, root))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := collectModuleSnapshot(fixedFiles); err == nil {
				t.Fatal("fixed source without the required file was accepted")
			}
		})
	}
}

func TestFlatHistoricalSourceLayoutRemainsReadable(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":                "module " + sourceRepository + "\n\ngo 1.27.1\n",
		"go.sum":                "example checksum\n",
		"README.md":             "Markitect\n",
		"LICENSE":               "Apache License\nVersion 2.0\n",
		"cmd/markitect/main.go": "package main\n",
		"internal/core/core.go": "package core\n",
	} {
		writeLayoutFile(t, root, name, content)
	}
	live, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}
	liveArchive, err := makeArchive(live)
	if err != nil {
		t.Fatal(err)
	}
	fixedFiles, err := snapshotModuleFiles(fixedSourceSnapshot(t, root))
	if err != nil {
		t.Fatal(err)
	}
	fixed, err := collectModuleSnapshot(fixedFiles)
	if err != nil {
		t.Fatal(err)
	}
	fixedArchive, err := makeArchive(fixed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(liveArchive, fixedArchive) {
		t.Fatal("historical flat live and fixed-source archives differ")
	}

	prefix, err := sourceLayoutPrefix(root)
	if err != nil || prefix != "" {
		t.Fatalf("flat source layout = %q, %v; want historical root layout", prefix, err)
	}
}

func transitionalSourceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{
		"go.mod":                                        "module " + sourceRepository + "\n\ngo 1.27.1\n",
		"go.sum":                                        "example checksum\n",
		"README.md":                                     "Markitect\n",
		"LICENSE":                                       "Apache License\nVersion 2.0\n",
		"src/cmd/markitect/main.go":                     "package main\n",
		"src/internal/engine/engine.go":                 "package engine\n",
		"internal/model/model.go":                       "package model\n",
		"internal/snapshot/tree.go":                     "package snapshot\n",
		"internal/organization/organization.go":         "package organization\n",
		"internal/realization/realization.go":           "package realization\n",
		"internal/model/testdata/fixture.go":            "package fixture\n",
		"internal/model/other_platform_windows_test.go": "package model_test\n",
	} {
		writeLayoutFile(t, root, name, body)
	}
	return root
}

func fixedSourceSnapshot(t *testing.T, root string) *snapshot.Snapshot {
	t.Helper()
	files := map[string][]byte{}
	modes := map[string]string{}
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		files[relative] = data
		modes[relative] = snapshot.RegularMode
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return &snapshot.Snapshot{Files: files, Modes: modes}
}

func writeLayoutFile(t *testing.T, root, relative, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTransitionalFoundationPathIsExact(t *testing.T) {
	for _, name := range []string{"internal/model/model.go", "internal/snapshot/tree.go", "internal/organization/x.go", "internal/realization/x.go"} {
		if !isTransitionalFoundationPath(name) {
			t.Errorf("foundation path %q was rejected", name)
		}
	}
	for _, name := range []string{"internal/modeling/x.go", "internal/adapters/x.go", "src/internal/model/model.go"} {
		if isTransitionalFoundationPath(name) {
			t.Errorf("unsupported foundation path %q was accepted", name)
		}
	}
	if err := validateTransitionalFoundationFiles([]string{"internal/adapters/x.go"}); err == nil || !strings.Contains(err.Error(), "unsupported root source path") {
		t.Fatalf("unsupported root source was not rejected: %v", err)
	}
}
