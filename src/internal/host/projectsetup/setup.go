// Package projectsetup builds a deterministic, project-local agent runtime
// proposal. It discovers and fingerprints tools but never starts an agent.
package projectsetup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"go.yaml.in/yaml/v3"
)

const (
	DefaultTimeout                      = time.Hour
	DefaultMaxStdout                    = 4 << 20
	DefaultMaxStderr                    = 1 << 20
	DefaultMaxDepth                     = 8
	DefaultMaxStarts                    = 256
	DefaultMaxParallel                  = 2
	DefaultMaxHelperStarts              = 8
	DefaultMaxEventBytes          int64 = 16 << 20
	DefaultMaxRunTime                   = 4 * time.Hour
	DefaultMaxFileBytes                 = 1 << 20
	DefaultMaxTotalBytes                = 8 << 20
	DefaultReviewMaxRounds              = 3
	DefaultReviewMaxManagerRounds       = 2
	// DefaultProcessMaxStdout leaves room for a full 8 MiB candidate in a
	// process executor's JSON response.
	DefaultProcessMaxStdout = 16 << 20
	DefaultCodexEffort      = "high"
	// UndeclaredProviderVersion is recorded when a process executor declares no
	// version. The executable bytes are still pinned by digest.
	UndeclaredProviderVersion = "undeclared"

	ProviderCodex   = "codex"
	ProviderProcess = "process"
	RoleManager     = "manager"
	RoleReviewer    = "reviewer"
	RoleVerifier    = "verifier"
)

// Options selects the default executor profile for every role. Roles
// optionally overrides it for Managers, reviewers or the verifier; mixed
// providers and models are allowed.
type Options struct {
	Provider              string                               `json:"provider"`
	Model                 string                               `json:"model"`
	Effort                string                               `json:"effort"`
	CodexProfile          string                               `json:"codexProfile"`
	WindowsSandboxBackend codexappserver.WindowsSandboxBackend `json:"windowsSandboxBackend,omitempty"`
	ProviderExecutable    string                               `json:"providerExecutable"`
	// ProviderArgs, ProviderVersion and Environment apply only to process
	// executors. Environment lists extra caller variable names to pass.
	ProviderArgs           []string     `json:"providerArgs,omitempty"`
	ProviderVersion        string       `json:"providerVersion,omitempty"`
	Environment            []string     `json:"environment,omitempty"`
	CostMode               string       `json:"costMode,omitempty"`
	InputMicrosPerMillion  int64        `json:"inputMicrosPerMillion"`
	OutputMicrosPerMillion int64        `json:"outputMicrosPerMillion"`
	MaxCostMicros          int64        `json:"maxCostMicros"`
	Roles                  *RoleOptions `json:"roles,omitempty"`
}

// RoleOptions overrides the default profile per role class. Every Manager
// shares the manager profile and every Manager's reviewer the reviewer profile.
type RoleOptions struct {
	Manager  *RoleProfile `json:"manager,omitempty"`
	Reviewer *RoleProfile `json:"reviewer,omitempty"`
	Verifier *RoleProfile `json:"verifier,omitempty"`
}

// RoleProfile changes selected fields of the default profile for one role.
// Empty fields inherit it. Changing the provider inherits only the rates, so
// the role must name its own model and, for a process, its executable.
type RoleProfile struct {
	Provider               string   `json:"provider,omitempty"`
	Model                  string   `json:"model,omitempty"`
	Effort                 string   `json:"effort,omitempty"`
	ProviderExecutable     string   `json:"providerExecutable,omitempty"`
	ProviderArgs           []string `json:"providerArgs,omitempty"`
	ProviderVersion        string   `json:"providerVersion,omitempty"`
	Environment            []string `json:"environment,omitempty"`
	CostMode               string   `json:"costMode,omitempty"`
	InputMicrosPerMillion  *int64   `json:"inputMicrosPerMillion,omitempty"`
	OutputMicrosPerMillion *int64   `json:"outputMicrosPerMillion,omitempty"`
}

// roleProfile is one resolved, validated role selection.
type roleProfile struct {
	provider, model, effort       string
	executable, version, costMode string
	args, environment             []string
	pricing                       projectrun.Pricing
}

type Tool struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
	Mode    string `json:"mode"`
}

type Discovery struct {
	Provider           string `json:"provider"`
	ProviderBinary     Tool   `json:"providerBinary"`
	Authentication     string `json:"authentication"`
	AuthenticationNote string `json:"authenticationNote"`
}

type Preview struct {
	APIVersion string    `json:"apiVersion"`
	Discovery  Discovery `json:"discovery"`
	// Roles shows the executable each role class pins when role profiles
	// override the default; Discovery then describes the Manager profile.
	Roles        map[string]Discovery `json:"roles,omitempty"`
	PricingBasis string               `json:"pricingBasis"`
	Mutation     projectwork.Mutation `json:"mutation"`
	EditPlan     projectwork.EditPlan `json:"editPlan"`
}

type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type DoctorReport struct {
	APIVersion     string  `json:"apiVersion"`
	Provider       string  `json:"provider"`
	Authentication string  `json:"authentication"`
	Checks         []Check `json:"checks"`
}

var versionPattern = regexp.MustCompile(`(?i)(?:^|[^0-9])v?([0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?)`)

func PreviewEdit(project *projectwork.Project, options Options) (Preview, error) {
	return previewEditWithDiscovery(project, options, Discover)
}

func previewEditWithDiscovery(project *projectwork.Project, options Options, discover func(Options) (Discovery, error)) (Preview, error) {
	var result Preview
	if project == nil {
		return result, errors.New("active project is required")
	}
	options, err := normalizeOptions(options)
	if err != nil {
		return result, err
	}
	profiles, err := resolveRoleProfiles(options)
	if err != nil {
		return result, err
	}
	// Discover only executables that a role uses, each once, Managers first.
	discoverOnce := cachedDiscovery(nil, discover)
	discovery, err := discoverOnce(roleSelection(options, profiles[RoleManager]))
	if err != nil {
		return result, err
	}
	runtimeConfig, err := buildRuntime(project, options, discoverOnce)
	if err != nil {
		return result, err
	}
	var roles map[string]Discovery
	if options.Roles != nil {
		roles = make(map[string]Discovery, len(profiles))
		for role, profile := range profiles {
			if roles[role], err = discoverOnce(roleSelection(options, profile)); err != nil {
				return result, err
			}
		}
	}
	content, err := yaml.Marshal(runtimeConfig)
	if err != nil {
		return result, fmt.Errorf("encode runtime proposal: %w", err)
	}
	mutation := projectwork.Mutation{
		APIVersion: projectwork.APIVersion,
		BaseDigest: project.Digest,
		Actor:      projectwork.HumanActor,
		Goal:       "Configure the project-local agent runtime from explicitly selected local tools",
		Files:      []projectwork.FileChange{{Path: projectwork.RuntimePath, Content: string(content)}},
	}
	plan, err := projectwork.PlanEdit(project, mutation)
	if err != nil {
		return result, fmt.Errorf("validate runtime edit proposal: %w", err)
	}
	return Preview{
		APIVersion:   "markitect.example.org/project-setup/v1alpha1",
		Discovery:    discovery,
		Roles:        roles,
		PricingBasis: "Caller-supplied budget estimate; not a provider price quote, invoice, or hard billing cap.",
		Mutation:     mutation, EditPlan: plan,
	}, nil
}

// BuildRuntime builds the runtime proposal. found is the discovery of the
// default profile's executable; a role that selects another executable is
// discovered separately.
func BuildRuntime(project *projectwork.Project, options Options, found Discovery) (projectrun.Runtime, error) {
	return buildRuntime(project, options, cachedDiscovery(map[string]Discovery{discoveryKey(options): found}, Discover))
}

// cachedDiscovery discovers each selected executable once, starting from the
// given known discoveries.
func cachedDiscovery(known map[string]Discovery, discover func(Options) (Discovery, error)) func(Options) (Discovery, error) {
	cache := map[string]Discovery{}
	for key, value := range known {
		cache[key] = value
	}
	return func(selected Options) (Discovery, error) {
		key := discoveryKey(selected)
		if cached, ok := cache[key]; ok {
			return cached, nil
		}
		discovered, err := discover(selected)
		if err != nil {
			return Discovery{}, err
		}
		cache[key] = discovered
		return discovered, nil
	}
}

func discoveryKey(options Options) string {
	version := options.ProviderVersion
	switch options.Provider {
	case ProviderCodex:
		version = ""
	case ProviderProcess:
		if version == "" {
			version = UndeclaredProviderVersion
		}
	}
	return options.Provider + "\x00" + options.ProviderExecutable + "\x00" + version
}

func buildRuntime(project *projectwork.Project, options Options, discover func(Options) (Discovery, error)) (projectrun.Runtime, error) {
	var config projectrun.Runtime
	if project == nil || len(project.Report.Managers) == 0 {
		return config, errors.New("active project must contain at least one Manager")
	}
	options, err := normalizeOptions(options)
	if err != nil {
		return config, err
	}
	if options.MaxCostMicros <= 0 {
		return config, errors.New("a positive max-cost-micros budget is required; it bounds the known estimated cost of metered agents")
	}
	profiles, err := resolveRoleProfiles(options)
	if err != nil {
		return config, err
	}
	builder := &agentBuilder{project: project, options: options, discover: discover}
	roleAgents := make(map[string]projectrun.Agent, len(profiles))
	for _, role := range []string{RoleManager, RoleReviewer, RoleVerifier} {
		agent, err := builder.agent(role, profiles[role])
		if err != nil {
			return config, err
		}
		roleAgents[role] = agent
	}
	agents := make(map[string]projectrun.Agent, len(project.Report.Managers))
	reviewAgents := make(map[string]projectrun.Agent, len(project.Report.Managers))
	for _, manager := range project.Report.Managers {
		if manager.ID == "" {
			return config, errors.New("active project has a Manager with an empty ID")
		}
		agents[manager.ID] = cloneAgent(roleAgents[RoleManager])
		reviewAgents[manager.ID] = cloneAgent(roleAgents[RoleReviewer])
	}
	verifier := cloneAgent(roleAgents[RoleVerifier])
	config = projectrun.Runtime{
		APIVersion: projectrun.APIVersion, Mode: projectrun.ModeControlledLocal, RequireIsolation: false,
		Agents: agents, Verifier: &verifier,
		Review: &projectrun.ReviewConfig{
			Agents: reviewAgents, MaxRounds: DefaultReviewMaxRounds, MaxManagerRounds: DefaultReviewMaxManagerRounds,
		},
		Limits: projectrun.Limits{
			MaxDepth: DefaultMaxDepth, MaxStarts: DefaultMaxStarts, MaxRetries: 1, MaxParallel: DefaultMaxParallel,
			MaxDuration: projectrun.Duration(DefaultMaxRunTime), MaxCostMicros: options.MaxCostMicros,
			MaxCandidateFileBytes: DefaultMaxFileBytes, MaxCandidateBytes: DefaultMaxTotalBytes,
		},
	}
	if err := projectrun.ValidateRuntime(config); err != nil {
		return projectrun.Runtime{}, fmt.Errorf("validate generated runtime: %w", err)
	}
	for _, roleAgents := range []map[string]projectrun.Agent{agents, reviewAgents} {
		for _, agent := range roleAgents {
			agentConfig, err := agent.AgentConfig()
			if err != nil {
				return projectrun.Runtime{}, err
			}
			if _, err := agentexec.Fingerprint(agentConfig); err != nil {
				return projectrun.Runtime{}, fmt.Errorf("validate selected runtime fingerprints: %w", err)
			}
		}
	}
	verifierConfig, err := verifier.AgentConfig()
	if err != nil {
		return config, err
	}
	if _, err := agentexec.Fingerprint(verifierConfig); err != nil {
		return config, err
	}
	return config, nil
}

func normalizeOptions(options Options) (Options, error) {
	if options.Provider != ProviderCodex && options.Provider != ProviderProcess {
		return options, unsupportedProvider(options.Provider)
	}
	if options.WindowsSandboxBackend != "" && options.WindowsSandboxBackend != codexappserver.WindowsSandboxBackendMXC {
		return options, errors.New("unsupported --windows-sandbox-backend; supported value is mxc")
	}
	if options.WindowsSandboxBackend != "" && runtime.GOOS != "windows" {
		return options, errors.New("--windows-sandbox-backend mxc is supported only on Windows")
	}
	return options, nil
}

func unsupportedProvider(provider string) error {
	return fmt.Errorf("unsupported provider %q; project setup supports %q (native Codex App Server) and %q (bring-your-own executor over agent-execution/v1alpha1)", provider, ProviderCodex, ProviderProcess)
}

// resolveRoleProfiles applies each role's overrides to the default profile and
// validates the result.
func resolveRoleProfiles(options Options) (map[string]roleProfile, error) {
	base := roleProfile{provider: options.Provider, model: options.Model, effort: options.Effort,
		executable: options.ProviderExecutable, version: options.ProviderVersion, costMode: options.CostMode,
		args: options.ProviderArgs, environment: options.Environment,
		pricing: projectrun.Pricing{InputMicrosPerMillion: options.InputMicrosPerMillion, OutputMicrosPerMillion: options.OutputMicrosPerMillion}}
	var overrides RoleOptions
	if options.Roles != nil {
		overrides = *options.Roles
	}
	profiles := make(map[string]roleProfile, 3)
	for _, role := range []struct {
		name     string
		override *RoleProfile
	}{{RoleManager, overrides.Manager}, {RoleReviewer, overrides.Reviewer}, {RoleVerifier, overrides.Verifier}} {
		profile := base
		if override := role.override; override != nil {
			if override.Provider != "" && override.Provider != base.provider {
				// A new provider keeps only the default rates; it is metered
				// unless the role itself declares another cost mode.
				profile = roleProfile{provider: override.Provider, pricing: base.pricing}
			}
			if override.Model != "" {
				profile.model = override.Model
			}
			if override.Effort != "" {
				profile.effort = override.Effort
			}
			if override.ProviderExecutable != "" {
				profile.executable = override.ProviderExecutable
			}
			if override.ProviderVersion != "" {
				profile.version = override.ProviderVersion
			}
			if override.ProviderArgs != nil {
				profile.args = override.ProviderArgs
			}
			if override.Environment != nil {
				profile.environment = override.Environment
			}
			if override.CostMode != "" {
				profile.costMode = override.CostMode
				if override.CostMode == projectrun.CostModeUnmetered {
					profile.pricing = projectrun.Pricing{}
				}
			}
			if override.InputMicrosPerMillion != nil {
				profile.pricing.InputMicrosPerMillion = *override.InputMicrosPerMillion
			}
			if override.OutputMicrosPerMillion != nil {
				profile.pricing.OutputMicrosPerMillion = *override.OutputMicrosPerMillion
			}
		}
		if err := validateRoleProfile(&profile); err != nil {
			return nil, fmt.Errorf("%s profile: %w", role.name, err)
		}
		profiles[role.name] = profile
	}
	return profiles, nil
}

func validateRoleProfile(profile *roleProfile) error {
	if strings.TrimSpace(profile.model) == "" || profile.model != strings.TrimSpace(profile.model) {
		return errors.New("model must be nonempty and have no surrounding whitespace")
	}
	switch profile.costMode {
	case "", projectrun.CostModeMetered:
		if profile.pricing.InputMicrosPerMillion < 0 || profile.pricing.OutputMicrosPerMillion < 0 ||
			(profile.pricing.InputMicrosPerMillion == 0 && profile.pricing.OutputMicrosPerMillion == 0) {
			return errors.New("a metered agent requires explicit nonnegative input/output rates with at least one positive rate")
		}
	case projectrun.CostModeUnmetered:
		if profile.pricing != (projectrun.Pricing{}) {
			return errors.New("an unmetered agent must not declare input/output rates")
		}
	default:
		return fmt.Errorf("unsupported cost mode %q; use %q or %q", profile.costMode, projectrun.CostModeMetered, projectrun.CostModeUnmetered)
	}
	switch profile.provider {
	case ProviderCodex:
		if profile.costMode == projectrun.CostModeUnmetered {
			return errors.New("the Codex App Server reports token usage and must be metered; unmetered applies to process executors")
		}
		if profile.effort == "" {
			profile.effort = DefaultCodexEffort
		}
		if !codexappserver.SupportedReasoningEffort(profile.effort) {
			return fmt.Errorf("unsupported Codex reasoning effort %q", profile.effort)
		}
		if len(profile.args) != 0 || profile.version != "" || len(profile.environment) != 0 {
			return errors.New("providerArgs, providerVersion and environment apply only to process executors")
		}
	case ProviderProcess:
		if profile.executable == "" || !filepath.IsAbs(profile.executable) {
			return errors.New("a process executor requires an absolute providerExecutable")
		}
		if profile.version == "" {
			profile.version = UndeclaredProviderVersion
		}
		if profile.version != strings.TrimSpace(profile.version) || profile.effort != strings.TrimSpace(profile.effort) {
			return errors.New("process providerVersion and effort must have no surrounding whitespace")
		}
	default:
		return unsupportedProvider(profile.provider)
	}
	return nil
}

// agentBuilder discovers each role's executable and builds its runtime agent.
// Native instruction pins are read once and only when a role uses Codex.
type agentBuilder struct {
	project          *projectwork.Project
	options          Options
	discover         func(Options) (Discovery, error)
	instructionsRead bool
	instructionPaths []string
	instructionFiles []agentexec.RuntimeFile
}

// roleSelection is the discovery input for one role's executable.
func roleSelection(options Options, profile roleProfile) Options {
	selected := options
	selected.Provider, selected.ProviderExecutable, selected.ProviderVersion, selected.Roles = profile.provider, profile.executable, profile.version, nil
	if profile.provider == ProviderCodex {
		// Codex identifies itself through --version; a declared label is not used.
		selected.ProviderVersion = ""
	}
	return selected
}

func (b *agentBuilder) agent(role string, profile roleProfile) (projectrun.Agent, error) {
	found, err := b.discover(roleSelection(b.options, profile))
	if err != nil {
		return projectrun.Agent{}, fmt.Errorf("%s executor: %w", role, err)
	}
	if found.Provider != profile.provider || found.ProviderBinary.Path == "" || !filepath.IsAbs(found.ProviderBinary.Path) {
		return projectrun.Agent{}, fmt.Errorf("%s executor discovery does not match the selected provider", role)
	}
	if profile.provider == ProviderProcess {
		return processAgent(profile, found), nil
	}
	if found.ProviderBinary.Version != codexappserver.SupportedProviderVersion {
		return projectrun.Agent{}, errors.New("native Codex setup requires Codex CLI 0.162.0")
	}
	if !b.instructionsRead {
		b.instructionPaths, b.instructionFiles, err = nativeInstructionFiles(b.project, ProviderCodex)
		if err != nil {
			return projectrun.Agent{}, err
		}
		b.instructionsRead = true
	}
	if len(b.instructionPaths) == 0 {
		return projectrun.Agent{}, errors.New("native Codex setup requires existing generated Codex project instructions; run project onboard first")
	}
	sandboxBackend := b.options.WindowsSandboxBackend
	if sandboxBackend == "" && runtime.GOOS == "windows" {
		sandboxBackend = codexappserver.WindowsSandboxBackendMXC
	}
	permissionProfile := b.options.CodexProfile
	if permissionProfile == "" {
		permissionProfile = ":workspace"
	}
	environment := []string{"PATH", "TEMP", "TMP"}
	if runtime.GOOS == "windows" {
		environment = append(environment, "SystemRoot", "USERPROFILE", "APPDATA", "LOCALAPPDATA")
	} else {
		environment = append(environment, "HOME")
	}
	runtimeFiles := append([]agentexec.RuntimeFile{runtimeFile(found.ProviderBinary)}, b.instructionFiles...)
	agent := selectedAgent(found, profile.model, profile.effort, permissionProfile, sandboxBackend, runtimeFiles, environment, profile.pricing)
	agent.CostMode = costModeField(profile.costMode)
	agent.WorkspaceMode = "git"
	agent.InstructionPaths = append([]string(nil), b.instructionPaths...)
	return agent, nil
}

func selectedAgent(found Discovery, model, effort, permissionProfile string, sandboxBackend codexappserver.WindowsSandboxBackend, files []agentexec.RuntimeFile, environment []string, pricing projectrun.Pricing) projectrun.Agent {
	return projectrun.Agent{
		Command: found.ProviderBinary.Path, Transport: projectrun.TransportCodexAppServer,
		AppServer: &projectrun.AppServerSettings{
			ReasoningEffort: effort,
			// BuildRuntime supplies one shared :workspace profile by default.
			PermissionProfile:     permissionProfile,
			EnvironmentMode:       projectrun.AppServerEnvironmentModeInherit,
			WindowsSandboxBackend: sandboxBackend,
			Helpers:               projectrun.AppServerHelpers{Enabled: true, MaxStartRequests: DefaultMaxHelperStarts, MaxDepth: 1},
			MaxEventBytes:         DefaultMaxEventBytes,
		},
		Model: model, ProviderVersion: found.ProviderBinary.Version,
		Timeout: projectrun.Duration(DefaultTimeout), MaxStdoutBytes: DefaultMaxStdout, MaxStderrBytes: DefaultMaxStderr,
		RuntimeFiles: append([]agentexec.RuntimeFile(nil), files...), Environment: append([]string(nil), environment...), Pricing: pricing,
	}
}

// processAgent selects a bring-your-own executor over the process transport.
// It receives each invocation on stdin and returns candidate files and reports
// on stdout; its executable bytes are pinned and its environment is explicit.
func processAgent(profile roleProfile, found Discovery) projectrun.Agent {
	var modelOptions any
	if profile.effort != "" {
		modelOptions = map[string]any{"reasoningEffort": profile.effort}
	}
	environment := []string{"PATH", "TEMP", "TMP"}
	if runtime.GOOS == "windows" {
		environment = append(environment, "SystemRoot")
	}
	for _, name := range profile.environment {
		if !containsString(environment, name) {
			environment = append(environment, name)
		}
	}
	return projectrun.Agent{
		Command: found.ProviderBinary.Path, Args: append([]string(nil), profile.args...),
		Model: profile.model, ModelOptions: modelOptions, ProviderVersion: found.ProviderBinary.Version,
		Timeout: projectrun.Duration(DefaultTimeout), MaxStdoutBytes: DefaultProcessMaxStdout, MaxStderrBytes: DefaultMaxStderr,
		RuntimeFiles: []agentexec.RuntimeFile{runtimeFile(found.ProviderBinary)}, Environment: environment,
		CostMode: costModeField(profile.costMode), Pricing: profile.pricing,
	}
}

// costModeField keeps metered agents in the existing runtime shape.
func costModeField(mode string) string {
	if mode == projectrun.CostModeUnmetered {
		return projectrun.CostModeUnmetered
	}
	return ""
}

func cloneAgent(agent projectrun.Agent) projectrun.Agent {
	agent.Args = append([]string(nil), agent.Args...)
	agent.InstructionPaths = append([]string(nil), agent.InstructionPaths...)
	agent.RuntimeFiles = append([]agentexec.RuntimeFile(nil), agent.RuntimeFiles...)
	agent.Environment = append([]string(nil), agent.Environment...)
	if agent.AppServer != nil {
		settings := *agent.AppServer
		agent.AppServer = &settings
	}
	return agent
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// Doctor performs local prerequisite checks without reading provider
// credentials or invoking login/status commands.
func Doctor(project *projectwork.Project, options Options) (DoctorReport, error) {
	result := DoctorReport{APIVersion: "markitect.example.org/project-doctor/v1alpha1", Provider: options.Provider, Authentication: "not-verified"}
	if project == nil {
		return result, errors.New("active project is required")
	}
	discovery, err := Discover(options)
	if err != nil {
		return result, err
	}
	executorCheck := "codex-app-server"
	if discovery.Provider == ProviderProcess {
		executorCheck = "process-executor"
	}
	result.Checks = append(result.Checks,
		Check{Name: executorCheck, Status: "available", Detail: discovery.ProviderBinary.Path + " (" + discovery.ProviderBinary.Version + ")"},
		Check{Name: "provider-authentication", Status: "not-verified", Detail: "No login-status probe or credential/config read was performed."},
	)
	for _, check := range project.Report.Checks {
		if len(check.Command) == 0 {
			result.Checks = append(result.Checks, Check{Name: "declared-check:" + check.ID, Status: "missing", Detail: "check has no executable argv"})
			continue
		}
		if _, err := exec.LookPath(check.Command[0]); err != nil {
			result.Checks = append(result.Checks, Check{Name: "declared-check:" + check.ID, Status: "missing", Detail: "declared check executable is not on PATH: " + check.Command[0]})
		} else {
			result.Checks = append(result.Checks, Check{Name: "declared-check:" + check.ID, Status: "available", Detail: check.Command[0]})
		}
	}
	branchCmd := exec.Command("git", "-C", project.Root, "branch", "--show-current")
	branch, branchErr := branchCmd.Output()
	if branchErr != nil || strings.TrimSpace(string(branch)) == "" {
		result.Checks = append(result.Checks, Check{Name: "feature-branch", Status: "blocked", Detail: "repository is not on a named branch"})
	} else if strings.EqualFold(strings.TrimSpace(string(branch)), "main") || strings.EqualFold(strings.TrimSpace(string(branch)), "master") {
		result.Checks = append(result.Checks, Check{Name: "feature-branch", Status: "blocked", Detail: "select a non-protected feature branch before applying edits"})
	} else {
		result.Checks = append(result.Checks, Check{Name: "feature-branch", Status: "ready", Detail: strings.TrimSpace(string(branch))})
	}
	sort.Slice(result.Checks, func(i, j int) bool { return result.Checks[i].Name < result.Checks[j].Name })
	return result, nil
}

func runtimeFile(tool Tool) agentexec.RuntimeFile {
	return agentexec.RuntimeFile{Path: tool.Path, Mode: tool.Mode, Digest: tool.Digest}
}

// nativeInstructionFiles selects only existing, non-operational Codex
// instruction files owned by onboarding. Paths stay project-relative in the
// invocation contract; runtime pins bind their exact absolute bytes and modes.
func nativeInstructionFiles(project *projectwork.Project, provider string) ([]string, []agentexec.RuntimeFile, error) {
	if project == nil || project.Root == "" {
		return nil, nil, errors.New("native Codex instructions require an active project root")
	}
	if provider != "codex" {
		return nil, nil, errors.New("native Codex instructions currently support only Codex")
	}
	root, err := filepath.Abs(project.Root)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve project root for native instructions: %w", err)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&fs.ModeSymlink != 0 {
		return nil, nil, errors.New("native instruction project root must be a real directory")
	}
	var paths []string
	var files []agentexec.RuntimeFile
	for _, tool := range projectwork.ToolPaths(project.Config) {
		path := tool.Selector
		lower := strings.ToLower(filepath.ToSlash(path))
		if tool.Operational || tool.Owner != "project-onboarding" || !strings.HasSuffix(lower, ".md") || strings.HasPrefix(lower, ".markitect/") {
			continue
		}
		if path != "AGENTS.md" && !strings.HasPrefix(path, ".agents/") {
			continue
		}
		absolute := filepath.Join(root, filepath.FromSlash(path))
		current := root
		missing := false
		segments := strings.Split(filepath.FromSlash(path), string(filepath.Separator))
		for index, segment := range segments {
			current = filepath.Join(current, segment)
			info, statErr := os.Lstat(current)
			if errors.Is(statErr, os.ErrNotExist) {
				missing = true
				break
			}
			if statErr != nil {
				return nil, nil, fmt.Errorf("inspect native instruction %s: %w", path, statErr)
			}
			if info.Mode()&fs.ModeSymlink != 0 {
				return nil, nil, fmt.Errorf("native instruction %s contains a symlink", path)
			}
			if index < len(segments)-1 && !info.IsDir() {
				return nil, nil, fmt.Errorf("native instruction %s has a non-directory parent", path)
			}
		}
		if missing {
			continue
		}
		toolFile, err := inspectFile(absolute)
		if err != nil {
			return nil, nil, fmt.Errorf("inspect native instruction %s: %w", path, err)
		}
		// Keep the project-root spelling used by projectrun's workspace
		// validator; resolving a Windows junction here would make the absolute
		// pin differ textually from the same project-relative instruction path.
		toolFile.Path = filepath.Clean(absolute)
		paths = append(paths, filepath.ToSlash(path))
		files = append(files, runtimeFile(toolFile))
	}
	sort.Strings(paths)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return paths, files, nil
}

// Discover identifies the direct native Codex executable or the selected
// process executor. It invokes only Codex --version and never starts a process
// executor; it never checks login state or reads provider config.
func Discover(options Options) (Discovery, error) {
	return discoverWithVersion(options, version)
}

// discoverWithVersion keeps executable discovery separately testable from
// running a provider command. The compiled native App Server has no
// source-file dependency and does not require or inspect a Markitect checkout.
func discoverWithVersion(options Options, readVersion func(string) (string, error)) (Discovery, error) {
	var result Discovery
	if options.Provider == ProviderProcess {
		return discoverProcess(options)
	}
	if options.Provider != ProviderCodex {
		return result, unsupportedProvider(options.Provider)
	}
	providerPath := options.ProviderExecutable
	if providerPath == "" {
		providerPath = discoverProvider(options.Provider)
	}
	providerTool, err := inspectFile(providerPath)
	if err != nil {
		return result, fmt.Errorf("native %s executable not found; pass --provider-executable with its absolute path: %w", options.Provider, err)
	}
	if isCommandShim(providerTool.Path) {
		return result, errors.New("provider executable must be a direct native executable; .ps1/.cmd/.bat and script shims are not accepted")
	}
	providerTool.Version, err = readVersion(providerTool.Path)
	if err != nil {
		return result, fmt.Errorf("read selected provider version: %w", err)
	}
	if options.Provider == "codex" {
		providerTool.Version = "codex-cli " + providerTool.Version
	}
	result = Discovery{
		Provider: options.Provider, ProviderBinary: providerTool,
		Authentication:     "not-verified",
		AuthenticationNote: "Markitect does not read credentials or probe provider login status; use the provider's existing OS-default sign-in.",
	}
	return result, nil
}

// discoverProcess pins a bring-your-own executor by path and digest. Its
// version is the caller's declaration; Markitect does not start it here.
func discoverProcess(options Options) (Discovery, error) {
	if options.ProviderExecutable == "" {
		return Discovery{}, errors.New("a process executor requires --provider-executable with its absolute path")
	}
	tool, err := inspectFile(options.ProviderExecutable)
	if err != nil {
		return Discovery{}, fmt.Errorf("process executor executable: %w", err)
	}
	tool.Version = options.ProviderVersion
	if tool.Version == "" {
		tool.Version = UndeclaredProviderVersion
	}
	return Discovery{
		Provider: ProviderProcess, ProviderBinary: tool,
		Authentication:     "not-verified",
		AuthenticationNote: "Markitect passes only the configured environment names to a process executor and does not inspect its credentials.",
	}, nil
}

func inspectFile(raw string) (Tool, error) {
	var result Tool
	if raw == "" || !filepath.IsAbs(raw) {
		return result, errors.New("path must be absolute")
	}
	info, err := os.Lstat(raw)
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 {
		return result, errors.New("path must be a regular non-symlink file")
	}
	if info.Size() <= 0 || info.Size() > agentexec.MaxRuntimeAssetBytes {
		return result, errors.New("file size is outside the supported 1..512 MiB bound")
	}
	path, err := filepath.EvalSymlinks(raw)
	if err != nil {
		return result, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return result, err
	}
	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || opened.Size() != info.Size() || opened.Mode() != info.Mode() {
		return result, errors.New("runtime asset changed before fingerprinting")
	}
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(file, agentexec.MaxRuntimeAssetBytes+1))
	if err != nil {
		return result, err
	}
	if count > agentexec.MaxRuntimeAssetBytes {
		return result, errors.New("file exceeds the runtime asset size bound")
	}
	after, err := file.Stat()
	current, currentErr := os.Lstat(path)
	if err != nil || currentErr != nil || count != opened.Size() || after.Size() != opened.Size() || after.Mode() != opened.Mode() || !after.ModTime().Equal(opened.ModTime()) || !os.SameFile(opened, current) || current.Mode() != opened.Mode() || current.Size() != opened.Size() {
		return result, errors.New("runtime asset changed during fingerprinting")
	}
	mode := runtimeFileModeForSetup(info.Mode())
	return Tool{Path: path, Digest: "sha256:" + hex.EncodeToString(hash.Sum(nil)), Mode: mode}, nil
}

func runtimeFileModeForSetup(mode fs.FileMode) string {
	if runtime.GOOS == "windows" {
		if mode.Perm()&0200 == 0 {
			return "0444"
		}
		return "0644"
	}
	return fmt.Sprintf("%04o", mode.Perm())
}

func isCommandShim(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if runtime.GOOS == "windows" {
		return ext != ".exe" && ext != ".com"
	}
	if ext == ".cmd" || ext == ".bat" || ext == ".ps1" || ext == ".sh" {
		return true
	}
	file, err := os.Open(path)
	if err != nil {
		return true
	}
	defer file.Close()
	var prefix [2]byte
	_, _ = file.Read(prefix[:])
	return prefix == [2]byte{'#', '!'}
}

func discoverProvider(provider string) string {
	if runtime.GOOS == "windows" && provider == "codex" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			candidate := filepath.Join(appData, "npm", "node_modules", "@openai", "codex", "node_modules", "@openai", "codex-win32-x64", "vendor", "x86_64-pc-windows-msvc", "bin", "codex.exe")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	name := provider
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if found, err := exec.LookPath(name); err == nil {
		return found
	}
	return ""
}

func version(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, path, "--version")
	command.Env = []string{}
	output := &boundedVersionOutput{cancel: cancel}
	command.Stdout = output
	command.Stderr = io.Discard
	err := command.Run()
	if err != nil || output.overflow {
		return "", errors.New("selected executable did not return a bounded version response")
	}
	match := versionPattern.FindSubmatch(output.bytes)
	if len(match) != 2 {
		return "", errors.New("selected executable version response did not contain a semantic version")
	}
	return string(match[1]), nil
}

type boundedVersionOutput struct {
	bytes    []byte
	cancel   context.CancelFunc
	overflow bool
}

func (output *boundedVersionOutput) Write(data []byte) (int, error) {
	const limit = 8 << 10
	remaining := limit - len(output.bytes)
	if remaining <= 0 {
		output.overflow = true
		output.cancel()
		return 0, errors.New("version output exceeded bound")
	}
	if len(data) > remaining {
		output.bytes = append(output.bytes, data[:remaining]...)
		output.overflow = true
		output.cancel()
		return remaining, errors.New("version output exceeded bound")
	}
	output.bytes = append(output.bytes, data...)
	return len(data), nil
}
