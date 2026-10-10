package source

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
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
	var treeArgs [][]string
	run := func(repo string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "ls-tree") {
			events = append(events, "metadata:"+joined)
			treeArgs = append(treeArgs, append([]string(nil), args...))
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
	if len(events) != 2 || !strings.HasPrefix(events[0], "metadata:") || events[1] != "content" {
		t.Fatalf("acquisition order = %#v, want one batched metadata check before contents", events)
	}
	if len(treeArgs) != 1 {
		t.Fatalf("selected metadata calls = %d, want one", len(treeArgs))
	}
	args := treeArgs[0]
	separator := -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 || !reflect.DeepEqual(args[separator+1:], []string{"one.txt", "two.txt"}) {
		t.Fatalf("selected metadata arguments = %#v, want only sorted literal selected paths", args)
	}
}

func TestSelectedTreePathBatchesBoundCountAndWindowsCommandLength(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	commit := strings.Repeat("a", 40)
	paths := make([]string, maxSelectedTreePathsPerCommand+1)
	for i := range paths {
		paths[i] = fmt.Sprintf("file-%03d", i)
	}
	batches, err := selectedTreePathBatches(root, commit, paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) != 2 || len(batches[0]) != maxSelectedTreePathsPerCommand || len(batches[1]) != 1 {
		t.Fatalf("count-bounded batches = %#v", batches)
	}
	var flattened []string
	for _, batch := range batches {
		if len(batch) > maxSelectedTreePathsPerCommand {
			t.Fatalf("batch has %d paths, limit is %d", len(batch), maxSelectedTreePathsPerCommand)
		}
		flattened = append(flattened, batch...)
	}
	if !reflect.DeepEqual(flattened, paths) {
		t.Fatalf("batched paths = %#v, want original sorted paths", flattened)
	}

	longPaths := []string{strings.Repeat("a", 9000), strings.Repeat("b", 9000), strings.Repeat("c", 9000), strings.Repeat("d", 9000)}
	longBatches, err := selectedTreePathBatches(root, commit, longPaths)
	if err != nil {
		t.Fatal(err)
	}
	if len(longBatches) < 2 {
		t.Fatalf("long paths fit in %d batch(es), want command-length splitting", len(longBatches))
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	baseArgs := []string{"git", "--no-replace-objects", "-c", "safe.directory=" + filepath.ToSlash(abs), "-C", abs}
	for _, batch := range longBatches {
		args := append(append([]string(nil), baseArgs...), selectedTreeArgs(commit, batch)...)
		if got := windowsCommandLineUnits(args); got > maxSelectedTreeCommandUnits {
			t.Fatalf("batch command length = %d UTF-16 units, limit is %d", got, maxSelectedTreeCommandUnits)
		}
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

func TestLoadSelectedDoesNotFetchPromisorObjectsOrMutateRepository(t *testing.T) {
	root, commit := partialCloneFixture(t)
	selectedOID := strings.TrimSpace(gitTest(t, root, "rev-parse", commit+":selected.txt"))
	unselectedOID := strings.TrimSpace(gitTest(t, root, "rev-parse", commit+":unselected.txt"))
	if selectedOID == unselectedOID {
		t.Fatal("fixture selected and unselected blob IDs unexpectedly match")
	}
	missingBefore := gitTest(t, root, "rev-list", "--objects", "--missing=print", commit)
	if !strings.Contains(missingBefore, "?"+selectedOID) || !strings.Contains(missingBefore, "?"+unselectedOID) {
		t.Fatalf("fixture blobs were not both absent before acquisition: %q", missingBefore)
	}
	stateBefore := partialGitState(t, root)

	// GIT_ALLOW_PROTOCOL must independently protect older Git builds which do
	// not recognize GIT_NO_LAZY_FETCH. Simulate that behavior by omitting the
	// latter variable while asking cat-file for the missing selected blob.
	fallbackEnv := CleanGitEnv()
	fallbackEnv = append(fallbackEnv, "GIT_ALLOW_PROTOCOL=")
	fallback := gitCommandWithEnv(root, fallbackEnv, "cat-file", "--batch")
	fallback.Stdin = strings.NewReader(selectedOID + "\n")
	fallbackOutput, fallbackErr := fallback.CombinedOutput()
	if fallbackErr == nil || !strings.Contains(string(fallbackOutput), "transport 'file' not allowed") {
		t.Fatalf("empty-protocol fallback output = %q, error = %v", fallbackOutput, fallbackErr)
	}
	if afterFallback := partialGitState(t, root); !reflect.DeepEqual(afterFallback, stateBefore) {
		t.Fatalf("empty GIT_ALLOW_PROTOCOL changed repository state: before=%v after=%v", stateBefore, afterFallback)
	}

	_, err := LoadSelected(root, commit, []string{"selected.txt"})
	if err == nil || !strings.Contains(err.Error(), "selected.txt") {
		t.Fatalf("selected missing promisor blob error = %v, want safe missing-object failure", err)
	}
	if stateAfter := partialGitState(t, root); !reflect.DeepEqual(stateAfter, stateBefore) {
		t.Fatalf("LoadSelected changed repository state: before=%v after=%v", stateBefore, stateAfter)
	}
	missingAfter := gitTest(t, root, "rev-list", "--objects", "--missing=print", commit)
	if !strings.Contains(missingAfter, "?"+selectedOID) || !strings.Contains(missingAfter, "?"+unselectedOID) {
		t.Fatalf("selected or unselected promisor blob was fetched: %q", missingAfter)
	}
}

func TestSelectiveGitEnvironmentDisablesLazyFetchAndWrites(t *testing.T) {
	values := make(map[string]string)
	for _, entry := range selectiveGitEnvironment() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	for key, want := range map[string]string{
		"GIT_NO_LAZY_FETCH":   "1",
		"GIT_ALLOW_PROTOCOL":  "",
		"GIT_OPTIONAL_LOCKS":  "0",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		if got, ok := values[key]; !ok || got != want {
			t.Errorf("selective environment %s = %q (present %v), want %q", key, got, ok, want)
		}
	}
}

func partialCloneFixture(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	source := filepath.Join(base, "source")
	remote := filepath.Join(base, "remote.git")
	partial := filepath.Join(base, "partial")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	gitTest(t, source, "init", "-q", "-b", "main")
	gitTest(t, source, "config", "user.email", "promisor-test@example.invalid")
	gitTest(t, source, "config", "user.name", "Promisor Test")
	writeTestFile(t, source, "selected.txt", "selected blob")
	writeTestFile(t, source, "unselected.txt", "unselected blob")
	gitTest(t, source, "add", "selected.txt", "unselected.txt")
	gitTest(t, source, "commit", "-qm", "promisor source")
	gitTest(t, base, "init", "--bare", "-q", "--initial-branch=main", remote)
	gitTest(t, remote, "config", "uploadpack.allowFilter", "true")
	gitTest(t, source, "remote", "add", "origin", localGitURL(remote))
	gitTest(t, source, "push", "-q", "origin", "main")
	gitTest(t, remote, "symbolic-ref", "HEAD", "refs/heads/main")
	gitTest(t, source, "-c", "protocol.file.allow=always", "clone", "--quiet", "--filter=blob:none", "--no-checkout", "--branch", "main", localGitURL(remote), partial)
	commit := strings.TrimSpace(gitTest(t, partial, "rev-parse", "HEAD"))
	config := gitTest(t, partial, "config", "--get-regexp", "remote\\.origin\\.promisor|remote\\.origin\\.partialclonefilter")
	if !strings.Contains(config, "remote.origin.promisor true") || !strings.Contains(config, "remote.origin.partialclonefilter blob:none") {
		t.Fatalf("partial clone promisor config = %q", config)
	}
	return partial, commit
}

func localGitURL(path string) string {
	gitPath := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" && !strings.HasPrefix(gitPath, "/") {
		gitPath = "/" + gitPath
	}
	return (&url.URL{Scheme: "file", Path: gitPath}).String()
}

func partialGitState(t *testing.T, root string) map[string]string {
	t.Helper()
	state := make(map[string]string)
	gitDir := filepath.Join(root, ".git")
	for _, name := range []string{"config", "index", "FETCH_HEAD"} {
		path := filepath.Join(gitDir, name)
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			state[name] = "<absent>"
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		state[name] = hex.EncodeToString(digest[:])
	}
	packDir := filepath.Join(gitDir, "objects", "pack")
	entries, err := os.ReadDir(packDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
		data, err := os.ReadFile(filepath.Join(packDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		state["pack:"+entry.Name()] = hex.EncodeToString(digest[:])
	}
	sort.Strings(names)
	state["packFiles"] = fmt.Sprint(names)
	return state
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
