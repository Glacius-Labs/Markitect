package github

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

const repositoryKey = "engineering/github.example.org/v1alpha1/Repository/markitect"
const captureFile = "evidence/github/markitect.json"

func TestObserveComparesSuppliedCaptureToCanonicalResource(t *testing.T) {
	req := validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
	first, second := Run(req), Run(req)
	if first.Status != "complete" || len(first.Findings) != 0 || len(first.Operations) != 0 {
		t.Fatalf("observe = %#v", first)
	}
	if first.Observed.Evidence != "github-rest-repository-metadata-capture" || first.Observed.GithubAPIVersion != githubAPIVersion || len(first.Observed.Scope) != 1 {
		t.Fatalf("observation = %#v", first.Observed)
	}
	var a, b bytes.Buffer
	_ = yaml.NewEncoder(&a).Encode(first)
	_ = yaml.NewEncoder(&b).Encode(second)
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("fixed model and byte input was nondeterministic")
	}
}

func TestPlanAndVerifyRetainReadOnlyDrift(t *testing.T) {
	req := validRequest("observe", 200, "Glacius-Labs/Markitect", "trunk")
	observed := Run(req)
	if observed.Status != "complete" || !hasFinding(observed, "default-branch-drift") {
		t.Fatalf("observe drift = %#v", observed)
	}
	planReq := req
	planReq.Action, planReq.Observation = "plan", &observed
	planned := Run(planReq)
	if planned.Status != "complete" || len(planned.Operations) != 0 || !hasFinding(planned, "default-branch-drift") {
		t.Fatalf("plan = %#v", planned)
	}
	verifyReq := req
	verifyReq.Action, verifyReq.Observation, verifyReq.Plan = "verify", &observed, &planned
	verified := Run(verifyReq)
	if verified.Status != "failed" || len(verified.Operations) != 0 || !hasFinding(verified, "default-branch-drift") {
		t.Fatalf("verify = %#v", verified)
	}
}

func TestPlanAndVerifyDetectChangedFixedCapture(t *testing.T) {
	req := validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
	observed := Run(req)
	planReq := req
	planReq.Action, planReq.Observation = "plan", &observed
	planned := Run(planReq)
	verifyReq := req
	verifyReq.Action, verifyReq.Observation, verifyReq.Plan = "verify", &observed, &planned
	verifyReq.Captures[captureFile] = capture(200, "Glacius-Labs/Markitect", "develop")
	verified := Run(verifyReq)
	if verified.Status != "failed" || !hasFinding(verified, "verification-drift") {
		t.Fatalf("changed capture verify = %#v", verified)
	}
}

func TestDesiredValueComesFromCoreIR(t *testing.T) {
	req := validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
	resource := req.Model.Resources[0]
	resource.Data["defaultBranch"] = "trunk"
	req.Model.Resources[0] = resource
	req.Model.ModelDigest = "sha256:changed"
	got := Run(req)
	if got.Status != "complete" || len(got.Findings) != 1 || got.Findings[0].Code != "default-branch-drift" {
		t.Fatalf("changed desired value = %#v", got)
	}
}

func TestCaptureFailuresAreIncomplete(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		status                           int
		fullName, bodyName, branch, code string
	}{
		{"unauthorized", 403, "Glacius-Labs/Markitect", "Glacius-Labs/Markitect", "main", "capture-unauthorized"},
		{"missing", 404, "Glacius-Labs/Markitect", "Glacius-Labs/Markitect", "main", "capture-repository-missing"},
		{"wrong request target", 200, "Glacius-Labs/Other", "Glacius-Labs/Other", "main", "capture-target-mismatch"},
		{"wrong response identity", 200, "Glacius-Labs/Markitect", "Glacius-Labs/Other", "main", "capture-target-mismatch"},
		{"empty branch", 200, "Glacius-Labs/Markitect", "Glacius-Labs/Markitect", "", "capture-field-missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validRequest("observe", tc.status, tc.fullName, tc.branch)
			if tc.bodyName != tc.fullName {
				req.Captures[captureFile] = captureWithBodyName(tc.status, tc.fullName, tc.bodyName, tc.branch)
			}
			got := Run(req)
			if got.Status != "incomplete" || !hasFinding(got, tc.code) {
				t.Fatalf("capture result = %#v", got)
			}
		})
	}
	req := validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
	delete(req.Captures, captureFile)
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "capture-invalid") {
		t.Fatalf("missing supplied bytes = %#v", got)
	}
}

func TestMalformedDuplicateAndOversizedCapturesAreRejected(t *testing.T) {
	for _, body := range []string{
		`{not-json`,
		`{"apiVersion":"` + captureAPIVersion + `","apiVersion":"other","request":{"method":"GET","path":"/repos/Glacius-Labs/Markitect","apiVersion":"2026-03-10"},"response":{"statusCode":200,"body":{}}}`,
		`{"apiVersion":"` + captureAPIVersion + `","request":{"method":"GET","path":"/repos/Glacius-Labs/Markitect","apiVersion":"2026-03-10"},"response":{"statusCode":200,"body":{"full_name":"Glacius-Labs/Markitect","default_branch":"main","default_branch":"trunk"}}}`,
	} {
		req := validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
		req.Captures[captureFile] = []byte(body)
		if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "capture-invalid") {
			t.Fatalf("malformed capture = %#v", got)
		}
	}
	req := validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
	req.Captures[captureFile] = bytes.Repeat([]byte(" "), maxCaptureBytes+1)
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "capture-invalid") {
		t.Fatalf("oversized supplied capture = %#v", got)
	}
}

func TestMappingsAndIdentityFailClosed(t *testing.T) {
	req := validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
	req.Config.Repositories[0].EvidenceFile = "../outside.json"
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "capture-path-invalid") {
		t.Fatalf("path traversal = %#v", got)
	}
	req = validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
	req.Config.Repositories = append(req.Config.Repositories, req.Config.Repositories[0])
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "repository-mapping-ambiguous") {
		t.Fatalf("duplicate mapping = %#v", got)
	}
	req = validRequest("apply", 200, "Glacius-Labs/Markitect", "main")
	if got := Run(req); got.Status != "failed" || !hasFinding(got, "unsupported-action") {
		t.Fatalf("apply = %#v", got)
	}
	req = validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
	req.Model.ValidationStatus = "failed"
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "model-not-validated") {
		t.Fatalf("invalid model = %#v", got)
	}
}

func TestCapturePathsAreOnlyExplicitSafeMappings(t *testing.T) {
	config := Config{Repositories: []repositoryMapping{{EvidenceFile: "b.json"}, {EvidenceFile: "../secret.json"}, {EvidenceFile: "a.json"}, {EvidenceFile: "a.json"}}}
	if got := CapturePaths(config); fmt.Sprint(got) != "[a.json b.json]" {
		t.Fatalf("paths = %v", got)
	}
}

func validRequest(action string, status int, requestedName, branch string) Request {
	return Request{
		APIVersion:    requestAPIVersion,
		Action:        action,
		Adapter:       Identity{Name: "github-repository-metadata", Type: "command", Version: "v0.1.0"},
		Model:         core.SemanticModel{APIVersion: semanticModelAPI, ModelDigest: "sha256:model", ValidationStatus: "passed", Resources: []core.ModelResource{{Identity: core.ModelIdentity{Key: repositoryKey, Kind: "Repository"}, Data: map[string]any{"fullName": "Glacius-Labs/Markitect", "defaultBranch": "main"}}}},
		Config:        Config{Repositories: []repositoryMapping{{Resource: repositoryKey, EvidenceFile: captureFile}}},
		Captures:      map[string][]byte{captureFile: captureWithBodyName(status, requestedName, "Glacius-Labs/Markitect", branch)},
		CaptureErrors: map[string]string{},
	}
}
func capture(status int, fullName, branch string) []byte {
	return captureWithBodyName(status, fullName, fullName, branch)
}
func captureWithBodyName(status int, requestedName, responseName, branch string) []byte {
	body := fmt.Sprintf(`{"full_name":%q,"default_branch":%q}`, responseName, branch)
	return []byte(fmt.Sprintf(`{"apiVersion":%q,"request":{"method":"GET","path":%q,"apiVersion":%q},"response":{"statusCode":%d,"body":%s}}`, captureAPIVersion, "/repos/"+requestedName, githubAPIVersion, status, body))
}
func hasFinding(result Result, code string) bool {
	for _, item := range result.Findings {
		if item.Code == code {
			return true
		}
	}
	return false
}

func TestCaptureVersionAndRequestShapeAreRequired(t *testing.T) {
	req := validRequest("observe", 200, "Glacius-Labs/Markitect", "main")
	req.Captures[captureFile] = []byte(strings.Replace(string(req.Captures[captureFile]), captureAPIVersion, "wrong/version", 1))
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "capture-version-unsupported") {
		t.Fatalf("version result = %#v", got)
	}
}
