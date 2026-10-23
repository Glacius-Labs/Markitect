package project

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

type executionField = configField
type executionDecoder struct {
	result      *executionDecoding
	source      SourceFile
	topology    organization.Topology
	definitions map[model.Identity]bool
}

func (d *executionDecoder) provenance(field executionField) organization.SettingSource {
	return organization.SettingSource{File: d.source.RepositoryPath.String(), ContentID: d.source.File.ContentID(), Field: field.pointer}
}

func (d *executionDecoder) issue(status model.Status, code string, field executionField, message string) {
	var position *model.Position
	if field.node != nil && field.node.Line > 0 {
		position = &model.Position{Line: field.node.Line, Column: field.node.Column}
	}
	d.result.diagnostics = append(d.result.diagnostics, ExecutionDiagnostic{Code: code, Source: d.provenance(field), Position: position, Message: message})
	if status == model.Failed || d.result.status != model.Failed {
		d.result.status = status
	}
}

func executionChild(parent executionField, name string, node *yaml.Node) executionField {
	return configChild(parent, name, node)
}

func (d *executionDecoder) yaml() configYAML {
	return configYAML{name: "execution", title: "Execution", issue: func(code string, field configField, message string) {
		d.issue(model.Failed, "execution."+code, field, message)
	}}
}

func (d *executionDecoder) restricted(field executionField) {
	d.yaml().restricted(field)
}

func (d *executionDecoder) object(field executionField, allowed ...string) map[string]executionField {
	return d.yaml().object(field, allowed...)
}

func (d *executionDecoder) required(fields map[string]executionField, parent executionField, name string) executionField {
	return d.yaml().required(fields, parent, name)
}

func (d *executionDecoder) text(field executionField, empty bool) (organization.Setting[string], bool) {
	value, ok := d.yaml().text(field, empty)
	if !ok {
		return organization.Setting[string]{}, false
	}
	return organization.Setting[string]{Value: value, Source: d.provenance(field)}, true
}

func (d *executionDecoder) strings(field executionField) (organization.Setting[[]string], bool) {
	if field.node == nil {
		return organization.Setting[[]string]{}, false
	}
	if field.node.Kind != yaml.SequenceNode {
		d.issue(model.Failed, "execution.list", field, "Expected an explicit list.")
		return organization.Setting[[]string]{}, false
	}
	var values []string
	valid := true
	for i, node := range field.node.Content {
		value, ok := d.text(executionChild(field, fmt.Sprint(i), node), true)
		if ok {
			values = append(values, value.Value)
		}
		valid = valid && ok
	}
	if !valid {
		return organization.Setting[[]string]{}, false
	}
	return organization.Setting[[]string]{Value: values, Source: d.provenance(field)}, true
}

func (d *executionDecoder) directory(field executionField) (organization.Setting[snapshot.DirectoryBoundary], bool) {
	text, ok := d.text(field, false)
	if !ok {
		return organization.Setting[snapshot.DirectoryBoundary]{}, false
	}
	if text.Value == "." {
		return organization.Setting[snapshot.DirectoryBoundary]{Value: snapshot.RootDirectory(), Source: text.Source}, true
	}
	path, err := snapshot.NewPath(text.Value)
	if err != nil {
		d.issue(model.Failed, "execution.directory", field, err.Error())
		return organization.Setting[snapshot.DirectoryBoundary]{}, false
	}
	boundary, _ := snapshot.DirectoryAt(path)
	return organization.Setting[snapshot.DirectoryBoundary]{Value: boundary, Source: text.Source}, true
}

func (d *executionDecoder) directories(field executionField) (organization.Setting[[]snapshot.DirectoryBoundary], bool) {
	if field.node == nil {
		return organization.Setting[[]snapshot.DirectoryBoundary]{}, false
	}
	if field.node.Kind != yaml.SequenceNode {
		d.issue(model.Failed, "execution.list", field, "Expected a directory list.")
		return organization.Setting[[]snapshot.DirectoryBoundary]{}, false
	}
	var values []snapshot.DirectoryBoundary
	valid := true
	for i, node := range field.node.Content {
		value, ok := d.directory(executionChild(field, fmt.Sprint(i), node))
		if ok {
			values = append(values, value.Value)
		}
		valid = valid && ok
	}
	if !valid {
		return organization.Setting[[]snapshot.DirectoryBoundary]{}, false
	}
	return organization.Setting[[]snapshot.DirectoryBoundary]{Value: values, Source: d.provenance(field)}, true
}
