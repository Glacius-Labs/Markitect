// Package source loads immutable source snapshots from either the current
// working tree or a committed Git tree.
package source

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Limits bounds memory and filesystem use while loading and materializing a
// source snapshot. Load uses DefaultLimits; callers with larger trusted inputs
// can use LoadWithLimits.
type Limits struct {
	MaxFiles      int
	MaxFileBytes  int64
	MaxTotalBytes int64
}

// DefaultMaxFiles, DefaultMaxFileBytes and DefaultMaxTotalBytes bound
// accidental or hostile input while allowing typical documentation trees.
const (
	DefaultMaxFiles      = 100_000
	DefaultMaxFileBytes  = 64 << 20
	DefaultMaxTotalBytes = 512 << 20
)

// DefaultLimits returns the package's default resource bounds.
func DefaultLimits() Limits {
	return Limits{MaxFiles: DefaultMaxFiles, MaxFileBytes: DefaultMaxFileBytes, MaxTotalBytes: DefaultMaxTotalBytes}
}

// Snapshot contains file bytes keyed by slash-separated repository paths.
// Modes use Git's regular-file modes (100644 and 100755).
type Snapshot struct {
	Revision    string
	Provisional bool
	Files       map[string][]byte
	Modes       map[string]string
}

type treeFile struct {
	path string
	mode string
	oid  string
	size int64
}

// Load reads the working tree when revision is empty, or the immutable Git
// commit named by revision otherwise.
func Load(root, revision string) (*Snapshot, error) {
	return LoadWithLimits(root, revision, DefaultLimits())
}

// LoadWithLimits is Load with caller-selected resource limits.
func LoadWithLimits(root, revision string, limits Limits) (*Snapshot, error) {
	if err := validateLimits(limits); err != nil {
		return nil, err
	}
	if root == "" {
		return nil, errors.New("source root is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve source root: %w", err)
	}
	if err := rejectSymlinkAncestors(abs); err != nil {
		return nil, fmt.Errorf("source root: %w", err)
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat source root: %w", err)
	}
	if isSymlink(info) || !info.IsDir() {
		return nil, fmt.Errorf("source root must be a real directory: %s", abs)
	}

	s := &Snapshot{Revision: revision, Provisional: revision == "", Files: map[string][]byte{}, Modes: map[string]string{}}
	if revision == "" {
		if err := loadWorkingTree(abs, s, limits); err != nil {
			return nil, err
		}
		return s, nil
	}
	commit, err := resolveCommit(abs, revision)
	if err != nil {
		return nil, err
	}
	s.Revision = commit
	if err := loadCommit(abs, commit, s, limits); err != nil {
		return nil, err
	}
	return s, nil
}

func validateLimits(l Limits) error {
	if l.MaxFiles <= 0 || l.MaxFileBytes <= 0 || l.MaxTotalBytes <= 0 || l.MaxFileBytes > l.MaxTotalBytes {
		return errors.New("invalid source limits")
	}
	return nil
}

func loadWorkingTree(root string, s *Snapshot, limits Limits) error {
	var total int64
	var walk func(string, string) error
	walk = func(dir, relDir string) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("read source directory %q: %w", relDir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			rel := name
			if relDir != "" {
				rel = relDir + "/" + name
			}
			if excludedFilePath(rel) {
				continue
			}
			info, err := os.Lstat(filepath.Join(dir, name))
			if err != nil {
				return fmt.Errorf("stat source path %q: %w", rel, err)
			}
			if info.IsDir() {
				if excludedDirectoryPath(rel) {
					continue
				}
				if err := validateRepoPath(rel); err != nil {
					return err
				}
				if isSymlink(info) {
					return fmt.Errorf("refusing to traverse symlink or reparse point %q", rel)
				}
				if err := walk(filepath.Join(dir, name), rel); err != nil {
					return err
				}
				continue
			}
			if isSymlink(info) {
				return fmt.Errorf("refusing to read symlink or reparse point %q", rel)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("unsupported non-regular source file %q", rel)
			}
			if err := validateRepoPath(rel); err != nil {
				return err
			}
			if info.Size() > limits.MaxFileBytes {
				return fmt.Errorf("source file %q exceeds per-file limit", rel)
			}
			if err := accountFile(s, limits); err != nil {
				return err
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				return fmt.Errorf("read source file %q: %w", rel, err)
			}
			if int64(len(data)) > limits.MaxFileBytes || total > limits.MaxTotalBytes-int64(len(data)) {
				return fmt.Errorf("source snapshot exceeds byte limit at %q", rel)
			}
			total += int64(len(data))
			s.Files[rel] = data
			mode := "100644"
			if info.Mode().Perm()&0111 != 0 {
				mode = "100755"
			}
			s.Modes[rel] = mode
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return err
	}
	return validatePortablePaths(snapshotPaths(s))
}

func excludedDirectoryPath(p string) bool {
	parts := strings.Split(p, "/")
	for _, component := range parts {
		if component == ".git" {
			return true
		}
	}
	if len(parts) > 0 {
		switch parts[0] {
		case "docs", ".agents", ".claude", ".codex":
			return false
		case ".artifacts", ".cache", ".worktrees", ".venv", "__pycache__", "vendor":
			return true
		}
	}
	for _, component := range parts {
		switch component {
		case "bin", "obj", "node_modules", "vendor":
			return true
		}
	}
	return false
}

func excludedFilePath(p string) bool {
	parts := strings.Split(p, "/")
	for _, component := range parts {
		if component == ".git" {
			return true // linked worktrees store a .git file instead of a directory
		}
	}
	if len(parts) < 2 {
		return false
	}
	return excludedDirectoryPath(strings.Join(parts[:len(parts)-1], "/"))
}

func accountFile(s *Snapshot, limits Limits) error {
	if len(s.Files) >= limits.MaxFiles {
		return errors.New("source snapshot exceeds file-count limit")
	}
	return nil
}

func resolveCommit(root, revision string) (string, error) {
	// --end-of-options prevents a revision beginning with '-' from becoming a
	// Git option. The commit peel rejects trees, tags that do not resolve, and
	// arbitrary rev expressions that are not commits.
	out, err := git(root, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve Git revision %q: %w", revision, err)
	}
	commit := strings.TrimSpace(string(out))
	if len(commit) != 40 && len(commit) != 64 {
		return "", fmt.Errorf("Git returned a non-full commit id for %q", revision)
	}
	if _, err := hex.DecodeString(commit); err != nil {
		return "", fmt.Errorf("Git returned an invalid commit id: %w", err)
	}
	return strings.ToLower(commit), nil
}

func loadCommit(root, commit string, s *Snapshot, limits Limits) error {
	// NUL-delimited output preserves all Git path bytes except NUL, which is
	// forbidden by Git itself. --long supplies blob object IDs for cat-file.
	out, err := git(root, "ls-tree", "-r", "-z", "--full-tree", "--long", commit)
	if err != nil {
		return fmt.Errorf("list Git tree %s: %w", commit, err)
	}
	files := make([]treeFile, 0)
	var total int64
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := bytes.Cut(record, []byte{'\t'})
		if !ok {
			return errors.New("malformed git ls-tree record")
		}
		fields := strings.Fields(string(header))
		if len(fields) != 4 {
			return errors.New("malformed git ls-tree header")
		}
		mode, kind, oid := fields[0], fields[1], fields[2]
		blobSize, parseErr := strconv.ParseInt(fields[3], 10, 64)
		if parseErr != nil || blobSize < 0 {
			return fmt.Errorf("invalid Git blob size for %q", string(name))
		}
		p := string(name)
		if excludedFilePath(p) {
			continue
		}
		if err := validateRepoPath(p); err != nil {
			return err
		}
		if kind == "commit" || mode == "160000" {
			return fmt.Errorf("Git submodule is not a source file: %q", p)
		}
		if mode == "120000" {
			return fmt.Errorf("Git symlink is not a source file: %q", p)
		}
		if kind != "blob" || (mode != "100644" && mode != "100755") {
			return fmt.Errorf("unsupported Git tree entry %q (mode %s, type %s)", p, mode, kind)
		}
		if blobSize > limits.MaxFileBytes {
			return fmt.Errorf("source file %q exceeds per-file limit", p)
		}
		if total > limits.MaxTotalBytes-blobSize {
			return fmt.Errorf("source snapshot exceeds byte limit at %q", p)
		}
		if len(files) >= limits.MaxFiles {
			return errors.New("source snapshot exceeds file-count limit")
		}
		total += blobSize
		files = append(files, treeFile{path: p, mode: mode, oid: oid, size: blobSize})
	}
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.path
	}
	if err := validatePortablePaths(paths); err != nil {
		return err
	}
	return loadBlobs(root, files, s)
}

func loadBlobs(root string, files []treeFile, s *Snapshot) error {
	if len(files) == 0 {
		return nil
	}
	var input bytes.Buffer
	for _, file := range files {
		input.WriteString(file.oid)
		input.WriteByte('\n')
	}
	cmd := gitCommand(root, "cat-file", "--batch")
	cmd.Stdin = &input
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open Git blob stream: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Git blob stream: %w", err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	r := bufio.NewReaderSize(stdout, 512)
	for _, file := range files {
		header, err := readBatchHeader(r)
		if err != nil {
			return fmt.Errorf("read Git blob header for %q: %w", file.path, err)
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[0] != file.oid || fields[1] != "blob" {
			return fmt.Errorf("unexpected Git blob header for %q: %q", file.path, header)
		}
		size, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || size != file.size {
			return fmt.Errorf("Git blob size changed while loading %q", file.path)
		}
		data := make([]byte, int(file.size))
		if _, err := io.ReadFull(r, data); err != nil {
			return fmt.Errorf("read Git blob for %q: %w", file.path, err)
		}
		separator, err := r.ReadByte()
		if err != nil || separator != '\n' {
			return fmt.Errorf("malformed Git blob separator for %q", file.path)
		}
		s.Files[file.path] = data
		s.Modes[file.path] = file.mode
	}
	if _, err := r.ReadByte(); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing data in Git blob stream")
		}
		return fmt.Errorf("finish Git blob stream: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		waited = true
		return fmt.Errorf("Git blob stream failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	waited = true
	return nil
}

func readBatchHeader(r *bufio.Reader) (string, error) {
	const maxHeader = 256
	var header strings.Builder
	for header.Len() <= maxHeader {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '\n' {
			return header.String(), nil
		}
		header.WriteByte(b)
	}
	return "", errors.New("Git blob header exceeds limit")
}

func git(root string, args ...string) ([]byte, error) {
	return GitOutput(root, args...)
}

// GitOutput runs Git against root while discarding inherited Git environment
// variables. Those variables can silently redirect repository discovery,
// indexes, objects or configuration to another checkout. The exact repository
// is selected explicitly and marked safe for Git's ownership check.
func GitOutput(root string, args ...string) ([]byte, error) {
	cmd := gitCommand(root, args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
	}
	return nil, err
}

// CleanGitEnv returns the process environment without Git-specific variables.
// It is also used for child gate processes so an ambient GIT_DIR cannot make
// their repository checks inspect a different checkout.
func CleanGitEnv() []string {
	current := os.Environ()
	clean := make([]string, 0, len(current))
	for _, entry := range current {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || strings.HasPrefix(strings.ToUpper(key), "GIT_") {
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}

func gitCommand(root string, args ...string) *exec.Cmd {
	abs, err := filepath.Abs(root)
	if err != nil {
		// Callers validate and resolve root before invoking Git. Preserve a
		// deterministic failing command if Abs nevertheless fails.
		abs = root
	}
	gitArgs := append([]string{"--no-replace-objects", "-c", "safe.directory=" + filepath.ToSlash(abs), "-C", abs}, args...)
	cmd := exec.Command("git", gitArgs...)
	cmd.Env = CleanGitEnv()
	return cmd
}

// Digest returns a stable SHA-256 digest of sorted paths, Git modes and bytes.
func (s *Snapshot) Digest() string {
	if s == nil {
		return ""
	}
	h := sha256.New()
	paths := make([]string, 0, len(s.Files))
	for p := range s.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var n [8]byte
	for _, p := range paths {
		writeField := func(b []byte) {
			binary.BigEndian.PutUint64(n[:], uint64(len(b)))
			_, _ = h.Write(n[:])
			_, _ = h.Write(b)
		}
		writeField([]byte(p))
		writeField([]byte(s.Modes[p]))
		writeField(s.Files[p])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Materialize writes a snapshot into destination, which must be absent or an
// empty real directory. Every path is validated before anything is written.
func Materialize(s *Snapshot, destination string) error {
	return MaterializeWithLimits(s, destination, DefaultLimits())
}

// MaterializeWithLimits is Materialize with caller-selected resource limits.
func MaterializeWithLimits(s *Snapshot, destination string, limits Limits) error {
	if s == nil {
		return errors.New("snapshot is nil")
	}
	if destination == "" {
		return errors.New("destination is empty")
	}
	if err := validateLimits(limits); err != nil {
		return err
	}
	if len(s.Files) > limits.MaxFiles {
		return errors.New("snapshot exceeds file-count limit")
	}
	var total int64
	paths := make([]string, 0, len(s.Files))
	for p, data := range s.Files {
		if err := validateRepoPath(p); err != nil {
			return err
		}
		mode := s.Modes[p]
		if mode != "100644" && mode != "100755" {
			return fmt.Errorf("unsupported snapshot mode %q for %q", mode, p)
		}
		if int64(len(data)) > limits.MaxFileBytes || total > limits.MaxTotalBytes-int64(len(data)) {
			return fmt.Errorf("snapshot exceeds byte limit at %q", p)
		}
		total += int64(len(data))
		paths = append(paths, p)
	}
	if err := validatePortablePaths(paths); err != nil {
		return err
	}
	sort.Strings(paths)
	dest, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	if err := rejectSymlinkAncestors(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	info, err := os.Lstat(dest)
	if err != nil {
		return fmt.Errorf("stat destination: %w", err)
	}
	if isSymlink(info) || !info.IsDir() {
		return errors.New("destination must be a real directory")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		return fmt.Errorf("read destination: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("destination directory is not empty")
	}
	for _, p := range paths {
		if err := writeMaterializedFile(dest, p, s.Files[p], s.Modes[p]); err != nil {
			return err
		}
	}
	return nil
}

func writeMaterializedFile(dest, repoPath string, data []byte, mode string) error {
	parts := strings.Split(repoPath, "/")
	current := dest
	for _, component := range parts[:len(parts)-1] {
		current = filepath.Join(current, component)
		if err := os.Mkdir(current, 0755); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create destination directory for %q: %w", repoPath, err)
		}
		info, err := os.Lstat(current)
		if err != nil || isSymlink(info) || !info.IsDir() {
			return fmt.Errorf("unsafe destination directory for %q", repoPath)
		}
	}
	file := filepath.Join(current, parts[len(parts)-1])
	perm := os.FileMode(0644)
	if mode == "100755" {
		perm = 0755
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("create destination file %q: %w", repoPath, err)
	}
	_, writeErr := io.Copy(f, bytes.NewReader(data))
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("write destination file %q: %w", repoPath, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close destination file %q: %w", repoPath, closeErr)
	}
	return nil
}

func rejectSymlinkAncestors(p string) error {
	vol := filepath.VolumeName(p)
	current := vol + string(filepath.Separator)
	rest := strings.TrimPrefix(p, current)
	if vol == "" {
		current = string(filepath.Separator)
		rest = strings.TrimPrefix(p, current)
	}
	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("stat destination ancestor: %w", err)
		}
		if isSymlink(info) {
			return fmt.Errorf("path traverses symlink or reparse point: %s", current)
		}
	}
	return nil
}

func validateRepoPath(p string) error {
	if p == "" || !utf8.ValidString(p) || strings.ContainsRune(p, 0) || strings.Contains(p, `\`) || strings.Contains(p, ":") || path.IsAbs(p) {
		return fmt.Errorf("unsafe repository path %q", p)
	}
	clean := path.Clean(p)
	if clean != p || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("unsafe repository path %q", p)
	}
	for _, component := range strings.Split(p, "/") {
		if component == "" || component == "." || component == ".." || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
			return fmt.Errorf("unsafe repository path %q", p)
		}
		for _, r := range component {
			if r < 32 || strings.ContainsRune(`<>"|?*`, r) {
				return fmt.Errorf("unsafe repository path %q", p)
			}
		}
		base := strings.ToUpper(strings.SplitN(component, ".", 2)[0])
		if windowsReservedName(base) {
			return fmt.Errorf("unsafe Windows device path %q", p)
		}
	}
	return nil
}

func snapshotPaths(s *Snapshot) []string {
	paths := make([]string, 0, len(s.Files))
	for p := range s.Files {
		paths = append(paths, p)
	}
	return paths
}

// validatePortablePaths rejects names that collapse to the same path on a
// case-insensitive filesystem, including a file/directory collision in any
// parent component.
func validatePortablePaths(paths []string) error {
	seen := make(map[string]struct {
		name string
		dir  bool
	}, len(paths)*2)
	for _, p := range paths {
		parts := strings.Split(p, "/")
		for i := range parts {
			name := strings.Join(parts[:i+1], "/")
			isDir := i < len(parts)-1
			key := foldPath(name)
			if prior, ok := seen[key]; ok && (prior.name != name || prior.dir != isDir) {
				return fmt.Errorf("case-insensitive path collision between %q and %q", prior.name, name)
			}
			seen[key] = struct {
				name string
				dir  bool
			}{name: name, dir: isDir}
		}
	}
	return nil
}

// ValidateIncludedPaths validates prospective repository paths using the same
// safety, inclusion, and portable-name rules as source snapshots. Callers pass
// existing snapshot paths together with new paths they plan to add.
func ValidateIncludedPaths(paths []string) error {
	for _, p := range paths {
		if err := validateRepoPath(p); err != nil {
			return err
		}
		folded := strings.ToLower(p)
		if excludedFilePath(p) || excludedDirectoryPath(p) || excludedFilePath(folded) || excludedDirectoryPath(folded) {
			return fmt.Errorf("repository path %q is excluded from Markitect source", p)
		}
	}
	return validatePortablePaths(paths)
}

func foldPath(p string) string {
	var folded strings.Builder
	for _, r := range p {
		min := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < min {
				min = next
			}
		}
		folded.WriteRune(min)
	}
	return folded.String()
}

func windowsReservedName(base string) bool {
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return true
	}
	return false
}
