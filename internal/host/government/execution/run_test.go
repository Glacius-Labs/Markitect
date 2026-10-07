package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

var fixtureBuild struct {
	once            sync.Once
	dir, executable string
	err             error
}

func TestMain(m *testing.M) {
	code := m.Run()
	if fixtureBuild.dir != "" {
		_ = os.RemoveAll(fixtureBuild.dir)
	}
	os.Exit(code)
}

func fixtureRunner(t *testing.T) string {
	t.Helper()
	fixtureBuild.once.Do(func() {
		fixtureBuild.dir, fixtureBuild.err = os.MkdirTemp("", "government-test-runner-")
		if fixtureBuild.err != nil {
			return
		}
		fixtureBuild.executable = filepath.Join(fixtureBuild.dir, "runner.exe")
		cmd := exec.Command("go", "build", "-o", fixtureBuild.executable, "./runner")
		cmd.Dir = "../../../../examples/government-g2"
		if out, err := cmd.CombinedOutput(); err != nil {
			fixtureBuild.err = fmt.Errorf("build actual external fixture: %w: %s", err, out)
		}
	})
	if fixtureBuild.err != nil {
		t.Fatal(fixtureBuild.err)
	}
	return fixtureBuild.executable
}

func fixtureOptions(t *testing.T) Options {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo, state, temp := filepath.Join(root, "repo"), filepath.Join(root, "state"), filepath.Join(root, "temporary")
	for _, dir := range []string{repo, state, temp} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	fixture, err := filepath.Abs("../../../../examples/government-g2")
	if err != nil {
		t.Fatal(err)
	}
	if err := filepath.WalkDir(fixture, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(fixture, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(repo, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string { return fixtureGit(t, repo, args...) }
	git("init", "-b", "feature")
	git("config", "user.name", "G2 mechanics test")
	git("config", "user.email", "g2@example.invalid")
	git("config", "core.autocrlf", "false")
	git("add", ".")
	git("-c", "commit.gpgsign=false", "commit", "-m", "prior active fixture")
	base := git("rev-parse", "HEAD")
	ref := "refs/markitect/government/active/test"
	git("update-ref", ref, base)
	runner := fixtureRunner(t)
	spec := func(slot, mode string) RunnerSpec {
		return RunnerSpec{SlotID: slot, Command: runner, Args: []string{mode}, Model: "deterministic-mechanics-fixture", ProviderVersion: "fixture-v1", TimeoutSeconds: 30, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20}
	}
	rt := Runtime{APIVersion: RuntimeVersion, TimeoutSeconds: 1800, ActiveRef: ref, ExpectedBase: base, StateDirectory: state, TemporaryDirectory: temp, Executor: spec("writer", "executor"), Verifier: spec("independent-review", "verifier"), Checks: []authoring.Check{{Name: "inventory-boundary", Run: []string{"go", "test", "./inventory", "-count=1"}, TimeoutSeconds: intPointer(60)}}}
	for i, name := range []string{"correctness", "maintainability"} {
		mode := "assent"
		if i == 1 {
			mode = "assent-unaffected"
		}
		rt.Ressorts = append(rt.Ressorts, RessortRunner{Ressort: core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Ressort", Namespace: "inventory", Name: name}, Runner: spec("ressort-"+name, mode)})
	}
	return Options{Repo: repo, ConfigPath: "government.yaml", OrderPath: "order.yaml", Runtime: rt}
}

func fixtureGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	out, err := gitOutput(context.Background(), repo, nil, nil, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestRunActualProcessesPromoteOneFinalTree(t *testing.T) {
	opts := fixtureOptions(t)
	before, err := os.ReadFile(filepath.Join(opts.Repo, "inventory/reservation.go"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("%s at %s: %v", report.Status, report.Stage, err)
	}
	if report.Status != "accepted-scoped" || report.Promotion == nil || report.Promotion.Status != "promoted" || report.Candidate == nil || report.Evidence == nil || report.Decision == nil {
		t.Fatalf("missing acceptance records: %+v", report)
	}
	if len(report.Actors) != 4 || len(report.Votes) != 2 || len(report.Checks) != 1 || report.Checks[0].ExitCode != 0 {
		t.Fatalf("missing actual executor/review/two-vote/check results: %+v", report)
	}
	runIDs := map[string]bool{}
	for _, actor := range report.Actors {
		r := actor.Result.Receipt
		if r.RunID == "" || runIDs[r.RunID] || r.ConfigDigest == "" || r.ExecutableDigest == "" || r.ContextDigest == "" || r.InputDigest == "" || r.StdoutDigest == "" {
			t.Fatalf("actor has no distinct actual receipt: %+v", actor)
		}
		runIDs[r.RunID] = true
	}
	if report.Votes[1].Outcome != government.VoteAssentUnaffected {
		t.Fatalf("unaffected assent was omitted: %+v", report.Votes)
	}
	for _, vote := range report.Votes {
		if vote.MaterialCandidateID != report.Candidate.ID || vote.EvidenceID != report.Evidence.ID || !runIDs[vote.Provenance.RunID] {
			t.Fatalf("unbound vote: %+v", vote)
		}
	}
	if report.Evidence.MaterialCandidateID != report.Candidate.ID || report.Decision.EvidenceID != report.Evidence.ID || len(report.Decision.VoteIDs) != 2 {
		t.Fatal("candidate/evidence/vote/decision chain is not bound")
	}
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != report.CandidateCommit {
		t.Fatalf("active ref did not promote: %s", active)
	}
	if head := fixtureGit(t, opts.Repo, "rev-parse", "HEAD"); head != opts.Runtime.ExpectedBase {
		t.Fatalf("user HEAD moved: %s", head)
	}
	after, err := os.ReadFile(filepath.Join(opts.Repo, "inventory/reservation.go"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("user working files changed")
	}
	final := fixtureGit(t, opts.Repo, "show", report.CandidateCommit+":inventory/reservation.go")
	if !strings.Contains(final, "available > math.MaxInt64-amount") {
		t.Fatal("actual promoted tree has no overflow guard")
	}
	if _, err := os.Stat(report.Promotion.IntentPath); err != nil {
		t.Fatal("durable intent missing", err)
	}
	assertRetainedReport(t, report)
}

func TestRunBlocksMissingRejectedStaleReviewAndVoteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		review     bool
		wantStage  string
	}{
		{"missing-assent", "incomplete", false, "ressort-votes"},
		{"objection", "objection", false, "ressort-votes"},
		{"foreign-vote", "stale-vote", false, "ressort-votes"},
		{"foreign-receipt", "wrong-binding", false, "ressort-votes"},
		{"candidate-mutated", "mutate-candidate", false, "ressort-votes"},
		{"missing-review", "missing-review", true, "independent-review"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := fixtureOptions(t)
			if tc.review {
				opts.Runtime.Verifier.Args = []string{tc.mode}
			} else {
				opts.Runtime.Ressorts[1].Runner.Args = []string{tc.mode}
			}
			report, err := Run(context.Background(), opts)
			if err == nil || report.Status == "accepted-scoped" || report.Stage != tc.wantStage || report.Promotion != nil {
				t.Fatalf("failed open: %s/%s: %v", report.Status, report.Stage, err)
			}
			if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != opts.Runtime.ExpectedBase {
				t.Fatal("rejected material was promoted")
			}
			assertRetainedReport(t, report)
		})
	}
}

func TestRunRejectsFailedTechnicalCheckAndIncompleteCabinet(t *testing.T) {
	for _, name := range []string{"failed-check", "missing-cabinet-slot", "stale-initial-base", "insufficient-run-budget"} {
		t.Run(name, func(t *testing.T) {
			opts := fixtureOptions(t)
			switch name {
			case "failed-check":
				opts.Runtime.Checks[0].Run = []string{"go", "test", "./nonexistent"}
			case "missing-cabinet-slot":
				opts.Runtime.Ressorts = opts.Runtime.Ressorts[:1]
			case "stale-initial-base":
				opts.Runtime.ExpectedBase = strings.Repeat("0", 40)
			case "insufficient-run-budget":
				opts.Runtime.TimeoutSeconds = 10
			}
			activeBefore := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef)
			report, err := Run(context.Background(), opts)
			if err == nil || report.Promotion != nil {
				t.Fatalf("failed open: %+v %v", report, err)
			}
			if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != activeBefore {
				t.Fatal("invalid run changed Active")
			}
			assertRetainedReport(t, report)
		})
	}
}

func TestReadRuntimeRequiresExternalTrustedFile(t *testing.T) {
	opts := fixtureOptions(t)
	data, err := json.Marshal(opts.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(opts.Repo, "runtime.json"), filepath.Join(opts.Repo, ".git", "runtime.json")} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadRuntime(opts.Repo, path); err == nil {
			t.Fatalf("admitted runtime under source/Git metadata: %s", path)
		}
	}
	path := filepath.Join(opts.Runtime.StateDirectory, "runtime.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	read, err := ReadRuntime(opts.Repo, path)
	if err != nil || read.ExpectedBase != opts.Runtime.ExpectedBase {
		t.Fatalf("explicit external runtime not read: %v", err)
	}
}

func TestReviewRequiresExplainedCoverage(t *testing.T) {
	response := agentexec.Response{Outcome: agentexec.OutcomePassed, VerifierObservations: []agentexec.Observation{{Subject: "required-scope", Outcome: agentexec.OutcomePassed}}}
	if err := requireReview(response, []string{"required-scope"}); err == nil {
		t.Fatal("unexplained pass was accepted as review")
	}
	response.VerifierObservations[0].Detail = "checked the candidate's declared invariant and focused test"
	if err := requireReview(response, []string{"required-scope", "missing-scope"}); err == nil {
		t.Fatal("partial coverage was accepted")
	}
	if err := requireReview(response, []string{"required-scope"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsActiveBaseChangingDuringFinalVote(t *testing.T) {
	opts := fixtureOptions(t)
	tree := fixtureGit(t, opts.Repo, "rev-parse", "HEAD^{tree}")
	alternative := fixtureGit(t, opts.Repo, "-c", "commit.gpgsign=false", "commit-tree", tree, "-p", opts.Runtime.ExpectedBase, "-m", "concurrent active candidate")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	last := &opts.Runtime.Ressorts[1].Runner
	actualRunner := last.Command
	last.Command = exe
	last.Args = []string{"-test.run=^TestG2ActiveChangeHelper$", "--", actualRunner, opts.Repo, opts.Runtime.ActiveRef, alternative}
	runnerBytes, err := os.ReadFile(actualRunner)
	if err != nil {
		t.Fatal(err)
	}
	runnerInfo, err := os.Stat(actualRunner)
	if err != nil {
		t.Fatal(err)
	}
	mode := fmt.Sprintf("%04o", runnerInfo.Mode().Perm())
	if runtime.GOOS == "windows" {
		mode = "0644"
	}
	last.RuntimeFiles = []agentexec.RuntimeFile{{Path: actualRunner, Mode: mode, Digest: government.BytesDigest(runnerBytes)}}
	report, err := Run(context.Background(), opts)
	if err == nil || report.Stage != "promote" || report.Promotion == nil || report.Promotion.Status != "stale-base" {
		t.Fatalf("stale Active not rejected at promotion: %s/%s %+v: %v", report.Status, report.Stage, report.Promotion, err)
	}
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != alternative {
		t.Fatal("CAS overwrote concurrent Active")
	}
	assertRetainedReport(t, report)
}

// This external helper delegates the real vote to the fixture, then simulates
// a cooperating competing writer advancing Active before the Host's CAS.
func TestG2ActiveChangeHelper(t *testing.T) {
	for i, arg := range os.Args {
		if arg != "--" || len(os.Args) != i+5 {
			continue
		}
		args := os.Args[i+1:]
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			os.Exit(2)
		}
		cmd := exec.Command(args[0], "assent-unaffected")
		cmd.Stdin = bytes.NewReader(input)
		output, err := cmd.Output()
		if err != nil {
			os.Exit(2)
		}
		git := exec.Command("git", "-C", args[1], "update-ref", args[2], args[3])
		if err := git.Run(); err != nil {
			os.Exit(2)
		}
		_, _ = os.Stdout.Write(output)
		os.Exit(0)
	}
}

func assertRetainedReport(t *testing.T, report Report) {
	t.Helper()
	data, err := os.ReadFile(report.ReportPath)
	if err != nil {
		t.Fatalf("abort/success report was not retained: %v", err)
	}
	var stored Report
	if err := json.Unmarshal(data, &stored); err != nil || stored.RunID != report.RunID || stored.Status != report.Status || stored.Stage != report.Stage {
		t.Fatalf("retained report differs: %v", err)
	}
}
