package cli

import (
	"encoding/json"
	"errors"
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

func TestCanonicalDurableAdoptionPartialFailureReportIncludesCASCause(t *testing.T) {
	base := map[string]any{}
	cause := errors.New("adoption attempt was appended but active selection failed; attempt remains inactive: projection record store head is stale")
	setCanonicalAdoptionApplyReport(base, host.CanonicalAdoptionApply{
		Status: records.StatePartialFailure, PlanDigest: "sha256:plan", LedgerHead: "sha256:head",
		ActiveRecordIDs: []string{"sha256:prior"}, ActiveSelectionStatus: "observed-not-selected",
	}, cause)
	encoded, err := yaml.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Status          string   `yaml:"status"`
		Error           string   `yaml:"error"`
		LedgerHead      string   `yaml:"ledgerHead"`
		ActiveRecordIDs []string `yaml:"activeRecordIds"`
		SelectionStatus string   `yaml:"activeSelectionStatus"`
	}
	if err := yaml.Unmarshal(encoded, &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != records.StatePartialFailure || report.Error != cause.Error() || report.LedgerHead != "sha256:head" || report.SelectionStatus != "observed-not-selected" || len(report.ActiveRecordIDs) != 1 || report.ActiveRecordIDs[0] != "sha256:prior" {
		t.Fatalf("partial adoption report omitted the diagnostic or unchanged active set: %#v", report)
	}
}

func TestCanonicalDurableAdoptionUnknownSelectionOmitsUnobservedLedgerState(t *testing.T) {
	base := map[string]any{"ledgerHead": "stale", "activeRecordIds": []string{"stale"}}
	setCanonicalAdoptionApplyReport(base, host.CanonicalAdoptionApply{
		Status: records.StatePartialFailure, PlanDigest: "sha256:plan", Record: &records.ProjectionRecord{ID: "sha256:attempt"},
		ActiveSelectionStatus: "unknown",
	}, errors.New("selection call failed; recovery read failed"))
	if _, ok := base["ledgerHead"]; ok {
		t.Fatalf("unknown selection report retained an unobserved ledger head: %#v", base)
	}
	if _, ok := base["activeRecordIds"]; ok {
		t.Fatalf("unknown selection report retained unobserved active ids: %#v", base)
	}
	if base["activeSelectionStatus"] != "unknown" || base["error"] != "selection call failed; recovery read failed" || base["record"].(*records.ProjectionRecord).ID != "sha256:attempt" {
		t.Fatalf("unknown selection report omitted its supported facts: %#v", base)
	}
}

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
		"src/Commerce/adoption-notes.txt":    "additional explicitly configured check input\n",
		"docs/represented/README.md":         "# Retained human-authored Markdown representation\n",
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
		CheckInputs: []string{"src/Commerce/adoption-notes.txt"},
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
	staleApplyArgs := append([]string{}, baseArgs...)
	staleApplyArgs = append(staleApplyArgs, "--expect", planned.Plan.PlanDigest, "--write")
	for name, mutate := range map[string]func(*host.CanonicalControllerConfig){
		"executor-config": func(config *host.CanonicalControllerConfig) { config.Executor.Command = "changed-executor" },
		"check-inputs": func(config *host.CanonicalControllerConfig) {
			config.CheckInputs = []string{"src/Commerce/Commerce.csproj"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			staleConfig := runtimeConfig
			mutate(&staleConfig)
			staleBytes, err := json.Marshal(staleConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(runtimePath, staleBytes, 0600); err != nil {
				t.Fatal(err)
			}
			if code := runCanonicalCLI(t, root, "adopt", configPath, staleApplyArgs, nil); code == 0 {
				t.Fatalf("durable adoption accepted a plan after %s changed", name)
			}
			if _, err := os.Stat(runtimeConfig.RecordStore); !os.IsNotExist(err) {
				t.Fatalf("stale %s approval initialized the ledger: %v", name, err)
			}
		})
	}
	runtimeBytes, err = json.Marshal(runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimePath, runtimeBytes, 0600); err != nil {
		t.Fatal(err)
	}

	// Changing the selected artifact's immutable bytes and Git mode produces a
	// distinct evidence revision; the old plan must be refused before bootstrap.
	changedHandler := filepath.Join(root, "src", "Commerce", "EffectAxis.cs")
	if err := os.WriteFile(changedHandler, []byte("namespace Commerce; public record EffectAxis { }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "Commerce", "adoption-notes.txt"), []byte("changed check input bytes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "src/Commerce/EffectAxis.cs")
	git(t, root, "update-index", "--chmod=+x", "src/Commerce/EffectAxis.cs")
	git(t, root, "commit", "-m", "change selected artifact bytes and mode")
	changedRevision := git(t, root, "rev-parse", "HEAD")
	changedEvidenceArgs := replaceArg(staleApplyArgs, targetRevision, changedRevision)
	if code := runCanonicalCLI(t, root, "adopt", configPath, changedEvidenceArgs, nil); code == 0 {
		t.Fatal("durable adoption accepted a plan for different selected artifact bytes/mode/revision")
	}
	if _, err := os.Stat(runtimeConfig.RecordStore); !os.IsNotExist(err) {
		t.Fatalf("stale selected-artifact approval initialized the ledger: %v", err)
	}
	statusBefore = git(t, root, "status", "--porcelain")
	headBefore = git(t, root, "rev-parse", "HEAD")
	indexBefore = git(t, root, "diff", "--cached", "--binary")

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

	// A separate Projection plan binds the existing ledger head. Appending an
	// unrelated complete attempt advances that head; the saved plan must stale
	// while preserving the original active owner.
	markdownSelectionPath := writeCanonicalSelection(t, map[string]any{
		"artifacts":       []string{"docs/represented/README.md"},
		"activeRecords":   []records.ProjectionRecord{},
		"reviewReference": "owner-review/markdown-durable-adoption-test",
	})
	markdownArgs := replaceArg(append([]string{}, baseArgs...), selectionPath, markdownSelectionPath)
	markdownArgs = replaceArg(markdownArgs, "application-dotnet", "application-markdown")
	var markdownPlan struct {
		Status string `yaml:"status"`
		Plan   struct {
			PlanDigest string                   `yaml:"planDigest"`
			Record     records.ProjectionRecord `yaml:"record"`
			Ledger     struct {
				Present bool   `yaml:"present"`
				Head    string `yaml:"head"`
			} `yaml:"ledger"`
		} `yaml:"plan"`
	}
	if code := runCanonicalCLI(t, root, "adopt-plan", configPath, markdownArgs, &markdownPlan); code != 0 {
		t.Fatalf("existing-ledger durable adopt-plan exit=%d result=%s", code, previewDump(markdownPlan))
	}
	if !markdownPlan.Plan.Ledger.Present || markdownPlan.Plan.Ledger.Head == "" {
		t.Fatalf("durable plan did not bind existing ledger head: %#v", markdownPlan)
	}
	state, err = store.AppendAttempt(state.Head, markdownPlan.Plan.Record)
	if err != nil {
		t.Fatal(err)
	}
	staleLedgerArgs := append([]string{}, markdownArgs...)
	staleLedgerArgs = append(staleLedgerArgs, "--expect", markdownPlan.Plan.PlanDigest, "--write")
	if code := runCanonicalCLI(t, root, "adopt", configPath, staleLedgerArgs, nil); code == 0 {
		t.Fatal("durable adoption accepted a plan after the existing ledger head changed")
	}
	state, err = store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Records) != 2 || len(state.ActiveSelection.RecordIDs) != 1 || state.ActiveSelection.RecordIDs[0] != applied.Record.ID {
		t.Fatalf("stale-head refusal changed existing active ownership: %#v", state)
	}

	// A new preview observes the active owner and refuses implicit replacement.
	if code := runCanonicalCLI(t, root, "adopt-plan", configPath, baseArgs, nil); code != 2 {
		t.Fatalf("durable adoption silently replaced an active owner; exit=%d", code)
	}
}
