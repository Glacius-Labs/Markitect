package projectcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

func TestExplorePreviewWriteStatusAndList(t *testing.T) {
	repo := copyProjectWorld(t)
	if err := os.Remove(filepath.Join(repo, ".markitect", "runtime.yaml")); err != nil {
		t.Fatalf("remove optional runtime configuration: %v", err)
	}
	project, err := projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Report.Managers) == 0 {
		t.Fatal("fixture has no Managers")
	}
	inputRecord := projectexplore.Record{
		APIVersion: projectexplore.APIVersion,
		ID:         "first-order-work",
		Status:     projectexplore.StatusActive,
		Request:    "Support cancellation before fulfillment.",
		Scopes: []projectexplore.Scope{{
			ID: "cancellation", Name: "Cancellation", Goal: "Implement cancellation before shipment.",
			Operation: "apply", ManagerIDs: []string{project.Report.Managers[0].ID},
		}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{},
		Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	input := writeDraft(t, repo, ".markitect/drafts/exploration.json", inputRecord)

	preview := decodeOutput[projectapp.ExploreResult](t, mustCLI(t, "explore", "--repo", repo, "--input", input))
	if preview.Plan == nil || preview.Plan.Digest == "" || preview.Plan.Next.ID != inputRecord.ID || preview.Plan.Target.Exists || preview.Persisted != nil {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if _, err := projectexplore.Load(repo, inputRecord.ID); err == nil {
		t.Fatal("read-only preview persisted the exploration")
	}
	if code, _, errout := runCLI(t, "explore", "--repo", repo, "--input", input, "--expect", "sha256:stale", "--write"); code != 2 || !strings.Contains(errout, "exact exploration write-plan digest") {
		t.Fatalf("explore write with a stale digest exit=%d stderr=%s", code, errout)
	}
	if _, err := projectexplore.Load(repo, inputRecord.ID); err == nil {
		t.Fatal("stale explore write persisted the exploration")
	}

	written := decodeOutput[projectapp.ExploreResult](t, mustCLI(t, "explore", "--repo", repo, "--input", input, "--expect", preview.Plan.Digest, "--write"))
	persisted, err := projectexplore.Load(repo, inputRecord.ID)
	if err != nil || persisted.Digest == "" || persisted.CreatedAgainst == "" || written.Persisted == nil || written.Persisted.Digest != persisted.Digest {
		t.Fatalf("persisted record = %#v err=%v", persisted, err)
	}
	inputRecord.Decisions = []projectexplore.Decision{{
		ID: "fulfillment-boundary", ScopeIDs: []string{"cancellation"}, Question: "Does this exclude shipped orders?",
		Status: "answered", Answer: "Yes, shipped orders are excluded.", Authority: "explicit caller decision", Provenance: "request-17",
	}}
	writeDraft(t, repo, input, inputRecord)
	updatePreview := decodeOutput[projectapp.ExploreResult](t, mustCLI(t, "explore", "--repo", repo, "--input", input))
	if updatePreview.Plan == nil || updatePreview.Plan.ExpectedStateDigest != persisted.Digest || len(updatePreview.Plan.Next.Decisions) != 1 {
		t.Fatalf("update preview is not bound to current record state: %#v", updatePreview.Plan)
	}
	mustCLI(t, "explore", "--repo", repo, "--input", input, "--expect", updatePreview.Plan.Digest, "--write")
	persisted, err = projectexplore.Load(repo, inputRecord.ID)
	if err != nil || len(persisted.Decisions) != 1 || persisted.Decisions[0].ID != "fulfillment-boundary" {
		t.Fatalf("updated record = %#v err=%v", persisted, err)
	}

	status := decodeOutput[projectapp.ExploreResult](t, mustCLI(t, "explore", "--repo", repo, "--exploration", inputRecord.ID))
	if status.Record == nil || status.Record.Digest != persisted.Digest {
		t.Fatalf("status = %#v", status)
	}
	list := decodeOutput[projectapp.ExploreResult](t, mustCLI(t, "explore", "--repo", repo))
	if len(list.Records) != 1 || list.Records[0].ID != inputRecord.ID {
		t.Fatalf("list = %#v", list)
	}
}

// The project overview is read-only and must also work for a project that has
// no runtime configuration yet.
func TestStatusOverviewNeedsNoRuntimeConfiguration(t *testing.T) {
	repo := copyProjectWorld(t)
	if err := os.Remove(filepath.Join(repo, ".markitect", "runtime.yaml")); err != nil {
		t.Fatal(err)
	}
	summary := decodeOutput[overview](t, mustCLI(t, "status", "--repo", repo))
	if summary.Runtime.Configured || len(summary.Runs) != 0 || summary.Project.Name != "shop-cancellation" {
		t.Fatalf("status overview without runtime = %+v", summary)
	}
	if _, err := os.Stat(filepath.Join(repo, ".markitect", "runs")); !os.IsNotExist(err) {
		t.Fatalf("status overview created runtime state: %v", err)
	}
}

func writeExploreTestRuntime(t *testing.T, repo string, managers []projectmodel.Manager) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	agent := projectrun.Agent{
		Command: executable, Args: []string{"-test.run=^TestNothing$"}, Model: "fixture", ProviderVersion: "fixture-v1",
		Timeout: projectrun.Duration(30 * time.Second), MaxStdoutBytes: 64 << 10, MaxStderrBytes: 16 << 10,
		Environment: []string{"PATH"},
		Pricing:     projectrun.Pricing{InputMicrosPerMillion: 100, OutputMicrosPerMillion: 100},
	}
	agents, reviewers := map[string]projectrun.Agent{}, map[string]projectrun.Agent{}
	for _, manager := range managers {
		agents[manager.ID], reviewers[manager.ID] = agent, agent
	}
	runtimeConfig := projectrun.Runtime{
		APIVersion: projectrun.APIVersion, Mode: projectrun.ModeControlledLocal,
		Agents: agents, Review: &projectrun.ReviewConfig{Agents: reviewers, MaxRounds: 1, MaxManagerRounds: 1},
		Limits: projectrun.Limits{MaxDepth: 8, MaxStarts: 200, MaxRetries: 1, MaxParallel: 2,
			MaxDuration: projectrun.Duration(10 * time.Minute), MaxCostMicros: 100_000_000,
			MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 8 << 20},
	}
	data, err := yaml.Marshal(runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".markitect", "runtime.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".markitect/runtime.yaml")
	runGitWithEnv(t, repo, testCommitEnv, "commit", "-m", "configure fixture runtime")
}

func TestReadyShowsExactBindingAndPreviewsExplicitAcknowledgement(t *testing.T) {
	repo := copyProjectWorld(t)
	project, err := projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Report.Managers) == 0 {
		t.Fatal("fixture has no Managers")
	}
	writeExploreTestRuntime(t, repo, project.Report.Managers)
	project, err = projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	inputRecord := projectexplore.Record{
		APIVersion: projectexplore.APIVersion, ID: "readiness-work", Status: projectexplore.StatusActive,
		Request:   "Implement a bounded change.",
		Scopes:    []projectexplore.Scope{{ID: "bounded-change", Name: "Bounded change", Goal: "Implement the bounded change.", Operation: "apply", ManagerIDs: []string{project.Report.Managers[0].ID}}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{},
		Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	input := writeDraft(t, repo, ".markitect/drafts/readiness-exploration.json", inputRecord)
	created := decodeOutput[projectapp.ExploreResult](t, mustCLI(t, "explore", "--repo", repo, "--input", input))
	if created.Plan == nil {
		t.Fatalf("explore preview has no write plan: %#v", created)
	}
	if _, err := projectexplore.Write(repo, *created.Plan, created.Plan.Digest, created.Plan.Binding); err != nil {
		t.Fatalf("persist initial exploration: %v", err)
	}

	const acknowledgedAt = "2026-10-09T12:00:00Z"
	args := []string{"ready", "--repo", repo, "--exploration", inputRecord.ID, "--scope", "bounded-change", "--acknowledge",
		"--actor", "caller", "--authority", "explicit user decision", "--decision-ref", "request-17", "--acknowledged-at", acknowledgedAt}
	response := decodeOutput[projectapp.ReadinessResult](t, mustCLI(t, args...))
	if response.WritePlan == nil || response.Readiness.StructureDigest == "" || response.Readiness.BindingDigest == "" {
		t.Fatalf("readiness response omitted its exact binding/ack plan: %#v", response)
	}
	if len(response.Binding.FileStructure) == 0 {
		t.Fatalf("readiness response omitted exact file structure: %#v", response.Binding)
	}
	if persisted, err := projectexplore.Load(repo, inputRecord.ID); err != nil || len(persisted.Acknowledgements) != 0 {
		t.Fatalf("read-only readiness preview mutated durable record: %v", err)
	}
	if code, _, errout := runCLI(t, append(args, "--expect", "sha256:stale", "--write")...); code != 2 || !strings.Contains(errout, "markitect ready:") {
		t.Fatalf("ready write with a stale digest exit=%d stderr=%s", code, errout)
	}
	written := decodeOutput[projectapp.ReadinessResult](t, mustCLI(t, append(args, "--expect", response.WritePlan.Digest, "--write")...))
	persisted, err := projectexplore.Load(repo, inputRecord.ID)
	if err != nil || len(persisted.Acknowledgements) != 1 || persisted.Acknowledgements[0].RecordedAt.Format(time.RFC3339) != acknowledgedAt || written.Persisted == nil {
		t.Fatalf("persisted acknowledgement = %#v err=%v", persisted.Acknowledgements, err)
	}
	summary := decodeOutput[overview](t, mustCLI(t, "status", "--repo", repo))
	if len(summary.Explorations) != 1 || len(summary.Explorations[0].Scopes) != 1 || !summary.Explorations[0].Scopes[0].Acknowledged {
		t.Fatalf("status overview does not show the acknowledged scope: %+v", summary.Explorations)
	}
}
