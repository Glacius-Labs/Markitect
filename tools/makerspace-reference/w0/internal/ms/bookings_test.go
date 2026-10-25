package ms_test

import "testing"

func book(e *env, actor, at, start, end string) resp {
	return e.send(actor, at, "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": start, "end": end})
}

func TestCatalog(t *testing.T) {
	e := newEnv(t)
	at := "2026-03-02T09:00Z"
	e.ok("admin", at, "addFamily", m{"familyId": "laser", "name": "Laser"})
	e.refused("admin", at, "addFamily", m{"familyId": "laser", "name": "Other"}, "already_exists")
	mach := obj(e.ok("admin", at, "addMachine", m{"machineId": "lc1", "name": "L", "familyId": "laser", "rateCentsPerMinute": 45})["machine"])
	assertKeys(t, mach, "machineId", "name", "familyId", "rateCentsPerMinute")
	e.refused("admin", at, "addMachine", m{"machineId": "lc1", "name": "L", "familyId": "laser", "rateCentsPerMinute": 45}, "already_exists")
	e.refused("admin", at, "addMachine", m{"machineId": "x", "name": "L", "familyId": "nofam", "rateCentsPerMinute": 45}, "not_found")
	e.refused("admin", at, "addMachine", m{"machineId": "x", "name": "L", "familyId": "laser", "rateCentsPerMinute": 0}, "invalid_argument")
	e.refused("admin", at, "addMachine", m{"machineId": "x", "name": "L", "familyId": "laser", "rateCentsPerMinute": 10001}, "invalid_argument")
	e.ok("admin", at, "addMachine", m{"machineId": "M10", "name": "L", "familyId": "laser", "rateCentsPerMinute": 10000})
	e.ok("admin", at, "addMachine", m{"machineId": "M9", "name": "L", "familyId": "laser", "rateCentsPerMinute": 1})
	e.ok("admin", at, "addMachine", m{"machineId": "m1", "name": "L", "familyId": "laser", "rateCentsPerMinute": 1})
}

func TestMachineListOrder(t *testing.T) {
	e := newEnv(t)
	at := "2026-03-02T09:00Z"
	e.ok("admin", at, "addFamily", m{"familyId": "laser", "name": "Laser"})
	for _, id := range []string{"m1", "M9", "M10", "lc_1", "lc-1"} {
		e.ok("admin", at, "addMachine", m{"machineId": id, "name": "L", "familyId": "laser", "rateCentsPerMinute": 1})
	}
	var got []string
	for _, x := range list(e.ok("admin", at, "listMachines", m{})["machines"]) {
		got = append(got, obj(x)["machineId"].(string))
	}
	want := []string{"M10", "M9", "lc-1", "lc_1", "m1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order %v, want %v", got, want)
		}
	}
}

func TestReservationSlotRules(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	refuse := func(code, start, end string) {
		t.Helper()
		r := book(e, "ada", now, start, end)
		if r["code"] != code {
			t.Errorf("%s-%s: %v, want %s", start, end, r, code)
		}
	}
	refuse("invalid_interval", "2026-03-03T10:00Z", "2026-03-03T10:00Z")
	refuse("invalid_interval", "2026-03-03T11:00Z", "2026-03-03T10:00Z")
	refuse("invalid_slot", "2026-03-03T10:15Z", "2026-03-03T11:00Z")
	refuse("invalid_slot", "2026-03-03T10:00Z", "2026-03-03T10:45Z")
	refuse("invalid_slot", "2026-03-03T10:00Z", "2026-03-03T14:30Z")
	refuse("starts_in_past", "2026-03-02T08:30Z", "2026-03-02T09:30Z")
	refuse("outside_opening_hours", "2026-03-03T08:30Z", "2026-03-03T09:30Z")
	refuse("outside_opening_hours", "2026-03-03T20:30Z", "2026-03-03T21:30Z")
	refuse("outside_opening_hours", "2026-03-08T12:00Z", "2026-03-08T13:00Z") // Sunday
	refuse("outside_opening_hours", "2026-03-07T17:30Z", "2026-03-07T18:30Z") // Saturday closes 18:00
	refuse("outside_opening_hours", "2026-03-07T09:30Z", "2026-03-07T10:30Z")
	// the slot must be inside the opening period of its start date
	refuse("outside_opening_hours", "2026-03-03T23:30Z", "2026-03-04T00:00Z")
	// a slot that is exactly the shortest, longest and touching closing time
	for _, s := range [][2]string{{"2026-03-03T09:00Z", "2026-03-03T09:30Z"}, {"2026-03-04T09:00Z", "2026-03-04T13:00Z"},
		{"2026-03-05T20:30Z", "2026-03-05T21:00Z"}, {"2026-03-07T17:30Z", "2026-03-07T18:00Z"}, {"2026-03-06T10:00Z", "2026-03-06T11:00Z"}} {
		r := book(e, "ada", now, s[0], s[1])
		if r["status"] != "ok" {
			t.Errorf("%v: %v", s, r)
		}
	}
	// starting exactly at the request time is not in the past
	if r := book(e, "ada", "2026-03-09T10:00Z", "2026-03-09T10:00Z", "2026-03-09T10:30Z"); r["status"] != "ok" {
		t.Errorf("%v", r)
	}
}

func TestReservationRuleOrderAndConflicts(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.ok("admin", now, "addMachine", m{"machineId": "lc2", "name": "L2", "familyId": "laser", "rateCentsPerMinute": 10})
	e.ok("admin", now, "addFamily", m{"familyId": "cnc", "name": "CNC"})
	e.ok("admin", now, "addMachine", m{"machineId": "cnc1", "name": "C", "familyId": "cnc", "rateCentsPerMinute": 10})
	slot := func(machine, start, end string) resp {
		return e.send("sam", now, "createReservation", m{"memberId": "ada", "machineId": machine, "start": start, "end": end})
	}
	if r := slot("cnc1", "2026-03-03T10:00Z", "2026-03-03T11:00Z"); r["code"] != "not_certified" {
		t.Errorf("%v", r)
	}
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T10:30Z", "end": "2026-03-03T12:00Z", "reason": "belt"})
	if r := slot("lc1", "2026-03-03T10:00Z", "2026-03-03T11:00Z"); r["code"] != "machine_in_maintenance" {
		t.Errorf("%v", r)
	}
	// touching the window is fine
	if r := slot("lc1", "2026-03-03T09:30Z", "2026-03-03T10:30Z"); r["status"] != "ok" {
		t.Errorf("%v", r)
	}
	e.ok("sam", now, "registerMember", m{"memberId": "bob", "name": "Bob", "tier": "basic"})
	e.ok("sam", now, "enrollInCourse", m{"courseId": "laser-101", "memberId": "bob"})
	e.ok("sam", now, "completeCourse", m{"courseId": "laser-101", "memberId": "bob"})
	bob := func(machine, start, end string) resp {
		return e.send("sam", now, "createReservation", m{"memberId": "bob", "machineId": machine, "start": start, "end": end})
	}
	if r := bob("lc1", "2026-03-03T09:00Z", "2026-03-03T10:00Z"); r["code"] != "machine_booked" {
		t.Errorf("%v", r)
	}
	if r := slot("lc2", "2026-03-03T09:00Z", "2026-03-03T10:00Z"); r["code"] != "member_double_booked" {
		t.Errorf("%v", r)
	}
	// maintenance is checked before the machine is found booked
	if r := bob("lc1", "2026-03-03T10:00Z", "2026-03-03T11:00Z"); r["code"] != "machine_in_maintenance" {
		t.Errorf("%v", r)
	}
	// a cancelled reservation no longer blocks
	e.ok("sam", now, "cancelReservation", m{"reservationId": "res-1"})
	if r := bob("lc1", "2026-03-03T09:00Z", "2026-03-03T10:00Z"); r["status"] != "ok" {
		t.Errorf("%v", r)
	}
	// suspension is checked before certification
	e.ok("sam", now, "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})
	if r := slot("cnc1", "2026-03-04T10:00Z", "2026-03-04T11:00Z"); r["code"] != "member_suspended" {
		t.Errorf("%v", r)
	}
}

func TestCertificateValidOnReservationDate(t *testing.T) {
	e := shop(t) // certified on 2026-03-02, expires 2027-03-02
	if r := book(e, "ada", "2027-03-01T09:00Z", "2027-03-02T10:00Z", "2027-03-02T11:00Z"); r["code"] != "not_certified" {
		t.Errorf("reservation on the expiry date: %v", r)
	}
	if r := book(e, "ada", "2027-03-01T09:00Z", "2027-03-01T10:00Z", "2027-03-01T11:00Z"); r["status"] != "ok" {
		t.Errorf("reservation on the last valid date: %v", r)
	}
	e.ok("sam", "2027-03-01T09:00Z", "revokeCertification", m{"memberId": "ada", "familyId": "laser"})
	if r := book(e, "ada", "2027-03-01T09:00Z", "2027-03-01T12:00Z", "2027-03-01T13:00Z"); r["code"] != "not_certified" {
		t.Errorf("revoked: %v", r)
	}
}

func TestMissedReservation(t *testing.T) {
	e := shop(t)
	book(e, "ada", "2026-03-02T09:00Z", "2026-03-03T10:00Z", "2026-03-03T11:00Z")
	list1 := func(at string) any {
		rs := list(e.ok("ada", at, "listMemberReservations", m{"memberId": "ada"})["reservations"])
		return obj(rs[0])["status"]
	}
	if list1("2026-03-03T10:59Z") != "reserved" || list1("2026-03-03T11:00Z") != "missed" {
		t.Fatal("missed boundary")
	}
	e.refused("ada", "2026-03-03T11:00Z", "checkIn", m{"reservationId": "res-1"}, "invalid_state")
	e.refused("ada", "2026-03-03T11:00Z", "cancelReservation", m{"reservationId": "res-1"}, "invalid_state")
	// a missed reservation no longer blocks and causes no charge
	if r := book(e, "ada", "2026-03-03T11:00Z", "2026-03-03T11:00Z", "2026-03-03T12:00Z"); r["status"] != "ok" {
		t.Errorf("%v", r)
	}
	// the schedule still shows it
	sched := e.ok("ada", "2026-03-03T11:00Z", "getMachineSchedule", m{"machineId": "lc1", "date": "2026-03-03"})
	if len(list(sched["reservations"])) != 2 {
		t.Fatalf("%v", sched)
	}
}

func TestCheckInAndCheckOut(t *testing.T) {
	e := shop(t)
	book(e, "ada", "2026-03-02T09:00Z", "2026-03-03T10:00Z", "2026-03-03T11:00Z")
	e.refused("ada", "2026-03-03T09:59Z", "checkIn", m{"reservationId": "res-1"}, "check_in_too_early")
	e.refused("ada", "2026-03-03T10:00Z", "checkOut", m{"reservationId": "res-1"}, "invalid_state")
	// a rate change before check-in applies; one after does not
	e.ok("admin", "2026-03-03T09:59Z", "setMachineRate", m{"machineId": "lc1", "rateCentsPerMinute": 45})
	in := obj(e.ok("ada", "2026-03-03T10:00Z", "checkIn", m{"reservationId": "res-1"})["reservation"])
	assertKeys(t, in, "reservationId", "memberId", "machineId", "start", "end", "status", "checkedInAt",
		"checkedOutAt", "rateCentsPerMinute", "meteredMinutes", "amountCents")
	if in["status"] != "checkedIn" || in["checkedInAt"] != "2026-03-03T10:00Z" || num(in["rateCentsPerMinute"]) != 45 ||
		in["checkedOutAt"] != nil || in["meteredMinutes"] != nil || in["amountCents"] != nil {
		t.Fatalf("%v", in)
	}
	e.refused("ada", "2026-03-03T10:01Z", "checkIn", m{"reservationId": "res-1"}, "invalid_state")
	e.refused("ada", "2026-03-03T10:01Z", "cancelReservation", m{"reservationId": "res-1"}, "invalid_state")
	e.ok("admin", "2026-03-03T10:05Z", "setMachineRate", m{"machineId": "lc1", "rateCentsPerMinute": 100})
	// ada is plus: 10 % discount; 37 minutes at 45 cents
	out := e.ok("ada", "2026-03-03T10:37Z", "checkOut", m{"reservationId": "res-1"})
	res, ch := obj(out["reservation"]), obj(out["charge"])
	assertKeys(t, ch, "kind", "ref", "quantity", "unitPriceCents", "discountPercent", "amountCents", "at")
	if res["status"] != "completed" || num(res["meteredMinutes"]) != 37 || num(res["amountCents"]) != 1499 ||
		ch["kind"] != "usage" || ch["ref"] != "res-1" || num(ch["quantity"]) != 37 || num(ch["unitPriceCents"]) != 45 ||
		num(ch["discountPercent"]) != 10 || num(ch["amountCents"]) != 1499 || ch["at"] != "2026-03-03T10:37Z" {
		t.Fatalf("%v", out)
	}
	e.refused("ada", "2026-03-03T10:40Z", "checkOut", m{"reservationId": "res-1"}, "invalid_state")
}

func TestCheckInChecksStateStandingAndCertification(t *testing.T) {
	e := shop(t)
	book(e, "ada", "2026-03-02T09:00Z", "2026-03-03T10:00Z", "2026-03-03T11:00Z")
	e.ok("sam", "2026-03-03T10:00Z", "revokeCertification", m{"memberId": "ada", "familyId": "laser"})
	e.refused("ada", "2026-03-03T10:00Z", "checkIn", m{"reservationId": "res-1"}, "not_certified")
	e.ok("sam", "2026-03-03T10:00Z", "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})
	e.refused("ada", "2026-03-03T10:00Z", "checkIn", m{"reservationId": "res-1"}, "member_suspended")
	// the certification is checked on the date of the request time
	e.ok("sam", "2026-03-03T10:00Z", "setMemberStanding", m{"memberId": "ada", "standing": "good"})
	e.ok("sam", "2026-03-03T10:00Z", "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"})
	e.ok("sam", "2026-03-03T10:00Z", "completeCourse", m{"courseId": "laser-101", "memberId": "ada"})
	e.ok("ada", "2026-03-03T10:30Z", "checkIn", m{"reservationId": "res-1"})
}

func TestCheckOutAfterReservationEndAndZeroMinutes(t *testing.T) {
	e := shop(t)
	book(e, "ada", "2026-03-02T09:00Z", "2026-03-03T20:00Z", "2026-03-03T21:00Z")
	e.ok("ada", "2026-03-03T20:00Z", "checkIn", m{"reservationId": "res-1"})
	// the session overruns the reservation end and the closing time and is not interrupted
	out := e.ok("ada", "2026-03-03T22:30Z", "checkOut", m{"reservationId": "res-1"})
	if num(obj(out["reservation"])["meteredMinutes"]) != 150 {
		t.Fatalf("%v", out)
	}
	book(e, "ada", "2026-03-04T09:00Z", "2026-03-05T10:00Z", "2026-03-05T11:00Z")
	e.ok("ada", "2026-03-05T10:00Z", "checkIn", m{"reservationId": "res-2"})
	out = e.ok("ada", "2026-03-05T10:00Z", "checkOut", m{"reservationId": "res-2"})
	if num(obj(out["charge"])["amountCents"]) != 0 || num(obj(out["reservation"])["amountCents"]) != 0 {
		t.Fatalf("%v", out)
	}
}

func TestChecksInStoredStateOfCheckedInReservationPastItsEnd(t *testing.T) {
	e := shop(t)
	book(e, "ada", "2026-03-02T09:00Z", "2026-03-03T10:00Z", "2026-03-03T11:00Z")
	e.ok("ada", "2026-03-03T10:00Z", "checkIn", m{"reservationId": "res-1"})
	// a checked-in reservation is active, so it blocks maintenance in its interval
	e.refused("sam", "2026-03-03T10:15Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T10:30Z", "end": "2026-03-03T12:00Z", "reason": "x"}, "reservation_conflict")
	// and it stays checked in after its end; it is never reported as missed
	rs := list(e.ok("ada", "2026-03-03T12:00Z", "listMemberReservations", m{"memberId": "ada"})["reservations"])
	if obj(rs[0])["status"] != "checkedIn" {
		t.Fatalf("%v", rs)
	}
}

func TestMaintenance(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	w := obj(e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-04T06:00Z", "end": "2026-03-04T07:00Z", "reason": "night"})["window"])
	assertKeys(t, w, "windowId", "machineId", "start", "end", "reason", "status")
	if w["windowId"] != "mnt-1" || w["status"] != "scheduled" {
		t.Fatalf("%v", w)
	}
	e.refused("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-04T06:00Z", "end": "2026-03-04T06:00Z", "reason": "x"}, "invalid_interval")
	e.refused("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-02T08:59Z", "end": "2026-03-02T10:00Z", "reason": "x"}, "starts_in_past")
	// invalid_interval comes before starts_in_past
	e.refused("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-02T08:00Z", "end": "2026-03-02T07:00Z", "reason": "x"}, "invalid_interval")
	book(e, "ada", now, "2026-03-03T10:00Z", "2026-03-03T11:00Z")
	e.refused("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T10:59Z", "end": "2026-03-03T12:00Z", "reason": "x"}, "reservation_conflict")
	// touching is fine, overlapping windows are fine
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T11:00Z", "end": "2026-03-03T12:00Z", "reason": "x"})
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T11:30Z", "end": "2026-03-03T12:30Z", "reason": "x"})
	// a refused request consumes no number
	if w := obj(e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-05T11:00Z", "end": "2026-03-05T12:00Z", "reason": "x"})["window"]); w["windowId"] != "mnt-4" {
		t.Fatalf("%v", w)
	}
	// a window overlapping a cancelled reservation is fine
	e.ok("sam", now, "cancelReservation", m{"reservationId": "res-1"})
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T10:00Z", "end": "2026-03-03T10:30Z", "reason": "x"})
}

func TestCancelMaintenance(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T10:00Z", "end": "2026-03-03T12:00Z", "reason": "x"})
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-02T09:30Z", "end": "2026-03-02T12:00Z", "reason": "x"})
	e.refused("sam", "2026-03-02T09:30Z", "cancelMaintenance", m{"windowId": "mnt-2"}, "invalid_state") // already started
	w := obj(e.ok("sam", now, "cancelMaintenance", m{"windowId": "mnt-1"})["window"])
	if w["status"] != "cancelled" {
		t.Fatalf("%v", w)
	}
	e.refused("sam", now, "cancelMaintenance", m{"windowId": "mnt-1"}, "invalid_state")
	e.refused("sam", now, "cancelMaintenance", m{"windowId": "mnt-9"}, "not_found")
	if r := book(e, "ada", now, "2026-03-03T10:00Z", "2026-03-03T11:00Z"); r["status"] != "ok" {
		t.Fatalf("%v", r)
	}
	sched := e.ok("sam", now, "getMachineSchedule", m{"machineId": "lc1", "date": "2026-03-03"})
	if len(list(sched["maintenance"])) != 0 {
		t.Fatalf("%v", sched)
	}
}

func TestMachineScheduleAndMemberReservations(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	book(e, "ada", now, "2026-03-03T14:00Z", "2026-03-03T15:00Z")
	book(e, "ada", now, "2026-03-03T10:00Z", "2026-03-03T11:00Z")
	book(e, "ada", now, "2026-03-04T10:00Z", "2026-03-04T11:00Z")
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-02T23:00Z", "end": "2026-03-03T00:30Z", "reason": "x"})
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T23:30Z", "end": "2026-03-04T01:00Z", "reason": "x"})
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T00:30Z", "end": "2026-03-03T01:00Z", "reason": "x"})
	e.ok("sam", now, "cancelReservation", m{"reservationId": "res-3"})
	s := e.ok("ada", now, "getMachineSchedule", m{"machineId": "lc1", "date": "2026-03-03"})
	assertKeys(t, s, "machineId", "date", "openingHours", "reservations", "maintenance")
	hours := obj(s["openingHours"])
	assertKeys(t, hours, "open", "close")
	if hours["open"] != "09:00" || hours["close"] != "21:00" {
		t.Fatalf("%v", hours)
	}
	rs, ms := list(s["reservations"]), list(s["maintenance"])
	if len(rs) != 2 || obj(rs[0])["reservationId"] != "res-2" || obj(rs[1])["reservationId"] != "res-1" {
		t.Fatalf("%v", rs)
	}
	if len(ms) != 3 || obj(ms[0])["windowId"] != "mnt-1" || obj(ms[1])["windowId"] != "mnt-3" || obj(ms[2])["windowId"] != "mnt-2" {
		t.Fatalf("%v", ms)
	}
	if sun := e.ok("ada", now, "getMachineSchedule", m{"machineId": "lc1", "date": "2026-03-08"}); sun["openingHours"] != nil {
		t.Fatalf("%v", sun)
	}
	if sat := obj(e.ok("ada", now, "getMachineSchedule", m{"machineId": "lc1", "date": "2026-03-07"})["openingHours"]); sat["open"] != "10:00" || sat["close"] != "18:00" {
		t.Fatalf("%v", sat)
	}
	all := list(e.ok("ada", now, "listMemberReservations", m{"memberId": "ada"})["reservations"])
	if len(all) != 3 || obj(all[0])["reservationId"] != "res-2" || obj(all[2])["status"] != "cancelled" {
		t.Fatalf("%v", all)
	}
}
