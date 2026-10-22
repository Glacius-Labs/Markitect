package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"
	"sync"
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
	RunID                 string
	Agents                runport.Agents
	AssignmentAgents      runport.AssignmentAgents
	AssignmentWorkspaces  apply.RealizationWorkspaces
	AssignmentIntegration runport.AssignmentIntegration
	CreateHome            func(context.Context, string) error
	CleanupHome           func(context.Context, string) error
	Workspaces            runport.Workspaces
	Checks                runport.Checks
	Evidence              *fileEvidence
	Publisher             apply.Publisher
	Control               apply.ControlCheck
	RemoveHome            func(context.Context, string) error
	// RequireVolumeSubpaths remains for the legacy Turn-based Extract path.
	// Assignment Apply mounts complete, independent workspace volumes.
	RequireVolumeSubpaths func(context.Context) error
	Harvest               harvestLimits
	// Deadline is the operator's absolute wall-clock limit; zero means none.
	Deadline     time.Time
	WorkDeadline time.Time
	Clock        func() time.Time
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
	if session.cfg.ProductBuild != request.Build {
		return report, errors.New("run configuration product build differs from this executable")
	}
	if session.cfg.WorkDeadline != "" {
		return report, errors.New("workDeadline is a legacy Pass limit and cannot bound assignment Apply; use deadline")
	}
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
	result, receipt, runErr := s.realizeAndPublish(ctx, request, prepared, session.cfg, ports)
	report.Assignment = cli.NewAssignmentReport(result)
	if observation, ok := ports.AssignmentWorkspaces.(interface {
		GitContainerID() string
		GitExecCount() int
	}); ok {
		report.Assignment.GitContainerID = observation.GitContainerID()
		report.Assignment.GitExecCount = observation.GitExecCount()
	}
	report.Publication = receipt
	report.Status = cli.AssignmentApplyStatus(result, receipt, runErr)
	err = runErr
	return report, err
}

func (s *executionService) realizeAndPublish(ctx context.Context, request cli.ApplyRequest, captured apply.CapturedSettingsResult, cfg runConfig, ports applyPorts) (result apply.AssignmentRealizationResult, receipt *apply.PublicationReceipt, err error) {
	prepared, ok := captured.Prepared()
	if !ok {
		return apply.AssignmentRealizationResult{}, nil, errors.New("complete captured settings are required")
	}
	if cfg.ProductBuild != request.Build {
		return apply.AssignmentRealizationResult{}, nil, errors.New("run configuration product build differs from this executable")
	}
	if cfg.MaxMessages == 0 || cfg.MessageTimeoutSeconds == 0 {
		return apply.AssignmentRealizationResult{}, nil, errors.New("assignment Apply requires finite maxMessages and messageTimeoutSeconds")
	}
	budget, complete := prepared.Settings().Budget()
	if !complete || budget.ExecutionTime.Value <= 0 {
		return apply.AssignmentRealizationResult{}, nil, errors.New("assignment Apply requires the prepared whole-run execution time")
	}
	deadline := time.Now().Add(budget.ExecutionTime.Value)
	if operator := cfg.deadline(); !operator.IsZero() && operator.Before(deadline) {
		deadline = operator
	}
	if !time.Now().Before(deadline) {
		return apply.AssignmentRealizationResult{}, nil, errors.New("assignment Apply deadline already passed")
	}
	bounded, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if ports.CleanupHome == nil {
		return result, nil, errors.New("assignment Apply requires observed provider home cleanup")
	}
	git, ok := ports.AssignmentWorkspaces.(interface {
		StartGit(context.Context, []runport.WorkspaceID) error
		CloseGit(context.Context) error
	})
	if !ok {
		return result, nil, errors.New("assignment workspaces need one persistent Git container")
	}
	ids, err := apply.AssignmentWorkspaceIDs(prepared, ports.RunID, cfg.MaxMessages)
	if err != nil {
		return result, nil, err
	}
	err = withPersistentGit(bounded, git, ids, func() error {
		var runErr error
		result, runErr = apply.RealizeAssignments(bounded, apply.AssignmentRealizationRequest{
			ID: ports.RunID, Build: request.Build, Prepared: prepared, OwnerRequest: cfg.OwnerRequest,
			MaxMessages: cfg.MaxMessages, MessageTimeout: time.Duration(cfg.MessageTimeoutSeconds) * time.Second,
			MaxEntries: ports.Harvest.MaxEntries, MaxBytes: ports.Harvest.MaxBytes,
			TranscriptDir: filepath.Join(ports.Evidence.Dir(), "transcripts"), Workspaces: ports.AssignmentWorkspaces,
			Agents: ports.AssignmentAgents, Integration: ports.AssignmentIntegration, Checks: ports.Checks,
			CheckRequest: func(selected apply.CheckSelection) (runport.CheckRequest, error) {
				return executionCheck(selected, cfg)
			},
			CreateHome: ports.CreateHome,
		})
		if runErr != nil || result.AcceptedCommit == "" {
			return runErr
		}
		receipt, runErr = apply.PublishAssignment(bounded, ports.RunID, request.Selection, prepared, result, ports.Publisher, ports.Control)
		if runErr == nil && receipt != nil && (receipt.Outcome == apply.PublicationApplied || receipt.Outcome == apply.PublicationNoChange) {
			for _, home := range assignmentHomes(result) {
				if cleanErr := ports.CleanupHome(bounded, home); cleanErr != nil {
					runErr = errors.Join(runErr, fmt.Errorf("provider home %s: %w", home, cleanErr))
				} else {
					result.HomesRemoved = append(result.HomesRemoved, home)
				}
			}
		}
		return runErr
	})
	return result, receipt, err
}

type persistentGit interface {
	StartGit(context.Context, []runport.WorkspaceID) error
	CloseGit(context.Context) error
}

// A failed StartGit may already hold a container. Close it on every exit,
// including failed startup, realization, publication, and home cleanup.
func withPersistentGit(ctx context.Context, git persistentGit, ids []runport.WorkspaceID, work func() error) (err error) {
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		err = errors.Join(err, git.CloseGit(closeCtx))
	}()
	if err := git.StartGit(ctx, ids); err != nil {
		return err
	}
	return work()
}

func assignmentHomes(result apply.AssignmentRealizationResult) []string {
	seen := map[string]bool{}
	var homes []string
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			homes = append(homes, name)
		}
	}
	add(result.Assessment.Home)
	if result.Work != nil {
		for _, name := range result.Work.Homes() {
			add(name)
		}
	}
	for _, review := range result.Reviews {
		add(review.Home)
	}
	return homes
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

// applyRequest keeps the legacy captured-run evaluation entries wired while
// the product Apply uses persistent assignments through realizeAndPublish.
func applyRequest(request cli.ApplyRequest, prepared apply.CapturedSettingsAttempt, cfg runConfig, ports applyPorts) apply.ApplyRequest {
	return apply.ApplyRequest{
		RunID: ports.RunID, Prepared: prepared, Selection: request.Selection, Build: request.Build,
		OwnerRequest: cfg.OwnerRequest, Deadline: ports.Deadline, WorkDeadline: ports.WorkDeadline,
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
	assignmentAgents, err := newAssignmentAgents(engine, a.Settings().ProfileObservations(), cfg)
	if err != nil {
		return applyPorts{}, err
	}
	helper := cfg.WorkspaceHelper
	helper.Workdir = "/work"
	workspaces, err := gitrepo.NewWorkspaces(engine, helper)
	if err != nil {
		return applyPorts{}, err
	}
	checker, err := checks.NewRunner(engine, workspaces, cfg.WorkspaceHelper.Labels)
	if err != nil {
		return applyPorts{}, err
	}
	evidence, err := newFileEvidence(cfg.StateDirectory, runID)
	if err != nil {
		return applyPorts{}, err
	}
	var homesMu sync.Mutex
	createdHomes := map[string]map[string]string{}
	createHome := func(ctx context.Context, name string) error {
		observed, err := engine.CreateVolumeExclusive(ctx, name, map[string]string{"markitect.assignment-run": runID})
		if err != nil {
			return err
		}
		if !observed.Exists || observed.Name != name || observed.Labels["markitect.assignment-run"] != runID {
			return errors.New("provider home creation ownership is not established")
		}
		homesMu.Lock()
		createdHomes[name] = maps.Clone(observed.Labels)
		homesMu.Unlock()
		return nil
	}
	cleanupHome := func(ctx context.Context, name string) error {
		homesMu.Lock()
		expected, known := createdHomes[name]
		expected = maps.Clone(expected)
		homesMu.Unlock()
		if !known {
			return errors.New("provider home has no exact creation receipt")
		}
		observed, err := engine.InspectVolume(ctx, name)
		if err != nil || !observed.Exists || observed.Name != name || !maps.Equal(observed.Labels, expected) {
			return errors.Join(err, errors.New("provider home ownership is not established"))
		}
		mounts, err := engine.VolumeMounts(ctx, name)
		if err != nil || len(mounts) != 0 {
			return errors.Join(err, errors.New("provider home remains mounted or mount census failed"))
		}
		removeErr := engine.RemoveVolume(ctx, name)
		after, inspectErr := engine.InspectVolume(ctx, name)
		if inspectErr != nil || after.Exists {
			return errors.Join(removeErr, inspectErr, errors.New("provider home removal is not established"))
		}
		if removeErr == nil {
			homesMu.Lock()
			delete(createdHomes, name)
			homesMu.Unlock()
		}
		return removeErr
	}
	return applyPorts{RunID: runID, Agents: agents, AssignmentAgents: assignmentAgents, Workspaces: workspaces, AssignmentWorkspaces: workspaces, AssignmentIntegration: workspaces, CreateHome: createHome, CleanupHome: cleanupHome, Checks: checker, Evidence: evidence,
		Publisher: s.repository, Control: lock, RemoveHome: engine.RemoveVolume, RequireVolumeSubpaths: engine.RequireVolumeSubpaths, Harvest: cfg.Harvest, Deadline: cfg.deadline(), WorkDeadline: cfg.workDeadline(), Clock: time.Now}, nil
}

// newAgents builds one adapter per provider that a selected profile uses and
// routes each profile to its adapter. The adapters resolve credentials only
// when a turn starts.
func newAgents(engine *container.Engine, observed []apply.PreparedProfile, cfg runConfig) (runport.Agents, error) {
	routes, err := newAgentRoutes(engine, observed, cfg)
	if err != nil {
		return nil, err
	}
	legacy := make(map[string]runport.Agents, len(routes))
	for name, agent := range routes {
		legacy[name] = agent
	}
	return newAgentsRouter(legacy), nil
}

type executionAgent interface {
	runport.Agents
	runport.AssignmentAgents
}

func newAssignmentAgents(engine *container.Engine, observed []apply.PreparedProfile, cfg runConfig) (runport.AssignmentAgents, error) {
	routes, err := newAgentRoutes(engine, observed, cfg)
	if err != nil {
		return nil, err
	}
	assignments := make(map[string]runport.AssignmentAgents, len(routes))
	for name, agent := range routes {
		assignments[name] = agent
	}
	return assignmentAgentsRouter{routes: assignments}, nil
}

func newAgentRoutes(engine *container.Engine, observed []apply.PreparedProfile, cfg runConfig) (map[string]executionAgent, error) {
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
	routes := map[string]executionAgent{}
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
	return routes, nil
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
