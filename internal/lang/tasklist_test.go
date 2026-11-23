package lang

import "testing"

// A list item that starts with a link whose text is x, X or a space is a link,
// as on GitHub; only a bracket followed by white space is a task list marker.
func TestListItemOfALinkIsNoTaskListItem(t *testing.T) {
	for _, item := range []string{"- [x](x.md)", "- [X](x.md)", "* [x](x.md) and more", "1. [x](x.md)", "- [x][ref]\n\n[ref]: x.md"} {
		d := Parse("a.md", []byte("# A\n\n"+item+"\n"))
		if len(d.Links) != 1 || d.Links[0].Target != "x.md" {
			t.Errorf("%q: links %+v", item, d.Links)
		}
		if len(d.Texts) == 0 || len(d.Texts[0].Links) != 1 {
			t.Errorf("%q: texts %+v", item, d.Texts)
		}
	}
}

// A link whose text is a space has no text to show, but it is a link.
func TestListItemOfALinkWithBlankTextIsALink(t *testing.T) {
	if d := Parse("a.md", []byte("# A\n\n- [ ](x.md)\n")); len(d.Links) != 1 {
		t.Errorf("links %+v", d.Links)
	}
}

func TestTaskListItemsStayTaskListItems(t *testing.T) {
	for item, text := range map[string]string{
		"- [x] done":          "done",
		"- [X] done":          "done",
		"- [ ] open":          "open",
		"- [x]\tdone":         "done",
		"- [x] [x](x.md)":     "x",
		"- [ ] see [x](x.md)": "see x",
	} {
		d := Parse("a.md", []byte("# A\n\n"+item+"\n"))
		if len(d.Texts) != 1 || d.Texts[0].Text != text {
			t.Errorf("%q: texts %+v", item, d.Texts)
		}
		wantLinks := 0
		if len(item) > 6 && item[len(item)-1] == ')' {
			wantLinks = 1
		}
		if len(d.Links) != wantLinks {
			t.Errorf("%q: links %+v", item, d.Links)
		}
	}
	// A marker without text stays a marker and is no link.
	if d := Parse("a.md", []byte("# A\n\n- [x]\n- [ ]\n")); len(d.Links) != 0 || len(d.Texts) != 0 {
		t.Errorf("empty items: %+v %+v", d.Links, d.Texts)
	}
}

func TestContentsListOfLinksNamedX(t *testing.T) {
	files := reportsFiles("- [x](report-1.md)\n- [X](report-2.md)\n", "[a](../reports/report-1.md) [b](../reports/report-2.md) [c](../reports/summary.md)\n")
	expect(t, checkExactly(t, files),
		`other/a.md:3: link.internal: "reports/summary.md" is internal to reports: reports/README.md does not list it under Contents`)
}
