package cli

import (
	"io"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
	"github.com/Glacius-Labs/Markitect/internal/state"
)

// showDocument is the JSON document of show.
type showDocument struct {
	header
	Path string `json:"path"`
	// Fingerprint is the document's fingerprint as status computes it, so
	// that a mark can name any document.
	Fingerprint string `json:"fingerprint"`
	Title       string `json:"title"`
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
	// The row of the state status takes: Verdict is "" without one, and
	// Conflict says that rows of it conflict.
	Verdict       string   `json:"verdict"`
	Conflict      bool     `json:"conflict,omitempty"`
	Artifacts     []string `json:"artifacts"`
	CounterChecks []string `json:"counter_checks"`
	Evidence      []string `json:"evidence"`
	Reason        string   `json:"reason"`
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
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	spec := w.Spec
	d, err := document(spec, o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	s, _, err := evaluate(w)
	if err != nil {
		return ExitFailed, err
	}
	doc := showDocument{header: newHeader("show"), Path: d.Path, Title: d.Title, Type: d.TypePath,
		Sections: []showSection{}, LinksTo: []showLink{}, LinkedFrom: []showLink{}, Documents: []string{},
		Artifacts: []string{}, CounterChecks: []string{}, Evidence: []string{}}
	if doc.Fingerprint, _, err = s.Fingerprint(d.Path); err != nil {
		return ExitFailed, err
	}
	if row, conflict, ok := s.Row(d.Path); ok {
		doc.Verdict, doc.Conflict, doc.Reason = row.Verdict, conflict, row.Reason
		doc.Artifacts, doc.CounterChecks = append(doc.Artifacts, row.Files()...), append(doc.CounterChecks, row.Checks()...)
		doc.Evidence = append(doc.Evidence, row.Evidence...)
	}
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
	b.WriteString(doc.Path + "@" + doc.Fingerprint + "\nTitle: " + title + "\nType: " + kind + "\n")
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
	verdict := doc.Verdict
	switch {
	case verdict == "":
		verdict = "(no row)"
	case doc.Conflict:
		verdict += " (rows conflict)"
	}
	b.WriteString("Verdict: " + verdict + "\n")
	list("Artifacts:", doc.Artifacts, true)
	list("Counter-checks:", doc.CounterChecks, true)
	list("Evidence:", doc.Evidence, true)
	if doc.Reason != "" {
		b.WriteString("Reason: " + doc.Reason + "\n")
	}
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
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	spec := w.Spec
	if len(o.positional) == 0 {
		s, _, err := evaluate(w)
		if err != nil {
			return ExitFailed, err
		}
		doc := stateTable(spec, s)
		return output(o, stdout, doc, stateTableText(doc))
	}
	t, err := document(spec, o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	doc := tableOf(spec, t)
	return output(o, stdout, doc, tableText(doc))
}

// tableOf is every document of the type t as a row.
func tableOf(spec *lang.Spec, t *lang.Document) tableDocument {
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
	return doc
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

// stateTable is every document of the spec with its verdict and artifacts:
// the state as a table. The type is "" and the columns are Verdict and
// Artifacts.
func stateTable(spec *lang.Spec, s *state.Status) tableDocument {
	doc := tableDocument{header: newHeader("table"), Columns: []string{"Verdict", "Artifacts"}, Rows: []tableRow{}}
	for _, d := range spec.Documents {
		verdict, artifacts := "", ""
		if row, conflict, ok := s.Row(d.Path); ok {
			verdict, artifacts = row.Verdict, strings.Join(row.Files(), ", ")
			if conflict {
				verdict += " (rows conflict)"
			}
		}
		doc.Rows = append(doc.Rows, tableRow{Path: d.Path, Title: d.Title, Cells: []string{verdict, artifacts}})
	}
	return doc
}

func stateTableText(doc tableDocument) string {
	var b strings.Builder
	b.WriteString("| Document | Verdict | Artifacts |\n| --- | --- | --- |\n")
	for _, r := range doc.Rows {
		cells := []string{r.Path}
		for _, c := range r.Cells {
			cells = append(cells, strings.ReplaceAll(c, "|", `\|`))
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}
	return b.String()
}
