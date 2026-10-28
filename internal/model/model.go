package model

import (
	"regexp"
	"sort"
	"strings"
)

// CoreKind is the one Definition Core offers: the Kind of every Kind.
const CoreKind = "Kind"

// RootPackage is the path of the Root Package, the model folder itself.
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

// Source is everything Compile reads: the model folder and the Libraries.
type Source struct {
	Dir       string // repository-relative model folder, for diagnostics
	Files     []File
	Libraries []LibrarySource
}

// Model is a compiled model. It is complete even when Compile reports errors;
// unresolved links then have an empty Target.
type Model struct {
	Definitions map[string]*Definition // by ID: "members/member", "std/Rule", "Kind"
	Packages    map[string]*Package    // by path; RootPackage is ""
	Libraries   map[string]*Library
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

// Package is one folder of the model.
type Package struct {
	Path        string
	Parent      string
	Children    []string
	Definitions []string // own Definition IDs, sorted
}

// Definition is one Markdown file of the model, of a Library, or Core's Kind.
type Definition struct {
	ID       string
	Library  string // "" for the project's model, "core" for Kind
	Package  string // Package path for the project's model
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

// InModel reports whether the Definition belongs to the project's model.
func (d *Definition) InModel() bool { return d.Library == "" }

// PackageName returns a Package path for people: "Root" for the Root Package.
func PackageName(path string) string {
	if path == RootPackage {
		return "Root"
	}
	return path
}

// ParentOf returns the parent of a Package path.
func ParentOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return RootPackage
}

// Within reports whether Package p is q or lies below it.
func Within(p, q string) bool {
	return q == RootPackage || p == q || strings.HasPrefix(p, q+"/")
}

// Ancestors returns the Packages above p, nearest first, ending with Root.
func Ancestors(p string) []string {
	var out []string
	for p != RootPackage {
		p = ParentOf(p)
		out = append(out, p)
	}
	return out
}

// PackageOf returns the Package a model Definition ID lies in.
func PackageOf(id string) string { return ParentOf(id) }

// SortedPackages returns every Package path in order, Root first.
func (m *Model) SortedPackages() []string {
	out := make([]string, 0, len(m.Packages))
	for p := range m.Packages {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// ModelDefinitions returns the IDs of the project's own Definitions, sorted.
func (m *Model) ModelDefinitions() []string {
	var out []string
	for id, d := range m.Definitions {
		if d.InModel() {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// Subtree returns p and every Package below it, sorted.
func (m *Model) Subtree(p string) []string {
	var out []string
	for q := range m.Packages {
		if Within(q, p) {
			out = append(out, q)
		}
	}
	sort.Strings(out)
	return out
}

var segment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// ValidSegment reports whether s may name a Package, a Definition or a Library.
func ValidSegment(s string) bool { return segment.MatchString(s) }

// validPath reports whether every segment of a slash-separated path is valid.
func validPath(p string) bool {
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
