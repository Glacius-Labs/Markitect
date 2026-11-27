package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

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

// modelInside refuses a model folder that, through a symbolic link or a
// junction, lies outside the project folder dir. The model folder exists: the
// project is loaded before this is asked.
func modelInside(dir, model string) error {
	project, err := realPath(dir, 0)
	if err != nil {
		return err
	}
	real, err := realPath(filepath.Join(dir, filepath.FromSlash(model)), 0)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(project, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("the model folder %s lies outside the project, at %s", model, real)
	}
	return nil
}
