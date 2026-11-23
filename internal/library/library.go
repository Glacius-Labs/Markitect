// Package library is the library built into the tool: the packages of types,
// the templates that apply and init copy into a spec, the skills of the
// standard that init copies into a project, and per package its reference
// values and its question catalog, which the tool reads and never copies.
//
// Its contents are described in docs/guides/library.md.
package library

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed packages templates references questions
var files embed.FS

//go:embed all:skills
var skills embed.FS

// FS returns the library: packages/NAME/ for each package, templates/NAME/
// for each template, and references/NAME.md with the reference values and
// questions/NAME.md with the question catalog of each package. Every file of
// it is a document of the language.
func FS() fs.FS { return files }

// Skills returns the skills of the standard: a folder NAME/ for each, with
// its SKILL.md and whatever else belongs to it. They are plain files, no
// documents of the language, and lie apart from FS so that nothing that loads
// the library as a spec meets them.
func Skills() fs.FS {
	sub, err := fs.Sub(skills, "skills")
	if err != nil {
		panic(err)
	}
	return sub
}

// SkillNames are the names of the skills, sorted: the folders of Skills.
func SkillNames() ([]string, error) {
	entries, err := fs.ReadDir(Skills(), ".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
