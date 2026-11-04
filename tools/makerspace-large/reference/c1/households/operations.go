package households

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "createHousehold", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.ID("householdId"), policies.Text("name"), policies.IDRef("payerId", m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				hid, payer := c.Str("householdId"), c.Str("payerId")
				if m.byID[hid] != nil {
					return nil, "", policies.Refuse("already_exists", "A household with identifier %s already exists.", hid)
				}
				if m.of[payer] != nil {
					return nil, "", policies.Refuse("already_in_household", "The member already belongs to a household.")
				}
				h := &Household{ID: hid, Name: c.Str("name"), PayerID: payer, MemberIDs: []string{payer}}
				m.byID[hid] = h
				m.of[payer] = h
				return policies.Rec{"household": householdView(h)}, hid, nil
			}},
		{Name: "addHouseholdMember", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("householdId", m.HouseholdExists), policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				h, mid := m.byID[c.Str("householdId")], c.Str("memberId")
				if m.of[mid] != nil {
					return nil, "", policies.Refuse("already_in_household", "The member already belongs to a household.")
				}
				if len(h.MemberIDs) >= maxMembers {
					return nil, "", policies.Refuse("household_full", "The household already has %d members.", maxMembers)
				}
				h.add(mid)
				m.of[mid] = h
				return policies.Rec{"household": householdView(h)}, h.ID + "/" + mid, nil
			}},
		{Name: "removeHouseholdMember", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("householdId", m.HouseholdExists), policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				h, mid := m.byID[c.Str("householdId")], c.Str("memberId")
				if m.of[mid] != h {
					return nil, "", policies.Refuse("not_in_household", "The member does not belong to the household.")
				}
				if mid == h.PayerID {
					return nil, "", policies.Refuse("payer_cannot_leave", "The member is the payer of the household.")
				}
				h.remove(mid)
				delete(m.of, mid)
				return policies.Rec{"household": householdView(h)}, h.ID + "/" + mid, nil
			}},
		{Name: "getHousehold", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("householdId", m.HouseholdExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				return policies.Rec{"household": householdView(m.byID[c.Str("householdId")])}, "", nil
			}},
	}
}
