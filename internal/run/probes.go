package run

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Probe outcomes. They are the probe's own assertion verdict, separate from
// the exit code of the application it exercises: a probe that requires a
// modeled refusal passes when it observes that refusal. Error means the
// Runtime could not establish an outcome: it did not start, timed out, was
// uncertain or does not match its request. An error never counts as a pass
// or as an expected refusal.
const (
	ProbePassed = "passed"
	ProbeFailed = "failed"
	ProbeError  = "error"
)

// Stands a probe runs on. A candidate is the immutable stand of a review; an
// accepted stand and a child's checked proposal are stable; a live stand is the
// accepted stand composed with the requester's own area while it may still
// write, which supports a submission only when the submission's checked stand
// is exactly that stand.
const (
	standCandidate = "candidate"
	standAccepted  = "accepted"
	standProposal  = "proposal"
	standLive      = "live"
)

// Dispositions of an adverse probe receipt (failed or error). A violation is a
// genuine finding that cites the receipt; an expectation error is a reasoned
// disproof of the probe itself, its expectation, assertion or execution, not of
// the candidate; a rerun replaces an error receipt by a later passing receipt of
// the same claim on the same stand.
const (
	DisposedViolation        = "violation"
	DisposedExpectationError = "expectation_error"
	DisposedRerun            = "rerun"
)

const (
	maxProbeBytes    = 64 << 10
	maxProbeArgs     = 32
	maxProbesPerTurn = 12
	probeShown       = 1500
)

// ProbeReceipt is the Orchestrator's record of one behavioral probe: the
// requester's declared obligation, claim and expectation, recorded before the
// probe ran; the exact stand and the probe bytes it ran on; the execution
// profile and invocation; and what the Runtime observed. An Agent cannot create
// one by writing fields: only the tool check does, and every receipt is kept,
// whatever a report cites.
type ProbeReceipt struct {
	ID         string `json:"id"`
	Pass       string `json:"pass"`
	Requester  string `json:"requester"`
	Turn       string `json:"turn"`
	Obligation string `json:"obligation"`
	Claim      string `json:"claim"`
	Expected   string `json:"expected"`
	// Check names the required check whose execution profile ran the probe.
	Check     string `json:"check"`
	StandKind string `json:"standKind"`
	Proposal  string `json:"proposal,omitempty"`
	// Stand is the digest of the stand probed; Tree the digest of the tree that
	// ran: the stand plus the probe file at Path.
	Stand         string   `json:"stand"`
	Tree          string   `json:"tree"`
	Path          string   `json:"path"`
	Content       string   `json:"content"`
	ContentDigest string   `json:"contentDigest"`
	ImageID       string   `json:"imageID"`
	Executable    string   `json:"executable"`
	Args          []string `json:"args"`
	Directory     string   `json:"directory"`
	Started       bool     `json:"started"`
	Exited        bool     `json:"exited"`
	ExitCode      int      `json:"exitCode"`
	TimedOut      bool     `json:"timedOut"`
	Uncertain     bool     `json:"uncertain"`
	Output        string   `json:"output,omitempty"`
	Truncated     bool     `json:"truncated,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	Outcome       string   `json:"outcome"`
	// Rerun names the error receipt whose exact probe this one repeats.
	Rerun string    `json:"rerun,omitempty"`
	At    time.Time `json:"at"`
	// Disposition is the accepted disposition of an adverse receipt.
	Disposition *ProbeDisposition `json:"disposition,omitempty"`
}

// ProbeDisposition is the accepted disposition of an adverse receipt. Pending
// marks Root's disproof of its own expectation in the same Pass: nobody has
// assessed it independently yet, so it keeps the Pass from being clean until a
// fresh reviewer confirms it or raises a finding.
type ProbeDisposition struct {
	As       string `json:"as"`
	By       string `json:"by"`
	Pass     string `json:"pass"`
	Proposal string `json:"proposal,omitempty"`
	Reason   string `json:"reason,omitempty"`
	Evidence string `json:"evidence,omitempty"`
	Finding  string `json:"finding,omitempty"`
	Rerun    string `json:"rerun,omitempty"`
	// Passing names, for an error disposed of as an expectation error, the
	// passing receipt of the same obligation on the same stand that establishes
	// the claim without the erring probe.
	Passing string `json:"passing,omitempty"`
	Pending bool   `json:"pending,omitempty"`
}

func (r *ProbeReceipt) adverse() bool { return r.Outcome != ProbePassed }

// disposed reports whether an adverse receipt has an effective disposition.
func (r *ProbeReceipt) disposed() bool { return r.Disposition != nil && !r.Disposition.Pending }

type probeIn struct {
	Obligation string   `json:"obligation"`
	Claim      string   `json:"claim"`
	Expected   string   `json:"expected"`
	Check      string   `json:"check"`
	Path       string   `json:"path"`
	Content    string   `json:"content"`
	Args       []string `json:"args"`
	Proposal   string   `json:"proposal"`
	Rerun      string   `json:"rerun"`
}

// probeSpec is a validated probe: what runs and what it claims.
type probeSpec struct {
	definition ModelDefinition
	claim      string
	expected   string
	check      *Check
	path       snapshot.Path
	content    string
	args       []string
	proposal   string
	rerun      string
}

// evidenceItem supports one claim of an obligation's coverage: a probe receipt
// (executed behavior) or a source location of the stand (inspection).
// Stand and Digest are the Orchestrator's binding: the stand the item was
// checked against and, for a source, the digest of the cited file there.
type evidenceItem struct {
	Claim     string `json:"claim"`
	Receipt   string `json:"receipt,omitempty"`
	Source    string `json:"source,omitempty"`
	Location  string `json:"location,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
	Stand     string `json:"stand,omitempty"`
	Digest    string `json:"digest,omitempty"`
}

type dispositionIn struct {
	Receipt  string `json:"receipt"`
	As       string `json:"as"`
	Reason   string `json:"reason"`
	Evidence string `json:"evidence"`
	Rerun    string `json:"rerun"`
	Passing  string `json:"passing"`
}

// probesAvailable reports whether a required check offers an execution profile
// that can run a probe.
func (e *engine) probesAvailable() bool { return e.cfg.Runner != nil && len(e.cfg.Checks) > 0 }

func (e *engine) receiptLocked(id string) *ProbeReceipt {
	for _, r := range e.receipts {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// probe serves check with a probe: it binds the declared claim and expectation
// before anything runs, places the probe file on a stable stand, runs it in its
// own container with the execution profile of a required check and records the
// receipt, whatever its outcome. A rerun repeats exactly the probe of an error
// receipt on the same candidate, so that a passing rerun can replace an error
// of the Runtime but never a different, weaker probe.
func (e *engine) probe(ctx context.Context, m *member, ts *turnState, in probeIn) ToolAnswer {
	if !e.probesAvailable() {
		return refuse("No probe can run: the project declares no check whose execution profile could run it. Use source inspection instead.")
	}
	var spec probeSpec
	var why string
	if strings.TrimSpace(in.Rerun) != "" {
		spec, why = e.rerunSpec(m, in)
	} else {
		spec, why = e.probeSpecOf(m, in)
	}
	if why != "" {
		return refuse("%s", why)
	}
	return e.runProbe(ctx, m, ts, spec)
}

// rerunSpec is the exact probe of an error receipt, for a review of the same
// candidate.
func (e *engine) rerunSpec(m *member, in probeIn) (probeSpec, string) {
	if in.Obligation != "" || in.Claim != "" || in.Expected != "" || in.Check != "" || in.Path != "" || in.Content != "" || in.Args != nil || in.Proposal != "" {
		return probeSpec{}, "A rerun repeats a receipt's probe exactly; give only probe.rerun."
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	r := e.receiptLocked(strings.TrimSpace(in.Rerun))
	if r != nil && r.Rerun != "" {
		// Every repetition repeats the original probe, so that one passing
		// repetition disposes of the original and its failed repetitions.
		r = e.receiptLocked(r.Rerun)
	}
	switch {
	case r == nil:
		return probeSpec{}, fmt.Sprintf("probe.rerun %q is no probe receipt of this run.", in.Rerun)
	case r.Outcome != ProbeError:
		return probeSpec{}, fmt.Sprintf("%s %s; only a Runtime error is run again. A failing assertion is disposed of by a finding or a reasoned expectation_error.", r.ID, describeOutcome(r))
	case !m.p.spec.Review || r.Stand != m.p.tree.Digest().String():
		return probeSpec{}, fmt.Sprintf("%s ran on another stand; a rerun repeats it on the same candidate in a review.", r.ID)
	case !e.answersForLocked(r, m.Path):
		return probeSpec{}, fmt.Sprintf("%s was requested by %s about %s, both outside your responsibility.", r.ID, r.Requester, r.Obligation)
	}
	d, _ := e.cfg.Definition(r.Obligation)
	var chk *Check
	for i := range e.cfg.Checks {
		if e.cfg.Checks[i].Name == r.Check {
			chk = &e.cfg.Checks[i]
		}
	}
	path, err := snapshot.NewPath(r.Path)
	if chk == nil || err != nil || len(r.Args) < chk.ProfileArgs {
		return probeSpec{}, fmt.Sprintf("%s cannot be repeated.", r.ID)
	}
	return probeSpec{definition: d, claim: r.Claim, expected: r.Expected, check: chk, path: path, content: r.Content, args: slices.Clone(r.Args[chk.ProfileArgs:]), rerun: r.ID}, ""
}

// probeSpecOf validates a new probe.
func (e *engine) probeSpecOf(m *member, in probeIn) (probeSpec, string) {
	claim, err := toolText("probe.claim", in.Claim)
	if err != nil {
		return probeSpec{}, err.Error()
	}
	expected, err := toolText("probe.expected", in.Expected)
	if err != nil {
		return probeSpec{}, err.Error()
	}
	d, ok := e.cfg.Definition(strings.TrimSpace(in.Obligation))
	if !ok {
		return probeSpec{}, fmt.Sprintf("probe.obligation %q is no Definition of the bound model.", in.Obligation)
	}
	e.mu.Lock()
	within := e.within(d.Owner, m.Path)
	e.mu.Unlock()
	if !within {
		return probeSpec{}, fmt.Sprintf("probe.obligation %s is owned by %s, outside your responsibility; probe your own obligations or those of your descendants.", d.ID, d.Owner)
	}
	var chk *Check
	for i := range e.cfg.Checks {
		if e.cfg.Checks[i].Name == strings.TrimSpace(in.Check) {
			chk = &e.cfg.Checks[i]
		}
	}
	if chk == nil {
		var names []string
		for _, c := range e.cfg.Checks {
			names = append(names, c.Name)
		}
		return probeSpec{}, fmt.Sprintf("probe.check names the required check whose execution profile runs the probe: one of %s.", strings.Join(names, ", "))
	}
	if in.Content == "" || len(in.Content) > maxProbeBytes || !utf8.ValidString(in.Content) || strings.ContainsRune(in.Content, 0) {
		return probeSpec{}, fmt.Sprintf("probe.content must be non-empty text of at most %d bytes.", maxProbeBytes)
	}
	if len(in.Args) > maxProbeArgs {
		return probeSpec{}, fmt.Sprintf("probe.args holds at most %d arguments.", maxProbeArgs)
	}
	for i, arg := range in.Args {
		if !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
			return probeSpec{}, fmt.Sprintf("probe.args[%d] is not valid text.", i)
		}
	}
	rel, err := snapshot.NewPath(strings.TrimSpace(in.Path))
	if err != nil {
		return probeSpec{}, fmt.Sprintf("probe.path must be a clean relative path below the check's directory %s.", chk.Request.Directory)
	}
	full := rel.String()
	if !chk.Request.Directory.IsRoot() {
		full = chk.Request.Directory.String() + "/" + full
	}
	path, err := snapshot.NewPath(full)
	if err != nil {
		return probeSpec{}, fmt.Sprintf("probe.path %q does not name a file below %s.", in.Path, chk.Request.Directory)
	}
	return probeSpec{definition: d, claim: claim, expected: expected, check: chk, path: path, content: in.Content, args: in.Args, proposal: strings.TrimSpace(in.Proposal)}, ""
}

// runProbe places a validated probe on its stand, runs it and records the
// receipt.
func (e *engine) runProbe(ctx context.Context, m *member, ts *turnState, spec probeSpec) ToolAnswer {
	d, chk, path := spec.definition, spec.check, spec.path
	e.mu.Lock()
	if ts.probes >= maxProbesPerTurn {
		e.mu.Unlock()
		return refuse("At most %d probes run in one turn.", maxProbesPerTurn)
	}
	// A probe ends before the turn's deadline with time left for the report:
	// a turn killed by its timeout would lose its work, not only the probe.
	reserve, least := e.cfg.Budget.TurnTimeout/5, e.cfg.Budget.TurnTimeout/10
	left := ts.deadline.Sub(e.now()) - reserve
	if !ts.deadline.IsZero() && left < least {
		e.mu.Unlock()
		return refuse("Too little of this turn is left for a probe (they take a minute or more when the Runtime is busy). Report what you have, with progress if you continue, and probe in your next turn.")
	}
	p := m.p
	if p != e.cur || p.sealed {
		e.mu.Unlock()
		return refuse("This turn belongs to %s, which no longer runs probes.", p.id)
	}
	stand, kind, proposalID, why := e.probeStandLocked(m, spec.proposal)
	e.mu.Unlock()
	if why != "" {
		return refuse("%s", why)
	}
	if kind == standLive {
		live, reason := e.liveStand(ctx, m)
		if reason != "" {
			return refuse("%s", reason)
		}
		stand = live
	}
	if _, exists := stand.Lookup(path); exists {
		return refuse("%s exists in the stand; a probe never replaces a project file. Choose another probe.path.", path)
	}
	entry, err := snapshot.NewEntry(path, snapshot.Regular, []byte(spec.content))
	if err != nil {
		return refuse("probe.path: %v", err)
	}
	tree, err := snapshot.NewTree(append(stand.Entries(), entry))
	if err != nil {
		return refuse("probe.path %s cannot be added to the stand: %v", path, err)
	}
	sum := sha256.Sum256([]byte(spec.content))
	e.mu.Lock()
	ts.probes++
	e.nextReceipt++
	r := &ProbeReceipt{ID: fmt.Sprintf("R%d", e.nextReceipt), Pass: p.id, Requester: m.Path, Turn: ts.id, Obligation: d.ID, Claim: spec.claim, Expected: spec.expected,
		Check: chk.Name, StandKind: kind, Proposal: proposalID, Stand: stand.Digest().String(), Tree: tree.Digest().String(), Path: path.String(),
		Content: spec.content, ContentDigest: hex.EncodeToString(sum[:]), Directory: chk.Request.Directory.String(), Rerun: spec.rerun}
	req := chk.Request
	req.ID = CheckID(fmt.Sprintf("%s-%s-%s", e.cfg.RunID, p.id, strings.ToLower(r.ID)))
	req.Tree = tree
	req.Args = append(slices.Clone(req.Args[:chk.ProfileArgs]), spec.args...)
	r.ImageID, r.Executable, r.Args = req.ImageID, req.Executable, req.Args
	// The claim and expectation are on record before the probe runs.
	_ = e.record(Event{Kind: "probe.requested", Pass: p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"id": r.ID, "obligation": r.Obligation, "claim": r.Claim, "expected": r.Expected,
		"check": r.Check, "standKind": r.StandKind, "proposal": r.Proposal, "stand": r.Stand, "path": r.Path, "contentDigest": r.ContentDigest, "args": r.Args, "rerun": r.Rerun}})
	e.mu.Unlock()
	began := e.now()
	probeCtx := ctx
	if !ts.deadline.IsZero() {
		// The cutoff is absolute: preparing the stand does not move it.
		var cancel context.CancelFunc
		probeCtx, cancel = context.WithTimeout(ctx, ts.deadline.Sub(began)-reserve)
		defer cancel()
	}
	rec, runErr := e.cfg.Runner.Run(probeCtx, req)
	cut := probeCtx.Err() != nil && ctx.Err() == nil
	e.mu.Lock()
	defer e.mu.Unlock()
	r.Started, r.Exited, r.ExitCode, r.TimedOut, r.Uncertain = rec.Started, rec.Exited, rec.ExitCode, rec.TimedOut, rec.Uncertain
	r.Output, r.Truncated, r.Reason, r.At = tail(rec.Output, outputTail), rec.Truncated, rec.Reason, e.now()
	if rec.ImageID != "" {
		r.ImageID = rec.ImageID
	}
	if runErr != nil {
		r.Reason = strings.TrimSpace(r.Reason + " " + runErr.Error())
	}
	r.Outcome = probeOutcome(rec, runErr, req)
	switch {
	case r.Outcome == ProbeError && cut:
		r.Reason = strings.TrimSpace("the probe was cut to keep time for the turn's report; " + r.Reason)
	case r.Outcome == ProbeError && r.Reason == "":
		r.Reason = fmt.Sprintf("the probe did not run to its assertions (exit %d)", r.ExitCode)
	}
	e.receipts = append(e.receipts, r)
	_ = e.record(Event{Kind: "probe.receipt", Pass: p.id, Manager: m.Path, Turn: ts.id, Data: *r})
	var b strings.Builder
	fmt.Fprintf(&b, "Receipt %s: %s, after %d seconds.", r.ID, describeOutcome(r), int(r.At.Sub(began).Seconds()))
	switch kind {
	case standLive:
		fmt.Fprintf(&b, " It ran on your live area composed with the accepted stand (%s); it supports your submission only if your area is unchanged when you submit.", short(r.Stand))
	default:
		fmt.Fprintf(&b, " It ran on the %s %s.", kind, short(r.Stand))
	}
	if r.Output != "" {
		fmt.Fprintf(&b, "\nOutput:\n%s", r.Output)
	}
	if r.Outcome != ProbePassed && p.spec.Review {
		b.WriteString("\nThis adverse receipt stays on record. Before a clean assessment, cite it in a genuine finding when it shows a violation, or dispose of it in your report as expectation_error with reason and evidence when your expectation or assertion was wrong; when it assesses a child's obligation, that child's reviewer judges it instead (return or reopen its proposal with the receipt), which avoids a further Verify for your own disproof. A Runtime error can be repeated exactly with probe.rerun; if the rerun passes, dispose of the error as rerun. An error within the probe's own execution, such as a probe too large for the check's time limit, is an expectation_error with reason, evidence and a passing receipt of the same obligation on this candidate; a timeout or a kill can also be a hang or crash of the candidate, so that passing probe must establish the claim itself. An error that points at the candidate, such as a hang, is a genuine finding.")
	}
	return ToolAnswer{Text: b.String()}
}

// probeStandLocked selects the stable stand a probe of this turn runs on.
func (e *engine) probeStandLocked(m *member, proposalID string) (snapshot.Tree, string, string, string) {
	p := m.p
	switch {
	case p.spec.Review:
		if proposalID != "" {
			return snapshot.Tree{}, "", "", "A review probe always runs on the candidate; omit probe.proposal."
		}
		return p.tree, standCandidate, "", ""
	case proposalID != "":
		if !p.spec.Writes {
			return snapshot.Tree{}, "", "", "In a Pass without writes every probe runs on the accepted stand; omit probe.proposal."
		}
		for _, pr := range p.proposals {
			if pr.id == proposalID {
				if pr.member.Parent != m.Path || pr.status != proposalChecked || pr.member.pending != pr {
					return snapshot.Tree{}, "", "", fmt.Sprintf("%s is not a checked proposal of one of your children awaiting your decision.", proposalID)
				}
				return pr.composite, standProposal, pr.id, ""
			}
		}
		return snapshot.Tree{}, "", "", fmt.Sprintf("%q is no proposal of this Pass.", proposalID)
	case !p.spec.Writes:
		return p.accepted, standAccepted, "", ""
	case m.stage == stageOwn || m.stage == stageIntegrate:
		return snapshot.Tree{}, standLive, "", ""
	}
	return p.accepted, standAccepted, "", ""
}

// liveStand composes the accepted stand with the requester's own live area.
func (e *engine) liveStand(ctx context.Context, m *member) (snapshot.Tree, string) {
	h, err := e.cfg.Workspaces.Peek(ctx, HarvestRequest{ID: e.workID, MaxEntries: e.cfg.MaxEntries, MaxBytes: e.cfg.MaxBytes})
	switch {
	case err != nil:
		return snapshot.Tree{}, fmt.Sprintf("The working tree could not be observed: %v.", err)
	case h.Uncertain:
		return snapshot.Tree{}, fmt.Sprintf("The working tree could not be observed completely: %s. Try again later.", h.Reason)
	case len(h.Unsupported) > 0:
		return snapshot.Tree{}, fmt.Sprintf("The working tree holds unsupported entries: %s.", listNames(h.Unsupported))
	}
	e.mu.Lock()
	accepted := m.p.accepted
	e.mu.Unlock()
	composite, err := overlay(accepted, h.Tree, m.Path, e.cfg.OwnerOf)
	if err != nil {
		return snapshot.Tree{}, err.Error()
	}
	return composite, ""
}

// probeOutcome classifies a receipt: passed only for an observed zero exit of
// exactly this request; failed for an observed nonzero exit of exactly this
// request from the probe's own assertions; error otherwise, including an exit
// that says the probe could not be executed (126, 127) or was killed by a
// signal (128 and above), such as for exhausted memory.
func probeOutcome(rec CheckReceipt, err error, req CheckRequest) string {
	match := rec.ID == req.ID && rec.ImageID == req.ImageID && rec.Tree == req.Tree.Digest()
	switch {
	case err == nil && rec.Passed(req):
		return ProbePassed
	case err == nil && match && rec.Started && rec.Exited && !rec.TimedOut && !rec.Uncertain && rec.ExitCode != 126 && rec.ExitCode != 127 && rec.ExitCode < 128:
		return ProbeFailed
	}
	return ProbeError
}

func describeOutcome(r *ProbeReceipt) string {
	switch r.Outcome {
	case ProbePassed:
		return "passed (the probe exited 0)"
	case ProbeFailed:
		return fmt.Sprintf("failed (the probe exited %d)", r.ExitCode)
	}
	reason := r.Reason
	if reason == "" {
		reason = "no outcome was established"
	}
	return "error: " + reason
}

func short(digest string) string { return digest[:min(12, len(digest))] }

// evidenceOf validates the evidence items of one coverage entry. A receipt must
// be the submitter's own, for this Definition, passed, and run on the stand the
// submission is bound to; stand is empty in a Pass with writes, whose stand is
// known only after the submitter's writers stopped (bindEvidence completes the
// check then). A source names a file and the inspected location with the
// reasoning; inference alone is no item. A receipt item takes its claim from
// the receipt, and one receipt supports one item of a submission (cited).
func (e *engine) evidenceOf(m *member, i int, c coverage, stand *snapshot.Tree, cited map[string]bool) ([]evidenceItem, error) {
	if len(c.Evidence) > maxToolItems {
		return nil, fmt.Errorf("obligations[%d]: at most %d evidence items", i, maxToolItems)
	}
	var out []evidenceItem
	for j, item := range c.Evidence {
		name := fmt.Sprintf("obligations[%d].evidence[%d]", i, j)
		switch {
		case item.Receipt != "" && item.Source != "":
			return nil, fmt.Errorf("%s cites either a receipt or a source, not both", name)
		case item.Receipt != "":
			r := e.receiptLocked(strings.TrimSpace(item.Receipt))
			switch {
			case r == nil:
				return nil, fmt.Errorf("%s: %q is no probe receipt of this run; a receipt comes only from the tool check", name, item.Receipt)
			case r.Requester != m.Path:
				return nil, fmt.Errorf("%s: %s was requested by %s; cite your own receipts", name, r.ID, r.Requester)
			case r.Obligation != c.Definition:
				return nil, fmt.Errorf("%s: %s assesses %s, not %s", name, r.ID, r.Obligation, c.Definition)
			case r.Outcome != ProbePassed:
				return nil, fmt.Errorf("%s: %s %s; an adverse receipt cannot support %s", name, r.ID, describeOutcome(r), c.Status)
			case stand != nil && r.Stand != stand.Digest().String():
				return nil, fmt.Errorf("%s: %s ran on the stand %s, not on %s; a stale receipt supports nothing here", name, r.ID, short(r.Stand), short(stand.Digest().String()))
			case cited[r.ID]:
				return nil, fmt.Errorf("%s: %s is already cited for another claim; one receipt supports one claim", name, r.ID)
			}
			cited[r.ID] = true
			item.Claim, item.Receipt, item.Source, item.Location, item.Reasoning = r.Claim, r.ID, "", "", ""
			item.Stand = r.Stand
		case item.Source != "":
			claim, err := toolText(name+".claim", item.Claim)
			if err != nil {
				return nil, err
			}
			item.Claim = claim
			path, err := snapshot.NewPath(strings.TrimSpace(item.Source))
			if err != nil {
				return nil, fmt.Errorf("%s.source must be a repository-relative file path", name)
			}
			if item.Location, err = toolText(name+".location", item.Location); err != nil {
				return nil, err
			}
			if item.Reasoning, err = toolText(name+".reasoning", item.Reasoning); err != nil {
				return nil, err
			}
			item.Source = path.String()
			if stand != nil {
				entry, ok := stand.Lookup(path)
				if !ok {
					return nil, fmt.Errorf("%s: %s is no file of the stand %s", name, path, short(stand.Digest().String()))
				}
				item.Stand, item.Digest = stand.Digest().String(), entry.ContentID().String()
			}
		default:
			return nil, fmt.Errorf("%s cites a probe receipt (receipt) or an inspected source (source, location, reasoning)", name)
		}
		out = append(out, item)
	}
	return out, nil
}

// bindEvidence binds the evidence of a submission to its checked stand: every
// cited receipt ran on exactly that stand and every cited source is a file of
// it. It returns the findings that return the submission to its owner.
func (e *engine) bindEvidence(pr *proposal, stand snapshot.Tree) []Finding {
	m := pr.member
	digest := stand.Digest().String()
	var out []Finding
	for i := range pr.env.Obligations {
		c := &pr.env.Obligations[i]
		for j := range c.Evidence {
			item := &c.Evidence[j]
			if item.Receipt != "" {
				if item.Stand != digest {
					out = append(out, Finding{Owner: m.Path, Obligation: "cited receipts ran on the submitted stand", Observed: fmt.Sprintf("%s of %s ran on %s; your submitted area composed with the accepted stand is %s. Probe again after your last change, or cite another item.", item.Receipt, c.Definition, short(item.Stand), short(digest))})
				}
				continue
			}
			path, _ := snapshot.NewPath(item.Source)
			entry, ok := stand.Lookup(path)
			if !ok {
				out = append(out, Finding{Owner: m.Path, Obligation: "cited sources exist in the submitted stand", Observed: fmt.Sprintf("%s cited for %s is no file of the submitted stand", item.Source, c.Definition)})
				continue
			}
			item.Stand, item.Digest = digest, entry.ContentID().String()
		}
	}
	return out
}

// dispositionsOf validates the dispositions of adverse receipts a review
// submission makes: receipts on its candidate requested within the submitter's
// responsibility and not yet disposed of.
func (e *engine) dispositionsOf(m *member, in []dispositionIn) (map[string]ProbeDisposition, error) {
	if len(in) == 0 {
		return nil, nil
	}
	if !m.p.spec.Review {
		return nil, fmt.Errorf("dispositions belong to an independent review")
	}
	if len(in) > maxToolItems {
		return nil, fmt.Errorf("at most %d dispositions per report", maxToolItems)
	}
	candidate := m.p.tree.Digest().String()
	out := map[string]ProbeDisposition{}
	listed := map[string]bool{}
	for i, d := range in {
		name := fmt.Sprintf("dispositions[%d]", i)
		r := e.receiptLocked(strings.TrimSpace(d.Receipt))
		switch {
		case r == nil:
			return nil, fmt.Errorf("%s: %q is no probe receipt of this run", name, d.Receipt)
		case !r.adverse():
			return nil, fmt.Errorf("%s: %s passed; only an adverse receipt needs a disposition", name, r.ID)
		case r.Stand != candidate:
			return nil, fmt.Errorf("%s: %s ran on another stand than this candidate", name, r.ID)
		case !e.answersForLocked(r, m.Path):
			return nil, fmt.Errorf("%s: %s was requested by %s about %s, both outside your responsibility", name, r.ID, r.Requester, r.Obligation)
		case r.disposed():
			return nil, fmt.Errorf("%s: %s is already disposed of", name, r.ID)
		case r.Disposition != nil && r.Requester == m.Path && r.Pass == m.p.id:
			return nil, fmt.Errorf("%s: your disproof of %s awaits an independent assessment in a later review", name, r.ID)
		case listed[r.ID]:
			return nil, fmt.Errorf("%s: %s is listed twice", name, r.ID)
		}
		listed[r.ID] = true
		x := ProbeDisposition{As: d.As, By: m.Path, Pass: m.p.id}
		switch d.As {
		case DisposedExpectationError:
			// A failed assertion, or an error within the probe's own execution
			// that no exact rerun passes, such as a probe too large for the
			// check's time limit. A timeout also looks like a hang of the
			// candidate, so an error names a passing receipt of the same
			// obligation that establishes the claim without the erring probe.
			// The same independent assessment decides both.
			if r.Outcome == ProbeError {
				if !probeCaused(r) {
					return nil, fmt.Errorf("%s: %s erred outside the probe's own execution (%s); only a passing exact rerun (probe.rerun) disposes of it", name, r.ID, describeOutcome(r))
				}
				passing := e.receiptLocked(strings.TrimSpace(d.Passing))
				if passing == nil || passing.Outcome != ProbePassed || passing.Stand != r.Stand || passing.Obligation != r.Obligation || !e.answersForLocked(passing, m.Path) {
					return nil, fmt.Errorf("%s: an error disposed of as expectation_error names in passing a passing receipt of %s on this candidate that establishes the claim without the erring probe", name, r.Obligation)
				}
				x.Passing = passing.ID
			}
			reason, err := toolText(name+".reason", d.Reason)
			if err != nil {
				return nil, err
			}
			evidence, err := toolText(name+".evidence", d.Evidence)
			if err != nil {
				return nil, fmt.Errorf("%v: a disproof states the evidence that the probe, not the candidate, was wrong", err)
			}
			x.Reason, x.Evidence = reason, evidence
			// Root's disproof of its own expectation in the same Pass has no
			// independent assessor yet.
			x.Pending = r.Requester == m.Path && r.Pass == m.p.id && m.Parent == ""
			if r.Outcome == ProbeError {
				for _, y := range e.erringRepetitionsLocked(r, m.Path) {
					if !listed[y.ID] && probeCaused(y) {
						out[y.ID] = x
					}
				}
			}
		case DisposedRerun:
			if r.Outcome != ProbeError {
				return nil, fmt.Errorf("%s: %s failed its assertion; a failing assertion is disposed of by a finding or a reasoned expectation_error, never by running it again", name, r.ID)
			}
			root := r.ID
			if r.Rerun != "" {
				root = r.Rerun
			}
			later := e.receiptLocked(strings.TrimSpace(d.Rerun))
			if later == nil || later.Rerun != root || later.Outcome != ProbePassed || later.Stand != r.Stand {
				return nil, fmt.Errorf("%s: rerun names a passing receipt that repeated %s exactly (probe.rerun) on the same candidate", name, root)
			}
			x.Rerun = later.ID
			for _, y := range e.erringRepetitionsLocked(r, m.Path) {
				if !listed[y.ID] {
					out[y.ID] = x
				}
			}
		default:
			return nil, fmt.Errorf("%s.as must be %q or %q; a violation is disposed of by a genuine finding that cites the receipt", name, DisposedExpectationError, DisposedRerun)
		}
		out[r.ID] = x
	}
	return out, nil
}

// probeCaused reports whether an error receipt ended within the probe's own
// execution: it ran out of its time limit, or it exited saying it could not run
// or was killed. Any other error is the Runtime's and only an exact rerun can
// replace it.
func probeCaused(r *ProbeReceipt) bool {
	return r.Started && (r.TimedOut || r.Exited && (r.ExitCode == 126 || r.ExitCode == 127 || r.ExitCode >= 128))
}

// erringRepetitionsLocked lists the other erring, undisposed runs of r's
// original probe on r's stand within the responsibility: one disposition of an
// error disposes of its repetitions too, while another requester's receipt
// stays for its own reviewer.
func (e *engine) erringRepetitionsLocked(r *ProbeReceipt, responsibility string) []*ProbeReceipt {
	root := r.ID
	if r.Rerun != "" {
		root = r.Rerun
	}
	var out []*ProbeReceipt
	for _, y := range e.receipts {
		if (y.ID == root || y.Rerun == root) && y.Outcome == ProbeError && y.Stand == r.Stand && y.Disposition == nil && y.ID != r.ID && e.answersForLocked(y, responsibility) {
			out = append(out, y)
		}
	}
	return out
}

// receiptsOf validates the receipts a finding cites.
func (e *engine) receiptsOf(m *member, name string, ids []string) ([]string, error) {
	var out []string
	for j, id := range ids {
		r := e.receiptLocked(strings.TrimSpace(id))
		switch {
		case r == nil:
			return nil, fmt.Errorf("%s.receipts[%d]: %q is no probe receipt of this run", name, j, id)
		case !e.answersForLocked(r, m.Path):
			return nil, fmt.Errorf("%s.receipts[%d]: %s was requested by %s about %s, both outside your responsibility", name, j, r.ID, r.Requester, r.Obligation)
		case m.p.spec.Review && r.Stand != m.p.tree.Digest().String():
			return nil, fmt.Errorf("%s.receipts[%d]: %s ran on another stand than this candidate", name, j, r.ID)
		case slices.Contains(out, r.ID):
			return nil, fmt.Errorf("%s.receipts[%d]: %s is cited twice", name, j, r.ID)
		}
		out = append(out, r.ID)
	}
	return out, nil
}

// answersForLocked reports whether a reviewer of a responsibility may dispose
// of a receipt: it requested it, a Manager below it did, or the receipt assesses
// an obligation of its responsibility. The last lets the owner's reviewer judge
// an ancestor's failed probe of its own claim independently, so that Root's
// probe of a child's obligation needs no further complete Verify.
func (e *engine) answersForLocked(r *ProbeReceipt, responsibility string) bool {
	if e.within(r.Requester, responsibility) {
		return true
	}
	d, ok := e.cfg.Definition(r.Obligation)
	return ok && e.within(d.Owner, responsibility)
}

// undisposedLocked lists the adverse receipts on a stand, requested within a
// responsibility, that have no effective disposition. A disposition that awaits
// an independent assessment counts only when it was made in the Pass named by
// pending: its author has done its part there, and a later review must still
// assess it. disposing and citing are a submission's own dispositions and
// findings, which take effect with its acceptance.
func (e *engine) undisposedLocked(stand, responsibility, pending string, disposing map[string]ProbeDisposition, citing []Finding) []*ProbeReceipt {
	var out []*ProbeReceipt
	for _, r := range e.receipts {
		if !r.adverse() || r.Stand != stand || !e.within(r.Requester, responsibility) || r.disposed() {
			continue
		}
		if d, ok := disposing[r.ID]; ok && (!d.Pending || pending != "") {
			continue
		}
		if slices.ContainsFunc(citing, func(f Finding) bool { return f.Blocking && slices.Contains(f.Receipts, r.ID) }) {
			continue
		}
		if r.Disposition != nil && pending != "" && r.Disposition.Pass == pending {
			continue
		}
		out = append(out, r)
	}
	return out
}

// takeDispositionsLocked records the dispositions and the finding citations of
// an accepted review submission.
func (e *engine) takeDispositionsLocked(pr *proposal, findings []*Finding) {
	p := pr.member.p
	set := func(r *ProbeReceipt, d ProbeDisposition) {
		r.Disposition = &d
		_ = e.record(Event{Kind: "probe.disposed", Pass: p.id, Manager: d.By, Data: map[string]any{"receipt": r.ID, "disposition": d}})
	}
	for _, f := range findings {
		for _, id := range f.Receipts {
			if r := e.receiptLocked(id); r != nil && r.adverse() && f.Blocking && !r.disposed() {
				set(r, ProbeDisposition{As: DisposedViolation, By: f.RaisedBy, Pass: p.id, Proposal: pr.id, Finding: f.ID})
			}
		}
	}
	for _, id := range slices.Sorted(mapKeys(pr.dispositions)) {
		r := e.receiptLocked(id)
		if r == nil || r.disposed() {
			continue
		}
		d := pr.dispositions[id]
		d.Proposal = pr.id
		set(r, d)
	}
}

// voidDispositionsLocked withdraws the expectation-error and rerun
// dispositions an accepted submission made once that submission is reopened or
// invalidated: they unblocked adverse receipts on its authority, which no longer
// stands. A violation stays with its recorded finding.
func (e *engine) voidDispositionsLocked(pr *proposal, why string) {
	for _, r := range e.receipts {
		if d := r.Disposition; d != nil && d.Proposal == pr.id && d.As != DisposedViolation {
			r.Disposition = nil
			_ = e.record(Event{Kind: "probe.disposed", Pass: pr.member.p.id, Manager: d.By, Data: map[string]any{"receipt": r.ID, "voided": d, "why": why}})
		}
	}
}

// writeReceipt renders a receipt for an assessor: its binding, outcome,
// disposition and the beginning of its probe bytes and output.
func writeReceipt(b *strings.Builder, r *ProbeReceipt, full bool) {
	fmt.Fprintf(b, "Receipt %s by %s in %s for %s: %s\n", r.ID, r.Requester, r.Pass, r.Obligation, describeOutcome(r))
	fmt.Fprintf(b, "  Claim: %s\n  Expected: %s\n", r.Claim, r.Expected)
	stand := r.StandKind + " " + short(r.Stand)
	if r.Proposal != "" {
		stand += " (" + r.Proposal + ")"
	}
	fmt.Fprintf(b, "  Ran on: %s; probe %s (sha256 %s) with %s %s in %s\n", stand, r.Path, short(r.ContentDigest), r.Executable, strings.Join(r.Args, " "), r.Directory)
	if d := r.Disposition; d != nil {
		status := d.As + " by " + d.By + " in " + d.Pass
		if d.Pending {
			status += ", awaiting an independent assessment"
		}
		switch {
		case d.Finding != "":
			status += ", finding " + d.Finding
		case d.Rerun != "":
			status += ", replaced by " + d.Rerun
		case d.Reason != "":
			status += ": " + d.Reason + " Evidence: " + d.Evidence
		}
		fmt.Fprintf(b, "  Disposition: %s\n", status)
	} else if r.adverse() {
		b.WriteString("  Disposition: none\n")
	}
	content, output := r.Content, r.Output
	if !full {
		content = clip(content, probeShown)
		output = clip(output, probeShown)
	}
	fmt.Fprintf(b, "  Probe:\n    %s\n", strings.ReplaceAll(strings.TrimRight(content, "\n"), "\n", "\n    "))
	if output != "" {
		fmt.Fprintf(b, "  Output:\n    %s\n", strings.ReplaceAll(strings.TrimRight(output, "\n"), "\n", "\n    "))
	}
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "\n[cut; look_up topic probe for the whole receipt]"
}

// writeEvidence renders the evidence items of one coverage entry.
func writeEvidence(b *strings.Builder, items []evidenceItem) {
	for _, item := range items {
		claim := strings.ReplaceAll(item.Claim, "\n", " ")
		if item.Receipt != "" {
			fmt.Fprintf(b, "    Claim %q: executed, receipt %s\n", claim, item.Receipt)
			continue
		}
		where := item.Source
		if item.Digest != "" {
			where += " (sha256 " + short(item.Digest) + ")"
		}
		fmt.Fprintf(b, "    Claim %q: source inspection of %s at %s: %s\n", claim, where, strings.ReplaceAll(item.Location, "\n", " "), strings.ReplaceAll(item.Reasoning, "\n", "\n      "))
	}
}

// probesOfLocked lists the receipts a Manager requested in a Pass.
func (e *engine) probesOfLocked(manager, pass string) []*ProbeReceipt {
	var out []*ProbeReceipt
	for _, r := range e.receipts {
		if r.Requester == manager && r.Pass == pass {
			out = append(out, r)
		}
	}
	return out
}
