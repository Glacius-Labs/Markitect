package project

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

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

// Retain declared references even for an invalid Package, so accounting still
// protects every named instruction and reports its missing acquisition.
func (d *executionDecoder) instructions(field executionField) *organization.Setting[[]ManagerInstructions] {
	fields := d.object(field)
	result := &organization.Setting[[]ManagerInstructions]{Source: d.provenance(field)}
	if field.node == nil || field.node.Kind != yaml.MappingNode {
		return result
	}
	seen := map[string]bool{}
	for i := 0; i+1 < len(field.node.Content); i += 2 {
		name := field.node.Content[i].Value
		if seen[name] {
			continue
		}
		seen[name] = true
		f, ok := fields[name]
		if !ok {
			continue
		}
		d.localPackage(f, name)
		files, ok := d.strings(f)
		if !ok {
			continue
		}
		for _, ref := range files.Value {
			if ref == "" {
				d.issue(model.Failed, "execution.instruction_reference", f, "Instruction references cannot be empty.")
			}
		}
		result.Value = append(result.Value, ManagerInstructions{Package: organization.Setting[string]{Value: name, Source: d.provenance(f)}, Files: files})
	}
	return result
}
