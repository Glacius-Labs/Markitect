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
// model shows; --all shows every one, and --json always does.
const shownPerSignal = 10

// references reads the reference values built into the tool, one file per module of the default.
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
	Named      []statementJSON          `json:"named"`
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
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	model := w.Model
	refs, err := references()
	if err != nil {
		return ExitFailed, err
	}
	a := signals.Analyze(model, refs)
	name := ""
	if len(o.positional) > 0 {
		name = o.positional[0]
	}
	t, ok := a.Topic(name)
	if !ok {
		return ExitFailed, fmt.Errorf("unknown topic: %s", name)
	}
	if name == "" {
		t.Kind, t.Path = "model", ""
	}
	doc := topicDocument{header: newHeader("topic"), Topic: t.Path, TopicKind: t.Kind, Documents: t.Docs,
		Strong: statementsJSON(t.Strong), LinkedFrom: statementsJSON(t.LinkedFrom), Named: statementsJSON(t.Named),
		Signals: map[string][]findingJSON{}, Typed: t.Documents}
	for _, s := range signals.Names {
		doc.Signals[s] = []findingJSON{}
		for _, f := range t.Findings[s] {
			doc.Signals[s] = append(doc.Signals[s], findingJSON{File: f.Path, Line: f.Line, Detail: f.Detail})
		}
	}
	return output(o, stdout, doc, topicText(model, t, o.all))
}

func statementsJSON(list []signals.Statement) []statementJSON {
	out := []statementJSON{}
	for _, st := range list {
		out = append(out, statementJSON{File: st.Doc, Line: st.Line, Section: st.Section, Text: st.Text})
	}
	return out
}

func topicText(model *lang.Model, t *signals.Topic, all bool) string {
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
	if t.Kind != "model" {
		fmt.Fprintf(&b, "# Topic: %s (%s)\n", t.Path, t.Kind)
		statements("Strong statements about it", t.Strong, false)
		switch t.Kind {
		case "type":
			d, _ := model.Document(t.Path)
			b.WriteString("\n## Definition\n\n" + text(d) + "\n## Documents\n\n" + tableText(tableOf(model, d)))
		case "document":
			d, _ := model.Document(t.Path)
			b.WriteString("\n## Text\n\n" + text(d))
		default:
			b.WriteString("\n## Documents\n")
			for _, p := range t.Docs {
				d, _ := model.Document(p)
				b.WriteString("\n### " + p + "\n\n" + text(d))
			}
		}
		statements("Linked from", t.LinkedFrom, true)
		statements("Named without a link", t.Named, false)
		b.WriteString("\n## Signals\n")
	} else {
		fmt.Fprintf(&b, "# Signals of the model\n")
	}
	for _, s := range signals.Names {
		list := t.Findings[s]
		fmt.Fprintf(&b, "\n### %s (%d)\n\n", s, len(list))
		if len(list) == 0 {
			b.WriteString("(none)\n")
		}
		for i, f := range list {
			if t.Kind == "model" && !all && i == shownPerSignal {
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
	Modules []moduleQuestions `json:"modules"`
}

type moduleQuestions struct {
	Module    string   `json:"module"`
	Questions []string `json:"questions"`
}

func questions(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	fsys := w.ModelFS
	doc := questionsDocument{header: newHeader("questions"), Modules: []moduleQuestions{}}
	entries, _ := fs.ReadDir(fsys, ".")
	var b strings.Builder
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := fs.ReadFile(library.FS(), path.Join("questions", e.Name()+".md"))
		if err != nil {
			continue
		}
		mq := moduleQuestions{Module: e.Name(), Questions: []string{}}
		for _, line := range strings.Split(string(data), "\n") {
			if q, ok := strings.CutPrefix(strings.TrimSpace(line), "- "); ok {
				mq.Questions = append(mq.Questions, q)
			}
		}
		doc.Modules = append(doc.Modules, mq)
		fmt.Fprintf(&b, "## %s\n\n", mq.Module)
		for i, q := range mq.Questions {
			fmt.Fprintf(&b, "%d. %s\n", i+1, q)
		}
		b.WriteString("\n")
	}
	if len(doc.Modules) == 0 {
		b.WriteString("The model has no module of the default, so there are no questions.\n")
	}
	return output(o, stdout, doc, strings.TrimSuffix(b.String(), "\n\n")+"\n")
}
