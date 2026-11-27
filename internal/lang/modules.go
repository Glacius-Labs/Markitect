package lang

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// A module is a folder of the model that holds at least one document directly;
// the model folder itself is one when it holds a document directly. A folder
// that holds only folders, or no document at all, is no module and is
// transparent: it counts as no level between two modules. The manifest of a
// module is its README.md; the section Contents of it lists what the module
// shows to the modules outside it: by links, and by code spans, which are
// patterns over the file names of the documents directly in its folder. Its
// sections Realized in list path patterns over the files of the project that
// build the module.

const (
	readmeName      = "README.md"
	contentsSection = "Contents"
	// realizedSection is the section of a README that says where in the project
	// the module is built.
	realizedSection = "Realized in"
	// modelFolder is the name of the model folder as a module folder.
	modelFolder = "."
)

// isReadme reports whether the path names a README, a document called
// README.md exactly.
func isReadme(p string) bool { return path.Base(p) == readmeName }

// readmeOf returns the path of the README of the module folder dir.
func readmeOf(dir string) string {
	if dir == modelFolder {
		return readmeName
	}
	return dir + "/" + readmeName
}

// readWithoutHead reads a README without its type line or extends line: it has
// neither, so it is never an object and never a type. Every such line of the
// head block is kept in ReadmeHeads for check, and is no stray line.
func (d *Document) readWithoutHead() {
	if d.TypeLine != 0 {
		d.ReadmeHeads = append(d.ReadmeHeads, ReadmeHead{d.TypeLine, "type"})
	}
	if d.ExtendsLine != 0 {
		d.ReadmeHeads = append(d.ReadmeHeads, ReadmeHead{d.ExtendsLine, "extends"})
	}
	d.TypeLine, d.TypeRaw, d.TypePath = 0, "", ""
	d.ExtendsLine, d.ExtendsRaw, d.ExtendsPath = 0, "", ""
	keep := func(lines []int, kind string) []int {
		var rest []int
		for _, line := range lines {
			if d.inHead(line) {
				d.ReadmeHeads = append(d.ReadmeHeads, ReadmeHead{line, kind})
			} else {
				rest = append(rest, line)
			}
		}
		return rest
	}
	d.StrayTypeLines = keep(d.StrayTypeLines, "type")
	d.StrayExtendsLines = keep(d.StrayExtendsLines, "extends")
	sort.Slice(d.ReadmeHeads, func(i, j int) bool { return d.ReadmeHeads[i].Line < d.ReadmeHeads[j].Line })
}

// readHeadReadme reads a type line or an extends line that names a README as
// naming no type; the path stays in HeadReadme.
func (d *Document) readHeadReadme(s *Model) {
	switch {
	case d.HasType() && isReadmeDocument(s, d.TypePath):
		d.HeadReadme = d.TypePath
		d.TypeRaw, d.TypePath = "", ""
	case d.HasExtends() && isReadmeDocument(s, d.ExtendsPath):
		d.HeadReadme = d.ExtendsPath
		d.ExtendsRaw, d.ExtendsPath = "", ""
	}
}

func isReadmeDocument(s *Model, p string) bool {
	_, ok := s.byPath[p]
	return ok && isReadme(p)
}

// moduleChain returns the module folders that contain a document of the
// folder dir, the outermost first and dir itself last: dir holds the document,
// so it is a module.
func (s *Model) moduleChain(dir string) []string {
	var chain []string
	for cur := dir; ; cur = path.Dir(cur) {
		if s.modules[cur] {
			chain = append(chain, cur)
		}
		if cur == modelFolder || cur == "/" {
			break
		}
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}

// shownByModule returns, for every module folder, the documents its README
// shows to the modules outside it: the README itself, and what the README
// links in its section Contents when that is a document directly in the folder
// of the module or the README of the nearest module below it.
func (s *Model) shownByModule() map[string]map[string]bool {
	shown := map[string]map[string]bool{}
	for dir := range s.modules {
		readme, ok := s.byPath[readmeOf(dir)]
		if !ok {
			continue
		}
		set := map[string]bool{readme.Path: true}
		shown[dir] = set
		for _, pattern := range contentsPatterns(readme) {
			for _, d := range s.Documents {
				if path.Dir(d.Path) != dir {
					continue
				}
				if ok, _ := path.Match(pattern.Text, path.Base(d.Path)); ok {
					set[d.Path] = true
				}
			}
		}
		for _, l := range readme.Links {
			if !strings.EqualFold(l.Section, contentsSection) {
				continue
			}
			if _, ok := s.byPath[l.Target]; !ok {
				continue
			}
			target := path.Dir(l.Target)
			switch {
			case target == dir:
				set[l.Target] = true
			case isReadme(l.Target) && s.modules[target]:
				// The nearest module below dir: the module before the target's
				// own in its chain.
				if chain := s.moduleChain(target); len(chain) > 1 && chain[len(chain)-2] == dir {
					set[l.Target] = true
				}
			}
		}
	}
	return shown
}

// contentsPatterns returns the code spans of the sections Contents of a README
// that are patterns: wildcards as path.Match reads them, over the file names of
// the documents directly in the folder. A code span with a "/" is plain code,
// such as a path or an address, and one that path.Match calls malformed shows
// nothing.
func contentsPatterns(readme *Document) []CodeSpan {
	var out []CodeSpan
	for _, c := range readme.CodeSpans {
		if strings.EqualFold(c.Section, contentsSection) && !strings.Contains(c.Text, "/") && validPattern(c.Text) {
			out = append(out, c)
		}
	}
	return out
}

// validPattern reports whether path.Match reads the pattern, whatever name it
// is matched against.
func validPattern(pattern string) bool {
	_, err := path.Match(pattern, "")
	return err == nil
}

// checkPatterns reports the code spans without a "/" in the section Contents of
// a README that are no valid pattern.
func checkPatterns(d *Document) []Diagnostic {
	if !isReadme(d.Path) {
		return nil
	}
	var out []Diagnostic
	for _, c := range d.CodeSpans {
		if strings.EqualFold(c.Section, contentsSection) && !strings.Contains(c.Text, "/") && !validPattern(c.Text) {
			out = append(out, Diagnostic{Path: d.Path, Line: c.Line, Code: CodeReadmePattern,
				Message: fmt.Sprintf("bad pattern \"%s\": a pattern uses the wildcards *, ? and [a-z]", c.Text)})
		}
	}
	return out
}

// SortModules sorts module folders by path, the model folder first.
func SortModules(modules []string) {
	sort.Slice(modules, func(i, j int) bool { return moduleLess(modules[i], modules[j]) })
}

// checkModules reports every module without a README.md, at the path the
// README would have, line 1.
func (s *Model) checkModules() []Diagnostic {
	var dirs []string
	for dir := range s.modules {
		if _, ok := s.byPath[readmeOf(dir)]; !ok {
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)
	var out []Diagnostic
	for _, dir := range dirs {
		message := fmt.Sprintf("the folder %s holds documents and has no README.md", dir)
		if dir == modelFolder {
			message = "the model folder holds documents and has no README.md"
		}
		out = append(out, Diagnostic{Path: readmeOf(dir), Line: 1, Code: CodeModuleReadme, Message: message})
	}
	return out
}

// checkReadmeType reports the type line or extends line of a README, and a
// type line or extends line that names a README.
func (s *Model) checkReadmeType(d *Document) []Diagnostic {
	switch {
	case len(d.ReadmeHeads) > 0:
		var out []Diagnostic
		for _, h := range d.ReadmeHeads {
			out = append(out, Diagnostic{Path: d.Path, Line: h.Line, Code: CodeReadmeType, Message: fmt.Sprintf("a README has no %s line", h.Kind)})
		}
		return out
	case d.HeadReadme != "":
		line := max(d.TypeLine, d.ExtendsLine)
		return []Diagnostic{{Path: d.Path, Line: line, Code: CodeReadmeType, Message: fmt.Sprintf("\"%s\" is a README, which is no type", d.HeadReadme)}}
	}
	return nil
}

// checkInternal reports every link of d, in text, on its type line, on its
// extends line or on a property type line, that reaches what the README of a module does not show to the
// documents outside it. A document inside a module may link to everything of
// that module and of the modules above it.
//
// Of the modules that contain the target, those that contain d too are left
// out; the others, M1 the outermost to Mk the target's own, must show the way:
// each Mi but the last the README of M(i+1), the last the target unless that is
// its README. The first step that is not shown makes the link an error.
func (s *Model) checkInternal(d *Document, shown map[string]map[string]bool) []Diagnostic {
	var out []Diagnostic
	seen := map[string]bool{}
	check := func(target string, line int) {
		if seen[target] {
			return
		}
		seen[target] = true
		if _, ok := s.byPath[target]; !ok {
			return
		}
		chain := s.moduleChain(path.Dir(target))
		inside := s.moduleChain(path.Dir(d.Path))
		for len(chain) > 0 && len(inside) > 0 && chain[0] == inside[0] {
			chain, inside = chain[1:], inside[1:]
		}
		for i, dir := range chain {
			need, what := target, "it"
			if i < len(chain)-1 {
				need = readmeOf(chain[i+1])
				what = need
			} else if target == readmeOf(dir) {
				break
			}
			if shown[dir][need] {
				continue
			}
			if need == target {
				what = "it"
			}
			// The cause: a module on the way without a README, or a README that
			// does not list what is behind it. The module that is checked is
			// never the model folder, which holds the linking document.
			reason := fmt.Sprintf("%s does not list %s under %s", readmeOf(dir), what, contentsSection)
			onTheWay := []string{dir}
			if i < len(chain)-1 {
				onTheWay = append(onTheWay, chain[i+1])
			}
			for _, m := range onTheWay {
				if _, ok := s.byPath[readmeOf(m)]; !ok {
					reason = m + " has no README.md"
					break
				}
			}
			out = append(out, Diagnostic{Path: d.Path, Line: line, Code: CodeLinkInternal,
				Message: fmt.Sprintf("\"%s\" is internal to %s: %s", target, dir, reason)})
			return
		}
	}
	switch {
	case d.HasType():
		check(d.TypePath, d.TypeLine)
	case d.HasExtends():
		check(d.ExtendsPath, d.ExtendsLine)
	case d.HeadReadme != "":
		check(d.HeadReadme, max(d.TypeLine, d.ExtendsLine))
	}
	// The links of the text and the links of the property type lines, in the
	// order of their lines.
	type step struct {
		target string
		line   int
	}
	var steps []step
	for _, l := range d.Links {
		steps = append(steps, step{l.Target, l.Line})
	}
	if d.holdsProperties() {
		for _, sec := range d.Sections {
			for _, r := range sec.Property.Refs {
				steps = append(steps, step{r.Target, sec.Property.Line})
			}
		}
	}
	if len(steps) > len(d.Links) {
		sort.SliceStable(steps, func(i, j int) bool { return steps[i].line < steps[j].line })
	}
	for _, st := range steps {
		check(st.target, st.line)
	}
	return out
}
