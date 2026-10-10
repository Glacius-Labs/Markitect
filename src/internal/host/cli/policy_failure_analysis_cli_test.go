package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
)

func TestPolicyFailureAnalysisCLIIsExplicitAndKeepsAcceptanceStrict(t *testing.T) {
	repo := newPolicyAnalysisCLIRepo(t)

	code, output, stderr := invokePolicyAnalysisCLI(t, "check", "--repo", repo.root, "--revision", repo.failing)
	if code != 1 {
		t.Fatalf("check exit=%d stderr=%s output=%s, want policy-failure exit 1", code, stderr, output)
	}
	checked := decodeYAML[report](t, output)
	if checked.Status != "failed" || len(checked.PolicyResults) != 1 || checked.PolicyResults[0].Status != "failed" {
		t.Fatalf("check hid the failing policy result: %#v", checked)
	}

	code, output, stderr = invokePolicyAnalysisCLI(t, "verify", "--repo", repo.root, "--revision", repo.failing)
	if code != 1 {
		t.Fatalf("verify exit=%d stderr=%s output=%s, want policy-failure exit 1 before gates", code, stderr, output)
	}
	verified := decodeYAML[report](t, output)
	if verified.Status != "failed" || len(verified.Gates) != 0 {
		t.Fatalf("verify did not stop at policy failure: %#v", verified)
	}
	for _, diagnostic := range verified.Diagnostics {
		if strings.HasPrefix(diagnostic.Code, "verify.") {
			t.Fatalf("verify ran or reported a repository gate before rejecting policy: %#v", diagnostic)
		}
	}

	code, output, stderr = invokePolicyAnalysisCLI(t, "context", "--repo", repo.root, "--revision", repo.failing, "--api-version", "report.example.org/v1", "--kind", "Module", "--name", "legacy", "--namespace", "engineering")
	if code != 1 {
		t.Fatalf("default context exit=%d stderr=%s output=%s, want strict policy failure", code, stderr, output)
	}
	strictContext := decodeYAML[report](t, output)
	if strictContext.Status != "failed" {
		t.Fatalf("default context unexpectedly returned analysis: %#v", strictContext)
	}

	code, output, stderr = invokePolicyAnalysisCLI(t, "context", "--repo", repo.root, "--revision", repo.failing, "--api-version", "report.example.org/v1", "--kind", "Module", "--name", "legacy", "--namespace", "engineering", "--analyze-policy-failures")
	if code != 1 {
		t.Fatalf("diagnostic context exit=%d stderr=%s output=%s, want useful output with failing status", code, stderr, output)
	}
	contextResult := decodeYAML[host.Context](t, output)
	if contextResult.Analysis == nil || contextResult.Analysis.Candidate.PolicyStatus != "failed" || !contextResult.Analysis.Complete {
		t.Fatalf("diagnostic context omitted explicit noncompliant status: %#v", contextResult)
	}
	if len(contextResult.PolicyResults) != 1 || contextResult.PolicyResults[0].Status != "failed" {
		t.Fatalf("diagnostic context omitted the failed rule: %#v", contextResult.PolicyResults)
	}

	code, output, stderr = invokePolicyAnalysisCLI(t, "impact", "--repo", repo.root, "--base", repo.base, "--revision", repo.failing, "--analyze-policy-failures")
	if code != 1 {
		t.Fatalf("diagnostic impact exit=%d stderr=%s output=%s, want useful output with failing candidate status", code, stderr, output)
	}
	impact := decodeYAML[host.Impact](t, output)
	if impact.Analysis == nil || impact.Analysis.Base == nil || impact.Analysis.Base.PolicyStatus != "passed" || impact.Analysis.Candidate.PolicyStatus != "failed" {
		t.Fatalf("impact did not distinguish base and candidate policy state: %#v", impact.Analysis)
	}
	if impact.DirectPolicySubjectCount == nil || *impact.DirectPolicySubjectCount != 1 || len(impact.PolicyChanges) != 1 || impact.PolicyChanges[0].Base.Status != "not-applicable" || impact.PolicyChanges[0].Candidate.Status != "failed" {
		t.Fatalf("impact did not expose the direct result transition: %#v", impact)
	}

	// The comparison remains nonzero when the base failed and the candidate
	// repaired it; each side still states its own policy status.
	code, output, stderr = invokePolicyAnalysisCLI(t, "impact", "--repo", repo.root, "--base", repo.failing, "--revision", repo.base, "--analyze-policy-failures")
	if code != 1 {
		t.Fatalf("repaired-candidate impact exit=%d stderr=%s output=%s, want base-failure exit 1", code, stderr, output)
	}
	reverse := decodeYAML[host.Impact](t, output)
	if reverse.Analysis == nil || reverse.Analysis.Base == nil || reverse.Analysis.Base.PolicyStatus != "failed" || reverse.Analysis.Candidate.PolicyStatus != "passed" {
		t.Fatalf("reverse impact did not distinguish failed base and passing candidate: %#v", reverse.Analysis)
	}
}

func TestPolicyFailingWorkingTreeRejectsReconciliationAndWritersWithoutMutation(t *testing.T) {
	repo := newPolicyAnalysisCLIRepo(t)

	// Make both writer paths observably capable of changes if they are ever
	// reached: render has an enabled projection target and Domain YAML has
	// noncanonical trailing whitespace for authoring.
	projectPath := filepath.Join(repo.root, "markitect.yaml")
	projectBytes, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	projectBytes = bytes.Replace(projectBytes, []byte("domains: [domain.yaml]"), []byte("domains: [domain.yaml]\n  targets: [markdown]"), 1)
	writeRepoFile(t, repo.root, "markitect.yaml", projectBytes)
	domainPath := filepath.Join(repo.root, "domain.yaml")
	domainBytes, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo.root, "domain.yaml", append(domainBytes, '\n'))

	before, err := host.Load(repo.root, "")
	if err != nil {
		t.Fatal(err)
	}
	beforeDigest := before.Snapshot.Digest()
	beforeHead := git(t, repo.root, "rev-parse", "HEAD")
	beforeStatus := git(t, repo.root, "status", "--porcelain")

	commands := [][]string{
		{"reconcile", "--repo", repo.root, "--action", "observe", "--adapter", "markitect-render"},
		{"reconcile", "--repo", repo.root, "--action", "plan", "--adapter", "markitect-render"},
		{"reconcile", "--repo", repo.root, "--action", "verify", "--adapter", "markitect-render", "--plan", "does-not-exist.yaml"},
		{"reconcile", "--repo", repo.root, "--action", "apply", "--adapter", "markitect-render", "--plan", "does-not-exist.yaml", "--write"},
		{"render", "--repo", repo.root, "--write"},
		{"format", "--repo", repo.root, "--write"},
	}
	for _, args := range commands {
		name := args[0]
		if args[0] == "reconcile" {
			name += "-" + args[4]
		}
		t.Run(name, func(t *testing.T) {
			code, output, stderr := invokePolicyAnalysisCLI(t, args...)
			if code != 1 {
				t.Fatalf("%v exit=%d stderr=%s output=%s, want policy-failure rejection before operation", args, code, stderr, output)
			}
			blocked := decodeYAML[report](t, output)
			if blocked.Status != "failed" || len(blocked.PolicyResults) == 0 {
				t.Fatalf("%v did not report the failing policy before operation: %#v", args, blocked)
			}
		})
	}

	after, err := host.Load(repo.root, "")
	if err != nil {
		t.Fatal(err)
	}
	if after.Snapshot.Digest() != beforeDigest || git(t, repo.root, "rev-parse", "HEAD") != beforeHead || git(t, repo.root, "status", "--porcelain") != beforeStatus {
		t.Fatalf("a reconciliation or writer command changed the policy-failing tree: before digest/status/head=%s/%q/%s after=%s/%q/%s",
			beforeDigest, beforeStatus, beforeHead, after.Snapshot.Digest(), git(t, repo.root, "status", "--porcelain"), git(t, repo.root, "rev-parse", "HEAD"))
	}
}

func TestPolicyFailureAnalysisBlocksStructuralDiagnostics(t *testing.T) {
	repo := newPolicyAnalysisCLIRepo(t)
	writeRepoFile(t, repo.root, "resources/legacy.yaml", []byte("apiVersion: report.example.org/v1\nkind: Module\nmetadata: {name: legacy, namespace: engineering}\nspec: {state: 42}\n"))
	git(t, repo.root, "add", "resources/legacy.yaml")
	git(t, repo.root, "commit", "-m", "break resource schema")
	structurallyInvalid := git(t, repo.root, "rev-parse", "HEAD")

	code, output, stderr := invokePolicyAnalysisCLI(t, "context", "--repo", repo.root, "--revision", structurallyInvalid, "--api-version", "report.example.org/v1", "--kind", "Module", "--name", "legacy", "--namespace", "engineering", "--analyze-policy-failures")
	if code != 1 {
		t.Fatalf("structural context analysis exit=%d stderr=%s output=%s, want hard failure", code, stderr, output)
	}
	blocked := decodeYAML[report](t, output)
	if blocked.Status != "failed" || len(blocked.Diagnostics) == 0 {
		t.Fatalf("structural diagnostic was not reported: %#v", blocked)
	}
	foundParse := false
	for _, diagnostic := range blocked.Diagnostics {
		if diagnostic.Code == "parse" {
			foundParse = true
			break
		}
	}
	if !foundParse {
		t.Fatalf("expected structural parse diagnostic, got %#v", blocked.Diagnostics)
	}

	// A structurally invalid candidate still returns only the ordinary failed
	// report. It must not emit analysis evidence.
	code, output, stderr = invokePolicyAnalysisCLI(t, "impact", "--repo", repo.root, "--base", repo.base, "--revision", structurallyInvalid, "--analyze-policy-failures")
	if code != 1 {
		t.Fatalf("structural candidate impact exit=%d stderr=%s output=%s, want hard diagnostic failure", code, stderr, output)
	}
	impactReport := decodeYAML[report](t, output)
	if impactReport.Status != "failed" || len(impactReport.Diagnostics) == 0 {
		t.Fatalf("structural candidate impact returned an analysis payload instead of diagnostics: %#v", impactReport)
	}

	// A structurally invalid base cannot participate in the comparison either.
	code, output, stderr = invokePolicyAnalysisCLI(t, "impact", "--repo", repo.root, "--base", structurallyInvalid, "--revision", repo.failing, "--analyze-policy-failures")
	if code != 2 || output != "" || !strings.Contains(stderr, "base has structural diagnostics") {
		t.Fatalf("structural base impact was not blocked cleanly: exit=%d stderr=%q output=%q", code, stderr, output)
	}
}

func invokePolicyAnalysisCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	binary := os.Getenv("MARKITECT_POLICY_ANALYSIS_BINARY")
	if binary == "" {
		return invoke(args...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	runErr := command.Run()
	if ctx.Err() != nil {
		t.Fatalf("packaged Markitect command exceeded 45-second timeout: %s %v\nstdout:\n%s\nstderr:\n%s", binary, args, stdout.String(), stderr.String())
	}
	if runErr == nil {
		return 0, stdout.String(), stderr.String()
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		return exitErr.ExitCode(), stdout.String(), stderr.String()
	}
	t.Fatalf("could not run packaged Markitect command %s: %v\nstdout:\n%s\nstderr:\n%s", binary, runErr, stdout.String(), stderr.String())
	return 0, stdout.String(), stderr.String()
}

type policyAnalysisCLIRepo struct {
	root    string
	base    string
	failing string
}

func newPolicyAnalysisCLIRepo(t *testing.T) policyAnalysisCLIRepo {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init", "-b", "work")
	git(t, root, "config", "user.name", "Markitect Test")
	git(t, root, "config", "user.email", "markitect-test@example.invalid")
	writeRepoFile(t, root, "markitect.yaml", []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: policy-analysis}
spec:
  areas: [{name: engineering, path: resources}]
  domains: [domain.yaml]
  checks:
    - name: should-not-run-before-policy
      run: [markitect-policy-analysis-test-command-must-not-exist]
`))
	writeRepoFile(t, root, "domain.yaml", []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: report}
spec:
  apiVersion: report.example.org/v1
  kinds:
    Module:
      properties: {state: {type: string}}
      required: [state]
`))
	writeRepoFile(t, root, "resources/legacy.yaml", []byte("apiVersion: report.example.org/v1\nkind: Module\nmetadata: {name: legacy, namespace: engineering}\nspec: {state: legacy}\n"))
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "valid architecture baseline")
	base := git(t, root, "rev-parse", "HEAD")
	writeRepoFile(t, root, "domain.yaml", []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: report}
spec:
  apiVersion: report.example.org/v1
  kinds:
    Module:
      properties: {state: {type: string}}
      required: [state]
  constraints:
    - name: active-module
      select: {kind: Module}
      assert: {op: equal, field: state, value: active}
`))
	git(t, root, "add", "domain.yaml")
	git(t, root, "commit", "-m", "require active modules")
	return policyAnalysisCLIRepo{root: root, base: base, failing: git(t, root, "rev-parse", "HEAD")}
}
