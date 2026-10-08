// Package canonical contains Host-owned codecs for canonical source files.
package canonical

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

// DecodeSchema reads one closed Schema YAML document and binds exact source
// provenance to it. A Schema file has apiVersion, purpose, and kinds at its
// root; it is not a Markitect Definition envelope.
func DecodeSchema(filePath string, data []byte) (core.Schema, error) {
	root, err := strictDocument(filePath, data, core.MaxSchemaBytes)
	if err != nil {
		return core.Schema{}, err
	}
	if err := validateSchemaNode(filePath, root); err != nil {
		return core.Schema{}, err
	}
	var wire schemaWire
	if err := root.Decode(&wire); err != nil {
		return core.Schema{}, fmt.Errorf("%s:%d: invalid Schema: %w", filePath, lineOf(root), err)
	}
	schema, err := wire.toCore()
	if err != nil {
		return core.Schema{}, fmt.Errorf("%s:%d: invalid Schema: %w", filePath, lineOf(root), err)
	}
	schema.Source = core.Source{Path: filePath, Digest: digestBytes(data), Line: lineOf(root)}
	return schema, nil
}

// DecodeDefinition reads one closed Definition document and binds its exact
// source path and bytes as provenance.
func DecodeDefinition(filePath string, data []byte) (core.Definition, error) {
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

type schemaWire struct {
	APIVersion string              `yaml:"apiVersion"`
	Purpose    string              `yaml:"purpose"`
	Kinds      map[string]kindWire `yaml:"kinds"`
}
type kindWire struct {
	Purpose    string                  `yaml:"purpose"`
	Properties map[string]propertyWire `yaml:"properties"`
}
type propertyWire struct {
	Purpose    string                  `yaml:"purpose"`
	Type       string                  `yaml:"type"`
	MinCount   int                     `yaml:"minCount"`
	MaxCount   any                     `yaml:"maxCount"`
	Values     []string                `yaml:"values,omitempty"`
	Target     *core.KindIdentity      `yaml:"target,omitempty"`
	Properties map[string]propertyWire `yaml:"properties,omitempty"`
}

func (w schemaWire) toCore() (core.Schema, error) {
	s := core.Schema{APIVersion: w.APIVersion, Purpose: w.Purpose, Kinds: make(map[string]core.Kind, len(w.Kinds))}
	for name, source := range w.Kinds {
		k := core.Kind{Purpose: source.Purpose, Properties: make(map[string]core.Property, len(source.Properties))}
		for propertyName, property := range source.Properties {
			p, err := property.toCore()
			if err != nil {
				return core.Schema{}, fmt.Errorf("Kind %q Property %q: %w", name, propertyName, err)
			}
			k.Properties[propertyName] = p
		}
		s.Kinds[name] = k
	}
	return s, nil
}
func (w propertyWire) toCore() (core.Property, error) {
	max, err := maxCount(w.MaxCount)
	if err != nil {
		return core.Property{}, err
	}
	p := core.Property{Purpose: w.Purpose, Type: w.Type, MinCount: w.MinCount, MaxCount: max, Values: append([]string(nil), w.Values...), Target: w.Target}
	if w.Properties != nil {
		p.Properties = make(map[string]core.Property, len(w.Properties))
		for name, nested := range w.Properties {
			v, e := nested.toCore()
			if e != nil {
				return core.Property{}, fmt.Errorf("nested Property %q: %w", name, e)
			}
			p.Properties[name] = v
		}
	}
	return p, nil
}
func maxCount(value any) (int, error) {
	switch n := value.(type) {
	case int:
		if n < 0 {
			return 0, fmt.Errorf("maxCount must be nonnegative or 'unbounded'")
		}
		return n, nil
	case int64:
		if n < 0 || int64(int(n)) != n {
			return 0, fmt.Errorf("maxCount is outside supported integer range")
		}
		return int(n), nil
	case string:
		if n == "unbounded" {
			return core.Unbounded, nil
		}
		return 0, fmt.Errorf("maxCount string must be exactly 'unbounded'")
	default:
		return 0, fmt.Errorf("maxCount must be an integer or 'unbounded'")
	}
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

// inspectNode enforces representation mechanics shared by every strict Host
// codec: one mapping-safe YAML tree, unique string keys, and no merge/alias/tag
// features. Semantic allowed-field checks remain with each typed codec.
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

func validateSchemaNode(file string, n *yaml.Node) error {
	if err := closedMapping(file, n, "Schema", "apiVersion", "purpose", "kinds"); err != nil {
		return err
	}
	if err := required(file, n, "apiVersion", "purpose", "kinds"); err != nil {
		return err
	}
	for _, f := range []string{"apiVersion", "purpose"} {
		if err := stringNode(file, child(n, f), f); err != nil {
			return err
		}
	}
	kinds := child(n, "kinds")
	if kinds.Kind != yaml.MappingNode || len(kinds.Content) == 0 {
		return fmt.Errorf("%s:%d: kinds must be a nonempty mapping", file, lineOf(kinds))
	}
	for i := 0; i+1 < len(kinds.Content); i += 2 {
		k := kinds.Content[i+1]
		if err := closedMapping(file, k, "Kind", "purpose", "properties"); err != nil {
			return err
		}
		if err := required(file, k, "purpose", "properties"); err != nil {
			return err
		}
		if err := stringNode(file, child(k, "purpose"), "Kind purpose"); err != nil {
			return err
		}
		ps := child(k, "properties")
		if ps.Kind != yaml.MappingNode {
			return fmt.Errorf("%s:%d: Kind properties must be a mapping", file, lineOf(ps))
		}
		for j := 0; j+1 < len(ps.Content); j += 2 {
			if err := validatePropertyNode(file, ps.Content[j+1]); err != nil {
				return err
			}
		}
	}
	return nil
}
func validatePropertyNode(file string, n *yaml.Node) error {
	if err := closedMapping(file, n, "Property", "purpose", "type", "minCount", "maxCount", "values", "target", "properties"); err != nil {
		return err
	}
	if err := required(file, n, "purpose", "type", "minCount", "maxCount"); err != nil {
		return err
	}
	for _, f := range []string{"purpose", "type"} {
		if err := stringNode(file, child(n, f), "Property "+f); err != nil {
			return err
		}
	}
	min, max := child(n, "minCount"), child(n, "maxCount")
	if min.Kind != yaml.ScalarNode || min.Tag != "!!int" {
		return fmt.Errorf("%s:%d: minCount must be an integer", file, lineOf(min))
	}
	if max.Kind != yaml.ScalarNode || (max.Tag != "!!int" && !(max.Tag == "!!str" && max.Value == "unbounded")) {
		return fmt.Errorf("%s:%d: maxCount must be an integer or exactly 'unbounded'", file, lineOf(max))
	}
	if values := child(n, "values"); values != nil {
		if values.Kind != yaml.SequenceNode {
			return fmt.Errorf("%s:%d: values must be a sequence", file, lineOf(values))
		}
		for _, v := range values.Content {
			if err := stringNode(file, v, "enum value"); err != nil {
				return err
			}
		}
	}
	if target := child(n, "target"); target != nil {
		if err := closedMapping(file, target, "target", "apiVersion", "kind"); err != nil {
			return err
		}
		if err := required(file, target, "apiVersion", "kind"); err != nil {
			return err
		}
		for _, f := range []string{"apiVersion", "kind"} {
			if err := stringNode(file, child(target, f), "target "+f); err != nil {
				return err
			}
		}
	}
	if nested := child(n, "properties"); nested != nil {
		if nested.Kind != yaml.MappingNode {
			return fmt.Errorf("%s:%d: nested properties must be a mapping", file, lineOf(nested))
		}
		for i := 0; i+1 < len(nested.Content); i += 2 {
			if err := validatePropertyNode(file, nested.Content[i+1]); err != nil {
				return err
			}
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
func safeModulePath(value string) (string, error) {
	if value == "" || len(value) > 1024 || strings.ContainsAny(value, "\\:\x00*?[]") || strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("unsafe Module path %q", value)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("unsafe Module path %q", value)
		}
	}
	return value, nil
}
