package realization_test

import (
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func candidateBase(t *testing.T, compiled model.Result, reg realization.Registry, tree snapshot.Tree, areas []realization.CreationArea, protections ...realization.Protection) realization.CandidateBase {
	t.Helper()
	b, err := realization.NewCandidateBase(compiled, facts(t, compiled, reg, tree, areas, protections...))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func contentTree(t *testing.T, name, content string) snapshot.Tree {
	t.Helper()
	e, err := snapshot.NewEntry(path(t, name), snapshot.Regular, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	tree, err := snapshot.NewTree([]snapshot.Entry{e})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func capture(t *testing.T, base realization.CandidateBase, who realization.Owner, claims realization.PathClaims, tree snapshot.Tree, reg realization.Registry) realization.Candidate {
	t.Helper()
	r := realization.CaptureCandidate(base, who, claims, tree, reg)
	c, ok := r.Candidate()
	if !ok || r.Status() != realization.CandidateAdmissible || len(r.Findings()) != 0 {
		t.Fatalf("capture: %+v", r.Findings())
	}
	return c
}

func captureRefused(t *testing.T, r realization.CandidateResult, code string) {
	t.Helper()
	if r.Status() != realization.CandidateRefused || len(r.Findings()) != 1 || r.Findings()[0].Code != code {
		t.Fatalf("capture status %v findings %+v; want %s", r.Status(), r.Findings(), code)
	}
	if c, ok := r.Candidate(); ok || c.Valid() {
		t.Fatal("refusal returned a partial candidate")
	}
}

func reserve(t *testing.T, f realization.PathFacts, who realization.Owner, name string, claims realization.PathClaims) realization.PathClaims {
	t.Helper()
	next, d := realization.ReserveNewPath(f, claims, request(t, name, name, who))
	decision(t, d, realization.PathAdmissible, "")
	return next
}

func TestCaptureAndComposeDisjointChildFilesAndRegistryRecords(t *testing.T) {
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
	if len(ca.TouchedPaths()) != 1 || len(ca.Changes()) != 1 || len(ca.RegistryChanges()) != 1 || len(ca.Claims()) != 1 {
		t.Fatal("file and record were not one touch within a part")
	}
	for _, parts := range [][]realization.Candidate{{ca, cb}, {cb, ca}} {
		r := realization.ComposeCandidates(base, parts)
		plan, ok := r.Plan()
		if !ok || r.Status() != realization.CandidateAdmissible {
			t.Fatalf("composition: %+v", r.Findings())
		}
		if !plan.Base().Equal(base) || !plan.Tree().Equal(fileTree(t, "out/a.go", "out/b.go")) || !reflect.DeepEqual(plan.Registry().Entries(), registry(t, ra, rb).Entries()) {
			t.Fatal("union lost actual tree/registry/base")
		}
		if len(plan.Changes()) != 2 || len(plan.RegistryChanges()) != 2 || len(plan.Parts()) != 2 || plan.Parts()[0].Owner() != a {
			t.Fatal("plan order or effects differ")
		}
	}
}

func TestChangedOrRestoredOrphanMustBeRejustifiedOrDeleted(t *testing.T) {
	who := owner(t, "empty")
	old := registry(t, registration(t, who, artifact(t, "old/file", model.Identity{Package: "gone", ID: "removed"})))
	repaired := registry(t, registration(t, who, artifact(t, "old/file", borrow)))
	files := fileTree(t, "old/file")
	base := candidateBase(t, compiledFacts(), old, files, nil)
	captureRefused(t, realization.CaptureCandidate(base, who, realization.PathClaims{}, contentTree(t, "old/file", "changed"), old), "candidate.dangling_basis")
	capture(t, base, who, realization.PathClaims{}, contentTree(t, "old/file", "changed"), repaired)
	c := capture(t, base, who, realization.PathClaims{}, snapshot.Tree{}, realization.Registry{})
	if len(c.Changes()) != 1 || c.Changes()[0].Kind() != snapshot.Delete {
		t.Fatal("orphan deletion lost")
	}
	c = capture(t, base, who, realization.PathClaims{}, files, repaired)
	if len(c.Changes()) != 0 || len(c.RegistryChanges()) != 1 {
		t.Fatal("registry-only rejustification not retained")
	}
	missing := candidateBase(t, compiledFacts(), old, snapshot.Tree{}, nil)
	captureRefused(t, realization.CaptureCandidate(missing, who, realization.PathClaims{}, files, old), "candidate.dangling_basis")
	capture(t, missing, who, realization.PathClaims{}, files, repaired)
	capture(t, missing, who, realization.PathClaims{}, snapshot.Tree{}, realization.Registry{})
	// Unchanged old problems do not prevent structural capture or prove completion.
	c = capture(t, base, who, realization.PathClaims{}, files, old)
	if len(c.Changes()) != 0 || realization.AccountFiles(compiledFacts(), c.Tree(), c.Registry(), nil).Status() == realization.Accounted {
		t.Fatal("no-change became accounting completion")
	}
}

func TestCaptureRejectsFullObservedScopeViolations(t *testing.T) {
	who := owner(t, "empty")
	for _, tc := range []struct {
		name       string
		reg        realization.Registry
		protection []realization.Protection
		code       string
	}{
		{"unknown", realization.Registry{}, nil, "candidate.intake_required"},
		{"foreign", registry(t, registration(t, owner(t, "lending"), artifact(t, "foreign", borrow))), nil, "candidate.not_owner"},
		{"model", registry(t, registration(t, who, artifact(t, "model", borrow))), func() []realization.Protection {
			p, _ := realization.ProtectFile(path(t, "model"), "Captured model.")
			return []realization.Protection{p}
		}(), "claim.protected"},
		{".markitect/registry/registry.yaml", realization.Registry{}, nil, "candidate.registry_storage_changed"},
		{".markitect/state/unexpected", realization.Registry{}, nil, "claim.protected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := candidateBase(t, compiledFacts(), tc.reg, fileTree(t, tc.name), nil, tc.protection...)
			captureRefused(t, realization.CaptureCandidate(base, who, realization.PathClaims{}, contentTree(t, tc.name, "changed"), tc.reg), tc.code)
		})
	}
	storageBase := candidateBase(t, compiledFacts(), realization.Registry{}, fileTree(t, ".markitect/registry/registry.yaml"), nil)
	r := realization.CaptureCandidate(storageBase, owner(t, "foreign"), realization.PathClaims{}, snapshot.Tree{}, realization.Registry{})
	captureRefused(t, r, "candidate.registry_storage_changed")
	r.Findings()[0].Code = "mutated"
	if r.Findings()[0].Code != "candidate.registry_storage_changed" {
		t.Fatal("mutable candidate finding")
	}
	f := facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, []realization.CreationArea{area(t, who, "out")})
	base, err := realization.NewCandidateBase(compiledFacts(), f)
	if err != nil {
		t.Fatal(err)
	}
	reg := registry(t, registration(t, who, artifact(t, "out/new", borrow)))
	captureRefused(t, realization.CaptureCandidate(base, who, realization.PathClaims{}, fileTree(t, "out/new"), reg), "candidate.claim_required")
	claims := reserve(t, f, who, "out/new", realization.PathClaims{})
	doc, err := realization.NewArtifact(path(t, "out/new"), realization.Doc, []model.Identity{borrow}, "Documentation.")
	if err != nil {
		t.Fatal(err)
	}
	captureRefused(t, realization.CaptureCandidate(base, who, claims, fileTree(t, "out/new"), registry(t, registration(t, who, doc))), "candidate.claim_form_mismatch")
}

func TestComposeRefusesSharedOrPrefixTouchesEvenWithIdenticalBytes(t *testing.T) {
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
	a, b := makePart("out/file"), makePart("out/file/child")
	for _, parts := range [][]realization.Candidate{{a, a}, {a, b}, {b, a}} {
		r := realization.ComposeCandidates(base, parts)
		if r.Status() != realization.CandidateRefused || len(r.Findings()) != 1 || r.Findings()[0].Code != "composition.overlap" {
			t.Fatalf("composition: %+v", r.Findings())
		}
		if p, ok := r.Plan(); ok || p.Valid() {
			t.Fatal("partial conflicting plan")
		}
	}
	// A registry-only edit and a byte edit conflict across parts as well.
	old := registry(t, registration(t, who, artifact(t, "owned", borrow)))
	base = candidateBase(t, compiledFacts(), old, fileTree(t, "owned"), nil)
	regPart := capture(t, base, who, realization.PathClaims{}, fileTree(t, "owned"), registry(t, registration(t, who, artifact(t, "owned", policy))))
	filePart := capture(t, base, who, realization.PathClaims{}, contentTree(t, "owned", "changed"), old)
	r := realization.ComposeCandidates(base, []realization.Candidate{regPart, filePart})
	if r.Status() != realization.CandidateRefused || r.Findings()[0].Code != "composition.overlap" {
		t.Fatalf("metadata overlap: %+v", r.Findings())
	}
}

func TestCompositionUsesDeltasInsteadOfOverwritingOtherPartsWithObservedOldFiles(t *testing.T) {
	a, b := owner(t, "empty"), owner(t, "lending")
	reg := registry(t, registration(t, a, artifact(t, "a", borrow)), registration(t, b, artifact(t, "b", policy)))
	base := candidateBase(t, compiledFacts(), reg, fileTree(t, "a", "b"), nil)
	changedA, err := snapshot.NewEntry(path(t, "a"), snapshot.Regular, []byte("changed"))
	if err != nil {
		t.Fatal(err)
	}
	changedB, err := snapshot.NewEntry(path(t, "b"), snapshot.Executable, []byte("b"))
	if err != nil {
		t.Fatal(err)
	}
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
	if len(cb.Changes()) != 1 || cb.Changes()[0].Kind() != snapshot.Modify {
		t.Fatal("mode-only change was not observed")
	}
	before, _ := cb.Changes()[0].Before()
	after, _ := cb.Changes()[0].After()
	if before.Mode() != snapshot.Regular || after.Mode() != snapshot.Executable {
		t.Fatal("mode evidence lost")
	}
	for _, parts := range [][]realization.Candidate{{ca, cb}, {cb, ca}} {
		result := realization.ComposeCandidates(base, parts)
		plan, ok := result.Plan()
		if !ok || !plan.Tree().Equal(makeTree(changedA, changedB)) {
			t.Fatalf("composed old observations over another owner's change: %+v", result.Findings())
		}
	}
}
