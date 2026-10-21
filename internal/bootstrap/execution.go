package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	runport "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

type executionService struct {
	preparation *apply.PreparationService
	repository  *gitrepo.Repository
	agents      apply.AgentProfilePreparer
	checks      apply.CheckProfilePreparer
}

func newExecution(preparation *apply.PreparationService, repository *gitrepo.Repository, agents apply.AgentProfilePreparer, checks apply.CheckProfilePreparer) *executionService {
	return &executionService{preparation: preparation, repository: repository, agents: agents, checks: checks}
}

// applyPorts is everything one Apply runs with: the concrete adapters, the
// run's evidence and the explicit bounds of the invocation.
type applyPorts struct {
	RunID      string
	Agents     runport.Agents
	Workspaces runport.Workspaces
	Checks     runport.Checks
	Evidence   *fileEvidence
	Publisher  apply.Publisher
	Control    apply.ControlCheck
	RemoveHome func(context.Context, string) error
	// RequireVolumeSubpaths checks the container engine for volume subpath
	// mounts, which enforced ownership needs.
	RequireVolumeSubpaths func(context.Context) error
	Harvest               harvestLimits
	// Deadline is the operator's absolute wall-clock limit; zero means none.
	Deadline time.Time
	Clock    func() time.Time
}

// Execute prepares the committed repository, composes the ports and runs the
// Apply. Nothing starts before preparation passed, and only one Apply
// holds a state directory at a time.
func (s *executionService) Execute(ctx context.Context, request cli.ApplyRequest) (report cli.ApplyReport, err error) {
	report.FormatVersion, report.Operation, report.Status = 1, "apply", model.NotCompleted
	session, failed, err := s.open(ctx, request.Selection.Repository, request.RunConfig)
	if err != nil {
		if failed {
			report.Status = model.Failed
		}
		return report, err
	}
	var evidence *fileEvidence
	defer func() { err = finishApplyReport(&report, err, evidence, session.lock.Release) }()
	prepared, err := s.preparation.CaptureSettings(ctx, apply.CaptureRequest{Selection: request.Selection}, model.Limits{}, s.agents, s.checks)
	report.Preparation = capturedExecutionReport(prepared, session.cfg.ProductBuild)
	if prepared.Status() != model.Passed || err != nil {
		report.Status = prepared.Status()
		return report, err
	}
	ports, err := s.ports(session.cfg, prepared, session.lock, session.runID)
	if err != nil {
		return report, err
	}
	report.EvidenceDirectory = ports.Evidence.Dir()
	evidence = ports.Evidence
	result, err := s.applyAndPublish(ctx, request, prepared, session.cfg, ports)
	if result.Binding.Strategy != "" {
		report.Binding = &result.Binding
	}
	if result.Outcome.Status != "" {
		report.Run = cli.NewRunReport(ports.RunID, result.Outcome)
	}
	report.Publication = result.Receipt
	report.RetainedWork = result.RetainedWork
	report.Status = cli.ApplyStatus(result.Outcome, result.Receipt, err)
	return report, err
}

// finishApplyReport includes event-log and lock failures before projecting the
// final report. Its run-specific file can be written after the lock is released.
// A definite rejection stays failed; a technical error cannot leave success.
func finishApplyReport(report *cli.ApplyReport, err error, evidence *fileEvidence, releaseLock func() error) error {
	if evidence != nil {
		err = errors.Join(err, evidence.Close())
	}
	err = errors.Join(err, releaseLock())
	fail := func(err error) {
		report.Error = err.Error()
		if report.Status != model.Failed {
			report.Status = model.NotCompleted
		}
	}
	if err != nil {
		fail(err)
	}
	if evidence != nil {
		if writeErr := evidence.WriteReport(cli.ReportValue(*report)); writeErr != nil {
			err = errors.Join(err, writeErr)
			fail(err)
		}
	}
	return err
}

// Extract prepares the committed repository as the start of an Extract, composes
// the ports and runs both phases. Like Apply it starts nothing before the
// preparation passed, and only one run holds a state directory at a time.
func (s *executionService) Extract(ctx context.Context, request cli.ExtractRequest) (report cli.ExtractReport, err error) {
	report.FormatVersion, report.Operation, report.Status = 1, "extract", model.NotCompleted
	report.Deviations = []apply.ArtifactDeviation{}
	session, failed, err := s.open(ctx, request.Selection.Repository, request.RunConfig)
	if err != nil {
		if failed {
			report.Status = model.Failed
		}
		return report, err
	}
	defer func() { err = errors.Join(err, session.lock.Release()) }()
	prepared, err := s.preparation.CaptureExtract(ctx, apply.CaptureRequest{Selection: request.Selection}, model.Limits{}, s.agents, s.checks)
	report.Preparation = capturedExecutionReport(prepared, session.cfg.ProductBuild)
	if prepared.Status() != model.Passed || err != nil {
		report.Status = prepared.Status()
		return report, err
	}
	ports, err := s.ports(session.cfg, prepared, session.lock, session.runID)
	if err != nil {
		return report, err
	}
	attempt, ok := prepared.Prepared()
	if !ok {
		return report, errors.New("complete captured settings are required")
	}
	preparation := report.Preparation
	report.EvidenceDirectory = ports.Evidence.Dir()
	defer func() {
		if err != nil {
			report.Error = err.Error()
		}
		if writeErr := ports.Evidence.WriteReport(cli.ReportValue(report)); writeErr != nil {
			err = errors.Join(err, writeErr)
			report.Status = model.NotCompleted
		}
		err = errors.Join(err, ports.Evidence.Close())
	}()
	result, err := apply.Extract(ctx, apply.ExtractRequest{
		RunID: ports.RunID, Prepared: attempt, Selection: request.Selection,
		OwnerRequest: session.cfg.OwnerRequest, Deadline: ports.Deadline,
		MaxEntries: ports.Harvest.MaxEntries, MaxBytes: ports.Harvest.MaxBytes,
		Agents: ports.Agents, Workspaces: ports.Workspaces, Evidence: ports.Evidence,
		Publisher: ports.Publisher, Control: ports.Control, Clock: ports.Clock, RemoveHome: ports.RemoveHome, RequireVolumeSubpaths: ports.RequireVolumeSubpaths,
	})
	report = cli.NewExtractReport(ports.RunID, result)
	report.FormatVersion, report.Operation, report.Preparation, report.EvidenceDirectory = 1, "extract", preparation, ports.Evidence.Dir()
	report.Status = cli.ExtractStatus(result, err)
	return report, err
}

// runSession is what every run command holds before it prepares: the run
// configuration, the exclusive state directory and the run identity.
type runSession struct {
	cfg   runConfig
	lock  *stateLock
	runID string
}

// open loads the run configuration, takes the state directory of the run and
// records the run's identity in it.
// failed says that the error is a definite refusal of the invocation rather
// than an incomplete observation. The caller releases the lock.
func (s *executionService) open(ctx context.Context, repository, runConfigPath string) (session runSession, failed bool, err error) {
	cfg, err := loadRunConfig(ctx, runConfigPath)
	if err != nil {
		return session, errors.Is(err, errRunConfig), err
	}
	if s == nil || s.preparation == nil || s.repository == nil {
		return session, false, errors.New("concrete execution composition is unavailable")
	}
	if err := checkStateOutsideRepository(cfg.StateDirectory, repository); err != nil {
		return session, true, err
	}
	lock, err := lockStateWithin(ctx, cfg.StateDirectory, startLockWait)
	if err != nil {
		return session, false, err
	}
	fail := func(err error) (runSession, bool, error) {
		return session, false, errors.Join(err, lock.Release())
	}
	runID, err := newRunID(time.Now())
	if err != nil {
		return fail(err)
	}
	// The cockpit tells the run that holds the lock by this name.
	if err := writeStateFile(filepath.Join(cfg.StateDirectory, currentRunFile), []byte(runID+"\n"), true); err != nil {
		return fail(err)
	}
	return runSession{cfg: cfg, lock: lock, runID: runID}, false, nil
}

// applyAndPublish runs the Apply use case over the captured preparation with
// the composed ports.
func (s *executionService) applyAndPublish(ctx context.Context, request cli.ApplyRequest, captured apply.CapturedSettingsResult, cfg runConfig, ports applyPorts) (apply.ApplyResult, error) {
	prepared, ok := captured.Prepared()
	if !ok {
		return apply.ApplyResult{}, errors.New("complete captured settings are required")
	}
	return apply.Apply(ctx, applyRequest(request, prepared, cfg, ports))
}

// applyRequest is the one request every entry of an Apply runs with, the
// product command and the evaluation harness alike: the captured preparation
// with its strategy and settings, the invocation's build identity and bounds,
// and the composed ports.
func applyRequest(request cli.ApplyRequest, prepared apply.CapturedSettingsAttempt, cfg runConfig, ports applyPorts) apply.ApplyRequest {
	return apply.ApplyRequest{
		RunID: ports.RunID, Prepared: prepared, Selection: request.Selection, Build: request.Build,
		OwnerRequest: cfg.OwnerRequest, Deadline: ports.Deadline,
		MaxEntries: ports.Harvest.MaxEntries, MaxBytes: ports.Harvest.MaxBytes,
		CheckRequest: func(selected apply.CheckSelection) (runport.CheckRequest, error) {
			return executionCheck(selected, cfg)
		},
		Agents: ports.Agents, Workspaces: ports.Workspaces, Runner: ports.Checks, Evidence: ports.Evidence,
		Publisher: ports.Publisher, Control: ports.Control, Clock: ports.Clock, RemoveHome: ports.RemoveHome, RequireVolumeSubpaths: ports.RequireVolumeSubpaths,
	}
}

// checkStateOutsideRepository refuses a state directory inside the repository
// or containing it: evidence and channel files must not become repository
// content, and the state directory is not the repository's to own.
func checkStateOutsideRepository(stateDirectory, repository string) error {
	if repository == "" {
		return errors.New("the repository root is required")
	}
	state, repo := resolvedPath(stateDirectory), resolvedPath(repository)
	for _, pair := range [][2]string{{repo, state}, {state, repo}} {
		if rel, err := filepath.Rel(pair[0], pair[1]); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errors.New("the state directory must not overlap the repository: it is inside it or contains it")
		}
	}
	return nil
}

// resolvedPath resolves symbolic links of the longest existing prefix, so a
// link cannot hide an overlap; the missing rest is appended unchanged.
func resolvedPath(path string) string {
	path = filepath.Clean(path)
	rest := ""
	for {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			return filepath.Join(resolved, rest)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return filepath.Join(path, rest)
		}
		rest, path = filepath.Join(filepath.Base(path), rest), parent
	}
}

func capturedExecutionReport(prepared apply.CapturedSettingsResult, productBuild string) any {
	attempts := []any{}
	for _, attempt := range prepared.Attempts() {
		structural := attempt.Structural()
		checked := structural.Model().Checked()
		entry := map[string]any{
			"head":        structural.Model().Inputs().Head,
			"modelStatus": checked.Status, "modelDiagnostics": checked.Diagnostics,
			"executionDiagnostics": structural.Execution().Diagnostics(),
			"registryDiagnostics":  structural.Registry().Diagnostics(),
			"fileDiagnostics":      structural.Files().Diagnostics(),
			"contextStatus":        structural.Contexts().Status(),
			"contextDiagnostics":   structural.Contexts().Diagnostics(),
			"defaultsStatus":       attempt.Defaults().Status(),
			"defaultsDiagnostics":  attempt.Defaults().Diagnostics(),
			"settingsDiagnostics":  attempt.Settings().Diagnostics(),
		}
		if attempt.Error() != nil {
			entry["error"] = attempt.Error().Error()
		}
		attempts = append(attempts, entry)
	}
	return map[string]any{"status": prepared.Status(), "productBuildAssertion": productBuild, "attempts": attempts}
}

// ports composes the concrete adapters from one complete capture. Every
// selected profile is validated against the explicit run engine before any
// adapter is constructed.
func (s *executionService) ports(cfg runConfig, prepared apply.CapturedSettingsResult, lock *stateLock, runID string) (applyPorts, error) {
	a, ok := prepared.Prepared()
	if !ok {
		return applyPorts{}, errors.New("complete captured settings are required")
	}
	selectedChecks, complete := a.Settings().Checks()
	if !complete {
		return applyPorts{}, errors.New("configured check selections are incomplete")
	}
	for _, selected := range selectedChecks {
		if _, err := executionCheck(selected, cfg); err != nil {
			return applyPorts{}, err
		}
	}
	engine, err := container.NewEngine(cfg.EngineExecutable)
	if err != nil {
		return applyPorts{}, err
	}
	agents, err := newAgents(engine, a.Settings().ProfileObservations(), cfg)
	if err != nil {
		return applyPorts{}, err
	}
	helper := cfg.WorkspaceHelper
	helper.Workdir = "/work"
	workspaces, err := gitrepo.NewWorkspaces(engine, helper)
	if err != nil {
		return applyPorts{}, err
	}
	checker, err := checks.NewRunner(engine, workspaces)
	if err != nil {
		return applyPorts{}, err
	}
	evidence, err := newFileEvidence(cfg.StateDirectory, runID)
	if err != nil {
		return applyPorts{}, err
	}
	return applyPorts{RunID: runID, Agents: agents, Workspaces: workspaces, Checks: checker, Evidence: evidence,
		Publisher: s.repository, Control: lock, RemoveHome: engine.RemoveVolume, RequireVolumeSubpaths: engine.RequireVolumeSubpaths, Harvest: cfg.Harvest, Deadline: cfg.deadline(), Clock: time.Now}, nil
}

// newAgents builds one adapter per provider that a selected profile uses and
// routes each profile to its adapter. The adapters resolve credentials only
// when a turn starts.
func newAgents(engine *container.Engine, observed []apply.PreparedProfile, cfg runConfig) (runport.Agents, error) {
	codexCredentials, claudeCredentials, err := newCredentialResolvers(cfg.Credentials)
	if err != nil {
		return nil, err
	}
	codexProfiles := map[string]codexappserver.Profile{}
	claudeProfiles := map[string]claudecode.Profile{}
	for _, profile := range observed {
		input := profile.Input()
		if input.Namespace() != "agentProfiles" {
			continue
		}
		if !profileEngine(profile, cfg.EngineExecutable) {
			return nil, errors.New("prepared Agent engine differs from the explicit run engine")
		}
		switch input.Section() {
		case "codexAppServer":
			var p codexappserver.Profile
			if p, err = codexappserver.ProfileFromPrepared(profile); err == nil {
				err = credentialMapped(cfg, p.CredentialRef, "codex-app-server")
			}
			codexProfiles[input.Name()] = p
		case "claudeCode":
			var p claudecode.Profile
			if p, err = claudecode.ProfileFromPrepared(profile); err == nil {
				err = credentialMapped(cfg, p.CredentialRef, "claude-code")
			}
			claudeProfiles[input.Name()] = p
		default:
			err = errors.New("selected Agent profile has no supported execution adapter")
		}
		if err != nil {
			return nil, fmt.Errorf("agent profile %q: %w", input.Name(), err)
		}
	}
	routes := map[string]runport.Agents{}
	if len(codexProfiles) > 0 {
		codex, err := codexappserver.NewAgents(engine, codexProfiles, codexCredentials)
		if err != nil {
			return nil, err
		}
		for name := range codexProfiles {
			routes[name] = codex
		}
	}
	if len(claudeProfiles) > 0 {
		claude, err := claudecode.NewAgents(engine, claudeProfiles, claudeCredentials)
		if err != nil {
			return nil, err
		}
		for name := range claudeProfiles {
			routes[name] = claude
		}
	}
	return newAgentsRouter(routes), nil
}

// credentialMapped requires the explicit run configuration to map the profile's
// exact credential reference to a credential of the profile's provider.
func credentialMapped(cfg runConfig, ref, provider string) error {
	if source, found := cfg.Credentials[ref]; !found || source.Provider != provider {
		return errors.New("prepared Agent has no exact explicit credential mapping")
	}
	return nil
}

func executionCheck(selected apply.CheckSelection, cfg runConfig) (runport.CheckRequest, error) {
	prepared := selected.Profile()
	if prepared.Input().Section() != "container" || !profileEngine(prepared, cfg.EngineExecutable) {
		return runport.CheckRequest{}, errors.New("execution requires a prepared container check using the explicit run engine")
	}
	profile, err := checks.ContainerProfileFromPrepared(prepared)
	if err != nil {
		return runport.CheckRequest{}, err
	}
	return profile.Request(selected.Declaration(), snapshot.Tree{}) // The engine fills the candidate tree.
}

// newRunID names one run: the start time makes evidence directories sort, the
// random suffix keeps two Applies in the same second apart.
func newRunID(now time.Time) (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	return now.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(suffix[:]), nil
}

// Only this already validated captured literal is needed for Runtime profile composition;
// provider/profile parsing remains entirely in its adapter.
func profileEngine(prepared apply.PreparedProfile, want string) bool {
	if _, complete := prepared.Identity(); !complete {
		return false
	}
	inline := prepared.Input().Inline()
	if inline == nil || inline.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(inline.Content); i += 2 {
		if inline.Content[i].Value == "engineExecutable" {
			value := inline.Content[i+1]
			return value.Kind == yaml.ScalarNode && value.Tag == "!!str" && value.Value == want
		}
	}
	return false
}
