package examples

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
)

func TestEngineeringDiscoveryProjectKeepsDossierOutsideCanonicalGraph(t *testing.T) {
	root := filepath.Join(exampleRepoRoot(t), "examples", "engineering-discovery")
	project, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("discovery example has graph diagnostics: %#v", project.Diagnostics)
	}
	if len(project.Graph.Resources) != 3 {
		t.Fatalf("canonical discovery Project resources = %d, want Project, Skill and Workflow only: %v", len(project.Graph.Resources), project.Graph.Resources)
	}
	for key := range project.Graph.Resources {
		if strings.Contains(key, "candidate") || strings.Contains(key, "evidence") || strings.Contains(key, "decision") {
			t.Errorf("staging dossier unexpectedly entered the canonical graph: %s", key)
		}
	}
}

func TestEngineeringDiscoveryDossierBindsEvidenceAndCandidateToDecision(t *testing.T) {
	moduleRoot := exampleRepoRoot(t)
	projectRoot := filepath.Join(t.TempDir(), "project")
	copyTree(t, filepath.Join(moduleRoot, "examples", "engineering-discovery", "project-template"), projectRoot)
	git(t, projectRoot, "init", "-q")
	git(t, projectRoot, "add", "--all")
	git(t, projectRoot, "-c", "user.name=Discovery Example", "-c", "user.email=example@invalid", "commit", "-m", "freeze selected evidence")
	revision := strings.TrimSpace(git(t, projectRoot, "rev-parse", "HEAD"))

	project, err := app.Load(projectRoot, revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("fixed synthetic Project has graph diagnostics: %#v", project.Diagnostics)
	}
	manifest := project.Snapshot.Files["context-run.yaml"]
	context, err := app.CompileRunContext(project, "context-run.yaml", manifest, "engineering-discovery-example", "synthetic-workflow")
	if err != nil {
		t.Fatal(err)
	}
	if !context.Complete || context.Run == nil {
		t.Fatalf("synthetic ContextRun is incomplete: %#v", context.Run)
	}

	dossierRoot := filepath.Join(t.TempDir(), "dossier")
	if err := os.MkdirAll(dossierRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	candidatePath := filepath.Join(moduleRoot, "examples", "engineering-discovery", "dossier-template", "candidate.md")
	candidateBytes, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(dossierRoot, "candidate.md")
	if err := os.WriteFile(candidate, candidateBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	sources := map[string]struct {
		id, stance, excerpt, note string
	}{
		"src/Refund/Handler.cs":                {"E-01", "supports", "public sealed class Handler", "Selected recent use-case structure; observation only."},
		"src/Services/RefundService.cs":        {"E-02", "counterexample", "public sealed class RefundService", "Older service-centric structure; retain as contrary evidence."},
		"docs/decisions/service-transition.md": {"E-03", "qualifies", "The older RefundService remains while existing callers migrate.", "Migration context does not settle preferred future policy."},
	}
	var sourceRows strings.Builder
	for _, input := range context.Inputs {
		if input.Role != "source" {
			continue
		}
		item, ok := sources[input.Path]
		if !ok {
			t.Fatalf("unexpected selected source %q", input.Path)
		}
		fmt.Fprintf(&sourceRows, "  - id: %s\n    stance: %s\n    path: %q\n    hash: %q\n    excerpt: %q\n    note: %q\n", item.id, item.stance, input.Path, input.Hash, item.excerpt, item.note)
	}
	evidenceBytes := []byte(fmt.Sprintf(
		"version: markitect.example.org/engineering-discovery-evidence/v1alpha1\nrepository: local:synthetic-engineering-project\nrevision: %q\nsnapshotDigest: %q\nentry: %q\ntoolVersion: %q\ntoolDigest: %q\nrun:\n  path: %q\n  manifestHash: %q\n  selectionHash: %q\n  contextDigest: %q\nsources:\n%s",
		project.Snapshot.ID, project.Snapshot.Digest(), context.Entry, "engineering-discovery-example", "synthetic-workflow", context.Run.ManifestPath,
		context.Run.ManifestHash, context.Run.SelectionHash, context.Digest, sourceRows.String(),
	))
	evidence := filepath.Join(dossierRoot, "evidence.yaml")
	if err := os.WriteFile(evidence, evidenceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	decisionBytes := []byte(fmt.Sprintf(
		"version: markitect.example.org/engineering-discovery-decision/v1alpha1\nstatus: accepted\nreviewer: synthetic-fixture-only\ndecidedAt: 2026-10-02\nrationale: Synthetic test input; no real review or approval occurred.\ncandidateHash: %q\nevidenceHash: %q\n",
		app.Hash(candidateBytes), app.Hash(evidenceBytes),
	))
	decision := filepath.Join(dossierRoot, "decision.yaml")
	if err := os.WriteFile(decision, decisionBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runDiscoveryChecker(moduleRoot, projectRoot, revision, evidence, candidate, decision)
	if err != nil {
		t.Fatalf("valid synthetic dossier rejected: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	for _, required := range []string{"3 selected evidence items", "accepted", "not authenticated", "no canonical resource or policy was adopted"} {
		if !strings.Contains(stdout, required) {
			t.Errorf("checker output omitted boundary %q: %s", required, stdout)
		}
	}
	oversizedEvidence := filepath.Join(dossierRoot, "oversized-evidence.yaml")
	if err := os.WriteFile(oversizedEvidence, bytes.Repeat([]byte("x"), (1<<20)+1), 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, err = runDiscoveryChecker(moduleRoot, projectRoot, revision, oversizedEvidence, candidate, decision)
	if err == nil || !strings.Contains(stderr, "between 1 byte and 1 MiB") {
		t.Fatalf("oversized dossier input was not rejected at its read bound: err=%v stderr=%s", err, stderr)
	}
	_, stderr, err = runDiscoveryChecker(moduleRoot, projectRoot, revision, evidence, dossierRoot, decision)
	if err == nil || !strings.Contains(stderr, "must be a regular file") {
		t.Fatalf("directory dossier input was not rejected as non-regular: err=%v stderr=%s", err, stderr)
	}
	alias := filepath.Join(dossierRoot, "candidate-alias.md")
	t.Run("rejects symlink aliases into Project", func(t *testing.T) {
		if err := os.Symlink(filepath.Join(projectRoot, "context-run.yaml"), alias); err != nil {
			t.Skipf("unsupportedWindowsSymlinks: %v", err)
		}
		_, stderr, err := runDiscoveryChecker(moduleRoot, projectRoot, revision, evidence, alias, decision)
		if err == nil || !strings.Contains(stderr, "outside the Project snapshot") {
			t.Fatalf("candidate symlink into the Project was not rejected: err=%v stderr=%s", err, stderr)
		}
	})

	// Rehashing a forged excerpt in a new decision cannot make it source truth.
	forgedEvidenceBytes := bytes.Replace(evidenceBytes, []byte("public sealed class Handler"), []byte("public sealed class InventedHandler"), 1)
	if bytes.Equal(forgedEvidenceBytes, evidenceBytes) {
		t.Fatal("test evidence excerpt was not found for tampering")
	}
	if err := os.WriteFile(evidence, forgedEvidenceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	forgedDecision := []byte(fmt.Sprintf(
		"version: markitect.example.org/engineering-discovery-decision/v1alpha1\nstatus: accepted\nreviewer: synthetic-fixture-only\ndecidedAt: 2026-10-02\nrationale: Synthetic test input; no real review or approval occurred.\ncandidateHash: %q\nevidenceHash: %q\n",
		app.Hash(candidateBytes), app.Hash(forgedEvidenceBytes),
	))
	if err := os.WriteFile(decision, forgedDecision, 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, err = runDiscoveryChecker(moduleRoot, projectRoot, revision, evidence, candidate, decision)
	if err == nil || !strings.Contains(stderr, "exact included ContextRun input") {
		t.Fatalf("forged excerpt passed after recomputing its hashes: err=%v stderr=%s", err, stderr)
	}
	if err := os.WriteFile(evidence, evidenceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(decision, decisionBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	// A candidate edit after the decision must fail its exact-byte binding.
	if err := os.WriteFile(candidate, append(candidateBytes, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, err = runDiscoveryChecker(moduleRoot, projectRoot, revision, evidence, candidate, decision)
	if err == nil || !strings.Contains(stderr, "decision is stale") {
		t.Fatalf("changed candidate did not invalidate the decision: err=%v stderr=%s", err, stderr)
	}

	// A new source commit changes the immutable snapshot; the old ledger cannot
	// be reused even if its prior candidate and decision files remain intact.
	if err := os.WriteFile(filepath.Join(projectRoot, "src", "Refund", "Handler.cs"), []byte("namespace Synthetic.Refund; // changed after review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, projectRoot, "add", "--all")
	git(t, projectRoot, "-c", "user.name=Discovery Example", "-c", "user.email=example@invalid", "commit", "-m", "change selected evidence")
	newRevision := strings.TrimSpace(git(t, projectRoot, "rev-parse", "HEAD"))
	_, stderr, err = runDiscoveryChecker(moduleRoot, projectRoot, newRevision, evidence, candidate, decision)
	if err == nil || !strings.Contains(stderr, "immutable revision and snapshot digest") {
		t.Fatalf("changed source snapshot did not invalidate the evidence ledger: err=%v stderr=%s", err, stderr)
	}
}

func runDiscoveryChecker(moduleRoot, projectRoot, revision, evidence, candidate, decision string) (string, string, error) {
	script := filepath.Join(moduleRoot, "examples", "engineering-discovery", "check-discovery.go")
	command := exec.Command("go", "run", script,
		"--repo", projectRoot, "--revision", revision, "--run", "context-run.yaml",
		"--evidence", evidence, "--candidate", candidate, "--decision", decision,
	)
	command.Dir = moduleRoot
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return stdout.String(), stderr.String(), err
}

func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	if err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		t.Fatalf("copy project template: %v", err)
	}
}

func git(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}

func exampleRepoRoot(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate engineering discovery test")
	}
	return filepath.Dir(filepath.Dir(sourceFile))
}
