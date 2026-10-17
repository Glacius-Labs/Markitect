package run

import (
	"crypto/sha256"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type managerValue struct {
	Slot                                                 organization.Manager
	HandoffID                                            string
	Session                                              SessionID
	Step                                                 StepID
	Packet                                               ExplorePacket
	Input                                                snapshot.ContentID
	Queued, Prepared, Sent, StepObserved, ReportObserved AppendReceipt
	Report                                               ExploreReport
	ReportProblem                                        string
}

// InitialManagerExplore distinguishes logical enqueue, transport-write and
// attributed step/report observations. None proves real provider delivery,
// semantic acceptance, completed realization or permission to write.
type InitialManagerExplore struct {
	Manager, Parent, HandoffID                                   string
	Session                                                      SessionID
	Step                                                         StepID
	Queued, Prepared, SentObserved, StepObserved, ReportObserved AppendReceipt
	Report                                                       *ExploreReport
	ReportProblem                                                string
	Quiescent                                                    bool
}
type InitialExploreView struct {
	Available, ReportsComplete, Clear bool
	Managers                          []InitialManagerExplore
}

func (s stateValue) managerForSession(id SessionID) (managerValue, bool) {
	for _, v := range s.managers {
		if v.Session == id && id != "" {
			return v, true
		}
	}
	return managerValue{}, false
}
func (s stateValue) managerQuiescent(v managerValue) bool {
	if v.Session == "" || v.ReportObserved.BatchID == "" {
		return false
	}
	session, ok := s.sessions[v.Session]
	if !ok || !session.Terminated || session.Uncertain || s.transportOpen(v.Session) || s.toolOpen(v.Session) {
		return false
	}
	for _, i := range s.invocations {
		if open(i) {
			return false
		}
	}
	return true
}
func (c *Coordinator) InitialExplore() InitialExploreView {
	out := InitialExploreView{Available: c.state.basisVersion == 3}
	if !out.Available {
		return out
	}
	out.ReportsComplete = true
	out.Clear = true
	for _, slot := range c.state.managerBasis.Managers {
		v := c.state.managers[slot.Package]
		row := InitialManagerExplore{Manager: slot.Package, Parent: slot.Parent, HandoffID: v.HandoffID, Session: v.Session, Step: v.Step, Queued: v.Queued, Prepared: v.Prepared, SentObserved: v.Sent, StepObserved: v.StepObserved, ReportObserved: v.ReportObserved, ReportProblem: v.ReportProblem, Quiescent: c.state.managerQuiescent(v)}
		if v.ReportObserved.BatchID != "" {
			r := copyReport(v.Report)
			row.Report = &r
		} else {
			out.ReportsComplete = false
		}
		if !row.Quiescent {
			out.Clear = false
		}
		out.Managers = append(out.Managers, row)
	}
	return out
}
func (s *stateValue) prepareManagers(b Binding) error {
	s.managers = nil
	s.managerBasis = bindingWire{}
	if b.version != 3 {
		return nil
	}
	w, _, err := decodeBindingRecord(b.payload)
	if err != nil {
		return err
	}
	s.managerBasis = w
	s.managers = map[string]managerValue{}
	for _, m := range w.Managers {
		s.managers[m.Package] = managerValue{Slot: m, HandoffID: fmt.Sprintf("explore-%x", sha256.Sum256([]byte(b.Ref().ContentID.String()+":"+m.Package)))}
	}
	return nil
}

// Historical replay uses the same reducer, but never supplies a fresh execution
// permit. Exact recorded retries are looked up before this fresh-admission gate.
func (c *Coordinator) freshManagerAdmission(cmd Command, digest snapshot.ContentID) error {
	if cmd.Kind == PrepareRun {
		if cmd.binding.version != 3 {
			return ErrState
		}
		return nil
	}
	switch cmd.Kind {
	case ReserveInvocation:
		return ErrState // No unscoped executable consumer exists.
	case PrepareSession, PrepareStep:
		if c.state.basisVersion != 3 || cmd.Manager == (ManagerRecord{}) || cmd.managerProof != digest {
			return ErrState
		}
	case PrepareTransport:
		if c.state.basisVersion != 3 && (!cmd.Session.Control || c.Current().State != Reconciling) {
			return ErrState
		}
	}
	return nil
}
func (s *stateValue) beforeManagerSession(c Command) error {
	if s.basisVersion != 3 {
		if c.Manager != (ManagerRecord{}) {
			return ErrInvalid
		}
		return nil
	}
	r := c.Session
	if c.Kind == PrepareSession {
		v, ok := s.managers[c.Manager.Manager]
		if !ok || !sameManager(recordFor(v), c.Manager) || v.Session != "" || v.Queued.BatchID == "" || r.Plan.Role != "manager" || !r.Plan.ReadOnly || r.Plan.Candidate != basisTreeID(s.managerBasis) || c.Reservation.Purpose != Writing {
			return ErrState
		}
		if s.hasLiveSessions() || s.hasOpen() {
			return ErrState
		}
		if v.Slot.Parent != "" {
			parent := s.managers[v.Slot.Parent]
			if parent.StepObserved.BatchID == "" {
				return ErrState
			}
		}
		v.Session = r.Session
		s.managers[v.Slot.Package] = v
		return nil
	}
	v, ok := s.managerForSession(r.Session)
	if !ok {
		return ErrState
	}
	switch c.Kind {
	case PrepareStep:
		if !sameManager(recordFor(v), c.Manager) || v.Step != "" || c.Reservation.Purpose != Writing {
			return ErrState
		}
		p, ok := payloadFor(c, r.StepPlan.Input)
		if !ok {
			return ErrInvalid
		}
		packet, err := validateExplorePacket(p, c, v, *s)
		if err != nil {
			return err
		}
		v.Step = r.Step
		v.Packet = packet
		v.Input = p.Ref().ContentID
		s.managers[v.Slot.Package] = v
	case PrepareTransport:
		if r.Control {
			if s.current.State != Reconciling {
				return ErrState
			}
			return nil
		}
		session := s.sessions[r.Session]
		if r.Step == "" {
			if session.Ready || session.Bound {
				return ErrState
			}
			for key, m := range s.transports {
				if key.Session == r.Session && !m.Record.Control {
					return ErrState
				}
			}
			return nil
		}
		if r.Step != v.Step {
			return ErrState
		}
		sentBefore := false
		for key, m := range s.transports {
			if key.Session == r.Session && m.Record.Step == r.Step && !m.Record.Control {
				sentBefore = true
			}
		}
		if sentBefore && r.Call == "" {
			// After the turn input, a send may only answer a recorded tool call;
			// it creates no execution authority and cannot resend the initial turn.
			return ErrState
		}
	case ObserveStep:
		if r.Step != v.Step {
			return ErrState
		}
		if !r.Uncertain {
			sent := false
			for key, m := range s.transports {
				if key.Session == r.Session && m.Record.Step == r.Step && !m.Record.Control && m.Outcome == Sent {
					sent = true
				}
			}
			if !sent {
				return ErrState
			}
		}
	}
	return nil
}
func basisTreeID(w bindingWire) snapshot.TreeID {
	entries := make([]snapshot.Entry, 0, len(w.Census))
	for _, f := range w.Census {
		p, e := snapshot.NewPath(f.Path)
		if e != nil {
			return snapshot.TreeID{}
		}
		entry, e := snapshot.NewEntry(p, f.Mode, f.Bytes)
		if e != nil {
			return snapshot.TreeID{}
		}
		entries = append(entries, entry)
	}
	tree, e := snapshot.NewTree(entries)
	if e != nil {
		return snapshot.TreeID{}
	}
	return tree.Digest()
}
func (s *stateValue) afterManagerSession(c Command) error {
	if s.basisVersion != 3 || c.Kind != ObserveStep || !c.Session.Terminal || c.Session.Uncertain {
		return nil
	}
	v, ok := s.managerForSession(c.Session.Session)
	if !ok {
		return ErrState
	}
	p, ok := payloadFor(c, c.Session.Proposed)
	if !ok {
		v.ReportProblem = "terminal step supplied no initial Explore report"
	} else {
		report, err := decodeExploreReport(p, v.Packet, v.Input)
		if err != nil {
			v.ReportProblem = "terminal proposal does not match the closed initial Explore report and admitted input"
		} else {
			v.Report = report
			v.ReportProblem = ""
		}
	}
	s.managers[v.Slot.Package] = v
	return nil
}
func (s *stateValue) managerReceipt(c Command, receipt AppendReceipt) {
	if s.basisVersion != 3 {
		return
	}
	if c.Kind == PrepareRun {
		for name, v := range s.managers {
			v.Queued = receipt
			s.managers[name] = v
		}
		return
	}
	v, ok := s.managerForSession(c.Session.Session)
	if !ok {
		return
	}
	switch c.Kind {
	case PrepareSession:
		v.Prepared = receipt
	case ObserveTransport:
		m := s.transports[messageKey{c.Session.Session, c.Session.Message}]
		if m.Record.Step == v.Step && v.Step != "" && !m.Record.Control && c.Session.Outcome == Sent && v.Sent.BatchID == "" {
			v.Sent = receipt
		}
	case ObserveStep:
		if !c.Session.Uncertain {
			v.StepObserved = receipt
		}
		if c.Session.Terminal && !c.Session.Uncertain && v.ReportProblem == "" && v.Report.Version == 1 {
			v.ReportObserved = receipt
		}
	}
	s.managers[v.Slot.Package] = v
}
