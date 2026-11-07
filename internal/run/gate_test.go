package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	folders "github.com/Glacius-Labs/Markitect/internal/adapters/repo"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// The work without a Run and without Git (DEC-096, DEC-097): a plain folder, and
// the test plays the driver and its agents while gate checks what they did.

// plain is a project in a folder with no Git, and its engine.
type plain struct {
	t      *testing.T
	dir    string
	engine *run.Engine
	now    time.Time
}

func newPlain(t *testing.T) *plain {
	t.Helper()
	return newPlainOf(t, library())
}

func newPlainOf(t *testing.T, files map[string]string) *plain {
	t.Helper()
	p := &plain{t: t, dir: testkit.TempDir(t), now: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)}
	for file, content := range files {
		write(t, p.dir, file, content)
	}
	// Every reading of the clock is a second later, so that each certificate has
	// its own time.
	p.engine = &run.Engine{Folder: folders.Folder{}, Cache: testkit.TempDir(t), Now: func() time.Time {
		p.now = p.now.Add(time.Second)
		return p.now
	}}
	return p
}

func (p *plain) write(file, content string) { write(p.t, p.dir, file, content) }

func (p *plain) remove(file string) {
	p.t.Helper()
	if err := os.Remove(filepath.Join(p.dir, filepath.FromSlash(file))); err != nil {
		p.t.Fatal(err)
	}
}

func (p *plain) list(req run.ChecklistRequest) string {
	p.t.Helper()
	if req.Dir == "" {
		req.Dir = p.dir
	}
	text, ok, err := p.engine.Checklist(context.Background(), req)
	if err != nil || !ok {
		p.t.Fatalf("checklist: %v\n%s", err, text)
	}
	return text
}

func (p *plain) gate(req run.GateRequest) run.Verdict {
	p.t.Helper()
	if req.Dir == "" {
		req.Dir = p.dir
	}
	v, err := p.engine.Gate(context.Background(), req)
	if err != nil {
		p.t.Fatal(err)
	}
	return v
}

// reviewKeys reads the items a review prompt asks about.
func reviewKeys(prompt string) []string {
	var keys []string
	for _, line := range strings.Split(prompt, "\n") {
		rest, ok := strings.CutPrefix(line, "[ ] ")
		if !ok {
			continue
		}
		if i := strings.Index(rest, `: "`); i >= 0 {
			rest = rest[:i]
		}
		keys = append(keys, rest)
	}
	return keys
}

// review answers what the review asks about, the way a reviewer does: ok with a
// pointer to README.md, except where except says otherwise.
func (p *plain) review(except map[string]string) (string, []string) {
	p.t.Helper()
	// The builder's checklist began the work; calling it again goes on with it.
	p.list(run.ChecklistRequest{})
	keys := reviewKeys(p.list(run.ChecklistRequest{Review: true}))
	var lines []string
	for _, k := range keys {
		line, ok := except[k]
		if !ok {
			line = "ok README.md:1"
		}
		lines = append(lines, k+": "+line)
	}
	text, _, err := p.engine.Answer(context.Background(), p.dir, []byte(strings.Join(lines, "\n")))
	if err != nil {
		p.t.Fatal(err)
	}
	return text, keys
}

// results lists the results folder's files with a suffix, oldest first.
func (p *plain) results(suffix string) []string {
	p.t.Helper()
	found, err := filepath.Glob(filepath.Join(p.dir, ".markitect", "results", "gate-*"+suffix))
	if err != nil {
		p.t.Fatal(err)
	}
	sort.Strings(found)
	return found
}

func (p *plain) read(file string) string {
	p.t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		p.t.Fatal(err)
	}
	return string(data)
}

// certify gives the folder its first certificate: every Definition, reviewed.
func (p *plain) certify() string {
	p.t.Helper()
	has(p.t, p.list(run.ChecklistRequest{}), "no certificate yet, so every Definition is on the list")
	p.review(nil)
	v := p.gate(run.GateRequest{})
	if !v.Done {
		p.t.Fatalf("not certified: %s", v.Text)
	}
	certs := p.results(".json")
	return certs[len(certs)-1]
}

const limit3 = "package loans\n\nconst Limit = 3\n"

// specChange is the owner's change of the spec, before the work.
func (p *plain) specChange() {
	p.write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
}

// The first certificate covers every Definition; a change then starts from it, and
// shows what the changed Definition said before.
func TestGateCertifiesWithoutGit(t *testing.T) {
	p := newPlain(t)
	first := p.certify()
	if _, err := os.Stat(filepath.Join(p.dir, ".git")); err == nil {
		t.Fatal("a repository appeared")
	}
	var cert struct {
		Kind        string            `json:"kind"`
		Files       map[string]string `json:"files"`
		Definitions map[string]string `json:"definitions"`
		Spec        map[string]string `json:"spec"`
	}
	if err := json.Unmarshal([]byte(p.read(first)), &cert); err != nil {
		t.Fatal(err)
	}
	if cert.Kind != "markitect-certificate" || len(cert.Files["src/members/loans/loan.go"]) != 64 || cert.Definitions["loan-limit"] == "" || !strings.Contains(cert.Spec[m+"loan-limit.md"], "At most `5`") {
		t.Fatalf("the certificate: %+v", cert)
	}
	has(t, p.read(strings.TrimSuffix(first, ".json")+".md"), "# Gate ", ": passed", "| Certificate | .markitect/results/gate-", "not measured")
	has(t, p.gate(run.GateRequest{}).Text, "Nothing to check", "matches its newest certificate")

	p.specChange()
	list := p.list(run.ChecklistRequest{})
	has(t, list, "Make this project match the spec", "The base is the certificate .markitect/results/gate-", "loan-limit", "src/members/loans/loan.go",
		"markitect gate", "## Definitions before the change", "### loan-limit (modified)", "At most `5` open loans")
	if strings.Contains(list, "catalog/book") {
		t.Fatalf("an unchanged Definition is on the list:\n%s", list)
	}

	p.write("src/members/loans/loan.go", limit3)
	v := p.gate(run.GateRequest{})
	has(t, v.Text, run.ReviewerAgent, "Review the work.", "markitect checklist --review", "run markitect gate again")
	if v.Done {
		t.Fatal("passed before a review")
	}
	prompt := p.list(run.ChecklistRequest{Review: true})
	has(t, prompt, "no benefit of the doubt", "markitect answer", "loan-limit", "At most `5` open loans")
	text, _ := p.review(nil)
	has(t, text, "every item has an answer")
	v = p.gate(run.GateRequest{})
	if !v.Done {
		t.Fatalf("did not pass: %s", v.Text)
	}
	has(t, v.Text, "The work passes", "Commit the work together with both, in one commit")
	certs := p.results(".json")
	if len(certs) != 2 {
		t.Fatalf("certificates %v", certs)
	}
	has(t, p.read(certs[1]), `"base": ".markitect/results/`+filepath.Base(first)+`"`)
	if v := p.gate(run.GateRequest{Verify: true}); !v.Done {
		t.Fatalf("the folder does not match its certificate: %s", v.Text)
	}
}

// Row 1: what the work may not change since checklist began it.
func TestGateRefusesProtectedFiles(t *testing.T) {
	for _, file := range []string{
		m + "catalog/book.md",
		project.IgnoreFile,
		project.ManifestFile,
		"AGENTS.md",
		".claude/skills/markitect-apply/x.md",
		".claude/agents/markitect-reviewer.md",
		"src/.gitignore",
	} {
		t.Run(file, func(t *testing.T) {
			p := newPlain(t)
			p.certify()
			p.specChange()
			p.list(run.ChecklistRequest{})
			before := len(p.results(".md"))
			p.write(file, "changed\n")
			v := p.gate(run.GateRequest{})
			has(t, v.Text, file, "Put them back", "its own step, before the work")
			if v.Done || len(p.results(".md")) != before {
				t.Fatalf("not refused: %s", v.Text)
			}
		})
	}
	t.Run("a link that leaves the folder", func(t *testing.T) {
		p := newPlain(t)
		p.certify()
		p.list(run.ChecklistRequest{})
		if err := os.Symlink(filepath.Join("..", "..", "elsewhere"), filepath.Join(p.dir, "src", "out")); err != nil {
			t.Skipf("this system makes no symbolic link here: %v", err)
		}
		has(t, p.gate(run.GateRequest{}).Text, "src/out is a link that leaves the folder")
	})
}

// A spec changed during the work stops gate and a plain checklist alike, so that
// a builder cannot change the spec and begin again; the owner begins the work
// again by naming the change.
func TestASpecChangedDuringTheWork(t *testing.T) {
	p := newPlain(t)
	first := p.certify()
	p.list(run.ChecklistRequest{})
	p.write("src/members/loans/loan.go", limit3)
	p.specChange()
	has(t, p.gate(run.GateRequest{}).Text, "Put them back", m+"loan-limit.md", "only the owner begins the work again")
	text, ok, err := p.engine.Checklist(context.Background(), run.ChecklistRequest{Dir: p.dir})
	if err != nil || ok {
		t.Fatalf("a plain checklist began the work again: %v %v\n%s", ok, err, text)
	}
	has(t, text, "Put them back", m+"loan-limit.md")
	has(t, p.list(run.ChecklistRequest{Base: first}), "loan-limit")
	has(t, p.gate(run.GateRequest{}).Text, run.ReviewerAgent)
}

// The certificates are the owner's during the work: one planted in the results
// folder stops gate, and does not become the base.
func TestAPlantedCertificate(t *testing.T) {
	p := newPlain(t)
	first := p.certify()
	p.specChange()
	p.list(run.ChecklistRequest{})
	p.write("src/members/loans/loan.go", limit3)
	var cert map[string]any
	if err := json.Unmarshal([]byte(p.read(first)), &cert); err != nil {
		t.Fatal(err)
	}
	cert["created"] = "2099-01-01T00:00:00Z"
	data, _ := json.Marshal(cert)
	p.write(".markitect/results/gate-20990101-000000-00000000.json", string(data))
	v := p.gate(run.GateRequest{})
	if v.Done {
		t.Fatalf("passed: %s", v.Text)
	}
	has(t, v.Text, "gate-20990101-000000-00000000.json", "Put them back")
	if text, ok, _ := p.engine.Checklist(context.Background(), run.ChecklistRequest{Dir: p.dir}); ok {
		t.Fatalf("a plain checklist went on:\n%s", text)
	}
}

// A builder that changes the spec and removes the base certificate does not begin
// the work again: the certificate is gone, and a plain checklist refuses.
func TestARemovedBaseCertificate(t *testing.T) {
	p := newPlain(t)
	first := p.certify()
	p.list(run.ChecklistRequest{})
	p.specChange()
	p.remove(strings.TrimPrefix(filepath.ToSlash(first), filepath.ToSlash(p.dir)+"/"))
	text, ok, err := p.engine.Checklist(context.Background(), run.ChecklistRequest{Dir: p.dir})
	if err != nil || ok {
		t.Fatalf("a plain checklist began the work again: %v %v\n%s", ok, err, text)
	}
	has(t, text, filepath.Base(first), m+"loan-limit.md", "only the owner begins the work again")
	if v := p.gate(run.GateRequest{}); v.Done {
		t.Fatalf("passed: %s", v.Text)
	}
}

// With no certificate the list holds every Definition, Findings or not, so that
// nothing the builder changes goes unreviewed; and work on Findings is not
// "nothing to check" while the Findings stand.
func TestFindingsWithoutACertificate(t *testing.T) {
	p := newPlain(t)
	findings := []analysis.Finding{{Definition: "loan-limit", Observed: "a fourth loan", Expected: "the limit"}}
	list := p.list(run.ChecklistRequest{Findings: findings})
	has(t, list, "finding 1 loan-limit", "catalog/book")
	v := p.gate(run.GateRequest{})
	has(t, v.Text, run.ReviewerAgent)

	q := newPlain(t)
	q.certify()
	q.list(run.ChecklistRequest{Findings: findings})
	if v := q.gate(run.GateRequest{}); v.Done {
		t.Fatalf("work on Findings passed with nothing reviewed: %s", v.Text)
	}
}

// The newest certificate is the one no other names as its base, whatever the
// clocks said.
func TestTheNewestCertificateFollowsItsBase(t *testing.T) {
	p := newPlain(t)
	p.certify()
	p.specChange()
	p.list(run.ChecklistRequest{})
	p.write("src/members/loans/loan.go", limit3)
	p.review(nil)
	if v := p.gate(run.GateRequest{}); !v.Done {
		t.Fatalf("%s", v.Text)
	}
	certs := p.results(".json")
	var second map[string]any
	if err := json.Unmarshal([]byte(p.read(certs[1])), &second); err != nil {
		t.Fatal(err)
	}
	// Made on a machine whose clock is an hour behind.
	second["created"] = "2026-11-06T11:00:00Z"
	data, _ := json.Marshal(second)
	p.write(".markitect/results/"+filepath.Base(certs[1]), string(data))
	if v := p.gate(run.GateRequest{Verify: true}); !v.Done {
		t.Fatalf("the base won over its successor: %s", v.Text)
	}
}

// A reviewer with no work under way, such as one of a Run that ended, is told to
// end, not to build.
func TestNoWorkForAReviewer(t *testing.T) {
	p := newPlain(t)
	text, ok, err := p.engine.Checklist(context.Background(), run.ChecklistRequest{Dir: p.dir, Review: true})
	if err != nil || ok {
		t.Fatalf("%v %v", ok, err)
	}
	has(t, text, "nothing to review", "End now")
	text, complete, err := p.engine.Answer(context.Background(), p.dir, []byte("loan-limit: ok README.md:1\n"))
	if err != nil || complete {
		t.Fatalf("%v %v", complete, err)
	}
	has(t, text, "nothing is stored", "End now")
}

// gate and answer need a checklist that began the work.
func TestGateWithoutAChecklist(t *testing.T) {
	p := newPlain(t)
	p.certify()
	p.write("src/members/loans/loan.go", limit3)
	v := p.gate(run.GateRequest{})
	has(t, v.Text, "No checklist began this work", "markitect checklist")
	if v.Done {
		t.Fatal("passed")
	}
	text, complete, err := p.engine.Answer(context.Background(), p.dir, []byte("loan-limit: ok README.md:1\n"))
	if err != nil || complete {
		t.Fatalf("%v %v", complete, err)
	}
	has(t, text, "No work is under way")
}

// Row 2: a changed file that lies in no item is named, not certified; a file of a
// Definition the list did not reach puts that Definition on it.
func TestGateRefusesAChangeOutsideTheList(t *testing.T) {
	files := library()
	files[project.IgnoreFile] = "NOTES.md\n"
	files["NOTES.md"] = "# Notes\n"
	p := newPlainOf(t, files)
	first := p.certify()
	p.specChange()
	p.list(run.ChecklistRequest{})
	p.write("src/members/loans/loan.go", limit3)
	p.write("NOTES.md", "# Notes\n\nA side edit.\n")
	v := p.gate(run.GateRequest{})
	has(t, v.Text, "NOTES.md (modified)", "lie in no item", "put them back")
	if v.Done {
		t.Fatal("passed")
	}
	p.write("NOTES.md", "# Notes\n")
	// A .gitignore is the owner's: changed during the work, it stops a plain
	// checklist, and the owner begins the work again by naming the change.
	files[".gitignore"] += "*.tmp\n"
	p.write(".gitignore", files[".gitignore"])
	if text, ok, err := p.engine.Checklist(context.Background(), run.ChecklistRequest{Dir: p.dir}); err != nil || ok || !strings.Contains(text, "only the owner begins the work again") {
		t.Fatalf("a plain checklist went on: %v %v\n%s", ok, err, text)
	}
	p.list(run.ChecklistRequest{Base: first})
	p.write("src/catalog/book.go", "package catalog\n\n// Book is a book.\n")
	has(t, p.gate(run.GateRequest{}).Text, run.ReviewerAgent)
	if keys := strings.Join(reviewKeys(p.list(run.ChecklistRequest{Review: true})), ","); !strings.Contains(keys, "catalog/book") {
		t.Fatalf("the review asks about %s", keys)
	}
}

// Row 3: check, of the spec and the realization.
func TestGateRefusesWhatCheckRefuses(t *testing.T) {
	p := newPlain(t)
	p.certify()
	p.specChange()
	p.list(run.ChecklistRequest{})
	p.write("src/members/stray.go", "package members\n")
	has(t, p.gate(run.GateRequest{}).Text, "markitect check fails", "src/members/stray.go")
	p.remove("src/members/stray.go")
	p.remove("src/catalog/book.go")
	has(t, p.gate(run.GateRequest{}).Text, "markitect check fails", "stale.file", "src/catalog/book.go")
}

// The answers: a bad one is refused at once, and nothing of that call is stored;
// a file the walk does not see does not exist for an answer.
func TestGateAnswerRefusesWhatItCannotAccept(t *testing.T) {
	p := newPlain(t)
	p.list(run.ChecklistRequest{})
	keys := reviewKeys(p.list(run.ChecklistRequest{Review: true}))
	p.write(".claude/worktrees/x/notes.md", "ignored by Git\n")
	ctx := context.Background()
	for _, line := range []string{"loan-limit: ok nosuch/file.go:1", "loan-limit: ok .claude/worktrees/x/notes.md:1"} {
		text, complete, err := p.engine.Answer(ctx, p.dir, []byte(line+"\nproject: ok README.md:1\n"))
		if err != nil || complete {
			t.Fatalf("%v %v", complete, err)
		}
		has(t, text, "not accepted", "does not exist")
	}
	has(t, p.gate(run.GateRequest{}).Text, strconv.Itoa(len(keys))+" items have no answer")
	text, complete, err := p.engine.Answer(ctx, p.dir, []byte("project: ok README.md:1\n"))
	if err != nil || complete {
		t.Fatalf("%v %v", complete, err)
	}
	has(t, text, "still have no answer")
}

// Row 6: wrong answers go back, one round per reviewed state; a repair review asks
// only about what the repair touched, and the third wrong review is refused.
func TestGateRepairRounds(t *testing.T) {
	p := newPlain(t)
	p.specChange()
	wrong := map[string]string{"loan-limit": "wrong src/members/loans/loan.go:3 -- still says 5"}
	p.review(wrong)
	has(t, p.gate(run.GateRequest{}).Text, "repair round 1 of 2", "still says 5", "Repair them")
	// Calling again without a change uses no further round.
	has(t, p.gate(run.GateRequest{}).Text, "repair round 1 of 2")

	p.write("src/members/loans/loan.go", limit3)
	has(t, p.gate(run.GateRequest{}).Text, run.ReviewerAgent)
	prompt := p.list(run.ChecklistRequest{Review: true})
	keys := strings.Join(reviewKeys(prompt), ",")
	if !strings.Contains(keys, "loan-limit") || strings.Contains(keys, "catalog/book") {
		t.Fatalf("a repair review asks about %s", keys)
	}
	has(t, prompt, "An earlier review", "still says 5")

	p.review(wrong)
	has(t, p.gate(run.GateRequest{}).Text, "repair round 2 of 2")
	p.write("src/members/loans/loan.go", "package loans\n\nconst Limit = 4\n")
	has(t, p.gate(run.GateRequest{}).Text, run.ReviewerAgent)
	p.review(wrong)
	v := p.gate(run.GateRequest{})
	if v.Done {
		t.Fatalf("passed: %s", v.Text)
	}
	has(t, v.Text, "after 2 repair rounds", "Stop, and give the owner the Result")
	if len(p.results(".json")) != 0 {
		t.Fatal("a refused gate wrote a certificate")
	}
	pages := p.results(".md")
	has(t, p.read(pages[len(pages)-1]), ": refused", "repair rounds ran out", "## Repair rounds")
}

// A builder that reads the reviewer's report and repairs before gate runs uses a
// round all the same.
func TestGateARepairBeforeGateCountsAsARound(t *testing.T) {
	p := newPlain(t)
	p.specChange()
	wrong := map[string]string{"loan-limit": "wrong src/members/loans/loan.go:3 -- still says 5"}
	p.review(wrong)
	p.write("src/members/loans/loan.go", limit3)
	has(t, p.gate(run.GateRequest{}).Text, run.ReviewerAgent)
	p.review(wrong)
	p.write("src/members/loans/loan.go", "package loans\n\nconst Limit = 4\n")
	has(t, p.gate(run.GateRequest{}).Text, run.ReviewerAgent)
	p.review(wrong)
	p.write("src/members/loans/loan.go", "package loans\n\nconst Limit = 6\n")
	v := p.gate(run.GateRequest{})
	if v.Done {
		t.Fatalf("passed: %s", v.Text)
	}
	has(t, v.Text, "the most the work gets")
}

// Row 5: a spec problem in a finished review stands, even if the builder carries
// on, and comes ahead of any wrong answer, with no round.
func TestGateASpecProblem(t *testing.T) {
	t.Run("stands through a repair", func(t *testing.T) {
		p := newPlain(t)
		p.review(map[string]string{"loan-limit": "spec problem -- the limit contradicts the loan page"})
		p.write("src/members/loans/loan.go", limit3)
		v := p.gate(run.GateRequest{})
		if v.Done {
			t.Fatalf("passed: %s", v.Text)
		}
		has(t, v.Text, "only the owner can settle", "contradicts")
		pages := p.results(".md")
		has(t, p.read(pages[len(pages)-1]), ": spec-problem", "Decisions for the owner", "contradicts")
	})
	t.Run("ahead of a wrong answer", func(t *testing.T) {
		p := newPlain(t)
		p.review(map[string]string{
			"loan-limit": "spec problem -- the limit contradicts the loan page",
			"project":    "wrong README.md:1 -- says nothing",
		})
		has(t, p.gate(run.GateRequest{}).Text, "only the owner can settle")
		pages := p.results(".md")
		if page := p.read(pages[len(pages)-1]); strings.Contains(page, "Repair rounds") {
			t.Fatalf("a round was counted:\n%s", page)
		}
	})
}

// An answer is bound to the files its item names and the files it points to: a
// change to one of them falls it away, and putting the file back brings it back.
func TestGateAnswersFallAwayWhenTheirFilesChange(t *testing.T) {
	p := newPlain(t)
	_, keys := p.review(nil)
	p.write("src/catalog/book.go", "package catalog\n\n// Book is a book.\n")
	has(t, p.gate(run.GateRequest{}).Text, "1 item has no answer")
	if got := reviewKeys(p.list(run.ChecklistRequest{Review: true})); strings.Join(got, ",") != "catalog/book" {
		t.Fatalf("the review asks about %v", got)
	}
	p.write("src/catalog/book.go", "package catalog\n")
	// Every answer points to README.md.
	p.write("README.md", "# Library\n\nBooks and loans.\n")
	has(t, p.gate(run.GateRequest{}).Text, strconv.Itoa(len(keys))+" items have no answer")
	p.write("README.md", "# Library\r\n")
	if v := p.gate(run.GateRequest{}); !v.Done {
		t.Fatalf("the answers did not come back with the file: %s", v.Text)
	}
}

// gate --verify checks the folder against its newest certificate; line endings do
// not count.
func TestGateVerify(t *testing.T) {
	p := newPlain(t)
	has(t, p.gate(run.GateRequest{Verify: true}).Text, "No certificate")
	p.certify()
	if v := p.gate(run.GateRequest{Verify: true}); !v.Done {
		t.Fatalf("%s", v.Text)
	}
	p.write("src/catalog/book.go", "package catalog\r\n")
	if v := p.gate(run.GateRequest{Verify: true}); !v.Done {
		t.Fatalf("line endings counted: %s", v.Text)
	}
	p.write("src/catalog/book.go", "package catalog\n\nvar x = 1\n")
	v := p.gate(run.GateRequest{Verify: true})
	if v.Done {
		t.Fatal("a changed file matched")
	}
	has(t, v.Text, "src/catalog/book.go", "do not match the newest certificate")
	if _, err := p.engine.Gate(context.Background(), run.GateRequest{Dir: p.dir, Verify: true, Findings: true}); !errors.Is(err, run.ErrUsage) {
		t.Fatalf("--verify with --findings: %v", err)
	}
}

// gate --findings closes a review of finished work: the Findings and the Result go
// into the folder, no certificate, and the Findings begin the repair.
func TestGateFindings(t *testing.T) {
	p := newPlain(t)
	first := p.certify()
	p.write("src/members/loans/loan.go", "package loans\n\nconst Limit = 9\n")
	has(t, p.gate(run.GateRequest{Findings: true}).Text, "No checklist began this review")
	has(t, p.list(run.ChecklistRequest{Base: first, Review: true}), "loan-limit")
	has(t, p.gate(run.GateRequest{Findings: true}).Text, "have no answer", "markitect gate --findings again")
	p.review(map[string]string{"loan-limit": "wrong src/members/loans/loan.go:3 -- says 9; the Definition says 5"})
	v := p.gate(run.GateRequest{Findings: true})
	if !v.Done {
		t.Fatalf("%s", v.Text)
	}
	has(t, v.Text, "findings", "there is no certificate", "markitect checklist --findings")
	if certs := p.results(".json"); len(certs) != 1 {
		t.Fatalf("certificates %v", certs)
	}
	found, verified := p.results("-findings.md"), p.results("-verify.md")
	if len(found) != 1 || len(verified) != 1 {
		t.Fatalf("pages %v %v", found, verified)
	}
	data := p.read(found[0])
	has(t, data, "[[loan-limit]]", "says 9")
	findings, errs := analysis.ParseFindings(found[0], []byte(data))
	if len(errs) != 0 || len(findings) != 1 {
		t.Fatalf("%v %v", findings, errs)
	}
	has(t, p.list(run.ChecklistRequest{Findings: findings}), "finding 1 loan-limit", "says 9")
}

// The skill that drives the work begins it with checklist --start, which names the
// agent to start; called again, it goes on with the work under way. A builder that
// is started again sees what the last review found wrong.
func TestChecklistStartsTheBuilder(t *testing.T) {
	p := newPlain(t)
	p.certify()
	p.specChange()
	start := p.list(run.ChecklistRequest{Start: true})
	has(t, start, "The base is the certificate", "The list has", run.BuilderAgent, "Build the work.", "run markitect gate and do what it prints")
	if strings.Contains(start, "Make this project match the spec") {
		t.Fatalf("the driver got the builder's task:\n%s", start)
	}
	has(t, p.list(run.ChecklistRequest{Start: true}), run.BuilderAgent)
	p.review(map[string]string{"loan-limit": "wrong src/members/loans/loan.go:3 -- still says 5"})
	has(t, p.gate(run.GateRequest{}).Text, "repair round 1 of 2")
	has(t, p.list(run.ChecklistRequest{}), "The last review found these wrong", "- loan-limit: still says 5")
	has(t, p.list(run.ChecklistRequest{Start: true}), run.BuilderAgent, "Build the work.")
}

// A review of finished work begins with checklist --review --start, which names the
// reviewer; with nothing to review it leaves no work behind.
func TestChecklistStartsTheReviewer(t *testing.T) {
	p := newPlain(t)
	p.certify()
	has(t, p.list(run.ChecklistRequest{Review: true, Start: true}), "Nothing to review")
	// No work was left behind: a change of the spec now begins new work.
	p.specChange()
	has(t, p.list(run.ChecklistRequest{}), "Make this project match the spec")
	q := newPlain(t)
	q.certify()
	q.write("src/members/loans/loan.go", "package loans\n\nconst Limit = 9\n")
	has(t, q.list(run.ChecklistRequest{Review: true, Start: true, Full: true}), "every Definition is on the list", run.ReviewerAgent, "Review the work.", "markitect gate --findings")
	q.review(map[string]string{"loan-limit": "wrong src/members/loans/loan.go:3 -- says 9"})
	has(t, q.list(run.ChecklistRequest{Review: true, Start: true}), "Every item has an answer", "markitect gate --findings")
	if v := q.gate(run.GateRequest{Findings: true}); !v.Done || !strings.Contains(v.Text, "Findings") {
		t.Fatalf("%+v", v)
	}
}

// Of the certificates in the folder, as after a merge of two branches, the newest
// counts.
func TestTheNewestCertificateCounts(t *testing.T) {
	p := newPlain(t)
	first := p.certify()
	p.specChange()
	p.list(run.ChecklistRequest{})
	p.write("src/members/loans/loan.go", limit3)
	p.review(nil)
	if v := p.gate(run.GateRequest{}); !v.Done {
		t.Fatalf("%s", v.Text)
	}
	// An older certificate from another branch arrives with a later name.
	p.write(".markitect/results/gate-99991231-235959-ffffffff.json", p.read(first))
	if v := p.gate(run.GateRequest{Verify: true}); !v.Done {
		t.Fatalf("the older certificate counted: %s", v.Text)
	}
	has(t, p.list(run.ChecklistRequest{}), "The base is the certificate .markitect/results/gate-20261106")
}

// While a Run is open in the folder, its work is the Run's; status adds the work
// without a Run and changes nothing.
func TestGateBesideARun(t *testing.T) {
	t.Run("an open run", func(t *testing.T) {
		f := newFixture(t)
		f.engine.Folder, f.engine.Cache = folders.Folder{}, testkit.TempDir(t)
		s, wt := f.apply()
		v, err := f.engine.Gate(context.Background(), run.GateRequest{Dir: wt})
		if err != nil || v.Done {
			t.Fatalf("%v %+v", err, v)
		}
		has(t, v.Text, "Run "+s.Run+" is open")
		if _, _, err := f.engine.Checklist(context.Background(), run.ChecklistRequest{Dir: wt, Full: true}); !errors.Is(err, run.ErrUsage) {
			t.Fatalf("checklist --full during a run: %v", err)
		}
		text, _, err := f.engine.Checklist(context.Background(), run.ChecklistRequest{Dir: wt})
		if err != nil {
			t.Fatal(err)
		}
		has(t, text, "You build Markitect run "+s.Run, "markitect submit")
		text, ok, err := f.engine.Checklist(context.Background(), run.ChecklistRequest{Dir: wt, Start: true})
		if err != nil || ok {
			t.Fatalf("checklist --start during a run: %v %v", ok, err)
		}
		has(t, text, "Run "+s.Run+" is open", "no work begins beside it")
	})
	t.Run("status", func(t *testing.T) {
		f := newFixture(t)
		f.engine.Folder, f.engine.Cache = folders.Folder{}, testkit.TempDir(t)
		ctx := context.Background()
		text, _, err := f.engine.Checklist(ctx, run.ChecklistRequest{Dir: f.repo.Dir})
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		for _, k := range reviewKeys(text) {
			lines = append(lines, k+": ok README.md:1")
		}
		if _, _, err := f.engine.Answer(ctx, f.repo.Dir, []byte(strings.Join(lines, "\n"))); err != nil {
			t.Fatal(err)
		}
		before := f.repo.Git("status", "--porcelain")
		status, err := f.engine.Status(ctx, f.repo.Dir)
		if err != nil {
			t.Fatal(err)
		}
		has(t, status, "No run on this branch yet", "Without a run", "0 without, 0 wrong", "No certificate yet")
		if again, _ := f.engine.Status(ctx, f.repo.Dir); again != status {
			t.Fatalf("status changed what it reads:\n%s\n%s", status, again)
		}
		if after := f.repo.Git("status", "--porcelain"); after != before {
			t.Fatalf("status changed the folder: %s", after)
		}
	})
}
