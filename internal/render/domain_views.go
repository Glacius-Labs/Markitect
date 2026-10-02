package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func renderDomainCompanion(r *core.Resource, target string, g *core.Graph) ([]byte, error) {
	var b strings.Builder
	b.WriteString("<!-- " + Marker + "; source: " + r.Path + " -->\n")
	b.WriteString("# " + r.Kind + ": " + r.Metadata.Name + "\n\n")
	b.WriteString("**API version:** `" + r.APIVersion + "`\n\n")
	if description, ok := r.Data["description"].(string); ok && strings.TrimSpace(description) != "" {
		b.WriteString(description + "\n\n")
	}
	fields := make([]string, 0, len(r.Data))
	for field := range r.Data {
		if field != "description" {
			fields = append(fields, field)
		}
	}
	sort.Strings(fields)
	if len(fields) > 0 {
		b.WriteString("## Declared facts\n\n")
		for _, field := range fields {
			b.WriteString("- **" + field + ":** " + displayValue(r.Data[field]) + "\n")
		}
		b.WriteByte('\n')
	}
	if links := genericRelationLinks(r, target, g); len(links) > 0 {
		b.WriteString("## Relationships\n\n")
		for _, link := range links {
			b.WriteString(link + "\n")
		}
		b.WriteByte('\n')
	}
	if constraints := applicableConstraints(r, g); len(constraints) > 0 {
		b.WriteString("## Applicable constraints\n\n")
		for _, constraint := range constraints {
			b.WriteString("- **" + constraint.Name + ":** " + constraint.Text + "\n")
		}
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

type domainViewConstraint struct{ Name, Text string }

func genericRelationLinks(r *core.Resource, target string, g *core.Graph) []string {
	var result []string
	for _, relationship := range g.Relationships {
		if relationship.From != r.GraphKey() {
			continue
		}
		destination := g.Resources[relationship.To]
		if destination == nil {
			continue
		}
		label := relationship.Relation + " " + destination.Kind + ": " + destination.Metadata.Name
		if destination.Package != "" {
			result = append(result, "- `"+relationship.Relation+"` → "+label+" (package `"+destination.Package+"`)")
			continue
		}
		view := destination.Path
		if markdownPath, err := MarkdownViewPath(g, destination); err == nil {
			view = markdownPath
		}
		result = append(result, "- [`"+relationship.Relation+"` → "+label+"]("+relative(target, view)+")")
	}
	sort.Strings(result)
	return uniqueStrings(result)
}

func applicableConstraints(r *core.Resource, g *core.Graph) []domainViewConstraint {
	if g.Registry == nil {
		return nil
	}
	var result []domainViewConstraint
	for _, domain := range g.Registry.Domains() {
		if domain.APIVersion != r.APIVersion {
			continue
		}
		for _, constraint := range domain.Constraints {
			if constraint.Select.Kind != "" && constraint.Select.Kind != r.Kind {
				continue
			}
			matches := true
			for key, value := range constraint.Select.Labels {
				actual, exists := r.Metadata.Labels[key]
				if !exists || actual != value {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
			text := constraintAssertionText(constraint.Assert, domain)
			if constraint.Description != "" {
				text += " Explanation: " + constraint.Description
			}
			result = append(result, domainViewConstraint{Name: constraint.Name, Text: text})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func constraintAssertionText(assertion core.ConstraintAssertion, domain core.DomainDefinition) string {
	switch assertion.Op {
	case "present":
		return fmt.Sprintf("`%s` must be present.", assertion.Field)
	case "equal":
		return fmt.Sprintf("`%s` must equal %s.", assertion.Field, displayValue(assertion.Value))
	case "allowed":
		return fmt.Sprintf("`%s` must be one of %s.", assertion.Field, displayValue(assertion.Values))
	case "allowed-targets":
		return fmt.Sprintf("`%s` relationships may target only kinds %s.", assertion.Relation, displayValue(assertion.Values))
	case "unique":
		return fmt.Sprintf("`%s` must be unique across the selected resources.", assertion.Field)
	case "count":
		what := "selected resources"
		if assertion.Relation != "" {
			what = "`" + assertion.Relation + "` relationships"
		}
		bounds := ""
		if assertion.Min != nil && assertion.Max != nil {
			bounds = fmt.Sprintf("between %d and %d", *assertion.Min, *assertion.Max)
		} else if assertion.Min != nil {
			bounds = fmt.Sprintf("at least %d", *assertion.Min)
		} else if assertion.Max != nil {
			bounds = fmt.Sprintf("at most %d", *assertion.Max)
		}
		return fmt.Sprintf("The count of %s must be %s.", what, bounds)
	default:
		return "The declared structured assertion must hold."
	}
}

func displayValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "`null`"
	case string:
		return "`" + strings.ReplaceAll(v, "`", "\\`") + "`"
	case bool, int, int64, float64:
		return fmt.Sprintf("`%v`", v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, displayValue(item))
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, "`"+key+"`: "+displayValue(v[key]))
		}
		return strings.Join(parts, "; ")
	default:
		return "`" + fmt.Sprint(v) + "`"
	}
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
