package releasecli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestRunUsesProvidedOutputStreamsAndPreservesExitCodes(t *testing.T) {
	var out, errout bytes.Buffer
	if code := Run([]string{"--help"}, &out, &errout); code != 0 || out.Len() != 0 || !strings.Contains(errout.String(), "Usage of markitect-release") {
		t.Fatalf("help invocation: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"--unknown"}, &out, &errout); code != 2 || out.Len() != 0 || errout.Len() == 0 {
		t.Fatalf("invalid invocation: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
}

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
