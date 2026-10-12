package projectworkspace

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func TestSameGitReportedPathRejectsDifferentAndMissingPaths(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	if err := os.Mkdir(left, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(right, 0700); err != nil {
		t.Fatal(err)
	}
	if sameGitReportedPath(left, right) {
		t.Fatal("distinct existing directories were treated as the same path")
	}
	if sameGitReportedPath(filepath.Join(root, "missing-left"), filepath.Join(root, "missing-right")) {
		t.Fatal("distinct nonexistent paths were treated as the same path")
	}
}

func TestSamePathRecognizesWindowsDirectoryAliasByIdentity(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows physical path aliases are the behavior under test")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	alias := filepath.Join(root, "alias")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	// A directory junction is available without symlink privilege on supported
	// Windows filesystems and gives the same directory a distinct absolute path.
	command := "mklink /J " + alias + " " + target
	if output, err := exec.Command("cmd.exe", "/d", "/c", command).CombinedOutput(); err != nil {
		t.Skipf("cannot create a test directory junction: %v (%s)", err, output)
	}
	left, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	right, err := os.Stat(alias)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(left, right) {
		t.Skip("this filesystem does not expose junction targets as one file identity")
	}
	if filepath.Clean(target) == filepath.Clean(alias) {
		t.Fatal("junction did not produce a distinct path spelling")
	}
	if !sameGitReportedPath(target, alias) {
		t.Fatal("same physical Windows directory was rejected through its junction alias")
	}
	if sameGitReportedPath(target, filepath.Join(root, "missing")) {
		t.Fatal("an unrelated missing path was accepted as a physical alias")
	}
}

func TestSameGitReportedPathRejectsFileIdentityAlias(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows filesystem identity comparison is the behavior under test")
	}
	root := t.TempDir()
	original := filepath.Join(root, "original")
	alias := filepath.Join(root, "alias")
	if err := os.WriteFile(original, []byte("identity"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(original, alias); err != nil {
		t.Skipf("cannot create a test hard link: %v", err)
	}
	left, err := os.Stat(original)
	if err != nil {
		t.Fatal(err)
	}
	right, err := os.Stat(alias)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(left, right) {
		t.Skip("this filesystem does not expose hard links as one file identity")
	}
	if sameGitReportedPath(original, alias) {
		t.Fatal("Git-reported directory identity check accepted a file hard-link alias")
	}
}

func TestSourceInventoryAcceptsWindowsGitTopLevelAlias(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows packaged path alias is the behavior under test")
	}
	fixture := newGitFixture(t)
	alias := filepath.Join(t.TempDir(), "source-alias")
	command := "mklink /J " + alias + " " + fixture.root
	if output, err := exec.Command("cmd.exe", "/d", "/c", command).CombinedOutput(); err != nil {
		t.Skipf("cannot create a test source directory junction: %v (%s)", err, output)
	}
	resolved, err := filepath.EvalSymlinks(alias)
	if err != nil {
		t.Fatal(err)
	}
	top := strings.TrimSpace(testGit(t, alias, "rev-parse", "--show-toplevel"))
	if samePath(resolved, top) {
		t.Skip("junction paths collapsed to the same spelling on this runtime")
	}
	if !sameGitReportedPath(resolved, top) {
		t.Fatal("test junction does not represent one existing Git top-level directory")
	}
	_, err = sourceInventory(context.Background(), alias, fixture.base)
	if err != nil {
		t.Fatalf("valid Git root reported through Windows package alias was rejected: %v", err)
	}
}

func TestSourceInventoryStillRejectsSubdirectoryAndDifferentBase(t *testing.T) {
	fixture := newGitFixture(t)
	if _, err := sourceInventory(context.Background(), filepath.Join(fixture.root, "src"), fixture.base); err == nil {
		t.Fatal("Git subdirectory was accepted as the repository root")
	}
	other := newGitFixture(t)
	writeFixtureFile(t, other.root, "src/other.go", []byte("package app\nconst Other = true\n"), 0644)
	testGit(t, other.root, "add", "src/other.go")
	testGit(t, other.root, "commit", "--quiet", "-m", "different base")
	if _, err := sourceInventory(context.Background(), other.root, fixture.base); err == nil {
		t.Fatal("a different commit was accepted for the selected base SHA")
	}
}

func TestDefaultStorageParentChildGitAliasLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fixture := newGitFixture(t)
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	defaultWorkspacesRoot := filepath.Join(cacheRoot, "Markitect", "workspaces")
	if err := os.MkdirAll(defaultWorkspacesRoot, 0700); err != nil {
		t.Fatal(err)
	}
	storageRoot, err := os.MkdirTemp(defaultWorkspacesRoot, "t-")
	if err != nil {
		t.Fatal(err)
	}
	var parentService *GitService
	parentClosed := false
	var parentHandle Handle
	parentPrepared := false
	var childPrepareService *GitService
	var childReopenService *GitService
	var childHandle Handle
	childPrepared := false
	childReopened := false
	childClosed := false
	defer func() {
		if childPrepared && !childClosed {
			closer := childPrepareService
			if childReopened {
				closer = childReopenService
			}
			if closer != nil {
				_ = closer.Close(context.Background(), childHandle)
			}
		}
		if parentPrepared && !parentClosed {
			_ = parentService.Close(context.Background(), parentHandle)
		}
		// This directory was created uniquely for this test. Removing it with
		// os.Remove succeeds only after the service-owned workspaces are gone.
		if err := os.Remove(storageRoot); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("test-owned workspace storage root is not empty after exact closes: %v", err)
		}
	}()
	parentService, err = NewGitService(storageRoot, gitServiceTestLimits)
	if err != nil {
		t.Fatal(err)
	}

	sourceIdentity, err := source.IdentifyGit(fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	sourceBinding, err := InspectRepository(ctx, fixture.root, fixture.base)
	if err != nil {
		t.Fatal(err)
	}
	parentRequest := Request{RepositoryRoot: fixture.root, RepositoryIdentity: sourceIdentity.Digest, BaseSHA: fixture.base,
		OverlayDigest: sourceBinding.OverlayDigest, TaskID: "directory-alias-parent", AllowedPaths: []string{"src"}}
	if err := parentRequest.Validate(); err != nil {
		t.Fatal(err)
	}
	parentOverlayDigest, err := CandidateOverlayDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	parentHandle, err = parentService.PrepareCandidate(ctx, parentRequest, nil, parentOverlayDigest)
	if err != nil {
		t.Fatalf("prepare parent in default packaged-app storage: %v", err)
	}
	parentPrepared = true
	resolvedParent, err := filepath.EvalSymlinks(parentHandle.CWD)
	if err != nil {
		t.Fatal(err)
	}
	gitParentTop := strings.TrimSpace(testGit(t, parentHandle.CWD, "rev-parse", "--show-toplevel"))
	aliasObserved := !samePath(resolvedParent, gitParentTop)
	if !sameGitReportedPath(resolvedParent, gitParentTop) {
		t.Fatalf("Git top-level does not identify the exact parent directory: resolved=%q git=%q", resolvedParent, gitParentTop)
	}
	if aliasObserved {
		t.Logf("WINDOWS_PATH_ALIAS_OBSERVED resolved=%q gitTop=%q sameFile=true", resolvedParent, gitParentTop)
	} else {
		t.Logf("WINDOWS_PATH_ALIAS_NOT_PRESENT resolved=%q gitTop=%q", resolvedParent, gitParentTop)
	}
	parentGitIdentity, err := source.IdentifyGit(parentHandle.CWD)
	if err != nil {
		t.Fatalf("identify prepared parent repository through its effective path: %v", err)
	}
	parentBinding, err := InspectRepository(ctx, parentHandle.CWD, parentHandle.BaseSHA)
	if err != nil {
		t.Fatalf("inspect prepared parent baseline through Git's physical path alias: %v", err)
	}

	childRequest := Request{RepositoryRoot: parentHandle.CWD, RepositoryIdentity: parentGitIdentity.Digest, BaseSHA: parentHandle.BaseSHA,
		OverlayDigest: parentBinding.OverlayDigest, TaskID: "directory-alias-child", AllowedPaths: []string{"src"}}
	childPrepareService, err = NewGitService(storageRoot, gitServiceTestLimits)
	if err != nil {
		t.Fatal(err)
	}
	childOverlayDigest, err := CandidateOverlayDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	childHandle, err = childPrepareService.PrepareCandidate(ctx, childRequest, nil, childOverlayDigest)
	if err != nil {
		t.Fatalf("prepare child from the original parent CWD: %v", err)
	}
	childPrepared = true
	firstDelta, err := childPrepareService.Harvest(ctx, childHandle)
	if err != nil {
		t.Fatalf("harvest untouched child candidate: %v", err)
	}
	firstNormalized, err := NormalizeDelta(childRequest, childHandle, firstDelta.Changes, gitServiceTestLimits)
	if err != nil || len(firstNormalized.Changes) != 0 || firstNormalized.Digest != firstDelta.Digest || firstDelta.RepositoryIdentity != childRequest.RepositoryIdentity || firstDelta.BaseSHA != childRequest.BaseSHA || firstDelta.OverlayDigest != childRequest.OverlayDigest || firstDelta.TaskID != childRequest.TaskID || firstDelta.BaseDigest != childHandle.BaseDigest {
		t.Fatalf("child initial harvest changed baseline or binding: changes=%d delta=%q normalized=%q err=%v", len(firstNormalized.Changes), firstDelta.Digest, firstNormalized.Digest, err)
	}

	childReopenService, err = NewGitService(storageRoot, gitServiceTestLimits)
	if err != nil {
		t.Fatal(err)
	}
	if err := childReopenService.ReopenCandidate(ctx, childRequest, childHandle, nil, childOverlayDigest, true); err != nil {
		t.Fatalf("reopen exact child ownership through aliased parent baseline: %v", err)
	}
	childReopened = true
	secondDelta, err := childReopenService.Harvest(ctx, childHandle)
	if err != nil {
		t.Fatalf("harvest reopened child candidate: %v", err)
	}
	secondNormalized, err := NormalizeDelta(childRequest, childHandle, secondDelta.Changes, gitServiceTestLimits)
	if err != nil || len(secondNormalized.Changes) != 0 || secondNormalized.Digest != firstNormalized.Digest || secondDelta.Digest != firstDelta.Digest {
		t.Fatalf("reopened child baseline/delta changed: first=%q second=%q normalized=%q changes=%d err=%v", firstDelta.Digest, secondDelta.Digest, secondNormalized.Digest, len(secondNormalized.Changes), err)
	}
	if err := childReopenService.Close(ctx, childHandle); err != nil {
		t.Fatalf("close exact child workspace once: %v", err)
	}
	childClosed = true
	if err := parentService.Close(ctx, parentHandle); err != nil {
		t.Fatalf("close exact parent workspace once: %v", err)
	}
	parentClosed = true
	if err := os.Remove(storageRoot); err != nil {
		t.Fatalf("test-owned default-storage root is not empty after both exact closes: %v", err)
	}
	t.Logf("PARENT_CHILD_GIT_BOUNDARY_OK aliasObserved=%t parent=%s child=%s delta=%s storageRootRemoved=true", aliasObserved, parentHandle.ID, childHandle.ID, secondNormalized.Digest)
}
