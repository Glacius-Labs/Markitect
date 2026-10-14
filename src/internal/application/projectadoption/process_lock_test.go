package projectadoption

import (
	"bufio"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const processLockHelperEnv = "MARKITECT_PROCESS_LOCK_HELPER"

func TestProcessFileLockReleasesAfterProcessKill(t *testing.T) {
	root := t.TempDir()
	store, err := recordstore.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, leaf := range []string{"session.lock", "manager-runs.lock"} {
		if err := store.EstablishKernelLeaf("process-test", leaf); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, filepath.FromSlash(recordstore.RootPath), "brownfield", "process-test", "session.lock")
	command := exec.Command(os.Args[0], "-test.run=^TestProcessFileLockHelper$")
	command.Env = append(os.Environ(), processLockHelperEnv+"="+path)
	var helperErrors strings.Builder
	command.Stderr = &helperErrors
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		ready <- line
	}()
	select {
	case line := <-ready:
		if strings.TrimSpace(line) != "LOCKED" {
			_ = command.Process.Kill()
			_ = command.Wait()
			t.Fatalf("lock subprocess did not signal acquisition: %q", line)
		}
	case <-time.After(10 * time.Second):
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatal("timed out waiting for lock subprocess")
	}
	if unexpectedUnlock, err := acquireTestRecordLease(path); err == nil {
		unexpectedUnlock()
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("competing process acquired a held lock; helper: %s", helperErrors.String())
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("killed lock subprocess unexpectedly exited successfully")
	}
	unlock, err := acquireTestRecordLease(path)
	if err != nil {
		t.Fatalf("OS lock was not released after process death: %v", err)
	}
	unlock()
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("persistent lock file should remain as a regular file: info=%v err=%v", info, err)
	}
}

func TestProcessFileLockHelper(t *testing.T) {
	path := os.Getenv(processLockHelperEnv)
	if path == "" {
		return
	}
	unlock, err := acquireTestRecordLease(path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	_, _ = fmt.Fprintln(os.Stdout, "LOCKED")
	_ = os.Stdout.Sync()
	_, _ = fmt.Fprintf(os.Stderr, "locked helper pid=%d path=%s\n", os.Getpid(), path)
	// Keep a timed wait alive: a bare select{} can make this standalone
	// helper die from Go's deadlock detector before the parent kills it.
	for {
		time.Sleep(time.Minute)
	}
}

func TestIncompleteLedgerPublicationFailsClosed(t *testing.T) {
	root := t.TempDir()
	store, err := recordstore.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Ensure(recordstore.Brownfield, "markers", ""); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, filepath.FromSlash(recordstore.RootPath), "brownfield", "markers")
	for _, name := range []string{".record.json.tmp", ".record.json.previous", ".manager-runs.json.tmp", ".manager-runs.json.previous"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("uncertain"), 0o600); err != nil {
			t.Fatal(err)
		}
		var checkErr error
		if strings.HasPrefix(name, ".manager-runs") {
			_, checkErr = store.ReadLegacyBrownfieldManagerLedger("markers", 1024)
		} else {
			checkErr = store.Incomplete(recordstore.Name{Kind: recordstore.Brownfield, ID: "markers", Role: "record.json"})
		}
		if checkErr == nil {
			t.Fatalf("publication uncertainty at %s must fail closed", name)
		}
		if data, err := os.ReadFile(path); err != nil || string(data) != "uncertain" {
			t.Fatalf("uncertain marker was changed: %q %v", data, err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFailedLedgerReplaceLeavesNoBlockingTemp(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("needs Windows semantics: replacing a file with an open handle fails")
	}
	t.Run("session", func(t *testing.T) {
		root, discovery, _, target := distillationDiscovery(t)
		session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := WriteBrownfieldSession(root, session, session.Digest); err != nil {
			t.Fatal(err)
		}
		next, err := BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: "root-pass", ManagerID: session.TargetContext.RootManagerID,
			EvidenceIDs: []string{"implementation"}, DelegationEvidenceIDs: []string{}, Purpose: "Map selected source", Review: "review-1"})
		if err != nil {
			t.Fatal(err)
		}
		dir, err := sessionDirectory(root, session.ID, false)
		if err != nil {
			t.Fatal(err)
		}
		holder, err := os.Open(filepath.Join(dir, "record.json"))
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := WriteBrownfieldSession(root, next, session.Digest)
		_ = holder.Close()
		if writeErr == nil {
			t.Skip("replace succeeded despite an open reader; failure could not be provoked")
		}
		if recordstore.OutcomeUnknown(writeErr) || recordstore.WasPublished(writeErr) {
			if _, err := LoadBrownfieldSession(root, session.ID); err == nil {
				t.Error("uncertain publication did not fail closed")
			}
			return // Never replay or retry an ambiguous publication.
		}
		if _, err := os.Lstat(filepath.Join(dir, ".record.json.tmp")); err == nil {
			t.Error("definitely unpublished failure retained an owned temp")
		}
		loaded, err := LoadBrownfieldSession(root, session.ID)
		if err != nil || loaded.Digest != session.Digest {
			t.Errorf("unpublished failure changed original domain state: %v", err)
		}
	})
	t.Run("manager-run-ledger", func(t *testing.T) {
		root := t.TempDir()
		store, err := recordstore.Open(root)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		if err := store.Ensure(recordstore.Brownfield, "failed-replace", ""); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(root, filepath.FromSlash(recordstore.RootPath), "brownfield", "failed-replace")
		ledger := ManagerRunLedger{APIVersion: managerRunVersion, SessionID: "failed-replace", Events: []ManagerRunEvent{}}
		sealManagerRunLedger(&ledger)
		if err := writeManagerRunLedger(dir, ledger); err != nil {
			t.Fatal(err)
		}
		holder, err := os.Open(filepath.Join(dir, managerRunLedgerName))
		if err != nil {
			t.Fatal(err)
		}
		writeErr := writeManagerRunLedger(dir, ledger)
		_ = holder.Close()
		if writeErr == nil {
			t.Skip("replace succeeded despite an open reader; failure could not be provoked")
		}
		if _, err := loadManagerRunLedger(root, "failed-replace"); err != nil {
			t.Errorf("unchanged manager-run ledger can no longer be loaded after a reported write failure: %v", err)
		}
		if err := writeManagerRunLedger(dir, ledger); err != nil {
			t.Errorf("retrying the same ledger write after the handle closed still fails: %v", err)
		}
	})
}

func acquireTestRecordLease(lockPath string) (func(), error) {
	marker := filepath.FromSlash(recordstore.RootPath + "/brownfield/process-test/session.lock")
	root := strings.TrimSuffix(lockPath, string(filepath.Separator)+marker)
	store, err := recordstore.Open(root)
	if err != nil {
		return nil, err
	}
	lease, err := store.Kernel("process-test", "session.lock")
	if err != nil {
		store.Close()
		return nil, err
	}
	return func() { lease.Close(); store.Close() }, nil
}
