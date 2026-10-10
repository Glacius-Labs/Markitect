package markdown

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

// Validate runs the pure Markdown-owned checks against normalized model data
// and the exact fixed-snapshot bytes supplied by Host.
func Validate(model core.SemanticModel, config Config, files map[string][]byte) []core.Diagnostic {
	findings := validateConsistency(model, config.FunctionalPredicates, files)
	findings = append(findings, ValidateDocumentationRouters(config.DocumentationRoots, files)...)
	return findings
}

// validateConsistency compares only explicitly declared functional assertions.
// It locates the exact quoted source; its meaning remains a human decision.
func validateConsistency(model core.SemanticModel, predicates []string, files map[string][]byte) []core.Diagnostic {
	if len(predicates) == 0 {
		return nil
	}
	functional := make(map[string]bool, len(predicates))
	for _, predicate := range predicates {
		functional[predicate] = true
	}
	type claim struct {
		owner, value, source string
		line                 int
	}
	seen := map[string]claim{}
	resources := append([]core.ModelResource(nil), model.Resources...)
	sort.Slice(resources, func(i, j int) bool { return resources[i].Identity.Key < resources[j].Identity.Key })
	var findings []core.Diagnostic
	for _, r := range resources {
		if r.Identity.Package != "" {
			continue
		}
		assertions, _ := r.Data["assertions"].([]any)
		listedFiles, _ := r.Data["files"].([]any)
		for _, raw := range assertions {
			a, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			subject, _ := a["subject"].(string)
			predicate, _ := a["predicate"].(string)
			value, _ := a["value"].(string)
			source, _ := a["source"].(string)
			quote, _ := a["quote"].(string)
			if !functional[predicate] {
				continue
			}
			declared := false
			for _, rawPath := range listedFiles {
				if p, ok := rawPath.(string); ok && p == source {
					declared = true
					break
				}
			}
			if !declared {
				findings = append(findings, core.Diagnostic{Code: "consistency.input", Path: r.Source.Path, Line: r.Source.Line, Message: fmt.Sprintf("assertion source %q must be declared in spec.files for snapshot context and impact", source)})
				continue
			}
			if !safeAssertionPath(source) {
				findings = append(findings, core.Diagnostic{Code: "consistency.source", Path: r.Source.Path, Line: r.Source.Line, Message: fmt.Sprintf("assertion source %q is not a normalized repository path", source)})
				continue
			}
			data, exists := files[source]
			if !exists {
				findings = append(findings, core.Diagnostic{Code: "consistency.source", Path: r.Source.Path, Line: r.Source.Line, Message: fmt.Sprintf("assertion source %q is missing", source)})
				continue
			}
			content := string(normalize(data))
			if strings.Count(content, quote) != 1 {
				findings = append(findings, core.Diagnostic{Code: "consistency.quote", Path: source, Message: fmt.Sprintf("quote for %s must occur exactly once in its source", r.Identity.Key)})
				continue
			}
			line := strings.Count(content[:strings.Index(content, quote)], "\n") + 1
			current := claim{owner: r.Identity.Key, value: value, source: source, line: line}
			identity := subject + "\x00" + predicate
			if prior, exists := seen[identity]; exists && prior.value != current.value {
				findings = append(findings, core.Diagnostic{Code: "consistency.conflict", Path: source, Line: line, Message: fmt.Sprintf("%s/%s=%q conflicts with %s:%d (%s)=%q; owners %s and %s", subject, predicate, value, prior.source, prior.line, prior.owner, prior.value, prior.owner, current.owner)})
			} else if !exists {
				seen[identity] = current
			}
		}
	}
	return findings
}

func safeAssertionPath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\:\x00*?[]{}") || path.IsAbs(value) || path.Clean(value) != value || value == "." || strings.HasPrefix(value, "../") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func normalize(data []byte) []byte { return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")) }
