package apply

import (
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"go.yaml.in/yaml/v3"
)

func (d *executionDecoder) roles(fields map[string]executionField, parent executionField, requireProfiles bool) (organization.RoleOverride, organization.RoleOverride) {
	manager, reviewer := organization.RoleOverride{}, organization.RoleOverride{}
	profiles, present := fields["profiles"]
	if requireProfiles && !present {
		profiles = d.required(fields, parent, "profiles")
	}
	if present || requireProfiles && profiles.node != nil {
		roles := d.object(profiles, "manager", "reviewer")
		for _, role := range []struct {
			name   string
			target *organization.RoleOverride
		}{{"manager", &manager}, {"reviewer", &reviewer}} {
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
		roles := d.object(instructions, "manager", "reviewer")
		for _, role := range []struct {
			name   string
			target *organization.RoleOverride
		}{{"manager", &manager}, {"reviewer", &reviewer}} {
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
	return manager, reviewer
}

func (d *executionDecoder) defaults(field executionField) ExecutionDefaults {
	r := ExecutionDefaults{}
	if field.node == nil {
		return r
	}
	f := d.object(field, "profiles", "instructions", "stepTimeout", "reviewRounds", "creationRoots", "batchNoChangeReviews")
	r.Manager, r.Reviewer = d.roles(f, field, true)
	r.StepTimeout, _ = d.duration(d.required(f, field, "stepTimeout"))
	if selected, found := f["reviewRounds"]; found {
		if value, ok := d.count(selected); ok {
			r.ReviewRounds = &value
		}
	}
	r.CreationRoots, _ = d.directories(d.required(f, field, "creationRoots"))
	if selected, found := f["batchNoChangeReviews"]; found {
		if selected.node.Kind != yaml.ScalarNode || selected.node.Tag != "!!bool" {
			d.issue(model.Failed, "execution.boolean", selected, "batchNoChangeReviews must be a boolean.")
		} else {
			var value bool
			if err := selected.node.Decode(&value); err != nil {
				d.issue(model.Failed, "execution.boolean", selected, err.Error())
			} else {
				r.BatchNoChangeReviews = &organization.Setting[bool]{Value: value, Source: d.provenance(selected)}
			}
		}
	}
	return r
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
		f := d.object(entry, "profiles", "instructions", "stepTimeout", "reviewRounds", "creationRoots")
		value := organization.PackageOverride{Package: name}
		value.Manager, value.Reviewer = d.roles(f, entry, false)
		if field, found := f["stepTimeout"]; found {
			if setting, ok := d.duration(field); ok {
				value.StepTimeout = &setting
			}
		}
		if field, found := f["reviewRounds"]; found {
			if setting, ok := d.count(field); ok {
				value.ReviewRounds = &setting
			}
		}
		if field, found := f["creationRoots"]; found {
			if setting, ok := d.directories(field); ok {
				value.CreationRoots = &setting
			}
		}
		result.Value = append(result.Value, value)
	}
	return result
}
