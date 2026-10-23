package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// GuardedWritePackage supplies the independently guarded document writer.
const GuardedWritePackage = "internal/tooling/guardedwrite"

var guardedWriteAPI = map[string]bool{
	"OpenRoot": true, "CaptureFiles": true, "CaptureV1Files": true, "ApplyOrderedChecked": true, "Change": true, "Result": true,
}

// CheckRepository runs every import rule over the repository at root: the
// coarse rules of Check and the code-map rules of CheckMapLayers and
// CheckGuardedWriteUse. It validates the complete code map from root before
// relying on its package layers; an invalid map is a repository error.
func CheckRepository(root string) ([]Violation, error) {
	edges, err := Inspect(root)
	if err != nil {
		return nil, err
	}
	m, err := LoadCodeMap(root)
	if err != nil {
		return nil, err
	}
	problems, err := CheckCodeMap(root, m)
	if err != nil {
		return nil, err
	}
	if len(problems) != 0 {
		return nil, fmt.Errorf("invalid code map:\n%s", strings.Join(problems, "\n"))
	}
	// A classified target name is not proof that its producer exists. Do not
	// admit edges to absent packages merely because their prefix is known.
	mapped := mapLayers(m)
	for _, e := range edges {
		if target, _ := targetLayer(e.To); target != "" && mapped[e.To] == "" {
			return nil, fmt.Errorf("%s:%d: target import %s has no actual package in %s", e.File, e.Line, e.To, CodeMapSource)
		}
	}
	guarded, err := CheckGuardedWriteUse(root, edges, m)
	if err != nil {
		return nil, err
	}
	out := Check(edges)
	out = append(out, guarded...)
	selfImports, err := checkTargetSelfImports(root, edges)
	if err != nil {
		return nil, err
	}
	out = append(out, selfImports...)
	sortViolations(out)
	return out, nil
}

// CheckGuardedWriteUse parses every file of a product package, other than
// GuardedWritePackage itself, that imports GuardedWritePackage. It reports
// each selector on the import name outside the guarded API, and every dot
// import, which would hide the identifiers used. A blank import references
// nothing. A local name that shadows the import name counts as the package.
func CheckGuardedWriteUse(root string, edges []Edge, m CodeMap) ([]Violation, error) {
	_ = m
	files := map[string]string{}
	for _, e := range edges {
		if e.To == GuardedWritePackage && e.From != GuardedWritePackage && e.From == "internal/adapters/documents" {
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
			report(imp.Pos(), "product package may not dot-import "+pkg+"; use qualified names so the gate can check them")
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
				report(sel.Pos(), fmt.Sprintf("product package may use only the guarded write API (%s), not %s.%s", strings.Join(api, ", "), pkg, sel.Sel.Name))
			}
		}
		return true
	})
	return out, nil
}

func mapLayers(m CodeMap) map[string]string {
	layers := make(map[string]string, len(m.Packages))
	for _, p := range m.Packages {
		layers[p.Path] = p.Layer
	}
	return layers
}

func sortViolations(out []Violation) {
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
}
