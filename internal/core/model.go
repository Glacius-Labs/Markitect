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
	Profile     string    `yaml:"profile,omitempty"`
	Targets     []string  `yaml:"targets,omitempty"`
	Areas       []Area    `yaml:"areas,omitempty"`
	Bindings    []Binding `yaml:"bindings,omitempty"`
	// Rules map provider rule entrypoint names to owning sources. This also
	// supports several canonical sources behind one legacy entrypoint.
	RuleAdapters map[string][]Ref `yaml:"ruleAdapters,omitempty"`
}

type Resource struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
	Path       string   `yaml:"-"`
	Line       int      `yaml:"-"`
}

func (r Resource) Key() string { return r.Metadata.Namespace + "/" + r.Kind + "/" + r.Metadata.Name }
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
	Line    int    `yaml:"line,omitempty"`
	Message string `yaml:"message"`
}

type Graph struct {
	Resources   map[string]*Resource
	Edges       map[string][]string
	Project     *Resource
	Diagnostics []Diagnostic
}
