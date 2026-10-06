package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"go.yaml.in/yaml/v3"
)

func TestCanonicalModulePreviewPinModelAndSelectedContext(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("modules/sample/module.yaml", `apiVersion: markitect.example.org/module/v1alpha1
name: sample-schema
version: 1.0.0
type: schema
purpose: Schema fixture.
requires:
  core: "1"
provides:
  schemas:
    - schemas/service.yaml
  projectors: []
`)
	write("modules/sample/schemas/service.yaml", `apiVersion: demo.example/v1
purpose: Service model.
kinds:
  Service:
    purpose: A deployable service.
    properties: {}
`)
	write("definitions/service.yaml", `apiVersion: demo.example/v1
kind: Service
metadata:
  namespace: application
  name: api
purpose: The API service.
spec: {}
`)
	config := `apiVersion: markitect.canonical/v1alpha1
kind: Source
definitions:
  - definitions/service.yaml
modules:
  - manifest: modules/sample/module.yaml
    files:
      - modules/sample/schemas/service.yaml
`
	write("canonical.yaml", config)
	before := canonicalFiles(t, root)

	var preview struct {
		Status      string                        `yaml:"status"`
		Provisional bool                          `yaml:"provisional"`
		Modules     []host.CanonicalModulePreview `yaml:"modules"`
	}
	if code := runCanonicalCLI(t, root, "modules", "canonical.yaml", nil, &preview); code != 0 {
		t.Fatalf("modules preview exit = %d, output: %s", code, previewDump(preview))
	}
	if preview.Status != "preview" || !preview.Provisional || len(preview.Modules) != 1 {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	pin := preview.Modules[0].Pin
	config += "    pin:\n      name: " + pin.Name + "\n      version: " + pin.Version + "\n      digest: " + pin.Digest + "\n"
	write("canonical.yaml", config)

	var model struct {
		Status string `yaml:"status"`
		Model  struct {
			Definitions []any `yaml:"definitions"`
		} `yaml:"model"`
	}
	if code := runCanonicalCLI(t, root, "model", "canonical.yaml", nil, &model); code != 0 {
		t.Fatalf("model exit = %d, output: %s", code, previewDump(model))
	}
	if model.Status != "passed" || len(model.Model.Definitions) != 1 {
		t.Fatalf("unexpected compiled Model: %#v", model)
	}

	var context struct {
		Status    string `yaml:"status"`
		Selection struct {
			Definition struct {
				Metadata struct {
					Name string `yaml:"name"`
				} `yaml:"metadata"`
			} `yaml:"definition"`
			Outgoing []any `yaml:"outgoingReferences"`
		} `yaml:"selection"`
	}
	selector := []string{"--api-version", "demo.example/v1", "--kind", "Service", "--namespace", "application", "--name", "api"}
	if code := runCanonicalCLI(t, root, "context", "canonical.yaml", selector, &context); code != 0 {
		t.Fatalf("context exit = %d, output: %s", code, previewDump(context))
	}
	if context.Status != "selected" || context.Selection.Definition.Metadata.Name != "api" || len(context.Selection.Outgoing) != 0 {
		t.Fatalf("unexpected selected context: %#v", context)
	}
	write("definitions/service.yaml", `apiVersion: demo.example/v1
kind: Service
metadata:
  namespace: application
  name: api
purpose: The API service.
spec:
  unexpected: true
`)
	var failed struct {
		Status      string `yaml:"status"`
		Diagnostics []any  `yaml:"diagnostics"`
	}
	if code := runCanonicalCLI(t, root, "model", "canonical.yaml", nil, &failed); code != 1 {
		t.Fatalf("structurally invalid model exit = %d, output: %s", code, previewDump(failed))
	}
	if failed.Status != "failed" || len(failed.Diagnostics) == 0 {
		t.Fatalf("structural failure was not reported: %#v", failed)
	}
	if after := canonicalFiles(t, root); !equalStrings(before, after) {
		t.Fatalf("canonical actions created or removed files: before=%v after=%v", before, after)
	}
}

func TestCanonicalControllerActionOptionBoundaries(t *testing.T) {
	const base = "0123456789abcdef0123456789abcdef01234567"
	const revision = "89abcdef0123456789abcdef0123456789abcdef"
	common := []string{"--action", "controller-propose", "--repo", ".", "--config", "canonical.yaml", "--runtime", "runtime.json", "--base", base, "--revision", revision}
	parse := func(args ...string) (commandOptions, int) {
		t.Helper()
		options, code, done := parseOptions("canonical", append([]string{"canonical"}, args...), mustCommandFlags("canonical"), &bytes.Buffer{}, &bytes.Buffer{})
		if !done {
			return options, 0
		}
		return options, code
	}
	if options, code := parse(common...); code != 0 || options.action != "controller-propose" || options.runtime != "runtime.json" {
		t.Fatalf("controller proposal options rejected: code=%d options=%#v", code, options)
	}
	for _, test := range []struct {
		name string
		args []string
	}{
		{"runtime required", []string{"--action", "controller-propose", "--repo", ".", "--config", "canonical.yaml", "--base", base, "--revision", revision}},
		{"base required", []string{"--action", "controller-propose", "--repo", ".", "--config", "canonical.yaml", "--runtime", "runtime.json", "--revision", revision}},
		{"revision required", []string{"--action", "controller-propose", "--repo", ".", "--config", "canonical.yaml", "--runtime", "runtime.json", "--base", base}},
		{"write forbidden on proposal", append(append([]string{}, common...), "--write")},
		{"plan forbidden on execute", append(append([]string{}, replaceArg(common, "controller-propose", "controller-execute")...), "--plan", "saved.json")},
		{"report forbidden", append(append([]string{}, common...), "--report", "report.json")},
		{"evidence forbidden", append(append([]string{}, common...), "--evidence", "evidence.json")},
		{"selectors forbidden", append(append([]string{}, common...), "--kind", "Service")},
		{"expect forbidden on proposal", append(append([]string{}, common...), "--expect", "sha256:abc")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, code := parse(test.args...); code != 2 {
				t.Fatalf("invalid controller options returned code %d", code)
			}
		})
	}

	apply := replaceArg(common, "controller-propose", "controller-apply")
	if _, code := parse(apply...); code != 2 {
		t.Fatal("controller apply accepted missing explicit write, saved run and expected digest")
	}
	apply = append(apply, "--plan", "saved.json", "--expect", "sha256:reviewed-run", "--write")
	if _, code := parse(apply...); code != 0 {
		t.Fatal("controller apply rejected its required write tuple")
	}
	verify := replaceArg(common, "controller-propose", "controller-verify")
	if _, code := parse(append(verify, "--write")...); code != 0 {
		t.Fatal("controller verify must allow its explicit external-ledger write")
	}
	if _, code := parse(append(verify, "--plan", "saved.json")...); code != 2 {
		t.Fatal("controller verify accepted --plan")
	}

	// Legacy canonical verbs retain their prior option contracts.
	legacyVerify := []string{"--action", "verify", "--repo", ".", "--config", "canonical.yaml", "--base", base, "--revision", revision, "--evidence", "record.json"}
	if _, code := parse(legacyVerify...); code != 0 {
		t.Fatal("legacy canonical verify options regressed")
	}
	if _, code := parse(append(legacyVerify, "--write")...); code != 2 {
		t.Fatal("legacy canonical verify unexpectedly accepted --write")
	}
	if _, code := parse(append(legacyVerify, "--runtime", "runtime.json")...); code != 2 {
		t.Fatal("legacy canonical verify unexpectedly accepted --runtime")
	}
}

func TestCanonicalControllerStatusExitKeepsMaterializationUnverified(t *testing.T) {
	for status, want := range map[string]int{
		"passed":                  0,
		"planned":                 0,
		"materialized-unverified": 0,
		"failed":                  1,
		"escalated":               1,
		"incomplete":              2,
		"unknown":                 2,
	} {
		if got := canonicalControllerStatusExit(status); got != want {
			t.Errorf("status %q exit = %d, want %d", status, got, want)
		}
	}
}

func TestCanonicalControllerFailureEmitsPartialJSONReport(t *testing.T) {
	runtimeFile := filepath.Join(t.TempDir(), "controller-runtime.json")
	runner := host.CanonicalRunnerConfig{
		Command: "agent", Args: []string{}, Model: "test-model", ModelOptions: json.RawMessage("null"),
		ProviderVersion: "test", TimeoutSeconds: 1, MaxStdoutBytes: 1024, MaxStderrBytes: 1024,
	}
	controllerConfig := host.CanonicalControllerConfig{
		APIVersion:  host.CanonicalControllerAPIVersion,
		RecordStore: filepath.Join(t.TempDir(), "records"), PrivateLogs: filepath.Join(t.TempDir(), "logs"),
		CheckInputs: []string{}, Executor: runner, Verifier: runner,
	}
	runtimeConfig, err := json.Marshal(controllerConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimeFile, runtimeConfig, 0600); err != nil {
		t.Fatal(err)
	}
	missingRoot := filepath.Join(t.TempDir(), "missing-repository")
	args := []string{
		"canonical", "--action", "controller-propose", "--repo", missingRoot,
		"--config", "canonical.yaml", "--runtime", runtimeFile,
		"--base", strings.Repeat("a", 40), "--revision", strings.Repeat("b", 40),
	}
	var out, errout bytes.Buffer
	if code := Run(args, &out, &errout); code != 2 {
		t.Fatalf("controller proposal failure exit=%d stderr=%s", code, errout.String())
	}
	if !json.Valid(out.Bytes()) || len(out.Bytes()) == 0 || out.Bytes()[0] != '{' {
		t.Fatalf("controller failure report is not JSON: %q", out.String())
	}
	var partial map[string]any
	if err := json.Unmarshal(out.Bytes(), &partial); err != nil {
		t.Fatalf("decode partial controller report: %v", err)
	}
	if partial["apiVersion"] != "markitect.canonical/controller/v1alpha1" {
		t.Fatalf("partial report lost its available API version: %#v", partial)
	}
}

func mustCommandFlags(command string) map[string]bool {
	flags, ok := commandFlags(command)
	if !ok {
		panic("unknown test command " + command)
	}
	return flags
}

func replaceArg(args []string, old, replacement string) []string {
	copy := append([]string(nil), args...)
	for i := range copy {
		if copy[i] == old {
			copy[i] = replacement
			return copy
		}
	}
	return copy
}

func TestCanonicalStructuralDiagnosticsAndConfigErrorsUseDistinctExitCodes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bad.yaml"), []byte("apiVersion: markitect.canonical/v1alpha1\nkind: Source\ndefinitions: [a.yaml]\nmodules: []\nunknown: true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if code := Run([]string{"canonical", "--action", "model", "--config", "bad.yaml", "--repo", root}, &output, &bytes.Buffer{}); code != 2 {
		t.Fatalf("invalid source config exit = %d, output: %s", code, output.String())
	}
}

func TestCanonicalRequestUsesOnlySelectedTargetPrefixFromFixedSnapshot(t *testing.T) {
	root, configPath, revision := newCanonicalProjectionRepo(t, map[string]string{
		"src/current.cs":  "// target file\n",
		"docs/outside.md": "not projection target\n",
	})
	before := canonicalFiles(t, root)
	selector := []string{"--api-version", "markitect.foundation/v1", "--kind", "Projection", "--namespace", "commerce", "--name", "application-dotnet"}
	var result struct {
		Status  string `yaml:"status"`
		Request struct {
			Definitions  []any             `yaml:"definitions"`
			TargetFiles  map[string][]byte `yaml:"targetFiles"`
			TargetPrefix string            `yaml:"targetPrefix"`
		} `yaml:"request"`
	}
	args := append([]string{"--revision", revision}, selector...)
	if code := runCanonicalCLI(t, root, "request", configPath, args, &result); code != 0 {
		t.Fatalf("request exit = %d, output: %s", code, previewDump(result))
	}
	if result.Status != "bound" || result.Request.TargetPrefix != "src" || len(result.Request.Definitions) != 3 {
		t.Fatalf("unexpected bounded Projection request: %#v", result)
	}
	if len(result.Request.TargetFiles) != 1 || string(result.Request.TargetFiles["src/current.cs"]) != "// target file\n" {
		t.Fatalf("request target files include content outside the explicit prefix: %#v", result.Request.TargetFiles)
	}
	if after := canonicalFiles(t, root); !equalStrings(before, after) {
		t.Fatalf("read-only request created or removed files: before=%v after=%v", before, after)
	}
	var impact struct {
		Status string `yaml:"status"`
		Impact struct {
			Changed []string `yaml:"changedDefinitions"`
		} `yaml:"impact"`
	}
	impactArgs := []string{"--base", revision, "--revision", revision}
	if code := runCanonicalCLI(t, root, "impact", configPath, impactArgs, &impact); code != 0 {
		t.Fatalf("canonical impact exit = %d, result=%s", code, previewDump(impact))
	}
	if impact.Status != "analyzed" || len(impact.Impact.Changed) != 0 {
		t.Fatalf("same-revision impact should report no changed Definitions: %#v", impact)
	}
}

func TestCanonicalReconcilePlanDerivesWorkWithoutSelectorsAndDoesNotWrite(t *testing.T) {
	root, configPath, revision := newCanonicalProjectionRepo(t, nil)
	before := canonicalFiles(t, root)
	var result struct {
		Status string `json:"status"`
		Plan   struct {
			Status string `json:"status"`
			Work   []struct {
				ProjectionID string `json:"projectionId"`
				Module       struct {
					Name string `json:"name"`
				} `json:"module"`
			} `json:"work"`
		} `json:"plan"`
	}
	args := []string{"--revision", revision, "--base", revision}
	if code := runCanonicalCLI(t, root, "reconcile-plan", configPath, args, &result); code != 0 {
		t.Fatalf("canonical reconcile-plan exit = %d, output=%s", code, previewDump(result))
	}
	if result.Status != "planned" || result.Plan.Status != "planned" || len(result.Plan.Work) != 2 {
		t.Fatalf("expected a read-only work proposal for each configured representation: %#v", result)
	}
	if after := canonicalFiles(t, root); !equalStrings(before, after) {
		t.Fatalf("reconcile-plan created or removed files: before=%v after=%v", before, after)
	}
}

func TestReadCanonicalProjectionRecordsRequiresClosedUniqueJSONArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	cases := map[string]string{
		"empty array":          `[]`,
		"not an array":         `{}`,
		"unknown record field": `[{"unexpected":true}]`,
		"duplicate object key": `[{"id":"first","id":"second"}]`,
		"multiple JSON values": `[] []`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(input), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := readCanonicalProjectionRecords(path)
			if name == "empty array" {
				if err != nil || len(got) != 0 {
					t.Fatalf("empty array = %#v, %v", got, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("accepted invalid Projection Record input %s", name)
			}
		})
	}
}

func TestReadCanonicalAdoptionSelectionRequiresClosedExactJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "selection.json")
	cases := map[string]string{
		"valid explicit empty active set": `{"artifacts":["src/a.cs"],"activeRecords":[],"reviewReference":"owner-review"}`,
		"missing active set":              `{"artifacts":["src/a.cs"],"reviewReference":"owner-review"}`,
		"null active set":                 `{"artifacts":["src/a.cs"],"activeRecords":null,"reviewReference":"owner-review"}`,
		"unknown field":                   `{"artifacts":["src/a.cs"],"activeRecords":[],"reviewReference":"owner-review","mode":"replace"}`,
		"duplicate field":                 `{"artifacts":["src/a.cs"],"artifacts":["src/b.cs"],"activeRecords":[],"reviewReference":"owner-review"}`,
		"duplicate artifact":              `{"artifacts":["src/a.cs","src/a.cs"],"activeRecords":[],"reviewReference":"owner-review"}`,
		"unsafe artifact":                 `{"artifacts":["../outside.cs"],"activeRecords":[],"reviewReference":"owner-review"}`,
		"trailing value":                  `{"artifacts":["src/a.cs"],"activeRecords":[],"reviewReference":"owner-review"} {}`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(input), 0600); err != nil {
				t.Fatal(err)
			}
			selection, err := readCanonicalAdoptionSelection(path)
			if name == "valid explicit empty active set" {
				if err != nil || len(selection.Artifacts) != 1 || selection.ActiveRecords == nil {
					t.Fatalf("valid selection = %#v, %v", selection, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("accepted invalid adoption selection %s", name)
			}
		})
	}
}

func TestCanonicalAdoptionPreservesExistingArtifactsAndRequiresFreshApproval(t *testing.T) {
	root, configPath, canonicalRevision := newCanonicalProjectionRepo(t, nil)
	targetFiles := map[string]string{
		"src/Commerce/CreateOrderHandler.cs": "namespace Commerce; public class CreateOrderHandler { }\n",
		"src/Commerce/EffectAxis.cs":         "namespace Commerce; public record EffectAxis { public string Boundary { get; init; } = \"application\"; }\n",
		"src/Commerce/Commerce.csproj":       "<Project><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>\n",
		"src/Commerce/Legacy.cs":             "// Existing artifact outside the explicit adoption selection.\n",
	}
	targetRevision := commitCanonicalFiles(t, root, targetFiles, "existing target representation")
	selectionPath := writeCanonicalSelection(t, map[string]any{
		"artifacts":       []string{"src/Commerce/CreateOrderHandler.cs", "src/Commerce/EffectAxis.cs", "src/Commerce/Commerce.csproj"},
		"activeRecords":   []records.ProjectionRecord{},
		"reviewReference": "owner-review-2026-10-06",
	})
	selector := []string{"--api-version", "markitect.foundation/v1", "--kind", "Projection", "--namespace", "commerce", "--name", "application-dotnet"}
	planArgs := append([]string{"--base", canonicalRevision, "--revision", targetRevision, "--report", selectionPath}, selector...)
	var planned struct {
		Status string `yaml:"status"`
		Plan   struct {
			PlanDigest       string                   `yaml:"planDigest"`
			EvidenceRevision string                   `yaml:"evidenceRevision"`
			Unmatched        []string                 `yaml:"unmatchedArtifacts"`
			Record           records.ProjectionRecord `yaml:"record"`
		} `yaml:"plan"`
	}
	statusBefore := git(t, root, "status", "--porcelain")
	if code := runCanonicalCLI(t, root, "adopt-plan", configPath, planArgs, &planned); code != 0 {
		t.Fatalf("canonical adopt-plan exit = %d, result=%s", code, previewDump(planned))
	}
	if planned.Status != "planned" || planned.Plan.PlanDigest == "" || planned.Plan.EvidenceRevision != targetRevision {
		t.Fatalf("adoption plan did not bind the exact source/target selection: %#v", planned)
	}
	if !containsCanonicalString(planned.Plan.Unmatched, "src/Commerce/Legacy.cs") {
		t.Fatalf("adoption plan hid the unselected in-scope artifact: %#v", planned.Plan.Unmatched)
	}
	if err := records.ValidateProjectionRecord(planned.Plan.Record); err != nil {
		t.Fatalf("prospective adoption Record is invalid: %v", err)
	}
	if after := git(t, root, "status", "--porcelain"); after != statusBefore {
		t.Fatalf("adopt-plan changed repository state: before=%q after=%q", statusBefore, after)
	}
	writeArgs := append(append([]string(nil), planArgs...), "--write")
	if code := runCanonicalCLI(t, root, "adopt-plan", configPath, writeArgs, nil); code != 2 {
		t.Fatalf("adopt-plan with --write exit = %d, want read-only refusal", code)
	}
	for path, want := range targetFiles {
		if got := git(t, root, "show", targetRevision+":"+path); got != strings.TrimSuffix(want, "\n") {
			t.Fatalf("planned adoption changed existing artifact %s: %q", path, got)
		}
	}

	wrongDigest := "sha256:" + strings.Repeat("0", 64)
	badArgs := append([]string{"--base", canonicalRevision, "--revision", targetRevision, "--report", selectionPath, "--expect", wrongDigest}, selector...)
	if code := runCanonicalCLI(t, root, "adopt", configPath, badArgs, nil); code != 2 {
		t.Fatalf("adopt with mismatched approval exit = %d, want 2", code)
	}
	if after := git(t, root, "status", "--porcelain"); after != statusBefore {
		t.Fatalf("mismatched approval changed repository state: before=%q after=%q", statusBefore, after)
	}

	adoptArgs := append([]string{"--base", canonicalRevision, "--revision", targetRevision, "--report", selectionPath, "--expect", planned.Plan.PlanDigest}, selector...)
	var adopted struct {
		Status       string                    `yaml:"status"`
		Record       *records.ProjectionRecord `yaml:"record"`
		Unmatched    []string                  `yaml:"unmatchedArtifacts"`
		Verification struct {
			Result records.VerificationResult `yaml:"result"`
		} `yaml:"verification"`
	}
	if code := runCanonicalCLI(t, root, "adopt", configPath, adoptArgs, &adopted); code != 0 {
		t.Fatalf("canonical adopt exit = %d, result=%s", code, previewDump(adopted))
	}
	if adopted.Status != "adopted" || adopted.Record == nil || adopted.Record.Origin != records.OriginAdopted || adopted.Verification.Result.Outcome != records.OutcomePassed {
		t.Fatalf("adoption did not return a passed, adopted provenance Record: %#v", adopted)
	}
	if !containsCanonicalString(adopted.Unmatched, "src/Commerce/Legacy.cs") {
		t.Fatalf("adopt output hid unresolved target ownership: %#v", adopted.Unmatched)
	}
	if err := records.ValidateProjectionRecord(*adopted.Record); err != nil {
		t.Fatalf("adopted Record is invalid: %v", err)
	}
	if after := git(t, root, "status", "--porcelain"); after != statusBefore {
		t.Fatalf("adopt changed repository state: before=%q after=%q", statusBefore, after)
	}

	targetFiles["src/Commerce/EffectAxis.cs"] = "namespace Commerce; public record EffectAxis { }\n"
	changedTargetRevision := commitCanonicalFiles(t, root, targetFiles, "break existing representation check")
	staleArgs := append([]string{"--base", canonicalRevision, "--revision", changedTargetRevision, "--report", selectionPath, "--expect", planned.Plan.PlanDigest}, selector...)
	if code := runCanonicalCLI(t, root, "adopt", configPath, staleArgs, nil); code != 2 {
		t.Fatalf("adopt with a stale target-revision digest exit = %d, want 2", code)
	}
	changedPlanArgs := append([]string{"--base", canonicalRevision, "--revision", changedTargetRevision, "--report", selectionPath}, selector...)
	var changedPlan struct {
		Plan struct {
			PlanDigest string `yaml:"planDigest"`
		} `yaml:"plan"`
	}
	if code := runCanonicalCLI(t, root, "adopt-plan", configPath, changedPlanArgs, &changedPlan); code != 0 || changedPlan.Plan.PlanDigest == "" {
		t.Fatalf("changed-target adoption plan exit=%d result=%s", code, previewDump(changedPlan))
	}
	failedArgs := append([]string{"--base", canonicalRevision, "--revision", changedTargetRevision, "--report", selectionPath, "--expect", changedPlan.Plan.PlanDigest}, selector...)
	var failedAdoption struct {
		Status string                    `yaml:"status"`
		Record *records.ProjectionRecord `yaml:"record"`
	}
	if code := runCanonicalCLI(t, root, "adopt", configPath, failedArgs, &failedAdoption); code != 1 {
		t.Fatalf("adopt with failing fixed check exit = %d, result=%s", code, previewDump(failedAdoption))
	}
	if failedAdoption.Status != "failed" || failedAdoption.Record != nil {
		t.Fatalf("failed checks emitted an adoption Record: %#v", failedAdoption)
	}
}

func containsCanonicalString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func writeCanonicalSelection(t *testing.T, selection map[string]any) string {
	t.Helper()
	data, err := json.Marshal(selection)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "selection.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func commitCanonicalFiles(t *testing.T, root string, files map[string]string, message string) string {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", message)
	return git(t, root, "rev-parse", "HEAD")
}

func TestCanonicalPlanApplyRequiresSavedReportAndWritesUnverifiedRecord(t *testing.T) {
	root, configPath, revision := newCanonicalProjectionRepo(t, map[string]string{"docs/outside.md": "outside target\n"})
	selector := []string{"--api-version", "markitect.foundation/v1", "--kind", "Projection", "--namespace", "commerce", "--name", "application-markdown"}
	planArgs := append([]string{"--revision", revision}, selector...)
	var planned struct {
		Status          string `yaml:"status"`
		CandidateDigest string `yaml:"candidateDigest"`
		Plan            any    `yaml:"plan"`
	}
	code, reportBytes := runCanonicalCLIOutput(t, root, "plan", configPath, planArgs, &planned)
	if code != 0 || planned.Status != "planned" || planned.CandidateDigest == "" || planned.Plan == nil {
		t.Fatalf("canonical plan failed: code=%d report=%s", code, reportBytes)
	}
	planRoot := t.TempDir()
	planPath := filepath.Join(planRoot, "reviewed-plan.yaml")
	if err := os.WriteFile(planPath, reportBytes, 0600); err != nil {
		t.Fatal(err)
	}
	badPlanPath := filepath.Join(planRoot, "tampered-plan.yaml")
	tampered := bytes.Replace(reportBytes, []byte("candidateDigest: "+planned.CandidateDigest), []byte("candidateDigest: sha256:"+strings.Repeat("0", 64)), 1)
	if bytes.Equal(tampered, reportBytes) {
		t.Fatalf("could not locate candidate digest in saved report")
	}
	if err := os.WriteFile(badPlanPath, tampered, 0600); err != nil {
		t.Fatal(err)
	}
	applyArgs := append([]string{"--revision", revision, "--plan", badPlanPath, "--expect", planned.CandidateDigest, "--write"}, selector...)
	var errout bytes.Buffer
	if code := Run(append([]string{"canonical", "--action", "apply", "--repo", root, "--config", configPath}, applyArgs...), &bytes.Buffer{}, &errout); code != 2 {
		t.Fatalf("tampered report exit = %d, stderr=%s", code, errout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "represented", "index.md")); !os.IsNotExist(err) {
		t.Fatalf("tampered plan produced output, stat err=%v", err)
	}

	applyArgs = append([]string{"--revision", revision, "--plan", planPath, "--expect", planned.CandidateDigest, "--write"}, selector...)
	var applied struct {
		Status  string                    `yaml:"status"`
		Written []string                  `yaml:"written"`
		Record  *records.ProjectionRecord `yaml:"record"`
		Error   string                    `yaml:"error"`
	}
	if code := runCanonicalCLI(t, root, "apply", configPath, applyArgs, &applied); code != 0 {
		t.Fatalf("canonical apply exit = %d, result=%s, error=%s", code, previewDump(applied), applied.Error)
	}
	if applied.Status != "materialized-unverified" || len(applied.Written) != 1 || applied.Record == nil {
		t.Fatalf("apply did not return its unverified operational record: %#v", applied)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "represented", "index.md")); err != nil {
		t.Fatalf("deterministic target was not written: %v", err)
	}
	if applied.Record == nil {
		t.Fatal("apply did not return its Projection Record")
	}
	if err := records.ValidateProjectionRecord(*applied.Record); err != nil {
		t.Fatalf("apply-returned record does not validate: %v", err)
	}
	recordBytes, err := json.Marshal(applied.Record)
	if err != nil {
		t.Fatal(err)
	}
	recordPath := filepath.Join(t.TempDir(), "projection-record.json")
	if err := os.WriteFile(recordPath, recordBytes, 0600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "docs/represented/index.md")
	git(t, root, "commit", "-m", "commit projection candidate for verification")
	targetRevision := git(t, root, "rev-parse", "HEAD")
	statusBeforeVerify := git(t, root, "status", "--porcelain")
	var verified struct {
		Status           string `yaml:"status"`
		EvidenceRevision string `yaml:"evidenceRevision"`
		Result           struct {
			Outcome string `yaml:"outcome"`
			Reason  string `yaml:"reason"`
		} `yaml:"result"`
		VerificationError string `yaml:"verificationError"`
	}
	verifyArgs := []string{"--base", revision, "--revision", targetRevision, "--evidence", recordPath}
	if code := runCanonicalCLI(t, root, "verify", configPath, verifyArgs, &verified); code != 1 {
		t.Fatalf("canonical verify with a failing fixed fixture check exit = %d, result=%s", code, previewDump(verified))
	}
	if verified.Status != "failed" || verified.Result.Outcome != "failed" || verified.EvidenceRevision != targetRevision {
		t.Fatalf("verification did not report gate failure at the exact target revision: %#v", verified)
	}
	if statusAfterVerify := git(t, root, "status", "--porcelain"); statusAfterVerify != statusBeforeVerify {
		t.Fatalf("verify changed repository state: before=%q after=%q", statusBeforeVerify, statusAfterVerify)
	}
}

func runCanonicalCLI(t *testing.T, root, action, config string, selector []string, result any) int {
	code, _ := runCanonicalCLIOutput(t, root, action, config, selector, result)
	return code
}

func runCanonicalCLIOutput(t *testing.T, root, action, config string, selector []string, result any) (int, []byte) {
	t.Helper()
	args := []string{"canonical", "--action", action, "--config", config, "--repo", root}
	args = append(args, selector...)
	var out, errout bytes.Buffer
	code := Run(args, &out, &errout)
	if errout.Len() > 0 && code != 0 {
		t.Logf("canonical stderr: %s", errout.String())
	}
	if result != nil && out.Len() > 0 {
		if err := yaml.Unmarshal(out.Bytes(), result); err != nil {
			t.Fatalf("decode canonical output %q: %v", out.String(), err)
		}
	}
	return code, append([]byte(nil), out.Bytes()...)
}

func canonicalFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			rel, e := filepath.Rel(root, file)
			if e != nil {
				return e
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func copyCanonicalFixture(sourceRoot, destination string) error {
	return filepath.WalkDir(sourceRoot, func(source string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(sourceRoot, source)
		if err != nil || relative == "." {
			return err
		}
		target := filepath.Join(destination, "examples", "canonical-projection", relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func newCanonicalProjectionRepo(t *testing.T, targetFiles map[string]string) (root, configPath, revision string) {
	t.Helper()
	workspace, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	sourceFixture := filepath.Join(workspace, "examples", "canonical-projection")
	root = t.TempDir()
	if err := copyCanonicalFixture(sourceFixture, root); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		t.Helper()
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range targetFiles {
		write(name, content)
	}
	configPath = "examples/canonical-projection/canonical.yaml"
	loaded, err := host.LoadCanonicalSource(root, "", configPath, false)
	if err != nil {
		t.Fatal(err)
	}
	previewByManifest := map[string]host.CanonicalModulePreview{}
	for _, preview := range loaded.Previews {
		previewByManifest[preview.Manifest] = preview
	}
	for i := range loaded.Config.Modules {
		preview, ok := previewByManifest[loaded.Config.Modules[i].Manifest]
		if !ok {
			t.Fatalf("module preview missing %s", loaded.Config.Modules[i].Manifest)
		}
		pin := preview.Pin
		loaded.Config.Modules[i].Pin = &pin
	}
	loaded.Config.ProjectionBindings = []canonical.ProjectionBinding{
		{Projection: core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "commerce", Name: "application-dotnet"}, Module: "markitect-dotnet"},
		{Projection: core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "commerce", Name: "application-markdown"}, Module: "markitect-markdown"},
	}
	configBytes, err := yaml.Marshal(loaded.Config)
	if err != nil {
		t.Fatal(err)
	}
	write(configPath, string(configBytes))
	git(t, root, "init", "-b", "codex/canonical-test")
	git(t, root, "config", "user.name", "Markitect Test")
	git(t, root, "config", "user.email", "markitect-test@example.invalid")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "canonical CLI fixture")
	revision = git(t, root, "rev-parse", "HEAD")
	return root, configPath, revision
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func previewDump(value any) string {
	data, _ := yaml.Marshal(value)
	return string(data)
}
