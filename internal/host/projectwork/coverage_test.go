package projectwork

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectcoverage"
)

func TestNativeSkillPathsHaveExactToolOwnership(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		paths := NativeSkillPaths(provider)
		if len(paths) != 1+2*len(NativeSkillNames()) {
			t.Fatalf("incomplete %s native paths", provider)
		}
		for _, path := range paths {
			if !IsToolPath(Config{}, path) {
				t.Fatalf("generated file lacks ownership: %s", path)
			}
		}
	}
	for _, path := range []string{".agents/skills/markitect-model-first/SKILL.md", ".claude/skills/markitect-model-first/SKILL.md"} {
		if IsToolPath(Config{}, path) {
			t.Fatalf("obsolete router path remains registered as an owned output: %s", path)
		}
	}
	for _, path := range []string{".agents/skills/custom/SKILL.md", ".agents/skills/markitect-implement/custom.md", ".claude/skills/markitect-check/private.md"} {
		if IsToolPath(Config{}, path) {
			t.Fatalf("unmanaged skill claimed by Markitect: %s", path)
		}
	}
}

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

func TestTransitionalExclusionIsAccountedButBlocksConformance(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, filepath.FromSlash(ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "documentPath: "+DefaultDocumentPath+"\n", "documentPath: README.md\n", 1)
	updated = strings.Replace(updated, "transitionalExclusions: []\n", "transitionalExclusions:\n  - path: "+DefaultDocumentPath+"\n    reason: Prior generated view awaits migration\n  - path: legacy/\n    reason: Existing file awaits explicit modeling\n", 1)
	writeFile(t, root, ManifestPath, updated)
	writeFile(t, root, "legacy/unmodeled.txt", "tracked by the migration boundary\n")
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if project.Coverage == nil || !project.Coverage.Accounted || project.Coverage.Conforming {
		t.Fatalf("transitional path must be accounted but nonconforming: %+v", project.Coverage)
	}
	for _, entry := range project.Coverage.Entries {
		if entry.Path == "legacy/unmodeled.txt" {
			if entry.Class != projectcoverage.ClassTransitional || entry.Reason != "Existing file awaits explicit modeling" {
				t.Fatalf("transitional path classification = %+v", entry)
			}
			return
		}
	}
	t.Fatal("coverage omitted transitional path")
}

func TestExplorationOperationalRegistrationIsExact(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	before, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/state/explorations/readiness.json", "operational exploration state\n")
	withExploration, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest != withExploration.Digest {
		t.Fatalf("registered operational exploration bytes changed the project binding: %s != %s", before.Digest, withExploration.Digest)
	}
	var explorationRegistered bool
	for _, entry := range withExploration.Coverage.Entries {
		if entry.Path == ".markitect/state/explorations/readiness.json" {
			explorationRegistered = entry.Class == projectcoverage.ClassToolOwned && entry.Operational
		}
	}
	if !explorationRegistered {
		t.Fatal("exploration state was not classified as operational tool-owned state")
	}
	writeFile(t, root, ".markitect/state/unregistered.json", "must remain visible\n")
	withUnknown, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if withUnknown.Coverage.Accounted || withUnknown.Coverage.Conforming {
		t.Fatal("registering exploration state made arbitrary Markitect state conforming")
	}
	for _, entry := range withUnknown.Coverage.Entries {
		if entry.Path == ".markitect/state/unregistered.json" && entry.Class != "" {
			t.Fatalf("unregistered state was silently assigned class %q", entry.Class)
		}
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
