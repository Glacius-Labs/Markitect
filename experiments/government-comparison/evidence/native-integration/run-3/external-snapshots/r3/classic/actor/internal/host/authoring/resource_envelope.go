package authoring

import (
	"bytes"
	"io"
	"strings"

	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

// IsResourceEnvelope reports whether YAML appears to be a Markitect resource,
// including a malformed or unsupported resource envelope. Other YAML shapes
// may be treated as ordinary inputs when explicitly declared.
func IsResourceEnvelope(data []byte) bool {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var document yaml.Node
		err := decoder.Decode(&document)
		if err == io.EOF {
			return false
		}
		if err != nil {
			return true
		}
		if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
			continue
		}
		root := document.Content[0]
		if hasDuplicateScalarYAMLKeys(root) {
			return true
		}
		var apiVersion, kind, metadata, spec *yaml.Node
		for i := 0; i+1 < len(root.Content); i += 2 {
			key := root.Content[i]
			if key.Kind != yaml.ScalarNode {
				continue
			}
			value := root.Content[i+1]
			switch key.Value {
			case "apiVersion":
				apiVersion = value
			case "kind":
				kind = value
			case "metadata":
				metadata = value
			case "spec":
				spec = value
			}
		}
		knownResourceShape := kind != nil && kind.Kind == yaml.ScalarNode &&
			AllowedSpecFields(kind.Value) != nil &&
			metadata != nil && metadata.Kind == yaml.MappingNode &&
			spec != nil && spec.Kind == yaml.MappingNode
		if apiVersion != nil {
			apiGroup, _, _ := strings.Cut(core.APIVersion, "/")
			if (apiVersion.Kind == yaml.ScalarNode && strings.HasPrefix(apiVersion.Value, apiGroup+"/")) || knownResourceShape {
				return true
			}
			continue
		}
		if knownResourceShape {
			return true
		}
	}
}

func hasDuplicateScalarYAMLKeys(node *yaml.Node) bool {
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]bool, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind == yaml.ScalarNode {
				identity := key.Tag + "\x00" + key.Value
				if seen[identity] {
					return true
				}
				seen[identity] = true
			}
		}
	}
	for _, child := range node.Content {
		if hasDuplicateScalarYAMLKeys(child) {
			return true
		}
	}
	return false
}
