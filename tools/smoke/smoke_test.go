package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresenceCheck(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "present.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if requirePresent(root, "present.txt") != nil || requirePresent(root, "missing.txt") == nil {
		t.Fatal("requirePresent misjudged a file")
	}
}

func TestFailuresShowTheTailOfLongOutput(t *testing.T) {
	long := strings.Repeat("x", 5000) + "the end"
	if got := tail(long); !strings.HasPrefix(got, "...") || !strings.HasSuffix(got, "the end") || len(got) > 2010 {
		t.Fatalf("tail kept %d bytes", len(got))
	}
	if err := requireContains("abc", "b", "z"); err == nil || !strings.Contains(err.Error(), `"z"`) {
		t.Fatalf("requireContains did not name the missing text: %v", err)
	}
}
