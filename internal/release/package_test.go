package release

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestPackageIsDeterministicAndContainsOnlyReleaseSources(t *testing.T) {
	root := fixtureRoot(t)
	first, lock1, err := Package(root, "v1.2.3-rc.4+build.7")
	if err != nil {
		t.Fatal(err)
	}
	second, lock2, err := Package(root, "v1.2.3-rc.4+build.7")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) || !bytes.Equal(lock1, lock2) {
		t.Fatal("packaging the same source twice changed output")
	}
	entries := archiveEntries(t, first)
	want := []string{"LICENSE", "README.md", "cmd/markitect/main.go", "go.mod", "go.sum", "internal/authoring/project.yaml", "internal/core/model.go", "internal/format/schema.go", "internal/release/package.go", "internal/release/package_test.go", "schema/manifest.yaml"}
	if strings.Join(entries, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected package contents:\n%v\nwant:\n%v", entries, want)
	}
	for _, name := range entries {
		if err := safeArchivePath(name); err != nil {
			t.Fatalf("unsafe archive path %q: %v", name, err)
		}
	}
	verifyNormalZipMetadata(t, first)
	hash := sha256.Sum256(first)
	wantLock := "version: \"v1.2.3-rc.4+build.7\"\nsource: \".markitect/tool/source.zip\"\nsha256: \"" + hex.EncodeToString(hash[:]) + "\"\n"
	if string(lock1) != wantLock {
		t.Fatalf("lock format or digest mismatch:\n%s\nwant:\n%s", lock1, wantLock)
	}
}

func TestPackageIncludesEmbeddedAuthoringProjectAndResourcesOnly(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate release test source")
	}
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", ".."))
	archive, _, err := Package(moduleRoot, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	contents := archiveContents(t, archive)
	license, err := os.ReadFile(filepath.Join(moduleRoot, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	license, err = normalizeTextSource("LICENSE", license)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents["LICENSE"], license) {
		t.Fatal("source archive omitted or changed the root license")
	}
	want := []string{
		"internal/authoring/resources/rule-canonical-ownership.yaml",
		"internal/authoring/resources/skill-authoring.yaml",
		"internal/authoring/resources/text-resource-modelling.yaml",
		"internal/authoring/resources/workflow-authoring-change.yaml",
		"internal/authoring/resources/workflow-constitution-change.yaml",
		"internal/authoring/resources/workflow-engineering-discovery.yaml",
		"internal/authoring/resources/workflow-markitect-first-change.yaml",
	}
	var got []string
	for name := range contents {
		if strings.HasPrefix(name, "internal/authoring/resources/") {
			got = append(got, name)
		}
	}
	sort.Strings(got)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("bundled authoring resources in source archive = %v, want %v", got, want)
	}
	projectPath := "internal/authoring/project.yaml"
	projectData, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(projectPath)))
	if err != nil {
		t.Fatal(err)
	}
	projectData, err = normalizeTextSource(projectPath, projectData)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents[projectPath], projectData) {
		t.Fatal("archive omitted or changed the embedded authoring Project")
	}
	if _, included := contents["internal/authoring/notes.yaml"]; included {
		t.Fatal("unrelated YAML under internal/authoring was packaged")
	}
	for _, name := range want {
		data, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		data, err = normalizeTextSource(name, data)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(contents[name], data) {
			t.Errorf("archive content differs from embedded resource source %s", name)
		}
	}
	if _, included := contents["internal/authoring/testdata/not-embedded.yaml"]; included {
		t.Fatal("unrelated YAML under internal was packaged")
	}
}

func TestPackageChangesWhenIncludedSourceChanges(t *testing.T) {
	root := fixtureRoot(t)
	before, beforeLock, err := Package(root, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "tools", "markitect", "internal", "core", "model.go")
	if err := os.WriteFile(path, []byte("package core\n// changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	after, afterLock, err := Package(root, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after) || bytes.Equal(beforeLock, afterLock) {
		t.Fatal("included source edit did not change archive and digest lock")
	}
}

func TestPackageNormalizesTextLineEndingsForReproducibleArchive(t *testing.T) {
	root := fixtureRoot(t)
	lfArchive, lfLock, err := Package(root, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	moduleDir := filepath.Join(root, "tools", "markitect")
	for _, relative := range []string{
		"go.mod",
		"go.sum",
		"LICENSE",
		"README.md",
		"cmd/markitect/main.go",
		"internal/core/model.go",
		"internal/format/schema.go",
		"internal/release/package.go",
		"internal/release/package_test.go",
		"schema/manifest.yaml",
	} {
		full := filepath.Join(moduleDir, filepath.FromSlash(relative))
		data, err := os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
		if err := os.WriteFile(full, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	crlfArchive, crlfLock, err := Package(root, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(lfArchive, crlfArchive) || !bytes.Equal(lfLock, crlfLock) {
		t.Fatal("CRLF checkout changed the canonical source archive or lock")
	}
	entries := archiveContents(t, crlfArchive)
	if got, want := string(entries["internal/core/model.go"]), "package core\n"; got != want {
		t.Fatalf("normalized source = %q, want %q", got, want)
	}
}

func TestPackageRejectsSymlinksInSourceTree(t *testing.T) {
	root := fixtureRoot(t)
	external := filepath.Join(t.TempDir(), "outside.go")
	if err := os.WriteFile(external, []byte("package outside\n"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "tools", "markitect", "cmd", "markitect", "linked.go")
	if err := os.Symlink(external, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, _, err := Package(root, "1.0.0"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestPackageAcceptsSemverReleasesAndRejectsMalformedVersions(t *testing.T) {
	for _, version := range []string{"0.0.1", "v1.2.3", "1.2.3-rc.1", "v1.2.3-alpha.2+sha.abc"} {
		if !validVersion(version) {
			t.Errorf("valid version rejected: %s", version)
		}
	}
	for _, version := range []string{"", "latest", "01.2.3", "1.02.3", "1.2", "1.2.3-01", "1.2.3+build+again", "v1.2.3/evil"} {
		if validVersion(version) {
			t.Errorf("invalid version accepted: %s", version)
		}
	}
}
