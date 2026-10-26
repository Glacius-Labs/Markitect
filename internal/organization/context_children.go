package organization

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

// ContextSummary compactly identifies one local Definition of a child area:
// identity, Kind, optional title and original source binding. It carries no
// body, fields, reasons or contracts and is not part of the selected set.
type ContextSummary struct {
	Identity model.Identity
	Kind     model.Identity
	Title    *string
	Source   model.Source
}

// ContextChild is one direct child Package with every local Definition of
// its whole subtree, in exact Source.Path order. An empty subtree has an
// explicit empty list. A child's handoff contains its descendants' work.
type ContextChild struct {
	Package     string
	Definitions []ContextSummary
}

// Children returns owned copies, one per direct child in topology order.
// A leaf and the zero selection return nil.
func (c ContextSelection) Children() []ContextChild {
	if len(c.children) == 0 {
		return nil
	}
	result := make([]ContextChild, len(c.children))
	for i, child := range c.children {
		result[i] = ContextChild{Package: child.Package, Definitions: make([]ContextSummary, len(child.Definitions))}
		for j, summary := range child.Definitions {
			if summary.Title != nil {
				title := *summary.Title
				summary.Title = &title
			}
			result[i].Definitions[j] = summary
		}
	}
	return result
}

func contextChildren(manager Manager, local []model.Definition) []ContextChild {
	children := make([]ContextChild, len(manager.Children))
	for i, child := range manager.Children {
		children[i] = ContextChild{Package: child, Definitions: []ContextSummary{}}
		for _, definition := range local {
			if pkg := definition.Identity.Package; pkg == child || strings.HasPrefix(pkg, child+"/") {
				summary := ContextSummary{Identity: definition.Identity, Kind: definition.Kind, Source: definition.Source}
				if definition.Title != nil {
					title := *definition.Title
					summary.Title = &title
				}
				children[i].Definitions = append(children[i].Definitions, summary)
			}
		}
		slices.SortFunc(children[i].Definitions, func(a, z ContextSummary) int {
			return cmp.Or(cmp.Compare(a.Source.Path, z.Source.Path), compareContextIDs(a.Identity, z.Identity))
		})
	}
	return children
}
