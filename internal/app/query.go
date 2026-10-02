package app

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

// FindQuery selects resources using a literal, case-insensitive substring and
// optional exact kind and namespace filters. An empty Query lists all matches.
type FindQuery struct {
	APIVersion string
	Query      string
	Kind       string
	Namespace  string
	Package    string
}

// FindMatch is a concise resource result; canonical prose is not returned.
// It aliases ResourceSummary so search and relationship identities share one
// public shape.
type FindMatch = ResourceSummary

// ExplainResult describes one exact resource and its directly resolved graph
// relationships. Results are unavailable while project diagnostics remain.
type ExplainResult struct {
	APIVersion              string              `yaml:"apiVersion,omitempty"`
	Key                     string              `yaml:"key"`
	Kind                    string              `yaml:"kind"`
	Name                    string              `yaml:"name"`
	Namespace               string              `yaml:"namespace,omitempty"`
	Package                 string              `yaml:"package,omitempty"`
	PackageVersion          string              `yaml:"packageVersion,omitempty"`
	Path                    string              `yaml:"path"`
	Description             string              `yaml:"description,omitempty"`
	Area                    *core.Area          `yaml:"area,omitempty"`
	Outgoing                []core.Relationship `yaml:"outgoing"`
	Incoming                []core.Relationship `yaml:"incoming"`
	DeclaredImplementations []ResourceSummary   `yaml:"declaredImplementations,omitempty"`
	SelectedImplementation  *ResourceSummary    `yaml:"selectedImplementation,omitempty"`
	DeclaredFiles           []string            `yaml:"declaredFiles,omitempty"`
}

// ResourceSummary is a concise identity for a related resource.
type ResourceSummary struct {
	APIVersion     string `yaml:"apiVersion,omitempty"`
	Key            string `yaml:"key"`
	Kind           string `yaml:"kind"`
	Name           string `yaml:"name"`
	Namespace      string `yaml:"namespace,omitempty"`
	Package        string `yaml:"package,omitempty"`
	PackageVersion string `yaml:"packageVersion,omitempty"`
	Path           string `yaml:"path"`
	Description    string `yaml:"description,omitempty"`
}

// Find searches resource identity, path, description and body using literal
// case-insensitive matching. It neither performs fuzzy matching nor returns
// the body prose.
func Find(p *Project, query FindQuery) ([]FindMatch, error) {
	if err := queryableProject(p); err != nil {
		return nil, err
	}
	needle := strings.ToLower(query.Query)
	keys := make([]string, 0, len(p.Graph.Resources))
	for key := range p.Graph.Resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	matches := make([]FindMatch, 0)
	for _, key := range keys {
		resource := p.Graph.Resources[key]
		if !p.exported(resource) || (query.Package != "" && resource.Package != query.Package) {
			continue
		}
		if query.Kind != "" && resource.Kind != query.Kind {
			continue
		}
		if query.APIVersion != "" && resource.APIVersion != query.APIVersion {
			continue
		}
		if query.Namespace != "" && resource.Metadata.Namespace != query.Namespace {
			continue
		}
		body := resource.Spec.Text
		if resource.Data != nil {
			encoded, err := YAML(resource.Data)
			if err != nil {
				return nil, err
			}
			body = string(encoded)
		}
		searchable := strings.ToLower(strings.Join([]string{
			key, resource.Kind, resource.Metadata.Name, resource.Metadata.Namespace,
			resource.Path, resourceDescription(resource), body,
		}, "\n"))
		if needle != "" && !strings.Contains(searchable, needle) {
			continue
		}
		matches = append(matches, p.summarize(resource))
	}
	return matches, nil
}

// Explain resolves an exact resource identity and reports its longest owning
// area, direct relationships with provenance, and Contract implementation
// declarations and selection when the resource is a Contract.
func Explain(p *Project, key string) (*ExplainResult, error) {
	if err := queryableProject(p); err != nil {
		return nil, err
	}
	resource := p.Graph.Resources[key]
	if resource == nil {
		return nil, fmt.Errorf("unknown resource %s", key)
	}
	if !p.exported(resource) {
		return nil, fmt.Errorf("package entry %s is not exported", key)
	}
	result := &ExplainResult{
		Key: key, Kind: resource.Kind, Name: resource.Metadata.Name,
		Namespace: resource.Metadata.Namespace, Path: resource.Path,
		Package: resource.Package, PackageVersion: p.packageVersion(resource.Package),
		APIVersion:  extensionAPI(resource),
		Description: resourceDescription(resource),
		Outgoing:    make([]core.Relationship, 0), Incoming: make([]core.Relationship, 0),
	}
	if area, ok := p.Graph.ResourceAreas[key]; ok {
		areaCopy := area
		result.Area = &areaCopy
	}
	for _, relation := range p.Graph.Relationships {
		if relation.From == key {
			result.Outgoing = append(result.Outgoing, relation)
		}
		if relation.To == key {
			result.Incoming = append(result.Incoming, relation)
		}
		if resource.Kind != "Contract" || relation.To != key {
			continue
		}
		implementation := p.Graph.Resources[relation.From]
		if implementation == nil {
			continue
		}
		switch relation.Relation {
		case "implements":
			result.DeclaredImplementations = append(result.DeclaredImplementations, p.summarize(implementation))
		case "binding":
			selected := p.summarize(implementation)
			result.SelectedImplementation = &selected
		}
	}
	if len(result.DeclaredImplementations) > 1 {
		sort.Slice(result.DeclaredImplementations, func(i, j int) bool {
			return result.DeclaredImplementations[i].Key < result.DeclaredImplementations[j].Key
		})
		unique := result.DeclaredImplementations[:1]
		for _, candidate := range result.DeclaredImplementations[1:] {
			if candidate.Key != unique[len(unique)-1].Key {
				unique = append(unique, candidate)
			}
		}
		result.DeclaredImplementations = unique
	}
	if files := p.InputFiles[key]; len(files) > 0 {
		result.DeclaredFiles = append(result.DeclaredFiles, files...)
		sort.Strings(result.DeclaredFiles)
	}
	return result, nil
}

func queryableProject(p *Project) error {
	if p == nil || p.Graph == nil {
		return errors.New("project graph is required")
	}
	if len(p.Diagnostics) != 0 || len(p.Graph.Diagnostics) != 0 {
		return errors.New("resource queries are unavailable while project diagnostics remain")
	}
	return nil
}

func (p *Project) summarize(resource *core.Resource) ResourceSummary {
	return ResourceSummary{
		Key: resource.GraphKey(), Kind: resource.Kind, Name: resource.Metadata.Name,
		Namespace: resource.Metadata.Namespace, Path: resource.Path,
		Package: resource.Package, PackageVersion: p.packageVersion(resource.Package),
		APIVersion:  extensionAPI(resource),
		Description: resourceDescription(resource),
	}
}

func extensionAPI(resource *core.Resource) string {
	if resource.APIVersion == core.APIVersion {
		return ""
	}
	return resource.APIVersion
}

func resourceDescription(resource *core.Resource) string {
	if value, ok := resource.Data["description"].(string); ok {
		return value
	}
	return resource.Spec.Description
}
