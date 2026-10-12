package projectcli

import "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectonboarding"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"

	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

// adoptionContractCases covers setup and Brownfield adoption at both public
// transports while keeping writes behind the product's preview digests.
func adoptionContractCases() map[string]contractCase {
	return map[string]contractCase{
		"init": {
			happy: func(t *testing.T, surface string) {
				repo := emptyContractRepo(t, "contract-init")
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				result := caller.Success("init", []string{"--name", "contract-init"}, map[string]any{"name": "contract-init"})
				assertInitPreview(t, result, "contract-init")
				assertNoAgents(t, invoker)
			},
			noWrite: func(t *testing.T, surface string) {
				repo := emptyContractRepo(t, "contract-init-no-write")
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				caller.NoWrite(func() {
					result := caller.Success("init", []string{"--name", "contract-init-no-write"}, map[string]any{"name": "contract-init-no-write"})
					assertInitPreview(t, result, "contract-init-no-write")
				})
				assertNoAgents(t, invoker)
			},
		},
		"config": {
			happy: func(t *testing.T, surface string) {
				repo := newContractModel(t)
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				executable := testExecutable(t)
				data := caller.Success("config", configArgs(surface, executable), contractConfigInput(surface, executable))
				assertConfigPreview(t, data)
				assertNoAgents(t, invoker)
			},
			noWrite: func(t *testing.T, surface string) {
				repo := newContractModel(t)
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				executable := testExecutable(t)
				caller.NoWrite(func() {
					data := caller.Success("config", configArgs(surface, executable), contractConfigInput(surface, executable))
					assertConfigPreview(t, data)
				})
				assertNoAgents(t, invoker)
			},
		},
		"onboard": {
			happy: func(t *testing.T, surface string) {
				repo := newContractModel(t)
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				data := caller.Success("onboard", []string{"--provider", "codex"}, map[string]any{"provider": "codex"})
				assertOnboardPreview(t, data)
				assertNoAgents(t, invoker)
			},
			noWrite: func(t *testing.T, surface string) {
				repo := newContractModel(t)
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				caller.NoWrite(func() {
					data := caller.Success("onboard", []string{"--provider", "codex"}, map[string]any{"provider": "codex"})
					assertOnboardPreview(t, data)
				})
				assertNoAgents(t, invoker)
			},
		},
		"doctor": {
			happy: func(t *testing.T, surface string) {
				repo := newContractModel(t)
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				executable := testExecutable(t)
				data := caller.Success("doctor", doctorArgs(surface, executable), map[string]any{"provider": "process", "providerExecutable": executable})
				assertDoctorReport(t, data)
				assertNoAgents(t, invoker)
			},
			noWrite: func(t *testing.T, surface string) {
				repo := newContractModel(t)
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				executable := testExecutable(t)
				caller.NoWrite(func() {
					data := caller.Success("doctor", doctorArgs(surface, executable), map[string]any{"provider": "process", "providerExecutable": executable})
					assertDoctorReport(t, data)
				})
				assertNoAgents(t, invoker)
			},
		},
		"adopt": {
			happy: func(t *testing.T, surface string) {
				repo, commit, request := adoptionFixture(t)
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				inputPath := writeAdoptInput(t, repo.Dir, "start", projectapp.BrownfieldStartInput{Request: request, ScopeStatuses: []projectadoption.ScopeStatus{}})
				previewResult := caller.Invoke("adopt", adoptArgs(surface, "start", "", commit, inputPath, "", false), contractAdoptInput(surface, "start", "", commit, projectapp.BrownfieldInput{Start: &projectapp.BrownfieldStartInput{Request: request, ScopeStatuses: []projectadoption.ScopeStatus{}}}, "", false))
				if previewResult.Code != 0 || !json.Valid(previewResult.Data) {
					t.Fatalf("adopt start preview: code=%d data=%s diagnostic=%+v", previewResult.Code, previewResult.Data, previewResult.Diagnostic)
				}
				preview := decodeOutput[projectapp.BrownfieldResult](t, previewResult.Data)
				assertAdoptPreview(t, preview, request.ID, commit)

				writtenResult := caller.Invoke("adopt", adoptArgs(surface, "start", "", commit, inputPath, preview.SessionDigest, true), contractAdoptInput(surface, "start", "", commit, projectapp.BrownfieldInput{Start: &projectapp.BrownfieldStartInput{Request: request, ScopeStatuses: []projectadoption.ScopeStatus{}}}, preview.SessionDigest, true))
				if writtenResult.Code != 0 || !json.Valid(writtenResult.Data) {
					t.Fatalf("adopt start write: code=%d data=%s diagnostic=%+v", writtenResult.Code, writtenResult.Data, writtenResult.Diagnostic)
				}
				written := decodeOutput[projectapp.BrownfieldResult](t, writtenResult.Data)
				if written.Status != "recorded" || written.SessionDigest != preview.SessionDigest || written.Session == nil || written.Session.ID != request.ID {
					t.Fatalf("adopt start write did not preserve preview identity: %+v", written)
				}

				statusResult := caller.Invoke("adopt", adoptArgs(surface, "status", request.ID, "", "", "", false), map[string]any{"action": "status", "session": request.ID})
				if statusResult.Code != 0 || !json.Valid(statusResult.Data) {
					t.Fatalf("adopt status: code=%d data=%s diagnostic=%+v", statusResult.Code, statusResult.Data, statusResult.Diagnostic)
				}
				status := decodeOutput[projectapp.BrownfieldResult](t, statusResult.Data)
				if status.Status != "status" || status.SessionDigest != written.SessionDigest || status.Session == nil || status.Session.ID != request.ID || status.Session.Source.Commit != commit || status.Session.Target.Revision != commit {
					t.Fatalf("adopt status lost durable session identity: %+v", status)
				}
				assertNoAgents(t, invoker)
			},
			noWrite: func(t *testing.T, surface string) {
				repo, commit, request := adoptionFixture(t)
				caller, invoker := adoptionCaller(t, surface, repo.Dir)
				inputPath := writeAdoptInput(t, repo.Dir, "start", projectapp.BrownfieldStartInput{Request: request, ScopeStatuses: []projectadoption.ScopeStatus{}})
				caller.NoWrite(func() {
					result := caller.Invoke("adopt", adoptArgs(surface, "start", "", commit, inputPath, "", false), contractAdoptInput(surface, "start", "", commit, projectapp.BrownfieldInput{Start: &projectapp.BrownfieldStartInput{Request: request, ScopeStatuses: []projectadoption.ScopeStatus{}}}, "", false))
					if result.Code != 0 || !json.Valid(result.Data) {
						t.Fatalf("adopt start preview: code=%d data=%s diagnostic=%+v", result.Code, result.Data, result.Diagnostic)
					}
					assertAdoptPreview(t, decodeOutput[projectapp.BrownfieldResult](t, result.Data), request.ID, commit)
				})
				assertNoAgents(t, invoker)
			},
		},
	}
}

func adoptionCaller(t *testing.T, surface, root string) (*contractCaller, *contractRejectingInvoker) {
	t.Helper()
	invoker := &contractRejectingInvoker{}
	ops := projectapp.NewOperations()
	ops.Invoker = invoker
	return newContractCaller(t, surface, root, ops), invoker
}

func emptyContractRepo(t *testing.T, branch string) *testkit.Repo {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Git("switch", "-c", branch)
	return repo
}

func configArgs(surface, executable string) []string {
	if surface != "cli" {
		return nil
	}
	return []string{"--provider", "process", "--model", "scripted", "--provider-executable", executable, "--cost-mode", "unmetered", "--max-cost-micros", "1"}
}

func doctorArgs(surface, executable string) []string {
	if surface != "cli" {
		return nil
	}
	return []string{"--provider", "process", "--provider-executable", executable}
}

func contractConfigInput(surface, executable string) any {
	if surface == "cli" {
		return nil
	}
	return map[string]any{"provider": "process", "model": "scripted", "providerExecutable": executable, "costMode": "unmetered", "maxCostMicros": 1}
}

func testExecutable(t *testing.T) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return executable
}

func adoptArgs(surface, action, session, revision, inputPath, expected string, write bool) []string {
	if surface != "cli" {
		return nil
	}
	args := []string{action}
	if session != "" {
		args = append(args, "--session", session)
	}
	if revision != "" {
		args = append(args, "--revision", revision)
	}
	if inputPath != "" {
		args = append(args, "--input", inputPath)
	}
	if expected != "" {
		args = append(args, "--expect", expected)
	}
	if write {
		args = append(args, "--write")
	}
	return args
}

func contractAdoptInput(surface, action, session, revision string, input projectapp.BrownfieldInput, expected string, write bool) any {
	if surface != "mcp" {
		return nil
	}
	value := map[string]any{"action": action}
	if session != "" {
		value["session"] = session
	}
	if revision != "" {
		value["revision"] = revision
	}
	if action != "status" {
		value["input"] = input
	}
	if expected != "" {
		value["expect"] = expected
	}
	if write {
		value["write"] = true
	}
	return value
}

func adoptionFixture(t *testing.T) (*testkit.Repo, string, projectadoption.DiscoveryRequest) {
	t.Helper()
	repo := newContractModel(t)
	repo.Write("docs/cancellation.md", "# Cancellation\n\nA caller may cancel an in-flight operation.\n")
	commit := repo.Commit("selected adoption evidence")
	request := projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion,
		ID:         "contract-adoption",
		Purpose:    "Create a bounded reverse-modeling session",
		Review:     "owner-review-contract-adoption",
		Commit:     commit,
		ScopeRoots: []string{"docs"},
		Selected: []projectadoption.SelectedPath{{
			ID: "cancellation-document", Path: "docs/cancellation.md", Reason: "Owner-selected current behavior", Basis: "documentation",
		}},
		Exclusions: []projectadoption.PathReason{},
		Unselected: []projectadoption.PathReason{},
	}
	return repo, commit, request
}

func assertInitPreview(t *testing.T, data json.RawMessage, name string) {
	t.Helper()
	plan := decodeOutput[struct {
		Name   string `json:"name"`
		Digest string `json:"digest"`
		Files  []struct {
			Path string `json:"path"`
		} `json:"files"`
	}](t, data)
	if plan.Name != name || plan.Digest == "" || len(plan.Files) == 0 {
		t.Fatalf("init preview is incomplete: %+v", plan)
	}
	paths := []string{}
	for _, file := range plan.Files {
		paths = append(paths, file.Path)
	}
	assertContractNames(t, "init owned files", []string{projectvalue.ManifestPath, projectvalue.RuntimePath, ".markitect/model/manager.yaml", ".markitect/.gitignore", "docs/markitect/project.md"}, paths)
}

func assertConfigPreview(t *testing.T, data json.RawMessage) {
	t.Helper()
	preview := decodeOutput[projectsetup.Preview](t, data)
	if preview.Discovery.Provider != "process" || preview.Discovery.ProviderBinary.Path == "" || preview.Discovery.ProviderBinary.Digest == "" || preview.EditPlan.Digest == "" || len(preview.EditPlan.Mutation.Files) == 0 {
		t.Fatalf("config preview is incomplete: %+v", preview)
	}
	paths := []string{}
	for _, file := range preview.EditPlan.Mutation.Files {
		paths = append(paths, file.Path)
	}
	assertContractNames(t, "config owned files", []string{projectvalue.RuntimePath}, paths)
}

func assertOnboardPreview(t *testing.T, data json.RawMessage) {
	t.Helper()
	plan := decodeOutput[projectonboarding.Plan](t, data)
	if plan.Digest == "" || len(plan.Files) == 0 || len(plan.Options.Providers) != 1 || plan.Options.Providers[0] != projectonboarding.Codex {
		t.Fatalf("onboard preview is incomplete: %+v", plan)
	}
	paths := []string{}
	for _, file := range plan.Files {
		paths = append(paths, file.Path)
	}
	for _, path := range []string{"AGENTS.md", ".agents/skills/markitect-implement/SKILL.md", ".markitect/workflows/model-first.md"} {
		if !slices.Contains(paths, path) {
			t.Fatalf("onboard preview omitted required file %s: %v", path, paths)
		}
	}
}

func assertDoctorReport(t *testing.T, data json.RawMessage) {
	t.Helper()
	report := decodeOutput[projectsetup.DoctorReport](t, data)
	if report.Provider != "process" || report.Authentication != "not-verified" {
		t.Fatalf("doctor returned unexpected provider/authentication: %+v", report)
	}
	checks := map[string]string{}
	for _, check := range report.Checks {
		checks[check.Name] = check.Status
	}
	if checks["git-version"] != "available" || checks["process-executor"] != "available" {
		t.Fatalf("doctor did not report installed Git and the inspected executable: %+v", report.Checks)
	}
}

func assertAdoptPreview(t *testing.T, result projectapp.BrownfieldResult, sessionID, commit string) {
	t.Helper()
	if result.Status != "preview" || result.Action != "start" || result.Session == nil || result.Session.ID != sessionID || result.SessionDigest == "" || result.Session.Digest != result.SessionDigest || result.Session.Source.Commit != commit || result.Session.Target.Revision != commit || result.Session.Source.EvidenceCount != 1 {
		t.Fatalf("adopt start preview is incomplete or lost fixed identity: %+v", result)
	}
}

func assertNoAgents(t *testing.T, invoker *contractRejectingInvoker) {
	t.Helper()
	if invoker.calls != 0 {
		t.Fatalf("setup/adoption contract started %d agents", invoker.calls)
	}
}
