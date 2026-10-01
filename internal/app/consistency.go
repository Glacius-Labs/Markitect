package app

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

// CheckConsistency compares only explicitly declared functional assertions.
// The exact quoted source is checked; its meaning remains a human decision.
func CheckConsistency(p *Project) []core.Diagnostic {
	if p.Graph.Project == nil || p.Graph.Project.Spec.Consistency == nil {
		return nil
	}
	functional := map[string]bool{}
	for _, predicate := range p.Graph.Project.Spec.Consistency.FunctionalPredicates {
		functional[predicate] = true
	}
	type claim struct {
		owner, value, source string
		line                 int
	}
	seen := map[string]claim{}
	var findings []core.Diagnostic
	keys := make([]string, 0, len(p.Graph.Resources))
	for key := range p.Graph.Resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		r := p.Graph.Resources[key]
		if r.Package != "" {
			continue
		}
		for _, a := range r.Spec.Assertions {
			if !functional[a.Predicate] {
				continue
			}
			declared := false
			for _, file := range r.Spec.Files {
				if file == a.Source {
					declared = true
					break
				}
			}
			if !declared {
				findings = append(findings, core.Diagnostic{Code: "consistency.input", Path: r.Path, Line: r.Line, Message: fmt.Sprintf("assertion source %q must be declared in spec.files for snapshot context and impact", a.Source)})
				continue
			}
			if !safeAssertionPath(a.Source) {
				findings = append(findings, core.Diagnostic{Code: "consistency.source", Path: r.Path, Line: r.Line, Message: fmt.Sprintf("assertion source %q is not a normalized repository path", a.Source)})
				continue
			}
			data, ok := p.Snapshot.Files[a.Source]
			if !ok {
				findings = append(findings, core.Diagnostic{Code: "consistency.source", Path: r.Path, Line: r.Line, Message: fmt.Sprintf("assertion source %q is missing", a.Source)})
				continue
			}
			content := string(normalize(data))
			if strings.Count(content, a.Quote) != 1 {
				findings = append(findings, core.Diagnostic{Code: "consistency.quote", Path: a.Source, Message: fmt.Sprintf("quote for %s must occur exactly once in its source", r.Key())})
				continue
			}
			line := strings.Count(content[:strings.Index(content, a.Quote)], "\n") + 1
			current := claim{owner: r.Key(), value: a.Value, source: a.Source, line: line}
			identity := a.Subject + "\x00" + a.Predicate
			if prior, exists := seen[identity]; exists && prior.value != current.value {
				findings = append(findings, core.Diagnostic{Code: "consistency.conflict", Path: a.Source, Line: line, Message: fmt.Sprintf("%s/%s=%q conflicts with %s:%d (%s)=%q; owners %s and %s", a.Subject, a.Predicate, a.Value, prior.source, prior.line, prior.owner, prior.value, prior.owner, current.owner)})
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
