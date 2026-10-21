package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/apply"
)

// runController composes the cockpit's start and stop of an Apply: the same
// execution function as markitect apply, with the cockpit's repository, model
// and dependency selection, the run configuration and the Runtime profile, in
// the cockpit's own process. It refuses to start while another Apply or a
// proposal publication holds the run configuration's state directory.
func (c cockpitService) runController(ctx context.Context, base apply.InputSelection, runConfigPath, runtimeProfile, stateDirectory, build string, fence *fencedStates) *cockpit.RunController {
	return cockpit.NewRunController(ctx, cockpit.RunControlConfig{
		StateDir: stateDirectory,
		Facts: func(ctx context.Context) (cockpit.RunFacts, error) {
			return c.runFacts(ctx, base, runConfigPath, runtimeProfile)
		},
		Head: func(ctx context.Context) (string, error) {
			return c.repository.ResolveRevision(ctx, base.Repository, "HEAD")
		},
		// Execute refuses a state directory that a foreign run holds within
		// its lock wait; after it, a run named in current-run is this one's.
		AttributeAfter: startLockWait + time.Second,
		Held: func() (string, error) {
			if fence.has(stateDirectory) {
				return "the cockpit is publishing an accepted proposal; start after it is written", nil
			}
			held, err := stateHeld(stateDirectory)
			if held {
				return "an Apply holds the state directory " + stateDirectory, nil
			}
			return "", err
		},
		Execute: func(ctx context.Context) cockpit.RunResult {
			selection, err := withRuntimeProfile(base, runtimeProfile)
			if err != nil {
				return cockpit.RunResult{Outcome: cockpit.RunRefused, Error: err.Error()}
			}
			report, err := c.execute(ctx, cli.ApplyRequest{Selection: selection, RunConfig: runConfigPath, Build: build})
			return runResult(report, err)
		},
	})
}

// runResult is what the owner sees of a finished execution: the run outcome as
// markitect apply reports it, the evidence directory and the error. An
// execution that created no evidence refused before the run started.
func runResult(report cli.ApplyReport, err error) cockpit.RunResult {
	result := cockpit.RunResult{Outcome: cli.RunOutcome(report.Status), Evidence: report.EvidenceDirectory, Error: report.Error}
	if err != nil {
		result.Error = err.Error()
	}
	if result.Evidence == "" {
		result.Outcome = cockpit.RunRefused
		if result.Error == "" {
			result.Error = "the run did not start: " + string(report.Status)
		}
	}
	return result
}

// runFacts reads what an Apply started now would run with: the committed
// execution budget, the run configuration's deadline and the Execute Agent
// profiles of the Managers, Root's first, each with the model its Runtime
// profile states. It captures the committed
// source with the Runtime profile as the Apply does and decodes with the
// product's own readers.
func (c cockpitService) runFacts(ctx context.Context, base apply.InputSelection, runConfigPath, runtimeProfile string) (cockpit.RunFacts, error) {
	cfg, a, err := c.captureWithRuntime(ctx, base, runConfigPath, runtimeProfile)
	if err != nil {
		return cockpit.RunFacts{}, err
	}
	budget, ok := a.Settings().Budget()
	if !ok {
		return cockpit.RunFacts{}, errors.New("the execution budget is required")
	}
	settings, ok := a.Settings().Settings()
	if !ok {
		return cockpit.RunFacts{}, errors.New("resolved settings are required")
	}
	managers := settings.Managers() // Root first
	if len(managers) == 0 {
		return cockpit.RunFacts{}, errors.New("Root has no resolved settings")
	}
	facts := cockpit.RunFacts{
		Head: a.Structural().Model().Inputs().Head,
		Budget: cockpit.RunBudget{
			ExecutionTime: durationText(budget.ExecutionTime.Value), TurnTimeout: durationText(budget.TurnTimeout.Value),
			AgentTurns: budget.AgentTurns.Value, Passes: budget.Passes.Value,
		},
		Deadline: cfg.Deadline,
	}
	index := map[string]int{}
	for _, manager := range managers {
		name := manager.Execute.Profile.Value.Name
		i, seen := index[name]
		if !seen {
			i = len(facts.Profiles)
			index[name] = i
			facts.Profiles = append(facts.Profiles, cockpit.RunProfile{Name: name})
		}
		facts.Profiles[i].Managers = append(facts.Profiles[i].Managers, manager.Package)
	}
	for i, profile := range facts.Profiles {
		model, err := executeProfileModel(a.Settings().ProfileObservations(), profile.Name)
		if err != nil {
			return cockpit.RunFacts{}, err
		}
		facts.Profiles[i].Model = model
	}
	return facts, nil
}

// executeProfileModel is the model the Runtime profile states for an Agent
// profile, decoded as the adapters do for an Apply; empty when the profile
// states none.
func executeProfileModel(observed []apply.PreparedProfile, name string) (string, error) {
	for _, profile := range observed {
		input := profile.Input()
		if input.Namespace() != "agentProfiles" || input.Name() != name {
			continue
		}
		switch input.Section() {
		case "claudeCode":
			p, err := claudecode.ProfileFromPrepared(profile)
			if err != nil {
				return "", fmt.Errorf("agent profile %q: %w", name, err)
			}
			return p.Session.Model, nil
		case "codexAppServer":
			p, err := codexappserver.ProfileFromPrepared(profile)
			if err != nil {
				return "", fmt.Errorf("agent profile %q: %w", name, err)
			}
			return p.Session.Model, nil
		}
	}
	return "", nil
}

// durationText writes a duration in the notation of the execution settings,
// such as 25m or 1h30m.
func durationText(d time.Duration) string {
	text := d.String()
	if strings.HasSuffix(text, "m0s") {
		text = strings.TrimSuffix(text, "0s")
	}
	if strings.HasSuffix(text, "h0m") {
		text = strings.TrimSuffix(text, "0m")
	}
	return text
}
