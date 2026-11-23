package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// sites is a small spec with a chain of three types: artifact, website that
// extends it and blog that extends website, and a document of each.
var sites = map[string]string{
	"types/artifact.md": "# Artifact\n\nSomething a team makes.\n\n## Owner\nWho answers for it.\n\n## Notes (optional)\nAnything else.\n",
	"types/website.md":  "# Website\n\nExtends: [Artifact](artifact.md)\n\nA site people open in a browser.\n\n## Pages\nThe pages, each a page.\n",
	"types/blog.md":     "# Blog\n\nExtends: [Website](website.md)\n\nA website of posts.\n\n## Feed\nWhere the posts are listed.\n\n## Notes\nNow required.\n",
	"things/logo.md":    "# Logo\n\nType: [Artifact](../types/artifact.md)\n\n## Owner\nAda\n",
	"sites/home.md":     "# Home\n\nType: [Website](../types/website.md)\n\n## Owner\nGrace\n\n## Pages\nThree | four\n",
	"sites/news.md":     "# News\n\nType: [Blog](../types/blog.md)\n\n## Owner\nAlan\n\n## Pages\nOne\n\n## Feed\nDaily\n\n## Notes\nNone.\n",
}

func TestCheckDerivedTypes(t *testing.T) {
	withReadmes := map[string]string{
		"types/README.md":  readme("Types", "artifact.md", "website.md", "blog.md"),
		"things/README.md": readme("Things", "logo.md"),
		"sites/README.md":  readme("Sites", "home.md", "news.md"),
	}
	for p, text := range sites {
		withReadmes[p] = text
	}
	dir := spec(t, withReadmes)
	if code, out, _ := run("check", "-C", dir); code != ExitOK || out != "0 errors in 9 documents\n" {
		t.Fatalf("%d %s", code, out)
	}
	broken := map[string]string{}
	for p, text := range withReadmes {
		broken[p] = text
	}
	broken["sites/news.md"] = "# News\n\nType: [Blog](../types/blog.md)\n\n## Owner\n\n## Pages\nOne\n\n## Feed\nDaily\n"
	broken["types/loop.md"] = "# Loop\n\nExtends: [Loop](loop.md)\n"
	broken["types/lost.md"] = "# Lost\n\nExtends: [Nowhere](nowhere.md)\n"
	broken["types/loose.md"] = "# Loose\n\nExtends: [Artifact](artifact.md)\n\n## Owner (optional)\nO.\n"
	broken["types/bad.md"] = "# Bad\n\nExtends: artifact\n\nExtends: [Artifact](artifact.md)\n"
	code, out, _ := run("check", "-C", spec(t, broken))
	want := `sites/news.md:3: section.missing: missing section "Notes" required by types/blog.md` + "\n" +
		`sites/news.md:5: section.empty: empty section "Owner"` + "\n" +
		`types/bad.md:3: extends.line: extends line must be exactly one link to a document` + "\n" +
		`types/bad.md:5: extends.line: an extends line stands only on the first line after the title` + "\n" +
		`types/loop.md:3: extends.cycle: the chain of extends lines comes back to itself` + "\n" +
		`types/loose.md:5: extends.loosens: section "Owner" is required by types/artifact.md` + "\n" +
		`types/lost.md:3: extends.unresolved: no document at "nowhere.md"` + "\n" +
		"7 errors in 13 documents\n"
	if code != ExitFound || out != want {
		t.Errorf("got %d\n%s\nwant\n%s", code, out, want)
	}
}

func TestShowDerivedType(t *testing.T) {
	dir := spec(t, sites)
	code, out, _ := run("show", "-C", dir, "types/blog")
	want := "types/blog.md\nTitle: Blog\nExtends: types/website.md\nSections:\n  Owner (from types/artifact.md)\n  Notes\n  Pages (from types/website.md)\n  Feed\n" +
		"Links to: (none)\nLinked from: (none)\nDocuments:\n  sites/news.md\n"
	if code != ExitOK || out != want {
		t.Errorf("got %d\n%s\nwant\n%s", code, out, want)
	}
	// A section that the type states again is its own, and the base lists the
	// documents of every type that extends it.
	_, out, _ = run("show", "-C", dir, "types/artifact")
	if !strings.Contains(out, "Type: (none)\nSections:\n  Owner\n  Notes (optional)\n") ||
		!strings.HasSuffix(out, "Documents:\n  sites/home.md\n  sites/news.md\n  things/logo.md\n") {
		t.Error(out)
	}
	// A base lists the types that extend it directly, after its sections.
	_, out, _ = run("show", "-C", dir, "types/artifact")
	want = "types/artifact.md\nTitle: Artifact\nType: (none)\nSections:\n  Owner\n  Notes (optional)\nExtended by:\n  types/website.md\n" +
		"Links to: (none)\nLinked from: (none)\nDocuments:\n  sites/home.md\n  sites/news.md\n  things/logo.md\n"
	if out != want {
		t.Errorf("got\n%s\nwant\n%s", out, want)
	}
	_, out, _ = run("show", "-C", dir, "types/artifact", "--json")
	if !strings.Contains(out, "\"extended_by\": [\n    \"types/website.md\"\n  ]") {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "types/blog", "--json")
	if strings.Contains(out, "extended_by") {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "types/website", "--json")
	var doc struct {
		Type         string
		TypeState    string `json:"type_state"`
		Extends      string
		ExtendsState string `json:"extends_state"`
		Sections     []struct {
			Heading    string
			DeclaredBy string `json:"declared_by"`
		}
		Documents []string
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Type != "" || doc.TypeState != "none" || doc.Extends != "types/artifact.md" || doc.ExtendsState != "ok" || len(doc.Sections) != 3 ||
		doc.Sections[0].Heading != "Owner" || doc.Sections[0].DeclaredBy != "types/artifact.md" || doc.Sections[1].Heading != "Notes (optional)" ||
		doc.Sections[2].Heading != "Pages" || doc.Sections[2].DeclaredBy != "" || strings.Join(doc.Documents, " ") != "sites/home.md sites/news.md" {
		t.Errorf("%s", out)
	}
	// An object and a type without an extends line print nothing new.
	for _, p := range []string{"sites/home", "types/artifact"} {
		_, out, _ = run("show", "-C", dir, p, "--json")
		if strings.Contains(out, "extends") || strings.Contains(out, "declared_by") {
			t.Errorf("%s: %s", p, out)
		}
	}
	_, out, _ = run("show", "-C", dir, "sites/home")
	if out != "sites/home.md\nTitle: Home\nType: types/website.md\nSections:\n  Owner\n  Pages\nLinks to: (none)\nLinked from: (none)\n" {
		t.Error(out)
	}
	// A base that is no document; an extends line that is no link counts for nothing.
	dir = spec(t, map[string]string{
		"lost.md": "# Lost\n\nExtends: [Nowhere](nowhere.md)\n\n## Own\nO.\n",
		"bad.md":  "# Bad\n\nExtends: nowhere\n\n## Own\nO.\n",
	})
	if _, out, _ := run("show", "-C", dir, "lost"); !strings.HasPrefix(out, "lost.md\nTitle: Lost\nExtends: nowhere.md (missing)\nSections:\n  Own\n") {
		t.Error(out)
	}
	if _, out, _ := run("show", "-C", dir, "bad"); !strings.HasPrefix(out, "bad.md\nTitle: Bad\nType: (none)\nSections:\n  Own\n") {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "lost", "--json")
	if !strings.Contains(out, `"extends": "nowhere.md"`) || !strings.Contains(out, `"extends_state": "missing"`) {
		t.Error(out)
	}
}

func TestTableOfDerivedTypes(t *testing.T) {
	dir := spec(t, sites)
	_, out, _ := run("table", "-C", dir, "types/artifact")
	// Rows are in path order: sites/ before things/.
	want := "| Title | Owner | Notes |\n| --- | --- | --- |\n" +
		"| Home | Grace |  |\n| News | Alan | None. |\n| Logo | Ada |  |\n"
	if out != want {
		t.Errorf("got\n%s\nwant\n%s", out, want)
	}
	_, out, _ = run("table", "-C", dir, "types/website")
	if out != "| Title | Owner | Notes | Pages |\n| --- | --- | --- | --- |\n| Home | Grace |  | Three \\| four |\n| News | Alan | None. | One |\n" {
		t.Error(out)
	}
	_, out, _ = run("table", "-C", dir, "types/blog", "--json")
	var doc tableDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Type != "types/blog.md" || strings.Join(doc.Columns, ",") != "Owner,Notes,Pages,Feed" || len(doc.Rows) != 1 || doc.Rows[0].Path != "sites/news.md" ||
		strings.Join(doc.Rows[0].Cells, ",") != "Alan,None.,One,Daily" {
		t.Error(out)
	}
	// The table of all documents shows each type as a document without a type.
	_, out, _ = run("table", "-C", dir)
	if !strings.Contains(out, "| types/blog.md |  | Blog |\n") || !strings.Contains(out, "| sites/news.md | types/blog.md | News |\n") {
		t.Error(out)
	}
}

func TestTopicOfDerivedTypes(t *testing.T) {
	dir := spec(t, sites)
	_, out, _ := run("topic", "-C", dir, "types/artifact", "--json")
	var doc struct {
		TopicKind string   `json:"topic_kind"`
		Typed     []string `json:"typed_documents"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.TopicKind != "type" || strings.Join(doc.Typed, " ") != "sites/home.md sites/news.md things/logo.md" {
		t.Error(out)
	}
	_, out, _ = run("topic", "-C", dir, "types/website")
	if !strings.HasPrefix(out, "# Topic: types/website.md (type)\n") || !strings.Contains(out, "\n## Documents\n\n| Title | Owner | Notes | Pages |\n") ||
		!strings.Contains(out, "| News | Alan | None. | One |\n") {
		t.Error(out)
	}
	// A derived type that no document names is a type all the same.
	_, out, _ = run("topic", "-C", dir, "types/blog", "--json")
	if !strings.Contains(out, `"topic_kind": "type"`) {
		t.Error(out)
	}
	dir = spec(t, map[string]string{"types/blog.md": sites["types/blog.md"], "types/website.md": sites["types/website.md"], "types/artifact.md": sites["types/artifact.md"]})
	for _, p := range []string{"types/blog", "types/website", "types/artifact"} {
		if _, out, _ := run("topic", "-C", dir, p, "--json"); !strings.Contains(out, `"topic_kind": "type"`) {
			t.Errorf("%s: %s", p, out)
		}
	}
}
