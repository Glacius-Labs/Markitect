package apply

import (
	"cmp"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// DeclarativeSettings supplies source-bound selections for later resolution.
// Profiles remain names and instructions remain ordered references. Overrides
// remain exact authored overrides, with no ancestor inheritance. This is not
// organization.Settings, acquired profile facts or runtime readiness.
type DeclarativeSettings struct {
	Defaults  organization.SettingsDefaults
	Overrides []organization.PackageOverride
}

type settingsPreparation struct {
	status      model.Status
	product     strategyDefaults
	effective   AuthoredExecution
	diagnostics []StrategyDefaultsDiagnostic
}

func (r settingsPreparation) Status() model.Status {
	if r.status == "" {
		return model.NotCompleted
	}
	return r.status
}

func (r settingsPreparation) ProductInput() StrategyDefaultsInput {
	input := r.product.input
	input.Bytes = slices.Clone(input.Bytes)
	return input
}

func (r settingsPreparation) ProductSource() organization.SettingSource { return r.product.source }

// Declarations exposes complete declarative selections only after their source
// and organization validation passed. No profile or instruction content is resolved.
func (r settingsPreparation) Declarations() (DeclarativeSettings, bool) {
	if r.Status() != model.Passed {
		return DeclarativeSettings{}, false
	}
	return strategySettingsValues(r.effective), true
}

// Diagnostics retain actual decoder sources. Organization validation errors
// have no field coordinates; their original values remain in Execution and
// ProductInput instead of being attributed to an invented execution field.
func (r settingsPreparation) Diagnostics() []StrategyDefaultsDiagnostic {
	result := slices.Clone(r.diagnostics)
	for i := range result {
		if result[i].Position != nil {
			position := *result[i].Position
			result[i].Position = &position
		}
	}
	return result
}

type CapturedDefaultsPreparation struct {
	settingsPreparation
	stage CapturedExecutionDecoding
}

func (r CapturedDefaultsPreparation) Execution() CapturedExecutionDecoding { return r.stage }

type PreviewDefaultsPreparation struct {
	settingsPreparation
	stage PreviewExecutionDecoding
}

func (r PreviewDefaultsPreparation) Execution() PreviewExecutionDecoding { return r.stage }

func (e CapturedExecutionDecoding) PrepareDefaults(input StrategyDefaultsInput) CapturedDefaultsPreparation {
	return CapturedDefaultsPreparation{settingsPreparation: prepareStrategySettings(e.executionDecoding, input), stage: e}
}

func (e PreviewExecutionDecoding) PrepareDefaults(input StrategyDefaultsInput) PreviewDefaultsPreparation {
	return PreviewDefaultsPreparation{settingsPreparation: prepareStrategySettings(e.executionDecoding, input), stage: e}
}

func strategySettingsValues(effective AuthoredExecution) DeclarativeSettings {
	e := cloneExecution(effective)
	r := DeclarativeSettings{Defaults: organization.SettingsDefaults{
		Execute: organization.RoleDefaults{Profile: *e.Defaults.Execute.Profile, Instructions: *e.Defaults.Execute.Instructions},
		Verify:  organization.RoleDefaults{Profile: *e.Defaults.Verify.Profile, Instructions: *e.Defaults.Verify.Instructions},
	}}
	if e.Packages != nil {
		r.Overrides = e.Packages.Value
	}
	return r
}

func prepareStrategySettings(parent executionDecoding, input StrategyDefaultsInput) (r settingsPreparation) {
	r = settingsPreparation{status: parent.Status(), product: decodeStrategyDefaults(input)}
	r.diagnostics = slices.Clone(r.product.diagnostics)
	for _, diagnostic := range parent.Diagnostics() {
		r.diagnostics = append(r.diagnostics, StrategyDefaultsDiagnostic(diagnostic))
	}
	defer func() {
		slices.SortFunc(r.diagnostics, func(a, b StrategyDefaultsDiagnostic) int {
			ap, bp := model.Position{}, model.Position{}
			an, bn := 0, 0
			if a.Position != nil {
				ap, an = *a.Position, 1
			}
			if b.Position != nil {
				bp, bn = *b.Position, 1
			}
			return cmp.Or(cmp.Compare(a.Source.File, b.Source.File), cmp.Compare(a.Source.Field, b.Source.Field), cmp.Compare(a.Code, b.Code), cmp.Compare(a.Message, b.Message), cmp.Compare(an, bn), cmp.Compare(ap.Line, bp.Line), cmp.Compare(ap.Column, bp.Column))
		})
	}()
	if r.product.status == model.Failed || r.status != model.Failed && r.product.status == model.NotCompleted {
		r.status = r.product.status
	}
	if parent.Status() != model.Passed {
		r.diagnostics = append(r.diagnostics, StrategyDefaultsDiagnostic{Code: "settings.execution_not_passed", Message: "Declarative settings require a passed bound execution decode."})
	}
	if r.status != model.Passed {
		return r
	}
	r.effective = cloneExecution(parent.authored)
	e := &r.effective
	if e.Defaults.Execute.Instructions == nil {
		e.Defaults.Execute.Instructions = executionListPointer(&r.product.execute)
	}
	if e.Defaults.Verify.Instructions == nil {
		e.Defaults.Verify.Instructions = executionListPointer(&r.product.verify)
	}
	return r
}
