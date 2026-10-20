package cockpit

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunLogReadsCompleteLinesAndKeepsTheTail(t *testing.T) {
	s, _ := library(t)
	lines := s.lines()
	state := t.TempDir()
	dir := filepath.Join(state, "runs", "r1")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "events.jsonl")
	first := splitLines(lines)[0]
	partial := append(append([]byte{}, first...), '\n')
	partial = append(partial, []byte(`{"at":"2026-10-20T12:00:02Z","kind":"pass.st`)...)
	if err := os.WriteFile(path, partial, 0o600); err != nil {
		t.Fatal(err)
	}
	l := newRunLog(state, "r1")
	if _, err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	records, src, _ := l.snapshot()
	if len(records) != 1 || src.TailBytes == 0 || src.Replaced != 0 {
		t.Fatalf("a partial line is a tail, not a record: %d records, %+v", len(records), src)
	}
	digest := src.EventDigest

	if err := os.WriteFile(path, lines, 0o600); err != nil {
		t.Fatal(err)
	}
	bump(t, path)
	if _, err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	records, src, _ = l.snapshot()
	if len(records) != len(splitLines(lines)) || src.TailBytes != 0 || src.Replaced != 0 || src.EventDigest == digest {
		t.Fatalf("appended lines extend the history: %d records, %+v", len(records), src)
	}

	if err := os.WriteFile(path, []byte(`{"at":"2026-10-20T13:00:00Z","kind":"run.start","data":{"managers":["."]}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bump(t, path)
	if _, err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	records, src, _ = l.snapshot()
	if len(records) != 1 || src.Replaced != 1 {
		t.Fatalf("a replaced file is read again, not spliced: %d records, %+v", len(records), src)
	}
}

func TestRunLogReadsTheReportOnceItExists(t *testing.T) {
	s, _ := library(t)
	state := s.writeRun("r1", nil)
	l := newRunLog(state, "r1")
	if _, err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	if _, src, report := l.snapshot(); src.Report || report != nil {
		t.Fatal("no report yet")
	}
	path := filepath.Join(state, "runs", "r1", "report.json")
	if err := os.WriteFile(path, mustJSON(t, applyReport("success", "applied")), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := l.refresh()
	if err != nil || !changed {
		t.Fatalf("refresh = %v, %v", changed, err)
	}
	_, src, report := l.snapshot()
	if !src.Report || report.Outcome != "success" || report.Publication == nil || report.Publication.Paths[0].State != "after" {
		t.Fatalf("report = %+v", report)
	}
}

func TestDecodeReportMapsEarlierReportsWithoutOutcome(t *testing.T) {
	for status, want := range map[string]string{"passed": "success", "failed": "failed", "not_completed": "incomplete"} {
		r, err := decodeReport([]byte(`{"operation":"apply","status":"` + status + `"}`))
		if err != nil || r.Outcome != want || !r.Derived {
			t.Fatalf("%s: %+v %v", status, r, err)
		}
	}
	r, err := decodeReport([]byte(`{"status":"passed","outcome":"incomplete"}`))
	if err != nil || r.Outcome != "incomplete" || r.Derived {
		t.Fatalf("a recorded outcome is used as recorded: %+v", r)
	}
}

func TestStateDirectoryListsRuns(t *testing.T) {
	s, _ := library(t)
	state := s.writeRun("20261020T120000Z-aaaa", nil)
	if err := os.MkdirAll(filepath.Join(state, "runs", "not a run"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "current-run"), []byte("20261020T120000Z-aaaa\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runs, err := (&stateDirectory{dir: state}).list()
	if err != nil || len(runs) != 1 || runs[0].ID != "20261020T120000Z-aaaa" || !runs[0].Current {
		t.Fatalf("runs = %+v, %v", runs, err)
	}
}

// bump moves a file's modification time forward, so a rewrite of the same
// size within the clock's resolution is still seen as a change.
func bump(t *testing.T, path string) {
	t.Helper()
	later := time.Now().Add(time.Duration(len(path)) * time.Second)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
}

func TestRunLogReadsAnAppendIncrementallyAndAReplacementAgain(t *testing.T) {
	s, _ := library(t)
	lines := splitLines(s.lines())
	state := t.TempDir()
	dir := filepath.Join(state, "runs", "r1")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "events.jsonl")
	join := func(ls [][]byte) []byte {
		var data []byte
		for _, line := range ls {
			data = append(append(data, line...), '\n')
		}
		return data
	}
	if err := os.WriteFile(path, join(lines[:10]), 0o600); err != nil {
		t.Fatal(err)
	}
	l := newRunLog(state, "r1")
	if _, err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(join(lines[10:20])); err != nil {
		t.Fatal(err)
	}
	f.Close()
	bump(t, path)
	if _, err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	records, src, _ := l.snapshot()
	if len(records) != 20 || src.Replaced != 0 || src.EventDigest != hexSum(join(lines[:20])) {
		t.Fatalf("an append continues the history: %d records, %+v", len(records), src)
	}

	other := append(join(lines[1:2]), join(lines[:25])...)
	replacement := filepath.Join(dir, "events.new")
	if err := os.WriteFile(replacement, other, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	bump(t, path)
	if _, err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	records, src, _ = l.snapshot()
	if src.Replaced != 1 || len(records) != 26 || src.EventDigest != hexSum(other) {
		t.Fatalf("a renamed-in file with another beginning is read again: %d records, %+v", len(records), src)
	}
}

func hexSum(data []byte) string {
	return hex.EncodeToString(sum(data))
}
