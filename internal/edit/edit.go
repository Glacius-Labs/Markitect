// Package edit computes the changes of the commands that write a spec: new,
// move and apply. Each reads only the file systems it is handed and returns a
// change set, files to move and files to write, which the caller applies.
//
// The rules are in docs/guides/language-reference.md.
package edit

import (
	"errors"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// Changes is what a command changes in a spec folder: first the moves, then
// the writes, each at its final path below the spec folder.
type Changes struct {
	Moves  []Move
	Writes []Write
}

// Move moves the file at From to To.
type Move struct{ From, To string }

// Write writes Data to Path. Create means that no file may exist there yet.
type Write struct {
	Path   string
	Data   []byte
	Create bool
}

// Refusal is a command that cannot be done, with every reason found. Nothing
// is changed.
type Refusal struct{ Problems []string }

func (r *Refusal) Error() string { return strings.Join(r.Problems, "\n") }

func refuse(problems ...string) error { return &Refusal{Problems: problems} }

// IsRefusal reports whether err is a Refusal.
func IsRefusal(err error) bool {
	var r *Refusal
	return errors.As(err, &r)
}

// DocumentPath turns a document's name on the command line into its path
// below the spec folder: backslashes become slashes and ".md" is added when it
// is missing. ok is false for a path that leaves the spec folder, is absolute,
// has a scheme or names no file.
func DocumentPath(arg string) (string, bool) {
	p := strings.TrimSpace(strings.ReplaceAll(arg, "\\", "/"))
	if !strings.HasSuffix(strings.ToLower(p), ".md") {
		p += ".md"
	}
	if strings.HasPrefix(p, "/") {
		return "", false
	}
	p = path.Clean(p)
	if p == ".." || strings.HasPrefix(p, "../") || hasScheme(p) || strings.EqualFold(path.Base(p), ".md") {
		return "", false
	}
	return p, true
}

// folderPath turns a folder on the command line into a path below its base,
// "" for the base itself; ok is false when it leaves the base.
func folderPath(arg string) (string, bool) {
	p := strings.TrimSpace(strings.ReplaceAll(arg, "\\", "/"))
	if p == "" {
		p = "."
	}
	if strings.HasPrefix(p, "/") {
		return "", false
	}
	p = path.Clean(p)
	if p == ".." || strings.HasPrefix(p, "../") || hasScheme(p) {
		return "", false
	}
	if p == "." {
		return "", true
	}
	return p, true
}

func hasScheme(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z':
		case i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '.' || c == '-'):
		case i > 0 && c == ':':
			return true
		default:
			return false
		}
	}
	return false
}

// exists reports whether anything, a file or a folder, is at p in fsys.
func exists(fsys fs.FS, p string) bool {
	_, err := fs.Stat(fsys, p)
	return err == nil
}

// dir returns the folder of a path below the spec folder, "" at its root.
func dir(p string) string {
	if d := path.Dir(p); d != "." {
		return d
	}
	return ""
}

// rel returns the relative path from a folder below the spec folder to
// target, a path below it or above it.
func rel(target, folder string) string {
	var a []string
	for _, s := range strings.Split(folder, "/") {
		if s != "" {
			a = append(a, s)
		}
	}
	b := strings.Split(target, "/")
	n := 0
	for n < len(a) && n < len(b)-1 && a[n] == b[n] {
		n++
	}
	parts := make([]string, 0, len(a)-n+len(b)-n)
	for range a[n:] {
		parts = append(parts, "..")
	}
	return strings.Join(append(parts, b[n:]...), "/")
}

// quote percent-encodes every byte of p except letters, digits, "_.-~" and
// "/".
func quote(p string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte("_.-~/", c) >= 0 {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
	return b.String()
}

// needsQuote reports whether a path cannot stand as a destination outside
// < and >: it holds white space, a parenthesis or an angle bracket.
func needsQuote(p string) bool {
	return strings.ContainsFunc(p, func(r rune) bool {
		return unicode.IsSpace(r) || r >= 0x1c && r <= 0x1f || strings.ContainsRune("()<>", r)
	})
}

// rewrite returns the document's bytes with the path of each destination of
// list for which newPath gives one replaced by it, keeping the #section and the < >
// form. The path is percent-encoded when the destination was, or when it
// cannot stand plain. Everything else stays byte for byte.
func rewrite(d *lang.Document, list []lang.Ref, newPath func(lang.Ref) (string, bool)) ([]byte, error) {
	refs := append([]lang.Ref(nil), list...)
	sort.SliceStable(refs, func(i, j int) bool { return refs[i].Start < refs[j].Start })
	var out []byte
	last := 0
	for _, ref := range refs {
		p, ok := newPath(ref)
		if !ok {
			continue
		}
		if ref.Start < 0 {
			return nil, refuse(d.Path + ": cannot rewrite the link at line " + strconv.Itoa(ref.Line))
		}
		if strings.Contains(ref.Dest, "%") || !ref.Angle && needsQuote(p) {
			p = quote(p)
		}
		out = append(out, d.Source[last:ref.Start]...)
		out = append(out, p+ref.Fragment()...)
		last = ref.End
	}
	return append(out, d.Source[last:]...), nil
}
