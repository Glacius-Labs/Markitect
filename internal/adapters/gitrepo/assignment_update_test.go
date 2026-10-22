package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func warmGitCopy(t *testing.T, w *Workspaces, source *assignmentWorkspace, id run.WorkspaceID) (*assignmentWorkspace, string) {
	t.Helper()
	root := t.TempDir()
	volume := w.volume(id)
	w.runtime.(*integrationRuntime).roots[volume], w.owned[id] = root, "owned"
	copy := &assignmentWorkspace{owner: w, id: id, volume: volume}
	if err := w.cloneAssignment(context.Background(), source, copy); err != nil {
		t.Fatal(err)
	}
	var err error
	copy.start, err = copy.git(context.Background(), "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return copy, root
}

func TestAssignmentUpdateSwitchesCleanWarmCopyToExactParentCommit(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "parent")
	parent := copies["parent"]
	child, root := warmGitCopy(t, w, parent, "child")
	initial, volume := child.StartingCommit(), child.Volume()
	integrationWrite(t, roots["parent"], "interface.txt", "parent Go interface\n")
	stand, err := parent.Commit(ctx, "Parent Go")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.UpdateAssignment(ctx, child, parent, stand, initial); err != nil {
		t.Fatal(err)
	}
	head, err := child.git(ctx, "rev-parse", "HEAD")
	if err != nil || head != stand || child.StartingCommit() != initial || child.Volume() != volume {
		t.Fatalf("exact switch replaced identity or missed stand: %s %v", head, err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "interface.txt")); err != nil || string(got) != "parent Go interface\n" {
		t.Fatalf("first task lacks parent interface: %q %v", got, err)
	}
	if err := w.UpdateAssignment(ctx, child, parent, strings.Repeat("0", 40), stand); err == nil {
		t.Fatal("updated from an unavailable parent commit")
	}
	if after, _ := child.git(ctx, "rev-parse", "HEAD"); after != stand {
		t.Fatal("invalid source changed child")
	}
	if _, err := child.git(ctx, "checkout", "--detach", initial); err != nil {
		t.Fatal(err)
	}
	if err := w.UpdateAssignment(ctx, child, parent, stand, stand); err == nil || !strings.Contains(err.Error(), "no longer descends") {
		t.Fatalf("stale child HEAD was treated as own work: %v", err)
	}
	if after, _ := child.git(ctx, "rev-parse", "HEAD"); after != initial {
		t.Fatal("stale-HEAD refusal changed the child")
	}
}

func TestAssignmentUpdatePreservesOwnDirtyWorkAndLeavesRealConflictsForChild(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(map[bool]string{false: "merge", true: "conflict"}[conflict], func(t *testing.T) {
			ctx := context.Background()
			w, copies, roots := integrationFixture(t, "parent")
			parent := copies["parent"]
			child, root := warmGitCopy(t, w, parent, "child")
			initial := child.StartingCommit()
			integrationWrite(t, root, "own.txt", "uncommitted child work\n")
			integrationWrite(t, roots["parent"], "sibling.txt", "reviewed sibling result\n")
			if conflict {
				integrationWrite(t, root, "common.txt", "child decision\n")
				integrationWrite(t, roots["parent"], "common.txt", "parent decision\n")
			}
			stand, err := parent.Commit(ctx, "Integrated parent stand")
			if err != nil {
				t.Fatal(err)
			}
			err = w.UpdateAssignment(ctx, child, parent, stand, initial)
			if conflict {
				if !errors.Is(err, run.ErrAssignmentUpdateConflict) {
					t.Fatalf("real conflict not distinguished: %v", err)
				}
				unmerged, _ := child.git(ctx, "diff", "--name-only", "--diff-filter=U")
				if !strings.Contains(unmerged, "common.txt") {
					t.Fatal("Host removed the child's conflict")
				}
				integrationWrite(t, root, "common.txt", "resolved by child\n")
			} else if err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(filepath.Join(root, "own.txt")); err != nil || string(got) != "uncommitted child work\n" {
				t.Fatalf("dirty own work lost: %q %v", got, err)
			}
			if got, err := os.ReadFile(filepath.Join(root, "sibling.txt")); err != nil || string(got) != "reviewed sibling result\n" {
				t.Fatalf("sibling result missing: %q %v", got, err)
			}
			result, err := child.Commit(ctx, "Child completed updated task")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := child.git(ctx, "merge-base", "--is-ancestor", stand, result); err != nil {
				t.Fatal("resolved child result does not contain delivered parent")
			}
			if child.StartingCommit() != initial {
				t.Fatal("initial copy identity was rewritten")
			}
			if _, err := w.Changes(ctx, child, run.AssignmentHandoff{StartingCommit: stand, ResultCommit: result}); err != nil {
				t.Fatalf("updated task base cannot be reviewed: %v", err)
			}
		})
	}
}

func TestAssignmentUpdateBindsEachTaskAfterSiblingIntegrationWithoutReapplyingOldResult(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "parent")
	parent := copies["parent"]
	child, childRoot := warmGitCopy(t, w, parent, "child")
	sibling, siblingRoot := warmGitCopy(t, w, parent, "sibling")
	integrationWrite(t, roots["parent"], "interface.txt", "initial interface\n")
	firstBase, _ := parent.Commit(ctx, "First Go")
	if err := w.UpdateAssignment(ctx, child, parent, firstBase, child.StartingCommit()); err != nil {
		t.Fatal(err)
	}
	integrationWrite(t, childRoot, "lending.txt", "first task\n")
	firstResult, _ := child.Commit(ctx, "First child task")
	first := run.AssignmentHandoff{StartingCommit: firstBase, ResultCommit: firstResult, Summary: "first child result"}
	firstDiff, err := w.Changes(ctx, child, first)
	if err != nil {
		t.Fatal(err)
	}
	afterChild, err := w.Integrate(ctx, parent, child, first, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.UpdateAssignment(ctx, sibling, parent, afterChild, sibling.StartingCommit()); err != nil {
		t.Fatal(err)
	}
	integrationWrite(t, siblingRoot, "catalog.txt", "reviewed catalog contract\n")
	siblingResult, _ := sibling.Commit(ctx, "Catalog result")
	secondBase, err := w.Integrate(ctx, parent, sibling, run.AssignmentHandoff{StartingCommit: afterChild, ResultCommit: siblingResult}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.UpdateAssignment(ctx, child, parent, secondBase, firstBase); err != nil {
		t.Fatal(err)
	}
	integrationWrite(t, childRoot, "lending.txt", "uses the reviewed catalog contract\n")
	secondResult, _ := child.Commit(ctx, "Later targeted child task")
	second := run.AssignmentHandoff{StartingCommit: secondBase, ResultCommit: secondResult}
	diff, err := w.Changes(ctx, child, second)
	if err != nil || strings.Contains(diff, "diff --git a/catalog.txt") || strings.Contains(diff, "diff --git a/interface.txt") || !strings.Contains(diff, "uses the reviewed") {
		t.Fatalf("new task diff used stale root/child basis: %s %v", diff, err)
	}
	if _, err := w.Integrate(ctx, parent, child, second, ""); err != nil {
		t.Fatalf("old accepted result was reapplied after refresh: %v", err)
	}
	if retained, err := w.Changes(ctx, child, first); err != nil || retained != firstDiff {
		t.Fatalf("earlier fixed result/base changed: %s %v", retained, err)
	}
}
