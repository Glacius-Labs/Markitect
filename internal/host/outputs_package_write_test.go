package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWritePackageConsumerUsesPhysicalArchiveAndPreservesPackageMembers(t *testing.T) {
	fixture := packageConsumerFixture(t, "Imported notes remain in the pinned archive.")
	root := tempRoot(t)
	initWriterRepo(t, root)
	writeFixture(t, root, fixture.Snapshot.Files)
	p, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	written, err := WriteOutputs(root, p)
	if err != nil || len(written) == 0 {
		t.Fatalf("consumer render treated archive members as physical sources: %v, %v", written, err)
	}
	for _, virtualPath := range []string{"markitect-package.yaml", "docs/review.yaml"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(virtualPath))); !os.IsNotExist(err) {
			t.Fatalf("render materialized a package member at %s: %v", virtualPath, err)
		}
	}
	if string(mustRead(t, filepath.Join(root, "docs", "input.txt"))) != "Unrelated local file." {
		t.Fatal("imported input overwrote the unrelated consumer file")
	}
	// A changed archive must still invalidate the captured write before effects.
	p, err = Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, "packages", "review-kit.zip")
	archive := mustRead(t, archivePath)
	if err := os.WriteFile(archivePath, append(archive, 'x'), 0644); err != nil {
		t.Fatal(err)
	}
	if paths, err := WriteOutputs(root, p); err == nil || len(paths) != 0 {
		t.Fatalf("changed archive accepted by a captured writer: %v, %v", paths, err)
	}
}
