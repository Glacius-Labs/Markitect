package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/runner"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const runnerTestImage = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

type runnerFixture struct {
	repo, engine, state, profile, config string
	logins                               []string
	cfg                                  runConfig
}

// newRunnerFixture writes a host run configuration with two logins and the
// example runtime profile, both naming one host engine.
func newRunnerFixture(t *testing.T) runnerFixture {
	t.Helper()
	dir := testkit.TempDir(t)
	f := runnerFixture{repo: filepath.Join(dir, "repo"), engine: filepath.Join(dir, "engine dir", "docker"), state: filepath.Join(dir, "state")}
	f.logins = []string{filepath.Join(dir, "a-login.json"), filepath.Join(dir, "b-login.json")}
	if err := os.MkdirAll(f.repo, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, login := range f.logins {
		if err := os.WriteFile(login, []byte(`{"tokens":{}}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config := map[string]any{
		"formatVersion": 4, "repositoryIdentity": "proj", "productBuild": "owner-asserted-build", "stateDirectory": f.state,
		"engineExecutable": f.engine, "deadline": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		"maxManagers": 4, "maxMessages": 16, "providerIdleTimeoutSeconds": 60, "cleanupTimeoutSeconds": 30, "persistenceTimeoutSeconds": 30,
		"credentials": map[string]any{
			"b-account": map[string]string{"provider": "codex-app-server", "mode": "dedicated-runtime-login", "path": f.logins[1]},
			"a-account": map[string]string{"provider": "codex-app-server", "mode": "dedicated-runtime-login", "path": f.logins[0]},
		},
	}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	f.config = filepath.Join(dir, "run.json")
	if err := os.WriteFile(f.config, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if f.cfg, err = loadRunConfig(context.Background(), f.config); err != nil {
		t.Fatal(err)
	}
	example, err := os.ReadFile("../../examples/runtime-profile.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f.profile = filepath.Join(dir, "runtime.yaml")
	if err := os.WriteFile(f.profile, []byte(strings.ReplaceAll(string(example), "ENGINE_EXECUTABLE", strconv.Quote(f.engine))), 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f runnerFixture) request(t *testing.T) cli.ApplyRequest {
	t.Helper()
	manifest, err := snapshot.NewPath(".markitect/model.yaml")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := snapshot.NewPath(filepath.Base(f.profile))
	if err != nil {
		t.Fatal(err)
	}
	selection := project.InputSelection{Repository: f.repo, Local: project.ModelSelection{Boundary: ".", ManifestPath: manifest}, RuntimeProfiles: []project.SourceLocation{{Boundary: filepath.Dir(f.profile), Path: profile}}}
	return cli.ApplyRequest{Selection: selection, RunConfig: f.config, Build: "host-build", Runner: runnerTestImage, ResultBranch: "dev/realized"}
}

func TestRunnerRunConfigNamesTheRunnerLayoutOnly(t *testing.T) {
	f := newRunnerFixture(t)
	data, logins, err := runnerRunConfig(f.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(logins, f.logins) {
		t.Fatalf("logins %q, want sorted by reference %q", logins, f.logins)
	}
	var translated runConfig
	if err := json.Unmarshal(data, &translated); err != nil {
		t.Fatal(err)
	}
	want := f.cfg
	want.StateDirectory, want.EngineExecutable = "/markitect/state", "/usr/local/bin/docker"
	want.Credentials = map[string]credentialSource{
		"a-account": {Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: "/markitect/credentials/1.json"},
		"b-account": {Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: "/markitect/credentials/2.json"},
	}
	if !reflect.DeepEqual(translated, want) {
		t.Fatalf("translated %+v\nwant %+v", translated, want)
	}
	if f.cfg.StateDirectory != f.state {
		t.Fatal("translation changed the host configuration")
	}
}

func TestRunnerApplyLaunchesWithTheOwnersFilesAndReportsTheBranch(t *testing.T) {
	f := newRunnerFixture(t)
	launchDir := filepath.Join(testkit.TempDir(t), "launch")
	if err := os.Mkdir(launchDir, 0o700); err != nil {
		t.Fatal(err)
	}
	var seen runner.LaunchRequest
	commit, base := strings.Repeat("2", 40), strings.Repeat("1", 40)
	launch := func(_ context.Context, r runner.LaunchRequest) (runner.Launch, error) {
		seen = r
		report := json.RawMessage(`{"status":"passed","id":"0123456789abcdef0123456789abcdef","providerRetries":3}`)
		return runner.Launch{ID: "launch1", Directory: launchDir, Base: base, Branch: "dev/realized", Commit: commit, Report: report,
			Result: runner.Result{FormatVersion: 1, ApplyStatus: runner.StatusPassed, ApplyID: "0123456789abcdef0123456789abcdef", Base: base, Commit: commit, Ref: runner.ResultRef, Paths: []string{"README.md"}}}, nil
	}
	out, err := runnerApply(context.Background(), f.request(t), f.cfg, launch)
	if err != nil {
		t.Fatal(err)
	}
	if seen.Engine != f.engine || seen.Image != runnerTestImage || seen.Repository != f.repo || seen.Model != ".markitect/model.yaml" || seen.ResultBranch != "dev/realized" {
		t.Fatalf("launch request %+v", seen)
	}
	if seen.OutputRoot != filepath.Join(f.state, "proj", "runner") || seen.StateVolume != "markitect-state-proj" || !reflect.DeepEqual(seen.Credentials, f.logins) || seen.StopTimeout <= 0 {
		t.Fatalf("launch request %+v", seen)
	}
	if strings.Contains(string(seen.RuntimeProfile), f.engine) || !strings.Contains(string(seen.RuntimeProfile), runner.EngineBinary) || !strings.Contains(string(seen.RunConfig), `"/markitect/state"`) {
		t.Fatalf("invocation documents not translated:\n%s\n%s", seen.RuntimeProfile, seen.RunConfig)
	}
	if out.Status != model.Passed || out.ID != "0123456789abcdef0123456789abcdef" || out.ProviderRetries != 3 || out.Runner == nil || out.Runner.Branch != "dev/realized" || out.Runner.Commit != commit || out.Runner.Launch != "launch1" {
		t.Fatalf("report %+v %+v", out, out.Runner)
	}
	if out.EvidencePath != filepath.Join(launchDir, "result", runner.ReportFile) {
		t.Fatalf("evidence path %s", out.EvidencePath)
	}
	data, err := os.ReadFile(filepath.Join(launchDir, "launch.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil || record["status"] != "passed" || record["runner"].(map[string]any)["report"].(map[string]any)["providerRetries"] != float64(3) {
		t.Fatalf("launch record %s %v", data, err)
	}
}

func TestRunnerApplyMapsRunnerOutcomes(t *testing.T) {
	base := strings.Repeat("1", 40)
	for _, tc := range []struct {
		name      string
		result    runner.Result
		launchErr error
		status    model.Status
		message   string
	}{
		{"no change", runner.Result{ApplyStatus: runner.StatusPassed, NoChange: true}, nil, model.Passed, ""},
		{"failed", runner.Result{ApplyStatus: runner.StatusFailed}, nil, model.Failed, "Apply in the runner failed: Assessment rejected"},
		{"incomplete", runner.Result{ApplyStatus: runner.StatusNotCompleted}, nil, model.NotCompleted, "did not complete: Assessment rejected"},
		{"hand-back", runner.Result{ApplyStatus: runner.StatusPassed, Error: "boom"}, errors.New("runner: boom"), model.NotCompleted, "runner: boom"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRunnerFixture(t)
			launch := func(context.Context, runner.LaunchRequest) (runner.Launch, error) {
				tc.result.FormatVersion, tc.result.Base = 1, base
				return runner.Launch{ID: "l", Base: base, Result: tc.result, Report: json.RawMessage(`{"status":"failed","error":"Assessment rejected"}`)}, tc.launchErr
			}
			out, err := runnerApply(context.Background(), f.request(t), f.cfg, launch)
			if out.Status != tc.status || (tc.message == "") != (err == nil) || err != nil && !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("status %s err %v", out.Status, err)
			}
		})
	}
}

func TestRunnerApplyRefusesInputsTheRunnerCannotCarry(t *testing.T) {
	f := newRunnerFixture(t)
	never := func(context.Context, runner.LaunchRequest) (runner.Launch, error) {
		t.Fatal("launched")
		return runner.Launch{}, nil
	}
	external := f.request(t)
	external.Selection.Local.Boundary = filepath.Join(f.repo, ".markitect")
	if _, err := runnerApply(context.Background(), external, f.cfg, never); err == nil || !strings.Contains(err.Error(), "repository-relative") {
		t.Fatalf("external model: %v", err)
	}
	other := f.cfg
	other.EngineExecutable = filepath.Join(filepath.Dir(f.engine), "other-docker")
	if _, err := runnerApply(context.Background(), f.request(t), other, never); err == nil || !strings.Contains(err.Error(), "engineExecutable") {
		t.Fatalf("engine mismatch: %v", err)
	}
	if err := os.Remove(f.logins[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := runnerApply(context.Background(), f.request(t), f.cfg, never); err == nil || !strings.Contains(err.Error(), "dedicated login 2") || strings.Contains(err.Error(), "b-login") {
		t.Fatalf("missing login refused without its path: %v", err)
	}
}

func TestRunnerModeSkipsTheHostBuildAndReachesTheCheckout(t *testing.T) {
	f := newRunnerFixture(t)
	request := f.request(t)
	request.Build = "a-different-host-build"
	out, err := applyExecution(nil, nil, nil, nil)(context.Background(), request)
	if err == nil || strings.Contains(err.Error(), "invocation build") || !strings.Contains(err.Error(), ".git is a directory") || out.Status != model.NotCompleted {
		t.Fatalf("runner mode: %v %+v", err, out)
	}
	request.Runner, request.ResultBranch = "", ""
	if _, err := applyExecution(nil, nil, nil, nil)(context.Background(), request); err == nil || !strings.Contains(err.Error(), "invocation build") {
		t.Fatalf("native mode keeps the build assertion: %v", err)
	}
}
