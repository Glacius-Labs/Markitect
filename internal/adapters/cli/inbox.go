package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// inboxUsage is the inbox command's usage.
const inboxUsage = `usage:
  markitect inbox [--repo DIR] [--json] [--all]
  markitect inbox add --source NAME --kind NAME --settle owner|agent -m "TEXT" [--ask "QUESTION OR CHANGE"] [--place PLACE]... [--repo DIR] [--json]
  markitect inbox close ID --settled|--answered|--dropped -m "WHY" [--repo DIR]`

// inbox lists, opens and closes the items of the project's inbox (DEC-118):
// what waits, for the owner or for an agent. It lists the open items, or with
// --all every item; add opens one; close closes one. The tool checks an item's
// shape; an item open already is not opened twice.
func (c CLI) inbox(ctx context.Context, args []string) (int, error) {
	o := struct {
		repo, source, kind, settle, text, ask, how string
		place                                      []string
		json, all                                  bool
	}{repo: "."}
	var positional []string
	set := map[string]bool{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, inline, hasInline := strings.Cut(strings.TrimLeft(a, "-"), "=")
		switch {
		case a == "-h" || a == "--help" || a == "-help":
			fmt.Fprintln(c.Stderr, inboxUsage)
			return Misused, flag.ErrHelp
		case !strings.HasPrefix(a, "-") || a == "-":
			positional = append(positional, a)
			continue
		}
		set[name] = true
		switch name {
		case "json", "all":
			if hasInline {
				return Misused, fmt.Errorf("--%s takes no value\n%s", name, inboxUsage)
			}
			o.json, o.all = o.json || name == "json", o.all || name == "all"
			continue
		case "settled", "answered", "dropped":
			if hasInline || o.how != "" {
				return Misused, fmt.Errorf("an item closes one way: --settled, --answered or --dropped\n%s", inboxUsage)
			}
			o.how = name
			continue
		case "repo", "source", "kind", "settle", "m", "ask", "place":
		default:
			return Misused, fmt.Errorf("unknown flag %s\n%s", a, inboxUsage)
		}
		v := inline
		if !hasInline {
			if i+1 >= len(args) {
				return Misused, fmt.Errorf("%s needs a value\n%s", a, inboxUsage)
			}
			i++
			v = args[i]
		}
		switch name {
		case "repo":
			o.repo = v
		case "source":
			o.source = v
		case "kind":
			o.kind = v
		case "settle":
			o.settle = v
		case "m":
			o.text = v
		case "ask":
			o.ask = v
		case "place":
			o.place = append(o.place, strings.Split(v, ",")...)
		}
	}
	allowed := map[string][]string{
		"":      {"repo", "json", "all"},
		"add":   {"repo", "json", "source", "kind", "settle", "m", "ask", "place"},
		"close": {"repo", "settled", "answered", "dropped", "m"},
	}
	what := ""
	if len(positional) > 0 {
		what = positional[0]
	}
	if _, ok := allowed[what]; !ok {
		return Misused, fmt.Errorf("the inbox lists, adds or closes, not %q\n%s", what, inboxUsage)
	}
	for f := range set {
		if !slices.Contains(allowed[what], f) {
			return Misused, fmt.Errorf("--%s does not go with %s\n%s", f, strings.TrimSpace("markitect inbox "+what), inboxUsage)
		}
	}
	usage := func(err error) (int, error) {
		if errors.Is(err, run.ErrUsage) {
			return Misused, fmt.Errorf("%s\n%s", strings.TrimPrefix(err.Error(), run.ErrUsage.Error()+": "), inboxUsage)
		}
		return Misused, err
	}
	switch what {
	case "add":
		if len(positional) != 1 {
			return Misused, fmt.Errorf("add takes its item as flags\n%s", inboxUsage)
		}
		item, opened, err := c.Record.InboxAdd(ctx, o.repo, run.InboxLine{Source: o.source, Kind: o.kind, Settle: o.settle, Text: o.text, Ask: o.ask, Place: o.place})
		if err != nil {
			return usage(err)
		}
		if o.json {
			return OK, c.printJSON(struct {
				envelope
				Opened bool          `json:"opened"`
				Item   run.InboxLine `json:"item"`
			}{newEnvelope("inbox-add"), opened, item})
		}
		if opened {
			fmt.Fprintf(c.Stdout, "Opened %s.\n", item.ID)
		} else {
			fmt.Fprintf(c.Stdout, "Open already as %s; nothing is added.\n", item.ID)
		}
		return OK, nil
	case "close":
		if len(positional) != 2 {
			return Misused, fmt.Errorf("close names the item, as markitect inbox lists it\n%s", inboxUsage)
		}
		refusal, err := c.Record.InboxClose(ctx, o.repo, strings.ToLower(positional[1]), o.how, o.text)
		if err != nil {
			return usage(err)
		}
		if refusal != "" {
			fmt.Fprintln(c.Stdout, refusal)
			return Failed, nil
		}
		fmt.Fprintf(c.Stdout, "Closed %s: %s.\n", strings.ToLower(positional[1]), o.how)
		return OK, nil
	}
	if len(positional) != 0 {
		return Misused, fmt.Errorf("the inbox lists, adds or closes, not %q\n%s", positional[0], inboxUsage)
	}
	b, err := c.Record.Inbox(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	items := b.Open()
	if o.all {
		items = b.Items
	}
	if o.json {
		if items == nil {
			items = []run.InboxItem{}
		}
		return OK, c.printJSON(struct {
			envelope
			Counts run.InboxCounts `json:"counts"`
			Items  []run.InboxItem `json:"items"`
		}{newEnvelope("inbox"), b.Counts(), items})
	}
	fmt.Fprint(c.Stdout, inboxText(b.Counts(), items))
	return OK, nil
}

// inboxText prints items as the inbox lists them: each with its ID, who may
// settle it, its kind, source and place, what it says and what it asks.
func inboxText(counts run.InboxCounts, items []run.InboxItem) string {
	if len(items) == 0 {
		return "The inbox holds no open item.\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "The inbox holds %d open %s, %d for the owner:\n", counts.Open, plural(counts.Open, "item", "items"), counts.ForOwner)
	for _, it := range items {
		fmt.Fprintf(&b, "\n%s  for the %s  %s, from %s", it.ID, it.Settle, it.Kind, it.Source)
		if len(it.Place) > 0 {
			fmt.Fprintf(&b, "  at %s", strings.Join(it.Place, ", "))
		}
		fmt.Fprintf(&b, "\n  %s\n", it.Text)
		if it.Ask != "" {
			fmt.Fprintf(&b, "  asks: %s\n", it.Ask)
		}
		if it.Closed != nil {
			fmt.Fprintf(&b, "  closed: %s, %s\n", it.Closed.How, it.Closed.Why)
		}
	}
	return b.String()
}
