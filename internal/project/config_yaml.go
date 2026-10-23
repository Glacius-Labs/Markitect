package project

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Private node checks shared by the execution and exact-file ignore consumers.
// Each consumer owns its document, source binding and diagnostic namespace.
type configField struct {
	node    *yaml.Node
	pointer string
}

type configYAML struct {
	name, title string
	issue       func(code string, field configField, message string)
}

func configChild(parent configField, name string, node *yaml.Node) configField {
	name = strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
	return configField{node: node, pointer: parent.pointer + "/" + name}
}

func (d configYAML) restricted(field configField) {
	node := field.node
	if node.Kind == yaml.AliasNode || node.Alias != nil || node.Anchor != "" || node.Tag == "!!null" || node.Tag != "" && !strings.HasPrefix(node.Tag, "!!") {
		d.issue("yaml_feature", field, d.title+" YAML cannot use aliases, anchors, nulls or custom tags.")
		return
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			child := configChild(field, node.Content[i].Value, node.Content[i])
			d.restricted(child)
			child.node = node.Content[i+1]
			d.restricted(child)
		}
	} else {
		for i, child := range node.Content {
			d.restricted(configChild(field, fmt.Sprint(i), child))
		}
	}
}

func (d configYAML) object(field configField, allowed ...string) map[string]configField {
	result := make(map[string]configField)
	if field.node == nil || field.node.Kind != yaml.MappingNode {
		d.issue("mapping", field, "Expected an "+d.name+" mapping.")
		return result
	}
	for i := 0; i+1 < len(field.node.Content); i += 2 {
		key, value := field.node.Content[i], field.node.Content[i+1]
		child := configChild(field, key.Value, value)
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			d.issue("key", child, d.title+" keys must be strings.")
			continue
		}
		if _, found := result[key.Value]; found {
			d.issue("duplicate_key", child, fmt.Sprintf("Duplicate %s key %q.", d.name, key.Value))
			continue
		}
		result[key.Value] = child
		if allowed != nil {
			known := false
			for _, name := range allowed {
				known = known || key.Value == name
			}
			if !known {
				d.issue("unknown_key", child, fmt.Sprintf("Unknown %s key %q.", d.name, key.Value))
			}
		}
	}
	return result
}

func (d configYAML) required(fields map[string]configField, parent configField, name string) configField {
	if field, found := fields[name]; found {
		return field
	}
	field := configChild(parent, name, nil)
	d.issue("field_missing", field, fmt.Sprintf("Required %s field %q is missing.", d.name, name))
	return field
}

func (d configYAML) text(field configField, empty bool) (string, bool) {
	if field.node == nil {
		return "", false
	}
	if field.node.Kind != yaml.ScalarNode || field.node.Tag != "!!str" || !empty && strings.TrimSpace(field.node.Value) == "" {
		d.issue("string", field, "Expected a literal string (nonempty unless explicitly allowed).")
		return "", false
	}
	return field.node.Value, true
}
