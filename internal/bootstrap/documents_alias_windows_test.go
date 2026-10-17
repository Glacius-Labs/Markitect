package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestUnselectedNativeAliasCannotMakeDocumentTargetAnExternalInput(t *testing.T) {
	repo := productFixture(t)
	convert := func(path string, call func(*uint16, *uint16, uint32) (uint32, error)) string {
		t.Helper()
		input, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			t.Fatal(err)
		}
		buffer := make([]uint16, 32768)
		n, err := call(input, &buffer[0], uint32(len(buffer)))
		if err != nil || n == 0 || n >= uint32(len(buffer)) {
			t.Fatalf("native path conversion: %d %v", n, err)
		}
		return filepath.Clean(syscall.UTF16ToString(buffer[:n]))
	}
	long := convert(repo.Dir, syscall.GetLongPathName)
	short := convert(long, syscall.GetShortPathName)
	if rel, err := filepath.Rel(long, short); err != nil {
		t.Fatal(err)
	} else if rel == "." {
		t.Skip("fixture filesystem provides no distinct native 8.3 spelling")
	}
	// The explicit dependency root physically includes the existing Markdown
	// target. Its unselected alias has no repository membership proof. Markdown
	// remains a valid non-Definition file in this otherwise valid Core model.
	moveFixtureCore(t, repo, filepath.Join(long, "docs"), ".")
	options := []string{"--dependency", filepath.Join(short, "docs", "model.yaml")}
	preview := docsPreview(t, long, options...)
	args := append([]string{"docs", "--write", "--expect", preview.View.PreviewDigest}, options...)
	out := invoke(t, long, 3, args...)
	if !strings.Contains(out, "requires repository-bound sources") {
		t.Fatal(out)
	}
	actual, err := os.ReadFile(filepath.Join(long, "docs", "markitect", "project.md"))
	if err != nil || string(actual) != "reviewed prior generated target\n" {
		t.Fatalf("protected target changed: %q %v", actual, err)
	}
}
