package examples

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// This integration control sends the complete compiler-produced DTO through the
// unchanged command runner, rather than testing an adapter's local DTO subset.
func TestParallelWaveRepositoryAdaptersUseOneCanonicalModelIndependently(t *testing.T) {
	_, current, _, _ := runtime.Caller(0)
	sourceRoot := filepath.Dir(filepath.Dir(current))
	tools := t.TempDir()
	for _, name := range []string{"markitect-adapter-github", "markitect-adapter-azure-devops"} {
		executable := name
		if runtime.GOOS == "windows" {
			executable += ".exe"
		}
		command := exec.Command("go", "build", "-o", filepath.Join(tools, executable), "./cmd/"+name)
		command.Dir = sourceRoot
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", name, err, output)
		}
	}
	t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	files := map[string][]byte{
		"markitect.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata:
  name: parallel-adapter-control
spec:
  areas: [{name: engineering, path: resources}]
  domains: [domains/repositories.yaml]
  adapters:
    - name: github-capture
      type: command
      version: fixture-v1
      config:
        target: github://example/source
        inputs: [captures/github.json]
        observe: [markitect-adapter-github]
        plan: [markitect-adapter-github]
        verify: [markitect-adapter-github]
        parameters:
          repositories:
            - resource: engineering/pilot.example.org/v1alpha1/Repository/github
              evidenceFile: captures/github.json
    - name: azure-capture
      type: command
      version: fixture-v1
      config:
        target: azure-devops://example-org/11111111-1111-4111-8111-111111111111
        inputs: [captures/azure.json]
        observe: [markitect-adapter-azure-devops]
        plan: [markitect-adapter-azure-devops]
        verify: [markitect-adapter-azure-devops]
        parameters:
          apiVersion: "7.1"
          organization: example-org
          projectId: 11111111-1111-4111-8111-111111111111
          repositories:
            - resource: engineering/pilot.example.org/v1alpha1/Repository/azure
              repositoryId: 22222222-2222-4222-8222-222222222222
              captureFile: captures/azure.json
`),
		"domains/repositories.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata:
  name: repository-metadata-control
spec:
  apiVersion: pilot.example.org/v1alpha1
  kinds:
    Repository:
      required: [fullName, defaultBranch]
      properties:
        fullName: {type: string}
        defaultBranch: {type: string}
`),
		"resources/README.md": []byte("# Integration control\n"),
		"resources/github.yaml": []byte(`apiVersion: pilot.example.org/v1alpha1
kind: Repository
metadata:
  name: github
  namespace: engineering
spec:
  fullName: example/source
  defaultBranch: main
`),
		"resources/azure.yaml": []byte(`apiVersion: pilot.example.org/v1alpha1
kind: Repository
metadata:
  name: azure
  namespace: engineering
spec:
  fullName: example-org/source
  defaultBranch: refs/heads/main
`),
		"captures/github.json": []byte(`{"apiVersion":"markitect.github.example.org/repository-capture/v1alpha1","request":{"method":"GET","path":"/repos/example/source","apiVersion":"2026-03-10"},"response":{"statusCode":200,"body":{"full_name":"example/source","default_branch":"main"}}}`),
		"captures/azure.json":  []byte(`{"request":{"method":"GET","url":"https://dev.azure.com/example-org/11111111-1111-4111-8111-111111111111/_apis/git/repositories/22222222-2222-4222-8222-222222222222?api-version=7.1"},"response":{"status":200,"body":{"id":"22222222-2222-4222-8222-222222222222","project":{"id":"11111111-1111-4111-8111-111111111111"},"defaultBranch":"refs/heads/main"}}}`),
	}
	modes := map[string]string{}
	for name := range files {
		modes[name] = snapshot.RegularMode
	}
	fixed := &snapshot.Snapshot{ID: "parallel-wave-fixed-control", Files: files, Modes: modes}
	project, err := app.Parse(fixed)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("structural fixture diagnostics: %#v", project.Diagnostics)
	}
	initialDigest := fixed.Digest()
	plans := map[string]app.CommandAdapterPlan{}
	for _, name := range []string{"github-capture", "azure-capture"} {
		observed, err := app.ObserveCommandAdapter(project, name)
		if err != nil || observed.Status != "complete" {
			t.Fatalf("%s observe: %#v, %v", name, observed, err)
		}
		plan, err := app.PlanCommandAdapter(project, name, "fixture-tool", "fixture-digest")
		if err != nil {
			t.Fatalf("%s plan: %v", name, err)
		}
		again, err := app.PlanCommandAdapter(project, name, "fixture-tool", "fixture-digest")
		if err != nil {
			t.Fatal(err)
		}
		one, _ := app.YAML(plan)
		two, _ := app.YAML(again)
		if string(one) != string(two) || len(plan.Result.Operations) != 0 {
			t.Fatalf("%s plan is nondeterministic or contains operations", name)
		}
		verified, err := app.VerifyCommandAdapter(project, name, "fixture-tool", "fixture-digest", plan)
		if err != nil || verified.Status != "complete" {
			t.Fatalf("%s verify: %#v, %v", name, verified, err)
		}
		if _, err := app.ApplyCommandAdapter(project, name, "fixture-tool", "fixture-digest", plan); err == nil || !strings.Contains(err.Error(), "does not declare apply") {
			t.Fatalf("%s Apply must remain unavailable: %v", name, err)
		}
		plans[name] = plan
	}
	if plans["github-capture"].ModelDigest != plans["azure-capture"].ModelDigest {
		t.Fatal("adapters did not consume the same normalized model")
	}
	if fixed.Digest() != initialDigest {
		t.Fatal("read-only adapter workflow changed canonical snapshot")
	}

	// Changing only one captured observation must not contaminate the other
	// observer. Overall saved-plan source invalidation stays conservative.
	files["captures/github.json"] = []byte(strings.Replace(string(files["captures/github.json"]), `"default_branch":"main"`, `"default_branch":"release"`, 1))
	changed, err := app.Parse(fixed)
	if err != nil || len(changed.Diagnostics) != 0 {
		t.Fatalf("changed fixture: %v", err)
	}
	azure, err := app.ObserveCommandAdapter(changed, "azure-capture")
	if err != nil || azure.Status != "complete" {
		t.Fatalf("unrelated observer affected: %v", err)
	}
	originalAzure, err := app.YAML(plans["azure-capture"].Observation)
	if err != nil {
		t.Fatal(err)
	}
	changedAzure, err := app.YAML(azure)
	if err != nil {
		t.Fatal(err)
	}
	if string(originalAzure) != string(changedAzure) {
		t.Fatal("changing GitHub capture contaminated Azure observation")
	}
	githubPlan, err := app.PlanCommandAdapter(changed, "github-capture", "fixture-tool", "fixture-digest")
	if err != nil {
		t.Fatal(err)
	}
	github, err := app.VerifyCommandAdapter(changed, "github-capture", "fixture-tool", "fixture-digest", githubPlan)
	if err != nil || github.Status != "failed" {
		t.Fatalf("capture drift silently passed: %#v, %v", github, err)
	}
	if _, err := app.VerifyCommandAdapter(changed, "azure-capture", "fixture-tool", "fixture-digest", plans["azure-capture"]); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("overall source change must invalidate saved plan: %v", err)
	}
}
