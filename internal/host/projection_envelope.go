package host

import (
	"bytes"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

// A projection may contain an opaque YAML envelope from another API, but may
// never introduce or hide resources of an active canonical Domain.
func isCanonicalProjectionEnvelope(data []byte, registry *core.Registry) bool {
	if registry == nil {
		return false
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			return false
		}
		if len(node.Content) == 0 || node.Content[0].Kind != yaml.MappingNode {
			continue
		}
		root := node.Content[0]
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value == "apiVersion" && registry.IsAPIVersionRegistered(root.Content[i+1].Value) {
				return true
			}
		}
	}
}
