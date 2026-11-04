package training

import (
	"sort"

	"makerspace/policies"
)

// Operations lists the operations of the module.
func (t *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addCourse", Mutation: true, Perms: "naa",
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
		{Name: "enrollInCourse", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", t.CourseExists), policies.MemberRef(t.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				co := t.courses[c.Str("courseId")]
				mid := c.Str("memberId")
				e, ref := t.enroll(co.id, mid, c.At)
				if ref != nil {
					return nil, "", ref
				}
				return policies.Rec{"enrollment": enrollmentView(e), "course": t.courseView(co)}, co.id + "/" + mid, nil
			}},
		{Name: "completeCourse", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", t.CourseExists), policies.MemberRef(t.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				co := t.courses[c.Str("courseId")]
				m := t.members.Member(c.Str("memberId"))
				e := t.enrollments[enrollmentKey(co.id, m.User.ID)]
				if e == nil || e.status != "enrolled" {
					return nil, "", policies.Refuse("not_enrolled", "The member is not enrolled in the course.")
				}
				if t.needs.AttendanceOwed(co.id, m.User.ID, e.enrolledAt, c.At) {
					return nil, "", policies.Refuse("attendance_missing", "The member did not attend a class of the course.")
				}
				if !t.needs.AssessmentMet(co.id, m.User.ID) {
					return nil, "", policies.Refuse("assessment_not_passed", "The member has not passed the assessment of the course.")
				}
				at := c.At
				e.status, e.completedAt = "completed", &at
				cert := &certification{memberID: m.User.ID, familyID: co.familyID, issuedOn: policies.DayOf(c.At)}
				t.certs[certKey(m.User.ID, co.familyID)] = cert
				return policies.Rec{"enrollment": enrollmentView(e), "certification": certView(cert, c.At)},
					co.id + "/" + m.User.ID, nil
			}},
		{Name: "revokeCertification", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.MemberRef(t.members.Exists), policies.IDRef("familyId", t.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				m := t.members.Member(c.Str("memberId"))
				cert := t.certs[certKey(m.User.ID, c.Str("familyId"))]
				if cert == nil || cert.revoked {
					return nil, "", policies.Refuse("not_certified", "There is no certification to revoke.")
				}
				cert.revoked = true
				return policies.Rec{"certification": certView(cert, c.At)}, m.User.ID + "/" + cert.familyID, nil
			}},
		{Name: "listCertifications", Perms: "oaa",
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
					out = append(out, certView(cert, c.At))
				}
				return policies.Rec{"memberId": mid, "certifications": out}, "", nil
			}},
	}
}
