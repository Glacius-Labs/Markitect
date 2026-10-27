package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/runner"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// runnerGrace covers the runner's clone, hand-back and graceful stop beyond
// Apply's own deadline, which the runner's Apply enforces itself.
const runnerGrace = 10 * time.Minute

// runnerStopTimeout bounds the runner's graceful stop after cancellation:
// Apply closes its containers and keeps its evidence within its cleanup and
// persistence bounds of at most 60 s each.
const runnerStopTimeout = 4 * time.Minute

type runnerLauncher func(context.Context, runner.LaunchRequest) (runner.Launch, error)

// runnerApply starts Apply in the Linux runner image (DEC-077). The owner's
// files keep their native meaning: the run configuration's engine starts the
// runner, its state directory receives the launch directory, and its dedicated
// logins are mounted read-only. The runner reads translated copies that name
// its own layout and keeps Apply state in a named volume per project.
func runnerApply(ctx context.Context, request cli.ApplyRequest, cfg runConfig, launch runnerLauncher) (cli.ApplyReport, error) {
	out := cli.ApplyReport{FormatVersion: 1, Operation: "apply", Status: model.NotCompleted, StateDirectory: cfg.StateDirectory}
	selection := request.Selection
	manifest, dependencies, err := runnerManifests(selection)
	if err != nil {
		return out, err
	}
	if len(selection.RuntimeProfiles) != 1 {
		return out, fmt.Errorf("the runner needs exactly one runtime profile")
	}
	location := selection.RuntimeProfiles[0]
	data, err := readRuntimeCredential(ctx, filepath.Join(location.Boundary, filepath.FromSlash(location.Path.String())))
	if err != nil {
		return out, fmt.Errorf("runtime profile could not be completely observed")
	}
	profile, err := runner.TranslateRuntimeProfile(data, cfg.EngineExecutable)
	if err != nil {
		return out, err
	}
	invocation, logins, err := runnerRunConfig(cfg)
	if err != nil {
		return out, err
	}
	volume := "markitect-state-" + cfg.RepositoryIdentity
	ctx, cancel := context.WithDeadline(ctx, cfg.deadline().Add(runnerGrace))
	defer cancel()
	launched, err := launch(ctx, runner.LaunchRequest{
		Engine: cfg.EngineExecutable, Image: request.Runner, Repository: selection.Repository,
		Model: manifest, Dependencies: dependencies, RunConfig: invocation, RuntimeProfile: profile,
		Credentials: logins, OutputRoot: filepath.Join(cfg.StateDirectory, cfg.RepositoryIdentity, "runner"),
		StateVolume: volume, ResultBranch: request.ResultBranch, StopTimeout: runnerStopTimeout, Log: request.Log,
	})
	result := launched.Result
	out.ID, out.EvidencePath = result.ApplyID, launched.ReportPath()
	out.ProviderRetries = innerReport(launched.Report).ProviderRetries
	out.Runner = &cli.RunnerReport{Image: request.Runner, Launch: launched.ID, Directory: launched.Directory, StateVolume: volume, Base: launched.Base, ApplyStatus: model.Status(result.ApplyStatus), NoChange: result.NoChange, Branch: launched.Branch, Commit: launched.Commit, Paths: result.Paths, Report: launched.Report}
	switch {
	case err != nil:
	case result.ApplyStatus == runner.StatusPassed:
		out.Status = model.Passed
	case result.ApplyStatus == runner.StatusFailed:
		out.Status = model.Failed
		err = withCause("Apply in the runner failed", innerReport(launched.Report).Error)
	default:
		err = withCause("Apply in the runner did not complete", innerReport(launched.Report).Error)
	}
	if err != nil {
		out.Error = err.Error()
	}
	if launched.Directory != "" {
		data, marshalErr := json.MarshalIndent(cli.ReportValue(out), "", "  ")
		if marshalErr == nil {
			marshalErr = writeStateFile(filepath.Join(launched.Directory, "launch.json"), append(data, '\n'), false)
		}
		if marshalErr != nil {
			out.Status = model.NotCompleted
			err = errors.Join(err, fmt.Errorf("runner launch record: %w", marshalErr))
		}
	}
	return out, err
}

// runnerManifests returns the repository-relative manifests the runner selects
// in its clone. External manifests would need mounts the runner does not have.
func runnerManifests(s project.InputSelection) (string, []string, error) {
	relative := func(m project.ModelSelection) (string, error) {
		p := m.ManifestPath.String()
		if m.Boundary != "." || !runner.RepositoryRelative(p, false) {
			return "", fmt.Errorf("the runner takes repository-relative --model and --dependency paths")
		}
		return p, nil
	}
	manifest, err := relative(s.Local)
	if err != nil {
		return "", nil, err
	}
	var dependencies []string
	for _, dependency := range s.Dependencies {
		p, err := relative(dependency)
		if err != nil {
			return "", nil, err
		}
		dependencies = append(dependencies, p)
	}
	return manifest, dependencies, nil
}

// runnerRunConfig returns the run configuration the runner reads and the host
// login files in mount order. Only the host paths change: the state directory,
// the engine and each login name the runner's layout. Limits, deadlines, the
// project identity and the build assertion stay the owner's.
func runnerRunConfig(cfg runConfig) ([]byte, []string, error) {
	translated := cfg
	translated.StateDirectory, translated.EngineExecutable = runner.StateDir, runner.EngineBinary
	refs := make([]string, 0, len(cfg.Credentials))
	for ref := range cfg.Credentials {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	translated.Credentials = make(map[string]credentialSource, len(refs))
	logins := make([]string, 0, len(refs))
	for i, ref := range refs {
		source := cfg.Credentials[ref]
		logins = append(logins, source.Path)
		source.Path = runner.CredentialPath(i + 1)
		translated.Credentials[ref] = source
	}
	data, err := json.MarshalIndent(translated, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return append(data, '\n'), logins, nil
}

func withCause(message, cause string) error {
	if cause == "" {
		return errors.New(message)
	}
	return fmt.Errorf("%s: %s", message, cause)
}

// innerReport reads the fields of the runner's Apply report that the host
// report repeats. An absent or unreadable report yields zero values.
func innerReport(data json.RawMessage) (report struct {
	Error           string `json:"error"`
	ProviderRetries int    `json:"providerRetries"`
}) {
	if len(data) != 0 {
		_ = json.Unmarshal(data, &report)
	}
	return report
}
