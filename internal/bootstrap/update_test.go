package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// emptyHome gives a test a home folder of its own, so that nothing reads or writes
// the developer's Claude Code folder.
func emptyHome(t *testing.T) string {
	t.Helper()
	home := testkit.TempDir(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	return filepath.Join(home, ".claude", "skills", "markitect", "SKILL.md")
}

func install(t *testing.T) {
	t.Helper()
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"install"}, &out, &errOut); code != 0 {
		t.Fatalf("install: %d %s", code, errOut.String())
	}
}

func wantAll(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("lacks %q:\n%s", w, got)
		}
	}
}

// The hooks of 0.25.2 and before call markitect by name, which fails where the
// hook's shell finds none, as with the plugin alone; update puts the current ones
// in their place, and init leaves a project with them as it is.
func TestUpdateReplacesTheBareHooks(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	settings := read(t, r.Dir, ".claude/settings.json")
	bare := strings.ReplaceAll(settings, projectHook, "markitect hook pretool")
	if bare == settings {
		t.Fatalf("init wrote no current hook:\n%s", settings)
	}
	r.Write(".claude/settings.json", bare)
	code, out, _ := run(t, r.Dir, "update", "--check", "--files")
	if code != 1 || !strings.Contains(out, ".claude/settings.json") {
		t.Fatalf("update --check = %d\n%s", code, out)
	}
	mustRun(t, r.Dir, "update", "--files")
	if got := read(t, r.Dir, ".claude/settings.json"); got != settings {
		t.Fatalf("update left:\n%s\nwant:\n%s", got, settings)
	}
	if out := mustRun(t, r.Dir, "init"); strings.Contains(out, "added the hooks") || read(t, r.Dir, ".claude/settings.json") != settings {
		t.Fatalf("init again:\n%s", out)
	}
}

// update puts a project made by an older markitect right: it rewrites the skills,
// agents and hook that differ and removes the ones the binary no longer ships,
// commits nothing, and touches nothing of the owner's.
func TestUpdateBringsAProjectInLine(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Demo")
	if out := mustRun(t, r.Dir, "update", "--check"); !strings.Contains(out, "project: up to date") {
		t.Fatalf("a fresh project is not up to date:\n%s", out)
	}

	r.Write(".claude/agents/markitect-builder.md", "from an older markitect\n")
	// The hook an older markitect wrote guards answer; the owner's own stays.
	r.Write(".claude/settings.json", `{
  "permissions": {"allow": ["Bash(go test *)"]},
  "hooks": {"PreToolUse": [
    {"matcher": "Bash", "hooks": [{"type": "command", "if": "Bash(markitect answer*)", "command": "markitect hook pretool"}]},
    {"matcher": "Edit", "hooks": [{"type": "command", "command": "my-own-check"}]}
  ]}
}
`)
	r.Write(".claude/skills/markitect-retired/SKILL.md", "no longer shipped\n")
	if err := os.Remove(filepath.Join(r.Dir, ".claude", "skills", "markitect-build", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	r.Write("AGENTS.md", "# Mine\n\nWritten by the owner, who models with the `markitect-modeling` skill.\n")
	spec := read(t, r.Dir, ".markitect/spec/project.md")
	before := snapshot(t, r.Dir)

	code, out, _ := run(t, r.Dir, "update", "--check")
	if code != 1 {
		t.Fatalf("update --check = %d\n%s", code, out)
	}
	wantAll(t, out, "project: 4 files out of date:",
		".claude/agents/markitect-builder.md", ".claude/settings.json",
		".claude/skills/markitect-build/SKILL.md", ".claude/skills/markitect-retired/SKILL.md")
	if strings.Contains(out, "lock:") {
		t.Fatalf("the lock was fine, yet update --check spoke of it:\n%s", out)
	}
	if diff := changed(before, snapshot(t, r.Dir)); len(diff) > 0 {
		t.Fatalf("update --check changed something: %v", diff)
	}

	out = mustRun(t, r.Dir, "update")
	wantAll(t, out, "project: updated 4 files:", "Markitect uses no Git: commit them when you like.")
	noRepository(t, r.Dir)
	settings := read(t, r.Dir, ".claude/settings.json")
	wantAll(t, settings, "Bash(go test *)", "my-own-check", "Bash(markitect mark*)", "Edit|Write|MultiEdit|NotebookEdit", "Agent|Task|Workflow", `"baseRef": "head"`, projectHook)
	if strings.Contains(settings, "Bash(markitect answer*)") {
		t.Fatalf("the older hook was not replaced:\n%s", settings)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, ".claude", "skills", "markitect-retired", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("the retired skill stayed: %v", err)
	}
	// It never changes AGENTS.md, and says when it names a skill that is gone.
	wantAll(t, out, "note: AGENTS.md names the skill markitect-modeling, which markitect-design replaced")
	if read(t, r.Dir, "AGENTS.md") != "# Mine\n\nWritten by the owner, who models with the `markitect-modeling` skill.\n" || read(t, r.Dir, ".markitect/spec/project.md") != spec {
		t.Fatal("update changed a file of the owner's")
	}

	after := snapshot(t, r.Dir)
	out = mustRun(t, r.Dir, "update")
	wantAll(t, out, "project: up to date")
	if diff := changed(after, snapshot(t, r.Dir)); len(diff) > 0 || strings.Contains(out, "commit") {
		t.Fatalf("a second update changed %v:\n%s", diff, out)
	}
}

// update moves the spec from .markitect/model/, where an earlier markitect kept
// it, to .markitect/spec/, and the record keeps every verdict, also one whose
// evidence is a file of the spec. Until then check names the move and init
// refuses; a second spec beside the first is refused.
func TestUpdateMovesTheSpecFolder(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Demo")
	r.Write(".markitect/spec/members/member.md", "---\nkind: std/Concept\n---\n# Member\n\nA person who borrows.\n")
	r.Write(".markitect/registry.md", read(t, r.Dir, ".markitect/registry.md")+"| `README.md` | [[project]] | Overview |\n")
	r.Write(".markitect/registry/members.md", "---\npaths: [src/members/**]\n---\n# Registry: members\n\n| File | Realizes | Why |\n|---|---|---|\n| `src/members/member.go` | [[members/member]] | The type |\n")
	r.Write("README.md", "# Demo\n")
	r.Write("src/members/member.go", "package members\n")
	verifyAll(t, r.Dir)
	_, out, _ := run(t, r.Dir, "status", "--all")
	item := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "members/member@") {
			item = line
		}
	}
	mustRun(t, r.Dir, "mark", item, "ok", "-e", ".markitect/spec/members/member.md:5", "-e", "src/members/member.go:1")
	record := read(t, r.Dir, ".markitect/review.sum")
	wantAll(t, record, `"evidence:.markitect/spec/members/member.md":"sha256:`)

	// The project as an earlier markitect left it.
	if err := os.Rename(filepath.Join(r.Dir, ".markitect", "spec"), filepath.Join(r.Dir, ".markitect", "model")); err != nil {
		t.Fatal(err)
	}
	r.Write(".markitect/review.sum", strings.ReplaceAll(record, ".markitect/spec/", ".markitect/model/"))
	r.Write("AGENTS.md", strings.ReplaceAll(read(t, r.Dir, "AGENTS.md"), ".markitect/spec/", ".markitect/model/"))
	if code, out, _ := run(t, r.Dir, "check"); code != 1 || !strings.HasPrefix(out, ".markitect/model: project.model: an earlier markitect kept the spec here; markitect update moves it to .markitect/spec\n1 error: 1 in the spec") {
		t.Fatalf("check before the move: %d\n%s", code, out)
	}
	before := snapshot(t, r.Dir)
	if code, out, errOut := run(t, r.Dir, "init"); code != 2 || out != "" || !strings.Contains(errOut, "run markitect update, which moves it to .markitect/spec") {
		t.Fatalf("init before the move: %d\n%s%s", code, out, errOut)
	}
	wantAll(t, mustRun(t, r.Dir, "version"), "project: out of date: the spec is still in .markitect/model; run markitect update")
	code, out, _ := run(t, r.Dir, "update", "--check")
	if code != 1 || !strings.Contains(out, "spec: still in .markitect/model, where an earlier markitect kept it; markitect update moves it to .markitect/spec and keeps the record") {
		t.Fatalf("update --check: %d\n%s", code, out)
	}
	if diff := changed(before, snapshot(t, r.Dir)); len(diff) > 0 {
		t.Fatalf("init or update --check changed something: %v", diff)
	}

	out = mustRun(t, r.Dir, "update")
	wantAll(t, out, "spec: moved .markitect/model to .markitect/spec\n", "record: renamed the evidence in .markitect/model in 1 line of .markitect/review.sum\n",
		"note: AGENTS.md names .markitect/model, where an earlier markitect kept the spec; it is .markitect/spec now")
	if read(t, r.Dir, ".markitect/review.sum") != record {
		t.Fatalf("the record is not the one verified:\n%s", read(t, r.Dir, ".markitect/review.sum"))
	}
	if code, out, _ := run(t, r.Dir, "status"); code != 0 || !strings.Contains(out, "Nothing to verify") {
		t.Fatalf("a verdict was lost in the move: %d\n%s", code, out)
	}
	// A second update finds nothing to move.
	if out := mustRun(t, r.Dir, "update"); !strings.Contains(out, "project: up to date") || strings.Contains(out, "spec:") || strings.Contains(out, "record:") {
		t.Fatalf("a second update:\n%s", out)
	}

	// A second spec beside the moved one is refused, and nothing changes.
	r.Write(".markitect/model/members/member.md", "---\nkind: std/Concept\n---\n# Member\n\nAn older text.\n")
	before = snapshot(t, r.Dir)
	if code, out, errOut := run(t, r.Dir, "update"); code != 2 || !strings.Contains(errOut, "both .markitect/model and .markitect/spec hold files") {
		t.Fatalf("update with two specs: %d\n%s%s", code, out, errOut)
	}
	if diff := changed(before, snapshot(t, r.Dir)); len(diff) > 0 {
		t.Fatalf("the refusal changed something: %v", diff)
	}
}

// The agents' model an earlier manifest names moves to the project's harness
// configuration (DEC-103): update moves it, keeps a model the configuration
// already sets, and writes the agents' files from it. A model that was never a
// name is dropped, as it was ignored.
func TestTheManifestsModelMovesToTheHarnessConfiguration(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	wantAll(t, read(t, r.Dir, ".claude/agents/markitect-builder.md"), "\nmodel: sonnet\neffort: high\n")
	if strings.Contains(read(t, r.Dir, ".markitect/manifest.yaml"), "model") {
		t.Fatal("a new manifest still speaks of the model")
	}
	r.Write(".markitect/manifest.yaml", "# The Libraries this project uses, with their versions. Run markitect tidy after a change.\n# model, optional, names the model Markitect's agents use; without it they use sonnet.\n# markitect update writes it into their agent files.\nlibraries:\n  std: 1.0.0\nmodel: opus\n")
	r.Write(".markitect/harness.yaml", "agents:\n  reviewer:\n    model: haiku\n")
	if code, out, _ := run(t, r.Dir, "check"); code != 1 || !strings.Contains(out, "manifest.model") {
		t.Fatalf("check said nothing of the model: %d\n%s", code, out)
	}
	code, out, _ := run(t, r.Dir, "update", "--check", "--files")
	if code != 1 || !strings.Contains(out, "manifest: names the agents' model") || strings.Contains(out, "lock: not rewritten") {
		t.Fatalf("update --check: %d\n%s", code, out)
	}
	mustRun(t, r.Dir, "tidy")
	out = mustRun(t, r.Dir, "update", "--files")
	wantAll(t, out, "manifest: moved the agents' model to .markitect/harness.yaml")
	if got := read(t, r.Dir, ".markitect/manifest.yaml"); got != "# The Libraries this project uses, with their versions. Run markitect tidy after a change.\nlibraries:\n  std: 1.0.0\n" {
		t.Fatalf("the manifest: %q", got)
	}
	for agent, model := range map[string]string{"builder": "opus", "architect": "opus", "auditor": "opus", "reviewer": "haiku"} {
		wantAll(t, read(t, r.Dir, ".claude/agents/markitect-"+agent+".md"), "\nmodel: "+model+"\n")
	}
	mustRun(t, r.Dir, "check")
	if out := mustRun(t, r.Dir, "update", "--files"); !strings.Contains(out, "project: up to date") {
		t.Fatalf("a second update:\n%s", out)
	}

	// init moves it too, when it runs again on an earlier project.
	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.0.0\nmodel: haiku\n")
	if err := os.Remove(filepath.Join(r.Dir, ".markitect", "harness.yaml")); err != nil {
		t.Fatal(err)
	}
	out = mustRun(t, r.Dir, "init")
	wantAll(t, out, "moved the agents' model from .markitect/manifest.yaml to .markitect/harness.yaml")
	wantAll(t, read(t, r.Dir, ".claude/agents/markitect-reviewer.md"), "\nmodel: haiku\n")

	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.0.0\nmodel: \"opus; rm -rf /\"\n")
	mustRun(t, r.Dir, "update", "--files")
	if got := read(t, r.Dir, ".markitect/manifest.yaml"); strings.Contains(got, "model") {
		t.Fatalf("the bad model stayed: %q", got)
	}
	if strings.Contains(read(t, r.Dir, ".claude/agents/markitect-builder.md"), "rm -rf") {
		t.Fatal("the bad model reached an agent file")
	}
}

// A checkout may hold a file with CRLF; that is the same file.
func TestUpdateIgnoresLineEndings(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	agent := read(t, r.Dir, ".claude/agents/markitect-builder.md")
	r.Write(".claude/agents/markitect-builder.md", strings.ReplaceAll(agent, "\n", "\r\n"))
	if out := mustRun(t, r.Dir, "update", "--check"); !strings.Contains(out, "project: up to date") {
		t.Fatalf("CRLF counted as a difference:\n%s", out)
	}
}

// The lock pins the Libraries the spec is checked against, so update rewrites it
// and says to read it before committing. A manifest it cannot lock fails.
func TestUpdateLeavesTheLockForReview(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	lock := read(t, r.Dir, ".markitect/lock.yaml")
	r.Write(".markitect/lock.yaml", lock+"# from an older std\n")

	code, out, _ := run(t, r.Dir, "update", "--check")
	if code != 1 || !strings.Contains(out, "lock: out of date") {
		t.Fatalf("update --check = %d\n%s", code, out)
	}
	out = mustRun(t, r.Dir, "update")
	wantAll(t, out, "lock: rewritten; read the change of .markitect/lock.yaml before you commit it")
	if read(t, r.Dir, ".markitect/lock.yaml") != lock {
		t.Fatal("the lock was not rewritten")
	}
	if strings.Contains(out, "project: updated") {
		t.Fatalf("update spoke of files with only the lock to write:\n%s", out)
	}

	r.Write(".markitect/manifest.yaml", "libraries:\n  nosuch: 1.0.0\n")
	code, out, errOut := run(t, r.Dir, "update")
	if code != 1 || !strings.Contains(out, "lock: not rewritten") || errOut == "" {
		t.Fatalf("update with a bad manifest = %d\n%s\n%s", code, out, errOut)
	}
}

// The personal skill is refreshed when there is one; update does not create it, since
// install is the command that writes into the home folder for the first time.
func TestUpdateRefreshesThePersonalSkill(t *testing.T) {
	file := emptyHome(t)
	dir := testkit.TempDir(t) // in no project

	out := mustRun(t, dir, "update")
	wantAll(t, out, "personal skill: not installed; markitect install writes", "project: none here (no folder here or above holds .markitect/manifest.yaml)")
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("update created the personal skill: %v", err)
	}

	install(t)
	fresh, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if out := mustRun(t, dir, "update", "--check"); !strings.Contains(out, "personal skill: up to date") {
		t.Fatalf("a fresh personal skill is not up to date:\n%s", out)
	}
	if err := os.WriteFile(file, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := run(t, dir, "update", "--check"); code != 1 || !strings.Contains(out, "personal skill: out of date") {
		t.Fatalf("update --check on an old skill = %d\n%s", code, out)
	}
	if got, _ := os.ReadFile(file); string(got) != "old\n" {
		t.Fatalf("update --check wrote the skill: %s", got)
	}
	wantAll(t, mustRun(t, dir, "update"), "personal skill: refreshed")
	if got, _ := os.ReadFile(file); !bytes.Equal(got, fresh) {
		t.Fatalf("update did not refresh the skill:\n%s", got)
	}
	var stdout, stderr bytes.Buffer
	if Run(context.Background(), []string{"update", "extra"}, &stdout, &stderr) != 2 {
		t.Fatal("update takes no argument")
	}
}

// version says which markitect this is and whether what it wrote earlier matches.
func TestVersionSaysWhatIsOutOfDate(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")

	out := mustRun(t, r.Dir, "version")
	if !strings.HasPrefix(out, "markitect ") {
		t.Fatalf("version = %s", out)
	}
	wantAll(t, out, "personal skill: not installed; run markitect install", "project: up to date")

	r.Write(".claude/agents/markitect-reviewer.md", "from an older markitect\n")
	wantAll(t, mustRun(t, r.Dir, "version"), "project: out of date (1 files); run markitect update")

	wantAll(t, mustRun(t, testkit.TempDir(t), "version"), "project: none here")
}

// Any command that reads the project says once that its skills and agents are old.
func TestACommandSaysWhenTheHarnessIsOld(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	if _, _, errOut := run(t, r.Dir, "check"); strings.Contains(errOut, "markitect update") {
		t.Fatalf("a current project got the notice:\n%s", errOut)
	}
	r.Write(".claude/agents/markitect-builder.md", "from an older markitect\n")
	for _, cmd := range []string{"check", "vet", "status"} {
		if _, _, errOut := run(t, r.Dir, cmd); !strings.Contains(errOut, "run markitect update") {
			t.Errorf("%s said nothing about the old harness:\n%s", cmd, errOut)
		}
	}
	mustRun(t, r.Dir, "update")
	if _, _, errOut := run(t, r.Dir, "check"); strings.Contains(errOut, "markitect update") {
		t.Fatalf("update did not settle the notice:\n%s", errOut)
	}
}

// A folder with no project above it is no project; the error says how to make it
// one. The commands of the Run are gone.
func TestACommandOutsideAProjectPointsToInit(t *testing.T) {
	dir := testkit.TempDir(t)
	for _, cmd := range []string{"check", "vet", "list", "status"} {
		code, _, errOut := run(t, dir, cmd)
		if code != 2 || !strings.Contains(errOut, "is in no Markitect project") || !strings.Contains(errOut, "markitect init makes one") {
			t.Errorf("%s outside a project: %d\n%s", cmd, code, errOut)
		}
	}
	for _, cmd := range []string{"apply", "verify", "submit", "stop"} {
		if code, _, errOut := run(t, dir, cmd); code != 2 || !strings.Contains(errOut, "unknown command") {
			t.Errorf("%s: %d\n%s", cmd, code, errOut)
		}
	}
}

// init never writes to the home folder, so it says what install would add.
func TestInitSaysWhenThePersonalSkillIsMissing(t *testing.T) {
	file := emptyHome(t)
	r := newFolder(t)
	wantAll(t, mustRun(t, r.Dir, "init"), "tip: markitect install")
	install(t)
	if out := mustRun(t, r.Dir, "init"); strings.Contains(out, "tip:") {
		t.Fatalf("init gave a tip with the skill in place:\n%s", out)
	}
	if err := os.WriteFile(file, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantAll(t, mustRun(t, r.Dir, "init"), "tip: your personal skill comes from another markitect; markitect update")
}

func TestPathWarning(t *testing.T) {
	dir := testkit.TempDir(t)
	exe := filepath.Join(dir, "markitect")
	other := filepath.Join(dir, "other", "markitect")
	for _, f := range []string{exe, other} {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte(f), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	self := func() (string, error) { return exe, nil }
	found := func(path string) func(string) (string, error) {
		return func(string) (string, error) { return path, nil }
	}
	if got := pathWarning(found(exe), self); got != "" {
		t.Errorf("the PATH leads here, yet: %s", got)
	}
	got := pathWarning(func(string) (string, error) { return "", errors.New("not found") }, self)
	wantAll(t, got, "not on your PATH", dir)
	wantAll(t, pathWarning(found(other), self), "the markitect on your PATH is "+other, "not this one")
	if got := pathWarning(found(exe), func() (string, error) { return "", errors.New("no") }); got != "" {
		t.Errorf("without a path of its own: %s", got)
	}
}

func TestUnderTheTemporaryFolder(t *testing.T) {
	dir := testkit.TempDir(t)
	if !under(filepath.Join(dir, "go-build", "markitect"), dir) || under(dir, filepath.Join(dir, "go-build")) {
		t.Fatal("under mixes up the folders")
	}
	if under(filepath.Join(dir+"x", "markitect"), dir) {
		t.Fatal("a folder with the same start counts as inside")
	}
}
