// Package inputs resolves explicitly declared non-Markitect files for a graph.
// It does not inspect file contents, infer dependencies, or recurse into inputs.
package inputs

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"markitect/internal/core"
)

// Resolve validates each resource's spec.files entries against the provided
// repository snapshot and returns resolved paths grouped by resource key.
// The paths in each slice retain the order declared in the resource.
func Resolve(graph *core.Graph, files map[string][]byte) (map[string][]string, []core.Diagnostic) {
	resolved := map[string][]string{}
	var diagnostics []core.Diagnostic
	add := func(resource *core.Resource, code, message string) {
		d := core.Diagnostic{Code: code, Message: message}
		if resource != nil {
			d.Path, d.Line = resource.Path, resource.Line
		}
		diagnostics = append(diagnostics, d)
	}
	if graph == nil {
		add(nil, "input.graph", "graph is nil")
		return resolved, diagnostics
	}
	if graph.Project == nil {
		add(nil, "input.project", "graph has no Project resource")
		return resolved, diagnostics
	}
	keys := make([]string, 0, len(graph.Resources))
	for key := range graph.Resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		resource := graph.Resources[key]
		if resource == nil || len(resource.Spec.Files) == 0 {
			continue
		}
		if resource.Kind == "Project" {
			add(resource, "input.project-files", "Project resources must not declare external input files")
			continue
		}
		declared := make([]string, 0, len(resource.Spec.Files))
		seen := map[string]bool{}
		for _, file := range resource.Spec.Files {
			clean, err := safePath(file)
			if err != nil {
				add(resource, "input.path", fmt.Sprintf("unsafe input path %q: %s", file, err))
				continue
			}
			if seen[clean] {
				add(resource, "input.duplicate", fmt.Sprintf("input path %q is declared more than once", clean))
				continue
			}
			seen[clean] = true
			if isProjectConfig(clean, graph.Project.Path) {
				add(resource, "input.project-config", fmt.Sprintf("project configuration %q cannot be an external input", clean))
				continue
			}
			if companion, typed := typedCompanion(graph, clean); typed {
				add(resource, "input.typed-companion", fmt.Sprintf("%q is the generated view for typed resource %s; reference the resource with uses or rules", clean, companion.Key()))
				continue
			}
			sourceArea := owner(graph, resource.Path)
			inputArea := owner(graph, clean)
			if sourceArea == nil || inputArea == nil {
				add(resource, "input.area", fmt.Sprintf("input path %q or its resource is outside every project area", clean))
				continue
			}
			if sourceArea.Name != inputArea.Name && !contains(sourceArea.Imports, inputArea.Name) {
				add(resource, "input.scope", fmt.Sprintf("input path %q belongs to area %q, which area %q does not import", clean, inputArea.Name, sourceArea.Name))
				continue
			}
			data, ok := files[clean]
			if !ok {
				if hasDescendant(files, clean) {
					add(resource, "input.directory", fmt.Sprintf("input path %q names a directory, not a file", clean))
				} else {
					add(resource, "input.missing", fmt.Sprintf("input file %q is missing from the repository snapshot", clean))
				}
				continue
			}
			if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
				add(resource, "input.binary", fmt.Sprintf("input file %q is not valid UTF-8 text", clean))
				continue
			}
			declared = append(declared, clean)
		}
		resolved[key] = declared
	}
	sort.Slice(diagnostics, func(i, j int) bool {
		a, b := diagnostics[i], diagnostics[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Message < b.Message
	})
	return resolved, diagnostics
}

func safePath(file string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.ContainsAny(file, "\\:\x00") {
		return "", fmt.Errorf("path must use repository-relative POSIX syntax")
	}
	if strings.ContainsAny(file, "*?[]{}") {
		return "", fmt.Errorf("glob patterns are not allowed")
	}
	if path.IsAbs(file) || path.Clean(file) != file || file == "." || strings.HasPrefix(file, "../") || strings.HasSuffix(file, "/") {
		return "", fmt.Errorf("path must be a normalized repository-relative file path")
	}
	for _, part := range strings.Split(file, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("path contains an unsafe component")
		}
	}
	return file, nil
}

func owner(graph *core.Graph, file string) *core.Area {
	bestLength := -1
	var best *core.Area
	for i := range graph.Project.Spec.Areas {
		area := &graph.Project.Spec.Areas[i]
		root, candidate := clean(area.Path), clean(file)
		if !within(candidate, root) {
			continue
		}
		if len(root) > bestLength || (len(root) == bestLength && (best == nil || area.Name < best.Name)) {
			best, bestLength = area, len(root)
		}
	}
	return best
}

func clean(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	return strings.Trim(path.Clean("/"+value), "/")
}

func within(candidate, root string) bool {
	return root != "" && (candidate == root || strings.HasPrefix(candidate, root+"/"))
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func isProjectConfig(file, projectPath string) bool {
	return (projectPath != "" && file == clean(projectPath)) || path.Base(file) == "markitect.yaml"
}

func typedCompanion(graph *core.Graph, file string) (*core.Resource, bool) {
	keys := make([]string, 0, len(graph.Resources))
	for key := range graph.Resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		resource := graph.Resources[key]
		if resource == nil || resource.Kind == "Project" || resource.Path == "" {
			continue
		}
		ext := path.Ext(resource.Path)
		companion := resource.Path + ".md"
		if ext != "" {
			companion = strings.TrimSuffix(resource.Path, ext) + ".md"
		}
		if clean(companion) == file {
			return resource, true
		}
	}
	return nil, false
}

func hasDescendant(files map[string][]byte, directory string) bool {
	prefix := directory + "/"
	for file := range files {
		if strings.HasPrefix(file, prefix) {
			return true
		}
	}
	return false
}
