package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	for _, want := range []string{"created .markitect/manifest.yaml", "created .markitect/model/project.md", "wrote .markitect/lock.yaml: std 1.0.0", "installed the skills markitect-design"} {
		if !strings.Contains(out, want) {
			t.Fatalf("init output lacks %q:\n%s", want, out)
		}
	}
	if !strings.Contains(read(t, r.Dir, ".claude/skills/markitect-design/SKILL.md"), "markitect-design") || !strings.Contains(read(t, r.Dir, ".agents/skills/markitect-design/reference.md"), "model form") {
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
	r.Write(".markitect/model/loan-limit.md", "---\nkind: std/Rule\n---\n# Loan limit\n\nAt most `3` open loans, for [[members]].\n")
	code, out, _ = run(t, r.Dir, "status")
	if code != 1 || !strings.Contains(out, "To verify (1 of 3 Definitions)") || !strings.Contains(out, "\nloan-limit@") || !strings.Contains(out, "changed: self\n") {
		t.Fatalf("status after a change of the spec: %d\n%s", code, out)
	}

	// doc shows a Definition with its text, its links both ways, its files and its
	// record; and a Package.
	wantAll(t, mustRun(t, r.Dir, "doc", "[[members/member]]"), "members/member: Member (std/Concept)\n", "Package members, .markitect/model/members/member.md\n", "A person who borrows.", "Realized by:\n  src/members/member.go, .markitect/registry/members.md:", "Record: members/member@", ", ok")
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
	wantAll(t, mustRun(t, r.Dir, "list", "--deps"), "Root (Ca 0, Ce 0, instability 0.00)\n  loan-limit: Loan limit (std/Rule)\n    links members\n")
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
	// vet warns, and exits 1 only with --strict.
	if code, out, _ := run(t, r.Dir, "vet"); code != 0 || out != "" {
		t.Fatalf("vet: %d %s", code, out)
	}
	r.Write(".markitect/model/members/orphan.md", "---\nkind: std/Concept\n---\n# Orphan\n\n"+strings.Repeat("word ", 1001)+"\n")
	if code, out, _ := run(t, r.Dir, "vet"); code != 0 || !strings.Contains(out, ".markitect/model/members/orphan.md: vet.definition_size: ") {
		t.Fatalf("vet with a warning: %d %s", code, out)
	}
	if code, _, _ := run(t, r.Dir, "vet", "--strict"); code != 1 {
		t.Fatalf("vet --strict with a warning: %d", code)
	}
	var vetted struct {
		Kind        string
		Diagnostics []struct{ Severity, Code, File string }
	}
	if code, out, _ := run(t, r.Dir, "vet", "--json"); code != 0 || json.Unmarshal([]byte(out), &vetted) != nil || vetted.Kind != "vet" || len(vetted.Diagnostics) != 1 || vetted.Diagnostics[0].Severity != "warning" {
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
	r.Write(".markitect/model/broken.md", "no front matter\n")
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
	r.Write(".markitect/model/broken.md", "no front matter\n")
	code, out, _ = run(t, r.Dir, "check", "--spec")
	if code != 1 || !strings.Contains(out, ".markitect/model/broken.md:1: definition.front_matter: ") || !strings.Contains(out, "2 in the spec, for the owner to settle; 0 in the realization") {
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
	for _, want := range []string{"name: markitect\n", "A Markitect project has one spec", "no need to look for it", "markitect-design", "/reload-skills", "markitect update", "claude plugin marketplace add Glacius-Labs/Claude-Plugins", "claude plugin install markitect@glacius-labs", "github.com/Glacius-Labs/Markitect/releases"} {
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
		strings.Count(settings, `"Agent|Task"`) != 1 || strings.Count(settings, `"baseRef": "head"`) != 1 {
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

// Settings init cannot read, or whose hooks have another shape, stay as they are,
// and init says what to add.
func TestInitLeavesSettingsItCannotRead(t *testing.T) {
	for name, content := range map[string]string{
		"not JSON":              "{ not json\n",
		"hooks not a map":       `{"hooks": ["something"]}` + "\n",
		"PreToolUse not a list": `{"hooks": {"PreToolUse": "x"}}` + "\n",
		"null":                  "null\n",
		"two values":            "{} {}\n",
		"worktree not a map":    `{"worktree": "x"}` + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			r := newFolder(t)
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
	if strings.Contains(read(t, r.Dir, ".claude/skills/markitect-design/SKILL.md"), "disable-model-invocation") {
		t.Error("Claude may not start the design skill")
	}
	for _, file := range []string{"routine.md", "review.md", "bughunt.md", "cleanup.md"} {
		if read(t, r.Dir, ".claude/skills/markitect-schedule/"+file) == "" {
			t.Fatalf("the routine prompt %s is empty", file)
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
			Hooks   []struct{ Type, If, Command string }
		}
		Worktree struct{ BaseRef string }
	}
	if err := json.Unmarshal([]byte(read(t, r.Dir, ".claude/settings.json")), &settings); err != nil {
		t.Fatal(err)
	}
	if pre := settings.Hooks["PreToolUse"]; len(pre) != 3 || pre[0].Matcher != "Bash" || len(pre[0].Hooks) != 1 ||
		pre[0].Hooks[0].Command != "markitect hook pretool" || pre[0].Hooks[0].If != "Bash(markitect mark*)" ||
		pre[1].Matcher != "Edit|Write|MultiEdit|NotebookEdit" || len(pre[1].Hooks) != 1 ||
		pre[1].Hooks[0].Command != "markitect hook pretool" || pre[1].Hooks[0].If != "" ||
		pre[2].Matcher != "Agent|Task" || len(pre[2].Hooks) != 1 ||
		pre[2].Hooks[0].Command != "markitect hook pretool" || pre[2].Hooks[0].If != "" {
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
	r.Write(".markitect/model/project.md", "---\nkind: std/Project\n---\n# Demo\n\nBeing edited.\n")
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
	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.0.0\n  rest: 1.0.0\n  web: 0.1.0\n")
	code, _, errOut := run(t, r.Dir, "tidy")
	if code != 1 || !strings.Contains(errOut, "library.missing") {
		t.Fatalf("lock without web: %d %s", code, errOut)
	}
	r.Write(".markitect/libraries/web/Page.md", "---\nkind: Kind\n---\n# Page\n")
	if out := mustRun(t, r.Dir, "tidy"); !strings.Contains(out, "rest 1.0.0, std 1.0.0, web 0.1.0") {
		t.Fatalf("lock = %s", out)
	}
	if lock := read(t, r.Dir, ".markitect/lock.yaml"); !strings.Contains(lock, "  web:\n    version: 0.1.0\n    digest: sha256:") {
		t.Fatalf("lock file:\n%s", lock)
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
	// For one release, the retired commands of the work say on standard output,
	// with exit 0, that markitect update is needed, so that an old project's skill
	// does not fail silently.
	for _, args := range [][]string{{"checklist", "--start"}, {"answer"}, {"gate", "--verify"}} {
		if code, out, _ := run(t, dir, args...); code != 0 || !strings.Contains(out, "markitect "+args[0]+" is gone") || !strings.Contains(out, "run markitect update") {
			t.Fatalf("%s: %d %s", args[0], code, out)
		}
	}
	// The commands step 2 of the tooling plan renamed are gone: vet, list, doc and
	// tidy take their place, and the commands stay flat.
	for _, cmd := range []string{"impact", "lint", "refs", "graph", "lock", "review"} {
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
	if code != 1 || !strings.Contains(out, "no record yet") || !strings.Contains(out, "look at: .markitect/model/project.md, README.md") {
		t.Fatalf("status: %d\n%s", code, out)
	}
	item := strings.Fields(out[strings.Index(out, "project@"):])[0]

	for _, args := range [][]string{
		{"mark", item, "maybe"},
		{"mark", "project"},
		{"mark", item, "fail"},
		{"mark", item, "ok"},
		{"mark", "project", "ok", "-e", "README.md"},
		{"mark", item, "ok", "README.md"},
		{"mark", item, "ok", "-e", "README.md", "--bogus"},
		{"mark", item, "ok", "-e"},
	} {
		if code, _, errOut := run(t, r.Dir, args...); code != 2 || !strings.Contains(errOut, "usage: markitect mark") {
			t.Fatalf("%v: %d %s", args, code, errOut)
		}
	}
	if code, out, _ := run(t, r.Dir, "mark", item, "ok", "-e", "tests/missing_test.go:3"); code != 1 || !strings.Contains(out, "tests/missing_test.go") {
		t.Fatalf("mark with missing evidence: %d\n%s", code, out)
	}
	// The flags may stand before the item and the verdict, too.
	wantAll(t, mustRun(t, r.Dir, "mark", "-e", "README.md:1 (TestReadme)", "-m", "says nothing of what the project is", item, "fail"), "Recorded: project fail.")
	wantAll(t, read(t, r.Dir, ".markitect/review.sum"), `"evidence":["README.md:1"]`)
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
	if code, out, _ := run(t, r.Dir, "mark", "--json", item, "ok", "-e", "README.md:3"); code != 0 || json.Unmarshal([]byte(out), &marked) != nil || marked.Kind != "mark" || !marked.Recorded {
		t.Fatalf("mark --json: %d %s", code, out)
	}
	r.Write("README.md", "# A project\n\nIt lends books, records and games.\n")
	if code, out, _ := run(t, r.Dir, "mark", item, "ok", "-e", "README.md:3", "--json"); code != 1 || json.Unmarshal([]byte(out), &marked) != nil || marked.Recorded || !strings.Contains(marked.Message, "Run markitect status again") {
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
// and says in its document when the target is neither a Definition nor a Package.
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
	r.Write(".markitect/model/big.md", "---\nkind: std/Rule\n---\n# Big\n\n"+strings.Repeat("word ", 1001)+"\n")
	var vetted struct {
		Diagnostics []struct{ Severity, Code string }
	}
	if code, out, _ := run(t, r.Dir, "vet", "--json", "--strict"); code != 1 || json.Unmarshal([]byte(out), &vetted) != nil || len(vetted.Diagnostics) != 1 || vetted.Diagnostics[0].Code != "vet.definition_size" {
		t.Fatalf("vet --json --strict: %d %s", code, out)
	}
	if code, _, errOut := run(t, r.Dir, "tidy", "--json"); code != 2 || !strings.Contains(errOut, "-json") {
		t.Fatalf("tidy --json: %d %s", code, errOut)
	}
}

// update turns the newest certificate of markitect 0.20 into the record, and
// removes the retired results folder.
func TestUpdateWritesTheRecordFromTheNewestCertificate(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	r.Write(".markitect/registry.md", "| File | Realizes | Why |\n|---|---|---|\n| README.md | [[project]] | Overview |\n")
	r.Write("README.md", "# A project\n")
	files := map[string]string{}
	spec := map[string]string{}
	for file, content := range snapshot(t, r.Dir) {
		sum := sha256.Sum256([]byte(content))
		files[file] = hex.EncodeToString(sum[:])
		if strings.HasPrefix(file, ".markitect/") {
			spec[file] = content
		}
	}
	cert, _ := json.Marshal(map[string]any{
		"kind": "markitect-certificate", "format": 1, "created": "2026-11-06T12:00:00Z", "files": files, "definitions": map[string]string{}, "spec": spec,
		"items": []map[string]any{{"item": "project", "kind": "definition", "definition": "project", "answer": "ok", "pointers": []map[string]any{{"file": "README.md", "line": 1}}}},
	})
	r.Write(".markitect/results/gate-20261106-120000-0a1b2c3d.json", string(cert))
	r.Write(".markitect/results/gate-20261106-120000-0a1b2c3d.md", "# Result\n")
	if code, out, _ := run(t, r.Dir, "check"); code != 1 || !strings.Contains(out, "project.results") {
		t.Fatalf("check with a results folder: %d\n%s", code, out)
	}
	code, out, _ := run(t, r.Dir, "update", "--check")
	if code != 1 || !strings.Contains(out, "record: the certificates under .markitect/results are retired") {
		t.Fatalf("update --check: %d\n%s", code, out)
	}
	wantAll(t, mustRun(t, r.Dir, "update", "--files"), "record: wrote .markitect/review.sum from the newest certificate", "record: removed the retired .markitect/results (2 files)")
	if code, out, _ := run(t, r.Dir, "status"); code != 0 || !strings.Contains(out, "Nothing to verify") {
		t.Fatalf("status after the migration: %d\n%s", code, out)
	}
	wantAll(t, read(t, r.Dir, ".markitect/review.sum"), `"evidence":["README.md:1"]`)
}
