package harness

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

// Parts of a checklist (concept 2.0 §6.2).
const (
	PartOwn       = "own"
	PartInherited = "inherited"
	PartFinding   = "finding"
	PartStale     = "stale"
)

// Item is one checklist item as a Manager receives it. Key names it in the
// Run's plan and in the verdict trailers; it holds no "|" and no line break.
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

// Items turns one Manager's checklist into its items, in the order own,
// inherited, Findings, stale. Own and inherited items are keyed by their
// Definition ID, Findings by "finding-N:" and their Definition, stale
// realization by "stale:" and the file, the missing Definition or the
// orphaned page. Stale entries that name the same thing are one item. The
// engine names the plan's checklist with these keys, so the hooks and the
// Managers agree on them.
func Items(c analysis.Checklist) []Item {
	var items []Item
	for _, it := range c.Own {
		items = append(items, Item{Key: it.Definition, Part: PartOwn, Definition: it.Definition, Detail: strings.Join(it.Reasons, "; ")})
	}
	for _, it := range c.Inherited {
		items = append(items, Item{Key: it.Definition, Part: PartInherited, Definition: it.Definition, Detail: strings.Join(it.Reasons, "; ")})
	}
	for i, f := range c.Findings {
		items = append(items, Item{
			Key: fmt.Sprintf("finding-%d:%s", i+1, f.Definition), Part: PartFinding, Definition: f.Definition,
			Detail: "observed: " + oneLine(f.Observed) + "; expected: " + oneLine(f.Expected),
		})
	}
	seen := map[string]int{}
	for _, s := range c.Stale {
		key, item := staleItem(s)
		if at, ok := seen[key]; ok {
			items[at].Detail += "; " + item.Detail
			continue
		}
		seen[key] = len(items)
		items = append(items, item)
	}
	return items
}

func staleItem(s realization.Stale) (string, Item) {
	var key, detail, definition string
	switch {
	case s.Code == "stale.definition":
		key, definition = "stale:"+s.Target, s.Target
		detail = fmt.Sprintf("%s:%d links %s, which no longer exists", s.PageFile, s.Line, s.Target)
	case s.File != "":
		key = "stale:" + s.File
		detail = fmt.Sprintf("%s lists %s (%s)", s.PageFile, s.File, s.Code)
	default:
		key = "stale:" + s.PageFile
		detail = fmt.Sprintf("%s is the page of a Package that no longer exists", s.PageFile)
	}
	key = strings.NewReplacer("|", "/", "\n", " ", "\r", " ").Replace(key)
	return key, Item{Key: key, Part: PartStale, Definition: definition, Detail: detail}
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
