package signals

import (
	"path"
	"sort"
	"strings"
)

// Topic is the topic view of a document, a type or a folder: everything that
// belongs to it.
type Topic struct {
	// Kind is "document", "type" or "folder".
	Kind string
	// Path is the document's path, or the folder's.
	Path string
	// Docs are the topic's own documents: the document, the type, or every
	// document in the folder, in path order.
	Docs []string
	// Documents are a type's documents.
	Documents []string
	// Strong are the strong statements of other documents about the topic.
	Strong []Statement
	// LinkedFrom are the statements of other documents that link to it.
	LinkedFrom []Statement
	// Mentions are the statements of other documents that name it by title
	// in a document without a link to it.
	Mentions []Statement
	// Findings are the signals' findings that concern it, by signal.
	Findings map[string][]Finding
}

// Topic gathers the topic view of name: a document path, with or without
// .md, or a folder below the spec folder. ok is false when it is neither.
func (a *Analysis) Topic(name string) (*Topic, bool) {
	name = strings.Trim(strings.ReplaceAll(name, "\\", "/"), "/")
	t := &Topic{Findings: map[string][]Finding{}}
	doc := name
	if !strings.HasSuffix(strings.ToLower(doc), ".md") {
		doc += ".md"
	}
	if d, ok := a.Spec.Document(path.Clean(doc)); ok {
		t.Kind, t.Path, t.Docs = "document", d.Path, []string{d.Path}
		for _, o := range a.Spec.Documents {
			if o.HasType() && o.TypePath == d.Path {
				t.Kind = "type"
				t.Documents = append(t.Documents, o.Path)
			}
		}
	} else {
		folder := path.Clean(name)
		if folder == "." {
			folder = ""
		}
		for _, d := range a.Spec.Documents {
			if folder == "" || strings.HasPrefix(d.Path, folder+"/") {
				t.Docs = append(t.Docs, d.Path)
			}
		}
		if folder != "" && len(t.Docs) == 0 || strings.HasPrefix(folder, "..") {
			return nil, false
		}
		t.Kind, t.Path = "folder", folder
	}
	own := map[string]bool{}
	for _, d := range t.Docs {
		own[d] = true
	}
	// A link to the README of a folder that holds a topic document counts as
	// a link to the topic.
	about := map[string]bool{}
	for _, d := range t.Docs {
		about[d] = true
		for dir := path.Dir(d); ; dir = path.Dir(dir) {
			readme := "README.md"
			if dir != "." {
				readme = dir + "/README.md"
			}
			if readme != d {
				if _, ok := a.Spec.Document(readme); ok && !own[readme] {
					about[readme] = true
				}
			}
			if dir == "." {
				break
			}
		}
	}
	for i, st := range a.Statements {
		if own[st.Doc] {
			continue
		}
		linked, named := false, false
		for _, x := range st.Targets {
			linked = linked || about[x]
		}
		for _, x := range a.Mentions[i] {
			named = named || own[x] && !a.links[st.Doc][x]
		}
		if (linked || named || anyIn(a.Mentions[i], own)) && len(st.Strong) > 0 {
			t.Strong = append(t.Strong, st)
		}
		if linked {
			t.LinkedFrom = append(t.LinkedFrom, st)
		}
		if named {
			t.Mentions = append(t.Mentions, st)
		}
	}
	for _, name := range Names {
		for _, f := range a.Findings[name] {
			if anyIn(f.Docs, own) {
				t.Findings[name] = append(t.Findings[name], f)
			}
		}
	}
	sort.Strings(t.Documents)
	return t, true
}

func anyIn(list []string, set map[string]bool) bool {
	for _, x := range list {
		if set[x] {
			return true
		}
	}
	return false
}
