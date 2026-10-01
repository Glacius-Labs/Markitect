package format

import (
	"path"
	"strings"

	"go.yaml.in/yaml/v3"
)

func validateDocumentation(file string, n *yaml.Node) error {
	if err := requireMapping(file, n, "documentation"); err != nil {
		return err
	}
	if err := checkKeys(file, n, set("roots")); err != nil {
		return err
	}
	if err := requireFields(file, n, "roots"); err != nil {
		return err
	}
	roots := child(n, "roots")
	if err := requireSequence(file, roots, "documentation roots"); err != nil {
		return err
	}
	if len(roots.Content) == 0 {
		return diagnostic(file, roots.Line, "documentation roots must not be empty")
	}
	var seen []string
	for _, item := range roots.Content {
		if err := checkScalar(file, item, "string"); err != nil {
			return err
		}
		value := item.Value
		if value == "" || path.IsAbs(value) || path.Clean(value) != value || value == "." || strings.HasPrefix(value, "../") || strings.ContainsAny(value, "\\:\x00*?[]{}") || strings.HasSuffix(value, "/") {
			return diagnostic(file, item.Line, "documentation root must be a normalized repository-relative POSIX path")
		}
		for _, prior := range seen {
			if strings.EqualFold(value, prior) || strings.HasPrefix(strings.ToLower(value), strings.ToLower(prior)+"/") || strings.HasPrefix(strings.ToLower(prior), strings.ToLower(value)+"/") {
				return diagnostic(file, item.Line, "documentation roots %q and %q overlap or collide", prior, value)
			}
		}
		seen = append(seen, value)
	}
	return nil
}
