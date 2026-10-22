package apply_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

type assignmentControl struct{}

func (assignmentControl) Check(context.Context) error { return nil }

func TestAssignmentPublicationNeedsExactIndependentAcceptance(t *testing.T) {
	req, start := layoutRequest(t)
	pub := &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationNoChange}}
	result := apply.AssignmentRealizationResult{Work: &run.AssignmentTree{}, Tree: start, TreeCommit: "integrated", AcceptedCommit: "integrated", Reviews: []run.AssignmentReviewResult{{AcceptedCommit: "integrated"}}}
	if _, err := apply.PublishAssignment(context.Background(), "run-1", req.Selection, req.Prepared, result, pub, assignmentControl{}); err != nil || len(pub.requests) != 1 || !reflect.DeepEqual(pub.requests[0].Guard, req.Prepared.Structural().Model().Inputs().Guard) {
		t.Fatalf("verified no-change result not bound to captured guard: %v, requests=%+v", err, pub.requests)
	}
	for _, altered := range []apply.AssignmentRealizationResult{
		{Work: result.Work, Tree: start, TreeCommit: "integrated", AcceptedCommit: "other", Reviews: result.Reviews},
		{Work: result.Work, Tree: start, TreeCommit: "integrated", AcceptedCommit: "integrated", Reviews: []run.AssignmentReviewResult{{AcceptedCommit: "other"}}},
		{Work: result.Work, Tree: start, TreeCommit: "integrated", Reviews: result.Reviews},
	} {
		if _, err := apply.PublishAssignment(context.Background(), "run-1", req.Selection, req.Prepared, altered, pub, assignmentControl{}); err == nil {
			t.Fatal("published an unverified assignment result")
		}
	}
	if len(pub.requests) != 1 {
		t.Fatalf("unverified results reached publisher: %d", len(pub.requests))
	}
}

func TestAssignmentGitReservationsCoverManagersAndFiniteReviewCopies(t *testing.T) {
	req, _ := layoutRequest(t)
	ids, err := apply.AssignmentWorkspaceIDs(req.Prepared, "library-test", 3)
	if err != nil || len(ids) < 5 || ids[0] != "library-test-work-0" || ids[1] != "library-test-assess" || ids[len(ids)-1] != "library-test-verify-3" {
		t.Fatalf("reserved IDs = %v, %v", ids, err)
	}
	seen := map[run.WorkspaceID]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate reservation %s", id)
		}
		seen[id] = true
	}
	if _, err := apply.AssignmentWorkspaceIDs(req.Prepared, "library-test", 0); err == nil {
		t.Fatal("zero reviewer bound admitted")
	}
}
