package signals

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"
)

// bitset is a set of link targets of one section name, by index.
type bitset []uint64

func (b bitset) and(o bitset) bitset {
	out := make(bitset, len(b))
	for i := range b {
		out[i] = b[i] & o[i]
	}
	return out
}

func (b bitset) count() int {
	n := 0
	for _, w := range b {
		n += bits.OnesCount64(w)
	}
	return n
}

// within reports whether every member of b is in o.
func (b bitset) within(o bitset) bool {
	for i := range b {
		if b[i]&^o[i] != 0 {
			return false
		}
	}
	return true
}

func (b bitset) key() string {
	var s strings.Builder
	for _, w := range b {
		fmt.Fprintf(&s, "%016x", w)
	}
	return s.String()
}

// linkGroups finds the groups of three or more link targets that stand
// together under the same section name in two or more documents. The groups
// are every intersection of the sections' target sets, closed under
// intersection, so a group shared by many documents is found even where each
// pair shares more. Of groups held by the same documents only the largest is
// listed.
func (a *Analysis) linkGroups() {
	type place struct {
		doc  string
		line int
		set  bitset
	}
	type section struct {
		index   map[string]int
		targets []string
		raw     []map[string]bool
		places  []place
	}
	sections := map[string]*section{}
	var names []string
	for _, d := range a.Spec.Documents {
		for _, s := range d.Sections {
			targets := map[string]bool{}
			for _, l := range d.Links {
				if l.Section == s.Heading {
					targets[l.Target] = true
				}
			}
			if len(targets) < 3 {
				continue
			}
			name := strings.ToLower(s.Heading)
			sec := sections[name]
			if sec == nil {
				sec = &section{index: map[string]int{}}
				sections[name] = sec
				names = append(names, name)
			}
			for _, t := range sortedKeys(targets) {
				if _, ok := sec.index[t]; !ok {
					sec.index[t] = len(sec.targets)
					sec.targets = append(sec.targets, t)
				}
			}
			sec.raw = append(sec.raw, targets)
			sec.places = append(sec.places, place{doc: d.Path, line: s.Line})
		}
	}
	sort.Strings(names)
	for _, name := range names {
		sec := sections[name]
		words := (len(sec.targets) + 63) / 64
		for i, targets := range sec.raw {
			set := make(bitset, words)
			for t := range targets {
				k := sec.index[t]
				set[k/64] |= 1 << (k % 64)
			}
			sec.places[i].set = set
		}
		// Close the places' sets under intersection.
		seen := map[string]bool{}
		var family []bitset
		for _, p := range sec.places {
			if !seen[p.set.key()] {
				seen[p.set.key()] = true
				family = append(family, p.set)
			}
		}
		for i := 0; i < len(family); i++ {
			for _, p := range sec.places {
				s := family[i].and(p.set)
				if s.count() < 3 || seen[s.key()] {
					continue
				}
				seen[s.key()] = true
				family = append(family, s)
			}
		}
		type group struct {
			set     bitset
			size    int
			holders []int
		}
		var groups []group
		for _, s := range family {
			g := group{set: s, size: s.count()}
			for k, p := range sec.places {
				if s.within(p.set) {
					g.holders = append(g.holders, k)
				}
			}
			if len(g.holders) >= 2 {
				groups = append(groups, g)
			}
		}
		for _, g := range groups {
			maximal := true
			for _, h := range groups {
				if h.size > g.size && len(h.holders) == len(g.holders) && g.set.within(h.set) {
					maximal = false
					break
				}
			}
			if !maximal {
				continue
			}
			var docs, targets []string
			for _, k := range g.holders {
				docs = append(docs, sec.places[k].doc)
			}
			for k, t := range sec.targets {
				if g.set[k/64]&(1<<(k%64)) != 0 {
					targets = append(targets, t)
				}
			}
			sort.Strings(targets)
			first := sec.places[g.holders[0]]
			heading := sectionHeading(a.Spec, first.doc, first.line)
			a.add(Finding{Signal: LinkGroup, Path: first.doc, Line: first.line, Docs: append(append([]string(nil), docs...), targets...),
				rank:   float64(len(docs)) + float64(len(targets))/100,
				Detail: fmt.Sprintf("section %q links the same %d documents in %s: %s (links: %s)", heading, len(targets), plural(len(docs), "document"), strings.Join(docs, ", "), strings.Join(targets, ", "))})
		}
	}
}
