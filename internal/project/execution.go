package project

import (
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ExecutionArea is one authored ownership area: Path is its selector, an exact
// repository-relative file, a directory ending in "/" or "./" for the whole
// repository. Reason is optional; a zero Setting is absent.
type ExecutionArea struct {
	Path   organization.Setting[string]
	Owner  organization.Setting[string]
	Reason organization.Setting[string]
}

// ExecutionCheck describes literal project configuration, not a prepared check
// capability, process request or check receipt. RequiredBy is optional.
type ExecutionCheck struct {
	ID         organization.Setting[string]
	Profile    organization.Setting[string]
	Directory  organization.Setting[snapshot.DirectoryBoundary]
	Args       organization.Setting[[]string]
	RequiredBy *organization.Setting[model.Identity]
	Timeout    organization.Setting[time.Duration]
}

// AuthoredExecution contains decoded authored values only. Optional absence is
// retained, even where later configuration resolution defines a default.
// In partial observations a zero Setting has no successfully decoded value.
type ManagerInstructions struct {
	Package organization.Setting[string]
	Files   organization.Setting[[]string]
}
type AuthoredExecution struct {
	APIVersion   organization.Setting[string]
	Instructions *organization.Setting[[]ManagerInstructions]
	Ownership    *organization.Setting[[]ExecutionArea]
	Scope        *organization.Setting[[]organization.Setting[organization.ScopeRule]]
	Checks       *organization.Setting[[]ExecutionCheck]
}

type ExecutionDiagnostic struct {
	Code     string
	Source   organization.SettingSource
	Position *model.Position
	Message  string
}

type executionDecoding struct {
	status      model.Status
	authored    AuthoredExecution
	layout      realization.Layout // Validated against the model; zero when no ownership is declared.
	diagnostics []ExecutionDiagnostic
}

// Passed means this authored project document was decoded and validated in
// this stage. It does not resolve settings, profiles, instructions or budgets,
// and never claims Check, Run or Apply readiness. Missing observations or tool
// representation limits give NotCompleted; known invalid data give Failed.
func (r executionDecoding) Status() model.Status {
	if r.status == "" {
		return model.NotCompleted
	}
	return r.status
}

func (r executionDecoding) Document() (AuthoredExecution, bool) {
	if r.Status() != model.Passed {
		return AuthoredExecution{}, false
	}
	return cloneExecution(r.authored), true
}

// Partial retains successfully decoded fields beside errors. The parent model
// retains the complete original bytes, including every malformed field.
func (r executionDecoding) Partial() AuthoredExecution { return cloneExecution(r.authored) }
func (r executionDecoding) Diagnostics() []ExecutionDiagnostic {
	result := slices.Clone(r.diagnostics)
	for i := range result {
		if result[i].Position != nil {
			p := *result[i].Position
			result[i].Position = &p
		}
	}
	return result
}

type CapturedExecutionDecoding struct {
	executionDecoding
	stage CapturedModel
}

func (r CapturedExecutionDecoding) Model() CapturedModel { return r.stage }

type PreviewExecutionDecoding struct {
	executionDecoding
	stage PreviewedModel
}

func (r PreviewExecutionDecoding) Model() PreviewedModel { return r.stage }

func (m CapturedModel) DecodeExecution() CapturedExecutionDecoding {
	return CapturedExecutionDecoding{executionDecoding: decodeExecution(m.checked, m.inputs.Raw), stage: m}
}
func (m PreviewedModel) DecodeExecution() PreviewExecutionDecoding {
	return PreviewExecutionDecoding{executionDecoding: decodeExecution(m.checked, m.inputs.Raw), stage: m}
}
