package projectrun

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestReviewCandidateChangesGroundsDeletionWithPriorBytes(t *testing.T) {
	base := &Snapshot{Files: map[string][]byte{
		"src/deleted.txt": []byte("prior contents"),
		"src/changed.txt": []byte("before"),
	}, Modes: map[string]string{
		"src/deleted.txt": snapshot.RegularMode,
		"src/changed.txt": snapshot.RegularMode,
	}}
	candidate := candidateData{Files: map[string]File{
		"src/added.txt":   {Path: "src/added.txt", Mode: "0644", Content: []byte("new")},
		"src/changed.txt": {Path: "src/changed.txt", Mode: "0644", Content: []byte("after")},
		"src/deleted.txt": {Path: "src/deleted.txt", Mode: "0644", Content: []byte("prior contents"), Delete: true},
	}}
	changes, deleted, err := reviewCandidateChanges(base, candidate, []string{"src/added.txt", "src/changed.txt", "src/deleted.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 3 || changes[0].Path != "src/added.txt" || changes[0].Operation != "add" ||
		changes[1].Path != "src/changed.txt" || changes[1].Operation != "modify" ||
		changes[2].Path != "src/deleted.txt" || changes[2].Operation != "delete" {
		t.Fatalf("unexpected typed change manifest: %+v", changes)
	}
	if changes[2].PriorDigest != rawContentDigest([]byte("prior contents")) || changes[2].PriorMode != "0644" {
		t.Fatalf("deletion is not grounded in exact prior bytes and mode: %+v", changes[2])
	}
	if len(deleted) != 1 || deleted[0].Path != "src/deleted.txt" || string(deleted[0].Content) != "prior contents" ||
		deleted[0].Digest != rawContentDigest([]byte("prior contents")) || deleted[0].Mode != "0644" {
		t.Fatalf("deletion evidence does not carry the exact prior file: %+v", deleted)
	}
}

func TestReviewContextAndScopeBindDeletedPathOperationAndPreimage(t *testing.T) {
	const managerID = "shop/orders"
	const path = "src/orders/implementation.txt"
	prior := []byte("prior implementation")
	base := &Snapshot{Files: map[string][]byte{path: prior}, Modes: map[string]string{path: snapshot.RegularMode}}
	candidate := candidateData{ID: "candidate", Files: map[string]File{
		path: {Path: path, Mode: "0644", Content: append([]byte(nil), prior...), Delete: true},
	}}
	project := &Project{
		Config:   projectwork.Config{InventoryRoots: []string{"src"}},
		Snapshot: &snapshot.Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}},
		Report: projectmodel.Report{
			Managers:  []projectmodel.Manager{{ID: managerID}},
			Artifacts: []projectmodel.Artifact{{ID: "orders-implementation", Owner: managerID, Required: true, Paths: []string{path}}},
			Files:     []projectmodel.FileEntry{{Path: path, Owner: managerID, Class: "source", Artifacts: []string{"orders-implementation"}}},
		},
	}
	task := ManagerTask{ManagerID: managerID, Goal: "Assess order implementation.", Artifacts: []string{"orders-implementation"}, WrittenPaths: []string{path}}
	plan := PlanRecord{Goal: task.Goal, Managers: []ManagerTask{task}}
	state := reviewCandidateState{Base: base, Candidate: candidate}
	context, files, refs, err := buildReviewerContext(plan, project, task, "work", 1, candidate, BriefingContext{}, RunReport{}, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(context.CandidateChanges) != 1 || context.CandidateChanges[0].Operation != "delete" ||
		context.CandidateChanges[0].PriorDigest != rawContentDigest(prior) || context.CandidateChanges[0].PriorMode != "0644" {
		t.Fatalf("review context omitted the typed, grounded deletion: %+v", context.CandidateChanges)
	}
	if len(files) != 1 || files[0].Path != path || string(files[0].Content) != string(prior) || files[0].Digest != rawContentDigest(prior) {
		t.Fatalf("review request does not carry exact non-empty prior bytes: %+v", files)
	}
	if len(refs) != 1 || refs[0].Operation != "delete" || refs[0].Digest != rawContentDigest(prior) ||
		!containsString(refs[0].Grounding, "operation:delete") || !containsString(refs[0].Grounding, "prior-file-bytes") {
		t.Fatalf("deleted path review ref is not explicitly grounded: %+v", refs)
	}
	if _, err := validateReviewFindings([]reviewFindingResponse{{Path: path, Expectation: "Retain required behavior.", Grounding: "operation:delete"}}, refs); err != nil {
		t.Fatalf("typed deletion reference cannot ground a finding: %v", err)
	}

	initialScope, err := reviewScopeDigest(plan, project, task, "work", RunReport{}, state)
	if err != nil {
		t.Fatal(err)
	}
	changedBase := &Snapshot{Files: map[string][]byte{path: []byte("different prior bytes")}, Modes: map[string]string{path: snapshot.RegularMode}}
	changedScope, err := reviewScopeDigest(plan, project, task, "work", RunReport{}, reviewCandidateState{Base: changedBase, Candidate: candidate})
	if err != nil {
		t.Fatal(err)
	}
	if changedScope == initialScope {
		t.Fatal("review freshness digest ignored changed prior bytes for a deletion")
	}
}

func TestReviewCandidateChangesUsesRetainedDeletionPreimageAndRequiresGrounding(t *testing.T) {
	candidate := candidateData{Files: map[string]File{
		"src/empty.txt": {Path: "src/empty.txt", Mode: "0644", Content: []byte{}, Delete: true},
	}}
	changes, deleted, err := reviewCandidateChanges(&Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}, candidate, []string{"src/empty.txt"})
	if err != nil {
		t.Fatalf("valid empty-file deletion preimage rejected: %v", err)
	}
	if len(changes) != 1 || changes[0].Operation != "delete" || changes[0].PriorDigest != rawContentDigest(nil) ||
		len(deleted) != 1 || len(deleted[0].Content) != 0 || deleted[0].Mode != "0644" {
		t.Fatalf("empty prior file was not represented as a grounded deletion: changes=%+v artifacts=%+v", changes, deleted)
	}

	missing := candidateData{Files: map[string]File{"src/missing.txt": {Path: "src/missing.txt", Delete: true}}}
	if _, _, err := reviewCandidateChanges(&Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}, missing, []string{"src/missing.txt"}); err == nil {
		t.Fatal("ungrounded deletion without prior bytes or mode was accepted")
	}
}
