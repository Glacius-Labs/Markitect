// Package materials holds the shop: the materials with their stock and price,
// the purchases, and the operations on them.
package materials

import (
	"makerspace/billing"
	"makerspace/members"
	"makerspace/policies"
	"makerspace/promotions"
)

type material struct {
	id, name    string
	unit, stock int
}

type purchase struct {
	id, memberID, materialID string
	quantity, unit, discount int
	amount, at               int
}

// Needs is what materials asks of the program without depending on the modules
// that know the answer: the protocol module supplies it from hazmat and
// stocktake.
type Needs struct {
	// HazardRefusal gives the refusal code of the hazard gate that applies to a
	// purchase of the quantity of the material by the member at the request
	// time at, or "" when none applies.
	HazardRefusal func(memberID, materialID string, quantity, at int) string
	// Counted reports whether the material has a count in an open stocktake.
	Counted func(materialID string) bool
}

// Module is the materials module: it owns every material and purchase.
type Module struct {
	needs      Needs
	members    *members.Module
	billing    *billing.Module
	promotions *promotions.Module
	ids        policies.IDs
	materials  map[string]*material
	purchases  []*purchase
	byID       map[string]*purchase
}

// New returns the module in the initial state.
func New(needs Needs, mem *members.Module, bill *billing.Module, prm *promotions.Module) *Module {
	return &Module{needs: needs, members: mem, billing: bill, promotions: prm,
		materials: map[string]*material{}, byID: map[string]*purchase{}}
}

// MaterialExists reports whether the identifier names a material.
func (s *Module) MaterialExists(id string) bool { _, ok := s.materials[id]; return ok }

// UnitPrice is the current unit price of the material in cents.
func (s *Module) UnitPrice(materialID string) int { return s.materials[materialID].unit }

// Stock is the current stock of the material.
func (s *Module) Stock(materialID string) int { return s.materials[materialID].stock }

// UnitsBought is the sum of quantity over the member's purchases of the
// material whose time lies in the month.
func (s *Module) UnitsBought(memberID, materialID string, month int) int {
	n := 0
	for _, p := range s.purchases {
		if p.memberID == memberID && p.materialID == materialID && policies.MonthOf(p.at) == month {
			n += p.quantity
		}
	}
	return n
}

// PurchaseExists reports whether the identifier names a purchase.
func (s *Module) PurchaseExists(id string) bool { return s.byID[id] != nil }

// PurchaseOf is the member and the amount in cents of the purchase. The last
// result is false when the identifier names no purchase.
func (s *Module) PurchaseOf(id string) (memberID string, amountCents int, ok bool) {
	p := s.byID[id]
	if p == nil {
		return "", 0, false
	}
	return p.memberID, p.amount, true
}

// AddStock increases the stock of the material by the quantity (received
// stock).
func (s *Module) AddStock(materialID string, quantity int) { s.materials[materialID].stock += quantity }

// SetStock sets the stock of the material to the number of units (stock from a
// count).
func (s *Module) SetStock(materialID string, units int) { s.materials[materialID].stock = units }

// Draw decreases the stock of the material by the smaller of the units and the
// stock and answers the number drawn, which may be 0.
func (s *Module) Draw(materialID string, units int) int {
	m := s.materials[materialID]
	drawn := min(units, m.stock)
	m.stock -= drawn
	return drawn
}

// Counted reports whether the material has a count in an open stocktake.
func (s *Module) Counted(materialID string) bool { return s.needs.Counted(materialID) }
