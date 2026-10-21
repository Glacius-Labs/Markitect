package apply_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func checkRequest(apply.CheckSelection) (run.CheckRequest, error) {
	directory, err := snapshot.DirectoryAt(path0("realization"))
	if err != nil {
		return run.CheckRequest{}, err
	}
	return run.CheckRequest{
		ImageID: "sha256:" + strings.Repeat("a", 64), Executable: "/usr/local/go/bin/go", Args: []string{"test", "./..."},
		Directory: directory, Limits: run.CheckLimits{MilliCPUs: 1000, MemoryBytes: 1 << 30, Processes: 64},
		Timeout: time.Minute, MaxOutputBytes: 1 << 20,
	}, nil
}

func path0(name string) snapshot.Path {
	p, err := snapshot.NewPath(name)
	if err != nil {
		panic(err)
	}
	return p
}

func layoutRequest(t *testing.T) (apply.ApplyRequest, snapshot.Tree) {
	t.Helper()
	selection, raw := executionInputs(t, withOwnership(t, libraryOwnershipYAML))
	libraryLayoutFiles(t, &selection, &raw)
	attempt := preparedAttempt(t, selection, raw)
	return apply.ApplyRequest{
		RunID: "run-1", Prepared: attempt, Selection: apply.InputSelection{Repository: "/repo"}, Build: "test-build", OwnerRequest: "Realize the model.",
		MaxEntries: 1000, MaxBytes: 1 << 20, CheckRequest: checkRequest,
		RequireVolumeSubpaths: func(context.Context) error { return nil },
	}, attempt.Structural().Model().Inputs().Raw.Census
}

func TestApplyConfigTeamBriefsChecksBudgetAndAccounting(t *testing.T) {
	req, start := layoutRequest(t)
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	cfg, err := apply.BuildApplyConfig(req, now)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RunID != "run-1" || cfg.OwnerRequest != "Realize the model." || cfg.MaxEntries != 1000 || cfg.MaxBytes != 1<<20 || !cfg.Start.Equal(start) {
		t.Fatalf("run inputs changed: %+v", cfg)
	}
	// Apply's Root ends the run for a defective model; the bound model is fixed.
	if !strings.Contains(cfg.RootGuidance, "abort true") || !strings.Contains(cfg.RootGuidance, "never ask the user to change or override it") {
		t.Fatalf("Apply's Root guidance: %q", cfg.RootGuidance)
	}
	if cfg.Team[0].Path != "." || cfg.Team[0].Parent != "" || !strings.Contains(cfg.Team[0].Role, "integration of the whole project") {
		t.Fatalf("Root is not first: %+v", cfg.Team[0])
	}
	seen := map[string]run.Manager{}
	var paths []string
	for _, m := range cfg.Team {
		if m.Parent != "" {
			if _, ok := seen[m.Parent]; !ok {
				t.Fatalf("%s listed before its parent %s", m.Path, m.Parent)
			}
		}
		if m.ExecuteProfile != "library-agent" || m.VerifyProfile != "library-agent" || m.Brief == "" || m.Role == "" {
			t.Fatalf("incomplete Manager: %+v", m)
		}
		seen[m.Path] = m
		paths = append(paths, m.Path)
	}
	for _, want := range []string{"lending", "lending/reminders", "library"} {
		if !slices.Contains(paths, want) {
			t.Fatalf("team %v lacks %s", paths, want)
		}
	}
	if !slices.Contains(seen["lending"].Children, "lending/reminders") || seen["lending/reminders"].Parent != "lending" {
		t.Fatal("hierarchy lost")
	}
	brief := seen["lending/reminders"].Brief
	for _, want := range []string{"Package lending/reminders", ".markitect/registry/lending/reminders/registry.yaml", "apiVersion: markitect.registry/v1", "one of code, test, doc, build, config", "or a directory ending in / for the files below it that you own", "/work/.markitect", "lending/reminders/assess-overdue", "Required checks", "project-tests: /usr/local/go/bin/go test ./..."} {
		if !strings.Contains(brief, want) {
			t.Errorf("brief lacks %q:\n%s", want, brief)
		}
	}
	if !strings.Contains(seen["."].Brief, "You are the Manager of Root") || !strings.Contains(seen["."].Brief, ".markitect/registry/registry.yaml") {
		t.Errorf("Root brief: %s", seen["."].Brief)
	}
	// Root has no Definitions of its own: it still sees what the required checks are for.
	rootBrief := seen["."].Brief
	for _, want := range []string{"realizes: [library/callable-interface]", "library/callable-interface (", "included because: required by check project-tests"} {
		if !strings.Contains(rootBrief, want) {
			t.Errorf("Root brief lacks %q:\n%s", want, rootBrief)
		}
	}
	if strings.Contains(rootBrief, "package/id") || strings.Contains(rootBrief, "none are included") {
		t.Errorf("Root brief keeps the placeholder:\n%s", rootBrief)
	}
	for _, manager := range []string{"lending", "lending/reminders", "library"} {
		if strings.Contains(seen[manager].Brief, "required by check") {
			t.Errorf("%s has Definitions of its own and needs no check-required ones:\n%s", manager, seen[manager].Brief)
		}
		if strings.Contains(seen[manager].Brief, "Project obligations") {
			t.Errorf("only Root receives the project obligations:\n%s", seen[manager].Brief)
		}
	}
	if len(cfg.Checks) != 1 || cfg.Checks[0].Name != "project-tests" || cfg.Checks[0].Owner != "." || cfg.Checks[0].RequiredBy != "library/callable-interface" {
		t.Fatalf("checks: %+v", cfg.Checks)
	}
	if cfg.Budget.TurnTimeout != 10*time.Minute || cfg.Budget.AgentTurns != 120 || cfg.Budget.Passes != 12 || !cfg.Budget.Deadline.Equal(now.Add(25*time.Minute)) {
		t.Fatalf("budget: %+v", cfg.Budget)
	}
	if cfg.OwnerOf == nil || cfg.OwnerOf(path(t, "realization/lending/borrow.go")) != "lending" || cfg.OwnerOf(path(t, "realization/go.mod")) != "." || cfg.OwnerOf(path(t, "docs/guide.md")) != "" {
		t.Fatal("ownership is not the declared one")
	}
	if cfg.ModelCheck == nil || cfg.FileCheck == nil || cfg.Registry == nil || cfg.Definition == nil {
		t.Fatalf("a gate or lookup is missing: model %v file %v registry %v definition %v", cfg.ModelCheck != nil, cfg.FileCheck != nil, cfg.Registry != nil, cfg.Definition != nil)
	}
	if got := cfg.FileCheck(start); len(got) != 0 {
		t.Fatalf("the captured start is not accounted: %+v", got)
	}
	if got := cfg.ModelCheck(start); len(got) != 0 {
		t.Fatalf("the captured start does not pass the Model check: %+v", got)
	}
	extra := change(t, start, map[string]*string{"realization/lending/new.go": text("package lending\n")})
	findings := cfg.FileCheck(extra)
	if len(findings) != 1 || findings[0].Owner != "lending" || findings[0].RaisedBy != run.MarkitectFinding || !findings[0].Blocking || findings[0].Obligation == "" || !strings.Contains(findings[0].Observed, "realization/lending/new.go") {
		t.Fatalf("unregistered file: %+v", findings)
	}
	view := cfg.Registry(start)
	if len(view.Registrations) != 6 || view.Shards[rootShard] == "" {
		t.Fatalf("registry view: %+v", view)
	}
	if definition, ok := cfg.Definition("lending/reminders/assess-overdue"); !ok || definition.Owner != "lending/reminders" || definition.Text == "" {
		t.Fatalf("definition lookup: %+v %v", definition, ok)
	}
	if _, ok := cfg.Definition("lending/missing"); ok {
		t.Fatal("look_up found a Definition the model lacks")
	}
}

func TestApplyDeadlineIsTheEarlierOfBudgetAndOperatorLimit(t *testing.T) {
	req, _ := layoutRequest(t)
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name     string
		deadline time.Time
		want     time.Time
	}{
		{"none", time.Time{}, now.Add(25 * time.Minute)},
		{"earlier", now.Add(10 * time.Minute), now.Add(10 * time.Minute)},
		{"later", now.Add(2 * time.Hour), now.Add(25 * time.Minute)},
	} {
		req.Deadline = tc.deadline
		cfg, err := apply.BuildApplyConfig(req, now)
		if err != nil || !cfg.Budget.Deadline.Equal(tc.want) {
			t.Errorf("%s: %v %v", tc.name, cfg.Budget.Deadline, err)
		}
	}
}

func TestApplyConfigRefusesAnUnusableCheckRequest(t *testing.T) {
	req, _ := layoutRequest(t)
	req.CheckRequest = func(apply.CheckSelection) (run.CheckRequest, error) {
		return run.CheckRequest{}, errors.New("not a container check")
	}
	if _, err := apply.BuildApplyConfig(req, time.Now()); err == nil || !strings.Contains(err.Error(), "project-tests") {
		t.Fatalf("%v", err)
	}
}

type fakePublisher struct {
	preflight, publish error
	receipt            apply.PublicationReceipt
	requests           []apply.PublicationRequest
}

func (p *fakePublisher) Preflight(_ context.Context, r apply.PublicationRequest, _ apply.ControlCheck) (apply.PublicationPermit, error) {
	p.requests = append(p.requests, r)
	return apply.PublicationPermit{Request: r, Token: "t"}, p.preflight
}

func (p *fakePublisher) Publish(context.Context, apply.PublicationPermit, apply.ControlCheck) (apply.PublicationReceipt, error) {
	return p.receipt, p.publish
}

func (p *fakePublisher) Inspect(context.Context, apply.PublicationRequest) (apply.PublicationReceipt, error) {
	return apply.PublicationReceipt{}, nil
}

func TestPublishBindsTheCaptureGuardAndMapsDefiniteRefusals(t *testing.T) {
	req, start := layoutRequest(t)
	guard := req.Prepared.Structural().Model().Inputs().Guard
	changes := snapshot.Compare(start, change(t, start, map[string]*string{"realization/lending/borrow.go": text("package lending // changed\n")}))
	for _, tc := range []struct {
		name      string
		publisher *fakePublisher
		outcome   apply.PublicationOutcome
		wantErr   bool
		noReceipt bool
	}{
		{"applied", &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationApplied}}, apply.PublicationApplied, false, false},
		{"stale", &fakePublisher{preflight: apply.ErrCaptureChanged}, apply.PublicationStale, false, false},
		{"unpublishable", &fakePublisher{preflight: apply.ErrUnpublishable}, apply.PublicationFailed, false, false},
		{"preflight error", &fakePublisher{preflight: errors.New("disk")}, "", true, true},
		{"partial with error", &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationPartial}, publish: errors.New("write")}, apply.PublicationPartial, true, false},
		{"error without receipt", &fakePublisher{publish: errors.New("write")}, "", true, true},
	} {
		req.Publisher = tc.publisher
		receipt, err := apply.PublishChanges(context.Background(), req, changes)
		if (err != nil) != tc.wantErr || (receipt == nil) != tc.noReceipt || receipt != nil && receipt.Outcome != tc.outcome {
			t.Errorf("%s: %+v %v", tc.name, receipt, err)
		}
		if r := tc.publisher.requests[0]; r.Guard != guard || r.Repository != "/repo" || r.Run != "run-1" || r.ID != "run-1" || len(r.Changes) != 1 {
			t.Errorf("%s: publication request %+v", tc.name, r)
		}
	}
}

// fakeWorkspaces materializes in memory and records closes.
type fakeWorkspaces struct {
	mu     sync.Mutex
	open   map[run.WorkspaceID]bool
	closed []run.WorkspaceID
}

func (w *fakeWorkspaces) Materialize(_ context.Context, spec run.WorkspaceSpec) (run.WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.open == nil {
		w.open = map[run.WorkspaceID]bool{}
	}
	w.open[spec.ID] = true
	return run.WorkspaceObservation{ID: spec.ID, Volume: "vol-" + string(spec.ID), Tree: spec.Tree.Digest()}, nil
}

func (w *fakeWorkspaces) Harvest(context.Context, run.HarvestRequest) (run.WorkspaceHarvest, error) {
	return run.WorkspaceHarvest{}, errors.New("not used")
}

func (w *fakeWorkspaces) Peek(_ context.Context, r run.HarvestRequest) (run.WorkspaceHarvest, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open[r.ID] {
		return run.WorkspaceHarvest{}, errors.New("no such volume")
	}
	return run.WorkspaceHarvest{ID: r.ID}, nil
}

func (w *fakeWorkspaces) Close(_ context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.open, id)
	w.closed = append(w.closed, id)
	return run.WorkspaceObservation{ID: id, Closed: true}, nil
}

type memoryEvidence struct {
	mu     sync.Mutex
	events []run.Event
}

func (e *memoryEvidence) Record(ev run.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
	return nil
}

func (e *memoryEvidence) Transcript(string) string { return "" }

type failingRunner struct{}

func (failingRunner) Run(context.Context, run.CheckRequest) (run.CheckReceipt, error) {
	return run.CheckReceipt{}, errors.New("no container engine")
}

// failingAgents makes every turn fail, so the run ends without a verified candidate.
type failingAgents struct{}

func (failingAgents) Run(context.Context, run.Turn, run.ToolHandler) (run.TurnResult, error) {
	return run.TurnResult{Stopped: true}, errors.New("provider unavailable")
}

func TestApplyPublishesNothingForAnUnverifiedRunAndRemovesItsHomes(t *testing.T) {
	req, _ := layoutRequest(t)
	workspaces := &fakeWorkspaces{}
	publisher := &fakePublisher{}
	req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = failingAgents{}, workspaces, failingRunner{}, &memoryEvidence{}, publisher
	var removed []string
	req.RemoveHome = func(_ context.Context, home string) error {
		removed = append(removed, home)
		return nil
	}
	result, err := apply.Apply(context.Background(), req)
	if result.Outcome.Status == "" || result.Outcome.Status == run.Verified || result.Receipt != nil || len(publisher.requests) != 0 {
		t.Fatalf("an unverified run published or has no outcome: %+v %v", result, err)
	}
	// The shared work volume is either kept and reported, or the run closed it itself.
	if result.RetainedWork == "" && !slices.Contains(workspaces.closed, "run-1-work") {
		t.Fatalf("work volume neither retained nor closed by the run: %v", workspaces.closed)
	}
	if result.RetainedWork != "" && result.RetainedWork != "run-1-work" {
		t.Fatalf("retained work names %q", result.RetainedWork)
	}
	// Homes are runtime volumes the Agents adapters mount by name, not
	// Workspaces; the Orchestrator's RemoveHome deletes them.
	if len(result.Outcome.Homes) == 0 || !slices.Equal(removed, result.Outcome.Homes) {
		t.Fatalf("homes %v, removed %v", result.Outcome.Homes, removed)
	}
	for _, home := range result.Outcome.Homes {
		if slices.Contains(workspaces.closed, run.WorkspaceID(home)) {
			t.Errorf("home %s was closed as a workspace", home)
		}
	}
}

// recordingAgents fails every turn after recording its kind.
type recordingAgents struct {
	mu    sync.Mutex
	kinds []run.TurnKind
}

func (a *recordingAgents) Run(_ context.Context, turn run.Turn, _ run.ToolHandler) (run.TurnResult, error) {
	a.mu.Lock()
	a.kinds = append(a.kinds, turn.Kind)
	a.mu.Unlock()
	return run.TurnResult{Stopped: true}, errors.New("provider unavailable")
}

// The evaluation entry runs only the independent Verify on the captured stand:
// every turn is a fresh read-only reviewer, nothing is published, and the
// shared work volume and the homes are removed.
func TestVerifyCapturedRunsOnlyAReviewAndNeverPublishes(t *testing.T) {
	req, _ := layoutRequest(t)
	workspaces, agents, publisher := &fakeWorkspaces{}, &recordingAgents{}, &fakePublisher{}
	req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = agents, workspaces, failingRunner{}, &memoryEvidence{}, publisher
	var removed []string
	req.RemoveHome = func(_ context.Context, home string) error {
		removed = append(removed, home)
		return nil
	}
	result, err := apply.VerifyCaptured(context.Background(), req)
	if result.Outcome.Status == "" || result.Outcome.Status == run.Verified || result.Receipt != nil || len(publisher.requests) != 0 {
		t.Fatalf("outcome %+v, err %v", result, err)
	}
	if len(agents.kinds) == 0 || slices.ContainsFunc(agents.kinds, func(k run.TurnKind) bool { return k != run.VerifyTurn }) {
		t.Fatalf("turn kinds %v", agents.kinds)
	}
	if len(result.Outcome.Passes) != 1 || result.Outcome.Passes[0].Purpose != run.Verify {
		t.Fatalf("passes %+v", result.Outcome.Passes)
	}
	if !slices.Contains(workspaces.closed, "run-1-work") || !slices.Equal(removed, result.Outcome.Homes) {
		t.Fatalf("closed %v, removed %v of %v", workspaces.closed, removed, result.Outcome.Homes)
	}
	// It binds its entry and no work Pass, recorded before any turn.
	evidence := req.Evidence.(*memoryEvidence)
	b := result.Binding
	if data, _ := evidence.events[0].Data.(map[string]any); b.Entry != "verify-captured" || len(b.Work) != 0 || b.Strategy != apply.Direct || !reflect.DeepEqual(data["binding"], b) {
		t.Fatalf("binding %+v, run.start %+v", b, evidence.events[0])
	}
}

func TestApplyRefusesEnforcedOwnershipOnAnEngineWithoutVolumeSubpaths(t *testing.T) {
	workspaces := &fakeWorkspaces{}
	for name, tc := range map[string]struct {
		check func(context.Context) error
		want  string
	}{
		"old engine":      {func(context.Context) error { return errors.New("Docker server 25.0.0; need 26") }, "Docker server 25.0.0"},
		"no engine check": {nil, "check for them is required"},
	} {
		t.Run(name, func(t *testing.T) {
			req, _ := layoutRequest(t)
			req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = failingAgents{}, workspaces, failingRunner{}, &memoryEvidence{}, &fakePublisher{}
			req.RequireVolumeSubpaths = tc.check
			_, err := apply.Apply(context.Background(), req)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			if len(workspaces.open) != 0 || len(workspaces.closed) != 0 {
				t.Fatalf("the run started before the engine check: %v %v", workspaces.open, workspaces.closed)
			}
		})
	}
}

// TestApplyClosesTheWorkVolumeOnlyAfterPublication covers a verified outcome:
// the shared work volume is the verified work, so it goes only once published.
func TestApplyClosesTheWorkVolumeOnlyAfterPublication(t *testing.T) {
	req, start := layoutRequest(t)
	candidate := change(t, start, map[string]*string{"realization/lending/borrow.go": text("package lending // changed\n")})
	for _, tc := range []struct {
		name      string
		publisher *fakePublisher
		tree      snapshot.Tree
		closed    bool
		wantErr   bool
	}{
		{"applied", &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationApplied}}, candidate, true, false},
		{"no change", &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationNoChange}}, start, true, false},
		{"stale", &fakePublisher{preflight: apply.ErrCaptureChanged}, candidate, false, false},
		{"partial", &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationPartial}, publish: errors.New("write")}, candidate, false, true},
		{"unknown", &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationUnknown}}, candidate, false, false},
		{"preflight error", &fakePublisher{preflight: errors.New("disk")}, candidate, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspaces := &fakeWorkspaces{}
			if _, err := workspaces.Materialize(context.Background(), run.WorkspaceSpec{ID: "run-1-work"}); err != nil {
				t.Fatal(err)
			}
			req := req
			req.Workspaces, req.Evidence, req.Publisher = workspaces, &memoryEvidence{}, tc.publisher
			result, err := apply.Conclude(context.Background(), req, start, run.Outcome{Status: run.Verified, Candidate: tc.tree}, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error %v", err)
			}
			closed := slices.Contains(workspaces.closed, "run-1-work")
			if closed != tc.closed {
				t.Fatalf("work volume closed = %v, want %v", closed, tc.closed)
			}
			if wantRetained := map[bool]string{true: "", false: "run-1-work"}[tc.closed]; result.RetainedWork != wantRetained {
				t.Fatalf("retained work %q, want %q", result.RetainedWork, wantRetained)
			}
		})
	}
}

func TestApplyConfigBriefsStateResponsibilitiesObligationsHelpersAndWritePlans(t *testing.T) {
	req, _ := layoutRequest(t)
	cfg, err := apply.BuildApplyConfig(req, time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	team := map[string]run.Manager{}
	for _, m := range cfg.Team {
		team[m.Path] = m
		// Every plan is enforced: no Manager has a writable base without a
		// whole-repository area.
		if m.Write.Base {
			t.Errorf("%s: write plan %+v has a writable base", m.Path, m.Write)
		}
		if len(m.Write.Mounts) == 0 {
			t.Errorf("%s: an empty write plan enforces nothing", m.Path)
		}
		if !strings.Contains(m.Brief, "Only these areas and your local registry are writable in your work turns; everything else in /work is read-only.") {
			t.Errorf("%s brief lacks the write statement:\n%s", m.Path, m.Brief)
		}
		// The product default team: one writing worker on the Manager's own Execute profile.
		if m.MaxHelpers != 1 || len(m.Helpers) != 1 || m.Helpers[0].Role != "worker" || !m.Helpers[0].Writes || m.Helpers[0].Profile != m.ExecuteProfile {
			t.Errorf("%s: helper roles %+v max %d", m.Path, m.Helpers, m.MaxHelpers)
		}
	}
	if !slices.Contains(team["lending/reminders"].Write.Mounts, run.PathMount{Path: "realization/lending/reminders", Writable: true}) ||
		!slices.Contains(team["lending"].Write.Mounts, run.PathMount{Path: "realization/lending/reminders"}) ||
		!slices.Contains(team["lending"].Write.Mounts, run.PathMount{Path: "realization/lending", Writable: true}) {
		t.Fatalf("write plans do not follow the ownership: lending %+v reminders %+v", team["lending"].Write, team["lending/reminders"].Write)
	}
	// Obligations are the identities of each Manager's own Definitions; Root has
	// none of its own.
	if len(team["."].Obligations) != 0 || !slices.Contains(team["lending/reminders"].Obligations, "lending/reminders/assess-overdue") || !slices.Contains(team["library"].Obligations, "library/callable-interface") {
		t.Errorf("obligations: root %v reminders %v library %v", team["."].Obligations, team["lending/reminders"].Obligations, team["library"].Obligations)
	}
	const children = "Your children realize their own Definitions in their areas. You supply the foundations they need within your own area, coordinate them and integrate their results; you do not implement their areas, and you cannot release them from their obligations."
	const own = "Your own Definitions are yours to realize in your area. Your parent coordinates; it cannot take them over or waive them."
	for manager, wantChildren := range map[string][]string{
		".": {"- lending\n  areas: realization/lending/ except realization/lending/reminders/ (Manager \"lending/reminders\")\n  Definition lending/", "- library\n  areas: realization/cmd/library/; realization/library/\n  Definition library/",
			// The Managers below a child are named, so no area looks unowned.
			"  below it: lending/reminders, a child of lending, areas: realization/lending/reminders/\n    Definition lending/reminders/assess-overdue: ",
			"Every area listed here has its owner in the team"},
		"lending":           {"- lending/reminders\n  areas: realization/lending/reminders/\n  Definition lending/reminders/assess-overdue: "},
		"library":           nil,
		"lending/reminders": nil,
	} {
		brief := team[manager].Brief
		if has := strings.Contains(brief, children); has != (wantChildren != nil) {
			t.Errorf("%s: children statement present = %v", manager, has)
		}
		for _, want := range wantChildren {
			if !strings.Contains(brief, want) {
				t.Errorf("%s brief lacks %q:\n%s", manager, want, brief)
			}
		}
		if has := strings.Contains(brief, own); has != (manager != ".") {
			t.Errorf("%s: own-Definitions statement present = %v", manager, has)
		}
		// DEC-060: every Manager, Root included, uses other areas' realizations.
		if !strings.Contains(brief, "Never realize another area's Definitions or touch anything outside your area") {
			t.Errorf("%s brief lacks the owner-realizes rule:\n%s", manager, brief)
		}
		if has := strings.Contains(brief, "Cross-area matters between them are yours to settle"); has != (wantChildren != nil) {
			t.Errorf("%s: cross-area statement present = %v", manager, has)
		}
	}
	root := team["."].Brief
	if !strings.Contains(root, `- realization/ except realization/cmd/library/ (Manager "library"), realization/lending/ (Manager "lending"), realization/library/ (Manager "library")`) ||
		!strings.Contains(root, "(owned by library)") || !strings.Contains(root, "required by check project-tests; realized by library, integrated and verified by you") {
		t.Errorf("Root brief:\n%s", root)
	}
	// Root coordinates the whole model: it receives every local Definition with its owner.
	for _, want := range []string{"Project obligations", "You coordinate the whole model.", "- lending/reminders/assess-overdue (", ", owned by lending/reminders", ", owned by library"} {
		if !strings.Contains(root, want) {
			t.Errorf("Root brief lacks %q:\n%s", want, root)
		}
	}
	if strings.Contains(team["lending"].Brief, "integrated and verified by you") {
		t.Errorf("only Root integrates the checks' Definitions:\n%s", team["lending"].Brief)
	}
	if !strings.Contains(team["lending/reminders"].Brief, "(owned by lending)") {
		t.Errorf("an included Definition does not show its owner:\n%s", team["lending/reminders"].Brief)
	}
}

func TestApplyConfigRefusesWithoutDeclaredOwnership(t *testing.T) {
	// A fully accounted project without an ownership section prepares, but Apply
	// refuses to run it: no write plan can enforce who writes what.
	selection, raw := executionInputs(t, documentedExecution(t))
	libraryLayoutFiles(t, &selection, &raw)
	attempt := preparedAttempt(t, selection, raw)
	req := apply.ApplyRequest{RunID: "run-1", Prepared: attempt, Selection: apply.InputSelection{Repository: "/repo"}, Build: "test-build", CheckRequest: checkRequest, RequireVolumeSubpaths: func(context.Context) error { return nil }}
	if _, err := apply.BuildApplyConfig(req, time.Now()); !errors.Is(err, apply.ErrUnsupportedOwnership) || !strings.Contains(err.Error(), "declares no ownership") {
		t.Fatalf("an Apply without declared ownership was configured: %v", err)
	}
	workspaces := &fakeWorkspaces{}
	req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = failingAgents{}, workspaces, failingRunner{}, &memoryEvidence{}, &fakePublisher{}
	result, err := apply.Apply(context.Background(), req)
	if !errors.Is(err, apply.ErrUnsupportedOwnership) || result.Outcome.Status != "" || len(workspaces.open) != 0 || len(workspaces.closed) != 0 {
		t.Fatalf("the run started without ownership: %+v %v %v %v", result.Outcome, err, workspaces.open, workspaces.closed)
	}
}

func TestApplyConfigRefusesAnExactFileAreaThatDoesNotExistInTheStartStand(t *testing.T) {
	selection, raw := executionInputs(t, withOwnership(t, foreignFileOwnership))
	libraryLayoutFiles(t, &selection, &raw)
	attempt := preparedAttempt(t, selection, raw)
	req := apply.ApplyRequest{RunID: "run-1", Prepared: attempt, Selection: apply.InputSelection{Repository: "/repo"}, Build: "test-build", CheckRequest: checkRequest, RequireVolumeSubpaths: func(context.Context) error { return nil }}
	if _, err := apply.BuildApplyConfig(req, time.Now()); !errors.Is(err, apply.ErrUnsupportedOwnership) || !strings.Contains(err.Error(), "realization/lending/Makefile") {
		t.Fatalf("an unmountable exact-file area was configured: %v", err)
	}
	workspaces := &fakeWorkspaces{}
	req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = failingAgents{}, workspaces, failingRunner{}, &memoryEvidence{}, &fakePublisher{}
	if _, err := apply.Apply(context.Background(), req); !errors.Is(err, apply.ErrUnsupportedOwnership) || len(workspaces.open) != 0 {
		t.Fatalf("the run started with an unenforceable shape: %v %v", err, workspaces.open)
	}
}
