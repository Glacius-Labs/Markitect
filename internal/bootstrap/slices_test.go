package bootstrap

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// sliceLine finds the command status --slices prints for a slice.
var sliceLine = regexp.MustCompile(`(?m)^\d+\. markitect (status.*)$`)

// A build in slices through the real commands (DEC-112): the test plays the
// session, which takes the slices status --slices prints in their order, and
// the reviewer, who records each item of a slice. Every slice lists only its
// own items, no item is in two slices, and after the last one nothing is
// left.
func TestABuildInSlicesThroughTheCommandLine(t *testing.T) {
	emptyHome(t)
	data, err := os.ReadFile(examplePage)
	if err != nil {
		t.Fatal(err)
	}
	r := newFolder(t)
	for path, text := range exampleFiles(t, strings.ReplaceAll(string(data), "\r\n", "\n")) {
		r.Write(".markitect/"+path, text)
		if strings.HasPrefix(path, "registry") {
			for _, row := range registryRow.FindAllStringSubmatch(text, -1) {
				r.Write(row[1], "x\n")
			}
		}
		if path == "ignore" {
			for _, line := range strings.Split(text, "\n") {
				if line != "" && !strings.HasPrefix(line, "#") {
					r.Write(line, "x\n")
				}
			}
		}
	}
	mustRun(t, r.Dir, "tidy")

	type status struct {
		Clean  bool
		Slice  []string
		Slices []struct {
			Packages []string
			Items    int
			Rest     bool
			Command  string
		}
		Items []struct {
			Definition, Fingerprint string
			Last                    bool
		}
	}
	read := func(args ...string) status {
		t.Helper()
		var st status
		_, out, _ := run(t, r.Dir, append(args, "--json")...)
		if err := json.Unmarshal([]byte(out), &st); err != nil {
			t.Fatalf("%v --json: %v\n%s", args, err, out)
		}
		return st
	}
	whole := read("status")
	if len(whole.Slices) != 1 || !whole.Slices[0].Rest || whole.Slices[0].Command != "markitect status" || whole.Slices[0].Items != len(whole.Items) {
		t.Fatalf("a short list is one slice: %+v", whole.Slices)
	}

	// Cut small, the list goes in several slices, and the text names the
	// same commands as the JSON.
	code, text, _ := run(t, r.Dir, "status", "--slices", "--size", "4")
	if code != 1 || !strings.Contains(text, "slices, one after the other, each built and verified before the next:") {
		t.Fatalf("status --slices: %d\n%s", code, text)
	}
	commands := sliceLine.FindAllStringSubmatch(text, -1)
	if len(commands) < 3 || commands[len(commands)-1][1] != "status" {
		t.Fatalf("the slices:\n%s", text)
	}
	seen := map[string]bool{}
	for i, c := range commands {
		args := strings.Fields(c[1])
		st := read(args...)
		last := i == len(commands)-1
		if !last && len(st.Items) == 0 {
			t.Fatalf("slice %d lists nothing: %v", i+1, args)
		}
		for _, it := range st.Items {
			if seen[it.Definition] {
				t.Errorf("%s is in two slices", it.Definition)
			}
			seen[it.Definition] = true
			if it.Last && !last {
				t.Errorf("%s is verified last and listed by %v", it.Definition, args)
			}
			mustRun(t, r.Dir, "mark", it.Definition+"@"+it.Fingerprint, "ok", "-e", "go.mod", "--cases", "1", "--covered", "1")
		}
		if code, out, _ := run(t, r.Dir, args...); code != 0 || !strings.Contains(out, "Nothing to verify") {
			t.Fatalf("slice %d is not settled: %d\n%s", i+1, code, out)
		}
	}
	if len(seen) != len(whole.Items) {
		t.Fatalf("the slices held %d of %d items", len(seen), len(whole.Items))
	}
	if code, out, _ := run(t, r.Dir, "status", "--slices"); code != 0 || !strings.Contains(out, "Nothing to verify") {
		t.Fatalf("after the last slice: %d\n%s", code, out)
	}

	for _, args := range [][]string{
		{"status", "--slice", "Root", "--slices"},
		{"status", "--slice", "Root", "--package", "Root"},
		{"status", "--slices", "--package", "Root"},
		{"status", "--size", "3"},
		{"status", "--slices", "--size", "-1"},
		{"status", "--slice", ","},
	} {
		if code, _, errOut := run(t, r.Dir, args...); code != 2 || errOut == "" {
			t.Errorf("%v: exit %d, %q", args, code, errOut)
		}
	}
	if code, _, errOut := run(t, r.Dir, "status", "--slice", "nowhere"); code != 2 || !strings.Contains(errOut, `no Package "nowhere"`) {
		t.Errorf("a slice of no Package: %d %s", code, errOut)
	}

	// The spec as one text to read, whole or below a folder, and the Kinds
	// with what the spec has of each.
	if code, out, _ := run(t, r.Dir, "doc", "--read"); code != 0 || !strings.HasPrefix(out, "# The spec\n") || strings.Count(out, "\n### ")+strings.Count(out, "\n#### ") != 26 {
		t.Errorf("doc --read: %d\n%s", code, out)
	}
	if code, out, _ := run(t, r.Dir, "doc", "--read", "purpose"); code != 0 || !strings.HasPrefix(out, "# The spec: purpose\n") || !strings.Contains(out, "\n## purpose/goals\n") || strings.Contains(out, "`domain/") {
		t.Errorf("doc --read purpose: %d\n%s", code, out)
	}
	if code, _, errOut := run(t, r.Dir, "doc", "--read", "nowhere"); code != 1 || !strings.Contains(errOut, "doc.unknown") {
		t.Errorf("doc --read of no folder: %d %s", code, errOut)
	}
	if code, _, _ := run(t, r.Dir, "doc", "--read", "purpose", "domain"); code != 2 {
		t.Errorf("doc --read of two folders: %d", code)
	}
	if code, out, _ := run(t, r.Dir, "list", "--kinds"); code != 0 || !strings.Contains(out, "  std/Goal: ") || !strings.Contains(out, "\nNo Definition yet: ") {
		t.Errorf("list --kinds: %d\n%s", code, out)
	}
	if code, _, _ := run(t, r.Dir, "list", "--kinds", "--deps"); code != 2 {
		t.Errorf("list --kinds --deps: %d", code)
	}
}
