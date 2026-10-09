// Package release builds a portable, reproducible Markitect source archive.
package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const sourcePath = ".markitect/tool/source.zip"

var excludedDirs = map[string]bool{
	".git": true, ".cache": true, ".gocache": true, ".artifacts": true,
	"bin": true, "obj": true, "vendor": true, "node_modules": true,
}

// Package packages the Markitect module's Go source, module files, root license,
// and optional README/schema files. It returns the source archive and its flat lock
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

const embeddedNoticesPath = "src/internal/tooling/licenses/notices.md"
const legacyEmbeddedNoticesPath = "internal/tooling/licenses/notices.md"

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
