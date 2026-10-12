package projectcli

import (
	"bytes"
	"strings"
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

func TestInputRecordsAreReadOnlyFromConfinedExistingJSON(t *testing.T) {
	repo := copyProjectWorld(t)
	want := []byte("{\"ok\":true}\n")
	path := writeDraft(t, repo, ".markitect/drafts/one.json", want)
	got, err := readRecord(repo, path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("readRecord = %q, err=%v", got, err)
	}
	if _, err := readRecord(repo, ".markitect/drafts/missing.json"); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("readRecord of a missing record = %v", err)
	}
	invalid := writeDraft(t, repo, ".markitect/drafts/invalid.json", []byte("{"))
	for _, tc := range []struct {
		input string
		fail  string
	}{
		{"docs/cancellation.md", "Markitect transport records must use the .json extension"},
		{"proposal.json", "Markitect records may be read or written only under .markitect/drafts/ or .markitect/runs/"},
		{".markitect/drafts/../outside.json", "record path must be a normalized repository-relative slash path"},
		{".markitect/drafts/missing.json", "does not exist"},
		{invalid, "--input " + invalid + " is not valid JSON"},
	} {
		code, out, errout := runCLI(t, "edit", "--repo", repo, "--input", tc.input)
		if code != 2 || out != "" || !strings.Contains(errout, "markitect edit: ") || !strings.Contains(errout, tc.fail) {
			t.Fatalf("edit --input %s exit=%d stdout=%q stderr=%q, want %q", tc.input, code, out, errout, tc.fail)
		}
	}
}
