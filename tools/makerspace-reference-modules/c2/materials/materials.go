// Package materials holds the shop: the materials with their stock and price,
// the purchases, and the operations on them.
package materials

import (
	"makerspace/billing"
	"makerspace/members"
	"makerspace/policies"
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

// Module is the materials module: it owns every material and purchase.
type Module struct {
	members   *members.Module
	billing   *billing.Module
	ids       policies.IDs
	materials map[string]*material
	purchases []*purchase
}

// New returns the module in the initial state.
func New(mem *members.Module, bill *billing.Module) *Module {
	return &Module{members: mem, billing: bill, materials: map[string]*material{}}
}

// MaterialExists reports whether the identifier names a material.
func (s *Module) MaterialExists(id string) bool { _, ok := s.materials[id]; return ok }
