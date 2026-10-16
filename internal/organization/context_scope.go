package organization

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func (b *contextBuilder) scoped(topology Topology, local []model.Definition, path string, selectors []Setting[ScopeRule]) error {
	seen := make(map[ScopeRule]bool)
	validation := settingsValidation{files: make(map[string]snapshot.ContentID), fields: make(map[SettingSource]any)}
	for _, selector := range selectors {
		if err := validation.field(selector.Source, selector.Value); err != nil {
			return err
		}
		if seen[selector.Value] {
			return fmt.Errorf("duplicate modeled scope selector %+v", selector.Value)
		}
		seen[selector.Value] = true
		name, err := b.scopeField(selector.Value)
		if err != nil {
			return err
		}
		for _, definition := range local {
			if definition.Kind != selector.Value.Kind {
				continue
			}
			value, present, err := contextField(definition, name)
			if err != nil {
				return err
			}
			if !present {
				continue
			}
			values, list := value.Items()
			if !list {
				values = []model.Value{value}
			}
			for i, item := range values {
				cover, text := item.Text()
				if _, found := topology.Manager(cover); !text || !found {
					return fmt.Errorf("invalid modeled scope Package %q in %+v", cover, definition.Identity)
				}
				if cover != Root && cover != path && !strings.HasPrefix(path, cover+"/") {
					continue
				}
				at := "/spec/" + contextPointer(name)
				if list {
					at += "/" + strconv.Itoa(i)
				}
				source := selector.Source
				b.add(definition.Identity, ContextReason{Kind: ModeledScope, From: definition.Identity, To: definition.Identity, At: at, Policy: &source, ScopePackage: cover})
			}
		}
	}
	return nil
}

func (b *contextBuilder) scopeField(rule ScopeRule) (string, error) {
	kind, found := b.index[rule.Kind]
	if !found || kind.Definition.Kind != contextCoreKind {
		return "", fmt.Errorf("scope selector has no bound Kind %+v", rule.Kind)
	}
	properties, err := contextProperties(kind.Definition)
	if err != nil {
		return "", err
	}
	declared := false
	for _, id := range properties {
		declared = declared || id == rule.Property
	}
	property, found := b.index[rule.Property]
	if !declared || !found || property.Definition.Kind != contextCoreProperty {
		return "", fmt.Errorf("scope Property %+v is not declared by Kind %+v", rule.Property, rule.Kind)
	}
	typeValue, _, err := contextField(property.Definition, "type")
	typeName, text := typeValue.Text()
	if err != nil || !text || typeName != "string" && typeName != "enum" {
		return "", fmt.Errorf("scope Property requires textual Package values")
	}
	nameValue, present, err := contextField(property.Definition, "name")
	if err != nil {
		return "", err
	}
	if present {
		name, text := nameValue.Text()
		if !text || name == "" {
			return "", fmt.Errorf("invalid bound scope field name")
		}
		return name, nil
	}
	return rule.Property.ID, nil
}
