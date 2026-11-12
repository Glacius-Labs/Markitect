package bootstrap

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// examplePage is the worked example of the architect skill, which an agent that
// writes a spec copies from.
const examplePage = "../skills/claude/skills/markitect-architect/examples.md"

var (
	// exampleFile finds the path a line gives the block after it: a file of the
	// example under .markitect/.
	exampleFile = regexp.MustCompile("^`((?:spec|registry)/[^`]+|registry\\.md|manifest\\.yaml|ignore)`")
	// registryRow finds the file a row of a registry page lists.
	registryRow = regexp.MustCompile("(?m)^\\| `([^`]+)` \\|")
)

// exampleFiles returns the files the page shows, by their path under
// .markitect/: each is a fenced block whose line before it starts with the
// file's path in backticks.
func exampleFiles(t *testing.T, page string) map[string]string {
	t.Helper()
	files := map[string]string{}
	var before, path string
	var body []string
	in := false
	for _, line := range strings.Split(page, "\n") {
		switch {
		case strings.HasPrefix(line, "```") && !in:
			in, body, path = true, nil, ""
			if m := exampleFile.FindStringSubmatch(before); m != nil {
				path = m[1]
			}
		case strings.HasPrefix(line, "```"):
			in, before = false, ""
			if path == "" {
				continue
			}
			if _, twice := files[path]; twice {
				t.Fatalf("the page shows %s twice", path)
			}
			files[path] = strings.Join(body, "\n") + "\n"
		case in:
			body = append(body, line)
		case strings.TrimSpace(line) != "":
			before = line
		}
	}
	return files
}

// The example is a project that compiles, and the page shows what the tools
// print for it: the summary of check, the list of status after the change it
// describes, and each error it warns of, word for word. So an agent that copies
// from the page copies what the compiler accepts, and a change of a message or
// of the Libraries fails here until the page follows.
func TestArchitectExampleIsWhatTheToolsPrint(t *testing.T) {
	emptyHome(t)
	data, err := os.ReadFile(examplePage)
	if err != nil {
		t.Fatal(err)
	}
	page := strings.ReplaceAll(string(data), "\r\n", "\n")
	files := exampleFiles(t, page)
	if len(files) != 19 {
		t.Fatalf("the page shows %d files of the example, want the manifest, 14 Definitions, 3 registry pages and the ignore file", len(files))
	}
	// build writes the example with a change into a folder, and every file its
	// registry pages and its ignore file name.
	build := func(change func(files map[string]string)) *plainFolder {
		t.Helper()
		own := map[string]string{}
		for path, text := range files {
			own[path] = text
		}
		if change != nil {
			change(own)
		}
		r := newFolder(t)
		for path, text := range own {
			r.Write(".markitect/"+path, text)
			if strings.HasPrefix(path, "registry") {
				for _, row := range registryRow.FindAllStringSubmatch(text, -1) {
					r.Write(row[1], "x\n")
				}
			}
		}
		for _, line := range strings.Split(own["ignore"], "\n") {
			if line != "" && !strings.HasPrefix(line, "#") {
				r.Write(line, "x\n")
			}
		}
		mustRun(t, r.Dir, "tidy")
		return r
	}
	shown := func(what, text string) {
		t.Helper()
		if !strings.Contains(page, text) {
			t.Errorf("%s: the page does not show what the tool prints:\n%s", what, text)
		}
	}
	listed := func(r *plainFolder) []string {
		t.Helper()
		var st struct {
			Items []struct{ Definition, Fingerprint string }
		}
		_, out, _ := run(t, r.Dir, "status", "--json")
		if err := json.Unmarshal([]byte(out), &st); err != nil {
			t.Fatalf("status --json: %v\n%s", err, out)
		}
		var items []string
		for _, it := range st.Items {
			items = append(items, it.Definition+"@"+it.Fingerprint)
		}
		return items
	}

	r := build(nil)
	shown("check", "$ markitect check\n"+strings.TrimSpace(mustRun(t, r.Dir, "check"))+"\n")
	if code, out, _ := run(t, r.Dir, "vet"); code != 0 || strings.Contains(out, ": vet.") {
		t.Errorf("vet warns about the example: %d\n%s", code, out)
	}

	// Verified once, the change of the limit the page describes lists what it
	// says.
	for _, item := range listed(r) {
		mustRun(t, r.Dir, "mark", item, "ok", "-e", "README.md", "--cases", "1", "--covered", "1")
	}
	mustRun(t, r.Dir, "status")
	const limit = ".markitect/spec/members/loan-limit.md"
	r.Write(limit, strings.Replace(read(t, r.Dir, limit), "`3`", "`5`", 1))
	var changed []string
	for _, item := range listed(r) {
		changed = append(changed, item[:strings.Index(item, "@")])
	}
	if got := strings.Join(changed, ", "); got != "catalog/borrow-book, members/loan-limit" {
		t.Errorf("after the limit changed, status lists %s", got)
	}
	shown("status", "`markitect status` lists `members/loan-limit` and `catalog/borrow-book`")

	// Each error the page warns of, from the change it names.
	for code, c := range map[string]struct {
		says   string
		change func(files map[string]string)
		file   string
	}{
		"package.cycle": {says: "The books a member holds are listed in [[catalog/book]].", change: func(f map[string]string) {
			f["spec/members/member.md"] += "- The books a member holds are listed in [[catalog/book]].\n"
		}},
		"link.downward": {says: "Staff lend books with [[catalog/post-loans]].", change: func(f map[string]string) {
			f["spec/project.md"] += "\nStaff lend books with [[catalog/post-loans]].\n"
		}},
		"link.internal": {says: "- The receipt goes to the member's [[members/email]].", change: func(f map[string]string) {
			f["spec/catalog/borrow-book.md"] += "- The receipt goes to the member's [[members/email]].\n"
		}},
		"registry.wrong_owner": {says: "| `internal/members/http.go` | [[members/post-members]] | The handler |", change: func(f map[string]string) {
			f["registry/catalog.md"] += "| `internal/members/http.go` | [[members/post-members]] | The handler |\n"
		}},
		"file.unregistered": {says: "`newfile.txt`", file: "newfile.txt"},
	} {
		t.Run(code, func(t *testing.T) {
			if !strings.Contains(page, c.says) {
				t.Errorf("the page does not name the change that causes %s: %s", code, c.says)
			}
			r := build(c.change)
			if c.file != "" {
				r.Write(c.file, "x\n")
			}
			exit, out, _ := run(t, r.Dir, "check")
			found := false
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, ": "+code+": ") {
					found = true
					if !strings.Contains(page, line) {
						t.Errorf("the page does not show what check prints:\n%s", line)
					}
				}
			}
			if exit != 1 || !found {
				t.Errorf("check does not report %s: %d\n%s", code, exit, out)
			}
		})
	}
}
