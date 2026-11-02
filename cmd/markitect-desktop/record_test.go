//go:build native

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordKeepsOneLinePerError(t *testing.T) {
	// The folder does not exist yet at a first start.
	dir := filepath.Join(t.TempDir(), "Markitect")
	record(dir, errors.Join(errors.New("first"), errors.New("second")))
	record(dir, errors.New("third"))
	log, err := os.ReadFile(filepath.Join(dir, "desktop-error.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(log), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), log)
	}
	for i, want := range []string{"first second", "third"} {
		when, message, _ := strings.Cut(lines[i], " ")
		if _, err := time.Parse(time.RFC3339, when); err != nil {
			t.Errorf("line %d has no time: %v", i, err)
		}
		if message != want {
			t.Errorf("line %d says %q, want %q", i, message, want)
		}
	}
}

func TestRecordWithNothingToWrite(t *testing.T) {
	dir := t.TempDir()
	record(dir, nil)
	if _, err := os.Stat(filepath.Join(dir, "desktop-error.log")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("no error wrote a log: %v", err)
	}
	// Without a config folder there is nowhere to write, and nothing breaks.
	t.Chdir(dir)
	record("", errors.New("nowhere"))
	if _, err := os.Stat("desktop-error.log"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("no folder wrote a log here: %v", err)
	}
}
