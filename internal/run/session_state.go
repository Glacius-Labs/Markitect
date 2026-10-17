package run

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var ErrObservationLimit = errors.New("known journal observations cannot establish or reproduce the requested effect")

type SessionView struct {
	Assignment, Run, Invocation  string
	Plan                         SessionPlan
	Prepared, Binding, Stop      AppendReceipt
	Readiness                    AppendReceipt
	Provider                     ProviderSessionID
	InitialStep                  StepID
	InitialSend                  MessageID
	Ready                        bool
	Bound, Uncertain, Terminated bool
}
type StepView struct {
	Assignment, Run, Invocation  string
	Plan                         StepPlan
	Prepared, Binding            AppendReceipt
	Provider                     ProviderStepID
	Started, Terminal, Uncertain bool
	Proposed                     BlobRef
}
type messageKey struct {
	Session SessionID
	Message MessageID
}
type toolKey struct {
	Session SessionID
	Call    ProviderCallID
}
type TransportView struct {
	Record             SessionRecord
	Prepared, Observed AppendReceipt
	Outcome            SendOutcome
}
type ToolView struct {
	Record             SessionRecord
	Requested, Replied AppendReceipt
	Reply              WireRecord
	OK                 bool
}

func imageIdentity(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") || s != strings.ToLower(s) {
		return false
	}
	_, err := hex.DecodeString(s[7:])
	return err == nil
}
func (w WireRecord) valid() bool {
	if !w.Journal.Valid() || w.Identity.ContentID == (snapshot.ContentID{}) || w.Identity.Size < 0 || w.Identity.Size > maxRecordBytes {
		return false
	}
	if w.Redacted {
		return boundedInput(w.ObservationLimit)
	}
	return w.ObservationLimit == "" && w.Identity.ContentID == w.Journal.ContentID && w.Identity.Size == w.Journal.Size
}
func (r SessionRecord) validate(kind string) error {
	if !identifier(string(r.Session)) {
		return ErrInvalid
	}
	a := SessionRecord{Session: r.Session}
	switch kind {
	case PrepareSession:
		p := r.Plan
		if p.ID != r.Session || (p.Role != "manager" && p.Role != "reviewer") || !identifier(p.Workspace) || !identifier(p.ProviderHome) || p.ProviderHome == p.Workspace || !imageIdentity(p.ImageID) || !identifier(p.CredentialRef) || p.Candidate == (snapshot.TreeID{}) || p.MaxWireBytes <= 0 || p.MaxWireBytes > maxRecordBytes || !p.Profile.Valid() || !p.Tools.Valid() {
			return ErrInvalid
		}
		if p.Resume != "" && !identifier(string(p.Resume)) || p.Role == "reviewer" && (!p.ReadOnly || p.Resume != "") {
			return ErrInvalid
		}
		a.Plan = p
	case PrepareStep:
		p := r.StepPlan
		if p.Session != r.Session || p.ID != r.Step || p.SessionBinding == (snapshot.ContentID{}) || !identifier(string(r.Step)) || !p.Input.Valid() || !p.OutputContract.Valid() {
			return ErrInvalid
		}
		a.Step = r.Step
		a.StepPlan = p
	case BindSession:
		if !boundedInput(r.Reason) || !r.Uncertain && (!identifier(string(r.ProviderSession)) || !r.Wire.valid()) || r.Uncertain && r.ProviderSession != "" && !identifier(string(r.ProviderSession)) {
			return ErrInvalid
		}
		a.ProviderSession = r.ProviderSession
		a.Uncertain = r.Uncertain
		a.Reason = r.Reason
		a.Wire = r.Wire
	case ReadySession:
		if !boundedInput(r.Reason) || !r.Wire.valid() {
			return ErrInvalid
		}
		a.Reason = r.Reason
		a.Wire = r.Wire
	case ObserveStep:
		if r.ProviderStep != "" && !identifier(string(r.ProviderStep)) {
			return ErrInvalid
		}
		if !identifier(string(r.Step)) || !boundedInput(r.Reason) || r.Terminal && r.Uncertain || !r.Uncertain && (!identifier(string(r.ProviderStep)) || !r.Wire.valid()) || r.Proposed != (BlobRef{}) && (!r.Terminal || !r.Proposed.Valid()) {
			return ErrInvalid
		}
		a.Step = r.Step
		a.ProviderStep = r.ProviderStep
		a.Terminal = r.Terminal
		a.Uncertain = r.Uncertain
		a.Reason = r.Reason
		a.Wire = r.Wire
		a.Proposed = r.Proposed
	case PrepareTransport:
		if !identifier(string(r.Message)) || !r.Wire.valid() || r.Step != "" && !identifier(string(r.Step)) || r.Call != "" && (!identifier(string(r.Call)) || r.Step == "" || r.Control) {
			return ErrInvalid
		}
		a.Message = r.Message
		a.Step = r.Step
		a.Call = r.Call
		a.Control = r.Control
		a.Wire = r.Wire
	case ObserveTransport:
		if !identifier(string(r.Message)) || !boundedInput(r.Reason) || r.Outcome != Sent && r.Outcome != NotSent && r.Outcome != SendUncertain {
			return ErrInvalid
		}
		a.Message = r.Message
		a.Outcome = r.Outcome
		a.Reason = r.Reason
	case RequestTool:
		if !identifier(string(r.Step)) || !identifier(string(r.Call)) || !identifier(r.Tool) || !r.Wire.valid() {
			return ErrInvalid
		}
		a.Step = r.Step
		a.Call = r.Call
		a.Tool = r.Tool
		a.Wire = r.Wire
	case ReplyTool:
		if !identifier(string(r.Call)) || !r.Wire.valid() || r.Wire.Redacted || r.Wire.Journal.MediaType != toolOutcomeMedia {
			return ErrInvalid
		}
		a.Call = r.Call
		a.Wire = r.Wire
	case ObserveSessionStop:
		if !boundedInput(r.Reason) {
			return ErrInvalid
		}
		a.Terminated = r.Terminated
		a.Reason = r.Reason
	default:
		return ErrInvalid
	}
	if r.Wire != (WireRecord{}) && !r.Wire.valid() {
		return ErrInvalid
	}
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(r)
	if string(x) != string(y) {
		return ErrInvalid
	}
	return nil
}
func (r SessionRecord) payloadRefs() []BlobRef {
	var refs []BlobRef
	seen := map[BlobRef]bool{}
	for _, ref := range []BlobRef{r.Plan.Profile, r.Plan.Tools, r.StepPlan.Input, r.StepPlan.OutputContract, r.Wire.Journal, r.Proposed} {
		if ref != (BlobRef{}) && !seen[ref] {
			refs = append(refs, ref)
			seen[ref] = true
		}
	}
	return refs
}
func sessionReconciliation(kind string) bool {
	return kind == RequestTool || kind == ReadySession || kind == BindSession || kind == ObserveStep || kind == ObserveTransport || kind == ReplyTool || kind == ObserveSessionStop
}
func (s stateValue) hasLiveSessions() bool {
	for _, v := range s.sessions {
		if !v.Terminated {
			return true
		}
	}
	return false
}
func (s stateValue) transportOpen(id SessionID) bool {
	for key, v := range s.transports {
		if key.Session == id && (v.Outcome == "" || v.Outcome == SendUncertain) {
			return true
		}
	}
	return false
}
func (s stateValue) toolOpen(id SessionID) bool {
	for key, v := range s.tools {
		if key.Session == id && v.Replied.BatchID == "" {
			return true
		}
	}
	return false
}
func (s stateValue) hasSessionActivity() bool {
	for id, v := range s.sessions {
		if !v.Terminated && (v.Uncertain || !v.Bound || s.transportOpen(id)) {
			return true
		}
		if s.toolOpen(id) || s.transportOpen(id) {
			return true
		}
	}
	for _, v := range s.steps {
		if !v.Terminal {
			return true
		}
	}
	return false
}
func (s *stateValue) applySession(c Command) error {
	r := c.Session
	refs := r.payloadRefs()
	provided := map[BlobRef]bool{}
	for _, p := range c.sessionPayloads {
		if !p.Valid() {
			return ErrInvalid
		}
		provided[p.Ref()] = true
	}
	if len(provided) != len(refs) {
		return ErrInvalid
	}
	for _, ref := range refs {
		if !provided[ref] {
			return ErrInvalid
		}
	}
	if c.Kind == PrepareSession {
		if _, exists := s.sessions[r.Session]; exists {
			return ErrConflict
		}
		if r.Plan.Role == "reviewer" && c.Reservation.Purpose != Assessment || r.Plan.Role == "manager" && c.Reservation.Purpose != Writing {
			return ErrInvalid
		}
		if err := s.reserve(c.Reservation); err != nil {
			return err
		}
		s.sessions[r.Session] = SessionView{Assignment: c.Assignment, Run: c.Run, Invocation: c.Reservation.ID, Plan: r.Plan}
		return nil
	}
	v, exists := s.sessions[r.Session]
	if !exists || v.Assignment != c.Assignment || v.Run != c.Run {
		return ErrState
	}
	if r.Wire != (WireRecord{}) && (r.Wire.Identity.Size > v.Plan.MaxWireBytes || r.Wire.Journal.Size > v.Plan.MaxWireBytes) {
		return ErrInvalid
	}
	switch c.Kind {
	case ReadySession:
		if v.Terminated || v.Ready || v.Bound || v.Provider != "" || s.transportOpen(r.Session) {
			return ErrState
		}
		for _, step := range s.steps {
			if step.Plan.Session == r.Session {
				return ErrState
			}
		}
		v.Ready = true
		s.sessions[r.Session] = v
	case PrepareStep:
		if r.StepPlan.SessionBinding != v.Plan.Binding() {
			return ErrState
		}
		if v.Uncertain || v.Terminated || s.transportOpen(r.Session) || s.toolOpen(r.Session) {
			return ErrState
		}
		if !v.Bound && (!v.Ready || v.Readiness.BatchID == "" || v.InitialStep != "" || s.invocations[v.Invocation].State != Settled) {
			return ErrState
		}
		if _, exists := s.steps[r.Step]; exists {
			return ErrConflict
		}
		for _, step := range s.steps {
			if step.Plan.Session == r.Session && !step.Terminal {
				return ErrState
			}
		}
		if v.Plan.Role == "reviewer" && c.Reservation.Purpose != Assessment || v.Plan.Role == "manager" && c.Reservation.Purpose != Writing {
			return ErrInvalid
		}
		if err := s.reserve(c.Reservation); err != nil {
			return err
		}
		if !v.Bound {
			v.InitialStep = r.Step
			s.sessions[r.Session] = v
		}
		s.steps[r.Step] = StepView{Assignment: c.Assignment, Run: c.Run, Invocation: c.Reservation.ID, Plan: r.StepPlan}
	case BindSession:
		if v.Terminated || v.Bound && !v.Uncertain || v.Provider != "" && r.ProviderSession != "" && v.Provider != r.ProviderSession {
			return ErrState
		}
		if !r.Uncertain && s.transportOpen(r.Session) {
			return ErrState
		}
		// A resumed conversation binds only under the identity it was asked to
		// continue; a different provider identity is an uncertain observation.
		if !r.Uncertain && v.Plan.Resume != "" && r.ProviderSession != v.Plan.Resume {
			return ErrState
		}
		if r.ProviderSession != "" {
			v.Provider = r.ProviderSession
		}
		v.Uncertain = r.Uncertain
		v.Bound = !r.Uncertain
		s.sessions[r.Session] = v
	case PrepareTransport:
		if v.Terminated || !r.Control && (s.transportOpen(r.Session) || s.current.State != Running || v.Uncertain) {
			return ErrState
		}
		if v.Ready && !v.Bound {
			if r.Control {
				// Before the actual identity binds, control is termination only;
				// it cannot relabel the first compiled turn or bypass its slot.
				if r.Step != "" || s.current.State != Reconciling {
					return ErrState
				}
			} else {
				if v.InitialStep == "" || r.Step != v.InitialStep || v.InitialSend != "" {
					return ErrState
				}
				v.InitialSend = r.Message
				s.sessions[r.Session] = v
			}
		}
		if r.Step != "" {
			step, ok := s.steps[r.Step]
			if !ok || step.Plan.Session != r.Session || !r.Control && (step.Terminal || step.Uncertain) {
				return ErrState
			}
		} else if v.Bound && !r.Control {
			return ErrState
		}
		if r.Call != "" {
			// An answer follows the recorded Host answer to a call of this step,
			// exactly once.
			tool, ok := s.tools[toolKey{r.Session, r.Call}]
			if !ok || tool.Record.Step != r.Step || tool.Replied.BatchID == "" {
				return ErrState
			}
			for key, m := range s.transports {
				if key.Session == r.Session && m.Record.Call == r.Call {
					return ErrState
				}
			}
		}
		key := messageKey{r.Session, r.Message}
		if _, exists := s.transports[key]; exists {
			return ErrConflict
		}
		s.transports[key] = TransportView{Record: r}
	case ObserveTransport:
		key := messageKey{r.Session, r.Message}
		m, ok := s.transports[key]
		if !ok || m.Outcome == Sent || m.Outcome == NotSent {
			return ErrState
		}
		m.Outcome = r.Outcome
		s.transports[key] = m
	case ObserveStep:
		step, ok := s.steps[r.Step]
		if !ok || step.Plan.Session != r.Session || step.Terminal || step.Provider != "" && r.ProviderStep != "" && step.Provider != r.ProviderStep {
			return ErrState
		}
		if !r.Uncertain && (!v.Bound || s.transportOpen(r.Session) || r.Terminal && s.toolOpen(r.Session)) {
			return ErrState
		}
		if r.ProviderStep != "" {
			step.Provider = r.ProviderStep
		}
		step.Uncertain = r.Uncertain
		step.Terminal = r.Terminal
		step.Started = !r.Uncertain
		step.Proposed = r.Proposed
		s.steps[r.Step] = step
	case RequestTool:
		step, ok := s.steps[r.Step]
		if !ok || step.Plan.Session != r.Session || !step.Started || step.Terminal || step.Uncertain || v.Terminated || v.Uncertain || (s.current.State != Running && (s.basisVersion != 3 || s.current.State != Reconciling)) {
			return ErrState
		}
		key := toolKey{r.Session, r.Call}
		if _, exists := s.tools[key]; exists {
			return ErrConflict
		}
		s.tools[key] = ToolView{Record: r}
	case ReplyTool:
		key := toolKey{r.Session, r.Call}
		tool, ok := s.tools[key]
		if !ok || tool.Replied.BatchID != "" {
			return ErrState
		}
		tool.Reply = r.Wire
		if answer, ok := payloadFor(c, r.Wire.Journal); ok {
			var outcome ToolOutcome
			tool.OK = strictJSON(answer.Bytes(), &outcome) == nil && outcome.Status == ToolOK
		}
		s.tools[key] = tool
	case ObserveSessionStop:
		if v.Terminated {
			return ErrState
		}
		v.Terminated = r.Terminated
		v.Uncertain = !r.Terminated
		s.sessions[r.Session] = v
		if r.Terminated {
			for id, step := range s.steps {
				if step.Plan.Session == r.Session && !step.Terminal {
					step.Terminal = true
					step.Uncertain = false
					s.steps[id] = step
				}
			}
		}
	default:
		return ErrInvalid
	}
	if r.Uncertain || r.Outcome == SendUncertain || c.Kind == ObserveSessionStop && !r.Terminated {
		s.current.State = Reconciling
		s.current.Reason = r.Reason
		s.runs[s.current.ID] = s.current
	}
	return nil
}
func (s *stateValue) sessionReceipt(c Command, receipt AppendReceipt) {
	r := c.Session
	switch c.Kind {
	case PrepareSession, ReadySession, BindSession, ObserveSessionStop:
		v := s.sessions[r.Session]
		if c.Kind == PrepareSession {
			v.Prepared = receipt
		} else if c.Kind == ReadySession {
			v.Readiness = receipt
		} else if c.Kind == BindSession {
			v.Binding = receipt
		} else {
			v.Stop = receipt
		}
		s.sessions[r.Session] = v
	case PrepareStep, ObserveStep:
		v := s.steps[r.Step]
		if c.Kind == PrepareStep {
			v.Prepared = receipt
		} else if r.ProviderStep != "" {
			v.Binding = receipt
		}
		s.steps[r.Step] = v
	case PrepareTransport, ObserveTransport:
		key := messageKey{r.Session, r.Message}
		v := s.transports[key]
		if c.Kind == PrepareTransport {
			v.Prepared = receipt
		} else {
			v.Observed = receipt
		}
		s.transports[key] = v
	case RequestTool, ReplyTool:
		key := toolKey{r.Session, r.Call}
		v := s.tools[key]
		if c.Kind == RequestTool {
			v.Requested = receipt
		} else {
			v.Replied = receipt
		}
		s.tools[key] = v
	}
}

func (s *stateValue) reserve(r Reservation) error {
	if s.current.State != Running {
		return ErrState
	}
	if _, exists := s.invocations[r.ID]; exists {
		return ErrConflict
	}
	for _, prior := range s.history {
		for _, i := range prior.Invocations {
			if i.Reservation.ID == r.ID {
				return ErrConflict
			}
		}
	}
	for _, i := range s.invocations {
		if open(i) && i.Reservation.Purpose == Writing && r.Purpose == Writing {
			return ErrState
		}
	}
	if err := admit(s.policy, budget(s.policy, int64(len(s.runs)), s.invocations), r); err != nil {
		return err
	}
	s.invocations[r.ID] = Invocation{Reservation: r, CapRevision: s.capRevision(), State: Reserved}
	return nil
}

func (s stateValue) sessionSettlementAllowed(invocation, kind string) bool {
	if kind == ObserveUncertain {
		return true
	}
	for id, v := range s.sessions {
		if v.Invocation == invocation {
			if kind == SettleInvocation {
				return v.Bound && !v.Uncertain || v.Ready && !v.Uncertain && s.invocations[invocation].State == Reserved && !s.transportOpen(id) || v.Terminated
			}
			if v.Ready || v.Bound || v.Provider != "" || !v.Terminated {
				return false
			}
			for key, m := range s.transports {
				if key.Session == id && m.Outcome != NotSent {
					return false
				}
			}
		}
	}
	for _, v := range s.steps {
		if v.Invocation == invocation {
			if kind == SettleInvocation {
				return v.Terminal
			}
			if v.Started || v.Provider != "" || !v.Terminal {
				return false
			}
			for key, m := range s.transports {
				if key.Session == v.Plan.Session && m.Record.Step == v.Plan.ID && m.Outcome != NotSent {
					return false
				}
			}
		}
	}
	return true
}
