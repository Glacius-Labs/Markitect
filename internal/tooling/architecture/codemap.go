package architecture

import (
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	RulesSource   = "internal/tooling/architecture/rules.go"
	CodeMapPage   = "docs/development/code-map.md"
	CodeMapUpdate = "go test ./internal/tooling/architecture -run TestCodeMapPage -update"
)

// Summaries returns the first sentence of every table package's doc comment,
// read from the repository at root. A package needs exactly one non-test file
// with a package doc comment.
func (r Rules) Summaries(root string) (map[string]string, error) {
	out := map[string]string{}
	var problems []string
	for _, p := range r.Packages {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(p.Path)))
		if err != nil {
			return nil, err
		}
		var docs []string
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(p.Path), name), nil, parser.PackageClauseOnly|parser.ParseComments)
			if err != nil {
				return nil, err
			}
			if f.Doc != nil {
				docs = append(docs, name)
				out[p.Path] = new(doc.Package).Synopsis(f.Doc.Text())
			}
		}
		switch {
		case len(docs) == 0:
			problems = append(problems, fmt.Sprintf("package %s has no package doc comment", p.Path))
		case len(docs) > 1:
			problems = append(problems, fmt.Sprintf("package %s has package doc comments in %s; keep one", p.Path, strings.Join(docs, ", ")))
		case out[p.Path] == "" || strings.Contains(out[p.Path], "|"):
			problems = append(problems, fmt.Sprintf("package %s: the first sentence of its doc comment must be non-empty and contain no '|'", p.Path))
		}
	}
	if len(problems) != 0 {
		return nil, fmt.Errorf("%s", strings.Join(problems, "\n"))
	}
	return out, nil
}

// RenderCodeMap returns the generated page docs/development/code-map.md for
// the table and the package doc comments in the repository at root.
func (r Rules) RenderCodeMap(root string) ([]byte, error) {
	summaries, err := r.Summaries(root)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	pageDir := path.Dir(CodeMapPage)
	fmt.Fprintf(&b, "# Code map\n\n")
	fmt.Fprintf(&b, "This page lists every Go package in the repository with its layer, the first sentence of its doc comment and its owning document. Read the owning document before you change a package, and update it when the package's behavior changes.\n\n")
	fmt.Fprintf(&b, "The page is generated from the rule table in [rules.go](%s) and the package doc comments. Do not edit it by hand. Change the table or the doc comment, then regenerate the page from the repository root:\n\n", relativeLink(pageDir, RulesSource))
	fmt.Fprintf(&b, "```powershell\n%s\n```\n\n", CodeMapUpdate)
	fmt.Fprintf(&b, "`TestRepositoryArchitecture` checks every production, test and platform file against the table: a package imports only the layers its own layer lists and the named exceptions, only test files import a test-only layer, a standalone layer imports no Markitect package, and the document writer `%s` uses only the reviewed guarded API of `%s`. Every Go package needs exactly one row, an existing owning document and one package doc comment. The source packager accepts exactly the table's packages below `cmd/` and `internal/`. [Architecture](%s) describes the layers.\n\n", DocumentWriterPackage, GuardedWritePackage, relativeLink(pageDir, "docs/architecture.md")+"#layers")

	counts := map[string]int{}
	for _, p := range r.Packages {
		counts[p.Layer]++
	}
	fmt.Fprintf(&b, "## Layers\n\n| Layer | Meaning | May import | Packages |\n|---|---|---|---|\n")
	for _, l := range r.Layers {
		imports := "nothing"
		if len(l.Imports) != 0 {
			imports = "`" + strings.Join(l.Imports, "`, `") + "`"
		}
		meaning := l.Meaning
		if l.TestOnly {
			meaning += " Only test files may import it."
		}
		if l.Standalone {
			meaning += " It imports no Markitect package, not even from tests."
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %d |\n", l.Name, meaning, imports, counts[l.Name])
	}

	exceptions := append([]Exception(nil), r.Exceptions...)
	sort.SliceStable(exceptions, func(i, j int) bool {
		a, b := exceptions[i], exceptions[j]
		if a.TestOnly != b.TestOnly {
			return !a.TestOnly
		}
		if a.From != b.From {
			return a.From < b.From
		}
		return a.To < b.To
	})
	if len(exceptions) != 0 {
		fmt.Fprintf(&b, "\n## Exceptions\n\n| From | To | Files | Reason |\n|---|---|---|---|\n")
		for _, x := range exceptions {
			files := "all"
			if x.TestOnly {
				files = "tests only"
			}
			fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s |\n", x.From, x.To, files, x.Reason)
		}
	}

	for _, l := range r.Layers {
		if counts[l.Name] == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n| Package | Summary | Owning document |\n|---|---|---|\n", l.Title)
		for _, p := range r.Packages {
			if p.Layer != l.Name {
				continue
			}
			file, anchor, _ := strings.Cut(p.Doc, "#")
			target := relativeLink(pageDir, file)
			if anchor != "" {
				target += "#" + anchor
			}
			fmt.Fprintf(&b, "| `%s` | %s | [%s](%s) |\n", p.Path, summaries[p.Path], p.Doc, target)
		}
	}
	return []byte(b.String()), nil
}

// relativeLink returns the slash path from directory from to file to; both
// are clean repository-relative paths.
func relativeLink(from, to string) string {
	a, b := strings.Split(from, "/"), strings.Split(to, "/")
	common := 0
	for common < len(a) && common < len(b)-1 && a[common] == b[common] {
		common++
	}
	parts := make([]string, 0, len(a)-common+len(b)-common)
	for range a[common:] {
		parts = append(parts, "..")
	}
	return strings.Join(append(parts, b[common:]...), "/")
}
