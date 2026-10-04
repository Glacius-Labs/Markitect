package azuredevops

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
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
