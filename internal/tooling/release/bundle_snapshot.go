package release

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

func snapshotModuleFiles(snapshot *snapshot.Snapshot) (map[string][]byte, error) {
	prefix := ""
	if _, ok := snapshot.Files["go.mod"]; !ok {
		prefix = "tools/markitect/"
		if _, exists := snapshot.Files[prefix+"go.mod"]; !exists {
			return nil, errors.New("fixed source snapshot is missing go.mod")
		}
	}
	module := make(map[string][]byte)
	for name, data := range snapshot.Files {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		relative := strings.TrimPrefix(name, prefix)
		if relative == "" {
			continue
		}
		if err := safeArchivePath(relative); err != nil {
			return nil, fmt.Errorf("unsafe source snapshot path %q: %w", name, err)
		}
		include := relative == "go.mod" || relative == "go.sum" || relative == "README.md" || relative == "LICENSE"
		if strings.HasPrefix(relative, "cmd/") || strings.HasPrefix(relative, "internal/") {
			ext := strings.ToLower(path.Ext(relative))
			include = ext == ".go" || (relative == embeddedNoticesPath || relative == "internal/licenses/notices.md") || embeddedAuthoringSource(relative)
		}
		if strings.HasPrefix(relative, "schema/") {
			ext := strings.ToLower(path.Ext(relative))
			include = ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".md"
		}
		if include {
			if !snapshotRegularFile(snapshot, name) {
				return nil, fmt.Errorf("source snapshot file %q is not regular", name)
			}
			module[relative] = data
		}
	}
	return module, nil
}

func collectModuleSnapshot(files map[string][]byte) ([]sourceFile, error) {
	selected := make([]sourceFile, 0, len(files))
	for _, name := range []string{"go.mod", "go.sum"} {
		data, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("fixed source snapshot is missing %s", name)
		}
		normalized, err := normalizeTextSource(name, data)
		if err != nil {
			return nil, fmt.Errorf("invalid source snapshot text %s: %w", name, err)
		}
		selected = append(selected, sourceFile{name: name, data: normalized})
	}
	for name, data := range files {
		if name == "go.mod" || name == "go.sum" {
			continue
		}
		ext := strings.ToLower(path.Ext(name))
		include := name == "README.md" || name == "LICENSE"
		if strings.HasPrefix(name, "cmd/") || strings.HasPrefix(name, "internal/") {
			include = ext == ".go" || (name == embeddedNoticesPath || name == "internal/licenses/notices.md") || embeddedAuthoringSource(name)
		}
		if strings.HasPrefix(name, "schema/") {
			include = ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".md"
		}
		if !include {
			continue
		}
		normalized, err := normalizeTextSource(name, data)
		if err != nil {
			return nil, fmt.Errorf("invalid source snapshot text %s: %w", name, err)
		}
		selected = append(selected, sourceFile{name: name, data: normalized})
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].name < selected[j].name })
	if len(selected) < 4 {
		return nil, errors.New("fixed source snapshot Markitect module is incomplete")
	}
	if err := validateArchivePaths(selected); err != nil {
		return nil, err
	}
	if err := validateSnapshotModuleLayout(selected); err != nil {
		return nil, err
	}
	return selected, nil
}

func embeddedAuthoringSource(name string) bool {
	if name == "internal/host/embedded/project.yaml" || name == "internal/authoring/project.yaml" {
		return true
	}
	ext := strings.ToLower(path.Ext(name))
	return (strings.HasPrefix(name, "internal/host/embedded/resources/") || strings.HasPrefix(name, "internal/authoring/resources/")) && (ext == ".yaml" || ext == ".yml")
}

func validateSnapshotModuleLayout(files []sourceFile) error {
	present := make(map[string]bool, len(files))
	for _, file := range files {
		present[file.name] = true
	}
	for _, name := range []string{"go.mod", "go.sum", "LICENSE", "cmd/markitect/main.go"} {
		if !present[name] {
			return fmt.Errorf("fixed source snapshot is missing required module file %s", name)
		}
	}
	internalFound := false
	for name := range present {
		if strings.HasPrefix(name, "internal/") {
			internalFound = true
			break
		}
	}
	if !internalFound {
		return errors.New("fixed source snapshot is missing the internal source tree")
	}
	var goMod []byte
	for _, file := range files {
		if file.name == "go.mod" {
			goMod = file.data
			break
		}
	}
	for _, line := range strings.Split(string(goMod), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if !strings.HasPrefix(line, "module ") {
			break
		}
		module := strings.TrimSpace(strings.TrimPrefix(line, "module "))
		if unquoted, err := strconv.Unquote(module); err == nil {
			module = unquoted
		}
		if module == sourceRepository {
			return nil
		}
		break
	}
	return fmt.Errorf("fixed source snapshot go.mod must declare module %s", sourceRepository)
}

func snapshotText(snapshot *snapshot.Snapshot, name string) ([]byte, error) {
	data, ok := snapshot.Files[name]
	if !ok {
		return nil, fmt.Errorf("fixed source snapshot is missing %s", name)
	}
	if !snapshotRegularFile(snapshot, name) {
		return nil, fmt.Errorf("source snapshot file %q is not regular", name)
	}
	return normalizeTextSource(name, data)
}

func snapshotRegularFile(snapshot *snapshot.Snapshot, name string) bool {
	if snapshot == nil || snapshot.Modes == nil {
		return false
	}
	mode := snapshot.Modes[name]
	return mode == "100644" || mode == "100755"
}

func validateSourceVersion(snapshot *snapshot.Snapshot, version string) error {
	sourcePath := "internal/host/cli/version.go"
	declarationToken := token.VAR
	// Historical immutable source distributions have their version in the
	// executable entrypoint. Compatibility is confined to release tooling.
	if _, exists := snapshot.Files[sourcePath]; !exists {
		sourcePath = "cmd/markitect/main.go"
		declarationToken = token.VAR
	}
	data, err := snapshotText(snapshot, sourcePath)
	if err != nil {
		return err
	}
	file, err := parser.ParseFile(token.NewFileSet(), sourcePath, data, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse fixed source version declaration: %w", err)
	}
	found := false
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != declarationToken {
			continue
		}
		for _, spec := range group.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range valueSpec.Names {
				if name.Name != "version" {
					continue
				}
				if found || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 || index != 0 {
					return errors.New("source version must be declared once as a string literal version declaration")
				}
				literal, ok := valueSpec.Values[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return errors.New("source version must be a string literal version declaration")
				}
				declared, err := strconv.Unquote(literal.Value)
				if err != nil || declared != version {
					return fmt.Errorf("bundle version %q does not match source version declaration", version)
				}
				found = true
			}
		}
	}
	if !found {
		return errors.New("fixed source snapshot is missing its version declaration")
	}
	return nil
}
