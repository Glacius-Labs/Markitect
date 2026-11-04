package run

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Kinds of checklist item (concept 3.0 §8).
const (
	// ItemDefinition is a Definition the change reaches.
	ItemDefinition = "definition"
	// ItemGone is a deleted Definition: its files are removed or have a new
	// basis.
	ItemGone = "gone"
	// ItemFinding is a Finding from verify or any other source.
	ItemFinding = "finding"
	// ItemStale is a stale registry entry to settle.
	ItemStale = "stale"
	// ItemUnlisted is a file a hand edit left out of the registry.
	ItemUnlisted = "unlisted"
	// ItemUnused is a Definition nothing links to and no file realizes.
	ItemUnused = "unused"
)

// Item is one checklist item with where to look. Its Key is what an answer
// names; it holds no line break.
type Item struct {
	Key  string `json:"key"`
	Kind string `json:"kind"`
	// Definition names the Definition the item is about, when there is one.
	Definition string `json:"definition,omitempty"`
	// Title is the Definition's title, to read the list by.
	Title string `json:"title,omitempty"`
	// Why says why the item is on the list.
	Why []string `json:"why,omitempty"`
	// Look lists the files the registry names for the item.
	Look []string `json:"look,omitempty"`
}

// checklist generates the items of a Command from its impact: every reached
// Definition once, with where to look, every deleted Definition, every Finding
// and stale entry, and every file the project does not account for.
func checklist(imp analysis.Impact, before, cur *project.Project, findings []analysis.Finding) []Item {
	look := realizers(cur)
	var items []Item
	for _, it := range imp.Reached {
		item := Item{Key: it.Definition, Kind: ItemDefinition, Definition: it.Definition, Why: it.Reasons, Look: look[it.Definition]}
		if d := cur.Model.Definitions[it.Definition]; d != nil {
			item.Title = d.Title
		}
		items = append(items, item)
	}
	gone := realizers(before)
	for _, c := range imp.Definitions {
		if c.Change == analysis.Removed {
			items = append(items, Item{Key: "gone " + c.Path, Kind: ItemGone, Definition: c.Path, Why: []string{"this is gone: its files are removed or have a new basis"}, Look: gone[c.Path]})
		}
	}
	for i, f := range findings {
		if d := cur.Model.Definitions[f.Definition]; d == nil || !d.InModel() {
			continue
		}
		items = append(items, Item{Key: fmt.Sprintf("finding %d %s", i+1, f.Definition), Kind: ItemFinding, Definition: f.Definition,
			Why: []string{fmt.Sprintf("observed %s; expected %s", oneLine(f.Observed), oneLine(f.Expected))}, Look: look[f.Definition]})
	}
	for _, s := range cur.Registry.Stale {
		subject := s.File
		if subject == "" {
			subject = s.Target
		}
		var at []string
		if s.File != "" {
			at = []string{s.File}
		}
		items = append(items, Item{Key: "stale " + subject, Kind: ItemStale, Why: []string{fmt.Sprintf("%s on %s", s.Code, s.PageFile)}, Look: at})
	}
	for _, d := range cur.Errors {
		switch d.Code {
		case project.CodeUnregistered:
			items = append(items, Item{Key: "unlisted " + d.File, Kind: ItemUnlisted, Why: []string{"neither registered nor ignored"}, Look: []string{d.File}})
		case project.CodeUnused:
			items = append(items, Item{Key: "unused " + d.File, Kind: ItemUnused, Why: []string{d.Message}, Look: []string{d.File}})
		}
	}
	return items
}

// realizers maps each Definition to the files the registry says realize it.
func realizers(p *project.Project) map[string][]string {
	out := map[string][]string{}
	if p == nil || p.Registry == nil {
		return out
	}
	for _, e := range p.Registry.Entries {
		for _, id := range e.Realizes {
			out[id] = append(out[id], e.File)
		}
	}
	for _, files := range out {
		sort.Strings(files)
	}
	return out
}

// mergeItems is the list after the work: what the rebuilt list says, and every
// earlier item it no longer holds, so that no item drops off because of what
// the work did.
func mergeItems(earlier, rebuilt []Item) []Item {
	at := map[string]int{}
	for i, it := range rebuilt {
		at[it.Key] = i
	}
	out := rebuilt
	for _, it := range earlier {
		if i, on := at[it.Key]; on {
			// Where it was to look stays: a repair that deletes a file and its
			// row is a change to what the item's answer rests on.
			out[i].Look = dedupe(append(out[i].Look, it.Look...))
			continue
		}
		// An unlisted file, stale entry or unused Definition the work settled is
		// gone from the project, and check has seen to it.
		if it.Kind != ItemUnlisted && it.Kind != ItemStale && it.Kind != ItemUnused {
			out = append(out, it)
		}
	}
	return out
}

// renderList is the list as a prompt shows it.
func renderList(items []Item) string {
	var b strings.Builder
	for _, it := range items {
		if it.Title != "" {
			fmt.Fprintf(&b, "[ ] %s: %q\n", it.Key, oneLine(it.Title))
		} else {
			fmt.Fprintf(&b, "[ ] %s\n", it.Key)
		}
		if len(it.Why) > 0 {
			fmt.Fprintf(&b, "    why: %s\n", oneLine(strings.Join(it.Why, "; ")))
		}
		if len(it.Look) > 0 {
			fmt.Fprintf(&b, "    look at: %s\n", strings.Join(it.Look, ", "))
		}
	}
	return b.String()
}
