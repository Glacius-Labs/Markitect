package githubcli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
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
	request := app.AdapterRequest{APIVersion: app.AdapterRequestVersion, Action: "observe", Adapter: app.AdapterIdentity{Name: "github-repository-metadata", Type: "command", Version: "v0.1.0", Parameters: map[string]any{"repositories": []any{map[string]any{"resource": key, "evidenceFile": "evidence/repository.json"}}}}, Model: core.SemanticModel{APIVersion: "markitect.example.org/semantic-model/v1alpha1", ModelDigest: "model", ValidationStatus: "passed", Resources: []core.ModelResource{{Identity: core.ModelIdentity{Key: key, Kind: "Repository"}, Data: map[string]any{"fullName": "Glacius-Labs/Markitect", "defaultBranch": "main"}}}}}
	input, err := app.YAML(request)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if code := Run(bytes.NewReader(input), &output, &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit code %d: %s", code, output.String())
	}
	var result app.AdapterResult
	if err := yaml.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "complete" || result.Action != "observe" || len(result.Findings) != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestThinCommandConsumesFrozenApplicationRequestFromParsedSnapshot(t *testing.T) {
	model := parsedRepositoryModel(t)
	resource := findModelResource(t, model, "Repository")
	if resource.Source.Path != "resources/repository.yaml" || resource.Source.Digest == "" {
		t.Fatalf("compiled source identity = %#v", resource.Source)
	}
	stage := t.TempDir()
	if err := os.MkdirAll(filepath.Join(stage, "evidence"), 0700); err != nil {
		t.Fatal(err)
	}
	capture := []byte(`{"apiVersion":"markitect.github.example.org/repository-capture/v1alpha1","request":{"method":"GET","path":"/repos/Glacius-Labs/Markitect","apiVersion":"2026-03-10"},"response":{"statusCode":200,"body":{"full_name":"Glacius-Labs/Markitect","default_branch":"main"}}}`)
	if err := os.WriteFile(filepath.Join(stage, "evidence", "repository.json"), capture, 0600); err != nil {
		t.Fatal(err)
	}
	request := app.AdapterRequest{APIVersion: app.AdapterRequestVersion, Action: "observe", Adapter: app.AdapterIdentity{Name: "github-repository-metadata", Type: "command", Version: "v0.1.0", Parameters: map[string]any{"repositories": []any{map[string]any{"resource": resource.Identity.Key, "evidenceFile": "evidence/repository.json"}}}}, Model: model}
	input, err := app.YAML(request)
	if err != nil {
		t.Fatal(err)
	}
	root := repositoryRoot(t)
	binary := buildAdapter(t, root, "./cmd/markitect-adapter-github")
	command := exec.Command(binary)
	command.Dir, command.Stdin = stage, bytes.NewReader(input)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("adapter process: %v: %s", err, output)
	}
	var result app.AdapterResult
	if err := yaml.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode frozen result: %v\n%s", err, output)
	}
	if result.Status != "complete" || result.ModelDigest != model.ModelDigest || result.Observed["evidence"] != "github-rest-repository-metadata-capture" {
		t.Fatalf("result did not bind parsed model and fixed capture: %#v", result)
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
	var result app.AdapterResult
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

func parsedRepositoryModel(t *testing.T) core.SemanticModel {
	t.Helper()
	files := map[string][]byte{
		"markitect.yaml":            []byte("apiVersion: markitect.example.org/v1alpha1\nkind: Project\nmetadata:\n  name: adapter-host-test\nspec:\n  targets: [markdown]\n  areas:\n    - name: engineering\n      path: resources\n  domains:\n    - domains/github.yaml\n"),
		"domains/github.yaml":       []byte("apiVersion: markitect.example.org/v1alpha1\nkind: Domain\nmetadata:\n  name: github\nspec:\n  apiVersion: github.example.org/v1alpha1\n  kinds:\n    Repository:\n      required: [fullName, defaultBranch]\n      properties:\n        fullName:\n          type: string\n        defaultBranch:\n          type: string\n"),
		"resources/repository.yaml": []byte("apiVersion: github.example.org/v1alpha1\nkind: Repository\nmetadata:\n  name: markitect\n  namespace: engineering\nspec:\n  fullName: Glacius-Labs/Markitect\n  defaultBranch: main\n"),
	}
	p, err := app.Parse(&snapshot.Snapshot{ID: "fixed-github-adapter-snapshot", Provisional: true, Files: files, Modes: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	model, err := app.CompileModel(p)
	if err != nil {
		t.Fatal(err)
	}
	if model.ValidationStatus != "passed" {
		t.Fatalf("compiled model = %#v", model)
	}
	return model
}

func findModelResource(t *testing.T, model core.SemanticModel, kind string) core.ModelResource {
	t.Helper()
	for _, resource := range model.Resources {
		if resource.Identity.Kind == kind {
			return resource
		}
	}
	t.Fatalf("model has no %s resource", kind)
	return core.ModelResource{}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func buildAdapter(t *testing.T, root, packagePath string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "adapter")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, packagePath)
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build adapter: %v\n%s", err, output)
	}
	return binary
}
