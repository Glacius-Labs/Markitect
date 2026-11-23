package lang

import "testing"

// reports is a module of four documents, a nested module and a README with the
// given Contents; other/a.md links the given targets.
func reportsFiles(contents string, links string) map[string]string {
	return map[string]string{
		"reports/README.md":       "# Reports\n\n## Contents\n" + contents,
		"reports/report-1.md":     "# Report 1\n",
		"reports/report-2.md":     "# Report 2\n",
		"reports/summary.md":      "# Summary\n",
		"reports/old/README.md":   "# Old\n",
		"reports/old/report-0.md": "# Report 0\n",
		"other/README.md":         "# Other\n",
		"other/a.md":              "# A\n\n" + links,
	}
}

func TestContentsPatterns(t *testing.T) {
	all := "[r1](../reports/report-1.md) [r2](../reports/report-2.md) [s](../reports/summary.md)\n"
	internal := func(p string) string {
		return `other/a.md:3: link.internal: "reports/` + p + `" is internal to reports: reports/README.md does not list it under Contents`
	}
	for name, tc := range map[string]struct {
		contents, links string
		want            []string
	}{
		"a star shows every document of the folder": {"- `*.md`\n", all, nil},
		"a star shows nothing below the folder": {
			"- `*.md`\n", "[x](../reports/old/report-0.md)\n",
			[]string{`other/a.md:3: link.internal: "reports/old/report-0.md" is internal to reports: reports/README.md does not list reports/old/README.md under Contents`}},
		"a partial pattern shows only its matches": {
			"- `report-*.md`\n", all, []string{internal("summary.md")}},
		"a question mark and a class": {
			"- `report-?.md` and `[s]ummary.md`\n", all, nil},
		"a pattern that matches nothing does no harm": {
			"- `archive-*.md`\n- `go test`\n", "[r](../reports/summary.md)\n", []string{internal("summary.md")}},
		"a pattern and a link together": {
			"- `report-*.md`\n- [Summary](summary.md)\n", all, nil},
		"patterns are case-sensitive": {
			"- `REPORT-*.md`\n", "[r](../reports/report-1.md)\n", []string{internal("report-1.md")}},
		"a pattern matching the README changes nothing": {
			"- `README.md`\n", "[r](../reports/report-1.md) [readme](../reports/README.md)\n", []string{internal("report-1.md")}},
		"a code span outside Contents shows nothing": {
			"- [Summary](summary.md)\n\n## Notes\nThe files are `report-*.md`.\n", all,
			[]string{internal("report-1.md"), internal("report-2.md")}},
		"a fenced block in Contents shows nothing": {
			"```\n`*.md`\n*.md\n```\n", all,
			[]string{internal("report-1.md"), internal("report-2.md"), internal("summary.md")}},
		"an indented block in Contents shows nothing": {
			"\n    *.md\n", all,
			[]string{internal("report-1.md"), internal("report-2.md"), internal("summary.md")}},
		"a pattern does not show the README of a module below": {
			"- `*`\n", "[old](../reports/old/README.md)\n",
			[]string{`other/a.md:3: link.internal: "reports/old/README.md" is internal to reports: reports/README.md does not list it under Contents`}},
	} {
		t.Run(name, func(t *testing.T) { expect(t, checkExactly(t, reportsFiles(tc.contents, tc.links)), tc.want...) })
	}
}

func TestContentsPatternMatchesTheCaseOfTheHeading(t *testing.T) {
	files := reportsFiles("- `*.md`\n", "[r](../reports/report-1.md)\n")
	files["reports/README.md"] = "# Reports\n\n## contents\n- `*.md`\n"
	expect(t, checkExactly(t, files))
}

func TestContentsPatternErrors(t *testing.T) {
	const bad = "a pattern uses the wildcards *, ? and [a-z]"
	files := reportsFiles("- `report-[1.md`\n- `report-2.md`\n- `?[`\n", "[r](../reports/report-1.md) [r2](../reports/report-2.md) [s](../reports/summary.md)\n")
	expect(t, checkExactly(t, files),
		`other/a.md:3: link.internal: "reports/report-1.md" is internal to reports: reports/README.md does not list it under Contents`,
		`other/a.md:3: link.internal: "reports/summary.md" is internal to reports: reports/README.md does not list it under Contents`,
		`reports/README.md:4: readme.pattern: bad pattern "report-[1.md": `+bad,
		`reports/README.md:6: readme.pattern: bad pattern "?[": `+bad)
	// Only the README holds patterns, and only in Contents.
	files = reportsFiles("- [Summary](summary.md)\n\n## Notes\n`[`\n", "")
	files["reports/summary.md"] = "# Summary\n\n## Contents\n`[` and `a/b`\n"
	expect(t, checkExactly(t, files))
}

// A code span with a "/" is plain code, such as a path or an address: no
// pattern, no error, and it shows nothing, also when it would be malformed.
func TestContentsCodeWithASlashIsPlainCode(t *testing.T) {
	files := reportsFiles("- [Summary](summary.md): the page at `/cart`\n- `old/*.md`, `*/*.md` and `[/`\n", "[r](../reports/report-1.md) [s](../reports/summary.md) [o](../reports/old/report-0.md)\n")
	expect(t, checkExactly(t, files),
		`other/a.md:3: link.internal: "reports/report-1.md" is internal to reports: reports/README.md does not list it under Contents`,
		`other/a.md:3: link.internal: "reports/old/report-0.md" is internal to reports: reports/README.md does not list reports/old/README.md under Contents`)
}

func TestCodeSpansOfADocument(t *testing.T) {
	d := Parse("a.md", []byte("# A\n\nText `one` and ``tw`o``.\n\n## Contents\n- ` *.md `\n- `x\ny`\n\n```\n`no`\n```\n\n    `no`\n"))
	var got []string
	for _, c := range d.CodeSpans {
		got = append(got, c.Section+"|"+c.Text)
	}
	expect(t, got, "|one", "|tw`o", "Contents|*.md", "Contents|x y")
	if d.CodeSpans[0].Line != 3 || d.CodeSpans[2].Line != 6 || d.CodeSpans[3].Line != 7 {
		t.Errorf("%+v", d.CodeSpans)
	}
}
