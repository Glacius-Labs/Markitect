package source

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWriteSelectedEvidenceCommitPreservesRepositoryAndLoadsSelectedBytes(t *testing.T) {
	root, parent := selectiveGitFixture(t)
	writeTestFile(t, root, "seed.txt", "staged change")
	gitTest(t, root, "add", "seed.txt")
	writeTestFile(t, root, "seed.txt", "worktree change")
	writeTestFile(t, root, "untracked.txt", "retain me")
	before := evidenceRepoState(t, root)
	files := map[string][]byte{"evidence/result.txt": []byte("reviewed bytes\n"), "tool.sh": []byte("#!/bin/sh\n")}
	modes := map[string]string{"evidence/result.txt": "100644", "tool.sh": "100755"}
	commit, err := WriteSelectedEvidenceCommit(root, parent, files, modes, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(commit) != 40 {
		t.Fatalf("commit ID = %q, want full SHA-1 ID", commit)
	}
	if after := evidenceRepoState(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("repository state changed: before=%v after=%v", before, after)
	}
	loaded, err := LoadSelected(root, commit, []string{"tool.sh", "evidence/result.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(loaded.Snapshot.Files["evidence/result.txt"], files["evidence/result.txt"]) || loaded.Snapshot.Modes["tool.sh"] != "100755" {
		t.Fatalf("selected evidence = %#v / %#v", loaded.Snapshot.Files, loaded.Snapshot.Modes)
	}
	if !strings.Contains(gitTest(t, root, "show", "-s", "--format=%P", commit), parent) {
		t.Fatal("evidence commit lost its expected parent")
	}
}

func TestWriteSelectedEvidenceCommitIsDeterministicAndContentBound(t *testing.T) {
	root, parent := selectiveGitFixture(t)
	files := map[string][]byte{"evidence.txt": []byte("one\n")}
	modes := map[string]string{"evidence.txt": "100644"}
	first, err := WriteSelectedEvidenceCommit(root, parent, files, modes, true)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := WriteSelectedEvidenceCommit(root, parent, files, modes, true)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := WriteSelectedEvidenceCommit(root, parent, map[string][]byte{"evidence.txt": []byte("two\n")}, modes, true)
	if err != nil {
		t.Fatal(err)
	}
	if first != repeated {
		t.Fatalf("same inputs produced different IDs: %s != %s", first, repeated)
	}
	if first == changed {
		t.Fatal("changed bytes produced the same commit ID")
	}
	if got := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD")); got != parent {
		t.Fatalf("HEAD moved from %s to %s", parent, got)
	}
}

func TestWriteSelectedEvidenceCommitDryRunWritesNoObjects(t *testing.T) {
	root, parent := selectiveGitFixture(t)
	before := gitTest(t, root, "count-objects", "-v")
	commit, err := WriteSelectedEvidenceCommit(root, parent, map[string][]byte{"evidence.txt": []byte("ok")}, map[string]string{"evidence.txt": "100644"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if commit != "" {
		t.Fatalf("dry-run commit ID = %q", commit)
	}
	if after := gitTest(t, root, "count-objects", "-v"); after != before {
		t.Fatalf("dry-run wrote objects: before=%q after=%q", before, after)
	}
}

func TestWriteSelectedEvidenceCommitRejectsInvalidPathsBeforeObjects(t *testing.T) {
	root, parent := selectiveGitFixture(t)
	cases := []struct {
		name  string
		files map[string][]byte
		modes map[string]string
	}{
		{"invalid utf8", map[string][]byte{"evidence.txt": {0xff}}, map[string]string{"evidence.txt": "100644"}},
		{"unsafe path", map[string][]byte{"../evidence.txt": []byte("x")}, map[string]string{"../evidence.txt": "100644"}},
		{"path alias", map[string][]byte{"A.txt": []byte("a"), "a.txt": []byte("b")}, map[string]string{"A.txt": "100644", "a.txt": "100644"}},
		{"mode mismatch", map[string][]byte{"evidence.txt": []byte("x")}, map[string]string{"other.txt": "100644"}},
		{"unsupported mode", map[string][]byte{"evidence.txt": []byte("x")}, map[string]string{"evidence.txt": "120000"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := gitTest(t, root, "count-objects", "-v")
			if _, err := WriteSelectedEvidenceCommit(root, parent, tc.files, tc.modes, true); err == nil {
				t.Fatal("invalid input was accepted")
			}
			if after := gitTest(t, root, "count-objects", "-v"); after != before {
				t.Fatalf("validation wrote objects: before=%q after=%q", before, after)
			}
		})
	}
}

func TestWriteSelectedEvidenceCommitRejectsUnselectedAliasesAndPrefixCollisionsBeforeObjects(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "dir/keep.txt", "keep")
	writeTestFile(t, root, "alias.txt", "alias")
	gitTest(t, root, "add", "dir/keep.txt", "alias.txt")
	gitTest(t, root, "commit", "-qm", "tree entries")
	parent := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	for _, path := range []string{"DIR/new.txt", "ALIAS.txt", "dir"} {
		before := gitTest(t, root, "count-objects", "-v")
		if _, err := WriteSelectedEvidenceCommit(root, parent, map[string][]byte{path: []byte("x")}, map[string]string{path: "100644"}, true); err == nil {
			t.Errorf("accepted path collision %q", path)
		}
		if after := gitTest(t, root, "count-objects", "-v"); after != before {
			t.Errorf("collision check wrote objects for %q", path)
		}
	}
}

func TestWriteSelectedEvidenceCommitRefusesSymlinkAndGitlinkReplacements(t *testing.T) {
	for _, tc := range []struct{ name, mode, path string }{{"symlink", "120000", "link.txt"}, {"gitlink", "160000", "module"}} {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := selectiveGitFixture(t)
			oid := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD:seed.txt"))
			if tc.mode == "160000" {
				oid = strings.Repeat("a", 40)
			}
			gitTest(t, root, "update-index", "--add", "--cacheinfo", tc.mode+","+oid+","+tc.path)
			gitTest(t, root, "commit", "-qm", "non-regular tree entry")
			parent := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
			before := gitTest(t, root, "count-objects", "-v")
			if _, err := WriteSelectedEvidenceCommit(root, parent, map[string][]byte{tc.path: []byte("replacement")}, map[string]string{tc.path: "100644"}, true); err == nil {
				t.Fatal("non-regular selected path was accepted")
			}
			if after := gitTest(t, root, "count-objects", "-v"); after != before {
				t.Fatal("non-regular path rejection wrote Git objects")
			}
		})
	}
}

func TestReadEvidenceTreeUsesMetadataWithoutBlobSizes(t *testing.T) {
	const commit = "0123456789012345678901234567890123456789"
	const blob = "abcdefabcdefabcdefabcdefabcdefabcdefabcd"
	var gotArgs []string
	run := func(root string, args ...string) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		return []byte("100755 blob " + blob + "\ttool.sh\x00"), nil
	}
	tree, err := readEvidenceTreeWith("repo", commit, run)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotArgs, []string{"ls-tree", "-r", "-z", "--full-tree", commit}) {
		t.Fatalf("tree metadata command = %#v", gotArgs)
	}
	if len(tree) != 1 || tree["tool.sh"].mode != "100755" || tree["tool.sh"].kind != "blob" || tree["tool.sh"].oid != blob {
		t.Fatalf("parsed tree metadata = %#v", tree)
	}
}

func TestReadEvidenceTreeEnforcesByteAndEntryBoundsDuringParsing(t *testing.T) {
	const blob = "abcdefabcdefabcdefabcdefabcdefabcdefabcd"
	record := []byte("100644 blob " + blob + "\tfile.txt\x00")
	secondRecord := []byte("100644 blob " + blob + "\tother.txt\x00")
	if _, err := readEvidenceTreeRecordsWithLimits(bytes.NewReader(record), int64(len(record)-1), 10); err == nil || !strings.Contains(err.Error(), "byte limit of") {
		t.Fatalf("byte-bound error = %v", err)
	}
	if _, err := readEvidenceTreeRecordsWithLimits(bytes.NewReader(append(record, secondRecord...)), int64(len(record)+len(secondRecord)), 1); err == nil || !strings.Contains(err.Error(), "entry limit of 1") {
		t.Fatalf("entry-bound error = %v", err)
	}
}

func TestWriteSelectedEvidenceCommitAllowsMissingUnselectedBlob(t *testing.T) {
	root, parent := partialCloneFixture(t)
	commit, err := WriteSelectedEvidenceCommit(root, parent, map[string][]byte{"selected.txt": []byte("replacement")}, map[string]string{"selected.txt": "100644"}, true)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := LoadSelected(root, commit, []string{"selected.txt"})
	if err != nil || string(selected.Snapshot.Files["selected.txt"]) != "replacement" {
		t.Fatalf("selected replacement = %#v, err=%v", selected, err)
	}
	missing := gitTest(t, root, "rev-list", "--objects", "--missing=print", commit)
	entry := strings.Fields(gitTest(t, root, "ls-tree", parent, "--", "unselected.txt"))
	if len(entry) < 3 {
		t.Fatalf("unselected tree metadata = %#v", entry)
	}
	if !strings.Contains(missing, "?"+entry[2]) {
		t.Fatal("unselected missing blob was fetched")
	}
}

func TestWriteSelectedEvidenceCommitSupportsLinkedWorktreeAndRejectsStaleHead(t *testing.T) {
	root, parent := selectiveGitFixture(t)
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "--detach", linked, parent)
	if _, err := WriteSelectedEvidenceCommit(linked, parent, map[string][]byte{"evidence.txt": []byte("linked")}, map[string]string{"evidence.txt": "100644"}, true); err != nil {
		t.Fatalf("linked worktree: %v", err)
	}
	writeTestFile(t, root, "next.txt", "advance")
	gitTest(t, root, "add", "next.txt")
	gitTest(t, root, "commit", "-qm", "advance HEAD")
	before := gitTest(t, root, "count-objects", "-v")
	if _, err := WriteSelectedEvidenceCommit(root, parent, map[string][]byte{"stale.txt": []byte("x")}, map[string]string{"stale.txt": "100644"}, true); err == nil || !strings.Contains(err.Error(), "HEAD") {
		t.Fatalf("stale parent error = %v", err)
	}
	if after := gitTest(t, root, "count-objects", "-v"); after != before {
		t.Fatal("stale-HEAD failure wrote objects")
	}
}

func TestEvidenceGitEnvironmentUsesPlatformNullDevice(t *testing.T) {
	values := make(map[string]string)
	for _, entry := range evidenceGitEnvironment(filepath.Join(t.TempDir(), "index")) {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	if got := values["GIT_CONFIG_GLOBAL"]; got != os.DevNull {
		t.Fatalf("GIT_CONFIG_GLOBAL = %q, want platform null device %q", got, os.DevNull)
	}
	if got := values["GIT_CONFIG_NOSYSTEM"]; got != "1" {
		t.Fatalf("GIT_CONFIG_NOSYSTEM = %q, want 1", got)
	}
}

func TestWriteSelectedEvidenceCommitHonorsCancellation(t *testing.T) {
	root, parent := selectiveGitFixture(t)
	before := evidenceRepoState(t, root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := WriteSelectedEvidenceCommitContext(ctx, root, parent, map[string][]byte{"evidence.txt": []byte("x")}, map[string]string{"evidence.txt": "100644"}, true)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled operation error = %v", err)
	}
	if after := evidenceRepoState(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("canceled operation changed state: before=%v after=%v", before, after)
	}
}

func evidenceRepoState(t *testing.T, root string) map[string]string {
	t.Helper()
	state := map[string]string{"head": strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD")), "refs": gitTest(t, root, "show-ref", "--head")}
	headPath := strings.TrimSpace(gitTest(t, root, "rev-parse", "--path-format=absolute", "--git-path", "HEAD"))
	headBytes, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatal(err)
	}
	headDigest := sha256.Sum256(headBytes)
	state["head-file"] = hex.EncodeToString(headDigest[:])
	indexPath := strings.TrimSpace(gitTest(t, root, "rev-parse", "--path-format=absolute", "--git-path", "index"))
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	state["index"] = hex.EncodeToString(digest[:])
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == filepath.Join(root, ".git") {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(contents)
		state["file:"+filepath.ToSlash(strings.TrimPrefix(path, root+string(os.PathSeparator)))] = hex.EncodeToString(hash[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}
