package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
		return "", fmt.Errorf("%s holds no %s; %s", arg, project.SpecFile, noProjectNote)
	}
	return abs, nil
}

// diffDocument is the JSON document of diff.
type diffDocument struct {
	header
	New     []string      `json:"new"`
	Gone    []string      `json:"gone"`
	Changed []changedJSON `json:"changed"`
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
	c := project.DiffSpecs(older, newer)
	doc := diffDocument{header: newHeader("diff"), New: nonNil(c.New), Gone: nonNil(c.Gone), Changed: []changedJSON{}}
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
	code, err := output(o, stdout, doc, b.String())
	if err == nil && c.Differs() {
		code = ExitFound
	}
	return code, err
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
