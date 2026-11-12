package skills

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// codeGroups are the first parts of the codes markitect prints: a diagnostic's
// code is a group, a dot and a name.
const codeGroups = "definition|spec|kind|field|part|library|manifest|lock|link|package|project|registry|file|ignore|harness|record|stale|vet|doc"

var (
	codeInSource = regexp.MustCompile(`"((?:` + codeGroups + `)\.[a-z_]+)"`)
	// addCall finds a diagnostic added with its code as a constant, whatever its
	// group, and anyCode a backticked name of the form of a code.
	addCall         = regexp.MustCompile(`\.Add\("([a-z]+)\.[a-z_]+"`)
	anyCode         = regexp.MustCompile("`([a-z]+\\.[a-z_]+)`")
	codeInReference = regexp.MustCompile("`((?:" + codeGroups + ")\\.[a-z_]+)`")
	// A file name has the same form as a code, such as project.md.
	fileName = regexp.MustCompile(`\.(md|yaml|json|go|sum|txt)$`)
)

// The architect skill's reference lists every code markitect prints, and no code
// it does not print: an agent that meets a code looks it up there, and one that
// is told of a code the tool does not have looks for an error that cannot
// happen. The codes are read from the source as its string constants, so a new
// code fails here until the reference names it.
func TestReferenceListsTheCodesMarkitectPrints(t *testing.T) {
	printed := map[string]string{}
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range codeInSource.FindAllStringSubmatch(string(data), -1) {
			if !fileName.MatchString(m[1]) {
				printed[m[1]] = filepath.ToSlash(path)
			}
		}
		// A code of a group this test does not know would pass unseen.
		for _, m := range addCall.FindAllStringSubmatch(string(data), -1) {
			if !strings.Contains("|"+codeGroups+"|", "|"+m[1]+"|") {
				t.Errorf("%s adds a diagnostic of the group %q, which codeGroups does not name", filepath.ToSlash(path), m[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(printed) < 50 {
		t.Fatalf("found only %d codes in the source; the search no longer fits it", len(printed))
	}
	reference, err := os.ReadFile("claude/skills/markitect-architect/reference.md")
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, m := range codeInReference.FindAllStringSubmatch(string(reference), -1) {
		if !fileName.MatchString(m[1]) {
			listed[m[1]] = true
		}
	}
	var missing, unknown []string
	for code, file := range printed {
		if !listed[code] {
			missing = append(missing, code+" ("+file+")")
		}
	}
	for code := range listed {
		if _, ok := printed[code]; !ok {
			unknown = append(unknown, code)
		}
	}
	sort.Strings(missing)
	sort.Strings(unknown)
	if len(missing) > 0 {
		t.Errorf("the reference does not list: %s", strings.Join(missing, ", "))
	}
	if len(unknown) > 0 {
		t.Errorf("the reference lists codes markitect does not print: %s", strings.Join(unknown, ", "))
	}
	// Below its heading on warnings, the reference names codes and nothing else
	// of their form, so a code of no known group is caught here too.
	_, codes, found := strings.Cut(string(reference), "\n## Warnings and notes")
	if !found {
		t.Fatal("the reference has no section on warnings and notes, where its codes begin")
	}
	for _, m := range anyCode.FindAllStringSubmatch(codes, -1) {
		if _, ok := printed[m[1]]; !ok && !fileName.MatchString(m[1]) {
			t.Errorf("the reference lists %s, which markitect does not print", m[1])
		}
	}
	// The skill itself names only codes the reference explains.
	skill, err := os.ReadFile("claude/skills/markitect-architect/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range codeInReference.FindAllStringSubmatch(string(skill), -1) {
		if !fileName.MatchString(m[1]) && !listed[m[1]] {
			t.Errorf("the skill names %s, which the reference does not list", m[1])
		}
	}
}
