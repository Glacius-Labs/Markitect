package model

import (
	"strings"
	"testing"
)

func TestCompileEmptyTitleIsAnErrorNotACrash(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"empty.md": def("std/Rule", "#\n"),
		"later.md": def("std/Rule", "# Title\n\n#\n"),
	})
	expect(t, ds, "definition.title", "empty.md:4: definition.title: the title heading is empty")
	expect(t, ds, "definition.title", "later.md")
}

func TestLinksWithWrappedOrCodeLabelsAreChecked(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"project.md": def("std/Rule", "# Project\n\nSee [[b/x|a label that\nwraps]] and [[b/y|the `y` thing]] and [[b/z#Part|<em>z</em>]].\n"),
		"b/x.md":     def("std/Rule", "# X\n"),
		"b/y.md":     def("std/Rule", "# Y\n"),
		"b/z.md":     def("std/Rule", "# Z\n"),
	})
	for _, target := range []string{"b/x inside", "b/y inside", "b/z inside"} {
		expect(t, ds, "link.downward", target)
	}
}

func TestEscapedLabelBarInTable(t *testing.T) {
	m, ds := compile(t, map[string]string{
		"a.md": def("std/Rule", "# A\n\n| Term | See |\n|---|---|\n| b | [[b\\|the b]] |\n"),
		"b.md": def("std/Rule", "# B\n"),
	})
	clean(t, ds)
	if links := m.Definitions["a"].Links; len(links) != 2 || links[1].Target != "b" {
		t.Fatalf("links = %+v", links)
	}
}

func TestKindFileAtRootIsReserved(t *testing.T) {
	_, ds := compile(t, map[string]string{"Kind.md": def("Kind", "# My Kind\n")})
	expect(t, ds, "model.reserved", "Kind.md: model.reserved: Kind is already the ID of Core's Kind")
}

func TestBrokenFrontMatterReportsOnce(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"broken.md": "---\nkind: std/Rule\n# never closed\n",
		"user.md":   def("std/Rule", "# User\n\nLinks [[broken]].\n"),
	})
	if len(ds) != 1 || ds[0].Code != "definition.front_matter" {
		t.Fatalf("one broken file must give one error, got:\n%v", ds)
	}
}

func TestKindPathMayEndInMd(t *testing.T) {
	m, ds := compile(t, map[string]string{
		"Medium.md": "---\nkind: Kind.md\n---\n# Medium\n",
		"dvd.md":    def("Medium.md", "# DVD\n"),
	})
	clean(t, ds)
	if m.Definitions["dvd"].Kind != "Medium" {
		t.Fatalf("kind = %q", m.Definitions["dvd"].Kind)
	}
}

func TestByteOrderMarkIsIgnored(t *testing.T) {
	_, ds := compile(t, map[string]string{"a.md": "\xef\xbb\xbf" + def("std/Rule", "# A\n")})
	clean(t, ds)
}

func TestMessagesAreOneLine(t *testing.T) {
	_, ds := compile(t, map[string]string{"a.md": "# No front matter\n"})
	_, more := ParseManifest("manifest.yaml", []byte("other: 1\n"))
	for _, d := range append(ds, more...) {
		if strings.Contains(d.Message, "\n") {
			t.Errorf("message spans lines: %q", d.Message)
		}
	}
}
