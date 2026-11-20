package signals

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

func spec(t *testing.T, files map[string]string) *lang.Spec {
	t.Helper()
	fsys := fstest.MapFS{}
	for p, text := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(text)}
	}
	s, err := lang.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func eq(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%#v\nwant\n%#v", got, want)
	}
}

func TestStatements(t *testing.T) {
	d := lang.Parse("lending/loan.md", []byte("# Loan\n\nType: [Loan type](../types/loan.md)\n\n"+
		"A loan lasts 2.5 weeks, e.g. for a novel. It ends early if the copy is reserved! Does it renew?\n"+
		"Books, maps etc. stay \"on loan.\" Done\n\n"+
		"## Rules\n- Every member may renew once, see [renewals](renewals.md).\n- Only if possible.\n\n"+
		"| Copy | Due |\n|---|---|\n| C-1 | [today](../days/today.md) |\n\n"+
		"### Not a statement\n\n```\nNo code either.\n```\n\n<div>No HTML.</div>\n\n[r]: x.md\n"))
	var got []string
	for _, st := range Statements(d) {
		got = append(got, st.Section+"|"+strconv.Itoa(st.Line)+"|"+st.Text+"|"+strings.Join(st.Targets, ",")+"|"+
			strings.Join(st.Conditions, ",")+"/"+strings.Join(st.Vague, ",")+"/"+strings.Join(st.Strong, ","))
	}
	eq(t, got, []string{
		"|5|A loan lasts 2.5 weeks, e.g. for a novel.||//",
		"|5|It ends early if the copy is reserved!||if//",
		"|5|Does it renew?||//",
		"|6|Books, maps etc. stay \"on loan.\"||/etc/",
		"|6|Done||//",
		"Rules|9|Every member may renew once, see renewals.|lending/renewals.md|//every",
		"Rules|10|Only if possible.||only if//",
		"Rules|14|C-1 | today|days/today.md|//",
	})
}

func TestMarkers(t *testing.T) {
	// The longest phrase wins at each word: "only if" is a condition, "if possible" vague.
	c, v, s := markers(words("Only if possible, unless no one must not act as long as needed; each user-friendly etc. Renew if possible."))
	eq(t, c, []string{"only if", "unless", "as long as"})
	eq(t, v, []string{"user-friendly", "etc", "if possible"})
	eq(t, s, []string{"no one", "must not", "each"})
}

// branches is a small spec: members, branches and rules of a lending library.
var branches = map[string]string{
	"types/member.md":   "# Member\n\nA card holder.\n\n## Card\nThe number.\n\n## Branch\nThe home branch.\n",
	"types/policy.md":   "# Policy\n\nSomething that holds.\n\n## Policy\nWhat holds.\n",
	"places/README.md":  "# Places\n\nThe branches.\n",
	"places/central.md": "# Central Library\n\nThe main building.\n",
	"places/east.md":    "# East Branch\n\nA small branch.\n",
	"members/ada.md": "# Ada\n\nType: [Member](../types/member.md)\n\nShe joined early. She reads a lot.\n\n" +
		"## Card\nL-1\n\n## Branch\n[Central](../places/central.md), [East](../places/east.md), [Places](../places/README.md).\n",
	"members/grace.md": "# Grace\n\nType: [Member](../types/member.md)\n\n## Card\nL-2\n\n" +
		"## Branch\n[Central](../places/central.md), [East](../places/east.md) and [Places](../places/README.md).\n",
	"members/alan.md": "# Alan\n\nType: [Member](../types/member.md)\n\n## Card\nL-3, usually renewed if the Central Library asks.\n\n" +
		"## Branch\n[Central](../places/central.md).\n\nHe also visits the east branch sometimes.\n",
	"rules/loans.md": "# Loans\n\nType: [Policy](../types/policy.md)\n\n## Policy\nEvery member returns books to [the branches](../places/README.md) within three weeks.\n" +
		"A member may keep a book longer only if the [Central Library](../places/central.md) agrees, unless it is reserved.\n",
	"rules/fees.md": "# Fees\n\nType: [Policy](../types/policy.md)\n\n## Policy\nIf a book is late at [East](../places/east.md), a fee is due.\n",
}

func findings(a *Analysis, signal string) []string {
	var out []string
	for _, f := range a.Findings[signal] {
		out = append(out, f.String())
	}
	return out
}

func TestWeakAndStrong(t *testing.T) {
	a := Analyze(spec(t, branches), nil)
	eq(t, findings(a, Weak), []string{
		`members/alan.md:6: weak: 2 (conditions if; vague usually): "L-3, usually renewed if the Central Library asks."`,
		`rules/loans.md:7: weak: 2 (conditions only if, unless): "A member may keep a book longer only if the Central Library agrees, unless it is reserved."`,
		`members/alan.md:11: weak: 1 (vague sometimes): "He also visits the east branch sometimes."`,
		`rules/fees.md:6: weak: 1 (conditions if): "If a book is late at East, a fee is due."`,
	})
	// The subject of the strong statement is what it links to; its reach is how
	// many documents link to or mention the subject.
	eq(t, findings(a, StrongStatements), []string{
		`rules/loans.md:6: strong: every about places/README.md, which 3 documents refer to: "Every member returns books to the branches within three weeks."`,
	})
}

func TestComplexityAndSize(t *testing.T) {
	refs := References{}
	if err := ParseReferences("lib", []byte("| Type | Statements per document | Words per statement |\n|---|---|---|\n| member | 1–2 | 2-10 |\n"), refs); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for k, v := range branches {
		files[strings.Replace(k, "types/member.md", "packages/lib/member.md", 1)] = strings.ReplaceAll(v, "../types/member.md", "../packages/lib/member.md")
	}
	a := Analyze(spec(t, files), refs)
	c := findings(a, Complexity)
	eq(t, c[:4], []string{
		"rules/loans.md:7: complexity: 17 words (project median 3, max 17), 1 term, 2 conditions",
		"rules/fees.md:6: complexity: 11 words (project median 3, max 17), 1 term, 1 condition",
		"rules/loans.md:6: complexity: 10 words (project median 3, max 17), 1 term, 0 conditions",
		"members/alan.md:6: complexity: 8 words (project median 3, max 17; reference 2–10), 0 terms, 1 condition",
	})
	eq(t, len(c), len(a.Statements))
	// Size ranks by the ratio to the median of the type, then by path.
	eq(t, findings(a, Size), []string{
		"members/ada.md:1: size: 4 statements (documents of type packages/lib/member.md: median 3, range 2–4, 3 documents; reference 1–2)",
		"rules/loans.md:1: size: 2 statements (documents of type types/policy.md: median 1.5, range 1–2, 2 documents)",
		"members/alan.md:1: size: 3 statements (documents of type packages/lib/member.md: median 3, range 2–4, 3 documents; reference 1–2)",
		"members/grace.md:1: size: 2 statements (documents of type packages/lib/member.md: median 3, range 2–4, 3 documents; reference 1–2)",
		"rules/fees.md:1: size: 1 statement (documents of type types/policy.md: median 1.5, range 1–2, 2 documents)",
	})
}

func TestStructureSignals(t *testing.T) {
	a := Analyze(spec(t, branches), nil)
	eq(t, findings(a, TextBeforeSections), []string{"members/ada.md:5: text-before-sections: 2 statements before the first section"})
	eq(t, findings(a, LinkGroup), []string{
		`members/ada.md:10: link-group: section "Branch" links the same 3 documents in 2 documents: members/ada.md, members/grace.md (links: places/README.md, places/central.md, places/east.md)`,
	})
	eq(t, findings(a, ScatteredRules), []string{
		"places/central.md:1: scattered-rules: conditional statements in 2 other documents refer to it: members/alan.md, rules/loans.md",
		"places/east.md:1: scattered-rules: conditional statements in 1 other document refer to it: rules/fees.md",
	})
	// Alan links the Central Library elsewhere, so only the East Branch is an unlinked mention.
	eq(t, findings(a, UnlinkedMention), []string{
		`members/alan.md:11: unlinked-mention: "East Branch" (places/east.md) 1 time, never linked`,
	})
}

func TestTopic(t *testing.T) {
	a := Analyze(spec(t, branches), nil)
	topic, ok := a.Topic("places/east")
	if !ok || topic.Kind != "document" {
		t.Fatal(topic)
	}
	var linked, strong, mentions []string
	for _, st := range topic.LinkedFrom {
		linked = append(linked, st.Doc+":"+strconv.Itoa(st.Line))
	}
	for _, st := range topic.Strong {
		strong = append(strong, st.Doc+":"+strconv.Itoa(st.Line))
	}
	for _, st := range topic.Mentions {
		mentions = append(mentions, st.Doc+":"+strconv.Itoa(st.Line))
	}
	// The link to places/README.md counts, so the rule for every branch shows.
	eq(t, linked, []string{"members/ada.md:11", "members/grace.md:9", "rules/fees.md:6", "rules/loans.md:6"})
	eq(t, strong, []string{"rules/loans.md:6"})
	eq(t, mentions, []string{"members/alan.md:11"})
	if len(topic.Findings[ScatteredRules]) != 1 || len(topic.Findings[LinkGroup]) != 1 {
		t.Error(topic.Findings)
	}
	typ, _ := a.Topic("types/member.md")
	eq(t, typ.Kind, "type")
	eq(t, typ.Documents, []string{"members/ada.md", "members/alan.md", "members/grace.md"})
	folder, ok := a.Topic(`places\`)
	if !ok || folder.Kind != "folder" {
		t.Fatal(folder)
	}
	eq(t, folder.Docs, []string{"places/README.md", "places/central.md", "places/east.md"})
	if _, ok := a.Topic("nowhere"); ok {
		t.Error("an unknown topic")
	}
}

func TestReferences(t *testing.T) {
	refs := References{}
	if err := ParseReferences("software", []byte("# Values\n\n| Type | Statements per document | Words per statement |\n|---|---|---|\n| operation | 1–5 | 5–25 |\n"), refs); err != nil {
		t.Fatal(err)
	}
	r, ok := refs.For("packages/software/operation.md")
	eq(t, ok, true)
	eq(t, r, Reference{Statements: [2]int{1, 5}, Words: [2]int{5, 25}})
	if _, ok := refs.For("types/operation.md"); ok {
		t.Error("a project's own type has a reference")
	}
	if err := ParseReferences("x", []byte("| a | 5–1 | 1–2 |\n"), refs); err == nil {
		t.Error("a reversed range")
	}
}
