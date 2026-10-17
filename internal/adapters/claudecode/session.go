package claudecode

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Sessions runs one admitted step per container. Calls for one session must be
// serialized; cancel an in-flight Open/Step before Stop. Entries are retained,
// including failed opens, so retries never implicitly recreate or resend.
type Sessions struct {
	mu          sync.Mutex
	start       RuntimeStarter
	credentials CredentialResolver
	entries     map[run.SessionID]*sessionEntry
}
type sessionEntry struct {
	mu        sync.Mutex
	plan      run.SessionPlan
	profile   SessionProfile
	tools     []frame.ToolSpec
	peer      RuntimeSession
	codec     *Codec
	reader    <-chan protocolRead
	readDone  chan struct{}
	readOnce  sync.Once
	queued    []Message
	decodeErr error
	secrets   [][]byte
	thread    ObservedSessionID
	turn      run.ProviderStepID
	step      run.StepID
	bound     run.AppendReceipt
	observed  run.AppendReceipt
	next      uint64
	stopped   bool
	ready     bool
}
type protocolRead struct {
	data []byte
	err  error
}

func NewSessions(engine *container.Engine, credentials CredentialResolver) (*Sessions, error) {
	if engine == nil {
		return nil, ErrSession
	}
	return NewSessionsWithRuntime(func(ctx context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		return engine.Start(ctx, spec)
	}, credentials)
}

// NewSessionsWithRuntime permits a Host-supplied runtime peer. Production peers
// must preserve Engine.Start and Session's physical-effect/exit semantics.
func NewSessionsWithRuntime(start RuntimeStarter, credentials CredentialResolver) (*Sessions, error) {
	if start == nil || credentials == nil {
		return nil, ErrSession
	}
	return &Sessions{start: start, credentials: credentials, entries: map[run.SessionID]*sessionEntry{}}, nil
}

func (s *Sessions) entry(id run.SessionID) (*sessionEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[id]
	if e == nil {
		return nil, run.ErrObservationLimit
	}
	return e, nil
}
func (e *sessionEntry) meta() run.RecordMeta {
	e.next++
	return run.RecordMeta{ID: fmt.Sprintf("claude-%x-%d", sha256.Sum256([]byte(e.plan.ID)), e.next), ObservedAt: time.Now().UTC()}
}
func (e *sessionEntry) safe(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrSession, e.redact([]byte(err.Error())))
}
func (e *sessionEntry) redact(data []byte) []byte {
	data = bytes.Clone(data)
	for _, secret := range e.secrets {
		data = bytes.ReplaceAll(data, secret, []byte("[REDACTED]"))
		encoded, _ := json.Marshal(string(secret))
		data = bytes.ReplaceAll(data, encoded[1:len(encoded)-1], []byte("[REDACTED]"))
	}
	return data
}
func (e *sessionEntry) wire(raw []byte) run.WirePayload {
	data := e.redact(raw)
	p, _ := run.NewPayload("application/vnd.markitect.claude-wire", "utf-8", data)
	w := run.WirePayload{Identity: run.WireIdentity{ContentID: sha256.Sum256(raw), Size: int64(len(raw))}, Journal: p}
	if !bytes.Equal(raw, data) {
		w.Redacted = true
		w.ObservationLimit = "known runtime credential redacted; journal cannot reproduce wire"
	}
	return w
}

// send does not inspect provider output until the actual write observation has
// been acknowledged. A write error may mean a partial frame: never retry it.
func (e *sessionEntry) send(ctx context.Context, channel run.SessionIO, step run.StepID, call run.ProviderCallID, raw []byte, write func() error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if int64(len(raw)) > e.plan.MaxWireBytes {
		return ErrLimit
	}
	meta := e.meta()
	message := run.MessageID(meta.ID)
	w := e.wire(raw)
	permit, err := channel.PrepareSend(ctx, run.TransportPreparation{RecordMeta: meta, Message: message, Step: step, Call: call, Wire: w})
	if err != nil {
		return err
	}
	if !permit.Dispatch || permit.Receipt.BatchID == "" || permit.Identity != w.Identity {
		return run.ErrObservationLimit
	}
	outcome := run.NotSent
	if ctx.Err() != nil {
		err = ctx.Err()
	} else {
		done := make(chan error, 1)
		go func() { done <- write() }()
		select {
		case err = <-done:
			outcome = run.Sent
			if err != nil {
				outcome = run.SendUncertain
			}
		case <-ctx.Done():
			err = ctx.Err()
			outcome = run.SendUncertain
		}
	}
	reason := "provider write completed"
	if err != nil {
		reason = "provider write not established"
	}
	observed, observeErr := channel.ObserveSend(context.WithoutCancel(ctx), run.TransportObservation{RecordMeta: e.meta(), Message: message, Outcome: outcome, Reason: reason})
	if observeErr == nil && observed.BatchID == "" {
		observeErr = run.ErrObservationLimit
	}
	return errors.Join(e.safe(err), observeErr)
}

func readProtocol(r io.Reader, done <-chan struct{}) <-chan protocolRead {
	ch := make(chan protocolRead)
	go func() {
		defer close(ch)
		// A bounded one-message queue avoids unbounded provider output retention.
		buf := make([]byte, 8192)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				select {
				case ch <- protocolRead{data: bytes.Clone(buf[:n])}:
				case <-done:
					return
				}
			}
			if err != nil {
				select {
				case ch <- protocolRead{err: err}:
				case <-done:
				}
				return
			}
		}
	}()
	return ch
}
func (e *sessionEntry) nextMessage(ctx context.Context) (Message, error) {
	for {
		if len(e.queued) > 0 {
			m := e.queued[0]
			e.queued = e.queued[1:]
			return m, nil
		}
		if e.decodeErr != nil {
			return Message{}, e.decodeErr
		}
		select {
		case <-ctx.Done():
			return Message{}, ctx.Err()
		case r, ok := <-e.reader:
			if !ok {
				return Message{}, io.EOF
			}
			if r.err != nil {
				if err := e.codec.End(); err != nil {
					return Message{raw: e.codec.Pending(), kind: RejectedMessage}, err
				}
				return Message{}, r.err
			}
			messages, err := e.codec.Feed(r.data)
			e.queued = append(e.queued, messages...)
			if err != nil {
				e.decodeErr = err
			}
		}
	}
}
func (s *Sessions) Open(ctx context.Context, a run.SessionAdmission, channel run.SessionIO) (out run.SessionObservation, err error) {
	profile, tools, err := readSessionProfile(a)
	if err != nil || channel == nil {
		return out, ErrSession
	}
	e := &sessionEntry{plan: a.Plan, profile: profile, tools: tools, readDone: make(chan struct{})}
	s.mu.Lock()
	if s.entries[a.Plan.ID] != nil {
		s.mu.Unlock()
		return out, run.ErrObservationLimit
	}
	s.entries[a.Plan.ID] = e
	s.mu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	var transcript []byte
	defer func() {
		if err != nil {
			out = run.SessionObservation{RecordMeta: e.meta(), Uncertain: true, Reason: "Open did not establish acknowledged control readiness"}
			if len(transcript) > 0 {
				out.Wire = e.wire(transcript)
			}
			_, recordErr := channel.Bind(context.WithoutCancel(ctx), out)
			err = errors.Join(e.safe(err), recordErr)
		}
	}()
	credential, err := s.credentials(ctx, a.Plan.CredentialRef)
	if err != nil || credential.Token == "" {
		return out, ErrSession
	}
	e.secrets = [][]byte{[]byte(credential.Token)}
	e.codec, err = NewCodec(int(a.Plan.MaxWireBytes))
	if err != nil {
		return out, err
	}
	e.peer, err = s.start(ctx, sessionSpec(a.Plan, profile))
	if err != nil {
		return out, err
	}
	if e.peer == nil {
		return out, ErrSession
	}
	done := make(chan error, 1)
	go func() {
		_, startErr := e.peer.StartProvider(ctx, frame.Start{Argv: sessionArgv(a.Plan, profile, tools), Dir: "/work", Env: []string{"CLAUDE_CODE_OAUTH_TOKEN=" + credential.Token}, Tools: tools})
		done <- startErr
	}()
	select {
	case err = <-done:
	case <-ctx.Done():
		err = ctx.Err()
	}
	if err != nil {
		return out, err
	}
	e.reader = readProtocol(e.peer.Stdout(), e.readDone)
	id := ControlRequestID("markitect-initialize")
	raw, err := e.codec.EncodeInitialize(id)
	if err != nil {
		return out, err
	}
	if err = e.send(ctx, channel, "", "", raw, func() error { return e.peer.WriteInput(raw) }); err != nil {
		return out, err
	}
	m, err := e.nextMessage(ctx)
	transcript = append(transcript, m.Raw()...)
	if err != nil {
		return out, err
	}
	ack, err := m.ReadInitializeResponse(id)
	if err != nil || ack.Status != ControlSuccess || ack.Version != "" && ack.Version != "2.1.301" {
		return out, ErrProtocol
	}
	out = run.SessionObservation{RecordMeta: e.meta(), Wire: e.wire(transcript), Ready: true, Reason: "Claude control ready; provider session is not yet observed"}
	receipt, err := channel.Ready(ctx, run.SessionReadiness{RecordMeta: out.RecordMeta, Wire: out.Wire, Reason: out.Reason})
	if err == nil && receipt.BatchID == "" {
		err = run.ErrObservationLimit
	}
	e.ready = err == nil
	return out, err
}
func (s *Sessions) Inspect(_ context.Context, request run.InspectionRequest) (run.StepObservation, error) {
	e, err := s.entry(request.Session)
	if err != nil {
		return run.StepObservation{}, err
	}
	if !e.mu.TryLock() {
		return run.StepObservation{}, run.ErrObservationLimit
	}
	defer e.mu.Unlock()
	if e.bound.BatchID == "" || e.observed.BatchID == "" || request.Step == "" || request.Step != e.step || string(request.ProviderSession) != string(e.thread) || string(request.ProviderStep) != string(e.turn) || request.SessionBinding != e.bound || request.StepBinding != e.observed {
		return run.StepObservation{}, run.ErrObservationLimit
	}
	// Memory is no physical/history observation. This method deliberately makes
	// no request, reads no mutable runtime handle and does not replay a turn.
	return run.StepObservation{Step: e.step, ProviderStep: run.ProviderStepID(e.turn), Uncertain: true, Reason: "Claude has no supported read-only exact-step history query in this adapter; adapter handle is not a native turn ID"}, run.ErrObservationLimit
}

func (s *Sessions) Stop(ctx context.Context, request run.StopRequest, channel run.SessionIO) (out run.StopObservation, err error) {
	e, err := s.entry(request.Session)
	if err != nil || channel == nil {
		return out, run.ErrObservationLimit
	}
	if !e.mu.TryLock() {
		return out, run.ErrObservationLimit
	}
	defer e.mu.Unlock()
	if e.stopped || request.ProviderSession != "" && string(request.ProviderSession) != string(e.thread) || request.Timeout <= 0 {
		return out, run.ErrObservationLimit
	}
	out = run.StopObservation{RecordMeta: e.meta(), Reason: "provider exit not established"}
	if e.peer != nil {
		stopCtx, cancel := context.WithTimeout(ctx, request.Timeout)
		defer cancel()
		type stopResult struct {
			exit frame.Exited
			err  error
		}
		done := make(chan stopResult, 1)
		go func() {
			exit, stopErr := e.peer.Stop(stopCtx, request.Timeout.Milliseconds())
			done <- stopResult{exit, stopErr}
		}()
		var exit frame.Exited
		var stopErr error
		select {
		case result := <-done:
			exit = result.exit
			stopErr = result.err
		case <-stopCtx.Done():
			stopErr = stopCtx.Err()
		}
		err = e.safe(stopErr)
		if stopErr == nil {
			out.Terminated = true
			out.Reason = "actual provider exit code=" + strconv.Itoa(exit.Code) + " signal=" + string(e.redact([]byte(exit.Signal)))
		}
		e.readOnce.Do(func() { close(e.readDone) })
		closeErr := e.peer.Close()
		err = errors.Join(err, e.safe(closeErr))
	}
	if !out.Terminated && err == nil {
		err = run.ErrObservationLimit
	}
	observed, recordErr := channel.ObserveStop(context.WithoutCancel(ctx), out)
	if recordErr == nil && observed.BatchID == "" {
		recordErr = run.ErrObservationLimit
	}
	e.stopped = out.Terminated
	return out, errors.Join(err, recordErr)
}

var _ run.Sessions = (*Sessions)(nil)
