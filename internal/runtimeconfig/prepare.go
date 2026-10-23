// Package runtimeconfig prepares closed static native/check profile metadata.
// It does not resolve model roles, strategies, mandates or execution budgets.
package runtimeconfig

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

type Result struct {
	Status               model.Status
	Profiles             []PreparedProfile
	Diagnostics          []RuntimeProfileDiagnostic
	RequiredInstructions []project.SourceLocation
}

// Prepare consumes acquired bytes; adapters cannot augment them with disk reads
// of project instructions. RequiredInstructions explicitly names missing inputs.
func Prepare(ctx context.Context, raw project.RawInputs, agents AgentProfilePreparer, checks CheckProfilePreparer) (Result, error) {
	e := decodeRuntime(raw)
	r := Result{Status: e.status, Diagnostics: e.diagnostics}
	if e.status != model.Passed {
		return r, nil
	}
	var errs error
	for _, input := range e.profiles {
		p, err := observeRuntimeProfile(ctx, input, agents, checks)
		errs = errors.Join(errs, err)
		r.Profiles = append(r.Profiles, p)
		r.Diagnostics = append(r.Diagnostics, p.observation.Diagnostics...)
		r.RequiredInstructions = append(r.RequiredInstructions, p.observation.RequiredInstructions...)
		if p.observation.Status == model.Failed || r.Status != model.Failed && p.observation.Status != model.Passed {
			r.Status = p.observation.Status
		}
	}
	return r, errs
}
