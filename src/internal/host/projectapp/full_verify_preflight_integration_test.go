package projectapp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"go.yaml.in/yaml/v3"
)

func TestFullVerifyRejectsUnknownRepositoryCoverageBeforeManagerCalls(t *testing.T) {
	root := deliveryFixture(t)
	logPath := filepath.Join(t.TempDir(), "full-verify.jsonl")
	t.Setenv(deliveryWorkerLogEnv, logPath)
	deliveryWrite(t, root, "outside/unclassified.txt", "not represented by the project model\n")
	git(t, root, "add", "outside/unclassified.txt")
	git(t, root, "commit", "-m", "add unclassified repository path")
	head := git(t, root, "rev-parse", "HEAD")
	report, err := NewOperations().FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
	if err == nil || report.Status != "failed" || !strings.Contains(report.Error, "coverage census") {
		t.Fatalf("unknown repository path did not fail the full census gate: report=%+v err=%v", report, err)
	}
	if report.Starts != 0 || len(report.Managers) != 0 {
		t.Fatalf("provider or check calls started before coverage was conforming: %+v", report)
	}
	assertDeliveryLogEmpty(t, logPath)
}

func TestFullVerifyRejectsStaleGeneratedDocumentBeforeManagerCalls(t *testing.T) {
	root := deliveryFixture(t)
	logPath := filepath.Join(t.TempDir(), "full-verify.jsonl")
	t.Setenv(deliveryWorkerLogEnv, logPath)
	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("stale hand-edited document\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "README.md")
	git(t, root, "commit", "-m", "make generated document stale")
	head := git(t, root, "rev-parse", "HEAD")
	report, err := NewOperations().FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
	if err == nil || report.Status != "failed" || !strings.Contains(report.Error, "generated documentation is missing or stale") {
		t.Fatalf("stale generated document did not fail the pre-provider gate: report=%+v err=%v", report, err)
	}
	if report.Starts != 0 || len(report.Managers) != 0 {
		t.Fatalf("provider or check calls started before document validation: %+v", report)
	}
	assertDeliveryLogEmpty(t, logPath)
	if data, readErr := os.ReadFile(readme); readErr != nil || string(data) != "stale hand-edited document\n" {
		t.Fatalf("preflight changed the stale source document: content=%q err=%v", data, readErr)
	}
}

func TestApplicationFullVerifyRejectsUnknownStrictnessBeforeWorkerStarts(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
	logPath := filepath.Join(t.TempDir(), "full-verify.jsonl")
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyLogEnv, logPath)
	updateDeliveryRuntime(t, root, func(runtime *projectconfig.Runtime) {
		runtime.Strictness = &projectconfig.StrictnessConfig{Managers: map[string]projectconfig.StrictnessProfile{"not-in-model": {Evidence: []string{"typed evidence"}}}}
	})
	head := deliveryGit(t, root, "rev-parse", "HEAD")
	report, err := deliveryOperations().FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
	if err == nil || report.Status != "failed" || !strings.Contains(report.Error, "Manager absent from the accepted model") {
		t.Fatalf("unknown strictness Manager was not rejected: report=%+v err=%v", report, err)
	}
	if report.Starts != 0 {
		t.Fatalf("strictness validation happened after checks or provider calls: starts=%d", report.Starts)
	}
	if data, readErr := os.ReadFile(logPath); readErr == nil && len(data) != 0 {
		t.Fatalf("strictness validation started a Manager worker: %s", data)
	} else if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatal(readErr)
	}
}

func assertDeliveryLogEmpty(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		if data, readErr := os.ReadFile(path); readErr == nil && len(data) != 0 {
			t.Fatalf("Manager called before preflight gate passed: %s", data)
		}
	}
}

func updateDeliveryRuntime(t *testing.T, root string, update func(*projectconfig.Runtime)) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(projectconfig.RuntimePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var runtime projectconfig.Runtime
	if err := yaml.Unmarshal(data, &runtime); err != nil {
		t.Fatal(err)
	}
	update(&runtime)
	updated, err := yaml.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	deliveryWrite(t, root, projectconfig.RuntimePath, string(updated))
	deliveryGit(t, root, "add", projectconfig.RuntimePath)
	deliveryGit(t, root, "commit", "-m", "update Runtime fixture")
}
