package release

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

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
