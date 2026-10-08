package projectcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordPathIsConfinedToMarkitectDraftsAndRuns(t *testing.T) {
	for _, valid := range []string{".markitect/drafts/proposal.json", ".markitect/runs/report.json"} {
		if got, err := recordPath(valid); err != nil || got != valid {
			t.Fatalf("recordPath(%q) = %q, %v", valid, got, err)
		}
	}
	for _, invalid := range []string{
		"proposal.json",
		".markitect/project.yaml",
		".markitect/drafts/../project.yaml",
		".markitect/drafts/sub\\proposal.json",
		".markitect/drafts/proposal.yml",
		"C:/outside.json",
	} {
		if _, err := recordPath(invalid); err == nil {
			t.Errorf("recordPath accepted %q", invalid)
		}
	}
}

func TestWriteRecordCreatesOnlyAbsentMarkitectRecord(t *testing.T) {
	repo := copyProjectWorld(t)
	path := ".markitect/drafts/one.json"
	want := []byte("{\"ok\":true}\n")
	if _, err := writeRecord(repo, path, want); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("written record = %q, err=%v", got, err)
	}
	if _, err := writeRecord(repo, path, []byte("replacement")); err == nil {
		t.Fatal("writeRecord replaced an existing record")
	}
	got, err = readRecord(repo, path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("readRecord = %q, err=%v", got, err)
	}
}
