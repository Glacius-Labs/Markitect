package projectrun

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectworkspace"
)

func TestExternalExecutionRequiresExplicitContext(t *testing.T) {
	if IsExternalExecution(context.Background()) {
		t.Fatal("ordinary execution was marked external")
	}
	ctx := ExternalExecutionContext(context.Background())
	if !IsExternalExecution(ctx) {
		t.Fatal("explicit external execution marker was lost")
	}
}

func TestExternalPacketReexportPreservesExactInvocationBytes(t *testing.T) {
	dir := t.TempDir()
	request := externalTestRequest(t)
	plan := PlanRecord{ID: "plan", Digest: "sha256:plan", BaseRevision: "revision", BaseSnapshot: "sha256:base"}
	report := RunReport{ID: "plan"}
	task := ManagerTask{ID: "task-id", ManagerID: "orders", WorkAttempts: 1}
	meta := externalRequestContext{Kind: "projectrun-task/v1", Phase: "work", ManagerID: "orders"}
	store := &runStore{}
	first, err := createOrReadExternalPacket(store, dir, plan, report, task, "orders-work", request, meta)
	if err != nil {
		t.Fatal(err)
	}
	second, err := createOrReadExternalPacket(store, dir, plan, report, task, "orders-work", request, meta)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.Digest != second.Digest || string(first.InvocationBytes) != string(second.InvocationBytes) {
		t.Fatal("re-export changed the frozen packet identity or invocation bytes")
	}
	loaded, err := readExternalPacket(dir, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(loaded.InvocationBytes) != string(first.InvocationBytes) {
		t.Fatal("persisted packet did not preserve its exact invocation bytes")
	}
	otherAttempt, err := createOrReadExternalPacket(store, dir, plan, report, task, "orders-work-2", request, meta)
	if err != nil {
		t.Fatal(err)
	}
	if otherAttempt.ID == first.ID {
		t.Fatal("distinct scheduler attempt IDs shared one packet identity")
	}
}

func TestExternalSubmissionDigestBindsResponseBytes(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "external", "responses"), 0o700); err != nil {
		t.Fatal(err)
	}
	packetID := "0123456789abcdef0123456789abcdef"
	response := []byte(`{"outcome":"proposed"}`)
	submission := externalSubmission{APIVersion: externalPacketVersion, PacketID: packetID,
		PacketDigest: "sha256:packet", ResponseHash: externalResponseDigest(response, nil, nil), Response: response}
	if err := writeImmutableJSON(externalSubmissionPath(dir, packetID), submission); err != nil {
		t.Fatal(err)
	}
	loaded, err := readExternalSubmission(dir, packetID)
	if err != nil || loaded.ResponseHash != externalResponseDigest(response, nil, nil) {
		t.Fatalf("read submitted response = %+v, %v", loaded, err)
	}
	loaded.Response = []byte(`{"outcome":"failed"}`)
	if err := writeImmutableJSON(externalSubmissionPath(dir, packetID), loaded); err == nil {
		t.Fatal("immutable response file accepted replacement bytes")
	}
}

func TestExternalResponseDigestBindsGitMetadataAndDelta(t *testing.T) {
	response := []byte(`{"outcome":"proposed"}`)
	candidate := &ExternalGitCandidate{Root: `C:\repo`, BaseRevision: "base", Revision: "candidate"}
	baseDelta := projectworkspace.Delta{RepositoryIdentity: "repo", BaseSHA: "base", OverlayDigest: "overlay", TaskID: "task", BaseDigest: "snapshot",
		Changes: []projectworkspace.Change{{Kind: projectworkspace.ChangeAdd, Path: "src/new.go", Mode: "100644", Content: []byte("one")}}}
	baseDelta.Digest = externalGitDeltaDigest(baseDelta)
	base := externalResponseDigest(response, candidate, &baseDelta)

	changedCandidate := *candidate
	changedCandidate.Revision = "other"
	if base == externalResponseDigest(response, &changedCandidate, &baseDelta) {
		t.Fatal("response digest did not bind Git candidate revisions")
	}
	changedDelta := baseDelta
	changedDelta.Changes = []projectworkspace.Change{{Kind: projectworkspace.ChangeAdd, Path: "src/new.go", Mode: "100644", Content: []byte("two")}}
	changedDelta.Digest = externalGitDeltaDigest(changedDelta)
	if base == externalResponseDigest(response, candidate, &changedDelta) {
		t.Fatal("response digest did not bind the imported delta")
	}
}

func TestGitCandidateProposalsMustEchoObservedWritesAndDeletes(t *testing.T) {
	delta := projectworkspace.Delta{Changes: []projectworkspace.Change{
		{Kind: projectworkspace.ChangeModify, Path: "src/keep.go", Mode: "100755", Content: []byte("observed")},
		{Kind: projectworkspace.ChangeDelete, Path: "src/remove.go"},
	}}
	matching := []agentexec.CandidateFile{
		{Path: "src/keep.go", Mode: "0755", Content: "observed"},
		{Path: "src/remove.go", Delete: true},
	}
	if err := validateGitCandidateProposals(matching, delta); err != nil {
		t.Fatalf("matching candidate proposals rejected: %v", err)
	}
	for name, proposals := range map[string][]agentexec.CandidateFile{
		"invented":            {{Path: "src/other.go", Mode: "0644", Content: "other"}},
		"wrong mode":          {{Path: "src/keep.go", Mode: "0644", Content: "observed"}},
		"wrong bytes":         {{Path: "src/keep.go", Mode: "0755", Content: "different"}},
		"delete contradicted": {{Path: "src/remove.go", Mode: "0644", Content: "recreated"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateGitCandidateProposals(proposals, delta); err == nil {
				t.Fatal("contradictory candidate proposal was accepted")
			}
		})
	}
}

func externalTestRequest(t *testing.T) agentexec.Request {
	t.Helper()
	contextBytes, err := json.Marshal(externalRequestContext{Kind: "projectrun-task/v1", Phase: "work", ManagerID: "orders"})
	if err != nil {
		t.Fatal(err)
	}
	return agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: "revision",
		ModelDigest:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ModulePin:    "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ProjectionID: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		ScopeIDs:     []string{"orders"}, PolicyIDs: []string{}, Context: contextBytes, Artifacts: []agentexec.Artifact{}}
}
