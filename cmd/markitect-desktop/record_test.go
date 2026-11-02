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

func TestEndingMessageSaysWhatToDo(t *testing.T) {
	dir := filepath.Join("config", "Markitect")
	logFile := filepath.Join(dir, "desktop-error.log")
	tests := []struct {
		name   string
		reason string
		want   []string
	}{
		{
			"a missing web view",
			"error calling Webview2Loader: no webview2 found",
			[]string{"WebView2 Runtime", "https://developer.microsoft.com/microsoft-edge/webview2/", "start Markitect again"},
		},
		{
			// As Wails words the 60 seconds; the test does not wait for them.
			"the 60 seconds",
			"WebView2 controller did not complete initialisation within 1m0s",
			[]string{"within 60 seconds", "Start Markitect again"},
		},
		{
			"any other failure",
			"unable to create window",
			[]string{"Markitect has to close: unable to create window", "Start Markitect again"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := endingMessage(dir, errors.New(tt.reason))
			for _, want := range append(tt.want, logFile) {
				if !strings.Contains(message, want) {
					t.Errorf("the message says %q, want it to hold %q", message, want)
				}
			}
			// Without a config folder nothing is recorded, so it names no log.
			if message := endingMessage("", errors.New(tt.reason)); strings.Contains(message, "desktop-error.log") {
				t.Errorf("without a folder the message says %q", message)
			}
		})
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
