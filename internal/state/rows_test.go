package state

import (
	"strings"
	"testing"
)

const hashA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestParseAndFormatState(t *testing.T) {
	data := "\xef\xbb\xbf" +
		`{"doc":"b.md","verdict":"ok","source":"mark","inputs":{"self":"` + hashA + `"},"evidence":[],"reason":"","later":1}` + "\r\n" +
		"\n" +
		`{"doc":"a.md","verdict":"fail","source":"move","inputs":{"self":"` + hashA + `","file:src/a.go":"` + hashA + `"},"evidence":["src/a.go:1"],"reason":"why"}` + "\n" +
		`{"doc":"b.md","verdict":"ok","source":"mark","inputs":{"self":"` + hashA + `"},"evidence":[],"reason":""}` + "\n"
	rows, problems := ParseState([]byte(data))
	if len(rows) != 3 || len(problems) != 0 {
		t.Fatalf("%v %v", rows, problems)
	}
	if got := rows[1].Files(); strings.Join(got, " ") != "src/a.go" {
		t.Error(got)
	}
	want := `{"doc":"a.md","verdict":"fail","source":"move","inputs":{"file:src/a.go":"` + hashA + `","self":"` + hashA + `"},"evidence":["src/a.go:1"],"reason":"why"}` + "\n" +
		`{"doc":"b.md","verdict":"ok","source":"mark","inputs":{"self":"` + hashA + `"},"evidence":[],"reason":""}` + "\n"
	if got := string(FormatState(rows)); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestUnreadableLines(t *testing.T) {
	row := func(fields string) string {
		return `{"doc":"a.md","verdict":"ok","source":"mark","inputs":{"self":"` + hashA + `"}` + fields + "}\n"
	}
	for _, tc := range []struct{ line, want string }{
		{"not json\n", "not a JSON object"},
		{`{"doc":"a.md"} {}` + "\n", "more than one JSON value"},
		{`{"doc":"../a.md","verdict":"ok","source":"mark","inputs":{"self":"` + hashA + `"}}` + "\n", "not a path below the spec folder"},
		{`{"doc":"a.md","verdict":"good","source":"mark","inputs":{"self":"` + hashA + `"}}` + "\n", `verdict "good"`},
		{`{"doc":"a.md","verdict":"ok","source":"hand","inputs":{"self":"` + hashA + `"}}` + "\n", `source "hand"`},
		{`{"doc":"a.md","verdict":"ok","source":"mark","inputs":{}}` + "\n", "no self input"},
		{row(`,"evidence":"x"`), "not a JSON object"},
		{`{"doc":"a.md","verdict":"ok","source":"mark","inputs":{"self":"sha256:xyz"}}` + "\n", "no sha256 hash"},
		{`{"doc":"a.md","verdict":"ok","source":"mark","inputs":{"self":"` + hashA + `","blob:x":"` + hashA + `"}}` + "\n", "unknown input blob:x"},
		{`{"doc":"a.md","verdict":"ok","source":"mark","inputs":{"self":"` + hashA + `","file:../x":"` + hashA + `"}}` + "\n", "not a clean relative path"},
	} {
		rows, problems := ParseState([]byte(row("") + tc.line))
		if len(rows) != 1 || len(problems) != 1 || problems[0].Line != 2 || !strings.Contains(problems[0].String(), tc.want) ||
			!strings.HasPrefix(problems[0].String(), ".markitect/state.jsonl:2: state.unreadable: ") {
			t.Errorf("%q: %v %v", tc.line, rows, problems)
		}
	}
}

func TestParseEvidence(t *testing.T) {
	for arg, want := range map[string]string{
		"src/a.go":                  "src/a.go",
		`src\a.go:12`:               "src/a.go:12",
		"src/a.go:3-9":              "src/a.go:3-9",
		"src/a.go:3:7":              "src/a.go:3:7",
		"tests/a_test.go:4 (TestA)": "tests/a_test.go:4",
		"./src/a.go":                "src/a.go",
	} {
		if _, got, ok := ParseEvidence(arg); !ok || got != want {
			t.Errorf("%q: got %q %v", arg, got, ok)
		}
	}
	for _, arg := range []string{"", "/abs.go", "../up.go", "C:/x.go", " (note)"} {
		if _, got, ok := ParseEvidence(arg); ok {
			t.Errorf("%q: took %q", arg, got)
		}
	}
}
