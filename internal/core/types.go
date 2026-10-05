// Package core defines the source-independent structural language and compiler.
// It has no knowledge of project workflows, policies, providers, or I/O.
package core

import "encoding/json"

const (
	TypeString        = "string"
	TypeBoolean       = "boolean"
	TypeInteger       = "integer"
	TypeNumber        = "number"
	TypeEnum          = "enum"
	TypeObject        = "object"
	TypeReference     = "reference"
	TypeKindReference = "kindReference"

	// Unbounded is the maxCount sentinel for repeated properties. Compilation
	// still enforces MaxValuesPerProperty so caller-controlled inputs stay bounded.
	Unbounded = -1

	MaxSchemas             = 256
	MaxDefinitions         = 10_000
	MaxPropertiesPerObject = 2_048
	MaxObjectDepth         = 32
	MaxValuesPerProperty   = 10_000
	MaxDefinitionBytes     = 8 << 20
	MaxSchemaBytes         = 8 << 20
)

// KindIdentity is the nominal identity of a Kind in one exact Schema version.
type KindIdentity struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
}

// DefinitionIdentity is the complete canonical identity of one Definition.
type DefinitionIdentity struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
	Namespace  string `json:"namespace" yaml:"namespace"`
	Name       string `json:"name" yaml:"name"`
}

// Key returns an unambiguous, deterministic representation suitable for map
// keys and graph endpoints. JSON string escaping keeps user text from colliding.
func (i DefinitionIdentity) Key() string {
	data, _ := json.Marshal([4]string{i.APIVersion, i.Kind, i.Namespace, i.Name})
	return string(data)
}

// KindKey returns an unambiguous key for a nominal Kind identity.
func (i KindIdentity) Key() string {
	data, _ := json.Marshal([2]string{i.APIVersion, i.Kind})
	return string(data)
}

// Source carries opaque input provenance. It is not part of semantic identity
// or the canonical model digest.
type Source struct {
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
	Digest string `json:"digest,omitempty" yaml:"digest,omitempty"`
	Line   int    `json:"line,omitempty" yaml:"line,omitempty"`
}

// Schema declares one exact API version and its nominal Kinds.
type Schema struct {
	APIVersion string          `json:"apiVersion" yaml:"apiVersion"`
	Purpose    string          `json:"purpose" yaml:"purpose"`
	Kinds      map[string]Kind `json:"kinds" yaml:"kinds"`
	Source     Source          `json:"source" yaml:"source"`
}

// Kind declares the closed Property contract for Definitions of its identity.
type Kind struct {
	Purpose    string              `json:"purpose" yaml:"purpose"`
	Properties map[string]Property `json:"properties" yaml:"properties"`
}

// Property declares a typed value and its cardinality. MaxCount is Unbounded
// for an unbounded list (still subject to compiler safety limits).
type Property struct {
	Purpose    string              `json:"purpose" yaml:"purpose"`
	Type       string              `json:"type" yaml:"type"`
	MinCount   int                 `json:"minCount" yaml:"minCount"`
	MaxCount   int                 `json:"maxCount" yaml:"maxCount"`
	Values     []string            `json:"values,omitempty" yaml:"values,omitempty"`
	Target     *KindIdentity       `json:"target,omitempty" yaml:"target,omitempty"`
	Properties map[string]Property `json:"properties,omitempty" yaml:"properties,omitempty"`
}

// Metadata is the identity-bearing metadata shared by Definition declarations.
type Metadata struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

// Definition is one desired canonical value. Spec is validated and deep-copied
// by Compile; callers should use the returned Model as immutable compiled data.
type Definition struct {
	APIVersion string         `json:"apiVersion" yaml:"apiVersion"`
	Kind       string         `json:"kind" yaml:"kind"`
	Metadata   Metadata       `json:"metadata" yaml:"metadata"`
	Purpose    string         `json:"purpose" yaml:"purpose"`
	Spec       map[string]any `json:"spec" yaml:"spec"`
	Source     Source         `json:"source" yaml:"source"`
}

// Model is the deterministic normalized result of compiling explicitly supplied
// Schemas and Definitions. Revision is source evidence and is excluded from Digest.
type Model struct {
	Revision    string       `json:"revision,omitempty" yaml:"revision,omitempty"`
	Digest      string       `json:"digest" yaml:"digest"`
	Schemas     []Schema     `json:"schemas" yaml:"schemas"`
	Definitions []Definition `json:"definitions" yaml:"definitions"`
	Edges       []Edge       `json:"edges" yaml:"edges"`
}

// Edge is a resolved reference Property fact; it carries no traversal or policy
// semantics beyond source Definition, exact Property path, and target Definition.
type Edge struct {
	From     string `json:"from" yaml:"from"`
	To       string `json:"to" yaml:"to"`
	Property string `json:"property" yaml:"property"`
	Source   Source `json:"source" yaml:"source"`
}

// Diagnostic describes one deterministic structural compilation finding.
type Diagnostic struct {
	Code     string `json:"code" yaml:"code"`
	Identity string `json:"identity,omitempty" yaml:"identity,omitempty"`
	Property string `json:"property,omitempty" yaml:"property,omitempty"`
	Message  string `json:"message" yaml:"message"`
	Source   Source `json:"source,omitempty" yaml:"source,omitempty"`
}

// Identity returns the complete identity of this Definition.
func (d Definition) Identity() DefinitionIdentity {
	return DefinitionIdentity{APIVersion: d.APIVersion, Kind: d.Kind, Namespace: d.Metadata.Namespace, Name: d.Metadata.Name}
}

func (m Model) Definition(identity DefinitionIdentity) (Definition, bool) {
	key := identity.Key()
	for _, definition := range m.Definitions {
		if (DefinitionIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind, Namespace: definition.Metadata.Namespace, Name: definition.Metadata.Name}).Key() == key {
			return cloneDefinition(definition), true
		}
	}
	return Definition{}, false
}

// Kind returns the matching Kind contract, if present.
func (m Model) Kind(identity KindIdentity) (Kind, bool) {
	for _, schema := range m.Schemas {
		if schema.APIVersion == identity.APIVersion {
			kind, ok := schema.Kinds[identity.Kind]
			if ok {
				return cloneKind(kind), true
			}
		}
	}
	return Kind{}, false
}
