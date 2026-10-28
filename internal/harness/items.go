package harness

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// Parts of a checklist (concept 2.0 §6.2).
const (
	PartOwn       = "own"
	PartInherited = "inherited"
	PartFinding   = "finding"
	PartStale     = "stale"
)

// Item is one checklist item as a Manager receives it. Key names it as the
// Run's plan and the verdict trailers do (runremote.Items).
type Item struct {
	Key  string
	Part string
	// Definition is the Definition the item is about; empty for stale
	// realization that names no Definition.
	Definition string
	// Detail says why the item is on the checklist: the reasons it is
	// reached, a Finding's observation and expectation, or the stale entry.
	Detail string
}

// Items turns one Manager's checklist into its items, keyed as the Run's
// plan keys them, in the order own, inherited, Findings, stale.
func Items(c analysis.Checklist) []Item {
	keys := runremote.Items(c)
	var items []Item
	for _, it := range c.Own {
		items = append(items, Item{Part: PartOwn, Definition: it.Definition, Detail: strings.Join(it.Reasons, "; ")})
	}
	for _, it := range c.Inherited {
		items = append(items, Item{Part: PartInherited, Definition: it.Definition, Detail: strings.Join(it.Reasons, "; ")})
	}
	for _, f := range c.Findings {
		items = append(items, Item{Part: PartFinding, Definition: f.Definition, Detail: "observed: " + oneLine(f.Observed) + "; expected: " + oneLine(f.Expected)})
	}
	for _, s := range c.Stale {
		item := Item{Part: PartStale, Detail: fmt.Sprintf("%s lists %s, which no longer exists", s.PageFile, s.File)}
		switch {
		case s.Code == "stale.definition":
			item.Definition = s.Target
			item.Detail = fmt.Sprintf("%s:%d links %s, which no longer exists", s.PageFile, s.Line, s.Target)
		case s.File == "":
			item.Detail = fmt.Sprintf("%s is the page of a Package that no longer exists", s.PageFile)
		}
		items = append(items, item)
	}
	for i := range items {
		items[i].Key = keys[i]
	}
	return items
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
