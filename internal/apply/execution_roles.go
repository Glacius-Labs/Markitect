package apply

import (
	"fmt"
	"regexp"
	"strings"

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

// ownership decodes the optional persistent ownership areas and validates them
// against the model's Package inventory, so declared ownership is complete or
// the document is Failed.
func (d *executionDecoder) ownership(field executionField, checked model.Result) *organization.Setting[[]ExecutionArea] {
	if field.node == nil {
		return nil
	}
	if field.node.Kind != yaml.SequenceNode {
		d.issue(model.Failed, "execution.list", field, "ownership must be a list of areas.")
		return nil
	}
	before := len(d.result.diagnostics)
	result := &organization.Setting[[]ExecutionArea]{Source: d.provenance(field)}
	var areas []realization.Area
	for i, node := range field.node.Content {
		entry := executionChild(field, fmt.Sprint(i), node)
		e := d.object(entry, "path", "owner", "reason")
		var value ExecutionArea
		var area realization.Area
		if text, ok := d.text(d.required(e, entry, "path"), false); ok {
			value.Path = text
			selector, err := areaSelector(text.Value)
			if err != nil {
				d.issue(model.Failed, "execution.ownership_path", e["path"], err.Error())
			}
			area = selector
		}
		value.Owner, _ = d.text(d.required(e, entry, "owner"), false)
		if value.Owner.Source.File != "" {
			d.localPackage(e["owner"], value.Owner.Value)
			area.Owner = realization.RootOwner()
			if value.Owner.Value != organization.Root {
				area.Owner, _ = realization.PackageOwner(value.Owner.Value)
			}
		}
		if reason, found := e["reason"]; found {
			value.Reason, _ = d.text(reason, false)
			area.Reason = value.Reason.Value
		}
		result.Value = append(result.Value, value)
		areas = append(areas, area)
	}
	if len(d.result.diagnostics) != before {
		return result
	}
	layout, err := realization.NewLayout(checked, areas)
	if err != nil {
		d.issue(model.Failed, "execution.ownership", field, err.Error())
		return result
	}
	d.result.layout = layout
	return result
}

// areaSelector reads an ownership selector: an exact repository-relative file,
// a directory ending in "/", or "./" for the whole repository.
func areaSelector(text string) (realization.Area, error) {
	if text == "./" {
		return realization.Area{Directory: true}, nil
	}
	name, directory := strings.CutSuffix(text, "/")
	if _, err := snapshot.NewPath(name); err != nil {
		return realization.Area{}, fmt.Errorf("%q is not an exact repository-relative file, a directory ending in \"/\" or \"./\": %v", text, err)
	}
	return realization.Area{Path: name, Directory: directory}, nil
}

// team decodes the optional local team template: the helper roles and the
// bound on active helpers per Manager.
func (d *executionDecoder) team(field executionField) *ExecutionTeam {
	f := d.object(field, "maxHelpers", "helpers")
	if field.node == nil || field.node.Kind != yaml.MappingNode {
		return nil
	}
	result := &ExecutionTeam{}
	result.MaxHelpers, _ = d.count(d.required(f, field, "maxHelpers"))
	helpers := d.required(f, field, "helpers")
	if helpers.node == nil {
		return result
	}
	if helpers.node.Kind != yaml.SequenceNode {
		d.issue(model.Failed, "execution.list", helpers, "team helpers must be a list, [] when empty.")
		return result
	}
	seen := map[string]bool{}
	for i, node := range helpers.node.Content {
		entry := executionChild(helpers, fmt.Sprint(i), node)
		e := d.object(entry, "role", "profile", "writes", "instructions")
		var value ExecutionHelper
		if role, ok := d.text(d.required(e, entry, "role"), false); ok {
			if !helperRoleName.MatchString(role.Value) || role.Value == "manager" || role.Value == "reviewer" || seen[role.Value] {
				d.issue(model.Failed, "execution.helper_role", e["role"], "A helper role is a unique lower-case name other than manager and reviewer.")
			}
			seen[role.Value] = true
			value.Role = role
		}
		if profile, found := e["profile"]; found {
			if text, ok := d.text(profile, false); ok {
				value.Profile = &text
			}
		}
		value.Writes, _ = d.boolean(d.required(e, entry, "writes"))
		if instructions, found := e["instructions"]; found {
			if list, ok := d.strings(instructions); ok {
				value.Instructions = &list
			}
		}
		result.Helpers = append(result.Helpers, value)
	}
	return result
}

var helperRoleName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)

func (d *executionDecoder) boolean(field executionField) (organization.Setting[bool], bool) {
	if field.node == nil {
		return organization.Setting[bool]{}, false
	}
	if field.node.Kind != yaml.ScalarNode || field.node.Tag != "!!bool" {
		d.issue(model.Failed, "execution.boolean", field, "Expected true or false.")
		return organization.Setting[bool]{}, false
	}
	var value bool
	if err := field.node.Decode(&value); err != nil {
		d.issue(model.Failed, "execution.boolean", field, "Expected true or false.")
		return organization.Setting[bool]{}, false
	}
	return organization.Setting[bool]{Value: value, Source: d.provenance(field)}, true
}
