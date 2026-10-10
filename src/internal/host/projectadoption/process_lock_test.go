package projectadoption

import (
	"bufio"
	"fmt"
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
	path := filepath.Join(t.TempDir(), "session.lock")
	command := exec.Command(os.Args[0], "-test.run=^TestProcessFileLockHelper$")
	command.Env = append(os.Environ(), processLockHelperEnv+"="+path)
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
	if _, err := acquireProcessFileLock(path); err == nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatal("competing process acquired a held lock")
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("killed lock subprocess unexpectedly exited successfully")
	}
	unlock, err := acquireProcessFileLock(path)
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
	unlock, err := acquireProcessFileLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	_, _ = fmt.Fprintln(os.Stdout, "LOCKED")
	_ = os.Stdout.Sync()
	select {}
}

func TestIncompleteLedgerPublicationFailsClosed(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".session.json.tmp", ".session.json.previous"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("uncertain"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := ensureNoIncompleteAtomicWrite(dir, "session.json"); err == nil {
			t.Fatalf("publication uncertainty at %s must fail closed", name)
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
		holder, err := os.Open(filepath.Join(dir, "session.json"))
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := WriteBrownfieldSession(root, next, session.Digest)
		_ = holder.Close()
		if writeErr == nil {
			t.Skip("replace succeeded despite an open reader; failure could not be provoked")
		}
		if _, err := os.Lstat(filepath.Join(dir, ".session.json.tmp")); err == nil {
			t.Errorf("failed replace left %s behind", filepath.Join(dir, ".session.json.tmp"))
		}
		if _, err := LoadBrownfieldSession(root, session.ID); err != nil {
			t.Errorf("unchanged session can no longer be loaded after a reported write failure: %v", err)
		}
		if _, err := WriteBrownfieldSession(root, next, session.Digest); err != nil {
			t.Errorf("retrying the same write after the handle closed still fails: %v", err)
		}
	})
	t.Run("manager-run-ledger", func(t *testing.T) {
		dir := t.TempDir()
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
		if _, err := loadManagerRunLedger(dir, "failed-replace"); err != nil {
			t.Errorf("unchanged manager-run ledger can no longer be loaded after a reported write failure: %v", err)
		}
		if err := writeManagerRunLedger(dir, ledger); err != nil {
			t.Errorf("retrying the same ledger write after the handle closed still fails: %v", err)
		}
	})
}
