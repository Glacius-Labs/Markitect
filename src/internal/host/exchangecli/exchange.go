// Package exchangecli implements markitect-exchange-executor, the reference
// bring-your-own executor for the agent-execution/v1alpha1 process transport.
// It hands each invocation to an external party through files: the request is
// written as request.json and the party answers with response.json. The party
// can be a person, a script or another agent session. The Host validates the
// returned candidate and report exactly as for any other executor.
package exchangecli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

const (
	RequestFile  = "request.json"
	ResponseFile = "response.json"
	// ErrorFile explains why the last response.json was rejected; the rejected
	// bytes are kept beside it and the adapter waits for a corrected response.
	ErrorFile = "response-error.txt"

	maxInvocationBytes = 64 << 20
	maxResponseBytes   = 16 << 20
)

// Run executes one invocation: stdin carries the invocation, stdout receives
// the completed response. The Host's role timeout bounds the wait.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("markitect-exchange-executor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", "", "absolute existing exchange directory outside the governed repository")
	poll := flags.Duration("poll", 2*time.Second, "interval between checks for response.json (100ms..1m)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *poll < 100*time.Millisecond || *poll > time.Minute {
		fmt.Fprintln(stderr, "usage: markitect-exchange-executor --dir ABSOLUTE_DIR [--poll DURATION]")
		return 2
	}
	raw, err := io.ReadAll(io.LimitReader(stdin, maxInvocationBytes+1))
	if err != nil || len(raw) > maxInvocationBytes {
		fmt.Fprintln(stderr, "markitect-exchange-executor: invocation could not be read within its bound")
		return 2
	}
	response, err := Exchange(context.Background(), *dir, *poll, raw, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "markitect-exchange-executor: %v\n", err)
		return 2
	}
	if logPath := os.Getenv("MARKITECT_AGENT_PRIVATE_LOG"); logPath != "" {
		// The private log is optional evidence; its digest enters the receipt.
		_ = os.WriteFile(logPath, []byte("exchange response accepted from "+*dir+"\n"), 0o600)
	}
	if _, err := stdout.Write(response); err != nil {
		return 2
	}
	return 0
}

// Exchange writes the invocation to DIR/RUN_ID/request.json and waits for a
// valid DIR/RUN_ID/response.json. The responder may omit the envelope fields
// apiVersion, runId, nonce, role and inputDigest; the adapter copies them from
// the invocation, and any supplied value must match it exactly. Missing arrays
// become empty. An invalid response is renamed aside with an explanation and
// the adapter keeps waiting.
func Exchange(ctx context.Context, dir string, poll time.Duration, raw []byte, notices io.Writer) ([]byte, error) {
	if !filepath.IsAbs(dir) {
		return nil, errors.New("--dir must be an absolute path")
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, errors.New("--dir must name an existing directory")
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		return nil, fmt.Errorf("decode invocation: %w", err)
	}
	if invocation.APIVersion != agentexec.APIVersion || !safeRunID(invocation.RunID) || invocation.Nonce == "" || invocation.InputDigest == "" {
		return nil, errors.New("invocation envelope is invalid")
	}
	exchangeDir := filepath.Join(dir, invocation.RunID)
	if err := os.Mkdir(exchangeDir, 0o700); err != nil {
		return nil, fmt.Errorf("create exchange directory: %w", err)
	}
	if err := writeAtomically(filepath.Join(exchangeDir, RequestFile), raw); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}
	fmt.Fprintf(notices, "markitect-exchange-executor: waiting for %s\n", filepath.Join(exchangeDir, ResponseFile))
	responsePath := filepath.Join(exchangeDir, ResponseFile)
	var previous os.FileInfo
	rejected := 0
	for {
		info, err := os.Stat(responsePath)
		switch {
		case err == nil && previous != nil && info.Size() == previous.Size() && info.ModTime().Equal(previous.ModTime()):
			// Accept only a file that did not change between two polls, so a
			// writer that does not rename atomically is not read half-written.
			response, reason := completeResponse(responsePath, invocation)
			if reason == nil {
				return response, nil
			}
			rejected++
			if err := rejectResponse(exchangeDir, rejected, reason); err != nil {
				return nil, err
			}
			fmt.Fprintf(notices, "markitect-exchange-executor: rejected response %d: %v\n", rejected, reason)
			previous = nil
		case err == nil:
			previous = info
		case errors.Is(err, os.ErrNotExist):
			previous = nil
		default:
			return nil, fmt.Errorf("inspect response: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(poll):
		}
	}
}

// completeResponse fills the envelope and validates the result with the same
// decoder the Host applies, so the responder learns about errors while the
// invocation is still waiting.
func completeResponse(path string, invocation agentexec.Invocation) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(file, maxResponseBytes+1))
	file.Close()
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxResponseBytes)
	}
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&fields); err != nil || fields == nil || decoder.More() {
		return nil, errors.New("response must be exactly one JSON object")
	}
	for name, want := range map[string]string{
		"apiVersion": agentexec.APIVersion, "runId": invocation.RunID, "nonce": invocation.Nonce,
		"role": invocation.Request.Role, "inputDigest": invocation.InputDigest,
	} {
		if supplied, ok := fields[name]; ok {
			var got string
			if json.Unmarshal(supplied, &got) != nil || got != want {
				return nil, fmt.Errorf("%s does not match this invocation", name)
			}
		}
		encoded, _ := json.Marshal(want)
		fields[name] = encoded
	}
	for _, name := range []string{"candidateFiles", "evidenceRefs", "verifierObservations", "uncertainty"} {
		if value, ok := fields[name]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			fields[name] = json.RawMessage("[]")
		}
	}
	completed, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	if _, err := agentexec.DecodeResponse(completed, invocation, ""); err != nil {
		return nil, err
	}
	return completed, nil
}

func rejectResponse(dir string, number int, reason error) error {
	rejectedPath := filepath.Join(dir, fmt.Sprintf("response.rejected-%d.json", number))
	if err := os.Rename(filepath.Join(dir, ResponseFile), rejectedPath); err != nil {
		return fmt.Errorf("set rejected response aside: %w", err)
	}
	message := fmt.Sprintf("%s was rejected and kept as %s: %v\nWrite a corrected %s; the invocation is still waiting.\n", ResponseFile, filepath.Base(rejectedPath), reason, ResponseFile)
	return writeAtomically(filepath.Join(dir, ErrorFile), []byte(message))
}

func writeAtomically(path string, data []byte) error {
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

// safeRunID keeps the Host-generated run ID usable as one directory name.
func safeRunID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
