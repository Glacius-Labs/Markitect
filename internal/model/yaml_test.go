package model

import (
	"strings"
	"testing"
)

func TestParseYAMLScalarsUseLanguageGrammar(t *testing.T) {
	got, problems, complete := parseYAML([]byte("values:\n  - true\n  - false\n  - True\n  - yes\n  - 2026-10-15\n  - 42\n  - 42.0\n  - 4.2e1\n  - \"42\"\n  - -42.0\n  - -1.25e-3\n  - 1e100000000000000000000000000000000000\n"), 16)
	if !complete || len(problems) != 0 {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
	fields, _ := got.Fields()
	items, _ := fields[0].Value.Items()
	wantKinds := []ValueKind{BooleanValue, BooleanValue, StringValue, StringValue, StringValue, IntegerValue, NumberValue, NumberValue, StringValue, NumberValue, NumberValue, NumberValue}
	if len(items) != len(wantKinds) {
		t.Fatalf("got %d values", len(items))
	}
	for i, want := range wantKinds {
		if items[i].Kind() != want {
			t.Errorf("value %d kind=%s, want %s", i, items[i].Kind(), want)
		}
	}
	for _, i := range []int{6, 7} {
		d, ok := items[i].Number()
		if !ok || d != (Decimal{Coefficient: "42", Exponent: "0"}) {
			t.Errorf("value %d decimal=%+v", i, d)
		}
	}
	checks := map[int]Decimal{
		9:  {Coefficient: "-42", Exponent: "0"},
		10: {Coefficient: "-125", Exponent: "-5"},
		11: {Coefficient: "1", Exponent: "100000000000000000000000000000000000"},
	}
	for i, want := range checks {
		if got, ok := items[i].Number(); !ok || got != want {
			t.Errorf("value %d decimal=%+v, want %+v", i, got, want)
		}
	}
}

func TestParseYAMLRejectsForbiddenAndAmbiguousForms(t *testing.T) {
	cases := []struct{ name, source, code string }{
		{"duplicate", "a: 1\na: 2\n", "yaml.duplicate_key"},
		{"null", "a: null\n", "yaml.null"},
		{"empty", "a:\n", "yaml.null"},
		{"quoted null key is text", "\"null\": x\n", ""},
		{"alias", "a: &x 1\nb: *x\n", "yaml.anchor_alias"},
		{"merge", "a: 1\nb:\n  <<: {c: 2}\n", "yaml.merge_key"},
		{"tag", "a: !!str x\n", "yaml.explicit_tag"},
		{"non-string key", "true: value\n", "yaml.key_type"},
		{"nonfinite .nan", "a: .NaN\n", "yaml.non_finite"},
		{"nonfinite .inf", "a: .INF\n", "yaml.non_finite"},
		{"nonfinite positive .inf", "a: +.inf\n", "yaml.non_finite"},
		{"nonfinite negative .inf", "a: -.INF\n", "yaml.non_finite"},
		{"nonfinite nan", "a: nAn\n", "yaml.non_finite"},
		{"nonfinite inf", "a: iNF\n", "yaml.non_finite"},
		{"nonfinite signed inf", "a: +iNF\n", "yaml.non_finite"},
		{"nonfinite infinity", "a: INFINITY\n", "yaml.non_finite"},
		{"nonfinite signed infinity", "a: -INFINITY\n", "yaml.non_finite"},
		{"quoted nonfinite is string", "a: '.Inf'\n", ""},
		{"double sign plus minus inf is text", "a: +-Inf\n", ""},
		{"double sign minus plus infinity is text", "a: -+Infinity\n", ""},
		{"double plus inf is text", "a: ++Inf\n", ""},
		{"double minus infinity is text", "a: --Infinity\n", ""},
		{"signed nan is ordinary text", "a: +NaN\n", ""},
		{"second document", "a: 1\n---\nb: 2\n", "yaml.documents"},
		{"wrong directive", "%YAML 1.1\na: 1\n", "yaml.directive"},
		{"malformed directive comment", "%YAML 1.2#comment\na: 1\n", "yaml.directive"},
		{"tag directive", "%TAG !e! tag:example.com,2026:\na: 1\n", "yaml.directive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, problems, complete := parseYAML([]byte(tc.source), 16)
			if !complete {
				t.Fatal("unexpected incomplete parse")
			}
			if tc.code == "" {
				if len(problems) != 0 {
					t.Fatalf("problems=%+v", problems)
				}
				return
			}
			for _, p := range problems {
				if p.code == tc.code {
					return
				}
			}
			t.Fatalf("wanted %s, got %+v", tc.code, problems)
		})
	}
}

func TestParseYAMLDoubleSignsAreText(t *testing.T) {
	for _, token := range []string{"+-Inf", "-+Infinity", "++Inf", "--Infinity"} {
		got, problems, complete := parseYAML([]byte("a: "+token+"\n"), 8)
		if !complete || len(problems) != 0 {
			t.Fatalf("token %q: complete=%v problems=%+v", token, complete, problems)
		}
		fields, _ := got.Fields()
		text, ok := fields[0].Value.Text()
		if !ok || fields[0].Value.Kind() != StringValue || text != token {
			t.Errorf("token %q decoded as kind=%s text=%q", token, fields[0].Value.Kind(), text)
		}
	}
}

func TestParseYAMLDirective12AndDocumentMarker(t *testing.T) {
	got, problems, complete := parseYAML([]byte("%YAML 1.2\n---\na: value # tail\n"), 8)
	if !complete || len(problems) != 0 {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
	fields, _ := got.Fields()
	if fields[0].Name != "a" {
		t.Fatalf("fields=%+v", fields)
	}
}

func TestParseYAMLDirectiveAfterBOMAndComments(t *testing.T) {
	got, problems, complete := parseYAML([]byte("\xef\xbb\xbf# leading comment\n%YAML 1.2\n---\na: value\n"), 8)
	if !complete || len(problems) != 0 {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
	fields, _ := got.Fields()
	if len(fields) != 1 || fields[0].Name != "a" {
		t.Fatalf("fields=%+v", fields)
	}
}

func TestParseYAMLDirectiveImmediatelyAfterBOM(t *testing.T) {
	_, problems, complete := parseYAML([]byte("\xef\xbb\xbf%YAML 1.2\n---\na: value\n"), 8)
	if !complete || len(problems) != 0 {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
}

func TestParseYAMLDirectiveLikeTextInsideMultilineScalar(t *testing.T) {
	source := "body: \"first\n%YAML 1.2\n  tail\"\n"
	got, problems, complete := parseYAML([]byte(source), 8)
	if !complete || len(problems) != 0 {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
	fields, _ := got.Fields()
	text, ok := fields[0].Value.Text()
	if !ok || !strings.Contains(text, "%YAML 1.2") {
		t.Fatalf("decoded text=%q, ok=%v", text, ok)
	}
}

func TestParseYAMLDepthLimitIsIncomplete(t *testing.T) {
	_, problems, complete := parseYAML([]byte("a:\n  b:\n    c: value\n"), 2)
	if complete {
		t.Fatal("expected incomplete parse")
	}
	if len(problems) != 1 || problems[0].code != "limit.yaml_depth" {
		t.Fatalf("problems=%+v", problems)
	}
}

func TestParseYAMLExactDuplicateOrderAndUTF8(t *testing.T) {
	_, problems, complete := parseYAML([]byte("a: 1\nb: 2\na: 3\n"), 8)
	if !complete || len(problems) != 1 || problems[0].code != "yaml.duplicate_key" || problems[0].position.Line != 3 {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
	_, problems, complete = parseYAML([]byte{0xff}, 8)
	if !complete || len(problems) != 1 || problems[0].code != "yaml.utf8" {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
	if problems[0].position != (Position{}) {
		t.Fatalf("invalid UTF-8 has an unsupported guessed position: %+v", problems[0].position)
	}
}

func TestParseYAMLUnicodeBeforeScalarPosition(t *testing.T) {
	got, problems, complete := parseYAML([]byte("é: x\n"), 8)
	if !complete || len(problems) != 0 {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
	fields, _ := got.Fields()
	if fields[0].Value.position != (Position{Line: 1, Column: 4}) {
		t.Fatalf("value position=%+v, want codepoint column 4", fields[0].Value.position)
	}
}

func TestParseYAMLContinuesAfterInvalidKey(t *testing.T) {
	_, problems, complete := parseYAML([]byte("true: null\n"), 8)
	if !complete {
		t.Fatal("expected completed examination")
	}
	seenKey, seenNull := false, false
	for _, problem := range problems {
		seenKey = seenKey || problem.code == "yaml.key_type"
		seenNull = seenNull || problem.code == "yaml.null"
	}
	if !seenKey || !seenNull {
		t.Fatalf("expected both key and value diagnostics, got %+v", problems)
	}
}

func TestParseYAMLMalformedEscapeDoesNotInventSurrogate(t *testing.T) {
	_, problems, complete := parseYAML([]byte(`a: "\uD800"`+"\n"), 8)
	if !complete || len(problems) == 0 || !strings.HasPrefix(problems[0].code, "yaml.") {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
}

func TestParseYAMLSyntaxDoesNotInventPosition(t *testing.T) {
	_, problems, complete := parseYAML([]byte("a: [\n"), 8)
	if !complete || len(problems) != 1 || problems[0].code != "yaml.syntax" {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
	if problems[0].position != (Position{}) {
		t.Fatalf("guessed parser position: %+v", problems[0].position)
	}
}

func TestParseYAMLNativeParserDepthLimitIsIncomplete(t *testing.T) {
	depth := 10010
	source := "a: " + strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth) + "\n"
	_, problems, complete := parseYAML([]byte(source), depth+2)
	if complete || len(problems) != 1 || problems[0].code != "limit.yaml_depth" {
		t.Fatalf("complete=%v problems=%+v", complete, problems)
	}
}
