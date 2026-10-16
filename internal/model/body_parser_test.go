package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestBodyMarkdownProtectionAndPrecedence(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		wanted     []string
	}{
		{"plain repeats", "[[api]] [[field]] [[api]]", []string{"api", "field", "api"}},
		{"format and containers", "# [[api]]\n\n> - **[[field]]**\n\n- [[api]]", []string{"api", "field", "api"}},
		{"model precedence", "[[api]](text)", []string{"api"}},
		{"unclosed bracket", "[ [[api]]", []string{"api"}},
		{"escaped opener", `\[[api]]`, nil},
		{"escaped backslash", `\\[[api]]`, []string{"api"}},
		{"inline code", "`[[broken` [[api]]", []string{"api"}},
		{"code fence", "```markdown\n[[broken\n```\n\n[[api]]", []string{"api"}},
		{"tilde fence", "~~~\n[[broken\n~~~\n\n[[api]]", []string{"api"}},
		{"indented code", "    [[broken\n\n[[api]]", []string{"api"}},
		{"ordinary link", "[label [[api]]](url) [[field]]", []string{"field"}},
		{"link destination and title", `[label](<url/[[broken> "[[broken") [[api]]`, []string{"api"}},
		{"image destination and title", `![label](<url/[[broken> "[[broken") [[api]]`, []string{"api"}},
		{"image", "![label [[api]]](url) [[field]]", []string{"field"}},
		{"protected bad label", "[label [[bad target]]](url) [[api]]", []string{"api"}},
		{"protected nested label", "[label [[api|[[nested]]]]](url) [[api]]", []string{"api"}},
		{"protected multiline label", "[label [[broken\nstill label]]](url) [[api]]", []string{"api"}},
		{"protected multiline image", "![label [[broken\nstill label]]](url) [[api]]", []string{"api"}},
		{"protected bad escape", `[label [[api|a\q]]](url) [[api]]`, []string{"api"}},
		{"protected lone close", "[label [[api|a] text]](url) [[api]]", []string{"api"}},
		{"protected malformed image", "![label [[bad target]]](url) [[api]]", []string{"api"}},
		{"reference link", "[label [[bad target]]][ref]\n\n[ref]: /url \"[[broken\"\n\n[[api]]", []string{"api"}},
		{"reference definition", "[ref]: /url \"[[broken\"\n\n[[api]]", []string{"api"}},
		{"autolink", "<https://example.org/[[bad]]> [[api]]", []string{"api"}},
		{"HTML inline tag", "a <span title='[[broken'>[[api]]</span> [[field]]", []string{"api", "field"}},
		{"HTML block", "<div>\n[[broken\n</div>\n\n[[api]]", []string{"api"}},
		{"HTML script", "<script>\n[[broken\n</script>\n\n[[api]]", []string{"api"}},
		{"HTML processing instruction", "<?test [[broken ?>\n\n[[api]]", []string{"api"}},
		{"HTML declaration", "<!DOCTYPE [[broken>\n\n[[api]]", []string{"api"}},
		{"HTML CDATA", "<![CDATA[ [[broken ]]>\n\n[[api]]", []string{"api"}},
		{"HTML custom block", "<custom>\n[[broken\n\n[[api]]", []string{"api"}},
		{"HTML comment", "<!-- [[broken --> [[api]]", nil},
		{"HTML inline comment", "a <!-- [[broken --> [[api]]", []string{"api"}},
		{"entity is not opener", "&#91;&#91;api]] &lbrack;&lbrack;api]] [[api]]", []string{"api"}},
		{"isolated closer", "]] [[api]]", []string{"api"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBodySyntax(tt.body, DefaultLimits())
			var actual []string
			for _, occurrence := range got.occurrences {
				actual = append(actual, occurrence.written)
				if tt.body[occurrence.span.StartByte:occurrence.span.EndByte] != "[["+occurrence.written+"]]" {
					t.Fatalf("wrong original body span: %+v", occurrence)
				}
			}
			if len(got.problems) != 0 || !reflect.DeepEqual(actual, tt.wanted) {
				t.Fatalf("body context differs: targets=%q wanted=%q problems=%+v", actual, tt.wanted, got.problems)
			}
		})
	}
}

func TestBodyNestingExactBoundaries(t *testing.T) {
	for _, tt := range []struct {
		body    string
		limit   int
		measure string
	}{
		{"[[api]]", 2, "constructed AST depth"},
		{strings.Repeat("[label ", 8) + "text", 8, "pending bracket labels"},
	} {
		limits := DefaultLimits()
		limits.MarkdownNesting = tt.limit
		if got := parseBodySyntax(tt.body, limits); len(got.problems) != 0 {
			t.Fatalf("at-budget input refused: %+v", got)
		}
		limits.MarkdownNesting--
		got := parseBodySyntax(tt.body, limits)
		if len(got.problems) != 1 || !got.problems[0].incomplete || !strings.Contains(got.problems[0].message, tt.measure) {
			t.Fatalf("above-budget input did not use declared measure: %+v", got)
		}
	}
	limits := DefaultLimits()
	limits.MarkdownNesting = 1
	got := parseBodySyntax("> > text", limits)
	if len(got.problems) != 1 || !strings.Contains(got.problems[0].message, "block AST depth") {
		t.Fatalf("block guard did not refuse before recursive inline walk: %+v", got)
	}
}

func TestBodyCompletedBlocksSurviveLaterInlineRefusal(t *testing.T) {
	limits := DefaultLimits()
	limits.MarkdownNesting = 8
	for _, first := range []string{"[[bad target]]", "[label [[bad target]]](url)", "[[api]]"} {
		got := parseBodySyntax(first+"\n\n"+strings.Repeat("[label ", 9)+"text", limits)
		var syntax, refusal bool
		for _, problem := range got.problems {
			syntax = syntax || problem.code == "body.syntax"
			refusal = refusal || problem.code == "limit.markdown_nesting"
		}
		wanted := 0
		if first == "[[api]]" {
			wanted = 1
		}
		if !refusal || syntax != (first == "[[bad target]]") || len(got.occurrences) != wanted {
			t.Fatalf("completed block proof lost or protected error invented: %q %+v", first, got)
		}
	}
}

func TestBodyGrammarAndLiteralDisplay(t *testing.T) {
	for _, body := range []string{
		"[[]]", "[[ api ]]", "[[api|]]", "[[api| \t\u3000]]", "[[api", "[[api|A|B]]", "[[api|[[other]]]]",
		"[[api|A\nB]]", "[[api|A\rB]]", "[[api|A\u0085B]]", "[[api|A\u2028B]]", "[[api|A\u2029B]]",
		`[[api|\q]]`, `[[api|\0]]`, `[[a\pi]]`, "[[api.field.more]]", "[[u\u0308ber]]", "[[api|a]b]]",
		"[label [[broken](url) [[api]]", "[label [[broken\nstill label](url) [[api]]", "[text](url) [[bad", "[[bad target]](url)",
	} {
		got := parseBodySyntax(body, DefaultLimits())
		if len(got.problems) == 0 || got.problems[0].code != "body.syntax" || got.problems[0].incomplete {
			t.Fatalf("active syntax violation accepted or treated as budget: %q %+v", body, got)
		}
	}
	for _, tt := range []struct{ body, display string }{
		{`[[api|A\| B \[C\] \\]]`, `A| B [C] \`},
		{"[[api|A\x00B]]", "A\x00B"},
		{"[[api|\x00]]", "\x00"},
		{"[[api|\ufffd\x00\ufffd]]", "\ufffd\x00\ufffd"},
		{"[[api|*literal* &amp; `code`]]", "*literal* &amp; `code`"},
	} {
		got := parseBodySyntax(tt.body, DefaultLimits())
		if len(got.problems) != 0 || len(got.occurrences) != 1 || got.occurrences[0].display == nil || *got.occurrences[0].display != tt.display || got.occurrences[0].span.EndByte != len(tt.body) {
			t.Fatalf("display text changed: %q %+v", tt.body, got)
		}
	}
	body := "📄\x00� [[api|A\x00B]]\r\n> [[api|�]]"
	got := parseBodySyntax(body, DefaultLimits())
	if len(got.problems) != 0 || len(got.occurrences) != 2 || got.occurrences[0].span.StartByte != strings.Index(body, "[[") || got.occurrences[1].span.EndByte != len(body) {
		t.Fatalf("NUL/UTF-8/CRLF changed raw body offsets: %+v", got)
	}
}

func TestBodyMalformedCandidateDoesNotInstallNestedAtoms(t *testing.T) {
	for _, body := range []string{"[[api|[[other]]]] [[api]]", "[[api|x [[other]] y]] [[api]]", "[[api [[other]]]] [[api]]", "[[api|[[other]] [[third]]]] [[api]]", "[[api [[other]] [[third]]]] [[api]]", `[[api|[[other]] \[literal\] [[third]]]] [[api]]`} {
		got := parseBodySyntax(body, DefaultLimits())
		if len(got.problems) != 1 || got.problems[0].code != "body.syntax" || len(got.occurrences) != 1 || got.occurrences[0].written != "api" || got.occurrences[0].span.StartByte != strings.LastIndex(body, "[[api]]") {
			t.Fatalf("invalid nested payload became an independent link: %q %+v", body, got)
		}
	}
}

func TestBodyBudgetsAreExplicitToolRefusals(t *testing.T) {
	limits := DefaultLimits()
	limits.BodyBytes = len("[[api|\x00]]")
	if got := parseBodySyntax("[[api|\x00]]", limits); len(got.problems) != 0 {
		t.Fatalf("NUL view expansion counted against original body bytes: %+v", got)
	}
	limits.BodyBytes--
	if got := parseBodySyntax("[[api|\x00]]", limits); len(got.problems) != 1 || got.problems[0].code != "limit.body_bytes" || !got.problems[0].incomplete {
		t.Fatalf("body byte budget refusal lost: %+v", got)
	}
	limits = DefaultLimits()
	limits.MarkdownNesting = 8
	for _, body := range []string{strings.Repeat("> ", 12) + "[[api]]", strings.Repeat("[label ", 12) + "text", strings.Repeat("![", 12) + "text" + strings.Repeat("](url)", 12), strings.Repeat("*", 40) + "text" + strings.Repeat("*", 40)} {
		got := parseBodySyntax(body, limits)
		if len(got.problems) == 0 || got.problems[0].code != "limit.markdown_nesting" || !got.problems[0].incomplete {
			t.Fatalf("Markdown nesting was silently accepted/truncated: %q %+v", body, got)
		}
	}
}
