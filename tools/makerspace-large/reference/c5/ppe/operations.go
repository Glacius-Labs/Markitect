package ppe

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addPpeItem", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.ID("itemId"), policies.Text("name"), policies.Integer("stock", 0, 100000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("itemId")
				if m.items[id] != nil {
					return nil, "", policies.Refuse("already_exists", "An item with identifier %s already exists.", id)
				}
				i := &item{id: id, name: c.Str("name"), stock: c.Num("stock")}
				m.items[id] = i
				return policies.Rec{"item": itemView(i)}, id, nil
			}},
		{Name: "requirePpe", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("familyId", m.equipment.FamilyExists), policies.IDRef("itemId", m.ItemExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				fid, iid := c.Str("familyId"), c.Str("itemId")
				if m.required[fid] == nil {
					m.required[fid] = map[string]bool{}
				}
				m.required[fid][iid] = true
				return policies.Rec{"requirement": policies.Rec{"familyId": fid, "itemId": iid}}, fid + "/" + iid, nil
			}},
		{Name: "issuePpe", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.IDRef("itemId", m.ItemExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, iid := c.Str("memberId"), c.Str("itemId")
				i := m.items[iid]
				if m.members.Member(mid).Standing != "good" {
					return nil, "", policies.Refuse("member_suspended", "The member is suspended.")
				}
				if m.held[mid][iid] {
					return nil, "", policies.Refuse("already_issued", "The member already holds the item.")
				}
				if i.stock == 0 {
					return nil, "", policies.Refuse("out_of_stock", "The item is out of stock.")
				}
				i.stock--
				if m.held[mid] == nil {
					m.held[mid] = map[string]bool{}
				}
				m.held[mid][iid] = true
				return policies.Rec{"item": itemView(i)}, mid + "/" + iid, nil
			}},
		{Name: "returnPpe", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.IDRef("itemId", m.ItemExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, iid := c.Str("memberId"), c.Str("itemId")
				if !m.held[mid][iid] {
					return nil, "", policies.Refuse("not_issued", "The member does not hold the item.")
				}
				i := m.items[iid]
				i.stock++
				delete(m.held[mid], iid)
				return policies.Rec{"item": itemView(i)}, mid + "/" + iid, nil
			}},
		{Name: "listMemberPpe", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, iid := range policies.SortedKeys(m.held[mid]) {
					out = append(out, itemView(m.items[iid]))
				}
				return policies.Rec{"memberId": mid, "items": out}, "", nil
			}},
	}
}
