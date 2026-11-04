package scholarships

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addFund", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.ID("fundId"), policies.Integer("budgetCents", 1, 100000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("fundId")
				if m.funds[id] != nil {
					return nil, "", policies.Refuse("already_exists", "A fund with identifier %s already exists.", id)
				}
				f := &Fund{ID: id, Budget: c.Num("budgetCents")}
				m.funds[id] = f
				return policies.Rec{"fund": fundView(f)}, id, nil
			}},
		{Name: "awardScholarship", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("fundId", m.FundExists), policies.MemberRef(m.members.Exists),
				policies.Integer("percent", 1, 100)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				if _, _, ok := m.ActiveAward(mid); ok {
					return nil, "", policies.Refuse("already_awarded", "The member already has an active award.")
				}
				a := &Award{ID: m.ids.Next("aws"), FundID: c.Str("fundId"), MemberID: mid, Percent: c.Num("percent")}
				m.list = append(m.list, a)
				return policies.Rec{"award": awardView(a)}, a.ID, nil
			}},
		{Name: "revokeScholarship", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("awardId", m.awardExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				a := m.award(c.Str("awardId"))
				if a.Revoked {
					return nil, "", policies.Refuse("invalid_state", "The award is already revoked.")
				}
				a.Revoked = true
				return policies.Rec{"award": awardView(a)}, a.ID, nil
			}},
		{Name: "listAwards", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, a := range m.list {
					if a.MemberID == mid {
						out = append(out, awardView(a))
					}
				}
				return policies.Rec{"memberId": mid, "awards": out}, "", nil
			}},
	}
}
