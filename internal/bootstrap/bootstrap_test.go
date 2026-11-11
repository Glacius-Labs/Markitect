package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
	r := newFolder(t)
	r.Write("README.md", "# Lending library\n")
	r.Write("src/members/member.go", "package members\n")

	out := mustRun(t, r.Dir, "init", "--name", "Lending library")
	noRepository(t, r.Dir)
	for _, want := range []string{"created .markitect/manifest.yaml", "created .markitect/spec/project.md", "wrote .markitect/lock.yaml: std 1.1.0", "installed the harness in .claude/: the architect, build"} {
		if !strings.Contains(out, want) {
			t.Fatalf("init output lacks %q:\n%s", want, out)
		}
	}
	if !strings.Contains(read(t, r.Dir, ".claude/skills/markitect-architect/SKILL.md"), "## Who writes") || !strings.Contains(read(t, r.Dir, ".claude/skills/markitect-architect/reference.md"), "spec form") {
		t.Fatal("skills not installed")
	}
	if !strings.Contains(read(t, r.Dir, ".markitect/spec/project.md"), "# Lending library") {
		t.Fatal("project anchor lacks the title")
	}

	code, out, _ := run(t, r.Dir, "check")
	if code != 1 || !strings.Contains(out, "README.md: file.unregistered") || !strings.Contains(out, "src/members/member.go: file.unregistered") {
		t.Fatalf("check before registration: exit %d\n%s", code, out)
	}

	r.Write(".markitect/spec/loan-limit.md", "---\nkind: std/Rule\n---\n# Loan limit\n\nAt most `5` open loans, for [[members]].\n")
	r.Write(".markitect/spec/members/member.md", "---\nkind: std/Concept\n---\n# Member\n\nA person who borrows.\n")
	r.Write(".markitect/registry.md", read(t, r.Dir, ".markitect/registry.md")+"| `README.md` | [[project]], [[loan-limit]] | Overview |\n")
	r.Write(".markitect/registry/members.md", "---\npaths: [src/members/**]\n---\n# Registry: members\n\n| File | Realizes | Why |\n|---|---|---|\n| `src/members/member.go` | [[members/member]] | The type |\n")
	if out := mustRun(t, r.Dir, "check"); !strings.HasPrefix(out, "ok: 3 Definitions in 2 Packages") {
		t.Fatalf("check = %s", out)
	}
	code, out, _ = run(t, r.Dir, "check", "--json")
	var checked struct {
		FormatVersion int `json:"format_version"`
		Kind          string
		OK            bool
		Diagnostics   []any
	}
	if code != 0 || json.Unmarshal([]byte(out), &checked) != nil || checked.FormatVersion != 1 || checked.Kind != "check" || !checked.OK || checked.Diagnostics == nil {
		t.Fatalf("check --json: %d %s", code, out)
	}
	// With no record, every Definition is to verify.
	code, out, _ = run(t, r.Dir, "status")
	if code != 1 || !strings.Contains(out, "To verify (3 of 3 Definitions)") || !strings.Contains(out, "\nloan-limit@") {
		t.Fatalf("status before a review: %d\n%s", code, out)
	}
	verifyAll(t, r.Dir)

	// init again changes nothing.
	before := snapshot(t, r.Dir)
	mustRun(t, r.Dir, "init")
	if diff := changed(before, snapshot(t, r.Dir)); len(diff) > 0 {
		t.Fatalf("a second init changed files: %v", diff)
	}

	// A changed Rule is to verify again, and nothing else is.
	r.Write(".markitect/spec/loan-limit.md", "---\nkind: std/Rule\n---\n# Loan limit\n\nAt most `3` open loans, for [[members]].\n")
	code, out, _ = run(t, r.Dir, "status")
	if code != 1 || !strings.Contains(out, "To verify (1 of 3 Definitions)") || !strings.Contains(out, "\nloan-limit@") || !strings.Contains(out, "changed: self\n") {
		t.Fatalf("status after a change of the spec: %d\n%s", code, out)
	}

	// doc shows a Definition with its text, its links both ways, its files and its
	// record; and a Package.
	wantAll(t, mustRun(t, r.Dir, "doc", "[[members/member]]"), "members/member: Member (std/Concept)\n", "Package members, .markitect/spec/members/member.md\n", "A person who borrows.", "Realized by:\n  src/members/member.go, .markitect/registry/members.md:", "Record: members/member@", ", ok")
	wantAll(t, mustRun(t, r.Dir, "doc", "loan-limit"), "Links to: members\n", "Realized by:\n  README.md", ", changed: self")
	wantAll(t, mustRun(t, r.Dir, "doc", "members"), "Package members, in Root\n", "paths src/members/**", "members/member: Member (std/Concept)", "Linked from:\n  loan-limit (link)", "Files:\n  src/members/member.go")
	if code, _, errOut := run(t, r.Dir, "doc", "members/nobody"); code != 1 || !strings.Contains(errOut, "neither a Definition nor a Package") {
		t.Fatalf("doc of nothing: %d %s", code, errOut)
	}
	var doc struct {
		Kind       string
		Definition struct {
			ID         string
			Text       string
			RealizedBy []struct{ File string } `json:"realized_by"`
			Record     struct{ State string }
		}
	}
	if code, out, _ := run(t, r.Dir, "doc", "--json", "members/member"); code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Kind != "doc" || doc.Definition.ID != "members/member" || len(doc.Definition.RealizedBy) != 1 || doc.Definition.Record.State != "ok" {
		t.Fatalf("doc --json: %d %s", code, out)
	}
	// list shows the Packages and their Definitions; --deps what each depends on.
	wantAll(t, mustRun(t, r.Dir, "list"), "Root\n  loan-limit: Loan limit (std/Rule)\n", "members\n  members/member: Member (std/Concept)\n")
	wantAll(t, mustRun(t, r.Dir, "list", "--deps"), "Root (Ca 0, Ce 0, instability 0.00, abstractness 0.00, distance 1.00)\n  loan-limit: Loan limit (std/Rule)\n    links members\n")
	var listed struct {
		Kind     string
		Packages []struct {
			Name string
			Deps *struct{ Efferent int }
		}
	}
	if code, out, _ := run(t, r.Dir, "list", "--json"); code != 0 || json.Unmarshal([]byte(out), &listed) != nil || listed.Kind != "list" || len(listed.Packages) != 2 || listed.Packages[0].Deps != nil {
		t.Fatalf("list --json: %d %s", code, out)
	}
	// vet warns, and exits 1 only with --strict. Plain vet counts its notes in one
	// line, --strict lists them, and they never change the exit code.
	if code, out, _ := run(t, r.Dir, "vet"); code != 0 || out != "" {
		t.Fatalf("vet: %d %s", code, out)
	}
	r.Write(".markitect/spec/members/orphan.md", "---\nkind: std/Concept\n---\n# Orphan\n\nA member within the loan limit. "+strings.Repeat("word ", 1001)+"\n")
	if code, out, _ := run(t, r.Dir, "vet"); code != 0 || out != ".markitect/spec/members/orphan.md: vet.definition_size: 1009 words (more than 1000); split it or move detail the spec does not need to the realization\nnotes: 2 unlinked mentions in 1 Definition; markitect vet --strict lists them\n" {
		t.Fatalf("vet with a warning and a note: %d %s", code, out)
	}
	if code, out, _ := run(t, r.Dir, "vet", "--strict"); code != 1 || !strings.Contains(out, ".markitect/spec/members/orphan.md:6: vet.unlinked_mention: names loan-limit and members/member without linking them\n") || strings.Contains(out, "notes:") {
		t.Fatalf("vet --strict with a warning and a note: %d %s", code, out)
	}
	var vetted struct {
		Kind        string
		Diagnostics []struct {
			Severity, Code, File string
			Targets              []string
		}
	}
	if code, out, _ := run(t, r.Dir, "vet", "--json"); code != 0 || json.Unmarshal([]byte(out), &vetted) != nil || vetted.Kind != "vet" || len(vetted.Diagnostics) != 2 || vetted.Diagnostics[0].Severity != "warning" || vetted.Diagnostics[1].Severity != "note" || strings.Join(vetted.Diagnostics[1].Targets, ",") != "loan-limit,members/member" {
		t.Fatalf("vet --json: %d %s", code, out)
	}
	var stated struct {
		Kind        string
		Clean       bool
		Items       []struct{ Definition, State, Fingerprint string }
		Diagnostics []struct{ Severity, Code string }
	}
	if code, out, _ := run(t, r.Dir, "status", "--json"); code != 1 || json.Unmarshal([]byte(out), &stated) != nil || stated.Kind != "status" || stated.Clean || len(stated.Items) != 2 || stated.Items[0].Definition != "loan-limit" || len(stated.Diagnostics) != 1 || stated.Diagnostics[0].Code != "definition.unused" {
		t.Fatalf("status --json: %d %s", code, out)
	}

	// status has no list for a spec that does not compile.
	r.Write(".markitect/spec/broken.md", "no front matter\n")
	code, out, _ = run(t, r.Dir, "status")
	if code != 1 || !strings.Contains(out, "there is no list until it compiles") {
		t.Fatalf("status on errors: %d %s", code, out)
	}
	code, _, errOut := run(t, r.Dir, "vet")
	if code != 0 || !strings.Contains(errOut, "the spec has") {
		t.Fatalf("vet on errors: %d %s", code, errOut)
	}
}

// check tells the spec from the realization, and --spec looks only at the spec.
func TestCheckGroupsItsErrors(t *testing.T) {
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Demo")
	r.Write("src/stray.go", "package stray\n")
	code, out, _ := run(t, r.Dir, "check")
	if code != 1 || !strings.Contains(out, "src/stray.go: file.unregistered: ") || !strings.Contains(out, "2 errors: 0 in the spec, for the owner to settle; 2 in the realization") {
		t.Fatalf("check on a project with a stray file: %d\n%s", code, out)
	}
	if out := mustRun(t, r.Dir, "check", "--spec"); !strings.HasPrefix(out, "ok:") {
		t.Fatalf("check --spec: %s", out)
	}
	r.Write(".markitect/spec/broken.md", "no front matter\n")
	code, out, _ = run(t, r.Dir, "check", "--spec")
	if code != 1 || !strings.Contains(out, ".markitect/spec/broken.md:1: definition.front_matter: ") || !strings.Contains(out, "2 in the spec, for the owner to settle; 0 in the realization") {
		t.Fatalf("check --spec on a broken spec: %d\n%s", code, out)
	}
}

// A new project passes check, in plain words. As soon as there is a file, it is
// held to account as ever, and so is a Definition once anything is registered.
func TestCheckWithNothingBuilt(t *testing.T) {
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Demo")
	code, out, _ := run(t, r.Dir, "check")
	if code != 0 || !strings.HasPrefix(out, "ok:") || !strings.Contains(out, "Nothing is built yet") || !strings.Contains(out, "/markitect-build") || strings.Contains(out, "definition.unused") {
		t.Fatalf("check on a new project: %d\n%s", code, out)
	}
	code, out, _ = run(t, r.Dir, "check", "--json")
	var checked struct {
		OK           bool
		NothingBuilt bool `json:"nothing_built"`
		Diagnostics  []any
	}
	if code != 0 || json.Unmarshal([]byte(out), &checked) != nil || !checked.OK || !checked.NothingBuilt || checked.Diagnostics == nil || len(checked.Diagnostics) != 0 {
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

	// What an extraction leaves: files registered, nothing certified yet.
	r.Write(".markitect/registry.md", read(t, r.Dir, ".markitect/registry.md")+"| `src/a.go` | [[project]] | The code |\n")
	r.Write(".markitect/spec/rule.md", "---\nkind: std/Rule\n---\n# Rule\n\nAlways.\n")
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
	r := newFolder(t)
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

	r := newFolder(t)
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
	for _, want := range []string{"name: markitect\n", "A Markitect project has one spec", "no need to look for it", "markitect-architect", "/reload-skills", "markitect update", "claude plugin marketplace add Glacius-Labs/Claude-Plugins", "claude plugin install markitect@glacius-labs", "github.com/Glacius-Labs/Markitect/releases"} {
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

// The hook joins settings the project already has, and is added once.
func TestInitKeepsTheProjectsSettings(t *testing.T) {
	r := newFolder(t)
	r.Write(".claude/settings.json", "{\n  \"permissions\": {\"allow\": [\"Bash(go test *)\"]},\n  \"cleanupPeriodDays\": 12345678901234567890\n}\n")
	mustRun(t, r.Dir, "init")
	mustRun(t, r.Dir, "init")
	settings := read(t, r.Dir, ".claude/settings.json")
	if !strings.Contains(settings, "Bash(go test *)") || !strings.Contains(settings, "12345678901234567890") ||
		strings.Count(settings, "Bash(markitect mark*)") != 1 || strings.Count(settings, "Edit|Write|MultiEdit|NotebookEdit") != 1 ||
		strings.Count(settings, `"Agent|Task|Workflow"`) != 1 || strings.Count(settings, `"baseRef": "head"`) != 1 {
		t.Fatalf("settings:\n%s", settings)
	}
}

// A base for the worktrees that the owner chose stays, and counts as current.
func TestInitKeepsTheOwnersWorktreeBase(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	r.Write(".claude/settings.json", `{"worktree": {"baseRef": "fresh", "sparsePaths": ["src"]}}`+"\n")
	out := mustRun(t, r.Dir, "init")
	settings := read(t, r.Dir, ".claude/settings.json")
	if strings.Contains(out, "worktree.baseRef") || !strings.Contains(settings, `"baseRef": "fresh"`) || !strings.Contains(settings, "sparsePaths") {
		t.Fatalf("init said:\n%s\nsettings:\n%s", out, settings)
	}
	if out := mustRun(t, r.Dir, "update", "--check", "--files"); !strings.Contains(out, "project: up to date") {
		t.Fatalf("the owner's base counts as out of date:\n%s", out)
	}
}

// A mention of the hook's command that is not the hook gets the hook added.
func TestInitAddsTheHookBesideAMention(t *testing.T) {
	r := newFolder(t)
	r.Write(".claude/settings.json", `{"hooks": {"PostToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "markitect hook pretool"}]}]}}`+"\n")
	mustRun(t, r.Dir, "init")
	if !strings.Contains(read(t, r.Dir, ".claude/settings.json"), "PreToolUse") {
		t.Fatalf("no hook added:\n%s", read(t, r.Dir, ".claude/settings.json"))
	}
}

// Settings init cannot read stay as they are, and init says what to add.
func TestInitLeavesSettingsItCannotRead(t *testing.T) {
	for name, content := range map[string]string{
		"not JSON":   "{ not json\n",
		"null":       "null\n",
		"two values": "{} {}\n",
	} {
		t.Run(name, func(t *testing.T) {
			r := newFolder(t)
			r.Write(".claude/settings.json", content)
			out := mustRun(t, r.Dir, "init")
			if !strings.Contains(out, "left .claude/settings.json as it is") || !strings.Contains(out, "add the hooks by hand") || !strings.Contains(out, "worktree.baseRef") {
				t.Fatalf("init said:\n%s", out)
			}
			if got := read(t, r.Dir, ".claude/settings.json"); got != content {
				t.Fatalf("settings changed:\n%s", got)
			}
		})
	}
}

// A part of the settings with another shape stays as it is, init says what to
// add there, and it still writes the other part: a worktree setting it cannot
// read does not cost the project its hooks.
func TestInitLeavesAPartItCannotRead(t *testing.T) {
	for name, c := range map[string]struct{ content, said, kept, added string }{
		"hooks not a map":       {`{"hooks": ["something"]}`, "left the hooks in", `"something"`, `"baseRef": "head"`},
		"PreToolUse not a list": {`{"hooks": {"PreToolUse": "x"}}`, "left the hooks in", `"PreToolUse": "x"`, `"baseRef": "head"`},
		"worktree not a map":    {`{"worktree": "x"}`, "left worktree in", `"worktree": "x"`, "Agent|Task"},
	} {
		t.Run(name, func(t *testing.T) {
			r := newFolder(t)
			r.Write(".claude/settings.json", c.content+"\n")
			out := mustRun(t, r.Dir, "init")
			settings := read(t, r.Dir, ".claude/settings.json")
			if !strings.Contains(out, c.said) || !strings.Contains(out, "by hand") || !strings.Contains(settings, c.kept) || !strings.Contains(settings, c.added) {
				t.Fatalf("init said:\n%s\nsettings:\n%s", out, settings)
			}
		})
	}
}

// Settings that already hold the hooks of this markitect get only the base of
// the worktrees, and a base set to null counts as none.
func TestInitAddsTheWorktreeBaseToGuardedSettings(t *testing.T) {
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	settings := read(t, r.Dir, ".claude/settings.json")
	r.Write(".claude/settings.json", strings.Replace(settings, `"baseRef": "head"`, `"baseRef": null`, 1))
	out := mustRun(t, r.Dir, "init")
	if got := read(t, r.Dir, ".claude/settings.json"); got != settings || strings.Contains(out, "added the hooks") || !strings.Contains(out, "worktree.baseRef") {
		t.Fatalf("init said:\n%s\nsettings:\n%s", out, got)
	}
}

// The hook runs the installed command, and only the reviewer may record.
func TestHookLetsOnlyTheReviewerRecord(t *testing.T) {
	hook := func(agent string) string {
		input, _ := json.Marshal(map[string]any{"agent_id": "a1", "agent_type": agent, "tool_name": "Bash", "tool_input": map[string]any{"command": "markitect mark project@0a1b2c ok -e README.md"}})
		var out, errOut bytes.Buffer
		code := newCLI(bytes.NewReader(input), &out, &errOut).Run(context.Background(), []string{"hook", "pretool"})
		if code != 0 {
			t.Fatalf("hook exit %d: %s", code, errOut.String())
		}
		return out.String()
	}
	if out := hook("markitect-builder"); !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("the builder may record: %s", out)
	}
	if out := hook("markitect-reviewer"); out != "" {
		t.Fatalf("the reviewer may not: %s", out)
	}
}

// projectHook is the hook init writes into the project's settings.
const projectHook = "command -v markitect >/dev/null 2>&1 || exit 0; markitect hook pretool"

// The project's hook runs the markitect its shell finds, with the input of the
// call, and does nothing where it finds none: with the plugin alone a hook sees
// no markitect, and the plugin's own hooks guard. Up to 0.25.2 it failed there
// with exit 127, before nearly every call.
func TestProjectHookRunsOnlyAMarkitectItFinds(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("the hook is a shell command and there is no sh")
	}
	const input = `{"tool_name":"Edit"}`
	hook := func(path string) (string, int) {
		cmd := exec.Command(sh, "-c", projectHook)
		cmd.Stdin = strings.NewReader(input)
		cmd.Env = append(os.Environ(), "PATH="+path)
		out, err := cmd.CombinedOutput()
		if exit, ok := err.(*exec.ExitError); ok {
			return string(out), exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return string(out), 0
	}
	if out, code := hook(testkit.TempDir(t)); code != 0 || out != "" {
		t.Fatalf("without a markitect: exit %d\n%s", code, out)
	}
	bin := testkit.TempDir(t)
	stub := "#!/bin/sh\nIFS= read -r line\nprintf '%s|%s' \"$*\" \"$line\"\n"
	if err := os.WriteFile(filepath.Join(bin, "markitect"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, code := hook(bin); code != 0 || out != "hook pretool|"+input {
		t.Fatalf("with a markitect: exit %d\n%s", code, out)
	}
}

// init has Git ignore the worktrees Claude Code makes in the project's folder,
// once, keeping what the ignore file holds and its line endings; update adds the
// line where it is missing.
func TestInitIgnoresClaudeCodesWorktrees(t *testing.T) {
	emptyHome(t)
	const lines = "# the worktrees Claude Code makes, as for Markitect's builders\n.claude/worktrees/\n"
	r := newFolder(t)
	out := mustRun(t, r.Dir, "init")
	if got := read(t, r.Dir, ".gitignore"); got != lines || !strings.Contains(out, "added .claude/worktrees/ to .gitignore") {
		t.Fatalf("init said:\n%s\n.gitignore: %q", out, got)
	}
	if out := mustRun(t, r.Dir, "init"); strings.Contains(out, ".gitignore") || read(t, r.Dir, ".gitignore") != lines {
		t.Fatalf("a second init:\n%s\n%q", out, read(t, r.Dir, ".gitignore"))
	}
	for before, after := range map[string]string{
		"bin/\r\n":          "bin/\r\n\r\n" + strings.ReplaceAll(lines, "\n", "\r\n"),
		"node_modules":      "node_modules\n\n" + lines,
		"/.claude/\n":       "/.claude/\n",
		".claude/worktrees": ".claude/worktrees",
		// The owner's own choice, to track them, stays.
		"!.claude/worktrees/\n":     "!.claude/worktrees/\n",
		".claude/*\n":               ".claude/*\n",
		"**/.claude/worktrees/**\n": "**/.claude/worktrees/**\n",
	} {
		r := newFolder(t)
		r.Write(".gitignore", before)
		mustRun(t, r.Dir, "init")
		if got := read(t, r.Dir, ".gitignore"); got != after {
			t.Errorf("init made %q of %q, not %q", got, before, after)
		}
	}
	r.Write(".gitignore", "bin/\n")
	if _, out, _ := run(t, r.Dir, "update", "--check", "--files"); !strings.Contains(out, "\n  .gitignore") {
		t.Fatalf("update --check misses the ignore line:\n%s", out)
	}
	mustRun(t, r.Dir, "update")
	if got := read(t, r.Dir, ".gitignore"); got != "bin/\n\n"+lines {
		t.Fatalf("update made %q", got)
	}
}

// init installs the harness with the hooks, and the record's Git attributes, and
// makes no repository and no commit.
func TestInitInstallsTheHarness(t *testing.T) {
	r := newFolder(t)
	out := mustRun(t, r.Dir, "init", "--name", "Demo")
	if !strings.Contains(out, "installed the harness") || !strings.Contains(out, "Markitect uses no Git") {
		t.Fatalf("init: %s", out)
	}
	noRepository(t, r.Dir)
	if got := read(t, r.Dir, ".markitect/.gitattributes"); got != "review.sum merge=union\n" {
		t.Fatalf(".markitect/.gitattributes: %q", got)
	}
	// The skills are plain prompts: they inject no command, so no exit code can
	// abort them. Only the owner starts the work that changes code and the
	// routines; Claude may start design, review and audit (DEC-100).
	for skill, ownerOnly := range map[string]bool{"build": true, "cleanup": true, "bughunt": true, "schedule": true, "review": false, "audit": false} {
		text := read(t, r.Dir, ".claude/skills/markitect-"+skill+"/SKILL.md")
		if !strings.Contains(text, "name: markitect-"+skill+"\n") || strings.Contains(text, "!`") {
			t.Fatalf("the %s skill:\n%s", skill, text)
		}
		if got := strings.Contains(text, "\ndisable-model-invocation: true\n"); got != ownerOnly {
			t.Errorf("the %s skill: only the owner starts it = %t, want %t", skill, got, ownerOnly)
		}
	}
	if strings.Contains(read(t, r.Dir, ".claude/skills/markitect-architect/SKILL.md"), "disable-model-invocation") {
		t.Error("Claude may not start the architect skill")
	}
	// A scheduled prompt is one line that points at the routine's file in the
	// project, so that update reaches a routine that is already scheduled. The
	// file holds nothing to fill in and the work of all three routines.
	routine := read(t, r.Dir, ".claude/skills/markitect-schedule/routine.md")
	if strings.Contains(routine, "{{") {
		t.Errorf("the routine's file has a value to fill in:\n%s", routine)
	}
	for _, work := range []string{"- **review:**", "- **bughunt:**", "- **cleanup:**", `"Verify every Definition: markitect status --all."`, `"Hunt bugs."`, `"Clean up."`} {
		if !strings.Contains(routine, work) {
			t.Errorf("the routine's file lacks %s", work)
		}
	}
	if setup := read(t, r.Dir, ".claude/skills/markitect-schedule/SKILL.md"); !strings.Contains(setup, "read .claude/skills/markitect-schedule/routine.md there and follow it.") || !strings.Contains(setup, "(routine.md)") {
		t.Errorf("the schedule skill does not point a routine at its file:\n%s", setup)
	}
	for _, gone := range []string{"review.md", "bughunt.md", "cleanup.md"} {
		if _, err := os.Stat(filepath.Join(r.Dir, ".claude", "skills", "markitect-schedule", gone)); !os.IsNotExist(err) {
			t.Errorf("init installed %s, a copy of a routine's work", gone)
		}
	}
	// Only the builder may start an agent, and its tools name the reviewer; the
	// hook holds that, since a subagent's tools do not.
	for agent, tools := range map[string]string{
		"builder":   "Read, Edit, Write, Glob, Grep, Bash, Agent(markitect-reviewer)",
		"reviewer":  "Read, Grep, Glob, Bash",
		"architect": "Read, Edit, Write, Glob, Grep, Bash",
		"auditor":   "Read, Grep, Glob, Bash",
	} {
		if text := read(t, r.Dir, ".claude/agents/markitect-"+agent+".md"); !strings.Contains(text, "\ntools: "+tools+"\n") {
			t.Fatalf("the %s's tools:\n%s", agent, text)
		}
	}
	var settings struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []struct{ Type, Shell, If, Command string }
		}
		Worktree struct{ BaseRef string }
	}
	if err := json.Unmarshal([]byte(read(t, r.Dir, ".claude/settings.json")), &settings); err != nil {
		t.Fatal(err)
	}
	if pre := settings.Hooks["PreToolUse"]; len(pre) != 3 || pre[0].Matcher != "Bash|PowerShell" || len(pre[0].Hooks) != 2 ||
		pre[0].Hooks[0].Command != projectHook || pre[0].Hooks[0].If != "Bash(markitect mark*)" ||
		pre[0].Hooks[1].Command != projectHook || pre[0].Hooks[1].If != "PowerShell(markitect mark*)" || pre[0].Hooks[1].Shell != "bash" ||
		pre[1].Matcher != "Edit|Write|MultiEdit|NotebookEdit" || len(pre[1].Hooks) != 1 ||
		pre[1].Hooks[0].Command != projectHook || pre[1].Hooks[0].If != "" ||
		pre[2].Matcher != "Agent|Task|Workflow" || len(pre[2].Hooks) != 1 ||
		pre[2].Hooks[0].Command != projectHook || pre[2].Hooks[0].If != "" ||
		pre[0].Hooks[0].Shell != "bash" || pre[1].Hooks[0].Shell != "bash" || pre[2].Hooks[0].Shell != "bash" {
		t.Fatalf("the guard hooks: %+v", settings)
	}
	// A builder's worktree starts from the owner's branch, not the default one.
	if settings.Worktree.BaseRef != "head" || !strings.Contains(out, "worktree.baseRef") {
		t.Fatalf("worktree.baseRef = %q\n%s", settings.Worktree.BaseRef, out)
	}
	code, checked, _ := run(t, r.Dir, "check")
	if code != 0 || strings.Contains(checked, "file.unregistered") {
		t.Fatalf("the harness needs no registry entry: %s", checked)
	}
	// Run again it changes nothing, not even what the owner is editing.
	r.Write(".markitect/spec/project.md", "---\nkind: std/Project\n---\n# Demo\n\nBeing edited.\n")
	r.Write(".claude/agents/mine.md", "private\n")
	before := snapshot(t, r.Dir)
	mustRun(t, r.Dir, "init", "--name", "Demo")
	if diff := changed(before, snapshot(t, r.Dir)); len(diff) > 0 {
		t.Fatalf("a second init changed files: %v", diff)
	}

	// Inside a Git repository, init commits nothing either.
	g := testkit.NewRepo(t)
	g.Write("README.md", "# Demo\n")
	head := g.Commit("the owner's")
	mustRun(t, g.Dir, "init", "--name", "Demo")
	if got := g.Git("rev-parse", "HEAD"); got != head {
		t.Fatalf("init committed: %s", g.Git("show", "--stat", "HEAD"))
	}
	if status := g.Git("status", "--porcelain"); !strings.Contains(status, ".markitect/") {
		t.Fatalf("init's files are not left for the owner: %q", status)
	}
}

func TestTidyWritesTheLock(t *testing.T) {
	r := newFolder(t)
	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.1.0\n  rest: 1.1.0\n  web: 0.1.0\n")
	code, _, errOut := run(t, r.Dir, "tidy")
	if code != 1 || !strings.Contains(errOut, "library.missing") {
		t.Fatalf("lock without web: %d %s", code, errOut)
	}
	r.Write(".markitect/libraries/web/Page.md", "---\nkind: Kind\n---\n# Page\n")
	if out := mustRun(t, r.Dir, "tidy"); !strings.Contains(out, "rest 1.1.0, std 1.1.0, web 0.1.0") {
		t.Fatalf("lock = %s", out)
	}
	if lock := read(t, r.Dir, ".markitect/lock.yaml"); !strings.Contains(lock, "  web:\n    version: 0.1.0\n    digest: sha256:") {
		t.Fatalf("lock file:\n%s", lock)
	}
}

// A changed Kind is a new content, so a standard Library's version moves with
// the binary. A project that names the version of an earlier binary gets one
// error, which says what to do, and none of what would follow from it: its
// Kinds still resolve, and the lock is not compared. tidy and update name the
// shipped version in the manifest, keeping every other byte of it, since the
// binary supplies no other; version says the project is out of date until then.
func TestAStandardLibraryMovesWithTheBinary(t *testing.T) {
	emptyHome(t)
	const earlier = "# The Libraries this project uses.\nlibraries:\n  std: 1.0.0 # the first\n  rest: 0.9.0\n"
	old := func() *plainFolder {
		r := newFolder(t)
		mustRun(t, r.Dir, "init")
		r.Write(".markitect/manifest.yaml", earlier)
		return r
	}
	r := old()
	code, out, _ := run(t, r.Dir, "check")
	wantAll(t, out, ".markitect/manifest.yaml:3: library.version: this markitect ships std 1.1.0, not 1.0.0; markitect tidy names 1.1.0 in the manifest", ".markitect/manifest.yaml:4: library.version: this markitect ships rest 1.1.0, not 0.9.0")
	if code != 1 || strings.Contains(out, "lock.mismatch") || strings.Contains(out, "kind.unresolved") || strings.Count(out, "library.version") != 2 {
		t.Fatalf("check on an earlier version: %d\n%s", code, out)
	}
	if _, out, _ := run(t, r.Dir, "version"); !strings.Contains(out, "project: out of date") {
		t.Fatalf("version on an earlier version:\n%s", out)
	}
	wantAll(t, mustRun(t, r.Dir, "tidy"),
		".markitect/manifest.yaml now names std 1.1.0, where it named 1.0.0",
		".markitect/manifest.yaml now names rest 1.1.0, where it named 0.9.0",
		"wrote .markitect/lock.yaml: rest 1.1.0, std 1.1.0")
	if got, want := read(t, r.Dir, ".markitect/manifest.yaml"), strings.NewReplacer("1.0.0", "1.1.0", "0.9.0", "1.1.0").Replace(earlier); got != want {
		t.Fatalf("the manifest after tidy:\n%s", got)
	}
	mustRun(t, r.Dir, "check")
	if out := mustRun(t, r.Dir, "tidy"); strings.Contains(out, "now names") {
		t.Fatalf("a second tidy:\n%s", out)
	}

	// update does the same, and says so.
	r = old()
	wantAll(t, mustRun(t, r.Dir, "update", "--files"), ".markitect/manifest.yaml now names std 1.1.0, where it named 1.0.0", "lock: rewritten")
	mustRun(t, r.Dir, "check")

	// The version is written where it stands, whatever else the line holds: in
	// quotes, in flow style behind a project Library of the same version, or on
	// a line of its own.
	for before, after := range map[string]string{
		"libraries:\r\n  \"std\" :   '1.0.0'   # 1.0.0 was the first\r\n": "libraries:\r\n  \"std\" :   '1.1.0'   # 1.0.0 was the first\r\n",
		"libraries: {web: 1.0.0, std: 1.0.0}\n":                           "libraries: {web: 1.0.0, std: 1.1.0}\n",
		"libraries:\n  std:\n    1.0.0\n  web: 1.0.0\n":                   "libraries:\n  std:\n    1.1.0\n  web: 1.0.0\n",
	} {
		r = old()
		if strings.Contains(before, "web") {
			r.Write(".markitect/libraries/web/Page.md", "---\nkind: Kind\n---\n# Page\n")
		}
		r.Write(".markitect/manifest.yaml", before)
		mustRun(t, r.Dir, "tidy")
		if got := read(t, r.Dir, ".markitect/manifest.yaml"); got != after {
			t.Errorf("tidy made %q of %q", got, before)
		}
		mustRun(t, r.Dir, "check", "--spec")
	}

	// A manifest with an error of its own is left as it is.
	r = old()
	r.Write(".markitect/manifest.yaml", earlier+"  web: 0.1.0\n")
	if code, _, errOut := run(t, r.Dir, "tidy"); code != 1 || !strings.Contains(errOut, "library.missing") || read(t, r.Dir, ".markitect/manifest.yaml") != earlier+"  web: 0.1.0\n" {
		t.Fatalf("tidy with a Library that is missing: %d %s", code, errOut)
	}
}

func TestUsageAndRefusals(t *testing.T) {
	var out, errOut bytes.Buffer
	if Run(context.Background(), nil, &out, &errOut) != 0 || !strings.Contains(out.String(), "DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]") {
		t.Fatalf("usage = %s", out.String())
	}
	if Run(context.Background(), []string{"bughunt"}, &out, &errOut) != 2 || !strings.Contains(errOut.String(), `unknown command "bughunt"`) {
		t.Fatalf("unknown command: %s", errOut.String())
	}
	dir := testkit.TempDir(t)
	for _, args := range [][]string{{"check"}, {"tidy"}, {"status"}, {"mark", "project@0a1b2c", "ok", "-e", "README.md"}} {
		if code, _, errOut := run(t, dir, args...); code != 2 || !strings.Contains(errOut, "is in no Markitect project") || !strings.Contains(errOut, "markitect init makes one") {
			t.Fatalf("%s outside a project: %d %s", args[0], code, errOut)
		}
	}
	// The commands step 2 of the tooling plan renamed are gone: vet, list, doc and
	// tidy take their place, and the commands stay flat.
	for _, cmd := range []string{"impact", "lint", "refs", "graph", "lock", "review", "checklist", "answer", "gate"} {
		if code, _, errOut := run(t, dir, cmd); code != 2 || !strings.Contains(errOut, "unknown command") {
			t.Fatalf("%s: %d %s", cmd, code, errOut)
		}
	}
	out.Reset()
	if Run(context.Background(), []string{"version"}, &out, &errOut) != 0 || !strings.HasPrefix(out.String(), "markitect ") {
		t.Fatalf("version = %s", out.String())
	}
}

// status says what is to verify, mark records a verdict at the fingerprint
// status printed, and the record is a file of the project; the exit codes say
// whether a command found something.
func TestStatusAndMarkThroughTheCommandLine(t *testing.T) {
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	r.Write(".markitect/registry.md", "| File | Realizes | Why |\n|---|---|---|\n| README.md | [[project]] | Overview |\n")
	r.Write("README.md", "# A project\n")
	code, out, _ := run(t, r.Dir, "status")
	if code != 1 || !strings.Contains(out, "no record yet") || !strings.Contains(out, "look at: .markitect/spec/project.md, README.md") {
		t.Fatalf("status: %d\n%s", code, out)
	}
	item := strings.Fields(out[strings.Index(out, "project@"):])[0]

	// Limited to a Package, status prints that Package's part, and --json names
	// the Package; one the spec does not have is a usage error.
	if code, out, _ := run(t, r.Dir, "status", "--package", "Root"); code != 1 || !strings.Contains(out, "To verify in Package Root (1 of 1 Definition)") || !strings.Contains(out, item) {
		t.Fatalf("status --package Root: %d\n%s", code, out)
	}
	var limited struct {
		Package string
		Clean   bool
		Items   []struct{ Definition string }
	}
	if code, out, _ := run(t, r.Dir, "status", "--package", "Root", "--json"); code != 1 || json.Unmarshal([]byte(out), &limited) != nil || limited.Package != "Root" || limited.Clean || len(limited.Items) != 1 {
		t.Fatalf("status --package Root --json: %d\n%s", code, out)
	}
	// The questions for the owner are a note of the whole list: --json names
	// their file, and a list limited to a Package leaves it out.
	r.Write(".markitect/questions.md", "# Questions\n\n1. Is a member a person or a household?\n")
	if code, out, _ := run(t, r.Dir, "status", "--json"); code != 1 || !strings.Contains(out, `"questions": ".markitect/questions.md"`) {
		t.Fatalf("status --json with questions: %d\n%s", code, out)
	}
	if code, out, _ := run(t, r.Dir, "status", "--package", "Root", "--json"); code != 1 || strings.Contains(out, `"questions"`) {
		t.Fatalf("status --package Root --json with questions: %d\n%s", code, out)
	}
	if err := os.Remove(filepath.Join(r.Dir, ".markitect", "questions.md")); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run(t, r.Dir, "status", "--package", "nowhere"); code != 2 || !strings.Contains(errOut, `no Package "nowhere"`) {
		t.Fatalf("status --package nowhere: %d %s", code, errOut)
	}

	for _, args := range [][]string{
		{"mark", item, "maybe"},
		{"mark", "project"},
		{"mark", item, "fail"},
		{"mark", item, "ok"},
		{"mark", "project", "ok", "-e", "README.md"},
		{"mark", item, "ok", "README.md"},
		{"mark", item, "ok", "-e", "README.md", "--bogus"},
		{"mark", item, "ok", "-e"},
		{"mark", item, "ok", "-e", "README.md", "--cases", "seven"},
		{"mark", item, "ok", "-e", "README.md", "--covered", "2"},
		{"mark", item, "ok", "-e", "README.md", "--cases=2", "--covered=3"},
		{"mark", item, "ok", "-e", "README.md", "--cases"},
		{"mark", item, "ok", "-e", "README.md"},
		{"mark", item, "ok", "-e", "README.md", "--cases", "2"},
		{"mark", item, "skip", "-m", "no test can reach it"},
	} {
		if code, _, errOut := run(t, r.Dir, args...); code != 2 || !strings.Contains(errOut, "usage: markitect mark") {
			t.Fatalf("%v: %d %s", args, code, errOut)
		}
	}
	if code, out, _ := run(t, r.Dir, "mark", item, "ok", "-e", "tests/missing_test.go:3", "--cases", "1", "--covered", "1"); code != 1 || !strings.Contains(out, "tests/missing_test.go") {
		t.Fatalf("mark with missing evidence: %d\n%s", code, out)
	}
	// The flags may stand before the item and the verdict, too, and the counts of
	// cases go on the line.
	wantAll(t, mustRun(t, r.Dir, "mark", "-e", "README.md:1 (TestReadme)", "--cases", "4", "-m", "says nothing of what the project is", item, "fail", "--covered=0"), "Recorded: project fail.")
	wantAll(t, read(t, r.Dir, ".markitect/review.sum"), `"evidence":["README.md:1"]`, `"cases":4,"covered":0}`)
	code, out, _ = run(t, r.Dir, "status")
	if code != 1 || !strings.Contains(out, "fail: says nothing of what the project is") {
		t.Fatalf("status after a fail: %d\n%s", code, out)
	}
	r.Write("README.md", "# A project\n\nIt lends books.\n")
	if code, out, _ := run(t, r.Dir, "mark", item, "ok", "-e", "README.md:3"); code != 1 || !strings.Contains(out, "Run markitect status again") {
		t.Fatalf("mark at a stale fingerprint: %d\n%s", code, out)
	}
	code, out, _ = run(t, r.Dir, "status")
	if code != 1 || !strings.Contains(out, "changed: file:README.md") || !strings.Contains(out, "recorded before: fail") {
		t.Fatalf("status after a change: %d\n%s", code, out)
	}
	verifyAll(t, r.Dir)
	wantAll(t, read(t, r.Dir, ".markitect/review.sum"), `{"definition":"project","verdict":"ok"`)
	if code, out, _ := run(t, r.Dir, "status", "--all"); code != 1 || !strings.Contains(out, "project@") {
		t.Fatalf("status --all: %d\n%s", code, out)
	}
	if code, out, _ := run(t, r.Dir, "check"); code != 0 {
		t.Fatalf("the record is part of the layout: %s", out)
	}

	// --json: mark, recorded and refused, and status with a line it cannot read.
	r.Write("README.md", "# A project\n\nIt lends books and records.\n")
	_, out, _ = run(t, r.Dir, "status")
	item = strings.Fields(out[strings.Index(out, "project@"):])[0]
	var marked struct {
		Kind     string
		Recorded bool
		Message  string
	}
	if code, out, _ := run(t, r.Dir, "mark", "--json", item, "ok", "-e", "README.md:3", "--cases", "1", "--covered", "1"); code != 0 || json.Unmarshal([]byte(out), &marked) != nil || marked.Kind != "mark" || !marked.Recorded {
		t.Fatalf("mark --json: %d %s", code, out)
	}
	r.Write("README.md", "# A project\n\nIt lends books, records and games.\n")
	if code, out, _ := run(t, r.Dir, "mark", item, "ok", "-e", "README.md:3", "--cases", "1", "--covered", "1", "--json"); code != 1 || json.Unmarshal([]byte(out), &marked) != nil || marked.Recorded || !strings.Contains(marked.Message, "Run markitect status again") {
		t.Fatalf("mark --json at a stale fingerprint: %d %s", code, out)
	}
	r.Write(".markitect/review.sum", read(t, r.Dir, ".markitect/review.sum")+"not json\n")
	var stated struct {
		Clean       bool
		Diagnostics []struct {
			Code string
			File string
			Line int
		}
		Unreadable []int `json:"unreadable_lines"`
	}
	if code, out, _ := run(t, r.Dir, "status", "--json"); code != 1 || json.Unmarshal([]byte(out), &stated) != nil || stated.Clean || len(stated.Diagnostics) != 1 || stated.Diagnostics[0].Code != "record.unreadable" || stated.Diagnostics[0].Line != 2 || len(stated.Unreadable) != 1 {
		t.Fatalf("status --json with a line it cannot read: %d %s", code, out)
	}
	noRepository(t, r.Dir)
}

// doc takes its flags after the target too, shows a Package and a Library's Kind,
// says in its document when the target is neither a Definition nor a Package,
// and shows a Package's public Definitions apart from its internal ones.
func TestDocAndVetDocuments(t *testing.T) {
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	var doc struct {
		Kind    string
		Package struct {
			Name        string
			Definitions []struct{ ID string }
		}
		Diagnostics []struct{ Severity, Code string }
	}
	if code, out, _ := run(t, r.Dir, "doc", "Root", "--json"); code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Kind != "doc" || doc.Package.Name != "Root" || len(doc.Package.Definitions) != 1 {
		t.Fatalf("doc Root --json: %d %s", code, out)
	}
	wantAll(t, mustRun(t, r.Dir, "doc", "std/Rule"), "std/Rule: ", "Library std\n")
	if code, out, _ := run(t, r.Dir, "doc", "nobody", "--json"); code != 1 || json.Unmarshal([]byte(out), &doc) != nil || len(doc.Diagnostics) != 1 || doc.Diagnostics[0].Code != "doc.unknown" {
		t.Fatalf("doc of nothing --json: %d %s", code, out)
	}
	r.Write(".markitect/spec/big.md", "---\nkind: std/Rule\n---\n# Big\n\n"+strings.Repeat("word ", 1001)+"\n")
	var vetted struct {
		Diagnostics []struct{ Severity, Code string }
	}
	if code, out, _ := run(t, r.Dir, "vet", "--json", "--strict"); code != 1 || json.Unmarshal([]byte(out), &vetted) != nil || len(vetted.Diagnostics) != 1 || vetted.Diagnostics[0].Code != "vet.definition_size" {
		t.Fatalf("vet --json --strict: %d %s", code, out)
	}
	// vet reads from the record how often a Definition's own text changed: the
	// anchor, linked once and changed 21 times, is a hotspot. A note leaves the exit
	// code alone.
	r.Write(".markitect/spec/big.md", "---\nkind: std/Rule\n---\n# Big\n\nFor the [[project]].\n")
	r.Write(".markitect/review.sum", `{"definition":"project","verdict":"ok","inputs":{"self":"sha256:0"},"revisions":21}`+"\n")
	if code, out, _ := run(t, r.Dir, "vet", "--strict"); code != 0 || !strings.Contains(out, ".markitect/spec/project.md: vet.hotspot: its own text changed 21 times between verdicts, and 1 Definition links to it or is of it as Kind, so its changes put about 21 back") {
		t.Fatalf("vet --strict with a hotspot: %d %s", code, out)
	}
	// vet reads the reviewer's counts from the record too: an ok on evidence for
	// 2 of 9 cases is a risk. A note leaves the exit code alone.
	r.Write(".markitect/review.sum", `{"definition":"project","verdict":"ok","inputs":{"self":"sha256:0"},"cases":9,"covered":2}`+"\n")
	if code, out, _ := run(t, r.Dir, "vet", "--strict"); code != 0 || strings.Contains(out, "vet.risk") {
		t.Fatalf("vet judges a line at another text of the Definition: %d %s", code, out)
	}
	self := sha256.Sum256(bytes.ReplaceAll([]byte(read(t, r.Dir, ".markitect/spec/project.md")), []byte("\r\n"), []byte("\n")))
	r.Write(".markitect/review.sum", fmt.Sprintf(`{"definition":"project","verdict":"ok","inputs":{"self":"sha256:%x"},"cases":9,"covered":2}`, self)+"\n")
	if code, out, _ := run(t, r.Dir, "vet", "--strict"); code != 0 || !strings.Contains(out, ".markitect/spec/project.md: vet.risk: its ok rests on evidence for 2 of its 9 cases (risk 58, more than 30)") {
		t.Fatalf("vet --strict with a thin ok: %d %s", code, out)
	}
	if code, _, errOut := run(t, r.Dir, "tidy", "--json"); code != 2 || !strings.Contains(errOut, "-json") {
		t.Fatalf("tidy --json: %d %s", code, errOut)
	}
	// A Package's public Definitions are what it publishes; doc shows them
	// apart from its internal ones, in its text and its document.
	r.Write(".markitect/spec/members/member.md", "---\nkind: std/Concept\n---\n# Member\n\nA person who borrows.\n")
	r.Write(".markitect/spec/members/limit.md", "---\nkind: std/Rule\ninternal: true\n---\n# Limit\n\nA [[members/member]] borrows at most five items.\n")
	wantAll(t, mustRun(t, r.Dir, "doc", "members"), "\nPublic Definitions:\n  members/member: Member (std/Concept)\n", "\nInternal Definitions:\n  members/limit: Limit (std/Rule)\n")
	var members struct {
		Package struct {
			Definitions []struct {
				ID       string
				Internal bool
			}
		}
	}
	if code, out, _ := run(t, r.Dir, "doc", "members", "--json"); code != 0 || json.Unmarshal([]byte(out), &members) != nil || len(members.Package.Definitions) != 2 ||
		members.Package.Definitions[0].ID != "members/limit" || !members.Package.Definitions[0].Internal || members.Package.Definitions[1].Internal {
		t.Fatalf("doc members --json: %d %s", code, out)
	}
	var limit struct {
		Definition struct{ Internal bool }
	}
	if code, out, _ := run(t, r.Dir, "doc", "members/limit", "--json"); code != 0 || json.Unmarshal([]byte(out), &limit) != nil || !limit.Definition.Internal {
		t.Fatalf("doc members/limit --json: %d %s", code, out)
	}
	var listed struct {
		Packages []struct {
			Name        string
			Definitions []struct {
				ID       string
				Internal bool
			}
		}
	}
	internal := map[string]bool{}
	if code, out, _ := run(t, r.Dir, "list", "--json"); code != 0 || json.Unmarshal([]byte(out), &listed) != nil {
		t.Fatalf("list --json: %d %s", code, out)
	}
	for _, p := range listed.Packages {
		for _, d := range p.Definitions {
			internal[d.ID] = d.Internal
		}
	}
	if is, ok := internal["members/limit"]; !ok || !is || internal["members/member"] {
		t.Fatalf("list --json internal: %v", internal)
	}
}
