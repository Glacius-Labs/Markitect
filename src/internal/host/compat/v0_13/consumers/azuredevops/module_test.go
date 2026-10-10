package azuredevops

import (
	"strings"
	"testing"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

const (
	resourceKey  = "engineering/software.markitect.org/v1alpha1/GitRepository/source"
	projectID    = "11111111-1111-4111-8111-111111111111"
	repositoryID = "22222222-2222-4222-8222-222222222222"
	organization = "example-org"
	captureFile  = "repo.json"
)

func TestObserveAndPlanUseOnlySuppliedCaptureBytes(t *testing.T) {
	req := validRequest("observe", `refs/heads/main`)
	observed := Run(req)
	if observed.Status != "complete" || observed.Observed == nil || len(observed.Findings) != 0 {
		t.Fatalf("observe = %#v", observed)
	}
	req.Action, req.Observation = "plan", &observed
	planned := Run(req)
	if planned.Status != "complete" || len(planned.Operations) != 0 || len(planned.Findings) != 0 {
		t.Fatalf("plan = %#v", planned)
	}
}

func TestDriftAndCaptureIdentityAreExplicit(t *testing.T) {
	req := validRequest("observe", `refs/heads/release`)
	observed := Run(req)
	req.Action, req.Observation = "plan", &observed
	planned := Run(req)
	if planned.Status != "complete" || len(planned.Operations) != 0 || !hasFinding(planned, "default-branch-drift") {
		t.Fatalf("drift plan = %#v", planned)
	}
	req = validRequest("observe", `refs/heads/main`)
	req.Captures[captureFile] = []byte(strings.Replace(string(req.Captures[captureFile]), repositoryID, "33333333-3333-4333-8333-333333333333", 1))
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "capture-invalid") {
		t.Fatalf("mismatched captured target = %#v", got)
	}
}

func TestMappingsAndActionsFailClosed(t *testing.T) {
	req := validRequest("apply", `refs/heads/main`)
	if got := Run(req); got.Status != "failed" {
		t.Fatalf("apply = %#v", got)
	}
	req = validRequest("observe", `refs/heads/main`)
	req.Config.Repositories[0].CaptureFile = "../outside.json"
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "capture-file-invalid") {
		t.Fatalf("escaping capture path = %#v", got)
	}
	req = validRequest("observe", `refs/heads/main`)
	delete(req.Captures, captureFile)
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "capture-file-invalid") {
		t.Fatalf("missing bytes = %#v", got)
	}
	req = validRequest("observe", `refs/heads/main`)
	req.Adapter.Target = "azure-devops://other-org/" + projectID
	if got := Run(req); got.Status != "failed" || !hasFinding(got, "target-identity-mismatch") {
		t.Fatalf("wrong target = %#v", got)
	}
	req = validRequest("observe", `refs/heads/main`)
	req.Config.Repositories = append(req.Config.Repositories, req.Config.Repositories[0])
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "repository-mapping-ambiguous") {
		t.Fatalf("duplicate mapping = %#v", got)
	}
	req = validRequest("observe", `refs/heads/main`)
	req.Adapter.Type = "other"
	if got := Run(req); got.Status != "failed" || !hasFinding(got, "adapter-identity-mismatch") {
		t.Fatalf("wrong adapter type = %#v", got)
	}
	req = validRequest("observe", `refs/heads/main`)
	req.APIVersion = "unsupported/version"
	if got := Run(req); got.Status != "failed" || !hasFinding(got, "unsupported-request-version") {
		t.Fatalf("wrong request version = %#v", got)
	}
	if _, err := DecodeConfig(map[string]any{"unknownParameter": true}); err == nil {
		t.Fatal("unknown adapter parameter was accepted")
	}
}

func TestAzureResponseExtensionsAndVerifyDriftStayReadOnly(t *testing.T) {
	req := validRequest("observe", `refs/heads/main`)
	req.Captures[captureFile] = []byte(strings.Replace(string(req.Captures[captureFile]), `"name":"source"`, `"name":"source","extra":{"unknown":true}`, 1))
	observed := Run(req)
	if observed.Status != "complete" {
		t.Fatalf("unknown response field = %#v", observed)
	}
	req.Action, req.Observation = "plan", &observed
	planned := Run(req)
	if planned.Status != "complete" {
		t.Fatalf("baseline plan = %#v", planned)
	}
	verify := req
	verify.Action, verify.Plan = "verify", &planned
	verify.Observation = nil
	verify.Captures[captureFile] = []byte(strings.Replace(string(verify.Captures[captureFile]), "refs/heads/main", "refs/heads/other", 1))
	if got := Run(verify); got.Status != "failed" || !hasFinding(got, "verification-drift") {
		t.Fatalf("changed capture verify = %#v", got)
	}

	drift := validRequest("observe", `refs/heads/release`)
	driftObservation := Run(drift)
	drift.Action, drift.Observation = "plan", &driftObservation
	driftPlan := Run(drift)
	if driftPlan.Status != "complete" || !hasFinding(driftPlan, "default-branch-drift") || len(driftPlan.Operations) != 0 {
		t.Fatalf("drift plan = %#v", driftPlan)
	}
	drift.Action, drift.Plan, drift.Observation = "verify", &driftPlan, nil
	verified := Run(drift)
	if verified.Status != "failed" || !hasFinding(verified, "default-branch-drift") || len(verified.Operations) != 0 {
		t.Fatalf("canonical drift verify = %#v", verified)
	}
}

func TestCapturePathsReturnOnlyExplicitSafePaths(t *testing.T) {
	cfg := Config{Repositories: []repositoryMapping{{CaptureFile: "b.json"}, {CaptureFile: "../outside"}, {CaptureFile: "a.json"}, {CaptureFile: "a.json"}}}
	got := CapturePaths(cfg)
	if len(got) != 2 || got[0] != "a.json" || got[1] != "b.json" {
		t.Fatalf("capture paths = %v", got)
	}
}

func validRequest(action, observedBranch string) Request {
	capture := `{"request":{"method":"GET","url":"https://dev.azure.com/` + organization + `/` + projectID + `/_apis/git/repositories/` + repositoryID + `?api-version=7.1"},"response":{"status":200,"body":{"id":"` + repositoryID + `","name":"source","project":{"id":"` + projectID + `"},"defaultBranch":"` + observedBranch + `"}}}`
	return Request{
		APIVersion: requestAPIVersion, Action: action,
		Adapter:  Identity{Name: "azure-git-metadata", Type: "command", Version: "v0.1.0", Target: targetID(organization, projectID)},
		Model:    core.SemanticModel{APIVersion: semanticModelAPI, ModelDigest: "model-digest", ValidationStatus: "passed", Resources: []core.ModelResource{{Identity: core.ModelIdentity{Kind: "GitRepository", Key: resourceKey}, Data: map[string]any{"defaultBranch": "refs/heads/main"}}}},
		Config:   Config{APIVersion: azureAPIVersion, Organization: organization, ProjectID: projectID, Repositories: []repositoryMapping{{Resource: resourceKey, RepositoryID: repositoryID, CaptureFile: captureFile}}},
		Captures: map[string][]byte{captureFile: []byte(capture)}, CaptureErrors: map[string]string{},
	}
}

func hasFinding(result Result, code string) bool {
	for _, finding := range result.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
