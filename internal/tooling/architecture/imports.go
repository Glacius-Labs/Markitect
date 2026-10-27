// Package architecture holds the repository's one architecture rule table and
// enforces it without compiling the inspected code: every production, test and
// platform-specific file imports only what its package's layer allows.
package architecture

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const ModulePath = "github.com/Glacius-Labs/Markitect/"

type Edge struct {
	File     string
	Line     int
	From, To string
	Test     bool
}
type Violation struct {
	Edge Edge
	Rule string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s:%d: %s imports %s: %s", v.Edge.File, v.Edge.Line, v.Edge.From, v.Edge.To, v.Rule)
}

// Check applies the table to import edges. A package without a row has no
// import privilege, including test-only and platform-specific packages. An
// edge with an empty To only classifies its file's package.
func (r Rules) Check(edges []Edge) []Violation {
	var out []Violation
	for _, e := range edges {
		from, fromKnown := r.pkg(e.From)
		fromLayer, _ := r.layer(from.Layer)
		to, toKnown := r.pkg(e.To)
		toLayer, _ := r.layer(to.Layer)
		rule := ""
		switch {
		case !fromKnown:
			rule = "unclassified package: add a row to the rule table"
		case e.To == "":
		case fromLayer.Standalone:
			rule = "layer " + from.Layer + " may not import Markitect packages"
		case !toKnown:
			rule = "unclassified package: the rule table has no row for " + e.To
		case e.From == e.To && !e.Test:
			rule = "self-import is forbidden"
		case e.From == e.To:
		case toLayer.TestOnly && !e.Test:
			rule = "only test files may import " + e.To
		case toLayer.TestOnly:
		case r.allows(from.Layer, to.Layer):
		case r.exception(e.From, e.To, e.Test):
		default:
			rule = fmt.Sprintf("layer %s may not import layer %s", from.Layer, to.Layer)
		}
		if rule != "" {
			out = append(out, Violation{e, rule})
		}
	}
	sortViolations(out)
	return out
}

// Inspect returns one classifying edge for every Go file below cmd/,
// internal/, tools/, examples/ and packaging/, and one edge for each of its
// imports of a Markitect package.
func Inspect(root string) ([]Edge, error) {
	var edges []Edge
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name != root && (entry.Name() == ".git" || entry.Name() == ".cache" || entry.Name() == ".artifacts" || entry.Name() == "vendor" || entry.Name() == "testdata" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			// A nested module outside the product is not part of the root
			// module, as in Packages. Product nested modules stay inspected;
			// Packages refuses them.
			if name != root && !isProductDir(root, name) {
				if _, err := os.Stat(filepath.Join(name, "go.mod")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !strings.HasPrefix(rel, "cmd/") && !strings.HasPrefix(rel, "internal/") && !strings.HasPrefix(rel, "tools/") && !strings.HasPrefix(rel, "examples/") && !strings.HasPrefix(rel, "packaging/") {
			return nil
		}
		packagePath := filepath.ToSlash(filepath.Dir(rel))
		// Classify every file, even when it has no local imports. A test file
		// visited first must not hide production code in a test-only package.
		edges = append(edges, Edge{File: rel, From: packagePath, Test: strings.HasSuffix(rel, "_test.go")})
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			imported, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			// Only imports between Markitect packages are edges; third-party
			// libraries are not restricted in any layer (DEC-020).
			if strings.HasPrefix(imported, ModulePath) {
				edges = append(edges, Edge{rel, fset.Position(imp.Pos()).Line, packagePath, strings.TrimPrefix(imported, ModulePath), strings.HasSuffix(rel, "_test.go")})
			}
		}
		return nil
	})
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.To < b.To
	})
	return edges, err
}

// isProductDir reports whether dir lies in cmd/ or internal/ below root.
func isProductDir(root, dir string) bool {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == "cmd" || rel == "internal" || strings.HasPrefix(rel, "cmd/") || strings.HasPrefix(rel, "internal/")
}
