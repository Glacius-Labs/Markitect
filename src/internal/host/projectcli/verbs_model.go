package projectcli

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectonboarding"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// Arguments shared by several verbs.
var (
	argRepo     = arg{name: "repo", value: "PATH", cliOnly: true, help: "Project root; defaults to the current directory. MCP fixes it with `mcp --repo`."}
	argRevision = arg{name: "revision", value: "R", help: "Fixed commit to read; without it the working tree is read and reported as provisional."}
	argWrite    = arg{name: "write", kind: kindBool, help: "Persist exactly the reviewed preview."}
	argExpect   = arg{name: "expect", value: "DIGEST", help: "Digest the write is bound to; the write fails as stale if it changed."}
	argExecute  = arg{name: "execute", kind: kindBool, help: "Start agents or configured checks."}
)

// --- Set up ---------------------------------------------------------------

type initInput struct {
	Name   string `json:"name"`
	Expect string `json:"expect,omitempty"`
	Write  bool   `json:"write,omitempty"`
}

var initVerb = define(verb{
	name: "init", group: "Set up", effect: effectWrite,
	summary:  "Preview or create the project control plane on a feature branch.",
	synopsis: "init --name NAME [--expect DIGEST --write]",
	args:     []arg{argRepo, {name: "name", value: "NAME", required: true, help: "Project name."}, argExpect, argWrite},
}, func(ctx context.Context, e env, in initInput) (projectwork.InitPlan, error) {
	return e.ops.Init(projectapp.InitOperation{Root: e.root, Name: in.Name, Write: in.Write, ExpectedDigest: in.Expect})
})

type configInput struct {
	Input                  *projectsetup.Options                `json:"input,omitempty"`
	Provider               string                               `json:"provider,omitempty"`
	Model                  string                               `json:"model,omitempty"`
	Effort                 string                               `json:"effort,omitempty"`
	CodexProfile           string                               `json:"codexProfile,omitempty"`
	WindowsSandboxBackend  codexappserver.WindowsSandboxBackend `json:"windowsSandboxBackend,omitempty"`
	ProviderExecutable     string                               `json:"providerExecutable,omitempty"`
	ProviderArg            []string                             `json:"providerArg,omitempty"`
	ProviderVersion        string                               `json:"providerVersion,omitempty"`
	CostMode               string                               `json:"costMode,omitempty"`
	InputMicrosPerMillion  *int64                               `json:"inputMicrosPerMillion,omitempty"`
	OutputMicrosPerMillion *int64                               `json:"outputMicrosPerMillion,omitempty"`
	MaxCostMicros          *int64                               `json:"maxCostMicros,omitempty"`
	Expect                 string                               `json:"expect,omitempty"`
	Write                  bool                                 `json:"write,omitempty"`
}

var configVerb = define(verb{
	name: "config", group: "Set up", effect: effectWrite,
	summary:  "Preview or write the runtime configuration that runs the project's agents.",
	synopsis: "config (--input FILE | --provider codex|process --model MODEL [--effort EFFORT] [--codex-profile PROFILE] [--windows-sandbox-backend mxc] [--provider-executable PATH] [--provider-arg ARG ...] [--provider-version TEXT] [--cost-mode metered|unmetered] [--input-micros-per-million N --output-micros-per-million N] --max-cost-micros N) [--expect DIGEST --write]",
	notes:    "The standard profile is --provider codex --model gpt-6-luna; Codex effort defaults to high. --input holds complete options, including per-role profiles. Metered agents need both rates; unmetered is for process executors. Windows defaults to process-local MXC. The preview runs only the provider's --version.",
	args: []arg{argRepo,
		{name: "input", kind: kindRecord, value: "FILE", help: "Complete options record, including per-role profiles; excludes the profile flags."},
		{name: "provider", value: "P", help: "codex or process."},
		{name: "model", value: "MODEL", help: "Model name."},
		{name: "effort", value: "EFFORT", help: "Reasoning effort."},
		{name: "codex-profile", value: "PROFILE", help: "Codex configuration profile."},
		{name: "windows-sandbox-backend", value: "mxc", help: "Windows sandbox backend."},
		{name: "provider-executable", value: "PATH", readOnlyOmit: true, help: "Absolute provider executable; required for a process executor."},
		{name: "provider-arg", kind: kindList, value: "ARG", help: "Process executor argument (repeatable)."},
		{name: "provider-version", value: "TEXT", help: "Declared process executor version."},
		{name: "cost-mode", value: "MODE", help: "metered (default) or unmetered."},
		{name: "input-micros-per-million", kind: kindInt, value: "N", help: "Input token price in micros per million tokens."},
		{name: "output-micros-per-million", kind: kindInt, value: "N", help: "Output token price in micros per million tokens."},
		{name: "max-cost-micros", kind: kindInt, value: "N", help: "Cost cap per run in micros."},
		argExpect, argWrite},
}, func(ctx context.Context, e env, in configInput) (projectsetup.Preview, error) {
	options, err := in.options(e.readOnly)
	if err != nil {
		return projectsetup.Preview{}, err
	}
	return e.ops.Setup(projectapp.SetupOperation{Root: e.root, Options: options, Write: in.Write, ExpectedDigest: in.Expect})
})

const maxMicros = 1_000_000_000_000

// options takes either one complete options record or the default-profile
// flags, never both.
func (in configInput) options(readOnly bool) (projectsetup.Options, error) {
	if in.Input != nil {
		if in.Provider != "" || in.Model != "" || in.Effort != "" || in.CodexProfile != "" || in.WindowsSandboxBackend != "" || in.ProviderExecutable != "" ||
			len(in.ProviderArg) != 0 || in.ProviderVersion != "" || in.CostMode != "" || in.InputMicrosPerMillion != nil || in.OutputMicrosPerMillion != nil || in.MaxCostMicros != nil {
			return projectsetup.Options{}, usagef("--input carries the complete options; do not combine it with profile flags")
		}
		if readOnly && namesExecutable(*in.Input) {
			return projectsetup.Options{}, usagef("this read-only MCP server does not accept provider executables")
		}
		return *in.Input, nil
	}
	if in.Provider == "" || in.Model == "" {
		return projectsetup.Options{}, usagef("requires --provider and --model, or --input with complete options")
	}
	if in.MaxCostMicros == nil {
		return projectsetup.Options{}, usagef("requires --max-cost-micros")
	}
	if *in.MaxCostMicros <= 0 || *in.MaxCostMicros > maxMicros {
		return projectsetup.Options{}, usagef("--max-cost-micros must be between 1 and %d", int64(maxMicros))
	}
	options := projectsetup.Options{
		Provider: in.Provider, Model: in.Model, Effort: in.Effort, CodexProfile: in.CodexProfile,
		WindowsSandboxBackend: in.WindowsSandboxBackend, ProviderExecutable: in.ProviderExecutable,
		ProviderArgs: append([]string(nil), in.ProviderArg...), ProviderVersion: in.ProviderVersion, CostMode: in.CostMode,
		MaxCostMicros: *in.MaxCostMicros,
	}
	if in.CostMode == projectrun.CostModeUnmetered {
		if in.InputMicrosPerMillion != nil || in.OutputMicrosPerMillion != nil {
			return projectsetup.Options{}, usagef("--cost-mode unmetered declares no price; omit the input and output rates")
		}
		return options, nil
	}
	if in.InputMicrosPerMillion == nil || in.OutputMicrosPerMillion == nil {
		return projectsetup.Options{}, usagef("metered agents require --input-micros-per-million and --output-micros-per-million")
	}
	input, output := *in.InputMicrosPerMillion, *in.OutputMicrosPerMillion
	if input < 0 || output < 0 {
		return projectsetup.Options{}, usagef("price rates must be nonnegative integers")
	}
	if input == 0 && output == 0 {
		return projectsetup.Options{}, usagef("at least one input or output price rate must be positive")
	}
	if input > maxMicros || output > maxMicros {
		return projectsetup.Options{}, usagef("price rates must not exceed %d micros per million tokens", int64(maxMicros))
	}
	options.InputMicrosPerMillion, options.OutputMicrosPerMillion = input, output
	return options, nil
}

func namesExecutable(options projectsetup.Options) bool {
	if options.ProviderExecutable != "" {
		return true
	}
	if options.Roles == nil {
		return false
	}
	for _, profile := range []*projectsetup.RoleProfile{options.Roles.Manager, options.Roles.Reviewer, options.Roles.Verifier} {
		if profile != nil && profile.ProviderExecutable != "" {
			return true
		}
	}
	return false
}

type onboardInput struct {
	Provider     string `json:"provider"`
	DocumentPath string `json:"documentPath,omitempty"`
	Expect       string `json:"expect,omitempty"`
	Write        bool   `json:"write,omitempty"`
}

var onboardVerb = define(verb{
	name: "onboard", group: "Set up", effect: effectWrite,
	summary:  "Preview or install project-local Codex or Claude guidance without replacing custom content.",
	synopsis: "onboard --provider codex|claude|both [--document-path PATH] [--expect DIGEST --write]",
	args: []arg{argRepo,
		{name: "provider", value: "P", required: true, help: "codex, claude or both."},
		{name: "document-path", value: "PATH", help: "Readable model document destination."},
		argExpect, argWrite},
}, func(ctx context.Context, e env, in onboardInput) (projectonboarding.Plan, error) {
	var providers []projectonboarding.Provider
	switch in.Provider {
	case "codex":
		providers = []projectonboarding.Provider{projectonboarding.Codex}
	case "claude":
		providers = []projectonboarding.Provider{projectonboarding.Claude}
	case "both":
		providers = []projectonboarding.Provider{projectonboarding.Codex, projectonboarding.Claude}
	default:
		return projectonboarding.Plan{}, usagef("--provider must be codex, claude or both")
	}
	return e.ops.Onboard(projectapp.OnboardOperation{Root: e.root, Options: projectonboarding.Options{Providers: providers, DocumentationPath: in.DocumentPath}, Write: in.Write, ExpectedDigest: in.Expect})
})

type doctorInput struct {
	Provider           string `json:"provider"`
	ProviderExecutable string `json:"providerExecutable,omitempty"`
}

var doctorVerb = define(verb{
	name: "doctor", group: "Set up", effect: effectRead,
	summary:  "Inspect local tools and authentication prerequisites without starting agents.",
	synopsis: "doctor --provider codex|process [--provider-executable PATH]",
	notes:    "It runs only the provider's --version.",
	args: []arg{argRepo,
		{name: "provider", value: "P", required: true, help: "codex or process."},
		{name: "provider-executable", value: "PATH", readOnlyOmit: true, help: "Provider executable to inspect."}},
}, func(ctx context.Context, e env, in doctorInput) (projectsetup.DoctorReport, error) {
	report, err := e.ops.Doctor(projectapp.DoctorOperation{Root: e.root, Options: projectsetup.Options{Provider: in.Provider, ProviderExecutable: in.ProviderExecutable}})
	if err != nil {
		return report, err
	}
	for _, check := range report.Checks {
		if check.Status == "blocked" || check.Status == "missing" {
			return report, outcomef("one or more local prerequisites need attention")
		}
	}
	return report, nil
})

var mcpVerb = verb{
	name: "mcp", group: "Set up", effect: effectServer, cliOnly: true,
	summary:  "Serve the verbs as MCP tools over stdio.",
	synopsis: "mcp [--repo PATH] [--read-only]",
	notes:    "The root is fixed here and never taken from tool arguments. --read-only serves reads and previews only.",
	args: []arg{argRepo,
		{name: "read-only", kind: kindBool, cliOnly: true, help: "Serve only reads and previews; nothing is written or started."}},
}

// --- Model ----------------------------------------------------------------

type noInput struct{}

var schemaVerb = define(verb{
	name: "schema", group: "Model", effect: effectRead,
	summary:  "Print the project model schema.",
	synopsis: "schema",
	args:     []arg{argRepo},
}, func(ctx context.Context, e env, in noInput) (json.RawMessage, error) {
	// core.Schema is recursive (a Property has Properties), so it is returned
	// as raw JSON: the derived MCP output schema stays finite.
	return json.Marshal(projectmodel.Schema())
})

type checkInput struct {
	Revision string `json:"revision,omitempty"`
	Coverage bool   `json:"coverage,omitempty"`
}

type checkReport struct {
	ProjectDigest string                  `json:"projectDigest"`
	Revision      string                  `json:"revision"`
	Provisional   bool                    `json:"provisional"`
	Status        string                  `json:"status"`
	Findings      []projectmodel.Finding  `json:"findings"`
	Unknown       []string                `json:"unknown"`
	Coverage      *projectcoverage.Report `json:"coverage,omitempty"`
}

var checkVerb = define(verb{
	name: "check", group: "Model", effect: effectRead,
	summary:  "Compile the selected project and report findings and coverage.",
	synopsis: "check [--revision R] [--coverage]",
	args: []arg{argRepo, argRevision,
		{name: "coverage", kind: kindBool, help: "Evaluate whole-repository coverage regardless of the project's coverage mode."}},
}, func(ctx context.Context, e env, in checkInput) (checkReport, error) {
	selection := projectapp.Selection{Root: e.root, Revision: in.Revision}
	result, err := e.ops.Check(selection)
	if err != nil {
		return checkReport{}, err
	}
	if in.Coverage && result.Coverage == nil {
		coverage, err := e.ops.Coverage(selection)
		if err != nil {
			return checkReport{}, err
		}
		result.Coverage = &coverage
	}
	report := checkReport{result.Source.ProjectDigest, result.Source.Revision, result.Source.Provisional, result.Report.Status, result.Findings, result.Unknown, result.Coverage}
	if result.Report.Status != "succeeded" {
		return report, outcomef("project report is %s", result.Report.Status)
	}
	if (in.Coverage || result.Source.CoverageMode == "full") && (result.Coverage == nil || !result.Coverage.Conforming) {
		return report, outcomef("whole-repository coverage is not conforming")
	}
	return report, nil
})

type revisionInput struct {
	Revision string `json:"revision,omitempty"`
}

var modelVerb = define(verb{
	name: "model", group: "Model", effect: effectRead,
	summary:  "Read the compiled project model and ownership index.",
	synopsis: "model [--revision R]",
	args:     []arg{argRepo, argRevision},
}, func(ctx context.Context, e env, in revisionInput) (projectmodel.Report, error) {
	return e.ops.Index(projectapp.Selection{Root: e.root, Revision: in.Revision})
})

type contextInput struct {
	Manager   string `json:"manager"`
	Revision  string `json:"revision,omitempty"`
	Trace     string `json:"trace,omitempty"`
	Direction string `json:"direction,omitempty"`
	Depth     int    `json:"depth,omitempty"`
}

// contextResult is a Manager's Context and, with --trace, a walk over its
// knowledge graph, which is built from that Context alone (KG-02).
type contextResult struct {
	projectmodel.ManagerContext
	Trace *projectmodel.TraceResult `json:"trace,omitempty"`
}

var contextVerb = define(verb{
	name: "context", group: "Model", effect: effectRead,
	summary:  "Read one Manager's context under the accepted-history rules; --trace walks its knowledge graph.",
	synopsis: "context MANAGER [--revision R] [--trace ID [--direction out|in|both] [--depth N]]",
	args: []arg{{name: "manager", value: "MANAGER", operand: true, required: true, help: "Manager ID."}, argRepo, argRevision,
		{name: "trace", value: "ID", help: "Walk the Manager's knowledge graph from this node and return each reached node with a shortest witness path."},
		{name: "direction", value: "out|in|both", help: "With --trace: follow relations outward (default), inward or both ways."},
		{name: "depth", kind: kindInt, value: "N", help: "With --trace: the most relations to follow; default 6, at most 32."}},
}, func(ctx context.Context, e env, in contextInput) (contextResult, error) {
	operation := projectapp.ContextOperation{Selection: projectapp.Selection{Root: e.root, Revision: in.Revision}, ManagerID: in.Manager}
	if in.Trace == "" {
		if in.Direction != "" || in.Depth != 0 {
			return contextResult{}, usagef("--direction and --depth require --trace")
		}
		managerContext, err := e.ops.Context(operation)
		return contextResult{ManagerContext: managerContext}, err
	}
	if !slices.Contains([]string{"", "out", "in", "both"}, in.Direction) {
		return contextResult{}, usagef("--direction must be out, in or both")
	}
	if in.Depth < 0 {
		return contextResult{}, usagef("--depth must not be negative")
	}
	traced, err := e.ops.Trace(projectapp.TraceOperation{ContextOperation: operation, Request: projectmodel.TraceRequest{From: in.Trace, Direction: in.Direction, MaxDepth: in.Depth}})
	if err != nil {
		return contextResult{}, err
	}
	return contextResult{ManagerContext: traced.Context, Trace: &traced.Trace}, nil
})

type impactInput struct {
	Since    string `json:"since"`
	Revision string `json:"revision"`
	Explain  bool   `json:"explain,omitempty"`
	Manager  string `json:"manager,omitempty"`
}

// impactResult is the change impact and, with --explain, why each element is
// in it and whether it must change or is context (DEC-023).
type impactResult struct {
	projectmodel.ChangeImpact
	Explanation *projectmodel.ImpactExplanation `json:"explanation,omitempty"`
}

var impactVerb = define(verb{
	name: "impact", group: "Model", effect: effectRead,
	summary:  "Compare two revisions with deterministic, conservative change impact; --explain says why each element is in it.",
	synopsis: "impact --since R1 --revision R2 [--explain [--manager ID]]",
	args: []arg{argRepo,
		{name: "since", value: "R1", required: true, help: "Older revision."},
		{name: "revision", value: "R2", required: true, help: "Newer revision."},
		{name: "explain", kind: kindBool, help: "Add each element's reason, shortest witness path and class: change or context."},
		{name: "manager", value: "ID", help: "With --explain: explain only what this Manager may see. The impact itself stays project scope."}},
}, func(ctx context.Context, e env, in impactInput) (impactResult, error) {
	operation := projectapp.ImpactOperation{Root: e.root, BaseRevision: in.Since, Revision: in.Revision}
	if !in.Explain {
		if in.Manager != "" {
			return impactResult{}, usagef("--manager requires --explain")
		}
		impact, err := e.ops.Impact(operation)
		return impactResult{ChangeImpact: impact}, err
	}
	explained, err := e.ops.Explain(projectapp.ExplainOperation{ImpactOperation: operation, ManagerID: in.Manager})
	if err != nil {
		return impactResult{}, err
	}
	return impactResult{ChangeImpact: explained.Impact, Explanation: &explained.Explanation}, nil
})

type docsInput struct {
	Revision string `json:"revision,omitempty"`
	Expect   string `json:"expect,omitempty"`
	Write    bool   `json:"write,omitempty"`
}

var docsVerb = define(verb{
	name: "docs", group: "Model", effect: effectWrite,
	summary:  "Preview or write the readable model document.",
	synopsis: "docs [--revision R] | docs --expect DIGEST --write",
	notes:    "A write uses the working tree, so it does not take --revision.",
	args:     []arg{argRepo, argRevision, argExpect, argWrite},
}, func(ctx context.Context, e env, in docsInput) (projectapp.DocumentResult, error) {
	if in.Write && in.Revision != "" {
		return projectapp.DocumentResult{}, usagef("--write uses the working tree; omit --revision")
	}
	return e.ops.Document(projectapp.DocumentOperation{Selection: projectapp.Selection{Root: e.root, Revision: in.Revision}, Write: in.Write, ExpectedDigest: in.Expect})
})

type editInput struct {
	Input    projectwork.Mutation `json:"input"`
	Revision string               `json:"revision,omitempty"`
	Expect   string               `json:"expect,omitempty"`
	Write    bool                 `json:"write,omitempty"`
}

var editVerb = define(verb{
	name: "edit", group: "Model", effect: effectWrite,
	summary:  "Preview or apply one canonical model mutation.",
	synopsis: "edit --input FILE [--revision R] [--expect DIGEST --write]",
	args: []arg{argRepo,
		{name: "input", kind: kindRecord, value: "FILE", required: true, help: "Model mutation record."},
		argRevision, argExpect, argWrite},
}, func(ctx context.Context, e env, in editInput) (projectwork.EditPlan, error) {
	return e.ops.Edit(projectapp.EditOperation{Selection: projectapp.Selection{Root: e.root, Revision: in.Revision}, Mutation: in.Input, Write: in.Write, ExpectedDigest: in.Expect})
})

// staleOnly drops the report of a stale write: the operation did not run.
func staleOnly[T any](report T, err error) (T, error) {
	if errors.Is(err, projectrun.ErrStale) {
		var zero T
		return zero, err
	}
	return report, err
}
