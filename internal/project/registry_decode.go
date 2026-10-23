package project

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

func (r *registryPreparation) invalid(source snapshot.Path, node *yaml.Node, code, message string) {
	var position *model.Position
	if node != nil && node.Line > 0 {
		position = &model.Position{Line: node.Line, Column: node.Column}
	}
	r.add(model.Failed, code, source, position, message)
}

func (r *registryPreparation) registryYAML(source snapshot.Path, node *yaml.Node) {
	if node.Kind == yaml.AliasNode || node.Alias != nil || node.Anchor != "" || node.Tag != "" && !strings.HasPrefix(node.Tag, "!!") {
		r.invalid(source, node, "registry.yaml_feature", "Registry YAML cannot use aliases, anchors or custom tags.")
		return
	}
	for _, child := range node.Content {
		r.registryYAML(source, child)
	}
}

func (r *registryPreparation) fields(source snapshot.Path, node *yaml.Node, allowed ...string) map[string]*yaml.Node {
	result := make(map[string]*yaml.Node)
	if node.Kind != yaml.MappingNode {
		r.invalid(source, node, "registry.mapping", "Expected a registry mapping.")
		return result
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			r.invalid(source, key, "registry.key", "Registry keys must be strings.")
			continue
		}
		if _, exists := result[key.Value]; exists {
			r.invalid(source, key, "registry.duplicate_key", fmt.Sprintf("Duplicate registry key %q.", key.Value))
			continue
		}
		result[key.Value] = value
		known := false
		for _, field := range allowed {
			known = known || field == key.Value
		}
		if !known {
			r.invalid(source, key, "registry.unknown_key", fmt.Sprintf("Unknown registry key %q.", key.Value))
		}
	}
	return result
}

func (r *registryPreparation) text(source snapshot.Path, fields map[string]*yaml.Node, name string, parent *yaml.Node) string {
	node, found := fields[name]
	if !found {
		r.invalid(source, parent, "registry.field_missing", fmt.Sprintf("Required registry field %q is missing.", name))
		return ""
	}
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		r.invalid(source, node, "registry.string", fmt.Sprintf("Registry field %q must be a string.", name))
		return ""
	}
	return node.Value
}

func (r *registryPreparation) decodeRegistry(source SourceFile, owner realization.Owner, checked model.Result, census snapshot.Tree) []realization.Registration {
	path := source.RepositoryPath
	decoder := yaml.NewDecoder(bytes.NewReader(source.File.Bytes()))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		r.invalid(path, nil, "registry.yaml", err.Error())
		return nil
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		r.invalid(path, &extra, "registry.documents", "Registry must contain exactly one YAML document.")
	}
	if len(document.Content) != 1 {
		r.invalid(path, &document, "registry.document", "Registry must contain one mapping document.")
		return nil
	}
	r.registryYAML(path, &document)
	root := document.Content[0]
	fields := r.fields(path, root, "apiVersion", "artifacts")
	if version := r.text(path, fields, "apiVersion", root); version != "markitect.registry/v1" {
		r.invalid(path, fields["apiVersion"], "registry.version", "Registry apiVersion must be markitect.registry/v1.")
	}
	artifacts, found := fields["artifacts"]
	if !found || artifacts.Kind != yaml.SequenceNode {
		r.invalid(path, artifacts, "registry.artifacts", "Registry artifacts must be an explicit list, including [] when empty.")
		return nil
	}
	identities := make(map[model.Identity]bool)
	for _, modelResult := range checked.Models {
		for _, definition := range modelResult.Definitions {
			identities[definition.Identity] = true
		}
	}
	var registrations []realization.Registration
	for _, node := range artifacts.Content {
		if registration, ok := r.decodeRecord(path, node, owner, identities, census); ok {
			registrations = append(registrations, registration)
		}
	}
	return registrations
}
