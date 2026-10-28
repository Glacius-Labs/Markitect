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
	want := []string{"LICENSE", "README.md", "go.mod", "go.sum", "schema/manifest.yaml", "cmd/markitect/main.go", "internal/model/model.go", "internal/model/format/schema.go", "internal/tooling/release/package.go", "internal/tooling/release/package_test.go"}
	sort.Strings(want)
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

func TestPackageOmitsRetiredEmbeddedAuthoringSource(t *testing.T) {
	moduleRoot := sourceModuleRoot(t)
	prefix, err := sourceLayoutPrefix(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
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
	for name := range contents {
		if strings.HasPrefix(name, prefix+"internal/project/embedded/") {
			t.Errorf("retired embedded authoring source was packaged: %s", name)
		}
	}
}

func sourceModuleRoot(t *testing.T) string {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate release test source")
	}
	// This package lives at internal/tooling/release under the root Go module.
	root := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolved source module root %q has no go.mod: %v", root, err)
	}
	return root
}

func TestPackageChangesWhenIncludedSourceChanges(t *testing.T) {
	root := fixtureRoot(t)
	before, beforeLock, err := Package(root, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "internal", "model", "model.go")
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
	moduleDir := root
	for _, relative := range []string{
		"go.mod",
		"go.sum",
		"LICENSE",
		"README.md",
		"cmd/markitect/main.go",
		"internal/model/model.go",
		"internal/model/format/schema.go",
		"internal/tooling/release/package.go",
		"internal/tooling/release/package_test.go",
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
	if got, want := string(entries["internal/model/model.go"]), "package core\n"; got != want {
		t.Fatalf("normalized source = %q, want %q", got, want)
	}
}

func TestPackageRejectsSymlinksInSourceTree(t *testing.T) {
	root := fixtureRoot(t)
	external := filepath.Join(t.TempDir(), "outside.go")
	if err := os.WriteFile(external, []byte("package outside\n"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "cmd", "markitect", "linked.go")
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
