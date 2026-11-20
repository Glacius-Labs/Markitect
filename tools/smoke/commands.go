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
// that realizes the operation, its counter-check, the project's README and
// the command that runs the tests.
var smokeFiles = map[string]string{
	"docs/spec/domain/README.md":          "# Domain\n\nType: [Domain](../packages/software/domain.md)\n\n## Concepts\n- [Member](member.md)\n\n## Operations\n- [Borrow](borrow.md)\n",
	"docs/spec/domain/member.md":          "# Member\n\nType: [Concept](../packages/core/concept.md)\n\n## Definition\nA person with a library card.\n",
	"docs/spec/domain/borrow.md":          "# Borrow\n\nType: [Operation](../packages/software/operation.md)\n\n## Does\nLends a book to a member for three weeks.\n\n## Who\nA [member](member.md).\n\n## Given\nThe book and the card.\n",
	"src/borrow.go":                       "package lending\n\nfunc Borrow() {}\n",
	"tests/counter-checks/borrow_test.go": "package lending\n\nfunc TestBorrow() {}\n",
	"README.md":                           "# Lending library\n",
}

// smokeReport is an open report on the operation, escalated to the owner.
const smokeReport = "# The fee for a late book\n\n## Source\njob read reader, 2026-11-18\n\n## Place\ndocs/spec/domain/borrow.md\n\n## Finding\nThe spec says how long a loan runs, and not what a late book costs.\n\n## Occurred or risk\nrisk\n\n## Violated property\ncomplete\n\n## Actions\n- escalated\n- Charge a fee per day late.\n"

// projectCommands drives the built CLI over a small fresh project in a plain
// folder, with no Git: init, apply, new and check; settings; status and a mark
// of every document it lists; move, which leaves status as it was; a change
// that status lists and compare names against a copy made before; a report
// that reports lists and status waits on; figures; and the refusal of the
// commands that are gone. It starts no agent.
func (s *smoke) projectCommands() error {
	repo := filepath.Join(s.work, "lending")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		return err
	}
	markitect := func(want int, args ...string) (string, error) { return s.exit(repo, want, args...) }

	if out, err := markitect(0, "init"); err != nil {
		return err
	} else if err := requireContains(out, "created .markitect/\n", "created docs/spec/\n"); err != nil {
		return err
	}
	if err := requirePresent(repo, ".markitect/version", ".markitect/settings.md"); err != nil {
		return err
	}
	if out, err := markitect(0, "apply", "software/domain", "domain"); err != nil {
		return err
	} else if err := requireContains(out, "copied domain/README.md\n", "copied packages/software/\n", "copied packages/core/\n"); err != nil {
		return err
	}
	// After apply, check finds nothing but the empty sections.
	if out, err := markitect(1, "check"); err != nil {
		return err
	} else if err := requireContains(out, "domain/README.md:5: section.empty: empty section \"Concepts\"\n", "domain/README.md:7: section.empty: empty section \"Operations\"\n", "2 errors in 20 documents\n"); err != nil {
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
	if err := writeFile(repo, ".markitect/settings.md", readFile(repo, ".markitect/settings.md")+"\n## Tests\n\ngo test ./...\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "check"); err != nil {
		return err
	} else if out != "0 errors in 22 documents\n" {
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
		if checked.FormatVersion != 1 || checked.Kind != "check" || checked.Documents != 22 || len(checked.Diagnostics) != 0 {
			return fmt.Errorf("check --json: %s", tail(out))
		}
	}
	if out, err := markitect(0, "settings"); err != nil {
		return err
	} else if err := requireContains(out, "  builder: sonnet\n", "Counter-checks: tests/counter-checks\n", "Tests: go test ./...\n"); err != nil {
		return err
	}

	// Nothing is confirmed yet: every document is changed, and the files
	// outside the spec are unexplained until a mark names them.
	if out, err := markitect(1, "status"); err != nil {
		return err
	} else if err := requireContains(out, "unexplained: src/borrow.go\n", "22 changed, 0 suspect, 3 unexplained\n"); err != nil {
		return err
	}
	if err := markAll(markitect, 22); err != nil {
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
	if out, err := markitect(1, "compare", before); err != nil {
		return err
	} else if err := requireContains(out, "## Moved (1)\n\ndomain/member.md -> domain/card-holder.md (links rewritten only)\n", "## Changed (1)\n\ndomain/borrow.md\n", "+Lends a book to a member for two weeks.\n", "## Suspect (2)\n"); err != nil {
		return err
	}

	if err := writeFile(repo, ".markitect/reports/late-fee.md", smokeReport); err != nil {
		return err
	}
	if out, err := markitect(0, "check"); err != nil {
		return err
	} else if out != "0 errors in 22 documents, 1 report and 0 samples\n" {
		return fmt.Errorf("check with a report: %s", tail(out))
	}
	if out, err := markitect(0, "reports", "--open"); err != nil {
		return err
	} else if err := requireContains(out, "late-fee: open, escalated\n", "  place: docs/spec/domain/borrow.md\n", "  proposal: Charge a fee per day late.\n", "1 report, 1 open, 1 of them escalated\n"); err != nil {
		return err
	}
	if out, err := markitect(1, "status"); err != nil {
		return err
	} else if err := requireContains(out, "waiting: domain/borrow.md@", ": report late-fee (escalated: waits on the owner): self\n", "0 changed, 2 suspect, 0 unexplained, 1 waiting, 1 of them on the owner\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "figures"); err != nil {
		return err
	} else if err := requireContains(out, "# Figures from the history\n\n1 change from 22 verdicts; moves left out, since they are mechanical: 1\n", "# Figures of the factory\n"); err != nil {
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

// markAll marks every document status lists ok, as a reviewer would: the
// operation with the code that realizes it and its counter-check, the domain
// README with the project's README as its artifact, and every other document
// with the README as evidence. Then status lists nothing.
func markAll(markitect func(int, ...string) (string, error), want int) error {
	list, err := markitect(1, "status")
	if err != nil {
		return err
	}
	marked := 0
	for _, line := range strings.Split(list, "\n") {
		entry, ok := strings.CutPrefix(line, "changed: ")
		if !ok {
			continue
		}
		doc, _, _ := strings.Cut(entry, ": ")
		args := []string{"mark", doc, "ok", "-e", "README.md:1"}
		switch {
		case strings.HasPrefix(doc, "domain/borrow.md@"):
			args = []string{"mark", doc, "ok", "--file", "src/borrow.go", "--check", "tests/counter-checks/borrow_test.go", "-e", "tests/counter-checks/borrow_test.go:3 (TestBorrow)"}
		case strings.HasPrefix(doc, "domain/README.md@"):
			args = append(args, "--file", "README.md")
		}
		out, err := markitect(0, args...)
		if err != nil {
			return err
		}
		name, _, _ := strings.Cut(doc, "@")
		if out != "marked "+name+" ok\n" {
			return fmt.Errorf("mark printed %q", out)
		}
		marked++
	}
	if marked != want {
		return fmt.Errorf("marked %d documents, want %d:\n%s", marked, want, tail(list))
	}
	if out, err := markitect(0, "status"); err != nil {
		return fmt.Errorf("status after every mark: %w", err)
	} else if out != "0 changed, 0 suspect, 0 unexplained\n" {
		return fmt.Errorf("status after every mark: %s", tail(out))
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
