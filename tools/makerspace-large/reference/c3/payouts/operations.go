package payouts

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	userExists := func(id string) bool { return m.members.User(id) != nil }
	return []*policies.Operation{
		{Name: "setInstructorRate", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("userId", userExists), policies.Integer("centsPerHour", 1, 1000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				uid := c.Str("userId")
				if !m.instructors.HoldsAny(uid) {
					return nil, "", policies.Refuse("not_instructor", "The user holds no qualification.")
				}
				m.rates[uid] = c.Num("centsPerHour")
				return policies.Rec{"rate": policies.Rec{"userId": uid, "centsPerHour": m.rates[uid]}}, uid, nil
			}},
		{Name: "generatePayouts", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.Month("month")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				month := c.Num("month")
				if c.At < policies.MonthStart(month+1) {
					return nil, "", policies.Refuse("month_not_ended", "The month has not ended.")
				}
				if m.generated[month] {
					return nil, "", policies.Refuse("already_generated", "Payouts for the month were generated already.")
				}
				m.generated[month] = true
				out := []policies.Rec{}
				for _, uid := range policies.SortedKeys(m.rates) {
					minutes, n := 0, 0
					for _, t := range m.classes.ScheduledInMonth(uid, month) {
						minutes += t.Minutes
						n++
					}
					if n == 0 {
						continue
					}
					p := &Payout{ID: m.ids.Next("pyt"), UserID: uid, Month: month, Minutes: minutes,
						Amount: (minutes*m.rates[uid] + 30) / 60}
					m.byID[p.ID] = p
					m.list = append(m.list, p)
					m.ledger.Post("expensePayout", "payable", p.Amount, p.ID, c.At)
					out = append(out, payoutView(p))
				}
				return policies.Rec{"month": c.Str("month"), "payouts": out}, c.Str("month"), nil
			}},
		{Name: "settlePayout", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("payoutId", m.PayoutExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				p := m.byID[c.Str("payoutId")]
				if p.Settled {
					return nil, "", policies.Refuse("invalid_state", "The payout is settled already.")
				}
				p.Settled = true
				m.ledger.Post("payable", "cash", p.Amount, p.ID, c.At)
				return policies.Rec{"payout": payoutView(p)}, p.ID, nil
			}},
		{Name: "listPayouts", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("userId", userExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				uid := c.Str("userId")
				out := []policies.Rec{}
				for _, p := range m.list {
					if p.UserID == uid {
						out = append(out, payoutView(p))
					}
				}
				return policies.Rec{"userId": uid, "payouts": out}, "", nil
			}},
	}
}
