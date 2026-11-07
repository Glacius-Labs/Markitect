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
	r.Write("AGENTS.md", "# Mine\n\nWritten by the owner.\n")
	spec := read(t, r.Dir, ".markitect/model/project.md")
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
	wantAll(t, settings, "Bash(go test *)", "my-own-check", "Bash(markitect review*)", "Edit|Write|MultiEdit|NotebookEdit")
	if strings.Contains(settings, "Bash(markitect answer*)") {
		t.Fatalf("the older hook was not replaced:\n%s", settings)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, ".claude", "skills", "markitect-retired", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("the retired skill stayed: %v", err)
	}
	if read(t, r.Dir, "AGENTS.md") != "# Mine\n\nWritten by the owner.\n" || read(t, r.Dir, ".markitect/model/project.md") != spec {
		t.Fatal("update changed a file of the owner's")
	}

	after := snapshot(t, r.Dir)
	out = mustRun(t, r.Dir, "update")
	wantAll(t, out, "project: up to date")
	if diff := changed(after, snapshot(t, r.Dir)); len(diff) > 0 || strings.Contains(out, "commit") {
		t.Fatalf("a second update changed %v:\n%s", diff, out)
	}
}

// The model the manifest names is the agents' model: update writes it into their
// files, and until then every command says the harness is out of date.
func TestTheManifestNamesTheAgentsModel(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	wantAll(t, read(t, r.Dir, ".claude/agents/markitect-builder.md"), "\nmodel: sonnet\n")
	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.0.0\nmodel: opus\n")
	if _, _, errOut := run(t, r.Dir, "status"); !strings.Contains(errOut, "run markitect update") {
		t.Fatalf("no note of the changed model:\n%s", errOut)
	}
	mustRun(t, r.Dir, "update", "--files")
	for _, agent := range []string{"markitect-builder", "markitect-reviewer"} {
		wantAll(t, read(t, r.Dir, ".claude/agents/"+agent+".md"), "\nmodel: opus\n")
	}
	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.0.0\nmodel: \"opus; rm -rf /\"\n")
	mustRun(t, r.Dir, "update", "--files")
	wantAll(t, read(t, r.Dir, ".claude/agents/markitect-builder.md"), "\nmodel: sonnet\n")
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
	for _, cmd := range []string{"check", "lint", "status"} {
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
	for _, cmd := range []string{"check", "lint", "graph", "status"} {
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
