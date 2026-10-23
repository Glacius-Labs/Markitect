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

func path(t *testing.T, name string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func owner(t *testing.T, name string) realization.Owner {
	t.Helper()
	o, err := realization.PackageOwner(name)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func artifact(t *testing.T, name string, basis ...model.Identity) realization.Artifact {
	t.Helper()
	a, err := realization.NewArtifact(path(t, name), realization.Code, basis, "Explains the model basis.")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func artifactForm(t *testing.T, name string, form realization.Form, basis ...model.Identity) realization.Artifact {
	t.Helper()
	a, err := realization.NewArtifact(path(t, name), form, basis, "Explains the model basis.")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func directoryArtifact(t *testing.T, name string, basis ...model.Identity) realization.Artifact {
	t.Helper()
	a, err := realization.NewDirectoryArtifact(path(t, name), realization.Code, basis, "Explains the model basis of the directory.")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func registration(t *testing.T, o realization.Owner, a realization.Artifact) realization.Registration {
	t.Helper()
	r, err := realization.NewRegistration(o, a)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func registry(t *testing.T, entries ...realization.Registration) realization.Registry {
	t.Helper()
	r, err := realization.NewRegistry(entries)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestFixedRegistryValuesAndManyToManyLinks(t *testing.T) {
	borrow := model.Identity{Package: "lending", ID: "borrow"}
	policy := model.Identity{Package: "lending", ID: "loan-policy"}
	basis := []model.Identity{policy, borrow}
	a, err := realization.NewArtifact(path(t, "src/Borrow.go"), realization.Code, basis, "  Enforces admission.  ")
	if err != nil {
		t.Fatal(err)
	}
	want := []model.Identity{borrow, policy}
	basis[0] = model.Identity{}
	a.Basis()[0] = model.Identity{}
	if !reflect.DeepEqual(a.Basis(), want) || a.Path().String() != "src/Borrow.go" || a.Why() != "  Enforces admission.  " || a.Form() != realization.Code {
		t.Fatal("artifact changed spelling, explanation or model basis")
	}
	entries := []realization.Registration{
		registration(t, owner(t, "lending"), a),
		registration(t, owner(t, "lending"), artifact(t, "docs/loans.md", borrow)),
	}
	fixed := registry(t, entries...)
	slices.Reverse(entries)
	if !reflect.DeepEqual(fixed.Entries(), registry(t, entries...).Entries()) {
		t.Fatal("order changed registry")
	}
	entries[0] = realization.Registration{}
	fixed.Entries()[0] = realization.Registration{}
	got, found := fixed.Lookup(a.Path())
	if !found || got.Owner() != owner(t, "lending") || !reflect.DeepEqual(got.Artifact().Basis(), want) {
		t.Fatal("registry exposed aliases")
	}
	if _, found := fixed.Lookup(path(t, "src/borrow.go")); found {
		t.Fatal("lookup folded case")
	}
	if _, found := (realization.Registry{}).Lookup(a.Path()); found {
		t.Fatal("empty registry found entry")
	}
	if name, present := realization.RootOwner().Package(); name != "" || present || !realization.RootOwner().IsRoot() {
		t.Fatal("root became a canonical Package")
	}
	if name, present := owner(t, "lending/reminders").Package(); name != "lending/reminders" || !present {
		t.Fatal("Package owner changed")
	}
}

func TestRegistryRejectsAmbiguousAndUnjustifiedValues(t *testing.T) {
	basis := []model.Identity{{Package: "feature", ID: "behavior"}}
	for _, name := range []string{"", ".", "../feature", "/feature"} {
		if _, err := realization.PackageOwner(name); !errors.Is(err, realization.ErrInvalidOwner) {
			t.Fatalf("owner %q: %v", name, err)
		}
	}
	for _, tc := range []struct {
		name  string
		p     snapshot.Path
		form  realization.Form
		links []model.Identity
		why   string
	}{
		{"zero path", snapshot.Path{}, realization.Code, basis, "why"},
		{"state", path(t, ".markitect/state/control"), realization.Config, basis, "why"},
		{"form", path(t, "a"), "unknown", basis, "why"},
		{"no basis", path(t, "a"), realization.Code, nil, "why"},
		{"no reason", path(t, "a"), realization.Code, basis, " \n "},
		{"incomplete identity", path(t, "a"), realization.Code, []model.Identity{{ID: "behavior"}}, "why"},
		{"duplicate basis", path(t, "a"), realization.Code, append(slices.Clone(basis), basis[0]), "why"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := realization.NewArtifact(tc.p, tc.form, tc.links, tc.why); !errors.Is(err, realization.ErrInvalidArtifact) {
				t.Fatal(err)
			}
		})
	}
	if _, err := realization.NewRegistration(realization.Owner{}, artifact(t, "a", basis...)); !errors.Is(err, realization.ErrInvalidRegistry) {
		t.Fatal(err)
	}
	if _, err := realization.NewRegistration(realization.RootOwner(), realization.Artifact{}); !errors.Is(err, realization.ErrInvalidRegistry) {
		t.Fatal(err)
	}
	if _, err := realization.NewRegistry([]realization.Registration{{}}); !errors.Is(err, realization.ErrInvalidRegistry) {
		t.Fatal(err)
	}
	for _, names := range [][]string{{"a", "a"}, {"a", "a-else", "a/b"}, {"a/b", "a/b/c"}} {
		var entries []realization.Registration
		for i, name := range names {
			o := realization.RootOwner()
			if i > 0 {
				o = owner(t, "feature")
			}
			entries = append(entries, registration(t, o, artifact(t, name, basis...)))
		}
		_, first := realization.NewRegistry(entries)
		slices.Reverse(entries)
		_, second := realization.NewRegistry(entries)
		if !errors.Is(first, realization.ErrInvalidRegistry) || second == nil || first.Error() != second.Error() {
			t.Fatalf("collision errors: %v / %v", first, second)
		}
	}
	// Artifact paths are generic exact names; physical alias policy stays outside.
	registry(t, registration(t, realization.RootOwner(), artifact(t, "A", basis...)), registration(t, realization.RootOwner(), artifact(t, "a", basis...)))
}
