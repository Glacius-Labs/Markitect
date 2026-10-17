package run

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const stepInputMedia = "application/vnd.markitect.step-input+json"

// ManagerRecord binds a session or step preparation to one task and phase.
// Zero is omitted from historical command JSON.
type ManagerRecord struct {
	Manager string
	Parent  *string
	Task    string
	Phase   Phase
}

var managerPhases = []Phase{PhaseExplore, PhaseWork, PhaseIntegrate, PhaseReviewerExpectations, PhaseReviewerCandidate}

func (m ManagerRecord) validate() error {
	if m == (ManagerRecord{}) {
		return nil
	}
	if !text(m.Manager) || !identifier(m.Task) || !slices.Contains(managerPhases, m.Phase) || (m.Manager == organization.Root) != (m.Parent == nil) || m.Parent != nil && !text(*m.Parent) {
		return ErrInvalid
	}
	return nil
}

type operationText struct {
	Product organization.ProductText
	Bytes   []byte
}
type managerPolicy struct {
	Version   uint32
	Operation operationText
}
type reviewDemand struct{ Subjects, Starts, Steps int64 }

func liveManagerPolicy() *managerPolicy {
	product, text := organization.ApplyOperation()
	return &managerPolicy{1, operationText{product, []byte(text)}}
}
func managerReviewDemand(managers []organization.Manager) reviewDemand {
	subjects := int64(len(managers))
	for _, m := range managers {
		if len(m.Children) > 0 {
			subjects++
		}
	}
	return reviewDemand{subjects, 2 * subjects, 2 * subjects}
}
func recordedReviewDemand(w bindingWire) reviewDemand {
	if w.Version == 3 {
		return managerReviewDemand(w.Managers)
	}
	n := reviewFloor(w.Managers)
	return reviewDemand{n, n, n}
}
func validateManagerPolicy(w bindingWire) error {
	if w.Version != 3 {
		if w.ManagerPolicy != nil || w.Scope != nil || w.MaxContextBytes != nil {
			return ErrInvalid
		}
		return nil
	}
	if w.MaxContextBytes == nil || w.MaxContextBytes.Value <= 0 || !recordedSettingSource(w, w.MaxContextBytes.Source) {
		return ErrInvalid
	}
	if w.Scope != nil {
		if *w.Scope == nil {
			return ErrInvalid
		}
		for _, scope := range *w.Scope {
			if !recordedSettingSource(w, scope.Source) || !text(scope.Value.Kind.Package) || !text(scope.Value.Kind.ID) || !text(scope.Value.Property.Package) || !text(scope.Value.Property.ID) {
				return ErrInvalid
			}
		}
	}
	p := w.ManagerPolicy
	if p == nil || p.Version != 1 || p.Operation.Product.ID != "markitect/operation/artifact-apply" || p.Operation.Product.Version != "1" || len(p.Operation.Bytes) == 0 || p.Operation.Product.ContentID != snapshot.ContentID(sha256.Sum256(p.Operation.Bytes)) {
		return ErrInvalid
	}
	product, text := organization.ApplyOperation()
	if p.Operation.Product != product || !bytes.Equal(p.Operation.Bytes, []byte(text)) {
		return ErrInvalid
	}
	return nil
}

// StepContext retains the actual rendered context bytes and product
// identities, inline. These identities are recorded producer assertions, not
// authenticated authority.
type StepContext struct {
	View                          organization.ContextView
	Package, ModelID, ModelDigest string
	Format, Template              organization.ProductText
	ContentID                     snapshot.ContentID
	Bytes                         []byte
}

// StepPacket is the complete input of one task step. It has no dereferenceable
// nested payload references; its own identity is StepPlan.Input.
type StepPacket struct {
	Version                 uint32
	Assignment, Run         string
	RunBasis                snapshot.ContentID
	Manager                 string
	Parent                  *string
	Task                    string
	Phase                   Phase
	Session                 SessionID
	SessionBinding          snapshot.ContentID
	Step                    StepID
	Operation               operationText
	Context                 StepContext
	OutputContractContentID snapshot.ContentID
}

func commandDigest(c Command) [32]byte { b, _ := json.Marshal(c); return sha256.Sum256(b) }
func sameParent(a, b *string) bool     { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func sameManager(a, b ManagerRecord) bool {
	return a.Manager == b.Manager && a.Task == b.Task && a.Phase == b.Phase && sameParent(a.Parent, b.Parent)
}

// PrepareTaskSessionCommand admits a new session for the run's next step. The
// plan must carry the phase's tool catalogue, a read-only workspace for
// Explore, the task's provider home and its last conversation for resume.
func (c *Coordinator) PrepareTaskSessionCommand(meta RecordMeta, item Admission, plan SessionPlan, reservation Reservation, profile, tools Payload) (Command, error) {
	next, ok := c.Next()
	t, found := c.state.tasks[item.Manager]
	if !ok || !found || next != item || c.state.basisVersion != 3 {
		return Command{}, ErrState
	}
	cmd := PrepareSessionCommand(meta, c.Current().Assignment, c.Current().ID, plan, reservation, profile, tools)
	cmd.Manager = recordForTask(t, item.Phase)
	cmd.managerProof = commandDigest(cmd)
	return cmd, nil
}

// StepPreparation supplies actual producer inputs. Preparation compiles the
// original model bytes, resolves settings again and matches the whole binding
// before rendering the step's context. Nothing is sent or acquired here.
type StepPreparation struct {
	Binding  BindingInput
	Settings organization.SettingsFacts
}

// StepFacts are the Host facts of one step that the reducer does not hold: the
// version its workspace starts from (core rules D3, T7) and the recorded
// payloads its context quotes, which StepQuotes names.
type StepFacts struct {
	Tree     snapshot.Tree
	Registry realization.Registry
	Quoted   []Payload
}

// StepQuotes names the recorded payloads the context of the manager's current
// step quotes: its children's reports and delivery proposals and, for a
// candidate review, the proposal and the recorded expectations.
func (c *Coordinator) StepQuotes(manager string) []BlobRef {
	t, ok := c.state.tasks[manager]
	if !ok {
		return nil
	}
	var refs []BlobRef
	if t.Phase != PhaseReviewerExpectations {
		for _, child := range t.Slot.Children {
			refs = append(refs, childReports(c.state.tasks[child])...)
		}
	}
	if t.Phase == PhaseReviewerCandidate {
		refs = append(refs, t.Submission.Proposal, t.Review.Expectations)
	}
	return refs
}

func childReports(t taskValue) []BlobRef {
	var refs []BlobRef
	if t.Report != (BlobRef{}) {
		refs = append(refs, t.Report)
	}
	if t.Submission.Proposal != (BlobRef{}) {
		refs = append(refs, t.Submission.Proposal)
	}
	for _, d := range t.Deliveries {
		refs = append(refs, d.Proposal)
	}
	return refs
}

func phaseView(p Phase) organization.ContextView {
	switch p {
	case PhaseReviewerExpectations:
		return organization.ReviewerExpectations
	case PhaseReviewerCandidate:
		return organization.ReviewerCandidate
	}
	return organization.ManagerView
}

// PrepareTaskStepCommand renders the step's input for the task that owns the
// session and records it with the phase's output contract.
func (c *Coordinator) PrepareTaskStepCommand(meta RecordMeta, plan StepPlan, reservation Reservation, input StepPreparation, facts StepFacts) (Command, error) {
	t, ok := c.state.taskForSession(plan.Session)
	if !ok || c.state.basisVersion != 3 || t.Step != "" {
		return Command{}, ErrState
	}
	compiled := model.Compile(input.Binding.ModelInput)
	topology, err := organization.BuildTopology(compiled)
	if err != nil {
		return Command{}, err
	}
	settings, err := organization.ResolveSettings(topology, input.Settings)
	if err != nil {
		return Command{}, err
	}
	input.Binding.Compilation = compiled
	input.Binding.Settings = settings
	bound, err := Bind(input.Binding)
	if err != nil || bound.Ref() != c.Current().Basis {
		return Command{}, ErrInvalid
	}
	selection, err := organization.SelectContext(compiled, t.Slot.Package, input.Binding.Scope)
	if err != nil {
		return Command{}, err
	}
	renderInput, err := c.state.renderInput(t, facts, c.Current().Basis.ContentID)
	if err != nil {
		return Command{}, err
	}
	renderInput.MaxContextBytes = input.Binding.MaxContextBytes
	if t.Slot.Package == organization.Root && renderInput.View == organization.ManagerView {
		renderInput.OwnerInput = &organization.HostText{Text: string(bound.owner.Bytes()), Source: organization.HostSource{RecordID: c.Current().Assignment, ContentID: bound.owner.Ref().ContentID}}
	}
	rendered, err := organization.RenderContext(selection, settings, renderInput)
	if err != nil {
		return Command{}, err
	}
	contract, err := NewStepContract(t.Phase)
	if err != nil {
		return Command{}, err
	}
	r := recordForTask(t, t.Phase)
	packet := StepPacket{Version: 1, Assignment: c.Current().Assignment, Run: c.Current().ID, RunBasis: c.Current().Basis.ContentID,
		Manager: r.Manager, Parent: r.Parent, Task: r.Task, Phase: r.Phase, Session: plan.Session, SessionBinding: plan.SessionBinding, Step: plan.ID,
		Operation:               c.state.managerBasis.ManagerPolicy.Operation,
		Context:                 StepContext{View: rendered.View(), Package: rendered.Package(), ModelID: selection.ModelID(), ModelDigest: selection.ModelDigest(), Format: rendered.Format(), Template: rendered.Template(), ContentID: rendered.ContentID(), Bytes: rendered.Bytes()},
		OutputContractContentID: contract.Ref().ContentID,
	}
	data, err := json.Marshal(packet)
	if err != nil {
		return Command{}, err
	}
	payload, err := NewPayload(stepInputMedia, "utf-8", data)
	if err != nil {
		return Command{}, err
	}
	plan.Input = payload.Ref()
	plan.OutputContract = contract.Ref()
	cmd := PrepareStepCommand(meta, c.Current().Assignment, c.Current().ID, plan, reservation, payload, contract)
	cmd.Manager = r
	cmd.managerProof = commandDigest(cmd)
	return cmd, nil
}

// renderInput projects the task's Host facts into its phase's view. Managers
// see their own artifacts in the step's version, their open findings and their
// children's states and reports. The expectations review sees the model and
// the open findings only; the candidate review also sees the frozen candidate,
// its proposal and the recorded expectations.
func (s stateValue) renderInput(t taskValue, facts StepFacts, basis snapshot.ContentID) (organization.RenderInput, error) {
	source := organization.HostSource{RecordID: t.ID, ContentID: basis}
	_, operation := organization.ApplyOperation()
	quote := func(ref BlobRef) (organization.HostText, error) {
		for _, p := range facts.Quoted {
			if p.Ref() == ref && p.Valid() && utf8.Valid(p.Bytes()) {
				return organization.HostText{Text: string(p.Bytes()), Source: organization.HostSource{RecordID: ref.MediaType, ContentID: ref.ContentID}}, nil
			}
		}
		return organization.HostText{}, ErrInvalid
	}
	findings := s.openFindings(t, source)
	if t.Failure != "" && !t.Phase.reviewer() {
		// The Host's own check of the last step is a finding the Manager sees
		// in its next step (core rule T6).
		id := t.ID + "-last-step"
		findings.IDs = append(findings.IDs, id)
		findings.Items = append(findings.Items, organization.FindingFact{ID: id, Grounds: organization.HostText{Text: "Your last step failed the Host's check: " + t.Failure, Source: organization.HostSource{RecordID: id, ContentID: basis}}})
	}
	in := organization.RenderInput{View: phaseView(t.Phase), InputBinding: source,
		Task: organization.TaskFacts{ID: t.ID, Package: t.Slot.Package, Outcome: organization.HostText{Text: operation, Source: source}, Findings: findings}}
	artifacts, err := ownedArtifacts(t.Slot.Package, facts)
	if err != nil {
		return in, err
	}
	for _, child := range t.Slot.Children {
		fact := organization.ChildFact{Package: child}
		if t.Phase != PhaseReviewerExpectations {
			ct := s.tasks[child]
			fact.State = &organization.HostText{Text: string(ct.State), Source: organization.HostSource{RecordID: ct.ID, ContentID: basis}}
			for _, ref := range childReports(ct) {
				report, err := quote(ref)
				if err != nil {
					return in, err
				}
				fact.Reports = append(fact.Reports, report)
			}
		}
		in.Children = append(in.Children, fact)
	}
	switch t.Phase {
	case PhaseExplore, PhaseWork, PhaseIntegrate:
		phase := map[Phase]organization.ManagerPhase{PhaseExplore: organization.ManagerExplore, PhaseWork: organization.ManagerWork, PhaseIntegrate: organization.ManagerIntegrate}[t.Phase]
		in.Process = &organization.ManagerProcess{Phase: phase, Source: source}
		in.OwnArtifacts = artifacts
	case PhaseReviewerCandidate:
		proposal, err := quote(t.Submission.Proposal)
		if err != nil {
			return in, err
		}
		expectations, err := quote(t.Review.Expectations)
		if err != nil {
			return in, err
		}
		in.Review = &organization.ReviewFacts{Package: t.Slot.Package,
			Subject:      organization.HostSource{RecordID: t.ID + "/" + t.Submission.Purpose, ContentID: t.Submission.Proposal.ContentID},
			Candidate:    organization.HostSource{RecordID: "candidate", ContentID: snapshot.ContentID(facts.Tree.Digest())},
			Expectations: organization.HostSource{RecordID: "expectations", ContentID: t.Review.Expectations.ContentID},
			Files:        artifacts, Evidence: []organization.HostText{proposal, expectations}}
	case PhaseReviewerExpectations:
	default:
		return in, ErrInvalid
	}
	return in, nil
}

// openFindings projects the task's open findings (core rule I1).
func (s stateValue) openFindings(t taskValue, source organization.HostSource) organization.OpenFindings {
	set := organization.OpenFindings{Source: source}
	for _, issue := range s.openIssues(t.Slot.Package) {
		if issue.Kind != issueFinding {
			continue
		}
		set.IDs = append(set.IDs, issue.ID)
		text := fmt.Sprintf("%s %s finding on %s: %s Requested evidence: %s", issue.Severity, issue.Purpose, strings.Join(issue.Definitions, ", "), issue.Grounds, issue.Request)
		set.Items = append(set.Items, organization.FindingFact{ID: issue.ID, Grounds: organization.HostText{Text: text, Source: organization.HostSource{RecordID: issue.ID, ContentID: issue.Source.ContentID}}})
	}
	return set
}

// ownedArtifacts lists the manager's registered artifacts in the step's
// version, with their text where it is UTF-8.
func ownedArtifacts(manager string, facts StepFacts) ([]organization.ArtifactFact, error) {
	var out []organization.ArtifactFact
	version := snapshot.ContentID(facts.Tree.Digest()).String()
	for _, registration := range facts.Registry.Entries() {
		if ownerName(registration.Owner()) != manager {
			continue
		}
		path := registration.Artifact().Path()
		entry, found := facts.Tree.Lookup(path)
		if !found {
			return nil, ErrInvalid
		}
		fact := organization.ArtifactFact{Package: manager, Path: path, ContentID: entry.ContentID(), Content: organization.ArtifactBinaryRef, Source: organization.HostSource{RecordID: version, ContentID: entry.ContentID(), Field: path.String()}}
		if utf8.Valid(entry.Bytes()) {
			text := string(entry.Bytes())
			fact.Content = organization.ArtifactText
			fact.Text = &text
		}
		out = append(out, fact)
	}
	return out, nil
}

// StepResult is the Host's account of one ended step (core rule S4). Input is
// the answered outcome call's recorded input; Candidate is the S4 candidate
// record of a writing step that changed files and Tree its observed tree.
// Aggregate is the aggregate version after the step. Failure, when set, fails
// the step whatever its outcome call said (core rule T6).
type StepResult struct {
	Input, Candidate Payload
	Tree             snapshot.TreeID
	Aggregate        uint64
	AggregateTree    snapshot.TreeID
	Failure          string
}

// ConcludeStepCommand records the end of the task's current step. A session
// the Host abandoned before its step was prepared concludes as failed.
func (c *Coordinator) ConcludeStepCommand(meta RecordMeta, manager string, result StepResult) (Command, error) {
	t, ok := c.state.tasks[manager]
	if !ok || t.Session == "" || c.state.basisVersion != 3 || t.Step == "" && result.Failure == "" {
		return Command{}, ErrState
	}
	call, count := c.state.stepOutcomeCall(t)
	r := ConclusionRecord{Manager: manager, Task: t.ID, Step: t.Step, Aggregate: result.Aggregate, AggregateTree: result.AggregateTree}
	var payloads []Payload
	switch {
	case result.Failure != "":
		r.Outcome, r.Reason = outcomeFailed, result.Failure
	case count == 0:
		r.Outcome, r.Reason = outcomeFailed, "the step ended without an outcome tool call"
	case count == 1:
		if !result.Input.Valid() || result.Input.Ref() != call.Record.Wire.Journal {
			return Command{}, ErrInvalid
		}
		r.Outcome, r.Input = call.Record.Tool, result.Input.Ref()
		payloads = append(payloads, result.Input)
		if result.Candidate.Valid() {
			r.Candidate, r.Tree = result.Candidate.Ref(), result.Tree
			payloads = append(payloads, result.Candidate)
		}
	default:
		return Command{}, ErrState
	}
	cmd := Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: TaskConclude, Assignment: c.Current().Assignment, Run: c.Current().ID, Conclusion: r, sessionPayloads: payloads}
	cmd.managerProof = commandDigest(cmd)
	return cmd, nil
}

func payloadFor(c Command, ref BlobRef) (Payload, bool) {
	for _, p := range c.sessionPayloads {
		if p.Ref() == ref && p.Valid() {
			return p, true
		}
	}
	return Payload{}, false
}

func validateStepPacket(p Payload, c Command, t taskValue, s stateValue) (StepPacket, error) {
	var packet StepPacket
	if p.Ref().MediaType != stepInputMedia || p.Ref().Encoding != "utf-8" || strictJSON(p.Bytes(), &packet) != nil {
		return packet, ErrInvalid
	}
	r := recordForTask(t, t.Phase)
	q := packet.Context
	operation, _ := json.Marshal(packet.Operation)
	expected, _ := json.Marshal(s.managerBasis.ManagerPolicy.Operation)
	if packet.Version != 1 || packet.Assignment != c.Assignment || packet.Run != c.Run || packet.RunBasis != s.current.Basis.ContentID || !sameManager(ManagerRecord{packet.Manager, packet.Parent, packet.Task, packet.Phase}, r) || packet.Session != c.Session.Session || packet.SessionBinding != c.Session.StepPlan.SessionBinding || packet.Step != c.Session.Step || !bytes.Equal(operation, expected) || packet.OutputContractContentID != c.Session.StepPlan.OutputContract.ContentID || q.View != phaseView(t.Phase) || q.Package != r.Manager || !text(q.ModelID) || !text(q.ModelDigest) || len(q.Bytes) == 0 || q.ContentID != snapshot.ContentID(sha256.Sum256(q.Bytes)) || !text(q.Format.ID) || !text(q.Format.Version) || q.Format.ContentID == (snapshot.ContentID{}) || !text(q.Template.ID) || !text(q.Template.Version) || q.Template.ContentID == (snapshot.ContentID{}) {
		return StepPacket{}, ErrInvalid
	}
	local := false
	for _, scope := range s.managerBasis.Compilation {
		if scope.Local && scope.ID == q.ModelID && scope.Digest == q.ModelDigest {
			local = true
		}
	}
	if !local {
		return StepPacket{}, ErrInvalid
	}
	return packet, nil
}

// reviewCapacity checks the version-selected demand without granting any phase.
func reviewCapacity(view BudgetView, demand reviewDemand) bool {
	return !view.Overrun && view.Remaining.Runs >= 1 && view.Remaining.SessionStarts >= demand.Starts && view.Remaining.Steps >= demand.Steps && (demand.Subjects == 0 || view.Remaining.ExecutionTime > 0)
}

func recordedSettingSource(w bindingWire, source organization.SettingSource) bool {
	if !text(source.Field) {
		return false
	}
	for _, d := range w.Documents {
		if d.Source.File == source.File && d.Source.ContentID == source.ContentID && d.Source.Field == "" && snapshot.ContentID(sha256.Sum256(d.Bytes)) == d.Source.ContentID {
			return true
		}
	}
	return false
}
