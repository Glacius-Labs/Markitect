package ms

func safetyOps() []*opDef {
	return []*opDef{
		{name: "reportIncident", mutation: true, perms: "naa",
			args: []argSpec{idRef("machineId", refMachine), enum("severity", "low", "medium", "high"),
				text("description")},
			run: func(c *call) (rec, string, *refusal) {
				i := &incident{id: c.s.nextID("inc"), machineID: c.str("machineId"), severity: c.str("severity"),
					description: c.str("description"), reportedBy: c.actor.id, reportedAt: c.at}
				c.s.incidents[i.id] = i
				c.s.incidentList = append(c.s.incidentList, i)
				return rec{"incident": incidentView(i)}, i.id, nil
			}},
		{name: "resolveIncident", mutation: true, perms: "naa",
			args: []argSpec{idRef("incidentId", refIncident), text("resolution")},
			run: func(c *call) (rec, string, *refusal) {
				i := c.s.incidents[c.str("incidentId")]
				if i.resolved {
					return nil, "", refuse("invalid_state", "The incident is already resolved.")
				}
				i.resolved, i.resolution, i.resolvedAt = true, c.str("resolution"), c.at
				return rec{"incident": incidentView(i)}, i.id, nil
			}},
		{name: "listIncidents", perms: "naa",
			run: func(c *call) (rec, string, *refusal) {
				out := make([]rec, 0, len(c.s.incidentList))
				for _, i := range c.s.incidentList {
					out = append(out, incidentView(i))
				}
				return rec{"incidents": out}, "", nil
			}},
	}
}

func auditOps() []*opDef {
	return []*opDef{
		{name: "listAuditEntries", perms: "nna",
			run: func(c *call) (rec, string, *refusal) {
				out := make([]rec, 0, len(c.s.audit))
				for _, e := range c.s.audit {
					out = append(out, auditView(e))
				}
				return rec{"entries": out}, "", nil
			}},
	}
}
