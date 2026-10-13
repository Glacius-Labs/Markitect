package recordstore

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepublicationAndPublishedFailureOutcomes(t *testing.T) {
	for _, published := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "after"}[published], func(t *testing.T) {
			s, _ := openFixture(t)
			failure := errors.New("injected failure")
			if published {
				s.hooks.directorySync = func() error { return failure }
			} else {
				s.hooks.fileSync = func() error { return failure }
			}
			err := s.Create(fixtureName(), []byte("immutable"))
			if !errors.Is(err, failure) || WasPublished(err) != published {
				t.Fatalf("outcome %v", err)
			}
			f, readErr := s.Read(fixtureName(), 20)
			if readErr != nil || f.Exists != published {
				t.Fatalf("observed %v %v", f, readErr)
			}
			if published && string(f.Bytes) != "immutable" {
				t.Fatal("published evidence rolled back")
			}
		})
	}
}
func TestNamedParentReplacementNeverTargetsReplacement(t *testing.T) {
	s, root := openFixture(t)
	n := fixtureName()
	if err := s.Ensure(Run, n.ID, ""); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(RootPath), "run", n.ID)
	moved := p + "-moved"
	s.hooks.beforePublish = func() error {
		if err := os.Rename(p, moved); err != nil {
			t.Skipf("actual parent substitution unavailable: %v", err)
		}
		if err := os.Mkdir(p, 0700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(p, "plan.json"), []byte("attacker"), 0600)
	}
	err := s.Create(n, []byte("safe"))
	if err == nil {
		t.Fatal("substitution succeeded as ordinary publication")
	}
	actual, e := os.ReadFile(filepath.Join(p, "plan.json"))
	if e != nil || !bytes.Equal(actual, []byte("attacker")) {
		t.Fatalf("replacement was targeted: %q %v", actual, e)
	}
}
func TestDomainGuardUnderPublicationKeepsPriorBytes(t *testing.T) {
	s, _ := openFixture(t)
	n := fixtureName()
	if err := s.Create(n, []byte("old")); err != nil {
		t.Fatal(err)
	}
	before, _ := s.Read(n, 10)
	called := false
	err := s.Replace(n, before, []byte("new"), func() error { called = true; return errors.New("source authority changed") })
	if err == nil || !called {
		t.Fatal("domain gate bypassed")
	}
	after, _ := s.Read(n, 10)
	if string(after.Bytes) != "old" {
		t.Fatal("guard failure changed record")
	}
}
