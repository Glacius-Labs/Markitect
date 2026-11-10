package project

import (
	"os"
	"regexp"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
)

// The pages that show a manifest, or say at which version the standard Libraries
// ship, name the version this binary ships: a reader who copies another gets
// library.version. So a move of the version fails here until the pages follow.
func TestPagesNameTheShippedLibraryVersion(t *testing.T) {
	named := regexp.MustCompile(`\b(?:(?:std|rest|cli):? |at version )(\d+\.\d+\.\d+)`)
	found := 0
	for _, page := range []string{
		"../skills/markitect-architect/SKILL.md",
		"../skills/markitect-architect/reference.md",
		"../skills/markitect-architect/examples.md",
		"../../plugin/skills/getting-started/SKILL.md",
		"../../docs/guides/spec-language.md",
		"../../docs/guides/project-workflow.md",
		"../../docs/concept.md",
		"../../README.md",
	} {
		data, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range named.FindAllStringSubmatch(string(data), -1) {
			found++
			if m[1] != libraries.Version {
				t.Errorf("%s names a standard Library at %s (%q); this binary ships %s", page, m[1], m[0], libraries.Version)
			}
		}
	}
	if found < 5 {
		t.Fatalf("found only %d places that name the version; the search no longer fits the pages", found)
	}
}
