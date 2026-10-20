package cockpit

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type sse struct {
	event string
	id    int
	data  string
}

// stream opens the run's event stream and returns its events as they arrive.
func stream(t *testing.T, base, run string, lastID int) (<-chan sse, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/runs/"+run+"/stream?after=0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if lastID > 0 {
		req.Header.Set("Last-Event-ID", strconv.Itoa(lastID))
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	out := make(chan sse, 64)
	go func() {
		defer close(out)
		defer res.Body.Close()
		scanner := bufio.NewScanner(res.Body)
		scanner.Buffer(make([]byte, 1<<20), 8<<20)
		var cur sse
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case line == "":
				if cur.event != "" {
					out <- cur
				}
				cur = sse{}
			case strings.HasPrefix(line, "event: "):
				cur.event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "id: "):
				cur.id, _ = strconv.Atoi(strings.TrimPrefix(line, "id: "))
			case strings.HasPrefix(line, "data: "):
				cur.data = strings.TrimPrefix(line, "data: ")
			}
		}
	}()
	return out, cancel
}

func next(t *testing.T, events <-chan sse, kind string) sse {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case e, ok := <-events:
			if !ok {
				t.Fatalf("the stream ended before a %s event", kind)
			}
			if e.event == kind {
				return e
			}
		case <-deadline:
			t.Fatalf("no %s event", kind)
		}
	}
}

type entriesEvent struct {
	From    int     `json:"from"`
	Entries []Entry `json:"entries"`
}

func liveServer(t *testing.T, state string, liveness Liveness) string {
	t.Helper()
	srv, err := New(Config{States: []string{state}, Liveness: liveness})
	if err != nil {
		t.Fatal(err)
	}
	srv.poll = 20 * time.Millisecond
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

func TestStreamDeliversAppendedRecordsOnceAndResumesAfterTheLastID(t *testing.T) {
	s, _ := library(t)
	lines := splitLines(s.lines())
	state := t.TempDir()
	dir := filepath.Join(state, "runs", "20261020T120000Z-live")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "events.jsonl")
	write := func(n int) {
		var data []byte
		for _, line := range lines[:n] {
			data = append(append(data, line...), '\n')
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(5)
	base := liveServer(t, state, nil)
	events, cancel := stream(t, base, "20261020T120000Z-live", 0)
	var first entriesEvent
	e := next(t, events, "entries")
	if err := json.Unmarshal([]byte(e.data), &first); err != nil || first.From != 1 || len(first.Entries) != 5 || e.id != 5 {
		t.Fatalf("first delivery = id %d %+v %v", e.id, first, err)
	}
	write(9)
	e = next(t, events, "entries")
	var second entriesEvent
	if err := json.Unmarshal([]byte(e.data), &second); err != nil || second.From != 6 || len(second.Entries) != 4 || second.Entries[0].Seq != 6 || e.id != 9 {
		t.Fatalf("appended delivery = id %d %+v %v", e.id, second, err)
	}
	cancel()

	write(12)
	events, cancel = stream(t, base, "20261020T120000Z-live", 9)
	defer cancel()
	e = next(t, events, "entries")
	var resumed entriesEvent
	if err := json.Unmarshal([]byte(e.data), &resumed); err != nil || resumed.From != 10 || len(resumed.Entries) != 3 {
		t.Fatalf("a reconnect resumes after Last-Event-ID: %+v %v", resumed, err)
	}
}

func TestStreamEndsWhenTheEventsFileIsReplaced(t *testing.T) {
	s, _ := library(t)
	state := s.writeRun("20261020T120000Z-live", nil)
	base := liveServer(t, state, nil)
	events, cancel := stream(t, base, "20261020T120000Z-live", 0)
	defer cancel()
	next(t, events, "entries")
	path := filepath.Join(state, "runs", "20261020T120000Z-live", "events.jsonl")
	if err := os.WriteFile(path, []byte(`{"at":"2026-10-20T13:00:00Z","kind":"run.start","data":{"managers":["."]}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bump(t, path)
	next(t, events, "replaced")
	if _, open := <-events; open {
		t.Fatal("the stream continued after a replacement")
	}
}

func TestHealthSeparatesTheRunStateFromTheRecord(t *testing.T) {
	s, _ := library(t)
	records := s.records()
	unfinished := records[:len(records)-1]
	state := t.TempDir()
	st := &stateDirectory{dir: state}
	l := newRunLog(state, "r1")
	writeCurrent := func(id string) {
		if err := os.WriteFile(filepath.Join(state, "current-run"), []byte(id+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	probe := func(held bool, err error) Liveness {
		return func(string) (bool, error) { return held, err }
	}
	cases := []struct {
		name     string
		liveness Liveness
		current  string
		records  []Record
		report   *Report
		want     string
	}{
		{"report written", probe(true, nil), "r1", records, &Report{Outcome: "success"}, "ended"},
		{"held by this run", probe(true, nil), "r1", unfinished, nil, "running"},
		{"outcome recorded, report pending", probe(true, nil), "r1", records, nil, "finishing"},
		{"held by another run", probe(true, nil), "r2", unfinished, nil, "ended-without-report"},
		{"not held", probe(false, nil), "r1", unfinished, nil, "ended-without-report"},
		{"probe failed", probe(false, errors.New("lock unsupported")), "r1", unfinished, nil, "unknown"},
		{"no probe", nil, "r1", unfinished, nil, "unknown"},
	}
	for _, c := range cases {
		writeCurrent(c.current)
		st.probedAt = time.Time{}
		srv := &Server{liveness: c.liveness}
		if got := srv.health(st, l, c.records, Source{}, c.report); got.Run != c.want || got.Why == "" {
			t.Errorf("%s: %+v, want %s", c.name, got, c.want)
		}
	}
}

func TestLockProbeIsRateLimitedPerStateDirectory(t *testing.T) {
	st := &stateDirectory{dir: t.TempDir()}
	calls := 0
	probe := func(string) (bool, error) { calls++; return true, nil }
	for range 5 {
		if held, err := st.held(probe); !held || err != nil {
			t.Fatal(held, err)
		}
	}
	if calls != 1 {
		t.Fatalf("the lock was probed %d times within one interval", calls)
	}
}
