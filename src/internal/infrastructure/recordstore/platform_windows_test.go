//go:build windows

package recordstore

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// This opt-in diagnostic retains its real fixtures outside the test temp tree.
// It observes normal release; it never reclaims or deletes an error-state leaf.
func TestNativeHistoryLeaseReleaseWithRetainedReaderDiagnostic(t *testing.T) {
	base := os.Getenv("MARKITECT_LEASE_DIAGNOSTIC_DIR")
	if base == "" {
		t.Skip("set an external fixture directory for the bounded native diagnostic")
	}
	if !filepath.IsAbs(base) {
		t.Fatal("diagnostic directory must be absolute")
	}
	for _, profile := range []string{"no-reader", "rooted-share-all", "plain-no-share-delete"} {
		t.Run(profile, func(t *testing.T) {
			root := filepath.Join(base, profile)
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			s, err := Open(root)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			owner, err := s.Exclusive(Briefing, "accepted", ".briefings.lock", []byte("diagnostic-owner"), 0, nil)
			if err != nil {
				t.Fatalf("initial actual acquisition: %v", err)
			}
			originalID := nativeTestObjectID(t, owner.file)
			var reader *os.File
			if profile == "rooted-share-all" {
				reader, err = owner.directory.root.Open(owner.leaf)
			} else if profile == "plain-no-share-delete" {
				reader, err = os.Open(filepath.Join(root, filepath.FromSlash(RootPath), "briefing", "accepted", owner.leaf))
			}
			if err != nil {
				t.Fatalf("actual reader open: %v", err)
			}
			observeHeld := func(phase string) {
				if reader == nil {
					return
				}
				if _, err := reader.Seek(0, 0); err != nil {
					t.Fatal(err)
				}
				wire, err := io.ReadAll(reader)
				id := nativeTestObjectID(t, reader)
				t.Logf("%s held-reader ID=%s bytes=%q err=%v", phase, id, wire, err)
				if err != nil || id != originalID || string(wire) != "diagnostic-owner" {
					t.Fatal("held original identity/bytes changed")
				}
			}
			observeHeld("before release")
			closeErr := owner.Close()
			t.Logf("normal owner Close ID=%s err=%v", originalID, closeErr)
			observeHeld("after release")
			relative := RootPath + "/briefing/accepted/.briefings.lock"
			observeName := func(phase string) {
				f, err := s.root.Open(relative)
				if err != nil {
					t.Logf("%s named open path=%s err=%v", phase, relative, err)
					return
				}
				wire, readErr := io.ReadAll(f)
				t.Logf("%s named ID=%s bytes=%q readErr=%v", phase, nativeTestObjectID(t, f), wire, readErr)
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
			}
			observeName("after release")
			request := func(phase string) *Lease {
				next, err := s.Exclusive(Briefing, "accepted", ".briefings.lock", []byte("next-owner"), 0, nil)
				var busy *LeaseBusy
				var errno syscall.Errno
				errors.As(err, &errno)
				t.Logf("%s actual Exclusive err=%v errno=%d busy=%t exists=%t published=%t unknown=%t", phase, err, errno, errors.As(err, &busy), errors.Is(err, fs.ErrExist), WasPublished(err), OutcomeUnknown(err))
				return next
			}
			next := request("immediate")
			if reader != nil {
				if err := reader.Close(); err != nil {
					t.Fatal(err)
				}
				t.Log("normal retained reader Close completed")
				observeName("after reader close")
				if next == nil {
					next = request("after reader close")
				}
			}
			if next != nil {
				t.Logf("normal next-owner Close err=%v", next.Close())
			}
			observeName("final retained state")
		})
	}
}

// This adversarial setup adds independent handles without closing any held
// production object. It has no rename fallback and never retries a failure.
func movePublishingParentFixture(t *testing.T, s *Store, from, to string) {
	t.Helper()
	if filepath.Dir(from) != filepath.Dir(to) {
		t.Fatal("parent setup must stay within its pinned kind")
	}
	relative, err := filepath.Rel(s.absolute, from)
	if err != nil || strings.HasPrefix(relative, "..") {
		t.Fatal("parent setup escaped repository")
	}
	before, err := s.root.Lstat(filepath.ToSlash(relative))
	if err != nil || !before.IsDir() || platformIsReparse(before) {
		t.Fatalf("unsafe parent setup source: %v", err)
	}
	if _, err := os.Lstat(to); !os.IsNotExist(err) {
		t.Fatalf("parent setup destination is not definitely absent: %v", err)
	}
	kind, err := s.root.Open(filepath.ToSlash(filepath.Dir(relative)))
	if err != nil {
		t.Fatal(err)
	}
	defer kind.Close()
	destination, err := platformWritableParent(kind)
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	a, u, err := attributes(destination, filepath.Base(from))
	if err != nil {
		t.Fatal(err)
	}
	if err := ntOpen.Find(); err != nil {
		t.Fatal(err)
	}
	var handle syscall.Handle
	var iosb [2]uintptr
	// READ_ATTRIBUTES is required for the actual held-object ID observation.
	// It changes only this independent adversarial fixture handle.
	status, _, _ := ntOpen.Call(uintptr(unsafe.Pointer(&handle)), deleteAccess|synchronize|0x80, uintptr(unsafe.Pointer(&a)), uintptr(unsafe.Pointer(&iosb)), shareAll, directoryFile|synchronousIO|openReparse)
	runtime.KeepAlive(a.ObjectName)
	runtime.KeepAlive(u)
	t.Logf("native parent setup open=0x%08x source=%s destination-parent=%s", uint32(status), filepath.Base(from), nativeTestObjectID(t, destination))
	if uint32(status) != 0 {
		t.Fatalf("actual parent setup source open failed, all production handles held: NTSTATUS0x%08x %v", uint32(status), nativeError(uint32(status)))
	}
	source := os.NewFile(uintptr(handle), from)
	defer source.Close()
	held, err := source.Stat()
	if err != nil || !os.SameFile(before, held) || !held.IsDir() || platformIsReparse(held) {
		t.Fatalf("parent setup source identity changed: %v", err)
	}
	sourceID := nativeTestObjectID(t, source)
	t.Logf("native parent setup source-ID=%s", sourceID)
	logNativeParentFixtureChildren(t, s, relative, "before native move")
	u, err = syscall.UTF16FromString(filepath.Base(to))
	if err != nil {
		t.Fatal(err)
	}
	u = u[:len(u)-1]
	data := make([]byte, 20+len(u)*2) // class65 flags0: no destination overwrite
	binary.LittleEndian.PutUint64(data[8:16], uint64(destination.Fd()))
	binary.LittleEndian.PutUint32(data[16:20], uint32(len(u)*2))
	for i, c := range u {
		binary.LittleEndian.PutUint16(data[20+i*2:], c)
	}
	if err := ntRename.Find(); err != nil {
		t.Fatal(err)
	}
	status, _, _ = ntRename.Call(source.Fd(), uintptr(unsafe.Pointer(&iosb)), uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 65)
	runtime.KeepAlive(data)
	t.Logf("native parent setup rename=0x%08x flags=0 original-ID=%s held-ID=%s from=%s to=%s", uint32(status), sourceID, nativeTestObjectID(t, source), filepath.Base(from), filepath.Base(to))
	if uint32(status) != 0 {
		logNativeParentFixtureChildren(t, s, relative, "after failed native move")
		stillNamed, nameErr := os.Stat(from)
		if nameErr != nil || !os.SameFile(held, stillNamed) {
			t.Errorf("failed setup changed original parent identity: %v", nameErr)
		}
		if _, nameErr := os.Lstat(to); !os.IsNotExist(nameErr) {
			t.Errorf("failed setup created moved name: %v", nameErr)
		}
		t.Fatalf("actual parent setup rename failed, all production handles held: NTSTATUS0x%08x %v", uint32(status), nativeError(uint32(status)))
	}
	moved, err := os.Stat(to)
	if err != nil || !os.SameFile(held, moved) {
		t.Fatalf("actual moved parent identity differs: %v", err)
	}
	if _, err := os.Lstat(from); !os.IsNotExist(err) {
		t.Fatalf("old parent name still present after native setup: %v", err)
	}
}

func logNativeParentFixtureChildren(t *testing.T, s *Store, relative, phase string) {
	t.Helper()
	directory, err := s.root.Open(filepath.ToSlash(relative))
	if err != nil {
		t.Fatalf("parent setup observation failed: %v", err)
	}
	entries, err := directory.ReadDir(-1)
	closeErr := directory.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("parent setup directory observation failed: %v %v", err, closeErr)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			t.Logf("%s child=%s directory", phase, entry.Name())
			continue
		}
		file, err := s.root.Open(filepath.ToSlash(filepath.Join(relative, entry.Name())))
		if err != nil {
			t.Fatalf("parent setup child observation failed: %v", err)
		}
		id := nativeTestObjectID(t, file)
		data, err := io.ReadAll(io.LimitReader(file, 4097))
		closeErr := file.Close()
		if err != nil || closeErr != nil || len(data) > 4096 {
			t.Fatalf("parent setup child observation failed: %v %v", err, closeErr)
		}
		t.Logf("%s child=%s ID=%s bytes=%q", phase, entry.Name(), id, data)
	}
}

func assertPublishingParentReplacementFixture(t *testing.T, s *Store, replacement, moved string) {
	t.Helper()
	old, oldErr := os.Stat(moved)
	current, currentErr := os.Stat(replacement)
	if oldErr != nil || currentErr != nil || !old.IsDir() || !current.IsDir() || os.SameFile(old, current) {
		t.Fatalf("actual parent replacement not established: %v %v", oldErr, currentErr)
	}
	for label, name := range map[string]string{"moved-original": moved, "replacement": replacement} {
		relative, err := filepath.Rel(s.absolute, name)
		if err != nil {
			t.Fatal(err)
		}
		file, err := s.root.Open(filepath.ToSlash(relative))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("actual parent %s ID=%s", label, nativeTestObjectID(t, file))
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNativeReplacementWithHeldReadersPreservesObjectsAndOutcomes(t *testing.T) {
	for _, rooted := range []bool{true, false} {
		t.Run(map[bool]string{true: "rooted-share-delete", false: "plain-no-share-delete"}[rooted], func(t *testing.T) {
			s, root := openFixture(t)
			n := fixtureName()
			if err := s.Create(n, []byte("old")); err != nil {
				t.Fatal(err)
			}
			if err := s.Create(n, []byte("immutable collision")); !errors.Is(err, os.ErrExist) {
				t.Fatalf("Create0 overwrote: %v", err)
			}
			prior, err := s.Read(n, 32)
			if err != nil || string(prior.Bytes) != "old" {
				t.Fatal(prior, err)
			}
			p, _ := n.Path()
			var held *os.File
			if rooted {
				d, err := s.openDirectory(filepath.ToSlash(filepath.Dir(p)), false, true)
				if err != nil {
					t.Fatal(err)
				}
				defer d.close()
				held, err = d.root.Open(filepath.Base(p))
				if err != nil {
					t.Fatal(err)
				}
			} else {
				held, err = os.Open(filepath.Join(root, filepath.FromSlash(p)))
				if err != nil {
					t.Fatal(err)
				}
			}
			defer held.Close()
			oldID := nativeTestObjectID(t, held)
			err = s.Replace(n, prior, []byte("new"), nil)
			if rooted && err != nil {
				t.Fatalf("rooted held reader blocked Replace3: %v", err)
			}
			if !rooted && !OutcomeUnknown(err) {
				t.Fatalf("plain-reader native failure lost Unknown qualification: %v", err)
			}
			if !rooted {
				var errno syscall.Errno
				if !errors.As(err, &errno) || errno != syscall.Errno(32) {
					t.Fatalf("measured plain-reader sharing failure differs: %v", err)
				}
			}
			if _, err := held.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			wire, err := io.ReadAll(held)
			if err != nil || string(wire) != "old" || nativeTestObjectID(t, held) != oldID {
				t.Fatalf("held original changed: %q %v", wire, err)
			}
			named, err := s.root.Open(filepath.FromSlash(p))
			if err != nil {
				t.Fatal(err)
			}
			defer named.Close()
			newID := nativeTestObjectID(t, named)
			wire, err = io.ReadAll(named)
			want := "old"
			if rooted {
				want = "new"
			}
			if err != nil || string(wire) != want || (newID != oldID) != rooted {
				t.Fatalf("named object: %q old=%s named=%s err=%v", wire, oldID, newID, err)
			}
			if err := held.Close(); err != nil {
				t.Fatal(err)
			}
			if wire, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p))); err != nil || string(wire) != want {
				t.Fatalf("closing holder changed target: %q %v", wire, err)
			}
			entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(filepath.Dir(p))))
			if err != nil {
				t.Fatal(err)
			}
			pending := 0
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), ".pending-") {
					pending++
					data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(filepath.Dir(p)), e.Name()))
					if err != nil || string(data) != "new" {
						t.Fatalf("pending source changed: %q %v", data, err)
					}
				}
			}
			if rooted && pending != 0 || !rooted && pending != 1 {
				t.Fatalf("wrong retained sources: %d", pending)
			}
			t.Logf("held original=%s named=%s named-bytes=%s pending=%d", oldID, newID, want, pending)
		})
	}
}

func nativeTestObjectID(t *testing.T, f *os.File) string {
	t.Helper()
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%08x:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow)
}

func TestNativeRelativeDirectoryCreation(t *testing.T) {
	s, root := openFixture(t)
	f, err := s.root.Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := platformWritableParent(f)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := platformMkdir(s.root, p, "child"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("actual entries: %v", entries)
	if _, err := s.root.Lstat("child"); err != nil {
		t.Fatal(err)
	}
}
func TestExclusiveLeaseRemainsReadableByExistingProcessAudit(t *testing.T) {
	s, root := openFixture(t)
	data := []byte("original Runtime owner bytes")
	lease, err := s.Exclusive(Run, "", ".write.lock", data, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(RootPath), "run", ".write.lock")
	wire, err := os.ReadFile(p)
	if err != nil || string(wire) != string(data) {
		t.Fatalf("plain audit read blocked: %q %v", wire, err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("lease retained after release")
	}
}
func TestActualWindowsJunctionRefusesWithoutTargetMutation(t *testing.T) {
	s, root := openFixture(t)
	target := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".markitect"), 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, ".markitect", "records")
	if output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Skipf("actual junction creation unavailable: %v %s", err, output)
	}
	if err := s.Create(fixtureName(), []byte("forbidden")); err == nil {
		t.Fatal("junction accepted")
	}
	if entries, err := os.ReadDir(target); err != nil || len(entries) != 0 {
		t.Fatalf("junction target mutated: %v %v", entries, err)
	}
}
func TestWrongWindowsCapabilityRefusesBeforeControlCreation(t *testing.T) {
	for _, control := range []string{"abi", "version", "api", "filesystem", "remote", "device", "query", "query-length", "rights", "flush"} {
		t.Run(control, func(t *testing.T) {
			abi, version, volume, apis, parent, flush := writerABI, writerVersion, writerVolume, findWriterAPIs, openWriterParent, writerFlush
			defer func() {
				writerABI, writerVersion, writerVolume, findWriterAPIs, openWriterParent, writerFlush = abi, version, volume, apis, parent, flush
			}()
			s, root := openFixture(t)
			switch control {
			case "abi":
				writerABI = func() bool { return false }
			case "version":
				writerVersion = func(v *versionInfo) uint32 { v.Major = 10; v.Build = 26101; return 0 }
			case "api":
				findWriterAPIs = func() error { return errors.New("required API unavailable") }
				openWriterParent = func(*os.File) (*os.File, error) {
					t.Fatal("writer parent opened before required API admission")
					return nil, syscall.EIO
				}
			case "rights":
				openWriterParent = func(*os.File) (*os.File, error) { return nil, syscall.EACCES }
			case "flush":
				writerFlush = func(*os.File) (uintptr, error) { return 0, syscall.EIO }
			default:
				writerVolume = func(f *os.File, data []byte, class uint32) (uintptr, uint32) {
					length, status := volume(f, data, class)
					if control == "query" {
						return 0, 0xC0000022
					}
					if control == "query-length" {
						return 1, 0
					}
					if class == 5 && control == "filesystem" {
						binary.LittleEndian.PutUint16(data[12:], uint16('X'))
					}
					if class == 4 && control == "remote" {
						binary.LittleEndian.PutUint32(data[4:], 0x10)
					}
					if class == 4 && control == "device" {
						binary.LittleEndian.PutUint32(data[:4], 0)
					}
					return length, status
				}
			}
			if _, err := s.Exclusive(Run, "", ".write.lock", nil, 0, nil); err == nil {
				t.Fatal("unsupported writer acquired lease")
			}
			if err := s.Create(fixtureName(), []byte("record")); err == nil {
				t.Fatal("unsupported writer created record")
			}
			if _, err := os.Stat(filepath.Join(root, ".markitect")); !os.IsNotExist(err) {
				t.Fatalf("capability refusal created control/container: %v", err)
			}
			if entries, err := s.List(Run, "", ""); err != nil || len(entries) != 0 {
				t.Fatalf("read-only route changed: %v %v", entries, err)
			}
		})
	}
}

func TestNativeStatusErrorPreservesExactDeletePendingIdentity(t *testing.T) {
	if err := ntDOS.Find(); err != nil {
		t.Fatal(err)
	}
	dos, _, _ := ntDOS.Call(uintptr(statusDeletePending))
	cause := syscall.Errno(dos)
	err := nativeError(statusDeletePending)
	var native *nativeStatusError
	if !errors.As(err, &native) || native.status != statusDeletePending {
		t.Fatalf("raw delete-pending identity lost: %#v", err)
	}
	want := fmt.Sprintf("NTSTATUS 0x%08x: %v", statusDeletePending, cause)
	if err.Error() != want || !errors.Is(err, cause) {
		t.Fatalf("status rendering or mapped cause changed: %q want %q cause=%v", err, want, cause)
	}
	if !platformLeaseCreatePending(err) {
		t.Fatal("exact native delete-pending status was not classified")
	}
	collision := nativeError(statusCollision)
	if !errors.Is(collision, fs.ErrExist) || platformLeaseCreatePending(collision) {
		t.Fatalf("ordinary native collision mapping changed: %v", collision)
	}
	for _, unrelated := range []error{cause, syscall.Errno(5), errors.New("NTSTATUS 0xc0000056: Access is denied."), nativeError(0xC0000022), nativeError(0xC0000043)} {
		if platformLeaseCreatePending(unrelated) {
			t.Fatalf("mapped errno, text, access denial, sharing violation, or unrelated status was retried: %v", unrelated)
		}
	}
}

type pendingAttempt struct {
	lease *Lease
	err   error
}

func startDeletePendingLease(t *testing.T, kind Kind, id, leaf string) (*Store, string, func(), func() error) {
	t.Helper()
	owner, root := openFixture(t)
	contender, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = contender.Close() })
	marked := make(chan struct{})
	releaseChannel := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseChannel) }) }
	owner.hooks.afterDeleteMark = func() {
		close(marked)
		<-releaseChannel
	}
	lease, err := owner.Exclusive(kind, id, leaf, []byte("native-pending-owner"), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	closeDone := make(chan error, 1)
	closeStarted := false
	joined := false
	var closeErr error
	join := func() error {
		if !closeStarted {
			return errors.New("normal owner Close was not started")
		}
		if joined {
			return closeErr
		}
		closeErr = <-closeDone
		joined = true
		return closeErr
	}
	// This cleanup is installed before starting the owner goroutine. It always
	// releases the mark barrier and performs a real receive before store cleanup.
	t.Cleanup(func() {
		release()
		if closeStarted {
			if err := join(); err != nil {
				t.Errorf("owner cleanup/join failed: %v", err)
			}
			return
		}
		owner.hooks.afterDeleteMark = nil
		if err := lease.Close(); err != nil {
			t.Errorf("owner lease setup cleanup: %v", err)
		}
	})
	originalID := nativeTestObjectID(t, lease.file)
	closeStarted = true
	go func() { closeDone <- lease.Close() }()
	select {
	case <-marked:
	case <-time.After(5 * time.Second):
		t.Fatal("normal owner release did not reach successful delete-mark checkpoint")
	}
	return contender, originalID, release, join
}

func startLeaseAttempt(t *testing.T, contender *Store, kind Kind, id, leaf string, data []byte, wait time.Duration, dead func([]byte) (bool, error), release func(), join func() error) (<-chan pendingAttempt, *bool, **Lease) {
	t.Helper()
	result := make(chan pendingAttempt, 1)
	taken := new(bool)
	var owned *Lease
	// Register cleanup before starting the contender. Watchdog failures in the
	// test body still release and join the real goroutine before stores close.
	t.Cleanup(func() {
		release()
		if err := join(); err != nil {
			t.Errorf("owner cleanup/join failed: %v", err)
		}
		if !*taken {
			attempt := <-result
			if attempt.lease != nil {
				if err := attempt.lease.Close(); err != nil {
					t.Errorf("unexpected lease cleanup: %v", err)
				}
			}
		}
		if owned != nil {
			if err := owned.Close(); err != nil {
				t.Errorf("contender lease cleanup: %v", err)
			}
		}
	})
	go func() {
		lease, err := contender.Exclusive(kind, id, leaf, data, wait, dead)
		result <- pendingAttempt{lease: lease, err: err}
	}()
	return result, taken, &owned
}

func captureCreateResult(channel chan<- error, err error) {
	select {
	case channel <- err:
	default:
	}
}

func receiveCreateResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("contender did not report an actual native create result")
		return nil
	}
}

func receiveLeaseAttempt(t *testing.T, result <-chan pendingAttempt) pendingAttempt {
	t.Helper()
	select {
	case attempt := <-result:
		return attempt
	case <-time.After(5 * time.Second):
		t.Fatal("contender attempt did not return")
		return pendingAttempt{}
	}
}

func TestExclusiveDeletePendingZeroWaitReturnsActualStatus(t *testing.T) {
	contender, _, release, join := startDeletePendingLease(t, Briefing, "accepted", ".briefings.lock")
	actual := make(chan error, 8)
	contender.hooks.afterLeaseCreateResult = func(err error) { captureCreateResult(actual, err) }
	result, taken, owned := startLeaseAttempt(t, contender, Briefing, "accepted", ".briefings.lock", []byte("contender"), 0, nil, release, join)
	got := receiveLeaseAttempt(t, result)
	*taken = true
	*owned = got.lease
	observed := receiveCreateResult(t, actual)
	var busy *LeaseBusy
	if !platformLeaseCreatePending(observed) || !platformLeaseCreatePending(got.err) || errors.Is(got.err, fs.ErrExist) || errors.As(got.err, &busy) {
		t.Fatalf("zero wait did not preserve actual typed status: observed=%v returned=%v", observed, got.err)
	}
	release()
	if err := join(); err != nil {
		t.Fatalf("normal owner Close: %v", err)
	}
}

func TestExclusiveDeletePendingPositiveWaitRetriesAfterJoinedRelease(t *testing.T) {
	contender, originalID, release, join := startDeletePendingLease(t, Briefing, "accepted", ".briefings.lock")
	pending := make(chan error, 1)
	created := make(chan struct{}, 1)
	contender.hooks.afterLeaseCreateResult = func(err error) {
		if err == nil {
			select {
			case created <- struct{}{}:
			default:
			}
			return
		}
		captureCreateResult(pending, err)
	}
	result, taken, owned := startLeaseAttempt(t, contender, Briefing, "accepted", ".briefings.lock", []byte("contender-after-release"), 2*time.Second, nil, release, join)
	first := receiveCreateResult(t, pending)
	var busy *LeaseBusy
	if !platformLeaseCreatePending(first) || errors.Is(first, fs.ErrExist) || errors.As(first, &busy) {
		t.Fatalf("expected distinct real delete-pending before owner release, got %v", first)
	}
	release()
	if err := join(); err != nil {
		t.Fatalf("normal owner Close: %v", err)
	}
	select {
	case <-created:
	case <-time.After(5 * time.Second):
		t.Fatal("waiter did not observe successful create after joined release")
	}
	got := receiveLeaseAttempt(t, result)
	*taken = true
	*owned = got.lease
	if got.err != nil || got.lease == nil {
		t.Fatalf("positive waiter failed after owner release: lease=%v err=%v", got.lease, got.err)
	}
	if err := got.lease.Check(); err != nil {
		t.Fatalf("returned lease check: %v", err)
	}
	if nextID := nativeTestObjectID(t, got.lease.file); nextID == originalID {
		t.Fatalf("fresh successful lease reused released physical identity %s", nextID)
	}
}

func TestExclusiveDeletePendingExpiresAsOriginalStatus(t *testing.T) {
	contender, _, release, join := startDeletePendingLease(t, Briefing, "accepted", ".briefings.lock")
	actual := make(chan error, 32)
	observedCount := 0
	contender.hooks.afterLeaseCreateResult = func(err error) {
		observedCount++
		captureCreateResult(actual, err)
	}
	wait := 80 * time.Millisecond
	started := time.Now()
	result, taken, owned := startLeaseAttempt(t, contender, Briefing, "accepted", ".briefings.lock", []byte("expired-contender"), wait, nil, release, join)
	got := receiveLeaseAttempt(t, result)
	*taken = true
	*owned = got.lease
	first := receiveCreateResult(t, actual)
	var busy *LeaseBusy
	if !platformLeaseCreatePending(first) || !platformLeaseCreatePending(got.err) || errors.Is(got.err, fs.ErrExist) || errors.As(got.err, &busy) || observedCount < 1 || time.Since(started) < wait {
		t.Fatalf("expired waiter did not preserve retried native status: first=%v returned=%v attempts=%d elapsed=%s", first, got.err, observedCount, time.Since(started))
	}
	release()
	if err := join(); err != nil {
		t.Fatalf("normal owner Close: %v", err)
	}
}

func TestExclusiveDeletePendingDoesNotInvokeRunDeadOwner(t *testing.T) {
	contender, _, release, join := startDeletePendingLease(t, Run, "", ".write.lock")
	actual := make(chan error, 8)
	contender.hooks.afterLeaseCreateResult = func(err error) { captureCreateResult(actual, err) }
	deadCalled := false
	result, taken, owned := startLeaseAttempt(t, contender, Run, "", ".write.lock", []byte("contender"), 0, func([]byte) (bool, error) {
		deadCalled = true
		return true, nil
	}, release, join)
	got := receiveLeaseAttempt(t, result)
	*taken = true
	*owned = got.lease
	observed := receiveCreateResult(t, actual)
	if deadCalled || !platformLeaseCreatePending(observed) || !platformLeaseCreatePending(got.err) {
		t.Fatalf("pending status entered dead-owner recovery: observed=%v err=%v deadCalled=%t", observed, got.err, deadCalled)
	}
	release()
	if err := join(); err != nil {
		t.Fatalf("normal owner Close: %v", err)
	}
}
