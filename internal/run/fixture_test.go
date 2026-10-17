package run_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var observed = time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)

func emptyLock() model.Entry {
	return model.Entry{Path: "model-lock.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nmodels: {}\n")}
}

func compiled(t *testing.T, input model.Input) model.Result {
	t.Helper()
	c := model.Compile(input)
	if c.Status != model.Passed {
		t.Fatalf("fixture compile: %+v", c.Diagnostics)
	}
	return c
}

func file(t *testing.T, name string, mode snapshot.Mode, data string) snapshot.Entry {
	t.Helper()
	p, err := snapshot.NewPath(name)
	if err != nil {
		t.Fatal(err)
	}
	e, err := snapshot.NewEntry(p, mode, []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func tree(t *testing.T, entries []snapshot.Entry) snapshot.Tree {
	t.Helper()
	v, err := snapshot.NewTree(entries)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func source(d run.Document, field string) organization.SettingSource {
	s := d.Source
	s.Field = field
	return s
}

func selected[T any](d run.Document, field string, value T) organization.Setting[T] {
	return organization.Setting[T]{Value: value, Source: source(d, field)}
}

func fixture(t *testing.T) run.BindingInput {
	return fixtureWith(t, nil, nil)
}

func fixtureWith(t *testing.T, changeModel func(*model.Input), changeSettings func(*organization.SettingsFacts)) run.BindingInput {
	t.Helper()
	core := coreSource()
	coreDigest := compiled(t, model.Input{Root: core}).Models[0].Digest
	extra := model.SourceModel{Origin: "z-extra", ManifestPath: "model.yaml", Entries: []model.Entry{
		{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: extra\nversion: '1'\nroot: model\ndependencies: {markitect-core: '1.0'}\n")},
		{Path: "model", Kind: model.Directory},
		{Path: "model-lock.yaml", Kind: model.Regular, Bytes: []byte(fmt.Sprintf("formatVersion: 1\nmodels:\n  markitect-core: {version: '1.0', digest: '%s'}\n", coreDigest))},
	}}
	extraDigest := compiled(t, model.Input{Root: extra, Dependencies: []model.SourceModel{core}}).Models[0].Digest
	local := model.SourceModel{Origin: "local", ManifestPath: "model.yaml", Entries: []model.Entry{
		{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: library\nroot: model\ndependencies: {markitect-core: '1.0', extra: '1'}\n")},
		{Path: "model/lending/item.yaml", Kind: model.Regular, Bytes: []byte("imports: [{package: markitect/core, as: Core}]\nid: Item\nkind: Core.Kind\nbody: A library responsibility.\nspec: {}\n")},
		{Path: "model/lending/reminders/item.yaml", Kind: model.Regular, Bytes: []byte("imports: [{package: markitect/core, as: Core}]\nid: Item\nkind: Core.Kind\nbody: Reminder responsibility.\nspec: {}\n")},
		{Path: "model/empty", Kind: model.Directory},
		{Path: "model-lock.yaml", Kind: model.Regular, Bytes: []byte(fmt.Sprintf("formatVersion: 1\nmodels:\n  markitect-core: {version: '1.0', digest: '%s'}\n  extra: {version: '1', digest: '%s'}\n", coreDigest, extraDigest))},
	}}
	// Deliberately reverse producer result ordering: Compile sorts dependencies.
	input := model.Input{Root: local, Dependencies: []model.SourceModel{extra, core}}
	if changeModel != nil {
		changeModel(&input)
	}
	c := compiled(t, input)
	topology, err := organization.BuildTopology(c)
	if err != nil {
		t.Fatal(err)
	}
	var docs []run.Document
	for _, item := range []struct{ name, data string }{
		{"execution.yaml", "explicit execution fixture bytes\r\n"},
		{"profiles.yaml", "explicit static profiles\n"},
		{"style.txt", "\xef\xbb\xbfUse exact sources.\r\n"},
		{"review.txt", "Review independently.\n"},
		{"registry.yaml", "explicit registry bytes\n"},
		{"ignore.yaml", "explicit ignore bytes\n"},
		{"strategy.md", "explicit strategy bytes\n"},
	} {
		e := file(t, item.name, snapshot.Regular, item.data)
		docs = append(docs, run.Document{File: e, Source: organization.SettingSource{File: item.name, ContentID: e.ContentID()}, RepositoryPath: e.Path()})
	}
	exec := docs[0]
	profiles := []organization.ProfileFact{
		{Name: "writer", Identity: "static-writer-v1", Source: source(docs[1], "/writer")},
		{Name: "critic", Identity: "static-critic-v1", Source: source(docs[1], "/critic")},
	}
	facts := organization.SettingsFacts{
		Defaults: organization.SettingsDefaults{
			Manager:     organization.RoleDefaults{Profile: selected(exec, "/defaults/manager", "writer"), Instructions: selected(exec, "/defaults/managerInstructions", []string{"style", "review", "style"})},
			Reviewer:    organization.RoleDefaults{Profile: selected(exec, "/defaults/reviewer", "critic"), Instructions: selected(exec, "/defaults/reviewerInstructions", []string{"review"})},
			StepTimeout: selected(exec, "/defaults/stepTimeout", time.Minute), RejectionLimit: selected(exec, "/defaults/rejectionLimit", 2),
			CreationRoots: selected(exec, "/defaults/creationRoots", []snapshot.DirectoryBoundary{}), BatchNoChangeReviews: selected(exec, "/batchNoChangeReviews", false),
		},
		Ceilings: organization.SettingsCeilings{
			StepTimeout: selected(exec, "/limits/stepTimeout", 2*time.Minute), RejectionLimit: selected(exec, "/limits/maxRejectionLimit", 3),
			CreationRoots: selected(exec, "/limits/creationRoots", []snapshot.DirectoryBoundary{}),
		}, Profiles: profiles,
		Instructions: []organization.InstructionFact{
			{ID: "style", Source: docs[2].Source, Bytes: docs[2].File.Bytes()},
			{ID: "review", Source: docs[3].Source, Bytes: docs[3].File.Bytes()},
		},
	}
	if changeSettings != nil {
		changeSettings(&facts)
	}
	settings, err := organization.ResolveSettings(topology, facts)
	if err != nil {
		t.Fatal(err)
	}
	var entries []snapshot.Entry
	var classifications []realization.Classification
	var protections []realization.Protection
	for _, d := range docs {
		entries = append(entries, d.File)
		cl, err := realization.NewClassification(d.File.Path(), realization.MarkitectInput, "Explicit run input")
		if err != nil {
			t.Fatal(err)
		}
		classifications = append(classifications, cl)
		p, err := realization.ProtectFile(d.File.Path(), "Fixed input")
		if err != nil {
			t.Fatal(err)
		}
		protections = append(protections, p)
	}
	owner, _ := realization.PackageOwner("lending")
	artifact := file(t, "out/library.go", snapshot.Regular, "package library\n")
	entries = append(entries, artifact)
	a, err := realization.NewArtifact(artifact.Path(), realization.Code, []model.Identity{{Package: "lending", ID: "Item"}}, "Existing realization")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := realization.NewRegistration(owner, a)
	registry, _ := realization.NewRegistry([]realization.Registration{r})
	request, _ := run.NewPayload("text/plain", "utf-8", []byte("Apply the captured library model."))
	return run.BindingInput{
		MaxContextBytes: selected(exec, "/limits/maxContextBytes", int64(1<<20)),
		Source:          run.SourceBasis{Repository: "repo", Revision: "fixed-source", Guard: "source-guard", ProductBuild: "test-build", Operation: "artifact-apply/v1"},
		OwnerRequest:    request, ModelInput: input, Compilation: c, Census: tree(t, entries), Registry: registry, Classifications: classifications, Protections: protections,
		Settings: settings, Profiles: profiles, Documents: docs, PolicySource: source(exec, "/budget"),
		Policy: run.BudgetPolicy{
			Ceilings:       run.Capacity{Runs: 8, SessionStarts: 128, Steps: 256, ExecutionTime: 20 * time.Minute},
			Assignment:     run.Capacity{Runs: 3, SessionStarts: 32, Steps: 40, ExecutionTime: 10 * time.Minute},
			Reserve:        run.Capacity{SessionStarts: 12, Steps: 12, ExecutionTime: 2 * time.Minute},
			MaxStepTimeout: 2 * time.Minute, HostOperationTimeout: time.Minute,
		},
	}
}

func binding(t *testing.T) run.Binding {
	t.Helper()
	b, err := run.Bind(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// exploreInputs keeps the real settings inputs so live RUN preparation resolves
// them again rather than trusting arbitrary host context JSON.
func exploreInputs(t *testing.T, onePackage bool) run.ExplorePreparation {
	var facts organization.SettingsFacts
	change := func(input *model.Input) {
		if !onePackage {
			return
		}
		var entries []model.Entry
		for _, e := range input.Root.Entries {
			if e.Path != "model/empty" && e.Path != "model/lending/reminders/item.yaml" {
				entries = append(entries, e)
			}
		}
		input.Root.Entries = entries
	}
	input := fixtureWith(t, change, func(f *organization.SettingsFacts) { facts = *f })
	subjects := int64(len(input.Settings.Managers()))
	topology, err := organization.BuildTopology(input.Compilation)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range topology.Managers() {
		if len(m.Children) > 0 {
			subjects++
		}
	}
	input.Policy.Reserve.SessionStarts = 2 * subjects
	input.Policy.Reserve.Steps = 2 * subjects
	return run.ExplorePreparation{Binding: input, Settings: facts}
}
func exploreSessionPreparation(t *testing.T, c *run.Coordinator, inputs run.ExplorePreparation, manager string, id run.SessionID) run.Command {
	t.Helper()
	profile, tools := payload(t, `{"profile":"synthetic recorded metadata"}`), payload(t, `{"tools":[]}`)
	plan := run.SessionPlan{ID: id, Role: "manager", Workspace: "linux-workspace-" + string(id), Candidate: inputs.Binding.Census.Digest(), ReadOnly: true, ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", CredentialRef: "credential-ref", MaxWireBytes: 1 << 20, Profile: profile.Ref(), Tools: tools.Ref()}
	cmd, err := c.PrepareExploreSessionCommand(meta("prepare-"+string(id)), manager, plan, run.Reservation{ID: "open-" + string(id), Kind: run.SessionStart, Purpose: run.Writing, Maximum: time.Minute}, profile, tools)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

func completeRootExplore(t *testing.T, c *run.Coordinator, j *memoryJournal, inputs run.ExplorePreparation) {
	t.Helper()
	io, p, plan := explorePreparedStep(t, c, j, inputs, ".", "root")
	exploreObserveReport(t, c, j, io, p, plan, nil)
	exploreStop(t, c, j, io, "root")
}
