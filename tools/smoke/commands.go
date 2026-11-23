package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// version checks that the first line of `markitect version` names a version.
func (s *smoke) version() error {
	out, err := s.run(s.work, nil, s.tool("markitect"), "version")
	if err != nil {
		return err
	}
	first, _, _ := strings.Cut(out, "\n")
	if fields := strings.Fields(first); len(fields) != 2 || fields[0] != "markitect" {
		return fmt.Errorf("unexpected version output: %q", out)
	}
	return nil
}

// smokeFiles fill the documents of a small lending library: the domain README
// that apply copies, and a concept and an operation that new creates.
var smokeFiles = map[string]string{
	"docs/spec/domain/README.md": "# Domain\n\nType: [Domain](../packages/software/domain.md)\n\n## Concepts\n- [Member](member.md)\n\n## Operations\n- [Borrow](borrow.md)\n",
	"docs/spec/domain/member.md": "# Member\n\nType: [Concept](../packages/core/concept.md)\n\n## Definition\nA person with a library card.\n",
	"docs/spec/domain/borrow.md": "# Borrow\n\nType: [Operation](../packages/software/operation.md)\n\n## Does\nLends a book to a member for three weeks.\n\n## Who\nA [member](member.md).\n\n## Given\nThe book and the card.\n",
}

// skillsOf lists the skills init copied into a folder of the project.
func skillsOf(repo, folder string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(folder)))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// sameTree says whether two folders hold the same files with the same bytes.
func sameTree(a, b string) error {
	files := func(root string) (map[string]string, error) {
		out := map[string]string{}
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, err := os.ReadFile(p)
			rel, _ := filepath.Rel(root, p)
			out[filepath.ToSlash(rel)] = string(data)
			return err
		})
		return out, err
	}
	x, err := files(a)
	if err != nil {
		return err
	}
	y, err := files(b)
	if err != nil {
		return err
	}
	if len(x) == 0 || len(x) != len(y) {
		return fmt.Errorf("%s holds %d files, %s %d", a, len(x), b, len(y))
	}
	for name, text := range x {
		if y[name] != text {
			return fmt.Errorf("%s differs between %s and %s", name, a, b)
		}
	}
	return nil
}

// codes returns the code of every diagnostic line of a check, and the file
// each stands in.
func codes(out string) (found, files []string) {
	for _, line := range strings.Split(out, "\n") {
		if parts := strings.SplitN(line, ": ", 3); len(parts) == 3 && strings.Contains(parts[0], ":") {
			found = append(found, parts[1])
			files = append(files, parts[0])
		}
	}
	return found, files
}

// documents is the number of documents of the spec that check counts.
func documents(markitect func(int, ...string) (string, error), want int) (int, error) {
	out, err := markitect(want, "check", "--json")
	if err != nil {
		return 0, err
	}
	var checked struct{ Documents int }
	if err := decode(out, &checked); err != nil {
		return 0, err
	}
	return checked.Documents, nil
}

// projectCommands drives the built CLI over a small fresh project in a plain
// folder, with no Git: init, which lays out the project; table, which lists
// what init laid out; apply, new and check; the look commands show, table,
// topic, questions and licenses; compare against a copy made before a change;
// and the refusal of the commands that are gone. It starts no agent and
// checks that no command writes a state or a history.
func (s *smoke) projectCommands() error {
	repo := filepath.Join(s.work, "lending")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		return err
	}
	markitect := func(want int, args ...string) (string, error) { return s.exit(repo, want, args...) }

	out, err := markitect(0, "init")
	if err != nil {
		return err
	}
	if err := requireContains(out, "created .markitect/\n", "created docs/spec/\n", "created docs/spec/project/\n", "created docs/spec/product/\n", "created docs/spec/architecture/\n",
		"created docs/spec/factory/\n", "created docs/spec/packages/factory/\n", "created AGENTS.md\n"); err != nil {
		return err
	}
	if err := requirePresent(repo, ".markitect/version", ".markitect/spec", ".markitect/ignore", ".markitect/.gitattributes",
		".markitect/context/.gitkeep", ".markitect/findings/pending/.gitkeep", ".markitect/findings/ready/.gitkeep", ".markitect/findings/closed/.gitkeep",
		".markitect/plan/.gitkeep", ".markitect/briefing/.gitkeep", "docs/spec/factory/README.md", "docs/spec/packages/factory/story.md", "AGENTS.md"); err != nil {
		return err
	}
	if entries, err := os.ReadDir(filepath.Join(repo, ".markitect")); err != nil {
		return err
	} else {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		if got := strings.Join(names, " "); got != ".gitattributes briefing context findings ignore plan spec version" {
			return fmt.Errorf(".markitect/ holds %s", got)
		}
	}
	if err := requireContains(readFile(repo, "AGENTS.md"), "skill `markitect-assistant`", "`docs/spec/factory/`"); err != nil {
		return err
	}
	claude, err := skillsOf(repo, ".claude/skills")
	if err != nil || len(claude) == 0 {
		return fmt.Errorf("init copied no skills to .claude/skills: %v", err)
	}
	agents, err := skillsOf(repo, ".agents/skills")
	if err != nil || strings.Join(claude, " ") != strings.Join(agents, " ") {
		return fmt.Errorf("the skills of .claude/skills %v and of .agents/skills %v differ: %v", claude, agents, err)
	}
	for _, name := range claude {
		if err := requirePresent(repo, ".claude/skills/"+name+"/SKILL.md"); err != nil {
			return err
		}
		if err := sameTree(filepath.Join(repo, ".claude", "skills", name), filepath.Join(repo, ".agents", "skills", name)); err != nil {
			return err
		}
	}
	if _, err := markitect(2, "init"); err != nil {
		return fmt.Errorf("a second init: %w", err)
	}

	// The spec passes check but for the sections a project fills, none of
	// them in the way of working.
	out, err = markitect(1, "check")
	if err != nil {
		return err
	}
	found, files := codes(out)
	if len(found) == 0 {
		return fmt.Errorf("check finds nothing in the templates:\n%s", tail(out))
	}
	for i, code := range found {
		if code != "section.empty" || strings.HasPrefix(files[i], "factory/") {
			return fmt.Errorf("check after init finds %s in %s", code, files[i])
		}
	}
	// table lists every document init laid out, sorted by path, with its
	// type and title.
	if out, err := markitect(0, "table"); err != nil {
		return err
	} else if err := requireContains(out, "| Document | Type | Title |\n| --- | --- | --- |\n", "| factory/README.md | ", "| project/README.md | "); err != nil {
		return err
	}

	// A small project of its own: the templates go, the packages stay.
	for _, tpl := range []string{"project", "product", "architecture", "factory"} {
		if err := os.RemoveAll(filepath.Join(repo, "docs", "spec", tpl)); err != nil {
			return err
		}
	}
	base, err := documents(markitect, 0)
	if err != nil {
		return err
	}
	if out, err := markitect(0, "apply", "software/domain", "domain"); err != nil {
		return err
	} else if out != "copied domain/README.md\n" {
		return fmt.Errorf("apply printed %q", out)
	}
	// After apply, check finds nothing but the empty sections.
	if out, err := markitect(1, "check"); err != nil {
		return err
	} else if err := requireContains(out, "domain/README.md:5: section.empty: empty section \"Concepts\"\n", "domain/README.md:7: section.empty: empty section \"Operations\"\n", fmt.Sprintf("2 errors in %d documents\n", base+1)); err != nil {
		return err
	}
	if out, err := markitect(0, "new", "packages/core/concept", "domain/member", "Member"); err != nil {
		return err
	} else if err := requireContains(out, "created domain/member.md\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "new", "packages/software/operation", "domain/borrow", "Borrow"); err != nil {
		return err
	} else if err := requireContains(out, "created domain/borrow.md\n"); err != nil {
		return err
	}
	if got := readFile(repo, "docs/spec/domain/borrow.md"); got != "# Borrow\n\nType: [Operation](../packages/software/operation.md)\n\n## Does\n\n## Who\n\n## Given\n\n" {
		return fmt.Errorf("new wrote %q", got)
	}

	for name, content := range smokeFiles {
		if err := writeFile(repo, name, content); err != nil {
			return err
		}
	}
	total := base + 3
	if out, err := markitect(0, "check"); err != nil {
		return err
	} else if out != fmt.Sprintf("0 errors in %d documents\n", total) {
		return fmt.Errorf("check over the filled spec: %s", tail(out))
	}
	if out, err := markitect(0, "check", "--json"); err != nil {
		return err
	} else {
		var checked struct {
			FormatVersion int `json:"format_version"`
			Kind          string
			Documents     int
			Diagnostics   []any
		}
		if err := decode(out, &checked); err != nil {
			return err
		}
		if checked.FormatVersion != 1 || checked.Kind != "check" || checked.Documents != total || len(checked.Diagnostics) != 0 || strings.Contains(out, `"work"`) {
			return fmt.Errorf("check --json: %s", tail(out))
		}
	}

	// The look commands: the topic view of the whole spec, show, table,
	// questions and licenses.
	if out, err := markitect(0, "topic"); err != nil {
		return err
	} else if err := requireContains(out, "# Signals of the spec\n", "\n### weak ("); err != nil {
		return err
	} else if strings.Contains(out, "hange signals") {
		return fmt.Errorf("topic prints change signals:\n%s", tail(out))
	}
	if out, err := markitect(0, "show", "domain/borrow"); err != nil {
		return err
	} else if err := requireContains(out, "domain/borrow.md\nTitle: Borrow\n", "Type: packages/software/operation.md\n", "Linked from:\n  domain/README.md (Operations)\n"); err != nil {
		return err
	} else if strings.Contains(out, "Verdict") || strings.Contains(out, "@") {
		return fmt.Errorf("show prints a state:\n%s", tail(out))
	}
	if out, err := markitect(0, "table"); err != nil {
		return err
	} else if err := requireContains(out, "| Document | Type | Title |\n", "| domain/borrow.md | packages/software/operation.md | Borrow |\n", "| domain/member.md | packages/core/concept.md | Member |\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "table", "packages/software/operation"); err != nil {
		return err
	} else if err := requireContains(out, "| Title | Does | Who | Given | Refused when | Returns |\n", "| Borrow | Lends a book to a member for three weeks. | A member. |"); err != nil {
		return err
	}
	if out, err := markitect(0, "questions"); err != nil {
		return err
	} else if err := requireContains(out, "## core\n\n1. ", "## software\n\n1. "); err != nil {
		return err
	}
	if out, err := markitect(0, "licenses"); err != nil {
		return err
	} else if err := requireContains(out, "goldmark"); err != nil {
		return err
	}

	// A change of the spec, seen against a copy made before it: a document
	// gone, one new, and a loan that runs two weeks, not three.
	if err := writeFile(repo, "docs/spec/domain/draft.md", "# Draft\n"); err != nil {
		return err
	}
	before := filepath.Join(s.work, "lending-before")
	if err := copyTree(repo, before); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(repo, "docs", "spec", "domain", "draft.md")); err != nil {
		return err
	}
	if _, err := markitect(0, "new", "packages/core/concept", "domain/fine", "Fine"); err != nil {
		return err
	}
	if err := writeFile(repo, "docs/spec/domain/borrow.md", strings.Replace(readFile(repo, "docs/spec/domain/borrow.md"), "three weeks", "two weeks", 1)); err != nil {
		return err
	}
	if out, err := markitect(1, "compare", before); err != nil {
		return err
	} else if err := requireContains(out, "## New (1)\n\ndomain/fine.md\n", "## Gone (1)\n\ndomain/draft.md\n", "## Changed (1)\n\ndomain/borrow.md\n",
		"--- docs/spec/domain/borrow.md\n+++ docs/spec/domain/borrow.md\n", "+Lends a book to a member for two weeks.\n"); err != nil {
		return err
	}
	if _, err := markitect(0, "compare", repo, repo); err != nil {
		return fmt.Errorf("a project against itself: %w", err)
	}

	for _, gone := range []string{"status", "confirm", "move", "vet", "list", "doc", "tidy", "run", "inbox", "update", "install", "config", "hook"} {
		if _, err := markitect(2, gone); err != nil {
			return fmt.Errorf("the removed %s command: %w", gone, err)
		}
	}
	for _, name := range []string{".markitect/state.jsonl", ".markitect/history.jsonl"} {
		if _, err := os.Lstat(filepath.Join(repo, filepath.FromSlash(name))); err == nil {
			return fmt.Errorf("a command wrote %s", name)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err == nil {
		return fmt.Errorf("a command made the project folder a Git repository")
	}
	return nil
}

// exit runs the built markitect in dir and checks its exit code.
func (s *smoke) exit(dir string, want int, args ...string) (string, error) {
	out, err := s.run(dir, nil, s.tool("markitect"), args...)
	code := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		return out, err
	}
	if code != want {
		if err == nil {
			err = fmt.Errorf("markitect %s exited 0\nstdout: %s", strings.Join(args, " "), tail(out))
		}
		return out, fmt.Errorf("want exit %d: %w", want, err)
	}
	return out, nil
}

// readFile reads a file of the project, or "" when it cannot.
func readFile(repo, file string) string {
	data, _ := os.ReadFile(filepath.Join(repo, filepath.FromSlash(file)))
	return string(data)
}

// writeFile writes a file of the project, with its folders.
func writeFile(repo, file, content string) error {
	path := filepath.Join(repo, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// copyTree copies the folder from to the new folder to, as compare reads an
// earlier version of a project.
func copyTree(from, to string) error {
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
