package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalControllerLeaseRemainsExclusiveAndReleasesOwnFile(t *testing.T) {
	parent := t.TempDir()
	cfg := CanonicalControllerConfig{RecordStore: filepath.Join(parent, "ledger")}
	leasePath := cfg.RecordStore + ".controller.lock"

	release, err := acquireCanonicalControllerLease(cfg)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Fatalf("lease content changed: size %d", info.Size())
	}
	if _, err := acquireCanonicalControllerLease(cfg); err == nil {
		t.Fatal("second lease acquisition succeeded while the first was held")
	}
	release()
	if _, err := os.Lstat(leasePath); !os.IsNotExist(err) {
		t.Fatalf("release did not remove its lease: %v", err)
	}
}

func TestCanonicalControllerLeaseReleaseAfterParentRenameDoesNotDeleteReplacement(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "record-store")
	moved := filepath.Join(root, "record-store-moved")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := CanonicalControllerConfig{RecordStore: filepath.Join(parent, "ledger")}
	release, err := acquireCanonicalControllerLease(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(parent, moved); err != nil {
		release()
		t.Skipf("directory rename unavailable while the pinned lease parent is open: %v", err)
	}
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	replacement := cfg.RecordStore + ".controller.lock"
	if err := os.WriteFile(replacement, []byte("another owner's lease"), 0600); err != nil {
		t.Fatal(err)
	}
	release()
	got, err := os.ReadFile(replacement)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "another owner's lease" {
		t.Fatalf("release changed replacement lease content: %q", got)
	}
	if _, err := os.Stat(filepath.Join(moved, "ledger.controller.lock")); err != nil {
		t.Fatalf("renamed lease should remain as a conservative crash remnant: %v", err)
	}
}
