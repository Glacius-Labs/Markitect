package release

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

func collect(moduleDir string) ([]sourceFile, error) {
	files := make([]sourceFile, 0, 64)
	for _, name := range []string{"go.mod", "go.sum"} {
		if err := appendRegularFile(moduleDir, name, &files, true); err != nil {
			return nil, err
		}
	}
	for _, tree := range []string{"cmd", "internal"} {
		root := filepath.Join(moduleDir, filepath.FromSlash(tree))
		if err := requireRealDirectory(root); err != nil {
			return nil, fmt.Errorf("required source tree %s: %w", tree, err)
		}
		if err := walkSourceTree(moduleDir, root, &files); err != nil {
			return nil, err
		}
	}
	for _, name := range []string{"README.md", "LICENSE"} {
		if err := appendRegularFile(moduleDir, name, &files, name == "LICENSE"); err != nil {
			return nil, err
		}
	}
	schemaDir := filepath.Join(moduleDir, "schema")
	if info, err := os.Lstat(schemaDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("optional schema path must be a real directory")
		}
		if err := walkSchemaTree(moduleDir, schemaDir, &files); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect optional schema directory: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	if len(files) < 4 {
		return nil, fmt.Errorf("Markitect source package is incomplete")
	}
	return files, nil
}

func walkSourceTree(moduleDir, start string, files *[]sourceFile) error {
	return filepath.WalkDir(start, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk Markitect source %q: %w", current, walkErr)
		}
		rel, err := filepath.Rel(moduleDir, current)
		if err != nil {
			return fmt.Errorf("resolve source path %q: %w", current, err)
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if current != start && excludedDirs[strings.ToLower(entry.Name())] {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in Markitect source tree: %s", rel)
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		embeddedResource := embeddedAuthoringSource(rel)
		if ext != ".go" && !embeddedResource && rel != embeddedNoticesPath {
			return nil
		}
		return appendRegularFile(moduleDir, rel, files, true)
	})
}

func walkSchemaTree(moduleDir, start string, files *[]sourceFile) error {
	return filepath.WalkDir(start, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk Markitect schema %q: %w", current, walkErr)
		}
		rel, err := filepath.Rel(moduleDir, current)
		if err != nil {
			return fmt.Errorf("resolve schema path %q: %w", current, err)
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if current != start && excludedDirs[strings.ToLower(entry.Name())] {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in Markitect schema tree: %s", rel)
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" && ext != ".md" {
			return nil
		}
		return appendRegularFile(moduleDir, rel, files, true)
	})
}

func appendRegularFile(moduleDir, relative string, files *[]sourceFile, required bool) error {
	if err := safeArchivePath(relative); err != nil {
		return fmt.Errorf("unsafe Markitect source path %q: %w", relative, err)
	}
	full := filepath.Join(moduleDir, filepath.FromSlash(relative))
	info, err := os.Lstat(full)
	if os.IsNotExist(err) && !required {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat Markitect source %s: %w", relative, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("Markitect source must be a regular file: %s", relative)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return fmt.Errorf("read Markitect source %s: %w", relative, err)
	}
	data, err = normalizeTextSource(relative, data)
	if err != nil {
		return fmt.Errorf("invalid Markitect text source %s: %w", relative, err)
	}
	*files = append(*files, sourceFile{name: relative, data: data})
	return nil
}

func normalizeTextSource(name string, data []byte) ([]byte, error) {
	base := strings.ToLower(filepath.Base(name))
	ext := strings.ToLower(filepath.Ext(base))
	isText := base == "go.mod" || base == "go.sum" || base == "license" || ext == ".go" || ext == ".md" || ext == ".yaml" || ext == ".yml"
	if !isText {
		return data, nil
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("text source is not valid UTF-8")
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return nil, fmt.Errorf("text source contains a NUL byte")
	}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	return data, nil
}

func requireRealDirectory(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("must be a real directory")
	}
	return nil
}
