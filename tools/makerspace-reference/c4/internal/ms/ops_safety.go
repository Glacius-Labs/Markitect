package ms

func safetyOps() []*opDef {
	return []*opDef{
		{name: "reportIncident", mutation: true, perms: "naaa",
			args: []argSpec{idRef("machineId", refMachine), enum("severity", "low", "medium", "high"),
				text("description")},
			run: func(c *call) (rec, string, *refusal) {
				i := &incident{id: c.s.nextID("inc"), machineID: c.str("machineId"), severity: c.str("severity"),
					description: c.str("description"), reportedBy: c.actor.id, reportedAt: c.at}
				c.s.incidents[i.id] = i
				c.s.incidentList = append(c.s.incidentList, i)
				return rec{"incident": incidentView(i)}, i.id, nil
			}},
		{name: "resolveIncident", mutation: true, perms: "naaa",
			args: []argSpec{idRef("incidentId", refIncident), text("resolution")},
			run: func(c *call) (rec, string, *refusal) {
				i := c.s.incidents[c.str("incidentId")]
				if i.resolved {
					return nil, "", refuse("invalid_state", "The incident is already resolved.")
				}
				i.resolved, i.resolution, i.resolvedAt = true, c.str("resolution"), c.at
				return rec{"incident": incidentView(i)}, i.id, nil
			}},
		{name: "lockMachine", mutation: true, perms: "naaa",
			args: []argSpec{idRef("incidentId", refIncident), stamp("start"), stamp("end")},
			run:  runLockMachine},
		{name: "listIncidents", perms: "naaa",
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
		{name: "listAuditEntries", perms: "nnan",
			run: func(c *call) (rec, string, *refusal) {
				out := make([]rec, 0, len(c.s.audit))
				for _, e := range c.s.audit {
					out = append(out, auditView(e))
				}
				return rec{"entries": out}, "", nil
			}},
	}
}

// lockoutCreditCents is the credit a member receives for each reservation a
// lockout cancels.
const lockoutCreditCents = 1000

func runLockMachine(c *call) (rec, string, *refusal) {
	inc := c.s.incidents[c.str("incidentId")]
	start, end := c.num("start"), c.num("end")
	if inc.resolved {
		return nil, "", refuse("invalid_state", "The incident is resolved.")
	}
	if end <= start {
		return nil, "", refuse("invalid_interval", "The end is not after the start.")
	}
	if start < c.at {
		return nil, "", refuse("starts_in_past", "The lockout starts before the request time.")
	}
	l := &lockout{id: c.s.nextID("lock"), incidentID: inc.id, machineID: inc.machineID, start: start, end: end}
	c.s.lockouts = append(c.s.lockouts, l)
	cancelled := []rec{}
	for _, r := range c.s.resList {
		if r.machineID != l.machineID || r.reportedStatus(c.at) != "reserved" || !overlaps(start, end, r.start, r.end) {
			continue
		}
		r.status = "cancelled"
		l.cancelledIDs = append(l.cancelledIDs, r.id)
		c.s.members[r.memberID].credit += lockoutCreditCents
		c.s.creditEvents = append(c.s.creditEvents, creditEvent{at: c.at, amount: lockoutCreditCents})
		cancelled = append(cancelled, reservationView(r, c.at))
	}
	return rec{"lockout": lockoutView(l), "cancelledReservations": cancelled}, l.id, nil
}
