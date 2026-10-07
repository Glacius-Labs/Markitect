package recordstore

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type preparedPathSet struct {
	root      string
	forbidden []string
}

func preparePaths(root string, forbiddenRoots []string, allowMissingRoot bool) (preparedPathSet, error) {
	if root == "" || !filepath.IsAbs(root) {
		return preparedPathSet{}, fmt.Errorf("store root must be an absolute path")
	}
	if forbiddenRoots == nil {
		return preparedPathSet{}, fmt.Errorf("Host must explicitly supply forbidden roots; use an empty slice only when none apply")
	}
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return preparedPathSet{}, err
	}
	if len(root) > 32768 {
		return preparedPathSet{}, fmt.Errorf("store root path is too long")
	}
	if err := inspectPathComponents(root, allowMissingRoot); err != nil {
		return preparedPathSet{}, err
	}
	forbidden := make([]string, 0, len(forbiddenRoots))
	for _, path := range forbiddenRoots {
		if path == "" || !filepath.IsAbs(path) {
			return preparedPathSet{}, fmt.Errorf("forbidden root must be absolute: %q", path)
		}
		path, err = filepath.Abs(filepath.Clean(path))
		if err != nil {
			return preparedPathSet{}, err
		}
		if len(path) > 32768 {
			return preparedPathSet{}, fmt.Errorf("forbidden root path is too long")
		}
		if err = inspectPathComponents(path, true); err != nil {
			return preparedPathSet{}, fmt.Errorf("inspect forbidden root %s: %w", path, err)
		}
		if pathsOverlap(root, path) {
			return preparedPathSet{}, fmt.Errorf("store root %s overlaps forbidden root %s", root, path)
		}
		for _, prior := range forbidden {
			if pathsOverlap(prior, path) {
				return preparedPathSet{}, fmt.Errorf("forbidden roots overlap: %s and %s", prior, path)
			}
		}
		forbidden = append(forbidden, path)
	}
	return preparedPathSet{root: root, forbidden: forbidden}, nil
}

func pathsOverlap(a, b string) bool {
	a, b = portableFold(a), portableFold(b)
	return withinPath(a, b) || withinPath(b, a)
}
func withinPath(path, root string) bool {
	if path == root {
		return true
	}
	root = strings.TrimRight(root, "/")
	if root == "" {
		root = "/"
	}
	return strings.HasPrefix(path, root+"/")
}
func portableFold(path string) string {
	value := strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
	if len(value) > 1 {
		value = strings.TrimRight(value, "/")
	}
	return value
}

// inspectPathComponents uses Lstat for each existing component so neither a
// symlink nor a Windows reparse point can redirect the store or a forbidden root.
func inspectPathComponents(path string, allowMissing bool) error {
	path = filepath.Clean(path)
	var components []string
	for current := path; ; {
		components = append(components, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		volume := filepath.VolumeName(current)
		if volume != "" && strings.EqualFold(parent, volume+string(filepath.Separator)) {
			components = append(components, parent)
			break
		}
		current = parent
	}
	for i := len(components) - 1; i >= 0; i-- {
		current := components[i]
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if !allowMissing || i != 0 {
				return fmt.Errorf("path component does not exist: %s", current)
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect path component %s: %w", current, err)
		}
		if err := rejectReparse(current, info); err != nil {
			return err
		}
		if i != 0 && !info.IsDir() {
			return fmt.Errorf("path ancestor is not a directory: %s", current)
		}
	}
	if allowMissing {
		// Store initialization may create only the final root. Its parent must exist.
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			parent := filepath.Dir(path)
			if _, err := os.Lstat(parent); err != nil {
				return fmt.Errorf("store parent must already exist: %s", parent)
			}
		}
	}
	return nil
}

func checkDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect directory %s: %w", path, err)
	}
	if err = rejectReparse(path, info); err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a real directory: %s", path)
	}
	return nil
}

func rejectReparse(path string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink refused: %s", path)
	}
	// Windows FileInfo.Sys exposes Win32FileAttributeData. Inspect by field name
	// to keep this file portable and avoid a platform-specific package dependency.
	sys := reflect.ValueOf(info.Sys())
	if sys.IsValid() && sys.Kind() == reflect.Pointer && !sys.IsNil() {
		sys = sys.Elem()
	}
	if sys.IsValid() && sys.Kind() == reflect.Struct {
		field := sys.FieldByName("FileAttributes")
		if field.IsValid() && field.CanUint() && field.Uint()&0x400 != 0 {
			return fmt.Errorf("reparse point refused: %s", path)
		}
	}
	return nil
}
