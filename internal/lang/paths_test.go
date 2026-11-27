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
		{"src/shop/*.go", []string{"src/shop/a.go", "src/shop/a.b.go", "src/shop/.go"}, []string{"src/shop/x/a.go", "src/shop/a.GO", "src/shop/a.gox", "src/a.go", "x/src/shop/a.go", "src/shop"}},
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

func TestModelRealizedIn(t *testing.T) {
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

// A name that starts with a dot is a name like any other: * and ? and a class
// match a leading dot, and a folder pattern covers everything below its folder.
func TestPathPatternDotNames(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		yes, no []string
	}{
		{".github/workflows/*.yaml", []string{".github/workflows/ci.yaml", ".github/workflows/.draft.yaml"}, []string{".github/workflows/sub/ci.yaml", ".github/ci.yaml", ".gitlab/workflows/ci.yaml"}},
		{"*/workflows/", []string{".github/workflows/ci.yaml", "app/workflows/.hidden", "app/workflows/.d/x"}, []string{"workflows/x", "a/b/workflows/x"}},
		{".*/x/", []string{".git/x/f", ".github/x/y/z", ".git/x/.d/f"}, []string{".git/y/f", "a/x/f", ".git/x"}},
		{".editorconfig", []string{".editorconfig"}, []string{"x/.editorconfig", ".editorconfig/x"}},
		{".claude/skills/", []string{".claude/skills/triage/SKILL.md", ".claude/skills/.cache/x", ".claude/skills/triage/.hidden"}, []string{".claude/skills", ".claude/worktrees/x"}},
		{"src/*", []string{"src/a", "src/.env"}, []string{"src", "src/a/b"}},
		{"src/?env", []string{"src/.env", "src/aenv"}, nil},
		{"src/[.a]env", []string{"src/.env", "src/aenv"}, []string{"src/benv"}},
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

// ** is a name of its own: at the start any folder, in the middle zero or more
// folders, at the end everything inside.
func TestPathPatternDoubleStar(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		yes, no []string
	}{
		{"**/main.go", []string{"main.go", "cmd/main.go", "a/b/c/main.go"}, []string{"main.go/x", "cmd/main.gox", "cmd/xmain.go"}},
		{"src/**/gen.go", []string{"src/gen.go", "src/a/gen.go", "src/a/b/gen.go"}, []string{"gen.go", "x/src/gen.go", "src/a/gen.gox"}},
		{"src/**", []string{"src/a", "src/a/b/c"}, []string{"src", "other/src/a"}},
		{"**", []string{"a", "a/b"}, nil},
		{"**/gen/", []string{"gen/x", "a/gen/x", "a/b/gen/y/z"}, []string{"gen", "a/gen", "a/genx/x"}},
		{"a/**/b/**/c", []string{"a/b/c", "a/x/b/c", "a/b/y/z/c", "a/x/y/b/z/c"}, []string{"a/c", "b/c", "a/b/d"}},
		{"**/*.go", []string{"a.go", "src/a.go", ".go"}, []string{"a.gox"}},
		// ** inside a name is a star, and two stars match what one does.
		{"src/a**b", []string{"src/ab", "src/aXb"}, []string{"src/x/ab"}},
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
	// check reads ** like any pattern, and reports a bad one as before.
	files := realizedFiles("- `**/*.go`\n- `src/**/x/`\n- `a/**`\n- `**/[`\n")
	expect(t, checkExactly(t, files), `shop/README.md:9: readme.pattern: bad pattern "**/[": a pattern uses the wildcards *, ? and [a-z]`)
}

// A dot name in a Realized in pattern is a pattern like any other.
func TestRealizedInDotNamesAreNoError(t *testing.T) {
	files := realizedFiles("- `.github/workflows/`\n- `src/.env`\n- `.*/x/`\n- `.editorconfig`\n")
	expect(t, checkExactly(t, files))
}
