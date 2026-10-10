package exchangecli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

func testInvocation(t *testing.T, role string) (agentexec.Invocation, []byte) {
	t.Helper()
	invocation, wire, err := agentexec.PrepareInvocation(agentexec.Request{Role: role, SourceRevision: strings.Repeat("a", 40),
		ModelDigest: "sha256:" + strings.Repeat("b", 64), ModulePin: "pin", ProjectionID: "projection",
		ScopeIDs: []string{"manager"}, PolicyIDs: []string{}, Context: json.RawMessage(`{"kind":"exchange-test"}`), Artifacts: []agentexec.Artifact{}})
	if err != nil {
		t.Fatal(err)
	}
	return invocation, wire
}

// exchangeObservation is what the external party saw while answering.
type exchangeObservation struct {
	request      []byte
	reasons      []string
	rejectedKept bool
}

// respondWhenRequested plays the external party: it waits for request.json and
// then writes each response in turn after the previous one was set aside. It
// reports what it saw, because the adapter removes the exchange on success.
func respondWhenRequested(t *testing.T, dir, runID string, responses ...string) <-chan exchangeObservation {
	t.Helper()
	exchangeDir := filepath.Join(dir, runID)
	observed := make(chan exchangeObservation, 1)
	go func() {
		var observation exchangeObservation
		for index, response := range responses {
			for {
				_, requestErr := os.Stat(filepath.Join(exchangeDir, RequestFile))
				_, responseErr := os.Stat(filepath.Join(exchangeDir, ResponseFile))
				if requestErr == nil && os.IsNotExist(responseErr) {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if index == 0 {
				observation.request, _ = os.ReadFile(filepath.Join(exchangeDir, RequestFile))
			} else {
				// Wait until the adapter explained why it set the previous response aside.
				for {
					if reason, err := os.ReadFile(filepath.Join(exchangeDir, ErrorFile)); err == nil {
						observation.reasons = append(observation.reasons, string(reason))
						break
					}
					time.Sleep(20 * time.Millisecond)
				}
				_, err := os.Stat(filepath.Join(exchangeDir, fmt.Sprintf("response.rejected-%d.json", index)))
				observation.rejectedKept = err == nil
				_ = os.Remove(filepath.Join(exchangeDir, ErrorFile))
			}
			if err := writeAtomically(filepath.Join(exchangeDir, ResponseFile), []byte(response)); err != nil {
				t.Error(err)
				return
			}
		}
		observed <- observation
	}()
	return observed
}

func TestExchangeWritesExactRequestAndCompletesTheResponseEnvelope(t *testing.T) {
	dir := t.TempDir()
	invocation, wire := testInvocation(t, agentexec.RoleExecutor)
	observed := respondWhenRequested(t, dir, invocation.RunID, `{"outcome":"proposed","reportJson":{"summary":"<html> & more"}}`)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	completed, err := Exchange(ctx, dir, 100*time.Millisecond, wire, io.Discard)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if observation := <-observed; !bytes.Equal(observation.request, wire) {
		t.Fatal("request.json does not hold the exact invocation bytes")
	}
	response, err := agentexec.DecodeResponse(completed, invocation, "")
	if err != nil {
		t.Fatalf("completed response was rejected by the Host decoder: %v\n%s", err, completed)
	}
	if response.Usage != nil || response.RunID != invocation.RunID || response.Nonce != invocation.Nonce || len(response.CandidateFiles) != 0 {
		t.Fatalf("unexpected completed response: %+v", response)
	}
	if !bytes.Contains(completed, []byte("<html> & more")) {
		t.Fatalf("completed response escaped markup and grew: %s", completed)
	}
	if _, err := os.Stat(filepath.Join(dir, invocation.RunID)); !os.IsNotExist(err) {
		t.Fatalf("exchange copy of the scoped request was not removed: %v", err)
	}
}

func TestExchangeSetsInvalidResponsesAsideAndKeepsWaiting(t *testing.T) {
	dir := t.TempDir()
	invocation, wire := testInvocation(t, agentexec.RoleVerifier)
	oversized := `{"outcome":"passed","uncertainty":["` + strings.Repeat("x", maxResponseBytes) + `"]}`
	observed := respondWhenRequested(t, dir, invocation.RunID,
		`{"nonce":"replayed","outcome":"passed","verifierObservations":[{"subject":"manager","outcome":"passed","detail":"checked"}]}`,
		oversized,
		`{"outcome":"passed","verifierObservations":[{"subject":"manager","outcome":"passed","detail":"checked"}]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	completed, err := Exchange(ctx, dir, 100*time.Millisecond, wire, io.Discard)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if _, err := agentexec.DecodeResponse(completed, invocation, ""); err != nil {
		t.Fatalf("corrected response was not valid: %v", err)
	}
	observation := <-observed
	if len(observation.reasons) != 2 || !strings.Contains(observation.reasons[0], "nonce does not match this invocation") ||
		!strings.Contains(observation.reasons[1], "response exceeds") || !observation.rejectedKept {
		t.Fatalf("rejections were not explained and kept: %+v", observation.reasons)
	}
}

func TestExchangeStopsWithItsCallerAndRejectsUnsafeInputs(t *testing.T) {
	dir := t.TempDir()
	_, wire := testInvocation(t, agentexec.RoleExecutor)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if _, err := Exchange(ctx, dir, 100*time.Millisecond, wire, io.Discard); err == nil {
		t.Fatal("Exchange returned without a response")
	}
	if _, err := Exchange(context.Background(), "relative", 100*time.Millisecond, wire, io.Discard); err == nil {
		t.Fatal("relative exchange directory was accepted")
	}
	if _, err := Exchange(context.Background(), dir, 100*time.Millisecond, []byte(`{"apiVersion":"x"}`), io.Discard); err == nil {
		t.Fatal("invalid invocation envelope was accepted")
	}
	var stderr bytes.Buffer
	if code := Run([]string{"--dir", dir, "--poll", "1ms"}, bytes.NewReader(wire), io.Discard, &stderr); code != 2 {
		t.Fatalf("out-of-range poll interval exit = %d", code)
	}
}

const (
	exchangeProcessEnv = "MARKITECT_EXCHANGE_TEST_PROCESS"
	exchangeDirEnv     = "MARKITECT_EXCHANGE_TEST_DIR"
)

// TestExchangeExecutorProcess is re-executed as the process executor.
func TestExchangeExecutorProcess(t *testing.T) {
	if os.Getenv(exchangeProcessEnv) != "1" {
		return
	}
	os.Exit(Run([]string{"--dir", os.Getenv(exchangeDirEnv), "--poll", "100ms"}, os.Stdin, os.Stdout, os.Stderr))
}

func TestExchangeExecutorThroughTheProcessTransport(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Setenv(exchangeProcessEnv, "1")
	t.Setenv(exchangeDirEnv, dir)
	allowlist := []string{"PATH", "SystemRoot", exchangeProcessEnv, exchangeDirEnv}
	config := agentexec.Config{Command: executable, Args: []string{"-test.run=^TestExchangeExecutorProcess$"}, Model: "person", ProviderVersion: "exchange-test",
		Timeout: time.Minute, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20, EnvironmentAllowlist: &allowlist}
	invocation, _ := testInvocation(t, agentexec.RoleExecutor)
	go func() {
		// The external party answers the first request that appears.
		for deadline := time.Now().Add(50 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			entries, _ := os.ReadDir(dir)
			for _, entry := range entries {
				exchangeDir := filepath.Join(dir, entry.Name())
				if _, err := os.Stat(filepath.Join(exchangeDir, RequestFile)); err == nil {
					_ = writeAtomically(filepath.Join(exchangeDir, ResponseFile), []byte(`{"outcome":"proposed","reportJson":{"status":"complete"}}`))
					return
				}
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	result, err := agentexec.Run(ctx, config, invocation.Request, agentexec.RunOptions{PrivateLogDirectory: filepath.Join(t.TempDir(), "private")})
	if err != nil {
		t.Fatalf("process transport rejected the exchange executor: %v", err)
	}
	if result.Response.Outcome != agentexec.OutcomeProposed || result.Receipt.Usage != nil || result.Receipt.PrivateLogDigest == "" {
		t.Fatalf("unexpected exchange result: response=%+v receipt=%+v", result.Response, result.Receipt)
	}
}
