package apply

import (
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ExecutionBudget is the authored bound of one run: ExecutionTime is the wall
// clock of the whole run, TurnTimeout bounds one Agent turn, AgentTurns counts
// every Agent turn of Managers, reviewers and helpers and Passes counts the
// Passes of the run, forward and correction Passes and Verify alike. Each
// Setting names the original file/content/field.
type ExecutionBudget struct {
	ExecutionTime organization.Setting[time.Duration]
	TurnTimeout   organization.Setting[time.Duration]
	AgentTurns    organization.Setting[int]
	Passes        organization.Setting[int]
}

// ExecutionDefaults retains absent instructions as nil. RoleOverride supplies
// actual declared profile/list facts without inventing resolved ProfileFacts.
type ExecutionDefaults struct {
	Execute organization.RoleOverride
	Verify  organization.RoleOverride
}

// ExecutionArea is one authored ownership area: Path is its selector, an exact
// repository-relative file, a directory ending in "/" or "./" for the whole
// repository. Reason is optional; a zero Setting is absent.
type ExecutionArea struct {
	Path   organization.Setting[string]
	Owner  organization.Setting[string]
	Reason organization.Setting[string]
}

// ExecutionHelper is one authored helper role of every Manager's team. A nil
// Profile means the Manager's own Execute profile; Instructions are optional.
type ExecutionHelper struct {
	Role         organization.Setting[string]
	Profile      *organization.Setting[string]
	Writes       organization.Setting[bool]
	Instructions *organization.Setting[[]string]
}

// ExecutionTeam is the authored local team template: the helper roles every
// Manager may instantiate and how many may be active for one Manager at once.
type ExecutionTeam struct {
	MaxHelpers organization.Setting[int]
	Helpers    []ExecutionHelper
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
	APIVersion organization.Setting[string]
	Strategy   organization.Setting[string]
	Budget     ExecutionBudget
	Defaults   ExecutionDefaults
	Packages   *organization.Setting[[]organization.PackageOverride]
	Ownership  *organization.Setting[[]ExecutionArea]
	Team       *ExecutionTeam
	Scope      *organization.Setting[[]organization.Setting[organization.ScopeRule]]
	Checks     *organization.Setting[[]ExecutionCheck]
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
