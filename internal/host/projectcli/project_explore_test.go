package projectcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
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
	input := ".markitect/drafts/exploration.json"
	if err := os.MkdirAll(filepath.Join(repo, ".markitect", "drafts"), 0755); err != nil {
		t.Fatal(err)
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
	data, err := json.Marshal(inputRecord)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(input)), data, 0600); err != nil {
		t.Fatal(err)
	}

	var previewOut bytes.Buffer
	if err := runExplore(options{repo: repo, input: input}, &previewOut); err != nil {
		t.Fatalf("preview: %v", err)
	}
	var preview projectexplore.WritePlan
	if err := json.Unmarshal(previewOut.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if preview.Digest == "" || preview.Next.ID != inputRecord.ID || preview.Target.Exists {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if _, err := projectexplore.Load(repo, inputRecord.ID); err == nil {
		t.Fatal("read-only preview persisted the exploration")
	}

	var writeOut bytes.Buffer
	if err := runExplore(options{repo: repo, input: input, write: true, expect: preview.Digest}, &writeOut); err != nil {
		t.Fatalf("write: %v", err)
	}
	persisted, err := projectexplore.Load(repo, inputRecord.ID)
	if err != nil || persisted.Digest == "" || persisted.CreatedAgainst == "" {
		t.Fatalf("persisted record = %#v err=%v", persisted, err)
	}
	inputRecord.Decisions = []projectexplore.Decision{{
		ID: "fulfillment-boundary", ScopeIDs: []string{"cancellation"}, Question: "Does this exclude shipped orders?",
		Status: "answered", Answer: "Yes, shipped orders are excluded.", Authority: "explicit caller decision", Provenance: "request-17",
	}}
	data, err = json.Marshal(inputRecord)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(input)), data, 0600); err != nil {
		t.Fatal(err)
	}
	var updatePreviewOut bytes.Buffer
	if err := runExplore(options{repo: repo, input: input}, &updatePreviewOut); err != nil {
		t.Fatalf("update preview: %v", err)
	}
	var updatePreview projectexplore.WritePlan
	if err := json.Unmarshal(updatePreviewOut.Bytes(), &updatePreview); err != nil {
		t.Fatalf("decode update preview: %v", err)
	}
	if updatePreview.ExpectedStateDigest != persisted.Digest || len(updatePreview.Next.Decisions) != 1 {
		t.Fatalf("update preview is not bound to current record state: %#v", updatePreview)
	}
	var updateOut bytes.Buffer
	if err := runExplore(options{repo: repo, input: input, write: true, expect: updatePreview.Digest}, &updateOut); err != nil {
		t.Fatalf("update write: %v", err)
	}
	persisted, err = projectexplore.Load(repo, inputRecord.ID)
	if err != nil || len(persisted.Decisions) != 1 || persisted.Decisions[0].ID != "fulfillment-boundary" {
		t.Fatalf("updated record = %#v err=%v", persisted, err)
	}

	var statusOut bytes.Buffer
	if err := runExplore(options{repo: repo, explorationID: inputRecord.ID}, &statusOut); err != nil {
		t.Fatalf("status: %v", err)
	}
	var status projectexplore.Record
	if err := json.Unmarshal(statusOut.Bytes(), &status); err != nil || status.Digest != persisted.Digest {
		t.Fatalf("status = %#v err=%v", status, err)
	}
	var listOut bytes.Buffer
	if err := runExplore(options{repo: repo}, &listOut); err != nil {
		t.Fatalf("list: %v", err)
	}
	var records []projectexplore.Record
	if err := json.Unmarshal(listOut.Bytes(), &records); err != nil || len(records) != 1 || records[0].ID != inputRecord.ID {
		t.Fatalf("list = %#v err=%v", records, err)
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
	runGitWithEnv(t, repo, []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "commit", "-m", "configure fixture runtime")
}

func TestReadinessShowsExactBindingAndPreviewsExplicitAcknowledgement(t *testing.T) {
	repo := copyProjectWorld(t)
	configureReadinessTestModel(t, repo)
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
	managerIDs := []string{project.Report.Managers[0].ID}
	inputRecord := projectexplore.Record{
		APIVersion: projectexplore.APIVersion, ID: "readiness-work", Status: projectexplore.StatusActive,
		Request:   "Implement a bounded change.",
		Scopes:    []projectexplore.Scope{{ID: "bounded-change", Name: "Bounded change", Goal: "Implement the bounded change.", Operation: "apply", ManagerIDs: managerIDs}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{},
		Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	data, err := json.Marshal(inputRecord)
	if err != nil {
		t.Fatal(err)
	}
	input := ".markitect/drafts/readiness-exploration.json"
	if err := os.MkdirAll(filepath.Join(repo, ".markitect", "drafts"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(input)), data, 0600); err != nil {
		t.Fatal(err)
	}
	var createOut bytes.Buffer
	if err := runExplore(options{repo: repo, input: input}, &createOut); err != nil {
		t.Fatal(err)
	}
	var createPlan projectexplore.WritePlan
	if err := json.Unmarshal(createOut.Bytes(), &createPlan); err != nil {
		t.Fatal(err)
	}
	if _, err := projectexplore.Write(repo, createPlan, createPlan.Digest, createPlan.Binding); err != nil {
		t.Fatalf("persist initial exploration: %v", err)
	}

	var readinessOut bytes.Buffer
	ackOptions := options{repo: repo, explorationID: inputRecord.ID, scope: "bounded-change", acknowledgeStructure: true,
		actor: "caller", authority: "explicit user decision", decisionRef: "request-17", acknowledgedAt: "2026-10-09T12:00:00Z"}
	if err := runReadiness(ackOptions, &readinessOut); err != nil {
		t.Fatalf("readiness preview: %v", err)
	}
	var response struct {
		Binding   projectexplore.Binding         `json:"binding"`
		Readiness projectexplore.ReadinessReport `json:"readiness"`
		WritePlan *projectexplore.WritePlan      `json:"writePlan"`
	}
	if err := json.Unmarshal(readinessOut.Bytes(), &response); err != nil {
		t.Fatalf("decode readiness: %v", err)
	}
	if response.WritePlan == nil || response.Readiness.StructureDigest == "" || response.Readiness.BindingDigest == "" {
		t.Fatalf("readiness response omitted its exact binding/ack plan: %#v", response)
	}
	if len(response.Binding.FileStructure) == 0 {
		t.Fatalf("readiness response omitted exact file structure: %#v", response.Binding)
	}
	if _, err := projectexplore.Load(repo, inputRecord.ID); err != nil {
		t.Fatalf("read-only readiness preview mutated durable record: %v", err)
	}
	ackOptions.write, ackOptions.expect = true, response.WritePlan.Digest
	var writtenOut bytes.Buffer
	if err := runReadiness(ackOptions, &writtenOut); err != nil {
		t.Fatalf("write exact previewed acknowledgement: %v", err)
	}
	persisted, err := projectexplore.Load(repo, inputRecord.ID)
	if err != nil || len(persisted.Acknowledgements) != 1 || persisted.Acknowledgements[0].RecordedAt.Format(time.RFC3339) != ackOptions.acknowledgedAt {
		t.Fatalf("persisted acknowledgement = %#v err=%v", persisted.Acknowledgements, err)
	}
}

func configureReadinessTestModel(t *testing.T, repo string) {
	t.Helper()
	path := filepath.Join(repo, ".markitect", "model", "project-artifacts.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	marker := "  required: true\n"
	if strings.Count(text, marker) != 1 {
		t.Fatalf("expected one project-level required artifact, got %d", strings.Count(text, marker))
	}
	text = strings.Replace(text, marker, "  checks:\n    - namespace: engineering\n      name: cancellation-tests\n"+marker, 1)
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".markitect/model/project-artifacts.yaml")
	runGitWithEnv(t, repo, []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "commit", "--amend", "--no-edit")
}
