// Package projectadoption validates fixed-snapshot Brownfield evidence and
// prepares owner-decided model-only adoption through projectwork.
package projectadoption

import "github.com/Glacius-Labs/Markitect/internal/host/projectwork"

const (
	DiscoveryVersion     = "markitect.example.org/project-discovery/v1alpha1"
	DistillationVersion  = "markitect.example.org/project-distillation/v1alpha1"
	ResolutionVersion    = "markitect.example.org/project-resolution/v1alpha1"
	AdoptionPlanVersion  = "markitect.example.org/project-adoption-plan/v1alpha1"
	RuntimeRecordVersion = "markitect.example.org/submitted-runtime-record/v1alpha1"
)

// PathReason identifies a path or directory prefix in the declared scope.
// ScopeRoots describe what the owner means to investigate; they do not grant
// permission to read every file below them.
type PathReason struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type SelectedPath struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
	Basis  string `json:"basis"` // owner-supplied source kind, never inferred from the path
}

type RepositoryIdentity struct {
	Root         string `json:"root"`
	GitDir       string `json:"gitDir"`
	CommonDir    string `json:"commonDir"`
	ObjectFormat string `json:"objectFormat"`
	Digest       string `json:"digest"`
}

// Evidence contains only a caller-selected regular UTF-8 Git blob. Content is
// preserved so later validation never needs to reopen or broaden the source.
type Evidence struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Basis   string `json:"basis"`
	Mode    string `json:"mode"`
	Digest  string `json:"digest"`
	Content string `json:"content"`
}

type DiscoveryRequest struct {
	APIVersion string         `json:"apiVersion"`
	ID         string         `json:"id"`
	Purpose    string         `json:"purpose"`
	Review     string         `json:"review"`
	Commit     string         `json:"commit"`
	ScopeRoots []string       `json:"scopeRoots"`
	Selected   []SelectedPath `json:"selected"`
	Exclusions []PathReason   `json:"exclusions"`
	Unselected []PathReason   `json:"unselected"`
}

type Discovery struct {
	APIVersion string             `json:"apiVersion"`
	ID         string             `json:"id"`
	Purpose    string             `json:"purpose"`
	Review     string             `json:"review"`
	Identity   RepositoryIdentity `json:"identity"`
	Commit     string             `json:"commit"`
	ScopeRoots []string           `json:"scopeRoots"`
	Selected   []SelectedPath     `json:"selected"`
	Exclusions []PathReason       `json:"exclusions"`
	Unselected []PathReason       `json:"unselected"`
	Evidence   []Evidence         `json:"evidence"`
	Digest     string             `json:"digest"`
}

type EvidenceRef struct {
	EvidenceID string `json:"evidenceId"`
	StartLine  int    `json:"startLine"`
	EndLine    int    `json:"endLine"`
	Excerpt    string `json:"excerpt"`
}

type Claim struct {
	ID          string              `json:"id"`
	ScopeID     string              `json:"scopeId"`
	Kind        string              `json:"kind"`   // observation, documented-intent, submitted-runtime-record, hypothesis
	Method      string              `json:"method"` // static-source, documentation, submitted-record, synthesis
	Statement   string              `json:"statement"`
	Evidence    []EvidenceRef       `json:"evidence"`
	Uncertainty []string            `json:"uncertainty"`
	Runtime     *RuntimeObservation `json:"runtime,omitempty"`
}

// RuntimeObservation copies a selected, caller-submitted record. Its values
// are not an attestation that Markitect executed or authenticated the command.
type RuntimeObservation struct {
	EvidenceID           string         `json:"evidenceId"`
	RecordSourceRevision string         `json:"recordSourceRevision"`
	SourceRelation       string         `json:"sourceRelation"` // same-discovery-commit or historical
	Command              []string       `json:"command"`
	ExitCode             *int           `json:"exitCode"`
	RunnerDigest         string         `json:"runnerDigest"`
	Inputs               []RuntimeInput `json:"inputs"`
}

// RuntimeInput is a path and content digest asserted by the submitted runtime
// record. It is not an execution attestation.
type RuntimeInput struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type TermOccurrence struct {
	EvidenceID string `json:"evidenceId"`
	StartLine  int    `json:"startLine"`
	EndLine    int    `json:"endLine"`
	Excerpt    string `json:"excerpt"`
}

type Term struct {
	ID          string           `json:"id"`
	Text        string           `json:"text"`
	Context     string           `json:"context"`
	Occurrences []TermOccurrence `json:"occurrences"`
	Synonyms    []string         `json:"synonyms"`
	Ambiguities []string         `json:"ambiguities"`
}

type Contradiction struct {
	ID          string   `json:"id"`
	ScopeID     string   `json:"scopeId"`
	ClaimIDs    []string `json:"claimIds"`
	QuestionID  string   `json:"questionId"`
	Description string   `json:"description"`
}

type Question struct {
	ID           string   `json:"id"`
	ScopeID      string   `json:"scopeId"`
	Prompt       string   `json:"prompt"`
	Alternatives []string `json:"alternatives"`
	ClaimIDs     []string `json:"claimIds"`
	Blocking     *bool    `json:"blocking"`
}

type ScopeProposal struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	ParentID       string   `json:"parentId,omitempty"`
	ClaimIDs       []string `json:"claimIds"`
	OwnerCandidate string   `json:"ownerCandidate,omitempty"`
}

type ModelProposal struct {
	Goal  string         `json:"goal"`
	Files []ProposedFile `json:"files"`
}

type ProposedFile struct {
	ScopeID string `json:"scopeId"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Distillation is an externally supplied, proposal-only report. The P1
// validator checks its exact evidence grounding and bindings; it does not run
// a provider or treat a confidence value as authority.
type Distillation struct {
	APIVersion      string          `json:"apiVersion"`
	DiscoveryDigest string          `json:"discoveryDigest"`
	Method          string          `json:"method"` // human-review, static-tool, or agent-assisted
	RunnerIdentity  string          `json:"runnerIdentity,omitempty"`
	RunnerDigest    string          `json:"runnerDigest,omitempty"`
	SchemaDigest    string          `json:"schemaDigest"`
	Claims          []Claim         `json:"claims"`
	Terms           []Term          `json:"terms"`
	Contradictions  []Contradiction `json:"contradictions"`
	Questions       []Question      `json:"questions"`
	Scopes          []ScopeProposal `json:"scopes"`
	Proposal        ModelProposal   `json:"proposal"`
	Digest          string          `json:"digest"`
}

type QuestionResolution struct {
	QuestionID  string `json:"questionId"`
	ScopeID     string `json:"scopeId"`
	Disposition string `json:"disposition"` // answer or defer
	Answer      string `json:"answer,omitempty"`
	Reason      string `json:"reason"`
}

type ScopeResolution struct {
	ScopeID string `json:"scopeId"`
	Status  string `json:"status"` // adopt or defer
	Reason  string `json:"reason"`
}

type Resolution struct {
	APIVersion         string               `json:"apiVersion"`
	DiscoveryDigest    string               `json:"discoveryDigest"`
	DistillationDigest string               `json:"distillationDigest"`
	ProposalDigest     string               `json:"proposalDigest"`
	TargetBasis        string               `json:"targetBasis"`
	SchemaDigest       string               `json:"schemaDigest"`
	BuildDigest        string               `json:"buildDigest"`
	Actor              string               `json:"actor"`
	AuthorityClaim     string               `json:"authorityClaim"`
	DecisionReference  string               `json:"decisionReference"`
	Authenticated      *bool                `json:"authenticated"`
	Questions          []QuestionResolution `json:"questions"`
	Scopes             []ScopeResolution    `json:"scopes"`
	Digest             string               `json:"digest"`
}

type AdoptionPlan struct {
	APIVersion         string               `json:"apiVersion"`
	DiscoveryDigest    string               `json:"discoveryDigest"`
	DistillationDigest string               `json:"distillationDigest"`
	ResolutionDigest   string               `json:"resolutionDigest"`
	TargetBasis        string               `json:"targetBasis"`
	PlanDigest         string               `json:"planDigest"`
	Status             string               `json:"status"` // complete or partial
	AdoptedScopes      []string             `json:"adoptedScopes"`
	DeferredScopes     []string             `json:"deferredScopes"`
	Edit               projectwork.EditPlan `json:"edit"`
}
