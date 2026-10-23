package authoring

import (
	"fmt"
	"strings"
	"testing"
)

func render(lines []DiffLine) string {
	var b strings.Builder
	for _, l := range lines {
		if l.Gap > 0 {
			fmt.Fprintf(&b, "... %d\n", l.Gap)
			continue
		}
		fmt.Fprintf(&b, "%s %d %d %s\n", l.Op, l.Old, l.New, l.Text)
	}
	return b.String()
}

func TestDiffShowsReplacedLinesWithCutContext(t *testing.T) {
	var a, b []string
	for i := 1; i <= 20; i++ {
		a = append(a, fmt.Sprintf("line %d", i))
	}
	b = append(b, a...)
	b[9] = "line ten"
	b = append(b, "line 21")
	got := render(diffLines(a, b))
	want := `... 6
= 7 7 line 7
= 8 8 line 8
= 9 9 line 9
- 10 0 line 10
+ 0 10 line ten
= 11 11 line 11
= 12 12 line 12
= 13 13 line 13
... 4
= 18 18 line 18
= 19 19 line 19
= 20 20 line 20
+ 0 21 line 21
`
	if got != want {
		t.Fatalf("diff =\n%s\nwant\n%s", got, want)
	}
}

func TestDiffOfAddedAndRemovedFiles(t *testing.T) {
	if got := render(diffLines(nil, splitLines("a\nb\n"))); got != "+ 0 1 a\n+ 0 2 b\n" {
		t.Fatalf("added file diff =\n%s", got)
	}
	if got := render(diffLines(splitLines("a\n"), nil)); got != "- 1 0 a\n" {
		t.Fatalf("removed file diff =\n%s", got)
	}
	if got := diffLines(splitLines("same\n"), splitLines("same\n")); len(got) != 0 {
		t.Fatalf("an unchanged file has a difference: %+v", got)
	}
}
