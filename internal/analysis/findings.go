package analysis

import (
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/yuin/goldmark/ast"
)

// Finding names a Definition and says what was observed and what is
// expected. Every source of Findings (verify, bughunt, security, later
// issues and incidents) writes them in this one form.
type Finding struct {
	Definition string `json:"definition"`
	Observed   string `json:"observed"`
	Expected   string `json:"expected"`
	Line       int    `json:"line"`
}

// ParseFindings reads a Findings page: every level-2 heading is one Finding
// and holds exactly one link, the Definition; the paragraphs below it that
// start with "Observed:" and "Expected:" say what was observed and what is
// expected.
func ParseFindings(file string, data []byte) ([]Finding, model.Diagnostics) {
	var ds model.Diagnostics
	page, err := model.ParsePage(data)
	if err != nil {
		ds.Add("findings.syntax", file, page.FrontLine, "%v", err)
		return nil, ds
	}
	var out []Finding
	var current *Finding
	finish := func() {
		if current == nil {
			return
		}
		if current.Observed == "" || current.Expected == "" {
			ds.Add("findings.syntax", file, current.Line, "the Finding for %s needs an Observed: and an Expected: paragraph", current.Definition)
		} else {
			out = append(out, *current)
		}
		current = nil
	}
	for n := page.Root.FirstChild(); n != nil; n = n.NextSibling() {
		if h, ok := n.(*ast.Heading); ok && h.Level <= 2 {
			finish()
			if h.Level == 1 {
				continue
			}
			line := page.Line(h.Lines().At(0).Start)
			links := page.LinksIn(h)
			if len(links) != 1 {
				ds.Add("findings.syntax", file, line, "a Finding's heading holds exactly one link, the Definition it names, such as ## [[members/member]]")
				continue
			}
			current = &Finding{Definition: strings.TrimSuffix(links[0].Target, ".md"), Line: line}
			continue
		}
		if current == nil {
			continue
		}
		text := page.InlineText(n)
		if rest, ok := cutLabel(text, "Observed:"); ok {
			current.Observed = rest
		} else if rest, ok := cutLabel(text, "Expected:"); ok {
			current.Expected = rest
		}
	}
	finish()
	return out, ds
}

func cutLabel(text, label string) (string, bool) {
	if len(text) < len(label) || !strings.EqualFold(text[:len(label)], label) {
		return "", false
	}
	return strings.TrimSpace(text[len(label):]), true
}
