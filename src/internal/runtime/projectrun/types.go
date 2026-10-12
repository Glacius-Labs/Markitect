// Package projectrun coordinates bounded project managers over immutable
// snapshots and staged candidates. It owns operational run state, not model
// semantics or provider transport.
package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"context"
	"errors"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectworkspace"
)

const (
	APIVersion = "markitect.example.org/project-run/v1alpha1"
	RunsPath   = ".markitect/runs"

	StatusPlanned          = "planned"
	StatusRunning          = "running"
	StatusAwaitingExternal = "awaiting-external"
	StatusInterrupted      = "interrupted"
	StatusBlocked          = "blocked"
	StatusFailed           = "failed"
	StatusIntegrated       = "integrated"
	StatusVerifying        = "verifying"
	StatusVerified         = "verified"
	StatusApplied          = "applied"
	StatusSuperseded       = "superseded"
)

var (
	ErrStale       = errors.New("project run is stale")
	ErrLocked      = errors.New("project runtime writer lock is held")
	ErrNotFound    = errors.New("project run was not found")
	ErrNotRunnable = errors.New("project run is not runnable")
)

type PlanRequest struct {
	ExplorationID string   `json:"explorationId,omitempty"`
	ScopeID       string   `json:"scopeId,omitempty"`
	Operation     string   `json:"operation,omitempty"`
	Goal          string   `json:"goal"`
	Managers      []string `json:"managers,omitempty"`
	BaseRevision  string   `json:"baseRevision,omitempty"`
	// SinceRevision is an optional older selected-project revision used only to
	// derive change impact. The execution basis remains BaseRevision (or HEAD).
	SinceRevision string                 `json:"sinceRevision,omitempty"`
	ModelEdit     *projectvalue.Mutation `json:"modelEdit,omitempty"`
	// ExecuteAuthorized is set by an explicitly invoked write/run command. It
	// represents the caller's existing authorization, not an agent decision.
	ExecuteAuthorized bool `json:"executeAuthorized"`
	// acceptedBriefings is accepted history a read preview in this package
	// already computed in memory; a preview Plan reads briefings from it.
	acceptedBriefings *projectbriefing.Store
}

type PlanRecord struct {
	ExplorationID           string                                     `json:"explorationId,omitempty"`
	ScopeID                 string                                     `json:"scopeId,omitempty"`
	ExplorationDigest       string                                     `json:"explorationDigest,omitempty"`
	ReadinessDigest         string                                     `json:"readinessDigest,omitempty"`
	ExplorationBinding      *projectvalue.Binding                      `json:"explorationBinding,omitempty"`
	BriefingDigests         map[string]string                          `json:"briefingDigests,omitempty"`
	Strictness              map[string]projectconfig.StrictnessProfile `json:"strictness,omitempty"`
	Operation               string                                     `json:"operation,omitempty"`
	APIVersion              string                                     `json:"apiVersion"`
	ID                      string                                     `json:"id"`
	Status                  string                                     `json:"status"`
	Goal                    string                                     `json:"goal"`
	ExecuteAuthorized       bool                                       `json:"executeAuthorized"`
	Root                    string                                     `json:"-"`
	BaseRevision            string                                     `json:"baseRevision"`
	ChangeBaseRevision      string                                     `json:"changeBaseRevision,omitempty"`
	ChangeBaseSnapshot      string                                     `json:"changeBaseSnapshot,omitempty"`
	ChangeBaseProjectDigest string                                     `json:"changeBaseProjectDigest,omitempty"`
	ChangeBaseModelDigest   string                                     `json:"changeBaseModelDigest,omitempty"`
	ChangeBaseReportDigest  string                                     `json:"changeBaseReportDigest,omitempty"`
	ChangeImpact            *projectmodel.ChangeImpact                 `json:"changeImpact,omitempty"`
	ChangeImpactDigest      string                                     `json:"changeImpactDigest,omitempty"`
	TargetBranch            string                                     `json:"targetBranch"`
	TargetHead              string                                     `json:"targetHead"`
	RepositoryDigest        string                                     `json:"repositoryDigest"`
	BaseSnapshot            string                                     `json:"baseSnapshot"`
	WorkingSnapshot         string                                     `json:"workingSnapshot"`
	BaseProjectDigest       string                                     `json:"baseProjectDigest"`
	WorkingProjectDigest    string                                     `json:"workingProjectDigest"`
	BaseModelDigest         string                                     `json:"baseModelDigest"`
	ModelDigest             string                                     `json:"modelDigest"`
	ReportDigest            string                                     `json:"reportDigest"`
	RuntimeDigest           string                                     `json:"runtimeDigest"`
	RuntimeBindingVersion   int                                        `json:"runtimeBindingVersion,omitempty"`
	PlannedAt               time.Time                                  `json:"plannedAt"`
	Managers                []ManagerTask                              `json:"managers"`
	Checks                  []CheckPlan                                `json:"checks"`
	ModelEdit               *projectvalue.EditPlan                     `json:"modelEdit,omitempty"`
	InitialCandidateID      string                                     `json:"initialCandidateId"`
	RuntimeAgents           map[string]string                          `json:"runtimeAgents"`
	Findings                []string                                   `json:"findings,omitempty"`
	Blockers                []string                                   `json:"blockers,omitempty"`
	Digest                  string                                     `json:"digest"`
}

type ManagerTask struct {
	ID                     string          `json:"id"`
	ManagerID              string          `json:"managerId"`
	ParentTask             string          `json:"parentTask,omitempty"`
	Depth                  int             `json:"depth"`
	Goal                   string          `json:"goal"`
	Owns                   []string        `json:"owns"`
	Statements             []string        `json:"statements"`
	Artifacts              []string        `json:"artifacts"`
	Checks                 []string        `json:"checks"`
	State                  string          `json:"state"`
	ReportStatus           string          `json:"reportStatus"`
	Attempts               int             `json:"attempts,omitempty"`
	WorkAttempts           int             `json:"workAttempts,omitempty"`
	IntegrationAttempts    int             `json:"integrationAttempts,omitempty"`
	RepairPhase            string          `json:"repairPhase,omitempty"`
	RepairDiagnostic       string          `json:"repairDiagnostic,omitempty"`
	ReportID               string          `json:"reportId,omitempty"`
	CandidateID            string          `json:"candidateId,omitempty"`
	IntegrationReportID    string          `json:"integrationReportId,omitempty"`
	IntegrationCandidateID string          `json:"integrationCandidateId,omitempty"`
	WrittenPaths           []string        `json:"writtenPaths,omitempty"`
	IntegratedPaths        []string        `json:"integratedPaths,omitempty"`
	Delegations            []Delegation    `json:"delegations,omitempty"`
	Summary                string          `json:"summary,omitempty"`
	Questions              []string        `json:"questions,omitempty"`
	Risks                  []string        `json:"risks,omitempty"`
	Obligations            []Obligation    `json:"obligations,omitempty"`
	ReviewStatus           string          `json:"reviewStatus,omitempty"`
	ReviewCandidateID      string          `json:"reviewCandidateId,omitempty"`
	ReviewRound            int             `json:"reviewRound,omitempty"`
	ReworkRequests         []ReworkRequest `json:"reworkRequests,omitempty"`
}

type ReworkRequest struct {
	ManagerID string `json:"managerId"`
	Goal      string `json:"goal"`
	Reason    string `json:"reason"`
}

// Obligation records the exact Manager chain for one unresolved question or
// risk. It is Host-authored durable provenance; response text alone is never
// enough to clear a legacy or ambiguously forwarded item.
type Obligation struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Text          string   `json:"text"`
	OriginManager string   `json:"originManager"`
	Chain         []string `json:"chain"`
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
	ExternalExecution       bool                   `json:"externalExecution,omitempty"`
	Operation               string                 `json:"operation,omitempty"`
	APIVersion              string                 `json:"apiVersion"`
	ID                      string                 `json:"id"`
	PlanID                  string                 `json:"planId"`
	Status                  string                 `json:"status"`
	Mode                    string                 `json:"mode"`
	StartedAt               time.Time              `json:"startedAt"`
	UpdatedAt               time.Time              `json:"updatedAt"`
	BaseRevision            string                 `json:"baseRevision"`
	BaseSnapshot            string                 `json:"baseSnapshot"`
	ModelDigest             string                 `json:"modelDigest"`
	RuntimeDigest           string                 `json:"runtimeDigest"`
	Candidate               CandidateRef           `json:"candidate"`
	Tasks                   []ManagerTask          `json:"tasks"`
	Invocations             []InvocationLog        `json:"invocations"`
	RoleStartReservations   []RoleStartReservation `json:"roleStartReservations,omitempty"`
	CostMicros              int64                  `json:"costMicros"`
	CostAccounting          string                 `json:"costAccounting"` // complete, partial, or unknown; cost is an estimate, not an invoice.
	Checks                  []CheckResult          `json:"checks"`
	Findings                []string               `json:"findings,omitempty"`
	Escalations             []Escalation           `json:"escalations,omitempty"`
	RepairRounds            []RepairRound          `json:"repairRounds,omitempty"`
	Reviews                 []ReviewRecord         `json:"reviews,omitempty"`
	ManagerReworkRounds     []ManagerReworkRound   `json:"managerReworkRounds,omitempty"`
	ActiveRepairCandidateID string                 `json:"activeRepairCandidateId,omitempty"`
	Revision                uint64                 `json:"revision"`
	Digest                  string                 `json:"digest"`
}

// RoleStartReservation is Host-owned quota evidence. Helper protocol roots
// attach to the Host request and do not consume another role-start slot.
type RoleStartReservation struct {
	Key               string                     `json:"key"`
	Kind              string                     `json:"kind"` // root or helper
	TaskID            string                     `json:"taskId,omitempty"`
	ManagerID         string                     `json:"managerId,omitempty"`
	Phase             string                     `json:"phase,omitempty"`
	ParentRunID       string                     `json:"parentRunId,omitempty"`
	Request           agentexec.RoleStartRequest `json:"request"`
	ProtocolRequestID string                     `json:"protocolRequestId,omitempty"`
	HelperDelivery    *HelperDelivery            `json:"helperDelivery,omitempty"`
	RecordedAt        time.Time                  `json:"recordedAt"`
}

// HelperDelivery is Host-authored proof that a helper's validated delta was
// applied to its parent candidate and its terminal workspace was closed.
// File contents are intentionally omitted; each write is represented by a
// digest and can be matched against the exact candidate shown to a reviewer.
type HelperDelivery struct {
	State          string                 `json:"state"` // applied-and-closed
	RequestID      string                 `json:"requestId"`
	Task           string                 `json:"task"`
	RequestedPaths []string               `json:"requestedPaths"`
	DeltaDigest    string                 `json:"deltaDigest"`
	Changes        []HelperDeliveryChange `json:"changes"`
}

type HelperDeliveryChange struct {
	Kind          string `json:"kind"`
	Path          string `json:"path"`
	OldPath       string `json:"oldPath,omitempty"`
	Mode          string `json:"mode,omitempty"`
	ContentDigest string `json:"contentDigest,omitempty"`
}

func cloneHelperDelivery(delivery HelperDelivery) HelperDelivery {
	delivery.RequestedPaths = append([]string{}, delivery.RequestedPaths...)
	delivery.Changes = append([]HelperDeliveryChange{}, delivery.Changes...)
	return delivery
}

type ManagerReworkRound struct {
	Number   int                          `json:"number"`
	Status   string                       `json:"status"`
	Requests []ManagerReworkRequestRecord `json:"requests"`
}

type ManagerReworkRequestRecord struct {
	Requester   string        `json:"requester"`
	Request     ReworkRequest `json:"request"`
	Status      string        `json:"status"`
	CandidateID string        `json:"candidateId,omitempty"`
}

// ReviewRecord binds a reviewer decision to the exact immutable candidate
// bytes and accepted scoped model supplied to that invocation.
type ReviewRecord struct {
	TaskID          string            `json:"taskId"`
	ManagerID       string            `json:"managerId"`
	Round           int               `json:"round"`
	Phase           string            `json:"phase"`
	CandidateID     string            `json:"candidateId"`
	CandidateDigest string            `json:"candidateDigest"`
	ScopeDigest     string            `json:"scopeDigest"`
	InputDigest     string            `json:"inputDigest"`
	Outcome         string            `json:"outcome"`
	Findings        []ReviewFinding   `json:"findings"`
	Receipt         agentexec.Receipt `json:"receipt"`
	CostMicros      int64             `json:"costMicros"`
	CostKnown       bool              `json:"costKnown"`
	CostOverflow    bool              `json:"costOverflow,omitempty"`
	At              time.Time         `json:"at"`
}

type ReviewFinding struct {
	Path        string `json:"path"`
	Expectation string `json:"expectation"`
	Grounding   string `json:"grounding"`
}

// RepairRound records an explicit manager loop started from a candidate whose
// required declared checks failed. Prior task and verification evidence remain
// durable here while the active task ledger is reset for this round.
type RepairRound struct {
	Number             int                   `json:"number"`
	Status             string                `json:"status"`
	StartedAt          time.Time             `json:"startedAt"`
	PriorCandidateID   string                `json:"priorCandidateId"`
	SeedCandidateID    string                `json:"seedCandidateId"`
	CandidateID        string                `json:"candidateId,omitempty"`
	VerificationDigest string                `json:"verificationDigest"`
	CheckFeedback      []RepairCheckFeedback `json:"checkFeedback"`
	PriorTasks         []ManagerTask         `json:"priorTasks"`
}

// RepairCheckFeedback is a bounded diagnostic extracted from a failed,
// required check. Output is treated as untrusted data by manager prompts.
type RepairCheckFeedback struct {
	ID       string   `json:"id"`
	Owner    string   `json:"owner"`
	Command  []string `json:"command"`
	ExitCode int      `json:"exitCode"`
	Duration string   `json:"duration"`
	Error    string   `json:"error,omitempty"`
	Stdout   string   `json:"stdout,omitempty"`
	Stderr   string   `json:"stderr,omitempty"`
}

type CandidateRef struct {
	ID         string            `json:"id"`
	Snapshot   string            `json:"snapshot"`
	Files      map[string]string `json:"files"`
	Parents    []string          `json:"parents,omitempty"`
	Integrated bool              `json:"integrated"`
}

type InvocationLog struct {
	TaskID       string            `json:"taskId"`
	Role         string            `json:"role"`
	Phase        string            `json:"phase"`
	InputDigest  string            `json:"inputDigest"`
	Receipt      agentexec.Receipt `json:"receipt"`
	ReportID     string            `json:"reportId"`
	Outcome      string            `json:"outcome"`
	CostMicros   int64             `json:"costMicros"`
	CostKnown    bool              `json:"costKnown"`
	CostOverflow bool              `json:"costOverflow,omitempty"`
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
	ObligationIDs []string `json:"obligationIds,omitempty"`
	Status        string   `json:"status"`
}

type StatusReport struct {
	Plan PlanRecord `json:"plan"`
	Run  RunReport  `json:"run"`
}

type VerifyReport struct {
	VerificationScope   string            `json:"verificationScope,omitempty"`
	ManagerVerification *FullVerifyReport `json:"managerVerification,omitempty"`
	APIVersion          string            `json:"apiVersion"`
	RunID               string            `json:"runId"`
	CandidateID         string            `json:"candidateId"`
	CandidateHash       string            `json:"candidateHash"`
	Status              string            `json:"status"`
	Checks              []CheckResult     `json:"checks"`
	Verifier            *VerifierReport   `json:"verifier,omitempty"`
	VerifiedAt          time.Time         `json:"verifiedAt"`
	Digest              string            `json:"digest"`
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

// ApplyPreflight is a read-only, exact-target contract for a later Apply call.
type ApplyPreflight struct {
	RunID              string   `json:"runId"`
	PlanID             string   `json:"planId"`
	CandidateID        string   `json:"candidateId"`
	VerificationDigest string   `json:"verificationDigest"`
	TargetBranch       string   `json:"targetBranch"`
	ExpectedHead       string   `json:"expectedHead"`
	ExpectedWorktree   string   `json:"expectedWorktree"`
	Paths              []string `json:"paths"`
}

// Host supplies the selected frontend and operational record observations.
// Application composes these operations; Runtime retains each lifecycle gate.
type Host struct {
	// Workspaces owns native Git candidate preparation, observation and cleanup.
	// Unknown execution preserves the original workspace for exact-turn recovery.
	Workspaces               projectworkspace.Service
	Load                     func(root, revision string) (*projectvalue.Project, error)
	FromSnapshot             func(root string, source *snapshot.Snapshot) (*projectvalue.Project, error)
	PlanEdit                 func(project *projectvalue.Project, mutation projectvalue.Mutation) (projectvalue.EditPlan, error)
	ApplyEdit                func(root string, plan projectvalue.EditPlan, expected string) (projectvalue.EditPlan, error)
	ClassifyCandidate        func(base, candidate *projectvalue.Project, deleted ...string) (*projectvalue.Project, error)
	RenderDocument           func(project *projectvalue.Project) (string, error)
	PrepareBriefings         func(root, revision string, persist bool) (*projectbriefing.Store, error)
	BriefingForManager       func(root, revision, modelDigest, managerID string, view *projectbriefing.Store) (BriefingContext, error)
	ResolveVerifiedBriefing  func(root string, evidence DeliveredResolution) error
	LoadDeliveryScope        func(root, explorationID, scopeID string) (DeliveryScope, error)
	LoadExplorationReadiness func(root, explorationID, scopeID string) (DeliveryScope, projectvalue.Binding, ExplorationReadiness, error)
	BindExploration          func(root string, binding projectvalue.Binding, request PlanRequest, plan PlanRecord) (ExplorationAdmission, error)
	ValidateExploration      func(root string, plan PlanRecord) error
	ExplorationCapturePaths  func(plan PlanRecord) ([]string, error)
	CompleteExploration      func(root string, completion ExplorationCompletion) error
}

// Invoker preserves the shared request/response/receipt boundary and permits
// protocol fixtures to exercise real subprocess invocations without a model.
type Invoker interface {
	Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error)
	Fingerprint(agentexec.Config) (string, error)
}
