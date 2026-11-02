package training

import (
	"sort"

	"makerspace/policies"
)

// Operations lists the operations of the module.
func (t *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addCourse", Mutation: true, Perms: "naan",
			Args: []policies.ArgSpec{policies.ID("courseId"), policies.IDRef("familyId", t.equipment.FamilyExists), policies.Text("title"),
				policies.Integer("feeCents", 0, 1000000), policies.Integer("capacity", 1, 1000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cid := c.Str("courseId")
				if t.courses[cid] != nil {
					return nil, "", policies.Refuse("already_exists", "A course with identifier %s already exists.", cid)
				}
				co := &course{id: cid, familyID: c.Str("familyId"), title: c.Str("title"),
					fee: c.Num("feeCents"), capacity: c.Num("capacity")}
				t.courses[cid] = co
				return policies.Rec{"course": t.courseView(co)}, cid, nil
			}},
		{Name: "enrollInCourse", Mutation: true, Perms: "oaan",
			Args: []policies.ArgSpec{policies.IDRef("courseId", t.CourseExists), policies.MemberRef(t.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				co := t.courses[c.Str("courseId")]
				m := t.members.Member(c.Str("memberId"))
				if m.Standing != "good" {
					return nil, "", policies.Refuse("member_suspended", "The member is suspended.")
				}
				key := enrollmentKey(co.id, m.User.ID)
				e := t.enrollments[key]
				if e != nil && e.status == "enrolled" {
					return nil, "", policies.Refuse("already_enrolled", "The member is already enrolled.")
				}
				if t.courseView(co)["enrolledCount"].(int) >= co.capacity {
					return nil, "", policies.Refuse("course_full", "The course has no free place.")
				}
				if e == nil {
					e = &enrollment{courseID: co.id, memberID: m.User.ID}
					t.enrollments[key] = e
				}
				e.status, e.enrolledAt, e.completedAt = "enrolled", c.At, nil
				t.billing.AddCharge(m.User.ID, "course", co.id, 1, co.fee, 0, c.At)
				return policies.Rec{"enrollment": enrollmentView(e), "course": t.courseView(co)},
					co.id + "/" + m.User.ID, nil
			}},
		{Name: "completeCourse", Mutation: true, Perms: "naan",
			Args: []policies.ArgSpec{policies.IDRef("courseId", t.CourseExists), policies.MemberRef(t.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				co := t.courses[c.Str("courseId")]
				m := t.members.Member(c.Str("memberId"))
				e := t.enrollments[enrollmentKey(co.id, m.User.ID)]
				if e == nil || e.status != "enrolled" {
					return nil, "", policies.Refuse("not_enrolled", "The member is not enrolled in the course.")
				}
				at := c.At
				e.status, e.completedAt = "completed", &at
				cert := &certification{memberID: m.User.ID, familyID: co.familyID, issuedOn: policies.DayOf(c.At)}
				t.certs[certKey(m.User.ID, co.familyID)] = cert
				return policies.Rec{"enrollment": enrollmentView(e), "certification": certView(cert)},
					co.id + "/" + m.User.ID, nil
			}},
		{Name: "revokeCertification", Mutation: true, Perms: "naan",
			Args: []policies.ArgSpec{policies.MemberRef(t.members.Exists), policies.IDRef("familyId", t.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				m := t.members.Member(c.Str("memberId"))
				cert := t.certs[certKey(m.User.ID, c.Str("familyId"))]
				if cert == nil || cert.revoked {
					return nil, "", policies.Refuse("not_certified", "There is no certification to revoke.")
				}
				cert.revoked = true
				return policies.Rec{"certification": certView(cert)}, m.User.ID + "/" + cert.familyID, nil
			}},
		{Name: "listCertifications", Perms: "oaan",
			Args: []policies.ArgSpec{policies.MemberRef(t.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				var list []*certification
				for _, cert := range t.certs {
					if cert.memberID == mid {
						list = append(list, cert)
					}
				}
				sort.Slice(list, func(i, j int) bool { return list[i].familyID < list[j].familyID })
				out := make([]policies.Rec, 0, len(list))
				for _, cert := range list {
					out = append(out, certView(cert))
				}
				return policies.Rec{"memberId": mid, "certifications": out}, "", nil
			}},
	}
}
