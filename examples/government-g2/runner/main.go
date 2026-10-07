// Command runner is a deterministic, standard-library-only actor for the
// Government G2 process-mechanics example. It demonstrates fresh protocol
// invocations; its assent modes are synthetic and are not human approval or
// evidence of model quality.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const protocolVersion = "markitect.example.org/agent-execution/v1alpha1"

type artifact struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Digest  string `json:"digest"`
	Content []byte `json:"content"`
}

type request struct {
	Role           string          `json:"role"`
	SourceRevision string          `json:"sourceRevision"`
	ModelDigest    string          `json:"modelDigest"`
	ModulePin      string          `json:"modulePin"`
	ProjectionID   string          `json:"projectionId"`
	ScopeIDs       []string        `json:"scopeIds"`
	PolicyIDs      []string        `json:"policyIds"`
	Context        json.RawMessage `json:"context"`
	Artifacts      []artifact      `json:"artifacts"`
}

type invocation struct {
	APIVersion  string  `json:"apiVersion"`
	RunID       string  `json:"runId"`
	Nonce       string  `json:"nonce"`
	InputDigest string  `json:"inputDigest"`
	Request     request `json:"request"`
}

type candidateFile struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Content string `json:"content"`
}

type observation struct {
	Subject string `json:"subject"`
	Outcome string `json:"outcome"`
	Detail  string `json:"detail"`
}

type response struct {
	APIVersion           string          `json:"apiVersion"`
	RunID                string          `json:"runId"`
	Nonce                string          `json:"nonce"`
	Role                 string          `json:"role"`
	InputDigest          string          `json:"inputDigest"`
	Outcome              string          `json:"outcome"`
	CandidateFiles       []candidateFile `json:"candidateFiles"`
	EvidenceRefs         []string        `json:"evidenceRefs"`
	VerifierObservations []observation   `json:"verifierObservations"`
	Uncertainty          []string        `json:"uncertainty"`
}

type actorContext struct {
	Phase     string `json:"phase"`
	Workspace string `json:"workspace"`
	Candidate struct {
		ID string `json:"id"`
	} `json:"candidate"`
	Evidence struct {
		ID    string `json:"id"`
		Round int    `json:"round"`
	} `json:"evidence"`
}

type voteDetail struct {
	Outcome             string `json:"outcome"`
	Reason              string `json:"reason"`
	MaterialCandidateID string `json:"materialCandidateId"`
	EvidenceID          string `json:"evidenceId"`
	Round               int    `json:"round"`
}

const proposedSource = `package inventory

import (
	"errors"
	"math"
)

var ErrOverflow = errors.New("available stock would overflow")

// Release adds released stock to the available count without wrapping.
func Release(available, amount int64) (int64, error) {
	if amount < 0 {
		return available, errors.New("release amount cannot be negative")
	}
	if amount > 0 && available > math.MaxInt64-amount {
		return available, ErrOverflow
	}
	return available + amount, nil
}
`

const proposedTest = `package inventory

import (
	"errors"
	"math"
	"testing"
)

func TestReleaseRejectsOverflow(t *testing.T) {
	if _, err := Release(math.MaxInt64, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("Release(MaxInt64, 1) error = %v, want ErrOverflow", err)
	}
}

func TestReleaseAddsWithinRange(t *testing.T) {
	got, err := Release(8, 3)
	if err != nil || got != 11 {
		t.Fatalf("Release(8, 3) = (%d, %v), want (11, nil)", got, err)
	}
}
`

func main() {
	if len(os.Args) != 2 {
		fatalf("usage: runner executor|verifier|assent|assent-unaffected|objection|stale-vote|missing-review|mutate-candidate|incomplete|wrong-binding")
	}
	var mode = os.Args[1]
	if mode != "executor" && mode != "verifier" && mode != "assent" && mode != "assent-unaffected" &&
		mode != "objection" && mode != "stale-vote" && mode != "missing-review" && mode != "mutate-candidate" &&
		mode != "incomplete" && mode != "wrong-binding" {
		fatalf("unsupported runner mode %q", mode)
	}
	var inv invocation
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 32<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&inv); err != nil {
		fatalf("decode invocation: %v", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		fatalf("invocation must contain exactly one JSON value")
	}
	if inv.APIVersion != protocolVersion || inv.RunID == "" || inv.Nonce == "" || inv.InputDigest == "" {
		fatalf("invocation is missing its protocol binding")
	}
	resp, err := run(mode, inv)
	if err != nil {
		fatalf("%v", err)
	}
	if mode == "wrong-binding" {
		resp.Nonce += "-wrong-binding"
	}
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(resp); err != nil {
		fatalf("encode response: %v", err)
	}
	if mode == "mutate-candidate" {
		if err := mutateWorkspace(inv.Request.Context); err != nil {
			fatalf("mutate candidate after vote response: %v", err)
		}
	}
}

func run(mode string, inv invocation) (response, error) {
	resp := response{
		APIVersion: protocolVersion, RunID: inv.RunID, Nonce: inv.Nonce,
		Role: inv.Request.Role, InputDigest: inv.InputDigest,
		CandidateFiles: []candidateFile{}, EvidenceRefs: []string{},
		VerifierObservations: []observation{}, Uncertainty: []string{},
	}
	switch mode {
	case "executor":
		if inv.Request.Role != "executor" {
			return response{}, errors.New("executor mode requires executor role")
		}
		resp.Outcome = "proposed"
		resp.CandidateFiles = []candidateFile{
			{Path: "inventory/reservation.go", Mode: "0644", Content: proposedSource},
			{Path: "inventory/reservation_test.go", Mode: "0644", Content: proposedTest},
		}
		for _, item := range inv.Request.Artifacts {
			if item.Path == "inventory/reservation.go" || item.Path == "inventory/reservation_test.go" {
				resp.EvidenceRefs = append(resp.EvidenceRefs, item.Path)
			}
		}
		if len(resp.EvidenceRefs) == 0 {
			return response{}, errors.New("executor needs the captured reservation source or test as input")
		}
	case "verifier", "missing-review":
		if inv.Request.Role != "verifier" {
			return response{}, errors.New("verifier mode requires verifier role")
		}
		resp.Outcome = "passed"
		passed, details := inspectCandidate(inv.Request.Artifacts)
		if !passed {
			resp.Outcome = "failed"
		}
		if len(inv.Request.ScopeIDs) == 0 {
			return response{}, errors.New("verifier request has no declared subjects to observe")
		}
		scopeIDs := inv.Request.ScopeIDs
		if mode == "missing-review" {
			scopeIDs = scopeIDs[:1]
		}
		for _, subject := range scopeIDs {
			outcome := "passed"
			if !passed {
				outcome = "failed"
			}
			resp.VerifierObservations = append(resp.VerifierObservations, observation{
				Subject: subject, Outcome: outcome, Detail: strings.Join(details, "; "),
			})
		}
		for _, item := range inv.Request.Artifacts {
			if item.Path == "inventory/reservation.go" || item.Path == "inventory/reservation_test.go" {
				resp.EvidenceRefs = append(resp.EvidenceRefs, item.Path)
			}
		}
	case "assent", "assent-unaffected", "objection", "stale-vote", "mutate-candidate":
		if inv.Request.Role != "verifier" {
			return response{}, errors.New("Ressort modes require verifier role")
		}
		ctx, err := decodeContext(inv.Request.Context)
		if err != nil {
			return response{}, err
		}
		if ctx.Candidate.ID == "" || ctx.Evidence.ID == "" || ctx.Evidence.Round < 1 {
			return response{}, errors.New("Ressort context must bind a material candidate, evidence and positive round")
		}
		outcome := "assent"
		if mode == "assent-unaffected" {
			outcome = "assent-unaffected"
		}
		reason := "deterministic mechanics fixture evaluated the exact candidate and evidence references supplied to this invocation"
		responseOutcome := "passed"
		if mode == "objection" {
			outcome = "objection"
			reason = "deterministic negative fixture withholds assent for this invocation"
			responseOutcome = "failed"
		}
		if mode == "assent-unaffected" {
			reason = "deterministic fixture explicitly records that this Ressort's concern is unaffected by the supplied candidate"
		}
		if mode == "stale-vote" {
			reason = "fault-injection fixture returns an otherwise valid assent bound to deliberately stale evidence"
		}
		if mode == "mutate-candidate" {
			reason = "fault-injection fixture emits a complete valid assent before changing the candidate workspace bytes"
		}
		evidenceID := ctx.Evidence.ID
		if mode == "stale-vote" {
			evidenceID = "sha256:" + strings.Repeat("0", 64)
		}
		body, err := json.Marshal(voteDetail{
			Outcome: outcome, Reason: reason, MaterialCandidateID: ctx.Candidate.ID,
			EvidenceID: evidenceID, Round: ctx.Evidence.Round,
		})
		if err != nil {
			return response{}, err
		}
		resp.Outcome = responseOutcome
		resp.VerifierObservations = []observation{{Subject: "government-vote", Outcome: responseOutcome, Detail: string(body)}}
	case "incomplete":
		resp.Outcome = "incomplete"
		resp.Uncertainty = []string{"fixture deliberately returns no review or vote evidence"}
	case "wrong-binding":
		// Emit an otherwise valid response; the host must reject its changed nonce.
		if inv.Request.Role == "executor" {
			resp.Outcome = "proposed"
			resp.CandidateFiles = []candidateFile{{Path: "inventory/reservation.go", Mode: "0644", Content: proposedSource}}
		} else {
			resp.Outcome = "incomplete"
		}
	default:
		return response{}, errors.New("unsupported mode")
	}
	return resp, nil
}

func decodeContext(raw json.RawMessage) (actorContext, error) {
	if len(raw) == 0 {
		return actorContext{}, errors.New("Ressort invocation has no context")
	}
	var ctx actorContext
	if err := json.Unmarshal(raw, &ctx); err != nil {
		return actorContext{}, fmt.Errorf("decode Ressort context: %w", err)
	}
	return ctx, nil
}

// mutateWorkspace is an intentional fault injection for the Host's final
// candidate-digest check. It is called only after the valid vote response has
// been encoded to stdout.
func mutateWorkspace(raw json.RawMessage) error {
	ctx, err := decodeContext(raw)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(ctx.Workspace) {
		return errors.New("fault injection requires an absolute candidate workspace")
	}
	root := filepath.Clean(ctx.Workspace)
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("candidate workspace is not a regular directory")
	}
	inventoryDir := filepath.Join(root, "inventory")
	dirInfo, err := os.Lstat(inventoryDir)
	if err != nil || !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("candidate inventory path is not a regular directory")
	}
	path := filepath.Join(inventoryDir, "reservation.go")
	fileInfo, err := os.Lstat(path)
	if err != nil || !fileInfo.Mode().IsRegular() || fileInfo.Mode()&os.ModeSymlink != 0 || fileInfo.Size() > 1<<20 {
		return errors.New("candidate source path is not a bounded regular file")
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(file, "\n// fault injection: bytes changed after a complete valid Ressort vote\n")
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func inspectCandidate(artifacts []artifact) (bool, []string) {
	files := make(map[string]artifact, len(artifacts))
	for _, item := range artifacts {
		files[item.Path] = item
	}
	source, haveSource := files["inventory/reservation.go"]
	test, haveTest := files["inventory/reservation_test.go"]
	checks := []struct {
		ok     bool
		detail string
	}{
		{haveSource, "reservation implementation is present"},
		{haveTest, "reservation boundary test is present"},
		{haveSource && bytes.Contains(source.Content, []byte("available > math.MaxInt64-amount")), "implementation checks the MaxInt64 addition boundary"},
		{haveTest && bytes.Contains(test.Content, []byte("Release(math.MaxInt64, 1)")), "test exercises release beyond MaxInt64"},
		{haveTest && bytes.Contains(test.Content, []byte("errors.Is(err, ErrOverflow)")), "test requires the typed overflow error"},
	}
	details := make([]string, 0, len(checks)+2)
	passed := true
	for _, check := range checks {
		if !check.ok {
			passed = false
			details = append(details, "failed: "+check.detail)
		} else {
			details = append(details, "passed: "+check.detail)
		}
	}
	if haveSource {
		details = append(details, "source="+digest(source.Content))
	}
	if haveTest {
		details = append(details, "test="+digest(test.Content))
	}
	return passed, details
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
