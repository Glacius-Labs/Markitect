package azuredevopscli

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

func TestRunReadsMappedCaptureAndEmitsApplicationResult(t *testing.T) {
	withTempDirectory(t)
	const key = "engineering/software.markitect.org/v1alpha1/GitRepository/source"
	const project = "11111111-1111-4111-8111-111111111111"
	const repository = "22222222-2222-4222-8222-222222222222"
	body := `{"request":{"method":"GET","url":"https://dev.azure.com/example-org/` + project + `/_apis/git/repositories/` + repository + `?api-version=7.1"},"response":{"status":200,"body":{"id":"` + repository + `","name":"source","project":{"id":"` + project + `"},"defaultBranch":"refs/heads/main"}}}`
	if err := os.WriteFile("repo.json", []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	request := app.AdapterRequest{APIVersion: app.AdapterRequestVersion, Action: "observe", Adapter: app.AdapterIdentity{Name: "azure-git-metadata", Type: "command", Version: "v0.1.0", Target: "azure-devops://example-org/" + project, Parameters: map[string]any{"apiVersion": "7.1", "organization": "example-org", "projectId": project, "repositories": []any{map[string]any{"resource": key, "repositoryId": repository, "captureFile": "repo.json"}}}}, Model: core.SemanticModel{APIVersion: "markitect.example.org/semantic-model/v1alpha1", ModelDigest: "model", ValidationStatus: "passed", Resources: []core.ModelResource{{Identity: core.ModelIdentity{Key: key, Kind: "GitRepository"}, Data: map[string]any{"defaultBranch": "refs/heads/main"}}}}}
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
	if err := os.WriteFile("repo.json", bytes.Repeat([]byte(" "), (1<<20)+1), 0600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if code := Run(bytes.NewReader(input), &output, &bytes.Buffer{}); code != 0 {
		t.Fatalf("oversized capture exit code %d: %s", code, output.String())
	}
	var oversized app.AdapterResult
	if err := yaml.Unmarshal(output.Bytes(), &oversized); err != nil {
		t.Fatal(err)
	}
	if oversized.Status != "incomplete" || !hasAdapterFinding(oversized, "capture-file-invalid") {
		t.Fatalf("oversized capture result = %#v", oversized)
	}
	if err := os.Remove("repo.json"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("repo.json", 0700); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if code := Run(bytes.NewReader(input), &output, &bytes.Buffer{}); code != 0 {
		t.Fatalf("non-regular capture exit code %d: %s", code, output.String())
	}
	var nonRegular app.AdapterResult
	if err := yaml.Unmarshal(output.Bytes(), &nonRegular); err != nil {
		t.Fatal(err)
	}
	if nonRegular.Status != "incomplete" || !hasAdapterFinding(nonRegular, "capture-file-invalid") {
		t.Fatalf("non-regular capture result = %#v", nonRegular)
	}
}

func TestThinCommandConsumesApplicationSPIFromParsedSourceSnapshot(t *testing.T) {
	model := parsedGitRepositoryModel(t)
	resource := findGitRepository(t, model)
	if resource.Source.Path != "resources/repository.yaml" || resource.Source.Digest == "" {
		t.Fatalf("compiled source identity = %#v", resource.Source)
	}
	stage := t.TempDir()
	projectID := "11111111-1111-4111-8111-111111111111"
	repositoryID := "22222222-2222-4222-8222-222222222222"
	capture := []byte(`{"request":{"method":"GET","url":"https://dev.azure.com/example-org/` + projectID + `/_apis/git/repositories/` + repositoryID + `?api-version=7.1"},"response":{"status":200,"body":{"id":"` + repositoryID + `","name":"source","project":{"id":"` + projectID + `"},"defaultBranch":"refs/heads/main"}}}`)
	if err := os.WriteFile(filepath.Join(stage, "repo.json"), capture, 0600); err != nil {
		t.Fatal(err)
	}
	request := app.AdapterRequest{APIVersion: app.AdapterRequestVersion, Action: "observe", Adapter: app.AdapterIdentity{Name: "azure-git-metadata", Type: "command", Version: "v0.1.0", Target: "azure-devops://example-org/" + projectID, Parameters: map[string]any{"apiVersion": "7.1", "organization": "example-org", "projectId": projectID, "repositories": []any{map[string]any{"resource": resource.Identity.Key, "repositoryId": repositoryID, "captureFile": "repo.json"}}}}, Model: model}
	input, err := app.YAML(request)
	if err != nil {
		t.Fatal(err)
	}
	binary := buildAdapter(t, repositoryRoot(t), "./cmd/markitect-adapter-azure-devops")
	command := exec.Command(binary)
	command.Dir, command.Stdin = stage, bytes.NewReader(input)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("adapter process: %v: %s", err, output)
	}
	var result app.AdapterResult
	if err := yaml.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode SPI result: %v\n%s", err, output)
	}
	if result.Status != "complete" || result.ModelDigest != model.ModelDigest || result.Observed["evidence"] != "captured-azure-devops-git-repository-default-branch" {
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

func parsedGitRepositoryModel(t *testing.T) core.SemanticModel {
	t.Helper()
	files := map[string][]byte{
		"markitect.yaml":            []byte("apiVersion: markitect.example.org/v1alpha1\nkind: Project\nmetadata:\n  name: adapter-host-test\nspec:\n  targets: [markdown]\n  areas:\n    - name: engineering\n      path: resources\n  domains:\n    - domains/azure.yaml\n"),
		"domains/azure.yaml":        []byte("apiVersion: markitect.example.org/v1alpha1\nkind: Domain\nmetadata:\n  name: azure\nspec:\n  apiVersion: azure.example.org/v1alpha1\n  kinds:\n    GitRepository:\n      required: [defaultBranch]\n      properties:\n        defaultBranch:\n          type: string\n"),
		"resources/repository.yaml": []byte("apiVersion: azure.example.org/v1alpha1\nkind: GitRepository\nmetadata:\n  name: source\n  namespace: engineering\nspec:\n  defaultBranch: refs/heads/main\n"),
	}
	p, err := app.Parse(&snapshot.Snapshot{ID: "fixed-azure-adapter-snapshot", Provisional: true, Files: files, Modes: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	model, err := app.CompileModel(p)
	if err != nil {
		t.Fatal(err)
	}
	if model.ValidationStatus != "passed" {
		t.Fatalf("compiled model validation = %s; diagnostics=%#v", model.ValidationStatus, model.Diagnostics)
	}
	return model
}

func findGitRepository(t *testing.T, model core.SemanticModel) core.ModelResource {
	t.Helper()
	for _, resource := range model.Resources {
		if resource.Identity.Kind == "GitRepository" {
			return resource
		}
	}
	t.Fatal("model has no GitRepository resource")
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

func hasAdapterFinding(result app.AdapterResult, code string) bool {
	for _, finding := range result.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
