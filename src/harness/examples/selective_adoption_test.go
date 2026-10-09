package examples

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/examples/selective-adoption/pathspell"
	"go.yaml.in/yaml/v3"
)

const selectiveCommit = "93181bb9bc1af0e663b3daa1a4ff772822320307"

type handoffPreview struct {
	RawOutput string `yaml:"-"`
	Status    string `yaml:"status"`
	Handoff   struct {
		ID              string `yaml:"id"`
		Digest          string `yaml:"digest"`
		SelectionDigest string `yaml:"selectionDigest"`
		Repositories    []struct {
			ID             string `yaml:"id"`
			Commit         string `yaml:"commit"`
			SnapshotDigest string `yaml:"snapshotDigest"`
			Files          []struct {
				Path   string `yaml:"path"`
				Digest string `yaml:"digest"`
			} `yaml:"files"`
		} `yaml:"repositories"`
	} `yaml:"handoff"`
}

type copyMeReport struct {
	HandoffIdentity string `yaml:"handoffIdentity"`
	Evidence        []struct {
		ID        string   `yaml:"id"`
		Conflicts []string `yaml:"conflicts"`
	} `yaml:"evidence"`
	Candidates []struct {
		Record struct {
			StableID        string   `yaml:"stableID"`
			Support         []string `yaml:"support"`
			Counterexamples []string `yaml:"counterexamples"`
			Qualifies       []string `yaml:"qualifies"`
			Uncertainty     []string `yaml:"uncertainty"`
		} `yaml:"record"`
	} `yaml:"candidates"`
	Coverage []struct {
		ID    string `yaml:"id"`
		State string `yaml:"state"`
	} `yaml:"coverage"`
	Decision struct {
		ID       string `yaml:"id"`
		Status   string `yaml:"status"`
		Reviewer string `yaml:"reviewer"`
	} `yaml:"decision"`
	UnauthenticatedReviewer bool `yaml:"unauthenticatedReviewer"`
	Adopted                 bool `yaml:"adopted"`
}

func TestSelectiveAdoptionCLI(t *testing.T) {
	root := selectiveRepositoryRoot(t)
	binary := buildSelectiveCLI(t, root)
	temp := canonicalSelectiveTemp(t)
	repo := filepath.Join(temp, "source-repo")
	if err := os.MkdirAll(filepath.Join(repo, "docs", "validation"), 0755); err != nil {
		t.Fatal(err)
	}
	repo = canonicalSelectivePath(t, repo)
	writeSelectiveFile(t, repo, "docs/architecture.md", "# Architecture\nA rule is supported by this exact public document.\n")
	writeSelectiveFile(t, repo, "docs/validation/public-report.md", "# Report\nA counterexample qualifies the proposed rule.\n")
	writeSelectiveFile(t, repo, "private/secret.md", "DO NOT CAPTURE: private fixture sentinel 67caeec9\n")
	writeSelectiveFile(t, repo, "docs/unselected.md", "Unselected content must remain outside the handoff.\n")
	initialCommit := initSelectiveGit(t, repo)
	initialSourceState := gitSelective(t, repo, "status", "--porcelain")
	if initialSourceState != "" {
		t.Fatalf("fixture source is dirty before preview: %s", initialSourceState)
	}

	externalParent := filepath.Join(temp, "external")
	if err := os.Mkdir(externalParent, 0755); err != nil {
		t.Fatal(err)
	}
	externalParent = canonicalSelectivePath(t, externalParent)
	handoffDir := filepath.Join(externalParent, "handoff-initial")
	scopePath := filepath.Join(temp, "scope.yaml")
	writeSelectiveScope(t, scopePath, repo, initialCommit)
	preview1 := runPreparePreview(t, binary, scopePath, handoffDir)
	if preview1.Status != "planned" || preview1.Handoff.ID != "selective-adoption-fixture" {
		t.Fatalf("unexpected preview: %#v", preview1)
	}
	if strings.Contains(preview1.RawOutput, "private fixture sentinel") || strings.Contains(preview1.RawOutput, "Unselected content must remain") {
		t.Fatal("preview output disclosed excluded or unselected source content")
	}
	if _, err := os.Lstat(handoffDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read-only preview created external destination: %v", err)
	}
	if got, want := pathsFromPreview(preview1), []string{"docs/architecture.md", "docs/validation/public-report.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected files = %v, want %v", got, want)
	}
	if got := gitSelective(t, repo, "status", "--porcelain"); got != initialSourceState {
		t.Fatalf("prepare changed source state: %q", got)
	}

	// Write only after binding the exact preview digest. The workspace contains
	// the two selected blobs and handoff manifest, never the excluded sentinel.
	runPrepareWrite(t, binary, scopePath, handoffDir, preview1.Handoff.Digest)
	workspaceBytes, err := os.ReadFile(filepath.Join(handoffDir, "handoff.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var stored handoffPreview
	if err := yaml.Unmarshal(workspaceBytes, &stored.Handoff); err != nil {
		t.Fatal(err)
	}
	if stored.Handoff.Digest != preview1.Handoff.Digest {
		t.Fatalf("stored handoff digest %q differs from preview", stored.Handoff.Digest)
	}
	workspacePaths := make([]string, 0)
	if err := filepath.WalkDir(handoffDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			rel, err := filepath.Rel(handoffDir, path)
			if err != nil {
				return err
			}
			workspacePaths = append(workspacePaths, filepath.ToSlash(rel))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sortStrings(workspacePaths)
	if got, want := workspacePaths, []string{"evidence/tiny/docs/architecture.md", "evidence/tiny/docs/validation/public-report.md", "handoff.yaml"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("workspace files = %v, want %v", got, want)
	}
	for _, p := range workspacePaths {
		if strings.Contains(string(mustRead(t, filepath.Join(handoffDir, filepath.FromSlash(p)))), "private fixture sentinel") {
			t.Fatalf("workspace contains excluded content at %s", p)
		}
	}

	architectureDigest := preview1.Handoff.Repositories[0].Files[0].Digest
	reportDigest := preview1.Handoff.Repositories[0].Files[1].Digest
	queueBytes, candidateBytes, decisionBytes := selectiveRecords(t, workspaceBytes, preview1.Handoff.Digest, architectureDigest, reportDigest)
	queuePath := filepath.Join(temp, "queue.yaml")
	if err := os.MkdirAll(filepath.Join(temp, "candidates"), 0755); err != nil {
		t.Fatal(err)
	}
	candidatePath := filepath.Join(temp, "candidates", "rule.yaml")
	decisionPath := filepath.Join(temp, "decision.yaml")
	writeSelectiveBytes(t, queuePath, queueBytes)
	writeSelectiveBytes(t, candidatePath, candidateBytes)
	writeSelectiveBytes(t, decisionPath, decisionBytes)
	copyOutput := runSelectiveCLI(t, binary, "copy-me", "--workspace", handoffDir, "--queue", queuePath, "--decision", decisionPath)
	var report copyMeReport
	if err := yaml.Unmarshal(copyOutput, &report); err != nil {
		t.Fatalf("decode Copy Me report: %v\n%s", err, copyOutput)
	}
	if report.HandoffIdentity != preview1.Handoff.Digest || report.Adopted || !report.UnauthenticatedReviewer {
		t.Fatalf("report overstates decision/adoption: %#v", report)
	}
	if report.Decision.ID != "review-001" || report.Decision.Status != "defer" || report.Decision.Reviewer != "example-reviewer" {
		t.Fatalf("decision not retained: %#v", report.Decision)
	}
	if len(report.Evidence) != 3 || report.Evidence[0].ID != "support" || report.Evidence[1].ID != "counterexample" || !reflect.DeepEqual(report.Evidence[0].Conflicts, []string{"counterexample"}) || !reflect.DeepEqual(report.Evidence[1].Conflicts, []string{"support"}) || report.Evidence[2].ID != "qualification" {
		t.Fatalf("conflicting evidence links not preserved: %#v", report.Evidence)
	}
	if len(report.Candidates) != 1 || report.Candidates[0].Record.StableID != "candidate-ordering" || !reflect.DeepEqual(report.Candidates[0].Record.Support, []string{"support"}) || !reflect.DeepEqual(report.Candidates[0].Record.Counterexamples, []string{"counterexample"}) || !reflect.DeepEqual(report.Candidates[0].Record.Qualifies, []string{"qualification"}) || !reflect.DeepEqual(report.Candidates[0].Record.Uncertainty, []string{"The selected sample may not represent other modules."}) {
		t.Fatalf("candidate qualification/uncertainty lost: %#v", report.Candidates)
	}
	if len(report.Coverage) != 1 || report.Coverage[0].ID != "question-1" || report.Coverage[0].State != "examined" {
		t.Fatalf("coverage lost: %#v", report.Coverage)
	}

	// Raw candidate/queue/handoff bytes bind the optional decision. A changed
	// candidate cannot reuse a previously validated reviewer record.
	changedCandidate := bytes.Replace(candidateBytes, []byte("Related public workflow documents"), []byte("Unconditionally adopt related workflow documents"), 1)
	writeSelectiveBytes(t, candidatePath, changedCandidate)
	if output, err := runSelectiveCLIExpectFailure(binary, "copy-me", "--workspace", handoffDir, "--queue", queuePath, "--decision", decisionPath); err == nil || !(strings.Contains(string(output), "decision") || strings.Contains(string(output), "digest")) {
		t.Fatalf("stale candidate decision was accepted: %v\n%s", err, output)
	}

	// An unrelated commit changes the source revision identity, but not the
	// selected-only content snapshot. New handoff identity requires new review.
	writeSelectiveFile(t, repo, "docs/new-unselected-note.md", "A later, unselected file.\n")
	unselectedCommit := commitSelective(t, repo, "Add an unrelated file")
	writeSelectiveScope(t, scopePath, repo, unselectedCommit)
	preview2 := runPreparePreview(t, binary, scopePath, filepath.Join(externalParent, "handoff-unselected"))
	if preview2.Handoff.Repositories[0].SnapshotDigest != preview1.Handoff.Repositories[0].SnapshotDigest {
		t.Fatal("unselected-only commit changed selected snapshot digest")
	}
	if preview2.Handoff.SelectionDigest == preview1.Handoff.SelectionDigest || preview2.Handoff.Digest == preview1.Handoff.Digest {
		t.Fatal("new source revision did not create a new selection/handoff identity")
	}

	// A selected-byte change requires another preview and invalidates the saved
	// expected digest before any external workspace can be created.
	writeSelectiveFile(t, repo, "docs/architecture.md", "# Architecture\nThe selected fact has changed.\n")
	selectedCommit := commitSelective(t, repo, "Change a selected fact")
	writeSelectiveScope(t, scopePath, repo, selectedCommit)
	preview3 := runPreparePreview(t, binary, scopePath, filepath.Join(externalParent, "handoff-stale"))
	if preview3.Handoff.Repositories[0].SnapshotDigest == preview2.Handoff.Repositories[0].SnapshotDigest {
		t.Fatal("selected-byte change did not change the selected snapshot digest")
	}
	staleDest := filepath.Join(externalParent, "must-not-be-created")
	staleOutput, staleErr := runSelectiveCLIExpectFailure(binary, "prepare", "--scope", scopePath, "--output", staleDest, "--write", "--expect", preview2.Handoff.Digest)
	if staleErr == nil || !strings.Contains(string(staleOutput), "stale adoption preview") {
		t.Fatalf("stale preview digest accepted: %v\n%s", staleErr, staleOutput)
	}
	if _, err := os.Lstat(staleDest); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale write created destination: %v", err)
	}
	if got := gitSelective(t, repo, "status", "--porcelain"); got != "" {
		t.Fatalf("prepare/copy-me altered source checkout: %s", got)
	}
}

func selectiveRepositoryRoot(t *testing.T) string {
	t.Helper()
	return harnessRepositoryRoot(t)
}

func canonicalSelectiveTemp(t *testing.T) string {
	t.Helper()
	return canonicalSelectivePath(t, t.TempDir())
}

func canonicalSelectivePath(t *testing.T, value string) string {
	t.Helper()
	path, err := pathspell.Canonical(value)
	if err != nil {
		t.Fatalf("resolve temporary fixture path: %v", err)
	}
	return path
}

func buildSelectiveCLI(t *testing.T, root string) string {
	t.Helper()
	if configured := os.Getenv("MARKITECT_ADOPTION_BINARY"); configured != "" {
		binary, err := filepath.Abs(configured)
		if err != nil {
			t.Fatalf("resolve MARKITECT_ADOPTION_BINARY: %v", err)
		}
		info, err := os.Stat(binary)
		if err != nil || info.IsDir() {
			t.Fatalf("MARKITECT_ADOPTION_BINARY must name a built Markitect executable: %s", binary)
		}
		return binary
	}
	name := "markitect"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("go", "build", "-o", binary, "./src/cmd/markitect")
	cmd.Dir = harnessRepositoryRoot(t)
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build Markitect CLI: %v\n%s", err, output)
	}
	return binary
}

func writeSelectiveScope(t *testing.T, path, repo, commit string) {
	t.Helper()
	scope := map[string]any{
		"apiVersion": "markitect.example.org/adoption-scope/v1alpha1",
		"id":         "selective-adoption-fixture", "purpose": "Test explicit public evidence selection without adopting a rule.",
		"review":    "synthetic-review-001",
		"privacy":   map[string]any{"constraints": "Only the explicitly selected public fixture docs may be captured.", "allowExcerpts": true},
		"retention": "Delete the temporary handoff at test completion.",
		"repositories": []any{map[string]any{
			"id": "tiny", "root": repo, "commit": commit,
			"paths": []any{
				map[string]any{"path": "docs/architecture.md", "reason": "Potential support for the candidate rule."},
				map[string]any{"path": "docs/validation/public-report.md", "reason": "Potential counterexample to the candidate rule."},
			},
			"exclusions": []any{map[string]any{"path": "private/secret.md", "reason": "Private sentinel is excluded from this public evidence review."}},
		}},
		"coverage": []any{map[string]any{"id": "question-1", "repository": "tiny", "question": "Do the selected documents support one consistent reusable rule?", "state": "examined", "reason": "Both explicitly selected documents were captured."}},
	}
	data, err := yaml.Marshal(scope)
	if err != nil {
		t.Fatal(err)
	}
	writeSelectiveBytes(t, path, data)
}

func selectiveRecords(t *testing.T, handoffBytes []byte, handoffDigest, supportDigest, counterDigest string) ([]byte, []byte, []byte) {
	t.Helper()
	candidate := map[string]any{
		"apiVersion": "markitect.example.org/copy-me-candidate/v1alpha1", "stableID": "candidate-ordering",
		"proposedRule": "Related public workflow documents should share a stable ordering convention.", "scope": "Only the selected synthetic fixture documents.",
		"conditions": []string{"The selected documents are representative of the explicitly stated scope."}, "classification": "unclear",
		"support": []string{"support"}, "counterexamples": []string{"counterexample"}, "qualifies": []string{"qualification"},
		"confidence": "low", "confidenceBasis": "Two selected documents with conflicting observations.",
		"alternatives": []string{"Treat each workflow document independently."}, "uncertainty": []string{"The selected sample may not represent other modules."},
		"questions": []string{"Should the candidate remain deferred?"},
	}
	candidateBytes, err := yaml.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	queue := map[string]any{
		"apiVersion": "markitect.example.org/copy-me-queue/v1alpha1",
		"evidence": []any{
			map[string]any{"id": "support", "repository": "tiny", "path": "docs/architecture.md", "sourceDigest": supportDigest, "stance": "supports", "observation": "The selected architecture note describes an ordering convention.", "excerpt": "A rule is supported by this exact public document.", "conflicts": []string{"counterexample"}},
			map[string]any{"id": "counterexample", "repository": "tiny", "path": "docs/validation/public-report.md", "sourceDigest": counterDigest, "stance": "counterexample", "observation": "The selected report describes an exception to the convention.", "excerpt": "A counterexample qualifies the proposed rule.", "conflicts": []string{"support"}},
			map[string]any{"id": "qualification", "repository": "tiny", "path": "docs/architecture.md", "sourceDigest": supportDigest, "stance": "qualifies", "observation": "The small selected sample does not support claims outside its stated scope."},
		},
		"candidates": []any{map[string]any{"stableID": "candidate-ordering", "path": "candidates/rule.yaml", "digest": rawSelectiveDigest(candidateBytes)}},
		"coverage":   []any{map[string]any{"id": "question-1", "repository": "tiny", "question": "Do the selected documents support one consistent reusable rule?", "state": "examined", "reason": "Both explicitly selected documents were captured."}},
		"requests":   []any{map[string]any{"repository": "tiny", "path": "docs/unselected.md", "reason": "It may add context to the proposal.", "insufficiency": "This path was not selected and cannot be read through a queue request."}},
	}
	queueBytes, err := yaml.Marshal(queue)
	if err != nil {
		t.Fatal(err)
	}
	decision := map[string]any{
		"apiVersion": "markitect.example.org/copy-me-decision/v1alpha1", "id": "review-001", "reviewer": "example-reviewer", "date": "2026-10-04",
		"rationale": "The conflict and limited sample require further review.", "scope": "Only the selected synthetic fixture documents.", "status": "defer", "candidateID": "candidate-ordering",
		"candidateDigest": rawSelectiveDigest(candidateBytes), "queueDigest": rawSelectiveDigest(queueBytes), "handoffDigest": rawSelectiveDigest(handoffBytes),
		"handoffIdentity": handoffDigest,
	}
	decisionBytes, err := yaml.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	return queueBytes, candidateBytes, decisionBytes
}

func runPreparePreview(t *testing.T, binary, scope, destination string) handoffPreview {
	t.Helper()
	output := runSelectiveCLI(t, binary, "prepare", "--scope", scope, "--output", destination)
	var preview handoffPreview
	if err := yaml.Unmarshal(output, &preview); err != nil {
		t.Fatalf("decode prepare preview: %v\n%s", err, output)
	}
	preview.RawOutput = string(output)
	if preview.Handoff.Digest == "" || preview.Handoff.ID == "" {
		t.Fatalf("preview missing handoff identity:\n%s", output)
	}
	return preview
}

func runPrepareWrite(t *testing.T, binary, scope, destination, digest string) {
	t.Helper()
	output := runSelectiveCLI(t, binary, "prepare", "--scope", scope, "--output", destination, "--write", "--expect", digest)
	var result struct {
		Status string `yaml:"status"`
	}
	if err := yaml.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode prepare write: %v\n%s", err, output)
	}
	if result.Status != "written" {
		t.Fatalf("write status %q, want written", result.Status)
	}
}

func runSelectiveCLI(t *testing.T, binary string, args ...string) []byte {
	t.Helper()
	output, err := exec.Command(binary, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("markitect %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return output
}

func runSelectiveCLIExpectFailure(binary string, args ...string) ([]byte, error) {
	return exec.Command(binary, args...).CombinedOutput()
}

func initSelectiveGit(t *testing.T, root string) string {
	t.Helper()
	gitSelective(t, root, "init", "-b", "fixture")
	gitSelective(t, root, "config", "user.name", "Selective Adoption Fixture")
	gitSelective(t, root, "config", "user.email", "fixture@example.invalid")
	return commitSelective(t, root, "Freeze explicit evidence fixture")
}

func commitSelective(t *testing.T, root, message string) string {
	t.Helper()
	gitSelective(t, root, "add", "--all")
	gitSelective(t, root, "commit", "--allow-empty", "-m", message)
	return gitSelective(t, root, "rev-parse", "HEAD")
}

func gitSelective(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func writeSelectiveFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	writeSelectiveBytes(t, path, []byte(contents))
}

func writeSelectiveBytes(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
}

func pathsFromPreview(preview handoffPreview) []string {
	paths := make([]string, 0)
	if len(preview.Handoff.Repositories) == 0 {
		return paths
	}
	for _, file := range preview.Handoff.Repositories[0].Files {
		paths = append(paths, file.Path)
	}
	sortStrings(paths)
	return paths
}

func sortStrings(values []string) { // local helper keeps fixture independent of internal packages
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func rawSelectiveDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
