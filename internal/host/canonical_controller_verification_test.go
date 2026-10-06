package host

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

func TestCanonicalControllerVerifierRequiresExactEvidenceReferences(t *testing.T) {
	request := agentexec.Request{
		ScopeIDs:  []string{"core/v1:Rule:policy", "core/v1:UseCase:orders"},
		PolicyIDs: []string{"core/v1:Rule:boundary"},
		Artifacts: []agentexec.Artifact{{Path: "docs/orders.md"}},
	}
	observations := []agentexec.Observation{{Subject: "docs/orders.md", Outcome: agentexec.OutcomePassed, Detail: "selected output reviewed"}}
	exact := []string{"docs/orders.md", "core/v1:Rule:boundary", "core/v1:UseCase:orders", "core/v1:Rule:policy"}
	if !canonicalControllerExactEvidenceRefs(exact, observations, request) {
		t.Fatal("exact selected scope, policy, and artifact references were rejected")
	}
	for _, refs := range [][]string{
		{"docs/orders.md", "core/v1:Rule:boundary", "core/v1:UseCase:orders"},
		{"docs/orders.md", "core/v1:Rule:boundary", "core/v1:UseCase:orders", "core/v1:Rule:policy", "README.md"},
		{"docs/orders.md", "core/v1:Rule:boundary", "core/v1:UseCase:orders", "core/v1:UseCase:orders"},
	} {
		if canonicalControllerExactEvidenceRefs(refs, observations, request) {
			t.Fatalf("inexact evidence references were accepted: %#v", refs)
		}
	}
	if canonicalControllerExactEvidenceRefs(exact, []agentexec.Observation{{Subject: "README.md", Outcome: agentexec.OutcomePassed, Detail: "outside"}}, request) {
		t.Fatal("observation subject outside the selected scope was accepted")
	}
}

func TestCanonicalControllerComposedOutcomeRetainsFailureAndEscalation(t *testing.T) {
	cases := []struct {
		left, right, want string
	}{
		{records.OutcomePassed, records.OutcomeIncomplete, records.OutcomeIncomplete},
		{records.OutcomeEscalated, records.OutcomeIncomplete, records.OutcomeEscalated},
		{records.OutcomeFailed, records.OutcomeEscalated, records.OutcomeFailed},
		{records.OutcomePassed, records.OutcomePassed, records.OutcomePassed},
	}
	for _, tc := range cases {
		if got := composeVerificationOutcomes(tc.left, tc.right); got != tc.want {
			t.Errorf("composeVerificationOutcomes(%q, %q) = %q, want %q", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestCanonicalControllerAssuranceRejectsParentMissingChildSourceIDs(t *testing.T) {
	parent := controllerVerificationTestRecord(t, "core/v1:Projection:parent", "core/v1:UseCase:orders")
	child := controllerVerificationTestRecord(t, "core/v1:Projection:child", "core/v1:Handler:billing")
	cfg := CanonicalControllerConfig{
		AssuranceRoots: []string{"parent"},
		AssuranceScopes: []CanonicalAssuranceScope{
			{ID: "parent", ProjectionID: parent.ProjectionID, Children: []string{"child"}, Checks: []authoring.Check{{Name: "parent-check", Run: []string{"go", "version"}}}},
			{ID: "child", ProjectionID: child.ProjectionID, Checks: []authoring.Check{{Name: "child-check", Run: []string{"go", "version"}}}},
		},
	}
	_, _, _, err := canonicalControllerVerificationNodes(cfg, []records.ProjectionRecord{parent, child})
	if err == nil || !strings.Contains(err.Error(), "core/v1:Handler:billing") {
		t.Fatalf("missing child canonical source identity was not reported: %v", err)
	}
}

func controllerVerificationTestRecord(t *testing.T, projectionID, scopeID string) records.ProjectionRecord {
	t.Helper()
	digest := sha256Prefix(sha256Hex([]byte(projectionID + scopeID)))
	record, err := records.NewProjectionRecord(records.ProjectionRecord{
		Revision: strings.Repeat("a", 40), ModelDigest: digest, PlanDigest: digest,
		InputSnapshotDigest: digest, RequestDigest: digest,
		ProjectionID: projectionID,
		Module:       records.ModuleIdentity{Name: "module", Version: "1", Digest: digest},
		Projector:    records.ProjectorIdentity{ID: "markdown", Version: "1"},
		ScopeIDs:     []string{scopeID}, PolicyIDs: []string{},
		Artifacts: []records.Artifact{{Path: "docs/" + strings.TrimPrefix(projectionID, "core/v1:Projection:") + ".md", Digest: digest, Mode: "100644", Change: records.ChangeCreated}},
		State:     records.StateMaterializedUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}
