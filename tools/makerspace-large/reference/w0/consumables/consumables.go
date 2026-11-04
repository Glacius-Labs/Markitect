// Package consumables holds how fast each machine uses up materials while it
// runs, and draws the stock for an ended session.
package consumables

import (
	"makerspace/billing"
	"makerspace/equipment"
	"makerspace/materials"
	"makerspace/policies"
)

// Module is the consumables module: it owns every consumption row.
type Module struct {
	equipment *equipment.Module
	materials *materials.Module
	billing   *billing.Module
	rows      map[string]map[string]int // machineID -> materialID -> unitsPerHour
}

// New returns the module in the initial state: no machine has a row.
func New(eq *equipment.Module, mat *materials.Module, bill *billing.Module) *Module {
	return &Module{equipment: eq, materials: mat, billing: bill, rows: map[string]map[string]int{}}
}

// Draw is the draw for one ended session at check-out: for each row of the
// machine, in ascending materialId order, it decreases the stock of the
// material by the units the metered minutes use, as far as the stock goes, and
// creates a material charge for the member when something was drawn. It
// answers the charges it created, which may be none.
func (m *Module) Draw(reservationID, memberID, machineID string, meteredMinutes, at int) []*billing.Charge {
	var out []*billing.Charge
	for _, mid := range policies.SortedKeys(m.rows[machineID]) {
		units := (meteredMinutes*m.rows[machineID][mid] + 59) / 60
		drawn := m.materials.Draw(mid, units)
		if drawn > 0 {
			out = append(out, m.billing.AddCharge(memberID, "material", reservationID, drawn,
				m.materials.UnitPrice(mid), 0, at))
		}
	}
	return out
}

func rowView(machineID, materialID string, unitsPerHour int) policies.Rec {
	return policies.Rec{"machineId": machineID, "materialId": materialID, "unitsPerHour": unitsPerHour}
}
