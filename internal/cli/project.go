package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// isProject says whether dir holds .markitect/spec, so that a .markitect/
// folder of another kind, such as the one in the home folder, is no project.
func isProject(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(project.SpecFile)))
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

// noProjectNote says what the file that makes a folder a project holds; no
// command creates it.
const noProjectNote = "that file holds one line, the path of the spec folder below the project folder"

// noProject says that no project was found, and what the missing file holds.
func noProject(o options, at string) error {
	if o.dir != "" {
		return fmt.Errorf("%s holds no %s; %s", o.dir, project.SpecFile, noProjectNote)
	}
	return fmt.Errorf("no folder from %s upward holds %s; %s", at, project.SpecFile, noProjectNote)
}

// mustProject is the project a command works in: the folder -C names, which
// must hold .markitect/spec, or the nearest folder from the working directory
// upward that holds it. It is an error when there is none.
func mustProject(o options) (string, error) {
	at, err := workingFolder(o)
	if err != nil {
		return "", err
	}
	if o.dir != "" {
		if !isProject(at) {
			return "", noProject(o, at)
		}
		return at, nil
	}
	dir, found := findProject(at, isProject)
	if !found {
		return "", noProject(o, at)
	}
	return dir, nil
}

// load loads the project a command works in.
func load(o options) (*project.Project, error) {
	dir, err := mustProject(o)
	if err != nil {
		return nil, err
	}
	return loadDir(dir)
}

// loadDir loads the project at dir.
func loadDir(dir string) (*project.Project, error) {
	p, err := project.Load(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("the project %s: %w", dir, err)
	}
	if err := specInside(dir, p.SpecFolder); err != nil {
		return nil, err
	}
	return p, nil
}

// versionResult is the JSON document of version. Build is the version with
// the source revision, when the build knows it.
type versionResult struct {
	header
	Version string `json:"version"`
	Build   string `json:"build"`
}

// version prints the tool's build: its version with the source revision.
func version(o options, stdout io.Writer) (int, error) {
	doc := versionResult{header: newHeader("version"), Version: buildinfo.Version(), Build: buildinfo.Build()}
	return output(o, stdout, doc, "markitect "+doc.Build+"\n")
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
