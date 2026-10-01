package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/release"
	"go.yaml.in/yaml/v3"
)

func recreateAutocrlfCheckout(t *testing.T, root string) {
	t.Helper()
	for _, name := range installPaths {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Fatal(err)
		}
	}
	args := append([]string{"checkout", "--"}, installPaths...)
	runWriterGit(t, root, args...)
	for _, name := range installPaths {
		if name == "tools/markitect/source.zip" {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(name))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS == "windows" && !bytes.Contains(data, []byte("\r\n")) {
			t.Fatalf("core.autocrlf=true did not materialize CRLF in %s", name)
		}
		if !bytes.Contains(data, []byte("\r\n")) {
			data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
			if err := os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func installTestRepo(t *testing.T, branch string) string {
	t.Helper()
	root := tempRoot(t)
	runWriterGit(t, root, "init", "-b", branch)
	runWriterGit(t, root, "config", "user.name", "Markitect Test")
	runWriterGit(t, root, "config", "user.email", "markitect-test@example.invalid")
	writeFixture(t, root, map[string][]byte{"README.md": []byte("test consumer\n")})
	commitInstallPins(t, root, "consumer baseline")
	return root
}

func commitInstallPins(t *testing.T, root, message string) {
	t.Helper()
	runWriterGit(t, root, "add", "-A")
	runWriterGit(t, root, "commit", "-m", message)
}

func installTestBundle(t *testing.T, sourceID, version, marker string) *release.Bundle {
	t.Helper()
	// Keep the fixture unmistakably binary so core.autocrlf never treats the
	// byte-exact archive pin as text during checkout.
	archive := []byte("PK\x03\x04source archive " + marker + "\x00\n")
	archiveHash := digestInstallBytes(archive)
	files := map[string][]byte{
		"markitect.lock.yaml":                 []byte("version: \"" + version + "\"\nsource: \"tools/markitect/source.zip\"\nsha256: \"" + archiveHash + "\"\n"),
		"scripts/markitect-bootstrap_test.go": []byte("package scripts // " + marker + "\n"),
		"scripts/run-markitect.go":            []byte("package main // " + marker + "\n"),
		"tools/markitect/source.zip":          archive,
	}
	manifest := release.BundleManifest{SchemaVersion: 1, Version: version, SourceCommit: strings.Repeat(sourceID, 40), SourceRepository: "github.com/Glacius-Labs/Markitect"}
	for _, name := range installPinPaths {
		manifest.Files = append(manifest.Files, release.BundleFile{Path: name, SHA256: digestInstallBytes(files[name])})
	}
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	files[releaseManifestPath] = manifestBytes
	bundle := &release.Bundle{Manifest: manifest, Files: files, SHA256: digestInstallBytes([]byte("outer bundle " + marker))}
	if err := validateIncomingBundle(bundle); err != nil {
		t.Fatalf("test bundle invalid: %v", err)
	}
	return bundle
}

func writeBundleToRoot(t *testing.T, root string, bundle *release.Bundle, includeManifest bool) {
	t.Helper()
	for _, name := range installPaths {
		if name == releaseManifestPath && !includeManifest {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(name))), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), bundle.Files[name], 0644); err != nil {
			t.Fatal(err)
		}
	}
}
