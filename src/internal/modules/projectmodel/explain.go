package projectmodel

import "sort"

// element is one thing a change can reach: an impact entry (statement, manager,
// file, check) or an artifact, decision or definition that links entries.
type element struct{ kind, id string }

// cause says why an element was routed: a root reason when from is zero,
// otherwise the element that led to it and the model relation between them.
type cause struct {
	reason   string
	from     element
	relation string
}

// causes records every cause of every routed element; Impact takes its
// element sets from it and Explain its witness paths and classes.
type causes map[element]map[cause]bool

func (c causes) add(e element, why cause) {
	if e.id == "" || why.from != (element{}) && why.from.id == "" {
		return
	}
	if c[e] == nil {
		c[e] = map[cause]bool{}
	}
	c[e][why] = true
}

func (c causes) ids(kind string) []string {
	var out []string
	for e := range c {
		if e.kind == kind {
			out = append(out, e.id)
		}
	}
	sort.Strings(out)
	return out
}

// Explain says why each element of the impact from base to candidate is in it,
// by a shortest witness path from a change, and whether it must change or is
// only context to read (DEC-023). It never alters the impact: the elements are
// exactly those of Impact(base, candidate), whose digest the result carries.
func Explain(base, candidate Report) ImpactExplanation {
	r := route(base, candidate)
	parents := r.causes.shortestCauses()
	change := r.changeClass()
	out := ImpactExplanation{APIVersion: APIVersion, ImpactDigest: r.impact.Digest}
	for _, kind := range []struct {
		name string
		ids  []string
	}{{"statement", r.impact.AffectedStatements}, {"manager", r.impact.Managers}, {"file", r.impact.Files}, {"check", r.impact.Checks}} {
		for _, id := range kind.ids {
			e := element{kind.name, id}
			explained := ExplainedElement{Kind: kind.name, ID: id, Class: "context", Reason: "unexplained", Witness: []WitnessStep{}}
			if change[e] {
				explained.Class = "change"
			}
			if _, ok := parents[e]; ok {
				explained.Reason = parents[e].reason
				explained.Witness = witness(e, parents)
			}
			out.Elements = append(out.Elements, explained)
		}
	}
	out.Digest = digest(struct {
		API, Impact string
		Elements    []ExplainedElement
	}{out.APIVersion, out.ImpactDigest, out.Elements})
	return out
}

// shortestCauses runs a breadth-first search from every root cause over the
// recorded causes and keeps, for each element, the cause that first reached it.
// Roots and edges are visited in sorted order, so the choice is deterministic.
func (c causes) shortestCauses() map[element]cause {
	type edge struct {
		to  element
		why cause
	}
	next := map[element][]edge{}
	var queue []element
	parents := map[element]cause{}
	for e, whys := range c {
		for why := range whys {
			if why.from == (element{}) {
				if old, ok := parents[e]; !ok || why.reason < old.reason {
					parents[e] = why
				}
				continue
			}
			next[why.from] = append(next[why.from], edge{e, why})
		}
	}
	for e := range parents {
		queue = append(queue, e)
	}
	sortElements(queue)
	for _, edges := range next {
		sort.Slice(edges, func(i, j int) bool {
			a, b := edges[i], edges[j]
			if a.to != b.to {
				return lessElement(a.to, b.to)
			}
			if a.why.relation != b.why.relation {
				return a.why.relation < b.why.relation
			}
			return a.why.reason < b.why.reason
		})
	}
	for len(queue) > 0 {
		e := queue[0]
		queue = queue[1:]
		for _, out := range next[e] {
			if _, seen := parents[out.to]; !seen {
				parents[out.to] = out.why
				queue = append(queue, out.to)
			}
		}
	}
	return parents
}

// witness follows the chosen causes back from e to a root and returns the
// steps from that root to e.
func witness(e element, parents map[element]cause) []WitnessStep {
	steps := []WitnessStep{}
	seen := map[element]bool{}
	for why := parents[e]; why.from != (element{}) && !seen[e]; why = parents[e] {
		seen[e] = true
		steps = append(steps, WitnessStep{FromKind: why.from.kind, From: why.from.id, Relation: why.relation, ToKind: e.kind, To: e.id})
		e = why.from
	}
	for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
		steps[i], steps[j] = steps[j], steps[i]
	}
	return steps
}

// changeClass marks what must change rather than only be read (DEC-023):
// changed definitions and files and the statements they reach directly; the
// direct consumers and requires targets of those statements; and the
// realizations, checks, files and owners of all of these. An edit that changes
// only how a definition is written (DEC-021) makes that definition, its model
// file, the definitions in that file and their owners change, and nothing
// further. Everything else in the impact, such as used context, transitive
// consumers and ancestor Managers, is context; under unknown scope every
// element is change. The class never removes anything from the impact.
func (r routing) changeClass() map[element]bool {
	change, rewrite := map[element]bool{}, map[element]bool{}
	carries := map[string]bool{"realizes": true, "exercises": true, "decides on": true, "owns": true, "maps to": true,
		"realized by": true, "exercised by": true, "expects": true, "checked by": true, "owned by": true, "written in": true, "declares": true}
	fromSeed := map[string]bool{"used by": true, "required by": true, "requires": true}
	rewriteCarries := map[string]bool{"written in": true, "declares": true, "owned by": true}
	for grew := true; grew; {
		grew = false
		for e, whys := range r.causes {
			for why := range whys {
				root := why.from == (element{})
				seed := why.from.kind == "statement" && r.seeds[why.from.id]
				if !change[e] && (root && why.reason != "rewritten" || change[why.from] && (carries[why.relation] || fromSeed[why.relation] && seed)) {
					change[e], grew = true, true
				}
				if !rewrite[e] && (root && why.reason == "rewritten" || rewrite[why.from] && rewriteCarries[why.relation]) {
					rewrite[e], grew = true, true
				}
			}
		}
	}
	for e := range rewrite {
		change[e] = true
	}
	return change
}

func sortElements(values []element) {
	sort.Slice(values, func(i, j int) bool { return lessElement(values[i], values[j]) })
}

func lessElement(a, b element) bool {
	if a.kind != b.kind {
		return a.kind < b.kind
	}
	return a.id < b.id
}
