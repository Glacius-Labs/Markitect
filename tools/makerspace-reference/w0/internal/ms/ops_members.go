package ms

import "sort"

func memberOps() []*opDef {
	return []*opDef{
		{name: "registerUser", mutation: true, perms: "nna",
			args: []argSpec{id("userId"), text("name"), enum("role", "staff", "admin")},
			run: func(c *call) (rec, string, *refusal) {
				uid := c.str("userId")
				if c.s.users[uid] != nil {
					return nil, "", refuse("already_exists", "A user with identifier %s already exists.", uid)
				}
				u := &user{id: uid, name: c.str("name"), role: c.str("role")}
				c.s.users[uid] = u
				return rec{"user": c.s.userView(u)}, uid, nil
			}},
		{name: "registerMember", mutation: true, perms: "naa",
			args: []argSpec{id("memberId"), text("name"), enum("tier", tiers...)},
			run: func(c *call) (rec, string, *refusal) {
				mid := c.str("memberId")
				if c.s.users[mid] != nil {
					return nil, "", refuse("already_exists", "A user with identifier %s already exists.", mid)
				}
				u := &user{id: mid, name: c.str("name"), role: "member"}
				m := &member{user: u, tier: c.str("tier"), standing: "good", registeredAt: c.at}
				c.s.users[mid] = u
				c.s.members[mid] = m
				return rec{"member": c.s.memberView(m)}, mid, nil
			}},
		{name: "getMember", perms: "oaa",
			args: []argSpec{idRef("memberId", refMember)},
			run: func(c *call) (rec, string, *refusal) {
				return rec{"member": c.s.memberView(c.member("memberId"))}, "", nil
			}},
		{name: "changeTier", mutation: true, perms: "naa",
			args: []argSpec{idRef("memberId", refMember), enum("tier", tiers...)},
			run: func(c *call) (rec, string, *refusal) {
				m := c.member("memberId")
				m.tier = c.str("tier")
				return rec{"member": c.s.memberView(m)}, m.user.id, nil
			}},
		{name: "setMemberStanding", mutation: true, perms: "naa",
			args: []argSpec{idRef("memberId", refMember), enum("standing", "good", "suspended")},
			run: func(c *call) (rec, string, *refusal) {
				m := c.member("memberId")
				m.standing = c.str("standing")
				return rec{"member": c.s.memberView(m)}, m.user.id, nil
			}},
	}
}

func trainingOps() []*opDef {
	return []*opDef{
		{name: "addCourse", mutation: true, perms: "naa",
			args: []argSpec{id("courseId"), idRef("familyId", refFamily), text("title"),
				integer("feeCents", 0, 1000000), integer("capacity", 1, 1000)},
			run: func(c *call) (rec, string, *refusal) {
				cid := c.str("courseId")
				if c.s.courses[cid] != nil {
					return nil, "", refuse("already_exists", "A course with identifier %s already exists.", cid)
				}
				co := &course{id: cid, familyID: c.str("familyId"), title: c.str("title"),
					fee: c.num("feeCents"), capacity: c.num("capacity")}
				c.s.courses[cid] = co
				return rec{"course": c.s.courseView(co)}, cid, nil
			}},
		{name: "enrollInCourse", mutation: true, perms: "oaa",
			args: []argSpec{idRef("courseId", refCourse), idRef("memberId", refMember)},
			run: func(c *call) (rec, string, *refusal) {
				co := c.s.courses[c.str("courseId")]
				m := c.member("memberId")
				if m.standing != "good" {
					return nil, "", refuse("member_suspended", "The member is suspended.")
				}
				key := enrollmentKey(co.id, m.user.id)
				e := c.s.enrollments[key]
				if e != nil && e.status == "enrolled" {
					return nil, "", refuse("already_enrolled", "The member is already enrolled.")
				}
				if c.s.courseView(co)["enrolledCount"].(int) >= co.capacity {
					return nil, "", refuse("course_full", "The course has no free place.")
				}
				if e == nil {
					e = &enrollment{courseID: co.id, memberID: m.user.id}
					c.s.enrollments[key] = e
				}
				e.status, e.enrolledAt, e.completedAt = "enrolled", c.at, nil
				c.s.addCharge(m.user.id, "course", co.id, 1, co.fee, 0, c.at)
				return rec{"enrollment": enrollmentView(e), "course": c.s.courseView(co)},
					co.id + "/" + m.user.id, nil
			}},
		{name: "completeCourse", mutation: true, perms: "naa",
			args: []argSpec{idRef("courseId", refCourse), idRef("memberId", refMember)},
			run: func(c *call) (rec, string, *refusal) {
				co := c.s.courses[c.str("courseId")]
				m := c.member("memberId")
				e := c.s.enrollments[enrollmentKey(co.id, m.user.id)]
				if e == nil || e.status != "enrolled" {
					return nil, "", refuse("not_enrolled", "The member is not enrolled in the course.")
				}
				at := c.at
				e.status, e.completedAt = "completed", &at
				cert := &certification{memberID: m.user.id, familyID: co.familyID, issuedOn: dayOf(c.at)}
				c.s.certs[certKey(m.user.id, co.familyID)] = cert
				return rec{"enrollment": enrollmentView(e), "certification": c.s.certView(cert, c.at)},
					co.id + "/" + m.user.id, nil
			}},
		{name: "revokeCertification", mutation: true, perms: "naa",
			args: []argSpec{idRef("memberId", refMember), idRef("familyId", refFamily)},
			run: func(c *call) (rec, string, *refusal) {
				m := c.member("memberId")
				cert := c.s.certs[certKey(m.user.id, c.str("familyId"))]
				if cert == nil || cert.revoked {
					return nil, "", refuse("not_certified", "There is no certification to revoke.")
				}
				cert.revoked = true
				return rec{"certification": c.s.certView(cert, c.at)}, m.user.id + "/" + cert.familyID, nil
			}},
		{name: "listCertifications", perms: "oaa",
			args: []argSpec{idRef("memberId", refMember)},
			run: func(c *call) (rec, string, *refusal) {
				mid := c.str("memberId")
				var list []*certification
				for _, cert := range c.s.certs {
					if cert.memberID == mid {
						list = append(list, cert)
					}
				}
				sort.Slice(list, func(i, j int) bool { return list[i].familyID < list[j].familyID })
				out := make([]rec, 0, len(list))
				for _, cert := range list {
					out = append(out, c.s.certView(cert, c.at))
				}
				return rec{"memberId": mid, "certifications": out}, "", nil
			}},
	}
}

// certified reports whether the member holds a certification for the family
// that is valid on the given day (section 7.2).
func (s *state) certified(memberID, familyID string, day int) bool {
	cert := s.certs[certKey(memberID, familyID)]
	return cert != nil && !cert.revoked && cert.issuedOn <= day && day < cert.issuedOn+365
}
