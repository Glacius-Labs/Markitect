package projectcli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/mcp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

type contractCase struct {
	happy   func(*testing.T, string)
	noWrite func(*testing.T, string)
}

type contractResult struct {
	Code       int
	Data       json.RawMessage
	Diagnostic *mcp.Diagnostic
}

type contractCaller struct {
	t       *testing.T
	surface string
	root    string
	ops     projectapp.Operations
}

func newContractCaller(t *testing.T, surface, root string, ops projectapp.Operations) *contractCaller {
	t.Helper()
	return &contractCaller{t: t, surface: surface, root: root, ops: ops}
}

// Invoke supplies independent argv and JSON inputs to the real transport
// adapters; the facade and all product use cases remain in the call path.
func (c *contractCaller) Invoke(name string, argv []string, input any) contractResult {
	c.t.Helper()
	if c.surface == "cli" {
		args := append([]string{name, "--repo", c.root}, argv...)
		code, output, _ := runCLIWith(c.t, c.ops, args...)
		return contractResult{Code: code, Data: json.RawMessage(output)}
	}
	server, err := newMCPServer(env{root: c.root, ops: c.ops})
	if err != nil {
		c.t.Fatal(err)
	}
	if input == nil {
		input = map[string]any{}
	}
	result, err := server.Call(context.Background(), name, mustRaw(c.t, input))
	if err != nil {
		c.t.Fatalf("%s MCP arguments: %v", name, err)
	}
	if err := server.ValidateResult(name, result); err != nil {
		c.t.Fatalf("%s MCP result does not match its published schema: %v", name, err)
	}
	outcome := decodeOutput[mcp.Outcome[json.RawMessage]](c.t, mustRaw(c.t, result.StructuredContent))
	code := 0
	if result.IsError {
		code = 1
		if outcome.Diagnostic != nil && outcome.Diagnostic.Code == "invalid_arguments" {
			code = 2
		}
	}
	var data json.RawMessage
	if outcome.Data != nil {
		data = *outcome.Data
	}
	return contractResult{Code: code, Data: data, Diagnostic: outcome.Diagnostic}
}

func (c *contractCaller) Success(name string, argv []string, input any) json.RawMessage {
	c.t.Helper()
	result := c.Invoke(name, argv, input)
	if result.Code != 0 || !json.Valid(result.Data) {
		c.t.Fatalf("%s %s: code=%d data=%s diagnostic=%+v", c.surface, name, result.Code, result.Data, result.Diagnostic)
	}
	return result.Data
}

// NoWrite includes all project files and directories, including private state
// and drafts. Git's internal metadata is outside the product write contract.
func (c *contractCaller) NoWrite(run func()) {
	c.t.Helper()
	before := contractInventory(c.t, c.root)
	run()
	if after := contractInventory(c.t, c.root); !reflect.DeepEqual(before, after) {
		paths := map[string]bool{}
		for path := range before {
			paths[path] = true
		}
		for path := range after {
			paths[path] = true
		}
		changed := []string{}
		for path := range paths {
			if before[path] != after[path] {
				changed = append(changed, fmt.Sprintf("%s: %q -> %q", path, before[path], after[path]))
			}
		}
		slices.Sort(changed)
		c.t.Fatalf("%s invocation without authorization changed project files/directories: %s", c.surface, strings.Join(changed, "; "))
	}
}

func contractInventory(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == ".git" {
			return filepath.SkipDir
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// NTFS directory timestamps drifted on a usage-rejection path that
			// returns before any project I/O; they are not a state-write signal.
			// Membership/modes and every file's bytes/mode/mtime remain checked.
			files[rel] = fmt.Sprintf("directory:%s", info.Mode())
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = fmt.Sprintf("file:%s:%d:%x", info.Mode(), info.ModTime().UnixNano(), sha256.Sum256(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// The initial committed guided model has no configured provider, so these
// fixtures cannot accidentally start an agent. Git identity/dates are hermetic.
func newContractModel(t *testing.T) *testkit.Repo {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Git("switch", "-c", "contract-fixture")
	plan, err := projectwork.Init(repo.Dir, "contract-project", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range plan.Files {
		content := file.Content
		if file.Path == projectwork.ManifestPath {
			content = strings.Replace(content, "coverageMode: full", "coverageMode: selected", 1)
		}
		repo.Write(file.Path, content)
	}
	repo.Commit("contract model")
	return repo
}

type contractRejectingInvoker struct{ calls int }

func (i *contractRejectingInvoker) Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	return agentexec.RunResult{}, fmt.Errorf("contract forbids starting an agent")
}
func (*contractRejectingInvoker) Fingerprint(agentexec.Config) (string, error) {
	return "contract", nil
}

func inspectionContractCases() map[string]contractCase {
	return map[string]contractCase{
		"schema":  {inspectionSchemaHappy, inspectionSchemaNoWrite},
		"check":   {inspectionCheckHappy, inspectionCheckNoWrite},
		"model":   {inspectionModelHappy, inspectionModelNoWrite},
		"context": {inspectionContextHappy, inspectionContextNoWrite},
		"impact":  {inspectionImpactHappy, inspectionImpactNoWrite},
		"docs":    {inspectionDocsHappy, inspectionDocsNoWrite},
		"edit":    {inspectionEditHappy, inspectionEditNoWrite},
		"explore": {inspectionExploreHappy, inspectionExploreNoWrite},
		"ready":   {inspectionReadyHappy, inspectionReadyNoWrite},
		"brief":   {inspectionBriefHappy, inspectionBriefNoWrite},
		"status":  {inspectionStatusHappy, inspectionStatusNoWrite},
	}
}

func inspectionCaller(t *testing.T, surface string) (*contractCaller, *contractRejectingInvoker, *testkit.Repo) {
	t.Helper()
	repo := newContractModel(t)
	invoker := &contractRejectingInvoker{}
	ops := projectapp.NewOperations()
	ops.Invoker = invoker
	return newContractCaller(t, surface, repo.Dir, ops), invoker, repo
}

func inspectionSchemaHappy(t *testing.T, surface string) {
	caller, invoker, _ := inspectionCaller(t, surface)
	inspectionSchema(t, caller)
	assertNoAgents(t, invoker)
}
func inspectionSchemaNoWrite(t *testing.T, surface string) {
	caller, invoker, _ := inspectionCaller(t, surface)
	caller.NoWrite(func() { inspectionSchema(t, caller) })
	assertNoAgents(t, invoker)
}
func inspectionSchema(t *testing.T, caller *contractCaller) {
	t.Helper()
	data := caller.Success("schema", nil, map[string]any{})
	if !strings.Contains(string(data), "project.markitect.example.org/v1alpha1") {
		t.Fatalf("schema omitted the project API version: %s", data)
	}
}

func inspectionCheckHappy(t *testing.T, surface string) {
	caller, invoker, repo := inspectionCaller(t, surface)
	inspectionCheck(t, caller, repo)
	assertNoAgents(t, invoker)
}
func inspectionCheckNoWrite(t *testing.T, surface string) {
	caller, invoker, repo := inspectionCaller(t, surface)
	caller.NoWrite(func() { inspectionCheck(t, caller, repo) })
	assertNoAgents(t, invoker)
}
func inspectionCheck(t *testing.T, caller *contractCaller, repo *testkit.Repo) {
	t.Helper()
	report := decodeOutput[checkReport](t, caller.Success("check", []string{"--revision", "HEAD"}, map[string]any{"revision": "HEAD"}))
	if report.Status != "succeeded" || report.ProjectDigest == "" || report.Revision != contractGitHead(t, repo.Dir) {
		t.Fatalf("check did not report the committed fixture: %+v", report)
	}
}

func inspectionModelHappy(t *testing.T, surface string) {
	caller, invoker, repo := inspectionCaller(t, surface)
	inspectionModel(t, caller, repo)
	assertNoAgents(t, invoker)
}
func inspectionModelNoWrite(t *testing.T, surface string) {
	caller, invoker, repo := inspectionCaller(t, surface)
	caller.NoWrite(func() { inspectionModel(t, caller, repo) })
	assertNoAgents(t, invoker)
}
func inspectionModel(t *testing.T, caller *contractCaller, repo *testkit.Repo) {
	t.Helper()
	result := decodeOutput[modelResult](t, caller.Success("model", []string{"--revision", "HEAD"}, map[string]any{"revision": "HEAD"}))
	if result.Revision != contractGitHead(t, repo.Dir) || len(result.Managers) != 1 || result.Managers[0].Name != "project-owner" {
		t.Fatalf("model result did not identify the fixture Manager: %+v", result)
	}
}

func inspectionContextHappy(t *testing.T, surface string) {
	caller, invoker, repo := inspectionCaller(t, surface)
	inspectionContext(t, caller, repo)
	assertNoAgents(t, invoker)
}
func inspectionContextNoWrite(t *testing.T, surface string) {
	repo, _, revision := inspectionChangedModelFixture(t)
	invoker := &contractRejectingInvoker{}
	ops := projectapp.NewOperations()
	ops.Invoker = invoker
	caller := newContractCaller(t, surface, repo.Dir, ops)
	manager := inspectionManagerID(t, caller, repo)
	accepted, err := projectbriefing.ReadAcceptedHistory(repo.Dir, revision)
	if err != nil || len(accepted.Receipt.Bundles) != 1 {
		t.Fatalf("fixture has no prospective accepted-history change: receipt=%+v err=%v", accepted.Receipt, err)
	}
	store := filepath.Join(repo.Dir, filepath.FromSlash(".markitect/state/briefings/history.json"))
	if _, err := os.Stat(store); !os.IsNotExist(err) {
		t.Fatalf("fixture materialized accepted history before context: %v", err)
	}
	caller.NoWrite(func() {
		data := caller.Success("context", []string{manager, "--revision", revision}, map[string]any{"manager": manager, "revision": revision})
		result := decodeOutput[contextResult](t, data)
		if result.Revision != revision || result.Manager.ID != manager {
			t.Fatalf("context returned another Manager: %+v", result.Manager)
		}
	})
	if _, err := os.Stat(store); !os.IsNotExist(err) {
		t.Fatalf("context persisted prospective accepted history: %v", err)
	}
	assertNoAgents(t, invoker)
}
func inspectionContext(t *testing.T, caller *contractCaller, repo *testkit.Repo) {
	t.Helper()
	revision := contractGitHead(t, repo.Dir)
	manager := inspectionManagerID(t, caller, repo)
	result := decodeOutput[contextResult](t, caller.Success("context", []string{manager, "--revision", revision}, map[string]any{"manager": manager, "revision": revision}))
	if result.Revision != revision || result.Manager.ID != manager || result.Manager.Name != "project-owner" {
		t.Fatalf("context is not for the selected Manager: %+v", result.Manager)
	}
}

func inspectionManagerID(t *testing.T, caller *contractCaller, repo *testkit.Repo) string {
	t.Helper()
	revision := contractGitHead(t, repo.Dir)
	index, err := caller.ops.Index(projectapp.Selection{Root: repo.Dir, Revision: revision})
	if err != nil || len(index.Managers) != 1 {
		t.Fatalf("load fixture Manager identity: %+v err=%v", index, err)
	}
	return index.Managers[0].ID
}

func inspectionImpactFixture(t *testing.T, surface string) (*contractCaller, *contractRejectingInvoker, string, string) {
	t.Helper()
	repo, since, revision := inspectionChangedModelFixture(t)
	invoker := &contractRejectingInvoker{}
	ops := projectapp.NewOperations()
	ops.Invoker = invoker
	return newContractCaller(t, surface, repo.Dir, ops), invoker, since, revision
}

func inspectionChangedModelFixture(t *testing.T) (*testkit.Repo, string, string) {
	t.Helper()
	repo := newContractModel(t)
	since := contractGitHead(t, repo.Dir)
	path := filepath.Join(repo.Dir, ".markitect", "model", "manager.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(content), "repository-wide engineering mandate", "repository-wide project mandate", 1)
	if changed == string(content) {
		t.Fatal("fixture Manager purpose did not contain the expected text")
	}
	repo.Write(".markitect/model/manager.yaml", changed)
	revision := repo.Commit("change the accepted Manager purpose")
	return repo, since, revision
}
func inspectionImpactHappy(t *testing.T, surface string) {
	caller, invoker, since, revision := inspectionImpactFixture(t, surface)
	inspectionImpact(t, caller, since, revision)
	assertNoAgents(t, invoker)
}
func inspectionImpactNoWrite(t *testing.T, surface string) {
	caller, invoker, since, revision := inspectionImpactFixture(t, surface)
	caller.NoWrite(func() { inspectionImpact(t, caller, since, revision) })
	assertNoAgents(t, invoker)
}
func inspectionImpact(t *testing.T, caller *contractCaller, since, revision string) {
	t.Helper()
	result := decodeOutput[impactResult](t, caller.Success("impact", []string{"--since", since, "--revision", revision}, map[string]any{"since": since, "revision": revision}))
	if result.Since != since || result.Revision != revision || result.Digest == "" || result.BaseDigest == "" || result.CandidateDigest == "" || len(result.ChangedDefinitions) != 1 || result.ChangedDefinitions[0] != contractManagerID() {
		t.Fatalf("impact omitted the changed Manager definition: %+v", result.ChangeImpact)
	}
}

func inspectionDocsHappy(t *testing.T, surface string) {
	caller, invoker, _ := inspectionCaller(t, surface)
	inspectionDocs(t, caller)
	assertNoAgents(t, invoker)
}
func inspectionDocsNoWrite(t *testing.T, surface string) {
	caller, invoker, _ := inspectionCaller(t, surface)
	caller.NoWrite(func() { inspectionDocs(t, caller) })
	assertNoAgents(t, invoker)
}
func inspectionDocs(t *testing.T, caller *contractCaller) {
	t.Helper()
	result := decodeOutput[projectapp.DocumentResult](t, caller.Success("docs", nil, map[string]any{}))
	if result.Written || result.Digest == "" || result.Path != projectwork.DefaultDocumentPath || !strings.Contains(result.Content, "project-owner") {
		t.Fatalf("docs preview is incomplete or persisted: %+v", result)
	}
}

func inspectionEditFixture(t *testing.T, surface string) (*contractCaller, *contractRejectingInvoker, string) {
	t.Helper()
	caller, invoker, repo := inspectionCaller(t, surface)
	project, err := projectwork.Load(repo.Dir, "")
	if err != nil {
		t.Fatal(err)
	}
	const path = ".markitect/model/manager.yaml"
	content, err := os.ReadFile(filepath.Join(repo.Dir, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(content), "repository-wide engineering mandate", "repository-wide product mandate", 1)
	if changed == string(content) {
		t.Fatal("fixture Manager purpose did not contain the expected text")
	}
	mutation := projectwork.Mutation{APIVersion: projectwork.APIVersion, BaseDigest: project.Digest, Actor: projectwork.HumanActor,
		Goal: "Clarify the project-owner mandate", Files: []projectwork.FileChange{{Path: path, Content: changed}}}
	encoded, err := projectwork.EncodeMutation(mutation)
	if err != nil {
		t.Fatal(err)
	}
	inputPath := writeDraft(t, repo.Dir, ".markitect/drafts/manager-purpose.json", encoded)
	return caller, invoker, inputPath
}
func inspectionEdit(t *testing.T, caller *contractCaller, inputPath string) {
	t.Helper()
	input := map[string]any{"input": readContractDraft(t, caller.root, inputPath)}
	plan := decodeOutput[projectwork.EditPlan](t, caller.Success("edit", []string{"--input", inputPath}, input))
	if plan.Digest == "" || plan.Mutation.Goal != "Clarify the project-owner mandate" || len(plan.Mutation.Files) != 1 || plan.Mutation.Files[0].Path != ".markitect/model/manager.yaml" || !strings.Contains(plan.Mutation.Files[0].Content, "repository-wide product mandate") || plan.CandidateDigest == "" || plan.CandidateDigest == plan.BaseDigest {
		t.Fatalf("edit did not return the valid mutation preview: %+v", plan)
	}
}

func inspectionBriefFixture(t *testing.T, surface string) (*contractCaller, *contractRejectingInvoker, string, string) {
	t.Helper()
	repo, since, revision := inspectionChangedModelFixture(t)
	invoker := &contractRejectingInvoker{}
	ops := projectapp.NewOperations()
	ops.Invoker = invoker
	return newContractCaller(t, surface, repo.Dir, ops), invoker, since, revision
}
func inspectionBriefHappy(t *testing.T, surface string) {
	caller, invoker, since, revision := inspectionBriefFixture(t, surface)
	const provenance = "contract-briefing"
	argv := []string{"--since", since, "--revision", revision, "--provenance", provenance}
	input := map[string]any{"since": since, "revision": revision, "provenance": provenance}
	caller.NoWrite(func() {
		result := decodeOutput[struct {
			Bundle      projectbriefing.Bundle `json:"bundle"`
			StateDigest string                 `json:"stateDigest"`
		}](t, caller.Success("brief", argv, input))
		if len(result.Bundle.Events) != 1 || result.Bundle.Digest == "" || result.StateDigest == "" || result.Bundle.SinceRevision != since || result.Bundle.Revision != revision {
			t.Fatalf("brief preview omitted accepted-history event/digest: %+v", result)
		}
		event := result.Bundle.Events[0]
		if event.DefinitionID.Kind != "Manager" || event.DefinitionID.Name != "project-owner" || event.Change != "modified" || event.After == nil || event.After.Purpose != "Owns the repository-wide project mandate and delegates bounded responsibility." {
			t.Fatalf("brief preview did not describe the expected Manager-purpose change: %+v", event)
		}
	})
	assertNoAgents(t, invoker)
}
func inspectionBriefNoWrite(t *testing.T, surface string) {
	caller, invoker, _, revision := inspectionBriefFixture(t, surface)
	accepted, err := projectbriefing.ReadAcceptedHistory(caller.root, revision)
	if err != nil || len(accepted.Receipt.Bundles) != 1 || len(accepted.Receipt.Bundles[0].Events) != 1 {
		t.Fatalf("fixture has no expected accepted-history event: receipt=%+v err=%v", accepted.Receipt, err)
	}
	expectedEvent := accepted.Receipt.Bundles[0].Events[0]
	caller.NoWrite(func() {
		result := decodeOutput[briefListOutput](t, caller.Success("brief", []string{"list"}, map[string]any{"action": "list"}))
		if len(result.Notifications) != 1 || result.StateDigest == "" || result.Notifications[0].EventID != expectedEvent.ID || result.Notifications[0].Revision != revision || result.Notifications[0].DefinitionID.Kind != "Manager" || result.Notifications[0].DefinitionID.Name != "project-owner" {
			t.Fatalf("brief list omitted predicted accepted history: %+v", result)
		}
	})
	assertNoAgents(t, invoker)
}

func inspectionStatusHappy(t *testing.T, surface string) {
	caller, invoker, _ := inspectionCaller(t, surface)
	inspectionStatus(t, caller)
	assertNoAgents(t, invoker)
}
func inspectionStatusNoWrite(t *testing.T, surface string) {
	caller, invoker, _ := inspectionCaller(t, surface)
	caller.NoWrite(func() { inspectionStatus(t, caller) })
	assertNoAgents(t, invoker)
}
func inspectionStatus(t *testing.T, caller *contractCaller) {
	t.Helper()
	result := decodeOutput[overview](t, caller.Success("status", nil, map[string]any{}))
	if result.Project.Name != "contract-project" || len(result.Runs) != 0 {
		t.Fatalf("status overview is not bound to the fixture project: %+v", result)
	}
}
func inspectionEditHappy(t *testing.T, surface string) {
	caller, invoker, inputPath := inspectionEditFixture(t, surface)
	inspectionEdit(t, caller, inputPath)
	assertNoAgents(t, invoker)
}
func inspectionEditNoWrite(t *testing.T, surface string) {
	caller, invoker, inputPath := inspectionEditFixture(t, surface)
	caller.NoWrite(func() { inspectionEdit(t, caller, inputPath) })
	assertNoAgents(t, invoker)
}
func readContractDraft(t *testing.T, root, path string) json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func inspectionExploreFixture(t *testing.T, surface string) (*contractCaller, *contractRejectingInvoker, projectexplore.Record, string) {
	t.Helper()
	caller, invoker, repo := inspectionCaller(t, surface)
	index, err := caller.ops.Index(projectapp.Selection{Root: repo.Dir})
	if err != nil || len(index.Managers) != 1 {
		t.Fatalf("load fixture Manager identity: %+v err=%v", index, err)
	}
	record := projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: "contract-exploration", Status: projectexplore.StatusActive,
		Request:   "Inspect a bounded implementation scope",
		Scopes:    []projectexplore.Scope{{ID: "contract-scope", Name: "Contract scope", Goal: "Implement the project-owner mandate.", Operation: "apply", ManagerIDs: []string{index.Managers[0].ID}}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}
	inputPath := writeDraft(t, repo.Dir, ".markitect/drafts/contract-exploration.json", record)
	return caller, invoker, record, inputPath
}
func inspectionExplore(t *testing.T, caller *contractCaller, record projectexplore.Record, inputPath string) {
	t.Helper()
	var argv []string
	var input any
	if caller.surface == "cli" {
		argv = []string{"--input", inputPath}
	} else {
		input = map[string]any{"input": record}
	}
	result := decodeOutput[projectapp.ExploreResult](t, caller.Success("explore", argv, input))
	if result.Plan == nil || result.Plan.Digest == "" || result.Plan.Next.ID != record.ID || result.Plan.Target.Exists || result.Persisted != nil {
		t.Fatalf("explore did not return the scoped preview: %+v", result)
	}
}
func inspectionExploreHappy(t *testing.T, surface string) {
	caller, invoker, record, inputPath := inspectionExploreFixture(t, surface)
	inspectionExplore(t, caller, record, inputPath)
	assertNoAgents(t, invoker)
}
func inspectionExploreNoWrite(t *testing.T, surface string) {
	caller, invoker, record, inputPath := inspectionExploreFixture(t, surface)
	caller.NoWrite(func() { inspectionExplore(t, caller, record, inputPath) })
	assertNoAgents(t, invoker)
}

func inspectionReadyFixture(t *testing.T, surface string) (*contractCaller, *contractRejectingInvoker, string, string) {
	t.Helper()
	_, ops, repo, _ := newRuntimeContract(t, surface, false)
	invoker := &contractRejectingInvoker{}
	ops.Invoker = invoker
	caller := newContractCaller(t, surface, repo.Dir, ops)
	const explorationID, scopeID = "contract-readiness", "contract-scope"
	goal := "Implement the modeled value behavior."
	binding, err := projectrun.ExplorationBinding(contractProjectHost(), repo.Dir, contractGitHead(t, repo.Dir), projectrun.PlanRequest{
		ScopeID: scopeID, Goal: goal, Operation: projectrun.OperationApply, Managers: []string{contractManagerID()},
	})
	if err != nil {
		t.Fatal(err)
	}
	record := projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: explorationID, Status: projectexplore.StatusActive, Request: goal,
		Scopes:    []projectexplore.Scope{{ID: scopeID, Name: scopeID, Goal: goal, Operation: projectrun.OperationApply, ManagerIDs: []string{contractManagerID()}}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}
	preview, err := projectexplore.CreatePreview(repo.Dir, record, binding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectexplore.Write(repo.Dir, preview, preview.Digest, binding); err != nil {
		t.Fatal(err)
	}
	return caller, invoker, explorationID, scopeID
}
func inspectionReady(t *testing.T, caller *contractCaller, exploration, scope string) {
	t.Helper()
	const timestamp = "2026-10-12T12:00:00Z"
	argv := []string{"--exploration", exploration, "--scope", scope, "--acknowledge", "--actor", "fixture-user", "--authority", "contract test preview", "--decision-ref", "contract-design", "--acknowledged-at", timestamp}
	input := map[string]any{"exploration": exploration, "scope": scope, "acknowledge": true, "actor": "fixture-user", "authority": "contract test preview", "decisionRef": "contract-design", "acknowledgedAt": timestamp}
	if caller.surface == "mcp" {
		argv = nil
	}
	result := decodeOutput[projectapp.ReadinessResult](t, caller.Success("ready", argv, input))
	if result.WritePlan == nil || result.WritePlan.Digest == "" || result.Readiness.BindingDigest == "" || result.Readiness.StructureDigest == "" || len(result.Binding.FileStructure) == 0 || !strings.Contains(strings.Join(result.Binding.FileStructure, "\n"), "src/value.txt") {
		t.Fatalf("ready did not return the exact acknowledgement preview: %+v", result)
	}
	record, err := projectexplore.Load(caller.root, exploration)
	if err != nil || len(record.Acknowledgements) != 0 {
		t.Fatalf("ready preview persisted acknowledgement: %+v err=%v", record.Acknowledgements, err)
	}
}
func inspectionReadyHappy(t *testing.T, surface string) {
	caller, invoker, exploration, scope := inspectionReadyFixture(t, surface)
	inspectionReady(t, caller, exploration, scope)
	assertNoAgents(t, invoker)
}
func inspectionReadyNoWrite(t *testing.T, surface string) {
	caller, invoker, exploration, scope := inspectionReadyFixture(t, surface)
	caller.NoWrite(func() { inspectionReady(t, caller, exploration, scope) })
	assertNoAgents(t, invoker)
}
