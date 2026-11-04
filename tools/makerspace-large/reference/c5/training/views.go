package training

import "makerspace/policies"

func (t *Module) courseView(c *course) policies.Rec {
	count := 0
	for _, e := range t.enrollments {
		if e.courseID == c.id && e.status == "enrolled" {
			count++
		}
	}
	return policies.Rec{"courseId": c.id, "familyId": c.familyID, "title": c.title,
		"feeCents": c.fee, "capacity": c.capacity, "enrolledCount": count}
}

func enrollmentView(e *enrollment) policies.Rec {
	return policies.Rec{"courseId": e.courseID, "memberId": e.memberID, "status": e.status,
		"enrolledAt": policies.FormatTime(e.enrolledAt), "completedAt": policies.OptTime(e.completedAt)}
}

func certView(c *certification, at int) policies.Rec {
	expires := c.issuedOn + 365
	status := "valid"
	if c.revoked {
		status = "revoked"
	} else if policies.DayOf(at) >= expires {
		status = "expired"
	}
	return policies.Rec{"memberId": c.memberID, "familyId": c.familyID, "issuedOn": policies.FormatDay(c.issuedOn),
		"expiresOn": policies.FormatDay(expires), "status": status}
}
