package harness

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// Items turns one Manager's checklist into the port's items, keyed as the
// Run's plan keys them (runremote.Items), in the order own, inherited,
// Findings, stale, each with why the change reaches it.
func Items(c analysis.Checklist) []agentport.Item {
	keys := runremote.Items(c)
	var items []agentport.Item
	for _, it := range c.Own {
		items = append(items, agentport.Item{Part: agentport.Own, Definition: it.Definition, Why: strings.Join(it.Reasons, "; ")})
	}
	for _, it := range c.Inherited {
		items = append(items, agentport.Item{Part: agentport.Inherited, Definition: it.Definition, Why: strings.Join(it.Reasons, "; ")})
	}
	for _, f := range c.Findings {
		items = append(items, agentport.Item{Part: agentport.Finding, Definition: f.Definition, Why: "observed: " + oneLine(f.Observed) + "; expected: " + oneLine(f.Expected)})
	}
	for _, s := range c.Stale {
		item := agentport.Item{Part: agentport.Stale, Why: fmt.Sprintf("%s lists %s, which no longer exists", s.PageFile, s.File)}
		switch {
		case s.Code == "stale.definition":
			item.Definition = s.Target
			item.Why = fmt.Sprintf("%s:%d links %s, which no longer exists", s.PageFile, s.Line, s.Target)
		case s.File == "":
			item.Why = fmt.Sprintf("%s is the page of a Package that no longer exists", s.PageFile)
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
