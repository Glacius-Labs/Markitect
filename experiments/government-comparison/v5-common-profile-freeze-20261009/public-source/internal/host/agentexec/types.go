package agentexec

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const (
	APIVersion        = "markitect.example.org/agent-execution/v1alpha1"
	RoleExecutor      = "executor"
	RoleVerifier      = "verifier"
	RoleInfer         = "infer"
	OutcomeProposed   = "proposed"
	OutcomePassed     = "passed"
	OutcomeFailed     = "failed"
	OutcomeIncomplete = "incomplete"
	OutcomeEscalated  = "escalated"
	// MaxVerifierObservations is the existing per-response wire limit used by
	// the Host to bound required observation coverage before invocation.
	MaxVerifierObservations = maxScopeCount
)

var (
	ErrInputChanged   = errors.New("agent execution input changed during invocation")
	ErrOutputTooLarge = errors.New("agent execution output exceeded its configured bound")
)

type Artifact struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Digest  string `json:"digest"`
	Content []byte `json:"content"`
}

type Request struct {
	Role           string          `json:"role"`
	SourceRevision string          `json:"sourceRevision"`
	ModelDigest    string          `json:"modelDigest"`
	ModulePin      string          `json:"modulePin"`
	ProjectionID   string          `json:"projectionId"`
	ScopeIDs       []string        `json:"scopeIds"`
	PolicyIDs      []string        `json:"policyIds"`
	Context        json.RawMessage `json:"context"`
	Artifacts      []Artifact      `json:"artifacts"`
}

type Invocation struct {
	APIVersion  string  `json:"apiVersion"`
	RunID       string  `json:"runId"`
	Nonce       string  `json:"nonce"`
	InputDigest string  `json:"inputDigest"`
	Request     Request `json:"request"`
}

type CandidateFile struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Content string `json:"content"`
}

type Observation struct {
	Subject string `json:"subject"`
	Outcome string `json:"outcome"`
	Detail  string `json:"detail"`
}

type Usage struct {
	Source       string `json:"source"`
	InputTokens  *int64 `json:"inputTokens,omitempty"`
	OutputTokens *int64 `json:"outputTokens,omitempty"`
	CachedTokens *int64 `json:"cachedTokens,omitempty"`
	ToolCalls    *int64 `json:"toolCalls,omitempty"`
}

type Response struct {
	APIVersion           string          `json:"apiVersion"`
	RunID                string          `json:"runId"`
	Nonce                string          `json:"nonce"`
	Role                 string          `json:"role"`
	InputDigest          string          `json:"inputDigest"`
	Outcome              string          `json:"outcome"`
	CandidateFiles       []CandidateFile `json:"candidateFiles"`
	EvidenceRefs         []string        `json:"evidenceRefs"`
	VerifierObservations []Observation   `json:"verifierObservations"`
	CandidateJSON        json.RawMessage `json:"candidateJson,omitempty"`
	ReportJSON           json.RawMessage `json:"reportJson,omitempty"`
	Uncertainty          []string        `json:"uncertainty"`
	Usage                *Usage          `json:"usage,omitempty"`
}

type RuntimeFile struct {
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Digest string `json:"digest"`
}

type Config struct {
	Command         string          `json:"command"`
	Args            []string        `json:"args"`
	Model           string          `json:"model"`
	ModelOptions    json.RawMessage `json:"modelOptions"`
	ProviderVersion string          `json:"providerVersion"`
	// EnvironmentAllowlist selects caller environment names passed to the child.
	// nil preserves the historical inherit-all behavior; a non-nil empty slice
	// passes no caller variables. Internal MARKITECT_AGENT_* values are added by
	// agentexec after this selection.
	EnvironmentAllowlist *[]string     `json:"environmentAllowlist,omitempty"`
	Timeout              time.Duration `json:"timeout"`
	MaxStdoutBytes       int           `json:"maxStdoutBytes"`
	MaxStderrBytes       int           `json:"maxStderrBytes"`
	RuntimeFiles         []RuntimeFile `json:"runtimeFiles"`
}

type RunOptions struct {
	InputRoots          []string
	TempParent          string
	PrivateLogDirectory string
}

type Receipt struct {
	APIVersion            string `json:"apiVersion"`
	RunID                 string `json:"runId"`
	InputDigest           string `json:"inputDigest"`
	ContextDigest         string `json:"contextDigest"`
	ConfigDigest          string `json:"configDigest"`
	CommandDigest         string `json:"commandDigest"`
	ExecutableDigest      string `json:"executableDigest"`
	RuntimeFilesDigest    string `json:"runtimeFilesDigest"`
	EnvironmentDigest     string `json:"environmentDigest"`
	ProviderVersion       string `json:"providerVersion"`
	ProviderVersionDigest string `json:"providerVersionDigest"`
	StdoutDigest          string `json:"stdoutDigest"`
	StderrDigest          string `json:"stderrDigest"`
	PrivateLogDigest      string `json:"privateLogDigest"`
	Outcome               string `json:"outcome"`
	WallTimeMilliseconds  int64  `json:"wallTimeMilliseconds"`
	RetryCount            int    `json:"retryCount"`
	Usage                 *Usage `json:"usage,omitempty"`
}

type RunResult struct {
	Response Response
	Receipt  Receipt
}

// Run starts one new external process in an empty temporary directory. InputRoots
// optionally selects read-only snapshots to compare before and after execution;
// empty InputRoots does not acquire or audit a repository tree. The process receives
// the serialized request on stdin and declared command arguments, with no automatic
// workspace writes.
func Run(ctx context.Context, cfg Config, req Request, opts RunOptions) (RunResult, error) {
	return run(ctx, cfg, req, opts)
}
