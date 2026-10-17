package realization_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func aggregate(t *testing.T, base realization.CandidateBase) realization.Aggregate {
	t.Helper()
	a, err := realization.NewAggregate(base)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func replayed(t *testing.T, a realization.Aggregate, c realization.Candidate) realization.Aggregate {
	t.Helper()
	r := realization.ReplayDelivery(a, c)
	next, ok := r.Aggregate()
	if !ok || r.Status() != realization.CandidateAdmissible || len(r.Findings()) != 0 {
		t.Fatalf("replay: %+v", r.Findings())
	}
	return next
}

func replayRefused(t *testing.T, a realization.Aggregate, c realization.Candidate, code string) {
	t.Helper()
	r := realization.ReplayDelivery(a, c)
	if r.Status() != realization.CandidateRefused || len(r.Findings()) != 1 || r.Findings()[0].Code != code {
		t.Fatalf("replay status %v findings %+v; want %s", r.Status(), r.Findings(), code)
	}
	if next, ok := r.Aggregate(); ok || next.Valid() {
		t.Fatal("refusal returned an aggregate")
	}
}

func TestAggregateStartsAtTheCapturedBaseAndCountsAcceptedDeliveries(t *testing.T) {
	who := owner(t, "empty")
	reg := registry(t, registration(t, who, artifact(t, "owned", borrow)))
	base := candidateBase(t, compiledFacts(), reg, fileTree(t, "owned"), nil)
	a := aggregate(t, base)
	if a.Version() != 0 || !a.Tree().Equal(base.Tree()) || !reflect.DeepEqual(a.Registry().Entries(), reg.Entries()) || !a.Base().Equal(base) || !a.CandidateBase().Equal(base) {
		t.Fatal("version zero is not the captured base")
	}
	noChange := capture(t, base, who, realization.PathClaims{}, fileTree(t, "owned"), reg)
	next := replayed(t, a, noChange)
	if next.Version() != 1 || !next.Tree().Equal(a.Tree()) || !next.Base().Equal(base) {
		t.Fatal("a no-change delivery did not advance the version alone")
	}
	if _, err := realization.NewAggregate(realization.CandidateBase{}); err == nil {
		t.Fatal("zero base accepted")
	}
}

func TestReplayComposesOwnedDeltasMadeOnEarlierVersions(t *testing.T) {
	a, b := owner(t, "empty"), owner(t, "lending")
	compiled := compiledFacts()
	f := facts(t, compiled, realization.Registry{}, snapshot.Tree{}, []realization.CreationArea{area(t, a, "out"), area(t, b, "out")})
	base, err := realization.NewCandidateBase(compiled, f)
	if err != nil {
		t.Fatal(err)
	}
	claims := reserve(t, f, a, "out/a.go", realization.PathClaims{})
	claims = reserve(t, f, b, "out/b.go", claims)
	ra := registration(t, a, artifact(t, "out/a.go", borrow))
	rb := registration(t, b, artifact(t, "out/b.go", policy))
	ca := capture(t, base, a, claims, fileTree(t, "out/a.go"), registry(t, ra))
	cb := capture(t, base, b, claims, fileTree(t, "out/b.go"), registry(t, rb))
	for _, order := range [][]realization.Candidate{{ca, cb}, {cb, ca}} {
		agg := aggregate(t, base)
		for _, c := range order {
			agg = replayed(t, agg, c)
		}
		if agg.Version() != 2 || !agg.Tree().Equal(fileTree(t, "out/a.go", "out/b.go")) || !reflect.DeepEqual(agg.Registry().Entries(), registry(t, ra, rb).Entries()) {
			t.Fatal("replay lost a delivery's tree or registry")
		}
	}
}

func TestReplayKeepsOtherOwnersChangesInsteadOfObservedOldFiles(t *testing.T) {
	a, b := owner(t, "empty"), owner(t, "lending")
	reg := registry(t, registration(t, a, artifact(t, "a", borrow)), registration(t, b, artifact(t, "b", policy)))
	base := candidateBase(t, compiledFacts(), reg, fileTree(t, "a", "b"), nil)
	changedA, _ := snapshot.NewEntry(path(t, "a"), snapshot.Regular, []byte("changed"))
	changedB, _ := snapshot.NewEntry(path(t, "b"), snapshot.Executable, []byte("b"))
	oldA, _ := base.Tree().Lookup(path(t, "a"))
	oldB, _ := base.Tree().Lookup(path(t, "b"))
	makeTree := func(entries ...snapshot.Entry) snapshot.Tree {
		tree, err := snapshot.NewTree(entries)
		if err != nil {
			t.Fatal(err)
		}
		return tree
	}
	ca := capture(t, base, a, realization.PathClaims{}, makeTree(changedA, oldB), reg)
	cb := capture(t, base, b, realization.PathClaims{}, makeTree(oldA, changedB), reg)
	for _, order := range [][]realization.Candidate{{ca, cb}, {cb, ca}} {
		agg := aggregate(t, base)
		for _, c := range order {
			agg = replayed(t, agg, c)
		}
		if !agg.Tree().Equal(makeTree(changedA, changedB)) {
			t.Fatal("replay wrote another owner's observed old file")
		}
	}
}

func TestReplayRefusesChangedContentRecordsAndOwnership(t *testing.T) {
	who, other := owner(t, "empty"), owner(t, "lending")
	reg := registry(t, registration(t, who, artifact(t, "owned", borrow)))
	base := candidateBase(t, compiledFacts(), reg, fileTree(t, "owned"), nil)
	edit := capture(t, base, who, realization.PathClaims{}, contentTree(t, "owned", "first"), reg)
	again := capture(t, base, who, realization.PathClaims{}, contentTree(t, "owned", "second"), reg)
	agg := replayed(t, aggregate(t, base), edit)
	replayRefused(t, agg, again, "replay.content_changed")

	record := capture(t, base, who, realization.PathClaims{}, fileTree(t, "owned"), registry(t, registration(t, who, artifact(t, "owned", policy))))
	recordAgain := capture(t, base, who, realization.PathClaims{}, fileTree(t, "owned"), registry(t, registration(t, who, artifact(t, "owned", borrow, policy))))
	replayRefused(t, replayed(t, aggregate(t, base), record), recordAgain, "replay.registry_changed")

	// A record transferred to another owner after capture (core rule D6) makes
	// the earlier owner's file change unowned.
	transferred := candidateBase(t, compiledFacts(), registry(t, registration(t, other, artifact(t, "owned", borrow))), fileTree(t, "owned"), nil)
	replayRefused(t, aggregate(t, transferred), edit, "replay.not_owned")
}

func TestReplayRefusesTreeConflictsBetweenDeliveries(t *testing.T) {
	who := owner(t, "empty")
	f := facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, []realization.CreationArea{area(t, who, "out")})
	base, err := realization.NewCandidateBase(compiledFacts(), f)
	if err != nil {
		t.Fatal(err)
	}
	makePart := func(name string) realization.Candidate {
		claims := reserve(t, f, who, name, realization.PathClaims{})
		return capture(t, base, who, claims, fileTree(t, name), registry(t, registration(t, who, artifact(t, name, borrow))))
	}
	file, child := makePart("out/file"), makePart("out/file/child")
	agg := replayed(t, aggregate(t, base), file)
	replayRefused(t, agg, file, "replay.registry_changed")
	replayRefused(t, agg, child, "replay.tree_collision")
}

func TestReplayRequiresTheSameInventoryAreasAndProtections(t *testing.T) {
	who := owner(t, "empty")
	reg := registry(t, registration(t, who, artifact(t, "owned", borrow)))
	files := fileTree(t, "owned")
	areas := []realization.CreationArea{area(t, who, "out")}
	p, _ := realization.ProtectFile(path(t, "protected"), "Fixed input.")
	base := candidateBase(t, compiledFacts(), reg, files, areas, p)
	part := capture(t, base, who, realization.PathClaims{}, contentTree(t, "owned", "changed"), reg)
	changed := compiledFacts()
	changed.Models[0].Digest = "changed"
	p2, _ := realization.ProtectFile(path(t, "protected"), "Changed policy reason.")
	for i, other := range []realization.CandidateBase{
		candidateBase(t, changed, reg, files, areas, p),
		candidateBase(t, compiledFacts(), reg, files, areas, p2),
		candidateBase(t, compiledFacts(), reg, files, []realization.CreationArea{area(t, who, "elsewhere")}, p),
	} {
		r := realization.ReplayDelivery(aggregate(t, other), part)
		if r.Status() != realization.CandidateRefused || r.Findings()[0].Code != "replay.base_mismatch" {
			t.Fatalf("drift %d: %+v", i, r.Findings())
		}
	}
	// Equivalent inventories are sets; caller ordering does not change the basis.
	reordered := compiledFacts()
	slices.Reverse(reordered.Models)
	for i := range reordered.Models {
		slices.Reverse(reordered.Models[i].Packages)
		slices.Reverse(reordered.Models[i].Definitions)
	}
	replayed(t, aggregate(t, candidateBase(t, reordered, reg, files, areas, p)), part)
}
