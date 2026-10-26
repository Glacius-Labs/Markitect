package organization

import (
	"fmt"
	"slices"
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
		name, expectList, err := b.scopeField(selector.Value)
		if err != nil {
			return err
		}
		for _, definition := range local {
			// A selector on an interface Kind also selects Definitions of every
			// Kind that implements it (language contract §3.3).
			lineage, err := b.kindLineage(definition.Kind)
			if err != nil {
				return err
			}
			if !slices.Contains(lineage, selector.Value.Kind) {
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
			if list != expectList {
				return fmt.Errorf("scope field %q contradicts its bound cardinality", name)
			}
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

func (b *contextBuilder) scopeField(rule ScopeRule) (string, bool, error) {
	kind, found := b.index[rule.Kind]
	if !found || kind.Definition.Kind != contextCoreKind {
		return "", false, fmt.Errorf("scope selector has no bound Kind %+v", rule.Kind)
	}
	// The scope field is declared by the Kind or by an interface it implements.
	lineage, err := b.kindLineage(rule.Kind)
	if err != nil {
		return "", false, err
	}
	declared := false
	for _, owner := range lineage {
		properties, err := contextProperties(b.index[owner].Definition)
		if err != nil {
			return "", false, err
		}
		for _, id := range properties {
			declared = declared || id == rule.Property
		}
	}
	property, found := b.index[rule.Property]
	if !declared || !found || property.Definition.Kind != contextCoreProperty {
		return "", false, fmt.Errorf("scope Property %+v is not declared by Kind %+v", rule.Property, rule.Kind)
	}
	typeValue, _, err := contextField(property.Definition, "valueType")
	typeName, text := typeValue.Text()
	if err != nil || !text || typeName != "string" && typeName != "enum" {
		return "", false, fmt.Errorf("scope Property requires textual Package values")
	}
	count, present, err := contextField(property.Definition, "maxCount")
	maximum, integer := count.Integer()
	if err != nil || !present || !integer {
		return "", false, fmt.Errorf("scope Property requires bound cardinality")
	}
	list := maximum != "1"
	nameValue, present, err := contextField(property.Definition, "name")
	if err != nil {
		return "", false, err
	}
	if present {
		name, text := nameValue.Text()
		if !text || name == "" {
			return "", false, fmt.Errorf("invalid bound scope field name")
		}
		return name, list, nil
	}
	return rule.Property.ID, list, nil
}
