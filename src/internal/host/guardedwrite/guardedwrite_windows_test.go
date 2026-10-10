//go:build windows

package guardedwrite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuardedWriteAcceptsShortAndLongRootSpellings(t *testing.T) {
	root, shortRoot := windowsTestPath(t)
	runGit(t, root, "init", "-b", "codex/short-root")
	for _, captureRoot := range []string{root, shortRoot} {
		for _, applyRoot := range []string{root, shortRoot} {
			t.Run(captureRoot+"/"+applyRoot, func(t *testing.T) {
				capture, err := CaptureFiles(captureRoot, []string{"result.txt"})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Apply(applyRoot, capture, []Change{{Path: "result.txt", Bytes: []byte("updated\n"), Mode: 0644}}); err != nil {
					t.Fatalf("apply with equivalent root spelling: %v", err)
				}
				if got := string(mustRead(t, filepath.Join(root, "result.txt"))); got != "updated\n" {
					t.Fatalf("written bytes = %q", got)
				}
				if err := os.Remove(filepath.Join(root, "result.txt")); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
