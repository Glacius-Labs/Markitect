// Package architecture enforces repository import ownership without compiling
// the inspected code. It also inspects tests and all platform-specific files.
package architecture

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
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

// layer names an explicit current responsibility. Unknown and removed packages
// have no import privilege, including test-only and platform-specific packages.
func layer(p string) (string, string) {
	if name, owner := targetLayer(p); name != "" {
		return name, owner
	}
	return "unknown", p
}

func Check(edges []Edge) []Violation {
	var out []Violation
	for _, e := range edges {
		from, owner := layer(e.From)
		to, target := layer(e.To)
		rule := ""
		switch {
		case from == "unknown":
			rule = "unclassified product package"
		case e.To == "":
		case from == "standalone" || from == "test-support":
			rule = "standalone tooling and test support may not import Markitect packages"
		case to == "unknown":
			rule = "unclassified internal package"
		case e.From == e.To && !e.Test:
			rule = "self-import is forbidden"
		case to == "test-support" && !e.Test:
			rule = "only test files may import testkit"
		case to == "test-support":
		case owner == target:
		case e.Test && contains(responsibilityTestImports[owner], target):
		case !contains(responsibilityImports[owner], target):
			rule = "responsibility direction forbids " + owner + " -> " + target
		}
		if rule != "" {
			out = append(out, Violation{e, rule})
		}
	}
	sortViolations(out)
	return out
}

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
		// Only product source and explicitly classified test/tool harnesses.
		if !strings.HasPrefix(rel, "cmd/") && !strings.HasPrefix(rel, "internal/") && !strings.HasPrefix(rel, "src/") && !strings.HasPrefix(rel, "tools/") && !strings.HasPrefix(rel, "examples/") && !strings.HasPrefix(rel, "integration/") && !strings.HasPrefix(rel, "experiments/") && !strings.HasPrefix(rel, "benchmark/") {
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
