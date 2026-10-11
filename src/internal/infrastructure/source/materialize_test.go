package source

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
)

func TestUnsafePathsRejectedBeforeMaterialize(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "escape.txt")
	s := &snapshot.Snapshot{
		Files: map[string][]byte{"../escape.txt": []byte("no")},
		Modes: map[string]string{"../escape.txt": snapshot.RegularMode},
	}
	err := Materialize(s, filepath.Join(t.TempDir(), "destination"))
	if err == nil || !strings.Contains(err.Error(), "unsafe repository path") {
		t.Fatalf("Materialize error = %v, want unsafe-path rejection", err)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("escaped file unexpectedly exists (stat error: %v)", err)
	}
}

func TestMaterializeDiagnosticsDoNotDependOnMapOrder(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "destination")
	collision := &snapshot.Snapshot{
		Files: map[string][]byte{"Docs/a.md": []byte("a"), "docs/b.md": []byte("b")},
		Modes: map[string]string{"Docs/a.md": snapshot.RegularMode, "docs/b.md": snapshot.RegularMode},
	}
	requireSameError(t, func() error { return Materialize(collision, destination) }, `between "Docs" and "docs"`)
	unsafe := &snapshot.Snapshot{
		Files: map[string][]byte{"a/../x": nil, "b:c": nil, "d\\e": nil},
		Modes: map[string]string{"a/../x": snapshot.RegularMode, "b:c": snapshot.RegularMode, "d\\e": snapshot.RegularMode},
	}
	requireSameError(t, func() error { return Materialize(unsafe, destination) }, `"a/../x"`)
}

// Snapshot paths are checked lexically, and Windows resolves an 8.3 short
// name such as MARKIT~1 to a directory written earlier, so Materialize once
// wrote MARKIT~1/runtime.yaml into .markitect.
func TestMaterializeRefusesWindowsAliasesOfExistingEntries(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("short names and case aliases are resolved by Windows")
	}
	regular := func(files map[string][]byte) *snapshot.Snapshot {
		s := &snapshot.Snapshot{Files: files, Modes: map[string]string{}}
		for name := range files {
			s.Modes[name] = snapshot.RegularMode
		}
		return s
	}
	// Every NTFS volume resolves case variants, so only the 8.3 case skips.
	// Case variants never reach the write: the lexical collision check
	// refuses them first.
	t.Run("case variant", func(t *testing.T) {
		destination := filepath.Join(t.TempDir(), "destination")
		err := Materialize(regular(map[string][]byte{"Docs/guide.md": []byte("guide\n"), "docs/new.md": []byte("alias\n")}), destination)
		if err == nil {
			t.Error("Materialize accepted case variant docs/new.md")
		}
		if _, statErr := os.Lstat(filepath.Join(destination, "Docs", "new.md")); !os.IsNotExist(statErr) {
			t.Errorf("case variant reached Docs/new.md: %v", statErr)
		}
	})
	t.Run("8.3 short name", func(t *testing.T) {
		probe := t.TempDir()
		if err := os.Mkdir(filepath.Join(probe, ".markitect"), 0o755); err != nil {
			t.Fatal(err)
		}
		longInfo, longErr := os.Stat(filepath.Join(probe, ".markitect"))
		aliasInfo, aliasErr := os.Stat(filepath.Join(probe, "MARKIT~1"))
		if longErr != nil || aliasErr != nil || !os.SameFile(longInfo, aliasInfo) {
			t.Skip("volume generates no 8.3 short names")
		}
		destination := filepath.Join(t.TempDir(), "destination")
		err := Materialize(regular(map[string][]byte{".markitect/project.yaml": []byte("project\n"), "MARKIT~1/runtime.yaml": []byte("alias\n")}), destination)
		if err == nil || !strings.Contains(err.Error(), "stored under another name") {
			t.Errorf("Materialize did not refuse alias MARKIT~1/runtime.yaml: %v", err)
		}
		if _, statErr := os.Lstat(filepath.Join(destination, ".markitect", "runtime.yaml")); !os.IsNotExist(statErr) {
			t.Errorf("alias write reached .markitect/runtime.yaml: %v", statErr)
		}
	})
}

func TestValidateRepoPathRejectsWindowsUnsafeNames(t *testing.T) {
	for _, p := range []string{"C:/outside", "a\\b", "CON.txt", "nested/NUL", "trailing.", "trailing ", "../x", "a/./b", "GIT~1/config", "docs/git~1/hooks/pre-commit"} {
		if err := validateRepoPath(p); err == nil {
			t.Errorf("validateRepoPath(%q) succeeded", p)
		}
	}
	if err := validateRepoPath("docs/notes~1/git~2.md"); err != nil {
		t.Errorf("ordinary 8.3-shaped name refused: %v", err)
	}
}

func TestValidatePortablePathsRejectsCaseCollisions(t *testing.T) {
	for _, paths := range [][]string{
		{"README.md", "Readme.md"},
		{"Area/one.txt", "area/two.txt"},
		{"config", "Config/nested.yaml"},
	} {
		if err := validatePortablePaths(paths); err == nil || !strings.Contains(err.Error(), "case-insensitive path collision") {
			t.Errorf("validatePortablePaths(%q) error = %v", paths, err)
		}
	}
	if err := validatePortablePaths([]string{"Area/one.txt", "Area/two.txt"}); err != nil {
		t.Fatalf("valid shared directory rejected: %v", err)
	}
}

func TestValidateIncludedPathsAppliesSnapshotBoundaries(t *testing.T) {
	for _, p := range []string{".artifacts/report/README.md", ".ARTIFACTS/report/README.md", "vendor/pkg/README.md", "app/node_modules/README.md"} {
		if err := ValidateIncludedPaths([]string{p}); err == nil || !strings.Contains(err.Error(), "excluded") {
			t.Errorf("ValidateIncludedPaths(%q) error = %v, want exclusion", p, err)
		}
	}
	for _, paths := range [][]string{
		{"docs/area/README.md", "docs/Area/other.md"},
		{"markitect.yaml", "Markitect.yaml"},
	} {
		if err := ValidateIncludedPaths(paths); err == nil || !strings.Contains(err.Error(), "case-insensitive path collision") {
			t.Errorf("ValidateIncludedPaths(%q) error = %v, want portable collision", paths, err)
		}
	}
	if err := ValidateIncludedPaths([]string{"docs/area/README.md", "markitect.yaml"}); err != nil {
		t.Fatalf("valid prospective paths rejected: %v", err)
	}
}
