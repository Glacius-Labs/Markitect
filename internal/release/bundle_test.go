package release

import (
	"archive/zip"
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/source"
	"go.yaml.in/yaml/v3"
)

func TestBuildBundleIsDeterministicAndBindsFiveFiles(t *testing.T) {
	snapshot := bundleSnapshot()
	first, err := BuildBundle(snapshot, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildBundle(snapshot, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("building from the same snapshot changed bundle bytes")
	}
	bundle, err := ParseBundle(first, digestBytes(first))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Manifest.Version != "1.2.3" || bundle.Manifest.SourceCommit != strings.Repeat("a", 40) || bundle.Manifest.SourceRepository != sourceRepository {
		t.Fatalf("unexpected release manifest: %+v", bundle.Manifest)
	}
	if len(bundle.Files) != 5 {
		t.Fatalf("bundle has %d files, want 5", len(bundle.Files))
	}
	sourceFiles := archiveContents(t, bundle.Files["tools/markitect/source.zip"])
	if got := string(sourceFiles["LICENSE"]); got != "Apache License\nVersion 2.0\n" {
		t.Fatalf("source archive license = %q", got)
	}
	for _, name := range append(append([]string(nil), bundlePaths...), releaseManifestPath) {
		if _, ok := bundle.Files[name]; !ok {
			t.Errorf("bundle omitted %s", name)
		}
	}
	if got := string(bundle.Files["scripts/run-markitect.go"]); strings.Contains(got, "\r") {
		t.Fatal("integration runner was not normalized to LF")
	}
	if got := string(bundle.Files["scripts/markitect-bootstrap_test.go"]); strings.Contains(got, "\r") {
		t.Fatal("integration test was not normalized to LF")
	}
	if err := ValidateBundleFiles(bundle.Manifest, bundle.Files, snapshot.Revision); err != nil {
		t.Fatalf("validate installed bundle files: %v", err)
	}
}

func TestBuildBundleRejectsUnfixedOrUnboundSource(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*source.Snapshot)
		version string
	}{
		{name: "provisional", change: func(s *source.Snapshot) { s.Provisional = true }, version: "1.2.3"},
		{name: "short commit", change: func(s *source.Snapshot) { s.Revision = "deadbeef" }, version: "1.2.3"},
		{name: "version mismatch", change: func(*source.Snapshot) {}, version: "1.2.4"},
		{name: "version with tag prefix", change: func(*source.Snapshot) {}, version: "v1.2.3"},
		{name: "nonliteral source version", change: func(s *source.Snapshot) {
			s.Files["cmd/markitect/main.go"] = []byte("package main\nvar version = currentVersion()\n")
		}, version: "1.2.3"},
		{name: "foreign module", change: func(s *source.Snapshot) { s.Files["go.mod"] = []byte("module example.invalid/foreign\n\ngo 1.27.1\n") }, version: "1.2.3"},
		{name: "missing paired test", change: func(s *source.Snapshot) {
			delete(s.Files, "integration/run-markitect_test.go")
			delete(s.Modes, "integration/run-markitect_test.go")
		}, version: "1.2.3"},
		{name: "missing license", change: func(s *source.Snapshot) {
			delete(s.Files, "LICENSE")
			delete(s.Modes, "LICENSE")
		}, version: "1.2.3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := bundleSnapshot()
			test.change(snapshot)
			if _, err := BuildBundle(snapshot, test.version); err == nil {
				t.Fatal("BuildBundle accepted invalid source binding")
			}
		})
	}
}

func TestParseBundleChecksOuterDigestAndMemberHashes(t *testing.T) {
	data, err := BuildBundle(bundleSnapshot(), "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseBundle(data, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong expected outer hash was accepted")
	}
	mutated := rewriteBundle(t, data, func(files map[string][]byte) {
		files["scripts/run-markitect.go"] = append(files["scripts/run-markitect.go"], []byte("// mutation\n")...)
	})
	if _, err := ParseBundle(mutated, digestBytes(mutated)); err == nil || !strings.Contains(err.Error(), "manifest SHA-256") {
		t.Fatalf("modified bootstrap result = %v", err)
	}
}

func TestParseBundleRejectsMissingExtraDuplicateAndUnsafeFiles(t *testing.T) {
	data, err := BuildBundle(bundleSnapshot(), "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("missing", func(t *testing.T) {
		bad := rewriteBundleEntries(t, data, func(entries *[]bundleTestEntry) {
			*entries = (*entries)[:len(*entries)-1]
		})
		assertBundleRejected(t, bad)
	})
	t.Run("extra", func(t *testing.T) {
		bad := rewriteBundleEntries(t, data, func(entries *[]bundleTestEntry) {
			(*entries)[0].name = "unexpected.txt"
		})
		assertBundleRejected(t, bad)
	})
	t.Run("duplicate", func(t *testing.T) {
		bad := rewriteBundleEntries(t, data, func(entries *[]bundleTestEntry) {
			duplicate := (*entries)[0]
			*entries = (*entries)[:len(*entries)-1]
			*entries = append(*entries, duplicate)
		})
		assertBundleRejected(t, bad)
	})
	t.Run("path traversal", func(t *testing.T) {
		bad := rewriteBundleEntries(t, data, func(entries *[]bundleTestEntry) {
			(*entries)[0].name = "../markitect.lock.yaml"
		})
		assertBundleRejected(t, bad)
	})
	t.Run("symlink", func(t *testing.T) {
		bad := rewriteBundleEntriesWithMode(t, data, func(entries *[]bundleTestEntry) {
			for i := range *entries {
				if (*entries)[i].name == "scripts/run-markitect.go" {
					(*entries)[i].mode = os.ModeSymlink | 0777
				}
			}
		})
		assertBundleRejected(t, bad)
	})
	t.Run("fifo", func(t *testing.T) {
		bad := rewriteBundleEntriesWithMode(t, data, func(entries *[]bundleTestEntry) {
			for i := range *entries {
				if (*entries)[i].name == "scripts/run-markitect.go" {
					(*entries)[i].mode = os.ModeNamedPipe | 0644
				}
			}
		})
		assertBundleRejected(t, bad)
	})
}

func TestParseBundleRejectsLockAndManifestMismatches(t *testing.T) {
	data, err := BuildBundle(bundleSnapshot(), "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	badVersion := rewriteBundle(t, data, func(files map[string][]byte) {
		files["markitect.lock.yaml"] = []byte("version: \"1.2.4\"\nsource: \"tools/markitect/source.zip\"\nsha256: \"" + digestBytes(files["tools/markitect/source.zip"]) + "\"\n")
		manifest, err := ParseBundleManifest(files[releaseManifestPath])
		if err != nil {
			t.Fatal(err)
		}
		for i := range manifest.Files {
			if manifest.Files[i].Path == "markitect.lock.yaml" {
				manifest.Files[i].SHA256 = digestBytes(files["markitect.lock.yaml"])
			}
		}
		files[releaseManifestPath], err = yaml.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
	})
	if _, err := ParseBundle(badVersion, digestBytes(badVersion)); err == nil || !strings.Contains(err.Error(), "lock") {
		t.Fatalf("lock version mismatch result = %v", err)
	}
	badManifest := rewriteBundle(t, data, func(files map[string][]byte) {
		files[releaseManifestPath] = append(files[releaseManifestPath], []byte("---\n")...)
	})
	if _, err := ParseBundle(badManifest, digestBytes(badManifest)); err == nil {
		t.Fatal("multi-document manifest was accepted")
	}
}

func TestParseBundleRejectsOversizedManifest(t *testing.T) {
	data, err := BuildBundle(bundleSnapshot(), "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	bad := rewriteBundle(t, data, func(files map[string][]byte) {
		files[releaseManifestPath] = bytes.Repeat([]byte("x"), maxManifestBytes+1)
	})
	if _, err := ParseBundle(bad, digestBytes(bad)); err == nil || !strings.Contains(err.Error(), "manifest exceeds size limit") {
		t.Fatalf("oversized manifest result = %v", err)
	}
}

func TestLegacyToolLockValidation(t *testing.T) {
	data, err := BuildBundle(bundleSnapshot(), "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := ParseBundle(data, digestBytes(data))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseToolLock(bundle.Files["markitect.lock.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateToolLockFiles(lock, bundle.Files); err != nil {
		t.Fatal(err)
	}
	files := cloneBundleFiles(bundle.Files)
	files["tools/markitect/source.zip"] = append(files["tools/markitect/source.zip"], 0)
	if err := ValidateToolLockFiles(lock, files); err == nil {
		t.Fatal("legacy lock accepted a modified source archive")
	}
}

func bundleSnapshot() *source.Snapshot {
	files := map[string][]byte{
		"go.mod":                 []byte("module github.com/Glacius-Labs/Markitect\n\ngo 1.27.1\n"),
		"go.sum":                 []byte("go.yaml.in/yaml/v3 v3.0.5 h1:fixture\n"),
		"README.md":              []byte("Markitect\r\n"),
		"LICENSE":                []byte("Apache License\r\nVersion 2.0\r\n"),
		"cmd/markitect/main.go":  []byte("package main\nvar version = \"1.2.3\"\n"),
		"internal/core/model.go": []byte("package core\n"),
		"internal/authoring/resources/skill.yaml": []byte("kind: Skill\n"),
		"integration/run-markitect.go":            []byte("package main\r\n"),
		"integration/run-markitect_test.go":       []byte("package main\r\n"),
		"schema/manifest.yaml":                    []byte("schemaVersion: 1\r\n"),
	}
	modes := make(map[string]string, len(files))
	for name := range files {
		modes[name] = "100644"
	}
	return &source.Snapshot{Revision: strings.Repeat("a", 40), Files: files, Modes: modes}
}

type bundleTestEntry struct {
	name string
	data []byte
	mode os.FileMode
}

func rewriteBundle(t *testing.T, data []byte, change func(map[string][]byte)) []byte {
	t.Helper()
	files := archiveContents(t, data)
	change(files)
	entries := make([]bundleTestEntry, 0, len(files))
	for name, content := range files {
		entries = append(entries, bundleTestEntry{name: name, data: content, mode: 0644})
	}
	return writeBundleTestZip(t, entries)
}

func rewriteBundleEntries(t *testing.T, data []byte, change func(*[]bundleTestEntry)) []byte {
	t.Helper()
	entries := bundleTestEntries(t, data)
	change(&entries)
	return writeBundleTestZip(t, entries)
}

func rewriteBundleEntriesWithMode(t *testing.T, data []byte, change func(*[]bundleTestEntry)) []byte {
	t.Helper()
	entries := bundleTestEntries(t, data)
	change(&entries)
	return writeBundleTestZip(t, entries)
}

func bundleTestEntries(t *testing.T, data []byte) []bundleTestEntry {
	t.Helper()
	files := archiveContents(t, data)
	entries := make([]bundleTestEntry, 0, len(files))
	for name, content := range files {
		entries = append(entries, bundleTestEntry{name: name, data: content, mode: 0644})
	}
	return entries
}

func writeBundleTestZip(t *testing.T, entries []bundleTestEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, item := range entries {
		header := &zip.FileHeader{Name: item.name, Method: zip.Store}
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(item.mode)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(item.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func assertBundleRejected(t *testing.T, data []byte) {
	t.Helper()
	if _, err := ParseBundle(data, digestBytes(data)); err == nil {
		t.Fatal("invalid release bundle was accepted")
	}
}

func cloneBundleFiles(files map[string][]byte) map[string][]byte {
	clone := make(map[string][]byte, len(files))
	for name, data := range files {
		clone[name] = append([]byte(nil), data...)
	}
	return clone
}

func TestBundleManifestRequiresSourceVersionLiteral(t *testing.T) {
	snapshot := bundleSnapshot()
	snapshot.Files["cmd/markitect/main.go"] = []byte("package main\nvar version = 1.2.3\n")
	if _, err := BuildBundle(snapshot, "1.2.3"); err == nil {
		t.Fatal("non-string source version literal was accepted")
	}
}

func TestBundleVersionErrorIncludesMismatch(t *testing.T) {
	snapshot := bundleSnapshot()
	_, err := BuildBundle(snapshot, "1.2.4")
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("version mismatch error = %v", err)
	}
}
