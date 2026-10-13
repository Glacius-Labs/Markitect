package recordstore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestKindWideExclusiveLeaseAndReplacementRelease(t *testing.T) {
	s, root := openFixture(t)
	lease, err := s.Exclusive(Run, "", ".write.lock", []byte("owner"), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Exclusive(Run, "", ".write.lock", nil, 0, nil); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("second writer not excluded: %v", err)
	}
	p := filepath.Join(root, filepath.FromSlash(RootPath), "run", ".write.lock")
	if err = os.Rename(p, p+".old"); err != nil {
		t.Skipf("lease replacement unavailable: %v", err)
	}
	if err = os.WriteFile(p, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = lease.Close(); err == nil {
		t.Fatal("lease replacement not reported")
	}
	wire, _ := os.ReadFile(p)
	if string(wire) != "replacement" {
		t.Fatal("release deleted another lease")
	}
}
func TestKernelLeavesEstablishedAndMissingReadNeverHeals(t *testing.T) {
	s, root := openFixture(t)
	id := "session"
	if _, err := s.Kernel(id, "session.lock"); err == nil {
		t.Fatal("missing lease accepted")
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect")); !os.IsNotExist(err) {
		t.Fatal("read created metadata")
	}
	for _, leaf := range []string{"session.lock", "manager-runs.lock"} {
		if err := s.EstablishKernelLeaf(id, leaf); err != nil {
			t.Fatal(err)
		}
	}
	lease, err := s.Kernel(id, "session.lock")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Kernel(id, "session.lock"); err == nil {
		t.Fatal("kernel lease contention accepted")
	}
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(RootPath), "brownfield", id, "manager-runs.lock")
	if err = os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Kernel(id, "session.lock"); err == nil {
		t.Fatal("missing companion lease accepted")
	}
	if _, err = os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("read healed missing lease")
	}
}
