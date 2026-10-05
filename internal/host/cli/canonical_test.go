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
