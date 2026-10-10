package projectcoverage

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestObserveWorkingFindsUntrackedOutsideSelectedRoot(t *testing.T) {
	root := initCoverageRepo(t)
	writeCoverageFile(t, root, "src/main.go", "package src\n")
	writeCoverageFile(t, root, ".gitignore", "docs/untracked.md\n")
	writeCoverageFile(t, root, IgnorePath, emptyIgnoreYAML)
	gitCoverage(t, root, "add", "src/main.go", IgnorePath, ".gitignore")
	gitCoverage(t, root, "commit", "-m", "base")
	writeCoverageFile(t, root, "docs/untracked.md", "untracked\n")
	universe, err := ObserveWorking(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if findState(universe, "docs/untracked.md") == nil {
		t.Fatal("untracked path outside the former inventory root was not observed")
	}
	report, err := Classify(Request{Universe: universe, Model: emptyModel(), LegacyRoots: []string{"src"}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accounted {
		t.Fatalf("untracked unknown path was accepted: %+v", report)
	}
}

func TestObserveWorkingUsesGitIndexModeWhenFileModeIsDisabled(t *testing.T) {
	root := initCoverageRepo(t)
	writeCoverageFile(t, root, "tools/run.sh", "#!/bin/sh\n")
	gitCoverage(t, root, "add", "tools/run.sh")
	gitCoverage(t, root, "update-index", "--chmod=+x", "tools/run.sh")
	gitCoverage(t, root, "config", "core.filemode", "false")
	gitCoverage(t, root, "commit", "-m", "track executable tool")
	if err := os.Chmod(filepath.Join(root, "tools", "run.sh"), 0o644); err != nil {
		t.Fatal(err)
	}
	if status := strings.TrimSpace(runGitCoverage(t, root, "status", "--porcelain")); status != "" {
		t.Fatalf("Git considered the core.filemode=false checkout dirty: %s", status)
	}

	universe, err := ObserveWorking(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	state := findState(universe, "tools/run.sh")
	if state == nil || state.Head.Mode != "100755" || state.Index.Mode != "100755" || state.Worktree.Mode != "100755" {
		t.Fatalf("tracked executable modes = %#v; want HEAD/index/worktree all 100755 under Git's disabled mode tracking", state)
	}
	if universe.Snapshot == nil || universe.Snapshot.Modes["tools/run.sh"] != "100755" {
		t.Fatalf("captured selected mode = %#v; want Git index mode 100755", universe.Snapshot)
	}
}

func TestObserveWorkingPreservesFilesystemModeWhenGitEnablesFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows filesystems do not expose POSIX executable mode bits")
	}
	root := initCoverageRepo(t)
	writeCoverageFile(t, root, "tools/run.sh", "#!/bin/sh\n")
	gitCoverage(t, root, "add", "tools/run.sh")
	gitCoverage(t, root, "update-index", "--chmod=+x", "tools/run.sh")
	gitCoverage(t, root, "config", "core.filemode", "true")
	gitCoverage(t, root, "commit", "-m", "track executable tool")
	if err := os.Chmod(filepath.Join(root, "tools", "run.sh"), 0o644); err != nil {
		t.Fatal(err)
	}

	universe, err := ObserveWorking(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	state := findState(universe, "tools/run.sh")
	if state == nil || state.Head.Mode != "100755" || state.Index.Mode != "100755" || state.Worktree.Mode != "100644" {
		t.Fatalf("tracked mode change = %#v; want HEAD/index 100755 and worktree 100644", state)
	}
}

func TestObserveWorkingRecheckIgnoresOnlyOperationalMembershipChanges(t *testing.T) {
	options := censusRaceOptions()
	for _, tc := range []struct {
		name   string
		before bool
		mutate func(*testing.T, string)
	}{
		{
			name: "host state added",
			mutate: func(t *testing.T, root string) {
				writeCoverageFile(t, root, ".markitect/runs/run-1/receipt.json", "Host receipt\n")
			},
		},
		{
			name:   "host state removed",
			before: true,
			mutate: func(t *testing.T, root string) {
				if err := os.Remove(filepath.Join(root, ".markitect", "runs", "run-1", "receipt.json")); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := censusRaceRepo(t)
			if tc.before {
				writeCoverageFile(t, root, ".markitect/runs/run-1/receipt.json", "Host receipt\n")
			}
			baseline, err := ObserveWorking(root, options)
			if err != nil {
				t.Fatal(err)
			}
			baselineReport, err := Classify(Request{Universe: baseline, Model: emptyModel(), Options: options})
			if err != nil {
				t.Fatal(err)
			}
			observed, err := observeWorking(root, options, func() error {
				tc.mutate(t, root)
				return nil
			})
			if err != nil {
				t.Fatalf("operational membership change failed the census recheck: %v", err)
			}
			observedReport, err := Classify(Request{Universe: observed, Model: emptyModel(), Options: options})
			if err != nil {
				t.Fatal(err)
			}
			if observed.Digest != baseline.Digest || observedReport.Digest != baselineReport.Digest {
				t.Fatalf("operational membership changed semantic digests: universe %s != %s, report %s != %s",
					observed.Digest, baseline.Digest, observedReport.Digest, baselineReport.Digest)
			}
		})
	}
}

func TestObserveWorkingRecheckStillRejectsNonOperationalChanges(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "source path added",
			mutate: func(t *testing.T, root string) {
				writeCoverageFile(t, root, "src/new.go", "package src\n")
			},
		},
		{
			name: "source path removed",
			mutate: func(t *testing.T, root string) {
				if err := os.Remove(filepath.Join(root, "src", "main.go")); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "source worktree mode changed",
			mutate: func(t *testing.T, root string) {
				if err := os.Chmod(filepath.Join(root, "src", "main.go"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "source index mode changed",
			mutate: func(t *testing.T, root string) {
				gitCoverage(t, root, "update-index", "--chmod=+x", "src/main.go")
			},
		},
		{
			name: "source index content changed",
			mutate: func(t *testing.T, root string) {
				writeCoverageFile(t, root, "src/main.go", "package src\n// staged during census\n")
				gitCoverage(t, root, "add", "src/main.go")
			},
		},
		{
			name: "non-operational tool-owned view added",
			mutate: func(t *testing.T, root string) {
				writeCoverageFile(t, root, ".markitect/views/generated.md", "# View\n")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "source worktree mode changed" && runtime.GOOS == "windows" {
				t.Skip("Windows filesystems do not expose POSIX executable mode bits")
			}
			root := censusRaceRepo(t)
			_, err := observeWorking(root, censusRaceOptions(), func() error {
				tc.mutate(t, root)
				return nil
			})
			if err == nil || !strings.Contains(err.Error(), "repository membership, mode, HEAD, or index changed during census") {
				t.Fatalf("non-operational change was not rejected by census recheck: %v", err)
			}
		})
	}
}

func censusRaceRepo(t *testing.T) string {
	t.Helper()
	root := initCoverageRepo(t)
	writeCoverageFile(t, root, "src/main.go", "package src\n")
	gitCoverage(t, root, "add", "src/main.go")
	gitCoverage(t, root, "commit", "-m", "census recheck fixture")
	return root
}

func censusRaceOptions() Options {
	return Options{ToolPaths: []ToolPath{
		{Selector: ".markitect/runs/", Owner: "projectrun", Operational: true},
		{Selector: ".markitect/views/", Owner: "projectwork"},
	}}
}

func TestObserveRevisionDoesNotMixLiveWorkingTree(t *testing.T) {
	root := initCoverageRepo(t)
	writeCoverageFile(t, root, "src/required.go", "package src\n")
	writeCoverageFile(t, root, IgnorePath, emptyIgnoreYAML)
	gitCoverage(t, root, "add", "src/required.go", IgnorePath)
	gitCoverage(t, root, "commit", "-m", "base")
	commit := strings.TrimSpace(runGitCoverage(t, root, "rev-parse", "HEAD"))
	writeCoverageFile(t, root, "outside/untracked.txt", "must not leak into fixed revision")
	universe, err := ObserveRevision(root, commit, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !universe.FixedRevision || findState(universe, "outside/untracked.txt") != nil {
		t.Fatal("fixed revision census included live untracked state")
	}
	report, err := Classify(Request{Universe: universe, Model: modelWithRequiredArtifact("src/required.go")})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Accounted || !report.Conforming {
		t.Fatalf("fixed revision report=%+v", report)
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

func TestObserveWorkingSupportsUnbornBranch(t *testing.T) {
	root := initCoverageRepo(t)
	writeCoverageFile(t, root, IgnorePath, emptyIgnoreYAML)
	writeCoverageFile(t, root, "src/main.go", "package src\n")
	gitCoverage(t, root, "add", IgnorePath, "src/main.go")
	universe, err := ObserveWorking(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if universe.Revision != "" || universe.FixedRevision || findState(universe, "src/main.go") == nil || universe.Snapshot == nil {
		t.Fatalf("unborn working census = %+v", universe)
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

func initCoverageRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitCoverage(t, root, "init", "-q")
	gitCoverage(t, root, "config", "user.email", "coverage@example.test")
	gitCoverage(t, root, "config", "user.name", "Coverage Test")
	return root
}

func writeCoverageFile(t *testing.T, root, name, contents string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitCoverage(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func runGitCoverage(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}
