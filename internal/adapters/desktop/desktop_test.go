package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
// window it opened, where its later windows arrive, and how to stop it.
func startService(t *testing.T, dir string) (url string, windows <-chan string, stop func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	opened := make(chan string, 4)
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
	return url, opened, func() error {
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
	url, _, stop := startService(t, dir)
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

func TestASecondStartHasTheRunningServiceOpenAWindow(t *testing.T) {
	dir := t.TempDir()
	first, windows, stop := startService(t, dir)
	defer stop()
	ownWindow := make(chan string, 1)
	cfg := testConfig(dir, ownWindow)
	out := &bytes.Buffer{}
	cfg.Stdout = out
	if err := Start(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	var second string
	select {
	case second = <-windows:
	case <-time.After(10 * time.Second):
		t.Fatal("the running service opened no window")
	}
	if len(ownWindow) != 0 {
		t.Error("the second start opened a window of its own")
	}
	firstBase, _, _ := strings.Cut(first, "/?t=")
	secondBase, _, _ := strings.Cut(second, "/?t=")
	if secondBase != firstBase || second == first {
		t.Fatalf("second window %q, first %q", second, first)
	}
	if status := get(t, secondBase+"/api/projects", session(t, second)); status != http.StatusOK {
		t.Errorf("the second window's session: %d", status)
	}
	if !strings.Contains(out.String(), "already running at "+firstBase) {
		t.Errorf("second start said %q", out)
	}
}

func TestASecondStartWithoutAWindowPrintsALink(t *testing.T) {
	dir := t.TempDir()
	first, windows, stop := startService(t, dir)
	defer stop()
	cfg := testConfig(dir, make(chan string, 1))
	out := &bytes.Buffer{}
	cfg.Stdout, cfg.Open = out, false
	if err := Start(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if len(windows) != 0 {
		t.Error("the running service opened a window")
	}
	base, _, _ := strings.Cut(first, "/?t=")
	_, rest, ok := strings.Cut(out.String(), "open "+base+"/?t=")
	code, _, _ := strings.Cut(rest, " ")
	if !ok || len(code) != 64 {
		t.Fatalf("no link in %q", out)
	}
	if status := get(t, base+"/api/projects", session(t, base+"/?t="+code)); status != http.StatusOK {
		t.Errorf("the printed link's session: %d", status)
	}
}

func TestAWindowThatFailsHandsTheLinkToTheSecondStart(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := testConfig(dir, nil)
	cfg.Open = false
	cfg.OpenWindow = func(string) error { return errors.New("no display") }
	s, err := Listen(ctx, cfg)
	if err != nil || s == nil {
		t.Fatalf("listen: %v %v", s, err)
	}
	served := make(chan error, 1)
	go func() { served <- s.Serve(ctx) }()
	second := testConfig(dir, make(chan string, 1))
	err = Start(ctx, second)
	if err == nil || !strings.Contains(err.Error(), "no display") || !strings.Contains(err.Error(), "/?t=") {
		t.Errorf("second start: %v", err)
	}
	cancel()
	if err := <-served; err != nil {
		t.Fatal(err)
	}
}

// recordedService publishes a record for a program that proves the record's
// secret, as only this user's Markitect can, and answers its other routes
// with answer.
func recordedService(t *testing.T, dir string, answer http.HandlerFunc) {
	t.Helper()
	const secret = "the-secret"
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/hello" {
			_ = json.NewEncoder(w).Encode(map[string]string{"proof": proof(secret, r.URL.Query().Get("c"))})
			return
		}
		answer(w, r)
	}))
	t.Cleanup(svc.Close)
	address := strings.TrimPrefix(svc.URL, "http://")
	if err := publishRecord(dir, []byte(`{"address":"`+address+`","secret":"`+secret+`","pid":1}`)); err != nil {
		t.Fatal(err)
	}
}

func TestALaterStartNamesAServiceOfAnEarlierVersion(t *testing.T) {
	dir := t.TempDir()
	// A service before /api/window asked for its session cookie instead.
	recordedService(t, dir, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Open Markitect with markitect desktop.", http.StatusUnauthorized)
	})
	err := Start(context.Background(), testConfig(dir, make(chan string, 1)))
	if err == nil || !strings.Contains(err.Error(), "earlier version") {
		t.Fatalf("later start: %v", err)
	}
}

func TestALaterStartStopsWaitingForAServiceThatDoesNotAnswer(t *testing.T) {
	dir := t.TempDir()
	release := make(chan struct{})
	defer close(release)
	recordedService(t, dir, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	wait := laterStartWait
	laterStartWait = 200 * time.Millisecond
	defer func() { laterStartWait = wait }()
	err := Start(context.Background(), testConfig(dir, make(chan string, 1)))
	if err == nil || !strings.Contains(err.Error(), "did not answer") {
		t.Fatalf("later start: %v", err)
	}
}

func TestAServiceAnswersFromListenOnAndLeavesNoRecordWhenClosed(t *testing.T) {
	dir := t.TempDir()
	opened := make(chan string, 4)
	s, err := Listen(context.Background(), testConfig(dir, opened))
	if err != nil || s == nil {
		t.Fatalf("listen: %v %v", s, err)
	}
	// A native shell serves only once its window loop runs; a start in
	// between already finds the service alive and gets its window there.
	if again, err := Listen(context.Background(), testConfig(dir, make(chan string, 1))); again != nil || err != nil {
		t.Fatalf("a second start before serving: %v %v", again, err)
	}
	if err := s.Window(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if url := <-opened; !strings.Contains(url, "/?t=") {
			t.Errorf("window %q", url)
		}
	}
	s.Close()
	s.Close()
	if _, err := os.Stat(filepath.Join(dir, recordFile)); !os.IsNotExist(err) {
		t.Errorf("record left behind: %v", err)
	}
	if err := s.Serve(context.Background()); err != nil {
		t.Errorf("serving a closed service: %v", err)
	}
}

func TestAStaleRecordIsReplaced(t *testing.T) {
	dir := t.TempDir()
	if err := publishRecord(dir, []byte(`{"address":"127.0.0.1:1","secret":"stale","pid":1}`)); err != nil {
		t.Fatal(err)
	}
	url, _, stop := startService(t, dir)
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
	url, _, stop := startService(t, dir)
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
	defer cancel()
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
	// The later start ends once the service opened its window; the service
	// ends with ctx.
	for i := range 2 {
		if i == 1 {
			cancel()
		}
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
