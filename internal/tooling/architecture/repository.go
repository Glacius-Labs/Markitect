package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	guarded, err := CheckGuardedWriteUse(root, edges)
	if err != nil {
		return nil, err
	}
	out = append(out, guarded...)
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

// CheckGuardedWriteUse parses every file of DocumentWriterPackage that imports
// GuardedWritePackage. It reports each selector on the import name outside
// the guarded API, and every dot import, which would hide the identifiers
// used. A blank import references nothing. A local name that shadows the
// import name counts as the package.
func CheckGuardedWriteUse(root string, edges []Edge) ([]Violation, error) {
	files := map[string]string{}
	for _, e := range edges {
		if e.To == GuardedWritePackage && e.From == DocumentWriterPackage {
			files[e.File] = e.From
		}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Violation
	for _, name := range names {
		found, err := guardedWriteReferences(root, name, files[name])
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}
	sortViolations(out)
	return out, nil
}

func guardedWriteReferences(root, rel, from string) ([]Violation, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(root, filepath.FromSlash(rel)), nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	pkg := path.Base(GuardedWritePackage)
	var out []Violation
	report := func(pos token.Pos, rule string) {
		out = append(out, Violation{Edge{rel, fset.Position(pos).Line, from, GuardedWritePackage, strings.HasSuffix(rel, "_test.go")}, rule})
	}
	local := map[string]bool{}
	for _, imp := range f.Imports {
		imported, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		if imported != ModulePath+GuardedWritePackage {
			continue
		}
		name := pkg
		if imp.Name != nil {
			name = imp.Name.Name
		}
		switch name {
		case "_":
		case ".":
			report(imp.Pos(), "the document writer may not dot-import "+pkg+"; use qualified names so the gate can check them")
		default:
			local[name] = true
		}
	}
	api := make([]string, 0, len(guardedWriteAPI))
	for name := range guardedWriteAPI {
		api = append(api, name)
	}
	sort.Strings(api)
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && local[x.Name] && !guardedWriteAPI[sel.Sel.Name] {
				report(sel.Pos(), fmt.Sprintf("the document writer may use only the guarded write API (%s), not %s.%s", strings.Join(api, ", "), pkg, sel.Sel.Name))
			}
		}
		return true
	})
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
