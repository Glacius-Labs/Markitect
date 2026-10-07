//go:build windows

package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalControllerVerifierFixtureNormalizesShortRepoBeforeCreatingExternalState(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	canonicalRepo := canonicalControllerTestDirectory(t, repo, "repository")
	shortRepo, err := windowsShortPath(canonicalRepo)
	if err != nil {
		t.Skipf("GetShortPathNameW is unavailable: %v", err)
	}
	if samePathSpelling(shortRepo, canonicalRepo) {
		t.Skipf("filesystem did not provide an 8.3 spelling for %q", canonicalRepo)
	}
	if _, err := realDirectory(shortRepo); err == nil || !strings.Contains(err.Error(), "canonical path spelling") {
		t.Fatalf("production realDirectory accepted test input with a short-path alias: %s", shortRepo)
	}

	normalizedRepo := canonicalControllerTestDirectory(t, shortRepo, "repository")
	if !samePathSpelling(normalizedRepo, canonicalRepo) {
		t.Fatalf("test fixture repo normalization changed the directory: got=%q want=%q", normalizedRepo, canonicalRepo)
	}
	externalParent := canonicalControllerTestDirectory(t, filepath.Dir(normalizedRepo), "external parent")
	external, err := os.MkdirTemp(externalParent, "canonical-verifier-short-path-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(external) })
	if _, err := realDirectory(external); err != nil {
		t.Fatalf("external test fixture created under canonical parent is not accepted: %v", err)
	}
}
