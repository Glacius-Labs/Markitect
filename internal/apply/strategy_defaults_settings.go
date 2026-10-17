package apply

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// DeclarativeSettings supplies source-bound selections for later resolution.
// Profiles remain names and instructions remain ordered references. Overrides
// remain exact authored overrides, with no ancestor inheritance. This is not
// organization.Settings, acquired profile facts or runtime readiness.
type DeclarativeSettings struct {
	Defaults      organization.SettingsDefaults
	Overrides     []organization.PackageOverride
	Ceilings      organization.SettingsCeilings
	OwnerApproval organization.Setting[[]string]
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
// and project ceilings passed. No profile or instruction content is resolved.
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
	return CapturedDefaultsPreparation{settingsPreparation: prepareStrategySettings(e.executionDecoding, e.stage.compiled, input), stage: e}
}

func (e PreviewExecutionDecoding) PrepareDefaults(input StrategyDefaultsInput) PreviewDefaultsPreparation {
	return PreviewDefaultsPreparation{settingsPreparation: prepareStrategySettings(e.executionDecoding, e.stage.compiled, input), stage: e}
}

func strategySettingsValues(effective AuthoredExecution) DeclarativeSettings {
	e := cloneExecution(effective)
	r := DeclarativeSettings{
		Defaults: organization.SettingsDefaults{
			Manager:     organization.RoleDefaults{Profile: *e.Defaults.Manager.Profile, Instructions: *e.Defaults.Manager.Instructions},
			Reviewer:    organization.RoleDefaults{Profile: *e.Defaults.Reviewer.Profile, Instructions: *e.Defaults.Reviewer.Instructions},
			StepTimeout: e.Defaults.StepTimeout, ReviewRounds: *e.Defaults.ReviewRounds,
			CreationRoots: e.Defaults.CreationRoots, BatchNoChangeReviews: *e.Defaults.BatchNoChangeReviews,
		},
		Ceilings:      organization.SettingsCeilings{StepTimeout: e.Limits.MaxStepTimeout, ReviewRounds: e.Limits.MaxReviewRounds, CreationRoots: e.Limits.CreationRoots},
		OwnerApproval: *e.OwnerApproval,
	}
	if e.Packages != nil {
		r.Overrides = e.Packages.Value
	}
	return r
}

func prepareStrategySettings(parent executionDecoding, compiled model.Result, input StrategyDefaultsInput) (r settingsPreparation) {
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
	if e.Defaults.ReviewRounds == nil {
		e.Defaults.ReviewRounds = executionPointer(&r.product.rounds)
	}
	if e.Defaults.Manager.Instructions == nil {
		e.Defaults.Manager.Instructions = executionListPointer(&r.product.manager)
	}
	if e.Defaults.Reviewer.Instructions == nil {
		e.Defaults.Reviewer.Instructions = executionListPointer(&r.product.reviewer)
	}
	if e.Defaults.BatchNoChangeReviews == nil {
		e.Defaults.BatchNoChangeReviews = executionPointer(&r.product.batch)
	}
	if e.OwnerApproval == nil {
		e.OwnerApproval = executionListPointer(&r.product.approval)
	}
	topology, err := organization.BuildTopology(compiled)
	if err == nil {
		values := strategySettingsValues(r.effective)
		err = organization.ValidateSettingsSelections(topology, values.Defaults, values.Overrides, values.Ceilings)
		for _, name := range values.OwnerApproval.Value {
			if _, found := topology.Manager(name); !found {
				r.status = model.Failed
				r.diagnostics = append(r.diagnostics, StrategyDefaultsDiagnostic{Code: "settings.owner_approval_package", Source: values.OwnerApproval.Source, Message: fmt.Sprintf("Unknown or imported owner-approval Package %q.", name)})
			}
		}
	}
	if err != nil {
		r.status = model.Failed
		r.diagnostics = append(r.diagnostics, StrategyDefaultsDiagnostic{Code: "settings.selections", Message: err.Error()})
	}
	return r
}
