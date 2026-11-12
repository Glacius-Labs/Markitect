package run_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

const (
	m     = project.SpecDir + "/"
	table = "| File | Realizes | Why |\n|---|---|---|\n"
)

func def(kind, title, body string) string {
	return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
}

// library is the fixture project: Root with a Rule, members with a sub-Package
// loans, and catalog, with every file accounted for. loan links member, and the
// anchor links the Packages.
func library() map[string]string {
	files, _ := libraries.Files("std")
	std := spec.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		std.Files = append(std.Files, spec.File{Path: f.Path, Data: f.Data})
	}
	return map[string]string{
		project.ManifestFile:                      "libraries:\n  std: 1.2.0\n",
		project.LockFile:                          string(spec.RenderLock([]spec.LibrarySource{std})),
		".claude/agents/markitect-reviewer.md":    "---\nname: markitect-reviewer\n---\nReview.\n",
		".claude/agents/markitect-builder.md":     "---\nname: markitect-builder\n---\nBuild.\n",
		".claude/settings.json":                   `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "command -v markitect >/dev/null 2>&1 || exit 0; markitect hook pretool"}]}]}}` + "\n",
		m + "project.md":                          def("std/Project", "Library", "Parts: [[members]], [[catalog]]."),
		m + "loan-limit.md":                       def("std/Rule", "Loan limit", "At most `5` open loans."),
		m + "members/members.md":                  def("std/Package", "Members", "The people who borrow."),
		m + "members/member.md":                   def("std/Concept", "Member", "A person."),
		m + "members/loans/loans.md":              def("std/Package", "Loans", "What a member has borrowed."),
		m + "members/loans/loan.md":               def("std/Concept", "Loan", "A [[members/member]] borrows."),
		m + "catalog/catalog.md":                  def("std/Package", "Catalog", "The books the library holds."),
		m + "catalog/book.md":                     def("std/Concept", "Book", "A book has a title."),
		project.RootPage:                          table + "| README.md | [[project]] | Overview |\n",
		project.RegistryDir + "/members.md":       "---\npaths: [src/members/**]\n---\n" + table + "| src/members/member.go | [[members/member]], [[members/members]] | Member |\n",
		project.RegistryDir + "/members/loans.md": "---\npaths: [src/members/loans/**]\n---\n" + table + "| src/members/loans/loan.go | [[members/loans/loan]], [[members/loans/loans]], [[loan-limit]] | Loans |\n",
		project.RegistryDir + "/catalog.md":       "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]], [[catalog/catalog]] | Book |\n",
		"README.md":                               "# Library\n",
		"src/members/member.go":                   "package members\n",
		"src/members/loans/loan.go":               "package loans\n\nconst Limit = 5\n",
		"src/catalog/book.go":                     "package catalog\n",
		".gitignore":                              ".claude/worktrees/\n",
	}
}

func write(t *testing.T, dir, file, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func has(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(text, w) {
			t.Fatalf("%q not in:\n%s", w, text)
		}
	}
}
