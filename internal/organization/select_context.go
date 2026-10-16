package organization

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

type contextBuilder struct {
	index    map[model.Identity]ContextDefinition
	selected map[model.Identity]ContextDefinition
	queue    []model.Identity
}

// SelectContext consumes only an overall Passed compiler result. An isolated
// passed root beside an overall Failed/NotCompleted supplied set is insufficient:
// only the overall proof establishes unique populated-Package/Definition owners
// across all supplied models. The exact global identity index locates already
// bound targets; mere delivery or a definitions-free Package prefix never grants
// visibility or identifies an owner. Foreign models create no Manager slots.
//
// Own Definitions anchor direct outgoing spec/body edges and direct LOCAL
// incoming edges. Connected/scoped Definitions get type contracts, but their
// further business edges never expand the selection. Scope is local and explicit.
// Coordinates are optional provenance, not a content-success gate. No source is
// acquired or parsed, no context is rendered, and every error returns zero.
func SelectContext(compiled model.Result, packagePath string, scope []Setting[ScopeRule]) (ContextSelection, error) {
	topology, err := BuildTopology(compiled)
	if err != nil {
		return ContextSelection{}, err
	}
	if _, found := topology.Manager(packagePath); !found {
		return ContextSelection{}, fmt.Errorf("context requires an exact local Manager: %q", packagePath)
	}
	b := contextBuilder{index: make(map[model.Identity]ContextDefinition), selected: make(map[model.Identity]ContextDefinition)}
	var local []model.Definition
	owners := make(map[string]bool)
	for _, owner := range compiled.Models {
		if owner.Status != model.Passed || owner.ID == "" || owner.Digest == "" || owners[owner.ID] {
			return ContextSelection{}, fmt.Errorf("inconsistent passed model owner %q", owner.ID)
		}
		owners[owner.ID] = true
		for _, definition := range owner.Definitions {
			if definition.ModelID != owner.ID || definition.Identity.Package == "" || definition.Identity.ID == "" {
				return ContextSelection{}, fmt.Errorf("inconsistent Definition owner/identity: %+v", definition.Identity)
			}
			if _, found := b.index[definition.Identity]; found {
				return ContextSelection{}, fmt.Errorf("duplicate global Definition %+v", definition.Identity)
			}
			b.index[definition.Identity] = ContextDefinition{Definition: definition, ModelDigest: owner.Digest}
		}
		if owner.Local {
			local = owner.Definitions
		}
	}
	own := make(map[model.Identity]bool)
	for _, definition := range local {
		if definition.Identity.Package == packagePath {
			own[definition.Identity] = true
			b.add(definition.Identity, ContextReason{Kind: OwnDefinition, From: definition.Identity, To: definition.Identity})
		}
	}
	for _, definition := range local {
		if err := walkContextEdges(definition, func(target model.Identity, at string) error {
			if _, found := b.index[target]; !found {
				return fmt.Errorf("missing bound target %+v from %+v at %s", target, definition.Identity, at)
			}
			if own[definition.Identity] {
				b.add(target, ContextReason{Kind: OutgoingReference, From: definition.Identity, To: target, At: at})
			}
			if own[target] {
				b.add(definition.Identity, ContextReason{Kind: IncomingReference, From: definition.Identity, To: target, At: at})
			}
			return nil
		}); err != nil {
			return ContextSelection{}, err
		}
	}
	if err := b.scoped(topology, local, packagePath, scope); err != nil {
		return ContextSelection{}, err
	}
	for i := 0; i < len(b.queue); i++ {
		if err := b.contracts(b.index[b.queue[i]].Definition); err != nil {
			return ContextSelection{}, err
		}
	}
	result := ContextSelection{topology: topology, packagePath: packagePath, modelID: topology.ModelID(), modelDigest: topology.ModelDigest(), index: make(map[model.Identity]int)}
	for _, definition := range b.selected {
		if definition.Definition.Source.Path == "" || definition.Definition.Source.Digest == "" {
			return ContextSelection{}, fmt.Errorf("missing original source binding for %+v", definition.Definition.Identity)
		}
		slices.SortFunc(definition.Reasons, compareContextReasons)
		result.definitions = append(result.definitions, cloneContextDefinition(definition))
	}
	slices.SortFunc(result.definitions, func(a, z ContextDefinition) int {
		if own[a.Definition.Identity] != own[z.Definition.Identity] {
			if own[a.Definition.Identity] {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(a.Definition.Source.Path, z.Definition.Source.Path), cmp.Compare(a.Definition.ModelID, z.Definition.ModelID), compareContextIDs(a.Definition.Identity, z.Definition.Identity))
	})
	for i, definition := range result.definitions {
		result.index[definition.Definition.Identity] = i
	}
	return result, nil
}

func (b *contextBuilder) add(id model.Identity, reason ContextReason) {
	entry, found := b.selected[id]
	if !found {
		entry = b.index[id]
		b.queue = append(b.queue, id)
	}
	for _, previous := range entry.Reasons {
		if compareContextReasons(previous, reason) == 0 {
			return
		}
	}
	entry.Reasons = append(entry.Reasons, reason)
	b.selected[id] = entry
}

func compareContextIDs(a, b model.Identity) int {
	return cmp.Or(cmp.Compare(a.Package, b.Package), cmp.Compare(a.ID, b.ID))
}

func compareContextReasons(a, b ContextReason) int {
	policy := func(r ContextReason) string {
		if r.Policy == nil {
			return ""
		}
		return r.Policy.File + "\x00" + r.Policy.ContentID.String() + "\x00" + r.Policy.Field
	}
	return cmp.Or(cmp.Compare(a.Kind, b.Kind), compareContextIDs(a.From, b.From), compareContextIDs(a.To, b.To), cmp.Compare(a.At, b.At), cmp.Compare(a.ScopePackage, b.ScopePackage), cmp.Compare(policy(a), policy(b)))
}

func contextPointer(field string) string {
	return strings.ReplaceAll(strings.ReplaceAll(field, "~", "~0"), "/", "~1")
}

func walkContextEdges(d model.Definition, visit func(model.Identity, string) error) error {
	var walk func(model.Value, string) error
	walk = func(value model.Value, at string) error {
		switch value.Kind() {
		case model.ReferenceValue:
			target, _ := value.Reference()
			return visit(target, at)
		case model.ObjectValue:
			fields, _ := value.Fields()
			for _, field := range fields {
				if err := walk(field.Value, at+"/"+contextPointer(field.Name)); err != nil {
					return err
				}
			}
		case model.ListValue:
			items, _ := value.Items()
			for i, item := range items {
				if err := walk(item, at+"/"+strconv.Itoa(i)); err != nil {
					return err
				}
			}
		case model.StringValue, model.EnumValue, model.BooleanValue, model.IntegerValue, model.NumberValue:
		default:
			return fmt.Errorf("invalid compiled value in %+v at %s", d.Identity, at)
		}
		return nil
	}
	for _, field := range d.Spec {
		if err := walk(field.Value, "/spec/"+contextPointer(field.Name)); err != nil {
			return err
		}
	}
	for i, link := range d.BodyLinks {
		if err := visit(link.Target, "/bodyLinks/"+strconv.Itoa(i)); err != nil {
			return err
		}
	}
	return nil
}
