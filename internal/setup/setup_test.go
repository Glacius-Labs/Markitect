package setup

import (
	"bytes"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/guard"
	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// memory is a project folder in memory: its files by path.
type memory map[string]string

func (m memory) Files() ([]string, error) {
	var files []string
	for f := range m {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

func (m memory) Read(file string) ([]byte, error) {
	if data, ok := m[file]; ok {
		return []byte(data), nil
	}
	return nil, fs.ErrNotExist
}

func (m memory) GitIgnored(string) bool { return false }

func (m memory) Write(file string, data []byte) error {
	m[file] = string(data)
	return nil
}

func (m memory) Exists(file string) bool {
	_, ok := m[file]
	return ok
}

func (m memory) List(dir string) ([]string, error) {
	var files []string
	for f := range m {
		if strings.HasPrefix(f, dir+"/") {
			files = append(files, f)
		}
	}
	sort.Strings(files)
	return files, nil
}

func (m memory) Remove(file string) error {
	delete(m, file)
	return nil
}

// A new project gets its first files, and a second run changes nothing.
func TestStartSpecAndInstructions(t *testing.T) {
	m := memory{}
	created, empty, err := StartSpec(m, "Library")
	if err != nil || !empty || !reflect.DeepEqual(created, []string{project.ManifestFile, project.SpecDir + "/project.md", project.RootPage}) {
		t.Fatalf("created %v, empty spec %v, %v", created, empty, err)
	}
	if !strings.Contains(m[project.SpecDir+"/project.md"], "# Library\n") {
		t.Fatalf("anchor:\n%s", m[project.SpecDir+"/project.md"])
	}
	kept, created, err := WriteInstructions(m, "Library")
	if err != nil || kept || !reflect.DeepEqual(created, []string{InstructionsFile, "CLAUDE.md"}) || m["CLAUDE.md"] != "@AGENTS.md\n" {
		t.Fatalf("kept %v, created %v, %v", kept, created, err)
	}
	if created, empty, err := StartSpec(m, "Library"); err != nil || empty || len(created) != 0 {
		t.Fatalf("again: created %v, empty spec %v, %v", created, empty, err)
	}
	if kept, created, err := WriteInstructions(m, "Library"); err != nil || !kept || len(created) != 0 {
		t.Fatalf("again: kept %v, created %v, %v", kept, created, err)
	}
}

// The ignore line for Claude Code's worktrees is added once, after a blank line
// and with the file's line endings, unless a line already ignores the worktrees,
// or keeps them, which is the owner's choice.
func TestEnsureIgnore(t *testing.T) {
	const lines = "# the worktrees Claude Code makes, as for Markitect's builders\n.claude/worktrees/\n"
	for before, after := range map[string]string{
		"":                          lines,
		"bin/\r\n":                  "bin/\r\n\r\n" + strings.ReplaceAll(lines, "\n", "\r\n"),
		"bin/\n":                    "bin/\n\n" + lines,
		"node_modules":              "node_modules\n\n" + lines,
		"/.claude/\n":               "/.claude/\n",
		".claude/worktrees":         ".claude/worktrees",
		"!.claude/worktrees/\n":     "!.claude/worktrees/\n",
		".claude/*\n":               ".claude/*\n",
		"**/.claude/worktrees/**\n": "**/.claude/worktrees/**\n",
	} {
		m := memory{}
		if before != "" {
			m[GitIgnoreFile] = before
		}
		added, err := EnsureIgnore(m)
		if err != nil || added != (before != after) || m[GitIgnoreFile] != after {
			t.Errorf("made %q of %q (added %v, %v), not %q", m[GitIgnoreFile], before, added, err, after)
		}
		if added, _ := EnsureIgnore(m); added {
			t.Errorf("added the line to %q twice", before)
		}
	}
}

// The hooks and the worktree base go into the settings beside what they hold,
// in place of an earlier markitect's hooks; settings it cannot read stay as they
// are, and a part with another shape stays while the other is written.
func TestEnsureSettings(t *testing.T) {
	const (
		addedHooks = "added the hooks that hold the agents' rights to .claude/settings.json"
		setBase    = `set worktree.baseRef to "head" in .claude/settings.json, so that a builder's worktree starts from your branch`
	)
	old := `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "` + guard.HookCommand + `"}]}]}}`
	for name, c := range map[string]struct {
		before string
		said   []string
		has    []string
		hooks  int
	}{
		"none":         {"", []string{addedHooks, setBase}, []string{`"baseRef": "head"`}, 4},
		"kept":         {"{\n  \"permissions\": {\"allow\": [\"Bash(go test *)\"]},\n  \"cleanupPeriodDays\": 12345678901234567890\n}\n", []string{addedHooks, setBase}, []string{"Bash(go test *)", "12345678901234567890"}, 4},
		"owner's base": {`{"worktree": {"baseRef": "fresh", "sparsePaths": ["src"]}}`, []string{addedHooks}, []string{`"baseRef": "fresh"`, "sparsePaths"}, 4},
		"earlier hook": {old, []string{addedHooks, setBase}, nil, 4},
		"a mention":    {`{"hooks": {"PostToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "markitect hook pretool"}]}]}}`, []string{addedHooks, setBase}, []string{"PostToolUse"}, 4},
		"hooks not a map": {`{"hooks": ["something"]}`, []string{
			`left the hooks in .claude/settings.json as they are, because they are not an object; add the hooks by hand: "` + strings.ReplaceAll(guard.HookCommand, `"`, `\"`) + `" before Bash for Bash(markitect mark*), before PowerShell for PowerShell(markitect mark*), before Edit|Write|MultiEdit|NotebookEdit and before Agent|Task|Workflow`,
			setBase,
		}, []string{`"something"`, `"baseRef": "head"`}, 0},
		// What it left comes first, then what it wrote.
		"worktree not a map": {`{"worktree": "x"}`, []string{
			`left worktree in .claude/settings.json as it is, because it is not an object; set worktree.baseRef to "head" by hand`,
			addedHooks,
		}, []string{`"worktree": "x"`}, 4},
	} {
		t.Run(name, func(t *testing.T) {
			m := memory{}
			if c.before != "" {
				m[guard.SettingsFile] = c.before
			}
			said, err := EnsureSettings(m)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(said, "\n") != strings.Join(c.said, "\n") {
				t.Fatalf("said:\n%s\nwant:\n%s", strings.Join(said, "\n"), strings.Join(c.said, "\n"))
			}
			after := m[guard.SettingsFile]
			for _, want := range c.has {
				if !strings.Contains(after, want) {
					t.Fatalf("%q not in:\n%s", want, after)
				}
			}
			if n := strings.Count(after, guard.HookCommand); n != c.hooks || (c.hooks > 0) != guard.HasGuard([]byte(after)) {
				t.Fatalf("%d hooks, want %d:\n%s", n, c.hooks, after)
			}
			// Run again, it writes nothing and says again only what it left.
			var left []string
			for _, line := range c.said {
				if strings.HasPrefix(line, "left ") {
					left = append(left, line)
				}
			}
			if said, err := EnsureSettings(m); err != nil || !reflect.DeepEqual(said, left) || m[guard.SettingsFile] != after {
				t.Fatalf("again said %v (%v):\n%s", said, err, m[guard.SettingsFile])
			}
		})
	}
	for _, unreadable := range []string{"{ not json\n", "null\n", "{} {}\n"} {
		m := memory{guard.SettingsFile: unreadable}
		said, err := EnsureSettings(m)
		if err != nil || len(said) != 1 || !strings.HasPrefix(said[0], "left .claude/settings.json as it is, because ") ||
			!strings.Contains(said[0], "add the hooks by hand") || !strings.HasSuffix(said[0], `set worktree.baseRef to "head" by hand`) || m[guard.SettingsFile] != unreadable {
			t.Fatalf("%q: said %v (%v), settings %q", unreadable, said, err, m[guard.SettingsFile])
		}
	}
}

// A dry refresh finds what differs and changes nothing; a real one writes it.
func TestRefreshHarness(t *testing.T) {
	cfg := harness.Resolve(harness.File{}, harness.File{})
	m := memory{}
	if files, _, err := RefreshHarness(m, cfg, false); err != nil || len(files) == 0 {
		t.Fatalf("first refresh: %v, %v", files, err)
	}
	if files, said, err := RefreshHarness(m, cfg, true); err != nil || len(files) != 0 || len(said) != 0 {
		t.Fatalf("a current project differs: %v, %v, %v", files, said, err)
	}
	agent := ".claude/agents/markitect-builder.md"
	m[agent] = "changed\n"
	m[".claude/skills/markitect-retired/SKILL.md"] = "retired\n"
	files, _, err := RefreshHarness(m, cfg, true)
	if want := []string{agent, ".claude/skills/markitect-retired/SKILL.md"}; err != nil || !reflect.DeepEqual(files, want) {
		t.Fatalf("dry: %v (%v), want %v", files, err, want)
	}
	if m[agent] != "changed\n" || !m.Exists(".claude/skills/markitect-retired/SKILL.md") {
		t.Fatal("a dry refresh changed the folder")
	}
	if _, _, err := RefreshHarness(m, cfg, false); err != nil || m[agent] == "changed\n" || m.Exists(".claude/skills/markitect-retired/SKILL.md") {
		t.Fatalf("the refresh left the folder as it was (%v)", err)
	}
	if !SameText([]byte("a\r\nb\r\n"), []byte("a\nb\n")) || SameText([]byte("a\n"), []byte("b\n")) {
		t.Fatal("SameText minds line endings or misses a change")
	}
}

// The agent files ship with the built-in model and effort of their role, so the
// files in the repository and the values init writes cannot drift apart.
func TestShippedAgentFilesCarryTheBuiltInValues(t *testing.T) {
	builtIn := harness.Resolve(harness.File{}, harness.File{})
	agents := 0
	for _, f := range skills.Harness() {
		role, ok := harness.RoleOf(f.Path)
		if !ok {
			continue
		}
		agents++
		if !bytes.Equal(builtIn.RenderAgent(role, f.Data), f.Data) {
			model, effort := builtIn.Agent(role)
			t.Errorf("%s does not name the built-in values of its role, %s and %s", f.Path, model, effort)
		}
	}
	if agents != len(harness.Roles) {
		t.Errorf("found %d agent files for %d roles", agents, len(harness.Roles))
	}
}
