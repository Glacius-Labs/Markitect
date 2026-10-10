package source

import (
	"bytes"
	"os/exec"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

var batchedGitIdentityArgs = []string{
	"rev-parse", "--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir", "--show-object-format",
}

func TestIdentifyGitUsesOneMetadataQueryAndRechecksAroundLoad(t *testing.T) {
	repo, revision := selectiveGitFixture(t)
	root := repo.Dir
	var identifyCalls [][]string
	run := func(repo string, args ...string) ([]byte, error) {
		if reflect.DeepEqual(args, batchedGitIdentityArgs) {
			identifyCalls = append(identifyCalls, append([]string(nil), args...))
		}
		return GitOutput(repo, args...)
	}

	identity, _, err := identifyGit(root, run)
	if err != nil {
		t.Fatal(err)
	}
	if len(identifyCalls) != 1 {
		t.Fatalf("identity metadata queries = %d, want one combined rev-parse", len(identifyCalls))
	}
	if !samePath(identity.Root, root) || identity.ObjectFormat != "sha1" || identity.GitDir == "" || identity.CommonDir == "" {
		t.Fatalf("combined identity = %#v", identity)
	}

	identifyCalls = nil
	selected, err := loadSelected(root, revision, []string{"seed.txt"}, run, readSelectedBlobs)
	if err != nil {
		t.Fatal(err)
	}
	if len(identifyCalls) != 2 {
		t.Fatalf("identity metadata queries around selected load = %d, want initial and final recheck", len(identifyCalls))
	}
	if len(selected.Snapshot.Files) != 1 || string(selected.Snapshot.Files["seed.txt"]) != "seed" {
		t.Fatalf("selected snapshot = %#v; batching must not broaden selected content", selected.Snapshot.Files)
	}
}

func TestIdentifyGitFallsBackWhenCombinedPathsContainNewlines(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
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
	identity, _, err := identifyGit(root, run)
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
	identity, err := IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ObjectFormat != "sha256" {
		t.Fatalf("object format = %q, want sha256", identity.ObjectFormat)
	}
}
