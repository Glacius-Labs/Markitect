//go:build windows

package agentexec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateLogWindowsDirectoryAndFileACL(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "private-logs")
	if err := createPrivateLogDirectory(directory); err != nil {
		t.Fatalf("create owner-only log directory: %v", err)
	}
	prepared, err := preparePrivateLogDirectory(directory, nil)
	if err != nil {
		t.Fatalf("verify existing owner-only log directory: %v", err)
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

func TestPrivateLogWindowsFileDefaultOwnerGroupIsNormalized(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "private-logs")
	prepared, err := preparePrivateLogDirectory(directory, nil)
	if err != nil {
		t.Fatalf("create owner-only log directory: %v", err)
	}
	logPath := filepath.Join(prepared, "run.jsonl")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	currentSID, err := currentUserSID()
	if err != nil {
		t.Fatal(err)
	}
	const administratorsSID = "S-1-5-32-544"
	if currentSID == administratorsSID {
		t.Skip("the current token user is itself BUILTIN\\Administrators")
	}
	if err := setPrivateObjectOwner(logPath, administratorsSID); err != nil {
		t.Skipf("cannot construct an alternate default owner with this token: %v", err)
	}
	if err := verifyPrivateLogFile(logPath); err != nil {
		t.Fatalf("file with inherited owner-only DACL should be pinned to the current user owner: %v", err)
	}
}

// A plain directory, as an older release's journal code created it, is refused
// with guidance rather than adopted: its contents cannot be attested.
func TestPrivateLogWindowsRejectsUnverifiedDirectoryACL(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "inherited-logs")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	_, err := PreparePrivateLogDirectory(directory)
	if err == nil {
		t.Skip("the test temp parent itself has the exact protected owner-only ACL")
	}
	if !strings.Contains(err.Error(), "is not owner-only") || !strings.Contains(err.Error(), "remove it") {
		t.Fatalf("refusal does not tell the user what is wrong and what to do: %v", err)
	}
	if err := verifyPrivateLogDirectory(directory); err == nil {
		t.Fatal("rejected existing directory was unexpectedly changed into an accepted ACL")
	}
}

func TestPrivateACEOwnerSIDSizeRejectsTruncatedSIDHeader(t *testing.T) {
	for _, size := range []int{8, 9, 15} {
		t.Run(fmt.Sprintf("%d-bytes", size), func(t *testing.T) {
			if _, err := privateACEOwnerSIDSize(make([]byte, size)); err == nil {
				t.Fatalf("expected %d-byte ACE body to fail before SID indexing", size)
			}
		})
	}

	ace := make([]byte, 16)
	size, err := privateACEOwnerSIDSize(ace)
	if err != nil || size != 8 {
		t.Fatalf("minimum complete SID header size = %d, %v; want 8, nil", size, err)
	}
}
