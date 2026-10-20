package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/authoring"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func authoringFixture(t *testing.T) (*testkit.Repo, string, *authoring.Service) {
	t.Helper()
	repo := productFixture(t)
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	state := testkit.TempDir(t)
	preparation := apply.NewPreparationService(apply.NewModelService(repository, repository))
	return repo, state, cockpitService{preparation: preparation, repository: repository}.authoring(repo.Dir, []string{state})
}

func TestCockpitPublishesAnAcceptedSettingsProposalUnderTheStateLock(t *testing.T) {
	ctx := context.Background()
	repo, state, service := authoringFixture(t)
	settings, err := service.Settings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Budget.AgentTurns == 0 || len(settings.Packages) == 0 {
		t.Fatalf("settings = %+v", settings)
	}
	settings.Budget.AgentTurns = 77
	settings.Team = &authoring.Team{MaxHelpers: 1, Helpers: []authoring.Helper{{Role: "worker", Writes: true}}}
	proposal, err := service.ProposeSettings(ctx, settings)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Check.Status != "passed" || len(proposal.Changes) != 1 || proposal.Changes[0].Path != authoring.ExecutionPath {
		t.Fatalf("proposal = %+v", proposal)
	}

	lock, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Accept(ctx, proposal.ID); err == nil || !strings.Contains(err.Error(), "holds the state directory") {
		t.Fatalf("accept while an Apply holds the state = %v", err)
	}
	if p, _ := service.Proposal(proposal.ID); p.State != authoring.Proposed {
		t.Fatalf("a fenced acceptance leaves the proposal %s", p.State)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}

	accepted, err := service.Accept(ctx, proposal.ID)
	if err != nil || accepted.State != authoring.Accepted || accepted.Receipt == nil || accepted.Receipt.Outcome != "applied" || accepted.Receipt.Paths[0].State != "after" {
		t.Fatalf("accept = %+v, %v", accepted, err)
	}
	data, err := os.ReadFile(filepath.Join(repo.Dir, ".markitect", "execution.yaml"))
	if err != nil || !strings.Contains(string(data), "agentTurns: 77") || !strings.Contains(string(data), "maxHelpers: 1") {
		t.Fatalf("the working tree holds:\n%s %v", data, err)
	}
	if _, err := service.Settings(ctx); !errors.Is(err, authoring.ErrNotClean) {
		t.Fatalf("the next proposal on an uncommitted change = %v, want ErrNotClean", err)
	}
	if _, err := service.ProposeSettings(ctx, settings); !errors.Is(err, authoring.ErrNotClean) {
		t.Fatalf("a proposal from the head read before the acceptance = %v, want ErrNotClean", err)
	}
}

func TestCockpitRefusesToAcceptAProposalThatFailsTheCheck(t *testing.T) {
	ctx := context.Background()
	repo, _, service := authoringFixture(t)
	before, err := os.ReadFile(filepath.Join(repo.Dir, ".markitect", "execution.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	settings, err := service.Settings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	settings.Budget.ExecutionTime = "forever"
	proposal, err := service.ProposeSettings(ctx, settings)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Check.Status == "passed" || len(proposal.Check.Findings) == 0 {
		t.Fatalf("an invalid budget passes the check: %+v", proposal.Check)
	}
	located, seen := false, map[apply.ProposalFinding]bool{}
	for _, f := range proposal.Check.Findings {
		f.Stage = ""
		if seen[f] {
			t.Fatalf("a finding is reported twice: %+v", proposal.Check.Findings)
		}
		seen[f] = true
		located = located || f.Path == authoring.ExecutionPath
	}
	if !located {
		t.Fatalf("no finding names %s: %+v", authoring.ExecutionPath, proposal.Check.Findings)
	}
	if _, err := service.Accept(ctx, proposal.ID); err == nil {
		t.Fatal("a failing proposal was accepted")
	}
	after, err := os.ReadFile(filepath.Join(repo.Dir, ".markitect", "execution.yaml"))
	if err != nil || string(after) != string(before) {
		t.Fatalf("a refused proposal changed the working tree: %v", err)
	}
	if p, err := service.Discard(proposal.ID); err != nil || p.State != authoring.Discarded {
		t.Fatalf("discard = %+v, %v", p, err)
	}
}

func TestCockpitRefusesAProposalWhoseSourceChangedSinceItWasMade(t *testing.T) {
	ctx := context.Background()
	repo, _, service := authoringFixture(t)
	settings, err := service.Settings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	settings.Budget.Passes = 9
	proposal, err := service.ProposeSettings(ctx, settings)
	if err != nil || proposal.Check.Status != "passed" {
		t.Fatalf("proposal = %+v, %v", proposal, err)
	}
	before, err := os.ReadFile(filepath.Join(repo.Dir, ".markitect", "execution.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	repo.Write("notes.md", "written after the proposal\n")
	repo.Commit("change the source after the proposal")
	p, err := service.Accept(ctx, proposal.ID)
	if err == nil || p.State != authoring.Failed || !strings.Contains(p.Error, "changed since the proposal") {
		t.Fatalf("accept on a changed source = %+v, %v", p, err)
	}
	after, err := os.ReadFile(filepath.Join(repo.Dir, ".markitect", "execution.yaml"))
	if err != nil || string(after) != string(before) {
		t.Fatalf("a stale proposal changed the working tree: %v", err)
	}
}
