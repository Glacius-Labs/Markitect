package release

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
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
	want := []string{"README.md", "cmd/markitect/main.go", "go.mod", "go.sum", "internal/authoring/resources/markitect.yaml", "internal/core/model.go", "internal/format/schema.go", "internal/release/package.go", "internal/release/package_test.go", "schema/manifest.yaml"}
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
	wantLock := "version: \"v1.2.3-rc.4+build.7\"\nsource: \"tools/markitect/source.zip\"\nsha256: \"" + hex.EncodeToString(hash[:]) + "\"\n"
	if string(lock1) != wantLock {
		t.Fatalf("lock format or digest mismatch:\n%s\nwant:\n%s", lock1, wantLock)
	}
}

func TestPackageIncludesOnlyEmbeddedAuthoringYAMLResources(t *testing.T) {
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
	want := []string{
		"internal/authoring/resources/markitect.yaml",
		"internal/authoring/resources/rule-canonical-ownership.yaml",
		"internal/authoring/resources/skill-authoring.yaml",
		"internal/authoring/resources/text-resource-modelling.yaml",
		"internal/authoring/resources/workflow-authoring-change.yaml",
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

func TestNormalizeTextSourceRejectsInvalidUTF8AndNUL(t *testing.T) {
	for _, input := range [][]byte{{0xff, 0xfe}, []byte("valid\x00text")} {
		if _, err := normalizeTextSource("docs/readme.md", input); err == nil {
			t.Errorf("accepted invalid text bytes %v", input)
		}
	}
	if got, err := normalizeTextSource("schema/manifest.yaml", []byte("a\r\nb\rc")); err != nil || string(got) != "a\nb\nc" {
		t.Fatalf("normalization = %q, err=%v", got, err)
	}
	if got, err := normalizeTextSource("schema/legacy.yml", []byte("a\r\nb\rc")); err != nil || string(got) != "a\nb\nc" {
		t.Fatalf(".yml normalization = %q, err=%v", got, err)
	}
	for _, input := range [][]byte{{0xff}, []byte("has\x00nul")} {
		if _, err := normalizeTextSource("schema/legacy.yml", input); err == nil {
			t.Errorf(".yml accepted invalid text bytes %v", input)
		}
	}
}

func TestNormalizeTextSourceDoesNotRewriteOtherFiles(t *testing.T) {
	for _, name := range []string{"schema/manifest.json", "assets/payload.bin"} {
		input := []byte{0xff, 0x00, '\r', '\n'}
		got, err := normalizeTextSource(name, input)
		if err != nil || !bytes.Equal(got, input) {
			t.Errorf("normalizeTextSource(%q) changed non-target bytes: %v, %v", name, got, err)
		}
	}
}

func TestMakeArchiveRejectsCaseInsensitivePathCollisions(t *testing.T) {
	for _, files := range [][]sourceFile{
		{{name: "cmd/Foo.go"}, {name: "cmd/foo.go"}},
		{{name: "Internal/a.go"}, {name: "internal/b.go"}},
		{{name: "cmd"}, {name: "CMD/a.go"}},
		{{name: "cmd/a.go"}, {name: "cmd/a.go"}},
	} {
		if _, err := makeArchive(files); err == nil || !strings.Contains(err.Error(), "archive path collision") && !strings.Contains(err.Error(), "duplicate archive entry") {
			t.Errorf("makeArchive(%v) error = %v, want path collision", files, err)
		}
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

func TestSafeArchivePathRejectsTraversalAndWindowsSpecialPaths(t *testing.T) {
	for _, candidate := range []string{"../outside.go", "/root.go", "C:/root.go", `internal\\escape.go`, "internal/../escape.go", "internal/CON.go", "internal/name?.go", "internal/a/../b.go"} {
		if err := safeArchivePath(candidate); err == nil {
			t.Errorf("accepted unsafe path %q", candidate)
		}
	}
	for _, candidate := range []string{"go.mod", "cmd/markitect/main.go", "schema/manifest.yaml"} {
		if err := safeArchivePath(candidate); err != nil {
			t.Errorf("rejected valid path %q: %v", candidate, err)
		}
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

func fixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(relative, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(filepath.ToSlash(filepath.Clean(relative))))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("tools/markitect/go.mod", "module markitect\n\ngo 1.24.0\n")
	write("tools/markitect/go.sum", "example checksum\n")
	write("tools/markitect/README.md", "Markitect\n")
	write("tools/markitect/cmd/markitect/main.go", "package main\n")
	write("tools/markitect/internal/core/model.go", "package core\n")
	write("tools/markitect/internal/format/schema.go", "package format\n")
	write("tools/markitect/internal/release/package.go", "package release\n")
	write("tools/markitect/internal/release/package_test.go", "package release\n")
	write("tools/markitect/internal/authoring/resources/markitect.yaml", "apiVersion: markitect.example.org/v1alpha1\nkind: Project\n")
	write("tools/markitect/internal/authoring/testdata/fixture.yaml", "should not be packaged\n")
	write("tools/markitect/schema/manifest.yaml", "schema: v1\n")
	write("tools/markitect/internal/fixture/customer-data.yaml", "proprietary: true\n")
	write("tools/markitect/bin/markitect.exe", "excluded binary")
	write("tools/markitect/.cache/temp.go", "package temp\n")
	write("tools/markitect/.gocache/entry.go", "package cache\n")
	write("tools/markitect/docs/notes.md", "excluded docs")
	return root
}

func archiveEntries(t *testing.T, data []byte) []string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	entries := make([]string, 0, len(reader.File))
	for _, entry := range reader.File {
		entries = append(entries, entry.Name)
	}
	return entries
}

func archiveContents(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	contents := make(map[string][]byte, len(reader.File))
	for _, entry := range reader.File {
		file, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents[entry.Name], err = io.ReadAll(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return contents
}

func verifyNormalZipMetadata(t *testing.T, data []byte) {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	wantTime := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, entry := range reader.File {
		if !entry.Modified.Equal(wantTime) {
			t.Errorf("%s timestamp = %s, want %s", entry.Name, entry.Modified, wantTime)
		}
		if entry.Mode().Perm() != 0644 {
			t.Errorf("%s mode = %o, want 644", entry.Name, entry.Mode().Perm())
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			t.Errorf("archive contains symlink %s", entry.Name)
		}
		file, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, file); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
