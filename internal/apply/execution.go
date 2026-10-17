package apply

import (
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ExecutionLimits are declared project ceilings, not allocated or consumed
// budgets. Each Setting names the original file/content/field that supplied it.
type ExecutionLimits struct {
	MaxRuns              organization.Setting[int64]
	MaxSessionStarts     organization.Setting[int64]
	MaxSteps             organization.Setting[int64]
	MaxExecutionTime     organization.Setting[time.Duration]
	MaxStepTimeout       organization.Setting[time.Duration]
	HostOperationTimeout organization.Setting[time.Duration]
	MaxReviewRounds      organization.Setting[int]
	MaxContextBytes      organization.Setting[int64]
	MaxSessionSteps      organization.Setting[int64]
	MaxTransientRetries  organization.Setting[int64]
	CreationRoots        organization.Setting[[]snapshot.DirectoryBoundary]
}

// AssignmentDefaults are authored defaults only, never an actual assignment.
type AssignmentDefaults struct {
	Runs          organization.Setting[int64]
	SessionStarts organization.Setting[int64]
	Steps         organization.Setting[int64]
	ExecutionTime organization.Setting[time.Duration]
}

type CompletionReserveDeclaration struct {
	SessionStarts organization.Setting[int64]
	Steps         organization.Setting[int64]
	ExecutionTime organization.Setting[time.Duration]
}

// ExecutionDefaults retains absent instructions as nil. RoleOverride supplies
// actual declared profile/list facts without inventing resolved ProfileFacts.
type ExecutionDefaults struct {
	Manager              organization.RoleOverride
	Reviewer             organization.RoleOverride
	StepTimeout          organization.Setting[time.Duration]
	ReviewRounds         organization.Setting[int]
	CreationRoots        organization.Setting[[]snapshot.DirectoryBoundary]
	BatchNoChangeReviews *organization.Setting[bool]
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
type AuthoredExecution struct {
	APIVersion        organization.Setting[string]
	Strategy          organization.Setting[string]
	Limits            ExecutionLimits
	Assignment        AssignmentDefaults
	CompletionReserve CompletionReserveDeclaration
	Defaults          ExecutionDefaults
	Packages          *organization.Setting[[]organization.PackageOverride]
	Scope             *organization.Setting[[]organization.Setting[organization.ScopeRule]]
	Checks            *organization.Setting[[]ExecutionCheck]
	OwnerApproval     *organization.Setting[[]string]
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
	return CapturedExecutionDecoding{executionDecoding: decodeExecution(m.compiled, m.inputs.Raw), stage: m}
}
func (m PreviewedModel) DecodeExecution() PreviewExecutionDecoding {
	return PreviewExecutionDecoding{executionDecoding: decodeExecution(m.compiled, m.inputs.Raw), stage: m}
}
