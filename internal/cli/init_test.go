package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/library"
)

// skillNames are the skills the library holds, whatever they are.
func skillNames(t *testing.T) []string {
	t.Helper()
	names, err := library.SkillNames()
	if err != nil || len(names) == 0 {
		t.Fatalf("skills %v: %v", names, err)
	}
	return names
}

// diagnosticCodes returns the codes of the lines check printed.
func diagnosticCodes(out string) map[string]int {
	codes := map[string]int{}
	for _, line := range strings.Split(out, "\n") {
		if parts := strings.SplitN(line, ": ", 3); len(parts) == 3 && strings.Contains(parts[0], ":") {
			codes[parts[1]]++
		}
	}
	return codes
}

func TestInitLaysOutTheProject(t *testing.T) {
	dir := t.TempDir()
	code, out, errOut := run("init", "-C", dir)
	if code != ExitOK || errOut != "" {
		t.Fatal(code, out, errOut)
	}
	for _, line := range []string{"created .markitect/", "created docs/spec/", "created docs/spec/factory/", "created docs/spec/packages/factory/", "created AGENTS.md"} {
		if !strings.Contains(out, line+"\n") {
			t.Errorf("init did not say %q:\n%s", line, out)
		}
	}
	for _, name := range skillNames(t) {
		for _, folder := range skillFolders {
			if !strings.Contains(out, "created "+folder+"/"+name+"/\n") {
				t.Errorf("init did not say it created %s/%s", folder, name)
			}
		}
	}

	// .markitect/ holds the files of the project and, in each folder of the
	// work, an empty .gitkeep; nothing else.
	var markitect []string
	for _, p := range tree(t, dir) {
		if strings.HasPrefix(p, ".markitect/") {
			markitect = append(markitect, p)
		}
	}
	want := []string{".markitect/", ".markitect/.gitattributes", ".markitect/briefing/", ".markitect/briefing/.gitkeep", ".markitect/context/", ".markitect/context/.gitkeep",
		".markitect/findings/", ".markitect/findings/closed/", ".markitect/findings/closed/.gitkeep", ".markitect/findings/pending/", ".markitect/findings/pending/.gitkeep",
		".markitect/findings/ready/", ".markitect/findings/ready/.gitkeep", ".markitect/ignore", ".markitect/plan/", ".markitect/plan/.gitkeep", ".markitect/spec", ".markitect/version"}
	if strings.Join(markitect, " ") != strings.Join(want, " ") {
		t.Errorf("init wrote\n%v\nwant\n%v", markitect, want)
	}
	if read(t, dir, ".markitect/version") != buildinfo.Version()+"\n" || read(t, dir, ".markitect/spec") != "docs/spec\n" || read(t, dir, ".markitect/ignore") != "" ||
		read(t, dir, ".markitect/.gitattributes") != "state.jsonl merge=union\nhistory.jsonl merge=union\n" {
		t.Error("init wrote other contents")
	}

	// The spec holds the four templates, each in a folder of its own name.
	for _, tpl := range initTemplates {
		if _, err := os.Stat(filepath.Join(specOf(dir), tpl, "README.md")); err != nil {
			t.Errorf("the template %s: %v", tpl, err)
		}
	}
	if _, err := os.Stat(filepath.Join(specOf(dir), "packages", "factory", "story.md")); err != nil {
		t.Error(err)
	}

	// The skills lie, byte for byte, in both places.
	for _, name := range skillNames(t) {
		err := fs.WalkDir(library.Skills(), name, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			want, err := fs.ReadFile(library.Skills(), p)
			if err != nil {
				return err
			}
			for _, folder := range skillFolders {
				if got := read(t, dir, folder+"/"+p); got != string(want) {
					t.Errorf("%s/%s differs from the skill of the library", folder, p)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// AGENTS.md names the spec, the way of working, the skills and the Assistant.
	agents := read(t, dir, "AGENTS.md")
	for _, part := range []string{"## Markitect", "`docs/spec/`", "`docs/spec/factory/`", "`.claude/skills/`", "`.agents/skills/`", "skill `markitect-assistant`"} {
		if !strings.Contains(agents, part) {
			t.Errorf("AGENTS.md lacks %s:\n%s", part, agents)
		}
	}

	// The spec passes check but for the sections a project fills, and none of
	// them stands in the way of working; status runs.
	code, out, _ = run("check", "-C", dir)
	codes := diagnosticCodes(out)
	if code != ExitFound || len(codes) != 1 || codes["section.empty"] == 0 {
		t.Errorf("check: %d %v\n%s", code, codes, out)
	}
	if strings.Contains(out, "\nfactory/") || strings.HasPrefix(out, "factory/") {
		t.Errorf("check finds something in factory/:\n%s", out)
	}
	// check reports the empty sections of the templates, and status lists and
	// confirm binds all the same: right after init, a reviewer can work.
	l := statusOf(t, dir)
	e, kind := l.entry("project/README.md")
	if l.code != ExitFound || kind != "changed" || len(e.Fingerprint) != 8 || len(l.State) != 0 {
		t.Fatalf("status: %d %v", l.code, l.summary())
	}
	if _, kind := l.entry("factory/README.md"); kind != "changed" {
		t.Error("the way of working is not listed")
	}
	unexplained := strings.Join(l.Unexplained, " ")
	for _, part := range []string{"AGENTS.md", ".claude/skills/markitect-assistant/SKILL.md", ".agents/skills/markitect-assistant/SKILL.md"} {
		if !strings.Contains(unexplained, part) {
			t.Errorf("%s is not unexplained: %s", part, unexplained)
		}
	}
	if code, out, errOut := run("confirm", "-C", dir, "project/README.md@"+e.Fingerprint, "skip", "-m", "the sections are filled later"); code != ExitOK {
		t.Fatalf("confirm: %d %s %s", code, out, errOut)
	}
	if _, kind := statusOf(t, dir).entry("project/README.md"); kind != "" {
		t.Error("the confirmed document is still listed")
	}
}

func TestInitWithASpecFolder(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "handbook/lending.md", "# Lending\n")
	code, out, _ := run("init", "-C", dir, `--spec=handbook\`, "--json")
	var doc initResult
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if code != ExitOK || doc.Kind != "init" || doc.Spec != "handbook" || doc.Version != buildinfo.Version() || doc.Created[0] != ".markitect/" || doc.Created[1] != "handbook/project/" ||
		doc.Appended == nil || len(doc.Appended) != 0 {
		t.Errorf("got %d %s", code, out)
	}
	if read(t, dir, ".markitect/spec") != "handbook\n" || !strings.Contains(read(t, dir, "AGENTS.md"), "`handbook/factory/`") {
		t.Error("the spec folder is not the one given")
	}
	if code, out, _ := run("check", "-C", dir); code != ExitFound || len(diagnosticCodes(out)) != 1 {
		t.Error(code, out)
	}
	for _, spec := range []string{"..", "/abs", ".markitect/spec", "."} {
		if code, _, errOut := run("init", "-C", t.TempDir(), "--spec", spec); code != ExitFailed || !strings.Contains(errOut, "spec folder") {
			t.Errorf("%s: %d %q", spec, code, errOut)
		}
	}
	file := t.TempDir()
	write(t, file, "spec", "")
	if code, _, errOut := run("init", "-C", file, "--spec", "spec"); code != ExitFailed || !strings.Contains(errOut, "is a file") {
		t.Error(code, errOut)
	}
	if got := tree(t, file); strings.Join(got, " ") != "spec" {
		t.Errorf("a refused init left %v", got)
	}
}

// init refuses a project that .markitect/ makes one already, and writes
// nothing.
func TestInitRefusesAnExistingProject(t *testing.T) {
	dir := t.TempDir()
	if code, _, errOut := run("init", "-C", dir); code != ExitOK {
		t.Fatal(errOut)
	}
	before := tree(t, dir)
	if code, out, errOut := run("init", "-C", dir); code != ExitFailed || out != "" || !strings.Contains(errOut, ".markitect/ exists already") {
		t.Error(code, out, errOut)
	}
	if got := tree(t, dir); strings.Join(got, " ") != strings.Join(before, " ") {
		t.Errorf("a refused init changed %v", got)
	}
}

// An AGENTS.md that exists gets the entry as a section of its own at the end,
// and nothing that stands there changes; skills of other names stay.
func TestInitAppendsToAnAgentsFileAndKeepsOtherSkills(t *testing.T) {
	for name, existing := range map[string]string{
		"with a final line end": "# Our agents\n\nBe kind.\n",
		"without one":           "# Our agents\n\nBe kind.",
		"with CRLF":             "# Our agents\r\n\r\nBe kind.\r\n",
		"already a blank line":  "# Our agents\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, "AGENTS.md", existing)
			write(t, dir, ".claude/skills/mine/SKILL.md", "---\nname: mine\ndescription: mine\n---\n")
			write(t, dir, ".agents/skills/other/SKILL.md", "---\nname: other\ndescription: other\n---\n")
			code, out, errOut := run("init", "-C", dir)
			if code != ExitOK || !strings.Contains(out, "appended an entry to AGENTS.md\n") || strings.Contains(out, "created AGENTS.md") {
				t.Fatal(code, out, errOut)
			}
			got := read(t, dir, "AGENTS.md")
			if !strings.HasPrefix(got, existing) {
				t.Errorf("AGENTS.md changed what stood there:\n%q", got)
			}
			rest := strings.TrimPrefix(got, existing)
			eol := "\n"
			if strings.Contains(existing, "\r\n") {
				eol = "\r\n"
			}
			if !strings.Contains(rest, "## Markitect"+eol+eol) || strings.Count(got, "## Markitect") != 1 ||
				eol == "\r\n" && strings.Count(rest, "\n") != strings.Count(rest, "\r\n") {
				t.Errorf("the entry is not a section of its own, in the line ends of the file:\n%q", rest)
			}
			if read(t, dir, ".claude/skills/mine/SKILL.md") != "---\nname: mine\ndescription: mine\n---\n" || read(t, dir, ".agents/skills/other/SKILL.md") != "---\nname: other\ndescription: other\n---\n" {
				t.Error("init changed a skill that was there")
			}
			if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", skillNames(t)[0], "SKILL.md")); err != nil {
				t.Error(err)
			}
		})
	}
}

// What is in the way refuses init, with every reason, and nothing is written.
func TestInitRefusesWhatIsInTheWay(t *testing.T) {
	first := skillNames(t)[0]
	for name, tc := range map[string]struct {
		file, want string
	}{
		"a skill of the standard in .claude": {".claude/skills/" + first + "/SKILL.md", ".claude/skills/" + first + ": already exists"},
		"a skill of the standard in .agents": {".agents/skills/" + first + "/notes.txt", ".agents/skills/" + first + ": already exists"},
		"a folder of a template":             {"docs/spec/factory/own.md", "docs/spec/factory: already exists"},
		"a template that is a file":          {"docs/spec/project", "docs/spec/project: already exists"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, tc.file, "# Mine\n")
			write(t, dir, "AGENTS.md", "# Ours\n")
			before := tree(t, dir)
			code, out, errOut := run("init", "-C", dir)
			if code != ExitFailed || out != "" || !strings.Contains(errOut, tc.want) {
				t.Fatal(code, out, errOut)
			}
			if got := tree(t, dir); strings.Join(got, " ") != strings.Join(before, " ") {
				t.Errorf("a refused init changed %v", got)
			}
			if read(t, dir, "AGENTS.md") != "# Ours\n" {
				t.Error("a refused init changed AGENTS.md")
			}
		})
	}
	t.Run("AGENTS.md that is no file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "AGENTS.md"), 0o755); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := run("init", "-C", dir)
		if code != ExitFailed || out != "" || !strings.Contains(errOut, "AGENTS.md: is a link or a folder") {
			t.Fatal(code, out, errOut)
		}
		if got := tree(t, dir); strings.Join(got, " ") != "AGENTS.md/" {
			t.Errorf("a refused init left %v", got)
		}
	})
}

// contents lists every file below dir with its text.
func contents(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		data, err := os.ReadFile(p)
		out[filepath.ToSlash(rel)] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestFailedInitLeavesNothing fails steps of init in turn, the first ones,
// the last ones and a spread between, and checks that the folder is as it
// was, so that init can run again.
func TestFailedInitLeavesNothing(t *testing.T) {
	// The project holds files that init must leave as they are: its AGENTS.md,
	// skills of other names, and documents in the spec folder, among them
	// a package of the library that apply would not overwrite.
	ours := map[string]string{
		"AGENTS.md":                         "# Ours\n",
		".claude/skills/mine/SKILL.md":      "---\nname: mine\n---\n",
		".agents/skills/theirs/SKILL.md":    "---\nname: theirs\n---\n",
		"docs/spec/main/own/README.md":      "# Own\n",
		"docs/spec/main/packages/core/x.md": "# Kept\n",
	}
	laidOut := func() string {
		dir := t.TempDir()
		for p, text := range ours {
			write(t, dir, p, text)
		}
		return dir
	}
	steps := 0
	fault = func(int) error { steps++; return nil }
	t.Cleanup(func() { fault = nil })
	if code, _, errOut := run("init", "-C", laidOut(), "--spec", "docs/spec/main"); code != ExitOK {
		t.Fatal(errOut)
	}
	if steps < 100 {
		t.Fatalf("only %d steps", steps)
	}
	for k := 1; k <= steps; k++ {
		if k > 12 && k < steps-3 && k%(steps/30) != 0 {
			continue
		}
		dir := laidOut()
		fault = func(step int) error {
			if step == k {
				return errors.New("injected")
			}
			return nil
		}
		if code, out, errOut := run("init", "-C", dir, "--spec", "docs/spec/main"); code != ExitFailed || out != "" || !strings.Contains(errOut, "injected") {
			t.Errorf("step %d: %d %q %q", k, code, out, errOut)
		}
		if got := contents(t, dir); !equalMaps(got, ours) {
			t.Errorf("step %d left %v, want %v", k, got, ours)
		}
		if got := tree(t, dir); len(got) != len(ours)+countDirs(ours) {
			t.Errorf("step %d left folders: %v", k, got)
		}
	}
}

// countDirs is the number of distinct folders the files of ours lie in,
// the project folder left out.
func countDirs(files map[string]string) int {
	dirs := map[string]bool{}
	for p := range files {
		for d := path.Dir(p); d != "."; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	return len(dirs)
}

func TestInitModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix modes")
	}
	dir := t.TempDir()
	write(t, dir, "AGENTS.md", "# Ours\n")
	if err := os.Chmod(filepath.Join(dir, "AGENTS.md"), 0o640); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run("init", "-C", dir); code != ExitOK {
		t.Fatal(errOut)
	}
	err := filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		want := os.FileMode(0o644)
		switch {
		case rel == "AGENTS.md":
			want = 0o640 // the mode the file had
		case e.IsDir():
			want = 0o755
		}
		if info.Mode().Perm() != want {
			t.Errorf("%s has mode %v, want %v", path.Clean(filepath.ToSlash(rel)), info.Mode().Perm(), want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
