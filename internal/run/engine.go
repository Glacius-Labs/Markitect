package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const candidateRecordMedia = "application/vnd.markitect.candidate-record+json"

// SessionProfile is the Host profile a role's sessions run with.
type SessionProfile struct {
	ImageID, CredentialRef string
	Profile                Payload
	MaxWireBytes           int64
}

// EngineLimits bound one step's invocations and observations.
type EngineLimits struct {
	StartTimeout, StepTimeout, StopTimeout time.Duration
	HarvestEntries                         int
	HarvestBytes                           int64
	Record                                 realization.CandidateRecordLimits
}

// EngineConfig supplies everything the engine needs from the application.
type EngineConfig struct {
	Coordinator *Coordinator
	Journal     Journal
	Sessions    Sessions
	Workspaces  Workspaces
	Preparation StepPreparation
	Profile     func(manager, role string) (SessionProfile, error)
	Clock       func() time.Time
	Limits      EngineLimits
}

// Engine drives a run's admitted steps through the ports, one at a time (core
// rules S1 to S6, T4). It holds the Host-side facts the reducer does not keep:
// the compiled model, every version of the run aggregate, the path
// reservations and the registry records each task proposed. It decides
// nothing the Coordinator has not admitted, and it stops at the first
// uncertain or failed effect.
type Engine struct {
	cfg        EngineConfig
	compiled   model.Result
	versions   []realization.Aggregate
	reserved   map[string][]realization.PathRequest
	registered map[string]map[snapshot.Path]realization.Registration
}

var errStep = errors.New("step stopped before its conclusion")

// NewEngine binds an engine to its run. It rebuilds the aggregate from the
// captured source and the run's accepted deliveries (core rule D2).
func NewEngine(ctx context.Context, cfg EngineConfig) (*Engine, error) {
	if cfg.Coordinator == nil || cfg.Journal == nil || cfg.Sessions == nil || cfg.Workspaces == nil || cfg.Profile == nil || cfg.Clock == nil {
		return nil, ErrInvalid
	}
	compiled := model.Compile(cfg.Preparation.Binding.ModelInput)
	topology, err := organization.BuildTopology(compiled)
	if err != nil {
		return nil, err
	}
	settings, err := organization.ResolveSettings(topology, cfg.Preparation.Settings)
	if err != nil {
		return nil, err
	}
	areas, err := creationAreas(settings)
	if err != nil {
		return nil, err
	}
	in := cfg.Preparation.Binding
	facts, err := realization.NewPathFacts(compiled, in.Registry, in.Census, areas, in.Protections)
	if err != nil {
		return nil, err
	}
	base, err := realization.NewCandidateBase(compiled, facts)
	if err != nil {
		return nil, err
	}
	aggregate, err := realization.NewAggregate(base)
	if err != nil {
		return nil, err
	}
	e := &Engine{cfg: cfg, compiled: compiled, versions: []realization.Aggregate{aggregate}, reserved: map[string][]realization.PathRequest{}, registered: map[string]map[snapshot.Path]realization.Registration{}}
	var composed []Delivery
	for _, t := range cfg.Coordinator.Tasks() {
		for _, d := range t.Deliveries {
			if d.Candidate != (BlobRef{}) {
				composed = append(composed, d)
			}
		}
	}
	slices.SortFunc(composed, func(a, b Delivery) int { return compareUint(a.Aggregate, b.Aggregate) })
	for _, d := range composed {
		candidate, err := e.candidate(ctx, d.Candidate)
		if err != nil {
			return nil, err
		}
		next, ok := realization.ReplayDelivery(e.aggregate(), candidate).Aggregate()
		if !ok || next.Version() != d.Aggregate {
			return nil, ErrState
		}
		e.versions = append(e.versions, next)
	}
	if uint64(len(e.versions)-1) != cfg.Coordinator.AggregateVersion() {
		return nil, ErrState
	}
	return e, nil
}

func creationAreas(settings organization.Settings) ([]realization.CreationArea, error) {
	var areas []realization.CreationArea
	for _, m := range settings.Managers() {
		owner, err := managerOwner(m.Package)
		if err != nil {
			return nil, err
		}
		for _, directory := range m.CreationRoots.Value {
			area, err := realization.NewCreationArea(owner, directory)
			if err != nil {
				return nil, err
			}
			areas = append(areas, area)
		}
	}
	return areas, nil
}

func managerOwner(manager string) (realization.Owner, error) {
	if manager == organization.Root {
		return realization.RootOwner(), nil
	}
	return realization.PackageOwner(manager)
}

func (e *Engine) aggregate() realization.Aggregate { return e.versions[len(e.versions)-1] }

// Aggregate returns the run's current accepted aggregate.
func (e *Engine) Aggregate() realization.Aggregate { return e.aggregate() }

// Run advances the run until no step is admissible and then records why: the
// root's completion leaves the run Verifying (core rule T13), an owner
// question moves it to Waiting (core rule W1), and an incomplete task, a
// budget that no longer fits the next step or a wait nothing can satisfy
// finishes it Incomplete (core rules T12, T14). Waits that only a parent's
// dispatch could resolve (core rule T12) end the run Incomplete in this
// version.
func (e *Engine) Run(ctx context.Context) error {
	c := e.cfg.Coordinator
	for {
		advanced, err := e.Advance(ctx)
		if errors.Is(err, ErrBudget) {
			return e.finish(ctx, "the budget does not fit the next admissible step")
		}
		if err != nil {
			return err
		}
		if !advanced {
			break
		}
	}
	if c.Current().State != Running {
		return nil
	}
	var incomplete, owner []string
	for _, t := range c.Tasks() {
		switch {
		case t.State == TaskIncomplete:
			incomplete = append(incomplete, t.Manager+": "+t.Reason)
		case t.State == TaskBlocked && slices.ContainsFunc(t.Waits, func(w Wait) bool { return w.Kind == waitOwner }):
			owner = append(owner, t.Manager)
		}
	}
	switch {
	case len(incomplete) > 0:
		return e.finish(ctx, "incomplete tasks: "+strings.Join(incomplete, "; "))
	case len(owner) > 0:
		return e.commit(ctx, Command{ID: e.meta("wait").ID, ObservedAt: e.cfg.Clock().UTC(), Kind: WaitRun, Assignment: c.Current().Assignment, Run: c.Current().ID,
			Reason: "the review of " + strings.Join(owner, ", ") + " needs an owner decision", NextActor: addressedToOwner})
	}
	return e.finish(ctx, "no step is admissible and the remaining waits cannot be satisfied")
}

func (e *Engine) finish(ctx context.Context, reason string) error {
	c := e.cfg.Coordinator
	return e.commit(ctx, Command{ID: e.meta("finish").ID, ObservedAt: e.cfg.Clock().UTC(), Kind: FinishRun, Assignment: c.Current().Assignment, Run: c.Current().ID, Outcome: Incomplete, Reason: reason})
}

// stepVersion is the content a step's workspace starts from.
type stepVersion struct {
	tree     snapshot.Tree
	registry realization.Registry
}

// Advance runs the run's next admissible step to its conclusion. It reports
// false when nothing is admissible.
func (e *Engine) Advance(ctx context.Context) (bool, error) {
	c := e.cfg.Coordinator
	item, ok := c.Next()
	if !ok {
		return false, nil
	}
	task, _ := c.Task(item.Manager)
	version, err := e.stepVersion(ctx, task, item.Phase)
	if err != nil {
		return true, err
	}
	sid := SessionID(e.id("s", task.ID))
	workspace := WorkspaceID(e.id("w", task.ID))
	materialized, err := e.cfg.Workspaces.Materialize(ctx, WorkspaceSpec{ID: workspace, Tree: version.tree})
	if err != nil || materialized.Uncertain || materialized.Tree != version.tree.Digest() {
		return true, errors.Join(errStep, err)
	}
	defer e.cfg.Workspaces.Close(context.WithoutCancel(ctx), workspace)
	role, purpose := stepRole(item.Phase)
	profile, err := e.cfg.Profile(item.Manager, role)
	if err != nil {
		return true, err
	}
	tools, err := ToolCatalog(item.Phase)
	if err != nil {
		return true, err
	}
	plan := SessionPlan{ID: sid, Role: role, Workspace: materialized.Volume, ProviderHome: item.Home, Resume: item.Resume, ImageID: profile.ImageID, CredentialRef: profile.CredentialRef,
		Candidate: version.tree.Digest(), ReadOnly: !item.Phase.writing(), MaxWireBytes: profile.MaxWireBytes, Profile: profile.Profile.Ref(), Tools: tools.Ref()}
	openReservation := Reservation{ID: "open-" + string(sid), Kind: SessionStart, Purpose: purpose, Maximum: e.cfg.Limits.StartTimeout}
	prepared, err := c.PrepareTaskSessionCommand(e.meta("prepare-session"), item, plan, openReservation, profile.Profile, tools)
	if err != nil {
		return true, err
	}
	if err := e.commit(ctx, prepared); err != nil {
		return true, err
	}
	channel, err := c.SessionChannel(e.cfg.Journal, sid, e.tools(task.Manager, item.Phase, version))
	if err != nil {
		return true, err
	}
	admission, err := c.SessionAdmission(ctx, e.cfg.Journal, sid)
	if err != nil {
		return true, err
	}
	began := e.cfg.Clock()
	opened, err := e.cfg.Sessions.Open(ctx, admission, channel)
	if err != nil || opened.Uncertain {
		return true, errors.Join(errStep, err)
	}
	if err := e.settle(ctx, openReservation.ID, began); err != nil {
		return true, err
	}
	stopped := false
	stop := func(ctx context.Context, reason string) error {
		request, err := c.StopRequest(sid, reason)
		if err != nil {
			return err
		}
		request.Timeout = e.cfg.Limits.StopTimeout
		observed, err := e.cfg.Sessions.Stop(ctx, request, channel)
		if err != nil || !observed.Terminated {
			return errors.Join(errStep, err)
		}
		stopped = true
		return nil
	}
	defer func() {
		if !stopped {
			_ = stop(context.WithoutCancel(ctx), "the Host ended the step early")
		}
	}()
	quoted, err := e.quotes(ctx, task.Manager)
	if err != nil {
		return true, err
	}
	stepID := StepID(e.id("step", task.ID))
	stepReservation := Reservation{ID: "step-" + string(stepID), Kind: AgentStep, Purpose: purpose, Maximum: e.cfg.Limits.StepTimeout}
	prepStep, err := c.PrepareTaskStepCommand(e.meta("prepare-step"), StepPlan{Session: sid, ID: stepID, SessionBinding: plan.Binding()}, stepReservation, e.cfg.Preparation,
		StepFacts{Tree: version.tree, Registry: version.registry, Quoted: quoted})
	if err != nil {
		return true, err
	}
	if err := e.commit(ctx, prepStep); err != nil {
		return true, err
	}
	stepAdmission, err := c.StepAdmission(ctx, e.cfg.Journal, stepID)
	if err != nil {
		return true, err
	}
	began = e.cfg.Clock()
	observed, err := e.cfg.Sessions.Step(ctx, stepAdmission, channel)
	if err != nil || observed.Uncertain || !observed.Terminal {
		return true, errors.Join(errStep, err)
	}
	if err := e.settle(ctx, stepReservation.ID, began); err != nil {
		return true, err
	}
	if err := stop(ctx, "the step ended"); err != nil {
		return true, err
	}
	harvest, err := e.cfg.Workspaces.Harvest(ctx, HarvestRequest{ID: workspace, MaxEntries: e.cfg.Limits.HarvestEntries, MaxBytes: e.cfg.Limits.HarvestBytes})
	if err != nil || !harvest.Complete() {
		return true, errors.Join(errStep, err)
	}
	return true, e.conclude(ctx, task, item.Phase, version, harvest.Tree)
}

// stepVersion builds the step's workspace content (core rules D3, T7): the
// current aggregate plus the Manager's own unaccepted changes, nothing for the
// expectations review, and the frozen candidate for the candidate review.
func (e *Engine) stepVersion(ctx context.Context, task TaskView, phase Phase) (stepVersion, error) {
	switch phase {
	case PhaseReviewerExpectations:
		tree, err := snapshot.NewTree(nil)
		return stepVersion{tree: tree}, err
	case PhaseReviewerCandidate:
		if task.Submission.Candidate == (BlobRef{}) {
			if task.Submission.Aggregate >= uint64(len(e.versions)) {
				return stepVersion{}, ErrState
			}
			a := e.versions[task.Submission.Aggregate]
			return stepVersion{a.Tree(), a.Registry()}, nil
		}
		candidate, err := e.candidate(ctx, task.Submission.Candidate)
		if err != nil {
			return stepVersion{}, err
		}
		return stepVersion{candidate.Tree(), candidate.Registry()}, nil
	}
	a := e.aggregate()
	if task.Own != (BlobRef{}) {
		own, err := e.candidate(ctx, task.Own)
		if err != nil {
			return stepVersion{}, err
		}
		replayed, ok := realization.ReplayDelivery(a, own).Aggregate()
		if !ok {
			return stepVersion{}, ErrState
		}
		a = replayed
	}
	return stepVersion{a.Tree(), a.Registry()}, nil
}

func (e *Engine) candidate(ctx context.Context, ref BlobRef) (realization.Candidate, error) {
	p, err := e.cfg.Journal.ReadBlob(ctx, ref)
	if err != nil || p.Ref() != ref {
		return realization.Candidate{}, ErrObservationLimit
	}
	record, err := realization.DecodeCandidateRecord(p.Bytes(), e.cfg.Limits.Record)
	if err != nil {
		return realization.Candidate{}, err
	}
	return record.Observation().Candidate(), nil
}

func (e *Engine) quotes(ctx context.Context, manager string) ([]Payload, error) {
	var out []Payload
	for _, ref := range e.cfg.Coordinator.StepQuotes(manager) {
		p, err := e.cfg.Journal.ReadBlob(ctx, ref)
		if err != nil || p.Ref() != ref {
			return nil, ErrObservationLimit
		}
		out = append(out, p)
	}
	return out, nil
}

// conclude records the step's S4 result and its outcome (core rules S4, T5,
// T6, T8). A read-only step that changed files, or a write outside the
// owner's claims, ends the step as failed. A candidate review that passes
// composes the delivery onto the aggregate first (core rule D2); a refused
// composition is a Host error that stops the run.
func (e *Engine) conclude(ctx context.Context, task TaskView, phase Phase, version stepVersion, observed snapshot.Tree) error {
	c := e.cfg.Coordinator
	result := StepResult{Aggregate: c.AggregateVersion()}
	if ref, ok := c.OutcomeInput(task.Manager); ok {
		p, err := e.cfg.Journal.ReadBlob(ctx, ref)
		if err != nil || p.Ref() != ref {
			return ErrObservationLimit
		}
		result.Input = p
	}
	var composed realization.Aggregate
	switch {
	case !phase.writing() && !observed.Equal(version.tree):
		result.Failure = "a read-only step changed its workspace"
	case phase.writing() && (!observed.Equal(version.tree) || len(e.registered[task.Manager]) > 0):
		// Registry proposals change the candidate even without file changes.
		record, err := e.capture(task.Manager, observed)
		if err != nil {
			result.Failure = "the step's changes are outside its owner's claims: " + err.Error()
			break
		}
		p, err := NewPayload(candidateRecordMedia, "utf-8", record.Bytes())
		if err != nil {
			return err
		}
		result.Candidate, result.Tree = p, observed.Digest()
	case phase == PhaseReviewerCandidate && result.Input.Valid() && task.Submission.Candidate != (BlobRef{}):
		verdict, err := c.ReviewVerdict(task.Manager, result.Input)
		if err != nil || verdict != VerdictAccept {
			break
		}
		candidate, err := e.candidate(ctx, task.Submission.Candidate)
		if err != nil {
			return err
		}
		replay := realization.ReplayDelivery(e.aggregate(), candidate)
		next, ok := replay.Aggregate()
		if !ok {
			reason := "the accepted delivery of " + task.Manager + " cannot be composed: " + findingText(replay.Findings())
			_ = e.commit(ctx, Command{ID: e.meta("stop").ID, ObservedAt: e.cfg.Clock().UTC(), Kind: StopRun, Assignment: c.Current().Assignment, Run: c.Current().ID, Reason: reason})
			return errors.New(reason)
		}
		composed, result.Aggregate = next, next.Version()
	}
	cmd, err := c.ConcludeStepCommand(e.meta("conclude"), task.Manager, result)
	if err != nil {
		return err
	}
	if err := e.commit(ctx, cmd); err != nil {
		return err
	}
	if composed.Valid() {
		e.versions = append(e.versions, composed)
		delete(e.reserved, task.Manager)
		delete(e.registered, task.Manager)
	}
	return nil
}

// capture checks the observed tree against the current aggregate and the
// task's claims and proposed registry records (core rule S4).
func (e *Engine) capture(manager string, observed snapshot.Tree) (realization.CandidateRecord, error) {
	owner, err := managerOwner(manager)
	if err != nil {
		return realization.CandidateRecord{}, err
	}
	base := e.aggregate().CandidateBase()
	claims, err := e.freshClaims()
	if err != nil {
		return realization.CandidateRecord{}, err
	}
	records := map[snapshot.Path]realization.Registration{}
	for _, r := range base.Registry().Entries() {
		records[r.Artifact().Path()] = r
	}
	for path, r := range e.registered[manager] {
		records[path] = r
	}
	var entries []realization.Registration
	for _, r := range records {
		entries = append(entries, r)
	}
	proposed, err := realization.NewRegistry(entries)
	if err != nil {
		return realization.CandidateRecord{}, err
	}
	result := realization.CaptureCandidate(base, owner, claims, observed, proposed)
	candidate, ok := result.Candidate()
	if !ok {
		return realization.CandidateRecord{}, errors.New(findingText(result.Findings()))
	}
	return realization.NewCandidateRecord(e.compiled, candidate, nil, nil, e.cfg.Limits.Record)
}

// freshClaims forms the run-wide claim set at the current aggregate version by
// reserving every recorded reservation again.
func (e *Engine) freshClaims() (realization.PathClaims, error) {
	facts := e.aggregate().PathFacts()
	var claims realization.PathClaims
	managers := make([]string, 0, len(e.reserved))
	for m := range e.reserved {
		managers = append(managers, m)
	}
	slices.Sort(managers)
	for _, m := range managers {
		for _, request := range e.reserved[m] {
			next, decision := realization.ReserveNewPath(facts, claims, request)
			if decision.Status() != realization.PathAdmissible {
				return realization.PathClaims{}, fmt.Errorf("reservation of %s no longer fits", request.Path())
			}
			claims = next
		}
	}
	return claims, nil
}

// tools answers the Host tools of one step. Outcome tools are accepted once
// per step, after the Coordinator's own check; reservations and registry
// proposals take effect for the task; read supplies Definitions and the
// step's files. Tools the core does not support yet are refused with a reason.
func (e *Engine) tools(manager string, phase Phase, version stepVersion) ToolHandler {
	answered := false
	return func(_ context.Context, call ToolCall, _ int64) (ToolOutcome, error) {
		refuse := func(reason string) (ToolOutcome, error) { return ToolOutcome{Status: ToolRefused, Reason: reason}, nil }
		ok := func(result any) (ToolOutcome, error) {
			data, err := json.Marshal(result)
			if err != nil {
				return refuse("the result cannot be encoded")
			}
			return ToolOutcome{Status: ToolOK, Result: data}, nil
		}
		if !PhaseTool(phase, call.Tool) {
			return refuse("this tool is not offered in the " + string(phase) + " phase")
		}
		if call.Input.Redacted {
			return refuse("redacted tool input cannot be evaluated")
		}
		args, err := CallArguments(call.Tool, call.Input.Journal.Bytes())
		if err != nil {
			return refuse(err.Error())
		}
		if OutcomeTool(phase, call.Tool) {
			if answered {
				return refuse("this step already has an outcome")
			}
			if err := e.cfg.Coordinator.CheckOutcome(manager, call.Tool, args); err != nil {
				return refuse(err.Error())
			}
			answered = true
			return ok(map[string]bool{"accepted": true})
		}
		switch call.Tool {
		case ToolRead:
			var in struct {
				Ref string `json:"ref"`
			}
			if err := decodeArgs(args, &in); err != nil {
				return refuse(err.Error())
			}
			return e.read(phase, version, in.Ref)
		case ToolReservePath:
			var in struct {
				Path string `json:"path"`
				Form string `json:"form"`
			}
			if err := decodeArgs(args, &in); err != nil {
				return refuse(err.Error())
			}
			owner, err := managerOwner(manager)
			if err != nil {
				return refuse(err.Error())
			}
			path, err := snapshot.NewPath(in.Path)
			if err != nil {
				return refuse("invalid path")
			}
			request, err := realization.NewPathRequest(realization.ClaimID(manager+":"+in.Path), owner, path, realization.Form(in.Form))
			if err != nil {
				return refuse("invalid reservation")
			}
			claims, err := e.freshClaims()
			if err != nil {
				return refuse(err.Error())
			}
			if _, decision := realization.ReserveNewPath(e.aggregate().PathFacts(), claims, request); decision.Status() != realization.PathAdmissible {
				return refuse(findingText(decision.Findings()))
			}
			if !slices.Contains(e.reserved[manager], request) {
				e.reserved[manager] = append(e.reserved[manager], request)
			}
			return ok(map[string]bool{"reserved": true})
		case ToolRegister:
			var in struct {
				Path     string   `json:"path"`
				Form     string   `json:"form"`
				Realizes []string `json:"realizes"`
				Why      string   `json:"why"`
			}
			if err := decodeArgs(args, &in); err != nil {
				return refuse(err.Error())
			}
			owner, err := managerOwner(manager)
			if err != nil {
				return refuse(err.Error())
			}
			path, err := snapshot.NewPath(in.Path)
			if err != nil {
				return refuse("invalid path")
			}
			var basis []model.Identity
			for _, d := range in.Realizes {
				id, found := definitionIdentity(d)
				if !found {
					return refuse("realizes must name Definitions as package/id")
				}
				basis = append(basis, id)
			}
			artifact, err := realization.NewArtifact(path, realization.Form(in.Form), basis, in.Why)
			if err != nil {
				return refuse("invalid registry record: " + err.Error())
			}
			record, err := realization.NewRegistration(owner, artifact)
			if err != nil {
				return refuse("invalid registry record: " + err.Error())
			}
			if e.registered[manager] == nil {
				e.registered[manager] = map[snapshot.Path]realization.Registration{}
			}
			e.registered[manager][path] = record
			return ok(map[string]bool{"registered": true})
		}
		return refuse("the " + call.Tool + " tool is not available in this version of the core")
	}
}

// read returns a Definition as package/id or a file of the step's version.
// The expectations review reads Definitions only (core rule T7).
func (e *Engine) read(phase Phase, version stepVersion, ref string) (ToolOutcome, error) {
	refuse := func(reason string) (ToolOutcome, error) { return ToolOutcome{Status: ToolRefused, Reason: reason}, nil }
	if id, ok := definitionIdentity(ref); ok {
		for _, m := range e.compiled.Models {
			for _, d := range m.Definitions {
				if d.Identity == id {
					data, err := json.Marshal(map[string]any{"definition": ref, "kind": d.Kind.Package + "/" + d.Kind.ID, "title": d.Title, "body": d.Body, "spec": d.Spec})
					if err != nil {
						return refuse("the Definition cannot be encoded")
					}
					return ToolOutcome{Status: ToolOK, Result: data}, nil
				}
			}
		}
	}
	if phase == PhaseReviewerExpectations {
		return refuse("before the candidate, read supplies Definitions only")
	}
	path, err := snapshot.NewPath(ref)
	if err != nil {
		return refuse("ref names neither a Definition as package/id nor a file path")
	}
	entry, found := version.tree.Lookup(path)
	if !found {
		return refuse("no Definition or file " + ref + " exists in this step's version")
	}
	result := map[string]any{"path": ref, "contentId": entry.ContentID().String(), "bytes": entry.ByteLength()}
	if utf8.Valid(entry.Bytes()) {
		result["text"] = string(entry.Bytes())
	}
	data, err := json.Marshal(result)
	if err != nil {
		return refuse("the file cannot be encoded")
	}
	return ToolOutcome{Status: ToolOK, Result: data}, nil
}

func findingText(findings []realization.Finding) string {
	var parts []string
	for _, f := range findings {
		parts = append(parts, f.Code+": "+f.Message)
	}
	return strings.Join(parts, "; ")
}

func definitionIdentity(s string) (model.Identity, bool) {
	i := strings.LastIndex(s, "/")
	if i <= 0 || i == len(s)-1 {
		return model.Identity{}, false
	}
	return model.Identity{Package: s[:i], ID: s[i+1:]}, true
}

func (e *Engine) id(kind, task string) string {
	return fmt.Sprintf("%s-%s-%d", kind, task[len(task)-12:], e.cfg.Coordinator.Revision())
}

func (e *Engine) meta(kind string) RecordMeta {
	return RecordMeta{ID: fmt.Sprintf("%s-%d", kind, e.cfg.Coordinator.Revision()), ObservedAt: e.cfg.Clock().UTC()}
}

func (e *Engine) settle(ctx context.Context, invocation string, began time.Time) error {
	c := Command{ID: fmt.Sprintf("settle-%s", invocation), ObservedAt: e.cfg.Clock().UTC(), Kind: SettleInvocation, Assignment: e.cfg.Coordinator.Current().Assignment, Run: e.cfg.Coordinator.Current().ID,
		Invocation: invocation, Elapsed: e.cfg.Clock().Sub(began), Reason: "Host observed the invocation's end and elapsed duration"}
	return e.commit(ctx, c)
}

func (e *Engine) commit(ctx context.Context, cmd Command) error {
	p, err := e.cfg.Coordinator.Propose(cmd)
	if err != nil {
		return err
	}
	_, err = e.cfg.Coordinator.Commit(ctx, e.cfg.Journal, p)
	return err
}
