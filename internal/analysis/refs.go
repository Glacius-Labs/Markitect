// Package analysis answers questions about a loaded project without starting
// agents: who links to a Definition, the Package dependency graph, what a
// change reaches and each Manager's checklist, and the lint warnings that
// measure the model like code.
//
// The commands are described in docs/guides/model-language.md.
package analysis

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Ref is one place that names a Definition or a Package.
type Ref struct {
	From string `json:"from"` // Definition ID, or the realizing file for registry entries
	Via  string `json:"via"`  // "link", "kind" or "registry"
	File string `json:"file"` // where the reference is written
	Line int    `json:"line,omitempty"`
}

// Refs lists every Definition that links to target (by body link or as its
// kind) and every registry entry that realizes it. target is a Definition ID
// such as members/member or std/Rule, or a Package path.
func Refs(p *project.Project, target string) ([]Ref, error) {
	target = strings.TrimSuffix(strings.Trim(target, "[]"), ".md")
	_, isDef := p.Model.Definitions[target]
	_, isPkg := p.Model.Packages[target]
	if !isDef && (!isPkg || target == model.RootPackage) {
		return nil, fmt.Errorf("%q is neither a Definition nor a Package of the model", target)
	}
	var refs []Ref
	for _, id := range sortedIDs(p.Model) {
		d := p.Model.Definitions[id]
		for _, l := range d.Links {
			if l.Target != target {
				continue
			}
			via := "link"
			if l.Kind {
				via = "kind"
			}
			refs = append(refs, Ref{From: id, Via: via, File: d.File, Line: l.Line})
		}
	}
	for _, file := range sortedKeys(p.Registry.Entries) {
		e := p.Registry.Entries[file]
		for _, id := range e.Realizes {
			if id == target {
				refs = append(refs, Ref{From: file, Via: "registry", File: e.PageFile, Line: e.Line})
			}
		}
	}
	return refs, nil
}

func sortedIDs(m *model.Model) []string {
	ids := make([]string, 0, len(m.Definitions))
	for id := range m.Definitions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
