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

// Sessions is implemented by the two container-based provider adapters. Open
// starts one container and provider session; Step sends one compiled turn.
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
// (including responses/notifications) requires a fresh acknowledged SendPermit
// matching its real wire identity BEFORE writing. A recorded preparation is not
// delivery or replay authority. Known secrets must be redacted by the Host before
// supplying journal payloads; RUN receives no secret value or credential bytes.
type SessionIO interface {
	Bind(context.Context, SessionObservation) (AppendReceipt, error)
	PrepareSend(context.Context, TransportPreparation) (SendPermit, error)
	ObserveSend(context.Context, TransportObservation) (AppendReceipt, error)
	ToolRequest(context.Context, ToolCall) (ToolReceipt, error)
	ToolReply(context.Context, ToolReply) (AppendReceipt, error)
	ObserveStep(context.Context, StepObservation) (AppendReceipt, error)
	ObserveStop(context.Context, StopObservation) (AppendReceipt, error)
}

type RecordMeta struct {
	ID         string
	ObservedAt time.Time
}

// Workspace is an opaque runtime-owned Linux workspace identity, not a Host
// path. Reviewer plans must be read-only. The profile payload is permitted Host
// configuration/metadata with credential references only, never secret values.
type SessionPlan struct {
	ID                                      SessionID
	Role, Workspace, ImageID, CredentialRef string
	Candidate                               snapshot.TreeID
	ReadOnly                                bool
	MaxWireBytes                            int64
	Profile, Tools                          BlobRef
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

type SessionAdmission struct {
	Plan       SessionPlan
	Receipt    AppendReceipt
	Invocation string
}
type StepAdmission struct {
	Plan       StepPlan
	Session    SessionPlan
	Receipt    AppendReceipt
	Invocation string
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
	Uncertain       bool
	Reason          string
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

type TransportPreparation struct {
	RecordMeta
	Message MessageID
	Step    StepID
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

// Tools are recorded ingress/reply exchanges only. Tool identifies one supplied
// Host schema; no tool is executed here and Fresh does not authorize an effect.
// A duplicate unresolved call remains pending, never executed again implicitly.
type ToolCall struct {
	RecordMeta
	Step  StepID
	Call  ProviderCallID
	Tool  string
	Input WirePayload
}
type ToolReply struct {
	RecordMeta
	Call   ProviderCallID
	Output WirePayload
}
type ToolReceipt struct {
	Receipt        AppendReceipt
	Fresh, Replied bool
	Reply          WireRecord
	Payload        Payload
}

const (
	PrepareSession     = "session.prepared"
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
