package waivers

import (
	"sort"

	"makerspace/policies"
)

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "publishWaiver", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("familyId", m.equipment.FamilyExists), policies.Text("text")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				fid := c.Str("familyId")
				w := m.waivers[fid]
				if w == nil {
					w = &waiver{familyID: fid}
					m.waivers[fid] = w
				}
				w.version++
				w.text = c.Str("text")
				return policies.Rec{"waiver": waiverView(w)}, fid, nil
			}},
		{Name: "getWaiver", Perms: "aaa",
			Args: []policies.ArgSpec{policies.IDRef("familyId", m.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				w := m.waivers[c.Str("familyId")]
				if w == nil {
					return nil, "", policies.Refuse("no_waiver", "The family has no waiver.")
				}
				return policies.Rec{"waiver": waiverView(w)}, "", nil
			}},
		{Name: "signWaiver", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.IDRef("familyId", m.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, fid := c.Str("memberId"), c.Str("familyId")
				w := m.waivers[fid]
				if w == nil {
					return nil, "", policies.Refuse("no_waiver", "The family has no waiver.")
				}
				if m.Signed(mid, fid, policies.DayOf(c.At)) {
					return nil, "", policies.Refuse("already_signed", "The member has signed the current waiver.")
				}
				if m.guardians.ConsentMissing(mid, fid, policies.DayOf(c.At)) {
					return nil, "", policies.Refuse("guardian_required", "The member is a minor and no guardian has consented.")
				}
				s := &signature{memberID: mid, familyID: fid, version: w.version, signedAt: c.At}
				m.latest[sigKey(mid, fid)] = s
				return policies.Rec{"signature": signatureView(s)}, mid + "/" + fid, nil
			}},
		{Name: "listMemberWaivers", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				var list []*signature
				for _, s := range m.latest {
					if s.memberID == mid {
						list = append(list, s)
					}
				}
				sort.Slice(list, func(i, j int) bool { return list[i].familyID < list[j].familyID })
				out := make([]policies.Rec, 0, len(list))
				for _, s := range list {
					out = append(out, signatureView(s))
				}
				return policies.Rec{"memberId": mid, "signatures": out}, "", nil
			}},
	}
}
