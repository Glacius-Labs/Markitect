package host

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/tooling/release"
	"go.yaml.in/yaml/v3"
)

const rootReleaseManifestPath = ".markitect/tool/release.yaml"

var rootInstallPinPaths = []string{
	".markitect/tool/lock.yaml",
	".markitect/bootstrap/run_test.go",
	".markitect/bootstrap/run.go",
	".markitect/tool/source.zip",
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
	archive := []byte("PK\x03\x04source archive " + marker + "\x00\n")
	archiveHash := rootInstallDigest(archive)
	files := map[string][]byte{
		".markitect/tool/lock.yaml":        []byte("version: \"" + version + "\"\nsource: \".markitect/tool/source.zip\"\nsha256: \"" + archiveHash + "\"\n"),
		".markitect/bootstrap/run_test.go": []byte("package main // " + marker + "\n"),
		".markitect/bootstrap/run.go":      []byte("package main // " + marker + "\n"),
		".markitect/tool/source.zip":       archive,
	}
	manifest := release.BundleManifest{SchemaVersion: 2, Version: version, SourceCommit: strings.Repeat(sourceID, 40), SourceRepository: "github.com/Glacius-Labs/Markitect"}
	names := append([]string(nil), rootInstallPinPaths...)
	sort.Strings(names)
	for _, name := range names {
		manifest.Files = append(manifest.Files, release.BundleFile{Path: name, SHA256: rootInstallDigest(files[name])})
	}
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	files[rootReleaseManifestPath] = manifestBytes
	bundle := &release.Bundle{Manifest: manifest, Files: files, SHA256: rootInstallDigest([]byte("outer bundle " + marker))}
	return bundle
}

func rootInstallDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
