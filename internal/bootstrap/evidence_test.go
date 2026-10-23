package bootstrap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	runport "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func readEvents(t *testing.T, e *fileEvidence) []runport.Event {
	t.Helper()
	file, err := os.Open(filepath.Join(e.Dir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var events []runport.Event
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		var event runport.Event
		if err := json.Unmarshal(lines.Bytes(), &event); err != nil {
			t.Fatalf("event line is not whole JSON: %q: %v", lines.Text(), err)
		}
		events = append(events, event)
	}
	return events
}

func TestEvidenceKeepsEachEventWholeAndVisibleWhenRecordReturns(t *testing.T) {
	state := testkit.TempDir(t)
	e, err := newFileEvidence(state, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if e.Dir() != filepath.Join(state, "runs", "run-1") {
		t.Fatalf("evidence directory: %s", e.Dir())
	}
	at := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	if err := e.Record(runport.Event{At: at, Kind: "pass.started", Pass: "plan-1", Data: map[string]any{"tree": "t1"}}); err != nil {
		t.Fatal(err)
	}
	// Each Record is on disk before it returns, so a second reader sees it now.
	if events := readEvents(t, e); len(events) != 1 || events[0].Kind != "pass.started" || !events[0].At.Equal(at) || events[0].Pass != "plan-1" {
		t.Fatalf("event not visible after Record: %+v", events)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := e.Record(runport.Event{At: at, Kind: "turn.ended", Turn: fmt.Sprintf("t%d", i), Manager: "lending/reminders"}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if events := readEvents(t, e); len(events) != 21 {
		t.Fatalf("concurrent events interleaved or lost: %d", len(events))
	}
}

func TestEvidenceRefusesUnrepresentableDataAndEventsAfterClose(t *testing.T) {
	e, err := newFileEvidence(testkit.TempDir(t), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Record(runport.Event{Kind: "bad", Data: make(chan int)}); err == nil {
		t.Fatal("an event that cannot be encoded was recorded")
	}
	if events := readEvents(t, e); len(events) != 0 {
		t.Fatalf("failed Record left a partial line: %+v", events)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if err := e.Close(); err != nil {
		t.Fatal("Close is not idempotent", err)
	}
	if err := e.Record(runport.Event{Kind: "late"}); err == nil {
		t.Fatal("event recorded after Close")
	}
}

func TestEvidenceTranscriptPathsAreInsideTheRunAndSafe(t *testing.T) {
	e, err := newFileEvidence(testkit.TempDir(t), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if got, want := e.Transcript("turn-7"), filepath.Join(e.Dir(), "transcripts", "turn-7.jsonl"); got != want || !filepath.IsAbs(got) {
		t.Fatalf("transcript path %q, want %q", got, want)
	}
	if info, err := os.Stat(filepath.Join(e.Dir(), "transcripts")); err != nil || !info.IsDir() {
		t.Fatal("transcripts directory is not ready for the adapter", err)
	}
	for _, unsafe := range []string{"", "../escape", `a\b`, "a/b", ".hidden", "a b", "turn\x00"} {
		if got := e.Transcript(unsafe); got != "" {
			t.Errorf("turn %q got transcript path %q", unsafe, got)
		}
	}
}

func TestEvidenceNeverOverwritesAnExistingRunAndRefusesUnsafeIdentities(t *testing.T) {
	state := testkit.TempDir(t)
	first, err := newFileEvidence(state, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	first.Close()
	if _, err := newFileEvidence(state, "run-1"); err == nil {
		t.Fatal("existing evidence directory was reused")
	}
	for _, unsafe := range []string{"", "..", "../run", "a/b", `a\b`, ".x"} {
		if _, err := newFileEvidence(state, unsafe); err == nil {
			t.Errorf("run identity %q accepted", unsafe)
		}
	}
}

func TestEvidenceReportIsWrittenAtomicallyAtTheEnd(t *testing.T) {
	e, err := newFileEvidence(testkit.TempDir(t), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if _, err := os.Stat(filepath.Join(e.Dir(), "report.json")); !os.IsNotExist(err) {
		t.Fatal("report exists before the run ended")
	}
	if err := e.WriteReport(map[string]any{"status": "incomplete"}); err != nil {
		t.Fatal(err)
	}
	if err := e.WriteReport(map[string]any{"status": "passed"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(e.Dir(), "report.json"))
	var report map[string]any
	if err != nil || json.Unmarshal(data, &report) != nil || report["status"] != "passed" {
		t.Fatalf("report: %s %v", data, err)
	}
	entries, err := os.ReadDir(e.Dir())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "report.json" && entry.Name() != "events.jsonl" && entry.Name() != "transcripts" {
			t.Errorf("stray evidence file %s", entry.Name())
		}
	}
}
