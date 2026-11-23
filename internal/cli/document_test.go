package cli

import "testing"

func TestDocumentPath(t *testing.T) {
	for arg, want := range map[string]string{
		"lending/loan": "lending/loan.md", "lending/loan.md": "lending/loan.md", `lending\loan`: "lending/loan.md",
		"./a/../b.MD": "b.MD", " open loans ": "open loans.md",
		"../out": "", "/abs": "", "a/../../out": "", "https://x": "", ".md": "", "sub/.md": "", "": "",
	} {
		got, ok := documentPath(arg)
		if want == "" && ok || want != "" && got != want {
			t.Errorf("%q: got %q %v, want %q", arg, got, ok, want)
		}
	}
}
