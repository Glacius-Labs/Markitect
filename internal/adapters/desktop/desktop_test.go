package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testConfig(dir string, opened chan<- string) Config {
	repos := newFakeRepos()
	return Config{Dir: dir, Listen: "127.0.0.1:0", Open: true, Repos: repos, Commands: fakeCommands{repos}, Stdout: &bytes.Buffer{},
		OpenWindow: func(u string) error { opened <- u; return nil }}
}

// startService runs Start in the background and returns the URL of the
// window it opened and how to stop it.
func startService(t *testing.T, dir string) (url string, stop func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	opened := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- Start(ctx, testConfig(dir, opened)) }()
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
		// The client may hold a connection it opened ahead and never used;
		// the service would wait for it through its grace period.
		http.DefaultTransport.(*http.Transport).CloseIdleConnections()
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

// session redeems a window's link as the browser does and returns the
// session cookie, or fails.
func session(t *testing.T, url string) *http.Cookie {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	for _, c := range resp.Cookies() {
		if strings.HasPrefix(c.Name, "markitect_") {
			return c
		}
	}
	t.Fatalf("no session from %s: %s", url, resp.Status)
	return nil
}

func get(t *testing.T, url string, c *http.Cookie) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(c)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestTheServiceServesTheWindowItOpensAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	url, stop := startService(t, dir)
	base, code, ok := strings.Cut(url, "/?t=")
	if !ok || !strings.HasPrefix(base, "http://127.0.0.1:") || len(code) != 64 {
		t.Fatalf("window URL %q", url)
	}
	if status := get(t, base+"/api/projects", session(t, url)); status != http.StatusOK {
		t.Fatalf("projects: %d", status)
	}
	data, err := os.ReadFile(filepath.Join(dir, recordFile))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if strings.Contains(string(data), code) {
		t.Error("the record holds the window's code")
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
	opened := make(chan string, 1)
	if err := Start(context.Background(), testConfig(dir, opened)); err != nil {
		t.Fatal(err)
	}
	second := <-opened
	firstBase, _, _ := strings.Cut(first, "/?t=")
	secondBase, _, _ := strings.Cut(second, "/?t=")
	if secondBase != firstBase || second == first {
		t.Fatalf("second window %q, first %q", second, first)
	}
	if status := get(t, secondBase+"/api/projects", session(t, second)); status != http.StatusOK {
		t.Errorf("the second window's session: %d", status)
	}
}

func TestAStaleRecordIsReplaced(t *testing.T) {
	dir := t.TempDir()
	if err := publishRecord(dir, []byte(`{"address":"127.0.0.1:1","secret":"stale","pid":1}`)); err != nil {
		t.Fatal(err)
	}
	url, stop := startService(t, dir)
	if strings.HasPrefix(url, "http://127.0.0.1:1/") {
		t.Errorf("opened the stale service: %s", url)
	}
	if err := stop(); err != nil {
		t.Fatal(err)
	}
}

func TestAProgramOnTheRecordedPortLearnsNoSecret(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]string{"proof": "anything", "code": "anything"})
	}))
	defer foreign.Close()
	dir := t.TempDir()
	address := strings.TrimPrefix(foreign.URL, "http://")
	if err := publishRecord(dir, []byte(`{"address":"`+address+`","secret":"the-secret","pid":1}`)); err != nil {
		t.Fatal(err)
	}
	url, stop := startService(t, dir)
	defer stop()
	if strings.Contains(url, address) {
		t.Errorf("opened a window on the foreign program: %s", url)
	}
	mu.Lock()
	defer mu.Unlock()
	for _, s := range seen {
		if strings.Contains(s, "the-secret") {
			t.Errorf("the foreign program received the secret: %s", s)
		}
	}
}

func TestTwoStartsAtOnceShareOneService(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	opened := make(chan string, 2)
	done := make(chan error, 2)
	for range 2 {
		go func() { done <- Start(ctx, testConfig(dir, opened)) }()
	}
	a, b := <-opened, <-opened
	aBase, _, _ := strings.Cut(a, "/?t=")
	bBase, _, _ := strings.Cut(b, "/?t=")
	if aBase != bBase {
		t.Errorf("two services: %s and %s", aBase, bBase)
	}
	cancel()
	for range 2 {
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Start did not end")
		}
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
