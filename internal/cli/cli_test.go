package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

func run(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := Run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// initialized is a new project: a folder with the file .markitect/model, which
// names docs/model, and that empty model folder. The tests of the commands start
// from it.
func initialized(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	markitectFile(t, dir)
	if err := os.MkdirAll(modelOf(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// markitectFile writes .markitect/model into dir.
func markitectFile(t *testing.T, dir string) {
	t.Helper()
	write(t, dir, project.ModelFile, "docs/model\n")
}

// modelOf is the model folder of a project made by initialized.
func modelOf(dir string) string { return filepath.Join(dir, "docs", "model") }

func write(t *testing.T, dir, p, text string) {
	t.Helper()
	file := filepath.Join(dir, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, p string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

// model sets up a project whose model holds files and returns its folder.
func model(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := initialized(t)
	for p, text := range files {
		file := filepath.Join(modelOf(dir), filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// readme is the README of a module: a title and, under Contents, a link to
// each of the documents it shows.
func readme(title string, shown ...string) string {
	text := "# " + title + "\n\n## Contents\n"
	for _, p := range shown {
		text += "- [" + p + "](" + p + ")\n"
	}
	return text
}

func TestCheckText(t *testing.T) {
	dir := model(t, map[string]string{
		"types/README.md":         readme("Types", "member.md"),
		"types/member.md":         "# Member\n\nA person.\n\n## Card\nThe number.\n",
		"lending/README.md":       readme("Lending", "borrow.md", "loans/README.md"),
		"lending/loans/README.md": readme("Loans", "ada.md"),
		"lending/loans/ada.md":    "# Ada\r\n\r\nType: [Member](../../types/member.md)\r\n\r\n## Card\r\n\r\n",
		"lending/borrow.md":       "# Borrow\n\nSee [the loans](loans/ghost.md).\n",
	})
	code, out, errOut := run("check", "-C", dir)
	want := strings.Join([]string{
		`lending/borrow.md:3: link.unresolved: no document at "loans/ghost.md"`,
		`lending/loans/ada.md:5: section.empty: empty section "Card"`,
		"2 errors in 6 documents", ""}, "\n")
	if code != ExitFound || out != want || errOut != "" {
		t.Errorf("got %d\n%s%s\nwant 1\n%s", code, out, errOut, want)
	}
}

func TestCheckClean(t *testing.T) {
	dir := model(t, map[string]string{"README.md": readme("A", "a.md"), "a.md": "# A\n"})
	if code, out, _ := run("check", "-C", dir); code != ExitOK || out != "0 errors in 2 documents\n" {
		t.Errorf("got %d %q", code, out)
	}
}

func TestCheckJSON(t *testing.T) {
	dir := model(t, map[string]string{
		"README.md":     readme("Root", "a.md", "sub/README.md"),
		"a.md":          "# A\n",
		"sub/README.md": readme("Sub", "b.md"),
		"sub/b.md":      "# B\n\n[x](../ghost.md)\n",
	})
	for _, args := range [][]string{{"check", "-C", dir, "--json"}, {"check", "--json", "-C", dir}, {"check", "-C", dir, "-json", "--"}} {
		code, out, _ := run(args...)
		var doc map[string]any
		if err := json.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		want := map[string]any{
			"format_version": float64(1), "kind": "check", "documents": float64(4),
			"diagnostics": []any{map[string]any{"file": "sub/b.md", "line": float64(3), "code": "link.unresolved", "message": `no document at "../ghost.md"`}},
		}
		if code != ExitFound || !reflect.DeepEqual(doc, want) {
			t.Errorf("%v: got %d %v", args, code, doc)
		}
	}
	clean := model(t, map[string]string{"README.md": readme("A", "a.md"), "a.md": "# A\n"})
	if code, out, _ := run("check", "--json", "-C", clean); code != ExitOK || !strings.Contains(out, `"diagnostics": []`) {
		t.Errorf("got %d %s", code, out)
	}
}

// The help groups the commands under three headings and names every command
// once.
func TestHelpGroupsTheCommands(t *testing.T) {
	_, out, _ := run("help")
	at := -1
	for _, heading := range []string{"Look", "Check", "Tool"} {
		i := strings.Index(out, "\n"+heading+"\n")
		if i <= at {
			t.Fatalf("heading %s is missing or out of order in\n%s", heading, out)
		}
		at = i
	}
	for name := range commands {
		if n := strings.Count(out, "\n  "+name+" "); n != 1 {
			t.Errorf("the help names %s %d times", name, n)
		}
	}
	// The commands are exactly these eight.
	var names []string
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	const want = "check diff licenses questions show table topic version"
	if got := strings.Join(names, " "); got != want {
		t.Errorf("the commands are %s, want %s", got, want)
	}
}

func TestUsage(t *testing.T) {
	dir := model(t, map[string]string{"a.md": "# A\n"})
	file := filepath.Join(modelOf(dir), "a.md")
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
		{[]string{"check", "-C", dir, "--model", "x"}, "unknown flag --model"},
		{[]string{"check", "-C", dir, "--all"}, "--all is a flag of topic"},
		{[]string{"check", "-C", dir, "--file", "a.md"}, "unknown flag --file"},
		{[]string{"init", "-C", dir}, `unknown command "init"`},
		{[]string{"apply", "-C", dir}, `unknown command "apply"`},
		{[]string{"new", "-C", dir}, `unknown command "new"`},
		{[]string{"status", "-C", dir}, `unknown command "status"`},
		{[]string{"confirm", "-C", dir}, `unknown command "confirm"`},
		{[]string{"move", "-C", dir}, `unknown command "move"`},
		{[]string{"compare", "-C", dir}, `unknown command "compare"`},
		{[]string{"check", "-C", filepath.Join(dir, "missing")}, "not a folder"},
		{[]string{"check", "-C", file}, "not a folder"},
		{[]string{"check", "-C", modelOf(dir)}, "holds no .markitect/model; that file holds one line, the path of the model folder below the project folder"},
	} {
		code, out, errOut := run(tc.args...)
		if code != ExitFailed || out != "" || !strings.Contains(errOut, tc.want) {
			t.Errorf("%v: got %d %q %q", tc.args, code, out, errOut)
		}
	}
	for _, help := range []string{"help", "-h", "--help"} {
		if code, out, _ := run(help); code != ExitOK || !strings.Contains(out, "diff OLD [NEW]") {
			t.Errorf("%s: %d %q", help, code, out)
		}
	}
}

// The flags may stand before the command, as in markitect -C PROJECT check, and between its
// arguments.
func TestFlagsMayStandBeforeTheCommand(t *testing.T) {
	dir := model(t, map[string]string{"README.md": readme("A", "a.md"), "a.md": "# A\n"})
	for _, args := range [][]string{
		{"-C", dir, "check"},
		{"--json", "-C", dir, "check"},
		{"-C", dir, "--json", "check"},
		{"-C", dir, "show", "a"},
		{"--json", "-C", dir, "show", "a"},
		{"-C", dir, "table"},
		{"-C", dir, "table", "--json"},
	} {
		code, out, errOut := run(args...)
		if code != ExitOK || errOut != "" || out == "" {
			t.Errorf("%v: %d %q %q", args, code, out, errOut)
		}
	}
	// The same as with the flags after the command.
	_, before, _ := run("check", "-C", dir, "--json")
	if _, after, _ := run("--json", "-C", dir, "check"); after != before {
		t.Errorf("%s differs from %s", after, before)
	}
	// A value of a flag is no command, and a missing command is an error.
	if code, _, errOut := run("-C", dir); code != ExitFailed || !strings.Contains(errOut, "flag -C needs a value") && !strings.Contains(errOut, "no command given") {
		t.Errorf("%d %q", code, errOut)
	}
	if code, _, errOut := run("-C", dir, "show"); code != ExitFailed || !strings.Contains(errOut, "usage: markitect show DOCUMENT") {
		t.Errorf("%d %q", code, errOut)
	}
}

// A wrong number of arguments prints the error and the one line of usage of
// the command, not the whole help.
func TestWrongArgumentsPrintOneLineOfUsage(t *testing.T) {
	dir := model(t, map[string]string{"a.md": "# A\n"})
	for args, want := range map[string]string{
		"show":        "usage: markitect show DOCUMENT [-C PROJECT] [--json]",
		"check extra": "usage: markitect check [-C PROJECT] [--json]",
		"diff":        "usage: markitect diff OLD [NEW]",
	} {
		code, out, errOut := run(append(strings.Fields(args), "-C", dir)...)
		if code != ExitFailed || out != "" || !strings.HasPrefix(errOut, "markitect "+strings.Fields(args)[0]+": wrong number of arguments\n") ||
			!strings.Contains(errOut, want) || strings.Count(errOut, "\n") > 4 {
			t.Errorf("%s: %d %q", args, code, errOut)
		}
	}
	if code, _, errOut := run("frobnicate"); code != ExitFailed || strings.Count(errOut, "\n") != 1 || !strings.Contains(errOut, "markitect --help") {
		t.Errorf("%d %q", code, errOut)
	}
}

// A flag's value may be named like a command, and what stands before the
// command is told as plainly as what stands after it.
func TestFlagsBeforeTheCommandAreRead(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "check")
	markitectFile(t, project)
	if err := os.MkdirAll(filepath.Join(project, "docs", "model"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(parent)
	if code, out, errOut := run("-C", "check", "check"); code != ExitOK || out != "0 errors in 0 documents\n" {
		t.Errorf("-C check check: %d %q %q", code, out, errOut)
	}
	for _, tc := range []struct {
		args []string
		code int
		want string // in the error, or in the output when the code is 0
	}{
		{[]string{"--version"}, ExitFailed, "unknown flag --version"},
		{[]string{"-x"}, ExitFailed, "unknown flag -x"},
		{[]string{"-C"}, ExitFailed, "flag -C needs a value"},
		{[]string{"--frob", "p1", "check"}, ExitFailed, "unknown flag --frob"},
		{[]string{"--json"}, ExitFailed, "no command given"},
		{[]string{"-C", "check"}, ExitFailed, "no command given"},
		{[]string{"-C", "p1", "--help"}, ExitOK, "usage: markitect COMMAND"},
		{[]string{"--json", "help"}, ExitOK, "usage: markitect COMMAND"},
		{[]string{"check", "--help"}, ExitOK, "usage: markitect COMMAND"},
		{[]string{"show", "a", "-h"}, ExitOK, "usage: markitect COMMAND"},
		{[]string{"-C", "check", "frobnicate"}, ExitFailed, `unknown command "frobnicate"`},
	} {
		code, out, errOut := run(tc.args...)
		got := errOut
		if tc.code == ExitOK {
			got = out
		}
		if code != tc.code || !strings.Contains(got, tc.want) || strings.Contains(errOut, `command ""`) {
			t.Errorf("%v: %d %q %q", tc.args, code, out, errOut)
		}
	}
}
