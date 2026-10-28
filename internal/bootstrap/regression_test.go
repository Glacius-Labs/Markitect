package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestLockAndCheckReadTheSameLibraryFiles(t *testing.T) {
	r := testkit.NewRepo(t)
	mustRun(t, r.Dir, "init")
	r.Write(".gitignore", "*.swp\n")
	r.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.0.0\n  web: 0.1.0\n")
	r.Write(".markitect/libraries/web/Page.md", "---\nkind: Kind\n---\n# Page\n")
	r.Write(".markitect/libraries/web/Page.md.swp", "editor swap file\n")
	r.Write(".markitect/registry.md", read(t, r.Dir, ".markitect/registry.md")+"| `.gitignore` | [[project]] | Git settings |\n")
	mustRun(t, r.Dir, "lock")
	if code, out, _ := run(t, r.Dir, "check"); code != 0 {
		t.Fatalf("check after lock: %d\n%s", code, out)
	}
}

func TestInitAgainUpdatesSkillsAndIgnorePage(t *testing.T) {
	r := testkit.NewRepo(t)
	r.Write(".markitect/ignore.md", "# Ignored\n\n| Path | Why |\n|---|---|\n| LICENSE | Legal text |\n")
	r.Write(".claude/skills/markitect-retired/SKILL.md", "an old skill\n")
	r.Write(".claude/skills/markitect-modeling/old-notes.md", "an old file\n")
	r.Write(".claude/skills/my-own-skill/SKILL.md", "the user's own skill\n")
	mustRun(t, r.Dir, "init")
	for _, gone := range []string{".claude/skills/markitect-retired/SKILL.md", ".claude/skills/markitect-modeling/old-notes.md"} {
		if _, err := os.Stat(filepath.Join(r.Dir, filepath.FromSlash(gone))); !os.IsNotExist(err) {
			t.Errorf("%s was not removed", gone)
		}
	}
	if read(t, r.Dir, ".claude/skills/my-own-skill/SKILL.md") != "the user's own skill\n" {
		t.Error("init touched a skill that is not Markitect's")
	}
	ignore := read(t, r.Dir, ".markitect/ignore.md")
	if !strings.Contains(ignore, "| LICENSE | Legal text |") || !strings.Contains(ignore, "`.claude/skills/markitect-*/**`") {
		t.Fatalf("ignore page:\n%s", ignore)
	}
	mustRun(t, r.Dir, "init")
	if again := read(t, r.Dir, ".markitect/ignore.md"); again != ignore {
		t.Fatalf("a second init changed the ignore page:\n%s", again)
	}
}
