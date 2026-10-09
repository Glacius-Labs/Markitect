package host

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/records"
)

func TestCanonicalVerifierRequiredObservationSubjectsAreTypedAndDeterministic(t *testing.T) {
	checks := []records.CheckIdentity{
		{ID: "lint", Version: "check/v1", Digest: "sha256:" + strings.Repeat("a", 64)},
	}
	first, err := canonicalVerifierRequiredObservationSubjects(
		[]string{"core/v1:UseCase:orders"}, []string{"core/v1:ProjectionPolicy:orders"}, []string{"docs/orders.md"}, checks,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := canonicalVerifierRequiredObservationSubjects(
		[]string{"core/v1:UseCase:orders"}, []string{"core/v1:ProjectionPolicy:orders"}, []string{"docs/orders.md"}, checks,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 4 {
		t.Fatalf("required subjects=%d, want scope, policy, artifact, and check", len(first))
	}
	for i := range first {
		if first[i] != second[i] || i > 0 && first[i-1].Subject >= first[i].Subject {
			t.Fatalf("subjects are not deterministic and sorted: %#v %#v", first, second)
		}
	}
	byKind := map[string]canonicalVerifierObservationSubject{}
	for _, item := range first {
		byKind[item.Kind] = item
	}
	if byKind["scope"].Subject != "scope:core/v1:UseCase:orders" ||
		byKind["policy"].Subject != "policy:core/v1:ProjectionPolicy:orders" ||
		byKind["artifact"].Subject != "artifact:docs/orders.md" {
		t.Fatalf("human-readable typed identities were lost: %#v", first)
	}
	check := byKind["check"]
	if !strings.HasPrefix(check.Subject, "check:") || check.ID != checks[0].ID || check.Version != checks[0].Version || check.Digest != checks[0].Digest {
		t.Fatalf("check subject does not preserve its exact human-readable identity: %#v", check)
	}
}

func TestCanonicalVerifierPassRequiresCompletePassingObservationCoverage(t *testing.T) {
	required := canonicalTestVerifierSubjects(t)
	full := observationsForCanonicalVerifierSubjects(required, agentexec.OutcomePassed)
	if err := validateCanonicalVerifierObservationCoverage(full, required, agentexec.OutcomePassed); err != nil {
		t.Fatalf("full passing coverage rejected: %v", err)
	}
	if err := validateCanonicalVerifierObservationCoverage(full[:1], required, agentexec.OutcomePassed); err == nil || !strings.Contains(err.Error(), "omitted required observations") {
		t.Fatalf("partial passing coverage was accepted: %v", err)
	}
	for i := range full {
		negative := append([]agentexec.Observation(nil), full...)
		negative[i].Outcome = agentexec.OutcomeFailed
		if err := validateCanonicalVerifierObservationCoverage(negative, required, agentexec.OutcomePassed); err == nil || !strings.Contains(err.Error(), "non-passing observation") {
			t.Fatalf("PASS accepted negative outcome for %q: %v", full[i].Subject, err)
		}
		incomplete := append([]agentexec.Observation(nil), full...)
		incomplete[i].Outcome = agentexec.OutcomeIncomplete
		if err := validateCanonicalVerifierObservationCoverage(incomplete, required, agentexec.OutcomePassed); err == nil || !strings.Contains(err.Error(), "non-passing observation") {
			t.Fatalf("PASS accepted incomplete outcome for %q: %v", full[i].Subject, err)
		}
	}
}

func TestCanonicalVerifierCoverageRejectsDuplicateUnexpectedAndInvalidSubjects(t *testing.T) {
	required := canonicalTestVerifierSubjects(t)
	full := observationsForCanonicalVerifierSubjects(required, agentexec.OutcomePassed)
	duplicate := append(append([]agentexec.Observation(nil), full...), full[0])
	if err := validateCanonicalVerifierObservationCoverage(duplicate, required, agentexec.OutcomePassed); err == nil || !strings.Contains(err.Error(), "duplicate verifier observation") {
		t.Fatalf("duplicate observation was accepted: %v", err)
	}
	extra := append(append([]agentexec.Observation(nil), full...), agentexec.Observation{Subject: "policy:unselected", Outcome: agentexec.OutcomePassed, Detail: "extra"})
	if err := validateCanonicalVerifierObservationCoverage(extra, required, agentexec.OutcomePassed); err == nil || !strings.Contains(err.Error(), "unexpected verifier observation") {
		t.Fatalf("unexpected observation was accepted: %v", err)
	}
	for _, invalid := range [][]canonicalVerifierObservationSubject{
		{required[0], {Subject: "", Kind: "scope", ID: "blank"}},
		{required[0], required[0]},
	} {
		if err := validateCanonicalVerifierObservationCoverage(full, invalid, agentexec.OutcomePassed); err == nil {
			t.Fatalf("invalid required subjects were accepted: %#v", invalid)
		}
	}
	if err := validateCanonicalVerifierObservationCoverage(nil, nil, agentexec.OutcomePassed); err == nil {
		t.Fatal("vacuous PASS with no required subjects was accepted")
	}
}

func TestCanonicalVerifierFailureAndIncompleteMayRetainPartialObservations(t *testing.T) {
	required := canonicalTestVerifierSubjects(t)
	partial := []agentexec.Observation{{Subject: required[0].Subject, Outcome: agentexec.OutcomeFailed, Detail: "failed evidence"}}
	for _, outcome := range []string{agentexec.OutcomeFailed, agentexec.OutcomeIncomplete, agentexec.OutcomeEscalated} {
		if err := validateCanonicalVerifierObservationCoverage(partial, required, outcome); err != nil {
			t.Fatalf("partial observations for %s were rejected: %v", outcome, err)
		}
	}
	for _, outcome := range []string{agentexec.OutcomeFailed, agentexec.OutcomeIncomplete, agentexec.OutcomeEscalated} {
		if err := validateCanonicalVerifierObservationCoverage([]agentexec.Observation{{Subject: "artifact:outside", Outcome: agentexec.OutcomePassed, Detail: "unexpected"}}, required, outcome); err == nil {
			t.Fatalf("%s accepted an unexpected observation subject", outcome)
		}
	}
}

func TestCanonicalVerifierRequiredObservationSubjectsRejectMalformedInputs(t *testing.T) {
	digest := "sha256:" + strings.Repeat("b", 64)
	cases := []struct {
		name      string
		scopes    []string
		policies  []string
		artifacts []string
		checks    []records.CheckIdentity
	}{
		{name: "blank scope", scopes: []string{" "}},
		{name: "duplicate scope", scopes: []string{"same", "same"}},
		{name: "blank artifact", artifacts: []string{""}},
		{name: "duplicate policy", policies: []string{"policy", "policy"}},
		{name: "bad fixed check digest", checks: []records.CheckIdentity{{ID: "test", Version: "v1", Digest: "not-a-digest"}}},
		{name: "duplicate fixed check", checks: []records.CheckIdentity{{ID: "test", Version: "v1", Digest: digest}, {ID: "test", Version: "v1", Digest: digest}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := canonicalVerifierRequiredObservationSubjects(tc.scopes, tc.policies, tc.artifacts, tc.checks); err == nil {
				t.Fatal("malformed observation identities were accepted")
			}
		})
	}
}

func TestCanonicalVerifierRequiredObservationSubjectsRefuseImpossibleWireAggregate(t *testing.T) {
	withinBound := make([]string, agentexec.MaxVerifierObservations)
	for i := range withinBound {
		withinBound[i] = fmt.Sprintf("core/v1:UseCase:scope-%03d", i)
	}
	if subjects, err := canonicalVerifierRequiredObservationSubjects(withinBound, nil, nil, nil); err != nil || len(subjects) != agentexec.MaxVerifierObservations {
		t.Fatalf("exact wire-limit aggregate refused: subjects=%d err=%v", len(subjects), err)
	}
	scopes := make([]string, agentexec.MaxVerifierObservations+1)
	for i := range scopes {
		scopes[i] = fmt.Sprintf("core/v1:UseCase:scope-%03d", i)
	}
	if _, err := canonicalVerifierRequiredObservationSubjects(scopes, nil, nil, nil); err == nil || !strings.Contains(err.Error(), "protocol bound") {
		t.Fatalf("oversized observation aggregate was not refused before invocation: %v", err)
	}
}

func canonicalTestVerifierSubjects(t *testing.T) []canonicalVerifierObservationSubject {
	t.Helper()
	result, err := canonicalVerifierRequiredObservationSubjects(
		[]string{"core/v1:UseCase:orders", "core/v1:Handler:billing"},
		[]string{"core/v1:ProjectionPolicy:orders"},
		[]string{"docs/orders.md", "src/Orders.cs"},
		[]records.CheckIdentity{{ID: "fixed-test", Version: "configured/v1", Digest: "sha256:" + strings.Repeat("c", 64)}},
	)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func observationsForCanonicalVerifierSubjects(subjects []canonicalVerifierObservationSubject, outcome string) []agentexec.Observation {
	result := make([]agentexec.Observation, 0, len(subjects))
	for _, subject := range subjects {
		result = append(result, agentexec.Observation{Subject: subject.Subject, Outcome: outcome, Detail: "test protocol observation"})
	}
	return result
}
