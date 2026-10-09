package release

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
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
	if bundle.Manifest.SchemaVersion != 2 {
		t.Fatalf("bundle schema version = %d, want 2", bundle.Manifest.SchemaVersion)
	}
	sourceFiles := archiveContents(t, bundle.Files[".markitect/tool/source.zip"])
	if got := string(sourceFiles["LICENSE"]); got != "Apache License\nVersion 2.0\n" {
		t.Fatalf("source archive license = %q", got)
	}
	for _, name := range append(append([]string(nil), bundlePaths...), releaseManifestPath) {
		if _, ok := bundle.Files[name]; !ok {
			t.Errorf("bundle omitted %s", name)
		}
	}
	if got := string(bundle.Files[bootstrapPath]); strings.Contains(got, "\r") {
		t.Fatal("integration runner was not normalized to LF")
	}
	if got := string(bundle.Files[bootstrapTestPath]); strings.Contains(got, "\r") {
		t.Fatal("integration test was not normalized to LF")
	}
	if err := ValidateBundleFiles(bundle.Manifest, bundle.Files, snapshot.ID); err != nil {
		t.Fatalf("validate installed bundle files: %v", err)
	}
}

func TestBuildBundleRejectsUnfixedOrUnboundSource(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*snapshot.Snapshot)
		version string
	}{
		{name: "provisional", change: func(s *snapshot.Snapshot) { s.Provisional = true }, version: "1.2.3"},
		{name: "short commit", change: func(s *snapshot.Snapshot) { s.ID = "deadbeef" }, version: "1.2.3"},
		{name: "version mismatch", change: func(*snapshot.Snapshot) {}, version: "1.2.4"},
		{name: "version with tag prefix", change: func(*snapshot.Snapshot) {}, version: "v1.2.3"},
		{name: "nonliteral source version", change: func(s *snapshot.Snapshot) {
			s.Files["src/internal/host/cli/version.go"] = []byte("package cli\nvar version = currentVersion()\n")
		}, version: "1.2.3"},
		{name: "foreign module", change: func(s *snapshot.Snapshot) {
			s.Files["go.mod"] = []byte("module example.invalid/foreign\n\ngo 1.27.1\n")
		}, version: "1.2.3"},
		{name: "missing paired test", change: func(s *snapshot.Snapshot) {
			delete(s.Files, "integration/run-markitect_test.go")
			delete(s.Modes, "integration/run-markitect_test.go")
		}, version: "1.2.3"},
		{name: "missing license", change: func(s *snapshot.Snapshot) {
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
