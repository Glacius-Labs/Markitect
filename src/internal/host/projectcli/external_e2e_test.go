package projectcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"go.yaml.in/yaml/v3"
)

// TestExternalGitCLICommitsReviewedCandidate exercises the shipped CLI process:
// it exports and lists a packet, rejects stale authority, imports a committed
// Git delta (including a deletion), gathers a separate review, verifies with
// the declared check, then performs guarded Apply from its CLI preflight.
func TestExternalGitCLICommitsReviewedCandidate(t *testing.T) {
	cli := buildExternalGitCLI(t)
	_, _, repo, _ := newRuntimeContract(t, "cli", false)
	artifactPath := filepath.Join(repo.Dir, ".markitect", "model", "artifact.yaml")
	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(artifact), "paths: [src/value.txt]") || !strings.Contains(string(artifact), "required: true") {
		t.Fatal("runtime fixture's required value artifact changed unexpectedly")
	}
	obsoleteArtifact := strings.Replace(string(artifact), "name: value-file", "name: obsolete-file", 1)
	obsoleteArtifact = strings.Replace(obsoleteArtifact, "paths: [src/value.txt]", "paths: [src/obsolete.txt]", 1)
	obsoleteArtifact = strings.Replace(obsoleteArtifact, "required: true", "required: false", 1)
	repo.Write(".markitect/model/obsolete-artifact.yaml", obsoleteArtifact)
	manifestPath := filepath.Join(repo.Dir, filepath.FromSlash(projectwork.ManifestPath))
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest projectwork.Config
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.ModelFiles = append(manifest.ModelFiles, ".markitect/model/obsolete-artifact.yaml")
	manifestBytes, err = yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	repo.Write("src/obsolete.txt", "obsolete implementation\n")
	runtimeConfig, err := projectconfig.LoadRuntime(repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	managerID := contractManagerID()
	reviewer, configured := runtimeConfig.Agents[managerID]
	if !configured {
		t.Fatalf("runtime fixture has no executor for reviewer Manager %s", managerID)
	}
	runtimeConfig.Review = &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{managerID: reviewer}, MaxRounds: 1, MaxManagerRounds: 1}
	runtimeBytes, err := yaml.Marshal(runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	repo.Write(projectconfig.RuntimePath, string(runtimeBytes))
	repo.Commit("extend runtime fixture artifact for deletion")

	goal := "Implement the modeled value behavior."
	previewArgs := []string{"plan", "--goal", goal, "--manager", contractManagerID()}
	previewCode, previewBytes, previewStderr := externalGitCLICall(t, cli, repo.Dir, previewArgs...)
	if previewCode != 0 {
		t.Fatalf("public CLI plan preview exit=%d stderr=%s stdout=%s", previewCode, previewStderr, previewBytes)
	}
	preview := decodeOutput[externalGitCLIPlan](t, previewBytes)
	if preview.Status != projectrun.StatusPlanned || preview.PreviewDigest == "" {
		t.Fatalf("public CLI plan preview omitted its status/digest binding: %+v", preview)
	}
	persistArgs := append(append([]string{}, previewArgs...), "--expect", preview.PreviewDigest, "--write")
	persistedBytes := externalGitCLISuccess(t, cli, repo.Dir, persistArgs...)
	persisted := decodeOutput[externalGitCLIPlan](t, persistedBytes)
	if persisted.Status != projectrun.StatusPlanned || !persisted.ExecuteAuthorized || persisted.ID == "" || persisted.BaseProjectDigest != preview.BaseProjectDigest || persisted.WorkingProjectDigest != preview.WorkingProjectDigest || persisted.RuntimeDigest != preview.RuntimeDigest {
		t.Fatalf("public CLI did not persist its reviewed preview: preview=%+v persisted=%+v", preview, persisted)
	}
	plan := persisted.PlanRecord
	defer preserveExternalGitCLIRun(t, repo.Dir, plan.ID)
	fixed, fixedErr := projectwork.Load(repo.Dir, plan.BaseRevision)
	working, workingErr := projectwork.Load(repo.Dir, "")
	identity, identityErr := source.IdentifyGit(repo.Dir)
	branch, head := repo.Git("branch", "--show-current"), repo.Git("rev-parse", "HEAD")
	diagnostic := map[string]any{"planId": plan.ID, "planDigest": plan.Digest, "planBaseRevision": plan.BaseRevision,
		"planBaseSnapshot": plan.BaseSnapshot, "planBaseProjectDigest": plan.BaseProjectDigest,
		"planWorkingSnapshot": plan.WorkingSnapshot, "planWorkingProjectDigest": plan.WorkingProjectDigest,
		"planTargetBranch": plan.TargetBranch, "planTargetHead": plan.TargetHead,
		"planRepositoryDigest": plan.RepositoryDigest, "planRuntimeDigest": plan.RuntimeDigest,
		"cliPreviewRuntimeDigest": preview.RuntimeDigest, "cliPreviewBaseSnapshot": preview.BaseSnapshot,
		"cliPreview":                preview,
		"planRuntimeBindingVersion": plan.RuntimeBindingVersion, "branch": branch, "head": head}
	if fixedErr == nil {
		diagnostic["fixedSnapshot"] = fixed.Snapshot.Digest()
		diagnostic["fixedProjectDigest"] = fixed.Digest
		diagnostic["fixedModelDigest"] = fixed.Report.ModelDigest
	}
	if workingErr == nil {
		diagnostic["workingSnapshot"] = working.Snapshot.Digest()
		diagnostic["workingProjectDigest"] = working.Digest
	}
	if identityErr == nil {
		diagnostic["repositoryDigest"] = identity.Digest
	}
	encodedDiagnostic, _ := json.Marshal(diagnostic)
	t.Logf("pre-export binding diagnostic: %s", encodedDiagnostic)
	if fixedErr != nil || workingErr != nil || identityErr != nil || preview.BaseRevision != plan.BaseRevision || preview.BaseSnapshot != plan.BaseSnapshot || preview.WorkingSnapshot != plan.WorkingSnapshot || preview.BaseProjectDigest != plan.BaseProjectDigest || preview.WorkingProjectDigest != plan.WorkingProjectDigest || preview.BaseModelDigest != plan.BaseModelDigest || preview.ModelDigest != plan.ModelDigest || preview.ReportDigest != plan.ReportDigest || preview.RuntimeDigest != plan.RuntimeDigest || preview.RepositoryDigest != plan.RepositoryDigest || preview.TargetBranch != plan.TargetBranch || preview.TargetHead != plan.TargetHead || fixed.Snapshot.Digest() != plan.BaseSnapshot || fixed.Report.ModelDigest != plan.BaseModelDigest || working.Snapshot.Digest() != plan.WorkingSnapshot || identity.Digest != plan.RepositoryDigest || branch != plan.TargetBranch || head != plan.TargetHead || plan.RuntimeBindingVersion != 2 {
		t.Fatalf("fixture does not match its just-persisted plan: %s", encodedDiagnostic)
	}
	code, stdout, stderr := externalGitCLICall(t, cli, repo.Dir, "packet-export", plan.ID, "--execute")
	if code != 0 {
		status, statusErr := projectrun.Status(repo.Dir, plan.ID)
		evidenceDir, evidenceErr := os.MkdirTemp("", "markitect-external-e2e-evidence-")
		if evidenceErr != nil {
			t.Fatalf("packet-export exit=%d stderr=%s stdout=%s (could not persist diagnostics: %v)", code, stderr, stdout, evidenceErr)
		}
		evidence, _ := json.MarshalIndent(map[string]any{"bindings": diagnostic, "argv": []string{"packet-export", plan.ID, "--execute"}, "exitCode": code, "stdout": string(stdout), "stderr": string(stderr), "status": status, "statusError": fmt.Sprint(statusErr)}, "", "  ")
		evidencePath := filepath.Join(evidenceDir, "packet-export-failure.json")
		if writeErr := os.WriteFile(evidencePath, evidence, 0o600); writeErr != nil {
			t.Fatalf("packet-export exit=%d stderr=%s stdout=%s (could not persist diagnostics: %v)", code, stderr, stdout, writeErr)
		}
		t.Fatalf("public CLI packet-export exit=%d stderr=%s stdout=%s; binding evidence: %s", code, stderr, stdout, evidencePath)
	}
	first := decodeOutput[externalGitCLIExport](t, stdout)
	if len(first.Packets) != 1 || first.Run.Status != projectrun.StatusAwaitingExternal {
		t.Fatalf("first external CLI export did not stop at one pending packet: status=%q packets=%d", first.Run.Status, len(first.Packets))
	}
	packet := first.Packets[0]
	if packet.Kind != "projectrun-task/v1" || packet.Phase != "work" || packet.CandidateDigest == "" {
		t.Fatalf("unexpected first packet for candidate import: kind=%q phase=%q candidate=%q", packet.Kind, packet.Phase, packet.CandidateDigest)
	}

	listed := externalGitCLIList(t, cli, repo.Dir, plan.ID)
	if len(listed) != 1 || listed[0].ID != packet.ID || listed[0].Digest != packet.Digest {
		t.Fatalf("packet-list changed the exported pending packet: got=%+v want=%+v", listed, packet)
	}
	reexport := externalGitCLIPacketExport(t, cli, repo.Dir, plan.ID)
	if len(reexport.Packets) != 1 || reexport.Packets[0].ID != packet.ID || reexport.Packets[0].Digest != packet.Digest || string(reexport.Packets[0].InvocationBytes) != string(packet.InvocationBytes) {
		t.Fatal("re-export changed the pending packet identity, digest, or invocation bytes")
	}

	base, err := projectwork.Load(repo.Dir, packet.BaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	if base.Snapshot.Digest() != packet.CandidateDigest {
		t.Fatalf("fixture cannot prove the packet base: project snapshot digest=%q packet candidate digest=%q", base.Snapshot.Digest(), packet.CandidateDigest)
	}
	candidateRevision := externalGitCLICommitCandidate(t, repo, packet.BaseRevision, "src/value.txt", "valid\n", "src/obsolete.txt")
	responseBytes := externalGitCLIResponse(t, packet, true)
	responsePath := externalGitCLIWriteResponse(t, repo.Dir, plan.ID, packet.ID, responseBytes)

	staleArgs := externalGitCLISubmitArgs(plan.ID, packet, "sha256:stale", responsePath, repo.Dir, packet.BaseRevision, candidateRevision)
	staleCode, _, _ := externalGitCLICall(t, cli, repo.Dir, staleArgs...)
	if staleCode == 0 {
		t.Fatal("packet-submit accepted an incorrect expected packet digest")
	}
	stillPending := externalGitCLIList(t, cli, repo.Dir, plan.ID)
	if len(stillPending) != 1 || stillPending[0].ID != packet.ID {
		t.Fatalf("stale submit changed pending packet state: %+v", stillPending)
	}

	submitArgs := externalGitCLISubmitArgs(plan.ID, packet, packet.Digest, responsePath, repo.Dir, packet.BaseRevision, candidateRevision)
	stored := decodeOutput[struct {
		PacketID      string `json:"packetId"`
		Digest        string `json:"digest"`
		Stored        bool   `json:"stored"`
		NewSubmission bool   `json:"newSubmission"`
	}](t, externalGitCLISuccess(t, cli, repo.Dir, submitArgs...))
	if stored.PacketID != packet.ID || stored.Digest != packet.Digest || !stored.Stored || !stored.NewSubmission {
		t.Fatalf("packet-submit did not durably store the candidate once: %+v", stored)
	}
	duplicate := decodeOutput[struct {
		PacketID      string `json:"packetId"`
		Digest        string `json:"digest"`
		Stored        bool   `json:"stored"`
		NewSubmission bool   `json:"newSubmission"`
	}](t, externalGitCLISuccess(t, cli, repo.Dir, submitArgs...))
	if duplicate.PacketID != packet.ID || duplicate.Digest != packet.Digest || !duplicate.Stored || duplicate.NewSubmission {
		t.Fatalf("exact duplicate submit was not idempotent: %+v", duplicate)
	}

	candidateUsed := true
	var completed externalGitCLIExport
	for attempt := 0; attempt < 16; attempt++ {
		current := externalGitCLIPacketExport(t, cli, repo.Dir, plan.ID)
		if len(current.Packets) == 0 {
			if current.Run.Status != projectrun.StatusIntegrated {
				t.Fatalf("external CLI stopped without a pending packet or integrated run: status=%q", current.Run.Status)
			}
			completed = current
			break
		}
		if len(current.Packets) != 1 {
			t.Fatalf("expected serial packet progression, found %d packets", len(current.Packets))
		}
		next := current.Packets[0]
		response := externalGitCLIResponse(t, next, false)
		path := externalGitCLIWriteResponse(t, repo.Dir, plan.ID, next.ID, response)
		args := externalGitCLISubmitArgs(plan.ID, next, next.Digest, path, "", "", "")
		if !candidateUsed && next.Kind == "projectrun-task/v1" && next.Phase == "work" {
			t.Fatal("candidate Git delta was unexpectedly deferred past the initial work packet")
		}
		externalGitCLISuccess(t, cli, repo.Dir, args...)
	}
	if completed.Run.ID == "" {
		t.Fatal("external CLI did not finish the run within the packet bound")
	}
	if !candidateUsed {
		t.Fatal("no packet received the committed Git candidate")
	}
	if !hasPassingExternalReview(completed.Run.Reviews) {
		t.Fatalf("integrated run has no independent passing Manager review: %+v", completed.Run.Reviews)
	}

	verification := decodeOutput[projectrun.VerifyReport](t, externalGitCLISuccess(t, cli, repo.Dir, "verify", plan.ID, "--execute"))
	if verification.Status != "verified" || verification.Digest == "" || verification.CandidateID != completed.Run.Candidate.ID || len(verification.Checks) == 0 || verification.Checks[0].Outcome != "passed" {
		t.Fatalf("ordinary CLI verification did not pass the exact integrated candidate: %+v", verification)
	}
	preflight := decodeOutput[projectrun.ApplyPreflight](t, externalGitCLISuccess(t, cli, repo.Dir,
		"apply", "--plan", plan.ID, "--run", plan.ID, "--candidate", completed.Run.Candidate.ID))
	if preflight.VerificationDigest != verification.Digest || preflight.CandidateID != verification.CandidateID || len(preflight.Paths) != 2 {
		t.Fatalf("guarded Apply preflight did not bind the verified candidate paths: %+v", preflight)
	}
	applied := decodeOutput[projectrun.ApplyReport](t, externalGitCLISuccess(t, cli, repo.Dir,
		"apply", "--plan", plan.ID, "--run", plan.ID, "--candidate", completed.Run.Candidate.ID,
		"--branch", preflight.TargetBranch, "--head", preflight.ExpectedHead,
		"--worktree", preflight.ExpectedWorktree, "--expect", preflight.VerificationDigest, "--write"))
	if applied.Status != "applied" || applied.CandidateID != completed.Run.Candidate.ID {
		t.Fatalf("guarded CLI Apply did not adopt the verified candidate: %+v", applied)
	}
	value, err := os.ReadFile(filepath.Join(repo.Dir, "src", "value.txt"))
	if err != nil || string(value) != "valid\n" {
		t.Fatalf("Apply did not materialize the externally observed modification: content=%q err=%v", value, err)
	}
	if _, err := os.Lstat(filepath.Join(repo.Dir, "src", "obsolete.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Apply did not materialize the externally observed deletion: err=%v", err)
	}
}

type externalGitCLIExport struct {
	Run     projectrun.RunReport        `json:"run"`
	Packets []projectrun.ExternalPacket `json:"packets"`
}

type externalGitCLIPlan struct {
	projectrun.PlanRecord
	PreviewDigest string `json:"previewDigest"`
}

func buildExternalGitCLI(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve external CLI test source path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	name := "markitect"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("go", "build", "-o", binary, "./src/cmd/markitect")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build public Markitect CLI: %v\n%s", err, output)
	}
	return binary
}

func externalGitCLICall(t *testing.T, binary, root string, args ...string) (int, []byte, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	argv := append(append([]string{}, args...), "--repo", root)
	cmd := exec.CommandContext(ctx, binary, argv...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err := recordExternalGitCLIInvocation(root, binary, argv, stdout.String(), stderr.String(), err); err != nil {
		t.Fatalf("persist public CLI evidence: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatalf("public CLI timed out for %v: %v\nstdout=%s\nstderr=%s", args, ctx.Err(), stdout.String(), stderr.String())
	}
	if err == nil {
		return 0, []byte(stdout.String()), []byte(stderr.String())
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), []byte(stdout.String()), []byte(stderr.String())
	}
	t.Fatalf("start public CLI for %v: %v", args, err)
	return -1, nil, nil
}

func recordExternalGitCLIInvocation(root, binary string, argv []string, stdout, stderr string, commandErr error) error {
	dir := strings.TrimSpace(os.Getenv("MARKITECT_RUN02_EVIDENCE_DIR"))
	if dir == "" {
		return nil
	} else {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		dir = abs
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return err
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return fmt.Errorf("MARKITECT_RUN02_EVIDENCE_DIR must be outside the selected repository")
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	code := 0
	if commandErr != nil {
		code = -1
		var exit *exec.ExitError
		if errors.As(commandErr, &exit) {
			code = exit.ExitCode()
		}
	}
	data, err := json.MarshalIndent(map[string]any{
		"binary": binary, "cwd": root, "argv": argv, "exitCode": code,
		"stdout": stdout, "stderr": stderr,
	}, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, "cli-invocation-*.json")
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Sync()
}

func preserveExternalGitCLIRun(t *testing.T, root, runID string) {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("MARKITECT_RUN02_EVIDENCE_DIR"))
	if dir == "" {
		return
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Errorf("resolve MARKITECT_RUN02_EVIDENCE_DIR: %v", err)
		return
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		t.Errorf("MARKITECT_RUN02_EVIDENCE_DIR must be outside selected repository: %q", dir)
		return
	}
	sourceDir := filepath.Join(root, ".markitect", "runs", runID)
	if _, err := os.Stat(sourceDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("inspect runtime evidence: %v", err)
		}
		return
	}
	destination := filepath.Join(dir, "run-"+runID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Errorf("create runtime evidence directory: %v", err)
		return
	}
	if _, err := os.Stat(destination); err == nil {
		t.Errorf("runtime evidence destination already exists: %s", destination)
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("inspect runtime evidence destination: %v", err)
		return
	}
	if err := filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		target := destination
		if relative != "." {
			target = filepath.Join(destination, relative)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("runtime evidence contains non-regular file %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	}); err != nil {
		t.Errorf("preserve runtime evidence at %s: %v", destination, err)
	}
}

func externalGitCLISuccess(t *testing.T, binary, root string, args ...string) []byte {
	t.Helper()
	code, stdout, stderr := externalGitCLICall(t, binary, root, args...)
	if code != 0 {
		t.Fatalf("public CLI %v: exit=%d\nstderr=%s\nstdout=%s", args, code, stderr, stdout)
	}
	return stdout
}

func externalGitCLIPacketExport(t *testing.T, binary, root, runID string) externalGitCLIExport {
	t.Helper()
	data := externalGitCLISuccess(t, binary, root, "packet-export", runID, "--execute")
	return decodeOutput[externalGitCLIExport](t, data)
}

func externalGitCLIList(t *testing.T, binary, root, runID string) []projectrun.ExternalPacket {
	t.Helper()
	return decodeOutput[[]projectrun.ExternalPacket](t, externalGitCLISuccess(t, binary, root, "packet-list", runID))
}

func externalGitCLISubmitArgs(runID string, packet projectrun.ExternalPacket, expected, responsePath, candidateRoot, baseRevision, candidateRevision string) []string {
	args := []string{"packet-submit", runID, "--packet", packet.ID, "--expect", expected, "--response", responsePath, "--execute"}
	if candidateRoot != "" {
		args = append(args, "--candidate-root", candidateRoot, "--base-revision", baseRevision, "--candidate-revision", candidateRevision)
	}
	return args
}

func externalGitCLIResponse(t *testing.T, packet projectrun.ExternalPacket, candidate bool) []byte {
	t.Helper()
	invocation := packet.Invocation
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest, CandidateFiles: []agentexec.CandidateFile{},
		EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}}
	if packet.Kind == "projectrun-review/v1" {
		response.Outcome = agentexec.OutcomeProposed
		response.ReportJSON = json.RawMessage(`{"status":"pass","summary":"The exact candidate satisfies the accepted artifact obligations.","findings":[]}`)
	} else {
		var requestContext struct {
			Phase string `json:"phase"`
		}
		if err := json.Unmarshal(invocation.Request.Context, &requestContext); err != nil {
			t.Fatal(err)
		}
		report := projectrun.TaskResponse{Status: "complete", Summary: "Completed the externally executed task.",
			Delegations: []projectrun.Delegation{}, ReworkRequests: []projectrun.ReworkRequest{}, Integrated: requestContext.Phase == "integrate",
			Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}, EscalateTo: ""}
		reportJSON, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		response.Outcome = agentexec.OutcomeProposed
		response.ReportJSON = reportJSON
		if candidate {
			response.CandidateFiles = []agentexec.CandidateFile{{Path: "src/value.txt", Mode: "0644", Content: "valid\n"}}
		}
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func externalGitCLIWriteResponse(t *testing.T, root, runID, packetID string, response []byte) string {
	t.Helper()
	dir := filepath.Join(root, ".markitect", "runs", runID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	relative := filepath.ToSlash(filepath.Join(".markitect", "runs", runID, "external-response-"+packetID+".json"))
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.WriteFile(path, response, 0o600); err != nil {
		t.Fatal(err)
	}
	return relative
}

func externalGitCLICommitCandidate(t *testing.T, repo interface {
	Git(...string) string
	Write(string, string)
	Commit(string) string
}, base, modifyPath, content, deletePath string) string {
	t.Helper()
	originalBranch := repo.Git("branch", "--show-current")
	branch := fmt.Sprintf("external-candidate-%d", time.Now().UnixNano())
	repo.Git("switch", "-c", branch, base)
	repo.Write(modifyPath, content)
	root := repo.Git("rev-parse", "--show-toplevel")
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(deletePath))); err != nil {
		t.Fatal(err)
	}
	candidate := repo.Commit("external Git candidate with deletion")
	repo.Git("switch", originalBranch)
	return candidate
}

func hasPassingExternalReview(reviews []projectrun.ReviewRecord) bool {
	for _, review := range reviews {
		if review.Outcome == "pass" && review.CandidateID != "" && review.InputDigest != "" && review.ScopeDigest != "" {
			return true
		}
	}
	return false
}
