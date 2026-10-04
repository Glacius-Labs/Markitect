package authoring

import (
	"fmt"
	"regexp"
	"strconv"

	"go.yaml.in/yaml/v3"
)

func inspectNode(file string, n *yaml.Node) error {
	if n.Anchor != "" {
		return diagnostic(file, n.Line, "anchors are not allowed")
	}
	if n.Kind == yaml.AliasNode {
		return diagnostic(file, n.Line, "aliases are not allowed")
	}
	if n.Tag != "" && !oneOf(n.Tag, "!!map", "!!seq", "!!str", "!!int", "!!float", "!!bool", "!!null", "!!timestamp", "!!binary") {
		return diagnostic(file, n.Line, "custom YAML tags are not allowed")
	}
	if n.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Value == "<<" || k.Tag == "!!merge" {
				return diagnostic(file, k.Line, "merge keys are not allowed")
			}
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" {
				return diagnostic(file, k.Line, "mapping keys must be strings")
			}
			if seen[k.Value] {
				return diagnostic(file, k.Line, "duplicate key %q", k.Value)
			}
			seen[k.Value] = true
		}
	}
	for _, c := range n.Content {
		if err := inspectNode(file, c); err != nil {
			return err
		}
	}
	return nil
}
func checkScalar(file string, n *yaml.Node, typ string) error {
	if n.Kind != yaml.ScalarNode {
		return diagnostic(file, n.Line, "expected %s scalar", typ)
	}
	if (typ == "string" && n.Tag != "!!str") || (typ == "integer" && n.Tag != "!!int") {
		return diagnostic(file, n.Line, "expected %s, got %s", typ, n.Tag)
	}
	return nil
}
func requireMapping(file string, n *yaml.Node, what string) error {
	if n == nil || n.Kind != yaml.MappingNode {
		return diagnostic(file, lineOf(n), "%s must be a mapping", what)
	}
	return nil
}
func requireSequence(file string, n *yaml.Node, what string) error {
	if n == nil || n.Kind != yaml.SequenceNode {
		return diagnostic(file, lineOf(n), "%s must be a sequence", what)
	}
	return nil
}
func checkKeys(file string, n *yaml.Node, allowed map[string]struct{}) error {
	if n == nil {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k := n.Content[i]
		if _, ok := allowed[k.Value]; !ok {
			return diagnostic(file, k.Line, "unknown field %q", k.Value)
		}
	}
	return nil
}
func requireFields(file string, n *yaml.Node, fields ...string) error {
	for _, f := range fields {
		if child(n, f) == nil {
			return diagnostic(file, lineOf(n), "required field %q is missing", f)
		}
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
func set(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}
func validName(s string) bool { return len(s) <= 63 && dnsLabel.MatchString(s) }
func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}
func isNull(n *yaml.Node) bool { return n != nil && n.Kind == yaml.ScalarNode && n.Tag == "!!null" }
func lineOf(n *yaml.Node) int {
	if n == nil || n.Line == 0 {
		return 1
	}
	return n.Line
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func yamlErrorLine(err error) int {
	loc := regexp.MustCompile(`line ([0-9]+):`).FindStringSubmatch(err.Error())
	if len(loc) == 2 {
		if line, e := strconv.Atoi(loc[1]); e == nil && line > 0 {
			return line
		}
	}
	return 1
}
func diagnostic(file string, line int, format string, args ...any) error {
	return fmt.Errorf("%s:%d: %s", file, maxInt(1, line), fmt.Sprintf(format, args...))
}
