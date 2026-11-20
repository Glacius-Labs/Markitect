package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// OutsideFS is a project file system that can also reach a path outside the
// project, as a .git file may name one. A file system without it treats such
// a .git file as naming no repository, so that the folder is walked.
type OutsideFS interface {
	fs.FS
	// Outside returns a file system that holds name, a path that is absolute
	// or that leaves the project folder with "..", relative to it, and the path
	// of name in that file system.
	Outside(name string) (fs.FS, string, bool)
}

// Walk returns every file of the project at fsys, by its path below the
// project folder with forward slashes, sorted. It leaves out every entry named
// .git, in any case, .markitect/ at the top, and what the project's
// .gitignore files or .markitect/ignore exclude. A folder that is another valid repository or
// worktree is one entry and is not walked. A symbolic link is one entry and is
// not followed. A Windows junction, or any other entry that leads to a folder
// without being a symbolic link, is walked like a folder, and dropped when it
// leads back to a folder the walk is in. Any other kind of file, a FIFO, a
// socket, a device or a junction that leads nowhere, is left out.
func Walk(fsys fs.FS) ([]string, error) {
	own, err := readIgnore(fsys, IgnoreFile)
	if err != nil {
		return nil, err
	}
	w := &walker{fsys: fsys, own: parseIgnore("", own), git: map[string][]pattern{}}
	if err := w.walk("", nil); err != nil {
		return nil, err
	}
	sort.Strings(w.files)
	return w.files, nil
}

// walk walks the folder p, "" for the top, below the folders above.
func (w *walker) walk(p string, above []string) error {
	name := p
	if name == "" {
		name = "."
	}
	entries, err := fs.ReadDir(w.fsys, name)
	if err != nil {
		return err
	}
	above = append(above, name)
	for _, e := range entries {
		child := path.Join(p, e.Name())
		if strings.EqualFold(e.Name(), ".git") || child == Dir {
			continue
		}
		mode := e.Type()
		dir, through := e.IsDir(), false
		if !dir && mode&fs.ModeSymlink == 0 && !mode.IsRegular() {
			// A junction is walked like a folder when it leads to one. Any
			// other kind of file, such as a FIFO, a socket or a device, is
			// left out, as Git leaves it out, so that reading it never blocks.
			info, err := fs.Stat(w.fsys, child)
			if err != nil || !info.IsDir() {
				continue
			}
			dir, through = true, true
		}
		excluded, err := w.excluded(child, dir)
		if err != nil {
			return err
		}
		switch {
		case excluded:
		case !dir:
			w.files = append(w.files, child)
		case nestedRepository(w.fsys, child):
			w.files = append(w.files, child)
		case through && w.cycle(child, above):
		default:
			if err := w.walk(child, above); err != nil {
				return err
			}
		}
	}
	return nil
}

// cycle says whether the folder a junction leads to is one the walk is in.
func (w *walker) cycle(p string, above []string) bool {
	target, err := fs.Stat(w.fsys, p)
	if err != nil {
		return true
	}
	for _, a := range above {
		if info, err := fs.Stat(w.fsys, a); err == nil && os.SameFile(info, target) {
			return true
		}
	}
	return false
}

// repositoryEntry is what another repository, one entry of the walk, reads as.
const repositoryEntry = "a repository of its own\n"

// Content reads a file the walk lists: a symbolic link, or a junction that
// leads to no folder, as where it points, with forward slashes, and another
// repository as one fixed line, as Git keeps them.
func Content(fsys fs.FS, name string) ([]byte, error) {
	info, err := fs.Lstat(fsys, name)
	if err != nil {
		return nil, err
	}
	mode := info.Mode()
	switch {
	case mode&fs.ModeSymlink != 0:
		to, err := fs.ReadLink(fsys, name)
		return []byte(filepath.ToSlash(to)), err
	case mode.IsDir() || !mode.IsRegular() && isFolder(fsys, name):
		if nestedRepository(fsys, name) {
			return []byte(repositoryEntry), nil
		}
		return nil, fmt.Errorf("%s: a folder", name)
	case !mode.IsRegular():
		if to, err := fs.ReadLink(fsys, name); err == nil {
			return []byte(filepath.ToSlash(to)), nil
		}
	}
	return fs.ReadFile(fsys, name)
}

func isFolder(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && info.IsDir()
}

// textProbe is how far Hash looks for a NUL byte, as Git does.
const textProbe = 8000

// Hash is the SHA-256 of a file's content as "sha256:" and 64 hexadecimal
// digits. A text file, one without a NUL byte in its first 8000 bytes as Git
// decides, whatever its encoding, hashes with each CRLF as LF, so that a
// checkout on Windows and one on Linux agree; any other file hashes as it is.
func Hash(data []byte) string {
	if bytes.IndexByte(data[:min(len(data), textProbe)], 0) < 0 {
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// HashFile is the Hash of a file the walk lists, read as Content reads it.
func HashFile(fsys fs.FS, name string) (string, error) {
	data, err := Content(fsys, name)
	if err != nil {
		return "", err
	}
	return Hash(data), nil
}

// nestedRepository says whether a folder is a valid repository or worktree of
// its own, as Git decides: its .git is a Git folder, or a file whose gitdir:
// line names one, relative to the folder or absolute.
func nestedRepository(fsys fs.FS, dir string) bool {
	git := path.Join(dir, ".git")
	info, err := fs.Stat(fsys, git)
	switch {
	case err != nil:
		return false
	case info.IsDir():
		return gitFolder(fsys, git)
	}
	data, err := fs.ReadFile(fsys, git)
	if err != nil {
		return false
	}
	line, _, _ := strings.Cut(string(data), "\n")
	target, ok := strings.CutPrefix(strings.TrimRight(line, "\r"), "gitdir: ")
	if !ok || target == "" {
		return false
	}
	target = filepath.ToSlash(target)
	if !filepath.IsAbs(filepath.FromSlash(target)) && !strings.HasPrefix(target, "/") {
		if inside := path.Join(dir, target); inside != ".." && !strings.HasPrefix(inside, "../") {
			return gitFolder(fsys, inside)
		}
		target = path.Join(dir, target)
	}
	outside, ok := fsys.(OutsideFS)
	if !ok {
		return false
	}
	host, name, ok := outside.Outside(target)
	return ok && gitFolder(host, name)
}

// validHead says whether a HEAD file is one Git accepts: "ref:" and a ref
// below refs/, or a full object hash of SHA-1 or SHA-256.
func validHead(data []byte) bool {
	s := string(data)
	if rest, ok := strings.CutPrefix(s, "ref:"); ok {
		return strings.HasPrefix(strings.TrimLeft(rest, " \t\n\r"), "refs/")
	}
	for _, n := range []int{64, 40} {
		if len(s) >= n && isHex(s[:n]) {
			return true
		}
	}
	return false
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) && (s[i] < 'a' || s[i] > 'f') {
			return false
		}
	}
	return true
}

// gitFolder says whether a folder is a Git folder: it holds a valid HEAD, and objects/
// and refs/ lie in it or in the common folder its commondir file names, as in
// a worktree's folder.
func gitFolder(fsys fs.FS, dir string) bool {
	head, err := fs.ReadFile(fsys, path.Join(dir, "HEAD"))
	if err != nil || !validHead(head) {
		return false
	}
	common := dir
	if data, err := fs.ReadFile(fsys, path.Join(dir, "commondir")); err == nil {
		c := filepath.ToSlash(strings.TrimSpace(string(data)))
		if c == "" || strings.HasPrefix(c, "/") || filepath.IsAbs(filepath.FromSlash(c)) {
			return false
		}
		common = path.Join(dir, c)
	}
	for _, sub := range []string{"objects", "refs"} {
		if !isFolder(fsys, path.Join(common, sub)) {
			return false
		}
	}
	return true
}

// walker holds the files and the ignore files of one walk.
type walker struct {
	fsys  fs.FS
	files []string
	// own is .markitect/ignore; git holds the patterns of the .gitignore file
	// of each folder read so far.
	own []pattern
	git map[string][]pattern
}
