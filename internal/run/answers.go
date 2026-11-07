package run

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// A reviewer's answers (concept 3.0 §8).
const (
	AnswerOK          = "ok"
	AnswerWrong       = "wrong"
	AnswerNotShown    = "not shown"
	AnswerSpecProblem = "spec problem"
)

// explanationMark separates an answer's pointers from its explanation.
const explanationMark = "--"

var answerWords = []string{AnswerOK, AnswerWrong, AnswerNotShown, AnswerSpecProblem}

// Pointer is one place a reviewer names: a file, a line, a test.
type Pointer struct {
	File string `json:"file"`
	Line int    `json:"line,omitempty"`
	// Tests names the tests in that file the answer rests on.
	Tests []string `json:"tests,omitempty"`
}

// Answer is one item's answer.
type Answer struct {
	Word        string    `json:"answer"`
	Pointers    []Pointer `json:"pointers,omitempty"`
	Explanation string    `json:"explanation,omitempty"`
}

// pointerPattern reads file, file:line or file:line-line, then what is in brackets.
var pointerPattern = regexp.MustCompile(`^(.+?)(?::(\d+)(?:-\d+)?)?(?:\s+\(([^()]*)\))?$`)

// parseAnswers reads a reviewer's answers: one line per item, "KEY: ANSWER
// POINTERS -- EXPLANATION". A line it cannot read, or one for an item that is
// not on the list, is a problem to correct.
func parseAnswers(raw []byte, items []Item) (map[string]Answer, []string) {
	known := map[string]bool{}
	for _, it := range items {
		known[it.Key] = true
	}
	answers := map[string]Answer{}
	var problems []string
	for _, line := range strings.Split(string(project.Normalize(raw)), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "[ ]"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, word, rest, ok := splitAnswer(line)
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("this line is no answer: %q", clip(line)))
		case !known[key]:
			problems = append(problems, fmt.Sprintf("%q is not on the checklist", key))
		default:
			if _, twice := answers[key]; twice {
				problems = append(problems, fmt.Sprintf("%s has two answers", key))
			}
			answers[key] = parseRest(word, rest)
		}
	}
	return answers, problems
}

// splitAnswer finds the first ": WORD" in the line, so that a key may hold
// anything a file name may.
func splitAnswer(line string) (key, word, rest string, ok bool) {
	best := -1
	for _, w := range answerWords {
		for at := 0; ; {
			i := strings.Index(line[at:], ": "+w)
			if i < 0 {
				break
			}
			i += at
			end := i + 2 + len(w)
			if end == len(line) || line[end] == ' ' {
				if best < 0 || i < best {
					best, word = i, w
				}
				break
			}
			at = end
		}
	}
	if best < 0 {
		return "", "", "", false
	}
	return strings.TrimSpace(line[:best]), word, strings.TrimSpace(line[best+2+len(word):]), true
}

func parseRest(word, rest string) Answer {
	a := Answer{Word: word}
	pointers := rest
	if i := strings.Index(" "+rest, " "+explanationMark+" "); i >= 0 {
		pointers = rest[:i]
		a.Explanation = strings.TrimSpace(rest[i+len(explanationMark)+1:])
	} else if p, ok := strings.CutPrefix(rest, explanationMark); ok {
		pointers, a.Explanation = "", strings.TrimSpace(p)
	}
	for _, part := range splitPointers(pointers) {
		m := pointerPattern.FindStringSubmatch(part)
		if m == nil {
			continue
		}
		p := Pointer{File: strings.TrimSpace(strings.TrimPrefix(m[1], "./"))}
		p.Tests = testNames(m[3])
		if m[2] != "" {
			p.Line, _ = strconv.Atoi(m[2])
		}
		a.Pointers = append(a.Pointers, p)
	}
	return a
}

// testNames reads what is in a pointer's brackets: names of tests, one word each.
// Anything else there is a remark.
func testNames(s string) []string {
	var names []string
	for _, n := range strings.Split(s, ",") {
		n = strings.TrimSpace(n)
		if n == "" || strings.ContainsAny(n, " \t") {
			return nil
		}
		names = append(names, n)
	}
	return names
}

// splitPointers splits at the commas outside brackets.
func splitPointers(s string) []string {
	var parts []string
	depth, start := 0, 0
	add := func(p string) {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	for i, r := range s {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case r == ',' && depth == 0:
			add(s[start:i])
			start = i + 1
		}
	}
	add(s[start:])
	return parts
}

// checkAnswers is what the tool checks of the answers it has: that each location
// exists in the folder as it is and that a named test exists. It returns the
// problems for the reviewer to correct, and the locations the registry does not
// list for their items, which are reported and refuse nothing.
func checkAnswers(state project.Repository, items []Item, answers map[string]Answer) (problems, missingLinks []string) {
	for _, it := range items {
		a, ok := answers[it.Key]
		if !ok {
			continue
		}
		if a.Word != AnswerOK && a.Explanation == "" {
			problems = append(problems, fmt.Sprintf("%s: %s needs an explanation after %q", it.Key, a.Word, explanationMark))
		}
		if a.Word == AnswerOK && len(a.Pointers) == 0 && it.Kind != ItemGone {
			problems = append(problems, fmt.Sprintf("%s: ok needs a pointer to where it holds", it.Key))
		}
		listed := map[string]bool{}
		for _, f := range it.Look {
			listed[f] = true
		}
		for i := range a.Pointers {
			p := &a.Pointers[i]
			data, err := state.Read(p.File)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %s does not exist in the folder", it.Key, p.File))
				continue
			}
			if p.Line > strings.Count(string(data), "\n")+1 {
				problems = append(problems, fmt.Sprintf("%s: %s has no line %d", it.Key, p.File, p.Line))
			}
			for _, name := range p.Tests {
				if !strings.Contains(string(data), name) {
					problems = append(problems, fmt.Sprintf("%s: %s names no test %s", it.Key, p.File, name))
				}
			}
			if it.Kind == ItemDefinition && !listed[p.File] {
				missingLinks = append(missingLinks, fmt.Sprintf("%s: %s is not listed for it", it.Key, p.File))
			}
		}
	}
	return problems, dedupe(missingLinks)
}

// clip shortens a line an agent wrote for a message back to it.
func clip(s string) string {
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
