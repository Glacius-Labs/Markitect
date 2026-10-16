package realization_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestCompositionBindsTheEntireBaseIncludingNoChangeParts(t *testing.T) {
	who := owner(t, "empty")
	compiled := compiledFacts()
	reg := registry(t, registration(t, who, artifact(t, "owned", borrow)))
	files := fileTree(t, "owned")
	areas := []realization.CreationArea{area(t, who, "out")}
	p, _ := realization.ProtectFile(path(t, "protected"), "Fixed input.")
	f := facts(t, compiled, reg, files, areas, p)
	base, err := realization.NewCandidateBase(compiled, f)
	if err != nil {
		t.Fatal(err)
	}
	part := capture(t, base, who, realization.PathClaims{}, files, reg)
	var variants []realization.CandidateBase
	variants = append(variants, candidateBase(t, compiled, reg, contentTree(t, "owned", "different"), areas, p))
	e, err := snapshot.NewEntry(path(t, "owned"), snapshot.Executable, []byte("owned"))
	if err != nil {
		t.Fatal(err)
	}
	modeTree, err := snapshot.NewTree([]snapshot.Entry{e})
	if err != nil {
		t.Fatal(err)
	}
	variants = append(variants, candidateBase(t, compiled, reg, modeTree, areas, p))
	variants = append(variants, candidateBase(t, compiled, registry(t, registration(t, who, artifact(t, "owned", policy))), files, areas, p))
	variants = append(variants, candidateBase(t, compiled, registry(t, registration(t, realization.RootOwner(), artifact(t, "owned", borrow))), files, areas, p))
	changed := compiledFacts()
	changed.Models[0].Digest = "changed"
	variants = append(variants, candidateBase(t, changed, reg, files, areas, p))
	changed = compiledFacts()
	changed.Models[0].Packages = append(changed.Models[0].Packages, model.Package{Name: "new-empty"})
	variants = append(variants, candidateBase(t, changed, reg, files, areas, p))
	changed = compiledFacts()
	changed.Models[0].Definitions = append(changed.Models[0].Definitions, model.Definition{Identity: model.Identity{Package: "empty", ID: "new"}, ModelID: "library-model"})
	variants = append(variants, candidateBase(t, changed, reg, files, areas, p))
	p2, _ := realization.ProtectFile(path(t, "protected"), "Changed policy reason.")
	variants = append(variants, candidateBase(t, compiled, reg, files, areas, p2))
	variants = append(variants, candidateBase(t, compiled, reg, files, []realization.CreationArea{area(t, who, "elsewhere")}, p))
	for i, b := range variants {
		if base.Equal(b) {
			t.Fatalf("base drift %d ignored", i)
		}
		for _, parts := range [][]realization.Candidate{{part}, {part, part}} {
			r := realization.ComposeCandidates(b, parts)
			if r.Status() != realization.CandidateRefused || len(r.Findings()) != 1 || r.Findings()[0].Code != "composition.base_mismatch" {
				t.Fatalf("base drift %d: %+v", i, r.Findings())
			}
			if plan, ok := r.Plan(); ok || plan.Valid() {
				t.Fatal("mismatched base returned a plan")
			}
		}
	}
	// Equivalent inventories are sets; caller ordering does not change the base.
	reordered := compiledFacts()
	slices.Reverse(reordered.Models)
	for i := range reordered.Models {
		slices.Reverse(reordered.Models[i].Packages)
		slices.Reverse(reordered.Models[i].Definitions)
	}
	equivalent := candidateBase(t, reordered, reg, files, areas, p)
	if !base.Equal(equivalent) {
		t.Fatal("equivalent reordered facts differ")
	}
	r := realization.ComposeCandidates(equivalent, []realization.Candidate{part})
	if _, ok := r.Plan(); !ok {
		t.Fatalf("equivalent base refused: %+v", r.Findings())
	}
}

func TestCandidateBaseRequiresCompleteMatchingModelAndCopiesInventory(t *testing.T) {
	who := owner(t, "empty")
	compiled := compiledFacts()
	reg := registry(t, registration(t, who, artifact(t, "owned", borrow)))
	files := fileTree(t, "owned")
	f := facts(t, compiled, reg, files, nil)
	base, err := realization.NewCandidateBase(compiled, f)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		change func(*model.Result)
		want   error
	}{
		{func(c *model.Result) { c.Status = model.NotCompleted }, realization.ErrIncompleteCandidateBase},
		{func(c *model.Result) { c.Status = model.Failed }, realization.ErrInvalidCandidateBase},
		{func(c *model.Result) { c.Models[0].Digest = "different" }, realization.ErrInvalidCandidateBase},
		{func(c *model.Result) {
			c.Models[0].Packages = append(c.Models[0].Packages, model.Package{Name: "extra"})
		}, realization.ErrInvalidCandidateBase},
	} {
		c := compiledFacts()
		tc.change(&c)
		invalid, err := realization.NewCandidateBase(c, f)
		if !errors.Is(err, tc.want) || invalid.Valid() {
			t.Fatalf("invalid base: %v", err)
		}
	}
	if b, err := realization.NewCandidateBase(compiled, realization.PathFacts{}); !errors.Is(err, realization.ErrIncompleteCandidateBase) || b.Valid() {
		t.Fatalf("zero facts: %v", err)
	}
	compiled.Models[0].Definitions[0].Identity = model.Identity{}
	compiled.Models[0].Packages[2].Name = "mutated"
	if !base.Equal(candidateBase(t, compiledFacts(), reg, files, nil)) {
		t.Fatal("input inventory aliases the fixed base")
	}
	capture(t, base, who, realization.PathClaims{}, contentTree(t, "owned", "updated"), reg)
}

func TestNewFileCannotUseClaimFromAnotherBasis(t *testing.T) {
	who := owner(t, "empty")
	old := registry(t, registration(t, who, artifact(t, "owned", borrow)))
	areas := []realization.CreationArea{area(t, who, "out")}
	f := facts(t, compiledFacts(), old, fileTree(t, "owned"), areas)
	claims := reserve(t, f, who, "out/new", realization.PathClaims{})
	newBase := candidateBase(t, compiledFacts(), old, contentTree(t, "owned", "changed base"), areas)
	entries := newBase.Tree().Entries()
	e, err := snapshot.NewEntry(path(t, "out/new"), snapshot.Regular, []byte("new"))
	if err != nil {
		t.Fatal(err)
	}
	observed, err := snapshot.NewTree(append(entries, e))
	if err != nil {
		t.Fatal(err)
	}
	proposed := registry(t, append(old.Entries(), registration(t, who, artifact(t, "out/new", borrow)))...)
	captureRefused(t, realization.CaptureCandidate(newBase, who, claims, observed, proposed), "claim.basis_changed")
	protection, _ := realization.ProtectFile(path(t, "out/new"), "Protected future path.")
	protected := candidateBase(t, compiledFacts(), old, fileTree(t, "owned"), areas, protection)
	captureRefused(t, realization.CaptureCandidate(protected, who, claims, fileTree(t, "owned", "out/new"), proposed), "claim.protected")
}

func TestCaptureEnforcesFileRecordAndOwnerPostconditions(t *testing.T) {
	who := owner(t, "empty")
	old := registry(t, registration(t, who, artifact(t, "owned", borrow)))
	files := fileTree(t, "owned")
	base := candidateBase(t, compiledFacts(), old, files, nil)
	for _, tc := range []struct {
		tree snapshot.Tree
		reg  realization.Registry
		code string
	}{
		{snapshot.Tree{}, old, "candidate.missing_file"},
		{files, realization.Registry{}, "candidate.registration_required"},
		{files, registry(t, registration(t, owner(t, "lending"), artifact(t, "owned", borrow))), "candidate.owner_changed"},
		{files, registry(t, registration(t, who, artifact(t, "owned", model.Identity{Package: "gone", ID: "new-missing"}))), "candidate.dangling_basis"},
	} {
		captureRefused(t, realization.CaptureCandidate(base, who, realization.PathClaims{}, tc.tree, tc.reg), tc.code)
	}
	foreign := candidateBase(t, compiledFacts(), registry(t, registration(t, owner(t, "lending"), artifact(t, "owned", borrow))), files, nil)
	captureRefused(t, realization.CaptureCandidate(foreign, who, realization.PathClaims{}, files, old), "candidate.not_owner")
	unknown := candidateBase(t, compiledFacts(), realization.Registry{}, files, nil)
	captureRefused(t, realization.CaptureCandidate(unknown, who, realization.PathClaims{}, snapshot.Tree{}, realization.Registry{}), "candidate.intake_required")
	captureRefused(t, realization.CaptureCandidate(base, owner(t, "foreign"), realization.PathClaims{}, files, old), "candidate.invalid_owner")
	// The final union must reject prefix conflicts, even for a no-change request.
	colliding := candidateBase(t, compiledFacts(), old, fileTree(t, "owned/child"), nil)
	captureRefused(t, realization.CaptureCandidate(colliding, who, realization.PathClaims{}, colliding.Tree(), old), "candidate.file_directory_conflict")
	r := realization.ComposeCandidates(colliding, nil)
	if r.Status() != realization.CandidateRefused || r.Findings()[0].Code != "composition.file_directory_conflict" {
		t.Fatalf("result collision: %+v", r.Findings())
	}
	// Explicit cleanup repairs this old collision without acquiring its unknown child.
	capture(t, colliding, who, realization.PathClaims{}, colliding.Tree(), realization.Registry{})
}

func TestFixedCandidatesAndPlansDoNotExposeMutableAliases(t *testing.T) {
	who := owner(t, "empty")
	f := facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, []realization.CreationArea{area(t, who, "out")})
	base, err := realization.NewCandidateBase(compiledFacts(), f)
	if err != nil {
		t.Fatal(err)
	}
	claims := reserve(t, f, who, "out/new", realization.PathClaims{})
	reg := registry(t, registration(t, who, artifact(t, "out/new", borrow)))
	c := capture(t, base, who, claims, fileTree(t, "out/new"), reg)
	parts := []realization.Candidate{c}
	r := realization.ComposeCandidates(base, parts)
	plan, ok := r.Plan()
	if !ok {
		t.Fatal(r.Findings())
	}
	parts[0] = realization.Candidate{}
	c.Changes()[0] = snapshot.Change{}
	c.RegistryChanges()[0] = realization.RegistryChange{}
	c.TouchedPaths()[0] = snapshot.Path{}
	c.Claims()[0] = realization.PathClaim{}
	c.Registry().Entries()[0] = realization.Registration{}
	c.Registry().Entries()[0].Artifact().Basis()[0] = model.Identity{}
	c.Tree().Entries()[0].Bytes()[0] = 'x'
	plan.Parts()[0] = realization.Candidate{}
	plan.Changes()[0] = snapshot.Change{}
	plan.RegistryChanges()[0] = realization.RegistryChange{}
	if len(c.Changes()) != 1 || c.Changes()[0].Path().String() != "out/new" || c.RegistryChanges()[0].Path().String() != "out/new" || c.TouchedPaths()[0].String() != "out/new" || c.Claims()[0].Request().Path().String() != "out/new" {
		t.Fatal("mutable candidate output alias")
	}
	if !plan.Parts()[0].Valid() || plan.Changes()[0].Path().String() != "out/new" || plan.RegistryChanges()[0].Path().String() != "out/new" || !plan.Tree().Equal(c.Tree()) || !reflect.DeepEqual(plan.Registry().Entries(), c.Registry().Entries()) || !plan.Base().Equal(base) {
		t.Fatal("mutable plan output alias")
	}
	if !c.Tree().Equal(fileTree(t, "out/new")) || c.Registry().Entries()[0].Artifact().Basis()[0] != borrow {
		t.Fatal("mutable bytes or Definition links escaped through value getters")
	}
	change := c.RegistryChanges()[0]
	if _, ok := change.Before(); ok {
		t.Fatal("addition has a Before")
	}
	if after, ok := change.After(); !ok || after.Owner() != who {
		t.Fatal("addition lost its After")
	}
}

func TestUnavailableValuesReturnNoPartialResultsAndEmptyCompositionIsStructural(t *testing.T) {
	var c realization.Candidate
	var p realization.CompositionPlan
	if c.Valid() || c.Base().Valid() || p.Valid() || p.Base().Valid() || (realization.CandidateBase{}).Equal(realization.CandidateBase{}) {
		t.Fatal("zero values claim validity")
	}
	result := realization.CaptureCandidate(realization.CandidateBase{}, realization.RootOwner(), realization.PathClaims{}, snapshot.Tree{}, realization.Registry{})
	if result.Status() != realization.CandidateIncomplete {
		t.Fatal("missing base is not incomplete")
	}
	if c, ok := result.Candidate(); ok || c.Valid() {
		t.Fatal("missing observation produced candidate")
	}
	base := candidateBase(t, compiledFacts(), realization.Registry{}, fileTree(t, "unknown"), nil)
	for _, r := range []realization.CompositionResult{realization.ComposeCandidates(realization.CandidateBase{}, nil), realization.ComposeCandidates(base, []realization.Candidate{{}})} {
		if r.Status() != realization.CandidateIncomplete {
			t.Fatal("unavailable composition was admitted")
		}
		if p, ok := r.Plan(); ok || p.Valid() {
			t.Fatal("partial plan")
		}
	}
	r := realization.ComposeCandidates(base, nil)
	p, ok := r.Plan()
	if !ok || !p.Tree().Equal(base.Tree()) || len(p.Changes()) != 0 {
		t.Fatal("empty composition lost base")
	}
	if realization.AccountFiles(compiledFacts(), p.Tree(), p.Registry(), nil).Status() == realization.Accounted {
		t.Fatal("structural empty plan became completion")
	}
	refused := realization.ComposeCandidates(base, []realization.Candidate{capture(t, base, realization.RootOwner(), realization.PathClaims{}, base.Tree(), base.Registry()), capture(t, candidateBase(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, nil), realization.RootOwner(), realization.PathClaims{}, snapshot.Tree{}, realization.Registry{})})
	copy := refused.Findings()
	copy[0].Code = "mutated"
	if refused.Findings()[0].Code != "composition.base_mismatch" {
		t.Fatal("mutable composition finding")
	}
}
