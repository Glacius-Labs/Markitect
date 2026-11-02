package policies

import "strconv"

// IDs generates the identifiers of one module's records: one counter per
// prefix, starting at 1, never reused.
type IDs struct{ n map[string]int }

// Next consumes and returns the next identifier of a kind.
func (g *IDs) Next(prefix string) string {
	if g.n == nil {
		g.n = map[string]int{}
	}
	g.n[prefix]++
	return prefix + "-" + strconv.Itoa(g.n[prefix])
}
