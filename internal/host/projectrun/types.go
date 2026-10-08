// Package projectrun coordinates bounded project managers over immutable
// snapshots and staged candidates. It owns operational run state, not model
// semantics or provider transport.
package projectrun

import (
	"context"
	"errors"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

type Project = projectwork.Project
type Mutation = projectwork.Mutation
type EditPlan = projectwork.EditPlan
type Snapshot = snapshot.Snapshot

const (
	APIVersion  = "markitect.example.org/project-run/v1alpha1"
	RuntimePath = ".markitect/runtime.yaml"
	RunsPath    = ".markitect/runs"

	ModeControlledLocal = "controlled-local"
	ModeIsolated        = "isolated"

	StatusPlanned     = "planned"
	StatusRunning     = "running"
	StatusInterrupted = "interrupted"
	StatusBlocked     = "blocked"
	StatusFailed      = "failed"
	StatusIntegrated  = "integrated"
	StatusVerified    = "verified"
	StatusApplied     = "applied"
	StatusSuperseded  = "superseded"
)

var (
	ErrStale       = errors.New("project run is stale")
	ErrLocked      = errors.New("project runtime writer lock is held")
	ErrNotFound    = errors.New("project run was not found")
	ErrNotRunnable = errors.New("project run is not runnable")
)

// Runtime binds one explicit agent invocation per manager role and finite
// resource ceilings. The configured mode is descriptive; only a verified
// host-controlled launcher can establish isolated execution.
type Runtime struct {
	APIVersion       string           `json:"apiVersion" yaml:"apiVersion"`
	Mode             string           `json:"mode" yaml:"mode"`
	RequireIsolation bool             `json:"requireIsolation" yaml:"requireIsolation"`
	Agents           map[string]Agent `json:"agents" yaml:"agents"`
	Verifier         *Agent           `json:"verifier,omitempty" yaml:"verifier,omitempty"`
	Limits           Limits           `json:"limits" yaml:"limits"`
}

type Agent struct {
	Command         string                  `json:"command" yaml:"command"`
	Args            []string                `json:"args" yaml:"args"`
	Model           string                  `json:"model" yaml:"model"`
	ModelOptions    any                     `json:"modelOptions,omitempty" yaml:"modelOptions"`
	ProviderVersion string                  `json:"providerVersion" yaml:"providerVersion"`
	Timeout         Duration                `json:"timeout" yaml:"timeout"`
	MaxStdoutBytes  int                     `json:"maxStdoutBytes" yaml:"maxStdoutBytes"`
	MaxStderrBytes  int                     `json:"maxStderrBytes" yaml:"maxStderrBytes"`
	RuntimeFiles    []agentexec.RuntimeFile `json:"runtimeFiles" yaml:"runtimeFiles"`
	Environment     []string                `json:"environment,omitempty" yaml:"environment,omitempty"`
	Pricing         Pricing                 `json:"pricing" yaml:"pricing"`
}

// Pricing is an explicit estimate used to enforce the configured run budget.
// It does not claim to be a provider invoice or billing record.
type Pricing struct {
	InputMicrosPerMillion  int64 `json:"inputMicrosPerMillion" yaml:"inputMicrosPerMillion"`
	OutputMicrosPerMillion int64 `json:"outputMicrosPerMillion" yaml:"outputMicrosPerMillion"`
}

type Limits struct {
	MaxDepth              int      `json:"maxDepth" yaml:"maxDepth"`
	MaxStarts             int      `json:"maxStarts" yaml:"maxStarts"`
	MaxRetries            int      `json:"maxRetries" yaml:"maxRetries"`
	MaxParallel           int      `json:"maxParallel" yaml:"maxParallel"`
	MaxDuration           Duration `json:"maxDuration" yaml:"maxDuration"`
	MaxCostMicros         int64    `json:"maxCostMicros" yaml:"maxCostMicros"`
	MaxCandidateFileBytes int64    `json:"maxCandidateFileBytes" yaml:"maxCandidateFileBytes"`
	MaxCandidateBytes     int64    `json:"maxCandidateBytes" yaml:"maxCandidateBytes"`
}

// Duration accepts an explicit Go duration string in runtime.yaml.
type Duration time.Duration

type PlanRequest struct {
	Goal         string    `json:"goal"`
	Managers     []string  `json:"managers,omitempty"`
	BaseRevision string    `json:"baseRevision,omitempty"`
	ModelEdit    *Mutation `json:"modelEdit,omitempty"`
	// ExecuteAuthorized is set by an explicitly invoked write/run command. It
	// represents the caller's existing authorization, not an agent decision.
	ExecuteAuthorized bool `json:"executeAuthorized"`
}

type PlanRecord struct {
	APIVersion           string            `json:"apiVersion"`
	ID                   string            `json:"id"`
	Status               string            `json:"status"`
	Goal                 string            `json:"goal"`
	ExecuteAuthorized    bool              `json:"executeAuthorized"`
	Root                 string            `json:"-"`
	BaseRevision         string            `json:"baseRevision"`
	RepositoryDigest     string            `json:"repositoryDigest"`
	BaseSnapshot         string            `json:"baseSnapshot"`
	WorkingSnapshot      string            `json:"workingSnapshot"`
	BaseProjectDigest    string            `json:"baseProjectDigest"`
	WorkingProjectDigest string            `json:"workingProjectDigest"`
	BaseModelDigest      string            `json:"baseModelDigest"`
	ModelDigest          string            `json:"modelDigest"`
	ReportDigest         string            `json:"reportDigest"`
	RuntimeDigest        string            `json:"runtimeDigest"`
	PlannedAt            time.Time         `json:"plannedAt"`
	Managers             []ManagerTask     `json:"managers"`
	Checks               []CheckPlan       `json:"checks"`
	ModelEdit            *EditPlan         `json:"modelEdit,omitempty"`
	InitialCandidateID   string            `json:"initialCandidateId"`
	RuntimeAgents        map[string]string `json:"runtimeAgents"`
	Findings             []string          `json:"findings,omitempty"`
	Blockers             []string          `json:"blockers,omitempty"`
	Digest               string            `json:"digest"`
}

type ManagerTask struct {
	ID                     string       `json:"id"`
	ManagerID              string       `json:"managerId"`
	ParentTask             string       `json:"parentTask,omitempty"`
	Depth                  int          `json:"depth"`
	Goal                   string       `json:"goal"`
	Owns                   []string     `json:"owns"`
	Statements             []string     `json:"statements"`
	Artifacts              []string     `json:"artifacts"`
	Checks                 []string     `json:"checks"`
	State                  string       `json:"state"`
	ReportStatus           string       `json:"reportStatus"`
	Attempts               int          `json:"attempts,omitempty"`
	ReportID               string       `json:"reportId,omitempty"`
	CandidateID            string       `json:"candidateId,omitempty"`
	IntegrationReportID    string       `json:"integrationReportId,omitempty"`
	IntegrationCandidateID string       `json:"integrationCandidateId,omitempty"`
	WrittenPaths           []string     `json:"writtenPaths,omitempty"`
	IntegratedPaths        []string     `json:"integratedPaths,omitempty"`
	Delegations            []Delegation `json:"delegations,omitempty"`
	Summary                string       `json:"summary,omitempty"`
	Questions              []string     `json:"questions,omitempty"`
	Risks                  []string     `json:"risks,omitempty"`
}

type CheckPlan struct {
	ID               string   `json:"id"`
	Owner            string   `json:"owner"`
	Command          []string `json:"command"`
	ExecutablePath   string   `json:"executablePath"`
	ExecutableDigest string   `json:"executableDigest"`
	Required         bool     `json:"required"`
}

type RunReport struct {
	APIVersion    string          `json:"apiVersion"`
	ID            string          `json:"id"`
	PlanID        string          `json:"planId"`
	Status        string          `json:"status"`
	Mode          string          `json:"mode"`
	StartedAt     time.Time       `json:"startedAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
	BaseRevision  string          `json:"baseRevision"`
	BaseSnapshot  string          `json:"baseSnapshot"`
	ModelDigest   string          `json:"modelDigest"`
	RuntimeDigest string          `json:"runtimeDigest"`
	Candidate     CandidateRef    `json:"candidate"`
	Tasks         []ManagerTask   `json:"tasks"`
	Invocations   []InvocationLog `json:"invocations"`
	Checks        []CheckResult   `json:"checks"`
	Findings      []string        `json:"findings,omitempty"`
	Escalations   []Escalation    `json:"escalations,omitempty"`
	Revision      uint64          `json:"revision"`
	Digest        string          `json:"digest"`
}

type CandidateRef struct {
	ID         string            `json:"id"`
	Snapshot   string            `json:"snapshot"`
	Files      map[string]string `json:"files"`
	Parents    []string          `json:"parents,omitempty"`
	Integrated bool              `json:"integrated"`
}

type InvocationLog struct {
	TaskID      string            `json:"taskId"`
	Role        string            `json:"role"`
	Phase       string            `json:"phase"`
	InputDigest string            `json:"inputDigest"`
	Receipt     agentexec.Receipt `json:"receipt"`
	ReportID    string            `json:"reportId"`
	Outcome     string            `json:"outcome"`
	CostMicros  int64             `json:"costMicros"`
}

type CheckResult struct {
	ID               string    `json:"id"`
	Command          []string  `json:"command"`
	ExecutablePath   string    `json:"executablePath"`
	ExecutableDigest string    `json:"executableDigest"`
	CandidateID      string    `json:"candidateId"`
	StartedAt        time.Time `json:"startedAt"`
	Duration         string    `json:"duration"`
	ExitCode         int       `json:"exitCode"`
	Outcome          string    `json:"outcome"`
	Stdout           string    `json:"stdout,omitempty"`
	Stderr           string    `json:"stderr,omitempty"`
	Error            string    `json:"error,omitempty"`
}

type Escalation struct {
	ID            string   `json:"id"`
	FromManager   string   `json:"fromManager"`
	ToManager     string   `json:"toManager"`
	Question      string   `json:"question"`
	AffectedTasks []string `json:"affectedTasks"`
	Status        string   `json:"status"`
}

type StatusReport struct {
	Plan PlanRecord `json:"plan"`
	Run  RunReport  `json:"run"`
}

type VerifyReport struct {
	APIVersion    string          `json:"apiVersion"`
	RunID         string          `json:"runId"`
	CandidateID   string          `json:"candidateId"`
	CandidateHash string          `json:"candidateHash"`
	Status        string          `json:"status"`
	Checks        []CheckResult   `json:"checks"`
	Verifier      *VerifierReport `json:"verifier,omitempty"`
	VerifiedAt    time.Time       `json:"verifiedAt"`
	Digest        string          `json:"digest"`
}

type VerifierReport struct {
	Role         string                  `json:"role"`
	InputDigest  string                  `json:"inputDigest"`
	Receipt      agentexec.Receipt       `json:"receipt"`
	Outcome      string                  `json:"outcome"`
	Observations []agentexec.Observation `json:"observations"`
}

type ApplyRequest struct {
	RunID                      string `json:"runId"`
	PlanID                     string `json:"planId"`
	CandidateID                string `json:"candidateId"`
	ExpectedVerificationDigest string `json:"expectedVerificationDigest"`
	TargetBranch               string `json:"targetBranch"`
	ExpectedHead               string `json:"expectedHead"`
	ExpectedWorktree           string `json:"expectedWorktree"`
}

type ApplyReport struct {
	APIVersion  string    `json:"apiVersion"`
	RunID       string    `json:"runId"`
	CandidateID string    `json:"candidateId"`
	Status      string    `json:"status"`
	Written     []string  `json:"written"`
	Journal     []string  `json:"journal"`
	AppliedAt   time.Time `json:"appliedAt"`
	Error       string    `json:"error,omitempty"`
}

// Host is the function bundle for the selected project frontend. It composes
// directly from projectwork's APIs without introducing a dependency cycle.
type Host struct {
	Load         func(root, revision string) (*Project, error)
	FromSnapshot func(root string, source *Snapshot) (*Project, error)
	PlanEdit     func(project *Project, mutation Mutation) (EditPlan, error)
	ApplyEdit    func(root string, plan EditPlan, expected string) (EditPlan, error)
}

// Invoker preserves the shared request/response/receipt boundary and permits
// protocol fixtures to exercise real subprocess invocations without a model.
type Invoker interface {
	Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error)
	Fingerprint(agentexec.Config) (string, error)
}
