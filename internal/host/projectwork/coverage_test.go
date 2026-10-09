package projectwork

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectcoverage"
)

func TestFullCoverageBindsUnknownOutsideLegacyRoots(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "notes/unknown.md", "unclassified repository file\n")
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if project.Coverage == nil || project.Coverage.Accounted || project.Coverage.Conforming {
		t.Fatalf("full coverage did not expose unknown outside-root path: %+v", project.Coverage)
	}
	found := false
	for _, entry := range project.Coverage.Entries {
		if entry.Path == "notes/unknown.md" {
			found = true
		}
	}
	if !found {
		t.Fatal("full census omitted untracked path outside InventoryRoots")
	}
}

func TestOperationalRunBytesDoNotChangeStableProjectBinding(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	before, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/runs/run-1/state.json", "first\n")
	after, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/runs/run-1/state.json", "changed\n")
	changed, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest != after.Digest || before.Digest != changed.Digest {
		t.Fatalf("operational run state changed stable binding: %s, %s, %s", before.Digest, after.Digest, changed.Digest)
	}
}

func TestDraftProposalBytesDoNotInvalidateTheirProjectBasis(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	before, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/drafts/proposal.json", "first reviewed proposal\n")
	after, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/drafts/proposal.json", "updated reviewed proposal\n")
	changed, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest != after.Digest || before.Digest != changed.Digest {
		t.Fatalf("draft proposal bytes invalidated the stable project basis: %s, %s, %s", before.Digest, after.Digest, changed.Digest)
	}
	var classified bool
	for _, entry := range changed.Coverage.Entries {
		if entry.Path == ".markitect/drafts/proposal.json" {
			classified = entry.Class == projectcoverage.ClassToolOwned && entry.Operational
		}
	}
	if !classified {
		t.Fatal("draft proposal was not retained as explicitly classified operational tool state")
	}
}

func TestIgnoredBytesAreUnboundButMembershipIsBound(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	ignore := "apiVersion: " + projectcoverage.IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries:\n  - path: scratch/\n    reason: Temporary local outputs\n"
	writeFile(t, root, projectcoverage.IgnorePath, ignore)
	writeFile(t, root, "scratch/output.bin", "first bytes\n")
	first, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "scratch/output.bin", "different bytes\n")
	second, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("ignored bytes changed project digest: %s != %s", first.Digest, second.Digest)
	}
	if err := os.Remove(filepath.Join(root, "scratch", "output.bin")); err != nil {
		t.Fatal(err)
	}
	third, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if third.Digest == second.Digest {
		t.Fatal("ignored path membership change did not stale project digest")
	}
}
