package lang

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// A module is a folder of the spec that holds at least one document directly;
// the spec folder itself is one when it holds a document directly. A folder
// that holds only folders, or no document at all, is no module and is
// transparent: it counts as no level between two modules. The manifest of a
// module is its README.md; the section Contents of it lists what the module
// shows to the modules outside it.

const (
	readmeName      = "README.md"
	contentsSection = "Contents"
	// specFolder is the name of the spec folder as a module folder.
	specFolder = "."
)

// isReadme reports whether the path names a README, a document called
// README.md exactly.
func isReadme(p string) bool { return path.Base(p) == readmeName }

// readmeOf returns the path of the README of the module folder dir.
func readmeOf(dir string) string {
	if dir == specFolder {
		return readmeName
	}
	return dir + "/" + readmeName
}

// readWithoutHead reads a README without its type line or extends line: it has
// neither, so it is never an object and never a type. The line is kept in
// ReadmeHeadLine for check.
func (d *Document) readWithoutHead() {
	switch {
	case d.TypeLine != 0:
		d.ReadmeHeadLine, d.ReadmeHeadKind = d.TypeLine, "type"
	case d.ExtendsLine != 0:
		d.ReadmeHeadLine, d.ReadmeHeadKind = d.ExtendsLine, "extends"
	}
	d.TypeLine, d.TypeRaw, d.TypePath = 0, "", ""
	d.ExtendsLine, d.ExtendsRaw, d.ExtendsPath = 0, "", ""
}

// readHeadReadme reads a type line or an extends line that names a README as
// naming no type; the path stays in HeadReadme.
func (d *Document) readHeadReadme(s *Spec) {
	switch {
	case d.HasType() && isReadmeDocument(s, d.TypePath):
		d.HeadReadme = d.TypePath
		d.TypeRaw, d.TypePath = "", ""
	case d.HasExtends() && isReadmeDocument(s, d.ExtendsPath):
		d.HeadReadme = d.ExtendsPath
		d.ExtendsRaw, d.ExtendsPath = "", ""
	}
}

func isReadmeDocument(s *Spec, p string) bool {
	_, ok := s.byPath[p]
	return ok && isReadme(p)
}

// moduleChain returns the module folders that contain a document of the
// folder dir, the outermost first and dir itself last: dir holds the document,
// so it is a module.
func (s *Spec) moduleChain(dir string) []string {
	var chain []string
	for cur := dir; ; cur = path.Dir(cur) {
		if s.modules[cur] {
			chain = append(chain, cur)
		}
		if cur == specFolder || cur == "/" {
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
func (s *Spec) shownByModule() map[string]map[string]bool {
	shown := map[string]map[string]bool{}
	for dir := range s.modules {
		readme, ok := s.byPath[readmeOf(dir)]
		if !ok {
			continue
		}
		set := map[string]bool{readme.Path: true}
		shown[dir] = set
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

// checkModules reports every module without a README.md, at the path the
// README would have, line 1.
func (s *Spec) checkModules() []Diagnostic {
	var dirs []string
	for dir := range s.modules {
		if _, ok := s.byPath[readmeOf(dir)]; !ok {
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)
	var out []Diagnostic
	for _, dir := range dirs {
		message := fmt.Sprintf("the folder \"%s\" holds documents and has no README.md", dir)
		if dir == specFolder {
			message = "the spec folder holds documents and has no README.md"
		}
		out = append(out, Diagnostic{Path: readmeOf(dir), Line: 1, Code: CodeModuleReadme, Message: message})
	}
	return out
}

// checkReadmeType reports the type line or extends line of a README, and a
// type line or extends line that names a README.
func (s *Spec) checkReadmeType(d *Document) []Diagnostic {
	switch {
	case d.ReadmeHeadLine != 0:
		return []Diagnostic{{Path: d.Path, Line: d.ReadmeHeadLine, Code: CodeReadmeType, Message: fmt.Sprintf("a README has no %s line", d.ReadmeHeadKind)}}
	case d.HeadReadme != "":
		line := max(d.TypeLine, d.ExtendsLine)
		return []Diagnostic{{Path: d.Path, Line: line, Code: CodeReadmeType, Message: fmt.Sprintf("\"%s\" is a README, which is no type", d.HeadReadme)}}
	}
	return nil
}

// moduleName names a module folder in a message.
func moduleName(dir string) string {
	if dir == specFolder {
		return "the spec folder"
	}
	return dir
}

// checkInternal reports every link of d, in text, on its type line or on its
// extends line, that reaches what the README of a module does not show to the
// documents outside it. A document inside a module may link to everything of
// that module and of the modules above it.
//
// Of the modules that contain the target, those that contain d too are left
// out; the others, M1 the outermost to Mk the target's own, must show the way:
// each Mi but the last the README of M(i+1), the last the target unless that is
// its README. The first step that is not shown makes the link an error.
func (s *Spec) checkInternal(d *Document, shown map[string]map[string]bool) []Diagnostic {
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
			reason := fmt.Sprintf("%s does not list %s under %s", readmeOf(dir), what, contentsSection)
			if _, ok := s.byPath[readmeOf(dir)]; !ok {
				reason = moduleName(dir) + " has no README.md"
			}
			out = append(out, Diagnostic{Path: d.Path, Line: line, Code: CodeLinkInternal,
				Message: fmt.Sprintf("\"%s\" is internal to %s: %s", target, moduleName(dir), reason)})
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
	for _, l := range d.Links {
		check(l.Target, l.Line)
	}
	return out
}
