package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/upgrade"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) {
	// No test reaches the real GitHub: a release page is empty unless a test
	// serves one.
	os.Setenv(upgrade.BaseEnv, httptest.NewServer(http.NotFoundHandler()).URL)
	testkit.Main(m)
}

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
	for _, want := range []string{"created .markitect/manifest.yaml", "created .markitect/model/project.md", "wrote .markitect/lock.yaml: std 1.0.0", "installed the skills markitect-extract, markitect-modeling, markitect-start"} {
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
	r.Write(".markitect/registry.md", read(t, r.Dir, ".markitect/registry.md")+"| `README.md` | [[project]], [[loan-limit]] | Overview |\n")
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
	mustRun(t, r.Dir, "lint")

	// impact refuses a model that does not compile, before reading the base.
	r.Write(".markitect/model/broken.md", "no front matter\n")
	code, _, errOut := run(t, r.Dir, "impact", "--base", base)
	if code != 1 || !strings.Contains(errOut, "impact needs a spec that compiles") {
		t.Fatalf("impact on errors: %d %s", code, errOut)
	}
	code, _, errOut = run(t, r.Dir, "lint")
	if code != 0 || !strings.Contains(errOut, "the spec has") {
		t.Fatalf("lint on errors: %d %s", code, errOut)
	}
}

// check tells the spec from the realization, and --spec looks only at the spec.
func TestCheckGroupsItsErrors(t *testing.T) {
	r := testkit.NewRepo(t)
	mustRun(t, r.Dir, "init", "--name", "Demo")
	r.Write("src/stray.go", "package stray\n")
	code, out, _ := run(t, r.Dir, "check")
	if code != 1 || !strings.Contains(out, "In the realization") || !strings.Contains(out, "file.unregistered") {
		t.Fatalf("check on a project with a stray file: %d\n%s", code, out)
	}
	if out := mustRun(t, r.Dir, "check", "--spec"); !strings.HasPrefix(out, "ok:") {
		t.Fatalf("check --spec: %s", out)
	}
	r.Write(".markitect/model/broken.md", "no front matter\n")
	code, out, _ = run(t, r.Dir, "check", "--spec")
	if code != 1 || !strings.Contains(out, "In the spec") {
		t.Fatalf("check --spec on a broken spec: %d\n%s", code, out)
	}
}

// A new project passes check, in plain words. As soon as there is a file, it is
// held to account as ever, and so is a Definition once anything is registered.
func TestCheckWithNothingBuilt(t *testing.T) {
	r := testkit.NewRepo(t)
	mustRun(t, r.Dir, "init", "--name", "Demo")
	code, out, _ := run(t, r.Dir, "check")
	if code != 0 || !strings.HasPrefix(out, "ok:") || !strings.Contains(out, "Nothing is built yet") || !strings.Contains(out, "markitect apply") || strings.Contains(out, "definition.unused") {
		t.Fatalf("check on a new project: %d\n%s", code, out)
	}
	code, out, _ = run(t, r.Dir, "check", "--json")
	var checked struct {
		OK           bool
		NothingBuilt bool
		Errors       []any
	}
	if code != 0 || json.Unmarshal([]byte(out), &checked) != nil || !checked.OK || !checked.NothingBuilt || len(checked.Errors) != 0 {
		t.Fatalf("check --json on a new project: %d %s", code, out)
	}
	if out := mustRun(t, r.Dir, "check", "--spec"); strings.Contains(out, "Nothing is built") {
		t.Fatalf("--spec looks only at the spec: %s", out)
	}

	r.Write("src/a.go", "package a\n")
	code, out, _ = run(t, r.Dir, "check")
	if code != 1 || !strings.Contains(out, "file.unregistered") || strings.Contains(out, "Nothing is built") {
		t.Fatalf("check with an unlisted file: %d\n%s", code, out)
	}

	// What an extraction leaves: files registered, no Run landed yet.
	r.Write(".markitect/registry.md", read(t, r.Dir, ".markitect/registry.md")+"| `src/a.go` | [[project]] | The code |\n")
	r.Write(".markitect/model/rule.md", "---\nkind: std/Rule\n---\n# Rule\n\nAlways.\n")
	code, out, _ = run(t, r.Dir, "check")
	if code != 1 || !strings.Contains(out, "definition.unused") || strings.Contains(out, "Nothing is built") {
		t.Fatalf("check with a registered file: %d\n%s", code, out)
	}
}

// Help and init explain Markitect, since a session that runs init loaded nothing
// of the project.
func TestTheToolExplainsMarkitect(t *testing.T) {
	const about = "A Markitect project has one spec"
	var out, errOut bytes.Buffer
	if Run(context.Background(), []string{"--help"}, &out, &errOut) != 0 || !strings.HasPrefix(out.String(), about) {
		t.Fatalf("help = %s", out.String())
	}
	r := testkit.NewRepo(t)
	said := mustRun(t, r.Dir, "init", "--name", "Demo")
	if !strings.Contains(said, about) || !strings.Contains(said, "/reload-skills") {
		t.Fatalf("init said:\n%s", said)
	}
	agents := read(t, r.Dir, "AGENTS.md")
	if !strings.Contains(agents, about) || !strings.Contains(agents, "`markitect help`") {
		t.Fatalf("AGENTS.md:\n%s", agents)
	}
}

// install writes the personal skill from the binary, and init stays out of the
// home folder.
func TestInstallWritesThePersonalSkill(t *testing.T) {
	home := testkit.TempDir(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	r := testkit.NewRepo(t)
	mustRun(t, r.Dir, "init")
	if _, err := os.Stat(filepath.Join(home, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("init wrote to the home folder: %v", err)
	}

	install := func() string {
		t.Helper()
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), []string{"install"}, &out, &errOut); code != 0 {
			t.Fatalf("install: %d %s", code, errOut.String())
		}
		return out.String()
	}
	file := filepath.Join(home, ".claude", "skills", "markitect", "SKILL.md")
	if said := install(); !strings.Contains(said, file) {
		t.Fatalf("install does not say where: %s", said)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	skill := string(data)
	for _, want := range []string{"name: markitect\n", "A Markitect project has one spec", "no need to look for it", "markitect-start", "markitect-extract", "/reload-skills", "markitect update", "claude plugin marketplace add Glacius-Labs/Claude-Plugins", "claude plugin install markitect@glacius-labs", "github.com/Glacius-Labs/Markitect/releases"} {
		if !strings.Contains(skill, want) {
			t.Errorf("the skill lacks %q:\n%s", want, skill)
		}
	}
	if strings.Contains(skill, "allowed-tools") {
		t.Errorf("the skill allows commands in advance:\n%s", skill)
	}
	// Run again, it refreshes the text.
	if err := os.WriteFile(file, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	install()
	if again, _ := os.ReadFile(file); string(again) != skill {
		t.Fatalf("install did not refresh: %s", again)
	}
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"install", "extra"}, &out, &errOut); code != 2 {
		t.Fatalf("install with an argument: %d", code)
	}
}

// init installs the harness with the hook on answer and commits what it wrote, so
// that the first apply finds a clean folder.
// The hook joins settings the project already has, and is added once.
func TestInitKeepsTheProjectsSettings(t *testing.T) {
	r := testkit.NewRepo(t)
	r.Write(".claude/settings.json", "{\n  \"permissions\": {\"allow\": [\"Bash(go test *)\"]},\n  \"cleanupPeriodDays\": 12345678901234567890\n}\n")
	mustRun(t, r.Dir, "init")
	mustRun(t, r.Dir, "init")
	settings := read(t, r.Dir, ".claude/settings.json")
	if !strings.Contains(settings, "Bash(go test *)") || !strings.Contains(settings, "12345678901234567890") ||
		strings.Count(settings, "Bash(markitect answer*)") != 1 || strings.Count(settings, "Bash(markitect checklist*)") != 1 {
		t.Fatalf("settings:\n%s", settings)
	}
}

// A mention of the hook's command that is not the hook gets the hook added.
func TestInitAddsTheHookBesideAMention(t *testing.T) {
	r := testkit.NewRepo(t)
	r.Write(".claude/settings.json", `{"hooks": {"PostToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "markitect hook pretool"}]}]}}`+"\n")
	mustRun(t, r.Dir, "init")
	if !strings.Contains(read(t, r.Dir, ".claude/settings.json"), "PreToolUse") {
		t.Fatalf("no hook added:\n%s", read(t, r.Dir, ".claude/settings.json"))
	}
}

// Settings init cannot read, or whose hooks have another shape, stay as they are,
// and init says what to add.
func TestInitLeavesSettingsItCannotRead(t *testing.T) {
	for name, content := range map[string]string{
		"not JSON":              "{ not json\n",
		"hooks not a map":       `{"hooks": ["something"]}` + "\n",
		"PreToolUse not a list": `{"hooks": {"PreToolUse": "x"}}` + "\n",
		"null":                  "null\n",
		"two values":            "{} {}\n",
	} {
		t.Run(name, func(t *testing.T) {
			r := testkit.NewRepo(t)
			r.Write(".claude/settings.json", content)
			out := mustRun(t, r.Dir, "init")
			if !strings.Contains(out, "left .claude/settings.json as it is") {
				t.Fatalf("init said:\n%s", out)
			}
			if got := read(t, r.Dir, ".claude/settings.json"); got != content {
				t.Fatalf("settings changed:\n%s", got)
			}
		})
	}
}

// The hook runs the installed command, and only the reviewer may answer.
func TestHookLetsOnlyTheReviewerAnswer(t *testing.T) {
	hook := func(agent string) string {
		input, _ := json.Marshal(map[string]any{"agent_type": agent, "tool_name": "Bash", "tool_input": map[string]any{"command": "markitect answer"}})
		var out, errOut bytes.Buffer
		code := cli.CLI{Git: git{}, Runs: runs{}, Stdin: bytes.NewReader(input), Stdout: &out, Stderr: &errOut}.Run(context.Background(), []string{"hook", "pretool"})
		if code != 0 {
			t.Fatalf("hook exit %d: %s", code, errOut.String())
		}
		return out.String()
	}
	if out := hook("markitect-builder"); !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("the builder may answer: %s", out)
	}
	if out := hook("markitect-reviewer"); out != "" {
		t.Fatalf("the reviewer may not: %s", out)
	}
}

func TestInitInstallsTheHarness(t *testing.T) {
	r := testkit.NewRepo(t)
	r.Git("config", "user.name", "Owner")
	r.Git("config", "user.email", "owner@example.invalid")
	out := mustRun(t, r.Dir, "init", "--name", "Demo")
	if !strings.Contains(out, "installed the harness") || !strings.Contains(out, "committed what init wrote") {
		t.Fatalf("init: %s", out)
	}
	for _, file := range []string{".claude/skills/markitect-apply/SKILL.md", ".claude/skills/markitect-verify/SKILL.md", ".claude/agents/markitect-reviewer.md", ".claude/agents/markitect-builder.md"} {
		if read(t, r.Dir, file) == "" {
			t.Fatalf("%s is empty", file)
		}
	}
	var settings struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []struct{ Type, If, Command string }
		}
	}
	if err := json.Unmarshal([]byte(read(t, r.Dir, ".claude/settings.json")), &settings); err != nil {
		t.Fatal(err)
	}
	if pre := settings.Hooks["PreToolUse"]; len(pre) != 1 || pre[0].Matcher != "Bash" || len(pre[0].Hooks) != 2 ||
		pre[0].Hooks[0].Command != "markitect hook pretool" || pre[0].Hooks[0].If != "Bash(markitect answer*)" ||
		pre[0].Hooks[1].Command != "markitect hook pretool" || pre[0].Hooks[1].If != "Bash(markitect checklist*)" {
		t.Fatalf("the guard hook: %+v", settings)
	}
	if status := r.Git("status", "--porcelain"); status != "" {
		t.Fatalf("init left the folder dirty: %s", status)
	}
	code, checked, _ := run(t, r.Dir, "check")
	if code != 0 || strings.Contains(checked, "file.unregistered") {
		t.Fatalf("the harness needs no registry entry: %s", checked)
	}
	// Run again it changes nothing, and commits nothing it did not write.
	r.Write(".markitect/model/project.md", "---\nkind: std/Project\n---\n# Demo\n\nBeing edited.\n")
	r.Write(".claude/agents/mine.md", "private\n")
	before := r.Git("rev-parse", "HEAD")
	mustRun(t, r.Dir, "init", "--name", "Demo")
	if got := r.Git("rev-parse", "HEAD"); got != before {
		t.Fatalf("a second init committed: %s", r.Git("show", "--stat", "HEAD"))
	}
	if status := r.Git("status", "--porcelain"); !strings.Contains(status, "project.md") || !strings.Contains(status, "mine.md") {
		t.Fatalf("what the owner is editing changed: %s", status)
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
	if Run(context.Background(), []string{"bughunt"}, &out, &errOut) != 2 || !strings.Contains(errOut.String(), `unknown command "bughunt"`) {
		t.Fatalf("unknown command: %s", errOut.String())
	}
	dir := testkit.TempDir(t)
	if code, _, errOut := run(t, dir, "check"); code != 2 || !strings.Contains(errOut, "not inside a Git repository") {
		t.Fatalf("check outside Git: %d %s", code, errOut)
	}
	r := testkit.NewRepo(t)
	if code, _, errOut := run(t, r.Dir, "impact"); code != 1 || !strings.Contains(errOut, "needs a spec that compiles") {
		t.Fatalf("impact without a project: %d %s", code, errOut)
	}
	if code, out, _ := run(t, r.Dir, "check"); code != 1 || !strings.Contains(out, "manifest.missing") {
		t.Fatalf("check without init: %d %s", code, out)
	}
	out.Reset()
	if Run(context.Background(), []string{"version"}, &out, &errOut) != 0 || !strings.HasPrefix(out.String(), "markitect ") {
		t.Fatalf("version = %s", out.String())
	}
}

// Without a run, checklist, answer and gate work on the project folder: the work
// passes into a certificate, which impact and gate --verify then read.
func TestGateThroughTheCommandLine(t *testing.T) {
	r := testkit.NewRepo(t)
	r.Git("config", "user.name", "Owner")
	r.Git("config", "user.email", "owner@example.invalid")
	mustRun(t, r.Dir, "init")
	r.Write(".markitect/registry.md", "| File | Realizes | Why |\n|---|---|---|\n| README.md | [[project]] | Overview |\n")
	r.Write("README.md", "# A project\n")
	if code, out, _ := run(t, r.Dir, "gate"); code != 1 || !strings.Contains(out, "No checklist began this work") {
		t.Fatalf("gate before checklist: %d\n%s", code, out)
	}
	// The skill's call always exits 0, so that its session sees what it printed.
	if code, out, _ := run(t, r.Dir, "checklist", "--start", "--findings", "nosuch.md"); code != 0 || !strings.Contains(out, "could not begin the work") {
		t.Fatalf("checklist --start with a missing Findings page: %d\n%s", code, out)
	}
	wantAll(t, mustRun(t, r.Dir, "checklist", "--start"), "markitect-builder", "Build the work.")
	wantAll(t, mustRun(t, r.Dir, "checklist"), "Make this project match the spec", "markitect gate", "[ ] project")
	if code, out, _ := run(t, r.Dir, "gate"); code != 1 || !strings.Contains(out, "markitect-reviewer") {
		t.Fatalf("gate before a review: %d\n%s", code, out)
	}
	// The builder's report of a spec problem says something, with one dash or two.
	for _, flag := range []string{"--spec-problem", "-spec-problem"} {
		if code, _, errOut := run(t, r.Dir, "gate", flag, " "); code != 2 || !strings.Contains(errOut, "what the problem is") {
			t.Fatalf("gate %s with no text: %d %s", flag, code, errOut)
		}
	}
	var out, errOut bytes.Buffer
	answer := cli.CLI{Git: git{}, Runs: runs{}, Stdin: strings.NewReader("project: ok README.md:1\n"), Stdout: &out, Stderr: &errOut}
	if code := answer.Run(context.Background(), []string{"answer", "--repo", r.Dir}); code != 0 {
		t.Fatalf("answer: %d\n%s%s", code, out.String(), errOut.String())
	}
	passed := mustRun(t, r.Dir, "gate")
	wantAll(t, passed, "The work passes", "Commit the work together with both")
	cert := ""
	for _, f := range strings.Fields(passed) {
		if f = strings.TrimSuffix(f, ","); strings.HasSuffix(f, ".json") {
			cert = f
		}
	}
	if cert == "" {
		t.Fatalf("no certificate named:\n%s", passed)
	}
	wantAll(t, mustRun(t, r.Dir, "gate", "--verify"), "matches its newest certificate")
	r.Write("README.md", "# A project, changed\n")
	if code, out, _ := run(t, r.Dir, "gate", "--verify"); code != 1 || !strings.Contains(out, "README.md") {
		t.Fatalf("gate --verify after a change: %d\n%s", code, out)
	}
	wantAll(t, mustRun(t, r.Dir, "impact", "--base", filepath.Join(r.Dir, filepath.FromSlash(cert))), "README.md")
	if code, _, errOut := run(t, r.Dir, "checklist", "--base", "nosuch.json"); code != 2 || !strings.Contains(errOut, "is no certificate") {
		t.Fatalf("checklist --base nosuch.json: %d %s", code, errOut)
	}
	if code, _, errOut := run(t, r.Dir, "gate", "--verify", "--findings"); code != 2 || !strings.Contains(errOut, "not both") {
		t.Fatalf("gate --verify --findings: %d %s", code, errOut)
	}
}
