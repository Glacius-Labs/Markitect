package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/lang"
	"github.com/Glacius-Labs/Markitect/internal/library"
)

// branches is a small spec of members and branches.
var branches = map[string]string{
	"types/member.md":   "# Member\n\nA card holder.\n\n## Card\nThe number.\n\n## Branch\nThe home branch, as a link.\n\n## Notes (optional)\nAnything else.\n",
	"types/branch.md":   "# Branch\n\nA place that lends books.\n",
	"places/central.md": "# Central\n\nType: [Branch](../types/branch.md)\n\nThe main building.\n",
	"members/ada.md":    "# Ada Lovelace\n\nType: [Member](../types/member.md)\n\n## Card\nL-1 | gold\n\n## Branch\n<!-- moved last year -->\n[Central](../places/central.md), see [Grace](grace.md#card).\n",
	"members/grace.md":  "# Grace Hopper\n\nType: [Member](../types/member.md)\n\n## card\nL-2\n\n## Branch\n[Main](../places/central.md)\n\n## Notes\n![photo](photo.png) on file, [missing](gone.md).\n",
}

// files returns the files of a project's spec folder, by their paths below it.
func files(t *testing.T, project string) map[string]string {
	t.Helper()
	dir := specOf(project)
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		rel, _ := filepath.Rel(dir, p)
		out[filepath.ToSlash(rel)] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestShow(t *testing.T) {
	dir := spec(t, branches)
	code, out, _ := run("show", "-C", dir, `members\ada`)
	want := "members/ada.md\nTitle: Ada Lovelace\nType: types/member.md\nSections:\n  Card\n  Branch\n" +
		"Links to:\n  members/grace.md\n  places/central.md\nLinked from: (none)\n"
	if code != ExitOK || out != want {
		t.Errorf("got %d\n%s\nwant\n%s", code, out, want)
	}
	_, out, _ = run("show", "-C", dir, "places/central.md")
	if !strings.HasSuffix(out, "Linked from:\n  members/ada.md (Branch)\n  members/grace.md (Branch)\n") {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "types/member")
	if !strings.Contains(out, "Type: (none)\nSections:\n  Card\n  Branch\n  Notes (optional)\n") ||
		!strings.HasSuffix(out, "Documents:\n  members/ada.md\n  members/grace.md\n") {
		t.Error(out)
	}
	_, out, _ = run("show", "-C", dir, "members/grace", "--json")
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["kind"] != "show" || doc["type_state"] != "ok" || len(doc["links_to"].([]any)) != 2 || len(doc["documents"].([]any)) != 0 {
		t.Error(out)
	}
	for _, arg := range []string{"nobody", "../outside", "types"} {
		if code, out, errOut := run("show", "-C", dir, arg); code != ExitFailed || out != "" || !strings.Contains(errOut, "unknown path") {
			t.Errorf("%s: %d %q %q", arg, code, out, errOut)
		}
	}
}

func TestTable(t *testing.T) {
	dir := spec(t, branches)
	code, out, _ := run("table", "-C", dir, "types/member")
	want := "| Title | Card | Branch | Notes |\n| --- | --- | --- | --- |\n" +
		"| Ada Lovelace | L-1 \\| gold | Central, see Grace. |  |\n" +
		"| Grace Hopper | L-2 | Main | photo on file, missing. |\n"
	if code != ExitOK || out != want {
		t.Errorf("got %d\n%s\nwant\n%s", code, out, want)
	}
}

func TestNewAndMove(t *testing.T) {
	dir := spec(t, branches)
	code, out, _ := run("new", "-C", dir, "types/member", "members/alan", "Alan Turing")
	if code != ExitOK || out != "created members/alan.md\n" {
		t.Fatal(code, out)
	}
	if got := files(t, dir)["members/alan.md"]; got != "# Alan Turing\n\nType: [Member](../types/member.md)\n\n## Card\n\n## Branch\n\n" {
		t.Error(got)
	}
	before := files(t, dir)
	if code, out, errOut := run("new", "-C", dir, "types/member", "members/ada"); code != ExitFailed || out != "" || !strings.Contains(errOut, "already exists") {
		t.Error(code, out, errOut)
	}
	code, out, _ = run("move", "-C", dir, "places/central", "places/main hall")
	if code != ExitOK || out != "moved places/central.md -> places/main hall.md\nchanged members/ada.md\nchanged members/grace.md\n" {
		t.Fatal(code, out)
	}
	after := files(t, dir)
	if !strings.Contains(after["members/ada.md"], "[Central](../places/main%20hall.md)") || after["places/main hall.md"] != before["places/central.md"] {
		t.Error(after)
	}
	moved := files(t, dir)
	for _, args := range [][]string{{"places/main hall", "members/ada"}, {"nobody", "x"}, {"members/ada", "../out"}} {
		if code, out, _ := run(append([]string{"move", "-C", dir}, args...)...); code != ExitFailed || out != "" {
			t.Errorf("%v: %d %q", args, code, out)
		}
	}
	if !equalMaps(files(t, dir), moved) {
		t.Error("a refused move changed files")
	}
}

func equalMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// templates lists every template of the library: each folder below templates/.
func templates(t *testing.T) []string {
	t.Helper()
	var out []string
	err := fs.WalkDir(library.FS(), "templates", func(p string, e fs.DirEntry, err error) error {
		if err == nil && e.IsDir() && p != "templates" {
			out = append(out, strings.TrimPrefix(p, "templates/"))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEveryTemplateApplies(t *testing.T) {
	tpls := templates(t)
	if len(tpls) < 10 {
		t.Fatalf("templates %v", tpls)
	}
	for _, tpl := range tpls {
		for _, target := range []string{"", "areas/" + path.Base(tpl)} {
			dir := initialized(t)
			args := []string{"apply", "-C", dir, tpl}
			if target != "" {
				args = append(args, target)
			}
			if code, out, errOut := run(args...); code != ExitOK {
				t.Fatalf("%s at %q: %d %s %s", tpl, target, code, out, errOut)
			}
			s, err := lang.Load(os.DirFS(specOf(dir)))
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range lang.Check(s) {
				if d.Code != lang.CodeSectionEmpty {
					t.Errorf("%s at %q: %s", tpl, target, d)
				}
			}
			for p := range files(t, dir) {
				if path.Base(p) == library.ReportType {
					t.Errorf("%s copied the report type", tpl)
				}
			}
		}
	}
}

func TestApplyCommand(t *testing.T) {
	// apply creates the spec folder when it is missing.
	dir := initialized(t)
	if err := os.Remove(specOf(dir)); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run("apply", "-C", dir, "software/domain")
	if code != ExitOK || out != "copied README.md\ncopied packages/software/\ncopied packages/core/\n" {
		t.Fatal(code, out)
	}
	if line := strings.Split(files(t, dir)["README.md"], "\n")[2]; line != "Type: [Domain](packages/software/domain.md)" {
		t.Error(line)
	}
	before := files(t, dir)
	code, out, errOut := run("apply", "-C", dir, "software")
	if code != ExitFailed || out != "" || !strings.Contains(errOut, "README.md: already exists") || !equalMaps(files(t, dir), before) {
		t.Error(code, out, errOut)
	}
	// A package the spec has is kept, never overwritten or completed.
	if err := os.WriteFile(filepath.Join(specOf(dir), "packages", "core", "role.md"), []byte("# Role\n\nOur own.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(specOf(dir), "packages", "core", "term.md")); err != nil {
		t.Fatal(err)
	}
	code, out, _ = run("apply", "-C", dir, "company", "orgs/main")
	if code != ExitOK || strings.Contains(out, "packages/core/") || !strings.Contains(out, "copied packages/company/\n") {
		t.Error(code, out)
	}
	after := files(t, dir)
	if after["packages/core/role.md"] != "# Role\n\nOur own.\n" || after["packages/core/term.md"] != "" {
		t.Error("the spec's package changed")
	}
	if !strings.Contains(after["orgs/main/team/README.md"], "(../../../packages/core/role.md)") {
		t.Error(after["orgs/main/team/README.md"])
	}
	for _, args := range [][]string{{"nothing"}, {".."}, {"software", "../up"}} {
		if code, out, _ := run(append([]string{"apply", "-C", initialized(t)}, args...)...); code != ExitFailed || out != "" {
			t.Errorf("%v: %d %q", args, code, out)
		}
	}
}

func TestLibrary(t *testing.T) {
	s, err := lang.Load(library.FS())
	if err != nil {
		t.Fatal(err)
	}
	var packages []string
	for _, d := range s.Documents {
		if strings.HasPrefix(d.Path, "packages/") {
			packages = append(packages, strings.Split(d.Path, "/")[1])
		}
	}
	if got := strings.Join(unique(packages), " "); got != "company core software" {
		t.Errorf("packages %s", got)
	}
	// Only the templates' sections are empty.
	for _, d := range lang.Check(s) {
		if d.Code != lang.CodeSectionEmpty || !strings.HasPrefix(d.Path, "templates/") {
			t.Error(d)
		}
	}
	report, _ := s.Document(library.ReportType)
	var names []string
	for _, d := range report.Declared() {
		if !d.Optional {
			names = append(names, d.Name)
		}
	}
	if got := strings.Join(names, ", "); got != "Source, Place, Finding, Occurred or risk, Violated property" {
		t.Errorf("required report sections: %s", got)
	}
	quality, _ := s.Document("packages/software/quality.md")
	if n := len(quality.Declared()); n != 10 {
		t.Errorf("quality declares %d sections", n)
	}
}

func unique(list []string) []string {
	sort.Strings(list)
	var out []string
	for _, s := range list {
		if len(out) == 0 || out[len(out)-1] != s {
			out = append(out, s)
		}
	}
	return out
}

func TestReadOnlyFileChangesNothing(t *testing.T) {
	dir := spec(t, map[string]string{"a.md": "# A\n", "z.md": "# Z\n\nSee [a](a.md).\n"})
	z := filepath.Join(specOf(dir), "z.md")
	if err := os.Chmod(z, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(z, 0o644) })
	if f, err := os.OpenFile(z, os.O_WRONLY, 0); err == nil {
		f.Close()
		t.Skip("the file system lets this user write a read-only file")
	}
	before := files(t, dir)
	code, out, errOut := run("move", "-C", dir, "a.md", "n.md")
	if code != ExitFailed || out != "" || !strings.Contains(errOut, "z.md: cannot be written") || !equalMaps(files(t, dir), before) {
		t.Error(code, out, errOut)
	}
}

// TestFailureLeavesTheSpecAsItWas fails each step of a move and an apply in turn
// and checks that the spec folder is as it was.
func TestFailureLeavesTheSpecAsItWas(t *testing.T) {
	cases := []struct {
		name string
		args func(dir string) []string
	}{
		{"move", func(dir string) []string {
			return []string{"move", "-C", dir, "places/central", "elsewhere/deep/main hall"}
		}},
		{"apply", func(dir string) []string { return []string{"apply", "-C", dir, "software", "areas/new"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := 0
			fault = func(int) error { steps++; return nil }
			if code, _, errOut := run(tc.args(spec(t, branches))...); code != ExitOK {
				t.Fatal(errOut)
			}
			if steps < 4 {
				t.Fatalf("only %d steps", steps)
			}
			for k := 1; k <= steps; k++ {
				dir := spec(t, branches)
				before := files(t, dir)
				fault = func(step int) error {
					if step == k {
						return errors.New("injected")
					}
					return nil
				}
				code, out, errOut := run(tc.args(dir)...)
				if code != ExitFailed || out != "" || !strings.Contains(errOut, "injected") {
					t.Errorf("step %d: %d %q %q", k, code, out, errOut)
				}
				if after := files(t, dir); !equalMaps(after, before) {
					t.Errorf("step %d changed the spec: %v", k, after)
				}
				var dirs []string
				_ = filepath.WalkDir(specOf(dir), func(p string, e fs.DirEntry, err error) error {
					if err == nil && e.IsDir() && p != specOf(dir) {
						dirs = append(dirs, p)
					}
					return err
				})
				if len(dirs) != 3 {
					t.Errorf("step %d left folders %v", k, dirs)
				}
			}
			fault = nil
		})
	}
}

func TestFailedUndoNamesWhatIsLeft(t *testing.T) {
	args := func(dir string) []string { return []string{"move", "-C", dir, "places/central", "elsewhere/main hall"} }
	steps := 0
	fault = func(int) error { steps++; return nil }
	run(args(spec(t, branches))...)
	fault = func(step int) error {
		if step == steps {
			return errors.New("injected")
		}
		return nil
	}
	undoFault = func(int) error { return errors.New("undo refused") }
	t.Cleanup(func() { fault, undoFault = nil, nil })
	dir := spec(t, branches)
	code, out, errOut := run(args(dir)...)
	if code != ExitFailed || out != "" || !strings.Contains(errOut, "injected") || !strings.Contains(errOut, "undoing it failed: undo refused") {
		t.Fatal(code, out, errOut)
	}
	_, left, ok := strings.Cut(strings.TrimSpace(errOut), "left behind: ")
	if !ok || !strings.Contains(left, ".markitect-") {
		t.Fatal(errOut)
	}
	for _, f := range strings.Split(left, ", ") {
		if _, err := os.Lstat(f); err != nil {
			t.Errorf("named %s, which is not there", f)
		}
	}
}

func TestCreateRefusesAFileThatAppeared(t *testing.T) {
	dir := t.TempDir()
	temp, path := filepath.Join(dir, "new.tmp"), filepath.Join(dir, "doc.md")
	if err := os.WriteFile(temp, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("there first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := createFrom(temp, path); !errors.Is(err, fs.ErrExist) {
		t.Errorf("got %v", err)
	}
	if data, _ := os.ReadFile(path); string(data) != "there first" {
		t.Errorf("overwrote it with %q", data)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := createFrom(temp, path); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != "new" {
		t.Errorf("created %q", data)
	}
}

func TestTopicAndQuestions(t *testing.T) {
	dir := spec(t, branches)
	code, out, _ := run("topic", "-C", dir)
	if code != ExitOK || !strings.HasPrefix(out, "# Signals of the spec\n") || !strings.Contains(out, "\n### weak (") || !strings.Contains(out, "\n### unlinked-mention (") {
		t.Fatal(code, out)
	}
	code, out, _ = run("topic", "-C", dir, "places/central")
	for _, part := range []string{"# Topic: places/central.md (document)\n", "\n## Strong statements about it\n", "\n## Text\n\n# Central\n",
		"\n## Linked from\n\nmembers/ada.md:10 (Branch): Central, see Grace.\n", "\n## Mentioned without a link\n", "\n## Signals\n"} {
		if code != ExitOK || !strings.Contains(out, part) {
			t.Errorf("missing %q in\n%s", part, out)
		}
	}
	_, out, _ = run("topic", "-C", dir, "types/member", "--json")
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["kind"] != "topic" || doc["topic_kind"] != "type" || len(doc["typed_documents"].([]any)) != 2 {
		t.Error(out)
	}
	if code, _, errOut := run("topic", "-C", dir, "nowhere"); code != ExitFailed || !strings.Contains(errOut, "unknown topic") {
		t.Error(code, errOut)
	}
	code, out, _ = run("questions", "-C", dir)
	if code != ExitOK || !strings.Contains(out, "no package of the library") {
		t.Error(code, out)
	}
	applied := initialized(t)
	run("apply", "-C", applied, "software")
	_, out, _ = run("questions", "-C", applied)
	if !strings.HasPrefix(out, "## core\n\n1. ") || !strings.Contains(out, "\n## software\n\n1. Who may call each operation") {
		t.Error(out)
	}
	if code, out, _ := run("topic", "-C", applied, "--json"); code != ExitOK || !strings.Contains(out, `"topic_kind": "spec"`) {
		t.Error(code, out)
	}
}

func TestReferenceValuesCoverEveryType(t *testing.T) {
	refs, err := references()
	if err != nil {
		t.Fatal(err)
	}
	err = fs.WalkDir(library.FS(), "packages", func(p string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			if _, ok := refs.For(p); !ok {
				t.Errorf("no reference value for %s", p)
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
