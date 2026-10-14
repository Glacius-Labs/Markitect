package recordstore

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLeafReadConflictRequiresSafeIdentityTransition(t *testing.T) {
	for _, control := range []string{"safe-replacement", "same-inode", "missing", "directory", "oversize", "wrong-mode", "case-alias", "parent-replacement"} {
		t.Run(control, func(t *testing.T) {
			if control == "wrong-mode" && runtime.GOOS == "windows" {
				t.Skip("numeric POSIX mode validation is not a Windows DACL control")
			}
			s, root := openFixture(t)
			n := fixtureName()
			if err := s.Create(n, []byte("old!")); err != nil {
				t.Fatal(err)
			}
			p, _ := n.Path()
			target := filepath.Join(root, filepath.FromSlash(p))
			mutated := false
			s.hooks.beforeLeafOpen = func() error {
				var err error
				if control == "same-inode" {
					err = os.WriteFile(target, []byte("next"), 0600)
				} else if control == "parent-replacement" {
					parent := filepath.Dir(target)
					if err = moveRecordParentFixture(parent, parent+"-original"); err == nil {
						err = os.Mkdir(parent, 0700)
					}
					if err == nil {
						err = os.WriteFile(target, []byte("next"), 0600)
					}
				} else {
					if err = os.Rename(target, target+"-original"); err == nil {
						switch control {
						case "directory":
							err = os.Mkdir(target, 0700)
						case "oversize":
							err = os.WriteFile(target, []byte(strings.Repeat("x", 17)), 0600)
						case "wrong-mode":
							err = os.WriteFile(target, []byte("next"), 0644)
							if err == nil {
								err = os.Chmod(target, 0644)
							}
						case "case-alias":
							err = os.WriteFile(filepath.Join(filepath.Dir(target), "PLAN.JSON"), []byte("next"), 0600)
						case "safe-replacement":
							err = os.WriteFile(target, []byte("next"), 0600)
						}
					}
				}
				mutated = err == nil
				return err
			}
			file, err := s.Read(n, 16)
			if !mutated {
				t.Fatalf("actual substitution setup failed: %v", err)
			}
			if control == "safe-replacement" {
				if !errors.Is(err, ErrReadConflict) || file.Exists || len(file.Bytes) != 0 {
					t.Fatalf("unsafe mixed observation returned: %+v %v", file, err)
				}
				if data, e := os.ReadFile(target); e != nil || string(data) != "next" {
					t.Fatalf("replacement changed: %q %v", data, e)
				}
			} else if errors.Is(err, ErrReadConflict) {
				t.Fatalf("%s misclassified as safe leaf conflict: %v", control, err)
			} else if control != "same-inode" && err == nil {
				t.Fatalf("unsafe %s observation accepted", control)
			}
		})
	}
}

func TestConcurrentMissingAncestorIsObservedOnceAndFlushed(t *testing.T) {
	root := t.TempDir()
	a, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	ready := make(chan struct{}, 2)
	releaseA, releaseB, created := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var onceA, onceB sync.Once
	unblockA := func() { onceA.Do(func() { close(releaseA) }) }
	unblockB := func() { onceB.Do(func() { close(releaseB) }) }
	t.Cleanup(unblockA)
	t.Cleanup(unblockB)
	for store, release := range map[*Store]chan struct{}{a: releaseA, b: releaseB} {
		first := true
		store.hooks.beforeMkdir = func() error {
			if first {
				first = false
				ready <- struct{}{}
				<-release
			}
			return nil
		}
	}
	var once sync.Once
	a.hooks.directoryCreated = func() error { once.Do(func() { close(created) }); return nil }
	flushes := 0
	b.hooks.directorySync = func() error { flushes++; return nil }
	results := make(chan error, 2)
	go func() { results <- a.Ensure(Run, fixtureName().ID, "") }()
	go func() { results <- b.Ensure(Run, fixtureName().ID, "") }()
	for range 2 {
		select {
		case <-ready:
		case <-time.After(5 * time.Second):
			t.Fatal("both callers did not observe the missing ancestor")
		}
	}
	unblockA()
	select {
	case <-created:
	case <-time.After(5 * time.Second):
		t.Fatal("first caller did not create the rendezvous ancestor")
	}
	unblockB()
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if flushes < 2 || flushes%2 != 0 {
		t.Fatalf("collided ancestors lacked child/parent flush: %d", flushes)
	}
	lease, err := a.Exclusive(Run, "", ".write.lock", nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Exclusive(Run, "", ".write.lock", nil, 0, nil); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("observed ancestor bypassed actual lease: %v", err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAncestorCollisionNeverHealsOrClaimsAnotherCreation(t *testing.T) {
	for _, control := range []string{"flush-child", "flush-parent", "wrong-type", "case-alias", "wrong-mode"} {
		t.Run(control, func(t *testing.T) {
			if control == "wrong-mode" && runtime.GOOS == "windows" {
				t.Skip("POSIX mode constraint has no numeric Windows equivalent")
			}
			s, root := openFixture(t)
			if err := os.Mkdir(filepath.Join(root, ".markitect"), 0700); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(root, ".markitect", "records")
			mkdirs, flushes, creations := 0, 0, 0
			s.hooks.beforeMkdir = func() error {
				mkdirs++
				if mkdirs != 1 {
					return errors.New("second mkdir is forbidden after collision")
				}
				if control == "wrong-type" {
					return os.WriteFile(p, []byte("retained"), 0600)
				}
				if control == "case-alias" {
					if runtime.GOOS != "windows" {
						if err := os.Mkdir(p, 0700); err != nil {
							return err
						}
					}
					p = filepath.Join(root, ".markitect", "RECORDS")
				}
				mode := os.FileMode(0700)
				if control == "wrong-mode" {
					mode = 0755
				}
				if err := os.Mkdir(p, mode); err != nil {
					return err
				}
				if control == "wrong-mode" {
					return os.Chmod(p, mode)
				}
				return nil
			}
			s.hooks.directoryCreated = func() error { creations++; return nil }
			s.hooks.directorySync = func() error {
				flushes++
				if control == "flush-child" || control == "flush-parent" && flushes == 2 {
					return errors.New("observed ancestor flush failed")
				}
				return nil
			}
			err := s.Ensure(Run, fixtureName().ID, "")
			if err == nil || WasPublished(err) || OutcomeUnknown(err) || creations != 0 || mkdirs != 1 {
				t.Fatalf("collider claimed/healed authority: err=%v created=%d mkdirs=%d", err, creations, mkdirs)
			}
			if _, err := os.Lstat(p); err != nil {
				t.Fatalf("collided object removed: %v", err)
			}
			if control == "flush-child" && flushes != 1 || control == "flush-parent" && flushes != 2 {
				t.Fatalf("wrong flush sequence: %d", flushes)
			}
		})
	}
}

func TestLegacyBrownfieldReadRefusesLeafSubstitutionWithoutRetry(t *testing.T) {
	s, root := openFixture(t)
	if err := s.Ensure(Brownfield, "legacy", ""); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(RootPath), "brownfield", "legacy", "manager-runs.json")
	if err := os.WriteFile(p, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	s.hooks.beforeLeafOpen = func() error {
		calls++
		if err := os.Rename(p, p+"-original"); err != nil {
			return err
		}
		return os.WriteFile(p, []byte("new"), 0600)
	}
	file, err := s.ReadLegacyBrownfieldManagerLedger("legacy", 16)
	if !errors.Is(err, ErrReadConflict) || file.Exists || calls != 1 {
		t.Fatalf("legacy substituted bytes admitted/retried: %+v calls=%d err=%v", file, calls, err)
	}
	for path, want := range map[string]string{p: "new", p + "-original": "old"} {
		if data, err := os.ReadFile(path); err != nil || string(data) != want {
			t.Fatalf("legacy evidence changed: %q %v", data, err)
		}
	}
}

func TestRootSubstitutionDuringReadNeverBecomesLeafConflict(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "run", true: "legacy-brownfield"}[legacy], func(t *testing.T) {
			container := filepath.Join(t.TempDir(), "container")
			root := filepath.Join(container, "repo")
			if err := os.MkdirAll(root, 0700); err != nil {
				t.Fatal(err)
			}
			s, err := Open(root)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			n := fixtureName()
			if legacy {
				n = Name{Kind: Brownfield, ID: "legacy", Role: "record.json"}
			}
			if err := s.Create(n, []byte("old")); err != nil {
				t.Fatal(err)
			}
			p, _ := n.Path()
			if legacy {
				p = filepath.ToSlash(filepath.Join(filepath.Dir(p), "manager-runs.json"))
				if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(p)), []byte("old"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			s.hooks.beforeLeafOpen = func() error {
				if err := moveRecordParentFixture(container, container+"-original"); err != nil {
					if runtime.GOOS == "windows" {
						t.Skipf("actual ancestor/root substitution with real root handle held unavailable: %v", err)
					}
					return err
				}
				if err := os.MkdirAll(filepath.Join(root, filepath.Dir(filepath.FromSlash(p))), 0700); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(root, filepath.FromSlash(p)), []byte("new"), 0600)
			}
			var file File
			if legacy {
				file, err = s.ReadLegacyBrownfieldManagerLedger(n.ID, 16)
			} else {
				file, err = s.Read(n, 16)
			}
			if err == nil || errors.Is(err, ErrReadConflict) || file.Exists {
				t.Fatalf("root drift admitted/retryable: %+v %v", file, err)
			}
			for root, want := range map[string]string{root: "new", filepath.Join(container+"-original", "repo"): "old"} {
				if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p))); err != nil || string(data) != want {
					t.Fatalf("root substitution evidence changed: %q %v", data, err)
				}
			}
		})
	}
}

func TestObservedDirectoryRemovalNeverBecomesAbsence(t *testing.T) {
	for _, caller := range []string{"read", "legacy", "list", "cutover", "incomplete"} {
		t.Run(caller, func(t *testing.T) {
			s, root := openFixture(t)
			n := fixtureName()
			lastOpen := 5 // one old-layout parent, then four new-layout parents
			if caller == "legacy" {
				n = Name{Kind: Brownfield, ID: "legacy", Role: "record.json"}
			}
			if err := s.Create(n, []byte("retained")); err != nil {
				t.Fatal(err)
			}
			p, _ := n.Path()
			directory := filepath.Join(root, filepath.Dir(filepath.FromSlash(p)))
			leaf := filepath.Base(p)
			if caller == "legacy" {
				leaf = "manager-runs.json"
				if err := os.WriteFile(filepath.Join(directory, leaf), []byte("retained"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if caller == "incomplete" {
				lastOpen = 4
			}
			if caller == "cutover" {
				directory = filepath.Join(root, ".markitect", "drafts", "brownfield")
				leaf = "old.json"
				if err := os.MkdirAll(directory, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(directory, leaf), []byte("retained"), 0600); err != nil {
					t.Fatal(err)
				}
				lastOpen = 3
			}
			calls, moved := 0, false
			s.hooks.beforeOpen = func() error {
				calls++
				if calls == lastOpen {
					if err := moveRecordParentFixture(directory, directory+"-observed"); err != nil {
						return err
					}
					moved = true
				}
				return nil
			}
			var err error
			switch caller {
			case "read":
				_, err = s.Read(n, 32)
			case "legacy":
				_, err = s.ReadLegacyBrownfieldManagerLedger(n.ID, 32)
			case "list":
				_, err = s.List(n.Kind, n.ID, "")
			case "cutover":
				err = s.CheckCutover(Brownfield)
			case "incomplete":
				err = s.Incomplete(n)
			}
			if !moved || err == nil || isDirectoryAbsent(err) || errors.Is(err, ErrReadConflict) {
				t.Fatalf("unsafe %s became absence/retryable: moved=%t calls=%d err=%v", caller, moved, calls, err)
			}
			if data, err := os.ReadFile(filepath.Join(directory+"-observed", leaf)); err != nil || string(data) != "retained" {
				t.Fatalf("observed old bytes changed: %q %v", data, err)
			}
		})
	}
}

func TestGuardNotExistIsNotDirectoryAbsence(t *testing.T) {
	for _, caller := range []string{"read", "legacy", "list", "cutover", "incomplete"} {
		t.Run(caller, func(t *testing.T) {
			s, _ := openFixture(t)
			n := fixtureName()
			if err := s.Create(n, []byte("retained")); err != nil {
				t.Fatal(err)
			}
			called := false
			s.hooks.beforeOpen = func() error { called = true; return fs.ErrNotExist }
			var err error
			switch caller {
			case "read":
				_, err = s.Read(n, 32)
			case "legacy":
				_, err = s.ReadLegacyBrownfieldManagerLedger("legacy", 32)
			case "list":
				_, err = s.List(Run, "", "")
			case "cutover":
				err = s.CheckCutover(Run)
			case "incomplete":
				err = s.Incomplete(n)
			}
			if !called || !errors.Is(err, fs.ErrNotExist) || isDirectoryAbsent(err) {
				t.Fatalf("guard NotExist swallowed for %s: called=%t err=%v", caller, called, err)
			}
		})
	}
}

func TestSafeAbsentDirectoriesAndLeavesNeverCreateOrSync(t *testing.T) {
	s, root := openFixture(t)
	s.hooks.directorySync = func() error { t.Fatal("absent reader synchronized a directory"); return nil }
	if f, err := s.Read(fixtureName(), 32); err != nil || f.Exists {
		t.Fatal(f, err)
	}
	if f, err := s.ReadLegacyBrownfieldManagerLedger("absent", 32); err != nil || f.Exists {
		t.Fatal(f, err)
	}
	if entries, err := s.List(Run, "", ""); err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
	if err := s.Incomplete(fixtureName()); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckCutover(Run); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("absent reads created names: %v %v", entries, err)
	}
	if err := s.Ensure(Run, fixtureName().ID, ""); err != nil {
		t.Fatal(err)
	}
	if f, err := s.Read(fixtureName(), 32); err != nil || f.Exists {
		t.Fatal(f, err)
	}
}

func TestImmutableCollisionAndExactPreimage(t *testing.T) {
	s, _ := openFixture(t)
	n := fixtureName()
	if err := s.Create(n, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(n, []byte("second")); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("collision: %v", err)
	}
	before, err := s.Read(n, 10)
	if err != nil || string(before.Bytes) != "first" {
		t.Fatalf("read %q: %v", before.Bytes, err)
	}
	if err = s.Replace(n, before, []byte("next"), nil); err != nil {
		t.Fatal(err)
	}
	if err = s.Replace(n, before, []byte("stale"), nil); err == nil {
		t.Fatal("stale byte CAS accepted")
	}
	after, _ := s.Read(n, 10)
	if string(after.Bytes) != "next" {
		t.Fatalf("unexpected bytes %q", after.Bytes)
	}
}
func TestReadListAbsentNeverCreate(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	f, err := s.Read(fixtureName(), 10)
	if err != nil || f.Exists {
		t.Fatal(f, err)
	}
	entries, err := s.List(Run, "", "")
	if err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 0 {
		t.Fatalf("reader created state: %v %v", files, err)
	}
}
func TestCutoverTransportOnlyAndMixedAuthority(t *testing.T) {
	s, root := openFixture(t)
	id := fixtureName().ID
	packet := "fedcba9876543210fedcba9876543210"
	input := filepath.ToSlash(filepath.Join(".markitect", "runs", id, "external-response-"+packet+".json"))
	wire := []byte("transport only")
	writeFixture(t, root, input, wire)
	if err := s.CheckCutover(Run); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(fixtureName(), []byte("canonical")); err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(input)))
	if !bytes.Equal(actual, wire) {
		t.Fatal("input changed")
	}
	writeFixture(t, root, filepath.ToSlash(filepath.Join(".markitect", "runs", id, "plan.json")), []byte("old authority"))
	if err := s.CheckCutover(Run); err == nil {
		t.Fatal("mixed old authority was hidden by transport")
	}
	if err := s.Create(Name{Kind: Run, ID: packet, Role: "plan.json"}, nil); err == nil {
		t.Fatal("old authority did not block new write")
	}
}
func TestCutoverRejectsEmptyOldIDAndEveryOtherKind(t *testing.T) {
	for _, kind := range []Kind{Run, Exploration, Briefing, Brownfield} {
		t.Run(string(kind), func(t *testing.T) {
			s, root := openFixture(t)
			var old string
			switch kind {
			case Run:
				old = ".markitect/runs/" + fixtureName().ID
			case Exploration:
				old = ".markitect/state/explorations/old.json"
			case Briefing:
				old = ".markitect/state/briefings/history.json"
			case Brownfield:
				old = ".markitect/drafts/brownfield/old/session.json"
			}
			if kind == Run {
				if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(old)), 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				writeFixture(t, root, old, []byte("preserved"))
			}
			if err := s.CheckCutover(kind); err == nil {
				t.Fatal("legacy authority accepted")
			}
		})
	}
}
func TestClosedNames(t *testing.T) {
	for _, bad := range []Name{{Kind: "other", ID: "x", Role: "record.json"}, {Kind: Run, ID: fixtureName().ID, Role: "../escape.json"}, {Kind: Exploration, ID: "CON", Role: "record.json"}, {Kind: Run, ID: fixtureName().ID, Role: "private/log.json"}, {Kind: Brownfield, ID: "session", Role: "unknown.json"}} {
		if _, err := bad.Path(); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}
