// Package authoring owns the source-facing Markitect YAML model. Resource is a
// transient typed view; Core is the sole normalized semantic value.
package authoring

import "github.com/Glacius-Labs/Markitect/internal/core"

type Ref = core.Ref
type Relationship = core.Relationship
type PolicyResult = core.PolicyResult
type DomainDefinition = core.DomainDefinition
type Diagnostic = core.Diagnostic
type PolicyException = core.PolicyException
type Metadata = core.Metadata

const APIVersion = core.APIVersion

type Core = core.Resource
type Resource struct {
	Core
	Spec Spec
}

// Source-facing configuration types. These values are decoded from canonical
// YAML and are not passed into Core as a second semantic representation.
type Spec struct {
	Text             string                `yaml:"text,omitempty"`
	Files            []string              `yaml:"files,omitempty"`
	Description      string                `yaml:"description,omitempty"`
	Rules            []core.Ref            `yaml:"rules,omitempty"`
	Uses             []core.Ref            `yaml:"uses,omitempty"`
	Needs            []core.Ref            `yaml:"needs,omitempty"`
	Implements       []core.Ref            `yaml:"implements,omitempty"`
	Input            []string              `yaml:"input,omitempty"`
	Output           []string              `yaml:"output,omitempty"`
	Kind             string                `yaml:"kind,omitempty"`
	Check            string                `yaml:"check,omitempty"`
	Providers        Providers             `yaml:"providers,omitempty"`
	Assertions       []Assertion           `yaml:"assertions,omitempty"`
	Targets          []string              `yaml:"targets,omitempty"`
	Areas            []Area                `yaml:"areas,omitempty"`
	Bindings         []Binding             `yaml:"bindings,omitempty"`
	Checks           []Check               `yaml:"checks,omitempty"`
	Documentation    *Documentation        `yaml:"documentation,omitempty"`
	Domains          []string              `yaml:"domains,omitempty"`
	Adapters         []AdapterConfig       `yaml:"adapters,omitempty"`
	PolicyDate       string                `yaml:"policyDate,omitempty"`
	PolicyExceptions []PolicyException     `yaml:"policyExceptions,omitempty"`
	RuleAdapters     map[string][]core.Ref `yaml:"ruleAdapters,omitempty"`
	ProviderAdapters *ProviderAdapters     `yaml:"providerAdapters,omitempty"`
	Consistency      *Consistency          `yaml:"consistency,omitempty"`
	Packages         []PackagePin          `yaml:"packages,omitempty"`
	Version          string                `yaml:"version,omitempty"`
	Exports          []core.Ref            `yaml:"exports,omitempty"`
}

type PackagePin struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Source  string `yaml:"source"`
	Archive string `yaml:"archive"`
	SHA256  string `yaml:"sha256"`
}
type Area struct {
	Name    string     `yaml:"name"`
	Path    string     `yaml:"path"`
	Imports []string   `yaml:"imports,omitempty"`
	Rules   []core.Ref `yaml:"rules,omitempty"`
}
type Binding struct {
	Contract       core.Ref `yaml:"contract"`
	Implementation core.Ref `yaml:"implementation"`
}
type Check struct {
	Name string   `yaml:"name"`
	Run  []string `yaml:"run"`
}
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
type ProviderAdapters struct {
	RuleSources     map[string][]string `yaml:"ruleSources,omitempty"`
	AgentContract   string              `yaml:"agentContract,omitempty"`
	RoleRegister    string              `yaml:"roleRegister,omitempty"`
	InlineAgentText bool                `yaml:"inlineAgentText,omitempty"`
	StrictInventory bool                `yaml:"strictInventory,omitempty"`
	RetiredSkills   []string            `yaml:"retiredSkills,omitempty"`
	RetiredAgents   []string            `yaml:"retiredAgents,omitempty"`
}
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
type AdapterConfig struct {
	Name    string         `yaml:"name"`
	Type    string         `yaml:"type"`
	Version string         `yaml:"version"`
	Config  map[string]any `yaml:"config,omitempty"`
}
