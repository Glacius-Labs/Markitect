// Package ppe holds the personal protective equipment the workshop lends to
// members, the families' requirements and what each member holds, and the
// operations on them.
package ppe

import (
	"makerspace/equipment"
	"makerspace/members"
	"makerspace/policies"
)

type item struct {
	id, name string
	stock    int
}

// Module is the ppe module: it owns every item, requirement and holding.
type Module struct {
	members   *members.Module
	equipment *equipment.Module
	items     map[string]*item
	required  map[string]map[string]bool // familyID -> itemID
	held      map[string]map[string]bool // memberID -> itemID
}

// New returns the module in the initial state.
func New(mem *members.Module, eq *equipment.Module) *Module {
	return &Module{members: mem, equipment: eq, items: map[string]*item{},
		required: map[string]map[string]bool{}, held: map[string]map[string]bool{}}
}

// ItemExists reports whether the identifier names an item.
func (m *Module) ItemExists(id string) bool { _, ok := m.items[id]; return ok }

// Required lists the items the family requires, ascending.
func (m *Module) Required(familyID string) []string { return policies.SortedKeys(m.required[familyID]) }

// Holds reports whether the member holds the item.
func (m *Module) Holds(memberID, itemID string) bool { return m.held[memberID][itemID] }

// Missing reports whether the member is refused ppe_missing for the family:
// the member does not hold every item the family requires.
func (m *Module) Missing(memberID, familyID string) bool {
	for itemID := range m.required[familyID] {
		if !m.held[memberID][itemID] {
			return true
		}
	}
	return false
}

func itemView(i *item) policies.Rec {
	return policies.Rec{"itemId": i.id, "name": i.name, "stock": i.stock}
}
