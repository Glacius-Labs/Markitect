package exchangecli

import (
	"bytes"
	"context"
	"encoding/json"
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

// respondWhenRequested plays the external party: it waits for request.json and
// then writes each response in turn after the previous one was consumed.
func respondWhenRequested(t *testing.T, dir, runID string, responses ...string) {
	t.Helper()
	exchangeDir := filepath.Join(dir, runID)
	go func() {
		for index, response := range responses {
			for {
				_, requestErr := os.Stat(filepath.Join(exchangeDir, RequestFile))
				_, responseErr := os.Stat(filepath.Join(exchangeDir, ResponseFile))
				if requestErr == nil && os.IsNotExist(responseErr) {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if index > 0 {
				// Wait until the adapter set the previous response aside.
				for {
					if _, err := os.Stat(filepath.Join(exchangeDir, ErrorFile)); err == nil {
						break
					}
					time.Sleep(20 * time.Millisecond)
				}
			}
			if err := writeAtomically(filepath.Join(exchangeDir, ResponseFile), []byte(response)); err != nil {
				t.Error(err)
				return
			}
		}
	}()
}

func TestExchangeWritesExactRequestAndCompletesTheResponseEnvelope(t *testing.T) {
	dir := t.TempDir()
	invocation, wire := testInvocation(t, agentexec.RoleExecutor)
	respondWhenRequested(t, dir, invocation.RunID, `{"outcome":"proposed","reportJson":{"status":"complete"}}`)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	completed, err := Exchange(ctx, dir, 100*time.Millisecond, wire, io.Discard)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	request, err := os.ReadFile(filepath.Join(dir, invocation.RunID, RequestFile))
	if err != nil || !bytes.Equal(request, wire) {
		t.Fatalf("request.json does not hold the exact invocation bytes: err=%v", err)
	}
	response, err := agentexec.DecodeResponse(completed, invocation, "")
	if err != nil {
		t.Fatalf("completed response was rejected by the Host decoder: %v\n%s", err, completed)
	}
	if response.Usage != nil || response.RunID != invocation.RunID || response.Nonce != invocation.Nonce || len(response.CandidateFiles) != 0 {
		t.Fatalf("unexpected completed response: %+v", response)
	}
}

func TestExchangeSetsInvalidResponsesAsideAndKeepsWaiting(t *testing.T) {
	dir := t.TempDir()
	invocation, wire := testInvocation(t, agentexec.RoleVerifier)
	respondWhenRequested(t, dir, invocation.RunID,
		`{"nonce":"replayed","outcome":"passed","verifierObservations":[{"subject":"manager","outcome":"passed","detail":"checked"}]}`,
		`{"outcome":"passed","verifierObservations":[{"subject":"manager","outcome":"passed","detail":"checked"}]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	completed, err := Exchange(ctx, dir, 100*time.Millisecond, wire, io.Discard)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if _, err := agentexec.DecodeResponse(completed, invocation, ""); err != nil {
		t.Fatalf("corrected response was not valid: %v", err)
	}
	reason, err := os.ReadFile(filepath.Join(dir, invocation.RunID, ErrorFile))
	if err != nil || !strings.Contains(string(reason), "nonce does not match this invocation") {
		t.Fatalf("rejection reason = %q, %v", reason, err)
	}
	if _, err := os.Stat(filepath.Join(dir, invocation.RunID, "response.rejected-1.json")); err != nil {
		t.Fatalf("rejected response was not kept: %v", err)
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
