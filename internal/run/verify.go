package run

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Verification and publication commands (execution contract, run states
// Verifying and Publishing).
const (
	PrepareCheck       = "check.prepared"
	ObserveCheck       = "check.observed"
	VerifyRun          = "run.verified"
	PreparePublication = "publication.prepared"
	PublishRun         = "run.published"
)

const (
	checkRequestMedia       = "application/vnd.markitect.check-request+json"
	checkReceiptMedia       = "application/vnd.markitect.check-receipt+json"
	publicationMedia        = "application/vnd.markitect.publication+json"
	publicationReceiptMedia = "application/vnd.markitect.publication-receipt+json"
)

// CheckRecord is one admitted project check. Name is the configured check it
// runs; Request is its recorded request and Tree the tree it runs on. Receipt
// is the recorded observation once the check ended.
type CheckRecord struct {
	ID      CheckID
	Name    string
	Request BlobRef
	Tree    snapshot.TreeID
	Receipt BlobRef
}

func (r CheckRecord) payloadRefs() []BlobRef {
	var refs []BlobRef
	for _, ref := range []BlobRef{r.Request, r.Receipt} {
		if ref != (BlobRef{}) {
			refs = append(refs, ref)
		}
	}
	return refs
}

// VerificationRecord is the Host's final evidence check (execution contract,
// Verifying): the exact aggregate and the passing required checks on it.
type VerificationRecord struct {
	Aggregate uint64
	Tree      snapshot.TreeID
	Checks    []CheckID
}

// PublicationRecord is the publication of the verified aggregate: its changes
// against the captured source, recorded before the first write, and the
// observed receipt afterwards.
type PublicationRecord struct {
	ID        string
	Aggregate uint64
	Changes   BlobRef
	Receipt   BlobRef
}

func (r PublicationRecord) payloadRefs() []BlobRef {
	var refs []BlobRef
	for _, ref := range []BlobRef{r.Changes, r.Receipt} {
		if ref != (BlobRef{}) {
			refs = append(refs, ref)
		}
	}
	return refs
}

type checkRequestWire struct {
	ID             CheckID
	Name           string
	Tree           snapshot.TreeID
	ImageID        string
	Executable     string
	Args, Env      []string
	Directory      string
	Limits         CheckLimits
	Timeout        time.Duration
	MaxOutputBytes int64
}

type checkValue struct {
	Record  CheckRecord
	Request checkRequestWire
	Passed  bool
}

// CheckView is the read-only projection of one recorded check.
type CheckView struct {
	Record CheckRecord
	Passed bool
}

// Checks returns the run's recorded checks in admission order.
func (c *Coordinator) Checks() []CheckView {
	var out []CheckView
	for _, id := range c.state.checkOrder {
		v := c.state.checks[id]
		out = append(out, CheckView{v.Record, v.Passed})
	}
	return out
}

// AggregateTree is the tree identity of the run's current aggregate.
func (c *Coordinator) AggregateTree() snapshot.TreeID { return c.state.aggregateTree }

// PrepareCheckCommand admits one required check on the aggregate while the run
// is Verifying. The reservation is project-check assessment work.
func (c *Coordinator) PrepareCheckCommand(meta RecordMeta, name string, req CheckRequest, reservation Reservation) (Command, error) {
	if !req.Valid() || !identifier(name) {
		return Command{}, ErrInvalid
	}
	w := checkRequestWire{ID: req.ID, Name: name, Tree: req.Tree.Digest(), ImageID: req.ImageID, Executable: req.Executable, Args: slices.Clone(req.Args), Env: slices.Clone(req.Env),
		Directory: req.Directory.String(), Limits: req.Limits, Timeout: req.Timeout, MaxOutputBytes: req.MaxOutputBytes}
	data, err := json.Marshal(w)
	if err != nil {
		return Command{}, err
	}
	p, err := NewPayload(checkRequestMedia, "utf-8", data)
	if err != nil {
		return Command{}, err
	}
	cmd := Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: PrepareCheck, Assignment: c.Current().Assignment, Run: c.Current().ID, Reservation: reservation,
		Check: CheckRecord{ID: req.ID, Name: name, Request: p.Ref(), Tree: w.Tree}, sessionPayloads: []Payload{p}}
	cmd.managerProof = commandDigest(cmd)
	return cmd, nil
}

// ObserveCheckCommand records a check's receipt and settles its reservation
// with the observed elapsed time; an uncertain receipt keeps the hold and
// moves the run to Reconciling.
func (c *Coordinator) ObserveCheckCommand(meta RecordMeta, receipt CheckReceipt, elapsed time.Duration) (Command, error) {
	data, err := json.Marshal(receipt)
	if err != nil {
		return Command{}, err
	}
	p, err := NewPayload(checkReceiptMedia, "utf-8", data)
	if err != nil {
		return Command{}, err
	}
	reason := "Host observed the check's end and elapsed duration"
	if receipt.Uncertain {
		reason = "the check's end is not known"
	}
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: ObserveCheck, Assignment: c.Current().Assignment, Run: c.Current().ID, Elapsed: elapsed, Reason: reason,
		Check: CheckRecord{ID: receipt.ID, Receipt: p.Ref()}, sessionPayloads: []Payload{p}}, nil
}

// VerifyRunCommand records the final evidence check and moves the run to
// Publishing.
func (c *Coordinator) VerifyRunCommand(meta RecordMeta, checks []CheckID) Command {
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: VerifyRun, Assignment: c.Current().Assignment, Run: c.Current().ID,
		Verification: VerificationRecord{Aggregate: c.state.aggregateVersion, Tree: c.state.aggregateTree, Checks: slices.Clone(checks)}}
}

// PublishedChange is one recorded publication change: a path and its content
// identity before and after; an absent side is nil.
type PublishedChange struct {
	Path   string
	Before *PublishedEntry
	After  *PublishedEntry
}

// PublishedEntry is the mode and content identity of one side of a change.
type PublishedEntry struct {
	Mode      snapshot.Mode
	ContentID snapshot.ContentID
}

// PreparePublicationCommand records the exact changes to publish before the
// first write, so that an interrupted publication can be inspected.
func (c *Coordinator) PreparePublicationCommand(meta RecordMeta, id string, changes []snapshot.Change) (Command, error) {
	recorded := []PublishedChange{}
	for _, ch := range changes {
		pc := PublishedChange{Path: ch.Path().String()}
		if e, ok := ch.Before(); ok {
			pc.Before = &PublishedEntry{e.Mode(), e.ContentID()}
		}
		if e, ok := ch.After(); ok {
			pc.After = &PublishedEntry{e.Mode(), e.ContentID()}
		}
		recorded = append(recorded, pc)
	}
	data, err := json.Marshal(recorded)
	if err != nil {
		return Command{}, err
	}
	p, err := NewPayload(publicationMedia, "utf-8", data)
	if err != nil {
		return Command{}, err
	}
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: PreparePublication, Assignment: c.Current().Assignment, Run: c.Current().ID,
		Publication: PublicationRecord{ID: id, Aggregate: c.state.aggregateVersion, Changes: p.Ref()}, sessionPayloads: []Payload{p}}, nil
}

// PublicationResult is the observed result of a publication, as the
// publication adapter reports it.
type PublicationResult struct {
	ID      string
	Outcome string
	Paths   []PublishedPathState
	Reason  string
}

// PublishedPathState is the observed state of one requested path: after,
// before or other.
type PublishedPathState struct {
	Path  string
	State string
}

const (
	publicationApplied  = "applied"
	publicationNoChange = "no_change"
	publicationStale    = "stale"
	publicationPartial  = "partial"
	publicationUnknown  = "unknown"
	publicationFailed   = "failed"
)

// PublishRunCommand records the publication's receipt. Applied and no change
// finish the run with that outcome; stale and failed finish it as such; a
// partial or unknown publication moves the run to Reconciling.
func (c *Coordinator) PublishRunCommand(meta RecordMeta, result PublicationResult) (Command, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return Command{}, err
	}
	p, err := NewPayload(publicationReceiptMedia, "utf-8", data)
	if err != nil {
		return Command{}, err
	}
	reason := result.Reason
	if reason == "" {
		reason = "the publication was " + result.Outcome
	}
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: PublishRun, Assignment: c.Current().Assignment, Run: c.Current().ID, Reason: reason,
		Publication: PublicationRecord{ID: c.state.publication.ID, Receipt: p.Ref()}, sessionPayloads: []Payload{p}}, nil
}

// applyVerification reduces the verification and publication commands.
func (s *stateValue) applyVerification(c Command) error {
	if s.basisVersion != 3 || len(s.tasks) == 0 {
		return ErrState
	}
	r := s.current
	switch c.Kind {
	case PrepareCheck:
		request, ok := payloadFor(c, c.Check.Request)
		var w checkRequestWire
		if r.State != Verifying || !ok || len(c.sessionPayloads) != 1 || request.Ref().MediaType != checkRequestMedia || strictJSON(request.Bytes(), &w) != nil {
			return ErrInvalid
		}
		if _, exists := s.checks[c.Check.ID]; exists || w.ID != c.Check.ID || w.Name != c.Check.Name || w.Tree != c.Check.Tree || c.Check.Tree != s.aggregateTree || s.hasOpen() {
			return ErrState
		}
		if c.Reservation.Kind != ProjectCheck || c.Reservation.Purpose != Assessment || c.Reservation.Maximum < w.Timeout {
			return ErrInvalid
		}
		if err := s.reserve(c.Reservation); err != nil {
			return err
		}
		s.checks[c.Check.ID] = checkValue{Record: c.Check, Request: w}
		s.checkOrder = append(s.checkOrder, c.Check.ID)
		s.checkInvocations[c.Check.ID] = c.Reservation.ID
		return nil
	case ObserveCheck:
		check, ok := s.checks[c.Check.ID]
		receipt, found := payloadFor(c, c.Check.Receipt)
		var observed CheckReceipt
		if !ok || check.Record.Receipt != (BlobRef{}) {
			return ErrState
		}
		if !found || len(c.sessionPayloads) != 1 || receipt.Ref().MediaType != checkReceiptMedia || strictJSON(receipt.Bytes(), &observed) != nil || observed.ID != c.Check.ID {
			return ErrInvalid
		}
		invocation, held := s.invocations[s.checkInvocations[c.Check.ID]]
		if !held || invocation.State != Reserved {
			return ErrState
		}
		check.Record.Receipt = c.Check.Receipt
		if observed.Uncertain {
			invocation.State = Uncertain
			r.State, r.Reason = Reconciling, c.Reason
		} else {
			invocation.State, invocation.Elapsed, invocation.Observation = Settled, c.Elapsed, c.Reason
			w := check.Request
			check.Passed = observed.ImageID == w.ImageID && observed.Tree == w.Tree && observed.Started && observed.Exited && observed.ExitCode == 0 && !observed.TimedOut
		}
		s.invocations[invocation.Reservation.ID] = invocation
		s.checks[c.Check.ID] = check
	case VerifyRun:
		v := c.Verification
		if r.State != Verifying || s.hasOpen() || s.hasLiveSessions() || v.Aggregate != s.aggregateVersion || v.Tree != s.aggregateTree {
			return ErrState
		}
		for _, t := range s.tasks {
			if t.State != TaskDone || len(s.openBlockingIssues(t.Slot.Package)) != 0 {
				return ErrState
			}
		}
		seen := map[CheckID]bool{}
		for _, id := range v.Checks {
			check, ok := s.checks[id]
			if !ok || seen[id] || !check.Passed || check.Record.Tree != s.aggregateTree {
				return ErrState
			}
			seen[id] = true
		}
		s.verification = v
		r.State = Publishing
	case PreparePublication:
		changes, ok := payloadFor(c, c.Publication.Changes)
		var recorded []PublishedChange
		if r.State != Publishing || s.publication.ID != "" || !identifier(c.Publication.ID) || c.Publication.Aggregate != s.aggregateVersion || c.Publication.Receipt != (BlobRef{}) {
			return ErrState
		}
		if !ok || len(c.sessionPayloads) != 1 || changes.Ref().MediaType != publicationMedia || strictJSON(changes.Bytes(), &recorded) != nil {
			return ErrInvalid
		}
		s.publication = c.Publication
		s.publicationChanges = len(recorded)
		return nil
	case PublishRun:
		receipt, ok := payloadFor(c, c.Publication.Receipt)
		var result PublicationResult
		if r.State != Publishing || s.publication.ID == "" || s.publication.Receipt != (BlobRef{}) || c.Publication.ID != s.publication.ID {
			return ErrState
		}
		if !ok || len(c.sessionPayloads) != 1 || receipt.Ref().MediaType != publicationReceiptMedia || strictJSON(receipt.Bytes(), &result) != nil || result.ID != s.publication.ID || !text(c.Reason) {
			return ErrInvalid
		}
		switch {
		case result.Outcome == publicationApplied && s.publicationChanges > 0:
			r.State, r.Outcome = Finished, Applied
		case result.Outcome == publicationNoChange && s.publicationChanges == 0:
			r.State, r.Outcome = Finished, NoChange
		case result.Outcome == publicationStale:
			r.State, r.Outcome = Finished, Stale
		case result.Outcome == publicationFailed:
			r.State, r.Outcome = Finished, Failed
		case result.Outcome == publicationPartial || result.Outcome == publicationUnknown:
			r.State = Reconciling
		default:
			return ErrInvalid
		}
		s.publication.Receipt = c.Publication.Receipt
		r.Reason, r.NextActor = c.Reason, ""
	default:
		return ErrState
	}
	s.current = r
	s.runs[r.ID] = r
	return nil
}

func (s stateValue) openBlockingIssues(manager string) []Issue {
	var out []Issue
	for _, issue := range s.openIssues(manager) {
		if issue.Severity == severityBlocking {
			out = append(out, issue)
		}
	}
	return out
}
