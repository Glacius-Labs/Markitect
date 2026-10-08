package projectwork

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

const maxProjectConfigBytes = 1 << 20

const APIVersion = projectmodel.APIVersion

var modelNamespaceSegment = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// DecodeConfig parses the deliberately closed project selection manifest.
// The manifest chooses exact model bytes; it never causes a directory scan.
func DecodeConfig(data []byte) (Config, error) {
	if len(data) == 0 || len(data) > maxProjectConfigBytes {
		return Config{}, fmt.Errorf("project config must be between 1 and %d bytes", maxProjectConfigBytes)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := dec.Decode(&document); err != nil {
		return Config{}, fmt.Errorf("invalid project config: %w", err)
	}
	if err := rejectProjectYAMLNode(&document); err != nil {
		return Config{}, err
	}
	if err := validateProjectYAMLFields(&document); err != nil {
		return Config{}, err
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return Config{}, fmt.Errorf("project config must contain exactly one YAML document")
		}
		return Config{}, fmt.Errorf("invalid project config: %w", err)
	}
	known := yaml.NewDecoder(bytes.NewReader(data))
	known.KnownFields(true)
	var config Config
	if err := known.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("invalid project config: %w", err)
	}
	if config.APIVersion != APIVersion {
		return Config{}, fmt.Errorf("project config apiVersion must be %q", APIVersion)
	}
	if strings.TrimSpace(config.Name) == "" || config.Name != strings.TrimSpace(config.Name) {
		return Config{}, fmt.Errorf("project config name must be nonempty and have no surrounding whitespace")
	}
	if len(config.ModelFiles) == 0 {
		return Config{}, fmt.Errorf("project config must select at least one model file")
	}
	if err := validateExactPaths("modelFiles", config.ModelFiles, true); err != nil {
		return Config{}, err
	}
	for _, file := range config.ModelFiles {
		if !strings.HasPrefix(file, ModelRoot+"/") || !strings.HasSuffix(file, ".yaml") && !strings.HasSuffix(file, ".yml") {
			return Config{}, fmt.Errorf("modelFiles entry %q must be an exact YAML file under %s/", file, ModelRoot)
		}
		if _, err := namespaceForModelPath(file); err != nil {
			return Config{}, err
		}
	}
	if err := validateExactPaths("inventoryRoots", config.InventoryRoots, false); err != nil {
		return Config{}, err
	}
	for i, root := range config.InventoryRoots {
		if strings.EqualFold(root, ".markitect") || strings.HasPrefix(strings.ToLower(root), ".markitect/") {
			return Config{}, fmt.Errorf("inventoryRoots entry %q may not include Markitect runtime, cache, model, or view files", root)
		}
		for j, other := range config.InventoryRoots {
			if i == j {
				continue
			}
			if pathWithin(root, other) || pathWithin(other, root) {
				return Config{}, fmt.Errorf("inventoryRoots entries %q and %q overlap", root, other)
			}
		}
	}
	for i, exclusion := range config.Exclusions {
		if err := validateRepoPath(exclusion.Path); err != nil {
			return Config{}, fmt.Errorf("exclusions[%d].path: %w", i, err)
		}
		if strings.TrimSpace(exclusion.Reason) == "" || exclusion.Reason != strings.TrimSpace(exclusion.Reason) {
			return Config{}, fmt.Errorf("exclusions[%d].reason must be nonempty and have no surrounding whitespace", i)
		}
		for j := 0; j < i; j++ {
			if strings.EqualFold(config.Exclusions[j].Path, exclusion.Path) {
				return Config{}, fmt.Errorf("exclusions repeats or aliases path %q", exclusion.Path)
			}
		}
	}
	return config, nil
}

func rejectProjectYAMLNode(document *yaml.Node) error {
	if document == nil || len(document.Content) != 1 {
		return fmt.Errorf("project config must contain one document")
	}
	var visit func(*yaml.Node) error
	visit = func(node *yaml.Node) error {
		if node.Kind == yaml.AliasNode || node.Alias != nil || node.Anchor != "" {
			return fmt.Errorf("project config YAML aliases and anchors are not supported")
		}
		if node.Kind == yaml.MappingNode {
			seen := map[string]bool{}
			for i := 0; i+1 < len(node.Content); i += 2 {
				key := node.Content[i]
				if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "<<" {
					return fmt.Errorf("project config mapping keys must be ordinary strings")
				}
				if seen[key.Value] {
					return fmt.Errorf("project config contains duplicate key %q", key.Value)
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
	return visit(document.Content[0])
}

func validateExactPaths(field string, values []string, _ bool) error {
	seen := map[string]string{}
	for _, value := range values {
		if err := validateRepoPath(value); err != nil {
			return fmt.Errorf("%s entry %q: %w", field, value, err)
		}
		if old, ok := seen[strings.ToLower(value)]; ok {
			return fmt.Errorf("%s contains duplicate or case-aliased paths %q and %q", field, old, value)
		}
		seen[strings.ToLower(value)] = value
	}
	return nil
}

func validateProjectYAMLFields(document *yaml.Node) error {
	root := document.Content[0]
	if err := exactYAMLFields(root, map[string]bool{"apiVersion": true, "name": true, "modelFiles": true, "inventoryRoots": true, "exclusions": true}); err != nil {
		return err
	}
	for i, exclusionList := range root.Content {
		if exclusionList.Value != "exclusions" || i+1 >= len(root.Content) {
			continue
		}
		list := root.Content[i+1]
		for _, item := range list.Content {
			if item.Kind == yaml.MappingNode {
				if err := exactYAMLFields(item, map[string]bool{"path": true, "reason": true}); err != nil {
					return fmt.Errorf("project config exclusions: %w", err)
				}
			}
		}
	}
	return nil
}

func exactYAMLFields(node *yaml.Node, allowed map[string]bool) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if !allowed[key] {
			for previous := range seen {
				if strings.EqualFold(previous, key) {
					return fmt.Errorf("project config field %q has incorrect casing or aliases %q", key, previous)
				}
			}
			return fmt.Errorf("project config contains unknown or incorrectly cased field %q", key)
		}
		seen[key] = true
	}
	return nil
}

func validateRepoPath(value string) error {
	if value == "" || value == "." || strings.ContainsAny(value, "\\:\x00") || strings.HasPrefix(value, "/") || path.Clean(value) != value {
		return fmt.Errorf("path must be a normalized repository-relative slash path")
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." || strings.EqualFold(component, ".git") {
			return fmt.Errorf("path contains an empty, traversal, or reserved component")
		}
	}
	return nil
}

func namespaceForModelPath(file string) (string, error) {
	if !strings.HasPrefix(file, ModelRoot+"/") {
		return "", fmt.Errorf("model path %q is outside %s", file, ModelRoot)
	}
	relative := strings.TrimPrefix(file, ModelRoot+"/")
	parent := path.Dir(relative)
	if parent == "." {
		return "", nil
	}
	segments := strings.Split(parent, "/")
	for _, segment := range segments {
		if !modelNamespaceSegment.MatchString(segment) {
			return "", fmt.Errorf("model path %q has invalid namespace segment %q; use lowercase kebab-case directories", file, segment)
		}
	}
	return strings.Join(segments, "."), nil
}

func pathWithin(candidate, root string) bool {
	candidate = strings.ToLower(candidate)
	root = strings.TrimSuffix(strings.ToLower(root), "/")
	return candidate == root || strings.HasPrefix(candidate, root+"/")
}
