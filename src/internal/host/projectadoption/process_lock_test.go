package projectadoption

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
