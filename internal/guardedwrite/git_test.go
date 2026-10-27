package guardedwrite

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

var batchedGitIdentityArgs = []string{
	"rev-parse", "--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir", "--show-object-format",
}

func identityGitFixture(t *testing.T) (*testkit.Repo, string) {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Write("seed.txt", "seed")
	return repo, repo.Commit("seed")
}

// samePath reports whether two spellings name the same path.
func samePath(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	if leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo) {
		return true
	}
	if filepath.VolumeName(left) != "" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func TestIdentifyGitUsesOneMetadataQuery(t *testing.T) {
	repo, _ := identityGitFixture(t)
	root := repo.Dir
	var identifyCalls [][]string
	run := func(repo string, args ...string) ([]byte, error) {
		if reflect.DeepEqual(args, batchedGitIdentityArgs) {
			identifyCalls = append(identifyCalls, append([]string(nil), args...))
		}
		return GitOutput(repo, args...)
	}

	identity, err := identifyGit(root, run)
	if err != nil {
		t.Fatal(err)
	}
	if len(identifyCalls) != 1 {
		t.Fatalf("identity metadata queries = %d, want one combined rev-parse", len(identifyCalls))
	}
	if !samePath(identity.Root, root) || identity.ObjectFormat != "sha1" || identity.GitDir == "" || identity.CommonDir == "" {
		t.Fatalf("combined identity = %#v", identity)
	}
}

func TestIdentifyGitFallsBackWhenCombinedPathsContainNewlines(t *testing.T) {
	repo, _ := identityGitFixture(t)
	root := repo.Dir
	var combinedCalls, individualCalls int
	run := func(repo string, args ...string) ([]byte, error) {
		if reflect.DeepEqual(args, batchedGitIdentityArgs) {
			combinedCalls++
			output, err := GitOutput(repo, args...)
			if err != nil {
				return nil, err
			}
			// Simulate a path containing a newline in the first returned field.
			firstLine := bytes.IndexByte(output, '\n')
			output = append(output[:firstLine+1], append([]byte("embedded\n"), output[firstLine+1:]...)...)
			return output, nil
		} else if len(args) > 0 && args[0] == "rev-parse" {
			individualCalls++
		}
		return GitOutput(repo, args...)
	}
	identity, err := identifyGit(root, run)
	if err != nil {
		t.Fatal(err)
	}
	if combinedCalls != 1 || individualCalls != 4 {
		t.Fatalf("combined/legacy identity queries = %d/%d, want one ambiguous combined attempt and four legacy queries", combinedCalls, individualCalls)
	}
	if !samePath(identity.Root, root) {
		t.Fatalf("newline path spelling was not preserved by legacy fallback: got %q want %q", identity.Root, root)
	}
}

func TestIdentifyGitPreservesSHA256ObjectFormat(t *testing.T) {
	// Run git directly: a Git without SHA-256 support skips the test rather
	// than failing it.
	root := testkit.TempDir(t)
	cmd := exec.Command("git", "init", "-q", "--object-format=sha256", root)
	cmd.Env = testkit.GitEnv()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("installed Git cannot create SHA-256 repositories: %v (%s)", err, output)
	}
	identity, err := IdentifyGitContext(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ObjectFormat != "sha256" {
		t.Fatalf("object format = %q, want sha256", identity.ObjectFormat)
	}
}

func TestIdentifyGitRejectsNestedRootAndIdentifiesWorktree(t *testing.T) {
	repo, head := identityGitFixture(t)
	root := repo.Dir
	ctx := context.Background()
	if _, err := IdentifyGitContext(ctx, filepath.Join(root, "nested")); err == nil {
		t.Fatal("nested non-repository root was accepted")
	}
	identity, err := IdentifyGitContext(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(identity.Root, root) || identity.GitDir == "" || identity.CommonDir == "" || identity.Digest != gitIdentityDigest(identity) {
		t.Fatalf("identity = %#v expectedRoot=%q rootSame=%v digestSame=%v", identity, root, samePath(identity.Root, root), identity.Digest == gitIdentityDigest(identity))
	}
	clone := testkit.TempDir(t)
	repo.Git("clone", "--quiet", root, clone)
	cloneIdentity, err := IdentifyGitContext(ctx, clone)
	if err != nil {
		t.Fatal(err)
	}
	if cloneIdentity.Digest == identity.Digest {
		t.Fatal("independent clone has same repository-location identity digest")
	}
	linked := filepath.Join(testkit.TempDir(t), "linked")
	repo.Git("worktree", "add", "--detach", linked, head)
	linkedIdentity, err := IdentifyGitContext(ctx, linked)
	if err != nil {
		t.Fatal(err)
	}
	if samePath(linkedIdentity.GitDir, identity.GitDir) || !samePath(linkedIdentity.CommonDir, identity.CommonDir) || linkedIdentity.Digest == identity.Digest {
		t.Fatalf("linked worktree identity = %#v; main = %#v", linkedIdentity, identity)
	}
}

func TestIdentityGitEnvironmentDisablesLazyFetchAndWrites(t *testing.T) {
	values := make(map[string]string)
	for _, entry := range identityGitEnvironment() {
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
			t.Errorf("identity environment %s = %q (present %v), want %q", key, got, ok, want)
		}
	}
}

func TestCleanGitEnvDropsGitVariablesCaseInsensitively(t *testing.T) {
	t.Setenv("gIt_DiR", "foreign")
	t.Setenv("GIT_CONFIG_PARAMETERS", "injected")
	t.Setenv("MARKITECT_TEST_VALUE", "preserved")
	values := make(map[string]string)
	for _, entry := range CleanGitEnv() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[strings.ToUpper(key)] = value
		}
	}
	for key := range values {
		if strings.HasPrefix(key, "GIT_") {
			t.Errorf("Git variable %s survived environment cleanup", key)
		}
	}
	if values["MARKITECT_TEST_VALUE"] != "preserved" {
		t.Fatal("non-Git environment variable was not preserved")
	}
}
