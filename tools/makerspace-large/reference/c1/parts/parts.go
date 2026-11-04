// Package parts holds the spare parts that repairs use up, and the operations
// on them.
package parts

import (
	"makerspace/equipment"
	"makerspace/policies"
)

// Part is a spare part that fits the machines of one tool family.
type Part struct {
	ID, Name, FamilyID string
	UnitCost, Stock    int
}

// Module is the parts module: it owns every part.
type Module struct {
	equipment *equipment.Module
	parts     map[string]*Part
}

// New returns the module in the initial state.
func New(eq *equipment.Module) *Module {
	return &Module{equipment: eq, parts: map[string]*Part{}}
}

// PartExists reports whether the identifier names a part.
func (m *Module) PartExists(id string) bool { _, ok := m.parts[id]; return ok }

// Part returns a copy of the part with its family, unit cost and stock, or nil.
func (m *Module) Part(id string) *Part {
	p := m.parts[id]
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

// DecreaseStock takes quantity units out of the part's stock. The caller has
// checked that the stock suffices.
func (m *Module) DecreaseStock(id string, quantity int) { m.parts[id].Stock -= quantity }

func partView(p *Part) policies.Rec {
	return policies.Rec{"partId": p.ID, "name": p.Name, "unitCostCents": p.UnitCost, "familyId": p.FamilyID,
		"stock": p.Stock}
}

// PartView is the part as it appears in results.
func PartView(p *Part) policies.Rec { return partView(p) }
