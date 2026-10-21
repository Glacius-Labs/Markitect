package run

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrContextLimit reports a turn whose prompt or Orchestrator answer the
// provider channel cannot carry. Agents return it instead of cutting the
// content; the Orchestrator does not repeat such a turn.
var ErrContextLimit = errors.New("the context exceeds what the provider channel carries")

// Agents runs one Agent turn. A turn runs in a working environment from the
// turn profile's image with Workspace mounted at /work (read-only when
// ReadOnly), the Mounts over it and Home mounted as the provider home: one
// provider conversation turn whose single user message is Prompt, with the
// Orchestrator tools served over MCP. Run returns when the provider finished
// the turn, the Timeout passed or ctx ended. In every case it ends every
// process of the turn before it returns and reports in TurnResult.Stopped
// whether that cessation was observed: of the whole environment, or of every
// process in an environment it keeps. A partial TurnResult is returned with an
// error. The handler answers each tool call; Run calls it serially within one
// turn, while other turns may run concurrently. A provider exit ends a turn;
// it never establishes the completion of any work.
//
// An implementation may keep the environment of a Turn.Environment between
// turns as long as the next turn's capabilities (image, mounts and their
// writability, home, resources) are exactly those it was created with; it
// then also implements Environments. Nothing may run in a kept environment
// between turns.
type Agents interface {
	Run(ctx context.Context, turn Turn, tools ToolHandler) (TurnResult, error)
}

// Environments ends kept working environments. Release ends every instance of
// the named environments, or of all when names is nil, and reports each; the
// Orchestrator calls it once their turns are over, before it removes the
// volumes they mount.
type Environments interface {
	Release(ctx context.Context, names []string) []EnvironmentRelease
}

// EnvironmentRelease is the end of one kept instance: Removed when its end was
// observed.
type EnvironmentRelease struct {
	Environment string
	Instance    string
	Removed     bool
	Error       string
}

// TurnKind separates work of the Execute Passes from independent assessment.
type TurnKind string

const (
	// ExecuteTurn works in a Plan, Prepare or Implement Pass (or an Extract
	// work Pass): it may change files where its mounts allow and may continue
	// the Manager's earlier work conversation.
	ExecuteTurn TurnKind = "execute"
	// VerifyTurn assesses one immutable candidate read-only. Its conversation
	// is fresh for every review Pass and never contains an author's dialogue; it
	// may continue only the same reviewer's conversation within that Pass.
	VerifyTurn TurnKind = "verify"
)

// Turn is everything one Agent turn runs with. ID is unique within a run and
// names the container. Profile names a Runtime Agent profile; the Agents
// implementation resolves model, effort, system prompt, image, credential and
// resource limits from it and chooses nothing itself. Resume names the provider
// conversation the turn continues; empty starts a fresh conversation.
// Transcript, when set, is an absolute path where the implementation writes
// the provider stream with known secrets redacted, as evidence for later
// inspection. Mounts overlay directories or files of the same Workspace volume
// below /work after the base mount (read-only when ReadOnly), in order: each is
// the volume's subpath Path mounted at /work/Path, writable only when Writable
// says so. Environment names the working environment the turn belongs to: the
// Orchestrator names one per Manager's work conversation in the run and one
// per reviewer of a review Pass; empty asks for a fresh environment that ends
// with the turn, such as a helper's.
type Turn struct {
	ID          string
	Environment string
	Manager     string
	Kind        TurnKind
	Profile     string
	Workspace   string
	ReadOnly    bool
	Mounts      []PathMount
	Home        string
	Resume      string
	Prompt      string
	Tools       []ToolSpec
	Timeout     time.Duration
	Transcript  string
}

// ToolSpec is one Orchestrator tool offered to the provider. InputSchema is a
// JSON Schema object.
type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolHandler answers one Orchestrator tool call with the call's JSON
// arguments.
type ToolHandler func(ctx context.Context, name string, arguments json.RawMessage) ToolAnswer

// ToolAnswer is returned to the provider as the tool result. A refused answer
// is reported to the provider as a tool error; it is an answer, not a
// transport failure.
type ToolAnswer struct {
	Text    string
	Refused bool
}

// TurnResult is what the implementation observed. Conversation is the
// provider conversation identity a later Execute turn may resume. Final is the
// provider's final text; it carries no outcome, which arrives only through
// tool calls. Stopped reports the observed end of the container. Compacted
// reports that the provider observably compacted the conversation during the
// turn, so earlier context such as the standing Brief may survive only in its
// summary; a provider that hides compaction never sets it.
// Instance tells which runtime instance the turn ran in.
type TurnResult struct {
	Conversation string
	Final        string
	Usage        Usage
	Started      time.Time
	Ended        time.Time
	Stopped      bool
	Compacted    bool
	Instance     InstanceObservation
	Reason       string
}

// InstanceObservation is the runtime instance of a turn: its name, its
// generation within the environment (1 for the first, counted up whenever a
// new instance replaces or adds to the environment's), whether it existed
// before the turn, and for a new one why it started ("fresh" for a turn
// without an environment, "first", "capabilities" for a capability view the
// environment did not hold, or the reason its predecessor was replaced) and
// how long its start took.
type InstanceObservation struct {
	Name        string `json:"name,omitempty"`
	Generation  int    `json:"generation,omitempty"`
	Reused      bool   `json:"reused,omitempty"`
	Started     string `json:"started,omitempty"`
	StartMillis int64  `json:"startMillis,omitempty"`
}

// Usage is what the provider reported for one turn. CostUSD is the provider's
// own estimate when it reports one, not a billed amount.
type Usage struct {
	Model               string  `json:"model,omitempty"`
	InputTokens         int64   `json:"inputTokens"`
	OutputTokens        int64   `json:"outputTokens"`
	CacheReadTokens     int64   `json:"cacheReadTokens"`
	CacheCreationTokens int64   `json:"cacheCreationTokens"`
	CostUSD             float64 `json:"costUSD"`
	ProviderTurns       int     `json:"providerTurns"`
}

// Add returns the sum of two usages; the model is kept when both agree.
func (u Usage) Add(o Usage) Usage {
	model := u.Model
	if model == "" {
		model = o.Model
	} else if o.Model != "" && o.Model != model {
		model = "mixed"
	}
	return Usage{Model: model, InputTokens: u.InputTokens + o.InputTokens, OutputTokens: u.OutputTokens + o.OutputTokens,
		CacheReadTokens: u.CacheReadTokens + o.CacheReadTokens, CacheCreationTokens: u.CacheCreationTokens + o.CacheCreationTokens,
		CostUSD: u.CostUSD + o.CostUSD, ProviderTurns: u.ProviderTurns + o.ProviderTurns}
}
