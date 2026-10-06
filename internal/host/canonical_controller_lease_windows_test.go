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
	var junctionErr error
	var renameErr error
	release, err := acquireCanonicalControllerLeaseWithHook(cfg, func() error {
		if renameErr = os.Rename(parent, moved); renameErr != nil {
			return renameErr
		}
		junctionErr = makeWindowsJunction(parent, outside)
		return junctionErr
	})
	if renameErr != nil {
		t.Skipf("directory rename unavailable while the pinned lease parent is open: %v", renameErr)
	}
	if junctionErr != nil {
		t.Skipf("directory junction unavailable: %v", junctionErr)
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
	if err := os.Remove(parent); err != nil {
		t.Fatalf("remove test junction: %v", err)
	}
}
