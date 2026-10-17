package release

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	write("go.mod", "module markitect\n\ngo 1.24.0\n")
	write("go.sum", "example checksum\n")
	write("README.md", "Markitect\n")
	write("LICENSE", "Apache License\nVersion 2.0\n")
	write("cmd/markitect/main.go", "package main\n")
	write("internal/model/model.go", "package core\n")
	write("internal/model/format/schema.go", "package format\n")
	write("internal/tooling/release/package.go", "package release\n")
	write("internal/tooling/release/package_test.go", "package release\n")
	write("internal/apply/embedded/project.yaml", "apiVersion: markitect.example.org/v1alpha1\nkind: Project\n")
	write("internal/apply/embedded/fixture.yaml", "should not be packaged\n")
	write("internal/apply/embedded/notes.yaml", "unrelated source documentation\n")
	write("schema/manifest.yaml", "schema: v1\n")
	write("customer-data.yaml", "proprietary: true\n")
	write("bin/markitect.exe", "excluded binary")
	write(".cache/temp.go", "package temp\n")
	write(".gocache/entry.go", "package cache\n")
	write("docs/notes.md", "excluded docs")
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
