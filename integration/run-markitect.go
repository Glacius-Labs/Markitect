// run-markitect.go is copied into a consumer repository and invoked with
// `go run .markitect/bootstrap/run.go ...`. It uses only the Go standard library.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const lockName = ".markitect/tool/lock.yaml"
const maxLockSize = 64 << 10

type limits struct {
	maxEntries int
	maxFile    int64
	maxTotal   int64
	maxArchive int64
}

var defaultLimits = limits{
	maxEntries: 100_000,
	maxFile:    64 << 20,
	maxTotal:   512 << 20,
	maxArchive: 512 << 20,
}

type manifest struct {
	version string
	source  string
	sha256  string
}

type archiveFile struct {
	name string
	file *zip.File
	mode os.FileMode
}

type sourceArchive struct {
	data  []byte
	files []archiveFile
	dirs  map[string]bool
}

type builderFunc func(goPath string, args []string, dir string, env []string) error

type toolchainVersionFunc func(goPath, dir string, env []string) (string, error)

const buildPolicy = "portable-native-v1"

// goBuildInputs lists environment settings that can change tool selection,
// build inputs, target features, or cache behavior. Other environment names
// (including GOOGLE_APPLICATION_CREDENTIALS and GOAUTH) pass through.
var goBuildInputs = map[string]bool{
	"GCCGO": true, "GCCGOTOOLDIR": true,
	"GO111MODULE": true, "GOBIN": true, "GOCACHEPROG": true, "GODEBUG": true,
	"GOENV": true, "GOFLAGS": true, "GOOS": true, "GOARCH": true, "GOROOT": true,
	"GOTOOLCHAIN": true, "GOWORK": true, "GO386": true, "GOAMD64": true,
	"GOARM": true, "GOARM64": true, "GOMIPS": true, "GOMIPS64": true,
	"GOPPC64": true, "GORISCV64": true, "GOWASM": true, "GOEXPERIMENT": true,
	"GOFIPS140": true, "GO_EXTLINK_ENABLED": true,
}

func parseFlatYAML(data []byte, expected []string, document string) (map[string]string, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%s must be UTF-8", document)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if strings.ContainsRune(text, '\r') {
		return nil, fmt.Errorf("%s must not contain lone CR line endings", document)
	}
	text = strings.TrimSuffix(text, "\n")
	lines := strings.Split(text, "\n")
	if len(lines) != len(expected) {
		return nil, fmt.Errorf("%s must contain exactly %d fields", document, len(expected))
	}
	values := make(map[string]string, len(expected))
	order := make([]string, 0, len(expected))
	for _, line := range lines {
		key, scalar, ok := strings.Cut(line, ": ")
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			return nil, fmt.Errorf("invalid %s field syntax %q", document, line)
		}
		if _, duplicate := values[key]; duplicate {
			return nil, fmt.Errorf("duplicate %s field %q", document, key)
		}
		var value string
		if len(scalar) < 2 || scalar[0] != '"' || scalar[len(scalar)-1] != '"' || json.Unmarshal([]byte(scalar), &value) != nil {
			return nil, fmt.Errorf("%s field %q must be a JSON-compatible double-quoted string", document, key)
		}
		values[key] = value
		order = append(order, key)
	}
	for i, key := range expected {
		if order[i] != key {
			return nil, fmt.Errorf("%s fields must be exactly %s in canonical order", document, strings.Join(expected, ", "))
		}
		if values[key] == "" {
			return nil, fmt.Errorf("%s field %q must not be empty", document, key)
		}
	}
	return values, nil
}

func parseManifest(data []byte) (manifest, error) {
	values, err := parseFlatYAML(data, []string{"version", "source", "sha256"}, lockName)
	if err != nil {
		return manifest{}, err
	}
	m := manifest{version: values["version"], source: values["source"], sha256: values["sha256"]}
	if !validVersion(m.version) {
		return manifest{}, fmt.Errorf("unsupported Markitect version %q", m.version)
	}
	if err := safeRepoPath(m.source); err != nil {
		return manifest{}, fmt.Errorf("unsafe source archive path: %w", err)
	}
	if len(m.sha256) != 64 {
		return manifest{}, errors.New("lock sha256 must contain 64 lowercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(m.sha256); err != nil || strings.ToLower(m.sha256) != m.sha256 {
		return manifest{}, errors.New("lock sha256 must contain 64 lowercase hexadecimal characters")
	}
	return m, nil
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
		if part == "" || len(part) > 1 && part[0] == '0' {
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

func safeRepoPath(name string) error {
	if name == "" || !utf8.ValidString(name) || strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") || path.IsAbs(name) || path.Clean(name) != name {
		return errors.New("path must be normalized and repository-relative")
	}
	for _, component := range strings.Split(name, "/") {
		if component == "" || component == "." || component == ".." || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
			return errors.New("path contains an unsafe component")
		}
		for _, r := range component {
			if r < 32 || strings.ContainsRune(`<>"|?*`, r) {
				return errors.New("path contains an unsafe character")
			}
		}
		base := strings.ToUpper(strings.SplitN(component, ".", 2)[0])
		if strings.EqualFold(component, ".git") {
			return errors.New("path contains forbidden .git component")
		}
		if reservedWindowsName(base) {
			return errors.New("path contains a reserved Windows device name")
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

func isReparseOrSymlink(info os.FileInfo) bool {
	if info == nil || info.Mode()&os.ModeSymlink != 0 {
		return info != nil
	}
	// Windows FileInfo.Sys exposes FileAttributes. Reflection keeps this
	// standalone source portable without OS-specific companion files.
	sys := info.Sys()
	if sys == nil {
		return false
	}
	v := reflect.Indirect(reflect.ValueOf(sys))
	if v.IsValid() && v.Kind() == reflect.Struct {
		field := v.FieldByName("FileAttributes")
		if field.IsValid() && field.CanUint() {
			return field.Uint()&0x400 != 0 // FILE_ATTRIBUTE_REPARSE_POINT
		}
	}
	return false
}

func checkPathChain(target string, create bool) error {
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	volume := filepath.VolumeName(abs)
	current := volume + string(filepath.Separator)
	rest := strings.TrimPrefix(abs, current)
	if volume == "" {
		current = string(filepath.Separator)
		rest = strings.TrimPrefix(abs, current)
	}
	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if create {
				if err := os.Mkdir(current, 0755); err != nil && !errors.Is(err, os.ErrExist) {
					return fmt.Errorf("create directory %s: %w", current, err)
				}
				info, err = os.Lstat(current)
			}
			if !create {
				continue
			}
		}
		if err != nil {
			return fmt.Errorf("inspect path %s: %w", current, err)
		}
		if isReparseOrSymlink(info) {
			return fmt.Errorf("path traverses symlink or reparse point: %s", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("path component is not a directory: %s", current)
		}
	}
	return nil
}

func checkedRepoPath(root, relative string, mustExist bool) (string, error) {
	if err := safeRepoPath(relative); err != nil {
		return "", err
	}
	full := filepath.Join(root, filepath.FromSlash(relative))
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes repository root")
	}
	if err := checkPathChain(filepath.Dir(full), false); err != nil {
		return "", err
	}
	info, err := os.Lstat(full)
	if errors.Is(err, os.ErrNotExist) && !mustExist {
		return full, nil
	}
	if err != nil {
		return "", err
	}
	if isReparseOrSymlink(info) {
		return "", errors.New("path is a symlink or reparse point")
	}
	return full, nil
}

func discoverRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(current)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		current = filepath.Dir(current)
	}
	for {
		candidate := filepath.Join(current, lockName)
		info, err := os.Lstat(candidate)
		if err == nil {
			if isReparseOrSymlink(info) || !info.Mode().IsRegular() {
				return "", fmt.Errorf("unsafe lock file: %s", candidate)
			}
			return current, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("could not find %s in current directory ancestors", lockName)
}

func readManifest(root string) (manifest, string, error) {
	lockPath := filepath.Join(root, lockName)
	info, err := os.Lstat(lockPath)
	if err != nil {
		return manifest{}, "", fmt.Errorf("inspect %s: %w", lockName, err)
	}
	if isReparseOrSymlink(info) || !info.Mode().IsRegular() || info.Size() > maxLockSize {
		return manifest{}, "", fmt.Errorf("%s must be a regular file no larger than %d bytes", lockName, maxLockSize)
	}
	lock, err := os.Open(lockPath)
	if err != nil {
		return manifest{}, "", fmt.Errorf("read %s: %w", lockName, err)
	}
	defer lock.Close()
	opened, err := lock.Stat()
	if err != nil || !opened.Mode().IsRegular() || opened.Size() > maxLockSize {
		return manifest{}, "", fmt.Errorf("%s must be a regular file no larger than %d bytes", lockName, maxLockSize)
	}
	data, err := io.ReadAll(io.LimitReader(lock, maxLockSize+1))
	if err != nil {
		return manifest{}, "", fmt.Errorf("read %s: %w", lockName, err)
	}
	if len(data) > maxLockSize {
		return manifest{}, "", fmt.Errorf("%s exceeds the %d byte limit", lockName, maxLockSize)
	}
	m, err := parseManifest(data)
	if err != nil {
		return manifest{}, "", err
	}
	archivePath, err := checkedRepoPath(root, m.source, true)
	if err != nil {
		return manifest{}, "", fmt.Errorf("source archive: %w", err)
	}
	info, err = os.Lstat(archivePath)
	if err != nil || isReparseOrSymlink(info) || !info.Mode().IsRegular() {
		return manifest{}, "", fmt.Errorf("source archive must be a regular file: %s", m.source)
	}
	return m, archivePath, nil
}

func readBoundedRegularFile(name string, maxBytes int64, label string) ([]byte, error) {
	lstat, err := os.Lstat(name)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", label, err)
	}
	if isReparseOrSymlink(lstat) || !lstat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file", label)
	}
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", label, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat opened %s: %w", label, err)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(lstat, opened) {
		return nil, fmt.Errorf("%s changed while opening", label)
	}
	if opened.Size() < 0 || opened.Size() > maxBytes {
		return nil, fmt.Errorf("%s exceeds %d byte limit", label, maxBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%s exceeds %d byte limit", label, maxBytes)
	}
	return data, nil
}

func readArchive(archivePath, expectedDigest string, lim limits) (*sourceArchive, error) {
	data, err := readBoundedRegularFile(archivePath, lim.maxArchive, "source archive")
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != expectedDigest {
		return nil, errors.New("source archive SHA-256 does not match lock")
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid source archive: %w", err)
	}
	a := &sourceArchive{data: data, dirs: make(map[string]bool)}
	if len(r.File) > lim.maxEntries {
		return nil, errors.New("source archive exceeds entry-count limit")
	}
	seen := make(map[string]bool, len(r.File))
	portable := make(map[string]pathRecord, len(r.File)*2)
	var total int64
	for _, f := range r.File {
		name := f.Name
		isDir := strings.HasSuffix(name, "/")
		if isDir {
			name = strings.TrimSuffix(name, "/")
		}
		if err := safeRepoPath(name); err != nil {
			return nil, fmt.Errorf("unsafe archive entry %q: %w", name, err)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate archive entry %q", name)
		}
		seen[name] = true
		mode := f.Mode()
		if f.ExternalAttrs&0x400 != 0 || isReparseOrSymlinkMode(mode) {
			return nil, fmt.Errorf("archive entry is a symlink or reparse point: %s", name)
		}
		typeBits := mode.Type()
		if typeBits != 0 && typeBits != os.ModeDir {
			return nil, fmt.Errorf("unsupported special archive entry: %s", name)
		}
		if isDir != (typeBits == os.ModeDir) && typeBits != 0 {
			return nil, fmt.Errorf("archive entry type mismatch: %s", name)
		}
		if isDir {
			if typeBits == 0 {
				mode |= os.ModeDir
			}
			a.dirs[name] = true
		} else {
			if f.UncompressedSize64 > uint64(lim.maxFile) {
				return nil, fmt.Errorf("archive file exceeds per-file limit: %s", name)
			}
			total += int64(f.UncompressedSize64)
			if total > lim.maxTotal {
				return nil, errors.New("source archive exceeds expanded-size limit")
			}
			if mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
				return nil, fmt.Errorf("unsafe archive mode bits: %s", name)
			}
			if mode.Perm() == 0 {
				mode = 0644
			}
			if mode.Perm() != 0644 && mode.Perm() != 0755 {
				return nil, fmt.Errorf("unsupported archive permissions for %s: %04o", name, mode.Perm())
			}
			a.files = append(a.files, archiveFile{name: name, file: f, mode: mode.Perm()})
		}
		parts := strings.Split(name, "/")
		for i := range parts {
			prefix := strings.Join(parts[:i+1], "/")
			isPrefixDir := i < len(parts)-1 || isDir
			key := strings.ToLower(prefix)
			if old, ok := portable[key]; ok && (old.name != prefix || old.dir != isPrefixDir) {
				return nil, fmt.Errorf("case-fold path collision: %s and %s", old.name, prefix)
			}
			portable[key] = pathRecord{name: prefix, dir: isPrefixDir}
			if i < len(parts)-1 {
				a.dirs[prefix] = true
			}
		}
	}
	if err := validateModuleLayout(a); err != nil {
		return nil, err
	}
	sort.Slice(a.files, func(i, j int) bool { return a.files[i].name < a.files[j].name })
	return a, nil
}

type pathRecord struct {
	name string
	dir  bool
}

func isReparseOrSymlinkMode(mode os.FileMode) bool {
	return mode&os.ModeSymlink != 0
}

func validateModuleLayout(a *sourceArchive) error {
	files := make(map[string]bool, len(a.files))
	for _, f := range a.files {
		files[f.name] = true
	}
	requiredDirs := []string{"cmd", "cmd/markitect", "internal"}
	for _, d := range requiredDirs {
		if !a.dirs[d] {
			return fmt.Errorf("source archive is not a canonical Markitect Go module: missing directory %s", d)
		}
	}
	if !files["go.mod"] || !files["cmd/markitect/main.go"] {
		return errors.New("source archive is not a canonical Markitect Go module: missing go.mod or command entry point")
	}
	return nil
}

func ensureDirectory(path string) error {
	return checkPathChain(path, true)
}

func randomID() (string, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func cacheBase(root string) (string, error) {
	base := filepath.Join(root, ".artifacts", "markitect")
	if err := ensureDirectory(base); err != nil {
		return "", fmt.Errorf("prepare Markitect cache: %w", err)
	}
	return base, nil
}

func verifyExtracted(sourceDir string, a *sourceArchive) (bool, error) {
	info, err := os.Lstat(sourceDir)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if isReparseOrSymlink(info) || !info.IsDir() {
		return false, fmt.Errorf("unsafe extracted source directory %s", sourceDir)
	}
	actualFiles := make(map[string]bool)
	actualDirs := make(map[string]bool)
	err = filepath.WalkDir(sourceDir, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == sourceDir {
			return nil
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if isReparseOrSymlink(info) {
			return fmt.Errorf("symlink or reparse point in extracted source: %s", current)
		}
		rel, err := filepath.Rel(sourceDir, current)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			actualDirs[rel] = true
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file in extracted source: %s", current)
		}
		actualFiles[rel] = true
		return nil
	})
	if err != nil {
		return false, err
	}
	wantDirs := make(map[string]bool, len(a.dirs))
	for dir := range a.dirs {
		wantDirs[dir] = true
	}
	if !sameSet(actualFiles, archiveFileSet(a)) || !sameSet(actualDirs, wantDirs) {
		return false, nil
	}
	for _, file := range a.files {
		full := filepath.Join(sourceDir, filepath.FromSlash(file.name))
		if err := compareFileToZip(full, file); err != nil {
			if errors.Is(err, errContentMismatch) {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}

var errContentMismatch = errors.New("extracted content mismatch")

func compareFileToZip(full string, entry archiveFile) error {
	local, err := os.Open(full)
	if err != nil {
		return err
	}
	defer local.Close()
	info, err := local.Stat()
	if err != nil {
		return err
	}
	if info.Size() < 0 || uint64(info.Size()) != entry.file.UncompressedSize64 {
		return errContentMismatch
	}
	zipped, err := entry.file.Open()
	if err != nil {
		return err
	}
	defer zipped.Close()
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 != entry.mode&0111 {
		return errContentMismatch
	}
	localHash := sha256.New()
	zipHash := sha256.New()
	if _, err := io.Copy(localHash, local); err != nil {
		return err
	}
	if _, err := io.Copy(zipHash, zipped); err != nil {
		return err
	}
	if !bytes.Equal(localHash.Sum(nil), zipHash.Sum(nil)) {
		return errContentMismatch
	}
	return nil
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if !b[key] {
			return false
		}
	}
	return true
}

func archiveFileSet(a *sourceArchive) map[string]bool {
	files := make(map[string]bool, len(a.files))
	for _, file := range a.files {
		files[file.name] = true
	}
	return files
}

func findOrExtractSource(digestDir string, archive *sourceArchive) (string, error) {
	entries, err := os.ReadDir(digestDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "source-") {
			continue
		}
		candidate := filepath.Join(digestDir, entry.Name())
		info, err := os.Lstat(candidate)
		if err != nil {
			return "", err
		}
		if isReparseOrSymlink(info) {
			return "", fmt.Errorf("unsafe source cache entry: %s", candidate)
		}
		valid, err := verifyExtracted(candidate, archive)
		if err != nil {
			return "", err
		}
		if valid {
			return candidate, nil
		}
	}
	id, err := randomID()
	if err != nil {
		return "", err
	}
	sourceDir := filepath.Join(digestDir, "source-"+id)
	if err := os.Mkdir(sourceDir, 0755); err != nil {
		return "", err
	}
	if err := extractSource(sourceDir, archive); err != nil {
		return "", err
	}
	valid, err := verifyExtracted(sourceDir, archive)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", errors.New("extracted source differs from verified archive")
	}
	return sourceDir, nil
}

func extractSource(sourceDir string, archive *sourceArchive) error {
	dirs := make([]string, 0, len(archive.dirs))
	for dir := range archive.dirs {
		dirs = append(dirs, dir)
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.Count(dirs[i], "/") < strings.Count(dirs[j], "/")
	})
	for _, dir := range dirs {
		if err := os.Mkdir(filepath.Join(sourceDir, filepath.FromSlash(dir)), 0755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	for _, entry := range archive.files {
		target := filepath.Join(sourceDir, filepath.FromSlash(entry.name))
		if err := ensureDirectory(filepath.Dir(target)); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, entry.mode)
		if err != nil {
			return err
		}
		in, err := entry.file.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		inErr := in.Close()
		outErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if inErr != nil {
			return inErr
		}
		if outErr != nil {
			return outErr
		}
		if err := os.Chmod(target, entry.mode); err != nil {
			return err
		}
	}
	return nil
}

func parseStamp(data []byte) (map[string]string, error) {
	return parseFlatYAML(data, []string{"version", "source_sha256", "toolchain", "build_policy", "executable_sha256"}, "build stamp")
}

func stampYAML(version, sourceDigest, toolchain, policy, executableDigest string) []byte {
	return []byte("version: " + strconv.Quote(version) + "\n" +
		"source_sha256: " + strconv.Quote(sourceDigest) + "\n" +
		"toolchain: " + strconv.Quote(toolchain) + "\n" +
		"build_policy: " + strconv.Quote(policy) + "\n" +
		"executable_sha256: " + strconv.Quote(executableDigest) + "\n")
}

func findCachedExecutable(platformDir string, m manifest, toolchain string) (string, error) {
	entries, err := os.ReadDir(platformDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "build-stamp-") || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		stampPath := filepath.Join(platformDir, name)
		info, err := os.Lstat(stampPath)
		if err != nil {
			return "", err
		}
		if isReparseOrSymlink(info) || !info.Mode().IsRegular() {
			return "", fmt.Errorf("unsafe build stamp: %s", stampPath)
		}
		data, err := readBoundedRegularFile(stampPath, maxLockSize, "build stamp")
		if err != nil {
			return "", err
		}
		stamp, err := parseStamp(data)
		if err != nil || stamp["version"] != m.version || stamp["source_sha256"] != m.sha256 || stamp["toolchain"] != toolchain || stamp["build_policy"] != buildPolicy || !isDigest(stamp["executable_sha256"]) {
			continue
		}
		nonce := strings.TrimSuffix(strings.TrimPrefix(name, "build-stamp-"), ".yaml")
		if len(nonce) != 16 {
			continue
		}
		if _, err := hex.DecodeString(nonce); err != nil {
			continue
		}
		exeName := "markitect-" + nonce
		if runtime.GOOS == "windows" {
			exeName += ".exe"
		}
		executable := filepath.Join(platformDir, exeName)
		exeInfo, err := os.Lstat(executable)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if isReparseOrSymlink(exeInfo) || !exeInfo.Mode().IsRegular() {
			return "", fmt.Errorf("unsafe cached Markitect executable: %s", executable)
		}
		digest, err := fileDigest(executable)
		if err != nil {
			return "", err
		}
		if digest == stamp["executable_sha256"] {
			return executable, nil
		}
	}
	return "", nil
}

func isDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func fileDigest(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func environmentForBuild(sourceDir, sharedCache string, original []string, selectedToolchain ...string) ([]string, error) {
	env := make(map[string]string, len(original)+8)
	for _, item := range original {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			upper := strings.ToUpper(key)
			// Drop documented Go build inputs and all cgo compiler settings. The
			// list is explicit so unrelated GO-prefixed integrations survive.
			if goBuildInputs[upper] || strings.HasPrefix(upper, "CGO_") {
				continue
			}
			if runtime.GOOS == "windows" {
				for existing := range env {
					if strings.EqualFold(existing, key) {
						delete(env, existing)
					}
				}
				key = upper
			}
			env[key] = value
		}
	}
	env["GOFLAGS"] = ""
	env["GOWORK"] = "off"
	env["GOENV"] = "off"
	env["GOOS"] = runtime.GOOS
	env["GOARCH"] = runtime.GOARCH
	env["CGO_ENABLED"] = "0"
	env["GOTOOLCHAIN"] = "auto"
	if len(selectedToolchain) != 0 && selectedToolchain[0] != "" {
		env["GOTOOLCHAIN"] = selectedToolchain[0]
	}
	switch runtime.GOARCH {
	case "amd64":
		env["GOAMD64"] = "v1"
	case "arm64":
		env["GOARM64"] = "v8.0"
	case "386":
		env["GO386"] = "sse2"
	case "arm":
		env["GOARM"] = "7"
	}
	for key, local := range map[string]string{"GOCACHE": "go-build", "GOTMPDIR": "go-tmp"} {
		if override := env[key]; override != "" {
			check := override
			if !filepath.IsAbs(check) {
				check = filepath.Join(sourceDir, check)
			}
			if err := checkPathChain(check, false); err != nil {
				return nil, fmt.Errorf("unsafe %s override: %w", key, err)
			}
			continue
		}
		localPath := filepath.Join(sharedCache, local)
		if err := checkPathChain(localPath, true); err != nil {
			return nil, fmt.Errorf("prepare local %s: %w", key, err)
		}
		env[key] = localPath
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+env[key])
	}
	return result, nil
}

func queryToolchainVersion(goPath, dir string, env []string) (string, error) {
	cmd := exec.Command(goPath, "env", "GOVERSION")
	cmd.Dir = dir
	cmd.Env = env
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// Go diagnostics can echo credential-bearing GOPROXY URLs. Keep the error
	// surface bounded and avoid reflecting environment-derived secrets.
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("select Go toolchain for pinned source: %w", err)
	}
	version := strings.TrimSpace(stdout.String())
	if !validToolchainVersion(version) {
		return "", fmt.Errorf("Go reported an invalid toolchain version")
	}
	return version, nil
}

func validToolchainVersion(version string) bool {
	if !strings.HasPrefix(version, "go") || len(version) < 5 {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(version, "go"), ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
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

func defaultBuilder(goPath string, args []string, dir string, env []string) error {
	cmd := exec.Command(goPath, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func buildOrReuse(root string, m manifest, archive *sourceArchive, goPath string, originalEnv []string, builder builderFunc, probe toolchainVersionFunc) (string, error) {
	base, err := cacheBase(root)
	if err != nil {
		return "", err
	}
	digestDir := filepath.Join(base, m.sha256)
	if err := ensureDirectory(digestDir); err != nil {
		return "", err
	}
	platformDir := filepath.Join(digestDir, runtime.GOOS+"-"+runtime.GOARCH)
	if err := ensureDirectory(platformDir); err != nil {
		return "", err
	}
	sourceDir, err := findOrExtractSource(digestDir, archive)
	if err != nil {
		return "", err
	}
	probeEnv, err := environmentForBuild(sourceDir, base, originalEnv, "auto")
	if err != nil {
		return "", err
	}
	if probe == nil {
		probe = queryToolchainVersion
	}
	toolchain, err := probe(goPath, sourceDir, probeEnv)
	if err != nil {
		return "", err
	}
	if !validToolchainVersion(toolchain) {
		return "", errors.New("toolchain probe returned an invalid version")
	}
	if cached, err := findCachedExecutable(platformDir, m, toolchain); err != nil || cached != "" {
		return cached, err
	}
	buildEnv, err := environmentForBuild(sourceDir, base, originalEnv, toolchain)
	if err != nil {
		return "", err
	}
	if builder == nil {
		builder = defaultBuilder
	}
	nonce, err := randomID()
	if err != nil {
		return "", err
	}
	tempPattern := "markitect-tmp-*"
	if runtime.GOOS == "windows" {
		tempPattern += ".exe"
	}
	temp, err := os.CreateTemp(platformDir, tempPattern)
	if err != nil {
		return "", err
	}
	tempName := temp.Name()
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(tempName); err != nil {
		return "", err
	}
	args := []string{"build", "-buildvcs=false", "-trimpath", "-ldflags", "-X main.version=" + m.version + " -X github.com/Glacius-Labs/Markitect/internal/host/cli.version=" + m.version, "-o", tempName, "./cmd/markitect"}
	if err := builder(goPath, args, sourceDir, buildEnv); err != nil {
		return "", fmt.Errorf("build pinned Markitect source: %w", err)
	}
	validSource, err := verifyExtracted(sourceDir, archive)
	if err != nil {
		return "", fmt.Errorf("verify pinned Markitect source after build: %w", err)
	}
	if !validSource {
		return "", errors.New("pinned Markitect source changed during build")
	}
	info, err := os.Lstat(tempName)
	if err != nil || isReparseOrSymlink(info) || !info.Mode().IsRegular() {
		return "", errors.New("Go build did not produce a regular executable")
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempName, info.Mode().Perm()|0111); err != nil {
			return "", err
		}
	}
	executable := filepath.Join(platformDir, "markitect-"+nonce)
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	if err := os.Rename(tempName, executable); err != nil {
		return "", err
	}
	executableDigest, err := fileDigest(executable)
	if err != nil {
		return "", err
	}
	stampPath := filepath.Join(platformDir, "build-stamp-"+nonce+".yaml")
	stampTemp, err := os.CreateTemp(platformDir, "build-stamp-tmp-*")
	if err != nil {
		return "", err
	}
	stampTempName := stampTemp.Name()
	if _, err := stampTemp.Write(stampYAML(m.version, m.sha256, toolchain, buildPolicy, executableDigest)); err != nil {
		stampTemp.Close()
		os.Remove(stampTempName)
		return "", err
	}
	if err := stampTemp.Sync(); err != nil {
		stampTemp.Close()
		os.Remove(stampTempName)
		return "", err
	}
	if err := stampTemp.Close(); err != nil {
		os.Remove(stampTempName)
		return "", err
	}
	if err := os.Rename(stampTempName, stampPath); err != nil {
		return "", err
	}
	return executable, nil
}

func injectRepo(args []string, root string) []string {
	if len(args) == 0 {
		return args
	}
	switch args[0] {
	case "version", "authoring", "licenses", "help", "--help", "-h":
		return args
	}
	for _, arg := range args[1:] {
		if arg == "--" {
			break
		}
		if arg == "--repo" || strings.HasPrefix(arg, "--repo=") {
			return args
		}
	}
	result := make([]string, 0, len(args)+2)
	result = append(result, args[0], "--repo", root)
	result = append(result, args[1:]...)
	return result
}

func run(args []string, cwd string, builder builderFunc) int {
	root, err := discoverRoot(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "markitect bootstrap:", err)
		return 2
	}
	m, archivePath, err := readManifest(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "markitect bootstrap:", err)
		return 2
	}
	archive, err := readArchive(archivePath, m.sha256, defaultLimits)
	if err != nil {
		fmt.Fprintln(os.Stderr, "markitect bootstrap:", err)
		return 2
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, "markitect bootstrap: Go is required:", err)
		return 2
	}
	executable, err := buildOrReuse(root, m, archive, goPath, os.Environ(), builder, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "markitect bootstrap:", err)
		return 2
	}
	cmd := exec.Command(executable, injectRepo(args, root)...)
	cmd.Dir = root
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			if exit.ExitCode() >= 0 {
				return exit.ExitCode()
			}
		}
		fmt.Fprintln(os.Stderr, "markitect bootstrap: run executable:", err)
		return 1
	}
	return 0
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "markitect bootstrap:", err)
		os.Exit(2)
	}
	os.Exit(run(os.Args[1:], cwd, nil))
}
