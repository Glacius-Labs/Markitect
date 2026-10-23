package release

import (
	"archive/zip"
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"
)

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
