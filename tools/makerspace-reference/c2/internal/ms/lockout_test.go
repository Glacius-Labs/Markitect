package ms_test

import "testing"

// lockShop adds bob (certified), a second machine lc2, and an open incident on lc1.
func lockShop(t *testing.T) *env {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.ok("sam", now, "registerMember", m{"memberId": "bob", "name": "Bob", "tier": "basic"})
	e.ok("sam", now, "enrollInCourse", m{"courseId": "laser-101", "memberId": "bob"})
	e.ok("sam", now, "completeCourse", m{"courseId": "laser-101", "memberId": "bob"})
	e.ok("admin", now, "addMachine", m{"machineId": "lc2", "name": "L2", "familyId": "laser", "rateCentsPerMinute": 10})
	e.ok("sam", now, "reportIncident", m{"machineId": "lc1", "severity": "high", "description": "smoke"})
	return e
}

func credit(e *env, at, member string) int {
	return num(obj(e.ok("admin", at, "getAccount", m{"memberId": member})["account"])["creditBalanceCents"])
}

func TestLockMachineCancelsUnusedReservationsAndCredits(t *testing.T) {
	e := lockShop(t)
	now := "2026-03-02T09:00Z"
	bookAs := func(member, machine, start, end string) {
		t.Helper()
		e.ok("sam", now, "createReservation", m{"memberId": member, "machineId": machine, "start": start, "end": end})
	}
	bookAs("ada", "lc1", "2026-03-03T10:00Z", "2026-03-03T11:00Z") // res-1: checked in, not changed
	bookAs("bob", "lc1", "2026-03-03T11:00Z", "2026-03-03T12:00Z") // res-2: overlaps the lockout
	bookAs("bob", "lc2", "2026-03-03T13:00Z", "2026-03-03T14:00Z") // res-3: other machine
	bookAs("ada", "lc1", "2026-03-03T12:00Z", "2026-03-03T13:00Z") // res-4: cancelled before the lockout
	bookAs("bob", "lc1", "2026-03-03T14:00Z", "2026-03-03T15:00Z") // res-5: touches the lockout end
	e.ok("sam", now, "cancelReservation", m{"reservationId": "res-4"})
	bookAs("ada", "lc1", "2026-03-03T12:00Z", "2026-03-03T13:00Z") // res-6: overlaps the lockout
	e.ok("ada", "2026-03-03T10:00Z", "checkIn", m{"reservationId": "res-1"})
	// the lockout is [10:30, 14:00): res-1 (checked in), res-2 and res-6 overlap; res-5 only touches
	out := e.ok("sam", "2026-03-03T10:00Z", "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-03T10:30Z", "end": "2026-03-03T14:00Z"})
	assertKeys(t, out, "lockout", "cancelledReservations")
	l := obj(out["lockout"])
	assertKeys(t, l, "lockoutId", "incidentId", "machineId", "start", "end", "cancelledReservationIds")
	ids := list(l["cancelledReservationIds"])
	if l["lockoutId"] != "lock-1" || l["machineId"] != "lc1" || len(ids) != 2 || ids[0] != "res-2" || ids[1] != "res-6" {
		t.Fatalf("%v", l)
	}
	cr := list(out["cancelledReservations"])
	if len(cr) != 2 || obj(cr[0])["reservationId"] != "res-2" || obj(cr[0])["status"] != "cancelled" {
		t.Fatalf("%v", cr)
	}
	rs := list(e.ok("ada", "2026-03-03T10:00Z", "listMemberReservations", m{"memberId": "ada"})["reservations"])
	if obj(rs[0])["status"] != "checkedIn" {
		t.Fatalf("%v", rs)
	}
	if credit(e, "2026-03-03T10:00Z", "ada") != 1000 || credit(e, "2026-03-03T10:00Z", "bob") != 1000 {
		t.Fatal("lockout credit")
	}
	// the checked-in session is not interrupted and its minutes are metered in full
	co := e.ok("ada", "2026-03-03T11:30Z", "checkOut", m{"reservationId": "res-1"})
	if num(obj(co["reservation"])["meteredMinutes"]) != 90 {
		t.Fatalf("%v", co)
	}
	// exactly one audit entry for the request, with the new lockout as target
	var count int
	for _, en := range list(e.ok("admin", "2026-03-03T11:30Z", "listAuditEntries", m{})["entries"]) {
		if obj(en)["op"] == "lockMachine" {
			count++
			if obj(en)["target"] != "lock-1" {
				t.Fatalf("%v", en)
			}
		}
	}
	if count != 1 {
		t.Fatalf("lockMachine entries: %d", count)
	}
}

func TestLockoutCancelsStartedButNotCheckedInAndSkipsMissed(t *testing.T) {
	e := lockShop(t)
	now := "2026-03-02T09:00Z"
	e.ok("sam", now, "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": "2026-03-03T10:00Z", "end": "2026-03-03T11:00Z"})
	e.ok("sam", now, "createReservation", m{"memberId": "bob", "machineId": "lc1", "start": "2026-03-03T12:00Z", "end": "2026-03-03T13:00Z"})
	// at 11:30 res-1 is missed and unchanged; res-2 has not started
	out := e.ok("sam", "2026-03-03T11:30Z", "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-03T11:30Z", "end": "2026-03-03T12:30Z"})
	if ids := list(obj(out["lockout"])["cancelledReservationIds"]); len(ids) != 1 || ids[0] != "res-2" {
		t.Fatalf("%v", out)
	}
	if credit(e, "2026-03-03T11:30Z", "ada") != 0 {
		t.Fatal("a missed reservation earns no credit")
	}
	// a reservation that has started, but is neither checked in nor missed, is cancelled
	e.ok("sam", "2026-03-04T09:00Z", "registerMember", m{"memberId": "cy", "name": "C", "tier": "basic"})
	e.ok("sam", "2026-03-04T09:00Z", "enrollInCourse", m{"courseId": "laser-101", "memberId": "cy"})
	e.ok("sam", "2026-03-04T09:00Z", "completeCourse", m{"courseId": "laser-101", "memberId": "cy"})
	e.ok("sam", "2026-03-04T09:00Z", "createReservation", m{"memberId": "cy", "machineId": "lc1", "start": "2026-03-04T10:00Z", "end": "2026-03-04T11:00Z"})
	out = e.ok("sam", "2026-03-04T10:30Z", "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-04T10:30Z", "end": "2026-03-04T12:00Z"})
	if ids := list(obj(out["lockout"])["cancelledReservationIds"]); len(ids) != 1 {
		t.Fatalf("%v", out)
	}
	// an incident can have several lockouts, and a lockout without reservations has an empty list
	out = e.ok("sam", "2026-03-04T10:30Z", "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-10T10:00Z", "end": "2026-03-10T12:00Z"})
	if obj(out["lockout"])["lockoutId"] != "lock-3" || len(list(obj(out["lockout"])["cancelledReservationIds"])) != 0 || len(list(out["cancelledReservations"])) != 0 {
		t.Fatalf("%v", out)
	}
}

func TestLockMachineRefusals(t *testing.T) {
	e := lockShop(t)
	now := "2026-03-02T09:00Z"
	e.refused("sam", now, "lockMachine", m{"incidentId": "inc-9", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"}, "not_found")
	e.refused("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T10:00Z"}, "invalid_interval")
	e.refused("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-02T08:59Z", "end": "2026-03-02T10:00Z"}, "starts_in_past")
	e.refused("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-02T08:00Z", "end": "2026-03-02T07:00Z"}, "invalid_interval")
	e.ok("sam", now, "resolveIncident", m{"incidentId": "inc-1", "resolution": "fixed"})
	// invalid_state comes first
	e.refused("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-02T08:00Z", "end": "2026-03-02T07:00Z"}, "invalid_state")
	// a refused request consumes no lockout number
	e.ok("sam", now, "reportIncident", m{"machineId": "lc1", "severity": "low", "description": "x"})
	out := e.ok("sam", now, "lockMachine", m{"incidentId": "inc-2", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"})
	if obj(out["lockout"])["lockoutId"] != "lock-1" {
		t.Fatalf("%v", out)
	}
	// resolving an incident does not end its lockouts
	e.ok("sam", now, "resolveIncident", m{"incidentId": "inc-2", "resolution": "fixed"})
	e.refused("ada", now, "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"}, "machine_locked")
}

func TestMachineLockedRuleOrder(t *testing.T) {
	e := lockShop(t)
	now := "2026-03-02T09:00Z"
	e.ok("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T12:00Z"})
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T10:30Z", "reason": "x"})
	slot := func(member, start, end string) resp {
		return e.send("sam", now, "createReservation", m{"memberId": member, "machineId": "lc1", "start": start, "end": end})
	}
	// maintenance is reported before the lockout
	if r := slot("ada", "2026-03-05T10:00Z", "2026-03-05T11:00Z"); r["code"] != "machine_in_maintenance" {
		t.Errorf("%v", r)
	}
	if r := slot("ada", "2026-03-05T10:30Z", "2026-03-05T11:00Z"); r["code"] != "machine_locked" {
		t.Errorf("%v", r)
	}
	// touching the lockout is allowed
	if r := slot("ada", "2026-03-05T09:00Z", "2026-03-05T10:00Z"); r["status"] != "ok" {
		t.Errorf("%v", r)
	}
	if r := slot("ada", "2026-03-05T12:00Z", "2026-03-05T13:00Z"); r["status"] != "ok" {
		t.Errorf("%v", r)
	}
	// the other machine is not locked
	if r := e.send("sam", now, "createReservation", m{"memberId": "bob", "machineId": "lc2", "start": "2026-03-05T10:30Z", "end": "2026-03-05T11:00Z"}); r["status"] != "ok" {
		t.Errorf("%v", r)
	}
	// maintenance may be scheduled over a lockout
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-05T11:00Z", "end": "2026-03-05T11:30Z", "reason": "x"})
	// a lockout over a booked slot cancels it and never causes reservation_conflict
	out := e.ok("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-05T09:00Z", "end": "2026-03-05T09:30Z"})
	if len(list(out["cancelledReservations"])) != 1 {
		t.Fatalf("%v", out)
	}
}

func TestLockoutSchedule(t *testing.T) {
	e := lockShop(t)
	now := "2026-03-02T09:00Z"
	e.ok("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-05T14:00Z", "end": "2026-03-05T15:00Z"})
	e.ok("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-04T23:00Z", "end": "2026-03-05T01:00Z"})
	e.ok("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-06T00:00Z", "end": "2026-03-06T01:00Z"})
	s := e.ok("ada", now, "getMachineSchedule", m{"machineId": "lc1", "date": "2026-03-05"})
	ls := list(s["lockouts"])
	if len(ls) != 2 || obj(ls[0])["lockoutId"] != "lock-2" || obj(ls[1])["lockoutId"] != "lock-1" {
		t.Fatalf("%v", ls)
	}
	if other := e.ok("ada", now, "getMachineSchedule", m{"machineId": "lc2", "date": "2026-03-05"}); len(list(other["lockouts"])) != 0 {
		t.Fatalf("%v", other)
	}
}

func TestLockoutUtilization(t *testing.T) {
	e := lockShop(t)
	now := "2026-03-02T09:00Z"
	// Wednesday 2026-03-04: open 09:00-21:00 (720 minutes)
	e.ok("sam", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-04T09:00Z", "end": "2026-03-04T11:00Z", "reason": "x"}) // 120
	e.ok("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-04T10:00Z", "end": "2026-03-04T13:00Z"})                     // 60 shared with maintenance
	e.ok("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-04T12:00Z", "end": "2026-03-04T14:00Z"})                     // overlapping lockouts count once
	e.ok("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-04T20:00Z", "end": "2026-03-04T23:00Z"})                     // 60 inside hours
	r := row(t, e, now, "2026-03-04", "2026-03-05", "lc1")
	// maintenance 120; lockouts 10:00-14:00 (240) and 20:00-21:00 (60) = 300; blocked 09:00-14:00 and 20:00-21:00 = 360
	if num(r["maintenanceMinutes"]) != 120 || num(r["lockoutMinutes"]) != 300 || num(r["availableMinutes"]) != 720-360 {
		t.Fatalf("%v", r)
	}
	out := e.ok("sam", now, "utilizationReport", m{"from": "2026-03-04", "to": "2026-03-05"})
	tot := obj(out["totals"])
	if num(tot["lockoutMinutes"]) != 300 || num(tot["openMinutes"]) != 1440 || num(tot["availableMinutes"]) != 360+720 {
		t.Fatalf("%v", tot)
	}
	// used minutes are not reduced by a lockout: this session overruns its reservation into the lockout
	e.ok("sam", now, "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": "2026-03-04T19:00Z", "end": "2026-03-04T20:00Z"})
	e.ok("ada", "2026-03-04T19:00Z", "checkIn", m{"reservationId": "res-1"})
	e.ok("ada", "2026-03-04T20:30Z", "checkOut", m{"reservationId": "res-1"})
	if r := row(t, e, "2026-03-04T21:00Z", "2026-03-04", "2026-03-05", "lc1"); num(r["usedMinutes"]) != 90 {
		t.Fatalf("%v", r)
	}
}

func TestLockoutCreditsAreCreditAndRevenue(t *testing.T) {
	e := lockShop(t)
	now := "2026-03-02T09:00Z"
	e.ok("sam", now, "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"})
	e.ok("sam", now, "setMemberStanding", m{"memberId": "ada", "standing": "suspended"}) // credit is given whatever the standing
	e.ok("sam", now, "lockMachine", m{"incidentId": "inc-1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"})
	e.ok("admin", now, "issueCredit", m{"memberId": "ada", "amountCents": 500, "reason": "x"})
	if credit(e, now, "ada") != 1500 {
		t.Fatal("credit balance")
	}
	r := e.ok("sam", now, "revenueReport", m{"month": "2026-03"})
	if num(r["creditsIssuedCents"]) != 1500 {
		t.Fatalf("%v", r)
	}
	// the lockout credit is applied to the next invoice
	inv := obj(list(e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})["invoices"])[0])
	if num(inv["creditAppliedCents"]) != 1500 || inv["memberId"] != "ada" {
		t.Fatalf("%v", inv)
	}
	// cancelling a reservation gives no credit
	e.ok("sam", "2026-04-02T09:00Z", "createReservation", m{"memberId": "bob", "machineId": "lc2", "start": "2026-04-03T10:00Z", "end": "2026-04-03T11:00Z"})
	e.ok("sam", "2026-04-02T09:00Z", "cancelReservation", m{"reservationId": "res-2"})
	if credit(e, "2026-04-02T09:00Z", "bob") != 0 {
		t.Fatal("cancellation credit")
	}
}
