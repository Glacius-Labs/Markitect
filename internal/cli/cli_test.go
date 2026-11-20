package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := Run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// spec writes a spec with a document in a nested folder and returns its folder.
func spec(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, text := range files {
		file := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestCheckText(t *testing.T) {
	dir := spec(t, map[string]string{
		"types/member.md":      "# Member\n\nA person.\n\n## Card\nThe number.\n",
		"lending/loans/ada.md": "# Ada\r\n\r\nType: [Member](../../types/member.md)\r\n\r\n## Card\r\n\r\n",
		"lending/borrow.md":    "# Borrow\n\nSee [the loans](loans/ghost.md).\n",
	})
	code, out, errOut := run("check", dir)
	want := strings.Join([]string{
		`lending/borrow.md:3: link.unresolved: no document at "loans/ghost.md"`,
		`lending/loans/ada.md:5: section.empty: empty section "Card"`,
		"2 errors in 3 documents", ""}, "\n")
	if code != ExitFound || out != want || errOut != "" {
		t.Errorf("got %d\n%s%s\nwant 1\n%s", code, out, errOut, want)
	}
}

func TestCheckClean(t *testing.T) {
	dir := spec(t, map[string]string{"a.md": "# A\n"})
	if code, out, _ := run("check", dir); code != ExitOK || out != "0 errors in 1 document\n" {
		t.Errorf("got %d %q", code, out)
	}
}

func TestCheckJSON(t *testing.T) {
	dir := spec(t, map[string]string{
		"a.md":     "# A\n",
		"sub/b.md": "# B\n\n[x](../ghost.md)\n",
	})
	for _, args := range [][]string{{"check", dir, "--json"}, {"check", "--json", dir}, {"check", "-json", "--", dir}} {
		code, out, _ := run(args...)
		var doc map[string]any
		if err := json.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		want := map[string]any{
			"format_version": float64(1), "kind": "check", "documents": float64(2),
			"diagnostics": []any{map[string]any{"file": "sub/b.md", "line": float64(3), "code": "link.unresolved", "message": `no document at "../ghost.md"`}},
		}
		if code != ExitFound || !reflect.DeepEqual(doc, want) {
			t.Errorf("%v: got %d %v", args, code, doc)
		}
	}
	clean := spec(t, map[string]string{"a.md": "# A\n"})
	if code, out, _ := run("check", "--json", clean); code != ExitOK || !strings.Contains(out, `"diagnostics": []`) {
		t.Errorf("got %d %s", code, out)
	}
}

func TestUsage(t *testing.T) {
	dir := spec(t, map[string]string{"a.md": "# A\n"})
	file := filepath.Join(dir, "a.md")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "usage:"},
		{[]string{"frobnicate"}, "unknown command"},
		{[]string{"check"}, "takes one spec folder"},
		{[]string{"check", dir, dir}, "takes one spec folder"},
		{[]string{"check", dir, "--strict"}, "unknown flag --strict"},
		{[]string{"check", filepath.Join(dir, "missing")}, "not a folder"},
		{[]string{"check", file}, "not a folder"},
	} {
		code, out, errOut := run(tc.args...)
		if code != ExitFailed || out != "" || !strings.Contains(errOut, tc.want) {
			t.Errorf("%v: got %d %q %q", tc.args, code, out, errOut)
		}
	}
	for _, help := range []string{"help", "-h", "--help"} {
		if code, out, _ := run(help); code != ExitOK || !strings.Contains(out, "markitect-next check SPECFOLDER") {
			t.Errorf("%s: %d %q", help, code, out)
		}
	}
}
