package run_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// oldHash is a file's hash as markitect 0.20 wrote it into a certificate.
func oldHash(data []byte) string {
	if bytes.IndexByte(data, 0) < 0 {
		data = project.Normalize(data)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// certify writes a certificate of the folder as it is, the way gate of markitect
// 0.20 did, with its Result page beside it, and returns its name.
func (p *plain) certify(name, base string, created time.Time, items []map[string]any) string {
	p.t.Helper()
	files, err := mustOpen(p.t, p.dir).Files()
	if err != nil {
		p.t.Fatal(err)
	}
	hashes, spec := map[string]string{}, map[string]string{}
	for _, f := range files {
		if strings.HasPrefix(f, run.ResultsDir+"/") {
			continue
		}
		data := []byte(p.read(f))
		hashes[f] = oldHash(data)
		if strings.HasPrefix(f, project.Dir+"/") {
			spec[f] = string(project.Normalize(data))
		}
	}
	cert := map[string]any{"kind": "markitect-certificate", "format": 1, "created": created, "base": base, "files": hashes, "definitions": map[string]string{}, "spec": spec, "items": items}
	data, err := json.Marshal(cert)
	if err != nil {
		p.t.Fatal(err)
	}
	file := run.ResultsDir + "/" + name + ".json"
	p.write(file, string(data))
	p.write(run.ResultsDir+"/"+name+".md", "# Result\n")
	return file
}

// update writes the record from the newest certificate: every Definition ok at
// the certified state, with the certificate's pointers as evidence, "not shown"
// as skip, and the results folder removed.
func TestMigrationFromTheNewestCertificate(t *testing.T) {
	p := newPlain(t)
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	older := p.certify("gate-20261106-120000-aaaaaaaa", "", at.Add(time.Hour), nil)
	p.write("src/members/loans/loan.go", limit3)
	p.certify("gate-20261106-110000-bbbbbbbb", older, at, []map[string]any{
		{"item": "members/loans/loan", "kind": "definition", "definition": "members/loans/loan", "answer": "ok", "pointers": []map[string]any{{"file": "src/members/loans/loan.go", "line": 3}, {"file": "README.md", "tests": []string{"TestReadme"}}}},
		{"item": "catalog/book", "kind": "definition", "definition": "catalog/book", "answer": "not shown", "explanation": "no test reaches\nthe shelf"},
		{"item": "gone members/old", "kind": "gone", "definition": "members/old", "answer": "ok"},
	})
	// The results folder is no longer part of the layout.
	has(t, p.status(false).Text(), "project.results", "markitect update writes .markitect/review.sum")

	dry, err := run.Migrate(p.dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(dry) != 5 || dry[0] != project.ReviewFile {
		t.Fatalf("dry run: %v", dry)
	}
	if _, err := os.Stat(filepath.Join(p.dir, project.ReviewFile)); !os.IsNotExist(err) {
		t.Fatal("a dry run wrote the record")
	}
	done, err := run.Migrate(p.dir, false)
	if err != nil || strings.Join(done, ",") != strings.Join(dry, ",") {
		t.Fatalf("%v %v", done, err)
	}
	if _, err := os.Stat(filepath.Join(p.dir, filepath.FromSlash(run.ResultsDir))); !os.IsNotExist(err) {
		t.Fatalf("the results folder is still there: %v", err)
	}
	st := p.status(false)
	if !st.Clean() || len(st.Skipped) != 1 || st.Skipped[0].Why != "no test reaches the shelf" {
		t.Fatalf("the newer certificate, by its base, is not the one recorded:\n%s", st.Text())
	}
	has(t, p.read(project.ReviewFile), `"evidence":["src/members/loans/loan.go:3","README.md"]`, `"evidence:README.md":"sha256:`)
	// The certificate's hashes count, not the folder's: what changed since is to
	// verify.
	p.write("src/members/loans/loan.go", "package loans\n\nconst Limit = 5\n")
	if got := listed(p.status(false)); got != "loan-limit,members/loans/loan" {
		t.Fatalf("listed %q", got)
	}
	// Run again, it finds nothing to do.
	if again, err := run.Migrate(p.dir, false); err != nil || len(again) != 0 {
		t.Fatalf("%v %v", again, err)
	}
}

// A certificate this markitect cannot read is not lost: the results folder stays,
// and update says what to do.
func TestMigrationKeepsAnUnreadableCertificate(t *testing.T) {
	p := newPlain(t)
	p.write(run.ResultsDir+"/gate-20261106-120000-aaaaaaaa.json", `{"kind": "markitect-certificate", "format": 2}`)
	p.write(run.ResultsDir+"/gate-20261106-120000-aaaaaaaa.md", "# Result\n")
	if done, err := run.Migrate(p.dir, false); err == nil || !strings.Contains(err.Error(), "no certificate this markitect can read") || len(done) != 0 {
		t.Fatalf("%v %v", done, err)
	}
	if p.read(run.ResultsDir+"/gate-20261106-120000-aaaaaaaa.md") == "" {
		t.Fatal("the results folder went")
	}
	// Result pages alone, without a certificate, simply go.
	p.remove(run.ResultsDir + "/gate-20261106-120000-aaaaaaaa.json")
	if done, err := run.Migrate(p.dir, false); err != nil || len(done) != 1 {
		t.Fatalf("%v %v", done, err)
	}
}

// A project that already has its record keeps it; the results folder goes.
func TestMigrationKeepsAnExistingRecord(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	before := p.read(project.ReviewFile)
	p.write("src/members/loans/loan.go", limit3)
	p.certify("gate-20261106-120000-aaaaaaaa", "", time.Now(), nil)
	done, err := run.Migrate(p.dir, false)
	if err != nil || len(done) != 2 || p.read(project.ReviewFile) != before {
		t.Fatalf("%v %v", done, err)
	}
}
