package projectwork

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"regexp"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"go.yaml.in/yaml/v3"
)

// decodeDefinition reads one closed Definition document and binds its exact
// source path and bytes as provenance.
func decodeDefinition(filePath string, data []byte) (core.Definition, error) {
	root, err := strictDocument(filePath, data, core.MaxDefinitionBytes)
	if err != nil {
		return core.Definition{}, err
	}
	if err := validateDefinitionNode(filePath, root); err != nil {
		return core.Definition{}, err
	}
	var wire definitionWire
	if err := root.Decode(&wire); err != nil {
		return core.Definition{}, fmt.Errorf("%s:%d: invalid Definition: %w", filePath, lineOf(root), err)
	}
	return core.Definition{
		APIVersion: wire.APIVersion, Kind: wire.Kind,
		Metadata: core.Metadata{Name: wire.Metadata.Name, Namespace: wire.Metadata.Namespace},
		Purpose:  wire.Purpose, Spec: wire.Spec,
		Source: core.Source{Path: filePath, Digest: digestBytes(data), Line: lineOf(root)},
	}, nil
}

type definitionWire struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Purpose string         `yaml:"purpose"`
	Spec    map[string]any `yaml:"spec"`
}

func strictDocument(file string, data []byte, limit int) (*yaml.Node, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("%s:1: empty YAML document", file)
	}
	if len(data) > limit {
		return nil, fmt.Errorf("%s:1: YAML input exceeds the %d-byte limit", file, limit)
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%s:1: YAML input must be valid UTF-8", file)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s:%d: invalid YAML: %w", file, yamlErrorLine(err), err)
	}
	if len(doc.Content) != 1 || doc.Content[0] == nil || doc.Content[0].Kind == 0 || nullNode(doc.Content[0]) {
		return nil, fmt.Errorf("%s:%d: empty YAML document", file, lineOf(&doc))
	}
	root := doc.Content[0]
	if err := inspectNode(file, root); err != nil {
		return nil, err
	}
	var extra yaml.Node
	err := dec.Decode(&extra)
	if err == nil {
		return nil, fmt.Errorf("%s:%d: multiple YAML documents are not allowed", file, lineOf(&extra))
	}
	if err != io.EOF {
		return nil, fmt.Errorf("%s:%d: invalid YAML: %w", file, yamlErrorLine(err), err)
	}
	return root, nil
}

// inspectNode enforces the representation mechanics of the strict Definition
// codec: one mapping-safe YAML tree, unique string keys, and no merge/alias/tag
// features. Allowed-field checks remain with validateDefinitionNode.
func inspectNode(file string, n *yaml.Node) error {
	return inspectNodeAt(file, n, 0)
}

const maxYAMLDepth = 64

func inspectNodeAt(file string, n *yaml.Node, depth int) error {
	if depth > maxYAMLDepth {
		return fmt.Errorf("%s:%d: YAML nesting exceeds the %d-level limit", file, lineOf(n), maxYAMLDepth)
	}
	if n.Anchor != "" || n.Kind == yaml.AliasNode {
		return fmt.Errorf("%s:%d: YAML anchors and aliases are not allowed", file, lineOf(n))
	}
	if n.Style&yaml.TaggedStyle != 0 {
		return fmt.Errorf("%s:%d: explicit YAML tags are not allowed", file, lineOf(n))
	}
	switch n.Tag {
	case "", "!!map", "!!seq", "!!str", "!!int", "!!float", "!!bool", "!!null":
	default:
		return fmt.Errorf("%s:%d: unsupported YAML tag %q", file, lineOf(n), n.Tag)
	}
	if n.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Value == "<<" || k.Tag == "!!merge" {
				return fmt.Errorf("%s:%d: YAML merge keys are not allowed", file, lineOf(k))
			}
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" {
				return fmt.Errorf("%s:%d: mapping keys must be strings", file, lineOf(k))
			}
			if seen[k.Value] {
				return fmt.Errorf("%s:%d: duplicate YAML key %q", file, lineOf(k), k.Value)
			}
			seen[k.Value] = true
		}
	}
	for _, c := range n.Content {
		if err := inspectNodeAt(file, c, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func validateDefinitionNode(file string, n *yaml.Node) error {
	if err := closedMapping(file, n, "Definition", "apiVersion", "kind", "metadata", "purpose", "spec"); err != nil {
		return err
	}
	if err := required(file, n, "apiVersion", "kind", "metadata", "purpose", "spec"); err != nil {
		return err
	}
	for _, f := range []string{"apiVersion", "kind", "purpose"} {
		if err := stringNode(file, child(n, f), f); err != nil {
			return err
		}
	}
	meta := child(n, "metadata")
	if err := closedMapping(file, meta, "metadata", "name", "namespace"); err != nil {
		return err
	}
	if err := required(file, meta, "name", "namespace"); err != nil {
		return err
	}
	for _, f := range []string{"name", "namespace"} {
		if err := stringNode(file, child(meta, f), "metadata."+f); err != nil {
			return err
		}
	}
	if spec := child(n, "spec"); spec.Kind != yaml.MappingNode {
		return fmt.Errorf("%s:%d: spec must be a mapping", file, lineOf(spec))
	}
	return nil
}

func closedMapping(file string, n *yaml.Node, what string, fields ...string) error {
	if n == nil || n.Kind != yaml.MappingNode {
		return fmt.Errorf("%s:%d: %s must be a mapping", file, lineOf(n), what)
	}
	allowed := map[string]bool{}
	for _, f := range fields {
		allowed[f] = true
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k := n.Content[i]
		if !allowed[k.Value] {
			return fmt.Errorf("%s:%d: unknown %s field %q", file, lineOf(k), what, k.Value)
		}
	}
	return nil
}

func required(file string, n *yaml.Node, fields ...string) error {
	for _, f := range fields {
		if child(n, f) == nil {
			return fmt.Errorf("%s:%d: required field %q is missing", file, lineOf(n), f)
		}
	}
	return nil
}

func stringNode(file string, n *yaml.Node, what string) error {
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
		return fmt.Errorf("%s:%d: %s must be a string", file, lineOf(n), what)
	}
	return nil
}

func child(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func nullNode(n *yaml.Node) bool { return n != nil && n.Kind == yaml.ScalarNode && n.Tag == "!!null" }

func lineOf(n *yaml.Node) int {
	if n == nil || n.Line < 1 {
		return 1
	}
	return n.Line
}

var yamlLinePattern = regexp.MustCompile(`line ([0-9]+):`)

func yamlErrorLine(err error) int {
	if m := yamlLinePattern.FindStringSubmatch(err.Error()); len(m) == 2 {
		var line int
		_, _ = fmt.Sscanf(m[1], "%d", &line)
		if line > 0 {
			return line
		}
	}
	return 1
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:])
}
