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

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
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
)

type Options struct {
	Provider               string `json:"provider"`
	Model                  string `json:"model"`
	Effort                 string `json:"effort"`
	CodexProfile           string `json:"codexProfile"`
	ProviderExecutable     string `json:"providerExecutable"`
	InputMicrosPerMillion  int64  `json:"inputMicrosPerMillion"`
	OutputMicrosPerMillion int64  `json:"outputMicrosPerMillion"`
	MaxCostMicros          int64  `json:"maxCostMicros"`
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
	APIVersion   string               `json:"apiVersion"`
	Discovery    Discovery            `json:"discovery"`
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
	var result Preview
	if project == nil {
		return result, errors.New("active project is required")
	}
	options, err := normalizeOptions(options)
	if err != nil {
		return result, err
	}
	discovery, err := Discover(options)
	if err != nil {
		return result, err
	}
	runtimeConfig, err := BuildRuntime(project, options, discovery)
	if err != nil {
		return result, err
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
		PricingBasis: "Caller-supplied budget estimate; not a provider price quote, invoice, or hard billing cap.",
		Mutation:     mutation, EditPlan: plan,
	}, nil
}

func BuildRuntime(project *projectwork.Project, options Options, found Discovery) (projectrun.Runtime, error) {
	var config projectrun.Runtime
	if project == nil || len(project.Report.Managers) == 0 {
		return config, errors.New("active project must contain at least one Manager")
	}
	if options.Provider != "codex" {
		return config, errors.New("project setup currently supports the native Codex App Server only; other providers are unsupported")
	}
	options, err := normalizeOptions(options)
	if err != nil {
		return config, err
	}
	if strings.TrimSpace(options.Model) == "" || options.Model != strings.TrimSpace(options.Model) {
		return config, errors.New("model must be nonempty and have no surrounding whitespace")
	}
	if options.Effort == "" {
		options.Effort = "high"
	}
	if options.Effort != "high" {
		return config, errors.New("setup currently supports only --effort high")
	}
	if options.CodexProfile != "luna-high" || options.Model != "gpt-6-luna" || options.Effort != "high" {
		return config, errors.New("native Codex App Server setup requires the luna-high model preset with model gpt-6-luna and effort high")
	}
	if found.ProviderBinary.Version != "codex-cli 0.162.0" {
		return config, errors.New("native Codex setup requires Codex CLI 0.162.0")
	}
	if options.InputMicrosPerMillion < 0 || options.OutputMicrosPerMillion < 0 ||
		(options.InputMicrosPerMillion == 0 && options.OutputMicrosPerMillion == 0) || options.MaxCostMicros <= 0 {
		return config, errors.New("explicit nonnegative input/output rates and a positive max-cost-micros budget are required")
	}
	if found.Provider != options.Provider || found.ProviderBinary.Path == "" || !filepath.IsAbs(found.ProviderBinary.Path) {
		return config, errors.New("tool discovery does not match the selected provider")
	}
	instructionPaths, instructionFiles, instructionErr := nativeInstructionFiles(project, options.Provider)
	if instructionErr != nil {
		return config, instructionErr
	}
	if len(instructionPaths) == 0 {
		return config, errors.New("native Codex setup requires existing generated Codex project instructions; run project onboard first")
	}
	runtimeFiles := append([]agentexec.RuntimeFile{runtimeFile(found.ProviderBinary)}, instructionFiles...)
	environment := []string{"PATH", "TEMP", "TMP"}
	if runtime.GOOS == "windows" {
		environment = append(environment, "SystemRoot", "USERPROFILE", "APPDATA", "LOCALAPPDATA")
	} else {
		environment = append(environment, "HOME")
	}
	pricing := projectrun.Pricing{InputMicrosPerMillion: options.InputMicrosPerMillion, OutputMicrosPerMillion: options.OutputMicrosPerMillion}
	agents := make(map[string]projectrun.Agent, len(project.Report.Managers))
	reviewAgents := make(map[string]projectrun.Agent, len(project.Report.Managers))
	for _, manager := range project.Report.Managers {
		if manager.ID == "" {
			return config, errors.New("active project has a Manager with an empty ID")
		}
		worker := selectedAgent(found, options.Model, options.Effort, runtimeFiles, environment, pricing)
		reviewer := selectedAgent(found, options.Model, options.Effort, runtimeFiles, environment, pricing)
		worker.WorkspaceMode = "git"
		worker.InstructionPaths = append([]string(nil), instructionPaths...)
		reviewer.WorkspaceMode = "git"
		reviewer.InstructionPaths = append([]string(nil), instructionPaths...)
		agents[manager.ID] = worker
		reviewAgents[manager.ID] = reviewer
	}
	config = projectrun.Runtime{
		APIVersion: projectrun.APIVersion, Mode: projectrun.ModeControlledLocal, RequireIsolation: false,
		Agents: agents,
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
	return config, nil
}

func normalizeOptions(options Options) (Options, error) {
	if options.Provider != "codex" {
		return options, errors.New("project setup currently supports native Codex only")
	}
	if options.CodexProfile == "" {
		options.CodexProfile = "luna-high"
	}
	if options.CodexProfile != "luna-high" {
		return options, errors.New("--codex-profile currently supports only luna-high")
	}
	return options, nil
}

func selectedAgent(found Discovery, model, effort string, files []agentexec.RuntimeFile, environment []string, pricing projectrun.Pricing) projectrun.Agent {
	return projectrun.Agent{
		Command: found.ProviderBinary.Path, Transport: projectrun.TransportCodexAppServer,
		AppServer: &projectrun.AppServerSettings{
			ReasoningEffort: effort,
			// Empty PermissionProfile preserves the user's existing Codex boundary.
			Helpers:       projectrun.AppServerHelpers{Enabled: true, MaxStartRequests: DefaultMaxHelperStarts, MaxDepth: 1},
			MaxEventBytes: DefaultMaxEventBytes,
		},
		Model: model, ProviderVersion: found.ProviderBinary.Version,
		Timeout: projectrun.Duration(DefaultTimeout), MaxStdoutBytes: DefaultMaxStdout, MaxStderrBytes: DefaultMaxStderr,
		RuntimeFiles: append([]agentexec.RuntimeFile(nil), files...), Environment: append([]string(nil), environment...), Pricing: pricing,
	}
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
	result.Checks = append(result.Checks,
		Check{Name: "codex-app-server", Status: "available", Detail: discovery.ProviderBinary.Path + " (" + discovery.ProviderBinary.Version + ")"},
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

// Discover identifies the direct native Codex executable. It invokes only
// --version; it never checks login state or reads provider config.
func Discover(options Options) (Discovery, error) {
	return discoverWithVersion(options, version)
}

// discoverWithVersion keeps executable discovery separately testable from
// running a provider command. The compiled native App Server has no
// source-file dependency and does not require or inspect a Markitect checkout.
func discoverWithVersion(options Options, readVersion func(string) (string, error)) (Discovery, error) {
	var result Discovery
	if options.Provider != "codex" {
		return result, errors.New("native runtime discovery currently supports Codex App Server only; other providers are unsupported")
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
	mode := "0644"
	if info.Mode().Perm()&0111 != 0 {
		mode = fmt.Sprintf("%04o", info.Mode().Perm())
	} else if runtime.GOOS != "windows" && strings.HasSuffix(strings.ToLower(path), ".py") {
		mode = fmt.Sprintf("%04o", info.Mode().Perm())
	}
	return Tool{Path: path, Digest: "sha256:" + hex.EncodeToString(hash.Sum(nil)), Mode: mode}, nil
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
