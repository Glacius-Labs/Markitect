// Package format reads and writes Markitect YAML resources.
package format

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
	"markitect/internal/core"
)

const maxResourceSize = 2 << 20

var dnsLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

var specFields = map[string][]string{
	"Text":     {"text", "files"},
	"Rule":     {"text", "check", "files"},
	"Contract": {"text", "kind", "input", "output", "files"},
	"Workflow": {"text", "rules", "uses", "needs", "implements", "input", "output", "files"},
	"Skill":    {"text", "description", "rules", "uses", "needs", "implements", "input", "output", "files"},
	"Agent":    {"text", "description", "rules", "uses", "needs", "implements", "input", "output", "providers", "files"},
	"Project":  {"profile", "targets", "areas", "bindings", "ruleAdapters"},
}

// AllowedSpecFields returns the accepted spec fields for a resource kind.
// Unknown kinds return nil. The returned slice is independent of the format package.
func AllowedSpecFields(kind string) []string {
	fields := specFields[kind]
	return append([]string(nil), fields...)
}

// Parse decodes exactly one strict Markitect YAML resource from data.
func Parse(filePath string, data []byte) (*core.Resource, error) {
	if len(data) > maxResourceSize {
		return nil, diagnostic(filePath, 1, "resource exceeds the 2 MiB limit")
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return nil, diagnostic(filePath, yamlErrorLine(err), "invalid YAML: %v", err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind == 0 || isNull(doc.Content[0]) {
		return nil, diagnostic(filePath, maxInt(1, doc.Line), "empty YAML document")
	}
	if err := inspectNode(filePath, &doc); err != nil {
		return nil, err
	}
	var extra yaml.Node
	err := dec.Decode(&extra)
	if err == nil {
		line := extra.Line
		if line == 0 {
			line = doc.Line + 1
		}
		return nil, diagnostic(filePath, line, "multiple YAML documents are not allowed")
	}
	if err != io.EOF {
		return nil, diagnostic(filePath, yamlErrorLine(err), "invalid YAML: %v", err)
	}

	root := doc.Content[0]
	if err := requireMapping(filePath, root, "resource"); err != nil {
		return nil, err
	}
	if err := checkKeys(filePath, root, set("apiVersion", "kind", "metadata", "spec")); err != nil {
		return nil, err
	}
	if err := requireFields(filePath, root, "apiVersion", "kind", "metadata", "spec"); err != nil {
		return nil, err
	}
	if err := checkScalar(filePath, child(root, "apiVersion"), "string"); err != nil {
		return nil, err
	}
	if err := checkScalar(filePath, child(root, "kind"), "string"); err != nil {
		return nil, err
	}
	kind := child(root, "kind").Value
	if !oneOf(kind, "Text", "Rule", "Workflow", "Skill", "Agent", "Contract", "Project") {
		return nil, diagnostic(filePath, child(root, "kind").Line, "unsupported resource kind %q", kind)
	}
	if err := validateMetadata(filePath, child(root, "metadata"), kind); err != nil {
		return nil, err
	}
	if err := validateSpec(filePath, child(root, "spec"), kind); err != nil {
		return nil, err
	}

	var r core.Resource
	strict := yaml.NewDecoder(bytes.NewReader(data))
	strict.KnownFields(true)
	if err := strict.Decode(&r); err != nil {
		return nil, diagnostic(filePath, yamlErrorLine(err), "invalid resource: %v", err)
	}
	if r.APIVersion != core.APIVersion {
		return nil, diagnostic(filePath, child(root, "apiVersion").Line, "apiVersion must be %q", core.APIVersion)
	}
	r.Path = filePath
	r.Line = root.Line
	return &r, nil
}

// Encode serializes a Go value as YAML.
func Encode(value any) ([]byte, error) { return yaml.Marshal(value) }

func validateMetadata(file string, n *yaml.Node, kind string) error {
	if err := requireMapping(file, n, "metadata"); err != nil {
		return err
	}
	if err := checkKeys(file, n, set("name", "namespace")); err != nil {
		return err
	}
	if err := requireFields(file, n, "name"); err != nil {
		return err
	}
	if kind == "Project" && child(n, "namespace") != nil {
		return diagnostic(file, child(n, "namespace").Line, "Project metadata must not have a namespace")
	}
	for _, field := range []string{"name", "namespace"} {
		v := child(n, field)
		if v == nil {
			continue
		}
		if err := checkScalar(file, v, "string"); err != nil {
			return err
		}
		if !validName(v.Value) {
			return diagnostic(file, v.Line, "%s must be a DNS label of at most 63 characters", field)
		}
	}
	return nil
}

func validateSpec(file string, n *yaml.Node, kind string) error {
	if err := requireMapping(file, n, "spec"); err != nil {
		return err
	}
	allowed := AllowedSpecFields(kind)
	keys := make(map[string]struct{}, len(allowed))
	for _, k := range allowed {
		keys[k] = struct{}{}
	}
	if err := checkKeys(file, n, keys); err != nil {
		return err
	}
	if kind != "Project" {
		text := child(n, "text")
		if text == nil {
			return diagnostic(file, n.Line, "%s spec requires nonempty text", kind)
		}
		if err := checkScalar(file, text, "string"); err != nil {
			return err
		}
		if strings.TrimSpace(text.Value) == "" {
			return diagnostic(file, text.Line, "%s spec text must not be empty", kind)
		}
	}
	if d := child(n, "description"); d != nil {
		if err := checkScalar(file, d, "string"); err != nil {
			return err
		}
	}
	if d := child(n, "check"); d != nil {
		if err := checkScalar(file, d, "string"); err != nil {
			return err
		}
	}
	if d := child(n, "kind"); d != nil {
		if err := checkScalar(file, d, "string"); err != nil {
			return err
		}
		if kind == "Contract" && !oneOf(d.Value, "Agent", "Workflow", "Skill") {
			return diagnostic(file, d.Line, "Contract kind must be Agent, Workflow, or Skill")
		}
	}
	for _, field := range []string{"profile"} {
		if d := child(n, field); d != nil {
			if err := checkScalar(file, d, "string"); err != nil {
				return err
			}
			if kind == "Project" && !oneOf(d.Value, "generic", "konfyra", "cockpit") {
				return diagnostic(file, d.Line, "Project profile must be generic, konfyra, or cockpit")
			}
		}
	}
	if kind == "Project" && child(n, "profile") == nil {
		return diagnostic(file, n.Line, "Project spec requires profile")
	}
	for _, field := range []string{"input", "output", "targets"} {
		if d := child(n, field); d != nil {
			if err := validateStrings(file, d, field, field == "targets"); err != nil {
				return err
			}
		}
	}
	if d := child(n, "files"); d != nil {
		if kind == "Project" {
			return diagnostic(file, d.Line, "Project spec must not declare files")
		}
		if err := validateStrings(file, d, "files", true); err != nil {
			return err
		}
	}
	for _, field := range []string{"rules", "uses", "needs", "implements"} {
		if d := child(n, field); d != nil {
			if err := validateRefs(file, d, field); err != nil {
				return err
			}
		}
	}
	if kind == "Contract" {
		for _, field := range []string{"input", "output"} {
			if d := child(n, field); d != nil {
				if err := uniqueNonemptyStrings(file, d, field); err != nil {
					return err
				}
			}
		}
	}
	if d := child(n, "providers"); d != nil {
		if err := validateProviders(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "areas"); d != nil {
		if err := validateAreas(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "bindings"); d != nil {
		if err := validateBindings(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "ruleAdapters"); d != nil {
		if err := validateRuleAdapters(file, d); err != nil {
			return err
		}
	}
	return nil
}

func validateRefs(file string, n *yaml.Node, field string) error {
	if err := requireSequence(file, n, field); err != nil {
		return err
	}
	for _, item := range n.Content {
		if err := requireMapping(file, item, field+" reference"); err != nil {
			return err
		}
		if err := checkKeys(file, item, set("kind", "name", "namespace")); err != nil {
			return err
		}
		if err := requireFields(file, item, "name"); err != nil {
			return err
		}
		for _, key := range []string{"kind", "name", "namespace"} {
			if v := child(item, key); v != nil {
				if err := checkScalar(file, v, "string"); err != nil {
					return err
				}
				if (key == "name" || key == "namespace") && !validName(v.Value) {
					return diagnostic(file, v.Line, "reference %s must be a DNS label of at most 63 characters", key)
				}
			}
		}
		if field == "uses" && child(item, "kind") == nil {
			return diagnostic(file, item.Line, "uses references require kind")
		}
		if field == "rules" && child(item, "kind") != nil && child(item, "kind").Value != "Rule" {
			return diagnostic(file, child(item, "kind").Line, "rules references must have kind Rule")
		}
	}
	return nil
}

func validateStrings(file string, n *yaml.Node, field string, unique bool) error {
	if err := requireSequence(file, n, field); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, item := range n.Content {
		if err := checkScalar(file, item, "string"); err != nil {
			return err
		}
		if unique && strings.TrimSpace(item.Value) == "" {
			return diagnostic(file, item.Line, "%s entries must not be empty", field)
		}
		if unique && seen[item.Value] {
			return diagnostic(file, item.Line, "%s entries must be unique", field)
		}
		seen[item.Value] = true
	}
	return nil
}
func uniqueNonemptyStrings(file string, n *yaml.Node, field string) error {
	return validateStrings(file, n, field, true)
}

func validateAreas(file string, n *yaml.Node) error {
	if err := requireSequence(file, n, "areas"); err != nil {
		return err
	}
	names, paths := map[string]bool{}, map[string]bool{}
	for _, a := range n.Content {
		if err := requireMapping(file, a, "area"); err != nil {
			return err
		}
		if err := checkKeys(file, a, set("name", "path", "imports", "rules")); err != nil {
			return err
		}
		if err := requireFields(file, a, "name", "path"); err != nil {
			return err
		}
		name, p := child(a, "name"), child(a, "path")
		for _, v := range []*yaml.Node{name, p} {
			if err := checkScalar(file, v, "string"); err != nil {
				return err
			}
		}
		if !validName(name.Value) {
			return diagnostic(file, name.Line, "area name must be a DNS label of at most 63 characters")
		}
		if names[name.Value] {
			return diagnostic(file, name.Line, "duplicate area name %q", name.Value)
		}
		names[name.Value] = true
		if p.Value == "" || path.IsAbs(p.Value) || path.Clean(p.Value) != p.Value || p.Value == "." || strings.HasPrefix(p.Value, "../") {
			return diagnostic(file, p.Line, "area path must be a normalized relative POSIX path without '..'")
		}
		for _, part := range strings.Split(p.Value, "/") {
			if part == ".." || part == "." || part == "" {
				return diagnostic(file, p.Line, "area path must be a normalized relative POSIX path without '..'")
			}
		}
		if paths[p.Value] {
			return diagnostic(file, p.Line, "duplicate area path %q", p.Value)
		}
		paths[p.Value] = true
		if x := child(a, "imports"); x != nil {
			if err := validateStrings(file, x, "area imports", true); err != nil {
				return err
			}
			for _, s := range x.Content {
				if !validName(s.Value) {
					return diagnostic(file, s.Line, "area import must be a DNS label")
				}
			}
		}
		if x := child(a, "rules"); x != nil {
			if err := validateRefs(file, x, "rules"); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBindings(file string, n *yaml.Node) error {
	if err := requireSequence(file, n, "bindings"); err != nil {
		return err
	}
	for _, b := range n.Content {
		if err := requireMapping(file, b, "binding"); err != nil {
			return err
		}
		if err := checkKeys(file, b, set("contract", "implementation")); err != nil {
			return err
		}
		if err := requireFields(file, b, "contract", "implementation"); err != nil {
			return err
		}
		for _, field := range []string{"contract", "implementation"} {
			r := child(b, field)
			if err := requireMapping(file, r, field); err != nil {
				return err
			}
			if err := checkKeys(file, r, set("kind", "name", "namespace")); err != nil {
				return err
			}
			if err := requireFields(file, r, "name"); err != nil {
				return err
			}
			for _, key := range []string{"kind", "name", "namespace"} {
				if x := child(r, key); x != nil {
					if err := checkScalar(file, x, "string"); err != nil {
						return err
					}
					if (key == "name" || key == "namespace") && !validName(x.Value) {
						return diagnostic(file, x.Line, "binding reference %s must be a DNS label", key)
					}
				}
			}
		}
		if child(child(b, "contract"), "kind") != nil && child(child(b, "contract"), "kind").Value != "Contract" {
			return diagnostic(file, child(child(b, "contract"), "kind").Line, "binding contract kind must be Contract")
		}
		if child(child(b, "implementation"), "kind") == nil {
			return diagnostic(file, child(b, "implementation").Line, "binding implementation requires kind")
		}
	}
	return nil
}

func validateRuleAdapters(file string, n *yaml.Node) error {
	if err := requireMapping(file, n, "ruleAdapters"); err != nil {
		return err
	}
	seen := map[string]bool{}
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if err := checkScalar(file, k, "string"); err != nil {
			return err
		}
		if strings.TrimSpace(k.Value) == "" {
			return diagnostic(file, k.Line, "rule adapter names must not be empty")
		}
		if seen[k.Value] {
			return diagnostic(file, k.Line, "duplicate rule adapter %q", k.Value)
		}
		seen[k.Value] = true
		if err := validateRefs(file, v, "rules"); err != nil {
			return err
		}
	}
	return nil
}

func validateProviders(file string, n *yaml.Node) error {
	if err := requireMapping(file, n, "providers"); err != nil {
		return err
	}
	if err := checkKeys(file, n, set("codex", "claude")); err != nil {
		return err
	}
	for _, provider := range []string{"codex", "claude"} {
		p := child(n, provider)
		if p == nil {
			continue
		}
		if err := requireMapping(file, p, provider+" provider"); err != nil {
			return err
		}
		allowed := set(allowedProviderFields(provider)...)
		if err := checkKeys(file, p, allowed); err != nil {
			return err
		}
		for _, key := range []string{"model", "effort", "sandbox", "permissionMode"} {
			if v := child(p, key); v != nil {
				if _, ok := allowed[key]; ok {
					if err := checkScalar(file, v, "string"); err != nil {
						return err
					}
				}
			}
		}
		for _, field := range []string{"tools", "disallowedTools"} {
			if v := child(p, field); v != nil {
				if provider != "claude" {
					return diagnostic(file, v.Line, "%s is only valid for claude", field)
				}
				if err := validateStrings(file, v, "provider "+field, true); err != nil {
					return err
				}
			}
		}
		if v := child(p, "maxTurns"); v != nil {
			if provider != "claude" {
				return diagnostic(file, v.Line, "maxTurns is only valid for claude")
			}
			if err := checkScalar(file, v, "integer"); err != nil {
				return err
			}
			var turns int
			if err := v.Decode(&turns); err != nil {
				return diagnostic(file, v.Line, "maxTurns must be an integer")
			}
			if turns < 0 {
				return diagnostic(file, v.Line, "maxTurns must not be negative")
			}
		}
	}
	return nil
}

func allowedProviderFields(provider string) []string {
	if provider == "codex" {
		return []string{"model", "effort", "sandbox"}
	}
	return []string{"model", "effort", "permissionMode", "tools", "disallowedTools", "maxTurns"}
}

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
