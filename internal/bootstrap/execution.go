package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	stderr      io.Writer
}

func newExecution(preparation *apply.PreparationService, repository *gitrepo.Repository, agents apply.AgentProfilePreparer, checks apply.CheckProfilePreparer, stderr io.Writer) *executionService {
	return &executionService{preparation: preparation, repository: repository, agents: agents, checks: checks, stderr: stderr}
}

// applyPorts is everything one Apply runs with: the concrete adapters, the
// run's evidence and owner channel, and the explicit bounds of the invocation.
type applyPorts struct {
	RunID      string
	Agents     runport.Agents
	Workspaces runport.Workspaces
	Checks     runport.Checks
	Owner      runport.Owner
	Evidence   *fileEvidence
	Publisher  apply.Publisher
	Harvest    harvestLimits
	// Deadline is the operator's absolute wall-clock limit; zero means none.
	Deadline time.Time
	Clock    func() time.Time
}

// Execute prepares the committed repository, composes the ports and runs the
// reconciliation. Nothing starts before preparation passed, and only one Apply
// holds a state directory at a time.
func (s *executionService) Execute(ctx context.Context, request cli.ApplyRequest) (report cli.ApplyReport, err error) {
	report.Status = model.NotCompleted
	cfg, err := loadRunConfig(ctx, request.RunConfig)
	if err != nil {
		if errors.Is(err, errRunConfig) {
			report.Status = model.Failed
		}
		return report, err
	}
	if s == nil || s.preparation == nil || s.repository == nil {
		return report, errors.New("concrete execution composition is unavailable")
	}
	lock, err := lockState(cfg.StateDirectory)
	if err != nil {
		return report, err
	}
	defer func() { err = errors.Join(err, lock.Release()) }()
	prepared, err := s.preparation.CaptureSettings(ctx, apply.CaptureRequest{Selection: request.Selection}, model.Limits{}, s.agents, s.checks)
	report.Preparation = capturedExecutionReport(prepared, cfg.ProductBuild)
	if prepared.Status() != model.Passed || err != nil {
		report.Status = prepared.Status()
		return report, err
	}
	ports, err := s.ports(cfg, request.RunConfig, prepared)
	if err != nil {
		return report, err
	}
	report.EvidenceDirectory = ports.Evidence.Dir()
	defer func() {
		// The report file is the last evidence of the run; when it cannot be
		// written the run cannot show what happened.
		if err != nil {
			report.Error = err.Error()
		}
		if writeErr := ports.Evidence.WriteReport(cli.ReportValue(report)); writeErr != nil {
			err = errors.Join(err, writeErr)
			report.Status = model.NotCompleted
		}
		err = errors.Join(err, ports.Evidence.Close())
	}()
	outcome, receipt, err := s.reconcileAndPublish(ctx, prepared, cfg, ports)
	if outcome.Status != "" {
		report.Run = cli.NewRunReport(ports.RunID, outcome)
	}
	report.Publication = receipt
	report.Status = cli.ApplyStatus(outcome, receipt, err)
	return report, err
}

// reconcileAndPublish is the seam of the Apply use case: it runs the Manager
// team over the captured repository and publishes a verified candidate. Its
// wiring belongs to the next increment.
func (s *executionService) reconcileAndPublish(context.Context, apply.CapturedSettingsResult, runConfig, applyPorts) (runport.Outcome, *apply.PublicationReceipt, error) {
	return runport.Outcome{}, nil, errors.New("the Apply use case is not wired yet")
}

// answer records the owner's answer for the Apply running on the configured
// state directory.
func answer(ctx context.Context, request cli.AnswerRequest) (cli.AnswerReport, error) {
	report := cli.AnswerReport{Status: model.NotCompleted}
	cfg, err := loadRunConfig(ctx, request.RunConfig)
	if err != nil {
		if errors.Is(err, errRunConfig) {
			report.Status = model.Failed
		}
		return report, err
	}
	path, err := writeAnswer(cfg.StateDirectory, request.Question, request.Text)
	if err != nil {
		return report, err
	}
	report.Status, report.Path, report.Bytes = model.Passed, path, len(request.Text)
	return report, nil
}

func capturedExecutionReport(prepared apply.CapturedSettingsResult, productBuild string) any {
	attempts := []any{}
	for _, attempt := range prepared.Attempts() {
		structural := attempt.Structural()
		compiled := structural.Model().Compilation()
		entry := map[string]any{
			"head":        structural.Model().Inputs().Head,
			"modelStatus": compiled.Status, "modelDiagnostics": compiled.Diagnostics,
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
func (s *executionService) ports(cfg runConfig, configPath string, prepared apply.CapturedSettingsResult) (applyPorts, error) {
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
	runID, err := newRunID(time.Now())
	if err != nil {
		return applyPorts{}, err
	}
	owner, err := newFileOwner(cfg.StateDirectory, runID, configPath, s.stderr)
	if err != nil {
		return applyPorts{}, err
	}
	evidence, err := newFileEvidence(cfg.StateDirectory, runID)
	if err != nil {
		return applyPorts{}, err
	}
	return applyPorts{RunID: runID, Agents: agents, Workspaces: workspaces, Checks: checker, Owner: owner, Evidence: evidence,
		Publisher: s.repository, Harvest: cfg.Harvest, Deadline: cfg.deadline(), Clock: time.Now}, nil
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

// Only this already validated captured literal is needed for Host composition;
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
