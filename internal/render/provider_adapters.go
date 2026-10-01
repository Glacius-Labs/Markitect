package render

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

// renderConfiguredAdapters adds Project-owned entrypoints after resource adapters.
func renderConfiguredAdapters(g *core.Graph, targets map[string]bool, adapters *core.ProviderAdapters, outputs map[string][]byte, owners map[string]map[string]bool) error {
	if adapters == nil {
		return nil
	}
	if targets["claude"] {
		names := make([]string, 0, len(adapters.RuleSources))
		for name := range adapters.RuleSources {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			target := ".claude/rules/" + name + ".md"
			var b strings.Builder
			b.WriteString("<!-- " + Marker + "; source: markitect.yaml -->\n# " + title(name) + "\n\nRead the linked canonical guidance before acting; it owns the operative requirements.\n\n")
			for _, source := range adapters.RuleSources[name] {
				if err := validRepoPath(source); err != nil {
					return fmt.Errorf("ruleSources.%s: %w", name, err)
				}
				label := strings.TrimSuffix(path.Base(source), path.Ext(source))
				b.WriteString("- [" + title(label) + "](" + relative(target, source) + ")\n")
			}
			if err := addOwned(outputs, owners, target, []byte(b.String()), g.Project.Key()); err != nil {
				return err
			}
		}
	}
	if adapters.RoleRegister != "" && (targets["codex"] || targets["claude"]) {
		if err := validRepoPath(adapters.RoleRegister); err != nil {
			return fmt.Errorf("roleRegister: %w", err)
		}
		var roleTargets []string
		if targets["codex"] {
			roleTargets = append(roleTargets, ".agents/roles.md")
		}
		if targets["claude"] {
			roleTargets = append(roleTargets, ".claude/roles.md")
		}
		for _, target := range roleTargets {
			body := "<!-- " + Marker + "; source: markitect.yaml -->\n# Repository role assignments\n\nThe canonical person-to-scope assignments are maintained in [the shared role register](" + relative(target, adapters.RoleRegister) + ").\n"
			if err := addOwned(outputs, owners, target, []byte(body), g.Project.Key()); err != nil {
				return err
			}
		}
	}
	return nil
}
