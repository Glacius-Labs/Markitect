// Package authoring turns the owner's changes to the project's .markitect
// inputs into proposals: a changed stand of the committed source, checked like
// the repository itself, shown as its difference and written only when the
// owner accepts it, through Apply's guarded publication with read-back. It
// never commits and never starts an Apply.
package authoring

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Capturer captures the repository's committed source as the structural check
// does, with the guard a later publication is checked against.
type Capturer func(context.Context) (apply.CapturedPreparation, error)

// Locker fences a publication against every Apply: it takes the state locks
// and returns the control check publication runs under and its release.
type Locker func(context.Context) (apply.ControlCheck, func(), error)

// Service holds the proposals of one cockpit. Proposals live as long as the
// cockpit process; what was accepted is in the working tree.
type Service struct {
	repository string
	capture    Capturer
	publisher  apply.Publisher
	lock       Locker
	now        func() time.Time

	publishing sync.WaitGroup // started publications

	mu        sync.Mutex
	proposals map[string]*proposal
	order     []string
	next      int
	last      *stand // the latest capture, reused by a proposal from its head
}

// New returns a Service for the repository at the given location.
func New(repository string, capture Capturer, publisher apply.Publisher, lock Locker) *Service {
	return &Service{repository: repository, capture: capture, publisher: publisher, lock: lock, now: time.Now, proposals: map[string]*proposal{}}
}

// Proposal is what the owner reviews: the changed files with their
// differences, the check of the proposed stand and, once decided, its state
// and publication receipt.
type Proposal struct {
	ID      string     `json:"id"`
	Kind    string     `json:"kind"`
	Title   string     `json:"title"`
	Created time.Time  `json:"created"`
	Head    string     `json:"head"`
	Changes []FileDiff `json:"changes"`
	Check   Check      `json:"check"`
	State   string     `json:"state"`
	Receipt *Receipt   `json:"receipt,omitempty"`
	Error   string     `json:"error,omitempty"`
	Decided time.Time  `json:"decided,omitzero"`
}

// Proposal states. Publishing lasts while an accepted proposal is written.
const (
	Proposed   = "proposed"
	Publishing = "publishing"
	Accepted   = "accepted"
	Discarded  = "discarded"
	Failed     = "failed"
)

// Check is the structural check of a proposed stand.
type Check struct {
	Status   string                  `json:"status"`
	Findings []apply.ProposalFinding `json:"findings"`
}

// Receipt is the publication's observed outcome and the read-back state of
// every path.
type Receipt struct {
	Outcome string      `json:"outcome"`
	Paths   []PathState `json:"paths"`
	Reason  string      `json:"reason,omitempty"`
}

// PathState is one published path and how it was read back.
type PathState struct {
	Path  string `json:"path"`
	State string `json:"state"`
}

type proposal struct {
	Proposal
	start   apply.CapturedPreparationAttempt
	changes []snapshot.Change
}

// Stand is a captured start: its structural preparation, tree and head.
type stand struct {
	attempt apply.CapturedPreparationAttempt
	tree    snapshot.Tree
	head    string
	at      time.Time
}

// standReuse bounds how long a capture serves proposals made from its head.
// Reuse is safe because acceptance checks the capture's guard: a source that
// changed since refuses the publication.
const standReuse = 15 * time.Minute

// ErrNotClean reports a capture refused because the working tree differs from
// its committed head: proposals start from committed source. The Capturer
// wraps it.
var ErrNotClean = errors.New("the working tree has uncommitted changes; commit them before proposing")

// ErrNoChange reports a proposal that changes nothing; ErrOutOfScope one that
// changes a path its producer may not change.
var (
	ErrNoChange   = errors.New("the proposal changes nothing")
	ErrOutOfScope = errors.New("the proposal changes a path it may not change")
)

func (s *Service) stand(ctx context.Context) (stand, error) {
	captured, err := s.capture(ctx)
	if err != nil {
		return stand{}, err
	}
	attempts := captured.Attempts()
	if len(attempts) == 0 {
		return stand{}, errors.New("the capture made no attempt")
	}
	final := attempts[len(attempts)-1]
	inputs := final.Model().Inputs()
	st := stand{attempt: final, tree: inputs.Raw.Census, head: inputs.Head, at: s.now()}
	s.mu.Lock()
	s.last = &st
	s.mu.Unlock()
	return st, nil
}

// standFrom returns the latest capture when it was made from head recently,
// and captures again otherwise.
func (s *Service) standFrom(ctx context.Context, head string) (stand, error) {
	s.mu.Lock()
	last := s.last
	s.mu.Unlock()
	if head != "" && last != nil && last.head == head && s.now().Sub(last.at) < standReuse {
		return *last, nil
	}
	return s.stand(ctx)
}

// propose checks a changed stand and keeps it as a proposal. allowed limits
// the paths the proposal may change; anything else is refused.
func (s *Service) propose(st stand, kind, title string, proposed snapshot.Tree, allowed func(string) bool) (Proposal, error) {
	changes := snapshot.Compare(st.tree, proposed)
	if len(changes) == 0 {
		return Proposal{}, ErrNoChange
	}
	for _, c := range changes {
		if !allowed(c.Path().String()) {
			return Proposal{}, fmt.Errorf("%w: a %s proposal may not change %s", ErrOutOfScope, kind, c.Path())
		}
	}
	checked := apply.CheckProposal(st.attempt, proposed)
	p := &proposal{start: st.attempt, changes: changes}
	p.Kind, p.Title, p.Created, p.Head, p.State = kind, title, s.now().UTC(), st.head, Proposed
	p.Check = Check{Status: string(checked.Status), Findings: checked.Findings}
	if p.Check.Findings == nil {
		p.Check.Findings = []apply.ProposalFinding{}
	}
	for _, c := range changes {
		p.Changes = append(p.Changes, fileDiff(c))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	p.ID = fmt.Sprintf("%s-%d", kind, s.next)
	s.proposals[p.ID] = p
	s.order = append(s.order, p.ID)
	return p.Proposal, nil
}

// Wait returns when every started publication has its receipt, so a cockpit
// that stops does not cut a write short.
func (s *Service) Wait() { s.publishing.Wait() }

// Proposals lists the proposals, newest first.
func (s *Service) Proposals() []Proposal {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Proposal, 0, len(s.order))
	for _, id := range slices.Backward(s.order) {
		out = append(out, s.proposals[id].Proposal)
	}
	return out
}

// Proposal returns one proposal.
func (s *Service) Proposal(id string) (Proposal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.proposals[id]
	if !ok {
		return Proposal{}, false
	}
	return p.Proposal, true
}

// Discard drops a proposal; nothing is written.
func (s *Service) Discard(id string) (Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.proposals[id]
	if !ok {
		return Proposal{}, fmt.Errorf("no proposal %s", id)
	}
	if p.State != Proposed {
		return p.Proposal, fmt.Errorf("proposal %s is already %s", id, p.State)
	}
	p.State, p.Decided = Discarded, s.now().UTC()
	return p.Proposal, nil
}

// Accept publishes a proposal whose check passed: under the state locks,
// against the guard of its capture, with read-back of every path. A source
// that changed since the capture refuses the publication; the proposal must
// then be made again. Only an applied publication accepts the proposal.
func (s *Service) Accept(ctx context.Context, id string) (Proposal, error) {
	s.mu.Lock()
	p, ok := s.proposals[id]
	switch {
	case !ok:
		s.mu.Unlock()
		return Proposal{}, fmt.Errorf("no proposal %s", id)
	case p.State != Proposed:
		s.mu.Unlock()
		return p.Proposal, fmt.Errorf("proposal %s is already %s", id, p.State)
	case p.Check.Status != string(model.Passed):
		s.mu.Unlock()
		return p.Proposal, fmt.Errorf("proposal %s does not pass the check (%s); it cannot be accepted", id, p.Check.Status)
	}
	p.State = Publishing
	s.publishing.Add(1)
	s.mu.Unlock()
	defer s.publishing.Done()

	receipt, started, err := s.publish(ctx, p)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !started {
		// Nothing was attempted (an Apply holds a state directory): the
		// proposal stays open for another attempt.
		p.State = Proposed
		return p.Proposal, err
	}
	// The publication touched the working tree: no later proposal may start
	// from the capture made before it.
	s.last = nil
	p.Decided = s.now().UTC()
	if receipt != nil {
		p.Receipt = receipt
	}
	switch {
	case err == nil && receipt != nil && (receipt.Outcome == string(apply.PublicationApplied) || receipt.Outcome == string(apply.PublicationNoChange)):
		p.State = Accepted
	case errors.Is(err, apply.ErrCaptureChanged):
		p.State, p.Error = Failed, "the committed source changed since the proposal was made; make it again"
	case err != nil:
		p.State, p.Error = Failed, err.Error()
	default:
		p.State, p.Error = Failed, "the publication was not applied: "+receipt.Outcome
	}
	if p.State == Failed && err == nil {
		err = errors.New(p.Error)
	}
	return p.Proposal, err
}

// publish writes an accepted proposal. started is false when the fence could
// not be taken, so nothing was attempted.
func (s *Service) publish(ctx context.Context, p *proposal) (*Receipt, bool, error) {
	control, release, err := s.lock(ctx)
	if err != nil {
		return nil, false, err
	}
	defer release()
	inputs := p.start.Model().Inputs()
	request := apply.PublicationRequest{ID: apply.PublicationID("authoring-" + p.ID), Run: "authoring-" + p.ID, Repository: s.repository, Guard: inputs.Guard, Changes: p.changes}
	permit, err := s.publisher.Preflight(ctx, request, control)
	if err != nil {
		return nil, true, err
	}
	observed, err := s.publisher.Publish(ctx, permit, control)
	receipt := &Receipt{Outcome: string(observed.Outcome), Reason: observed.Reason, Paths: []PathState{}}
	for _, path := range observed.Paths {
		receipt.Paths = append(receipt.Paths, PathState{Path: path.Path.String(), State: string(path.State)})
	}
	return receipt, true, err
}

// replace returns the tree with one file's content replaced or added.
func replace(tree snapshot.Tree, name string, content []byte) (snapshot.Tree, error) {
	path, err := snapshot.NewPath(name)
	if err != nil {
		return snapshot.Tree{}, err
	}
	mode := snapshot.Regular
	var entries []snapshot.Entry
	for _, e := range tree.Entries() {
		if e.Path() == path {
			mode = e.Mode()
			continue
		}
		entries = append(entries, e)
	}
	entry, err := snapshot.NewEntry(path, mode, content)
	if err != nil {
		return snapshot.Tree{}, err
	}
	return snapshot.NewTree(append(entries, entry))
}
