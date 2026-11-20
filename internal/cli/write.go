package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/edit"
	"github.com/Glacius-Labs/Markitect/internal/library"
)

// newResult, moveResult and applyResult are the JSON documents of the commands
// that write, with the paths below the spec folder.
type newResult struct {
	header
	Path string `json:"path"`
}

type moveResult struct {
	header
	From    string   `json:"from"`
	To      string   `json:"to"`
	Changed []string `json:"changed"`
}

type applyResult struct {
	header
	Documents []string `json:"documents"`
	Packages  []string `json:"packages"`
}

func newDocument(o options, stdout io.Writer) (int, error) {
	folder := o.positional[0]
	spec, fsys, err := load(folder)
	if err != nil {
		return ExitFailed, err
	}
	t, err := document(spec, o.positional[1])
	if err != nil {
		return ExitFailed, err
	}
	title := ""
	if len(o.positional) > 3 {
		title = o.positional[3]
	}
	p, changes, err := edit.NewDocument(spec, fsys, t.Path, o.positional[2], title)
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(folder, changes); err != nil {
		return ExitFailed, err
	}
	return output(o, stdout, newResult{newHeader("new"), p}, "created "+p+"\n")
}

func move(o options, stdout io.Writer) (int, error) {
	folder := o.positional[0]
	spec, fsys, err := load(folder)
	if err != nil {
		return ExitFailed, err
	}
	m, err := edit.MoveDocument(spec, fsys, o.positional[1], o.positional[2])
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(folder, m.Changes); err != nil {
		return ExitFailed, err
	}
	text := "moved " + m.From + " -> " + m.To + "\n"
	for _, p := range m.Changed {
		text += "changed " + p + "\n"
	}
	return output(o, stdout, moveResult{newHeader("move"), m.From, m.To, append([]string{}, m.Changed...)}, text)
}

func apply(o options, stdout io.Writer) (int, error) {
	folder := o.positional[0]
	if info, err := os.Stat(folder); err == nil && !info.IsDir() || err != nil && !errors.Is(err, fs.ErrNotExist) {
		return ExitFailed, fmt.Errorf("not a folder: %s", folder)
	}
	target := ""
	if len(o.positional) > 2 {
		target = o.positional[2]
	}
	a, err := edit.Apply(os.DirFS(folder), library.FS(), o.positional[1], target)
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(folder, a.Changes); err != nil {
		return ExitFailed, err
	}
	var b strings.Builder
	for _, p := range a.Documents {
		b.WriteString("copied " + p + "\n")
	}
	for _, p := range a.Packages {
		b.WriteString("copied packages/" + p + "/\n")
	}
	return output(o, stdout, applyResult{newHeader("apply"), append([]string{}, a.Documents...), append([]string{}, a.Packages...)}, b.String())
}
