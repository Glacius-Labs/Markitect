package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestLockAndCheckReadTheSameLibraryFiles(t *testing.T) {
	r := newFolder(t)
	mustRun(t, r.Dir, "init")
	if err := os.WriteFile(filepath.Join(r.Dir, ".gitignore"), []byte("*.swp\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.Write("README.md", "# Project\n")
	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 2.0.0\n  web: 0.1.0\n")
	r.Write(".markitect/libraries/web/Page.md", "---\nkind: Kind\n---\n# Page\n")
	r.Write(".markitect/libraries/web/Page.md.swp", "editor swap file\n")
	r.Write(".markitect/registry.md", read(t, r.Dir, ".markitect/registry.md")+"| `README.md` | [[project]] | Overview |\n")
	mustRun(t, r.Dir, "tidy")
	if code, out, _ := run(t, r.Dir, "check"); code != 0 {
		t.Fatalf("check after lock: %d\n%s", code, out)
	}
}

func TestInitAgainUpdatesSkills(t *testing.T) {
	r := newFolder(t)
	r.Write(".markitect/ignore", "# Legal text\nLICENSE\n")
	r.Write(".claude/skills/markitect-retired/SKILL.md", "an old skill\n")
	r.Write(".claude/skills/markitect-architect/old-notes.md", "an old file\n")
	r.Write(".claude/skills/markitect-start/SKILL.md", "a skill merged into markitect-architect\n")
	r.Write(".claude/skills/markitect-design/SKILL.md", "the skill markitect-architect replaced\n")
	r.Write(".claude/skills/my-own-skill/SKILL.md", "the user's own skill\n")
	mustRun(t, r.Dir, "init")
	for _, gone := range []string{".claude/skills/markitect-retired/SKILL.md", ".claude/skills/markitect-architect/old-notes.md", ".claude/skills/markitect-start/SKILL.md", ".claude/skills/markitect-design/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(r.Dir, filepath.FromSlash(gone))); !os.IsNotExist(err) {
			t.Errorf("%s was not removed", gone)
		}
	}
	if read(t, r.Dir, ".claude/skills/my-own-skill/SKILL.md") != "the user's own skill\n" {
		t.Error("init touched a skill that is not Markitect's")
	}
	if read(t, r.Dir, ".markitect/ignore") != "# Legal text\nLICENSE\n" {
		t.Error("init touched the ignore file")
	}
}

// A new project gets the instructions and no empty files; the guide is installed.
func TestInitWritesTheInstructionsAndNoEmptyFiles(t *testing.T) {
	r := newFolder(t)
	out := mustRun(t, r.Dir, "init", "--name", "Lending library")
	if !strings.Contains(read(t, r.Dir, "AGENTS.md"), "# Lending library") || read(t, r.Dir, "CLAUDE.md") != "@AGENTS.md\n" {
		t.Fatal("the instruction files are wrong")
	}
	if skill := ".claude/skills/markitect-architect/SKILL.md"; !strings.Contains(read(t, r.Dir, skill), "markitect-architect") {
		t.Fatalf("%s is not installed", skill)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, ".agents")); !os.IsNotExist(err) {
		t.Fatal("init wrote a skills folder for other coding agents, which Markitect does not support")
	}
	for _, none := range []string{".markitect/ignore", ".markitect/execution.yaml"} {
		if _, err := os.Stat(filepath.Join(r.Dir, filepath.FromSlash(none))); !os.IsNotExist(err) {
			t.Errorf("init wrote %s", none)
		}
	}
	if !strings.Contains(out, "/markitect-architect") {
		t.Fatalf("no next step:\n%s", out)
	}
	if out := mustRun(t, r.Dir, "check", "--spec"); !strings.HasPrefix(out, "ok:") {
		t.Fatalf("check --spec on a new project: %s", out)
	}
}

// Instructions that exist stay as they are.
func TestInitLeavesExistingInstructions(t *testing.T) {
	r := newFolder(t)
	r.Write("AGENTS.md", "my own\n")
	r.Write("CLAUDE.md", "mine too\n")
	out := mustRun(t, r.Dir, "init")
	if read(t, r.Dir, "AGENTS.md") != "my own\n" || read(t, r.Dir, "CLAUDE.md") != "mine too\n" {
		t.Fatal("init changed the instructions")
	}
	if !strings.Contains(out, "left AGENTS.md as it is") {
		t.Fatalf("no notice:\n%s", out)
	}
}

// An empty folder is all init needs, and it stays a plain folder. In a folder of
// the project, init works on the project, not on a new one.
func TestInitInAnEmptyFolder(t *testing.T) {
	dir := testkit.TempDir(t)
	mustRun(t, dir, "init", "--name", "New")
	noRepository(t, dir)
	sub := filepath.Join(dir, "src", "a")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mustRun(t, sub, "init")
	if _, err := os.Stat(filepath.Join(sub, ".markitect")); !os.IsNotExist(err) {
		t.Fatalf("init made a project inside the project: %v", err)
	}
}
