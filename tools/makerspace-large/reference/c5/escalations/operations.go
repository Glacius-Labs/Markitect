package escalations

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "setEscalationPolicy", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("familyId", m.equipment.FamilyExists),
				policies.Enum("minSeverity", "low", "medium", "high")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				fid := c.Str("familyId")
				m.policies[fid] = c.Str("minSeverity")
				return policies.Rec{"policy": policies.Rec{"familyId": fid, "minSeverity": m.policies[fid]}}, fid, nil
			}},
		{Name: "escalateIncident", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("incidentId", m.safety.IncidentExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				inc := m.safety.Incident(c.Str("incidentId"))
				if inc.Resolved {
					return nil, "", policies.Refuse("invalid_state", "The incident is resolved.")
				}
				if m.byIncident[inc.ID] != nil {
					return nil, "", policies.Refuse("already_escalated", "The incident has been escalated before.")
				}
				family := m.equipment.Machine(inc.MachineID).FamilyID
				minSev, ok := m.policies[family]
				if !ok {
					return nil, "", policies.Refuse("no_policy", "The family has no escalation policy.")
				}
				if severityRank[inc.Severity] < severityRank[minSev] {
					return nil, "", policies.Refuse("below_policy", "The severity is below the policy.")
				}
				wo := m.workorders.OpenLinked(inc.MachineID, "Escalated incident "+inc.ID, c.ActorID, inc.ID, c.At)
				cancelled := []string{}
				for _, r := range m.bookings.UpcomingReservations(inc.MachineID, c.At) {
					if m.bookings.CancelReserved(r.ID, c.At) {
						m.notices.Post(r.MemberID, "cancellation", r.ID, c.At)
						cancelled = append(cancelled, r.ID)
					}
				}
				e := &escalation{id: m.ids.Next("esc"), incidentID: inc.ID, machineID: inc.MachineID,
					workOrderID: wo.ID, cancelled: cancelled, at: c.At}
				m.byIncident[inc.ID] = e
				m.list = append(m.list, e)
				return policies.Rec{"escalation": view(e)}, e.id, nil
			}},
		{Name: "listEscalations", Perms: "naa",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				out := []policies.Rec{}
				for _, e := range m.list {
					out = append(out, view(e))
				}
				return policies.Rec{"escalations": out}, "", nil
			}},
	}
}
