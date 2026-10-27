package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The test binary doubles as a fake engine CLI and a fake markitect binary, so
// the launcher, the mount set, the entrypoint and the import run end to end
// with real Git but without a container engine or a provider.
const (
	helperRole  = "MARKITECT_RUNNER_TEST_ROLE"
	helperMode  = "MARKITECT_RUNNER_TEST_MODE"
	fakeApplyID = "0123456789abcdef0123456789abcdef"
)

func TestMain(m *testing.M) {
	switch os.Getenv(helperRole) {
	case "engine":
		os.Exit(fakeEngine(os.Args[1:]))
	case "apply":
		os.Exit(fakeApply(os.Args[1:]))
	}
	os.Exit(m.Run())
}

// fakeEngine plays "docker run": it maps every mount target to its host source
// and runs the entrypoint with that layout, the clone beside the hand-back area.
func fakeEngine(args []string) int {
	if len(args) > 0 && args[0] == "stop" {
		return 0
	}
	sources := map[string]string{}
	var entryArgs []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--mount" && i+1 < len(args) {
			var source, target string
			for _, field := range strings.Split(args[i+1], ",") {
				key, value, _ := strings.Cut(field, "=")
				switch key {
				case "source":
					source = value
				case "target":
					target = value
				}
			}
			sources[target] = source
			i++
			continue
		}
		if strings.HasPrefix(args[i], "sha256:") {
			entryArgs = args[i+1:]
			break
		}
	}
	for _, target := range []string{CheckoutDir, InvocationDir, OutputDir, StateDir, EngineSocket, CredentialPath(1)} {
		if sources[target] == "" {
			fmt.Fprintln(os.Stderr, "fake engine: missing mount", target)
			return 125
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return 125
	}
	os.Setenv(helperRole, "apply")
	output := filepath.FromSlash(sources[OutputDir])
	layout := Layout{Checkout: filepath.FromSlash(sources[CheckoutDir]), Invocation: filepath.FromSlash(sources[InvocationDir]), Output: output, Clone: filepath.Join(filepath.Dir(output), "fake-clone"), Markitect: executable}
	return entry(context.Background(), entryArgs, os.Stderr, layout)
}

// fakeApply plays "markitect apply": it publishes into --repo as the mode says
// and prints a report shaped like the real one.
func fakeApply(args []string) int {
	values := map[string]string{}
	for i := 1; i+1 < len(args); i += 2 {
		values[args[i]] = args[i+1]
	}
	repo := values["--repo"]
	for _, file := range []string{values["--run-config"], values["--runtime-profile"]} {
		if _, err := os.Stat(file); err != nil {
			fmt.Fprintln(os.Stderr, "fake apply: missing invocation file", err)
			return 3
		}
	}
	write := func(name, content string) {
		path := filepath.Join(repo, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(content), 0o644)
	}
	report := func(status string, noChange bool, paths []string, code int) int {
		publication := map[string]any{"noChange": noChange, "completedPaths": paths, "readback": true}
		value := map[string]any{"formatVersion": 1, "operation": "apply", "status": status, "id": fakeApplyID, "providerRetries": 2, "outcome": status}
		if status == StatusPassed {
			value["apply"] = map[string]any{"status": status, "publication": publication}
		} else {
			value["error"] = "Assessment rejected the model"
		}
		data, _ := json.MarshalIndent(value, "", "  ")
		os.Stdout.Write(append(data, '\n'))
		return code
	}
	switch mode := os.Getenv(helperMode); mode {
	case "change", "stray":
		write("README.md", "realized\n")
		write("docs/new.txt", "new\n")
		os.Remove(filepath.Join(repo, "old.txt"))
		write(".markitect/state/write.lock", "lock\n")
		if mode == "stray" {
			write("extra.txt", "not published\n")
		}
		return report(StatusPassed, false, []string{"README.md", "docs/new.txt", "old.txt"}, 0)
	case "nochange":
		return report(StatusPassed, true, []string{}, 0)
	case "failed":
		return report(StatusFailed, false, nil, 1)
	}
	return 3
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := (gitCall{dir: dir}).text(context.Background(), args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func ownerCheckout(t *testing.T) (string, string) {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "owner repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, "", "init", "--quiet", "--initial-branch=dev/work", repo)
	for name, content := range map[string]string{"README.md": "start\n", "old.txt": "old\n", "src/main.go": "package main\n"} {
		path := filepath.Join(repo, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, repo, "add", "--all")
	git(t, repo, "-c", "user.name=Owner", "-c", "user.email=owner@example.invalid", "commit", "--quiet", "-m", "start")
	return repo, git(t, repo, "rev-parse", "HEAD")
}

func launchRequest(t *testing.T, repo, mode string) LaunchRequest {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(helperRole, "engine")
	t.Setenv(helperMode, mode)
	login := filepath.Join(t.TempDir(), "login.json")
	if err := os.WriteFile(login, []byte(`{"tokens":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return LaunchRequest{
		Engine: executable, Image: "sha256:" + strings.Repeat("c", 64), Repository: repo, Model: ".markitect/model.yaml",
		RunConfig: []byte("{}\n"), RuntimeProfile: []byte("apiVersion: markitect.runtime/v1\n"), Credentials: []string{login},
		OutputRoot: filepath.Join(t.TempDir(), "state", "proj", "runner"), StateVolume: "markitect-state-proj",
		StopTimeout: time.Minute, Log: io.Discard,
	}
}

func TestStartImportsTheVerifiedChangeAsANewBranch(t *testing.T) {
	repo, base := ownerCheckout(t)
	launched, err := Start(context.Background(), launchRequest(t, repo, "change"))
	if err != nil {
		t.Fatalf("%v\n%s", err, logOf(launched))
	}
	wantBranch := "markitect/apply-" + fakeApplyID
	if launched.Base != base || launched.Branch != wantBranch || launched.Result.ApplyStatus != StatusPassed || launched.Result.NoChange {
		t.Fatalf("launch %+v", launched)
	}
	if got := git(t, repo, "rev-parse", "refs/heads/"+wantBranch); got != launched.Commit {
		t.Fatalf("branch at %s, want %s", got, launched.Commit)
	}
	if parent := git(t, repo, "rev-parse", launched.Commit+"^"); parent != base {
		t.Fatalf("parent %s, want base %s", parent, base)
	}
	if changed := git(t, repo, "diff", "--name-only", base, launched.Commit); changed != "README.md\ndocs/new.txt\nold.txt" {
		t.Fatalf("changed paths %q", changed)
	}
	if readme := git(t, repo, "show", launched.Commit+":README.md"); readme != "realized" {
		t.Fatalf("README %q", readme)
	}
	// The checkout itself is untouched: same branch, HEAD, index and files.
	if head, branch := git(t, repo, "rev-parse", "HEAD"), git(t, repo, "branch", "--show-current"); head != base || branch != "dev/work" {
		t.Fatalf("checkout moved to %s on %s", head, branch)
	}
	if status := git(t, repo, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("checkout changed: %q", status)
	}
	var report struct {
		ProviderRetries int `json:"providerRetries"`
	}
	if err := json.Unmarshal(launched.Report, &report); err != nil || report.ProviderRetries != 2 {
		t.Fatalf("report %s %v", launched.Report, err)
	}
	for _, name := range []string{"invocation/run.json", "invocation/runtime-profile.yaml", "result/result.json", "result/result.bundle", "result/apply-report.json", "runner.log"} {
		if _, err := os.Stat(filepath.Join(launched.Directory, filepath.FromSlash(name))); err != nil {
			t.Errorf("launch directory lacks %s", name)
		}
	}
}

func TestStartReportsOutcomesWithoutABranch(t *testing.T) {
	for _, tc := range []struct {
		mode, status string
		noChange     bool
	}{
		{"nochange", StatusPassed, true},
		{"failed", StatusFailed, false},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			repo, _ := ownerCheckout(t)
			launched, err := Start(context.Background(), launchRequest(t, repo, tc.mode))
			if err != nil {
				t.Fatalf("%v\n%s", err, logOf(launched))
			}
			if launched.Result.ApplyStatus != tc.status || launched.Result.NoChange != tc.noChange || launched.Branch != "" || launched.Commit != "" {
				t.Fatalf("launch %+v", launched)
			}
			if branches := git(t, repo, "branch", "--list", "markitect/*"); branches != "" {
				t.Fatalf("created %q", branches)
			}
			if _, err := os.Stat(filepath.Join(launched.Directory, "result", BundleFile)); err == nil {
				t.Fatal("bundle without a verified change")
			}
		})
	}
}

func TestStartRefusesAnUnpublishedChange(t *testing.T) {
	repo, _ := ownerCheckout(t)
	launched, err := Start(context.Background(), launchRequest(t, repo, "stray"))
	if err == nil || !strings.Contains(err.Error(), "unpublished change") || !strings.Contains(err.Error(), "extra.txt") {
		t.Fatalf("stray change: %v", err)
	}
	if launched.Result.Error == "" || launched.Branch != "" {
		t.Fatalf("launch %+v", launched)
	}
	if branches := git(t, repo, "branch", "--list", "markitect/*"); branches != "" {
		t.Fatalf("created %q", branches)
	}
}

func TestStartRefusesCheckoutsTheRunnerCannotClone(t *testing.T) {
	t.Run("tracked change", func(t *testing.T) {
		repo, _ := ownerCheckout(t)
		os.WriteFile(filepath.Join(repo, "README.md"), []byte("edited\n"), 0o644)
		if _, err := Start(context.Background(), launchRequest(t, repo, "change")); err == nil || !strings.Contains(err.Error(), "committed history only") {
			t.Fatalf("dirty checkout: %v", err)
		}
	})
	t.Run("detached HEAD", func(t *testing.T) {
		repo, base := ownerCheckout(t)
		git(t, repo, "checkout", "--quiet", "--detach", base)
		if _, err := Start(context.Background(), launchRequest(t, repo, "change")); err == nil || !strings.Contains(err.Error(), "named branch") {
			t.Fatalf("detached HEAD: %v", err)
		}
	})
	t.Run("linked worktree", func(t *testing.T) {
		repo, _ := ownerCheckout(t)
		linked := filepath.Join(t.TempDir(), "linked")
		git(t, repo, "worktree", "add", "--quiet", "-b", "dev/linked", linked)
		if _, err := Start(context.Background(), launchRequest(t, linked, "change")); err == nil || !strings.Contains(err.Error(), "linked worktrees") {
			t.Fatalf("linked worktree: %v", err)
		}
	})
	t.Run("borrowed objects", func(t *testing.T) {
		repo, _ := ownerCheckout(t)
		shared := filepath.Join(t.TempDir(), "shared")
		git(t, "", "clone", "--quiet", "--shared", repo, shared)
		if _, err := Start(context.Background(), launchRequest(t, shared, "change")); err == nil || !strings.Contains(err.Error(), "alternates") {
			t.Fatalf("shared clone: %v", err)
		}
	})
	t.Run("existing result branch", func(t *testing.T) {
		repo, _ := ownerCheckout(t)
		git(t, repo, "branch", "taken")
		request := launchRequest(t, repo, "change")
		request.ResultBranch = "taken"
		if _, err := Start(context.Background(), request); err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("existing branch: %v", err)
		}
	})
	t.Run("invalid result branch", func(t *testing.T) {
		repo, _ := ownerCheckout(t)
		request := launchRequest(t, repo, "change")
		request.ResultBranch = "bad..name"
		if _, err := Start(context.Background(), request); err == nil || !strings.Contains(err.Error(), "not a valid branch name") {
			t.Fatalf("invalid branch: %v", err)
		}
	})
}

func TestImportRefusesABundleThatDoesNotMatchTheResult(t *testing.T) {
	repo, _ := ownerCheckout(t)
	launched, err := Start(context.Background(), launchRequest(t, repo, "change"))
	if err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(launched.Directory, "result", BundleFile)
	result := launched.Result
	for name, mutate := range map[string]func(*Result){
		"other paths":  func(r *Result) { r.Paths = []string{"README.md"} },
		"other base":   func(r *Result) { r.Base = launched.Commit },
		"other commit": func(r *Result) { r.Commit = strings.Repeat("e", len(r.Commit)) },
	} {
		r := result
		mutate(&r)
		if err := importResult(context.Background(), repo, bundle, r, "markitect/check-"+strings.ReplaceAll(name, " ", "-")); err == nil {
			t.Errorf("%s: imported", name)
		}
	}
	// The verified result itself imports once more under another name.
	if err := importResult(context.Background(), repo, bundle, result, "markitect/again"); err != nil {
		t.Fatal(err)
	}
	if err := importResult(context.Background(), repo, bundle, result, "markitect/again"); err == nil {
		t.Fatal("import overwrote an existing branch")
	}
}

func logOf(l Launch) string {
	if l.Directory == "" {
		return ""
	}
	data, _ := os.ReadFile(filepath.Join(l.Directory, logFile))
	return string(data)
}
