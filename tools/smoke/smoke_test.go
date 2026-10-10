package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, content := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUnzipExtractsNestedFiles(t *testing.T) {
	target := t.TempDir()
	if err := unzip(writeZip(t, map[string]string{"src/cmd/markitect/main.go": "package main\n", "go.mod": "module x\n"}), target); err != nil {
		t.Fatal(err)
	}
	if err := requirePresent(target, "src/cmd/markitect/main.go", "go.mod"); err != nil {
		t.Fatal(err)
	}
}

func TestUnzipRefusesEntriesOutsideTheTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	err := unzip(writeZip(t, map[string]string{"../escaped.txt": "x"}), target)
	if err == nil || !strings.Contains(err.Error(), "leaves the target") {
		t.Fatalf("unzip accepted an escaping entry: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(target), "escaped.txt")); statErr == nil {
		t.Fatal("the escaping entry was written")
	}
}

func TestPresenceAndDigestChecks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "present.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if requirePresent(root, "present.txt") != nil || requirePresent(root, "missing.txt") == nil {
		t.Fatal("requirePresent misjudged a file")
	}
	if requireAbsent(root, "missing.txt") != nil || requireAbsent(root, "present.txt") == nil {
		t.Fatal("requireAbsent misjudged a file")
	}
	if !isDigest(strings.Repeat("a", 64)) || isDigest(strings.Repeat("A", 64)) || isDigest("abc") {
		t.Fatal("isDigest accepts only 64 lowercase hexadecimal characters")
	}
}

func TestFailuresShowTheTailOfLongOutput(t *testing.T) {
	long := strings.Repeat("x", 5000) + "the end"
	if got := tail(long); !strings.HasPrefix(got, "...") || !strings.HasSuffix(got, "the end") || len(got) > 2010 {
		t.Fatalf("tail kept %d bytes", len(got))
	}
	if err := requireContains("abc", "b", "z"); err == nil || !strings.Contains(err.Error(), `"z"`) {
		t.Fatalf("requireContains did not name the missing text: %v", err)
	}
}
