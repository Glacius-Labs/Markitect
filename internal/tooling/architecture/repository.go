package architecture

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// CheckRepository runs every rule of the table over the repository at root.
// It first checks the table and the package inventory; an invalid table, a
// Go package without a row, a row without a package or a missing owning
// document is a repository error. It then reports every import violation.
func (r Rules) CheckRepository(root string) ([]Violation, error) {
	problems, err := r.CheckInventory(root)
	if err != nil {
		return nil, err
	}
	if len(problems) != 0 {
		return nil, fmt.Errorf("invalid rule table:\n%s", strings.Join(problems, "\n"))
	}
	edges, err := Inspect(root)
	if err != nil {
		return nil, err
	}
	out := r.Check(edges)
	selfImports, err := r.checkSelfImports(root, edges)
	if err != nil {
		return nil, err
	}
	out = append(out, selfImports...)
	sortViolations(out)
	return out, nil
}

// CheckInventory reports every inconsistency of the table, every difference
// between its packages and the Go packages below root, and every owning
// document that does not exist.
func (r Rules) CheckInventory(root string) ([]string, error) {
	packages, err := Packages(root)
	if err != nil {
		return nil, err
	}
	problems := r.Validate()
	anchors := map[string]map[string]bool{}
	present := map[string]bool{}
	for _, p := range packages {
		present[p] = true
		if _, ok := r.pkg(p); !ok {
			problems = append(problems, fmt.Sprintf("missing package %s: add a row to the rule table", p))
		}
	}
	for _, p := range r.Packages {
		if !present[p.Path] {
			problems = append(problems, fmt.Sprintf("extra row %s: no Go package exists there; remove it from the rule table", p.Path))
		}
		if err := checkDoc(root, p.Doc, anchors); err != nil {
			problems = append(problems, fmt.Sprintf("package %s: owning document: %v", p.Path, err))
		}
	}
	return problems, nil
}

// Packages lists the repository-relative directory of every Go package below
// root, following `go list ./...`: a directory with at least one .go file,
// including test-only and platform-specific packages. It skips .git, vendor,
// testdata, node_modules, names starting with "." or "_", and nested modules.
// Nested modules under product cmd/ and internal/ are errors: the product
// uses the root module and must not disappear from this inventory.
func Packages(root string) ([]string, error) {
	found := map[string]bool{}
	var modules []string
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		base := entry.Name()
		if entry.IsDir() {
			if rel != "." && (base == ".git" || base == "vendor" || base == "testdata" || base == "node_modules" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		switch {
		case base == "go.mod" && rel != "go.mod":
			if strings.HasPrefix(rel, "cmd/") || strings.HasPrefix(rel, "internal/") {
				return fmt.Errorf("nested Go module %s is forbidden: cmd/ and internal/ use the root Go module", rel)
			}
			modules = append(modules, path.Dir(rel))
		case strings.HasSuffix(base, ".go") && !strings.HasPrefix(base, ".") && !strings.HasPrefix(base, "_"):
			found[path.Dir(rel)] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out []string
	for dir := range found {
		nested := false
		for _, module := range modules {
			nested = nested || dir == module || strings.HasPrefix(dir, module+"/")
		}
		if !nested {
			out = append(out, dir)
		}
	}
	sort.Strings(out)
	return out, nil
}

// Edge.Test alone cannot distinguish internal from external Go tests.
// checkSelfImports verifies the package clause before admitting a test's
// import of its own directory; ordinary test imports keep the same policy.
func (r Rules) checkSelfImports(root string, edges []Edge) ([]Violation, error) {
	var out []Violation
	for _, e := range edges {
		if _, known := r.pkg(e.From); !known || !e.Test || e.From != e.To {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(e.File)), nil, parser.PackageClauseOnly)
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(f.Name.Name, "_test") {
			out = append(out, Violation{e, "only external test packages may import their own package"})
		}
	}
	return out, nil
}

func checkDoc(root, doc string, cache map[string]map[string]bool) error {
	file, anchor, _ := strings.Cut(doc, "#")
	if file == "" || file != path.Clean(file) || path.IsAbs(file) || strings.HasPrefix(file, "../") || strings.Contains(file, "\\") {
		return fmt.Errorf("%q is not a clean repository-relative path", doc)
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s does not exist", file)
	}
	if anchor == "" {
		return nil
	}
	if cache[file] == nil {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return err
		}
		cache[file] = markdownAnchors(string(data))
	}
	if !cache[file][anchor] {
		return fmt.Errorf("%s has no anchor #%s", file, anchor)
	}
	return nil
}

var explicitID = regexp.MustCompile(`[<\s]id="([^"]+)"`)

// markdownAnchors returns GitHub-style ATX heading anchors and explicit HTML
// ids outside fenced code: the subset of scripts/check-docs.py these documents use.
func markdownAnchors(text string) map[string]bool {
	anchors := map[string]bool{}
	counts := map[string]int{}
	fence := ""
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimLeft(line, " ")
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			continue
		}
		for _, match := range explicitID.FindAllStringSubmatch(line, -1) {
			anchors[match[1]] = true
		}
		heading := strings.TrimLeft(trimmed, "#")
		if level := len(trimmed) - len(heading); level == 0 || level > 6 || heading != "" && heading[0] != ' ' {
			continue
		}
		heading = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(heading), "#"))
		var slug strings.Builder
		for _, r := range strings.ToLower(heading) {
			switch {
			case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
				slug.WriteRune(r)
			case unicode.IsSpace(r):
				slug.WriteRune('-')
			}
		}
		base := strings.Trim(slug.String(), "-")
		if base == "" {
			continue
		}
		candidate := base
		if n := counts[base]; n > 0 {
			candidate = fmt.Sprintf("%s-%d", base, n)
		}
		counts[base]++
		anchors[candidate] = true
	}
	return anchors
}
