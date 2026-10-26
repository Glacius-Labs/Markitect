package harness

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// The rendered context of Root, a parent and a leaf over a passed check
// result: a local model in a non-root repository area, an in-repository
// Library and an external Core without a repository file.
func TestWriteContextSectionsChildAreasAndFiles(t *testing.T) {
	coreKind := model.Identity{Package: "markitect/core", ID: "Kind"}
	definition := func(modelID, origin, pkg, id string, links ...model.Identity) model.Definition {
		d := model.Definition{
			Identity: model.Identity{Package: pkg, ID: id}, ModelID: modelID, Kind: coreKind, Body: "Body of " + pkg + "/" + id,
			Source: model.Source{Origin: origin, Path: "model/" + pkg + "/" + id + ".yaml", Digest: "sha256:" + id},
		}
		for _, link := range links {
			d.BodyLinks = append(d.BodyLinks, model.BodyLink{Target: link})
		}
		return d
	}
	core := definition("markitect-core", "external:core", "markitect/core", "Kind")
	shared := definition("lib", "repo:lib", "lib", "Shared")
	near := definition("project", "repo:project", "b", "near")
	own := definition("project", "repo:project", "a", "own", near.Identity, shared.Identity)
	child := definition("project", "repo:project", "a/child", "child")
	title := "Child title"
	child.Title = &title
	checked := model.Result{Status: model.Passed, Models: []model.ModelResult{
		{Origin: "repo:project", Local: true, ID: "project", Status: model.Passed, Digest: "sha256:project",
			Packages: []model.Package{{Name: "a"}, {Name: "a/child"}, {Name: "b"}, {Name: "empty"}}, Definitions: []model.Definition{own, child, near}},
		{Origin: "repo:lib", ID: "lib", Status: model.Passed, Digest: "sha256:lib", Definitions: []model.Definition{shared}},
		{Origin: "external:core", ID: "markitect-core", Status: model.Passed, Digest: "sha256:core", Definitions: []model.Definition{core}},
	}}
	areas := map[string]string{"repo:project": "sub", "repo:lib": "libraries/lib", "external:core": ""}
	render := func(pkg string) string {
		t.Helper()
		selection, err := organization.SelectContext(checked, pkg, nil)
		if err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		if err := writeContext(&b, selection, areas); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	// between reports whether part lies after from and before to.
	between := func(text, part, from, to string) bool {
		i, j, k := strings.Index(text, from), strings.Index(text, part), strings.Index(text, to)
		return i >= 0 && j > i && (to == "" || k > j)
	}
	const ownLabel, seeAlso, contracts, children = "Own Definitions (", "See also: linked and scoped Definitions (", "Type contracts (", "Child areas ("

	root := render(".")
	for _, part := range []string{
		"Own Definitions: none\n", "See also: linked and scoped Definitions: none\n", "Type contracts: none\n",
		"Child areas (compact index of each direct child's subtree):\n" +
			"a:\n- a/child/child kind markitect/core/Kind title \"Child title\" file sub/model/a/child/child.yaml\n- a/own kind markitect/core/Kind file sub/model/a/own.yaml\n" +
			"b:\n- b/near kind markitect/core/Kind file sub/model/b/near.yaml\n" +
			"empty: none\n",
	} {
		if !strings.Contains(root, part) {
			t.Errorf("Root context lacks %q:\n%s", part, root)
		}
	}
	if strings.Contains(root, "Body of") {
		t.Errorf("Root context renders descendant bodies:\n%s", root)
	}

	parent := render("a")
	for _, check := range []struct{ part, from, to string }{
		{"Body of a/own", ownLabel, seeAlso},
		{"Body of b/near", seeAlso, contracts},
		{"Body of lib/Shared", seeAlso, contracts},
		{"Body of markitect/core/Kind", contracts, children},
		{"a/child:\n- a/child/child kind markitect/core/Kind title \"Child title\" file sub/model/a/child/child.yaml\n", children, ""},
	} {
		if !between(parent, check.part, check.from, check.to) {
			t.Errorf("parent context lacks %q between %q and %q:\n%s", check.part, check.from, check.to, parent)
		}
	}
	for _, file := range []string{`"File":"sub/model/a/own.yaml"`, `"File":"sub/model/b/near.yaml"`, `"File":"libraries/lib/model/lib/Shared.yaml"`} {
		if !strings.Contains(parent, file) {
			t.Errorf("parent context lacks %s", file)
		}
	}
	if strings.Count(parent, `"File":`) != 3 || strings.Contains(parent, "Body of a/child/child") {
		t.Errorf("external Core gained a file or a child body was inlined:\n%s", parent)
	}

	if incoming := render("b"); !between(incoming, "Body of a/own", seeAlso, contracts) {
		t.Errorf("incoming Definition missing from See also:\n%s", incoming)
	}

	leaf := render("a/child")
	for _, part := range []string{"See also: linked and scoped Definitions: none\n", "Child areas: none\n"} {
		if !strings.Contains(leaf, part) {
			t.Errorf("leaf context lacks %q:\n%s", part, leaf)
		}
	}
	if !between(leaf, "Body of a/child/child", ownLabel, contracts) {
		t.Errorf("leaf own body missing:\n%s", leaf)
	}
}
