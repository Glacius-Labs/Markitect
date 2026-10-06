// Package core owns the normalized semantic contract consumed by Modules.
package core

const SemanticModelVersion = "markitect.example.org/semantic-model/v1alpha1"

// SemanticModel is the stable, serialization-independent boundary consumed by
// adapters. It excludes authoring YAML and the legacy typed Spec projection.
type SemanticModel struct {
	APIVersion       string              `yaml:"apiVersion"`
	Snapshot         ModelSnapshot       `yaml:"snapshot"`
	ConfigDigest     string              `yaml:"configDigest"`
	ModelDigest      string              `yaml:"modelDigest"`
	ValidationStatus string              `yaml:"validationStatus"`
	StructuralStatus string              `yaml:"structuralStatus"`
	PolicyStatus     string              `yaml:"policyStatus"`
	Diagnostics      []Diagnostic        `yaml:"diagnostics,omitempty"`
	PolicyResults    []PolicyResult      `yaml:"policyResults,omitempty"`
	DomainInputs     []ModelDomainInput  `yaml:"domainInputs,omitempty"`
	Domains          []ModelDomain       `yaml:"domains,omitempty"`
	Resources        []ModelResource     `yaml:"resources"`
	Relationships    []ModelRelationship `yaml:"relationships,omitempty"`
}

type ModelSnapshot struct {
	ID          string `yaml:"id,omitempty"`
	Provisional bool   `yaml:"provisional"`
	Digest      string `yaml:"digest"`
}

type ModelResource struct {
	Identity ModelIdentity     `yaml:"identity"`
	Labels   map[string]string `yaml:"labels,omitempty"`
	Data     map[string]any    `yaml:"data"`
	Source   ModelSource       `yaml:"source"`
	Area     string            `yaml:"area,omitempty"`
}

type ModelIdentity struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Namespace  string `yaml:"namespace,omitempty"`
	Name       string `yaml:"name"`
	Package    string `yaml:"package,omitempty"`
	Key        string `yaml:"key"`
}

type ModelSource struct {
	Path   string `yaml:"path"`
	Line   int    `yaml:"line,omitempty"`
	Digest string `yaml:"digest"`
}

type ModelRelationship struct {
	From       string      `yaml:"from"`
	To         string      `yaml:"to"`
	Type       string      `yaml:"type"`
	Source     ModelSource `yaml:"source"`
	Context    bool        `yaml:"context"`
	Invalidate bool        `yaml:"invalidate"`
	Acyclic    bool        `yaml:"acyclic"`
}

type ModelDomainInput struct {
	APIVersion     string `yaml:"apiVersion"`
	Name           string `yaml:"name"`
	Path           string `yaml:"path"`
	Package        string `yaml:"package,omitempty"`
	PackageVersion string `yaml:"packageVersion,omitempty"`
	Digest         string `yaml:"digest"`
}

// ModelDomain carries normalized schema and policy descriptors without the
// authoring resource envelope or its custom YAML marshaler.
type ModelDomain struct {
	Name        string                        `yaml:"name"`
	APIVersion  string                        `yaml:"apiVersion"`
	Kinds       map[string]KindDefinition     `yaml:"kinds"`
	Relations   map[string]RelationDefinition `yaml:"relations,omitempty"`
	Constraints []ConstraintDefinition        `yaml:"constraints,omitempty"`
}
