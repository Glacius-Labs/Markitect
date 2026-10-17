package journal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsPrivateACLAndDeleteSharing(t *testing.T) {
	r := request(t)
	s := acquire(t, r)
	if err := os.Rename(filepath.Join(r.Repository, ".markitect", "state", "journal.lock"), filepath.Join(r.Repository, ".markitect", "state", "moved.lock")); err == nil {
		t.Fatal("Windows live lock permitted delete sharing")
	}
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	r = request(t)
	// Ordinary Mkdir inherits this test's broad parent DACL. The adapter must
	// refuse it rather than claiming that chmod(0700) supplied a private ACL.
	if err := os.Mkdir(r.StateDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	if got, err := New().Acquire(context.Background(), r); err == nil {
		got.Close()
		t.Fatal("inherited non-private ACL authorized")
	}
}
