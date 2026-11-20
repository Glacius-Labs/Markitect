package cli

import (
	"io"
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
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	t, err := document(w.Spec, o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	title := ""
	if len(o.positional) > 2 {
		title = o.positional[2]
	}
	p, changes, err := edit.NewDocument(w.Spec, w.SpecFS, t.Path, o.positional[1], title)
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(w.specDir(), changes); err != nil {
		return ExitFailed, err
	}
	return output(o, stdout, newResult{newHeader("new"), p}, "created "+p+"\n")
}

func move(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	m, err := edit.MoveDocument(w.Spec, w.SpecFS, o.positional[0], o.positional[1])
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(w.specDir(), m.Changes); err != nil {
		return ExitFailed, err
	}
	text := "moved " + m.From + " -> " + m.To + "\n"
	for _, p := range m.Changed {
		text += "changed " + p + "\n"
	}
	return output(o, stdout, moveResult{newHeader("move"), m.From, m.To, append([]string{}, m.Changed...)}, text)
}

func apply(o options, stdout io.Writer) (int, error) {
	dir, err := mustProject(o)
	if err != nil {
		return ExitFailed, err
	}
	folder, err := specFolderOf(dir)
	if err != nil {
		return ExitFailed, err
	}
	target := ""
	if len(o.positional) > 1 {
		target = o.positional[1]
	}
	a, err := edit.Apply(os.DirFS(folder), library.FS(), o.positional[0], target)
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
