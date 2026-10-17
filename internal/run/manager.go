package run

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"slices"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// This is an initial-assessment protocol, not work or publication authority.
const initialExplore = "initial-explore"
const exploreInputMedia = "application/vnd.markitect.explore-input+json"
const exploreReportMedia = "application/vnd.markitect.explore-report+json"

// ManagerRecord associates the existing atomic session/step preparation with a
// queued responsibility. Zero is omitted from historical command JSON.
type ManagerRecord struct {
	Manager   string
	Parent    *string
	HandoffID string
	Phase     string
}

func (m ManagerRecord) validate() error {
	if m == (ManagerRecord{}) {
		return nil
	}
	if !text(m.Manager) || !identifier(m.HandoffID) || m.Phase != initialExplore || (m.Manager == organization.Root) != (m.Parent == nil) || m.Parent != nil && !text(*m.Parent) {
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

// ExploreContext retains actual producer bytes and product identities, inline.
// These identities are recorded producer assertions, not authenticated authority.
type ExploreContext struct {
	View                          organization.ContextView
	Package, ModelID, ModelDigest string
	Format, Template              organization.ProductText
	ContentID                     snapshot.ContentID
	Bytes                         []byte
}

// ExplorePacket has no dereferenceable nested payload references. Its own input
// identity is the existing StepPlan.Input, outside these self-contained bytes.
type ExplorePacket struct {
	Version                 uint32
	Assignment, Run         string
	RunBasis                snapshot.ContentID
	Manager                 string
	Parent                  *string
	HandoffID, Phase        string
	Session                 SessionID
	SessionBinding          snapshot.ContentID
	Step                    StepID
	Operation               operationText
	Context                 ExploreContext
	OutputContractContentID snapshot.ContentID
}

// ExploreReport is an Agent explanation only. Explicit uncertainty/questions may
// complete initial participation, never prerequisites, review or a delivery.
type ExploreReport struct {
	Version                                                  uint32
	Assignment, Run                                          string
	RunBasis                                                 snapshot.ContentID
	Manager                                                  string
	Parent                                                   *string
	HandoffID                                                string
	Session                                                  SessionID
	SessionBinding                                           snapshot.ContentID
	Step                                                     StepID
	InputContentID, OperationContentID, ContextContentID     snapshot.ContentID
	Summary                                                  string
	ProposedChanges, Prerequisites, Uncertainties, Questions []string
}

func commandDigest(c Command) [32]byte { b, _ := json.Marshal(c); return sha256.Sum256(b) }
func sameParent(a, b *string) bool     { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func recordFor(v managerValue) ManagerRecord {
	m := ManagerRecord{Manager: v.Slot.Package, HandoffID: v.HandoffID, Phase: initialExplore}
	if v.Slot.Parent != "" {
		p := v.Slot.Parent
		m.Parent = &p
	}
	return m
}
func sameManager(a, b ManagerRecord) bool {
	return a.Manager == b.Manager && a.HandoffID == b.HandoffID && a.Phase == b.Phase && sameParent(a.Parent, b.Parent)
}

// PrepareExploreSessionCommand admits only a queued initial RO Manager session.
// Writing remains the expense class protecting completion capacity; it grants
// no write effect. Actual workspace isolation remains a Host obligation.
func (c *Coordinator) PrepareExploreSessionCommand(meta RecordMeta, manager string, plan SessionPlan, reservation Reservation, profile, tools Payload) (Command, error) {
	v, ok := c.state.managers[manager]
	if !ok || c.state.basisVersion != 3 {
		return Command{}, ErrState
	}
	cmd := PrepareSessionCommand(meta, c.Current().Assignment, c.Current().ID, plan, reservation, profile, tools)
	cmd.Manager = recordFor(v)
	cmd.managerProof = commandDigest(cmd)
	return cmd, nil
}

// ExplorePreparation supplies actual producer inputs. Live preparation compiles
// original model bytes, resolves settings again, and matches the whole resulting
// binding before calling SelectContext and RenderContext. Raw host JSON cannot
// substitute for that checked path. Nothing is sent or acquired here.
type ExplorePreparation struct {
	Binding  BindingInput
	Settings organization.SettingsFacts
}

func (c *Coordinator) PrepareExploreStepCommand(meta RecordMeta, plan StepPlan, reservation Reservation, input ExplorePreparation) (Command, error) {
	v, ok := c.state.managerForSession(plan.Session)
	if !ok || c.state.basisVersion != 3 {
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
	selection, err := organization.SelectContext(compiled, v.Slot.Package, input.Binding.Scope)
	if err != nil {
		return Command{}, err
	}
	source := organization.HostSource{RecordID: v.HandoffID, ContentID: c.Current().Basis.ContentID}
	_, operation := organization.ApplyOperation()
	renderInput := organization.RenderInput{
		View: organization.ManagerView, Process: &organization.ManagerProcess{Phase: organization.ManagerExplore, Source: source}, InputBinding: source,
		Task:            organization.TaskFacts{ID: v.HandoffID, Package: v.Slot.Package, Outcome: organization.HostText{Text: operation, Source: source}, Findings: organization.OpenFindings{Source: source}},
		MaxContextBytes: input.Binding.MaxContextBytes,
	}
	if v.Slot.Package == organization.Root {
		renderInput.OwnerInput = &organization.HostText{Text: string(bound.owner.Bytes()), Source: organization.HostSource{RecordID: c.Current().Assignment, ContentID: bound.owner.Ref().ContentID}}
	}
	for _, registration := range input.Binding.Registry.Entries() {
		owner := ownerName(registration.Owner())
		if owner != v.Slot.Package {
			continue
		}
		path := registration.Artifact().Path()
		entry, found := input.Binding.Census.Lookup(path)
		if !found {
			return Command{}, ErrInvalid
		}
		fact := organization.ArtifactFact{Package: owner, Path: path, ContentID: entry.ContentID(), Content: organization.ArtifactBinaryRef, Source: organization.HostSource{RecordID: c.Current().Basis.ContentID.String(), ContentID: entry.ContentID(), Field: path.String()}}
		if utf8.Valid(entry.Bytes()) {
			text := string(entry.Bytes())
			fact.Content = organization.ArtifactText
			fact.Text = &text
		}
		renderInput.OwnArtifacts = append(renderInput.OwnArtifacts, fact)
	}
	rendered, err := organization.RenderContext(selection, settings, renderInput)
	if err != nil {
		return Command{}, err
	}
	r := recordFor(v)
	packet := ExplorePacket{Version: 1, Assignment: c.Current().Assignment, Run: c.Current().ID, RunBasis: c.Current().Basis.ContentID,
		Manager: r.Manager, Parent: r.Parent, HandoffID: r.HandoffID, Phase: r.Phase, Session: plan.Session, SessionBinding: plan.SessionBinding, Step: plan.ID,
		Operation:               c.state.managerBasis.ManagerPolicy.Operation,
		Context:                 ExploreContext{View: rendered.View(), Package: rendered.Package(), ModelID: selection.ModelID(), ModelDigest: selection.ModelDigest(), Format: rendered.Format(), Template: rendered.Template(), ContentID: rendered.ContentID(), Bytes: rendered.Bytes()},
		OutputContractContentID: exploreOutputContract().Ref().ContentID,
	}
	data, err := json.Marshal(packet)
	if err != nil {
		return Command{}, err
	}
	payload, err := NewPayload(exploreInputMedia, "utf-8", data)
	if err != nil {
		return Command{}, err
	}
	contract := exploreOutputContract()
	plan.Input = payload.Ref()
	plan.OutputContract = contract.Ref()
	cmd := PrepareStepCommand(meta, c.Current().Assignment, c.Current().ID, plan, reservation, payload, contract)
	cmd.Manager = r
	cmd.managerProof = commandDigest(cmd)
	return cmd, nil
}

// The contract is a closed format identity, not a provider schema interpreter.
// Its complete field/type catalogue is retained with the admitted step.
func exploreOutputContract() Payload {
	data := []byte(`{"Format":"markitect.explore-report/v1","Fields":{"Version":"integer=1","Assignment":"string","Run":"string","RunBasis":"ContentID","Manager":"string","Parent":"string|null","HandoffID":"string","Session":"SessionID","SessionBinding":"ContentID","Step":"StepID","InputContentID":"ContentID","OperationContentID":"ContentID","ContextContentID":"ContentID","Summary":"nonblank-string","ProposedChanges":"string[]","Prerequisites":"string[]","Uncertainties":"string[]","Questions":"string[]"},"AdditionalFields":false}`)
	p, _ := NewPayload("application/vnd.markitect.explore-contract+json", "utf-8", data)
	return p
}

// NewExploreReportPayload retains a proposal, not a Host-accepted result.
// The reducer separately checks its association with an actual observed step.
func NewExploreReportPayload(report ExploreReport) (Payload, error) {
	if report.Version != 1 || !text(report.Summary) || report.ProposedChanges == nil || report.Prerequisites == nil || report.Uncertainties == nil || report.Questions == nil {
		return Payload{}, ErrInvalid
	}
	for _, list := range [][]string{report.ProposedChanges, report.Prerequisites, report.Uncertainties, report.Questions} {
		for _, item := range list {
			if !text(item) {
				return Payload{}, ErrInvalid
			}
		}
	}
	data, err := json.Marshal(report)
	if err != nil {
		return Payload{}, err
	}
	return NewPayload(exploreReportMedia, "utf-8", data)
}
func decodeExploreReport(p Payload, packet ExplorePacket, input snapshot.ContentID) (ExploreReport, error) {
	var r ExploreReport
	if p.Ref().MediaType != exploreReportMedia || p.Ref().Encoding != "utf-8" || strictJSON(p.Bytes(), &r) != nil {
		return r, ErrInvalid
	}
	checked, err := NewExploreReportPayload(r)
	if err != nil || !bytes.Equal(checked.Bytes(), p.Bytes()) || r.Assignment != packet.Assignment || r.Run != packet.Run || r.RunBasis != packet.RunBasis || r.Manager != packet.Manager || !sameParent(r.Parent, packet.Parent) || r.HandoffID != packet.HandoffID || r.Session != packet.Session || r.SessionBinding != packet.SessionBinding || r.Step != packet.Step || r.InputContentID != input || r.OperationContentID != packet.Operation.Product.ContentID || r.ContextContentID != packet.Context.ContentID {
		return ExploreReport{}, ErrInvalid
	}
	return r, nil
}
func payloadFor(c Command, ref BlobRef) (Payload, bool) {
	for _, p := range c.sessionPayloads {
		if p.Ref() == ref && p.Valid() {
			return p, true
		}
	}
	return Payload{}, false
}
func copyReport(r ExploreReport) ExploreReport {
	r.ProposedChanges = slices.Clone(r.ProposedChanges)
	r.Prerequisites = slices.Clone(r.Prerequisites)
	r.Uncertainties = slices.Clone(r.Uncertainties)
	r.Questions = slices.Clone(r.Questions)
	if r.Parent != nil {
		p := *r.Parent
		r.Parent = &p
	}
	return r
}
func validateExplorePacket(p Payload, c Command, v managerValue, s stateValue) (ExplorePacket, error) {
	var packet ExplorePacket
	if p.Ref().MediaType != exploreInputMedia || p.Ref().Encoding != "utf-8" || strictJSON(p.Bytes(), &packet) != nil {
		return packet, ErrInvalid
	}
	r := recordFor(v)
	q := packet.Context
	operation, _ := json.Marshal(packet.Operation)
	expected, _ := json.Marshal(s.managerBasis.ManagerPolicy.Operation)
	if packet.Version != 1 || packet.Assignment != c.Assignment || packet.Run != c.Run || packet.RunBasis != s.current.Basis.ContentID || !sameManager(ManagerRecord{packet.Manager, packet.Parent, packet.HandoffID, packet.Phase}, r) || packet.Session != c.Session.Session || packet.SessionBinding != c.Session.StepPlan.SessionBinding || packet.Step != c.Session.Step || !bytes.Equal(operation, expected) || packet.OutputContractContentID != c.Session.StepPlan.OutputContract.ContentID || q.View != organization.ManagerView || q.Package != r.Manager || !text(q.ModelID) || !text(q.ModelDigest) || len(q.Bytes) == 0 || q.ContentID != snapshot.ContentID(sha256.Sum256(q.Bytes)) || !text(q.Format.ID) || !text(q.Format.Version) || q.Format.ContentID == (snapshot.ContentID{}) || !text(q.Template.ID) || !text(q.Template.Version) || q.Template.ContentID == (snapshot.ContentID{}) {
		return ExplorePacket{}, ErrInvalid
	}
	local := false
	for _, scope := range s.managerBasis.Compilation {
		if scope.Local && scope.ID == q.ModelID && scope.Digest == q.ModelDigest {
			local = true
		}
	}
	if !local {
		return ExplorePacket{}, ErrInvalid
	}
	contract, ok := payloadFor(c, c.Session.StepPlan.OutputContract)
	if !ok || contract.Ref() != exploreOutputContract().Ref() {
		return ExplorePacket{}, ErrInvalid
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
