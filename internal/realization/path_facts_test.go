package realization_test

import (
	"errors"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestPathSelectorsRequireExplicitValidValues(t *testing.T) {
	who := owner(t, "empty")
	p := path(t, "a")
	if _, err := realization.NewCreationArea(realization.Owner{}, snapshot.RootDirectory()); !errors.Is(err, realization.ErrInvalidClaimFacts) {
		t.Fatalf("zero owner: %v", err)
	}
	if _, err := realization.NewCreationArea(who, snapshot.DirectoryBoundary{}); !errors.Is(err, realization.ErrInvalidClaimFacts) {
		t.Fatalf("implicit root: %v", err)
	}
	for _, tc := range []struct {
		path snapshot.Path
		why  string
	}{{snapshot.Path{}, "Reason"}, {p, " \t"}} {
		if _, err := realization.ProtectFile(tc.path, tc.why); !errors.Is(err, realization.ErrInvalidClaimFacts) {
			t.Fatalf("invalid file protection: %v", err)
		}
	}
	for _, tc := range []struct {
		directory snapshot.DirectoryBoundary
		why       string
	}{{snapshot.DirectoryBoundary{}, "Reason"}, {snapshot.RootDirectory(), "\n"}} {
		if _, err := realization.ProtectDirectory(tc.directory, tc.why); !errors.Is(err, realization.ErrInvalidClaimFacts) {
			t.Fatalf("invalid directory protection: %v", err)
		}
	}
	for _, tc := range []struct {
		id   realization.ClaimID
		who  realization.Owner
		path snapshot.Path
		form realization.Form
	}{
		{"", who, p, realization.Code}, {" \t", who, p, realization.Code}, {"id", realization.Owner{}, p, realization.Code}, {"id", who, snapshot.Path{}, realization.Code}, {"id", who, p, "unknown"},
	} {
		if _, err := realization.NewPathRequest(tc.id, tc.who, tc.path, tc.form); !errors.Is(err, realization.ErrInvalidPathClaim) {
			t.Fatalf("invalid request: %v", err)
		}
	}
	for _, form := range []realization.Form{realization.Code, realization.Test, realization.Doc, realization.Build, realization.Config} {
		r, err := realization.NewPathRequest(" exact opaque ID ", who, p, form)
		if err != nil || r.ID() != " exact opaque ID " || r.Form() != form {
			t.Fatalf("opaque identity/form changed: %v", err)
		}
	}
	a := area(t, who, ".")
	if a.Owner() != who || !a.Directory().IsRoot() {
		t.Fatal("selection facts changed")
	}
	file, _ := realization.ProtectFile(p, " reason ")
	if got, ok := file.File(); !ok || got != p || file.Why() != " reason " {
		t.Fatal("file protection changed")
	}
	if _, ok := file.Directory(); ok {
		t.Fatal("file became directory")
	}
	dir, _ := realization.ProtectDirectory(snapshot.RootDirectory(), "all")
	if _, ok := dir.File(); ok {
		t.Fatal("directory became file")
	}
	if d, ok := dir.Directory(); !ok || !d.IsRoot() {
		t.Fatal("directory facts changed")
	}
}

func TestPathFactsRefuseContradictoryOrZeroSelections(t *testing.T) {
	who := owner(t, "empty")
	a := area(t, who, ".")
	for _, tc := range []struct {
		areas       []realization.CreationArea
		protections []realization.Protection
	}{
		{areas: []realization.CreationArea{{}}}, {areas: []realization.CreationArea{a, a}}, {protections: []realization.Protection{{}}},
	} {
		if _, err := realization.NewPathFacts(compiledFacts(), realization.Registry{}, snapshot.Tree{}, tc.areas, tc.protections); !errors.Is(err, realization.ErrInvalidClaimFacts) {
			t.Fatalf("invalid selection facts: %v", err)
		}
	}
	compiled := compiledFacts()
	compiled.Models[0].Packages = append(compiled.Models[0].Packages, model.Package{Name: "empty"})
	if _, err := realization.NewPathFacts(compiled, realization.Registry{}, snapshot.Tree{}, nil, nil); !errors.Is(err, realization.ErrInvalidClaimFacts) {
		t.Fatalf("contradictory inventory: %v", err)
	}
}

func TestReplayBindsTreeContentAndRegistryBasisWithoutMutableAliases(t *testing.T) {
	who := owner(t, "empty")
	areas := []realization.CreationArea{area(t, who, "out")}
	makeTree := func(mode snapshot.Mode, content string) snapshot.Tree {
		e, err := snapshot.NewEntry(path(t, "existing"), mode, []byte(content))
		if err != nil {
			t.Fatal(err)
		}
		tree, err := snapshot.NewTree([]snapshot.Entry{e})
		if err != nil {
			t.Fatal(err)
		}
		return tree
	}
	reg := registry(t, registration(t, who, artifact(t, "existing", borrow)))
	baseline := facts(t, compiledFacts(), reg, makeTree(snapshot.Regular, "before"), areas)
	r := request(t, "id", "out/new", who)
	claims, d := realization.ReserveNewPath(baseline, realization.PathClaims{}, r)
	decision(t, d, realization.PathAdmissible, "")
	for _, changed := range []realization.PathFacts{
		facts(t, compiledFacts(), reg, makeTree(snapshot.Regular, "after"), areas),
		facts(t, compiledFacts(), reg, makeTree(snapshot.Executable, "before"), areas),
		facts(t, compiledFacts(), registry(t, registration(t, who, artifact(t, "existing", policy))), makeTree(snapshot.Regular, "before"), areas),
		facts(t, compiledFacts(), registry(t, registration(t, realization.RootOwner(), artifact(t, "existing", borrow))), makeTree(snapshot.Regular, "before"), areas),
	} {
		_, d := realization.ReserveNewPath(changed, claims, r)
		decision(t, d, realization.PathRefused, "claim.basis_changed")
	}
	// New unregistered files cannot be explained retroactively by an old reservation.
	f := facts(t, compiledFacts(), reg, fileTree(t, "existing", "out/new"), areas)
	_, d = realization.ReserveNewPath(f, claims, r)
	decision(t, d, realization.PathRefused, "claim.basis_changed")
	_, d = realization.ReserveNewPath(f, realization.PathClaims{}, r)
	decision(t, d, realization.PathRefused, "claim.intake_required")
	// A removed local owner also fails even when the old key and request match.
	c := compiledFacts()
	c.Models[0].Packages = append(c.Models[0].Packages[:2], c.Models[0].Packages[3:]...)
	_, d = realization.ReserveNewPath(facts(t, c, reg, makeTree(snapshot.Regular, "before"), areas), claims, r)
	decision(t, d, realization.PathRefused, "claim.unknown_owner")
}
