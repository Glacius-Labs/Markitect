package run_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// asBefore turns the project into one an earlier markitect left: the spec in
// the old folder, and the record and the certificates naming files there.
func (p *plain) asBefore() {
	p.t.Helper()
	from := filepath.Join(p.dir, filepath.FromSlash(project.SpecDir))
	if err := os.Rename(from, filepath.Join(p.dir, filepath.FromSlash(project.OldSpecDir))); err != nil {
		p.t.Fatal(err)
	}
	files := []string{project.ReviewFile}
	if entries, err := os.ReadDir(filepath.Join(p.dir, filepath.FromSlash(run.ResultsDir))); err == nil {
		for _, e := range entries {
			files = append(files, run.ResultsDir+"/"+e.Name())
		}
	}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(p.dir, filepath.FromSlash(f))); err == nil {
			p.write(f, strings.ReplaceAll(p.read(f), project.SpecDir+"/", project.OldSpecDir+"/"))
		}
	}
}

// update moves the spec from the folder an earlier markitect kept it in, and the
// record keeps every verdict: its lines are keyed by Definition and hash content,
// and the evidence that names a file of the spec is renamed with it.
func TestMoveSpecKeepsTheRecord(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	limit := p.item(p.status(true), "loan-limit")
	if text, ok := p.review("loan-limit", run.VerdictOK, limit.Fingerprint, []string{project.SpecDir + "/loan-limit.md:6", "src/members/loans/loan.go:3"}, ""); !ok {
		t.Fatalf("not recorded: %s", text)
	}
	verified := p.read(project.ReviewFile)
	// A line markitect cannot read stays as it is.
	p.write(project.ReviewFile, verified+"{\"definition\": \"by hand\"\n")
	p.asBefore()
	old := p.read(project.ReviewFile)
	has(t, old, `"evidence:.markitect/model/loan-limit.md":"sha256:`)
	// Until it moves, the old folder is an error, named once, and nothing else:
	// the spec compiles where it is.
	st := p.status(false)
	if len(st.SpecErrors) != 1 || st.SpecErrors[0].Code != "project.model" || !strings.Contains(st.SpecErrors[0].Message, "markitect update moves it to .markitect/spec") || len(st.Errors) != 0 {
		t.Fatalf("status before the move:\n%s", st.Text())
	}

	dry, err := run.MoveSpec(p.dir, true)
	if err != nil || !dry.Moved || dry.Lines != 1 {
		t.Fatalf("dry run: %+v %v", dry, err)
	}
	if p.read(project.ReviewFile) != old || p.read(project.OldSpecDir+"/project.md") == "" {
		t.Fatal("a dry run changed something")
	}
	done, err := run.MoveSpec(p.dir, false)
	if err != nil || done != dry {
		t.Fatalf("%+v %v", done, err)
	}
	if _, err := os.Stat(filepath.Join(p.dir, filepath.FromSlash(project.OldSpecDir))); !os.IsNotExist(err) {
		t.Fatalf("the old folder is still there: %v", err)
	}
	if got := p.read(project.ReviewFile); got != verified+"{\"definition\": \"by hand\"\n" {
		t.Fatalf("the record is not the one verified, with the unreadable line in place:\n%s", got)
	}
	p.write(project.ReviewFile, verified)
	if st := p.status(false); !st.Clean() {
		t.Fatalf("a verdict was lost in the move:\n%s", st.Text())
	}
	// Run again, it finds nothing to do.
	if again, err := run.MoveSpec(p.dir, false); err != nil || again.Changed() {
		t.Fatalf("%+v %v", again, err)
	}
	// A line a merge brings in from a branch that has not moved is renamed too.
	p.write(project.ReviewFile, strings.ReplaceAll(verified, project.SpecDir+"/", project.OldSpecDir+"/"))
	if mv, err := run.MoveSpec(p.dir, false); err != nil || mv.Moved || mv.Lines != 1 || p.read(project.ReviewFile) != verified {
		t.Fatalf("%+v %v", mv, err)
	}
}

// A project with files in both folders has two specs: update refuses, says why,
// and changes nothing. Empty folders are no spec.
func TestMoveSpecRefusesTwoSpecs(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	p.asBefore()
	p.write(project.SpecDir+"/project.md", def("std/Project", "Fresh", "Started anew."))
	before := p.read(project.ReviewFile)
	if mv, err := run.MoveSpec(p.dir, false); err == nil || mv.Changed() || !strings.Contains(err.Error(), "both .markitect/model and .markitect/spec hold files") {
		t.Fatalf("%+v %v", mv, err)
	}
	if p.read(project.ReviewFile) != before || p.read(project.OldSpecDir+"/project.md") == "" {
		t.Fatal("the refusal changed something")
	}
	// An empty folder left in the new place is no second spec.
	p.remove(project.SpecDir + "/project.md")
	if err := os.MkdirAll(filepath.Join(p.dir, filepath.FromSlash(project.SpecDir), "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if mv, err := run.MoveSpec(p.dir, false); err != nil || !mv.Moved {
		t.Fatalf("%+v %v", mv, err)
	}
	if st := p.status(false); !st.Clean() {
		t.Fatalf("after the move:\n%s", st.Text())
	}
	// An empty old folder, as a checkout may leave behind, is nothing to move.
	if err := os.MkdirAll(filepath.Join(p.dir, filepath.FromSlash(project.OldSpecDir), "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if mv, err := run.MoveSpec(p.dir, false); err != nil || mv.Changed() {
		t.Fatalf("%+v %v", mv, err)
	}
}

// A certificate of markitect 0.20 names the spec in the old folder; the record
// written from it names the new one.
func TestMigrationFromACertificateOfTheOldFolder(t *testing.T) {
	p := newPlain(t)
	p.certify("gate-20261106-120000-aaaaaaaa", "", time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC), []map[string]any{
		{"item": "loan-limit", "kind": "definition", "definition": "loan-limit", "answer": "ok", "pointers": []map[string]any{{"file": project.SpecDir + "/loan-limit.md", "line": 6}}},
	})
	p.asBefore()
	// As update does: the folder first, then the record from the certificate.
	if mv, err := run.MoveSpec(p.dir, false); err != nil || !mv.Moved || mv.Lines != 0 {
		t.Fatalf("%+v %v", mv, err)
	}
	if _, err := run.Migrate(p.dir, false); err != nil {
		t.Fatal(err)
	}
	record := p.read(project.ReviewFile)
	has(t, record, `"evidence":[".markitect/spec/loan-limit.md:6"]`, `"evidence:.markitect/spec/loan-limit.md":"sha256:`)
	if strings.Contains(record, project.OldSpecDir) {
		t.Fatalf("the record names the old folder:\n%s", record)
	}
	if st := p.status(false); !st.Clean() {
		t.Fatalf("the certified state is not recorded:\n%s", st.Text())
	}
}
