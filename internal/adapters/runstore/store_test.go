package runstore

import (
	"bytes"
	"context"
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
		notes, err := reader.Notes(ctx, "shared")
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
	notes, err := fresh.Notes(ctx, "shared")
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
	notes, err := s.Notes(ctx, "project")
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
	other, err := s.Notes(ctx, "other")
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
	if notes, err := s.Notes(ctx, "project"); err != nil || len(notes) != 0 {
		t.Fatal("partial Note visible", notes, err)
	}
	if _, err = s.Read(ctx, "../escape", "one"); err == nil {
		t.Fatal("unsafe project path accepted")
	}
}
