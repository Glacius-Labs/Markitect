package volunteering

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "setVolunteerRate", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.Integer("centsPerHour", 0, 100000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				m.rate = c.Num("centsPerHour")
				return policies.Rec{"centsPerHour": m.rate}, "rate", nil
			}},
		{Name: "logVolunteerTime", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.Date("date"),
				policies.Integer("minutes", 1, 720), policies.Text("note")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				if c.Num("date") > policies.DayOf(c.At) {
					return nil, "", policies.Refuse("date_in_future", "The date is after the date of the request time.")
				}
				e := &Entry{ID: m.ids.Next("vol"), MemberID: c.Str("memberId"), Date: c.Num("date"),
					Minutes: c.Num("minutes"), Note: c.Str("note"), Status: "pending"}
				m.byID[e.ID] = e
				m.list = append(m.list, e)
				return policies.Rec{"entry": entryView(e)}, e.ID, nil
			}},
		{Name: "decideVolunteerTime", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.OwnedIDRef("entryId", m.EntryExists, m.EntryOwner),
				policies.Enum("decision", "approved", "rejected")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				e, decision := m.byID[c.Str("entryId")], c.Str("decision")
				if e.Status != "pending" {
					return nil, "", policies.Refuse("invalid_state", "The entry is not pending.")
				}
				if c.ActorRole == "staff" && !m.staffing.OnShift(c.ActorID, c.At) {
					return nil, "", policies.Refuse("approver_not_on_shift", "The actor has no scheduled shift at the request time.")
				}
				if decision == "approved" {
					month := policies.MonthOf(e.Date * policies.MinPerDay)
					if m.approvedMinutes(e.MemberID, month)+e.Minutes > monthlyCap {
						return nil, "", policies.Refuse("volunteer_cap_reached", "The member's approved minutes of the month would pass %d.", monthlyCap)
					}
					credit := (e.Minutes*m.rate + 30) / 60
					e.Credit = &credit
					m.billing.AddCredit(e.MemberID, credit, "expenseVolunteer", e.ID, c.At)
				}
				e.Status = decision
				return policies.Rec{"entry": entryView(e)}, e.ID, nil
			}},
		{Name: "listVolunteerTime", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, e := range m.list {
					if e.MemberID == mid {
						out = append(out, entryView(e))
					}
				}
				return policies.Rec{"memberId": mid, "entries": out}, "", nil
			}},
	}
}
