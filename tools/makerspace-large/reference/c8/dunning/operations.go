package dunning

import "makerspace/policies"

// run takes the steps that are due for every invoice overdue at the time.
func (m *Module) run(at int) []policies.Rec {
	actions := []policies.Rec{}
	for _, inv := range m.billing.OverdueInvoices(at) {
		days := (at - inv.DueAt) / policies.MinPerDay
		cs := m.byInv[inv.ID]
		// A case is created by the first step taken for the invoice.
		touch := func() *Case {
			if cs == nil {
				cs = &Case{ID: m.ids.Next("dun"), InvoiceID: inv.ID, MemberID: inv.MemberID}
				m.byInv[inv.ID] = cs
				m.list = append(m.list, cs)
			}
			return cs
		}
		stamp := at
		if days >= m.ladder.NoticeDays && (cs == nil || cs.NoticedAt == nil) {
			touch().NoticedAt = &stamp
			m.notices.Post(inv.MemberID, "dunning", inv.ID, at)
			actions = append(actions, policies.Rec{"invoiceId": inv.ID, "step": "notice"})
		}
		if days >= m.ladder.FeeDays && (cs == nil || cs.FeeChargedAt == nil) {
			touch().FeeChargedAt = &stamp
			m.billing.AddCharge(inv.MemberID, "lateFee", inv.ID, 1, m.ladder.FeeCents, 0, at)
			actions = append(actions, policies.Rec{"invoiceId": inv.ID, "step": "fee"})
		}
		if days >= m.ladder.SuspendDays && (cs == nil || cs.SuspendedAt == nil) {
			touch().SuspendedAt = &stamp
			m.members.SetStanding(inv.MemberID, "suspended")
			m.notices.Post(inv.MemberID, "suspension", inv.ID, at)
			actions = append(actions, policies.Rec{"invoiceId": inv.ID, "step": "suspend"})
		}
	}
	return actions
}

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "setDunningLadder", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.Integer("noticeDays", 1, 3650), policies.Integer("feeDays", 1, 3650),
				policies.Integer("feeCents", 1, 1000000), policies.Integer("suspendDays", 1, 3650)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				l := &Ladder{NoticeDays: c.Num("noticeDays"), FeeDays: c.Num("feeDays"), FeeCents: c.Num("feeCents"),
					SuspendDays: c.Num("suspendDays")}
				if !(l.NoticeDays < l.FeeDays && l.FeeDays < l.SuspendDays) {
					return nil, "", policies.Refuse("invalid_ladder", "The days must increase from notice to fee to suspension.")
				}
				m.ladder = l
				return policies.Rec{"ladder": ladderView(l)}, "ladder", nil
			}},
		{Name: "runDunning", Mutation: true, Perms: "naa",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				if m.ladder == nil {
					return nil, "", policies.Refuse("no_ladder", "No dunning ladder has been set.")
				}
				return policies.Rec{"actions": m.run(c.At)}, "dunning", nil
			}},
		{Name: "listDunningCases", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, cs := range m.list {
					if cs.MemberID == mid {
						out = append(out, caseView(cs))
					}
				}
				return policies.Rec{"memberId": mid, "cases": out}, "", nil
			}},
	}
}
