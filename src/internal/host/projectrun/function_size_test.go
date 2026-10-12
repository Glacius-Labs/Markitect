package projectrun

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Measure declarations and literals by the same physical source span as the
// CLI guard, including supported-platform files and excluding test sources.
func TestRuntimeFunctionSize(t *testing.T) {
	for _, directory := range []string{".", "../projectconfig", "../projectexecutor"} {
		t.Run(filepath.Base(directory), func(t *testing.T) {
			positions := token.NewFileSet()
			packages, err := parser.ParseDir(positions, directory, func(info os.FileInfo) bool {
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
							start, end := positions.Position(node.Pos()), positions.Position(node.End())
							if lines := end.Line - start.Line + 1; lines > 300 {
								t.Errorf("%s: runtime function has %d lines; maximum is 300", start, lines)
							}
						}
						return true
					})
				}
			}
		})
	}
}
