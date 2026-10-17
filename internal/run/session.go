package run

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type SessionID string
type StepID string
type MessageID string
type ProviderSessionID string
type ProviderStepID string
type ProviderCallID string

// Sessions is implemented by the two container-based provider adapters. In the
// first strategy one Session is one container serving exactly one Step; the
// Host prepares a new Session for every Step. Open starts the container with
// the plan's workspace and provider home and establishes either control
// readiness or the actual provider session identity. When the plan names
// Resume, Open continues that provider conversation from the provider home and
// binds it only if the provider reports the same identity. Step sends the one
// compiled turn. A ready-but-unbound Open permits only the first admitted Step.
// Inspect reads only the exact journal-bound handles; it never creates or sends.
// Stop reports actual termination or uncertainty, never cancellation as success.
// Useful partial observations survive errors. The Host records uncertainty and
// retains admission holds after possible dispatch, callback failure or cancel.
// No method executes a child session inside a tool callback.
type Sessions interface {
	Open(context.Context, SessionAdmission, SessionIO) (SessionObservation, error)
	Step(context.Context, StepAdmission, SessionIO) (StepObservation, error)
	Inspect(context.Context, InspectionRequest) (StepObservation, error)
	Stop(context.Context, StopRequest, SessionIO) (StopObservation, error)
}

// SessionIO is bound by the Host to one confirmed session channel. Every send
// (including tool replies and notifications) requires a fresh acknowledged
// SendPermit matching its real wire identity BEFORE writing. A recorded
// preparation is not delivery or replay authority. Call records a tool call,
// lets the Host answer it and records the answer; the adapter returns that
// answer unchanged. Known secrets must be redacted by the Host before supplying
// journal payloads; RUN receives no secret value or credential bytes.
type SessionIO interface {
	Ready(context.Context, SessionReadiness) (AppendReceipt, error)
	Bind(context.Context, SessionObservation) (AppendReceipt, error)
	PrepareSend(context.Context, TransportPreparation) (SendPermit, error)
	ObserveSend(context.Context, TransportObservation) (AppendReceipt, error)
	Call(context.Context, ToolCall) (ToolResult, error)
	ObserveStep(context.Context, StepObservation) (AppendReceipt, error)
	ObserveStop(context.Context, StopObservation) (AppendReceipt, error)
}

type RecordMeta struct {
	ID         string
	ObservedAt time.Time
}

// Workspace and ProviderHome are opaque runtime-owned Linux volume identities,
// not Host paths. The workspace holds the step's materialized version; the
// provider home holds the provider's conversation state and survives the
// container. Resume names the provider conversation to continue; empty starts
// a fresh one. Reviewer plans are read-only and never resume. The profile
// payload is permitted Host configuration/metadata with credential references
// only, never secret values; the adapter resolves CredentialRef at Open.
type SessionPlan struct {
	ID                                                    SessionID
	Role, Workspace, ProviderHome, ImageID, CredentialRef string
	Resume                                                ProviderSessionID
	Candidate                                             snapshot.TreeID
	ReadOnly                                              bool
	MaxWireBytes                                          int64
	Profile, Tools                                        BlobRef
}
type StepPlan struct {
	Session               SessionID
	ID                    StepID
	SessionBinding        snapshot.ContentID
	Input, OutputContract BlobRef
}

// Binding identifies the fixed profile/image/workspace/candidate plan. A Step
// explicitly asserts this still-current binding; changed or unprovable inputs
// require a new session. The Host must establish physical workspace currentness.
func (p SessionPlan) Binding() snapshot.ContentID {
	data, _ := json.Marshal(p)
	return snapshot.ContentID(sha256.Sum256(data))
}

// SessionAdmission and StepAdmission carry the verified bytes of every payload
// their plans reference. An adapter uses exactly these bytes and resolves only
// the references the plans name; it chooses nothing itself.
type SessionAdmission struct {
	Plan           SessionPlan
	Profile, Tools Payload
	Receipt        AppendReceipt
	Invocation     string
}
type StepAdmission struct {
	Plan                  StepPlan
	Session               SessionPlan
	Input, OutputContract Payload
	Receipt               AppendReceipt
	Invocation            string
}
type InspectionRequest struct {
	Session                     SessionID
	Step                        StepID
	ProviderSession             ProviderSessionID
	ProviderStep                ProviderStepID
	SessionBinding, StepBinding AppendReceipt
}
type StopRequest struct {
	Session         SessionID
	ProviderSession ProviderSessionID
	Timeout         time.Duration
	Reason          string
}

type WireIdentity struct {
	ContentID snapshot.ContentID
	Size      int64
}

// WirePayload separates the real sent/received identity from the permitted
// journal representation. Redacted bytes cannot reproduce the original wire.
// The Host computes the real identity and redacts known secrets before append.
type WirePayload struct {
	Identity         WireIdentity
	Journal          Payload
	Redacted         bool
	ObservationLimit string
}
type WireRecord struct {
	Identity         WireIdentity
	Journal          BlobRef
	Redacted         bool
	ObservationLimit string
}

func (w WirePayload) record() WireRecord {
	return WireRecord{w.Identity, w.Journal.Ref(), w.Redacted, w.ObservationLimit}
}

type SessionObservation struct {
	RecordMeta
	ProviderSession ProviderSessionID
	Wire            WirePayload
	Ready           bool
	Uncertain       bool
	Reason          string
}

// SessionReadiness records the incoming control acknowledgment, not a requested
// UUID or a provider Session ID. It cannot clear an earlier uncertain effect.
type SessionReadiness struct {
	RecordMeta
	Wire   WirePayload
	Reason string
}

// Valid checks an Open observation's exclusive ready, bound or uncertain shape.
// Actual provider protocol interpretation and receipt acknowledgment are separate.
func (o SessionObservation) Valid() bool {
	if !boundedInput(o.Reason) {
		return false
	}
	w := o.Wire.record()
	if o.Ready {
		return !o.Uncertain && o.ProviderSession == "" && w.valid()
	}
	if !o.Uncertain {
		return identifier(string(o.ProviderSession)) && w.valid()
	}
	return (o.ProviderSession == "" || identifier(string(o.ProviderSession))) && (w == (WireRecord{}) || w.valid())
}

type StepObservation struct {
	RecordMeta
	Step                StepID
	ProviderStep        ProviderStepID
	Wire                WirePayload
	Proposed            Payload
	Terminal, Uncertain bool
	Reason              string
}
type StopObservation struct {
	RecordMeta
	Terminated bool
	Reason     string
}
type SendOutcome string

const (
	Sent          SendOutcome = "sent"
	NotSent       SendOutcome = "not_sent"
	SendUncertain SendOutcome = "uncertain"
)

// TransportPreparation is one provider write. Call names the tool call this
// write answers, if any; an answer may follow only a recorded Host answer to
// that call in the same step, and only once.
type TransportPreparation struct {
	RecordMeta
	Message MessageID
	Step    StepID
	Call    ProviderCallID
	Control bool
	Wire    WirePayload
}
type TransportObservation struct {
	RecordMeta
	Message MessageID
	Outcome SendOutcome
	Reason  string
}
type SendPermit struct {
	Receipt  AppendReceipt
	Identity WireIdentity
	Dispatch bool
}

// ToolCall is one provider tool call as the adapter observed it. Tool names one
// entry of the plan's tool catalogue. Input is a JSON object that names the
// tool and carries its arguments as the arguments member, as the agentd frame
// does; CallArguments extracts them. A duplicate call with identical content
// returns the recorded answer; an unanswered duplicate stays pending and is
// never executed again implicitly.
type ToolCall struct {
	RecordMeta
	Step  StepID
	Call  ProviderCallID
	Tool  string
	Input WirePayload
}

// ToolOutcome is the Host's closed answer to one tool call. A refusal is an
// answer that the adapter reports to the provider as a tool error, not a
// transport failure.
type ToolOutcome struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result,omitempty"`
	Reason string          `json:"reason,omitempty"`
}

const (
	ToolOK      = "ok"
	ToolRefused = "refused"

	toolOutcomeMedia = "application/vnd.markitect.tool-outcome+json"
)

// ToolHandler authorizes and executes one Host tool call. The session channel
// records the call before and the outcome after the handler; the handler does
// no recording of its own. Its encoded outcome must fit maxBytes, the
// session's wire limit. An error or an invalid or oversize outcome leaves the
// call pending until reconciled. The channel does not call the handler while
// the run admits no new work; it records a fixed refusal instead, which only
// settles the call in the journal and is never sent, because the Host stops
// the session.
type ToolHandler func(ctx context.Context, call ToolCall, maxBytes int64) (ToolOutcome, error)

// RefuseTools answers every call with a refusal, for slices without Host tools.
func RefuseTools(context.Context, ToolCall, int64) (ToolOutcome, error) {
	return ToolOutcome{Status: ToolRefused, Reason: "no Host tool is available in this slice"}, nil
}

func refuseHalted(context.Context, ToolCall, int64) (ToolOutcome, error) {
	return ToolOutcome{Status: ToolRefused, Reason: "the run admits no new work"}, nil
}

func (o ToolOutcome) payload() (Payload, error) {
	ok := o.Status == ToolOK && o.Reason == "" && (o.Result == nil || json.Valid(o.Result))
	refused := o.Status == ToolRefused && o.Result == nil && o.Reason != "" && boundedInput(o.Reason)
	if !ok && !refused {
		return Payload{}, ErrInvalid
	}
	data, err := json.Marshal(o)
	if err != nil {
		return Payload{}, ErrInvalid
	}
	return NewPayload(toolOutcomeMedia, "utf-8", data)
}

// ToolResult is the answer to one call. Output holds the encoded Outcome that
// the adapter returns to the provider unchanged. When recording the answer
// fails, Call returns the error with Outcome and Output still set but no
// Receipt; the adapter must not send it. After reconciliation shows the answer
// absent, the Host records that Outcome through ToolReconciliation.
type ToolResult struct {
	Receipt AppendReceipt
	Outcome ToolOutcome
	Output  Payload
}

const (
	PrepareSession     = "session.prepared"
	ReadySession       = "session.ready"
	BindSession        = "session.observed"
	PrepareStep        = "step.prepared"
	ObserveStep        = "step.observed"
	PrepareTransport   = "transport.prepared"
	ObserveTransport   = "transport.observed"
	RequestTool        = "tool.requested"
	ReplyTool          = "tool.replied"
	ObserveSessionStop = "session.stop-observed"
)

// SessionRecord is the closed value union for this bounded protocol slice.
// It records no task interpretation, provider schema or runtime-engine type.
type SessionRecord struct {
	Session                                  SessionID
	Step                                     StepID
	Message                                  MessageID
	Call                                     ProviderCallID
	Plan                                     SessionPlan
	StepPlan                                 StepPlan
	ProviderSession                          ProviderSessionID
	ProviderStep                             ProviderStepID
	Wire                                     WireRecord
	Proposed                                 BlobRef
	Tool                                     string
	Outcome                                  SendOutcome
	Control, Terminal, Uncertain, Terminated bool
	Reason                                   string
}

func PrepareSessionCommand(meta RecordMeta, assignment, runID string, plan SessionPlan, reservation Reservation, profile, tools Payload) Command {
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: PrepareSession, Assignment: assignment, Run: runID,
		Reservation: reservation, Session: SessionRecord{Session: plan.ID, Plan: plan}, sessionPayloads: []Payload{profile, tools}}
}
func PrepareStepCommand(meta RecordMeta, assignment, runID string, plan StepPlan, reservation Reservation, input, contract Payload) Command {
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: PrepareStep, Assignment: assignment, Run: runID,
		Reservation: reservation, Session: SessionRecord{Session: plan.Session, Step: plan.ID, StepPlan: plan}, sessionPayloads: []Payload{input, contract}}
}
