package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *smoke) version() error {
	out, err := s.run(s.repo, nil, s.tool("markitect"), "version")
	if err != nil {
		return err
	}
	if version := strings.TrimSpace(strings.TrimPrefix(out, "markitect ")); version == "" || version == strings.TrimSpace(out) {
		return fmt.Errorf("unexpected version output: %q", out)
	}
	return nil
}

// smokeModel is a small project in the model language: Root with a Rule, and
// the Packages members and catalog, with registry pages for every file.
var smokeModel = map[string]string{
	".markitect/model/loan-limit.md":     "---\nkind: std/Rule\n---\n# Loan limit\n\nA member has at most `5` open loans; see [[members]].\n",
	".markitect/model/members/member.md": "---\nkind: std/Concept\n---\n# Member\n\nA person who borrows copies.\n",
	".markitect/model/catalog/copy.md":   "---\nkind: std/Concept\n---\n# Copy\n\nA copy is lent to a [[members/member]].\n",
	".markitect/registry/members.md":     "---\npaths: [src/members/**]\n---\n# Registry: members\n\n| File | Realizes | Why |\n|---|---|---|\n| `src/members/member.go` | [[members/member]] [[loan-limit]] | Member type and loan limit |\n",
	".markitect/registry/catalog.md":     "---\npaths: [src/catalog/**]\n---\n# Registry: catalog\n\n| File | Realizes | Why |\n|---|---|---|\n| `src/catalog/copy.go` | [[catalog/copy]] | Copy type |\n",
	"README.md":                          "# Lending library\n",
	"src/members/member.go":              "package members\n",
	"src/catalog/copy.go":                "package catalog\n",
}

// modelCommands drives the packaged CLI over a small project in a plain folder,
// with no Git: init, check, lock, refs, graph and lint; the work (checklist,
// answer and gate) that certifies the folder; impact from that certificate; and
// the refusal of a model that does not compile. It starts no agent.
func (s *smoke) modelCommands() error {
	repo := filepath.Join(s.work, "model-commands")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		return err
	}
	markitect := func(args ...string) (string, error) { return s.run(repo, nil, s.tool("markitect"), args...) }
	if out, err := markitect("init", "--name", "Lending library"); err != nil {
		return err
	} else if err := requireContains(out, "created .markitect/manifest.yaml", "wrote .markitect/lock.yaml: std 1.0.0", "installed the skills"); err != nil {
		return err
	}
	if err := requirePresent(repo, ".claude/skills/markitect-modeling/SKILL.md", ".agents/skills/markitect-extract/SKILL.md", ".markitect/model/project.md"); err != nil {
		return err
	}
	for name, content := range smokeModel {
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
	if err := requireContains(out, "ok: 4 Definitions in 3 Packages, 3 files accounted"); err != nil {
		return err
	}
	if out, err := markitect("lock"); err != nil {
		return err
	} else if err := requireContains(out, "std 1.0.0"); err != nil {
		return err
	}
	if out, err := markitect("refs", "members/member"); err != nil {
		return err
	} else if err := requireContains(out, "[[catalog/copy]]", "`src/members/member.go`"); err != nil {
		return err
	}
	if out, err := markitect("graph"); err != nil {
		return err
	} else if err := requireContains(out, "```mermaid", "| catalog | members |"); err != nil {
		return err
	}
	if out, err := markitect("lint"); err != nil {
		return err
	} else if err := requireContains(out, "# Lint"); err != nil {
		return err
	}
	cert, err := s.certify(repo, markitect)
	if err != nil {
		return err
	}

	// M2's change: the loan limit drops from five to three. Every Manager
	// below Root inherits the changed Rule.
	limit := filepath.Join(repo, ".markitect", "model", "loan-limit.md")
	if err := os.WriteFile(limit, []byte(strings.Replace(smokeModel[".markitect/model/loan-limit.md"], "`5`", "`3`", 1)), 0o644); err != nil {
		return err
	}
	if out, err := markitect("gate", "--verify"); err == nil || !strings.Contains(out, "loan-limit.md") {
		return fmt.Errorf("gate --verify accepted a changed spec: %s", tail(out))
	}
	out, err = markitect("impact", "--json")
	if err != nil {
		return err
	}
	var impact struct {
		Base       string
		Packages   []string
		Checklists []struct {
			Package   string
			Inherited []struct{ Definition string }
		}
	}
	if err := decode(out, &impact); err != nil {
		return err
	}
	if impact.Base != cert {
		return fmt.Errorf("impact starts from %q, not the certificate %q", impact.Base, cert)
	}
	if strings.Join(impact.Packages, ",") != ",catalog,members" {
		return fmt.Errorf("impact of the loan limit reaches %v", impact.Packages)
	}
	for _, c := range impact.Checklists[1:] {
		if len(c.Inherited) != 1 || c.Inherited[0].Definition != "loan-limit" {
			return fmt.Errorf("checklist of %s lacks the inherited loan limit: %s", c.Package, tail(out))
		}
	}

	if err := os.WriteFile(filepath.Join(repo, ".markitect", "model", "broken.md"), []byte("no front matter\n"), 0o644); err != nil {
		return err
	}
	if out, err := markitect("check"); err == nil || !strings.Contains(out, "definition.front_matter") {
		return fmt.Errorf("check accepted a Definition without front matter: %s", tail(out))
	}
	if _, err := markitect("impact"); err == nil {
		return fmt.Errorf("impact ran over a model that does not compile")
	}
	for _, retired := range []string{"apply", "verify", "submit", "stop"} {
		if _, err := markitect(retired); err == nil {
			return fmt.Errorf("the retired %s command was accepted", retired)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err == nil {
		return fmt.Errorf("a command made the project folder a Git repository")
	}
	return nil
}

// certify passes the work of the folder through gate, every item answered ok
// with a pointer to README.md, as a reviewer would, and returns the certificate
// gate wrote.
func (s *smoke) certify(repo string, markitect func(...string) (string, error)) (string, error) {
	if _, err := markitect("checklist"); err != nil {
		return "", err
	}
	list, err := markitect("checklist", "--review")
	if err != nil {
		return "", err
	}
	var lines []string
	for _, line := range strings.Split(list, "\n") {
		if rest, ok := strings.CutPrefix(line, "[ ] "); ok {
			if i := strings.Index(rest, `: "`); i >= 0 {
				rest = rest[:i]
			}
			lines = append(lines, rest+": ok README.md:1")
		}
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("the review lists no item:\n%s", tail(list))
	}
	// Outside the project folder, which accounts for every file of its own.
	answers := filepath.Join(s.work, "answers.txt")
	if err := os.WriteFile(answers, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return "", err
	}
	if _, err := markitect("answer", answers); err != nil {
		return "", err
	}
	passed, err := markitect("gate")
	if err != nil {
		return "", err
	}
	for _, f := range strings.Fields(passed) {
		if f = strings.TrimSuffix(f, ","); strings.HasSuffix(f, ".json") {
			if _, err := markitect("gate", "--verify"); err != nil {
				return "", err
			}
			return f, nil
		}
	}
	return "", fmt.Errorf("gate named no certificate:\n%s", tail(passed))
}
