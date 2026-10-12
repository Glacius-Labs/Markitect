package projectcli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Product transports may use service DTOs, but calls and composition belong
// to projectapp. This catches a new bypass without exercising a provider.
func TestApplicationFacadeBoundary(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	const prefix = "github.com/Glacius-Labs/Markitect/src/internal/"
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, entry.Name(), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		services := map[string]string{}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(path, prefix) || strings.HasSuffix(path, "/projectapp") || strings.HasSuffix(path, "/mcp") {
				continue
			}
			name := filepath.Base(path)
			if spec.Name != nil {
				name = spec.Name.Name
			}
			if name == "." {
				t.Errorf("%s: service dot import bypasses the facade", fset.Position(spec.Pos()))
			}
			services[name] = path
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			name, ok := selector.X.(*ast.Ident)
			if ok && services[name.Name] != "" {
				t.Errorf("%s: %s.%s calls a service directly; delegate through projectapp", fset.Position(call.Pos()), name.Name, selector.Sel.Name)
			}
			return true
		})
	}
}

func TestCLIFunctionSize(t *testing.T) {
	for _, dir := range []string{".", "../cli"} {
		fset := token.NewFileSet()
		packages, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
			return !strings.HasSuffix(info.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range packages {
			for _, file := range pkg.Files {
				ast.Inspect(file, func(node ast.Node) bool {
					switch node.(type) {
					case *ast.FuncDecl, *ast.FuncLit:
						start, end := fset.Position(node.Pos()), fset.Position(node.End())
						if lines := end.Line - start.Line + 1; lines > 300 {
							t.Errorf("%s: CLI function has %d lines; maximum is 300", start, lines)
						}
					}
					return true
				})
			}
		}
	}
}
