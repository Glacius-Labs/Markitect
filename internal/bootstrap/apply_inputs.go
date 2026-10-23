package bootstrap

import (
	"context"
	"fmt"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
)

// Runtime instructions close through a fresh complete acquisition, never an
// adapter file read or an extension of an already captured Project value.
func captureApplyInputs(ctx context.Context, preparation *project.PreparationService, source project.Source, selection project.InputSelection, agents runtimeconfig.AgentProfilePreparer, checks runtimeconfig.CheckProfilePreparer) (project.Prepared, runtimeconfig.Result, model.Status, error) {
	var original project.CapturedPreparationAttempt
	var metadata runtimeconfig.Result
	for cycle := 0; cycle < 2; cycle++ {
		captured, err := preparation.Capture(ctx, project.CaptureRequest{Selection: selection}, model.Limits{})
		if err != nil {
			return project.Prepared{}, metadata, captured.Status(), err
		}
		attempts := captured.Attempts()
		if len(attempts) == 0 {
			return project.Prepared{}, metadata, model.NotCompleted, fmt.Errorf("missing complete acquisition")
		}
		attempt := attempts[len(attempts)-1]
		if cycle == 1 {
			confirmed, err := source.Capture(ctx, original.Model().Request())
			if err != nil {
				return project.Prepared{}, metadata, model.NotCompleted, err
			}
			if confirmed.Guard != original.Model().Inputs().Guard {
				return project.Prepared{}, metadata, model.NotCompleted, fmt.Errorf("original Project inputs changed while closing Runtime instructions")
			}
		}
		// An uncaptured repository instruction can initially be unaccounted.
		// All other known structural failures remain failures before metadata.
		if attempt.Model().Checked().Status != model.Passed || attempt.Registry().Status() != model.Passed || attempt.Execution().Status() != model.Passed || attempt.Contexts().Status() != model.Passed || attempt.Files().Status() == model.NotCompleted {
			return project.Prepared{}, metadata, captured.Status(), fmt.Errorf("Apply structural preflight did not pass")
		}
		metadata, err = runtimeconfig.Prepare(ctx, attempt.Model().Inputs().Raw, agents, checks)
		if err != nil {
			return project.Prepared{}, metadata, model.NotCompleted, err
		}
		if len(metadata.RequiredInstructions) > 0 && metadata.Status != model.Failed && cycle == 0 {
			original = attempt
			selection = attempt.Model().Request().Selection
			added := false
			for _, required := range metadata.RequiredInstructions {
				if !slices.Contains(selection.Instructions, required) {
					selection.Instructions = append(selection.Instructions, required)
					added = true
				}
			}
			if !added {
				return project.Prepared{}, metadata, model.NotCompleted, fmt.Errorf("Runtime instruction acquisition did not converge")
			}
			continue
		}
		if metadata.Status != model.Passed || len(metadata.RequiredInstructions) != 0 {
			return project.Prepared{}, metadata, metadata.Status, fmt.Errorf("complete captured native and check profile metadata required")
		}
		if captured.Status() != model.Passed {
			return project.Prepared{}, metadata, captured.Status(), fmt.Errorf("Apply file accounting preflight did not pass")
		}
		prepared, err := project.Prepare(attempt)
		return prepared, metadata, captured.Status(), err
	}
	return project.Prepared{}, metadata, model.NotCompleted, fmt.Errorf("Runtime instruction acquisition did not converge")
}
