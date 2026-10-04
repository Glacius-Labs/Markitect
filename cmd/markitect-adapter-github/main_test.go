package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

const (
	adapterName   = "github-repository-metadata"
	repositoryKey = "engineering/github.example.org/v1alpha1/Repository/markitect"
	captureFile   = "evidence/github/markitect.json"
)

func TestObserveComparesCapturedRepositoryToCanonicalResource(t *testing.T) {
	root := fixture(t, 200, "Glacius-Labs/Markitect", "main")
	withWorkingDirectory(t, root)
	req := validRequest("observe")
	first := run(req)
	second := run(req)
	if first.Status != "complete" || len(first.Findings) != 0 {
		t.Fatalf("status/findings = %s/%v", first.Status, first.Findings)
	}
	if len(first.Operations) != 0 || first.Observed.Evidence != "github-rest-repository-metadata-capture" || first.Observed.GithubAPIVersion != githubAPIVersion {
		t.Fatalf("unexpected observation envelope: %+v", first)
	}
	if !reflect.DeepEqual(first.Observed.Scope, []string{repositoryKey}) {
		t.Fatalf("scope = %v", first.Observed.Scope)
	}
	got, ok := first.Observed.Repositories[repositoryKey]
	if !ok || got.FullName != "Glacius-Labs/Markitect" || got.DefaultBranch != "main" || got.RequestPath != "/repos/Glacius-Labs/Markitect" || !strings.HasPrefix(got.CaptureDigest, "sha256:") {
		t.Fatalf("repository evidence = %+v", got)
	}
	one, _ := yaml.Marshal(first)
	two, _ := yaml.Marshal(second)
	if !bytes.Equal(one, two) {
		t.Fatal("the same fixed model and capture did not produce deterministic output")
	}
}

func TestDefaultBranchDriftProducesReadOnlyPlanAndFailedVerify(t *testing.T) {
	root := fixture(t, 200, "Glacius-Labs/Markitect", "trunk")
	withWorkingDirectory(t, root)
	req := validRequest("observe")
	observed := run(req)
	if observed.Status != "complete" || !hasFinding(observed, "default-branch-drift") {
		t.Fatalf("observation should be complete and visibly report drift: %+v", observed)
	}
	planRequest := req
	planRequest.Action = "plan"
	planRequest.Observation = &observed
	planned := run(planRequest)
	if planned.Status != "complete" || len(planned.Operations) != 0 || !hasFinding(planned, "default-branch-drift") {
		t.Fatalf("plan must report drift without proposing operations: %+v", planned)
	}
	verifyRequest := req
	verifyRequest.Action = "verify"
	verifyRequest.Observation = &observed
	verifyRequest.Plan = &planned
	verified := run(verifyRequest)
	if verified.Status != "failed" || !hasFinding(verified, "default-branch-drift") || len(verified.Operations) != 0 {
		t.Fatalf("verify must fail for completed drift and remain read-only: %+v", verified)
	}
}

func TestPlanAndVerifyDetectChangedFixedCapture(t *testing.T) {
	root := fixture(t, 200, "Glacius-Labs/Markitect", "main")
	withWorkingDirectory(t, root)
	req := validRequest("observe")
	observed := run(req)
	planReq := req
	planReq.Action = "plan"
	planReq.Observation = &observed
	planned := run(planReq)
	if planned.Status != "complete" {
		t.Fatalf("baseline plan status = %s: %+v", planned.Status, planned.Findings)
	}
	writeCapture(t, root, 200, "Glacius-Labs/Markitect", "develop")
	verifyReq := req
	verifyReq.Action = "verify"
	verifyReq.Observation = &observed
	verifyReq.Plan = &planned
	verified := run(verifyReq)
	if verified.Status != "failed" || !hasFinding(verified, "verification-drift") {
		t.Fatalf("changed captured bytes must invalidate verification: %+v", verified)
	}
}

func TestDesiredValuesComeOnlyFromNormalizedResource(t *testing.T) {
	root := fixture(t, 200, "Glacius-Labs/Markitect", "main")
	withWorkingDirectory(t, root)
	req := validRequest("observe")
	resource := req.Model.Resources[0]
	resource.Data["defaultBranch"] = "trunk"
	req.Model.Resources[0] = resource
	req.Model.ModelDigest = "sha256:changed-canonical-model"
	got := run(req)
	if got.Status != "complete" || len(got.Findings) != 1 || got.Findings[0].Code != "default-branch-drift" {
		t.Fatalf("changed normalized desired value should change the comparison: %+v", got)
	}
}

func TestIncompleteCapturesCannotPass(t *testing.T) {
	for _, test := range []struct {
		name                               string
		status                             int
		fullName, branch, responseFullName string
		removeFile                         bool
		wantCode                           string
	}{
		{name: "unauthorized response", status: 403, wantCode: "capture-unauthorized"},
		{name: "missing repository response", status: 404, wantCode: "capture-repository-missing"},
		{name: "missing staged file", removeFile: true, wantCode: "capture-invalid"},
		{name: "request points elsewhere", status: 200, fullName: "Glacius-Labs/Other", branch: "main", wantCode: "capture-target-mismatch"},
		{name: "response identity points elsewhere", status: 200, fullName: "Glacius-Labs/Markitect", branch: "main", wantCode: "capture-target-mismatch", responseFullName: "Glacius-Labs/Other"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := fixture(t, 200, "Glacius-Labs/Markitect", "main")
			if test.status != 0 {
				writeCapture(t, root, test.status, "Glacius-Labs/Markitect", "main")
			}
			if test.fullName != "" {
				writeCaptureWithRequestAndResponse(t, root, test.fullName, test.responseFullName, test.branch)
			}
			if test.removeFile {
				if err := os.Remove(filepath.Join(root, filepath.FromSlash(captureFile))); err != nil {
					t.Fatal(err)
				}
			}
			withWorkingDirectory(t, root)
			got := run(validRequest("observe"))
			if got.Status != "incomplete" || !hasFinding(got, test.wantCode) {
				t.Fatalf("status/findings = %s/%v, want incomplete/%s", got.Status, got.Findings, test.wantCode)
			}
		})
	}
}

func TestMalformedCaptureAndCapturePathAreIncomplete(t *testing.T) {
	root := fixture(t, 200, "Glacius-Labs/Markitect", "main")
	withWorkingDirectory(t, root)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(captureFile)), []byte(`{"apiVersion":"wrong","unexpected":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got := run(validRequest("observe")); got.Status != "incomplete" || !hasFinding(got, "capture-invalid") {
		t.Fatalf("malformed/unknown capture fields should be incomplete: %+v", got)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(captureFile)), bytes.Repeat([]byte(" "), maxCaptureBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if got := run(validRequest("observe")); got.Status != "incomplete" || !hasFinding(got, "capture-invalid") {
		t.Fatalf("capture larger than the fixed 1 MiB bound should be incomplete: %+v", got)
	}

	req := validRequest("observe")
	parameters := req.Adapter.Parameters["repositories"].([]repositoryMapping)
	parameters[0].EvidenceFile = "../outside.json"
	req.Adapter.Parameters["repositories"] = parameters
	if got := run(req); got.Status != "incomplete" || !hasFinding(got, "capture-path-invalid") {
		t.Fatalf("path traversal should be rejected: %+v", got)
	}
}

func TestDuplicateCaptureJSONKeysAreIncomplete(t *testing.T) {
	root := fixture(t, 200, "Glacius-Labs/Markitect", "main")
	withWorkingDirectory(t, root)
	for _, raw := range []string{
		`{"apiVersion":"markitect.github.example.org/repository-capture/v1alpha1","apiVersion":"other","request":{"method":"GET","path":"/repos/Glacius-Labs/Markitect","apiVersion":"2026-03-10"},"response":{"statusCode":200,"body":{}}}`,
		`{"apiVersion":"markitect.github.example.org/repository-capture/v1alpha1","ApiVersion":"other","request":{"method":"GET","path":"/repos/Glacius-Labs/Markitect","apiVersion":"2026-03-10"},"response":{"statusCode":200,"body":{}}}`,
		`{"apiVersion":"markitect.github.example.org/repository-capture/v1alpha1","request":{"method":"GET","path":"/repos/Glacius-Labs/Markitect","apiVersion":"2026-03-10"},"response":{"statusCode":200,"body":{"full_name":"Glacius-Labs/Markitect","default_branch":"main","default_branch":"trunk"}}}`,
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(captureFile)), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		got := run(validRequest("observe"))
		if got.Status != "incomplete" || !hasFinding(got, "capture-invalid") {
			t.Fatalf("duplicate capture keys must not be accepted: %+v", got)
		}
	}
}

func TestMappingsFailClosedOnUnknownOrDuplicateEntries(t *testing.T) {
	root := fixture(t, 200, "Glacius-Labs/Markitect", "main")
	withWorkingDirectory(t, root)
	for _, test := range []struct {
		name   string
		mutate func(*request)
		want   string
	}{
		{name: "unknown canonical identity", mutate: func(r *request) {
			maps := mappings(r)
			maps[0].Resource = "engineering/Repository/other"
			setMappings(r, maps)
		}, want: "resource-mapping-invalid"},
		{name: "duplicate resource", mutate: func(r *request) { maps := mappings(r); maps = append(maps, maps[0]); setMappings(r, maps) }, want: "repository-mapping-ambiguous"},
		{name: "duplicate evidence input", mutate: func(r *request) {
			secondKey := "engineering/github.example.org/v1alpha1/Repository/other"
			r.Model.Resources = append(r.Model.Resources, modelResource{Identity: resourceIdentity{Kind: "Repository", Key: secondKey}, Data: map[string]any{"fullName": "Glacius-Labs/Other", "defaultBranch": "main"}})
			maps := mappings(r)
			maps = append(maps, repositoryMapping{Resource: secondKey, EvidenceFile: captureFile})
			setMappings(r, maps)
		}, want: "capture-mapping-ambiguous"},
		{name: "unknown parameter", mutate: func(r *request) { r.Adapter.Parameters["inferRepositories"] = true }, want: "adapter-parameters-invalid"},
		{name: "wrong resource kind", mutate: func(r *request) { r.Model.Resources[0].Identity.Kind = "Module" }, want: "resource-kind-unsupported"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := validRequest("observe")
			test.mutate(&req)
			got := run(req)
			if got.Status != "incomplete" && got.Status != "failed" || !hasFinding(got, test.want) {
				t.Fatalf("status/findings = %s/%v, want finding %s", got.Status, got.Findings, test.want)
			}
		})
	}
}

func TestAdapterRejectsApplyAndUnsupportedCaptureVersions(t *testing.T) {
	root := fixture(t, 200, "Glacius-Labs/Markitect", "main")
	withWorkingDirectory(t, root)
	apply := run(validRequest("apply"))
	if apply.Status != "failed" || !hasFinding(apply, "unsupported-action") {
		t.Fatalf("apply should be rejected: %+v", apply)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(captureFile)), captureBytes("other/v1", 200, "Glacius-Labs/Markitect", "main"), 0600); err != nil {
		t.Fatal(err)
	}
	unsupported := run(validRequest("observe"))
	if unsupported.Status != "incomplete" || !hasFinding(unsupported, "capture-version-unsupported") {
		t.Fatalf("unsupported capture version should be incomplete: %+v", unsupported)
	}
}

func TestBuiltExecutableConsumesFrozenCommandProtocol(t *testing.T) {
	root := fixture(t, 200, "Glacius-Labs/Markitect", "main")
	packageDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(t.TempDir(), "go-cache")
	t.Setenv("GOCACHE", cache)
	binary := filepath.Join(t.TempDir(), "markitect-adapter-github")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = packageDirectory
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build adapter independently: %v\n%s", err, output)
	}
	actual := fullAdapterRequest()
	actual.Model.Resources[0].Data["description"] = strings.Repeat("x", maxCaptureBytes+64)
	input, err := yaml.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(input) <= maxCaptureBytes || len(input) > maxRequestBytes {
		t.Fatalf("full DTO request size = %d; want between capture and request bounds", len(input))
	}
	command := exec.Command(binary)
	command.Dir = root
	command.Stdin = bytes.NewReader(input)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("invoke adapter: %v", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(output))
	var response result
	if err := decoder.Decode(&response); err != nil {
		t.Fatalf("decode adapter response: %v\n%s", err, output)
	}
	if response.APIVersion != resultAPIVersion || response.Adapter != adapterName || response.Action != "observe" || response.Status != "complete" || response.Observed.Repositories[repositoryKey].DefaultBranch != "main" {
		t.Fatalf("adapter protocol response = %+v", response)
	}

	actual.Model.Resources[0].Data["description"] = strings.Repeat("x", maxRequestBytes+1)
	oversized, err := yaml.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(oversized) <= maxRequestBytes {
		t.Fatalf("oversized test request has only %d bytes", len(oversized))
	}
	command = exec.Command(binary)
	command.Dir = root
	command.Stdin = bytes.NewReader(oversized)
	output, err = command.Output()
	if err == nil || !bytes.Contains(output, []byte("adapter request exceeds this prototype's 10 MiB input limit")) {
		t.Fatalf("request over the documented adapter bound should be rejected: err=%v output=%s", err, output)
	}
}

func fullAdapterRequest() host.AdapterRequest {
	base := validRequest("observe")
	model := host.SemanticModel{
		APIVersion:   semanticModelAPI,
		Snapshot:     host.ModelSnapshot{ID: "commit:abcdef", Digest: "sha256:fixed-snapshot"},
		ConfigDigest: "sha256:fixed-project-config", ModelDigest: "sha256:fixed-model",
		ValidationStatus: "passed", StructuralStatus: "passed", PolicyStatus: "passed",
		DomainInputs: []host.ModelDomainInput{{APIVersion: "github.example.org/v1alpha1", Name: "repositories", Path: "domains/repositories.yaml", Digest: "sha256:domain"}},
		Domains: []host.ModelDomain{{Name: "repositories", APIVersion: "github.example.org/v1alpha1", Kinds: map[string]core.KindDefinition{"Repository": {
			Required: []string{"fullName", "defaultBranch", "description"},
			Properties: map[string]core.PropertyDefinition{
				"fullName": {Type: "string"}, "defaultBranch": {Type: "string"}, "description": {Type: "string"},
			},
		}}}},
		Resources: []host.ModelResource{
			{Identity: host.ModelIdentity{APIVersion: "github.example.org/v1alpha1", Kind: "Repository", Namespace: "engineering", Name: "markitect", Key: repositoryKey}, Labels: map[string]string{"tier": "platform"}, Data: map[string]any{"fullName": "Glacius-Labs/Markitect", "defaultBranch": "main"}, Source: host.ModelSource{Path: "resources/markitect.repository.yaml", Line: 1, Digest: "sha256:resource"}, Area: "engineering"},
			{Identity: host.ModelIdentity{APIVersion: "github.example.org/v1alpha1", Kind: "Repository", Namespace: "engineering", Name: "parent", Key: "engineering/github.example.org/v1alpha1/Repository/parent"}, Data: map[string]any{"fullName": "Glacius-Labs/Parent", "defaultBranch": "main"}, Source: host.ModelSource{Path: "resources/parent.repository.yaml", Digest: "sha256:parent"}, Area: "engineering"},
		},
		Relationships: []host.ModelRelationship{{From: repositoryKey, To: "engineering/github.example.org/v1alpha1/Repository/parent", Type: "dependsOn", Source: host.ModelSource{Path: "resources/markitect.repository.yaml", Digest: "sha256:relation"}, Context: true, Invalidate: true, Acyclic: true}},
	}
	return host.AdapterRequest{
		APIVersion: requestAPIVersion,
		Action:     "observe",
		Adapter:    host.AdapterIdentity{Name: base.Adapter.Name, Type: base.Adapter.Type, Version: base.Adapter.Version, Parameters: base.Adapter.Parameters},
		Model:      model,
	}
}

func validRequest(action string) request {
	return request{
		APIVersion: requestAPIVersion, Action: action,
		Adapter: adapterRequest{Name: adapterName, Type: "command", Version: "markitect-github-repository-metadata/v0.1.0", Parameters: map[string]any{
			"repositories": []repositoryMapping{{Resource: repositoryKey, EvidenceFile: captureFile}},
		}},
		Model: semanticModel{APIVersion: semanticModelAPI, ModelDigest: "sha256:fixed-model", ValidationStatus: "passed", Resources: []modelResource{{
			Identity: resourceIdentity{Kind: "Repository", Key: repositoryKey},
			Data:     map[string]any{"fullName": "Glacius-Labs/Markitect", "defaultBranch": "main"},
		}}},
	}
}

func fixture(t *testing.T, status int, fullName, defaultBranch string) string {
	t.Helper()
	root := t.TempDir()
	writeCapture(t, root, status, fullName, defaultBranch)
	return root
}

func writeCapture(t *testing.T, root string, status int, fullName, defaultBranch string) {
	t.Helper()
	writeCaptureWithRequestAndResponse(t, root, fullName, fullName, defaultBranch)
	if status != 200 {
		capture := captureEnvelope{APIVersion: captureAPIVersion, Request: captureRequest{Method: "GET", Path: "/repos/Glacius-Labs/Markitect", APIVersion: githubAPIVersion}, Response: captureResponse{StatusCode: status, Body: json.RawMessage(`{"message":"captured error"}`)}}
		writeCaptureEnvelope(t, root, capture)
	}
}

func writeCaptureWithRequestAndResponse(t *testing.T, root, requestFullName, responseFullName, branch string) {
	t.Helper()
	owner, repository, found := strings.Cut(requestFullName, "/")
	if !found {
		t.Fatal("test fullName requires owner/repository")
	}
	capture := captureEnvelope{
		APIVersion: captureAPIVersion,
		Request:    captureRequest{Method: "GET", Path: "/repos/" + owner + "/" + repository, APIVersion: githubAPIVersion},
		Response:   captureResponse{StatusCode: 200, Body: json.RawMessage(`{"full_name":"` + responseFullName + `","default_branch":"` + branch + `","archived":false}`)},
	}
	writeCaptureEnvelope(t, root, capture)
}

func writeCaptureEnvelope(t *testing.T, root string, capture captureEnvelope) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(captureFile))
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(capture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func captureBytes(apiVersion string, status int, fullName, branch string) []byte {
	body, _ := json.Marshal(map[string]any{"full_name": fullName, "default_branch": branch})
	capture := captureEnvelope{APIVersion: apiVersion, Request: captureRequest{Method: "GET", Path: "/repos/Glacius-Labs/Markitect", APIVersion: githubAPIVersion}, Response: captureResponse{StatusCode: status, Body: body}}
	encoded, _ := json.Marshal(capture)
	return encoded
}

func withWorkingDirectory(t *testing.T, directory string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func mappings(req *request) []repositoryMapping {
	return req.Adapter.Parameters["repositories"].([]repositoryMapping)
}

func setMappings(req *request, entries []repositoryMapping) {
	req.Adapter.Parameters["repositories"] = entries
}

func hasFinding(result result, code string) bool {
	for _, finding := range result.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
