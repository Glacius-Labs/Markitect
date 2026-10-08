package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
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

func TestValidateRepoPathRejectsWindowsUnsafeNames(t *testing.T) {
	for _, p := range []string{"C:/outside", "a\\b", "CON.txt", "nested/NUL", "trailing.", "trailing ", "../x", "a/./b"} {
		if err := validateRepoPath(p); err == nil {
			t.Errorf("validateRepoPath(%q) succeeded", p)
		}
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
