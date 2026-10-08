//go:build linux

package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalControllerLeaseRejectsParentSwapBeforeCreate(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "record-store")
	moved := filepath.Join(root, "record-store-original")
	outside := filepath.Join(root, "outside")
	for _, dir := range []string{parent, outside} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	cfg := CanonicalControllerConfig{RecordStore: filepath.Join(parent, "ledger")}
	var renameErr, symlinkErr error
	release, err := acquireCanonicalControllerLeaseWithHook(cfg, func() error {
		if renameErr = os.Rename(parent, moved); renameErr != nil {
			return renameErr
		}
		symlinkErr = os.Symlink(outside, parent)
		return symlinkErr
	})
	if renameErr != nil {
		t.Fatal(renameErr)
	}
	if symlinkErr != nil {
		t.Skipf("directory symlink unavailable: %v", symlinkErr)
	}
	if err == nil {
		release()
		t.Fatal("lease acquisition succeeded after its parent path was replaced")
	}
	for _, path := range []string{
		filepath.Join(outside, "ledger.controller.lock"),
		filepath.Join(moved, "ledger.controller.lock"),
	} {
		if _, statErr := os.Lstat(path); !os.IsNotExist(statErr) {
			t.Fatalf("lease unexpectedly created at %s: %v", path, statErr)
		}
	}
}
