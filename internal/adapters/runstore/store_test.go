package runstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestIndependentProcessesAppendCompleteAttributedNotes(t *testing.T) {
	const workers, perWorker = 4, 8
	const workerEnv, rootEnv = "MARKITECT_NOTE_TEST_WORKER", "MARKITECT_NOTE_TEST_ROOT"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	note := func(worker, sequence int) run.Note {
		return run.Note{Project: "shared", Run: fmt.Sprintf("run-%d", worker), Manager: fmt.Sprintf("area-%d", worker), Context: fmt.Sprintf("fixed-candidate-%d", worker), Text: fmt.Sprintf("note-%d-%d:%s", worker, sequence, strings.Repeat("complete attributed hint;", 64))}
	}
	if selected := os.Getenv(workerEnv); selected != "" {
		worker, err := strconv.Atoi(selected)
		if err != nil || worker < 0 || worker >= workers {
			t.Fatal("invalid subprocess identity")
		}
		root := os.Getenv(rootEnv)
		store, err := New(root)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.Mkdir(filepath.Join(root, fmt.Sprintf("ready-%d-%d", worker, os.Getpid())), 0700); err != nil {
			t.Fatal(err)
		}
		for {
			if _, err = os.Stat(filepath.Join(root, "start")); err == nil {
				break
			}
			if !os.IsNotExist(err) {
				t.Fatal(err)
			}
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(time.Millisecond):
			}
		}
		for sequence := range perWorker {
			if _, err = store.Append(ctx, note(worker, sequence)); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	root := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		worker int
		err    error
		output string
	}
	results := make(chan result, workers)
	remaining := 0
	defer func() {
		cancel()
		for remaining > 0 {
			<-results
			remaining--
		}
	}()
	var processes []*exec.Cmd
	for worker := range workers {
		cmd := exec.CommandContext(ctx, executable, "-test.run=^TestIndependentProcessesAppendCompleteAttributedNotes$", "-test.count=1", "-test.timeout=30s")
		cmd.Env = append(os.Environ(), workerEnv+"="+strconv.Itoa(worker), rootEnv+"="+root)
		output := &bytes.Buffer{}
		cmd.Stdout, cmd.Stderr = output, output
		if err = cmd.Start(); err != nil {
			t.Fatal(err)
		}
		processes = append(processes, cmd)
		remaining++
		go func() { err := cmd.Wait(); results <- result{worker, err, output.String()} }()
	}
	// Every independent OS process reaches the barrier before any can append.
	for worker, cmd := range processes {
		for {
			info, readErr := os.Stat(filepath.Join(root, fmt.Sprintf("ready-%d-%d", worker, cmd.Process.Pid)))
			if readErr == nil {
				if !info.IsDir() || cmd.Process.Pid == os.Getpid() {
					t.Fatal("writer did not run in its own OS process")
				}
				break
			}
			if !os.IsNotExist(readErr) {
				t.Fatal(readErr)
			}
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(time.Millisecond):
			}
		}
	}
	want := map[string]run.Note{}
	for worker := range workers {
		for sequence := range perWorker {
			value := note(worker, sequence)
			want[value.Text] = value
		}
	}
	validate := func(notes []run.Note) {
		t.Helper()
		ids := map[string]bool{}
		texts := map[string]bool{}
		for _, actual := range notes {
			expected, ok := want[actual.Text]
			expected.ID = actual.ID
			if !ok || actual.ID == "" || ids[actual.ID] || texts[actual.Text] || actual != expected {
				t.Fatal("partial, duplicate or misattributed Note", actual)
			}
			ids[actual.ID], texts[actual.Text] = true, true
		}
	}
	if err = os.WriteFile(filepath.Join(root, "start"), []byte("append"), 0600); err != nil {
		t.Fatal(err)
	}
	reader, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for remaining > 0 {
		notes, err := allNotes(ctx, reader, "shared")
		if err != nil {
			t.Fatal("reader observed a partial record", err)
		}
		validate(notes)
		select {
		case result := <-results:
			remaining--
			if result.err != nil {
				t.Fatalf("writer %d: %v\n%s", result.worker, result.err, result.output)
			}
		case <-tick.C:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	fresh, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	notes, err := allNotes(ctx, fresh, "shared")
	if err != nil || len(notes) != workers*perWorker {
		t.Fatal("a process append was lost", len(notes), err)
	}
	validate(notes)
}

func TestIndependentInstancesAppendNotesWithoutOverwriteOrPartialRecords(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	const count = 32
	var wg sync.WaitGroup
	failures := make(chan error, count)
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := New(root)
			if err == nil {
				_, err = s.Append(ctx, run.Note{Project: "project", Run: "run", Manager: "a", Context: "fixed candidate", Text: "attributed hint"})
			}
			failures <- err
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	s, _ := New(root)
	notes, err := allNotes(ctx, s, "project")
	if err != nil || len(notes) != count {
		t.Fatal(len(notes), err)
	}
	seen := map[string]bool{}
	for _, n := range notes {
		if seen[n.ID] || n.Project != "project" || n.Run != "run" || n.Manager != "a" {
			t.Fatal("lost provenance or unique append", n)
		}
		seen[n.ID] = true
	}
	other, err := allNotes(ctx, s, "other")
	if err != nil || len(other) != 0 {
		t.Fatal("project Notes crossed boundary", other, err)
	}
}
func TestTerminalRecordsAreCreateOnceAndIgnorePendingNotes(t *testing.T) {
	ctx := context.Background()
	s, _ := New(t.TempDir())
	r := run.Result{ID: "one", Project: "project", Technical: run.Incomplete, Cleanup: []run.Closure{{Resource: "session", Closed: false, Error: "uncertain"}}}
	if err := s.Save(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.Technical = run.Completed
	if err := s.Save(ctx, r); err == nil {
		t.Fatal("terminal result overwritten")
	}
	saved, err := s.Read(ctx, "project", "one")
	if err != nil || saved.Technical != run.Incomplete {
		t.Fatal(saved, err)
	}
	p, _ := s.path("project", "notes", "ignored")
	if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(filepath.Dir(p), ".pending-incomplete"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if notes, err := allNotes(ctx, s, "project"); err != nil || len(notes) != 0 {
		t.Fatal("partial Note visible", notes, err)
	}
	if _, err = s.Read(ctx, "../escape", "one"); err == nil {
		t.Fatal("unsafe project path accepted")
	}
}

// allNotes reads every page of project Notes in order.
func allNotes(ctx context.Context, records interface {
	Notes(context.Context, string, string) (run.NotePage, error)
}, project string) ([]run.Note, error) {
	var all []run.Note
	for after := ""; ; {
		page, err := records.Notes(ctx, project, after)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Notes...)
		if page.Next == "" {
			return all, nil
		}
		after = page.Next
	}
}

func appendSized(t *testing.T, s *Store, project string, count, size int) []run.Note {
	t.Helper()
	var stored []run.Note
	for i := range count {
		n, err := s.Append(context.Background(), run.Note{Project: project, Run: "run", Manager: "m", Context: "area", Text: fmt.Sprintf("%06d", i) + strings.Repeat("x", size-6)})
		if err != nil {
			t.Fatal(err)
		}
		stored = append(stored, n)
	}
	return stored
}

// Stored Notes whose total is far above one tool transport frame are read
// completely, in creation order, in pages that each stay within the limits.
func TestNotesExceedingOneFrameAreReadCompletelyInStablePages(t *testing.T) {
	ctx := context.Background()
	s, _ := New(t.TempDir())
	const count = 160 // 160 * 16 KiB is about 2.5 MiB, above the 1 MiB frame
	stored := appendSized(t, s, "p", count, run.MaxNoteText)
	var got []run.Note
	pages := 0
	for after := ""; ; pages++ {
		page, err := s.Notes(ctx, "p", after)
		if err != nil {
			t.Fatal(err)
		}
		encoded := 0
		for _, n := range page.Notes {
			raw, _ := json.Marshal(n)
			encoded += len(raw)
		}
		if len(page.Notes) == 0 || len(page.Notes) > run.NotePageCount || encoded > run.NotePageBytes {
			t.Fatal("page outside its limits", len(page.Notes), encoded)
		}
		got = append(got, page.Notes...)
		if page.Next == "" {
			break
		}
		if page.Next != page.Notes[len(page.Notes)-1].ID {
			t.Fatal("cursor is not the last Note of the page", page.Next)
		}
		after = page.Next
	}
	if pages < 2 || len(got) != count {
		t.Fatal("not every Note was returned", pages, len(got))
	}
	for i, n := range got {
		if n.ID != stored[i].ID || n.Text != stored[i].Text {
			t.Fatalf("Note %d out of creation order or altered", i)
		}
	}
}

func TestNotePagesStopAtTheCountLimitAndEmptyStoresHaveNoPages(t *testing.T) {
	ctx := context.Background()
	s, _ := New(t.TempDir())
	if page, err := s.Notes(ctx, "none", ""); err != nil || len(page.Notes) != 0 || page.Next != "" || page.Notes == nil {
		t.Fatal("empty project", page, err)
	}
	stored := appendSized(t, s, "p", run.NotePageCount+1, 10)
	first, err := s.Notes(ctx, "p", "")
	if err != nil || len(first.Notes) != run.NotePageCount || first.Next != stored[run.NotePageCount-1].ID {
		t.Fatal("first page", len(first.Notes), first.Next, err)
	}
	last, err := s.Notes(ctx, "p", first.Next)
	if err != nil || len(last.Notes) != 1 || last.Next != "" || last.Notes[0].ID != stored[run.NotePageCount].ID {
		t.Fatal("last page", last, err)
	}
	if _, err = s.Notes(ctx, "p", "../escape"); err == nil || !strings.Contains(err.Error(), "next value") {
		t.Fatal("invalid cursor accepted or unexplained", err)
	}
}

func TestOversizedNotesAreRefusedWithReasonAndNothingIsStored(t *testing.T) {
	ctx := context.Background()
	s, _ := New(t.TempDir())
	base := run.Note{Project: "p", Run: "run", Manager: "m", Context: "area", Text: "ok"}
	long := base
	long.Text = strings.Repeat("x", run.MaxNoteText+1)
	if _, err := s.Append(ctx, long); err == nil || !strings.Contains(err.Error(), "shorten it") {
		t.Fatal("oversized text accepted or unexplained", err)
	}
	wide := base
	wide.Context = strings.Repeat("x", run.MaxNoteContext+1)
	if _, err := s.Append(ctx, wide); err == nil || !strings.Contains(err.Error(), "more briefly") {
		t.Fatal("oversized context accepted or unexplained", err)
	}
	edge := base
	edge.Text, edge.Context = strings.Repeat("x", run.MaxNoteText), strings.Repeat("x", run.MaxNoteContext)
	if _, err := s.Append(ctx, edge); err != nil {
		t.Fatal("Note at the limit refused", err)
	}
	if all, err := allNotes(ctx, s, "p"); err != nil || len(all) != 1 {
		t.Fatal("refused Notes were stored", len(all), err)
	}
}

// Even a Note made only of bytes that need the longest JSON escape fits a page.
func TestWorstCaseEscapedNotesStayWithinThePageBudget(t *testing.T) {
	ctx := context.Background()
	s, _ := New(t.TempDir())
	for range 20 {
		if _, err := s.Append(ctx, run.Note{Project: "p", Run: "run", Manager: "m", Context: strings.Repeat("\x01", run.MaxNoteContext), Text: strings.Repeat("\x01", run.MaxNoteText)}); err != nil {
			t.Fatal(err)
		}
	}
	for after, seen := "", 0; ; {
		page, err := s.Notes(ctx, "p", after)
		if err != nil || len(page.Notes) == 0 {
			t.Fatal(err, len(page.Notes))
		}
		raw, _ := json.Marshal(page)
		if len(raw) > run.NotePageBytes+1024 {
			t.Fatal("escaped page too large", len(raw))
		}
		if seen += len(page.Notes); page.Next == "" {
			if seen != 20 {
				t.Fatal("Notes lost", seen)
			}
			return
		}
		after = page.Next
	}
}

// A Note stored before the size limits existed is never dropped or cut: the
// page before it ends there, and reading it explains how to continue.
func TestStoredNoteAbovePageSizeIsReportedNotTruncated(t *testing.T) {
	ctx := context.Background()
	s, _ := New(t.TempDir())
	first := appendSized(t, s, "p", 1, 10)[0]
	huge := run.Note{ID: first.ID + "-huge", Project: "p", Run: "run", Manager: "m", Context: "old", Text: strings.Repeat("y", 2*run.NotePageBytes)}
	path, _ := s.path("p", "notes", huge.ID)
	if err := write(ctx, path, huge); err != nil {
		t.Fatal(err)
	}
	last := appendSized(t, s, "p", 1, 10)[0]
	page, err := s.Notes(ctx, "p", "")
	if err != nil || len(page.Notes) != 1 || page.Next != first.ID {
		t.Fatal("page before the large Note", page.Next, err)
	}
	_, err = s.Notes(ctx, "p", page.Next)
	if err == nil || !strings.Contains(err.Error(), "continue after it") {
		t.Fatal("large Note not reported with a next step", err)
	}
	rest, err := s.Notes(ctx, "p", huge.ID)
	if err != nil || len(rest.Notes) != 1 || rest.Notes[0].ID != last.ID {
		t.Fatal("reading did not continue after the large Note", rest, err)
	}
}
