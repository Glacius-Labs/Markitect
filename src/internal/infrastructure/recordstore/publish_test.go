package recordstore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
func TestTemporarySubstitutionRefusesPublicationAndRetainsReplacement(t *testing.T) {
	for _, earlyFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "final-validation", true: "cleanup"}[earlyFailure], func(t *testing.T) {
			s, root := openFixture(t)
			n := fixtureName()
			var replacement string
			s.hooks.beforePublish = func() error {
				dir := filepath.Join(root, filepath.FromSlash(RootPath), "run", n.ID)
				entries, err := os.ReadDir(dir)
				if err != nil {
					return err
				}
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), ".pending-") {
						replacement = filepath.Join(dir, entry.Name())
						if err := os.Rename(replacement, replacement+"-original"); err != nil {
							return err
						}
						if err := os.WriteFile(replacement, []byte("replacement"), 0600); err != nil {
							return err
						}
						if earlyFailure {
							return errors.New("fail before final validation")
						}
						return nil
					}
				}
				return errors.New("temporary record missing")
			}
			if err := s.Create(n, []byte("original")); err == nil || WasPublished(err) {
				t.Fatalf("substitution outcome: %v", err)
			}
			wire, err := os.ReadFile(replacement)
			if err != nil || string(wire) != "replacement" {
				t.Fatalf("replacement cleanup: %q %v", wire, err)
			}
			file, err := s.Read(n, 0)
			if err != nil || file.Exists {
				t.Fatalf("target changed before publication: %+v %v", file, err)
			}
		})
	}
}
func TestEveryCreatedAncestorFailureRetainsQualifiedOutcome(t *testing.T) {
	for failAt := 1; failAt <= 4; failAt++ {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			s, root := openFixture(t)
			calls := 0
			s.hooks.directoryCreated = func() error {
				calls++
				if calls == failAt {
					return errors.New("created ancestor observation failed")
				}
				return nil
			}
			err := s.Create(fixtureName(), []byte("record"))
			if !WasPublished(err) || calls != failAt {
				t.Fatalf("created outcome: %v calls=%d", err, calls)
			}
			parts := []string{".markitect", "records", "run", fixtureName().ID}
			if _, err := os.Stat(filepath.Join(append([]string{root}, parts[:failAt]...)...)); err != nil {
				t.Fatalf("created ancestor lost: %v", err)
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
func TestParentMoveAfterPublicationRetainsBothObjects(t *testing.T) {
	s, root := openFixture(t)
	n := fixtureName()
	p := filepath.Join(root, filepath.FromSlash(RootPath), "run", n.ID)
	s.hooks.afterPublish = func() error {
		if err := os.Rename(p, p+"-published"); err != nil {
			t.Skipf("actual post-publication parent replacement unavailable: %v", err)
		}
		if err := os.Mkdir(p, 0700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(p, "plan.json"), []byte("replacement"), 0600)
	}
	err := s.Create(n, []byte("published"))
	if !WasPublished(err) {
		t.Fatalf("lost published outcome: %v", err)
	}
	for dir, want := range map[string]string{p: "replacement", p + "-published": "published"} {
		wire, err := os.ReadFile(filepath.Join(dir, "plan.json"))
		if err != nil || string(wire) != want {
			t.Fatalf("object lost at %s: %q %v", dir, wire, err)
		}
	}
}

func TestSameByteLeafReplacementAfterPublicationRetainsBothIdentities(t *testing.T) {
	s, root := openFixture(t)
	n := fixtureName()
	relative, err := n.Path()
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	wire := []byte("published")
	s.hooks.afterPublish = func() error {
		if err := os.Rename(target, target+"-published"); err != nil {
			return err
		}
		return os.WriteFile(target, wire, 0600)
	}
	err = s.Create(n, wire)
	if !WasPublishedAt(err, relative) || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("same-byte replacement accepted: %v", err)
	}
	original, err := os.Stat(target + "-published")
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(original, replacement) {
		t.Fatal("fixture did not replace identity")
	}
	for _, p := range []string{target, target + "-published"} {
		got, err := os.ReadFile(p)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("published or replacement bytes lost at %s: %q %v", p, got, err)
		}
	}
}
