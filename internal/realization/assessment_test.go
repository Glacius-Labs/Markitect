package realization_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestCensusChecksMinimumFormWithoutInferringDispositions(t *testing.T) {
	f := censusSetup(t)
	for _, tc := range []struct {
		name   string
		change func(*realization.DefinitionAssessment)
		status realization.CensusStatus
		code   string
	}{
		{"reason", func(r *realization.DefinitionAssessment) { r.Reason = " \t" }, realization.CensusIncomplete, "census.reason_required"},
		{"evidence", func(r *realization.DefinitionAssessment) { r.Evidence = nil }, realization.CensusIncomplete, "census.evidence_required"},
		{"handle", func(r *realization.DefinitionAssessment) { r.Evidence[0].Ref = " " }, realization.CensusIncomplete, "census.evidence_incomplete"},
		{"relevance", func(r *realization.DefinitionAssessment) { r.Evidence[0].Relevance = "\n" }, realization.CensusIncomplete, "census.evidence_incomplete"},
		{"disposition", func(r *realization.DefinitionAssessment) { r.Disposition = "" }, realization.CensusInvalid, "census.invalid_disposition"},
		{"artifacts", func(r *realization.DefinitionAssessment) { r.Artifacts = nil }, realization.CensusIncomplete, "census.realized_artifact_required"},
		{"duplicate artifact", func(r *realization.DefinitionAssessment) { r.Artifacts = append(r.Artifacts, r.Artifacts[0]) }, realization.CensusInvalid, "census.duplicate_artifact"},
		{"duplicate use", func(r *realization.DefinitionAssessment) { r.Uses = []model.Identity{policy, policy} }, realization.CensusInvalid, "census.duplicate_use"},
		{"imported use", func(r *realization.DefinitionAssessment) {
			r.Uses = []model.Identity{{Package: "foreign", ID: "Vocabulary"}}
		}, realization.CensusInvalid, "census.invalid_use"},
		{"unknown use", func(r *realization.DefinitionAssessment) {
			r.Uses = []model.Identity{{Package: "lending", ID: "absent"}}
		}, realization.CensusInvalid, "census.invalid_use"},
		{"duplicate evidence", func(r *realization.DefinitionAssessment) { r.Evidence = append(r.Evidence, r.Evidence[0]) }, realization.CensusInvalid, "census.duplicate_evidence"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := ownAssessments(t)
			tc.change(&rows[0])
			result := realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows)
			censusStatus(t, result, tc.status, tc.code)
		})
	}
}

func TestConstrainedAndVocabularyUsesRemainReviewableProposals(t *testing.T) {
	f := censusSetup(t)
	rows := ownAssessments(t)
	rows[1].Disposition = realization.Constrains
	rows[1].Artifacts = nil
	censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows), realization.CensusIncomplete, "census.constrained_realization_required")
	rows[1].Uses = []model.Identity{f.child}
	censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows), realization.CensusStructurallyComplete)
	rows[1].Uses = nil
	rows[1].Artifacts = []snapshot.Path{path(t, "out/reminders")}
	// The other owner's realization is a read-only citation, not a grant.
	censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows), realization.CensusStructurallyComplete)
	rows[1] = assessment(policy, realization.Vocabulary)
	rows[1].Uses = []model.Identity{borrow}
	censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows), realization.CensusStructurallyComplete)
	rows[0].Disposition = realization.Unresolved
	result := realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows)
	censusStatus(t, result, realization.CensusIncomplete, "census.unresolved")
	if !slices.Equal(result.Unresolved(), []model.Identity{borrow}) {
		t.Fatal("vocabulary erased unresolved use obligations")
	}
	// A no-use claim is still proposal text, not a graph analysis or a proof.
	rows = ownAssessments(t)
	rows[1] = assessment(policy, realization.Vocabulary)
	rows[1].Reason = "No concrete use is claimed here; reviewer must examine the cited model observation."
	rows[1].Evidence[0].Ref = "host:model-observation"
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows), realization.CensusStructurallyComplete)
	rows[1].Evidence = nil
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows), realization.CensusIncomplete, "census.evidence_required")
	// Unresolved may retain valid partial references without becoming complete.
	rows = ownAssessments(t)
	rows[0].Disposition = realization.Unresolved
	rows[0].Uses = []model.Identity{policy}
	result = realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows)
	censusStatus(t, result, realization.CensusIncomplete, "census.unresolved")
	if len(result.Assessments()[0].Artifacts) != 1 || len(result.Assessments()[0].Evidence) != 1 {
		t.Fatal("partial assessment was discarded")
	}
}

func TestArtifactCitationsNeedPresentValidUnconflictedDirectBasis(t *testing.T) {
	f := censusSetup(t)
	rows := ownAssessments(t)
	rows[0].Artifacts = []snapshot.Path{path(t, "out/reminders")}
	censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows), realization.CensusInvalid, "census.realized_basis_mismatch")
	rows[0].Artifacts = append(rows[0].Artifacts, path(t, "out/lending"))
	censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows), realization.CensusStructurallyComplete)
	for _, name := range []string{"absent", "unknown", ".markitect/state/current", ".markitect/registry/registry.yaml"} {
		rows = ownAssessments(t)
		rows[0].Artifacts = []snapshot.Path{path(t, name)}
		censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows), realization.CensusInvalid, "census.invalid_artifact")
	}
	protected, _ := realization.ProtectFile(path(t, "out/lending"), "Fixed protected input.")
	base := candidateBase(t, f.compiled, f.plan.Registry(), f.plan.Tree(), nil, protected)
	c := capture(t, base, owner(t, "lending"), realization.PathClaims{}, base.Tree(), base.Registry())
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, c, nil, ownAssessments(t)), realization.CensusInvalid, "census.invalid_artifact")
	for _, reg := range []realization.Registry{
		registry(t, registration(t, owner(t, "gone"), artifact(t, "out/lending", borrow, policy))),
		registry(t, registration(t, owner(t, "lending"), artifact(t, "out/lending", borrow, model.Identity{Package: "gone", ID: "missing"}))),
	} {
		base = candidateBase(t, f.compiled, reg, fileTree(t, "out/lending"), nil)
		c = capture(t, base, owner(t, "lending"), realization.PathClaims{}, base.Tree(), base.Registry())
		censusStatus(t, realization.CheckCandidateCensus(f.compiled, c, nil, ownAssessments(t)), realization.CensusInvalid, "census.invalid_artifact")
	}
	reg := registry(t, registration(t, owner(t, "lending"), artifact(t, "out/lending", borrow, policy)))
	base = candidateBase(t, f.compiled, reg, snapshot.Tree{}, nil)
	c = capture(t, base, owner(t, "lending"), realization.PathClaims{}, base.Tree(), base.Registry())
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, c, nil, ownAssessments(t)), realization.CensusInvalid, "census.invalid_artifact")
}

func TestCensusRejectsContradictoryClassificationsEvenForEmptyOwnScope(t *testing.T) {
	f := censusSetup(t)
	for _, classes := range [][]realization.Classification{
		{classified(t, "out/lending", realization.IgnoredFile), classified(t, "out/lending", realization.IgnoredFile)},
		{classified(t, "absent", realization.CanonicalInput)},
		{{}},
	} {
		censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, realization.RootOwner(), classes, nil), realization.CensusInvalid)
	}
}

func TestCensusBindsCompleteModelSubjectOwnerAndClassifications(t *testing.T) {
	f := censusSetup(t)
	for _, change := range []func(*model.Result){
		func(c *model.Result) { c.Models[0].Digest = "new digest" },
		func(c *model.Result) {
			c.Models[0].Packages = append(c.Models[0].Packages, model.Package{Name: "new-empty"})
		},
		func(c *model.Result) {
			c.Models[0].Definitions = append(c.Models[0].Definitions, model.Definition{Identity: model.Identity{Package: "lending", ID: "new"}, ModelID: "library-model"})
		},
	} {
		g := censusSetup(t)
		change(&g.compiled)
		result := realization.CheckCandidateCensus(g.compiled, f.candidate, nil, ownAssessments(t))
		censusStatus(t, result, realization.CensusInvalid)
		if result.Basis().Valid() {
			t.Fatal("mismatched model established a binding")
		}
	}
	for _, owner := range []realization.Owner{{}, owner(t, "foreign"), owner(t, "absent")} {
		censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner, nil, nil), realization.CensusInvalid, "census.invalid_owner")
	}
	for _, status := range []model.Status{model.NotCompleted, model.Failed} {
		g := censusSetup(t)
		g.compiled.Status = status
		result := realization.CheckCompositionCensus(g.compiled, g.plan, realization.RootOwner(), nil, nil)
		want := realization.CensusIncomplete
		if status == model.Failed {
			want = realization.CensusInvalid
		}
		censusStatus(t, result, want, "census.model_unavailable")
	}
	for _, result := range []realization.DefinitionCensus{
		{}, realization.CheckCandidateCensus(f.compiled, realization.Candidate{}, nil, nil),
		realization.CheckCompositionCensus(f.compiled, realization.Aggregate{}, realization.RootOwner(), nil, nil),
		realization.CheckProjectCensus(f.compiled, realization.Aggregate{}, nil, nil),
	} {
		censusStatus(t, result, realization.CensusIncomplete)
	}
	// Another actual tree creates a distinct bound report; raw evidence handles
	// are still only strings. Run must separately reject stale receipt bindings.
	before := realization.CheckCandidateCensus(f.compiled, f.candidate, nil, ownAssessments(t))
	base := candidateBase(t, f.compiled, f.candidate.Registry(), f.candidate.Tree(), nil)
	changed := capture(t, base, owner(t, "lending"), realization.PathClaims{}, contentTree(t, "out/lending", "changed bytes"), base.Registry())
	after := realization.CheckCandidateCensus(f.compiled, changed, nil, ownAssessments(t))
	censusStatus(t, after, realization.CensusStructurallyComplete)
	oldSubject, _ := before.Candidate()
	newSubject, _ := after.Candidate()
	if oldSubject.Tree().Equal(newSubject.Tree()) || !oldSubject.Tree().Equal(f.candidate.Tree()) {
		t.Fatal("report subject was aliased/rebound to another observation")
	}
}

func TestCensusCopiesNestedProposalAndClassificationFacts(t *testing.T) {
	f := censusSetup(t)
	extra, _ := snapshot.NewEntry(path(t, "README"), snapshot.Regular, []byte("readme"))
	tree, err := snapshot.NewTree(append(f.plan.Tree().Entries(), extra))
	if err != nil {
		t.Fatal(err)
	}
	base := candidateBase(t, f.compiled, f.plan.Registry(), tree, nil)
	c := capture(t, base, owner(t, "lending"), realization.PathClaims{}, tree, base.Registry())
	classes := []realization.Classification{classified(t, "README", realization.IgnoredFile)}
	rows := ownAssessments(t)
	rows[0].Uses = []model.Identity{policy}
	rows[0].Reason = " literal reason "
	rows[0].Evidence[0].Ref = " opaque host handle "
	result := realization.CheckCandidateCensus(f.compiled, c, classes, rows)
	censusStatus(t, result, realization.CensusStructurallyComplete)
	rows[0].Artifacts[0] = snapshot.Path{}
	rows[0].Uses[0] = model.Identity{}
	rows[0].Evidence[0] = realization.EvidenceUse{}
	rows[0].Reason = "mutated"
	classes[0] = realization.Classification{}
	f.compiled.Models[0].Definitions[0].Identity = model.Identity{}
	result.Assessments()[0].Artifacts[0] = snapshot.Path{}
	result.Assessments()[0].Uses[0] = model.Identity{}
	result.Assessments()[0].Evidence[0] = realization.EvidenceUse{}
	result.Classifications()[0] = realization.Classification{}
	result.Expected()[0] = model.Identity{}
	saved := result.Assessments()[0]
	if saved.Artifacts[0].String() != "out/lending" || saved.Uses[0] != policy || saved.Evidence[0].Ref != " opaque host handle " || saved.Reason != " literal reason " || result.Classifications()[0].Path().String() != "README" || !slices.Equal(result.Expected(), []model.Identity{borrow, policy}) {
		t.Fatal("mutable census input/output alias")
	}
	partial := realization.CheckCandidateCensus(censusSetup(t).compiled, c, nil, nil)
	partial.Missing()[0] = model.Identity{}
	partial.Findings()[0].Code = "mutated"
	if !slices.Equal(partial.Missing(), []model.Identity{borrow, policy}) || partial.Findings()[0].Code != "census.missing_definition" {
		t.Fatal("mutable incomplete report")
	}
	rows = ownAssessments(t)
	rows[0].Disposition = realization.Unresolved
	partial = realization.CheckCandidateCensus(censusSetup(t).compiled, c, nil, rows)
	partial.Unresolved()[0] = model.Identity{}
	if !slices.Equal(partial.Unresolved(), []model.Identity{borrow}) {
		t.Fatal("mutable unresolved report")
	}
}

func TestCensusFindingsAndInventoryDoNotDependOnProposalOrder(t *testing.T) {
	f := censusSetup(t)
	rows := ownAssessments(t)
	rows = append(rows, rows[0])
	rows[0].Uses = []model.Identity{{Package: "missing", ID: "use"}}
	a := realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows)
	slices.Reverse(rows)
	b := realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows)
	if !reflect.DeepEqual(a.Findings(), b.Findings()) || !slices.Equal(a.Expected(), b.Expected()) || !slices.Equal(a.Missing(), b.Missing()) {
		t.Fatal("diagnostics depended on proposal order")
	}
}

func TestUncitedOldRegistryClassificationConflictIsNotGlobalCensusGate(t *testing.T) {
	f := censusSetup(t)
	extra, err := snapshot.NewEntry(path(t, "old/input"), snapshot.Regular, []byte("old"))
	if err != nil {
		t.Fatal(err)
	}
	tree, err := snapshot.NewTree(append(f.plan.Tree().Entries(), extra))
	if err != nil {
		t.Fatal(err)
	}
	reg := registry(t, append(f.plan.Registry().Entries(), registration(t, owner(t, "lending"), artifact(t, "old/input", borrow)))...)
	base := candidateBase(t, f.compiled, reg, tree, nil)
	c := capture(t, base, owner(t, "lending"), realization.PathClaims{}, tree, reg)
	classes := []realization.Classification{classified(t, "old/input", realization.IgnoredFile)}
	result := realization.CheckCandidateCensus(f.compiled, c, classes, ownAssessments(t))
	censusStatus(t, result, realization.CensusStructurallyComplete)
	if !slices.Contains(findingCodes(result.Accounting()), "file.conflicting_classification") {
		t.Fatal("old conflict was hidden")
	}
	rows := ownAssessments(t)
	rows[1].Disposition = realization.Constrains
	rows[1].Artifacts = []snapshot.Path{path(t, "old/input")}
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, c, classes, rows), realization.CensusInvalid, "census.invalid_artifact")
}

func TestRootCandidateAndEmptyPackageAreNotWholeProjectCoverage(t *testing.T) {
	f := censusSetup(t)
	base := candidateBase(t, f.compiled, f.plan.Registry(), f.plan.Tree(), nil)
	root := capture(t, base, realization.RootOwner(), realization.PathClaims{}, base.Tree(), base.Registry())
	result := realization.CheckCandidateCensus(f.compiled, root, nil, nil)
	censusStatus(t, result, realization.CensusStructurallyComplete)
	if result.ProjectScope() || len(result.Expected()) != 0 {
		t.Fatal("root-own was treated as the project")
	}
	result = realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "empty"), nil, nil)
	censusStatus(t, result, realization.CensusStructurallyComplete)
	if result.ProjectScope() || len(result.Expected()) != 0 {
		t.Fatal("authoritative empty package was broadened")
	}
	partial := ownAssessments(t)
	censusStatus(t, realization.CheckProjectCensus(f.compiled, f.plan, nil, partial), realization.CensusIncomplete, "census.missing_definition")
	// Read-only cross-owner citation does not alter the registry's write owner.
	partial[1].Disposition = realization.Constrains
	partial[1].Artifacts = []snapshot.Path{path(t, "out/reminders")}
	censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, partial), realization.CensusStructurallyComplete)
	fixed := facts(t, f.compiled, f.plan.Registry(), f.plan.Tree(), nil)
	decision(t, realization.CheckOwnedPath(fixed, owner(t, "lending"), path(t, "out/reminders")), realization.PathRefused, "claim.not_owner")
}
