package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/edit"
)

// fault, when set by a test, is called before each step that changes the
// folder and fails that step when it returns an error.
var fault func(step int) error

// applyChanges carries a change set out in the folder, all or nothing. It
// checks first that every move and every created file meets nothing, that
// every file to rewrite can be written and that every target folder can be
// written in. It then writes each new content to a temporary file beside its
// target, makes the moves and renames the temporary files into place, keeping
// each replaced file aside until the end. When a step fails, every step done
// is undone.
func applyChanges(folder string, c edit.Changes) (err error) {
	local := func(p string) string { return filepath.Join(folder, filepath.FromSlash(p)) }
	if err := precheck(folder, c, local); err != nil {
		return err
	}
	t := &transaction{}
	defer func() {
		if err != nil {
			t.rollback()
		} else {
			t.commit()
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

// transaction records how to undo each step it made.
type transaction struct {
	step    int
	undo    []func()
	cleanup []func()
}

func (t *transaction) next() error {
	t.step++
	if fault != nil {
		return fault(t.step)
	}
	return nil
}

func (t *transaction) rollback() {
	for i := len(t.undo) - 1; i >= 0; i-- {
		t.undo[i]()
	}
}

func (t *transaction) commit() {
	for _, f := range t.cleanup {
		f()
	}
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
	t.undo = append(t.undo, func() { os.Remove(dir) })
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
	t.undo = append(t.undo, func() { os.Remove(name) })
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
	t.undo = append(t.undo, func() { os.Rename(to, from) })
	return nil
}

// place renames a temporary file to path. A file already at path is kept
// aside until the end, unless create says that none may be there.
func (t *transaction) place(temp, path string, create bool) error {
	if _, err := os.Lstat(path); err == nil {
		if create {
			return fmt.Errorf("%s: already exists", path)
		}
		f, err := os.CreateTemp(filepath.Dir(path), ".markitect-*.bak")
		if err != nil {
			return err
		}
		backup := f.Name()
		f.Close()
		if err := t.next(); err != nil {
			os.Remove(backup)
			return err
		}
		if err := os.Rename(path, backup); err != nil {
			os.Remove(backup)
			return err
		}
		t.undo = append(t.undo, func() { os.Rename(backup, path) })
		t.cleanup = append(t.cleanup, func() { os.Remove(backup) })
	}
	if err := t.next(); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		return err
	}
	t.undo = append(t.undo, func() { os.Remove(path) })
	return nil
}
