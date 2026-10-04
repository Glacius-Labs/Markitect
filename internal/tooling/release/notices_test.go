package release

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/tooling/licenses"
)

func TestSourceDistributionsRetainEmbeddedNotices(t *testing.T) {
	root := fixtureRoot(t)
	module := filepath.Join(root, "tools", "markitect")
	full := filepath.Join(module, filepath.FromSlash(embeddedNoticesPath))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(licenses.Text), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(full), "unrelated.md"), []byte("unrelated"), 0644); err != nil {
		t.Fatal(err)
	}
	archive, _, err := Package(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	assertNotices := func(archive []byte) {
		t.Helper()
		contents := archiveContents(t, archive)
		if !bytes.Equal(contents[embeddedNoticesPath], []byte(licenses.Text)) {
			t.Fatal("source archive omitted or changed the complete embedded notices")
		}
		if _, exists := contents["internal/tooling/licenses/unrelated.md"]; exists {
			t.Fatal("notice inclusion broadened to unrelated internal Markdown")
		}
	}
	assertNotices(archive)

	fixed := bundleSnapshot()
	fixed.Files[embeddedNoticesPath] = []byte(licenses.Text)
	fixed.Modes[embeddedNoticesPath] = "100644"
	fixed.Files["internal/tooling/licenses/unrelated.md"] = []byte("unrelated")
	fixed.Modes["internal/tooling/licenses/unrelated.md"] = "100644"
	bundleBytes, err := BuildBundle(fixed, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := ParseBundle(bundleBytes, digestBytes(bundleBytes))
	if err != nil {
		t.Fatal(err)
	}
	assertNotices(bundle.Files[".markitect/tool/source.zip"])
}
