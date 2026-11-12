package main

import (
	"fmt"
	"os"
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

// smokeSpec is a small project in the spec language: Root with a Rule, and
// the Packages members and catalog, with registry pages for every file.
var smokeSpec = map[string]string{
	".markitect/spec/loan-limit.md":      "---\nkind: std/Rule\n---\n# Loan limit\n\nA member has at most `5` open loans; see [[members]].\n",
	".markitect/spec/members/members.md": "---\nkind: Package\n---\n# Members\n\nWho may borrow.\n",
	".markitect/spec/members/member.md":  "---\nkind: std/Concept\n---\n# Member\n\nA person who borrows copies.\n",
	".markitect/spec/catalog/catalog.md": "---\nkind: Package\n---\n# Catalog\n\nWhat can be borrowed.\n",
	".markitect/spec/catalog/copy.md":    "---\nkind: std/Concept\n---\n# Copy\n\nA copy is lent to a [[members/member]].\n",
	".markitect/registry/members.md":     "---\npaths: [src/members/**]\n---\n# Registry: members\n\n| File | Realizes | Why |\n|---|---|---|\n| `src/members/member.go` | [[members/members]] [[members/member]] [[loan-limit]] | Member type and loan limit |\n",
	".markitect/registry/catalog.md":     "---\npaths: [src/catalog/**]\n---\n# Registry: catalog\n\n| File | Realizes | Why |\n|---|---|---|\n| `src/catalog/copy.go` | [[catalog/catalog]] [[catalog/copy]] | Copy type |\n",
	"README.md":                          "# Lending library\n",
	"src/members/member.go":              "package members\n",
	"src/catalog/copy.go":                "package catalog\n",
}

// specCommands drives the built CLI over a small project in a plain folder,
// with no Git: init, check, tidy, doc, list and vet; status and mark that
// verify every Definition; status after a change of the spec; and the refusal of
// a spec that does not compile. It starts no agent.
func (s *smoke) specCommands() error {
	repo := filepath.Join(s.work, "spec-commands")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		return err
	}
	markitect := func(args ...string) (string, error) { return s.run(repo, nil, s.tool("markitect"), args...) }
	if out, err := markitect("init", "--name", "Lending library"); err != nil {
		return err
	} else if err := requireContains(out, "created .markitect/manifest.yaml", "wrote .markitect/lock.yaml: std 1.1.0", "installed the harness"); err != nil {
		return err
	}
	if err := requirePresent(repo, ".claude/skills/markitect-architect/SKILL.md", ".claude/agents/markitect-architect.md", ".claude/workflows/markitect-parallel.js", ".markitect/spec/project.md"); err != nil {
		return err
	}
	for name, content := range smokeSpec {
		path := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	root := filepath.Join(repo, ".markitect", "registry.md")
	page, err := os.ReadFile(root)
	if err != nil {
		return err
	}
	if err := os.WriteFile(root, append(page, "| `README.md` | [[project]] | Overview |\n"...), 0o644); err != nil {
		return err
	}
	out, err := markitect("check")
	if err != nil {
		return err
	}
	if err := requireContains(out, "ok: 6 Definitions in 3 Packages, 3 files accounted"); err != nil {
		return err
	}
	if out, err := markitect("tidy"); err != nil {
		return err
	} else if err := requireContains(out, "std 1.1.0"); err != nil {
		return err
	}
	if out, err := markitect("doc", "members/member"); err != nil {
		return err
	} else if err := requireContains(out, "members/member: Member (std/Concept)", "catalog/copy (link)", "src/members/member.go, .markitect/registry/members.md:"); err != nil {
		return err
	}
	if out, err := markitect("list", "--deps"); err != nil {
		return err
	} else if err := requireContains(out, "catalog (Ca 0, Ce 1, instability 1.00, abstractness 0.00, distance 0.00)\n  depends on members: catalog/copy links members/member\n"); err != nil {
		return err
	}
	if out, err := markitect("vet", "--strict"); err != nil {
		return err
	} else if out != "" {
		return fmt.Errorf("vet warned or noted: %s", tail(out))
	}
	if out, err := markitect("check", "--json"); err != nil {
		return err
	} else {
		var checked struct {
			FormatVersion int `json:"format_version"`
			Kind          string
			OK            bool
		}
		if err := decode(out, &checked); err != nil {
			return err
		}
		if checked.FormatVersion != 1 || checked.Kind != "check" || !checked.OK {
			return fmt.Errorf("check --json is not ok: %s", tail(out))
		}
	}
	if err := verifyAll(markitect, 6); err != nil {
		return err
	}
	if !strings.Contains(readFile(repo, ".markitect/review.sum"), `{"definition":"catalog/copy","verdict":"ok","inputs":{`) {
		return fmt.Errorf("the record is not as expected:\n%s", tail(readFile(repo, ".markitect/review.sum")))
	}

	// A change of the spec: the loan limit drops from five to three. Only the Rule itself
	// is to verify again: nothing links to it, and no file changed.
	limit := filepath.Join(repo, ".markitect", "spec", "loan-limit.md")
	if err := os.WriteFile(limit, []byte(strings.Replace(smokeSpec[".markitect/spec/loan-limit.md"], "`5`", "`3`", 1)), 0o644); err != nil {
		return err
	}
	if out, err := markitect("status"); err == nil || !strings.Contains(out, "To verify (1 of 6 Definitions)") || !strings.Contains(out, "\nloan-limit@") || !strings.Contains(out, "changed: self") {
		return fmt.Errorf("status after the change of the loan limit: %v\n%s", err, tail(out))
	}
	if err := verifyAll(markitect, 1); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(repo, ".markitect", "spec", "broken.md"), []byte("no front matter\n"), 0o644); err != nil {
		return err
	}
	if out, err := markitect("check"); err == nil || !strings.Contains(out, "definition.front_matter") {
		return fmt.Errorf("check accepted a Definition without front matter: %s", tail(out))
	}
	if out, err := markitect("status"); err == nil || !strings.Contains(out, "no list until it compiles") {
		return fmt.Errorf("status listed over a spec that does not compile: %s", tail(out))
	}
	for _, retired := range []string{"apply", "verify", "submit", "stop", "impact", "lint", "refs", "graph", "lock", "checklist", "answer", "gate"} {
		if _, err := markitect(retired); err == nil {
			return fmt.Errorf("the retired %s command was accepted", retired)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err == nil {
		return fmt.Errorf("a command made the project folder a Git repository")
	}
	return nil
}

// verifyAll records every item status lists ok, with the last file to look at
// as evidence, as a reviewer would, and checks that status then lists nothing.
func verifyAll(markitect func(...string) (string, error), want int) error {
	list, err := markitect("status")
	if err == nil {
		return fmt.Errorf("status listed nothing to verify:\n%s", tail(list))
	}
	recorded := 0
	item := ""
	for _, line := range strings.Split(list, "\n") {
		if !strings.HasPrefix(line, " ") && strings.Contains(line, "@") {
			item = line
			continue
		}
		if look, ok := strings.CutPrefix(line, "  look at: "); ok && item != "" {
			files := strings.Split(look, ", ")
			if _, err := markitect("mark", item, "ok", "-e", files[len(files)-1]+":1", "--cases", "1", "--covered", "1"); err != nil {
				return err
			}
			item = ""
			recorded++
		}
	}
	if recorded != want {
		return fmt.Errorf("recorded %d verdicts, want %d:\n%s", recorded, want, tail(list))
	}
	if out, err := markitect("status"); err != nil || !strings.Contains(out, "Nothing to verify") {
		return fmt.Errorf("status after every verdict: %v\n%s", err, tail(out))
	}
	return nil
}

// readFile reads a file of the project, or "" when it cannot.
func readFile(repo, file string) string {
	data, _ := os.ReadFile(filepath.Join(repo, filepath.FromSlash(file)))
	return string(data)
}
