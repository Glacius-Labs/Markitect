package realization

import (
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"go.yaml.in/yaml/v3"
)

// Input is everything Account reads.
type Input struct {
	Model *model.Model
	// Pages are the registry pages by their path below the registry folder:
	// "" for Root's page, "members" for .../registry/members.md.
	Pages []PageFile
	// Ignore is the ignore page, if the project has one.
	Ignore *model.File
	// Files are every repository file Git can see (tracked, or untracked and
	// not ignored) outside Markitect's own folder.
	Files []string
	// GitIgnored reports whether Git ignores a path that Files lacks.
	GitIgnored func(file string) bool
	// OwnDir is Markitect's own folder, ".markitect"; registry entries may not
	// name files below it.
	OwnDir string
}

// PageFile is the source of one registry page.
type PageFile struct {
	Package string // Package path the page belongs to; "" for Root
	File    string // repository-relative file, for diagnostics
	Data    []byte
}

// Registry is the checked realization record.
type Registry struct {
	Pages   map[string]*Page  // by Package path, including orphaned pages
	Owner   map[string]string // every accounted repository file -> owning Package
	Entries map[string]*Entry // every registered file -> its live entry
	Ignored map[string]bool   // files the ignore page matches
	Stale   []Stale
}

// Page is one Package's registry page.
type Page struct {
	Package  string
	File     string
	Patterns []Pattern
	Entries  []*Entry
	Orphaned bool // the Package no longer exists in the model
}

// Entry is one file row of a registry page.
type Entry struct {
	File     string
	Package  string // the page's Package
	PageFile string
	Line     int
	Realizes []string // resolved Definition IDs
	Reason   string
}

// Stale is realization the model or the repository no longer supports: a
// link to a Definition that is gone, an entry for a file that is gone, or the
// page of a Package that is gone. It is no compile error; impact makes it a
// checklist item of the Manager that must clean it up, and lint warns.
type Stale struct {
	Code     string `json:"code"`     // stale.definition, stale.file or stale.page
	Package  string `json:"package"`  // the Manager that must clean it up
	PageFile string `json:"pageFile"` // the registry page that holds it
	Line     int    `json:"line,omitempty"`
	File     string `json:"file,omitempty"`   // the registered file
	Target   string `json:"target,omitempty"` // the missing Definition
}

// Account parses the registry and ignore pages and accounts every repository
// file: each has exactly one owning Package and is listed on its owner's page
// or ignored.
func Account(in Input) (*Registry, model.Diagnostics) {
	a := &accountant{in: in, reg: &Registry{
		Pages:   map[string]*Page{},
		Owner:   map[string]string{},
		Entries: map[string]*Entry{},
		Ignored: map[string]bool{},
	}}
	var ignore []Pattern
	if in.Ignore != nil {
		ignore = a.parseIgnore(*in.Ignore)
	}
	pages := append([]PageFile(nil), in.Pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Package < pages[j].Package })
	for _, p := range pages {
		a.parsePage(p)
	}
	a.checkPatterns()
	visible := map[string]bool{}
	for _, f := range in.Files {
		visible[f] = true
		for _, p := range ignore {
			if p.Match(f) {
				a.reg.Ignored[f] = true
			}
		}
		if !a.reg.Ignored[f] {
			a.reg.Owner[f] = a.owner(f)
		}
	}
	a.checkEntries(visible)
	for _, f := range in.Files {
		owner, accounted := a.reg.Owner[f]
		if !accounted || a.reg.Entries[f] != nil || a.staleOnly[f] {
			continue
		}
		a.diags.Add("file.unregistered", f, 0, "neither registered nor ignored: list it on %s, the page of its owner %s, or add it to the ignore page", a.pagePath(owner), model.PackageName(owner))
	}
	sort.Slice(a.reg.Stale, func(i, j int) bool {
		x, y := a.reg.Stale[i], a.reg.Stale[j]
		if x.PageFile != y.PageFile {
			return x.PageFile < y.PageFile
		}
		return x.Line < y.Line
	})
	return a.reg, a.diags.Sorted()
}

type accountant struct {
	in        Input
	reg       *Registry
	diags     model.Diagnostics
	staleOnly map[string]bool // files listed only on orphaned pages
	ambiguous map[[2]string]bool
}

// pagePath names the registry page of a Package for messages.
func (a *accountant) pagePath(pkg string) string {
	if p := a.reg.Pages[pkg]; p != nil {
		return p.File
	}
	if pkg == model.RootPackage {
		return a.in.OwnDir + "/registry.md"
	}
	return a.in.OwnDir + "/registry/" + pkg + ".md"
}

func (a *accountant) parsePage(src PageFile) {
	page := &Page{Package: src.Package, File: src.File}
	if _, ok := a.in.Model.Packages[src.Package]; !ok {
		page.Orphaned = true
	}
	a.reg.Pages[src.Package] = page
	doc, err := model.ParsePage(src.Data)
	if err != nil {
		a.diags.Add("registry.front_matter", src.File, doc.FrontLine, "%v", err)
		return
	}
	for key, node := range doc.Front {
		if key != "paths" {
			a.diags.Add("registry.front_matter", src.File, node.Line, "unknown key %q: a registry page's front matter holds only paths", key)
			continue
		}
		if node.Kind != yaml.SequenceNode {
			a.diags.Add("registry.front_matter", src.File, node.Line, "paths must be a list of path patterns")
			continue
		}
		for _, item := range node.Content {
			if item.Kind != yaml.ScalarNode {
				a.diags.Add("registry.pattern", src.File, item.Line, "each path pattern is a string such as src/members/**")
				continue
			}
			if err := ValidatePattern(item.Value); err != nil {
				a.diags.Add("registry.pattern", src.File, item.Line, "%v", err)
				continue
			}
			page.Patterns = append(page.Patterns, Pattern{Text: item.Value, Line: item.Line + doc.FrontLine})
		}
	}
	if src.Package == model.RootPackage && len(page.Patterns) > 0 {
		a.diags.Add("registry.pattern", src.File, page.Patterns[0].Line, "Root owns every file no other Package claims; its page names no paths")
		page.Patterns = nil
	}
	forTables(doc, "file", func(cells []ast.Node, line int) {
		if len(cells) != 3 {
			a.diags.Add("registry.table", src.File, line, "a registry row has three cells: File, Realizes, Why")
			return
		}
		file := strings.Trim(doc.InlineText(cells[0]), "` ")
		if !ValidFile(file) {
			a.diags.Add("registry.table", src.File, line, "%q is not a repository-relative file path", file)
			return
		}
		entry := &Entry{File: file, Package: src.Package, PageFile: src.File, Line: line, Reason: doc.InlineText(cells[2])}
		for _, raw := range doc.LinksIn(cells[1]) {
			if id, ok := a.resolve(src, raw, entry); ok {
				entry.Realizes = append(entry.Realizes, id)
			}
		}
		page.Entries = append(page.Entries, entry)
	})
}

// resolve resolves one registry link to a model Definition. A link to a
// Definition that does not exist is stale, not an error.
func (a *accountant) resolve(src PageFile, raw model.RawLink, entry *Entry) (string, bool) {
	target := strings.TrimSuffix(raw.Target, ".md")
	d, ok := a.in.Model.Definitions[target]
	switch {
	case ok && d.InModel():
		return target, true
	case ok || a.in.Model.Packages[target] != nil:
		a.diags.Add("registry.link", src.File, raw.Line, "[[%s]] is not a Definition of the model; a registry entry links the Definitions its file realizes", raw.Target)
	case !validLinkPath(target):
		a.diags.Add("registry.link", src.File, raw.Line, "[[%s]] is not a model path such as members/member", raw.Target)
	case a.reg.Pages[src.Package].Orphaned:
		// The whole page is stale; its links need no item of their own.
	default:
		a.reg.Stale = append(a.reg.Stale, Stale{Code: "stale.definition", Package: src.Package, PageFile: src.File, Line: raw.Line, File: entry.File, Target: target})
	}
	return "", false
}

func validLinkPath(p string) bool {
	for _, s := range strings.Split(p, "/") {
		if !model.ValidSegment(s) {
			return false
		}
	}
	return true
}

func (a *accountant) parseIgnore(f model.File) []Pattern {
	doc, err := model.ParsePage(f.Data)
	if err != nil {
		a.diags.Add("ignore.syntax", f.Path, doc.FrontLine, "%v", err)
		return nil
	}
	if len(doc.Front) > 0 {
		a.diags.Add("ignore.syntax", f.Path, doc.FrontLine, "the ignore page has no front matter")
	}
	var patterns []Pattern
	forTables(doc, "path", func(cells []ast.Node, line int) {
		text := strings.Trim(doc.InlineText(cells[0]), "` ")
		if len(cells) != 2 || strings.TrimSpace(doc.InlineText(cells[1])) == "" {
			a.diags.Add("ignore.syntax", f.Path, line, "an ignore row has two cells: Path and Why, with a reason")
			return
		}
		if err := ValidatePattern(text); err != nil {
			a.diags.Add("ignore.syntax", f.Path, line, "%v", err)
			return
		}
		if strings.HasPrefix(text, a.in.OwnDir+"/") {
			a.diags.Add("ignore.syntax", f.Path, line, "%s is Markitect's own folder and accounted automatically", a.in.OwnDir)
			return
		}
		patterns = append(patterns, Pattern{Text: text, Line: line})
	})
	return patterns
}

// forTables calls row for every body row of every table whose first header
// cell is first (ignoring case).
func forTables(doc model.Page, first string, row func(cells []ast.Node, line int)) {
	_ = ast.Walk(doc.Root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		table, ok := n.(*east.Table)
		if !entering || !ok {
			return ast.WalkContinue, nil
		}
		header, ok := table.FirstChild().(*east.TableHeader)
		if !ok || header.FirstChild() == nil || !strings.EqualFold(doc.InlineText(header.FirstChild()), first) {
			return ast.WalkSkipChildren, nil
		}
		for r := header.NextSibling(); r != nil; r = r.NextSibling() {
			var cells []ast.Node
			line := 0
			for c := r.FirstChild(); c != nil; c = c.NextSibling() {
				cells = append(cells, c)
				if line == 0 && c.Lines().Len() > 0 {
					line = doc.Line(c.Lines().At(0).Start)
				}
			}
			if line == 0 {
				line = doc.Line(0)
			}
			row(cells, line)
		}
		return ast.WalkSkipChildren, nil
	})
}

// checkPatterns reports a pattern claimed by two Packages, or twice by one.
func (a *accountant) checkPatterns() {
	claimed := map[string]*Page{}
	for _, pkg := range a.sortedLivePages() {
		page := a.reg.Pages[pkg]
		for _, p := range page.Patterns {
			if other := claimed[p.Text]; other != nil {
				a.diags.Add("registry.duplicate_pattern", page.File, p.Line, "%s is already claimed by %s on %s; every pattern has one owner", p.Text, model.PackageName(other.Package), other.File)
				continue
			}
			claimed[p.Text] = page
		}
	}
}

func (a *accountant) sortedLivePages() []string {
	var out []string
	for pkg, page := range a.reg.Pages {
		if !page.Orphaned {
			out = append(out, pkg)
		}
	}
	sort.Strings(out)
	return out
}

// owner returns the Package whose most specific pattern matches the file, or
// Root. Equally specific patterns of two Packages are an error.
func (a *accountant) owner(file string) string {
	best, bestScore := model.RootPackage, -1
	var tied []string
	for _, pkg := range a.sortedLivePages() {
		page := a.reg.Pages[pkg]
		for _, p := range page.Patterns {
			if !p.Match(file) {
				continue
			}
			switch s := p.Specificity(); {
			case s > bestScore:
				best, bestScore, tied = pkg, s, nil
			case s == bestScore && pkg != best:
				tied = append(tied, pkg)
			}
		}
	}
	for _, other := range tied {
		key := [2]string{best, other}
		if a.ambiguous == nil {
			a.ambiguous = map[[2]string]bool{}
		}
		if !a.ambiguous[key] {
			a.ambiguous[key] = true
			a.diags.Add("registry.ambiguous_owner", file, 0, "equally specific patterns of %s and %s match this file; make one of them more specific", model.PackageName(best), model.PackageName(other))
		}
	}
	return best
}

// checkEntries checks every registry row against the repository and the
// owners.
func (a *accountant) checkEntries(visible map[string]bool) {
	a.staleOnly = map[string]bool{}
	var orphaned []*Entry
	for _, pkg := range a.sortedAllPages() {
		page := a.reg.Pages[pkg]
		if page.Orphaned {
			a.reg.Stale = append(a.reg.Stale, Stale{Code: "stale.page", Package: model.RootPackage, PageFile: page.File})
		}
		for _, e := range page.Entries {
			switch {
			case strings.HasPrefix(e.File, a.in.OwnDir+"/"):
				a.diags.Add("registry.own_file", e.PageFile, e.Line, "%s lies in %s, which Markitect accounts for itself", e.File, a.in.OwnDir)
			case !visible[e.File] && a.in.GitIgnored != nil && a.in.GitIgnored(e.File):
				a.diags.Add("registry.git_ignored", e.PageFile, e.Line, "%s is ignored by Git, so agents never see it; remove the entry or stop ignoring the file", e.File)
			case !visible[e.File]:
				a.reg.Stale = append(a.reg.Stale, Stale{Code: "stale.file", Package: e.Package, PageFile: e.PageFile, Line: e.Line, File: e.File})
			case a.reg.Ignored[e.File]:
				a.diags.Add("registry.ignored_file", e.PageFile, e.Line, "%s is also on the ignore page; a file is either registered or ignored", e.File)
			case page.Orphaned:
				orphaned = append(orphaned, e)
			case a.reg.Entries[e.File] != nil:
				first := a.reg.Entries[e.File]
				a.diags.Add("registry.duplicate_entry", e.PageFile, e.Line, "%s is already listed on %s line %d; a file has one entry", e.File, first.PageFile, first.Line)
			case a.reg.Owner[e.File] != e.Package:
				a.diags.Add("registry.wrong_owner", e.PageFile, e.Line, "%s is owned by %s, so it belongs on %s", e.File, model.PackageName(a.reg.Owner[e.File]), a.pagePath(a.reg.Owner[e.File]))
			default:
				a.reg.Entries[e.File] = e
			}
		}
	}
	// A file still explained only by the page of a removed Package is stale for
	// its current owner, who must register or remove it.
	for _, e := range orphaned {
		if a.reg.Entries[e.File] == nil {
			a.staleOnly[e.File] = true
			a.reg.Stale = append(a.reg.Stale, Stale{Code: "stale.page", Package: a.reg.Owner[e.File], PageFile: e.PageFile, Line: e.Line, File: e.File})
		}
	}
}

func (a *accountant) sortedAllPages() []string {
	var out []string
	for pkg := range a.reg.Pages {
		out = append(out, pkg)
	}
	sort.Strings(out)
	return out
}
