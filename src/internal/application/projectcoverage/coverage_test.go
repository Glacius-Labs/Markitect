package projectcoverage

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	modelcoverage "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectcoverage"
)

func TestObserveWorkingFindsUntrackedOutsideSelectedRoot(t *testing.T) {
	root := initCoverageRepo(t)
	writeCoverageFile(t, root, "src/main.go", "package src\n")
	writeCoverageFile(t, root, ".gitignore", "docs/untracked.md\n")
	writeCoverageFile(t, root, modelcoverage.IgnorePath, emptyIgnoreYAML)
	gitCoverage(t, root, "add", "src/main.go", modelcoverage.IgnorePath, ".gitignore")
	gitCoverage(t, root, "commit", "-m", "base")
	writeCoverageFile(t, root, "docs/untracked.md", "untracked\n")
	universe, err := ObserveWorking(root, modelcoverage.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if findState(universe, "docs/untracked.md") == nil {
		t.Fatal("untracked path outside the former inventory root was not observed")
	}
	report, err := modelcoverage.Classify(modelcoverage.Request{Universe: universe, Model: emptyModel(), LegacyRoots: []string{"src"}})
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

	universe, err := ObserveWorking(root, modelcoverage.Options{})
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

	universe, err := ObserveWorking(root, modelcoverage.Options{})
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
			baselineReport, err := modelcoverage.Classify(modelcoverage.Request{Universe: baseline, Model: emptyModel(), Options: options})
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
			observedReport, err := modelcoverage.Classify(modelcoverage.Request{Universe: observed, Model: emptyModel(), Options: options})
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

func censusRaceOptions() modelcoverage.Options {
	return modelcoverage.Options{ToolPaths: []modelcoverage.ToolPath{
		{Selector: ".markitect/runs/", Owner: "projectrun", Operational: true},
		{Selector: ".markitect/views/", Owner: "projectwork"},
	}}
}

func TestObserveRevisionDoesNotMixLiveWorkingTree(t *testing.T) {
	root := initCoverageRepo(t)
	writeCoverageFile(t, root, "src/required.go", "package src\n")
	writeCoverageFile(t, root, modelcoverage.IgnorePath, emptyIgnoreYAML)
	gitCoverage(t, root, "add", "src/required.go", modelcoverage.IgnorePath)
	gitCoverage(t, root, "commit", "-m", "base")
	commit := strings.TrimSpace(runGitCoverage(t, root, "rev-parse", "HEAD"))
	writeCoverageFile(t, root, "outside/untracked.txt", "must not leak into fixed revision")
	universe, err := ObserveRevision(root, commit, modelcoverage.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !universe.FixedRevision || findState(universe, "outside/untracked.txt") != nil {
		t.Fatal("fixed revision census included live untracked state")
	}
	report, err := modelcoverage.Classify(modelcoverage.Request{Universe: universe, Model: modelWithRequiredArtifact("src/required.go")})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Accounted || !report.Conforming {
		t.Fatalf("fixed revision report=%+v", report)
	}
}

func TestObserveWorkingSupportsUnbornBranch(t *testing.T) {
	root := initCoverageRepo(t)
	writeCoverageFile(t, root, modelcoverage.IgnorePath, emptyIgnoreYAML)
	writeCoverageFile(t, root, "src/main.go", "package src\n")
	gitCoverage(t, root, "add", modelcoverage.IgnorePath, "src/main.go")
	universe, err := ObserveWorking(root, modelcoverage.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if universe.Revision != "" || universe.FixedRevision || findState(universe, "src/main.go") == nil || universe.Snapshot == nil {
		t.Fatalf("unborn working census = %+v", universe)
	}
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

const emptyIgnoreYAML = "apiVersion: " + modelcoverage.IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries: []\n"

func emptyModel() projectmodel.Report {
	return projectmodel.Report{Status: "succeeded", Artifacts: []projectmodel.Artifact{}}
}

func modelWithRequiredArtifact(path string) projectmodel.Report {
	model := emptyModel()
	model.Statements = []projectmodel.Statement{{ID: "Statement:required", Owner: "Manager:root"}}
	model.Artifacts = []projectmodel.Artifact{{ID: "Artifact:required", Owner: "Manager:root", Required: true, Paths: []string{path}, Realizes: []string{"Statement:required"}}}
	return model
}

func findState(universe *modelcoverage.Universe, name string) *modelcoverage.PathState {
	for i := range universe.Paths {
		if universe.Paths[i].Path == name {
			return &universe.Paths[i]
		}
	}
	return nil
}
