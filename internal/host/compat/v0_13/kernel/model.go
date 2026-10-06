// Package core defines the normalized, provider-independent semantic IR.
package core

import "strings"

const APIVersion = "markitect.example.org/v1alpha1"

const (
	PolicyPassed = "passed"
	PolicyFailed = "failed"
	PolicyWaived = "waived"
)

type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}
type Ref struct {
	APIVersion string `yaml:"apiVersion,omitempty"`
	Kind       string `yaml:"kind,omitempty"`
	Name       string `yaml:"name"`
	Namespace  string `yaml:"namespace,omitempty"`
	Package    string `yaml:"package,omitempty"`
}

type DomainDefinition struct {
	Name        string                        `yaml:"-"`
	APIVersion  string                        `yaml:"apiVersion"`
	Kinds       map[string]KindDefinition     `yaml:"kinds"`
	Relations   map[string]RelationDefinition `yaml:"relations,omitempty"`
	Constraints []ConstraintDefinition        `yaml:"constraints,omitempty"`
	Path        string                        `yaml:"-"`
	Line        int                           `yaml:"-"`
}
type KindDefinition struct {
	Required    []string                      `yaml:"required,omitempty"`
	InputsField string                        `yaml:"inputsField,omitempty"`
	Properties  map[string]PropertyDefinition `yaml:"properties"`
}
type PropertyDefinition struct {
	Type       string                        `yaml:"type"`
	Items      *PropertyDefinition           `yaml:"items,omitempty"`
	Properties map[string]PropertyDefinition `yaml:"properties,omitempty"`
	Required   []string                      `yaml:"required,omitempty"`
	Enum       []any                         `yaml:"enum,omitempty"`
	RefKind    string                        `yaml:"refKind,omitempty"`
}
type RelationDefinition struct {
	Description string   `yaml:"description,omitempty"`
	Field       string   `yaml:"field"`
	SourceKinds []string `yaml:"sourceKinds"`
	TargetKinds []string `yaml:"targetKinds"`
	MinTargets  *int     `yaml:"minTargets,omitempty"`
	MaxTargets  *int     `yaml:"maxTargets,omitempty"`
	Context     bool     `yaml:"context,omitempty"`
	Invalidate  bool     `yaml:"invalidate,omitempty"`
	Acyclic     bool     `yaml:"acyclic,omitempty"`
}
type ResourceSelector struct {
	Kind   string            `yaml:"kind,omitempty"`
	Labels map[string]string `yaml:"labels,omitempty"`
}
type ConstraintDefinition struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description,omitempty"`
	Select      ResourceSelector    `yaml:"select"`
	Assert      ConstraintAssertion `yaml:"assert"`
}
type ConstraintAssertion struct {
	Op       string   `yaml:"op"`
	Scope    string   `yaml:"scope,omitempty"`
	Field    string   `yaml:"field,omitempty"`
	Relation string   `yaml:"relation,omitempty"`
	Value    any      `yaml:"value,omitempty"`
	Values   []any    `yaml:"values,omitempty"`
	Min      *int     `yaml:"min,omitempty"`
	Max      *int     `yaml:"max,omitempty"`
	Left     []string `yaml:"left,omitempty"`
	Right    []string `yaml:"right,omitempty"`
}

// PolicyException is source-supplied evidence for one exact failing result. All
// fields are opaque to Core except the generic identity, digests and expiry.
type PolicyException struct {
	Name             string `yaml:"name"`
	APIVersion       string `yaml:"apiVersion"`
	Constraint       string `yaml:"constraint"`
	Subject          string `yaml:"subject"`
	ConstraintDigest string `yaml:"constraintDigest"`
	SubjectDigest    string `yaml:"subjectDigest"`
	Rationale        string `yaml:"rationale"`
	Owner            string `yaml:"owner"`
	Decision         string `yaml:"decision"`
	ExpiresOn        string `yaml:"expiresOn,omitempty"`
}
type PolicyResult struct {
	APIVersion       string            `yaml:"apiVersion"`
	Constraint       string            `yaml:"constraint"`
	Subject          string            `yaml:"subject"`
	Status           string            `yaml:"status"`
	Message          string            `yaml:"message,omitempty"`
	ConstraintDigest string            `yaml:"constraintDigest"`
	SubjectDigest    string            `yaml:"subjectDigest"`
	ExceptionName    string            `yaml:"exceptionName,omitempty"`
	Rationale        string            `yaml:"rationale,omitempty"`
	Owner            string            `yaml:"owner,omitempty"`
	Decision         string            `yaml:"decision,omitempty"`
	ExpiresOn        string            `yaml:"expiresOn,omitempty"`
	PolicyDate       string            `yaml:"policyDate,omitempty"`
	Comparison       *TargetComparison `yaml:"comparison,omitempty"`
}
type TargetComparison struct {
	Left  TargetPath `yaml:"left"`
	Right TargetPath `yaml:"right"`
}
type TargetPath struct {
	Relations []string     `yaml:"relations"`
	Steps     []TargetStep `yaml:"steps"`
	Target    string       `yaml:"target"`
}
type TargetStep struct {
	From             string `yaml:"from"`
	To               string `yaml:"to"`
	Relation         string `yaml:"relation"`
	DomainAPIVersion string `yaml:"domainApiVersion"`
	Path             string `yaml:"path,omitempty"`
	Line             int    `yaml:"line,omitempty"`
}
type PolicyDependency struct {
	Subject    string `yaml:"subject"`
	Input      string `yaml:"input"`
	APIVersion string `yaml:"apiVersion"`
	Constraint string `yaml:"constraint"`
	Relation   string `yaml:"relation"`
	Path       string `yaml:"path,omitempty"`
	Line       int    `yaml:"line,omitempty"`
}

// Resource carries one normalized canonical value. Package is only an opaque
// origin label; Path and Line are opaque provenance and are never interpreted.
type Resource struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   Metadata       `yaml:"metadata"`
	Data       map[string]any `yaml:"-"`
	Path       string         `yaml:"-"`
	Line       int            `yaml:"-"`
	Package    string         `yaml:"-"`
}

func (r Resource) Key() string {
	return r.Metadata.Namespace + "/" + r.TypeKey() + "/" + r.Metadata.Name
}
func (r Resource) TypeKey() string {
	if r.APIVersion == APIVersion || r.APIVersion == "" {
		return r.Kind
	}
	return r.APIVersion + "/" + r.Kind
}
func (r Resource) GraphKey() string {
	return Ref{APIVersion: r.APIVersion, Kind: r.Kind, Namespace: r.Metadata.Namespace, Name: r.Metadata.Name, Package: r.Package}.GraphKey("", "", "")
}
func (r Ref) GraphKey(defaultOrigin, namespace, kind string) string {
	if r.Package != "" {
		defaultOrigin = r.Package
	}
	key := r.Key(namespace, kind)
	if defaultOrigin == "" {
		return key
	}
	return defaultOrigin + "::" + key
}
func (r Ref) Key(namespace, kind string) string {
	if r.Namespace != "" {
		namespace = r.Namespace
	}
	if r.Kind != "" {
		kind = r.Kind
	}
	if r.APIVersion != "" && r.APIVersion != APIVersion {
		kind = r.APIVersion + "/" + kind
	}
	return namespace + "/" + kind + "/" + r.Name
}

type Diagnostic struct {
	Code         string           `yaml:"code"`
	Path         string           `yaml:"path,omitempty"`
	Package      string           `yaml:"package,omitempty"`
	Line         int              `yaml:"line,omitempty"`
	Message      string           `yaml:"message"`
	PolicyResult *PolicyResultRef `yaml:"policyResult,omitempty"`
}
type PolicyResultRef struct {
	APIVersion string `yaml:"apiVersion"`
	Constraint string `yaml:"constraint"`
	Subject    string `yaml:"subject"`
}
type Graph struct {
	Resources              map[string]*Resource
	Edges                  map[string][]string
	Relationships          []Relationship
	Registry               *Registry
	Diagnostics            []Diagnostic
	InvalidationEdges      map[string][]string
	PolicyResults          []PolicyResult
	PolicyDependencies     []PolicyDependency
	invalidPolicyPaths     map[string]bool
	subjectDigestEncodings map[string][]byte
}
type Relationship struct {
	From             string `yaml:"from"`
	To               string `yaml:"to"`
	Relation         string `yaml:"relation"`
	DomainAPIVersion string `yaml:"domainApiVersion,omitempty"`
	Path             string `yaml:"path,omitempty"`
	Line             int    `yaml:"resourceLine,omitempty"`
	Reference        Ref    `yaml:"reference,omitempty"`
	Selected         bool   `yaml:"selected,omitempty"`
	Context          bool   `yaml:"context"`
	Invalidate       bool   `yaml:"invalidate"`
	Acyclic          bool   `yaml:"acyclic"`
}

func (r Resource) IdentityVersion() string {
	parts := strings.Split(r.APIVersion, "/")
	if len(parts) == 2 {
		return parts[0]
	}
	return r.APIVersion
}
