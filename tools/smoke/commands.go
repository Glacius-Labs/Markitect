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

// smokeFiles are the documents of a small lending library that a person writes
// by hand into the model beside the default: a module domain with its README,
// a concept of core's type, a type of their own with an operation of it, and a
// type that extends another with a document of it.
var smokeFiles = map[string]string{
	"docs/model/domain/README.md":       "# Domain\n\nHow members borrow books.\n\n## Contents\n- [Member](member.md)\n- [Borrow](borrow.md)\n\n## Concepts\n- [Member](member.md)\n\n## Operations\n- [Borrow](borrow.md)\n",
	"docs/model/domain/member.md":       "# Member\n\nType: [Concept](../core/concept.md)\n\n## Definition\nA person with a library card.\n",
	"docs/model/domain/operation.md":    "# Operation\n\nSomething a member can do.\n\n## Does\nWhat happens.\n\n## Who\nWho may do it.\n\n## Given\nWhat it needs.\n",
	"docs/model/domain/borrow.md":       "# Borrow\n\nType: [Operation](operation.md)\n\n## Does\nLends a book to a member for three weeks.\n\n## Who\nA [member](member.md).\n\n## Given\nThe book and the card.\n",
	"docs/model/domain/artifact.md":     "# Artifact\n\nSomething the library keeps.\n\n## Owner\nWho answers for it.\n",
	"docs/model/domain/catalog.md":      "# Catalog\n\nExtends: [Artifact](artifact.md)\n\nA list of books.\n\n## Entries\nThe books it lists.\n",
	"docs/model/domain/main-catalog.md": "# Main catalog\n\nType: [Catalog](catalog.md)\n\n## Owner\nThe head librarian.\n\n## Entries\nEvery book.\n",
}

// snapshot reads every file below root, by its path with forward slashes, and
// lists every folder with a trailing slash and no content.
func snapshot(root string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == root {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if d.IsDir() {
			out[filepath.ToSlash(rel)+"/"] = ""
			return nil
		}
		data, err := os.ReadFile(p)
		out[filepath.ToSlash(rel)] = string(data)
		return err
	})
	return out, err
}

// sameFiles says whether two snapshots hold the same files with the same
// bytes, and names the first difference.
func sameFiles(before, after map[string]string) error {
	for name, text := range before {
		got, ok := after[name]
		switch {
		case !ok:
			return fmt.Errorf("%s is gone", name)
		case got != text:
			return fmt.Errorf("%s changed", name)
		}
	}
	for name := range after {
		if _, ok := before[name]; !ok {
			return fmt.Errorf("%s is new", name)
		}
	}
	return nil
}

// documents is the number of documents of the model that check counts.
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

// layOut builds a project by hand, the way a person does since no command lays
// one out: an empty folder, the default copied from plugin/default/ of the
// repository into a model folder, docs/model, and the file .markitect/model that
// names that folder.
func layOut(repo, project string) error {
	if err := copyTree(defaultFolder(repo), filepath.Join(project, "docs", "model")); err != nil {
		return err
	}
	return writeFile(project, ".markitect/model", "docs/model\n")
}

// defaultFolder is the default that ships with Markitect: the model folder that a
// project copies.
func defaultFolder(repo string) string { return filepath.Join(repo, "plugin", "default") }

// countDocuments counts the Markdown files below dir, which are the documents
// that check counts once the folder is a model.
func countDocuments(dir string) (int, error) {
	n := 0
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") {
			n++
		}
		return err
	})
	return n, err
}

// projectCommands drives the built CLI over a small project that it lays out
// by hand in a plain folder: the commands that read it, check,
// table, show, topic, questions, licenses and diff against a copy made before a
// change, and the refusal of the commands that are gone. It starts no agent
// and checks that no command writes anything into the project.
func (s *smoke) projectCommands() error {
	project := filepath.Join(s.work, "lending")
	markitect := func(want int, args ...string) (string, error) { return s.exit(project, want, args...) }

	// Outside a project, the commands that need one name the file that is
	// missing and what it holds; version and licenses need none.
	if err := os.MkdirAll(project, 0o755); err != nil {
		return err
	}
	if err := s.refused(project, ".markitect/model", "check"); err != nil {
		return fmt.Errorf("check in an empty folder: %w", err)
	}
	if out, err := markitect(0, "licenses"); err != nil || !strings.Contains(out, "goldmark") {
		return fmt.Errorf("licenses in an empty folder: %v", err)
	}

	if err := layOut(s.repo, project); err != nil {
		return err
	}
	if err := requirePresent(project, ".markitect/model", "docs/model/core/concept.md", "docs/model/core/term.md"); err != nil {
		return err
	}
	if entries, err := os.ReadDir(filepath.Join(project, ".markitect")); err != nil {
		return err
	} else if len(entries) != 1 {
		return fmt.Errorf(".markitect/ holds %d entries, not only model", len(entries))
	}

	// The default alone passes check, and table lists its documents, sorted by
	// path, with their type and title.
	inDefault, err := countDocuments(defaultFolder(s.repo))
	if err != nil {
		return err
	}
	if out, err := markitect(0, "check"); err != nil {
		return err
	} else if out != fmt.Sprintf("0 errors in %d documents\n", inDefault) {
		return fmt.Errorf("check over the default: %s", tail(out))
	}
	if out, err := markitect(0, "table"); err != nil {
		return err
	} else if err := requireContains(out, "| Document | Type | Title |\n| --- | --- | --- |\n", "| core/README.md |  | Core |\n", "| core/concept.md |  | Concept |\n", "| core/term.md |  | Term |\n"); err != nil {
		return err
	}
	base, err := documents(markitect, 0)
	if err != nil {
		return err
	}

	for name, content := range smokeFiles {
		if err := writeFile(project, name, content); err != nil {
			return err
		}
	}
	total := base + len(smokeFiles)
	if out, err := markitect(0, "check"); err != nil {
		return err
	} else if out != fmt.Sprintf("0 errors in %d documents\n", total) {
		return fmt.Errorf("check over the filled model: %s", tail(out))
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
		if checked.FormatVersion != 1 || checked.Kind != "check" || checked.Documents != total || len(checked.Diagnostics) != 0 {
			return fmt.Errorf("check --json: %s", tail(out))
		}
	}
	// A document that breaks a rule is found, with its file and line.
	if err := writeFile(project, "docs/model/domain/broken.md", "# Broken\n\nSee [nothing](nothing.md).\n"); err != nil {
		return err
	}
	if out, err := markitect(1, "check"); err != nil {
		return err
	} else if err := requireContains(out, "domain/broken.md:3: link.unresolved: no document at \"nothing.md\"\n", fmt.Sprintf("1 error in %d documents\n", total+1)); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(project, "docs", "model", "domain", "broken.md")); err != nil {
		return err
	}
	// A module needs a README, and what its README does not list under Contents
	// is internal to it: a link from outside is found.
	if err := writeFile(project, "docs/model/shop/cart.md", "# Cart\n\nWhat a customer buys.\n"); err != nil {
		return err
	}
	if err := writeFile(project, "docs/model/domain/shopping.md", "# Shopping\n\nSee [the cart](../shop/cart.md).\n"); err != nil {
		return err
	}
	if out, err := markitect(1, "check"); err != nil {
		return err
	} else if err := requireContains(out, "domain/shopping.md:3: link.internal: \"shop/cart.md\" is internal to shop: shop has no README.md\n",
		"shop/README.md:1: module.readme: the folder shop holds documents and has no README.md\n", fmt.Sprintf("2 errors in %d documents\n", total+2)); err != nil {
		return err
	}
	if err := writeFile(project, "docs/model/shop/README.md", "# Shop\n\nWhat customers buy.\n"); err != nil {
		return err
	}
	if out, err := markitect(1, "check", "--json"); err != nil {
		return err
	} else if err := requireContains(out, `"code": "link.internal"`, `"message": "\"shop/cart.md\" is internal to shop: shop/README.md does not list it under Contents"`); err != nil {
		return err
	}
	if err := writeFile(project, "docs/model/shop/README.md", "# Shop\n\nWhat customers buy.\n\n## Contents\n- [Cart](cart.md)\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "check"); err != nil {
		return err
	} else if out != fmt.Sprintf("0 errors in %d documents\n", total+3) {
		return fmt.Errorf("check with a module that lists its cart: %s", tail(out))
	}
	// A pattern in Contents shows the documents of the folder that match it.
	if err := writeFile(project, "docs/model/shop/README.md", "# Shop\n\n## Contents\n- `ca*.md`\n"); err != nil {
		return err
	} else if out, err := markitect(0, "check"); err != nil || out != fmt.Sprintf("0 errors in %d documents\n", total+3) {
		return fmt.Errorf("check with a module that shows its cart by a pattern: %v %s", err, tail(out))
	}
	if err := os.RemoveAll(filepath.Join(project, "docs", "model", "shop")); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(project, "docs", "model", "domain", "shopping.md")); err != nil {
		return err
	}
	// A derived type that makes a section of its base optional is found.
	if err := writeFile(project, "docs/model/domain/loose.md", "# Loose\n\nExtends: [Artifact](artifact.md)\n\n## Owner (optional)\nNobody.\n"); err != nil {
		return err
	}
	if out, err := markitect(1, "check"); err != nil {
		return err
	} else if err := requireContains(out, `domain/loose.md:5: extends.loosens: section "Owner" is required by domain/artifact.md`+"\n", fmt.Sprintf("1 error in %d documents\n", total+1)); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(project, "docs", "model", "domain", "loose.md")); err != nil {
		return err
	}
	if err := s.propertyTypes(project, total); err != nil {
		return fmt.Errorf("property types: %w", err)
	}

	// Everything from here on reads: nothing of the project may change.
	before, err := snapshot(project)
	if err != nil {
		return err
	}

	// The look commands: the topic view of the whole model, show, table,
	// questions and licenses.
	if out, err := markitect(0, "topic"); err != nil {
		return err
	} else if err := requireContains(out, "# Signals of the model\n", "\n### weak ("); err != nil {
		return err
	}
	if out, err := markitect(0, "show", "domain/borrow"); err != nil {
		return err
	} else if err := requireContains(out, "domain/borrow.md\nTitle: Borrow\n", "Type: domain/operation.md\n", "Linked from:\n  domain/README.md (Contents)\n  domain/README.md (Operations)\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "table"); err != nil {
		return err
	} else if err := requireContains(out, "| Document | Type | Title |\n", "| domain/borrow.md | domain/operation.md | Borrow |\n", "| domain/member.md | core/concept.md | Member |\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "table", "domain/operation"); err != nil {
		return err
	} else if err := requireContains(out, "| Title | Does | Who | Given |\n", "| Borrow | Lends a book to a member for three weeks. | A member. |"); err != nil {
		return err
	}
	// A type that extends another: show names the base and where each section
	// comes from, and table and topic count the documents of the derived type
	// among those of the base.
	if out, err := markitect(0, "show", "domain/catalog"); err != nil {
		return err
	} else if err := requireContains(out, "domain/catalog.md\nTitle: Catalog\nExtends: domain/artifact.md\nSections:\n  Owner (from domain/artifact.md)\n  Entries\n", "Documents:\n  domain/main-catalog.md\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "table", "domain/artifact"); err != nil {
		return err
	} else if out != "| Title | Owner |\n| --- | --- |\n| Main catalog | The head librarian. |\n" {
		return fmt.Errorf("table of the base type: %s", tail(out))
	}
	if out, err := markitect(0, "topic", "domain/artifact", "--json"); err != nil {
		return err
	} else if err := requireContains(out, `"topic_kind": "type"`, `"domain/main-catalog.md"`); err != nil {
		return err
	}
	if out, err := markitect(0, "questions"); err != nil {
		return err
	} else if err := requireContains(out, "## core\n\n1. Which role is responsible for each rule"); err != nil {
		return err
	}
	if out, err := markitect(0, "licenses"); err != nil {
		return err
	} else if err := requireContains(out, "goldmark"); err != nil {
		return err
	}
	if out, err := markitect(0, "version", "--json"); err != nil {
		return err
	} else {
		var v struct {
			FormatVersion int `json:"format_version"`
			Kind          string
			Version       string
			Build         string
		}
		if err := decode(out, &v); err != nil {
			return err
		}
		if v.FormatVersion != 1 || v.Kind != "version" || v.Version == "" || !strings.HasPrefix(v.Build, v.Version) || strings.Contains(out, "pinned") {
			return fmt.Errorf("version --json: %s", tail(out))
		}
	}

	// The commands find the project from a folder inside it.
	inside := filepath.Join(project, "docs", "model", "domain")
	if out, err := s.exit(inside, 0, "check"); err != nil {
		return err
	} else if out != fmt.Sprintf("0 errors in %d documents\n", total) {
		return fmt.Errorf("check from inside the model: %s", tail(out))
	}

	read, err := snapshot(project)
	if err != nil {
		return err
	}
	if err := sameFiles(before, read); err != nil {
		return fmt.Errorf("a command wrote into the project: %w", err)
	}

	// A change of the model, seen against a copy made before it: a document
	// gone, one new, and a loan that runs two weeks, not three. Writing the
	// change is the person's, so it is done here by hand, after the check that
	// no command wrote.
	if err := writeFile(project, "docs/model/domain/draft.md", "# Draft\n"); err != nil {
		return err
	}
	copyOf := filepath.Join(s.work, "lending-before")
	if err := copyTree(project, copyOf); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(project, "docs", "model", "domain", "draft.md")); err != nil {
		return err
	}
	if err := writeFile(project, "docs/model/domain/fine.md", "# Fine\n\nType: [Concept](../core/concept.md)\n\n## Definition\nWhat a late book costs.\n"); err != nil {
		return err
	}
	if err := writeFile(project, "docs/model/domain/borrow.md", strings.Replace(readFile(project, "docs/model/domain/borrow.md"), "three weeks", "two weeks", 1)); err != nil {
		return err
	}
	changed, err := snapshot(project)
	if err != nil {
		return err
	}
	if out, err := markitect(1, "diff", copyOf); err != nil {
		return err
	} else if err := requireContains(out, "## New (1)\n\ndomain/fine.md\n", "## Gone (1)\n\ndomain/draft.md\n", "## Changed (1)\n\ndomain/borrow.md\n",
		"--- docs/model/domain/borrow.md\n+++ docs/model/domain/borrow.md\n", "+Lends a book to a member for two weeks.\n"); err != nil {
		return err
	}
	if out, err := markitect(1, "diff", copyOf, "--json"); err != nil {
		return err
	} else if err := requireContains(out, `"kind": "diff"`); err != nil {
		return err
	}
	if _, err := markitect(0, "diff", project, project); err != nil {
		return fmt.Errorf("a project against itself: %w", err)
	}

	// The commands that are gone are unknown commands.
	for _, gone := range []string{"init", "apply", "new", "status", "confirm", "move", "compare", "vet", "list", "doc", "tidy", "run", "inbox", "update", "install", "config", "hook"} {
		if err := s.refused(project, fmt.Sprintf("unknown command %q", gone), gone); err != nil {
			return fmt.Errorf("the removed %s command: %w", gone, err)
		}
	}

	// No command wrote anything into the project: not a file, a folder, a
	// state or a history.
	after, err := snapshot(project)
	if err != nil {
		return err
	}
	if err := sameFiles(changed, after); err != nil {
		return fmt.Errorf("a command wrote into the project: %w", err)
	}
	return s.diffFiles(project)
}

// diffFiles drives diff over the files of a project: a module whose README says
// under Realized in where it is built, in a dot folder too, a file that changed,
// one that is new, one that no module names, files that the ignore file leaves
// out, and takes in again from a folder it left out the contents of, and a bad
// line of the ignore file. It works on copies of the project, so that the check that no
// command wrote into the project stands.
func (s *smoke) diffFiles(project string) error {
	older := filepath.Join(s.work, "lending-files-before")
	newer := filepath.Join(s.work, "lending-files-after")
	if err := copyTree(project, older); err != nil {
		return err
	}
	readme := readFile(older, "docs/model/domain/README.md") + "\n## Realized in\n- `src/borrow/`\n- `.claude/skills/triage/`\n"
	for name, content := range map[string]string{
		"docs/model/domain/README.md":    readme,
		"src/borrow/borrow.go":           "package borrow\n",
		".claude/skills/triage/SKILL.md": "triage\n",
		".claude/skills/other/SKILL.md":  "nobody names it\n",
		"tools/build.sh":                 "#!/bin/sh\n",
		"out/result.txt":                 "one\n",
		".claude/settings.json":          "{}\n",
		".claude/worktrees/w/big.txt":    "one\n",
		".markitect/ignore":              "# build output, and of .claude only the skills\nout/\n.claude/*\n!.claude/skills/\n",
	} {
		if err := writeFile(older, name, content); err != nil {
			return err
		}
	}
	if err := copyTree(older, newer); err != nil {
		return err
	}
	if _, err := s.exit(newer, 0, "diff", older); err != nil {
		return fmt.Errorf("two copies of a project with files: %w", err)
	}
	for name, content := range map[string]string{
		"src/borrow/borrow.go":           "package borrow // three weeks\n",
		"src/borrow/renew.go":            "package borrow\n",
		".claude/skills/triage/SKILL.md": "triage, changed\n",
		".claude/skills/other/SKILL.md":  "changed, but nobody names it\n",
		"tools/build.sh":                 "#!/bin/sh\nset -e\n",
		"out/result.txt":                 "two\n",
		".claude/settings.json":          "{ \"changed\": true }\n",
		".claude/worktrees/w/big.txt":    "two\n",
	} {
		if err := writeFile(newer, name, content); err != nil {
			return err
		}
	}
	if out, err := s.exit(newer, 1, "diff", older); err != nil {
		return err
	} else if err := requireContains(out, "## Modules (1)\n\ndomain: files changed (3)\n  changed .claude/skills/triage/SKILL.md\n  changed src/borrow/borrow.go\n  new     src/borrow/renew.go\n\n",
		"## Files no module names (2)\n\nchanged .claude/skills/other/SKILL.md\nchanged tools/build.sh\n\nFiles of NEW that no module names: 2\n"); err != nil {
		return err
	}
	if out, err := s.exit(newer, 1, "diff", older, "--json"); err != nil {
		return err
	} else if err := requireContains(out, `"module": "domain"`, `"changed_files": [`, `"change": "changed"`, `"total": 2`); err != nil {
		return err
	}
	// A bad line of the ignore file of NEW stops diff, and names file and line.
	if err := writeFile(newer, ".markitect/ignore", "out/\n../x\n"); err != nil {
		return err
	}
	return s.refused(newer, `.markitect/ignore:2: bad pattern "../x"`, "diff", older)
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

// refused runs the built markitect in dir and checks that it exits with 2 and
// that what it says on its error output contains want.
func (s *smoke) refused(dir, want string, args ...string) error {
	_, err := s.run(dir, nil, s.tool("markitect"), args...)
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 {
		return fmt.Errorf("markitect %s: want exit 2: %v", strings.Join(args, " "), err)
	}
	return requireContains(err.Error(), want)
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

// copyTree copies the folder from to the new folder to, as diff reads an
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
