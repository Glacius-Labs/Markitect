package apply

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

// roles reads the optional-per-package or required-project `profiles` and
// `instructions` selections of the Execute and Verify roles.
func (d *executionDecoder) roles(fields map[string]executionField, parent executionField, requireProfiles bool) (organization.RoleOverride, organization.RoleOverride) {
	execute, verify := organization.RoleOverride{}, organization.RoleOverride{}
	profiles, present := fields["profiles"]
	if requireProfiles && !present {
		profiles = d.required(fields, parent, "profiles")
	}
	if present || requireProfiles && profiles.node != nil {
		roles := d.object(profiles, "execute", "verify")
		for _, role := range []struct {
			name   string
			target *organization.RoleOverride
		}{{"execute", &execute}, {"verify", &verify}} {
			field, found := roles[role.name]
			if requireProfiles && !found {
				field = d.required(roles, profiles, role.name)
			}
			if field.node != nil {
				if value, ok := d.text(field, false); ok {
					role.target.Profile = &value
				}
			}
		}
	}
	if instructions, found := fields["instructions"]; found {
		roles := d.object(instructions, "execute", "verify")
		for _, role := range []struct {
			name   string
			target *organization.RoleOverride
		}{{"execute", &execute}, {"verify", &verify}} {
			if field, found := roles[role.name]; found {
				if value, ok := d.strings(field); ok {
					valid := true
					for _, ref := range value.Value {
						if ref == "" {
							valid = false
							d.issue(model.Failed, "execution.instruction_reference", field, "Instruction references cannot be empty.")
						}
					}
					if valid {
						role.target.Instructions = &value
					}
				}
			}
		}
	}
	return execute, verify
}

func (d *executionDecoder) packages(field executionField) *organization.Setting[[]organization.PackageOverride] {
	fields := d.object(field)
	if field.node.Kind != yaml.MappingNode {
		return nil
	}
	result := &organization.Setting[[]organization.PackageOverride]{Source: d.provenance(field)}
	seen := make(map[string]bool)
	for i := 0; i+1 < len(field.node.Content); i += 2 {
		name := field.node.Content[i].Value
		if seen[name] {
			continue
		}
		seen[name] = true
		entry, ok := fields[name]
		if !ok {
			continue
		}
		d.localPackage(entry, name)
		f := d.object(entry, "profiles", "instructions")
		value := organization.PackageOverride{Package: name}
		value.Execute, value.Verify = d.roles(f, entry, false)
		result.Value = append(result.Value, value)
	}
	return result
}

// layout decodes the optional persistent artifact layout and validates it
// against the model's Package inventory, so a declared layout is complete or
// the document is Failed.
func (d *executionDecoder) layout(field executionField, compiled model.Result) *ExecutionLayout {
	f := d.object(field, "roots", "exceptions")
	if field.node.Kind != yaml.MappingNode {
		return nil
	}
	before := len(d.result.diagnostics)
	result := &ExecutionLayout{}
	result.Roots, _ = d.directories(d.required(f, field, "roots"))
	if exceptions, found := f["exceptions"]; found {
		if exceptions.node.Kind != yaml.SequenceNode {
			d.issue(model.Failed, "execution.list", exceptions, "layout exceptions must be a list.")
		} else {
			for i, node := range exceptions.node.Content {
				entry := executionChild(exceptions, fmt.Sprint(i), node)
				e := d.object(entry, "path", "kind", "owner", "reason")
				var value ExecutionLayoutException
				if text, ok := d.text(d.required(e, entry, "path"), false); ok {
					path, err := snapshot.NewPath(text.Value)
					if err != nil {
						d.issue(model.Failed, "execution.layout_path", e["path"], err.Error())
					} else {
						value.Path = organization.Setting[snapshot.Path]{Value: path, Source: text.Source}
					}
				}
				if kind, found := e["kind"]; found {
					if text, ok := d.text(kind, false); ok {
						if text.Value != "file" && text.Value != "directory" {
							d.issue(model.Failed, "execution.layout_kind", kind, "an exception kind must be file or directory.")
						} else {
							value.Kind = text
						}
					}
				}
				value.Owner, _ = d.text(d.required(e, entry, "owner"), false)
				if value.Owner.Source.File != "" {
					d.localPackage(e["owner"], value.Owner.Value)
				}
				value.Reason, _ = d.text(d.required(e, entry, "reason"), false)
				result.Exceptions = append(result.Exceptions, value)
			}
		}
	}
	if len(d.result.diagnostics) != before {
		return result
	}
	var exceptions []realization.LayoutException
	for _, exception := range result.Exceptions {
		owner := realization.RootOwner()
		if exception.Owner.Value != organization.Root {
			owner, _ = realization.PackageOwner(exception.Owner.Value)
		}
		exceptions = append(exceptions, realization.LayoutException{Path: exception.Path.Value, File: exception.Kind.Value == "file", Owner: owner, Reason: exception.Reason.Value})
	}
	layout, err := realization.NewLayout(compiled, result.Roots.Value, exceptions)
	if err != nil {
		d.issue(model.Failed, "execution.layout", field, err.Error())
		return result
	}
	d.result.layout = layout
	return result
}
