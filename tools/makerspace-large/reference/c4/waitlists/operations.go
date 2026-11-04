package waitlists

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "joinWaitlist", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", m.training.CourseExists), policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cid, mid := c.Str("courseId"), c.Str("memberId")
				if !m.members.InGoodStanding(mid) {
					return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
				}
				if e, ok := m.training.EnrollmentOf(cid, mid); ok && e.Status == "enrolled" {
					return nil, "", policies.Refuse("already_enrolled", "The member is already enrolled.")
				}
				if m.training.FreePlaces(cid) > 0 {
					return nil, "", policies.Refuse("course_not_full", "The course has a free place.")
				}
				if m.indexOf(cid, mid) >= 0 {
					return nil, "", policies.Refuse("already_waiting", "The member is already on the waiting list.")
				}
				e := &Entry{CourseID: cid, MemberID: mid, JoinedAt: c.At}
				m.lists[cid] = append(m.lists[cid], e)
				return policies.Rec{"entry": entryView(e)}, cid + "/" + mid, nil
			}},
		{Name: "leaveWaitlist", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", m.training.CourseExists), policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cid, mid := c.Str("courseId"), c.Str("memberId")
				i := m.indexOf(cid, mid)
				if i < 0 {
					return nil, "", policies.Refuse("not_waiting", "The member is not on the waiting list.")
				}
				e := m.lists[cid][i]
				m.lists[cid] = append(m.lists[cid][:i:i], m.lists[cid][i+1:]...)
				return policies.Rec{"entry": entryView(e)}, cid + "/" + mid, nil
			}},
		{Name: "listWaitlist", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", m.training.CourseExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cid := c.Str("courseId")
				out := []policies.Rec{}
				for _, e := range m.lists[cid] {
					out = append(out, entryView(e))
				}
				return policies.Rec{"courseId": cid, "entries": out}, "", nil
			}},
		{Name: "promoteWaitlist", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", m.training.CourseExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cid := c.Str("courseId")
				enrolled, skipped := []policies.Rec{}, []policies.Rec{}
				var remaining []*Entry
				tried := append([]*Entry(nil), m.lists[cid]...)
				for i, e := range tried {
					if m.training.FreePlaces(cid) <= 0 {
						remaining = append(remaining, tried[i:]...)
						break
					}
					en, ref := m.training.Enroll(cid, e.MemberID, c.At)
					if ref != nil {
						skipped = append(skipped, policies.Rec{"memberId": e.MemberID, "code": ref.Code})
						remaining = append(remaining, e)
						continue
					}
					enrolled = append(enrolled, enrollmentView(en))
					m.notices.Post(e.MemberID, "waitlist", cid, c.At)
				}
				m.lists[cid] = remaining
				return policies.Rec{"courseId": cid, "enrollments": enrolled, "skipped": skipped}, cid, nil
			}},
	}
}
