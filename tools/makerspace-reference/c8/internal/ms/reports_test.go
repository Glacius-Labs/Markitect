package ms_test

import "testing"

func TestIncidents(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	inc := obj(e.ok("sam", now, "reportIncident", m{"machineId": "lc1", "severity": "high", "description": "smoke"})["incident"])
	assertKeys(t, inc, "incidentId", "machineId", "severity", "description", "reportedBy", "reportedAt", "status", "resolution", "resolvedAt")
	if inc["incidentId"] != "inc-1" || inc["reportedBy"] != "sam" || inc["status"] != "open" || inc["resolution"] != nil || inc["resolvedAt"] != nil {
		t.Fatalf("%v", inc)
	}
	e.refused("sam", now, "reportIncident", m{"machineId": "lc1", "severity": "urgent", "description": "x"}, "invalid_argument")
	e.refused("sam", now, "reportIncident", m{"machineId": "nomachine", "severity": "low", "description": "x"}, "not_found")
	e.ok("admin", now, "reportIncident", m{"machineId": "lc1", "severity": "low", "description": "scratch"})
	res := obj(e.ok("sam", "2026-03-03T09:00Z", "resolveIncident", m{"incidentId": "inc-1", "resolution": "fan replaced"})["incident"])
	if res["status"] != "resolved" || res["resolution"] != "fan replaced" || res["resolvedAt"] != "2026-03-03T09:00Z" {
		t.Fatalf("%v", res)
	}
	e.refused("sam", "2026-03-03T09:00Z", "resolveIncident", m{"incidentId": "inc-1", "resolution": "again"}, "invalid_state")
	e.refused("sam", "2026-03-03T09:00Z", "resolveIncident", m{"incidentId": "inc-9", "resolution": "x"}, "not_found")
	all := list(e.ok("sam", "2026-03-03T09:00Z", "listIncidents", m{})["incidents"])
	if len(all) != 2 || obj(all[0])["incidentId"] != "inc-1" || obj(all[1])["incidentId"] != "inc-2" {
		t.Fatalf("%v", all)
	}
	// an incident blocks nothing
	if r := book(e, "ada", "2026-03-03T09:00Z", "2026-03-04T10:00Z", "2026-03-04T11:00Z"); r["status"] != "ok" {
		t.Fatalf("%v", r)
	}
}

func row(t *testing.T, e *env, at, from, to, machine string) map[string]any {
	t.Helper()
	out := e.ok("sam", at, "utilizationReport", m{"from": from, "to": to})
	for _, r := range list(out["machines"]) {
		if obj(r)["machineId"] == machine {
			return obj(r)
		}
	}
	t.Fatalf("no row for %s: %v", machine, out)
	return nil
}

func TestUtilizationReport(t *testing.T) {
	e := shop(t)
	// Monday 2026-03-02 to Monday 2026-03-09: 4 x 840 + 720 + 480 + 240 = 4800 open minutes
	out := e.ok("sam", "2026-03-02T09:00Z", "utilizationReport", m{"from": "2026-03-02", "to": "2026-03-09"})
	assertKeys(t, out, "from", "to", "machines", "totals")
	r := obj(list(out["machines"])[0])
	assertKeys(t, r, "machineId", "openMinutes", "maintenanceMinutes", "lockoutMinutes", "availableMinutes", "usedMinutes", "utilizationPercent")
	assertKeys(t, obj(out["totals"]), "openMinutes", "maintenanceMinutes", "lockoutMinutes", "availableMinutes", "usedMinutes", "utilizationPercent")
	if num(r["openMinutes"]) != 4800 || num(r["availableMinutes"]) != 4800 || num(r["usedMinutes"]) != 0 || num(r["utilizationPercent"]) != 0 {
		t.Fatalf("%v", r)
	}
	// maintenance counts only inside opening hours; overlapping windows count once
	e.ok("sam", "2026-03-02T09:00Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T08:00Z", "end": "2026-03-03T10:00Z", "reason": "a"})
	e.ok("sam", "2026-03-02T09:00Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-03T09:30Z", "end": "2026-03-03T10:30Z", "reason": "b"})
	e.ok("sam", "2026-03-02T09:00Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-08T00:00Z", "end": "2026-03-09T00:00Z", "reason": "sunday"})
	e.ok("sam", "2026-03-02T09:00Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-04T20:00Z", "end": "2026-03-04T23:00Z", "reason": "c"})
	e.ok("sam", "2026-03-02T09:00Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-05T09:00Z", "end": "2026-03-05T10:00Z", "reason": "cancelled"})
	e.ok("sam", "2026-03-02T09:00Z", "cancelMaintenance", m{"windowId": "mnt-5"})
	r = row(t, e, "2026-03-02T09:00Z", "2026-03-02", "2026-03-09", "lc1")
	// 08:00-10:30 on Tuesday (150), the open 12:00-16:00 of Sunday (240) and 20:00-22:00 on Wednesday (120)
	if num(r["maintenanceMinutes"]) != 510 || num(r["availableMinutes"]) != 4290 {
		t.Fatalf("%v", r)
	}
	// the range excludes its end date and counts partial windows
	r = row(t, e, "2026-03-02T09:00Z", "2026-03-03", "2026-03-04", "lc1")
	if num(r["openMinutes"]) != 840 || num(r["maintenanceMinutes"]) != 150 {
		t.Fatalf("%v", r)
	}
	// every machine gets the full open count regardless of when it was added
	e.ok("admin", "2026-03-02T09:00Z", "addMachine", m{"machineId": "lc0", "name": "N", "familyId": "laser", "rateCentsPerMinute": 1})
	out = e.ok("sam", "2026-03-02T09:00Z", "utilizationReport", m{"from": "2026-03-02", "to": "2026-03-09"})
	rows := list(out["machines"])
	if len(rows) != 2 || obj(rows[0])["machineId"] != "lc0" || num(obj(rows[0])["openMinutes"]) != 4800 {
		t.Fatalf("%v", rows)
	}
	tot := obj(out["totals"])
	if num(tot["openMinutes"]) != 9600 || num(tot["maintenanceMinutes"]) != 510 || num(tot["availableMinutes"]) != 9090 {
		t.Fatalf("%v", tot)
	}
}

func TestUtilizationUsedMinutesAndPercent(t *testing.T) {
	e := shop(t)
	// 20:00-22:00 on Tuesday; the used minutes are counted whatever the opening hours
	book(e, "ada", "2026-03-02T09:00Z", "2026-03-03T20:00Z", "2026-03-03T21:00Z")
	e.ok("ada", "2026-03-03T20:00Z", "checkIn", m{"reservationId": "res-1"})
	e.ok("ada", "2026-03-03T22:00Z", "checkOut", m{"reservationId": "res-1"})
	// a checked-in session is not counted
	book(e, "ada", "2026-03-03T22:00Z", "2026-03-04T10:00Z", "2026-03-04T11:00Z")
	e.ok("ada", "2026-03-04T10:00Z", "checkIn", m{"reservationId": "res-2"})
	r := row(t, e, "2026-03-04T10:30Z", "2026-03-03", "2026-03-04", "lc1")
	if num(r["usedMinutes"]) != 120 || num(r["openMinutes"]) != 840 || num(r["utilizationPercent"]) != 14 { // 120/840 = 14.29
		t.Fatalf("%v", r)
	}
	// only the part inside the range counts
	r = row(t, e, "2026-03-04T10:30Z", "2026-03-04", "2026-03-05", "lc1")
	if num(r["usedMinutes"]) != 0 {
		t.Fatalf("%v", r)
	}
}

func TestUtilizationRangeRules(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.refused("sam", now, "utilizationReport", m{"from": "2026-03-02", "to": "2026-03-02"}, "invalid_interval")
	e.refused("sam", now, "utilizationReport", m{"from": "2026-03-03", "to": "2026-03-02"}, "invalid_interval")
	e.refused("sam", now, "utilizationReport", m{"from": "2026-03-02", "to": "2027-03-04"}, "invalid_interval") // 367 days
	e.ok("sam", now, "utilizationReport", m{"from": "2026-03-02", "to": "2027-03-03"})                          // 366 days
	e.ok("sam", now, "utilizationReport", m{"from": "2026-03-02", "to": "2026-03-03"})
	sunday := e.ok("admin", now, "utilizationReport", m{"from": "2026-03-08", "to": "2026-03-09"}) // Sunday only
	tot := obj(sunday["totals"])
	if num(tot["openMinutes"]) != 240 || num(tot["utilizationPercent"]) != 0 {
		t.Fatalf("%v", tot)
	}
}

func TestUtilizationWithoutMachines(t *testing.T) {
	e := newEnv(t)
	out := e.ok("admin", "2026-03-02T09:00Z", "utilizationReport", m{"from": "2026-03-02", "to": "2026-03-09"})
	if len(list(out["machines"])) != 0 {
		t.Fatalf("%v", out)
	}
	for k, v := range obj(out["totals"]) {
		if num(v) != 0 {
			t.Fatalf("%s = %v", k, v)
		}
	}
}

func TestRevenueReport(t *testing.T) {
	e := usageMonth(t)
	e.ok("sam", "2026-03-04T09:00Z", "addMaterial", m{"materialId": "a", "name": "A", "unitPriceCents": 100})
	e.ok("sam", "2026-03-04T09:00Z", "restockMaterial", m{"materialId": "a", "quantity": 5})
	e.ok("ada", "2026-03-04T09:00Z", "purchaseMaterial", m{"memberId": "ada", "materialId": "a", "quantity": 2})
	e.ok("admin", "2026-03-10T09:00Z", "issueCredit", m{"memberId": "ada", "amountCents": 500, "reason": "x"})
	before := e.ok("sam", "2026-03-10T09:00Z", "revenueReport", m{"month": "2026-03"})
	assertKeys(t, before, "month", "invoicesGenerated", "invoiceCount", "membershipCents", "usageCents", "materialCents",
		"courseCents", "invoicedCents", "creditAppliedCents", "outstandingCents", "paymentsCents", "creditsIssuedCents")
	if before["invoicesGenerated"] != false || num(before["invoiceCount"]) != 0 || num(before["invoicedCents"]) != 0 ||
		num(before["creditsIssuedCents"]) != 500 {
		t.Fatalf("%v", before)
	}
	e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})
	// membership 4900, course 3000, usage 1499, material 200; credit 500 applied
	e.ok("ada", "2026-04-02T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 10000})
	r := e.ok("sam", "2026-04-02T08:00Z", "revenueReport", m{"month": "2026-03"})
	want := map[string]int{"invoiceCount": 1, "membershipCents": 4900, "usageCents": 1499, "materialCents": 200,
		"courseCents": 3000, "invoicedCents": 9599, "creditAppliedCents": 500, "outstandingCents": 0,
		"paymentsCents": 0, "creditsIssuedCents": 500}
	for k, v := range want {
		if num(r[k]) != v {
			t.Errorf("%s = %v, want %d", k, r[k], v)
		}
	}
	if r["invoicesGenerated"] != true {
		t.Fatalf("%v", r)
	}
	// the payment (9099 applied, 901 credited) belongs to April, with all its parts
	apr := e.ok("sam", "2026-04-02T08:00Z", "revenueReport", m{"month": "2026-04"})
	if num(apr["paymentsCents"]) != 10000 || apr["invoicesGenerated"] != false || num(apr["creditsIssuedCents"]) != 0 {
		t.Fatalf("%v", apr)
	}
}
