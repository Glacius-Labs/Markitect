package agentexec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestCandidateFileDeletionIsAnExplicitWireOperation(t *testing.T) {
	var file CandidateFile
	if err := strictDecode([]byte(`{"path":"src/old.txt","delete":true}`), &file); err != nil {
		t.Fatalf("decode explicit deletion: %v", err)
	}
	if file.Path != "src/old.txt" || !file.Delete || file.Mode != "" || file.Content != "" {
		t.Fatalf("decoded deletion is not explicit: %+v", file)
	}
	wire, err := json.Marshal(CandidateFile{Path: "src/old.txt", Delete: true})
	if err != nil || string(wire) != `{"path":"src/old.txt","delete":true}` {
		t.Fatalf("marshal deletion wire form = %s, err=%v", wire, err)
	}

	for _, raw := range []string{
		`{"path":"src/old.txt","delete":true,"mode":"0644"}`,
		`{"path":"src/old.txt","delete":true,"content":""}`,
		`{"path":"src/old.txt","delete":true,"mode":null}`,
		`{"path":"src/old.txt","delete":true,"content":null}`,
		`{"path":"src/old.txt","delete":true,"extra":false}`,
		`{"path":"src/new.txt","mode":"0644","content":"","delete":false}`,
		`{"path":"src/new.txt","mode":"0644","content":"","delete":null}`,
	} {
		t.Run(raw, func(t *testing.T) {
			var got CandidateFile
			if err := strictDecode([]byte(raw), &got); err == nil {
				t.Fatalf("mixed or invalid candidate operation fields were accepted: %s", raw)
			}
		})
	}

	var write CandidateFile
	if err := strictDecode([]byte(`{"path":"src/new.txt","mode":"0644","content":""}`), &write); err != nil {
		t.Fatalf("decode empty write: %v", err)
	}
	if write.Delete || write.Mode != "0644" || write.Content != "" {
		t.Fatalf("write was confused with deletion: %+v", write)
	}
}

func TestValidateResponseAcceptsDeletionButRejectsMixedOperation(t *testing.T) {
	request := Request{Role: RoleExecutor}
	invocation := Invocation{APIVersion: APIVersion, RunID: "run", Nonce: "nonce", InputDigest: "input"}
	response := Response{
		APIVersion: APIVersion, RunID: "run", Nonce: "nonce", Role: RoleExecutor,
		InputDigest: "input", Outcome: OutcomeProposed,
		CandidateFiles: []CandidateFile{{Path: "src/old.txt", Delete: true}},
		EvidenceRefs:   []string{}, VerifierObservations: []Observation{}, Uncertainty: []string{},
	}
	if err := validateResponse(response, request, invocation, ""); err != nil {
		t.Fatalf("valid deletion response rejected: %v", err)
	}

	response.CandidateFiles[0].Content = "unexpected"
	if _, err := json.Marshal(response); err == nil || !strings.Contains(err.Error(), "cannot include mode or content") {
		t.Fatalf("delete plus content field was serialized: %v", err)
	}
	if err := validateResponse(response, request, invocation, ""); err == nil || !strings.Contains(err.Error(), "cannot include mode or content") {
		t.Fatalf("delete plus content was accepted: %v", err)
	}

	response.CandidateFiles[0] = CandidateFile{Path: "src/old.txt", Delete: true, Mode: "0644"}
	if err := validateResponse(response, request, invocation, ""); err == nil || !strings.Contains(err.Error(), "cannot include mode or content") {
		t.Fatalf("delete plus mode was accepted: %v", err)
	}
}

func TestNativeDeltaDigestBindsDeleteOperationWithoutEmptyWriteShape(t *testing.T) {
	deletion := []CandidateFile{{Path: "src/old.txt", Delete: true}}
	got, err := nativeDeltaDigest(deletion)
	if err != nil {
		t.Fatal(err)
	}
	// This literal is the independently specified canonical operation-only
	// delete row. It intentionally contains no digest or mode fields.
	canonicalDelete := []byte(`[{"delete":true,"path":"src/old.txt"}]`)
	sum := sha256.Sum256(canonicalDelete)
	want := "sha256:" + hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("delete delta digest = %s, want canonical operation digest %s", got, want)
	}

	emptyWriteDigest, err := nativeDeltaDigest([]CandidateFile{{Path: "src/old.txt", Mode: "0644", Content: ""}})
	if err != nil {
		t.Fatal(err)
	}
	if got == emptyWriteDigest {
		t.Fatal("deletion delta digest collides with an empty write")
	}
}
