package projectcoverage

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"
)

const IgnoreAPIVersion = "project.markitect.example.org/repository-ignore/v1alpha1"

type IgnoreEntry struct {
	Path   string `json:"path" yaml:"path"`
	Reason string `json:"reason" yaml:"reason"`
}

type IgnoreFile struct {
	APIVersion string        `json:"apiVersion" yaml:"apiVersion"`
	Kind       string        `json:"kind" yaml:"kind"`
	Entries    []IgnoreEntry `json:"entries" yaml:"entries"`
}

func DecodeIgnore(data []byte) (IgnoreFile, error) {
	if len(data) == 0 {
		return IgnoreFile{APIVersion: IgnoreAPIVersion, Kind: "RepositoryIgnore", Entries: []IgnoreEntry{}}, nil
	}
	if len(data) > 1<<20 {
		return IgnoreFile{}, fmt.Errorf("%s exceeds the 1 MiB limit", IgnorePath)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var node yaml.Node
	if err := dec.Decode(&node); err != nil {
		return IgnoreFile{}, fmt.Errorf("decode %s: %w", IgnorePath, err)
	}
	if err := validateYAMLNode(&node); err != nil {
		return IgnoreFile{}, fmt.Errorf("%s: %w", IgnorePath, err)
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return IgnoreFile{}, fmt.Errorf("%s must contain exactly one YAML document", IgnorePath)
		}
		return IgnoreFile{}, fmt.Errorf("decode trailing %s: %w", IgnorePath, err)
	}
	var cfg IgnoreFile
	known := yaml.NewDecoder(bytes.NewReader(data))
	known.KnownFields(true)
	if err := known.Decode(&cfg); err != nil {
		return IgnoreFile{}, fmt.Errorf("decode %s: %w", IgnorePath, err)
	}
	if cfg.APIVersion != IgnoreAPIVersion || cfg.Kind != "RepositoryIgnore" {
		return IgnoreFile{}, fmt.Errorf("%s must use apiVersion %q and kind RepositoryIgnore", IgnorePath, IgnoreAPIVersion)
	}
	if cfg.Entries == nil {
		cfg.Entries = []IgnoreEntry{}
	}
	seen := map[string]string{}
	for i, entry := range cfg.Entries {
		selector, ok := normalizeSelector(entry.Path)
		if !ok {
			return IgnoreFile{}, fmt.Errorf("%s entries[%d].path must be a normalized repository-relative file or trailing-slash directory prefix", IgnorePath, i)
		}
		if selector == IgnorePath || selectorWithin(selector, ".git/") || selectorWithin(selector, ".markitect/") {
			return IgnoreFile{}, fmt.Errorf("%s entries[%d].path targets reserved Markitect or Git metadata", IgnorePath, i)
		}
		if strings.TrimSpace(entry.Reason) == "" || entry.Reason != strings.TrimSpace(entry.Reason) {
			return IgnoreFile{}, fmt.Errorf("%s entries[%d].reason must be nonempty without surrounding whitespace", IgnorePath, i)
		}
		key := strings.ToLower(selector)
		if previous, exists := seen[key]; exists {
			return IgnoreFile{}, fmt.Errorf("%s repeats or case-aliases %q and %q", IgnorePath, previous, selector)
		}
		for _, old := range cfg.Entries[:i] {
			if selectorsOverlap(strings.ToLower(old.Path), strings.ToLower(selector)) {
				return IgnoreFile{}, fmt.Errorf("%s entries %q and %q overlap", IgnorePath, old.Path, selector)
			}
		}
		seen[key] = selector
		cfg.Entries[i].Path = selector
	}
	return cfg, nil
}

func validateYAMLNode(document *yaml.Node) error {
	if document == nil || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("must contain one YAML mapping")
	}
	var visit func(*yaml.Node) error
	visit = func(node *yaml.Node) error {
		if node.Kind == yaml.AliasNode || node.Alias != nil || node.Anchor != "" {
			return fmt.Errorf("YAML aliases and anchors are not supported")
		}
		if node.Kind == yaml.MappingNode {
			seen := map[string]bool{}
			for i := 0; i+1 < len(node.Content); i += 2 {
				key := node.Content[i]
				if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "<<" {
					return fmt.Errorf("mapping keys must be ordinary strings")
				}
				if seen[key.Value] {
					return fmt.Errorf("duplicate key %q", key.Value)
				}
				seen[key.Value] = true
			}
		}
		for _, child := range node.Content {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(document.Content[0]); err != nil {
		return err
	}
	root := document.Content[0]
	if err := exactFields(root, map[string]bool{"apiVersion": true, "kind": true, "entries": true}); err != nil {
		return err
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "entries" {
			continue
		}
		for _, item := range root.Content[i+1].Content {
			if item.Kind == yaml.MappingNode {
				if err := exactFields(item, map[string]bool{"path": true, "reason": true}); err != nil {
					return fmt.Errorf("entries: %w", err)
				}
			}
		}
	}
	return nil
}

func exactFields(node *yaml.Node, allowed map[string]bool) error {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if !allowed[node.Content[i].Value] {
			return fmt.Errorf("unknown or incorrectly cased field %q", node.Content[i].Value)
		}
	}
	return nil
}

func normalizeSelector(raw string) (string, bool) {
	if raw == "" || strings.ContainsAny(raw, "\\:\x00*?[]!") || strings.HasPrefix(raw, "/") {
		return "", false
	}
	prefix := strings.HasSuffix(raw, "/")
	body := strings.TrimSuffix(raw, "/")
	if body == "" || path.Clean(body) != body {
		return "", false
	}
	for _, part := range strings.Split(body, "/") {
		if !safePathComponent(part) || strings.ContainsAny(part, "[]!") {
			return "", false
		}
	}
	if prefix {
		body += "/"
	}
	return body, true
}

func selectorWithin(selector, reserved string) bool {
	selector = strings.TrimSuffix(strings.ToLower(selector), "/")
	reserved = strings.TrimSuffix(strings.ToLower(reserved), "/")
	return selector == reserved || strings.HasPrefix(selector, reserved+"/") || strings.HasPrefix(reserved, selector+"/")
}

func selectorMatches(selector, file string) bool {
	if strings.HasSuffix(selector, "/") {
		return strings.HasPrefix(file, selector)
	}
	return selector == file
}

func selectorsOverlap(a, b string) bool {
	aPrefix, bPrefix := strings.HasSuffix(a, "/"), strings.HasSuffix(b, "/")
	switch {
	case !aPrefix && !bPrefix:
		return a == b
	case aPrefix && bPrefix:
		return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
	case aPrefix:
		return strings.HasPrefix(b, a)
	default:
		return strings.HasPrefix(a, b)
	}
}
