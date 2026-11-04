package loans

import (
	"makerspace/billing"
	"makerspace/policies"
)

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addTool", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.ID("toolId"), policies.Text("name"), policies.IDRef("familyId", m.equipment.FamilyExists),
				policies.Integer("lateFeeCentsPerDay", 1, 100000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("toolId")
				if m.tools[id] != nil {
					return nil, "", policies.Refuse("already_exists", "A tool with identifier %s already exists.", id)
				}
				t := &Tool{ID: id, Name: c.Str("name"), FamilyID: c.Str("familyId"), LateFeePerDay: c.Num("lateFeeCentsPerDay")}
				m.tools[id] = t
				return policies.Rec{"tool": toolView(t)}, id, nil
			}},
		{Name: "listTools", Perms: "aaa",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				ids := policies.SortedKeys(m.tools)
				out := make([]policies.Rec, 0, len(ids))
				for _, id := range ids {
					out = append(out, toolView(m.tools[id]))
				}
				return policies.Rec{"tools": out}, "", nil
			}},
		{Name: "borrowTool", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.IDRef("toolId", m.ToolExists)},
			Run:  m.runBorrowTool},
		{Name: "returnTool", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("loanId", m.LoanExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				l := m.byID[c.Str("loanId")]
				if l.ReturnedAt != nil {
					return nil, "", policies.Refuse("invalid_state", "The loan has been returned.")
				}
				var charge any
				if ch := m.chargeLateDays(l, c.At); ch != nil {
					charge = billing.ChargeView(ch)
				}
				at := c.At
				l.ReturnedAt = &at
				t := m.tools[l.ToolID]
				t.OnLoan = false
				return policies.Rec{"loan": loanView(l, c.At), "tool": toolView(t), "charge": charge}, l.ID, nil
			}},
		{Name: "renewLoan", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.OwnedIDRef("loanId", m.LoanExists, m.LoanOwner)},
			Run:  m.runRenewLoan},
		{Name: "listMemberLoans", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, l := range m.list {
					if l.MemberID == mid {
						out = append(out, loanView(l, c.At))
					}
				}
				return policies.Rec{"memberId": mid, "loans": out}, "", nil
			}},
		{Name: "chargeOverdueLoans", Mutation: true, Perms: "naa",
			Run: m.runChargeOverdueLoans},
	}
}

func (m *Module) runBorrowTool(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	mid, t := c.Str("memberId"), m.tools[c.Str("toolId")]
	day := policies.DayOf(c.At)
	if c.ActorRole == "staff" && !m.staffing.OnShift(c.ActorID, c.At) {
		return nil, "", policies.Refuse("approver_not_on_shift", "The actor has no scheduled shift at the request time.")
	}
	if !m.members.InGoodStanding(mid) {
		return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
	}
	if m.guardians.ConsentMissing(mid, t.FamilyID, day) {
		return nil, "", policies.Refuse("guardian_consent_missing", "The member is a minor without consent for the family.")
	}
	if !m.training.Certified(mid, t.FamilyID, day) {
		return nil, "", policies.Refuse("not_certified", "The member is not certified for the family of the tool.")
	}
	if m.waivers.Missing(mid, t.FamilyID, day) {
		return nil, "", policies.Refuse("waiver_missing", "The member has not signed the current waiver of the family.")
	}
	if t.OnLoan {
		return nil, "", policies.Refuse("tool_on_loan", "The tool has an active loan.")
	}
	if m.HasOverdue(mid, day) {
		return nil, "", policies.Refuse("loan_overdue", "The member has an overdue tool loan.")
	}
	active := 0
	for _, l := range m.list {
		if l.MemberID == mid && l.ReturnedAt == nil {
			active++
		}
	}
	if active >= maxActive {
		return nil, "", policies.Refuse("loan_limit", "The member already has %d active loans.", maxActive)
	}
	l := &Loan{ID: m.ids.Next("lon"), ToolID: t.ID, MemberID: mid, BorrowedAt: c.At, DueOn: day + loanDays}
	m.byID[l.ID] = l
	m.list = append(m.list, l)
	t.OnLoan = true
	return policies.Rec{"loan": loanView(l, c.At), "tool": toolView(t)}, l.ID, nil
}

func (m *Module) runRenewLoan(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	l := m.byID[c.Str("loanId")]
	if l.ReturnedAt != nil {
		return nil, "", policies.Refuse("invalid_state", "The loan has been returned.")
	}
	if !m.members.InGoodStanding(l.MemberID) {
		return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
	}
	if l.OverdueOn(policies.DayOf(c.At)) {
		return nil, "", policies.Refuse("loan_overdue", "The loan is overdue.")
	}
	if l.Renewals >= maxRenewals {
		return nil, "", policies.Refuse("renewal_limit", "The loan has been renewed %d times.", maxRenewals)
	}
	l.DueOn += loanDays
	l.Renewals++
	return policies.Rec{"loan": loanView(l, c.At)}, l.ID, nil
}

// runChargeOverdueLoans charges the unpaid late days of every overdue loan and
// posts a notice for the first charge of a loan. The loans are in creation
// order, which is ascending loan number.
func (m *Module) runChargeOverdueLoans(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	out := []policies.Rec{}
	for _, l := range m.list {
		first := l.ChargedDays == 0
		ch := m.chargeLateDays(l, c.At)
		if ch == nil {
			continue
		}
		if first {
			m.notices.Post(l.MemberID, "loan", l.ID, c.At)
		}
		out = append(out, billing.ChargeView(ch))
	}
	return policies.Rec{"charges": out}, "overdueLoans", nil
}
