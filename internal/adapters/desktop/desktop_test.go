package desktop

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// startService runs Start in the background and returns the URL of the
// window it opened and how to stop it.
func startService(t *testing.T, dir string) (url string, stop func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	opened := make(chan string, 1)
	done := make(chan error, 1)
	repos := newFakeRepos()
	go func() {
		done <- Start(ctx, Config{Dir: dir, Listen: "127.0.0.1:0", Open: true, Repos: repos, Commands: fakeCommands{repos}, Stdout: &bytes.Buffer{},
			OpenWindow: func(u string) error { opened <- u; return nil }})
	}()
	select {
	case url = <-opened:
	case err := <-done:
		cancel()
		t.Fatalf("Start ended: %v", err)
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("no window opened")
	}
	return url, func() error {
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(10 * time.Second):
			t.Fatal("Start did not end")
			return nil
		}
	}
}

func TestTheServiceServesTheWindowItOpensAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	url, stop := startService(t, dir)
	base, token, ok := strings.Cut(url, "/?t=")
	if !ok || !strings.HasPrefix(base, "http://127.0.0.1:") || len(token) != 64 {
		t.Fatalf("window URL %q", url)
	}
	req, _ := http.NewRequest(http.MethodGet, base+"/api/ping", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ping: %d", resp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(dir, recordFile)); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := stop(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, recordFile)); !os.IsNotExist(err) {
		t.Errorf("record left behind: %v", err)
	}
}

func TestASecondStartOpensAWindowOnTheRunningService(t *testing.T) {
	dir := t.TempDir()
	first, stop := startService(t, dir)
	defer stop()
	var second string
	err := Start(context.Background(), Config{Dir: dir, Listen: "127.0.0.1:0", Open: true, Stdout: &bytes.Buffer{},
		OpenWindow: func(u string) error { second = u; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Errorf("second window %q, want %q", second, first)
	}
}

func TestAStaleRecordDoesNotStopANewService(t *testing.T) {
	dir := t.TempDir()
	if err := writeRecord(dir, record{Address: "127.0.0.1:1", Token: "stale", PID: 1}); err != nil {
		t.Fatal(err)
	}
	url, stop := startService(t, dir)
	if strings.Contains(url, "t=stale") {
		t.Errorf("opened the stale service: %s", url)
	}
	if err := stop(); err != nil {
		t.Fatal(err)
	}
}

func TestTheServiceListensOnlyOnLoopback(t *testing.T) {
	for _, listen := range []string{"0.0.0.0:0", "192.0.2.1:0", ":0", "example.com:0"} {
		err := Start(context.Background(), Config{Dir: t.TempDir(), Listen: listen, Stdout: &bytes.Buffer{}})
		if err == nil || !strings.Contains(err.Error(), "--listen") {
			t.Errorf("%s: %v", listen, err)
		}
	}
}
