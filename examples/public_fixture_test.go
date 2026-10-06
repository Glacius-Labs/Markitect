package examples

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// publicExampleFixture creates an immutable source snapshot from one checked-in
// public fixture directory. The source checkout itself does not need Git
// metadata, as in Markitect's materialized verify snapshots.
func publicExampleFixture(t *testing.T, fixtureDirectory string) (string, string) {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate public fixture test helper")
	}
	repositoryRoot := filepath.Dir(filepath.Dir(testFile))
	sourceDirectory := filepath.Join(repositoryRoot, filepath.FromSlash(fixtureDirectory))
	if info, err := os.Stat(sourceDirectory); err != nil || !info.IsDir() {
		t.Fatalf("public fixture directory %q is unavailable: %v", sourceDirectory, err)
	}

	root := filepath.Join(t.TempDir(), "fixture-repository")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	copyPublicFixtureTree(t, sourceDirectory, filepath.Join(root, filepath.FromSlash(fixtureDirectory)))
	if err := os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("* -text\n"), 0644); err != nil {
		t.Fatal(err)
	}
	proofGit(t, root, "init", "--template=", "--object-format=sha1", "--initial-branch=codex/public-example-fixture")
	proofGit(t, root, "config", "core.autocrlf", "false")
	revision := proofCommit(t, root, "freeze public example fixture")
	if len(revision) != 40 || strings.Trim(revision, "0123456789abcdef") != "" {
		t.Fatalf("fixture commit is not a full SHA-1 revision: %q", revision)
	}
	return root, revision
}

func copyPublicFixtureTree(t *testing.T, sourceDirectory, targetDirectory string) {
	t.Helper()
	err := filepath.WalkDir(sourceDirectory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceDirectory, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetDirectory, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("public fixture contains unsupported file type: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
