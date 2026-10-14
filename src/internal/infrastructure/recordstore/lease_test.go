package recordstore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLeaseBusyOnlyWrapsTerminalLeaseLeafCollision(t *testing.T) {
	s, _ := openFixture(t)
	lease, err := s.Exclusive(Briefing, "accepted", ".briefings.lock", nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	for _, wait := range []time.Duration{0, 20 * time.Millisecond} {
		start := time.Now()
		_, err := s.Exclusive(Briefing, "accepted", ".briefings.lock", nil, wait, nil)
		var busy *LeaseBusy
		if !errors.As(err, &busy) || !errors.Is(err, fs.ErrExist) || time.Since(start) < wait {
			t.Fatalf("actual terminal lease collision: %v wait=%s", err, wait)
		}
	}
	other, _ := openFixture(t)
	other.hooks.beforeMkdir = func() error { return fs.ErrExist }
	_, err = other.Exclusive(Briefing, "accepted", ".briefings.lock", nil, 0, nil)
	var busy *LeaseBusy
	if !errors.Is(err, fs.ErrExist) || errors.As(err, &busy) {
		t.Fatalf("ancestor error mislabeled as held leaf: %v", err)
	}
}

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
func TestFiniteLeaseMechanismsRejectBeforeCreatingAnything(t *testing.T) {
	for _, call := range []func(*Store) error{
		func(s *Store) error {
			_, err := s.Exclusive(Brownfield, "session", "session.lock", nil, 0, nil)
			return err
		},
		func(s *Store) error {
			_, err := s.Exclusive(Briefing, "accepted", ".briefings.lock", nil, 0, func([]byte) (bool, error) { return true, nil })
			return err
		},
		func(s *Store) error {
			_, err := s.Exclusive(Briefing, "accepted", ".briefings.lock", nil, 11*time.Second, nil)
			return err
		},
	} {
		s, root := openFixture(t)
		if err := call(s); err == nil {
			t.Fatal("invalid lease mechanism accepted")
		}
		if _, err := os.Stat(filepath.Join(root, ".markitect")); !os.IsNotExist(err) {
			t.Fatalf("invalid policy created metadata: %v", err)
		}
	}
}
func TestKernelLeafSyncFailurePreservesCreatedIncompleteControl(t *testing.T) {
	s, root := openFixture(t)
	s.hooks.fileSync = func() error { return errors.New("injected lease sync failure") }
	err := s.EstablishKernelLeaf("session", "session.lock")
	if !WasPublished(err) {
		t.Fatalf("missing created outcome: %v", err)
	}
	p := filepath.Join(root, filepath.FromSlash(RootPath), "brownfield", "session", "session.lock")
	if i, err := os.Stat(p); err != nil || i.Size() != 0 {
		t.Fatalf("created leaf lost: %v", err)
	}
	if _, err := s.Kernel("session", "session.lock"); err == nil {
		t.Fatal("incomplete dual lease accepted")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(p), "manager-runs.lock")); !os.IsNotExist(err) {
		t.Fatal("read healed incomplete control")
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
