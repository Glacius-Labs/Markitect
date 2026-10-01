package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func renderRules(g *core.Graph) (map[string][]byte, map[string][]string, error) {
	out := map[string][]byte{}
	owners := map[string][]string{}
	if g.Project == nil || g.Project.Spec.RuleAdapters == nil {
		return out, owners, nil
	}
	names := make([]string, 0, len(g.Project.Spec.RuleAdapters))
	for name := range g.Project.Spec.RuleAdapters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		refs := g.Project.Spec.RuleAdapters[name]
		var b strings.Builder
		b.WriteString("<!-- " + Marker + " -->\n# " + title(name) + "\n\n")
		for _, ref := range refs {
			if ref.Package != "" {
				return nil, nil, fmt.Errorf("external ruleAdapters are not supported: %q", ref.GraphKey("", "", "Rule"))
			}
			r := findResource(g, ref, "", "", "Rule")
			if r == nil {
				return nil, nil, fmt.Errorf("rule adapter %q references unresolved rule %q", name, ref.Name)
			}
			p := ".claude/rules/" + name + ".md"
			b.WriteString("- [" + r.Metadata.Name + "](" + relative(p, companionPath(r.Path)) + ")\n")
			owners[p] = append(owners[p], r.Key())
		}
		out[".claude/rules/"+name+".md"] = []byte(b.String())
	}
	for output := range owners {
		sort.Strings(owners[output])
	}
	return out, owners, nil
}
