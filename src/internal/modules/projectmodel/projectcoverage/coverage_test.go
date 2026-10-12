package projectcoverage

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestUnknownOutsideLegacyRootsIsNotAccounted(t *testing.T) {
	universe := testUniverse(PathState{Path: "docs/new.md", Worktree: FileState{Present: true, Mode: "100644", Digest: "abc"}})
	report, err := Classify(Request{Universe: universe, Model: emptyModel(), LegacyRoots: []string{"src"}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accounted || report.Conforming || len(report.LegacyDiagnostics) != 1 {
		t.Fatalf("report = %+v; legacy roots must not claim coverage", report)
	}
}

func TestFromSnapshotAccountsOnlySuppliedCandidatePathsWithoutIO(t *testing.T) {
	s := &snapshot.Snapshot{ID: "candidate", Provisional: true, Files: map[string][]byte{
		"outside/unknown.txt":    []byte("candidate bytes"),
		".markitect/custom.yaml": []byte("unregistered control data"),
	}, Modes: map[string]string{"outside/unknown.txt": "100644", ".markitect/custom.yaml": "100644"}}
	universe, err := FromSnapshot(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	report, err := Classify(Request{Universe: universe, Model: emptyModel(), LegacyRoots: []string{"src"}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accounted || len(report.Entries) != 2 {
		t.Fatalf("snapshot candidate hid unclassified files: %+v", report)
	}
}

func TestFromSnapshotCannotAcceptDeletedRequiredArtifact(t *testing.T) {
	s := &snapshot.Snapshot{ID: "candidate", Provisional: true, Files: map[string][]byte{
		IgnorePath: []byte(emptyIgnoreYAML),
	}, Modes: map[string]string{IgnorePath: "100644"}}
	universe, err := FromSnapshot(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	report, err := Classify(Request{Universe: universe, Model: modelWithRequiredArtifact("src/required.go")})
	if err != nil {
		t.Fatal(err)
	}
	if report.Conforming || !hasFinding(report, "coverage.required-artifact-missing") {
		t.Fatalf("candidate without required realization was accepted: %+v", report)
	}
}

func TestTransitionalExclusionIsVisibleButLegacyExclusionIsDiagnosticOnly(t *testing.T) {
	state := PathState{Path: "legacy/module.go", Worktree: FileState{Present: true, Mode: "100644", Digest: "abc"}}
	transitional, err := Classify(Request{Universe: testUniverse(state), Model: emptyModel(), Options: Options{
		Transitional: []TransitionalExclusion{{Path: "legacy/", Reason: "Awaiting domain mapping"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !transitional.Accounted || transitional.Conforming || transitional.Entries[0].Class != ClassTransitional {
		t.Fatalf("transitional report=%+v", transitional)
	}
	legacy, err := Classify(Request{Universe: testUniverse(state), Model: emptyModel(), LegacyExclusions: []TransitionalExclusion{{Path: "legacy/", Reason: "old selection"}}})
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Accounted || len(legacy.LegacyDiagnostics) != 1 || legacy.Entries[0].Class != "" {
		t.Fatalf("legacy exclusion implied coverage: %+v", legacy)
	}
}

func TestCandidateIgnorePolicyChangeBindsBytesAndMembers(t *testing.T) {
	universe := testUniverse(
		PathState{Path: IgnorePath, Worktree: FileState{Present: true, Mode: "100644", Digest: "old"}},
		PathState{Path: "build/output.bin", Worktree: FileState{Present: true, Mode: "100644", Digest: "file-hash"}},
	)
	before, err := Classify(Request{Universe: universe, Model: emptyModel()})
	if err != nil || before.Accounted {
		t.Fatalf("before report=%+v err=%v", before, err)
	}
	newIgnore := []byte("apiVersion: " + IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries:\n  - path: build/\n    reason: Generated build output\n")
	candidate, err := ValidateCandidate(universe, []Delta{{Path: IgnorePath, Mode: "100644", Content: newIgnore}}, emptyModel(), Options{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !candidate.Accounted || !candidate.Conforming || candidate.IgnoreBytesDigest == before.IgnoreBytesDigest || candidate.IgnoreMembersDigest == before.IgnoreMembersDigest || candidate.Digest == before.Digest {
		t.Fatalf("candidate did not bind ignore bytes and membership: before=%+v after=%+v", before, candidate)
	}
	changedIgnoredBytes := testUniverse(
		PathState{Path: IgnorePath, Worktree: FileState{Present: true, Mode: "100644", Digest: sha256Hex(newIgnore)}},
		PathState{Path: "build/output.bin", Worktree: FileState{Present: true, Mode: "100644", Digest: "different-ignored-file-hash"}},
	)
	changedIgnoredBytes.IgnoreBytes = newIgnore
	changedIgnoredBytes.IgnoreBytesDigest = sha256Hex(newIgnore)
	unchangedCoverage, err := Classify(Request{Universe: changedIgnoredBytes, Model: emptyModel()})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Digest != unchangedCoverage.Digest {
		t.Fatalf("ignored file content changed coverage digest: %s != %s", candidate.Digest, unchangedCoverage.Digest)
	}
}

func TestRequiredArtifactCannotBeHiddenOrDeleted(t *testing.T) {
	model := modelWithRequiredArtifact("src/required.go")
	universe := testUniverse(PathState{Path: "src/required.go", Worktree: FileState{Present: true, Mode: "100644", Digest: "old"}})
	base, err := Classify(Request{Universe: universe, Model: model})
	if err != nil || !base.Conforming {
		t.Fatalf("base report=%+v err=%v", base, err)
	}
	candidate, err := ValidateCandidate(universe, []Delta{{Path: "src/required.go", Delete: true}}, model, Options{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Conforming || !hasFinding(candidate, "coverage.required-artifact-missing") {
		t.Fatalf("required artifact deletion was accepted: %+v", candidate)
	}
	ignore := []byte("apiVersion: " + IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries:\n  - path: src/required.go\n    reason: hidden\n")
	_, err = ValidateCandidate(universe, []Delta{{Path: IgnorePath, Mode: "100644", Content: ignore}}, model, Options{}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "required Artifact") {
		t.Fatalf("required artifact ignore error = %v", err)
	}
}

func TestCandidateDeleteLeavesTheUniverse(t *testing.T) {
	universe := testUniverse(PathState{Path: "src/old.go", Head: FileState{Present: true, Mode: "100644", Digest: "old"}})
	universe.FixedRevision = true
	candidate, err := ValidateCandidate(universe, []Delta{{Path: "src/old.go", Delete: true}, {Path: "src/never.go", Delete: true}}, emptyModel(), Options{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !candidate.Accounted || !candidate.Conforming || len(candidate.Entries) != 0 {
		t.Fatalf("deleted paths remained in the candidate universe: %+v", candidate)
	}
}

func TestUnboundPathsListOnlyNewlyAdmittedUnreadFiles(t *testing.T) {
	universe := testUniverse(
		PathState{Path: "legacy/old.txt", Worktree: FileState{Present: true, Mode: "100644"}},
		PathState{Path: "src/read.go", Worktree: FileState{Present: true, Mode: "100644", Digest: "read"}},
		PathState{Path: "src/gone.go", Head: FileState{Present: true, Mode: "100644"}},
	)
	transitional := Options{Transitional: []TransitionalExclusion{{Path: "legacy/", Reason: "awaits modeling"}}}
	if paths, err := UnboundPaths(universe, transitional); err != nil || len(paths) != 0 {
		t.Fatalf("excluded or read files listed as unbound: %v %v", paths, err)
	}
	if paths, err := UnboundPaths(universe, Options{}); err != nil || len(paths) != 1 || paths[0] != "legacy/old.txt" {
		t.Fatalf("newly admitted unread file = %v %v", paths, err)
	}
}

func TestOperationalToolStateIsExplicitAndDoesNotChangeStableDigest(t *testing.T) {
	options := Options{ToolPaths: []ToolPath{{Selector: ".markitect/runs/", Owner: "projectrun", Operational: true}}}
	first := testUniverse(PathState{Path: ".markitect/runs/run-1/state.json", Worktree: FileState{Present: true, Mode: "100644"}})
	second := testUniverse(
		PathState{Path: ".markitect/runs/run-1/state.json", Worktree: FileState{Present: true, Mode: "100644"}},
		PathState{Path: ".markitect/runs/run-1/receipt.json", Worktree: FileState{Present: true, Mode: "100644"}},
	)
	firstReport, err := Classify(Request{Universe: first, Model: emptyModel(), Options: options})
	if err != nil {
		t.Fatal(err)
	}
	secondReport, err := Classify(Request{Universe: second, Model: emptyModel(), Options: options})
	if err != nil {
		t.Fatal(err)
	}
	if firstReport.Digest != secondReport.Digest || firstReport.Entries[0].Class != ClassToolOwned || !firstReport.Entries[0].Operational {
		t.Fatalf("operational run files changed stable coverage: first=%+v second=%+v", firstReport, secondReport)
	}
	unknown := testUniverse(PathState{Path: ".markitect/custom.yaml", Worktree: FileState{Present: true, Mode: "100644", Digest: "x"}})
	unknownReport, err := Classify(Request{Universe: unknown, Model: emptyModel(), Options: options})
	if err != nil {
		t.Fatal(err)
	}
	if unknownReport.Accounted {
		t.Fatal("unregistered .markitect file was implicitly tool-owned")
	}
}

func TestDecodeIgnoreRejectsPatternsAndAliases(t *testing.T) {
	for _, invalid := range []string{
		"apiVersion: " + IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries:\n  - path: '*.go'\n    reason: wildcard\n",
		"apiVersion: " + IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries:\n  - path: src/\n    reason: one\n  - path: src/a.go\n    reason: overlap\n",
	} {
		if _, err := DecodeIgnore([]byte(invalid)); err == nil {
			t.Fatalf("accepted invalid ignore config:\n%s", invalid)
		}
	}
}

func TestDecodeIgnoreOverlapNamesFirstEarlierEntry(t *testing.T) {
	data := []byte("apiVersion: " + IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries:\n" +
		"  - path: a/x\n    reason: r\n  - path: a/y\n    reason: r\n  - path: a/z\n    reason: r\n  - path: a/\n    reason: r\n")
	for i := 0; i < 100; i++ {
		_, err := DecodeIgnore(data)
		if err == nil || !strings.HasSuffix(err.Error(), `entries "a/x" and "a/" overlap`) {
			t.Fatalf("run %d: overlap error=%v, want the first earlier entry a/x", i, err)
		}
	}
}

func TestValidateOptionsOverlapNamesFirstEarlierSelector(t *testing.T) {
	options := Options{Transitional: []TransitionalExclusion{
		{Path: "a/x", Reason: "r"}, {Path: "a/y", Reason: "r"}, {Path: "a/z", Reason: "r"}, {Path: "a/", Reason: "r"},
	}}
	for i := 0; i < 100; i++ {
		err := validateOptions(options)
		if err == nil || err.Error() != `path selectors "a/x" and "a/" overlap` {
			t.Fatalf("run %d: overlap error=%v, want the first earlier selector a/x", i, err)
		}
	}
}

func TestRequiredArtifactCannotClaimCanonicalOrToolOwnedPath(t *testing.T) {
	model := modelWithRequiredArtifact("AGENTS.md")
	options := Options{ToolPaths: []ToolPath{{Selector: "AGENTS.md", Owner: "onboarding"}}}
	if _, err := Classify(Request{Universe: testUniverse(PathState{Path: "AGENTS.md", Worktree: FileState{Present: true, Mode: "100644", Digest: "x"}}), Model: model, Options: options}); err == nil || !strings.Contains(err.Error(), "overlaps tool-owned") {
		t.Fatalf("required Artifact on tool-owned path error=%v", err)
	}
	model = modelWithRequiredArtifact(".markitect/model/project.yaml")
	options = Options{CanonicalModelPaths: []string{".markitect/model/project.yaml"}}
	if _, err := Classify(Request{Universe: testUniverse(PathState{Path: ".markitect/model/project.yaml", Worktree: FileState{Present: true, Mode: "100644", Digest: "x"}}), Model: model, Options: options}); err == nil || !strings.Contains(err.Error(), "overlaps canonical model") {
		t.Fatalf("required Artifact on canonical path error=%v", err)
	}
}

func TestPathSetRejectsCaseAliasAcrossFileAncestor(t *testing.T) {
	paths := map[string]*PathState{
		"Foo":     {Path: "Foo"},
		"foo/bar": {Path: "foo/bar"},
	}
	if err := validatePathSet(paths); err == nil || !strings.Contains(err.Error(), "case-colliding") {
		t.Fatalf("file/ancestor case collision error=%v", err)
	}
}

const emptyIgnoreYAML = "apiVersion: " + IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries: []\n"

func emptyModel() projectmodel.Report {
	return projectmodel.Report{APIVersion: projectmodel.APIVersion, ModelDigest: "model", Status: "succeeded",
		Managers: []projectmodel.Manager{}, Statements: []projectmodel.Statement{}, Artifacts: []projectmodel.Artifact{}, Checks: []projectmodel.Check{}, Files: []projectmodel.FileEntry{}, Findings: []projectmodel.Finding{}}
}

func modelWithRequiredArtifact(path string) projectmodel.Report {
	model := emptyModel()
	model.Statements = []projectmodel.Statement{{ID: "Statement:required", Owner: "Manager:root"}}
	model.Artifacts = []projectmodel.Artifact{{ID: "Artifact:required", Owner: "Manager:root", Required: true, Paths: []string{path}, Realizes: []string{"Statement:required"}}}
	return model
}

func testUniverse(paths ...PathState) *Universe {
	return &Universe{Revision: "revision", IdentityDigest: "repo", Paths: paths, IgnoreBytes: []byte(emptyIgnoreYAML), IgnoreBytesDigest: sha256Hex([]byte(emptyIgnoreYAML))}
}

func findState(universe *Universe, name string) *PathState {
	for i := range universe.Paths {
		if universe.Paths[i].Path == name {
			return &universe.Paths[i]
		}
	}
	return nil
}

func hasFinding(report Report, code string) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
