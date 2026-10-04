package app

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adoption"
)

func TestPrepareAdoptionPreviewWriteAndReadWorkspace(t *testing.T) {
	base := adoptionTestTempDir(t)
	root, commit := adoptionTestRepository(t, filepath.Join(base, "source"))
	destination := filepath.Join(base, "prepared")
	scope := adoptionTestScope(root, commit)

	preview, err := PrepareAdoption(scope, destination, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Status != "planned" || preview.Handoff.Digest == "" || preview.Destination != destination {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatalf("preview created destination: %v", err)
	}

	if _, err := PrepareAdoption(scope, destination, "wrong", true); err == nil || !strings.Contains(err.Error(), "stale adoption preview") {
		t.Fatalf("stale expected digest accepted: %v", err)
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatalf("stale write created destination: %v", err)
	}

	written, err := PrepareAdoption(scope, destination, preview.Handoff.Digest, true)
	if err != nil {
		t.Fatal(err)
	}
	if written.Status != "written" || written.Handoff.Digest != preview.Handoff.Digest || len(written.CreatedFiles) != 2 {
		t.Fatalf("unexpected write result: %#v", written)
	}
	manifestBytes, blobs, err := ReadAdoptionWorkspace(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(manifestBytes, []byte(preview.Handoff.Digest)) || len(blobs) != 1 || string(blobs[adoption.BlobKey("main", "docs/selected.md")]) != "selected evidence\n" {
		t.Fatalf("unexpected workspace read: manifest=%s blobs=%#v", manifestBytes, blobs)
	}
	if _, err := os.Stat(filepath.Join(destination, "evidence", "main", "notes", "unselected.md")); !os.IsNotExist(err) {
		t.Fatalf("unselected evidence exists: %v", err)
	}
	if _, err := PrepareAdoption(scope, destination, preview.Handoff.Digest, true); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing workspace was reused: %v", err)
	}
}

func TestPrepareAdoptionRejectsDestinationOverlap(t *testing.T) {
	base := adoptionTestTempDir(t)
	root, commit := adoptionTestRepository(t, filepath.Join(base, "source"))
	scope := adoptionTestScope(root, commit)
	destination := filepath.Join(root, "handoff")
	if _, err := PrepareAdoption(scope, destination, "", false); err == nil || !strings.Contains(err.Error(), "overlaps source or Git metadata") {
		t.Fatalf("destination inside source accepted: %v", err)
	}
}

func TestPrepareAdoptionReportsPartialWorkspaceAndNeverRollsBack(t *testing.T) {
	base := adoptionTestTempDir(t)
	root, commit := adoptionTestRepository(t, filepath.Join(base, "source"))
	scope := adoptionTestScope(root, commit)
	identities, identityRepos, err := identifyAdoptionScope(scope)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := captureAdoptionScope(scope, identities)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(base, "partial")
	parent, canonicalDestination, parentInfo, err := validateAdoptionDestination(destination, identityRepos)
	if err != nil {
		t.Fatal(err)
	}
	result := &AdoptionPreparation{Status: "planned", Handoff: prepared.handoff, Destination: canonicalDestination}
	seen := 0
	errInjected := errors.New("injected filesystem failure")
	err = writeAdoptionWorkspace(result, parent, canonicalDestination, parentInfo, prepared, adoptionWriteOps{before: func(_, path string) error {
		seen++
		if path == "evidence/main/docs/selected.md" {
			return errInjected
		}
		return nil
	}})
	if !errors.Is(err, errInjected) || result.Status != "failed" || result.Recovery == "" {
		t.Fatalf("partial failure result = %#v, err=%v", result, err)
	}
	if len(result.CreatedDirectories) != 4 || len(result.CreatedFiles) != 0 {
		t.Fatalf("partial paths not reported exactly: %#v", result)
	}
	if _, err := os.Stat(destination); err != nil {
		t.Fatalf("partial workspace was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "handoff.yaml")); !os.IsNotExist(err) {
		t.Fatalf("handoff was written before evidence: %v", err)
	}
	if seen < 4 {
		t.Fatalf("fault seam saw only %d operations", seen)
	}
	if _, err := PrepareAdoption(scope, destination, prepared.handoff.Digest, true); err == nil {
		t.Fatal("rerun accepted an existing partial workspace")
	}
}

func TestPrepareAdoptionChoosesDeterministicFirstInvalidEvidence(t *testing.T) {
	base := adoptionTestTempDir(t)
	root, _ := adoptionTestRepository(t, filepath.Join(base, "source"))
	for path, data := range map[string][]byte{
		"docs/a-invalid.md": {0xff},
		"docs/z-invalid.md": {0xfe},
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	gitAt := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	gitAt("add", "docs/a-invalid.md", "docs/z-invalid.md")
	gitAt("commit", "-qm", "invalid utf8 evidence")
	scope := adoptionTestScope(root, gitAt("rev-parse", "HEAD"))
	scope.Repositories[0].Paths = append(scope.Repositories[0].Paths,
		adoption.SelectedPath{Path: "docs/z-invalid.md", Reason: "later invalid fixture"},
		adoption.SelectedPath{Path: "docs/a-invalid.md", Reason: "first invalid fixture"},
	)
	for attempt := 0; attempt < 8; attempt++ {
		_, err := PrepareAdoption(scope, filepath.Join(base, "workspace"), "", false)
		if err == nil || !strings.Contains(err.Error(), "docs/a-invalid.md is not UTF-8") {
			t.Fatalf("attempt %d error = %v, want lexically first invalid path", attempt, err)
		}
	}
}

func TestReadAdoptionWorkspaceRejectsUnlistedFiles(t *testing.T) {
	base := adoptionTestTempDir(t)
	root, commit := adoptionTestRepository(t, filepath.Join(base, "source"))
	scope := adoptionTestScope(root, commit)
	destination := filepath.Join(base, "workspace")
	preview, err := PrepareAdoption(scope, destination, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareAdoption(scope, destination, preview.Handoff.Digest, true); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "unexpected.txt"), []byte("extra"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadAdoptionWorkspace(destination); err == nil || !strings.Contains(err.Error(), "unexpected workspace file") {
		t.Fatalf("unlisted file accepted: %v", err)
	}
}

func TestReadAdoptionWorkspaceRejectsUnsafeManifestBeforeEvidence(t *testing.T) {
	base := adoptionTestTempDir(t)
	root, commit := adoptionTestRepository(t, filepath.Join(base, "source"))
	destination := filepath.Join(base, "workspace")
	scope := adoptionTestScope(root, commit)
	preview, err := PrepareAdoption(scope, destination, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareAdoption(scope, destination, preview.Handoff.Digest, true); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(destination, "handoff.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var handoff adoption.Handoff
	if err := adoption.Decode(data, &handoff); err != nil {
		t.Fatal(err)
	}
	handoff.Repositories[0].Files[0].Path = "../outside.md"
	adoption.Seal(&handoff)
	data, err = adoption.Encode(handoff)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadAdoptionWorkspace(destination); err == nil || !strings.Contains(err.Error(), "unsafe exact path") {
		t.Fatalf("unsafe manifest path was accepted or evidence was opened first: %v", err)
	}
}

func TestReadAdoptionCandidateRejectsCaseAlias(t *testing.T) {
	root := adoptionTestTempDir(t)
	if err := os.WriteFile(filepath.Join(root, "candidate.yaml"), []byte("candidate: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAdoptionCandidate(root, "CANDIDATE.yaml"); err == nil {
		t.Fatal("case-alias candidate path accepted")
	}
}

func TestReadAdoptionRecordAndCandidateRejectLinksAndTraversal(t *testing.T) {
	root, err := canonicalUserPath(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(root, "scope.yaml")
	if err := os.WriteFile(record, []byte("kind: scope\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAdoptionRecord(record); err != nil {
		t.Fatal(err)
	}
	queue := filepath.Join(root, "queue")
	if err := os.Mkdir(queue, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queue, "candidate.yaml"), []byte("kind: candidate\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAdoptionCandidate(queue, "candidate.yaml"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAdoptionCandidate(queue, "../scope.yaml"); err == nil {
		t.Fatal("candidate path traversal accepted")
	}
	if err := os.Symlink(record, filepath.Join(queue, "linked.yaml")); err == nil {
		if _, err := ReadAdoptionCandidate(queue, "linked.yaml"); err == nil {
			t.Fatal("candidate symlink accepted")
		}
	}
}

func adoptionTestRepository(t *testing.T, root string) (string, string) {
	t.Helper()
	parent, err := canonicalUserPath(filepath.Dir(root))
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Join(parent, filepath.Base(root))
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "feature/adoption")
	git("config", "user.name", "Markitect Test")
	git("config", "user.email", "markitect-test@example.invalid")
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "selected.md"), []byte("selected evidence\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes", "unselected.md"), []byte("not selected\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", "docs/selected.md", "notes/unselected.md")
	git("commit", "-qm", "source fixture")
	return root, git("rev-parse", "HEAD")
}

func adoptionTestTempDir(t *testing.T) string {
	t.Helper()
	raw := t.TempDir()
	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil {
		t.Fatal(err)
	}
	root, err := canonicalUserPath(resolved)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func adoptionTestScope(root, commit string) adoption.Scope {
	return adoption.Scope{
		APIVersion: adoption.ScopeVersion,
		ID:         "adoption-test",
		Purpose:    "test exact evidence acquisition",
		Review:     "owner-reviewed fixture scope",
		Privacy:    adoption.Privacy{Constraints: "synthetic test data only", AllowExcerpts: false},
		Retention:  "delete after test",
		Repositories: []adoption.ScopeRepository{{
			ID:     "main",
			Root:   root,
			Commit: commit,
			Paths:  []adoption.SelectedPath{{Path: "docs/selected.md", Reason: "the single selected fixture"}},
		}},
	}
}
