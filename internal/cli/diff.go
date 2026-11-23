package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// projectAt is a project folder named on the command line, which must hold
// .markitect/spec.
func projectAt(arg string) (string, error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return "", fmt.Errorf("not a folder: %s", arg)
	}
	if !isProject(abs) {
		return "", notAProject(arg, abs)
	}
	return abs, nil
}

// diffDocument is the JSON document of diff.
type diffDocument struct {
	header
	New     []string      `json:"new"`
	Gone    []string      `json:"gone"`
	Changed []changedJSON `json:"changed"`
	// Modules are the modules whose spec or files changed; Unassigned the
	// changed files that no module names.
	Modules    []moduleJSON   `json:"modules"`
	Unassigned unassignedJSON `json:"unassigned"`
}

// moduleJSON is a module with a change; Module is the folder below the spec
// folder, "." for the spec folder itself.
type moduleJSON struct {
	Module       string   `json:"module"`
	Spec         bool     `json:"spec"`
	Files        bool     `json:"files"`
	NewFiles     []string `json:"new_files"`
	GoneFiles    []string `json:"gone_files"`
	ChangedFiles []string `json:"changed_files"`
}

// unassignedJSON are the changed files that no module names, each with how it
// changed, and how many files of NEW no module names in all.
type unassignedJSON struct {
	Files []unassignedFileJSON `json:"files"`
	Total int                  `json:"total"`
}

type unassignedFileJSON struct {
	File   string `json:"file"`
	Change string `json:"change"`
}

type changedJSON struct {
	Doc  string `json:"doc"`
	Diff string `json:"diff"`
}

// diff compares the spec of the project OLD with that of NEW, by default the
// project the command works in.
func diff(o options, stdout io.Writer) (int, error) {
	oldDir, err := projectAt(o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	var newDir string
	if len(o.positional) > 1 {
		newDir, err = projectAt(o.positional[1])
	} else {
		newDir, err = mustProject(o)
	}
	if err != nil {
		return ExitFailed, err
	}
	older, err := loadDir(oldDir)
	if err != nil {
		return ExitFailed, err
	}
	newer, err := loadDir(newDir)
	if err != nil {
		return ExitFailed, err
	}
	ignore, err := project.ReadIgnore(newer.FS)
	if err != nil {
		return ExitFailed, err
	}
	d, err := project.DiffProjects(older, newer, ignore)
	if err != nil {
		return ExitFailed, err
	}
	c := d.Spec
	doc := diffDocument{header: newHeader("diff"), New: nonNil(c.New), Gone: nonNil(c.Gone), Changed: []changedJSON{},
		Modules: []moduleJSON{}, Unassigned: unassignedJSON{Files: []unassignedFileJSON{}, Total: d.UnassignedTotal}}
	var b strings.Builder
	section := func(title string, n int) {
		fmt.Fprintf(&b, "## %s (%d)\n\n", title, n)
		if n == 0 {
			b.WriteString("(none)\n")
		}
	}
	b.WriteString("# Difference\n\n")
	section("New", len(c.New))
	for _, p := range c.New {
		b.WriteString(p + "\n")
	}
	b.WriteString("\n")
	section("Gone", len(c.Gone))
	for _, p := range c.Gone {
		b.WriteString(p + "\n")
	}
	b.WriteString("\n")
	section("Changed", len(c.Changed))
	for _, ch := range c.Changed {
		doc.Changed = append(doc.Changed, changedJSON{ch.Doc, ch.Diff})
		b.WriteString(ch.Doc + "\n" + ch.Diff)
	}
	b.WriteString("\n")
	section("Modules", len(d.Modules))
	for i, m := range d.Modules {
		if i > 0 {
			b.WriteString("\n")
		}
		doc.Modules = append(doc.Modules, moduleJSON{Module: m.Module, Spec: m.Spec, Files: m.Files,
			NewFiles: nonNil(m.New), GoneFiles: nonNil(m.Gone), ChangedFiles: nonNil(m.Changed)})
		fmt.Fprintf(&b, "%s: %s changed\n", moduleName(m.Module), what(m))
		for _, f := range m.Changes() {
			fmt.Fprintf(&b, "  %-7s %s\n", f.Kind, f.Path)
		}
	}
	b.WriteString("\n")
	section("Files no module names", len(d.Unassigned))
	for _, f := range d.Unassigned {
		doc.Unassigned.Files = append(doc.Unassigned.Files, unassignedFileJSON{f.Path, f.Kind})
		fmt.Fprintf(&b, "%-7s %s\n", f.Kind, f.Path)
	}
	fmt.Fprintf(&b, "\nFiles of NEW that no module names: %d\n", d.UnassignedTotal)
	code, err := output(o, stdout, doc, b.String())
	if err == nil && d.Differs() {
		code = ExitFound
	}
	return code, err
}

// moduleName names a module in the text: its folder below the spec folder.
func moduleName(m string) string {
	if m == lang.SpecModule {
		return "(spec folder)"
	}
	return m
}

// what says what changed in a module that has a change.
func what(m project.ModuleChange) string {
	switch {
	case m.Spec && m.Files:
		return "spec and files"
	case m.Spec:
		return "spec"
	}
	return "files"
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
