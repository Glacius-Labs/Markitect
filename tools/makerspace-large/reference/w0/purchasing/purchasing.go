// Package purchasing holds the suppliers and the orders of materials from
// them, and the operations on them.
package purchasing

import (
	"makerspace/ledger"
	"makerspace/materials"
	"makerspace/policies"
)

type supplier struct {
	id, name, status string
}

type line struct {
	materialID               string
	quantity, unitCost, rcvd int
}

type order struct {
	id, supplierID, status, createdBy string
	lines                             []*line
}

func (o *order) total() int {
	t := 0
	for _, l := range o.lines {
		t += l.quantity * l.unitCost
	}
	return t
}

func (o *order) lineOf(materialID string) *line {
	for _, l := range o.lines {
		if l.materialID == materialID {
			return l
		}
	}
	return nil
}

// Module is the purchasing module: it owns every supplier and order.
type Module struct {
	materials *materials.Module
	ledger    *ledger.Module
	ids       policies.IDs
	suppliers map[string]*supplier
	orders    map[string]*order
	orderList []*order
}

// New returns the module in the initial state.
func New(mat *materials.Module, led *ledger.Module) *Module {
	return &Module{materials: mat, ledger: led, suppliers: map[string]*supplier{}, orders: map[string]*order{}}
}

// SupplierExists reports whether the identifier names a supplier.
func (m *Module) SupplierExists(id string) bool { _, ok := m.suppliers[id]; return ok }

// OrderExists reports whether the identifier names an order.
func (m *Module) OrderExists(id string) bool { _, ok := m.orders[id]; return ok }

func supplierView(s *supplier) policies.Rec {
	return policies.Rec{"supplierId": s.id, "name": s.name, "status": s.status}
}

func orderView(o *order) policies.Rec {
	lines := []policies.Rec{}
	for _, l := range o.lines {
		lines = append(lines, policies.Rec{"materialId": l.materialID, "quantity": l.quantity,
			"unitCostCents": l.unitCost, "receivedQuantity": l.rcvd})
	}
	return policies.Rec{"orderId": o.id, "supplierId": o.supplierID, "status": o.status, "createdBy": o.createdBy,
		"totalCents": o.total(), "lines": lines}
}
