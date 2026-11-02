package members

import "makerspace/policies"

func (m *Module) member(c *policies.Call, name string) *Member { return m.members[c.Vals[name].S] }

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "registerUser", Mutation: true, Perms: "nnan",
			Args: []policies.ArgSpec{policies.ID("userId"), policies.Text("name"), policies.Enum("role", "staff", "admin", "steward")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				uid := c.Str("userId")
				if m.users[uid] != nil {
					return nil, "", policies.Refuse("already_exists", "A user with identifier %s already exists.", uid)
				}
				u := &User{ID: uid, Name: c.Str("name"), Role: c.Str("role")}
				m.users[uid] = u
				return policies.Rec{"user": userView(u)}, uid, nil
			}},
		{Name: "registerMember", Mutation: true, Perms: "naan",
			Args: []policies.ArgSpec{policies.ID("memberId"), policies.Text("name"), policies.Enum("tier", tiers...)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				if m.users[mid] != nil {
					return nil, "", policies.Refuse("already_exists", "A user with identifier %s already exists.", mid)
				}
				u := &User{ID: mid, Name: c.Str("name"), Role: "member"}
				mem := &Member{User: u, Tier: c.Str("tier"), Standing: "good", RegisteredAt: c.At}
				m.users[mid] = u
				m.members[mid] = mem
				return policies.Rec{"member": memberView(mem)}, mid, nil
			}},
		{Name: "getMember", Perms: "oaan",
			Args: []policies.ArgSpec{policies.MemberRef(m.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				return policies.Rec{"member": memberView(m.member(c, "memberId"))}, "", nil
			}},
		{Name: "changeTier", Mutation: true, Perms: "naan",
			Args: []policies.ArgSpec{policies.MemberRef(m.Exists), policies.Enum("tier", tiers...)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mem := m.member(c, "memberId")
				mem.Tier = c.Str("tier")
				return policies.Rec{"member": memberView(mem)}, mem.User.ID, nil
			}},
		{Name: "setMemberStanding", Mutation: true, Perms: "naan",
			Args: []policies.ArgSpec{policies.MemberRef(m.Exists), policies.Enum("standing", "good", "suspended")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mem := m.member(c, "memberId")
				mem.Standing = c.Str("standing")
				return policies.Rec{"member": memberView(mem)}, mem.User.ID, nil
			}},
	}
}
