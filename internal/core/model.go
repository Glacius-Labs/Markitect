// Package core defines the provider-independent Markitect resource model.
package core

const APIVersion = "markitect.example.org/v1alpha1"

type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
}

type Ref struct {
	Kind      string `yaml:"kind,omitempty"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
	Package   string `yaml:"package,omitempty"`
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

type Spec struct {
	Text string `yaml:"text,omitempty"`
	// Files lists explicit repository-relative non-Markitect inputs needed by this resource.
	Files       []string  `yaml:"files,omitempty"`
	Description string    `yaml:"description,omitempty"`
	Rules       []Ref     `yaml:"rules,omitempty"`
	Uses        []Ref     `yaml:"uses,omitempty"`
	Needs       []Ref     `yaml:"needs,omitempty"`
	Implements  []Ref     `yaml:"implements,omitempty"`
	Input       []string  `yaml:"input,omitempty"`
	Output      []string  `yaml:"output,omitempty"`
	Kind        string    `yaml:"kind,omitempty"`
	Check       string    `yaml:"check,omitempty"`
	Providers   Providers `yaml:"providers,omitempty"`
	Targets     []string  `yaml:"targets,omitempty"`
	Areas       []Area    `yaml:"areas,omitempty"`
	Bindings    []Binding `yaml:"bindings,omitempty"`
	Checks      []Check   `yaml:"checks,omitempty"`
	// Rules map provider rule entrypoint names to owning sources. This also
	// supports several canonical sources behind one legacy entrypoint.
	RuleAdapters map[string][]Ref `yaml:"ruleAdapters,omitempty"`
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
	Spec       Spec     `yaml:"spec"`
	Path       string   `yaml:"-"`
	Line       int      `yaml:"-"`
	// Package is the runtime origin package ID; it is not serialized in resource YAML.
	Package string `yaml:"-"`
}

func (r Resource) Key() string { return r.Metadata.Namespace + "/" + r.Kind + "/" + r.Metadata.Name }

// GraphKey identifies a resource within its origin while preserving Key as
// the canonical package-local namespace/kind/name identity.
func (r Resource) GraphKey() string {
	packageName := r.Package
	if r.Kind == "Package" && packageName == "" {
		packageName = r.Metadata.Name
	}
	return Ref{Kind: r.Kind, Namespace: r.Metadata.Namespace, Name: r.Metadata.Name, Package: packageName}.GraphKey("", "", "")
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
	return namespace + "/" + kind + "/" + r.Name
}

type Diagnostic struct {
	Code    string `yaml:"code"`
	Path    string `yaml:"path,omitempty"`
	Package string `yaml:"package,omitempty"`
	Line    int    `yaml:"line,omitempty"`
	Message string `yaml:"message"`
}

type Graph struct {
	Resources     map[string]*Resource
	Packages      map[string]*Resource
	Edges         map[string][]string
	Relationships []Relationship
	ResourceAreas map[string]Area
	Project       *Resource
	Diagnostics   []Diagnostic
}

// Relationship records why a resolved graph edge exists. Edges remains the
// compact adjacency view used by existing context and impact logic.
type Relationship struct {
	From      string `yaml:"from"`
	To        string `yaml:"to"`
	Relation  string `yaml:"relation"`
	Path      string `yaml:"path,omitempty"`
	Line      int    `yaml:"resourceLine,omitempty"` // Start of the declaring resource, not the nested reference.
	Reference Ref    `yaml:"reference,omitempty"`
	Area      string `yaml:"area,omitempty"`
	Selected  bool   `yaml:"selected,omitempty"`
}
