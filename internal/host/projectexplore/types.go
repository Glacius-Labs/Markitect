// Package projectexplore stores contributor-authored exploration, open
// decisions, named-scope readiness, and successful Apply receipts. Drafts in
// this package are operational state; they never become an accepted model.
package projectexplore

import "time"

const (
	APIVersion      = "markitect.example.org/project-exploration/v1alpha1"
	StatusActive    = "active"
	StatusCompleted = "completed"
)

// Record is the durable state for one ordinary-language work item. Digest is
// calculated over the record with Digest cleared.
type Record struct {
	APIVersion       string                     `json:"apiVersion"`
	ID               string                     `json:"id"`
	Status           string                     `json:"status"`
	Request          string                     `json:"request"`
	CreatedAgainst   string                     `json:"createdAgainstBindingDigest"`
	Scopes           []Scope                    `json:"scopes"`
	Decisions        []Decision                 `json:"decisions"`
	Drafts           []DraftProposal            `json:"drafts"`
	Acknowledgements []StructureAcknowledgement `json:"structureAcknowledgements"`
	Completions      []ApplyReceipt             `json:"completions"`
	Digest           string                     `json:"digest"`
}

type Scope struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Goal       string   `json:"goal"`
	Operation  string   `json:"operation"`
	ManagerIDs []string `json:"managerIds"`
}

type Decision struct {
	ID         string   `json:"id"`
	ScopeIDs   []string `json:"scopeIds"`
	Question   string   `json:"question"`
	Blocking   bool     `json:"blocking"`
	Status     string   `json:"status"` // open, answered, deferred
	Answer     string   `json:"answer,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	Authority  string   `json:"authority,omitempty"`
	Provenance string   `json:"provenance,omitempty"`
}

// DraftProposal holds model-file text being discussed. It is never applied by
// this package and is never evidence that the accepted model changed.
type DraftProposal struct {
	ID    string      `json:"id"`
	Goal  string      `json:"goal"`
	Files []DraftFile `json:"files"`
}

type DraftFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Delete  bool   `json:"delete,omitempty"`
}

// BasisFile binds source bytes which are not adequately represented by Git
// HEAD alone, such as project configuration, runtime configuration, or state.
type BasisFile struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// Binding is supplied by the Host after it has loaded the exact fixed project
// model and computed the requested work-item scope. Plan identifiers are
// intentionally absent so this binding remains stable across plan creation.
type Binding struct {
	RepositoryRoot        string      `json:"repositoryRoot"`
	Branch                string      `json:"branch"`
	Head                  string      `json:"head"`
	ModelRevision         string      `json:"modelRevision"`
	ModelAccepted         bool        `json:"modelAccepted"`
	AcceptancePolicy      string      `json:"acceptancePolicy"`
	ProjectDigest         string      `json:"projectDigest"`
	ModelDigest           string      `json:"modelDigest"`
	SnapshotDigest        string      `json:"snapshotDigest"`
	SelectionDigest       string      `json:"selectionDigest"`
	RuntimeDigest         string      `json:"runtimeDigest,omitempty"`
	BriefingDigest        string      `json:"briefingDigest,omitempty"`
	ScopeID               string      `json:"scopeId"`
	ScopeName             string      `json:"scopeName"`
	Goal                  string      `json:"goal"`
	Operation             string      `json:"operation"`
	ManagerIDs            []string    `json:"managerIds"`
	ResponsibleManagerIDs []string    `json:"responsibleManagerIds"`
	RequiredArtifacts     []string    `json:"requiredArtifacts"`
	FileStructure         []string    `json:"fileStructure"`
	Checks                []string    `json:"checks"`
	BasisFiles            []BasisFile `json:"basisFiles"`
}

type StructureAcknowledgement struct {
	ScopeID         string    `json:"scopeId"`
	BindingDigest   string    `json:"bindingDigest"`
	StructureDigest string    `json:"structureDigest"`
	Actor           string    `json:"actor"`
	Authority       string    `json:"authority"`
	Provenance      string    `json:"provenance"`
	RecordedAt      time.Time `json:"recordedAt"`
}

type ReadinessReport struct {
	APIVersion      string   `json:"apiVersion"`
	ExplorationID   string   `json:"explorationId"`
	ScopeID         string   `json:"scopeId"`
	Ready           bool     `json:"ready"`
	Blockers        []string `json:"blockers"`
	OpenDecisions   []string `json:"openDecisions"`
	BindingDigest   string   `json:"bindingDigest"`
	StructureDigest string   `json:"structureDigest"`
	Digest          string   `json:"digest"`
}

// ApplyReceipt is accepted only when it describes a successful, verified
// Apply for the exact exploration binding and acknowledged structure.
type ApplyReceipt struct {
	ScopeID            string    `json:"scopeId"`
	BindingDigest      string    `json:"bindingDigest"`
	StructureDigest    string    `json:"structureDigest"`
	Status             string    `json:"status"`
	RunID              string    `json:"runId"`
	PlanID             string    `json:"planId"`
	PlanDigest         string    `json:"planDigest"`
	CandidateID        string    `json:"candidateId"`
	CandidateDigest    string    `json:"candidateDigest"`
	VerificationID     string    `json:"verificationId"`
	VerificationStatus string    `json:"verificationStatus"`
	VerificationDigest string    `json:"verificationDigest"`
	ApplyID            string    `json:"applyId"`
	ApplyDigest        string    `json:"applyDigest"`
	AppliedAt          time.Time `json:"appliedAt"`
}

type TargetBasis struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Digest string `json:"digest,omitempty"`
}

// WritePlan is an exact preview for creating or CAS-updating one record.
type WritePlan struct {
	APIVersion          string      `json:"apiVersion"`
	RepositoryRoot      string      `json:"repositoryRoot"`
	Branch              string      `json:"branch"`
	Head                string      `json:"head"`
	Binding             Binding     `json:"binding"`
	BindingDigest       string      `json:"bindingDigest"`
	VerifyBasis         bool        `json:"verifyBasis"`
	Path                string      `json:"path"`
	ExpectedStateDigest string      `json:"expectedStateDigest,omitempty"`
	Target              TargetBasis `json:"target"`
	Next                Record      `json:"next"`
	Digest              string      `json:"digest"`
	allowCompletion     bool
}
