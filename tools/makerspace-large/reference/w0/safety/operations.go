package safety

import "makerspace/policies"

// Operations lists the operations of the module.
func (s *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "reportIncident", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", s.equipment.MachineExists), policies.Enum("severity", "low", "medium", "high"),
				policies.Text("description")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				i := &Incident{ID: s.ids.Next("inc"), MachineID: c.Str("machineId"), Severity: c.Str("severity"),
					Description: c.Str("description"), ReportedBy: c.ActorID, ReportedAt: c.At}
				s.incidents[i.ID] = i
				s.incidentList = append(s.incidentList, i)
				return policies.Rec{"incident": incidentView(i)}, i.ID, nil
			}},
		{Name: "resolveIncident", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("incidentId", s.IncidentExists), policies.Text("resolution")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				i := s.incidents[c.Str("incidentId")]
				if i.Resolved {
					return nil, "", policies.Refuse("invalid_state", "The incident is already resolved.")
				}
				i.Resolved, i.Resolution, i.ResolvedAt = true, c.Str("resolution"), c.At
				return policies.Rec{"incident": incidentView(i)}, i.ID, nil
			}},
		{Name: "listIncidents", Perms: "naa",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				out := make([]policies.Rec, 0, len(s.incidentList))
				for _, i := range s.incidentList {
					out = append(out, incidentView(i))
				}
				return policies.Rec{"incidents": out}, "", nil
			}},
	}
}
