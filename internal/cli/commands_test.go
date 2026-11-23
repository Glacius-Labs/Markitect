package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// branches is a small spec of members and branches.
var branches = map[string]string{
	"types/member.md":   "# Member\n\nA card holder.\n\n## Card\nThe number.\n\n## Branch\nThe home branch, as a link.\n\n## Notes (optional)\nAnything else.\n",
	"types/branch.md":   "# Branch\n\nA place that lends books.\n",
	"places/central.md": "# Central\n\nType: [Branch](../types/branch.md)\n\nThe main building.\n",
	"members/ada.md":    "# Ada Lovelace\n\nType: [Member](../types/member.md)\n\n## Card\nL-1 | gold\n\n## Branch\n<!-- moved last year -->\n[Central](../places/central.md), see [Grace](grace.md#card).\n",
	"members/grace.md":  "# Grace Hopper\n\nType: [Member](../types/member.md)\n\n## card\nL-2\n\n## Branch\n[Main](../places/central.md)\n\n## Notes\n![photo](photo.png) on file, [missing](gone.md).\n",
}

func TestShow(t *testing.T) {
	dir := spec(t, branches)
	code, out, _ := run("show", "-C", dir, `members\ada`)
	want := "members/ada.md\nTitle: Ada Lovelace\nType: types/member.md\nSections:\n  Card\n  Branch\n" +
		"Links to:\n  members/grace.md\n  places/central.md\nLinked from: (none)\n"
	if code != ExitOK || out != want {
		t.Errorf("got %d\n%s\nwant\n%s", code, out, want)
	}
	_, out, _ = run("show", "-C", dir, "places/central.md")
	if !strings.HasSuffix(out, "Linked from:\n  members/ada.md (Branch)\n  members/grace.md (Branch)\n") {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "types/member")
	if !strings.Contains(out, "Type: (none)\nSections:\n  Card\n  Branch\n  Notes (optional)\n") ||
		!strings.HasSuffix(out, "Documents:\n  members/ada.md\n  members/grace.md\n") {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "members/grace", "--json")
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["kind"] != "show" || doc["type_state"] != "ok" || len(doc["links_to"].([]any)) != 2 || len(doc["documents"].([]any)) != 0 {
		t.Error(out)
	}
	for _, key := range []string{"fingerprint", "verdict", "artifacts", "counter_checks", "evidence", "reason", "conflict"} {
		if _, ok := doc[key]; ok {
			t.Errorf("show has the key %s", key)
		}
	}
	for _, arg := range []string{"nobody", "../outside", "types"} {
		if code, out, errOut := run("show", "-C", dir, arg); code != ExitFailed || out != "" || !strings.Contains(errOut, "unknown path") {
			t.Errorf("%s: %d %q %q", arg, code, out, errOut)
		}
	}
}

func TestTable(t *testing.T) {
	dir := spec(t, branches)
	code, out, _ := run("table", "-C", dir, "types/member")
	want := "| Title | Card | Branch | Notes |\n| --- | --- | --- | --- |\n" +
		"| Ada Lovelace | L-1 \\| gold | Central, see Grace. |  |\n" +
		"| Grace Hopper | L-2 | Main | photo on file, missing. |\n"
	if code != ExitOK || out != want {
		t.Errorf("got %d\n%s\nwant\n%s", code, out, want)
	}
}

func TestSpecTable(t *testing.T) {
	dir := spec(t, map[string]string{
		"types/operation.md": "# Operation\n\n## Rule\nWhat holds.\n",
		"lending/borrow.md":  "# Borrow | Lend\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA member borrows.\n",
		"lending/ghost.md":   "# Ghost\n\nType: [Nothing](../types/nothing.md)\n",
		"README.md":          "# Lending\n",
		"lending/return.md":  "# Return\n\nType:\n",
	})
	code, out, errOut := run("table", "-C", dir)
	want := "| Document | Type | Title |\n| --- | --- | --- |\n" +
		"| README.md |  | Lending |\n" +
		"| lending/borrow.md | types/operation.md | Borrow \\| Lend |\n" +
		"| lending/ghost.md | types/nothing.md | Ghost |\n" +
		"| lending/return.md |  | Return |\n" +
		"| types/operation.md |  | Operation |\n"
	if code != ExitOK || out != want || errOut != "" {
		t.Errorf("got %d\n%s\nwant\n%s", code, out, want)
	}
	code, out, _ = run("table", "-C", dir, "--json")
	var doc tableDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if code != ExitOK || doc.Kind != "table" || doc.Type != "" || strings.Join(doc.Columns, " ") != "Type" || len(doc.Rows) != 5 ||
		doc.Rows[1].Path != "lending/borrow.md" || doc.Rows[1].Title != "Borrow | Lend" || strings.Join(doc.Rows[1].Cells, " ") != "types/operation.md" {
		t.Errorf("got %d %s", code, out)
	}
	if code, out, _ := run("table", "-C", spec(t, nil)); code != ExitOK || out != "| Document | Type | Title |\n| --- | --- | --- |\n" {
		t.Errorf("an empty spec: %d %q", code, out)
	}
}

func TestTopicAndQuestions(t *testing.T) {
	dir := spec(t, branches)
	code, out, _ := run("topic", "-C", dir)
	if code != ExitOK || !strings.HasPrefix(out, "# Signals of the spec\n") || !strings.Contains(out, "\n### weak (") || !strings.Contains(out, "\n### unlinked-mention (") ||
		strings.Contains(out, "hange signals") {
		t.Fatal(code, out)
	}
	code, out, _ = run("topic", "-C", dir, "places/central")
	for _, part := range []string{"# Topic: places/central.md (document)\n", "\n## Strong statements about it\n", "\n## Text\n\n# Central\n",
		"\n## Linked from\n\nmembers/ada.md:10 (Branch): Central, see Grace.\n", "\n## Mentioned without a link\n", "\n## Signals\n"} {
		if code != ExitOK || !strings.Contains(out, part) {
			t.Errorf("missing %q in\n%s", part, out)
		}
	}
	_, out, _ = run("topic", "-C", dir, "types/member", "--json")
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["kind"] != "topic" || doc["topic_kind"] != "type" || len(doc["typed_documents"].([]any)) != 2 {
		t.Error(out)
	}
	if code, _, errOut := run("topic", "-C", dir, "nowhere"); code != ExitFailed || !strings.Contains(errOut, "unknown topic") {
		t.Error(code, errOut)
	}
	code, out, _ = run("questions", "-C", dir)
	if code != ExitOK || !strings.Contains(out, "no package of the library") {
		t.Error(code, out)
	}
	applied := initialized(t)
	copyFromLibrary(t, applied, "packages/core", "packages/software")
	_, out, _ = run("questions", "-C", applied)
	if !strings.HasPrefix(out, "## core\n\n1. ") || !strings.Contains(out, "\n## software\n\n1. Who may call each operation") {
		t.Error(out)
	}
	if code, out, _ := run("topic", "-C", applied, "--json"); code != ExitOK || !strings.Contains(out, `"topic_kind": "spec"`) {
		t.Error(code, out)
	}
}
