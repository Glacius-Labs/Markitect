package run

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The inbox is what waits (DEC-118): a question for the owner, a finding of an
// audit, and later signals from outside. One line opens an item and a later
// line closes it; nothing is rewritten, and an item's state comes from its
// lines. The history is what happened; neither belongs in the spec.

// The types of the inbox's lines.
const (
	InboxOpen  = "open"
	InboxClose = "close"
)

// Who may settle an item: the owner alone, for a decision only the owner can
// make, or an agent, which may settle it in the spec on its own, such as a
// missing link.
const (
	SettleOwner = "owner"
	SettleAgent = "agent"
)

// Settlers are who may settle an item, and Closings how an item closes: settled
// in the spec, answered, or dropped.
var (
	Settlers = []string{SettleOwner, SettleAgent}
	Closings = []string{"settled", "answered", "dropped"}
)

// name is what a source or a kind may look like: a free name, so that new
// sources and kinds need no change of the tool.
var name = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// InboxLine is one line of the inbox: an item opened, with what it says, or
// closed, with how and why.
type InboxLine struct {
	Time time.Time `json:"time"`
	Type string    `json:"type"`
	ID   string    `json:"id"`
	// Source names who opened it, such as architect, audit, reviewer or owner;
	// Kind what it is, such as contradiction, gap, unclear, duplicate,
	// missing-link, coverage, design or question; Place the Definitions or
	// paths it concerns, none for a signal from outside; Text what is wrong or
	// open; Ask the question to answer or the change proposed; Settle who may
	// settle it.
	Source string   `json:"source,omitempty"`
	Kind   string   `json:"kind,omitempty"`
	Place  []string `json:"place,omitempty"`
	Text   string   `json:"text,omitempty"`
	Ask    string   `json:"ask,omitempty"`
	Settle string   `json:"settle,omitempty"`
	// How and Why say how an item closed and why.
	How string `json:"how,omitempty"`
	Why string `json:"why,omitempty"`
}

// InboxItem is an item of the inbox: the line that opened it, and the line
// that closed it, if one did.
type InboxItem struct {
	InboxLine
	Closed *InboxLine `json:"closed,omitempty"`
}

// Inbox is the items of a project's inbox, in the order they were opened.
type Inbox struct {
	Items []InboxItem
}

// ParseInbox reads an inbox: one JSON object per line, in time order, a line
// that both sides of a merge hold once. A close counts for an item opened
// anywhere in the file, and of several closes the first counts. A line it
// cannot read counts nothing.
func ParseInbox(data []byte) Inbox {
	var lines []InboxLine
	seen := map[string]bool{}
	for _, raw := range strings.Split(string(project.Normalize(data)), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		var l InboxLine
		if err := json.Unmarshal([]byte(raw), &l); err != nil || l.Time.IsZero() || l.ID == "" {
			continue
		}
		lines = append(lines, l)
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Time.Before(lines[j].Time) })
	var b Inbox
	at := map[string]int{}
	for _, l := range lines {
		if _, ok := at[l.ID]; !ok && l.Type == InboxOpen {
			at[l.ID] = len(b.Items)
			b.Items = append(b.Items, InboxItem{InboxLine: l})
		}
	}
	for _, l := range lines {
		if i, ok := at[l.ID]; ok && l.Type == InboxClose && b.Items[i].Closed == nil {
			c := l
			b.Items[i].Closed = &c
		}
	}
	return b
}

// Open returns the items no line has closed.
func (b Inbox) Open() []InboxItem {
	var out []InboxItem
	for _, it := range b.Items {
		if it.Closed == nil {
			out = append(out, it)
		}
	}
	return out
}

// Item returns the item of an ID.
func (b Inbox) Item(id string) (InboxItem, bool) {
	for _, it := range b.Items {
		if it.ID == id {
			return it, true
		}
	}
	return InboxItem{}, false
}

// InboxCounts is how many items are open, and how many of them only the owner
// may settle.
type InboxCounts struct {
	Open     int `json:"open"`
	ForOwner int `json:"for_owner"`
}

// Counts counts the open items.
func (b Inbox) Counts() InboxCounts {
	var c InboxCounts
	for _, it := range b.Open() {
		c.Open++
		if it.Settle == SettleOwner {
			c.ForOwner++
		}
	}
	return c
}

// check refuses an item whose shape the inbox does not take.
func (l *InboxLine) check() error {
	l.Text, l.Ask = oneLine(l.Text), oneLine(l.Ask)
	var place []string
	for _, p := range l.Place {
		if p = strings.TrimSpace(p); p != "" && !slices.Contains(place, p) {
			place = append(place, p)
		}
	}
	sort.Strings(place)
	l.Place = place
	switch {
	case !name.MatchString(l.Source):
		return fmt.Errorf("%w: --source names who opens the item, in lower case, such as architect, audit, reviewer or owner, not %q", ErrUsage, l.Source)
	case !name.MatchString(l.Kind):
		return fmt.Errorf("%w: --kind names what the item is, in lower case, such as contradiction, gap, unclear, duplicate, missing-link, coverage, design or question, not %q", ErrUsage, l.Kind)
	case !slices.Contains(Settlers, l.Settle):
		return fmt.Errorf("%w: --settle says who may settle the item, %s, not %q", ErrUsage, strings.Join(Settlers, " or "), l.Settle)
	case l.Text == "":
		return fmt.Errorf("%w: an item says what is wrong or open, with -m", ErrUsage)
	}
	return nil
}

// same says whether two items say the same thing: of one kind, at one place,
// with one text.
func (l InboxLine) same(o InboxLine) bool {
	return l.Kind == o.Kind && l.Text == o.Text && slices.Equal(l.Place, o.Place)
}

// readInbox reads a project's inbox; empty without one.
func readInbox(root string) (Inbox, error) {
	data, err := readOwnFile(root, project.InboxFile)
	if err != nil {
		return Inbox{}, err
	}
	return ParseInbox(data), nil
}

// Inbox reads the inbox of the project that holds dir.
func (e *Engine) Inbox(_ context.Context, dir string) (Inbox, error) {
	root, err := e.Folder.Root(dir)
	if err != nil {
		return Inbox{}, err
	}
	return readInbox(root)
}

// InboxAdd opens an item in the inbox of the project that holds dir, under the
// lock of its record. An item open already with the same kind, place and text
// is not opened twice: it returns that one, and opened false.
func (e *Engine) InboxAdd(_ context.Context, dir string, l InboxLine) (item InboxLine, opened bool, err error) {
	if err := l.check(); err != nil {
		return InboxLine{}, false, err
	}
	root, release, err := e.lockRecord(dir)
	if err != nil {
		return InboxLine{}, false, err
	}
	defer release()
	b, err := readInbox(root)
	if err != nil {
		return InboxLine{}, false, err
	}
	for _, it := range b.Open() {
		if it.same(l) {
			return it.InboxLine, false, nil
		}
	}
	l.Type, l.Time, l.How, l.Why = InboxOpen, e.now(), "", ""
	sum := sha256.Sum256([]byte(l.Time.Format(time.RFC3339Nano) + "\x00" + l.Source + "\x00" + l.Kind + "\x00" + strings.Join(l.Place, "\x01") + "\x00" + l.Text))
	l.ID = "i" + hex.EncodeToString(sum[:6])
	if err := appendLine(root, project.InboxFile, l); err != nil {
		return InboxLine{}, false, err
	}
	return l, true, nil
}

// InboxClose closes an open item of the inbox of the project that holds dir,
// under the lock of its record: settled in the spec, answered or dropped, and
// why. It refuses an item that is not open, saying why.
func (e *Engine) InboxClose(_ context.Context, dir, id, how, why string) (refusal string, err error) {
	why = oneLine(why)
	switch {
	case !slices.Contains(Closings, how):
		return "", fmt.Errorf("%w: an item closes %s, not %q", ErrUsage, strings.Join(Closings, ", "), how)
	case why == "":
		return "", fmt.Errorf("%w: closing says why, with -m", ErrUsage)
	}
	root, release, err := e.lockRecord(dir)
	if err != nil {
		return "", err
	}
	defer release()
	b, err := readInbox(root)
	if err != nil {
		return "", err
	}
	it, ok := b.Item(id)
	switch {
	case !ok:
		return fmt.Sprintf("The inbox has no item %s; markitect inbox lists the open ones.", id), nil
	case it.Closed != nil:
		return fmt.Sprintf("%s is closed already: %s, %s.", id, it.Closed.How, it.Closed.Why), nil
	}
	return "", appendLine(root, project.InboxFile, InboxLine{Time: e.now(), Type: InboxClose, ID: id, How: how, Why: why})
}

// lockRecord takes the lock of the record of the project that holds dir.
func (e *Engine) lockRecord(dir string) (root string, release func(), err error) {
	if root, err = e.Folder.Root(dir); err != nil {
		return "", nil, err
	}
	locks, err := e.lockDir()
	if err != nil {
		return "", nil, err
	}
	if release, err = lock(locks, recordKey(root), lockWait); err != nil {
		return "", nil, fmt.Errorf("the record of %s is locked: %w", root, err)
	}
	return root, release, nil
}
