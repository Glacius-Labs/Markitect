package runstore

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

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
