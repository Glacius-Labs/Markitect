package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The test executable doubles as a portable fake engine. Every invocation
// consumes one scripted response and records its literal arguments on disk.
type engineStep struct {
	Out, Err    string
	Code        int
	SleepMillis int
}

func TestMain(m *testing.M) {
	if dir := os.Getenv("MARKITECT_FAKE_COMMAND_ENGINE"); dir != "" {
		data, _ := os.ReadFile(filepath.Join(dir, "steps.json"))
		var steps []engineStep
		_ = json.Unmarshal(data, &steps)
		indexData, _ := os.ReadFile(filepath.Join(dir, "index"))
		index, _ := strconv.Atoi(string(indexData))
		if index >= len(steps) {
			fmt.Fprintln(os.Stderr, "unexpected engine call")
			os.Exit(99)
		}
		_ = os.WriteFile(filepath.Join(dir, "index"), []byte(strconv.Itoa(index+1)), 0600)
		args, _ := json.Marshal(os.Args[1:])
		f, _ := os.OpenFile(filepath.Join(dir, "calls"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		_, _ = f.Write(append(args, '\n'))
		_ = f.Close()
		step := steps[index]
		if strings.Contains(step.Out, "CREATION_TOKEN") {
			data, _ := os.ReadFile(filepath.Join(dir, "calls"))
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var recorded []string
				_ = json.Unmarshal([]byte(line), &recorded)
				for _, arg := range recorded {
					if token, ok := strings.CutPrefix(arg, creationLabel+"="); ok {
						step.Out = strings.ReplaceAll(step.Out, "CREATION_TOKEN", token)
					}
				}
			}
		}
		time.Sleep(time.Duration(step.SleepMillis) * time.Millisecond)
		fmt.Fprint(os.Stdout, step.Out)
		fmt.Fprint(os.Stderr, step.Err)
		os.Exit(step.Code)
	}
	os.Exit(m.Run())
}

func fakeCommandEngine(t *testing.T, steps ...engineStep) (*Engine, func() [][]string) {
	t.Helper()
	dir := t.TempDir()
	data, _ := json.Marshal(steps)
	if err := os.WriteFile(filepath.Join(dir, "steps.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARKITECT_FAKE_COMMAND_ENGINE", dir)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	engine, err := NewEngine(exe)
	if err != nil {
		t.Fatal(err)
	}
	return engine, func() [][]string {
		data, err := os.ReadFile(filepath.Join(dir, "calls"))
		if err != nil {
			t.Fatal(err)
		}
		var calls [][]string
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			var args []string
			if err := json.Unmarshal([]byte(line), &args); err != nil {
				t.Fatal(err)
			}
			calls = append(calls, args)
		}
		return calls
	}
}

const commandID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const commandImage = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func commandSpec() ContainerSpec {
	return ContainerSpec{Name: "check-test", Image: commandImage, User: "1000:1000", Labels: map[string]string{"io.markitect.store": "test"}, Workdir: "/work", Mounts: []Mount{{Volume: "copy", Target: "/work"}}, CPUs: 1.5, MemoryBytes: 123456, Pids: 32, TmpBytes: 8192, Network: "none"}
}

func commandState(status string, started bool, exit int) string {
	start := "0001-01-01T00:00:00Z"
	if started {
		start = "2026-10-17T12:00:00Z"
	}
	return fmt.Sprintf(`[{"Id":%q,"Image":%q,"State":{"Status":%q,"Running":%v,"StartedAt":%q,"FinishedAt":"2026-10-17T12:00:01Z","ExitCode":%d}}]`, commandID, commandImage, status, status == "running", start, exit)
}

func TestRunCommandUsesObservedExitAndLiteralInvocation(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: "stdout", Err: "stderr", Code: 7}, engineStep{Out: commandState("exited", true, 7)}, engineStep{})
	result, err := e.RunCommand(context.Background(), commandSpec(), []string{"/usr/bin/check", "--rm", "space $literal"}, []string{"CHECK=value with spaces"}, 20)
	if err != nil || !result.Started || !result.Exited || result.ExitCode != 7 || result.Uncertain || result.ImageID != commandImage {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !strings.Contains(string(result.Output), "stdout") || !strings.Contains(string(result.Output), "stderr") {
		t.Fatalf("combined output %q", result.Output)
	}
	all := calls()
	first := all[0]
	if first[0] != "create" || !reflect.DeepEqual(first[len(first)-3:], []string{commandImage, "--rm", "space $literal"}) {
		t.Fatalf("literal args %q", first)
	}
	for _, want := range []string{"--read-only", "--cpus", "1.5", "--memory", "123456", "--pids-limit", "32", "--network", "none", "type=volume,src=copy,dst=/work", "CHECK=value with spaces", "/usr/bin/check"} {
		if !containsArg(first, want) {
			t.Errorf("missing %q in %q", want, first)
		}
	}
	if !reflect.DeepEqual(all[3], []string{"rm", "--force", commandID}) {
		t.Fatalf("cleanup %q", all[3])
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func TestRunCommandTruncatesCombinedOutput(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: "123456789", Err: "abcdefgh"}, engineStep{Out: commandState("exited", true, 0)}, engineStep{})
	r, err := e.RunCommand(context.Background(), commandSpec(), []string{"/check"}, nil, 5)
	if err != nil || !r.Truncated || len(r.Output) != 5 {
		t.Fatalf("result=%+v err=%v", r, err)
	}
}

func TestRunCommandDoesNotInferExitFromCLIZero(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{}, engineStep{Out: commandState("created", false, 0)}, engineStep{})
	r, err := e.RunCommand(context.Background(), commandSpec(), []string{"/check"}, nil, 5)
	if err == nil || r.Started || r.Exited || !r.Uncertain {
		t.Fatalf("result=%+v err=%v", r, err)
	}
}

func TestRunCommandCancellationStillObservesAndCleans(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{SleepMillis: 10000}, engineStep{Out: commandState("running", true, 0)}, engineStep{})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	r, err := e.RunCommand(ctx, commandSpec(), []string{"/check"}, nil, 10)
	if !errors.Is(err, context.DeadlineExceeded) || !r.Started || r.Exited || !r.Uncertain || len(calls()) != 4 {
		t.Fatalf("result=%+v err=%v calls=%q", r, err, calls())
	}
}

func TestRunCommandRetainsCleanupUncertainty(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{}, engineStep{Out: commandState("exited", true, 0)}, engineStep{Code: 1, Err: "cleanup denied"})
	r, err := e.RunCommand(context.Background(), commandSpec(), []string{"/check"}, nil, 10)
	if err == nil || !r.Exited || !r.Uncertain || !strings.Contains(r.Reason, "cleanup") {
		t.Fatalf("result=%+v err=%v", r, err)
	}
}

func TestRunCommandRetainsAttachmentFailureWithKnownExit(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Code: 125, Err: "attachment failed"}, engineStep{Out: commandState("exited", true, 7)}, engineStep{})
	r, err := e.RunCommand(context.Background(), commandSpec(), []string{"/check"}, nil, 10)
	if err == nil || !r.Started || !r.Exited || r.ExitCode != 7 || !r.Uncertain {
		t.Fatalf("result=%+v err=%v", r, err)
	}
}

func TestRunCommandCancellationRetainsUncertaintyWithKnownExit(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{SleepMillis: 10000}, engineStep{Out: commandState("exited", true, 0)}, engineStep{})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	r, err := e.RunCommand(ctx, commandSpec(), []string{"/check"}, nil, 10)
	if !errors.Is(err, context.DeadlineExceeded) || !r.Exited || !r.Uncertain {
		t.Fatalf("result=%+v err=%v", r, err)
	}
}

func TestRunCommandRefusesIncompleteStateObservation(t *testing.T) {
	out := strings.Replace(commandState("exited", true, 0), `,"ExitCode":0`, "", 1)
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{}, engineStep{Out: out}, engineStep{})
	r, err := e.RunCommand(context.Background(), commandSpec(), []string{"/check"}, nil, MaxCommandOutputBytes)
	if err == nil || r.Exited || !r.Uncertain {
		t.Fatalf("result=%+v err=%v", r, err)
	}
}

func TestRunCommandRefusesInvalidInputsBeforeEngine(t *testing.T) {
	e := &Engine{path: "must-not-run"}
	for _, env := range [][]string{{"BAD"}, {"A=1", "A=2"}, {"A=\x00"}} {
		if _, err := e.RunCommand(context.Background(), commandSpec(), []string{"/check"}, env, 10); err == nil {
			t.Fatalf("accepted %q", env)
		}
	}
}
