package host

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

func TestCanonicalCandidateDependenciesAreExplicitBoundedAndDeduplicated(t *testing.T) {
	cfg := CanonicalControllerConfig{AssuranceScopes: []CanonicalAssuranceScope{
		{ID: "top", ProjectionID: "projection/top", Children: []string{"right", "left"}},
		{ID: "left", ProjectionID: "projection/left", Children: []string{"leaf"}},
		{ID: "right", ProjectionID: "projection/right", Children: []string{"leaf"}},
		{ID: "leaf", ProjectionID: "projection/leaf"},
	}}
	actual, err := canonicalControllerDescendantProjections(cfg, "projection/top")
	expected := []string{"projection/leaf", "projection/left", "projection/right"}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fatalf("explicit descendants: %v %v", actual, err)
	}
	cfg.AssuranceScopes[3].Children = []string{"top"}
	if _, err := canonicalControllerDescendantProjections(cfg, "projection/top"); err == nil {
		t.Fatal("cycle accepted")
	}
	cfg.AssuranceScopes[3].Children = []string{"missing"}
	if _, err := canonicalControllerDescendantProjections(cfg, "projection/top"); err == nil {
		t.Fatal("missing child accepted")
	}
}

func TestCanonicalCandidateStagingKeepsObservedInputsImmutable(t *testing.T) {
	observed := &snapshot.Snapshot{ID: "working-tree", Provisional: true, Files: map[string][]byte{"child/code.cs": []byte("old")}, Modes: map[string]string{"child/code.cs": snapshot.RegularMode}}
	stage := cloneCanonicalCandidateSnapshot(observed)
	candidate := []byte("proposed child")
	stageCanonicalCandidate(stage, map[string][]byte{"child/code.cs": candidate})
	candidate[0] = 'X'
	if !bytes.Equal(observed.Files["child/code.cs"], []byte("old")) || !bytes.Equal(stage.Files["child/code.cs"], []byte("proposed child")) {
		t.Fatal("candidate bytes mutated observed inputs or retained caller aliases")
	}
}

func TestCanonicalExecutorReceivesChildCandidatesWithoutTransferringOwnership(t *testing.T) {
	cfg := CanonicalControllerConfig{AssuranceScopes: []CanonicalAssuranceScope{
		{ID: "parent", ProjectionID: "p", Children: []string{"child"}}, {ID: "child", ProjectionID: "c"},
	}}
	stage := &snapshot.Snapshot{Files: map[string][]byte{"child/code.cs": []byte("candidate")}, Modes: map[string]string{"child/code.cs": snapshot.RegularMode}}
	p := CanonicalScopedProposal{ProjectionID: "p"}
	artifacts, dependencies, err := canonicalControllerExecutorArtifacts(cfg, p, nil, stage, map[string]map[string][]byte{"c": {"child/code.cs": []byte("candidate")}}, map[string]bool{"c": true})
	if err != nil || len(artifacts) != 1 || len(dependencies) != 1 || len(p.Request.TargetFiles) != 0 {
		t.Fatalf("candidate evidence boundary: %+v %v %v", artifacts, dependencies, err)
	}
	if _, _, err := canonicalControllerExecutorArtifacts(cfg, p, nil, stage, nil, nil); err == nil {
		t.Fatal("missing required child evidence accepted")
	}
	record := records.ProjectionRecord{ProjectionID: "c", Artifacts: []records.Artifact{{Path: "child/code.cs", Digest: sha256Prefix(sha256Hex([]byte("old"))), Mode: snapshot.RegularMode}}}
	if _, _, err := canonicalControllerExecutorArtifacts(cfg, p, []records.ProjectionRecord{record}, stage, nil, nil); err == nil {
		t.Fatal("drifted retained child accepted as evidence")
	}
}

func TestCanonicalExecutorCannotSubstituteOldActiveBytesForFailedChildWork(t *testing.T) {
	cfg := CanonicalControllerConfig{AssuranceScopes: []CanonicalAssuranceScope{
		{ID: "parent", ProjectionID: "p", Children: []string{"child"}}, {ID: "child", ProjectionID: "c"},
	}}
	data := []byte("old valid implementation")
	stage := &snapshot.Snapshot{Files: map[string][]byte{"child/code.cs": data}, Modes: map[string]string{"child/code.cs": snapshot.RegularMode}}
	active := []records.ProjectionRecord{{ProjectionID: "c", Artifacts: []records.Artifact{{Path: "child/code.cs", Mode: snapshot.RegularMode, Digest: sha256Prefix(sha256Hex(data))}}}}
	if _, _, err := canonicalControllerExecutorArtifacts(cfg, CanonicalScopedProposal{ProjectionID: "p"}, active, stage, nil, map[string]bool{"c": true}); err == nil {
		t.Fatal("parent used previous active bytes after scheduled child work failed")
	}
}
