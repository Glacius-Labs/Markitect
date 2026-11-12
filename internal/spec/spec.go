package spec

import (
	"regexp"
	"sort"
	"strings"
)

// CoreKind and CorePackage are the two Definitions Core offers: the Kind of
// every Kind, and the Kind of the Definition that makes its folder a Package.
const (
	CoreKind    = "Kind"
	CorePackage = "Package"
)

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
	// PlainFolders holds the folders of the spec that are no Package. Such a
	// folder is a namespace and nothing more: what lies in it belongs to the
	// Package above.
	PlainFolders map[string]bool
	Libraries    map[string]*Library
	// HasParts holds the Definitions that are a whole: at least one other
	// Definition is a part of each.
	HasParts map[string]bool
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

// Package is one module of the spec: the spec folder itself, or a folder
// that holds its anchor, a Definition of kind Package that carries the
// folder's name. Any other folder is a namespace of the Package above.
type Package struct {
	Path        string
	Parent      string   // the Package above, which need not be the folder above
	Children    []string // the Packages directly below, sorted
	Definitions []string // own Definition IDs, sorted
	Anchor      string   // the Definition that says what the Package is; "" for Root
}

// Definition is one Markdown file of the spec or of a Library, or one of
// Core's two: Kind and Package.
type Definition struct {
	ID       string
	Library  string // "" for the project's spec, "core" for Kind and Package
	Package  string // Package path for the project's spec
	File     string // repository-relative file
	Kind     string // resolved Kind ID; "" when it does not resolve
	Internal bool
	// Whole is the Definition this one is a part of: the nearest Definition
	// above it that carries its folder's name, within the same Package. ""
	// for a Definition that is no part of another.
	Whole string
	Title string
	Words int
	Data  []byte // the file's bytes, for comparison
	// Links holds the front matter's Kind reference first (Kind is true),
	// then every body link.
	Links []Link
	// Fields are the labelled fields of the text, in order.
	Fields []Field
	// Schema is what a Kind declares about its Definitions; nil for a
	// Definition that is no Kind and for a Kind that declares nothing.
	Schema *Schema
}

// Link is one resolved reference of a Definition.
type Link struct {
	Raw     string // the path as written
	Target  string // Definition ID or Package path; "" when unresolved
	Package bool   // the target is a Package
	Kind    bool   // the front matter's kind
	Line    int
	// Field is the label of the field the link stands in; "" for a link in
	// the text before the first field.
	Field string
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

// Thing returns the Definition a folder that is no Package stands for: the
// one in it that carries the folder's name. A link to the folder means it.
func (s *Spec) Thing(folder string) (string, bool) {
	if !s.PlainFolders[folder] {
		return "", false
	}
	d := s.Definitions[AnchorOf(folder)]
	return AnchorOf(folder), d != nil && d.Own()
}

// Lookup returns the Definition a path names: the Definition of that ID, or
// the one a folder stands for as a thing with parts.
func (s *Spec) Lookup(target string) *Definition {
	if d := s.Definitions[target]; d != nil {
		return d
	}
	if thing, ok := s.Thing(target); ok {
		return s.Definitions[thing]
	}
	return nil
}

// IsPartOf reports whether a Definition is a part of another, directly or
// through a part.
func (s *Spec) IsPartOf(id, whole string) bool {
	for d := s.Definitions[id]; d != nil && d.Whole != ""; d = s.Definitions[d.Whole] {
		if d.Whole == whole {
			return true
		}
	}
	return false
}

// Parts returns the Definitions that are parts of a Definition, directly or
// through a part, in the order the Definition links them and then by ID.
func (s *Spec) Parts(id string) []string {
	whole := s.Definitions[id]
	if whole == nil {
		return nil
	}
	rank := map[string]int{}
	for i, l := range whole.Links {
		if _, seen := rank[l.Target]; !seen && l.Target != "" {
			rank[l.Target] = i + 1
		}
	}
	var out []string
	for _, other := range s.OwnDefinitions() {
		if s.IsPartOf(other, id) {
			out = append(out, other)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := rank[out[i]], rank[out[j]]
		if (a == 0) != (b == 0) {
			return a != 0
		}
		return a < b
	})
	return out
}

// Units returns what is built and verified as one: every Definition of the
// spec that is no part of another, or that has parts of its own, with the
// parts it covers, sorted. A thing with parts is one unit with its parts; a
// part that is itself a thing with parts is a unit of its own and is not
// covered by the thing above it.
func (s *Spec) Units() map[string][]string {
	parts := map[string][]string{}
	own := s.OwnDefinitions()
	for _, id := range own {
		if whole := s.Definitions[id].Whole; whole != "" {
			parts[whole] = append(parts[whole], id)
		}
	}
	units := map[string][]string{}
	for _, id := range own {
		if s.Definitions[id].Whole != "" && len(parts[id]) == 0 {
			continue
		}
		covered := []string{}
		for _, part := range parts[id] {
			if len(parts[part]) == 0 {
				covered = append(covered, part)
			}
		}
		units[id] = covered
	}
	return units
}

// UnitOf returns the unit that covers a Definition of the spec: the
// Definition itself, or the whole it is a part of when it has no parts of
// its own.
func (s *Spec) UnitOf(id string) string {
	d := s.Definitions[id]
	if d == nil || d.Whole == "" || s.HasParts[id] {
		return id
	}
	return d.Whole
}

// WhyNoPackage says, for a folder that is no Package, what it lacks: an
// anchor, or the kind Package on the Definition that carries its name.
func (s *Spec) WhyNoPackage(folder string) string {
	if d := s.Definitions[AnchorOf(folder)]; d != nil && d.Own() {
		return "its Definition " + AnchorOf(folder) + " is not of kind Package"
	}
	return "it holds no Definition of kind Package, " + AnchorOf(folder) + ".md"
}

// FolderOf returns the folder a path lies in: "" for a path at the top.
func FolderOf(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return RootPackage
}

// AnchorOf returns the ID a Package's anchor has: the folder's name inside
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
