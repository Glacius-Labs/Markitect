package ms_test

import "testing"

// Latest computed values of a certification (section 2.8).
func TestCertificationExpiryAfter2099(t *testing.T) {
	e := lateShop(t, "2099-03-01T09:00Z")
	e.ok("sam", "2099-03-01T09:00Z", "addCourse", m{"courseId": "c", "familyId": "laser", "title": "T", "feeCents": 0, "capacity": 2})
	e.ok("sam", "2099-03-01T09:00Z", "enrollInCourse", m{"courseId": "c", "memberId": "ada"})
	done := e.ok("sam", "2099-03-01T09:00Z", "completeCourse", m{"courseId": "c", "memberId": "ada"})
	// 2100 is not a leap year: 2099-03-01 plus 365 days is 2100-03-01
	if obj(done["certification"])["expiresOn"] != "2100-03-01" {
		t.Fatalf("%v", done)
	}
	e.ok("sam", "2099-12-31T09:00Z", "enrollInCourse", m{"courseId": "c", "memberId": "ada"})
	done = e.ok("sam", "2099-12-31T09:00Z", "completeCourse", m{"courseId": "c", "memberId": "ada"})
	cert := obj(done["certification"])
	if cert["expiresOn"] != "2100-12-31" || cert["status"] != "valid" {
		t.Fatalf("%v", cert)
	}
}
