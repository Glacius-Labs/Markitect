package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

func run(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(context.Background(), append(args[:1:1], append([]string{"--repo", dir}, args[1:]...)...), &out, &errOut)
	return code, out.String(), errOut.String()
}

func mustRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	code, out, errOut := run(t, dir, args...)
	if code != 0 {
		t.Fatalf("markitect %s: exit %d\n%s%s", strings.Join(args, " "), code, out, errOut)
	}
	return out
}

func read(t *testing.T, dir, file string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestCommandsEndToEnd(t *testing.T) {
	r := testkit.NewRepo(t)
	r.Write("README.md", "# Lending library\n")
	r.Write("src/members/member.go", "package members\n")

	out := mustRun(t, r.Dir, "init", "--name", "Lending library")
	for _, want := range []string{"created .markitect/manifest.yaml", "created .markitect/model/project.md", "wrote .markitect/lock.yaml: std 1.0.0", "installed the skills markitect-extract, markitect-modeling"} {
		if !strings.Contains(out, want) {
			t.Fatalf("init output lacks %q:\n%s", want, out)
		}
	}
	if !strings.Contains(read(t, r.Dir, ".claude/skills/markitect-modeling/SKILL.md"), "markitect-modeling") || !strings.Contains(read(t, r.Dir, ".agents/skills/markitect-extract/SKILL.md"), "markitect-extract") {
		t.Fatal("skills not installed")
	}
	if !strings.Contains(read(t, r.Dir, ".markitect/model/project.md"), "# Lending library") {
		t.Fatal("project anchor lacks the title")
	}

	code, out, _ := run(t, r.Dir, "check")
	if code != 1 || !strings.Contains(out, "README.md: file.unregistered") || !strings.Contains(out, "src/members/member.go: file.unregistered") {
		t.Fatalf("check before registration: exit %d\n%s", code, out)
	}

	r.Write(".markitect/model/loan-limit.md", "---\nkind: std/Rule\n---\n# Loan limit\n\nAt most `5` open loans, for [[members]].\n")
	r.Write(".markitect/model/members/member.md", "---\nkind: std/Concept\n---\n# Member\n\nA person who borrows.\n")
	r.Write(".markitect/registry.md", read(t, r.Dir, ".markitect/registry.md")+"| `README.md` | [[project]] | Overview |\n")
	r.Write(".markitect/registry/members.md", "---\npaths: [src/members/**]\n---\n# Registry: members\n\n| File | Realizes | Why |\n|---|---|---|\n| `src/members/member.go` | [[members/member]] | The type |\n")
	if out := mustRun(t, r.Dir, "check"); !strings.HasPrefix(out, "ok: 3 Definitions in 2 Packages") {
		t.Fatalf("check = %s", out)
	}
	code, out, _ = run(t, r.Dir, "check", "--json")
	var checked struct {
		OK     bool
		Errors []any
	}
	if code != 0 || json.Unmarshal([]byte(out), &checked) != nil || !checked.OK || checked.Errors == nil {
		t.Fatalf("check --json: %d %s", code, out)
	}
	base := r.Commit("model")

	// init again changes nothing in the model and keeps the manifest.
	before := read(t, r.Dir, ".markitect/manifest.yaml")
	mustRun(t, r.Dir, "init")
	if read(t, r.Dir, ".markitect/manifest.yaml") != before || r.Git("status", "--porcelain") != "" {
		t.Fatalf("a second init changed files: %q", r.Git("status", "--porcelain"))
	}

	r.Write(".markitect/model/loan-limit.md", "---\nkind: std/Rule\n---\n# Loan limit\n\nAt most `3` open loans, for [[members]].\n")
	findings := filepath.Join(testkit.TempDir(t), "findings.md")
	if err := os.WriteFile(findings, []byte("## [[members/member]]\n\nObserved: no card number.\n\nExpected: a card number.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out = mustRun(t, r.Dir, "impact", "--base", base, "--json", "--findings", findings)
	var im struct {
		Base       string
		Packages   []string
		Checklists []struct {
			Package   string
			Own       []struct{ Definition string }
			Inherited []struct{ Definition string }
			Findings  []struct{ Definition string }
		}
	}
	if err := json.Unmarshal([]byte(out), &im); err != nil {
		t.Fatal(err, out)
	}
	if im.Base != base || strings.Join(im.Packages, ",") != ",members" || len(im.Checklists) != 2 {
		t.Fatalf("impact = %s", out)
	}
	members := im.Checklists[1]
	if members.Package != "members" || len(members.Inherited) != 1 || members.Inherited[0].Definition != "loan-limit" || len(members.Findings) != 1 {
		t.Fatalf("members checklist = %+v", members)
	}
	md := mustRun(t, r.Dir, "impact", "--base", "HEAD")
	if !strings.Contains(md, "- [[loan-limit]] modified") || !strings.Contains(md, "### members") {
		t.Fatalf("impact Markdown:\n%s", md)
	}

	if out := mustRun(t, r.Dir, "refs", "members/member"); !strings.Contains(out, "`src/members/member.go` | registry") {
		t.Fatalf("refs:\n%s", out)
	}
	if out := mustRun(t, r.Dir, "graph"); !strings.Contains(out, "```mermaid") {
		t.Fatalf("graph:\n%s", out)
	}
	if out := mustRun(t, r.Dir, "lint"); !strings.Contains(out, "lint.unused") || !strings.Contains(out, "loan-limit") {
		t.Fatalf("lint:\n%s", out)
	}

	// impact refuses a model that does not compile, before reading the base.
	r.Write(".markitect/model/broken.md", "no front matter\n")
	code, _, errOut := run(t, r.Dir, "impact", "--base", base)
	if code != 1 || !strings.Contains(errOut, "impact needs a model that compiles") {
		t.Fatalf("impact on errors: %d %s", code, errOut)
	}
	code, _, errOut = run(t, r.Dir, "lint")
	if code != 0 || !strings.Contains(errOut, "compile errors") {
		t.Fatalf("lint on errors: %d %s", code, errOut)
	}
}

func TestLockCommand(t *testing.T) {
	r := testkit.NewRepo(t)
	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.0.0\n  rest: 1.0.0\n  web: 0.1.0\n")
	code, _, errOut := run(t, r.Dir, "lock")
	if code != 1 || !strings.Contains(errOut, "library.missing") {
		t.Fatalf("lock without web: %d %s", code, errOut)
	}
	r.Write(".markitect/libraries/web/Page.md", "---\nkind: Kind\n---\n# Page\n")
	if out := mustRun(t, r.Dir, "lock"); !strings.Contains(out, "rest 1.0.0, std 1.0.0, web 0.1.0") {
		t.Fatalf("lock = %s", out)
	}
	if lock := read(t, r.Dir, ".markitect/lock.yaml"); !strings.Contains(lock, "  web:\n    version: 0.1.0\n    digest: sha256:") {
		t.Fatalf("lock file:\n%s", lock)
	}
}

func TestUsageAndRefusals(t *testing.T) {
	var out, errOut bytes.Buffer
	if Run(context.Background(), nil, &out, &errOut) != 0 || !strings.Contains(out.String(), "markitect impact") {
		t.Fatalf("usage = %s", out.String())
	}
	if Run(context.Background(), []string{"apply"}, &out, &errOut) != 2 || !strings.Contains(errOut.String(), `unknown command "apply"`) {
		t.Fatalf("unknown command: %s", errOut.String())
	}
	dir := testkit.TempDir(t)
	if code, _, errOut := run(t, dir, "check"); code != 2 || !strings.Contains(errOut, "not inside a Git repository") {
		t.Fatalf("check outside Git: %d %s", code, errOut)
	}
	r := testkit.NewRepo(t)
	if code, _, errOut := run(t, r.Dir, "impact"); code != 2 || !strings.Contains(errOut, "needs --base") {
		t.Fatalf("impact without base: %d %s", code, errOut)
	}
	if code, out, _ := run(t, r.Dir, "check"); code != 1 || !strings.Contains(out, "manifest.missing") {
		t.Fatalf("check without init: %d %s", code, out)
	}
	out.Reset()
	if Run(context.Background(), []string{"version"}, &out, &errOut) != 0 || !strings.HasPrefix(out.String(), "markitect ") {
		t.Fatalf("version = %s", out.String())
	}
}
