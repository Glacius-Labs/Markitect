package lang

import (
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestParsePathPatternRefusals(t *testing.T) {
	for text, want := range map[string]string{
		"":              "a pattern is not empty",
		"   ":           "a pattern is not empty",
		`src\shop`:      `a pattern separates folders with /, not \`,
		`\src`:          `a pattern separates folders with /, not \`,
		"C:/src":        "a pattern is relative to the project folder and has no drive",
		"c:src":         "a pattern is relative to the project folder and has no drive",
		"/src/shop":     "a pattern is relative to the project folder, not absolute",
		"/":             "a pattern is relative to the project folder, not absolute",
		"../src":        "a pattern stays below the project folder, without ..",
		"src/../x":      "a pattern stays below the project folder, without ..",
		"src/..":        "a pattern stays below the project folder, without ..",
		"src//shop":     "a pattern has no empty folder name and no .",
		"src//":         "a pattern has no empty folder name and no .",
		"./src":         "a pattern has no empty folder name and no .",
		"src/./shop.go": "a pattern has no empty folder name and no .",
		"src/[a-/x.go":  "a pattern uses the wildcards *, ? and [a-z]",
		"src/x[.go":     "a pattern uses the wildcards *, ? and [a-z]",
	} {
		if _, err := ParsePathPattern(text); err == nil || err.Error() != want {
			t.Errorf("%q: got %v, want %q", text, err, want)
		}
	}
}

func TestPathPatternNames(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		yes, no []string
	}{
		// A pattern without a trailing slash names the files whose whole path matches.
		{"src/shop/*.go", []string{"src/shop/a.go", "src/shop/.go", "src/shop/a.b.go"}, []string{"src/shop/x/a.go", "src/shop/a.GO", "src/shop/a.gox", "src/a.go", "x/src/shop/a.go", "src/shop"}},
		// A trailing slash names the folder and everything below it.
		{"src/shop/", []string{"src/shop/a.go", "src/shop/x/y/z.txt"}, []string{"src/shop", "src/shopx/a.go", "src/a.go", "x/src/shop/a.go"}},
		{"src/shop", []string{"src/shop"}, []string{"src/shop/a.go", "src/shopx"}},
		// A star stays within one segment; a name with spaces and dots is a name.
		{"*/main.go", []string{"cmd/main.go"}, []string{"main.go", "a/b/main.go"}},
		{"docs/my file (1).txt", []string{"docs/my file (1).txt"}, []string{"docs/my  file (1).txt", "docs/my file (1)xtxt"}},
		{"v1.2/lib.so.1", []string{"v1.2/lib.so.1"}, []string{"v1x2/lib.so.1", "v1.2/lib.so.2"}},
		// Classes and the single character.
		{"src/[a-c]?.go", []string{"src/ab.go", "src/c1.go"}, []string{"src/d1.go", "src/a.go", "src/abc.go"}},
		{"src/[^a-c]*.go", []string{"src/d.go", "src/zeta.go"}, []string{"src/a.go", "src/b1.go"}},
		// Case counts.
		{"Src/Shop.go", []string{"Src/Shop.go"}, []string{"src/shop.go", "Src/shop.go"}},
		// A folder pattern with a wildcard.
		{"*/gen/", []string{"a/gen/x.go", "b/gen/y/z.go"}, []string{"gen/x.go", "a/b/gen/x.go", "a/gen"}},
	} {
		p, err := ParsePathPattern(tc.pattern)
		if err != nil {
			t.Fatalf("%q: %v", tc.pattern, err)
		}
		for _, f := range tc.yes {
			if !p.Names(f) {
				t.Errorf("%q should name %q", tc.pattern, f)
			}
		}
		for _, f := range tc.no {
			if p.Names(f) {
				t.Errorf("%q should not name %q", tc.pattern, f)
			}
		}
	}
}

func TestPathPatternIs(t *testing.T) {
	p, err := ParsePathPattern("src/gen/")
	if err != nil {
		t.Fatal(err)
	}
	// Is compares the path of a file or a folder with the pattern as written, without
	// the trailing slash: the caller decides whether a folder is wanted.
	if !p.Folder || !p.Is("src/gen") || p.Is("src/gen/x") || p.Is("x/src/gen") {
		t.Errorf("%+v", p)
	}
}

func realizedFiles(realized string) map[string]string {
	return map[string]string{
		"shop/README.md": "# Shop\n\nThe shop.\n\n## Realized in\n" + realized + "\n## Contents\n- `*.md`\n",
	}
}

func TestRealizedInPatternErrors(t *testing.T) {
	const wildcards = "a pattern uses the wildcards *, ? and [a-z]"
	files := realizedFiles("- `src/shop/*.go`\n- `/abs/x`\n- `C:/x`\n- `src/../x`\n- `src\\shop`\n- `src/x[`\n- `src//x`\n- ` `\n- `src/shop/`\n")
	expect(t, checkExactly(t, files),
		`shop/README.md:7: readme.pattern: bad pattern "/abs/x": a pattern is relative to the project folder, not absolute`,
		`shop/README.md:8: readme.pattern: bad pattern "C:/x": a pattern is relative to the project folder and has no drive`,
		`shop/README.md:9: readme.pattern: bad pattern "src/../x": a pattern stays below the project folder, without ..`,
		`shop/README.md:10: readme.pattern: bad pattern "src\shop": a pattern separates folders with /, not \`,
		`shop/README.md:11: readme.pattern: bad pattern "src/x[": `+wildcards,
		`shop/README.md:12: readme.pattern: bad pattern "src//x": a pattern has no empty folder name and no .`,
		`shop/README.md:13: readme.pattern: bad pattern " ": a pattern is not empty`)
}

// Every section of that name counts, whatever its case; only a README is read;
// code blocks, and code outside the section, are no patterns; check does not
// look whether a path exists.
func TestRealizedInSections(t *testing.T) {
	files := map[string]string{
		"shop/README.md": "# Shop\n\nSee `a\\b` here.\n\n## realized IN\n- `x/[`\n\n```\n`y/[`\n```\n\n    `z/[`\n\n## Contents\n- `*.md`\n- `c/[`\n\n## Realized in\n`w/[` and `never/existing/path.go`\n",
		"shop/notes.md":  "# Notes\n\n## Realized in\n`/abs`\n",
		"README.md":      "# Root\n\n## Realized in\nNo code here.\n",
	}
	expect(t, checkExactly(t, files),
		`shop/README.md:6: readme.pattern: bad pattern "x/[": a pattern uses the wildcards *, ? and [a-z]`,
		`shop/README.md:19: readme.pattern: bad pattern "w/[": a pattern uses the wildcards *, ? and [a-z]`)
}

func TestSpecRealizedIn(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md":            {Data: []byte("# Root\n\n## Realized in\n- `cmd/`\n")},
		"a.md":                 {Data: []byte("# A\n")},
		"shop/README.md":       {Data: []byte("# Shop\n\n## Realized in\n- `src/shop/*.go`\n- `bad[`\n\n## Realized in\n`docs/`\n")},
		"shop/x.md":            {Data: []byte("# X\n")},
		"nore/x.md":            {Data: []byte("# X\n")},
		"empty/README.md":      {Data: []byte("# Empty\n\n## Realized in\nBuilt somewhere.\n")},
		"transparent/sub/a.md": {Data: []byte("# A\n")},
	}
	s, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := s.Modules(), []string{".", "empty", "nore", "shop", "transparent/sub"}; !reflect.DeepEqual(got, want) {
		t.Errorf("modules %v", got)
	}
	texts := func(m string) string {
		var out []string
		for _, p := range s.RealizedIn(m) {
			out = append(out, p.Text)
		}
		return strings.Join(out, " ")
	}
	for m, want := range map[string]string{".": "cmd/", "shop": "src/shop/*.go docs/", "empty": "", "nore": "", "nowhere": ""} {
		if got := texts(m); got != want {
			t.Errorf("%s: %q, want %q", m, got, want)
		}
	}
}

// A name that starts with a dot is always left out of the files of a project, so
// a pattern with one names nothing.
func TestRealizedInDotNames(t *testing.T) {
	const dot = "a name that starts with a dot is always left out, so a pattern with one names nothing"
	files := realizedFiles("- `.github/workflows/`\n- `src/.env`\n- `.*`\n- `src/*.go`\n- `a.b/c.d`\n- `.`\n")
	expect(t, checkExactly(t, files),
		`shop/README.md:6: readme.pattern: bad pattern ".github/workflows/": `+dot,
		`shop/README.md:7: readme.pattern: bad pattern "src/.env": `+dot,
		`shop/README.md:8: readme.pattern: bad pattern ".*": `+dot,
		`shop/README.md:11: readme.pattern: bad pattern ".": a pattern has no empty folder name and no .`)
}
