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
// that apply copies, a concept and an operation that new creates, the code
// that realizes the operation, its counter-check and the project's README.
var smokeFiles = map[string]string{
	"docs/spec/domain/README.md":          "# Domain\n\nType: [Domain](../packages/software/domain.md)\n\n## Concepts\n- [Member](member.md)\n\n## Operations\n- [Borrow](borrow.md)\n",
	"docs/spec/domain/member.md":          "# Member\n\nType: [Concept](../packages/core/concept.md)\n\n## Definition\nA person with a library card.\n",
	"docs/spec/domain/borrow.md":          "# Borrow\n\nType: [Operation](../packages/software/operation.md)\n\n## Does\nLends a book to a member for three weeks.\n\n## Who\nA [member](member.md).\n\n## Given\nThe book and the card.\n",
	"src/borrow.go":                       "package lending\n\nfunc Borrow() {}\n",
	"tests/counter-checks/borrow_test.go": "package lending\n\nfunc TestBorrow() {}\n",
	"README.md":                           "# Lending library\n",
}

// smokeFinding is a finding of the work on the operation: its type line names
// the type of the factory package in the spec's own copy, and its links lead
// to a document of the spec and to the code.
const smokeFinding = "# The fee for a late book\n\n" +
	"Type: [Finding](../../../docs/spec/packages/factory/finding.md)\n\n" +
	"## Type\nQuestion\n\n## Importance\nnormal\n\n## Area\nThe lending of books.\n\n" +
	"## What\nThe spec says how long a loan runs in [Borrow](../../../docs/spec/domain/borrow.md), and not what a late book costs.\n\n" +
	"## Source\nA reader of [the code](../../../src/borrow.go).\n\n## Context\nSaid while reading the loan rules.\n\n" +
	"## Why this importance\nNobody pays a fee today.\n"

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
// folder, with no Git: init, which lays out the project; apply, new and
// check; status and a confirm of every document it lists; move, which leaves
// status as it was; a change that status lists and compare names against a
// copy made before; the documents of the work, which check checks and neither
// status nor the spec know; and the refusal of the commands that are gone. It
// starts no agent.
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
	// them in the way of working, and status gives no list until it does.
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
	// status lists and confirm binds although check reports the empty
	// sections: each command does one thing. This runs in a project of its
	// own, since the project below drops the templates.
	fresh := filepath.Join(s.work, "after-init")
	if err := os.MkdirAll(fresh, 0o755); err != nil {
		return err
	}
	if _, err := s.exit(fresh, 0, "init"); err != nil {
		return err
	}
	list, err := s.exit(fresh, 1, "status")
	if err != nil {
		return err
	}
	if err := requireContains(list, "changed: project/README.md@", "changed: factory/README.md@", "unexplained: AGENTS.md\n"); err != nil {
		return err
	}
	entry, _, _ := strings.Cut(list[strings.Index(list, "changed: project/README.md@")+len("changed: "):], ": ")
	if out, err := s.exit(fresh, 0, "confirm", entry, "skip", "-m", "its sections are filled later"); err != nil {
		return err
	} else if err := requireContains(out, "confirmed project/README.md skip\n", "files that realize it, all of them: (none)\n"); err != nil {
		return err
	}
	if out, err := s.exit(fresh, 0, "status", "project/README"); err != nil {
		return err
	} else if err := requireContains(out, "note: project/README.md: skip: its sections are filled later\n", "0 changed, 0 suspect, 0 unexplained\n"); err != nil {
		return err
	}

	// A small project of its own: the templates go, the packages stay. The
	// skills and AGENTS.md are files of the project that realize nothing of
	// its spec, which its ignore file says.
	for _, tpl := range []string{"project", "product", "architecture", "factory"} {
		if err := os.RemoveAll(filepath.Join(repo, "docs", "spec", tpl)); err != nil {
			return err
		}
	}
	if err := writeFile(repo, ".markitect/ignore", "AGENTS.md\n.claude/\n.agents/\n"); err != nil {
		return err
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
			Work          int
			Diagnostics   []any
		}
		if err := decode(out, &checked); err != nil {
			return err
		}
		if checked.FormatVersion != 1 || checked.Kind != "check" || checked.Documents != total || checked.Work != 0 || len(checked.Diagnostics) != 0 {
			return fmt.Errorf("check --json: %s", tail(out))
		}
	}

	// Nothing is confirmed yet: every document is changed, and the files
	// outside the spec are unexplained until a confirm names them.
	if out, err := markitect(1, "status"); err != nil {
		return err
	} else if err := requireContains(out, "unexplained: src/borrow.go\n", fmt.Sprintf("%d changed, 0 suspect, 3 unexplained\n", total)); err != nil {
		return err
	}
	if err := confirmAll(markitect, total); err != nil {
		return err
	}
	if err := requirePresent(repo, ".markitect/state.jsonl", ".markitect/history.jsonl"); err != nil {
		return err
	}

	before := filepath.Join(s.work, "lending-before")
	if err := copyTree(repo, before); err != nil {
		return err
	}
	if out, err := markitect(0, "move", "domain/member", "domain/card-holder"); err != nil {
		return err
	} else if out != "moved domain/member.md -> domain/card-holder.md\nchanged domain/README.md\nchanged domain/borrow.md\n" {
		return fmt.Errorf("move printed %q", out)
	}
	if out, err := markitect(0, "status"); err != nil {
		return fmt.Errorf("status after the move: %w", err)
	} else if out != "0 changed, 0 suspect, 0 unexplained\n" {
		return fmt.Errorf("status after the move: %s", tail(out))
	}

	// The look commands: the topic view of the whole spec ends with the change
	// signals of the history, one change of every document's first confirm and
	// the move left out; show, table, questions and licenses.
	if out, err := markitect(0, "topic"); err != nil {
		return err
	} else if err := requireContains(out, "# Signals of the spec\n", fmt.Sprintf("\n# Change signals of the history\n\n1 change from %s; moves left out, since they are mechanical: 1\n", plural(total, "verdict")),
		"\n### wide-change (1)\n", "\n### weakening\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "show", "domain/borrow"); err != nil {
		return err
	} else if err := requireContains(out, "domain/borrow.md@", "Type: packages/software/operation.md\n", "Verdict: ok\n", "  src/borrow.go\n", "Linked from:\n  domain/README.md (Operations)\n"); err != nil {
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

	// A change of the spec: a loan runs two weeks, not three. The operation
	// is changed, and the documents that link to it are suspect.
	if err := writeFile(repo, "docs/spec/domain/borrow.md", strings.Replace(readFile(repo, "docs/spec/domain/borrow.md"), "three weeks", "two weeks", 1)); err != nil {
		return err
	}
	if out, err := markitect(1, "status"); err != nil {
		return err
	} else if err := requireContains(out, "changed: domain/borrow.md@", ": self\n", "suspect: domain/card-holder.md@", ": link:domain/borrow.md\n", "1 changed, 2 suspect, 0 unexplained\n"); err != nil {
		return err
	}
	// status for a path lists what lies there only, and its exit code follows.
	if out, err := markitect(1, "status", "domain/borrow"); err != nil {
		return err
	} else if err := requireContains(out, "changed: domain/borrow.md@", "1 changed, 0 suspect, 0 unexplained\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "status", "src"); err != nil {
		return err
	} else if out != "0 changed, 0 suspect, 0 unexplained\n" {
		return fmt.Errorf("status of src: %s", tail(out))
	}
	if _, err := markitect(2, "status", "nowhere/at/all"); err != nil {
		return fmt.Errorf("status of an unknown path: %w", err)
	}

	// The work: a finding with a type line into the spec, a dropped note and
	// a checkpoint without a type, and a hidden folder that is passed over.
	// check checks them, compare names them, and status does not know them.
	for name, content := range map[string]string{
		".markitect/findings/pending/late-fee.md": smokeFinding,
		".markitect/context/capture/note.md":      "# A dropped note\n\nNo type, no links.\n",
		".markitect/context/build/checkpoint.md":  "# Checkpoint\n\nContinue with [the finding](../../findings/pending/late-fee.md).\n",
		".markitect/context/capture/.raw/log.md":  "Raw, with [a dead link](nowhere.md).\n",
	} {
		if err := writeFile(repo, name, content); err != nil {
			return err
		}
	}
	if out, err := markitect(0, "check"); err != nil {
		return err
	} else if out != fmt.Sprintf("0 errors in %d documents, 3 documents of the work\n", total) {
		return fmt.Errorf("check with the work: %s", tail(out))
	}
	if out, err := markitect(1, "status"); err != nil {
		return err
	} else if err := requireContains(out, "1 changed, 2 suspect, 0 unexplained\n"); err != nil {
		return err
	}
	if out, err := markitect(1, "compare", before); err != nil {
		return err
	} else if err := requireContains(out, "## Moved (1)\n\ndomain/member.md -> domain/card-holder.md (links rewritten only)\n", "## Changed (1)\n\ndomain/borrow.md\n",
		"+Lends a book to a member for two weeks.\n", "## Suspect (2)\n", "## Work (3)\n\nnew: .markitect/context/build/checkpoint.md\n", "new: .markitect/findings/pending/late-fee.md\n"); err != nil {
		return err
	}

	// Each defect of a document of the work has its diagnostic.
	if err := writeFile(repo, ".markitect/plan/story.md", "# A story\n\nType: [Story](../../docs/spec/packages/factory/story.md)\n\nSee [nothing](../../docs/spec/gone.md).\n"); err != nil {
		return err
	}
	if out, err := markitect(1, "check"); err != nil {
		return err
	} else if err := requireContains(out, ".markitect/plan/story.md:3: section.missing: missing section \"State\" required by packages/factory/story.md\n",
		".markitect/plan/story.md:5: link.unresolved: no file at \"../../docs/spec/gone.md\"\n"); err != nil {
		return err
	}
	// The errors of the work block nothing of status.
	if out, err := markitect(1, "status"); err != nil {
		return err
	} else if err := requireContains(out, "1 changed, 2 suspect, 0 unexplained\n"); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(repo, ".markitect", "plan", "story.md")); err != nil {
		return err
	}
	if err := writeFile(repo, ".markitect/plan/lost.md", "# Lost\n\nType: [Story](../../docs/spec/packages/factory/ghost.md)\n"); err != nil {
		return err
	}
	if out, err := markitect(1, "check"); err != nil {
		return err
	} else if err := requireContains(out, ".markitect/plan/lost.md:3: type.unresolved: no document at \"../../docs/spec/packages/factory/ghost.md\"\n"); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(repo, ".markitect", "plan", "lost.md")); err != nil {
		return err
	}

	for _, gone := range []string{"vet", "list", "doc", "tidy", "run", "inbox", "update", "install", "config", "hook"} {
		if _, err := markitect(2, gone); err != nil {
			return fmt.Errorf("the removed %s command: %w", gone, err)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err == nil {
		return fmt.Errorf("a command made the project folder a Git repository")
	}
	return nil
}

// confirmAll confirms every document status lists ok, as a reviewer would:
// the operation with the code that realizes it and its counter-check, the
// domain README with the project's README as its artifact, and every other
// document with the README as evidence. Then status lists nothing.
func confirmAll(markitect func(int, ...string) (string, error), want int) error {
	list, err := markitect(1, "status")
	if err != nil {
		return err
	}
	confirmed := 0
	for _, line := range strings.Split(list, "\n") {
		entry, ok := strings.CutPrefix(line, "changed: ")
		if !ok {
			continue
		}
		doc, _, _ := strings.Cut(entry, ": ")
		args := []string{"confirm", doc, "ok", "-e", "README.md:1"}
		switch {
		case strings.HasPrefix(doc, "domain/borrow.md@"):
			args = []string{"confirm", doc, "ok", "--file", "src/borrow.go", "--check", "tests/counter-checks/borrow_test.go", "-e", "tests/counter-checks/borrow_test.go:3 (TestBorrow)"}
		case strings.HasPrefix(doc, "domain/README.md@"):
			args = append(args, "--file", "README.md")
		}
		out, err := markitect(0, args...)
		if err != nil {
			return err
		}
		name, _, _ := strings.Cut(doc, "@")
		if !strings.HasPrefix(out, "confirmed "+name+" ok\n") {
			return fmt.Errorf("confirm printed %q", out)
		}
		confirmed++
	}
	if confirmed != want {
		return fmt.Errorf("confirmed %d documents, want %d:\n%s", confirmed, want, tail(list))
	}
	if out, err := markitect(0, "status"); err != nil {
		return fmt.Errorf("status after every confirm: %w", err)
	} else if out != "0 changed, 0 suspect, 0 unexplained\n" {
		return fmt.Errorf("status after every confirm: %s", tail(out))
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
// earlier state of a project.
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

// plural is a number and a noun, in the plural unless the number is one.
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
