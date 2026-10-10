package dotnetcli

import (
	"bytes"
	"os"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

func TestRunReadsMappedProjectInputAndEmitsApplicationResult(t *testing.T) {
	withTempDirectory(t)
	if err := os.MkdirAll("src", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("src/a.csproj", []byte(`<Project />`), 0600); err != nil {
		t.Fatal(err)
	}
	key := "engineering/example/v1/Module/a"
	request := host.AdapterRequest{APIVersion: host.AdapterRequestVersion, Action: "observe", Adapter: host.AdapterIdentity{Name: "dotnet-architecture", Type: "command", Version: "v0.1.0", Parameters: map[string]any{"projectMappings": []any{map[string]any{"resource": key, "projectFile": "src/a.csproj"}}}}, Model: core.SemanticModel{APIVersion: "markitect.example.org/semantic-model/v1alpha1", ModelDigest: "model", ValidationStatus: "passed", Resources: []core.ModelResource{{Identity: core.ModelIdentity{Key: key, Kind: "Module"}}}}}
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
	if _, err := readStagedFile("../outside.csproj", 10); err == nil {
		t.Fatal("accepted traversal")
	}
	if err := os.Mkdir("directory", 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := readStagedFile("directory", 10); err == nil {
		t.Fatal("accepted directory as file")
	}
	if err := os.WriteFile("large.csproj", []byte("too long"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readStagedFile("large.csproj", 2); err == nil {
		t.Fatal("accepted oversized file")
	}
	if err := os.WriteFile("target.csproj", []byte(`<Project />`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.csproj", "linked.csproj"); err == nil {
		if _, err := readStagedFile("linked.csproj", 100); err == nil {
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
