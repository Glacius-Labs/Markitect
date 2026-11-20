package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/state"
)

// isProject says whether dir holds .markitect/version, so that a .markitect/
// folder of another kind, such as the one in the home folder, is no project.
func isProject(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(state.VersionFile)))
	return err == nil && info.Mode().IsRegular()
}

// findProject returns the nearest folder, start or above it, for which
// isProject holds.
func findProject(start string, isProject func(string) bool) (string, bool) {
	for dir := start; ; {
		if isProject(dir) {
			return dir, true
		}
		up := filepath.Dir(dir)
		if up == dir {
			return "", false
		}
		dir = up
	}
}

// workingFolder is the folder -C names, which must exist, or the working
// directory.
func workingFolder(o options) (string, error) {
	if o.dir == "" {
		return os.Getwd()
	}
	abs, err := filepath.Abs(o.dir)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return "", fmt.Errorf("not a folder: %s", o.dir)
	}
	return abs, nil
}

// projectFolder is the project a command works in: the folder -C names, which
// must hold .markitect/, or the nearest folder from the working directory
// upward that holds it. found is false when there is none.
func projectFolder(o options) (dir string, found bool, err error) {
	at, err := workingFolder(o)
	if err != nil {
		return "", false, err
	}
	if o.dir != "" {
		return at, isProject(at), nil
	}
	dir, found = findProject(at, isProject)
	return dir, found, nil
}

// noProject says that no project was found, and how to make one.
func noProject(o options, at string) error {
	if o.dir != "" {
		return fmt.Errorf("%s holds no %s; markitect-next init makes one", o.dir, state.VersionFile)
	}
	return fmt.Errorf("no folder from %s upward holds %s; markitect-next init makes one", at, state.VersionFile)
}

// mustProject is the project a command works in, or an error when there is
// none.
func mustProject(o options) (string, error) {
	dir, found, err := projectFolder(o)
	if err != nil {
		return "", err
	}
	if !found {
		at, _ := workingFolder(o)
		return "", noProject(o, at)
	}
	return dir, nil
}

// workspace is a loaded project and its folder on disk.
type workspace struct {
	dir string
	*state.Project
}

// load loads the project a command works in.
func load(o options) (*workspace, error) {
	dir, err := mustProject(o)
	if err != nil {
		return nil, err
	}
	p, err := state.Load(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("the project %s: %w", dir, err)
	}
	return &workspace{dir: dir, Project: p}, nil
}

// specDir is the spec folder on disk.
func (w *workspace) specDir() string { return filepath.Join(w.dir, filepath.FromSlash(w.SpecFolder)) }

// initResult is the JSON document of init.
type initResult struct {
	header
	Version string   `json:"version"`
	Spec    string   `json:"spec"`
	Created []string `json:"created"`
}

// initProject sets up the folder -C names, or the working directory: it
// writes .markitect/ and creates the spec folder when it is missing. It
// refuses when .markitect/ exists. It builds .markitect/ under a temporary
// name and renames it into place, so that a failed init leaves nothing behind.
func initProject(o options, stdout io.Writer) (code int, err error) {
	dir, err := workingFolder(o)
	if err != nil {
		return ExitFailed, err
	}
	spec := state.DefaultSpec
	if o.specSet {
		if spec, err = state.SpecPath(o.spec); err != nil {
			return ExitFailed, err
		}
	}
	target := filepath.Join(dir, state.Dir)
	if _, err := os.Lstat(target); err == nil {
		return ExitFailed, fmt.Errorf("%s/ exists already in %s", state.Dir, dir)
	}
	specDir := filepath.Join(dir, filepath.FromSlash(spec))
	info, statErr := os.Stat(specDir)
	if statErr == nil && !info.IsDir() {
		return ExitFailed, fmt.Errorf("the spec folder %s is a file", spec)
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
	created := []string{state.Dir + "/"}
	if statErr != nil {
		if err := t.mkdirAll(specDir); err != nil {
			return ExitFailed, err
		}
		created = append(created, spec+"/")
	}
	if err := t.next(); err != nil {
		return ExitFailed, err
	}
	temp, err := os.MkdirTemp(dir, ".markitect-*.tmp")
	if err != nil {
		return ExitFailed, err
	}
	t.files = append(t.files, temp)
	t.undo = append(t.undo, func() error { return os.RemoveAll(temp) })
	for _, f := range state.InitFiles(buildinfo.Version(), spec) {
		if err := t.next(); err != nil {
			return ExitFailed, err
		}
		name := filepath.Join(temp, filepath.FromSlash(strings.TrimPrefix(f.Path, state.Dir+"/")))
		if err := os.WriteFile(name, f.Data, 0o644); err != nil {
			return ExitFailed, err
		}
	}
	if _, err := os.Lstat(target); err == nil {
		return ExitFailed, fmt.Errorf("%s/ exists already in %s", state.Dir, dir)
	}
	if err := t.rename(temp, target); err != nil {
		return ExitFailed, err
	}
	text := ""
	for _, c := range created {
		text += "created " + c + "\n"
	}
	return output(o, stdout, initResult{newHeader("init"), buildinfo.Version(), spec, created}, text)
}

// versionResult is the JSON document of version. Pinned is the version the
// project pins, "" outside a project.
type versionResult struct {
	header
	Version string `json:"version"`
	Pinned  string `json:"pinned"`
	Matches bool   `json:"matches"`
}

// version prints the tool's version and, in a project, the version it pins.
// It exits 1 when the two differ.
func version(o options, stdout io.Writer) (int, error) {
	doc := versionResult{header: newHeader("version"), Version: buildinfo.Version()}
	text := "markitect-next " + doc.Version + "\n"
	dir, found, err := projectFolder(o)
	switch {
	case err != nil:
		return ExitFailed, err
	case !found && o.dir != "":
		return ExitFailed, noProject(o, dir)
	case found:
		layout, err := state.ReadLayout(os.DirFS(dir))
		if err != nil {
			return ExitFailed, fmt.Errorf("the project %s: %w", dir, err)
		}
		doc.Pinned, doc.Matches = layout.Version, layout.Version == doc.Version
		if doc.Matches {
			text += "the project pins " + doc.Pinned + "\n"
		} else {
			text += "the project pins " + doc.Pinned + ", not this version\n"
		}
	}
	code, err := output(o, stdout, doc, text)
	if err == nil && found && !doc.Matches {
		code = ExitFound
	}
	return code, err
}

// specFolderOf is the spec folder of the project at dir, which apply creates
// when it is missing.
func specFolderOf(dir string) (string, error) {
	layout, err := state.ReadLayout(os.DirFS(dir))
	if err != nil {
		return "", fmt.Errorf("the project %s: %w", dir, err)
	}
	folder := filepath.Join(dir, filepath.FromSlash(layout.SpecFolder))
	if info, err := os.Stat(folder); err == nil && !info.IsDir() || err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("the spec folder %s, which %s names, is no folder", layout.SpecFolder, state.SpecFile)
	}
	return folder, nil
}
