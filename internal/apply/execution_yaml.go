package apply

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

type executionField struct {
	node    *yaml.Node
	pointer string
}
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
	name = strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
	return executionField{node: node, pointer: parent.pointer + "/" + name}
}

func (d *executionDecoder) restricted(field executionField) {
	node := field.node
	if node.Kind == yaml.AliasNode || node.Alias != nil || node.Anchor != "" || node.Tag == "!!null" || node.Tag != "" && !strings.HasPrefix(node.Tag, "!!") {
		d.issue(model.Failed, "execution.yaml_feature", field, "Execution YAML cannot use aliases, anchors, nulls or custom tags.")
		return
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			child := executionChild(field, node.Content[i].Value, node.Content[i])
			d.restricted(child)
			child.node = node.Content[i+1]
			d.restricted(child)
		}
	} else {
		for i, child := range node.Content {
			d.restricted(executionChild(field, fmt.Sprint(i), child))
		}
	}
}

func (d *executionDecoder) object(field executionField, allowed ...string) map[string]executionField {
	result := make(map[string]executionField)
	if field.node == nil || field.node.Kind != yaml.MappingNode {
		d.issue(model.Failed, "execution.mapping", field, "Expected an execution mapping.")
		return result
	}
	for i := 0; i+1 < len(field.node.Content); i += 2 {
		key, value := field.node.Content[i], field.node.Content[i+1]
		child := executionChild(field, key.Value, value)
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			d.issue(model.Failed, "execution.key", child, "Execution keys must be strings.")
			continue
		}
		if _, found := result[key.Value]; found {
			d.issue(model.Failed, "execution.duplicate_key", child, fmt.Sprintf("Duplicate execution key %q.", key.Value))
			continue
		}
		result[key.Value] = child
		if allowed != nil {
			known := false
			for _, name := range allowed {
				known = known || key.Value == name
			}
			if !known {
				d.issue(model.Failed, "execution.unknown_key", child, fmt.Sprintf("Unknown execution key %q.", key.Value))
			}
		}
	}
	return result
}

func (d *executionDecoder) required(fields map[string]executionField, parent executionField, name string) executionField {
	if field, found := fields[name]; found {
		return field
	}
	field := executionChild(parent, name, nil)
	d.issue(model.Failed, "execution.field_missing", field, fmt.Sprintf("Required execution field %q is missing.", name))
	return field
}

func (d *executionDecoder) text(field executionField, empty bool) (organization.Setting[string], bool) {
	if field.node == nil {
		return organization.Setting[string]{}, false
	}
	if field.node.Kind != yaml.ScalarNode || field.node.Tag != "!!str" || !empty && strings.TrimSpace(field.node.Value) == "" {
		d.issue(model.Failed, "execution.string", field, "Expected a literal string (nonempty unless explicitly allowed).")
		return organization.Setting[string]{}, false
	}
	return organization.Setting[string]{Value: field.node.Value, Source: d.provenance(field)}, true
}

func (d *executionDecoder) integer(field executionField) (organization.Setting[int64], bool) {
	if field.node == nil {
		return organization.Setting[int64]{}, false
	}
	// YAML resolves decimal integers beyond its own integer range as floats.
	// Recognize their literal syntax without accepting floats or quoted numbers
	// as integer caps. The original source remains the value's authority.
	if field.node.Kind == yaml.ScalarNode && (field.node.Tag == "!!int" || field.node.Style == 0 && executionDecimal.MatchString(field.node.Value)) {
		base := 0
		if field.node.Tag != "!!int" {
			base = 10
		}
		if value, ok := new(big.Int).SetString(strings.ReplaceAll(field.node.Value, "_", ""), base); ok {
			if value.Sign() <= 0 {
				d.issue(model.Failed, "execution.positive", field, "Declared caps must be positive.")
				return organization.Setting[int64]{}, false
			}
			if !value.IsInt64() {
				d.issue(model.NotCompleted, "execution.integer_range", field, "The declared integer exceeds this decoder's integer representation.")
				return organization.Setting[int64]{}, false
			}
		}
	}
	if field.node.Kind != yaml.ScalarNode || field.node.Tag != "!!int" {
		d.issue(model.Failed, "execution.integer", field, "Expected a positive integer cap.")
		return organization.Setting[int64]{}, false
	}
	var value int64
	if err := field.node.Decode(&value); err != nil {
		d.issue(model.Failed, "execution.integer", field, "Expected a positive integer cap.")
		return organization.Setting[int64]{}, false
	}
	if value <= 0 {
		d.issue(model.Failed, "execution.positive", field, "Declared caps must be positive.")
		return organization.Setting[int64]{}, false
	}
	return organization.Setting[int64]{Value: value, Source: d.provenance(field)}, true
}

func (d *executionDecoder) count(field executionField) (organization.Setting[int], bool) {
	value, ok := d.integer(field)
	if !ok {
		return organization.Setting[int]{}, false
	}
	if int64(int(value.Value)) != value.Value {
		d.issue(model.NotCompleted, "execution.integer_range", field, "The declared integer exceeds the producer's int representation.")
		return organization.Setting[int]{}, false
	}
	return organization.Setting[int]{Value: int(value.Value), Source: value.Source}, true
}

var executionDecimal = regexp.MustCompile(`^[+-]?[0-9](?:_?[0-9])*$`)

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
		d.issue(model.Failed, "execution.list", field, "Expected a creation-root list.")
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
