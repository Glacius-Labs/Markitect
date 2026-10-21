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
// execution budget, the run configuration's deadline and Root's Execute Agent
// profile with the model of its Runtime profile. It captures the committed
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
	root := managers[0]
	facts := cockpit.RunFacts{
		Head: a.Structural().Model().Inputs().Head,
		Budget: cockpit.RunBudget{
			ExecutionTime: durationText(budget.ExecutionTime.Value), TurnTimeout: durationText(budget.TurnTimeout.Value),
			AgentTurns: budget.AgentTurns.Value, Passes: budget.Passes.Value,
		},
		Deadline: cfg.Deadline,
		Profile:  root.Execute.Profile.Value.Name,
	}
	for _, profile := range a.Settings().ProfileObservations() {
		input := profile.Input()
		if input.Namespace() != "agentProfiles" || input.Name() != facts.Profile {
			continue
		}
		switch input.Section() {
		case "claudeCode":
			p, err := claudecode.ProfileFromPrepared(profile)
			if err != nil {
				return cockpit.RunFacts{}, fmt.Errorf("agent profile %q: %w", facts.Profile, err)
			}
			facts.Model = p.Session.Model
		case "codexAppServer":
			p, err := codexappserver.ProfileFromPrepared(profile)
			if err != nil {
				return cockpit.RunFacts{}, fmt.Errorf("agent profile %q: %w", facts.Profile, err)
			}
			facts.Model = p.Session.Model
		}
	}
	return facts, nil
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
