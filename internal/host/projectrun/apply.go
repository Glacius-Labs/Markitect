package projectrun

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	hostwrite "github.com/Glacius-Labs/Markitect/internal/host"
)

// ApplyPaths returns the exact candidate delta paths for CLI preflight and
// CaptureTarget. It includes additions, modifications and deletions only.
func ApplyPaths(host Host, root, runID string) ([]string, error) {
	plan, run, candidate, base, err := loadApplyInputs(host, root, runID)
	_ = plan
	_ = run
	if err != nil {
		return nil, err
	}
	return candidateDeltaPaths(base, candidate), nil
}

// CaptureTarget computes a digest for the exact files Apply will compare and
// mutate. The digest binds repository identity, branch, HEAD, selected bytes,
// modes and absent paths. Callers obtain paths through ApplyPaths.
func CaptureTarget(root string, paths []string) (string, error) {
	capture, err := hostwrite.CaptureGuardedWrite(root, paths)
	if err != nil {
		return "", err
	}
	return TargetDigest(capture)
}

// TargetDigest is the precondition token accepted by ApplyRequest.
func TargetDigest(capture *hostwrite.GuardedWriteCapture) (string, error) {
	if capture == nil {
		return "", fmt.Errorf("target capture is required")
	}
	return digest(struct {
		Root     string                                `json:"root"`
		Identity any                                   `json:"identity"`
		Branch   string                                `json:"branch"`
		Head     string                                `json:"head"`
		Files    map[string]hostwrite.GuardedWriteFile `json:"files"`
	}{capture.Root, capture.Identity, capture.Branch, capture.Head, capture.Files})
}

// Apply writes a previously verified candidate after exact target, branch,
// HEAD, source, runtime and candidate checks. Partial Host writes are returned
// and durably journaled; this is not presented as a multi-file transaction.
func Apply(host Host, invoker Invoker, root string, request ApplyRequest) (ApplyReport, error) {
	out := ApplyReport{APIVersion: APIVersion, RunID: request.RunID, CandidateID: request.CandidateID, Status: "rejected", Written: []string{}, Journal: []string{}}
	if host.Load == nil || host.FromSnapshot == nil || invoker == nil {
		return out, fmt.Errorf("apply requires Host and runtime invoker")
	}
	s, err := newRunStore(root)
	if err != nil {
		return out, err
	}
	unlock, err := s.lock()
	if err != nil {
		return out, err
	}
	defer unlock()
	plan, err := s.readPlan(request.PlanID)
	if err != nil {
		return out, err
	}
	if plan.ID != request.RunID {
		return out, fmt.Errorf("apply run and plan IDs do not match")
	}
	run, err := s.readLatestState(request.RunID)
	if err != nil {
		return out, err
	}
	if run.Status != StatusVerified {
		return out, fmt.Errorf("run status %s is not verified", run.Status)
	}
	if run.PlanID != plan.ID || run.Candidate.ID != request.CandidateID || !run.Candidate.Integrated {
		return out, fmt.Errorf("requested candidate is not the integrated candidate for this run")
	}
	if !plan.ExecuteAuthorized {
		return out, ErrNotRunnable
	}
	runtime, err := LoadRuntime(root)
	if err != nil {
		return out, err
	}
	rtDigest, err := runtimeDigestWithInvoker(invoker, runtime)
	if err != nil {
		return out, err
	}
	if rtDigest != plan.RuntimeDigest {
		return out, ErrStale
	}
	base, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		return out, err
	}
	if base.Snapshot == nil || base.Snapshot.Digest() != plan.BaseSnapshot || base.Digest != plan.BaseProjectDigest || base.Report.ModelDigest != plan.BaseModelDigest {
		return out, ErrStale
	}
	if err := repositoryMatches(root, plan); err != nil {
		return out, err
	}
	working, err := host.Load(root, "")
	if err != nil {
		return out, err
	}
	if working.Snapshot == nil || working.Snapshot.Digest() != plan.WorkingSnapshot || working.Digest != plan.WorkingProjectDigest {
		return out, ErrStale
	}
	dir, _ := s.runDir(request.RunID)
	candidate, err := s.readCandidate(dir, request.CandidateID)
	if err != nil {
		return out, err
	}
	verify, err := latestVerification(dir, candidate.ID)
	if err != nil {
		return out, err
	}
	if verify.Status != "verified" || verify.CandidateID != candidate.ID || verify.CandidateHash != candidate.Digest || request.ExpectedVerificationDigest == "" || verify.Digest != request.ExpectedVerificationDigest {
		return out, fmt.Errorf("candidate lacks matching successful verification")
	}
	compiled, err := projectForCandidate(host, root, base.Snapshot, candidate)
	if err != nil {
		return out, err
	}
	if compiled.Report.ModelDigest != plan.ModelDigest || hasErrorFinding(compiled.Report.Findings) {
		return out, fmt.Errorf("candidate project model changed since plan")
	}
	if err := requireArtifacts(compiled.Report); err != nil {
		return out, err
	}
	paths := candidateDeltaPaths(base.Snapshot, candidate)
	if len(paths) == 0 {
		return out, fmt.Errorf("candidate has no changes to apply")
	}
	if request.TargetBranch == "" || request.ExpectedHead == "" || request.ExpectedWorktree == "" || request.ExpectedVerificationDigest == "" {
		return out, fmt.Errorf("apply requires exact verification digest, target branch, HEAD and working-file digest")
	}
	capture, err := hostwrite.CaptureGuardedWrite(root, paths)
	if err != nil {
		return out, err
	}
	if capture.Branch != request.TargetBranch {
		return out, fmt.Errorf("target branch %q differs from requested %q", capture.Branch, request.TargetBranch)
	}
	if capture.Head != request.ExpectedHead {
		return out, fmt.Errorf("target HEAD differs from requested precondition")
	}
	targetDigest, err := TargetDigest(capture)
	if err != nil {
		return out, err
	}
	if targetDigest != request.ExpectedWorktree {
		return out, fmt.Errorf("target working-file bytes differ from requested precondition")
	}
	if err := compareCaptureToBase(capture, base.Snapshot, paths); err != nil {
		return out, err
	}
	changes, err := guardedChanges(candidate, paths)
	if err != nil {
		return out, err
	}
	validate := func() error {
		fresh, loadErr := host.Load(root, "")
		if loadErr != nil {
			return loadErr
		}
		if fresh.Snapshot == nil || fresh.Snapshot.Digest() != plan.WorkingSnapshot || fresh.Digest != plan.WorkingProjectDigest {
			return ErrStale
		}
		if err := repositoryMatches(root, plan); err != nil {
			return err
		}
		currentRuntime, loadErr := LoadRuntime(root)
		if loadErr != nil {
			return loadErr
		}
		currentDigest, loadErr := runtimeDigestWithInvoker(invoker, currentRuntime)
		if loadErr != nil {
			return loadErr
		}
		if currentDigest != plan.RuntimeDigest {
			return ErrStale
		}
		stored, loadErr := s.readCandidate(dir, request.CandidateID)
		if loadErr != nil {
			return loadErr
		}
		if stored.Digest != candidate.Digest {
			return ErrStale
		}
		latest, loadErr := s.readLatestState(request.RunID)
		if loadErr != nil {
			return loadErr
		}
		if latest.Status != StatusVerified || latest.Candidate.ID != request.CandidateID {
			return ErrStale
		}
		latestVerify, loadErr := latestVerification(dir, request.CandidateID)
		if loadErr != nil {
			return loadErr
		}
		if latestVerify.Status != "verified" || latestVerify.CandidateHash != candidate.Digest || latestVerify.Digest != request.ExpectedVerificationDigest {
			return ErrStale
		}
		return nil
	}
	result, applyErr := hostwrite.ApplyGuardedWriteChecked(root, capture, changes, validate)
	out.Written = append([]string(nil), result.CompletedPaths...)
	out.Journal = append([]string(nil), result.CompletedPaths...)
	out.AppliedAt = time.Now().UTC()
	if applyErr != nil {
		out.Status = "partial"
		out.Error = applyErr.Error()
		run.Status = StatusFailed
		run.Findings = append(run.Findings, "guarded apply partially or wholly failed: "+applyErr.Error())
		_ = persistState(s, &run)
		_ = persistApply(s, dir, out)
		return out, applyErr
	}
	out.Status = StatusApplied
	run.Status = StatusApplied
	if err := persistState(s, &run); err != nil {
		out.Status = "partial"
		out.Error = "files applied but run state journal failed: " + err.Error()
		_ = persistApply(s, dir, out)
		return out, fmt.Errorf("files applied; journal state update failed: %w", err)
	}
	if err := persistApply(s, dir, out); err != nil {
		out.Status = "partial"
		out.Error = "files applied but apply report journal failed: " + err.Error()
		return out, fmt.Errorf("files applied; apply report journal failed: %w", err)
	}
	return out, nil
}

func loadApplyInputs(host Host, root, runID string) (PlanRecord, RunReport, candidateData, *Snapshot, error) {
	var p PlanRecord
	var r RunReport
	var c candidateData
	if host.Load == nil {
		return p, r, c, nil, fmt.Errorf("Host load is required")
	}
	s, err := newRunStore(root)
	if err != nil {
		return p, r, c, nil, err
	}
	p, err = s.readPlan(runID)
	if err != nil {
		return p, r, c, nil, err
	}
	r, err = s.readLatestState(runID)
	if err != nil {
		return p, r, c, nil, err
	}
	dir, _ := s.runDir(runID)
	c, err = s.readCandidate(dir, r.Candidate.ID)
	if err != nil {
		return p, r, c, nil, err
	}
	project, err := host.Load(root, p.BaseRevision)
	if err != nil {
		return p, r, c, nil, err
	}
	return p, r, c, project.Snapshot, nil
}

func candidateDeltaPaths(base *Snapshot, c candidateData) []string {
	var paths []string
	for p, f := range c.Files {
		old, exists := base.Files[p]
		mode := base.Modes[p]
		if f.Delete {
			if exists {
				paths = append(paths, p)
			}
			continue
		}
		if !exists || string(old) != string(f.Content) || mode != f.Mode {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	return paths
}
func compareCaptureToBase(capture *hostwrite.GuardedWriteCapture, base *Snapshot, paths []string) error {
	for _, p := range paths {
		current, ok := capture.Files[p]
		if !ok {
			return fmt.Errorf("target capture omitted %s", p)
		}
		old, exists := base.Files[p]
		if exists != current.Exists {
			return fmt.Errorf("target path %s does not match fixed base existence", p)
		}
		if exists {
			mode := fs.FileMode(0o644)
			if base.Modes[p] == "100755" {
				mode = 0o755
			}
			if string(old) != string(current.Bytes) || current.Mode.Perm() != mode {
				return fmt.Errorf("target path %s differs from fixed base bytes or mode", p)
			}
		}
	}
	return nil
}
func guardedChanges(c candidateData, paths []string) ([]hostwrite.GuardedWriteChange, error) {
	changes := make([]hostwrite.GuardedWriteChange, 0, len(paths))
	for _, p := range paths {
		f, ok := c.Files[p]
		if !ok {
			return nil, fmt.Errorf("candidate delta %s is missing", p)
		}
		if f.Delete {
			changes = append(changes, hostwrite.GuardedWriteChange{Path: p, Delete: true})
			continue
		}
		mode := fs.FileMode(0o644)
		if f.Mode == "100755" {
			mode = 0o755
		} else if f.Mode != "100644" {
			return nil, fmt.Errorf("candidate mode %s is unsupported for %s", f.Mode, p)
		}
		changes = append(changes, hostwrite.GuardedWriteChange{Path: p, Bytes: append([]byte(nil), f.Content...), Mode: mode})
	}
	return changes, nil
}

func latestVerification(dir, candidateID string) (VerifyReport, error) {
	var best VerifyReport
	entries, err := os.ReadDir(filepath.Join(dir, "verification"))
	if err != nil {
		return best, err
	}
	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		var current VerifyReport
		if err := readJSON(filepath.Join(dir, "verification", name), &current); err != nil {
			return best, err
		}
		if current.CandidateID == candidateID && current.VerifiedAt.After(best.VerifiedAt) {
			best = current
		}
	}
	if best.CandidateID == "" {
		return best, fmt.Errorf("no verification report for candidate %s", candidateID)
	}
	return best, nil
}
func persistApply(s *runStore, dir string, report ApplyReport) error {
	if err := ensureDirectory(filepath.Join(dir, "apply")); err != nil {
		return err
	}
	id, err := newID()
	if err != nil {
		return err
	}
	return writeImmutableJSON(filepath.Join(dir, "apply", id+".json"), report)
}
