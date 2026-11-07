package run

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Verdict is what gate tells whoever called it. Done means there is nothing more
// to do: the work passed, the folder matches its certificate, or the report of a
// review is written.
type Verdict struct {
	Done bool
	Text string
}

// verdictsFor reads what the answers to items say, with what the owner is told
// to do about a spec problem.
func verdictsFor(items []Item, answer func(key string) Answer, settle string) ([]string, []OwnerDecision) {
	var wrong []string
	var decisions []OwnerDecision
	for _, it := range items {
		switch a := answer(it.Key); a.Word {
		case AnswerSpecProblem:
			decisions = append(decisions, OwnerDecision{Matter: fmt.Sprintf("%s: %s", it.Key, a.Explanation), Recommendation: settle})
		case AnswerWrong:
			wrong = append(wrong, fmt.Sprintf("%s: %s %s", it.Key, a.Explanation, pointers(a.Pointers)))
		}
	}
	return wrong, decisions
}

// report is what a review of finished work found.
type report struct {
	state, why string
	findings   []ResultFinding
	decisions  []OwnerDecision
	unsettled  []string
}

// verifyReport reads a finished review that changes nothing: a wrong answer about a
// Definition is a Finding, a spec problem a decision for the owner, and what does
// not hold and names no Definition, with the errors of the realization in p, stays
// in the Result, not in the Findings, which each name one.
func verifyReport(items []Item, answer func(key string) Answer, p *project.Project) report {
	var r report
	for _, it := range items {
		a := answer(it.Key)
		switch a.Word {
		case AnswerWrong:
			if it.Definition != "" {
				r.findings = append(r.findings, ResultFinding{Definition: it.Definition, Observed: strings.TrimSpace(a.Explanation + " " + pointers(a.Pointers)), Expected: "what the Definition says"})
			} else {
				r.unsettled = append(r.unsettled, it.Key)
			}
		case AnswerSpecProblem:
			r.decisions = append(r.decisions, OwnerDecision{Matter: fmt.Sprintf("%s: %s", it.Key, a.Explanation), Recommendation: "Settle it in the spec."})
		}
	}
	for _, d := range p.RealizationErrors() {
		r.unsettled = append(r.unsettled, d.String())
	}
	r.state, r.why = StateVerified, "every item answered without a Finding"
	var found []string
	if n := len(r.findings); n > 0 {
		found = append(found, fmt.Sprintf("%d Findings", n))
	}
	if n := len(r.decisions); n > 0 {
		found = append(found, fmt.Sprintf("%d decisions for the owner", n))
	}
	if n := len(r.unsettled); n > 0 {
		found = append(found, fmt.Sprintf("%d things that do not hold: %s", n, strings.Join(r.unsettled, "; ")))
	}
	if len(found) > 0 {
		r.state, r.why = StateFindings, "the review found "+strings.Join(found, " and ")
	}
	return r
}

// errorLines are check's errors for a refusal, at most maxRefusals of them.
func errorLines(p *project.Project) []string {
	var out []string
	for i, d := range p.Errors {
		if i == maxRefusals {
			out = append(out, fmt.Sprintf("and %d more errors", len(p.Errors)-maxRefusals))
			break
		}
		out = append(out, d.String())
	}
	return out
}

// registryPage says whether a file is a registry page, which the work may change.
func registryPage(file string) bool {
	return file == project.RootPage || strings.HasPrefix(file, project.RegistryDir+"/")
}

// leavesTree says whether a symbolic link at file points outside the folder.
func leavesTree(file, to string) bool {
	if path.IsAbs(to) || strings.Contains(to, `\`) {
		return true
	}
	// A drive letter leaves the folder on Windows, where the link is made as it
	// stands.
	if len(to) > 1 && to[1] == ':' {
		return true
	}
	// A ".." after another part can pass through a link, where the text says
	// nothing about the way.
	inner := false
	for _, part := range strings.Split(to, "/") {
		switch {
		case part == ".." && inner:
			return true
		case part != ".." && part != "" && part != ".":
			inner = true
		}
	}
	target := path.Join(path.Dir(file), to)
	return target == ".." || strings.HasPrefix(target, "../")
}

// dedupe keeps the first of each value, in order.
func dedupe(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// sortedKeys returns a map's keys, sorted.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// plural picks the word for a count.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
