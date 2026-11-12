package spec

import (
	"regexp"
	"sort"
	"strings"
)

// CoreKind is the one Definition Core offers: the Kind of every Kind.
const CoreKind = "Kind"

// RootPackage is the path of the Root Package, the spec folder itself.
const RootPackage = ""

// File is one input file, its path relative to the folder it was read from,
// with forward slashes.
type File struct {
	Path string
	Data []byte
}

// LibrarySource is one Library the manifest names, with its files.
type LibrarySource struct {
	Name    string
	Version string
	Dir     string // where the files come from, for diagnostics
	Files   []File
}

// Source is everything Compile reads: the spec folder and the Libraries.
type Source struct {
	Dir       string // repository-relative spec folder, for diagnostics
	Files     []File
	Libraries []LibrarySource
}

// Spec is a compiled spec. It is complete even when Compile reports errors;
// unresolved links then have an empty Target.
type Spec struct {
	Definitions map[string]*Definition // by ID: "members/member", "std/Rule", "Kind"
	Packages    map[string]*Package    // by path; RootPackage is ""
	// Sorting holds the folders of the spec that are no Package: they hold no
	// anchor, and what lies in them belongs to the Package above.
	Sorting   map[string]bool
	Libraries map[string]*Library
	// Edges are the explicit dependencies between Packages: from, to, and the
	// links that make them. The implicit edge from a child to its parent is
	// not listed.
	Edges map[string]map[string][]Use
}

// Library is one loaded Library.
type Library struct {
	Name        string
	Version     string
	Definitions []string
}

// Package is one part of the spec: the spec folder itself, or a folder that
// holds a Definition of its own name, its anchor. A folder without an anchor
// only sorts, and what lies in it belongs to the Package above.
type Package struct {
	Path        string
	Parent      string   // the Package above, which need not be the folder above
	Children    []string // the Packages directly below, sorted
	Definitions []string // own Definition IDs, sorted
	Anchor      string   // the Definition that says what the Package is; "" for Root
}

// Definition is one Markdown file of the spec, of a Library, or Core's Kind.
type Definition struct {
	ID       string
	Library  string // "" for the project's spec, "core" for Kind
	Package  string // Package path for the project's spec
	File     string // repository-relative file
	Kind     string // resolved Kind ID; "" when it does not resolve
	Internal bool
	Title    string
	Words    int
	Data     []byte // the file's bytes, for comparison
	// Links holds the front matter's Kind reference first (Kind is true),
	// then every body link.
	Links []Link
}

// Link is one resolved reference of a Definition.
type Link struct {
	Raw     string // the path as written
	Target  string // Definition ID or Package path; "" when unresolved
	Package bool   // the target is a Package
	Kind    bool   // the front matter's kind
	Line    int
}

// Use is one link that makes a Package depend on another.
type Use struct {
	From   string // Definition ID
	Target string // linked Definition ID or Package path
	Kind   bool
	File   string
	Line   int
}

// Own reports whether the Definition belongs to the project's own spec, not to a Library.
func (d *Definition) Own() bool { return d.Library == "" }

// PackageName returns a Package path for people: "Root" for the Root Package.
func PackageName(path string) string {
	if path == RootPackage {
		return "Root"
	}
	return path
}

// Parent returns the Package above a Package: Root for Root and for a path
// that names no Package.
func (s *Spec) Parent(pkg string) string {
	if p := s.Packages[pkg]; p != nil {
		return p.Parent
	}
	return RootPackage
}

// PackageOf returns the Package a Definition ID belongs to. For an ID the
// spec no longer holds it is the nearest folder above that is a Package, and
// Root when there is none.
func (s *Spec) PackageOf(id string) string {
	if d := s.Definitions[id]; d != nil && d.Own() {
		return d.Package
	}
	for at := FolderOf(id); at != RootPackage; at = FolderOf(at) {
		if s.Packages[at] != nil {
			return at
		}
	}
	return RootPackage
}

// FolderOf returns the folder a path lies in: "" for a path at the top.
func FolderOf(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return RootPackage
}

// AnchorOf returns the ID a Package's anchor has: the folder's own name inside
// the folder, such as members/members for the Package members.
func AnchorOf(pkg string) string {
	return pkg + "/" + pkg[strings.LastIndexByte(pkg, '/')+1:]
}

// Within reports whether Package p is q or lies below it.
func Within(p, q string) bool {
	return q == RootPackage || p == q || strings.HasPrefix(p, q+"/")
}

// SortedPackages returns every Package path in order, Root first.
func (s *Spec) SortedPackages() []string {
	out := make([]string, 0, len(s.Packages))
	for p := range s.Packages {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// OwnDefinitions returns the IDs of the project's own Definitions, sorted.
func (s *Spec) OwnDefinitions() []string {
	var out []string
	for id, d := range s.Definitions {
		if d.Own() {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

var segment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// ValidSegment reports whether s may name a Package, a Definition or a Library.
func ValidSegment(s string) bool { return segment.MatchString(s) }

// ValidPath reports whether every segment of a slash-separated path is valid.
func ValidPath(p string) bool {
	if p == "" {
		return false
	}
	for _, s := range strings.Split(p, "/") {
		if !ValidSegment(s) {
			return false
		}
	}
	return true
}
