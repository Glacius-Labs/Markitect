package projectapp

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

func TestRunRequiresExplicitRootBeforeLoadingOrInvoking(t *testing.T) {
	loads := 0
	host := projectrun.Host{Load: func(string, string) (*projectrun.Project, error) {
		loads++
		return nil, errors.New("unexpected load")
	}}
	invoker := &countingInvoker{}
	_, err := (Operations{Host: host, Invoker: invoker}).Run(context.Background(), RunOperation{RunID: "run-1"})
	if err == nil || err.Error() != "selected project root is required" {
		t.Fatalf("Run without a root error = %v", err)
	}
	if loads != 0 || invoker.calls != 0 {
		t.Fatalf("missing root reached a port: loads=%d invocations=%d", loads, invoker.calls)
	}
}

func TestPlanPassesExplicitSelectionAndUnauthenticatedRequestToExistingService(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-b", "facade-test")
	runGit(t, root, "config", "user.email", "projectapp@example.test")
	runGit(t, root, "config", "user.name", "Project App Test")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "fixture")
	revision := gitOutput(t, root, "rev-parse", "HEAD")
	loadCalls := 0
	var loadedRevision string
	loadErr := errors.New("stop at selected revision")
	host := projectrun.Host{
		Load: func(_ string, selectedRevision string) (*projectrun.Project, error) {
			loadCalls++
			loadedRevision = selectedRevision
			return nil, loadErr
		},
		FromSnapshot: func(string, *projectrun.Snapshot) (*projectrun.Project, error) {
			return nil, errors.New("unexpected snapshot")
		},
		PlanEdit: func(*projectrun.Project, projectrun.Mutation) (projectrun.EditPlan, error) {
			return projectrun.EditPlan{}, errors.New("unexpected edit")
		},
	}
	request := projectrun.PlanRequest{Operation: projectrun.OperationApply, Goal: "Review a bounded change", ExecuteAuthorized: false}
	_, err := (Operations{Host: host}).Plan(PlanOperation{Selection: Selection{Root: root, Revision: revision}, Request: request})
	if !errors.Is(err, loadErr) {
		t.Fatalf("Plan error = %v, want selected frontend error", err)
	}
	if loadCalls != 1 || loadedRevision != revision {
		t.Fatalf("Plan changed explicit selection: calls=%d revision=%q want=%q", loadCalls, loadedRevision, revision)
	}
}

func TestPlanRejectsConflictingExplicitRevisionsBeforeHost(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-b", "facade-test")
	runGit(t, root, "config", "user.email", "projectapp@example.test")
	runGit(t, root, "config", "user.name", "Project App Test")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "fixture")
	loads := 0
	operations := Operations{Host: projectrun.Host{
		Load: func(string, string) (*projectrun.Project, error) {
			loads++
			return nil, errors.New("Host must not be called")
		},
		FromSnapshot: func(string, *projectrun.Snapshot) (*projectrun.Project, error) {
			return nil, errors.New("unexpected snapshot")
		},
		PlanEdit: func(*projectrun.Project, projectrun.Mutation) (projectrun.EditPlan, error) {
			return projectrun.EditPlan{}, errors.New("unexpected edit")
		},
	}}
	_, err := operations.Plan(PlanOperation{
		Selection: Selection{Root: root, Revision: "revision-a"},
		Request:   projectrun.PlanRequest{Operation: projectrun.OperationApply, Goal: "Review a bounded change", BaseRevision: "revision-b"},
	})
	if err == nil || err.Error() != "selected revision conflicts with plan request baseRevision" {
		t.Fatalf("conflicting revisions error = %v", err)
	}
	if loads != 0 {
		t.Fatalf("conflicting revisions reached project Host %d times", loads)
	}
}

func TestUnauthenticatedPlanReturnsPreviewWithoutCreatingRun(t *testing.T) {
	root := makePlanFixture(t)
	operations := Operations{Host: projectrun.Host{
		Load:         projectwork.Load,
		FromSnapshot: projectwork.FromSnapshot,
		PlanEdit:     projectwork.PlanEdit,
		ApplyEdit:    projectwork.ApplyEdit,
	}}
	plan, err := operations.Plan(PlanOperation{
		Selection: Selection{Root: root},
		Request:   projectrun.PlanRequest{Operation: projectrun.OperationApply, Goal: "Review a bounded change"},
	})
	if err != nil {
		t.Fatalf("unauthenticated Plan: %v", err)
	}
	if plan.Status != projectrun.StatusPlanned || plan.ExecuteAuthorized || plan.Digest == "" {
		t.Fatalf("unauthenticated plan did not retain ordinary preview semantics: %#v", plan)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect", "runs")); !os.IsNotExist(err) {
		t.Fatalf("unauthenticated plan created a durable run: %v", err)
	}
}

func makePlanFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	plan, err := projectwork.Init(root, "Facade fixture", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range plan.Files {
		content := file.Content
		if file.Path == projectwork.ManifestPath {
			content = strings.Replace(content, "coverageMode: full", "coverageMode: selected", 1)
		}
		path := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	managerID := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Name: "project-owner"}.Key()
	runtime := projectrun.Runtime{
		APIVersion: projectrun.APIVersion, Mode: projectrun.ModeControlledLocal,
		Agents: map[string]projectrun.Agent{managerID: {
			Command: executable, Model: "fixture-model", ProviderVersion: "fixture/1",
			Timeout: projectrun.Duration(30 * time.Second), MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
			Pricing: projectrun.Pricing{InputMicrosPerMillion: 1},
		}},
		Limits: projectrun.Limits{MaxDepth: 2, MaxStarts: 4, MaxParallel: 1, MaxDuration: projectrun.Duration(time.Minute), MaxCostMicros: 1000,
			MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 1 << 20},
	}
	encodedRuntime, err := yaml.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(projectwork.RuntimePath)), encodedRuntime, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "init", "-b", "facade-plan")
	runGit(t, root, "config", "user.email", "projectapp@example.test")
	runGit(t, root, "config", "user.name", "Project App Test")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "project fixture")
	return root
}

type countingInvoker struct{ calls int }

func (i *countingInvoker) Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	return agentexec.RunResult{}, errors.New("unexpected invocation")
}

func (*countingInvoker) Fingerprint(agentexec.Config) (string, error) { return "fixture", nil }

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(output[:len(output)-1])
}
