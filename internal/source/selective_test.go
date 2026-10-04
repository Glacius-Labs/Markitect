package source

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSelectedReadsOnlyExactBlobsAfterAllMetadata(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "one.txt", "first")
	writeTestFile(t, root, "two.txt", "second")
	writeTestFile(t, root, "unselected.txt", "do not read")
	gitTest(t, root, "add", "one.txt", "two.txt", "unselected.txt")
	gitTest(t, root, "commit", "-qm", "selected files")
	commit := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))

	var events []string
	run := func(repo string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "ls-tree") {
			events = append(events, "metadata:"+joined)
		}
		return GitOutput(repo, args...)
	}
	read := func(repo string, files []treeFile) (map[string][]byte, error) {
		events = append(events, "content")
		if len(files) != 2 || files[0].path != "one.txt" || files[1].path != "two.txt" {
			t.Fatalf("blob request = %#v, want only sorted selected files", files)
		}
		return readSelectedBlobs(repo, files)
	}
	got, err := loadSelected(root, commit, []string{"two.txt", "one.txt"}, run, read)
	if err != nil {
		t.Fatal(err)
	}
	if got.Snapshot.Provisional || got.Snapshot.ID != commit {
		t.Fatalf("snapshot identity = (%q, provisional %v)", got.Snapshot.ID, got.Snapshot.Provisional)
	}
	if len(got.Snapshot.Files) != 2 || !bytes.Equal(got.Snapshot.Files["one.txt"], []byte("first")) || !bytes.Equal(got.Snapshot.Files["two.txt"], []byte("second")) {
		t.Fatalf("selected contents = %#v", got.Snapshot.Files)
	}
	if _, ok := got.Snapshot.Files["unselected.txt"]; ok {
		t.Fatal("unselected path entered snapshot")
	}
	if got.Identity.Digest == "" || !samePath(got.Identity.Root, root) || got.Identity.ObjectFormat != "sha1" {
		t.Fatalf("unexpected Git identity: %#v expectedRoot=%q rootSame=%v digestSame=%v", got.Identity, root, samePath(got.Identity.Root, root), got.Identity.Digest == gitIdentityDigest(got.Identity))
	}
	if len(events) != 3 || !strings.HasPrefix(events[0], "metadata:") || !strings.HasPrefix(events[1], "metadata:") || events[2] != "content" {
		t.Fatalf("acquisition order = %#v, want both metadata checks before contents", events)
	}
}

func TestLoadSelectedRejectsInvalidPreflightBeforeAnyBlobRead(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "dir/file.txt", "file")
	writeTestFile(t, root, "link-target.txt", "target")
	gitTest(t, root, "add", "dir/file.txt", "link-target.txt")
	gitTest(t, root, "commit", "-qm", "tree and file")
	commit := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	blobOID := strings.Fields(gitTest(t, root, "rev-parse", "HEAD:link-target.txt"))[0]
	gitTest(t, root, "update-index", "--add", "--cacheinfo", "120000,"+blobOID+",link.txt")
	gitTest(t, root, "commit", "-qm", "symlink")
	commit = strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))

	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "missing", path: "missing.txt", want: "does not identify exactly one"},
		{name: "tree", path: "dir", want: "is a tree"},
		{name: "symlink", path: "link.txt", want: "symlink"},
		{name: "case alias miss", path: "DIR/file.txt", want: "does not identify exactly one"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contentReads := 0
			_, err := loadSelected(root, commit, []string{tc.path}, GitOutput, func(string, []treeFile) (map[string][]byte, error) {
				contentReads++
				return nil, nil
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want containing %q", err, tc.want)
			}
			if contentReads != 0 {
				t.Fatalf("content reads = %d, want zero after failed preflight", contentReads)
			}
		})
	}
	_, err := loadSelected(root, commit, []string{"A.txt", "a.txt"}, GitOutput, func(string, []treeFile) (map[string][]byte, error) {
		t.Fatal("content reader called for aliased selections")
		return nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("selection aliases error = %v", err)
	}
}

func TestLoadSelectedRejectsSubmoduleBeforeBlobRead(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	gitTest(t, root, "update-index", "--add", "--cacheinfo", "160000,"+strings.Repeat("a", 40)+",module")
	gitTest(t, root, "commit", "-qm", "submodule entry")
	commit := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	contentReads := 0
	_, err := loadSelected(root, commit, []string{"module"}, GitOutput, func(string, []treeFile) (map[string][]byte, error) {
		contentReads++
		return nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), "submodule") {
		t.Fatalf("error = %v, want submodule rejection", err)
	}
	if contentReads != 0 {
		t.Fatalf("content reads = %d, want zero", contentReads)
	}
}

func TestLoadSelectedAllowsExplicitlySelectedNormallyExcludedPath(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "vendor/evidence.txt", "selected evidence")
	gitTest(t, root, "add", "vendor/evidence.txt")
	gitTest(t, root, "commit", "-qm", "explicit evidence")
	commit := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	loaded, err := LoadSelected(root, commit, []string{"vendor/evidence.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if string(loaded.Snapshot.Files["vendor/evidence.txt"]) != "selected evidence" {
		t.Fatalf("explicitly selected content = %q", loaded.Snapshot.Files["vendor/evidence.txt"])
	}
}

func TestLoadSelectedRejectsGlobsAndDuplicatesBeforeBlobRead(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "evidence[1].txt", "content")
	gitTest(t, root, "add", "evidence[1].txt")
	gitTest(t, root, "commit", "-qm", "bracketed path")
	commit := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	for _, paths := range [][]string{{"evidence[1].txt"}, {"seed.txt", "seed.txt"}} {
		contentReads := 0
		_, err := loadSelected(root, commit, paths, GitOutput, func(string, []treeFile) (map[string][]byte, error) {
			contentReads++
			return nil, nil
		})
		if err == nil {
			t.Errorf("accepted invalid selected path set %#v", paths)
		}
		if contentReads != 0 {
			t.Errorf("read blobs for invalid selected path set %#v", paths)
		}
	}
}

func TestLoadSelectedVerifiesReturnedBlobBytesAgainstOID(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "evidence.txt", "correct")
	gitTest(t, root, "add", "evidence.txt")
	gitTest(t, root, "commit", "-qm", "selected blob")
	commit := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	_, err := loadSelected(root, commit, []string{"evidence.txt"}, GitOutput, func(string, []treeFile) (map[string][]byte, error) {
		return map[string][]byte{"evidence.txt": []byte("corrupt")}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("error = %v, want blob hash mismatch", err)
	}
}

func TestIdentifyGitRejectsNestedRootAndIdentifiesWorktree(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	if _, err := IdentifyGit(filepath.Join(root, "nested")); err == nil {
		t.Fatal("nested non-repository root was accepted")
	}
	identity, err := IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(identity.Root, root) || identity.GitDir == "" || identity.CommonDir == "" || identity.Digest != gitIdentityDigest(identity) {
		t.Fatalf("identity = %#v expectedRoot=%q rootSame=%v digestSame=%v", identity, root, samePath(identity.Root, root), identity.Digest == gitIdentityDigest(identity))
	}
	clone := t.TempDir()
	if err := exec.Command("git", "clone", "--quiet", root, clone).Run(); err != nil {
		t.Fatal(err)
	}
	cloneIdentity, err := IdentifyGit(clone)
	if err != nil {
		t.Fatal(err)
	}
	if cloneIdentity.Digest == identity.Digest {
		t.Fatal("independent clone has same repository-location identity digest")
	}
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "--detach", linked, strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD")))
	linkedIdentity, err := IdentifyGit(linked)
	if err != nil {
		t.Fatal(err)
	}
	if samePath(linkedIdentity.GitDir, identity.GitDir) || !samePath(linkedIdentity.CommonDir, identity.CommonDir) || linkedIdentity.Digest == identity.Digest {
		t.Fatalf("linked worktree identity = %#v; main = %#v", linkedIdentity, identity)
	}
}

func TestLoadSelectedRejectsCommitSelectorsAndRunnerFailure(t *testing.T) {
	root, commit := selectiveGitFixture(t)
	for _, selector := range []string{"HEAD", commit[:12], strings.ToUpper(commit)} {
		if _, err := loadSelected(root, selector, nil, GitOutput, func(string, []treeFile) (map[string][]byte, error) { return nil, nil }); err == nil {
			t.Errorf("accepted non-full-lowercase commit selector %q", selector)
		}
	}
	failed := errors.New("injected Git failure")
	_, err := loadSelected(root, commit, nil, func(string, ...string) ([]byte, error) { return nil, failed }, func(string, []treeFile) (map[string][]byte, error) { return nil, nil })
	if !errors.Is(err, failed) {
		t.Fatalf("error = %v, want injected failure", err)
	}
}

func selectiveGitFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	gitTest(t, root, "init", "-q", "-b", "main")
	gitTest(t, root, "config", "user.email", "selective-test@example.invalid")
	gitTest(t, root, "config", "user.name", "Selective Test")
	writeTestFile(t, root, "seed.txt", "seed")
	gitTest(t, root, "add", "seed.txt")
	gitTest(t, root, "commit", "-qm", "seed")
	return root, strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
}
