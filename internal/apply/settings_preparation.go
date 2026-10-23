package apply

import (
	"context"
	"reflect"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

type CheckSelection struct {
	declaration ExecutionCheck
	profile     PreparedProfile
}

func (c CheckSelection) Declaration() ExecutionCheck {
	r := c.declaration
	r.Args.Value = slices.Clone(r.Args.Value)
	r.RequiredBy = executionPointer(r.RequiredBy)
	return r
}
func (c CheckSelection) Profile() PreparedProfile { return clonePreparedProfile(c.profile) }

type runtimeSettingsPreparation struct {
	status         model.Status
	settings       *organization.Settings
	facts          *organization.SettingsFacts
	budget         *ExecutionBudget
	strategy       *Strategy
	checks         []CheckSelection
	checksComplete bool
	profiles       []PreparedProfile
	required       []SourceLocation
	diagnostics    []RuntimeProfileDiagnostic
}

// Status describes static preparation only, never runtime readiness. Parent
// failures dominate incomplete observations; partial evidence remains available.
func (r runtimeSettingsPreparation) Status() model.Status {
	if r.status == "" {
		return model.NotCompleted
	}
	return r.status
}
func (r runtimeSettingsPreparation) Settings() (organization.Settings, bool) {
	if r.settings == nil {
		return organization.Settings{}, false
	}
	return *r.settings, true
}

// SettingsFacts returns the facts the settings were resolved from.
func (r runtimeSettingsPreparation) SettingsFacts() (organization.SettingsFacts, bool) {
	if r.facts == nil {
		return organization.SettingsFacts{}, false
	}
	f := *r.facts
	f.Overrides = slices.Clone(f.Overrides)
	f.Profiles = slices.Clone(f.Profiles)
	f.Instructions = slices.Clone(f.Instructions)
	return f, true
}

// Budget returns the authored bounds of the run, available together with the
// resolved Settings.
func (r runtimeSettingsPreparation) Budget() (ExecutionBudget, bool) {
	if r.budget == nil {
		return ExecutionBudget{}, false
	}
	return *r.budget, true
}

// Strategy returns the authored strategy of the run, available together with
// the resolved Settings.
func (r runtimeSettingsPreparation) Strategy() (Strategy, bool) {
	if r.strategy == nil {
		return "", false
	}
	return *r.strategy, true
}
func (r runtimeSettingsPreparation) Checks() ([]CheckSelection, bool) {
	if !r.checksComplete {
		return nil, false
	}
	result := slices.Clone(r.checks)
	for i := range result {
		result[i].declaration = result[i].Declaration()
		result[i].profile = clonePreparedProfile(result[i].profile)
	}
	return result, true
}
func (r runtimeSettingsPreparation) ProfileObservations() []PreparedProfile {
	result := slices.Clone(r.profiles)
	for i := range result {
		result[i] = clonePreparedProfile(result[i])
	}
	return result
}
func (r runtimeSettingsPreparation) RequiredInstructions() []SourceLocation {
	return slices.Clone(r.required)
}
func (r runtimeSettingsPreparation) Diagnostics() []RuntimeProfileDiagnostic {
	return cloneRuntimeDiagnostics(r.diagnostics)
}

func clonePreparedProfile(p PreparedProfile) PreparedProfile {
	p.input = p.Input()
	p.observation = p.Observation()
	return p
}

type CapturedSettingsPreparation struct {
	runtimeSettingsPreparation
	parent   CapturedFileCheck
	defaults CapturedDefaultsPreparation
}

func (r CapturedSettingsPreparation) FileCheck() CapturedFileCheck          { return r.parent }
func (r CapturedSettingsPreparation) Defaults() CapturedDefaultsPreparation { return r.defaults }

type PreviewSettingsPreparation struct {
	runtimeSettingsPreparation
	parent   PreviewFileCheck
	defaults PreviewDefaultsPreparation
}

func (r PreviewSettingsPreparation) FileCheck() PreviewFileCheck          { return r.parent }
func (r PreviewSettingsPreparation) Defaults() PreviewDefaultsPreparation { return r.defaults }

// PrepareSettings accepts only the actual Defaults preparation from the same
// complete Model/Raw/Request/Guard parent. No free execution or model is accepted.
func (p CapturedFileCheck) PrepareSettings(ctx context.Context, defaults CapturedDefaultsPreparation, agents AgentProfilePreparer, checks CheckProfilePreparer) (CapturedSettingsPreparation, error) {
	m := p.stage.stage
	r, err := prepareRuntimeSettings(ctx, p.Status(), defaults.settingsPreparation, m.checked, m.inputs.Raw, m.DecodeExecution().executionDecoding, reflect.DeepEqual(m, defaults.stage.stage), agents, checks)
	return CapturedSettingsPreparation{runtimeSettingsPreparation: r, parent: p, defaults: defaults}, err
}

// Preview retains its own working/fixed revision and never acquires a guard.
func (p PreviewFileCheck) PrepareSettings(ctx context.Context, defaults PreviewDefaultsPreparation, agents AgentProfilePreparer, checks CheckProfilePreparer) (PreviewSettingsPreparation, error) {
	m := p.stage.stage
	r, err := prepareRuntimeSettings(ctx, p.Status(), defaults.settingsPreparation, m.checked, m.inputs.Raw, m.DecodeExecution().executionDecoding, reflect.DeepEqual(m, defaults.stage.stage), agents, checks)
	return PreviewSettingsPreparation{runtimeSettingsPreparation: r, parent: p, defaults: defaults}, err
}

func runtimeStatus(a, b model.Status) model.Status {
	if a == model.Failed || b == model.Failed {
		return model.Failed
	}
	if a != model.Passed || b != model.Passed {
		return model.NotCompleted
	}
	return model.Passed
}

func (r *runtimeSettingsPreparation) issue(status model.Status, code string, source organization.SettingSource, message string) {
	r.status = runtimeStatus(r.Status(), status)
	r.diagnostics = append(r.diagnostics, RuntimeProfileDiagnostic{Code: "settings." + code, Source: source, Message: message})
}
