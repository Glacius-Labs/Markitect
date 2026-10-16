package organization

import (
	"fmt"
	"strconv"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

// Only the fixed v1 Core meta identities distinguish contract declarations.
// An ordinary data field named properties or target has no special effect.
var contextCoreKind = model.Identity{Package: "markitect/core", ID: "Kind"}
var contextCoreProperty = model.Identity{Package: "markitect/core", ID: "Property"}

func (b *contextBuilder) typedTarget(from model.Definition, target, expected model.Identity, kind ContextReasonKind, at string) error {
	entry, found := b.index[target]
	if !found || entry.Definition.Kind != expected {
		return fmt.Errorf("missing or inconsistent bound type contract %+v from %+v at %s", target, from.Identity, at)
	}
	b.add(target, ContextReason{Kind: kind, From: from.Identity, To: target, At: at})
	return nil
}

func (b *contextBuilder) contracts(definition model.Definition) error {
	if err := b.typedTarget(definition, definition.Kind, contextCoreKind, KindContract, "/kind"); err != nil {
		return err
	}
	if definition.Kind == contextCoreKind {
		return b.properties(definition)
	}
	if definition.Kind == contextCoreProperty {
		value, present, err := contextField(definition, "type")
		text, textual := value.Text()
		if err != nil || !present || !textual {
			return fmt.Errorf("missing bound Property type in %+v", definition.Identity)
		}
		switch text {
		case "object":
			return b.properties(definition)
		case "reference":
			value, present, err := contextField(definition, "target")
			target, reference := value.Reference()
			if err != nil || !present || !reference {
				return fmt.Errorf("missing bound Property target in %+v", definition.Identity)
			}
			return b.typedTarget(definition, target, contextCoreKind, KindContract, "/spec/target")
		case "string", "enum", "boolean", "integer", "number":
			return nil
		default:
			return fmt.Errorf("invalid bound Property type %q", text)
		}
	}
	return nil
}

func (b *contextBuilder) properties(definition model.Definition) error {
	ids, err := contextProperties(definition)
	if err != nil {
		return err
	}
	for i, id := range ids {
		if err := b.typedTarget(definition, id, contextCoreProperty, PropertyContract, "/spec/properties/"+strconv.Itoa(i)); err != nil {
			return err
		}
	}
	return nil
}

func contextProperties(definition model.Definition) ([]model.Identity, error) {
	value, present, err := contextField(definition, "properties")
	if err != nil || !present {
		return nil, err
	}
	items, list := value.Items()
	if !list {
		return nil, fmt.Errorf("inconsistent bound properties in %+v", definition.Identity)
	}
	ids := make([]model.Identity, len(items))
	for i, item := range items {
		id, reference := item.Reference()
		if !reference {
			return nil, fmt.Errorf("inconsistent bound Property reference in %+v", definition.Identity)
		}
		ids[i] = id
	}
	return ids, nil
}

func contextField(definition model.Definition, name string) (model.Value, bool, error) {
	var value model.Value
	found := false
	for _, field := range definition.Spec {
		if field.Name == name {
			if found {
				return model.Value{}, false, fmt.Errorf("duplicate bound field %q in %+v", name, definition.Identity)
			}
			value, found = field.Value, true
		}
	}
	return value, found, nil
}
