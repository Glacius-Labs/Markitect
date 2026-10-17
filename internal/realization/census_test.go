package realization_test

import (
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type censusFixture struct {
	compiled  model.Result
	candidate realization.Candidate
	plan      realization.Aggregate
	child     model.Identity
}

func assessment(id model.Identity, disposition realization.Disposition, paths ...snapshot.Path) realization.DefinitionAssessment {
	return realization.DefinitionAssessment{Definition: id, Disposition: disposition, Artifacts: paths, Reason: "Explains the proposed observable realization and its limits.", Evidence: []realization.EvidenceUse{{Ref: "host:" + id.Package + "/" + id.ID, Relevance: "The cited observation should discriminate the stated obligation; review must judge it."}}}
}

func censusSetup(t *testing.T) censusFixture {
	t.Helper()
	c := compiledFacts()
	child := model.Identity{Package: "lending/reminders", ID: "send-reminder"}
	c.Models[0].Packages = append(c.Models[0].Packages, model.Package{Name: child.Package})
	c.Models[0].Definitions = append(c.Models[0].Definitions, model.Definition{Identity: child, ModelID: "library-model"})
	a, b, d := owner(t, "lending"), owner(t, child.Package), owner(t, "library")
	f := facts(t, c, realization.Registry{}, snapshot.Tree{}, []realization.CreationArea{area(t, a, "out"), area(t, b, "out"), area(t, d, "out")})
	base, err := realization.NewCandidateBase(c, f)
	if err != nil {
		t.Fatal(err)
	}
	claims := reserve(t, f, a, "out/lending", realization.PathClaims{})
	claims = reserve(t, f, b, "out/reminders", claims)
	claims = reserve(t, f, d, "out/library", claims)
	ca := capture(t, base, a, claims, fileTree(t, "out/lending"), registry(t, registration(t, a, artifact(t, "out/lending", borrow, policy))))
	cb := capture(t, base, b, claims, fileTree(t, "out/reminders"), registry(t, registration(t, b, artifact(t, "out/reminders", child))))
	cd := capture(t, base, d, claims, fileTree(t, "out/library"), registry(t, registration(t, d, artifact(t, "out/library", callable))))
	plan := aggregate(t, base)
	for _, part := range []realization.Candidate{ca, cb, cd} {
		plan = replayed(t, plan, part)
	}
	return censusFixture{c, ca, plan, child}
}

func ownAssessments(t *testing.T) []realization.DefinitionAssessment {
	t.Helper()
	return []realization.DefinitionAssessment{assessment(borrow, realization.Realized, path(t, "out/lending")), assessment(policy, realization.Realized, path(t, "out/lending"))}
}

func censusCodes(c realization.DefinitionCensus) []string {
	var codes []string
	for _, f := range c.Findings() {
		codes = append(codes, f.Code)
	}
	return codes
}

func censusStatus(t *testing.T, c realization.DefinitionCensus, want realization.CensusStatus, codes ...string) {
	t.Helper()
	if c.Status() != want {
		t.Fatalf("census status %v want %v: %+v", c.Status(), want, c.Findings())
	}
	for _, code := range codes {
		if !slices.Contains(censusCodes(c), code) {
			t.Fatalf("missing %s: %+v", code, c.Findings())
		}
	}
}

func TestCensusSeparatesCandidateIntermediateCompositionAndWholeProject(t *testing.T) {
	f := censusSetup(t)
	rows := ownAssessments(t)
	local := realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows)
	censusStatus(t, local, realization.CensusStructurallyComplete)
	if !slices.Equal(local.Expected(), []model.Identity{borrow, policy}) || local.ProjectScope() {
		t.Fatal("local census included descendants or project")
	}
	if who, ok := local.Owner(); !ok || who != owner(t, "lending") {
		t.Fatal("candidate owner binding lost")
	}
	integrated := realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows)
	censusStatus(t, integrated, realization.CensusStructurallyComplete)
	if plan, ok := integrated.Aggregate(); !ok || !plan.Tree().Equal(f.plan.Tree()) || !integrated.Basis().Equal(f.plan.Base()) {
		t.Fatal("intermediate census lost exact integrated subject")
	}
	root := realization.CheckCompositionCensus(f.compiled, f.plan, realization.RootOwner(), nil, nil)
	censusStatus(t, root, realization.CensusStructurallyComplete)
	if len(root.Expected()) != 0 || root.ProjectScope() {
		t.Fatal("root-own became project census")
	}
	project := realization.CheckProjectCensus(f.compiled, f.plan, nil, nil)
	censusStatus(t, project, realization.CensusIncomplete, "census.missing_definition")
	if !project.ProjectScope() || len(project.Expected()) != 4 || len(project.Missing()) != 4 {
		t.Fatal("project census did not require all local Definitions")
	}
	all := append(rows, assessment(f.child, realization.Realized, path(t, "out/reminders")), assessment(callable, realization.Vocabulary))
	all[3].Uses = []model.Identity{borrow}
	project = realization.CheckProjectCensus(f.compiled, f.plan, nil, all)
	censusStatus(t, project, realization.CensusStructurallyComplete)
}

func TestCensusRejectsMissingDuplicateImportedAndUnresolvedRows(t *testing.T) {
	f := censusSetup(t)
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, f.candidate, nil, ownAssessments(t)[:1]), realization.CensusIncomplete, "census.missing_definition")
	rows := ownAssessments(t)
	rows = append(rows, rows[0])
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows), realization.CensusInvalid, "census.duplicate_definition")
	rows = ownAssessments(t)
	rows = append(rows, assessment(model.Identity{Package: "foreign", ID: "Vocabulary"}, realization.Vocabulary))
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows), realization.CensusInvalid, "census.unexpected_definition")
	rows = ownAssessments(t)
	rows = append(rows, assessment(f.child, realization.Vocabulary))
	censusStatus(t, realization.CheckCompositionCensus(f.compiled, f.plan, owner(t, "lending"), nil, rows), realization.CensusInvalid, "census.unexpected_definition")
	rows = ownAssessments(t)
	rows[0].Disposition = realization.Unresolved
	rows[0].Evidence = nil
	result := realization.CheckCandidateCensus(f.compiled, f.candidate, nil, rows)
	censusStatus(t, result, realization.CensusIncomplete, "census.unresolved")
	if result.Accounting().Status() != realization.Accounted || !slices.Equal(result.Unresolved(), []model.Identity{borrow}) {
		t.Fatal("Accounted masked unresolved local work")
	}
}

func TestCensusUsesCapturedClassificationsRatherThanRegistryPresence(t *testing.T) {
	f := censusSetup(t)
	for _, class := range []realization.FileClass{realization.IgnoredFile, realization.CanonicalInput, realization.MarkitectInput} {
		c := classified(t, "out/lending", class)
		result := realization.CheckCandidateCensus(f.compiled, f.candidate, []realization.Classification{c}, ownAssessments(t))
		censusStatus(t, result, realization.CensusInvalid, "census.invalid_artifact")
	}
	classes := []realization.Classification{classified(t, "out/lending", realization.IgnoredFile), classified(t, "out/lending", realization.CanonicalInput)}
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, f.candidate, classes, ownAssessments(t)), realization.CensusInvalid, "classification.duplicate")
	classes = []realization.Classification{{}}
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, f.candidate, classes, ownAssessments(t)), realization.CensusInvalid, "classification.invalid")
}

func TestNoChangeRequiresSameCensusAndDoesNotRequireGlobalAccounted(t *testing.T) {
	f := censusSetup(t)
	base := candidateBase(t, f.compiled, f.plan.Registry(), f.plan.Tree(), nil)
	noChange := capture(t, base, owner(t, "lending"), realization.PathClaims{}, base.Tree(), base.Registry())
	if len(noChange.Changes()) != 0 {
		t.Fatal("fixture was not no-change")
	}
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, noChange, nil, nil), realization.CensusIncomplete, "census.missing_definition")
	censusStatus(t, realization.CheckCandidateCensus(f.compiled, noChange, nil, ownAssessments(t)), realization.CensusStructurallyComplete)
	// An unrelated, unregistered file remains visible without blocking own scope.
	extra, _ := snapshot.NewEntry(path(t, "old/unexplained"), snapshot.Regular, []byte("old"))
	tree, err := snapshot.NewTree(append(f.plan.Tree().Entries(), extra))
	if err != nil {
		t.Fatal(err)
	}
	base = candidateBase(t, f.compiled, f.plan.Registry(), tree, nil)
	noChange = capture(t, base, owner(t, "lending"), realization.PathClaims{}, tree, base.Registry())
	result := realization.CheckCandidateCensus(f.compiled, noChange, nil, ownAssessments(t))
	censusStatus(t, result, realization.CensusStructurallyComplete)
	if result.Accounting().Status() == realization.Accounted {
		t.Fatal("unrelated accounting problem was erased")
	}
}
