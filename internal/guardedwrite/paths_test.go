package guardedwrite

import (
	"strings"
	"testing"
)

func TestValidateRepoPathRejectsWindowsUnsafeNames(t *testing.T) {
	for _, p := range []string{"C:/outside", "a\\b", "CON.txt", "nested/NUL", "trailing.", "trailing ", "../x", "a/./b", "GIT~1/config", "docs/git~1/hooks/pre-commit"} {
		if err := ValidateRepoPath(p); err == nil {
			t.Errorf("ValidateRepoPath(%q) succeeded", p)
		}
	}
	if err := ValidateRepoPath("docs/notes~1/git~2.md"); err != nil {
		t.Errorf("ordinary 8.3-shaped name refused: %v", err)
	}
}

func TestValidatePortablePathsRejectsCaseCollisions(t *testing.T) {
	for _, paths := range [][]string{
		{"README.md", "Readme.md"},
		{"Area/one.txt", "area/two.txt"},
		{"config", "Config/nested.yaml"},
	} {
		if err := ValidatePortablePaths(paths); err == nil || !strings.Contains(err.Error(), "case-insensitive path collision") {
			t.Errorf("ValidatePortablePaths(%q) error = %v", paths, err)
		}
	}
	if err := ValidatePortablePaths([]string{"Area/one.txt", "Area/two.txt"}); err != nil {
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
