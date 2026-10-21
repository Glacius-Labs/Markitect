package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func setupArgs(t *testing.T) (runtimeDir, credential string, args []string) {
	t.Helper()
	root := t.TempDir()
	runtimeDir, credential = filepath.Join(root, "runtime"), filepath.Join(root, "token")
	return runtimeDir, credential, []string{"setup", "--repo", filepath.Join(root, "repo"), "--runtime", runtimeDir, "--image", "runtime:1", "--credential", credential, "--engine", filepath.Join(root, "docker"), "--agent-model", "claude-haiku-5-5"}
}

func runSetup(c *Command, args ...string) (code int, stdout, stderr string) {
	var out, errout bytes.Buffer
	code = c.Run(context.Background(), args, &out, &errout)
	return code, out.String(), errout.String()
}

func TestSetupParsesFlagsAndReportsTheApplyCommand(t *testing.T) {
	runtimeDir, credential, args := setupArgs(t)
	var got SetupRequest
	c := New(nil, nil, nil, nil, nil, nil).WithSetup(func(_ context.Context, r SetupRequest) (SetupReport, error) {
		got = r
		return SetupReport{Status: model.Passed, Files: []SetupFile{{Path: filepath.Join(runtimeDir, "runtime.yaml"), SHA256: "ab", Action: "written"}}}, nil
	})
	args = append(args, "--check", "go=/usr/local/go/bin/go", "--check", "lint=/bin/lint", "--deadline", "2030-01-01T00:00:00Z", "--store", "s", "--request", "R", "--dependency", "dep/model.yaml")
	code, out, errout := runSetup(c, args...)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errout)
	}
	if got.RuntimeDir != runtimeDir || got.Credential != credential || got.Image != "runtime:1" || got.AgentModel != "claude-haiku-5-5" || got.Effort != "high" || !strings.HasSuffix(got.Engine, "docker") || got.Store != "s" || got.OwnerRequest != "R" || got.Deadline != "2030-01-01T00:00:00Z" || got.ProductBuild == "" ||
		!reflect.DeepEqual(got.Checks, map[string]string{"go": "/usr/local/go/bin/go", "lint": "/bin/lint"}) || len(got.Selection.Dependencies) != 1 || got.Selection.Local.ManifestPath.String() != ".markitect/model.yaml" {
		t.Fatalf("%+v", got)
	}
	for _, want := range []string{`"operation": "setup"`, `"status": "passed"`, `"applyCommand"`, `--runtime-profile`, `--run-config`, `--dependency`, `"agentProfiles": []`} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %s:\n%s", want, out)
		}
	}
}

func TestSetupRefusesInvalidInvocationsBeforeAnyEffect(t *testing.T) {
	_, _, args := setupArgs(t)
	called := false
	c := New(nil, nil, nil, nil, nil, nil).WithSetup(func(context.Context, SetupRequest) (SetupReport, error) {
		called = true
		return SetupReport{}, nil
	})
	for name, bad := range map[string][]string{
		"no runtime directory":       {"setup", "--image", "i", "--credential", "/abs", "--agent-model", "m"},
		"no model":                   withoutFlag(args, "--agent-model"),
		"relative runtime directory": append(append([]string{}, args...), "--runtime", "relative"),
		"relative token":             append(append([]string{}, args...), "--credential", "token"),
		"relative engine":            append(append([]string{}, args...), "--engine", "docker"),
		"malformed check":            append(append([]string{}, args...), "--check", "go"),
		"empty executable":           append(append([]string{}, args...), "--check", "go="),
		"duplicate check":            append(append([]string{}, args...), "--check", "go=/a", "--check", "go=/b"),
		"stray argument":             append(append([]string{}, args...), "stray"),
		"unknown flag":               append(append([]string{}, args...), "--unknown"),
		"retired owner decision":     append(append([]string{}, args...), "--decision", "Use euro."),
	} {
		if code, out, _ := runSetup(c, bad...); code != 2 || called {
			t.Errorf("%s: exit %d called=%v %s", name, code, called, out)
		}
	}
	unavailable := New(nil, nil, nil, nil, nil, nil)
	if code, out, _ := runSetup(unavailable, args...); code != 3 || !strings.Contains(out, "unavailable") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestSetupExitCodesFollowTheReportStatus(t *testing.T) {
	_, _, args := setupArgs(t)
	for status, want := range map[model.Status]int{model.Failed: 1, model.NotCompleted: 3} {
		c := New(nil, nil, nil, nil, nil, nil).WithSetup(func(context.Context, SetupRequest) (SetupReport, error) {
			return SetupReport{Status: status}, nil
		})
		code, out, _ := runSetup(c, args...)
		if code != want || strings.Contains(out, "applyCommand") {
			t.Errorf("%s: exit %d: %s", status, code, out)
		}
	}
}

func TestApplyCommandQuotesOnlyWhatTheShellNeeds(t *testing.T) {
	plain := applyCommand("/r", ".markitect/model.yaml", nil, "/h/runtime.yaml", "/h/run.json")
	if plain != "markitect apply --repo /r --runtime-profile /h/runtime.yaml --run-config /h/run.json" {
		t.Fatal(plain)
	}
	custom := applyCommand("/r", "m/model.yaml", []string{"/d/model.yaml"}, "/h x/runtime.yaml", "/h x/run.json")
	if !strings.Contains(custom, "--model m/model.yaml --dependency /d/model.yaml") || !strings.Contains(custom, "h x/runtime.yaml") || strings.Count(custom, "h x") != 2 || strings.Contains(custom, "--runtime-profile /h x") {
		t.Fatal(custom)
	}
}

func withoutFlag(args []string, flag string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		if args[i] == flag {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}
