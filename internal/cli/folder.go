package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/edit"
)

// fault, when set by a test, is called before each step that changes the
// folder and fails that step when it returns an error.
var fault func(step int) error

// undoFault, when set by a test, is called before each undo step and fails it
// when it returns an error.
var undoFault func(step int) error

// applyChanges carries a change set out in the folder, all or nothing. It
// checks first that every move and every created file meets nothing, that
// every file to rewrite can be written and that every target folder can be
// written in. It then writes each new content to a temporary file beside its
// target, makes the moves and renames the temporary files into place, keeping
// each replaced file aside until the end. When a step fails, every step done
// is undone; when an undo fails too, the error names the original failure and
// every file left behind, so that the user can restore by hand.
func applyChanges(folder string, c edit.Changes) (err error) {
	local := func(p string) string { return filepath.Join(folder, filepath.FromSlash(p)) }
	if err := precheck(folder, c, local); err != nil {
		return err
	}
	t := &transaction{}
	defer func() {
		if err == nil {
			t.commit()
			return
		}
		if failed := t.rollback(); len(failed) > 0 {
			err = fmt.Errorf("%w; undoing it failed: %s; left behind: %s", err, strings.Join(failed, "; "), strings.Join(t.leftovers(), ", "))
		}
	}()
	var dirs []string
	for _, m := range c.Moves {
		dirs = append(dirs, filepath.Dir(local(m.To)))
	}
	for _, w := range c.Writes {
		dirs = append(dirs, filepath.Dir(local(w.Path)))
	}
	for _, d := range dirs {
		if err := t.mkdirAll(d); err != nil {
			return err
		}
	}
	temps := make([]string, len(c.Writes))
	for i, w := range c.Writes {
		if temps[i], err = t.temp(filepath.Dir(local(w.Path)), w.Data); err != nil {
			return err
		}
	}
	for _, m := range c.Moves {
		if err := t.rename(local(m.From), local(m.To)); err != nil {
			return err
		}
	}
	for i, w := range c.Writes {
		if err := t.place(temps[i], local(w.Path), w.Create); err != nil {
			return err
		}
	}
	return nil
}

// precheck finds what would stop the change set before anything changes.
func precheck(folder string, c edit.Changes, local func(string) string) error {
	movedTo := map[string]string{}
	for _, m := range c.Moves {
		if _, err := os.Lstat(local(m.To)); err == nil {
			return fmt.Errorf("%s: already exists", m.To)
		}
		movedTo[m.To] = m.From
	}
	probed := map[string]error{}
	for _, w := range c.Writes {
		file := local(w.Path)
		if from, ok := movedTo[w.Path]; ok {
			file = local(from)
		}
		_, statErr := os.Lstat(file)
		switch {
		case w.Create && statErr == nil:
			return fmt.Errorf("%s: already exists", w.Path)
		case !w.Create:
			f, err := os.OpenFile(file, os.O_WRONLY, 0)
			if err != nil {
				return fmt.Errorf("%s: cannot be written: %w", w.Path, err)
			}
			f.Close()
		}
		dir := existingAncestor(filepath.Dir(local(w.Path)))
		if _, done := probed[dir]; !done {
			probed[dir] = probe(dir)
		}
		if err := probed[dir]; err != nil {
			return fmt.Errorf("%s: cannot be written: %w", w.Path, err)
		}
	}
	return nil
}

// existingAncestor returns dir or its nearest ancestor that exists.
func existingAncestor(dir string) string {
	for {
		if _, err := os.Lstat(dir); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

// probe checks that a file can be created in dir.
func probe(dir string) error {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("not a folder: %s", dir)
	}
	f, err := os.CreateTemp(dir, ".markitect-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

// transaction records how to undo each step it made, and the temporary and
// backup files it made.
type transaction struct {
	step    int
	undo    []func() error
	cleanup []func()
	files   []string
}

func (t *transaction) next() error {
	t.step++
	if fault != nil {
		return fault(t.step)
	}
	return nil
}

// rollback undoes every step, the last first, and returns what it could not
// undo.
func (t *transaction) rollback() []string {
	var failed []string
	for i := len(t.undo) - 1; i >= 0; i-- {
		err := error(nil)
		if undoFault != nil {
			err = undoFault(len(t.undo) - i)
		}
		if err == nil {
			err = t.undo[i]()
		}
		if err != nil {
			failed = append(failed, err.Error())
		}
	}
	return failed
}

// leftovers returns the temporary and backup files that still exist.
func (t *transaction) leftovers() []string {
	var out []string
	for _, f := range t.files {
		if _, err := os.Lstat(f); err == nil {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return []string{"none of the temporary or backup files"}
	}
	return out
}

func (t *transaction) commit() {
	for _, f := range t.cleanup {
		f()
	}
}

// removeIfThere removes a file and ignores that it is gone already.
func removeIfThere(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// mkdirAll creates dir and its missing ancestors, each to be removed again on
// rollback.
func (t *transaction) mkdirAll(dir string) error {
	if _, err := os.Lstat(dir); err == nil {
		return nil
	}
	if parent := filepath.Dir(dir); parent != dir {
		if err := t.mkdirAll(parent); err != nil {
			return err
		}
	}
	if err := t.next(); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0o755); errors.Is(err, fs.ErrExist) {
		return nil
	} else if err != nil {
		return err
	}
	t.undo = append(t.undo, func() error { return os.Remove(dir) })
	return nil
}

// temp writes data to a new temporary file in dir.
func (t *transaction) temp(dir string, data []byte) (string, error) {
	if err := t.next(); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, ".markitect-*.tmp")
	if err != nil {
		return "", err
	}
	name := f.Name()
	t.files = append(t.files, name)
	t.undo = append(t.undo, func() error { return removeIfThere(name) })
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return name, err
}

func (t *transaction) rename(from, to string) error {
	if err := t.next(); err != nil {
		return err
	}
	if err := os.Rename(from, to); err != nil {
		return err
	}
	t.undo = append(t.undo, func() error { return os.Rename(to, from) })
	return nil
}

// place puts a temporary file at path. A new file is linked there, which
// fails when anything is at path, then the temporary name is removed; where
// the file system has no links, it is created exclusively and filled. A file
// already at path is renamed aside first and kept until the end.
func (t *transaction) place(temp, path string, create bool) error {
	if create {
		if err := t.next(); err != nil {
			return err
		}
		if err := createFrom(temp, path); err != nil {
			return err
		}
		t.undo = append(t.undo, func() error { return os.Remove(path) })
		return removeIfThere(temp)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".markitect-*.bak")
	if err != nil {
		return err
	}
	backup := f.Name()
	f.Close()
	t.files = append(t.files, backup)
	t.cleanup = append(t.cleanup, func() { os.Remove(backup) })
	if err := t.next(); err != nil {
		os.Remove(backup)
		return err
	}
	if err := os.Rename(path, backup); err != nil {
		os.Remove(backup)
		return err
	}
	t.undo = append(t.undo, func() error { return os.Rename(backup, path) })
	if err := t.next(); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		return err
	}
	t.undo = append(t.undo, func() error { return os.Remove(path) })
	return nil
}

// createFrom makes path a new file with the contents of temp and refuses when
// anything is at path.
func createFrom(temp, path string) error {
	err := os.Link(temp, path)
	if err == nil || errors.Is(err, fs.ErrExist) {
		return err
	}
	data, rerr := os.ReadFile(temp)
	if rerr != nil {
		return rerr
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}
