package architecture

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePrefix = "example.com/acme/modular-service/internal/modules/"

func TestModulesDoNotImportSiblingModules(t *testing.T) {
	err := filepath.WalkDir(filepath.Join("..", "modules"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			imported := strings.Trim(imp.Path.Value, "\"")
			if strings.HasPrefix(imported, modulePrefix) {
				return violation{path, imported}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

type violation struct{ path, imported string }

func (v violation) Error() string { return v.path + " imports forbidden sibling module " + v.imported }
