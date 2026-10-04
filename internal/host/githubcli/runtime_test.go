package githubcli

import (
	"bytes"
	"os"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"go.yaml.in/yaml/v3"
)

func TestRunReadsOnlyMappedCaptureAndEmitsApplicationResult(t *testing.T) {
	withTempDirectory(t)
	if err := os.MkdirAll("evidence", 0700); err != nil {
		t.Fatal(err)
	}
	const key = "engineering/github.example.org/v1alpha1/Repository/markitect"
	data := []byte(`{"apiVersion":"markitect.github.example.org/repository-capture/v1alpha1","request":{"method":"GET","path":"/repos/Glacius-Labs/Markitect","apiVersion":"2026-03-10"},"response":{"statusCode":200,"body":{"full_name":"Glacius-Labs/Markitect","default_branch":"main"}}}`)
	if err := os.WriteFile("evidence/repository.json", data, 0600); err != nil {
		t.Fatal(err)
	}
	request := host.AdapterRequest{APIVersion: host.AdapterRequestVersion, Action: "observe", Adapter: host.AdapterIdentity{Name: "github-repository-metadata", Type: "command", Version: "v0.1.0", Parameters: map[string]any{"repositories": []any{map[string]any{"resource": key, "evidenceFile": "evidence/repository.json"}}}}, Model: core.SemanticModel{APIVersion: "markitect.example.org/semantic-model/v1alpha1", ModelDigest: "model", ValidationStatus: "passed", Resources: []core.ModelResource{{Identity: core.ModelIdentity{Key: key, Kind: "Repository"}, Data: map[string]any{"fullName": "Glacius-Labs/Markitect", "defaultBranch": "main"}}}}}
	input, err := host.YAML(request)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if code := Run(bytes.NewReader(input), &output, &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit code %d: %s", code, output.String())
	}
	var result host.AdapterResult
	if err := yaml.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" || result.Action != "observe" || len(result.Findings) != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestReadStagedFileRejectsTraversalDirectoriesAndOversize(t *testing.T) {
	withTempDirectory(t)
	if _, err := readStagedFile("../outside.json", 10); err == nil {
		t.Fatal("accepted traversal")
	}
	if err := os.Mkdir("directory", 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := readStagedFile("directory", 10); err == nil {
		t.Fatal("accepted directory as file")
	}
	if err := os.WriteFile("large.json", []byte("too long"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readStagedFile("large.json", 2); err == nil {
		t.Fatal("accepted oversized file")
	}
	if err := os.WriteFile("target.json", []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.json", "linked.json"); err == nil {
		if _, err := readStagedFile("linked.json", 100); err == nil {
			t.Fatal("accepted symlink")
		}
	}
}

func TestMalformedProtocolReturnsStructuredFailureAndExitTwo(t *testing.T) {
	var output bytes.Buffer
	if code := Run(bytes.NewBufferString("not: [yaml"), &output, &bytes.Buffer{}); code != 2 {
		t.Fatalf("exit code = %d", code)
	}
	var result host.AdapterResult
	if err := yaml.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "failed" || len(result.Findings) != 1 || result.Findings[0].Code != "invalid-request" {
		t.Fatalf("result = %#v", result)
	}
}

func withTempDirectory(t *testing.T) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Errorf("restore cwd: %v", err)
		}
	})
}
