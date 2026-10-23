package agentd_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

func initializeHome(h *harness) {
	h.t.Helper()
	h.sendControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 17, Initialize: &frame.Initialize{CodexAuth: []byte("synthetic-login")}})
	if c := h.control(frame.TypeInitialized); c.Request != 17 {
		h.t.Fatal("setup reply lost request identity")
	}
}

func TestPrivateHomeInitializedBeforeProviderAndClearedAfterExit(t *testing.T) {
	h := newHarness(t)
	initializeHome(h)
	path := filepath.Join(h.home, ".codex", "auth.json")
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "synthetic-login" {
		t.Fatal("private file not installed")
	}
	if runtime.GOOS != "windows" {
		for p, mode := range map[string]os.FileMode{filepath.Dir(path): 0700, path: 0600} {
			info, err := os.Stat(p)
			if err != nil || info.Mode().Perm() != mode {
				t.Fatalf("private permissions not %o", mode)
			}
		}
	}
	// A repeated setup must neither replace nor erase the original login.
	h.sendControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 18, Initialize: &frame.Initialize{CodexAuth: []byte("other-login")}})
	h.control(frame.TypeFailure)
	data, _ = os.ReadFile(path)
	if string(data) != "synthetic-login" {
		t.Fatal("repeated setup changed login")
	}
	h.start("echo")
	h.sendControl(frame.ControlMessage{Type: frame.TypeClearPrivateHome, Request: 19, ClearPrivateHome: &frame.ClearPrivateHome{}})
	h.control(frame.TypeFailure)
	h.sendControl(frame.ControlMessage{Type: frame.TypeStop, Stop: &frame.Stop{GraceMillis: 100}})
	h.control(frame.TypeExited)
	h.sendControl(frame.ControlMessage{Type: frame.TypeClearPrivateHome, Request: 20, ClearPrivateHome: &frame.ClearPrivateHome{}})
	if c := h.control(frame.TypePrivateHomeCleared); c.Request != 20 {
		t.Fatal("cleanup reply lost request identity")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("login remains after cleanup")
	}
	h.sendControl(frame.ControlMessage{Type: frame.TypeStart, Request: 21, Start: &frame.Start{Argv: []string{os.Args[0]}, Dir: h.home}})
	h.control(frame.TypeFailure)
}

func TestPrivateHomeRefusesExistingAuthWithoutChangingIt(t *testing.T) {
	h := newHarness(t)
	dir := filepath.Join(h.home, ".codex")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte("existing-private-login"), 0600); err != nil {
		t.Fatal(err)
	}
	h.sendControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 1, Initialize: &frame.Initialize{CodexAuth: []byte("new-secret")}})
	c := h.control(frame.TypeFailure)
	if strings.Contains(c.Failure.Message, "secret") || strings.Contains(c.Failure.Message, h.home) {
		t.Fatal("setup failure leaked private data")
	}
	h.sendControl(frame.ControlMessage{Type: frame.TypeStart, Request: 2, Start: &frame.Start{Argv: []string{os.Args[0]}, Dir: h.home}})
	h.control(frame.TypeFailure)
	h.sendControl(frame.ControlMessage{Type: frame.TypeClearPrivateHome, Request: 3, ClearPrivateHome: &frame.ClearPrivateHome{}})
	h.control(frame.TypePrivateHomeCleared)
	data, _ := os.ReadFile(path)
	if string(data) != "existing-private-login" {
		t.Fatal("unowned login was changed")
	}
}

func TestPrivateHomeDoesNotFollowDirectoryOrFileLinks(t *testing.T) {
	for _, target := range []string{"directory", "file"} {
		t.Run(target, func(t *testing.T) {
			h := newHarness(t)
			external := t.TempDir()
			outside := filepath.Join(external, "auth.json")
			if err := os.WriteFile(outside, []byte("outside-login"), 0600); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(h.home, ".codex")
			var err error
			if target == "directory" {
				err = os.Symlink(external, dir)
			} else {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(outside, filepath.Join(dir, "auth.json"))
			}
			if err != nil {
				t.Skip("symlink creation unavailable on this host")
			}
			h.sendControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 1, Initialize: &frame.Initialize{CodexAuth: []byte("new-secret")}})
			h.control(frame.TypeFailure)
			data, _ := os.ReadFile(outside)
			if string(data) != "outside-login" {
				t.Fatal("setup touched a link target")
			}
		})
	}
}

func closePrivateHarness(t *testing.T, h *harness) error {
	t.Helper()
	_ = h.toAgnt.Close()
	select {
	case err := <-h.served:
		h.served <- err // preserve the harness cleanup's completion observation
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("private setup did not close")
		return nil
	}
}

func TestPrivateHomeDisconnectCleansWithoutAProvider(t *testing.T) {
	h := newHarness(t)
	initializeHome(h)
	if err := closePrivateHarness(t, h); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(h.home, ".codex", "auth.json")); !os.IsNotExist(err) {
		t.Fatal("disconnect left a login")
	}
}

func TestPrivateHomeLostSetupReplyStillCleansTheWrittenLogin(t *testing.T) {
	home := t.TempDir()
	in, send := io.Pipe()
	out, reply := io.Pipe()
	defer send.Close()
	defer out.Close()
	done := make(chan error, 1)
	go func() {
		done <- agentd.Serve(context.Background(), in, reply, agentd.Config{PrivateHome: home})
		_ = reply.Close()
	}()
	if _, err := frame.Read(out); err != nil {
		t.Fatal(err)
	}
	// Lose the acknowledgement after sending setup. The server has to clean
	// its written file even though the caller never learned whether it existed.
	_ = out.Close()
	payload, err := frame.EncodeControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 1, Initialize: &frame.Initialize{CodexAuth: []byte("synthetic-login")}})
	if err != nil {
		t.Fatal(err)
	}
	if err := frame.Write(send, frame.Frame{Channel: frame.Control, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("lost setup acknowledgement was hidden")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("lost setup acknowledgement did not end setup")
	}
	if _, err := os.Lstat(filepath.Join(home, ".codex", "auth.json")); !os.IsNotExist(err) {
		t.Fatal("lost setup reply left a login")
	}
}

func TestPrivateHomeCleanupFailurePreventsSuccessfulServe(t *testing.T) {
	h := newHarness(t)
	initializeHome(h)
	path := filepath.Join(h.home, ".codex", "auth.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "blocker"), []byte("private-sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	h.sendControl(frame.ControlMessage{Type: frame.TypeClearPrivateHome, Request: 22, ClearPrivateHome: &frame.ClearPrivateHome{}})
	c := h.control(frame.TypeFailure)
	if c.Request != 22 || c.Failure.Message != "private home cleanup failed" {
		t.Fatal("cleanup failure not safely reported")
	}
	err := closePrivateHarness(t, h)
	if err == nil || bytes.Contains([]byte(err.Error()), []byte("private-sentinel")) {
		t.Fatal("cleanup error hidden or leaked data")
	}
}
