package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/state"
)

// projectFS is a project folder on disk as the state reads it: os.DirFS,
// which also reaches a path outside the folder that a .git file names.
type projectFS struct {
	fs.FS
	dir string
}

func diskProject(dir string) projectFS { return projectFS{os.DirFS(dir), dir} }

func (p projectFS) ReadDir(name string) ([]fs.DirEntry, error) { return fs.ReadDir(p.FS, name) }
func (p projectFS) ReadFile(name string) ([]byte, error)       { return fs.ReadFile(p.FS, name) }
func (p projectFS) Stat(name string) (fs.FileInfo, error)      { return fs.Stat(p.FS, name) }
func (p projectFS) Lstat(name string) (fs.FileInfo, error)     { return fs.Lstat(p.FS, name) }
func (p projectFS) ReadLink(name string) (string, error)       { return fs.ReadLink(p.FS, name) }

// Outside returns the volume that holds name, absolute or relative to the
// project folder, and the path of name on it.
func (p projectFS) Outside(name string) (fs.FS, string, bool) {
	abs := filepath.FromSlash(name)
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(p.dir, abs)
	}
	root := filepath.VolumeName(abs) + string(filepath.Separator)
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, "", false
	}
	return os.DirFS(root), filepath.ToSlash(rel), true
}

var _ state.OutsideFS = projectFS{}

// maxLinks is how many links realPath follows before it gives up.
const maxLinks = 255

// realPath resolves every symbolic link and junction in an absolute path. Go's
// filepath.EvalSymlinks does not follow a Windows junction; os.Readlink reads
// both.
func realPath(p string, links int) (string, error) {
	p = filepath.Clean(p)
	vol := filepath.VolumeName(p)
	parts := strings.Split(strings.Trim(p[len(vol):], string(filepath.Separator)), string(filepath.Separator))
	at := vol + string(filepath.Separator)
	for i, part := range parts {
		if part == "" {
			continue
		}
		next := filepath.Join(at, part)
		info, err := os.Lstat(next)
		if err != nil {
			return "", err
		}
		if info.Mode()&fs.ModeSymlink != 0 || info.Mode()&fs.ModeIrregular != 0 {
			if target, err := os.Readlink(next); err == nil {
				if links++; links > maxLinks {
					return "", fmt.Errorf("%s: too many links", p)
				}
				if !filepath.IsAbs(target) {
					target = filepath.Join(at, target)
				}
				return realPath(filepath.Join(append([]string{target}, parts[i+1:]...)...), links)
			}
		}
		at = next
	}
	return at, nil
}

// specInside refuses a spec folder that, through a symbolic link or a
// junction, lies outside the project folder dir. A spec folder that does not
// exist yet is judged by the nearest folder above it that does.
func specInside(dir, spec string) error {
	project, err := realPath(dir, 0)
	if err != nil {
		return err
	}
	at := filepath.Join(dir, filepath.FromSlash(spec))
	for {
		if _, err := os.Lstat(at); err == nil {
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		at = filepath.Dir(at)
	}
	real, err := realPath(at, 0)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(project, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("the spec folder %s lies outside the project, at %s", spec, real)
	}
	return nil
}
