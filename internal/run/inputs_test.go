package run_test

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Evidence is path:line or path; a test name in parentheses after it is
// dropped. The file part is what counts as an input.
func TestParseEvidence(t *testing.T) {
	for text, want := range map[string]run.Evidence{
		"src/fees.go":                        {Text: "src/fees.go", File: "src/fees.go"},
		"src/fees.go:40":                     {Text: "src/fees.go:40", File: "src/fees.go"},
		"src/fees.go:40-52":                  {Text: "src/fees.go:40-52", File: "src/fees.go"},
		"src/fees.go:40:7":                   {Text: "src/fees.go:40:7", File: "src/fees.go"},
		"tests/fees_test.go:12 (TestWaiver)": {Text: "tests/fees_test.go:12", File: "tests/fees_test.go"},
		"./src\\fees.go:3":                   {Text: "src/fees.go:3", File: "src/fees.go"},
		"docs/user  guide.md:4":              {Text: "docs/user guide.md:4", File: "docs/user guide.md"},
		"docs/a:b.md":                        {},
		"/etc/passwd":                        {},
		"C:/Users/x/fees.go":                 {},
		"../outside.go":                      {},
		"src/../../outside.go":               {},
		"":                                   {},
	} {
		got, err := run.ParseEvidence(text)
		if want.File == "" {
			if err == nil {
				t.Errorf("%q was taken: %+v", text, got)
			}
			continue
		}
		if err != nil || got != want {
			t.Errorf("%q: %+v %v, want %+v", text, got, err, want)
		}
	}
}
