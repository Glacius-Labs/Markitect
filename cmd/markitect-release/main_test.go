package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGHRunnerPreservesHTTPErrorStdoutWithoutMergingStderr(t *testing.T) {
	if os.Getenv("MARKITECT_GH_RUNNER_HELPER") == "1" {
		fmt.Fprint(os.Stdout, "HTTP/2.0 404 Not Found\r\nContent-Type: application/json\r\n\r\n{\"message\":\"Not Found\"}")
		fmt.Fprint(os.Stderr, "PRIVATE_STDERR_SENTINEL")
		os.Exit(1)
	}

	t.Setenv("MARKITECT_GH_RUNNER_HELPER", "1")
	runner := ghRunner{
		executable: os.Args[0],
		prefixArgs: []string{"-test.run=^TestGHRunnerPreservesHTTPErrorStdoutWithoutMergingStderr$"},
	}
	stdout, err := runner.Run(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "status 1") {
		t.Fatalf("nonzero helper process error = %v", err)
	}
	if got, want := string(stdout), "HTTP/2.0 404 Not Found\r\nContent-Type: application/json\r\n\r\n{\"message\":\"Not Found\"}"; got != want {
		t.Fatalf("captured stdout = %q, want exact HTTP response %q", got, want)
	}
	if strings.Contains(string(stdout), "PRIVATE_STDERR_SENTINEL") || strings.Contains(err.Error(), "PRIVATE_STDERR_SENTINEL") {
		t.Fatal("stderr leaked into the API response or surfaced error")
	}
}
