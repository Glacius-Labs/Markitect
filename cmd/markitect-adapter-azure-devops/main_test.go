package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const (
	resourceKey  = "engineering/software.markitect.org/v1alpha1/GitRepository/source"
	projectID    = "11111111-1111-4111-8111-111111111111"
	repositoryID = "22222222-2222-4222-8222-222222222222"
	organization = "example-org"
)

func TestObserveAndPlanUseCapturedRepositoryAndEmitNoOperations(t *testing.T) {
	withCapture(t, validCapture("refs/heads/main"), func(req request) {
		observed := run(req)
		if observed.Status != "complete" || len(observed.Findings) != 0 || observed.Observed == nil {
			t.Fatalf("observe = %#v", observed)
		}
		if observed.Observed.Repositories[0].DefaultBranch != "refs/heads/main" || observed.Observed.Evidence != evidenceKind {
			t.Fatalf("observed = %#v", observed.Observed)
		}
		req.Action = "plan"
		req.Observation = &observed
		planned := run(req)
		if planned.Status != "complete" || len(planned.Findings) != 0 || len(planned.Operations) != 0 {
			t.Fatalf("plan = %#v", planned)
		}
	})
}

func TestPlanReportsDefaultBranchMismatchAsFindingNotOperation(t *testing.T) {
	withCapture(t, validCapture("refs/heads/release"), func(req request) {
		req.Action = "plan"
		observationReq := req
		observationReq.Action = "observe"
		observed := run(observationReq)
		req.Observation = &observed
		planned := run(req)
		if planned.Status != "complete" || !hasFinding(planned, "default-branch-drift") || len(planned.Operations) != 0 {
			t.Fatalf("plan = %#v", planned)
		}
	})
}

func TestCaptureIdentityAndAPIErrorsAreIncomplete(t *testing.T) {
	tests := []struct {
		name string
		body string
		code string
	}{
		{"wrong response repository", stringsReplace(validCapture("refs/heads/main"), `"id":"`+repositoryID+`"`, `"id":"33333333-3333-4333-8333-333333333333"`), "capture-invalid"},
		{"wrong response project", stringsReplace(validCapture("refs/heads/main"), `"project":{"id":"`+projectID+`"}`, `"project":{"id":"33333333-3333-4333-8333-333333333333"}`), "capture-invalid"},
		{"wrong request project", stringsReplace(validCapture("refs/heads/main"), "/"+projectID+"/_apis", "/33333333-3333-4333-8333-333333333333/_apis"), "capture-invalid"},
		{"wrong API version", stringsReplace(validCapture("refs/heads/main"), "api-version=7.1", "api-version=7.0"), "capture-invalid"},
		{"unauthorized", stringsReplace(validCapture("refs/heads/main"), `"status":200`, `"status":401`), "capture-invalid"},
		{"malformed JSON", `{not-json`, "capture-invalid"},
		{"missing branch", stringsReplace(validCapture("refs/heads/main"), `,"defaultBranch":"refs/heads/main"`, ""), "capture-invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withCapture(t, test.body, func(req request) {
				got := run(req)
				if got.Status != "incomplete" || !hasFinding(got, test.code) {
					t.Fatalf("result = %#v", got)
				}
			})
		})
	}
}

func TestWrongTargetAndAmbiguousMappingsFailClosed(t *testing.T) {
	withCapture(t, validCapture("refs/heads/main"), func(req request) {
		req.Adapter.Target = "azure-devops://other-org/" + projectID
		if got := run(req); got.Status != "failed" || !hasFinding(got, "target-identity-mismatch") {
			t.Fatalf("wrong target = %#v", got)
		}
		req = validRequest("observe")
		params, err := decodeParameters(req.Adapter.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		params.Repositories = append(params.Repositories, params.Repositories[0])
		setParameters(t, &req, params)
		if got := run(req); got.Status != "incomplete" || !hasFinding(got, "repository-mapping-ambiguous") {
			t.Fatalf("duplicate mapping = %#v", got)
		}
	})
}

func TestVerifyFailsWhenCompletedPlanCaptureChanges(t *testing.T) {
	withCapture(t, validCapture("refs/heads/main"), func(req request) {
		observationReq := req
		observationReq.Action = "observe"
		observed := run(observationReq)
		planReq := req
		planReq.Action = "plan"
		planReq.Observation = &observed
		plan := run(planReq)
		if plan.Status != "complete" {
			t.Fatalf("plan = %#v", plan)
		}
		if err := os.WriteFile("repo.json", []byte(validCapture("refs/heads/other")), 0o600); err != nil {
			t.Fatal(err)
		}
		verifyReq := req
		verifyReq.Action = "verify"
		verifyReq.Plan = &plan
		verified := run(verifyReq)
		if verified.Status != "failed" || !hasFinding(verified, "verification-drift") {
			t.Fatalf("verify = %#v", verified)
		}
	})
}

func TestVerifyFailsWhenCompletedCaptureShowsCanonicalDrift(t *testing.T) {
	withCapture(t, validCapture("refs/heads/release"), func(req request) {
		observationReq := req
		observationReq.Action = "observe"
		observed := run(observationReq)
		planReq := req
		planReq.Action = "plan"
		planReq.Observation = &observed
		plan := run(planReq)
		if plan.Status != "complete" || !hasFinding(plan, "default-branch-drift") {
			t.Fatalf("plan = %#v", plan)
		}
		verifyReq := req
		verifyReq.Action = "verify"
		verifyReq.Plan = &plan
		verified := run(verifyReq)
		if verified.Status != "failed" || !hasFinding(verified, "default-branch-drift") {
			t.Fatalf("verify = %#v", verified)
		}
	})
}

func TestCapturePathCannotEscapeStagedInputs(t *testing.T) {
	if _, err := safeCapturePath(filepath.Join("..", "outside.json")); err == nil {
		t.Fatal("path escape accepted")
	}
}

func withCapture(t *testing.T, content string, check func(request)) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.WriteFile(filepath.Join(root, "repo.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	check(validRequest("observe"))
}
func validRequest(action string) request {
	params := adapterParameters{APIVersion: azureAPIVersion, Organization: organization, ProjectID: projectID, Repositories: []repositoryMapping{{Resource: resourceKey, RepositoryID: repositoryID, CaptureFile: "repo.json"}}}
	encoded, _ := yaml.Marshal(params)
	var values map[string]any
	_ = yaml.Unmarshal(encoded, &values)
	return request{APIVersion: requestAPIVersion, Action: action, Adapter: adapterRequest{Name: "azure-git-metadata", Type: "command", Version: "v0.1.0", Target: targetID(organization, projectID), Parameters: values}, Model: semanticModel{APIVersion: semanticModelAPI, ModelDigest: "model-digest", ValidationStatus: "passed", Resources: []modelResource{{Identity: modelIdentity{Key: resourceKey}, Data: map[string]any{"defaultBranch": "refs/heads/main"}}}}}
}
func setParameters(t *testing.T, req *request, params adapterParameters) {
	t.Helper()
	encoded, err := yaml.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if err := yaml.Unmarshal(encoded, &values); err != nil {
		t.Fatal(err)
	}
	req.Adapter.Parameters = values
}
func validCapture(branch string) string {
	return `{"request":{"method":"GET","url":"https://dev.azure.com/` + organization + `/` + projectID + `/_apis/git/repositories/` + repositoryID + `?api-version=7.1"},"response":{"status":200,"body":{"id":"` + repositoryID + `","name":"source","project":{"id":"` + projectID + `"},"defaultBranch":"` + branch + `"}}}`
}
func hasFinding(got result, code string) bool {
	for _, f := range got.Findings {
		if f.Code == code {
			return true
		}
	}
	return false
}
func stringsReplace(s, old, new string) string { return strings.Replace(s, old, new, 1) }
