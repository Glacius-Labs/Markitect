package render

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func add(outputs map[string][]byte, name string, data []byte) error {
	if err := validRepoPath(name); err != nil {
		return fmt.Errorf("invalid output path %q: %w", name, err)
	}
	if _, exists := outputs[name]; exists {
		return fmt.Errorf("duplicate generated output %q", name)
	}
	outputs[name] = data
	return nil
}

func checkCollisions(outputs map[string][]byte, sources map[string]*core.Resource) (map[string][]byte, error) {
	seen := map[string]string{}
	for source := range sources {
		seen[foldPath(source)] = "source " + source
	}
	paths := make([]string, 0, len(outputs))
	for p := range outputs {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		key := foldPath(p)
		if prior, ok := seen[key]; ok {
			return nil, fmt.Errorf("output path collision: %s conflicts with %q", prior, p)
		}
		seen[key] = "output " + p
	}
	return outputs, nil
}

func validRepoPath(p string) error {
	if p == "" || strings.ContainsAny(p, "\\:\x00") || strings.HasPrefix(p, "/") || path.Clean(p) != p || p == "." || strings.HasPrefix(p, "../") {
		return fmt.Errorf("path must be repository-relative and normalized")
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("path contains unsafe component")
		}
	}
	return nil
}

func foldPath(p string) string { return strings.ToLower(p) }
