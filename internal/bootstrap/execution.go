package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/adapters/journal"
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
	// A private source-test seam. Production always supplies productionPorts;
	// an absent factory refuses rather than supplying success or fake ports.
	ports func(runConfig, *apply.ControlledRun, apply.CapturedSettingsResult) (apply.ExecutionPorts, error)
}

func newExecution(preparation *apply.PreparationService, repository *gitrepo.Repository, agents apply.AgentProfilePreparer, checks apply.CheckProfilePreparer) *executionService {
	s := &executionService{preparation: preparation, repository: repository, agents: agents, checks: checks}
	s.ports = s.productionPorts
	return s
}

func (s *executionService) Execute(ctx context.Context, request cli.ExecutionRequest) (report cli.ExecutionReport, err error) {
	report.Status = model.NotCompleted
	cfg, err := loadRunConfig(ctx, request.RunConfig)
	if err != nil {
		if errors.Is(err, errRunConfig) {
			report.Status = model.Failed
		}
		return report, err
	}
	if s == nil || s.preparation == nil || s.repository == nil || s.ports == nil {
		return report, errors.New("concrete execution composition is unavailable")
	}
	owner, err := runport.NewPayload("text/plain", "utf-8", []byte(cfg.OwnerRequest))
	if err != nil {
		report.Status = model.Failed
		return report, errors.New("ownerRequest cannot form the bound UTF-8 owner input")
	}
	prepared, err := s.preparation.CaptureSettings(ctx, apply.CaptureRequest{Selection: request.Selection}, model.Limits{}, s.agents, s.checks)
	report.Preparation = capturedExecutionReport(prepared, cfg.ProductBuild)
	if prepared.Status() != model.Passed || err != nil {
		report.Status = prepared.Status()
		return report, err
	}
	opened, err := apply.NewControlService(s.repository, journal.New()).Open(ctx, prepared,
		apply.RunBindingFacts{RepositoryIdentity: cfg.RepositoryIdentity, ProductBuild: cfg.ProductBuild, OwnerRequest: owner, Allocation: cfg.Allocation},
		apply.ControlOptions{StateDirectory: cfg.StateDirectory, RestoreLimits: cfg.RestoreLimits})
	report.Control = opened.Binding()
	if err != nil {
		return report, err
	}
	controlled, ok := opened.Controlled()
	if !ok {
		return report, errors.New("repository control did not return a live controlled run")
	}
	defer func() {
		if closeErr := controlled.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
			report.Status = model.NotCompleted
		}
	}()
	defer func() {
		report.OpenQuestions = controlled.Coordinator().OwnerQuestions()
		if report.Execution == nil {
			// Actual restored observations remain visible when Answer or port
			// construction refuses before Execute. No run is invented here.
			current := controlled.Coordinator()
			observed := apply.ExecutionResult{Run: current.Current(), Tasks: current.Tasks(), Checks: current.Checks()}
			for _, task := range observed.Tasks {
				observed.Issues = append(observed.Issues, current.Issues(task.Manager)...)
			}
			report.Execution = &observed
		}
	}()
	ports, err := s.ports(cfg, controlled, prepared)
	if err != nil {
		return report, err
	}
	if request.Question != "" {
		if controlled.Coordinator().Current().Basis != controlled.Basis().Ref() {
			return report, errors.New("answer requires the original captured basis and run configuration")
		}
		if err := controlled.Answer(ctx, request.Question, request.Answer, ports.Clock); err != nil {
			return report, err
		}
	}
	result, err := controlled.Execute(ctx, ports)
	report.Execution = &result
	report.Status = executionStatus(result.Run, err)
	return report, err
}

func executionStatus(observed runport.RunView, err error) model.Status {
	if observed.State != runport.Finished {
		return model.NotCompleted
	}
	switch observed.Outcome {
	case runport.Applied, runport.NoChange:
		if err == nil {
			return model.Passed
		}
	case runport.Failed:
		return model.Failed
	}
	return model.NotCompleted
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

func (s *executionService) productionPorts(cfg runConfig, controlled *apply.ControlledRun, prepared apply.CapturedSettingsResult) (apply.ExecutionPorts, error) {
	a, ok := prepared.Prepared()
	if !ok {
		return apply.ExecutionPorts{}, errors.New("complete captured settings are required")
	}
	for _, profile := range a.Settings().ProfileObservations() {
		if profile.Input().Namespace() == "agentProfiles" {
			if _, err := sessionProfile(profile, false, cfg); err != nil {
				return apply.ExecutionPorts{}, err
			}
		}
	}
	selectedChecks, complete := a.Settings().Checks()
	if !complete {
		return apply.ExecutionPorts{}, errors.New("configured check selections are incomplete")
	}
	for _, selected := range selectedChecks {
		if _, err := executionCheck(selected, cfg); err != nil {
			return apply.ExecutionPorts{}, err
		}
	}
	engine, err := container.NewEngine(cfg.EngineExecutable)
	if err != nil {
		return apply.ExecutionPorts{}, err
	}
	codexCredentials, claudeCredentials, err := newCredentialResolvers(cfg.Credentials)
	if err != nil {
		return apply.ExecutionPorts{}, err
	}
	codex, err := codexappserver.NewSessions(engine, codexCredentials)
	if err != nil {
		return apply.ExecutionPorts{}, err
	}
	claude, err := claudecode.NewSessions(engine, claudeCredentials)
	if err != nil {
		return apply.ExecutionPorts{}, err
	}
	helper := cfg.WorkspaceHelper
	helper.Workdir = "/work"
	workspaces, err := gitrepo.NewWorkspaces(engine, helper)
	if err != nil {
		return apply.ExecutionPorts{}, err
	}
	checker, err := checks.NewRunner(engine, workspaces)
	if err != nil {
		return apply.ExecutionPorts{}, err
	}
	return apply.ExecutionPorts{Sessions: NewSessionsRouter(codex, claude), Workspaces: workspaces, Checks: checker, Publisher: s.repository,
		Clock: time.Now, Limits: cfg.EngineLimits,
		Profile: func(request runport.ProfileRequest) (runport.SessionProfile, error) {
			if request.Role != "manager" && request.Role != "reviewer" {
				return runport.SessionProfile{}, errors.New("unsupported session role")
			}
			profile, err := controlled.RoleProfile(request.Manager, request.Role)
			if err != nil {
				return runport.SessionProfile{}, err
			}
			return sessionProfile(profile, request.ReadOnly, cfg)
		},
		CheckRequest: func(selected apply.CheckSelection) (runport.CheckRequest, error) {
			return executionCheck(selected, cfg)
		},
	}, nil
}

func sessionProfile(prepared apply.PreparedProfile, readOnly bool, cfg runConfig) (runport.SessionProfile, error) {
	if !profileEngine(prepared, cfg.EngineExecutable) {
		return runport.SessionProfile{}, errors.New("prepared Agent engine differs from the explicit run engine")
	}
	var profile runport.Payload
	var reference, provider string
	var wire int64
	var err error
	switch prepared.Input().Section() {
	case "codexAppServer":
		provider = "codex-app-server"
		profile, reference, wire, err = codexappserver.SessionProfileFromPrepared(prepared)
	case "claudeCode":
		provider = "claude-code"
		profile, reference, wire, err = claudecode.SessionProfileFromPrepared(prepared, readOnly)
	default:
		err = errors.New("selected Agent profile has no supported execution adapter")
	}
	if err != nil {
		return runport.SessionProfile{}, err
	}
	credential, found := cfg.Credentials[reference]
	image := prepared.Observation().AgentImage
	if !found || credential.Provider != provider || image == nil {
		return runport.SessionProfile{}, errors.New("prepared Agent has no exact explicit credential mapping or image observation")
	}
	return runport.SessionProfile{ImageID: image.ImageID, CredentialRef: reference, Profile: profile, MaxWireBytes: wire}, nil
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
	return profile.Request(selected.Declaration(), snapshot.Tree{}) // Core fills the admitted tree.
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
