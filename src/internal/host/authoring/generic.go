package authoring

import (
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

func validateGenericSpec(file string, node *yaml.Node, registry *core.Registry, apiVersion, kind string) error {
	if err := requireMapping(file, node, "spec"); err != nil {
		return err
	}
	var data map[string]any
	if err := node.Decode(&data); err != nil {
		return diagnostic(file, node.Line, "invalid spec: %v", err)
	}
	if err := registry.ValidateData(apiVersion, kind, data); err != nil {
		return diagnostic(file, node.Line, "invalid %s/%s spec: %v", apiVersion, kind, err)
	}
	return nil
}

// IsResourceEnvelopeWithRegistry recognizes any valid envelope whose API is
// registered, including a malformed or unknown-kind resource of a declared
// API. Such files remain typed inputs and cannot be downgraded to prose.
func IsResourceEnvelopeWithRegistry(data []byte, registry *core.Registry) bool {
	if registry == nil {
		registry = NewRegistry()
	}
	if IsResourceEnvelope(data) {
		return true
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		return false
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return false
	}
	api, kind := "", ""
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]
		if key.Value == "apiVersion" && value.Kind == yaml.ScalarNode {
			api = value.Value
		}
		if key.Value == "kind" && value.Kind == yaml.ScalarNode {
			kind = value.Value
		}
	}
	return api != "" && kind != "" && registry.IsAPIVersionRegistered(api)
}
