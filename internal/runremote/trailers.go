package runremote

import (
	"fmt"
	"strings"
)

// The trailer keys of a commit message (README: "Verdict trailers").
const (
	KeyManager = "Markitect-Manager"
	KeyVerdict = "Markitect-Verdict"
	KeyReview  = "Markitect-Review"
	KeyCheck   = "Markitect-Check"
)

// Verdicts a maker gives a checklist item.
const (
	Conforms     = "conforms"
	Changed      = "changed"
	ModelProblem = "model-problem"
)

// Review outcomes and per-item answers of a checker.
const (
	Approved  = "approved"
	Rejected  = "rejected"
	Confirmed = "confirmed"
	Disputed  = "disputed"
)

// Kind classifies a commit by its Markitect trailers.
type Kind int

const (
	// Plain: no Markitect trailer, an ordinary work commit.
	Plain Kind = iota
	// Submission: a maker's verdicts (or, in verify, a checker's).
	Submission
	// Review: a checker's answer to a submission.
	Review
)

// Line is one verdict or check line: the checklist item, the verdict or
// answer, and its evidence.
type Line struct {
	Item, Word, Evidence string
}

// Trailers are the Markitect trailers of one commit message.
type Trailers struct {
	Kind     Kind
	Manager  string
	Verdicts []Line
	Review   string
	Checks   []Line
}

// ParseMessage reads the Markitect trailers of a commit message. They must
// form the message's last paragraph, as Git trailers do; Git's other
// trailers may stand beside them. A message without them is Plain. An error
// names the first line that breaks the format.
func ParseMessage(message string) (Trailers, error) {
	lines := strings.Split(strings.ReplaceAll(strings.TrimRight(message, "\n"), "\r\n", "\n"), "\n")
	start := len(lines)
	for start > 0 && strings.TrimSpace(lines[start-1]) != "" {
		start--
	}
	for i, l := range lines[:start] {
		if markitectKey(l) {
			return Trailers{}, fmt.Errorf("line %d %q: Markitect trailer lines must form the message's last paragraph", i+1, l)
		}
	}
	var t Trailers
	var entries [][2]string
	for _, l := range lines[start:] {
		if (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t")) && len(entries) > 0 {
			entries[len(entries)-1][1] += " " + strings.TrimSpace(l)
			continue
		}
		key, value, ok := strings.Cut(l, ":")
		if !ok {
			key = ""
		}
		entries = append(entries, [2]string{key, strings.TrimSpace(value)})
	}
	found := false
	for _, e := range entries {
		found = found || markitectKey(e[0])
	}
	if !found {
		return Trailers{}, nil
	}
	for _, e := range entries {
		key, value := e[0], e[1]
		if !markitectKey(key) {
			if key == "" || strings.ContainsAny(key, " \t") {
				return Trailers{}, fmt.Errorf("%q: every line of the trailer paragraph must be \"Key: value\"", key+value)
			}
			continue
		}
		switch key {
		case KeyManager:
			if t.Manager != "" {
				return Trailers{}, fmt.Errorf("%s appears twice", KeyManager)
			}
			if value == "" {
				return Trailers{}, fmt.Errorf("%s: names no Manager", KeyManager)
			}
			t.Manager = value
		case KeyVerdict:
			l, err := parseLine(key, value, Conforms, Changed, ModelProblem)
			if err != nil {
				return Trailers{}, err
			}
			t.Verdicts = append(t.Verdicts, l)
		case KeyCheck:
			l, err := parseLine(key, value, Confirmed, Disputed)
			if err != nil {
				return Trailers{}, err
			}
			t.Checks = append(t.Checks, l)
		case KeyReview:
			if t.Review != "" {
				return Trailers{}, fmt.Errorf("%s appears twice", KeyReview)
			}
			if value != Approved && value != Rejected {
				return Trailers{}, fmt.Errorf("%s: %q must be %q or %q", KeyReview, value, Approved, Rejected)
			}
			t.Review = value
		default:
			return Trailers{}, fmt.Errorf("%s: unknown Markitect trailer; use exactly %s, %s, %s or %s", key, KeyManager, KeyVerdict, KeyReview, KeyCheck)
		}
	}
	switch {
	case t.Manager == "":
		return Trailers{}, fmt.Errorf("Markitect trailers need a %s line naming your Manager", KeyManager)
	case len(t.Verdicts) > 0 && (t.Review != "" || len(t.Checks) > 0):
		return Trailers{}, fmt.Errorf("a commit holds either %s lines (a submission) or %s and %s lines (a review), not both", KeyVerdict, KeyReview, KeyCheck)
	case t.Review != "":
		t.Kind = Review
	case len(t.Checks) > 0:
		return Trailers{}, fmt.Errorf("%s lines need a %s line", KeyCheck, KeyReview)
	default:
		// A Manager with an empty checklist, a parent that only integrates,
		// submits with its Manager line alone.
		t.Kind = Submission
	}
	return t, nil
}

// markitectKey reports whether a line or key starts with "Markitect-" in any
// case: Git matches trailer keys without regard to case, so a key in another
// case must be refused, not read as plain text.
func markitectKey(s string) bool {
	return len(s) >= len("Markitect-") && strings.EqualFold(s[:len("Markitect-")], "Markitect-")
}

func parseLine(key, value string, words ...string) (Line, error) {
	parts := strings.SplitN(value, "|", 3)
	if len(parts) != 3 {
		return Line{}, fmt.Errorf("%s: %q must read \"ITEM | %s | EVIDENCE\"", key, value, strings.Join(words, "|"))
	}
	l := Line{Item: strings.TrimSpace(parts[0]), Word: strings.TrimSpace(parts[1]), Evidence: strings.TrimSpace(parts[2])}
	if l.Item == "" {
		return Line{}, fmt.Errorf("%s: %q names no checklist item", key, value)
	}
	ok := false
	for _, w := range words {
		ok = ok || l.Word == w
	}
	if !ok {
		return Line{}, fmt.Errorf("%s: %q for %s must be one of %s", key, l.Word, l.Item, strings.Join(words, ", "))
	}
	if l.Evidence == "" {
		return Line{}, fmt.Errorf("%s: %s %s gives no evidence", key, l.Item, l.Word)
	}
	return l, nil
}

// String writes the trailers as the last paragraph of a commit message.
func (t Trailers) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n", KeyManager, t.Manager)
	for _, l := range t.Verdicts {
		fmt.Fprintf(&b, "%s: %s | %s | %s\n", KeyVerdict, l.Item, l.Word, l.Evidence)
	}
	if t.Review != "" {
		fmt.Fprintf(&b, "%s: %s\n", KeyReview, t.Review)
	}
	for _, l := range t.Checks {
		fmt.Fprintf(&b, "%s: %s | %s | %s\n", KeyCheck, l.Item, l.Word, l.Evidence)
	}
	return b.String()
}
