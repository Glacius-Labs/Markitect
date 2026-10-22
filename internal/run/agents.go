package run

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ErrContextLimit reports a turn whose prompt or Orchestrator answer the
// provider channel cannot carry. Agents return it instead of cutting the
// content; the Orchestrator does not repeat such a turn.
var ErrContextLimit = errors.New("the context exceeds what the provider channel carries")

var assignmentName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,89}$`)

// AssignmentSpec names one Agent's long-lived runtime. The caller supplies a
// fresh workspace and home for every assignment, including independent Verify.
// Provider responses do not replace this identity or close its container.
type AssignmentSpec struct {
	ID         string
	Profile    string
	Workspace  string
	Home       string
	Tools      []ToolSpec
	Transcript string
}

// AssignmentSession keeps one provider conversation in one container. Message
// returns a native provider observation; it does not decide whether the work
// should be accepted. Close ends the container even after cancellation.
type AssignmentSession interface {
	Message(context.Context, string, time.Duration) (TurnResult, error)
	Close(context.Context) (TurnResult, error)
}

// AssignmentAgents starts one independent Agent runtime. Legacy Agents.Run
// remains for the implemented Pass path until its consumers migrate.
type AssignmentAgents interface {
	OpenAssignment(context.Context, AssignmentSpec, ToolHandler) (AssignmentSession, error)
}

// AssignmentQuestion is the sole special communication signal in this slice.
// A parent can answer directly or arrange a later reply through its caller.
// The callback should honor ctx so an aborted assignment leaves no waiter.
type AssignmentQuestion func(context.Context, string) (string, error)

const AskParentTool = "ask_parent"

var askParentSpec = ToolSpec{
	Name: AskParentTool, Description: "Ask the parent Manager a question about this assignment.",
	InputSchema: json.RawMessage(`{"type":"object","properties":{"question":{"type":"string"}},"required":["question"],"additionalProperties":false}`),
}

// AssignmentRequest has no report format or completion phrase. The Host fixes
// a result commit when Submit is called; the Agent's last native final message
// is the summary that travels with that result.
type AssignmentRequest struct {
	ID         WorkspaceID
	Profile    string
	Tree       snapshot.Tree
	Home       string
	Transcript string
	Tools      []ToolSpec
	Handler    ToolHandler
	AskParent  AssignmentQuestion
}

type AssignmentHandoff struct {
	StartingCommit string
	ResultCommit   string
	Summary        string
}

// Assignment owns a running Agent and its independent Git copy. The caller
// closes the Agent when accepted or aborted and separately releases retained
// workspace data when it is no longer needed.
type Assignment struct {
	mu        sync.Mutex
	session   AssignmentSession
	work      AssignmentWorkspace
	last      TurnResult
	ready     bool
	active    bool
	closed    bool
	closing   TurnResult
	closeErr  error
	closeDone chan struct{}
}

func StartAssignment(ctx context.Context, workspaces AssignmentWorkspaces, agents AssignmentAgents, request AssignmentRequest) (*Assignment, error) {
	if workspaces == nil || agents == nil || !assignmentName.MatchString(string(request.ID)) || request.Profile == "" || request.Home == "" {
		return nil, errors.New("invalid Agent assignment")
	}
	tools := append([]ToolSpec(nil), request.Tools...)
	handler := request.Handler
	if handler == nil {
		handler = func(context.Context, string, json.RawMessage) ToolAnswer {
			return ToolAnswer{Text: "unknown assignment tool", Refused: true}
		}
	}
	if request.AskParent != nil {
		for _, tool := range tools {
			if tool.Name == AskParentTool {
				return nil, errors.New("ask_parent is reserved for assignment questions")
			}
		}
		tools = append(tools, askParentSpec)
		handler = func(ctx context.Context, name string, args json.RawMessage) ToolAnswer {
			if name != AskParentTool {
				if request.Handler == nil {
					return ToolAnswer{Text: "unknown assignment tool", Refused: true}
				}
				return request.Handler(ctx, name, args)
			}
			var question struct {
				Question string `json:"question"`
			}
			if err := json.Unmarshal(args, &question); err != nil || question.Question == "" {
				return ToolAnswer{Text: "a non-empty question is required", Refused: true}
			}
			answer, err := request.AskParent(ctx, question.Question)
			if err != nil {
				return ToolAnswer{Text: err.Error(), Refused: true}
			}
			return ToolAnswer{Text: answer}
		}
	}
	work, err := workspaces.PrepareAssignment(ctx, WorkspaceSpec{ID: request.ID, Tree: request.Tree})
	if err != nil {
		if work != nil {
			return &Assignment{work: work}, err
		}
		return nil, err
	}
	if work == nil {
		return nil, errors.New("assignment workspace was not established")
	}
	session, err := agents.OpenAssignment(ctx, AssignmentSpec{ID: string(request.ID), Profile: request.Profile, Workspace: work.Volume(), Home: request.Home, Tools: tools, Transcript: request.Transcript}, handler)
	if err != nil {
		return &Assignment{work: work}, err
	}
	return &Assignment{session: session, work: work}, nil
}

func (a *Assignment) Message(ctx context.Context, text string, timeout time.Duration) (TurnResult, error) {
	if a == nil || a.session == nil {
		return TurnResult{}, errors.New("Agent assignment is closed")
	}
	a.mu.Lock()
	if a.closed || a.active {
		a.mu.Unlock()
		return TurnResult{}, errors.New("Agent assignment is closed or busy")
	}
	a.active = true
	a.mu.Unlock()
	result, err := a.session.Message(ctx, text, timeout)
	a.mu.Lock()
	a.active = false
	a.last, a.ready = result, err == nil && result.Completed
	a.mu.Unlock()
	return result, err
}

// Submit fixes the current Git contents without freezing the working copy.
// It also works after a normal Close, using the last completed message as the
// summary. Corrections can produce another commit in the same assignment.
func (a *Assignment) Submit(ctx context.Context) (AssignmentHandoff, error) {
	if a == nil || a.work == nil {
		return AssignmentHandoff{}, errors.New("no completed Agent response to submit")
	}
	a.mu.Lock()
	if a.active || !a.ready || a.closed && (a.closeErr != nil || !a.closing.Stopped) {
		a.mu.Unlock()
		return AssignmentHandoff{}, errors.New("no completed Agent response to submit")
	}
	a.active = true
	summary := a.last.Final
	a.mu.Unlock()
	commit, err := a.work.Commit(ctx, "Agent assignment result")
	a.mu.Lock()
	a.active = false
	if err == nil {
		a.ready = false
	}
	a.mu.Unlock()
	if err != nil {
		return AssignmentHandoff{}, err
	}
	return AssignmentHandoff{StartingCommit: a.work.StartingCommit(), ResultCommit: commit, Summary: summary}, nil
}

// Close ends the container. The Git copy remains retained, including on error
// or cancellation, so an incomplete assignment stays inspectable.
func (a *Assignment) Close(ctx context.Context) (TurnResult, error) {
	if a == nil || a.session == nil {
		return TurnResult{}, nil
	}
	a.mu.Lock()
	if a.closed {
		done := a.closeDone
		a.mu.Unlock()
		if done != nil {
			<-done
		}
		return a.closing, a.closeErr
	}
	a.closed = true
	a.closeDone = make(chan struct{})
	a.mu.Unlock()
	closing, closeErr := a.session.Close(ctx)
	a.mu.Lock()
	a.closing, a.closeErr = closing, closeErr
	if closeErr != nil || !closing.Stopped {
		a.ready = false
	}
	close(a.closeDone)
	a.mu.Unlock()
	return closing, closeErr
}

func (a *Assignment) Workspace() AssignmentWorkspace {
	if a == nil {
		return nil
	}
	return a.work
}

// Agents runs one Agent turn. A turn is one fresh container from the turn
// profile's image with Workspace mounted at /work (read-only when ReadOnly)
// and Home mounted as the provider home, one provider conversation turn whose
// single user message is Prompt, and the Orchestrator tools served over MCP.
// Run returns when the provider finished the turn, the Timeout passed or ctx
// ended. In every case it stops the container before it returns and reports
// in TurnResult.Stopped whether the end of the container, and with it of every
// process the turn started, was observed. A partial TurnResult is returned
// with an error. The handler answers each tool call; Run calls it serially
// within one turn, while other turns may run concurrently. A provider exit
// ends a turn; it never establishes the completion of any work.
type Agents interface {
	Run(ctx context.Context, turn Turn, tools ToolHandler) (TurnResult, error)
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
// says so.
type Turn struct {
	ID         string
	Manager    string
	Kind       TurnKind
	Profile    string
	Workspace  string
	ReadOnly   bool
	Mounts     []PathMount
	Home       string
	Resume     string
	Prompt     string
	Tools      []ToolSpec
	Timeout    time.Duration
	Transcript string
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
// provider's final-answer assessment; the Host captures work at the observed
// native end and validates its meaning separately. Stopped reports the observed
// end of the container. Compacted
// reports that the provider observably compacted the conversation during the
// turn, so earlier context such as the standing Brief may survive only in its
// summary; a provider that hides compaction never sets it.
type TurnResult struct {
	Conversation string
	Final        string
	// Completed is a successful native provider end event, independently of
	// transport closure and observed container stop. Final is final-answer text,
	// never commentary. Neither field establishes semantic success.
	Completed bool
	Commands  []CommandObservation
	Usage     Usage
	Started   time.Time
	Ended     time.Time
	Stopped   bool
	Compacted bool
	Reason    string
}

// CommandObservation is an actual completed provider command item. It is not
// a candidate-bound conformance receipt and says nothing about relevance.
type CommandObservation struct {
	ID       string `json:"id"`
	Command  string `json:"command"`
	Status   string `json:"status"`
	ExitCode *int   `json:"exitCode,omitempty"`
	Output   string `json:"output,omitempty"`
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
