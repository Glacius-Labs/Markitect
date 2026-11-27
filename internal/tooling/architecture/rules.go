package architecture

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Layer is one row of the layer list. A package may import a package of
// another layer only when its own layer lists that layer in Imports, and a
// package of its own layer only when Imports lists its own layer.
type Layer struct {
	Name    string
	Title   string // section heading in the code map
	Meaning string // one line for the code map
	Imports []string
	// TestOnly layers may be imported only by test files.
	TestOnly bool
	// Standalone layers import no Markitect package, not even from tests.
	Standalone bool
}

// Package assigns one Go package directory to a layer and names the document
// that owns its behavior. The code map takes its summary from the package doc
// comment.
type Package struct {
	Path  string // repository-relative directory with forward slashes
	Layer string
	Doc   string // repository-relative owning document, optionally with #anchor
}

// Exception admits one import edge that the layers do not allow. A TestOnly
// exception admits it only from test files.
type Exception struct {
	From, To string
	TestOnly bool
	Reason   string
}

// Rules is an architecture rule table: the layers, every package's layer and
// the named exceptions.
type Rules struct {
	Layers     []Layer
	Packages   []Package
	Exceptions []Exception
}

// Table is the repository's one architecture rule table. The import gate and the
// generated code map derive from it.
var Table = Rules{
	Layers: []Layer{
		{Name: "build", Title: "Build data", Meaning: "Version identity and embedded license notices."},
		{Name: "language", Title: "Language", Meaning: "The model language over a supplied file system: documents, types, sections, links and check."},
		{Name: "library", Title: "Library", Meaning: "The library: the reference values and question catalog of the default's modules, built into the tool."},
		{Name: "signals", Title: "Signals", Meaning: "The statements of a model, its signals and the topic view, over a parsed model.", Imports: []string{"language"}},
		{Name: "project", Title: "Project", Meaning: "A project over a supplied file system: the file .markitect/model, the project loaded with its model, its ignore file, and the difference between two projects: their models and, for each module, its files.", Imports: []string{"language"}},
		{Name: "command-line", Title: "Command line", Meaning: "The command line: its arguments, the project it works in, the commands that read its model, version and licenses, and the text and JSON output. It writes nothing.", Imports: []string{"build", "language", "library", "signals", "project"}},
		{Name: "entrypoint", Title: "Entrypoint", Meaning: "The thin entrypoint of markitect.", Imports: []string{"command-line"}},
		{Name: "maintenance", Title: "Maintenance", Meaning: "The import gate and the code map, separate from product execution."},
		{Name: "test-support", Title: "Test support", Meaning: "Hermetic fixtures.", TestOnly: true, Standalone: true},
		{Name: "standalone", Title: "Standalone tools", Meaning: "The command smoke.", Standalone: true},
	},
	Packages: []Package{
		{"internal/buildinfo", "build", "docs/architecture.md#distribution"},
		{"internal/lang", "language", "docs/guides/language-reference.md#documents"},
		{"internal/library", "library", "docs/guides/library.md"},
		{"internal/signals", "signals", "docs/guides/signals.md"},
		{"internal/project", "project", "docs/guides/project-reference.md"},
		{"internal/cli", "command-line", "docs/guides/language-reference.md#commands"},
		{"cmd/markitect", "entrypoint", "docs/guides/language-reference.md#commands"},
		{"internal/tooling/architecture", "maintenance", "docs/architecture.md#layers"},
		{"internal/testkit", "test-support", "docs/architecture.md#layers"},
		{"tools/smoke", "standalone", "docs/architecture.md#distribution"},
	},
}

func (r Rules) layer(name string) (Layer, bool) {
	for _, l := range r.Layers {
		if l.Name == name {
			return l, true
		}
	}
	return Layer{}, false
}

func (r Rules) pkg(p string) (Package, bool) {
	for _, candidate := range r.Packages {
		if candidate.Path == p {
			return candidate, true
		}
	}
	return Package{}, false
}

// allows reports whether a package of layer from may import one of layer to.
func (r Rules) allows(from, to string) bool {
	l, _ := r.layer(from)
	return contains(l.Imports, to)
}

func (r Rules) exception(from, to string, test bool) bool {
	for _, x := range r.Exceptions {
		if x.From == from && x.To == to && (test || !x.TestOnly) {
			return true
		}
	}
	return false
}

var layerName = regexp.MustCompile(`^[a-z][a-z-]*[a-z]$`)

// Validate reports every inconsistency of the table itself: unknown or
// duplicate names, a layer importing a layer listed after it, unordered
// packages, standalone layers with imports and exceptions that name unknown
// packages or that the layers already allow.
func (r Rules) Validate() []string {
	var problems []string
	report := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	line := func(s string) bool { return strings.TrimSpace(s) != "" && !strings.ContainsAny(s, "|\n") }
	order := map[string]int{}
	for i, l := range r.Layers {
		if !layerName.MatchString(l.Name) {
			report("layer %q: name must be lower-case words joined by hyphens", l.Name)
		}
		if _, dup := order[l.Name]; dup {
			report("layer %q: duplicate", l.Name)
		}
		order[l.Name] = i
		if !line(l.Title) || !line(l.Meaning) {
			report("layer %q: title and meaning must be single non-empty lines without '|'", l.Name)
		}
		if l.Standalone && len(l.Imports) != 0 {
			report("layer %q: a standalone layer imports nothing", l.Name)
		}
	}
	for _, l := range r.Layers {
		seen := map[string]bool{}
		for _, to := range l.Imports {
			target, known := r.layer(to)
			switch {
			case !known:
				report("layer %q: imports unknown layer %q", l.Name, to)
			case seen[to]:
				report("layer %q: imports layer %q twice", l.Name, to)
			case target.TestOnly:
				report("layer %q: test-only layer %q needs no import rule", l.Name, to)
			case order[to] > order[l.Name]:
				report("layer %q: may import only itself and layers listed before it, not %q", l.Name, to)
			}
			seen[to] = true
		}
	}
	seen := map[string]bool{}
	for i, p := range r.Packages {
		if p.Path == "" || p.Path != path.Clean(p.Path) || path.IsAbs(p.Path) || strings.HasPrefix(p.Path, "../") || strings.Contains(p.Path, "\\") {
			report("package %q: path must be a clean repository-relative path with forward slashes", p.Path)
		}
		if seen[p.Path] {
			report("package %q: duplicate", p.Path)
		}
		seen[p.Path] = true
		rank, known := order[p.Layer]
		if !known {
			report("package %q: unknown layer %q", p.Path, p.Layer)
		} else if i > 0 {
			prev := r.Packages[i-1]
			if prevRank, ok := order[prev.Layer]; ok && (prevRank > rank || prevRank == rank && prev.Path >= p.Path) {
				report("package %q: packages must be ordered by layer, then by path", p.Path)
			}
		}
		if strings.TrimSpace(p.Doc) == "" {
			report("package %q: owning document is missing", p.Path)
		}
	}
	pairs := map[[2]string]bool{}
	for _, x := range r.Exceptions {
		from, fromKnown := r.pkg(x.From)
		to, toKnown := r.pkg(x.To)
		name := x.From + " -> " + x.To
		switch {
		case !fromKnown || !toKnown:
			report("exception %s: names an unknown package", name)
		case x.From == x.To:
			report("exception %s: a package may not except itself", name)
		case pairs[[2]string{x.From, x.To}]:
			report("exception %s: duplicate", name)
		case r.allows(from.Layer, to.Layer):
			report("exception %s: layer %s already may import layer %s", name, from.Layer, to.Layer)
		default:
			fl, _ := r.layer(from.Layer)
			tl, _ := r.layer(to.Layer)
			if fl.Standalone || tl.TestOnly {
				report("exception %s: standalone and test-only layers take no exceptions", name)
			}
		}
		pairs[[2]string{x.From, x.To}] = true
		if !line(x.Reason) {
			report("exception %s: reason must be a single non-empty line without '|'", name)
		}
	}
	return problems
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func sortViolations(out []Violation) {
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
}
