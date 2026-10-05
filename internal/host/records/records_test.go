package records

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func testDigest(data string) string {
	sum := sha256.Sum256([]byte(data))
	return "sha256:" + hex.EncodeToString(sum[:])
}
func testRecord(t *testing.T) ProjectionRecord {
	t.Helper()
	r, e := NewProjectionRecord(ProjectionRecord{Revision: strings.Repeat("a", 40), ModelDigest: testDigest("model"), PlanDigest: testDigest("plan"), InputSnapshotDigest: testDigest("inputs"), RequestDigest: testDigest("request"), ProjectionID: "foundation/v1:ProjectionPolicy:docs", Module: ModuleIdentity{Name: "foundation", Version: "1.0.0", Digest: testDigest("module")}, Projector: ProjectorIdentity{ID: "markdown", Version: "2.1.0"}, ScopeIDs: []string{"software/v1:UseCase:orders/create"}, PolicyIDs: []string{"foundation/v1:Rule:docs"}, Artifacts: []Artifact{{Path: "src/create.go", Digest: testDigest("src"), Mode: "100644", Change: ChangeModified}, {Path: "docs/create.md", Digest: testDigest("doc"), Mode: "100644", Change: ChangeCreated}}, State: StateMaterializedUnverified})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func testFacts(r ProjectionRecord) []ArtifactFact {
	f := make([]ArtifactFact, 0, len(r.Artifacts)+2)
	for _, a := range r.Artifacts {
		f = append(f, ArtifactFact{Path: a.Path, Role: RoleProjectionTarget, Digest: a.Digest, Mode: a.Mode})
	}
	return append(f, ArtifactFact{Path: "legacy/old.md", Role: RoleExcluded, Digest: testDigest("legacy"), Mode: "100644", Reason: "not migrated"}, ArtifactFact{Path: "scratch/tmp.bin", Role: RoleUnknown, Digest: testDigest("scratch"), Mode: "100644"})
}
func TestProjectionRecordCanonicalAppendPayloadAndTamperDetection(t *testing.T) {
	r := testRecord(t)
	if r.Artifacts[0].Path != "docs/create.md" {
		t.Fatalf("not sorted: %#v", r.Artifacts)
	}
	if e := ValidateProjectionRecord(r); e != nil {
		t.Fatal(e)
	}
	a, e := AppendPayload(r)
	if e != nil {
		t.Fatal(e)
	}
	b, e := AppendPayload(r)
	if e != nil {
		t.Fatal(e)
	}
	if string(a) != string(b) {
		t.Fatal("payload is nondeterministic")
	}
	var env struct {
		Record        ProjectionRecord `json:"record"`
		ContentDigest string           `json:"contentDigest"`
	}
	if e = json.Unmarshal(a, &env); e != nil {
		t.Fatal(e)
	}
	if env.ContentDigest != r.ID {
		t.Fatalf("digest=%q", env.ContentDigest)
	}
	r.Module.Version = "9.9.9"
	if e = ValidateProjectionRecord(r); e == nil {
		t.Fatal("tampering accepted")
	}
}
func TestProjectionRecordRejectsDuplicateAndUnsupportedFacts(t *testing.T) {
	base := ProjectionRecord{Revision: strings.Repeat("b", 40), ModelDigest: testDigest("model"), PlanDigest: testDigest("plan"), InputSnapshotDigest: testDigest("inputs"), RequestDigest: testDigest("request"), ProjectionID: "core/v1:Projection:render", Module: ModuleIdentity{Name: "mod", Version: "1", Digest: testDigest("mod")}, Projector: ProjectorIdentity{ID: "p", Version: "1"}, ScopeIDs: []string{"scope"}, State: StateMaterializedUnverified, Artifacts: []Artifact{{Path: "out/a.md", Digest: testDigest("a"), Mode: "100644", Change: ChangeCreated}}}
	cases := []struct {
		name string
		edit func(*ProjectionRecord)
	}{{"duplicate scope", func(r *ProjectionRecord) { r.ScopeIDs = []string{"scope", "scope"} }}, {"duplicate policy", func(r *ProjectionRecord) { r.PolicyIDs = []string{"p", "p"} }}, {"duplicate artifact", func(r *ProjectionRecord) { r.Artifacts = append(r.Artifacts, r.Artifacts[0]) }}, {"deletion", func(r *ProjectionRecord) { r.Artifacts[0].Change = "deleted" }}, {"traversal", func(r *ProjectionRecord) { r.Artifacts[0].Path = "../outside" }}, {"nonportable", func(r *ProjectionRecord) { r.Artifacts[0].Path = "C:\\outside" }}, {"missing projection", func(r *ProjectionRecord) { r.ProjectionID = "" }}, {"missing plan digest", func(r *ProjectionRecord) { r.PlanDigest = "" }}, {"missing input digest", func(r *ProjectionRecord) { r.InputSnapshotDigest = "" }}, {"missing request digest", func(r *ProjectionRecord) { r.RequestDigest = "" }}, {"reserved device path", func(r *ProjectionRecord) { r.Artifacts[0].Path = "CON.txt" }}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			r.ScopeIDs = append([]string(nil), base.ScopeIDs...)
			r.PolicyIDs = append([]string(nil), base.PolicyIDs...)
			r.Artifacts = append([]Artifact(nil), base.Artifacts...)
			tc.edit(&r)
			if _, e := NewProjectionRecord(r); e == nil {
				t.Fatal("invalid record accepted")
			}
		})
	}
}
func TestProjectionAndVerificationFreshnessBindExactInputs(t *testing.T) {
	r := testRecord(t)
	facts := testFacts(r)[:len(r.Artifacts)]
	fresh := ProjectionFreshness{Revision: r.Revision, ModelDigest: r.ModelDigest, PlanDigest: r.PlanDigest, InputSnapshotDigest: r.InputSnapshotDigest, RequestDigest: r.RequestDigest, Targets: facts}
	if e := ValidateProjectionFreshness(r, fresh); e != nil {
		t.Fatal(e)
	}
	staleRevision := fresh
	staleRevision.Revision = strings.Repeat("c", 40)
	if e := ValidateProjectionFreshness(r, staleRevision); e == nil {
		t.Fatal("stale revision accepted")
	}
	stalePlan := fresh
	stalePlan.PlanDigest = testDigest("other plan")
	if e := ValidateProjectionFreshness(r, stalePlan); e == nil {
		t.Fatal("different reviewed plan accepted")
	}
	staleInputs := fresh
	staleInputs.InputSnapshotDigest = testDigest("other inputs")
	if e := ValidateProjectionFreshness(r, staleInputs); e == nil {
		t.Fatal("different captured input accepted")
	}
	staleRequest := fresh
	staleRequest.RequestDigest = testDigest("other request")
	if e := ValidateProjectionFreshness(r, staleRequest); e == nil {
		t.Fatal("different projection request accepted")
	}
	changed := append([]ArtifactFact(nil), facts...)
	changed[0].Digest = testDigest("changed")
	staleTarget := fresh
	staleTarget.Targets = changed
	if e := ValidateProjectionFreshness(r, staleTarget); e == nil {
		t.Fatal("changed target accepted")
	}
	v := VerifierIdentity{ID: "project-checks", Version: "3", Digest: testDigest("verifier")}
	checks := []CheckIdentity{{ID: "unit", Version: "2", Digest: testDigest("unit")}, {ID: "architecture", Version: "1", Digest: testDigest("arch")}}
	result, e := NewVerificationResult(VerificationResult{RecordID: r.ID, Revision: r.Revision, ModelDigest: r.ModelDigest, TargetSnapshotDigest: r.TargetSnapshotDigest, Verifier: v, Checks: []CheckResult{{ID: "unit", Version: "2", Digest: testDigest("unit"), Outcome: CheckPassed}, {ID: "architecture", Version: "1", Digest: testDigest("arch"), Outcome: CheckPassed}}, Outcome: OutcomePassed})
	if e != nil {
		t.Fatal(e)
	}
	f := Freshness{Revision: r.Revision, ModelDigest: r.ModelDigest, RecordID: r.ID, TargetSnapshotDigest: r.TargetSnapshotDigest, Verifier: v, Checks: checks}
	if e = ValidateVerificationFreshness(result, r, f); e != nil {
		t.Fatal(e)
	}
	if _, e = VerificationAppendPayload(result); e != nil {
		t.Fatal(e)
	}
	missing := f
	missing.Checks = checks[:1]
	if e = ValidateVerificationFreshness(result, r, missing); e == nil {
		t.Fatal("missing check accepted")
	}
	extra := f
	extra.Checks = append(append([]CheckIdentity(nil), checks...), CheckIdentity{ID: "extra", Version: "1", Digest: testDigest("extra")})
	if e = ValidateVerificationFreshness(result, r, extra); e == nil {
		t.Fatal("undeclared check accepted")
	}
	stale := f
	stale.TargetSnapshotDigest = testDigest("old")
	if e = ValidateVerificationFreshness(result, r, stale); e == nil {
		t.Fatal("stale snapshot accepted")
	}
}
func TestPassedVerificationRequiresCompleteDigestsAndPassingChecks(t *testing.T) {
	r := testRecord(t)
	v := VerifierIdentity{ID: "verify", Version: "1", Digest: testDigest("verifier")}
	base := VerificationResult{RecordID: r.ID, Revision: r.Revision, ModelDigest: r.ModelDigest, TargetSnapshotDigest: r.TargetSnapshotDigest, Verifier: v, Outcome: OutcomePassed}
	if _, e := NewVerificationResult(base); e == nil {
		t.Fatal("empty pass accepted")
	}
	base.Checks = []CheckResult{{ID: "check", Version: "1", Digest: "", Outcome: CheckPassed}}
	if _, e := NewVerificationResult(base); e == nil {
		t.Fatal("empty check digest accepted")
	}
	base.Verifier.Digest = ""
	base.Checks[0].Digest = testDigest("check")
	if _, e := NewVerificationResult(base); e == nil {
		t.Fatal("empty verifier digest accepted")
	}
	base.Verifier.Digest = testDigest("verifier")
	base.Checks[0].Outcome = CheckIncomplete
	if _, e := NewVerificationResult(base); e == nil {
		t.Fatal("incomplete check accepted")
	}
}
func TestOwnershipIndexBidirectionalAndVisibleFacts(t *testing.T) {
	r := testRecord(t)
	idx, e := BuildOwnershipIndex([]ProjectionRecord{r}, testFacts(r))
	if e != nil {
		t.Fatal(e)
	}
	if got := idx.ByScope[r.ScopeIDs[0]]; len(got) != 2 || got[0] != "docs/create.md" || got[1] != "src/create.go" {
		t.Fatalf("scope=%v", got)
	}
	if got := idx.Artifacts["docs/create.md"]; got.Status != OwnershipManaged || len(got.OwnerRecordIDs) != 1 {
		t.Fatalf("managed=%+v", got)
	}
	if got := idx.Artifacts["scratch/tmp.bin"]; got.Status != OwnershipUnknown {
		t.Fatalf("unknown=%+v", got)
	}
	if got := idx.Artifacts["legacy/old.md"]; got.Status != OwnershipExcluded || got.Reason != "not migrated" {
		t.Fatalf("excluded=%+v", got)
	}
	facts := testFacts(r)
	facts[0].Digest = testDigest("drift")
	drift, e := BuildOwnershipIndex([]ProjectionRecord{r}, facts)
	if e != nil {
		t.Fatal(e)
	}
	if drift.Artifacts["docs/create.md"].Status != OwnershipDrift {
		t.Fatal("drift was not detected")
	}
}
func TestOwnershipIndexCollisionAndIncompleteInventory(t *testing.T) {
	a := testRecord(t)
	b, e := NewProjectionRecord(ProjectionRecord{Revision: a.Revision, ModelDigest: a.ModelDigest, PlanDigest: a.PlanDigest, InputSnapshotDigest: a.InputSnapshotDigest, RequestDigest: a.RequestDigest, ProjectionID: "core/v1:Projection:other", Module: a.Module, Projector: ProjectorIdentity{ID: "other", Version: "1"}, ScopeIDs: []string{"other-scope"}, Artifacts: []Artifact{{Path: a.Artifacts[0].Path, Digest: a.Artifacts[0].Digest, Mode: a.Artifacts[0].Mode, Change: ChangeRetained}}, State: StateMaterializedUnverified})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = BuildOwnershipIndex([]ProjectionRecord{a, b}, nil); e == nil {
		t.Fatal("owner collision accepted")
	}
	idx, e := BuildOwnershipIndex([]ProjectionRecord{a}, nil)
	if e != nil {
		t.Fatal(e)
	}
	if idx.Artifacts["docs/create.md"].Status != OwnershipUnobserved {
		t.Fatal("missing observation was treated as complete")
	}
}

func TestRecordReferencesRejectDanglingPriorRecord(t *testing.T) {
	first := testRecord(t)
	second, err := NewProjectionRecord(ProjectionRecord{
		Revision: first.Revision, ModelDigest: first.ModelDigest, PlanDigest: first.PlanDigest, InputSnapshotDigest: first.InputSnapshotDigest, RequestDigest: first.RequestDigest, ProjectionID: first.ProjectionID,
		Module: first.Module, Projector: first.Projector, ScopeIDs: first.ScopeIDs, PolicyIDs: first.PolicyIDs,
		Artifacts:     []Artifact{{Path: first.Artifacts[0].Path, Digest: first.Artifacts[0].Digest, Mode: first.Artifacts[0].Mode, Change: ChangeRetained}},
		PriorRecordID: first.ID, State: StateMaterializedUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateRecordReferences([]ProjectionRecord{first, second}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRecordReferences([]ProjectionRecord{second}); err == nil {
		t.Fatal("dangling predecessor accepted")
	}
	foreign, err := NewProjectionRecord(ProjectionRecord{
		Revision: first.Revision, ModelDigest: first.ModelDigest, PlanDigest: first.PlanDigest,
		InputSnapshotDigest: first.InputSnapshotDigest, RequestDigest: first.RequestDigest,
		ProjectionID: "core/v1:Projection:other", Module: first.Module, Projector: first.Projector,
		ScopeIDs: first.ScopeIDs, PolicyIDs: first.PolicyIDs,
		Artifacts:     []Artifact{{Path: first.Artifacts[0].Path, Digest: first.Artifacts[0].Digest, Mode: first.Artifacts[0].Mode, Change: ChangeRetained}},
		PriorRecordID: first.ID, State: StateMaterializedUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateRecordReferences([]ProjectionRecord{first, foreign}); err == nil {
		t.Fatal("predecessor from another Projection accepted")
	}
}

func TestOperationalDigestsNormalizeEmptyCollectionsAcrossWireFormats(t *testing.T) {
	record := ProjectionRecord{PolicyIDs: nil}
	before, err := projectionRecordDigest(record)
	if err != nil {
		t.Fatal(err)
	}
	record.PolicyIDs = []string{}
	after, err := projectionRecordDigest(record)
	if err != nil || before != after {
		t.Fatalf("empty policy set changed identity: %s %s %v", before, after, err)
	}
	result := VerificationResult{Checks: nil}
	before, err = verificationDigest(result)
	if err != nil {
		t.Fatal(err)
	}
	result.Checks = []CheckResult{}
	after, err = verificationDigest(result)
	if err != nil || before != after {
		t.Fatalf("empty check set changed identity: %s %s %v", before, after, err)
	}
}
