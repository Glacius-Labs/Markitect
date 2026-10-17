package apply

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func encodeEntry(t *testing.T, path, content string) snapshot.Entry {
	t.Helper()
	p, err := snapshot.NewPath(path)
	if err != nil {
		t.Fatal(err)
	}
	e, err := snapshot.NewEntry(p, snapshot.Regular, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func encodeRecord(t *testing.T, owner realization.Owner, path, why string, defs ...string) realization.Registration {
	t.Helper()
	p, err := snapshot.NewPath(path)
	if err != nil {
		t.Fatal(err)
	}
	var basis []model.Identity
	for _, d := range defs {
		i := strings.LastIndex(d, "/")
		basis = append(basis, model.Identity{Package: d[:i], ID: d[i+1:]})
	}
	a, err := realization.NewArtifact(p, realization.Code, basis, why)
	if err != nil {
		t.Fatal(err)
	}
	r, err := realization.NewRegistration(owner, a)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// A publication regenerates exactly the registry shards whose records changed.
// Remaining records keep their captured order; new ones follow in path order.
func TestPublicationTreeRegeneratesChangedShardsOnly(t *testing.T) {
	lending, _ := realization.PackageOwner("lending")
	other, _ := realization.PackageOwner("other")
	shard := "apiVersion: markitect.registry/v1\nartifacts:\n  - path: \"src/z.go\"\n    form: code\n    realizes: [lending/Item]\n    why: Z first.\n  - path: \"src/a.go\"\n    form: code\n    realizes: [lending/Item]\n    why: A second.\n"
	otherShard := "apiVersion: markitect.registry/v1\nartifacts:\n  - path: \"src/o.go\"\n    form: code\n    realizes: [other/Thing]\n    why: Unchanged.\n"
	files := []snapshot.Entry{
		encodeEntry(t, ".markitect/registry/lending/registry.yaml", shard), encodeEntry(t, ".markitect/registry/other/registry.yaml", otherShard),
		encodeEntry(t, "src/z.go", "z"), encodeEntry(t, "src/a.go", "a"), encodeEntry(t, "src/o.go", "o"),
	}
	captured, err := snapshot.NewTree(files)
	if err != nil {
		t.Fatal(err)
	}
	before, err := realization.NewRegistry([]realization.Registration{
		encodeRecord(t, lending, "src/z.go", "Z first.", "lending/Item"), encodeRecord(t, lending, "src/a.go", "A second.", "lending/Item"), encodeRecord(t, other, "src/o.go", "Unchanged.", "other/Thing"),
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := realization.NewRegistry([]realization.Registration{
		encodeRecord(t, lending, "src/z.go", "Z first.", "lending/Item"), encodeRecord(t, lending, "src/a.go", "A changed.", "lending/Item", "lending/Rule"),
		encodeRecord(t, lending, "src/b.go", "B new.", "lending/Item"), encodeRecord(t, other, "src/o.go", "Unchanged.", "other/Thing"),
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err := snapshot.NewTree(append(files, encodeEntry(t, "src/b.go", "b")))
	if err != nil {
		t.Fatal(err)
	}
	target, err := publicationTree(captured, before, aggregate, after)
	if err != nil {
		t.Fatal(err)
	}
	changes := snapshot.Compare(captured, target)
	var paths []string
	for _, c := range changes {
		paths = append(paths, c.Path().String())
	}
	if strings.Join(paths, ",") != ".markitect/registry/lending/registry.yaml,src/b.go" {
		t.Fatal("unexpected changes", paths)
	}
	p, _ := snapshot.NewPath(".markitect/registry/lending/registry.yaml")
	entry, _ := target.Lookup(p)
	want := "apiVersion: markitect.registry/v1\nartifacts:\n  - path: \"src/z.go\"\n    form: code\n    realizes: [lending/Item]\n    why: Z first.\n  - path: \"src/a.go\"\n    form: code\n    realizes: [lending/Item, lending/Rule]\n    why: A changed.\n  - path: \"src/b.go\"\n    form: code\n    realizes: [lending/Item]\n    why: B new.\n"
	if string(entry.Bytes()) != want {
		t.Fatalf("regenerated shard:\n%s\nwant:\n%s", entry.Bytes(), want)
	}
	if got := shardOrder(entry.Bytes()); strings.Join(got, ",") != "src/z.go,src/a.go,src/b.go" {
		t.Fatal("regenerated shard does not decode", got)
	}
}
