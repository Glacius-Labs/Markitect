package projectrun

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

const (
	checkTimingModeEnv   = "MARKITECT_PROJECTRUN_CHECK_TIMING_MODE"
	checkTimingMarkerEnv = "MARKITECT_PROJECTRUN_CHECK_TIMING_MARKER"
)

func TestRunCheckTimeoutFactoryReceivesMinimumAndStartRefusalPreventsLaunch(t *testing.T) {
	executable, digest := pinnedTestExecutable(t)
	marker := filepath.Join(testkit.TempDir(t), "launched")
	t.Setenv(checkTimingModeEnv, "marker")
	t.Setenv(checkTimingMarkerEnv, marker)
	check := timingCheck(executable, digest)
	for _, tc := range []struct {
		name    string
		agent   time.Duration
		maximum time.Duration
		want    time.Duration
	}{
		{name: "agent timeout is lower", agent: 15 * time.Second, maximum: 20 * time.Second, want: 15 * time.Second},
		{name: "maximum is lower", agent: 25 * time.Second, maximum: 20 * time.Second, want: 20 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got time.Duration
			result := runCheckWithTimeoutContext(context.Background(), testkit.TempDir(t), check,
				timingAgent(tc.agent), projectconfig.Duration(tc.maximum),
				func(CheckResult) error { return errors.New("persist refused") },
				func(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
					got = timeout
					return context.WithCancel(parent)
				})
			if got != tc.want {
				t.Fatalf("timeout constructor received %s, want %s", got, tc.want)
			}
			if result.Error != "could not persist check start" {
				t.Fatalf("start refusal result = %q, want persisted-start failure", result.Error)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("check launched despite start refusal; marker stat error = %v", err)
			}
		})
	}
}

func TestRunCheckTimeoutContextClassifiesDeadlineAndCallerCancellation(t *testing.T) {
	executable, digest := pinnedTestExecutable(t)
	check := timingCheck(executable, digest)
	for _, tc := range []struct {
		name        string
		parent      context.Context
		withTimeout func(context.Context, time.Duration) (context.Context, context.CancelFunc)
		wantTimeout bool
	}{
		{
			name:        "deadline",
			parent:      context.Background(),
			wantTimeout: true,
			withTimeout: func(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
				return context.WithDeadline(parent, time.Now().Add(-time.Second))
			},
		},
		{
			name:   "caller cancellation",
			parent: canceledContext(t),
			withTimeout: func(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
				return context.WithTimeout(parent, timeout)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := runCheckWithTimeoutContext(tc.parent, testkit.TempDir(t), check,
				projectconfig.Agent{Timeout: projectconfig.Duration(time.Minute)}, projectconfig.Duration(time.Minute), nil, tc.withTimeout)
			if result.Outcome != "failed" {
				t.Fatalf("runCheck outcome = %q, want failed (error %q)", result.Outcome, result.Error)
			}
			if got := result.Error == "check timed out"; got != tc.wantTimeout {
				t.Fatalf("timeout classification = %v, want %v (error %q)", got, tc.wantTimeout, result.Error)
			}
		})
	}
}

func TestRunCheckOutputOverflowPrecedesProcessResult(t *testing.T) {
	executable, digest := pinnedTestExecutable(t)
	t.Setenv(checkTimingModeEnv, "overflow")
	check := timingCheck(executable, digest)
	result := runCheckWithTimeoutContext(context.Background(), testkit.TempDir(t), check,
		timingAgent(time.Minute), projectconfig.Duration(time.Minute), nil,
		func(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
			return context.WithCancel(parent)
		})
	if result.Error != "check output exceeded 256 KiB" {
		t.Fatalf("overflow result = %q, want output overflow", result.Error)
	}
	if len(result.Stdout) != maxCheckOutputBytes {
		t.Fatalf("captured stdout length = %d, want %d", len(result.Stdout), maxCheckOutputBytes)
	}
}

func TestRunCheckPinPreflightPrecedesTimeoutConstruction(t *testing.T) {
	called := false
	result := runCheckWithTimeoutContext(context.Background(), testkit.TempDir(t), CheckPlan{ID: "unpinned"},
		projectconfig.Agent{Timeout: projectconfig.Duration(time.Second)}, projectconfig.Duration(time.Second), nil,
		func(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
			called = true
			return context.WithCancel(parent)
		})
	if called {
		t.Fatal("timeout constructor ran before executable pin preflight")
	}
	if result.Error != "check executable was not pinned at plan time" {
		t.Fatalf("unpinned check result = %q", result.Error)
	}
}

func TestCheckTimingHelperProcess(t *testing.T) {
	switch os.Getenv(checkTimingModeEnv) {
	case "marker":
		if err := os.WriteFile(os.Getenv(checkTimingMarkerEnv), []byte("started"), 0o600); err != nil {
			t.Fatal(err)
		}
	case "overflow":
		if _, err := os.Stdout.Write(bytes.Repeat([]byte("x"), maxCheckOutputBytes+1)); err != nil {
			t.Fatal(err)
		}
	}
}

func pinnedTestExecutable(t *testing.T) (string, string) {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	resolved, raw, err := readPinnedExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved, rawContentDigest(raw)
}

func timingCheck(executable, digest string) CheckPlan {
	return CheckPlan{
		ID:               "controlled-timeout",
		Owner:            "orders",
		Command:          []string{filepath.Base(executable), "-test.run=^TestCheckTimingHelperProcess$"},
		ExecutablePath:   executable,
		ExecutableDigest: digest,
	}
}

func timingAgent(timeout time.Duration) projectconfig.Agent {
	return projectconfig.Agent{
		Timeout:     projectconfig.Duration(timeout),
		Environment: []string{checkTimingModeEnv, checkTimingMarkerEnv, "SystemRoot"},
	}
}

func canceledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
