package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/host/recordstore"
	"go.yaml.in/yaml/v3"
)

func TestCanonicalDurableAdoptionInitializesAbsentLedgerWithoutRepositoryMutation(t *testing.T) {
	root, configPath, canonicalRevision := newCanonicalProjectionRepo(t, nil)
	loaded, err := host.LoadCanonicalSource(root, "", configPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Config.Checks) == 0 {
		t.Fatal("canonical fixture has no declared fixed check")
	}
	loaded.Config.Checks[0].Run = []string{"go", "version"}
	updatedConfig, err := yaml.Marshal(loaded.Config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(configPath)), updatedConfig, 0644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "use local fixed check for durable adoption test")
	canonicalRevision = git(t, root, "rev-parse", "HEAD")
	targetFiles := map[string]string{
		"src/Commerce/CreateOrderHandler.cs": "namespace Commerce; public class CreateOrderHandler { }\n",
		"src/Commerce/EffectAxis.cs":         "namespace Commerce; public record EffectAxis { public string Boundary { get; init; } = \"application\"; }\n",
		"src/Commerce/Commerce.csproj":       "<Project><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>\n",
	}
	targetRevision := commitCanonicalFiles(t, root, targetFiles, "reviewed existing target artifacts")
	selectionPath := writeCanonicalSelection(t, map[string]any{
		"artifacts":       []string{"src/Commerce/CreateOrderHandler.cs", "src/Commerce/EffectAxis.cs", "src/Commerce/Commerce.csproj"},
		"activeRecords":   []records.ProjectionRecord{},
		"reviewReference": "owner-review/durable-adoption-test",
	})
	tempRoot := os.TempDir()
	if runtime.GOOS == "windows" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		tempRoot = filepath.Join(home, "AppData", "Local")
	}
	tempRoot, err = filepath.EvalSymlinks(tempRoot)
	if err != nil {
		t.Fatal(err)
	}
	external, err := os.MkdirTemp(tempRoot, "markitect-adoption-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(external) })
	runtimeConfig := host.CanonicalControllerConfig{
		APIVersion:  host.CanonicalControllerAPIVersion,
		RecordStore: filepath.Join(external, "ledger"), PrivateLogs: filepath.Join(external, "private-logs"),
		CheckInputs: []string{},
		Executor:    host.CanonicalRunnerConfig{Command: "must-not-run-executor", Model: "unused", ProviderVersion: "unused/1", TimeoutSeconds: 10},
		Verifier:    host.CanonicalRunnerConfig{Command: "must-not-run-verifier", Model: "unused", ProviderVersion: "unused/1", TimeoutSeconds: 10},
	}
	runtimePath := filepath.Join(external, "runtime.json")
	runtimeBytes, err := json.Marshal(runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimePath, runtimeBytes, 0600); err != nil {
		t.Fatal(err)
	}
	selector := []string{"--api-version", "markitect.foundation/v1", "--kind", "Projection", "--namespace", "commerce", "--name", "application-dotnet"}
	baseArgs := []string{"--base", canonicalRevision, "--revision", targetRevision, "--report", selectionPath, "--runtime", runtimePath}
	baseArgs = append(baseArgs, selector...)
	statusBefore := git(t, root, "status", "--porcelain")
	headBefore := git(t, root, "rev-parse", "HEAD")
	indexBefore := git(t, root, "diff", "--cached", "--binary")
	var planned struct {
		Status string `yaml:"status"`
		Plan   struct {
			PlanDigest string                              `yaml:"planDigest"`
			Record     records.ProjectionRecord            `yaml:"record"`
			Ledger     host.CanonicalAdoptionLedgerBinding `yaml:"ledger"`
		} `yaml:"plan"`
	}
	if code := runCanonicalCLI(t, root, "adopt-plan", configPath, baseArgs, &planned); code != 0 {
		t.Fatalf("durable adopt-plan exit=%d result=%s", code, previewDump(planned))
	}
	if planned.Status != "planned" || planned.Plan.PlanDigest == "" || planned.Plan.Ledger.Present || planned.Plan.Record.State != records.StateMaterializedUnverified {
		t.Fatalf("durable preview did not bind absent ledger and unverified record: %#v", planned)
	}
	if _, err := os.Stat(runtimeConfig.RecordStore); !os.IsNotExist(err) {
		t.Fatalf("adopt-plan initialized external ledger: %v", err)
	}
	staleConfig := runtimeConfig
	staleConfig.ReferenceDepth = 1
	staleBytes, err := json.Marshal(staleConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimePath, staleBytes, 0600); err != nil {
		t.Fatal(err)
	}
	staleApplyArgs := append([]string{}, baseArgs...)
	staleApplyArgs = append(staleApplyArgs, "--expect", planned.Plan.PlanDigest, "--write")
	if code := runCanonicalCLI(t, root, "adopt", configPath, staleApplyArgs, nil); code == 0 {
		t.Fatal("durable adoption accepted a plan after runtime configuration changed")
	}
	if _, err := os.Stat(runtimeConfig.RecordStore); !os.IsNotExist(err) {
		t.Fatalf("stale runtime approval initialized the ledger: %v", err)
	}
	runtimeBytes, err = json.Marshal(runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimePath, runtimeBytes, 0600); err != nil {
		t.Fatal(err)
	}

	applyArgs := append([]string{}, baseArgs...)
	applyArgs = append(applyArgs, "--expect", planned.Plan.PlanDigest, "--write")
	var applied struct {
		Status          string                    `yaml:"status"`
		Record          *records.ProjectionRecord `yaml:"record"`
		PlanDigest      string                    `yaml:"planDigest"`
		ActiveRecordIDs []string                  `yaml:"activeRecordIds"`
	}
	if code := runCanonicalCLI(t, root, "adopt", configPath, applyArgs, &applied); code != 0 {
		t.Fatalf("durable adopt exit=%d result=%s", code, previewDump(applied))
	}
	if applied.Status != records.StateMaterializedUnverified || applied.Record == nil || applied.Record.Origin != records.OriginAdopted || applied.Record.State != records.StateMaterializedUnverified || len(applied.ActiveRecordIDs) != 1 || applied.ActiveRecordIDs[0] != applied.Record.ID {
		t.Fatalf("durable adoption did not select one unverified adopted record: %#v", applied)
	}
	store, err := recordstore.Open(runtimeConfig.RecordStore, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Records) != 1 || len(state.Verifications) != 0 || len(state.ActiveSelection.RecordIDs) != 1 || state.ActiveSelection.RecordIDs[0] != applied.Record.ID {
		t.Fatalf("durable adoption ledger does not contain only an active unverified record: %#v", state)
	}
	if git(t, root, "status", "--porcelain") != statusBefore || git(t, root, "rev-parse", "HEAD") != headBefore || git(t, root, "diff", "--cached", "--binary") != indexBefore {
		t.Fatal("durable adoption changed source worktree, index, or HEAD")
	}
	for name, content := range targetFiles {
		if got := git(t, root, "show", targetRevision+":"+name); got != strings.TrimSuffix(content, "\n") {
			t.Fatalf("durable adoption changed target artifact %s: %q", name, got)
		}
		treeEntry := git(t, root, "ls-tree", targetRevision, name)
		if !strings.HasPrefix(treeEntry, "100644 blob ") {
			t.Fatalf("durable adoption target mode changed for %s: %q", name, treeEntry)
		}
	}

	// A new preview observes the active owner and refuses implicit replacement.
	if code := runCanonicalCLI(t, root, "adopt-plan", configPath, baseArgs, nil); code != 2 {
		t.Fatalf("durable adoption silently replaced an active owner; exit=%d", code)
	}
}
