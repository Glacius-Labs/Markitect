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
			writeDataField(&b, field, r.Data[field])
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
	writePolicyOutcomes(&b, r.APIVersion, r.GraphKey(), g, target)
	return []byte(strings.TrimRight(b.String(), "\n") + "\n"), nil
}

func writePolicyOutcomes(b *strings.Builder, apiVersion, subject string, g *core.Graph, target string) {
	if g == nil {
		return
	}
	results := make([]core.PolicyResult, 0)
	for _, result := range g.PolicyResults {
		if result.APIVersion != apiVersion || subject != "" && result.Subject != subject {
			continue
		}
		results = append(results, result)
	}
	if subject != "" && len(results) == 0 {
		return
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Constraint != results[j].Constraint {
			return results[i].Constraint < results[j].Constraint
		}
		return results[i].Subject < results[j].Subject
	})
	b.WriteString("## Policy outcomes\n\n")
	if len(results) == 0 {
		b.WriteString("No policy outcomes were evaluated for this domain.\n\n")
		return
	}
	b.WriteString("| Constraint | Subject | Status | Result | Waiver details |\n| --- | --- | --- | --- | --- |\n")
	for _, result := range results {
		subjectLabel := "All selected subjects"
		if result.Subject != "" {
			subjectLabel = policySubjectLink(result.Subject, g, target)
		}
		message := result.Message
		if message == "" {
			message = statusMessage(result.Status)
		}
		waiver := "—"
		if result.Status == core.PolicyWaived {
			parts := []string{
				"exception: " + result.ExceptionName,
				"rationale: " + result.Rationale,
				"owner: " + result.Owner,
				"decision: " + result.Decision,
			}
			if result.ExpiresOn != "" {
				parts = append(parts, "expiresOn: "+result.ExpiresOn)
			}
			if result.PolicyDate != "" {
				parts = append(parts, "policyDate: "+result.PolicyDate)
			}
			waiver = markdownCell(strings.Join(parts, "; "))
		}
		fmt.Fprintf(b, "| `%s` | %s | **%s** | %s | %s |\n",
			markdownCell(result.Constraint), subjectLabel, markdownText(strings.ToUpper(result.Status)), markdownCell(message), waiver)
	}
	b.WriteByte('\n')
}

func policySubjectLink(key string, g *core.Graph, target string) string {
	resource := g.Resources[key]
	if resource == nil {
		return markdownCode(key)
	}
	label := resource.Kind + ": " + resource.Metadata.Name
	if resource.Metadata.Namespace != "" {
		label = resource.Metadata.Namespace + "/" + label
	}
	if resource.Package != "" {
		return markdownCode(label + " (package " + resource.Package + ")")
	}
	if view, err := MarkdownViewPath(g, resource); err == nil {
		return "[" + markdownText(label) + "](" + relative(target, view) + ")"
	}
	return markdownCode(label)
}

func statusMessage(status string) string {
	switch status {
	case core.PolicyPassed:
		return "Constraint passed."
	case core.PolicyFailed:
		return "Constraint failed."
	case core.PolicyWaived:
		return "Violation waived."
	default:
		return "Policy evaluation status: " + status
	}
}

func markdownCell(value string) string {
	return markdownText(value)
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
	if g == nil || g.Registry == nil {
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
			assertion := constraint.Assert
			if assertion.Relation != "" && (assertion.Op == "count" || assertion.Op == "allowed-targets") {
				relation := domain.Relations[assertion.Relation]
				if len(relation.SourceKinds) > 0 && !containsKind(relation.SourceKinds, r.Kind) {
					continue
				}
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
			text := constraintAssertionText(assertion, domain)
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
		return fmt.Sprintf("`%s` relationships may target only kinds %s.", relationField(domain, assertion.Relation), displayValue(assertion.Values))
	case "same-target":
		return fmt.Sprintf("Paths `%s` and `%s` must each resolve exactly one target per step and end at the same canonical identity.", strings.Join(assertion.Left, " → "), strings.Join(assertion.Right, " → "))
	case "unique":
		return fmt.Sprintf("`%s` must be unique across the selected resources.", assertion.Field)
	case "count":
		what := "selected resources"
		if assertion.Relation != "" {
			what = "`" + relationField(domain, assertion.Relation) + "` relationship targets"
		}
		bounds := ""
		if assertion.Min != nil && assertion.Max != nil {
			bounds = fmt.Sprintf("between %d and %d", *assertion.Min, *assertion.Max)
		} else if assertion.Min != nil {
			bounds = fmt.Sprintf("at least %d", *assertion.Min)
		} else if assertion.Max != nil {
			bounds = fmt.Sprintf("at most %d", *assertion.Max)
		}
		if assertion.Scope == "resource" {
			return fmt.Sprintf("Each selected resource's count of %s must be %s.", what, bounds)
		}
		return fmt.Sprintf("The selection-wide count of %s must be %s.", what, bounds)
	default:
		return "The declared structured assertion must hold."
	}
}

func relationField(domain core.DomainDefinition, relation string) string {
	if definition, ok := domain.Relations[relation]; ok && definition.Field != "" {
		return definition.Field
	}
	return relation
}

func containsKind(kinds []string, kind string) bool {
	for _, candidate := range kinds {
		if candidate == kind {
			return true
		}
	}
	return false
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
		if len(v) == 0 {
			return "`[]`"
		}
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, displayValue(item))
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		if len(v) == 0 {
			return "`{}`"
		}
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

func writeDataField(b *strings.Builder, field string, value any) {
	fmt.Fprintf(b, "- **%s:** %s\n", markdownText(field), displayValue(value))
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
