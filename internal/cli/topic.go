package cli

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
	"github.com/Glacius-Labs/Markitect/internal/library"
	"github.com/Glacius-Labs/Markitect/internal/signals"
)

// shownPerSignal is how many findings of each signal the text of the whole
// spec shows; --all shows every one, and --json always does.
const shownPerSignal = 10

// references reads the reference values of every package of the library.
func references() (signals.References, error) {
	refs := signals.References{}
	entries, err := fs.ReadDir(library.FS(), "references")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		data, err := fs.ReadFile(library.FS(), "references/"+e.Name())
		if err != nil {
			return nil, err
		}
		if err := signals.ParseReferences(strings.TrimSuffix(e.Name(), ".md"), data, refs); err != nil {
			return nil, err
		}
	}
	return refs, nil
}

// topicDocument is the JSON document of topic.
type topicDocument struct {
	header
	Topic      string                   `json:"topic"`
	TopicKind  string                   `json:"topic_kind"`
	Documents  []string                 `json:"documents"`
	Strong     []statementJSON          `json:"strong"`
	LinkedFrom []statementJSON          `json:"linked_from"`
	Mentions   []statementJSON          `json:"mentions"`
	Signals    map[string][]findingJSON `json:"signals"`
	Typed      []string                 `json:"typed_documents,omitempty"`
}

type statementJSON struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Section string `json:"section"`
	Text    string `json:"text"`
}

type findingJSON struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Detail string `json:"detail"`
}

func topic(o options, stdout io.Writer) (int, error) {
	spec, _, err := load(o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	refs, err := references()
	if err != nil {
		return ExitFailed, err
	}
	a := signals.Analyze(spec, refs)
	name := ""
	if len(o.positional) > 1 {
		name = o.positional[1]
	}
	t, ok := a.Topic(name)
	if !ok {
		return ExitFailed, fmt.Errorf("unknown topic: %s", name)
	}
	if name == "" {
		t.Kind, t.Path = "spec", ""
	}
	doc := topicDocument{header: newHeader("topic"), Topic: t.Path, TopicKind: t.Kind, Documents: t.Docs,
		Strong: statementsJSON(t.Strong), LinkedFrom: statementsJSON(t.LinkedFrom), Mentions: statementsJSON(t.Mentions),
		Signals: map[string][]findingJSON{}, Typed: t.Documents}
	for _, s := range signals.Names {
		doc.Signals[s] = []findingJSON{}
		for _, f := range t.Findings[s] {
			doc.Signals[s] = append(doc.Signals[s], findingJSON{File: f.Path, Line: f.Line, Detail: f.Detail})
		}
	}
	return output(o, stdout, doc, topicText(spec, t, o.all))
}

func statementsJSON(list []signals.Statement) []statementJSON {
	out := []statementJSON{}
	for _, st := range list {
		out = append(out, statementJSON{File: st.Doc, Line: st.Line, Section: st.Section, Text: st.Text})
	}
	return out
}

func topicText(spec *lang.Spec, t *signals.Topic, all bool) string {
	var b strings.Builder
	statements := func(heading string, list []signals.Statement, withSection bool) {
		b.WriteString("\n## " + heading + "\n\n")
		if len(list) == 0 {
			b.WriteString("(none)\n")
		}
		for _, st := range list {
			where := fmt.Sprintf("%s:%d", st.Doc, st.Line)
			if withSection && st.Section != "" {
				where += " (" + st.Section + ")"
			}
			b.WriteString(where + ": " + st.Text + "\n")
		}
	}
	if t.Kind != "spec" {
		fmt.Fprintf(&b, "# Topic: %s (%s)\n", t.Path, t.Kind)
		statements("Strong statements about it", t.Strong, false)
		switch t.Kind {
		case "type":
			d, _ := spec.Document(t.Path)
			b.WriteString("\n## Definition\n\n" + text(d) + "\n## Documents\n\n" + tableText(tableOf(spec, d)))
		case "document":
			d, _ := spec.Document(t.Path)
			b.WriteString("\n## Text\n\n" + text(d))
		default:
			b.WriteString("\n## Documents\n")
			for _, p := range t.Docs {
				d, _ := spec.Document(p)
				b.WriteString("\n### " + p + "\n\n" + text(d))
			}
		}
		statements("Linked from", t.LinkedFrom, true)
		statements("Mentioned without a link", t.Mentions, false)
		b.WriteString("\n## Signals\n")
	} else {
		fmt.Fprintf(&b, "# Signals of the spec\n")
	}
	for _, s := range signals.Names {
		list := t.Findings[s]
		fmt.Fprintf(&b, "\n### %s (%d)\n\n", s, len(list))
		if len(list) == 0 {
			b.WriteString("(none)\n")
		}
		for i, f := range list {
			if t.Kind == "spec" && !all && i == shownPerSignal {
				fmt.Fprintf(&b, "… %d more; --all shows them\n", len(list)-i)
				break
			}
			b.WriteString(f.String() + "\n")
		}
	}
	return b.String()
}

// text returns a document's text as written, ending with a line break.
func text(d *lang.Document) string {
	s := strings.ReplaceAll(strings.TrimPrefix(string(d.Source), "\xef\xbb\xbf"), "\r\n", "\n")
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s
}

// questionsDocument is the JSON document of questions.
type questionsDocument struct {
	header
	Packages []packageQuestions `json:"packages"`
}

type packageQuestions struct {
	Package   string   `json:"package"`
	Questions []string `json:"questions"`
}

func questions(o options, stdout io.Writer) (int, error) {
	_, fsys, err := load(o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	doc := questionsDocument{header: newHeader("questions"), Packages: []packageQuestions{}}
	entries, _ := fs.ReadDir(fsys, "packages")
	var b strings.Builder
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := fs.ReadFile(library.FS(), path.Join("questions", e.Name()+".md"))
		if err != nil {
			continue
		}
		pq := packageQuestions{Package: e.Name(), Questions: []string{}}
		for _, line := range strings.Split(string(data), "\n") {
			if q, ok := strings.CutPrefix(strings.TrimSpace(line), "- "); ok {
				pq.Questions = append(pq.Questions, q)
			}
		}
		doc.Packages = append(doc.Packages, pq)
		fmt.Fprintf(&b, "## %s\n\n", pq.Package)
		for i, q := range pq.Questions {
			fmt.Fprintf(&b, "%d. %s\n", i+1, q)
		}
		b.WriteString("\n")
	}
	if len(doc.Packages) == 0 {
		b.WriteString("The spec has no package of the library, so there are no questions.\n")
	}
	return output(o, stdout, doc, strings.TrimSuffix(b.String(), "\n\n")+"\n")
}
