package workorders

import (
	"makerspace/parts"
	"makerspace/policies"
)

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "openWorkOrder", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists), policies.Text("summary")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				w := m.open(c.Str("machineId"), c.Str("summary"), c.ActorID, c.At)
				return policies.Rec{"workOrder": View(w)}, w.ID, nil
			}},
		{Name: "linkIncident", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("workOrderId", m.WorkOrderExists),
				policies.IDRef("incidentId", m.safety.IncidentExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				w := m.byID[c.Str("workOrderId")]
				inc := m.safety.Incident(c.Str("incidentId"))
				if w.Closed {
					return nil, "", policies.Refuse("invalid_state", "The work order is closed.")
				}
				if inc.MachineID != w.MachineID {
					return nil, "", policies.Refuse("incident_other_machine", "The incident is about another machine.")
				}
				if w.IncidentID != "" {
					return nil, "", policies.Refuse("already_linked", "The work order is already linked to an incident.")
				}
				w.IncidentID = inc.ID
				return policies.Rec{"workOrder": View(w)}, w.ID, nil
			}},
		{Name: "useParts", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("workOrderId", m.WorkOrderExists),
				policies.IDRef("partId", m.parts.PartExists), policies.Integer("quantity", 1, 1000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				w := m.byID[c.Str("workOrderId")]
				p := m.parts.Part(c.Str("partId"))
				qty := c.Num("quantity")
				if w.Closed {
					return nil, "", policies.Refuse("invalid_state", "The work order is closed.")
				}
				if p.FamilyID != m.equipment.Machine(w.MachineID).FamilyID {
					return nil, "", policies.Refuse("part_wrong_family", "The part fits another family.")
				}
				if p.Stock < qty {
					return nil, "", policies.Refuse("insufficient_parts", "The part stock is lower than the quantity.")
				}
				m.parts.DecreaseStock(p.ID, qty)
				w.PartsCost += qty * p.UnitCost
				return policies.Rec{"workOrder": View(w), "part": parts.PartView(m.parts.Part(p.ID))}, w.ID, nil
			}},
		{Name: "closeWorkOrder", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("workOrderId", m.WorkOrderExists), policies.Text("note")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				w := m.byID[c.Str("workOrderId")]
				if w.Closed {
					return nil, "", policies.Refuse("invalid_state", "The work order is already closed.")
				}
				if w.IncidentID != "" && !m.safety.Incident(w.IncidentID).Resolved {
					return nil, "", policies.Refuse("incident_open", "The linked incident is not resolved.")
				}
				w.Closed = true
				t, note := c.At, c.Str("note")
				w.ClosedAt, w.Note = &t, &note
				m.lockouts.Release(w.LockoutID, c.At)
				return policies.Rec{"workOrder": View(w)}, w.ID, nil
			}},
		{Name: "listWorkOrders", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("machineId")
				out := []policies.Rec{}
				for _, w := range m.list {
					if w.MachineID == mid {
						out = append(out, View(w))
					}
				}
				return policies.Rec{"machineId": mid, "workOrders": out}, "", nil
			}},
	}
}
