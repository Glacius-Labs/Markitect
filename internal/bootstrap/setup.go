package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const (
	setupRuntimeFile = "runtime.yaml"
	setupConfigFile  = "run.json"
	setupStateDir    = "state"
)

// setupEngine is everything setup asks of the container engine: image
// resolution and never-started image reads for the metadata preparation, the
// engine's version. *container.Engine implements it; tests supply controlled
// fakes.
type setupEngine interface {
	claudecode.ImageReader
	checks.ImageReader
	ServerVersion(context.Context) (string, error)
}

// containerSetupEngine binds the explicit engine executable.
func containerSetupEngine(path string) (setupEngine, error) { return container.NewEngine(path) }

// setupRefusal marks a known refusal of the invocation, as opposed to an
// incomplete observation.
type setupRefusal struct{ error }

func (r setupRefusal) Unwrap() error { return r.error }

func refuse(format string, args ...any) error { return setupRefusal{fmt.Errorf(format, args...)} }

// setupService prepares a Runtime directory. It reads the project's profile
// selections from a structural preview and observes engine and image through
// the engine adapter; it starts no container and reads no credential.
type setupService struct {
	preparation *apply.PreparationService
	newEngine   func(path string) (setupEngine, error)
}

func newSetup(preparation *apply.PreparationService, newEngine func(path string) (setupEngine, error)) *setupService {
	return &setupService{preparation: preparation, newEngine: newEngine}
}

// Setup observes the engine and image, composes runtime.yaml and run.json, checks
// them with the decoders and preparation that Apply uses, and writes them.
func (s *setupService) Setup(ctx context.Context, request cli.SetupRequest) (report cli.SetupReport, err error) {
	report.Status = model.NotCompleted
	defer func() {
		if err == nil {
			return
		}
		if errors.As(err, new(setupRefusal)) {
			report.Status = model.Failed
		}
	}()
	if s == nil || s.preparation == nil || s.newEngine == nil {
		return report, errors.New("concrete setup composition is unavailable")
	}
	repository := request.Selection.Repository
	if err := checkOutsideRepository(request.RuntimeDir, repository, "the Runtime directory"); err != nil {
		return report, err
	}
	stateDirectory := filepath.Join(request.RuntimeDir, setupStateDir) // inside the Runtime directory, so outside the repository as well
	if err := checkCredentialFile(request.Credential, repository); err != nil {
		return report, err
	}

	selected, modelID, err := s.selectedProfiles(ctx, request.Selection)
	if err != nil {
		return report, err
	}
	report.AgentProfiles, report.CheckProfiles = selected.agents, selected.checks
	for _, name := range selected.checks {
		if request.Checks[name] == "" {
			return report, refuse("the project selects the check profile %q; pass --check %s=EXECUTABLE", name, name)
		}
	}
	for name := range request.Checks {
		if !slices.Contains(selected.checks, name) {
			return report, refuse("--check names the profile %q, which the project does not select", name)
		}
	}
	store := request.Store
	if store == "" {
		store = modelID
	}
	if !stateName.MatchString(store) {
		return report, refuse("the store label %q is not a plain name; pass --store", store)
	}

	engine, err := s.newEngine(request.Engine)
	if err != nil {
		return report, err
	}
	engineVersion, err := engine.ServerVersion(ctx)
	if err != nil {
		return report, err
	}
	if err := container.RequireSubpathVersion(engineVersion); err != nil {
		return report, err
	}
	observed, err := claudecode.ObserveRuntime(ctx, engine, request.Image)
	if err != nil {
		return report, err
	}
	report.Bound = &cli.SetupBound{ImageID: observed.ImageID, AgentdVersion: observed.AgentdVersion, CLIVersion: observed.CLIVersion, EngineVersion: engineVersion}

	runtimeBytes, err := composeRuntimeProfile(request, selected, observed, store)
	if err != nil {
		return report, refuse("%v", err)
	}
	configBytes, err := encodeRunConfig(setupRunConfig(request, observed.ImageID, store, stateDirectory))
	if err != nil {
		return report, refuse("%v", err)
	}
	if err := s.validate(ctx, request, engine, runtimeBytes, configBytes); err != nil {
		return report, err
	}

	report.Files, err = writeSetupFiles(request.RuntimeDir, stateDirectory, []setupFile{{setupRuntimeFile, runtimeBytes}, {setupConfigFile, configBytes}})
	if err != nil {
		return report, err
	}
	report.Status = model.Passed
	return report, nil
}

// selection is the profile names that the project's execution settings select.
type selection struct{ agents, checks []string }

// selectedProfiles reads the working draft's settings without any Runtime file:
// the effective execute and verify profile of the defaults and of every
// per-Package override, and the profile of every declared check. It returns the
// local model id as well.
func (s *setupService) selectedProfiles(ctx context.Context, selected apply.InputSelection) (selection, string, error) {
	result, err := s.preparation.Preview(ctx, apply.PreviewRequest{Selection: selected}, model.Limits{})
	attempt, ok := result.Prepared()
	if !ok || err != nil {
		reason := "run `markitect check` to see why"
		if err != nil {
			reason = err.Error()
		}
		if result.Status() == model.Failed {
			return selection{}, "", refuse("the working draft does not pass structural preparation: %s", reason)
		}
		return selection{}, "", fmt.Errorf("the working draft could not be completely prepared: %s", reason)
	}
	defaults := attempt.Execution().PrepareDefaults(apply.ProductStrategyDefaults())
	declarations, ok := defaults.Declarations()
	if !ok {
		return selection{}, "", refuse("the execution declarations and product defaults do not pass; run `markitect check`")
	}
	var names selection
	add := func(list *[]string, name string) {
		if name != "" && !slices.Contains(*list, name) {
			*list = append(*list, name)
		}
	}
	add(&names.agents, declarations.Defaults.Execute.Profile.Value)
	add(&names.agents, declarations.Defaults.Verify.Profile.Value)
	for _, helper := range declarations.Defaults.Team.Helpers {
		if helper.Profile != nil {
			add(&names.agents, helper.Profile.Value)
		}
	}
	for _, override := range declarations.Overrides {
		if profile := override.Execute.Profile; profile != nil {
			add(&names.agents, profile.Value)
		}
		if profile := override.Verify.Profile; profile != nil {
			add(&names.agents, profile.Value)
		}
	}
	if document, ok := attempt.Execution().Document(); ok && document.Checks != nil {
		for _, check := range document.Checks.Value {
			add(&names.checks, check.Profile.Value)
		}
	}
	slices.Sort(names.agents)
	slices.Sort(names.checks)
	id := ""
	for _, m := range attempt.Model().Checked().Models {
		if m.Local {
			id = m.ID
		}
	}
	return names, id, nil
}

// validate runs the composed files through the decoders and preparation that
// Apply uses: the run configuration through its decoder, and the Runtime profile
// through the settings and metadata preparation of `check --metadata
// --runtime-profile`, staged in a private directory so no file is written yet.
// It also projects every prepared profile as Apply does.
func (s *setupService) validate(ctx context.Context, request cli.SetupRequest, engine setupEngine, runtimeBytes, configBytes []byte) error {
	cfg, err := decodeRunConfig(configBytes)
	if err != nil {
		return refuse("the composed run configuration is invalid: %v", err)
	}
	stage, err := os.MkdirTemp("", "markitect-setup-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if resolved, err := filepath.EvalSymlinks(stage); err == nil {
		stage = resolved
	}
	if err := os.WriteFile(filepath.Join(stage, setupRuntimeFile), runtimeBytes, 0o600); err != nil {
		return err
	}
	path, err := snapshot.NewPath(setupRuntimeFile)
	if err != nil {
		return err
	}
	selection := request.Selection
	selection.RuntimeProfiles = []apply.SourceLocation{{Boundary: stage, Path: path}}
	agents := agentMetadataRouter{claude: claudecode.NewMetadataPreparer(func(claudecode.EngineConfig) (claudecode.ImageReader, error) { return engine, nil })}
	checker := checks.NewMetadataPreparer(func(checks.EngineConfig) (checks.ImageReader, error) { return engine, nil })
	result, err := s.preparation.PreviewSettings(ctx, apply.PreviewRequest{Selection: selection}, model.Limits{}, agents, checker)
	attempt, ok := result.Prepared()
	if !ok || err != nil {
		message := "the composed Runtime profile does not pass settings and metadata preparation"
		if summary := preparationDiagnostics(result); summary != "" {
			message += ": " + summary
		}
		if err != nil {
			message += ": " + err.Error()
		}
		if result.Status() == model.Failed {
			return refuse("%s", message)
		}
		return errors.New(message)
	}
	settings := attempt.Settings()
	selectedChecks, complete := settings.Checks()
	if !complete {
		return refuse("the composed Runtime profile does not resolve every check profile")
	}
	for _, selected := range selectedChecks {
		if _, err := executionCheck(selected, cfg); err != nil {
			return refuse("check %q: %v", selected.Declaration().ID.Value, err)
		}
	}
	for _, profile := range settings.ProfileObservations() {
		input := profile.Input()
		if input.Namespace() != "agentProfiles" {
			continue
		}
		p, err := claudecode.ProfileFromPrepared(profile)
		if err == nil {
			err = credentialMapped(cfg, p.CredentialRef, "claude-code")
		}
		if err == nil && (!profileEngine(profile, cfg.EngineExecutable) || p.Session.Model != request.AgentModel) {
			err = errors.New("the prepared profile differs from the requested engine or model")
		}
		if err != nil {
			return refuse("agent profile %q: %v", input.Name(), err)
		}
	}
	return nil
}

// preparationDiagnostics lists the settings diagnostics of the last attempt.
func preparationDiagnostics(result apply.PreviewSettingsResult) string {
	attempts := result.Attempts()
	if len(attempts) == 0 {
		return ""
	}
	var parts []string
	for _, d := range attempts[len(attempts)-1].Settings().Diagnostics() {
		parts = append(parts, d.Code+": "+d.Message)
		if len(parts) == 5 {
			break
		}
	}
	return strings.Join(parts, "; ")
}

// setupRunConfig composes the run configuration of the setup.
func setupRunConfig(request cli.SetupRequest, imageID, store, stateDirectory string) runConfigDocument {
	ownerRequest := request.OwnerRequest
	if ownerRequest == "" {
		ownerRequest = defaultOwnerRequest
	}
	return runConfigDocument{
		FormatVersion: 2, RepositoryIdentity: store, ProductBuild: request.ProductBuild, OwnerRequest: ownerRequest,
		OwnerDecisions: request.OwnerDecisions, StateDirectory: stateDirectory, Deadline: request.Deadline,
		EngineExecutable: request.Engine,
		WorkspaceHelper: workspaceHelperDocument{
			Name: "markitect-ws", Image: imageID, Labels: map[string]string{"markitect.store": store}, User: "1000:1000",
			CPUs: 1, MemoryBytes: 512 << 20, Pids: 128, TmpBytes: 128 << 20, Network: "none",
		},
		Harvest:     harvestLimits{MaxEntries: 50000, MaxBytes: 256 << 20},
		Credentials: map[string]credentialSource{setupCredentialRef: {Provider: "claude-code", Mode: "dedicated-runtime-token", Path: request.Credential}},
	}
}
