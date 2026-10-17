package apply

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

type FileCheckDiagnostic struct {
	Code     string
	Source   organization.SettingSource
	Position *model.Position
	Message  string
}

type fileCheck struct {
	status          model.Status
	accounting      *realization.Accounting
	classifications []realization.Classification
	protections     []realization.Protection
	diagnostics     []FileCheckDiagnostic
}

// Passed describes only structural file accounting and complete input closure.
// It proves no settings, profiles, context, realization or execution readiness.
func (r fileCheck) Status() model.Status {
	if r.status == "" {
		return model.NotCompleted
	}
	return r.status
}

// Accounting returns the actual report whenever accounting ran, even on failure
// or incomplete policy observations. Inspect its findings alongside Status;
// unexplained files may still depend on an uncaptured policy or execution input.
func (r fileCheck) Accounting() (realization.Accounting, bool) {
	if r.accounting == nil {
		return realization.Accounting{}, false
	}
	return *r.accounting, true
}

// Classifications and Protections expose complete sets only after success.
// Protections are lexical restrictions, never creation areas or authority.
func (r fileCheck) Classifications() ([]realization.Classification, bool) {
	if r.Status() != model.Passed {
		return nil, false
	}
	return slices.Clone(r.classifications), true
}
func (r fileCheck) Protections() ([]realization.Protection, bool) {
	if r.Status() != model.Passed {
		return nil, false
	}
	return slices.Clone(r.protections), true
}
func (r fileCheck) Diagnostics() []FileCheckDiagnostic {
	result := slices.Clone(r.diagnostics)
	for i := range result {
		if result[i].Position != nil {
			position := *result[i].Position
			result[i].Position = &position
		}
	}
	return result
}

type CapturedFileCheck struct {
	fileCheck
	stage CapturedRegistryPreparation
}

func (r CapturedFileCheck) RegistryPreparation() CapturedRegistryPreparation { return r.stage }

type PreviewFileCheck struct {
	fileCheck
	stage PreviewRegistryPreparation
}

func (r PreviewFileCheck) RegistryPreparation() PreviewRegistryPreparation { return r.stage }

func (r CapturedRegistryPreparation) CheckFiles() CapturedFileCheck {
	execution := r.stage.DecodeExecution()
	return CapturedFileCheck{fileCheck: checkFiles(r.registryPreparation, r.stage.compiled, r.stage.inputs.Raw, execution.executionDecoding), stage: r}
}
func (r PreviewRegistryPreparation) CheckFiles() PreviewFileCheck {
	execution := r.stage.DecodeExecution()
	return PreviewFileCheck{fileCheck: checkFiles(r.registryPreparation, r.stage.compiled, r.stage.inputs.Raw, execution.executionDecoding), stage: r}
}
