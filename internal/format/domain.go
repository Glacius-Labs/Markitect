package format

import (
	"bytes"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

// ParseDomain reads a strict kernel Domain envelope. The returned descriptor
// carries its authoring name and source provenance outside its serialized spec.
func ParseDomain(filePath string, data []byte) (core.DomainDefinition, error) {
	if len(data) > maxResourceSize {
		return core.DomainDefinition{}, diagnostic(filePath, 1, "domain resource exceeds the 2 MiB limit")
	}
	doc, err := parseOne(filePath, data)
	if err != nil {
		return core.DomainDefinition{}, err
	}
	root := doc.Content[0]
	if err := requireMapping(filePath, root, "domain resource"); err != nil {
		return core.DomainDefinition{}, err
	}
	if err := checkKeys(filePath, root, set("apiVersion", "kind", "metadata", "spec")); err != nil {
		return core.DomainDefinition{}, err
	}
	if err := requireFields(filePath, root, "apiVersion", "kind", "metadata", "spec"); err != nil {
		return core.DomainDefinition{}, err
	}
	if child(root, "apiVersion").Value != core.APIVersion {
		return core.DomainDefinition{}, diagnostic(filePath, child(root, "apiVersion").Line, "Domain envelope apiVersion must be %q", core.APIVersion)
	}
	if child(root, "kind").Value != "Domain" {
		return core.DomainDefinition{}, diagnostic(filePath, child(root, "kind").Line, "expected kind Domain")
	}
	if err := validateMetadata(filePath, child(root, "metadata"), "Domain"); err != nil {
		return core.DomainDefinition{}, err
	}
	specNode := child(root, "spec")
	if err := requireMapping(filePath, specNode, "Domain spec"); err != nil {
		return core.DomainDefinition{}, err
	}
	if err := checkKeys(filePath, specNode, set("apiVersion", "kinds", "relations", "constraints")); err != nil {
		return core.DomainDefinition{}, err
	}
	encodedData, err := yaml.Marshal(specNode)
	if err != nil {
		return core.DomainDefinition{}, diagnostic(filePath, specNode.Line, "invalid Domain spec: %v", err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(encodedData))
	dec.KnownFields(true)
	var def core.DomainDefinition
	if err := dec.Decode(&def); err != nil {
		return core.DomainDefinition{}, diagnostic(filePath, yamlErrorLine(err), "invalid Domain spec: %v", err)
	}
	def.Name = child(child(root, "metadata"), "name").Value
	def.Path = filePath
	def.Line = root.Line
	registry := core.NewRegistry()
	if err := registry.AddDomain(def); err != nil {
		return core.DomainDefinition{}, diagnostic(filePath, specNode.Line, "invalid Domain definition: %v", err)
	}
	return def, nil
}

// EncodeDomain serializes the full Domain authoring envelope deterministically.
func EncodeDomain(domain core.DomainDefinition) ([]byte, error) {
	if domain.Name == "" {
		return nil, fmt.Errorf("Domain name is required")
	}
	return yaml.Marshal(domain)
}

func parseOne(filePath string, data []byte) (*yaml.Node, error) {
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
	if err.Error() != "EOF" {
		return nil, diagnostic(filePath, yamlErrorLine(err), "invalid YAML: %v", err)
	}
	return &doc, nil
}
