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

type hostSettingsPreparation struct {
	status         model.Status
	settings       *organization.Settings
	checks         []CheckSelection
	checksComplete bool
	profiles       []PreparedProfile
	required       []SourceLocation
	diagnostics    []HostProfileDiagnostic
}

// Status describes static preparation only, never runtime readiness. Parent
// failures dominate incomplete observations; partial evidence remains available.
func (r hostSettingsPreparation) Status() model.Status {
	if r.status == "" {
		return model.NotCompleted
	}
	return r.status
}
func (r hostSettingsPreparation) Settings() (organization.Settings, bool) {
	if r.settings == nil {
		return organization.Settings{}, false
	}
	return *r.settings, true
}
func (r hostSettingsPreparation) Checks() ([]CheckSelection, bool) {
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
func (r hostSettingsPreparation) ProfileObservations() []PreparedProfile {
	result := slices.Clone(r.profiles)
	for i := range result {
		result[i] = clonePreparedProfile(result[i])
	}
	return result
}
func (r hostSettingsPreparation) RequiredInstructions() []SourceLocation {
	return slices.Clone(r.required)
}
func (r hostSettingsPreparation) Diagnostics() []HostProfileDiagnostic {
	return cloneHostDiagnostics(r.diagnostics)
}

func clonePreparedProfile(p PreparedProfile) PreparedProfile {
	p.input = p.Input()
	p.observation = p.Observation()
	return p
}

type CapturedSettingsPreparation struct {
	hostSettingsPreparation
	parent   CapturedFileCheck
	defaults CapturedDefaultsPreparation
}

func (r CapturedSettingsPreparation) FileCheck() CapturedFileCheck          { return r.parent }
func (r CapturedSettingsPreparation) Defaults() CapturedDefaultsPreparation { return r.defaults }

type PreviewSettingsPreparation struct {
	hostSettingsPreparation
	parent   PreviewFileCheck
	defaults PreviewDefaultsPreparation
}

func (r PreviewSettingsPreparation) FileCheck() PreviewFileCheck          { return r.parent }
func (r PreviewSettingsPreparation) Defaults() PreviewDefaultsPreparation { return r.defaults }

// PrepareSettings accepts only the actual Defaults preparation from the same
// complete Model/Raw/Request/Guard parent. No free execution or model is accepted.
func (p CapturedFileCheck) PrepareSettings(ctx context.Context, defaults CapturedDefaultsPreparation, agents AgentProfilePreparer, checks CheckProfilePreparer) (CapturedSettingsPreparation, error) {
	m := p.stage.stage
	r, err := prepareHostSettings(ctx, p.Status(), defaults.settingsPreparation, m.compiled, m.inputs.Raw, m.DecodeExecution().executionDecoding, reflect.DeepEqual(m, defaults.stage.stage), agents, checks)
	return CapturedSettingsPreparation{hostSettingsPreparation: r, parent: p, defaults: defaults}, err
}

// Preview retains its own working/fixed revision and never acquires a guard.
func (p PreviewFileCheck) PrepareSettings(ctx context.Context, defaults PreviewDefaultsPreparation, agents AgentProfilePreparer, checks CheckProfilePreparer) (PreviewSettingsPreparation, error) {
	m := p.stage.stage
	r, err := prepareHostSettings(ctx, p.Status(), defaults.settingsPreparation, m.compiled, m.inputs.Raw, m.DecodeExecution().executionDecoding, reflect.DeepEqual(m, defaults.stage.stage), agents, checks)
	return PreviewSettingsPreparation{hostSettingsPreparation: r, parent: p, defaults: defaults}, err
}

func hostStatus(a, b model.Status) model.Status {
	if a == model.Failed || b == model.Failed {
		return model.Failed
	}
	if a != model.Passed || b != model.Passed {
		return model.NotCompleted
	}
	return model.Passed
}

func (r *hostSettingsPreparation) issue(status model.Status, code string, source organization.SettingSource, message string) {
	r.status = hostStatus(r.Status(), status)
	r.diagnostics = append(r.diagnostics, HostProfileDiagnostic{Code: "settings." + code, Source: source, Message: message})
}
