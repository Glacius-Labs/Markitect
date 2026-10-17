package apply

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"go.yaml.in/yaml/v3"
)

func (d *executionDecoder) scope(field executionField) *organization.Setting[[]organization.Setting[organization.ScopeRule]] {
	if field.node.Kind != yaml.SequenceNode {
		d.issue(model.Failed, "execution.list", field, "scope must be a list.")
		return nil
	}
	result := &organization.Setting[[]organization.Setting[organization.ScopeRule]]{Source: d.provenance(field)}
	for i, node := range field.node.Content {
		entry := executionChild(field, fmt.Sprint(i), node)
		f := d.object(entry, "kind", "property")
		kind, kindOK := d.identity(d.required(f, entry, "kind"))
		property, propertyOK := d.identity(d.required(f, entry, "property"))
		if kindOK && propertyOK {
			result.Value = append(result.Value, organization.Setting[organization.ScopeRule]{Value: organization.ScopeRule{Kind: kind.Value, Property: property.Value}, Source: d.provenance(entry)})
		}
	}
	return result
}

func (d *executionDecoder) checks(field executionField) *organization.Setting[[]ExecutionCheck] {
	if field.node.Kind != yaml.SequenceNode {
		d.issue(model.Failed, "execution.list", field, "checks must be a list.")
		return nil
	}
	result := &organization.Setting[[]ExecutionCheck]{Source: d.provenance(field)}
	ids := make(map[string]bool)
	for i, node := range field.node.Content {
		entry := executionChild(field, fmt.Sprint(i), node)
		f := d.object(entry, "id", "profile", "directory", "args", "requiredBy", "timeout")
		value := ExecutionCheck{}
		value.ID, _ = d.text(d.required(f, entry, "id"), false)
		value.Profile, _ = d.text(d.required(f, entry, "profile"), false)
		value.Directory, _ = d.directory(d.required(f, entry, "directory"))
		value.Args, _ = d.strings(d.required(f, entry, "args"))
		value.Timeout, _ = d.duration(d.required(f, entry, "timeout"))
		if field, found := f["requiredBy"]; found {
			if setting, ok := d.identity(field); ok {
				value.RequiredBy = &setting
			}
		}
		if value.ID.Source.File != "" {
			if ids[value.ID.Value] {
				d.issue(model.Failed, "execution.duplicate_check", f["id"], "Check ids must be unique.")
			}
			ids[value.ID.Value] = true
		}
		result.Value = append(result.Value, value)
	}
	return result
}
