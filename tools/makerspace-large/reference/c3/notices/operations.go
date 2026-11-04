package notices

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "postNotice", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.Text("subject")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				n := m.add(c.Str("memberId"), "manual", c.Str("subject"), nil, c.At)
				return policies.Rec{"notice": noticeView(n)}, n.ID, nil
			}},
		{Name: "listNotices", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, n := range m.list {
					if n.MemberID == mid {
						out = append(out, noticeView(n))
					}
				}
				return policies.Rec{"memberId": mid, "notices": out}, "", nil
			}},
		{Name: "acknowledgeNotice", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.OwnedIDRef("noticeId", m.NoticeExists, m.NoticeOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				n := m.byID[c.Str("noticeId")]
				if n.AcknowledgedAt != nil {
					return nil, "", policies.Refuse("invalid_state", "The notice is already acknowledged.")
				}
				at := c.At
				n.AcknowledgedAt = &at
				return policies.Rec{"notice": noticeView(n)}, n.ID, nil
			}},
	}
}
