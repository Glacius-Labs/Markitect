package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// isProject says whether dir holds .markitect/version, so that a .markitect/
// folder of another kind, such as the one in the home folder, is no project.
func isProject(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(project.VersionFile)))
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
// must hold .markitect/version, or the nearest folder from the working
// directory upward that holds it. found is false when there is none.
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
		return fmt.Errorf("%s holds no %s; markitect init makes one", o.dir, project.VersionFile)
	}
	return fmt.Errorf("no folder from %s upward holds %s; markitect init makes one", at, project.VersionFile)
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
	*project.Project
}

// load loads the project a command works in.
func load(o options) (*workspace, error) {
	dir, err := mustProject(o)
	if err != nil {
		return nil, err
	}
	return loadDir(dir)
}

// loadDir loads the project at dir.
func loadDir(dir string) (*workspace, error) {
	p, err := project.Load(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("the project %s: %w", dir, err)
	}
	if err := specInside(dir, p.SpecFolder); err != nil {
		return nil, err
	}
	return &workspace{dir: dir, Project: p}, nil
}

// specDir is the spec folder on disk.
func (w *workspace) specDir() string { return filepath.Join(w.dir, filepath.FromSlash(w.SpecFolder)) }

// versionResult is the JSON document of version. Build is the version with
// the source revision, when the build knows it; Pinned is the version the
// project pins, "" outside a project.
type versionResult struct {
	header
	Version string `json:"version"`
	Build   string `json:"build"`
	Pinned  string `json:"pinned"`
	Matches bool   `json:"matches"`
}

// version prints the tool's build, its version with the source revision,
// and, in a project, the version it pins. It exits 1 when the version and the
// pinned one differ.
func version(o options, stdout io.Writer) (int, error) {
	doc := versionResult{header: newHeader("version"), Version: buildinfo.Version(), Build: buildinfo.Build()}
	text := "markitect " + doc.Build + "\n"
	dir, found, err := projectFolder(o)
	switch {
	case err != nil:
		return ExitFailed, err
	case !found && o.dir != "":
		return ExitFailed, noProject(o, dir)
	case found:
		layout, err := project.ReadLayout(os.DirFS(dir))
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

// licensesResult is the JSON document of licenses.
type licensesResult struct {
	header
	Notices string `json:"notices"`
}

// licenses prints the license notices of the modules the tool is built from.
func licenses(o options, stdout io.Writer) (int, error) {
	return output(o, stdout, licensesResult{newHeader("licenses"), buildinfo.Notices}, buildinfo.Notices)
}

// specFolderOf is the spec folder of the project at dir, which apply creates
// when it is missing.
func specFolderOf(dir string) (string, error) {
	layout, err := project.ReadLayout(os.DirFS(dir))
	if err != nil {
		return "", fmt.Errorf("the project %s: %w", dir, err)
	}
	if err := specInside(dir, layout.SpecFolder); err != nil {
		return "", err
	}
	folder := filepath.Join(dir, filepath.FromSlash(layout.SpecFolder))
	if info, err := os.Stat(folder); err == nil && !info.IsDir() || err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("the spec folder %s, which %s names, is no folder", layout.SpecFolder, project.SpecFile)
	}
	return folder, nil
}
