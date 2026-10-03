// Package core defines the provider-independent Markitect resource model.
package core

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

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

// DomainDefinition registers one versioned vocabulary with the language kernel.
// It contains declarative schemas and constraints only; there are no executable hooks.
type DomainDefinition struct {
	Name        string                        `yaml:"-"`
	APIVersion  string                        `yaml:"apiVersion"`
	Kinds       map[string]KindDefinition     `yaml:"kinds"`
	Relations   map[string]RelationDefinition `yaml:"relations,omitempty"`
	Constraints []ConstraintDefinition        `yaml:"constraints,omitempty"`
	Path        string                        `yaml:"-"`
	Line        int                           `yaml:"-"`
}

func (d DomainDefinition) MarshalYAML() (any, error) {
	type domainSpec DomainDefinition
	metadata := struct {
		Name string `yaml:"name"`
	}{Name: d.Name}
	return struct {
		APIVersion string     `yaml:"apiVersion"`
		Kind       string     `yaml:"kind"`
		Metadata   any        `yaml:"metadata"`
		Spec       domainSpec `yaml:"spec"`
	}{APIVersion: APIVersion, Kind: "Domain", Metadata: metadata, Spec: domainSpec(d)}, nil
}

type KindDefinition struct {
	Required    []string                      `yaml:"required,omitempty"`
	InputsField string                        `yaml:"inputsField,omitempty"`
	Properties  map[string]PropertyDefinition `yaml:"properties"`
}

// PropertyDefinition supports bounded scalar, array, closed object and typed-ref schemas.
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

// PolicyException is a source-bound, explicit waiver for one failing
// per-resource constraint result. It carries no identity or approval proof.
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
	ExpiresOn        string `yaml:"expiresOn,omitempty"` // Exclusive; policyDate >= expiresOn is expired.
}

// PolicyResult is the deterministic evaluation of one constraint against one
// subject, or a selected collection when Subject is empty.
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
	PolicyDate       string            `yaml:"policyDate,omitempty"` // Explicit as-of date; no wall clock is consulted.
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

type AdapterConfig struct {
	Name    string         `yaml:"name"`
	Type    string         `yaml:"type"`
	Version string         `yaml:"version"`
	Config  map[string]any `yaml:"config,omitempty"`
}

// PackagePin selects one exact offline content archive in a Project.
// Source records provenance only; it is never fetched by Markitect.
type PackagePin struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Source  string `yaml:"source"`
	Archive string `yaml:"archive"`
	SHA256  string `yaml:"sha256"`
}

type Area struct {
	Name    string   `yaml:"name"`
	Path    string   `yaml:"path"`
	Imports []string `yaml:"imports,omitempty"`
	Rules   []Ref    `yaml:"rules,omitempty"`
}

type Binding struct {
	Contract       Ref `yaml:"contract"`
	Implementation Ref `yaml:"implementation"`
}

// Check describes one explicit project verification gate. Run is argv, not a
// shell command string; consumers execute it without shell interpretation.
type Check struct {
	Name string   `yaml:"name"`
	Run  []string `yaml:"run"`
}

// Documentation enables structural checks for explicitly selected local trees.
// It does not assign resource identity or declare graph dependencies.
type Documentation struct {
	Roots []string `yaml:"roots"`
}

type Provider struct {
	Model           string   `yaml:"model,omitempty"`
	Effort          string   `yaml:"effort,omitempty"`
	Sandbox         string   `yaml:"sandbox,omitempty"`
	PermissionMode  string   `yaml:"permissionMode,omitempty"`
	Tools           []string `yaml:"tools,omitempty"`
	DisallowedTools []string `yaml:"disallowedTools,omitempty"`
	MaxTurns        int      `yaml:"maxTurns,omitempty"`
}

type Providers struct {
	Codex  *Provider `yaml:"codex,omitempty"`
	Claude *Provider `yaml:"claude,omitempty"`
}

// ProviderAdapters configures additional, explicitly owned provider entrypoints.
// Paths are canonical repository inputs, never discovered from prose links.
type ProviderAdapters struct {
	RuleSources     map[string][]string `yaml:"ruleSources,omitempty"`
	AgentContract   string              `yaml:"agentContract,omitempty"`
	RoleRegister    string              `yaml:"roleRegister,omitempty"`
	InlineAgentText bool                `yaml:"inlineAgentText,omitempty"`
	StrictInventory bool                `yaml:"strictInventory,omitempty"`
	RetiredSkills   []string            `yaml:"retiredSkills,omitempty"`
	RetiredAgents   []string            `yaml:"retiredAgents,omitempty"`
}

// Assertion is an explicit functional fact. Only predicates listed by the
// Project as functional are compared; text is never interpreted as a fact.
type Assertion struct {
	Subject   string `yaml:"subject"`
	Predicate string `yaml:"predicate"`
	Value     string `yaml:"value"`
	Source    string `yaml:"source"`
	Quote     string `yaml:"quote"`
}

type Consistency struct {
	FunctionalPredicates []string `yaml:"functionalPredicates,omitempty"`
}

type Spec struct {
	Text string `yaml:"text,omitempty"`
	// Files lists explicit repository-relative non-Markitect inputs needed by this resource.
	Files            []string          `yaml:"files,omitempty"`
	Description      string            `yaml:"description,omitempty"`
	Rules            []Ref             `yaml:"rules,omitempty"`
	Uses             []Ref             `yaml:"uses,omitempty"`
	Needs            []Ref             `yaml:"needs,omitempty"`
	Implements       []Ref             `yaml:"implements,omitempty"`
	Input            []string          `yaml:"input,omitempty"`
	Output           []string          `yaml:"output,omitempty"`
	Kind             string            `yaml:"kind,omitempty"`
	Check            string            `yaml:"check,omitempty"`
	Providers        Providers         `yaml:"providers,omitempty"`
	Assertions       []Assertion       `yaml:"assertions,omitempty"`
	Targets          []string          `yaml:"targets,omitempty"`
	Areas            []Area            `yaml:"areas,omitempty"`
	Bindings         []Binding         `yaml:"bindings,omitempty"`
	Checks           []Check           `yaml:"checks,omitempty"`
	Documentation    *Documentation    `yaml:"documentation,omitempty"`
	Domains          []string          `yaml:"domains,omitempty"`
	Adapters         []AdapterConfig   `yaml:"adapters,omitempty"`
	PolicyDate       string            `yaml:"policyDate,omitempty"`
	PolicyExceptions []PolicyException `yaml:"policyExceptions,omitempty"`
	// Rules map provider rule entrypoint names to owning sources. This also
	// supports several canonical sources behind one legacy entrypoint.
	RuleAdapters     map[string][]Ref  `yaml:"ruleAdapters,omitempty"`
	ProviderAdapters *ProviderAdapters `yaml:"providerAdapters,omitempty"`
	Consistency      *Consistency      `yaml:"consistency,omitempty"`
	// Packages pins the direct offline content archives available to this Project.
	Packages []PackagePin `yaml:"packages,omitempty"`
	// Version identifies a Package manifest's exact content version.
	Version string `yaml:"version,omitempty"`
	// Exports declares the package-local resource identities consumers may reference.
	Exports []Ref `yaml:"exports,omitempty"`
}

type Resource struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	// Data is the normalized canonical spec for all resource kinds.
	Data map[string]any `yaml:"-"`
	// Spec is a lowering view retained for the built-in AI and bootstrap domain.
	Spec Spec   `yaml:"-"`
	Path string `yaml:"-"`
	Line int    `yaml:"-"`
	// Package is the runtime origin package ID; it is not serialized in resource YAML.
	Package string `yaml:"-"`
}

func (r Resource) MarshalYAML() (any, error) {
	spec := any(r.Spec)
	if r.Data != nil && r.APIVersion != APIVersion {
		spec = r.Data
	}
	return struct {
		APIVersion string   `yaml:"apiVersion"`
		Kind       string   `yaml:"kind"`
		Metadata   Metadata `yaml:"metadata"`
		Spec       any      `yaml:"spec"`
	}{r.APIVersion, r.Kind, r.Metadata, spec}, nil
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

// GraphKey identifies a resource within its origin while preserving Key as
// the canonical package-local namespace/kind/name identity.
func (r Resource) GraphKey() string {
	packageName := r.Package
	if r.Kind == "Package" && packageName == "" {
		packageName = r.Metadata.Name
	}
	return Ref{APIVersion: r.APIVersion, Kind: r.Kind, Namespace: r.Metadata.Namespace, Name: r.Metadata.Name, Package: packageName}.GraphKey("", "", "")
}

// GraphKey returns the canonical graph identity for a reference. An explicit
// package wins; otherwise the declaring resource's package is used.
func (r Ref) GraphKey(defaultPackage, namespace, kind string) string {
	if r.Package != "" {
		defaultPackage = r.Package
	}
	key := r.Key(namespace, kind)
	if defaultPackage == "" {
		return key
	}
	return defaultPackage + "::" + key
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

// PolicyResultRef identifies an ordinary policy evaluation that produced a
// diagnostic. It is deliberately a reference rather than an embedded result,
// so diagnostics do not create a second owner for policy state.
type PolicyResultRef struct {
	APIVersion string `yaml:"apiVersion"`
	Constraint string `yaml:"constraint"`
	Subject    string `yaml:"subject"`
}

type Graph struct {
	Resources          map[string]*Resource
	Packages           map[string]*Resource
	Edges              map[string][]string
	Relationships      []Relationship
	ResourceAreas      map[string]Area
	Project            *Resource
	Registry           *Registry
	Diagnostics        []Diagnostic
	InvalidationEdges  map[string][]string
	PolicyResults      []PolicyResult
	PolicyDependencies []PolicyDependency
	invalidPolicyPaths map[string]bool
}

// Relationship records why a resolved graph edge exists. Edges remains the
// compact adjacency view used by existing context and impact logic.
type Relationship struct {
	From             string `yaml:"from"`
	To               string `yaml:"to"`
	Relation         string `yaml:"relation"`
	DomainAPIVersion string `yaml:"domainApiVersion,omitempty"`
	Path             string `yaml:"path,omitempty"`
	Line             int    `yaml:"resourceLine,omitempty"` // Start of the declaring resource, not the nested reference.
	Reference        Ref    `yaml:"reference,omitempty"`
	Area             string `yaml:"area,omitempty"`
	Selected         bool   `yaml:"selected,omitempty"`
	Context          bool   `yaml:"context"`
	Invalidate       bool   `yaml:"invalidate"`
	Acyclic          bool   `yaml:"acyclic"`
}

// IdentityVersion removes the patch/minor version suffix from API version
// identity where the group is conventional group/version. Active versions of
// the same group/kind/name are rejected rather than treated as separate facts.
func (r Resource) IdentityVersion() string {
	parts := strings.Split(r.APIVersion, "/")
	if len(parts) == 2 {
		return parts[0]
	}
	return r.APIVersion
}

// SetData attaches normalized canonical data and lowers built-in resources.
func (r *Resource) SetData(data map[string]any) error {
	if data == nil {
		return fmt.Errorf("resource data must be an object")
	}
	r.Data = data
	if r.APIVersion == APIVersion {
		encoded, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		return yaml.Unmarshal(encoded, &r.Spec)
	}
	return nil
}
