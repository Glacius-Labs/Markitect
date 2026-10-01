// Package release builds a portable, reproducible Markitect source archive.
package release

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const sourcePath = "tools/markitect/source.zip"

var excludedDirs = map[string]bool{
	".git": true, ".cache": true, ".gocache": true, ".artifacts": true,
	"bin": true, "obj": true, "vendor": true, "node_modules": true,
}

// Package packages the Markitect module's Go source, module files, and optional
// root README/schema files. It returns the source archive and its flat lock
// manifest without writing to the filesystem.
func Package(root, version string) (archive, lock []byte, err error) {
	if !validVersion(version) {
		return nil, nil, fmt.Errorf("invalid semantic version %q", version)
	}
	if root == "" {
		return nil, nil, fmt.Errorf("repository root is empty")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve repository root: %w", err)
	}
	if err := requireRealDirectory(absRoot); err != nil {
		return nil, nil, fmt.Errorf("repository root: %w", err)
	}
	moduleDir := absRoot
	if _, err := os.Lstat(filepath.Join(moduleDir, "go.mod")); os.IsNotExist(err) {
		if err := requireRealDirectory(filepath.Join(absRoot, "tools")); err != nil {
			return nil, nil, fmt.Errorf("tools directory: %w", err)
		}
		moduleDir = filepath.Join(absRoot, "tools", "markitect")
	}
	if err := requireRealDirectory(moduleDir); err != nil {
		return nil, nil, fmt.Errorf("Markitect module: %w", err)
	}
	files, err := collect(moduleDir)
	if err != nil {
		return nil, nil, err
	}
	archive, err = makeArchive(files)
	if err != nil {
		return nil, nil, err
	}
	digest := sha256.Sum256(archive)
	lock = []byte("version: " + strconv.Quote(version) + "\n" +
		"source: " + strconv.Quote(sourcePath) + "\n" +
		"sha256: " + strconv.Quote(hex.EncodeToString(digest[:])) + "\n")
	return archive, lock, nil
}

type sourceFile struct {
	name string
	data []byte
}

const embeddedNoticesPath = "internal/licenses/notices.md"

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
	if err := appendRegularFile(moduleDir, "README.md", &files, false); err != nil {
		return nil, err
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
		embeddedResource := strings.HasPrefix(rel, "internal/authoring/resources/") && ext == ".yaml"
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
	isText := base == "go.mod" || base == "go.sum" || ext == ".go" || ext == ".md" || ext == ".yaml" || ext == ".yml"
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

func makeArchive(files []sourceFile) ([]byte, error) {
	if err := validateArchivePaths(files); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	w := zip.NewWriter(&buffer)
	fixedTime := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, file := range files {
		if err := safeArchivePath(file.name); err != nil {
			return nil, fmt.Errorf("unsafe archive entry %q: %w", file.name, err)
		}
		header := &zip.FileHeader{Name: file.name, Method: zip.Store}
		header.SetModTime(fixedTime)
		header.SetMode(0644)
		entry, err := w.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("create archive entry %q: %w", file.name, err)
		}
		if _, err := entry.Write(file.data); err != nil {
			return nil, fmt.Errorf("write archive entry %q: %w", file.name, err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("finish Markitect source archive: %w", err)
	}
	return buffer.Bytes(), nil
}

type archivePathRecord struct {
	name string
	dir  bool
}

func validateArchivePaths(files []sourceFile) error {
	seenFiles := make(map[string]bool, len(files))
	portable := make(map[string]archivePathRecord, len(files)*2)
	for _, file := range files {
		if err := safeArchivePath(file.name); err != nil {
			return fmt.Errorf("unsafe archive entry %q: %w", file.name, err)
		}
		if seenFiles[file.name] {
			return fmt.Errorf("duplicate archive entry %q", file.name)
		}
		seenFiles[file.name] = true
		parts := strings.Split(file.name, "/")
		for i := range parts {
			prefix := strings.Join(parts[:i+1], "/")
			isDir := i < len(parts)-1
			key := strings.ToLower(prefix)
			if previous, ok := portable[key]; ok && (previous.name != prefix || previous.dir != isDir) {
				return fmt.Errorf("case-insensitive archive path collision: %q and %q", previous.name, prefix)
			}
			portable[key] = archivePathRecord{name: prefix, dir: isDir}
		}
	}
	return nil
}

func safeArchivePath(name string) error {
	if name == "" || strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") || path.IsAbs(name) || path.Clean(name) != name || name == "." || strings.HasPrefix(name, "../") {
		return fmt.Errorf("path must be normalized and repository-relative")
	}
	for _, component := range strings.Split(name, "/") {
		if component == "" || component == "." || component == ".." || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
			return fmt.Errorf("path contains an unsafe component")
		}
		for _, r := range component {
			if r < 32 || strings.ContainsRune(`<>"|?*`, r) {
				return fmt.Errorf("path contains an unsafe character")
			}
		}
		base := strings.ToUpper(strings.TrimSuffix(component, filepath.Ext(component)))
		if reservedWindowsName(base) {
			return fmt.Errorf("path contains a reserved Windows device name")
		}
	}
	return nil
}

func reservedWindowsName(base string) bool {
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9'
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

func validVersion(version string) bool {
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}
	if version == "" || strings.Count(version, "+") > 1 {
		return false
	}
	withoutBuild := version
	if before, build, ok := strings.Cut(version, "+"); ok {
		if !validIdentifiers(build, false) {
			return false
		}
		withoutBuild = before
	}
	core := withoutBuild
	if before, pre, ok := strings.Cut(withoutBuild, "-"); ok {
		if !validIdentifiers(pre, true) {
			return false
		}
		core = before
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func validIdentifiers(value string, rejectNumericLeadingZero bool) bool {
	if value == "" {
		return false
	}
	for _, part := range strings.Split(value, ".") {
		if part == "" {
			return false
		}
		numeric := true
		for _, r := range part {
			if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '-') {
				return false
			}
			if r < '0' || r > '9' {
				numeric = false
			}
		}
		if rejectNumericLeadingZero && numeric && len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}
