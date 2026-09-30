package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
)

func TestVerifyWithoutDeclaredChecksIsIncompleteGraphOnlyEvidence(t *testing.T) {
	repo := newVerifyCLIRepo(t, nil, nil)
	code, output, stderr := invoke("verify", "--repo", repo.root, "--revision", repo.base)
	if code != 2 {
		t.Fatalf("verify exit=%d, want incomplete exit 2; stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[report](t, output)
	if result.Status != "incomplete" || result.Revision != repo.base || result.Provisional || result.Digest == "" || result.ToolDigest == "" {
		t.Fatalf("verify omitted fixed snapshot identity or incomplete status: %#v", result)
	}
	if len(result.Gates) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "verify.incomplete-evidence" || !strings.Contains(result.Diagnostics[0].Message, "graph-only") {
		t.Fatalf("missing checks were not clearly reported as graph-only evidence: %#v", result)
	}
	if strings.Contains(result.Coverage, "all declared repository checks passed") || !strings.Contains(stderr, "incomplete-evidence") {
		t.Fatalf("incomplete verification made a passing-coverage claim or omitted stderr summary: coverage=%q stderr=%q", result.Coverage, stderr)
	}
}

func TestVerifyRunsDeclaredCheckFromFixedSnapshotDespiteWorkingTreeEdit(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go unavailable for the portable snapshot check")
	}
	fixed := []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"fixed snapshot marker\") }\n")
	check := core.Check{Name: "snapshot-content", Run: []string{"go", "run", "scripts/verify-check.go"}}
	repo := newVerifyCLIRepo(t, []core.Check{check}, fixed)
	writeRepoFile(t, repo.root, "scripts/verify-check.go", []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"mutated worktree marker\") }\n"))

	code, output, stderr := invoke("verify", "--repo", repo.root, "--revision", repo.base)
	if code != 0 {
		t.Fatalf("verify exit=%d, want pass; stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[report](t, output)
	if result.Status != "passed" || result.Revision != repo.base || result.Provisional || len(result.Gates) != 1 {
		t.Fatalf("verify did not report one completed fixed-snapshot check: %#v", result)
	}
	gate := result.Gates[0]
	if gate.Name != check.Name || gate.Tool != "go" || gate.ExitCode != 0 || !strings.Contains(gate.Output, "fixed snapshot marker") || strings.Contains(gate.Output, "mutated worktree marker") {
		t.Fatalf("check did not execute the immutable revision bytes: %#v", gate)
	}
	if !strings.Contains(result.Coverage, "all declared repository checks passed") {
		t.Fatalf("coverage did not describe the declared checks accurately: %q", result.Coverage)
	}
}

func TestVerifyToolMissingAndGateFailureAreDistinct(t *testing.T) {
	missingRepo := newVerifyCLIRepo(t, []core.Check{{Name: "missing-tool", Run: []string{"markitect-unavailable-check-tool-20260930"}}}, nil)
	code, output, stderr := invoke("verify", "--repo", missingRepo.root, "--revision", missingRepo.base)
	if code != 2 {
		t.Fatalf("missing-tool verify exit=%d, want incomplete exit 2; stderr=%s output=%s", code, stderr, output)
	}
	missing := decodeYAML[report](t, output)
	if missing.Status != "incomplete" || len(missing.Gates) != 0 || len(missing.Diagnostics) != 1 || missing.Diagnostics[0].Code != "verify.tool-missing" {
		t.Fatalf("unknown tool was not classified as incomplete evidence: %#v", missing)
	}

	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go unavailable for the portable failing-check test")
	}
	checks := []core.Check{
		{Name: "go-version", Run: []string{"go", "version"}},
		{Name: "unknown-go-subcommand", Run: []string{"go", "markitect-no-such-go-subcommand"}},
	}
	failureRepo := newVerifyCLIRepo(t, checks, nil)
	code, output, stderr = invoke("verify", "--repo", failureRepo.root, "--revision", failureRepo.base)
	if code != 1 {
		t.Fatalf("failed-check verify exit=%d, want failure exit 1; stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[report](t, output)
	if result.Status != "failed" || len(result.Gates) != 2 || result.Gates[0].Name != "go-version" || result.Gates[0].ExitCode != 0 || result.Gates[1].Name != "unknown-go-subcommand" || result.Gates[1].ExitCode == 0 {
		t.Fatalf("verify did not preserve successful and failing declared checks: %#v", result.Gates)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "verify.gate-failure" || result.Diagnostics[0].Path != "unknown-go-subcommand" {
		t.Fatalf("failure diagnostic did not identify the declared check: %#v", result.Diagnostics)
	}
	if strings.Contains(result.Coverage, "all declared repository checks passed") || !strings.Contains(result.Coverage, "later checks were not run") || !strings.Contains(stderr, "gate-failure") {
		t.Fatalf("failed verification made a complete-coverage claim: coverage=%q stderr=%q", result.Coverage, stderr)
	}
}

func newVerifyCLIRepo(t *testing.T, checks []core.Check, checkSource []byte) cliRepo {
	t.Helper()
	root := t.TempDir()
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-b", "verify-fixture")
	git(t, root, "config", "user.name", "Markitect Verify Test")
	git(t, root, "config", "user.email", "markitect-verify-test@example.invalid")

	project := core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: "verification-fixture"},
		Spec:       core.Spec{Checks: checks},
	}
	data, err := format.Encode(&project)
	if err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, root, "markitect.yaml", data)
	if checkSource != nil {
		writeRepoFile(t, root, "scripts/verify-check.go", checkSource)
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "verification fixture")
	return cliRepo{root: root, base: git(t, root, "rev-parse", "HEAD")}
}
