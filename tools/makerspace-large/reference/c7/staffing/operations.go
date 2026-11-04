package staffing

import (
	"sort"

	"makerspace/policies"
)

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	userExists := func(id string) bool { return m.members.User(id) != nil }
	return []*policies.Operation{
		{Name: "addShift", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("userId", userExists), policies.Stamp("start"), policies.Stamp("end")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				uid, start, end := c.Str("userId"), c.Num("start"), c.Num("end")
				if m.members.User(uid).Role == "member" {
					return nil, "", policies.Refuse("not_staff", "The user is a member.")
				}
				if end <= start {
					return nil, "", policies.Refuse("invalid_interval", "The end is not after the start.")
				}
				if start < c.At {
					return nil, "", policies.Refuse("starts_in_past", "The shift starts before the request time.")
				}
				if end-start > maxShift {
					return nil, "", policies.Refuse("shift_too_long", "The shift is longer than %d minutes.", maxShift)
				}
				for _, s := range m.shiftList {
					if !s.Cancelled && s.UserID == uid && policies.Overlaps(start, end, s.Start, s.End) {
						return nil, "", policies.Refuse("shift_overlap", "Another shift of the user overlaps.")
					}
				}
				s := &Shift{ID: m.ids.Next("shf"), UserID: uid, Start: start, End: end}
				m.shifts[s.ID] = s
				m.shiftList = append(m.shiftList, s)
				return policies.Rec{"shift": shiftView(s)}, s.ID, nil
			}},
		{Name: "cancelShift", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("shiftId", m.ShiftExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				s := m.shifts[c.Str("shiftId")]
				if s.Cancelled || s.Start <= c.At {
					return nil, "", policies.Refuse("invalid_state", "The shift cannot be cancelled.")
				}
				if m.needed(s, c.At) {
					return nil, "", policies.Refuse("shift_needed", "An active reservation of a supervised family needs the shift.")
				}
				s.Cancelled = true
				return policies.Rec{"shift": shiftView(s)}, s.ID, nil
			}},
		{Name: "listShifts", Perms: "naa",
			Args: []policies.ArgSpec{policies.Date("date")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				from := c.Num("date") * policies.MinPerDay
				var list []*Shift
				for _, s := range m.shiftList {
					if !s.Cancelled && policies.Overlaps(s.Start, s.End, from, from+policies.MinPerDay) {
						list = append(list, s)
					}
				}
				sort.SliceStable(list, func(i, j int) bool { return list[i].Start < list[j].Start })
				out := make([]policies.Rec, 0, len(list))
				for _, s := range list {
					out = append(out, shiftView(s))
				}
				return policies.Rec{"date": c.Str("date"), "shifts": out}, "", nil
			}},
		{Name: "setSupervision", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("familyId", m.equipment.FamilyExists), policies.Flag("required")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				fid := c.Str("familyId")
				m.supervised[fid] = c.Flag("required")
				return policies.Rec{"supervision": policies.Rec{"familyId": fid, "required": m.supervised[fid]}}, fid, nil
			}},
	}
}
