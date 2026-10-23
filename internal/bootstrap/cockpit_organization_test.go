package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"

	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

func organizationFixture(t *testing.T) (cockpitService, setupFixture, apply.InputSelection, string, string) {
	t.Helper()
	f := newSetupFixture(t)
	data, err := os.ReadFile(filepath.Join(f.repo.Dir, ".markitect/execution.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.Replace(text, "packages: {}", "packages:\n  product/cli:\n    profiles: {execute: second-agent, verify: second-agent}", 1)
	text += "team:\n  maxHelpers: 2\n  helpers:\n    - role: architect\n      profile: helper-agent\n      writes: false\n    - role: worker\n      writes: true\nownership:\n  - path: ./\n    owner: .\n  - path: docs/\n    owner: documentation\n"
	f.repo.Write(".markitect/execution.yaml", text)
	f.repo.Commit("organization with defaults, override, helpers and nested ownership")
	if code, out, errout := f.run(t); code != 0 {
		t.Fatalf("setup %d %s %s", code, out, errout)
	}
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	c := cockpitService{repository: repository, preparation: apply.NewPreparationService(apply.NewModelService(repository, repository)),
		agents: agentMetadataRouter{claude: claudecode.NewMetadataPreparer(func(claudecode.EngineConfig) (claudecode.ImageReader, error) { return f.fake, nil })},
		checks: checks.NewMetadataPreparer(func(checks.EngineConfig) (checks.ImageReader, error) { return f.fake, nil })}
	selection, err := repositorySelection(f.repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	return c, f, selection, filepath.Join(f.runtimeDir, "run.json"), filepath.Join(f.runtimeDir, "runtime.yaml")
}

func TestCockpitDeclaredOrganizationPreservesSourcesAndUsesProductOwnership(t *testing.T) {
	c, f, selection, _, _ := organizationFixture(t)
	p, err := c.project(context.Background(), selection)
	if err != nil {
		t.Fatal(err)
	}
	v := p.Organization
	if v == nil || v.Mode != "declared" || v.Error != "" || len(v.Managers) != p.Packages+1 || v.Managers[0].Path != "." || len(v.Managers[0].Definitions) != 0 {
		t.Fatalf("organization = %+v", v)
	}
	if v.Head != f.repo.Git("rev-parse", "HEAD") || v.Strategy != "direct" || v.Declarations.Defaults.Execute.Source.File != ".markitect/execution.yaml" {
		t.Fatalf("sources = %+v", v)
	}
	root := v.Managers[0]
	if len(v.Checks) == 0 || v.Checks[0].Source == nil || v.Checks[0].Source.Digest == "" || v.Checks[0].Source.Field != "" {
		t.Fatal("check declaration must bind its whole source document")
	}
	if len(root.Areas) != 1 || !strings.Contains(root.Areas[0], "except docs/ (Manager \"documentation\")") {
		t.Fatalf("area exclusions = %v", root.Areas)
	}
	for _, m := range v.Managers {
		if m.Path == "product/cli" && (m.Execute.Name != "second-agent" || m.Verify.Name != "second-agent" || m.Execute.Source.Field == v.Declarations.Defaults.Execute.Source.Field) {
			t.Fatalf("override = %+v", m)
		}
		if m.Team.MaxHelpers != 2 || m.Team.Helpers[1].Profile.Name != "" {
			t.Fatalf("declared team invented a resolved profile: %+v", m.Team)
		}
	}
	f.repo.Write("unaccounted.txt", "unregistered draft\n")
	p, err = c.project(context.Background(), selection)
	if err != nil {
		t.Fatal(err)
	}
	if p.Organization.Status != "failed" || !strings.Contains(p.Organization.Error, "ownership areas are unavailable") {
		t.Fatalf("failed File check concealed ownership: %+v", p.Organization)
	}
}

// stopAtStart refuses evidence immediately after retaining the real run.start.
// All other execution ports are inert: no workspace, provider or check runs.
type stopAtStart struct{ event runport.Event }

var errOrganizationStop = errors.New("organization test stops at run.start")

func (s *stopAtStart) Record(e runport.Event) error {
	if e.Kind == "run.start" {
		s.event = e
	}
	return errOrganizationStop
}
func (*stopAtStart) Transcript(string) string { panic("no Agent may start") }

type organizationUnusedAgents struct{ runport.Agents }
type organizationUnusedChecks struct{ runport.Checks }
type organizationUnusedWorkspaces struct{ runport.Workspaces }

func (organizationUnusedWorkspaces) Peek(context.Context, runport.HarvestRequest) (runport.WorkspaceHarvest, error) {
	return runport.WorkspaceHarvest{}, errors.New("no test workspace exists")
}

func (organizationUnusedWorkspaces) Close(_ context.Context, id runport.WorkspaceID) (runport.WorkspaceObservation, error) {
	return runport.WorkspaceObservation{ID: id, Closed: true}, nil
}

func TestCockpitResolvedOrganizationEqualsActualRunStartBindingWithoutProviders(t *testing.T) {
	c, _, selection, configPath, profilePath := organizationFixture(t)
	ctx := context.Background()
	facts, err := c.runFacts(ctx, selection, configPath, profilePath)
	if err != nil {
		t.Fatal(err)
	}
	_, a, err := c.captureWithRuntime(ctx, selection, configPath, profilePath)
	if err != nil {
		t.Fatal(err)
	}
	evidence := &stopAtStart{}
	_, err = apply.Apply(ctx, apply.ApplyRequest{RunID: "organization-test", Build: "test", Prepared: a, Selection: selection, MaxEntries: 10000, MaxBytes: 64 << 20,
		RequireVolumeSubpaths: func(context.Context) error { return nil }, Evidence: evidence,
		Agents: organizationUnusedAgents{}, Runner: organizationUnusedChecks{}, Workspaces: organizationUnusedWorkspaces{},
		CheckRequest: func(c apply.CheckSelection) (runport.CheckRequest, error) {
			d := c.Declaration()
			return runport.CheckRequest{ImageID: bootstrapImageID, Executable: "/usr/local/go/bin/go", Args: d.Args.Value, Directory: d.Directory.Value, Timeout: d.Timeout.Value, MaxOutputBytes: 1024, Limits: runport.CheckLimits{MilliCPUs: 1000, MemoryBytes: 1 << 20, Processes: 16}}, nil
		}})
	if evidence.event.Kind != "run.start" {
		t.Fatalf("no run.start was recorded: %v", err)
	}
	data, _ := json.Marshal(evidence.event.Data)
	var recorded struct {
		Binding apply.RunBinding `json:"binding"`
	}
	if err := json.Unmarshal(data, &recorded); err != nil {
		t.Fatal(err)
	}
	v := facts.Organization
	if v.Mode != "resolved" || v.Head != recorded.Binding.Head || len(v.Managers) != len(recorded.Binding.Managers) {
		t.Fatalf("resolved = %+v, binding = %+v", v, recorded.Binding)
	}
	for i, m := range v.Managers {
		b := recorded.Binding.Managers[i]
		if m.Path != b.Manager || m.Execute.Name != b.Execute.Profile || m.Verify.Name != b.Verify.Profile || m.Execute.Identity != b.Execute.Identity || m.Verify.Identity != b.Verify.Identity || m.Team.MaxHelpers != b.MaxHelpers || len(m.Team.Helpers) != len(b.Helpers) {
			t.Fatalf("manager %+v differs from binding %+v", m, b)
		}
		if m.Execute.Model != "claude-haiku-5-5" || m.Verify.Model != "claude-haiku-5-5" {
			t.Fatalf("provider model = %+v", m)
		}
		for j, h := range m.Team.Helpers {
			bound := b.Helpers[j]
			if h.Role != bound.Role || h.Writes != bound.Writes || h.Profile.Name != bound.Profile || h.Profile.Identity != bound.Identity {
				t.Fatalf("helper %+v differs from binding %+v", h, bound)
			}
		}
	}
}
