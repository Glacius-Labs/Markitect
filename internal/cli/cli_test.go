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

// initialized is a new project that init set up.
func initialized(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	saved := fault
	fault = nil
	defer func() { fault = saved }()
	if code, _, errOut := run("init", "-C", dir); code != ExitOK {
		t.Fatal(errOut)
	}
	return dir
}

// specOf is the spec folder of a project that init set up.
func specOf(project string) string { return filepath.Join(project, "docs", "spec") }

// spec sets up a project whose spec holds files and returns its folder.
func spec(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := initialized(t)
	for p, text := range files {
		file := filepath.Join(specOf(dir), filepath.FromSlash(p))
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
	code, out, errOut := run("check", "-C", dir)
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
	if code, out, _ := run("check", "-C", dir); code != ExitOK || out != "0 errors in 1 document\n" {
		t.Errorf("got %d %q", code, out)
	}
}

func TestCheckJSON(t *testing.T) {
	dir := spec(t, map[string]string{
		"a.md":     "# A\n",
		"sub/b.md": "# B\n\n[x](../ghost.md)\n",
	})
	for _, args := range [][]string{{"check", "-C", dir, "--json"}, {"check", "--json", "-C", dir}, {"check", "-C", dir, "-json", "--"}} {
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
	if code, out, _ := run("check", "--json", "-C", clean); code != ExitOK || !strings.Contains(out, `"diagnostics": []`) {
		t.Errorf("got %d %s", code, out)
	}
}

func TestUsage(t *testing.T) {
	dir := spec(t, map[string]string{"a.md": "# A\n"})
	file := filepath.Join(specOf(dir), "a.md")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "usage:"},
		{[]string{"frobnicate"}, "unknown command"},
		{[]string{"check", "-C", dir, "extra"}, "wrong number of arguments"},
		{[]string{"show", "-C", dir}, "wrong number of arguments"},
		{[]string{"check", "-C", dir, "--strict"}, "unknown flag --strict"},
		{[]string{"check", "-C"}, "flag -C needs a value"},
		{[]string{"check", "-C", ""}, "flag -C needs a folder"},
		{[]string{"check", "-C", dir, "--spec", "x"}, "--spec is a flag of init"},
		{[]string{"init", "--spec"}, "flag --spec needs a value"},
		{[]string{"check", "-C", filepath.Join(dir, "missing")}, "not a folder"},
		{[]string{"check", "-C", file}, "not a folder"},
		{[]string{"check", "-C", specOf(dir)}, "holds no .markitect/version; markitect-next init makes one"},
	} {
		code, out, errOut := run(tc.args...)
		if code != ExitFailed || out != "" || !strings.Contains(errOut, tc.want) {
			t.Errorf("%v: got %d %q %q", tc.args, code, out, errOut)
		}
	}
	for _, help := range []string{"help", "-h", "--help"} {
		if code, out, _ := run(help); code != ExitOK || !strings.Contains(out, "init [--spec PATH]") {
			t.Errorf("%s: %d %q", help, code, out)
		}
	}
}
