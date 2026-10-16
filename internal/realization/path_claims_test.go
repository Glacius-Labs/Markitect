package realization_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func area(t *testing.T, who realization.Owner, name string) realization.CreationArea {
	t.Helper()
	d := snapshot.RootDirectory()
	if name != "." {
		var err error
		d, err = snapshot.DirectoryAt(path(t, name))
		if err != nil {
			t.Fatal(err)
		}
	}
	a, err := realization.NewCreationArea(who, d)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func request(t *testing.T, id, name string, who realization.Owner) realization.PathRequest {
	t.Helper()
	r, err := realization.NewPathRequest(realization.ClaimID(id), who, path(t, name), realization.Code)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func facts(t *testing.T, compiled model.Result, reg realization.Registry, tree snapshot.Tree, areas []realization.CreationArea, protections ...realization.Protection) realization.PathFacts {
	t.Helper()
	f, err := realization.NewPathFacts(compiled, reg, tree, areas, protections)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func decision(t *testing.T, d realization.PathDecision, status realization.PathStatus, code string) {
	t.Helper()
	if d.Status() != status {
		t.Fatalf("status = %v, want %v; findings %+v", d.Status(), status, d.Findings())
	}
	f := d.Findings()
	if code == "" {
		if len(f) != 0 {
			t.Fatalf("unexpected findings %+v", f)
		}
	} else if len(f) != 1 || f[0].Code != code {
		t.Fatalf("findings %+v, want %s", f, code)
	}
}

func TestNewFileClaimsUseOwnerSelectionsAndExactDirectoryBoundaries(t *testing.T) {
	who, other := owner(t, "empty"), owner(t, "lending")
	f := facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, []realization.CreationArea{area(t, who, "out"), area(t, other, "elsewhere")})
	var claims realization.PathClaims
	for i, tc := range []struct {
		name, code string
		who        realization.Owner
	}{
		{"out/a.go", "", who}, {"out/b.go", "", who},
		{"outside/a.go", "claim.outside_creation_area", who},
		{"out", "claim.exclusive_conflict", who},
		{"out/a.go/child", "claim.exclusive_conflict", who},
		{"out/a.go", "claim.exclusive_conflict", other},
		{"out/c.go", "claim.outside_creation_area", other},
		{"elsewhere/c.go", "", other},
		{"out/foreign.go", "claim.unknown_owner", owner(t, "foreign")},
	} {
		// Each attempt has a distinct key even when testing the same exact file.
		r := request(t, strconv.Itoa(i), tc.name, tc.who)
		next, d := realization.ReserveNewPath(f, claims, r)
		want := realization.PathAdmissible
		if tc.code != "" {
			want = realization.PathRefused
		}
		decision(t, d, want, tc.code)
		if d.Repeated() {
			t.Fatal("new request reported as repeat")
		}
		if _, ok := d.Claim(); ok != (want == realization.PathAdmissible) {
			t.Fatal("claim availability differs")
		}
		claims = next
	}
	if len(claims.Entries()) != 3 {
		t.Fatalf("claims = %d", len(claims.Entries()))
	}
	rootFacts := facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, []realization.CreationArea{area(t, realization.RootOwner(), ".")})
	_, d := realization.ReserveNewPath(rootFacts, realization.PathClaims{}, request(t, "root", "top.txt", realization.RootOwner()))
	decision(t, d, realization.PathAdmissible, "")
	_, d = realization.ReserveNewPath(f, realization.PathClaims{}, request(t, "directory", "out", who))
	decision(t, d, realization.PathRefused, "claim.outside_creation_area")
}

func TestProtectionWinsOverOwnershipAndNewSelections(t *testing.T) {
	who := owner(t, "empty")
	file, err := realization.ProtectFile(path(t, "private/config"), "Fixed input.")
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := snapshot.DirectoryAt(path(t, "secret"))
	protected, err := realization.ProtectDirectory(dir, "Fixed protected subtree.")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"private", "private/config", "private/config/child", "secret", "secret/child", ".markitect", ".markitect/state", ".markitect/state/future"} {
		t.Run(name, func(t *testing.T) {
			reg := realization.Registry{}
			// State itself cannot be registered; ancestor and other protected files can.
			if name != ".markitect/state" && name != ".markitect/state/future" {
				reg = registry(t, registration(t, who, artifact(t, name, borrow)))
			}
			f := facts(t, compiledFacts(), reg, snapshot.Tree{}, []realization.CreationArea{area(t, who, ".")}, file, protected)
			decision(t, realization.CheckOwnedPath(f, who, path(t, name)), realization.PathRefused, "claim.protected")
			_, d := realization.ReserveNewPath(f, realization.PathClaims{}, request(t, name, name, who))
			decision(t, d, realization.PathRefused, "claim.protected")
		})
	}
	f := facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, []realization.CreationArea{area(t, who, ".")}, file, protected)
	for _, unknown := range []realization.Owner{{}, owner(t, "foreign")} {
		decision(t, realization.CheckOwnedPath(f, unknown, path(t, "private/config")), realization.PathRefused, "claim.protected")
		decision(t, realization.CheckOwnedPath(f, unknown, path(t, ".markitect/state/future")), realization.PathRefused, "claim.protected")
	}
	_, denied := realization.ReserveNewPath(f, realization.PathClaims{}, request(t, "foreign-protected", "private/config", owner(t, "foreign")))
	decision(t, denied, realization.PathRefused, "claim.protected")
	for _, name := range []string{"private/configuration", "secrets/child", ".markitect/statex"} {
		_, d := realization.ReserveNewPath(f, realization.PathClaims{}, request(t, name, name, who))
		decision(t, d, realization.PathAdmissible, "")
	}
	root, _ := realization.ProtectDirectory(snapshot.RootDirectory(), "All fixed.")
	f = facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, []realization.CreationArea{area(t, who, ".")}, root)
	_, d := realization.ReserveNewPath(f, realization.PathClaims{}, request(t, "all", "a", who))
	decision(t, d, realization.PathRefused, "claim.protected")
}

func TestNewClaimsNeverAcquireExistingOrUnexplainedFiles(t *testing.T) {
	who := owner(t, "empty")
	for _, present := range []bool{false, true} {
		reg := registry(t, registration(t, owner(t, "lending"), artifact(t, "out/owned", borrow)))
		files := snapshot.Tree{}
		if present {
			files = fileTree(t, "out/owned")
		}
		f := facts(t, compiledFacts(), reg, files, []realization.CreationArea{area(t, who, ".")})
		for _, name := range []string{"out", "out/owned", "out/owned/child"} {
			_, d := realization.ReserveNewPath(f, realization.PathClaims{}, request(t, name, name, who))
			decision(t, d, realization.PathRefused, "claim.registered_conflict")
		}
	}
	f := facts(t, compiledFacts(), realization.Registry{}, fileTree(t, "out/unknown"), []realization.CreationArea{area(t, who, ".")})
	for _, name := range []string{"out", "out/unknown", "out/unknown/child"} {
		_, d := realization.ReserveNewPath(f, realization.PathClaims{}, request(t, name, name, who))
		decision(t, d, realization.PathRefused, "claim.intake_required")
	}
	_, d := realization.ReserveNewPath(f, realization.PathClaims{}, request(t, "sibling", "out/sibling", who))
	decision(t, d, realization.PathAdmissible, "")
}

func TestOwnedRepairDoesNotRequireCompleteAccountingOrCreationArea(t *testing.T) {
	who := owner(t, "empty")
	reg := registry(t, registration(t, who, artifact(t, "old/file", model.Identity{Package: "gone", ID: "removed"})))
	files := fileTree(t, "old/file", "unexplained")
	if realization.AccountFiles(compiledFacts(), files, reg, nil).Status() == realization.Accounted {
		t.Fatal("fixture should need repair")
	}
	f := facts(t, compiledFacts(), reg, files, nil)
	decision(t, realization.CheckOwnedPath(f, who, path(t, "old/file")), realization.PathAdmissible, "")
	decision(t, realization.CheckOwnedPath(f, realization.RootOwner(), path(t, "old/file")), realization.PathRefused, "claim.not_owner")
	decision(t, realization.CheckOwnedPath(f, who, path(t, "unexplained")), realization.PathRefused, "claim.not_registered")
	f = facts(t, compiledFacts(), reg, snapshot.Tree{}, nil)
	decision(t, realization.CheckOwnedPath(f, who, path(t, "old/file")), realization.PathAdmissible, "")
	for _, collision := range []string{"old", "old/file/child"} {
		f = facts(t, compiledFacts(), reg, fileTree(t, collision), nil)
		decision(t, realization.CheckOwnedPath(f, who, path(t, "old/file")), realization.PathRefused, "claim.file_directory_conflict")
	}
	// Removing a Package does not silently assign its artifacts to its ancestor.
	removed := owner(t, "empty/leaf")
	reg = registry(t, registration(t, removed, artifact(t, "old/file", borrow)))
	compiled := compiledFacts()
	compiled.Models[0].Packages = compiled.Models[0].Packages[:3]
	f = facts(t, compiled, reg, files, nil)
	decision(t, realization.CheckOwnedPath(f, removed, path(t, "old/file")), realization.PathRefused, "claim.unknown_owner")
	decision(t, realization.CheckOwnedPath(f, who, path(t, "old/file")), realization.PathRefused, "claim.not_owner")
	// The Host must supply its recorded takeover as explicit updated ownership.
	reg = registry(t, registration(t, who, artifact(t, "old/file", borrow)))
	f = facts(t, compiled, reg, files, nil)
	decision(t, realization.CheckOwnedPath(f, who, path(t, "old/file")), realization.PathAdmissible, "")
}

func TestClaimReplayRechecksFactsAndPreservesImmutableValues(t *testing.T) {
	who := owner(t, "empty")
	compiled := compiledFacts()
	areas := []realization.CreationArea{area(t, who, "out"), area(t, who, "other")}
	p, _ := realization.ProtectFile(path(t, "input"), "Protected input.")
	protections := []realization.Protection{p}
	f := facts(t, compiled, realization.Registry{}, snapshot.Tree{}, areas, protections...)
	r := request(t, "opaque-key", "out/file", who)
	var before realization.PathClaims
	claims, d := realization.ReserveNewPath(f, before, r)
	decision(t, d, realization.PathAdmissible, "")
	if len(before.Entries()) != 0 {
		t.Fatal("reservation changed old set")
	}
	claim, ok := d.Claim()
	if !ok || claim.Request() != r || r.ID() != "opaque-key" || r.Owner() != who || r.Path().String() != "out/file" || r.Form() != realization.Code {
		t.Fatal("request facts lost")
	}
	claims.Entries()[0] = realization.PathClaim{}
	compiled.Models[0].Packages[2].Name = "changed"
	areas[0] = realization.CreationArea{}
	protections[0] = realization.Protection{}
	_, d = realization.ReserveNewPath(f, claims, r)
	decision(t, d, realization.PathAdmissible, "")
	if !d.Repeated() || len(claims.Entries()) != 1 {
		t.Fatal("exact retry not recognized")
	}
	// Equivalent reordered facts have the same basis.
	areas = []realization.CreationArea{area(t, who, "other"), area(t, who, "out")}
	reordered := compiledFacts()
	slices.Reverse(reordered.Models)
	for i := range reordered.Models {
		slices.Reverse(reordered.Models[i].Packages)
	}
	f2 := facts(t, reordered, realization.Registry{}, snapshot.Tree{}, areas, p)
	_, d = realization.ReserveNewPath(f2, claims, r)
	decision(t, d, realization.PathAdmissible, "")
	if !d.Repeated() {
		t.Fatal("input order changed retry")
	}
	for _, changed := range []realization.PathRequest{
		request(t, "opaque-key", "out/other", who), request(t, "opaque-key", "out/file", owner(t, "lending")),
		func() realization.PathRequest {
			r, _ := realization.NewPathRequest("opaque-key", who, path(t, "out/file"), realization.Doc)
			return r
		}(),
	} {
		_, d = realization.ReserveNewPath(f, claims, changed)
		decision(t, d, realization.PathRefused, "claim.id_conflict")
	}
	for _, tc := range []struct {
		name   string
		change func() realization.PathFacts
		code   string
	}{
		{"tree", func() realization.PathFacts {
			return facts(t, compiledFacts(), realization.Registry{}, fileTree(t, "unrelated"), areas, p)
		}, "claim.basis_changed"},
		{"registry", func() realization.PathFacts {
			reg := registry(t, registration(t, who, artifact(t, "unrelated", borrow)))
			return facts(t, compiledFacts(), reg, snapshot.Tree{}, areas, p)
		}, "claim.basis_changed"},
		{"digest", func() realization.PathFacts {
			c := compiledFacts()
			c.Models[0].Digest = "changed"
			return facts(t, c, realization.Registry{}, snapshot.Tree{}, areas, p)
		}, "claim.basis_changed"},
		{"empty package", func() realization.PathFacts {
			c := compiledFacts()
			c.Models[0].Packages = append(c.Models[0].Packages, model.Package{Name: "new-empty"})
			return facts(t, c, realization.Registry{}, snapshot.Tree{}, areas, p)
		}, "claim.basis_changed"},
		{"selection", func() realization.PathFacts {
			return facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, nil, p)
		}, "claim.basis_changed"},
		{"protection", func() realization.PathFacts {
			protected, _ := realization.ProtectFile(path(t, "out/file"), "Now protected.")
			return facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, areas, protected)
		}, "claim.protected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next, d := realization.ReserveNewPath(tc.change(), claims, r)
			decision(t, d, realization.PathRefused, tc.code)
			if len(next.Entries()) != 1 || d.Repeated() {
				t.Fatal("refusal changed claims")
			}
			copy := d.Findings()
			copy[0].Code = "changed"
			if d.Findings()[0].Code != tc.code {
				t.Fatal("mutable finding alias")
			}
		})
	}
}

func TestClaimFactsRequireCompiledModelButNotAccountedRegistry(t *testing.T) {
	for _, tc := range []struct {
		status model.Status
		want   error
	}{{model.NotCompleted, realization.ErrIncompleteClaimFacts}, {model.Failed, realization.ErrInvalidClaimFacts}} {
		c := compiledFacts()
		c.Status = tc.status
		_, err := realization.NewPathFacts(c, realization.Registry{}, snapshot.Tree{}, nil, nil)
		if !errors.Is(err, tc.want) {
			t.Fatalf("error %v, want %v", err, tc.want)
		}
	}
	decision(t, realization.CheckOwnedPath(realization.PathFacts{}, realization.RootOwner(), path(t, "a")), realization.PathIncomplete, "claim.facts_unavailable")
	_, d := realization.ReserveNewPath(realization.PathFacts{}, realization.PathClaims{}, request(t, "key", "a", realization.RootOwner()))
	decision(t, d, realization.PathIncomplete, "claim.facts_unavailable")
	f := facts(t, compiledFacts(), realization.Registry{}, snapshot.Tree{}, nil)
	_, d = realization.ReserveNewPath(f, realization.PathClaims{}, realization.PathRequest{})
	decision(t, d, realization.PathRefused, "claim.invalid_request")
}
