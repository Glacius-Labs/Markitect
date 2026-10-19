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
// runs; Request is its recorded request and Tree the tree it runs on. Task
// names the Manager whose review or writing step requested it; it is empty for
// the aggregate's required checks. Receipt is the recorded observation once the
// check ended.
type CheckRecord struct {
	ID      CheckID
	Name    string
	Task    string
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
// Verifying): the aggregate version, the exact tree that publishing it writes
// (its files and regenerated registry shards), and the passing required
// checks on that tree.
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
	// Source is the captured tree the changes apply to, Target the verified
	// tree they produce.
	Source, Target snapshot.TreeID
	Changes        BlobRef
	Receipt        BlobRef
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

// PrepareReviewCheckCommand admits a check that a manager's task requests: its
// candidate review on the frozen candidate, or its writing step on the
// workspace it observed.
func (c *Coordinator) PrepareReviewCheckCommand(meta RecordMeta, manager, name string, req CheckRequest, reservation Reservation) (Command, error) {
	cmd, err := c.PrepareCheckCommand(meta, name, req, reservation)
	if err != nil {
		return Command{}, err
	}
	cmd.Check.Task = manager
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

// VerifyRunCommand records the final evidence check of the tree that
// publishing the aggregate writes, and moves the run to Publishing.
func (c *Coordinator) VerifyRunCommand(meta RecordMeta, target snapshot.TreeID, checks []CheckID) Command {
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: VerifyRun, Assignment: c.Current().Assignment, Run: c.Current().ID,
		Verification: VerificationRecord{Aggregate: c.state.aggregateVersion, Tree: target, Checks: slices.Clone(checks)}}
}

// RequiredChecks names the checks the basis requires before publication.
func (c *Coordinator) RequiredChecks() []string {
	if c.state.managerBasis.RequiredChecks == nil {
		return nil
	}
	return slices.Clone(*c.state.managerBasis.RequiredChecks)
}

// Verification returns the run's final evidence check, once recorded.
func (c *Coordinator) Verification() VerificationRecord {
	v := c.state.verification
	v.Checks = slices.Clone(v.Checks)
	return v
}

// Publication returns the run's recorded publication, once prepared.
func (c *Coordinator) Publication() PublicationRecord { return c.state.publication }

// SourceTree is the identity of the run's captured source.
func (c *Coordinator) SourceTree() snapshot.TreeID { return c.state.sourceTree }

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
func (c *Coordinator) PreparePublicationCommand(meta RecordMeta, id string, target snapshot.TreeID, changes []snapshot.Change) (Command, error) {
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
		Publication: PublicationRecord{ID: id, Aggregate: c.state.aggregateVersion, Source: c.state.sourceTree, Target: target, Changes: p.Ref()}, sessionPayloads: []Payload{p}}, nil
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
		if !ok || len(c.sessionPayloads) != 1 || request.Ref().MediaType != checkRequestMedia || strictJSON(request.Bytes(), &w) != nil {
			return ErrInvalid
		}
		if _, exists := s.checks[c.Check.ID]; exists || w.ID != c.Check.ID || w.Name != c.Check.Name || w.Tree != c.Check.Tree || !s.checkAdmissible(c.Check) {
			return ErrState
		}
		if c.Reservation.Kind != ProjectCheck || c.Reservation.Purpose != s.checkPurpose(c.Check) || c.Reservation.Maximum < w.Timeout {
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
		if r.State != Verifying || s.hasOpen() || s.hasLiveSessions() || v.Aggregate != s.aggregateVersion {
			return ErrState
		}
		for _, t := range s.tasks {
			if t.State != TaskDone || len(s.openBlockingIssues(t.Slot.Package)) != 0 {
				return ErrState
			}
		}
		seen, names := map[CheckID]bool{}, map[string]bool{}
		for _, id := range v.Checks {
			check, ok := s.checks[id]
			if !ok || seen[id] || !check.Passed || check.Record.Task != "" || check.Record.Tree != v.Tree {
				return ErrState
			}
			seen[id], names[check.Record.Name] = true, true
		}
		if s.managerBasis.RequiredChecks == nil {
			return ErrState
		}
		for _, name := range *s.managerBasis.RequiredChecks {
			if !names[name] {
				return ErrState // every required check passed on the published tree
			}
		}
		s.verification = v
		r.State = Publishing
	case PreparePublication:
		changes, ok := payloadFor(c, c.Publication.Changes)
		var recorded []PublishedChange
		if r.State != Publishing || s.publication.ID != "" || !identifier(c.Publication.ID) || c.Publication.Aggregate != s.aggregateVersion || c.Publication.Receipt != (BlobRef{}) ||
			c.Publication.Source != s.sourceTree || c.Publication.Target != s.verification.Tree {
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

// checkPurpose is the budget purpose of a check: writing work while the task
// runs a writing step, assessment for a review and the aggregate's verification.
func (s stateValue) checkPurpose(k CheckRecord) Purpose {
	if t, ok := s.tasks[k.Task]; ok && t.Phase.writing() {
		return Writing
	}
	return Assessment
}

// checkAdmissible admits the aggregate's required checks while the run is
// Verifying and nothing else is open, a review's check while that review's
// candidate step runs, on its frozen candidate, and a writing step's check on
// any tree while the task holds an admitted work or integrate step, one check
// at a time.
func (s stateValue) checkAdmissible(k CheckRecord) bool {
	if k.Task == "" {
		return s.current.State == Verifying && !s.hasOpen()
	}
	t, ok := s.tasks[k.Task]
	if !ok || s.current.State != Running || t.Step == "" {
		return false
	}
	switch {
	case t.Phase.writing():
	case t.Phase == PhaseReviewerCandidate && s.sessions[t.Session].Plan.Candidate == k.Tree:
	default:
		return false
	}
	for _, id := range s.checkOrder {
		if open(s.invocations[s.checkInvocations[id]]) {
			return false
		}
	}
	return true
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
