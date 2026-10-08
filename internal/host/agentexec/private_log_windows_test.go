//go:build windows

package agentexec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateLogWindowsDirectoryAndFileACL(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "private-logs")
	prepared, err := preparePrivateLogDirectory(directory, nil)
	if err != nil {
		t.Fatalf("create owner-only log directory: %v", err)
	}
	if filepath.Base(prepared) != filepath.Base(directory) {
		t.Fatalf("prepared path = %q, want %q", prepared, directory)
	}
	if err := verifyPrivateLogDirectory(prepared); err != nil {
		t.Fatalf("verify owner-only log directory: %v", err)
	}
	logPath := filepath.Join(prepared, "run.jsonl")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := verifyPrivateLogFile(logPath); err != nil {
		t.Fatalf("verify inherited owner-only log file ACL: %v", err)
	}
}

func TestPrivateLogWindowsRejectsUnverifiedDirectoryACL(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "inherited-logs")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := preparePrivateLogDirectory(directory, nil); err == nil {
		t.Skip("the test temp parent itself has the exact protected owner-only ACL")
	}
	if err := verifyPrivateLogDirectory(directory); err == nil {
		t.Fatal("rejected existing directory was unexpectedly changed into an accepted ACL")
	}
}
