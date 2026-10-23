package apply_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var extractNow = time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)

func extractRequest(t *testing.T) apply.ExtractRequest {
	t.Helper()
	return apply.ExtractRequest{RunID: "x-1", Prepared: extractAttempt(t), Selection: apply.InputSelection{Repository: "/repo"}, MaxEntries: 1000, MaxBytes: 1 << 20,
		RequireVolumeSubpaths: func(context.Context) error { return nil }}
}

func TestExtractPreparationAcceptsUnregisteredArtifactsApplyRefuses(t *testing.T) {
	selection, raw := extractInputs(t)
	attempt := prepareExtract(t, selection, raw)
	if attempt.Settings().Status() != model.Passed {
		t.Fatalf("settings: %s", attempt.Settings().Status())
	}
	// The same start blocks Apply: its artifacts are unexplained.
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		return captured(preparationObservation(t, request.Selection, raw)), nil
	}}
	agent, check := runtimePorts(t)
	result, err := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
	if result.Status() == model.Passed {
		t.Fatalf("Apply prepared unregistered artifacts: %v", err)
	}
	if _, err := apply.NewCandidates(attempt); err != nil {
		t.Fatalf("candidate accounting of the extract start: %v", err)
	}
}

func TestExtractRefusesAStartThatIsNotCase3(t *testing.T) {
	// A model with Definitions is Apply's input.
	selection, raw := libraryInputs(t)
	censusFile(t, &raw, ".markitect/execution.yaml", snapshot.Regular, []byte(strings.Replace(documentedExecution(t), "checks:\n", "ownership:\n  - {path: realization/, owner: .}\nchecks:\n", 1)))
	entry, _ := raw.Census.Lookup(path(t, ".markitect/execution.yaml"))
	selection.Execution = []apply.SourceLocation{{Boundary: selection.Repository, Path: entry.Path()}}
	raw.Execution = []apply.SourceFile{{Location: selection.Execution[0], File: entry, RepositoryPath: entry.Path()}}
	addRuntimeDocument(t, &selection, &raw, testRuntimeDocument)
	attempt := prepareExtract(t, selection, raw)
	workspaces := &fakeWorkspaces{}
	_, err := apply.Extract(context.Background(), apply.ExtractRequest{RunID: "x-1", Prepared: attempt, Workspaces: workspaces, Evidence: &memoryEvidence{},
		RequireVolumeSubpaths: func(context.Context) error { return nil }})
	if !errors.Is(err, apply.ErrNotExtractable) || !strings.Contains(err.Error(), "22 Definitions") || len(workspaces.open) != 0 {
		t.Fatalf("a model with Definitions started an Extract: %v", err)
	}
	// A project that declares no ownership areas, or an empty list, has no owners
	// for the artifacts.
	for name, replacement := range map[string]string{
		"no ownership":    "",
		"empty ownership": "ownership: []\n",
	} {
		selection, raw = extractInputs(t)
		var changed []snapshot.Entry
		for _, e := range raw.Census.Entries() {
			if e.Path().String() == ".markitect/execution.yaml" {
				document := strings.Replace(extractExecution, "ownership:\n  - path: README.md\n    owner: .\n  - path: realization/\n    owner: .\n", replacement, 1)
				if document == extractExecution {
					t.Fatal("the fixture's ownership was not replaced")
				}
				e, _ = snapshot.NewEntry(e.Path(), e.Mode(), []byte(document))
			}
			changed = append(changed, e)
		}
		raw.Census, _ = snapshot.NewTree(changed)
		entry, _ = raw.Census.Lookup(path(t, ".markitect/execution.yaml"))
		raw.Execution[0].File = entry
		attempt = prepareExtract(t, selection, raw)
		if _, err := apply.Extract(context.Background(), apply.ExtractRequest{RunID: "x-1", Prepared: attempt, RequireVolumeSubpaths: func(context.Context) error { return nil }}); !errors.Is(err, apply.ErrNotExtractable) || !strings.Contains(err.Error(), "no ownership areas") {
			t.Fatalf("%s: a project without ownership started an Extract: %v", name, err)
		}
	}
}

func TestExtractRefusesBeforeAnyEffectWithoutVolumeSubpaths(t *testing.T) {
	req := extractRequest(t)
	workspaces := &fakeWorkspaces{}
	req.Workspaces, req.Evidence, req.Agents = workspaces, &memoryEvidence{}, failingAgents{}
	req.RequireVolumeSubpaths = func(context.Context) error { return errors.New("Docker server 25.0.0; need 26") }
	if _, err := apply.Extract(context.Background(), req); err == nil || !strings.Contains(err.Error(), "Docker server 25.0.0") || len(workspaces.open) != 0 {
		t.Fatalf("err = %v", err)
	}
	req.RequireVolumeSubpaths = nil
	if _, err := apply.Extract(context.Background(), req); err == nil || !strings.Contains(err.Error(), "check for them is required") {
		t.Fatalf("err = %v", err)
	}
}

// extractAttemptWithTurns prepares the library case 3 with another agent-turn
// budget in its execution document.
func extractAttemptWithTurns(t *testing.T, turns int) apply.CapturedSettingsAttempt {
	t.Helper()
	selection, raw := extractInputs(t)
	execution := strings.Replace(extractExecution, "agentTurns: 120", "agentTurns: "+strconv.Itoa(turns), 1)
	entry, err := snapshot.NewEntry(path(t, ".markitect/execution.yaml"), snapshot.Regular, []byte(execution))
	if err != nil {
		t.Fatal(err)
	}
	var census []snapshot.Entry
	for _, e := range raw.Census.Entries() {
		if e.Path() != entry.Path() {
			census = append(census, e)
		}
	}
	if raw.Census, err = snapshot.NewTree(append(census, entry)); err != nil {
		t.Fatal(err)
	}
	raw.Execution[0].File = entry
	return prepareExtract(t, selection, raw)
}

func TestExtractRefusesTooFewAgentTurnsForBothPhases(t *testing.T) {
	req := extractRequest(t)
	req.Prepared = extractAttemptWithTurns(t, 3)
	workspaces := &fakeWorkspaces{}
	req.Workspaces, req.Evidence, req.Agents = workspaces, &memoryEvidence{}, failingAgents{}
	if _, err := apply.Extract(context.Background(), req); err == nil || !strings.Contains(err.Error(), "at least 12 agent turns") || len(workspaces.open) != 0 {
		t.Fatalf("err = %v", err)
	}
}

func TestExtractStructurePhaseWritePlanAndBrief(t *testing.T) {
	cfg, err := apply.ExtractStructureConfig(extractRequest(t), extractNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Team) != 1 || cfg.Team[0].Path != "." || cfg.Team[0].Parent != "" || len(cfg.Team[0].Children) != 0 || cfg.RunID != "x-1-structure" {
		t.Fatalf("the structure phase is Root alone: %+v", cfg.Team)
	}
	if len(cfg.Checks) != 0 || !cfg.Start.Equal(extractStart(t)) || cfg.ModelCheck == nil || cfg.FileCheck == nil || cfg.Registry == nil || cfg.Definition == nil || cfg.ResolveDefinition != nil {
		t.Fatalf("config: checks %d, gates %v %v %v %v, resolver %v", len(cfg.Checks), cfg.ModelCheck != nil, cfg.FileCheck != nil, cfg.Registry != nil, cfg.Definition != nil, cfg.ResolveDefinition != nil)
	}
	mandate, verifyMandate, guidance, advisory := apply.ExtractPhaseTexts(apply.ExtractStructurePhase)
	if !strings.Contains(mandate, "structure") || !strings.Contains(mandate, "Folder names are clues, not authority") || !strings.Contains(mandate, "propose those commands as entries of checks in .markitect/execution.yaml; they apply only once the owner adopts the model.") ||
		!strings.Contains(mandate, "as areas in ownership of .markitect/execution.yaml") || strings.Contains(mandate, "layout") {
		t.Errorf("mandate: %s", mandate)
	}
	if verifyMandate == "" || !strings.Contains(guidance, "an ownership area that maps artifacts to a Package whose responsibility they do not evidence") || advisory {
		t.Errorf("the structure review: %q %q advisory %v", verifyMandate, guidance, advisory)
	}
	// Every plan is enforced; the artifacts stay read-only because the base is.
	want := run.WritePlan{Base: false, Mounts: []run.PathMount{
		{Path: ".markitect", Writable: true},
		{Path: ".markitect/dependencies/core"}, {Path: ".markitect/dependencies/software"},
	}}
	if got := cfg.Team[0].Write; !slices.Equal(got.Mounts, want.Mounts) || got.Base != want.Base {
		t.Fatalf("write plan:\n got %+v\nwant %+v", got, want)
	}
	if len(cfg.Team[0].Helpers) != 0 || cfg.Team[0].MaxHelpers != 0 {
		t.Errorf("the structure phase delegates nothing: %+v", cfg.Team[0].Helpers)
	}
	brief := cfg.Team[0].Brief
	for _, w := range []string{"structure phase of an Extract", "The project holds 9 artifact file(s).", "realization/lending/reminders: 2 file(s)", "the artifacts and the imported dependency models under .markitect/dependencies/core, .markitect/dependencies/software.", "ownership", "The areas declared now", "- README.md -> Root", "- realization/ -> Root", "Software.Concept",
		"software/Concept (use as Software.Concept)", "software/Rule", ".markitect/model/<package-path>/<id>.yaml", "[[Software.other-id]]", "declared checks of its artifacts", "project-tests", "Extract does not run them",
		"ownership and checks of .markitect/execution.yaml", "The check profiles the Runtime offers", "- library-tests: runs program", "timeout: 2m"} {
		if !strings.Contains(brief, w) {
			t.Errorf("brief lacks %q:\n%s", w, brief)
		}
	}
	if strings.Contains(brief, "small library") || strings.Contains(brief, "borrow") && strings.Contains(brief, "Definition borrow") {
		t.Errorf("the brief leaks model content:\n%s", brief)
	}
	// Account is a pure function of the tree: no Definition, no Package yet.
	got := cfg.FileCheck(cfg.Start)
	codes := summarize(got)
	if !slices.Contains(codes, ". model.no_package") {
		t.Errorf("the start has no proposal: %v", codes)
	}
	if cfg.OwnerOf(path(t, "realization/lending/borrow.go")) != "." {
		t.Error("Root owns everything in the structure phase")
	}
}

func extractStart(t *testing.T) snapshot.Tree {
	t.Helper()
	return extractAttempt(t).Structural().Model().Inputs().Raw.Census
}

func summarize(findings []run.Finding) []string {
	var lines []string
	for _, f := range findings {
		lines = append(lines, f.Owner+" "+f.Obligation)
	}
	return lines
}

func TestExtractStructureAccounting(t *testing.T) {
	req := extractRequest(t)
	cfg, err := apply.ExtractStructureConfig(req, extractNow)
	if err != nil {
		t.Fatal(err)
	}
	start := cfg.Start
	if got := cfg.FileCheck(proposedTree(t, start)); len(got) != 0 {
		t.Fatalf("a complete proposal is not accounted: %+v", got)
	}
	for _, tc := range []struct {
		name    string
		changes map[string]*string
		want    string // "owner obligation path"
	}{
		{"model failing Model check", map[string]*string{".markitect/model/library/library.yaml": text("id: library\nkind: Software.Concept\nbody:  \nspec: {}\n")}, ". model."},
		{"unresolved kind", map[string]*string{".markitect/model/library/library.yaml": text(overview("library", "Library.")[:0] + "id: library\nkind: Nope.Concept\nbody: x\nspec: {}\n")}, ". model."},
		{"package without Definition", map[string]*string{".markitect/model/lending/lending.yaml": nil, ".markitect/model/lending/reminders/reminders.yaml": text(overview("reminders", "Overdue notices."))}, ". model.empty_package"},
		{"artifact without owner", map[string]*string{".markitect/model/members/members.yaml": nil, ".markitect/model/catalog/catalog.yaml": nil, ".markitect/model/library/library.yaml": nil, ".markitect/model/lending/lending.yaml": nil, ".markitect/model/lending/reminders/reminders.yaml": text(overview("reminders", "Overdue notices.")), ".markitect/execution.yaml": text(strings.Replace(extractExecution, "  - path: realization/\n    owner: .\n", "", 1))}, ". layout.no_owner"},
		{"changed artifact", map[string]*string{"realization/lending/borrow.go": text("package lending // edited\n")}, ". input.changed"},
		{"deleted artifact", map[string]*string{"realization/go.mod": nil}, ". input.deleted"},
		{"new artifact", map[string]*string{"realization/members/new.go": text("package members\n")}, ". input.added"},
		{"changed manifest", map[string]*string{".markitect/model.yaml": text("formatVersion: 1\nid: other\nversion: 1.0.0\nroot: model\ndependencies: {}\n")}, ". input.changed"},
		{"changed dependency", map[string]*string{".markitect/dependencies/software/model/software/rule.yaml": text("id: Rule\n")}, ". input.changed"},
		{"execution beyond the ownership", map[string]*string{".markitect/execution.yaml": text(strings.Replace(extractExecutionProposed, "agentTurns: 120", "agentTurns: 999", 1))}, ". input.changed"},
		{"execution without ownership", map[string]*string{".markitect/execution.yaml": text(strings.Replace(extractExecutionProposed, extractOwnershipProposed, "", 1))}, ". layout.no_owner"},
		{"registry written in the structure phase", map[string]*string{rootShard: text(shardYAML())}, ". input.added"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tree := change(t, proposedTree(t, start), tc.changes)
			findings := cfg.FileCheck(tree)
			if len(findings) == 0 {
				t.Fatal("no finding")
			}
			found := false
			for _, f := range findings {
				if f.Owner != "." || !f.Blocking || f.RaisedBy != run.MarkitectFinding || f.Observed == "" {
					t.Errorf("finding %+v", f)
				}
				found = found || strings.HasPrefix(f.Owner+" "+f.Obligation, tc.want)
			}
			if !found {
				t.Errorf("want %q in %v", tc.want, summarize(findings))
			}
		})
	}
	// A changed artifact quotes its original so that Root can restore it.
	changed := cfg.FileCheck(change(t, proposedTree(t, start), map[string]*string{"realization/lending/borrow.go": text("package lending // edited\n")}))
	if len(changed) != 1 || !strings.Contains(changed[0].Observed, "Original content") || !strings.Contains(changed[0].Observed, "package lending\n") || !strings.Contains(changed[0].Observed, "Extract never changes artifacts") {
		t.Errorf("%+v", changed)
	}
}

func TestExtractStructureAccountsProposedChecks(t *testing.T) {
	cfg, err := apply.ExtractStructureConfig(extractRequest(t), extractNow)
	if err != nil {
		t.Fatal(err)
	}
	account := func(execution string) []run.Finding {
		return cfg.FileCheck(change(t, proposedTree(t, cfg.Start), map[string]*string{".markitect/execution.yaml": text(execution)}))
	}
	if got := account(extractExecutionProposed + extractProposedCheck("library-tests")); len(got) != 0 {
		t.Fatalf("a check with an offered profile is not accounted: %+v", got)
	}
	for _, tc := range []struct {
		name, execution, want string
	}{
		{"unknown profile", extractExecutionProposed + extractProposedCheck("no-such-profile"), `names the profile "no-such-profile", which the Runtime does not offer. The Runtime offers the check profiles: library-tests.`},
		{"check that does not decode", extractExecutionProposed + "  - id: broken\n    profile: library-tests\n", "The Runtime offers the check profiles: library-tests."},
		{"change elsewhere beside a check", strings.Replace(extractExecutionProposed, "agentTurns: 120", "agentTurns: 999", 1) + extractProposedCheck("library-tests"), "changed beyond ownership and checks"},
		{"changed owner check", strings.Replace(extractExecutionProposed, "args: [test, ./...]", "args: [test, -run, Nothing, ./...]", 1), "changed beyond ownership and checks"},
		{"removed owner check", strings.Replace(extractExecutionProposed, "checks:\n  - id: project-tests\n    profile: library-tests\n    directory: realization\n    args: [test, ./...]\n    timeout: 2m\n", "", 1) + "checks:\n" + extractProposedCheck("library-tests"), "changed beyond ownership and checks"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := account(tc.execution)
			if len(findings) == 0 {
				t.Fatal("no finding")
			}
			for _, f := range findings {
				if f.Owner != "." || !f.Blocking || !strings.Contains(f.Observed, tc.want) && !strings.Contains(f.Obligation, tc.want) {
					t.Errorf("finding %+v lacks %q", f, tc.want)
				}
			}
		})
	}
}

func TestExtractModelPhaseTeamIsTheProposedTopology(t *testing.T) {
	req := extractRequest(t)
	start := extractStart(t)
	cfg, err := apply.ExtractModelConfig(req, proposedTree(t, start), extractNow)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RunID != "x-1-model" || len(cfg.Checks) != 0 || cfg.ModelCheck == nil || cfg.FileCheck == nil || cfg.Registry == nil || cfg.Definition == nil {
		t.Fatalf("config %+v", cfg)
	}
	mandate, verifyMandate, guidance, advisory := apply.ExtractPhaseTexts(apply.ExtractModelPhase)
	if mandate == "" || verifyMandate == "" || !strings.Contains(guidance, "report it with subject artifact") || !advisory {
		t.Errorf("the model phase texts: %q %q %q advisory %v", mandate, verifyMandate, guidance, advisory)
	}
	var paths []string
	team := map[string]run.Manager{}
	for _, m := range cfg.Team {
		if m.Parent != "" && !slices.Contains(paths, m.Parent) {
			t.Fatalf("%s listed before its parent %s", m.Path, m.Parent)
		}
		if m.ExecuteProfile != "library-agent" || m.VerifyProfile != "library-agent" || m.Write.Base || m.Brief == "" || m.Role == "" {
			t.Fatalf("incomplete Manager %+v", m)
		}
		paths = append(paths, m.Path)
		team[m.Path] = m
	}
	if !slices.Equal(paths, []string{".", "catalog", "lending", "library", "members", "lending/reminders"}) && !slices.Equal(paths, []string{".", "catalog", "library", "lending", "lending/reminders", "members"}) {
		// Root first, every parent before its children; the sibling order is the topology's.
		for _, want := range []string{".", "catalog", "lending", "lending/reminders", "library", "members"} {
			if !slices.Contains(paths, want) {
				t.Fatalf("team %v lacks %s", paths, want)
			}
		}
	}
	if len(cfg.Team) != 6 || paths[0] != "." || !slices.Equal(team["."].Children, []string{"catalog", "lending", "library", "members"}) || !slices.Equal(team["lending"].Children, []string{"lending/reminders"}) {
		t.Fatalf("team %v root children %v", paths, team["."].Children)
	}
	// Write plans: no artifact is writable, each Manager writes only its Package's
	// model directory and its shard.
	for name, want := range map[string][]run.PathMount{
		".": {{Path: ".markitect/registry", Writable: true}, {Path: ".markitect/registry/catalog"}, {Path: ".markitect/registry/lending"}, {Path: ".markitect/registry/library"}, {Path: ".markitect/registry/members"}},
		"lending": {{Path: ".markitect/model/lending", Writable: true}, {Path: ".markitect/model/lending/reminders"},
			{Path: ".markitect/registry/lending", Writable: true}, {Path: ".markitect/registry/lending/reminders"}},
		"lending/reminders": {{Path: ".markitect/model/lending/reminders", Writable: true}, {Path: ".markitect/registry/lending/reminders", Writable: true}},
		"library":           {{Path: ".markitect/model/library", Writable: true}, {Path: ".markitect/registry/library", Writable: true}},
	} {
		got := team[name].Write.Mounts
		if !slices.Equal(got, want) {
			t.Errorf("%s write plan:\n got %+v\nwant %+v", name, got, want)
		}
	}
	// Briefs.
	root := team["."].Brief
	for _, w := range []string{"Root in the model phase", ".markitect/registry/registry.yaml", "realization/go.mod", "README.md", "You have no Package", "- lending\n  artifact areas: realization/lending/ except realization/lending/reminders/ (Manager \"lending/reminders\")", "Definition lending/lending: lending"} {
		if !strings.Contains(root, w) {
			t.Errorf("Root brief lacks %q:\n%s", w, root)
		}
	}
	reminders := team["lending/reminders"].Brief
	for _, w := range []string{"Package lending/reminders in the model phase", "Your Package directory is .markitect/model/lending/reminders;", ".markitect/registry/lending/reminders/registry.yaml", "realization/lending/reminders/remind.go", "realization/lending/reminders/remind_test.go",
		"lending/reminders/reminders (software/Concept)", "realizes: [lending/reminders/reminders]", "[[Software.other-id]]", "Observed behavior is evidence, not intent", "project-tests"} {
		if !strings.Contains(reminders, w) {
			t.Errorf("reminders brief lacks %q:\n%s", w, reminders)
		}
	}
	if strings.Contains(reminders, "realization/lending/borrow.go") {
		t.Errorf("a child lists its parent's artifacts:\n%s", reminders)
	}
	lending := team["lending"].Brief
	for _, w := range []string{"- lending/reminders\n  artifact areas: realization/lending/reminders/", "You integrate their results in your own Package", "realization/lending/borrow.go"} {
		if !strings.Contains(lending, w) {
			t.Errorf("lending brief lacks %q:\n%s", w, lending)
		}
	}
	library := team["library"].Brief
	if !strings.Contains(library, "realization/cmd/library/main.go") || !strings.Contains(library, "realization/library/library.go") {
		t.Errorf("the ownership area of the program entry is not in the library's areas:\n%s", library)
	}
	if cfg.OwnerOf(path(t, ".markitect/model/lending/reminders/reminders.yaml")) != "lending/reminders" ||
		cfg.OwnerOf(path(t, ".markitect/model/lending/lending.yaml")) != "lending" ||
		cfg.OwnerOf(path(t, "realization/cmd/library/main.go")) != "library" ||
		cfg.OwnerOf(path(t, "realization/go.mod")) != "." ||
		cfg.OwnerOf(path(t, ".markitect/registry/members/registry.yaml")) != "members" {
		t.Error("ownership of model, artifact and shard paths")
	}
}

// extractRemindersArea is the ownership area of the lending/reminders Package in
// extractExecutionProposed.
const extractRemindersArea = `  - path: realization/lending/reminders/
    owner: lending/reminders
`

func TestExtractModelAccounting(t *testing.T) {
	req := extractRequest(t)
	start := extractStart(t)
	proposed := proposedTree(t, start)
	cfg, err := apply.ExtractModelConfig(req, proposed, extractNow)
	if err != nil {
		t.Fatal(err)
	}
	complete := registeredModel(t, start)
	if got := cfg.FileCheck(complete); len(got) != 0 {
		t.Fatalf("a complete registered model is not accounted: %+v", got)
	}
	// Unregistered artifacts are the owners' findings.
	got := summarize(cfg.FileCheck(proposed))
	for _, want := range []string{". file.unexplained", "catalog file.unexplained", "library file.unexplained", "lending file.unexplained", "lending/reminders file.unexplained", "members file.unexplained"} {
		if !slices.Contains(got, want) {
			t.Errorf("want %q in %v", want, got)
		}
	}
	for _, tc := range []struct {
		name    string
		changes map[string]*string
		want    string
	}{
		{"registration in the wrong shard", map[string]*string{lendingShard: text(shardYAML(shardEntry("realization/lending/borrow.go", "code", "lending/lending"), shardEntry("realization/members/member.go", "code", "members/members")))}, "lending registry.outside_area"},
		{"registration of an absent Definition", map[string]*string{rootShard: text(shardYAML(shardEntry("README.md", "doc", "library/nothing"), shardEntry("realization/go.mod", "build", "library/library")))}, ". registry.definition_missing"},
		{"changed artifact", map[string]*string{"realization/lending/borrow.go": text("package lending // edited\n")}, "lending input.changed"},
		{"deleted artifact", map[string]*string{"realization/members/member.go": nil}, "members input.deleted"},
		{"changed execution document", map[string]*string{".markitect/execution.yaml": text(extractExecution)}, ". input.changed"},
		{"invalid execution document", map[string]*string{".markitect/execution.yaml": text("apiVersion: nothing\n")}, ". input.changed"},
		{"changed dependency", map[string]*string{".markitect/dependencies/core/model.yaml": text("id: core\n")}, ". input.changed"},
		{"Definition failing Model check", map[string]*string{".markitect/model/members/members.yaml": text("id: members\nkind: Software.Concept\n")}, "members model."},
		{"unresolved link", map[string]*string{".markitect/model/members/members.yaml": text(overview("members", "Members, see [[Lending.nothing]]."))}, "members model."},
		{"new Package", map[string]*string{".markitect/model/lending/loans/loans.yaml": text(overview("loans", "Loans."))}, "lending model.structure_changed"},
		// The execution document owns the artifacts of every Package: a Package
		// that disappears while its ownership area stays leaves a document that
		// names a Package the model no longer has.
		{"Package removed while its area remains", map[string]*string{".markitect/model/lending/reminders/reminders.yaml": nil}, ". execution.package"},
		{"Package removed with its area", map[string]*string{".markitect/model/lending/reminders/reminders.yaml": nil, ".markitect/execution.yaml": text(strings.Replace(extractExecutionProposed, extractRemindersArea, "", 1))}, ". model.structure_changed"},
		{"new file in an unmapped directory", map[string]*string{"realization/other/x.go": text("package other\n")}, ". input.added"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := summarize(cfg.FileCheck(change(t, registeredModel(t, start), tc.changes)))
			found := false
			for _, line := range findings {
				found = found || strings.HasPrefix(line, tc.want)
			}
			if !found {
				t.Errorf("want %q in %v", tc.want, findings)
			}
		})
	}
	// A Manager may extend its own Package and shard: that is its work.
	extended := change(t, complete, map[string]*string{".markitect/model/members/status.yaml": text(overview("status", "A member is active or suspended."))})
	if got := cfg.FileCheck(extended); len(got) != 0 {
		t.Fatalf("a new Definition in an existing Package is not accounted: %+v", got)
	}
}

func TestExtractAccountIsSafeForConcurrentCalls(t *testing.T) {
	req := extractRequest(t)
	start := extractStart(t)
	cfg, err := apply.ExtractModelConfig(req, proposedTree(t, start), extractNow)
	if err != nil {
		t.Fatal(err)
	}
	trees := []snapshot.Tree{proposedTree(t, start), registeredModel(t, start)}
	want := []int{len(cfg.FileCheck(trees[0])), len(cfg.FileCheck(trees[1]))}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j, tree := range trees {
				if got := len(cfg.FileCheck(tree)); got != want[j] {
					t.Errorf("tree %d: %d findings, want %d", j, got, want[j])
				}
			}
		}()
	}
	wg.Wait()
}

func TestExtractPublishesOnlyTheModel(t *testing.T) {
	start := extractStart(t)
	complete := registeredModel(t, start)
	if err := apply.RequireModelChangesOnly(snapshot.Compare(start, complete)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"realization/lending/borrow.go", ".markitect/model.yaml", ".markitect/dependencies/core/model.yaml"} {
		changes := snapshot.Compare(start, change(t, complete, map[string]*string{name: text("changed\n")}))
		if err := apply.RequireModelChangesOnly(changes); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestExtractDeviationsGroupOpenArtifactFindingsByDefinition(t *testing.T) {
	findings := []run.Finding{
		{ID: "f1", Owner: "lending", RaisedBy: "lending", Subject: "artifact", Definition: "lending/lending", Observed: "borrow.go admits a sixth loan."},
		{ID: "f2", Owner: "members", RaisedBy: "members", Subject: "artifact", Definition: "members/members", Observed: "closed one", Closed: true},
		{ID: "f3", Owner: "catalog", RaisedBy: "catalog", Subject: "model", Obligation: "Missing statement", Blocking: true},
		{ID: "f4", Owner: "lending/reminders", RaisedBy: "lending", Subject: "artifact", Definition: "lending/lending", Observed: "remind.go counts 14 days."},
		{ID: "f5", Owner: "catalog", RaisedBy: "library", Subject: "artifact", Definition: "catalog/catalog", Observed: "search.go ignores case."},
		{ID: "f6", Owner: "lending", RaisedBy: "lending", Obligation: "Better wording"},
		{ID: "f7", Owner: "lending", RaisedBy: "lending", Subject: "artifact", Definition: "lending/lending", Observed: "README.md says 14 days."},
	}
	got := apply.ExtractDeviations(findings)
	want := []apply.ArtifactDeviation{
		{Definition: "catalog/catalog", Owners: []string{"catalog"}, RaisedBy: []string{"library"}, Observed: []string{"search.go ignores case."}},
		{Definition: "lending/lending", Owners: []string{"lending", "lending/reminders"}, RaisedBy: []string{"lending"}, Observed: []string{"borrow.go admits a sixth loan.", "remind.go counts 14 days.", "README.md says 14 days."}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
	if got := apply.ExtractDeviations(nil); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestExtractModelPhaseResolvesDefinitionsOfTheCandidate(t *testing.T) {
	req := extractRequest(t)
	start := extractStart(t)
	proposed := proposedTree(t, start)
	model, err := apply.ExtractModelConfig(req, proposed, extractNow)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, advisory := apply.ExtractPhaseTexts(apply.ExtractModelPhase); !advisory || model.ResolveDefinition == nil {
		t.Fatalf("the model phase does not report artifact findings: %v %+v", advisory, model)
	}
	for id, want := range map[string]bool{"lending/lending": true, "lending/reminders/reminders": true, "lending/reminders": false, "lending/other": false, "other/lending": false, "": false} {
		if got := model.ResolveDefinition(proposed, id); got != want {
			t.Errorf("%q resolves %v", id, got)
		}
	}
	// The identities are those of the tree that is asked about.
	without := change(t, proposed, map[string]*string{".markitect/model/lending/lending.yaml": nil})
	if model.ResolveDefinition(without, "lending/lending") || !model.ResolveDefinition(proposed, "lending/lending") {
		t.Error("the resolver kept the identities of another tree")
	}
	structure, err := apply.ExtractStructureConfig(req, extractNow)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, advisory := apply.ExtractPhaseTexts(apply.ExtractStructurePhase); advisory || structure.ResolveDefinition != nil {
		t.Error("the structure phase reports artifact findings")
	}
}

func TestExtractModelAccountingReportsEachFindingOnce(t *testing.T) {
	req := extractRequest(t)
	start := extractStart(t)
	cfg, err := apply.ExtractModelConfig(req, proposedTree(t, start), extractNow)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		changes map[string]*string
		want    []string
	}{
		{"changed artifact", map[string]*string{"realization/lending/borrow.go": text("package lending // edited\n")}, []string{"lending input.changed"}},
		{"deleted artifact", map[string]*string{"realization/members/member.go": nil}, []string{". files.registry_not_passed", "members registry.artifact_missing", "members input.deleted"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := cfg.FileCheck(change(t, registeredModel(t, start), tc.changes))
			for _, f := range findings {
				if f.Owner == "" || f.Obligation == "" || f.Observed == "" {
					t.Errorf("incomplete finding %+v", f)
				}
			}
			if got := summarize(findings); !slices.Equal(got, tc.want) {
				t.Errorf("findings:\n got %v\nwant %v", got, tc.want)
			}
		})
	}
}

func TestExtractGuardLeavesTheRuntimeStateAndGitDataAlone(t *testing.T) {
	req := extractRequest(t)
	start := extractStart(t)
	structure, err := apply.ExtractStructureConfig(req, extractNow)
	if err != nil {
		t.Fatal(err)
	}
	model, err := apply.ExtractModelConfig(req, proposedTree(t, start), extractNow)
	if err != nil {
		t.Fatal(err)
	}
	state := map[string]*string{".markitect/state/runs/x.json": text("{}\n")}
	git := map[string]*string{".git/HEAD": text("ref: refs/heads/main\n")}
	if got := structure.FileCheck(change(t, proposedTree(t, start), state)); len(got) != 0 {
		t.Errorf("state in the structure phase: %+v", got)
	}
	if got := structure.FileCheck(change(t, proposedTree(t, start), git)); len(got) != 0 {
		t.Errorf("git data in the structure phase: %+v", got)
	}
	if got := model.FileCheck(change(t, registeredModel(t, start), state)); len(got) != 0 {
		t.Errorf("state in the model phase: %+v", got)
	}
	for _, f := range model.FileCheck(change(t, registeredModel(t, start), git)) {
		if strings.HasPrefix(f.Obligation, "input.") {
			t.Errorf("git data is guarded in the model phase: %+v", f)
		}
	}
}
