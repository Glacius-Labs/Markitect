package project

import (
	"strings"
	"testing"
)

// A Definition nothing links and no file realizes is unused. A thing with
// parts counts as one, and a Kind may declare that no file realizes its
// Definitions.
func TestUnusedCountsAThingAsOneAndLeavesOutWhatIsNotRealized(t *testing.T) {
	def := func(kind, title, body string) string {
		return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
	}
	r := memRepo{files: map[string]string{
		ManifestFile:            "libraries:\n  std: 2.1.0\n",
		LockFile:                stdLock(t),
		SpecDir + "/project.md": def("std/Project", "Demo", "A demo."),
		// Not realized by its Kind: an Exclusion and an Assumption.
		SpecDir + "/purpose/exclusions/payment.md":      def("std/Exclusion", "Payment", "Payment is left out."),
		SpecDir + "/context/assumptions/readers.md":     def("std/Assumption", "Readers", "People read.\n\n**Ensured by:** them.\n\n**Therefore not built:** reading aloud."),
		SpecDir + "/rules/alone.md":                     def("std/Rule", "Alone", "Nothing names this."),
		SpecDir + "/rules/fees/fees.md":                 def("std/Rule", "Fees", "A fee."),
		SpecDir + "/rules/fees/per-day.md":              def("std/Rule", "Per day", "`1` euro."),
		SpecDir + "/rules/late/late.md":                 def("std/Rule", "Late", "Late is late."),
		SpecDir + "/rules/late/by-a-day.md":             def("std/Rule", "By a day", "A day counts."),
		SpecDir + "/rules/linked/linked.md":             def("std/Rule", "Linked", "Its part links it: [[rules/linked/inside]]."),
		SpecDir + "/rules/linked/inside.md":             def("std/Rule", "Inside", "See [[rules/linked]]."),
		SpecDir + "/rules/from-outside/from-outside.md": def("std/Rule", "From outside", "A whole."),
		SpecDir + "/rules/from-outside/part.md":         def("std/Rule", "Part", "A part that [[rules/user]] links."),
		SpecDir + "/rules/user.md":                      def("std/Rule", "User", "Uses [[rules/from-outside/part]]."),
		// A thing whose part is a thing of its own, which a file realizes: the
		// outer thing is used through it. One beside it that nothing names is not.
		SpecDir + "/rules/outer/outer.md":       def("std/Rule", "Outer", "The outer thing."),
		SpecDir + "/rules/outer/inner/inner.md": def("std/Rule", "Inner", "A part with a part."),
		SpecDir + "/rules/outer/inner/leaf.md":  def("std/Rule", "Leaf", "Its part."),
		// Three levels, with a row that names only the innermost thing.
		SpecDir + "/rules/top/top.md":           def("std/Rule", "Top", "The top."),
		SpecDir + "/rules/top/mid/mid.md":       def("std/Rule", "Mid", "The middle."),
		SpecDir + "/rules/top/mid/low/low.md":   def("std/Rule", "Low", "The lowest."),
		SpecDir + "/rules/top/mid/low/leaf.md":  def("std/Rule", "Leaf", "Its part."),
		SpecDir + "/rules/outer/other/other.md": def("std/Rule", "Other", "A second part with a part."),
		SpecDir + "/rules/outer/other/leaf.md":  def("std/Rule", "Leaf", "Its part."),
		RootPage:                                "| File | Realizes | Why |\n|---|---|---|\n| fees.go | [[rules/fees/per-day]] [[rules/outer/inner]] [[rules/top/mid/low]] | Rows that name a part |\n| user.go | [[rules/user]] | User |\n",
		"fees.go":                               "package x\n",
		"user.go":                               "package x\n",
	}}
	p := load(t, r)
	var unused []string
	for _, d := range p.Errors {
		if d.Code != CodeUnused {
			t.Fatalf("unexpected error: %v", d)
		}
		unused = append(unused, d.Message)
	}
	// alone: nothing names it. late: nothing names the thing or its part.
	// linked: links inside a thing do not count.
	// outer/other: a thing of its own that nothing names, though its whole is used.
	want := "nothing links to rules/alone and no file realizes it|nothing links to rules/late/late and no file realizes it|nothing links to rules/linked/linked and no file realizes it|nothing links to rules/outer/other/other and no file realizes it"
	if got := strings.Join(unused, "|"); got != want {
		t.Fatalf("unused:\n%s\nwant:\n%s", got, want)
	}
}
