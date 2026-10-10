package examples

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// harnessRepositoryRoot locates the checkout by its module and source markers,
// rather than assuming the harness package sits directly under the checkout.
func harnessRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		testFile = ""
	}
	start := ""
	if filepath.IsAbs(testFile) {
		start = filepath.Dir(testFile)
	} else if current, err := os.Getwd(); err == nil {
		start = current
	} else {
		t.Fatalf("cannot locate harness checkout root: %v", err)
	}
	for current := start; ; current = filepath.Dir(current) {
		if info, err := os.Stat(filepath.Join(current, "go.mod")); err == nil && !info.IsDir() {
			if _, err := os.Stat(filepath.Join(current, "src", "internal")); err == nil {
				return current
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("could not find Markitect checkout root above %s", start)
		}
	}
}
