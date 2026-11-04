package protocol_test

import "testing"

// A waiver signature lapses on the date 365 days after its signedAt, judged at
// the request time.
func TestWaiverSignatureLapses(t *testing.T) {
	e := shop(t)
	e.ok("admin", "2026-03-02T09:00Z", "publishWaiver", m{"familyId": "laser", "text": "Safety"})
	e.ok("sam", "2026-03-02T09:00Z", "signWaiver", m{"memberId": "ada", "familyId": "laser"})
	signed := func(at string) bool {
		rows := list(e.ok("sam", at, "complianceSummary", m{"memberId": "ada"})["families"])
		return obj(rows[0])["waiverSigned"] == true
	}
	if !signed("2027-03-01T23:59Z") {
		t.Fatal("the signature counts on the day before its lapse date")
	}
	e.refused("sam", "2027-03-01T23:59Z", "signWaiver", m{"memberId": "ada", "familyId": "laser"}, "already_signed")
	if signed("2027-03-02T00:00Z") {
		t.Fatal("the signature does not count on its lapse date")
	}
	// the date of the slot plays no part
	blockers := e.ok("sam", "2027-03-02T00:00Z", "checkEligibility", m{"memberId": "ada", "machineId": "lc1", "date": "2026-03-03"})["blockers"]
	if len(list(blockers)) != 1 || list(blockers)[0] != "waiver_missing" {
		t.Fatalf("%v", blockers)
	}
	e.ok("sam", "2027-03-02T00:00Z", "signWaiver", m{"memberId": "ada", "familyId": "laser"})
	if !signed("2027-03-02T00:00Z") {
		t.Fatal("the new signature of the same version counts")
	}
	sigs := list(e.ok("sam", "2027-03-02T00:00Z", "listMemberWaivers", m{"memberId": "ada"})["signatures"])
	if len(sigs) != 1 || obj(sigs[0])["signedAt"] != "2027-03-02T00:00Z" {
		t.Fatalf("%v", sigs)
	}
}
