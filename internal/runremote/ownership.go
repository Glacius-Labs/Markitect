package runremote

import (
	"slices"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

// Ownership answers which Package owns a repository file, with the
// registry's own pattern rules (docs/guides/model-language.md): the most
// specific matching pattern wins, Root owns what no pattern claims.
type Ownership struct {
	patterns map[string][]realization.Pattern
}

// NewOwnership reads the Managers' patterns of a plan.
func NewOwnership(p *Plan) Ownership {
	o := Ownership{patterns: map[string][]realization.Pattern{}}
	for _, m := range p.Managers {
		for _, text := range m.Patterns {
			o.patterns[m.ID] = append(o.patterns[m.ID], realization.Pattern{Text: text})
		}
	}
	return o
}

// Owner returns the Package that owns file, and the Packages that claim it
// equally when two are tied, which the compiler refuses.
func (o Ownership) Owner(file string) (string, []string) {
	best, score := model.RootPackage, -1
	var tied []string
	ids := make([]string, 0, len(o.patterns))
	for id := range o.patterns {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, p := range o.patterns[id] {
			if !p.Match(file) {
				continue
			}
			switch s := p.Specificity(); {
			case s > score:
				best, score, tied = id, s, nil
			case s == score && id != best && !slices.Contains(tied, id):
				tied = append(tied, id)
			}
		}
	}
	if len(tied) > 0 {
		return best, append([]string{best}, tied...)
	}
	return best, nil
}
