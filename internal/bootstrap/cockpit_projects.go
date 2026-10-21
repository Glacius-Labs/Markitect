package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/apply"
)

// projectsUnsetReason is what a project without a runtime says about runs.
const projectsUnsetReason = "This project is not set up to run: it has no run configuration and Runtime profile yet. Set them up from the project list."

// hostConfig composes the cockpit's host: the per-user project list under the
// operating system's user configuration directory, the opener that builds a
// project's Server as the flags would, and the function behind markitect
// setup for the guided setup.
func (c cockpitService) hostConfig(build string) cockpit.HostConfig {
	cfg := cockpit.HostConfig{
		Open: c.entryOpener(build),
		Setup: func(ctx context.Context, in cockpit.SetupInputs) cockpit.SetupResult {
			return c.guidedSetup(ctx, in, build)
		},
		Defaults: cockpit.SetupDefaults{Engine: cli.DefaultSetupEngine, Effort: cli.DefaultSetupEffort},
	}
	dir, err := c.configDir()
	if err != nil {
		cfg.Reason = fmt.Sprintf("The per-user project list needs the user configuration directory: %v", err)
		return cfg
	}
	cfg.Store = cockpit.NewProjectStore(filepath.Join(dir, "markitect", "cockpit", "projects.json"))
	cfg.RuntimeBase = filepath.Join(dir, "markitect", "runtimes")
	return cfg
}

func (c cockpitService) configDir() (string, error) {
	if c.userConfigDir != nil {
		return c.userConfigDir()
	}
	return os.UserConfigDir()
}

// entryOpener builds a listed project's Server as the cockpit's flags would
// for the same repository, model and dependency models, with the run
// configuration and Runtime profile setup wrote into its runtime directory.
// A project without a runtime is observed and proposes, and starts no run and
// no model session.
func (c cockpitService) entryOpener(build string) cockpit.ProjectOpener {
	return func(ctx context.Context, entry cockpit.ProjectEntry) (cockpit.OpenedProject, error) {
		selection, err := cli.RepositorySelection(entry.Repository, entry.Model, entry.Dependencies)
		if err != nil {
			return cockpit.OpenedProject{}, fmt.Errorf("the project's model selection is invalid: %w", err)
		}
		spec := projectSpec{selection: selection, build: build, runConfig: entry.RunConfigPath(), runtimeProfile: entry.RuntimeProfilePath()}
		if entry.Runtime == "" {
			spec.runsReason = projectsUnsetReason
		}
		return c.open(ctx, spec)
	}
}

// directProject names the project the flags opened, for the page: its
// repository, its model selection and, when the run configuration and the
// Runtime profile are the pair setup writes into one directory, that directory
// as its runtime. Explicit state directories are not part of an entry.
func directProject(request cli.CockpitRequest) cockpit.ProjectEntry {
	selection := request.Selection
	entry := cockpit.ProjectEntry{Name: filepath.Base(selection.Repository), Repository: selection.Repository, Model: manifestText(selection.Local), Dependencies: []string{}}
	if selection.Repository == "" {
		entry.Name = "State directories"
	}
	for _, dependency := range selection.Dependencies {
		entry.Dependencies = append(entry.Dependencies, manifestText(dependency))
	}
	dir := filepath.Dir(request.RunConfig)
	if request.RunConfig != "" && filepath.Base(request.RunConfig) == "run.json" && request.RuntimeProfile == filepath.Join(dir, "runtime.yaml") {
		entry.Runtime = dir
	}
	return entry
}

// manifestText is the spelling of a model selection that the flags use: the
// path relative to the repository, or absolute outside it.
func manifestText(m apply.ModelSelection) string {
	if m.Boundary == "" || m.Boundary == "." {
		return m.ManifestPath.String()
	}
	return filepath.Join(m.Boundary, m.ManifestPath.String())
}

// guidedSetup is the setup form's call of the function behind markitect
// setup, with the project's repository, model and dependency models and the
// product build the cockpit was started with. It adds no owner request and no
// decisions.
func (c cockpitService) guidedSetup(ctx context.Context, in cockpit.SetupInputs, build string) cockpit.SetupResult {
	selection, err := cli.RepositorySelection(in.Project.Repository, in.Project.Model, in.Project.Dependencies)
	if err != nil {
		return cockpit.SetupResult{Status: "not_completed", Error: err.Error()}
	}
	setup := c.setup
	if setup == nil {
		setup = newSetup(c.preparation, containerSetupEngine).Setup
	}
	report, err := setup(ctx, cli.SetupRequest{
		Selection: selection, RuntimeDir: in.RuntimeDir, Image: in.Image, Credential: in.Credential, Checks: in.Checks,
		AgentModel: in.AgentModel, Effort: in.Effort, Engine: in.Engine, Store: in.Store, Deadline: in.Deadline,
		ProductBuild: build,
	})
	result := cockpit.SetupResult{Status: string(report.Status), AgentProfiles: report.AgentProfiles, CheckProfiles: report.CheckProfiles, Error: report.Error}
	if report.Bound != nil {
		result.Bound = &cockpit.SetupBound{ImageID: report.Bound.ImageID, AgentdVersion: report.Bound.AgentdVersion, CLIVersion: report.Bound.CLIVersion, EngineVersion: report.Bound.EngineVersion}
	}
	for _, f := range report.Files {
		result.Files = append(result.Files, cockpit.SetupFile{Path: f.Path, SHA256: f.SHA256, Action: f.Action})
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}
