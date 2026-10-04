// Package copyme validates interpretation records over an explicitly supplied
// selective-adoption handoff. It does not acquire source data or adopt policy.
package copyme

import "github.com/Glacius-Labs/Markitect/internal/adoption"

const (
	QueueVersion     = "markitect.example.org/copy-me-queue/v1alpha1"
	CandidateVersion = "markitect.example.org/copy-me-candidate/v1alpha1"
	DecisionVersion  = "markitect.example.org/copy-me-decision/v1alpha1"
	maxRecordBytes   = 8 << 20
)

type Queue struct {
	APIVersion string               `yaml:"apiVersion"`
	Evidence   []Evidence           `yaml:"evidence"`
	Candidates []CandidateReference `yaml:"candidates"`
	Coverage   []adoption.Coverage  `yaml:"coverage"`
	Requests   []EvidenceRequest    `yaml:"requests"`
}

type Evidence struct {
	ID           string   `yaml:"id"`
	Repository   string   `yaml:"repository"`
	Path         string   `yaml:"path"`
	SourceDigest string   `yaml:"sourceDigest"`
	StartLine    *int     `yaml:"startLine,omitempty"`
	EndLine      *int     `yaml:"endLine,omitempty"`
	Stance       string   `yaml:"stance"`
	Observation  string   `yaml:"observation"`
	Excerpt      string   `yaml:"excerpt,omitempty"`
	Duplicates   []string `yaml:"duplicates,omitempty"`
	Conflicts    []string `yaml:"conflicts,omitempty"`
}

type CandidateReference struct {
	StableID string `yaml:"stableID"`
	Path     string `yaml:"path"`
	Digest   string `yaml:"digest"`
}

type EvidenceRequest struct {
	Repository    string `yaml:"repository"`
	Path          string `yaml:"path"`
	Reason        string `yaml:"reason"`
	Insufficiency string `yaml:"insufficiency"`
}

type Candidate struct {
	APIVersion      string          `yaml:"apiVersion"`
	StableID        string          `yaml:"stableID"`
	ProposedRule    string          `yaml:"proposedRule"`
	Scope           string          `yaml:"scope"`
	Conditions      []string        `yaml:"conditions"`
	Classification  string          `yaml:"classification"`
	Support         []string        `yaml:"support"`
	Counterexamples []string        `yaml:"counterexamples"`
	Qualifies       []string        `yaml:"qualifies"`
	Confidence      string          `yaml:"confidence"`
	ConfidenceBasis string          `yaml:"confidenceBasis"`
	Alternatives    []string        `yaml:"alternatives"`
	Uncertainty     []string        `yaml:"uncertainty"`
	Questions       []string        `yaml:"questions"`
	Frequency       *FrequencyClaim `yaml:"frequency,omitempty"`
}

type FrequencyClaim struct {
	Numerator     int    `yaml:"numerator"`
	Denominator   int    `yaml:"denominator"`
	SelectionRule string `yaml:"selectionRule"`
	Repository    string `yaml:"repository"`
	Commit        string `yaml:"commit"`
	Period        string `yaml:"period"`
}

type Decision struct {
	APIVersion      string   `yaml:"apiVersion"`
	ID              string   `yaml:"id"`
	Reviewer        string   `yaml:"reviewer"`
	Date            string   `yaml:"date"`
	Rationale       string   `yaml:"rationale"`
	Scope           string   `yaml:"scope"`
	Status          string   `yaml:"status"`
	CandidateID     string   `yaml:"candidateID"`
	CandidateDigest string   `yaml:"candidateDigest"`
	QueueDigest     string   `yaml:"queueDigest"`
	HandoffDigest   string   `yaml:"handoffDigest"`
	HandoffIdentity string   `yaml:"handoffIdentity"`
	FollowOnIDs     []string `yaml:"followOnIDs,omitempty"`
}

type CandidateReport struct {
	Record Candidate `yaml:"record"`
	Digest string    `yaml:"digest"`
}

type Report struct {
	APIVersion              string                      `yaml:"apiVersion"`
	HandoffID               string                      `yaml:"handoffID"`
	HandoffIdentity         string                      `yaml:"handoffIdentity"`
	HandoffByteDigest       string                      `yaml:"handoffByteDigest"`
	QueueByteDigest         string                      `yaml:"queueByteDigest"`
	Evidence                []Evidence                  `yaml:"evidence"`
	Candidates              []CandidateReport           `yaml:"candidates"`
	PreparationCoverage     []adoption.Coverage         `yaml:"preparationCoverage"`
	Coverage                []adoption.Coverage         `yaml:"coverage"`
	Requests                []EvidenceRequest           `yaml:"requests"`
	Decision                *Decision                   `yaml:"decision,omitempty"`
	DecisionByteDigest      string                      `yaml:"decisionByteDigest,omitempty"`
	OptionalMarkitectRuns   []OptionalMarkitectIdentity `yaml:"optionalMarkitectRuns,omitempty"`
	UnauthenticatedReviewer bool                        `yaml:"unauthenticatedReviewer"`
	Adopted                 bool                        `yaml:"adopted"`
}

type OptionalMarkitectIdentity struct {
	Repository string                     `yaml:"repository"`
	Evidence   adoption.MarkitectEvidence `yaml:"evidence"`
}
