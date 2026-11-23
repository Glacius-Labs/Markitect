package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// shops is a small spec with property types: a type with a reference property
// and a property of fixed values, one that extends it, and a document of each.
var shops = map[string]string{
	"types/README.md":    readme("Types", "artifact.md", "website.md", "page.md", "orphan.md"),
	"types/artifact.md":  "# Artifact\n\nSomething a team makes.\n\n## Owner\nWho answers for it.\n\n## State\nType: `draft`, `live`, `retired`\n\nWhere it stands.\n",
	"types/website.md":   "# Website\n\nExtends: [Artifact](artifact.md)\n\nA site people open in a browser.\n\n## Pages\nType: [Page](page.md), [Ghost](ghost.md)\n\nThe pages of the site.\n\n## Hosting\nWhere and how it runs.\n",
	"types/page.md":      "# Page\n\nA page.\n\n## Body\nText.\n",
	"types/orphan.md":    "# Orphan\n\nNo one uses it.\n\n## Kind\nType: `a`\n",
	"sites/README.md":    readme("Sites", "home.md", "start.md"),
	"sites/home.md":      "# Home\n\nType: [Website](../types/website.md)\n\n## Owner\nAda\n\n## State\nlive\n\n## Pages\n[Start](start.md)\n\n## Hosting\nSomewhere.\n",
	"sites/start.md":     "# Start\n\nType: [Page](../types/page.md)\n\n## Body\nHello.\n",
	"sites/elsewhere.md": "# Elsewhere\n\nA document of no type.\n",
}

func TestShowTypeWithPropertyTypes(t *testing.T) {
	dir := spec(t, shops)
	_, out, _ := run("show", "-C", dir, "types/website")
	want := "types/website.md\nTitle: Website\nExtends: types/artifact.md\nSections:\n" +
		"  Owner (from types/artifact.md)\n" +
		"  State (from types/artifact.md): one of `draft`, `live`, `retired`\n" +
		"  Pages: reference to types/page.md, types/ghost.md (missing)\n" +
		"  Hosting\n" +
		"Links to: (none)\nLinked from:\n  types/README.md (Contents)\nDocuments:\n  sites/home.md\n"
	if out != want {
		t.Errorf("got\n%s\nwant\n%s", out, want)
	}
	// A type without an extends line shows its property types too.
	_, out, _ = run("show", "-C", dir, "types/artifact")
	if !strings.Contains(out, "Sections:\n  Owner\n  State: one of `draft`, `live`, `retired`\nExtended by:\n  types/website.md\n") {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "types/orphan")
	if out != "types/orphan.md\nTitle: Orphan\nType: (none)\nSections:\n  Kind: one of `a`\nLinks to: (none)\nLinked from:\n  types/README.md (Contents)\n" {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "types/website", "--json")
	var doc struct {
		Sections []struct {
			Heading    string
			DeclaredBy string `json:"declared_by"`
			References []struct {
				Path    string
				Missing bool
			}
			Values []string
		}
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Sections) != 4 || doc.Sections[0].Values != nil || doc.Sections[0].References != nil ||
		strings.Join(doc.Sections[1].Values, ",") != "draft,live,retired" || doc.Sections[1].DeclaredBy != "types/artifact.md" ||
		len(doc.Sections[2].References) != 2 || doc.Sections[2].References[0].Path != "types/page.md" || doc.Sections[2].References[0].Missing ||
		doc.Sections[2].References[1].Path != "types/ghost.md" || !doc.Sections[2].References[1].Missing ||
		doc.Sections[3].Values != nil || doc.Sections[3].References != nil {
		t.Errorf("%s", out)
	}
	// Text is left out of the JSON, and so is everything for what is no type.
	for _, p := range []string{"types/page", "sites/home", "sites/README"} {
		_, out, _ = run("show", "-C", dir, p, "--json")
		if strings.Contains(out, `"references"`) || strings.Contains(out, `"values"`) {
			t.Errorf("%s: %s", p, out)
		}
	}
	// An object, a table and a topic print nothing new.
	_, out, _ = run("show", "-C", dir, "sites/home")
	if out != "sites/home.md\nTitle: Home\nType: types/website.md\nSections:\n  Owner\n  State\n  Pages\n  Hosting\nLinks to:\n  sites/start.md\nLinked from:\n  sites/README.md (Contents)\n" {
		t.Error(out)
	}
	_, out, _ = run("table", "-C", dir, "types/website")
	if out != "| Title | Owner | State | Pages | Hosting |\n| --- | --- | --- | --- | --- |\n| Home | Ada | live | Start | Somewhere. |\n" {
		t.Error(out)
	}
}

func TestTypeByPropertyLineAloneIsAType(t *testing.T) {
	dir := spec(t, shops)
	_, out, _ := run("topic", "-C", dir, "types/orphan", "--json")
	if !strings.Contains(out, `"topic_kind": "type"`) {
		t.Error(out)
	}
	// The text under its sections is no statement, and its title is no name.
	_, out, _ = run("topic", "-C", dir, "sites/elsewhere", "--json")
	if !strings.Contains(out, `"topic_kind": "document"`) {
		t.Error(out)
	}
}

func TestCheckPropertyTypes(t *testing.T) {
	files := map[string]string{}
	for p, text := range shops {
		files[p] = text
	}
	files["sites/home.md"] = "# Home\n\nType: [Website](../types/website.md)\n\n## Owner\nAda\n\n## State\nwip\n\n## Pages\n[Start](start.md), [Elsewhere](elsewhere.md)\n\n## Hosting\nSomewhere.\n"
	files["sites/README.md"] = readme("Sites", "home.md", "start.md", "elsewhere.md")
	code, out, _ := run("check", "-C", spec(t, files))
	want := "sites/home.md:9: property.value: the first line of section \"State\" is \"wip\", not one of `draft`, `live`, `retired`\n" +
		"sites/home.md:12: reference.type: section \"Pages\" refers to documents of types/page.md or a type that extends it, but \"sites/elsewhere.md\" is a document without a type\n" +
		"types/website.md:8: property.unresolved: no document at \"ghost.md\"\n" +
		"3 errors in 9 documents\n"
	if code != ExitFound || out != want {
		t.Errorf("%d\n%s", code, out)
	}
	// A cycle is one diagnostic, in the JSON too.
	files = map[string]string{
		"a/README.md": readme("A", "x.md", "t.md"), "b/README.md": readme("B", "y.md", "t.md"),
		"a/t.md": "# T\n\nA.\n", "b/t.md": "# T\n\nB.\n",
		"a/x.md": "# X\n\nType: [T](../b/t.md)\n", "b/y.md": "# Y\n\nType: [T](../a/t.md)\n",
	}
	dir := spec(t, files)
	code, out, _ = run("check", "-C", dir)
	if code != ExitFound || out != "a/x.md:3: module.cycle: module cycle a -> b -> a: a/x.md:3 is of type b/t.md, b/y.md:3 is of type a/t.md\n1 error in 6 documents\n" {
		t.Errorf("%d\n%s", code, out)
	}
	_, out, _ = run("check", "-C", dir, "--json")
	if !strings.Contains(out, `"code": "module.cycle"`) || !strings.Contains(out, `"file": "a/x.md"`) {
		t.Error(out)
	}
}

func TestShowFixedValuesWithBackticks(t *testing.T) {
	dir := spec(t, map[string]string{"t.md": "# T\n\nA.\n\n## State\nType: ``a`b``, `live`\n"})
	_, out, _ := run("show", "-C", dir, "t")
	if !strings.Contains(out, "Sections:\n  State: one of ``a`b``, `live`\n") {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "t", "--json")
	if !strings.Contains(out, "\"a`b\"") {
		t.Error(out)
	}
}
