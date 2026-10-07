package government

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func bindingTestDigest(label string) string { return BytesDigest([]byte(label)) }
func bindingTestIdentity(kind, name string) core.DefinitionIdentity {
	return core.DefinitionIdentity{APIVersion: APIVersion, Kind: kind, Namespace: "test", Name: name}
}
func bindingTestCabinetMember(name string) CabinetMember {
	return CabinetMember{
		Ressort:       bindingTestIdentity("Ressort", name),
		PriorMandate:  bindingTestIdentity("Mandate", "mandate-"+name),
		MandateDigest: bindingTestDigest("mandate-version-" + name),
		SlotID:        "slot-" + name,
	}
}
func bindingTestCandidate(t *testing.T) MaterialCandidate {
	t.Helper()
	c, err := NewMaterialCandidate(MaterialCandidateInput{
		PriorConstitutionDigest: bindingTestDigest("constitution"),
		BaseRevision:            strings.Repeat("a", 40), RepositoryTreeDigest: bindingTestDigest("tree"),
		ModelDigest: bindingTestDigest("model"), PlanDigest: bindingTestDigest("plan"),
		CheckDefinitionsDigest: bindingTestDigest("checks"), ToolPinsDigest: bindingTestDigest("tools"),
		InventoryDigest: bindingTestDigest("inventory"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func bindingTestEvidence(t *testing.T, c MaterialCandidate, round uint64, result string) Evidence {
	t.Helper()
	e, err := NewEvidence(c, round, []string{bindingTestDigest(result)}, []string{bindingTestDigest("report")})
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func bindingTestVote(t *testing.T, c MaterialCandidate, e Evidence, ressort, outcome string) RessortVote {
	t.Helper()
	v, err := NewRessortVote(c, e, bindingTestIdentity("Mandate", "mandate-"+ressort), bindingTestDigest("mandate-version-"+ressort), bindingTestIdentity("Ressort", ressort), VoteOutcome(outcome), "reviewed exact candidate and evidence", VoteProvenance{RunID: "run-" + ressort, SlotID: "slot-" + ressort})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestBindingDomainsAndEvidenceRoundInvalidation(t *testing.T) {
	candidate := bindingTestCandidate(t)
	changed := candidate.Input
	changed.ModelDigest = bindingTestDigest("changed model bytes")
	changedCandidate, err := NewMaterialCandidate(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedCandidate.ID == candidate.ID {
		t.Fatal("changed model bytes retained MaterialCandidateID")
	}
	e1 := bindingTestEvidence(t, candidate, 1, "result-1")
	e2 := bindingTestEvidence(t, candidate, 2, "result-1")
	e3 := bindingTestEvidence(t, candidate, 1, "changed-result")
	if e1.ID == e2.ID || e1.ID == e3.ID {
		t.Fatal("round or result changes must invalidate EvidenceID")
	}
	if _, err := NewRessortVote(changedCandidate, e1, bindingTestIdentity("Mandate", "m"), bindingTestDigest("m"), bindingTestIdentity("Ressort", "r"), VoteAssent, "ok", VoteProvenance{"run", "slot"}); err == nil {
		t.Fatal("stale evidence accepted for changed candidate")
	}
	if _, err := NewRessortVote(candidate, e2, bindingTestIdentity("Mandate", "m"), bindingTestDigest("m"), bindingTestIdentity("Ressort", "r"), VoteAssent, "ok", VoteProvenance{"run", "slot"}); err != nil {
		t.Fatalf("round-bound evidence should make a valid new vote: %v", err)
	}
}

func TestDecisionRequiresCompleteUniqueUnanimousFrozenCabinet(t *testing.T) {
	c := bindingTestCandidate(t)
	e := bindingTestEvidence(t, c, 1, "result")
	memberA, memberB := bindingTestCabinetMember("architecture"), bindingTestCabinetMember("security")
	cabinet := []CabinetMember{memberA, memberB}
	a := bindingTestVote(t, c, e, "architecture", string(VoteAssent))
	b := bindingTestVote(t, c, e, "security", string(VoteAssentUnaffected))
	a.Provenance.SlotID = memberA.SlotID
	a.ID = bindingHash("ressort-vote/v1", votePayload(a))
	b.Provenance.SlotID = memberB.SlotID
	b.ID = bindingHash("ressort-vote/v1", votePayload(b))
	d, err := NewAcceptanceDecision(c, e, c.Input.PriorConstitutionDigest, cabinet, []RessortVote{b, a})
	if err != nil {
		t.Fatalf("complete assent set rejected: %v", err)
	}
	if len(d.VoteIDs) != 2 || d.VoteIDs[0] > d.VoteIDs[1] {
		t.Fatalf("decision vote IDs are not complete and sorted: %#v", d.VoteIDs)
	}
	if _, err := NewAcceptanceDecision(c, e, c.Input.PriorConstitutionDigest, cabinet, []RessortVote{a}); err == nil {
		t.Fatal("absent cabinet vote accepted")
	}
	if _, err := NewAcceptanceDecision(c, e, c.Input.PriorConstitutionDigest, cabinet, []RessortVote{a, a}); err == nil {
		t.Fatal("duplicate role/vote accepted")
	}
	for _, outcome := range []VoteOutcome{VoteObjection, VoteIncomplete} {
		bad := bindingTestVote(t, c, e, "security", string(outcome))
		bad.Provenance.SlotID = memberB.SlotID
		bad.ID = bindingHash("ressort-vote/v1", votePayload(bad))
		if _, err := NewAcceptanceDecision(c, e, c.Input.PriorConstitutionDigest, cabinet, []RessortVote{a, bad}); err == nil {
			t.Fatalf("%s vote accepted", outcome)
		}
	}
	newEvidence := bindingTestEvidence(t, c, 2, "result")
	if _, err := NewAcceptanceDecision(c, newEvidence, c.Input.PriorConstitutionDigest, cabinet, []RessortVote{a, b}); err == nil {
		t.Fatal("votes from prior evidence round accepted")
	}
	if _, err := NewAcceptanceDecision(c, e, bindingTestDigest("different authority"), cabinet, []RessortVote{a, b}); err == nil {
		t.Fatal("decision authority differing from candidate's prior Constitution accepted")
	}
	wrongMandate := b
	wrongMandate.PriorMandate = bindingTestIdentity("Mandate", "different")
	wrongMandate.ID = bindingHash("ressort-vote/v1", votePayload(wrongMandate))
	if _, err := NewAcceptanceDecision(c, e, c.Input.PriorConstitutionDigest, cabinet, []RessortVote{a, wrongMandate}); err == nil {
		t.Fatal("vote with a different prior mandate accepted")
	}
	wrongSlot := b
	wrongSlot.Provenance.SlotID = "other-slot"
	wrongSlot.ID = bindingHash("ressort-vote/v1", votePayload(wrongSlot))
	if _, err := NewAcceptanceDecision(c, e, c.Input.PriorConstitutionDigest, cabinet, []RessortVote{a, wrongSlot}); err == nil {
		t.Fatal("vote from a different configured slot accepted")
	}
	duplicateCabinet := []CabinetMember{memberA, memberA}
	if _, err := NewAcceptanceDecision(c, e, c.Input.PriorConstitutionDigest, duplicateCabinet, []RessortVote{a, b}); err == nil {
		t.Fatal("duplicate Ressort in frozen cabinet accepted")
	}
	missingReason := b
	missingReason.Reason = " "
	missingReason.ID = bindingHash("ressort-vote/v1", votePayload(missingReason))
	if _, err := NewAcceptanceDecision(c, e, c.Input.PriorConstitutionDigest, cabinet, []RessortVote{a, missingReason}); err == nil {
		t.Fatal("manually constructed vote without reason accepted")
	}
}

func TestBindingConstructorsRejectMalformedInputs(t *testing.T) {
	bad := MaterialCandidateInput{BaseRevision: "short"}
	if _, err := NewMaterialCandidate(bad); err == nil {
		t.Fatal("incomplete candidate accepted")
	}
	c := bindingTestCandidate(t)
	if _, err := NewEvidence(c, 0, nil, []string{bindingTestDigest("report")}); err == nil {
		t.Fatal("zero evidence round accepted")
	}
	e := bindingTestEvidence(t, c, 1, "result")
	if _, err := NewRessortVote(c, e, core.DefinitionIdentity{}, bindingTestDigest("m"), bindingTestIdentity("Ressort", "r"), VoteAssent, "ok", VoteProvenance{"run", "slot"}); err == nil {
		t.Fatal("missing mandate accepted")
	}
	malformedCandidate := c
	malformedCandidate.Input.ModelDigest = "not-a-digest"
	malformedCandidate.ID = bindingHash("material-candidate/v1", malformedCandidate.Input)
	if _, err := NewEvidence(malformedCandidate, 1, nil, []string{bindingTestDigest("report")}); err == nil {
		t.Fatal("candidate with malformed input and matching hand-computed ID accepted")
	}
	if _, err := NewRessortVote(c, e, bindingTestIdentity("Other", "m"), bindingTestDigest("m"), bindingTestIdentity("Ressort", "r"), VoteAssent, "ok", VoteProvenance{"run", "slot"}); err == nil {
		t.Fatal("non-Mandate identity accepted")
	}
}
