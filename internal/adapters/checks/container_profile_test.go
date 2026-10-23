package checks

import (
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

func containerProfileInputs(t *testing.T) (ContainerProfile, project.ExecutionCheck, run.CheckRequest) {
	t.Helper()
	want := checkRequest(t)
	profile := ContainerProfile{
		ImageID: want.ImageID, Executable: want.Executable,
		ArgumentPrefix: []string{want.Args[0]}, Env: append([]string(nil), want.Env...),
		Limits: want.Limits, MaxOutputBytes: want.MaxOutputBytes,
	}
	check := project.ExecutionCheck{
		ID:      organization.Setting[string]{Value: string(want.ID)},
		Profile: organization.Setting[string]{Value: "explicit-runtime-profile"},
	}
	check.Directory.Value = want.Directory
	check.Args.Value = append([]string(nil), want.Args[1:]...)
	check.Timeout.Value = want.Timeout
	return profile, check, want
}

func TestContainerProfileProjectsExactRequestAndOwnsSlices(t *testing.T) {
	profile, check, want := containerProfileInputs(t)
	got, err := profile.Request(check, want.Tree)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("request=%+v err=%v want=%+v", got, err, want)
	}
	profile.ArgumentPrefix[0] = "changed prefix"
	check.Args.Value[0] = "changed project argument"
	profile.Env[0] = "CHANGED=caller"
	if !reflect.DeepEqual(got, want) {
		t.Fatal("caller mutation changed the projected request")
	}
	got.Args[0], got.Args[1], got.Env[0] = "request prefix", "request argument", "CHANGED=request"
	if profile.ArgumentPrefix[0] != "changed prefix" || check.Args.Value[0] != "changed project argument" || profile.Env[0] != "CHANGED=caller" {
		t.Fatal("request mutation changed caller slices")
	}
}

func TestContainerProfileRequestReachesExistingRunner(t *testing.T) {
	profile, check, want := containerProfileInputs(t)
	req, err := profile.Request(check, want.Tree)
	if err != nil {
		t.Fatal(err)
	}
	workspaces := &workspacePeer{}
	calls := 0
	runner, err := NewRunnerWithRuntime(commandPeer{func(ctx context.Context, spec container.ContainerSpec, args, env []string, max int64) (container.CommandResult, error) {
		calls++
		if !workspaces.request.Tree.Equal(want.Tree) || workspaces.request.Tree.Digest() != want.Tree.Digest() {
			t.Fatal("runner did not materialize the actual immutable tree")
		}
		if spec.Image != want.ImageID || spec.Workdir != "/work/pkg" || spec.CPUs != 1.25 || spec.MemoryBytes != want.Limits.MemoryBytes || spec.Pids != int(want.Limits.Processes) {
			t.Fatalf("container spec=%+v", spec)
		}
		if !reflect.DeepEqual(args, append([]string{want.Executable}, want.Args...)) || !reflect.DeepEqual(env, want.Env) || max != want.MaxOutputBytes {
			t.Fatal("runner changed the projected literal command or bounds")
		}
		if _, bounded := ctx.Deadline(); !bounded {
			t.Fatal("runner omitted the declared timeout")
		}
		return passedCommand(want), nil
	}}, workspaces, nil)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := runner.Run(context.Background(), req)
	if err != nil || !receipt.Passed(want) || calls != 1 || workspaces.closed != 2 {
		t.Fatalf("receipt=%+v err=%v calls=%d closed=%d", receipt, err, calls, workspaces.closed)
	}
}

func TestContainerProfileRefusesInvalidValuesBeforeEffects(t *testing.T) {
	for _, kind := range []string{"missing image", "unclean executable", "NUL prefix", "NUL project argument", "duplicate environment", "missing limits", "missing timeout", "missing output bound"} {
		t.Run(kind, func(t *testing.T) {
			profile, check, want := containerProfileInputs(t)
			switch kind {
			case "missing image":
				profile.ImageID = ""
			case "unclean executable":
				profile.Executable = "/usr/bin/../bin/check"
			case "NUL prefix":
				profile.ArgumentPrefix = []string{"bad\x00prefix"}
			case "NUL project argument":
				check.Args.Value = []string{"bad\x00argument"}
			case "duplicate environment":
				profile.Env = []string{"A=first", "A=second"}
			case "missing limits":
				profile.Limits = run.CheckLimits{}
			case "missing timeout":
				check.Timeout.Value = 0
			case "missing output bound":
				profile.MaxOutputBytes = 0
			}
			workspaces := &workspacePeer{}
			runner, err := NewRunnerWithRuntime(commandPeer{func(context.Context, container.ContainerSpec, []string, []string, int64) (container.CommandResult, error) {
				t.Fatal("invalid configuration reached runtime")
				return container.CommandResult{}, nil
			}}, workspaces, nil)
			if err != nil {
				t.Fatal(err)
			}
			req, err := profile.Request(check, want.Tree)
			if err == nil || !reflect.DeepEqual(req, run.CheckRequest{}) {
				t.Fatalf("invalid projection request=%+v err=%v", req, err)
			}
			if _, err := runner.Run(context.Background(), req); err == nil || len(workspaces.requests) != 0 || workspaces.closed != 0 {
				t.Fatalf("invalid request caused effects: err=%v workspaces=%+v", err, workspaces)
			}
		})
	}
}
