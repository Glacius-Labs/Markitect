package cli

import (
	"io"
	"sort"
	"strings"
)

// showDocument is the JSON document of show. Later steps add the document's
// artifacts, counter-checks and verdict beside these fields.
type showDocument struct {
	header
	Path  string `json:"path"`
	Title string `json:"title"`
	// Type is the path of the type, "" without one; TypeState is "none",
	// "invalid" for a type line that is not one link, "missing" for a type
	// that is no document, or "ok".
	Type       string        `json:"type"`
	TypeState  string        `json:"type_state"`
	Sections   []showSection `json:"sections"`
	LinksTo    []showLink    `json:"links_to"`
	LinkedFrom []showLink    `json:"linked_from"`
	// Documents are the documents whose type this document is.
	Documents []string `json:"documents"`
}

type showSection struct {
	Heading string `json:"heading"`
}

type showLink struct {
	Path    string `json:"path"`
	Section string `json:"section,omitempty"`
	Missing bool   `json:"missing,omitempty"`
}

func show(o options, stdout io.Writer) (int, error) {
	spec, _, err := load(o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	d, err := document(spec, o.positional[1])
	if err != nil {
		return ExitFailed, err
	}
	doc := showDocument{header: newHeader("show"), Path: d.Path, Title: d.Title, Type: d.TypePath,
		Sections: []showSection{}, LinksTo: []showLink{}, LinkedFrom: []showLink{}, Documents: []string{}}
	switch _, ok := spec.Document(d.TypePath); {
	case d.TypeLine == 0:
		doc.TypeState = "none"
	case !d.HasType():
		doc.TypeState = "invalid"
	case !ok:
		doc.TypeState = "missing"
	default:
		doc.TypeState = "ok"
	}
	for _, s := range d.Sections {
		doc.Sections = append(doc.Sections, showSection{Heading: s.Heading})
	}
	seen := map[string]bool{}
	for _, l := range d.Links {
		if !seen[l.Target] {
			seen[l.Target] = true
			_, ok := spec.Document(l.Target)
			doc.LinksTo = append(doc.LinksTo, showLink{Path: l.Target, Missing: !ok})
		}
	}
	for _, other := range spec.Documents {
		seen := map[string]bool{}
		for _, l := range other.Links {
			if l.Target == d.Path && !seen[l.Section] {
				seen[l.Section] = true
				doc.LinkedFrom = append(doc.LinkedFrom, showLink{Path: other.Path, Section: l.Section})
			}
		}
		if other.HasType() && other.TypePath == d.Path {
			doc.Documents = append(doc.Documents, other.Path)
		}
	}
	sortLinks(doc.LinksTo)
	sortLinks(doc.LinkedFrom)
	return output(o, stdout, doc, showText(doc))
}

// sortLinks sorts links by their text without regard to case, then as written.
func sortLinks(links []showLink) {
	key := func(l showLink) string {
		if l.Section != "" {
			return l.Path + " (" + l.Section + ")"
		}
		return l.Path
	}
	sort.SliceStable(links, func(i, j int) bool {
		a, b := key(links[i]), key(links[j])
		if la, lb := strings.ToLower(a), strings.ToLower(b); la != lb {
			return la < lb
		}
		return a < b
	})
}

func showText(doc showDocument) string {
	var b strings.Builder
	title := doc.Title
	if title == "" {
		title = "(none)"
	}
	kind := doc.Type
	switch doc.TypeState {
	case "none":
		kind = "(none)"
	case "invalid":
		kind = "(invalid line)"
	case "missing":
		kind += " (missing)"
	}
	b.WriteString(doc.Path + "\nTitle: " + title + "\nType: " + kind + "\n")
	list := func(label string, items []string, always bool) {
		if len(items) == 0 {
			if always {
				b.WriteString(label + " (none)\n")
			}
			return
		}
		b.WriteString(label + "\n")
		for _, i := range items {
			b.WriteString("  " + i + "\n")
		}
	}
	var sections, to, from []string
	for _, s := range doc.Sections {
		sections = append(sections, s.Heading)
	}
	for _, l := range doc.LinksTo {
		if l.Missing {
			to = append(to, l.Path+" (missing)")
		} else {
			to = append(to, l.Path)
		}
	}
	for _, l := range doc.LinkedFrom {
		if l.Section != "" {
			from = append(from, l.Path+" ("+l.Section+")")
		} else {
			from = append(from, l.Path)
		}
	}
	list("Sections:", sections, true)
	list("Links to:", to, true)
	list("Linked from:", from, true)
	list("Documents:", doc.Documents, false)
	return b.String()
}

// tableDocument is the JSON document of table.
type tableDocument struct {
	header
	Type    string     `json:"type"`
	Columns []string   `json:"columns"`
	Rows    []tableRow `json:"rows"`
}

type tableRow struct {
	Path  string   `json:"path"`
	Title string   `json:"title"`
	Cells []string `json:"cells"`
}

func table(o options, stdout io.Writer) (int, error) {
	spec, _, err := load(o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	t, err := document(spec, o.positional[1])
	if err != nil {
		return ExitFailed, err
	}
	doc := tableDocument{header: newHeader("table"), Type: t.Path, Columns: []string{}, Rows: []tableRow{}}
	for _, s := range t.Declared() {
		doc.Columns = append(doc.Columns, s.Name)
	}
	for _, d := range spec.Documents {
		if !d.HasType() || d.TypePath != t.Path {
			continue
		}
		row := tableRow{Path: d.Path, Title: d.Title, Cells: []string{}}
		for _, name := range doc.Columns {
			s, _ := d.Section(name)
			row.Cells = append(row.Cells, s.First)
		}
		doc.Rows = append(doc.Rows, row)
	}
	return output(o, stdout, doc, tableText(doc))
}

func tableText(doc tableDocument) string {
	var b strings.Builder
	line := func(cells []string) { b.WriteString("| " + strings.Join(cells, " | ") + " |\n") }
	line(append([]string{"Title"}, doc.Columns...))
	sep := make([]string, len(doc.Columns)+1)
	for i := range sep {
		sep[i] = "---"
	}
	line(sep)
	for _, r := range doc.Rows {
		title := r.Title
		if title == "" {
			title = r.Path
		}
		cells := []string{title}
		for _, c := range r.Cells {
			cells = append(cells, strings.ReplaceAll(c, "|", `\|`))
		}
		line(cells)
	}
	return b.String()
}
