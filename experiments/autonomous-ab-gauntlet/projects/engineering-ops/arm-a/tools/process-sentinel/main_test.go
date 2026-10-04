package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestObserveReturnsObservedChildExitCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := observe(ctx, os.Args[0], []string{"-test.run=TestChildProcessHelper"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestObserveReportsChildFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	t.Setenv("PROCESS_SENTINEL_CHILD", "fail")
	result, err := observe(ctx, os.Args[0], []string{"-test.run=TestChildProcessHelper"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", result.ExitCode)
	}
}

func TestObserveReportsCancellationDeadline(t *testing.T) {
	t.Setenv("PROCESS_SENTINEL_CHILD", "wait")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	result, err := observe(ctx, os.Args[0], []string{"-test.run=TestChildProcessHelper"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 124 {
		t.Fatalf("exit code = %d, want timeout code 124", result.ExitCode)
	}
}

func TestChildProcessHelper(t *testing.T) {
	switch os.Getenv("PROCESS_SENTINEL_CHILD") {
	case "fail":
		os.Exit(7)
	case "wait":
		time.Sleep(time.Second)
	}
}
