package ms_test

import "testing"

func TestRegisterUserAndMember(t *testing.T) {
	e := newEnv(t)
	at := "2026-03-02T09:00Z"
	u := obj(e.ok("admin", at, "registerUser", m{"userId": "root2", "name": "Root", "role": "admin"})["user"])
	assertKeys(t, u, "userId", "name", "role")
	if u["role"] != "admin" {
		t.Fatalf("%v", u)
	}
	e.refused("admin", at, "registerUser", m{"userId": "admin", "name": "x", "role": "staff"}, "already_exists")
	e.refused("admin", at, "registerUser", m{"userId": "u", "name": "x", "role": "member"}, "invalid_argument")
	mem := obj(e.ok("admin", at, "registerMember", m{"memberId": "ada", "name": "Ada Byron", "tier": "plus"})["member"])
	assertKeys(t, mem, "memberId", "name", "tier", "standing", "registeredAt")
	if mem["standing"] != "good" || mem["tier"] != "plus" || mem["registeredAt"] != at {
		t.Fatalf("%v", mem)
	}
	// users share one identifier space
	e.refused("admin", at, "registerMember", m{"memberId": "ada", "name": "x", "tier": "basic"}, "already_exists")
	e.refused("admin", at, "registerMember", m{"memberId": "root2", "name": "x", "tier": "basic"}, "already_exists")
	e.refused("admin", at, "registerUser", m{"userId": "ada", "name": "x", "role": "staff"}, "already_exists")
	// the registry never replaces an existing record
	got := obj(e.ok("admin", at, "getMember", m{"memberId": "ada"})["member"])
	if got["name"] != "Ada Byron" {
		t.Fatalf("%v", got)
	}
}

func TestTierAndStanding(t *testing.T) {
	e := shop(t)
	at := "2026-03-02T09:00Z"
	mem := obj(e.ok("sam", at, "changeTier", m{"memberId": "ada", "tier": "pro"})["member"])
	if mem["tier"] != "pro" {
		t.Fatalf("%v", mem)
	}
	e.ok("sam", at, "changeTier", m{"memberId": "ada", "tier": "pro"})
	e.refused("sam", at, "changeTier", m{"memberId": "ada", "tier": "gold"}, "invalid_argument")
	mem = obj(e.ok("sam", at, "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})["member"])
	if mem["standing"] != "suspended" {
		t.Fatalf("%v", mem)
	}
	e.ok("sam", at, "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})
	e.refused("sam", at, "setMemberStanding", m{"memberId": "ada", "standing": "banned"}, "invalid_argument")
	e.refused("sam", at, "setMemberStanding", m{"memberId": "nobody", "standing": "good"}, "not_found")
}

func TestSuspendedMemberKeepsWhatItHas(t *testing.T) {
	e := shop(t)
	e.ok("sam", "2026-03-02T09:00Z", "createReservation", m{"memberId": "ada", "machineId": "lc1",
		"start": "2026-03-03T10:00Z", "end": "2026-03-03T11:00Z"})
	e.ok("sam", "2026-03-02T09:00Z", "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})
	// existing reservation stays and can be cancelled; new restricted actions are refused
	res := list(e.ok("ada", "2026-03-02T09:00Z", "listMemberReservations", m{"memberId": "ada"})["reservations"])
	if len(res) != 1 || obj(res[0])["status"] != "reserved" {
		t.Fatalf("%v", res)
	}
	e.refused("ada", "2026-03-02T09:00Z", "createReservation", m{"memberId": "ada", "machineId": "lc1",
		"start": "2026-03-03T12:00Z", "end": "2026-03-03T13:00Z"}, "member_suspended")
	e.refused("ada", "2026-03-03T10:00Z", "checkIn", m{"reservationId": "res-1"}, "member_suspended")
	e.refused("ada", "2026-03-02T09:00Z", "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"}, "member_suspended")
	e.ok("ada", "2026-03-02T09:00Z", "cancelReservation", m{"reservationId": "res-1"})
}

func TestCourses(t *testing.T) {
	e := shop(t)
	at := "2026-03-02T09:00Z"
	assertKeys(t, obj(e.ok("sam", at, "addCourse", m{"courseId": "c2", "familyId": "laser", "title": "T", "feeCents": 0, "capacity": 1})["course"]),
		"courseId", "familyId", "title", "feeCents", "capacity", "enrolledCount")
	e.refused("sam", at, "addCourse", m{"courseId": "c2", "familyId": "laser", "title": "T", "feeCents": 0, "capacity": 1}, "already_exists")
	e.refused("sam", at, "addCourse", m{"courseId": "c3", "familyId": "laser", "title": "T", "feeCents": 0, "capacity": 0}, "invalid_argument")
	e.refused("sam", at, "addCourse", m{"courseId": "c3", "familyId": "laser", "title": "T", "feeCents": 0, "capacity": 1001}, "invalid_argument")

	// ada completed laser-101, so a place is free again; enrolling after completion resets the record
	r := e.ok("sam", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"})
	en := obj(r["enrollment"])
	assertKeys(t, en, "courseId", "memberId", "status", "enrolledAt", "completedAt")
	if en["status"] != "enrolled" || en["completedAt"] != nil || num(obj(r["course"])["enrolledCount"]) != 1 {
		t.Fatalf("%v", r)
	}
	e.refused("sam", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"}, "already_enrolled")
	e.ok("sam", at, "registerMember", m{"memberId": "bob", "name": "Bob", "tier": "basic"})
	e.ok("sam", at, "registerMember", m{"memberId": "cy", "name": "Cy", "tier": "basic"})
	e.ok("sam", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "bob"})
	e.refused("sam", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "cy"}, "course_full")
	// suspension is checked before enrollment state and capacity
	e.ok("sam", at, "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})
	e.refused("sam", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"}, "member_suspended")
	// completion frees a place and does not check standing
	done := e.ok("sam", at, "completeCourse", m{"courseId": "laser-101", "memberId": "ada"})
	if obj(done["enrollment"])["status"] != "completed" || obj(done["enrollment"])["completedAt"] != at {
		t.Fatalf("%v", done)
	}
	e.refused("sam", at, "completeCourse", m{"courseId": "laser-101", "memberId": "ada"}, "not_enrolled")
	e.refused("sam", at, "completeCourse", m{"courseId": "laser-101", "memberId": "cy"}, "not_enrolled")
	e.ok("sam", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "cy"})
}

func TestEnrollmentChargeAndZeroFee(t *testing.T) {
	e := shop(t)
	at := "2026-03-02T09:00Z"
	e.ok("sam", at, "addCourse", m{"courseId": "free", "familyId": "laser", "title": "Free", "feeCents": 0, "capacity": 5})
	e.ok("sam", at, "enrollInCourse", m{"courseId": "free", "memberId": "ada"})
	// the course fee of 3000 from the fixture plus a 0 cent charge
	acc := obj(e.ok("ada", at, "getAccount", m{"memberId": "ada"})["account"])
	if num(acc["unbilledCents"]) != 3000 {
		t.Fatalf("%v", acc)
	}
	e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})
	inv := obj(list(e.ok("ada", "2026-04-01T08:00Z", "listInvoices", m{"memberId": "ada"})["invoices"])[0])
	lines := list(inv["lines"])
	if len(lines) != 3 || obj(lines[0])["kind"] != "membership" || obj(lines[1])["ref"] != "laser-101" ||
		num(obj(lines[2])["amountCents"]) != 0 || obj(lines[2])["discountPercent"].(float64) != 0 {
		t.Fatalf("%v", lines)
	}
}

func TestCertifications(t *testing.T) {
	e := shop(t)
	at := "2026-03-02T09:00Z"
	certs := list(e.ok("ada", at, "listCertifications", m{"memberId": "ada"})["certifications"])
	if len(certs) != 1 {
		t.Fatalf("%v", certs)
	}
	c := obj(certs[0])
	assertKeys(t, c, "memberId", "familyId", "issuedOn", "status")
	if c["issuedOn"] != "2026-03-02" || c["status"] != "valid" {
		t.Fatalf("%v", c)
	}
	// completing the course again reissues the certification and replaces the old one
	e.ok("sam", "2027-03-01T09:00Z", "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"})
	done := e.ok("sam", "2027-03-01T09:00Z", "completeCourse", m{"courseId": "laser-101", "memberId": "ada"})
	if obj(done["certification"])["issuedOn"] != "2027-03-01" {
		t.Fatalf("%v", done)
	}
	if n := len(list(e.ok("ada", "2027-03-01T09:00Z", "listCertifications", m{"memberId": "ada"})["certifications"])); n != 1 {
		t.Fatalf("certifications: %d", n)
	}
	rv := obj(e.ok("sam", "2027-03-01T09:00Z", "revokeCertification", m{"memberId": "ada", "familyId": "laser"})["certification"])
	if rv["status"] != "revoked" || rv["issuedOn"] != "2027-03-01" {
		t.Fatalf("%v", rv)
	}
	e.refused("sam", "2027-03-01T09:00Z", "revokeCertification", m{"memberId": "ada", "familyId": "laser"}, "not_certified")
	// completing the course again replaces a revoked certification with a valid one
	e.ok("sam", "2027-03-01T09:00Z", "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"})
	e.ok("sam", "2027-03-01T09:00Z", "completeCourse", m{"courseId": "laser-101", "memberId": "ada"})
	got := obj(list(e.ok("ada", "2027-03-01T09:00Z", "listCertifications", m{"memberId": "ada"})["certifications"])[0])
	if got["status"] != "valid" {
		t.Fatalf("%v", got)
	}
}

func TestCertificationsNeverExpire(t *testing.T) {
	e := shop(t)
	e.ok("admin", "2026-03-02T09:00Z", "addFamily", m{"familyId": "cnc", "name": "CNC"})
	e.refused("sam", "2026-03-02T09:00Z", "revokeCertification", m{"memberId": "ada", "familyId": "cnc"}, "not_certified")
	for _, at := range []string{"2027-03-02T00:00Z", "2030-01-01T00:00Z", "2099-12-31T00:00Z"} {
		c := obj(list(e.ok("admin", at, "listCertifications", m{"memberId": "ada"})["certifications"])[0])
		assertKeys(t, c, "memberId", "familyId", "issuedOn", "status")
		if c["status"] != "valid" {
			t.Fatalf("%s: %v", at, c)
		}
	}
	// revoking works long after issue
	e.ok("admin", "2099-12-31T00:00Z", "revokeCertification", m{"memberId": "ada", "familyId": "laser"})
	c := obj(list(e.ok("admin", "2099-12-31T00:00Z", "listCertifications", m{"memberId": "ada"})["certifications"])[0])
	if c["status"] != "revoked" {
		t.Fatalf("%v", c)
	}
}
