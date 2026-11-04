package deposits

import (
	"makerspace/billing"
	"makerspace/policies"
)

// paid refuses with deposit_unpaid unless the deposit's charge is on a paid
// invoice.
func (m *Module) unpaid(d *Deposit) *policies.Refusal {
	if !m.billing.ChargeOnPaidInvoice("deposit", d.ID) {
		return policies.Refuse("deposit_unpaid", "The deposit's charge is not on a paid invoice.")
	}
	return nil
}

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "takeDeposit", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.Text("purpose"),
				policies.Integer("amountCents", 1, 100000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				if !m.members.InGoodStanding(mid) {
					return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
				}
				d := &Deposit{ID: m.ids.Next("dep"), MemberID: mid, Purpose: c.Str("purpose"), Status: "held",
					Amount: c.Num("amountCents"), TakenAt: c.At}
				m.byID[d.ID] = d
				m.list = append(m.list, d)
				ch := m.billing.AddCharge(mid, "deposit", d.ID, 1, d.Amount, 0, c.At)
				return policies.Rec{"deposit": depositView(d), "charge": billing.ChargeView(ch)}, d.ID, nil
			}},
		{Name: "releaseDeposit", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.OwnedIDRef("depositId", m.DepositExists, m.DepositOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				d := m.byID[c.Str("depositId")]
				if d.Status != "held" {
					return nil, "", policies.Refuse("invalid_state", "The deposit is not held.")
				}
				if r := m.unpaid(d); r != nil {
					return nil, "", r
				}
				at := c.At
				d.Status, d.ClosedAt = "released", &at
				m.billing.AddCredit(d.MemberID, d.Amount, "depositLiability", d.ID, c.At)
				return policies.Rec{"deposit": depositView(d),
					"creditBalanceCents": m.billing.CreditBalance(d.MemberID)}, d.ID, nil
			}},
		{Name: "forfeitDeposit", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.OwnedIDRef("depositId", m.DepositExists, m.DepositOwner), policies.Text("reason")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				d := m.byID[c.Str("depositId")]
				if d.Status != "held" {
					return nil, "", policies.Refuse("invalid_state", "The deposit is not held.")
				}
				if r := m.unpaid(d); r != nil {
					return nil, "", r
				}
				at := c.At
				d.Status, d.ClosedAt = "forfeited", &at
				m.ledger.Post("depositLiability", "revenueDeposits", d.Amount, d.ID, c.At)
				return policies.Rec{"deposit": depositView(d)}, d.ID, nil
			}},
		{Name: "listDeposits", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, d := range m.list {
					if d.MemberID == mid {
						out = append(out, depositView(d))
					}
				}
				return policies.Rec{"memberId": mid, "deposits": out}, "", nil
			}},
	}
}
